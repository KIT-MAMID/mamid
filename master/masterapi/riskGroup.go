package masterapi

import (
	"encoding/json"
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"net/http"
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
