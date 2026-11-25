package e2e

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// machinery:verifies inv:every-change-audited inv:refusals-are-audited inv:audit-append-only

// TestEveryChangeAudited: a deal created, qualified and lost and a task
// started and completed have one applied row per change, in order, by the
// caller, and nothing else.
func TestEveryChangeAudited(t *testing.T) {
	wants(t, "inv:every-change-audited")
	rep := createUser(t, "Rep")
	deal := newDeal(t, rep, 1000)
	wantTrail(t, "deal:"+deal)
	qualify(t, deal, rep)
	wantTrail(t, "deal:"+deal, trailRow{rep, "qualify", "applied"})
	fire(t, "/deals/"+deal+"/lose", rep)
	wantRow(t, "deals", deal, "Lost")
	wantTrail(t, "deal:"+deal, trailRow{rep, "qualify", "applied"}, trailRow{rep, "lose", "applied"})
	task := newTask(t, rep)
	fire(t, "/tasks/"+task+"/start", rep)
	fire(t, "/tasks/"+task+"/complete", rep)
	wantTrail(t, "task:"+task, trailRow{rep, "start", "applied"}, trailRow{rep, "complete", "applied"})
	pass(t, "inv:every-change-audited", "create, qualify, lose of a deal and start, complete of a task: one applied row per change, in order, by the caller")
}

// TestRefusalsAudited: a denied qualify and a denied start each write
// exactly one refused row and no applied row.
func TestRefusalsAudited(t *testing.T) {
	wants(t, "inv:refusals-are-audited")
	rep, other := createUser(t, "Rep"), createUser(t, "Rep")
	deal := newDeal(t, rep, 1000)
	refused(t, "/deals/"+deal+"/qualify", other, http.StatusForbidden)
	wantRow(t, "deals", deal, "Lead")
	wantTrail(t, "deal:"+deal, trailRow{other, "dealdenied", "refused"})
	task := newTask(t, rep)
	refused(t, "/tasks/"+task+"/start", other, http.StatusForbidden)
	wantRow(t, "tasks", task, "Open")
	wantTrail(t, "task:"+task, trailRow{other, "taskdenied", "refused"})
	pass(t, "inv:refusals-are-audited", "a denied qualify and a denied start: 403, one refused row each, no applied row")
}

// TestAuditAppendOnly: UPDATE, DELETE and TRUNCATE of the audit trail fail
// even for the admin role, and the row is unchanged.
func TestAuditAppendOnly(t *testing.T) {
	wants(t, "inv:audit-append-only")
	rep := createUser(t, "Rep")
	deal := newDeal(t, rep, 1000)
	qualify(t, deal, rep)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var id int64
	if err := adminPool.QueryRow(ctx, "SELECT id FROM audit_entries WHERE subject = $1", "deal:"+deal).Scan(&id); err != nil {
		t.Fatalf("read the audit row of deal %s: %v", deal, err)
	}
	for _, q := range []string{
		"UPDATE audit_entries SET outcome = 'refused' WHERE id = $1",
		"DELETE FROM audit_entries WHERE id = $1",
	} {
		if _, err := adminPool.Exec(ctx, q, id); err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Fatalf("%s as the admin role: %v, want the append-only error", q, err)
		}
	}
	tx, err := adminPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, "TRUNCATE audit_entries")
	_ = tx.Rollback(ctx)
	if err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("TRUNCATE audit_entries as the admin role: %v, want the append-only error", err)
	}
	wantTrail(t, "deal:"+deal, trailRow{rep, "qualify", "applied"})
	pass(t, "inv:audit-append-only", "UPDATE, DELETE and TRUNCATE of audit_entries as the admin role fail with the append-only error; the row is unchanged")
}

// TestAuditAtomicWithChange: when the state change of a task fails in the
// database, its audit row is rolled back with it.
func TestAuditAtomicWithChange(t *testing.T) {
	rep := createUser(t, "Rep")
	task := newTask(t, rep)
	name := "crm_e2e_fail_" + strings.ReplaceAll(task, "-", "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := adminPool.Exec(ctx, "DROP TRIGGER IF EXISTS "+name+" ON tasks; DROP FUNCTION IF EXISTS "+name+"()"); err != nil {
			t.Errorf("drop trigger %s: %v", name, err)
		}
	})
	if _, err := adminPool.Exec(ctx, `CREATE FUNCTION `+name+`() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
	IF OLD.id = '`+task+`' THEN
		RAISE EXCEPTION 'e2e: the update of task `+task+` fails';
	END IF;
	RETURN NEW;
END $$;
CREATE TRIGGER `+name+` BEFORE UPDATE ON tasks FOR EACH ROW EXECUTE FUNCTION `+name+`()`); err != nil {
		t.Fatalf("install trigger %s: %v", name, err)
	}
	code, ans := post(t, "/tasks/"+task+"/start", rep)
	if code < 500 || ans.Code == "" {
		t.Fatalf("POST start with the update failing = %d %+v, want a 5xx error", code, ans)
	}
	wantRow(t, "tasks", task, "Open")
	wantTrail(t, "task:"+task)
}
