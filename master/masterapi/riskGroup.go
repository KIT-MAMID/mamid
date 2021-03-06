package masterapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

type RiskGroup struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func (m *MasterAPI) RiskGroupIndex(w http.ResponseWriter, r *http.Request) {
	tx := m.DB.Begin()
	defer tx.Rollback()

	var riskGroups []*model.RiskGroup
	err := tx.Order("id", false).Find(&riskGroups).Error
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	out := make([]*RiskGroup, len(riskGroups))
	for i, v := range riskGroups {
		out[i] = ProjectModelRiskGroupToRiskGroup(v)
	}
	json.NewEncoder(w).Encode(out)
}

func (m *MasterAPI) RiskGroupById(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["riskgroupId"]
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

	var riskgroup model.RiskGroup
	res := tx.First(&riskgroup, id)

	if res.RecordNotFound() {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err = res.Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	json.NewEncoder(w).Encode(ProjectModelRiskGroupToRiskGroup(&riskgroup))
	return
}

func (m *MasterAPI) RiskGroupPut(w http.ResponseWriter, r *http.Request) {
	var postRiskGroup RiskGroup
	err := json.NewDecoder(r.Body).Decode(&postRiskGroup)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "cannot parse object (%s)", err.Error())
		return
	}

	// Validation

	if postRiskGroup.ID != 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "must not specify the risk group ID in PUT request")
		return
	}

	modelRiskGroup, err := ProjectRiskGroupToModelRiskGroup(&postRiskGroup)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, err.Error())
		return
	}

	// Persist to database

	tx := m.DB.Begin()

	err = tx.Create(&modelRiskGroup).Error

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

	m.attemptCommit(tx, w)

	// Return created risk group

	json.NewEncoder(w).Encode(ProjectModelRiskGroupToRiskGroup(modelRiskGroup))

	return
}

func (m *MasterAPI) RiskGroupUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["riskgroupId"]
	id, err := strconv.ParseInt(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var postRiskGroup RiskGroup
	err = json.NewDecoder(r.Body).Decode(&postRiskGroup)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "cannot parse object (%s)", err.Error())
		return
	}

	// Validation

	if postRiskGroup.ID != id {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "must not change the id of an object")
		return
	}

	// Check if risk group with id exists

	tx := m.DB.Begin()

	var modelRiskGroup model.RiskGroup
	findRes := tx.First(&modelRiskGroup, id)
	if findRes.RecordNotFound() {
		tx.Rollback()
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err = findRes.Error; err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	// Allow update

	save, err := ProjectRiskGroupToModelRiskGroup(&postRiskGroup)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, err.Error())
		return
	}

	// Persist to database

	err = tx.Save(&save).Error

	// Check db specific errors
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

	m.attemptCommit(tx, w)

}

func (m *MasterAPI) RiskGroupDelete(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["riskgroupId"]
	id, err := strconv.ParseInt(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tx := m.DB.Begin()

	// Can only delete risk groups without slaves
	var currentRiskGroup model.RiskGroup
	findRes := tx.First(&currentRiskGroup, id)
	if findRes.RecordNotFound() {
		w.WriteHeader(http.StatusNotFound)
		tx.Rollback()
		return
	} else if err = findRes.Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		tx.Rollback()
		fmt.Fprint(w, err.Error())
		return
	}

	if err = tx.First(&currentRiskGroup, id).Related(&currentRiskGroup.Slaves, "Slaves").Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		tx.Rollback()
		fmt.Fprint(w, err.Error())
		return
	}

	if len(currentRiskGroup.Slaves) != 0 {
		w.WriteHeader(http.StatusForbidden)
		tx.Rollback()
		fmt.Fprintf(w, "riskgroup with id %d has assigned slaves", currentRiskGroup.ID)
		return
	}

	// Allow delete

	s := tx.Delete(&model.RiskGroup{ID: id})
	if s.Error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		tx.Rollback()
		return
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

	m.attemptCommit(tx, w)

}

