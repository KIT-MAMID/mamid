package master

import (
	"fmt"
	. "github.com/KIT-MAMID/mamid/model"
	"github.com/Sirupsen/logrus"
	"github.com/jinzhu/gorm"
	"time"
)

var caLog = logrus.WithField("module", "cluster_allocator")

type ClusterAllocator struct {
	BusWriteChannel *chan<- interface{}
}

type persistence uint

const (
	Persistent persistence = 0
	Volatile   persistence = 1
)

type memberCountTuple map[persistence]uint

func (c *ClusterAllocator) Run(db *DB) {
	ticker := time.NewTicker(11 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				caLog.Info("Periodic cluster allocator run")
				tx := db.Begin()
				compileErr := c.CompileMongodLayout(tx)
				if compileErr != nil {
					caLog.WithError(compileErr).Error("Periodic cluster allocator run failed")
					continue
				}
				if commitErr := tx.Commit().Error; commitErr != nil {
					caLog.WithError(commitErr).Error("Periodic cluster allocator commit failed")
					continue
				}

			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (c *ClusterAllocator) CompileMongodLayout(tx *gorm.DB) (err error) {

	defer func() {
		r := recover()
		if r == nil {
			return
		}
		switch r {
		case r == nil:
			return
		case r == gorm.ErrInvalidTransaction:
			err = r.(error)
		default:
			panic(r)
		}
	}()

	// mark orphaned Mongod.DesiredState as destroyed
	// orphaned = Mongods whose parent Replica Set has been destroyed
	// NOTE: slaves do not cause orphaned Mongods as only slaves without Monogds can be deleted from the DB
	caLog.Debug("updating desired state of orphaned Mongods")
	markOrphanedMongodsDestroyedRes := tx.Exec(`
                       UPDATE mongod_states SET execution_state=?
                       WHERE id IN (
		         SELECT desired_state_id FROM mongods m WHERE replica_set_id IS NULL
		       )`, MongodExecutionStateDestroyed)
	if markOrphanedMongodsDestroyedRes.Error != nil {
		caLog.Errorf("error updating desired state of orphaned Mongods: %s", markOrphanedMongodsDestroyedRes.Error)
		panic(markOrphanedMongodsDestroyedRes.Error)
	} else {
		caLog.Debugf("marked `%d` Mongod.DesiredState of orphaned Mongods as `destroyed`", markOrphanedMongodsDestroyedRes.RowsAffected)
	}

	// remove destroyed Mongods from the database
	// destroyed: desired & observed state is `destroyed` OR no observed state
	caLog.Debug("removing destroyed Mongods from the database")
	removeDestroyedMongodsRes := tx.Exec(`
		DELETE FROM mongods WHERE id IN ( -- we use cascadation to also delete the mongod states
			-- all mongod id's whose desired and observed state are in ExecutionState destroyed
			SELECT m.id
			FROM mongods m
			LEFT OUTER JOIN mongod_states desired_state ON m.desired_state_id = desired_state.id
			LEFT OUTER JOIN mongod_states observed_state ON m.observed_state_id = observed_state.id
			WHERE
				desired_state.execution_state = ?
				AND
				(
					observed_state.execution_state = ?
					OR
					observed_state.execution_state IS NULL --don't have observed state
				)
		)
	`, MongodExecutionStateDestroyed, MongodExecutionStateDestroyed)
	if removeDestroyedMongodsRes.Error != nil {
		caLog.Errorf("error removing destroyed Mongods from the database: %s", removeDestroyedMongodsRes.Error)
		panic(removeDestroyedMongodsRes.Error)
	} else {
		caLog.Debugf("removed `%d` destroyed Mongods from the database", removeDestroyedMongodsRes.RowsAffected)
	}

	// list of replica sets with number of excess mongods
	replicaSets, err := tx.Raw(`SELECT
			r.id,
			(SELECT COUNT(*) FROM replica_set_effective_members WHERE replica_set_id = r.id AND persistent_storage = ?)
				- r.persistent_member_count AS deletable_persistent,
			(SELECT COUNT(*) FROM replica_set_effective_members WHERE replica_set_id = r.id AND persistent_storage = ?)
				- r.volatile_member_count AS deletable_volatile
		    FROM replica_sets r`, true, false,
	).Rows()

	if err != nil {
		panic(err)
	}
	defer replicaSets.Close()

	for replicaSets.Next() {

		var (
			replicaSetID                             uint
			deletable_persistent, deletable_volatile int
		)
		err := replicaSets.Scan(&replicaSetID, &deletable_persistent, &deletable_volatile)
		if err != nil {
			panic(err)
		}

		for _, p := range []persistence{Persistent, Volatile} {

			var deletable_count int
			if p.PersistentStorage() {
				deletable_count = deletable_persistent
			} else {
				deletable_count = deletable_volatile
			}

			// Assert that deletable_count >= 0
			// SQLite will not LIMIT if deletable_count is negative!
			if deletable_count < 0 {
				continue
			}

			caLog.Infof("removing excess mongods for replica set `%#v`: up to `%d` `%s` mongods", replicaSetID, deletable_count, p)

			var deletableMongds []*Mongod

			err := tx.Raw(`SELECT m.*
				FROM replica_sets r
				JOIN mongods m ON m.replica_set_id = r.id
				JOIN slaves s ON s.id = m.parent_slave_id
				JOIN slave_utilization su ON s.id = su.id
				WHERE
					r.id = ?
					AND s.persistent_storage = ?
					AND s.configured_state != ?
				ORDER BY (CASE WHEN s.configured_state = ? THEN 1 ELSE 2 END) ASC, su.utilization DESC
				LIMIT ?`, replicaSetID, p.PersistentStorage(), SlaveStateMaintenance, SlaveStateDisabled, deletable_count,
			).Find(&deletableMongds).Error
			if err != nil {
				panic(err)
			}

			caLog.Infof("setting %d mongods for replica set `%#v` to desired state `destroyed`", len(deletableMongds), replicaSetID)

			for _, m := range deletableMongds {
				caLog.Debugf("setting desired mongod_state of mongod `%#v` to `destroyed`", m)

				res := tx.Exec("UPDATE mongod_states SET execution_state=? WHERE id=?", MongodExecutionStateDestroyed, m.DesiredStateID)
				if res.Error != nil {
					panic(res.Error)
				}

				if res.RowsAffected < 1 {
					caLog.Errorf("setting desired mongod_state of mongod `%#v` to `destroyed` did not affect any row", m)
				}
				if res.RowsAffected > 1 {
					caLog.Errorf("internal inconsistency: setting desired mongod_state of mongod `%#v` to `destroyed` affected more than one row", m)
				}

			}

		}

	}

	//All unsatisfiable replica sets (independent of persistence)
	unsatisfiable_replica_set_ids := []int64{}

	// Now add new members

	for _, p := range []persistence{Persistent, Volatile} {

		var memberCountColumnName string
		if p.PersistentStorage() {
			memberCountColumnName = "persistent_member_count"
		} else {
			memberCountColumnName = "volatile_member_count"
		}

		//Unsatisfiable replica sets for the current persistence
		unsatisfiable_replica_set_ids_by_persistance := []int64{0} // we always start at 1, this is a workaround for the statement generator producing (NULL) in case of an empty set otherwise

		for {

			replicaSet := struct {
				ReplicaSet
				ConfiguredMemberCount int
			}{}

			// HEAD of degraded replica sets PQ
			res := tx.Raw(`SELECT r.*, COUNT(DISTINCT members.mongod_id) as "configured_member_count"
					FROM replica_sets r
					LEFT OUTER JOIN replica_set_configured_members members
						ON r.id = members.replica_set_id
						AND members.persistent_storage = ?
					WHERE
						r.`+memberCountColumnName+` != 0
						AND
						r.id NOT IN (?)
					GROUP BY r.id
					HAVING COUNT(DISTINCT members.mongod_id) < r.`+memberCountColumnName+`
					ORDER BY COUNT(DISTINCT members.mongod_id) / r.`+memberCountColumnName+`
					LIMIT 1`, p.PersistentStorage(), unsatisfiable_replica_set_ids_by_persistance,
			).Scan(&replicaSet)

			if res.RecordNotFound() {
				caLog.Infof("finished repairing degraded replica sets in need of `%s` members", p)
				break
			} else if res.Error != nil {
				panic(res.Error)
			}

			caLog.Debugf("looking for least busy `%s` slave suitable as mongod host for replica set `%s`", p, replicaSet.Name)

			var leastBusySuitableSlave Slave
			res = tx.Raw(`SELECT s.*
			      	      FROM slave_utilization s
			      	      WHERE
			      	        s.persistent_storage = ?
			      	        AND
			      	      	s.free_mongods > 0
					AND
					s.configured_state = ?
			      	      	AND (
			      	      		s.risk_group_id NOT IN (
			      	      			SELECT DISTINCT s.risk_group_id
			      	      			FROM mongods m
			      	      			JOIN slaves s ON m.parent_slave_id = s.id
			      	      			WHERE m.replica_set_id = ?
			      	      		)
			      	      		-- 0 is the default risk group that is not a risk group,
			      	      		-- i.e from which multiple slaves can be allocated for the same replica set
			      	      		OR s.risk_group_id IS NULL
			      	      	)
					AND
					s.id NOT IN ( -- Slaves already hosting a Mongod of the Replica Set
						SELECT DISTINCT m.parent_slave_id
						FROM mongods m
						WHERE m.replica_set_id = ?
					)
			      	      ORDER BY s.utilization ASC
			      	      LIMIT 1`, p.PersistentStorage(), SlaveStateActive, replicaSet.ID, replicaSet.ID,
			).Scan(&leastBusySuitableSlave)

			if res.RecordNotFound() {
				caLog.Warn("unsatisfiable replica set `%s`: not enough suitable `%s` slaves", replicaSet.Name, p)
				unsatisfiable_replica_set_ids_by_persistance = append(unsatisfiable_replica_set_ids_by_persistance, replicaSet.ID)
				unsatisfiable_replica_set_ids = append(unsatisfiable_replica_set_ids, replicaSet.ID)
				continue
			} else if res.Error != nil {
				panic(res.Error)
			}

			caLog.Debugf("found slave `%s` as host for new mongod for replica set `%s`", leastBusySuitableSlave.Hostname, replicaSet.Name)

			m, err := c.spawnMongodOnSlave(tx, &leastBusySuitableSlave, &replicaSet.ReplicaSet)
			if err != nil {
				caLog.Errorf("could not spawn mongod on slave `%s`: %s", leastBusySuitableSlave.Hostname, err.Error())
				// the queries should have not returned a slave without free ports
				panic(err)
			} else {
				caLog.Debugf("spawned mongod `%d` for replica set `%s` on slave `%s`", m.ID, replicaSet.Name, leastBusySuitableSlave.Hostname)
			}

		}
	}

	// Send replica set constraint status messages on bus for every replica set
	if c.BusWriteChannel != nil {

		// Get replica sets and the count of their actually configured members from the database
		replicaSetsWithMemberCounts, err := tx.Raw(`SELECT
			r.*,
			(SELECT COUNT(*) FROM replica_set_configured_members WHERE replica_set_id = r.id AND persistent_storage = ?)
				AS configured_persistent_members,
			(SELECT COUNT(*) FROM replica_set_configured_members WHERE replica_set_id = r.id AND persistent_storage = ?)
				AS configured_volatile_members
		    	FROM replica_sets r
		`, true, false).Rows()
		if err != nil {
			panic(err)
		}

		for replicaSetsWithMemberCounts.Next() {
			var replicaSet ReplicaSet
			tx.ScanRows(replicaSetsWithMemberCounts, &replicaSet)

			configuredMemberCounts := struct {
				ConfiguredPersistentMembers uint
				ConfiguredVolatileMembers   uint
			}{}
			tx.ScanRows(replicaSetsWithMemberCounts, &configuredMemberCounts)

			unsatisfied := false
			//Check if replica set is in unsatisfiable list
			for _, id := range unsatisfiable_replica_set_ids {
				unsatisfied = unsatisfied || (id == replicaSet.ID)
			}

			*c.BusWriteChannel <- DesiredReplicaSetConstraintStatus{
				Unsatisfied:               unsatisfied,
				ReplicaSet:                replicaSet,
				ConfiguredPersistentCount: configuredMemberCounts.ConfiguredPersistentMembers,
				ConfiguredVolatileCount:   configuredMemberCounts.ConfiguredVolatileMembers,
			}
		}
	}

	if err == nil {
		caLog.Info("Cluster allocator done successfully")
	} else {
		caLog.WithError(err).Error("Cluster allocator done with error")
	}
	return err
}

func (c *ClusterAllocator) replicaSets(tx *gorm.DB) (replicaSets []*ReplicaSet) {

	if err := tx.Where(ReplicaSet{}).Find(&replicaSets).Error; err != nil {
		panic(err)
	}

	for _, r := range replicaSets {

		if err := tx.Model(r).Related(&r.Mongods, "Mongods").Error; err != nil {
			panic(err)
		}

		for _, m := range r.Mongods {

			res := tx.Model(m).Related(&m.ObservedState, "ObservedState")
			if err := res.Error; !res.RecordNotFound() && err != nil {
				panic(err)
			}
			res = tx.Model(m).Related(&m.DesiredState, "DesiredState")
			if err := res.Error; !res.RecordNotFound() && err != nil {
				panic(err)
			}

			//m.ParentSlave is a pointer and gorm does not initialize pointers on its own
			var parentSlave Slave
			res = tx.Model(m).Related(&parentSlave, "ParentSlave")
			if err := res.Error; err != nil {
				panic(err)
			}
			m.ParentSlave = &parentSlave

		}

	}

	return replicaSets
}

func slavePersistence(s *Slave) persistence {
	switch s.PersistentStorage {
	case true:
		return Persistent
	default:
		return Volatile
	}
}

func (p persistence) PersistentStorage() bool {
	switch p {
	case Persistent:
		return true
	case Volatile:
		return false
	default:
		panic("invalid value for persistence")
	}
}

func (p persistence) String() string {
	switch p {
	case Persistent:
		return "persistent"
	case Volatile:
		return "volatile"
	default:
		panic("invalid value for persistence")
	}
}

func (c *ClusterAllocator) spawnMongodOnSlave(tx *gorm.DB, s *Slave, r *ReplicaSet) (*Mongod, error) {

	var usedPorts []PortNumber
	res := tx.Raw(`
		SELECT m.port
		FROM mongods m
		WHERE m.parent_slave_id = ?
		ORDER BY m.port ASC
	`, s.ID).Pluck("port", &usedPorts)

	if !res.RecordNotFound() && res.Error != nil {
		panic(res.Error)
	}

	caLog.Debugf("slave: %#v: found used ports: %v", s, usedPorts)

	unusedPort, found := findUnusedPort(usedPorts, s.MongodPortRangeBegin, s.MongodPortRangeEnd)

	if !found {
		return nil, fmt.Errorf("could not spawn Mongod: no free port on slave `%s`", s.Hostname)
	}

	m := &Mongod{
		Port:          unusedPort,
		ReplSetName:   r.Name,
		ParentSlaveID: s.ID,
		ReplicaSetID:  r.ID,
	}
	if err := tx.Create(&m).Error; err != nil {
		panic(err)
	}

	desiredState := MongodState{
		ParentMongodID:         m.ID,
		IsShardingConfigServer: r.ConfigureAsShardingConfigServer,
		ExecutionState:         MongodExecutionStateRunning,
	}
	if err := tx.Create(&desiredState).Error; err != nil {
		panic(err)
	}

	if err := tx.Model(&m).Update("DesiredStateID", desiredState.ID).Error; err != nil {
		panic(err)
	}

	return m, nil

}

// find free port using merge-join-like loop. results are in [minPort, maxPort)
// assuming usedPorts is sorted ascending
func findUnusedPort(usedPorts []PortNumber, minPort, maxPort PortNumber) (unusedPort PortNumber, found bool) {

	usedPortIndex := 0

	// make usedPortIndex satisfy invariant
	for ; usedPortIndex < len(usedPorts) && !(usedPorts[usedPortIndex] >= minPort); usedPortIndex++ {
	}

	for currentPort := minPort; currentPort < maxPort; currentPort++ {
		if usedPortIndex >= len(usedPorts) { // we passed all used ports
			return currentPort, true
		}

		if usedPorts[usedPortIndex] == currentPort { // current port is used
			usedPortIndex++
		} else if usedPorts[usedPortIndex] > currentPort { // next used port is after current port
			return currentPort, true
		}
		// invariant: usedPorts[usedPortIndex] >= currentPort || usedPortIndex >= len(usedPorts)
		// 							i.e. no more used ports to check for
	}
	return 0, false
}

func slaveMaxNumberOfMongods(s *Slave) PortNumber {
	res := s.MongodPortRangeEnd - s.MongodPortRangeBegin
	if res <= 0 {
		panic("datastructure invariant violated: the range of Mongod ports for a slave must be sized greater than 0")
	}
	return res
}

func slaveUsage(s *Slave) (runningMongods, maxMongods uint) {
	return uint(len(s.Mongods)), uint(slaveMaxNumberOfMongods(s))
}

func slaveBusyRate(s *Slave) float64 {
	runningMongods, maxMongods := slaveUsage(s)
	return float64(runningMongods) / float64(maxMongods)
}
