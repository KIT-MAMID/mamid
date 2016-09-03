package master

import (
	"github.com/KIT-MAMID/mamid/model"
	"github.com/KIT-MAMID/mamid/msp"
	"github.com/stretchr/testify/assert"
	_ "sync"
	"testing"
)

func TestDeployer_mspMongodStateFromExecutionState_errorBehavior(t *testing.T) {

	var err error
	//var executionState msp.MongodState

	_, err = mspMongodStateFromExecutionState(0)
	assert.Error(t, err)

	//executionState = executionState

}

func TestDeployer_mspMongodStateRepresentation(t *testing.T) {

	var hostPort msp.HostPort
	var mspMongod msp.Mongod
	var err error

	db, err := createDB(t)
	defer db.CloseAndDrop()
	assert.Nil(t, err)

	d := Deployer{
		DB: db,
	}

	tx := db.Begin()
	defer tx.Rollback()

	var dbMongod model.Mongod
	var parentSlave model.Slave
	var desiredState model.MongodState
	assert.Nil(t, tx.First(&dbMongod).Error)
	assert.Nil(t, tx.Model(&dbMongod).Related(&parentSlave, "ParentSlave").Error)
	assert.Nil(t, tx.Model(&dbMongod).Related(&desiredState, "DesiredState").Error)
	dbMongod.ParentSlave = &parentSlave
	dbMongod.DesiredState = desiredState

	hostPort, mspMongod, err = d.mspMongodStateRepresentation(tx, model.Mongod{ID: 1})
	assert.NotNil(t, err, "Should not be able to find hostPort for Mongod without ParentSlaveID")
	assert.Zero(t, hostPort)

	hostPort, mspMongod, err = d.mspMongodStateRepresentation(tx, model.Mongod{
		ParentSlaveID: dbMongod.ParentSlaveID,
	})
	assert.NotNil(t, err, "Should not be able to find hostPort for Mongod without DesiredStateID")

	hostPort, mspMongod, err = d.mspMongodStateRepresentation(tx, dbMongod)
	assert.Nil(t, err, "ParentSlaveID and DesiredStateID should suffice to build MSP MongodState representation")

	assert.EqualValues(t, msp.HostPort{dbMongod.ParentSlave.Hostname, msp.PortNumber(dbMongod.ParentSlave.Port)}, hostPort)

	expectedMongodState, _ := mspMongodStateFromExecutionState(dbMongod.DesiredState.ExecutionState)

	assert.Equal(t, msp.Mongod{
		Port: msp.PortNumber(dbMongod.Port),
		ReplicaSetConfig: msp.ReplicaSetConfig{
			ReplicaSetName: dbMongod.ReplSetName,
			ShardingRole:   msp.ShardingRole(dbMongod.DesiredState.ShardingRole),
			ReplicaSetMembers: []msp.ReplicaSetMember{msp.ReplicaSetMember{
				HostPort: msp.HostPort{Hostname: "host1", Port: 2000},
				Priority: ReplicaSetMemberPriorityLow,
			}},
			RootCredential: msp.MongodCredential{Username: "user", Password: "pass"},
		},
		KeyfileContent: "keyfile",

		// TODO: this is hardcoded knowlege about the contents of the test database.
		// Use something auto-generated instead.
		// Also: is this field actually relevant in an EstablishState call?
		State: expectedMongodState,
	}, mspMongod)

}

func TestDeployer_mspDesiredReplicaSetMembersForMongod(t *testing.T) {

	var err error

	db, err := createDB(t)
	defer db.CloseAndDrop()
	assert.Nil(t, err)

	tx := db.Begin()
	defer tx.Rollback()

	var dbMongod model.Mongod
	var parentSlave model.Slave
	var desiredState model.MongodState
	assert.Nil(t, tx.First(&dbMongod).Error)
	assert.Nil(t, tx.Model(&dbMongod).Related(&parentSlave, "ParentSlave").Error)
	assert.Nil(t, tx.Model(&dbMongod).Related(&desiredState, "DesiredState").Error)

	var members []msp.ReplicaSetMember

	// Test for one slave in DB
	members, err = DesiredMSPReplicaSetMembersForReplicaSetID(tx, *model.NullIntToPtr(dbMongod.ReplicaSetID))
	assert.Nil(t, err)
	assert.EqualValues(t, 1, len(members))
	assert.EqualValues(t, msp.ReplicaSetMember{HostPort: msp.HostPort{Hostname: parentSlave.Hostname, Port: msp.PortNumber(dbMongod.Port)}, Priority: ReplicaSetMemberPriorityLow}, members[0],
		"the list of replica set members of mongod m should include mongod m") // TODO do we actually want this?

	// Set the desired state to not running
	assert.EqualValues(t, 1, tx.Model(&desiredState).Update("ExecutionState", model.MongodExecutionStateNotRunning).RowsAffected)
	members, err = DesiredMSPReplicaSetMembersForReplicaSetID(tx, *model.NullIntToPtr(dbMongod.ReplicaSetID))
	assert.Nil(t, err)
	assert.EqualValues(t, 0, len(members),
		"a mongod with desired execution state != running should have no replica set members")

	// TODO test for multiple mongods and replica sets

}
