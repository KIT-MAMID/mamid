package master

import (
	. "github.com/KIT-MAMID/mamid/model"
)

type pqMongods struct {
}

func (q *pqMongods) PopMongodOnBusiestSlave() *Mongod {
	return nil
}

func (c *ClusterAllocator) pqMongods(mongods []*Mongod, p persistence) *pqMongods {
	return &pqMongods{}
}
