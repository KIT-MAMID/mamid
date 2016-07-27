package main

import (
	"os"
	"os/signal"
	"fmt"
)
var email EmailNotifier
var lastProblems []Problem
//var notifiers []Notifier
type Problem struct{

}

func main() {
	// Wait forever
	//Test
	var problem Problem
	notify(problem)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	var currentProblems []Problem
        currentProblems = diffProblems(currentProblems)
        for i := 0; i < len(currentProblems); i++{
                notify(currentProblems[i])
        }
	<-c
	fmt.Println("A1");
	os.Exit(0)
	fmt.Println("B");
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
	email.SendProblem(problem)
}

