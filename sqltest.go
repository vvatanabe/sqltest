package sqltest

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
)

func init() {
	// Since this file is used only in _test.go, suppress logging only during tests.
	mysql.SetLogger(log.New(io.Discard, "", 0))
}

// RunOption is a function that modifies a dockertest.RunOptions.
type RunOption func(*dockertest.RunOptions)

// NewDockerDB starts a Docker container using the specified run options,
// container port, driver name, and a function to generate the DSN.
// It returns a connected *sql.DB and a cleanup function.
func NewDockerDB(t testing.TB, runOpts *dockertest.RunOptions, containerPort, driverName string, dsnFunc func(actualPort string) string) (*sql.DB, func()) {
	t.Helper()

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("failed to connect to docker: %s", err)
	}

	resource, err := pool.RunWithOptions(runOpts)
	if err != nil {
		t.Fatalf("failed to start %s container: %s", driverName, err)
	}

	actualPort := resource.GetHostPort(containerPort)
	if actualPort == "" {
		_ = pool.Purge(resource)
		t.Fatalf("no host port was assigned for the %s container", driverName)
	}
	t.Logf("%s container is running on host port '%s'", driverName, actualPort)

	var db *sql.DB
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err = pool.Retry(func() error {
		dsn := dsnFunc(actualPort)
		db, err = sql.Open(driverName, dsn)
		if err != nil {
			return err
		}
		return db.PingContext(ctx)
	}); err != nil {
		_ = pool.Purge(resource)
		t.Fatalf("failed to connect to %s: %s", driverName, err)
	}

	cleanup := func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close DB: %s", err)
		}
		if err := pool.Purge(resource); err != nil {
			t.Logf("failed to remove %s container: %s", driverName, err)
		}
	}

	return db, cleanup
}

// NewMySQL starts a MySQL container using Docker and returns a connected *sql.DB,
// along with a cleanup function to remove the container after tests.
// The 'tag' parameter specifies the MySQL version to use (e.g., "8.0").
// Additional RunOption functions can be provided to override default settings.
func NewMySQL(t testing.TB, tag string, opts ...RunOption) (*sql.DB, func()) {
	// Set default run options for MySQL.
	runOpts := &dockertest.RunOptions{
		Repository: "mysql",
		Tag:        tag,
		Env: []string{
			"MYSQL_ROOT_PASSWORD=secret",
			"MYSQL_DATABASE=test",
		},
	}

	// Apply any provided RunOption functions to override defaults.
	for _, opt := range opts {
		opt(runOpts)
	}

	return NewDockerDB(t, runOpts, "3306/tcp", "mysql", func(actualPort string) string {
		return fmt.Sprintf("root:secret@tcp(%s)/test?parseTime=true", actualPort)
	})
}

// NewPostgres starts a PostgreSQL container using Docker and returns a connected *sql.DB,
// along with a cleanup function to remove the container after tests.
// The 'tag' parameter specifies the PostgreSQL version to use (e.g., "13").
// Additional RunOption functions can be provided to override default settings.
func NewPostgres(t testing.TB, tag string, opts ...RunOption) (*sql.DB, func()) {
	// Set default run options for PostgreSQL.
	runOpts := &dockertest.RunOptions{
		Repository: "postgres",
		Tag:        tag,
		Env: []string{
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_DB=test",
		},
	}

	// Apply any provided RunOption functions to override defaults.
	for _, opt := range opts {
		opt(runOpts)
	}

	return NewDockerDB(t, runOpts, "5432/tcp", "postgres", func(actualPort string) string {
		return fmt.Sprintf("postgres://postgres:secret@%s/test?sslmode=disable", actualPort)
	})
}

// InitialDBSetup is used to set up the database before tests.
// SchemaSQL contains DDL statements for creating tables or indexes,
// and InitialData contains SQL statements to insert initial data.
type InitialDBSetup struct {
	// SchemaSQL contains DDL statements (e.g., table or index creation).
	SchemaSQL string
	// InitialData contains SQL statements for seeding initial data.
	InitialData []string
}

// PrepDatabase executes the provided schema and initial data SQL statements sequentially
// to prepare the test database. It returns an error if any step fails.
func PrepDatabase(t testing.TB, db *sql.DB, setups ...InitialDBSetup) error {
	t.Helper()

	for _, setup := range setups {
		if setup.SchemaSQL != "" {
			if _, err := db.Exec(setup.SchemaSQL); err != nil {
				return fmt.Errorf("failed to execute schema SQL: %w", err)
			}
		}
		// Execute the initial data insertion (DML) within a transaction.
		if len(setup.InitialData) > 0 {
			tx, err := db.Begin()
			if err != nil {
				return fmt.Errorf("failed to begin transaction: %w", err)
			}
			for _, stmt := range setup.InitialData {
				if _, err := tx.Exec(stmt); err != nil {
					_ = tx.Rollback()
					return fmt.Errorf("failed to execute initial data SQL: %w", err)
				}
			}
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit transaction: %w", err)
			}
		}
	}
	return nil
}
