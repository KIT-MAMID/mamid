package main

import (
	"github.com/KIT-MAMID/mamid/masterslaveprotocol"
)

func main() {
	controller := NewController()
	server := masterslaveprotocol.NewMSPServer(controller)
	server.Listen()
}


