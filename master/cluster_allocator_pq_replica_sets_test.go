package master

import (
	. "github.com/KIT-MAMID/mamid/model"
	"github.com/stretchr/testify/assert"
	_ "log"
	"testing"
)

type fixturesContainer_pqReplicaSets struct {
	s1, s2, s3 *Slave // s1 is persistent, others are volatile
	// lacks 1 of 2 volatile members
	r1 *ReplicaSet
	// lacks 1 of 2 persistent members
	r2 *ReplicaSet
	// needs 3 of 3 (volatile|persistent) members
	r3 *ReplicaSet
}

func fixtures_pqReplicaSets() (*ClusterAllocator, fixturesContainer_pqReplicaSets) {
	s1 := &Slave{
		PersistentStorage: true,
		Mongods:           []*Mongod{&Mongod{}, &Mongod{}},
	}
	s1.Mongods[0].ParentSlave = s1
	s1.Mongods[1].ParentSlave = s1

	s2 := &Slave{Mongods: []*Mongod{&Mongod{}}}
	s2.Mongods[0].ParentSlave = s2

	s3 := &Slave{Mongods: []*Mongod{&Mongod{}}}
	s3.Mongods[0].ParentSlave = s3

	r1 := &ReplicaSet{
		Name: "r1",
		PersistentMemberCount: 1,
		VolatileMemberCount:   2,
		Mongods: []*Mongod{
			s1.Mongods[0],
			s3.Mongods[0],
		},
	}

	r2 := &ReplicaSet{
		Name: "r2",
		PersistentMemberCount: 2,
		VolatileMemberCount:   1,
		Mongods: []*Mongod{
			s1.Mongods[1],
			s2.Mongods[0],
		},
	}

	r3 := &ReplicaSet{
		Name: "r3",
		PersistentMemberCount: 3,
		VolatileMemberCount:   3,
		Mongods:               []*Mongod{},
	}

	return &ClusterAllocator{}, fixturesContainer_pqReplicaSets{
		s1, s2, s3,
		r1, r2, r3,
	}
}

func TestClusterAllocator_pqReplicaSets_emptyReturnsNil(t *testing.T) {
	c, _ := fixtures_pqReplicaSets()
	q := c.pqReplicaSets([]*ReplicaSet{}, Persistent)

	assert.Equal(t, 0, q.slice.Len())
	assert.Nil(t, q.Pop())
}

func TestClusterAllocator_pqReplicaSets_filteredByPersistenceDimension(t *testing.T) {
	c, f := fixtures_pqReplicaSets()

	q := c.pqReplicaSets([]*ReplicaSet{f.r1}, Persistent)
	assert.Nil(t, q.Pop(), "host not degraded in the specified persistence dimension should not be in queue")

	replicaSets := []*ReplicaSet{f.r1, f.r2, f.r3}

	// Filter persistent
	q = c.pqReplicaSets(replicaSets, Persistent)

	assert.Equal(t, f.r3, q.Pop(), "heaver degraded first")
	assert.Equal(t, f.r2, q.Pop(), "less degraded second")
	assert.Nil(t, q.Pop(), "hosts not degraded in 'Persistent' persistence dimension should not be in queue")

	// Filter volatile
	q = c.pqReplicaSets(replicaSets, Volatile)
	assert.Equal(t, f.r3, q.Pop(), "heavier degraded first")
	assert.Equal(t, f.r1, q.Pop(), "less degraded second")
	assert.Nil(t, q.Pop(), "hosts not degraded in 'Persistent' persistence dimension should not be in queue")

}

func TestClusterAllocator_pqReplicaSets_onlyDegradedPushedToHeap(t *testing.T) {
	c, f := fixtures_pqReplicaSets()

	q := c.pqReplicaSets([]*ReplicaSet{}, Persistent)
	assert.Nil(t, q.Pop())

	r := f.r2

	q.PushIfDegraded(r)
	assert.Equal(t, r, q.Pop(), "should allow push of degraded replica set")

	// fix member count, i.e. fix degradation
	r.PersistentMemberCount = 1

	q.PushIfDegraded(r)
	assert.Nil(t, q.Pop(), "should not allow push of non-degraded replica set")

}
