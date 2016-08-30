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
	certFile     string
	keyFile      string
}

func NewServer(listener Consumer, listenString string, certFile string, keyFile string) *Listener {
	s := new(Listener)
	s.listener = listener
	s.listenString = listenString
	s.certFile = certFile
	s.keyFile = keyFile

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
	return http.ListenAndServeTLS(s.listenString, s.certFile, s.keyFile,
		s.router)
}
