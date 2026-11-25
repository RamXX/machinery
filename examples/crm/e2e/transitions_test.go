package e2e

// TestTransitions walks the committed runtime tables: one subtest per
// transition, ignore and clause case, each driving the real service over
// HTTP and checking the real database. The tests always read
// design/gen/tables, never MACHINERY_TABLES_DIR: during a probe the service
// follows a mutated table while the tests still expect the committed one.
// The onError and after cases are induced on the real database by the
// hooks of failure_test.go.
//
// machinery:verifies tc:DEAL-* tc:TASK-*

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/RamXX/machinery/pkg/receipt"
)

// caseTable is the part of a runtime table (docs/formats.md section 7.3)
// the walker reads.
type caseTable struct {
	Machine string `json:"machine"`
	States  []struct {
		Path   string   `json:"path"`
		Invoke []string `json:"invoke"`
	} `json:"states"`
	Transitions []tcase `json:"transitions"`
	Ignores     []struct {
		ID, State, Event, Reason string
	} `json:"ignores"`
	Clauses []struct {
		ID, Transition, Guard, Clause string
	} `json:"clauses"`
	Delays map[string]int `json:"delays"`
}

type tcase struct {
	ID      string   `json:"id"`
	Source  string   `json:"source"`
	Kind    string   `json:"kind"`
	Event   string   `json:"event"`
	Guard   string   `json:"guard"`
	Target  string   `json:"target"`
	Effects []string `json:"effects"`
}

// guards maps each guard to a shape of data for which it holds and one for
// which it does not.
var guards = map[string]struct{ holds, fails arrangement }{
	"canWriteDeal": {arrangement{byOwner, 1000}, arrangement{byOther, 1000}},
	"canWinDeal":   {arrangement{byOwner, 1000}, arrangement{byOther, 1000}},
	"isAdmin":      {arrangement{byAdmin, 1000}, arrangement{byOwner, 1000}},
	"canWriteTask": {arrangement{byOwner, 0}, arrangement{byOther, 0}},
}

// clauseFalse maps each declared clause to the shape of data that makes it,
// and only it, false.
var clauseFalse = map[string]arrangement{
	"callerMayWriteDeal": {byOther, 1000},
	"amountIsPositive":   {byOwner, 0},
}

// refusalCode reads the API code an ignore reason promises
// ("... the API refuses with DEAL_LOST").
var refusalCode = regexp.MustCompile(`refuses with ([A-Z][A-Z0-9_]*)`)

// observer checks that an effect really happened to the arranged record; nil
// means it did. An effect without an observer is not claimed in a receipt.
type observer func(t *testing.T, a arranged) error

// observers maps each real effect to its check: the audit row it writes.
var observers = map[string]observer{
	"auditQualify":      auditRow("qualify", "applied"),
	"auditLose":         auditRow("lose", "applied"),
	"auditReopen":       auditRow("reopen", "applied"),
	"auditWin":          auditRow("win", "applied"),
	"stampWinAttempt":   auditRow("win-attempt", "applied"),
	"auditDealDenied":   auditRow("dealdenied", "refused"),
	"auditStart":        auditRow("start", "applied"),
	"auditComplete":     auditRow("complete", "applied"),
	"auditCancel":       auditRow("cancel", "applied"),
	"auditTaskDenied":   auditRow("taskdenied", "refused"),
	"auditCloseFailure": auditRow("closefailure", "failed"),
	"auditCloseTimeout": auditRow("closetimeout", "failed"),
}

// auditRow observes an effect that writes one audit row: exactly one row
// with the record as subject, the caller as actor, the action and outcome.
func auditRow(action, outcome string) observer {
	return func(t *testing.T, a arranged) error {
		n := auditRows(t, a.subject(), action, a.caller, outcome)
		if n != 1 {
			return fmt.Errorf("%d audit rows (%s, %s, %s), want 1", n, action, a.caller, outcome)
		}
		return nil
	}
}

// answer is either a Change or an Error of the contract.
type answer struct {
	change
	Code    string `json:"code"`
	Message string `json:"message"`
	Case    string `json:"case"`
}

func post(t *testing.T, path, user string) (int, answer) {
	t.Helper()
	code, b := call(t, "POST", path, user, nil)
	var a answer
	if err := json.Unmarshal(b, &a); err != nil {
		t.Fatalf("POST %s: %d answer is not JSON: %v: %s", path, code, err, b)
	}
	return code, a
}

func loadTables(t *testing.T) []caseTable {
	t.Helper()
	files, err := filepath.Glob("../design/gen/tables/*.table.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no tables in design/gen/tables (%v)", err)
	}
	var out []caseTable
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var tb caseTable
		if err := json.Unmarshal(b, &tb); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		out = append(out, tb)
	}
	return out
}

