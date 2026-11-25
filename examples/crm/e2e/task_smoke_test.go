package e2e

import (
	"net/http"
	"testing"
)

// machinery:verifies tc:TASK-067fe3 tc:TASK-d307aa tc:TASK-ac3aa3 tc:TASK-2a7cdb
// machinery:verifies fm:task-create-without-user

// newTask creates a task over HTTP as user on a fresh deal it owns.
func newTask(t *testing.T, user string) string {
	t.Helper()
	deal := insertDeal(t, user, "Lead", 1000)
	var c change
	code := callInto(t, "POST", "/tasks", user, map[string]string{"title": unique("task"), "dealId": deal}, &c)
	if code != http.StatusCreated || c.ID == "" {
		t.Fatalf("POST /tasks = %d %+v, want 201 with an id", code, c)
	}
	wantChange(t, c, "Open", nil, nil, nil)
	wantRow(t, "tasks", c.ID, "Open")
	return c.ID
}

func TestTaskHappyPath(t *testing.T) {
	wants(t, "tc:TASK-067fe3", "tc:TASK-d307aa")
	createUser(t, "Admin")
	rep := createUser(t, "Rep")
	id := newTask(t, rep)
	c := fire(t, "/tasks/"+id+"/start", rep)
	wantChange(t, c, "InProgress", []string{"tc:TASK-067fe3"}, []string{"auditStart"}, nil)
	wantRow(t, "tasks", id, "InProgress")
	pass(t, "tc:TASK-067fe3", "Open start by the assignee: 200, InProgress in the answer and the row")
	c = fire(t, "/tasks/"+id+"/complete", rep)
	wantChange(t, c, "Done", []string{"tc:TASK-d307aa"}, []string{"auditComplete"}, nil)
	wantRow(t, "tasks", id, "Done")
	pass(t, "tc:TASK-d307aa", "InProgress complete by the assignee: 200, Done in the answer and the row")
}

func TestTaskIgnoredEvent(t *testing.T) {
	wants(t, "tc:TASK-ac3aa3")
	rep := createUser(t, "Rep")
	id := newTask(t, rep)
	e := refused(t, "/tasks/"+id+"/complete", rep, http.StatusConflict)
	wantRefusal(t, e, "TASK_NOT_STARTED", "tc:TASK-ac3aa3")
	wantRow(t, "tasks", id, "Open")
	pass(t, "tc:TASK-ac3aa3", "complete on an Open task: 409 TASK_NOT_STARTED, the row stays Open")
}

func TestTaskDenied(t *testing.T) {
	wants(t, "tc:TASK-2a7cdb", "tc:TASK-067fe3")
	admin := createUser(t, "Admin")
	rep, other := createUser(t, "Rep"), createUser(t, "Rep")
	id := newTask(t, rep)
	e := refused(t, "/tasks/"+id+"/start", other, http.StatusForbidden)
	wantRefusal(t, e, "FORBIDDEN", "tc:TASK-2a7cdb")
	wantRow(t, "tasks", id, "Open")
	pass(t, "tc:TASK-2a7cdb", "start by another Rep: 403 FORBIDDEN, the row stays Open")
	c := fire(t, "/tasks/"+id+"/start", admin)
	wantChange(t, c, "InProgress", []string{"tc:TASK-067fe3"}, []string{"auditStart"}, nil)
	wantRow(t, "tasks", id, "InProgress")
	pass(t, "tc:TASK-067fe3", "Open start by an Admin: 200, InProgress in the answer and the row")
}

func TestIdentity(t *testing.T) {
	title := unique("orphan")
	var e apiError
	if code := callInto(t, "POST", "/tasks", "", map[string]string{"title": title, "dealId": "00000000-0000-0000-0000-000000000000"}, &e); code != http.StatusUnauthorized || e.Code != "UNAUTHENTICATED" {
		t.Fatalf("POST /tasks without X-User-Id = %d %+v, want 401 UNAUTHENTICATED", code, e)
	}
	if code := callInto(t, "GET", "/tasks", "6b1f0c38-3a5d-4c39-9a4e-000000000001", nil, &e); code != http.StatusUnauthorized || e.Code != "UNAUTHENTICATED" {
		t.Fatalf("GET /tasks with an unknown user = %d %+v, want 401 UNAUTHENTICATED", code, e)
	}
	var n int
	if err := adminPool.QueryRow(t.Context(), "SELECT count(*) FROM tasks WHERE title = $1", title).Scan(&n); err != nil || n != 0 {
		t.Fatalf("tasks titled %s: %d (%v), want 0", title, n, err)
	}
	pass(t, "fm:task-create-without-user", "POST /tasks without a user: 401 UNAUTHENTICATED and no task row")
	rep := createUser(t, "Rep")
	if code := callInto(t, "GET", "/tasks/not-a-uuid", rep, nil, &e); code != http.StatusBadRequest || e.Code != "BAD_REQUEST" {
		t.Fatalf("GET /tasks/not-a-uuid = %d %+v, want 400 BAD_REQUEST", code, e)
	}
	if code := callInto(t, "POST", "/tasks/6b1f0c38-3a5d-4c39-9a4e-000000000002/start", rep, nil, &e); code != http.StatusNotFound || e.Code != "NOT_FOUND" {
		t.Fatalf("POST start on an unknown task = %d %+v, want 404 NOT_FOUND", code, e)
	}
}
