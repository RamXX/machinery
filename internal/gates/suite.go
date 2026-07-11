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

// knownGateSet is the full gate vocabulary. Select and the hook-config
// validator (internal/hook) must agree on it, so both read this set through
// KnownGate; two hand-kept lists once drifted.
var knownGateSet = map[string]bool{
	"gm": true, "gs": true, "gp": true, "gi": true, "gn": true, "g2": true,
	"g3": true, "gx": true, "gb": true, "g4": true, "gt": true, "g5": true,
}

// KnownGate reports whether name names a gate this suite can run.
func KnownGate(name string) bool { return knownGateSet[name] }

// Select resolves a --gate list (or, when gateList is empty, the default
// suite) for design: the default narrows to g2,g5 (plus the artifact-activated
// gates whose sources exist) on a decomposed parent with no machines/
// directory, and an unknown gate name is an error.
func Select(design, gateList string) (Selection, error) {
	sel := Selection{Run: map[string]bool{}, Explicit: gateList != ""}
	list := "gm,gs,gp,gi,gn,g2,g3,gx,gb,g4,gt,g5"
	if !sel.Explicit && pack.HasDecomposition(design) {
		if ms, _ := filepath.Glob(filepath.Join(design, "machines", "*.machine.json")); len(ms) == 0 {
			// a pure decomposed parent authors no machines: its behavior
			// layer is the children's, held by the packs; only the
			// machine-dependent gates (G3, Gx, G4, Gt) narrow away. Every
			// artifact-activated gate keeps its auto-activation: v0.3.0
			// narrowed gp/gi/gn away too, silently skipping the relational
			// layers on a decomposed parent that carried them. Gb stays too:
			// the parent's manifest BUILD.md is still its artifact and its
			// plan shape is still checkable. Machine-less means no
			// *.machine.json, not no directory: an empty machines/ dir once
			// defeated this narrowing and failed a decomposed parent on
			// G3/Gx (the H2 dogfood finding). The note lists what actually
			// runs; the golden corpus pins its text byte for byte.
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
			parts = append(parts, "g2")
			if HasBuildDoc(design) {
				parts = append(parts, "gb")
			}
			parts = append(parts, "g5")
			list = strings.Join(parts, ",")
			sel.Note = "note: decomposed parent with no machines/; running " + list + " (G3/Gx/G4/Gt run on the child designs)"
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
		if !KnownGate(g) {
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
// G2, G3, Gx, Gb, G4, Gt, G5) with `machinery check`'s applicability rules:
// opt-in gates run only when their source exists (or when explicitly
// requested), G4 and Gt only with an impl dir, and G5 only when explicitly
// requested or when the design is decomposed. The returned gates carry their
// findings; the caller emits them.
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
	if sel.Run["gb"] && (sel.Explicit || HasBuildDoc(design)) {
		out = append(out, CheckBuildPlan(design))
	}
	if sel.Run["g4"] && impl != "" {
		out = append(out, CheckImports(design, impl))
	}
	if sel.Run["gt"] && impl != "" {
		out = append(out, CheckOracleCoverage(design, impl))
	}
	if sel.Run["g5"] && (sel.Explicit || pack.HasDecomposition(design) || pack.HasPack(design)) {
		out = append(out, CheckPack(design))
	}
	return out
}
