package master

import (
	"container/heap"
	. "github.com/KIT-MAMID/mamid/model"
	"github.com/jinzhu/gorm"
)

type pqSlavesByRiskGroup struct {
	slaveQueues map[uint]pqSlice
}

func (q *pqSlavesByRiskGroup) PopSlaveInNonconflictingRiskGroup() *Slave {

	// ARGMAX over the slave queues heads' slaveBusyRate
	var currentSlave *Slave
	for _, slavePQ := range q.slaveQueues {
		// TODO actual peek would be better
		peek, assertionOK := heap.Pop(&slavePQ).(*Slave)
		if !assertionOK {
			panic("unexpected type in slave pqSlice")
		}
		if currentSlave == nil || slaveBusyRate(peek) < slaveBusyRate(currentSlave) {
			currentSlave = peek
		} else {
			heap.Push(&slavePQ, peek)
		}
	}

	return currentSlave
}

func (q *pqSlavesByRiskGroup) slaveComparator(a, b interface{}) bool {
	s1, s1_ok := a.(*Slave)
	s2, s2_ok := b.(*Slave)
	if s1_ok || s2_ok {
		panic("unexpected type in slave pqSlice")
	}
	return slaveBusyRate(s1) < slaveBusyRate(s2) // least busy slave first
}

func (c *ClusterAllocator) pqRiskGroups(tx *gorm.DB, r *ReplicaSet, p persistence) *pqSlavesByRiskGroup {

	usedRiskGroupIDs := make([]uint, 0)

	for _, m := range r.Mongods {
		usedRiskGroupIDs = append(usedRiskGroupIDs, m.ParentSlave.RiskGroupID)
	}

	var candidateRiskGroups []*RiskGroup
	if err := tx.Where("id not in (?)", usedRiskGroupIDs).Find(&candidateRiskGroups).Error; err != nil {
		panic(err)
	}

	// find usable slaves among candidate risk groups
	var candidateSlaves []*Slave
	if err := tx.Where("risk_group_id not in (?)", usedRiskGroupIDs).Where("persistent_storage = ?", p.PersistentStorage()).Find(&candidateSlaves).Error; err != nil {
		panic(err)
	}

	// partition slaves by RiskGroupID
	pq := &pqSlavesByRiskGroup{
		slaveQueues: make(map[uint]pqSlice),
	}
	for _, s := range candidateSlaves {
		if slavePersistence(s) == p {
			slice, ok := pq.slaveQueues[s.RiskGroupID]
			if !ok {
				slice = pqSlice{make([]interface{}, 0), pq.slaveComparator}
				pq.slaveQueues[s.RiskGroupID] = slice
			}

			runningMongods, maxMongods := slaveUsage(s)
			if slavePersistence(s) == p && runningMongods < maxMongods {
				slice.Push(s)
			}

		}
	}

	// Initialize slave groups as heaps
	for _, slice := range pq.slaveQueues {
		heap.Init(&slice)
	}

	return pq

}
