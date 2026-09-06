package cli_test

// Transition witnesses bind current committed oracle expectations to real Fire results.

import (
	"crm/internal/testoracle"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"crm/internal/cli"
	"crm/internal/model"
)

func cmd(state cli.CmdState) *cli.CommandExecution { return &cli.CommandExecution{State: state} }

func withArgs(c *cli.CommandExecution, args []string) *cli.CommandExecution {
	c.Args = args
	return c
}
func withPhase(c *cli.CommandExecution, phase string, retries int) *cli.CommandExecution {
	c.Phase = phase
	c.Retries = retries
	return c
}
func withRetries(c *cli.CommandExecution, n int) *cli.CommandExecution { c.Retries = n; return c }
func withAuthorize(c *cli.CommandExecution, allowed bool) *cli.CommandExecution {
	commandAuthorityFixtures[c] = allowed
	c.Authorize = func(model.User, model.Verb, model.EntityType, string, string) (bool, string) {
		if allowed {
			return true, ""
		}
		return false, "not authorized"
	}
	return c
}

var (
	validArgs   = []string{"deal", "create", "--title", "X", "--amount", "100"}
	invalidArgs = []string{"this-is-not-a-command"}
)

func errE(e error) cli.CmdEvent { return cli.CmdEvent{Kind: cli.CEvInvokeError, Err: e} }
func cmdDone() cli.CmdEvent     { return cli.CmdEvent{Kind: cli.CEvInvokeDone} }
func cmdAlways() cli.CmdEvent   { return cli.CmdEvent{Kind: cli.CEvAlways} }

type cmdCase struct {
	id    string
	ce    *cli.CommandExecution
	event cli.CmdEvent
	rowID string
}

func TestCommandExecutionTransitions(t *testing.T) {
	cs := []cmdCase{
		// --- Parsing ---
		{"T-CMD-01_COMM-44671c", withArgs(cmd(cli.CParsing), validArgs), cmdAlways(), "COMM-44671c"},
		{"T-CMD-02_COMM-6f50f3", withArgs(cmd(cli.CParsing), invalidArgs), cmdAlways(), "COMM-6f50f3"},

		// --- Opening (openDatabase) ---
		{"T-CMD-03_COMM-5bc5e0", cmd(cli.COpening), cmdDone(), "COMM-5bc5e0"},
		{"T-CMD-04_COMM-ea53cf", cmd(cli.COpening), errE(model.ErrLocked), "COMM-ea53cf"},
		{"T-CMD-05_COMM-b9aee2", cmd(cli.COpening), errE(model.ErrCorrupt), "COMM-b9aee2"},
		{"T-CMD-06_COMM-8a2a55", cmd(cli.COpening), errE(model.ErrUnavailable), "COMM-8a2a55"},
		{"T-CMD-07_COMM-343b9c", cmd(cli.COpening), errE(model.ErrConflict), "COMM-343b9c"},
		{"T-CMD-08_COMM-5e3106", cmd(cli.COpening), cli.CmdEvent{Kind: cli.CEvOpenTimeout}, "COMM-5e3106"},

		// --- DBLocked ---
		{"T-CMD-09_COMM-00c530", withRetries(cmd(cli.CDBLocked), 3), cmdAlways(), "COMM-00c530"},
		{"T-CMD-10_COMM-215300", withPhase(cmd(cli.CDBLocked), "open", 0), cli.CmdEvent{Kind: cli.CEvDbRetryBackoff}, "COMM-215300"},
		{"T-CMD-11_COMM-71162c", withPhase(cmd(cli.CDBLocked), "execute", 0), cli.CmdEvent{Kind: cli.CEvDbRetryBackoff}, "COMM-71162c"},

		// --- ResolvingSession (resolveSession) ---
		{"T-CMD-12_COMM-968d17", cmd(cli.CResolvingSession), cmdDone(), "COMM-968d17"},
		{"T-CMD-13_COMM-ed4f93", cmd(cli.CResolvingSession), errE(model.ErrNoSession), "COMM-ed4f93"},
		{"T-CMD-14_COMM-cc7919", cmd(cli.CResolvingSession), errE(model.ErrExpired), "COMM-cc7919"},
		{"T-CMD-15_COMM-22d79f", cmd(cli.CResolvingSession), errE(model.ErrLocked), "COMM-22d79f"},
		{"T-CMD-16_COMM-2151b8", cmd(cli.CResolvingSession), errE(model.ErrUnavailable), "COMM-2151b8"},
		{"T-CMD-17_COMM-35a500", cmd(cli.CResolvingSession), cli.CmdEvent{Kind: cli.CEvSessionResolveTimeout}, "COMM-35a500"},

		// --- Authorizing (pure authz decision, routed through domain) ---
		{"T-CMD-18_COMM-8c204a", withAuthorize(cmd(cli.CAuthorizing), true), cmdAlways(), "COMM-8c204a"},
		{"T-CMD-19_COMM-7f1685", withAuthorize(cmd(cli.CAuthorizing), false), cmdAlways(), "COMM-7f1685"},

		// --- Executing (executeInTx) ---
		{"T-CMD-20_COMM-5d7be9", cmd(cli.CExecuting), cmdDone(), "COMM-5d7be9"},
		{"T-CMD-21_COMM-ec7aeb", cmd(cli.CExecuting), errE(model.ErrConstraint), "COMM-ec7aeb"},
		{"T-CMD-22_COMM-d6cfde", cmd(cli.CExecuting), errE(model.ErrLocked), "COMM-d6cfde"},
		{"T-CMD-23_COMM-8be203", cmd(cli.CExecuting), errE(model.ErrConflict), "COMM-8be203"},
		{"T-CMD-24_COMM-40743b", cmd(cli.CExecuting), errE(model.ErrDiskFull), "COMM-40743b"},
		{"T-CMD-25_COMM-cb11e8", cmd(cli.CExecuting), errE(model.ErrTimeout), "COMM-cb11e8"},
		{"T-CMD-26_COMM-0b53b2", cmd(cli.CExecuting), errE(errors.New("boom")), "COMM-0b53b2"},
		{"T-CMD-27_COMM-84ddf1", cmd(cli.CExecuting), cli.CmdEvent{Kind: cli.CEvQueryTimeout}, "COMM-84ddf1"},

		// --- Rendering ---
		{"T-CMD-28_COMM-121e81", cmd(cli.CRendering), cmdAlways(), "COMM-121e81"},
	}
	suite, err := testoracle.Load("../../../design/machines", "CommandExecution")
	if err != nil {
		t.Fatal(err)
	}
	expectations := make([]testoracle.Expectation, len(cs))
	registrations := make([]testoracle.Registration, len(cs))
	for i, tc := range cs {
		expected, err := suite.Bind(commandWitness(t, tc))
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
			got := tc.ce.Fire(tc.event)
			if err := expectations[i].Check(string(tc.ce.State), got.Actions); err != nil {
				t.Fatal(err)
			}
			if registrations[i].Native != t.Name() {
				t.Fatalf("oracle-native-name: got=%q want=%q", t.Name(), registrations[i].Native)
			}
			raw, err := json.Marshal(testoracle.Observation{Registration: registrations[i], Next: string(tc.ce.State), Actions: got.Actions})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("oracle-executed %s", raw)
		})
	}
}

