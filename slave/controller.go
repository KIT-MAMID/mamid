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
	mongods := make([]msp.Mongod, 0, len(ports))
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

	if len(mongods) != len(ports) {
		for m := range mongodChannel {
			mongods = append(mongods, m)
			if len(mongods) == len(ports) {
				break
			}
		}
	}
	return mongods, nil
}

func (c *Controller) EstablishMongodState(m msp.Mongod) *msp.Error {

	c.busyTableLock.Lock()
	// acquire a lock if possible [otherwise there is no process and we need to respawn immediately]
	if _, exists := c.busyTable[m.Port]; !exists {
		c.busyTable[m.Port] = &sync.Mutex{}
	}
	mutex := c.busyTable[m.Port]
	mutex.Lock()
	defer mutex.Unlock()
	c.busyTableLock.Unlock()

	if m.State == msp.MongodStateRunning {

		// check if still existing after locking [else possible race condition: process might have been in destruction phase while we were waiting for the lock],
		// else we need to respawn
		if _, exists := c.procManager.runningProcesses[m.Port]; !exists {

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

		applyErr := c.configurator.ApplyMongodConfiguration(m)
		if applyErr != nil {
			log.Errorf("controller: error applying Mongod configuration: %s", applyErr)
		}
		return applyErr

	} else if m.State == msp.MongodStateDestroyed || m.State == msp.MongodStateNotRunning {
		c.stopMongod(m)
		//TODO Delete files if destroy
		return nil
	}

	// release lock preventing simultaneous configuration
	return &msp.Error{
		Identifier:  msp.BadStateDescription,
		Description: fmt.Sprintf("Unknown desired state"),
	}
}

func (c *Controller) stopMongod(m msp.Mongod) {
	applyErr := c.configurator.ApplyMongodConfiguration(m) // ignore error of destruction, will be killed
	if applyErr != nil {
		log.WithField("error", applyErr).Errorf("could not soft shutdown mongod on port %d", m.Port)
	}

	killErr := c.procManager.KillProcess(m.Port) //TODO This should actually just be a sanity check and remove from running processes
	if killErr != nil {
		log.WithField("error", killErr).Error("could not soft shutdown mongod on port %d", m.Port)
	}
}
