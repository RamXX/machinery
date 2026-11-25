package cli

import (
	"github.com/RamXX/machinery/internal/affected"
	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/evidence"
	"github.com/RamXX/machinery/internal/pm"
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
	story := fs.String("story", "", "judge only the ids of a backlog story")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery verify takes no arguments")
	}
	if *targeted != (*base != "") {
		return e.Usage("--affected and --base go together: machinery verify --affected --base <ref>")
	}
	if *targeted && *story != "" {
		return e.Usage("--affected and --story do not combine")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	ctx := &stages.Context{P: p}
	switch {
	case *targeted:
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
	case *story != "":
		c, err := backlog.New(p, "")
		if err != nil {
			return e.Fail(err)
		}
		is, code := showStory(e, c, *story)
		if is == nil {
			return code
		}
		d := ctx.Design()
		ids := backlog.ParseTags(is).IDs
		if len(ids) == 0 {
			return report.PrintAll(e.Stdout, "verify", []*report.Stage{report.New("verify").NotDemonstrated("story names no design ids")})
		}
		ev, err := evidence.Evaluate(p, d, *allowDirty)
		if err != nil {
			s := report.New("verify")
			s.Errorf("%v", err)
			return report.PrintAll(e.Stdout, "verify", []*report.Stage{s})
		}
		return report.PrintAll(e.Stdout, "verify", []*report.Stage{pm.StoryVerify(d, ev, ids)})
	}
	return report.PrintAll(e.Stdout, "verify", []*report.Stage{stages.Verify(ctx, *allowDirty)})
}
