// Package repo is the only component that may import go-ladybug (BUILD.md 4.5,
// enforced by C-ARCH-01). This file holds the boundary interfaces (Repo, Tx)
// from BUILD.md 4.6; the go-ladybug-backed implementation lives in
// repo_ladybug.go behind the `ladybug` build tag so the default build needs no
// native library and no network.
package repo

import "crm/internal/model"

// Tx is an opaque open write transaction handle (BUILD.md 4.6). It is an empty
// interface so any concrete driver handle (the go-ladybug transaction, or the
// in-memory fake) satisfies it; an unexported marker method would make the
// interface unimplementable outside this package.
type Tx interface{}

// Repo is the persistence boundary (BUILD.md 4.6). Only repo imports go-ladybug.
// The typed errors it returns are the model.Err* sentinels.
//
// NOTE ON activity-immutable (C-REPO-23): there is deliberately NO UpdateActivity
// / mutate-activity method. SaveActivity is append-only. The absence of a
// mutation method is the structural enforcement of activity-immutable and is
// asserted by C-REPO-23.
type Repo interface {
	// Lifecycle.
	Open(path string) (Tx, error) // ErrLocked, ErrCorrupt, ErrUnavailable
	BeginWrite(tx Tx) error       // Cypher BEGIN TRANSACTION
	Commit(tx Tx) error
	Rollback(tx Tx) error // idempotent; store guarantees no partial write

	// Reads (safe to retry).
	GetUserByName(tx Tx, name string) (model.User, error) // ErrNotFound
	GetUser(tx Tx, id string) (model.User, error)         // ErrNotFound
	GetDeal(tx Tx, id string) (model.Deal, error)         // ErrNotFound
	GetTask(tx Tx, id string) (model.Task, error)         // ErrNotFound
	GetAccount(tx Tx, id string) (model.Account, error)   // ErrNotFound
	GetContact(tx Tx, id string) (model.Contact, error)   // ErrNotFound

	// Writes (retried only on ErrLocked; the Tx never partially commits).
	SaveDeal(tx Tx, d model.Deal) error         // ErrConstraint,ErrConflict,ErrDiskFull,ErrTimeout,ErrLocked
	SaveTask(tx Tx, t model.Task) error         // same error set
	SaveUser(tx Tx, u model.User) error         // ErrConstraint (username-unique) ...
	SaveAccount(tx Tx, a model.Account) error   // ...
	SaveContact(tx Tx, c model.Contact) error   // ...
	SaveActivity(tx Tx, a model.Activity) error // append-only; no update path (activity-immutable)
	SavePipeline(tx Tx, p model.Pipeline) error
	SetDefaultPipeline(tx Tx, id string) error // atomic read-modify-write (one-default-pipeline)
	SaveTag(tx Tx, t model.Tag) error          // ErrConstraint (tag-name-unique)
	SaveTeam(tx Tx, t model.Team) error        // ErrConstraint (team-name-unique)

	// CountDefaultPipelines supports the P-one-default-pipeline post-condition
	// check (count(isDefault==true)==1).
	CountDefaultPipelines(tx Tx) (int, error)
}