// The five terminal states set the process exit classification via their entry
// action (record*Exit). Each is asserted on the Fire that ENTERS the
// terminal state, checking both the entry action and the exit class.
func TestCommandExecutionTerminalExits(t *testing.T) {
	type term struct {
		id     string
		ce     *cli.CommandExecution
		event  cli.CmdEvent
		want   cli.CmdState
		action string
		exit   cli.ExitClass
	}
	cases := []term{
		{"T-CMD-29", cmd(cli.CRendering), cmdAlways(), cli.CDone, "recordSuccessExit", cli.ExitSuccess},
		{"T-CMD-30", withAuthorize(cmd(cli.CAuthorizing), false), cmdAlways(), cli.CDenied, "recordDeniedExit", cli.ExitDenied},
		{"T-CMD-31", withArgs(cmd(cli.CParsing), invalidArgs), cmdAlways(), cli.CValidationFailed, "recordValidationExit", cli.ExitValidation},
		{"T-CMD-32", cmd(cli.COpening), errE(model.ErrUnavailable), cli.CDBError, "recordDBErrorExit", cli.ExitDBError},
		{"T-CMD-33", cmd(cli.COpening), errE(model.ErrCorrupt), cli.CCorrupt, "recordCorruptExit", cli.ExitCorrupt},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			got := tc.ce.Fire(tc.event)
			if tc.ce.State != tc.want {
				t.Errorf("%s: next state = %q, want %q", tc.id, tc.ce.State, tc.want)
			}
			if !got.Has(tc.action) {
				t.Errorf("%s: entering %s must fire %q, got %v", tc.id, tc.want, tc.action, got.Actions)
			}
			if tc.ce.Exit != tc.exit {
				t.Errorf("%s: exit class = %q, want %q", tc.id, tc.ce.Exit, tc.exit)
			}
		})
	}
}

// Populated by withAuthorize from its actual fixture argument, never from a row
// ID or a production guard. Terminal supplements use that same fixture helper.
var commandAuthorityFixtures = map[*cli.CommandExecution]bool{}

