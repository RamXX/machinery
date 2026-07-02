package session

import (
	"errors"
	"time"

	"crm/internal/model"
)

// SessionState is the Session machine state (BUILD.md 5.4, Session.machine.json).
type SessionState string

const (
	SAnonymous          SessionState = "Anonymous"
	SAuthenticating     SessionState = "Authenticating"
	SVerifyRetry        SessionState = "VerifyRetry"
	SWritingSession     SessionState = "WritingSession"
	SResolving          SessionState = "Resolving"
	SCheckingUser       SessionState = "CheckingUser"
	SActive             SessionState = "Active"
	SLoggingOut         SessionState = "LoggingOut"
	SExpired            SessionState = "Expired"
	SLoggedOut          SessionState = "LoggedOut"
	SAuthFailed         SessionState = "AuthFailed"
	SAuthDenied         SessionState = "AuthDenied"
	SInvalidated        SessionState = "Invalidated"
	SSessionUnavailable SessionState = "SessionUnavailable"
)

// SessionEventKind is the trigger for a Session transition. User events
// (login/resume/logout/useSession), invoke results (onDone/onError), the named
// after delays, and the always condition (Session.machine.json).
type SessionEventKind string

const (
	SEvLogin      SessionEventKind = "login"
	SEvResume     SessionEventKind = "resume"
	SEvLogout     SessionEventKind = "logout"
	SEvUseSession SessionEventKind = "useSession"

	SEvInvokeDone  SessionEventKind = "invokeDone"
	SEvInvokeError SessionEventKind = "invokeError"

	SEvVerifyTimeout      SessionEventKind = "verifyTimeout"
	SEvVerifyRetryBackoff SessionEventKind = "verifyRetryBackoff"
	SEvFileIoTimeout      SessionEventKind = "fileIoTimeout"
	SEvLoadUserTimeout    SessionEventKind = "loadUserTimeout"
	SEvSessionTTL         SessionEventKind = "sessionTTL"
	SEvAlways             SessionEventKind = "always"
)

// SessionEvent is a single trigger. User carries the verified/loaded user on an
// invoke onDone; Token carries the parsed token; Err carries the classified
// error on an invoke onError.
type SessionEvent struct {
	Kind     SessionEventKind
	Username string
	Password string
	User     model.User
	Token    model.Session
	Err      error
}

// SessionMachine is the explicit Session state machine that the T-SESS oracle
// (BUILD.md 7.1 T-SESS-01..60) exercises. It mirrors Session.machine.json. The
// Sessions boundary API (session.go) drives this machine internally.
type SessionMachine struct {
	State      SessionState
	Username   string
	UserID     string
	Role       model.UserRole
	TeamID     string
	UserStatus model.UserStatus
	ExpiresAt  time.Time
	Retries    int
	LastError  error
	Reason     string
}

// Fire applies an event, mirroring Session.machine.json (BUILD.md 7.1
// T-SESS-01..60). Every state handles every event explicitly (a no-session state
// rejects useSession/logout rather than silently ignoring).
func (m *SessionMachine) Fire(evt SessionEvent) model.Effect {
	switch m.State {
	case SAnonymous:
		return m.fireAnonymous(evt)
	case SAuthenticating:
		return m.fireAuthenticating(evt)
	case SVerifyRetry:
		return m.fireVerifyRetry(evt)
	case SWritingSession:
		return m.fireWritingSession(evt)
	case SResolving:
		return m.fireResolving(evt)
	case SCheckingUser:
		return m.fireCheckingUser(evt)
	case SActive:
		return m.fireActive(evt)
	case SLoggingOut:
		return m.fireLoggingOut(evt)
	case SExpired:
		return m.fireExpired(evt)
	case SLoggedOut:
		return m.fireLoggedOut(evt)
	case SAuthFailed:
		return m.fireNoSessionState(evt, SAuthFailed)
	case SAuthDenied:
		return m.fireNoSessionState(evt, SAuthDenied)
	case SInvalidated:
		return m.fireNoSessionState(evt, SInvalidated)
	case SSessionUnavailable:
		return m.fireSessionUnavailable(evt)
	}
	return model.Effect{}
}

