package cli

import (
	"fmt"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/pm"
)

func init() {
	Register(&Command{Name: "plan", Summary: "Print backlog stories for owed ids; --apply creates them.", Run: runPlan})
}

func runPlan(e *Env, args []string) int {
	fs := e.Flags("plan")
	asJSON := fs.Bool("json", false, "print a bd create --graph plan")
	apply := fs.Bool("apply", false, "create the stories and edges in the backlog")
	parent := fs.String("parent", "", "parent of the created stories (with --apply)")
	link := fs.String("link", "", "attach the ids given as arguments to this existing story")
	pos, ok := parseMixed(e, fs, args)
	if !ok {
		return 2
	}
	switch {
	case *link != "" && (*apply || *asJSON || *parent != ""):
		return e.Usage("machinery plan --link takes no other flag")
	case *link != "" && len(pos) == 0:
		return e.Usage("machinery plan --link <story> needs at least one id")
	case *link == "" && len(pos) > 0:
		return e.Usage("machinery plan takes no arguments")
	case *parent != "" && !*apply:
		return e.Usage("--parent needs --apply")
	case *apply && *asJSON:
		return e.Usage("--apply and --json do not go together")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	if p.Cfg.Backlog == nil {
		_, err := backlog.New(p, "")
		return e.Fail(err)
	}
	d := design.Load(p)
	if *link != "" {
		return runLink(e, d, *link, pos)
	}
	plan, err := pm.BuildPlan(d)
	if err != nil {
		return e.Fail(err)
	}
	if *apply {
		c, err := backlog.New(p, "")
		if err != nil {
			return e.Fail(err)
		}
		res, err := pm.Apply(c, d, plan, *parent)
		if err != nil {
			return e.Fail(err)
		}
		fmt.Fprintf(e.Stdout, "plan: %d created, %d existing, %d edge(s)\n", res.Created, res.Existing, res.Edges)
		return 0
	}
	if len(plan.Stories) == 0 {
		return 0
	}
	if *asJSON {
		e.Stdout.Write(pm.JSON(plan.Graph(nil, "")))
		return 0
	}
	plan.Markdown(e.Stdout)
	return 0
}

func runLink(e *Env, d *design.Design, story string, ids []string) int {
	for _, id := range ids {
		if !d.Known(id) {
			return e.Usage("%s is not a design id", id)
		}
	}
	c, err := backlog.New(d.P, "")
	if err != nil {
		return e.Fail(err)
	}
	if is, code := showStory(e, c, story); is == nil {
		return code
	}
	if err := pm.Link(c, d, story, ids); err != nil {
		return e.Fail(err)
	}
	fmt.Fprintf(e.Stdout, "plan: %d id(s) linked to %s\n", len(ids), story)
	return 0
}
