package pack

// Child-side pack conformance: design/packmap.yaml maps the child's exposed
// machine onto the pack's contract machine, and the generated refinement
// module proves (under TLC, via verify-formal) that the child refines the
// contract its neighbors rely on. The mapping is RECONCILED against both
// machines before anything is emitted: a drifted map fails generation instead
// of proving a stale twin.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/tla"
)

// PackMap is the parsed design/packmap.yaml.
type PackMap struct {
	Subsystem string            // must equal the pack's subsystem id
	PackHash  string            // pins the pack content_hash the child was built against
	Machine   string            // the child machine (basename, no .machine.json) that realizes the contract
	Mapping   map[string]string // child top-level state -> contract state
	Order     []string          // mapping keys in source order (deterministic emission)
}

// LoadPackMap reads and parses design/packmap.yaml.
func LoadPackMap(design string) (*PackMap, error) {
	data, err := os.ReadFile(filepath.Join(design, "packmap.yaml"))
	if err != nil {
		return nil, fmt.Errorf("pack: cannot read packmap.yaml: %w", err)
	}
	v, err := ir.LoadYAML(data)
	if err != nil || v.AsObject() == nil {
		return nil, fmt.Errorf("pack: packmap.yaml is not a yaml mapping")
	}
	o := v.AsObject()
	pm := &PackMap{
		Subsystem: o.GetString("subsystem"),
		PackHash:  o.GetString("pack_hash"),
		Machine:   o.GetString("machine"),
		Mapping:   map[string]string{},
	}
	mo := o.GetObject("mapping")
	for _, k := range mo.Keys() {
		pm.Mapping[k] = mo.GetString(k)
		pm.Order = append(pm.Order, k)
	}
	return pm, nil
}

// LoadPackManifest reads design/pack/pack.yaml of a child design.
func LoadPackManifest(design string) (*ir.Object, error) {
	data, err := os.ReadFile(filepath.Join(design, "pack", "pack.yaml"))
	if err != nil {
		return nil, fmt.Errorf("pack: cannot read pack/pack.yaml: %w", err)
	}
	v, err := ir.LoadYAML(data)
	if err != nil || v.AsObject() == nil {
		return nil, fmt.Errorf("pack: pack/pack.yaml is not a yaml mapping")
	}
	return v.AsObject(), nil
}

// PackFilesOnDisk reads the committed pack files of a child design (for hash
// verification and freshness diffs).
func PackFilesOnDisk(design string) (map[string]string, error) {
	dir := filepath.Join(design, "pack")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("pack: %w", err)
	}
	files := map[string]string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		files[e.Name()] = string(data)
	}
	return files, nil
}

func topStateNames(m *ir.Value) []string {
	var out []string
	for _, s := range ir.WalkStates(m.AsObject().Get2("states"), "") {
		if !strings.Contains(s.Path, ".") {
			out = append(out, s.Name)
		}
	}
	return out
}

// ReconcileMap validates the packmap against the child machine and the
// contract machine. Every finding is returned (not just the first).
func ReconcileMap(pm *PackMap, child, contract *ir.Value) error {
	var errs []string
	childStates := topStateNames(child)
	contractStates := map[string]bool{}
	for _, s := range topStateNames(contract) {
		contractStates[s] = true
	}
	childSet := map[string]bool{}
	for _, s := range childStates {
		childSet[s] = true
	}
	for _, s := range childStates {
		if _, ok := pm.Mapping[s]; !ok {
			errs = append(errs, fmt.Sprintf("child state %s has no mapping entry; the map must be total", ir.Repr(s)))
		}
	}
	for k, v := range pm.Mapping {
		if !childSet[k] {
			errs = append(errs, fmt.Sprintf("mapping names %s, which is not a child machine state", ir.Repr(k)))
		}
		if !contractStates[v] {
			errs = append(errs, fmt.Sprintf("mapping sends %s to %s, which is not a contract state", ir.Repr(k), ir.Repr(v)))
		}
	}
	ci := child.AsObject().GetString("initial")
	ki := contract.AsObject().GetString("initial")
	if pm.Mapping[ci] != ki {
		errs = append(errs, fmt.Sprintf("child initial %s maps to %s; the contract starts at %s", ir.Repr(ci), ir.Repr(pm.Mapping[ci]), ir.Repr(ki)))
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("pack: MAP RECONCILIATION FAILED: %s", strings.Join(errs, "; "))
	}
	return nil
}

