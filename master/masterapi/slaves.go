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

type Slave struct {
	ID                           int64  `json:"id"`
	Hostname                     string `json:"hostname"`
	Port                         uint   `json:"slave_port"`
	MongodPortRangeBegin         uint   `json:"mongod_port_range_begin"` //inclusive
	MongodPortRangeEnd           uint   `json:"mongod_port_range_end"`   //exclusive
	PersistentStorage            bool   `json:"persistent_storage"`
	ConfiguredState              string `json:"configured_state"`
	ConfiguredStateTransitioning bool   `json:"configured_state_transitioning"`
	RiskGroupID                  *int64 `json:"risk_group_id"`
}

func (m *MasterAPI) SlaveIndex(w http.ResponseWriter, r *http.Request) {
	tx := m.DB.Begin()
	defer tx.Rollback()

	var slaves []*model.Slave
	err := tx.Order("id", false).Find(&slaves).Error
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	out := make([]*Slave, len(slaves))
	for i, v := range slaves {
		out[i], err = ProjectModelSlaveToSlave(tx, v)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "cannot project model slave to slave: %s", err)
			return
		}
	}

	json.NewEncoder(w).Encode(out)
}

func (m *MasterAPI) SlaveById(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["slaveId"]
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

	var slave model.Slave
	res := tx.First(&slave, id)

	if res.RecordNotFound() {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err = res.Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	apiSlave, err := ProjectModelSlaveToSlave(tx, &slave)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "cannot project model slave to slave: %s", err)
		return
	}

	json.NewEncoder(w).Encode(apiSlave)

	return
}

func (m *MasterAPI) SlavePut(w http.ResponseWriter, r *http.Request) {
	var postSlave Slave
	err := json.NewDecoder(r.Body).Decode(&postSlave)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "cannot parse object (%s)", err.Error())
		return
	}

	// Validation

	if postSlave.ID != 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "must not specify the slave ID in PUT request")
		return
	}

	modelSlave, err := ProjectSlaveToModelSlave(&postSlave)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}

	// Persist to database

	tx := m.DB.Begin()

	err = tx.Create(&modelSlave).Error

	//Check db specific errors
	if model.IsIntegrityConstraintViolation(err) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		tx.Rollback()
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		tx.Rollback()
		return
	}

	// Trigger cluster allocator
	if err = m.attemptClusterAllocator(tx, w); err != nil {
		return
	}

	// Return created slave
	apiSlave, err := ProjectModelSlaveToSlave(tx, modelSlave)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "cannot project model slave to slave: %s", err)
		tx.Rollback()
		return
	}

	json.NewEncoder(w).Encode(apiSlave)

	tx.Commit()

	return
}

func (m *MasterAPI) SlaveUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["slaveId"]
	id, err := strconv.ParseInt(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var postSlave Slave
	err = json.NewDecoder(r.Body).Decode(&postSlave)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "cannot parse object (%s)", err.Error())
		return
	}

	// Validation

	if postSlave.ID != id {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "must not change the id of an object")
		return
	}

	if err = postSlave.assertNoZeroFieldsSet(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "must not POST JSON with zero values in any field: %s", err.Error())
		return
	}

	// Only allow changes to both observed and desired disabled slaves

	tx := m.DB.Begin()

	var modelSlave model.Slave
	modelSlaveRes := tx.First(&modelSlave, id)
	if modelSlaveRes.RecordNotFound() {
		w.WriteHeader(http.StatusNotFound)
		tx.Rollback()
		return
	} else if err = modelSlaveRes.Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		tx.Rollback()
		return
	}

	updatedModelSlave, err := ProjectSlaveToModelSlave(&postSlave)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err)
		tx.Rollback()
		return
	}

	// Only allow changes to both observed and desired disabled slaves

	permissionError, dbError := changeToSlaveAllowed(tx, &modelSlave, updatedModelSlave)
	if dbError != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, dbError)
		tx.Rollback()
		return
	}
	if permissionError != nil {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, permissionError)
		tx.Rollback()
		return
	}

	// Persist to database
	err = tx.Save(&updatedModelSlave).Error

	//Check db specific errors
	if model.IsIntegrityConstraintViolation(err) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		tx.Rollback()
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		tx.Rollback()
		return
	}

	// Trigger cluster allocator
	if err = m.attemptClusterAllocator(tx, w); err != nil {
		return
	}

	tx.Commit()

}

func (m *MasterAPI) SlaveDelete(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["slaveId"]
	id, err := strconv.ParseInt(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tx := m.DB.Begin()

	// Can only delete disabled slaves
	var currentSlave model.Slave
	if err = tx.First(&currentSlave, id).Related(&currentSlave.Mongods, "Mongods").Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		tx.Rollback()
		return
	}

	if len(currentSlave.Mongods) != 0 {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "slave with id %d has active Mongods", currentSlave.ID)
		tx.Rollback()
		return
	}

	// Allow delete

	s := tx.Delete(&model.Slave{ID: id})
	if s.Error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, s.Error.Error())
		tx.Rollback()
		return
	}

	if s.RowsAffected > 1 {
		masterapiLog.Errorf("inconsistency: slave DELETE affected more than one row. Slave.ID = %v", id)
	}

	if s.RowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		tx.Rollback()
		return
	}

	// Trigger cluster allocator
	if err = m.attemptClusterAllocator(tx, w); err != nil {
		return
	}

	tx.Commit()
}

func changeToSlaveAllowed(tx *gorm.DB, currentSlave *model.Slave, updatedSlave *model.Slave) (permissionError, dbError error) {

	// Allow change of state if nothing else is changed
	// NOTE: changing the slave state is an indication to the ClusterAllocator but has no direct consequences in deployment
	if currentSlave.ID == updatedSlave.ID &&
		currentSlave.Hostname == updatedSlave.Hostname &&
		currentSlave.Port == updatedSlave.Port &&
		currentSlave.MongodPortRangeBegin == updatedSlave.MongodPortRangeBegin &&
		currentSlave.MongodPortRangeEnd == updatedSlave.MongodPortRangeEnd &&
		currentSlave.PersistentStorage == updatedSlave.PersistentStorage &&
		currentSlave.RiskGroupID == updatedSlave.RiskGroupID {
		return nil, nil
	}
	if currentSlave.ConfiguredState != model.SlaveStateDisabled && currentSlave.ConfiguredState != model.SlaveStateMaintenance {
		return fmt.Errorf("slave's desired state must be `maintenance` or `disabled`"), nil
	}

	if err := tx.Model(&currentSlave).Related(&currentSlave.Mongods, "Mongods").Error; err != nil {
		return nil, err
	}

	if len(currentSlave.Mongods) != 0 {
		//The port range of a slave with mongods may not be reduced because the mongods may be using the ports
		if updatedSlave.MongodPortRangeBegin > currentSlave.MongodPortRangeBegin || updatedSlave.MongodPortRangeEnd < currentSlave.MongodPortRangeEnd {
			return fmt.Errorf("Cannot reduce port range to a subinterval while this Slave has Mongods"), nil
		}

		//The persistence of slaves with mongods may not be changed because the mongods depend on the persistence
		if updatedSlave.PersistentStorage != currentSlave.PersistentStorage {
			return fmt.Errorf("Cannot change persistence attribute while this Slave has Mongods"), nil
		}
	}

	return nil, nil

}
