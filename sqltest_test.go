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

func TestSetupPostgres(t *testing.T) {
	// Start a PostgreSQL container and obtain a connection (version can be specified, e.g., "13").
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
	// In PostgreSQL, parameter placeholders use $1, $2, etc.
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
