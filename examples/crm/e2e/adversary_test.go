package e2e

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// machinery:verifies inv:deal-owner-or-admin-writes inv:deal-reopen-admin-only inv:deal-won-needs-amount inv:deal-won-is-final
// machinery:verifies inv:task-assignee-or-admin-writes inv:task-terminal-is-final inv:user-email-unique
// machinery:verifies fm:deal-qualify-by-non-owner fm:deal-win-by-non-owner fm:deal-lose-by-non-owner fm:deal-reopen-by-rep
// machinery:verifies fm:task-start-by-non-assignee fm:task-complete-by-non-assignee fm:task-cancel-by-non-assignee

// write is one event on a record in a state, with the state it leads to.
type write struct {
	event, from, to string
	abuse           string // the abuse failure mode a refused caller is, if any
}

// succeeds fires the event as the caller and expects the change.
func succeeds(t *testing.T, a arranged, w write) {
	t.Helper()
	c := fire(t, a.path(w.event), a.caller)
	if c.State != w.to {
		t.Fatalf("%s %s from %s = %+v, want %s", a.kind, w.event, w.from, c, w.to)
	}
	wantRow(t, a.kind+"s", a.id, w.to)
}

// deniedAudited expects 403 FORBIDDEN, the row unchanged, and exactly one
// refused audit row by the caller as the whole trail of the record.
func deniedAudited(t *testing.T, a arranged, w write) {
	t.Helper()
	before := counts(t)
	e := refused(t, a.path(w.event), a.caller, http.StatusForbidden)
	if e.Code != "FORBIDDEN" {
		t.Fatalf("%s %s by %s = %+v, want FORBIDDEN", a.kind, w.event, a.caller, e)
	}
	wantRow(t, a.kind+"s", a.id, w.from)
	wantTrail(t, a.subject(), trailRow{a.caller, a.kind + "denied", "refused"})
	before.auditEntries++
	wantCounts(t, before)
}

// finalRefused expects 409 NOT_ALLOWED_IN_STATE and nothing changed.
func finalRefused(t *testing.T, a arranged, event, state string) {
	t.Helper()
	before, trailBefore := counts(t), len(trail(t, a.subject()))
	e := refused(t, a.path(event), a.caller, http.StatusConflict)
	if e.Code != "NOT_ALLOWED_IN_STATE" {
		t.Fatalf("%s %s on %s = %+v, want NOT_ALLOWED_IN_STATE", a.kind, event, state, e)
	}
	wantRow(t, a.kind+"s", a.id, state)
	if n := len(trail(t, a.subject())); n != trailBefore {
		t.Fatalf("%s %s on %s: the audit trail grew from %d to %d rows", a.kind, event, state, trailBefore, n)
	}
	wantCounts(t, before)
}

// ownership checks every write of a machine against the owner, an Admin and
// another Rep, and emits the abuse receipts the refusals prove.
func ownership(t *testing.T, kind string, writes []write) {
	t.Helper()
	for _, w := range writes {
		for _, who := range []string{byOwner, byAdmin} {
			succeeds(t, arrange(t, kind, w.from, arrangement{who, 1000}), w)
		}
		deniedAudited(t, arrange(t, kind, w.from, arrangement{byOther, 1000}), w)
		if w.abuse != "" {
			pass(t, w.abuse, "another Rep "+w.event+" on a "+w.from+" "+kind+": 403 FORBIDDEN, still "+w.from+", one refused audit row")
		}
	}
}

func TestDealOwnerOrAdminWrites(t *testing.T) {
	wants(t, "inv:deal-owner-or-admin-writes", "fm:deal-qualify-by-non-owner", "fm:deal-win-by-non-owner", "fm:deal-lose-by-non-owner")
	ownership(t, "deal", []write{
		{"qualify", "Lead", "Qualified", "fm:deal-qualify-by-non-owner"},
		{"win", "Qualified", "Won", "fm:deal-win-by-non-owner"},
		{"lose", "Lead", "Lost", ""},
		{"lose", "Qualified", "Lost", "fm:deal-lose-by-non-owner"},
	})
	pass(t, "inv:deal-owner-or-admin-writes", "qualify, win and lose: the owner and an Admin succeed; another Rep gets 403, the row unchanged")
}