func (m *SessionMachine) fireAnonymous(evt SessionEvent) model.Effect {
	switch evt.Kind {
	case SEvLogin:
		m.setCredentials(evt)
		m.State = SAuthenticating
		return sEffect("setCredentials")
	case SEvResume:
		m.State = SResolving
		return sEffect()
	case SEvLogout:
		m.recordNoSessionToLogout()
		return sEffect("recordNoSessionToLogout")
	case SEvUseSession:
		m.recordNoActiveSession()
		return sEffect("recordNoActiveSession")
	}
	return model.Effect{}
}

func (m *SessionMachine) fireAuthenticating(evt SessionEvent) model.Effect {
	switch evt.Kind {
	case SEvInvokeDone:
		if m.guardUserDisabled(evt) {
			m.recordDisabled(evt)
			m.State = SAuthDenied
			return sEffect("recordDisabled")
		}
		m.captureUser(evt)
		m.State = SWritingSession
		return sEffect("captureUser")
	case SEvInvokeError:
		switch {
		case m.isErrBadCredentials(evt):
			m.recordBadCredentials(evt)
			m.State = SAuthFailed
			return sEffect("recordBadCredentials")
		case m.isErrDisabled(evt):
			m.recordDisabled(evt)
			m.State = SAuthDenied
			return sEffect("recordDisabled")
		case m.isErrLocked(evt):
			m.recordError(evt)
			m.State = SVerifyRetry
			return sEffect("recordError")
		default:
			m.recordVerifyError(evt)
			m.State = SSessionUnavailable
			return sEffect("recordVerifyError")
		}
	case SEvVerifyTimeout:
		m.recordTimeout(evt)
		m.State = SSessionUnavailable
		return sEffect("recordTimeout")
	}
	return model.Effect{}
}

func (m *SessionMachine) fireVerifyRetry(evt SessionEvent) model.Effect {
	switch evt.Kind {
	case SEvAlways:
		if m.retriesExhausted() {
			m.recordRetriesExhausted()
			m.State = SSessionUnavailable
			return sEffect("recordRetriesExhausted")
		}
	case SEvVerifyRetryBackoff:
		m.incrementRetries()
		m.State = SAuthenticating
		return sEffect("incrementRetries")
	}
	return model.Effect{}
}

func (m *SessionMachine) fireWritingSession(evt SessionEvent) model.Effect {
	switch evt.Kind {
	case SEvInvokeDone:
		m.State = SActive
		return sEffect()
	case SEvInvokeError:
		m.recordFileError(evt)
		m.State = SSessionUnavailable
		return sEffect("recordFileError")
	case SEvFileIoTimeout:
		m.recordTimeout(evt)
		m.State = SSessionUnavailable
		return sEffect("recordTimeout")
	}
	return model.Effect{}
}

func (m *SessionMachine) fireResolving(evt SessionEvent) model.Effect {
	switch evt.Kind {
	case SEvInvokeDone:
		if m.guardSessionExpired(evt) {
			m.recordExpired()
			m.State = SExpired
			return sEffect("recordExpired")
		}
		m.captureToken(evt)
		m.State = SCheckingUser
		return sEffect("captureToken")
	case SEvInvokeError:
		switch {
		case m.isErrNoSession(evt):
			m.recordNoSession()
			m.State = SAnonymous
			return sEffect("recordNoSession")
		case m.isErrExpired(evt):
			m.recordExpired()
			m.State = SExpired
			return sEffect("recordExpired")
		default:
			m.recordFileError(evt)
			m.State = SSessionUnavailable
			return sEffect("recordFileError")
		}
	case SEvFileIoTimeout:
		m.recordTimeout(evt)
		m.State = SSessionUnavailable
		return sEffect("recordTimeout")
	}
	return model.Effect{}
}

func (m *SessionMachine) fireCheckingUser(evt SessionEvent) model.Effect {
	switch evt.Kind {
	case SEvInvokeDone:
		if m.guardSessionUserActive(evt) {
			m.captureUser(evt)
			m.State = SActive
			return sEffect("captureUser")
		}
		m.recordUserNotActive(evt)
		m.State = SInvalidated
		return sEffect("recordUserNotActive")
	case SEvInvokeError:
		switch {
		case m.isErrLocked(evt):
			m.recordError(evt)
			m.State = SVerifyRetry
			return sEffect("recordError")
		case m.isErrNotFound(evt):
			m.recordUserMissing(evt)
			m.State = SInvalidated
			return sEffect("recordUserMissing")
		default:
			m.recordLoadError(evt)
			m.State = SSessionUnavailable
			return sEffect("recordLoadError")
		}
	case SEvLoadUserTimeout:
		m.recordTimeout(evt)
		m.State = SSessionUnavailable
		return sEffect("recordTimeout")
	}
	return model.Effect{}
}

