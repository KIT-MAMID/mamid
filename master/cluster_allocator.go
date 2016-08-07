package master

import (
	"fmt"
	. "github.com/KIT-MAMID/mamid/model"
	"github.com/jinzhu/gorm"
)

type ClusterAllocator struct {
}

type persistence uint

const (
	Persistent persistence = 0
	Volatile   persistence = 1
)

type memberCountTuple map[persistence]uint

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

	replicaSets := c.replicaSets(tx)
	for _, r := range replicaSets {
		c.removeUnneededMembers(tx, r)
	}

	c.addMembers(tx, replicaSets)

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

func (c *ClusterAllocator) removeUnneededMembers(tx *gorm.DB, r *ReplicaSet) {
	for persistence, count := range c.effectiveMemberCount(tx, r) {
		c.removeUnneededMembersByPersistence(tx, r, persistence, count)
	}
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

func (c *ClusterAllocator) removeUnneededMembersByPersistence(tx *gorm.DB, r *ReplicaSet, p persistence, initialCount uint) {

	/*
		-- effective member count (before deleting anything)
		SELECT r.replica_set_id, COUNT(DISTINCT m.ID)
		FROM replica_sets r
		JOIN mongods m ON m.replica_set_id = r.id
		JOIN slaves s ON s.id = m.parent_slave
		JOIN slave_states observed ON observed.id = m.observed_state_id
		JOIN slave_states desired ON desired.id = m.desired_state_id
		WHERE
			s.persistent_storage = ?p
			AND
			observed.execution_state = ?running
			AND
			desired.execution_state = ?running
		GROUP BY r.id;

		-- view: slave_utilization
		CREATE VIEW slave_utilization AS
		SELECT
			*,
			CASE max_mongods = 0 THEN 1 ELSE current_mongods/max_mongods END AS utilization,
			max_mongods - current_mongods AS free_mongods
		FROM (
			SELECT
				s.*,
				s.mongod_port_range_end - s.mongod_port_range_begin AS max_mongods,
				COUNT(DISTINCT m.id) as current_mongods
			FROM slaves s
			LEFT OUTER JOIN mongods m ON m.replica_set_id = r.id
		)

		-- HEAD of prioritized list of deletable members
		-- 	parametrized by: configured_state
		SELECT m.*
		FROM replica_sets r
		JOIN mongods m ON m.replica_set_id = r.id
		JOIN slaves s ON s.id = m.parent_slave
		JOIN slave_utilization su ON s.id = u.slave_id
		WHERE r.id = ?r.ID AND s.persistent_storage = ?p AND s.configured_state = ?configured_state
		ORDER BY DESC su.utilization -- TODO how is it ordered? we want =disabled first. Do it in 2 queries (prepared statements?)
		LIMIT 1

	*/

	var configuredMemberCount uint
	if p == Persistent {
		configuredMemberCount = r.PersistentMemberCount
	} else if p == Volatile {
		configuredMemberCount = r.VolatileMemberCount
	}

	// Destroy any Mongod running on disabled slaves (no specific priority)
	for initialCount > configuredMemberCount {
		for _, m := range r.Mongods {

			if m.ParentSlave.ConfiguredState == SlaveStateDisabled &&
				slavePersistence(m.ParentSlave) == p {

				c.destroyMongod(tx, m)
				initialCount--
			}
		}
	}

	// Remove superfluous Mongods on busiest slaves first
	removalPQ := c.pqMongods(r.Mongods, p)
	for initialCount > configuredMemberCount {
		// Destroy any Mongod (lower priority)
		m := removalPQ.PopMongodOnBusiestSlave()

		if m == nil {
			break
		}

		// destroy
		c.destroyMongod(tx, m)
		initialCount--

	}

}

func (c *ClusterAllocator) destroyMongod(tx *gorm.DB, m *Mongod) {

	// Set the desired execution state to disabled

	m.DesiredState.ExecutionState = MongodExecutionStateDestroyed
	if err := tx.Model(&m.DesiredState).Update("execution_state", MongodExecutionStateDestroyed); err != nil {
		panic(err)
	}

	// TODO MongodMatchStatus

}

func (c *ClusterAllocator) effectiveMemberCount(tx *gorm.DB, r *ReplicaSet) memberCountTuple {

	var res memberCountTuple

	for _, m := range r.Mongods {

		if m.ObservedState.ExecutionState == MongodExecutionStateRunning &&
			m.DesiredState.ExecutionState == MongodExecutionStateRunning {
			if m.ParentSlave.PersistentStorage {
				res[Persistent]++
			} else {
				res[Volatile]++
			}
		}
	}

	return res
}

func (c *ClusterAllocator) addMembersForPersistence(tx *gorm.DB, p persistence) {

	/*

		-- view: replica set members
		CREATE VIEW replica_set_configured_members AS
		SELECT r.id as replica_set_id, m.id as mongod_id, s.persistent_storage
		FROM replica_set r
		JOIN mongods m ON m.replica_set_id = r.id
		JOIN mongod_states desired_state ON m.desired_state_id = desired_state.id
		JOIN slaves s ON m.parent_slave_id = s.id
		WHERE
			s.configured_state != ?disabled
			AND
			desired_state.execution_state NOT IN (?NotRunning, ?Destroyed)

		-- HEAD of degraded replica sets PQ
		SELECT r.*, COUNT(DISTINCT members.mongod_id) as "configured_member_count"
		FROM replica_sets r
		LEFT OUTER JOIN replica_set_configured_members members ON r.id = members.replica_set_id
		WHERE r.persistent_storage = ?p AND ?r.Peristent|VolatileMemberCount != 0
		GROUP BY r.id
		ORDER BY COUNT(members.mongod_id) / r.Persistent|VolatileMemberCount
		LIMIT 1

		-- parameters: replica_set_id
		SELECT s.*
		FROM slave_utilization s
		WHERE
			s.free_mongods > 0
			AND (
				s.risk_group_id NOT IN (
					SELECT DISTINCT s.risk_group_id
					FROM mongods m
					JOIN slaves s ON m.parent_slave_id = s.id
					WHERE m.replica_set_id = ?replica_set_id
				)
				-- 0 is the default risk group that is not a risk group,
				-- i.e from which multiple slaves can be allocated for the same replica set
				OR s.risk_group_id = 0
			)
		ORDER BY s.utilization ASC
		LIMIT 1

	*/

}

func (c *ClusterAllocator) addMembers(tx *gorm.DB, replicaSets []*ReplicaSet) {

	for _, persistence := range []persistence{Volatile, Persistent} {
		c.addMembersForPersistence(tx, persistence)
	}

	// TODO remove this code once SQL works
	for _, persistence := range []persistence{Volatile, Persistent} {

		// build prioritization datastructures
		// will only return items that match current persistence and actually need more members

		pqReplicaSets := c.pqReplicaSets(replicaSets, persistence)

		for r := pqReplicaSets.Pop(); r != nil; {

			pqRiskGroups := c.pqRiskGroups(tx, r, persistence)

			if s := pqRiskGroups.PopSlaveInNonconflictingRiskGroup(); s != nil {

				// spawn new Mongod m on s and add it to r.Mongods
				// compute MongodState for m and set the DesiredState variable
				_ = c.spawnMongodOnSlave(tx, s, r)
				// TODO send DesiredReplicaSetConstraintStatus

				pqReplicaSets.PushIfDegraded(r)

			} else {

				// TODO send DesiredReplicaSetConstraintStatus
				panic("not implemented")

			}
		}

	}
}

func (c *ClusterAllocator) spawnMongodOnSlave(tx *gorm.DB, s *Slave, r *ReplicaSet) *Mongod {

	// Get a port number, validates expected invariant that there's a free port as a side effect
	portNumber, err := c.slaveNextMongodPort(tx, s)
	if err != nil {
		panic(err)
	}

	m := &Mongod{
		Port:        portNumber,
		ReplSetName: r.Name,
		ParentSlave: s,
		ReplicaSet:  r,
		DesiredState: MongodState{ // TODO verify this nested initialization works with gorm
			IsShardingConfigServer: r.ConfigureAsShardingConfigServer,
			ExecutionState:         MongodExecutionStateRunning,
		},
	}

	if err := tx.Create(&m).Error; err != nil {
		panic(err)
	}

	return m

}

func (c *ClusterAllocator) slaveNextMongodPort(tx *gorm.DB, s *Slave) (portNumber PortNumber, err error) {

	var mongods []*Mongod

	if err = tx.Model(s).Related(&mongods).Error; err != nil {
		return PortNumber(0), err
	}

	maxMongodCount := slaveMaxNumberOfMongods(s)
	if len(mongods) >= int(maxMongodCount) {
		return PortNumber(0), fmt.Errorf("slave '%s' is full or is running more than maximum of '%d' Mongods", s.Hostname, maxMongodCount)
	}

	if len(mongods) <= 0 {
		return s.MongodPortRangeBegin, nil
	}

	portsUsed := make([]bool, maxMongodCount)
	for _, m := range mongods {
		portsUsed[m.Port-s.MongodPortRangeBegin] = true
	}
	for i := PortNumber(0); i < maxMongodCount; i++ {
		if !portsUsed[i] {
			return s.MongodPortRangeBegin + i, nil
		}
	}

	panic("algorithm invariant violated: this code should not be reached")
	return PortNumber(0), nil
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
