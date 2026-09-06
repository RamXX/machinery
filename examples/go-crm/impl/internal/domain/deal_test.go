package domain_test

// Transition witnesses bind current committed oracle expectations to real Fire results.

import (
	"crm/internal/testoracle"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"crm/internal/authz"
	"crm/internal/domain"
	"crm/internal/model"
)

// Shared actor fixtures (roles + team membership drive the rbac re-check guards).
var (
	uAdmin    = model.User{ID: "admin", Role: model.RoleAdmin, Status: model.StatusActive}
	uReadOnly = model.User{ID: "ro", Role: model.RoleReadOnly, Status: model.StatusActive, TeamID: "t1"}
	uRepOwner = model.User{ID: "rep1", Role: model.RoleRep, Status: model.StatusActive, TeamID: "t1"}
	uRepOther = model.User{ID: "rep3", Role: model.RoleRep, Status: model.StatusActive, TeamID: "t1"}
	uMgrT1    = model.User{ID: "m1", Role: model.RoleManager, Status: model.StatusActive, TeamID: "t1"}
	uMgrT2    = model.User{ID: "m2", Role: model.RoleManager, Status: model.StatusActive, TeamID: "t2"}
)

func newDeal(state domain.DealState) *domain.Deal {
	return &domain.Deal{
		State:       state,
		DealID:      "d1",
		OwnerID:     "rep1",
		TeamID:      "t1",
		AmountCents: 1000,
		Actor:       uAdmin,
		Authz:       authz.New(),
	}
}

func closeDatePtr() *time.Time { t := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC); return &t }

type dealCase struct {
	id    string
	deal  *domain.Deal
	event domain.DealEvent
	rowID string
}

