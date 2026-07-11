package domain_test

// Deal transition oracle. One table case per BUILD.md 7.1 T-DEAL row (and one
// sub-case per falsifying clause of every conjunction guard, per the 7.1
// guard-branch completeness note). Source: machines/Deal.matrix.md.
//
// Each case sets up the "given state + context", fires the event, and asserts
// the expected next state (the aggregate's State field) and the exact ordered
// list of fired actions (Effect.Actions). Against the scaffolding stubs every
// row is RED (Fire returns the zero Effect and does not change state).

import (
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
	id      string
	deal    *domain.Deal
	event   domain.DealEvent
	want    domain.DealState
	actions []string
}

func dealCases() []dealCase {
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
		{"T-DEAL-01_DEAL-eb0c40", newDeal(domain.DSLead), adv, domain.DSPersisting, []string{"setPendingAdvance"}},
		{"T-DEAL-02a_notWritable_DEAL-38ba11", notWritable(newDeal(domain.DSLead)), adv, domain.DSLead, []string{"recordAdvanceDenied"}},
		{"T-DEAL-02b_negAmount_DEAL-38ba11", negAmount(newDeal(domain.DSLead)), adv, domain.DSLead, []string{"recordAdvanceDenied"}},
		{"T-DEAL-03_DEAL-1fe825", newDeal(domain.DSLead), winOK, domain.DSPersisting, []string{"setPendingWin"}},
		{"T-DEAL-04a_noCloseDate_DEAL-e786d8", newDeal(domain.DSLead), winNoDate, domain.DSLead, []string{"recordWinDenied"}},
		{"T-DEAL-04b_notWritable_DEAL-e786d8", notWritable(newDeal(domain.DSLead)), winOK, domain.DSLead, []string{"recordWinDenied"}},
		{"T-DEAL-04c_negAmount_DEAL-e786d8", negAmount(newDeal(domain.DSLead)), winOK, domain.DSLead, []string{"recordWinDenied"}},
		{"T-DEAL-05_DEAL-b76457", newDeal(domain.DSLead), lose, domain.DSPersisting, []string{"setPendingLose"}},
		{"T-DEAL-06a_notWritable_DEAL-fdf795", notWritable(newDeal(domain.DSLead)), lose, domain.DSLead, []string{"recordLoseDenied"}},
		{"T-DEAL-06b_negAmount_DEAL-fdf795", negAmount(newDeal(domain.DSLead)), lose, domain.DSLead, []string{"recordLoseDenied"}},
		{"T-DEAL-07_DEAL-1d9aa0", newDeal(domain.DSLead), reopen, domain.DSLead, []string{"recordReopenNotTerminal"}},

		// --- Qualified ---
		{"T-DEAL-08_DEAL-a14020", newDeal(domain.DSQualified), adv, domain.DSPersisting, []string{"setPendingAdvance"}},
		{"T-DEAL-09a_notWritable_DEAL-0c4c47", notWritable(newDeal(domain.DSQualified)), adv, domain.DSQualified, []string{"recordAdvanceDenied"}},
		{"T-DEAL-09b_negAmount_DEAL-0c4c47", negAmount(newDeal(domain.DSQualified)), adv, domain.DSQualified, []string{"recordAdvanceDenied"}},
		{"T-DEAL-10_DEAL-492234", newDeal(domain.DSQualified), winOK, domain.DSPersisting, []string{"setPendingWin"}},
		{"T-DEAL-11a_noCloseDate_DEAL-81d0ab", newDeal(domain.DSQualified), winNoDate, domain.DSQualified, []string{"recordWinDenied"}},
		{"T-DEAL-11b_notWritable_DEAL-81d0ab", notWritable(newDeal(domain.DSQualified)), winOK, domain.DSQualified, []string{"recordWinDenied"}},
		{"T-DEAL-11c_negAmount_DEAL-81d0ab", negAmount(newDeal(domain.DSQualified)), winOK, domain.DSQualified, []string{"recordWinDenied"}},
		{"T-DEAL-12_DEAL-f7d8b2", newDeal(domain.DSQualified), lose, domain.DSPersisting, []string{"setPendingLose"}},
		{"T-DEAL-13a_notWritable_DEAL-9f48af", notWritable(newDeal(domain.DSQualified)), lose, domain.DSQualified, []string{"recordLoseDenied"}},
		{"T-DEAL-13b_negAmount_DEAL-9f48af", negAmount(newDeal(domain.DSQualified)), lose, domain.DSQualified, []string{"recordLoseDenied"}},
		{"T-DEAL-14_DEAL-990c3b", newDeal(domain.DSQualified), reopen, domain.DSQualified, []string{"recordReopenNotTerminal"}},

		// --- Proposal ---
		{"T-DEAL-15_DEAL-388687", newDeal(domain.DSProposal), adv, domain.DSPersisting, []string{"setPendingAdvance"}},
		{"T-DEAL-16a_notWritable_DEAL-5df488", notWritable(newDeal(domain.DSProposal)), adv, domain.DSProposal, []string{"recordAdvanceDenied"}},
		{"T-DEAL-16b_negAmount_DEAL-5df488", negAmount(newDeal(domain.DSProposal)), adv, domain.DSProposal, []string{"recordAdvanceDenied"}},
		{"T-DEAL-17_DEAL-7e1e9b", newDeal(domain.DSProposal), winOK, domain.DSPersisting, []string{"setPendingWin"}},
		{"T-DEAL-18a_noCloseDate_DEAL-df4442", newDeal(domain.DSProposal), winNoDate, domain.DSProposal, []string{"recordWinDenied"}},
		{"T-DEAL-18b_notWritable_DEAL-df4442", notWritable(newDeal(domain.DSProposal)), winOK, domain.DSProposal, []string{"recordWinDenied"}},
		{"T-DEAL-18c_negAmount_DEAL-df4442", negAmount(newDeal(domain.DSProposal)), winOK, domain.DSProposal, []string{"recordWinDenied"}},
		{"T-DEAL-19_DEAL-fde084", newDeal(domain.DSProposal), lose, domain.DSPersisting, []string{"setPendingLose"}},
		{"T-DEAL-20a_notWritable_DEAL-e16eea", notWritable(newDeal(domain.DSProposal)), lose, domain.DSProposal, []string{"recordLoseDenied"}},
		{"T-DEAL-20b_negAmount_DEAL-e16eea", negAmount(newDeal(domain.DSProposal)), lose, domain.DSProposal, []string{"recordLoseDenied"}},
		{"T-DEAL-21_DEAL-44482d", newDeal(domain.DSProposal), reopen, domain.DSProposal, []string{"recordReopenNotTerminal"}},

		// --- Negotiation (no forward stage) ---
		{"T-DEAL-22_DEAL-708606", newDeal(domain.DSNegotiation), adv, domain.DSNegotiation, []string{"recordAdvanceDenied"}},
		{"T-DEAL-23_DEAL-38140e", newDeal(domain.DSNegotiation), winOK, domain.DSPersisting, []string{"setPendingWin"}},
		{"T-DEAL-24a_noCloseDate_DEAL-3bbe10", newDeal(domain.DSNegotiation), winNoDate, domain.DSNegotiation, []string{"recordWinDenied"}},
		{"T-DEAL-24b_notWritable_DEAL-3bbe10", notWritable(newDeal(domain.DSNegotiation)), winOK, domain.DSNegotiation, []string{"recordWinDenied"}},
		{"T-DEAL-24c_negAmount_DEAL-3bbe10", negAmount(newDeal(domain.DSNegotiation)), winOK, domain.DSNegotiation, []string{"recordWinDenied"}},
		{"T-DEAL-25_DEAL-8fde14", newDeal(domain.DSNegotiation), lose, domain.DSPersisting, []string{"setPendingLose"}},
		{"T-DEAL-26a_notWritable_DEAL-b5154b", notWritable(newDeal(domain.DSNegotiation)), lose, domain.DSNegotiation, []string{"recordLoseDenied"}},
		{"T-DEAL-26b_negAmount_DEAL-b5154b", negAmount(newDeal(domain.DSNegotiation)), lose, domain.DSNegotiation, []string{"recordLoseDenied"}},
		{"T-DEAL-27_DEAL-69312c", newDeal(domain.DSNegotiation), reopen, domain.DSNegotiation, []string{"recordReopenNotTerminal"}},

		// --- Won (terminal) ---
		{"T-DEAL-28_DEAL-99392a", dealWithActor(newDeal(domain.DSWon), uMgrT1), reopen, domain.DSPersisting, []string{"setPendingReopen"}},
		{"T-DEAL-29a_repNotAuthority_DEAL-5746cc", dealWithActor(newDeal(domain.DSWon), uRepOwner), reopen, domain.DSWon, []string{"recordReopenDenied"}},
		{"T-DEAL-29b_mgrOutOfScope_DEAL-5746cc", dealWithActor(newDeal(domain.DSWon), uMgrT2), reopen, domain.DSWon, []string{"recordReopenDenied"}},
		{"T-DEAL-30_DEAL-e0bdaf", newDeal(domain.DSWon), adv, domain.DSWon, []string{"recordTerminalRejected"}},
		{"T-DEAL-31_DEAL-d27905", newDeal(domain.DSWon), winOK, domain.DSWon, []string{"recordTerminalRejected"}},
		{"T-DEAL-32_DEAL-a45f13", newDeal(domain.DSWon), lose, domain.DSWon, []string{"recordTerminalRejected"}},

		// --- Lost (terminal) ---
		{"T-DEAL-33_DEAL-0fef3d", dealWithActor(newDeal(domain.DSLost), uMgrT1), reopen, domain.DSPersisting, []string{"setPendingReopen"}},
		{"T-DEAL-34a_repNotAuthority_DEAL-7bb594", dealWithActor(newDeal(domain.DSLost), uRepOwner), reopen, domain.DSLost, []string{"recordReopenDenied"}},
		{"T-DEAL-34b_mgrOutOfScope_DEAL-7bb594", dealWithActor(newDeal(domain.DSLost), uMgrT2), reopen, domain.DSLost, []string{"recordReopenDenied"}},
		{"T-DEAL-35_DEAL-0a25a2", newDeal(domain.DSLost), adv, domain.DSLost, []string{"recordTerminalRejected"}},
		{"T-DEAL-36_DEAL-0ec705", newDeal(domain.DSLost), winOK, domain.DSLost, []string{"recordTerminalRejected"}},
		{"T-DEAL-37_DEAL-e9e60a", newDeal(domain.DSLost), lose, domain.DSLost, []string{"recordTerminalRejected"}},

		// --- persist success routing (persisting, invoke onDone) ---
		{"T-DEAL-38_DEAL-5abbd2", dealPending(domain.DSPersisting, domain.DSQualified), saveDone(), domain.DSQualified, []string{"commitStage"}},
		{"T-DEAL-39_DEAL-da0ce2", dealPending(domain.DSPersisting, domain.DSProposal), saveDone(), domain.DSProposal, []string{"commitStage"}},
		{"T-DEAL-40_DEAL-47ce0d", dealPending(domain.DSPersisting, domain.DSNegotiation), saveDone(), domain.DSNegotiation, []string{"commitStage"}},
		{"T-DEAL-41_DEAL-e5d58e", dealPendingWon(), saveDone(), domain.DSWon, []string{"commitStage", "commitCloseDate"}},
		{"T-DEAL-42_DEAL-03d4fb", dealPending(domain.DSPersisting, domain.DSLost), saveDone(), domain.DSLost, []string{"commitStage"}},
		{"T-DEAL-43_DEAL-92b688", dealPending(domain.DSPersisting, domain.DSLead), saveDone(), domain.DSRolledBack, []string{"recordRoutingError"}},

		// --- persist error routing (persisting, invoke onError) ---
		{"T-DEAL-44_DEAL-809c09", newDeal(domain.DSPersisting), saveErr(model.ErrLocked), domain.DSPersistRetry, []string{"recordError"}},
		{"T-DEAL-45_DEAL-cf2596", newDeal(domain.DSPersisting), saveErr(model.ErrConstraint), domain.DSRolledBack, []string{"recordConstraint"}},
		{"T-DEAL-46_DEAL-daae59", newDeal(domain.DSPersisting), saveErr(model.ErrDiskFull), domain.DSRolledBack, []string{"recordDiskFull"}},
		{"T-DEAL-47_DEAL-41c002", newDeal(domain.DSPersisting), saveErr(model.ErrTimeout), domain.DSRolledBack, []string{"recordTimeout"}},
		{"T-DEAL-48_DEAL-7d1911", newDeal(domain.DSPersisting), saveErr(model.ErrConflict), domain.DSRolledBack, []string{"recordUnknownError"}},
		{"T-DEAL-49_DEAL-24f320", newDeal(domain.DSPersisting), domain.DealEvent{Kind: domain.DEvPersistTimeout}, domain.DSRolledBack, []string{"recordTimeout"}},

		// --- persistRetry ---
		{"T-DEAL-50_DEAL-8c9948", dealRetries(domain.DSPersistRetry, 3), domain.DealEvent{Kind: domain.DEvAlways}, domain.DSRolledBack, []string{"recordRetriesExhausted"}},
		{"T-DEAL-51_DEAL-450b55", dealRetries(domain.DSPersistRetry, 0), domain.DealEvent{Kind: domain.DEvRetryBackoff}, domain.DSPersisting, []string{"incrementRetries"}},

		// --- rolledBack routing (atomic rollback to priorStage; no action) ---
		{"T-DEAL-52_DEAL-210c14", dealPrior(domain.DSLead), domain.DealEvent{Kind: domain.DEvAlways}, domain.DSLead, nil},
		{"T-DEAL-53_DEAL-793e1f", dealPrior(domain.DSQualified), domain.DealEvent{Kind: domain.DEvAlways}, domain.DSQualified, nil},
		{"T-DEAL-54_DEAL-97c3ea", dealPrior(domain.DSProposal), domain.DealEvent{Kind: domain.DEvAlways}, domain.DSProposal, nil},
		{"T-DEAL-55_DEAL-8a4caf", dealPrior(domain.DSNegotiation), domain.DealEvent{Kind: domain.DEvAlways}, domain.DSNegotiation, nil},
		{"T-DEAL-56_DEAL-9b6ee7", dealPrior(domain.DSWon), domain.DealEvent{Kind: domain.DEvAlways}, domain.DSWon, nil},
		{"T-DEAL-57_DEAL-21905a", dealPrior(domain.DSLost), domain.DealEvent{Kind: domain.DEvAlways}, domain.DSLost, nil},
	}
	return cs
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

func TestDealTransitions(t *testing.T) {
	for _, tc := range dealCases() {
		t.Run(tc.id, func(t *testing.T) {
			got := tc.deal.Fire(tc.event)
			if tc.deal.State != tc.want {
				t.Errorf("%s: next state = %q, want %q", tc.id, tc.deal.State, tc.want)
			}
			if !firedInOrder(got.Actions, tc.actions) {
				t.Errorf("%s: actions = %v, want (in order) %v", tc.id, got.Actions, tc.actions)
			}
		})
	}
}

// firedInOrder reports whether every action in want appears in got in the same
// relative order (want is an ordered subsequence of got). Containment rather than
// exact equality reconciles the matrix "actions" column with the machine-JSON
// entry actions the matrices omit (for example Task Done/Cancelled fire the
// entry action recordTaskClosed in addition to commitStatus, and
// CommandExecution Opening/Executing fire setPhaseOpen/setPhaseExecute on entry).
// The row's listed actions must all fire, in order; the stub fires none, so every
// non-empty expectation is RED.
func firedInOrder(got, want []string) bool {
	i := 0
	for _, g := range got {
		if i < len(want) && g == want[i] {
			i++
		}
	}
	return i == len(want)
}
