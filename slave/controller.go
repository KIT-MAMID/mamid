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
	procManager  *ProcessManager
	configurator MongodConfigurator

	busyTable     map[msp.PortNumber]*sync.Mutex
	busyTableLock sync.Mutex

	mongodHardShutdownTimeout time.Duration
}

func NewController(processManager *ProcessManager, configurator MongodConfigurator, mongodHardShutdownTimeout time.Duration) *Controller {
	return &Controller{
		procManager:               processManager,
		configurator:              configurator,
		busyTable:                 make(map[msp.PortNumber]*sync.Mutex),
		busyTableLock:             sync.Mutex{},
		mongodHardShutdownTimeout: mongodHardShutdownTimeout,
	}
}

func (c *Controller) RequestStatus() ([]msp.Mongod, *msp.Error) {
	ports := c.procManager.RunningProcesses()
	mongods := make([]msp.Mongod, len(ports))
	mongodChannel := make(chan msp.Mongod, len(ports))
	for _, port := range ports {
		go func(resultsChan chan<- msp.Mongod, port msp.PortNumber) {
			mongod, err := c.configurator.MongodConfiguration(port)
			if err != nil {
				log.Errorf("controller: error querying Mongod configuration: %s", err)
				resultsChan <- msp.Mongod{
					Port:        port,
					StatusError: err,
					State:       msp.MongodStateNotRunning,
				}
			} else {
				resultsChan <- mongod
			}
		}(mongodChannel, port)
	}

	for m := range mongodChannel {
		mongods = append(mongods, m)
		if len(mongods) == len(ports) {
			break
		}
	}
	return mongods, nil
}

func (c *Controller) EstablishMongodState(m msp.Mongod) *msp.Error {
	c.busyTableLock.Lock()
	if _, exists := c.busyTable[m.Port]; exists {
		c.busyTable[m.Port].Lock()
		c.busyTableLock.Unlock()
	} else if m.State == msp.MongodStateDestroyed {
		return nil
	}

	c.busyTable[m.Port] = &sync.Mutex{}
	c.busyTable[m.Port].Lock()
	c.busyTableLock.Unlock()
	err := c.procManager.SpawnProcess(m)

	if err != nil {
		log.Errorf("controller: error spawning process: %s", err)
		return &msp.Error{
			Identifier:      msp.SlaveSpawnError,
			Description:     fmt.Sprintf("Unable to start a Mongod instance on port %d", m.Port),
			LongDescription: fmt.Sprintf("ProcessManager.spawnProcess() failed to spawn Mongod on port `%d` with name `%s`: %s", m.Port, m.ReplicaSetName, err),
		}
	}

	if m.State == msp.MongodStateDestroyed {
		go func() {
			time.Sleep(c.mongodHardShutdownTimeout)
			if killProcessError := c.procManager.KillProcess(m.Port); killProcessError != nil {
				log.Error(killProcessError)
			}
			c.busyTable[m.Port].Unlock() // TODO document this line together with the Unlock() in m.State != msp.MongodStateDestroyed
		}()
		c.configurator.ApplyMongodConfiguration(m) // ignore error of destruction
		return nil
	}

	applyErr := c.configurator.ApplyMongodConfiguration(m)
	if applyErr != nil {
		log.Errorf("controller: error applying Mongod configuration: %s", applyErr)
	}

	// do wait until the old instance is destroyed. Having a half destroyed unlocked instance flying around should be dangerous
	if m.State != msp.MongodStateDestroyed {
		c.busyTable[m.Port].Unlock()
	}
	return applyErr
}
