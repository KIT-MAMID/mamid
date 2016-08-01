package slave

import (
	"github.com/KIT-MAMID/mamid/msp"
	"fmt"
	"gopkg.in/mgo.v2"
	"sync"
	"time"
)

const DataDBDir = "db"

const MongodSoftShutdownTimeout = 3
const MongodHardShutdownTimeout = 5

type Controller struct {
	processes ProcessManager
	configurator MongodConfigurator

	busyTable map[msp.PortNumber]*sync.Mutex
	busyTableLock sync.Mutex
}

func NewController(dataDir string) *Controller {
	return &Controller{
		processes: NewProcessManager(fmt.Sprintf("mongod --dbpath %s/%s", dataDir, DataDBDir)),
		configurator: &ConcreteMongodConfigurator{ mgo.Dial },
		busyTable: make(map[msp.PortNumber]*sync.Mutex),
		busyTableLock: sync.Mutex{},
	}
}

func (c *Controller) RequestStatus() ([]msp.Mongod, *msp.SlaveError) {
	ports := c.processes.RunningProcesses()
	mongods := make([]msp.Mongod, len(ports))
	for k, port := range ports {
		var err *msp.SlaveError
		mongods[k], err = c.configurator.MongodConfiguration(port)
		if err != nil {
			return nil, err
		}
	}
	return mongods, nil
}

func (c *Controller) EstablishMongodState(m msp.Mongod) *msp.SlaveError {
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
			return &msp.SlaveError{
				Identifier: fmt.Sprintf("spawn_%d", m.Port),
				Description: fmt.Sprintf("Unable to start a mongod instance on port %d", m.Port),
				LongDescription: fmt.Sprintf("ProcessManager.spawnProcess() failed for mongod on port %d with name %s\n%s", m.Port, m.ReplicaSetName, err.Error()),
			}
		}
	} else {
		return nil
	}

	if m.State == msp.MongodStateDestroyed {
		go func() {
			time.Sleep(MongodHardShutdownTimeout * time.Second)
			c.processes.KillProcess(m.Port)
			c.busyTable[m.Port].Unlock()
		}()
	}

	err := c.configurator.ApplyMongodConfiguration(m)

	// do wait until the old instance is destroyed. Having a half destroyed unlocked instance flying around should be dangerous
	if m.State != msp.MongodStateDestroyed {
		c.busyTable[m.Port].Unlock()
	}
	return err
}
