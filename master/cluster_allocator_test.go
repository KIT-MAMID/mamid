package master

import (
	. "github.com/KIT-MAMID/mamid/model"
	"github.com/stretchr/testify/assert"
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

func TestClutserAllocator_findUnusedPort(t *testing.T) {

	unusedPort, found := findUnusedPort([]PortNumber{2, 3, 5}, 2, 6)
	assert.EqualValues(t, 4, unusedPort, "should find lowest free port number")

	unusedPort, found = findUnusedPort([]PortNumber{}, 2, 5)
	assert.EqualValues(t, 2, unusedPort, "should use minPort when no port used")

	unusedPort, found = findUnusedPort([]PortNumber{0}, 2, 5)
	assert.EqualValues(t, 2, unusedPort)

	var uninitialized []PortNumber
	unusedPort, found = findUnusedPort(uninitialized, 2, 5)
	assert.EqualValues(t, 2, unusedPort)

	unusedPort, found = findUnusedPort([]PortNumber{2, 3, 4}, 2, 5)
	assert.Equal(t, false, found, "should not find a port if no port free")

}
