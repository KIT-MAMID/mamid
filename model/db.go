package model

import (
	"bufio"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lib/pq"
	"math/rand"
	"os"
	"time"
)

var modelLog = logrus.WithField("module", "model")

const SCHEMA_VERSION string = "0.0.1"

/*
	The structs defined in this file are stored in a database using the `gorm` package.

	Remember to
		Set primary key for a struct.
		Set constraints on specific fields where appropriate.
		Define UniqueIndexes either through a tag or through gorm.DB.AddUniqueIndex()
			for a Unique constraint over multiple fields

	Unless you have a good reason, declare attributes of a struct not null.

	Example:

		type MyType struct {
			Name string `gorm:"not null"`
		}


	Special Cases:

	Enums: 	EnumType.EnumItem => const EnumTypeEnumItem

		Structs using such 'enums' should declare appropriate constraints in the corresponding FieldTag

		Example:

			type MyType struct {
				Name string `sql:"unique"`
			}

*/

type Slave struct {
	ID                   int64  `gorm:"primary_key"`
	Hostname             string `gorm:"unique_index"`
	Port                 PortNumber
	MongodPortRangeBegin PortNumber
	MongodPortRangeEnd   PortNumber
	PersistentStorage    bool
	Mongods              []*Mongod `gorm:"ForeignKey:ParentSlaveID"`
	ConfiguredState      SlaveState

	Problems []*Problem

	// Foreign keys
	RiskGroupID sql.NullInt64 `sql:"type:integer NULL REFERENCES risk_groups(id) DEFERRABLE INITIALLY DEFERRED"`

	ObservationError   *MSPError     // error in observation that is not tied to a specific Mongod
	ObservationErrorID sql.NullInt64 `sql:"type:integer NULL REFERENCES msp_errors(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED"` // TODO not cleaned up on slave deletion right now
}

type PortNumber uint16

const (
	PortNumberMin PortNumber = 1
	PortNumberMax            = 65535
)

type SlaveState uint

const (
	_                           = 0
	SlaveStateActive SlaveState = iota
	SlaveStateMaintenance
	SlaveStateDisabled
)

type ShardingRole string

const (
	ShardingRoleNone         ShardingRole = "none"
	ShardingRoleShardServer  ShardingRole = "shardsvr"
	ShardingRoleConfigServer ShardingRole = "configsvr"
)

func (s ShardingRole) Value() (driver.Value, error) {
	return string(s), nil
}

type ReplicaSet struct {
	ID                    int64  `gorm:"primary_key"` //TODO needs to start incrementing at 1
	Name                  string `gorm:"unique_index"`
	PersistentMemberCount uint
	VolatileMemberCount   uint
	ShardingRole          ShardingRole
	Initiated             bool
	Mongods               []*Mongod

	Problems []*Problem
}

type RiskGroup struct {
	ID     int64  `gorm:"primary_key"` //TODO needs to start incrementing at 1, 0 is special value for slaves "out of risk" => define a constant?
	Name   string `gorm:"unique_index"`
	Slaves []*Slave
}

type Mongod struct {
	// TODO missing UNIQUE constraint
	ID          int64 `gorm:"primary_key"`
	Port        PortNumber
	ReplSetName string

	ObservationError   MSPError
	ObservationErrorID sql.NullInt64 `sql:"type:integer NULL REFERENCES msp_errors(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED"` // TODO not cleaned up on Mongod deletion right now

	LastEstablishStateError   MSPError
	LastEstablishStateErrorID sql.NullInt64 `sql:"type:integer NULL REFERENCES msp_errors(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED"` // TODO not cleaned up on Mongod deletion right now

	ParentSlave   *Slave
	ParentSlaveID int64 `sql:"type:integer REFERENCES slaves(id) DEFERRABLE INITIALLY DEFERRED"`

	ReplicaSet   *ReplicaSet
	ReplicaSetID sql.NullInt64 `sql:"type:integer NULL REFERENCES replica_sets(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED"`

	DesiredState   MongodState
	DesiredStateID int64 `sql:"type:integer NOT NULL REFERENCES mongod_states(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED"` // NOTE: we cascade on delete because Monogd cannot be without DesiredState

	ObservedState   MongodState
	ObservedStateID sql.NullInt64 `sql:"type:integer NULL REFERENCES mongod_states(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED"`
}

