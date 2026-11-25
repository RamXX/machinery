package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Lifecycle is what the database lets the app role do to the state column
// of one machine's records: create them in Initial, move them along Edges
// (pairs of source and target state).
type Lifecycle struct {
	Machine string
	Initial string
	Edges   [][2]string
}

// transition marks the transaction as the transition path of a case: the
// lifecycle trigger lets the app role change a state column only when
// crm.transition is set. Only the save functions call it, immediately
// before their update.
func (t *Tx) transition(caseID string) error {
	_, err := t.tx.Exec(t.ctx, "SELECT set_config('crm.transition', $1, true)", caseID)
	return err
}

// syncLifecycles replaces the content of lifecycle_edges with the given
// lifecycles in one transaction, as the admin role. An initial state is the
// row with an empty source.
func syncLifecycles(ctx context.Context, conn *pgx.Conn, ls []Lifecycle) error {
	return pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", migrateLock); err != nil {
			return fmt.Errorf("lock lifecycle_edges: %w", err)
		}
		if _, err := tx.Exec(ctx, "DELETE FROM lifecycle_edges"); err != nil {
			return fmt.Errorf("clear lifecycle_edges: %w", err)
		}
		for _, l := range ls {
			rows := append([][2]string{{"", l.Initial}}, l.Edges...)
			for _, e := range rows {
				if _, err := tx.Exec(ctx, "INSERT INTO lifecycle_edges (machine, source, target) VALUES ($1, $2, $3)", l.Machine, e[0], e[1]); err != nil {
					return fmt.Errorf("lifecycle_edges %s %s -> %s: %w", l.Machine, e[0], e[1], err)
				}
			}
		}
		return nil
	})
}
