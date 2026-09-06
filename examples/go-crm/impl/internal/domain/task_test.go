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
	id    string
	task  *domain.Task
	event domain.TaskEvent
	rowID string
}

func TestTaskTransitions(t *testing.T) {
	start := domain.TaskEvent{Kind: domain.TEvStart}
	complete := domain.TaskEvent{Kind: domain.TEvComplete}
	cancel := domain.TaskEvent{Kind: domain.TEvCancel}
	// reassign to an in-scope assignee (same team as the manager assigner)
	reassignIn := domain.TaskEvent{Kind: domain.TEvReassign, NewAssignee: uRepOther}
	// reassign to an out-of-scope assignee (different team)
	reassignOut := domain.TaskEvent{Kind: domain.TEvReassign, NewAssignee: uOtherTeam}

	cs := []taskCase{
		// --- Open ---
		{"T-TASK-01_TASK-db41f8", newTask(domain.TSOpen), start, "TASK-db41f8"},
		{"T-TASK-02_notWritable_TASK-2a7cdb", taskWith(domain.TSOpen, uReadOnly), start, "TASK-2a7cdb"},
		{"T-TASK-03_TASK-2019ec", newTask(domain.TSOpen), complete, "TASK-2019ec"},
		{"TASK-2019eca / notWritable / TASK-84d702", taskWith(domain.TSOpen, uReadOnly), complete, "TASK-84d702"},
		{"T-TASK-05_TASK-b819d1", newTask(domain.TSOpen), cancel, "TASK-b819d1"},
		{"TASK-b819d1a / notWritable / TASK-36d38a", taskWith(domain.TSOpen, uReadOnly), cancel, "TASK-36d38a"},
		{"T-TASK-07_TASK-7ab0ac", taskWith(domain.TSOpen, uMgrT1), reassignIn, "TASK-7ab0ac"},
		{"TASK-7ab0aca / assigneeOutOfScope / TASK-b179c7", taskWith(domain.TSOpen, uMgrT1), reassignOut, "TASK-b179c7"},
		{"TASK-7ab0acb / callerNotAuthority / TASK-b179c7", taskWith(domain.TSOpen, uRepOwner), reassignIn, "TASK-b179c7"},
		{"TASK-7ab0acc / sourceOutOfWriteScope / TASK-b179c7", taskWith(domain.TSOpen, uMgrT2), reassignOut, "TASK-b179c7"},

		// --- InProgress ---
		{"T-TASK-09_TASK-173f61", newTask(domain.TSInProgress), start, "TASK-173f61"},
		{"T-TASK-10_TASK-72ad76", newTask(domain.TSInProgress), complete, "TASK-72ad76"},
		{"TASK-72ad76a / notWritable / TASK-7d91c2", taskWith(domain.TSInProgress, uReadOnly), complete, "TASK-7d91c2"},
		{"T-TASK-12_TASK-cdda50", newTask(domain.TSInProgress), cancel, "TASK-cdda50"},
		{"TASK-cdda50a / notWritable / TASK-d159c9", taskWith(domain.TSInProgress, uReadOnly), cancel, "TASK-d159c9"},
		{"T-TASK-14_TASK-2f2bc8", taskWith(domain.TSInProgress, uMgrT1), reassignIn, "TASK-2f2bc8"},
		{"TASK-2f2bc8a / assigneeOutOfScope / TASK-91fb4d", taskWith(domain.TSInProgress, uMgrT1), reassignOut, "TASK-91fb4d"},
		{"TASK-2f2bc8b / callerNotAuthority / TASK-91fb4d", taskWith(domain.TSInProgress, uRepOwner), reassignIn, "TASK-91fb4d"},
		{"TASK-2f2bc8c / sourceOutOfWriteScope / TASK-91fb4d", taskWith(domain.TSInProgress, uMgrT2), reassignOut, "TASK-91fb4d"},

		// --- persist success routing ---
		{"T-TASK-18_TASK-6d5eb1", taskPending(domain.TSOpen), taskSaveDone(), "TASK-6d5eb1"},
		{"T-TASK-19_TASK-ae4260", taskPending(domain.TSInProgress), taskSaveDone(), "TASK-ae4260"},
		{"T-TASK-20_TASK-c56bd7", taskPending(domain.TSDone), taskSaveDone(), "TASK-c56bd7"},
		{"T-TASK-21_TASK-67b0ff", taskPending(domain.TSCancelled), taskSaveDone(), "TASK-67b0ff"},
		{"T-TASK-22_TASK-d5bcc8", taskPending(domain.TaskState("bogus")), taskSaveDone(), "TASK-d5bcc8"},

		// --- persist error routing ---
		{"T-TASK-23_TASK-8d6955", newTask(domain.TSPersisting), taskSaveErr(model.ErrLocked), "TASK-8d6955"},
		{"T-TASK-24_TASK-376b22", newTask(domain.TSPersisting), taskSaveErr(model.ErrConstraint), "TASK-376b22"},
		{"T-TASK-25_TASK-dc5fe1", newTask(domain.TSPersisting), taskSaveErr(model.ErrDiskFull), "TASK-dc5fe1"},
		{"T-TASK-26_TASK-21e793", newTask(domain.TSPersisting), taskSaveErr(model.ErrTimeout), "TASK-21e793"},
		{"T-TASK-27_TASK-be8721", newTask(domain.TSPersisting), taskSaveErr(model.ErrConflict), "TASK-be8721"},
		{"T-TASK-28_TASK-b4999d", newTask(domain.TSPersisting), domain.TaskEvent{Kind: domain.TEvPersistTimeout}, "TASK-b4999d"},

		// --- persistRetry ---
		{"T-TASK-29_TASK-0dd646", taskRetries(3), domain.TaskEvent{Kind: domain.TEvAlways}, "TASK-0dd646"},
		{"T-TASK-30_TASK-168d9b", taskRetries(0), domain.TaskEvent{Kind: domain.TEvRetryBackoff}, "TASK-168d9b"},

		// --- rolledBack routing (only non-terminal prior states persist) ---
		{"T-TASK-31_TASK-3f585f", taskPrior(domain.TSOpen), domain.TaskEvent{Kind: domain.TEvAlways}, "TASK-3f585f"},
		{"T-TASK-32_TASK-98c3ba", taskPrior(domain.TSInProgress), domain.TaskEvent{Kind: domain.TEvAlways}, "TASK-98c3ba"},
		{"TASK-754183 / fail-closed rollback routing", taskPrior(domain.TaskState("bogus")), domain.TaskEvent{Kind: domain.TEvAlways}, "TASK-754183"},
	}
	suite, err := testoracle.Load("../../../design/machines", "Task")
	if err != nil {
		t.Fatal(err)
	}
	expectations := make([]testoracle.Expectation, len(cs))
	registrations := make([]testoracle.Registration, len(cs))
	for i, tc := range cs {
		expected, err := suite.Bind(taskWitness(t, tc))
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
			if tc.rowID == "TASK-754183" || tc.rowID == "TASK-67b0ff" || tc.rowID == "TASK-c56bd7" {
				tc.task.Rejection = "seed-before-entry"
			}
			got := tc.task.Fire(tc.event)
			if err := expectations[i].Check(string(tc.task.State), got.Actions); err != nil {
				t.Fatal(err)
			}
			if tc.rowID == "TASK-754183" || tc.rowID == "TASK-67b0ff" || tc.rowID == "TASK-c56bd7" {
				if tc.task.Rejection != "" {
					t.Fatalf("oracle-entry-context %s: got=%q want=%q", tc.rowID, tc.task.Rejection, "")
				}
			}
			if registrations[i].Native != t.Name() {
				t.Fatalf("oracle-native-name: got=%q want=%q", t.Name(), registrations[i].Native)
			}
			raw, err := json.Marshal(testoracle.Observation{Registration: registrations[i], Next: string(tc.task.State), Actions: got.Actions})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("oracle-executed %s", raw)
		})
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

