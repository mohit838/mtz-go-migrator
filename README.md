# mtz-go-migrator

A lightweight SQL migration runner for Go services.

The core library lives in `migrator/`. The `test/` module is a small PostgreSQL-backed demo app used for integration testing.

## What It Does

- Creates paired `.up.sql` and `.down.sql` migration files.
- Applies pending migrations in timestamp order.
- Groups each `up` run into a rollback batch.
- Rolls back the latest batch in reverse order.
- Stores applied versions, names, batches, checksums, service names, and execution time.
- Protects applied `.up.sql` files with SHA-256 checksums.
- Leaves database connection setup to your app.

## Project Layout

```text
.
├── go.work
├── migrator/
│   ├── migration/       # Core library package
│   ├── README.md        # Full library usage guide
│   └── go.mod
└── test/
    ├── cmd/api/         # Demo API
    ├── cmd/migrate/     # Demo migration CLI
    ├── internal/
    ├── migrations/
    └── go.mod
```

## Install

In your own app:

```sh
go get github.com/mohit838/mtz-go-migrator/migrator/migration
```

Add your database driver separately. For PostgreSQL with `pgx`:

```sh
go get github.com/jackc/pgx/v5/stdlib
```

## Quick Usage

Create a CLI entry point:

```text
cmd/migrate/main.go
migrations/
```

Example `cmd/migrate/main.go`:

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

Example `.up.sql`:

```sql
CREATE TABLE users (
	id BIGSERIAL PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Example `.down.sql`:

```sql
DROP TABLE IF EXISTS users;
```

Run commands:

```sh
go run ./cmd/migrate up
go run ./cmd/migrate status
go run ./cmd/migrate rollback
```

See [migrator/README.md](./migrator/README.md) for the full API and usage guide.

## CLI Commands

| Command | Aliases | Needs DB | Description |
|---------|---------|----------|-------------|
| `up` | none | Yes | Apply all pending migrations |
| `status` | none | Yes | Show migration status |
| `rollback` | `down` | Yes | Roll back the latest batch |
| `make <name>` | `create`, `new` | No | Create paired migration files |
| `help` | `-h`, `--help` | No | Show usage |

## Migration Files

Migration files are paired by version and name:

```text
migrations/
├── 20260627120000_create_users_table.up.sql
└── 20260627120000_create_users_table.down.sql
```

Rules:

- Version must be 14 digits: `YYYYMMDDHHMMSS`.
- Both `.up.sql` and `.down.sql` must exist.
- Do not edit an applied `.up.sql` file. Add a new migration instead.

## Safety Rules

- Migrations run in transactions.
- Applied `.up.sql` files are checksum validated on `up`, `status`, and `rollback`.
- Rollback fails if the latest applied batch has no matching local migration files.
- The tracking table name is validated before use.
- Generated migration files never overwrite existing files.

## Demo App

The `test/` module includes a small migration CLI and API.

Set up a PostgreSQL URL in `test/.env`:

```sh
cd test
cp .env.example .env
```

Then run:

```sh
go run ./cmd/migrate status
go run ./cmd/migrate up
go run ./cmd/api
```

## Tests

Core library tests, no database required:

```sh
go test ./migrator/...
```

Integration tests, database required through `test/.env` or `DATABASE_URL`:

```sh
go test ./test/...
```

Everything from the repo root:

```sh
go test ./migrator/... ./test/...
```

## Contributing

See [CONTRIBUTING.md](./migrator/CONTRIBUTING.md).

## License

MIT
