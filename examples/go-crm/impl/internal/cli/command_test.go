package cli_test

// CommandExecution machine transition oracle. One case per BUILD.md 7.1 T-CMD
// row. Source: machines/CommandExecution.matrix.md.
//
// The CommandExecution machine has entry actions (Opening: setPhaseOpen,
// Executing: setPhaseExecute, Rendering: renderOutput, and the five terminal
// states: record*Exit). Standard XState semantics fire a state's entry action
// when it is ENTERED, so a Fire that transitions into such a state produces the
// transition action(s) FOLLOWED BY the entry action(s). Tests therefore assert
// action CONTAINMENT (firedInOrder): the row's listed actions must fire in order,
// which tolerates the entry action the matrix omits (for example row 1 lists
// captureArgs; entering Opening also fires setPhaseOpen). The five terminal-entry
// rows T-CMD-29..33 are asserted on the Fire that ENTERS the terminal state (that
// is when record*Exit fires), and also check the exit classification.

import (
	"errors"
	"testing"

	"crm/internal/cli"
	"crm/internal/model"
)

func firedInOrder(got, want []string) bool {
	i := 0
	for _, g := range got {
		if i < len(want) && g == want[i] {
			i++
		}
	}
	return i == len(want)
}

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
	id      string
	ce      *cli.CommandExecution
	event   cli.CmdEvent
	want    cli.CmdState
	actions []string
}

func cmdCases() []cmdCase {
	return []cmdCase{
		// --- Parsing ---
		{"T-CMD-01_COMM-44671c", withArgs(cmd(cli.CParsing), validArgs), cmdAlways(), cli.COpening, []string{"captureArgs"}},
		{"T-CMD-02_COMM-6f50f3", withArgs(cmd(cli.CParsing), invalidArgs), cmdAlways(), cli.CValidationFailed, []string{"recordParseError"}},

		// --- Opening (openDatabase) ---
		{"T-CMD-03_COMM-5bc5e0", cmd(cli.COpening), cmdDone(), cli.CResolvingSession, []string{"captureTx"}},
		{"T-CMD-04_COMM-ea53cf", cmd(cli.COpening), errE(model.ErrLocked), cli.CDBLocked, []string{"recordError"}},
		{"T-CMD-05_COMM-b9aee2", cmd(cli.COpening), errE(model.ErrCorrupt), cli.CCorrupt, []string{"recordCorrupt"}},
		{"T-CMD-06_COMM-8a2a55", cmd(cli.COpening), errE(model.ErrUnavailable), cli.CDBError, []string{"recordUnavailable"}},
		{"T-CMD-07_COMM-343b9c", cmd(cli.COpening), errE(model.ErrConflict), cli.CDBError, []string{"recordOpenError"}},
		{"T-CMD-08_COMM-5e3106", cmd(cli.COpening), cli.CmdEvent{Kind: cli.CEvOpenTimeout}, cli.CDBError, []string{"recordTimeout"}},

		// --- DBLocked ---
		{"T-CMD-09_COMM-00c530", withRetries(cmd(cli.CDBLocked), 3), cmdAlways(), cli.CDBError, []string{"recordLockExhausted"}},
		{"T-CMD-10_COMM-215300", withPhase(cmd(cli.CDBLocked), "open", 0), cli.CmdEvent{Kind: cli.CEvDbRetryBackoff}, cli.COpening, []string{"incrementRetries"}},
		{"T-CMD-11_COMM-71162c", withPhase(cmd(cli.CDBLocked), "execute", 0), cli.CmdEvent{Kind: cli.CEvDbRetryBackoff}, cli.CExecuting, []string{"incrementRetries"}},

		// --- ResolvingSession (resolveSession) ---
		{"T-CMD-12_COMM-968d17", cmd(cli.CResolvingSession), cmdDone(), cli.CAuthorizing, []string{"captureActor"}},
		{"T-CMD-13_COMM-ed4f93", cmd(cli.CResolvingSession), errE(model.ErrNoSession), cli.CDenied, []string{"recordNeedLogin"}},
		{"T-CMD-14_COMM-cc7919", cmd(cli.CResolvingSession), errE(model.ErrExpired), cli.CDenied, []string{"recordNeedLogin"}},
		{"T-CMD-15_COMM-22d79f", cmd(cli.CResolvingSession), errE(model.ErrLocked), cli.CDBLocked, []string{"recordError"}},
		{"T-CMD-16_COMM-2151b8", cmd(cli.CResolvingSession), errE(model.ErrUnavailable), cli.CDBError, []string{"recordSessionError"}},
		{"T-CMD-17_COMM-35a500", cmd(cli.CResolvingSession), cli.CmdEvent{Kind: cli.CEvSessionResolveTimeout}, cli.CDBError, []string{"recordTimeout"}},

		// --- Authorizing (pure authz decision, routed through domain) ---
		{"T-CMD-18_COMM-8c204a", withAuthorize(cmd(cli.CAuthorizing), true), cmdAlways(), cli.CExecuting, []string{"recordAllowed"}},
		{"T-CMD-19_COMM-7f1685", withAuthorize(cmd(cli.CAuthorizing), false), cmdAlways(), cli.CDenied, []string{"recordDenyReason"}},

		// --- Executing (executeInTx) ---
		{"T-CMD-20_COMM-5d7be9", cmd(cli.CExecuting), cmdDone(), cli.CRendering, []string{"captureResult"}},
		{"T-CMD-21_COMM-ec7aeb", cmd(cli.CExecuting), errE(model.ErrConstraint), cli.CValidationFailed, []string{"ensureRolledBack", "recordConstraint"}},
		{"T-CMD-22_COMM-d6cfde", cmd(cli.CExecuting), errE(model.ErrLocked), cli.CDBLocked, []string{"ensureRolledBack", "recordError"}},
		{"T-CMD-23_COMM-8be203", cmd(cli.CExecuting), errE(model.ErrConflict), cli.CDBLocked, []string{"ensureRolledBack", "recordConflict"}},
		{"T-CMD-24_COMM-40743b", cmd(cli.CExecuting), errE(model.ErrDiskFull), cli.CDBError, []string{"ensureRolledBack", "recordDiskFull"}},
		{"T-CMD-25_COMM-cb11e8", cmd(cli.CExecuting), errE(model.ErrTimeout), cli.CDBError, []string{"ensureRolledBack", "recordTimeout"}},
		{"T-CMD-26_COMM-0b53b2", cmd(cli.CExecuting), errE(errors.New("boom")), cli.CDBError, []string{"ensureRolledBack", "recordExecuteError"}},
		{"T-CMD-27_COMM-84ddf1", cmd(cli.CExecuting), cli.CmdEvent{Kind: cli.CEvQueryTimeout}, cli.CDBError, []string{"ensureRolledBack", "recordTimeout"}},

		// --- Rendering ---
		{"T-CMD-28_COMM-121e81", cmd(cli.CRendering), cmdAlways(), cli.CDone, nil},
	}
}

func TestCommandExecutionTransitions(t *testing.T) {
	for _, tc := range cmdCases() {
		t.Run(tc.id, func(t *testing.T) {
			got := tc.ce.Fire(tc.event)
			if tc.ce.State != tc.want {
				t.Errorf("%s: next state = %q, want %q", tc.id, tc.ce.State, tc.want)
			}
			if !firedInOrder(got.Actions, tc.actions) {
				t.Errorf("%s: actions = %v, want (in order) %v", tc.id, got.Actions, tc.actions)
			}
		})
	}
}

// T-CMD-29..33: the five terminal states set the process exit classification via
// their entry action (record*Exit). Each is asserted on the Fire that ENTERS the
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
