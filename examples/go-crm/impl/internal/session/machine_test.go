package session_test

// Transition witnesses bind current committed oracle expectations to real Fire results.

import (
	"crm/internal/testoracle"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"crm/internal/model"
	"crm/internal/session"
)

func past() time.Time   { return time.Now().Add(-time.Hour) }
func future() time.Time { return time.Now().Add(time.Hour) }

type sessCase struct {
	id      string
	machine *session.SessionMachine
	event   session.SessionEvent
	rowID   string
}

func sm(state session.SessionState) *session.SessionMachine {
	return &session.SessionMachine{State: state}
}
func smRetries(state session.SessionState, n int) *session.SessionMachine {
	return &session.SessionMachine{State: state, Retries: n}
}

func TestSessionTransitions(t *testing.T) {
	login := session.SessionEvent{Kind: session.SEvLogin, Username: "u", Password: "p"}
	resume := session.SessionEvent{Kind: session.SEvResume}
	logout := session.SessionEvent{Kind: session.SEvLogout}
	use := session.SessionEvent{Kind: session.SEvUseSession}
	done := session.SessionEvent{Kind: session.SEvInvokeDone}
	doneActive := session.SessionEvent{Kind: session.SEvInvokeDone, User: model.User{ID: "u1", Status: model.StatusActive}}
	doneDisabled := session.SessionEvent{Kind: session.SEvInvokeDone, User: model.User{ID: "u1", Status: model.StatusDisabled}}
	doneTokenValid := session.SessionEvent{Kind: session.SEvInvokeDone, Token: model.Session{UserID: "u1", ExpiresAt: future()}}
	doneTokenExpired := session.SessionEvent{Kind: session.SEvInvokeDone, Token: model.Session{UserID: "u1", ExpiresAt: past()}}
	errBad := session.SessionEvent{Kind: session.SEvInvokeError, Err: model.ErrBadCredentials}
	errDisabled := session.SessionEvent{Kind: session.SEvInvokeError, Err: model.ErrDisabled}
	errLocked := session.SessionEvent{Kind: session.SEvInvokeError, Err: model.ErrLocked}
	errOther := session.SessionEvent{Kind: session.SEvInvokeError, Err: model.ErrUnavailable}
	errNoSession := session.SessionEvent{Kind: session.SEvInvokeError, Err: model.ErrNoSession}
	errExpired := session.SessionEvent{Kind: session.SEvInvokeError, Err: model.ErrExpired}
	errUnreadable := session.SessionEvent{Kind: session.SEvInvokeError, Err: model.ErrUnreadable}
	errNotFound := session.SessionEvent{Kind: session.SEvInvokeError, Err: model.ErrNotFound}

	cs := []sessCase{
		// --- Anonymous ---
		{"T-SESS-01_SESS-ee5c17", sm(session.SAnonymous), login, "SESS-ee5c17"},
		{"T-SESS-02_SESS-f3cc5e", sm(session.SAnonymous), resume, "SESS-f3cc5e"},
		{"T-SESS-03_SESS-27e1b6", sm(session.SAnonymous), logout, "SESS-27e1b6"},
		{"T-SESS-04_SESS-fd60bd", sm(session.SAnonymous), use, "SESS-fd60bd"},

		// --- Authenticating (verifyCredentials) ---
		{"T-SESS-05_SESS-abcf53", sm(session.SAuthenticating), doneDisabled, "SESS-abcf53"},
		{"T-SESS-06_SESS-c78e53", sm(session.SAuthenticating), doneActive, "SESS-c78e53"},
		{"T-SESS-07_SESS-fbf094", sm(session.SAuthenticating), errBad, "SESS-fbf094"},
		{"T-SESS-08_SESS-e12d58", sm(session.SAuthenticating), errDisabled, "SESS-e12d58"},
		{"T-SESS-09_SESS-2dee87", sm(session.SAuthenticating), errLocked, "SESS-2dee87"},
		{"T-SESS-10_SESS-9c998d", sm(session.SAuthenticating), errOther, "SESS-9c998d"},
		{"T-SESS-11_SESS-63844a", sm(session.SAuthenticating), session.SessionEvent{Kind: session.SEvVerifyTimeout}, "SESS-63844a"},

		// --- VerifyRetry ---
		{"T-SESS-12_SESS-aaa861", smRetries(session.SVerifyRetry, 3), session.SessionEvent{Kind: session.SEvAlways}, "SESS-aaa861"},
		{"T-SESS-13_SESS-316622", smRetries(session.SVerifyRetry, 0), session.SessionEvent{Kind: session.SEvVerifyRetryBackoff}, "SESS-316622"},

		// --- WritingSession (writeSessionFile) ---
		{"T-SESS-14_SESS-b95638", sm(session.SWritingSession), done, "SESS-b95638"},
		{"T-SESS-15_SESS-7b8023", sm(session.SWritingSession), session.SessionEvent{Kind: session.SEvInvokeError, Err: model.ErrUnavailable}, "SESS-7b8023"},
		{"T-SESS-16_SESS-402e07", sm(session.SWritingSession), session.SessionEvent{Kind: session.SEvFileIoTimeout}, "SESS-402e07"},

		// --- Resolving (readSessionFile) ---
		{"T-SESS-17_SESS-e01d44", sm(session.SResolving), doneTokenExpired, "SESS-e01d44"},
		{"T-SESS-18_SESS-e6484d", sm(session.SResolving), doneTokenValid, "SESS-e6484d"},
		{"T-SESS-19_SESS-4f7245", sm(session.SResolving), errNoSession, "SESS-4f7245"},
		{"T-SESS-20_SESS-dddfd5", sm(session.SResolving), errExpired, "SESS-dddfd5"},
		{"T-SESS-21_SESS-12a601", sm(session.SResolving), errUnreadable, "SESS-12a601"},
		{"T-SESS-22_SESS-c552bf", sm(session.SResolving), session.SessionEvent{Kind: session.SEvFileIoTimeout}, "SESS-c552bf"},

		// --- CheckingUser (loadUser) ---
		{"T-SESS-23_SESS-85613f", sm(session.SCheckingUser), doneActive, "SESS-85613f"},
		{"T-SESS-24_SESS-55a08c", sm(session.SCheckingUser), doneDisabled, "SESS-55a08c"},
		{"T-SESS-25_SESS-10b95d", sm(session.SCheckingUser), errLocked, "SESS-10b95d"},
		{"T-SESS-26_SESS-54c2ea", sm(session.SCheckingUser), errNotFound, "SESS-54c2ea"},
		{"T-SESS-27_SESS-a337ab", sm(session.SCheckingUser), errOther, "SESS-a337ab"},
		{"T-SESS-28_SESS-aafc2f", sm(session.SCheckingUser), session.SessionEvent{Kind: session.SEvLoadUserTimeout}, "SESS-aafc2f"},

		// --- Active ---
		{"T-SESS-29_SESS-69da3c", sm(session.SActive), logout, "SESS-69da3c"},
		{"T-SESS-30_SESS-448752", sm(session.SActive), use, "SESS-448752"},
		{"T-SESS-31_SESS-fd231f", sm(session.SActive), login, "SESS-fd231f"},
		{"T-SESS-32_SESS-7fed3c", sm(session.SActive), resume, "SESS-7fed3c"},
		{"T-SESS-33_SESS-87f3f3", sm(session.SActive), session.SessionEvent{Kind: session.SEvSessionTTL}, "SESS-87f3f3"},

		// --- LoggingOut (clearSessionFile) ---
		{"T-SESS-34_SESS-706156", sm(session.SLoggingOut), done, "SESS-706156"},
		{"T-SESS-35_SESS-51ded9", sm(session.SLoggingOut), session.SessionEvent{Kind: session.SEvInvokeError, Err: model.ErrUnavailable}, "SESS-51ded9"},
		{"T-SESS-36_SESS-927687", sm(session.SLoggingOut), session.SessionEvent{Kind: session.SEvFileIoTimeout}, "SESS-927687"},

		// --- Expired ---
		{"T-SESS-37_SESS-90aa72", sm(session.SExpired), login, "SESS-90aa72"},
		{"T-SESS-38_SESS-5e1b28", sm(session.SExpired), resume, "SESS-5e1b28"},
		{"T-SESS-39_SESS-2587f3", sm(session.SExpired), logout, "SESS-2587f3"},
		{"T-SESS-40_SESS-25fc00", sm(session.SExpired), use, "SESS-25fc00"},

		// --- LoggedOut ---
		{"T-SESS-41_SESS-bb2221", sm(session.SLoggedOut), login, "SESS-bb2221"},
		{"T-SESS-42_SESS-3cee4a", sm(session.SLoggedOut), resume, "SESS-3cee4a"},
		{"T-SESS-43_SESS-656e49", sm(session.SLoggedOut), logout, "SESS-656e49"},
		{"T-SESS-44_SESS-f09ff1", sm(session.SLoggedOut), use, "SESS-f09ff1"},

		// --- AuthFailed ---
		{"T-SESS-45_SESS-61d379", sm(session.SAuthFailed), login, "SESS-61d379"},
		{"T-SESS-46_SESS-f821b7", sm(session.SAuthFailed), resume, "SESS-f821b7"},
		{"T-SESS-47_SESS-65b326", sm(session.SAuthFailed), logout, "SESS-65b326"},
		{"T-SESS-48_SESS-aca1a6", sm(session.SAuthFailed), use, "SESS-aca1a6"},

		// --- AuthDenied ---
		{"T-SESS-49_SESS-5a47c2", sm(session.SAuthDenied), login, "SESS-5a47c2"},
		{"T-SESS-50_SESS-22bc20", sm(session.SAuthDenied), resume, "SESS-22bc20"},
		{"T-SESS-51_SESS-3e386e", sm(session.SAuthDenied), logout, "SESS-3e386e"},
		{"T-SESS-52_SESS-75e8c1", sm(session.SAuthDenied), use, "SESS-75e8c1"},

		// --- Invalidated ---
		{"T-SESS-53_SESS-f59ab7", sm(session.SInvalidated), login, "SESS-f59ab7"},
		{"T-SESS-54_SESS-134dea", sm(session.SInvalidated), resume, "SESS-134dea"},
		{"T-SESS-55_SESS-de6aa7", sm(session.SInvalidated), logout, "SESS-de6aa7"},
		{"T-SESS-56_SESS-19fd5c", sm(session.SInvalidated), use, "SESS-19fd5c"},

		// --- SessionUnavailable ---
		{"T-SESS-57_SESS-e7bef7", sm(session.SSessionUnavailable), login, "SESS-e7bef7"},
		{"T-SESS-58_SESS-0488b7", sm(session.SSessionUnavailable), resume, "SESS-0488b7"},
		{"T-SESS-59_SESS-f6a536", sm(session.SSessionUnavailable), logout, "SESS-f6a536"},
		{"T-SESS-60_SESS-b830a0", sm(session.SSessionUnavailable), use, "SESS-b830a0"},
	}
	suite, err := testoracle.Load("../../../design/machines", "Session")
	if err != nil {
		t.Fatal(err)
	}
	expectations := make([]testoracle.Expectation, len(cs))
	registrations := make([]testoracle.Registration, len(cs))
	for i, tc := range cs {
		expected, err := suite.Bind(sessionWitness(t, tc))
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
			got := tc.machine.Fire(tc.event)
			if err := expectations[i].Check(string(tc.machine.State), got.Actions); err != nil {
				t.Fatal(err)
			}
			if registrations[i].Native != t.Name() {
				t.Fatalf("oracle-native-name: got=%q want=%q", t.Name(), registrations[i].Native)
			}
			raw, err := json.Marshal(testoracle.Observation{Registration: registrations[i], Next: string(tc.machine.State), Actions: got.Actions})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("oracle-executed %s", raw)
		})
	}
}

