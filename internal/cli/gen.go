package cli

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/RamXX/machinery/internal/gen"
	"github.com/RamXX/machinery/internal/modelith"
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/stages"
)

func init() {
	Register(&Command{Name: "gen", Summary: "Write the generated files; --check reports drift.", Run: runGen})
}

func runGen(e *Env, args []string) int {
	fs := e.Flags("gen")
	check := fs.Bool("check", false, "report drift and write nothing")
	if !e.Parse(fs, args) {
		return 2
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	ctx := &stages.Context{P: p}
	if *check {
		return e.RunStages("gen", ctx, []string{"gen-drift"})
	}
	written, blocked, err := Generate(ctx)
	if err != nil {
		return e.Fail(err)
	}
	if len(blocked) > 0 {
		for _, s := range blocked {
			report.Print(e.Stdout, s)
		}
		fmt.Fprintln(e.Stdout, "gen: refused: the design has errors; nothing was written")
		return 1
	}
	fmt.Fprintf(e.Stdout, "gen: wrote %d file(s)\n", len(written))
	for _, w := range written {
		fmt.Fprintf(e.Stdout, "  %s\n", w)
	}
	return 0
}

// Generate writes every generated file. It refuses, returning the blocking
// stages, when the model, machines or failure-modes stage has an ERROR or the
// model stage cannot run.
func Generate(ctx *stages.Context) (written []string, blocked []*report.Stage, err error) {
	for _, name := range []string{"model", "machines", "failure-modes"} {
		s := stages.RunOne(ctx, name)
		errs, _, _ := s.Count3()
		if errs > 0 || (name == "model" && s.Outcome() == report.NotDemonstrated) {
			blocked = append(blocked, s)
		}
	}
	if len(blocked) > 0 {
		return nil, blocked, nil
	}
	p := ctx.P
	for _, rel := range p.Cfg.Design.Models {
		want, rerr := modelith.Render(p, rel)
		if rerr != nil {
			return written, nil, rerr
		}
		md := modelith.RenderedPath(rel)
		if got, gerr := p.Read(md); gerr == nil && bytes.Equal(got, want) {
			continue
		}
		if werr := p.WriteAtomic(md, want); werr != nil {
			return written, nil, werr
		}
		written = append(written, md)
	}
	w, werr := gen.Write(ctx.Design())
	written = append(written, w...)
	sort.Strings(written)
	ctx.Reload()
	return written, nil, werr
}
