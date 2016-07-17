package main

import (
	"github.com/KIT-MAMID/mamid/master/masterapi"
)

func main() {
	masterApiServer := masterapi.MasterAPIServer{}
	masterApiServer.Run()
}
