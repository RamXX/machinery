package e2e

// The failure paths of the invoked actors, induced on the real Postgres at
// the boundary the service really uses, and the database taken away.
//
// These tests change how the database behaves for every client: a revoked
// privilege, a held table lock, the container paused. They must not run in
// parallel with any other test, so no test of this suite calls t.Parallel
// (go test then runs them one at a time) and the e2e command passes -p 1.
// Every change is undone in a cleanup that always runs.
//
// machinery:verifies fm:deal-win-store-failure fm:database-unreachable

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// pgContainer is the one Postgres container of make up.
var pgContainer = env("CRM_PG_CONTAINER", "machinery-crm-pg")

// lockHold is how long lockWinTable holds its lock: longer than the
// closeTimeout delay, so only the deadline can end the write.
const lockHold = 2 * time.Second

// inducer puts a failure in place for the store write of an actor and
// returns its undo, which waits until the failure no longer holds. The undo
// is also registered as a cleanup, so it always runs, once.
type inducer func(t *testing.T) (undo func())

// actorHooks is what the suite knows about one invoked actor: how to make
// its write fail (onError) or outlast its deadline (after), and how many
// rows it wrote for a record.
type actorHooks struct {
	fail, stall inducer
	rows        func(t *testing.T, a arranged) int
}

var actors = map[string]actorHooks{
	"recordWin": {fail: revokeWinInsert, stall: lockWinTable, rows: winRows},
}

// once registers undo as a cleanup and returns it, guarded so it runs once.
func once(t *testing.T, undo func()) func() {
	var o sync.Once
	f := func() { o.Do(undo) }
	t.Cleanup(f)
	return f
}

// revokeWinInsert takes INSERT on deal_wins away from crm_app, through the
// admin pool: the insert of recordWin fails with a permission error.
func revokeWinInsert(t *testing.T) func() {
	t.Helper()
	admin(t, "REVOKE INSERT ON deal_wins FROM crm_app")
	return once(t, func() { admin(t, "GRANT INSERT ON deal_wins TO crm_app") })
}

// lockWinTable holds ACCESS EXCLUSIVE on deal_wins from a second admin
// connection for lockHold: the insert of recordWin waits on the lock.
func lockWinTable(t *testing.T) func() {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect a second admin connection: %v", err)
	}
	tx, err := conn.Begin(ctx)
	if err == nil {
		_, err = tx.Exec(ctx, "LOCK TABLE deal_wins IN ACCESS EXCLUSIVE MODE")
	}
	if err != nil {
		_ = conn.Close(context.Background())
		t.Fatalf("lock deal_wins: %v", err)
	}
	until := time.Now().Add(lockHold)
	return once(t, func() {
		time.Sleep(time.Until(until))
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("release the lock on deal_wins: %v", err)
		}
		_ = conn.Close(ctx)
	})
}

func admin(t *testing.T, sql string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := adminPool.Exec(ctx, sql); err != nil {
		t.Errorf("%s: %v", sql, err)
	}
}

// winRows counts the deal_wins rows of a deal.
func winRows(t *testing.T, a arranged) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var n int
	if err := adminPool.QueryRow(ctx, "SELECT count(*) FROM deal_wins WHERE deal_id = $1::uuid", a.id).Scan(&n); err != nil {
		t.Fatalf("count deal_wins rows of %s: %v", a.id, err)
	}
	return n
}

// failedCase arranges a record in the state before the invoking state,
// induces the failure the case's kind needs, fires the event that enters the
// invoking state and checks the answer: 503 STORE_UNAVAILABLE naming the
// case, in time, the row back in the case's target, the actor's write
// undone. The effects are left to the caller to observe.
func failedCase(t *testing.T, tb caseTable, c tcase) (arranged, string) {
	t.Helper()
	actor := invokeOf(tb, c.Source)
	hooks, ok := actors[actor]
	if !ok {
		t.Fatalf("%s: no failure hooks for actor %q of state %s", c.ID, actor, c.Source)
	}
	induce, limit := hooks.fail, time.Duration(tb.Delays[timerOf(tb, c.Source)])*time.Millisecond
	if c.Kind == "after" {
		induce = hooks.stall
	}
	i := slices.IndexFunc(tb.Transitions, func(o tcase) bool { return o.Kind == "on" && o.Target == c.Source })
	if i < 0 || limit == 0 {
		t.Fatalf("%s: no on branch enters %s, or it has no after delay", c.ID, c.Source)
	}
	in := tb.Transitions[i]
	a := arrange(t, tb.Machine, in.Source, arrangementFor(t, tb, in))
	undo := induce(t)
	began := time.Now()
	code, ans := post(t, a.path(in.Event), a.caller)
	took := time.Since(began)
	undo()
	want := []string{in.ID, c.ID}
	if code != http.StatusServiceUnavailable || ans.Code != "STORE_UNAVAILABLE" || ans.Case != c.ID || !sameList(ans.Cases, want) {
		t.Fatalf("POST %s with %s induced = %d %+v, want 503 STORE_UNAVAILABLE case %s cases %v", a.path(in.Event), c.Kind, code, ans, c.ID, want)
	}
	switch {
	case c.Kind == "after" && (took < limit || took > limit+time.Second):
		t.Fatalf("%s: the answer took %s, want about the %s deadline", c.ID, took, limit)
	case c.Kind == "onError" && took >= limit:
		t.Fatalf("%s: the answer took %s, want less than the %s deadline", c.ID, took, limit)
	}
	wantRow(t, tb.Machine+"s", a.id, c.Target)
	if n := hooks.rows(t, a); n != 0 {
		t.Fatalf("%s: %s left %d rows after it failed, want 0", c.ID, actor, n)
	}
	failed := 0
	for _, r := range trail(t, a.subject()) {
		if r.outcome == "failed" {
			failed++
		}
	}
	if failed != 1 {
		t.Fatalf("%s: %d failed audit rows, want 1", a.subject(), failed)
	}
	return a, fmt.Sprintf("%s %s with %s %s: 503 STORE_UNAVAILABLE after %s, the row is %s, no %s row, one failed audit row",
		in.Source, in.Event, actor, map[string]string{"onError": "failing", "after": "stalled"}[c.Kind], took.Round(100*time.Millisecond), c.Target, actor)
}

