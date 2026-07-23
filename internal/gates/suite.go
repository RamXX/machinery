// Gate-suite selection and execution shared by `machinery check` and the
// Claude Code hook handler (`machinery hook`), so both run the exact same
// suite semantics: one implementation of which gates apply to a design.

package gates

import (
	"fmt"
	"os"
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
	"g3": true, "gx": true, "gk": true, "gb": true, "g4": true, "gt": true, "g5": true,
}

// KnownGate reports whether name names a gate this suite can run.
func KnownGate(name string) bool { return knownGateSet[name] }

// HasMachines reports whether design/machines holds any *.machine.json. It
// lists the directory (sortedGlob) rather than filepath.Glob: a design PATH
// containing glob metacharacters ([ ] * ?) must never defeat machine
// detection, which once silently dropped gates (GATE-2).
func HasMachines(design string) bool {
	return len(sortedGlob(filepath.Join(design, "machines"), "*.machine.json")) > 0
}

// HasModelith reports whether the design carries a *.modelith.yaml domain
// model (the Gx source artifact; the hook's stop-time selection keys on it).
func HasModelith(design string) bool {
	entries, _ := os.ReadDir(design)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".modelith.yaml") {
			return true
		}
	}
	return false
}

// Select resolves a --gate list (or, when gateList is empty, the default
// suite) for design. impl is the implementation directory ("" when none was
// supplied); it decides whether the machine-less-parent narrowing keeps G4.
// The default narrows on a decomposed parent with no machines/, and an
// unknown or empty gate name is an error.
func Select(design, gateList, impl string) (Selection, error) {
	sel := Selection{Run: map[string]bool{}, Explicit: gateList != ""}
	list := "gm,gs,gp,gi,gn,g2,g3,gx,gk,gb,g4,gt,g5"
	if !sel.Explicit && pack.HasDecomposition(design) {
		if !HasMachines(design) {
			// a pure decomposed parent authors no machines: its behavior
			// layer is the children's, held by the packs; only the
			// machine-dependent gates (G3, Gx, Gt) narrow away. G4 is NOT
			// machine-dependent (the contract and the code suffice), so an
			// explicit --impl keeps it: v0.3.x silently dropped G4 here and
			// exited 0 over contract-DENIED edges (GATE-1). Every
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
			if HasCheckers(design) {
				parts = append(parts, "gk")
			}
			if HasBuildDoc(design) {
				parts = append(parts, "gb")
			}
			if impl != "" {
				parts = append(parts, "g4")
			}
			parts = append(parts, "g5")
			list = strings.Join(parts, ",")
			sel.Note = "note: decomposed parent with no machines/; running " + list + " (G3/Gx run on the child designs; gt skipped: no machines)"
		}
	}
	if sel.Explicit {
		list = gateList
	}
	for _, g := range strings.Split(strings.ToLower(list), ",") {
		tok := strings.TrimSpace(g)
		if tok == "" {
			// "g2," once yielded `unknown gate(s): ` with an empty name
			return sel, fmt.Errorf("gate list %q contains an empty gate name (doubled or trailing comma)", gateList)
		}
		sel.Run[tok] = true
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
	if sel.Run["gk"] && (sel.Explicit || HasCheckers(design)) {
		out = append(out, CheckExternalCheckers(design)...)
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
