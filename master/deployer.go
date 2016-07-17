package master

import (
	"github.com/KIT-MAMID/mamid/model"
)

/*
  Listens on the bus for state mismatches and tries to solve them by pushing the desired state to the Mongod
*/
type Deployer struct {
}

func (d *Deployer) Run() {
}

func (d *Deployer) pushMongodState(mongod model.Mongod) {

}
