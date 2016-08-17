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
)

var log = logrus.WithField("module", "slave")

const MongodExecutableDefaultName = "mongod"
const DefaultMongodSoftShutdownTimeout = "3s" // seconds
const DefaultMongodHardShutdownTimeout = "5s" // seconds

func main() {

	log.SetLevel(logrus.DebugLevel)

	var (
		mongodExecutable, dataDir                                  string
		mongodSoftShutdownTimeoutStr, mongodHardShutdownTimeoutStr string
	)

	flag.StringVar(&dataDir, "data", "", "Persistent data and slave configuration directory")
	mongodExecutableLookupPath, _ := exec.LookPath(MongodExecutableDefaultName)
	flag.StringVar(&mongodExecutable, "mongodExecutable", mongodExecutableLookupPath, "Path to or name of Mongod binary")

	flag.StringVar(&mongodSoftShutdownSeconds, "mongod.shutdownTimeout.soft", DefaultMongodSoftShutdownTimeout,
		"Duration to wait for regular Mongod shutdown call to return. Specify with suffix [ms,s,min,...]")
	flag.StringVar(&mongodHardShutdownSeconds, "mongod.shutdownTimeout.hard", DefaultMongodHardShutdownTimeout,
		"Duration to wait after issuing a shutdown call before the Mongod is killed (SIGKILL). Specify with suffix [ms,s,min,...]")

	flag.Parse()

	// Assert dataDir is valid. TODO should we do this lazyly?

	if dataDir == "" {
		log.Fatal("No root data directory passed; specify with -data=/path/to/root/dir")
	}

	if err := unix.Access(dataDir, unix.W_OK); err != nil {
		log.Fatal(fmt.Sprintf("Root data directory %s does not exist or is not writable", dataDir))
	}

	dbDir := fmt.Sprintf("%s/%s", dataDir, DataDBDir) // TODO directory creation should happen in the component that uses the path
	if err := unix.Access(dbDir, unix.R_OK|unix.W_OK|unix.X_OK); err != nil {
		if err := unix.Mkdir(dbDir, 0700); err != nil {
			log.Printf("Could not create a readable and writable directory at %s: %s", dbDir, err)
			return
		}
	}

	// Convert timeouts to internal representation

	mongodSoftShutdownTimeout, err := time.ParseDuration(mongodSoftShutdownTimeoutStr)
	if !err {
		log.Fatal("could not convert soft shutdown timeout to time.Duration: %s", err)
	}

	mongodHardShutdownTimeout, err := time.Duration(mongodHardShutdownTimeoutStr)
	if !err {
		log.Fatal("could not convert hard shutdown timeout to time.Duration: %s", err)
	}

	processManager := NewProcessManager(mongodExecutable, dataDir)
	configurator := &ConcreteMongodConfigurator{
		dial: mgo.Dial,
		MongodSoftShutdownTimeout: mongodSoftShutdownTimeout,
	}

	controller := NewController(processManager, configurator, mongodHardShutdownTimeout)
	server := msp.NewServer(controller)
	server.Run()
}
