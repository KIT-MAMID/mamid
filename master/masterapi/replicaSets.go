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

type ReplicaSet struct {
	ID                              int64  `json:"id"`
	Name                            string `json:"name"`
	PersistentNodeCount             uint   `json:"persistent_node_count"`
	VolatileNodeCount               uint   `json:"volatile_node_count"`
	ConfigureAsShardingConfigServer bool   `json:"configure_as_sharding_config_server"`
}

func (m *MasterAPI) ReplicaSetIndex(w http.ResponseWriter, r *http.Request) {
	tx := m.DB.Begin()
	defer tx.Rollback()

	var replicasets []*model.ReplicaSet
	err := tx.Order("id", false).Find(&replicasets).Error
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

	if id == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "id may not be 0")
		return
	}

	tx := m.DB.Begin()
	defer tx.Rollback()

	var replSet model.ReplicaSet
	res := tx.First(&replSet, id)

	if res.RecordNotFound() {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err = res.Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	json.NewEncoder(w).Encode(ProjectModelReplicaSetToReplicaSet(&replSet))
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

	modelReplSet, err := ProjectReplicaSetToModelReplicaSet(&postReplSet)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, err.Error())
		return
	}

	tx := m.DB.Begin()

	// Validation
	if allowed, msg, err := changeToReplicaSetAllowed(tx, nil, modelReplSet); !allowed || err != nil {
		tx.Rollback()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "error validating update permission: %s", err)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, msg)
		}
		return
	}

	// Persist to database

	err = tx.Create(&modelReplSet).Error

	//Check db specific errors
	if model.IsIntegrityConstraintViolation(err) {
		tx.Rollback()
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	} else if err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	// Trigger cluster allocator
	if err = m.attemptClusterAllocator(tx, w); err != nil {
		return
	}

	tx.Commit()

	// Return created slave

	json.NewEncoder(w).Encode(ProjectModelReplicaSetToReplicaSet(modelReplSet))

	return
}

func (m *MasterAPI) ReplicaSetUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["replicasetId"]
	id, err := strconv.ParseInt(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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

	tx := m.DB.Begin()

	var modelReplSet model.ReplicaSet

	dbRes := tx.First(&modelReplSet, id)

	if dbRes.RecordNotFound() {
		tx.Rollback()
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err = dbRes.Error; err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	replSet, err := ProjectReplicaSetToModelReplicaSet(&postReplSet)
	if err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err)
		return
	}

	if allowed, msg, err := changeToReplicaSetAllowed(tx, &modelReplSet, replSet); !allowed || err != nil {
		tx.Rollback()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "error validating update permission: %s", err)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, msg)
		}
		return
	}

	// Persist to database

	err = tx.Save(replSet).Error

	//Check db specific errors
	if model.IsIntegrityConstraintViolation(err) {
		tx.Rollback()
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}

	// Trigger cluster allocator
	if err = m.attemptClusterAllocator(tx, w); err != nil {
		return
	}

	tx.Commit()
}

func (m *MasterAPI) ReplicaSetDelete(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["replicasetId"]
	id, err := strconv.ParseInt(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Allow delete

	tx := m.DB.Begin()

	s := tx.Delete(&model.ReplicaSet{ID: id})

	if s.RowsAffected == 0 {
		tx.Rollback()
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if s.Error != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	if s.RowsAffected > 1 {
		masterapiLog.Errorf("inconsistency: slave DELETE affected more than one row. Slave.ID = %v", id)
	}

	// Trigger cluster allocator
	// TODO having removed the replica set, the cluster allocator should mark the
	// affected mongod's desired state as deleted
	// check issue #9
	if err = m.attemptClusterAllocator(tx, w); err != nil {
		return
	}

	tx.Commit()

}

func (m *MasterAPI) ReplicaSetGetSlaves(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["replicasetId"]
	id64, err := strconv.ParseUint(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := uint(id64)

	if id == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "id may not be 0")
		return
	}

	tx := m.DB.Begin()
	defer tx.Rollback()

	var replSet model.ReplicaSet
	res := tx.First(&replSet, id)

	if res.RecordNotFound() {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err = res.Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	var slaves []*model.Slave
	res = tx.Raw("SELECT s.* FROM slaves s JOIN mongods m ON m.parent_slave_id = s.id WHERE m.replica_set_id = ?", id).Scan(&slaves)
	if err = res.Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	out := make([]*Slave, len(slaves))
	for i, v := range slaves {
		out[i], err = ProjectModelSlaveToSlave(tx, v)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}
	}
	json.NewEncoder(w).Encode(out)
	return
}

// Validate attributes of a replica set.
// `current` may be nil if there is no current replica set
func changeToReplicaSetAllowed(tx *gorm.DB, current *model.ReplicaSet, new *model.ReplicaSet) (allowed bool, msg string, err error) {

	if current != nil {

		if current.ConfigureAsShardingConfigServer != new.ConfigureAsShardingConfigServer {
			return false, "cannot change sharding config server role of a replica set after creation", nil
		}

		if current.Name != new.Name {
			return false, "cannot change name of a replica set after creation", nil
		}

	}

	if (new.VolatileMemberCount+new.PersistentMemberCount)%2 == 0 {
		// TODO remove this once deployment of arbiters is enabled
		// TODO the above sum assumes all members are eligible to vote. This may not be true, because #voting_members must be < 7
		// 	https://docs.mongodb.com/manual/core/replica-set-architectures/
		return false, "sum of persistent and volatile member counts must be odd", nil
	}

	return true, "", nil

}
