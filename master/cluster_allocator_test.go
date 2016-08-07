package master

import (
	. "github.com/KIT-MAMID/mamid/model"
	_ "github.com/stretchr/testify/assert"
	"testing"
)

/*
Testing the cluster allocator:

Mocking: Bus (Mismatch Messages, Constraint Status)

Fixtures:
 * Flexible infrastructure for creating test scenarios
 * Elegant way to compare pre- and post-state of the database?

What to test?

count methods

priority queue builders

idempotence: test after...
  run of removal
  run of add
  => cancels out every run of the entire algorithm

completeness of the object graph? do we fetch it at the beginning? what about locking?

mismatch generation => use mock of Bus?

*/

// Testing this helper function used in effectiveMemberCount and alreadyAddedMemberCount
func TestClusterAllocator_traverseReplicaSetMongods(t *testing.T) {
	t.Error("test not implemented")
}

func TestClusterAllocator_effectiveMemberCount(t *testing.T) {
	db, _ := InitializeInMemoryDB("")
	allocator := &ClusterAllocator{}

	allocator.effectiveMemberCount(db, &ReplicaSet{})

	t.Error("test not implemented")
}
