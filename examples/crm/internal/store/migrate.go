package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed migrations/*.sql
var migrations embed.FS

// migrateLock serializes migrations of several instances on one database.
const migrateLock = 7243

// Migrate applies the embedded migrations in name order, each in its own
// transaction as the admin role, and records them in schema_migrations.
func Migrate(ctx context.Context, adminURL string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return fmt.Errorf("connect as admin: %w", err)
	}
	defer conn.Close(context.Background())
	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name text PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	names, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, name := range names {
		if err := apply(ctx, conn, name); err != nil {
			return err
		}
	}
	return nil
}

func apply(ctx context.Context, conn *pgx.Conn, name string) error {
	sql, err := migrations.ReadFile(name)
	if err != nil {
		return err
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", migrateLock); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}
	var done bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name = $1)", name).Scan(&done); err != nil {
		return err
	}
	if done {
		return nil
	}
	if _, err := tx.Exec(ctx, string(sql)); err != nil {
		return fmt.Errorf("migration %s: %w", name, err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (name) VALUES ($1)", name); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// MigrateUntilDone retries Migrate until it succeeds or ctx ends, then marks
// the store ready. A database that comes up late is picked up.
func (s *Store) MigrateUntilDone(ctx context.Context, adminURL string) {
	last := ""
	for {
		err := Migrate(ctx, adminURL)
		if err == nil {
			s.migrated.Store(true)
			log.Print("crm: migrations applied")
			return
		}
		if msg := err.Error(); msg != last {
			log.Printf("crm: migrations not applied, retrying: %v", err)
			last = msg
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}
