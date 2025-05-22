package postgres

import (
	"github.com/jmoiron/sqlx"
	"testing"
	"log"
	"fmt"
	"os"
	"gopkg.in/ory-am/dockertest.v3"
	"github.com/stretchr/testify/require"
)

var pqdb *sqlx.DB

func cleanUp(t *testing.T) {
	_, err := pqdb.Exec("DELETE FROM enum_test")
	require.NoError(t, err)
	_, err = pqdb.Exec("DELETE FROM test")
	require.NoError(t, err)
	_, err = pqdb.Exec("DELETE FROM foreign_test")
	require.NoError(t, err)
}

// TestMain runs a docker container with Postgres and applies db migration
func TestMain(m *testing.M) {
	var (
		err              error
		connectionString string
	)

	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker pool: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.Run("mdillon/postgis", "9.6-alpine", []string{})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		connectionString = fmt.Sprintf("user=postgres dbname=postgres sslmode=disable port=%s", resource.GetPort("5432/tcp"))
		pqdb, err = sqlx.Open("postgres", connectionString)
		if err != nil {
			return err
		}
		return pqdb.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	if _, err = pqdb.Exec(`CREATE TABLE test(id INT UNIQUE)`); err != nil {
		log.Fatalf("Failed to create table: %s", err)
	}

	if _, err = pqdb.Exec(`CREATE TABLE foreign_test(id INT REFERENCES test(id))`); err != nil {
		log.Fatalf("Failed to create table: %s", err)
	}

	if _, err = pqdb.Exec(`CREATE TYPE custom_enum AS ENUM('val');
								 CREATE TABLE enum_test(test custom_enum);`); err != nil {
		log.Fatalf("Failed to create enum and table: %s", err)
	}

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func TestIsUniqueError(t *testing.T) {
	defer cleanUp(t)

	_, err := pqdb.Exec(`INSERT INTO test VALUES(1)`)
	require.NoError(t, err)

	_, err = pqdb.Exec(`INSERT INTO test VALUES(1)`)
	require.Error(t, err)
	isUniqueErr, constraint := IsUniqueError(err)
	require.True(t, isUniqueErr)
	require.Equal(t, constraint, "test_id_key")
}

func TestIsForeignKeyViolationError(t *testing.T) {
	defer cleanUp(t)

	_, err := pqdb.Exec(`INSERT INTO foreign_test VALUES(1)`)
	require.Error(t, err)
	isForeignKeyViolationErr, constraint := IsForeignKeyViolationError(err)
	require.True(t, isForeignKeyViolationErr)
	require.Equal(t, constraint, "foreign_test_id_fkey")
}

func TestIsInvalidTextRepresentation(t *testing.T) {
	defer cleanUp(t)

	_, err := pqdb.Exec(`INSERT INTO enum_test VALUES('1')`)
	require.Error(t, err)
	require.True(t, IsInvalidTextRepresentation(err))

	_, err = pqdb.Exec(`INSERT INTO enum_test VALUES('val')`)
	require.NoError(t, err)
}
