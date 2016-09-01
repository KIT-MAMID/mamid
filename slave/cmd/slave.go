package main

import (
	"flag"
	"fmt"
	"github.com/KIT-MAMID/mamid/msp"
	. "github.com/KIT-MAMID/mamid/slave"
	"github.com/Sirupsen/logrus"
	"golang.org/x/sys/unix"
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

	// Assert dataDir is valid. TODO should we do this lazyly?

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
		log.Fatal("No master verification ca passen; specify with -master.verifyCA=/path/to/cert")
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
