START TRANSACTION;

CREATE DOMAIN sharding_role AS VARCHAR(255) CHECK( value IN ('none', 'shardsvr', 'configsvr'));

CREATE TABLE "risk_groups" (
	"id" BIGSERIAL PRIMARY KEY,
	"name" VARCHAR(255) UNIQUE
);

-- CREATE UNIQUE INDEX uix_risk_groups_name ON "risk_groups"("name");

CREATE TABLE "replica_sets" (
	"id" BIGSERIAL PRIMARY KEY,
	"name" VARCHAR(255) UNIQUE,
	"persistent_member_count" INTEGER,
	"volatile_member_count" INTEGER,
	"sharding_role" sharding_role NOT NULL,
	"initiated" BOOLEAN NOT NULL
);

-- CREATE UNIQUE INDEX uix_replica_sets_name ON "replica_sets"("name");

CREATE TABLE "msp_errors" (
	"id" BIGSERIAL PRIMARY KEY,
	"identifier" VARCHAR(255),
	"description" TEXT,
	"long_description" TEXT
);

CREATE TABLE slaves (
	"id" BIGSERIAL PRIMARY KEY,
	"hostname" VARCHAR(255) UNIQUE,
	"port" INTEGER,
	"mongod_port_range_begin" INTEGER,
	"mongod_port_range_end" INTEGER,
	"persistent_storage" bool,
	"configured_state" INTEGER,
	"risk_group_id" BIGINT NULL REFERENCES risk_groups(id) DEFERRABLE INITIALLY DEFERRED,
	"observation_error_id" BIGINT NULL REFERENCES msp_errors(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED
);

-- CREATE UNIQUE INDEX uix_slaves_hostname ON "slaves"("hostname")

CREATE TABLE "mongod_states" (
	"id" BIGSERIAL PRIMARY KEY,
	"parent_mongod_id" BIGINT NOT NULL, -- foreign key constraint added below
	"sharding_role" sharding_role,
	"execution_state" INTEGER
);

CREATE TABLE "mongods" (
	"id" BIGSERIAL PRIMARY KEY,
	"port" INTEGER,
	"repl_set_name" VARCHAR(255),
	"observation_error_id" BIGINT NULL REFERENCES msp_errors(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED, -- error encountered when observing this specific Mongod
	"last_establish_state_error_id" BIGINT NULL REFERENCES msp_errors(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED,
	"parent_slave_id" BIGINT REFERENCES slaves(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
	"replica_set_id" BIGINT NULL REFERENCES replica_sets(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED,
	"desired_state_id" BIGINT NOT NULL REFERENCES mongod_states(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
	"observed_state_id" BIGINT NULL REFERENCES mongod_states(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED
    -- +-------------------+-----------------------------+---------------------+
    -- |                   | obs_state = NULL            | obs_state != NULL   |
    -- +-------------------+-----------------------------+---------------------+
    -- | obs_error = NULL  | Never attempted to observe  | Valid observation   |
    -- +-------------------+-----------------------------+---------------------+
    -- | obs_error != NULL | Never successfully observed | Invalid observation |
    -- +-------------------+-----------------------------+---------------------+

);

ALTER TABLE mongod_states ADD CONSTRAINT constr_parent_mongod FOREIGN KEY (parent_mongod_id) REFERENCES mongods(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE "problems" (
	"id" BIGSERIAL PRIMARY KEY,
	"description" TEXT,
	"long_description" TEXT,
	"problem_type" INTEGER,
	"first_occurred" TIMESTAMP,
	"last_updated" TIMESTAMP,
	"slave_id" BIGINT NULL REFERENCES slaves(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
	"replica_set_id" BIGINT NULL REFERENCES replica_sets(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
	"mongod_id" BIGINT NULL REFERENCES mongods(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED
);

CREATE OR REPLACE VIEW replica_set_effective_members AS
	SELECT r.id as replica_set_id, m.id as mongod_id, s.persistent_storage
	FROM replica_sets r
	JOIN mongods m ON m.replica_set_id = r.id
	JOIN slaves s ON s.id = m.parent_slave_id
	JOIN mongod_states observed ON observed.id = m.observed_state_id
	JOIN mongod_states desired ON desired.id = m.desired_state_id
	WHERE
	observed.execution_state = 6 -- running
	AND
	desired.execution_state = 6; -- running

CREATE OR REPLACE VIEW replica_set_effective_members_for_monitoring AS
	SELECT r.id as replica_set_id, m.id as mongod_id, s.persistent_storage
	FROM replica_sets r
	JOIN mongods m ON m.replica_set_id = r.id
	JOIN slaves s ON s.id = m.parent_slave_id
	JOIN mongod_states observed ON observed.id = m.observed_state_id
	JOIN mongod_states desired ON desired.id = m.desired_state_id
	WHERE
	observed.execution_state = 6 AND s.observation_error_id IS NULL -- running
	AND
	desired.execution_state = 6; -- running


CREATE OR REPLACE VIEW slave_utilization AS
	SELECT
		subquery.*,
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
		) subquery;

CREATE OR REPLACE VIEW replica_set_configured_members AS
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
		3, -- not running
		2, -- destroyed
		1 -- force destroyed
	);

CREATE TABLE "mamid_metadata" (
	"key" VARCHAR(255) PRIMARY KEY,
	"value" TEXT
);

-- Mongod "keyfiles" for "Internal Authentication" between Mongods in a Replica Set
CREATE TABLE "mongod_keyfiles" (
	"id" BIGSERIAL PRIMARY KEY,
	"content" TEXT NOT NULL
);

-- MongoDB credentials with "root" role used by MAMID to configure the cluster
CREATE TABLE "mongodb_root_credentials" (
	"id" BIGSERIAL PRIMARY KEY,
	"username" VARCHAR(255) NOT NULL,
	"password" VARCHAR(255) NOT NULL
);

COMMIT;
