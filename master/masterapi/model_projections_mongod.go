package masterapi

import (
	"github.com/KIT-MAMID/mamid/master"
	"github.com/KIT-MAMID/mamid/model"
)

func ProjectModelMongodToMongod(m *model.Mongod) *Mongod {
	var executionState string
	if m.ObservedStateID.Valid {
		executionState = string(master.ModelExecutionStateToMspMongodState(m.ObservedState.ExecutionState))
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
