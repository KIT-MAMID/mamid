package slave

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/msp"
	"gopkg.in/mgo.v2/bson"
	"strconv"
	"strings"
	"time"
)

const (
	replSetStartup    = 0
	replSetPrimary    = 1
	replSetSecondary  = 2
	replSetRecovering = 3
	replSetUnknown    = 6
	replSetRemoved    = 10
)

type replSetState int

type MongodConfigurator interface {
	MongodConfiguration(p msp.PortNumber) (msp.Mongod, *msp.Error)
	ApplyMongodConfiguration(m msp.Mongod) *msp.Error
	InitiateReplicaSet(m msp.RsInitiateMessage) *msp.Error
}

type ConcreteMongodConfigurator struct {
	MongodSoftShutdownTimeout time.Duration
}

const mongodbAdminDatabase string = "admin"

func (c *ConcreteMongodConfigurator) fetchConfiguration(ctx *mgoContext) (mongod msp.Mongod, err *msp.Error, state replSetState) {

	mongod = msp.Mongod{
		Port: ctx.Port,
	}

	var isMasterRes bson.M
	if err = ctx.IsMaster(&isMasterRes); err != nil {
		return
	}
	if _, exists := isMasterRes["setName"]; !exists {
		return msp.Mongod{
			Port:                    ctx.Port,
			StatusError:             nil,
			LastEstablishStateError: nil,
			State: msp.MongodStateUninitialized,
		}, nil, replSetStartup
	}

	var status bson.M
	state, err = ctx.ReplSetGetStatus(&status)
	if state == replSetRemoved {
		mongod.State = msp.MongodStateRemoved
		return mongod, nil, replSetRemoved
	} else if err != nil {
		return msp.Mongod{}, err, replSetUnknown
	}

	config, err := ctx.ReplSetGetConfig()
	if err != nil {
		return msp.Mongod{}, err, replSetUnknown
	}

	mongod.ReplicaSetConfig.ReplicaSetName = status["set"].(string)

	if status_state, valid := status["myState"]; valid {
		var state msp.MongodState
		switch replSetState(status_state.(int)) {
		case replSetRecovering:
			state = msp.MongodStateRecovering
		case replSetPrimary:
			state = msp.MongodStateRunning
		case replSetSecondary:
			state = msp.MongodStateRunning
		case replSetRemoved:
			state = msp.MongodStateRemoved
		}
		mongod.State = state
	} else {
		return msp.Mongod{}, &msp.Error{
			Identifier:      msp.SlaveGetMongodStatusError,
			Description:     fmt.Sprintf("Mongod on port %d returned no status", ctx.Port),
			LongDescription: fmt.Sprintf("status[myState] does not exist"),
		}, replSetUnknown
	}

	var members []msp.ReplicaSetMember

	if configMembers, valid := config["members"]; valid {
		members = make([]msp.ReplicaSetMember, len(configMembers.([]interface{})))
		for k, member := range configMembers.([]interface{}) {
			pair := strings.Split(member.(bson.M)["host"].(string), ":")
			remotePort, _ := strconv.Atoi(pair[1])
			priority := member.(bson.M)["priority"].(float64)
			members[k] = msp.ReplicaSetMember{
				HostPort: msp.HostPort{Hostname: pair[0], Port: msp.PortNumber(remotePort)},
				Priority: priority,
			}
		}
	} else {
		log.Errorf("No members list in rs config")
	}
	mongod.ReplicaSetConfig.ReplicaSetMembers = members

	var shardingRole msp.ShardingRole
	if configsvr, valid := config["configsvr"]; valid && configsvr.(bool) == true {
		shardingRole = msp.ShardingRoleConfigServer
	} else {
		// Fall back to parsing command line options
		cmdLineShardingRole, err := ctx.ParseCmdLineShardingRole()
		if err != nil {
			return msp.Mongod{}, err, replSetUnknown
		}

		switch cmdLineShardingRole {
		case "shardsvr":
			shardingRole = msp.ShardingRoleShardServer
		case "configsvr":
			shardingRole = msp.ShardingRoleConfigServer
		default:
			shardingRole = msp.ShardingRoleNone
		}
	}
	mongod.ReplicaSetConfig.ShardingRole = shardingRole

	return mongod, nil, replSetState(status["myState"].(int))
}

