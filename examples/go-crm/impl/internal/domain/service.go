package domain

import (
	"crm/internal/authz"
	"crm/internal/model"
	"crm/internal/repo"
)

// Service holds the pure-record CRUD paths (Account, Contact, Pipeline,
// Activity, Tag, Team) and the cross-record setDefault operation, plus the
// authorization delegation used by the Command Layer (BUILD.md 4.2, 5.6).
//
// It exposes CheckAuthorization so crm.commands can authorize WITHOUT importing
// crm.authz (BUILD.md 4.5 deny: commands->authz; authorization is routed
// commands->domain->authz). Domain imports authz and repo (both allowed).
type Service struct {
	Repo  repo.Repo
	Authz authz.Authorizer
}

// NewService constructs a Service.
func NewService(r repo.Repo, a authz.Authorizer) *Service {
	return &Service{Repo: r, Authz: a}
}

// NewDefaultService constructs a Service wired to the pure authz decision
// function. The command layer uses this so it never has to name crm.authz
// (BUILD.md 4.5 deny: commands->authz).
func NewDefaultService(r repo.Repo) *Service {
	return &Service{Repo: r, Authz: authz.New()}
}

// CheckAuthorization delegates to the pure authz decision and returns only the
// boolean outcome and reason, so the command layer never depends on
// authz.Decision (BUILD.md 4.5, 5.6).
func (s *Service) CheckAuthorization(actor model.User, verb model.Verb, entity model.EntityType, ownerID, teamID string) (allowed bool, reason string) {
	d := s.Authz.Authorize(actor, verb, entity, ownerID, teamID)
	return d.Allowed, d.Reason
}

// --- Pure-record create paths (owner set at create; BUILD.md 5.6). ---

// CreateAccount persists an Account with its owner fixed at create
// (account-owned).
func (s *Service) CreateAccount(tx repo.Tx, a model.Account) (model.Account, error) {
	if err := s.Repo.SaveAccount(tx, a); err != nil {
		return model.Account{}, err
	}
	return a, nil
}

// CreateContact persists a Contact with its owner fixed at create
// (contact-owned).
func (s *Service) CreateContact(tx repo.Tx, c model.Contact) (model.Contact, error) {
	if err := s.Repo.SaveContact(tx, c); err != nil {
		return model.Contact{}, err
	}
	return c, nil
}

// CreateDeal creates a Deal at Lead owned by its creator; a negative amount is
// rejected (deal-owned, deal-amount-nonneg).
func (s *Service) CreateDeal(tx repo.Tx, d model.Deal, actor model.User) (model.Deal, error) {
	if d.AmountCents < 0 {
		return model.Deal{}, model.ErrConstraint // deal-amount-nonneg
	}
	d.OwnerID = actor.ID
	d.Stage = model.StageLead
	if err := s.Repo.SaveDeal(tx, d); err != nil {
		return model.Deal{}, err
	}
	return d, nil
}

// CreateTask creates a Task at Open owned by its creator (task-owned).
func (s *Service) CreateTask(tx repo.Tx, t model.Task, actor model.User) (model.Task, error) {
	t.OwnerID = actor.ID
	t.Status = model.TaskOpen
	if err := s.Repo.SaveTask(tx, t); err != nil {
		return model.Task{}, err
	}
	return t, nil
}

// CreatePipeline persists a Pipeline.
func (s *Service) CreatePipeline(tx repo.Tx, p model.Pipeline) (model.Pipeline, error) {
	if err := s.Repo.SavePipeline(tx, p); err != nil {
		return model.Pipeline{}, err
	}
	return p, nil
}

// SetDefaultPipeline is the atomic read-modify-write that enforces
// one-default-pipeline inside the one write Tx (BUILD.md 6, 11). It is the only
// write path to isDefault: the repo unsets the prior default and sets the new one.
func (s *Service) SetDefaultPipeline(tx repo.Tx, id string) error {
	return s.Repo.SetDefaultPipeline(tx, id)
}

// LogActivity appends an immutable Activity recording the acting user
// (activity-immutable, activity-owned). SaveActivity is append-only; there is no
// update path.
func (s *Service) LogActivity(tx repo.Tx, a model.Activity) (model.Activity, error) {
	if err := s.Repo.SaveActivity(tx, a); err != nil {
		return model.Activity{}, err
	}
	return a, nil
}

// CreateTag creates a Tag; a duplicate name -> ErrConstraint (tag-name-unique).
func (s *Service) CreateTag(tx repo.Tx, t model.Tag) (model.Tag, error) {
	if err := s.Repo.SaveTag(tx, t); err != nil {
		return model.Tag{}, err
	}
	return t, nil
}

// CreateTeam creates a Team; a duplicate name -> ErrConstraint (team-name-unique).
func (s *Service) CreateTeam(tx repo.Tx, t model.Team) (model.Team, error) {
	if err := s.Repo.SaveTeam(tx, t); err != nil {
		return model.Team{}, err
	}
	return t, nil
}

// AssignTeam assigns a User to at most one Team (single-team): it overwrites the
// user's single TeamID rather than adding a second membership.
func (s *Service) AssignTeam(tx repo.Tx, userID, teamID string) error {
	u, err := s.Repo.GetUser(tx, userID)
	if err != nil {
		return err
	}
	u.TeamID = teamID
	return s.Repo.SaveUser(tx, u)
}
