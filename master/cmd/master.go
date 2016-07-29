package main

import (
	"github.com/KIT-MAMID/mamid/master"
	"github.com/KIT-MAMID/mamid/master/masterapi"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/KIT-MAMID/mamid/msp"
	"github.com/gorilla/mux"
	"net/http"
)

func main() {

	// Setup controllers

	bus := master.NewBus()
	go bus.Run()

	db, err := model.InitializeInMemoryDB("")
	dieOnError(err)

	clusterAllocator := &master.ClusterAllocator{}

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

	monitor := master.Monitor{
		DB:              db,
		BusWriteChannel: bus.GetNewWriteChannel(),
		MSPClient:       msp.MSPClientImpl{},
	}
	go monitor.Run()

	// Listen

	err = http.ListenAndServe(":8080", mainRouter)
	dieOnError(err)
}

func dieOnError(err error) {
	if err != nil {
		panic(err)
	}
}
