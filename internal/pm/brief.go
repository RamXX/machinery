package pm

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/RamXX/machinery/v2/internal/backlog"
	"github.com/RamXX/machinery/v2/internal/design"
	"github.com/RamXX/machinery/v2/internal/formal"
	"github.com/RamXX/machinery/v2/internal/gen"
	"github.com/RamXX/machinery/v2/internal/gitx"
	"github.com/RamXX/machinery/v2/internal/project"
)

// Roles of a brief (section 15.5).
const (
	Builder  = "builder"
	Verifier = "verifier"
	Modeler  = "modeler"
)

const formatsDoc = "docs/formats.md"

// Brief writes the brief of a story for a role, as pointers. ids are the
// story's design ids (for a capstone, those of its intake's other stories);
// decisions are the current decisions of the backlog (section 15.8).
func Brief(w io.Writer, d *design.Design, is *backlog.Issue, role string, ids []string, decisions []Decision) error {
	p := d.P
	t := backlog.ParseTags(is)
	tier, areas := StoryTier(d, is, ids)
	specs, err := formal.Load(p)
	if err != nil {
		return err
	}
	base := "main"
	if p.Cfg.Backlog != nil {
		base = p.Cfg.Backlog.Base
	}
	if role == Modeler && len(CriticalAreas(d, ids)) == 0 {
		return project.Errf("story %s has no id in a critical area: the modeler role is for a story that needs a formal model", is.ID)
	}
	fmt.Fprintf(w, "BRIEF %s (%s)\n", is.ID, role)
	fmt.Fprintf(w, "title: %s\n", is.Title)
	fmt.Fprintf(w, "tier: %s%s\n", tier, areaWhy(d, areas))
	fmt.Fprintf(w, "route: %s\n", Route(d, tier))
	agent := fmt.Sprintf("%s-%s", is.ID, role)
	if role != Modeler {
		fmt.Fprintf(w, "agent: %s\n", agent)
	}
	head := "none"
	if h, err := gitx.Head(p.Root); err == nil {
		head = gitx.Short(h)
	}
	fmt.Fprintf(w, "contract: %s at %s\n", ContractOf(d).Short(), head)
	fmt.Fprintln(w, "ids:")
	for _, id := range ids {
		fmt.Fprintf(w, "  %s  %s\n", id, IDTitle(d, id))
	}
	critical := CriticalAreas(d, ids)
	mine := specsOf(specs, critical, ids)
	var read, run []string
	switch role {
	case Modeler:
		read = modelerRead(d, mine, critical)
		run = append(run, "machinery formal gen")
		names := specNames(mine, critical)
		for _, n := range names {
			run = append(run, "machinery formal check "+n)
		}
		run = append(run, "machinery formal baseline")
	case Verifier:
		read = idRead(d, mine, ids)
		run = []string{
			"export MACHINERY_AGENT=" + agent,
			fmt.Sprintf("machinery predict %s --observation \"...\" --expect \"...\"", is.ID),
			"machinery proof --story " + is.ID,
			fmt.Sprintf("machinery accept %s --verdict accept|reject --observed \"...\"", is.ID),
		}
	default:
		read = idRead(d, mine, ids)
		e2e := "<the end-to-end command>"
		if len(p.Cfg.Commands.E2E) > 0 {
			e2e = strings.Join(p.Cfg.Commands.E2E, " ")
		}
		run = []string{
			"export MACHINERY_AGENT=" + agent,
			fmt.Sprintf("machinery check --base %s --story %s", base, is.ID),
			fmt.Sprintf("machinery run --affected --base %s -- %s", base, e2e),
			"machinery verify --story " + is.ID,
			fmt.Sprintf("machinery proof --story %s --post", is.ID),
		}
	}
	section(w, "read", read)
	var governing []string
	for _, dec := range decisions {
		if dec.Governs(ids) {
			governing = append(governing, dec.ID+"  "+dec.Title)
		}
	}
	section(w, "decisions", governing)
	scope := append([]string{}, t.Scope...)
	sort.Strings(scope)
	section(w, "scope", scope)
	section(w, "run", run)
	if role == Verifier {
		weak, err := weakenedSince(d, base)
		if err != nil {
			return err
		}
		section(w, "weakened since "+base, weak)
	}
	fmt.Fprintf(w, "done when: %s\n", doneWhen(is.ID, role, tier, critical))
	return nil
}

func section(w io.Writer, name string, lines []string) {
	fmt.Fprintf(w, "%s:\n", name)
	if len(lines) == 0 {
		fmt.Fprintln(w, "  none")
	}
	for _, l := range lines {
		fmt.Fprintf(w, "  %s\n", l)
	}
}

