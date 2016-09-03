package main

import (
	"flag"
	"fmt"
	"github.com/KIT-MAMID/mamid/msp"
	. "github.com/KIT-MAMID/mamid/slave"
	"github.com/Sirupsen/logrus"
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
		mongodExecutable, dataDir, listenString, x509CertFile, x509KeyFile, caCert string
		mongodSoftShutdownTimeout, mongodHardShutdownTimeout                       time.Duration
	)

	flag.StringVar(&dataDir, "data", "", "Persistent data and slave configuration directory")
	mongodExecutableLookupPath, _ := exec.LookPath(MongodExecutableDefaultName)
	flag.StringVar(&mongodExecutable, "mongodExecutable", mongodExecutableLookupPath, "Path to or name of Mongod binary")

	flag.DurationVar(&mongodSoftShutdownTimeout, "mongod.shutdownTimeout.soft", DefaultMongodSoftShutdownTimeout,
		"Duration to wait for regular Mongod shutdown call to return. Specify with suffix [ms,s,min,...]")
	flag.DurationVar(&mongodHardShutdownTimeout, "mongod.shutdownTimeout.hard", DefaultMongodHardShutdownTimeout,
		"Duration to wait after issuing a shutdown call before the Mongod is killed (SIGKILL). Specify with suffix [ms,s,min,...]")

	flag.StringVar(&listenString, "listen", ":8081", "net.Listen() string, e.g. addr:port")
	flag.StringVar(&x509CertFile, "slave.auth.cert", "", "The x509 cert file for the slave server")
	flag.StringVar(&x509KeyFile, "slave.auth.key", "", "The x509 key file for x509 cert the slave server")
	flag.StringVar(&caCert, "master.verifyCA", "", "The x509 ca that signed the certificates and to authenticate the master against")
	flag.Parse()

	// Validate non-optional fields

	if dataDir == "" {
		log.Fatal("No root data directory passed; specify with -data=/path/to/root/dir")
	}
	if x509CertFile == "" {
		log.Fatal("No server cert file passed; specify with -slave.auth.cert=/path/to/cert")
	}
	if x509KeyFile == "" {
		log.Fatal("No server cert key file passed; specify with -slave.auth.key=/path/to/cert")
	}
	if caCert == "" {
		log.Fatal("No master verification ca passed; specify with -master.verifyCA=/path/to/cert")
	}

	// Application setup

	processManager := NewProcessManager(mongodExecutable, dataDir)
	if err := processManager.CreateManagedDirs(); err != nil {
		log.Fatal(fmt.Sprintf("cannot not create or access slave data directory `%s` (-data): %s", dataDir, err))
	}
	processManager.Run()

	configurator := &ConcreteMongodConfigurator{
		MongodSoftShutdownTimeout: mongodSoftShutdownTimeout,
	}

	controller := NewController(processManager, configurator, mongodHardShutdownTimeout)

	server := msp.NewServer(controller, listenString, caCert, x509CertFile, x509KeyFile)
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
