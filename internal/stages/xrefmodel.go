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

// bindingsOf returns the binding of every machine, in machine order, with
// the findings that need the other machines: an entity has one lifecycle
// machine, or several that each name a different _aspect (section 5.2).
func bindingsOf(d *design.Design) []binding {
	bs := make([]binding, len(d.Machines))
	count := map[string]int{}
	for i, m := range d.Machines {
		bs[i] = bindingOf(d, m)
		if bs[i].operational && m.Aspect != "" {
			bs[i].problems = append(bs[i].problems, "_aspect is set on an operational machine; an aspect is part of an entity's lifecycle: remove _aspect, or bind the machine with _lifecycle_of")
		}
		if e := bs[i].entity; e != nil {
			count[e.Name]++
		}
	}
	first := map[string]*machine.Machine{}
	aspects := map[string]map[string]*machine.Machine{}
	for i, m := range d.Machines {
		e := bs[i].entity
		if e == nil || count[e.Name] < 2 {
			continue
		}
		f := first[e.Name]
		if f == nil {
			first[e.Name] = m
			aspects[e.Name] = map[string]*machine.Machine{}
		}
		if m.Aspect == "" || (f != nil && f.Aspect == "") {
			if f != nil {
				bs[i].problems = append(bs[i].problems, fmt.Sprintf("a second lifecycle machine of %s, beside %s; several lifecycle machines of one entity are aspects of its lifecycle: set _aspect on each, a short name unique for %s", e.Name, f.ID, e.Name))
			}
			continue
		}
		if other := aspects[e.Name][m.Aspect]; other != nil {
			bs[i].problems = append(bs[i].problems, fmt.Sprintf("_aspect %s is also the aspect of %s; the aspects of %s need different names", m.Aspect, other.ID, e.Name))
			continue
		}
		aspects[e.Name][m.Aspect] = m
	}
	return bs
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
	// aliases holds one line per alias in effect (section 6.4)
	aliases []string
}

// lifecycleGroup is an entity with its lifecycle machines, in machine order.
type lifecycleGroup struct {
	entity   *design.Entity
	machines []*machine.Machine
}

// checkModelMachine applies the rules of section 6.4 to the design.
func checkModelMachine(d *design.Design) (fs xfindings, c xrefCounts) {
	c.machines = len(d.Machines)
	var groups []*lifecycleGroup
	byEntity := map[string]*lifecycleGroup{}
	bs := bindingsOf(d)
	for i, m := range d.Machines {
		b := bs[i]
		for _, p := range b.problems {
			fs.add(ruleMachineBinding, m.ID, "%s", p)
		}
		if b.operational {
			c.operational++
			if len(b.problems) == 0 {
				c.events += checkOperationalEvents(&fs, &c.aliases, d, m)
			}
			continue
		}
		e := b.entity
		if e == nil {
			continue
		}
		c.lifecycle++
		g := byEntity[e.Name]
		if g == nil {
			g = &lifecycleGroup{entity: e}
			byEntity[e.Name] = g
			groups = append(groups, g)
		}
		g.machines = append(g.machines, m)
	}
	for _, g := range groups {
		for _, m := range g.machines {
			c.states += len(domainStates(m))
		}
		checkStateEnum(&fs, d, g)
		c.events += checkLifecycleEvents(&fs, &c.aliases, g)
		c.actions += len(g.entity.Actions)
	}
	for _, md := range d.Models {
		for _, e := range md.Entities {
			if len(e.Actions) == 0 {
				continue
			}
			c.withActions++
			if byEntity[e.Name] == nil {
				fs.add(ruleEntityMachine, "entity:"+e.Name, "declares %d action(s) and no machine is its lifecycle; write its lifecycle machine (_lifecycle_of: %s), or waive the rule with the reason it has no state", len(e.Actions), e.Name)
			}
		}
	}
	return fs, c
}

