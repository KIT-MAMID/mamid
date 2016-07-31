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

	addMembers(replicaSets)

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

	if err := tx.Model(r).Related(&r.Mongods, "Mongods").Error; err != nil {
		panic(err)
	}

	for initialCount > configuredMemberCount {
		// Destroy any Mongod running on disabled slaves (higher priority)
		for _, m := range r.Mongods {

			if err := tx.Model(m).Related(&m.ParentSlave, "ParentSlave").Error; err != nil {
				panic(err)
			}

			if m.ParentSlave.ConfiguredState == SlaveStateDisabled &&
				slavePersistence(m.ParentSlave) == p {
				// destroy
				panic("not implemented")

				initialCount--
			}
		}
	}

	for initialCount > configuredMemberCount {
		// Destroy any Mongod (lower priority)
		for _, m := range r.Mongods {

			// Only fetch ParentSlave where not already fetched
			if m.ParentSlave == nil {
				if err := tx.Model(m).Related(&m.ParentSlave, "ParentSlave").Error; err != nil {
					panic(err)
				}
			}

			if slavePersistence(m.ParentSlave) == p {
				// destroy
				panic("not implemented")
			}

		}

	}

}

func (c *ClusterAllocator) effectiveMemberCount(tx *gorm.DB, r *ReplicaSet) memberCountTuple {

	var res memberCountTuple

	traverseReplicaSetMongods(tx, r, func(m *Mongod) {

		if m.ObservedState.ExecutionState == MongodExecutionStateRunning &&
			m.DesiredState.ExecutionState == MongodExecutionStateRunning {
			if m.ParentSlave.PersistentStorage {
				res[Persistent]++
			} else {
				res[Volatile]++
			}
		}
	})

	return res
}

func (c *ClusterAllocator) addMembers(tx *gorm.DB, replicaSets []*ReplicaSet) {

	for _, persistence := range []persistence{Volatile, Persistent} {

		// build prioritization datastructures
		// will only return items that match current persistence and actually need more members

		pqReplicaSets := c.pqReplicaSets(replicaSets, persistence)
		pqRiskGroups := c.pqRiskGroups(tx, persistence)

		for r := pqReplicaSets.Pop(); r != nil; {

			if s := pqRiskGroups.popSlaveinNonconflictingRiskGroup(r); g != nil {

				// spawn new Mongod m on s and add it to r.Mongods
				// compute MongodState for m and set the DesiredState variable
				panic("not implemented")

				if replicaSetNeedsMoreMembers(r, p) {
					pqReplicaSets.Push(r)
				}

				if slaveHasFreePorts(s) {
					pqRiskGroups.pushSlave(s)
				}

			} else {
				// send constraint not fulfilled notification
				panic("not implemented")
			}
		}

	}
}

func (c *ClusterAllocator) alreadyAddedMemberCount(tx *gorm.DB, r *ReplicaSet) memberCountTuple {

	var res memberCountTuple

	traverseReplicaSetMongods(tx, r, func(m *Mongod) {

		if m.ParentSlave.ConfiguredState != SlaveStateDisabled &&
			m.DesiredState.ExecutionState != MongodExecutionStateNotRunning &&
			m.DesiredState.ExecutionState != MongodExecutionStateDestroyed {
			if m.ParentSlave.PersistentStorage {
				res[Persistent]++
			} else {
				res[Volatile]++
			}
		}

	})

	return res
}

// Traverse a Replica Set's Mongods, for which the following Attributes have been fetched
// 	ParentSlave
//	ObservedState
// 	DesiredState
func traverseReplicaSetMongods(tx *gorm.DB, r *ReplicaSet, handler func(m *Mongod)) {

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

		handler(m)

	}

}
