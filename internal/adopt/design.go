package adopt

import (
	"bytes"
	"sort"

	"github.com/RamXX/machinery/internal/arch"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gen"
	"github.com/RamXX/machinery/internal/modelith"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/testpolicy"
	"github.com/RamXX/machinery/internal/vital"
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
		cores, err := vital.Cores(design.Load(p))
		if err != nil {
			return err
		}
		vs, _, err := arch.Violations(p, cores)
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

// pendingGen lists the generated files machinery gen would write now, and
// the renderings among them that adopt may not write, which it leaves.
func pendingGen(d *design.Design, may func(string) bool) ([]string, []Left) {
	set := map[string]bool{}
	if diffs, _, err := gen.Compare(d); err == nil {
		for _, df := range diffs {
			if df.Kind != gen.Unexpected {
				set[df.Path] = true
			}
		}
	}
	var left []Left
	for _, rel := range d.P.Cfg.Design.Models {
		want, err := modelith.Render(d.P, rel)
		if err != nil {
			continue
		}
		md := modelith.RenderedPath(rel)
		got, rerr := d.P.Read(md)
		switch {
		case rerr == nil && bytes.Equal(got, want):
		case !may(md):
			left = append(left, Left{Path: md, Missing: rerr != nil})
		default:
			set[md] = true
		}
	}
	var res []string
	for f := range set {
		res = append(res, f)
	}
	sort.Strings(res)
	sort.Slice(left, func(i, j int) bool { return left[i].Path < left[j].Path })
	return res, left
}
