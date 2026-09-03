// Gate-suite selection and execution shared by `machinery check` and the
// Claude Code hook handler (`machinery hook`), so both run the exact same
// suite semantics: one implementation of which gates apply to a design.

package gates

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/alloy"
	"github.com/RamXX/machinery/internal/designlock"
	"github.com/RamXX/machinery/internal/pack"
)

// Selection is a resolved gate list for one suite run.
type Selection struct {
	Run      map[string]bool
	Explicit bool   // the list was caller-supplied rather than the default
	Note     string // decomposed-parent narrowing note (default selection only)
}

// RunOptions carries the run-time inputs a gate needs beyond the committed
// artifacts. They come from the caller's environment, never from the design,
// so a gate that reads one degrades to a stated non-check when it is absent
// instead of passing silently.
type RunOptions struct {
	// Commit is an optional repository-history anchor (--commit, or
	// MACHINERY_COMMIT). "" makes Ga derive HEAD from the design repository.
	Commit string
	// Complete is final-handoff mode: phase artifacts are no longer optional
	// adoption points and the closing completeness gate is appended.
	Complete bool
	// GitDesign is the logical checkout path used only for VCS object queries
	// when design itself is an immutable private materialization.
	GitDesign string
	// cargoWorkspaceManifest is an immutable exact-file snapshot of a Cargo
	// workspace root above --impl. It is populated only by Snapshot.RunSelected.
	cargoWorkspaceManifest string
	cargoWorkspaceLogical  string
}

// Snapshot holds one exclusive design snapshot across selection, gate
// execution, and any derived reporting that must agree with those reads.
type Snapshot struct {
	design        string
	logicalDesign string
	lock          *designlock.Lock
	workspace     *designlock.ExternalTreeSnapshot
}

var cargoAfterImplementationSnapshot = func() {}

func AcquireSnapshot(design string) (*Snapshot, error) {
	lock, err := designlock.AcquireReader(design)
	if err != nil {
		return nil, err
	}
	workspace, err := lock.MaterializeDesignWorkspace()
	if err != nil {
		return nil, errors.Join(err, lock.Release())
	}
	return &Snapshot{design: workspace.Path(), logicalDesign: design, lock: lock, workspace: workspace}, nil
}

func (s *Snapshot) Release() error { return errors.Join(s.workspace.Close(), s.lock.Release()) }

// DesignPath is the immutable, topology-preserving design root consumers must
// use for every design-derived read while the snapshot is held.
func (s *Snapshot) DesignPath() string { return s.design }

func (s *Snapshot) LogicalError(err error) error { return s.lock.LogicalError(err) }

func (s *Snapshot) TrackExternal(paths ...string) error { return s.lock.TrackExternal(paths...) }

func (s *Snapshot) TrackExternalTree(path string) error { return s.lock.TrackExternalTree(path) }

func (s *Snapshot) CheckUnchanged() error { return s.lock.CheckUnchanged() }

func (s *Snapshot) Select(gateList, impl string) (Selection, error) {
	sel, err := selectInSnapshot(s.design, gateList, impl)
	if err != nil {
		return sel, fmt.Errorf("%s", strings.ReplaceAll(err.Error(), s.design, s.logicalDesign))
	}
	return sel, s.lock.CheckUnchanged()
}

