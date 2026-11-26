package stages

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
)

func init() { Bind("xref-failure", XrefFailure) }

var xrefFailureRules = []string{"subject-resolves", "invoke-failure-mode", "dependency-failure-mode", "tiered-action-abuse",
	"failure-state-mode", "failure-event-mode"}

// The rules that hold the failure states and failure events of the machines.
const (
	ruleFailureState = "failure-state-mode"
	ruleFailureEvent = "failure-event-mode"
)

// XrefFailure is stage xref-failure (section 6.6).
func XrefFailure(ctx *Context) *report.Stage {
	s := report.New("xref-failure")
	p := ctx.P
	d := ctx.Design()
	fstates, fevents := failureSubjects(d)
	if !d.HasFailureModes() {
		if len(fstates) == 0 && len(fevents) == 0 {
			return s.NotApplicable("no design.failure_modes configured")
		}
		// the machines declare failures and nothing says what to do about
		// them: only the two rules that ask for it run
		w := waiversFor(p, ruleFailureState, ruleFailureEvent)
		failureGaps(s, w, fstates, fevents, nil, func(when string) string {
			return "configure design.failure_modes and declare what the system must do when it " + when
		})
		w.stale(s)
		return s.Count(len(fstates), "failure states").Count(len(fevents), "failure events")
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
	systems := map[string]bool{}
	for _, sy := range p.Cfg.Systems {
		systems["system:"+sy.Name] = true
	}
	named, abused := map[string]bool{}, map[string]bool{}
	// linked: the failure mode names a subject of a known form that resolves
	// in the design, with a required value. unlinked: every other one that is
	// not declared subject-less: an imported row no entry names, a subject
	// that is missing, malformed or resolves to nothing, a no_subject without
	// a reason or beside a subject.
	// noRequired: of the unlinked, those whose subject resolves and that
	// only lack a valid required value.
	linked, unlinked, subjectLess, unlinkedRows, noRequired := 0, 0, 0, 0, 0
	for _, fm := range d.FailureModes {
		if fm.SubjectLess() {
			subjectLess++
			continue // names nothing, and says why (section 5.4)
		}
		// valued: the subject comes with a required value; without one the
		// subject is looked up all the same, and stage failure-modes has
		// the finding
		resolves, valued := fm.NamesSubject(), fm.WellFormed()
		switch {
		case !resolves:
			// stage failure-modes has the finding, when there is one
		case strings.HasPrefix(fm.Subject, "system:"):
			if !systems[fm.Subject] {
				resolves = false
				if !valued {
					break
				}
				// not a subject-resolves finding: no waiver covers an unknown system
				s.Errorf("unknown-system: %s: failure mode %s: %s is not a configured system; declare it under systems in machinery.yaml (no waiver covers an unknown system)",
					fm.Subject, fm.ID, strings.TrimPrefix(fm.Subject, "system:"))
			}
		default:
			if why := unresolved(d, fm.Subject, invokes, deps, fstates, fevents); why != "" {
				resolves = false
				if valued {
					// waived or not, the subject names nothing: the row is not linked
					w.find(s, "subject-resolves", fm.Subject, "failure mode %s: %s", fm.ID, why)
				}
			}
		}
		if resolves && !valued {
			// it names something of the design and does not say what the
			// system must do: counted on its own, and still not linked
			noRequired++
			resolves = false
		}
		if !resolves {
			unlinked++
			if fm.Imported {
				unlinkedRows++
			}
			continue
		}
		linked++
		named[fm.Subject] = true
		if fm.Kind == "abuse" {
			abused[fm.Subject] = true
		}
	}
	for _, id := range sortedSet(invokes) {
		if !named[id] {
			w.find(s, "invoke-failure-mode", id, "no failure mode names this invoke; declare what happens when it fails in %s", d.FailureModeTarget())
		}
	}
	for _, id := range sortedSet(deps) {
		if !named[id] {
			w.find(s, "dependency-failure-mode", id, "no failure mode names this dependency; declare what happens when it fails in %s", d.FailureModeTarget())
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
	failureGaps(s, w, fstates, fevents, named, func(when string) string {
		return "declare what the system must do when it " + when + " in " + d.FailureModeTarget()
	})
	w.stale(s)
	s.Count(len(d.FailureModes), "failure modes")
	if d.ImportsFM || unlinked > 0 {
		s.Count(linked, "linked").Count(unlinked, "unlinked")
	}
	if noRequired > 0 {
		s.Count(noRequired, "name a subject and no required value")
	}
	if subjectLess > 0 {
		s.Count(subjectLess, "subject-less")
	}
	s.Count(len(invokes), "invokes").Count(len(deps), "dependencies").Count(tiered, "tiered actions").
		Count(len(fstates), "failure states").Count(len(fevents), "failure events")
	if len(d.FailureModes) == 0 && len(invokes) == 0 && len(deps) == 0 && tiered == 0 && len(fstates) == 0 && len(fevents) == 0 && len(d.FMProblems) == 0 {
		s.NotDemonstrated("nothing to check")
	}
	switch {
	case unlinked > 0:
		// a row that names nothing and does not say why was not compared
		// with the design: one linked row among many is no pass
		todo := "give each a subject in " + d.FailureModeTarget()
		if unlinkedRows > 0 {
			todo = "link each in " + linksTarget(p)
		}
		reason := fmt.Sprintf("%d of %d failure modes name no subject; %s or declare it there with no_subject and a reason",
			unlinked, len(d.FailureModes), todo)
		if noRequired > 0 {
			reason += fmt.Sprintf("; %d of them name a subject and no required value: add required to each", noRequired)
		}
		s.NotDemonstrated(reason)
	case len(d.FailureModes) > 0 && linked == 0:
		// every failure mode is declared subject-less: none was compared
		s.NotDemonstrated("no failure mode names a subject")
	}
	return s
}

// linksTarget names where an imported row is linked or declared subject-less.
func linksTarget(p *project.Project) string {
	if im := p.Cfg.ImportedFailureModes(); im != nil && im.Links != "" {
		return im.Links
	}
	return "a links file (import.failure_modes.links)"
}

// failureSubjects returns the subject of every failure state
// (state:<machine id>.<path>) and of every failure event
// (event:<machine id>.<event>) of the machines.
func failureSubjects(d *design.Design) (states, events map[string]bool) {
	states, events = map[string]bool{}, map[string]bool{}
	for _, m := range d.Machines {
		for _, p := range m.FailureStates() {
			states["state:"+m.ID+"."+p] = true
		}
		for _, e := range m.FailureEvents() {
			events["event:"+m.ID+"."+e] = true
		}
	}
	return states, events
}

// failureGaps reports every failure state and failure event no failure mode
// names; todo says what to do, given when the failure happens.
func failureGaps(s *report.Stage, w *waivers, states, events, named map[string]bool, todo func(when string) string) {
	for _, id := range sortedSet(states) {
		if !named[id] {
			w.find(s, ruleFailureState, id, "no failure mode names this failure state; %s", todo("is entered"))
		}
	}
	for _, id := range sortedSet(events) {
		if !named[id] {
			w.find(s, ruleFailureEvent, id, "no failure mode names this failure event; %s", todo("arrives"))
		}
	}
}

// UnmodeledFailures returns the findings of failure-state-mode and
// failure-event-mode that no waiver covers, as waivers without a reason,
// sorted by rule and subject (section 17.2). It returns none while the
// failure-mode file has problems: what it names is then unknown.
func UnmodeledFailures(ctx *Context) []project.Waiver {
	d := ctx.Design()
	if len(d.FMProblems) > 0 {
		return nil
	}
	states, events := failureSubjects(d)
	named := map[string]bool{}
	for _, fm := range d.FailureModes {
		if fm.WellFormed() {
			named[fm.Subject] = true
		}
	}
	waived := map[string]bool{}
	for _, wv := range ctx.P.Cfg.Waivers {
		waived[wv.Rule+" "+wv.ID] = true
	}
	var res []project.Waiver
	for _, g := range []struct {
		rule string
		ids  map[string]bool
	}{{ruleFailureEvent, events}, {ruleFailureState, states}} {
		for _, id := range sortedSet(g.ids) {
			if !named[id] && !waived[g.rule+" "+id] {
				res = append(res, project.Waiver{Rule: g.rule, ID: id})
			}
		}
	}
	return res
}

// unresolved says why a well-formed subject names nothing, or "".
// Malformed subjects are the failure-modes stage's finding.
func unresolved(d *design.Design, subject string, invokes, deps, fstates, fevents map[string]bool) string {
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
	case "state":
		if fstates[subject] {
			return ""
		}
		mid, path, _ := strings.Cut(name, ".")
		m := d.MachineByID(mid)
		switch {
		case m == nil:
			return fmt.Sprintf("no machine has the id %s", mid)
		case !m.Resolver.PathSet[path]:
			return fmt.Sprintf("machine %s has no state %s", mid, path)
		}
		return fmt.Sprintf("state %s of machine %s is not tagged failure", path, mid)
	case "event":
		if fevents[subject] {
			return ""
		}
		mid, ev, _ := strings.Cut(name, ".")
		m := d.MachineByID(mid)
		if m == nil {
			return fmt.Sprintf("no machine has the id %s", mid)
		}
		if m.RefusalEvents()[ev] != "" {
			// a declared refusal: a failure mode may name it, none is asked for
			return ""
		}
		return fmt.Sprintf("%s is not in the _failure_events of machine %s (nor in its _refusal_events)", ev, mid)
	case "machine":
		// the machine as a whole, bound to the model or not
		if d.MachineByID(name) == nil {
			return fmt.Sprintf("no machine has the id %s", name)
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
