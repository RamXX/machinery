package machine

import (
	"fmt"
	"regexp"

	"github.com/RamXX/machinery/internal/ir"
)

// reServesID is the form of a _serves entry: a model entity or a model action
// (docs/formats.md section 5.2).
var reServesID = regexp.MustCompile(`^(entity:[A-Za-z_][A-Za-z0-9_]*|act:[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_][A-Za-z0-9_]*)$`)

// Serves returns the root _serves of the machine: the model ids an
// operational machine exists for, the strings in declaration order. present
// is false when the key is absent.
func (m *Machine) Serves() (ids []string, present bool) {
	ro := m.Root.AsObject()
	if ro == nil {
		return nil, false
	}
	v := ro.Get2("_serves")
	if v == nil {
		return nil, false
	}
	for _, e := range v.AsArray() {
		if e != nil && e.Kind == ir.KindString {
			ids = append(ids, e.AsString())
		}
	}
	return ids, true
}

// lintServes holds the shape of the root _serves: a non-empty list of
// distinct entity:<Name> and act:<Entity>.<action> ids. Whether they are in
// the model is stage xref-model-machine's.
func lintServes(base string, ro *ir.Object) (errs []string) {
	v := ro.Get2("_serves")
	if v == nil {
		return nil
	}
	if v.Kind != ir.KindArray || len(v.AsArray()) == 0 {
		return []string{base + ": _serves must be a non-empty list of model ids (entity:<Name> or act:<Entity>.<action>)"}
	}
	seen := map[string]bool{}
	for _, e := range v.AsArray() {
		if e == nil || e.Kind != ir.KindString || !reServesID.MatchString(e.AsString()) {
			errs = append(errs, fmt.Sprintf("%s: _serves entry %s is not an entity:<Name> or act:<Entity>.<action> id", base, ir.Repr(e)))
			continue
		}
		if id := e.AsString(); seen[id] {
			errs = append(errs, fmt.Sprintf("%s: _serves names %s twice", base, ir.Repr(id)))
		} else {
			seen[id] = true
		}
	}
	return errs
}
