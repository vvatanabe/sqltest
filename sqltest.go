package sqltest

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"io"
	"log"
	"testing"
	"time"
)

func init() {
	// Since this file is used only in _test.go, suppress logging only during tests.
	mysql.SetLogger(log.New(io.Discard, "", 0))
}

// RunOption is a function that modifies a dockertest.RunOptions.
type RunOption func(*dockertest.RunOptions)

// NewDockerDB starts a Docker container using the specified run options,
// container port, driver name, and a function to generate the DSN.
// Additionally, it accepts optional host configuration functions.
// It returns a connected *sql.DB and a cleanup function.
func NewDockerDB(t testing.TB, runOpts *dockertest.RunOptions, containerPort, driverName string, dsnFunc func(actualPort string) string, hostOpts ...func(*docker.HostConfig)) (*sql.DB, func()) {
	t.Helper()

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("failed to connect to docker: %s", err)
	}

	// Pass optional host configuration options.
	resource, err := pool.RunWithOptions(runOpts, hostOpts...)
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

// NewMySQL starts a MySQL Docker container using the default settings and returns a connected *sql.DB
// along with a cleanup function. It uses the default MySQL image ("mysql") with tag "8.0". For more
// customization, use NewMySQLWithOptions.
func NewMySQL(t testing.TB) (*sql.DB, func()) {
	return NewMySQLWithOptions(t, nil)
}

const (
	defaultMySQLImage = "mysql"
	defaultMySQLTag   = "8.0"
)

// NewMySQLWithOptions starts a MySQL Docker container using Docker and returns a connected *sql.DB
// along with a cleanup function. It applies the default settings:
//   - Repository: "mysql"
//   - Tag: "8.0"
//   - Environment: MYSQL_ROOT_PASSWORD=secret, MYSQL_DATABASE=test
//
// Additional RunOption functions can be provided via the runOpts parameter to override these defaults,
// and optional host configuration functions can be provided via hostOpts.
// The DSN is generated in the format:
//
//	"root:<MYSQL_ROOT_PASSWORD>@tcp(<actualPort>)/<MYSQL_DATABASE>?parseTime=true".
func NewMySQLWithOptions(t testing.TB, runOpts []RunOption, hostOpts ...func(*docker.HostConfig)) (*sql.DB, func()) {
	// Set default run options for MySQL.
	defaultRunOpts := &dockertest.RunOptions{
		Repository: defaultMySQLImage,
		Tag:        defaultMySQLTag,
		Env: []string{
			"MYSQL_ROOT_PASSWORD=secret",
			"MYSQL_DATABASE=test",
		},
	}

	// Apply any provided RunOption functions to override defaults.
	for _, opt := range runOpts {
		opt(defaultRunOpts)
	}

	pass := getEnvValue(defaultRunOpts.Env, "MYSQL_ROOT_PASSWORD")
	db := getEnvValue(defaultRunOpts.Env, "MYSQL_DATABASE")

	return NewDockerDB(t, defaultRunOpts, "3306/tcp", "mysql", func(actualPort string) string {
		return fmt.Sprintf("root:%s@tcp(%s)/%s?parseTime=true", pass, actualPort, db)
	}, hostOpts...)
}

const (
	defaultPostgresImage = "postgres"
	defaultPostgresTag   = "13"
)

// NewPostgres starts a PostgreSQL Docker container using the default settings and returns a connected *sql.DB
// along with a cleanup function. It uses the default PostgreSQL image ("postgres") with tag "13". For more
// customization, use NewPostgresWithOptions.
func NewPostgres(t testing.TB) (*sql.DB, func()) {
	return NewPostgresWithOptions(t, nil)
}

// NewPostgresWithOptions starts a PostgreSQL Docker container using Docker and returns a connected *sql.DB
// along with a cleanup function. It applies the default settings:
//   - Repository: "postgres"
//   - Tag: "13"
//   - Environment: POSTGRES_PASSWORD=secret, POSTGRES_DB=test
//
// Additional RunOption functions can be provided via the runOpts parameter to override these defaults,
// and optional host configuration functions can be provided via hostOpts.
// The DSN is generated in the format:
//
//	"postgres://postgres:<POSTGRES_PASSWORD>@<actualPort>/<POSTGRES_DB>?sslmode=disable".
func NewPostgresWithOptions(t testing.TB, runOpts []RunOption, hostOpts ...func(*docker.HostConfig)) (*sql.DB, func()) {
	// Set default run options for PostgreSQL.
	defaultRunOpts := &dockertest.RunOptions{
		Repository: defaultPostgresImage,
		Tag:        defaultPostgresTag,
		Env: []string{
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_DB=test",
		},
	}

	// Apply any provided RunOption functions to override defaults.
	for _, opt := range runOpts {
		opt(defaultRunOpts)
	}

	pass := getEnvValue(defaultRunOpts.Env, "POSTGRES_PASSWORD")
	db := getEnvValue(defaultRunOpts.Env, "POSTGRES_DB")

	return NewDockerDB(t, defaultRunOpts, "5432/tcp", "postgres", func(actualPort string) string {
		return fmt.Sprintf("postgres://postgres:%s@%s/%s?sslmode=disable", pass, actualPort, db)
	}, hostOpts...)
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

// getEnvValue searches the given slice of environment variable strings for the specified key
// and returns its value. If the key is not found, it returns an empty string.
func getEnvValue(env []string, key string) string {
	prefix := key + "="
	for _, v := range env {
		if len(v) >= len(prefix) && v[:len(prefix)] == prefix {
			return v[len(prefix):]
		}
	}
	return ""
}
