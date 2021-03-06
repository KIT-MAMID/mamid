package masterapi

import (
	"encoding/json"
	"fmt"
	"github.com/KIT-MAMID/mamid/master"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func createDBAndMasterAPI(t *testing.T) (db *model.DB, mainRouter *mux.Router, err error) {
	// Setup database
	db, path, err := model.InitializeTestDB()
	assert.Nil(t, err)
	t.Logf("creating test db: %s", path)

	tx := db.Begin()

	dbRiskGroup := model.RiskGroup{
		Name: "risk1",
	}
	assert.NoError(t, tx.Create(&dbRiskGroup).Error)

	dbRiskGroup2 := model.RiskGroup{
		Name: "risk2",
	}
	assert.NoError(t, tx.Create(&dbRiskGroup2).Error)

	dbRiskGroup3 := model.RiskGroup{
		Name: "risk3",
	}
	assert.NoError(t, tx.Create(&dbRiskGroup3).Error)

	dbSlave := model.Slave{
		Hostname:             "host1",
		Port:                 1,
		MongodPortRangeBegin: 2,
		MongodPortRangeEnd:   3,
		PersistentStorage:    true,
		Mongods:              []*model.Mongod{},
		ConfiguredState:      model.SlaveStateActive,
		RiskGroupID:          model.NullIntValue(dbRiskGroup2.ID),
	}
	assert.NoError(t, tx.Create(&dbSlave).Error)

	dbReplicaset := model.ReplicaSet{
		Name: "repl1",
		PersistentMemberCount: 1,
		VolatileMemberCount:   2,
		ShardingRole:          model.ShardingRoleNone,
	}
	assert.NoError(t, tx.Create(&dbReplicaset).Error)

	m1 := model.Mongod{
		Port:          5001,
		ReplSetName:   "repl1",
		ReplicaSetID:  model.NullIntValue(dbReplicaset.ID),
		ParentSlaveID: dbSlave.ID,
		//		DesiredStateID: 1,
	}
	assert.NoError(t, tx.Create(&m1).Error)
	ds1 := model.MongodState{
		ParentMongodID: m1.ID,
		ShardingRole:   model.ShardingRoleNone,
	}
	assert.NoError(t, tx.Create(&ds1).Error)
	assert.NoError(t, tx.Model(&m1).Update("DesiredStateID", ds1.ID).Error)

	dbSlave2 := model.Slave{
		Hostname:             "host2",
		Port:                 1,
		MongodPortRangeBegin: 100,
		MongodPortRangeEnd:   200,
		PersistentStorage:    false,
		Mongods:              []*model.Mongod{},
		ConfiguredState:      model.SlaveStateDisabled,
		RiskGroupID:          model.NullIntValue(dbRiskGroup.ID),
	}
	assert.NoError(t, tx.Create(&dbSlave2).Error)

	dbSlave3 := model.Slave{
		Hostname:             "host3",
		Port:                 1,
		MongodPortRangeBegin: 100,
		MongodPortRangeEnd:   200,
		PersistentStorage:    false,
		Mongods:              []*model.Mongod{},
		ConfiguredState:      model.SlaveStateDisabled,
		RiskGroupID:          model.NullInt(),
	}
	assert.NoError(t, tx.Create(&dbSlave3).Error)

	utc, err := time.LoadLocation("UTC")
	assert.NoError(t, err)

	dbProblem := model.Problem{
		Description:   "foo",
		FirstOccurred: time.Date(2000, time.January, 1, 0, 0, 0, 0, utc),
		SlaveID:       model.NullIntValue(dbSlave.ID),
	}
	assert.NoError(t, tx.Create(&dbProblem).Error)

	dbProblem2 := model.Problem{
		Description:   "bar",
		FirstOccurred: time.Date(2010, time.January, 1, 0, 0, 0, 0, utc),
		ReplicaSetID:  model.NullIntValue(dbReplicaset.ID),
	}
	assert.NoError(t, tx.Create(&dbProblem2).Error)
	tx.Commit()

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
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	// Test correct get
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/slaves", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	var getSlaveResult []Slave
	err = json.NewDecoder(resp.Body).Decode(&getSlaveResult)
	assert.NoError(t, err)

	assert.Equal(t, 3, len(getSlaveResult))
	assert.Equal(t, "host1", getSlaveResult[0].Hostname)
	assert.EqualValues(t, 1, getSlaveResult[0].Port)
	assert.EqualValues(t, 2, getSlaveResult[0].MongodPortRangeBegin)
	assert.EqualValues(t, 3, getSlaveResult[0].MongodPortRangeEnd)
	assert.Equal(t, true, getSlaveResult[0].PersistentStorage)
	assert.Equal(t, "active", getSlaveResult[0].ConfiguredState)
}

func TestMasterAPI_SlaveById(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
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
	assert.Equal(t, false, getSlaveResult.ConfiguredStateTransitioning)
}

func TestMasterAPI_SlavePut(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
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
	{
		tx := db.Begin()
		tx.First(&createdSlave, "hostname = ?", "createdhost")
		tx.Rollback()
	}

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
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
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
	defer db.CloseAndDrop()
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
	{
		tx := db.Begin()
		assert.NoError(t, tx.First(&invalidSlave, "hostname = ?", "createdhost").Error)
		tx.Rollback()
	}
}

func TestMasterAPI_SlavePut_missingField(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)
	//Test invalid put (non existing field)
	resp := httptest.NewRecorder()

	// missing field: mongod_port_range_begin
	req_body := "{\"hostname\":\"createdhost_invalid\",\"slave_port\":1912,\"mongod_port_range_begin\":20000,\"persistent_storage\":false,\"configured_state\":\"disabled\"}"
	req, err := http.NewRequest("PUT", "/api/slaves", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code)

	var invalidSlave model.Slave
	{
		tx := db.Begin()
		assert.Error(t, tx.First(&invalidSlave, "hostname = ?", "createdhost_invalid").Error)
		tx.Rollback()
	}
}

