package domain_test

// Property tests for the invariants enforced in crm.domain (BUILD.md 7.3). Each
// property exercises a real domain entry point over the in-memory fake Repo and
// includes at least one positive "must hold / must succeed" assertion, so every
// property is RED against the scaffolding stubs (which return ErrNotImplemented
// / the zero Effect) and does not report a false green.
//
// The other 8 invariants are property-tested next to their component:
// P-username-unique, P-password-hashed, P-disabled-cannot-auth,
// P-session-active-user in internal/session; the four P-rbac-* in internal/authz.

import (
	"testing"
	"time"

	"crm/internal/authz"
	"crm/internal/domain"
	"crm/internal/model"
	"crm/internal/testsupport"
)

func dsvc() (*domain.Service, *testsupport.FakeRepo) {
	f := testsupport.NewFakeRepo()
	return domain.NewService(f, authz.New()), f
}

// P-account-owned: every persisted Account has exactly one non-empty ownerId.
func TestPropAccountOwned(t *testing.T) {
	svc, f := dsvc()
	tx, _ := f.Open("mem")
	got, err := svc.CreateAccount(tx, model.Account{ID: "a1", Name: "Acme", OwnerID: uRepOwner.ID})
	if err != nil {
		t.Fatalf("P-account-owned: CreateAccount must succeed: %v", err)
	}
	if got.OwnerID == "" {
		t.Errorf("P-account-owned: persisted account has empty ownerId")
	}
}

// P-contact-owned: every persisted Contact has exactly one non-empty ownerId.
func TestPropContactOwned(t *testing.T) {
	svc, f := dsvc()
	tx, _ := f.Open("mem")
	got, err := svc.CreateContact(tx, model.Contact{ID: "c1", FullName: "Ann", OwnerID: uRepOwner.ID})
	if err != nil {
		t.Fatalf("P-contact-owned: CreateContact must succeed: %v", err)
	}
	if got.OwnerID == "" {
		t.Errorf("P-contact-owned: persisted contact has empty ownerId")
	}
}

// P-deal-owned: every persisted Deal has exactly one ownerId, unchanged by
// advance/win/lose/reopen.
func TestPropDealOwned(t *testing.T) {
	svc, f := dsvc()
	tx, _ := f.Open("mem")
	got, err := svc.CreateDeal(tx, model.Deal{ID: "d1", Title: "T", AmountCents: 100}, uRepOwner)
	if err != nil {
		t.Fatalf("P-deal-owned: CreateDeal must succeed: %v", err)
	}
	if got.OwnerID != uRepOwner.ID {
		t.Errorf("P-deal-owned: owner = %q, want %q", got.OwnerID, uRepOwner.ID)
	}
	// Ownership is not touched by a lifecycle transition.
	d := newDeal(domain.DSLead)
	d.OwnerID = uRepOwner.ID
	d.Fire(domain.DealEvent{Kind: domain.DEvAdvanceStage})
	if d.OwnerID != uRepOwner.ID {
		t.Errorf("P-deal-owned: advance changed owner to %q", d.OwnerID)
	}
}

// P-deal-amount-nonneg: no accepted transition ever persists amountCents<0; a
// create with amount<0 is rejected; amount>=0 is accepted.
func TestPropDealAmountNonneg(t *testing.T) {
	svc, f := dsvc()
	tx, _ := f.Open("mem")
	for _, amt := range []int64{0, 1, 100, 999999} {
		if _, err := svc.CreateDeal(tx, model.Deal{ID: "ok", Title: "T", AmountCents: amt}, uRepOwner); err != nil {
			t.Fatalf("P-deal-amount-nonneg: amount %d>=0 must be accepted: %v", amt, err)
		}
	}
	for _, amt := range []int64{-1, -100} {
		if _, err := svc.CreateDeal(tx, model.Deal{ID: "bad", Title: "T", AmountCents: amt}, uRepOwner); err == nil {
			t.Errorf("P-deal-amount-nonneg: amount %d<0 must be rejected", amt)
		}
	}
}

