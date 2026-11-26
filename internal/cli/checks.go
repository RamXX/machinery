package cli

import (
	"fmt"

	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/stages"
)

func init() {
	Register(&Command{Name: "checks", Summary: "Print the declaration of applicable checks.", Run: runChecks})
}

// runChecks renders section 3.4: one line per stage, whether it applies, and
// the behavior it protects. Stages that need a run or a story are listed with
// how they are invoked.
func runChecks(e *Env, args []string) int {
	fs := e.Flags("checks")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery checks takes no arguments")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	ctx := &stages.Context{P: p}
	static := map[string]bool{}
	for _, n := range stages.CheckStages {
		static[n] = true
	}
	onDemand := map[string]string{
		"ratchet": "on demand: machinery check --base <ref>",
		"scope":   "on demand: machinery check --base <ref> --story <id>",
		"verify":  "on demand: machinery verify",
		"probe":   "on demand: machinery probe -- <command>",
		"story":   "on demand: machinery story <id>",
	}
	for _, def := range stages.Defs {
		state := "applicable"
		if static[def.Name] {
			if s := stages.RunOne(ctx, def.Name); s.Outcome() == report.NotApplicable {
				state = "not applicable: " + s.NA
			}
		} else {
			state = onDemand[def.Name]
		}
		fmt.Fprintf(e.Stdout, "%-18s  %s\n", def.Name, state)
		fmt.Fprintf(e.Stdout, "%-18s  protects: %s\n", "", def.Protects)
	}
	return 0
}
