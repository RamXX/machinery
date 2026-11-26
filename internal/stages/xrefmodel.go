package stages

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/machine"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
)

func init() { Bind("xref-model-machine", XrefModelMachine) }

const (
	ruleMachineBinding = "machine-binding"
	ruleEntityMachine  = "entity-machine"
)

var xrefModelRules = []string{ruleMachineBinding, "state-enum", "event-action", "action-event", ruleEntityMachine}

// xfinding is one finding of a cross-reference rule, before waivers.
type xfinding struct{ rule, subject, msg string }

type xfindings []xfinding

func (fs *xfindings) add(rule, subject, format string, a ...any) {
	*fs = append(*fs, xfinding{rule, subject, fmt.Sprintf(format, a...)})
}

// binding is what a machine binds to in the model (section 5.2): the entity
// of a lifecycle machine, or the ids an operational machine serves. problems
// are its machine-binding findings; a machine without any is bound.
type binding struct {
	entity      *design.Entity
	operational bool
	problems    []string
}

func (b binding) bound() bool { return len(b.problems) == 0 && (b.entity != nil || b.operational) }

func bindingOf(d *design.Design, m *machine.Machine) binding {
	serves, hasServes := m.Serves()
	switch {
	case m.LifecycleOf != "" && m.Role == "operational":
		return binding{problems: []string{"_lifecycle_of and _role: operational are both set; a machine is one or the other"}}
	case m.Role == "operational":
		b := binding{operational: true}
		if len(serves) == 0 {
			b.problems = append(b.problems, "operational and serves nothing in the model; bind it to an entity as its lifecycle (_lifecycle_of), or name what it serves in _serves (entity:<Name> or act:<Entity>.<action>)")
		}
		for _, id := range serves {
			if !servesResolves(d, id) {
				b.problems = append(b.problems, fmt.Sprintf("_serves names %s, which is not an entity or an action of the model", id))
			}
		}
		return b
	}
	ent := d.LifecycleEntity(m)
	if ent == "" {
		return binding{problems: []string{"not a lifecycle machine of a model entity and not operational; set _lifecycle_of or name the file after an entity, or set _role: operational and name what it serves in _serves"}}
	}
	e, _ := design.FindEntity(d.Models, ent)
	if e == nil {
		return binding{problems: []string{fmt.Sprintf("_lifecycle_of names entity %s, which is not in the model", ent)}}
	}
	b := binding{entity: e}
	if hasServes {
		b.problems = append(b.problems, fmt.Sprintf("_serves is set on the lifecycle machine of %s; a lifecycle machine serves its entity: remove _serves, or set _role: operational", e.Name))
	}
	return b
}

// servesResolves reports whether a _serves id names an entity or an action
// of the model.
func servesResolves(d *design.Design, id string) bool {
	kind, name, _ := strings.Cut(id, ":")
	switch kind {
	case "entity":
		e, _ := design.FindEntity(d.Models, name)
		return e != nil
	case "act":
		ent, act, ok := strings.Cut(name, ".")
		e, _ := design.FindEntity(d.Models, ent)
		if !ok || e == nil {
			return false
		}
		for _, a := range e.Actions {
			if a.Name == act {
				return true
			}
		}
	}
	return false
}

// xrefCounts is what the stage reports as checked.
type xrefCounts struct {
	machines, lifecycle, operational, states, events, actions, withActions int
}

