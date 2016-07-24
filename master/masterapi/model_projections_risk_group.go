package masterapi

import "github.com/KIT-MAMID/mamid/model"

func ProjectModelRiskGroupToRiskGroup(m *model.RiskGroup) *RiskGroup {
	return &RiskGroup{
		ID:   m.ID,
		Name: m.Name,
	}
}

func ProjectRiskGroupToModelRiskGroup(r RiskGroup) *model.RiskGroup {
	return &model.RiskGroup{
		ID:   r.ID,
		Name: r.Name,
	}
}
