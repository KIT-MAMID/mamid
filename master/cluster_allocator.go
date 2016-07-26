package master

import (
	. "github.com/KIT-MAMID/mamid/model"
	"github.com/jinzhu/gorm"
)

type ClusterAllocator struct {
	DB *gorm.DB
}

type persistence uint

const (
	Persistent persistence = 0
	Volatile   persistence = 1
)

type memberCountTuple map[persistence]uint

func (c *ClusterAllocator) CompileMongodLayout() {
	replicaSets, err := c.replicaSets()
	_, _ = err.(error)
	for _, r := range replicaSets {
		c.removeUnneededMembers(r)
		c.addMembers(r)
	}
}

func (c *ClusterAllocator) replicaSets() (replicaSets []ReplicaSet, err error) {
	return []ReplicaSet{}, nil
}

func (c *ClusterAllocator) removeUnneededMembers(r ReplicaSet) {
	for persistence, count := range c.effectiveMemberCount(r) {
		c.removeUnneededMembersByPersistence(r, persistence, count)
	}
}

func (c *ClusterAllocator) removeUnneededMembersByPersistence(r ReplicaSet, p persistence, initialCount uint) {
}

func (c *ClusterAllocator) effectiveMemberCount(r ReplicaSet) memberCountTuple {
	return nil
}

func (c *ClusterAllocator) addMembers(r ReplicaSet) {
	for persistence, count := range c.alreadyAddedMemberCount(r) {
		c.addMembersByPersistence(r, persistence, count)
	}
}

func (c *ClusterAllocator) alreadyAddedMemberCount(r ReplicaSet) memberCountTuple {
	return nil
}

func (c *ClusterAllocator) addMembersByPersistence(r ReplicaSet, p persistence, initialCount uint) {
	replicaSets := c.pqReplicaSets()
	riskGroups := c.pqRiskGroups()
	_ = replicaSets
	_ = riskGroups
}
