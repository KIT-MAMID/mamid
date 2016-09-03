package master

import (
	"fmt"
	. "github.com/KIT-MAMID/mamid/model"
	"github.com/KIT-MAMID/mamid/msp"
	"github.com/Sirupsen/logrus"
	"github.com/jinzhu/gorm"
)

var deployerLog = logrus.WithField("module", "deployer")

/*
  Listens on the bus for state mismatches and tries to solve them by pushing the desired state to the Mongod
*/
type Deployer struct {
	DB             *DB
	MSPClient      msp.MSPClient
	BusReadChannel <-chan interface{}
}

func (d *Deployer) Run() {

	for {
		msg := <-d.BusReadChannel
		switch msg.(type) {
		case MongodMatchStatus:
			go d.handleMatchStatus(msg.(MongodMatchStatus))
		case ReplicaSetInitiationStatus:
			go d.handleReplicaSetInitiationStatus(msg.(ReplicaSetInitiationStatus))
		}

	}
}

func (d *Deployer) handleMatchStatus(m MongodMatchStatus) {
	if !m.Mismatch {
		return
	}
	d.pushMongodState(m.Mongod)
}

func (d *Deployer) handleReplicaSetInitiationStatus(s ReplicaSetInitiationStatus) {

	if s.Initiated {
		return
	}

	var msg msp.RsInitiateMessage
	var mspErr *msp.Error

	tx := d.DB.Begin()

	slave, initiator, err := d.findInitiatorForReplicaSet(tx, s.ReplicaSet)
	if err != nil {
		deployerLog.WithError(err).Errorf("could not find initiator for replica set `%s`", s.ReplicaSet.Name)
		goto rollbackAndReturn
	}

	msg = msp.RsInitiateMessage{
		Port: msp.PortNumber(initiator.Port),
	}

	msg.ReplicaSetConfig, err = d.replicaSetConfig(tx, s.ReplicaSet)
	if err != nil {
		deployerLog.WithError(err).Errorf("could not generate replica set config `%s`", s.ReplicaSet.Name)
		goto rollbackAndReturn
	}

	deployerLog.Debugf("initializing Replica Set `%s` from `%s` using message: %#v", s.ReplicaSet.Name, slave.Hostname, msg)

	mspErr = d.MSPClient.InitiateReplicaSet(msp.HostPort{slave.Hostname, msp.PortNumber(slave.Port)}, msg)

	if mspErr != nil {
		deployerLog.Errorf("error initializing Replica Set `%s` from `%s`: %s", s.ReplicaSet.Name, slave.Hostname, mspErr)
		goto rollbackAndReturn
	} else {

		if err = tx.Model(&s.ReplicaSet).Update("Initiated", true).Error; err != nil {
			deployerLog.Errorf("error initializing Replica Set `%s` from `%s`: %s", s.ReplicaSet.Name, slave.Hostname, mspErr)
			goto rollbackAndReturn
		}
		tx.Commit()
	}

	return

rollbackAndReturn:
	tx.Rollback()
	return

}

// Find a Mongod of ReplicaSet r for `initiiating` the ReplicaSet
func (d *Deployer) findInitiatorForReplicaSet(tx *gorm.DB, r ReplicaSet) (Slave, Mongod, error) {
	_, initiator, err := DesiredMSPReplicaSetMembersForReplicaSetID(tx, r.ID)
	if err != nil {
		return Slave{}, Mongod{}, err
	}
	if initiator.ID == 0 {
		return Slave{}, Mongod{}, fmt.Errorf("Could not find initiator - Is replica set empty?")
	}

	var s Slave
	if err := tx.Model(&initiator).Related(&s, "ParentSlaveID").Error; err != nil {
		return Slave{}, Mongod{}, err
	}

	deployerLog.Debugf("Found initiator for Replica Set %s: %#v", r, initiator)

	return s, initiator, nil
}

