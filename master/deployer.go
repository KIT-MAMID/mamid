package master

import (
	"fmt"
	. "github.com/KIT-MAMID/mamid/model"
	"github.com/KIT-MAMID/mamid/msp"
	"github.com/jinzhu/gorm"
	"log"
)

/*
  Listens on the bus for state mismatches and tries to solve them by pushing the desired state to the Mongod
*/
type Deployer struct {
	DB             *gorm.DB
	MSPClient      msp.MSPClient
	BusReadChannel <-chan interface{}
}

func (d *Deployer) Run() {

	for {
		msg := <-d.BusReadChannel
		switch msg.(type) {
		case MongodMatchStatus:
			d.handleMatchStatus(msg.(MongodMatchStatus))
		}
	}
}

func (d *Deployer) handleMatchStatus(m MongodMatchStatus) {
	if !m.Mismatch {
		return
	}
	d.pushMongodState(m.Mongod)
}

func (d *Deployer) pushMongodState(mongod Mongod) {

	// Readonly tx
	tx := d.DB.Begin()

	hostPort, mspMongod, err := d.mspMongodStateRepresentation(tx, mongod)
	if err != nil {
		log.Println(err)
	}
	// Readonly tx
	tx.Rollback()

	mspError := d.MSPClient.EstablishMongodState(hostPort, mspMongod)
	if mspError != nil {
		log.Printf("deployer: MSP error establishing mongod state for Mongod `(%s(id=%s),%s,)`: %s",
			mongod.ParentSlave, mongod.ParentSlaveID, mongod.Port, mongod.ReplSetName, mspError)
	}

}

// Generate an MSP-compatible representation of the deisred Mongod state
// uses tx readonly
// When err != nil is returned, the tx. should be rolled back and the error be reported
func (d *Deployer) mspMongodStateRepresentation(tx *gorm.DB, mongod Mongod) (hostPort msp.HostPort, mspMongod msp.Mongod, err error) {

	var slave *Slave
	var desiredState *MongodState
	var mspMongodState msp.MongodState

	// Fetch master representation
	if err = tx.Model(&mongod).Related(&slave, "ParentSlave").Error; err != nil {
		return
	}
	if err = tx.Model(&mongod).Related(&desiredState, "DesiredState").Error; err != nil {
		return
	}
	mspMongodState, err = mspMongodStateFromExecutionState(desiredState.ExecutionState)
	if err != nil {
		return
	}

	// Construct msp representation
	hostPort = msp.HostPort{
		Hostname: slave.Hostname,
		Port:     uint16(slave.Port),
	}
	mspMongod = msp.Mongod{
		Port:                 uint16(mongod.Port),
		ReplicaSetName:       mongod.ReplSetName,
		ReplicaSetMembers:    []msp.HostPort{}, // TODO
		ShardingConfigServer: desiredState.IsShardingConfigServer,
		State:                mspMongodState,
	}

	return

}

func mspMongodStateFromExecutionState(s MongodExecutionState) (msp.MongodState, error) {
	switch s {
	case MongodExecutionStateDestroyed:
		return msp.MongodStateDestroyed, nil
	case MongodExecutionStateNotRunning:
		return msp.MongodStateNotRunning, nil
	case MongodExecutionStateRecovering:
		return msp.MongodStateRecovering, nil
	case MongodExecutionStateRunning:
		return msp.MongodStateRunning, nil
	default:
		return "", fmt.Errorf("deployer: unable to map `%v` from model.ExecutionState to msp.MongodState", s)
	}
}