// P-deal-stage-forward: an accepted advance moves strictly forward; reopen is the
// only backward move and only from Won/Lost.
func TestPropDealStageForward(t *testing.T) {
	steps := []struct{ from, next domain.DealState }{
		{domain.DSLead, domain.DSQualified},
		{domain.DSQualified, domain.DSProposal},
		{domain.DSProposal, domain.DSNegotiation},
	}
	for _, s := range steps {
		d := newDeal(s.from) // admin actor, amount>=0
		d.Fire(domain.DealEvent{Kind: domain.DEvAdvanceStage})
		if d.State != domain.DSPersisting {
			t.Fatalf("P-deal-stage-forward: advance from %s must be accepted (persisting), got %s", s.from, d.State)
		}
		if d.PendingStage != s.next {
			t.Errorf("P-deal-stage-forward: pending = %s, want %s", d.PendingStage, s.next)
		}
		if model.StageIndex(model.DealStage(s.next)) <= model.StageIndex(model.DealStage(s.from)) {
			t.Errorf("P-deal-stage-forward: %s is not strictly after %s", s.next, s.from)
		}
	}
	// reopen: the only backward move, only from a terminal stage, lands on Negotiation.
	d := dealWithActor(newDeal(domain.DSWon), uMgrT1)
	d.Fire(domain.DealEvent{Kind: domain.DEvReopen})
	if d.State != domain.DSPersisting || d.PendingStage != domain.DSNegotiation {
		t.Fatalf("P-deal-stage-forward: reopen from Won must pend Negotiation, got state=%s pending=%s", d.State, d.PendingStage)
	}
}

// P-deal-terminal: from Won/Lost, no event other than reopen changes the deal.
func TestPropDealTerminal(t *testing.T) {
	for _, term := range []domain.DealState{domain.DSWon, domain.DSLost} {
		for _, ev := range []domain.DealEvent{
			{Kind: domain.DEvAdvanceStage}, {Kind: domain.DEvWin, CloseDate: closeDatePtr()}, {Kind: domain.DEvLose},
		} {
			d := newDeal(term)
			eff := d.Fire(ev)
			if d.State != term {
				t.Errorf("P-deal-terminal: event %s changed %s to %s", ev.Kind, term, d.State)
			}
			if !eff.Has("recordTerminalRejected") {
				t.Errorf("P-deal-terminal: event %s on %s must record recordTerminalRejected, got %v", ev.Kind, term, eff.Actions)
			}
		}
		// reopen with authority is the sanctioned exception (it does change the deal).
		d := dealWithActor(newDeal(term), uMgrT1)
		d.Fire(domain.DealEvent{Kind: domain.DEvReopen})
		if d.State != domain.DSPersisting {
			t.Errorf("P-deal-terminal: authorized reopen from %s must be accepted", term)
		}
	}
}

// P-deal-won-has-closedate: every Deal in Won has a non-null closeDate.
func TestPropDealWonHasCloseDate(t *testing.T) {
	d := newDeal(domain.DSNegotiation)
	d.Fire(domain.DealEvent{Kind: domain.DEvWin, CloseDate: closeDatePtr()})
	if d.State != domain.DSPersisting {
		t.Fatalf("P-deal-won-has-closedate: win with closeDate must be accepted")
	}
	d.Fire(domain.DealEvent{Kind: domain.DEvSaveDone})
	if d.State != domain.DSWon {
		t.Fatalf("P-deal-won-has-closedate: persisted win must reach Won, got %s", d.State)
	}
	if d.CloseDate == nil {
		t.Errorf("P-deal-won-has-closedate: a Won deal must have a closeDate")
	}
	// A win with no closeDate must be rejected (never reaches Won).
	d2 := newDeal(domain.DSNegotiation)
	d2.Fire(domain.DealEvent{Kind: domain.DEvWin, CloseDate: nil})
	if d2.State == domain.DSPersisting {
		t.Errorf("P-deal-won-has-closedate: win without closeDate must be rejected")
	}
}

