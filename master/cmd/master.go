package main

import (
	"github.com/KIT-MAMID/mamid/master"
	"github.com/KIT-MAMID/mamid/master/masterapi"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/KIT-MAMID/mamid/msp"
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net/http"
)

var masterLog = logrus.WithField("module", "master")

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	masterLog.Info("Startup")

	// Setup controllers

	bus := master.NewBus()
	go bus.Run()

	db, err := model.InitializeFileFromFile("mamid.sqlite3")
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

	err = http.ListenAndServe(":8080", mainRouter)
	dieOnError(err)
}

func dieOnError(err error) {
	if err != nil {
		panic(err)
	}
}
