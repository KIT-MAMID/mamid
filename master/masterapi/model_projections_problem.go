package masterapi

import "github.com/KIT-MAMID/mamid/model"

func ProjectModelProblemToProblem(m *model.Problem) *Problem {
	return &Problem{
		ID:              m.ID,
		Description:     m.Description,
		LongDescription: m.LongDescription,
		FirstOccurred:   m.FirstOccurred,
		LastUpdated:     m.LastUpdated,
		SlaveId:         m.SlaveID,
		ReplicaSetId:    m.ReplicaSetID,
	}
}
