package master

import (
	"database/sql"
	"fmt"
	. "github.com/KIT-MAMID/mamid/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

/*
Testing the cluster allocator:

Mocking: Bus (Mismatch Messages, Constraint Status)

Fixtures:
 * Flexible infrastructure for creating test scenarios
 * Elegant way to compare pre- and post-state of the database?

What to test?

count methods

priority queue builders

idempotence: test after...
  run of removal
  run of add
  => cancels out every run of the entire algorithm

completeness of the object graph? do we fetch it at the beginning? what about locking?

mismatch generation => use mock of Bus?

*/

// Create a scenario from a running setup:
//  pg_dump --insert -Oa <additional, optional connection parameter> <dbname>

func TestClutserAllocator_findUnusedPort(t *testing.T) {

	unusedPort, found := findUnusedPort([]PortNumber{2, 3, 5}, 2, 6)
	assert.EqualValues(t, 4, unusedPort, "should find lowest free port number")

	unusedPort, found = findUnusedPort([]PortNumber{}, 2, 5)
	assert.EqualValues(t, 2, unusedPort, "should use minPort when no port used")

	unusedPort, found = findUnusedPort([]PortNumber{0}, 2, 5)
	assert.EqualValues(t, 2, unusedPort)

	var uninitialized []PortNumber
	unusedPort, found = findUnusedPort(uninitialized, 2, 5)
	assert.EqualValues(t, 2, unusedPort)

	unusedPort, found = findUnusedPort([]PortNumber{2, 3, 4}, 2, 5)
	assert.Equal(t, false, found, "should not find a port if no port free")

}

func saveDB(dsn string, driver string) (dump map[string][]string, err error) {
	dump = make(map[string][]string)
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return
	}
	defer db.Close()
	tables, err := db.Query("select tablename from pg_tables WHERE schemaname='public' ORDER by tablename")
	//tables, err := db.Query("select repl_set_name from mongods")

	defer tables.Close()
	if err != nil {
		return
	}
	for tables.Next() {
		var table string
		err = tables.Scan(&table)
		if err != nil {
			return dump, err
		}
		res, err := db.Query(fmt.Sprintf(`SELECT
		pg_attribute.attname
		FROM pg_index, pg_class, pg_attribute, pg_namespace
		WHERE
		pg_class.oid='%s'::regclass AND
		indrelid = pg_class.oid AND
		nspname = 'public' AND
		pg_class.relnamespace = pg_namespace.oid AND
		pg_attribute.attrelid = pg_class.oid AND
		pg_attribute.attnum = any(pg_index.indkey)
		AND indisprimary;`, table))
		if err != nil {
			return dump, err
		}
		var key string
		res.Next()
		res.Scan(&key)
		res.Close()
		tableContents, tableContentErr := db.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY %s", table, key))
		if tableContentErr != nil {
			return dump, tableContentErr
		}
		cols, colGetError := tableContents.Columns()
		if colGetError != nil {
			return dump, colGetError
		}
		dest := make([]interface{}, len(cols))
		rawResult := make([][]byte, len(cols))
		result := make([]string, len(cols))
		for i, _ := range rawResult {
			dest[i] = &rawResult[i] // Put pointers to each string in the interface slice
		}
		for tableContents.Next() {
			tableGetContentsErr := tableContents.Scan(dest...)
			if tableGetContentsErr != nil {
				return dump, err
			}
			for i, raw := range rawResult {
				if raw == nil {
					result[i] = "\\N"
				} else {
					result[i] = string(raw)
				}
			}
		}
		tableContents.Close()
		dump[table] = result
	}
	return
}

func TestTestSaveDB(t *testing.T) {
	// Check equality
	db, dsn, err := InitializeTestDBFromFile("cluster_allocator_test_fixture_allocate_full.sql")
	assert.NoError(t, err)
	dump, err := saveDB(dsn, db.Driver)
	assert.NoError(t, err)
	db, dsn, err = InitializeTestDBFromFile("cluster_allocator_test_fixture_allocate_full.sql")
	assert.NoError(t, err)
	dump2, err := saveDB(dsn, db.Driver)
	assert.NoError(t, err)
	assert.Equal(t, dump, dump2)

	//Check non-equality
	db, dsn, err = InitializeTestDBFromFile("cluster_allocator_test_fixture_allocated_degraded.sql")
	assert.NoError(t, err)
	dump2, err = saveDB(dsn, db.Driver)
	assert.NoError(t, err)
	assert.NotEqual(t, dump, dump2)
}

func TestClusterAllocator_CompileMongodLayout_Idempotence_Simple(t *testing.T) {
	db, dsn, err := InitializeTestDBFromFile("cluster_allocator_test_fixture_allocate_full.sql")
	assert.NoError(t, err)
	dump, err := saveDB(dsn, db.Driver)
	assert.NoError(t, err)
	var alloc ClusterAllocator
	tx := db.Begin()
	alloc.CompileMongodLayout(tx)
	dump2, err := saveDB(dsn, db.Driver)
	assert.NoError(t, err)
	assert.Equal(t, dump, dump2)
	alloc.CompileMongodLayout(tx)
	dump2, err = saveDB(dsn, db.Driver)
	assert.NoError(t, err)
	assert.Equal(t, dump, dump2)
}
