# mtz-go-migrator

A small SQL migration library for Go services.

It handles migration files, batches, rollback tracking, checksums, and status output. Your app stays responsible for opening the database connection.

Built for PostgreSQL through Go's standard `database/sql` package. The library itself has no third-party dependencies.

## Install

```sh
go get github.com/mohit838/mtz-go-migrator/migrator/migration
```

Your app still needs a database driver. For PostgreSQL, `pgx` is a common choice:

```sh
go get github.com/jackc/pgx/v5/stdlib
```

## Quick Start

Create this layout in your app:

```text
my-app/
├── cmd/
│   └── migrate/
│       └── main.go
└── migrations/
```

Add `cmd/migrate/main.go`:

```go
package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mohit838/mtz-go-migrator/migrator/migration"
)

func main() {
	args := os.Args[1:]

	cfg := migration.Config{
		Dir:         "migrations",
		ServiceName: "my-service",
	}

	if !migration.NeedsDatabase(args) {
		runner := migration.NewRunner(nil, cfg)
		if err := runner.Run(context.Background(), args); err != nil {
			log.Fatal(err)
		}
		return
	}

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

Create a migration:

```sh
go run ./cmd/migrate make create_users_table
```

This creates two files:

```text
migrations/
├── 20260627120000_create_users_table.up.sql
└── 20260627120000_create_users_table.down.sql
```

Example `20260627120000_create_users_table.up.sql`:

```sql
CREATE TABLE users (
	id BIGSERIAL PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Example `20260627120000_create_users_table.down.sql`:

```sql
DROP TABLE IF EXISTS users;
```

Apply the migration:

```sh
DATABASE_URL='postgres://user:pass@localhost:5432/app?sslmode=disable' go run ./cmd/migrate up
```

Check status:

```sh
go run ./cmd/migrate status
```

Rollback the latest batch:

```sh
go run ./cmd/migrate rollback
```

## Commands

Run commands through your own `cmd/migrate` program:

```sh
go run ./cmd/migrate [command] [name]
```

| Command | Aliases | Needs DB | What it does |
|---------|---------|----------|--------------|
| `up` | none | Yes | Runs all pending `.up.sql` files in version order |
| `status` | none | Yes | Prints each local migration as `pending` or `ran batch=N` |
| `rollback` | `down` | Yes | Runs `.down.sql` files from the latest batch in reverse order |
| `make <name>` | `create`, `new` | No | Creates paired `.up.sql` and `.down.sql` files |
| `help` | `-h`, `--help` | No | Prints usage |

Useful examples:

```sh
go run ./cmd/migrate make add_email_to_users
go run ./cmd/migrate up
go run ./cmd/migrate status
go run ./cmd/migrate rollback
go run ./cmd/migrate down
go run ./cmd/migrate help
```

From a monorepo root, use `go -C`:

```sh
go -C services/auth run ./cmd/migrate up
go -C services/payments run ./cmd/migrate status
```

## Migration File Rules

Migration files must be paired:

```text
<14-digit-version>_<name>.up.sql
<14-digit-version>_<name>.down.sql
```

Example:

```text
20260627120000_create_users_table.up.sql
20260627120000_create_users_table.down.sql
```

Rules:

- The version must be a 14-digit UTC timestamp: `YYYYMMDDHHMMSS`.
- The name should use lowercase words separated by underscores.
- Every `.up.sql` file must have a matching `.down.sql` file.
- Every `.down.sql` file must have a matching `.up.sql` file.
- Do not edit an applied `.up.sql` file. Create a new migration instead.

## API

### `Config`

```go
type Config struct {
	Dir         string
	TableName   string
	ServiceName string
	Writer      io.Writer
	Now         func() time.Time
}
```

Defaults:

| Field | Default | Purpose |
|-------|---------|---------|
| `Dir` | `migrations` | Folder containing SQL migration files |
| `TableName` | `schema_migrations` | Database table used to track applied migrations |
| `ServiceName` | empty | Optional label stored in tracking rows and printed in logs |
| `Writer` | `os.Stdout` | Destination for command output |
| `Now` | `time.Now` | Clock used when generating migration filenames |

### `NewRunner`

```go
runner := migration.NewRunner(db, migration.Config{
	Dir:         "migrations",
	TableName:   "schema_migrations",
	ServiceName: "billing",
})
```

Pass `nil` for `db` when running commands that do not need a database:

```go
runner := migration.NewRunner(nil, migration.Config{Dir: "migrations"})
err := runner.Run(context.Background(), []string{"make", "create_orders_table"})
```

### `Run`

```go
err := runner.Run(context.Background(), os.Args[1:])
```

`Run` is the easiest entry point for a CLI. It dispatches to `Up`, `Status`, `Rollback`, or `Make`.

### `NeedsDatabase`

```go
if migration.NeedsDatabase(os.Args[1:]) {
	// Open the database connection.
}
```

Returns `true` only for `up`, `status`, `rollback`, and `down`.

### Direct Method Usage

You can call methods directly when you do not want a CLI dispatcher:

```go
runner := migration.NewRunner(db, migration.Config{Dir: "migrations"})

if err := runner.Up(context.Background()); err != nil {
	log.Fatal(err)
}
```

Available methods:

- `Up(ctx)` applies pending migrations.
- `Status(ctx)` prints local migration status.
- `Rollback(ctx)` rolls back the latest batch.
- `Make(name)` creates paired migration files.

## Tracking Table

The runner creates the tracking table automatically on `up`, `status`, or `rollback`:

```sql
CREATE TABLE schema_migrations (
	id BIGSERIAL PRIMARY KEY,
	version TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	service_name TEXT NOT NULL DEFAULT '',
	batch INTEGER NOT NULL,
	checksum TEXT NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	execution_ms BIGINT NOT NULL DEFAULT 0
);
```

Use `Config.TableName` when each service or tenant needs a separate tracking table.

## Safety Behavior

- Each migration runs inside a database transaction.
- Applied `.up.sql` files are protected by SHA-256 checksums.
- `up`, `status`, and `rollback` fail if an applied `.up.sql` file was changed.
- `rollback` fails clearly if the latest applied batch no longer has matching local migration files.
- Generated files use `O_EXCL`, so `make` never overwrites existing files.
- `TableName` is validated before it is used in SQL.

## Production Usage

Run migrations as a deploy step before starting the new application version:

```sh
go run ./cmd/migrate up
```

Avoid running migrations from normal API server startup. In multi-instance deployments, startup migrations can run concurrently and fight over schema changes.

## Tests

From this module:

```sh
go test ./...
```

From the repository root:

```sh
go test ./migrator/...
```

## License

MIT
