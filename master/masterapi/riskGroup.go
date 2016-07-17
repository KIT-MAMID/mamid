package masterapi

import (
	"net/http"
	"encoding/json"
)

type RiskGroup struct {
	Id uint					`json:"id"`
	Name string				`json:"name"`
}

func RiskGroupIndex(w http.ResponseWriter, r *http.Request) {
	riskGroups := []RiskGroup{
		RiskGroup{Id: 1, Name: "Rack A"},
		RiskGroup{Id: 2, Name: "Rack B"},
	}
	json.NewEncoder(w).Encode(riskGroups)
}