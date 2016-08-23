package master

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/KIT-MAMID/mamid/msp"
	"github.com/Sirupsen/logrus"
	"github.com/jinzhu/gorm"
	"sync"
	"time"
)

var monitorLog = logrus.WithField("module", "monitor")

type Monitor struct {
	DB              *model.DB
	BusWriteChannel chan<- interface{}
	MSPClient       msp.MSPClient
	Interval        time.Duration
}

func (m *Monitor) Run() {
	ticker := time.NewTicker(m.Interval)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				monitorLog.Info("Monitor running")

				//Get all slaves from database
				tx := m.DB.Begin()
				var slaves []model.Slave
				err := tx.Find(&slaves).Error
				if err != nil {
					monitorLog.WithError(err).Error("Could not get slaves")
				}
				tx.Rollback()

				wg := sync.WaitGroup{}

				//Observe active slaves
				for _, slave := range slaves {
					if slave.ConfiguredState == model.SlaveStateActive {
						wg.Add(1)
						go func(s model.Slave) {
							m.observeSlave(s)
							wg.Done()
						}(slave)
					}
				}

				//Wait for all slaves to be observed
				wg.Wait()

				//Check degradation of replica sets
				m.observeReplicaSets()

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

	if mspError != nil {
		//TODO Handle other slave errors => check identifiers != CommunicationError
		//monitorLog.Error("monitor: error observing slave: %#v", mspError)
		return
	}

	tx := m.DB.Begin()

	if err := m.updateObservedStateInDB(tx, slave, observedMongods); err != nil {
		monitorLog.WithError(err).Error()
		tx.Rollback()
		return
	}

	if err := m.handleUnobservedMongodsOfSlave(tx, slave, observedMongods); err != nil {
		monitorLog.WithError(err).Error()
		tx.Rollback()
		return
	}

	tx.Commit()

	// Read-only transaction
	tx = m.DB.Begin()
	defer tx.Rollback()
	if err := m.sendMongodMismatchStatusToBus(tx, slave); err != nil {
		monitorLog.WithError(err).Error()
	}

}

// Update database Mongod.ObservedState with newly observedMongods
// Errors returned by this method should be handled by aborting the transaction tx
func (m *Monitor) updateObservedStateInDB(tx *gorm.DB, slave model.Slave, observedMongods []msp.Mongod) (criticalError error) {

	for _, observedMongod := range observedMongods {

		monitorLog.Debug("monitor: updating observed state for mongod `%s` in database`", mongodTuple(slave, observedMongod))

		var dbMongod model.Mongod

		dbMongodRes := tx.First(&dbMongod, &model.Mongod{
			ParentSlaveID: slave.ID,
			Port:          model.PortNumber(observedMongod.Port),
			ReplSetName:   observedMongod.ReplicaSetName,
		})

		if dbMongodRes.RecordNotFound() {
			return fmt.Errorf("monitor: internal inconsistency: did not find corresponding database Mongod to observed Mongod `%s`: %s",
				mongodTuple(slave, observedMongod), dbMongodRes.Error)
		} else if dbMongodRes.Error != nil {
			return fmt.Errorf("monitor: database error when querying for Mongod corresponding to observed Mongod `%s`: %s",
				mongodTuple(slave, observedMongod), dbMongodRes.Error)
		}

		//Get desired state if it exists
		relatedResult := tx.Model(&dbMongod).Related(&dbMongod.DesiredState, "DesiredState")
		if !relatedResult.RecordNotFound() && relatedResult.Error != nil {
			return fmt.Errorf("monitor: internal inconsistency: could not get desired state for Mongod `%s`: %s",
				mongodTuple(slave, observedMongod), relatedResult.Error.Error())
		}

		//Get observed state if it exists
		relatedResult = tx.Model(&dbMongod).Related(&dbMongod.ObservedState, "ObservedState")
		if !relatedResult.RecordNotFound() && relatedResult.Error != nil {
			return fmt.Errorf("monitor: database error when querying for observed state of Mongod `%s`: %s",
				mongodTuple(slave, observedMongod), relatedResult.Error)
		}

		// Update database representation of observation
		if observedMongod.StatusError == nil {
			//TODO Finish this
			//Put observations into model
			dbMongod.ObservedState.ExecutionState = mspMongodStateToModelExecutionState(observedMongod.State)
			dbMongod.ObservedState.IsShardingConfigServer = observedMongod.ShardingConfigServer
			dbMongod.ObservationError = model.MSPError{}
		} else {
			dbMongod.ObservationError = mspErrorToModelMSPError(observedMongod.StatusError)
		}

		// Persist updated database representation
		//TODO Only update observed state and errors to prevent collisions with cluster allocator
		saveErr := tx.Save(&dbMongod).Error
		if saveErr != nil {
			return fmt.Errorf("monitor: error persisting updated observed state for mongod `%s`: %s",
				mongodTuple(slave, observedMongod), saveErr.Error())
		}

		monitorLog.Debug("monitor: finished updating observed state for mongod `%s` in database`", mongodTuple(slave, observedMongod))

	}

	return nil

}

