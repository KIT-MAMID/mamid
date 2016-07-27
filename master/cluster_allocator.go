package master

import (
	. "github.com/KIT-MAMID/mamid/model"
	"github.com/jinzhu/gorm"
	"log"
)

type ClusterAllocator struct {
}

type persistence uint

const (
	Persistent persistence = 0
	Volatile   persistence = 1
)

type memberCountTuple map[persistence]uint

func (c *ClusterAllocator) CompileMongodLayout(tx *gorm.DB) {

	replicaSets, err := c.replicaSets(tx)
	_, _ = err.(error)
	for _, r := range replicaSets {
		c.removeUnneededMembers(tx, r)
		c.addMembers(tx, r)
	}
}

func (c *ClusterAllocator) replicaSets(tx *gorm.DB) (replicaSets []ReplicaSet, err error) {
	return []ReplicaSet{}, nil
}

func (c *ClusterAllocator) removeUnneededMembers(tx *gorm.DB, r ReplicaSet) {
	for persistence, count := range c.effectiveMemberCount(tx, r) {
		c.removeUnneededMembersByPersistence(r, persistence, count)
	}
}

func (c *ClusterAllocator) removeUnneededMembersByPersistence(tx *gorm.DB, r ReplicaSet, p persistence, initialCount uint) {
}

func (c *ClusterAllocator) effectiveMemberCount(tx *gorm.DB, r ReplicaSet) memberCountTuple {
	return nil
}

func (c *ClusterAllocator) addMembers(tx *gorm.DB, r ReplicaSet) {
	for persistence, count := range c.alreadyAddedMemberCount(r) {
		c.addMembersByPersistence(tx, r, persistence, count)
	}
}

func (c *ClusterAllocator) alreadyAddedMemberCount(tx *gorm.DB, r ReplicaSet) memberCountTuple {
	return nil
}

func (c *ClusterAllocator) addMembersByPersistence(tx *gorm.DB, r ReplicaSet, p persistence, initialCount uint) {
	replicaSets := c.pqReplicaSets(tx)
	riskGroups := c.pqRiskGroups(tx)
	_ = replicaSets
	_ = riskGroups
}
