CREATE TABLE slaves (
	"id" integer primary key autoincrement,
	"hostname" varchar(255),
	"port" integer,
	"mongod_port_range_begin" integer,
	"mongod_port_range_end" integer,
	"persistent_storage" bool,
	"configured_state" integer,
	"risk_group_id" integer NULL REFERENCES risk_groups(id) DEFERRABLE INITIALLY DEFERRED,
	"observation_error_id" integer NULL REFERENCES msp_errors(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED
);

-- CREATE UNIQUE INDEX uix_slaves_hostname ON "slaves"("hostname")

CREATE TABLE "replica_sets" (
	"id" integer primary key autoincrement,
	"name" varchar(255),
	"persistent_member_count" integer,
	"volatile_member_count" integer,
	"configure_as_sharding_config_server" bool
);

-- CREATE UNIQUE INDEX uix_replica_sets_name ON "replica_sets"("name");

CREATE TABLE "risk_groups" (
	"id" integer primary key autoincrement,
	"name" varchar(255)
);

-- CREATE UNIQUE INDEX uix_risk_groups_name ON "risk_groups"("name");

CREATE TABLE "mongods" (
	"id" integer primary key autoincrement,
	"port" integer,
	"repl_set_name" varchar(255),
	"observation_error_id" integer NULL REFERENCES msp_errors(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED,
	"last_establish_state_error_id" integer NULL REFERENCES msp_errors(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED,
	"parent_slave_id" integer REFERENCES slaves(id) DEFERRABLE INITIALLY DEFERRED,
	"replica_set_id" integer NULL REFERENCES replica_sets(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED,
	"desired_state_id" integer NOT NULL REFERENCES mongod_states(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
	"observed_state_id" integer NULL REFERENCES mongod_states(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE "mongod_states" (
	"id" integer primary key autoincrement,
	"parent_mongod_id" integer NOT NULL REFERENCES mongods(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
	"is_sharding_config_server" bool,
	"execution_state" integer
);

CREATE TABLE "replica_set_members" (
	"id" integer primary key autoincrement,
	"hostname" varchar(255),
	"port" integer,
	"mongod_state_id" integer REFERENCES mongod_states(id) DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE "problems" (
	"id" integer primary key autoincrement,
	"description" varchar(255),
	"long_description" varchar(255),
	"problem_type" integer,
	"first_occurred" datetime,
	"last_updated" datetime,
	"slave_id" integer NULL REFERENCES slaves(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
	"replica_set_id" integer NULL REFERENCES replica_sets(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
	"mongod_id" integer NULL REFERENCES mongods(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE "msp_errors" (
	"id" integer primary key autoincrement,
	"identifier" varchar(255),
	"description" varchar(255),
	"long_description" varchar(255)
);

DROP VIEW IF EXISTS replica_set_effective_members;
CREATE VIEW replica_set_effective_members AS
	SELECT r.id as replica_set_id, m.id as mongod_id, s.persistent_storage
	FROM replica_sets r
	JOIN mongods m ON m.replica_set_id = r.id
	JOIN slaves s ON s.id = m.parent_slave_id
	JOIN mongod_states observed ON observed.id = m.observed_state_id
	JOIN mongod_states desired ON desired.id = m.desired_state_id
	WHERE
	observed.execution_state = 4 -- running
	AND
	desired.execution_state = 4; -- running


DROP VIEW IF EXISTS slave_utilization;
CREATE VIEW slave_utilization AS
	SELECT
		*,
		CASE WHEN max_mongods = 0 THEN 1 ELSE current_mongods*1.0/max_mongods END AS utilization,
		(max_mongods - current_mongods) AS free_mongods
	FROM
		(
			SELECT
				s.*,
				s.mongod_port_range_end - s.mongod_port_range_begin AS max_mongods,
				COUNT(DISTINCT m.id) as current_mongods
			FROM slaves s
			LEFT OUTER JOIN mongods m ON m.parent_slave_id = s.id
			GROUP BY s.id
		);

DROP VIEW IF EXISTS replica_set_configured_members;
CREATE VIEW replica_set_configured_members AS
	SELECT
		r.id as replica_set_id,
		m.id as mongod_id,
		s.persistent_storage
	FROM replica_sets r
	JOIN mongods m ON m.replica_set_id = r.id
	JOIN mongod_states desired_state ON m.desired_state_id = desired_state.id
	JOIN slaves s ON m.parent_slave_id = s.id
	WHERE
		s.configured_state != 3 -- disabled
		AND
		desired_state.execution_state NOT IN (
		2 -- not running
		,1 -- destroyed
	);

