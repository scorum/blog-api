package utils

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rubenv/sql-migrate"
	"gopkg.in/ory-am/dockertest.v3"
)

func DockertestMain(m *testing.M, up func(dbWrite, dbRead *sqlx.DB)) {
	var (
		dbWrite, dbRead *sqlx.DB
		err             error
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
		writeConnectionString := fmt.Sprintf("user=postgres dbname=postgres sslmode=disable port=%s", resource.GetPort("5432/tcp"))
		dbWrite, err = sqlx.Open("postgres", writeConnectionString)
		if err != nil {
			return err
		}
		if err := dbWrite.Ping(); err != nil {
			return err
		}
		if err = createReadonlyUser(dbWrite); err != nil {
			return err
		}
		readConnectionString := fmt.Sprintf("user=readonly dbname=postgres sslmode=disable port=%s", resource.GetPort("5432/tcp"))
		dbRead, err = sqlx.Open("postgres", readConnectionString)
		return dbRead.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// migrate sql
	migrations := &migrate.FileMigrationSource{
		Dir: os.Getenv("GOPATH") + "/src/gitlab.scorum.com/blog/api/db/migrations",
	}
	n, err := migrate.Exec(dbWrite.DB, "postgres", migrations, migrate.Up)
	if err != nil {
		log.Fatalf("Could not migrage sql: %s", err)
	}
	fmt.Printf("Applied %d migrations!\n", n)
	_, err = dbWrite.Exec(`GRANT SELECT ON ALL TABLES IN SCHEMA public TO readonly;`)
	if err != nil {
		log.Fatalf("Could not grant access to readonly user")
	}

	up(dbWrite, dbRead)

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func createReadonlyUser(db *sqlx.DB) error {
	_, err := db.Exec(`CREATE USER readonly PASSWORD '';
				   GRANT CONNECT ON DATABASE postgres TO readonly;
				   GRANT USAGE ON SCHEMA public TO readonly;
	`)
	return err
}