func TestTransitions(t *testing.T) {
	n := 0
	for _, tb := range loadTables(t) {
		for _, c := range tb.Transitions {
			switch c.Kind {
			case "on":
				n++
				t.Run(c.ID, func(t *testing.T) { driveCase(t, c.ID, func() (string, []string) { return runOn(t, tb, c) }) })
			case "onDone":
				n++
				t.Run(c.ID, func(t *testing.T) { driveCase(t, c.ID, func() (string, []string) { return runOnDone(t, tb, c) }) })
			case "onError", "after":
				n++
				t.Run(c.ID, func(t *testing.T) { driveCase(t, c.ID, func() (string, []string) { return runFailure(t, tb, c) }) })
			default:
				t.Errorf("%s: no driver for transition kind %q", c.ID, c.Kind)
			}
		}
		for _, ig := range tb.Ignores {
			n++
			t.Run(ig.ID, func(t *testing.T) {
				driveCase(t, ig.ID, func() (string, []string) { return runIgnore(t, tb.Machine, ig.State, ig.Event, ig.Reason, ig.ID) })
			})
		}
		for _, cl := range tb.Clauses {
			n++
			t.Run(cl.ID, func(t *testing.T) {
				driveCase(t, cl.ID, func() (string, []string) { return runClause(t, tb, cl.Transition, cl.Clause) })
			})
		}
	}
	if n == 0 {
		t.Fatal("the tables hold no case this suite can drive")
	}
}

// driveCase runs one case when the run asks for it, emits a fail receipt
// when an assertion failed and a pass receipt with the observed effects
// after every assertion held.
func driveCase(t *testing.T, id string, body func() (string, []string)) {
	if !receipt.Wanted(id) {
		t.Skipf("MACHINERY_ONLY does not ask for %s", id)
	}
	defer func() {
		if t.Failed() {
			if err := receipt.Fail(id, t.Name(), "an assertion failed; see the test log"); err != nil {
				t.Errorf("emit receipt %s: %v", id, err)
			}
		}
	}()
	observed, effects := body()
	if t.Failed() {
		return
	}
	if err := receipt.Pass(id, t.Name(), observed, effects...); err != nil {
		t.Errorf("emit receipt %s: %v", id, err)
	}
}

// arrangementFor picks the caller of a branch: one for whom its guard holds,
// or, for the unguarded fallback, one for whom the guard before it fails.
func arrangementFor(t *testing.T, tb caseTable, c tcase) arrangement {
	t.Helper()
	if c.Guard != "" {
		g, ok := guards[c.Guard]
		if !ok {
			t.Fatalf("%s: no arrangement for guard %s", c.ID, c.Guard)
		}
		return g.holds
	}
	var before []string
	for _, o := range tb.Transitions {
		if o.ID == c.ID {
			break
		}
		if o.Source == c.Source && o.Kind == c.Kind && o.Event == c.Event && o.Guard != "" {
			before = append(before, o.Guard)
		}
	}
	switch len(before) {
	case 0:
		return arrangement{byOwner, 1000}
	case 1:
		g, ok := guards[before[0]]
		if !ok {
			t.Fatalf("%s: no arrangement for guard %s", c.ID, before[0])
		}
		return g.fails
	}
	t.Fatalf("%s: no arrangement makes the guards %v false at once", c.ID, before)
	return arrangement{}
}

// settle follows the onDone branches from state while it invokes an actor,
// as a walk whose actors succeed does: the case ids and the resting state.
func settle(t *testing.T, tb caseTable, state string) ([]string, string) {
	t.Helper()
	var ids []string
	for step := 0; invokes(tb, state); step++ {
		if step > len(tb.States) {
			t.Fatalf("%s: the onDone walk from %s does not end", tb.Machine, state)
		}
		i := slices.IndexFunc(tb.Transitions, func(o tcase) bool {
			return o.Source == state && o.Kind == "onDone" && o.Guard == ""
		})
		if i < 0 {
			t.Fatalf("%s: state %s invokes an actor and has no onDone branch", tb.Machine, state)
		}
		ids, state = append(ids, tb.Transitions[i].ID), tb.Transitions[i].Target
	}
	return ids, state
}

func invokes(tb caseTable, state string) bool {
	for _, s := range tb.States {
		if s.Path == state {
			return len(s.Invoke) > 0
		}
	}
	return false
}

// observe runs the observer of each effect the case names and returns the
// effects seen. An effect with an observer that does not see it fails.
func observe(t *testing.T, a arranged, effects []string) []string {
	t.Helper()
	var seen []string
	for _, name := range effects {
		obs, ok := observers[name]
		if !ok {
			continue
		}
		if err := obs(t, a); err != nil {
			t.Fatalf("effect %s on %s not observed: %v", name, a.subject(), err)
		}
		seen = append(seen, name)
	}
	return seen
}

