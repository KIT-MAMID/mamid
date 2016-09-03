package msp

import (
	"fmt"
)

type MongodState string

const (
	MongodStateForceDestroyed = "forcedestroyed"
	MongodStateDestroyed      = "destroyed"
	MongodStateNotRunning     = "notrunning"
	MongodStateUninitialized  = "uninitiated"
	MongodStateRecovering     = "recovering"
	MongodStateRunning        = "running"
	MongodStateRemoved        = "removed"
)

func (s MongodState) IsProcessExecuting() bool {
	switch s {
	case MongodStateForceDestroyed, MongodStateDestroyed, MongodStateNotRunning:
		return false
	default:
		return true
	}
}

type PortNumber uint16

type HostPort struct {
	Hostname string
	Port     PortNumber
}

type ShardingRole string

const (
	ShardingRoleNone         ShardingRole = "none"
	ShardingRoleShardServer  ShardingRole = "shardsvr"
	ShardingRoleConfigServer ShardingRole = "configsvr"
)

type ReplicaSetConfig struct {
	ReplicaSetName    string
	ReplicaSetMembers []ReplicaSetMember
	ShardingRole      ShardingRole
	RootCredential    MongodCredential // user with the Superuse Role `root`, see https://docs.mongodb.com/manual/reference/built-in-roles/#root
}

type MongodCredential struct {
	Username, Password string
}

func (m MongodCredential) GoString() string {
	return fmt.Sprintf("msp.MongodCredential{Username:\"%s\", Password:\"<redacted>\"}", m.Username)
}

type ReplicaSetMember struct {
	HostPort HostPort
	Priority float64
	Votes    int
}

type Mongod struct {
	Port                    PortNumber
	KeyfileContent          string
	ReplicaSetConfig        ReplicaSetConfig
	StatusError             *Error
	LastEstablishStateError *Error
	State                   MongodState
}

func (m Mongod) GoString() string {
	return fmt.Sprintf("msp.Mongod{Port: %d, KeyfileContent:\"<redacted>\", ReplicaSetConfig:%#v, StatusError:%#v, LastEstablishStateError:%#v, State:%#v}",
		m.Port, m.ReplicaSetConfig, m.StatusError, m.LastEstablishStateError, m.State)
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
const SlaveReplicaSetCreateRootUserError string = "SLAVEREPLSETCREATEROOTUSER"
const SlaveReplicaSetConfigError string = "SLAVEREPLSETCONFIG"
const SlaveMongodProtocolError string = "SLAVEMONGODPROTOERR"
const NotImplementedError string = "NOTIMPLEMENTED"
const SlaveShutdownError string = "SLAVESHUTDOWNERR"