// Remove observed state of mongods the slave does not report
// Errors returned by this method should be handled by aborting the transaction tx
func (m *Monitor) handleUnobservedMongodsOfSlave(tx *gorm.DB, slave model.Slave, observedMongods []msp.Mongod) (err error) {

	var modelMongods []model.Mongod
	if err := tx.Model(&slave).Related(&modelMongods, "Mongods").Error; err != nil {
		return err
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
			return deleteErr
		}
	}

	return nil

}

// Check every Mongod of the Slave for mismatches between DesiredState and ObservedState
// and send an appropriate MongodMismatchStatus to the Bus
func (m *Monitor) sendMongodMismatchStatusToBus(tx *gorm.DB, slave model.Slave) (err error) {

	var modelMongods []model.Mongod
	if err := tx.Model(&slave).Related(&modelMongods, "Mongods").Error; err != nil {
		return err
	}

	for _, modelMongod := range modelMongods {

		if err := tx.Model(modelMongod).Related(&modelMongod.DesiredState, "DesiredState").Error; err != nil {
			return fmt.Errorf("monitor: error fetching DesiredState for mongod `%v`: %s", modelMongod, err)
		}

		observedStateRes := tx.Model(modelMongod).Related(&modelMongod.ObservedState, "ObservedState")
		if !observedStateRes.RecordNotFound() && observedStateRes.Error != nil {
			return fmt.Errorf("monitor: error fetching ObservedState for mongod `%v`: %s", modelMongod, observedStateRes.Error)
		}

		if observedStateRes.RecordNotFound() {
			// This happens when
			// 	a new Mongod with no observations is found
			// 	a Mongod with desired state = destroyed is still in the database
			m.BusWriteChannel <- model.MongodMatchStatus{
				Mismatch: true,
				Mongod:   modelMongod,
			}
		} else {
			m.BusWriteChannel <- compareStates(modelMongod)
		}

	}

	return nil
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

func (m *Monitor) observeReplicaSets() {
	tx := m.DB.Begin()

	// Get replica sets and the count of their actually configured members from the database
	replicaSetsWithMemberCounts, err := tx.Raw(`SELECT
				r.*,
				(SELECT COUNT(*) FROM replica_set_configured_members WHERE replica_set_id = r.id AND persistent_storage = ?)
					AS configured_persistent_members,
				(SELECT COUNT(*) FROM replica_set_configured_members WHERE replica_set_id = r.id AND persistent_storage = ?)
					AS configured_volatile_members,
				(SELECT COUNT(*) FROM replica_set_effective_members WHERE replica_set_id = r.id AND persistent_storage = ?)
					AS actual_persistent_members,
				(SELECT COUNT(*) FROM replica_set_effective_members WHERE replica_set_id = r.id AND persistent_storage = ?)
					AS actual_volatile_members
				FROM replica_sets r
				`, true, false, true, false).Rows()
	tx.Rollback()
	if err != nil {
		monitorLog.WithError(err).Error("Error getting configured and actual member counts of replica sets")
		return
	}

	for replicaSetsWithMemberCounts.Next() {
		var replicaSet model.ReplicaSet
		tx.ScanRows(replicaSetsWithMemberCounts, &replicaSet)

		memberCounts := struct {
			ConfiguredPersistentMembers uint
			ConfiguredVolatileMembers   uint
			ActualPersistentMembers     uint
			ActualVolatileMembers       uint
		}{}
		tx.ScanRows(replicaSetsWithMemberCounts, &memberCounts)

		unsatisfied := memberCounts.ConfiguredVolatileMembers > memberCounts.ActualVolatileMembers ||
			memberCounts.ConfiguredPersistentMembers > memberCounts.ActualPersistentMembers

		m.BusWriteChannel <- model.ObservedReplicaSetConstraintStatus{
			Unsatisfied:               unsatisfied,
			ReplicaSet:                replicaSet,
			ConfiguredPersistentCount: memberCounts.ConfiguredPersistentMembers,
			ConfiguredVolatileCount:   memberCounts.ConfiguredVolatileMembers,
			ActualPersistentCount:     memberCounts.ActualPersistentMembers,
			ActualVolatileCount:       memberCounts.ActualVolatileMembers,
		}
	}
}
