package masterapi

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"net/http/httptest"
	"github.com/KIT-MAMID/mamid/model"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/KIT-MAMID/mamid/master"
	"encoding/json"
	"github.com/jinzhu/gorm"
	"strings"
)

func createDBAndMasterAPI() (db *gorm.DB, mainRouter *mux.Router, err error) {
	// Setup database
	db, err = model.InitializeInMemoryDB("")
	dbSlave := model.Slave{
		Hostname:             "host1",
		Port:                 1,
		MongodPortRangeBegin: 2,
		MongodPortRangeEnd:   3,
		PersistentStorage:    true,
		Mongods:              []*model.Mongod{},
		ConfiguedState:       model.SlaveStateActive,
	}
	db.Create(&dbSlave)

	dbSlave2 := model.Slave{
		Hostname:             "host2",
		Port:                 1,
		MongodPortRangeBegin: 100,
		MongodPortRangeEnd:   200,
		PersistentStorage:    false,
		Mongods:              []*model.Mongod{},
		ConfiguedState:       model.SlaveStateDisabled,
	}
	db.Create(&dbSlave2)


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
	_, mainRouter, err := createDBAndMasterAPI();
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
	assert.Equal(t, 1, getSlaveResult[0].Port)
	assert.Equal(t, 2, getSlaveResult[0].MongodPortRangeBegin)
	assert.Equal(t, 3, getSlaveResult[0].MongodPortRangeEnd)
	assert.Equal(t, true, getSlaveResult[0].PersistantStorage)
	assert.Equal(t, "active", getSlaveResult[0].State)
}

func TestMasterAPI_SlaveById(t *testing.T) {
	_, mainRouter, err := createDBAndMasterAPI();
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
	assert.Equal(t, 1, getSlaveResult.Port)
	assert.Equal(t, 2, getSlaveResult.MongodPortRangeBegin)
	assert.Equal(t, 3, getSlaveResult.MongodPortRangeEnd)
	assert.Equal(t, true, getSlaveResult.PersistantStorage)
	assert.Equal(t, "active", getSlaveResult.State)
}

func TestMasterAPI_SlavePut(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI();
	assert.NoError(t, err)

	//Test correct put
	resp := httptest.NewRecorder()

	req_body := "{\"id\":0,\"hostname\":\"createdhost\",\"slave_port\":1912,\"mongod_port_range_begin\":20000,\"mongod_port_range_end\":20001,\"persistant_storage\":false,\"desired_state\":\"disabled\"}"
	req, err := http.NewRequest("PUT", "/api/slaves", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	var createdSlave model.Slave
	db.First(&createdSlave, "hostname = ?", "createdhost")

	assert.NotEmpty(t, createdSlave.ID)
	assert.Equal(t, "createdhost", createdSlave.Hostname)
	assert.Equal(t, 1912, createdSlave.Port)
	assert.Equal(t, 20000, createdSlave.MongodPortRangeBegin)
	assert.Equal(t, 20001, createdSlave.MongodPortRangeEnd)
	assert.Equal(t, false, createdSlave.PersistentStorage)
	assert.Equal(t, model.SlaveStateDisabled, createdSlave.ConfiguedState)
}

func TestMasterAPI_SlavePut_invalid(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI();
	assert.NoError(t, err)
	//Test invalid put (non existing field)
	resp := httptest.NewRecorder()

	req_body := "{\"id_invalid_blabla\":0,\"hostname\":\"createdhost_invalid\",\"slave_port\":1912,\"mongod_port_range_begin\":20000,\"mongod_port_range_end\":20001,\"persistant_storage\":false,\"desired_state\":\"disabled\"}"
	req, err := http.NewRequest("PUT", "/api/slaves", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code)

	var invalidSlave model.Slave
	db.First(&invalidSlave, "hostname = ?", "createdhost_invalid")

	assert.Empty(t, invalidSlave.ID)
}

func TestMasterAPI_SlaveUpdate(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI();
	assert.NoError(t, err)

	//Test valid update
	resp := httptest.NewRecorder()

	req_body := "{\"id\":2,\"hostname\":\"updHost\",\"slave_port\":2,\"mongod_port_range_begin\":101,\"mongod_port_range_end\":201,\"persistant_storage\":true,\"desired_state\":\"disabled\"}"
	req, err := http.NewRequest("POST", "/api/slaves/2", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	var updatedSlave model.Slave
	db.First(&updatedSlave, 2)

	assert.Equal(t, "updHost", updatedSlave.Hostname)
	assert.Equal(t, 2, updatedSlave.Port)
	assert.Equal(t, 101, updatedSlave.MongodPortRangeBegin)
	assert.Equal(t, 201, updatedSlave.MongodPortRangeEnd)
	assert.Equal(t, true, updatedSlave.PersistentStorage)
	assert.Equal(t, model.SlaveStateDisabled, updatedSlave.ConfiguedState)
}

func TestMasterAPI_SlaveUpdate_invalid(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI();
	assert.NoError(t, err)

	//Test invalid update (slave is in active state)
	resp := httptest.NewRecorder()

	req_body := "{\"id\":1,\"hostname\":\"updHost\",\"slave_port\":1912,\"mongod_port_range_begin\":20000,\"mongod_port_range_end\":20001,\"persistant_storage\":false,\"desired_state\":\"active\"}"
	req, err := http.NewRequest("POST", "/api/slaves/1", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code)

	var updatedSlave model.Slave
	db.First(&updatedSlave, 1)

	assert.Equal(t, "host1", updatedSlave.Hostname)
	assert.Equal(t, 1, updatedSlave.Port)
	assert.Equal(t, 2, updatedSlave.MongodPortRangeBegin)
	assert.Equal(t, 3, updatedSlave.MongodPortRangeEnd)
	assert.Equal(t, true, updatedSlave.PersistentStorage)
	assert.Equal(t, model.SlaveStateActive, updatedSlave.ConfiguedState)
}

func TestMasterAPI_SlaveUpdate_change_desired_state(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI();
	assert.NoError(t, err)

	//Test valid state change
	resp := httptest.NewRecorder()

	req_body := "{\"id\":2,\"hostname\":\"host2\",\"slave_port\":1,\"mongod_port_range_begin\":100,\"mongod_port_range_end\":200,\"persistant_storage\":false,\"desired_state\":\"active\"}"
	req, err := http.NewRequest("POST", "/api/slaves/2", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	var updatedSlave model.Slave
	db.First(&updatedSlave, 2)

	assert.Equal(t, model.SlaveStateActive, updatedSlave.ConfiguedState)
}

func TestMasterAPI_SlaveUpdate_change_desired_state_invalid(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI();
	assert.NoError(t, err)

	//Test invalid state change (should not be able to change state while changing another parameter)
	resp := httptest.NewRecorder()

	req_body := "{\"id\":2,\"hostname\":\"host2\",\"slave_port\":1,\"mongod_port_range_begin\":100,\"mongod_port_range_end\":150,\"persistant_storage\":false,\"desired_state\":\"active\"}"
	req, err := http.NewRequest("POST", "/api/slaves/2", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code)

	var updatedSlave model.Slave
	db.First(&updatedSlave, 2)

	assert.Equal(t, model.SlaveStateDisabled, updatedSlave.ConfiguedState)
}

func TestMasterAPI_SlaveDelete(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI();
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
	db, mainRouter, err := createDBAndMasterAPI();
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