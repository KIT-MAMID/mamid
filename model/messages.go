package model

import "github.com/KIT-MAMID/mamid/msp"

type StatusMessage interface {
}

type ConnectionStatus struct {
	Unreachable        bool
	Slave              Slave
	CommunicationError msp.CommunicationError
}

type MongodMatchStatus struct {
	Mismatch bool
	Mongod   Mongod
}

type DesiredReplicaSetConstraintStatus struct {
	Unsatisfied           bool
	ReplicaSet            ReplicaSet
	ActualVolatileCount   uint
	ActualPersistentCount uint
}

type ObservedReplicaSetConstraintStatus struct {
	Unsatisfied           bool
	ReplicaSet            ReplicaSet
	ActualVolatileCount   uint
	ActualPersistentCount uint
}
