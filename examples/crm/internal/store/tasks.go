package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Task is a unit of work on a deal.
type Task struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	DealID     string `json:"dealId"`
	AssigneeID string `json:"assigneeId"`
}

const taskCols = "id::text, title, status, deal_id::text, assignee_id::text"

func scanTask(r pgx.Row) (Task, error) {
	var t Task
	err := r.Scan(&t.ID, &t.Title, &t.Status, &t.DealID, &t.AssigneeID)
	return t, classify(err)
}

// CreateTask inserts a task; an unknown deal is ErrBadReference.
func (s *Store) CreateTask(ctx context.Context, t Task) (Task, error) {
	ctx, cancel := bound(ctx)
	defer cancel()
	return scanTask(s.pool.QueryRow(ctx, `INSERT INTO tasks (title, status, deal_id, assignee_id)
		VALUES ($1, $2, $3::uuid, $4::uuid) RETURNING `+taskCols, t.Title, t.Status, t.DealID, t.AssigneeID))
}

// GetTask loads a task; an unknown id is ErrNotFound.
func (s *Store) GetTask(ctx context.Context, id string) (Task, error) {
	ctx, cancel := bound(ctx)
	defer cancel()
	return scanTask(s.pool.QueryRow(ctx, "SELECT "+taskCols+" FROM tasks WHERE id = $1::uuid", id))
}

// ListTasks returns every task, ordered by id.
func (s *Store) ListTasks(ctx context.Context) ([]Task, error) {
	ctx, cancel := bound(ctx)
	defer cancel()
	rows, err := s.pool.Query(ctx, "SELECT "+taskCols+" FROM tasks ORDER BY id")
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Task, error) { return scanTask(r) })
}

// LockTask loads a task FOR UPDATE inside the transaction.
func (t *Tx) LockTask(id string) (Task, error) {
	return scanTask(t.tx.QueryRow(t.ctx, "SELECT "+taskCols+" FROM tasks WHERE id = $1::uuid FOR UPDATE", id))
}

// SetTaskStatus stores the new status of a locked task.
func (t *Tx) SetTaskStatus(id, status string) error {
	_, err := t.tx.Exec(t.ctx, "UPDATE tasks SET status = $2 WHERE id = $1::uuid", id, status)
	return err
}
