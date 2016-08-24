package masterapi

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/master"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"net/http"
)

var masterapiLog = logrus.WithField("module", "masterapi")

type MasterAPI struct {
	DB               *model.DB
	ClusterAllocator *master.ClusterAllocator
	Router           *mux.Router
}

func (m *MasterAPI) Setup() {

	m.Router.Methods("GET").Path("/slaves").Name("SlaveIndex").HandlerFunc(m.SlaveIndex)
	m.Router.Methods("GET").Path("/slaves/{slaveId}").Name("SlaveById").HandlerFunc(m.SlaveById)
	m.Router.Methods("PUT").Path("/slaves").Name("SlavePut").HandlerFunc(m.SlavePut)
	m.Router.Methods("POST").Path("/slaves/{slaveId}").Name("SlaveUpdate").HandlerFunc(m.SlaveUpdate)
	m.Router.Methods("DELETE").Path("/slaves/{slaveId}").Name("SlaveDelete").HandlerFunc(m.SlaveDelete)

	m.Router.Methods("GET").Path("/replicasets").Name("ReplicaSetIndex").HandlerFunc(m.ReplicaSetIndex)
	m.Router.Methods("GET").Path("/replicasets/{replicasetId}").Name("ReplicaSetById").HandlerFunc(m.ReplicaSetById)
	m.Router.Methods("PUT").Path("/replicasets").Name("ReplicaSetPut").HandlerFunc(m.ReplicaSetPut)
	m.Router.Methods("POST").Path("/replicasets/{replicasetId}").Name("ReplicaSetUpdate").HandlerFunc(m.ReplicaSetUpdate)
	m.Router.Methods("DELETE").Path("/replicasets/{replicasetId}").Name("ReplicaSetDelete").HandlerFunc(m.ReplicaSetDelete)
	m.Router.Methods("GET").Path("/replicasets/{replicasetId}/slaves").Name("ReplicaSetGetSlaves").HandlerFunc(m.ReplicaSetGetSlaves)

	m.Router.Methods("GET").Path("/riskgroups").Name("RiskGroupIndex").HandlerFunc(m.RiskGroupIndex)
	m.Router.Methods("GET").Path("/riskgroups/{riskgroupId}").Name("RiskGroupById").HandlerFunc(m.RiskGroupById)
	m.Router.Methods("PUT").Path("/riskgroups").Name("RiskGroupPut").HandlerFunc(m.RiskGroupPut)
	m.Router.Methods("POST").Path("/riskgroups/{riskgroupId}").Name("RiskGroupUpdate").HandlerFunc(m.RiskGroupUpdate)
	m.Router.Methods("DELETE").Path("/riskgroups/{riskgroupId}").Name("RiskGroupDelete").HandlerFunc(m.RiskGroupDelete)
	m.Router.Methods("GET").Path("/riskgroups/{riskgroupId}/slaves").Name("RiskGroupGetSlaves").HandlerFunc(m.RiskGroupGetSlaves)
	m.Router.Methods("PUT").Path("/riskgroups/{riskgroupId}/slaves/{slaveId}").Name("RiskGroupAssignSlave").HandlerFunc(m.RiskGroupAssignSlave)
	m.Router.Methods("DELETE").Path("/riskgroups/{riskgroupId}/slaves/{slaveId}").Name("RiskGroupRemoveSlave").HandlerFunc(m.RiskGroupRemoveSlave)

	m.Router.Methods("GET").Path("/problems").Name("ProblemIndex").HandlerFunc(m.ProblemIndex)
	m.Router.Methods("GET").Path("/problems/{problemId}").Name("ProblemById").HandlerFunc(m.ProblemById)
	m.Router.Methods("GET").Path("/slaves/{slaveId}/problems").Name("ProblemBySlave").HandlerFunc(m.ProblemBySlave)
	m.Router.Methods("GET").Path("/replicasets/{replicasetId}/problems").Name("ProblemByReplicaSet").HandlerFunc(m.ProblemByReplicaSet)

	m.Router.Methods("GET").Path("/slaves/{slaveId}/mongods").Name("MongodsBySlave").HandlerFunc(m.MongodsBySlave)
	m.Router.Methods("GET").Path("/replicasets/{replicasetId}/mongods").Name("MongodsByReplicaSet").HandlerFunc(m.MongodsByReplicaSet)

	m.Router.Methods("GET").Path("/system/secret").Name("SecretGet").HandlerFunc(m.SecretGet)
	m.Router.Methods("POST").Path("/system/secret").Name("SecretUpdate").HandlerFunc(m.SecretUpdate)
	m.Router.Methods("GET").Path("/system/runningconfig").Name("DumpDB").HandlerFunc(m.DumpDB)
	m.Router.Methods("POST").Path("/system/runningconfig").Name("RestoreDB").HandlerFunc(m.RestoreDB)
}

func (m *MasterAPI) attemptClusterAllocator(tx *gorm.DB, w http.ResponseWriter) (err error) {
	err = m.ClusterAllocator.CompileMongodLayout(tx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "cluster allocator failure: %s\n", err)
		if err = tx.Rollback().Error; err != nil {
			fmt.Fprintf(w, "cluster allocator rollback failure: %s\n", err)
		}
	}
	return err
}
