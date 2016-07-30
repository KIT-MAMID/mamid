package masterapi

import (
	"encoding/json"
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"strconv"
)

type Slave struct {
	ID                   uint   `json:"id"`
	Hostname             string `json:"hostname"`
	Port                 uint   `json:"slave_port"`
	MongodPortRangeBegin uint   `json:"mongod_port_range_begin"` //inclusive
	MongodPortRangeEnd   uint   `json:"mongod_port_range_end"`   //exclusive
	PersistentStorage    bool   `json:"persistent_storage"`
	ConfiguredState      string `json:"configured_state"`
	RiskGroupID          uint   `json:"risk_group_id"`
}

func (m *MasterAPI) SlaveIndex(w http.ResponseWriter, r *http.Request) {

	var slaves []*model.Slave
	err := m.DB.Order("id", false).Find(&slaves).Error
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	out := make([]*Slave, len(slaves))
	for i, v := range slaves {
		out[i] = ProjectModelSlaveToSlave(v)
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

	var slaves []model.Slave
	err = m.DB.Find(&slaves, &model.Slave{ID: id}).Error

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	if len(slaves) == 0 { // Not found?
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if len(slaves) > 1 {
		log.Printf("inconsistency: multiple slaves for slave.ID = %d found in database", len(slaves))
	}
	json.NewEncoder(w).Encode(ProjectModelSlaveToSlave(&slaves[0]))
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

	err = m.DB.Create(&modelSlave).Error

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

	json.NewEncoder(w).Encode(ProjectModelSlaveToSlave(modelSlave))

	return
}

func (m *MasterAPI) SlaveUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["slaveId"]
	id64, err := strconv.ParseUint(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := uint(id64)

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

	var modelSlave model.Slave
	modelSlaveRes := m.DB.First(&modelSlave, id)
	if modelSlaveRes.RecordNotFound() {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err = modelSlaveRes.Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	updatedModelSlave, err := ProjectSlaveToModelSlave(&postSlave)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err)
		return
	}

	// Only allow changes to both observed and desired disabled slaves

	permissionError, dbError := changeToSlaveAllowed(m.DB, &modelSlave, updatedModelSlave)
	if dbError != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, dbError)
		return
	}
	if permissionError != nil {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, permissionError)
		return
	}

	// Persist to database

	m.DB.Save(&updatedModelSlave)

	//Check db specific errors
	if driverErr, ok := err.(sqlite3.Error); ok {
		if driverErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, driverErr.Error())
			return
		}
	}
}

func (m *MasterAPI) SlaveDelete(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["slaveId"]
	id64, err := strconv.ParseUint(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := uint(id64)

	// Can only delete disabled slaves
	var currentSlave model.Slave
	if err = m.DB.First(&currentSlave, id).Related(&currentSlave.Mongods, "Mongods").Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	if len(currentSlave.Mongods) != 0 {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "slave with id %d has active Mongods", currentSlave.ID)
		return
	}

	// Allow delete

	s := m.DB.Delete(&model.Slave{ID: id})
	if s.Error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	if s.RowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
	}

	if s.RowsAffected > 1 {
		log.Printf("inconsistency: slave DELETE affected more than one row. Slave.ID = %v", id)
	}
}

func changeToSlaveAllowed(db *gorm.DB, currentSlave *model.Slave, updatedSlave *model.Slave) (permissionError, dbError error) {

	//Allow change of state if nothing else is changed
	if currentSlave.ID == updatedSlave.ID &&
		currentSlave.Hostname == updatedSlave.Hostname &&
		currentSlave.Port == updatedSlave.Port &&
		currentSlave.MongodPortRangeBegin == updatedSlave.MongodPortRangeBegin &&
		currentSlave.MongodPortRangeEnd == updatedSlave.MongodPortRangeEnd &&
		currentSlave.PersistentStorage == updatedSlave.PersistentStorage &&
		currentSlave.RiskGroupID == updatedSlave.RiskGroupID {
		return nil, nil
	}
	if currentSlave.ConfiguredState != model.SlaveStateDisabled {
		return fmt.Errorf("slave's desired state must be = disabled"), nil
	}

	if err := db.Model(&currentSlave).Related(&currentSlave.Mongods, "Mongods").Error; err != nil {
		return nil, err
	}

	if len(currentSlave.Mongods) != 0 {
		return fmt.Errorf("slave has active Mongods"), nil
	}

	return nil, nil

}
