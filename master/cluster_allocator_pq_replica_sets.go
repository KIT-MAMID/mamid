package master

import (
	"container/heap"
	. "github.com/KIT-MAMID/mamid/model"
)

type pqReplicaSets struct {
	slice pqSlice
	p     persistence
}

type pqReplicaSetItem struct {
	r               *ReplicaSet
	p               persistence
	relMemberCounts map[persistence]float64
	degraded        map[persistence]bool
}

func (c *ClusterAllocator) pqReplicaSets(replicaSets []*ReplicaSet, p persistence) *pqReplicaSets {

	pq := &pqReplicaSets{
		p: p,
		slice: pqSlice{
			Slice: make([]interface{}, 0),
			LessComparator: func(i, j interface{}) bool {
				return i.(*pqReplicaSetItem).relMemberCounts[p] < j.(*pqReplicaSetItem).relMemberCounts[p]
			},
		},
	}

	for _, r := range replicaSets {
		item := replicaSetItemFromReplicaSet(r)
		if item.degraded[p] {
			pq.slice.Push(item)
		}
	}

	heap.Init(&pq.slice)

	return pq
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

	relMemberCounts := make(map[persistence]float64, 2)
	degraded := make(map[persistence]bool, 2)

	for _, p := range []persistence{Persistent, Volatile} {

		var memberCount uint
		if p.PersistentStorage() {
			memberCount = r.PersistentMemberCount
		} else {
			memberCount = r.VolatileMemberCount
		}

		degraded[p] = memberCount > 0 && desiredCounts[p] < memberCount

		if degraded[p] {
			relMemberCounts[p] = float64(desiredCounts[p]) / float64(memberCount)
		} else {
			relMemberCounts[p] = float64(1.0)
		}

	}

	return &pqReplicaSetItem{
		r:               r,
		relMemberCounts: relMemberCounts,
		degraded:        degraded,
	}
}

func (q *pqReplicaSets) Pop() *ReplicaSet {
	if q.slice.Len() <= 0 {
		return nil
	}
	return heap.Pop(&q.slice).(*pqReplicaSetItem).r
}

func (q *pqReplicaSets) PushIfDegraded(r *ReplicaSet) {
	item := replicaSetItemFromReplicaSet(r)
	if item.degraded[q.p] {
		heap.Push(&q.slice, item)
	}
}
