package slave

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/msp"
	"sync"
	"time"
)

// TODO make all these constants defaults for CLI parameters.
const DataDBDir = "db"

type Controller struct {
	processes    *ProcessManager // TODO rename variable
	configurator MongodConfigurator

	busyTable     map[msp.PortNumber]*sync.Mutex
	busyTableLock sync.Mutex

	mongodHardShutdownTimeout time.Duration
}

func NewController(processManager *ProcessManager, configurator MongodConfigurator, mongodHardShutdownTimeout time.Duration) *Controller {
	return &Controller{
		processes:                 processManager,
		configurator:              configurator,
		busyTable:                 make(map[msp.PortNumber]*sync.Mutex),
		busyTableLock:             sync.Mutex{},
		mongodHardShutdownTimeout: mongodHardShutdownTimeout,
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
				Identifier:      msp.SlaveSpawnError,
				Description:     fmt.Sprintf("Unable to start a Mongod instance on port %d", m.Port),
				LongDescription: fmt.Sprintf("ProcessManager.spawnProcess() failed to spawn Mongod on port `%d` with name `%s`: %s", m.Port, m.ReplicaSetName, err),
			}
		}
	} else {
		return nil
	}

	if m.State == msp.MongodStateDestroyed {
		go func() {
			time.Sleep(c.mongodHardShutdownTimeout)
			c.processes.KillProcess(m.Port) // TODO error handling => log error
			c.busyTable[m.Port].Unlock()    // TODO document this line together with the Unlock() in m.State != msp.MongodStateDestroyed
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
