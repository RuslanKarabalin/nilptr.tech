// Package migrate applies the embedded SQL migrations.
package migrate

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed migrations/*.sql
var files embed.FS

// lockID is the advisory lock key that serializes concurrent runs.
const lockID = 7_368_756_900

// Migration is one SQL file; Version is the file name without ".sql".
type Migration struct {
	Version string
	SQL     string
}

// List returns the embedded migrations sorted by version.
func List() ([]Migration, error) {
	entries, err := fs.ReadDir(files, "migrations")
	if err != nil {
		return nil, err
	}
	var out []Migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		b, err := files.ReadFile(path.Join("migrations", e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, Migration{Version: strings.TrimSuffix(e.Name(), ".sql"), SQL: string(b)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

// Run applies all pending migrations, each in its own transaction.
func Run(ctx context.Context, conn *pgx.Conn, log *slog.Logger) error {
	migrations, err := List()
	if err != nil {
		return err
	}
	if _, err := conn.Exec(ctx, `select pg_advisory_lock($1)`, lockID); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer func() { _, _ = conn.Exec(context.WithoutCancel(ctx), `select pg_advisory_unlock($1)`, lockID) }()

	if _, err := conn.Exec(ctx, `
		create table if not exists schema_migrations (
			version    text primary key,
			applied_at timestamptz not null default now()
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	rows, err := conn.Query(ctx, `select version from schema_migrations`)
	if err != nil {
		return err
	}
	applied, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	done := make(map[string]bool, len(applied))
	for _, v := range applied {
		done[v] = true
	}

	for _, m := range migrations {
		if done[m.Version] {
			continue
		}
		err := pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, m.SQL); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `insert into schema_migrations (version) values ($1)`, m.Version)
			return err
		})
		if err != nil {
			return fmt.Errorf("migration %s: %w", m.Version, err)
		}
		log.Info("migration applied", "version", m.Version)
	}
	return nil
}
