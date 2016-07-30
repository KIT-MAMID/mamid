package master

import (
	"github.com/KIT-MAMID/mamid/model"
	"github.com/KIT-MAMID/mamid/msp"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func createDB(t *testing.T) (db *gorm.DB, err error) {
	// Setup database
	db, err = model.InitializeTestDB()

	dbSlave := model.Slave{
		ID:                   1,
		Hostname:             "host1",
		Port:                 1,
		MongodPortRangeBegin: 2,
		MongodPortRangeEnd:   3,
		PersistentStorage:    true,
		Mongods:              []*model.Mongod{},
		ConfiguredState:      model.SlaveStateActive,
	}
	assert.NoError(t, db.Create(&dbSlave).Error)
	m1 := model.Mongod{
		Port:           2000,
		ReplSetName:    "repl1",
		ParentSlaveID:  1,
		DesiredStateID: 1,
	}
	des1 := model.MongodState{
		ID: 1,
		IsShardingConfigServer: false,
		ExecutionState:         model.MongodExecutionStateRunning,
	}
	assert.NoError(t, db.Create(&des1).Error)
	assert.NoError(t, db.Create(&m1).Error)

	return
}

type FakeMSPClient struct {
	msp.MSPClient
	Status []msp.Mongod
	Error  msp.Error
}

func (m FakeMSPClient) RequestStatus(Target msp.HostPort) ([]msp.Mongod, msp.Error) {
	return m.Status, m.Error
}

func TestMonitor_observeSlave(t *testing.T) {
	db, err := createDB(t)
	assert.NoError(t, err)

	mspClient := FakeMSPClient{
		Status: []msp.Mongod{
			msp.Mongod{
				Port:                    2000,
				ReplicaSetName:          "repl1",
				ReplicaSetMembers:       []msp.HostPort{},
				ShardingConfigServer:    false,
				StatusError:             nil,
				LastEstablishStateError: nil,
				State: msp.MongodStateRunning,
			},
		},
		Error: nil,
	}

	wg := new(sync.WaitGroup)
	bus := NewBus()
	readChannel := bus.GetNewReadChannel()
	monitor := Monitor{
		DB:              db,
		BusWriteChannel: bus.GetNewWriteChannel(),
		MSPClient:       mspClient,
	}

	wg.Add(1)
	go func() {
		bus.Run()
		wg.Done()
	}()

	//Observe Slave
	var slave model.Slave
	db.First(&slave, 1)

	monitor.observeSlave(slave)

	var mongod model.Mongod
	db.First(&mongod, 1)
	db.Model(&mongod).Related(&mongod.ObservedState, "ObservedState")
	assert.Equal(t, model.MongodExecutionStateRunning, mongod.ObservedState.ExecutionState)

	connStatusX := <-readChannel
	connStatus, ok := connStatusX.(model.ConnectionStatus)
	assert.True(t, ok)
	assert.False(t, connStatus.Unreachable)

	mismatchX := <-readChannel
	mismatch, ok := mismatchX.(model.MongodMatchStatus)
	assert.False(t, mismatch.Mismatch)

	//Slave cannot observe mongod
	monitor.MSPClient = FakeMSPClient{
		Status: []msp.Mongod{
			msp.Mongod{
				Port:           2000,
				ReplicaSetName: "repl1",
				StatusError: &msp.SlaveError{
					Identifier:  "foo",
					Description: "bar",
				},
			},
		},
		Error: nil,
	}

	db.First(&slave, 1)

	monitor.observeSlave(slave)

	db.First(&mongod, 1)

	//Mongod should have an observation error
	db.Model(&mongod).Related(&mongod.ObservationError, "ObservationError")
	assert.NotZero(t, mongod.ObservationErrorID)

	connStatusX = <-readChannel
	connStatus, ok = connStatusX.(model.ConnectionStatus)
	assert.True(t, ok)
	assert.False(t, connStatus.Unreachable)

	<-readChannel //mismatch

	//Slave becomes unreachable
	monitor.MSPClient = FakeMSPClient{
		Status: []msp.Mongod{},
		Error:  msp.CommunicationError{},
	}

	db.First(&slave, 1)

	monitor.observeSlave(slave)

	connStatusX = <-readChannel
	connStatus, ok = connStatusX.(model.ConnectionStatus)
	assert.True(t, ok)
	assert.True(t, connStatus.Unreachable)

	bus.Kill()
	wg.Wait()
}
