package masterapi

import (
	"encoding/json"
	"github.com/KIT-MAMID/mamid/master"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"fmt"
)

func createDBAndMasterAPI(t *testing.T) (db *gorm.DB, mainRouter *mux.Router, err error) {
	// Setup database
	db, err = model.InitializeInMemoryDB("")
	dbSlave := model.Slave{
		Hostname:             "host1",
		Port:                 1,
		MongodPortRangeBegin: 2,
		MongodPortRangeEnd:   3,
		PersistentStorage:    true,
		Mongods:              []*model.Mongod{},
		ConfiguredState:      model.SlaveStateActive,
	}
	assert.NoError(t, db.Create(&dbSlave).Error)

	dbSlave2 := model.Slave{
		Hostname:             "host2",
		Port:                 1,
		MongodPortRangeBegin: 100,
		MongodPortRangeEnd:   200,
		PersistentStorage:    false,
		Mongods:              []*model.Mongod{},
		ConfiguredState:      model.SlaveStateDisabled,
	}
	assert.NoError(t, db.Create(&dbSlave2).Error)

	// Setup masterapi
	clusterAllocator := &master.ClusterAllocator{}

	mainRouter = mux.NewRouter().StrictSlash(true)
	masterAPI := &MasterAPI{
		DB:               db,
		ClusterAllocator: clusterAllocator,
		Router:           mainRouter.PathPrefix("/api/").Subrouter(),
	}
	masterAPI.Setup()

	return
}

func TestMasterAPI_SlaveIndex(t *testing.T) {
	_, mainRouter, err := createDBAndMasterAPI(t)
	assert.NoError(t, err)

	// Test correct get
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/slaves", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	var getSlaveResult []Slave
	err = json.NewDecoder(resp.Body).Decode(&getSlaveResult)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(getSlaveResult))
	assert.Equal(t, "host1", getSlaveResult[0].Hostname)
	assert.EqualValues(t, 1, getSlaveResult[0].Port)
	assert.EqualValues(t, 2, getSlaveResult[0].MongodPortRangeBegin)
	assert.EqualValues(t, 3, getSlaveResult[0].MongodPortRangeEnd)
	assert.Equal(t, true, getSlaveResult[0].PersistentStorage)
	assert.Equal(t, "active", getSlaveResult[0].ConfiguredState)
}

func TestMasterAPI_SlaveById(t *testing.T) {
	_, mainRouter, err := createDBAndMasterAPI(t)
	assert.NoError(t, err)

	// Test correct get
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/slaves/1", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	var getSlaveResult Slave
	err = json.NewDecoder(resp.Body).Decode(&getSlaveResult)
	assert.NoError(t, err)

	assert.Equal(t, "host1", getSlaveResult.Hostname)
	assert.EqualValues(t, 1, getSlaveResult.Port)
	assert.EqualValues(t, 2, getSlaveResult.MongodPortRangeBegin)
	assert.EqualValues(t, 3, getSlaveResult.MongodPortRangeEnd)
	assert.Equal(t, true, getSlaveResult.PersistentStorage)
	assert.Equal(t, "active", getSlaveResult.ConfiguredState)
}

