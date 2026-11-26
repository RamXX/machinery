// Package stages holds the check stages (docs/formats.md sections 3.1 and 3.4).
// Each stage lives in its own file and binds itself in an init function.
package stages

import (
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
)

// Context is what a stage gets.
type Context struct {
	P    *project.Project
	Base string // --base git ref, "" when absent
	// Story is the story stage scope holds to its scope (--story), "" when absent.
	Story string
	// NoBacklog, when set, is why no stage may run the backlog tool.
	NoBacklog string

	d *design.Design
}

// Design loads the design once per command.
func (c *Context) Design() *design.Design {
	if c.d == nil {
		c.d = design.Load(c.P)
	}
	return c.d
}

// Reload drops the cached design (after a command wrote design files).
func (c *Context) Reload() { c.d = nil }

// Def declares a stage and the product behavior it protects.
type Def struct {
	Name     string
	Protects string
}

// Defs lists every stage in its usual order (section 3.4).
var Defs = []Def{
	{"model", "The vocabulary every other artifact uses: entities, actions, invariants, scenarios."},
	{"machines", "Every state answers every event: a transition, a refusal or a reasoned ignore."},
	{"failure-modes", "Each declared failure and abuse case is well formed and has a required behavior."},
	{"risk", "Each risk area and status entry names real design ids, so tier gates and statuses attach to real behavior."},
	{"xref-model-machine", "The lifecycle a machine implements is the lifecycle the model promises."},
	{"xref-contract", "Every action is reachable through a published operation."},
	{"xref-failure", "Every side effect, dependency, failure state, failure event and tiered action has a declared failure behavior."},
	{"xstate", "A generated table means what XState would execute for the same machine."},
	{"gen-drift", "Tables and obligations in the tree are the ones the design produces."},
	{"formal", "The formal model of a critical area is current, not vacuous and not weakened."},
	{"checkers", "Verdicts of external checkers are for this exact design."},
	{"arch", "The layering the design declares holds in the code."},
	{"tests", "Evidence comes from the real system: no unit tests, no test doubles."},
	{"trace", "Every action is served by code and every obligation is claimed by a test or owed."},
	{"contract", "Stories running in parallel build against the same design."},
	{"intake", "A requirement reaches the backlog only through the design it changed, and is closed once delivered."},
	{"ratchet", "What was proven, covered or modeled stays so."},
	{"scope", "A story writes only inside its declared scope, and leaves the design to an intake."},
	{"verify", "Each obligation is observed on the running system at this commit."},
	{"probe", "The tests would notice a wrong transition or a missing effect."},
	{"story", "A story is accepted with the formality its risk tier requires."},
}

// LintStages are the stages of machinery lint, in order.
var LintStages = []string{"model", "machines", "failure-modes", "risk", "xref-model-machine", "xref-contract", "xref-failure", "xstate"}

// CheckStages are the stages of machinery check without --base, in order.
var CheckStages = append(append([]string{}, LintStages...), "gen-drift", "formal", "checkers", "arch", "tests", "trace", "contract", "intake")

var bound = map[string]func(*Context) *report.Stage{}

// Bind attaches the implementation of a stage. Called from init functions.
func Bind(name string, fn func(*Context) *report.Stage) {
	if _, dup := bound[name]; dup {
		panic("stage bound twice: " + name)
	}
	known := false
	for _, d := range Defs {
		if d.Name == name {
			known = true
		}
	}
	if !known {
		panic("unknown stage: " + name)
	}
	bound[name] = fn
}

// Known reports whether a stage name exists.
func Known(name string) bool {
	for _, d := range Defs {
		if d.Name == name {
			return true
		}
	}
	return false
}

// RunOne runs one stage.
func RunOne(ctx *Context, name string) *report.Stage {
	fn, ok := bound[name]
	if !ok {
		return report.New(name).NotDemonstrated("this stage is not built")
	}
	return fn(ctx)
}

// Run runs the named stages in the given order.
func Run(ctx *Context, names []string) []*report.Stage {
	var res []*report.Stage
	for _, n := range names {
		res = append(res, RunOne(ctx, n))
	}
	return res
}

// Select filters a stage list by the --stage flag values, keeping the usual
// order. unknown is the first name that is not in the list.
func Select(all, want []string) (sel []string, unknown string) {
	if len(want) == 0 {
		return all, ""
	}
	set := map[string]bool{}
	for _, w := range want {
		found := false
		for _, a := range all {
			if a == w {
				found = true
			}
		}
		if !found {
			return nil, w
		}
		set[w] = true
	}
	for _, a := range all {
		if set[a] {
			sel = append(sel, a)
		}
	}
	return sel, ""
}
