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
	sess, err := c.dial(fmt.Sprintf("mongodb://127.0.0.1:%d/?connect=direct", port))

/*
	mgo.SetDebug(true)

	var aLogger *log.Logger
	aLogger = log.New(os.Stderr, "", log.LstdFlags)
	mgo.SetLogger(aLogger)
*/

	if err != nil {
		return nil, &msp.Error{
			Identifier: fmt.Sprintf("conn_%d", port),
			Description: fmt.Sprintf("Establishing a connection to mongod instance on port %d failed", port),
			LongDescription: fmt.Sprintf("ConcreteMongodConfigurator.dial() failed with\n%s", err.Error()),
		}
	}
	sess.SetMode(mgo.Monotonic, true)

	return sess, nil
}

func (c *ConcreteMongodConfigurator) fetchConfiguration(sess *mgo.Session, port msp.PortNumber) (msp.Mongod, *msp.Error, replSetState) {
	running := bson.M{}
	if err := sess.Run("isMaster", &running); err != nil {
		return msp.Mongod{}, &msp.Error{
			Identifier: fmt.Sprintf("getstatus_%d", port),
			Description: fmt.Sprintf("Getting master information from mongod instance on port %d failed", port),
			LongDescription: fmt.Sprintf("mgo/Session.Run(\"isMaster\") failed with\n%s", err.Error()),
		}, replSetUnknown
	}

	if _, exists := running["setName"]; !exists {
		return msp.Mongod{
			Port: port,
			ReplicaSetName: "",
			ReplicaSetMembers: nil,
			ShardingConfigServer: false,
			StatusError: nil,
			LastEstablishStateError: nil,
			State: msp.MongodStateNotRunning,
		}, nil, replSetStartup
	}

	status := bson.M{}
	if err := sess.Run("replSetGetStatus", &status); err != nil {
		return msp.Mongod{}, &msp.Error{
			Identifier: fmt.Sprintf("getstatus_%d", port),
			Description: fmt.Sprintf("Getting replica set status information from mongod instance on port %d failed", port),
			LongDescription: fmt.Sprintf("mgo/Session.Run(\"replSetGetStatus\") failed with\n%s", err.Error()),
		}, replSetUnknown
	}

	members := make([]msp.HostPort, len(status["members"].([]interface{})))
	for k, member := range status["members"].([]interface{}) {
		pair := strings.Split(member.(bson.M)["name"].(string), ":")
		remotePort, _ := strconv.Atoi(pair[1])
		members[k] = msp.HostPort{pair[0], msp.PortNumber(remotePort) }
	}

	var state msp.MongodState
	if replSetState(status["myState"].(int)) == replSetRecovering {
		state = msp.MongodStateRecovering
	} else {
		state = msp.MongodStateRunning
	}

	return msp.Mongod{
		Port: port,
		ReplicaSetName: status["set"].(string),
		ReplicaSetMembers: members,
		ShardingConfigServer: false,
		StatusError: nil,
		LastEstablishStateError: nil,
		State: state,
	}, nil, replSetState(status["myState"].(int))
}

func (c *ConcreteMongodConfigurator) MongodConfiguration(port msp.PortNumber) (msp.Mongod, *msp.Error) {
	sess, err := c.connect(port)
	defer sess.Close()
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
	defer sess.Close()
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
		sess.Run(bson.D{{"shutdown", 1}, {"timeoutSecs", MongodSoftShutdownTimeout }}, result)
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

		members := make([]bson.M, len(m.ReplicaSetMembers))
		for k, member := range m.ReplicaSetMembers {
			members[k] = bson.M{ "_id": k, "host": fmt.Sprintf("%s:%d", member.Hostname, member.Port) }
		}

		var result interface{}
		cmd := bson.D{{"replSetReconfig", bson.M{"_id": m.ReplicaSetName, "version": 1, "members": members}}, {"force", true}}
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
