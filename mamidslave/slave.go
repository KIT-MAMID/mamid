package mamidslave

import (
	"github.com/KIT-MAMID/mamid/masterslaveprotocol"
	"github.com/KIT-MAMID/mamid/mamidslave"
)

func main() {
	controller := mamidslave.NewController()
	server := masterslaveprotocol.NewMSPServer(controller)
	server.Listen()
}


