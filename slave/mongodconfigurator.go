package slave

import (
	"github.com/KIT-MAMID/mamid/msp"
	"gopkg.in/mgo.v2"
	"fmt"
	"gopkg.in/mgo.v2/bson"
	"strings"
	"sort"
	"strconv"
)

type replSetMemberStatus struct {
	name string `bson:"name"`
	state int `bson:"state"`
}
type replSetStatus struct {
	set string `bson:"set"`
	state replSetState `bson:"myState"`
	members []replSetMemberStatus `bson:"members"`
}

type replSetInitiateMember struct{
	id int `bson:"_id"`
	host string `bson:"host"`
}

const (
	replSetStartup = 0
	replSetPrimary = 1
	replSetSecondary = 2
	replSetRecovering = 3
	replSetUnknown = 6
)
type replSetState int

type MongodConfigurator interface {
	MongodConfiguration(p msp.PortNumber) (msp.Mongod, *msp.Error)
	ApplyMongodConfiguration(m msp.Mongod) *msp.Error
}

type ConcreteMongodConfigurator struct {
	dial func(url string) (*mgo.Session, error)
}

func (c *ConcreteMongodConfigurator) connect(port msp.PortNumber) (*mgo.Session, *msp.Error) {
	sess, err := mgo.Dial(fmt.Sprintf("mongo://127.0.0.1:%d/?connect=direct", port))

	if err != nil {
		return nil, &msp.Error{
			Identifier: fmt.Sprintf("conn_%d", port),
			Description: fmt.Sprintf("Establishing a connection to mongod instance on port %d failed", port),
			LongDescription: fmt.Sprintf("mgo.Dial() failed with\n%s", err.Error()),
		}
	}

	return sess, nil
}

func (c *ConcreteMongodConfigurator) fetchConfiguration(sess *mgo.Session, port msp.PortNumber) (msp.Mongod, *msp.Error, replSetState) {
	var status replSetStatus
	if err := sess.Run("replSetGetStatus", &status); err != nil {
		return msp.Mongod{}, &msp.Error{
			Identifier: fmt.Sprintf("conn_%d", port),
			Description: fmt.Sprintf("Getting replica set status information from mongod instance on port %d failed", port),
			LongDescription: fmt.Sprintf("mgo/Session.Run() failed with\n%s", err.Error()),
		}, replSetUnknown
	}

	members := make([]msp.HostPort, len(status.members))
	for k, member := range status.members {
		pair := strings.Split(member.name, ":")
		remotePort, _ := strconv.Atoi(pair[1])
		members[k] = msp.HostPort{pair[0], msp.PortNumber(remotePort) }
	}

	var state msp.MongodState
	if status.state == replSetRecovering {
		state = msp.MongodStateRecovering
	} else {
		state = msp.MongodStateRunning
	}

	return msp.Mongod{
		Port: port,
		ReplicaSetName: status.set,
		ReplicaSetMembers: members,
		ShardingConfigServer: false,
		StatusError: nil,
		LastEstablishStateError: nil,
		State: state,
	}, nil, status.state
}

func (c *ConcreteMongodConfigurator) MongodConfiguration(port msp.PortNumber) (msp.Mongod, *msp.Error) {
	sess, err := c.connect(port)
	if err != nil {
		return msp.Mongod{}, err
	}

	mongod, err, _ := c.fetchConfiguration(sess, port)
	return mongod, err
}

type mongodMembers []msp.HostPort

func (m mongodMembers) Len() int {
	return len(m)
}
func (m mongodMembers) Less(i, j int) bool {
	diff := m[i].Port - m[j].Port
	if diff < 0 {
		return true
	}
	if diff > 0 {
		return false
	}
	return m[i].Hostname < m[j].Hostname
}
func (m mongodMembers) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (c *ConcreteMongodConfigurator) ApplyMongodConfiguration(m msp.Mongod) *msp.Error {
	sess, err := c.connect(m.Port)
	if err != nil {
		return err
	}

	current, err, state := c.fetchConfiguration(sess, m.Port)
	if err != nil {
		return err
	}

	sort.Sort(mongodMembers(current.ReplicaSetMembers))
	sort.Sort(mongodMembers(m.ReplicaSetMembers))

	if m.State == msp.MongodStateDestroyed {
		var result interface{}
		sess.Run(bson.M{ "shutdown": 1, "timeoutSecs": MongodSoftShutdownTimeout }, result)
		return nil // shutdown never errors ... We'll just try to force kill the process after another timeout
	}

	if m.State == msp.MongodStateRunning {
		if state == replSetStartup {
			var result interface{}
			err := sess.Run("replSetInitiate", &result)
			if err != nil {
				return &msp.Error{
					Identifier: fmt.Sprintf("replsetinit_%d", m.Port),
					Description: fmt.Sprintf("Replica set %s could not be initiated on instance on port %d", m.ReplicaSetName, m.Port),
					LongDescription: fmt.Sprintf("Command replSetInitiate failed with\n%s", err.Error()),
				}
			}
		}

		var members []replSetInitiateMember = make([]replSetInitiateMember, len(m.ReplicaSetMembers))
		for k, member := range m.ReplicaSetMembers {
			members[k].id = k
			members[k].host = fmt.Sprintf("%s:%d", member.Hostname, member.Port)
		}

		var result interface{}
		cmd := bson.M{"replSetReconfig": bson.M{ "_id": m.ReplicaSetName, "members": &members }, "force": true}
		err := sess.Run(cmd, &result)
		if err != nil {
			return &msp.Error{
				Identifier: fmt.Sprintf("replsetreconfig_%d", m.Port),
				Description: fmt.Sprintf("Replica set %s could not be reconfigured with replicaset members on instance on port %d", m.ReplicaSetName, m.Port),
				LongDescription: fmt.Sprintf("Command %v failed with\n%s", cmd, err.Error()),
			}
		}


		return nil
	}

	return &msp.Error{
		Identifier: "protocol",
		Description: "Protocol error",
		LongDescription: fmt.Sprintf("Invalid msp.Mongod.State value %d received", m.State),
	}
}
