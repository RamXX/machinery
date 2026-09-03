package pack

// Child-side pack conformance: design/packmap.yaml maps the child's exposed
// machine onto the pack's contract machine, and the generated refinement
// module proves (under TLC, via verify-formal) that the child refines the
// contract its neighbors rely on. The mapping is RECONCILED against both
// machines before anything is emitted: a drifted map fails generation instead
// of proving a stale twin.

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/artifactset"
	"github.com/RamXX/machinery/internal/designlock"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/version"
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
	root, err := openDesignRoot(design)
	if err != nil {
		return nil, err
	}
	pm, loadErr := loadPackMapRoot(root)
	return pm, errors.Join(loadErr, root.Close())
}

func loadPackMapRoot(root *os.Root) (*PackMap, error) {
	data, err := readDesignFileRoot(root, "packmap.yaml")
	if err != nil {
		return nil, fmt.Errorf("pack: cannot read packmap.yaml: %w", err)
	}
	v, err := ir.LoadYAML(data)
	if err != nil || v.AsObject() == nil {
		return nil, fmt.Errorf("pack: packmap.yaml is not a yaml mapping")
	}
	o := v.AsObject()
	allowed := map[string]bool{"subsystem": true, "pack_hash": true, "machine": true, "mapping": true}
	var unknown []string
	for _, key := range o.Keys() {
		if !allowed[key] {
			unknown = append(unknown, key)
		}
	}
	sort.Strings(unknown)
	if len(unknown) > 0 {
		return nil, fmt.Errorf("pack: packmap.yaml contains unknown root key %s", ir.Repr(unknown[0]))
	}
	for _, key := range []string{"subsystem", "pack_hash", "machine"} {
		field := o.Get2(key)
		if field == nil || field.Kind != ir.KindString || strings.TrimSpace(field.AsString()) == "" {
			return nil, fmt.Errorf("pack: packmap.yaml %s is required and must be a non-empty string", key)
		}
	}
	mappingValue := o.Get2("mapping")
	if mappingValue == nil || mappingValue.Kind != ir.KindObject {
		return nil, fmt.Errorf("pack: packmap.yaml mapping is required and must be a mapping of child state to contract state")
	}
	pm := &PackMap{
		Subsystem: o.GetString("subsystem"),
		PackHash:  o.GetString("pack_hash"),
		Machine:   o.GetString("machine"),
		Mapping:   map[string]string{},
	}
	mo := mappingValue.AsObject()
	for _, k := range mo.Keys() {
		value := mo.Get2(k)
		if value == nil || value.Kind != ir.KindString || strings.TrimSpace(value.AsString()) == "" {
			return nil, fmt.Errorf("pack: packmap.yaml mapping value for %s must be a non-empty string", ir.Repr(k))
		}
		pm.Mapping[k] = value.AsString()
		pm.Order = append(pm.Order, k)
	}
	return pm, nil
}

// LoadPackManifest reads design/pack/pack.yaml of a child design.
func LoadPackManifest(design string) (*ir.Object, error) {
	root, err := openDesignRoot(design)
	if err != nil {
		return nil, err
	}
	manifest, loadErr := loadPackManifestRoot(root)
	return manifest, errors.Join(loadErr, root.Close())
}

