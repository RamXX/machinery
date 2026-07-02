package domain_test

// Task transition oracle. One case per BUILD.md 7.1 T-TASK row (with a sub-case
// per falsifying clause of the conjunction guards). Source: machines/Task.matrix.md.

import (
	"testing"

	"crm/internal/authz"
	"crm/internal/domain"
	"crm/internal/model"
)

var uOtherTeam = model.User{ID: "z9", Role: model.RoleRep, Status: model.StatusActive, TeamID: "t2"}

func newTask(state domain.TaskState) *domain.Task {
	return &domain.Task{
		State:   state,
		TaskID:  "tk1",
		OwnerID: "rep1",
		TeamID:  "t1",
		Actor:   uRepOwner, // owner may write own task
		Authz:   authz.New(),
	}
}

func taskWith(state domain.TaskState, actor model.User) *domain.Task {
	t := newTask(state)
	t.Actor = actor
	return t
}

func taskPending(pending domain.TaskState) *domain.Task {
	t := newTask(domain.TSPersisting)
	t.PendingStatus = pending
	return t
}

func taskPrior(prior domain.TaskState) *domain.Task {
	t := newTask(domain.TSRolledBack)
	t.PriorStatus = prior
	return t
}

type taskCase struct {
	id      string
	task    *domain.Task
	event   domain.TaskEvent
	want    domain.TaskState
	actions []string
}

func taskCases() []taskCase {
	start := domain.TaskEvent{Kind: domain.TEvStart}
	complete := domain.TaskEvent{Kind: domain.TEvComplete}
	cancel := domain.TaskEvent{Kind: domain.TEvCancel}
	// reassign to an in-scope assignee (same team as the manager assigner)
	reassignIn := domain.TaskEvent{Kind: domain.TEvReassign, NewAssignee: uRepOther}
	// reassign to an out-of-scope assignee (different team)
	reassignOut := domain.TaskEvent{Kind: domain.TEvReassign, NewAssignee: uOtherTeam}

	return []taskCase{
		// --- Open ---
		{"T-TASK-01", newTask(domain.TSOpen), start, domain.TSPersisting, []string{"setPendingStart"}},
		{"T-TASK-02_notWritable", taskWith(domain.TSOpen, uReadOnly), start, domain.TSOpen, []string{"recordStartDenied"}},
		{"T-TASK-03", newTask(domain.TSOpen), complete, domain.TSPersisting, []string{"setPendingComplete"}},
		{"T-TASK-04_notWritable", taskWith(domain.TSOpen, uReadOnly), complete, domain.TSOpen, []string{"recordCompleteDenied"}},
		{"T-TASK-05", newTask(domain.TSOpen), cancel, domain.TSPersisting, []string{"setPendingCancel"}},
		{"T-TASK-06_notWritable", taskWith(domain.TSOpen, uReadOnly), cancel, domain.TSOpen, []string{"recordCancelDenied"}},
		{"T-TASK-07", taskWith(domain.TSOpen, uMgrT1), reassignIn, domain.TSPersisting, []string{"setPendingReassign"}},
		{"T-TASK-08a_assigneeOutOfScope", taskWith(domain.TSOpen, uMgrT1), reassignOut, domain.TSOpen, []string{"recordReassignDenied"}},
		{"T-TASK-08b_callerNotAuthority", taskWith(domain.TSOpen, uRepOwner), reassignIn, domain.TSOpen, []string{"recordReassignDenied"}},

		// --- InProgress ---
		{"T-TASK-09", newTask(domain.TSInProgress), start, domain.TSInProgress, []string{"recordAlreadyStarted"}},
		{"T-TASK-10", newTask(domain.TSInProgress), complete, domain.TSPersisting, []string{"setPendingComplete"}},
		{"T-TASK-11_notWritable", taskWith(domain.TSInProgress, uReadOnly), complete, domain.TSInProgress, []string{"recordCompleteDenied"}},
		{"T-TASK-12", newTask(domain.TSInProgress), cancel, domain.TSPersisting, []string{"setPendingCancel"}},
		{"T-TASK-13_notWritable", taskWith(domain.TSInProgress, uReadOnly), cancel, domain.TSInProgress, []string{"recordCancelDenied"}},
		{"T-TASK-14", taskWith(domain.TSInProgress, uMgrT1), reassignIn, domain.TSPersisting, []string{"setPendingReassign"}},
		{"T-TASK-15a_assigneeOutOfScope", taskWith(domain.TSInProgress, uMgrT1), reassignOut, domain.TSInProgress, []string{"recordReassignDenied"}},
		{"T-TASK-15b_callerNotAuthority", taskWith(domain.TSInProgress, uRepOwner), reassignIn, domain.TSInProgress, []string{"recordReassignDenied"}},

		// --- persist success routing ---
		{"T-TASK-18", taskPending(domain.TSOpen), taskSaveDone(), domain.TSOpen, []string{"commitStatus"}},
		{"T-TASK-19", taskPending(domain.TSInProgress), taskSaveDone(), domain.TSInProgress, []string{"commitStatus"}},
		{"T-TASK-20", taskPending(domain.TSDone), taskSaveDone(), domain.TSDone, []string{"commitStatus"}},
		{"T-TASK-21", taskPending(domain.TSCancelled), taskSaveDone(), domain.TSCancelled, []string{"commitStatus"}},
		{"T-TASK-22", taskPending(domain.TaskState("bogus")), taskSaveDone(), domain.TSRolledBack, []string{"recordRoutingError"}},

		// --- persist error routing ---
		{"T-TASK-23", newTask(domain.TSPersisting), taskSaveErr(model.ErrLocked), domain.TSPersistRetry, []string{"recordError"}},
		{"T-TASK-24", newTask(domain.TSPersisting), taskSaveErr(model.ErrConstraint), domain.TSRolledBack, []string{"recordConstraint"}},
		{"T-TASK-25", newTask(domain.TSPersisting), taskSaveErr(model.ErrDiskFull), domain.TSRolledBack, []string{"recordDiskFull"}},
		{"T-TASK-26", newTask(domain.TSPersisting), taskSaveErr(model.ErrTimeout), domain.TSRolledBack, []string{"recordTimeout"}},
		{"T-TASK-27", newTask(domain.TSPersisting), taskSaveErr(model.ErrConflict), domain.TSRolledBack, []string{"recordUnknownError"}},
		{"T-TASK-28", newTask(domain.TSPersisting), domain.TaskEvent{Kind: domain.TEvPersistTimeout}, domain.TSRolledBack, []string{"recordTimeout"}},

		// --- persistRetry ---
		{"T-TASK-29", taskRetries(3), domain.TaskEvent{Kind: domain.TEvAlways}, domain.TSRolledBack, []string{"recordRetriesExhausted"}},
		{"T-TASK-30", taskRetries(0), domain.TaskEvent{Kind: domain.TEvRetryBackoff}, domain.TSPersisting, []string{"incrementRetries"}},

		// --- rolledBack routing (only non-terminal prior states persist) ---
		{"T-TASK-31", taskPrior(domain.TSOpen), domain.TaskEvent{Kind: domain.TEvAlways}, domain.TSOpen, nil},
		{"T-TASK-32", taskPrior(domain.TSInProgress), domain.TaskEvent{Kind: domain.TEvAlways}, domain.TSInProgress, nil},
	}
}

