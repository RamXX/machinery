package e2e

// TestReplayDealWinRace replays the counterexample of the deal-win race run
// against the real service. The trace shows two callers that both read a
// Qualified deal (their begin steps) before either writes (their commit
// steps), so the deal is won twice. For each round the test creates a fresh
// qualified deal, holds its row from an admin transaction until both win
// requests are waiting on it, so both are in flight before either commits,
// then lets them go and checks the property the trace violates on the real
// database. Assumption A1 says the row lock keeps the trace from being
// followed to its end: the replay must be refuted.
//
// machinery:verifies cx:deal-win:race

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/v2/pkg/receipt"
)

const (
	raceTrace    = "../design/formal/traces/deal-win.race.trace.json"
	replayRounds = 60
)

// traceFile is the part of a counterexample trace (docs/formats.md section
// 16.10) the replay reads.
type traceFile struct {
	ID       string `json:"id"`
	Violates string `json:"violates"`
	Expect   string `json:"expect"`
	Machine  string `json:"machine"`
	Steps    []struct {
		Case     string            `json:"case"`
		Instance string            `json:"instance"`
		Caller   string            `json:"caller"`
		Step     string            `json:"step"`
		State    map[string]string `json:"state"`
	} `json:"steps"`
}

// race is the concurrent part of a trace: callers whose begin steps all come
// before their commit steps, on one instance in one state.
type race struct {
	state   string   // the state every begin step read
	callers []string // in the order of their begin steps
	events  []string // the event each caller fires
}

func readTrace(t *testing.T, path string) traceFile {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the trace: %v", err)
	}
	var tf traceFile
	if err := json.Unmarshal(b, &tf); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return tf
}

// concurrent finds the first point of the trace where two callers have a
// step begun and not committed, and returns the callers in flight there.
func concurrent(t *testing.T, tf traceFile, tb caseTable) race {
	t.Helper()
	event := map[string]string{}
	for _, c := range tb.Transitions {
		event[c.ID] = c.Event
	}
	var r race
	open := map[string]bool{}
	for _, s := range tf.Steps {
		switch s.Step {
		case "begin":
			if len(open) == 0 {
				r = race{state: s.State[s.Instance]}
			}
			open[s.Caller] = true
			r.callers = append(r.callers, s.Caller)
			r.events = append(r.events, event[s.Case])
		case "commit":
			if len(open) >= 2 {
				return r
			}
			delete(open, s.Caller)
		}
	}
	t.Fatalf("%s: no two callers in flight at once", tf.ID)
	return r
}

// fireAll sends the events of the race at once and returns each status code.
func fireAll(dealID string, users, events []string) []int {
	codes := make([]int, len(users))
	done := make(chan struct{}, len(users))
	for k := range users {
		go func(k int) {
			defer func() { done <- struct{}{} }()
			req, err := http.NewRequest("POST", baseURL+"/deals/"+dealID+"/"+events[k], nil)
			if err != nil {
				return
			}
			req.Header.Set("X-User-Id", users[k])
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return
			}
			resp.Body.Close()
			codes[k] = resp.StatusCode
		}(k)
	}
	for range users {
		<-done
	}
	return codes
}

