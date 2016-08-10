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
	var executionState msp.MongodState

	executionState, err = mspMongodStateFromExecutionState(0)
	assert.NotNil(t, err)

	executionState = executionState

}

func TestDeployer_mspMongodStateRepresentation(t *testing.T) {

	var hostPort msp.HostPort
	var mspMongod msp.Mongod
	var err error

	db, err := createDB(t)
	assert.Nil(t, err)

	d := Deployer{
		DB: db,
	}

	var dbMongod model.Mongod
	var parentSlave model.Slave
	var desiredState model.MongodState
	assert.Nil(t, db.First(&dbMongod).Error)
	assert.Nil(t, db.Model(&dbMongod).Related(&parentSlave, "ParentSlave").Error)
	assert.Nil(t, db.Model(&dbMongod).Related(&desiredState, "DesiredState").Error)
	dbMongod.ParentSlave = &parentSlave
	dbMongod.DesiredState = desiredState

	hostPort, mspMongod, err = d.mspMongodStateRepresentation(d.DB, model.Mongod{ID: 1})
	assert.NotNil(t, err, "Should not be able to find hostPort for Mongod without ParentSlaveID")
	assert.Zero(t, hostPort)

	hostPort, mspMongod, err = d.mspMongodStateRepresentation(db, model.Mongod{
		ParentSlaveID: dbMongod.ParentSlaveID,
	})
	assert.NotNil(t, err, "Should not be able to find hostPort for Mongod without DesiredStateID")

	hostPort, mspMongod, err = d.mspMongodStateRepresentation(db, dbMongod)
	assert.Nil(t, err, "ParentSlaveID and DesiredStateID should suffice to build MSP MongodState representation")

	assert.EqualValues(t, msp.HostPort{dbMongod.ParentSlave.Hostname, uint16(dbMongod.ParentSlave.Port)}, hostPort)

	expectedMongodState, _ := mspMongodStateFromExecutionState(dbMongod.DesiredState.ExecutionState)

	assert.Equal(t, msp.Mongod{
		Port:           uint16(dbMongod.Port),
		ReplicaSetName: dbMongod.ReplSetName,

		// TODO: this is hardcoded knowlege about the contents of the test database.
		// Use something auto-generated instead.
		// Also: is this field actually relevant in an EstablishState call?
		ReplicaSetMembers: []msp.HostPort{msp.HostPort{"host1", 2000}},

		ShardingConfigServer: dbMongod.DesiredState.IsShardingConfigServer,
		State:                expectedMongodState,
	}, mspMongod)

}
