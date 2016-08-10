package model

import (
	"fmt"
	"github.com/mattn/go-sqlite3"
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
		ConfiguredState:      SlaveStateActive,
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

func TestDeleteBehavior(t *testing.T) {

	db, _ := InitializeInMemoryDB("")

	m := fixtureEmptyMongod()
	m.ID = 1000

	// Create it
	db.Create(&m)

	var mdb Mongod

	// Read it once
	d := db.First(&mdb)

	assert.NoError(t, d.Error)
	assert.Equal(t, mdb.ID, m.ID)

	// Destroy it once, by ID
	d = db.Delete(&Mongod{ID: 1000})

	assert.NoError(t, d.Error)
	assert.EqualValues(t, 1, d.RowsAffected)

	// Destroy it a second time.
	// No Error will occur, have to check RowsAffected if we deleted something
	d = db.Delete(&Mongod{ID: 1000})

	assert.NoError(t, d.Error)
	assert.EqualValues(t, 0, d.RowsAffected)

}

func TestGormFirstBehavior(t *testing.T) {
	db, _ := InitializeInMemoryDB("")
	var m Mongod
	assert.Error(t, db.First(&m).Error)
}

func TestGormFindBehavior(t *testing.T) {
	db, _ := InitializeInMemoryDB("")

	var ms []Mongod
	d := db.Find(&ms)

	assert.NoError(t, d.Error)
	assert.EqualValues(t, 0, d.RowsAffected) // RowsAffected does NOT indicate "nothing found"!!!!
	assert.Equal(t, 0, len(ms))              // Use this instead

}

func TestGormTransactions(t *testing.T) {
	db, _ := InitializeTestDB()

	//Create a slave
	tx0 := db.Begin()
	m := fixtureEmptySlave()
	m.ID = 1
	m.Hostname = "baz"
	m.Port = 5
	assert.NoError(t, tx0.Create(&m).Error)
	assert.NoError(t, tx0.Commit().Error)

	fmt.Println("Insert slaves from tx0 done and committed")

	//Modify same slave in two transactions
	tx1 := db.Begin()
	fmt.Println("begin tx1")
	assert.NoError(t, tx1.First(&Slave{}, 1).Update("hostname", "foo").Error)
	assert.NoError(t, tx1.First(&Slave{}, 1).Update("port", 15).Error)
	fmt.Println("Update slave 1 from tx1 done")

	tx2 := db.Begin()
	fmt.Println("begin tx2")

	//Should be able to read slave 1 and see old state
	var slaveReadX Slave
	assert.NoError(t, tx2.First(&slaveReadX, 1).Error)
	assert.Equal(t, "baz", slaveReadX.Hostname)
	fmt.Println("Read slave 1 from tx2 done")

	err := tx2.First(&Slave{}, 1).Update("hostname", "bar").Error
	assert.Error(t, err)
	driverErr, ok := err.(sqlite3.Error)
	assert.True(t, ok)
	assert.Equal(t, sqlite3.ErrBusy, driverErr.Code) //https://www.sqlite.org/rescode.html#busy
	fmt.Println("Update slave 1 from tx2 done")

	//Commit Tx2
	assert.NoError(t, tx2.Commit().Error)
	fmt.Println("tx2 done")
	//Commit Tx1
	assert.NoError(t, tx1.Commit().Error)
	fmt.Println("tx1 done")

	var slave Slave
	db.First(&slave, 1)
	assert.Equal(t, "foo", slave.Hostname)
}
