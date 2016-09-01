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
