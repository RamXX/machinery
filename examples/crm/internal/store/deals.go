package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Deal is a sales opportunity.
type Deal struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Stage       string `json:"stage"`
	AmountCents int64  `json:"amountCents"`
	OwnerID     string `json:"ownerId"`
}

const dealCols = "id::text, title, stage, amount_cents, owner_id::text"

func scanDeal(r pgx.Row) (Deal, error) {
	var d Deal
	err := r.Scan(&d.ID, &d.Title, &d.Stage, &d.AmountCents, &d.OwnerID)
	return d, classify(err)
}

// CreateDeal inserts a deal.
func (s *Store) CreateDeal(ctx context.Context, d Deal) (Deal, error) {
	ctx, cancel := bound(ctx)
	defer cancel()
	return scanDeal(s.pool.QueryRow(ctx, `INSERT INTO deals (title, stage, amount_cents, owner_id)
		VALUES ($1, $2, $3, $4::uuid) RETURNING `+dealCols, d.Title, d.Stage, d.AmountCents, d.OwnerID))
}

// ListDeals returns every deal, ordered by id.
func (s *Store) ListDeals(ctx context.Context) ([]Deal, error) {
	ctx, cancel := bound(ctx)
	defer cancel()
	rows, err := s.pool.Query(ctx, "SELECT "+dealCols+" FROM deals ORDER BY id")
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Deal, error) { return scanDeal(r) })
}

// LockDeal loads a deal FOR UPDATE inside the transaction.
func (t *Tx) LockDeal(id string) (Deal, error) {
	return scanDeal(t.tx.QueryRow(t.ctx, "SELECT "+dealCols+" FROM deals WHERE id = $1::uuid FOR UPDATE", id))
}

// SetDealStage stores the new stage of a locked deal, inside the transition
// path of the case that fired.
func (t *Tx) SetDealStage(id, stage, caseID string) error {
	if err := t.transition(caseID); err != nil {
		return err
	}
	_, err := t.tx.Exec(t.ctx, "UPDATE deals SET stage = $2 WHERE id = $1::uuid", id, stage)
	return err
}

// GetDeal loads a deal; an unknown id is ErrNotFound.
func (s *Store) GetDeal(ctx context.Context, id string) (Deal, error) {
	ctx, cancel := bound(ctx)
	defer cancel()
	return scanDeal(s.pool.QueryRow(ctx, "SELECT "+dealCols+" FROM deals WHERE id = $1::uuid", id))
}
