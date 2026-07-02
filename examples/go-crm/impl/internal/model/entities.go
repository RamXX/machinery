package model

import "time"

// The nine record entities from BUILD.md 3.3. Every entity has a synthetic
// string ID (primary key). Owned entities carry an OwnerID referencing the
// owning User.

// User is the operator record (BUILD.md 3.3). PasswordHash holds only an
// argon2id encoded hash (invariant password-hashed). Status is the User
// aggregate state.
type User struct {
	ID           string
	Username     string
	PasswordHash string
	Role         UserRole
	Status       UserStatus
	CreatedAt    time.Time
	TeamID       string // at most one team (invariant single-team)
}

// Team groups Users for visibility scope (BUILD.md 3.3).
type Team struct {
	ID   string
	Name string
}

// Account groups Contacts and Deals (BUILD.md 3.3).
type Account struct {
	ID       string
	Name     string
	Domain   string
	Industry string
	OwnerID  string
}

// Contact is a person record (BUILD.md 3.3).
type Contact struct {
	ID       string
	FullName string
	Email    string
	Phone    string
	Title    string
	OwnerID  string
}

// Deal is a sales opportunity; Stage is the Deal aggregate state (BUILD.md 3.3).
type Deal struct {
	ID          string
	Title       string
	AmountCents int64 // >= 0 (invariant deal-amount-nonneg)
	Stage       DealStage
	CloseDate   *time.Time // required when Stage == Won (deal-won-has-closedate)
	OwnerID     string
	ContactIDs  []string
}

// Pipeline is a namespace for Deals; exactly one is default (BUILD.md 3.3).
type Pipeline struct {
	ID        string
	Name      string
	IsDefault bool
}

// Activity is an append-only log entry; Body and OccurredAt are immutable after
// create (invariant activity-immutable) (BUILD.md 3.3).
type Activity struct {
	ID         string
	Type       ActivityType
	Subject    string
	Body       string
	OccurredAt time.Time
	OwnerID    string
	ContactID  string
}

// Task is a to-do; Status is the Task aggregate state (BUILD.md 3.3).
type Task struct {
	ID      string
	Title   string
	DueDate *time.Time
	Status  TaskStatus
	OwnerID string
	DealID  string // optional link
}

// Tag is a freeform label (BUILD.md 3.3).
type Tag struct {
	ID    string
	Name  string
	Color string
}

// Session is the local expiring credential from the glossary (BUILD.md 2). It
// is not a Modelith entity; it identifies the acting User for later commands.
type Session struct {
	UserID    string
	ExpiresAt time.Time
}