func TestTaskAssigneeOrAdminWrites(t *testing.T) {
	wants(t, "inv:task-assignee-or-admin-writes", "fm:task-start-by-non-assignee", "fm:task-complete-by-non-assignee", "fm:task-cancel-by-non-assignee")
	ownership(t, "task", []write{
		{"start", "Open", "InProgress", "fm:task-start-by-non-assignee"},
		{"complete", "InProgress", "Done", "fm:task-complete-by-non-assignee"},
		{"cancel", "Open", "Cancelled", "fm:task-cancel-by-non-assignee"},
		{"cancel", "InProgress", "Cancelled", ""},
	})
	pass(t, "inv:task-assignee-or-admin-writes", "start, complete and cancel: the assignee and an Admin succeed; another Rep gets 403, the row unchanged")
}

func TestDealReopenAdminOnly(t *testing.T) {
	wants(t, "inv:deal-reopen-admin-only", "fm:deal-reopen-by-rep")
	w := write{"reopen", "Lost", "Lead", ""}
	deniedAudited(t, arrange(t, "deal", "Lost", arrangement{byOwner, 1000}), w)
	pass(t, "fm:deal-reopen-by-rep", "the owning Rep reopens a Lost deal: 403 FORBIDDEN, still Lost, one refused audit row")
	deniedAudited(t, arrange(t, "deal", "Lost", arrangement{byOther, 1000}), w)
	succeeds(t, arrange(t, "deal", "Lost", arrangement{byAdmin, 1000}), w)
	pass(t, "inv:deal-reopen-admin-only", "reopen of a Lost deal: the owning Rep and another Rep get 403; an Admin reopens it to Lead")
}

func TestDealWonNeedsAmount(t *testing.T) {
	wants(t, "inv:deal-won-needs-amount")
	w := write{"win", "Qualified", "Won", ""}
	for _, who := range []string{byOwner, byAdmin} {
		deniedAudited(t, arrange(t, "deal", "Qualified", arrangement{who, 0}), w)
	}
	rep := createUser(t, "Rep")
	before := counts(t)
	var e apiError
	if code := callInto(t, "POST", "/deals", rep, map[string]any{"title": unique("negative"), "amountCents": -1}, &e); code != http.StatusBadRequest || e.Code != "BAD_REQUEST" {
		t.Fatalf("POST /deals with amountCents -1 = %d %+v, want 400 BAD_REQUEST", code, e)
	}
	wantCounts(t, before)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := adminPool.Exec(ctx, `INSERT INTO deals (title, stage, amount_cents, owner_id)
		VALUES ('negative', 'Qualified', -1, $1::uuid)`, rep); err == nil {
		t.Fatal("the database stored a deal with a negative amount")
	}
	succeeds(t, arrange(t, "deal", "Qualified", arrangement{byOwner, 1}), w)
	pass(t, "inv:deal-won-needs-amount", "win at amount 0 is 403 for the owner and an Admin; a negative amount is refused by the API and the database; amount 1 wins")
}

func TestDealWonIsFinal(t *testing.T) {
	wants(t, "inv:deal-won-is-final")
	for _, who := range []string{byOwner, byAdmin} {
		a := arrange(t, "deal", "Qualified", arrangement{who, 1000})
		succeeds(t, a, write{"win", "Qualified", "Won", ""})
		for _, event := range []string{"qualify", "win", "lose", "reopen", "win"} {
			finalRefused(t, a, event, "Won")
		}
	}
	pass(t, "inv:deal-won-is-final", "every event on a Won deal, and a replayed win, by the owner and an Admin: 409, still Won, no audit row")
}

func TestTaskTerminalIsFinal(t *testing.T) {
	wants(t, "inv:task-terminal-is-final")
	for _, state := range []string{"Done", "Cancelled"} {
		for _, who := range []string{byOwner, byAdmin} {
			a := arrange(t, "task", state, arrangement{who, 0})
			for _, event := range []string{"start", "complete", "cancel"} {
				finalRefused(t, a, event, state)
			}
		}
	}
	pass(t, "inv:task-terminal-is-final", "start, complete and cancel on a Done and on a Cancelled task, by the assignee and an Admin: 409, unchanged")
}

