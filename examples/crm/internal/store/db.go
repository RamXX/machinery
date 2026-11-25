// Package store owns the database: the connection pool, the migrations and
// the health probe.
package store

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgconn/ctxwatch"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Timeouts. Every database call of the service runs under Timeout (a
// transaction under one Timeout as a whole). A call whose context ends is
// cancelled on the server, and the connection is given up CancelGrace
// later if the server does not answer the cancel: so a database that does
// not answer costs a request at most Timeout plus CancelGrace.
const (
	Timeout       = 2 * time.Second
	CancelGrace   = 500 * time.Millisecond
	HealthTimeout = Timeout
)

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
	cfg.ConnConfig.ConnectTimeout = Timeout
	// A cancel request, not a closed connection: a statement cancelled by
	// its deadline leaves the transaction usable after ROLLBACK TO SAVEPOINT.
	cfg.ConnConfig.BuildContextWatcherHandler = func(c *pgconn.PgConn) ctxwatch.Handler {
		return &pgconn.CancelRequestContextWatcherHandler{Conn: c, DeadlineDelay: CancelGrace}
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	return &Store{pool: pool}, nil
}

// bound gives a database call its Timeout.
func bound(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, Timeout)
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

// Health answers nil when SELECT 1 answers within HealthTimeout and the
// migrations are applied.
func (s *Store) Health(ctx context.Context) error {
	ctx, cancel := bound(ctx)
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
