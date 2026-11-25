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
	return scanDeal(s.pool.QueryRow(ctx, `INSERT INTO deals (title, stage, amount_cents, owner_id)
		VALUES ($1, $2, $3, $4::uuid) RETURNING `+dealCols, d.Title, d.Stage, d.AmountCents, d.OwnerID))
}

// GetDeal loads a deal; an unknown id is ErrNotFound.
func (s *Store) GetDeal(ctx context.Context, id string) (Deal, error) {
	return scanDeal(s.pool.QueryRow(ctx, "SELECT "+dealCols+" FROM deals WHERE id = $1::uuid", id))
}