func sessionWitness(t *testing.T, tc sessCase) testoracle.Witness {
	t.Helper()
	d, e := tc.machine, tc.event
	now := time.Now()
	if d.State == session.SResolving && e.Kind == session.SEvInvokeDone && e.Token.ExpiresAt.Sub(now) > -time.Minute && e.Token.ExpiresAt.Sub(now) < time.Minute {
		t.Fatal("oracle-clause session: expiry witness is too near the boundary")
	}
	all := map[string]bool{
		"guardUserDisabled": e.User.Status == model.StatusDisabled, "guardSessionUserActive": e.User.Status == model.StatusActive,
		"guardSessionExpired": !e.Token.ExpiresAt.After(now), "retriesExhausted": d.Retries >= 3,
		"isErrBadCredentials": errors.Is(e.Err, model.ErrBadCredentials), "isErrDisabled": errors.Is(e.Err, model.ErrDisabled),
		"isErrLocked": errors.Is(e.Err, model.ErrLocked), "isErrNoSession": errors.Is(e.Err, model.ErrNoSession),
		"isErrExpired": errors.Is(e.Err, model.ErrExpired), "isErrNotFound": errors.Is(e.Err, model.ErrNotFound),
	}
	facts := func(keys ...string) map[string]bool {
		out := map[string]bool{}
		for _, key := range keys {
			v, ok := all[key]
			if !ok {
				t.Fatalf("oracle-guard-key session: %s", key)
			}
			out[key] = v
		}
		return out
	}
	w := testoracle.Witness{Name: tc.id, RowID: tc.rowID, Source: string(d.State)}
	switch d.State {
	case session.SAnonymous, session.SActive, session.SExpired, session.SLoggedOut, session.SAuthFailed, session.SAuthDenied, session.SInvalidated, session.SSessionUnavailable:
		switch e.Kind {
		case session.SEvLogin, session.SEvResume, session.SEvLogout, session.SEvUseSession:
			w.Trigger = "on:" + string(e.Kind)
			return w
		case session.SEvSessionTTL:
			if d.State == session.SActive {
				w.Trigger = "after:sessionTTL"
				return w
			}
		}
	case session.SVerifyRetry:
		switch e.Kind {
		case session.SEvAlways:
			w.Trigger = "always"
			w.Guards = facts("retriesExhausted")
			return w
		case session.SEvVerifyRetryBackoff:
			w.Trigger = "after:verifyRetryBackoff"
			return w
		}
	case session.SAuthenticating, session.SWritingSession, session.SResolving, session.SCheckingUser, session.SLoggingOut:
		invoke := ""
		delay := ""
		var done, errorKeys []string
		switch d.State {
		case session.SAuthenticating:
			invoke = "verifyCredentials"
			delay = "verifyTimeout"
			done = []string{"guardUserDisabled"}
			errorKeys = []string{"isErrBadCredentials", "isErrDisabled", "isErrLocked"}
		case session.SWritingSession:
			invoke = "writeSessionFile"
			delay = "fileIoTimeout"
		case session.SResolving:
			invoke = "readSessionFile"
			delay = "fileIoTimeout"
			done = []string{"guardSessionExpired"}
			errorKeys = []string{"isErrNoSession", "isErrExpired"}
		case session.SCheckingUser:
			invoke = "loadUser"
			delay = "loadUserTimeout"
			done = []string{"guardSessionUserActive"}
			errorKeys = []string{"isErrLocked", "isErrNotFound"}
		case session.SLoggingOut:
			invoke = "clearSessionFile"
			delay = "fileIoTimeout"
		}
		switch e.Kind {
		case session.SEvInvokeDone:
			w.Trigger = "onDone:" + invoke
			w.Guards = facts(done...)
			return w
		case session.SEvInvokeError:
			w.Trigger = "onError:" + invoke
			w.Guards = facts(errorKeys...)
			return w
		case session.SEvVerifyTimeout, session.SEvFileIoTimeout, session.SEvLoadUserTimeout:
			if string(e.Kind) == delay {
				w.Trigger = "after:" + delay
				return w
			}
		}
	}
	t.Fatalf("oracle-event session: unsupported actual source=%q event=%q", d.State, e.Kind)
	return w
}