func (s *Snapshot) RunSelected(impl string, sel Selection, opt RunOptions) []*Gate {
	if opt.GitDesign == "" {
		opt.GitDesign = s.logicalDesign
	}
	logicalImpl := impl
	var stable *designlock.ExternalTreeSnapshot
	var cargoWorkspace *designlock.RegularFileSnapshot
	if impl != "" {
		var err error
		stable, err = s.lock.MaterializeExternalTree(impl)
		if err != nil {
			return []*Gate{{Title: "G0-snapshot", Errs: []string{"snapshot implementation tree: " + err.Error()}}}
		}
		if sel.Run["g4"] {
			cargoAfterImplementationSnapshot()
			workspacePath, locateErr := findCargoWorkspaceAuthority(stable.Path(), logicalImpl)
			if locateErr != nil {
				return []*Gate{{Title: "G0-snapshot", Errs: []string{"locate Cargo workspace authority: " + errors.Join(locateErr, stable.Close()).Error()}}}
			}
			if workspacePath != "" {
				cargoWorkspace, err = s.lock.MaterializeRegularFile(workspacePath)
				if err != nil {
					return []*Gate{{Title: "G0-snapshot", Errs: []string{"snapshot Cargo workspace authority: " + errors.Join(err, stable.Close()).Error()}}}
				}
				opt.cargoWorkspaceManifest = cargoWorkspace.Path()
				opt.cargoWorkspaceLogical = workspacePath
			}
		}
		impl = stable.Path()
	}
	out := runSelectedInSnapshot(s.design, impl, sel, opt)
	remapGatePaths(out, s.design, s.logicalDesign)
	if stable != nil {
		remapGatePaths(out, stable.Path(), logicalImpl)
	}
	if cargoWorkspace != nil {
		remapGatePaths(out, cargoWorkspace.Path(), opt.cargoWorkspaceLogical)
	}
	if err := s.lock.CheckUnchanged(); err != nil {
		out = append(out, &Gate{Title: "G0-snapshot", Errs: []string{err.Error()}})
	}
	if stable != nil {
		if err := stable.Close(); err != nil {
			out = append(out, &Gate{Title: "G0-snapshot", Errs: []string{"remove private implementation snapshot: " + err.Error()}})
		}
	}
	if cargoWorkspace != nil {
		if err := cargoWorkspace.Close(); err != nil {
			out = append(out, &Gate{Title: "G0-snapshot", Errs: []string{"remove private Cargo workspace snapshot: " + err.Error()}})
		}
	}
	return out
}

func remapGatePaths(gs []*Gate, from, to string) {
	remap := func(items []string) {
		for i := range items {
			items[i] = strings.ReplaceAll(items[i], from, to)
		}
	}
	for _, gate := range gs {
		remap(gate.Errs)
		remap(gate.Drift)
		remap(gate.Warns)
		remap(gate.Notes)
		remap(gate.checkedExtra)
	}
}

func (s *Snapshot) VersionSkewNote(gs []*Gate) string {
	return strings.ReplaceAll(VersionSkewNote(s.design, gs), s.design, s.logicalDesign)
}

// knownGateSet is the full gate vocabulary. Select and the hook-config
// validator (internal/hook) must agree on it, so both read this set through
// KnownGate; two hand-kept lists once drifted.
var knownGateSet = map[string]bool{
	"gm": true, "gs": true, "gu": true, "gp": true, "gi": true, "gn": true, "gc": true, "g2": true,
	"g3": true, "gd": true, "gl": true, "gx": true, "gk": true, "gb": true, "ge": true, "ga": true, "gj": true, "gv": true, "g4": true, "gt": true, "g5": true,
}

// KnownGate reports whether name names a gate this suite can run.
func KnownGate(name string) bool { return knownGateSet[name] }

// HasMachines reports whether design/machines holds any *.machine.json. It
// lists the directory (sortedGlob) rather than filepath.Glob: a design PATH
// containing glob metacharacters ([ ] * ?) must never defeat machine
// detection, which once silently dropped gates (GATE-2).
func HasMachines(design string) bool {
	has, err := probeMachines(design)
	return has || err != nil
}

func probeMachines(design string) (bool, error) {
	paths, invalid, err := regularGlob(filepath.Join(design, "machines"), "*.machine.json")
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("enumerate machines: %w", err)
	}
	if len(invalid) > 0 {
		return false, fmt.Errorf("machines/%s must be a regular file; symlinks and special entries are rejected", filepath.Base(invalid[0]))
	}
	return len(paths) > 0, nil
}

// HasModelith reports whether the design carries a *.modelith.yaml domain
// model (the Gx source artifact; the hook's stop-time selection keys on it).
func HasModelith(design string) bool {
	has, err := probeModelith(design)
	return has || err != nil
}

func probeModelith(design string) (bool, error) {
	paths, invalid, err := regularGlob(design, "*.modelith.yaml")
	if err != nil {
		return false, fmt.Errorf("enumerate modelith models: %w", err)
	}
	if len(invalid) > 0 {
		return false, fmt.Errorf("%s must be a regular modelith source inside the design; symlinks and special entries are rejected", filepath.Base(invalid[0]))
	}
	return len(paths) > 0, nil
}

