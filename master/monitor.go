package master

import (
	"github.com/KIT-MAMID/mamid/model"
)

type Monitor struct {
}

func (m *Monitor) Run() {

}

func (m *Monitor) compareStates(mongod model.Mongod) model.MongodMatchStatus {
	return model.MongodMatchStatus{}
}