func loadPackManifestRoot(root *os.Root) (*ir.Object, error) {
	data, err := readDesignFileRoot(root, filepath.Join("pack", "pack.yaml"))
	if err != nil {
		return nil, fmt.Errorf("pack: cannot read pack/pack.yaml: %w", err)
	}
	v, err := ir.LoadYAML(data)
	if err != nil || v.AsObject() == nil {
		return nil, fmt.Errorf("pack: pack/pack.yaml is not a yaml mapping")
	}
	o := v.AsObject()
	allowed := map[string]bool{"pack_version": true, "pack_revision": true, "subsystem": true, "contract_module": true, "owns": true, "components": true, "boundaries": true, "delegated_invariants": true, "content_hash": true}
	var unknown []string
	for _, key := range o.Keys() {
		if !allowed[key] {
			unknown = append(unknown, key)
		}
	}
	sort.Strings(unknown)
	if len(unknown) > 0 {
		return nil, fmt.Errorf("pack: pack/pack.yaml contains unknown root key %s", ir.Repr(unknown[0]))
	}
	numberField := func(key string, exact int) error {
		field := o.Get2(key)
		if field == nil || field.Kind != ir.KindNumber {
			return fmt.Errorf("pack: pack/pack.yaml %s is required and must be the integer %d", key, exact)
		}
		value, err := strconv.Atoi(string(field.AsNumber()))
		if err != nil || value != exact {
			return fmt.Errorf("pack: pack/pack.yaml %s must be the integer %d", key, exact)
		}
		return nil
	}
	if err := numberField("pack_version", 1); err != nil {
		return nil, err
	}
	revision := o.Get2("pack_revision")
	if revision == nil || revision.Kind != ir.KindNumber {
		return nil, fmt.Errorf("pack: pack/pack.yaml pack_revision is required and must be an integer >= 1")
	}
	revisionNumber, revisionErr := strconv.Atoi(string(revision.AsNumber()))
	if revisionErr != nil || revisionNumber < 1 {
		return nil, fmt.Errorf("pack: pack/pack.yaml pack_revision must be an integer >= 1")
	}
	for _, key := range []string{"subsystem", "contract_module", "content_hash"} {
		field := o.Get2(key)
		if field == nil || field.Kind != ir.KindString || strings.TrimSpace(field.AsString()) == "" {
			return nil, fmt.Errorf("pack: pack/pack.yaml %s is required and must be a non-empty string", key)
		}
	}
	hashText := o.GetString("content_hash")
	if decoded, err := hex.DecodeString(hashText); err != nil || len(decoded) != 32 {
		return nil, fmt.Errorf("pack: pack/pack.yaml content_hash must be exactly 64 lowercase hexadecimal characters")
	}
	if hashText != strings.ToLower(hashText) {
		return nil, fmt.Errorf("pack: pack/pack.yaml content_hash must be exactly 64 lowercase hexadecimal characters")
	}
	for _, key := range []string{"owns", "components", "boundaries", "delegated_invariants"} {
		field := o.Get2(key)
		if field == nil || field.Kind != ir.KindArray {
			return nil, fmt.Errorf("pack: pack/pack.yaml %s is required and must be an array of strings", key)
		}
		for index, item := range field.AsArray() {
			if item == nil || item.Kind != ir.KindString || strings.TrimSpace(item.AsString()) == "" {
				return nil, fmt.Errorf("pack: pack/pack.yaml %s[%d] must be a non-empty string", key, index)
			}
		}
	}
	return o, nil
}

// PackFilesOnDisk reads the committed pack files of a child design (for hash
// verification and freshness diffs). A directory inside pack/ is a hard
// error: the content hash covers files only, so a smuggled subdirectory would
// carry unhashed content under the pack's authority.
func PackFilesOnDisk(design string) (map[string]string, error) {
	root, err := openDesignRoot(design)
	if err != nil {
		return nil, err
	}
	files, loadErr := packFilesOnDiskRoot(root)
	return files, errors.Join(loadErr, root.Close())
}

