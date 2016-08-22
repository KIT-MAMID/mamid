package master

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/Sirupsen/logrus"
	"time"
)

type ProblemManager struct {
	DB             *model.DB
	BusReadChannel <-chan interface{}
}

var pmLog = logrus.WithField("module", "problem_manager")

func (p *ProblemManager) Run() {
	for {
		message := <-p.BusReadChannel
		tx := p.DB.Begin()
		switch message.(type) {
		case model.ConnectionStatus:
			connStatus := message.(model.ConnectionStatus)
			if connStatus.Unreachable {
				var problem model.Problem
				tx.Where(&model.Problem{
					ProblemType: model.ProblemTypeConnection,
					SlaveID:     model.NullIntValue(connStatus.Slave.ID),
				}).Assign(&model.Problem{
					Description: fmt.Sprintf("Slave `%s` is unreachable", connStatus.Slave.Hostname),
					LastUpdated: time.Now(),
				}).Attrs(&model.Problem{
					FirstOccurred: time.Now(),
				}).FirstOrCreate(&problem)
			} else {
				tx.Where(&model.Problem{
					ProblemType: model.ProblemTypeConnection,
					SlaveID:     model.NullIntValue(connStatus.Slave.ID),
				}).Delete(&model.Problem{})
			}
		case model.DesiredReplicaSetConstraintStatus:
			constrStatus := message.(model.DesiredReplicaSetConstraintStatus)
			if constrStatus.Unsatisfied {
				var problem model.Problem
				tx.Where(&model.Problem{
					ProblemType:  model.ProblemTypeDesiredReplicaSetConstraint,
					ReplicaSetID: model.NullIntValue(constrStatus.ReplicaSet.ID),
				}).Assign(&model.Problem{
					Description: fmt.Sprintf("Replica Set `%s` with unsatisfiable constraints", constrStatus.ReplicaSet.Name),
					LongDescription: fmt.Sprintf(
						"Not enough free ports on suitable Slaves are available.\n"+
							"This Replica Set's member counts are less than desired (%d/%d persistent, %d/%d volatile).",
						constrStatus.ConfiguredPersistentCount, constrStatus.ReplicaSet.PersistentMemberCount,
						constrStatus.ConfiguredVolatileCount, constrStatus.ReplicaSet.VolatileMemberCount),
					LastUpdated: time.Now(),
				}).Attrs(&model.Problem{
					FirstOccurred: time.Now(),
				}).FirstOrCreate(&problem)
			} else {
				tx.Where(&model.Problem{
					ProblemType:  model.ProblemTypeDesiredReplicaSetConstraint,
					ReplicaSetID: model.NullIntValue(constrStatus.ReplicaSet.ID),
				}).Delete(&model.Problem{})
			}
		case model.ObservedReplicaSetConstraintStatus:
			constrStatus := message.(model.ObservedReplicaSetConstraintStatus)
			if constrStatus.Unsatisfied {
				var problem model.Problem
				tx.Where(&model.Problem{
					ProblemType:  model.ProblemTypeObservedReplicaSetConstraint,
					ReplicaSetID: model.NullIntValue(constrStatus.ReplicaSet.ID),
				}).Assign(&model.Problem{
					Description: fmt.Sprintf("Replica Set `%s` is degraded", constrStatus.ReplicaSet.Name),
					LongDescription: fmt.Sprintf(
						"One or more Mongods in this Replica Set are not running (%d/%d persistent, %d/%d volatile).",
						constrStatus.ActualPersistentCount, constrStatus.ConfiguredPersistentCount,
						constrStatus.ActualVolatileCount, constrStatus.ConfiguredVolatileCount),
					LastUpdated: time.Now(),
				}).Attrs(&model.Problem{
					FirstOccurred: time.Now(),
				}).FirstOrCreate(&problem)
			} else {
				tx.Where(&model.Problem{
					ProblemType:  model.ProblemTypeObservedReplicaSetConstraint,
					ReplicaSetID: model.NullIntValue(constrStatus.ReplicaSet.ID),
				}).Delete(&model.Problem{})
			}
		}
		tx.Commit()
	}

}

func (p *ProblemManager) generateProblem(e model.StatusMessage) model.Problem {
	return model.Problem{}
}

func (p *ProblemManager) updateProblem(problem model.Problem) {

}

func (p *ProblemManager) removeProblem(problem model.Problem) {

}
