package slave

import "github.com/KIT-MAMID/mamid/msp"

type Controller struct {

}

func NewController() *Controller {
	return &Controller{}
}

func (c Controller) MspSetDataPath(path string) msp.MSPError {
	return msp.NewMSPError("Not Implemented")
}

func (c Controller) MspStatusRequest() ([]msp.Mongod, msp.MSPError) {
	return []msp.Mongod{
		msp.Mongod{Port: 1234, ReplSetName: "hello world", State: msp.MongodStateRunning},
	}, nil
}

func (c Controller) MspEstablishMongodState(m msp.Mongod) msp.MSPError {
	return msp.NewMSPError("Not Implemented")
}
