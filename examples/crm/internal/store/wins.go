package store

import (
	"context"
	"errors"
)

// RecordWin inserts the deal_wins row of a won deal inside the transaction,
// under a savepoint. ctx bounds the insert (the engine's closeTimeout
// deadline). On any error, the insert cancelled by that deadline included,
// it rolls back to the savepoint and returns the error, so the transaction
// stays usable for the audit row that describes the failure.
func (t *Tx) RecordWin(ctx context.Context, dealID string, amountCents int64) error {
	sp, err := t.tx.Begin(t.ctx)
	if err != nil {
		return err
	}
	if _, err := sp.Exec(ctx, "INSERT INTO deal_wins (deal_id, amount_cents) VALUES ($1::uuid, $2)", dealID, amountCents); err != nil {
		if rerr := sp.Rollback(t.ctx); rerr != nil {
			return errors.Join(err, rerr)
		}
		return err
	}
	return sp.Commit(t.ctx)
}