// checkModelMachine applies the rules of section 6.4 to the design.
func checkModelMachine(d *design.Design) (fs xfindings, c xrefCounts) {
	c.machines = len(d.Machines)
	hasLifecycle := map[string]bool{}
	for _, m := range d.Machines {
		b := bindingOf(d, m)
		for _, p := range b.problems {
			fs.add(ruleMachineBinding, m.ID, "%s", p)
		}
		if b.operational {
			c.operational++
			continue
		}
		e := b.entity
		if e == nil {
			continue
		}
		c.lifecycle++
		hasLifecycle[e.Name] = true
		domain := domainStates(m)
		c.states += len(domain)
		checkStateEnum(&fs, d, m, e, domain)

		external, ok := eventSet(m, "_external_events")
		if !ok {
			fs.add("event-action", m.ID, "_external_events must be a list of event names")
		}
		// a failure event arrives from outside: it is external
		failure, ok := eventSet(m, "_failure_events")
		if !ok {
			fs.add("event-action", m.ID, "_failure_events must be a list of event names")
		}
		for ev := range failure {
			external[ev] = true
		}
		acts := map[string]bool{}
		for _, a := range e.Actions {
			acts[a.Name] = true
		}
		evs := m.Table.Events()
		c.events += len(evs)
		c.actions += len(e.Actions)
		evSet := map[string]bool{}
		for _, ev := range evs {
			evSet[ev] = true
			if !acts[ev] && !external[ev] {
				fs.add("event-action", m.ID+"."+ev, "event is not an action of %s; add the action to the model or list the event in _external_events (or in _failure_events when it reports a failure)", e.Name)
			}
		}
		for _, a := range e.Actions {
			if !evSet[a.Name] {
				fs.add("action-event", design.ActID(e.Name, a.Name), "no on event %s in machine %s (%s)", a.Name, m.ID, m.Path)
			}
		}
	}
	for _, md := range d.Models {
		for _, e := range md.Entities {
			if len(e.Actions) == 0 {
				continue
			}
			c.withActions++
			if !hasLifecycle[e.Name] {
				fs.add(ruleEntityMachine, "entity:"+e.Name, "declares %d action(s) and no machine is its lifecycle; write its lifecycle machine (_lifecycle_of: %s), or waive the rule with the reason it has no state", len(e.Actions), e.Name)
			}
		}
	}
	return fs, c
}

// XrefModelMachine is stage xref-model-machine (section 6.4).
func XrefModelMachine(ctx *Context) *report.Stage {
	s := report.New("xref-model-machine")
	p := ctx.P
	if !p.Cfg.HasMachines() {
		return s.NotApplicable("no design.machines configured")
	}
	if len(p.Cfg.Design.Models) == 0 {
		return s.NotApplicable("no design.models configured")
	}
	d := ctx.Design()
	w := waiversFor(p, xrefModelRules...)
	fs, c := checkModelMachine(d)
	for _, f := range fs {
		w.find(s, f.rule, f.subject, "%s", f.msg)
	}
	w.stale(s)
	s.Count(c.machines, "machines").Count(c.lifecycle, "lifecycle machines").Count(c.operational, "operational machines").
		Count(c.states, "domain states").Count(c.events, "events").Count(c.actions, "actions").Count(c.withActions, "entities with actions")
	if len(d.Machines) == 0 && len(d.MachineProblems) == 0 {
		s.NotDemonstrated("nothing to check")
	}
	if c.lifecycle == 0 {
		// nothing was compared with the model: never a pass
		s.NotDemonstrated("no machine is bound to a model entity")
	}
	return s
}

// BindingCounts returns how many machines bind to the model and how many do
// not (section 6.4); a waiver does not bind a machine. Every machine is
// unbound when there is no model.
func BindingCounts(ctx *Context) (bound, unbound int) {
	d := ctx.Design()
	for _, m := range d.Machines {
		if bindingOf(d, m).bound() {
			bound++
		} else {
			unbound++
		}
	}
	return bound, unbound
}

// UnboundDesign returns the findings of machine-binding and entity-machine
// that no waiver covers, as waivers without a reason, one per rule and
// subject, sorted (section 17.2). It returns none when the stage does not
// apply, or while a model or a machine cannot be loaded: what binds to what
// is then unknown.
func UnboundDesign(ctx *Context) []project.Waiver {
	p := ctx.P
	if !p.Cfg.HasMachines() || len(p.Cfg.Design.Models) == 0 {
		return nil
	}
	d := ctx.Design()
	if len(d.ModelProblems) > 0 || len(d.MachineProblems) > 0 {
		return nil
	}
	waived := map[string]bool{}
	for _, wv := range p.Cfg.Waivers {
		waived[wv.Rule+" "+wv.ID] = true
	}
	fs, _ := checkModelMachine(d)
	seen := map[string]bool{}
	var res []project.Waiver
	for _, f := range fs {
		key := f.rule + " " + f.subject
		if (f.rule != ruleMachineBinding && f.rule != ruleEntityMachine) || waived[key] || seen[key] {
			continue
		}
		seen[key] = true
		res = append(res, project.Waiver{Rule: f.rule, ID: f.subject})
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].Rule != res[j].Rule {
			return res[i].Rule < res[j].Rule
		}
		return res[i].ID < res[j].ID
	})
	return res
}

