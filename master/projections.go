package master

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/KIT-MAMID/mamid/msp"
)

func ProjectModelShardingRoleToMSPShardingRole(r model.ShardingRole) (out msp.ShardingRole, err error) {
	switch r {
	case model.ShardingRoleNone:
		out = msp.ShardingRoleNone
	case model.ShardingRoleShardServer:
		out = msp.ShardingRoleShardServer
	case model.ShardingRoleConfigServer:
		out = msp.ShardingRoleConfigServer
	default:
		out = ""
		err = fmt.Errorf("cannot convert unknown model.ShardingRole `%s`", r)
	}
	return
}

func ProjectMSPShardingRoleToModelShardingRole(r msp.ShardingRole) (out model.ShardingRole, err error) {
	switch r {
	case msp.ShardingRoleNone:
		out = model.ShardingRoleNone
	case msp.ShardingRoleShardServer:
		out = model.ShardingRoleShardServer
	case msp.ShardingRoleConfigServer:
		out = model.ShardingRoleConfigServer
	default:
		out = ""
		err = fmt.Errorf("cannot convert unknown msp.ShardingRole `%s`", r)
	}
	return
}

func mspMongodStateFromExecutionState(s model.MongodExecutionState) (msp.MongodState, error) {
	switch s {
	case model.MongodExecutionStateDestroyed:
		return msp.MongodStateDestroyed, nil
	case model.MongodExecutionStateNotRunning:
		return msp.MongodStateNotRunning, nil
	case model.MongodExecutionStateRecovering:
		return msp.MongodStateRecovering, nil
	case model.MongodExecutionStateRunning:
		return msp.MongodStateRunning, nil
	case model.MongodExecutionStateForceDestroyed:
		return msp.MongodStateForceDestroyed, nil
	default:
		return "", fmt.Errorf("deployer: unable to map `%v` from model.ExecutionState to msp.MongodState", s)
	}
}

func ProjectModelMongodbCredentialToMSPMongodCredential(m model.MongodbCredential) msp.MongodCredential {
	return msp.MongodCredential{
		Username: m.Username,
		Password: m.Password,
	}
}
