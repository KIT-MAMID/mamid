package slave

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/msp"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type mgoContext struct {
	Session         *mgo.Session
	Port            msp.PortNumber
	LoginSuccessful bool
}

func (ctx *mgoContext) Close() {
	ctx.Session.Close()
}

func (ctx *mgoContext) IsMaster(isMasterRes *bson.M) *msp.Error {
	if err := ctx.Session.Run("isMaster", isMasterRes); err != nil {
		return &msp.Error{
			Identifier:      msp.SlaveGetMongodStatusError,
			Description:     fmt.Sprintf("Getting master information from mongod instance on port %d failed", ctx.Port),
			LongDescription: fmt.Sprintf("mgo/Session.Run(\"isMaster\") failed with\n%s", err.Error()),
		}
	}
	return nil
}

// Run replSetGetStatus
// returns
// 	mongodState the state of the Mongod. Valid even if err != nil
// 	err an MSPError indicating replSetGetStatus failed
func (ctx *mgoContext) ReplSetGetStatus(status *bson.M) (replSetMemberState replSetState, err *msp.Error) {

	replSetMemberState = replSetUnknown

	cmdRunErr := ctx.Session.Run("replSetGetStatus", status)

	//If the replica set is in removed state the replSetGetStatus command returns an error
	//This error still contains the replica set state but in the "state" field instead of the "myState" field
	var stateFieldName string
	if cmdRunErr != nil {
		stateFieldName = "state"
	} else {
		stateFieldName = "myState"
	}

	if errorDocState, valid := (*status)[stateFieldName]; valid {
		replSetMemberState = replSetState(errorDocState.(int))
	} else {
		if cmdRunErr != nil {
			// an real error must have occurred
			err = &msp.Error{
				Identifier:      msp.SlaveGetMongodStatusError,
				Description:     fmt.Sprintf("Getting Replica Set status information from Mongod instance on port `%d` failed", ctx.Port),
				LongDescription: fmt.Sprintf("mgo/Session.Run(\"replSetGetStatus\") result was %#v", status),
			}
		} else {
			//myState field does not exist
			err = &msp.Error{
				Identifier:      msp.SlaveGetMongodStatusError,
				Description:     fmt.Sprintf("Getting Replica Set status information from Mongod instance on port `%d` failed", ctx.Port),
				LongDescription: fmt.Sprintf("`myState` field does not exist in mgo/Session.Run(\"replSetGetStatus\") result: %#v", status),
			}
		}
	}

	return
}

func (ctx *mgoContext) ReplSetGetConfig() (bson.M, *msp.Error) {

	configResult := bson.M{}
	if err := ctx.Session.Run("replSetGetConfig", &configResult); err != nil {
		return bson.M{}, &msp.Error{
			Identifier:      msp.SlaveGetMongodStatusError,
			Description:     fmt.Sprintf("Getting Replica Set config information from Mongod instance on port `%d` failed", ctx.Port),
			LongDescription: fmt.Sprintf("mgo/Session.Run(\"replSetGetConfig\") result was %#v", configResult),
		}
	}
	return configResult["config"].(bson.M), nil

}

func (ctx *mgoContext) ReplSetReconfig(config bson.M) *msp.Error {
	cmd := bson.D{{"replSetReconfig", config}}
	var result bson.M
	reconfigErr := ctx.Session.Run(cmd, &result)
	if reconfigErr != nil {
		return &msp.Error{
			Identifier:      msp.SlaveReplicaSetConfigError,
			Description:     fmt.Sprintf("Could not reconfigure Replica Set"),
			LongDescription: fmt.Sprintf("replSetReconfig on Mongod instance on port `%d` failed. Config: %#v Error: %#v", cmd, reconfigErr),
		}
	}
	return nil
}

// Parse sharding command line options
// return an empty string and err = nil if option not specified but no other error occurred
func (ctx *mgoContext) ParseCmdLineShardingRole() (role string, err *msp.Error) {

	cmdLineOptsRes := bson.M{}
	if err := ctx.Session.Run("getCmdLineOpts", &cmdLineOptsRes); err != nil {
		return "", &msp.Error{
			Identifier:      msp.SlaveGetMongodStatusError,
			Description:     fmt.Sprintf("Getting command line options from Mongod instance on port `%d` failed", ctx.Port),
			LongDescription: fmt.Sprintf("getCmdLineOpts failed with error: %#v", err),
		}
	}

	parsed := cmdLineOptsRes["parsed"].(bson.M)
	sharding, ok := parsed["sharding"]
	if ok {
		clusterRole, ok := sharding.(bson.M)["clusterRole"]
		if ok {
			role = clusterRole.(string)
		}
	}
	return
}

