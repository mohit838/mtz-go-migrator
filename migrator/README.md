# mtz-migrator

A lightweight, dependency-free SQL migration library for Go.

Handles the boring migration mechanics so your service handles the database connection.
Built for PostgreSQL via Go's standard `database/sql` interface.

---

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [CLI Reference](#cli-reference)
- [API Reference](#api-reference)
- [Migration File Format](#migration-file-format)
- [Project Layout](#project-layout)
- [Safety & Integrity](#safety--integrity)
- [Production Usage](#production-usage)
- [Contributing](#contributing)
- [License](#license)

---

## Features

- **Paired migrations** — every change has a `.up.sql` and `.down.sql`
- **Batch rollback** — roll back all migrations from the last `up` run at once
- **Checksum protection** — prevents silent modification of applied migrations
- **No database for file commands** — `make`, `create`, and `help` run without a DB connection
- **Zero external dependencies** — uses only the Go standard library
- **Configurable** — custom table name, migration directory, log writer, and clock
- **Portable** — works in single-app projects and microservice monorepos

---

## Quick Start

**1. Add the library to your project:**

```sh
go get github.com/mohit838/mtz-migrator/migrator/migration
```

**2. Create the migration CLI entry point:**

```
cmd/migrate/main.go
migrations/
```

**3. Paste this into `cmd/migrate/main.go`:**

```go
package main

import (
    "context"
    "database/sql"
    "log"
    "os"

    _ "github.com/jackc/pgx/v5/stdlib"
    "github.com/mohit838/mtz-migrator/migrator/migration"
)

func main() {
    args := os.Args[1:]

    cfg := migration.Config{
        Dir:         "migrations",
        ServiceName: "my-service",
    }

    // make, create, new, help — no DB needed
    if !migration.NeedsDatabase(args) {
        runner := migration.NewRunner(nil, cfg)
        if err := runner.Run(context.Background(), args); err != nil {
            log.Fatal(err)
        }
        return
    }

    // up, status, rollback — DB required
    db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    runner := migration.NewRunner(db, cfg)
    if err := runner.Run(context.Background(), args); err != nil {
        log.Fatal(err)
    }
}
```

**4. Create your first migration:**

```sh
go run ./cmd/migrate make create_users_table
```

**5. Edit the generated `.up.sql` and `.down.sql`, then apply:**

```sh
go run ./cmd/migrate up
```

That's it. ✅

---

## Installation

```sh
go get github.com/mohit838/mtz-migrator/migrator/migration
```

Requires Go 1.21 or later.

The library has **zero external dependencies** — it only uses the Go standard library.

Your project is responsible for the database driver (e.g., `github.com/jackc/pgx/v5/stdlib`).

---

## CLI Reference

All commands are run through your own `cmd/migrate/main.go`.

```
go run ./cmd/migrate [command] [name]
```

| Command | Alias | DB needed | Description |
|---------|-------|-----------|-------------|
| `up` | — | ✅ | Apply all pending migrations in version order |
| `status` | — | ✅ | Print the state of every migration file |
| `rollback` | `down` | ✅ | Roll back all migrations from the latest batch |
| `make <name>` | `create`, `new` | ❌ | Generate a paired `.up.sql` / `.down.sql` file |
| `help` | `-h`, `--help` | ❌ | Print usage |

### Examples

```sh
# Create a new migration
go run ./cmd/migrate make add_email_to_users

# Apply pending migrations
go run ./cmd/migrate up

# Check current status
go run ./cmd/migrate status

# Roll back the latest batch
go run ./cmd/migrate rollback
```

### Using `go -C` (Go 1.21+)

Run migration commands from any directory without `cd`:

```sh
go -C services/auth run ./cmd/migrate up
go -C services/payments run ./cmd/migrate status
```

---

## API Reference

### `Config`

```go
type Config struct {
    Dir         string       // Migration folder. Default: "migrations"
    TableName   string       // Tracking table. Default: "schema_migrations"
    ServiceName string       // Optional label for logs and tracking rows
    Writer      io.Writer    // Log output. Default: os.Stdout
    Now         func() time.Time // Clock for generated timestamps. Default: time.Now
}
```

`Writer` and `Now` are mainly useful for tests and tools that want to capture output.

---

### `NewRunner`

```go
func NewRunner(db *sql.DB, cfg Config) *Runner
```

Creates a new migration runner. Pass `nil` for `db` when running commands that don't need a database connection (`make`, `help`).

---

### `Run`

```go
func (r *Runner) Run(ctx context.Context, args []string) error
```

Dispatches to the appropriate command based on `args[0]`. This is the main entry point used by the CLI.

---

### `NeedsDatabase`

```go
func NeedsDatabase(args []string) bool
```

Returns `false` for `make`, `create`, `new`, `help`, `-h`, `--help`. Use this to skip opening a DB connection for commands that don't need one.

---

### `Up`

```go
func (r *Runner) Up(ctx context.Context) error
```

Applies all pending migrations in ascending version order. Each `up` run is assigned a new batch number for rollback grouping.

---

### `Rollback`

```go
func (r *Runner) Rollback(ctx context.Context) error
```

Rolls back all migrations from the latest batch in reverse order.

---

### `Status`

```go
func (r *Runner) Status(ctx context.Context) error
```

Prints the version, name, and state (`pending` or `ran batch=N`) of every migration file.

---

### `Make`

```go
func (r *Runner) Make(name string) error
```

Creates a paired `<timestamp>_<name>.up.sql` and `<timestamp>_<name>.down.sql` file in the configured directory.

---

## Migration File Format

Each migration is a pair of SQL files:

```
migrations/
├── 20260627120000_create_users_table.up.sql    ← applies the change
└── 20260627120000_create_users_table.down.sql  ← rolls it back
```

**Naming rules:**
- Version is a 14-digit UTC timestamp: `YYYYMMDDHHMMSS`
- Name is lowercase with underscores: `create_users_table`
- Both `.up.sql` and `.down.sql` are required — an unpaired file is an error

**Example `.up.sql`:**

```sql
CREATE TABLE users (
    id         BIGSERIAL PRIMARY KEY,
    email      TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Example `.down.sql`:**

```sql
DROP TABLE IF EXISTS users;
```

> [!WARNING]
> Do **not** edit a `.up.sql` file after it has been applied. The checksum will no longer match and the next `up` will fail. Create a new migration instead.

---

## Tracking Table

The library creates a `schema_migrations` table automatically on the first `up` or `status` run:

```sql
CREATE TABLE schema_migrations (
    id           BIGSERIAL PRIMARY KEY,
    version      TEXT        NOT NULL UNIQUE,
    name         TEXT        NOT NULL,
    service_name TEXT        NOT NULL DEFAULT '',
    batch        INTEGER     NOT NULL,
    checksum     TEXT        NOT NULL,
    applied_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    execution_ms BIGINT      NOT NULL DEFAULT 0
);
```

Override the table name in `Config.TableName` if needed.

---

## Project Layout

### Single app

```
my-app/
├── cmd/
│   └── migrate/
│       └── main.go
├── migrations/
│   ├── 20260627120000_create_users_table.up.sql
│   └── 20260627120000_create_users_table.down.sql
└── ...
```

### Microservice monorepo

Each service owns its DB connection, its `.env`, and its `migrations/` folder:

```
services/
├── auth/
│   ├── cmd/migrate/main.go
│   └── migrations/
├── payments/
│   ├── cmd/migrate/main.go
│   └── migrations/
└── notifications/
    ├── cmd/migrate/main.go
    └── migrations/
```

Run from the repo root:

```sh
go -C services/auth     run ./cmd/migrate up
go -C services/payments run ./cmd/migrate status
```

---

## Safety & Integrity

| Rule | Behaviour |
|------|-----------|
| Checksum protection | `up` fails if a previously-applied `.up.sql` has been modified |
| Paired files required | An `.up.sql` without a matching `.down.sql` (or vice versa) is an error |
| Version format enforced | Versions must be 14-digit timestamps |
| Safe table identifier | `TableName` is validated to contain only `[a-zA-Z0-9_]` with a letter or `_` first |
| No overwrite on `make` | Generated files are never overwritten |
| Transactional execution | Each migration runs inside a transaction; failure rolls back the SQL and the tracking row |

---

## Production Usage

> [!IMPORTANT]
> **Do not run migrations inside the API server process.**

Run migrations only during deploy when there is a schema change:

```sh
# In CI/CD, before starting the new server binary
go -C services/auth run ./cmd/migrate up
```

If there is no schema change this deploy, skip the migration step entirely.

---

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for the full guide.

Short version:

1. Fork the repo and create a feature branch
2. Make your change
3. Run `go test ./...` — all tests must pass
4. Open a pull request with a clear description

---

## License

MIT — see [LICENSE](./LICENSE) for details.
