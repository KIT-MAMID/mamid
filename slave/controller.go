package slave

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/msp"
	"time"
)

type Controller struct {
	procManager               *ProcessManager
	configurator              MongodConfigurator
	busyTable                 *busyTable
	mongodHardShutdownTimeout time.Duration
}

func NewController(processManager *ProcessManager, configurator MongodConfigurator, mongodHardShutdownTimeout time.Duration) *Controller {
	return &Controller{
		procManager:               processManager,
		configurator:              configurator,
		busyTable:                 NewBusyTable(),
		mongodHardShutdownTimeout: mongodHardShutdownTimeout,
	}
}

func (c *Controller) RequestStatus() ([]msp.Mongod, *msp.Error) {

	replSetNameByPortNumber, err := c.procManager.parseProcessDirTree()
	if err != nil {
		return []msp.Mongod{}, &msp.Error{
			Identifier:      msp.SlaveGetMongodStatusError,
			Description:     fmt.Sprintf("Unable to read mongods from db directory"),
			LongDescription: fmt.Sprintf("ProcessManager.ExistingDataDirectories() failed: %s", err),
		}
	}

	mongods := make([]msp.Mongod, 0, len(replSetNameByPortNumber))
	mongodChannel := make(chan msp.Mongod, len(replSetNameByPortNumber))

	for port, replSetName := range replSetNameByPortNumber {
		go func(resultsChan chan<- msp.Mongod, port msp.PortNumber, replSetName string) {
			if c.procManager.HasProcess(port) {
				mongod, err := c.configurator.MongodConfiguration(port)
				if err != nil {
					//Process is running but we cant get the state
					log.Errorf("controller: error querying Mongod configuration: %s", err)
					resultsChan <- msp.Mongod{
						Port:        port,
						StatusError: err,
						State:       msp.MongodStateNotRunning,
					}
				} else {
					//Could get state successfully
					resultsChan <- mongod
				}
			} else {
				//Process is not running
				resultsChan <- msp.Mongod{
					Port: port,
					ReplicaSetConfig: msp.ReplicaSetConfig{
						ReplicaSetName: replSetName,
					},
					State: msp.MongodStateNotRunning,
				}
			}
		}(mongodChannel, port, replSetName)
	}

	// Wait for all goroutines to return
	if len(mongods) != len(replSetNameByPortNumber) {
		for m := range mongodChannel {
			mongods = append(mongods, m)
			if len(mongods) == len(replSetNameByPortNumber) {
				break
			}
		}
	}

	return mongods, nil
}

func (c *Controller) EstablishMongodState(m msp.Mongod) *msp.Error {

	defer c.busyTable.AcquireLock(m.Port).Unlock()

	switch m.State {

	case msp.MongodStateRunning:

		// check if still existing after locking [else possible race condition: process might have been in destruction phase while we were waiting for the lock],
		// else we need to respawn
		if !c.procManager.HasProcess(m.Port) {

			err := c.procManager.SpawnProcess(m)

			if err != nil {
				log.Errorf("error spawning process: %s", err)
				return &msp.Error{
					Identifier:      msp.SlaveSpawnError,
					Description:     fmt.Sprintf("Unable to start a Mongod instance on port `%d`", m.Port),
					LongDescription: fmt.Sprintf("ProcessManager.spawnProcess() failed to spawn Mongod on port `%d` with name `%s`: %s", m.Port, m.ReplicaSetConfig.ReplicaSetName, err),
				}
			}
		}

		applyErr := c.configurator.ApplyMongodConfiguration(m)
		if applyErr != nil {
			log.Errorf("controller: error applying Mongod configuration: %s", applyErr)
		}
		return applyErr

	case msp.MongodStateNotRunning:

		stopErr := c.configurator.ApplyMongodConfiguration(m)
		if stopErr != nil {
			log.WithField("error", stopErr).Errorf("could not soft shutdown mongod on port `%d`", m.Port)
		}
		return nil

	case msp.MongodStateDestroyed:

		stopErr := c.configurator.ApplyMongodConfiguration(m)
		if stopErr != nil {
			log.WithField("error", stopErr).Errorf("could not soft shutdown Mongod on port `%d`", m.Port)
		}

		//Destroy data when process is not running anymore
		if !c.procManager.HasProcess(m.Port) {
			c.procManager.destroyDataDirectory(m)
		}

		return nil

	case msp.MongodStateForceDestroyed:

		log.Debugf("Force killing Mongod on port `%d`", m.Port)

		killErr := c.procManager.KillProcess(m.Port)

		if killErr != nil {
			log.WithField("error", killErr).Errorf("could not kill Mongod on port `%d`", m.Port)
			return &msp.Error{
				Identifier:      msp.SlaveShutdownError,
				Description:     fmt.Sprintf("could not kill Mongod on port `%d`", m.Port),
				LongDescription: fmt.Sprintf("error was: %s", killErr.Error()),
			}
		}

		//Destroy data when process is not running anymore
		if !c.procManager.HasProcess(m.Port) {
			c.procManager.destroyDataDirectory(m)
		}

		return nil

	default:

		return &msp.Error{
			Identifier:  msp.BadStateDescription,
			Description: fmt.Sprintf("Unknown desired state"),
		}

	}

}

func (c *Controller) RsInitiate(m msp.RsInitiateMessage) *msp.Error {
	defer c.busyTable.AcquireLock(m.Port).Unlock()
	return c.configurator.InitiateReplicaSet(m)
}