func (ctx *mgoContext) ShutdownWithTimeout(seconds int64) *msp.Error {
	var result interface{}
	err := ctx.Session.Run(bson.D{{"shutdown", 1}, {"timeoutSecs", seconds}}, result)
	if err != nil {
		return &msp.Error{
			Identifier:      msp.SlaveShutdownError,
			Description:     fmt.Sprintf("Could not shutdown Mongod with timeout `%d` on port `%d` (mongodb returned error)", seconds, ctx.Port),
			LongDescription: fmt.Sprintf("mgo/Session.Run(\"shutdown\") failed with: %s", err),
		}
	}
	return nil
}

func (ctx *mgoContext) ReplSetStepDown(stepDownSec int64) *msp.Error {
	var stepDownRes interface{}
	stepDownErr := ctx.Session.Run(bson.D{{"replSetStepDown", stepDownSec}}, stepDownRes)
	if stepDownErr != nil {
		return &msp.Error{
			Identifier:      msp.SlaveShutdownError,
			Description:     fmt.Sprintf("could not step down Mongod on port `%d` (mongodb returned error)", ctx.Port),
			LongDescription: fmt.Sprintf("mgo/Session.Run(\"replSetStepDown\") failed: %s", stepDownErr),
		}
	}
	return nil
}

// Create a user on the ctx's database with predefined roles
// See https://docs.mongodb.com/manual/core/security-built-in-roles/
func (ctx *mgoContext) CreateUser(user, password, purpose string, roles []string) *msp.Error {
	var result interface{}
	cmd := bson.D{
		{"createUser", user},
		{"pwd", password},
		{"roles", roles},
		{"writeConcern", bson.M{"w": "majority"}}, // Config servers need the majority write concern. And it doesn't hurt for non-config servers, too
	}
	err := ctx.Session.Run(cmd, &result)
	log.Debugf("error creating user: %#v", err)
	if err != nil {
		return &msp.Error{
			Identifier:      msp.SlaveReplicaSetCreateRootUserError,
			Description:     fmt.Sprintf("Could not create %s `%s`", purpose, user),
			LongDescription: fmt.Sprintf("Command `createUser` on port `%d` failed: %#v", ctx.Port, err),
		}
	}
	return nil
}

func (ctx *mgoContext) ReplSetInitiate(config bson.M, force bool) (alreadyInitialized bool, mspErr *msp.Error) {

	var result interface{}
	cmd := bson.D{{"replSetInitiate", config}, {"force", force}}

	err := ctx.Session.Run(cmd, &result)

	if err != nil {
		queryErr, valid := err.(*mgo.QueryError)
		switch {
		case valid && queryErr.Code == 23: // Replica Set is already initalized
			return true, nil
		default:
			return false, &msp.Error{
				Identifier:      msp.SlaveReplicaSetInitError,
				Description:     fmt.Sprintf("Replica Set could not be initiated on Mongod on port `%d`", ctx.Port),
				LongDescription: fmt.Sprintf("Command `replSetInitiate` failed:\nConfig: %#v\nError: %#v", config, err),
			}
		}
	}
	return false, nil

}

func (c *ConcreteMongodConfigurator) connect(port msp.PortNumber, replicaSetName string, credential msp.MongodCredential) (ctx *mgoContext, err *msp.Error) {

	mgo.SetDebug(false)

	sess, dialErr := mgo.DialWithInfo(&mgo.DialInfo{
		Addrs:    []string{fmt.Sprintf("127.0.0.1:%d", port)}, // TODO shouldn't we use localhost instead? otherwise, this will break the day IPv4 is dropped
		Direct:   true,
		Timeout:  c.MongodResponseTimeout,
		Database: mongodbAdminDatabase,
	})

	if dialErr != nil {
		return nil, &msp.Error{
			Identifier:      msp.SlaveConnectMongodError,
			Description:     fmt.Sprintf("Establishing a connection to mongod instance on port %d failed", port),
			LongDescription: fmt.Sprintf("ConcreteMongodConfigurator.connect() failed with: %s", dialErr),
		}
	}

	// Decrease the level of consistency, allowing reads from other members than PRIMARY
	// sess.Login() requires read access
	sess.SetMode(mgo.Monotonic, true)

	// attempt login because replica set management commands don't work if unauthenticated & RS already initialized
	loginError := sess.Login(&mgo.Credential{Username: credential.Username, Password: credential.Password})
	if loginError != nil {
		log.Infof("ignoring login error, assuming Replica Set is uninitialized: %s", loginError)
	}

	ctx = &mgoContext{
		Session:         sess,
		Port:            port,
		LoginSuccessful: loginError == nil,
	}

	return ctx, nil
}
