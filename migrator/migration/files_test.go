package migration

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFilesSortsAndPairsMigrations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "20260627073001_second.up.sql"), "SELECT 2;")
	writeTestFile(t, filepath.Join(dir, "20260627073001_second.down.sql"), "SELECT -2;")
	writeTestFile(t, filepath.Join(dir, "20260627073000_first.up.sql"), "SELECT 1;")
	writeTestFile(t, filepath.Join(dir, "20260627073000_first.down.sql"), "SELECT -1;")

	runner := NewRunner(nil, Config{Dir: dir, Writer: &bytes.Buffer{}})
	files, err := runner.loadFiles()
	if err != nil {
		t.Fatalf("loadFiles() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	if files[0].version != "20260627073000" || files[0].name != "first" {
		t.Fatalf("first migration = %s_%s, want 20260627073000_first", files[0].version, files[0].name)
	}
	if files[1].version != "20260627073001" || files[1].name != "second" {
		t.Fatalf("second migration = %s_%s, want 20260627073001_second", files[1].version, files[1].name)
	}
}

func TestLoadFilesRequiresPairs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "20260627073000_missing_down.up.sql"), "SELECT 1;")

	runner := NewRunner(nil, Config{Dir: dir, Writer: &bytes.Buffer{}})
	if _, err := runner.loadFiles(); err == nil {
		t.Fatal("loadFiles() error = nil, want missing pair error")
	}
}

func TestLoadFilesRejectsInvalidVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "not_a_timestamp_name.up.sql"), "SELECT 1;")
	writeTestFile(t, filepath.Join(dir, "not_a_timestamp_name.down.sql"), "SELECT -1;")

	runner := NewRunner(nil, Config{Dir: dir, Writer: &bytes.Buffer{}})
	if _, err := runner.loadFiles(); err == nil {
		t.Fatal("loadFiles() error = nil, want invalid version error")
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