func TestMasterAPI_SlavePut(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	assert.NoError(t, err)

	//Test correct put
	resp := httptest.NewRecorder()

	req_body := "{\"id\":0,\"hostname\":\"createdhost\",\"slave_port\":1912,\"mongod_port_range_begin\":20000,\"mongod_port_range_end\":20001,\"persistent_storage\":false,\"configured_state\":\"disabled\"}"
	req, err := http.NewRequest("PUT", "/api/slaves", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	if !assert.Equal(t, 200, resp.Code) {
		fmt.Println(resp.Body.String())
	}

	var createdSlave model.Slave
	db.First(&createdSlave, "hostname = ?", "createdhost")

	//Check created database entry
	assert.NotEmpty(t, createdSlave.ID)
	assert.Equal(t, "createdhost", createdSlave.Hostname)
	assert.EqualValues(t, 1912, createdSlave.Port)
	assert.EqualValues(t, 20000, createdSlave.MongodPortRangeBegin)
	assert.EqualValues(t, 20001, createdSlave.MongodPortRangeEnd)
	assert.Equal(t, false, createdSlave.PersistentStorage)
	assert.Equal(t, model.SlaveStateDisabled, createdSlave.ConfiguredState)

	//Check returned object
	var getSlaveResult Slave
	err = json.NewDecoder(resp.Body).Decode(&getSlaveResult)
	assert.NoError(t, err)

	assert.NotEmpty(t, getSlaveResult.ID)
	assert.Equal(t, "createdhost", getSlaveResult.Hostname)
}


func TestMasterAPI_SlavePut_existing_hostname(t *testing.T) {
	_, mainRouter, err := createDBAndMasterAPI(t)
	assert.NoError(t, err)

	//Test invalid put (slave with same hostname exists)
	resp := httptest.NewRecorder()

	req_body := "{\"id\":0,\"hostname\":\"host1\",\"slave_port\":1912,\"mongod_port_range_begin\":20000,\"mongod_port_range_end\":20001,\"persistent_storage\":false,\"configured_state\":\"disabled\"}"
	req, err := http.NewRequest("PUT", "/api/slaves", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code)
}

func TestMasterAPI_SlavePut_additionalUnknownField(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	assert.NoError(t, err)
	//Test invalid put (non existing field)
	resp := httptest.NewRecorder()

	// additional unknown field:  id_invalid_blabla
	req_body := "{\"hostname\":\"createdhost\",\"slave_port\":1912,\"mongod_port_range_begin\":20000,\"mongod_port_range_end\":20002,\"persistent_storage\":false,\"configured_state\":\"disabled\"}"
	req, err := http.NewRequest("PUT", "/api/slaves", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)
	var invalidSlave model.Slave
	assert.NoError(t, db.First(&invalidSlave, "hostname = ?", "createdhost").Error)
}

func TestMasterAPI_SlavePut_missingField(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	assert.NoError(t, err)
	//Test invalid put (non existing field)
	resp := httptest.NewRecorder()

	// additional unknown field:  id_invalid_blabla
	// missing field: mongod_port_range_begin
	req_body := "{\"hostname\":\"createdhost_invalid\",\"slave_port\":1912,\"mongod_port_range_begin\":20000,\"persistent_storage\":false,\"configured_state\":\"disabled\"}"
	req, err := http.NewRequest("PUT", "/api/slaves", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code)

	var invalidSlave model.Slave
	assert.Error(t, db.First(&invalidSlave, "hostname = ?", "createdhost_invalid").Error)
}

func TestMasterAPI_SlaveUpdate(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	assert.NoError(t, err)

	//Test valid update
	resp := httptest.NewRecorder()

	req_body := "{\"id\":2,\"hostname\":\"updHost\",\"slave_port\":2,\"mongod_port_range_begin\":101,\"mongod_port_range_end\":201,\"persistent_storage\":true,\"configured_state\":\"disabled\"}"
	req, err := http.NewRequest("POST", "/api/slaves/2", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	var updatedSlave model.Slave
	db.First(&updatedSlave, 2)

	assert.Equal(t, "updHost", updatedSlave.Hostname)
	assert.EqualValues(t, 2, updatedSlave.Port)
	assert.EqualValues(t, 101, updatedSlave.MongodPortRangeBegin)
	assert.EqualValues(t, 201, updatedSlave.MongodPortRangeEnd)
	assert.Equal(t, true, updatedSlave.PersistentStorage)
	assert.Equal(t, model.SlaveStateDisabled, updatedSlave.ConfiguredState)
}

func TestMasterAPI_SlaveUpdate_invalid(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	assert.NoError(t, err)

	//Test invalid update (slave is in active state)
	resp := httptest.NewRecorder()

	req_body := "{\"id\":1,\"hostname\":\"updHost\",\"slave_port\":1912,\"mongod_port_range_begin\":20000,\"mongod_port_range_end\":20001,\"persistent_storage\":false,\"configured_state\":\"active\"}"
	req, err := http.NewRequest("POST", "/api/slaves/1", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code)

	var updatedSlave model.Slave
	db.First(&updatedSlave, 1)

	assert.Equal(t, "host1", updatedSlave.Hostname)
	assert.EqualValues(t, 1, updatedSlave.Port)
	assert.EqualValues(t, 2, updatedSlave.MongodPortRangeBegin)
	assert.EqualValues(t, 3, updatedSlave.MongodPortRangeEnd)
	assert.Equal(t, true, updatedSlave.PersistentStorage)
	assert.Equal(t, model.SlaveStateActive, updatedSlave.ConfiguredState)
}

func TestMasterAPI_SlaveUpdate_change_desired_state(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	assert.NoError(t, err)

	//Test valid state change
	resp := httptest.NewRecorder()

	req_body := "{\"id\":2,\"hostname\":\"host2\",\"slave_port\":1,\"mongod_port_range_begin\":100,\"mongod_port_range_end\":200,\"persistent_storage\":false,\"configured_state\":\"active\"}"
	req, err := http.NewRequest("POST", "/api/slaves/2", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	var updatedSlave model.Slave
	db.First(&updatedSlave, 2)

	assert.Equal(t, model.SlaveStateActive, updatedSlave.ConfiguredState)
}

func TestMasterAPI_SlaveUpdate_change_desired_state_invalid(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	assert.NoError(t, err)

	//Test invalid state change (should not be able to change state while changing another parameter)
	resp := httptest.NewRecorder()

	req_body := "{\"id\":2,\"hostname\":\"host2\",\"slave_port\":1,\"mongod_port_range_begin\":100,\"mongod_port_range_end\":150,\"persistent_storage\":false,\"configured_state\":\"active\"}"
	req, err := http.NewRequest("POST", "/api/slaves/2", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code)

	var updatedSlave model.Slave
	db.First(&updatedSlave, 2)

	assert.Equal(t, model.SlaveStateDisabled, updatedSlave.ConfiguredState)
}

func TestMasterAPI_SlaveDelete(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	assert.NoError(t, err)

	//Test valid delete
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("DELETE", "/api/slaves/2", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	var updatedSlave model.Slave
	db.First(&updatedSlave, 2)

	assert.Empty(t, updatedSlave.ID)
}

func TestMasterAPI_SlaveDelete_invalid(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	assert.NoError(t, err)

	//Test invalid delete (active slave)
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("DELETE", "/api/slaves/1", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code)

	var updatedSlave model.Slave
	db.First(&updatedSlave, 1)

	assert.NotEmpty(t, updatedSlave.ID)
}
