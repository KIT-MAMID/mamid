package masterapi

import (
	"encoding/json"
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"net/http"
	"strconv"
)

type Mongod struct {
	ID                     int64  `json:"id"`
	Port                   uint   `json:"slave_port"`
	ReplicaSetID           int64  `json:"replica_set_id"`
	ParentSlaveID          int64  `json:"parent_slave_id"`
	ObservedExecutionState string `json:"observed_execution_state"`
}

func mongodToApiMongod(tx *gorm.DB, m *model.Mongod) (*Mongod, error) {
	if res := tx.Model(&m).Related(&m.ObservedState, "ObservedState"); res.Error != nil && !res.RecordNotFound() {
		return &Mongod{}, res.Error
	}
	return ProjectModelMongodToMongod(m), nil
}

func (m *MasterAPI) MongodsBySlave(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["slaveId"]
	slaveId, err := strconv.ParseInt(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tx := m.DB.Begin()
	defer tx.Rollback()

	var slave model.Slave
	getSlaveRes := tx.First(&slave, slaveId)
	if getSlaveRes.RecordNotFound() {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err := getSlaveRes.Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, err.Error())
		return
	}

	if err := tx.Model(&slave).Order("id", false).Related(&slave.Mongods, "Mongods").Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, err.Error())
		return
	}

	out := make([]*Mongod, len(slave.Mongods))
	for i, v := range slave.Mongods {
		mongod, err := mongodToApiMongod(tx, v)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, err.Error())
			return
		} else {
			out[i] = mongod
		}
	}
	json.NewEncoder(w).Encode(out)
}

func (m *MasterAPI) MongodsByReplicaSet(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["replicasetId"]
	slaveId, err := strconv.ParseInt(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tx := m.DB.Begin()
	defer tx.Rollback()

	var replicaSet model.ReplicaSet
	getReplicaSetRes := tx.First(&replicaSet, slaveId)
	if getReplicaSetRes.RecordNotFound() {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err := getReplicaSetRes.Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, err.Error())
		return
	}

	if err := tx.Model(&replicaSet).Order("id", false).Related(&replicaSet.Mongods, "Mongods").Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, err.Error())
		return
	}

	out := make([]*Mongod, len(replicaSet.Mongods))
	for i, v := range replicaSet.Mongods {
		mongod, err := mongodToApiMongod(tx, v)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, err.Error())
			return
		} else {
			out[i] = mongod
		}
	}
	json.NewEncoder(w).Encode(out)
}
