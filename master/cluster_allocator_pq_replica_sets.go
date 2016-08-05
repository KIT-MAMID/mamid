package master

import (
	"container/heap"
	. "github.com/KIT-MAMID/mamid/model"
)

////////////////////////////////////////////////////////////////////////////////
// Public Interface
// TODO use pqSlice instead of pqReplicaSetItemSlice
////////////////////////////////////////////////////////////////////////////////

type pqReplicaSets struct {
	slice pqReplicaSetItemSlice
}

func (c *ClusterAllocator) pqReplicaSets(replicaSets []*ReplicaSet, p persistence) *pqReplicaSets {

	pq := &pqReplicaSets{
		slice: pqReplicaSetItemSlice{
			items: make([]*pqReplicaSetItem, len(replicaSets)),
			p:     p,
		},
	}

	for i, r := range replicaSets {
		pq.slice.items[i] = replicaSetItemFromReplicaSet(r)
	}

	heap.Init(&pq.slice)

	return pq
}

func (q *pqReplicaSets) Pop() *ReplicaSet {
	return heap.Pop(&q.slice).(*ReplicaSet)
}

func (q *pqReplicaSets) PushIfDegraded(r *ReplicaSet) {
	item := replicaSetItemFromReplicaSet(r)
	if !item.degraded[q.slice.p] {
		heap.Push(&q.slice, item)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Hidden Implementation using container/heap
////////////////////////////////////////////////////////////////////////////////

type pqReplicaSetItemSlice struct {
	items []*pqReplicaSetItem
	p     persistence
}

type pqReplicaSetItem struct {
	r               *ReplicaSet
	p               persistence
	relMemberCounts map[persistence]float64
	degraded        map[persistence]bool
}

func (s pqReplicaSetItemSlice) Len() int {
	return len(s.items)
}

func (s pqReplicaSetItemSlice) Less(left, right int) bool {
	return s.items[left].relMemberCounts[s.p] < s.items[right].relMemberCounts[s.p]
}

func (s *pqReplicaSetItemSlice) Swap(i, j int) {
	s.items[i], s.items[j] = s.items[j], s.items[i]
}

func (s pqReplicaSetItemSlice) Push(i interface{}) {
	item, ok := i.(*pqReplicaSetItem)
	if !ok {
		panic("pqReplicaSetItemSlice should only be used with *ReplicaSet")
	}
	s.items = append(s.items, item)
}

func (s pqReplicaSetItemSlice) Pop() interface{} {
	ret := s.items[len(s.items)-1]
	s.items = s.items[0 : len(s.items)-1]
	return ret
}

func replicaSetItemFromReplicaSet(r *ReplicaSet) *pqReplicaSetItem {
	// Find all persistent
	desiredCounts := map[persistence]uint{
		Persistent: 0,
		Volatile:   0,
	}

	for _, m := range r.Mongods {
		if m.ParentSlave.PersistentStorage { // TODO is ParentSlave always resolved?
			desiredCounts[Persistent]++
		} else {
			desiredCounts[Volatile]++
		}
	}

	return &pqReplicaSetItem{
		r: r,
		relMemberCounts: map[persistence]float64{
			Persistent: float64(desiredCounts[Persistent]) / float64(r.PersistentMemberCount),
			Volatile:   float64(desiredCounts[Volatile]) / float64(r.VolatileMemberCount),
		},
		degraded: map[persistence]bool{
			Persistent: desiredCounts[Persistent] < r.PersistentMemberCount,
			Volatile:   desiredCounts[Volatile] < r.VolatileMemberCount,
		},
	}
}
