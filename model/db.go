package model

import (
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
	RiskGroupID uint
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
	ObservationErrorID uint

	LastEstablishStateError   MSPError
	LastEstablishStateErrorID uint

	ParentSlave   *Slave
	ParentSlaveID uint

	ReplicaSet   *ReplicaSet
	ReplicaSetID uint

	DesiredState   MongodState
	DesiredStateID uint

	ObservedState   MongodState
	ObservedStateID uint
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
	MongodStateID uint
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
	MSPErrorID uint
}

type SlaveError struct {
	msp.SlaveError
	ID         uint `gorm:"primary_key"`
	MSPErrorID uint
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
	SlaveID uint

	ReplicaSet   *ReplicaSet
	ReplicaSetID uint

	Mongod   *Mongod
	MongodID uint
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