func TestDealTransitions(t *testing.T) {
	adv := domain.DealEvent{Kind: domain.DEvAdvanceStage}
	lose := domain.DealEvent{Kind: domain.DEvLose}
	reopen := domain.DealEvent{Kind: domain.DEvReopen}
	winOK := domain.DealEvent{Kind: domain.DEvWin, CloseDate: closeDatePtr()}
	winNoDate := domain.DealEvent{Kind: domain.DEvWin, CloseDate: nil}

	// helpers to mutate a fresh deal
	notWritable := func(d *domain.Deal) *domain.Deal { d.Actor = uReadOnly; return d }
	negAmount := func(d *domain.Deal) *domain.Deal { d.AmountCents = -1; return d }

	cs := []dealCase{
		// --- Lead ---
		{"T-DEAL-01_DEAL-eb0c40", newDeal(domain.DSLead), adv, "DEAL-eb0c40"},
		{"DEAL-eb0c40a / notWritable / DEAL-38ba11", notWritable(newDeal(domain.DSLead)), adv, "DEAL-38ba11"},
		{"DEAL-eb0c40b / negAmount / DEAL-38ba11", negAmount(newDeal(domain.DSLead)), adv, "DEAL-38ba11"},
		{"T-DEAL-03_DEAL-1fe825", newDeal(domain.DSLead), winOK, "DEAL-1fe825"},
		{"DEAL-1fe825a / noCloseDate / DEAL-e786d8", newDeal(domain.DSLead), winNoDate, "DEAL-e786d8"},
		{"DEAL-1fe825b / notWritable / DEAL-e786d8", notWritable(newDeal(domain.DSLead)), winOK, "DEAL-e786d8"},
		{"DEAL-1fe825c / negAmount / DEAL-e786d8", negAmount(newDeal(domain.DSLead)), winOK, "DEAL-e786d8"},
		{"T-DEAL-05_DEAL-b76457", newDeal(domain.DSLead), lose, "DEAL-b76457"},
		{"DEAL-b76457a / notWritable / DEAL-fdf795", notWritable(newDeal(domain.DSLead)), lose, "DEAL-fdf795"},
		{"DEAL-b76457b / negAmount / DEAL-fdf795", negAmount(newDeal(domain.DSLead)), lose, "DEAL-fdf795"},
		{"T-DEAL-07_DEAL-1d9aa0", newDeal(domain.DSLead), reopen, "DEAL-1d9aa0"},

		// --- Qualified ---
		{"T-DEAL-08_DEAL-a14020", newDeal(domain.DSQualified), adv, "DEAL-a14020"},
		{"DEAL-a14020a / notWritable / DEAL-0c4c47", notWritable(newDeal(domain.DSQualified)), adv, "DEAL-0c4c47"},
		{"DEAL-a14020b / negAmount / DEAL-0c4c47", negAmount(newDeal(domain.DSQualified)), adv, "DEAL-0c4c47"},
		{"T-DEAL-10_DEAL-492234", newDeal(domain.DSQualified), winOK, "DEAL-492234"},
		{"DEAL-492234a / noCloseDate / DEAL-81d0ab", newDeal(domain.DSQualified), winNoDate, "DEAL-81d0ab"},
		{"DEAL-492234b / notWritable / DEAL-81d0ab", notWritable(newDeal(domain.DSQualified)), winOK, "DEAL-81d0ab"},
		{"DEAL-492234c / negAmount / DEAL-81d0ab", negAmount(newDeal(domain.DSQualified)), winOK, "DEAL-81d0ab"},
		{"T-DEAL-12_DEAL-f7d8b2", newDeal(domain.DSQualified), lose, "DEAL-f7d8b2"},
		{"DEAL-f7d8b2a / notWritable / DEAL-9f48af", notWritable(newDeal(domain.DSQualified)), lose, "DEAL-9f48af"},
		{"DEAL-f7d8b2b / negAmount / DEAL-9f48af", negAmount(newDeal(domain.DSQualified)), lose, "DEAL-9f48af"},
		{"T-DEAL-14_DEAL-990c3b", newDeal(domain.DSQualified), reopen, "DEAL-990c3b"},

		// --- Proposal ---
		{"T-DEAL-15_DEAL-388687", newDeal(domain.DSProposal), adv, "DEAL-388687"},
		{"DEAL-388687a / notWritable / DEAL-5df488", notWritable(newDeal(domain.DSProposal)), adv, "DEAL-5df488"},
		{"DEAL-388687b / negAmount / DEAL-5df488", negAmount(newDeal(domain.DSProposal)), adv, "DEAL-5df488"},
		{"T-DEAL-17_DEAL-7e1e9b", newDeal(domain.DSProposal), winOK, "DEAL-7e1e9b"},
		{"DEAL-7e1e9ba / noCloseDate / DEAL-df4442", newDeal(domain.DSProposal), winNoDate, "DEAL-df4442"},
		{"DEAL-7e1e9bb / notWritable / DEAL-df4442", notWritable(newDeal(domain.DSProposal)), winOK, "DEAL-df4442"},
		{"DEAL-7e1e9bc / negAmount / DEAL-df4442", negAmount(newDeal(domain.DSProposal)), winOK, "DEAL-df4442"},
		{"T-DEAL-19_DEAL-fde084", newDeal(domain.DSProposal), lose, "DEAL-fde084"},
		{"DEAL-fde084a / notWritable / DEAL-e16eea", notWritable(newDeal(domain.DSProposal)), lose, "DEAL-e16eea"},
		{"DEAL-fde084b / negAmount / DEAL-e16eea", negAmount(newDeal(domain.DSProposal)), lose, "DEAL-e16eea"},
		{"T-DEAL-21_DEAL-44482d", newDeal(domain.DSProposal), reopen, "DEAL-44482d"},

		// --- Negotiation (no forward stage) ---
		{"T-DEAL-22_DEAL-708606", newDeal(domain.DSNegotiation), adv, "DEAL-708606"},
		{"T-DEAL-23_DEAL-38140e", newDeal(domain.DSNegotiation), winOK, "DEAL-38140e"},
		{"DEAL-38140ea / noCloseDate / DEAL-3bbe10", newDeal(domain.DSNegotiation), winNoDate, "DEAL-3bbe10"},
		{"DEAL-38140eb / notWritable / DEAL-3bbe10", notWritable(newDeal(domain.DSNegotiation)), winOK, "DEAL-3bbe10"},
		{"DEAL-38140ec / negAmount / DEAL-3bbe10", negAmount(newDeal(domain.DSNegotiation)), winOK, "DEAL-3bbe10"},
		{"T-DEAL-25_DEAL-8fde14", newDeal(domain.DSNegotiation), lose, "DEAL-8fde14"},
		{"DEAL-8fde14a / notWritable / DEAL-b5154b", notWritable(newDeal(domain.DSNegotiation)), lose, "DEAL-b5154b"},
		{"DEAL-8fde14b / negAmount / DEAL-b5154b", negAmount(newDeal(domain.DSNegotiation)), lose, "DEAL-b5154b"},
		{"T-DEAL-27_DEAL-69312c", newDeal(domain.DSNegotiation), reopen, "DEAL-69312c"},

		// --- Won (terminal) ---
		{"T-DEAL-28_DEAL-99392a", dealWithActor(newDeal(domain.DSWon), uMgrT1), reopen, "DEAL-99392a"},
		{"T-DEAL-29a_repNotAuthority_DEAL-5746cc", dealWithActor(newDeal(domain.DSWon), uRepOwner), reopen, "DEAL-5746cc"},
		{"T-DEAL-29b_mgrOutOfScope_DEAL-5746cc", dealWithActor(newDeal(domain.DSWon), uMgrT2), reopen, "DEAL-5746cc"},
		{"T-DEAL-30_DEAL-e0bdaf", newDeal(domain.DSWon), adv, "DEAL-e0bdaf"},
		{"T-DEAL-31_DEAL-d27905", newDeal(domain.DSWon), winOK, "DEAL-d27905"},
		{"T-DEAL-32_DEAL-a45f13", newDeal(domain.DSWon), lose, "DEAL-a45f13"},

		// --- Lost (terminal) ---
		{"T-DEAL-33_DEAL-0fef3d", dealWithActor(newDeal(domain.DSLost), uMgrT1), reopen, "DEAL-0fef3d"},
		{"T-DEAL-34a_repNotAuthority_DEAL-7bb594", dealWithActor(newDeal(domain.DSLost), uRepOwner), reopen, "DEAL-7bb594"},
		{"T-DEAL-34b_mgrOutOfScope_DEAL-7bb594", dealWithActor(newDeal(domain.DSLost), uMgrT2), reopen, "DEAL-7bb594"},
		{"T-DEAL-35_DEAL-0a25a2", newDeal(domain.DSLost), adv, "DEAL-0a25a2"},
		{"T-DEAL-36_DEAL-0ec705", newDeal(domain.DSLost), winOK, "DEAL-0ec705"},
		{"T-DEAL-37_DEAL-e9e60a", newDeal(domain.DSLost), lose, "DEAL-e9e60a"},

		// --- persist success routing (persisting, invoke onDone) ---
		{"T-DEAL-38_DEAL-5abbd2", dealPending(domain.DSPersisting, domain.DSQualified), saveDone(), "DEAL-5abbd2"},
		{"T-DEAL-39_DEAL-da0ce2", dealPending(domain.DSPersisting, domain.DSProposal), saveDone(), "DEAL-da0ce2"},
		{"T-DEAL-40_DEAL-47ce0d", dealPending(domain.DSPersisting, domain.DSNegotiation), saveDone(), "DEAL-47ce0d"},
		{"T-DEAL-41_DEAL-e5d58e", dealPendingWon(), saveDone(), "DEAL-e5d58e"},
		{"T-DEAL-42_DEAL-03d4fb", dealPending(domain.DSPersisting, domain.DSLost), saveDone(), "DEAL-03d4fb"},
		{"T-DEAL-43_DEAL-92b688", dealPending(domain.DSPersisting, domain.DSLead), saveDone(), "DEAL-92b688"},

		// --- persist error routing (persisting, invoke onError) ---
		{"T-DEAL-44_DEAL-809c09", newDeal(domain.DSPersisting), saveErr(model.ErrLocked), "DEAL-809c09"},
		{"T-DEAL-45_DEAL-cf2596", newDeal(domain.DSPersisting), saveErr(model.ErrConstraint), "DEAL-cf2596"},
		{"T-DEAL-46_DEAL-daae59", newDeal(domain.DSPersisting), saveErr(model.ErrDiskFull), "DEAL-daae59"},
		{"T-DEAL-47_DEAL-41c002", newDeal(domain.DSPersisting), saveErr(model.ErrTimeout), "DEAL-41c002"},
		{"T-DEAL-48_DEAL-7d1911", newDeal(domain.DSPersisting), saveErr(model.ErrConflict), "DEAL-7d1911"},
		{"T-DEAL-49_DEAL-24f320", newDeal(domain.DSPersisting), domain.DealEvent{Kind: domain.DEvPersistTimeout}, "DEAL-24f320"},

		// --- persistRetry ---
		{"T-DEAL-50_DEAL-8c9948", dealRetries(domain.DSPersistRetry, 3), domain.DealEvent{Kind: domain.DEvAlways}, "DEAL-8c9948"},
		{"T-DEAL-51_DEAL-450b55", dealRetries(domain.DSPersistRetry, 0), domain.DealEvent{Kind: domain.DEvRetryBackoff}, "DEAL-450b55"},

		// --- rolledBack routing (atomic rollback to priorStage; no action) ---
		{"T-DEAL-52_DEAL-210c14", dealPrior(domain.DSLead), domain.DealEvent{Kind: domain.DEvAlways}, "DEAL-210c14"},
		{"T-DEAL-53_DEAL-793e1f", dealPrior(domain.DSQualified), domain.DealEvent{Kind: domain.DEvAlways}, "DEAL-793e1f"},
		{"T-DEAL-54_DEAL-97c3ea", dealPrior(domain.DSProposal), domain.DealEvent{Kind: domain.DEvAlways}, "DEAL-97c3ea"},
		{"T-DEAL-55_DEAL-8a4caf", dealPrior(domain.DSNegotiation), domain.DealEvent{Kind: domain.DEvAlways}, "DEAL-8a4caf"},
		{"T-DEAL-56_DEAL-9b6ee7", dealPrior(domain.DSWon), domain.DealEvent{Kind: domain.DEvAlways}, "DEAL-9b6ee7"},
		{"T-DEAL-57_DEAL-21905a", dealPrior(domain.DSLost), domain.DealEvent{Kind: domain.DEvAlways}, "DEAL-21905a"},
		{"DEAL-b48a23 / fail-closed rollback routing", dealPrior(domain.DealState("bogus")), domain.DealEvent{Kind: domain.DEvAlways}, "DEAL-b48a23"},
	}
	suite, err := testoracle.Load("../../../design/machines", "Deal")
	if err != nil {
		t.Fatal(err)
	}
	expectations := make([]testoracle.Expectation, len(cs))
	registrations := make([]testoracle.Registration, len(cs))
	for i, tc := range cs {
		expected, err := suite.Bind(dealWitness(t, tc))
		if err != nil {
			t.Fatal(err)
		}
		expectations[i] = expected
		registrations[i] = testoracle.Registration{Machine: suite.Name, Native: t.Name() + "/" + strings.ReplaceAll(tc.id, " ", "_"), Identity: expected.Row.Identity}
		raw, err := json.Marshal(registrations[i])
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("oracle-registered %s", raw)
	}
	if err := suite.Finish(); err != nil {
		t.Fatal(err)
	}
	for i, tc := range cs {
		t.Run(tc.id, func(t *testing.T) {
			got := tc.deal.Fire(tc.event)
			if err := expectations[i].Check(string(tc.deal.State), got.Actions); err != nil {
				t.Fatal(err)
			}
			if registrations[i].Native != t.Name() {
				t.Fatalf("oracle-native-name: got=%q want=%q", t.Name(), registrations[i].Native)
			}
			raw, err := json.Marshal(testoracle.Observation{Registration: registrations[i], Next: string(tc.deal.State), Actions: got.Actions})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("oracle-executed %s", raw)
		})
	}
}

