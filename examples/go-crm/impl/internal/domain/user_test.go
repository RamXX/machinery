package domain_test

// User status-lifecycle transition oracle. One case per BUILD.md 7.1 T-USER row.
// Source: machines/User.matrix.md. guardAdminAuthority is a single clause
// (actor.role == Admin), so each guard-false row has one falsifying case.

import (
	"testing"

	"crm/internal/authz"
	"crm/internal/domain"
	"crm/internal/model"
)

func newUserAgg(state domain.UserState, actor model.User) *domain.User {
	return &domain.User{
		State:  state,
		UserID: "u1",
		Actor:  actor,
		Authz:  authz.New(),
	}
}

func userPending(pending domain.UserState) *domain.User {
	u := newUserAgg(domain.USPersisting, uAdmin)
	u.PendingStatus = pending
	return u
}

func userPrior(prior domain.UserState) *domain.User {
	u := newUserAgg(domain.USRolledBack, uAdmin)
	u.PriorStatus = prior
	return u
}

type userCase struct {
	id      string
	user    *domain.User
	event   domain.UserEvent
	want    domain.UserState
	actions []string
}

func userCases() []userCase {
	disable := domain.UserEvent{Kind: domain.UEvDisable}
	enable := domain.UserEvent{Kind: domain.UEvEnable}
	return []userCase{
		{"T-USER-01", newUserAgg(domain.USActive, uAdmin), disable, domain.USPersisting, []string{"setPendingDisable"}},
		{"T-USER-02_notAdmin", newUserAgg(domain.USActive, uMgrT1), disable, domain.USActive, []string{"recordAuthorityDenied"}},
		{"T-USER-03", newUserAgg(domain.USActive, uAdmin), enable, domain.USActive, []string{"recordAlreadyActive"}},
		{"T-USER-04", newUserAgg(domain.USDisabled, uAdmin), enable, domain.USPersisting, []string{"setPendingEnable"}},
		{"T-USER-05_notAdmin", newUserAgg(domain.USDisabled, uMgrT1), enable, domain.USDisabled, []string{"recordAuthorityDenied"}},
		{"T-USER-06", newUserAgg(domain.USDisabled, uAdmin), disable, domain.USDisabled, []string{"recordAlreadyDisabled"}},

		{"T-USER-07", userPending(domain.USActive), userSaveDone(), domain.USActive, []string{"commitStatus"}},
		{"T-USER-08", userPending(domain.USDisabled), userSaveDone(), domain.USDisabled, []string{"commitStatus"}},
		{"T-USER-09", userPending(domain.UserState("bogus")), userSaveDone(), domain.USRolledBack, []string{"recordRoutingError"}},

		{"T-USER-10", newUserAgg(domain.USPersisting, uAdmin), userSaveErr(model.ErrLocked), domain.USPersistRetry, []string{"recordError"}},
		{"T-USER-11", newUserAgg(domain.USPersisting, uAdmin), userSaveErr(model.ErrConstraint), domain.USRolledBack, []string{"recordConstraint"}},
		{"T-USER-12", newUserAgg(domain.USPersisting, uAdmin), userSaveErr(model.ErrDiskFull), domain.USRolledBack, []string{"recordDiskFull"}},
		{"T-USER-13", newUserAgg(domain.USPersisting, uAdmin), userSaveErr(model.ErrTimeout), domain.USRolledBack, []string{"recordTimeout"}},
		{"T-USER-14", newUserAgg(domain.USPersisting, uAdmin), userSaveErr(model.ErrConflict), domain.USRolledBack, []string{"recordUnknownError"}},
		{"T-USER-15", newUserAgg(domain.USPersisting, uAdmin), domain.UserEvent{Kind: domain.UEvPersistTimeout}, domain.USRolledBack, []string{"recordTimeout"}},

		{"T-USER-16", userRetries(3), domain.UserEvent{Kind: domain.UEvAlways}, domain.USRolledBack, []string{"recordRetriesExhausted"}},
		{"T-USER-17", userRetries(0), domain.UserEvent{Kind: domain.UEvRetryBackoff}, domain.USPersisting, []string{"incrementRetries"}},

		{"T-USER-18", userPrior(domain.USActive), domain.UserEvent{Kind: domain.UEvAlways}, domain.USActive, nil},
		{"T-USER-19", userPrior(domain.USDisabled), domain.UserEvent{Kind: domain.UEvAlways}, domain.USDisabled, nil},
	}
}

func userRetries(n int) *domain.User {
	u := newUserAgg(domain.USPersistRetry, uAdmin)
	u.Retries = n
	return u
}

func userSaveDone() domain.UserEvent { return domain.UserEvent{Kind: domain.UEvSaveDone} }
func userSaveErr(e error) domain.UserEvent {
	return domain.UserEvent{Kind: domain.UEvSaveError, Err: e}
}

func TestUserTransitions(t *testing.T) {
	for _, tc := range userCases() {
		t.Run(tc.id, func(t *testing.T) {
			got := tc.user.Fire(tc.event)
			if tc.user.State != tc.want {
				t.Errorf("%s: next state = %q, want %q", tc.id, tc.user.State, tc.want)
			}
			if !firedInOrder(got.Actions, tc.actions) {
				t.Errorf("%s: actions = %v, want (in order) %v", tc.id, got.Actions, tc.actions)
			}
		})
	}
}
