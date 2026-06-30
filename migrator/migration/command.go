package migration

import (
	"context"
	"fmt"
)

// Run processes the command-line arguments and dispatches them to the corresponding Runner method.
// Returns an error if the command is unrecognized, missing parameters, or if the execution fails.
func (r *Runner) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%s", Usage())
	}

	switch args[0] {
	case "up":
		return r.Up(ctx)
	case "down", "rollback":
		return r.Rollback(ctx)
	case "status":
		return r.Status(ctx)
	case "make", "create", "new":
		if len(args) < 2 {
			return fmt.Errorf("migration name is required\n\n%s", Usage())
		}
		return r.Make(args[1])
	case "help", "-h", "--help":
		r.println(Usage())
		return nil
	default:
		return fmt.Errorf("unknown migration command: %s\n\n%s", args[0], Usage())
	}
}

// NeedsDatabase checks if the command-line arguments require an active database connection.
// Returns false for commands like make, create, new, help, and true for up, status, rollback.
func NeedsDatabase(args []string) bool {
	if len(args) == 0 {
		return false
	}

	switch args[0] {
	case "up", "status", "down", "rollback":
		return true
	default:
		return false
	}
}

// Usage returns the standard CLI help and commands usage string.
func Usage() string {
	return `usage: go run ./cmd/migrate [command] [name]

commands:
  up                 run all pending migrations
  status             show migration status
  rollback, down     rollback the latest migration batch
  make <name>        create paired .up.sql and .down.sql files
  create <name>      alias for make
  new <name>         alias for make
  help               show this help`
}
