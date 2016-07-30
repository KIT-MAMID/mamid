package master

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/jinzhu/gorm"
	"time"
)

type ProblemManager struct {
	DB             *gorm.DB
	BusReadChannel chan interface{}
}

func (p *ProblemManager) Run() {
	for {
		message := <-p.BusReadChannel
		switch message.(type) {
		case model.ConnectionStatus:
			connStatus := message.(model.ConnectionStatus)
			if connStatus.Unreachable {
				var problem model.Problem
				p.DB.Where(&model.Problem{
					ProblemType: model.ProblemTypeConnection,
					SlaveID:     connStatus.Slave.ID,
				}).Assign(&model.Problem{
					Description: fmt.Sprintf("Slave %s is unreachable", connStatus.Slave.Hostname),
					LastUpdated: time.Now(),
				}).Attrs(&model.Problem{
					FirstOccurred: time.Now(),
				}).FirstOrCreate(&problem)
			} else {
				p.DB.Where(&model.Problem{
					ProblemType: model.ProblemTypeConnection,
					SlaveID:     connStatus.Slave.ID,
				}).Delete(&model.Problem{})
			}
		}
	}

}

func (p *ProblemManager) generateProblem(e model.StatusMessage) model.Problem {
	return model.Problem{}
}

func (p *ProblemManager) updateProblem(problem model.Problem) {

}

func (p *ProblemManager) removeProblem(problem model.Problem) {

}
