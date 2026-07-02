package session_test

// Session machine transition oracle. One table case per BUILD.md 7.1 T-SESS row.
// Source: machines/Session.matrix.md. Each case sets up the given state (+context
// for the guarded rows: verified/loaded user status, token expiry, retry count),
// fires the trigger, and asserts the next state and that the row's actions fired
// in order. Against the SessionMachine.Fire stub every row is RED.

import (
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
	want    session.SessionState
	actions []string
}

func sm(state session.SessionState) *session.SessionMachine {
	return &session.SessionMachine{State: state}
}
func smRetries(state session.SessionState, n int) *session.SessionMachine {
	return &session.SessionMachine{State: state, Retries: n}
}

func firedInOrder(got, want []string) bool {
	i := 0
	for _, g := range got {
		if i < len(want) && g == want[i] {
			i++
		}
	}
	return i == len(want)
}

func sessCases() []sessCase {
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

	return []sessCase{
		// --- Anonymous ---
		{"T-SESS-01", sm(session.SAnonymous), login, session.SAuthenticating, []string{"setCredentials"}},
		{"T-SESS-02", sm(session.SAnonymous), resume, session.SResolving, nil},
		{"T-SESS-03", sm(session.SAnonymous), logout, session.SAnonymous, []string{"recordNoSessionToLogout"}},
		{"T-SESS-04", sm(session.SAnonymous), use, session.SAnonymous, []string{"recordNoActiveSession"}},

		// --- Authenticating (verifyCredentials) ---
		{"T-SESS-05", sm(session.SAuthenticating), doneDisabled, session.SAuthDenied, []string{"recordDisabled"}},
		{"T-SESS-06", sm(session.SAuthenticating), doneActive, session.SWritingSession, []string{"captureUser"}},
		{"T-SESS-07", sm(session.SAuthenticating), errBad, session.SAuthFailed, []string{"recordBadCredentials"}},
		{"T-SESS-08", sm(session.SAuthenticating), errDisabled, session.SAuthDenied, []string{"recordDisabled"}},
		{"T-SESS-09", sm(session.SAuthenticating), errLocked, session.SVerifyRetry, []string{"recordError"}},
		{"T-SESS-10", sm(session.SAuthenticating), errOther, session.SSessionUnavailable, []string{"recordVerifyError"}},
		{"T-SESS-11", sm(session.SAuthenticating), session.SessionEvent{Kind: session.SEvVerifyTimeout}, session.SSessionUnavailable, []string{"recordTimeout"}},

		// --- VerifyRetry ---
		{"T-SESS-12", smRetries(session.SVerifyRetry, 3), session.SessionEvent{Kind: session.SEvAlways}, session.SSessionUnavailable, []string{"recordRetriesExhausted"}},
		{"T-SESS-13", smRetries(session.SVerifyRetry, 0), session.SessionEvent{Kind: session.SEvVerifyRetryBackoff}, session.SAuthenticating, []string{"incrementRetries"}},

		// --- WritingSession (writeSessionFile) ---
		{"T-SESS-14", sm(session.SWritingSession), done, session.SActive, nil},
		{"T-SESS-15", sm(session.SWritingSession), session.SessionEvent{Kind: session.SEvInvokeError, Err: model.ErrUnavailable}, session.SSessionUnavailable, []string{"recordFileError"}},
		{"T-SESS-16", sm(session.SWritingSession), session.SessionEvent{Kind: session.SEvFileIoTimeout}, session.SSessionUnavailable, []string{"recordTimeout"}},

		// --- Resolving (readSessionFile) ---
		{"T-SESS-17", sm(session.SResolving), doneTokenExpired, session.SExpired, []string{"recordExpired"}},
		{"T-SESS-18", sm(session.SResolving), doneTokenValid, session.SCheckingUser, []string{"captureToken"}},
		{"T-SESS-19", sm(session.SResolving), errNoSession, session.SAnonymous, []string{"recordNoSession"}},
		{"T-SESS-20", sm(session.SResolving), errExpired, session.SExpired, []string{"recordExpired"}},
		{"T-SESS-21", sm(session.SResolving), errUnreadable, session.SSessionUnavailable, []string{"recordFileError"}},
		{"T-SESS-22", sm(session.SResolving), session.SessionEvent{Kind: session.SEvFileIoTimeout}, session.SSessionUnavailable, []string{"recordTimeout"}},

		// --- CheckingUser (loadUser) ---
		{"T-SESS-23", sm(session.SCheckingUser), doneActive, session.SActive, []string{"captureUser"}},
		{"T-SESS-24", sm(session.SCheckingUser), doneDisabled, session.SInvalidated, []string{"recordUserNotActive"}},
		{"T-SESS-25", sm(session.SCheckingUser), errLocked, session.SVerifyRetry, []string{"recordError"}},
		{"T-SESS-26", sm(session.SCheckingUser), errNotFound, session.SInvalidated, []string{"recordUserMissing"}},
		{"T-SESS-27", sm(session.SCheckingUser), errOther, session.SSessionUnavailable, []string{"recordLoadError"}},
		{"T-SESS-28", sm(session.SCheckingUser), session.SessionEvent{Kind: session.SEvLoadUserTimeout}, session.SSessionUnavailable, []string{"recordTimeout"}},

		// --- Active ---
		{"T-SESS-29", sm(session.SActive), logout, session.SLoggingOut, nil},
		{"T-SESS-30", sm(session.SActive), use, session.SActive, []string{"recordSessionUsed"}},
		{"T-SESS-31", sm(session.SActive), login, session.SActive, []string{"recordAlreadyActive"}},
		{"T-SESS-32", sm(session.SActive), resume, session.SActive, []string{"recordAlreadyResolved"}},
		{"T-SESS-33", sm(session.SActive), session.SessionEvent{Kind: session.SEvSessionTTL}, session.SExpired, []string{"recordExpired"}},

		// --- LoggingOut (clearSessionFile) ---
		{"T-SESS-34", sm(session.SLoggingOut), done, session.SLoggedOut, nil},
		{"T-SESS-35", sm(session.SLoggingOut), session.SessionEvent{Kind: session.SEvInvokeError, Err: model.ErrUnavailable}, session.SLoggedOut, []string{"recordLogoutWarning"}},
		{"T-SESS-36", sm(session.SLoggingOut), session.SessionEvent{Kind: session.SEvFileIoTimeout}, session.SLoggedOut, []string{"recordLogoutWarning"}},

		// --- Expired ---
		{"T-SESS-37", sm(session.SExpired), login, session.SAuthenticating, []string{"setCredentials"}},
		{"T-SESS-38", sm(session.SExpired), resume, session.SExpired, []string{"recordExpiredNeedsLogin"}},
		{"T-SESS-39", sm(session.SExpired), logout, session.SExpired, []string{"recordNoSessionToLogout"}},
		{"T-SESS-40", sm(session.SExpired), use, session.SExpired, []string{"recordSessionExpired"}},

		// --- LoggedOut ---
		{"T-SESS-41", sm(session.SLoggedOut), login, session.SAuthenticating, []string{"setCredentials"}},
		{"T-SESS-42", sm(session.SLoggedOut), resume, session.SLoggedOut, []string{"recordNoSession"}},
		{"T-SESS-43", sm(session.SLoggedOut), logout, session.SLoggedOut, []string{"recordNoSessionToLogout"}},
		{"T-SESS-44", sm(session.SLoggedOut), use, session.SLoggedOut, []string{"recordNoActiveSession"}},

		// --- AuthFailed ---
		{"T-SESS-45", sm(session.SAuthFailed), login, session.SAuthenticating, []string{"setCredentials"}},
		{"T-SESS-46", sm(session.SAuthFailed), resume, session.SAuthFailed, []string{"recordNoSession"}},
		{"T-SESS-47", sm(session.SAuthFailed), logout, session.SAuthFailed, []string{"recordNoSessionToLogout"}},
		{"T-SESS-48", sm(session.SAuthFailed), use, session.SAuthFailed, []string{"recordNoActiveSession"}},

		// --- AuthDenied ---
		{"T-SESS-49", sm(session.SAuthDenied), login, session.SAuthenticating, []string{"setCredentials"}},
		{"T-SESS-50", sm(session.SAuthDenied), resume, session.SAuthDenied, []string{"recordNoSession"}},
		{"T-SESS-51", sm(session.SAuthDenied), logout, session.SAuthDenied, []string{"recordNoSessionToLogout"}},
		{"T-SESS-52", sm(session.SAuthDenied), use, session.SAuthDenied, []string{"recordNoActiveSession"}},

		// --- Invalidated ---
		{"T-SESS-53", sm(session.SInvalidated), login, session.SAuthenticating, []string{"setCredentials"}},
		{"T-SESS-54", sm(session.SInvalidated), resume, session.SInvalidated, []string{"recordNoSession"}},
		{"T-SESS-55", sm(session.SInvalidated), logout, session.SInvalidated, []string{"recordNoSessionToLogout"}},
		{"T-SESS-56", sm(session.SInvalidated), use, session.SInvalidated, []string{"recordNoActiveSession"}},

		// --- SessionUnavailable ---
		{"T-SESS-57", sm(session.SSessionUnavailable), login, session.SAuthenticating, []string{"setCredentials"}},
		{"T-SESS-58", sm(session.SSessionUnavailable), resume, session.SResolving, nil},
		{"T-SESS-59", sm(session.SSessionUnavailable), logout, session.SSessionUnavailable, []string{"recordNoSessionToLogout"}},
		{"T-SESS-60", sm(session.SSessionUnavailable), use, session.SSessionUnavailable, []string{"recordNoActiveSession"}},
	}
}

func TestSessionTransitions(t *testing.T) {
	for _, tc := range sessCases() {
		t.Run(tc.id, func(t *testing.T) {
			got := tc.machine.Fire(tc.event)
			if tc.machine.State != tc.want {
				t.Errorf("%s: next state = %q, want %q", tc.id, tc.machine.State, tc.want)
			}
			if !firedInOrder(got.Actions, tc.actions) {
				t.Errorf("%s: actions = %v, want (in order) %v", tc.id, got.Actions, tc.actions)
			}
		})
	}
}