func packFilesOnDiskRoot(root *os.Root) (map[string]string, error) {
	info, lerr := root.Lstat("pack")
	if lerr != nil {
		return nil, fmt.Errorf("pack: %w", lerr)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("pack: pack/ must be a real directory, not a symlink or non-directory")
	}
	packRoot, err := root.OpenRoot("pack")
	if err != nil {
		return nil, fmt.Errorf("pack: %w", err)
	}
	defer packRoot.Close()
	dir, err := packRoot.Open(".")
	if err != nil {
		return nil, fmt.Errorf("pack: %w", err)
	}
	entries, readErr := dir.ReadDir(-1)
	if err := errors.Join(readErr, dir.Close()); err != nil {
		return nil, fmt.Errorf("pack: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	files := map[string]string{}
	for _, e := range entries {
		if e.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("pack: pack/ contains symlink %s; every hash-covered pack member must be a regular file", ir.Repr(e.Name()))
		}
		if e.IsDir() {
			return nil, fmt.Errorf("pack: pack/ contains a directory %s; a pack is a flat generated file set and a subdirectory escapes the content hash entirely; remove it and re-copy the pack", ir.Repr(e.Name()))
		}
		data, err := readDesignFileRoot(packRoot, e.Name())
		if err != nil {
			return nil, err
		}
		files[e.Name()] = string(data)
	}
	return files, nil
}

// BoundaryEventNames parses the pack's generated events.md and returns the
// event names (any table with event and direction columns).
func BoundaryEventNames(eventsMD string) []string {
	var out []string
	for _, tbl := range ir.ParseMdTables(eventsMD) {
		ei := ir.FindCol(tbl.Header, "event")
		di := ir.FindCol(tbl.Header, "direction")
		if ei < 0 || di < 0 {
			continue
		}
		for _, r := range tbl.Rows {
			if ei < len(r) {
				if n := ir.CleanCell(r[ei]); n != "" {
					out = append(out, n)
				}
			}
		}
	}
	return out
}

// checkRefinementBinding refuses a packmap that binds a machine with no stake
// in the contract. The map's type checks pass for ANY machine whose top
// states cover the contract states, so a byte-copied shim of the contract
// machine used to prove "refinement" trivially. The bound machine must either
// be the lifecycle machine of a pack-owned entity (declared _lifecycle_of or
// named after the entity) or touch at least one of the pack's boundary events
// (handle, ignore, or emit it).
func checkRefinementBinding(pm *PackMap, manifest *ir.Object, files map[string]string, child *ir.Value) error {
	owns := map[string]bool{}
	for _, v := range manifest.Get2("owns").AsArray() {
		if v != nil && v.Kind == ir.KindString {
			owns[v.AsString()] = true
		}
	}
	if owns[child.AsObject().GetString("_lifecycle_of")] || owns[pm.Machine] {
		return nil
	}
	touched := map[string]bool{}
	for _, s := range ir.WalkStates(child.AsObject().Get2("states"), "") {
		so := s.Node.AsObject()
		if so == nil {
			continue
		}
		for _, k := range so.GetObject("on").Keys() {
			touched[k] = true
		}
		for _, k := range so.GetObject("_ignores").Keys() {
			touched[k] = true
		}
		for a := range ir.ActionsOf(s.Node, nil, s.Path) {
			touched[a] = true
		}
		for _, inv := range ir.InvokesOf(s.Node) {
			if io := inv.AsObject(); io != nil {
				if src := io.GetString("src"); src != "" {
					touched[src] = true
				}
			}
		}
	}
	for _, ev := range BoundaryEventNames(files["events.md"]) {
		if touched[ev] {
			return nil
		}
	}
	return fmt.Errorf("pack: packmap binds machine %s, which is neither the lifecycle machine of a pack-owned entity (_lifecycle_of or the entity's name) nor handles, ignores, or emits any of the pack's boundary events; a machine with no stake in the contract proves nothing, bind the machine that realizes it", ir.Repr(pm.Machine))
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
func GenerateRefinement(design string) (result map[string]string, retErr error) {
	root, err := openDesignRoot(design)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, root.Close())
		if retErr != nil {
			result = nil
		}
	}()
	pm, err := loadPackMapRoot(root)
	if err != nil {
		return nil, err
	}
	manifest, err := loadPackManifestRoot(root)
	if err != nil {
		return nil, err
	}
	if pm.Subsystem != manifest.GetString("subsystem") {
		return nil, fmt.Errorf("pack: packmap subsystem %s does not match the pack's %s",
			ir.Repr(pm.Subsystem), ir.Repr(manifest.GetString("subsystem")))
	}
	files, err := packFilesOnDiskRoot(root)
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
	contractRel := filepath.Join("pack", cmod+".machine.json")
	contractRaw, err := readDesignFileRoot(root, contractRel)
	if err != nil {
		return nil, fmt.Errorf("pack: %w", err)
	}
	contractPath := filepath.Join(design, contractRel)
	contract, err := ir.LoadMachineJSONBytes(contractPath, contractRaw)
	if err != nil {
		return nil, fmt.Errorf("pack: %w", err)
	}
	childRel := filepath.Join("machines", pm.Machine+".machine.json")
	childRaw, err := readDesignFileRoot(root, childRel)
	if err != nil {
		return nil, fmt.Errorf("pack: %w", err)
	}
	childPath := filepath.Join(design, childRel)
	child, err := ir.LoadMachineJSONBytes(childPath, childRaw)
	if err != nil {
		return nil, fmt.Errorf("pack: %w", err)
	}
	if err := checkRefinementBinding(pm, manifest, files, child); err != nil {
		return nil, err
	}
	if err := ReconcileMap(pm, child, contract); err != nil {
		return nil, err
	}

	childMid, _, childCfg, gerr := generateTLAFromMachineBytes(childPath, childRaw)
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
	// Stamp the GENERATED modules only (P-F10): the contract module above
	// stays a byte-copy of the pack's hash-covered file, so it never carries
	// a stamp. G5 strips the stamp line before its freshness diff.
	out[mod+".tla"] = version.StampTLAModule(b.String())

	// cfg: same constants as the child spec, property = the contract spec
	var cfg strings.Builder
	for _, line := range strings.Split(childCfg, "\n") {
		if strings.HasPrefix(line, "CONSTANT") {
			cfg.WriteString(line + "\n")
		}
	}
	cfg.WriteString("SPECIFICATION Spec\nPROPERTY CSpecHolds\n")
	out[mod+".cfg"] = version.StampCfg(cfg.String())
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
	snapshot, err := designlock.Acquire(design)
	if err != nil {
		return nil, err
	}
	if err := snapshot.ResumeExpected("pack-refine", "rerun `machinery pack refine` with the same arguments"); err != nil {
		return nil, errors.Join(err, snapshot.Release())
	}
	files, retErr := GenerateRefinement(snapshot.SourceRoot())
	var names []string
	if retErr == nil {
		stale, staleErr := staleRefinementArtifacts(snapshot.SourceRoot(), design, files)
		if staleErr != nil {
			retErr = staleErr
		}
		expected := make([]designlock.OutputExpectation, 0, len(files)+len(stale))
		for name, body := range files {
			expected = append(expected, designlock.ExpectFile(filepath.Join(design, "formal", name), []byte(body), 0o644))
		}
		for _, name := range stale {
			expected = append(expected, designlock.ExpectAbsent(filepath.Join(design, "formal", name.Name)))
		}
		if retErr == nil {
			fdir := filepath.Join(design, "formal")
			retErr = snapshot.PublishExpectedRooted("pack-refine", "rerun `machinery pack refine` with the same arguments", expected, func(outputs *designlock.OutputScope) error {
				return outputs.WithRoot(fdir, func(root *os.Root) error {
					var err error
					names, err = writeRefinementArtifactsSetRooted(fdir, root, files, stale)
					return err
				})
			})
		}
	}
	retErr = errors.Join(retErr, snapshot.Release())
	retErr = snapshot.LogicalError(retErr)
	if retErr != nil {
		return nil, retErr
	}
	return names, nil
}

