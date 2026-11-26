package cli

import (
	"fmt"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/pm"
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/stages"
)

func init() {
	Register(&Command{Name: "waves", Summary: "Print the parallel waves of the backlog graph; --freeze, --thaw.", Run: runWaves})
}

func runWaves(e *Env, args []string) int {
	fs := e.Flags("waves")
	asJSON := fs.Bool("json", false, "print the waves as one JSON document")
	freeze := fs.Bool("freeze", false, "freeze the contract for the first wave")
	thaw := fs.Bool("thaw", false, "remove the frozen wave")
	if !e.Parse(fs, args) {
		return 2
	}
	switch {
	case fs.NArg() > 0:
		return e.Usage("machinery waves takes no arguments")
	case *thaw && (*freeze || *asJSON):
		return e.Usage("--thaw takes no other flag")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	c, err := backlog.New(p, "")
	if err != nil {
		return e.Fail(err)
	}
	if *thaw {
		removed, err := pm.Thaw(p)
		if err != nil {
			return e.Fail(fmt.Errorf("cannot thaw: %v", err))
		}
		if removed {
			fmt.Fprintln(e.Stdout, "wave: thawed")
		} else {
			fmt.Fprintln(e.Stdout, "wave: none frozen")
		}
		return 0
	}
	if *freeze {
		if _, err := pm.WavePath(p); err != nil {
			return e.Fail(fmt.Errorf("cannot freeze: %v", err))
		}
	}
	issues, err := c.List()
	if err != nil {
		return e.Fail(err)
	}
	ctx := &stages.Context{P: p}
	d := ctx.Design()
	w := pm.BuildWaves(d, issues)
	if !*freeze {
		if *asJSON {
			return w.JSON(e.Stdout)
		}
		return w.Print(e.Stdout)
	}
	if p.Cfg.ReviewOn(pm.ReviewBacklog) {
		who, err := backlog.WhoFromEnv()
		if err != nil {
			return e.Usage("%v", err)
		}
		pm.HoldBacklogReview(w.Stage, issues, w.Contract, w.Wave1(), pm.Who{Actor: c.Actor, Session: who.Session, Agent: who.Agent})
	}
	return freezeWave(e, ctx, d, w, *asJSON)
}

// freezeWave writes the wave file after a clean gen-drift stage and a
// passing waves stage (section 15.4).
func freezeWave(e *Env, ctx *stages.Context, d *design.Design, w *pm.Waves, asJSON bool) int {
	p := ctx.P
	head, err := gitx.Head(p.Root)
	if err != nil {
		return e.Fail(fmt.Errorf("cannot freeze: no commit: %v", err))
	}
	drift := stages.RunOne(ctx, "gen-drift")
	if drift.Outcome() != report.Passed {
		report.Print(e.Stdout, drift)
		fmt.Fprintln(e.Stderr, "machinery: cannot freeze: stage gen-drift does not pass; run machinery gen and commit")
		return 1
	}
	var code int
	if asJSON {
		code = w.JSON(e.Stdout)
	} else {
		code = w.Print(e.Stdout)
	}
	if code != 0 {
		fmt.Fprintln(e.Stderr, "machinery: cannot freeze: stage waves does not pass")
		return code
	}
	wave := &pm.Wave{Version: 1, Contract: w.Contract.Hash, Commit: head, Files: w.Contract.Files, Stories: w.Wave1()}
	if err := pm.WriteWave(p, wave); err != nil {
		return e.Fail(err)
	}
	if !asJSON {
		fmt.Fprintf(e.Stdout, "wave: frozen %s at %s, %d stories\n", w.Contract.Short(), gitx.Short(head), len(wave.Stories))
	}
	return 0
}