func areaWhy(d *design.Design, areas []string) string {
	var parts []string
	for _, name := range areas {
		for _, a := range d.Areas {
			if a.Name == name {
				parts = append(parts, a.Name+": "+a.Why)
			}
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, "; ") + ")"
}

// specsOf returns the specs that model one of the areas or name a cx id of
// the story.
func specsOf(specs []*formal.Spec, areas, ids []string) []*formal.Spec {
	want := setOf(areas)
	var res []*formal.Spec
	for _, s := range specs {
		hit := false
		for _, a := range s.Areas {
			hit = hit || want[a]
		}
		for _, id := range ids {
			hit = hit || strings.HasPrefix(id, "cx:"+s.Spec+":")
		}
		if hit {
			res = append(res, s)
		}
	}
	return res
}

func specNames(specs []*formal.Spec, areas []string) []string {
	if len(specs) == 0 {
		return areas
	}
	var res []string
	for _, s := range specs {
		res = append(res, s.Spec)
	}
	return res
}

// idRead lists the files a builder or verifier reads for the ids.
func idRead(d *design.Design, specs []*formal.Spec, ids []string) []string {
	set := map[string]bool{}
	for _, id := range ids {
		kind, _, _ := strings.Cut(id, ":")
		switch kind {
		case "tc":
			if m := d.MachineOfCase(id); m != nil {
				set[m.Path] = true
				set[gen.TablePath(d, m)] = true
			}
		case "act", "inv", "sc":
			if src := sourceOf(d, id); src != "" {
				set[src] = true
			}
		case "fm":
			if o := d.Obligation(id); o != nil {
				set[o.Source] = true
			}
			if im := d.P.Cfg.ImportedFailureModes(); im != nil && im.Links != "" {
				set[im.Links] = true
			}
		case "fv", "cx":
			for _, s := range specs {
				set[s.Path] = true
			}
			for _, t := range d.Traces {
				if t.ID == id {
					set[t.Path] = true
				}
			}
		}
	}
	if d.P.Exists(formatsDoc) {
		set[formatsDoc] = true
	}
	delete(set, "")
	return sortedKeys(set)
}

func sourceOf(d *design.Design, id string) string {
	if o := d.Obligation(id); o != nil {
		return o.Source
	}
	for _, s := range d.Served {
		if s.ID == id {
			return s.Source
		}
	}
	return ""
}

// modelerRead lists the machine or model each spec is derived from and its
// manifest; an area without a spec points at the risk file.
func modelerRead(d *design.Design, specs []*formal.Spec, areas []string) []string {
	set := map[string]bool{}
	for _, s := range specs {
		set[s.Path] = true
		if s.Model != "" {
			set[s.Model] = true
		}
		for _, m := range d.Machines {
			if s.Machine != "" && (m.ID == s.Machine || m.Base == s.Machine) {
				set[m.Path] = true
			}
		}
	}
	if len(specs) == 0 && len(areas) > 0 {
		set[d.P.Cfg.Design.Risk] = true
	}
	delete(set, "")
	return sortedKeys(set)
}

// weakenedSince lists the formal baseline acceptances the base does not have.
func weakenedSince(d *design.Design, base string) ([]string, error) {
	p := d.P
	cur, err := formal.LoadBaseline(p)
	if err != nil {
		return nil, err
	}
	data, ok := gitx.Show(p.Root, base, formal.BaselinePath(p))
	var readErr error
	if !ok {
		readErr = fmt.Errorf("absent at %s", base)
	}
	old, err := formal.ParseBaseline(data, readErr)
	if err != nil {
		return nil, err
	}
	var res []string
	for spec, sb := range cur.Specs {
		had := map[formal.Acceptance]bool{}
		if ob := old.Specs[spec]; ob != nil {
			for _, a := range ob.Accepted {
				had[a] = true
			}
		}
		for _, a := range sb.Accepted {
			if !had[a] {
				res = append(res, fmt.Sprintf("%s: %s (%s)", spec, a.Change, a.Reason))
			}
		}
	}
	sort.Strings(res)
	return res, nil
}

func doneWhen(id, role, tier string, critical []string) string {
	switch role {
	case Verifier:
		return fmt.Sprintf("machinery accept %s --verdict accept exits 0", id)
	case Modeler:
		var fv []string
		for _, a := range critical {
			fv = append(fv, "fv:"+a)
		}
		if len(fv) == 0 {
			return "machinery check --stage formal passes"
		}
		return "machinery check --stage formal passes and " + strings.Join(fv, ", ") + " is proven"
	}
	if tier == design.Routine {
		return fmt.Sprintf("machinery story %s passes", id)
	}
	return fmt.Sprintf("the proof is posted and a fresh verifier's machinery accept %s exits 0", id)
}
