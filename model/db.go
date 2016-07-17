package model

import (
	"github.com/KIT-MAMID/mamid/msp"
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
	ID                   uint `gorm:"primary_key"`
	Hostname             string
	Port                 PortNumber
	MongodPortRangeBegin PortNumber
	MongodPortRangeEnd   PortNumber
	PersistentStorage    bool
	Mongods              []*Mongod
}

type PortNumber uint16

type SlaveState uint

const (
	_                           = iota
	SlaveStateActive SlaveState = 1
	SlaveStateMaintenance
	SlaveStateDisabled
)

type ReplicaSet struct {
	ID                              uint `gorm:"primary_key"`
	Name                            string
	PersistentMemberCount           uint
	VolatileMemberCount             uint
	ConfigureAsShardingConfigServer bool
	Mongods                         []*Mongod
}

type RiskGroup struct {
	ID     uint `gorm:"primary_key"`
	Name   string
	Slaves []*Slave
}

type Mongod struct {
	// TODO missing UNIQUE constraint
	Port                    PortNumber `gorm:"primary_key"`
	ReplSetName             string     `gorm:"primary_key"`
	ObservationError        *msp.Error
	LastEstablishStateError *msp.SlaveError
	ObservedState           *MongodState
	ReplicaSet              *ReplicaSet
	ParentSlave             *Slave `gorm:"primary_key"`
}

type MongodState struct {
	// TODO missing primary key. Auto inc? Need garbage collection in app
	IsShardingConfigServer bool
	ExecutionState         MongodExecutionState
	ReplicaSetMembers      []HostPort
}

type MongodExecutionState uint

const (
	_                                                  = iota
	MongodExecutionStateDestroyed MongodExecutionState = 1
	MongodExecutionStateNotRunning
	MongodExecutionStateRecovering // invalid for a desired MongodState
	MongodExecutionStateRunning
)

type HostPort struct {
	// TODO missing primary key.
	Hostname string
	Port     PortNumber
}
