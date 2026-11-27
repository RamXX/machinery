package cli

import (
	"fmt"

	"github.com/RamXX/machinery/v2/internal/backlog"
	"github.com/RamXX/machinery/v2/internal/gitx"
	"github.com/RamXX/machinery/v2/internal/pm"
	"github.com/RamXX/machinery/v2/internal/stages"
)

func init() {
	Register(&Command{Name: "review", Summary: "Record the checker's review of the backlog, or of an intake's design or stories.", Run: runReview})
}

func runReview(e *Env, args []string) int {
	fs := e.Flags("review")
	verdict := fs.String("verdict", "", "approved or rejected")
	var issues StringList
	fs.Var(&issues, "issue", "with rejected: \"<rule>: <scope>: <instances>\" (repeatable, at most five)")
	actor := fs.String("actor", "", "who reviews (default BEADS_ACTOR, else git user.name)")
	pos, ok := parseMixed(e, fs, args)
	if !ok {
		return 2
	}
	kind, id := "", ""
	switch {
	case len(pos) == 1 && pos[0] == pm.ReviewBacklog:
		kind = pos[0]
	case len(pos) == 2 && (pos[0] == pm.ReviewDesign || pos[0] == pm.ReviewIntake):
		kind, id = pos[0], pos[1]
	default:
		return e.Usage("machinery review takes backlog, design <intake id> or intake <intake id>")
	}
	switch {
	case *verdict != pm.Approved && *verdict != pm.Rejected:
		return e.Usage("machinery review needs --verdict approved|rejected")
	case *verdict == pm.Approved && len(issues) > 0:
		return e.Usage("an approved review takes no --issue")
	case *verdict == pm.Rejected && (len(issues) < 1 || len(issues) > 5):
		return e.Usage("a rejected review needs one to five --issue \"<rule>: <scope>: <instances>\"")
	}
	for _, is := range issues {
		if _, _, _, ok := pm.ParseIssue(is); !ok {
			return e.Usage("--issue %q: an issue names its rule and its scope: \"<rule>: <scope>: <instances>\"", is)
		}
	}
	who, err := backlog.WhoFromEnv()
	if err != nil {
		return e.Usage("%v", err)
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
	var in *pm.Intake
	if id != "" {
		is, code := showStory(e, c, id)
		if is == nil {
			return code
		}
		if in, ok = pm.ParseIntake(is); !ok {
			return e.Usage("%s is not an intake item: it has no machinery:intake line", is.ID)
		}
	}
	head, err := gitx.Head(p.Root)
	if err != nil {
		return e.Fail(fmt.Errorf("cannot read the current commit: %v", err))
	}
	all, err := c.List()
	if err != nil {
		return e.Fail(err)
	}
	ctx := &stages.Context{P: p}
	d := ctx.Design()
	contract := pm.ContractOf(d)
	var stories []string
	var planners []pm.Who
	if in != nil {
		planners = pm.IntakePlanners(in.Tags)
	} else {
		stories = pm.BuildWaves(d, all).Wave1()
		planners = pm.BacklogPlanners(all, stories)
	}
	me := pm.Who{Actor: c.Actor, Session: who.Session, Agent: who.Agent}
	same, warns := pm.Compare(me, planners)
	if same != "" {
		fmt.Fprintf(e.Stderr, "machinery: refused: the reviewer is the planner: %s; the checker reviews in a fresh context\n", same)
		return 1
	}
	line := pm.ReviewLine(kind, *verdict, head, contract.Hash, me, stories, issues)
	target := id
	if in == nil {
		if item := pm.ReviewItem(all); item != nil {
			target = item.ID
		} else {
			target, err = c.CreateItem(backlog.Item{Title: "Reviews of the backlog", Type: "task",
				Description: "Reviews of the backlog by the checker, one machinery:review line each.\n",
				Labels:      []string{pm.ReviewLabel}})
			if err != nil {
				return e.Fail(err)
			}
			if err := c.Close(target, "a record of the checker's reviews, never work"); err != nil {
				return e.Fail(err)
			}
		}
	}
	if err := c.Note(target, line); err != nil {
		return e.Fail(err)
	}
	for _, w := range warns {
		fmt.Fprintf(e.Stdout, "warn: %s\n", w)
	}
	what := kind
	if id != "" {
		what += " " + id
	}
	fmt.Fprintf(e.Stdout, "review: %s %s at %s on %s\n", what, *verdict, contract.Short(), target)
	return 0
}