func (m *SessionMachine) fireActive(evt SessionEvent) model.Effect {
	switch evt.Kind {
	case SEvLogout:
		m.State = SLoggingOut
		return sEffect()
	case SEvUseSession:
		m.recordSessionUsed()
		return sEffect("recordSessionUsed")
	case SEvLogin:
		m.recordAlreadyActive()
		return sEffect("recordAlreadyActive")
	case SEvResume:
		m.recordAlreadyResolved()
		return sEffect("recordAlreadyResolved")
	case SEvSessionTTL:
		m.recordExpired()
		m.State = SExpired
		return sEffect("recordExpired")
	}
	return model.Effect{}
}

func (m *SessionMachine) fireLoggingOut(evt SessionEvent) model.Effect {
	switch evt.Kind {
	case SEvInvokeDone:
		m.State = SLoggedOut
		return sEffect()
	case SEvInvokeError:
		m.recordLogoutWarning(evt)
		m.State = SLoggedOut
		return sEffect("recordLogoutWarning")
	case SEvFileIoTimeout:
		m.recordLogoutWarning(evt)
		m.State = SLoggedOut
		return sEffect("recordLogoutWarning")
	}
	return model.Effect{}
}

func (m *SessionMachine) fireExpired(evt SessionEvent) model.Effect {
	switch evt.Kind {
	case SEvLogin:
		m.setCredentials(evt)
		m.State = SAuthenticating
		return sEffect("setCredentials")
	case SEvResume:
		m.recordExpiredNeedsLogin()
		return sEffect("recordExpiredNeedsLogin")
	case SEvLogout:
		m.recordNoSessionToLogout()
		return sEffect("recordNoSessionToLogout")
	case SEvUseSession:
		m.recordSessionExpired()
		return sEffect("recordSessionExpired")
	}
	return model.Effect{}
}

func (m *SessionMachine) fireLoggedOut(evt SessionEvent) model.Effect {
	switch evt.Kind {
	case SEvLogin:
		m.setCredentials(evt)
		m.State = SAuthenticating
		return sEffect("setCredentials")
	case SEvResume:
		m.recordNoSession()
		return sEffect("recordNoSession")
	case SEvLogout:
		m.recordNoSessionToLogout()
		return sEffect("recordNoSessionToLogout")
	case SEvUseSession:
		m.recordNoActiveSession()
		return sEffect("recordNoActiveSession")
	}
	return model.Effect{}
}

// fireNoSessionState covers AuthFailed, AuthDenied, and Invalidated: login
// re-authenticates; resume/logout/useSession are rejected without changing state.
func (m *SessionMachine) fireNoSessionState(evt SessionEvent, self SessionState) model.Effect {
	switch evt.Kind {
	case SEvLogin:
		m.setCredentials(evt)
		m.State = SAuthenticating
		return sEffect("setCredentials")
	case SEvResume:
		m.recordNoSession()
		return sEffect("recordNoSession")
	case SEvLogout:
		m.recordNoSessionToLogout()
		return sEffect("recordNoSessionToLogout")
	case SEvUseSession:
		m.recordNoActiveSession()
		return sEffect("recordNoActiveSession")
	}
	return model.Effect{}
}

func (m *SessionMachine) fireSessionUnavailable(evt SessionEvent) model.Effect {
	switch evt.Kind {
	case SEvLogin:
		m.setCredentials(evt)
		m.State = SAuthenticating
		return sEffect("setCredentials")
	case SEvResume:
		m.State = SResolving
		return sEffect()
	case SEvLogout:
		m.recordNoSessionToLogout()
		return sEffect("recordNoSessionToLogout")
	case SEvUseSession:
		m.recordNoActiveSession()
		return sEffect("recordNoActiveSession")
	}
	return model.Effect{}
}

// sEffect builds an ordered Effect from the fired action names.
func sEffect(actions ...string) model.Effect { return model.Effect{Actions: actions} }

// --- Guards (BUILD.md 5.4). ---

