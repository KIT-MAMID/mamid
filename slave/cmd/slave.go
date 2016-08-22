package main

import (
	"flag"
	"fmt"
	"github.com/KIT-MAMID/mamid/msp"
	. "github.com/KIT-MAMID/mamid/slave"
	"github.com/Sirupsen/logrus"
	"golang.org/x/sys/unix"
	"gopkg.in/mgo.v2"
	"os/exec"
	"time"
)

var log = logrus.WithField("module", "slave")

const MongodExecutableDefaultName = "mongod"

var DefaultMongodSoftShutdownTimeout, _ = time.ParseDuration("3s") // seconds
var DefaultMongodHardShutdownTimeout, _ = time.ParseDuration("5s") // seconds

func main() {

	logrus.SetLevel(logrus.DebugLevel)

	var (
		mongodExecutable, dataDir                            string
		mongodSoftShutdownTimeout, mongodHardShutdownTimeout time.Duration
	)

	flag.StringVar(&dataDir, "data", "", "Persistent data and slave configuration directory")
	mongodExecutableLookupPath, _ := exec.LookPath(MongodExecutableDefaultName)
	flag.StringVar(&mongodExecutable, "mongodExecutable", mongodExecutableLookupPath, "Path to or name of Mongod binary")

	flag.DurationVar(&mongodSoftShutdownTimeout, "mongod.shutdownTimeout.soft", DefaultMongodSoftShutdownTimeout,
		"Duration to wait for regular Mongod shutdown call to return. Specify with suffix [ms,s,min,...]")
	flag.DurationVar(&mongodHardShutdownTimeout, "mongod.shutdownTimeout.hard", DefaultMongodHardShutdownTimeout,
		"Duration to wait after issuing a shutdown call before the Mongod is killed (SIGKILL). Specify with suffix [ms,s,min,...]")

	port := flag.Uint("port", 8081, "Listening port number of slave server")

	flag.Parse()

	// Assert dataDir is valid. TODO should we do this lazyly?

	if dataDir == "" {
		log.Fatal("No root data directory passed; specify with -data=/path/to/root/dir")
	}

	if err := unix.Access(dataDir, unix.W_OK); err != nil {
		log.Fatal(fmt.Sprintf("Root data directory %s does not exist or is not writable", dataDir))
	}

	// ensure that main database directory exists
	dbDir := fmt.Sprintf("%s/%s", dataDir, DataDBDir)
	if err := unix.Access(dbDir, unix.R_OK|unix.W_OK|unix.X_OK); err != nil {
		if err := unix.Mkdir(dbDir, 0700); err != nil {
			log.Printf("Could not create a readable and writable directory at %s: %s", dbDir, err)
			return
		}
	}

	processManager := NewProcessManager(mongodExecutable, dataDir)
	configurator := &ConcreteMongodConfigurator{
		Dial: mgo.Dial,
		MongodSoftShutdownTimeout: mongodSoftShutdownTimeout,
	}

	controller := NewController(processManager, configurator, mongodHardShutdownTimeout)
	server := msp.NewServer(controller, msp.PortNumber(*port))
	server.Run()
}
