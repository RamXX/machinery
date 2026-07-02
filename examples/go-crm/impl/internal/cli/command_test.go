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
		{"T-CMD-01", withArgs(cmd(cli.CParsing), validArgs), cmdAlways(), cli.COpening, []string{"captureArgs"}},
		{"T-CMD-02", withArgs(cmd(cli.CParsing), invalidArgs), cmdAlways(), cli.CValidationFailed, []string{"recordParseError"}},

		// --- Opening (openDatabase) ---
		{"T-CMD-03", cmd(cli.COpening), cmdDone(), cli.CResolvingSession, []string{"captureTx"}},
		{"T-CMD-04", cmd(cli.COpening), errE(model.ErrLocked), cli.CDBLocked, []string{"recordError"}},
		{"T-CMD-05", cmd(cli.COpening), errE(model.ErrCorrupt), cli.CCorrupt, []string{"recordCorrupt"}},
		{"T-CMD-06", cmd(cli.COpening), errE(model.ErrUnavailable), cli.CDBError, []string{"recordUnavailable"}},
		{"T-CMD-07", cmd(cli.COpening), errE(model.ErrConflict), cli.CDBError, []string{"recordOpenError"}},
		{"T-CMD-08", cmd(cli.COpening), cli.CmdEvent{Kind: cli.CEvOpenTimeout}, cli.CDBError, []string{"recordTimeout"}},

		// --- DBLocked ---
		{"T-CMD-09", withRetries(cmd(cli.CDBLocked), 3), cmdAlways(), cli.CDBError, []string{"recordLockExhausted"}},
		{"T-CMD-10", withPhase(cmd(cli.CDBLocked), "open", 0), cli.CmdEvent{Kind: cli.CEvDbRetryBackoff}, cli.COpening, []string{"incrementRetries"}},
		{"T-CMD-11", withPhase(cmd(cli.CDBLocked), "execute", 0), cli.CmdEvent{Kind: cli.CEvDbRetryBackoff}, cli.CExecuting, []string{"incrementRetries"}},

		// --- ResolvingSession (resolveSession) ---
		{"T-CMD-12", cmd(cli.CResolvingSession), cmdDone(), cli.CAuthorizing, []string{"captureActor"}},
		{"T-CMD-13", cmd(cli.CResolvingSession), errE(model.ErrNoSession), cli.CDenied, []string{"recordNeedLogin"}},
		{"T-CMD-14", cmd(cli.CResolvingSession), errE(model.ErrExpired), cli.CDenied, []string{"recordNeedLogin"}},
		{"T-CMD-15", cmd(cli.CResolvingSession), errE(model.ErrLocked), cli.CDBLocked, []string{"recordError"}},
		{"T-CMD-16", cmd(cli.CResolvingSession), errE(model.ErrUnavailable), cli.CDBError, []string{"recordSessionError"}},
		{"T-CMD-17", cmd(cli.CResolvingSession), cli.CmdEvent{Kind: cli.CEvSessionResolveTimeout}, cli.CDBError, []string{"recordTimeout"}},

		// --- Authorizing (pure authz decision, routed through domain) ---
		{"T-CMD-18", withAuthorize(cmd(cli.CAuthorizing), true), cmdAlways(), cli.CExecuting, []string{"recordAllowed"}},
		{"T-CMD-19", withAuthorize(cmd(cli.CAuthorizing), false), cmdAlways(), cli.CDenied, []string{"recordDenyReason"}},

		// --- Executing (executeInTx) ---
		{"T-CMD-20", cmd(cli.CExecuting), cmdDone(), cli.CRendering, []string{"captureResult"}},
		{"T-CMD-21", cmd(cli.CExecuting), errE(model.ErrConstraint), cli.CValidationFailed, []string{"ensureRolledBack", "recordConstraint"}},
		{"T-CMD-22", cmd(cli.CExecuting), errE(model.ErrLocked), cli.CDBLocked, []string{"ensureRolledBack", "recordError"}},
		{"T-CMD-23", cmd(cli.CExecuting), errE(model.ErrConflict), cli.CDBLocked, []string{"ensureRolledBack", "recordConflict"}},
		{"T-CMD-24", cmd(cli.CExecuting), errE(model.ErrDiskFull), cli.CDBError, []string{"ensureRolledBack", "recordDiskFull"}},
		{"T-CMD-25", cmd(cli.CExecuting), errE(model.ErrTimeout), cli.CDBError, []string{"ensureRolledBack", "recordTimeout"}},
		{"T-CMD-26", cmd(cli.CExecuting), errE(errors.New("boom")), cli.CDBError, []string{"ensureRolledBack", "recordExecuteError"}},
		{"T-CMD-27", cmd(cli.CExecuting), cli.CmdEvent{Kind: cli.CEvQueryTimeout}, cli.CDBError, []string{"ensureRolledBack", "recordTimeout"}},

		// --- Rendering ---
		{"T-CMD-28", cmd(cli.CRendering), cmdAlways(), cli.CDone, nil},
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
