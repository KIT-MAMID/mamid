package slave

import "github.com/KIT-MAMID/mamid/msp"

type Controller struct {
}

func NewController() *Controller {
	return &Controller{}
}

func (c Controller) RequestStatus() ([]msp.Mongod, *msp.SlaveError) {
	return []msp.Mongod{
		msp.Mongod{Port: 1234, ReplicaSetName: "hello world", State: msp.MongodStateRunning},
	}, nil
}

func (c Controller) EstablishMongodState(m msp.Mongod) *msp.SlaveError {
	return &msp.SlaveError{Description: "Not Implemented"}
}
