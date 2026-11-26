package cli

import (
	"fmt"
	"strings"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/pm"
)

func init() {
	Register(&Command{Name: "decide", Summary: "Record a decision in the backlog, with the ids it governs.", Run: runDecide})
}

func runDecide(e *Env, args []string) int {
	fs := e.Flags("decide")
	why := fs.String("why", "", "the reason and the alternatives")
	var ids StringList
	fs.Var(&ids, "ids", "ids or id patterns the decision governs (repeatable, comma-separated)")
	supersedes := fs.String("supersedes", "", "the decision this one replaces")
	vetoable := fs.Bool("vetoable", false, "an agent decided within its remit; the owner may overturn it")
	actor := fs.String("actor", "", "who decides (default BEADS_ACTOR, else git user.name)")
	pos, ok := parseMixed(e, fs, args)
	if !ok {
		return 2
	}
	switch {
	case len(pos) != 1 || strings.TrimSpace(pos[0]) == "":
		return e.Usage("machinery decide takes one title")
	case strings.TrimSpace(*why) == "":
		return e.Usage("machinery decide needs --why")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	governed := splitIDs(ids)
	if bad := unknownPattern(design.Load(p), governed); bad != "" {
		return e.Usage("--ids %s names no design id", bad)
	}
	c, err := backlog.New(p, *actor)
	if err != nil {
		return e.Fail(err)
	}
	if c.Actor == "" {
		return e.Usage("no actor: pass --actor or set BEADS_ACTOR")
	}
	if *supersedes != "" {
		old, code := showStory(e, c, *supersedes)
		if old == nil {
			return code
		}
		if !pm.IsDecision(old) {
			return e.Usage("%s is not a decision: it has no machinery:decision line", old.ID)
		}
		issues, err := c.List()
		if err != nil {
			return e.Fail(err)
		}
		for _, d := range pm.Decisions(issues) {
			if d.ID == old.ID && d.By != "" {
				return e.Usage("%s is superseded already by %s", old.ID, d.By)
			}
		}
	}
	id, err := c.CreateItem(backlog.Item{Title: pm.OneLine(pos[0]), Type: "decision",
		Description: pm.DecisionDescription(*why, c.Actor, governed, *supersedes, *vetoable)})
	if err != nil {
		return e.Fail(err)
	}
	fmt.Fprintf(e.Stdout, "decision: %s\n", id)
	if *supersedes != "" {
		if err := c.Supersede(*supersedes, id); err != nil {
			return e.Fail(err)
		}
		fmt.Fprintf(e.Stdout, "superseded: %s by %s\n", *supersedes, id)
	}
	return 0
}

// unknownPattern returns the first id or pattern that names no declared id
// of the design, or "".
func unknownPattern(d *design.Design, pats []string) string {
	ids := d.DeclaredIDs()
	for _, pt := range pats {
		found := false
		for _, id := range ids {
			if design.MatchID(pt, id) {
				found = true
				break
			}
		}
		if !found {
			return pt
		}
	}
	return ""
}