func (c *ConcreteMongodConfigurator) MongodConfiguration(port msp.PortNumber) (msp.Mongod, *msp.Error) {

	// TODO get credential from store filled by EstablishState
	// connect unauthenticated in case the replica set is not initialized
	ctx, err := c.connect(port, "r1", msp.MongodCredential{"mamid", "mamid"})
	if err != nil {
		return msp.Mongod{}, err
	}
	defer ctx.Close()

	mongod, err, _ := c.fetchConfiguration(ctx)
	return mongod, err
}

type mongodMembers []msp.ReplicaSetMember

func (m mongodMembers) Len() int {
	return len(m)
}

func (m mongodMembers) Less(i, j int) bool {
	diff := m[i].HostPort.Port - m[j].HostPort.Port
	if diff < 0 {
		return true
	}
	if diff > 0 {
		return false
	}
	return m[i].HostPort.Hostname < m[j].HostPort.Hostname
}
func (m mongodMembers) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (c *ConcreteMongodConfigurator) ApplyMongodConfiguration(m msp.Mongod) *msp.Error {
	ctx, err := c.connect(m.Port, m.ReplicaSetConfig.ReplicaSetName, m.ReplicaSetConfig.RootCredential)
	if err != nil {
		return err
	}
	defer ctx.Close()

	//current, err, state := c.fetchConfiguration(sess, m.Port) TODO What is this for?
	//if err != nil {
	//	return err
	//}
	//
	//sort.Sort(mongodMembers(current.ReplicaSetConfig.ReplicaSetMembers))
	//sort.Sort(mongodMembers(m.ReplicaSetConfig.ReplicaSetMembers))

	log.Debugf("Applying Mongod configuration: %#v", m)

	var isMasterRes bson.M
	if err = ctx.IsMaster(&isMasterRes); err != nil {
		return err
	}
	isMaster := isMasterRes["ismaster"] == true

	if isMaster {
		log.Debugf("Mongod on port `%d` is PRIMARY of its ReplicaSet", m.Port)
	} else {
		log.Debugf("Mongod on port `%d` is NOT PRIMARY of its ReplicaSet", m.Port)
	}

	if m.State == msp.MongodStateDestroyed {

		var status bson.M
		replSetState, err := ctx.ReplSetGetStatus(&status)

		if err != nil {
			if replSetState == replSetRemoved {
				//Mongod was removed by the primary so it can be shut down
				err = ctx.ShutdownWithTimeout(int64(c.MongodSoftShutdownTimeout.Seconds()))
			}
			return err
		} else {
			if isMaster {
				//Cant remove ourselves so somebody else has to become master and remove us
				log.Debugf("Letting Mongod on port `%d` step down to have it be removed by the new PRIMARY", m.Port)
				return ctx.ReplSetStepDown(120)
			} else {
				//Wait for this Mongod to be removed by the primary
				log.Debugf("Waiting for Mongod on port `%d` to be removed by the PRIMARY")
				return nil
			}
		}

	} else if m.State == msp.MongodStateNotRunning {

		//Temporary maintenance - just shut down without removing from replica set
		log.Debugf("shutting down Mongod: %f", m)
		return ctx.ShutdownWithTimeout(int64(c.MongodSoftShutdownTimeout.Seconds()))

	} else if m.State == msp.MongodStateRunning {
		if isMaster {

			config, err := ctx.ReplSetGetConfig()
			if err != nil {
				return err
			}

			config, updateErr := updateConfig(config, m)
			if updateErr != nil {
				return updateErr
			}

			log.Debugf("`replSetReconfig` ReplicaSet `%s` from its PRIMARY Mongod on port `%d`: %#v",
				m.ReplicaSetConfig.ReplicaSetName, m.Port, config)

			ctx.ReplSetReconfig(config)

		}

		return nil
	}

	return &msp.Error{
		Identifier:      msp.SlaveMongodProtocolError,
		Description:     "Protocol error",
		LongDescription: fmt.Sprintf("Unexpected msp.Mongod.State value %s received", m.State),
	}
}

