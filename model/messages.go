package model

import "github.com/KIT-MAMID/mamid/msp"

type StatusMessage interface {
}

type ConnectionStatus struct {
	Unreachable        bool
	Slave              Slave
	CommunicationError msp.Error // Only valid if Unreachable=true
}

type MongodMatchStatus struct {
	Mismatch bool
	Mongod   Mongod
}

type DesiredReplicaSetConstraintStatus struct {
	Unsatisfied               bool
	ReplicaSet                ReplicaSet
	ConfiguredVolatileCount   uint
	ConfiguredPersistentCount uint
}

type ObservedReplicaSetConstraintStatus struct {
	Unsatisfied               bool
	ReplicaSet                ReplicaSet
	ConfiguredVolatileCount   uint
	ConfiguredPersistentCount uint
	ActualVolatileCount       uint
	ActualPersistentCount     uint
}
