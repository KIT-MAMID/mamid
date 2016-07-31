package master

import (
	. "github.com/KIT-MAMID/mamid/model"
	"github.com/jinzhu/gorm"
)

type ClusterAllocator struct {
}

type persistence uint

const (
	Persistent persistence = 0
	Volatile   persistence = 1
)

type memberCountTuple map[persistence]uint

func (c *ClusterAllocator) CompileMongodLayout(tx *gorm.DB) (err error) {

	defer func() {
		r := recover()
		if r == nil {
			return
		}
		switch r {
		case r == nil:
			return
		case r == gorm.ErrInvalidTransaction:
			err = r.(error)
		default:
			panic(r)
		}
	}()

	replicaSets := c.replicaSets(tx)
	for _, r := range replicaSets {
		c.removeUnneededMembers(tx, r)
		c.addMembers(tx, r)
	}

	return err
}

func (c *ClusterAllocator) replicaSets(tx *gorm.DB) []*ReplicaSet {
	return []*ReplicaSet{}
}

func (c *ClusterAllocator) removeUnneededMembers(tx *gorm.DB, r *ReplicaSet) {
	for persistence, count := range c.effectiveMemberCount(tx, r) {
		c.removeUnneededMembersByPersistence(tx, r, persistence, count)
	}
}

func (c *ClusterAllocator) removeUnneededMembersByPersistence(tx *gorm.DB, r *ReplicaSet, p persistence, initialCount uint) {
}

func (c *ClusterAllocator) effectiveMemberCount(tx *gorm.DB, r *ReplicaSet) memberCountTuple {

	if err := tx.Related(&r.Mongods, "Mongods").Error; err != nil {
		panic(err)
	}

	var res memberCountTuple

	for _, m := range r.Mongods {

		if err := tx.Related(&m.ObservedState, "ObservedState").Error; err != nil {
			panic(err)
		}
		if err := tx.Related(&m.DesiredState, "DesiredState").Error; err != nil {
			panic(err)
		}
		if err := tx.Related(&m.ParentSlave, "ParentSlave").Error; err != nil {
			panic(err)
		}

		if m.ObservedState.ExecutionState == MongodExecutionStateRunning &&
			m.DesiredState.ExecutionState == MongodExecutionStateRunning {
			if m.ParentSlave.PersistentStorage {
				res[Persistent]++
			} else {
				res[Volatile]++
			}
		}
	}

	return res
}

func (c *ClusterAllocator) addMembers(tx *gorm.DB, r *ReplicaSet) {
	for persistence, count := range c.alreadyAddedMemberCount(tx, r) {
		c.addMembersByPersistence(tx, r, persistence, count)
	}
}

func (c *ClusterAllocator) alreadyAddedMemberCount(tx *gorm.DB, r *ReplicaSet) memberCountTuple {
	if err := tx.Related(&r.Mongods, "Mongods").Error; err != nil {
		panic(err)
	}

	var res memberCountTuple

	for _, m := range r.Mongods {

		if err := tx.Related(&m.DesiredState, "DesiredState").Error; err != nil {
			panic(err)
		}

		if err := tx.Related(&m.ParentSlave, "ParentSlave").Error; err != nil {
			panic(err)
		}

		if m.ParentSlave.ConfiguredState != SlaveStateDisabled &&
			m.DesiredState.ExecutionState != MongodExecutionStateNotRunning &&
			m.DesiredState.ExecutionState != MongodExecutionStateDestroyed {
			if m.ParentSlave.PersistentStorage {
				res[Persistent]++
			} else {
				res[Volatile]++
			}
		}
	}

	return res
}

func (c *ClusterAllocator) addMembersByPersistence(tx *gorm.DB, r *ReplicaSet, p persistence, initialCount uint) {
	replicaSets := c.pqReplicaSets(tx)
	riskGroups := c.pqRiskGroups(tx)
	_ = replicaSets
	_ = riskGroups
}