func validateActivationDiscovery(design string) error {
	probes := []struct {
		name string
		fn   func() (bool, error)
	}{
		{"machines", func() (bool, error) { return probeMachines(design) }},
		{"modelith", func() (bool, error) { return probeModelith(design) }},
		{"markdown", func() (bool, error) { return probeEmbedActive(design) }},
		{"policy annotation", func() (bool, error) { return probeRegularFile(design, filepath.Join("formal", alloy.AnnotationName)) }},
		{"integrity annotation", func() (bool, error) {
			return probeRegularFile(design, filepath.Join("formal", alloy.IntegrityAnnotationName))
		}},
		{"isolation annotation", func() (bool, error) {
			return probeRegularFile(design, filepath.Join("formal", alloy.IsolationAnnotationName))
		}},
		{"BUILD.md", func() (bool, error) { return probeRegularFile(design, "BUILD.md") }},
		{"acceptance", func() (bool, error) { return probeRealDir(design, AcceptanceDirName) }},
		{"adjudications", func() (bool, error) { return probeRealDir(design, AdjudicationDirName) }},
		{"attestations", func() (bool, error) { return probeRegularFile(design, AttestationsFileName) }},
	}
	for _, probe := range probes {
		if _, err := probe.fn(); err != nil {
			return fmt.Errorf("cannot determine %s gate activation: %w", probe.name, err)
		}
	}
	return nil
}

// Select resolves a --gate list (or, when gateList is empty, the default
// suite) for design. impl is the implementation directory ("" when none was
// supplied); it decides whether the machine-less-parent narrowing keeps G4.
// The default narrows on a decomposed parent with no machines/, and an
// unknown or empty gate name is an error.
func Select(design, gateList, impl string) (Selection, error) {
	snapshot, err := AcquireSnapshot(design)
	if err != nil {
		return Selection{}, err
	}
	sel, selectErr := snapshot.Select(gateList, impl)
	return sel, errors.Join(selectErr, snapshot.Release())
}

