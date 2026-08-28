// Annotation name-rot (S5 of the dogfood systemic findings, as amended, v1
// scope). Machine _comment, _refusal, _ignores and _exhaustive strings and
// matrix prose carry normative narrative no gate reads, and it rots: ops.md
// once cited a machine state its machine never had. The mechanizable
// slice with near-zero false positives is the DOTTED REFERENCE: a backticked
// `Machine.state` (or `Machine.state.event`) token whose machine half names a
// committed machine must name a state (and event) that machine actually has.
// Single-token names stay unchecked in v1: prose legitimately backticks
// fields, knobs and other files' vocabulary, and a warning tier that cannot
// be held at zero teaches readers to ignore it.

package gates

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/lint"
)

// dottedRef matches `Name.part` or `Name.part.part` inside backticks, where
// Name is TitleCase (a machine basename) and the parts are identifier-shaped.
var dottedRef = regexp.MustCompile("`([A-Z][A-Za-z0-9]*)\\.([A-Za-z_][A-Za-z0-9_]*)(?:\\.([A-Za-z_][A-Za-z0-9_]*))?`")

type machineNames struct {
	states map[string]bool
	events map[string]bool
	units  map[string]bool // guards + actions + actors: dotted unit refs are legitimate
}

// collectMachineNames loads every machine's state and event vocabulary,
// keyed by the machine file's basename (the name prose uses). Where the
// design's model declares an entity of the same name, its attribute,
// action, and relationship vocabulary is admitted too: `Query.trace_ref`
// and `GapAnalysis.complete` are entity references prose makes constantly,
// and flagging them would bury the rot signal.
func collectMachineNames(design string) map[string]machineNames {
	out := map[string]machineNames{}
	for _, path := range sortedGlob(filepath.Join(design, "machines"), "*.machine.json") {
		m, err := ir.LoadMachineJSON(path)
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(path), ".machine.json")
		mn := machineNames{states: map[string]bool{}, events: map[string]bool{}, units: map[string]bool{}}
		for _, s := range ir.WalkStates(m.AsObject().Get2("states"), "") {
			mn.states[s.Name] = true
			if s.Node == nil || s.Node.Kind != ir.KindObject {
				continue
			}
			so := s.Node.AsObject()
			for _, key := range []string{"on", "_ignores", "_refusal"} {
				if ev := so.GetObject(key); ev != nil {
					for _, k := range ev.Keys() {
						mn.events[k] = true
					}
				}
			}
		}
		gset, aset, actset := lint.MachineUnitNames(m)
		for n := range gset {
			mn.units[n] = true
		}
		for n := range aset {
			mn.units[n] = true
		}
		for n := range actset {
			mn.units[n] = true
		}
		// model actions referenced dot-wise (Entity.action) share the event
		// vocabulary; anything the machine handles or ignores is above.
		out[name] = mn
	}
	silent := NewGate("namerot-model-load")
	if dm := loadModelith(design, silent); dm != nil {
		entities := dm.AsObject().GetObject("entities")
		if entities != nil {
			for _, ename := range entities.Keys() {
				mn, ok := out[ename]
				if !ok {
					continue
				}
				eo := entities.Get2(ename).AsObject()
				for _, a := range objSlice(eo.Get2("attributes")) {
					if n := a.AsObject().GetString("name"); n != "" {
						mn.units[n] = true
					}
				}
				for _, a := range objSlice(eo.Get2("actions")) {
					if n := a.AsObject().GetString("name"); n != "" {
						mn.units[n] = true
					}
				}
				for _, r := range objSlice(eo.Get2("relationships")) {
					if n := r.AsObject().GetString("entity"); n != "" {
						mn.units[strings.ToLower(n[:1])+n[1:]] = true
					}
				}
			}
		}
	}
	return out
}

