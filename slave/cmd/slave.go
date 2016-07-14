package main

import (
	"github.com/KIT-MAMID/mamid/msp"
	. "github.com/KIT-MAMID/mamid/slave"
)

func main() {
	controller := NewController()
	server := msp.NewMSPServer(controller)
	server.Listen()
}


