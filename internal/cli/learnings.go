package cli

import (
	"fmt"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/pm"
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
		var known bool
		var err error
		if sha, known, err = gitx.Commit(p.Root, *since); err != nil {
			return e.Fail(err)
		}
		if !known {
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
	groups, stories, err := pm.Learnings(p.Root, issues, sha)
	if err != nil {
		return e.Fail(err)
	}
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