// annotationStrings walks one machine's annotation strings: root and state
// _comment, every _refusal / _ignores / _exhaustive value, and transition
// description fields. Each is returned with a locator for the warning.
func annotationStrings(m *ir.Value, base string) map[string]string {
	out := map[string]string{}
	ro := m.AsObject()
	if c := ro.GetString("_comment"); c != "" {
		out[base+": _comment"] = c
	}
	for _, s := range ir.WalkStates(ro.Get2("states"), "") {
		if s.Node == nil || s.Node.Kind != ir.KindObject {
			continue
		}
		so := s.Node.AsObject()
		if c := so.GetString("_comment"); c != "" {
			out[base+": "+s.Name+"._comment"] = c
		}
		if c := so.GetString("_exhaustive"); c != "" {
			out[base+": "+s.Name+"._exhaustive"] = c
		}
		for _, key := range []string{"_refusal", "_ignores"} {
			if o := so.GetObject(key); o != nil {
				for _, k := range o.Keys() {
					if v := o.GetString(k); v != "" {
						out[base+": "+s.Name+"."+key+"["+k+"]"] = v
					}
				}
			}
		}
	}
	return out
}

// checkDottedRefs warns for every dotted reference into a known machine that
// names a state, event, or unit that machine does not have.
func checkDottedRefs(g *Gate, where, text string, universe map[string]machineNames) {
	for _, m := range dottedRef.FindAllStringSubmatch(text, -1) {
		machine, first, second := m[1], m[2], m[3]
		mn, known := universe[machine]
		if !known {
			continue // not a machine reference: entity fields, module paths, other vocabulary
		}
		// `X.machine.json`, `X.matrix.md`, `X.oracle.md` are FILE references,
		// not references into the machine's vocabulary.
		if first == "machine" || first == "matrix" || first == "oracle" {
			continue
		}
		g.Count("dotted references checked")
		if mn.states[first] || mn.events[first] || mn.units[first] {
			if second != "" && !mn.states[second] && !mn.events[second] && !mn.units[second] &&
				second != "on" && second != "onDone" && second != "onError" {
				g.Warns = append(g.Warns, where+": `"+machine+"."+first+"."+second+"` names no state, event, or unit of "+machine+"; the narrative has rotted or the name moved")
			}
			continue
		}
		// model-action dotted form Entity.action where the machine handles the
		// action as an event is covered above. A snake_case remainder that
		// resolves nowhere is usually a NAMED-ACT vocabulary machinery does
		// not model (dotted requesting-action ids like Review.route_review),
		// so it lands in the notes tier; a state-shaped remainder that
		// resolves nowhere is the rot class this check exists for, and warns.
		if strings.Contains(first, "_") {
			g.Notes = append(g.Notes, where+": `"+machine+"."+first+"` resolves to no modeled name of "+machine+" (a named-act vocabulary, or rot; verify by eye)")
			continue
		}
		g.Warns = append(g.Warns, where+": `"+machine+"."+first+"` names no state, event, or unit of "+machine+"; the narrative has rotted or the name moved")
	}
}

// CheckNameRot runs the dotted-reference audit over every machine's
// annotation strings and every matrix file's prose. Called from
// CheckMachines; warnings only.
func CheckNameRot(g *Gate, design string) {
	universe := collectMachineNames(design)
	if len(universe) == 0 {
		return
	}
	files := sortedGlob(filepath.Join(design, "machines"), "*.machine.json")
	for _, path := range files {
		m, err := ir.LoadMachineJSON(path)
		if err != nil {
			continue
		}
		base := filepath.Base(path)
		anns := annotationStrings(m, base)
		var keys []string
		for k := range anns {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			checkDottedRefs(g, k, anns[k], universe)
		}
	}
	for _, path := range sortedGlob(filepath.Join(design, "machines"), "*.matrix.md") {
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		base := filepath.Base(path)
		for i, line := range strings.Split(string(body), "\n") {
			checkDottedRefs(g, base+":"+strconv.Itoa(i+1), line, universe)
		}
	}
}
