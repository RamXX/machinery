package e2e

import (
	"context"
	"testing"
	"time"
)

// The callers a case can be fired by, relative to the record.
const (
	byOwner = "owner" // the Rep who owns the deal or is assigned the task
	byAdmin = "admin" // an Admin who neither owns nor is assigned it
	byOther = "other" // another Rep
)

// arrangement is the shape of data a case needs: who fires the event and
// the amount of the deal.
type arrangement struct {
	caller string
	amount int64
}

// arranged is a record put in a given state, with its users.
type arranged struct {
	kind   string // the machine: deal or task
	id     string
	owner  string // the owner of the deal or the assignee of the task
	caller string // the user who fires the event
}

// subject is how the audit trail names the record.
func (a arranged) subject() string { return a.kind + ":" + a.id }

// path is the event operation of the record.
func (a arranged) path(event string) string { return "/" + a.kind + "s/" + a.id + "/" + event }

// arrange creates a fresh owner, puts a new record of the machine in state
// with SQL through the admin pool, and picks the caller.
func arrange(t *testing.T, kind, state string, how arrangement) arranged {
	t.Helper()
	a := arranged{kind: kind, owner: createUser(t, "Rep")}
	switch kind {
	case "deal":
		a.id = insertDeal(t, a.owner, state, how.amount)
	case "task":
		a.id = insertTask(t, a.owner, insertDeal(t, a.owner, "Lead", 1000), state)
	default:
		t.Fatalf("arrange: no arrangement for machine %q", kind)
	}
	switch how.caller {
	case byOwner:
		a.caller = a.owner
	case byAdmin:
		a.caller = createUser(t, "Admin")
	case byOther:
		a.caller = createUser(t, "Rep")
	default:
		t.Fatalf("arrange: unknown caller %q", how.caller)
	}
	return a
}

// insertTask writes a task row with SQL through the admin pool.
func insertTask(t *testing.T, assignee, deal, status string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var id string
	err := adminPool.QueryRow(ctx, `INSERT INTO tasks (title, status, deal_id, assignee_id)
		VALUES ($1, $2, $3::uuid, $4::uuid) RETURNING id::text`, unique("task"), status, deal, assignee).Scan(&id)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}
	return id
}

// auditCount counts the audit rows of one subject.
func auditCount(t *testing.T, subject string) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var n int
	if err := adminPool.QueryRow(ctx, "SELECT count(*) FROM audit_entries WHERE subject = $1", subject).Scan(&n); err != nil {
		t.Fatalf("count audit rows of %s: %v", subject, err)
	}
	return n
}
