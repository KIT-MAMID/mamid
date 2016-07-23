package masterapi

import (
	"encoding/json"
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/gorilla/mux"
	"github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"strconv"
)

type ReplicaSet struct {
	ID                              uint   `json:"id"`
	Name                            string `json:"name"`
	PersistentNodeCount             uint   `json:"persistent_node_count"`
	VolatileNodeCount               uint   `json:"volatile_node_count"`
	ConfigureAsShardingConfigServer bool   `json:"configure_as_sharding_config_server"`
}

func (m *MasterAPI) ReplicaSetIndex(w http.ResponseWriter, r *http.Request) {
	var replicasets []*model.ReplicaSet
	err := m.DB.Order("id", false).Find(&replicasets).Error
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	out := make([]*ReplicaSet, len(replicasets))
	for i, v := range replicasets {
		out[i] = ProjectModelReplicaSetToReplicaSet(v)
	}
	json.NewEncoder(w).Encode(out)
}

func (m *MasterAPI) ReplicaSetById(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["replicasetId"]
	id64, err := strconv.ParseUint(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := uint(id64)

	var replSets []model.ReplicaSet
	err = m.DB.Find(&replSets, &model.Slave{ID: id}).Error

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	if len(replSets) == 0 { // Not found?
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if len(replSets) > 1 {
		log.Printf("inconsistency: multiple slaves for slave.ID = %d found in database", len(replSets))
	}
	json.NewEncoder(w).Encode(ProjectModelReplicaSetToReplicaSet(&replSets[0]))
	return
}

func (m *MasterAPI) ReplicaSetPut(w http.ResponseWriter, r *http.Request) {
	var postReplSet ReplicaSet
	err := json.NewDecoder(r.Body).Decode(&postReplSet)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "cannot parse object (%s)", err.Error())
		return
	}

	// Validation

	if postReplSet.ID != 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "must not specify the slave ID in PUT request")
		return
	}

	modelReplSet := ProjectReplicaSetToModelReplicaSet(&postReplSet)

	// Persist to database

	err = m.DB.Create(&modelReplSet).Error

	//Check db specific errors
	if driverErr, ok := err.(sqlite3.Error); ok {
		if driverErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, driverErr.Error())
			return
		}
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	// Return created slave

	json.NewEncoder(w).Encode(ProjectModelReplicaSetToReplicaSet(modelReplSet))

	return
}

func (m *MasterAPI) ReplicaSetUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["replicasetId"]
	id64, err := strconv.ParseUint(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := uint(id64)

	var postReplSet ReplicaSet
	err = json.NewDecoder(r.Body).Decode(&postReplSet)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "cannot parse object (%s)", err.Error())
		return
	}

	// Validation

	if postReplSet.ID != id {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "must not change the id of an object")
		return
	}

	var modelReplSet model.ReplicaSet
	dbRes := m.DB.First(&modelReplSet, id)
	if dbRes.RecordNotFound() {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err = dbRes.Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	replSet := ProjectReplicaSetToModelReplicaSet(&postReplSet)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err)
		return
	}

	if replSet.ConfigureAsShardingConfigServer != modelReplSet.ConfigureAsShardingConfigServer ||
		replSet.Name != modelReplSet.Name {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "name and configure_as_sharding_server may not be changed")
		return
	}

	// Persist to database

	m.DB.Model(&modelReplSet).Updates(replSet)

	//Check db specific errors
	if driverErr, ok := err.(sqlite3.Error); ok {
		if driverErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, driverErr.Error())
			return
		}
	}
}
