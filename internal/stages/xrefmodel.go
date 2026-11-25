package stages

import (
	"sort"
	"strings"
	"unicode"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/machine"
	"github.com/RamXX/machinery/internal/report"
)

func init() { Bind("xref-model-machine", XrefModelMachine) }

var xrefModelRules = []string{"machine-binding", "state-enum", "event-action", "action-event"}

// XrefModelMachine is stage xref-model-machine (section 6.4).
func XrefModelMachine(ctx *Context) *report.Stage {
	s := report.New("xref-model-machine")
	p := ctx.P
	if p.Cfg.Design.Machines == "" {
		return s.NotApplicable("no design.machines configured")
	}
	if len(p.Cfg.Design.Models) == 0 {
		return s.NotApplicable("no design.models configured")
	}
	d := ctx.Design()
	w := waiversFor(p, xrefModelRules...)
	var states, events, actions int
	for _, m := range d.Machines {
		ent := d.LifecycleEntity(m)
		if m.LifecycleOf != "" && m.Role == "operational" {
			w.find(s, "machine-binding", m.ID, "_lifecycle_of and _role: operational are both set; a machine is one or the other")
			continue
		}
		if m.Role == "operational" {
			continue
		}
		if ent == "" {
			w.find(s, "machine-binding", m.ID, "not a lifecycle machine of a model entity and not operational; set _lifecycle_of, name the file after an entity, or set _role: operational")
			continue
		}
		e, _ := design.FindEntity(d.Models, ent)
		if e == nil {
			w.find(s, "machine-binding", m.ID, "_lifecycle_of names entity %s, which is not in the model", ent)
			continue
		}
		domain := domainStates(m)
		states += len(domain)
		checkStateEnum(s, w, d, m, e, domain)

		external, ok := eventSet(m, "_external_events")
		if !ok {
			w.find(s, "event-action", m.ID, "_external_events must be a list of event names")
		}
		// a failure event arrives from outside: it is external
		failure, ok := eventSet(m, "_failure_events")
		if !ok {
			w.find(s, "event-action", m.ID, "_failure_events must be a list of event names")
		}
		for ev := range failure {
			external[ev] = true
		}
		acts := map[string]bool{}
		for _, a := range e.Actions {
			acts[a.Name] = true
		}
		evs := m.Table.Events()
		events += len(evs)
		actions += len(e.Actions)
		evSet := map[string]bool{}
		for _, ev := range evs {
			evSet[ev] = true
			if !acts[ev] && !external[ev] {
				w.find(s, "event-action", m.ID+"."+ev, "event is not an action of %s; add the action to the model or list the event in _external_events (or in _failure_events when it reports a failure)", ent)
			}
		}
		for _, a := range e.Actions {
			if !evSet[a.Name] {
				w.find(s, "action-event", design.ActID(ent, a.Name), "no on event %s in machine %s (%s)", a.Name, m.ID, m.Path)
			}
		}
	}
	w.stale(s)
	s.Count(len(d.Machines), "machines").Count(states, "domain states").Count(events, "events").Count(actions, "actions")
	if len(d.Machines) == 0 && len(d.MachineProblems) == 0 {
		s.NotDemonstrated("nothing to check")
	}
	return s
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
func checkStateEnum(s *report.Stage, w *waivers, d *design.Design, m *machine.Machine, e *design.Entity, domain []string) {
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
		w.find(s, "state-enum", m.ID, "more than one enum of %s has the domain states of the machine: %s", e.Name, strings.Join(exact, ", "))
	case len(cands) == 0:
		w.find(s, "state-enum", m.ID, "entity %s has no attribute whose type is an enum; the domain states (%s) need one", e.Name, strings.Join(domain, ", "))
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
		w.find(s, "state-enum", m.ID, "domain states differ from enum %s (%s)", c.name, strings.Join(parts, "; "))
	}
}
