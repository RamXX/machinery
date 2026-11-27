package cli

import (
	"fmt"

	"github.com/RamXX/machinery/v2/internal/backlog"
	"github.com/RamXX/machinery/v2/internal/gitx"
	"github.com/RamXX/machinery/v2/internal/pm"
)

func init() {
	Register(&Command{Name: "learnings", Summary: "Print the learnings of closed stories, by intake.", Run: runLearnings})
}

func runLearnings(e *Env, args []string) int {
	fs := e.Flags("learnings")
	since := fs.String("since", "", "only the stories with a proof after this git ref")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery learnings takes no arguments")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	sha := ""
	if *since != "" {
		var err error
		if sha, err = gitx.Resolve(p.Root, *since); err != nil {
			return e.Usage("--since %s: not a commit of the project", *since)
		}
	}
	c, err := backlog.New(p, "")
	if err != nil {
		return e.Fail(err)
	}
	issues, err := c.List()
	if err != nil {
		return e.Fail(err)
	}
	groups, stories := pm.Learnings(p.Root, issues, sha)
	for _, g := range groups {
		switch {
		case g.Intake == "":
			fmt.Fprintln(e.Stdout, "outside an intake")
		case g.Title == "":
			fmt.Fprintf(e.Stdout, "intake %s\n", g.Intake)
		default:
			fmt.Fprintf(e.Stdout, "intake %s: %s\n", g.Intake, g.Title)
		}
		for _, l := range g.Lines {
			fmt.Fprintf(e.Stdout, "  %s  %s\n", l[0], l[1])
		}
	}
	line := fmt.Sprintf("learnings: %d from %d closed stories", pm.CountLearnings(groups), stories)
	if sha != "" {
		line += " since " + gitx.Short(sha)
	}
	fmt.Fprintln(e.Stdout, line)
	return 0
}
