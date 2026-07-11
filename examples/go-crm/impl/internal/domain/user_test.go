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
		{"T-USER-01_USER-e20d04", newUserAgg(domain.USActive, uAdmin), disable, domain.USPersisting, []string{"setPendingDisable"}},
		{"T-USER-02_notAdmin_USER-2b2218", newUserAgg(domain.USActive, uMgrT1), disable, domain.USActive, []string{"recordAuthorityDenied"}},
		{"T-USER-03_USER-0ef83a", newUserAgg(domain.USActive, uAdmin), enable, domain.USActive, []string{"recordAlreadyActive"}},
		{"T-USER-04_USER-e59219", newUserAgg(domain.USDisabled, uAdmin), enable, domain.USPersisting, []string{"setPendingEnable"}},
		{"T-USER-05_notAdmin_USER-ffd41a", newUserAgg(domain.USDisabled, uMgrT1), enable, domain.USDisabled, []string{"recordAuthorityDenied"}},
		{"T-USER-06_USER-799d7d", newUserAgg(domain.USDisabled, uAdmin), disable, domain.USDisabled, []string{"recordAlreadyDisabled"}},

		{"T-USER-07_USER-930b15", userPending(domain.USActive), userSaveDone(), domain.USActive, []string{"commitStatus"}},
		{"T-USER-08_USER-dd6c98", userPending(domain.USDisabled), userSaveDone(), domain.USDisabled, []string{"commitStatus"}},
		{"T-USER-09_USER-7b324b", userPending(domain.UserState("bogus")), userSaveDone(), domain.USRolledBack, []string{"recordRoutingError"}},

		{"T-USER-10_USER-dde0a6", newUserAgg(domain.USPersisting, uAdmin), userSaveErr(model.ErrLocked), domain.USPersistRetry, []string{"recordError"}},
		{"T-USER-11_USER-d2cfe6", newUserAgg(domain.USPersisting, uAdmin), userSaveErr(model.ErrConstraint), domain.USRolledBack, []string{"recordConstraint"}},
		{"T-USER-12_USER-8e0d4c", newUserAgg(domain.USPersisting, uAdmin), userSaveErr(model.ErrDiskFull), domain.USRolledBack, []string{"recordDiskFull"}},
		{"T-USER-13_USER-388821", newUserAgg(domain.USPersisting, uAdmin), userSaveErr(model.ErrTimeout), domain.USRolledBack, []string{"recordTimeout"}},
		{"T-USER-14_USER-838e85", newUserAgg(domain.USPersisting, uAdmin), userSaveErr(model.ErrConflict), domain.USRolledBack, []string{"recordUnknownError"}},
		{"T-USER-15_USER-a986a8", newUserAgg(domain.USPersisting, uAdmin), domain.UserEvent{Kind: domain.UEvPersistTimeout}, domain.USRolledBack, []string{"recordTimeout"}},

		{"T-USER-16_USER-1c13da", userRetries(3), domain.UserEvent{Kind: domain.UEvAlways}, domain.USRolledBack, []string{"recordRetriesExhausted"}},
		{"T-USER-17_USER-081a5d", userRetries(0), domain.UserEvent{Kind: domain.UEvRetryBackoff}, domain.USPersisting, []string{"incrementRetries"}},

		{"T-USER-18_USER-adccd9", userPrior(domain.USActive), domain.UserEvent{Kind: domain.UEvAlways}, domain.USActive, nil},
		{"T-USER-19_USER-7cf0fc", userPrior(domain.USDisabled), domain.UserEvent{Kind: domain.UEvAlways}, domain.USDisabled, nil},
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
