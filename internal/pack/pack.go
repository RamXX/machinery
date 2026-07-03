// Package pack implements recursive decomposition via contract packs.
//
// A parent design declares its subsystems in design/decomposition.yaml. Each
// subsystem gets a generated, frozen PACK: the domain slice it owns, the
// boundary event contracts it may rely on, the abstract contract machine its
// neighbors hold it to (plus that machine's TLA+ module), the parent
// invariants delegated to it, and a content hash. The pack is simultaneously
// the parent's entire model of the child and the child's entire view of the
// parent: a child design consumes exactly one pack (copied to design/pack/),
// maps its exposed machine onto the contract machine (design/packmap.yaml),
// and proves the refinement under TLC. G5-pack holds both sides to it.
//
// Everything here is generated and byte-deterministic, so freshness is a
// byte-diff and staleness is drift, exactly like the oracles.
package pack

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/lint"
	"github.com/RamXX/machinery/internal/tla"
)

// Subsystem is one entry of decomposition.yaml.
type Subsystem struct {
	ID                  string
	Owns                []string // Modelith entities owned (exactly-once across subsystems)
	Components          []string // producer/consumer names in the event-contract table
	Boundaries          []string // Architecture Contract boundary ids
	ContractMachine     string   // path relative to the design dir (contracts/X.machine.json)
	DelegatedInvariants []string
	ChildDesign         string // optional: child design dir relative to the parent design dir
}

// Decomposition is the parsed, validated decomposition.yaml.
type Decomposition struct {
	Version    int
	Subsystems []Subsystem
}

// EventRow is one boundary event contract from the parent's event-contract table.
type EventRow struct {
	Event, Producer, Consumer, Payload, Delivery, Ordering, Dedupe string
}

// DecompositionPath returns the canonical location.
func DecompositionPath(design string) string {
	return filepath.Join(design, "decomposition.yaml")
}

// HasDecomposition reports whether the design is a decomposed parent.
func HasDecomposition(design string) bool {
	_, err := os.Stat(DecompositionPath(design))
	return err == nil
}

// HasPack reports whether the design is a decomposed child (consumes a pack).
func HasPack(design string) bool {
	_, err := os.Stat(filepath.Join(design, "pack", "pack.yaml"))
	return err == nil
}

