package mamidslave

import "github.com/KIT-MAMID/mamid/masterslaveprotocol"

type Controller struct {

}

func NewController() *Controller {
	return &Controller{}
}

func (c Controller) MspSetDataPath(path string) error {
	return nil
}

func (c Controller) MspStatusRequest() []masterslaveprotocol.Mongod {
	return []masterslaveprotocol.Mongod{
		masterslaveprotocol.Mongod{Port: 1234, ReplSetName: "hello world"},
	}
}

func (c Controller) MspEstablishMongodState(m masterslaveprotocol.Mongod) error {
	return nil
}