func (m *SessionMachine) guardUserDisabled(evt SessionEvent) bool {
	return evt.User.Status == model.StatusDisabled
}
func (m *SessionMachine) guardSessionUserActive(evt SessionEvent) bool {
	return evt.User.Status == model.StatusActive
}
func (m *SessionMachine) guardSessionExpired(evt SessionEvent) bool {
	return !evt.Token.ExpiresAt.After(time.Now())
}

func (m *SessionMachine) isErrBadCredentials(evt SessionEvent) bool {
	return sessErrIs(evt.Err, model.ErrBadCredentials)
}
func (m *SessionMachine) isErrDisabled(evt SessionEvent) bool {
	return sessErrIs(evt.Err, model.ErrDisabled)
}
func (m *SessionMachine) isErrLocked(evt SessionEvent) bool {
	return sessErrIs(evt.Err, model.ErrLocked)
}
func (m *SessionMachine) isErrNoSession(evt SessionEvent) bool {
	return sessErrIs(evt.Err, model.ErrNoSession)
}
func (m *SessionMachine) isErrExpired(evt SessionEvent) bool {
	return sessErrIs(evt.Err, model.ErrExpired)
}
func (m *SessionMachine) isErrNotFound(evt SessionEvent) bool {
	return sessErrIs(evt.Err, model.ErrNotFound)
}
func (m *SessionMachine) retriesExhausted() bool { return m.Retries >= sessMaxRetries }

// sessMaxRetries is the verify/load retry bound (BUILD.md 9).
const sessMaxRetries = 3

// sessErrIs matches err against a sentinel, tolerating a nil err.
func sessErrIs(err, target error) bool { return err != nil && errors.Is(err, target) }

// --- Actions (BUILD.md 5.4). Context mutations back the load-act flow. ---

func (m *SessionMachine) setCredentials(evt SessionEvent) { m.Username = evt.Username }
func (m *SessionMachine) captureUser(evt SessionEvent) {
	m.UserID = evt.User.ID
	m.Role = evt.User.Role
	m.TeamID = evt.User.TeamID
	m.UserStatus = evt.User.Status
}
func (m *SessionMachine) captureToken(evt SessionEvent) {
	m.UserID = evt.Token.UserID
	m.ExpiresAt = evt.Token.ExpiresAt
}
func (m *SessionMachine) incrementRetries()                     { m.Retries++ }
func (m *SessionMachine) recordExpired()                        { m.Reason = "session-active-user: token expired" }
func (m *SessionMachine) recordDisabled(evt SessionEvent)       { m.Reason = "disabled-cannot-auth" }
func (m *SessionMachine) recordBadCredentials(evt SessionEvent) { m.LastError = evt.Err }
func (m *SessionMachine) recordUserNotActive(evt SessionEvent) {
	m.Reason = "session-active-user: user not Active"
}
func (m *SessionMachine) recordUserMissing(evt SessionEvent)   { m.LastError = evt.Err }
func (m *SessionMachine) recordError(evt SessionEvent)         { m.LastError = evt.Err }
func (m *SessionMachine) recordVerifyError(evt SessionEvent)   { m.LastError = evt.Err }
func (m *SessionMachine) recordFileError(evt SessionEvent)     { m.LastError = evt.Err }
func (m *SessionMachine) recordLoadError(evt SessionEvent)     { m.LastError = evt.Err }
func (m *SessionMachine) recordTimeout(evt SessionEvent)       { m.LastError = model.ErrTimeout }
func (m *SessionMachine) recordRetriesExhausted()              { m.LastError = model.ErrLocked }
func (m *SessionMachine) recordLogoutWarning(evt SessionEvent) { m.Reason = "logout best-effort" }
func (m *SessionMachine) recordSessionUsed()                   { m.Reason = "" }
func (m *SessionMachine) recordAlreadyActive()                 { m.Reason = "session already Active" }
func (m *SessionMachine) recordAlreadyResolved()               { m.Reason = "session already resolved" }
func (m *SessionMachine) recordNoSession()                     { m.Reason = "no session" }
func (m *SessionMachine) recordNoSessionToLogout()             { m.Reason = "no session to logout" }
func (m *SessionMachine) recordNoActiveSession()               { m.Reason = "no active session" }
func (m *SessionMachine) recordExpiredNeedsLogin()             { m.Reason = "expired; login required" }
func (m *SessionMachine) recordSessionExpired()                { m.Reason = "session expired" }
