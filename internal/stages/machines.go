package stages

import (
	"sort"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/machine"
	"github.com/RamXX/machinery/internal/report"
)

func init() { Bind("machines", Machines) }

// Machines is stage machines (section 6.2).
func Machines(ctx *Context) *report.Stage {
	s := report.New("machines")
	if !ctx.P.Cfg.HasMachines() {
		return s.NotApplicable("no design.machines configured")
	}
	d := ctx.Design()
	for _, prob := range d.MachineProblems {
		s.Errorf("%s", prob)
	}
	states, transitions := 0, 0
	for _, m := range d.Machines {
		errs, counts := m.Lint()
		for _, e := range errs {
			s.Errorf("%s", e)
		}
		for _, h := range m.Hints(lifecycleActions(d, m)) {
			s.Warnf("%s", h)
		}
		states += counts.States
		transitions += counts.Transitions
	}
	collisions(s, d.Machines)
	s.Count(len(d.Machines), "machines").Count(states, "states").Count(transitions, "transitions")
	if entries, observed, on := checkRegister(s, d); on {
		s.Count(entries, "register entries").Count(observed, "observed transitions")
	}
	if len(d.Machines) == 0 && len(d.MachineProblems) == 0 {
		s.NotDemonstrated("nothing to check")
	}
	return s
}

// lifecycleActions returns the action names of the entity a machine is the
// lifecycle of (none for an operational machine), for the name hints.
func lifecycleActions(d *design.Design, m *machine.Machine) map[string]bool {
	res := map[string]bool{}
	if e, _ := design.FindEntity(d.Models, d.LifecycleEntity(m)); e != nil {
		for _, a := range e.Actions {
			res[a.Name] = true
		}
	}
	return res
}

// collisions reports machines that share a case-id tag or a machine id: their
// generated ids or table files would overwrite each other.
func collisions(s *report.Stage, ms []*machine.Machine) {
	byTag, byID := map[string][]*machine.Machine{}, map[string][]*machine.Machine{}
	for _, m := range ms {
		byTag[m.Tag] = append(byTag[m.Tag], m)
		byID[m.ID] = append(byID[m.ID], m)
	}
	report1 := func(group []*machine.Machine, format string, key string) {
		if len(group) < 2 {
			return
		}
		sort.Slice(group, func(i, j int) bool { return group[i].Path < group[j].Path })
		others := ""
		for i, o := range group[1:] {
			if i > 0 {
				others += ", "
			}
			others += o.Path
		}
		s.Errorf(format, group[0].Path, key, others)
	}
	for tag, g := range byTag {
		report1(g, "%s: case-id tag '%s' is also used by %s; set _oracle_tag on one of the machines", tag)
	}
	for id, g := range byID {
		report1(g, "%s: machine id '%s' is also declared by %s; machine ids must be unique", id)
	}
}
