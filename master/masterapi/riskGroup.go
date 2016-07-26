package masterapi

import (
	"encoding/json"
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/gorilla/mux"
	"github.com/mattn/go-sqlite3"
	"net/http"
	"strconv"
)

type RiskGroup struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

func (m *MasterAPI) RiskGroupIndex(w http.ResponseWriter, r *http.Request) {
	var riskGroups []*model.RiskGroup
	err := m.DB.Order("id", false).Find(&riskGroups).Error
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

	var riskgroups []model.RiskGroup
	err = m.DB.Find(&riskgroups, &model.RiskGroup{ID: id}).Error

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	if len(riskgroups) == 0 { // Not found?
		w.WriteHeader(http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(ProjectModelRiskGroupToRiskGroup(&riskgroups[0]))
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

	modelRiskGroup := ProjectRiskGroupToModelRiskGroup(&postRiskGroup)

	// Persist to database

	err = m.DB.Create(&modelRiskGroup).Error

	//Check db specific errors
	if driverErr, ok := err.(sqlite3.Error); ok {
		if driverErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, driverErr.Error())
			return
		}
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	// Return created risk group

	json.NewEncoder(w).Encode(ProjectModelRiskGroupToRiskGroup(modelRiskGroup))

	return
}

func (m *MasterAPI) RiskGroupUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["riskgroupId"]
	id64, err := strconv.ParseUint(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := uint(id64)

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

	var modelRiskGroup model.RiskGroup
	findRes := m.DB.First(&modelRiskGroup, id)
	if findRes.RecordNotFound() {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err = findRes.Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	// Allow update

	save := ProjectRiskGroupToModelRiskGroup(&postRiskGroup)

	// Persist to database

	m.DB.Model(&modelRiskGroup).Updates(&save)

	// Check db specific errors
	if driverErr, ok := err.(sqlite3.Error); ok {
		if driverErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, driverErr.Error())
			return
		}
	}
}

func (m *MasterAPI) RiskGroupDelete(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["riskgroupId"]
	id64, err := strconv.ParseUint(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := uint(id64)

	// Can only delete risk groups without slaves
	var currentRiskGroup model.RiskGroup
	findRes := m.DB.First(&currentRiskGroup, id)
	if findRes.RecordNotFound() {
		w.WriteHeader(http.StatusNotFound)
	} else if err = findRes.Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	if err = m.DB.First(&currentRiskGroup, id).Related(&currentRiskGroup.Slaves, "Slaves").Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	if len(currentRiskGroup.Slaves) != 0 {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "riskgroup with id %d has assigned slaves", currentRiskGroup.ID)
		return
	}

	// Allow delete

	s := m.DB.Delete(&model.RiskGroup{ID: id})
	if s.Error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	if s.RowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
	}
}
