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

	keyfile := model.MongodKeyfile{
		Content: "keyfile",
	}
	assert.NoError(t, tx.Create(&keyfile).Error)

	rootCredential := model.MongodbCredential{
		Username: "user",
		Password: "pass",
	}
	assert.NoError(t, tx.Table("mongodb_root_credentials").Create(&rootCredential).Error)

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
		Name:         "foo",
		ShardingRole: model.ShardingRoleNone,
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
		ParentMongodID: m1.ID,
		ShardingRole:   model.ShardingRoleNone,
		ExecutionState: model.MongodExecutionStateRunning,
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
			Port: 2000,
			ReplicaSetConfig: msp.ReplicaSetConfig{
				ReplicaSetName: "repl1",
				ReplicaSetMembers: []msp.ReplicaSetMember{
					msp.ReplicaSetMember{
						HostPort: msp.HostPort{
							Hostname: slave.Hostname,
							Port:     2000,
						},
						Priority: ReplicaSetMemberPriorityLow,
					},
				},
				ShardingRole: msp.ShardingRoleNone,
			},
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
			Port: 2000,
			ReplicaSetConfig: msp.ReplicaSetConfig{
				ReplicaSetName: "repl1",
			},
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

	mspMongod := msp.Mongod{}

	msg, err := monitor.compareStates(tx, dbMongod, mspMongod)
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

	//msp result for replica set members as they are not stored in the database
	mspMongod = msp.Mongod{
		ReplicaSetConfig: msp.ReplicaSetConfig{
			ReplicaSetMembers: []msp.ReplicaSetMember{
				msp.ReplicaSetMember{
					HostPort: msp.HostPort{
						Hostname: "host1",
						Port:     2000,
					},
					Priority: ReplicaSetMemberPriorityLow,
				},
			},
		},
	}

	msg, err = monitor.compareStates(tx, dbMongod, mspMongod)
	assert.NoError(t, err)
	assert.EqualValues(t, false, msg.Mismatch, "Mongods with equal Observed & Desired states should not result in a mismatch")

	// Save this state, we check single unequal attributes from here on
	tx.Commit()
	tx = db.Begin()

	// Test with unequal execution state
	assert.NoError(t, tx.Model(&dbMongod.ObservedState).Update("ExecutionState", model.MongodExecutionStateNotRunning).Error)
	msg, err = monitor.compareStates(tx, dbMongod, mspMongod)
	assert.NoError(t, err)
	assert.EqualValues(t, true, msg.Mismatch, "unequal ExecutionState should result in a mismatch")

	tx.Rollback()
	tx = db.Begin()

	// Test with unequal ShardingRole field
	assert.NoError(t, tx.Model(&dbMongod.ObservedState).Update("ShardingRole", model.ShardingRoleShardServer).Error)
	msg, err = monitor.compareStates(tx, dbMongod, mspMongod)
	assert.NoError(t, err)
	assert.EqualValues(t, true, msg.Mismatch, "unequal IsShardingConfigServer should result in a mismatch")

	tx.Rollback()
	tx = db.Begin()

	// Test with same number of members but different values
	mspMongodSameNumDiffValMembers := mspMongod
	mspMongodSameNumDiffValMembers.ReplicaSetConfig.ReplicaSetMembers[0].Priority = 12

	msg, err = monitor.compareStates(tx, dbMongod, mspMongod)
	assert.NoError(t, err)
	assert.EqualValues(t, true, msg.Mismatch, "unequal ReplicaSetMembers should result in a mismatch")

	tx.Rollback()
	tx = db.Begin()

	// Test with different number of members but same values
	mspMongodDiffNumSameValMembers := mspMongod
	mspMongodDiffNumSameValMembers.ReplicaSetConfig.ReplicaSetMembers =
		append(mspMongodDiffNumSameValMembers.ReplicaSetConfig.ReplicaSetMembers,
			mspMongodDiffNumSameValMembers.ReplicaSetConfig.ReplicaSetMembers[0])

	msg, err = monitor.compareStates(tx, dbMongod, mspMongodDiffNumSameValMembers)
	assert.NoError(t, err)
	assert.EqualValues(t, true, msg.Mismatch, "different sets of ReplicaSetMembers should result in a mismatch")

	tx.Rollback()

}

func TestMonitor_ReplicaSetMembersEquivalent(t *testing.T) {
	assert.True(t, ReplicaSetMembersEquivalent(msp.ReplicaSetMember{HostPort: msp.HostPort{Hostname: "host1", Port: 100}, Priority: 1}, msp.ReplicaSetMember{HostPort: msp.HostPort{Hostname: "host1", Port: 100}, Priority: 1}))
	assert.False(t, ReplicaSetMembersEquivalent(msp.ReplicaSetMember{HostPort: msp.HostPort{Hostname: "host1", Port: 100}, Priority: 2}, msp.ReplicaSetMember{HostPort: msp.HostPort{Hostname: "host1", Port: 100}, Priority: 1}))
	assert.False(t, ReplicaSetMembersEquivalent(msp.ReplicaSetMember{HostPort: msp.HostPort{Hostname: "host1", Port: 100}, Priority: 1}, msp.ReplicaSetMember{HostPort: msp.HostPort{Hostname: "host1", Port: 200}, Priority: 1}))
	assert.False(t, ReplicaSetMembersEquivalent(msp.ReplicaSetMember{HostPort: msp.HostPort{Hostname: "host1", Port: 100}, Priority: 1}, msp.ReplicaSetMember{HostPort: msp.HostPort{Hostname: "host2", Port: 100}, Priority: 1}))
	assert.False(t, ReplicaSetMembersEquivalent(msp.ReplicaSetMember{HostPort: msp.HostPort{Hostname: "host1", Port: 100}, Priority: 1}, msp.ReplicaSetMember{HostPort: msp.HostPort{Hostname: "host2", Port: 200}, Priority: 1}))
}
