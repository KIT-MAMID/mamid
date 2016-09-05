package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var lastProblems map[uint]Problem
var notifiers []Notifier

var log = logrus.WithField("module", "slave")

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	var p Parser
	var email EmailNotifier
	// Load contacts from ini file
	if len(os.Args) != 2 {
		log.Println("No config file supplied! Usage: notifier <config file>")
		return
	}
	config, configError := p.ParseConfig(os.Args[1])
	if configError != nil {
		log.Fatalf("Error loading config: %#v", configError)
	}
	contacts, contactsParseErr := p.Parse(config.contactsFile)

	if contactsParseErr != nil {
		log.Fatalf("Error loading contacts file `%s`: %#v", config.contactsFile, contactsParseErr)
	}

	emailContacts := make([]*EmailContact, 0)
	for i := 0; i < len(contacts); i++ {
		switch t := contacts[i].(type) {
		case EmailContact:
			emailContacts = append(emailContacts, &t)
		}
	}
	email.Contacts = emailContacts
	email.Relay = config.relay
	email.MamidHost = config.apiHost
	lastProblems = make(map[uint]Problem)
	notifiers = append(notifiers, &email)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		os.Exit(0)
	}()
	// Initiate api client and load certs
	certPool, err := x509.SystemCertPool()
	if err != nil {
		log.Fatalf("Error loading system keystore: %#v", err)
	}
	if config.masterCA != "" {
		cert, err := loadCertificateFromFile(config.masterCA)
		if err != nil {
			log.Fatalf("Error loading matser CA file `%s`: %#v", config.masterCA, err)
		}
		certPool.AddCert(cert)
	}
	var apiClient APIClient
	var httpTransport *http.Transport
	httpTransport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: certPool,
		},
	}
	if config.apiCert != "" {
		clientAuthCert, err := tls.LoadX509KeyPair(config.apiCert, config.apiKey)
		if err != nil {
			log.Fatalf("Error loading keypair `%s`, `%s`: %#v", config.apiCert, config.apiKey, err)
		}
		httpTransport.TLSClientConfig.Certificates = []tls.Certificate{clientAuthCert}
	}

	apiClient = APIClient{
		httpClient: http.Client{
			Transport: httpTransport,
		},
	}
	for {
		//receive Problems through API
		currentProblems, err := apiClient.Receive(config.apiHost)
		if err != nil {
			log.Errorf("Error querying API: %#v", err)
		}
		currentProblems = diffProblems(currentProblems)
		for i := 0; i < len(currentProblems); i++ {
			notify(currentProblems[i])
		}
		time.Sleep(10 * time.Second)
	}

}
func diffProblems(received []Problem) []Problem {
	// Clean old problems
	for id := range lastProblems {
		var contained bool
		for i := 0; i < len(received); i++ {
			contained = received[i].Id == id
			if contained {
				break
			}
		}
		if !contained {
			delete(lastProblems, id)
		}
	}
	// Add problems to the map of already notified problems and remove already notified problems from the resulting slice
	for i := 0; i < len(received); i++ {
		if _, ok := lastProblems[received[i].Id]; ok {
			received = append(received[:i], received[i+1:]...)
			i--
			continue
		}
		lastProblems[received[i].Id] = received[i]
	}
	return received
}

func notify(problem Problem) {
	for i := 0; i < len(notifiers); i++ {
		err := notifiers[i].SendProblem(problem)
		if err != nil {
			log.Errorf("Error sending notification: %#v", err)
		}
	}
}

func loadCertificateFromFile(file string) (cert *x509.Certificate, err error) {
	certFile, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	block, _ := pem.Decode(certFile)
	cert, err = x509.ParseCertificate(block.Bytes)
	return
}
