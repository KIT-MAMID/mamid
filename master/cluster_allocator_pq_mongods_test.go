package master

import (
	"github.com/KIT-MAMID/mamid/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func generateMongod(id uint, numMongodsOnSlave uint, maxMongods uint, persistence persistence) *model.Mongod {
	mongods := make([]*model.Mongod, numMongodsOnSlave)
	for i := uint(0); i < numMongodsOnSlave; i++ {
		mongods[i] = &model.Mongod{}
	}
	return &model.Mongod{
		ID: id,
		ParentSlave: &model.Slave{
			Mongods:              mongods,
			PersistentStorage:    persistence == Persistent,
			MongodPortRangeBegin: model.PortNumber(2000),
			MongodPortRangeEnd:   model.PortNumber(2000 + maxMongods),
		},
	}
}

func TestClusterAllocator_PqMongods(t *testing.T) {
	allocator := ClusterAllocator{}
	pq := allocator.pqMongods([]*model.Mongod{
		generateMongod(1, 1, 10, Persistent),
		generateMongod(2, 1, 1, Persistent),
		generateMongod(3, 3, 10, Persistent),
		generateMongod(4, 4, 10, Persistent),
		generateMongod(5, 8, 10, Volatile),
	}, Persistent)

	assert.EqualValues(t, 2, pq.PopMongodOnBusiestSlave().ID)
	assert.EqualValues(t, 4, pq.PopMongodOnBusiestSlave().ID)
	assert.EqualValues(t, 3, pq.PopMongodOnBusiestSlave().ID)
	assert.EqualValues(t, 1, pq.PopMongodOnBusiestSlave().ID)
	assert.Nil(t, pq.PopMongodOnBusiestSlave())
}