// externalEvents returns the events a machine declares as arriving from
// outside: _external_events and _failure_events (a failure event arrives
// from outside). A list of the wrong shape is a finding of event-action,
// and so is a listed event that is an action (entityOf names the entity of
// every action the machine could be asked for): an action is a caller's
// command, and no list hides one. The finding says where the event is
// listed: the entry of import.machines.annotate that added it, or the file.
func externalEvents(fs *xfindings, m *machine.Machine, entityOf map[string]string) map[string]bool {
	external := map[string]bool{}
	for _, key := range []string{"_external_events", "_failure_events"} {
		set, ok := eventSet(m, key)
		if !ok {
			fs.add("event-action", m.ID, "%s must be a list of event names", key)
		}
		for _, ev := range sortedNames(set) {
			if external[ev] {
				continue // in both lists: stage machines reports it
			}
			external[ev] = true
			if ent, isAction := entityOf[ev]; isAction {
				where := fmt.Sprintf("remove it from %s in %s", key, m.Path)
				if by := m.Listed[key+" "+ev]; by != "" {
					where = by + "; narrow the pattern or remove it"
				}
				fs.add("event-action", m.ID+"."+ev, "listed in %s and an action of %s; an action is a caller's command, not an event from outside: %s", key, ent, where)
			}
		}
	}
	return external
}

func sortedNames(set map[string]bool) []string {
	res := make([]string, 0, len(set))
	for n := range set {
		res = append(res, n)
	}
	sort.Strings(res)
	return res
}

// aliasRules is what the aliases of a machine are held to (section 6.4).
type aliasRules struct {
	entityOf map[string]string // every action behind the machine, with its entity: none takes an alias
	isAction map[string]bool   // the actions an alias may name
	of       string            // what those are the actions of
	missing  string            // what to do about an action that is not one
	ownIn    map[string]string // an action that has an on event of its own name, with the machine that has it
}

// aliasedEvents returns the events of a machine that are an action under
// another name (_aliases), each with its action, for the ones that hold: the
// event is no action of any entity behind the machine, the action is one an
// alias may name, and that action has no event of its own name. The others
// are findings of event-action. Every alias that holds is a line of lines:
// none is silent.
func aliasedEvents(fs *xfindings, lines *[]string, m *machine.Machine, r aliasRules) (valid map[string]string, all map[string]bool) {
	valid, all = map[string]string{}, map[string]bool{}
	aliases := m.Aliases()
	evs := make([]string, 0, len(aliases))
	for ev := range aliases {
		evs = append(evs, ev)
	}
	sort.Strings(evs)
	for _, ev := range evs {
		act := aliases[ev]
		all[ev] = true
		where := "remove it from _aliases in " + m.Path
		if by := m.Listed["_aliases "+ev]; by != "" {
			where = by
		}
		ent, evIsAction := r.entityOf[ev]
		switch {
		case evIsAction:
			fs.add("event-action", m.ID+"."+ev, "_aliases gives the event the action %s, and %s is itself an action of %s; an action is not hidden behind another: %s", act, ev, ent, where)
		case !r.isAction[act]:
			fs.add("event-action", m.ID+"."+ev, "_aliases names action %s, which is not an action of %s; %s, or %s", act, r.of, r.missing, where)
		case r.ownIn[act] != "":
			fs.add("event-action", m.ID+"."+ev, "_aliases gives the event the action %s, and %s has an event of its own name in %s; an action has its own event or stands behind aliases, not both: %s", act, act, r.ownIn[act], where)
		default:
			valid[ev] = act
			*lines = append(*lines, fmt.Sprintf("alias: %s: %s stands for %s", m.ID, ev, act))
		}
	}
	return valid, all
}

