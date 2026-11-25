package e2e

// The database refuses any state change of a deal or a task by the app role
// that is not an edge of its machine, made inside the transition path. The
// adversary is application code, so these tests connect as crm_app.
//
// machinery:verifies inv:lifecycle-single-writer

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// singleWriterHeld records which of the four adversary tests held; the
// last of them emits the receipt when all four did.
var singleWriterHeld = map[string]bool{}

const singleWriterTests = 4

func heldSingleWriter(t *testing.T) {
	t.Helper()
	if t.Failed() {
		return
	}
	singleWriterHeld[t.Name()] = true
	if len(singleWriterHeld) == singleWriterTests {
		pass(t, "inv:lifecycle-single-writer", "as crm_app: a direct state update, a forged crm.transition on a forbidden edge and an insert in a non-initial state are refused with lifecycle:; lifecycle_edges holds exactly the machine edges")
	}
}

// appPool connects as the app role, closed when the test ends.
func appPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := connect(context.Background(), appURL)
	if err != nil {
		t.Fatalf("connect as crm_app: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// wantLifecycleError expects an error raised by the lifecycle trigger.
func wantLifecycleError(t *testing.T, what string, err error) {
	t.Helper()
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || !strings.HasPrefix(pe.Message, "lifecycle:") {
		t.Fatalf("%s as crm_app: %v, want an error starting lifecycle:", what, err)
	}
}

func ctx3(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// TestDirectUpdateRefused: an UPDATE of the state column outside the
// transition path fails, on a forbidden edge and on an allowed one.
func TestDirectUpdateRefused(t *testing.T) {
	wants(t, "inv:lifecycle-single-writer")
	app, rep := appPool(t), createUser(t, "Rep")
	for _, stage := range []string{"Won", "Qualified"} {
		deal := insertDeal(t, rep, "Lead", 1000)
		_, err := app.Exec(ctx3(t), "UPDATE deals SET stage = $2 WHERE id = $1::uuid", deal, stage)
		wantLifecycleError(t, "UPDATE deals SET stage = "+stage+" on a Lead", err)
		wantRow(t, "deals", deal, "Lead")
	}
	task := insertTask(t, rep, insertDeal(t, rep, "Lead", 1000), "Open")
	for _, status := range []string{"Done", "InProgress"} {
		_, err := app.Exec(ctx3(t), "UPDATE tasks SET status = $2 WHERE id = $1::uuid", task, status)
		wantLifecycleError(t, "UPDATE tasks SET status = "+status+" on an Open task", err)
		wantRow(t, "tasks", task, "Open")
	}
	heldSingleWriter(t)
}

// TestForgedSettingOnForbiddenEdge: setting crm.transition by hand does not
// open an edge the machine does not have.
func TestForgedSettingOnForbiddenEdge(t *testing.T) {
	wants(t, "inv:lifecycle-single-writer")
	app, rep := appPool(t), createUser(t, "Rep")
	deal := insertDeal(t, rep, "Lead", 1000)
	task := insertTask(t, rep, insertDeal(t, rep, "Lead", 1000), "Open")
	for _, q := range []struct{ what, sql, id string }{
		{"Lead to Won", "UPDATE deals SET stage = 'Won' WHERE id = $1::uuid", deal},
		{"Open to Done", "UPDATE tasks SET status = 'Done' WHERE id = $1::uuid", task},
	} {
		err := pgx.BeginFunc(ctx3(t), app, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx3(t), "SELECT set_config('crm.transition', 'tc:FORGED-000000', true)"); err != nil {
				t.Fatalf("set crm.transition: %v", err)
			}
			_, err := tx.Exec(ctx3(t), q.sql, q.id)
			return err
		})
		wantLifecycleError(t, "a forged crm.transition on "+q.what, err)
	}
	wantRow(t, "deals", deal, "Lead")
	wantRow(t, "tasks", task, "Open")
	heldSingleWriter(t)
}

// TestInsertInNonInitialState: the app role may create a record only in
// its machine's initial state.
func TestInsertInNonInitialState(t *testing.T) {
	wants(t, "inv:lifecycle-single-writer")
	app, rep := appPool(t), createUser(t, "Rep")
	title := unique("forged")
	insert := func(stage string) error {
		_, err := app.Exec(ctx3(t), "INSERT INTO deals (title, stage, amount_cents, owner_id) VALUES ($1, $2, 1000, $3::uuid)", title, stage, rep)
		return err
	}
	wantLifecycleError(t, "INSERT a Qualified deal", insert("Qualified"))
	if err := insert("Lead"); err != nil {
		t.Fatalf("INSERT a Lead deal as crm_app: %v", err)
	}
	var n int
	if err := adminPool.QueryRow(ctx3(t), "SELECT count(*) FROM deals WHERE title = $1", title).Scan(&n); err != nil || n != 1 {
		t.Fatalf("deals titled %s: %d (%v), want the one Lead", title, n, err)
	}
	deal := insertDeal(t, rep, "Lead", 1000)
	_, err := app.Exec(ctx3(t), "INSERT INTO tasks (title, status, deal_id, assignee_id) VALUES ($1, 'Done', $2::uuid, $3::uuid)", title, deal, rep)
	wantLifecycleError(t, "INSERT a Done task", err)
	if _, err := app.Exec(ctx3(t), "INSERT INTO tasks (title, status, deal_id, assignee_id) VALUES ($1, 'Open', $2::uuid, $3::uuid)", title, deal, rep); err != nil {
		t.Fatalf("INSERT an Open task as crm_app: %v", err)
	}
	heldSingleWriter(t)
}

// TestEdgesMatchTables: lifecycle_edges holds exactly the persisted edges
// and the initial states of the committed tables.
func TestEdgesMatchTables(t *testing.T) {
	wants(t, "inv:lifecycle-single-writer")
	want := []string{
		"deal  -> Lead", "deal Lead -> Lost", "deal Lead -> Qualified", "deal Lost -> Lead",
		"deal Qualified -> Lost", "deal Qualified -> Won",
		"task  -> Open", "task InProgress -> Cancelled", "task InProgress -> Done",
		"task Open -> Cancelled", "task Open -> InProgress",
	}
	rows, err := adminPool.Query(ctx3(t), "SELECT machine || ' ' || source || ' -> ' || target FROM lifecycle_edges ORDER BY machine, source, target")
	if err != nil {
		t.Fatalf("read lifecycle_edges: %v", err)
	}
	got, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("read lifecycle_edges: %v", err)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("lifecycle_edges = %q, want %q", got, want)
	}
	app := appPool(t)
	if _, err := app.Exec(ctx3(t), "DELETE FROM lifecycle_edges"); err == nil {
		t.Fatal("crm_app deleted from lifecycle_edges, want permission denied")
	}
	heldSingleWriter(t)
}

// TestServicePathStillWorks: the service's own transition path passes the
// trigger: create, qualify, win over HTTP.
func TestServicePathStillWorks(t *testing.T) {
	rep := createUser(t, "Rep")
	id := newDeal(t, rep, 1000)
	qualify(t, id, rep)
	if c := fire(t, "/deals/"+id+"/win", rep); c.State != "Won" {
		t.Fatalf("win = %+v, want Won", c)
	}
	wantRow(t, "deals", id, "Won")
}

// TestAdminArrangeUnaffected: the admin role, which the tests arrange data
// with, may still insert and update a record in any state.
func TestAdminArrangeUnaffected(t *testing.T) {
	rep := createUser(t, "Rep")
	for _, stage := range []string{"Lead", "Qualified", "closing", "Won", "Lost"} {
		wantRow(t, "deals", insertDeal(t, rep, stage, 1000), stage)
	}
	deal := insertDeal(t, rep, "Lead", 1000)
	if _, err := adminPool.Exec(ctx3(t), "UPDATE deals SET stage = 'Won' WHERE id = $1::uuid", deal); err != nil {
		t.Fatalf("UPDATE a Lead to Won as the admin role: %v", err)
	}
	wantRow(t, "deals", deal, "Won")
}
