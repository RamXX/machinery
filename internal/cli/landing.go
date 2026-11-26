package cli

import (
	"fmt"
	"strings"

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
	repair := fs.Bool("repair", false, "land at another head while the integration branch is red")
	clearRed := fs.Bool("clear-red", false, "remove the red marker after a revert (with --reason)")
	reason := fs.String("reason", "", "with --clear-red: why the marker goes")
	if !e.Parse(fs, args) {
		return 2
	}
	if *clearRed {
		return runClearRed(e, *reason, *base != "" || fs.NArg() > 0 || *repair || *stack != "" || *doProbe)
	}
	if *base == "" {
		return e.Usage("machinery landing --base <ref> [--stack <description>] [--repair] [-- <command> [args...]]")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	if !e.CheckRef(p, *base) {
		return 2
	}
	exit, err := landing.Run(p, landing.Options{Base: *base, Stack: *stack, Command: fs.Args(), Probe: *doProbe, ProbeMax: *probeMax, Workers: *workers, Repair: *repair, Stdout: e.Stdout, Stderr: e.Stderr})
	if err != nil {
		return e.Fail(err)
	}
	return exit
}

// runClearRed removes the red marker after a revert (section 12.2).
func runClearRed(e *Env, reason string, more bool) int {
	switch {
	case strings.TrimSpace(reason) == "":
		return e.Usage("--clear-red needs --reason <text>")
	case more:
		return e.Usage("machinery landing --clear-red --reason <text> takes nothing else")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	r, err := landing.ClearRed(p)
	if err != nil {
		return e.Fail(err)
	}
	if r == nil {
		fmt.Fprintln(e.Stdout, "red: none")
		return 0
	}
	fmt.Fprintf(e.Stdout, "red: cleared %s: %s\n", r.What(), strings.TrimSpace(reason))
	return 0
}
