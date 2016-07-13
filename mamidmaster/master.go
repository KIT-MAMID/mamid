package mamidmaster

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/masterslaveprotocol"
)

func main() {
	target := masterslaveprotocol.HostPort{"localhost", 8081}
	client := masterslaveprotocol.NewMSPClient(target)
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