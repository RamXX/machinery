package stages

import (
	"strings"

	"github.com/RamXX/machinery/v2/internal/design"
	"github.com/RamXX/machinery/v2/internal/dialect"
	"github.com/RamXX/machinery/v2/internal/project"
	"github.com/RamXX/machinery/v2/internal/report"
)

// failureAccount is the one account of the failure modes (section 6.6):
// which are linked, which are subject-less, which are unaccounted for
// (unlinked), and of those which name a subject and no required value. It is
// computed once per command; stages failure-modes and xref-failure print it
// with count, and xref-failure reports the subjects that name nothing.
type failureAccount struct {
	linked, unlinked, subjectLess int
	// noRequired counts the unlinked ones whose subject resolves and that
	// only lack a valid required value.
	noRequired int
	// unlinkedRows counts the imported rows among the unlinked.
	unlinkedRows int
	// named holds the subjects of the linked failure modes, abused those of
	// kind abuse.
	named, abused map[string]bool
	// what a subject may name: invoke:<machine id>.<src>, dependency:<name>,
	// the failure states and the failure events of the machines
	invokes, deps, fstates, fevents map[string]bool
	// unresolved lists the subjects that name nothing, in the order of the
	// failure modes, whatever their required value.
	unresolved []unresolvedSubject
}

// unresolvedSubject is a subject of one of the forms of section 5.4 that
// names nothing in the design. system is set for an unknown system, which
// no waiver covers.
type unresolvedSubject struct {
	fm, subject, why string
	system           bool
}

// failureAccount returns the account of the failure modes of the design,
// computed on first use.
func (c *Context) failureAccount() *failureAccount {
	if c.fa == nil {
		c.fa = accountFailures(c.P, c.Design())
	}
	return c.fa
}

func accountFailures(p *project.Project, d *design.Design) *failureAccount {
	a := &failureAccount{named: map[string]bool{}, abused: map[string]bool{}, invokes: map[string]bool{}, deps: map[string]bool{}}
	a.fstates, a.fevents = failureSubjects(d)
	for _, m := range d.Machines {
		for _, st := range m.Table.States {
			for _, src := range st.Invoke {
				a.invokes["invoke:"+m.ID+"."+src] = true
			}
		}
	}
	for _, dep := range p.Cfg.Dependencies {
		a.deps["dependency:"+dep.Name] = true
	}
	systems := map[string]bool{}
	for _, sy := range p.Cfg.Systems {
		systems["system:"+sy.Name] = true
	}
	for _, fm := range d.FailureModes {
		if fm.SubjectLess() {
			a.subjectLess++
			continue // names nothing, and says why (section 5.4)
		}
		// a subject of a known form is looked up whatever its required
		// value: one that names nothing is a finding either way
		resolves := fm.NamesSubject()
		switch {
		case !resolves:
			// stage failure-modes has the finding, when there is one
		case strings.HasPrefix(fm.Subject, "system:"):
			if !systems[fm.Subject] {
				resolves = false
				a.unresolved = append(a.unresolved, unresolvedSubject{fm: fm.ID, subject: fm.Subject, system: true,
					why: strings.TrimPrefix(fm.Subject, "system:") + " is not a configured system"})
			}
		default:
			if why := unresolved(d, fm.Subject, a.invokes, a.deps, a.fstates, a.fevents); why != "" {
				// waived or not, the subject names nothing: the row is not linked
				resolves = false
				a.unresolved = append(a.unresolved, unresolvedSubject{fm: fm.ID, subject: fm.Subject, why: why})
			}
		}
		if resolves && !dialect.ValidRequired(fm.Required) {
			// it names something of the design and does not say what the
			// system must do: counted on its own, and still not linked
			a.noRequired++
			resolves = false
		}
		if !resolves {
			a.unlinked++
			if fm.Imported {
				a.unlinkedRows++
			}
			continue
		}
		a.linked++
		a.named[fm.Subject] = true
		if fm.Kind == "abuse" {
			a.abused[fm.Subject] = true
		}
	}
	return a
}

// count adds the account to a checked: line, the same way for both stages:
// linked and unlinked when a catalog is imported or a failure mode is
// unaccounted for, then those that name a subject and no required value and
// the subject-less ones, each when there is one.
func (a *failureAccount) count(s *report.Stage, imports bool) {
	if imports || a.unlinked > 0 {
		s.Count(a.linked, "linked").Count(a.unlinked, "unlinked")
	}
	if a.noRequired > 0 {
		s.Count(a.noRequired, "name a subject and no required value")
	}
	if a.subjectLess > 0 {
		s.Count(a.subjectLess, "subject-less")
	}
}
