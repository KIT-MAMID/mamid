package masterapi

import (
	"encoding/json"
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"net/http"
	"time"
)

type Problem struct {
	ID              uint      `json:"id"`
	Description     string    `json:"description"`
	LongDescription string    `json:"long_description"`
	FirstOccurred   time.Time `json:"first_occurred"`
	LastUpdated     time.Time `json:"last_updated"`
	SlaveId         uint      `json:"slave_id"`
	ReplicaSetId    uint      `json:"replica_set_id"`
}

func (m *MasterAPI) ProblemIndex(w http.ResponseWriter, r *http.Request) {

	var problems []*model.Problem
	err := m.DB.Order("id", false).Find(&problems).Error
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	out := make([]*Problem, len(problems))
	for i, v := range problems {
		out[i] = ProjectModelProblemToProblem(v)
	}
	json.NewEncoder(w).Encode(out)
}
