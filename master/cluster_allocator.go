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
	}

	c.addMembers(tx, replicaSets)

	return err
}

func (c *ClusterAllocator) replicaSets(tx *gorm.DB) (replicaSets []*ReplicaSet) {

	if err := tx.Where(ReplicaSet{}).Find(&replicaSets).Error; err != nil {
		panic(err)
	}

	for _, r := range replicaSets {

		if err := tx.Model(r).Related(&r.Mongods, "Mongods").Error; err != nil {
			panic(err)
		}

		for _, m := range r.Mongods {

			if err := tx.Model(m).Related(&m.ObservedState, "ObservedState").Error; err != nil {
				panic(err)
			}
			if err := tx.Model(m).Related(&m.DesiredState, "DesiredState").Error; err != nil {
				panic(err)
			}
			if err := tx.Model(m).Related(&m.ParentSlave, "ParentSlave").Error; err != nil {
				panic(err)
			}

		}

	}

	return replicaSets
}

func (c *ClusterAllocator) removeUnneededMembers(tx *gorm.DB, r *ReplicaSet) {
	for persistence, count := range c.effectiveMemberCount(tx, r) {
		c.removeUnneededMembersByPersistence(tx, r, persistence, count)
	}
}

func slavePersistence(s *Slave) persistence {
	switch s.PersistentStorage {
	case true:
		return Persistent
	default:
		return Volatile
	}
}

func (c *ClusterAllocator) removeUnneededMembersByPersistence(tx *gorm.DB, r *ReplicaSet, p persistence, initialCount uint) {

	var configuredMemberCount uint
	if p == Persistent {
		configuredMemberCount = r.PersistentMemberCount
	} else if p == Volatile {
		configuredMemberCount = r.VolatileMemberCount
	}

	// Destroy any Mongod running on disabled slaves (no specific priority)
	for initialCount > configuredMemberCount {
		for _, m := range r.Mongods {

			if m.ParentSlave.ConfiguredState == SlaveStateDisabled &&
				slavePersistence(m.ParentSlave) == p {
				// destroy
				panic("not implemented")

				initialCount--
			}
		}
	}

	// Remove superfluous Mongods on busiest slaves first
	removalPQ := c.pqMongods(r.Mongods, p)
	for initialCount > configuredMemberCount {
		// Destroy any Mongod (lower priority)
		m := removalPQ.PopMongodOnBusiestSlave()

		if m == nil {
			break
		}

		// destroy
		panic("not implemented")

		initialCount--

	}

}

func (c *ClusterAllocator) effectiveMemberCount(tx *gorm.DB, r *ReplicaSet) memberCountTuple {

	var res memberCountTuple

	for _, m := range r.Mongods {

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

func (c *ClusterAllocator) addMembers(tx *gorm.DB, replicaSets []*ReplicaSet) {

	for _, persistence := range []persistence{Volatile, Persistent} {

		// build prioritization datastructures
		// will only return items that match current persistence and actually need more members

		pqReplicaSets := c.pqReplicaSets(replicaSets, persistence)
		pqRiskGroups := c.pqRiskGroups(tx, persistence)

		for r := pqReplicaSets.Pop(); r != nil; {

			if s := pqRiskGroups.PopSlaveinNonconflictingRiskGroup(r); s != nil {

				// spawn new Mongod m on s and add it to r.Mongods
				// compute MongodState for m and set the DesiredState variable
				panic("not implemented")

				pqReplicaSets.PushIfDegraded(r)
				pqRiskGroups.PushSlaveIfFreePorts(s)

			} else {
				// send constraint not fulfilled notification
				panic("not implemented")
			}
		}

	}
}

func (c *ClusterAllocator) alreadyAddedMemberCount(tx *gorm.DB, r *ReplicaSet) memberCountTuple {

	var res memberCountTuple

	for _, m := range r.Mongods {

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
