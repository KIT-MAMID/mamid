package master

import (
	"database/sql"
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

				type observation struct {
					result   []msp.Mongod
					err      *msp.Error
					theSlave model.Slave
				}
				observationChan := make(chan observation)

				//Observe active slaves
				for _, slave := range slaves {
					wg.Add(1)
					go func(s model.Slave) {
						//Request mongod states from slave
						observedMongods, mspError := m.MSPClient.RequestStatus(msp.HostPort{s.Hostname, msp.PortNumber(s.Port)})
						observationChan <- observation{
							result:   observedMongods,
							err:      mspError,
							theSlave: s, //Do not call this slave or vet will fail
						}
						wg.Done()
					}(slave)
				}

				//Wait for all slaves to be observed and close channel to make consumer loop break
				go func() {
					wg.Wait()
					close(observationChan)
				}()

				//Consumer loop that saves result to database
				//We do this so that all transactions happen after eachother == prevent concurrent database access
				for observationRes := range observationChan {
					m.handleObservation(observationRes.result, observationRes.err, observationRes.theSlave)
				}

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
	return fmt.Sprintf("(%s(id=%d),%d,%s)", s.Hostname, s.ID, m.Port, m.ReplicaSetConfig.ReplicaSetName)
}

func (m *Monitor) handleObservation(observedMongods []msp.Mongod, mspError *msp.Error, slave model.Slave) {
	// Notify about reachablility
	comErr := msp.Error{}
	if mspError != nil {
		//TODO Handle other slave errors => check identifiers != CommunicationError
		monitorLog.Errorf("monitor: error observing slave: %#v", mspError)
		comErr = *mspError
	}
	m.BusWriteChannel <- model.ConnectionStatus{
		Slave:              slave,
		Unreachable:        mspError != nil && mspError.Identifier == msp.CommunicationError,
		CommunicationError: comErr,
	}

	tx := m.DB.Begin()

	if err := m.updateObservedStateInDB(tx, slave, mspError, observedMongods); err != nil {
		monitorLog.WithError(err).Error()
		tx.Rollback()
		return
	}

	// Return early if there were observation errors.
	if mspError != nil {
		if err := tx.Commit().Error; err != nil {
			monitorLog.WithError(err).Error("could not commit monitor run")
		}
		return
	}

	// NOTE: from now on, we assume observedMongods to be valid.

	if err := m.handleUnobservedMongodsOfSlave(tx, slave, observedMongods); err != nil {
		monitorLog.WithError(err).Error()
		tx.Rollback()
		return
	}

	if err := tx.Commit().Error; err != nil {
		monitorLog.WithError(err).Error("Could not commit monitor run")
	}

	// Read-only transaction
	tx = m.DB.Begin()
	defer tx.Rollback()
	if err := m.sendMongodMismatchStatusToBus(tx, slave); err != nil {
		monitorLog.WithError(err).Error()
	}

}

