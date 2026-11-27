package cli

import (
	"github.com/RamXX/machinery/v2/internal/backlog"
	"github.com/RamXX/machinery/v2/internal/stages"
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
	base, story := "", ""
	skip := false
	if withBase {
		fs.StringVar(&base, "base", "", "git ref to ratchet against")
		fs.StringVar(&story, "story", "", "with --base: hold this story to its scope (stage scope)")
		fs.BoolVar(&skip, "no-backlog", false, "do not run the backlog tool; stage intake says "+notRead)
	}
	if !e.Parse(fs, args) {
		return 2
	}
	skip = withBase && noBacklog(skip)
	if story != "" && skip {
		return e.Usage("%s", storyNeedsBacklog)
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery %s takes no arguments", name)
	}
	if story != "" && base == "" {
		return e.Usage("--story needs --base <ref>")
	}
	list := all
	if base != "" {
		list = append(append([]string{}, all...), "ratchet")
	}
	if withBase && (story != "" || wants(want, "scope")) {
		list = append(append([]string{}, list...), "scope")
	}
	sel, unknown := stages.Select(list, want)
	if unknown != "" {
		return e.Usage("unknown stage %q for machinery %s", unknown, name)
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	if base != "" && !e.CheckRef(p, base) {
		return 2
	}
	if story != "" {
		c, err := backlog.New(p, "")
		if err != nil {
			return e.Fail(err)
		}
		if is, code := showStory(e, c, story); is == nil {
			return code
		}
	}
	ctx := &stages.Context{P: p, Base: base, Story: story}
	if skip {
		ctx.NoBacklog = notRead
	}
	return e.RunStages(name, ctx, sel)
}

func wants(want []string, stage string) bool {
	for _, w := range want {
		if w == stage {
			return true
		}
	}
	return false
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