func TestMasterAPI_SlaveUpdate(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	//Test valid update
	resp := httptest.NewRecorder()

	req_body := "{\"id\":2,\"hostname\":\"updHost\",\"slave_port\":2,\"mongod_port_range_begin\":101,\"mongod_port_range_end\":201,\"persistent_storage\":true,\"configured_state\":\"disabled\"}"
	req, err := http.NewRequest("POST", "/api/slaves/2", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	var updatedSlave model.Slave
	{
		tx := db.Begin()
		tx.First(&updatedSlave, 2)
		tx.Rollback()
	}

	assert.Equal(t, "updHost", updatedSlave.Hostname)
	assert.EqualValues(t, 2, updatedSlave.Port)
	assert.EqualValues(t, 101, updatedSlave.MongodPortRangeBegin)
	assert.EqualValues(t, 201, updatedSlave.MongodPortRangeEnd)
	assert.Equal(t, true, updatedSlave.PersistentStorage)
	assert.Equal(t, model.SlaveStateDisabled, updatedSlave.ConfiguredState)
}

func TestMasterAPI_SlaveUpdate_invalid(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	//Test invalid update (slave is in active state)
	resp := httptest.NewRecorder()

	req_body := "{\"id\":1,\"hostname\":\"updHost\",\"slave_port\":1912,\"mongod_port_range_begin\":20000,\"mongod_port_range_end\":20001,\"persistent_storage\":false,\"configured_state\":\"active\"}"
	req, err := http.NewRequest("POST", "/api/slaves/1", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 403, resp.Code)

	var updatedSlave model.Slave
	{
		tx := db.Begin()
		tx.First(&updatedSlave, 1)
		tx.Rollback()
	}

	assert.Equal(t, "host1", updatedSlave.Hostname)
	assert.EqualValues(t, 1, updatedSlave.Port)
	assert.EqualValues(t, 2, updatedSlave.MongodPortRangeBegin)
	assert.EqualValues(t, 3, updatedSlave.MongodPortRangeEnd)
	assert.Equal(t, true, updatedSlave.PersistentStorage)
	assert.Equal(t, model.SlaveStateActive, updatedSlave.ConfiguredState)
}

func TestMasterAPI_SlaveUpdate_invalid_ports(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	//Test valid update
	resp := httptest.NewRecorder()

	req_body := "{\"id\":2,\"hostname\":\"updHost\",\"slave_port\":2,\"mongod_port_range_begin\":200,\"mongod_port_range_end\":100,\"persistent_storage\":true,\"configured_state\":\"disabled\"}"
	req, err := http.NewRequest("POST", "/api/slaves/2", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	fmt.Println(resp.Body)

	assert.Equal(t, 400, resp.Code)
}

func TestMasterAPI_SlaveUpdate_change_desired_state(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	//Test valid state change
	resp := httptest.NewRecorder()

	req_body := "{\"id\":2,\"hostname\":\"host2\",\"slave_port\":1,\"mongod_port_range_begin\":100,\"mongod_port_range_end\":200,\"persistent_storage\":false,\"configured_state\":\"active\", \"risk_group_id\":1}"
	req, err := http.NewRequest("POST", "/api/slaves/2", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	var updatedSlave model.Slave
	{
		tx := db.Begin()
		tx.First(&updatedSlave, 2)
		tx.Rollback()
	}

	assert.Equal(t, model.SlaveStateActive, updatedSlave.ConfiguredState)
}

func TestMasterAPI_SlaveUpdate_change_desired_state_disabled(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	//Test valid state change
	resp := httptest.NewRecorder()

	req_body := "{\"id\":1,\"hostname\":\"host1\",\"slave_port\":1,\"mongod_port_range_begin\":2,\"mongod_port_range_end\":3,\"persistent_storage\":true,\"configured_state\":\"disabled\", \"risk_group_id\":2}"
	req, err := http.NewRequest("POST", "/api/slaves/1", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	var updatedSlave model.Slave
	{
		tx := db.Begin()
		tx.First(&updatedSlave, 2)
		tx.Rollback()
	}

	assert.Equal(t, model.SlaveStateDisabled, updatedSlave.ConfiguredState)
}

func TestMasterAPI_SlaveUpdate_existingHostname(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	//Test invalid update (hostname already exists)
	resp := httptest.NewRecorder()

	req_body := "{\"id\":2,\"hostname\":\"host1\",\"slave_port\":2,\"mongod_port_range_begin\":101,\"mongod_port_range_end\":201,\"persistent_storage\":true,\"configured_state\":\"disabled\"}"
	req, err := http.NewRequest("POST", "/api/slaves/2", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code)
}

func TestMasterAPI_SlaveDelete(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	//Test valid delete
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("DELETE", "/api/slaves/2", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	var updatedSlave model.Slave
	{
		tx := db.Begin()
		tx.First(&updatedSlave, 2)
		tx.Rollback()
	}

	assert.Empty(t, updatedSlave.ID)
}

func TestMasterAPI_SlaveDelete_invalid(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	//Test invalid delete (active slave)
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("DELETE", "/api/slaves/1", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 403, resp.Code)

	var updatedSlave model.Slave
	{
		tx := db.Begin()
		tx.First(&updatedSlave, 1)
		tx.Rollback()
	}

	assert.NotEmpty(t, updatedSlave.ID)
}

// Test correct get of replica sets
func TestMasterAPI_ReplicaSetIndex(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/replicasets", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	var getReplsetResult []ReplicaSet
	err = json.NewDecoder(resp.Body).Decode(&getReplsetResult)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(getReplsetResult))
	assert.Equal(t, "repl1", getReplsetResult[0].Name)
	assert.EqualValues(t, 1, getReplsetResult[0].PersistentNodeCount)
	assert.EqualValues(t, 2, getReplsetResult[0].VolatileNodeCount)
	assert.EqualValues(t, model.ShardingRoleNone, getReplsetResult[0].ShardingRole)
}

func TestMasterAPI_ReplicaSetById(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	// Test correct get
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/replicasets/1", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	if !assert.Equal(t, 200, resp.Code) {
		fmt.Println(resp.Body.String())
	}

	var getReplSetResult ReplicaSet
	err = json.NewDecoder(resp.Body).Decode(&getReplSetResult)
	assert.NoError(t, err)

	assert.EqualValues(t, 1, getReplSetResult.ID)
	assert.Equal(t, "repl1", getReplSetResult.Name)
	assert.EqualValues(t, 1, getReplSetResult.PersistentNodeCount)
	assert.EqualValues(t, 2, getReplSetResult.VolatileNodeCount)
	assert.EqualValues(t, model.ShardingRoleNone, getReplSetResult.ShardingRole)
}

func TestMasterAPI_ReplicaSetById_not_existing(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/replicasets/9000", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	if !assert.Equal(t, 404, resp.Code) {
		fmt.Println(resp.Body.String())
	}
}

func TestMasterAPI_ReplicaSetPut(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	//Test correct put
	resp := httptest.NewRecorder()

	req_body := "{\"id\":0,\"name\":\"repl2\",\"persistent_node_count\":1," +
		"\"volatile_node_count\":2,\"sharding_role\":\"configsvr\"}"
	req, err := http.NewRequest("PUT", "/api/replicasets", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	if !assert.Equal(t, 200, resp.Code) {
		fmt.Println(resp.Body.String())
	}

	var createdReplSet model.ReplicaSet
	{
		tx := db.Begin()
		tx.First(&createdReplSet, "name = ?", "repl2")
		tx.Rollback()
	}

	//Check created database entry
	assert.NotEmpty(t, createdReplSet.ID)
	assert.Equal(t, "repl2", createdReplSet.Name)
	assert.EqualValues(t, 1, createdReplSet.PersistentMemberCount)
	assert.EqualValues(t, 2, createdReplSet.VolatileMemberCount)
	assert.Equal(t, model.ShardingRoleConfigServer, createdReplSet.ShardingRole)

	//Check returned object
	var getReplicaSetResult ReplicaSet
	err = json.NewDecoder(resp.Body).Decode(&getReplicaSetResult)
	assert.NoError(t, err)

	assert.NotEmpty(t, getReplicaSetResult.ID)
	assert.Equal(t, "repl2", getReplicaSetResult.Name)
}

func TestMasterAPI_ReplicaSetUpdate(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req_body := "{\"id\":1,\"name\":\"repl1\",\"persistent_node_count\":1," +
		"\"volatile_node_count\":4,\"sharding_role\":\"none\"}"
	req, err := http.NewRequest("POST", "/api/replicasets/1", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	var updateReplSet model.ReplicaSet
	{
		tx := db.Begin()
		tx.First(&updateReplSet, 1)
		tx.Rollback()
	}

	assert.Equal(t, "repl1", updateReplSet.Name)
	assert.EqualValues(t, 1, updateReplSet.PersistentMemberCount)
	assert.EqualValues(t, 4, updateReplSet.VolatileMemberCount)
	assert.EqualValues(t, model.ShardingRoleNone, updateReplSet.ShardingRole)
}

func TestMasterAPI_ReplicaSetUpdate_zero_values(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req_body := "{\"id\":1,\"name\":\"repl1\",\"persistent_node_count\":0," +
		"\"volatile_node_count\":3,\"sharding_role\":\"none\"}"
	req, err := http.NewRequest("POST", "/api/replicasets/1", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	var updateReplSet model.ReplicaSet
	{
		tx := db.Begin()
		tx.First(&updateReplSet, 1)
		tx.Rollback()
	}

	assert.Equal(t, "repl1", updateReplSet.Name)
	assert.EqualValues(t, 0, updateReplSet.PersistentMemberCount)
	assert.EqualValues(t, 3, updateReplSet.VolatileMemberCount)
	assert.EqualValues(t, model.ShardingRoleNone, updateReplSet.ShardingRole)
}

func TestMasterAPI_ReplicaSetUpdate_not_existing(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req_body := "{\"id\":9000,\"name\":\"repl1\",\"persistent_node_count\":1," +
		"\"volatile_node_count\":4,\"configure_as_sharding_config_server\":false}"
	req, err := http.NewRequest("POST", "/api/replicasets/9000", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 404, resp.Code)
}

func TestMasterAPI_ReplicaSetDelete(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req, err := http.NewRequest("DELETE", "/api/replicasets/1", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	{
		tx := db.Begin()
		assert.True(t, tx.First(&model.ReplicaSet{}, 1).RecordNotFound())
		tx.Rollback()
	}
}

func TestMasterAPI_ReplicaSetDelete_not_existing(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req, err := http.NewRequest("DELETE", "/api/replicasets/9000", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 404, resp.Code)
}

func TestMasterAPI_ReplicaSetGetSlaves(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/replicasets/1/slaves", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	if !assert.Equal(t, 200, resp.Code) {
		fmt.Println(resp.Body.String())
	}

	var getSlavesRes []Slave
	err = json.NewDecoder(resp.Body).Decode(&getSlavesRes)
	assert.NoError(t, err)

	assert.EqualValues(t, 1, len(getSlavesRes))
	assert.EqualValues(t, 1, getSlavesRes[0].ID)
}

func TestMasterAPI_ProblemIndex(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/problems", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	var getProblemsResult []Problem
	err = json.NewDecoder(resp.Body).Decode(&getProblemsResult)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(getProblemsResult))
	assert.Equal(t, "foo", getProblemsResult[0].Description)
}

func TestMasterAPI_ProblemById(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/problems/1", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.EqualValues(t, 200, resp.Code)

	var getProblemsResult Problem
	err = json.NewDecoder(resp.Body).Decode(&getProblemsResult)
	assert.NoError(t, err)

	assert.Equal(t, "foo", getProblemsResult.Description)
}

func TestMasterAPI_ProblemBySlave(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/slaves/1/problems", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.EqualValues(t, 200, resp.Code)

	var getProblemsResult []Problem
	err = json.NewDecoder(resp.Body).Decode(&getProblemsResult)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(getProblemsResult))
	assert.Equal(t, "foo", getProblemsResult[0].Description)
}

func TestMasterAPI_ProblemByReplicaSet(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/replicasets/1/problems", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.EqualValues(t, 200, resp.Code)

	var getProblemsResult []Problem
	err = json.NewDecoder(resp.Body).Decode(&getProblemsResult)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(getProblemsResult))
	assert.Equal(t, "bar", getProblemsResult[0].Description)
}

func TestMasterAPI_RiskGroupIndex(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/riskgroups", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.EqualValues(t, 200, resp.Code)

	var getRiskGroupsResult []RiskGroup
	err = json.NewDecoder(resp.Body).Decode(&getRiskGroupsResult)
	assert.NoError(t, err)

	assert.Equal(t, 3, len(getRiskGroupsResult))
	assert.Equal(t, "risk1", getRiskGroupsResult[0].Name)
}

func TestMasterAPI_RiskGroupById(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	// Test correct get
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/riskgroups/1", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.EqualValues(t, 200, resp.Code)

	var getRiskGroupResult RiskGroup
	err = json.NewDecoder(resp.Body).Decode(&getRiskGroupResult)
	assert.NoError(t, err)

	assert.EqualValues(t, 1, getRiskGroupResult.ID)
	assert.Equal(t, "risk1", getRiskGroupResult.Name)
}

func TestMasterAPI_RiskGroupById_zero(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	// Test correct get
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/riskgroups/0", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.EqualValues(t, 400, resp.Code)
}

func TestMasterAPI_RiskGroupPut(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	//Test correct put
	resp := httptest.NewRecorder()

	req_body := "{\"id\":0,\"name\":\"newrisk\"}"
	req, err := http.NewRequest("PUT", "/api/riskgroups", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	if !assert.Equal(t, 200, resp.Code) {
		fmt.Println(resp.Body.String())
	}

	var createdRiskGroup model.RiskGroup
	{
		tx := db.Begin()
		tx.First(&createdRiskGroup, "name = ?", "newrisk")
		tx.Rollback()
	}

	//Check created database entry
	assert.NotEmpty(t, createdRiskGroup.ID)
	assert.Equal(t, "newrisk", createdRiskGroup.Name)

	//Check returned object
	var getRiskGroupResult RiskGroup
	err = json.NewDecoder(resp.Body).Decode(&getRiskGroupResult)
	assert.NoError(t, err)

	assert.NotEmpty(t, getRiskGroupResult.ID)
	assert.Equal(t, "newrisk", getRiskGroupResult.Name)
}

func TestMasterAPI_RiskGroupPut_existing_name(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req_body := "{\"id\":0,\"name\":\"risk1\"}"
	req, err := http.NewRequest("PUT", "/api/riskgroups", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 400, resp.Code)
}

func TestMasterAPI_RiskGroupUpdate(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req_body := "{\"id\":1,\"name\":\"foo\"}"
	req, err := http.NewRequest("POST", "/api/riskgroups/1", strings.NewReader(req_body))
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	var updatedRiskGroup model.RiskGroup
	{
		tx := db.Begin()
		tx.First(&updatedRiskGroup, 1)
		tx.Rollback()
	}

	assert.Equal(t, "foo", updatedRiskGroup.Name)
}

func TestMasterAPI_RiskGroupDelete(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req, err := http.NewRequest("DELETE", "/api/riskgroups/3", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	var deletedRiskGroup model.RiskGroup
	{
		tx := db.Begin()
		tx.First(&deletedRiskGroup, 3)
		tx.Rollback()
	}

	assert.Empty(t, deletedRiskGroup.ID)
}

func TestMasterAPI_RiskGroupDelete_has_slaves(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req, err := http.NewRequest("DELETE", "/api/riskgroups/1", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 403, resp.Code)

	var deletedRiskGroup model.Slave
	{
		tx := db.Begin()
		tx.First(&deletedRiskGroup, 1)
		tx.Rollback()
	}

	assert.NotEmpty(t, deletedRiskGroup.ID)
}

func TestMasterAPI_RiskGroupDelete_not_existing(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req, err := http.NewRequest("DELETE", "/api/riskgroups/9000", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.Equal(t, 404, resp.Code)
}

func TestMasterAPI_RiskGroupAssignSlave(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	// Test correct get
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("PUT", "/api/riskgroups/2/slaves/2", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.EqualValues(t, 200, resp.Code)

	var assignedSlave model.Slave
	{
		tx := db.Begin()
		tx.First(&assignedSlave, 2)
		tx.Rollback()
	}

	assert.EqualValues(t, 2, assignedSlave.RiskGroupID.Int64)
}

func TestMasterAPI_RiskGroupAssignSlave_active(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	// Test correct get
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("PUT", "/api/riskgroups/1/slaves/1", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.EqualValues(t, 403, resp.Code)
}

func TestMasterAPI_RiskGroupRemoveSlave(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	// Test correct get
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("DELETE", "/api/riskgroups/1/slaves/2", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.EqualValues(t, 200, resp.Code)

	var removedSlave model.Slave
	{
		tx := db.Begin()
		tx.First(&removedSlave, 2)
		tx.Rollback()
	}

	assert.False(t, removedSlave.RiskGroupID.Valid)
}

func TestMasterAPI_RiskGroupRemoveSlave_active(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	// Test correct get
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("DELETE", "/api/riskgroups/2/slaves/1", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.EqualValues(t, 403, resp.Code)
}

func TestMasterAPI_RiskGroupRemoveSlave_not_in_risk_group(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	// Test correct get
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("DELETE", "/api/riskgroups/2/slaves/2", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.EqualValues(t, 404, resp.Code)
}

func TestMasterAPI_RiskGroupGetSlaves(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	// Test correct get
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/riskgroups/1/slaves", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.EqualValues(t, 200, resp.Code)

	var getSlaveResult []Slave
	err = json.NewDecoder(resp.Body).Decode(&getSlaveResult)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(getSlaveResult))
	assert.Equal(t, "host2", getSlaveResult[0].Hostname)
}

func TestMasterAPI_RiskGroupGetUnassignedSlaves(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	// Test correct get
	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/riskgroups/null/slaves", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.EqualValues(t, 200, resp.Code)

	var getSlaveResult []Slave
	err = json.NewDecoder(resp.Body).Decode(&getSlaveResult)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(getSlaveResult))
	assert.Equal(t, "host3", getSlaveResult[0].Hostname)
}

func TestMasterAPI_MongodsBySlave(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/slaves/1/mongods", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.EqualValues(t, 200, resp.Code)

	var getMongodsResult []Mongod
	err = json.NewDecoder(resp.Body).Decode(&getMongodsResult)
	assert.NoError(t, err)

	assert.EqualValues(t, 1, len(getMongodsResult))
	assert.EqualValues(t, 5001, getMongodsResult[0].Port)
}

func TestMasterAPI_MongodsByReplicaSet(t *testing.T) {
	db, mainRouter, err := createDBAndMasterAPI(t)
	defer db.CloseAndDrop()
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/api/replicasets/1/mongods", nil)
	assert.NoError(t, err)
	mainRouter.ServeHTTP(resp, req)

	assert.EqualValues(t, 200, resp.Code)

	var getMongodsResult []Mongod
	err = json.NewDecoder(resp.Body).Decode(&getMongodsResult)
	assert.NoError(t, err)

	assert.EqualValues(t, 1, len(getMongodsResult))
	assert.EqualValues(t, 5001, getMongodsResult[0].Port)
}
