# mtz-migrator

A lightweight, dependency-free SQL migration engine written in Go.

Handles the mechanics of schema migration (tracking, checksumming, version checking, rolling back) while leaving the database connection configuration in your hands. Built for PostgreSQL via Go's standard `database/sql` interface.

This repository is organized as a monorepo containing both the core library and a minimal, Postgres-based testing ground/demo application.

---

## Table of Contents

- [Project Structure](#project-structure)
- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [CLI Reference](#cli-reference)
- [API Reference](#api-reference)
- [Migration File Format](#migration-file-format)
- [Tracking Table Schema](#tracking-table-schema)
- [Running the Test App & CLI](#running-the-test-app--cli)
- [Running Tests](#running-tests)
- [Safety & Integrity Rules](#safety--integrity-rules)
- [Production Best Practices](#production-best-practices)
- [Contributing](#contributing)
- [License](#license)

---

## Project Structure

```
.
├── migrator/                  # Core migration library module
│   ├── migration/             # Migration engine implementation
│   │   ├── checksum.go        # Integrity & checksum verification
│   │   ├── command.go         # CLI dispatcher & NeedsDatabase
│   │   ├── files.go           # File loaders & validation
│   │   ├── make.go            # Migration file generator (make/create/new)
│   │   └── runner.go          # Database operations (Up, Status, Rollback)
│   ├── CONTRIBUTING.md        # Contribution guidelines
│   ├── README.md              # Library README
│   └── go.mod                 # Core library module definition
│
└── test/                      # Minimal demo app & migration testing ground
    ├── cmd/
    │   ├── api/               # HTTP API server with health probes
    │   └── migrate/           # CLI entry point wrapping the migrator library
    ├── internal/
    │   ├── config/            # Configuration loaders (.env)
    │   ├── database/          # PostgreSQL database connector
    │   └── router/            # Route configuration and health handlers
    ├── migrations/            # Demo migrations (.up.sql / .down.sql pairs)
    ├── go.mod                 # Test application Go module (references local migrator)
    └── integration_test.go    # End-to-end integration tests
```

---

## Features

- **Paired migrations** — every change requires a `.up.sql` and `.down.sql`
- **Batch rollback** — roll back all migrations from the latest `up` execution at once
- **Checksum protection** — prevents silent modifications of applied migrations
- **No database required for local commands** — `make`, `create`, `new`, and `help` execute without a DB connection
- **Zero external dependencies** — uses only the Go standard library
- **Highly configurable** — override the table name, migration directory, log writer, and clock source
- **Microservice ready** — works seamlessly in single-app structures and microservice monorepos

---

## Installation

```sh
go get github.com/mohit838/mtz-migrator/migrator/migration
```

Requires Go 1.21 or later. The library itself has **zero external dependencies**. Your application is responsible for importing the database driver (e.g., `github.com/jackc/pgx/v5/stdlib`).

---

## Quick Start

**1. Create your migration CLI entry point:**

```
cmd/migrate/main.go
migrations/
```

**2. Paste this template into `cmd/migrate/main.go`:**

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

    // local file operations do not require a DB connection
    if !migration.NeedsDatabase(args) {
        runner := migration.NewRunner(nil, cfg)
        if err := runner.Run(context.Background(), args); err != nil {
            log.Fatal(err)
        }
        return
    }

    // up, status, rollback require an active database connection
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

**3. Generate your first paired migration:**

```sh
go run ./cmd/migrate make create_users_table
```

**4. Edit the generated `.up.sql` and `.down.sql`, then apply:**

```sh
go run ./cmd/migrate up
```

---

## CLI Reference

All commands are run through your own CLI entry point (`go run ./cmd/migrate`).

| Command | Aliases | DB Needed | Description |
|---------|---------|-----------|-------------|
| `up` | — | ✅ Yes | Apply all pending migrations in ascending chronological order |
| `status` | — | ✅ Yes | Show the status (pending/ran batch) of all migration files |
| `rollback` | `down` | ✅ Yes | Roll back the latest batch of applied migrations |
| `make <name>` | `create`, `new` | ❌ No | Generate a paired `.up.sql` / `.down.sql` file |
| `help` | `-h`, `--help` | ❌ No | Show command usage |

### Examples

```sh
# Generate new migration files
go run ./cmd/migrate make add_email_index

# Run all pending migrations
go run ./cmd/migrate up

# Check status of migrations
go run ./cmd/migrate status

# Rollback last run batch
go run ./cmd/migrate rollback
```

---

## API Reference

### `Config` Struct

```go
type Config struct {
    Dir         string           // Migration folder. Default: "migrations"
    TableName   string           // Tracking database table. Default: "schema_migrations"
    ServiceName string           // Label for log outputs and tracking records
    Writer      io.Writer        // Destination for log output. Default: os.Stdout
    Now         func() time.Time // Clock for file generation. Default: time.Now
}
```

### `NewRunner`

```go
func NewRunner(db *sql.DB, cfg Config) *Runner
```

Initializes the migration runner. `db` can be `nil` when running `NeedsDatabase(args) == false` commands.

### `Run`

```go
func (r *Runner) Run(ctx context.Context, args []string) error
```

Dispatches command-line arguments to the appropriate runner action.

---

## Migration File Format

Migrations are stored as pairs of versioned SQL files:

```
migrations/
├── 20260627120000_create_users_table.up.sql    ← Applies changes
└── 20260627120000_create_users_table.down.sql  ← Reverts changes
```

- **Version**: A 14-digit UTC timestamp (`YYYYMMDDHHMMSS`)
- **Name**: Lowercase with underscores (e.g., `create_users_table`)
- **Pairs**: Both `.up.sql` and `.down.sql` must exist. An unpaired file returns an error.

> [!WARNING]
> Do **not** edit an `.up.sql` file after it has been applied. The runner validates SHA-256 checksums, and modifying applied files will cause future runs to fail.

---

## Tracking Table Schema

The tracking table is automatically created by the runner on the first `up` or `status` run:

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

---

## Running the Test App & CLI

The `test/` directory is configured as a fully working demo environment.

1. Navigate to the test app folder:
   ```sh
   cd test
   ```
2. Configure credentials in `.env`:
   ```sh
   cp .env.example .env
   ```
3. Run CLI commands:
   ```sh
   go run ./cmd/migrate status
   go run ./cmd/migrate up
   ```
4. Run the API:
   ```sh
   go run ./cmd/api
   ```

---

## Running Tests

### Unit Tests
To run core library tests (no database needed):
```sh
cd migrator
go test -v ./...
```

### Integration Tests
To run end-to-end integration tests (requires active database defined in `test/.env`):
```sh
cd test
go test -v ./...
```

---

## Safety & Integrity Rules

- **Transactional Execution**: Each migration file executes inside its own database transaction. Failure rolls back both the SQL and the tracking row.
- **Checksum Protection**: The runner checks the SHA-256 hash of existing `.up.sql` files against the database on every run.
- **Table Name Validation**: The configured `TableName` is strictly validated to prevent SQL injection.
- **No Overwrite**: Generated migration files will never overwrite existing files.

---

## Production Best Practices

> [!IMPORTANT]
> **Never run migrations concurrently inside the API server startup routine.**
> Run migrations as a separate step in your CI/CD pipeline or deployment job *prior* to starting the server container:
> ```sh
> go run ./cmd/migrate up
> ```

---

## Contributing

For instructions on setting up your local environment, running integration suites, and formatting your code, check [CONTRIBUTING.md](./migrator/CONTRIBUTING.md).

## License

This project is licensed under the MIT License. See [LICENSE](./LICENSE) for details.
