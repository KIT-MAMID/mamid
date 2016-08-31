package masterapi

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
)

func ProjectModelReplicaSetToReplicaSet(m *model.ReplicaSet) *ReplicaSet {
	shardingRole, err := ProjectModelShardingRoleToShardingRole(m.ShardingRole)
	if err != nil {
		panic(err) // Forward projections should always be possible
	}
	return &ReplicaSet{
		ID:                  m.ID,
		Name:                m.Name,
		PersistentNodeCount: m.PersistentMemberCount,
		VolatileNodeCount:   m.VolatileMemberCount,
		ShardingRole:        shardingRole,
	}
}

func ProjectReplicaSetToModelReplicaSet(r *ReplicaSet) (*model.ReplicaSet, error) {
	if r.Name == "" {
		return nil, fmt.Errorf("Replica set name may not be empty")
	}
	shardingRole, err := ProjectShardingRoleToModelShardingRole(r.ShardingRole)
	if err != nil {
		return nil, err
	}
	return &model.ReplicaSet{
		ID:   r.ID,
		Name: r.Name,
		PersistentMemberCount: r.PersistentNodeCount,
		VolatileMemberCount:   r.VolatileNodeCount,
		ShardingRole:          shardingRole,
	}, nil
}

func ProjectShardingRoleToModelShardingRole(api ShardingRole) (out model.ShardingRole, err error) {
	switch api {
	case ShardingRoleNone:
		out = model.ShardingRoleNone
	case ShardingRoleShardServer:
		out = model.ShardingRoleShardServer
	case ShardingRoleConfigServer:
		out = model.ShardingRoleConfigServer
	default:
		out = ""
		err = fmt.Errorf("cannot convert unknown model.ShardingRole `%s`", api)
	}
	return
}

func ProjectModelShardingRoleToShardingRole(modelRole model.ShardingRole) (out ShardingRole, err error) {
	switch modelRole {
	case model.ShardingRoleNone:
		out = ShardingRoleNone
	case model.ShardingRoleShardServer:
		out = ShardingRoleShardServer
	case model.ShardingRoleConfigServer:
		out = ShardingRoleConfigServer
	default:
		out = ""
		err = fmt.Errorf("cannot convert unknown masterapi.ShardingRole `%s`", modelRole)
	}
	return
}