// GenerateRefinement builds the refinement artifacts for a child design, in
// memory: filename -> content, all destined for design/formal/. It emits the
// contract's TLA module (copied from the pack so the child proves against the
// SAME bytes the parent composition instances) and the refinement module that
// TLC checks via verify-formal.
func GenerateRefinement(design string) (map[string]string, error) {
	pm, err := LoadPackMap(design)
	if err != nil {
		return nil, err
	}
	manifest, err := LoadPackManifest(design)
	if err != nil {
		return nil, err
	}
	if pm.Subsystem != manifest.GetString("subsystem") {
		return nil, fmt.Errorf("pack: packmap subsystem %s does not match the pack's %s",
			ir.Repr(pm.Subsystem), ir.Repr(manifest.GetString("subsystem")))
	}
	files, err := PackFilesOnDisk(design)
	if err != nil {
		return nil, err
	}
	if got := ContentHash(files); got != manifest.GetString("content_hash") {
		return nil, fmt.Errorf("pack: the copied pack fails its own content hash (edited by hand, or a partial copy); re-copy it from the parent")
	}
	if pm.PackHash != manifest.GetString("content_hash") {
		return nil, fmt.Errorf("pack: packmap pins pack_hash %s but the copied pack is %s; the pack changed since the map was written, re-verify every obligation and update pack_hash",
			shortHash(pm.PackHash), shortHash(manifest.GetString("content_hash")))
	}
	cmod := manifest.GetString("contract_module")
	contractPath := filepath.Join(design, "pack", cmod+".machine.json")
	contract, err := ir.LoadMachineJSON(contractPath)
	if err != nil {
		return nil, fmt.Errorf("pack: %w", err)
	}
	childPath := filepath.Join(design, "machines", pm.Machine+".machine.json")
	child, err := ir.LoadMachineJSON(childPath)
	if err != nil {
		return nil, fmt.Errorf("pack: %w", err)
	}
	if err := ReconcileMap(pm, child, contract); err != nil {
		return nil, err
	}

	childMid, _, childCfg, gerr := tla.Generate(childPath)
	if gerr != nil {
		return nil, fmt.Errorf("pack: child machine: %w", gerr)
	}

	out := map[string]string{}
	// the contract module, byte-identical to the pack's copy
	out[cmod+".tla"] = files[cmod+".tla"]

	// the refinement module: child spec, mapped, instancing the contract
	var b strings.Builder
	mod := childMid + "PackRefinement"
	fmt.Fprintf(&b, "---- MODULE %s ----\n", mod)
	b.WriteString("\\* GENERATED by machinery pack refine. RECONCILED against the child machine\n")
	b.WriteString("\\* and the pack's contract machine; a drifted packmap fails generation.\n")
	fmt.Fprintf(&b, "\\* Obligation: %s (child) refines %s (the contract the parent composition\n", childMid, cmod)
	b.WriteString("\\* instances). TLC checks Spec => C!Spec under the state mapping below.\n")
	fmt.Fprintf(&b, "EXTENDS %s\n\n", childMid)
	b.WriteString("Map(s) ==\n")
	// deterministic order: child machine state order, not map order
	for i, s := range topStateNames(child) {
		sep := "CASE"
		if i > 0 {
			sep = "  []"
		}
		fmt.Fprintf(&b, "  %s s = \"%s\" -> \"%s\"\n", sep, s, pm.Mapping[s])
	}
	fmt.Fprintf(&b, "\nC == INSTANCE %s WITH st <- Map(st)\n", cmod)
	b.WriteString("CSpecHolds == C!Spec\n")
	b.WriteString("====\n")
	out[mod+".tla"] = b.String()

	// cfg: same constants as the child spec, property = the contract spec
	var cfg strings.Builder
	for _, line := range strings.Split(childCfg, "\n") {
		if strings.HasPrefix(line, "CONSTANT") {
			cfg.WriteString(line + "\n")
		}
	}
	cfg.WriteString("SPECIFICATION Spec\nPROPERTY CSpecHolds\n")
	out[mod+".cfg"] = cfg.String()
	return out, nil
}

func shortHash(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	if h == "" {
		return "(empty)"
	}
	return h
}

// WriteRefinement generates and writes the refinement artifacts to design/formal/.
func WriteRefinement(design string) ([]string, error) {
	files, err := GenerateRefinement(design)
	if err != nil {
		return nil, err
	}
	fdir := filepath.Join(design, "formal")
	if err := os.MkdirAll(fdir, 0o755); err != nil {
		return nil, err
	}
	var names []string
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(fdir, n), []byte(files[n]), 0o644); err != nil {
			return nil, err
		}
	}
	return names, nil
}