// Update database Mongod.ObservedState with newly observedMongods
// Errors returned by this method should be handled by aborting the transaction tx
func (m *Monitor) updateObservedStateInDB(tx *gorm.DB, slave model.Slave, slaveObservationError *msp.Error, observedMongods []msp.Mongod) (criticalError error) {

	if slaveObservationError != nil { // update observation error field

		monitorLog.Debugf("monitor: persisting observation error for slave `%s:%d` in database", slave.Hostname, slave.Port)

		var updateError error
		modelObservationErr := mspErrorToModelMSPError(slaveObservationError)
		if slave.ObservationErrorID.Valid {
			// Replace existing entry
			monitorLog.Debugf("monitor: replacing existing observation error for slave `%s:%d` in database", slave.Hostname, slave.Port)
			modelObservationErr.ID = slave.ObservationErrorID.Int64
			updateError = tx.Save(&modelObservationErr).Error
		} else {
			monitorLog.Debugf("monitor: creating observation error for slave `%s:%d` in database", slave.Hostname, slave.Port)
			updateError = tx.Create(&modelObservationErr).Error
			if updateError == nil {
				updateError = tx.Model(&slave).Update("ObservationErrorID", modelObservationErr.ID).Error
			}
		}

		if updateError != nil {
			return fmt.Errorf("monitor: database error when updating slave `%s:%d` ObservationErrorID field: %s", slave.Hostname, slave.Port, updateError)
		}

		// return early as there should not be observedMongods in case of slaveObservationError
		return nil

	} else if slave.ObservationErrorID.Valid { // clear observation error field

		monitorLog.Debugf("monitor: clearing observation error for slave `%s:%d` in database", slave.Hostname, slave.Port)

		res := tx.Exec(`DELETE FROM msp_errors WHERE id=?`, slave.ObservationErrorID.Int64)
		switch {
		case res.Error != nil:
			return fmt.Errorf("monitor: database error when clearing observation error for slave `%s:%d`: %s", slave.Hostname, slave.Port, res.Error)
		case res.RowsAffected == 0:
			return fmt.Errorf("monitor: clearing observation error for slave `%s:%d` affected 0 rows", slave.Hostname, slave.Port)
		case res.RowsAffected == 1:
			monitorLog.Debugf("monitor: cleared observation error for slave `%s:%d`", slave.Hostname, slave.Port)
		case res.RowsAffected > 1:
			monitorLog.Warnf("monitor: clearing observation error for slave `%s:%d` affected %d != 1 rows", slave.Hostname, slave.Port, res.RowsAffected)
		}

	}

	// NOTE: we assume observedMonogds to be valid from this point on (caught by early return in case of observation error)

	for _, observedMongod := range observedMongods {

		monitorLog.Debugf("monitor: updating observed state for mongod `%s` in database`", mongodTuple(slave, observedMongod))

		var dbMongod model.Mongod

		dbMongodRes := tx.First(&dbMongod, &model.Mongod{
			ParentSlaveID: slave.ID,
			Port:          model.PortNumber(observedMongod.Port),
			ReplSetName:   observedMongod.ReplicaSetConfig.ReplicaSetName,
		})

		if dbMongodRes.Error != nil && !dbMongodRes.RecordNotFound() {
			// Early exit
			return fmt.Errorf("monitor: database error when querying for Mongod corresponding to observed Mongod `%s`: %s",
				mongodTuple(slave, observedMongod), dbMongodRes.Error)

		} else if dbMongodRes.RecordNotFound() {

			// The slave is running a Mongod which is not in the database
			// => model this in the database
			// => then continue as if the Mongod had always been in the database

			dbMongod = model.Mongod{
				ParentSlaveID: slave.ID,
				Port:          model.PortNumber(observedMongod.Port),
				ReplSetName:   observedMongod.ReplicaSetConfig.ReplicaSetName,
				ReplicaSetID:  sql.NullInt64{Valid: false},
			}
			if err := tx.Create(&dbMongod).Error; err != nil {
				return fmt.Errorf("monitor: could not create database representation for unknown observed Mongod `%s`: %s",
					mongodTuple(slave, observedMongod), err)
			}
			desiredState := model.MongodState{
				ParentMongodID: dbMongod.ID,
				ExecutionState: model.MongodExecutionStateDestroyed,
			}
			if err := tx.Create(&desiredState).Error; err != nil {
				return fmt.Errorf("monitor: could not create desired MongodState for unknown observed Mongod `%s`: %s",
					mongodTuple(slave, observedMongod), err)
			}
			if err := tx.Model(&dbMongod).Update("DesiredStateID", desiredState.ID).Error; err != nil {
				return fmt.Errorf("monitor: could not update DesiredStateID column for unknown observed Mongod `%s`: %s",
					mongodTuple(slave, observedMongod), err)
			}

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
			dbMongod.ObservedState.ParentMongodID = dbMongod.ID // we could be creating the ObservedState of Mongod on first observation
			dbMongod.ObservedState.ExecutionState = mspMongodStateToModelExecutionState(observedMongod.State)
			dbMongod.ObservedState.IsShardingConfigServer = observedMongod.ReplicaSetConfig.ShardingConfigServer
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

		monitorLog.Debugf("monitor: finished updating observed state for mongod `%s` in database`", mongodTuple(slave, observedMongod))

	}

	return nil

}

// Remove observed state of mongods the slave does not report
// Errors returned by this method should be handled by aborting the transaction tx
func (m *Monitor) handleUnobservedMongodsOfSlave(tx *gorm.DB, slave model.Slave, observedMongods []msp.Mongod) (err error) {

	monitorLog.Debugf("monitor: handling unobserved Mongods of slave `%s`", slave.Hostname)

	var modelMongods []model.Mongod
	if err := tx.Model(&slave).Related(&modelMongods, "Mongods").Error; err != nil {
		return err
	}

outer:
	for _, modelMongod := range modelMongods {

		//Check if slave reported this mongod
		for _, observedMongod := range observedMongods {
			if modelMongod.Port == model.PortNumber(observedMongod.Port) &&
				modelMongod.ReplSetName == observedMongod.ReplicaSetConfig.ReplicaSetName {
				continue outer
			}
		}

		if modelMongod.ObservedStateID.Valid {

			// We didn't find the expected modelMongod on the slave
			// => delete the observed state to indicate this to other components

			monitorLog.Infof("removing observed state of Mongod `%s:%d` as it was not reported by slave `%s`", slave.Hostname, modelMongod.Port, slave.Hostname)
			deleteErr := tx.Delete(&model.MongodState{ID: modelMongod.ObservedStateID.Int64}).Error
			if deleteErr != nil {
				monitorLog.Errorf("error removing observed state of Mongod `%s:%d`: %s", slave.Hostname, modelMongod.Port, deleteErr)
				return deleteErr
			}

		}
	}

	monitorLog.Debugf("monitor: finished handling unobserved Mongods of slave `%s`", slave.Hostname)

	return nil

}

// Check every Mongod of the Slave for mismatches between DesiredState and ObservedState
// and send an appropriate MongodMismatchStatus to the Bus
func (m *Monitor) sendMongodMismatchStatusToBus(tx *gorm.DB, slave model.Slave) (err error) {

	monitorLog.Debugf("monitor: preparing Mongod Mismatch Status messages for slave `%s`", slave.Hostname)

	var modelMongods []model.Mongod
	if err := tx.Model(&slave).Related(&modelMongods, "Mongods").Error; err != nil {
		return err
	}

	for _, modelMongod := range modelMongods {

		var busMessage interface{}

		busMessage, err = m.compareStates(tx, modelMongod)
		if err != nil {
			monitorLog.Errorf("error comparing Mongod `%d on %s` Desired and Observed state: %s", modelMongod.ID, slave.Hostname, err)
			continue
		}

		monitorLog.Debugf("monitor: sending bus message for slave `%s`", slave.Hostname, busMessage)
		m.BusWriteChannel <- busMessage
		monitorLog.Debugf("monitor: sent bus message for slave `%s`", slave.Hostname, busMessage)

	}

	monitorLog.Debugf("monitor: finished sending Mongod Mismatch Status messages for slave `%s`", slave.Hostname)

	return nil
}

func (m *Monitor) compareStates(tx *gorm.DB, mongod model.Mongod) (s model.MongodMatchStatus, err error) {

	if !mongod.ObservedStateID.Valid {
		// If we have no observations, it is a mismatch (since we can't know what the actual state is)
		// Example: a new Mongod that has never been observed. Will be deployed by the Deployer when informed about Mismatch
		return model.MongodMatchStatus{
			Mismatch: true,
			Mongod:   mongod,
		}, nil
	}

	if err = tx.Model(mongod).Related(&mongod.DesiredState, "DesiredState").Error; err != nil {
		return s, fmt.Errorf("error fetching DesiredState for mongod `%v`: %s", mongod, err)
	}

	observedStateRes := tx.Model(mongod).Related(&mongod.ObservedState, "ObservedState")
	if observedStateRes.Error != nil {
		return s, fmt.Errorf("error fetching ObservedState for mongod `%v`: %s", mongod, observedStateRes.Error)
	}

	equivalent, err := m.mongodStatesEquivalent(tx, mongod.DesiredState, mongod.ObservedState)
	if err != nil {
		return s, fmt.Errorf("error comparing Mongod states: %s", err)
	}

	return model.MongodMatchStatus{
		Mismatch: !equivalent,
		Mongod:   mongod,
	}, nil

}

func (m *Monitor) mongodStatesEquivalent(tx *gorm.DB, a, b model.MongodState) (e bool, err error) {

	e = a.ExecutionState == b.ExecutionState
	e = e && a.IsShardingConfigServer == b.IsShardingConfigServer

	// Compare []ReplicaSetMembers are equal
	// Since Go's `sort` package is a pain and Go has no sets
	// use SQL (we need to fetch the ReplicaSetMembers anyway
	getSortedMembers := func(tx *gorm.DB, s model.MongodState, out *[]model.ReplicaSetMember) (err error) {
		return tx.Order("hostname").Order("port").Find(out, model.ReplicaSetMember{MongodStateID: s.ID}).Error
	}
	var (
		aMembers = make([]model.ReplicaSetMember, 0, 0)
		bMembers = make([]model.ReplicaSetMember, 0, 0)
	)
	if err = getSortedMembers(tx, a, &aMembers); err != nil {
		return false, err
	}
	if err = getSortedMembers(tx, b, &bMembers); err != nil {
		return false, err
	}

	e = e && compareSortedReplicaSetMembers(aMembers, bMembers)

	return e, nil

}

func compareSortedReplicaSetMembers(a, b []model.ReplicaSetMember) (equal bool) {

	if len(a) != len(b) {
		return false
	}

	for i, _ := range a {
		if !ReplicaSetMembersEquivalent(a[i], b[i]) {
			return false
		}
	}

	return true

}

func ReplicaSetMembersEquivalent(a, b model.ReplicaSetMember) bool {
	return a.Hostname == b.Hostname && a.Port == b.Port
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
	case msp.MongodStateForceDestroyed:
		return model.MongodExecutionStateForceDestroyed
	default:
		return 0 // Invalid
		//TODO New states
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
	defer tx.Rollback()

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
	if err != nil {
		monitorLog.WithError(err).Error("Error getting configured and actual member counts of replica sets")
		return
	}
	defer replicaSetsWithMemberCounts.Close()

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

		m.BusWriteChannel <- model.ReplicaSetInitiationStatus{
			Initiated:  replicaSet.Initiated,
			ReplicaSet: replicaSet,
		}

	}

	replicaSetsWithMemberCounts.Close()

}
