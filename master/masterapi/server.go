package masterapi

import (
	"net/http"
	"github.com/gorilla/mux"
	"log"
)

type MasterAPIServer struct {
}

func (s MasterAPIServer) Run() {
	router := mux.NewRouter().StrictSlash(true)

	staticServer := http.FileServer(http.Dir("./gui/"))
	router.Handle("/", staticServer)
	router.PathPrefix("/static/").Handler(staticServer)
	router.PathPrefix("/pages/").Handler(staticServer)
	router.Methods("GET").Path("/api/slaves").Name("SlaveIndex").HandlerFunc(SlaveIndex)
	router.Methods("GET").Path("/api/slaves/{slaveId}").Name("SlaveById").HandlerFunc(SlaveById)
	router.Methods("PUT").Path("/api/slaves").Name("SlavePut").HandlerFunc(SlavePut)
	router.Methods("POST").Path("/api/slaves/{slaveId}").Name("SlaveUpdate").HandlerFunc(SlaveUpdate)
	router.Methods("DELETE").Path("/api/slaves/{slaveId}").Name("SlaveDelete").HandlerFunc(SlaveDelete)
	router.Methods("GET").Path("/api/replicasets").Name("ReplicaSetIndex").HandlerFunc(ReplicaSetIndex)
	router.Methods("GET").Path("/api/riskgroups").Name("RiskGroupIndex").HandlerFunc(RiskGroupIndex)

	log.Fatal(http.ListenAndServe(":8080", router))
}