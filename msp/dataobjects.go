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

type Mongod struct {
	Port                    PortNumber
	ReplicaSetName          string
	ReplicaSetMembers       []HostPort
	ShardingConfigServer    bool
	StatusError             *Error
	LastEstablishStateError *Error
	State                   MongodState
}

type Error struct {
	// See constants in this package for list of identifiers
	Identifier      string
	Description     string
	LongDescription string
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
