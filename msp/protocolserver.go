package msp

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
)

type Consumer interface {
	RequestStatus() ([]Mongod, *SlaveError)
	EstablishMongodState(m Mongod) *SlaveError
}

type Listener struct {
	listener Consumer
	router   *mux.Router
}

func NewMSPServer(listener Consumer) *Listener {
	s := new(Listener)
	s.listener = listener

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

func (s Listener) Run() {
	http.ListenAndServe(":8081", s.router)
}
