package slave

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/msp"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"time"
)

type mgoContext struct {
	Session *mgo.Session
	Port    msp.PortNumber
}

func (ctx *mgoContext) Close() {
	ctx.Session.Close()
}

func (ctx *mgoContext) IsMaster(isMasterRes interface{}) *msp.Error {
	if err := ctx.Session.Run("isMaster", &isMasterRes); err != nil {
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
	queryErr, isQueryErr := cmdRunErr.(*mgo.QueryError)

	if cmdRunErr == nil || isQueryErr {
		// in case of QueryError, mgo marshals the resulting error-document into &status
		// => the "state" field is set in QueryError and no-error cases
		if errorDocState, valid := (*status)["state"]; valid {
			replSetMemberState = errorDocState.(replSetState)
		} else if cmdRunErr == nil {
			// Don't know what to do if expected field state is not found
			err = &msp.Error{
				Identifier:      msp.SlaveGetMongodStatusError,
				Description:     fmt.Sprintf("Getting Replica Set status information from Mongod instance on port `%d` failed", ctx.Port),
				LongDescription: fmt.Sprintf("field `state` does not exist in non-error response: %#v", status),
			}
		}
	} else {
		// an error must have occurred
		err = &msp.Error{
			Identifier:      msp.SlaveGetMongodStatusError,
			Description:     fmt.Sprintf("Getting Replica Set status information from Mongod instance on port `%d` failed", ctx.Port),
			LongDescription: fmt.Sprintf("mgo/Session.Run(\"replSetGetStatus\") result was %#v", status),
		}
	}

	return
}

func (c *ConcreteMongodConfigurator) connect(port msp.PortNumber, replicaSetName string, credential msp.MongodCredential) (ctx *mgoContext, err *msp.Error) {

	mgo.SetDebug(false)

	sess, dialErr := mgo.DialWithInfo(&mgo.DialInfo{
		Addrs:    []string{fmt.Sprintf("127.0.0.1:%d", port)}, // TODO shouldn't we use localhost instead? otherwise, this will break the day IPv4 is dropped
		Direct:   true,
		Timeout:  4 * time.Second,
		Database: mongodbAdminDatabase,
	})

	if dialErr != nil {
		return nil, &msp.Error{
			Identifier:      msp.SlaveConnectMongodError,
			Description:     fmt.Sprintf("Establishing a connection to mongod instance on port %d failed", port),
			LongDescription: fmt.Sprintf("ConcreteMongodConfigurator.connect() failed with: %s", err),
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
		Session: sess,
		Port:    port,
	}

	return ctx, nil
}