func commandWitness(t *testing.T, tc cmdCase) testoracle.Witness {
	t.Helper()
	d, e := tc.ce, tc.event
	all := map[string]bool{
		"phaseIsOpen": d.Phase == "open", "phaseIsExecute": d.Phase == "execute", "retriesExhausted": d.Retries >= 3,
		"isErrLocked": errors.Is(e.Err, model.ErrLocked), "isErrCorrupt": errors.Is(e.Err, model.ErrCorrupt),
		"isErrUnavailable": errors.Is(e.Err, model.ErrUnavailable), "isErrNoSession": errors.Is(e.Err, model.ErrNoSession),
		"isErrExpired": errors.Is(e.Err, model.ErrExpired), "isErrConstraint": errors.Is(e.Err, model.ErrConstraint),
		"isErrConflict": errors.Is(e.Err, model.ErrConflict), "isErrDiskFull": errors.Is(e.Err, model.ErrDiskFull), "isErrTimeout": errors.Is(e.Err, model.ErrTimeout),
	}
	facts := func(keys ...string) map[string]bool {
		out := map[string]bool{}
		for _, key := range keys {
			v, ok := all[key]
			if !ok {
				t.Fatalf("oracle-guard-key commandExecution: %s", key)
			}
			out[key] = v
		}
		return out
	}
	w := testoracle.Witness{Name: tc.id, RowID: tc.rowID, Source: string(d.State)}
	switch d.State {
	case cli.CParsing:
		if e.Kind == cli.CEvAlways {
			switch {
			case reflect.DeepEqual(d.Args, []string{"deal", "create", "--title", "X", "--amount", "100"}):
				all["guardParseOk"] = true
			case reflect.DeepEqual(d.Args, []string{"this-is-not-a-command"}):
				all["guardParseOk"] = false
			default:
				t.Fatalf("oracle-clause commandExecution: unknown actual argv fixture %q", d.Args)
			}
			w.Trigger = "always"
			w.Guards = facts("guardParseOk")
			return w
		}
	case cli.CAuthorizing:
		if e.Kind == cli.CEvAlways {
			want, known := commandAuthorityFixtures[d]
			if !known || d.Authorize == nil {
				t.Fatal("oracle-clause commandExecution: missing actual authorization fixture")
			}
			before := *d
			before.Args = append([]string(nil), d.Args...)
			before.Authorize = nil
			for i := 0; i < 2; i++ {
				allowed, reason := d.Authorize(d.Actor, d.Verb, d.EntityType, d.TargetOwnerID, d.TargetTeamID)
				wantReason := ""
				if !want {
					wantReason = "not authorized"
				}
				if allowed != want || reason != wantReason {
					t.Fatalf("oracle-clause commandExecution: authorization fixture changed result allowed=%t reason=%q", allowed, reason)
				}
			}
			after := *d
			after.Authorize = nil
			if !reflect.DeepEqual(before, after) {
				t.Fatal("oracle-clause commandExecution: authorization fixture mutated command")
			}
			all["guardAuthorized"] = want
			w.Trigger = "always"
			w.Guards = facts("guardAuthorized")
			return w
		}
	case cli.CRendering:
		if e.Kind == cli.CEvAlways {
			w.Trigger = "always"
			return w
		}
	case cli.CDBLocked:
		switch e.Kind {
		case cli.CEvAlways:
			w.Trigger = "always"
			w.Guards = facts("retriesExhausted")
			return w
		case cli.CEvDbRetryBackoff:
			w.Trigger = "after:dbRetryBackoff"
			w.Guards = facts("phaseIsOpen", "phaseIsExecute")
			return w
		}
	case cli.COpening, cli.CResolvingSession, cli.CExecuting:
		invoke, delay := "", ""
		var errorKeys []string
		switch d.State {
		case cli.COpening:
			invoke = "openDatabase"
			delay = "openTimeout"
			errorKeys = []string{"isErrLocked", "isErrCorrupt", "isErrUnavailable"}
		case cli.CResolvingSession:
			invoke = "resolveSession"
			delay = "sessionResolveTimeout"
			errorKeys = []string{"isErrNoSession", "isErrExpired", "isErrLocked"}
		case cli.CExecuting:
			invoke = "executeInTx"
			delay = "queryTimeout"
			errorKeys = []string{"isErrConstraint", "isErrLocked", "isErrConflict", "isErrDiskFull", "isErrTimeout"}
		}
		switch e.Kind {
		case cli.CEvInvokeDone:
			w.Trigger = "onDone:" + invoke
			return w
		case cli.CEvInvokeError:
			w.Trigger = "onError:" + invoke
			w.Guards = facts(errorKeys...)
			return w
		case cli.CEvOpenTimeout, cli.CEvSessionResolveTimeout, cli.CEvQueryTimeout:
			if string(e.Kind) == delay {
				w.Trigger = "after:" + delay
				return w
			}
		}
	}
	t.Fatalf("oracle-event commandExecution: unsupported actual source=%q event=%q", d.State, e.Kind)
	return w
}
