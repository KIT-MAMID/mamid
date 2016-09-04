package masterapi

import (
	"github.com/KIT-MAMID/mamid/model"
	"github.com/KIT-MAMID/mamid/msp"
)

func ProjectModelMongodToMongod(m *model.Mongod) *Mongod {
	var executionState string
	if m.ObservedStateID.Valid {
		executionState = string(ModelExecutionStateToApiMongodState(m.ObservedState.ExecutionState))
	} else {
		executionState = "unobserved"
	}
	return &Mongod{
		ID:                     m.ID,
		Port:                   uint(m.Port),
		ParentSlaveID:          m.ParentSlaveID,
		ReplicaSetID:           m.ReplicaSetID.Int64,
		ObservedExecutionState: executionState,
	}
}

func ModelExecutionStateToApiMongodState(e model.MongodExecutionState) msp.MongodState {
	switch e {
	case model.MongodExecutionStateDestroyed:
		return msp.MongodStateDestroyed
	case model.MongodExecutionStateNotRunning:
		return msp.MongodStateNotRunning
	case model.MongodExecutionStateRecovering:
		return msp.MongodStateRecovering
	case model.MongodExecutionStateRunning:
		return msp.MongodStateRunning
	case model.MongodExecutionStateForceDestroyed:
		return msp.MongodStateForceDestroyed
	default:
		return "invalid" // Invalid
		//TODO New states
	}
}
