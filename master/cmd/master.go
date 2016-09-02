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
		logLevel                                                                 LogLevelFlag = LogLevelFlag{logrus.DebugLevel}
		listenString                                                             string
		slaveVerifyCA, slaveAuthCert, slaveAuthKey, apiCert, apiKey, apiVerifyCA string
		dbDriver, dbDSN                                                          string
		monitorInterval                                                          time.Duration
	)

	flag.Var(&logLevel, "log.level", "possible values: debug, info, warning, error, fatal, panic")
	flag.StringVar(&dbDriver, "db.driver", "postgres", "the database driver to use. See https://golang.org/pkg/database/sql/#Open")
	flag.StringVar(&dbDSN, "db.dsn", "", "the data source name to use. for PostgreSQL, checkout https://godoc.org/github.com/lib/pq")
	flag.StringVar(&listenString, "listen", ":8080", "net.Listen() string, e.g. addr:port")
	flag.StringVar(&slaveVerifyCA, "slave.verifyCA", "", "The CA certificate to verify slaves against")
	flag.StringVar(&slaveAuthCert, "slave.auth.cert", "", "The client certificate for authentication against the slave")
	flag.StringVar(&slaveAuthKey, "slave.auth.key", "", "The key for the client certificate for authentication against the slave")
	flag.StringVar(&apiCert, "api.cert", "", "Optional: a certificate for the api/webinterface")
	flag.StringVar(&apiKey, "api.key", "", "Optional: the key for the certificate for the api/webinterface")
	flag.StringVar(&apiVerifyCA, "api.verifyCA", "", "Optional: a ca to check client certs of webinterface/api users. Implies authentication")

	flag.DurationVar(&monitorInterval, "monitor.interval", time.Duration(10*time.Second),
		"Interval in which the monitoring component should poll slaves for status updates. Specify with suffix [ms,s,min,...]")
	flag.Parse()

	if dbDriver != "postgres" {
		masterLog.Fatal("-db.driver: only 'postgres' is supported")
	}
	if dbDSN == "" {
		masterLog.Fatal("-db.dsn cannot be empty")
	}
	if slaveVerifyCA == "" {
		masterLog.Fatal("No root certificate for the slave server communication passed. Specify with -slave.verifyCA")
	}
	if slaveAuthCert == "" {
		masterLog.Fatal("No client certificate for the slave server communication passed. Specify with -slave.auth.cert")
	}
	if slaveAuthKey == "" {
		masterLog.Fatal("No key for the client certificate for the slave server communication passed. Specify with -slave.auth.key")
	}
	if check := apiKey + apiCert; check != "" && (check == apiKey || check == apiCert) {
		masterLog.Fatal("Either -apiCert specified without -apiKey or vice versa.")
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

	tx := db.Begin()
	if err := clusterAllocator.InitializeGlobalSecrets(tx); err != nil {
		tx.Rollback()
		masterLog.Fatalf("Error initializing global secrets: %s", err)
	}
	tx.Commit()

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
	cert, err := loadCertificateFromFile(slaveVerifyCA)
	dieOnError(err)
	certPool.AddCert(cert)
	clientAuthCert, err := tls.LoadX509KeyPair(slaveAuthCert, slaveAuthKey)
	dieOnError(err)
	httpTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:      certPool,
			Certificates: []tls.Certificate{clientAuthCert},
		},
	}
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

	listenAndServe(listenString, mainRouter, apiCert, apiKey, apiVerifyCA)
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

func listenAndServe(listenString string, mainRouter *mux.Router, apiCert string, apiKey string, apiVerifyCA string) {
	// Listen...
	if apiCert != "" {
		// ...with TLS but WITHOUT client cert auth
		if apiVerifyCA == "" {
			err := http.ListenAndServeTLS(listenString, apiCert, apiKey, mainRouter)
			dieOnError(err)
		} else { // ...with TLS AND client cert auth
			certPool := x509.NewCertPool()
			caCertContent, err := ioutil.ReadFile(apiVerifyCA)
			dieOnError(err)
			certPool.AppendCertsFromPEM(caCertContent)
			tlsConfig := &tls.Config{
				ClientCAs:  certPool,
				ClientAuth: tls.RequireAndVerifyClientCert,
			}
			server := &http.Server{
				TLSConfig: tlsConfig,
				Addr:      listenString,
				Handler:   mainRouter,
			}
			err = server.ListenAndServeTLS(apiCert, apiKey)
			dieOnError(err)
		}
	} else {
		// ...insecure and unauthenticated
		err := http.ListenAndServe(listenString, mainRouter)
		dieOnError(err)
	}
}
