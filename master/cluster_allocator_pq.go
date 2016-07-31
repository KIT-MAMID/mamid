package master

import (
	. "github.com/KIT-MAMID/mamid/model"
	"github.com/jinzhu/gorm"
)

type pqSlavesByRiskGroup struct {
}

func (q *pqSlavesByRiskGroup) pushSlave(s Slave) {
	// assuming RiskGroupID is set
}

func (q *pqSlavesByRiskGroup) popSlaveinNonconflictingRiskGroup(r RiskGroup) {

}

func (c *ClusterAllocator) pqRiskGroups(tx *gorm.DB) *pqSlavesByRiskGroup {
	return &pqSlavesByRiskGroup{}
}

type pqReplicaSets struct {
}

func (q *pqReplicaSets) Push(r *ReplicaSet) {

}

func (q *pqReplicaSets) Pop(r *ReplicaSet) {

}

func (c *ClusterAllocator) pqReplicaSets(tx *gorm.DB) *pqReplicaSets {
	return &pqReplicaSets{}
}
