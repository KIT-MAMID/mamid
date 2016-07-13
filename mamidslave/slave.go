package mamidslave

import (
	"github.com/KIT-MAMID/mamid/masterslaveprotocol"
)

func main() {
	controller := mamidslave.NewController()
	server := masterslaveprotocol.NewMSPServer(controller)
	server.Listen()
}


