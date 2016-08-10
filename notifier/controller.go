package main

import (
	"log"
	"os"
	"os/signal"
	"time"
)

var lastProblems map[uint]Problem
var notifiers []Notifier

func main() {
	var p Parser
	var email EmailNotifier
	// Load contacts from ini file
	if len(os.Args) != 2 {
		log.Println("No config file supplied! Usage: notifier <config file>")
		return
	}
	relay, apiHost, contactsFile, configError := p.ParseConfig(os.Args[1])
	if configError != nil {
		log.Println("Error loading config:", configError)
		return
	}
	contacts, _ := p.Parse(contactsFile)
	emailContacts := make([]*EmailContact, 0)
	for i := 0; i < len(contacts); i++ {
		switch t := contacts[i].(type) {
		case EmailContact:
			emailContacts = append(emailContacts, &t)
		}
	}
	email.Contacts = emailContacts
	email.Relay = relay
	lastProblems = make(map[uint]Problem)
	notifiers = append(notifiers, &email)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		os.Exit(0)
	}()
	var apiClient APIClient
	for {
		//receive Problems through API
		currentProblems, err := apiClient.Receive(apiHost)
		if err != nil {
			log.Println("Error querying API:", err)
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
			log.Println("Error sending notification:", err)
		}
	}
}
