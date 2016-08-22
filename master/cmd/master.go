package main

import (
	"flag"
	"github.com/KIT-MAMID/mamid/master"
	"github.com/KIT-MAMID/mamid/master/masterapi"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/KIT-MAMID/mamid/msp"
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net/http"
)

var masterLog = logrus.WithField("module", "master")

type LogLevelFlag struct { // flag.Value
	lvl logrus.Level
}

func (f LogLevelFlag) String() string { return f.lvl.String() }
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
		logLevel             LogLevelFlag = LogLevelFlag{logrus.DebugLevel}
		dbPath, listenString string
	)

	flag.Var(&logLevel, "log.level", "possible values: debug, info, warning, error, fatal, panic")
	flag.StringVar(&dbPath, "db.path", "", "path to the SQLite file where MAMID data is stored")
	flag.StringVar(&listenString, "listen", ":8080", "net.Listen() string, e.g. addr:port")
	flag.Parse()

	if dbPath == "" {
		masterLog.Fatal("-db.path cannot be empty")
	}

	// Start application
	logrus.SetLevel(logLevel.lvl)
	masterLog.Info("Startup")

	// Setup controllers

	bus := master.NewBus()
	go bus.Run()

	db, err := model.InitializeFileFromFile(dbPath)
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

	mspClient := msp.MSPClientImpl{}

	monitor := master.Monitor{
		DB:              db,
		BusWriteChannel: bus.GetNewWriteChannel(),
		MSPClient:       mspClient,
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
