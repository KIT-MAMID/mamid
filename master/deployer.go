package master

import (
	"fmt"
	. "github.com/KIT-MAMID/mamid/model"
	"github.com/KIT-MAMID/mamid/msp"
	"github.com/Sirupsen/logrus"
	"github.com/jinzhu/gorm"
)

var deployerLog = logrus.WithField("module", "deployer")

/*
  Listens on the bus for state mismatches and tries to solve them by pushing the desired state to the Mongod
*/
type Deployer struct {
	DB             *DB
	MSPClient      msp.MSPClient
	BusReadChannel <-chan interface{}
}

func (d *Deployer) Run() {

	for {
		msg := <-d.BusReadChannel
		switch msg.(type) {
		case MongodMatchStatus:
			go d.handleMatchStatus(msg.(MongodMatchStatus))
		case ReplicaSetInitiationStatus:
			go d.handleReplicaSetInitiationStatus(msg.(ReplicaSetInitiationStatus))
		}

	}
}

func (d *Deployer) handleMatchStatus(m MongodMatchStatus) {
	if !m.Mismatch {
		return
	}
	d.pushMongodState(m.Mongod)
}

func (d *Deployer) handleReplicaSetInitiationStatus(s ReplicaSetInitiationStatus) {

	if s.Initiated {
		return
	}

	var msg msp.RsInitiateMessage
	var mspErr *msp.Error

	tx := d.DB.Begin()

	slave, initiator, err := d.findInitiatorForReplicaSet(tx, s.ReplicaSet)
	if err != nil {
		deployerLog.WithError(err).Errorf("could not find initiator for replica set `%s`", s.ReplicaSet.Name)
		goto rollbackAndReturn
	}

	msg = msp.RsInitiateMessage{
		Port: msp.PortNumber(initiator.Port),
	}

	msg.ReplicaSetConfig, err = d.replicaSetConfig(tx, s.ReplicaSet)
	if err != nil {
		deployerLog.WithError(err).Errorf("could not generate replica set config `%s`", s.ReplicaSet.Name)
		goto rollbackAndReturn
	}

	deployerLog.Debugf("initializing Replica Set `%s` from `%s` using message: %#v", s.ReplicaSet.Name, slave.Hostname, msg)

	mspErr = d.MSPClient.InitiateReplicaSet(msp.HostPort{slave.Hostname, msp.PortNumber(slave.Port)}, msg)

	if mspErr != nil {
		deployerLog.Errorf("error initializing Replica Set `%s` from `%s`: %s", s.ReplicaSet.Name, slave.Hostname, mspErr)
	} else {

		if err = tx.Model(&s.ReplicaSet).Update("Initiated", true).Error; err != nil {
			deployerLog.Errorf("error initializing Replica Set `%s` from `%s`: %s", s.ReplicaSet.Name, slave.Hostname, mspErr)
			goto rollbackAndReturn
		}
		tx.Commit()
	}

	return

rollbackAndReturn:
	tx.Rollback()
	return

}

// Find a Mongod of ReplicaSet r for `initiiating` the ReplicaSet
func (d *Deployer) findInitiatorForReplicaSet(tx *gorm.DB, r ReplicaSet) (s Slave, m Mongod, err error) {

	err = tx.Raw(`
		SELECT m.*
		FROM mongods m
		JOIN replica_sets r ON m.replica_set_id = r.id
		WHERE
			r.initiated = false
			AND
			m.replica_set_id = ?
		LIMIT 1
			`, r.ID).Scan(&m).Error
	if err != nil {
		return
	}

	if err = tx.Model(&m).Related(&s, "ParentSlaveID").Error; err != nil {
		return
	}

	return

}

func (d *Deployer) pushMongodState(mongod Mongod) {

	deployerLog.Debugf("fetch Mongod state representation: `%d` on slave `%d`", mongod.ID, mongod.ParentSlaveID)
	// Readonly tx
	tx := d.DB.Begin()

	hostPort, mspMongod, err := d.mspMongodStateRepresentation(tx, mongod)
	if err != nil {
		deployerLog.Println(err)
	}
	// Readonly tx
	tx.Rollback()
	deployerLog.Debugf("finish fetching Mongod state representation: `%d` on slave `%d`", mongod.ID, mongod.ParentSlaveID)

	deployerLog.Debugf("establishing Mongod state on `%s` (%#v)", hostPort, mspMongod)

	mspError := d.MSPClient.EstablishMongodState(hostPort, mspMongod)
	if mspError != nil {
		deployerLog.Errorf("MSP error establishing mongod state on `%s` for Mongod `(%v(id=%d),%d,)` in Replica Set `%s`: %s",
			hostPort, mongod.ParentSlave, mongod.ParentSlaveID, mongod.Port, mongod.ReplSetName, mspError)
	} else {
		deployerLog.Debugf("finished establishing Mongod state on %s", hostPort)
	}

}

