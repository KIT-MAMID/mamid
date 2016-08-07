package model

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/msp"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"io/ioutil"
	"log"
	"os"
	"time"
)

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

		Structs using such 'enums' should declare appropriate constraints in the corresponding FieldTag,
		using go-sqlite3 syntax

		Example:

			type MyType struct {
				Name string `sql:"unique"`
			}

*/

type Slave struct {
	ID                   uint   `gorm:"primary_key"`
	Hostname             string `gorm:"unique_index"`
	Port                 PortNumber
	MongodPortRangeBegin PortNumber
	MongodPortRangeEnd   PortNumber
	PersistentStorage    bool
	Mongods              []*Mongod `gorm:"ForeignKey:ParentSlaveID"`
	ConfiguredState      SlaveState

	Problems []*Problem

	// Foreign keys
	RiskGroupID uint `sql:"type:integer REFERENCES risk_groups(id)"`
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

type ReplicaSet struct {
	ID                              uint   `gorm:"primary_key"`
	Name                            string `gorm:"unique_index"`
	PersistentMemberCount           uint
	VolatileMemberCount             uint
	ConfigureAsShardingConfigServer bool
	Mongods                         []*Mongod

	Problems []*Problem
}

type RiskGroup struct {
	ID     uint   `gorm:"primary_key"`
	Name   string `gorm:"unique_index"`
	Slaves []*Slave
}

type Mongod struct {
	// TODO missing UNIQUE constraint
	ID          uint `gorm:"primary_key"`
	Port        PortNumber
	ReplSetName string

	ObservationError   MSPError
	ObservationErrorID uint `sql:"type:integer REFERENCES msp_errors(id)"`

	LastEstablishStateError   MSPError
	LastEstablishStateErrorID uint `sql:"type:integer REFERENCES msp_errors(id)"`

	ParentSlave   *Slave
	ParentSlaveID uint `sql:"type:integer REFERENCES slaves(id)"`

	ReplicaSet   *ReplicaSet
	ReplicaSetID uint `sql:"type:integer REFERENCES replica_sets(id)"`

	DesiredState   MongodState
	DesiredStateID uint `sql:"type:integer REFERENCES mongod_states(id)"`

	ObservedState   MongodState
	ObservedStateID uint `sql:"type:integer REFERENCES mongod_states(id)"`
}

type MongodState struct {
	ID                     uint `gorm:"primary_key"`
	IsShardingConfigServer bool
	ExecutionState         MongodExecutionState
	ReplicaSetMembers      []ReplicaSetMember
}

type MongodExecutionState uint

const (
	_                                                  = 0
	MongodExecutionStateDestroyed MongodExecutionState = iota
	MongodExecutionStateNotRunning
	MongodExecutionStateRecovering // invalid for a desired MongodState
	MongodExecutionStateRunning
)

type ReplicaSetMember struct { // was ReplicaSetMember in UML
	// TODO missing primary key.
	ID       uint `gorm:"primary_key"`
	Hostname string
	Port     PortNumber

	// Foreign key to parent MongodState
	MongodStateID uint `sql:"type:integer REFERENCES mongod_states(id)"`
}

type MSPError struct {
	// Union type for the different errors returned by msp
	// Necessary to decouple MSP from ORM / DB logic
	ID                 uint `gorm:"primary_key"`
	CommunicationError CommunicationError
	SlaveError         SlaveError
}

type CommunicationError struct {
	msp.CommunicationError
	ID         uint `gorm:"primary_key"`
	MSPErrorID uint `sql:"type:integer REFERENCES msp_errors(id)"`
}

type SlaveError struct {
	msp.SlaveError
	ID         uint `gorm:"primary_key"`
	MSPErrorID uint `sql:"type:integer REFERENCES msp_errors(id)"`
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
	ID              uint `gorm:"primary_key"`
	Description     string
	LongDescription string
	ProblemType     ProblemType
	FirstOccurred   time.Time
	LastUpdated     time.Time

	Slave   *Slave
	SlaveID uint `sql:"type:integer REFERENCES slaves(id)"`

	ReplicaSet   *ReplicaSet
	ReplicaSetID uint `sql:"type:integer REFERENCES replica_sets(id)"`

	Mongod   *Mongod
	MongodID uint `sql:"type:integer REFERENCES mongods(id)"`
}

func InitializeFileFromFile(path string) (db *gorm.DB, err error) {

	db, err = initializeDB(path)
	if err != nil {
		return nil, err
	}

	migrateDB(db)

	return db, nil

}

func InitializeTestDB() (db *gorm.DB, err error) {

	path := "/tmp/mamid_test.db"
	os.Remove(path)
	db, err = initializeDB(path)
	if err != nil {
		return nil, err
	}

	migrateDB(db)

	return db, nil

}

func InitializeTestDBWithSQL(sqlFilePath string) (db *gorm.DB, err error) {

	path := "/tmp/mamid_test.db"
	os.Remove(path)
	db, err = initializeDB(path)
	if err != nil {
		return nil, err
	}

	if sqlFilePath != "" {
		statements, err := ioutil.ReadFile(sqlFilePath)
		if err != nil {
			return nil, err
		}

		db.Exec(string(statements), []interface{}{})

	}

	migrateDB(db)

	return db, nil

}

func InitializeInMemoryDB(sqlFilePath string) (db *gorm.DB, err error) {

	db, err = initializeDB(":memory:")
	if err != nil {
		return
	}

	if sqlFilePath != "" {
		statements, err := ioutil.ReadFile(sqlFilePath)
		if err != nil {
			return nil, err
		}

		db.Exec(string(statements), []interface{}{})

	}

	migrateDB(db)

	return db, nil

}

func initializeDB(dsn string) (db *gorm.DB, err error) {

	db, err = gorm.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	return db, nil

}

func migrateDB(db *gorm.DB) {
	db.AutoMigrate(&Slave{}, &ReplicaSet{}, &RiskGroup{}, &Mongod{}, &MongodState{}, &ReplicaSetMember{}, &Problem{}, &MSPError{}, &CommunicationError{}, &SlaveError{})
	if err := createSlaveUtilizationView(db); err != nil {
		panic(err)
	}
	if err := createReplicaSetEffectiveMembersView(db); err != nil {
		panic(err)
	}
	if err := createReplicaSetConfiguredMembersView(db); err != nil {
		panic(err)
	}
}

func createReplicaSetEffectiveMembersView(tx *gorm.DB) error {
	return tx.Exec(`
		DROP VIEW IF EXISTS replica_set_effective_members;
		CREATE VIEW replica_set_effective_members AS
		SELECT r.id as replica_set_id, m.id as mongod_id, s.persistent_storage
		FROM replica_sets r
		JOIN mongods m ON m.replica_set_id = r.id
		JOIN slaves s ON s.id = m.parent_slave_id
		JOIN mongod_states observed ON observed.id = m.observed_state_id
		JOIN mongod_states desired ON desired.id = m.desired_state_id
		WHERE
		observed.execution_state = ` + fmt.Sprintf("%d", MongodExecutionStateRunning) + `
		AND
		desired.execution_state = ` + fmt.Sprintf("%d", MongodExecutionStateRunning) + `;`).Error
}

func createSlaveUtilizationView(tx *gorm.DB) error {
	return tx.Exec(`
		DROP VIEW IF EXISTS slave_utilization;
		CREATE VIEW slave_utilization AS
		SELECT
			*,
			CASE WHEN max_mongods = 0 THEN 1 ELSE current_mongods*1.0/max_mongods END AS utilization,
			(max_mongods - current_mongods) AS free_mongods
		FROM (
			SELECT
				s.*,
				s.mongod_port_range_end - s.mongod_port_range_begin AS max_mongods,
				COUNT(DISTINCT m.id) as current_mongods
			FROM slaves s
			LEFT OUTER JOIN mongods m ON m.parent_slave_id = s.id
			GROUP BY s.id
		);`).Error
}

func createReplicaSetConfiguredMembersView(tx *gorm.DB) error {
	return tx.Exec(`
		DROP VIEW IF EXISTS replica_set_configured_members;
		CREATE VIEW replica_set_configured_members AS
		SELECT r.id as replica_set_id, m.id as mongod_id, s.persistent_storage
		FROM replica_sets r
		JOIN mongods m ON m.replica_set_id = r.id
		JOIN mongod_states desired_state ON m.desired_state_id = desired_state.id
		JOIN slaves s ON m.parent_slave_id = s.id
		WHERE
			s.configured_state != ` + fmt.Sprintf("%d", SlaveStateDisabled) + `
			AND
			desired_state.execution_state NOT IN (` +
		fmt.Sprintf("%d", MongodExecutionStateNotRunning) +
		`, ` + fmt.Sprintf("%d", MongodExecutionStateDestroyed) +
		`);`).Error
}

func RollbackOnTransactionError(tx *gorm.DB, rollbackError *error) {
	switch e := recover(); e {
	case e == gorm.ErrInvalidTransaction:
		log.Printf("ClusterAllocator: rolling back transaction after error: %v", e)
		*rollbackError = tx.Rollback().Error
		if *rollbackError != nil {
			log.Printf("ClusterAllocator: failed rolling back transaction: %v", *rollbackError)
		}
	default:
		panic(e)
	}
}
