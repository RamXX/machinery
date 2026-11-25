package machine

import (
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// lintTerminal holds the shape and the placement of a state's _terminal: a
// non-empty reason string on a non-final leaf state. Whether the state is
// left at all is the dead-end rule's question (Lint).
func lintTerminal(base, p string, o *ir.Object, isFinal bool) (errs []string) {
	v, ok := o.Get("_terminal")
	if !ok {
		return nil
	}
	if v == nil || v.Kind != ir.KindString || strings.TrimSpace(v.AsString()) == "" {
		errs = append(errs, base+": state "+p+" _terminal must be a non-empty reason string saying why the state is never left")
	}
	switch {
	case isFinal:
		errs = append(errs, base+": final state "+p+" declares _terminal; a final state is terminal already (drop the annotation)")
	case o.Get2("states") != nil:
		errs = append(errs, base+": compound state "+p+" declares _terminal; _terminal belongs on a leaf state")
	}
	return errs
}
