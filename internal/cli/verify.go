package cli

import (
	"github.com/RamXX/machinery/internal/affected"
	"github.com/RamXX/machinery/internal/evidence"
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/stages"
)

func init() {
	Register(&Command{Name: "verify", Summary: "Join the obligations to the evidence at this commit.", Run: runVerify})
}

func runVerify(e *Env, args []string) int {
	fs := e.Flags("verify")
	allowDirty := fs.Bool("allow-dirty", false, "use evidence from a dirty tree")
	targeted := fs.Bool("affected", false, "report unproven only for the ids a change touches")
	base := fs.String("base", "", "git ref for --affected")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery verify takes no arguments")
	}
	if *targeted != (*base != "") {
		return e.Usage("--affected and --base go together: machinery verify --affected --base <ref>")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	ctx := &stages.Context{P: p}
	if !*targeted {
		return report.PrintAll(e.Stdout, "verify", []*report.Stage{stages.Verify(ctx, *allowDirty)})
	}
	if !e.CheckRef(p, *base) {
		return 2
	}
	d := ctx.Design()
	ids, err := affected.Compute(p, d, *base)
	if err != nil {
		return e.Fail(err)
	}
	ev, err := evidence.Evaluate(p, d, *allowDirty)
	if err != nil {
		return e.Fail(err)
	}
	return report.PrintAll(e.Stdout, "verify", []*report.Stage{stages.VerifyStage(affected.Restrict(ev, ids))})
}