func writeRefinementArtifacts(design string, files map[string]string) ([]string, error) {
	stale, err := staleRefinementArtifacts(design, design, files)
	if err != nil {
		return nil, err
	}
	return writeRefinementArtifactsSet(design, files, stale)
}

func staleRefinementArtifacts(sourceDesign, liveDesign string, files map[string]string) (stale []artifactset.RemovalPrecondition, retErr error) {
	fdir, err := designPathNoSymlinks(sourceDesign, "formal")
	if err != nil {
		return nil, fmt.Errorf("pack: formal output: %w", err)
	}
	info, err := os.Lstat(fdir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("pack: formal output must be a real directory, not a symlink or non-directory")
	}
	root, err := os.OpenRoot(fdir)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	dir, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	entries, readErr := dir.ReadDir(-1)
	if err := errors.Join(readErr, dir.Close()); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	liveFormal := filepath.Join(liveDesign, "formal")
	for _, entry := range entries {
		name := entry.Name()
		if _, keep := files[name]; keep || !strings.HasSuffix(name, "PackRefinement.tla") {
			continue
		}
		body, err := root.ReadFile(name)
		if err != nil {
			return nil, err
		}
		if !canonicalPackRefinementTLA(name, body) {
			continue
		}
		liveBody, condition, err := artifactset.InspectRemovalCandidate(liveFormal, name)
		if err != nil || !bytes.Equal(liveBody, body) {
			if err == nil {
				err = fmt.Errorf("live stale refinement artifact changed after snapshot")
			}
			return nil, err
		}
		stale = append(stale, condition)
		cfg := strings.TrimSuffix(name, ".tla") + ".cfg"
		sourceCfg, cfgErr := root.ReadFile(cfg)
		if os.IsNotExist(cfgErr) {
			continue
		}
		if cfgErr != nil {
			return nil, cfgErr
		}
		if !canonicalPackRefinementCFG(sourceCfg) {
			continue
		}
		liveCfg, cfgCondition, err := artifactset.InspectRemovalCandidate(liveFormal, cfg)
		if err != nil || !bytes.Equal(liveCfg, sourceCfg) {
			if err == nil {
				err = fmt.Errorf("live stale refinement config changed after snapshot")
			}
			return nil, err
		}
		stale = append(stale, cfgCondition)
	}
	for _, entry := range entries {
		name := entry.Name()
		if _, keep := files[name]; keep || !strings.HasSuffix(name, "PackRefinement.cfg") {
			continue
		}
		anchor := strings.TrimSuffix(name, ".cfg") + ".tla"
		if _, err := root.Lstat(anchor); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		sourceCfg, err := root.ReadFile(name)
		if err != nil {
			return nil, err
		}
		if !canonicalPackRefinementCFG(sourceCfg) {
			continue
		}
		liveCfg, condition, err := artifactset.InspectRemovalCandidate(liveFormal, name)
		if err != nil || !bytes.Equal(liveCfg, sourceCfg) {
			if err == nil {
				err = fmt.Errorf("live stale refinement config changed after snapshot")
			}
			return nil, err
		}
		stale = append(stale, condition)
	}
	sort.Slice(stale, func(i, j int) bool { return stale[i].Name < stale[j].Name })
	return stale, nil
}

