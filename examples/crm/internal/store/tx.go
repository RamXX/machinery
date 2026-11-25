package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Errors the API maps to answers.
var (
	ErrNotFound     = errors.New("not found")
	ErrEmailTaken   = errors.New("email already taken")
	ErrBadReference = errors.New("referenced record does not exist")
)

// Tx is one database transaction of a request.
type Tx struct {
	ctx context.Context
	tx  pgx.Tx
}

// Context is the context the transaction runs under.
func (t *Tx) Context() context.Context { return t.ctx }

// InTx runs fn in one transaction and commits when fn returns nil.
func (s *Store) InTx(ctx context.Context, fn func(*Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(context.Background())
	if err := fn(&Tx{ctx: ctx, tx: tx}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// classify maps driver errors to the errors above.
func classify(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		switch pe.Code {
		case "23505":
			return ErrEmailTaken
		case "23503":
			return ErrBadReference
		}
	}
	return err
}