func dealWithActor(d *domain.Deal, a model.User) *domain.Deal { d.Actor = a; return d }

func dealPending(state, pending domain.DealState) *domain.Deal {
	d := newDeal(state)
	d.PendingStage = pending
	return d
}

func dealPendingWon() *domain.Deal {
	d := newDeal(domain.DSPersisting)
	d.PendingStage = domain.DSWon
	d.PendingCloseDate = closeDatePtr()
	return d
}

func dealRetries(state domain.DealState, n int) *domain.Deal {
	d := newDeal(state)
	d.Retries = n
	return d
}

func dealPrior(prior domain.DealState) *domain.Deal {
	d := newDeal(domain.DSRolledBack)
	d.PriorStage = prior
	return d
}

func saveDone() domain.DealEvent { return domain.DealEvent{Kind: domain.DEvSaveDone} }
func saveErr(e error) domain.DealEvent {
	return domain.DealEvent{Kind: domain.DEvSaveError, Err: e}
}

// These fixture predicates spell out the existing policy witnesses independently
// of production guard functions. They never inspect oracle IDs or expected rows.
func oracleWriteScope(actor model.User, owner, team string) bool {
	owns := owner != "" && owner == actor.ID
	sameTeam := team != "" && team == actor.TeamID
	return actor.Role == model.RoleAdmin || actor.Role == model.RoleManager && (owns || sameTeam) || actor.Role == model.RoleRep && owns
}
func oracleAuthority(actor model.User, team string) bool {
	return actor.Role == model.RoleAdmin || actor.Role == model.RoleManager && team != "" && actor.TeamID == team
}
func oracleFacts(t *testing.T, all map[string]bool, keys ...string) map[string]bool {
	facts := map[string]bool{}
	for _, key := range keys {
		value, ok := all[key]
		if !ok {
			t.Fatalf("oracle-guard-key domain: %s", key)
		}
		facts[key] = value
	}
	return facts
}
func oracleClause(t *testing.T, label, clause string, holds bool) {
	t.Helper()
	if strings.Contains(label, clause) && holds {
		t.Fatalf("oracle-clause %s: falsifying clause %s unexpectedly true", label, clause)
	}
}

