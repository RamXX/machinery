// Package store owns the database: the connection pool, the migrations and
// the health probe.
package store

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthTimeout bounds the health probe.
const HealthTimeout = 2 * time.Second

// ErrNotMigrated means the migrations have not been applied yet.
var ErrNotMigrated = errors.New("migrations not applied yet")

// Store is the service's handle on the database, connected as the app role.
type Store struct {
	pool     *pgxpool.Pool
	migrated atomic.Bool
}

// Open creates the pool. It does not connect: the database may come later.
func Open(url string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

// Health answers nil when SELECT 1 answers within HealthTimeout and the
// migrations are applied.
func (s *Store) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, HealthTimeout)
	defer cancel()
	var one int
	if err := s.pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		return fmt.Errorf("database not answering: %w", err)
	}
	if !s.migrated.Load() {
		return ErrNotMigrated
	}
	return nil
}
