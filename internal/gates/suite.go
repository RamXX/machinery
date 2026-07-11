// Gate-suite selection and execution shared by `machinery check` and the
// Claude Code hook handler (`machinery hook`), so both run the exact same
// suite semantics: one implementation of which gates apply to a design.

package gates

import (
	"fmt"
	"path/filepath"
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

// Select resolves a --gate list (or, when gateList is empty, the default
// suite) for design: the default narrows to g2,g5 (plus gm/gs when present)
// on a decomposed parent with no machines/ directory, and an unknown gate
// name is an error.
func Select(design, gateList string) (Selection, error) {
	sel := Selection{Run: map[string]bool{}, Explicit: gateList != ""}
	list := "gm,gs,gp,gi,gn,g2,g3,gx,g4,g5"
	if !sel.Explicit && pack.HasDecomposition(design) {
		if ms, _ := filepath.Glob(filepath.Join(design, "machines", "*.machine.json")); len(ms) == 0 {
			// a pure decomposed parent authors no machines: its behavior
			// layer is the children's, held by the packs; only the
			// machine-dependent gates (G3, Gx, G4) narrow away. Every
			// artifact-activated gate keeps its auto-activation: v0.3.0
			// narrowed gp/gi/gn away too, silently skipping the relational
			// layers on a decomposed parent that carried them. Machine-less
			// means no *.machine.json, not no directory: an empty machines/
			// dir once defeated this narrowing and failed a decomposed
			// parent on G3/Gx (the H2 dogfood finding). The note lists what
			// actually runs; the golden corpus pins its text byte for byte.
			var parts []string
			for _, opt := range []struct {
				gate string
				has  func(string) bool
			}{
				{"gm", HasMigrationContract},
				{"gs", HasSurfaceLedger},
				{"gp", HasPolicyAnnotation},
				{"gi", HasIntegrityAnnotation},
				{"gn", HasIsolationAnnotation},
			} {
				if opt.has(design) {
					parts = append(parts, opt.gate)
				}
			}
			parts = append(parts, "g2", "g5")
			list = strings.Join(parts, ",")
			sel.Note = "note: decomposed parent with no machines/; running " + list + " (G3/Gx/G4 run on the child designs)"
		}
	}
	if sel.Explicit {
		list = gateList
	}
	for _, g := range strings.Split(strings.ToLower(list), ",") {
		sel.Run[strings.TrimSpace(g)] = true
	}
	var unknown []string
	for g := range sel.Run {
		if g != "gm" && g != "gs" && g != "gp" && g != "gi" && g != "gn" && g != "g2" && g != "g3" && g != "gx" && g != "g4" && g != "g5" {
			unknown = append(unknown, g)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return sel, fmt.Errorf("unknown gate(s): %s", strings.Join(unknown, ", "))
	}
	return sel, nil
}

// RunSelected runs the selected gates in canonical order (Gm, Gs, Gp, Gi, Gn,
// G2, G3, Gx, G4, G5) with `machinery check`'s applicability rules: opt-in
// gates run only when their source exists (or when explicitly requested), G4
// only with an impl dir, and G5 only when explicitly requested or when the
// design is decomposed. The returned gates carry their findings; the caller
// emits them.
func RunSelected(design, impl string, sel Selection) []*Gate {
	var out []*Gate
	if sel.Run["gm"] && (sel.Explicit || HasMigrationContract(design)) {
		out = append(out, CheckMigration(design))
	}
	if sel.Run["gs"] && (sel.Explicit || HasSurfaceLedger(design)) {
		out = append(out, CheckSurface(design))
	}
	if sel.Run["gp"] && (sel.Explicit || HasPolicyAnnotation(design)) {
		out = append(out, CheckPolicy(design))
	}
	if sel.Run["gi"] && (sel.Explicit || HasIntegrityAnnotation(design)) {
		out = append(out, CheckIntegrity(design))
	}
	if sel.Run["gn"] && (sel.Explicit || HasIsolationAnnotation(design)) {
		out = append(out, CheckIsolation(design))
	}
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
