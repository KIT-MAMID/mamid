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

func (q *pqSlavesByRiskGroup) popSlaveinNonconflictingRiskGroup() (r *RiskGroup) {
	return nil
}

func (c *ClusterAllocator) pqRiskGroups(tx *gorm.DB, p persistence) *pqSlavesByRiskGroup {
	return &pqSlavesByRiskGroup{}
}
