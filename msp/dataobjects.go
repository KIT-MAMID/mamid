package msp

type MongodState string

const (
	MongodStateDestroyed  = "destroyed"
	MongodStateNotRunning = "notrunning"
	MongodStateRecovering = "recovering"
	MongodStateRunning    = "running"
)

type HostPort struct {
	Hostname string
	Port     uint16
}

type Mongod struct {
	Port                    uint16
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

type PortNumber uint16
