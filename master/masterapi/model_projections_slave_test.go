package masterapi

import (
	"github.com/KIT-MAMID/mamid/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_ProjectModelSlaveToSlave(t *testing.T) {

	dbSlave := model.Slave{
		Hostname:             "host1",
		Port:                 1,
		MongodPortRangeBegin: 2,
		MongodPortRangeEnd:   3,
		PersistentStorage:    true,
		Mongods:              []*model.Mongod{},
		ConfiguredState:      model.SlaveStateActive,
	}

	s, err := ProjectModelSlaveToSlave(nil, &dbSlave) // we can get away without DB because ConfiguredState = Active
	assert.Nil(t, err)

	assert.EqualValues(t, dbSlave.Hostname, s.Hostname)
	assert.EqualValues(t, dbSlave.Port, s.Port)
	assert.EqualValues(t, dbSlave.MongodPortRangeBegin, s.MongodPortRangeBegin)
	assert.EqualValues(t, dbSlave.MongodPortRangeEnd, s.MongodPortRangeEnd)
	assert.EqualValues(t, dbSlave.PersistentStorage, s.PersistentStorage)

}
