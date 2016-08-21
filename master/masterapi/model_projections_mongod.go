package masterapi

import "github.com/KIT-MAMID/mamid/model"

func ProjectModelMongodToMongod(m *model.Mongod) *Mongod {
	return &Mongod{
		ID:            m.ID,
		Port:          uint(m.Port),
		ParentSlaveID: m.ParentSlaveID,
		ReplicaSetID:  m.ReplicaSetID,
	}
}
