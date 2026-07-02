// Package testsupport provides an in-memory fake Repo for the unit tests
// (BUILD.md 7 mock policy: unit tests may mock the single Repo collaborator).
// The go-ladybug-backed repo and its integration tests live behind the
// `ladybug` build tag; this fake keeps the default suite hermetic (no native
// lib, no network). It is honest storage so that GREEN unit tests exercise real
// read-modify-write behavior; per-method error hooks let a test force a typed
// repo error (ErrLocked, ErrConstraint, ...).
package testsupport

import (
	"crm/internal/model"
	"crm/internal/repo"
)

// FakeTx is the in-memory transaction handle.
type FakeTx struct{ open bool }

// Interface assertion: the fake satisfies the Repo boundary.
var _ repo.Repo = (*FakeRepo)(nil)

// FakeRepo is an in-memory repo.Repo.
type FakeRepo struct {
	Users      map[string]model.User
	Deals      map[string]model.Deal
	Tasks      map[string]model.Task
	Accounts   map[string]model.Account
	Contacts   map[string]model.Contact
	Activities map[string]model.Activity
	Pipelines  map[string]model.Pipeline
	Tags       map[string]model.Tag
	Teams      map[string]model.Team

	// Error hooks: when set, the matching method returns this error before any
	// state change. Lets tests drive the ErrLocked/ErrConstraint/... paths.
	OpenErr          error
	SaveDealErr      error
	SaveTaskErr      error
	SaveUserErr      error
	SetDefaultErr    error
	GetUserByNameErr error
	GetUserErr       error

	Committed  bool
	RolledBack bool
}

// NewFakeRepo returns an empty FakeRepo.
func NewFakeRepo() *FakeRepo {
	return &FakeRepo{
		Users:      map[string]model.User{},
		Deals:      map[string]model.Deal{},
		Tasks:      map[string]model.Task{},
		Accounts:   map[string]model.Account{},
		Contacts:   map[string]model.Contact{},
		Activities: map[string]model.Activity{},
		Pipelines:  map[string]model.Pipeline{},
		Tags:       map[string]model.Tag{},
		Teams:      map[string]model.Team{},
	}
}

// Open returns an open FakeTx (or OpenErr if set).
func (f *FakeRepo) Open(path string) (repo.Tx, error) {
	if f.OpenErr != nil {
		return nil, f.OpenErr
	}
	return &FakeTx{open: true}, nil
}

func (f *FakeRepo) BeginWrite(tx repo.Tx) error { return nil }

func (f *FakeRepo) Commit(tx repo.Tx) error { f.Committed = true; return nil }

func (f *FakeRepo) Rollback(tx repo.Tx) error { f.RolledBack = true; return nil }

// --- Reads ---

func (f *FakeRepo) GetUserByName(tx repo.Tx, name string) (model.User, error) {
	if f.GetUserByNameErr != nil {
		return model.User{}, f.GetUserByNameErr
	}
	for _, u := range f.Users {
		if u.Username == name {
			return u, nil
		}
	}
	return model.User{}, model.ErrNotFound
}

func (f *FakeRepo) GetUser(tx repo.Tx, id string) (model.User, error) {
	if f.GetUserErr != nil {
		return model.User{}, f.GetUserErr
	}
	if u, ok := f.Users[id]; ok {
		return u, nil
	}
	return model.User{}, model.ErrNotFound
}

func (f *FakeRepo) GetDeal(tx repo.Tx, id string) (model.Deal, error) {
	if d, ok := f.Deals[id]; ok {
		return d, nil
	}
	return model.Deal{}, model.ErrNotFound
}

func (f *FakeRepo) GetTask(tx repo.Tx, id string) (model.Task, error) {
	if t, ok := f.Tasks[id]; ok {
		return t, nil
	}
	return model.Task{}, model.ErrNotFound
}

func (f *FakeRepo) GetAccount(tx repo.Tx, id string) (model.Account, error) {
	if a, ok := f.Accounts[id]; ok {
		return a, nil
	}
	return model.Account{}, model.ErrNotFound
}

func (f *FakeRepo) GetContact(tx repo.Tx, id string) (model.Contact, error) {
	if c, ok := f.Contacts[id]; ok {
		return c, nil
	}
	return model.Contact{}, model.ErrNotFound
}

// --- Writes ---

func (f *FakeRepo) SaveDeal(tx repo.Tx, d model.Deal) error {
	if f.SaveDealErr != nil {
		return f.SaveDealErr
	}
	f.Deals[d.ID] = d
	return nil
}

func (f *FakeRepo) SaveTask(tx repo.Tx, t model.Task) error {
	if f.SaveTaskErr != nil {
		return f.SaveTaskErr
	}
	f.Tasks[t.ID] = t
	return nil
}

func (f *FakeRepo) SaveUser(tx repo.Tx, u model.User) error {
	if f.SaveUserErr != nil {
		return f.SaveUserErr
	}
	for id, ex := range f.Users {
		if ex.Username == u.Username && id != u.ID {
			return model.ErrConstraint // username-unique
		}
	}
	f.Users[u.ID] = u
	return nil
}

func (f *FakeRepo) SaveAccount(tx repo.Tx, a model.Account) error {
	f.Accounts[a.ID] = a
	return nil
}

func (f *FakeRepo) SaveContact(tx repo.Tx, c model.Contact) error {
	f.Contacts[c.ID] = c
	return nil
}

func (f *FakeRepo) SaveActivity(tx repo.Tx, a model.Activity) error {
	if _, exists := f.Activities[a.ID]; exists {
		// Append-only: an existing activity is never overwritten in place
		// (activity-immutable). A real repo rejects an id collision.
		return model.ErrConstraint
	}
	f.Activities[a.ID] = a
	return nil
}

func (f *FakeRepo) SavePipeline(tx repo.Tx, p model.Pipeline) error {
	f.Pipelines[p.ID] = p
	return nil
}

func (f *FakeRepo) SetDefaultPipeline(tx repo.Tx, id string) error {
	if f.SetDefaultErr != nil {
		return f.SetDefaultErr
	}
	if _, ok := f.Pipelines[id]; !ok {
		return model.ErrNotFound
	}
	for pid, p := range f.Pipelines {
		p.IsDefault = pid == id
		f.Pipelines[pid] = p
	}
	return nil
}

func (f *FakeRepo) SaveTag(tx repo.Tx, t model.Tag) error {
	for id, ex := range f.Tags {
		if ex.Name == t.Name && id != t.ID {
			return model.ErrConstraint // tag-name-unique
		}
	}
	f.Tags[t.ID] = t
	return nil
}

func (f *FakeRepo) SaveTeam(tx repo.Tx, t model.Team) error {
	for id, ex := range f.Teams {
		if ex.Name == t.Name && id != t.ID {
			return model.ErrConstraint // team-name-unique
		}
	}
	f.Teams[t.ID] = t
	return nil
}

func (f *FakeRepo) CountDefaultPipelines(tx repo.Tx) (int, error) {
	n := 0
	for _, p := range f.Pipelines {
		if p.IsDefault {
			n++
		}
	}
	return n, nil
}
