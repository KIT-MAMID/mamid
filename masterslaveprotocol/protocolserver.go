package masterslaveprotocol

import (
	"net/http"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
)

type MSPListenerDelegate interface {
	MspSetDataPath(path string) error
	MspStatusRequest() []Mongod
	MspEstablishMongodState(m Mongod) error
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
	status := s.listener.MspStatusRequest()
	if status != nil {
		json.NewEncoder(w).Encode(status)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Controller did not return status")
	}
}

func (s MSPServer) handleMspSetDataPath(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	fmt.Fprintf(w, "Not implemented")
}

func (s MSPServer) handleMspEstablishMongodState(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	fmt.Fprintf(w, "Not implemented")
}

func (s MSPServer) Listen() {
	http.ListenAndServe(":8081", s.router)
}