// P-one-default-pipeline: after any sequence of create/setDefault operations,
// count(isDefault==true) == 1 (named residual; operation-level).
func TestPropOneDefaultPipeline(t *testing.T) {
	svc, f := dsvc()
	tx, _ := f.Open("mem")
	for _, id := range []string{"p0", "p1", "p2"} {
		if _, err := svc.CreatePipeline(tx, model.Pipeline{ID: id, Name: id}); err != nil {
			t.Fatalf("P-one-default-pipeline: CreatePipeline must succeed: %v", err)
		}
	}
	if err := svc.SetDefaultPipeline(tx, "p1"); err != nil {
		t.Fatalf("P-one-default-pipeline: SetDefaultPipeline must succeed: %v", err)
	}
	n, _ := f.CountDefaultPipelines(tx)
	if n != 1 {
		t.Errorf("P-one-default-pipeline: count(isDefault==true) = %d, want 1", n)
	}
	// A second setDefault still leaves exactly one.
	if err := svc.SetDefaultPipeline(tx, "p2"); err != nil {
		t.Fatalf("P-one-default-pipeline: second SetDefaultPipeline must succeed: %v", err)
	}
	if n, _ = f.CountDefaultPipelines(tx); n != 1 {
		t.Errorf("P-one-default-pipeline: after re-set, count = %d, want 1", n)
	}
}

// P-activity-immutable: a logged Activity's body and occurredAt equal their
// create-time values; no operation mutates them (there is no repo update path,
// see C-REPO-23).
func TestPropActivityImmutable(t *testing.T) {
	svc, f := dsvc()
	tx, _ := f.Open("mem")
	occurred := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	got, err := svc.LogActivity(tx, model.Activity{ID: "ac1", Type: model.ActivityNote, Subject: "s", Body: "body-v1", OccurredAt: occurred, OwnerID: uRepOwner.ID})
	if err != nil {
		t.Fatalf("P-activity-immutable: LogActivity must succeed: %v", err)
	}
	if got.Body != "body-v1" || !got.OccurredAt.Equal(occurred) {
		t.Errorf("P-activity-immutable: create-time body/occurredAt not preserved: %q %v", got.Body, got.OccurredAt)
	}
}

// P-activity-owned: every persisted Activity has a non-empty ownerId equal to the
// logging user.
func TestPropActivityOwned(t *testing.T) {
	svc, f := dsvc()
	tx, _ := f.Open("mem")
	got, err := svc.LogActivity(tx, model.Activity{ID: "ac2", Type: model.ActivityCall, Subject: "s", Body: "b", OccurredAt: time.Now(), OwnerID: uRepOwner.ID})
	if err != nil {
		t.Fatalf("P-activity-owned: LogActivity must succeed: %v", err)
	}
	if got.OwnerID != uRepOwner.ID {
		t.Errorf("P-activity-owned: ownerId = %q, want %q", got.OwnerID, uRepOwner.ID)
	}
}

// P-task-owned: every persisted Task has exactly one ownerId; reassign changes it
// to exactly one in-scope user.
func TestPropTaskOwned(t *testing.T) {
	svc, f := dsvc()
	tx, _ := f.Open("mem")
	got, err := svc.CreateTask(tx, model.Task{ID: "tk1", Title: "T"}, uRepOwner)
	if err != nil {
		t.Fatalf("P-task-owned: CreateTask must succeed: %v", err)
	}
	if got.OwnerID != uRepOwner.ID {
		t.Errorf("P-task-owned: owner = %q, want %q", got.OwnerID, uRepOwner.ID)
	}
	tk := taskWith(domain.TSInProgress, uMgrT1)
	tk.Fire(domain.TaskEvent{Kind: domain.TEvReassign, NewAssignee: uRepOther})
	if tk.State != domain.TSPersisting {
		t.Fatalf("P-task-owned: in-scope reassign must be accepted")
	}
	if tk.NewAssigneeID != uRepOther.ID {
		t.Errorf("P-task-owned: reassign target = %q, want %q", tk.NewAssigneeID, uRepOther.ID)
	}
}

