package masterapi

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
)

func ProjectModelReplicaSetToReplicaSet(m *model.ReplicaSet) *ReplicaSet {
	return &ReplicaSet{
		ID:                              m.ID,
		Name:                            m.Name,
		PersistentNodeCount:             m.PersistentMemberCount,
		VolatileNodeCount:               m.VolatileMemberCount,
		ConfigureAsShardingConfigServer: m.ConfigureAsShardingConfigServer,
	}
}

func ProjectReplicaSetToModelReplicaSet(r *ReplicaSet) (*model.ReplicaSet, error) {
	if r.Name == "" {
		return nil, fmt.Errorf("Replica set name may not be empty")
	}
	return &model.ReplicaSet{
		ID:   r.ID,
		Name: r.Name,
		PersistentMemberCount:           r.PersistentNodeCount,
		VolatileMemberCount:             r.VolatileNodeCount,
		ConfigureAsShardingConfigServer: r.ConfigureAsShardingConfigServer,
	}, nil
}
