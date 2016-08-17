package slave

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/msp"
	"gopkg.in/mgo.v2"
	"sync"
	"time"
)

// TODO make all these constants defaults for CLI parameters.
const DataDBDir = "db"

const MongodSoftShutdownTimeout = 3
const MongodHardShutdownTimeout = 5

type Controller struct {
	processes    ProcessManager // TODO rename variable
	configurator MongodConfigurator

	busyTable     map[msp.PortNumber]*sync.Mutex
	busyTableLock sync.Mutex
}

func NewController(mongodExecutablePath, dataDir string) *Controller {
	return &Controller{
		processes:     NewProcessManager(mongodExecutablePath, dataDir),
		configurator:  &ConcreteMongodConfigurator{mgo.Dial},
		busyTable:     make(map[msp.PortNumber]*sync.Mutex),
		busyTableLock: sync.Mutex{},
	}
}

func (c *Controller) RequestStatus() ([]msp.Mongod, *msp.Error) {
	ports := c.processes.RunningProcesses()
	mongods := make([]msp.Mongod, len(ports))
	for k, port := range ports {
		var err *msp.Error
		mongods[k], err = c.configurator.MongodConfiguration(port) // TODO parallelize this? => use goroutines that send into a channel whic appends (single-threaded) to the array
		if err != nil {
			return nil, err
		}
	}
	return mongods, nil
}

func (c *Controller) EstablishMongodState(m msp.Mongod) *msp.Error {
	c.busyTableLock.Lock()
	if _, exists := c.busyTable[m.Port]; exists {
		c.busyTable[m.Port].Lock()
		c.busyTableLock.Unlock()
	} else if m.State != msp.MongodStateDestroyed {
		c.busyTable[m.Port] = &sync.Mutex{}
		c.busyTable[m.Port].Lock()
		c.busyTableLock.Unlock()
		err := c.processes.SpawnProcess(m)

		if err != nil {
			return &msp.Error{
				Identifier:      fmt.Sprintf("spawn_%d", m.Port), // TODO this is not how msp.Error identifiers are supposed to be. Suggestion: ESLAVESPAWN or similar. This identifier needs defined as a constant in the msp dataobjects file.
				Description:     fmt.Sprintf("Unable to start a mongod instance on port %d", m.Port),
				LongDescription: fmt.Sprintf("ProcessManager.spawnProcess() failed for mongod on port %d with name %s\n%s", m.Port, m.ReplicaSetName, err.Error()), // TODO no newlines in errors?
			}
		}
	} else {
		return nil
	}

	if m.State == msp.MongodStateDestroyed {
		go func() {
			time.Sleep(MongodHardShutdownTimeout * time.Second)
			c.processes.KillProcess(m.Port)
			c.busyTable[m.Port].Unlock() // TODO document this line together with the Unlock() in m.State != msp.MongodStateDestroyed
		}()
	}

	err := c.configurator.ApplyMongodConfiguration(m)

	// do wait until the old instance is destroyed. Having a half destroyed unlocked instance flying around should be dangerous
	// TODO better refer to the hard shutdown timeout above. AND: restructure this code because this is really the else part of the `if` above.
	if m.State != msp.MongodStateDestroyed {
		c.busyTable[m.Port].Unlock()
	}
	return err
}
