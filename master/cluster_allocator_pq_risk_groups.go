package master

import (
	. "github.com/KIT-MAMID/mamid/model"
	"github.com/jinzhu/gorm"
)

type pqSlavesByRiskGroup struct {
}

func (q *pqSlavesByRiskGroup) PushSlaveIfFreePorts(s *Slave) {
	// assuming RiskGroupID is set
}

func (q *pqSlavesByRiskGroup) popSlaveinNonconflictingRiskGroup(r *ReplicaSet) *Slave {
	return nil
}

func (c *ClusterAllocator) pqRiskGroups(tx *gorm.DB, p persistence) *pqSlavesByRiskGroup {
	return &pqSlavesByRiskGroup{}
}
