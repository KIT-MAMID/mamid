package main

import (
	"os"
	"os/signal"
	//"fmt"
)
var p Parser
var email EmailNotifier
var lastProblems []Problem
var notifiers []Notifier

func main() {
	p.Parse("/home/niklas/GO/src/github.com/KIT-MAMID/mamid/notifier/contacts.txt")
	notifiers = append(notifiers, &email)
	// Wait forever
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	var currentProblems []Problem
        currentProblems = diffProblems(currentProblems)
        for i := 0; i < len(currentProblems); i++{
                notify(currentProblems[i])
        }
	<-c
	os.Exit(0)
	//receive Problems through API
}
func diffProblems(received []Problem) []Problem {
	for i := 0; i < len(received); i++ {
		for j := 0; j < len(lastProblems); j++{
			if(true){
				received = append(received[:i], received[i+1:]...)
			}
		}
	}
	return received
}

func notify(problem Problem){
	for i :=0; i < len(notifiers); i++{
		notifiers[i].SendProblem(problem)
	}
}



