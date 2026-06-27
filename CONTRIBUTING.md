# Contributing to mtz-go-migrator

Thank you for considering contributing! This document explains how to set up your environment, run tests, and submit changes.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Development Setup](#development-setup)
- [Running Tests](#running-tests)
- [Project Structure](#project-structure)
- [How to Add a New Command](#how-to-add-a-new-command)
- [Submitting a Pull Request](#submitting-a-pull-request)
- [Reporting Issues](#reporting-issues)
- [Code Style](#code-style)

---

## Code of Conduct

Be respectful. Everyone is welcome regardless of background or experience level.
Keep discussions focused on technical merit. Constructive criticism is fine; personal attacks are not.

---

## Development Setup

**Requirements:**
- Go 1.21 or later
- PostgreSQL (for integration tests)
- Git

**Clone the repo:**

```sh
git clone https://github.com/mohit838/mtz-go-migrator.git
cd mtz-go-migrator
```

**The library lives in:**

```
migrator/migration/
```

**The test app lives in:**

```
test/
```

---

## Running Tests

### Unit tests (no database needed)

```sh
cd migrator
go test ./...
```

### Integration tests

Integration tests require a running PostgreSQL instance.
Set `DATABASE_URL` before running:

```sh
export DATABASE_URL=postgres://user:password@localhost:5432/testdb?sslmode=disable
cd migrator
go test ./... -tags integration
```

> Tests use a unique table name per run so they don't interfere with your existing schema.

### Test the CLI manually (no database needed)

```sh
cd test
go run ./cmd/migrate help
go run ./cmd/migrate make my_test_migration
```

### Build check

```sh
cd test
go build ./...
```

---

## Project Structure

```
mtz-go-migrator/
├── migrator/
│   ├── migration/
│   │   ├── runner.go          ← Up, Rollback, Status, ensureStore
│   │   ├── command.go         ← Run dispatcher, NeedsDatabase, Usage
│   │   ├── make.go            ← Make (file generation)
│   │   ├── files.go           ← loadFiles, file parsing
│   │   ├── checksum.go        ← SHA-256 checksum helper
│   │   ├── doc.go             ← Package documentation
│   │   ├── runner_test.go
│   │   ├── command_test.go
│   │   ├── make_test.go
│   │   └── files_test.go
│   ├── README.md
│   ├── CONTRIBUTING.md
│   └── go.mod
└── test/                      ← Test app that consumes the library
    ├── cmd/
    │   ├── api/main.go            ← HTTP server (health checks)
    │   └── migrate/main.go        ← Migrator CLI entry point
    ├── internal/
    │   ├── config/
    │   ├── constants/
    │   ├── database/
    │   ├── response/
    │   └── router/
    ├── migrations/                ← Demo migration files
    └── go.mod
```

---

## How to Add a New Command

1. **Add the case to `command.go`** in the `Run` switch:

```go
case "your-command":
    return r.YourCommand(ctx, args[1:])
```

2. **Update `NeedsDatabase`** if your command doesn't require a DB:

```go
case "your-command":
    return false
```

3. **Update `Usage()`** to include the new command in the help text.

4. **Implement the method** on `*Runner` in a new or existing `.go` file.

5. **Write tests** — unit tests in `*_test.go` files alongside the implementation.

---

## Submitting a Pull Request

1. **Fork** the repository on GitHub
2. **Create a branch** from `main`:
   ```sh
   git checkout -b feature/your-feature-name
   ```
3. **Make your changes** — keep commits focused and atomic
4. **Run tests** — all tests must pass before submitting:
   ```sh
   cd migrator && go test ./...
   cd test && go build ./...
   ```
5. **Open a PR** against `main` with:
   - A clear title describing the change
   - A description of *why* the change is needed
   - Links to any related issues

### PR Checklist

- [ ] All existing tests pass
- [ ] New functionality has tests
- [ ] `go vet ./...` reports no issues
- [ ] No new external dependencies added (the library is dependency-free)
- [ ] `README.md` updated if the public API changed
- [ ] `CONTRIBUTING.md` updated if the development workflow changed

---

## Reporting Issues

Open a GitHub issue and include:

- **Go version** (`go version`)
- **PostgreSQL version** (if relevant)
- **What you did** — the exact commands or code
- **What you expected**
- **What actually happened** — include the full error output

For security vulnerabilities, please do **not** open a public issue.
Contact the maintainer directly.

---

## Code Style

- Standard Go formatting — run `gofmt -w .` before committing
- Follow existing naming conventions (e.g., unexported types for internal structs)
- Keep the library free of external dependencies — standard library only
- Prefer explicit error returns over panics
- Add a comment to every exported function and type

Run the linter if you have it installed:

```sh
golangci-lint run ./...
```

If you don't have it, `go vet ./...` catches the most common issues.
