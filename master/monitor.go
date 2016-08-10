package master

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/KIT-MAMID/mamid/msp"
	"github.com/jinzhu/gorm"
	"log"
	"time"
)

type Monitor struct {
	DB              *gorm.DB
	BusWriteChannel chan<- interface{}
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

func mongodTuple(s model.Slave, m msp.Mongod) string {
	return fmt.Sprintf("(%s(id=%d),%d,%s)", s.Hostname, s.ID, m.Port, m.ReplicaSetName)
}

func (m *Monitor) observeSlave(slave model.Slave) {
	//Request mongod states from slave
	observedMongods, mspError := m.MSPClient.RequestStatus(msp.HostPort{slave.Hostname, msp.PortNumber(slave.Port)})

	// Notify about reachablility
	comErr := msp.Error{}
	if mspError != nil {
		comErr = *mspError
	}
	m.BusWriteChannel <- model.ConnectionStatus{
		Slave:              slave,
		Unreachable:        mspError != nil && mspError.Identifier == msp.CommunicationError,
		CommunicationError: comErr,
	}
	// TODO do we need to write this to the DB (currently there is no field for this in model.Slave)

	if mspError == nil {
		tx := m.DB.Begin()
		for _, observedMongod := range observedMongods {

			var dbMongod model.Mongod
			dbMongodRes := tx.First(&dbMongod, &model.Mongod{
				ParentSlaveID: slave.ID,
				Port:          model.PortNumber(observedMongod.Port),
				ReplSetName:   observedMongod.ReplicaSetName,
			})

			if dbMongodRes.RecordNotFound() {
				log.Printf("monitor: internal inconsistency: did not find corresponding database Mongod to observed Mongod `%s`: %s",
					mongodTuple(slave, observedMongod), dbMongodRes.Error)
				tx.Rollback()
				return
			} else if dbMongodRes.Error != nil {
				log.Printf("monitor: database error when querying for Mongod corresponding to observed Mongod `%s`: %s",
					mongodTuple(slave, observedMongod), dbMongodRes.Error)
				tx.Rollback()
				return
			}

			//Get desired state if it exists
			relatedResult := tx.Model(&dbMongod).Related(&dbMongod.DesiredState, "DesiredState")
			if !relatedResult.RecordNotFound() && relatedResult.Error != nil {
				log.Printf("monitor: internal inconsistency: could not get desired state for Mongod `%s`: %s",
					mongodTuple(slave, observedMongod), relatedResult.Error.Error())
				tx.Rollback()
				return
			}

			//Get observed state if it exists
			relatedResult = tx.Model(&dbMongod).Related(&dbMongod.ObservedState, "ObservedState")
			if !relatedResult.RecordNotFound() && relatedResult.Error != nil {
				log.Printf("monitor: database error when querying for observed state of Mongod `%s`: %s",
					mongodTuple(slave, observedMongod), relatedResult.Error)
				tx.Rollback()
				return
			}

			if observedMongod.StatusError == nil {
				//TODO Finish this
				//Put observations into model
				dbMongod.ObservedState.ExecutionState = mspMongodStateToModelExecutionState(observedMongod.State)
				dbMongod.ObservedState.IsShardingConfigServer = observedMongod.ShardingConfigServer
				dbMongod.ObservationError = model.MSPError{}
			} else {
				dbMongod.ObservationError = mspErrorToModelMSPError(observedMongod.StatusError)
			}

			//TODO Only update observed state and errors to prevent collisions with cluster allocator
			saveErr := tx.Save(&dbMongod).Error
			if saveErr != nil {
				log.Println(saveErr.Error())
				tx.Rollback()
				return
			}

		}

		//Remove observed state of mongods the slave does not report
		var modelMongods []model.Mongod
		getMongodsErr := tx.Model(&slave).Related(&modelMongods, "Mongods").Error
		if getMongodsErr != nil {
			log.Println(getMongodsErr.Error())
			tx.Rollback()
			return
		}
	outer:
		for _, modelMongod := range modelMongods {
			//Check if slave reported this mongod
			for _, observedMongod := range observedMongods {
				if modelMongod.Port == model.PortNumber(observedMongod.Port) &&
					modelMongod.ReplSetName == observedMongod.ReplicaSetName {
					continue outer
				}
			}

			//Else remove observed state
			deleteErr := tx.Delete(&model.MongodState{}, "id = ?", modelMongod.ObservedStateID).Error
			if deleteErr != nil {
				log.Println(deleteErr.Error())
				tx.Rollback()
				return
			}
		}

		tx.Commit()

		//Check every mongod for mismatches
		for _, modelMongod := range modelMongods {
			m.BusWriteChannel <- compareStates(modelMongod)
		}
	} else {
		//TODO Handle other slave errors => check identifiers != CommunicationError
		//log.Printf("monitor: error observing slave: %#v", mspError)
		return
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

func mspErrorToModelMSPError(mspError *msp.Error) model.MSPError {
	return model.MSPError{
		Identifier:      mspError.Identifier,
		Description:     mspError.Description,
		LongDescription: mspError.LongDescription,
	}
}