func taskRetries(n int) *domain.Task {
	t := newTask(domain.TSPersistRetry)
	t.Retries = n
	return t
}

func taskSaveDone() domain.TaskEvent { return domain.TaskEvent{Kind: domain.TEvSaveDone} }
func taskSaveErr(e error) domain.TaskEvent {
	return domain.TaskEvent{Kind: domain.TEvSaveError, Err: e}
}

func TestTaskTransitions(t *testing.T) {
	for _, tc := range taskCases() {
		t.Run(tc.id, func(t *testing.T) {
			got := tc.task.Fire(tc.event)
			if tc.task.State != tc.want {
				t.Errorf("%s: next state = %q, want %q", tc.id, tc.task.State, tc.want)
			}
			if !firedInOrder(got.Actions, tc.actions) {
				t.Errorf("%s: actions = %v, want (in order) %v", tc.id, got.Actions, tc.actions)
			}
		})
	}
}

// T-TASK-16, T-TASK-17: Done and Cancelled are final; every event is structurally
// rejected (task-terminal) with no state change and no action. These pin the
// structural guarantee (the machine has no transition out of a final state), so
// they hold for the scaffolding too.
func TestTaskTerminalRejectsEverything(t *testing.T) {
	events := []domain.TaskEvent{
		{Kind: domain.TEvStart}, {Kind: domain.TEvComplete},
		{Kind: domain.TEvCancel}, {Kind: domain.TEvReassign, NewAssignee: uRepOther},
	}
	for _, term := range []struct {
		id    string
		state domain.TaskState
	}{
		{"T-TASK-16", domain.TSDone},
		{"T-TASK-17", domain.TSCancelled},
	} {
		t.Run(term.id, func(t *testing.T) {
			for _, ev := range events {
				task := newTask(term.state)
				got := task.Fire(ev)
				if task.State != term.state {
					t.Errorf("%s: event %s changed terminal state to %q", term.id, ev.Kind, task.State)
				}
				if len(got.Actions) != 0 {
					t.Errorf("%s: event %s fired actions %v on a terminal task", term.id, ev.Kind, got.Actions)
				}
			}
		})
	}
}
