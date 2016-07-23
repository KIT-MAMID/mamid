package masterapi

import (
	"encoding/json"
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
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

func (m *MasterAPI) ProblemById(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["problemId"]
	id64, err := strconv.ParseUint(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := uint(id64)

	var problems []model.Problem
	err = m.DB.Find(&problems, &model.Slave{ID: id}).Error

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	if len(problems) == 0 { // Not found?
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if len(problems) > 1 {
		log.Printf("inconsistency: multiple problems for problem.ID = %d found in database", len(problems))
	}
	json.NewEncoder(w).Encode(ProjectModelProblemToProblem(&problems[0]))
	return
}
