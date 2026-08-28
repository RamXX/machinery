// Branch-order annotation (S8 of the dogfood systemic findings, as amended).
// G3 checks guard shadowing structurally but cannot see that one guard's
// truth set contains another's, and a round-7 BLOCKER was exactly an
// inverted precedence between two overlapping guards on one always list.
// Full overlap detection needs semantics; the cheap mechanical half rides
// the traceability spine: when two guarded branches of ONE branch list cite
// overlapping invariant ids in their matrix maps-to columns, the guards
// argue about the same law and their order carries meaning, so the state
// must say why the order is correct in a `_branch_order` annotation.
// Warnings tier.

package gates

import (
	"regexp"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// invariantIDToken matches the kebab-case invariant-id vocabulary inside a
// maps-to cell (`coverage-gap-subject-backed`).
var invariantIDToken = regexp.MustCompile("`([a-z][a-z0-9]*(?:-[a-z0-9]+)+)`")

// guardInvariants parses the matrix named-unit table for guard rows and
// returns each guard's cited invariant-id set from its maps-to column.
// Invariants cited by three or more of the machine's guards are its AMBIENT
// law (privileged-actions-audited, tenant-scoped-data) and are excluded:
// sharing one says nothing about two branches arguing over the same rule,
// and counting them buried the real overlap signal under 35 false hits on
// the first dogfood calibration.
func guardInvariants(mtext string) map[string]map[string]bool {
	raw := map[string]map[string]bool{}
	freq := map[string]int{}
	for _, line := range strings.Split(mtext, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 4 {
			continue
		}
		name := strings.Trim(strings.TrimSpace(cells[1]), "`")
		kind := strings.TrimSpace(cells[2])
		if kind != "guard" || name == "" {
			continue
		}
		ids := map[string]bool{}
		for _, m := range invariantIDToken.FindAllStringSubmatch(line, -1) {
			ids[m[1]] = true
		}
		if len(ids) > 0 {
			raw[name] = ids
			for id := range ids {
				freq[id]++
			}
		}
	}
	out := map[string]map[string]bool{}
	for name, ids := range raw {
		kept := map[string]bool{}
		for id := range ids {
			if freq[id] < 3 {
				kept[id] = true
			}
		}
		if len(kept) > 0 {
			out[name] = kept
		}
	}
	return out
}

// CheckBranchOrder warns for every branch list whose guarded branches cite
// overlapping invariants while the state carries no `_branch_order` note.
func CheckBranchOrder(g *Gate, m *ir.Value, mtext, base string) {
	byGuard := guardInvariants(mtext)
	if len(byGuard) == 0 {
		return
	}
	checkList := func(state, event string, v *ir.Value, hasNote bool) {
		if v == nil || v.Kind != ir.KindArray {
			return
		}
		var guards []string
		for _, it := range v.AsArray() {
			if it == nil || it.Kind != ir.KindObject {
				continue
			}
			if gv := it.AsObject().GetString("guard"); gv != "" {
				guards = append(guards, gv)
			}
		}
		if len(guards) < 2 {
			return
		}
		for i := 0; i < len(guards); i++ {
			for j := i + 1; j < len(guards); j++ {
				a, b := byGuard[guards[i]], byGuard[guards[j]]
				var shared []string
				for id := range a {
					if b[id] {
						shared = append(shared, id)
					}
				}
				if len(shared) == 0 || hasNote {
					continue
				}
				sort.Strings(shared)
				g.Warns = append(g.Warns, base+": state "+state+" "+event+" orders guards "+guards[i]+" then "+guards[j]+", which share invariant "+strings.Join(shared, ", ")+" in their maps-to columns; when overlapping guards are ordered, say why the order is correct in a _branch_order note on the state")
				return
			}
		}
	}
	for _, s := range ir.WalkStates(m.AsObject().Get2("states"), "") {
		if s.Node == nil || s.Node.Kind != ir.KindObject {
			continue
		}
		so := s.Node.AsObject()
		hasNote := strings.TrimSpace(so.GetString("_branch_order")) != ""
		checkList(s.Name, "always", so.Get2("always"), hasNote)
		if on := so.GetObject("on"); on != nil {
			for _, k := range on.Keys() {
				checkList(s.Name, "on:"+k, on.Get2(k), hasNote)
			}
		}
	}
}
