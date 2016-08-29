package msp

import (
	"fmt"
)

type MongodState string

const (
	MongodStateDestroyed  = "destroyed"
	MongodStateNotRunning = "notrunning"
	MongodStateRecovering = "recovering"
	MongodStateRunning    = "running"
)

type PortNumber uint16

type HostPort struct {
	Hostname string
	Port     PortNumber
}

type ReplicaSetConfig struct {
	ReplicaSetName       string
	ReplicaSetMembers    []ReplicaSetMember
	ShardingConfigServer bool
}

type ReplicaSetMember struct {
	HostPort HostPort
	Priority uint
}

type Mongod struct {
	Port                    PortNumber
	ReplicaSetConfig        ReplicaSetConfig
	StatusError             *Error
	LastEstablishStateError *Error
	State                   MongodState
}

type RsInitiateMessage struct {
	Port             PortNumber
	ReplicaSetConfig ReplicaSetConfig
}

type Error struct {
	// See constants in this package for list of identifiers
	Identifier      string
	Description     string
	LongDescription string
}

func (e Error) String() string {
	return fmt.Sprintf("{%s: %s : %s}", e.Identifier, e.Description, e.LongDescription)
}

func (e *Error) validateFields() error {

	validationError := func(fieldname string) error {
		return fmt.Errorf("invalid msp.Error: `%s` is a mandatory field", fieldname)
	}

	if e.Identifier == "" {
		return validationError("Identifier")
	}

	return nil
}

// List of Error identifiers
const CommunicationError string = "COMM" // slave is unreachable or slave response not understood
const BadStateDescription string = "BADSTATE"
const SlaveSpawnError string = "SLAVESPAWN"
const SlaveConnectMongodError string = "SLAVECONERR"
const SlaveGetMongodStatusError string = "SLAVEGETSTATUSERR"
const SlaveReplicaSetInitError string = "SLAVEREPLSETINIT"
const SlaveReplicaSetConfigError string = "SLAVEREPLSETCONFIG"
const SlaveMongodProtocolError string = "SLAVEMONGODPROTOERR"
const NotImplementedError string = "NOTIMPLEMENTED"
