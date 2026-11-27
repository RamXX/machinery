package stages

import (
	"fmt"
	"sort"

	"github.com/RamXX/machinery/internal/arch"
	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/testpolicy"
)

func init() { Bind("scope", Scope) }

// Scope is stage scope (section 15.11): every file a story changed since the
// base lies inside its machinery:scope globs, and none is a design source
// unless the story lands the design of its intake.
func Scope(ctx *Context) *report.Stage {
	s := report.New("scope")
	if ctx.Story == "" {
		return s.NotApplicable("no story named; run machinery check --base <ref> --story <id>")
	}
	if ctx.Base == "" {
		return s.NotDemonstrated("no base ref; run machinery check --base <ref> --story " + ctx.Story)
	}
	p := ctx.P
	c, err := backlog.New(p, "")
	if err != nil {
		return s.NotDemonstrated(err.Error())
	}
	is, err := c.Show(ctx.Story)
	if err != nil {
		return s.NotDemonstrated(fmt.Sprintf("the story %s cannot be read: %v", ctx.Story, err))
	}
	t := backlog.ParseTags(is)
	if len(t.Scope) == 0 {
		return s.NotDemonstrated(is.ID + " declares no scope; the planner adds a machinery:scope line")
	}
	mb, err := gitx.MergeBase(p.Root, ctx.Base, "HEAD")
	if err != nil {
		s.Errorf("git merge-base: %v", err)
		return s
	}
	if mb == "" {
		return s.NotDemonstrated(fmt.Sprintf("no merge base of %s and HEAD", ctx.Base))
	}
	files, err := gitx.StagedSince(p.Root, mb)
	if err != nil {
		s.Errorf("git diff-index: %v", err)
		return s
	}
	sort.Strings(files)
	// the owed list and the baselines are pruned by the builder and held by the ratchet
	exempt := map[string]bool{p.Cfg.Design.Owed: true, testpolicy.BaselinePath(p): true, arch.BaselinePath(p): true}
	sources := map[string]bool{}
	for _, f := range ctx.Design().ContractFiles() {
		sources[f] = true
	}
	n := 0
	for _, f := range files {
		if exempt[f] {
			continue
		}
		n++
		if sources[f] && !t.IntakeDesign {
			s.Errorf("scope: %s is a design source; a design change is an intake", f)
			continue
		}
		if !project.Match(t.Scope, f) {
			s.Errorf("scope: %s is outside the scope of %s", f, is.ID)
		}
	}
	s.Count(n, "files").Count(len(t.Scope), "scope globs")
	if n == 0 {
		s.NotDemonstrated("nothing changed since " + ctx.Base)
	}
	return s
}
