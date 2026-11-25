package cli

import (
	"github.com/RamXX/machinery/internal/probe"
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/stages"
)

func init() {
	Register(&Command{Name: "probe", Summary: "Mutate the generated tables and expect the tests to notice.", Run: runProbe})
}

func runProbe(e *Env, args []string) int {
	fs := e.Flags("probe")
	var ids, prefixes StringList
	fs.Var(&ids, "ids", "comma-separated case ids (repeatable)")
	fs.Var(&prefixes, "prefix", "id prefix (repeatable)")
	max := fs.Int("max", 0, "keep the first n mutations")
	if !e.Parse(fs, args) {
		return 2
	}
	argv := fs.Args()
	if len(argv) == 0 {
		return e.Usage("machinery probe [--ids <id>,...] [--prefix <id prefix>] [--max <n>] -- <command> [args...]")
	}
	if *max < 0 {
		return e.Usage("--max must not be negative")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	ctx := &stages.Context{P: p}
	s := stages.Probe(ctx, probe.Options{IDs: splitIDs(ids), Prefixes: prefixes, Max: *max, Argv: argv,
		Stdout: e.Stderr, Stderr: e.Stderr})
	return report.PrintAll(e.Stdout, "probe", []*report.Stage{s})
}