func (m *MasterAPI) RiskGroupGetSlaves(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["riskgroupId"]

	id := sql.NullInt64{}

	// Check if risk group exists
	// Special case: id == 0 => Get unassigned slaves
	if idStr != "null" {
		idInt64, err := strconv.ParseInt(idStr, 10, 0)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		id.Valid = true
		id.Int64 = idInt64
	}
	// else id is NULL

	tx := m.DB.Begin()
	defer tx.Rollback()

	// Check if risk group exists
	// Special case: id == 0 => Get unassigned slaves
	if id.Valid {
		var riskgroup model.RiskGroup
		riskgroupRes := tx.First(&riskgroup, id)
		if riskgroupRes.RecordNotFound() {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Riskgroup not found")
			return
		} else if err := riskgroupRes.Error; err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, err.Error())
			return
		}
	}

	var slaves []*model.Slave
	var err error
	if id.Valid { // gorm/gosql does not query for IS NULL automatically
		err = tx.Where("risk_group_id = ?", id).Find(&slaves).Error
	} else {
		err = tx.Where("risk_group_id IS NULL").Find(&slaves).Error
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, err.Error())
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
}

func (m *MasterAPI) RiskGroupAssignSlave(w http.ResponseWriter, r *http.Request) {
	riskgroupIdStr := mux.Vars(r)["riskgroupId"]
	riskgroupId, err := strconv.ParseInt(riskgroupIdStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	slaveIdStr := mux.Vars(r)["slaveId"]
	slaveId, err := strconv.ParseInt(slaveIdStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if riskgroupId == 0 || slaveId == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Id may not be 0")
		return
	}

	tx := m.DB.Begin()

	var riskgroup model.RiskGroup
	riskgroupRes := tx.First(&riskgroup, riskgroupId)
	if riskgroupRes.RecordNotFound() {
		tx.Rollback()
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Riskgroup not found")
		return
	} else if err = riskgroupRes.Error; err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, err.Error())
		return
	}

	// Only allow changes to both observed and desired disabled slaves

	var modelSlave model.Slave
	if err = tx.First(&modelSlave, slaveId).Error; err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	updatedSlave := modelSlave
	updatedSlave.RiskGroupID = model.NullIntValue(riskgroupId)

	permissionError, dbError := changeToSlaveAllowed(tx, &modelSlave, &updatedSlave)
	if dbError != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, dbError)
		return
	}
	if permissionError != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, permissionError)
		return
	}

	// Persist to database

	err = tx.Save(&updatedSlave).Error

	//Check db specific errors
	if model.IsIntegrityConstraintViolation(err) {
		tx.Rollback()
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
	}

	m.attemptCommit(tx, w)
}

func (m *MasterAPI) RiskGroupRemoveSlave(w http.ResponseWriter, r *http.Request) {
	riskgroupIdStr := mux.Vars(r)["riskgroupId"]
	riskgroupId, err := strconv.ParseInt(riskgroupIdStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	slaveIdStr := mux.Vars(r)["slaveId"]
	slaveId, err := strconv.ParseInt(slaveIdStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if riskgroupId == 0 || slaveId == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Id may not be 0")
		return
	}

	tx := m.DB.Begin()

	var riskgroup model.RiskGroup
	riskgroupRes := tx.First(&riskgroup, riskgroupId)
	if riskgroupRes.RecordNotFound() {
		tx.Rollback()
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Riskgroup not found")
		return
	} else if err = riskgroupRes.Error; err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, err.Error())
		return
	}

	// Only allow changes to both observed and desired disabled slaves

	var modelSlave model.Slave
	if err = tx.First(&modelSlave, slaveId).Error; err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	if !modelSlave.RiskGroupID.Valid || modelSlave.RiskGroupID.Int64 != riskgroupId {
		tx.Rollback()
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Slave not found in this riskgroup. (Slave is in other riskgroup)")
		return
	}

	updatedSlave := modelSlave
	updatedSlave.RiskGroupID = model.NullInt()

	permissionError, dbError := changeToSlaveAllowed(tx, &modelSlave, &updatedSlave)
	if dbError != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, dbError)
		return
	}
	if permissionError != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, permissionError)
		return
	}

	// Persist to database

	tx.Model(&modelSlave).Update("RiskGroupID", model.NullInt())

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

	m.attemptCommit(tx, w)
}