// subsystemIDRe restricts ids to a bare-name charset: they become path
// segments (packs/<id>.pack) and must not traverse out of the design tree.
// Interior dots are path-safe (orders.v2); separators and empty segments
// (which is what ".." would be) are not.
var subsystemIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]+(\.[A-Za-z0-9_-]+)*$`)

// pathInsideDesign reports whether the (relative) path stays inside the
// design directory once cleaned. Absolute paths never qualify.
func pathInsideDesign(design, rel string) bool {
	if filepath.IsAbs(rel) {
		return false
	}
	joined := filepath.Clean(filepath.Join(design, rel))
	back, err := filepath.Rel(filepath.Clean(design), joined)
	if err != nil {
		return false
	}
	return back != ".." && !strings.HasPrefix(back, ".."+string(filepath.Separator))
}

func strList(v *ir.Value) []string {
	var out []string
	for _, e := range v.AsArray() {
		if e != nil && e.Kind == ir.KindString {
			out = append(out, e.AsString())
		}
	}
	return out
}

// LoadDecomposition parses and validates decomposition.yaml against the
// design's modelith model, Architecture Contract, and contract machines.
// Validation failures are returned as a joined error listing every finding.
func LoadDecomposition(design string) (*Decomposition, error) {
	data, err := os.ReadFile(DecompositionPath(design))
	if err != nil {
		return nil, fmt.Errorf("pack: cannot read decomposition.yaml: %w", err)
	}
	v, err := ir.LoadYAML(data)
	if err != nil || v.AsObject() == nil {
		return nil, fmt.Errorf("pack: decomposition.yaml is not a yaml mapping")
	}
	o := v.AsObject()
	ver := o.Get2("decomposition_version")
	if ver == nil || ver.Kind != ir.KindNumber || string(ver.AsNumber()) != "1" {
		return nil, fmt.Errorf("pack: decomposition_version must be 1")
	}
	var d Decomposition
	d.Version = 1
	for _, sv := range o.Get2("subsystems").AsArray() {
		so := sv.AsObject()
		if so == nil {
			return nil, fmt.Errorf("pack: subsystems entries must be mappings")
		}
		d.Subsystems = append(d.Subsystems, Subsystem{
			ID:                  so.GetString("id"),
			Owns:                strList(so.Get2("owns")),
			Components:          strList(so.Get2("components")),
			Boundaries:          strList(so.Get2("boundaries")),
			ContractMachine:     so.GetString("contract_machine"),
			DelegatedInvariants: strList(so.Get2("delegated_invariants")),
			ChildDesign:         so.GetString("child_design"),
		})
	}

	var errs []string
	report := func(format string, args ...interface{}) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}
	if len(d.Subsystems) < 2 {
		report("a decomposition needs at least two subsystems (one subsystem is just the design itself)")
	}
	seen := map[string]bool{}
	for _, s := range d.Subsystems {
		if s.ID == "" {
			report("subsystem without an id")
			continue
		}
		if !subsystemIDRe.MatchString(s.ID) {
			report("subsystem id %s is not a bare name (letters, digits, underscore, hyphen); ids become path segments under packs/ and must not traverse", ir.Repr(s.ID))
			continue
		}
		if seen[s.ID] {
			report("duplicate subsystem id %s", ir.Repr(s.ID))
		}
		seen[s.ID] = true
		if len(s.Owns) == 0 {
			report("subsystem %s owns no entities", ir.Repr(s.ID))
		}
		if s.ContractMachine == "" {
			report("subsystem %s declares no contract_machine", ir.Repr(s.ID))
		} else if !pathInsideDesign(design, s.ContractMachine) {
			report("subsystem %s contract_machine %s resolves outside the design directory", ir.Repr(s.ID), ir.Repr(s.ContractMachine))
		}
		if filepath.IsAbs(s.ChildDesign) {
			report("subsystem %s child_design %s must be a relative path", ir.Repr(s.ID), ir.Repr(s.ChildDesign))
		}
	}

	// ownership: every owned entity exists; every entity owned exactly once
	dm, dmErr := loadModelith(design)
	if dmErr != nil {
		report("%s", dmErr.Error())
	} else {
		entities := dm.AsObject().GetObject("entities")
		owner := map[string]string{}
		for _, s := range d.Subsystems {
			for _, e := range s.Owns {
				if !entities.Has(e) {
					report("subsystem %s owns unknown entity %s", ir.Repr(s.ID), ir.Repr(e))
					continue
				}
				if prev, ok := owner[e]; ok {
					report("entity %s is owned by both %s and %s; ownership must be exactly-once", ir.Repr(e), ir.Repr(prev), ir.Repr(s.ID))
				}
				owner[e] = s.ID
			}
		}
		for _, e := range entities.Keys() {
			if _, ok := owner[e]; !ok {
				report("entity %s is owned by no subsystem; every entity needs exactly one owner", ir.Repr(e))
			}
		}
		// delegated invariants must exist (entity-level or top-level)
		known := invariantIDs(dm)
		for _, s := range d.Subsystems {
			for _, iid := range s.DelegatedInvariants {
				if !known[iid] {
					report("subsystem %s delegates unknown invariant %s", ir.Repr(s.ID), ir.Repr(iid))
				}
			}
		}
	}

	// boundaries must exist in the Architecture Contract
	bids := contractBoundaryIDs(design)
	for _, s := range d.Subsystems {
		for _, b := range s.Boundaries {
			if !bids[b] {
				report("subsystem %s references boundary %s, which the Architecture Contract does not declare", ir.Repr(s.ID), ir.Repr(b))
			}
		}
	}

	// contract machines must exist, lint clean, and stay inside the contract
	// subset (on-transitions and finals only: a contract is an interface
	// protocol, not an implementation)
	for _, s := range d.Subsystems {
		if s.ContractMachine == "" || !pathInsideDesign(design, s.ContractMachine) {
			continue
		}
		mp := filepath.Join(design, s.ContractMachine)
		m, mErr := ir.LoadMachineJSON(mp)
		if mErr != nil {
			report("subsystem %s contract machine: %s", ir.Repr(s.ID), mErr.Error())
			continue
		}
		lintErrs, _, _, _ := lint.LintMachine(m, filepath.Base(mp))
		for _, le := range lintErrs {
			report("subsystem %s contract machine: %s", ir.Repr(s.ID), le)
		}
		for _, st := range ir.WalkStates(m.AsObject().Get2("states"), "") {
			so := st.Node.AsObject()
			if so == nil {
				continue
			}
			for _, forbidden := range []string{"invoke", "after", "always", "states"} {
				if so.Get2(forbidden) != nil {
					report("subsystem %s contract machine: state %s uses %s; contract machines are restricted to plain on-transitions and final states (the abstract protocol, not the implementation)",
						ir.Repr(s.ID), st.Path, ir.Repr(forbidden))
				}
			}
		}
	}

	if len(errs) > 0 {
		sort.Strings(errs)
		return nil, fmt.Errorf("pack: %s", strings.Join(errs, "; "))
	}
	return &d, nil
}

func loadModelith(design string) (*ir.Value, error) {
	entries, _ := os.ReadDir(design)
	var paths []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".modelith.yaml") {
			paths = append(paths, filepath.Join(design, e.Name()))
		}
	}
	if len(paths) != 1 {
		return nil, fmt.Errorf("expected exactly one *.modelith.yaml in %s, found %d", design, len(paths))
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		return nil, err
	}
	v, err := ir.LoadYAML(data)
	if err != nil || v.AsObject() == nil {
		return nil, fmt.Errorf("%s is not a yaml mapping", filepath.Base(paths[0]))
	}
	return v, nil
}

func invariantIDs(dm *ir.Value) map[string]bool {
	out := map[string]bool{}
	o := dm.AsObject()
	for _, i := range o.Get2("invariants").AsArray() {
		if id := i.AsObject().GetString("id"); id != "" {
			out[id] = true
		}
	}
	entities := o.GetObject("entities")
	for _, en := range entities.Keys() {
		for _, i := range entities.Get2(en).AsObject().Get2("invariants").AsArray() {
			if id := i.AsObject().GetString("id"); id != "" {
				out[id] = true
			}
		}
	}
	return out
}

func contractBoundaryIDs(design string) map[string]bool {
	out := map[string]bool{}
	text, err := os.ReadFile(filepath.Join(design, "ARCHITECTURE.md"))
	if err != nil {
		return out
	}
	// the same heading-anchored locator G2 uses (ir.ContractFence); a laxer
	// locator here once rejected every boundary on a design G2 passed
	fence, ok := ir.ContractFence(string(text))
	if !ok {
		return out
	}
	v, err := ir.LoadYAML([]byte(fence))
	if err != nil || v.AsObject() == nil {
		return out
	}
	for _, b := range v.AsObject().Get2("boundaries").AsArray() {
		if id := b.AsObject().GetString("id"); id != "" {
			out[id] = true
		}
	}
	return out
}

// EventRows parses the parent's event-contract table.
func EventRows(design string) []EventRow {
	data, err := os.ReadFile(filepath.Join(design, "ARCHITECTURE.md"))
	if err != nil {
		return nil
	}
	var out []EventRow
	for _, tbl := range ir.ParseMdTables(string(data)) {
		hl := strings.ToLower(strings.Join(tbl.Header, " "))
		if !strings.Contains(hl, "producer") || !strings.Contains(hl, "consumer") || !strings.Contains(hl, "delivery") {
			continue
		}
		ei := ir.FindCol(tbl.Header, "event")
		pi := ir.FindCol(tbl.Header, "producer")
		ci := ir.FindCol(tbl.Header, "consumer")
		yi := ir.FindCol(tbl.Header, "payload")
		di := ir.FindCol(tbl.Header, "delivery")
		oi := ir.FindCol(tbl.Header, "ordering")
		ki := ir.FindCol(tbl.Header, "dedupe")
		cell := func(r []string, i int) string {
			if i >= 0 && i < len(r) {
				return ir.CleanCell(r[i])
			}
			return ""
		}
		for _, r := range tbl.Rows {
			out = append(out, EventRow{
				Event: cell(r, ei), Producer: cell(r, pi), Consumer: cell(r, ci),
				Payload: cell(r, yi), Delivery: cell(r, di), Ordering: cell(r, oi), Dedupe: cell(r, ki),
			})
		}
		break
	}
	return out
}

// GeneratePacks builds every subsystem's pack in memory:
// subsystem id -> filename -> content. Pure given the design dir contents.
func GeneratePacks(design string) (map[string]map[string]string, error) {
	d, err := LoadDecomposition(design)
	if err != nil {
		return nil, err
	}
	dm, err := loadModelith(design)
	if err != nil {
		return nil, fmt.Errorf("pack: %w", err)
	}
	events := EventRows(design)
	out := map[string]map[string]string{}
	for _, s := range d.Subsystems {
		files := map[string]string{}

		// 1. the domain slice
		files["domain.modelith.yaml"] = sliceModelith(dm, s)

		// 2. the contract machine (verbatim) + its TLA module
		mp := filepath.Join(design, s.ContractMachine)
		raw, rerr := os.ReadFile(mp)
		if rerr != nil {
			return nil, fmt.Errorf("pack: %s: %w", s.ID, rerr)
		}
		cname := strings.TrimSuffix(filepath.Base(mp), ".machine.json")
		files[cname+".machine.json"] = string(raw)
		mid, tlaBody, cfgBody, gerr := tla.Generate(mp)
		if gerr != nil {
			return nil, fmt.Errorf("pack: %s contract machine: %w", s.ID, gerr)
		}
		files[mid+".tla"] = tlaBody
		files[mid+".cfg"] = cfgBody

		// 3. the boundary event contracts (rows this subsystem produces or consumes)
		files["events.md"] = eventsSlice(events, s)

		// 4. the manifest, hash last: the content_hash covers every file
		// including the manifest itself (normalized minus the hash line)
		body := manifestBody(s, mid)
		files["pack.yaml"] = body
		files["pack.yaml"] = body + "content_hash: " + ContentHash(files) + "\n"
		out[s.ID] = files
	}
	return out, nil
}

// ContentHash hashes the pack files deterministically. The manifest
// (pack.yaml) is covered too, normalized with its own content_hash line
// removed (the hash is written into the manifest, so the manifest is hashed
// minus that line). Excluding the manifest entirely once let a child edit
// pack/pack.yaml (deleting a delegated invariant) and still pass G5.
func ContentHash(files map[string]string) string {
	var names []string
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, n := range names {
		body := files[n]
		if n == "pack.yaml" {
			body = stripContentHashLine(body)
		}
		fmt.Fprintf(h, "%s\n%d\n%s\n", n, len(body), body)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// stripContentHashLine removes the manifest's content_hash line so the hash
// written into the manifest does not feed back into itself.
func stripContentHashLine(manifest string) string {
	lines := strings.Split(manifest, "\n")
	var kept []string
	for _, l := range lines {
		if strings.HasPrefix(l, "content_hash:") {
			continue
		}
		kept = append(kept, l)
	}
	return strings.Join(kept, "\n")
}

// manifestBody builds the manifest without its content_hash line; the caller
// appends the line after hashing the full file set.
func manifestBody(s Subsystem, contractModule string) string {
	var b strings.Builder
	b.WriteString("# GENERATED by machinery pack. The frozen interface between the parent\n")
	b.WriteString("# design and this subsystem's child design. DO NOT EDIT: regenerate at the\n")
	b.WriteString("# parent (machinery pack generate) and re-copy.\n")
	b.WriteString("pack_version: 1\n")
	b.WriteString("subsystem: " + s.ID + "\n")
	b.WriteString("contract_module: " + contractModule + "\n")
	writeList := func(key string, xs []string) {
		if len(xs) == 0 {
			b.WriteString(key + ": []\n")
			return
		}
		b.WriteString(key + ":\n")
		for _, x := range xs {
			b.WriteString("  - " + x + "\n")
		}
	}
	writeList("owns", s.Owns)
	writeList("components", s.Components)
	writeList("boundaries", s.Boundaries)
	writeList("delegated_invariants", s.DelegatedInvariants)
	return b.String()
}

func eventsSlice(events []EventRow, s Subsystem) string {
	comp := map[string]bool{}
	for _, c := range s.Components {
		comp[c] = true
	}
	var b strings.Builder
	b.WriteString("# Boundary event contracts: " + s.ID + "\n\n")
	b.WriteString("GENERATED by machinery pack from the parent event-contract table. Every\n")
	b.WriteString("event this subsystem consumes or produces crosses its boundary under these\n")
	b.WriteString("contracts; there are no other cross-boundary events.\n\n")
	b.WriteString("| event | direction | peer | payload | delivery | ordering | dedupe |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")
	n := 0
	for _, e := range events {
		prod, cons := comp[e.Producer], comp[e.Consumer]
		if !prod && !cons {
			continue
		}
		dir, peer := "consumes", e.Producer
		if prod {
			dir, peer = "produces", e.Consumer
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n",
			e.Event, dir, peer, e.Payload, e.Delivery, e.Ordering, e.Dedupe)
		n++
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "Boundary events: %d\n", n)
	return b.String()
}

// sliceModelith emits the subsystem's domain slice: owned entities verbatim,
// the enums their attributes reference, the delegated invariants as top-level
// invariants, and a foreign: list naming out-of-slice entities.
func sliceModelith(dm *ir.Value, s Subsystem) string {
	o := dm.AsObject()
	entities := o.GetObject("entities")
	owns := map[string]bool{}
	for _, e := range s.Owns {
		owns[e] = true
	}
	// enums referenced by owned entities' attribute types
	enums := o.GetObject("enums")
	usedEnums := map[string]bool{}
	for _, en := range entities.Keys() {
		if !owns[en] {
			continue
		}
		for _, a := range entities.Get2(en).AsObject().Get2("attributes").AsArray() {
			t := a.AsObject().GetString("type")
			if enums.Has(t) {
				usedEnums[t] = true
			}
		}
	}
	var b strings.Builder
	b.WriteString("# GENERATED by machinery pack: the domain slice owned by subsystem '" + s.ID + "'.\n")
	b.WriteString("# The child design extends this INTERNALLY; the entities, enum values, and\n")
	b.WriteString("# delegated invariants below are the frozen public shape. DO NOT EDIT.\n")
	b.WriteString("kind: " + yamlScalar(o.Get2("kind")) + "\n")
	b.WriteString("version: " + yamlScalar(o.Get2("version")) + "\n")
	b.WriteString("title: " + yamlQuote(o.GetString("title")+" / "+s.ID) + "\n")
	b.WriteString("enums:\n")
	for _, en := range enums.Keys() {
		if !usedEnums[en] {
			continue
		}
		emitYAML(&b, 1, en, ir.ObjectValue(enums.Get2(en).AsObject()))
	}
	b.WriteString("entities:\n")
	for _, en := range entities.Keys() {
		if !owns[en] {
			continue
		}
		emitYAML(&b, 1, en, entities.Get2(en))
	}
	// delegated invariants restated at top level so Gx traces them in the child
	b.WriteString("invariants:\n")
	inv := delegatedInvariantDefs(dm, s.DelegatedInvariants)
	if len(inv) == 0 {
		b.WriteString("  []\n")
	}
	for _, iv := range inv {
		emitYAMLListItem(&b, 1, iv)
	}
	var foreign []string
	for _, en := range entities.Keys() {
		if !owns[en] {
			foreign = append(foreign, en)
		}
	}
	sort.Strings(foreign)
	b.WriteString("scenarios: []\n")
	if len(foreign) > 0 {
		b.WriteString("# foreign entities (owned elsewhere; reference only): " + strings.Join(foreign, ", ") + "\n")
	}
	return b.String()
}

// delegatedInvariantDefs finds the full invariant definitions by id, wherever
// they live (top-level or entity-level).
func delegatedInvariantDefs(dm *ir.Value, ids []string) []*ir.Value {
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	var out []*ir.Value
	take := func(list *ir.Value) {
		for _, i := range list.AsArray() {
			if want[i.AsObject().GetString("id")] {
				out = append(out, i)
			}
		}
	}
	o := dm.AsObject()
	take(o.Get2("invariants"))
	entities := o.GetObject("entities")
	for _, en := range entities.Keys() {
		take(entities.Get2(en).AsObject().Get2("invariants"))
	}
	return out
}

// WritePacks generates and writes design/packs/<id>.pack/ for every subsystem.
func WritePacks(design string) ([]string, error) {
	packs, err := GeneratePacks(design)
	if err != nil {
		return nil, err
	}
	var ids []string
	for id := range packs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		dir := filepath.Join(design, "packs", id+".pack")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		var names []string
		for n := range packs[id] {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			if err := os.WriteFile(filepath.Join(dir, n), []byte(packs[id][n]), 0o644); err != nil {
				return nil, err
			}
		}
	}
	return ids, nil
}