// domainStates lists the top-level states that start with an uppercase letter (section 5.2).
func domainStates(m *machine.Machine) []string {
	var res []string
	for _, st := range m.Table.States {
		if strings.Contains(st.Path, ".") {
			continue
		}
		if r := []rune(st.Path); len(r) > 0 && unicode.IsUpper(r[0]) {
			res = append(res, st.Path)
		}
	}
	sort.Strings(res)
	return res
}

// eventSet reads a root list of event names (_external_events,
// _failure_events). ok is false when it is present and not a list of strings.
func eventSet(m *machine.Machine, key string) (set map[string]bool, ok bool) {
	set = map[string]bool{}
	v := m.Root.AsObject().Get2(key)
	if v == nil {
		return set, true
	}
	if v.Kind != ir.KindArray {
		return set, false
	}
	ok = true
	for _, e := range v.AsArray() {
		if e == nil || e.Kind != ir.KindString {
			ok = false
			continue
		}
		set[e.AsString()] = true
	}
	return set, ok
}

type enumCand struct {
	name           string
	missing, extra []string
	overlap        int
	exact          bool
}

// checkStateEnum applies the state-enum rule to one lifecycle machine.
func checkStateEnum(fs *xfindings, d *design.Design, m *machine.Machine, e *design.Entity, domain []string) {
	_, model := design.FindEntity(d.Models, e.Name)
	want := map[string]bool{}
	for _, n := range domain {
		want[n] = true
	}
	var cands []enumCand
	seen := map[string]bool{}
	for _, at := range e.Attributes {
		if seen[at.Type] {
			continue
		}
		for _, en := range model.Enums {
			if en.Name != at.Type {
				continue
			}
			seen[at.Type] = true
			c := enumCand{name: en.Name}
			have := map[string]bool{}
			for _, v := range en.Values {
				have[v] = true
				if want[v] {
					c.overlap++
				} else {
					c.missing = append(c.missing, v)
				}
			}
			for _, n := range domain {
				if !have[n] {
					c.extra = append(c.extra, n)
				}
			}
			c.exact = len(c.missing) == 0 && len(c.extra) == 0
			cands = append(cands, c)
		}
	}
	var exact []string
	for _, c := range cands {
		if c.exact {
			exact = append(exact, c.name)
		}
	}
	switch {
	case len(exact) == 1:
		return
	case len(exact) > 1:
		sort.Strings(exact)
		fs.add("state-enum", m.ID, "more than one enum of %s has the domain states of the machine: %s", e.Name, strings.Join(exact, ", "))
	case len(cands) == 0:
		fs.add("state-enum", m.ID, "entity %s has no attribute whose type is an enum; the domain states (%s) need one", e.Name, strings.Join(domain, ", "))
	default:
		sort.SliceStable(cands, func(i, j int) bool {
			if cands[i].overlap != cands[j].overlap {
				return cands[i].overlap > cands[j].overlap
			}
			return cands[i].name < cands[j].name
		})
		c := cands[0]
		var parts []string
		if len(c.missing) > 0 {
			parts = append(parts, "missing from the machine: "+strings.Join(c.missing, ", "))
		}
		if len(c.extra) > 0 {
			parts = append(parts, "not in the enum: "+strings.Join(c.extra, ", "))
		}
		fs.add("state-enum", m.ID, "domain states differ from enum %s (%s)", c.name, strings.Join(parts, "; "))
	}
}
