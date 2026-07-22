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
	"strconv"
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
	// BoundaryEventsNone is the boundary_events: {none: "<reason>"} waiver: a
	// non-empty reason declares that NO event crosses this subsystem's
	// boundary on purpose. Without it, extracting zero boundary events for a
	// subsystem is a generation error (almost always a table defect).
	BoundaryEventsNone string
}

// Decomposition is the parsed, validated decomposition.yaml.
type Decomposition struct {
	Version    int
	Revision   int               // monotonic amendment counter (optional, default 1)
	Retained   map[string]string // top-level invariants the parent keeps enforcing, id -> reason
	Subsystems []Subsystem
}

// EventRow is one boundary event contract from the parent's event-contract
// table. The named cells are cleaned (backticks and parentheticals stripped);
// RawProducer/RawConsumer keep the original cell text so a validation finding
// can quote exactly what the author wrote, and Row is the 1-based data-row
// number for findings on rows whose event cell is empty.
type EventRow struct {
	Event, Producer, Consumer, Payload, Delivery, Ordering, Dedupe string
	RawProducer, RawConsumer                                       string
	Row                                                            int
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

// LooksLikeDesignDir reports whether the directory contains any machinery
// design signal: a *.modelith.yaml, a machines/ directory, or a
// decomposition.yaml. `machinery scale` refuses to measure a directory with
// none of these; measuring emptiness once produced a confident "single-run
// design" recommendation for a directory that was not a design at all.
func LooksLikeDesignDir(design string) bool {
	if entries, err := os.ReadDir(design); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".modelith.yaml") {
				return true
			}
		}
	}
	if st, err := os.Stat(filepath.Join(design, "machines")); err == nil && st.IsDir() {
		return true
	}
	return HasDecomposition(design)
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
	var errs []string
	report := func(format string, args ...interface{}) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}
	var d Decomposition
	d.Version = 1
	// revision is the amendment-visibility counter: it flows into every
	// manifest as pack_revision so a child can see rev N -> N+1 without
	// diffing bytes. Optional; a design without one is at revision 1.
	d.Revision = 1
	if rv := o.Get2("revision"); rv != nil {
		n := -1
		if rv.Kind == ir.KindNumber {
			if parsed, perr := strconv.Atoi(string(rv.AsNumber())); perr == nil {
				n = parsed
			}
		}
		if n < 1 {
			report("revision must be an integer >= 1 (the monotonic amendment counter emitted into every pack as pack_revision)")
		} else {
			d.Revision = n
		}
	}
	// retained: the top-level invariants the parent keeps enforcing itself,
	// each with a reason. Everything top-level that is neither delegated nor
	// retained is enforced by nobody (a decomposed parent has no machines).
	d.Retained = map[string]string{}
	if rv := o.Get2("retained"); rv != nil {
		ro := rv.AsObject()
		if ro == nil {
			report("retained must be a mapping of invariant id to a non-empty reason string")
		} else {
			for _, k := range ro.Keys() {
				val := ro.Get2(k)
				if val == nil || val.Kind != ir.KindString || strings.TrimSpace(val.AsString()) == "" {
					report("retained invariant %s needs a non-empty reason string (why the parent keeps enforcing it)", ir.Repr(k))
					continue
				}
				d.Retained[k] = val.AsString()
			}
		}
	}
	for _, sv := range o.Get2("subsystems").AsArray() {
		so := sv.AsObject()
		if so == nil {
			return nil, fmt.Errorf("pack: subsystems entries must be mappings")
		}
		sub := Subsystem{
			ID:                  so.GetString("id"),
			Owns:                strList(so.Get2("owns")),
			Components:          strList(so.Get2("components")),
			Boundaries:          strList(so.Get2("boundaries")),
			ContractMachine:     so.GetString("contract_machine"),
			DelegatedInvariants: strList(so.Get2("delegated_invariants")),
			ChildDesign:         so.GetString("child_design"),
		}
		if be := so.Get2("boundary_events"); be != nil {
			sub.BoundaryEventsNone = parseBoundaryEventsWaiver(be, sub.ID, report)
		}
		d.Subsystems = append(d.Subsystems, sub)
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

	// components: claimed exactly once across subsystems. A duplicate claim
	// makes the event-table direction test (comp[e.Producer]) true in both
	// packs: the duplicating subsystem's pack flips a consumed event to
	// produces, silently deleting the handling obligation.
	compOwner := map[string]string{}
	for _, s := range d.Subsystems {
		for _, c := range s.Components {
			if prev, ok := compOwner[c]; ok {
				if prev == s.ID {
					report("subsystem %s lists component %s twice", ir.Repr(s.ID), ir.Repr(c))
				} else {
					report("component %s is claimed by both %s and %s; components must be claimed exactly once across subsystems (a duplicate claim flips boundary event directions in the duplicating pack)", ir.Repr(c), ir.Repr(prev), ir.Repr(s.ID))
				}
				continue
			}
			compOwner[c] = s.ID
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
		delegatedTo := map[string]string{}
		for _, s := range d.Subsystems {
			for _, iid := range s.DelegatedInvariants {
				if !known[iid] {
					report("subsystem %s delegates unknown invariant %s", ir.Repr(s.ID), ir.Repr(iid))
				}
				delegatedTo[iid] = s.ID
			}
		}
		// retained ids must exist and must not also be delegated (enforcement
		// needs exactly one owner)
		for iid := range d.Retained {
			if !known[iid] {
				report("retained names unknown invariant %s", ir.Repr(iid))
			}
			if sub, ok := delegatedTo[iid]; ok {
				report("invariant %s is both retained and delegated to subsystem %s; enforcement must have exactly one owner", ir.Repr(iid), ir.Repr(sub))
			}
		}
		// every top-level (cross-entity) invariant must be delegated or
		// retained: a decomposed parent has no machines, so an invariant that
		// enters no pack is enforced by nothing while every gate stays green
		for _, iv := range dm.AsObject().Get2("invariants").AsArray() {
			iid := iv.AsObject().GetString("id")
			if iid == "" {
				continue
			}
			if _, ok := delegatedTo[iid]; ok {
				continue
			}
			if _, ok := d.Retained[iid]; ok {
				continue
			}
			report("top-level invariant %s is delegated to no subsystem and not declared retained; a decomposed parent has no machines to enforce it. Delegate it via a subsystem's delegated_invariants or keep it parent-enforced with retained: {%s: \"<reason>\"}", ir.Repr(iid), iid)
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

// parseBoundaryEventsWaiver validates the boundary_events value of one
// subsystem: the only accepted shape is a mapping with the single key none
// and a non-empty string reason. Anything else is reported and ignored, so a
// typo degrades into the strict default (zero events fail) instead of a
// silent waiver.
func parseBoundaryEventsWaiver(be *ir.Value, id string, report func(string, ...interface{})) string {
	beo := be.AsObject()
	if beo != nil && beo.Len() == 1 {
		if rv := beo.Get2("none"); rv != nil && rv.Kind == ir.KindString && strings.TrimSpace(rv.AsString()) != "" {
			// the reason is written verbatim into the generated events.md; an
			// embedded newline could forge its "Boundary events: N" count line
			if strings.ContainsAny(rv.AsString(), "\n\r") {
				report("subsystem %s boundary_events waiver reason must be a single line; an embedded newline can forge the generated events.md count line", ir.Repr(id))
				return ""
			}
			return rv.AsString()
		}
	}
	report("subsystem %s boundary_events must be a mapping with the single key none and a non-empty reason string (boundary_events: {none: \"<reason>\"})", ir.Repr(id))
	return ""
}

// contractBoundaries parses the Architecture Contract fence (the same
// heading-anchored locator G2 uses, ir.ContractFence; a laxer locator here
// once rejected every boundary on a design G2 passed) and returns the
// boundary mappings. Missing file or fence yields nil.
func contractBoundaries(design string) []*ir.Object {
	text, err := os.ReadFile(filepath.Join(design, "ARCHITECTURE.md"))
	if err != nil {
		return nil
	}
	fence, ok := ir.ContractFence(string(text))
	if !ok {
		return nil
	}
	v, err := ir.LoadYAML([]byte(fence))
	if err != nil || v.AsObject() == nil {
		return nil
	}
	var out []*ir.Object
	for _, b := range v.AsObject().Get2("boundaries").AsArray() {
		if bo := b.AsObject(); bo != nil {
			out = append(out, bo)
		}
	}
	return out
}

func contractBoundaryIDs(design string) map[string]bool {
	out := map[string]bool{}
	for _, b := range contractBoundaries(design) {
		if id := b.GetString("id"); id != "" {
			out[id] = true
		}
	}
	return out
}

// knownEventParticipants is every name a producer or consumer cell of the
// event-contract table may resolve to: the union of every subsystem's
// components: from decomposition.yaml and the Architecture Contract boundary
// elements. The boundary elements cover the contract-bound participants that
// own no entities and get no pack (a gateway, a ui). Contract externals do
// not qualify: events are produced and consumed by components; an external
// system that genuinely participates must be declared as a boundary.
func knownEventParticipants(design string, d *Decomposition) map[string]bool {
	out := map[string]bool{}
	for _, s := range d.Subsystems {
		for _, c := range s.Components {
			out[c] = true
		}
	}
	for _, b := range contractBoundaries(design) {
		if el := b.GetString("element"); el != "" {
			out[el] = true
		}
	}
	return out
}

// eventTable is the parsed parent event-contract table plus what the strict
// validation needs to know about its shape.
type eventTable struct {
	found    bool // a table with producer, consumer, and delivery columns exists
	eventCol bool // that table has an event column
	rows     []EventRow
}

// EventRows parses the parent's event-contract table leniently: cleaned
// cells, no validation. `machinery scale` uses it as a size signal on designs
// that may predate decomposition; pack generation goes through the strict
// path (parseEventTable + validateEventTable) instead.
func EventRows(design string) []EventRow {
	return parseEventTable(design).rows
}

// parseEventTable parses EVERY markdown table whose header names producer,
// consumer, and delivery, and concatenates their rows (row numbers run
// cumulatively across tables so findings stay addressable). Taking only the
// first matching table once silently excluded every later event-contract
// table from packs, validation, and the G5 regeneration while the packs still
// claimed boundary completeness.
func parseEventTable(design string) eventTable {
	data, err := os.ReadFile(filepath.Join(design, "ARCHITECTURE.md"))
	if err != nil {
		return eventTable{}
	}
	out := eventTable{eventCol: true}
	row := 0
	for _, tbl := range ir.ParseMdTables(string(data)) {
		hl := strings.ToLower(strings.Join(tbl.Header, " "))
		if !strings.Contains(hl, "producer") || !strings.Contains(hl, "consumer") || !strings.Contains(hl, "delivery") {
			continue
		}
		out.found = true
		ei := ir.FindCol(tbl.Header, "event")
		pi := ir.FindCol(tbl.Header, "producer")
		ci := ir.FindCol(tbl.Header, "consumer")
		yi := ir.FindCol(tbl.Header, "payload")
		di := ir.FindCol(tbl.Header, "delivery")
		oi := ir.FindCol(tbl.Header, "ordering")
		ki := ir.FindCol(tbl.Header, "dedupe")
		if ei < 0 {
			out.eventCol = false
		}
		cell := func(r []string, i int) string {
			if i >= 0 && i < len(r) {
				return ir.CleanCell(r[i])
			}
			return ""
		}
		raw := func(r []string, i int) string {
			if i >= 0 && i < len(r) {
				return r[i]
			}
			return ""
		}
		for _, r := range tbl.Rows {
			row++
			out.rows = append(out.rows, EventRow{
				Event: cell(r, ei), Producer: cell(r, pi), Consumer: cell(r, ci),
				Payload: cell(r, yi), Delivery: cell(r, di), Ordering: cell(r, oi), Dedupe: cell(r, ki),
				RawProducer: raw(r, pi), RawConsumer: raw(r, ci), Row: row,
			})
		}
	}
	if !out.found {
		return eventTable{}
	}
	return out
}

// eventCellFormat states the machine-checkable format contract once, so every
// finding teaches the same fix.
const eventCellFormat = "the format contract is one row per producer-consumer pair, exactly one component per producer/consumer cell, annotations only in parentheses, fan-outs expanded to one row per pair"

// validateEventTable enforces the format contract pack generation depends on.
// Every producer and consumer cell must resolve (after ir.CleanCell) to
// exactly one known participant; a cell that resolves to nothing or names
// several components is a finding naming the row and the offending cell text.
// Rows between two non-pack participants (a gateway publishing to a ui) are
// validated like every other row; they simply emit no pack rows. There are no
// silent drops of non-empty cells, ever: dropping them once shipped packs
// asserting boundary completeness while most contracts were missing.
func validateEventTable(design string, tbl eventTable, d *Decomposition) []string {
	allWaived := len(d.Subsystems) > 0
	for _, s := range d.Subsystems {
		if s.BoundaryEventsNone == "" {
			allWaived = false
		}
	}
	if !tbl.found {
		if allWaived {
			return nil
		}
		return []string{"ARCHITECTURE.md has no event-contract table (a markdown table whose header names producer, consumer, and delivery); a decomposed parent must declare its boundary event contracts"}
	}
	var errs []string
	if !tbl.eventCol {
		errs = append(errs, "the event-contract table has no event column; pack extraction needs one ("+eventCellFormat+")")
	}
	known := knownEventParticipants(design, d)
	names := make([]string, 0, len(known))
	for n := range known {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, r := range tbl.rows {
		where := fmt.Sprintf("event-contract row %d", r.Row)
		if r.Event != "" {
			where = fmt.Sprintf("%s (event %s)", where, ir.Repr(r.Event))
		} else if tbl.eventCol {
			errs = append(errs, where+": empty event cell; every row names its event")
		}
		for _, c := range []struct{ col, raw, clean string }{
			{"producer", r.RawProducer, r.Producer},
			{"consumer", r.RawConsumer, r.Consumer},
		} {
			if known[c.clean] {
				continue
			}
			switch {
			case c.clean == "":
				errs = append(errs, fmt.Sprintf("%s: empty %s cell; %s", where, c.col, eventCellFormat))
			case strings.ContainsAny(c.clean, ",/ \t") || strings.Contains(c.clean, "->") || strings.Contains(c.clean, "→"):
				errs = append(errs, fmt.Sprintf("%s: %s cell %s names more than one component; %s", where, c.col, ir.Repr(c.raw), eventCellFormat))
			default:
				errs = append(errs, fmt.Sprintf("%s: %s cell %s is not a known component (subsystem components plus Architecture Contract boundary elements: %s); %s", where, c.col, ir.Repr(c.raw), strings.Join(names, ", "), eventCellFormat))
			}
		}
	}
	return errs
}

// subsystemEvents filters the table down to the rows this subsystem produces
// or consumes.
func subsystemEvents(events []EventRow, s Subsystem) []EventRow {
	comp := map[string]bool{}
	for _, c := range s.Components {
		comp[c] = true
	}
	var out []EventRow
	for _, e := range events {
		if comp[e.Producer] || comp[e.Consumer] {
			out = append(out, e)
		}
	}
	return out
}

// GeneratePacks builds every subsystem's pack in memory:
// subsystem id -> filename -> content. Pure given the design dir contents.
//
// Extraction from the event-contract table is strict: the table must satisfy
// the format contract (validateEventTable) and every subsystem must extract
// at least one boundary event unless decomposition.yaml waives it with
// boundary_events: {none: "<reason>"}. A subsystem no event crosses into or
// out of is almost always a table defect, and silently generating an empty
// events.md that claims completeness is exactly the trust failure the gates
// exist to prevent.
func GeneratePacks(design string) (map[string]map[string]string, error) {
	d, err := LoadDecomposition(design)
	if err != nil {
		return nil, err
	}
	dm, err := loadModelith(design)
	if err != nil {
		return nil, fmt.Errorf("pack: %w", err)
	}
	tbl := parseEventTable(design)
	if errs := validateEventTable(design, tbl, d); len(errs) > 0 {
		return nil, fmt.Errorf("pack: %s", strings.Join(errs, "; "))
	}
	rowsBySub := map[string][]EventRow{}
	var evErrs []string
	// a row whose producer and consumer belong to the same subsystem has no
	// boundary direction: the pack emitter would pick "produces" because the
	// producer matches first, deleting a handling obligation. Reported
	// together with the zero-event findings so a mutation that self-loops
	// every row still names the subsystems it starved.
	owner := map[string]string{}
	for _, s := range d.Subsystems {
		for _, c := range s.Components {
			owner[c] = s.ID
		}
	}
	for _, r := range tbl.rows {
		po, co := owner[r.Producer], owner[r.Consumer]
		if po == "" || po != co {
			continue
		}
		where := fmt.Sprintf("event-contract row %d", r.Row)
		if r.Event != "" {
			where = fmt.Sprintf("%s (event %s)", where, ir.Repr(r.Event))
		}
		evErrs = append(evErrs, fmt.Sprintf("%s: producer %s and consumer %s both belong to subsystem %s; an intra-subsystem row is no boundary contract and its pack direction is ambiguous, fix the row or the components lists", where, ir.Repr(r.Producer), ir.Repr(r.Consumer), ir.Repr(po)))
	}
	for _, s := range d.Subsystems {
		rows := subsystemEvents(tbl.rows, s)
		rowsBySub[s.ID] = rows
		switch {
		case len(rows) == 0 && s.BoundaryEventsNone == "":
			evErrs = append(evErrs, fmt.Sprintf("subsystem %s extracts zero boundary events from the event-contract table; a subsystem no event crosses into or out of is almost always a table defect. If it is intentional, waive it in decomposition.yaml with boundary_events: {none: \"<reason>\"}", ir.Repr(s.ID)))
		case len(rows) > 0 && s.BoundaryEventsNone != "":
			evErrs = append(evErrs, fmt.Sprintf("subsystem %s declares boundary_events: none but %d event-contract rows name its components; remove the stale waiver", ir.Repr(s.ID), len(rows)))
		}
	}
	if len(evErrs) > 0 {
		return nil, fmt.Errorf("pack: %s", strings.Join(evErrs, "; "))
	}
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
		files["events.md"] = eventsSlice(rowsBySub[s.ID], s)

		// 4. the manifest, hash last: the content_hash covers every file
		// including the manifest itself (normalized minus the hash line)
		body := manifestBody(s, mid, d.Revision)
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
func manifestBody(s Subsystem, contractModule string, revision int) string {
	var b strings.Builder
	b.WriteString("# GENERATED by machinery pack. The frozen interface between the parent\n")
	b.WriteString("# design and this subsystem's child design. DO NOT EDIT: regenerate at the\n")
	b.WriteString("# parent (machinery pack generate) and re-copy.\n")
	b.WriteString("pack_version: 1\n")
	fmt.Fprintf(&b, "pack_revision: %d\n", revision)
	b.WriteString("subsystem: " + s.ID + "\n")
	b.WriteString("contract_module: " + contractModule + "\n")
	// list items go through yamlQuote: an id containing ": " left plain would
	// reparse as a mapping and vanish from the child's delegation list
	writeList := func(key string, xs []string) {
		if len(xs) == 0 {
			b.WriteString(key + ": []\n")
			return
		}
		b.WriteString(key + ":\n")
		for _, x := range xs {
			b.WriteString("  - " + yamlQuote(x) + "\n")
		}
	}
	writeList("owns", s.Owns)
	writeList("components", s.Components)
	writeList("boundaries", s.Boundaries)
	writeList("delegated_invariants", s.DelegatedInvariants)
	return b.String()
}

// eventsSlice renders the subsystem's boundary event contracts. The
// completeness sentence is a strong claim (there are no other cross-boundary
// events) and extraction is strict, so the claim is earned; a waived-empty
// subsystem gets the waiver reason instead, never an unearned claim over an
// empty table.
func eventsSlice(events []EventRow, s Subsystem) string {
	if len(events) == 0 && s.BoundaryEventsNone != "" {
		var b strings.Builder
		b.WriteString("# Boundary event contracts: " + s.ID + "\n\n")
		b.WriteString("GENERATED by machinery pack from the parent event-contract table. No event\n")
		b.WriteString("crosses this subsystem's boundary; decomposition.yaml waives it with reason:\n")
		b.WriteString(s.BoundaryEventsNone + "\n\n")
		b.WriteString("Boundary events: 0 (waived)\n")
		return b.String()
	}
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
		dir, peer := "consumes", e.Producer
		if comp[e.Producer] {
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

// boundaryCountRe matches the count line every generated events.md ends with
// (waived files carry "0 (waived)"; the digits still parse).
var boundaryCountRe = regexp.MustCompile(`(?m)^Boundary events: (\d+)`)

// CountBoundaryEvents reads the count line of a generated events.md, so G5
// can print per-pack boundary-event counts and a zero is visible in every
// gate run. Returns -1 when the line is absent (not a generated events file).
// Anchors on the LAST match: the generated count line always comes last, so
// text injected earlier in the file (a forged waiver reason) cannot shadow it.
func CountBoundaryEvents(eventsMD string) int {
	ms := boundaryCountRe.FindAllStringSubmatch(eventsMD, -1)
	if len(ms) == 0 {
		return -1
	}
	n, err := strconv.Atoi(ms[len(ms)-1][1])
	if err != nil {
		return -1
	}
	return n
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
	// a parent model without a title yields just the subsystem id; the old
	// unconditional concatenation emitted the nonsense title " / core"
	title := s.ID
	if pt := o.GetString("title"); pt != "" {
		title = pt + " / " + s.ID
	}
	b.WriteString("title: " + yamlQuote(title) + "\n")
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
