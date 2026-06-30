package migration

import (
	"strings"
	"testing"
)

func TestSafeIdentifier(t *testing.T) {
	t.Parallel()

	tests := map[string]bool{
		"schema_migrations": true,
		"SchemaMigrations":  true,
		"_private":          true,
		"schema1":           true,
		"":                  false,
		"1schema":           false,
		"schema-name":       false,
		"schema.name":       false,
	}
	for input, want := range tests {
		if got := safeIdentifier(input); got != want {
			t.Fatalf("safeIdentifier(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestRollbackFilesUsesLatestBatchInReverseVersionOrder(t *testing.T) {
	t.Parallel()

	files := []fileMigration{
		{version: "20260627073000", name: "first"},
		{version: "20260627073001", name: "second"},
		{version: "20260627073002", name: "third"},
	}
	applied := map[string]appliedMigration{
		"20260627073000": {version: "20260627073000", batch: 1},
		"20260627073001": {version: "20260627073001", batch: 2},
		"20260627073002": {version: "20260627073002", batch: 2},
	}

	got, batch, err := rollbackFiles(files, applied)
	if err != nil {
		t.Fatalf("rollbackFiles() error = %v", err)
	}
	if batch != 2 {
		t.Fatalf("batch = %d, want 2", batch)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].version != "20260627073002" || got[1].version != "20260627073001" {
		t.Fatalf("rollback order = [%s %s], want reverse latest batch order", got[0].version, got[1].version)
	}
}

func TestRollbackFilesErrorsWhenLatestBatchHasNoLocalFiles(t *testing.T) {
	t.Parallel()

	files := []fileMigration{
		{version: "20260627073000", name: "first"},
	}
	applied := map[string]appliedMigration{
		"20260627073000": {version: "20260627073000", batch: 1},
		"20260627073001": {version: "20260627073001", batch: 2},
	}

	_, _, err := rollbackFiles(files, applied)
	if err == nil {
		t.Fatal("rollbackFiles() error = nil, want missing local files error")
	}
	if !strings.Contains(err.Error(), "latest migration batch 2 has no matching local migration files") {
		t.Fatalf("rollbackFiles() error = %v, want missing latest batch files error", err)
	}
}
