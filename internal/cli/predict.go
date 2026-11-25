package cli

import (
	"fmt"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/evidence"
	"github.com/RamXX/machinery/internal/gitx"
)

func init() {
	Register(&Command{Name: "predict", Summary: "Record a verifier's prediction on a backlog story.", Run: runPredict})
}

func runPredict(e *Env, args []string) int {
	fs := e.Flags("predict")
	observation := fs.String("observation", "", "what would disprove the claim")
	expect := fs.String("expect", "", "the predicted outcome")
	actor := fs.String("actor", "", "who predicts (default BEADS_ACTOR, else git user.name)")
	pos, ok := parseMixed(e, fs, args)
	if !ok {
		return 2
	}
	switch {
	case len(pos) != 1:
		return e.Usage("machinery predict takes one story id")
	case *observation == "" || *expect == "":
		return e.Usage("machinery predict needs --observation and --expect")
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
	for _, pr := range backlog.ParseTags(is).Proofs {
		if pr.Actor == c.Actor && evidence.ValidFor(p, head, pr.Commit) {
			fmt.Fprintf(e.Stderr, "machinery: refused: %s posted the proof for %s; a fresh verifier predicts\n", c.Actor, gitx.Short(head))
			return 1
		}
	}
	line := fmt.Sprintf("machinery:prediction commit=%s actor=%s observation=%s expect=%s",
		head, c.Actor, backlog.Quote(*observation), backlog.Quote(*expect))
	if err := c.Note(is.ID, line); err != nil {
		return e.Fail(err)
	}
	fmt.Fprintln(e.Stdout, line)
	return 0
}
