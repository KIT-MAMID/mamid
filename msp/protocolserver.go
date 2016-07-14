package msp

import (
	"net/http"
	"encoding/json"
	"github.com/gorilla/mux"
)

type MSPListenerDelegate interface {
	MspSetDataPath(path string) MSPError
	MspStatusRequest() ([]Mongod, MSPError)
	MspEstablishMongodState(m Mongod) MSPError
}

type MSPServer struct {
	listener MSPListenerDelegate
	router *mux.Router
}

func NewMSPServer(listener MSPListenerDelegate) *MSPServer {
	s := new(MSPServer)
	s.listener = listener

	s.router = mux.NewRouter().StrictSlash(true)
	s.router.Methods("GET").Path("/msp/status").Name("MspStatusRequest").HandlerFunc(s.handleMspStatusRequest)
	s.router.Methods("POST").Path("/msp/setDataPath").Name("MspStatusRequest").HandlerFunc(s.handleMspSetDataPath)
	s.router.Methods("POST").Path("/msp/establishMongodState").Name("MspStatusRequest").HandlerFunc(s.handleMspEstablishMongodState)

	return s
}

func (s MSPServer) handleMspStatusRequest(w http.ResponseWriter, r *http.Request) {
	status, err := s.listener.MspStatusRequest()
	if status != nil {
		json.NewEncoder(w).Encode(status)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		err.encodeJson(w)
	}
}

func (s MSPServer) handleMspSetDataPath(w http.ResponseWriter, r *http.Request) {
	var path string
	json.NewDecoder(r.Body).Decode(&path) //TODO Check decode error
	err := s.listener.MspSetDataPath(path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		err.encodeJson(w)
	}
}

func (s MSPServer) handleMspEstablishMongodState(w http.ResponseWriter, r *http.Request) {
	var mongodState Mongod
	json.NewDecoder(r.Body).Decode(&mongodState) //TODO Check decode error
	err := s.listener.MspEstablishMongodState(mongodState)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		err.encodeJson(w)
	}
}

func (s MSPServer) Listen() {
	http.ListenAndServe(":8081", s.router)
}