func updateConfig(currentConfig bson.M, m msp.Mongod) (bson.M, *msp.Error) {
	config := currentConfig

	config["_id"] = m.ReplicaSetConfig.ReplicaSetName

	resultingMembers, updateListErr := updateMembersList(config, m.ReplicaSetConfig.ReplicaSetMembers)
	if updateListErr != nil {
		return bson.M{}, updateListErr
	}
	config["members"] = resultingMembers

	version := 0
	if ver, valid := config["version"].(int); valid {
		version = ver
	}
	config["version"] = version + 1 // Defaults to 1 if no version is set in currentConfig

	config["configsvr"] = m.ReplicaSetConfig.ShardingRole == msp.ShardingRoleConfigServer

	return config, nil
}

func updateMembersList(currentConfig bson.M, desiredMembers []msp.ReplicaSetMember) ([]bson.M, *msp.Error) {
	//Update config members list
	//Only use ids not used before for new members
	//https://docs.mongodb.com/manual/reference/replica-configuration/#rsconf.members[n]._id
	usedIds := make(map[int]bool)
	reportedMembersByHostPortString := make(map[string]bson.M)

	if currentMembers, valid := currentConfig["members"]; valid {
		log.Debugf("members is %#v", currentMembers)
		for _, value := range currentMembers.([]interface{}) {
			member := value.(bson.M)
			usedIds[member["_id"].(int)] = true
			reportedMembersByHostPortString[member["host"].(string)] = member
		}
	}

	var resultingMembers []bson.M

	for _, desiredMember := range desiredMembers {

		hostPortString := fmt.Sprintf("%s:%d", desiredMember.HostPort.Hostname, desiredMember.HostPort.Port)

		member, ok := reportedMembersByHostPortString[hostPortString]
		if !ok {
			freeId := 0
			found := false
			// Create it
			for j := 0; j < 256; j++ {
				if _, used := usedIds[j]; !used {
					freeId = j
					found = true
					usedIds[j] = true
					break
				}
			}
			if !found {
				return []bson.M{}, &msp.Error{
					Identifier:      msp.SlaveReplicaSetConfigError,
					Description:     fmt.Sprintf("Could not find free member `_id`"),
					LongDescription: fmt.Sprintf("No free member `_id` left for ReplicaSetMembers of ReplicaSet of Mongod"),
				}
			}
			member = bson.M{"_id": freeId}

		}

		member["host"] = hostPortString
		member["priority"] = desiredMember.Priority

		resultingMembers = append(resultingMembers, member)

	}
	return resultingMembers, nil
}

func (c *ConcreteMongodConfigurator) InitiateReplicaSet(m msp.RsInitiateMessage) *msp.Error {
	// connect unauthenticated in case the replica set is not initialized
	ctx, mspErr := c.connect(m.Port, m.ReplicaSetConfig.ReplicaSetName, m.ReplicaSetConfig.RootCredential)
	if mspErr != nil {
		return mspErr
	}
	defer ctx.Close()

	members := make([]bson.M, len(m.ReplicaSetConfig.ReplicaSetMembers))
	for k, member := range m.ReplicaSetConfig.ReplicaSetMembers {
		members[k] = bson.M{"_id": k, "host": fmt.Sprintf("%s:%d", member.HostPort.Hostname, member.HostPort.Port)}
	}

	config, updateErr := updateConfig(bson.M{}, msp.Mongod{
		Port:             m.Port,
		ReplicaSetConfig: m.ReplicaSetConfig,
	})
	if updateErr != nil {
		return updateErr
	}

	log.Debugf("CONFIG %#v", config)

	alreadyInitialized, err := ctx.ReplSetInitiate(config, true)
	if err != nil {
		return err
	} else if alreadyInitialized {
		return nil
	}

	// Create root user on admin database
	// TODO make this idempotent
	err = ctx.CreateUser(
		m.ReplicaSetConfig.RootCredential.Username,
		m.ReplicaSetConfig.RootCredential.Password,
		"MAMID administrative user",
		[]string{"root"},
	)

	return err

}
