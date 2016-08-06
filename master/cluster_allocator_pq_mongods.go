package master

import (
	"container/heap"
	. "github.com/KIT-MAMID/mamid/model"
)

type pqMongods struct {
	slice pqSlice
}

type pqMongodItem struct {
	Mongod         *Mongod
	ParentBusyRate float64
}

func pqMongodItemFromMongod(mongod *Mongod) *pqMongodItem {
	return &pqMongodItem{
		Mongod:         mongod,
		ParentBusyRate: slaveBusyRate(mongod.ParentSlave),
	}
}

func (q *pqMongods) PopMongodOnBusiestSlave() *Mongod {

	if q.slice.Len() <= 0 {
		return nil
	}

	item, ok := heap.Pop(&q.slice).(*pqMongodItem)
	if !ok {
		panic("unexpected type in pqMongods")
	}
	return item.Mongod
}

func (c *ClusterAllocator) pqMongods(mongods []*Mongod, p persistence) *pqMongods {

	items := make([]interface{}, len(mongods))
	for _, m := range mongods {
		if slavePersistence(m.ParentSlave) == p {
			items = append(items, pqMongodItemFromMongod(m))
		}
	}

	q := &pqMongods{
		slice: pqSlice{
			items,
			func(i, j interface{}) bool {
				a, a_ok := i.(*pqMongodItem)
				b, b_ok := j.(*pqMongodItem)
				if !a_ok || !b_ok {
					panic("unexpected type in pqMongods")
				}
				return a.ParentBusyRate > b.ParentBusyRate //'>' to have busiest node at top of heap TODO correct
			},
		},
	}

	heap.Init(&q.slice)

	return q
}
