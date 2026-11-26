package cli

import (
	"fmt"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/landing"
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
	repair := fs.Bool("repair", false, "with --close: close the story while the integration branch is red, as its repair")
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
	case *repair && !*closeIt:
		return e.Usage("--repair needs --close")
	}
	who, err := backlog.WhoFromEnv()
	if err != nil {
		return e.Usage("%v", err)
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	var red *landing.Red
	if *closeIt {
		if red, err = landing.LoadRed(p); err != nil {
			return e.Fail(err)
		}
		if red != nil && !*repair {
			fmt.Fprintln(e.Stdout, red.Refusal())
			return 1
		}
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
	if red != nil {
		if err := c.Note(is.ID, "machinery:repair "+gitx.Short(red.Commit)); err != nil {
			return e.Fail(err)
		}
	}
	line := fmt.Sprintf("machinery:verdict %s commit=%s actor=%s%s observed=%s", *verdict, head, c.Actor, who.Part(), backlog.Quote(*observed))
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
	s, err := gateOf(p, c, is)
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
