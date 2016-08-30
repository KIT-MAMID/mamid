package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"github.com/KIT-MAMID/mamid/master"
	"github.com/KIT-MAMID/mamid/master/masterapi"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/KIT-MAMID/mamid/msp"
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"time"
)

var masterLog = logrus.WithField("module", "master")

type LogLevelFlag struct {
	// flag.Value
	lvl logrus.Level
}

func (f LogLevelFlag) String() string {
	return f.lvl.String()
}
func (f LogLevelFlag) Set(val string) error {
	l, err := logrus.ParseLevel(val)
	if err != nil {
		f.lvl = l
	}
	return err
}

func main() {

	// Command Line Flags
	var (
		logLevel                     LogLevelFlag = LogLevelFlag{logrus.DebugLevel}
		dbPath, listenString, rootCA string
		dbDriver, dbDSN              string
		monitorInterval              time.Duration
	)

	flag.Var(&logLevel, "log.level", "possible values: debug, info, warning, error, fatal, panic")
	flag.StringVar(&dbPath, "db.path", "", "path to the SQLite file where MAMID data is stored")
	flag.StringVar(&dbDriver, "db.driver", "postgres", "the database driver to use. See https://golang.org/pkg/database/sql/#Open")
	flag.StringVar(&dbDSN, "db.dsn", "", "the data source name to use. for PostgreSQL, checkout https://godoc.org/github.com/lib/pq")
	flag.StringVar(&listenString, "listen", ":8080", "net.Listen() string, e.g. addr:port")
	flag.StringVar(&rootCA, "cacert", "", "The CA certificate to verify slaves against it")
	flag.DurationVar(&monitorInterval, "monitor.interval", time.Duration(10*time.Second),
		"Interval in which the monitoring component should poll slaves for status updates. Specify with suffix [ms,s,min,...]")
	flag.Parse()

	if dbDriver != "postgres" {
		masterLog.Fatal("-db.driver: only 'postgres' is supported")
	}
	if dbDSN == "" {
		masterLog.Fatal("-db.dsn cannot be empty")
	}
	if rootCA == "" {
		masterLog.Fatal("No root certificate for the slave server communication passed. Specify with -cacert")
	}
	// Start application
	logrus.SetLevel(logLevel.lvl)
	masterLog.Info("Startup")

	// Setup controllers

	bus := master.NewBus()
	go bus.Run()

	db, err := model.InitializeDB(dbDriver, dbDSN)
	dieOnError(err)

	clusterAllocatorBusWriteChannel := bus.GetNewWriteChannel()
	clusterAllocator := &master.ClusterAllocator{
		BusWriteChannel: &clusterAllocatorBusWriteChannel,
	}
	go clusterAllocator.Run(db)

	mainRouter := mux.NewRouter().StrictSlash(true)

	httpStatic := http.FileServer(http.Dir("./gui/"))
	mainRouter.Handle("/", httpStatic)
	mainRouter.PathPrefix("/static/").Handler(httpStatic)
	mainRouter.PathPrefix("/pages/").Handler(httpStatic)

	masterAPI := &masterapi.MasterAPI{
		DB:               db,
		ClusterAllocator: clusterAllocator,
		Router:           mainRouter.PathPrefix("/api/").Subrouter(),
	}
	masterAPI.Setup()

	certPool := x509.NewCertPool()
	cert, err := loadCertificateFromFile(rootCA)
	dieOnError(err)
	certPool.AddCert(cert)
	httpTransport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: certPool}}
	mspClient := msp.MSPClientImpl{HttpClient: http.Client{Transport: httpTransport}}

	monitor := master.Monitor{
		DB:              db,
		BusWriteChannel: bus.GetNewWriteChannel(),
		MSPClient:       mspClient,
		Interval:        monitorInterval,
	}
	go monitor.Run()

	deployer := master.Deployer{
		DB:             db,
		BusReadChannel: bus.GetNewReadChannel(),
		MSPClient:      mspClient,
	}
	go deployer.Run()

	problemManager := master.ProblemManager{
		DB:             db,
		BusReadChannel: bus.GetNewReadChannel(),
	}
	go problemManager.Run()

	// Listen

	err = http.ListenAndServe(listenString, mainRouter)
	dieOnError(err)
}

func dieOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func loadCertificateFromFile(file string) (cert *x509.Certificate, err error) {
	certFile, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	block, _ := pem.Decode(certFile)
	cert, err = x509.ParseCertificate(block.Bytes)
	return
}
