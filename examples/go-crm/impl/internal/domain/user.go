package domain

import (
	"crm/internal/authz"
	"crm/internal/model"
)

// UserState is the User machine state: the two statuses plus the persist overlay
// (BUILD.md 5.3, User.machine.json). Only the Active <-> Disabled status
// lifecycle lives here (register/changePassword/assignRole belong to
// crm.session).
type UserState string

const (
	USActive       UserState = "Active"
	USDisabled     UserState = "Disabled"
	USPersisting   UserState = "persisting"
	USPersistRetry UserState = "persistRetry"
	USRolledBack   UserState = "rolledBack"
)

// UserEventKind is the trigger for a User transition (User.machine.json).
type UserEventKind string

const (
	UEvDisable        UserEventKind = "disable"
	UEvEnable         UserEventKind = "enable"
	UEvSaveDone       UserEventKind = "saveDone"
	UEvSaveError      UserEventKind = "saveError"
	UEvPersistTimeout UserEventKind = "persistTimeout"
	UEvAlways         UserEventKind = "always"
	UEvRetryBackoff   UserEventKind = "retryBackoff"
)

// UserEvent is a single trigger. Err carries the classified repo error.
type UserEvent struct {
	Kind UserEventKind
	Err  error
}

// User is the User status aggregate (BUILD.md 5.3, 9). Target is the user whose
// status is changing; Actor is the acting user (must be Admin).
type User struct {
	UserID        string
	State         UserState
	Actor         model.User
	PendingStatus UserState
	PriorStatus   UserState
	Retries       int
	LastError     error
	Rejection     string

	Authz authz.Authorizer
}

// Fire applies an event to the User aggregate, mirroring User.machine.json
// (BUILD.md 7.1 T-USER-01..19). Only the Admin-only Active<->Disabled status
// lifecycle lives here; the redundant direction is an idempotent no-op.
func (u *User) Fire(evt UserEvent) Effect {
	switch u.State {
	case USActive:
		switch evt.Kind {
		case UEvDisable:
			if u.guardAdminAuthority(evt) {
				u.setPendingDisable()
				u.State = USPersisting
				return effect("setPendingDisable")
			}
			u.recordAuthorityDenied(evt)
			return effect("recordAuthorityDenied")
		case UEvEnable:
			u.recordAlreadyActive()
			return effect("recordAlreadyActive")
		}
	case USDisabled:
		switch evt.Kind {
		case UEvEnable:
			if u.guardAdminAuthority(evt) {
				u.setPendingEnable()
				u.State = USPersisting
				return effect("setPendingEnable")
			}
			u.recordAuthorityDenied(evt)
			return effect("recordAuthorityDenied")
		case UEvDisable:
			u.recordAlreadyDisabled()
			return effect("recordAlreadyDisabled")
		}
	case USPersisting:
		return u.firePersisting(evt)
	case USPersistRetry:
		return u.firePersistRetry(evt)
	case USRolledBack:
		return u.fireRolledBack(evt)
	}
	return Effect{}
}

func (u *User) firePersisting(evt UserEvent) Effect {
	switch evt.Kind {
	case UEvSaveDone:
		switch {
		case u.pendingIsActive(), u.pendingIsDisabled():
			u.commitStatus()
			return effect("commitStatus")
		default:
			u.recordRoutingError()
			u.State = USRolledBack
			return effect("recordRoutingError")
		}
	case UEvSaveError:
		switch {
		case u.isErrLocked(evt):
			u.recordError(evt)
			u.State = USPersistRetry
			return effect("recordError")
		case u.isErrConstraint(evt):
			u.recordConstraint(evt)
			u.State = USRolledBack
			return effect("recordConstraint")
		case u.isErrDiskFull(evt):
			u.recordDiskFull(evt)
			u.State = USRolledBack
			return effect("recordDiskFull")
		case u.isErrTimeout(evt):
			u.recordTimeout(evt)
			u.State = USRolledBack
			return effect("recordTimeout")
		default:
			u.recordUnknownError(evt)
			u.State = USRolledBack
			return effect("recordUnknownError")
		}
	case UEvPersistTimeout:
		u.recordTimeout(evt)
		u.State = USRolledBack
		return effect("recordTimeout")
	}
	return Effect{}
}

func (u *User) firePersistRetry(evt UserEvent) Effect {
	switch evt.Kind {
	case UEvAlways:
		if u.retriesExhausted() {
			u.recordRetriesExhausted()
			u.State = USRolledBack
			return effect("recordRetriesExhausted")
		}
	case UEvRetryBackoff:
		u.incrementRetries()
		u.State = USPersisting
		return effect("incrementRetries")
	}
	return Effect{}
}

func (u *User) fireRolledBack(evt UserEvent) Effect {
	if evt.Kind != UEvAlways {
		return Effect{}
	}
	switch {
	case u.priorIsActive():
		u.State = USActive
	case u.priorIsDisabled():
		u.State = USDisabled
	}
	return Effect{}
}

// --- Guards (BUILD.md 5.3). ---

// guardAdminAuthority is true iff the acting user is an Admin (disable/enable are
// Admin verbs, rbac-crud-verbs).
func (u *User) guardAdminAuthority(evt UserEvent) bool { return u.Actor.Role == model.RoleAdmin }

func (u *User) pendingIsActive() bool   { return u.PendingStatus == USActive }
func (u *User) pendingIsDisabled() bool { return u.PendingStatus == USDisabled }
func (u *User) priorIsActive() bool     { return u.PriorStatus == USActive }
func (u *User) priorIsDisabled() bool   { return u.PriorStatus == USDisabled }

func (u *User) isErrLocked(evt UserEvent) bool     { return errIs(evt.Err, model.ErrLocked) }
func (u *User) isErrConstraint(evt UserEvent) bool { return errIs(evt.Err, model.ErrConstraint) }
func (u *User) isErrDiskFull(evt UserEvent) bool   { return errIs(evt.Err, model.ErrDiskFull) }
func (u *User) isErrTimeout(evt UserEvent) bool    { return errIs(evt.Err, model.ErrTimeout) }
func (u *User) retriesExhausted() bool             { return u.Retries >= maxRetries }

// --- Actions (BUILD.md 5.3). ---

func (u *User) setPendingDisable() {
	u.PriorStatus = u.State
	u.PendingStatus = USDisabled
}
func (u *User) setPendingEnable() {
	u.PriorStatus = u.State
	u.PendingStatus = USActive
}
func (u *User) commitStatus()     { u.State = u.PendingStatus }
func (u *User) incrementRetries() { u.Retries++ }

func (u *User) recordError(evt UserEvent)        { u.LastError = evt.Err }
func (u *User) recordConstraint(evt UserEvent)   { u.LastError = evt.Err }
func (u *User) recordDiskFull(evt UserEvent)     { u.LastError = evt.Err }
func (u *User) recordTimeout(evt UserEvent)      { u.LastError = evt.Err }
func (u *User) recordUnknownError(evt UserEvent) { u.LastError = evt.Err }
func (u *User) recordRetriesExhausted()          { u.LastError = model.ErrLocked }
func (u *User) recordRoutingError()              { u.Rejection = "user: unroutable pending status" }
func (u *User) recordAuthorityDenied(evt UserEvent) {
	u.Rejection = "rbac-crud-verbs: disable/enable are Admin verbs"
}
func (u *User) recordAlreadyActive()   { u.Rejection = "user: already Active" }
func (u *User) recordAlreadyDisabled() { u.Rejection = "user: already Disabled" }