// Generate an MSP-compatible representation of the deisred Mongod state
// uses tx readonly
// When err != nil is returned, the tx. should be rolled back and the error be reported
func (d *Deployer) mspMongodStateRepresentation(tx *gorm.DB, mongod Mongod) (hostPort msp.HostPort, mspMongod msp.Mongod, err error) {

	var slave Slave
	var desiredState MongodState
	var mspMongodState msp.MongodState
	var replicaSetMembers []msp.ReplicaSetMember

	// Fetch master representation
	if err = tx.Model(&mongod).Related(&slave, "ParentSlave").Error; err != nil {
		return
	}
	if err = tx.Model(&mongod).Related(&desiredState, "DesiredState").Error; err != nil {
		return
	}
	mspMongodState, err = mspMongodStateFromExecutionState(desiredState.ExecutionState)
	if err != nil {
		return
	}

	// TODO use DesiredState once this is set
	if !mongod.ReplicaSetID.Valid {
		replicaSetMembers = make([]msp.ReplicaSetMember, 0, 0)
	} else {
		if replicaSetMembers, err = mspDesiredReplicaSetMembersForReplicaSetID(tx, mongod.ReplicaSetID.Int64); err != nil {
			return
		}
	}

	// Construct msp representation
	hostPort = msp.HostPort{
		Hostname: slave.Hostname,
		Port:     msp.PortNumber(slave.Port),
	}
	mspMongod = msp.Mongod{
		Port: msp.PortNumber(mongod.Port),
		ReplicaSetConfig: msp.ReplicaSetConfig{
			ReplicaSetName:       mongod.ReplSetName,
			ReplicaSetMembers:    replicaSetMembers,
			ShardingConfigServer: desiredState.IsShardingConfigServer,
		},
		State: mspMongodState,
	}

	return

}

func mspMongodStateFromExecutionState(s MongodExecutionState) (msp.MongodState, error) {
	switch s {
	case MongodExecutionStateDestroyed:
		return msp.MongodStateDestroyed, nil
	case MongodExecutionStateNotRunning:
		return msp.MongodStateNotRunning, nil
	case MongodExecutionStateRecovering:
		return msp.MongodStateRecovering, nil
	case MongodExecutionStateRunning:
		return msp.MongodStateRunning, nil
	default:
		return "", fmt.Errorf("deployer: unable to map `%v` from model.ExecutionState to msp.MongodState", s)
	}
}

// Generate a ReplicaSetConfig used to describe the ReplicaSet r
func (d *Deployer) replicaSetConfig(tx *gorm.DB, r ReplicaSet) (config msp.ReplicaSetConfig, err error) {

	config = msp.ReplicaSetConfig{
		ReplicaSetName:       r.Name,
		ReplicaSetMembers:    make([]msp.ReplicaSetMember, 0, 0),
		ShardingConfigServer: r.ConfigureAsShardingConfigServer,
	}

	config.ReplicaSetMembers, err = mspDesiredReplicaSetMembersForReplicaSetID(tx, r.ID)

	return
}

// Return the list of msp.HostPort a Mongod should have as members
// Includes the mongod passed as parameter m
func mspDesiredReplicaSetMembersForReplicaSetID(tx *gorm.DB, replicaSetID int64) (replicaSetMembers []msp.ReplicaSetMember, err error) {

	rows, err := tx.Raw(`SELECT s.hostname, m.port
		FROM mongods m
		JOIN replica_sets r ON m.replica_set_id = r.id
		JOIN mongod_states desired_state ON m.desired_state_id = desired_state.id
		JOIN slaves s ON m.parent_slave_id = s.id
		WHERE r.id = ?
		      AND desired_state.execution_state = ?
		`, replicaSetID, MongodExecutionStateRunning,
	).Rows()
	defer rows.Close()

	if err != nil {
		return []msp.ReplicaSetMember{}, fmt.Errorf("could not fetch replica set members for ReplicaSet.ID `%v`: %s", replicaSetID, err)
	}

	for rows.Next() {
		member := msp.ReplicaSetMember{}
		err = rows.Scan(&member.HostPort.Hostname, &member.HostPort.Port)
		if err != nil {
			return
		}
		replicaSetMembers = append(replicaSetMembers, member)
	}

	return

}
