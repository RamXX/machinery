package domain

import (
	"crm/internal/authz"
	"crm/internal/model"
)

// TaskState is the Task machine state: the two non-terminal statuses, the two
// terminal (final) statuses, and the persist overlay (BUILD.md 5.2,
// Task.machine.json).
type TaskState string

const (
	TSOpen         TaskState = "Open"
	TSInProgress   TaskState = "InProgress"
	TSDone         TaskState = "Done"
	TSCancelled    TaskState = "Cancelled"
	TSPersisting   TaskState = "persisting"
	TSPersistRetry TaskState = "persistRetry"
	TSRolledBack   TaskState = "rolledBack"
)

// TaskEventKind is the trigger for a Task transition (Task.machine.json).
type TaskEventKind string

const (
	TEvStart          TaskEventKind = "start"
	TEvComplete       TaskEventKind = "complete"
	TEvCancel         TaskEventKind = "cancel"
	TEvReassign       TaskEventKind = "reassign"
	TEvSaveDone       TaskEventKind = "saveDone"
	TEvSaveError      TaskEventKind = "saveError"
	TEvPersistTimeout TaskEventKind = "persistTimeout"
	TEvAlways         TaskEventKind = "always"
	TEvRetryBackoff   TaskEventKind = "retryBackoff"
)

// TaskEvent is a single trigger. NewAssignee is supplied on reassign; Err
// carries the classified repo error on saveError.
type TaskEvent struct {
	Kind        TaskEventKind
	NewAssignee model.User
	Err         error
}

// Task is the Task aggregate (BUILD.md 5.2, 9).
type Task struct {
	TaskID        string
	Title         string
	State         TaskState
	OwnerID       string
	TeamID        string
	DealID        string
	Actor         model.User
	PendingStatus TaskState
	PriorStatus   TaskState
	NewAssigneeID string
	Retries       int
	LastError     error
	Rejection     string

	Authz authz.Authorizer
}

// Fire applies an event to the Task, mirroring Task.machine.json (BUILD.md 7.1
// T-TASK-01..32). Done and Cancelled are final: they accept no events, which
// enforces task-terminal structurally.
func (t *Task) Fire(evt TaskEvent) Effect {
	switch t.State {
	case TSOpen:
		return t.fireOpen(evt)
	case TSInProgress:
		return t.fireInProgress(evt)
	case TSDone, TSCancelled:
		return Effect{} // final: task-terminal, no transition, no action
	case TSPersisting:
		return t.firePersisting(evt)
	case TSPersistRetry:
		return t.firePersistRetry(evt)
	case TSRolledBack:
		return t.fireRolledBack(evt)
	}
	return Effect{}
}

// fireOpen handles the Open state (start/complete/cancel/reassign).
func (t *Task) fireOpen(evt TaskEvent) Effect {
	switch evt.Kind {
	case TEvStart:
		if t.guardCanStart(evt) {
			t.setPendingStart()
			t.State = TSPersisting
			return effect("setPendingStart")
		}
		t.recordStartDenied(evt)
		return effect("recordStartDenied")
	case TEvComplete:
		return t.fireComplete(evt)
	case TEvCancel:
		return t.fireCancel(evt)
	case TEvReassign:
		return t.fireReassign(evt)
	}
	return Effect{}
}

// fireInProgress handles the InProgress state; start is an idempotent no-op.
func (t *Task) fireInProgress(evt TaskEvent) Effect {
	switch evt.Kind {
	case TEvStart:
		t.recordAlreadyStarted()
		return effect("recordAlreadyStarted")
	case TEvComplete:
		return t.fireComplete(evt)
	case TEvCancel:
		return t.fireCancel(evt)
	case TEvReassign:
		return t.fireReassign(evt)
	}
	return Effect{}
}

func (t *Task) fireComplete(evt TaskEvent) Effect {
	if t.guardCanComplete(evt) {
		t.setPendingComplete()
		t.State = TSPersisting
		return effect("setPendingComplete")
	}
	t.recordCompleteDenied(evt)
	return effect("recordCompleteDenied")
}

func (t *Task) fireCancel(evt TaskEvent) Effect {
	if t.guardCanCancel(evt) {
		t.setPendingCancel()
		t.State = TSPersisting
		return effect("setPendingCancel")
	}
	t.recordCancelDenied(evt)
	return effect("recordCancelDenied")
}

func (t *Task) fireReassign(evt TaskEvent) Effect {
	if t.guardCanReassign(evt) {
		t.setPendingReassign(evt)
		t.State = TSPersisting
		return effect("setPendingReassign")
	}
	t.recordReassignDenied(evt)
	return effect("recordReassignDenied")
}

// firePersisting routes the save outcome onto the resting status.
func (t *Task) firePersisting(evt TaskEvent) Effect {
	switch evt.Kind {
	case TEvSaveDone:
		switch {
		case t.pendingIsOpen(), t.pendingIsInProgress():
			t.commitStatus()
			return effect("commitStatus")
		case t.pendingIsDone(), t.pendingIsCancelled():
			t.commitStatus()
			t.recordTaskClosed()
			return effect("commitStatus", "recordTaskClosed")
		default:
			t.recordRoutingError()
			t.State = TSRolledBack
			return effect("recordRoutingError")
		}
	case TEvSaveError:
		switch {
		case t.isErrLocked(evt):
			t.recordError(evt)
			t.State = TSPersistRetry
			return effect("recordError")
		case t.isErrConstraint(evt):
			t.recordConstraint(evt)
			t.State = TSRolledBack
			return effect("recordConstraint")
		case t.isErrDiskFull(evt):
			t.recordDiskFull(evt)
			t.State = TSRolledBack
			return effect("recordDiskFull")
		case t.isErrTimeout(evt):
			t.recordTimeout(evt)
			t.State = TSRolledBack
			return effect("recordTimeout")
		default:
			t.recordUnknownError(evt)
			t.State = TSRolledBack
			return effect("recordUnknownError")
		}
	case TEvPersistTimeout:
		t.recordTimeout(evt)
		t.State = TSRolledBack
		return effect("recordTimeout")
	}
	return Effect{}
}