type MongodState struct {
	ID             int64 `gorm:"primary_key"`
	ParentMongod   *Mongod
	ParentMongodID int64 `sql:"type:integer NOT NULL REFERENCES mongods(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED"`
	ShardingRole   ShardingRole
	ExecutionState MongodExecutionState
}

type MongodExecutionState uint

const (
	_                                                       = 0
	MongodExecutionStateForceDestroyed MongodExecutionState = iota
	MongodExecutionStateDestroyed
	MongodExecutionStateNotRunning
	MongodExecutionStateUninitiated
	MongodExecutionStateRecovering // invalid for a desired MongodState
	MongodExecutionStateRunning
)

// msp.Error
// duplicated for decoupling protocol & internal representation
type MSPError struct {
	ID              int64 `gorm:"primary_key"`
	Identifier      string
	Description     string
	LongDescription string
}

type ProblemType uint

const (
	_                                 = 0
	ProblemTypeConnection ProblemType = iota
	ProblemTypeMismatch
	ProblemTypeDesiredReplicaSetConstraint
	ProblemTypeObservedReplicaSetConstraint
)

type Problem struct {
	ID              int64 `gorm:"primary_key"`
	Description     string
	LongDescription string
	ProblemType     ProblemType
	FirstOccurred   time.Time
	LastUpdated     time.Time

	Slave   *Slave
	SlaveID sql.NullInt64 `sql:"type:integer NULL REFERENCES slaves(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED"`

	ReplicaSet   *ReplicaSet
	ReplicaSetID sql.NullInt64 `sql:"type:integer NULL REFERENCES replica_sets(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED"`

	Mongod   *Mongod
	MongodID sql.NullInt64 `sql:"type:integer NULL REFERENCES mongods(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED"`
}

type MamidMetadata struct {
	Key, Value string
}

type MongodKeyfile struct {
	ID      int64 `gorm:"primary_key"`
	Content string
}

type MongodbCredential struct {
	ID       int64 `gorm:"primary_key"`
	Username string
	Password string
}

type DB struct {
	Driver  string
	gormDB  *gorm.DB
	dbName  sql.NullString
	connDSN sql.NullString
}

func (db *DB) Begin() *gorm.DB {
	tx := db.gormDB.Begin()
	return tx
}

func (db *DB) CloseAndDrop() {

	if !(db.dbName.Valid && db.connDSN.Valid) {
		modelLog.Fatalf("model.DB object not initialized for dropping database")
	}

	if err := db.gormDB.Close(); err != nil {
		modelLog.Fatalf("could not close connection with database open: %s", err)
	}

	const driver = "postgres"
	c, err := sql.Open(driver, db.connDSN.String)
	if err != nil {
		modelLog.Fatalf("cannot connect to test database: %s", err)
	}
	defer c.Close()

	res, err := c.Exec("DROP DATABASE " + db.dbName.String)
	if err != nil {
		modelLog.Fatalf("could not drop database `%s`: %s", db.dbName.String, err)
	} else {
		modelLog.Infof("dropped database `%s`: %s", db.dbName.String, res)
	}

}

// Idempotently migrate the database schema.
// Currenlty, only creation of the schema is supported.
func (dbWrapper *DB) migrate() (err error) {

	db := dbWrapper.gormDB

	if !db.HasTable(&MamidMetadata{}) {

		// run the populating query

		ddlStatements, err := Asset("model/sql/mamid_postgresql.sql")
		if err != nil {
			return fmt.Errorf("sql DDL data not found: %s", err)
		}

		err = db.Exec(string(ddlStatements), []interface{}{}).Error
		if err != nil {
			return fmt.Errorf("error running DDL statements: %s", err)
		}

		// persist schema version
		if err = dbWrapper.setMetadata("schema_version", SCHEMA_VERSION); err != nil {
			return fmt.Errorf("error setting schema version: %s", err)
		}

	} else {

		version, err := dbWrapper.schemaVersion()
		if err != nil {
			return fmt.Errorf("error determining schema version: %s", err)
		}

		if version != SCHEMA_VERSION {
			return fmt.Errorf("the database has already been populated, migrations are not supported")
		}

	}

	return nil
}

