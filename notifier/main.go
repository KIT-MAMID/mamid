package main

import (
	"os"
	"os/signal"
)

func main () {
	// Wait forever
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c
	os.Exit(0)
}
