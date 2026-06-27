package test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/mohit838/mtz-migrator/migrator/migration"
)

func TestMigrationIntegration(t *testing.T) {
	// Load environment variables from .env
	_ = godotenv.Load(".env")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping integration test: DATABASE_URL not set")
	}

	// Connect to database
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Create a unique migration table and directory per test run to avoid conflicts
	tempDir := t.TempDir()
	tableName := fmt.Sprintf("test_schema_migrations_%d", time.Now().UnixNano())

	// Clean up table after test
	defer func() {
		_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName))
		_, _ = db.Exec("DROP TABLE IF EXISTS test_users;")
	}()

	runnerCfg := migration.Config{
		Dir:         tempDir,
		TableName:   tableName,
		ServiceName: "integration-test-service",
	}

	runner := migration.NewRunner(db, runnerCfg)

	// 1. Check initially empty status
	err = runner.Status(context.Background())
	if err != nil {
		t.Fatalf("Failed status on empty migrations: %v", err)
	}

	// 2. Generate a migration using runner.Make
	migrationName := "create_test_users_table"
	err = runner.Make(migrationName)
	if err != nil {
		t.Fatalf("Failed to make migration: %v", err)
	}

	// Find the created files
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read migrations directory: %v", err)
	}

	var upFile, downFile string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".up.sql") {
			upFile = filepath.Join(tempDir, f.Name())
		} else if strings.HasSuffix(f.Name(), ".down.sql") {
			downFile = filepath.Join(tempDir, f.Name())
		}
	}

	if upFile == "" || downFile == "" {
		t.Fatal("Migration files were not generated properly")
	}

	// Write migration content
	upSQL := `
	CREATE TABLE test_users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	downSQL := `DROP TABLE IF EXISTS test_users;`

	if err := os.WriteFile(upFile, []byte(upSQL), 0644); err != nil {
		t.Fatalf("Failed to write to up.sql: %v", err)
	}
	if err := os.WriteFile(downFile, []byte(downSQL), 0644); err != nil {
		t.Fatalf("Failed to write to down.sql: %v", err)
	}

	// 3. Run Up to apply migrations
	err = runner.Up(context.Background())
	if err != nil {
		t.Fatalf("Failed to run Up migrations: %v", err)
	}

	// Verify the table was created by attempting to insert a row
	_, err = db.Exec("INSERT INTO test_users (username) VALUES ('testuser');")
	if err != nil {
		t.Fatalf("Failed to insert into migrated table: %v", err)
	}

	// Check table content count
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_users;").Scan(&count)
	if err != nil || count != 1 {
		t.Fatalf("Failed to retrieve row from migrated table: count=%d, err=%v", count, err)
	}

	// 4. Run Status and verify it lists our migration as ran
	// We'll capture the status output using a custom writer
	var statusBuffer strings.Builder
	statusRunnerCfg := runnerCfg
	statusRunnerCfg.Writer = &statusBuffer
	statusRunner := migration.NewRunner(db, statusRunnerCfg)

	err = statusRunner.Status(context.Background())
	if err != nil {
		t.Fatalf("Failed to get migration status: %v", err)
	}

	statusOutput := statusBuffer.String()
	if !strings.Contains(statusOutput, "ran batch=1") || !strings.Contains(statusOutput, migrationName) {
		t.Fatalf("Unexpected status output: %s", statusOutput)
	}

	// 5. Test Checksum validation
	// Edit the up migration file (which should trigger a checksum validation failure)
	if err := os.WriteFile(upFile, []byte(upSQL+"\n-- modified comment"), 0644); err != nil {
		t.Fatalf("Failed to modify up.sql for checksum check: %v", err)
	}

	err = runner.Up(context.Background())
	if err == nil {
		t.Fatal("Expected Up to fail due to checksum mismatch, but it succeeded")
	}
	if !strings.Contains(err.Error(), "migration checksum changed after apply") {
		t.Fatalf("Expected checksum error, got: %v", err)
	}

	// Restore correct file for rollback test
	if err := os.WriteFile(upFile, []byte(upSQL), 0644); err != nil {
		t.Fatalf("Failed to restore up.sql: %v", err)
	}

	// 6. Run Rollback and verify it cleans up
	err = runner.Rollback(context.Background())
	if err != nil {
		t.Fatalf("Failed to rollback migrations: %v", err)
	}

	// Verify table no longer exists (insert should fail)
	_, err = db.Exec("INSERT INTO test_users (username) VALUES ('another');")
	if err == nil {
		t.Fatal("Expected insert to fail after rollback because table should be dropped")
	}
}
