package model

import "errors"

// Typed domain errors. The repo maps go-ladybug driver errors to these
// (BUILD.md 4.6, 9). Session and CommandExecution classify on them via
// errors.Is. Compared with errors.Is so wrapped errors still match.
var (
	// Repo / store errors (BUILD.md 4.6).
	ErrLocked      = errors.New("crm: store locked")
	ErrCorrupt     = errors.New("crm: store corrupt")
	ErrUnavailable = errors.New("crm: store unavailable")
	ErrNotFound    = errors.New("crm: not found")
	ErrConstraint  = errors.New("crm: constraint violation")
	ErrConflict    = errors.New("crm: write conflict")
	ErrDiskFull    = errors.New("crm: disk full")
	ErrTimeout     = errors.New("crm: timeout")

	// Session / auth errors (BUILD.md 4.6, 5.4).
	ErrBadCredentials = errors.New("crm: bad credentials")
	ErrDisabled       = errors.New("crm: user disabled")
	ErrNoSession      = errors.New("crm: no session")
	ErrExpired        = errors.New("crm: session expired")
	ErrUnreadable     = errors.New("crm: token unreadable")

	// ErrNotImplemented is returned by the test-writer scaffolding stubs so the
	// suite compiles and runs to completion (every logic test fails on its
	// assertions, RED, rather than crashing the package with a panic). The
	// implementer replaces every stub; this sentinel then disappears from the
	// exercised paths.
	ErrNotImplemented = errors.New("crm: not implemented")
)
