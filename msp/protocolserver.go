package msp

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
)

type Consumer interface {
	RequestStatus() ([]Mongod, *Error)
	EstablishMongodState(m Mongod) *Error
}

type Listener struct {
	listenString string
	listener     Consumer
	router       *mux.Router
}

func NewServer(listener Consumer, listenString string) *Listener {
	s := new(Listener)
	s.listener = listener
	s.listenString = listenString

	s.router = mux.NewRouter().StrictSlash(true)
	s.router.Methods("GET").Path("/msp/status").Name("RequestStatus").HandlerFunc(s.handleRequestStatus)
	s.router.Methods("POST").Path("/msp/establishMongodState").Name("EstablishMongodState").HandlerFunc(s.handleMspEstablishMongodState)

	return s
}

func (s Listener) handleRequestStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.listener.RequestStatus()
	if status != nil {
		json.NewEncoder(w).Encode(status)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
	}
}

func (s Listener) handleMspEstablishMongodState(w http.ResponseWriter, r *http.Request) {
	var mongodState Mongod
	json.NewDecoder(r.Body).Decode(&mongodState) //TODO Check decode error
	err := s.listener.EstablishMongodState(mongodState)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
	}
}

func (s Listener) Run() error {
	return http.ListenAndServe(s.listenString, s.router)
}
