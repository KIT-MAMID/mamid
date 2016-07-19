package masterapi

import (
	"encoding/json"
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/gorilla/mux"
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
	ConfiguredState      string `json:"state"`
}

func (m *MasterAPI) SlaveIndex(w http.ResponseWriter, r *http.Request) {

	var slaves []model.Slave
	err := m.DB.Find(&slaves).Error
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	json.NewEncoder(w).Encode(slaves)
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
		fmt.Fprintf(w, "must not change the slave ID in PUT request")
		return
	}

	modelSlave, err := ProjectSlaveToModelSlave(&postSlave)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}

	// Persist to database

	err = m.DB.Create(modelSlave).Error
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	// TODO set location header. Would it be better to return the ID? YES.

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

	modelSlave, err := ProjectSlaveToModelSlave(&postSlave)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err)
		return
	}

	// Persist to database

	m.DB.Model(&modelSlave).Updates(&modelSlave)
}

func (m *MasterAPI) SlaveDelete(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["slaveId"]
	id64, err := strconv.ParseUint(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := uint(id64)

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
