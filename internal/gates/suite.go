// Gate-suite selection and execution shared by `machinery check` and the
// Claude Code hook handler (`machinery hook`), so both run the exact same
// suite semantics: one implementation of which gates apply to a design.

package gates

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/pack"
)

// Selection is a resolved gate list for one suite run.
type Selection struct {
	Run      map[string]bool
	Explicit bool   // the list was caller-supplied rather than the default
	Note     string // decomposed-parent narrowing note (default selection only)
}

// decomposedParentNote matches the historical `machinery check` output byte
// for byte; the golden corpus pins it.
const decomposedParentNote = "note: decomposed parent with no machines/; running g2,g5 (G3/Gx/G4 run on the child designs)"

// Select resolves a --gate list (or, when gateList is empty, the default
// suite) for design: the default narrows to g2,g5 on a decomposed parent
// with no machines/ directory, and an unknown gate name is an error.
func Select(design, gateList string) (Selection, error) {
	sel := Selection{Run: map[string]bool{}, Explicit: gateList != ""}
	gs := "g2,g3,gx,g4,g5"
	if !sel.Explicit && pack.HasDecomposition(design) {
		if fi, err := os.Stat(design + "/machines"); err != nil || !fi.IsDir() {
			// a pure decomposed parent authors no machines: its behavior
			// layer is the children's, held by the packs; G3/Gx run there
			gs = "g2,g5"
			sel.Note = decomposedParentNote
		}
	}
	if sel.Explicit {
		gs = gateList
	}
	for _, g := range strings.Split(strings.ToLower(gs), ",") {
		sel.Run[strings.TrimSpace(g)] = true
	}
	var unknown []string
	for g := range sel.Run {
		if g != "g2" && g != "g3" && g != "gx" && g != "g4" && g != "g5" {
			unknown = append(unknown, g)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return sel, fmt.Errorf("unknown gate(s): %s", strings.Join(unknown, ", "))
	}
	return sel, nil
}

// RunSelected runs the selected gates in canonical order (G2, G3, Gx, G4,
// G5) with `machinery check`'s applicability rules: G4 only with an impl
// dir, and G5 only when explicitly requested or when the design is
// decomposed, so a plain design never runs it by accident. The returned
// gates carry their findings; the caller emits them.
func RunSelected(design, impl string, sel Selection) []*Gate {
	var out []*Gate
	if sel.Run["g2"] {
		out = append(out, CheckC4(design))
	}
	if sel.Run["g3"] {
		out = append(out, CheckMachines(design))
	}
	if sel.Run["gx"] {
		out = append(out, CheckTraceability(design))
	}
	if sel.Run["g4"] && impl != "" {
		out = append(out, CheckImports(design, impl))
	}
	if sel.Run["g5"] && (sel.Explicit || pack.HasDecomposition(design) || pack.HasPack(design)) {
		out = append(out, CheckPack(design))
	}
	return out
}