// checkOperationalEvents applies event-action to an operational machine
// that binds to the model and returns how many events it has: each event is
// an action of an entity it serves, the action of an act id it serves, or
// declared external. A machine that serves only entities with no action
// compared nothing, and that is one finding, not one per event.
func checkOperationalEvents(fs *xfindings, aliasLines *[]string, d *design.Design, m *machine.Machine) int {
	serves, _ := m.Serves()
	allowed := map[string]bool{}
	// entityOf holds every action of every entity behind what the machine
	// serves, the entity of an act id too: none of them is external
	entityOf := map[string]string{}
	anyAction := false
	var bare []string
	for _, id := range serves {
		kind, name, _ := strings.Cut(id, ":")
		entName, _, _ := strings.Cut(name, ".")
		if e, _ := design.FindEntity(d.Models, entName); e != nil {
			for _, a := range e.Actions {
				if _, have := entityOf[a.Name]; !have {
					entityOf[a.Name] = e.Name
				}
			}
		}
		switch kind {
		case "entity":
			e, _ := design.FindEntity(d.Models, name)
			if e == nil {
				continue
			}
			if len(e.Actions) == 0 {
				bare = append(bare, e.Name)
			}
			for _, a := range e.Actions {
				allowed[a.Name] = true
				anyAction = true
			}
		case "act":
			_, act, _ := strings.Cut(name, ".")
			allowed[act] = true
			anyAction = true
		}
	}
	of := fmt.Sprintf("what the machine serves (%s)", strings.Join(serves, ", "))
	external := externalEvents(fs, m, entityOf)
	evs := m.Table.Events()
	ownIn := map[string]string{}
	for _, ev := range evs {
		ownIn[ev] = m.ID
	}
	_, aliased := aliasedEvents(fs, aliasLines, m, aliasRules{entityOf: entityOf, isAction: allowed, of: of, missing: "serve the action (act:<Entity>.<action>)", ownIn: ownIn})
	var unmatched []string
	for _, ev := range evs {
		if !allowed[ev] && !external[ev] && !aliased[ev] {
			unmatched = append(unmatched, ev)
		}
	}
	switch {
	case len(unmatched) == 0:
	case !anyAction:
		sort.Strings(bare)
		fs.add("event-action", m.ID, "serves only entities that declare no actions (%s), so none of its %d event(s) was compared with the model (%s); add the actions to the model, serve the actions the machine runs (act:<Entity>.<action>), or list the events in _external_events (or in _failure_events when one reports a failure)",
			strings.Join(bare, ", "), len(unmatched), strings.Join(unmatched, ", "))
	default:
		for _, ev := range unmatched {
			fs.add("event-action", m.ID+"."+ev, "event is not an action of what the machine serves (%s); serve the action (act:<Entity>.<action>), or list the event in _external_events (or in _failure_events when it reports a failure)", strings.Join(serves, ", "))
		}
	}
	return len(evs)
}