func TestUserEmailUnique(t *testing.T) {
	wants(t, "inv:user-email-unique")
	email := unique("dup") + "@example.test"
	if code, b := call(t, "POST", "/users", "", map[string]string{"email": email, "role": "Rep"}); code != http.StatusCreated {
		t.Fatalf("first POST /users = %d %s, want 201", code, b)
	}
	for _, again := range []string{email, strings.ToUpper(email)} {
		var e apiError
		if code := callInto(t, "POST", "/users", "", map[string]string{"email": again, "role": "Admin"}, &e); code != http.StatusConflict || e.Code != "EMAIL_TAKEN" {
			t.Fatalf("POST /users with %s after %s = %d %+v, want 409 EMAIL_TAKEN", again, email, code, e)
		}
	}
	var n int
	if err := adminPool.QueryRow(t.Context(), "SELECT count(*) FROM users WHERE lower(email) = lower($1)", email).Scan(&n); err != nil || n != 1 {
		t.Fatalf("users with email %s in any case: %d (%v), want 1", email, n, err)
	}
	pass(t, "inv:user-email-unique", "a second user with the same email, and with it in upper case: 409 EMAIL_TAKEN, one row")
}

// TestAnonymousAndMalformed sends every write operation of the contract
// without a user, with an unknown user, with a header that is not a uuid,
// and with a malformed record id. Nothing may change.
func TestAnonymousAndMalformed(t *testing.T) {
	admin := createUser(t, "Admin")
	deal := arrange(t, "deal", "Qualified", arrangement{byOwner, 1000})
	task := arrange(t, "task", "Open", arrangement{byOwner, 0})
	ids := map[string]string{"deal": deal.id, "task": task.id}
	bodies := map[string]any{
		"/deals": map[string]any{"title": unique("anon"), "amountCents": 1},
		"/tasks": map[string]any{"title": unique("anon"), "dealId": deal.id},
	}
	n := 0
	for _, op := range contractOperations(t) {
		if op.method != "POST" || op.path == "/users" {
			continue
		}
		n++
		kind := strings.TrimSuffix(strings.Split(op.path, "/")[1], "s")
		path := strings.ReplaceAll(op.path, "{id}", ids[kind])
		before := counts(t)
		for _, user := range []string{"", "6b1f0c38-3a5d-4c39-9a4e-0000000000aa", "not-a-uuid"} {
			var e apiError
			if code := callInto(t, op.method, path, user, bodies[op.path], &e); code != http.StatusUnauthorized || e.Code != "UNAUTHENTICATED" {
				t.Errorf("%s %s as %q = %d %+v, want 401 UNAUTHENTICATED", op.method, path, user, code, e)
			}
		}
		if strings.Contains(op.path, "{id}") {
			bad := strings.ReplaceAll(op.path, "{id}", "not-a-uuid")
			var e apiError
			if code := callInto(t, op.method, bad, admin, nil, &e); code != http.StatusBadRequest || e.Code != "BAD_REQUEST" {
				t.Errorf("%s %s = %d %+v, want 400 BAD_REQUEST", op.method, bad, code, e)
			}
		}
		wantCounts(t, before)
	}
	if n == 0 {
		t.Fatal("no write operation read from the contract")
	}
	wantRow(t, "deals", deal.id, "Qualified")
	wantRow(t, "tasks", task.id, "Open")
	wantTrail(t, deal.subject())
	wantTrail(t, task.subject())
}

// rowCounts is the number of rows of every table a write can touch.
type rowCounts struct{ users, deals, tasks, auditEntries int }

func counts(t *testing.T) rowCounts {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var c rowCounts
	err := adminPool.QueryRow(ctx, `SELECT (SELECT count(*) FROM users), (SELECT count(*) FROM deals),
		(SELECT count(*) FROM tasks), (SELECT count(*) FROM audit_entries)`).Scan(&c.users, &c.deals, &c.tasks, &c.auditEntries)
	if err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return c
}

func wantCounts(t *testing.T, want rowCounts) {
	t.Helper()
	if got := counts(t); got != want {
		t.Fatalf("row counts = %+v, want %+v", got, want)
	}
}