// TASK-95f75f and TASK-841a9c: Done and Cancelled are final; every event is structurally
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
		{"TASK-95f75f", domain.TSDone},
		{"TASK-841a9c", domain.TSCancelled},
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

func taskWitness(t *testing.T, tc taskCase) testoracle.Witness {
	t.Helper()
	d, e := tc.task, tc.event
	write := oracleWriteScope(d.Actor, d.OwnerID, d.TeamID)
	callerAuthority := d.Actor.Role == model.RoleAdmin || d.Actor.Role == model.RoleManager
	sourceScope := d.Actor.Role == model.RoleAdmin || d.TeamID != "" && d.Actor.TeamID == d.TeamID
	targetScope := d.Actor.Role == model.RoleAdmin || d.Actor.TeamID != "" && d.Actor.TeamID == e.NewAssignee.TeamID
	oracleClause(t, tc.id, "notWritable", write)
	oracleClause(t, tc.id, "assigneeOutOfScope", targetScope)
	oracleClause(t, tc.id, "callerNotAuthority", callerAuthority)
	oracleClause(t, tc.id, "sourceOutOfWriteScope", sourceScope)
	all := map[string]bool{
		"guardCanStart": write, "guardCanComplete": write, "guardCanCancel": write,
		"guardCanReassign": callerAuthority && sourceScope && targetScope,
		"pendingIsOpen":    d.PendingStatus == domain.TSOpen, "pendingIsInProgress": d.PendingStatus == domain.TSInProgress,
		"pendingIsDone": d.PendingStatus == domain.TSDone, "pendingIsCancelled": d.PendingStatus == domain.TSCancelled,
		"priorIsOpen": d.PriorStatus == domain.TSOpen, "priorIsInProgress": d.PriorStatus == domain.TSInProgress,
		"retriesExhausted": d.Retries >= 3, "isErrLocked": errors.Is(e.Err, model.ErrLocked),
		"isErrConstraint": errors.Is(e.Err, model.ErrConstraint), "isErrDiskFull": errors.Is(e.Err, model.ErrDiskFull), "isErrTimeout": errors.Is(e.Err, model.ErrTimeout),
	}
	w := testoracle.Witness{Name: tc.id, RowID: tc.rowID, Source: string(d.State)}
	switch d.State {
	case domain.TSOpen, domain.TSInProgress:
		var key string
		switch e.Kind {
		case domain.TEvStart:
			if d.State == domain.TSOpen {
				key = "guardCanStart"
			}
		case domain.TEvComplete:
			key = "guardCanComplete"
		case domain.TEvCancel:
			key = "guardCanCancel"
		case domain.TEvReassign:
			key = "guardCanReassign"
		default:
			t.Fatalf("oracle-event task: unsupported event=%q", e.Kind)
		}
		w.Trigger = "on:" + string(e.Kind)
		if key != "" {
			w.Guards = oracleFacts(t, all, key)
		}
		return w
	case domain.TSPersisting:
		switch e.Kind {
		case domain.TEvSaveDone:
			w.Trigger = "onDone:saveTask"
			w.Guards = oracleFacts(t, all, "pendingIsOpen", "pendingIsInProgress", "pendingIsDone", "pendingIsCancelled")
			return w
		case domain.TEvSaveError:
			w.Trigger = "onError:saveTask"
			w.Guards = oracleFacts(t, all, "isErrLocked", "isErrConstraint", "isErrDiskFull", "isErrTimeout")
			return w
		case domain.TEvPersistTimeout:
			w.Trigger = "after:persistTimeout"
			return w
		}
	case domain.TSPersistRetry:
		switch e.Kind {
		case domain.TEvAlways:
			w.Trigger = "always"
			w.Guards = oracleFacts(t, all, "retriesExhausted")
			return w
		case domain.TEvRetryBackoff:
			w.Trigger = "after:persistRetryBackoff"
			return w
		}
	case domain.TSRolledBack:
		if e.Kind == domain.TEvAlways {
			w.Trigger = "always"
			w.Guards = oracleFacts(t, all, "priorIsOpen", "priorIsInProgress")
			return w
		}
	}
	t.Fatalf("oracle-event task: unsupported actual source=%q event=%q", d.State, e.Kind)
	return w
}

func TestTaskRollbackNonAlwaysPreservesContext(t *testing.T) {
	for _, prior := range []domain.TaskState{domain.TSOpen, domain.TSInProgress, "bogus"} {
		t.Run(string(prior), func(t *testing.T) {
			task := taskPrior(prior)
			task.Rejection = "seed-before-nontransition"
			got := task.Fire(domain.TaskEvent{Kind: domain.TEvStart})
			if task.State != domain.TSRolledBack || task.PriorStatus != prior || task.Rejection != "seed-before-nontransition" || len(got.Actions) != 0 {
				t.Fatalf("task-nontransition: state=%q prior=%q rejection=%q actions=%q", task.State, task.PriorStatus, task.Rejection, got.Actions)
			}
		})
	}
}
