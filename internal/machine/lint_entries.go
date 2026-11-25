package machine

import (
	"fmt"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// Entries returns the entry states of the machine (root _entries): the
// top-level states, besides initial, in which an instance may be created
// directly. Only names of existing top-level states other than initial are
// returned, in declaration order and without duplicates; lintEntries reports
// the rest.
func (m *Machine) Entries() []string {
	ro := m.Root.AsObject()
	if ro == nil {
		return nil
	}
	top := ro.GetObject("states")
	init := ro.GetString("initial")
	var res []string
	seen := map[string]bool{}
	for _, e := range ro.Get2("_entries").AsArray() {
		name := e.AsString()
		if name == "" || name == init || seen[name] || top == nil || !top.Has(name) {
			continue
		}
		seen[name] = true
		res = append(res, name)
	}
	return res
}

// lintEntries holds the root _entries annotation: a non-empty array naming
// existing top-level states other than initial, each once.
func lintEntries(base string, ro *ir.Object) (errs []string) {
	v := ro.Get2("_entries")
	if v == nil {
		return nil
	}
	if v.Kind != ir.KindArray || len(v.AsArray()) == 0 {
		return []string{base + ": _entries must be a non-empty array of top-level state names in which an instance may be created directly"}
	}
	top := ro.GetObject("states")
	init := ro.GetString("initial")
	seen := map[string]bool{}
	for _, e := range v.AsArray() {
		if e == nil || e.Kind != ir.KindString || strings.TrimSpace(e.AsString()) == "" {
			errs = append(errs, base+": _entries items must be non-empty state-name strings")
			continue
		}
		name := e.AsString()
		switch {
		case seen[name]:
			errs = append(errs, fmt.Sprintf("%s: _entries names %s twice", base, ir.Repr(name)))
		case name == init:
			errs = append(errs, fmt.Sprintf("%s: _entries names %s, the initial state; list only the other states an instance may be created in", base, ir.Repr(name)))
		case top == nil || !top.Has(name):
			errs = append(errs, fmt.Sprintf("%s: _entries names %s, which is not a top-level state", base, ir.Repr(name)))
		}
		seen[name] = true
	}
	return errs
}
