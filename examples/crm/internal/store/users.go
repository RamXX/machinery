package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// User is a person who acts on deals and tasks.
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// CreateUser inserts a user; a taken email is ErrEmailTaken.
func (s *Store) CreateUser(ctx context.Context, email, role string) (User, error) {
	u := User{Email: email, Role: role}
	err := s.pool.QueryRow(ctx, "INSERT INTO users (email, role) VALUES ($1, $2) RETURNING id::text", email, role).Scan(&u.ID)
	return u, classify(err)
}

// GetUser loads a user by id; an unknown id is ErrNotFound.
func (s *Store) GetUser(ctx context.Context, id string) (User, error) {
	u := User{ID: id}
	err := s.pool.QueryRow(ctx, "SELECT email, role FROM users WHERE id = $1::uuid", id).Scan(&u.Email, &u.Role)
	return u, classify(err)
}

// ListUsers returns every user, ordered by email.
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, "SELECT id::text, email, role FROM users ORDER BY email")
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (User, error) {
		var u User
		err := r.Scan(&u.ID, &u.Email, &u.Role)
		return u, err
	})
}
