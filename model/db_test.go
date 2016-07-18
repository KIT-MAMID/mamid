package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func fixtureEmptySlave() *Slave {
	return &Slave{
		Hostname:             "host1",
		Port:                 1,
		MongodPortRangeBegin: 2,
		MongodPortRangeEnd:   3,
		PersistentStorage:    true,
		Mongods:              []*Mongod{},
		ConfiguedState:       SlaveStateActive,
	}
}

func fixtureEmptyMongod() *Mongod {
	return &Mongod{
		Port:        8080,
		ReplSetName: "repl1",
	}
}

func fixtureEmptyRiskGroup() *RiskGroup {
	return &RiskGroup{
		Name:   "rg1",
		Slaves: []*Slave{},
	}
}

func fixtureEmptyReplicaSet() *ReplicaSet {
	return &ReplicaSet{
		Name: "repl1",
		PersistentMemberCount:           1,
		VolatileMemberCount:             2,
		ConfigureAsShardingConfigServer: false,
	}
}

////////////////////////////////////////////////////////////////////////////////

func TestCanInitializeDB(t *testing.T) {
	_, err := InitializeInMemoryDB("")
	assert.NoError(t, err)
}

/*
 This elaborate test demonstrates how resolving an association works in gorm.
 Check the assertions to learn about the behavior of gorm.
*/
func TestRelationshipMongodParentSlave(t *testing.T) {

	db, _ := InitializeInMemoryDB("")

	s := fixtureEmptySlave()

	db.Create(s)

	m := fixtureEmptyMongod()
	m.ParentSlave = s

	db.Create(m)

	assert.Equal(t, m.ParentSlaveID, s.ID)

	assert.Equal(t, s.Mongods, []*Mongod{})

	var sdb Slave

	// Check what happens when just SELECTing the slave
	err := db.First(&sdb).Error

	assert.NoError(t, err)
	assert.Nil(t, sdb.Mongods)

	// Now resolve the slave->mongod 1:n association
	err = db.Model(&sdb).Related(&sdb.Mongods, "Mongods").Error

	assert.NoError(t, err)
	assert.Equal(t, len(sdb.Mongods), 1)
	assert.Equal(t, sdb.Mongods[0].ReplSetName, m.ReplSetName)
	assert.Zero(t, sdb.Mongods[0].ParentSlave)
	assert.Equal(t, sdb.Mongods[0].ParentSlaveID, s.ID)

	// Now resolve the mongod->(parent)slave relation
	parentSlave := &Slave{}
	err = db.Model(&sdb.Mongods[0]).Related(parentSlave, "ParentSlave").Error
	assert.NoError(t, err)
	assert.NotZero(t, parentSlave)
	assert.Equal(t, s.ID, parentSlave.ID)

}

// Test RiskGroup Slave relationship
func TestRiskGroupSlaveRelationship(t *testing.T) {

	db, _ := InitializeInMemoryDB("")

	s := fixtureEmptySlave()
	r := fixtureEmptyRiskGroup()
	r.Slaves = []*Slave{s}

	err := db.Create(&r).Error
	assert.NoError(t, err)

	var rdb RiskGroup

	err = db.First(&rdb).Error

	assert.NoError(t, err)
	assert.Zero(t, rdb.Slaves)

	err = db.Model(&rdb).Related(&rdb.Slaves, "Slaves").Error
	assert.NoError(t, err)
	assert.NotZero(t, rdb.Slaves)
	assert.Equal(t, len(rdb.Slaves), 1)
	assert.Equal(t, rdb.Slaves[0].ID, s.ID)

}

// Test ReplicaSet - Mongod Relationship
func TestReplicaSetMongodRelationship(t *testing.T) {

	db, _ := InitializeInMemoryDB("")

	r := fixtureEmptyReplicaSet()
	m := fixtureEmptyMongod()
	r.Mongods = []*Mongod{m}

	err := db.Create(&r).Error
	assert.NoError(t, err)

	var rdb ReplicaSet

	err = db.First(&rdb).Error

	assert.NoError(t, err)
	assert.Zero(t, rdb.Mongods)

	err = db.Model(&rdb).Related(&rdb.Mongods, "Mongods").Error
	assert.NoError(t, err)
	assert.NotZero(t, rdb.Mongods)
	assert.Equal(t, len(rdb.Mongods), 1)
	assert.Equal(t, rdb.Mongods[0].ID, m.ID)

}

// Test Mongod - MongodState relationship
func TestMongodMongodStateRelationship(t *testing.T) {

	db, _ := InitializeInMemoryDB("")

	m := fixtureEmptyMongod()

	o := MongodState{
		IsShardingConfigServer: false,
		ExecutionState:         MongodExecutionStateNotRunning,
		ReplicaSetMembers:      []ReplicaSetMember{},
	}

	d := MongodState{
		IsShardingConfigServer: false,
		ExecutionState:         MongodExecutionStateRunning,
		ReplicaSetMembers:      []ReplicaSetMember{},
	}

	m.ObservedState = o
	m.DesiredState = d

	assert.NoError(t, db.Create(m).Error)

	var mdb Mongod

	// Observed
	assert.NoError(t, db.First(&mdb).Error)
	assert.Zero(t, mdb.ObservedState)

	assert.NoError(t, db.Model(&mdb).Related(&mdb.ObservedState, "ObservedState").Error)
	assert.NotZero(t, mdb.ObservedState)
	assert.Equal(t, mdb.ObservedState.ExecutionState, MongodExecutionStateNotRunning)

	assert.NoError(t, db.Model(&mdb).Related(&mdb.DesiredState, "DesiredState").Error)
	assert.NotZero(t, mdb.DesiredState)
	assert.Equal(t, mdb.DesiredState.ExecutionState, MongodExecutionStateRunning)

}

// Test MongodState - ReplicaSetMember relationship
func TestMongodStateReplicaSetMembersRelationship(t *testing.T) {
	db, _ := InitializeInMemoryDB("")

	m := ReplicaSetMember{Hostname: "h1"}

	s := MongodState{ReplicaSetMembers: []ReplicaSetMember{m}}

	assert.NoError(t, db.Create(&s).Error)

	var sdb MongodState

	assert.NoError(t, db.First(&sdb).Error)
	assert.Zero(t, sdb.ReplicaSetMembers)

	assert.NoError(t, db.Model(&sdb).Related(&sdb.ReplicaSetMembers, "ReplicaSetMembers").Error)
	assert.NotZero(t, sdb.ReplicaSetMembers)
	assert.Equal(t, len(sdb.ReplicaSetMembers), 1)
	assert.Equal(t, sdb.ReplicaSetMembers[0].Hostname, m.Hostname)

}
