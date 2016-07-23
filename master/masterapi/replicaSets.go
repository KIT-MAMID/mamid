package masterapi

import (
	"encoding/json"
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"net/http"
)

type ReplicaSet struct {
	ID                              uint   `json:"id"`
	Name                            string `json:"name"`
	PersistentNodeCount             uint   `json:"presistent_node_count"`
	VolatileNodeCount               uint   `json:"volatile_node_count"`
	ConfigureAsShardingConfigServer bool   `json:"configure_as_sharding_config_server"`
}

func (m *MasterAPI) ReplicaSetIndex(w http.ResponseWriter, r *http.Request) {
	var replicasets []*model.ReplicaSet
	err := m.DB.Order("id", false).Find(&replicasets).Error
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	out := make([]*ReplicaSet, len(replicasets))
	for i, v := range replicasets {
		out[i] = ProjectModelReplicaSetToReplicaSet(v)
	}
	json.NewEncoder(w).Encode(out)
}
