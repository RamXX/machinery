//go:build ladybug

// This file is the ONLY importer of go-ladybug (BUILD.md 4.5, 9; verified by
// C-ARCH-01). It is behind the `ladybug` build tag so the default build and the
// default `go test ./...` need no native library and no network. go-ladybug is
// cgo (liblbug), so building with `-tags ladybug` also requires the native
// library and `go get github.com/LadybugDB/go-ladybug@v0.17.0`.
//
// SCAFFOLDING: the constructor wires the driver; the data methods are the
// implementer's job (translate domain reads/writes to parameterized Cypher and
// map driver errors to the model.Err* typed errors, applying SetTimeout(10s) +
// Interrupt per BUILD.md 9). Bodies panic until implemented.

package repo

import (
	lbug "github.com/LadybugDB/go-ladybug"

	"crm/internal/model"
)

// ladybugRepo is the embedded LadybugDB-backed Repo.
type ladybugRepo struct{}

// ladybugTx carries the open database + connection for one write transaction.
type ladybugTx struct {
	db   *lbug.Database
	conn *lbug.Connection
}

// NewLadybug constructs the go-ladybug-backed Repo.
func NewLadybug() Repo { return &ladybugRepo{} }

// mapErr maps a go-ladybug driver error to a typed model error (implementer).
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	panic("not implemented: map go-ladybug driver error to model.Err*")
}

// Open opens the database for writing at path (BUILD.md 4.6). ErrLocked,
// ErrCorrupt, ErrUnavailable.
func (r *ladybugRepo) Open(path string) (Tx, error) {
	db, err := lbug.OpenDatabase(path, lbug.DefaultSystemConfig())
	if err != nil {
		return nil, mapErr(err)
	}
	conn, err := lbug.OpenConnection(db)
	if err != nil {
		return nil, mapErr(err)
	}
	return &ladybugTx{db: db, conn: conn}, nil
}

func (r *ladybugRepo) BeginWrite(tx Tx) error { panic("not implemented") }
func (r *ladybugRepo) Commit(tx Tx) error     { panic("not implemented") }
func (r *ladybugRepo) Rollback(tx Tx) error   { panic("not implemented") }

func (r *ladybugRepo) GetUserByName(tx Tx, name string) (model.User, error) { panic("not implemented") }
func (r *ladybugRepo) GetUser(tx Tx, id string) (model.User, error)         { panic("not implemented") }
func (r *ladybugRepo) GetDeal(tx Tx, id string) (model.Deal, error)         { panic("not implemented") }
func (r *ladybugRepo) GetTask(tx Tx, id string) (model.Task, error)         { panic("not implemented") }
func (r *ladybugRepo) GetAccount(tx Tx, id string) (model.Account, error)   { panic("not implemented") }
func (r *ladybugRepo) GetContact(tx Tx, id string) (model.Contact, error)   { panic("not implemented") }

func (r *ladybugRepo) SaveDeal(tx Tx, d model.Deal) error         { panic("not implemented") }
func (r *ladybugRepo) SaveTask(tx Tx, t model.Task) error         { panic("not implemented") }
func (r *ladybugRepo) SaveUser(tx Tx, u model.User) error         { panic("not implemented") }
func (r *ladybugRepo) SaveAccount(tx Tx, a model.Account) error   { panic("not implemented") }
func (r *ladybugRepo) SaveContact(tx Tx, c model.Contact) error   { panic("not implemented") }
func (r *ladybugRepo) SaveActivity(tx Tx, a model.Activity) error { panic("not implemented") }
func (r *ladybugRepo) SavePipeline(tx Tx, p model.Pipeline) error { panic("not implemented") }
func (r *ladybugRepo) SetDefaultPipeline(tx Tx, id string) error  { panic("not implemented") }
func (r *ladybugRepo) SaveTag(tx Tx, t model.Tag) error           { panic("not implemented") }
func (r *ladybugRepo) SaveTeam(tx Tx, t model.Team) error         { panic("not implemented") }

func (r *ladybugRepo) CountDefaultPipelines(tx Tx) (int, error) { panic("not implemented") }