func selectInSnapshot(design, gateList, impl string) (Selection, error) {
	sel := Selection{Run: map[string]bool{}, Explicit: gateList != ""}
	if err := validateDesignInventory(design); err != nil {
		return sel, fmt.Errorf("cannot inventory design artifacts: %w", err)
	}
	if err := validateActivationDiscovery(design); err != nil {
		return sel, err
	}
	list := "gm,gs,gu,gp,gi,gn,gc,g2,g3,gd,gl,gx,gk,gb,ge,ga,gj,gv,g4,gt,g5"
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
			// G3/Gx (the dogfood finding). The note lists what actually
			// runs; the golden corpus pins its text byte for byte.
			var parts []string
			for _, opt := range []struct {
				gate string
				has  func(string) bool
			}{
				{"gm", HasMigrationContract},
				{"gs", HasSurfaceLedger},
				{"gu", func(d string) bool { return HasTargetSurfaces(d) || HasHumanActions(d) }},
				{"gp", HasPolicyAnnotation},
				{"gi", HasIntegrityAnnotation},
				{"gn", HasIsolationAnnotation},
				// Gc is NOT machine-dependent: the parent's model declares
				// the invariants, so the parent carries the reconciliation.
				// Narrowing it away here would repeat the original defect
				// (no gate looked at invariants until the children built
				// machines, months after the declarations were authored).
				{"gc", HasModelith},
			} {
				if opt.has(design) {
					parts = append(parts, opt.gate)
				}
			}
			parts = append(parts, "g2", "gl")
			if HasCheckers(design) {
				parts = append(parts, "gk")
			}
			if HasBuildDoc(design) {
				parts = append(parts, "gb")
			}
			if EmbedActive(design) {
				parts = append(parts, "ge")
			}
			if AcceptanceActive(design) {
				parts = append(parts, "ga")
			}
			if AdjudicationActive(design) {
				parts = append(parts, "gj")
			}
			if AttestationActive(design) || AttestationOwed(design) {
				parts = append(parts, "gv")
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

// SelectRunAndNote resolves selection, runs every gate, and derives the
// version-skew note while holding one indivisible design snapshot.
func SelectRunAndNote(design, impl, gateList string, opt RunOptions) (sel Selection, run []*Gate, note string, retErr error) {
	snapshot, err := AcquireSnapshot(design)
	if err != nil {
		return sel, nil, "", err
	}
	defer func() { retErr = errors.Join(retErr, snapshot.Release()) }()
	sel, err = snapshot.Select(gateList, impl)
	if err != nil {
		return sel, nil, "", err
	}
	run = snapshot.RunSelected(impl, sel, opt)
	note = snapshot.VersionSkewNote(run)
	return sel, run, note, nil
}

// RunSelected runs the selected gates in canonical order (Gm, Gs, Gu, Gp, Gi,
// Gn, Gc, G2, G3, Gd, Gl, Gx, Gk, Gb, Ge, Ga, Gj, Gv, G4, Gt, G5) with `machinery check`'s applicability
// rules: opt-in gates run only when their source exists (or when explicitly
// requested), G4 and Gt only with an impl dir, and G5 only when explicitly
// requested or when the design is decomposed. opt carries the run-time inputs
// (the commit under review); the zero value checks nothing that needs one.
// The returned gates carry their findings; the caller emits them.
func RunSelected(design, impl string, sel Selection, opt RunOptions) []*Gate {
	snapshot, err := AcquireSnapshot(design)
	if err != nil {
		return []*Gate{{Title: "G0-snapshot", Errs: []string{err.Error()}}}
	}
	out := snapshot.RunSelected(impl, sel, opt)
	if err := snapshot.Release(); err != nil {
		out = append(out, &Gate{Title: "G0-snapshot", Errs: []string{"release design snapshot lock: " + err.Error()}})
	}
	return out
}

func runSelectedInSnapshot(design, impl string, sel Selection, opt RunOptions) []*Gate {
	var out []*Gate
	if sel.Run["gm"] && (sel.Explicit || HasMigrationContract(design)) {
		out = append(out, CheckMigration(design, impl))
	}
	if sel.Run["gs"] && (sel.Explicit || HasSurfaceLedger(design)) {
		out = append(out, CheckSurface(design))
	}
	if sel.Run["gu"] && (sel.Explicit || HasTargetSurfaces(design) || HasHumanActions(design)) {
		out = append(out, CheckTargetSurfaces(design))
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
	if sel.Run["gc"] && (sel.Explicit || HasModelith(design)) {
		out = append(out, CheckCarriers(design))
	}
	if sel.Run["g2"] {
		out = append(out, CheckC4(design))
	}
	if sel.Run["g3"] {
		out = append(out, CheckMachines(design))
	}
	if sel.Run["gd"] && (sel.Explicit || HasMachines(design)) {
		out = append(out, CheckIDCitations(design))
	}
	if sel.Run["gl"] {
		out = append(out, CheckLedger(design))
	}
	if sel.Run["gx"] {
		out = append(out, CheckTraceability(design))
	}
	if sel.Run["gk"] && (sel.Explicit || HasCheckers(design)) {
		out = append(out, CheckExternalCheckers(design)...)
	}
	if sel.Run["gb"] && (sel.Explicit || opt.Complete || HasBuildDoc(design)) {
		out = append(out, CheckBuildPlan(design))
	}
	if sel.Run["ge"] && (sel.Explicit || EmbedActive(design)) {
		out = append(out, CheckEmbeds(design))
	}
	if sel.Run["ga"] && (sel.Explicit || opt.Complete || AcceptanceActive(design)) {
		gitDesign := opt.GitDesign
		if gitDesign == "" {
			gitDesign = design
		}
		out = append(out, checkAcceptanceWithGit(design, gitDesign, opt.Commit, opt.Complete))
	}
	if sel.Run["gj"] && (sel.Explicit || AdjudicationActive(design)) {
		out = append(out, CheckAdjudications(design))
	}
	if sel.Run["gv"] && (sel.Explicit || opt.Complete || AttestationActive(design) || AttestationOwed(design)) {
		out = append(out, CheckAttestations(design))
	}
	if sel.Run["g4"] && impl != "" {
		out = append(out, checkImportsWithWorkspace(design, impl, nil, opt.cargoWorkspaceManifest))
	}
	if sel.Run["gt"] && impl != "" {
		out = append(out, CheckOracleCoverage(design, impl))
	}
	if sel.Run["g5"] && (sel.Explicit || pack.HasDecomposition(design) || pack.HasPack(design)) {
		out = append(out, CheckPack(design))
	}
	if opt.Complete {
		out = append(out, CheckFinalHandoff(design))
	}
	return out
}