func (d *Deployer) pushMongodState(mongod Mongod) {

	deployerLog.Debugf("fetch Mongod state representation: `%d` on slave `%d`", mongod.ID, mongod.ParentSlaveID)
	// Readonly tx
	tx := d.DB.Begin()
	defer tx.Rollback()

	hostPort, mspMongod, err := d.mspMongodStateRepresentation(tx, mongod)
	if err != nil {
		deployerLog.Println(err)
	}
	// Readonly tx
	tx.Rollback()
	deployerLog.Debugf("finish fetching Mongod state representation: `%d` on slave `%d`", mongod.ID, mongod.ParentSlaveID)

	deployerLog.Debugf("establishing Mongod state on `%s` (%#v)", hostPort, mspMongod)

	mspError := d.MSPClient.EstablishMongodState(hostPort, mspMongod)
	if mspError != nil {
		deployerLog.Errorf("MSP error establishing mongod state on `%s` for Mongod `(%v(id=%d),%d,)` in Replica Set `%s`: %s",
			hostPort, mongod.ParentSlave, mongod.ParentSlaveID, mongod.Port, mongod.ReplSetName, mspError)
	} else {
		deployerLog.Debugf("finished establishing Mongod state on %s", hostPort)
	}

}

// Generate an MSP-compatible representation of the deisred Mongod state
// uses tx readonly
// When err != nil is returned, the tx. should be rolled back and the error be reported
func (d *Deployer) mspMongodStateRepresentation(tx *gorm.DB, mongod Mongod) (hostPort msp.HostPort, mspMongod msp.Mongod, err error) {

	var slave Slave
	var desiredState MongodState
	var shardingRole msp.ShardingRole
	var mspMongodState msp.MongodState
	var replicaSetMembers []msp.ReplicaSetMember

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

	if !mongod.ReplicaSetID.Valid {
		replicaSetMembers = make([]msp.ReplicaSetMember, 0, 0)
	} else {
		if replicaSetMembers, _, err = DesiredMSPReplicaSetMembersForReplicaSetID(tx, mongod.ReplicaSetID.Int64); err != nil {
			return
		}
	}

	shardingRole, err = ProjectModelShardingRoleToMSPShardingRole(desiredState.ShardingRole)
	if err != nil {
		return
	}

	var managementCredential msp.MongodCredential

	if managementCredential, err = d.managementMongodCredential(tx); err != nil {
		return
	}

	var keyfileContents string
	if keyfileContents, err = d.keyfileContents(tx); err != nil {
		return
	}

	// Construct msp representation
	hostPort = msp.HostPort{
		Hostname: slave.Hostname,
		Port:     msp.PortNumber(slave.Port),
	}
	mspMongod = msp.Mongod{
		Port: msp.PortNumber(mongod.Port),
		ReplicaSetConfig: msp.ReplicaSetConfig{
			ReplicaSetName:    mongod.ReplSetName,
			ReplicaSetMembers: replicaSetMembers,
			ShardingRole:      shardingRole,
			RootCredential:    managementCredential,
		},
		KeyfileContent: keyfileContents,
		State:          mspMongodState,
	}

	return

}

// Generate a ReplicaSetConfig used to describe the ReplicaSet r
func (d *Deployer) replicaSetConfig(tx *gorm.DB, r ReplicaSet) (config msp.ReplicaSetConfig, err error) {

	var shardingRole msp.ShardingRole
	shardingRole, err = ProjectModelShardingRoleToMSPShardingRole(r.ShardingRole)
	if err != nil {
		return config, err
	}

	var managementCredential msp.MongodCredential
	managementCredential, err = d.managementMongodCredential(tx)
	if err != nil {
		return config, err
	}

	config = msp.ReplicaSetConfig{
		ReplicaSetName:    r.Name,
		ReplicaSetMembers: make([]msp.ReplicaSetMember, 0, 0),
		ShardingRole:      shardingRole,
		RootCredential:    managementCredential,
	}

	config.ReplicaSetMembers, _, err = DesiredMSPReplicaSetMembersForReplicaSetID(tx, r.ID)
	if err != nil {
		return config, err
	}

	return
}

func (d *Deployer) managementMongodCredential(tx *gorm.DB) (credential msp.MongodCredential, err error) {

	var rootCredential MongodbCredential
	res := tx.Table("mongodb_root_credentials").First(&rootCredential)
	switch { // Assume there is at most one
	case res.Error != nil && !res.RecordNotFound():
		return credential, res.Error
	case res.Error == nil:
		// Do nothing, assume already created
		return ProjectModelMongodbCredentialToMSPMongodCredential(rootCredential), nil
	}

	return

}

func (d *Deployer) keyfileContents(tx *gorm.DB) (keyfileContents string, err error) {
	var keyfile MongodKeyfile
	res := tx.First(&keyfile)
	if res.Error != nil {
		return "", res.Error
	} else {
		return keyfile.Content, nil
	}
}