// runOn fires the event of an on branch from its source state.
func runOn(t *testing.T, tb caseTable, c tcase) (string, []string) {
	a := arrange(t, tb.Machine, c.Source, arrangementFor(t, tb, c))
	if c.Target == "" {
		code, ans := post(t, a.path(c.Event), a.caller)
		if code != http.StatusForbidden || ans.Code != "FORBIDDEN" || ans.Case != c.ID {
			t.Fatalf("POST %s = %d %+v, want 403 FORBIDDEN case %s", a.path(c.Event), code, ans, c.ID)
		}
		wantRow(t, tb.Machine+"s", a.id, c.Source)
		return c.Source + " " + c.Event + " by a caller the guard refuses: 403 FORBIDDEN, the row unchanged", observe(t, a, c.Effects)
	}
	rest, final := settle(t, tb, c.Target)
	want := append([]string{c.ID}, rest...)
	ans := fire(t, a.path(c.Event), a.caller)
	if ans.State != final || !sameList(ans.Cases, want) {
		t.Fatalf("POST %s = %+v, want state %s cases %v", a.path(c.Event), ans, final, want)
	}
	wantRow(t, tb.Machine+"s", a.id, final)
	return c.Source + " " + c.Event + ": 200, cases " + strings.Join(want, " ") + ", the row is " + final, observe(t, a, c.Effects)
}

// runOnDone reaches an invoking state through the on branch that enters it
// and observes the onDone case as the next case of the same answer.
func runOnDone(t *testing.T, tb caseTable, c tcase) (string, []string) {
	i := slices.IndexFunc(tb.Transitions, func(o tcase) bool { return o.Kind == "on" && o.Target == c.Source })
	if i < 0 {
		t.Fatalf("%s: no on branch enters %s", c.ID, c.Source)
	}
	in := tb.Transitions[i]
	a := arrange(t, tb.Machine, in.Source, arrangementFor(t, tb, in))
	rest, final := settle(t, tb, c.Source)
	if len(rest) == 0 || rest[0] != c.ID {
		t.Fatalf("%s: the walk from %s takes %v first, not this case", c.ID, c.Source, rest)
	}
	want := append([]string{in.ID}, rest...)
	ans := fire(t, a.path(in.Event), a.caller)
	if ans.State != final || !sameList(ans.Cases, want) {
		t.Fatalf("POST %s = %+v, want state %s cases %v", a.path(in.Event), ans, final, want)
	}
	wantRow(t, tb.Machine+"s", a.id, final)
	if hooks, ok := actors[c.Event]; ok {
		if n := hooks.rows(t, a); n != 1 {
			t.Fatalf("%s: %s wrote %d rows for %s, want 1", c.ID, c.Event, n, a.subject())
		}
	}
	return in.Source + " " + in.Event + " entered " + c.Source + "; " + c.Event + " done: cases " + strings.Join(want, " ") + ", the row is " + final, observe(t, a, c.Effects)
}

// runIgnore fires an event the state ignores: 409 with the code the reason
// promises, the row unchanged, no audit row.
func runIgnore(t *testing.T, machine, state, event, reason, id string) (string, []string) {
	m := refusalCode.FindStringSubmatch(reason)
	if m == nil {
		t.Fatalf("%s: the reason %q names no API code", id, reason)
	}
	a := arrange(t, machine, state, arrangement{byOwner, 1000})
	code, ans := post(t, a.path(event), a.caller)
	if code != http.StatusConflict || ans.Code != m[1] || ans.Case != id || ans.Message == "" {
		t.Fatalf("POST %s = %d %+v, want 409 %s case %s", a.path(event), code, ans, m[1], id)
	}
	wantRow(t, machine+"s", a.id, state)
	if n := auditCount(t, a.subject()); n != 0 {
		t.Fatalf("%s: %d audit rows after an ignored event, want 0", a.subject(), n)
	}
	return event + " on " + state + ": 409 " + m[1] + ", the row unchanged, no audit row", nil
}

// runClause makes one clause of a guard false and the others true: the
// guarded branch must not fire and the row must not change.
func runClause(t *testing.T, tb caseTable, transition, clause string) (string, []string) {
	i := slices.IndexFunc(tb.Transitions, func(o tcase) bool { return o.ID == transition })
	if i < 0 {
		t.Fatalf("clause %s: no transition %s", clause, transition)
	}
	tr := tb.Transitions[i]
	how, ok := clauseFalse[clause]
	if !ok {
		t.Fatalf("no arrangement makes clause %s false", clause)
	}
	a := arrange(t, tb.Machine, tr.Source, how)
	code, ans := post(t, a.path(tr.Event), a.caller)
	if code == http.StatusOK || ans.Case == tr.ID || slices.Contains(ans.Cases, tr.ID) {
		t.Fatalf("POST %s with %s false = %d %+v, want the branch %s not to fire", a.path(tr.Event), clause, code, ans, tr.ID)
	}
	wantRow(t, tb.Machine+"s", a.id, tr.Source)
	return "only " + clause + " false: " + tr.ID + " did not fire, the row stays " + tr.Source, nil
}
