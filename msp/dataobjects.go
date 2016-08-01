package msp

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
	StatusError             *SlaveError
	LastEstablishStateError *SlaveError
	State                   MongodState
}

type Error interface {
}

type SlaveError struct {
	Identifier      string
	Description     string
	LongDescription string
}

type CommunicationError struct {
	Message string
}