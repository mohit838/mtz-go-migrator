package migration

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Runner struct {
	db          *sql.DB
	dir         string
	tableName   string
	serviceName string
	writer      io.Writer
	now         func() time.Time
}

type Config struct {
	Dir         string
	TableName   string
	ServiceName string
	Writer      io.Writer
	Now         func() time.Time
}

type fileMigration struct {
	version  string
	name     string
	upPath   string
	downPath string
	checksum string
}

type appliedMigration struct {
	version  string
	name     string
	batch    int
	checksum string
}

func NewRunner(db *sql.DB, cfg Config) *Runner {
	if cfg.Dir == "" {
		cfg.Dir = "migrations"
	}
	if cfg.TableName == "" {
		cfg.TableName = "schema_migrations"
	}
	if cfg.Writer == nil {
		cfg.Writer = os.Stdout
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Runner{
		db:          db,
		dir:         cfg.Dir,
		tableName:   cfg.TableName,
		serviceName: cfg.ServiceName,
		writer:      cfg.Writer,
		now:         cfg.Now,
	}
}

func (r *Runner) Up(ctx context.Context) error {
	if err := r.ensureStore(ctx); err != nil {
		return err
	}

	files, err := r.loadFiles()
	if err != nil {
		return err
	}
	applied, err := r.applied(ctx)
	if err != nil {
		return err
	}
	if err := validateChecksums(files, applied); err != nil {
		return err
	}

	batch := latestBatch(applied) + 1
	ran := 0
	for _, file := range files {
		if _, ok := applied[file.version]; ok {
			continue
		}
		if err := r.runUp(ctx, file, batch); err != nil {
			return err
		}
		ran++
	}
	if ran == 0 {
		r.println("Nothing to migrate.")
		return nil
	}
	r.printf("%sMigrated %d migration(s).\n", r.logPrefix(), ran)
	return nil
}

func (r *Runner) Rollback(ctx context.Context) error {
	if err := r.ensureStore(ctx); err != nil {
		return err
	}

	files, err := r.loadFiles()
	if err != nil {
		return err
	}
	applied, err := r.applied(ctx)
	if err != nil {
		return err
	}
	if len(applied) == 0 {
		r.println("Nothing to rollback.")
		return nil
	}

	batch := latestBatch(applied)
	toRollback := make([]fileMigration, 0)
	for _, file := range files {
		if appliedFile, ok := applied[file.version]; ok && appliedFile.batch == batch {
			toRollback = append(toRollback, file)
		}
	}
	sort.Slice(toRollback, func(i, j int) bool {
		return toRollback[i].version > toRollback[j].version
	})

	for _, file := range toRollback {
		if err := r.runDown(ctx, file); err != nil {
			return err
		}
	}
	r.printf("%sRolled back %d migration(s) from batch %d.\n", r.logPrefix(), len(toRollback), batch)
	return nil
}

func (r *Runner) Status(ctx context.Context) error {
	if err := r.ensureStore(ctx); err != nil {
		return err
	}

	files, err := r.loadFiles()
	if err != nil {
		return err
	}
	applied, err := r.applied(ctx)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		r.println("No migration files found.")
		return nil
	}
	for _, file := range files {
		state := "pending"
		if appliedFile, ok := applied[file.version]; ok {
			state = fmt.Sprintf("ran batch=%d", appliedFile.batch)
		}
		r.printf("%s%s %-8s %s\n", r.logPrefix(), file.version, state, file.name)
	}
	return nil
}

func (r *Runner) ensureStore(ctx context.Context) error {
	if r.db == nil {
		return fmt.Errorf("migration db is nil")
	}
	if !safeIdentifier(r.tableName) {
		return fmt.Errorf("invalid migration table name: %s", r.tableName)
	}
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id BIGSERIAL PRIMARY KEY,
	version TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	service_name TEXT NOT NULL DEFAULT '',
	batch INTEGER NOT NULL,
	checksum TEXT NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	execution_ms BIGINT NOT NULL DEFAULT 0
)`, r.tableName))
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS service_name TEXT NOT NULL DEFAULT ''`, r.tableName))
	return err
}

func (r *Runner) runUp(ctx context.Context, file fileMigration, batch int) error {
	sqlBody, err := os.ReadFile(file.upPath)
	if err != nil {
		return err
	}

	start := time.Now()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, string(sqlBody)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("run %s: %w", file.upPath, err)
	}
	executionMS := time.Since(start).Milliseconds()
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
INSERT INTO %s (version, name, service_name, batch, checksum, execution_ms)
VALUES ($1, $2, $3, $4, $5, $6)`, r.tableName), file.version, file.name, r.serviceName, batch, file.checksum, executionMS); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	r.println(r.logPrefix()+"Migrated", filepath.Base(file.upPath))
	return nil
}

func (r *Runner) runDown(ctx context.Context, file fileMigration) error {
	sqlBody, err := os.ReadFile(file.downPath)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, string(sqlBody)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("rollback %s: %w", file.downPath, err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE version = $1`, r.tableName), file.version); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	r.println(r.logPrefix()+"Rolled back", filepath.Base(file.downPath))
	return nil
}

func (r *Runner) applied(ctx context.Context) (map[string]appliedMigration, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT version, name, batch, checksum FROM %s ORDER BY version`, r.tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]appliedMigration)
	for rows.Next() {
		var item appliedMigration
		if err := rows.Scan(&item.version, &item.name, &item.batch, &item.checksum); err != nil {
			return nil, err
		}
		applied[item.version] = item
	}
	return applied, rows.Err()
}

func (r *Runner) logPrefix() string {
	if r.serviceName == "" {
		return ""
	}
	return "[" + r.serviceName + "] "
}

func (r *Runner) printf(format string, args ...any) {
	fmt.Fprintf(r.writer, format, args...)
}

func (r *Runner) println(args ...any) {
	fmt.Fprintln(r.writer, args...)
}

func safeIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, char := range value {
		if char == '_' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || i > 0 && char >= '0' && char <= '9' {
			continue
		}
		return false
	}
	return true
}

func validateChecksums(files []fileMigration, applied map[string]appliedMigration) error {
	for _, file := range files {
		appliedFile, ok := applied[file.version]
		if ok && appliedFile.checksum != file.checksum {
			return fmt.Errorf("migration checksum changed after apply: %s_%s", file.version, file.name)
		}
	}
	return nil
}

func latestBatch(applied map[string]appliedMigration) int {
	latest := 0
	for _, item := range applied {
		if item.batch > latest {
			latest = item.batch
		}
	}
	return latest
}