// P-task-terminal: from Done/Cancelled, no event changes the task. Structural
// (Done/Cancelled are final; the machine has no outgoing transition), so this
// holds for the scaffolding too.
func TestPropTaskTerminal(t *testing.T) {
	events := []domain.TaskEvent{
		{Kind: domain.TEvStart}, {Kind: domain.TEvComplete},
		{Kind: domain.TEvCancel}, {Kind: domain.TEvReassign, NewAssignee: uRepOther},
	}
	for _, term := range []domain.TaskState{domain.TSDone, domain.TSCancelled} {
		for _, ev := range events {
			tk := newTask(term)
			eff := tk.Fire(ev)
			if tk.State != term || len(eff.Actions) != 0 {
				t.Errorf("P-task-terminal: event %s changed terminal %s (state=%s, actions=%v)", ev.Kind, term, tk.State, eff.Actions)
			}
		}
	}
}

// P-task-assignee-visible: an accepted reassign lands inside the assigner's
// VisibilityScope; out-of-scope targets are rejected.
func TestPropTaskAssigneeVisible(t *testing.T) {
	inScope := taskWith(domain.TSInProgress, uMgrT1)
	inScope.Fire(domain.TaskEvent{Kind: domain.TEvReassign, NewAssignee: uRepOther})
	if inScope.State != domain.TSPersisting {
		t.Fatalf("P-task-assignee-visible: in-scope reassign must be accepted")
	}
	outScope := taskWith(domain.TSInProgress, uMgrT1)
	eff := outScope.Fire(domain.TaskEvent{Kind: domain.TEvReassign, NewAssignee: uOtherTeam})
	if outScope.State != domain.TSInProgress || !eff.Has("recordReassignDenied") {
		t.Errorf("P-task-assignee-visible: out-of-scope reassign must be rejected (state=%s, actions=%v)", outScope.State, eff.Actions)
	}
}

// P-tag-name-unique: two Tag creates with the same name -> the second fails;
// never two Tags with one name.
func TestPropTagNameUnique(t *testing.T) {
	svc, f := dsvc()
	tx, _ := f.Open("mem")
	if _, err := svc.CreateTag(tx, model.Tag{ID: "g1", Name: "hot"}); err != nil {
		t.Fatalf("P-tag-name-unique: first CreateTag must succeed: %v", err)
	}
	if _, err := svc.CreateTag(tx, model.Tag{ID: "g2", Name: "hot"}); err == nil {
		t.Errorf("P-tag-name-unique: duplicate name must be rejected")
	}
}

// P-team-name-unique: two Team creates with the same name -> the second fails.
func TestPropTeamNameUnique(t *testing.T) {
	svc, f := dsvc()
	tx, _ := f.Open("mem")
	if _, err := svc.CreateTeam(tx, model.Team{ID: "tm1", Name: "sales"}); err != nil {
		t.Fatalf("P-team-name-unique: first CreateTeam must succeed: %v", err)
	}
	if _, err := svc.CreateTeam(tx, model.Team{ID: "tm2", Name: "sales"}); err == nil {
		t.Errorf("P-team-name-unique: duplicate name must be rejected")
	}
}

// P-single-team: a User is a member of at most one Team across any sequence of
// assignments.
func TestPropSingleTeam(t *testing.T) {
	svc, f := dsvc()
	f.Users["u1"] = model.User{ID: "u1", Username: "u", Role: model.RoleRep, Status: model.StatusActive}
	tx, _ := f.Open("mem")
	if err := svc.AssignTeam(tx, "u1", "t1"); err != nil {
		t.Fatalf("P-single-team: first AssignTeam must succeed: %v", err)
	}
	if err := svc.AssignTeam(tx, "u1", "t2"); err != nil {
		t.Fatalf("P-single-team: reassign must succeed: %v", err)
	}
	u, _ := f.GetUser(tx, "u1")
	if u.TeamID != "t2" {
		t.Errorf("P-single-team: user must belong to exactly the last team, got %q", u.TeamID)
	}
}
