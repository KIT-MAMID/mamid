package masterapi

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
)

func ProjectModelRiskGroupToRiskGroup(m *model.RiskGroup) *RiskGroup {
	return &RiskGroup{
		ID:   m.ID,
		Name: m.Name,
	}
}

func ProjectRiskGroupToModelRiskGroup(r *RiskGroup) (*model.RiskGroup, error) {
	if r.Name == "" {
		return nil, fmt.Errorf("Risk group name may not be empty")
	}
	return &model.RiskGroup{
		ID:   r.ID,
		Name: r.Name,
	}, nil
}
