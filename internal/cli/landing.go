package cli

import (
	"github.com/RamXX/machinery/internal/landing"
)

func init() {
	Register(&Command{Name: "landing", Summary: "The end gate: check with the ratchet, the whole suite, verify.", Run: runLanding})
}

func runLanding(e *Env, args []string) int {
	fs := e.Flags("landing")
	base := fs.String("base", "", "git ref to ratchet against")
	stack := fs.String("stack", "", "what the tests run against")
	doProbe := fs.Bool("probe", false, "also probe the proven transition cases after verify")
	probeMax := fs.Int("probe-max", 0, "with --probe: keep the first n mutations (0: all)")
	workers := fs.Int("workers", 2, "TLC worker threads for the formal re-run")
	if !e.Parse(fs, args) {
		return 2
	}
	if *base == "" {
		return e.Usage("machinery landing --base <ref> [--stack <description>] [-- <command> [args...]]")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	if !e.CheckRef(p, *base) {
		return 2
	}
	exit, err := landing.Run(p, landing.Options{Base: *base, Stack: *stack, Command: fs.Args(), Probe: *doProbe, ProbeMax: *probeMax, Workers: *workers, Stdout: e.Stdout, Stderr: e.Stderr})
	if err != nil {
		return e.Fail(err)
	}
	return exit
}
