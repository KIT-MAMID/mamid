package main

import "github.com/KIT-MAMID/mamid/masterslaveprotocol"

type Controller struct {

}

func NewController() *Controller {
	return &Controller{}
}

func (c Controller) MspSetDataPath(path string) masterslaveprotocol.MSPError {
	return masterslaveprotocol.NewMSPError("Not Implemented")
}

func (c Controller) MspStatusRequest() ([]masterslaveprotocol.Mongod, masterslaveprotocol.MSPError) {
	return []masterslaveprotocol.Mongod{
		masterslaveprotocol.Mongod{Port: 1234, ReplSetName: "hello world", State: masterslaveprotocol.MongodStateRunning},
	}, nil
}

func (c Controller) MspEstablishMongodState(m masterslaveprotocol.Mongod) masterslaveprotocol.MSPError {
	return masterslaveprotocol.NewMSPError("Not Implemented")
}