// invokeOf is the actor a state invokes, or "".
func invokeOf(tb caseTable, state string) string {
	for _, s := range tb.States {
		if s.Path == state && len(s.Invoke) > 0 {
			return s.Invoke[0]
		}
	}
	return ""
}

// timerOf is the delay of the after branch of an invoking state.
func timerOf(tb caseTable, state string) string {
	for _, o := range tb.Transitions {
		if o.Source == state && o.Kind == "after" {
			return o.Event
		}
	}
	return ""
}

// runFailure drives an onError or after case of the table walker.
func runFailure(t *testing.T, tb caseTable, c tcase) (string, []string) {
	a, observed := failedCase(t, tb, c)
	return observed, observe(t, a, c.Effects)
}

// TestDealWinStoreFailure: the win's store write fails and times out; in
// neither case is a deal Won, or a deal_wins row there, without its win
// audit row, and once the failure is gone the same win succeeds.
func TestDealWinStoreFailure(t *testing.T) {
	wants(t, "fm:deal-win-store-failure")
	var deal caseTable
	for _, tb := range loadTables(t) {
		if tb.Machine == "deal" {
			deal = tb
		}
	}
	kinds := 0
	for _, c := range deal.Transitions {
		if invokeOf(deal, c.Source) != "recordWin" || (c.Kind != "onError" && c.Kind != "after") {
			continue
		}
		kinds++
		a, _ := failedCase(t, deal, c)
		wantWinAudited(t, a, false)
		if ans := fire(t, a.path("win"), a.caller); ans.State != "Won" {
			t.Fatalf("win of %s after the failure is gone = %+v, want Won", a.id, ans)
		}
		wantWinAudited(t, a, true)
	}
	if kinds != 2 {
		t.Fatalf("the deal table has %d failure branches of recordWin, want 2 (onError, after)", kinds)
	}
	pass(t, "fm:deal-win-store-failure", "recordWin failing and stalled: 503 STORE_UNAVAILABLE, back to Qualified, no deal_wins row, no Won deal without its win audit row; the same win then succeeds")
}

// wantWinAudited checks that a deal is Won with a deal_wins row and one win
// audit row, or none of the three.
func wantWinAudited(t *testing.T, a arranged, won bool) {
	t.Helper()
	stage, rows, audits := rowState(t, "deals", a.id), winRows(t, a), auditRows(t, a.subject(), "win", a.caller, "applied")
	if won != (stage == "Won") || won != (rows == 1) || won != (audits == 1) || rows > 1 || audits > 1 {
		t.Fatalf("deal %s: stage %s, %d deal_wins rows, %d win audit rows; want Won, 1, 1 together or none", a.id, stage, rows, audits)
	}
}

// docker runs one docker command on the Postgres container.
func docker(t *testing.T, verb string) {
	t.Helper()
	out, err := exec.Command("docker", verb, pgContainer).CombinedOutput()
	if err != nil {
		t.Fatalf("docker %s %s: %v: %s", verb, pgContainer, err, out)
	}
}

// timed sends one request with a client that gives up after ten seconds and
// returns the status, the body and how long the answer took.
func timed(t *testing.T, method, path, user string) (int, string, time.Duration) {
	t.Helper()
	req, err := http.NewRequest(method, baseURL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-User-Id", user)
	client := http.Client{Timeout: 10 * time.Second}
	began := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b), time.Since(began)
}

// TestDatabaseUnreachable pauses the Postgres container: a read and a
// write answer 503 STORE_UNAVAILABLE within three seconds each and change
// nothing; after the container is unpaused the same write succeeds.
func TestDatabaseUnreachable(t *testing.T) {
	wants(t, "fm:database-unreachable")
	rep := createUser(t, "Rep")
	id := newDeal(t, rep, 1000)
	before := counts(t)
	docker(t, "pause")
	paused := true
	unpause := func() {
		if paused {
			paused = false
			docker(t, "unpause")
		}
	}
	t.Cleanup(unpause)
	for _, req := range [][2]string{{"GET", "/deals/" + id}, {"POST", "/deals/" + id + "/qualify"}, {"GET", "/healthz"}} {
		code, body, took := timed(t, req[0], req[1], rep)
		t.Logf("%s %s with the database paused: %d after %s", req[0], req[1], code, took)
		if code != http.StatusServiceUnavailable || !strings.Contains(body, `"STORE_UNAVAILABLE"`) || took > 3*time.Second {
			t.Fatalf("%s %s with the database paused = %d %s after %s, want 503 STORE_UNAVAILABLE within 3s", req[0], req[1], code, body, took)
		}
	}
	unpause()
	if err := waitStatus(baseURL+"/healthz", http.StatusOK, 15*time.Second); err != nil {
		t.Fatalf("the service did not recover after unpause: %v", err)
	}
	wantCounts(t, before)
	wantRow(t, "deals", id, "Lead")
	wantTrail(t, "deal:"+id)
	qualify(t, id, rep)
	wantTrail(t, "deal:"+id, trailRow{rep, "qualify", "applied"})
	pass(t, "fm:database-unreachable", "container paused: GET deal, POST qualify and healthz answer 503 STORE_UNAVAILABLE within 3s; unpaused: nothing changed, the same qualify succeeds")
}
