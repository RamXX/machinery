package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// AuditEntry records that one user performed one action, with the outcome.
type AuditEntry struct {
	ID      string    `json:"id"`
	ActorID string    `json:"actorId"`
	Action  string    `json:"action"`
	Subject string    `json:"subject"`
	Outcome string    `json:"outcome"`
	At      time.Time `json:"at"`
}

// ListAudit returns the audit trail, oldest first.
func (s *Store) ListAudit(ctx context.Context) ([]AuditEntry, error) {
	rows, err := s.pool.Query(ctx, "SELECT id::text, actor_id::text, action, subject, outcome, at FROM audit_entries ORDER BY id")
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (AuditEntry, error) {
		var e AuditEntry
		err := r.Scan(&e.ID, &e.ActorID, &e.Action, &e.Subject, &e.Outcome, &e.At)
		return e, err
	})
}
