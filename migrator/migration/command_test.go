package migration

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunHelpWritesToConfiguredWriter(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	runner := NewRunner(nil, Config{Writer: &out})

	if err := runner.Run(context.Background(), []string{"help"}); err != nil {
		t.Fatalf("Run(help) error = %v", err)
	}
	if !strings.Contains(out.String(), "usage: go run ./cmd/migrate") {
		t.Fatalf("help output = %q, want usage text", out.String())
	}
}

func TestRunCreateAliasesRequireName(t *testing.T) {
	t.Parallel()

	runner := NewRunner(nil, Config{Writer: &bytes.Buffer{}})
	for _, command := range []string{"make", "create", "new"} {
		if err := runner.Run(context.Background(), []string{command}); err == nil {
			t.Fatalf("Run(%s) error = nil, want required name error", command)
		}
	}
}

func TestRunCreateAliasCreatesMigration(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runner := NewRunner(nil, Config{
		Dir:    dir,
		Writer: &bytes.Buffer{},
		Now: func() time.Time {
			return time.Date(2026, 6, 27, 8, 0, 0, 0, time.UTC)
		},
	})

	if err := runner.Run(context.Background(), []string{"create", "add_roles_table"}); err != nil {
		t.Fatalf("Run(create) error = %v", err)
	}

	assertFileContains(t, filepath.Join(dir, "20260627080000_add_roles_table.up.sql"), "-- Write migration SQL here.\n")
	assertFileContains(t, filepath.Join(dir, "20260627080000_add_roles_table.down.sql"), "-- Write rollback SQL here.\n")
}

func TestNeedsDatabase(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args []string
		want bool
	}{
		"empty":    {args: nil, want: false},
		"help":     {args: []string{"help"}, want: false},
		"make":     {args: []string{"make", "create_users_table"}, want: false},
		"create":   {args: []string{"create", "create_users_table"}, want: false},
		"new":      {args: []string{"new", "create_users_table"}, want: false},
		"up":       {args: []string{"up"}, want: true},
		"status":   {args: []string{"status"}, want: true},
		"rollback": {args: []string{"rollback"}, want: true},
	}
	for name, tt := range tests {
		if got := NeedsDatabase(tt.args); got != tt.want {
			t.Fatalf("%s: NeedsDatabase(%v) = %v, want %v", name, tt.args, got, tt.want)
		}
	}
}