func (dbWrapper *DB) schemaVersion() (version string, err error) {
	return dbWrapper.metadata("schema_version")
}

// Return metadata or an empty `value` if the entry does not exist
func (dbWrapper *DB) metadata(key string) (value string, err error) {
	metadata := MamidMetadata{}
	if res := dbWrapper.gormDB.Find(&metadata, MamidMetadata{Key: key}); res.Error != nil {
		if res.RecordNotFound() {
			return "", nil
		} else {
			return "", err
		}
	} else {
		return metadata.Value, nil
	}
}

func (dbWrapper *DB) setMetadata(key, value string) (err error) {
	metadata := MamidMetadata{
		Key:   key,
		Value: value,
	}
	err = dbWrapper.gormDB.Save(&metadata).Error
	return
}

func InitializeDB(driver, dsn string) (*DB, error) {

	gormDB, err := gorm.Open(driver, dsn)
	if err != nil {
		return nil, err
	}

	gormDB.SetLogger(modelLog)

	db := &DB{
		Driver: driver,
		gormDB: gormDB}

	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("could not migrate database: %s", err)
	}

	return db, err

}

func InitializeTestDBFromFile(file string) (db *DB, dsn string, err error) {
	db, dsn, err = InitializeTestDB()
	if err != nil {
		return
	}
	fd, err := os.Open(file)
	if err != nil {
		return
	}
	defer fd.Close()
	scann := bufio.NewScanner(fd)
	nativeDb, err := sql.Open("postgres", dsn)
	defer nativeDb.Close()
	if err != nil {
		return
	}
	tx, err := nativeDb.Begin()
	if err != nil {
		return
	}
	_, err = nativeDb.Exec("DELETE FROM mamid_metadata")
	if err != nil {
		return nil, "", err
	}
	line := 1
	for scann.Scan() {
		_, err := tx.Exec(scann.Text())
		if err != nil {
			return nil, "", err
		}
		line++
	}
	err = tx.Commit()
	return
}

func InitializeTestDB() (db *DB, dsn string, err error) {

	const driver = "postgres"
	connDSN := os.Getenv("MAMID_TESTDB_DSN")
	if connDSN == "" {
		modelLog.Panic("MAMID_TESTDB_DSN environment variable is not set")
	}

	c, err := sql.Open(driver, connDSN)
	if err != nil {
		modelLog.Fatalf("cannot connect to test database: %s", err)
	}
	defer c.Close()

	dbName := randomDBName("mamid_testing_", 20)
	_, err = c.Exec("CREATE DATABASE " + dbName)
	if err != nil {
		modelLog.Fatalf("cannot create test database `%s`: %s", dbName, err)
	}
	c.Close()

	// NOTE: in the current implementation of pq (postgres driver), the last key-value pair wins over previous ones with the same key
	dsn = fmt.Sprintf("%s dbname=%s", connDSN, dbName)
	gormDB, err := gorm.Open(driver, dsn)
	if err != nil {
		modelLog.Fatalf("cannot open just created test database `%s`: %s", dsn, err)
	}

	gormDB.SetLogger(modelLog)

	db = &DB{
		Driver:  driver,
		gormDB:  gormDB,
		dbName:  sql.NullString{String: dbName, Valid: true},
		connDSN: sql.NullString{String: connDSN, Valid: true},
	}

	if err := db.migrate(); err != nil {
		return nil, dsn, fmt.Errorf("could not migrate database: %s", err)
	}

	return db, dsn, nil

}

func randomDBName(prefix string, strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return prefix + string(result)
}

func NullIntValue(value int64) sql.NullInt64 {
	return sql.NullInt64{Int64: value, Valid: true}
}

func NullInt() sql.NullInt64 {
	return sql.NullInt64{}
}

func NullIntToPtr(nullint sql.NullInt64) *int64 {
	if nullint.Valid {
		value := nullint.Int64
		return &value
	} else {
		return nil
	}
}

func PtrToNullInt(value *int64) sql.NullInt64 {
	if value != nil {
		return NullIntValue(*value)
	} else {
		return NullInt()
	}
}

func IsIntegrityConstraintViolation(err error) bool {
	if driverErr, ok := err.(*pq.Error); ok && driverErr.Code.Class() == "23" {
		// Integrity Constraint Violation
		return true
	} else {
		return false
	}
}
