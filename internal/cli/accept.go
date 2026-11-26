package cli

import (
	"fmt"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/pm"
	"github.com/RamXX/machinery/internal/report"
)

func init() {
	Register(&Command{Name: "accept", Summary: "Record a verifier's verdict and run the story gate.", Run: runAccept})
}

func runAccept(e *Env, args []string) int {
	fs := e.Flags("accept")
	verdict := fs.String("verdict", "", "accept or reject")
	observed := fs.String("observed", "", "what was seen")
	closeIt := fs.Bool("close", false, "close the story when the gate passes")
	actor := fs.String("actor", "", "who decides (default BEADS_ACTOR, else git user.name)")
	pos, ok := parseMixed(e, fs, args)
	if !ok {
		return 2
	}
	switch {
	case len(pos) != 1:
		return e.Usage("machinery accept takes one story id")
	case *verdict != "accept" && *verdict != "reject":
		return e.Usage("--verdict must be accept or reject")
	case *observed == "":
		return e.Usage("machinery accept needs --observed")
	case *closeIt && *verdict != "accept":
		return e.Usage("--close needs --verdict accept")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	c, err := backlog.New(p, *actor)
	if err != nil {
		return e.Fail(err)
	}
	if c.Actor == "" {
		return e.Usage("no actor: pass --actor or set BEADS_ACTOR")
	}
	is, code := showStory(e, c, pos[0])
	if is == nil {
		return code
	}
	head, err := gitx.Head(p.Root)
	if err != nil {
		return e.Fail(fmt.Errorf("cannot read the current commit: %v", err))
	}
	line := fmt.Sprintf("machinery:verdict %s commit=%s actor=%s observed=%s", *verdict, head, c.Actor, backlog.Quote(*observed))
	if err := c.Note(is.ID, line); err != nil {
		return e.Fail(err)
	}
	fmt.Fprintln(e.Stdout, line)
	if *verdict != "accept" {
		return 0
	}
	if is, code = showStory(e, c, is.ID); is == nil {
		return code
	}
	issues, err := capstoneBacklog(c, is)
	if err != nil {
		return e.Fail(err)
	}
	s, err := pm.StoryGate(design.Load(p), is, issues)
	if err != nil {
		return e.Fail(err)
	}
	code = report.PrintAll(e.Stdout, "story", []*report.Stage{s})
	if code == 0 && *closeIt {
		if err := c.Close(is.ID, "accepted at "+gitx.Short(head)); err != nil {
			return e.Fail(err)
		}
		fmt.Fprintf(e.Stdout, "closed: %s\n", is.ID)
	}
	return code
}
