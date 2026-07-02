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
		{"T-DEAL-01", newDeal(domain.DSLead), adv, domain.DSPersisting, []string{"setPendingAdvance"}},
		{"T-DEAL-02a_notWritable", notWritable(newDeal(domain.DSLead)), adv, domain.DSLead, []string{"recordAdvanceDenied"}},
		{"T-DEAL-02b_negAmount", negAmount(newDeal(domain.DSLead)), adv, domain.DSLead, []string{"recordAdvanceDenied"}},
		{"T-DEAL-03", newDeal(domain.DSLead), winOK, domain.DSPersisting, []string{"setPendingWin"}},
		{"T-DEAL-04a_noCloseDate", newDeal(domain.DSLead), winNoDate, domain.DSLead, []string{"recordWinDenied"}},
		{"T-DEAL-04b_notWritable", notWritable(newDeal(domain.DSLead)), winOK, domain.DSLead, []string{"recordWinDenied"}},
		{"T-DEAL-04c_negAmount", negAmount(newDeal(domain.DSLead)), winOK, domain.DSLead, []string{"recordWinDenied"}},
		{"T-DEAL-05", newDeal(domain.DSLead), lose, domain.DSPersisting, []string{"setPendingLose"}},
		{"T-DEAL-06a_notWritable", notWritable(newDeal(domain.DSLead)), lose, domain.DSLead, []string{"recordLoseDenied"}},
		{"T-DEAL-06b_negAmount", negAmount(newDeal(domain.DSLead)), lose, domain.DSLead, []string{"recordLoseDenied"}},
		{"T-DEAL-07", newDeal(domain.DSLead), reopen, domain.DSLead, []string{"recordReopenNotTerminal"}},

		// --- Qualified ---
		{"T-DEAL-08", newDeal(domain.DSQualified), adv, domain.DSPersisting, []string{"setPendingAdvance"}},
		{"T-DEAL-09a_notWritable", notWritable(newDeal(domain.DSQualified)), adv, domain.DSQualified, []string{"recordAdvanceDenied"}},
		{"T-DEAL-09b_negAmount", negAmount(newDeal(domain.DSQualified)), adv, domain.DSQualified, []string{"recordAdvanceDenied"}},
		{"T-DEAL-10", newDeal(domain.DSQualified), winOK, domain.DSPersisting, []string{"setPendingWin"}},
		{"T-DEAL-11a_noCloseDate", newDeal(domain.DSQualified), winNoDate, domain.DSQualified, []string{"recordWinDenied"}},
		{"T-DEAL-11b_notWritable", notWritable(newDeal(domain.DSQualified)), winOK, domain.DSQualified, []string{"recordWinDenied"}},
		{"T-DEAL-11c_negAmount", negAmount(newDeal(domain.DSQualified)), winOK, domain.DSQualified, []string{"recordWinDenied"}},
		{"T-DEAL-12", newDeal(domain.DSQualified), lose, domain.DSPersisting, []string{"setPendingLose"}},
		{"T-DEAL-13a_notWritable", notWritable(newDeal(domain.DSQualified)), lose, domain.DSQualified, []string{"recordLoseDenied"}},
		{"T-DEAL-13b_negAmount", negAmount(newDeal(domain.DSQualified)), lose, domain.DSQualified, []string{"recordLoseDenied"}},
		{"T-DEAL-14", newDeal(domain.DSQualified), reopen, domain.DSQualified, []string{"recordReopenNotTerminal"}},

		// --- Proposal ---
		{"T-DEAL-15", newDeal(domain.DSProposal), adv, domain.DSPersisting, []string{"setPendingAdvance"}},
		{"T-DEAL-16a_notWritable", notWritable(newDeal(domain.DSProposal)), adv, domain.DSProposal, []string{"recordAdvanceDenied"}},
		{"T-DEAL-16b_negAmount", negAmount(newDeal(domain.DSProposal)), adv, domain.DSProposal, []string{"recordAdvanceDenied"}},
		{"T-DEAL-17", newDeal(domain.DSProposal), winOK, domain.DSPersisting, []string{"setPendingWin"}},
		{"T-DEAL-18a_noCloseDate", newDeal(domain.DSProposal), winNoDate, domain.DSProposal, []string{"recordWinDenied"}},
		{"T-DEAL-18b_notWritable", notWritable(newDeal(domain.DSProposal)), winOK, domain.DSProposal, []string{"recordWinDenied"}},
		{"T-DEAL-18c_negAmount", negAmount(newDeal(domain.DSProposal)), winOK, domain.DSProposal, []string{"recordWinDenied"}},
		{"T-DEAL-19", newDeal(domain.DSProposal), lose, domain.DSPersisting, []string{"setPendingLose"}},
		{"T-DEAL-20a_notWritable", notWritable(newDeal(domain.DSProposal)), lose, domain.DSProposal, []string{"recordLoseDenied"}},
		{"T-DEAL-20b_negAmount", negAmount(newDeal(domain.DSProposal)), lose, domain.DSProposal, []string{"recordLoseDenied"}},
		{"T-DEAL-21", newDeal(domain.DSProposal), reopen, domain.DSProposal, []string{"recordReopenNotTerminal"}},

		// --- Negotiation (no forward stage) ---
		{"T-DEAL-22", newDeal(domain.DSNegotiation), adv, domain.DSNegotiation, []string{"recordAdvanceDenied"}},
		{"T-DEAL-23", newDeal(domain.DSNegotiation), winOK, domain.DSPersisting, []string{"setPendingWin"}},
		{"T-DEAL-24a_noCloseDate", newDeal(domain.DSNegotiation), winNoDate, domain.DSNegotiation, []string{"recordWinDenied"}},
		{"T-DEAL-24b_notWritable", notWritable(newDeal(domain.DSNegotiation)), winOK, domain.DSNegotiation, []string{"recordWinDenied"}},
		{"T-DEAL-24c_negAmount", negAmount(newDeal(domain.DSNegotiation)), winOK, domain.DSNegotiation, []string{"recordWinDenied"}},
		{"T-DEAL-25", newDeal(domain.DSNegotiation), lose, domain.DSPersisting, []string{"setPendingLose"}},
		{"T-DEAL-26a_notWritable", notWritable(newDeal(domain.DSNegotiation)), lose, domain.DSNegotiation, []string{"recordLoseDenied"}},
		{"T-DEAL-26b_negAmount", negAmount(newDeal(domain.DSNegotiation)), lose, domain.DSNegotiation, []string{"recordLoseDenied"}},
		{"T-DEAL-27", newDeal(domain.DSNegotiation), reopen, domain.DSNegotiation, []string{"recordReopenNotTerminal"}},

		// --- Won (terminal) ---
		{"T-DEAL-28", dealWithActor(newDeal(domain.DSWon), uMgrT1), reopen, domain.DSPersisting, []string{"setPendingReopen"}},
		{"T-DEAL-29a_repNotAuthority", dealWithActor(newDeal(domain.DSWon), uRepOwner), reopen, domain.DSWon, []string{"recordReopenDenied"}},
		{"T-DEAL-29b_mgrOutOfScope", dealWithActor(newDeal(domain.DSWon), uMgrT2), reopen, domain.DSWon, []string{"recordReopenDenied"}},
		{"T-DEAL-30", newDeal(domain.DSWon), adv, domain.DSWon, []string{"recordTerminalRejected"}},
		{"T-DEAL-31", newDeal(domain.DSWon), winOK, domain.DSWon, []string{"recordTerminalRejected"}},
		{"T-DEAL-32", newDeal(domain.DSWon), lose, domain.DSWon, []string{"recordTerminalRejected"}},

		// --- Lost (terminal) ---
		{"T-DEAL-33", dealWithActor(newDeal(domain.DSLost), uMgrT1), reopen, domain.DSPersisting, []string{"setPendingReopen"}},
		{"T-DEAL-34a_repNotAuthority", dealWithActor(newDeal(domain.DSLost), uRepOwner), reopen, domain.DSLost, []string{"recordReopenDenied"}},
		{"T-DEAL-34b_mgrOutOfScope", dealWithActor(newDeal(domain.DSLost), uMgrT2), reopen, domain.DSLost, []string{"recordReopenDenied"}},
		{"T-DEAL-35", newDeal(domain.DSLost), adv, domain.DSLost, []string{"recordTerminalRejected"}},
		{"T-DEAL-36", newDeal(domain.DSLost), winOK, domain.DSLost, []string{"recordTerminalRejected"}},
		{"T-DEAL-37", newDeal(domain.DSLost), lose, domain.DSLost, []string{"recordTerminalRejected"}},

		// --- persist success routing (persisting, invoke onDone) ---
		{"T-DEAL-38", dealPending(domain.DSPersisting, domain.DSQualified), saveDone(), domain.DSQualified, []string{"commitStage"}},
		{"T-DEAL-39", dealPending(domain.DSPersisting, domain.DSProposal), saveDone(), domain.DSProposal, []string{"commitStage"}},
		{"T-DEAL-40", dealPending(domain.DSPersisting, domain.DSNegotiation), saveDone(), domain.DSNegotiation, []string{"commitStage"}},
		{"T-DEAL-41", dealPendingWon(), saveDone(), domain.DSWon, []string{"commitStage", "commitCloseDate"}},
		{"T-DEAL-42", dealPending(domain.DSPersisting, domain.DSLost), saveDone(), domain.DSLost, []string{"commitStage"}},
		{"T-DEAL-43", dealPending(domain.DSPersisting, domain.DSLead), saveDone(), domain.DSRolledBack, []string{"recordRoutingError"}},

		// --- persist error routing (persisting, invoke onError) ---
		{"T-DEAL-44", newDeal(domain.DSPersisting), saveErr(model.ErrLocked), domain.DSPersistRetry, []string{"recordError"}},
		{"T-DEAL-45", newDeal(domain.DSPersisting), saveErr(model.ErrConstraint), domain.DSRolledBack, []string{"recordConstraint"}},
		{"T-DEAL-46", newDeal(domain.DSPersisting), saveErr(model.ErrDiskFull), domain.DSRolledBack, []string{"recordDiskFull"}},
		{"T-DEAL-47", newDeal(domain.DSPersisting), saveErr(model.ErrTimeout), domain.DSRolledBack, []string{"recordTimeout"}},
		{"T-DEAL-48", newDeal(domain.DSPersisting), saveErr(model.ErrConflict), domain.DSRolledBack, []string{"recordUnknownError"}},
		{"T-DEAL-49", newDeal(domain.DSPersisting), domain.DealEvent{Kind: domain.DEvPersistTimeout}, domain.DSRolledBack, []string{"recordTimeout"}},

		// --- persistRetry ---
		{"T-DEAL-50", dealRetries(domain.DSPersistRetry, 3), domain.DealEvent{Kind: domain.DEvAlways}, domain.DSRolledBack, []string{"recordRetriesExhausted"}},
		{"T-DEAL-51", dealRetries(domain.DSPersistRetry, 0), domain.DealEvent{Kind: domain.DEvRetryBackoff}, domain.DSPersisting, []string{"incrementRetries"}},

		// --- rolledBack routing (atomic rollback to priorStage; no action) ---
		{"T-DEAL-52", dealPrior(domain.DSLead), domain.DealEvent{Kind: domain.DEvAlways}, domain.DSLead, nil},
		{"T-DEAL-53", dealPrior(domain.DSQualified), domain.DealEvent{Kind: domain.DEvAlways}, domain.DSQualified, nil},
		{"T-DEAL-54", dealPrior(domain.DSProposal), domain.DealEvent{Kind: domain.DEvAlways}, domain.DSProposal, nil},
		{"T-DEAL-55", dealPrior(domain.DSNegotiation), domain.DealEvent{Kind: domain.DEvAlways}, domain.DSNegotiation, nil},
		{"T-DEAL-56", dealPrior(domain.DSWon), domain.DealEvent{Kind: domain.DEvAlways}, domain.DSWon, nil},
		{"T-DEAL-57", dealPrior(domain.DSLost), domain.DealEvent{Kind: domain.DEvAlways}, domain.DSLost, nil},
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
