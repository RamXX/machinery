package cli

import (
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/stages"
)

func init() {
	Register(&Command{Name: "verify", Summary: "Join the obligations to the evidence at this commit.", Run: runVerify})
}

func runVerify(e *Env, args []string) int {
	fs := e.Flags("verify")
	allowDirty := fs.Bool("allow-dirty", false, "use evidence from a dirty tree")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery verify takes no arguments")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	ctx := &stages.Context{P: p}
	return report.PrintAll(e.Stdout, "verify", []*report.Stage{stages.Verify(ctx, *allowDirty)})
}