func canonicalPackRefinementTLA(name string, body []byte) bool {
	module := strings.TrimSuffix(name, ".tla")
	lines := bytes.SplitN(body, []byte("\n"), 4)
	return strings.HasSuffix(module, "PackRefinement") && len(lines) >= 3 &&
		string(lines[0]) == "---- MODULE "+module+" ----" && bytes.HasPrefix(lines[1], []byte(`\* machinery-version: `)) &&
		string(lines[2]) == `\* GENERATED by machinery pack refine. RECONCILED against the child machine`
}

func canonicalPackRefinementCFG(body []byte) bool {
	lines := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
	if len(lines) < 3 || !strings.HasPrefix(lines[0], `\* machinery-version: `) || lines[len(lines)-2] != "SPECIFICATION Spec" || lines[len(lines)-1] != "PROPERTY CSpecHolds" {
		return false
	}
	for _, line := range lines[1 : len(lines)-2] {
		if !strings.HasPrefix(line, "CONSTANT ") || strings.TrimSpace(line) != line {
			return false
		}
	}
	return true
}

func writeRefinementArtifactsSet(design string, files map[string]string, stale []artifactset.RemovalPrecondition) ([]string, error) {
	fdir, err := designPathNoSymlinks(design, "formal")
	if err != nil {
		return nil, fmt.Errorf("pack: formal output: %w", err)
	}
	if err := os.MkdirAll(fdir, 0o755); err != nil {
		return nil, err
	}
	info, err := os.Lstat(fdir)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("pack: formal output must be a real directory, not a symlink or non-directory")
	}
	var names []string
	for n := range files {
		if filepath.Base(n) != n || n == "." {
			return nil, fmt.Errorf("pack: generated unsafe refinement artifact name %s", ir.Repr(n))
		}
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		info, statErr := os.Lstat(filepath.Join(fdir, n))
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return nil, statErr
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("pack: generated target %s must be a regular file, not a symlink or special file", ir.Repr(n))
		}
	}
	committed := make(map[string][]byte, len(names))
	for _, n := range names {
		committed[n] = []byte(files[n])
	}
	if err := artifactset.ReconcilePlanned(fdir, committed, stale); err != nil {
		return nil, err
	}
	return names, nil
}

func writeRefinementArtifactsSetRooted(scope string, root *os.Root, files map[string]string, stale []artifactset.RemovalPrecondition) ([]string, error) {
	names := make([]string, 0, len(files))
	committed := make(map[string][]byte, len(files))
	for name, body := range files {
		if filepath.Base(name) != name || name == "." {
			return nil, fmt.Errorf("pack: generated unsafe refinement artifact name %s", ir.Repr(name))
		}
		names = append(names, name)
		committed[name] = []byte(body)
	}
	sort.Strings(names)
	if err := artifactset.ReconcilePlannedRooted(scope, root, committed, stale); err != nil {
		return nil, err
	}
	return names, nil
}
