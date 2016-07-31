package master

import (
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

/*
Should partition the
  set of ALL slaves of type either persistent|volatile
  with >= 1 free Mongod Port
  into a map[RiskGroup]PriorityQueue
*/
func TestClusterAllocator_pqRiskGroups(t *testing.T) {
	t.Error("test not implemented")
}

/*
Should
  prioritize ReplicaSets heavy-degraded replica sets
    (meaning relative amount of missing persistent/volatile members)
  only contain degraded ReplicaSets
*/
func TestClusterAllocator_pqReplicaSets(t *testing.T) {
	t.Error("test not implemented")
}