func dealWitness(t *testing.T, tc dealCase) testoracle.Witness {
	t.Helper()
	d, e := tc.deal, tc.event
	write := oracleWriteScope(d.Actor, d.OwnerID, d.TeamID)
	amount := d.AmountCents >= 0
	date := e.CloseDate != nil
	authority := oracleAuthority(d.Actor, d.TeamID)
	forward := d.State == domain.DSLead || d.State == domain.DSQualified || d.State == domain.DSProposal
	oracleClause(t, tc.id, "notWritable", write)
	oracleClause(t, tc.id, "negAmount", amount)
	oracleClause(t, tc.id, "noCloseDate", date)
	oracleClause(t, tc.id, "repNotAuthority", d.Actor.Role == model.RoleAdmin || d.Actor.Role == model.RoleManager)
	oracleClause(t, tc.id, "mgrOutOfScope", d.TeamID != "" && d.Actor.TeamID == d.TeamID)
	all := map[string]bool{
		"guardCanAdvance": write && amount && forward, "guardCanWin": write && amount && date,
		"guardCanLose": write && amount, "guardCanReopen": authority,
		"pendingIsQualified": d.PendingStage == domain.DSQualified, "pendingIsProposal": d.PendingStage == domain.DSProposal,
		"pendingIsNegotiation": d.PendingStage == domain.DSNegotiation, "pendingIsWon": d.PendingStage == domain.DSWon,
		"pendingIsLost": d.PendingStage == domain.DSLost, "retriesExhausted": d.Retries >= 3,
		"priorIsLead": d.PriorStage == domain.DSLead, "priorIsQualified": d.PriorStage == domain.DSQualified,
		"priorIsProposal": d.PriorStage == domain.DSProposal, "priorIsNegotiation": d.PriorStage == domain.DSNegotiation,
		"priorIsWon": d.PriorStage == domain.DSWon, "priorIsLost": d.PriorStage == domain.DSLost,
		"isErrLocked": errors.Is(e.Err, model.ErrLocked), "isErrConstraint": errors.Is(e.Err, model.ErrConstraint),
		"isErrDiskFull": errors.Is(e.Err, model.ErrDiskFull), "isErrTimeout": errors.Is(e.Err, model.ErrTimeout),
	}
	w := testoracle.Witness{Name: tc.id, RowID: tc.rowID, Source: string(d.State)}
	switch d.State {
	case domain.DSLead, domain.DSQualified, domain.DSProposal, domain.DSNegotiation, domain.DSWon, domain.DSLost:
		switch e.Kind {
		case domain.DEvAdvanceStage, domain.DEvWin, domain.DEvLose, domain.DEvReopen:
			w.Trigger = "on:" + string(e.Kind)
			keys := []string{}
			terminal := d.State == domain.DSWon || d.State == domain.DSLost
			switch {
			case terminal && e.Kind == domain.DEvReopen:
				keys = []string{"guardCanReopen"}
			case !terminal && e.Kind == domain.DEvAdvanceStage && forward:
				keys = []string{"guardCanAdvance"}
			case !terminal && e.Kind == domain.DEvWin:
				keys = []string{"guardCanWin"}
			case !terminal && e.Kind == domain.DEvLose:
				keys = []string{"guardCanLose"}
			}
			w.Guards = oracleFacts(t, all, keys...)
			return w
		}
	case domain.DSPersisting:
		switch e.Kind {
		case domain.DEvSaveDone:
			w.Trigger = "onDone:saveDeal"
			w.Guards = oracleFacts(t, all, "pendingIsQualified", "pendingIsProposal", "pendingIsNegotiation", "pendingIsWon", "pendingIsLost")
			return w
		case domain.DEvSaveError:
			w.Trigger = "onError:saveDeal"
			w.Guards = oracleFacts(t, all, "isErrLocked", "isErrConstraint", "isErrDiskFull", "isErrTimeout")
			return w
		case domain.DEvPersistTimeout:
			w.Trigger = "after:persistTimeout"
			return w
		}
	case domain.DSPersistRetry:
		switch e.Kind {
		case domain.DEvAlways:
			w.Trigger = "always"
			w.Guards = oracleFacts(t, all, "retriesExhausted")
			return w
		case domain.DEvRetryBackoff:
			w.Trigger = "after:persistRetryBackoff"
			return w
		}
	case domain.DSRolledBack:
		if e.Kind == domain.DEvAlways {
			w.Trigger = "always"
			w.Guards = oracleFacts(t, all, "priorIsLead", "priorIsQualified", "priorIsProposal", "priorIsNegotiation", "priorIsWon", "priorIsLost")
			return w
		}
	}
	t.Fatalf("oracle-event deal: unsupported actual source=%q event=%q", d.State, e.Kind)
	return w
}
