# sqltest

**sqltest** is a Go testing library that leverages [dockertest](https://github.com/ory/dockertest) to simplify integration testing with SQL databases using Docker containers. It allows you to easily spin up MySQL and PostgreSQL (and, potentially, other SQL databases) within your unit tests, execute schema and data setup, and automatically clean up resources after tests are complete.

## Features

### Docker-based Database Containers
Easily launch and manage database containers for MySQL and PostgreSQL during your tests.

### Automatic Connection and Cleanup
Quickly obtain a connected `*sql.DB` instance along with a cleanup function that removes the container when the test finishes.

### Flexible Database Setup
Set up the database schema and seed initial data using provided helper functions that execute SQL statements (with transactional support for data insertion).

### Version Flexibility
Specify the database version (tag) when starting a container, so you can test against different versions.

## Installation

Use `go get` to install the package:

```bash
go get github.com/vvatanabe/sqltest
```

## Usage

The package provides several key functions:

### NewMySQL
  Starts a MySQL container using Docker. The function accepts a version tag (e.g., `"8.0"`) and returns a connected `*sql.DB` and a cleanup function.

### NewPostgres
Starts a PostgreSQL container using Docker. The function accepts a version tag (e.g., `"13"`) and returns a connected `*sql.DB` and a cleanup function.

### NewDockerDB
A helper function that starts a Docker container with the given run options, waits for the database to be ready, and returns a connected `*sql.DB` along with a cleanup function.

### PrepDatabase
Prepares the test database by executing provided schema (DDL) and initial data (DML) SQL statements. The initial data insertion is performed within a transaction to ensure consistency.

### InitialDBSetup
A helper struct used with `PrepDatabase` to specify the schema and initial data for setting up your test database.

## Example

### MySQL Test Example

Below is an example test that starts a MySQL container, creates a `users` table, inserts a row, and then verifies that the data is stored correctly.

```go
package sqltest_test

import (
	"testing"

	"github.com/vvatanabe/sqltest"
)

func TestSetupMySQL(t *testing.T) {
	// Start a MySQL container and obtain a connection.
	db, clean := sqltest.NewMySQL(t, "8.0")
	defer clean()

	// SQL for creating the table.
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		email VARCHAR(255) NOT NULL UNIQUE
	);
	`

	// SQL for inserting initial data.
	insertStmt := `INSERT INTO users (name, email) VALUES ('John Doe', 'john@example.com');`

	// Create the table and insert initial data.
	if err := sqltest.PrepDatabase(t, db, sqltest.InitialDBSetup{
		SchemaSQL:   schema,
		InitialData: []string{insertStmt},
	}); err != nil {
		t.Fatalf("failed to prepare database: %v", err)
	}

	// Validate the data.
	var name, email string
	err := db.QueryRow("SELECT name, email FROM users WHERE email = ?", "john@example.com").Scan(&name, &email)
	if err != nil {
		t.Fatalf("failed to retrieve data: %v", err)
	}

	if name != "John Doe" {
		t.Errorf("expected name 'John Doe', but got '%s'", name)
	}
	if email != "john@example.com" {
		t.Errorf("expected email 'john@example.com', but got '%s'", email)
	}
}
```

### PostgreSQL Test Example

Below is an example test that starts a PostgreSQL container, creates a `users` table (using PostgreSQL syntax), inserts a row, and verifies the inserted data.

```go
package sqltest_test

import (
	"testing"

	"github.com/vvatanabe/sqltest"
)

func TestSetupPostgres(t *testing.T) {
	// Start a PostgreSQL container and obtain a connection (e.g., version "13").
	db, clean := sqltest.NewPostgres(t, "13")
	defer clean()

	// SQL for creating the table.
	// In PostgreSQL, use SERIAL instead of AUTO_INCREMENT.
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		email VARCHAR(255) NOT NULL UNIQUE
	);
	`

	// SQL for inserting initial data.
	insertStmt := `INSERT INTO users (name, email) VALUES ('John Doe', 'john@example.com');`

	// Create the table and insert initial data.
	if err := sqltest.PrepDatabase(t, db, sqltest.InitialDBSetup{
		SchemaSQL:   schema,
		InitialData: []string{insertStmt},
	}); err != nil {
		t.Fatalf("failed to prepare database: %v", err)
	}

	// Validate the data.
	var name, email string
	// PostgreSQL uses $1, $2, ... as parameter placeholders.
	err := db.QueryRow("SELECT name, email FROM users WHERE email = $1", "john@example.com").Scan(&name, &email)
	if err != nil {
		t.Fatalf("failed to retrieve data: %v", err)
	}

	if name != "John Doe" {
		t.Errorf("expected name 'John Doe', but got '%s'", name)
	}
	if email != "john@example.com" {
		t.Errorf("expected email 'john@example.com', but got '%s'", email)
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
