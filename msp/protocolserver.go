package msp

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
)

type Consumer interface {
	RequestStatus() ([]Mongod, *Error)
	EstablishMongodState(m Mongod) *Error
	RsInitiate(m RsInitiateMessage) *Error
}

type Listener struct {
	listenString string
	consumer     Consumer
	router       *mux.Router
	certFile     string
	keyFile      string
	tlsConfig    *tls.Config
}

func NewServer(listener Consumer, listenString string, caFile string, certFile string, keyFile string) *Listener {
	s := new(Listener)
	s.consumer = listener
	s.listenString = listenString
	s.certFile = certFile
	s.keyFile = keyFile

	certPool := x509.NewCertPool()
	caCertContent, err := ioutil.ReadFile(caFile)
	if err != nil {
		mspLog.Fatal(err)
	}
	certPool.AppendCertsFromPEM(caCertContent)
	s.tlsConfig = &tls.Config{
		ClientCAs:  certPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}
	s.tlsConfig.BuildNameToCertificate()

	s.router = mux.NewRouter().StrictSlash(true)
	s.router.Methods("GET").Path("/msp/status").Name("RequestStatus").HandlerFunc(s.handleRequestStatus)
	s.router.Methods("POST").Path("/msp/establishMongodState").Name("EstablishMongodState").HandlerFunc(s.handleMspEstablishMongodState)
	s.router.Methods("POST").Path("/msp/rsInitiate").Name("RsInitiate").HandlerFunc(s.handleRsInitiate)

	return s
}

func (s Listener) handleRequestStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.consumer.RequestStatus()
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
	err := s.consumer.EstablishMongodState(mongodState)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
	}
}

func (s Listener) handleRsInitiate(w http.ResponseWriter, r *http.Request) {
	var rsInitiateMessage RsInitiateMessage
	json.NewDecoder(r.Body).Decode(&rsInitiateMessage) //TODO Check decode error
	err := s.consumer.RsInitiate(rsInitiateMessage)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
	}
}

func (s Listener) Run() error {
	server := &http.Server{
		TLSConfig: s.tlsConfig,
		Addr:      s.listenString,
		Handler:   s.router,
	}
	return server.ListenAndServeTLS(s.certFile, s.keyFile)
}
