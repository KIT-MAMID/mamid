package masterapi

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
)

func ProjectModelSlaveToSlave(m *model.Slave) *Slave {

	return &Slave{
		ID:                   m.ID,
		Hostname:             m.Hostname,
		Port:                 uint(m.Port),
		MongodPortRangeBegin: uint(m.MongodPortRangeBegin),
		MongodPortRangeEnd:   uint(m.MongodPortRangeEnd),
		PersistentStorage:    m.PersistentStorage,
		ConfiguredState:      SlaveStateToJSONRepresentation(m.ConfiguredState),
		RiskGroupID:          m.RiskGroupID,
	}
}

func ProjectSlaveToModelSlave(s *Slave) (*model.Slave, error) {

	genericErr := fmt.Errorf("Could not map slave representation to internal representation")

	for _, i := range []uint{s.Port, s.MongodPortRangeBegin, s.MongodPortRangeEnd} {
		if err := assertIsPortNumber(i); err != nil {
			return nil, concatErrors(genericErr, err)
		}
	}

	state, stateErr := SlaveJSONRepresentationToStruct(s.ConfiguredState)
	if stateErr != nil {
		return nil, concatErrors(genericErr, stateErr)
	}

	return &model.Slave{
		ID:                   s.ID,
		Hostname:             s.Hostname,
		Port:                 model.PortNumber(s.Port),
		MongodPortRangeBegin: model.PortNumber(s.MongodPortRangeBegin),
		MongodPortRangeEnd:   model.PortNumber(s.MongodPortRangeEnd),
		PersistentStorage:    s.PersistentStorage,
		ConfiguredState:      state,
		RiskGroupID:          s.RiskGroupID,
	}, nil
}

func SlaveJSONRepresentationToStruct(s string) (model.SlaveState, error) {
	switch {
	case s == "active":
		return model.SlaveStateActive, nil
	case s == "disabled":
		return model.SlaveStateDisabled, nil
	case s == "maintenance":
		return model.SlaveStateMaintenance, nil
	default:
		return model.SlaveState(0), fmt.Errorf("cannot convert JSON slave state representation '%s' to internal value", s)
	}
}

func SlaveStateToJSONRepresentation(s model.SlaveState) string {
	switch {
	case s == model.SlaveStateActive:
		return "active"
	case s == model.SlaveStateDisabled:
		return "disabled"
	case s == model.SlaveStateMaintenance:
		return "maintenance"
	default:
		return "undefined"
	}
}
