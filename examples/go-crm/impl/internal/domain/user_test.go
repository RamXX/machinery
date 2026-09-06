package domain_test

// Transition witnesses bind current committed oracle expectations to real Fire results.

import (
	"crm/internal/testoracle"
	"encoding/json"
	"errors"
	"strings"
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
	id    string
	user  *domain.User
	event domain.UserEvent
	rowID string
}

func TestUserTransitions(t *testing.T) {
	disable := domain.UserEvent{Kind: domain.UEvDisable}
	enable := domain.UserEvent{Kind: domain.UEvEnable}
	cs := []userCase{
		{"T-USER-01_USER-e20d04", newUserAgg(domain.USActive, uAdmin), disable, "USER-e20d04"},
		{"T-USER-02_notAdmin_USER-2b2218", newUserAgg(domain.USActive, uMgrT1), disable, "USER-2b2218"},
		{"T-USER-03_USER-0ef83a", newUserAgg(domain.USActive, uAdmin), enable, "USER-0ef83a"},
		{"T-USER-04_USER-e59219", newUserAgg(domain.USDisabled, uAdmin), enable, "USER-e59219"},
		{"T-USER-05_notAdmin_USER-ffd41a", newUserAgg(domain.USDisabled, uMgrT1), enable, "USER-ffd41a"},
		{"T-USER-06_USER-799d7d", newUserAgg(domain.USDisabled, uAdmin), disable, "USER-799d7d"},

		{"T-USER-07_USER-930b15", userPending(domain.USActive), userSaveDone(), "USER-930b15"},
		{"T-USER-08_USER-dd6c98", userPending(domain.USDisabled), userSaveDone(), "USER-dd6c98"},
		{"T-USER-09_USER-7b324b", userPending(domain.UserState("bogus")), userSaveDone(), "USER-7b324b"},

		{"T-USER-10_USER-dde0a6", newUserAgg(domain.USPersisting, uAdmin), userSaveErr(model.ErrLocked), "USER-dde0a6"},
		{"T-USER-11_USER-d2cfe6", newUserAgg(domain.USPersisting, uAdmin), userSaveErr(model.ErrConstraint), "USER-d2cfe6"},
		{"T-USER-12_USER-8e0d4c", newUserAgg(domain.USPersisting, uAdmin), userSaveErr(model.ErrDiskFull), "USER-8e0d4c"},
		{"T-USER-13_USER-388821", newUserAgg(domain.USPersisting, uAdmin), userSaveErr(model.ErrTimeout), "USER-388821"},
		{"T-USER-14_USER-838e85", newUserAgg(domain.USPersisting, uAdmin), userSaveErr(model.ErrConflict), "USER-838e85"},
		{"T-USER-15_USER-a986a8", newUserAgg(domain.USPersisting, uAdmin), domain.UserEvent{Kind: domain.UEvPersistTimeout}, "USER-a986a8"},

		{"T-USER-16_USER-1c13da", userRetries(3), domain.UserEvent{Kind: domain.UEvAlways}, "USER-1c13da"},
		{"T-USER-17_USER-081a5d", userRetries(0), domain.UserEvent{Kind: domain.UEvRetryBackoff}, "USER-081a5d"},

		{"T-USER-18_USER-adccd9", userPrior(domain.USActive), domain.UserEvent{Kind: domain.UEvAlways}, "USER-adccd9"},
		{"T-USER-19_USER-7cf0fc", userPrior(domain.USDisabled), domain.UserEvent{Kind: domain.UEvAlways}, "USER-7cf0fc"},
		{"USER-453743 / fail-closed rollback routing", userPrior(domain.UserState("bogus")), domain.UserEvent{Kind: domain.UEvAlways}, "USER-453743"},
	}
	suite, err := testoracle.Load("../../../design/machines", "User")
	if err != nil {
		t.Fatal(err)
	}
	expectations := make([]testoracle.Expectation, len(cs))
	registrations := make([]testoracle.Registration, len(cs))
	for i, tc := range cs {
		expected, err := suite.Bind(userWitness(t, tc))
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
			got := tc.user.Fire(tc.event)
			if err := expectations[i].Check(string(tc.user.State), got.Actions); err != nil {
				t.Fatal(err)
			}
			if registrations[i].Native != t.Name() {
				t.Fatalf("oracle-native-name: got=%q want=%q", t.Name(), registrations[i].Native)
			}
			raw, err := json.Marshal(testoracle.Observation{Registration: registrations[i], Next: string(tc.user.State), Actions: got.Actions})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("oracle-executed %s", raw)
		})
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

func userWitness(t *testing.T, tc userCase) testoracle.Witness {
	t.Helper()
	d, e := tc.user, tc.event
	admin := d.Actor.Role == model.RoleAdmin
	oracleClause(t, tc.id, "notAdmin", admin)
	all := map[string]bool{
		"guardAdminAuthority": admin, "pendingIsActive": d.PendingStatus == domain.USActive, "pendingIsDisabled": d.PendingStatus == domain.USDisabled,
		"priorIsActive": d.PriorStatus == domain.USActive, "priorIsDisabled": d.PriorStatus == domain.USDisabled,
		"retriesExhausted": d.Retries >= 3, "isErrLocked": errors.Is(e.Err, model.ErrLocked), "isErrConstraint": errors.Is(e.Err, model.ErrConstraint),
		"isErrDiskFull": errors.Is(e.Err, model.ErrDiskFull), "isErrTimeout": errors.Is(e.Err, model.ErrTimeout),
	}
	w := testoracle.Witness{Name: tc.id, RowID: tc.rowID, Source: string(d.State)}
	switch d.State {
	case domain.USActive, domain.USDisabled:
		if e.Kind == domain.UEvEnable || e.Kind == domain.UEvDisable {
			w.Trigger = "on:" + string(e.Kind)
			if d.State == domain.USActive && e.Kind == domain.UEvDisable || d.State == domain.USDisabled && e.Kind == domain.UEvEnable {
				w.Guards = oracleFacts(t, all, "guardAdminAuthority")
			}
			return w
		}
	case domain.USPersisting:
		switch e.Kind {
		case domain.UEvSaveDone:
			w.Trigger = "onDone:saveUser"
			w.Guards = oracleFacts(t, all, "pendingIsActive", "pendingIsDisabled")
			return w
		case domain.UEvSaveError:
			w.Trigger = "onError:saveUser"
			w.Guards = oracleFacts(t, all, "isErrLocked", "isErrConstraint", "isErrDiskFull", "isErrTimeout")
			return w
		case domain.UEvPersistTimeout:
			w.Trigger = "after:persistTimeout"
			return w
		}
	case domain.USPersistRetry:
		switch e.Kind {
		case domain.UEvAlways:
			w.Trigger = "always"
			w.Guards = oracleFacts(t, all, "retriesExhausted")
			return w
		case domain.UEvRetryBackoff:
			w.Trigger = "after:persistRetryBackoff"
			return w
		}
	case domain.USRolledBack:
		if e.Kind == domain.UEvAlways {
			w.Trigger = "always"
			w.Guards = oracleFacts(t, all, "priorIsActive", "priorIsDisabled")
			return w
		}
	}
	t.Fatalf("oracle-event user: unsupported actual source=%q event=%q", d.State, e.Kind)
	return w
}
