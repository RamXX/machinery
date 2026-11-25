package stages

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/report"
)

func init() { Bind("xref-failure", XrefFailure) }

var xrefFailureRules = []string{"subject-resolves", "invoke-failure-mode", "dependency-failure-mode", "tiered-action-abuse"}

// XrefFailure is stage xref-failure (section 6.6).
func XrefFailure(ctx *Context) *report.Stage {
	s := report.New("xref-failure")
	p := ctx.P
	d := ctx.Design()
	if !d.HasFM {
		return s.NotApplicable("no design.failure_modes configured")
	}
	w := waiversFor(p, xrefFailureRules...)

	invokes := map[string]bool{} // invoke:<machine id>.<src>
	for _, m := range d.Machines {
		for _, st := range m.Table.States {
			for _, src := range st.Invoke {
				invokes["invoke:"+m.ID+"."+src] = true
			}
		}
	}
	deps := map[string]bool{}
	for _, dep := range p.Cfg.Dependencies {
		deps["dependency:"+dep.Name] = true
	}
	named, abused := map[string]bool{}, map[string]bool{}
	for _, fm := range d.FailureModes {
		named[fm.Subject] = true
		if fm.Kind == "abuse" {
			abused[fm.Subject] = true
		}
		if why := unresolved(d, fm.Subject, invokes, deps); why != "" {
			w.find(s, "subject-resolves", fm.Subject, "failure mode %s: %s", fm.ID, why)
		}
	}
	for _, id := range sortedSet(invokes) {
		if !named[id] {
			w.find(s, "invoke-failure-mode", id, "no failure mode names this invoke; declare what happens when it fails in %s", p.Cfg.Design.FailureModes)
		}
	}
	for _, id := range sortedSet(deps) {
		if !named[id] {
			w.find(s, "dependency-failure-mode", id, "no failure mode names this dependency; declare what happens when it fails in %s", p.Cfg.Design.FailureModes)
		}
	}
	tiered := 0
	for _, sv := range d.Served {
		tier, areas := d.Tier(sv.ID)
		if tier == design.Routine {
			continue
		}
		tiered++
		if !abused[sv.ID] {
			var names []string
			for _, a := range areas {
				names = append(names, a.Name)
			}
			sort.Strings(names)
			w.find(s, "tiered-action-abuse", sv.ID, "%s action (area %s) has no failure mode of kind abuse; add one with subject %s", tier, strings.Join(names, ", "), sv.ID)
		}
	}
	w.stale(s)
	s.Count(len(d.FailureModes), "failure modes").Count(len(invokes), "invokes").Count(len(deps), "dependencies").Count(tiered, "tiered actions")
	if len(d.FailureModes) == 0 && len(invokes) == 0 && len(deps) == 0 && tiered == 0 && len(d.FMProblems) == 0 {
		s.NotDemonstrated("nothing to check")
	}
	return s
}

// unresolved says why a well-formed subject names nothing, or "".
// Malformed subjects are the failure-modes stage's finding.
func unresolved(d *design.Design, subject string, invokes, deps map[string]bool) string {
	kind, name, ok := strings.Cut(subject, ":")
	if !ok {
		return ""
	}
	switch kind {
	case "invoke":
		if invokes[subject] {
			return ""
		}
		mid, _, _ := strings.Cut(name, ".")
		if d.MachineByID(mid) == nil {
			return fmt.Sprintf("no machine has the id %s", mid)
		}
		return fmt.Sprintf("machine %s has no invoke with that src", mid)
	case "dependency":
		if !deps[subject] {
			return fmt.Sprintf("%s is not a configured dependency", name)
		}
	case "act":
		if !d.IsAct(subject) {
			return fmt.Sprintf("%s is not an action of the model", name)
		}
	case "entity":
		if e, _ := design.FindEntity(d.Models, name); e == nil {
			return fmt.Sprintf("%s is not an entity of the model", name)
		}
	}
	return ""
}

func sortedSet(m map[string]bool) []string {
	res := make([]string, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	sort.Strings(res)
	return res
}
