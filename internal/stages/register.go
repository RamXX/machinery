package stages

import (
	"strings"

	"github.com/RamXX/machinery/v2/internal/design"
	"github.com/RamXX/machinery/v2/internal/dialect"
	"github.com/RamXX/machinery/v2/internal/machine"
	"github.com/RamXX/machinery/v2/internal/report"
)

// The rules of a register of observed transitions (section 5.2.1).
const (
	ruleRegisterUndeclared = "register-undeclared"
	ruleRegisterUnobserved = "register-unobserved"
)

// checkRegister holds the imported machines to the register of
// import.machines.register: every observed transition is declared, and every
// declared transition is observed. It returns the entries and the observed
// transitions it read; on is false when no register is configured.
func checkRegister(s *report.Stage, d *design.Design) (entries, observed int, on bool) {
	im := d.P.Cfg.ImportedMachines()
	es, problems, on := dialect.Register(d.P)
	if !on {
		return 0, 0, false
	}
	file := im.Register.File
	for _, pr := range problems {
		s.Errorf("%s", pr)
	}
	byName := map[string]*machine.Machine{}
	for _, m := range d.Machines {
		byName[m.ID] = m
		if m.Original != "" {
			byName[m.Original] = m
		}
	}
	entryOf := map[*machine.Machine]int{}
	seen := map[*machine.Machine]map[int]bool{} // the table rows observed
	for _, e := range es {
		entries++
		observed += len(e.Observed)
		m := byName[e.Machine]
		switch {
		case m == nil:
			s.Errorf("%s: entry %d: %s %s names no imported machine; add its file to import.machines.files, or correct the entry", file, e.N, im.Register.Machine, e.Machine)
			continue
		case entryOf[m] != 0:
			s.Errorf("%s: entry %d: machine %s is also named by entry %d; keep one", file, e.N, e.Machine, entryOf[m])
			continue
		}
		entryOf[m] = e.N
		seen[m] = map[int]bool{}
		for _, o := range e.Observed {
			from, to := statePath(im.States, o.From), statePath(im.States, o.To)
			matched := false
			for i, tr := range m.Table.Transitions {
				if tr.Kind == "on" && tr.Event == o.Event && observes(m, tr, from, to) {
					seen[m][i] = true
					matched = true
				}
			}
			if !matched {
				s.Errorf("%s: %s: %s observes %s -> %s on %s, which the machine does not declare; declare it in %s, or correct the register",
					ruleRegisterUndeclared, m.ID, file, o.From, o.To, o.Event, m.Path)
			}
		}
	}
	for _, m := range d.Machines {
		var rows []machine.Transition
		for _, tr := range m.Table.Transitions {
			if tr.Kind == "on" {
				rows = append(rows, tr)
			}
		}
		if seen[m] == nil {
			if len(rows) > 0 {
				s.Errorf("%s: %s: no entry of %s names the machine, so none of its %d transition(s) is observed; add an entry for it (%s: %s)",
					ruleRegisterUnobserved, m.ID, file, len(rows), im.Register.Machine, m.Original)
			}
			continue
		}
		for i, tr := range m.Table.Transitions {
			if tr.Kind != "on" || seen[m][i] {
				continue
			}
			target := tr.Target
			if target == "" {
				target = tr.Source + " (internal)"
			}
			guard := ""
			if tr.Guard != "" {
				guard = " [" + tr.Guard + "]"
			}
			s.Errorf("%s: %s: %s -> %s on %s%s is declared and not observed in %s; add it to the register, or remove the transition from %s",
				ruleRegisterUnobserved, m.ID, tr.Source, target, tr.Event, guard, file, m.Path)
		}
	}
	return entries, observed, true
}

// statePath writes a state path of the register as the machine names it:
// each segment with the states setting of import.machines.
func statePath(mode, p string) string {
	segs := strings.Split(p, ".")
	for i := range segs {
		segs[i] = dialect.StateName(mode, segs[i])
	}
	return strings.Join(segs, ".")
}

// observes reports whether an observed from and to match a table row: the
// row is defined on from or on an ancestor of it, and it enters to (its
// target, or the leaf its target enters; from itself for an internal row).
func observes(m *machine.Machine, tr machine.Transition, from, to string) bool {
	if tr.Source != from && !machine.IsDescendant(from, tr.Source) {
		return false
	}
	if tr.Target == "" {
		return to == from
	}
	return to == tr.Target || to == m.Resolver.Leaf(tr.Target)
}
