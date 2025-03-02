# sqltest

**sqltest** is a Go testing library that leverages [dockertest](https://github.com/ory/dockertest) to simplify integration testing with SQL databases using Docker containers. It allows you to easily spin up MySQL and PostgreSQL (and, potentially, other SQL databases) within your unit tests, execute schema and data setup, and automatically clean up resources after tests are complete.

## Features

### Docker-based Database Containers
Easily launch and manage database containers for MySQL and PostgreSQL during your tests.

### Automatic Connection and Cleanup
Quickly obtain a connected `*sql.DB` instance along with a cleanup function that removes the container when the test finishes.

### Flexible Database Setup
Set up the database schema and seed initial data using provided helper functions that execute SQL statements (with transactional support for data insertion).

## Installation

Use `go get` to install the package:

```bash
go get github.com/vvatanabe/sqltest
```

## Usage

The package provides several key functions:

### NewMySQL
This function starts a MySQL Docker container using default settings. It uses the MySQL image (`"mysql"`) with the default tag (`"8.0"`). It returns a connected `*sql.DB` instance along with a cleanup function that ensures the container is removed after the test completes. For most cases, you can use NewMySQL directly for a quick setup.

### NewMySQLWithOptions
For advanced usage, `NewMySQLWithOptions` allows you to customize the container’s settings. In addition to the defaults used by `NewMySQL`, you can pass one or more `RunOption` functions to override any default configuration (for example, changing the environment variables, command, mounts, etc.).
You can also provide optional host configuration options (via variadic functions) that allow you to adjust Docker’s `HostConfig` settings (e.g., setting `AutoRemove` to true).

### NewPostgres
This function starts a PostgreSQL Docker container using default settings. It uses the PostgreSQL image (`"postgres"`) with the default tag (`"13"`). It returns a connected *sql.DB and a cleanup function that removes the container after the test is done. For most cases, you can use NewPostgres directly for a quick setup.

### NewPostgresWithOptions
Similar to the MySQL variant, `NewPostgresWithOptions` allows you to override the default settings by accepting additional `RunOption` functions. You can customize the container configuration (e.g., changing environment variables or other run options) and supply optional host configuration functions to adjust Docker's `HostConfig` (such as setting `AutoRemove`).

### NewDockerDB
A helper function that starts a Docker container with the given run options, waits for the database to be ready, and returns a connected `*sql.DB` along with a cleanup function.

### PrepDatabase
Prepares the test database by executing provided schema (DDL) and initial data (DML) SQL statements. The initial data insertion is performed within a transaction to ensure consistency.

### InitialDBSetup
A helper struct used with `PrepDatabase` to specify the schema and initial data for setting up your test database.

## Example

Below is an example test that starts a MySQL and Postgres container, creates a `users` table, inserts a row, and then verifies that the data is stored correctly.

```go
package sqltest_test

import (
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/vvatanabe/sqltest"
	"testing"
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
```

## Running Tests

Since **sqltest** is intended for use in unit tests, you can run your tests as usual:

```bash
go test -v ./...
```

## Acknowledgments

- [dockertest](https://github.com/ory/dockertest) helps you boot up ephermal docker images for your Go tests with minimal work.
- [dynamotest](https://github.com/upsidr/dynamotest) is a package to help set up a DynamoDB Local Docker instance on your machine as a part of Go test code.

## Authors

* **[vvatanabe](https://github.com/vvatanabe/)** - *Main contributor*
* Currently, there are no other contributors

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.
