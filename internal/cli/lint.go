package cli

import (
	"github.com/RamXX/machinery/internal/stages"
)

func init() {
	Register(&Command{Name: "lint", Summary: "Lint the design sources and cross-check them.", Run: func(e *Env, args []string) int {
		return runStageCommand(e, "lint", args, stages.LintStages, false)
	}})
	Register(&Command{Name: "check", Summary: "Every static stage; with --base also the ratchet.", Run: func(e *Env, args []string) int {
		return runStageCommand(e, "check", args, stages.CheckStages, true)
	}})
}

func runStageCommand(e *Env, name string, args []string, all []string, withBase bool) int {
	fs := e.Flags(name)
	var want StringList
	fs.Var(&want, "stage", "run only this stage (repeatable)")
	base := ""
	if withBase {
		fs.StringVar(&base, "base", "", "git ref to ratchet against")
	}
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery %s takes no arguments", name)
	}
	list := all
	if base != "" {
		list = append(append([]string{}, all...), "ratchet")
	}
	sel, unknown := stages.Select(list, want)
	if unknown != "" {
		return e.Usage("unknown stage %q for machinery %s", unknown, name)
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	return e.RunStages(name, &stages.Context{P: p, Base: base}, sel)
}

// singleStage registers a command that runs exactly one stage.
func singleStage(name, stage, summary string) {
	Register(&Command{Name: name, Summary: summary, Run: func(e *Env, args []string) int {
		fs := e.Flags(name)
		if !e.Parse(fs, args) {
			return 2
		}
		p, ok := e.Project()
		if !ok {
			return 2
		}
		return e.RunStages(name, &stages.Context{P: p}, []string{stage})
	}})
}
