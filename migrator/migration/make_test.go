package migration

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMakeCreatesPairedMigrationFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var out bytes.Buffer
	runner := NewRunner(nil, Config{
		Dir:         dir,
		ServiceName: "auth",
		Writer:      &out,
		Now: func() time.Time {
			return time.Date(2026, 6, 27, 7, 30, 0, 0, time.UTC)
		},
	})

	if err := runner.Make("Create Users Table"); err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	upPath := filepath.Join(dir, "20260627073000_create_users_table.up.sql")
	downPath := filepath.Join(dir, "20260627073000_create_users_table.down.sql")
	assertFileContains(t, upPath, "-- Write migration SQL here.\n")
	assertFileContains(t, downPath, "-- Write rollback SQL here.\n")

	if got := out.String(); got == "" {
		t.Fatal("expected Make to write command output")
	}
}

func TestMakeDoesNotOverwriteExistingFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runner := NewRunner(nil, Config{
		Dir:    dir,
		Writer: &bytes.Buffer{},
		Now: func() time.Time {
			return time.Date(2026, 6, 27, 7, 30, 0, 0, time.UTC)
		},
	})

	if err := runner.Make("create_users_table"); err != nil {
		t.Fatalf("first Make() error = %v", err)
	}
	if err := runner.Make("create_users_table"); err == nil {
		t.Fatal("second Make() error = nil, want file exists error")
	}
}

func TestSanitizeName(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"Create Users Table":       "create_users_table",
		"  create-users table!!  ": "create_users_table",
		"roles":                    "roles",
		"!!!":                      "",
	}
	for input, want := range tests {
		if got := sanitizeName(input); got != want {
			t.Fatalf("sanitizeName(%q) = %q, want %q", input, got, want)
		}
	}
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(content) != want {
		t.Fatalf("content of %s = %q, want %q", path, string(content), want)
	}
}
