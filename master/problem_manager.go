package master

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/jinzhu/gorm"
	"time"
)

type ProblemManager struct {
	DB             *gorm.DB
	BusReadChannel <-chan interface{}
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
		case model.DesiredReplicaSetConstraintStatus:
			constrStatus := message.(model.DesiredReplicaSetConstraintStatus)
			if constrStatus.Unsatisfied {
				var problem model.Problem
				p.DB.Where(&model.Problem{
					ProblemType:  model.ProblemTypeDesiredReplicaSetConstraint,
					ReplicaSetID: constrStatus.ReplicaSet.ID,
				}).Assign(&model.Problem{
					Description: fmt.Sprintf("Replica set %s is degraded", constrStatus.ReplicaSet.Name),
					LongDescription: fmt.Sprintf(
						"Not enough free ports are available."+
							"This replica set is now configured to have %d persistent and %d volatile mongods"+
							" instead of the %d persistent and %d volatile mongods it should have",
						constrStatus.ActualPersistentCount, constrStatus.ActualVolatileCount,
						constrStatus.ReplicaSet.PersistentMemberCount, constrStatus.ReplicaSet.VolatileMemberCount),
					LastUpdated: time.Now(),
				}).Attrs(&model.Problem{
					FirstOccurred: time.Now(),
				}).FirstOrCreate(&problem)
			} else {
				p.DB.Where(&model.Problem{
					ProblemType:  model.ProblemTypeDesiredReplicaSetConstraint,
					ReplicaSetID: constrStatus.ReplicaSet.ID,
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
