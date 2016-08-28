package master

import (
	"github.com/KIT-MAMID/mamid/model"
	"github.com/KIT-MAMID/mamid/msp"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func createDB(t *testing.T) (db *model.DB, err error) {
	// Setup database
	db, path, err := model.InitializeTestDB()
	t.Logf("creating test db: %s", path)

	tx := db.Begin()

	dbSlave := model.Slave{
		Hostname:             "host1",
		Port:                 1,
		MongodPortRangeBegin: 2,
		MongodPortRangeEnd:   3,
		PersistentStorage:    true,
		Mongods:              []*model.Mongod{},
		ConfiguredState:      model.SlaveStateActive,
	}
	assert.NoError(t, tx.Create(&dbSlave).Error)
	dbReplSet := model.ReplicaSet{
		Name: "foo",
	}
	assert.NoError(t, tx.Create(&dbReplSet).Error)
	m1 := model.Mongod{
		Port:          2000,
		ReplSetName:   "repl1",
		ParentSlaveID: dbSlave.ID,
		ReplicaSetID:  model.NullIntValue(dbReplSet.ID),
	}
	assert.NoError(t, tx.Create(&m1).Error)
	des1 := model.MongodState{
		ParentMongodID:         m1.ID,
		IsShardingConfigServer: false,
		ExecutionState:         model.MongodExecutionStateRunning,
	}
	assert.NoError(t, tx.Create(&des1).Error)
	assert.NoError(t, tx.Model(&m1).Update("DesiredStateID", des1.ID).Error)

	tx.Commit()
	return
}

type FakeMSPClient struct {
	msp.MSPClient
	Status []msp.Mongod
	Error  *msp.Error
}

func (m FakeMSPClient) RequestStatus(Target msp.HostPort) ([]msp.Mongod, *msp.Error) {
	return m.Status, m.Error
}

func TestMonitor_observeSlave(t *testing.T) {
	db, err := createDB(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	wg := new(sync.WaitGroup)
	bus := NewBus()
	readChannel := bus.GetNewReadChannel()
	monitor := Monitor{
		DB:              db,
		BusWriteChannel: bus.GetNewWriteChannel(),
	}

	wg.Add(1)
	go func() {
		bus.Run()
		wg.Done()
	}()

	//Observe Slave
	var slave model.Slave
	{
		tx := db.Begin()
		assert.NoError(t, tx.First(&slave).Error)
		tx.Rollback()
	}

	monitor.handleObservation([]msp.Mongod{
		msp.Mongod{
			Port:                    2000,
			ReplicaSetName:          "repl1",
			ReplicaSetMembers:       []msp.HostPort{},
			ShardingConfigServer:    false,
			StatusError:             nil,
			LastEstablishStateError: nil,
			State: msp.MongodStateRunning,
		},
	}, nil, slave)

	var mongod model.Mongod
	{
		tx := db.Begin()
		assert.NoError(t, tx.First(&mongod).Error)
		assert.Nil(t, tx.Model(&mongod).Related(&mongod.ObservedState, "ObservedState").Error, "after observation, the observed state should be != nil")
		tx.Rollback()
	}
	assert.Equal(t, model.MongodExecutionStateRunning, mongod.ObservedState.ExecutionState)

	connStatusX := <-readChannel
	connStatus, ok := connStatusX.(model.ConnectionStatus)
	assert.True(t, ok)
	assert.False(t, connStatus.Unreachable)

	mismatchX := <-readChannel
	mismatch, ok := mismatchX.(model.MongodMatchStatus)
	assert.False(t, mismatch.Mismatch)

	//-----------------
	//Slave cannot observe mongod
	//-----------------
	{
		tx := db.Begin()
		assert.NoError(t, tx.First(&slave).Error)
		tx.Rollback()
	}

	monitor.handleObservation([]msp.Mongod{
		msp.Mongod{
			Port:           2000,
			ReplicaSetName: "repl1",
			StatusError: &msp.Error{
				Identifier:  "foo",
				Description: "cannot observe mongod",
			},
		},
	}, nil, slave)

	{
		tx := db.Begin()
		assert.NoError(t, tx.First(&mongod).Error)

		//Mongod should have an observation error
		tx.Model(&mongod).Related(&mongod.ObservationError, "ObservationError")
		assert.EqualValues(t, "cannot observe mongod", mongod.ObservationError.Description)
		tx.Rollback()
	}
	assert.NotZero(t, mongod.ObservationErrorID)

	connStatusX = <-readChannel
	connStatus, ok = connStatusX.(model.ConnectionStatus)
	assert.True(t, ok)
	assert.False(t, connStatus.Unreachable)

	<-readChannel //mismatch

	//-----------------
	//Mongod gone
	//-----------------
	monitor.MSPClient = FakeMSPClient{
		Status: []msp.Mongod{},
		Error:  nil,
	}

	{
		tx := db.Begin()
		assert.NoError(t, tx.First(&slave).Error)
		tx.Rollback()
	}

	monitor.handleObservation([]msp.Mongod{}, nil, slave)

	{
		tx := db.Begin()
		assert.NoError(t, tx.First(&mongod).Error)

		//Mongod should not have observed state anymore
		assert.True(t, tx.Model(&mongod).Related(&mongod.ObservedState, "ObservedState").RecordNotFound())
		tx.Rollback()
	}

	connStatusX = <-readChannel
	connStatus, ok = connStatusX.(model.ConnectionStatus)
	assert.True(t, ok)
	assert.False(t, connStatus.Unreachable)

	<-readChannel //mismatch

	//-----------------
	//Slave becomes unreachable
	//-----------------
	monitor.MSPClient = FakeMSPClient{
		Status: []msp.Mongod{},
		Error:  &msp.Error{Identifier: msp.CommunicationError},
	}

	{
		tx := db.Begin()
		assert.NoError(t, tx.First(&slave).Error)
		tx.Rollback()
	}

	monitor.handleObservation([]msp.Mongod{}, &msp.Error{Identifier: msp.CommunicationError}, slave)

	connStatusX = <-readChannel
	connStatus, ok = connStatusX.(model.ConnectionStatus)
	assert.True(t, ok)
	assert.True(t, connStatus.Unreachable)

	bus.Kill()
	wg.Wait()
}

func TestMonitor_compareStates(t *testing.T) {

	db, err := createDB(t)
	assert.NoError(t, err)
	defer db.CloseAndDrop()

	monitor := Monitor{
		DB: db,
	}

	// Test without observed state
	tx := db.Begin()

	var dbMongod model.Mongod
	assert.NoError(t, tx.First(&dbMongod).Error)

	msg, err := monitor.compareStates(tx, dbMongod)
	assert.NoError(t, err)
	assert.EqualValues(t, true, msg.Mismatch, "Mongods without ObservedState should always result in mismatch")

	tx.Rollback()
	tx = db.Begin()

	// Test with equal observed state
	// => duplicate DesiredState
	assert.NoError(t, tx.Model(&dbMongod).Related(&dbMongod.ObservedState, "DesiredState").Error)
	dbMongod.ObservedState.ID = 0
	assert.NoError(t, tx.Create(&dbMongod.ObservedState).Error)
	assert.NoError(t, tx.Model(&dbMongod).Update("ObservedStateID", dbMongod.ObservedState.ID).Error)

	msg, err = monitor.compareStates(tx, dbMongod)
	assert.NoError(t, err)
	assert.EqualValues(t, false, msg.Mismatch, "Mongods with equal Observed & Desired states should not result in a mismatch")

	// Save this state, we check single unequal attributes from here on
	tx.Commit()
	tx = db.Begin()

	// Test with unequal execution state
	assert.NoError(t, tx.Model(&dbMongod.ObservedState).Update("ExecutionState", model.MongodExecutionStateNotRunning).Error)
	msg, err = monitor.compareStates(tx, dbMongod)
	assert.NoError(t, err)
	assert.EqualValues(t, true, msg.Mismatch, "unequal ExecutionState should result in a mismatch")

	tx.Rollback()
	tx = db.Begin()

	// Test with unequal IsShardingConfigServer field
	assert.NoError(t, tx.Model(&dbMongod.ObservedState).Update("IsShardingConfigServer", !dbMongod.DesiredState.IsShardingConfigServer).Error)
	msg, err = monitor.compareStates(tx, dbMongod)
	assert.NoError(t, err)
	assert.EqualValues(t, true, msg.Mismatch, "unequal IsShardingConfigServer should result in a mismatch")

	tx.Rollback()
	tx = db.Begin()

	var slave model.Slave
	assert.NoError(t, tx.First(&slave).Error)

	desiredMember1 := model.ReplicaSetMember{
		MongodStateID: dbMongod.DesiredStateID,
		Hostname:      slave.Hostname,
		Port:          dbMongod.Port,
	}
	assert.NoError(t, tx.Create(&desiredMember1).Error)
	assert.True(t, dbMongod.ObservedStateID.Valid)
	observedMember1 := model.ReplicaSetMember{
		MongodStateID: dbMongod.ObservedStateID.Int64,
		Hostname:      desiredMember1.Hostname,
		Port:          desiredMember1.Port,
	}
	assert.NoError(t, tx.Create(&observedMember1).Error)

	// Test with equal ReplicaSetMembers
	msg, err = monitor.compareStates(tx, dbMongod)
	assert.NoError(t, err)
	assert.EqualValues(t, false, msg.Mismatch, "equal ReplicaSetMembers should not result in a mismatch")

	// Keep them equal
	tx.Commit()
	tx = db.Begin()

	// Test with same number of members but different values
	assert.NoError(t, tx.Model(&observedMember1).Update("Port", observedMember1.Port+1).Error)
	assert.NoError(t, tx.Model(&observedMember1).Update("Hostname", "someunknownhost").Error)

	var desiredMemberRefetched model.ReplicaSetMember
	assert.NoError(t, tx.Model(&dbMongod).Related(&desiredMemberRefetched, "DesiredState").Error)
	assert.EqualValues(t, slave.Hostname, desiredMemberRefetched.Hostname)
	assert.EqualValues(t, "someunknownhost", observedMember1.Hostname)

	msg, err = monitor.compareStates(tx, dbMongod)
	assert.NoError(t, err)
	assert.EqualValues(t, true, msg.Mismatch, "unequal ReplicaSetMembers should result in a mismatch")

	tx.Rollback()
	tx = db.Begin()

	// Test with different number of members but same values
	observedMember2 := observedMember1
	observedMember2.Hostname = "anotherhost"
	assert.False(t, observedMember2.Hostname == observedMember1.Hostname, "error in test logic")
	observedMember2.ID = 0
	assert.NoError(t, tx.Create(&observedMember2).Error)

	msg, err = monitor.compareStates(tx, dbMongod)
	assert.NoError(t, err)
	assert.EqualValues(t, true, msg.Mismatch, "different sets of ReplicaSetMembers should result in a mismatch")

	tx.Rollback()

}

func TestMonitor_ReplicaSetMembersEquivalent(t *testing.T) {
	assert.True(t, ReplicaSetMembersEquivalent(model.ReplicaSetMember{Hostname: "host1", Port: 100}, model.ReplicaSetMember{Hostname: "host1", Port: 100}))
	assert.False(t, ReplicaSetMembersEquivalent(model.ReplicaSetMember{Hostname: "host1", Port: 100}, model.ReplicaSetMember{Hostname: "host1", Port: 200}))
	assert.False(t, ReplicaSetMembersEquivalent(model.ReplicaSetMember{Hostname: "host1", Port: 100}, model.ReplicaSetMember{Hostname: "host2", Port: 100}))
	assert.False(t, ReplicaSetMembersEquivalent(model.ReplicaSetMember{Hostname: "host1", Port: 100}, model.ReplicaSetMember{Hostname: "host2", Port: 200}))
}
