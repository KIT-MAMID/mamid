package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

var p Parser
var email EmailNotifier
var lastProblems []Problem
var notifiers []Notifier
var apiClient APIClient

func main() {
	p.Parse("contacts.txt")
	notifiers = append(notifiers, &email)
	// Wait forever
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
	for i := 0; i < len(received); i++ {
		for j := 0; j < len(lastProblems); j++ {
			if true {
				received = append(received[:i], received[i+1:]...)
			}
		}
	}
	return received
}

func notify(problem Problem) {
	for i := 0; i < len(notifiers); i++ {
		notifiers[i].SendProblem(problem)
	}
}
