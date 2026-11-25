package adopt

import (
	"bytes"
	"sort"

	"github.com/RamXX/machinery/internal/arch"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gen"
	"github.com/RamXX/machinery/internal/modelith"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/tags"
	"github.com/RamXX/machinery/internal/testpolicy"
)

// baselines is step 2: the tests baseline and, when constraints are
// configured, the architecture baseline.
func (w *worker) baselines() error {
	p := w.p
	if testpolicy.Applies(p) && testpolicy.BaselinePath(p) != "" {
		vs, _, err := testpolicy.Violations(p)
		if err != nil {
			return err
		}
		inTests := func(old *project.Project, f string) bool {
			return old.InScope(f) || project.Match(old.TestRoots(), f)
		}
		if err := w.merge(testpolicy.BaselinePath(p), vs, inTests); err != nil {
			return err
		}
	}
	if arch.Applies(p) && arch.BaselinePath(p) != "" {
		vs, _, err := arch.Violations(p)
		if err != nil {
			return err
		}
		inScope := func(old *project.Project, f string) bool { return old.InScope(f) }
		if err := w.merge(arch.BaselinePath(p), vs, inScope); err != nil {
			return err
		}
	}
	return nil
}

// servedIDs are the act ids some code file serves with a machinery:serves tag.
func servedIDs(p *project.Project, d *design.Design) map[string]bool {
	res := map[string]bool{}
	if p.Cfg.Trace == nil {
		return res
	}
	code, err := tags.CodeFiles(p)
	if err != nil {
		return res
	}
	found, err := tags.Scan(p, code)
	if err != nil {
		return res
	}
	ids := d.AllIDs()
	for _, t := range found {
		if t.Kind != tags.Serves {
			continue
		}
		for _, tok := range t.IDs {
			for _, id := range tags.Expand(tok, ids) {
				res[id] = true
			}
		}
	}
	return res
}

// pendingGen lists the generated files machinery gen would write now.
func pendingGen(d *design.Design) []string {
	set := map[string]bool{}
	if diffs, _, err := gen.Compare(d); err == nil {
		for _, df := range diffs {
			if df.Kind != gen.Unexpected {
				set[df.Path] = true
			}
		}
	}
	for _, rel := range d.P.Cfg.Design.Models {
		want, err := modelith.Render(d.P, rel)
		if err != nil {
			continue
		}
		md := modelith.RenderedPath(rel)
		if got, rerr := d.P.Read(md); rerr != nil || !bytes.Equal(got, want) {
			set[md] = true
		}
	}
	var res []string
	for f := range set {
		res = append(res, f)
	}
	sort.Strings(res)
	return res
}
