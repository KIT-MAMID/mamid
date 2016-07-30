package master

import (
	"github.com/KIT-MAMID/mamid/model"
	"github.com/KIT-MAMID/mamid/msp"
	"github.com/jinzhu/gorm"
	"log"
	"time"
)

type Monitor struct {
	DB              *gorm.DB
	BusWriteChannel chan interface{}
	MSPClient       msp.MSPClient
}

func (m *Monitor) Run() {
	ticker := time.NewTicker(10 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Println("Monitor running")

				//Get all slaves from database
				var slaves []model.Slave
				err := m.DB.Find(&slaves).Error
				if err != nil {
					log.Println(err.Error())
				}

				//Observe active slaves
				for _, slave := range slaves {
					if slave.ConfiguredState == model.SlaveStateActive {
						go m.observeSlave(slave)
					}
				}

			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (m *Monitor) observeSlave(slave model.Slave) {
	//Request mongod states from slave
	mongods, err := m.MSPClient.RequestStatus(msp.HostPort{slave.Hostname, uint16(slave.Port)})

	//Send connection status to bus
	commError, isCommError := err.(msp.CommunicationError)
	m.BusWriteChannel <- model.ConnectionStatus{
		Slave:              slave,
		Unreachable:        isCommError,
		CommunicationError: commError,
	}

	if err == nil {
		for _, observedMongod := range mongods {
			var modelMongod model.Mongod
			getOrCreateErr := m.DB.FirstOrCreate(&modelMongod, &model.Mongod{
				ParentSlaveID: slave.ID,
				Port:          model.PortNumber(observedMongod.Port),
				ReplSetName:   observedMongod.ReplicaSetName,
			}).Error
			if getOrCreateErr != nil {
				log.Println(getOrCreateErr.Error())
				return
			}

			//Get desired state if it exists
			relatedResult := m.DB.Model(&modelMongod).Related(&modelMongod.DesiredState, "DesiredState")
			if !relatedResult.RecordNotFound() && relatedResult.Error != nil {
				log.Println(relatedResult.Error.Error())
				return
			}

			//Get observed state if it exists
			relatedResult = m.DB.Model(&modelMongod).Related(&modelMongod.ObservedState, "ObservedState")
			if !relatedResult.RecordNotFound() && relatedResult.Error != nil {
				log.Println(relatedResult.Error.Error())
				return
			}

			if observedMongod.StatusError == nil {
				//TODO Finish this
				//Put observations into model
				modelMongod.ObservedState.ExecutionState = mspMongodStateToModelExecutionState(observedMongod.State)
				modelMongod.ObservedState.IsShardingConfigServer = observedMongod.ShardingConfigServer
				modelMongod.ObservationError = model.MSPError{}
			} else {
				modelMongod.ObservationError = slaveErrorToMspError(*observedMongod.StatusError)
			}

			//TODO Only update observed state and errors to prevent collisions with cluster allocator
			m.DB.Save(&modelMongod)

			m.BusWriteChannel <- compareStates(modelMongod)

		}
	} else {
		if commError, ok := err.(msp.CommunicationError); ok {
			//TODO Handle
			log.Printf("monitor: %#v", commError)
			return
		} else if slaveError, ok := err.(msp.SlaveError); ok {
			//TODO Handle
			log.Printf("monitor: %#v", slaveError)
			return
		} else {
			log.Println("Unknown error in Monitor.observeSlave")
			return
		}
	}
}

func compareStates(mongod model.Mongod) (m model.MongodMatchStatus) {
	//TODO Finish this
	m.Mismatch =
		mongod.DesiredState.ExecutionState != mongod.ObservedState.ExecutionState ||
			mongod.DesiredState.IsShardingConfigServer != mongod.ObservedState.IsShardingConfigServer
	m.Mongod = mongod
	return
}

func mspMongodStateToModelExecutionState(e msp.MongodState) model.MongodExecutionState {
	switch e {
	case msp.MongodStateDestroyed:
		return model.MongodExecutionStateDestroyed
	case msp.MongodStateNotRunning:
		return model.MongodExecutionStateNotRunning
	case msp.MongodStateRecovering:
		return model.MongodExecutionStateRecovering
	case msp.MongodStateRunning:
		return model.MongodExecutionStateRunning
	default:
		return 0 // Invalid
	}
}

func slaveErrorToMspError(e msp.SlaveError) model.MSPError {
	return model.MSPError{
		SlaveError: model.SlaveError{
			SlaveError: e,
		},
	}
}

func communicationErrorToMspError(e msp.CommunicationError) model.MSPError {
	return model.MSPError{
		CommunicationError: model.CommunicationError{
			CommunicationError: e,
		},
	}
}
