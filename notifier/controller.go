package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

var p Parser
var email EmailNotifier
var lastProblems map[uint]Problem
var notifiers []Notifier
var apiClient APIClient

func main() {
	p.Parse("contacts.txt")
	lastProblems = make(map[uint]Problem)
	notifiers = append(notifiers, &email)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		os.Exit(0)
	}()
	for {
		//receive Problems through API
		currentProblems, err := apiClient.Receive("localhost:8080")
		fmt.Print(err)
		currentProblems = diffProblems(currentProblems)
		for i := 0; i < len(currentProblems); i++ {
			print(currentProblems[i].Description)
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
		notifiers[i].SendProblem(problem)
	}
}
