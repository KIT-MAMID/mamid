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
	var mutex *sync.Mutex = nil

	c.busyTableLock.Lock()
	// acquire a lock if possible [otherwise there is no process and we need to respawn immediately]
	if _, exists := c.busyTable[m.Port]; exists {
		mutex = c.busyTable[m.Port]
		c.busyTableLock.Unlock()
		mutex.Lock()
		c.busyTableLock.Lock()
	} else if m.State == msp.MongodStateDestroyed {
		// if not existing, we have to do nothing
		c.busyTableLock.Unlock()
		return nil
	}

	// check if still existing after locking [else possible race condition: process might have been in destruction phase while we were waiting for the lock],
	// else we need to respawn
	if _, exists := c.busyTable[m.Port]; !exists {
		c.busyTable[m.Port] = &sync.Mutex{}
		c.busyTable[m.Port].Lock()
		if mutex != nil {
			mutex.Unlock() // prevent deadlock, there might be multiple goroutines waiting on the same mutex
		}
		mutex = c.busyTable[m.Port]
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
	}

	if m.State == msp.MongodStateDestroyed {
		go func() {
			time.Sleep(c.mongodHardShutdownTimeout)
			if killProcessError := c.procManager.KillProcess(m.Port); killProcessError != nil {
				log.Error(killProcessError)
			}
			// do wait until the old instance is destroyed. Having a half destroyed unlocked instance flying around should be dangerous
			delete(c.busyTable, m.Port)
			mutex.Unlock()
		}()
		c.configurator.ApplyMongodConfiguration(m) // ignore error of destruction
		return nil
	}

	applyErr := c.configurator.ApplyMongodConfiguration(m)
	if applyErr != nil {
		log.Errorf("controller: error applying Mongod configuration: %s", applyErr)
	}

	// release lock preventing simultaneous configuration
	mutex.Unlock()
	return applyErr
}
