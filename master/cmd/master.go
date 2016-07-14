package main

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/msp"
)

func main() {
	target := msp.HostPort{"localhost", 8081}
	client := msp.NewMSPClient(target)
	status, err := client.MspStatusRequest()
	if err == nil {
		fmt.Printf("status: %s\n", status)
	} else {
		fmt.Println(err.Error())
	}

	err = client.MspSetDataPath("/data")
	if err == nil {
		fmt.Println("Set path ok")
	} else {
		fmt.Printf("Set path error: %s\n", err.Error())
	}
}
