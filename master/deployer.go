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
			d.handleMatchStatus(msg.(MongodMatchStatus))
		}
	}
}

func (d *Deployer) handleMatchStatus(m MongodMatchStatus) {
	if !m.Mismatch {
		return
	}
	d.pushMongodState(m.Mongod)
}

func (d *Deployer) pushMongodState(mongod Mongod) {

	// Readonly tx
	tx := d.DB.Begin()

	hostPort, mspMongod, err := d.mspMongodStateRepresentation(tx, mongod)
	if err != nil {
		deployerLog.Println(err)
	}
	// Readonly tx
	tx.Rollback()

	mspError := d.MSPClient.EstablishMongodState(hostPort, mspMongod)
	if mspError != nil {
		deployerLog.Printf("deployer: MSP error establishing mongod state for Mongod `(%v(id=%d),%d,)` in Replica Set `%s`: %s",
			mongod.ParentSlave, mongod.ParentSlaveID, mongod.Port, mongod.ReplSetName, mspError)
	}

}

// Generate an MSP-compatible representation of the deisred Mongod state
// uses tx readonly
// When err != nil is returned, the tx. should be rolled back and the error be reported
func (d *Deployer) mspMongodStateRepresentation(tx *gorm.DB, mongod Mongod) (hostPort msp.HostPort, mspMongod msp.Mongod, err error) {

	var slave Slave
	var desiredState MongodState
	var mspMongodState msp.MongodState
	var replicaSetMembers []msp.HostPort

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
	if replicaSetMembers, err = mspDesiredReplicaSetMembersForMongod(tx, mongod); err != nil {
		return
	}

	// Construct msp representation
	hostPort = msp.HostPort{
		Hostname: slave.Hostname,
		Port:     msp.PortNumber(slave.Port),
	}
	mspMongod = msp.Mongod{
		Port:                 msp.PortNumber(mongod.Port),
		ReplicaSetName:       mongod.ReplSetName,
		ReplicaSetMembers:    replicaSetMembers,
		ShardingConfigServer: desiredState.IsShardingConfigServer,
		State:                mspMongodState,
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

// Return the list of msp.HostPort a Mongod should have as members
// Includes the mongod passed as parameter m
func mspDesiredReplicaSetMembersForMongod(tx *gorm.DB, m Mongod) (replicaSetMembers []msp.HostPort, err error) {

	res := tx.Raw(`SELECT s.hostname, m.port
		FROM mongods m
		JOIN replica_sets r ON m.replica_set_id = r.id
		JOIN mongod_states desired_state ON m.desired_state_id = desired_state.id
		JOIN slaves s ON m.parent_slave_id = s.id
		WHERE r.id = ?
		      AND desired_state.execution_state = ?
		`, m.ReplicaSetID, MongodExecutionStateRunning,
	).Find(&replicaSetMembers)

	if res.Error != nil {
		return []msp.HostPort{}, fmt.Errorf("deployer: could not fetch replica set members for mongod `%v`: %s", m, res.Error)
	}

	return

}