// checkLifecycleEvents applies event-action and action-event to the
// lifecycle machines of one entity and returns how many events they have.
// An entity that declares no action is one finding, not one per event: the
// one missing declaration is the cause of all of them.
func checkLifecycleEvents(fs *xfindings, aliasLines *[]string, g *lifecycleGroup) (events int) {
	e := g.entity
	acts := map[string]bool{}
	for _, a := range e.Actions {
		acts[a.Name] = true
	}
	entityOf := map[string]string{}
	for _, a := range e.Actions {
		entityOf[a.Name] = e.Name
	}
	evSet := map[string]bool{}
	var unmatched, ids []string
	seen := map[string]bool{}
	// an action with an on event of its own name, in any lifecycle machine
	// of the entity, with the first machine that has it
	ownIn := map[string]string{}
	for _, m := range g.machines {
		for _, ev := range m.Table.Events() {
			if ownIn[ev] == "" {
				ownIn[ev] = m.ID
			}
		}
	}
	for _, m := range g.machines {
		external := externalEvents(fs, m, entityOf)
		alias, aliased := aliasedEvents(fs, aliasLines, m, aliasRules{entityOf: entityOf, isAction: acts, of: e.Name, missing: "name an action of " + e.Name, ownIn: ownIn})
		evs := m.Table.Events()
		events += len(evs)
		ids = append(ids, m.ID)
		for _, ev := range evs {
			evSet[ev] = true
			if act, ok := alias[ev]; ok {
				evSet[act] = true // the event is that action under another name
			}
			if acts[ev] || external[ev] || aliased[ev] {
				continue
			}
			if len(e.Actions) > 0 {
				fs.add("event-action", m.ID+"."+ev, "event is not an action of %s; add the action to the model or list the event in _external_events (or in _failure_events when it reports a failure)", e.Name)
			} else if !seen[ev] {
				seen[ev] = true
				unmatched = append(unmatched, ev)
			}
		}
	}
	if len(unmatched) > 0 {
		sort.Strings(unmatched)
		fs.add("event-action", "entity:"+e.Name, "declares no actions, so none of the %d event(s) of %s is one of its actions (%s); add the actions to the model, or list the events in _external_events (or in _failure_events when one reports a failure)",
			len(unmatched), strings.Join(ids, ", "), strings.Join(unmatched, ", "))
	}
	for _, a := range e.Actions {
		if evSet[a.Name] {
			continue
		}
		if len(g.machines) == 1 {
			m := g.machines[0]
			fs.add("action-event", design.ActID(e.Name, a.Name), "no on event %s in machine %s (%s)", a.Name, m.ID, m.Path)
			continue
		}
		var where []string
		for _, m := range g.machines {
			where = append(where, fmt.Sprintf("%s (%s)", m.ID, m.Path))
		}
		fs.add("action-event", design.ActID(e.Name, a.Name), "no on event %s in any lifecycle machine of %s: %s", a.Name, e.Name, strings.Join(where, ", "))
	}
	return events
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
	for _, line := range c.aliases {
		s.Warnf("%s", line)
	}
	s.Count(c.machines, "machines").Count(c.lifecycle, "lifecycle machines").Count(c.operational, "operational machines").
		Count(c.states, "domain states").Count(c.events, "events").Count(c.actions, "actions").Count(c.withActions, "entities with actions")
	if len(c.aliases) > 0 {
		s.Count(len(c.aliases), "aliases")
	}
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
	for _, b := range bindingsOf(ctx.Design()) {
		if b.bound() {
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

// enumCands compares the domain states of a machine with every enum used as
// the type of an attribute of its entity, in attribute order.
func enumCands(model *design.Model, e *design.Entity, domain []string) []enumCand {
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
	return cands
}

// checkStateEnum applies the state-enum rule to the lifecycle machines of
// one entity. An entity with no enum-typed attribute is one finding, not one
// per machine. Aspect machines each have an enum of their own: an enum whose
// values are the domain states of one machine is claimed by it, and no enum
// is claimed twice.
func checkStateEnum(fs *xfindings, d *design.Design, g *lifecycleGroup) {
	e := g.entity
	_, model := design.FindEntity(d.Models, e.Name)
	if len(enumCands(model, e, nil)) == 0 {
		var parts []string
		for _, m := range g.machines {
			parts = append(parts, fmt.Sprintf("%s (%s)", m.ID, strings.Join(domainStates(m), ", ")))
		}
		need := "need one"
		if len(parts) > 1 {
			need = "need one each"
		}
		fs.add("state-enum", "entity:"+e.Name, "has no state enum (no attribute of %s has an enum type); the domain states of %s %s", e.Name, strings.Join(parts, " and of "), need)
		return
	}
	cands := make([][]enumCand, len(g.machines))
	exact := make([][]string, len(g.machines))
	for i, m := range g.machines {
		cands[i] = enumCands(model, e, domainStates(m))
		for _, c := range cands[i] {
			if c.exact {
				exact[i] = append(exact[i], c.name)
			}
		}
		sort.Strings(exact[i])
	}
	claimed := map[string]*machine.Machine{}
	done := make([]bool, len(g.machines))
	// a machine with exactly one exact enum claims it first
	for i, m := range g.machines {
		if len(exact[i]) != 1 {
			continue
		}
		done[i] = true
		en := exact[i][0]
		if other := claimed[en]; other != nil {
			fs.add("state-enum", m.ID, "enum %s is already the state enum of %s; each aspect of %s needs an enum attribute of its own", en, other.ID, e.Name)
			continue
		}
		claimed[en] = m
	}
	for i, m := range g.machines {
		if done[i] || len(exact[i]) == 0 {
			continue
		}
		done[i] = true
		var free []string
		for _, en := range exact[i] {
			if claimed[en] == nil {
				free = append(free, en)
			}
		}
		switch len(free) {
		case 1:
			claimed[free[0]] = m
		case 0:
			fs.add("state-enum", m.ID, "enum %s is already the state enum of %s; each aspect of %s needs an enum attribute of its own", exact[i][0], claimed[exact[i][0]].ID, e.Name)
		default:
			fs.add("state-enum", m.ID, "more than one enum of %s has the domain states of the machine: %s", e.Name, strings.Join(free, ", "))
		}
	}
	for i, m := range g.machines {
		if done[i] {
			continue
		}
		domain := domainStates(m)
		var open []enumCand
		for _, c := range cands[i] {
			if claimed[c.name] == nil {
				open = append(open, c)
			}
		}
		if len(open) == 0 {
			fs.add("state-enum", m.ID, "entity %s has no state enum for this machine: every enum of its attributes is the state enum of another aspect; add an enum attribute with the domain states (%s)", e.Name, strings.Join(domain, ", "))
			continue
		}
		sort.SliceStable(open, func(i, j int) bool {
			if open[i].overlap != open[j].overlap {
				return open[i].overlap > open[j].overlap
			}
			return open[i].name < open[j].name
		})
		c := open[0]
		if c.overlap == 0 {
			var names []string
			for _, cd := range open {
				names = append(names, cd.name)
			}
			sort.Strings(names)
			fs.add("state-enum", m.ID, "entity %s has no state enum for this machine: no enum of its attributes (%s) has a value among the domain states (%s); add an enum attribute with these values", e.Name, strings.Join(names, ", "), strings.Join(domain, ", "))
			continue
		}
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
