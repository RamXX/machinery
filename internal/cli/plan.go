package cli

import (
	"fmt"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/pm"
)

func init() {
	Register(&Command{Name: "plan", Summary: "Print backlog stories for owed ids; --summary prints counts; --apply creates stories.", Run: runPlan})
}

func runPlan(e *Env, args []string) int {
	fs := e.Flags("plan")
	asJSON := fs.Bool("json", false, "print a bd create --graph plan")
	summary := fs.Bool("summary", false, "print only story counts")
	apply := fs.Bool("apply", false, "create the stories and edges in the backlog")
	parent := fs.String("parent", "", "parent of the created stories (with --apply)")
	link := fs.String("link", "", "attach the ids given as arguments to this existing story")
	pos, ok := parseMixed(e, fs, args)
	if !ok {
		return 2
	}
	switch {
	case *link != "" && (*apply || *asJSON || *summary || *parent != ""):
		return e.Usage("machinery plan --link takes no other flag")
	case *link != "" && len(pos) == 0:
		return e.Usage("machinery plan --link <story> needs at least one id")
	case *link == "" && len(pos) > 0:
		return e.Usage("machinery plan takes no arguments")
	case *parent != "" && !*apply:
		return e.Usage("--parent needs --apply")
	case *apply && *asJSON:
		return e.Usage("--apply and --json do not go together")
	case *summary && *apply:
		return e.Usage("--summary and --apply do not go together")
	case *summary && *asJSON:
		return e.Usage("--summary and --json do not go together")
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
		plan.Summary(e.Stdout)
		for _, x := range res.External {
			fmt.Fprintf(e.Stdout, "plan: %s: external story %s, not in the backlog; nothing created\n", x.ID, x.Story)
		}
		return 0
	}
	if len(plan.Stories) == 0 {
		return 0
	}
	if *asJSON {
		e.Stdout.Write(pm.JSON(plan.Graph(nil, "")))
		return 0
	}
	if !*summary {
		plan.Markdown(e.Stdout)
		fmt.Fprintln(e.Stdout)
	}
	plan.Summary(e.Stdout)
	return 0
}

func runLink(e *Env, d *design.Design, story string, ids []string) int {
	for _, id := range ids {
		if !d.Known(id) {
			return e.Usage("%s is not a design id", id)
		}
	}
	if x, ok := pm.Supplied(d, ids); ok && x.Story != story {
		return e.Usage("%s carries the story %s the project planned it under; plan never replaces it: edit %s to change it", x.ID, x.Story, d.P.Cfg.Design.Owed)
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
