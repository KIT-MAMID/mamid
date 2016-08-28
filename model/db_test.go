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
		ConfiguredState:      SlaveStateActive,
	}
}

func fixtureEmptyMongod() *Mongod {
	return &Mongod{
		Port:        8080,
		ReplSetName: "repl1",
	}
}

func fixtureEmptyMongodState() MongodState {
	return MongodState{}
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

func fixtureEmptyProblem() *Problem {
	return &Problem{
		Description: "Test",
	}
}

////////////////////////////////////////////////////////////////////////////////

func TestCanInitializeDB(t *testing.T) {
	db, _, err := InitializeTestDB()
	defer db.CloseAndDrop()
	assert.NoError(t, err)
}

/*
 This elaborate test demonstrates how resolving an association works in gorm.
 Check the assertions to learn about the behavior of gorm.
*/
func TestRelationshipMongodParentSlave(t *testing.T) {

	db, _, _ := InitializeTestDB()
	defer db.CloseAndDrop()
	tx := db.Begin()
	defer tx.Rollback()

	s := fixtureEmptySlave()

	tx.Create(s)

	m := fixtureEmptyMongod()
	m.ParentSlave = s

	tx.Create(m)

	assert.Equal(t, m.ParentSlaveID, s.ID)

	assert.Equal(t, s.Mongods, []*Mongod{})

	var sdb Slave

	// Check what happens when just SELECTing the slave
	err := tx.First(&sdb).Error

	assert.NoError(t, err)
	assert.Nil(t, sdb.Mongods)

	// Now resolve the slave->mongod 1:n association
	err = tx.Model(&sdb).Related(&sdb.Mongods, "Mongods").Error

	assert.NoError(t, err)
	assert.Equal(t, len(sdb.Mongods), 1)
	assert.Equal(t, sdb.Mongods[0].ReplSetName, m.ReplSetName)
	assert.Zero(t, sdb.Mongods[0].ParentSlave)
	assert.Equal(t, sdb.Mongods[0].ParentSlaveID, s.ID)

	// Now resolve the mongod->(parent)slave relation
	parentSlave := &Slave{}
	err = tx.Model(&sdb.Mongods[0]).Related(parentSlave, "ParentSlave").Error
	assert.NoError(t, err)
	assert.NotZero(t, parentSlave)
	assert.Equal(t, s.ID, parentSlave.ID)

}

// Test RiskGroup Slave relationship
func TestRiskGroupSlaveRelationship(t *testing.T) {

	db, _, _ := InitializeTestDB()
	defer db.CloseAndDrop()
	tx := db.Begin()
	defer tx.Rollback()

	s := fixtureEmptySlave()
	r := fixtureEmptyRiskGroup()
	r.Slaves = []*Slave{s}

	err := tx.Create(&r).Error
	assert.NoError(t, err)

	var rdb RiskGroup

	err = tx.First(&rdb).Error

	assert.NoError(t, err)
	assert.Zero(t, rdb.Slaves)

	err = tx.Model(&rdb).Related(&rdb.Slaves, "Slaves").Error
	assert.NoError(t, err)
	assert.NotZero(t, rdb.Slaves)
	assert.Equal(t, len(rdb.Slaves), 1)
	assert.Equal(t, rdb.Slaves[0].ID, s.ID)

}

// Test ReplicaSet - Mongod Relationship
func TestReplicaSetMongodRelationship(t *testing.T) {

	db, _, _ := InitializeTestDB()
	defer db.CloseAndDrop()
	tx := db.Begin()
	defer tx.Rollback()

	r := fixtureEmptyReplicaSet()
	m := fixtureEmptyMongod()
	r.Mongods = []*Mongod{m}

	err := tx.Create(&r).Error
	assert.NoError(t, err)

	var rdb ReplicaSet

	err = tx.First(&rdb).Error

	assert.NoError(t, err)
	assert.Zero(t, rdb.Mongods)

	err = tx.Model(&rdb).Related(&rdb.Mongods, "Mongods").Error
	assert.NoError(t, err)
	assert.NotZero(t, rdb.Mongods)
	assert.Equal(t, len(rdb.Mongods), 1)
	assert.Equal(t, rdb.Mongods[0].ID, m.ID)

}

// Test Mongod - MongodState relationship
func TestMongodMongodStateRelationship(t *testing.T) {

	db, _, _ := InitializeTestDB()
	defer db.CloseAndDrop()
	tx := db.Begin()
	defer tx.Rollback()

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

	assert.NoError(t, tx.Create(m).Error)
	o.ParentMongodID = m.ID
	d.ParentMongodID = m.ID
	assert.NoError(t, tx.Create(&o).Error)
	assert.NoError(t, tx.Create(&d).Error)
	assert.NoError(t, tx.Model(&m).Update("DesiredStateID", d.ID).Error)
	assert.NoError(t, tx.Model(&m).Update("ObservedStateID", o.ID).Error)

	var mdb Mongod

	// Observed
	assert.NoError(t, tx.First(&mdb).Error)
	assert.Zero(t, mdb.ObservedState)

	assert.NoError(t, tx.Model(&mdb).Related(&mdb.ObservedState, "ObservedState").Error)
	assert.NotZero(t, mdb.ObservedState)
	assert.Equal(t, mdb.ObservedState.ExecutionState, MongodExecutionStateNotRunning)

	assert.NoError(t, tx.Model(&mdb).Related(&mdb.DesiredState, "DesiredState").Error)
	assert.NotZero(t, mdb.DesiredState)
	assert.Equal(t, mdb.DesiredState.ExecutionState, MongodExecutionStateRunning)

}

// Test MongodState - ReplicaSetMember relationship
func TestMongodStateReplicaSetMembersRelationship(t *testing.T) {
	db, _, _ := InitializeTestDB()
	defer db.CloseAndDrop()
	tx := db.Begin()
	defer tx.Rollback()

	m := ReplicaSetMember{Hostname: "h1"}

	s := MongodState{ReplicaSetMembers: []ReplicaSetMember{m}}

	assert.NoError(t, tx.Create(&m).Error)
	s.ParentMongodID = m.ID
	assert.NoError(t, tx.Create(&s).Error)
	assert.NoError(t, tx.Model(&m).Update("DesiredStateID", s.ID).Error)

	var sdb MongodState

	assert.NoError(t, tx.First(&sdb).Error)
	assert.Zero(t, sdb.ReplicaSetMembers)

	assert.NoError(t, tx.Model(&sdb).Related(&sdb.ReplicaSetMembers, "ReplicaSetMembers").Error)
	assert.NotZero(t, sdb.ReplicaSetMembers)
	assert.Equal(t, len(sdb.ReplicaSetMembers), 1)
	assert.Equal(t, sdb.ReplicaSetMembers[0].Hostname, m.Hostname)

}

func TestDeleteBehavior(t *testing.T) {

	db, _, _ := InitializeTestDB()
	defer db.CloseAndDrop()
	tx := db.Begin()
	defer tx.Rollback()

	m := fixtureEmptyMongod()
	m.ID = 1000

	// Create it
	tx.Create(&m)

	var mdb Mongod

	// Read it once
	d := tx.First(&mdb)

	assert.NoError(t, d.Error)
	assert.Equal(t, mdb.ID, m.ID)

	// Destroy it once, by ID
	d = tx.Delete(&Mongod{ID: 1000})

	assert.NoError(t, d.Error)
	assert.EqualValues(t, 1, d.RowsAffected)

	// Destroy it a second time.
	// No Error will occur, have to check RowsAffected if we deleted something
	d = tx.Delete(&Mongod{ID: 1000})

	assert.NoError(t, d.Error)
	assert.EqualValues(t, 0, d.RowsAffected)

}

func TestGormFirstBehavior(t *testing.T) {
	db, _, _ := InitializeTestDB()
	defer db.CloseAndDrop()
	tx := db.Begin()
	defer tx.Rollback()

	var m Mongod
	assert.Error(t, tx.First(&m).Error)
}

func TestGormFindBehavior(t *testing.T) {
	db, _, _ := InitializeTestDB()
	defer db.CloseAndDrop()
	tx := db.Begin()
	defer tx.Rollback()

	var ms []Mongod
	d := tx.Find(&ms)

	assert.NoError(t, d.Error)
	assert.EqualValues(t, 0, d.RowsAffected) // RowsAffected does NOT indicate "nothing found"!!!!
	assert.Equal(t, 0, len(ms))              // Use this instead

}

func TestCascade(t *testing.T) {
	db, _, _ := InitializeTestDB()
	defer db.CloseAndDrop()

	tx0 := db.Begin()
	assert.NoError(t, tx0.Exec("CREATE TABLE foo(id int primary key);").Error)
	assert.NoError(t, tx0.Exec("CREATE TABLE bar3(foreignKey int null references foo(id) on delete cascade deferrable initially deferred);").Error)
	assert.NoError(t, tx0.Commit().Error)

	tx1 := db.Begin()
	assert.NoError(t, tx1.Exec("INSERT INTO foo VALUES(1); INSERT INTO bar3 VALUES(1);").Error)
	assert.NoError(t, tx1.Commit().Error)

	tx3 := db.Begin()
	var count int
	tx3.Raw("SELECT count(*) FROM bar3;").Row().Scan(&count)
	assert.EqualValues(t, 1, count)
	tx3.Commit()

	tx2 := db.Begin()
	tx2.Exec("DELETE FROM foo WHERE id = 1;")
	assert.NoError(t, tx2.Commit().Error)

	tx3 = db.Begin()
	tx3.Raw("SELECT count(*) FROM bar3;").Row().Scan(&count)
	assert.EqualValues(t, 0, count)
	tx3.Commit()
}

func TestCascadeSlaves(t *testing.T) {
	db, _, _ := InitializeTestDB()
	defer db.CloseAndDrop()
	tx := db.Begin()

	r := fixtureEmptyRiskGroup()
	assert.NoError(t, tx.Create(r).Error)

	rs := fixtureEmptyReplicaSet()
	assert.NoError(t, tx.Create(rs).Error)

	s := fixtureEmptySlave()
	s.RiskGroupID = NullIntValue(r.ID)
	assert.NoError(t, tx.Create(s).Error)

	ds := fixtureEmptyMongodState()
	assert.NoError(t, tx.Create(&ds).Error)

	m := fixtureEmptyMongod()
	m.ParentSlaveID = s.ID
	m.ReplicaSetID = NullIntValue(rs.ID)
	m.DesiredStateID = ds.ID
	assert.NoError(t, tx.Create(m).Error)
	assert.NoError(t, tx.Model(&ds).Update("ParentMongodID", m.ID).Error)

	p := fixtureEmptyProblem()
	p.SlaveID = NullIntValue(s.ID)
	assert.NoError(t, tx.Create(p).Error)

	assert.NoError(t, tx.Commit().Error)

	tx1 := db.Begin()
	tx1.Delete(&Mongod{}, m.ID)
	assert.NoError(t, tx1.Commit().Error)

	tx2 := db.Begin()
	tx2.Delete(&Slave{}, s.ID)
	assert.NoError(t, tx2.Commit().Error)

	tx3 := db.Begin()
	assert.True(t, tx3.First(&Problem{}, p.ID).RecordNotFound())
	tx3.Rollback()
}

// Test case demonstrating how to do overwrites
func TestObservationErrorOverwriteBehavior(t *testing.T) {

	// Assume a situation where Slave already has an ObservationError
	db, _, _ := InitializeTestDB()
	defer db.CloseAndDrop()
	tx := db.Begin()

	o1 := MSPError{
		Identifier: "id1",
	}

	assert.NoError(t, tx.Create(&o1).Error)

	s := Slave{
		Hostname:             "s1",
		Port:                 1,
		MongodPortRangeBegin: 1,
		MongodPortRangeEnd:   2,
		ObservationErrorID:   NullIntValue(o1.ID),
	}

	assert.NoError(t, tx.Create(&s).Error)

	var countBeforeUpdate int64
	assert.NoError(t, tx.Model(&MSPError{}).Count(&countBeforeUpdate).Error)
	assert.EqualValues(t, 1, countBeforeUpdate)

	// Now we attempt to observe Slave again and want to update the value pointed to by slave
	// We could create, update, delete
	// Or we could use the following hack to just UPDATE all values of the existing observation,
	//    saving us an annoying DELETE
	o2 := MSPError{
		Identifier: "id2",
	}
	o2.ID = o1.ID
	tx.Save(&o2)
	// That was it

	var countAfterUpdate int64
	assert.NoError(t, tx.Model(&MSPError{}).Count(&countAfterUpdate).Error)

	assert.EqualValues(t, 1, countAfterUpdate, "updates should remove the row previously referenced in the overwritten column")

	tx.Commit()

}
