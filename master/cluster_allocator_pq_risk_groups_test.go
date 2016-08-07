package master

import (
	"github.com/KIT-MAMID/mamid/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

/*
Should partition the
  set of ALL slaves of type either persistent|volatile
  with >= 1 free Mongod Port
  into a map[RiskGroup]PriorityQueue
*/
func TestClusterAllocator_pqRiskGroups(t *testing.T) {
	db, err := model.InitializeTestDBWithSQL("cluster_allocator_pq_fixture.sql")
	assert.NoError(t, err)

	c := ClusterAllocator{}

	//assert.NoError(t, db.Find(&r, 1).Error)
	//assert.NoError(t, db.Model(&r).Related(&r.Mongods, "Mongods").Error)
	//for _, m := range r.Mongods {
	//	var parentSlave model.Slave
	//	assert.NoError(t, db.Model(&m).Related(&parentSlave, "ParentSlave").Error)
	//	m.ParentSlave = &parentSlave
	//	for _,
	//}

	replicaSets := c.replicaSets(db)
	var r model.ReplicaSet
	for _, repl := range replicaSets {
		if repl.ID == 1 {
			r = *repl
		}
	}
	assert.EqualValues(t, 1, r.ID) // Check if found

	tx := db.Begin()
	pqRiskGroups := c.pqRiskGroups(tx, &r, Volatile)
	slave1 := pqRiskGroups.PopSlaveInNonconflictingRiskGroup()
	assert.EqualValues(t, 3, slave1.ID)
	slave2 := pqRiskGroups.PopSlaveInNonconflictingRiskGroup()
	assert.Nil(t, slave2)
	slave2 = pqRiskGroups.PopSlaveInNonconflictingRiskGroup()
	assert.Nil(t, slave2)
	slave2 = pqRiskGroups.PopSlaveInNonconflictingRiskGroup()
	assert.Nil(t, slave2)

	//Check no panic occurs when emptying the pq
	//for slaveN := pqRiskGroups.PopSlaveInNonconflictingRiskGroup(); slaveN != nil; {
	//}

	tx.Commit()
}
