package sqltest_test

import (
	"github.com/ory/dockertest/v3"
	"testing"

	"github.com/ory/dockertest/v3/docker"
	"github.com/vvatanabe/sqltest"
)

// TestDefaultMySQL demonstrates using NewMySQL with default options.
func TestDefaultMySQL(t *testing.T) {
	// Start a MySQL container with default options.
	db, cleanup := sqltest.NewMySQL(t)
	defer cleanup()

	// Schema SQL for creating a table in MySQL.
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		email VARCHAR(255) NOT NULL UNIQUE
	);
	`
	// SQL for inserting initial data.
	insertStmt := `INSERT INTO users (name, email) VALUES ('Alice', 'alice@example.com');`

	// Prepare the database by creating the table and inserting initial data.
	if err := sqltest.PrepDatabase(t, db, sqltest.InitialDBSetup{
		SchemaSQL:   schema,
		InitialData: []string{insertStmt},
	}); err != nil {
		t.Fatalf("PrepDatabase failed: %v", err)
	}

	// Validate that the data was inserted correctly.
	var name, email string
	err := db.QueryRow("SELECT name, email FROM users WHERE email = ?", "alice@example.com").Scan(&name, &email)
	if err != nil {
		t.Fatalf("failed to retrieve data: %v", err)
	}

	if name != "Alice" {
		t.Errorf("expected name 'Alice', but got '%s'", name)
	}
	if email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', but got '%s'", email)
	}
}

// TestMySQLWithCustomRunOptions demonstrates overriding default RunOptions.
func TestMySQLWithCustomRunOptions(t *testing.T) {
	// Custom RunOption to override the default environment variables.
	customEnv := func(opts *dockertest.RunOptions) {
		opts.Env = []string{
			"MYSQL_ROOT_PASSWORD=secret",
			// Override the default database name.
			"MYSQL_DATABASE=custom_test",
		}
	}

	// Start a MySQL container with a custom database name.
	db, cleanup := sqltest.NewMySQLWithOptions(t, []sqltest.RunOption{customEnv})
	defer cleanup()

	// Schema SQL for creating a table.
	schema := `
	CREATE TABLE IF NOT EXISTS customers (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		email VARCHAR(255) NOT NULL UNIQUE
	);
	`
	insertStmt := `INSERT INTO customers (name, email) VALUES ('Bob', 'bob@example.com');`

	// Prepare the database.
	if err := sqltest.PrepDatabase(t, db, sqltest.InitialDBSetup{
		SchemaSQL:   schema,
		InitialData: []string{insertStmt},
	}); err != nil {
		t.Fatalf("PrepDatabase failed: %v", err)
	}

	// Validate data insertion.
	var name, email string
	err := db.QueryRow("SELECT name, email FROM customers WHERE email = ?", "bob@example.com").Scan(&name, &email)
	if err != nil {
		t.Fatalf("failed to retrieve data: %v", err)
	}

	if name != "Bob" {
		t.Errorf("expected name 'Bob', but got '%s'", name)
	}
	if email != "bob@example.com" {
		t.Errorf("expected email 'bob@example.com', but got '%s'", email)
	}
}

// TestDefaultPostgres demonstrates using NewPostgres with default options.
func TestDefaultPostgres(t *testing.T) {
	// Start a PostgreSQL container with default options.
	db, cleanup := sqltest.NewPostgres(t)
	defer cleanup()

	// Schema SQL for creating a table in PostgreSQL.
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		email VARCHAR(255) NOT NULL UNIQUE
	);
	`
	insertStmt := `INSERT INTO users (name, email) VALUES ('Charlie', 'charlie@example.com');`

	// Prepare the database.
	if err := sqltest.PrepDatabase(t, db, sqltest.InitialDBSetup{
		SchemaSQL:   schema,
		InitialData: []string{insertStmt},
	}); err != nil {
		t.Fatalf("PrepDatabase failed: %v", err)
	}

	// Validate data; note PostgreSQL uses $1, $2, ... as parameter placeholders.
	var name, email string
	err := db.QueryRow("SELECT name, email FROM users WHERE email = $1", "charlie@example.com").Scan(&name, &email)
	if err != nil {
		t.Fatalf("failed to retrieve data: %v", err)
	}

	if name != "Charlie" {
		t.Errorf("expected name 'Charlie', but got '%s'", name)
	}
	if email != "charlie@example.com" {
		t.Errorf("expected email 'charlie@example.com', but got '%s'", email)
	}
}

// TestPostgresWithCustomHostOptions demonstrates providing host configuration options (e.g., AutoRemove = true).
func TestPostgresWithCustomHostOptions(t *testing.T) {
	// Host option to set AutoRemove to true.
	autoRemove := func(hc *docker.HostConfig) {
		hc.AutoRemove = true
	}

	// Start a PostgreSQL container with the AutoRemove option.
	db, cleanup := sqltest.NewPostgresWithOptions(t, nil, autoRemove)
	defer cleanup()

	// Schema SQL for creating a table.
	schema := `
	CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY,
		item VARCHAR(255) NOT NULL,
		quantity INT NOT NULL
	);
	`
	insertStmt := `INSERT INTO orders (item, quantity) VALUES ('Widget', 10);`

	// Prepare the database.
	if err := sqltest.PrepDatabase(t, db, sqltest.InitialDBSetup{
		SchemaSQL:   schema,
		InitialData: []string{insertStmt},
	}); err != nil {
		t.Fatalf("PrepDatabase failed: %v", err)
	}

	// Validate the inserted data.
	var item string
	var quantity int
	err := db.QueryRow("SELECT item, quantity FROM orders WHERE item = $1", "Widget").Scan(&item, &quantity)
	if err != nil {
		t.Fatalf("failed to retrieve data: %v", err)
	}

	if item != "Widget" {
		t.Errorf("expected item 'Widget', but got '%s'", item)
	}
	if quantity != 10 {
		t.Errorf("expected quantity 10, but got %d", quantity)
	}
}