// holdRow locks the deal row from an admin transaction and returns a release
// function that waits until n requests of the service are waiting on a lock,
// then lets them go. No other test runs at the same time.
func holdRow(t *testing.T, dealID string) func(n int) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	tx, err := adminPool.Begin(ctx)
	if err != nil {
		cancel()
		t.Fatalf("begin the holding transaction: %v", err)
	}
	if _, err := tx.Exec(ctx, "SELECT 1 FROM deals WHERE id = $1::uuid FOR UPDATE", dealID); err != nil {
		cancel()
		t.Fatalf("lock deal %s: %v", dealID, err)
	}
	return func(n int) error {
		defer cancel()
		defer tx.Rollback(context.Background())
		deadline := time.Now().Add(1500 * time.Millisecond)
		for {
			var waiting int
			err := adminPool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity
				WHERE usename = 'crm_app' AND wait_event_type = 'Lock'`).Scan(&waiting)
			if err != nil {
				return err
			}
			if waiting >= n {
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("%d of %d requests waiting on the row lock", waiting, n)
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
}

// replayRound runs the race once on a fresh deal and reports whether the bad
// outcome of the trace occurred: more than one success or more than one win
// audit entry.
func replayRound(t *testing.T, r race, users []string) (bool, string) {
	t.Helper()
	a := arranged{kind: "deal", owner: users[0], id: insertDeal(t, users[0], r.state, 1000)}
	release := holdRow(t, a.id)
	codes := make(chan []int, 1)
	go func() { codes <- fireAll(a.id, users, r.events) }()
	if err := release(len(users)); err != nil {
		<-codes
		t.Fatalf("deal %s: %v", a.id, err)
	}
	got := <-codes
	ok := 0
	for _, c := range got {
		if c == http.StatusOK {
			ok++
		}
	}
	audits := 0
	for _, u := range users {
		audits += auditRows(t, a.subject(), "win", u, "applied")
	}
	wins := winRows(t, a)
	desc := fmt.Sprintf("deal %s: answers %v, %d win audit row(s), %d deal_wins row(s), stage %s", a.id, got, audits, wins, rowState(t, "deals", a.id))
	if ok > 1 || audits > 1 || wins > 1 {
		return true, desc
	}
	if ok != 1 || audits != 1 || wins != 1 || rowState(t, "deals", a.id) != "Won" {
		t.Fatalf("%s, want one 200 and one win", desc)
	}
	for _, c := range got {
		if c != http.StatusOK && c != http.StatusConflict {
			t.Fatalf("%s, want the other caller refused with 409", desc)
		}
	}
	return false, desc
}

func TestReplayDealWinRace(t *testing.T) {
	wants(t, "cx:deal-win:race")
	tf := readTrace(t, raceTrace)
	if tf.ID != "cx:deal-win:race" || tf.Expect != receipt.Refuted || tf.Violates != "WinAuditedOnce" {
		t.Fatalf("%s: id %s, expect %s, violates %s: not the trace this test replays", raceTrace, tf.ID, tf.Expect, tf.Violates)
	}
	var deal caseTable
	for _, tb := range loadTables(t) {
		if tb.Machine == tf.Machine {
			deal = tb
		}
	}
	r := concurrent(t, tf, deal)
	if r.state != "Qualified" || strings.Join(r.events, ",") != "win,win" {
		t.Fatalf("%s: in flight at once in %s: %v firing %v; want two wins on a Qualified deal", tf.ID, r.state, r.callers, r.events)
	}
	// The first caller of the trace is the owner, every other one an Admin:
	// each of them may win the deal on its own.
	owner := createUser(t, "Rep")
	users := []string{owner}
	for range r.callers[1:] {
		users = append(users, createUser(t, "Admin"))
	}
	var bad []string
	for range replayRounds {
		if reproduced, desc := replayRound(t, r, users); reproduced {
			bad = append(bad, desc)
		}
	}
	if len(bad) > 0 {
		obs := fmt.Sprintf("%d of %d rounds won one deal twice: %s", len(bad), replayRounds, bad[0])
		if err := receipt.Replay(tf.ID, receipt.Test(0, t.Name()), receipt.Reproduced, obs); err != nil {
			t.Errorf("emit receipt %s: %v", tf.ID, err)
		}
		t.Fatalf("assumption A1 is false in the real service: %s", obs)
	}
	obs := fmt.Sprintf("%d rounds: %s by owner and Admin at once, both held on the row lock before either wrote: one 200, one 409, one win audit row, one deal_wins row each", replayRounds, strings.Join(r.events, " and "))
	if err := receipt.Replay(tf.ID, receipt.Test(0, t.Name()), receipt.Refuted, obs); err != nil {
		t.Errorf("emit receipt %s: %v", tf.ID, err)
	}
}
