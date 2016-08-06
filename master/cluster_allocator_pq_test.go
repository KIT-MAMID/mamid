package master

import (
	"container/heap"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Assert we can build a priority queue with pqSlice
func TestClusterAllocator_pqSlice(t *testing.T) {

	slice := pqSlice{
		Slice: []interface{}{
			10, 20, 11, 21, 30,
		},
		LessComparator: func(i, j interface{}) bool {
			a, a_ok := i.(int)
			b, b_ok := j.(int)
			if !a_ok || !b_ok {
				panic("unexpected type")
			}
			return a < b
		},
	}

	heap.Init(&slice)

	popInt := func() int {
		popped, ok := heap.Pop(&slice).(int)
		if !ok {
			panic("unexpected type")
		}
		fmt.Printf("%#v\n", slice)
		return popped
	}

	assert.Equal(t, 10, popInt())
	assert.Equal(t, 11, popInt())
	assert.Equal(t, 20, popInt())
	assert.Equal(t, 21, popInt())
	assert.Equal(t, 30, popInt())

	// panic when heap is empty
	assert.Panics(t, func() { heap.Pop(&slice) })

}
