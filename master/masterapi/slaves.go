package masterapi

import (
	"net/http"
	"encoding/json"
	"github.com/gorilla/mux"
	"strconv"
	"fmt"
)

var slaves = []Slave{
	Slave{Id: 0, Hostname: "mksuns31", Port: 1912, MongodPortRangeBegin: 20000, MongodPortRangeEnd:20100, PersistantStorage:true, RootDataDirectory:"/home/mongo/data", State: "active"},
	Slave{Id: 1, Hostname: "mksuns32", Port: 1912, MongodPortRangeBegin: 20000, MongodPortRangeEnd:20001, PersistantStorage:false, RootDataDirectory:"/home/mongo/data", State: "active"},
	Slave{Id: 2, Hostname: "mksuns33", Port: 1912, MongodPortRangeBegin: 20000, MongodPortRangeEnd:20001, PersistantStorage:false, RootDataDirectory:"/home/mongo/data", State: "active"},
	Slave{Id: 3, Hostname: "mksuns34", Port: 1912, MongodPortRangeBegin: 20000, MongodPortRangeEnd:20001, PersistantStorage:false, RootDataDirectory:"/home/mongo/data", State: "active"},
}

type Slave struct {
	Id uint			   `json:"id"`
	Hostname string		   `json:"hostname"`
	Port uint		   `json:"slave_port"`
	MongodPortRangeBegin uint  `json:"mongod_port_range_begin"` //inclusive
	MongodPortRangeEnd uint    `json:"mongod_port_range_end"`   //exclusive
	PersistantStorage bool     `json:"persistant_storage"`
	RootDataDirectory string   `json:"root_data_directory"`
	State string               `json:"state"`
}

func SlaveIndex(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(slaves)
}

func SlaveById(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["slaveId"]
	id64, err := strconv.ParseUint(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := uint(id64)
	for _,slave := range slaves {
		if slave.Id == id {
			json.NewEncoder(w).Encode(slave)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	return
}

func SlavePut(w http.ResponseWriter, r *http.Request) {
	var postSlave Slave
	err := json.NewDecoder(r.Body).Decode(&postSlave)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Could not parse object (%s)", err.Error())
		return
	}

	var maxId uint = 0;
	for _, slave := range slaves {
		if slave.Id > maxId {
			maxId = slave.Id
		}
	}
	postSlave.Id = maxId + 1

	slaves = append(slaves, postSlave)
	return
}

func SlaveUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["slaveId"]
	id64, err := strconv.ParseUint(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := uint(id64)

	var postSlave Slave
	err = json.NewDecoder(r.Body).Decode(&postSlave)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Could not parse object (%s)", err.Error())
		return
	}

	for idx, slave := range slaves {
		if slave.Id == id {
			if postSlave.Id != id {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "You can not change the id of an object")
				return
			}
			slaves[idx] = postSlave
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	return
}

func SlaveDelete(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["slaveId"]
	id64, err := strconv.ParseUint(idStr, 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := uint(id64)

	var slaveIdx int = -1
	for idx, slave := range slaves {
		if slave.Id == id {
			slaveIdx = idx
		}
	}
	if slaveIdx == -1 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	slaves = append(slaves[:slaveIdx], slaves[slaveIdx+1:]...)
}