func (t *Task) firePersistRetry(evt TaskEvent) Effect {
	switch evt.Kind {
	case TEvAlways:
		if t.retriesExhausted() {
			t.recordRetriesExhausted()
			t.State = TSRolledBack
			return effect("recordRetriesExhausted")
		}
	case TEvRetryBackoff:
		t.incrementRetries()
		t.State = TSPersisting
		return effect("incrementRetries")
	}
	return Effect{}
}

func (t *Task) fireRolledBack(evt TaskEvent) Effect {
	if evt.Kind != TEvAlways {
		return Effect{}
	}
	switch {
	case t.priorIsOpen():
		t.State = TSOpen
	case t.priorIsInProgress():
		t.State = TSInProgress
	}
	return Effect{}
}

// canWrite is the rbac-write-scope re-check shared by the write guards.
func (t *Task) canWrite() bool {
	return t.Authz.Authorize(t.Actor, model.VerbUpdate, model.EntityTask, t.OwnerID, t.TeamID).Allowed
}

// --- Guards (BUILD.md 5.2). ---

func (t *Task) guardCanStart(evt TaskEvent) bool    { return t.canWrite() }
func (t *Task) guardCanComplete(evt TaskEvent) bool { return t.canWrite() }
func (t *Task) guardCanCancel(evt TaskEvent) bool   { return t.canWrite() }

// guardCanReassign is the complete reassignment decision: authority over the
// record (Admin, or Manager in scope) AND the target rule (an Admin may
// reassign to any User, a Manager only to a member of its own Team)
// (task-assignee-visible, rbac-reassign-authority, rbac-write-scope).
func (t *Task) guardCanReassign(evt TaskEvent) bool {
	return t.Authz.AuthorizeReassign(t.Actor, model.EntityTask, t.OwnerID, t.TeamID, evt.NewAssignee).Allowed
}

func (t *Task) pendingIsOpen() bool       { return t.PendingStatus == TSOpen }
func (t *Task) pendingIsInProgress() bool { return t.PendingStatus == TSInProgress }
func (t *Task) pendingIsDone() bool       { return t.PendingStatus == TSDone }
func (t *Task) pendingIsCancelled() bool  { return t.PendingStatus == TSCancelled }

func (t *Task) priorIsOpen() bool       { return t.PriorStatus == TSOpen }
func (t *Task) priorIsInProgress() bool { return t.PriorStatus == TSInProgress }

func (t *Task) isErrLocked(evt TaskEvent) bool     { return errIs(evt.Err, model.ErrLocked) }
func (t *Task) isErrConstraint(evt TaskEvent) bool { return errIs(evt.Err, model.ErrConstraint) }
func (t *Task) isErrDiskFull(evt TaskEvent) bool   { return errIs(evt.Err, model.ErrDiskFull) }
func (t *Task) isErrTimeout(evt TaskEvent) bool    { return errIs(evt.Err, model.ErrTimeout) }
func (t *Task) retriesExhausted() bool             { return t.Retries >= maxRetries }

// --- Actions (BUILD.md 5.2). ---

func (t *Task) setPendingStart() {
	t.PriorStatus = t.State
	t.PendingStatus = TSInProgress
}
func (t *Task) setPendingComplete() {
	t.PriorStatus = t.State
	t.PendingStatus = TSDone
}
func (t *Task) setPendingCancel() {
	t.PriorStatus = t.State
	t.PendingStatus = TSCancelled
}
func (t *Task) setPendingReassign(evt TaskEvent) {
	t.PriorStatus = t.State
	t.PendingStatus = t.State // reassign keeps the status
	t.NewAssigneeID = evt.NewAssignee.ID
}
func (t *Task) commitStatus() {
	t.State = t.PendingStatus
	if t.NewAssigneeID != "" {
		t.OwnerID = t.NewAssigneeID
	}
}
func (t *Task) incrementRetries() { t.Retries++ }

func (t *Task) recordError(evt TaskEvent)          { t.LastError = evt.Err }
func (t *Task) recordConstraint(evt TaskEvent)     { t.LastError = evt.Err }
func (t *Task) recordDiskFull(evt TaskEvent)       { t.LastError = evt.Err }
func (t *Task) recordTimeout(evt TaskEvent)        { t.LastError = evt.Err }
func (t *Task) recordUnknownError(evt TaskEvent)   { t.LastError = evt.Err }
func (t *Task) recordRetriesExhausted()            { t.LastError = model.ErrLocked }
func (t *Task) recordRoutingError()                { t.Rejection = "task: unroutable pending status" }
func (t *Task) recordStartDenied(evt TaskEvent)    { t.Rejection = "rbac-write-scope" }
func (t *Task) recordCompleteDenied(evt TaskEvent) { t.Rejection = "task-terminal/rbac-write-scope" }
func (t *Task) recordCancelDenied(evt TaskEvent)   { t.Rejection = "task-terminal/rbac-write-scope" }
func (t *Task) recordReassignDenied(evt TaskEvent) {
	t.Rejection = "task-assignee-visible/rbac-reassign-authority/rbac-write-scope"
}
func (t *Task) recordAlreadyStarted() { t.Rejection = "task: already started" }
func (t *Task) recordTaskClosed()     { t.Rejection = "" }
