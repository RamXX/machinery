// Action-ownership table (G2, opt-in by table presence). "Every Modelith
// action maps to an owning component" has been an attested line since the c4
// reference was written, yet both sides are closed sets: the model enumerates
// the actions and workspace.dsl enumerates the components. This check gives
// the mapping an artifact: a markdown table in ARCHITECTURE.md whose header
// names an action column and an owner column. Adoption is per design (an
// absent table carries no obligation, exactly like READS, CLAUSES, and the
// declared embeds); once the table exists, coverage is total in both
// directions: every model action appears exactly once (or is waived), every
// named owner resolves, and every row's action exists in the model.

package gates

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

var (
	// unownedWaiverRe waives one row's OWNER: the action deliberately has no
	// single owning component. Distinct from the placement and contract
	// waivers, as always: one generic token would let an answer to one
	// question silently discharge another.
	unownedWaiverRe = regexp.MustCompile(`\(unowned:\s*([^)]*)\)`)
	// ownershipActRe is the Entity.action grammar the surface ledger already
	// uses for act keys.
	ownershipActRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*\.[A-Za-z_][A-Za-z0-9_-]*$`)
)

// ownershipTables returns every action-ownership table in text: a header
// naming an action column plus an owner column ("owning component" or
// "owner").
func ownershipTables(text string) []ir.MdTable {
	var out []ir.MdTable
	for _, tbl := range ir.ParseMdTables(text) {
		if colContaining(tbl.Header, "action") < 0 {
			continue
		}
		if colContaining(tbl.Header, "owning component") < 0 && colContaining(tbl.Header, "owner") < 0 {
			continue
		}
		out = append(out, tbl)
	}
	return out
}

// HasActionOwnership reports whether ARCHITECTURE.md carries an
// action-ownership table (the check's opt-in trigger).
func HasActionOwnership(text string) bool { return len(ownershipTables(text)) > 0 }

// checkActionOwnership holds the action-ownership table in both directions.
// els and declared are G2's already-parsed dsl elements and contract ids; the
// domain model is loaded here because G2 otherwise never needs it, and only
// once the table opted the design in.
func checkActionOwnership(g *Gate, design, text string, els map[string]dslEl, declared map[string]bool) {
	tables := ownershipTables(text)
	if len(tables) == 0 {
		return
	}
	model, err := readTargetActModel(design)
	if err != nil {
		g.Errs = append(g.Errs, "ARCHITECTURE.md has an action-ownership table but the domain model cannot be read ("+err.Error()+"); the table resolves against the model's actions")
		return
	}
	seen := map[string]int{}
	for _, tbl := range tables {
		ai := colContaining(tbl.Header, "action")
		oi := colContaining(tbl.Header, "owning component")
		if oi < 0 {
			oi = colContaining(tbl.Header, "owner")
		}
		for _, r := range tbl.Rows {
			if len(r) == 0 {
				continue
			}
			g.Count("action-ownership rows")
			actCell := ""
			if ai < len(r) {
				actCell = r[ai]
			}
			var acts []string
			malformed := false
			for _, seg := range strings.Split(strings.ReplaceAll(actCell, "`", " "), ",") {
				seg = strings.TrimSpace(seg)
				if seg == "" {
					continue
				}
				if !ownershipActRe.MatchString(seg) {
					g.Errs = append(g.Errs, "action-ownership row action "+ir.Repr(seg)+" is not an Entity.action; the action cell holds one or more Entity.action keys separated by commas")
					malformed = true
					continue
				}
				acts = append(acts, seg)
			}
			if len(acts) == 0 {
				if !malformed {
					g.Errs = append(g.Errs, "action-ownership row names no action; the action cell holds one or more Entity.action keys")
				}
				continue
			}
			for _, act := range acts {
				seen[act]++
				if seen[act] > 1 {
					g.Errs = append(g.Errs, "action-ownership table states "+ir.Repr(act)+" more than once; every action is owned exactly once")
				}
				if _, ok := model.index[act]; !ok {
					entity := strings.SplitN(act, ".", 2)[0]
					if !model.entities[entity] {
						g.Errs = append(g.Errs, "action-ownership row names unknown entity "+ir.Repr(entity)+"; the table resolves against the domain model")
					} else {
						g.Errs = append(g.Errs, "action-ownership row names unknown action "+ir.Repr(act)+"; the table resolves against the domain model")
					}
				}
			}
			ownerCell := ""
			if oi >= 0 && oi < len(r) {
				ownerCell = r[oi]
			}
			if m := unownedWaiverRe.FindStringSubmatch(ownerCell); m != nil {
				if strings.TrimSpace(m[1]) == "" {
					g.Errs = append(g.Errs, "action-ownership waiver for "+ir.Repr(acts[0])+" names no reason; write '(unowned: <reason>)'")
				} else {
					g.Count("actions ownership-waived", len(acts))
				}
				continue
			}
			owners := mitTokRe.FindAllStringSubmatch(parenAnnotationRe.ReplaceAllString(ownerCell, " "), -1)
			if len(owners) == 0 {
				g.Errs = append(g.Errs, "action-ownership row for "+ir.Repr(acts[0])+" names no owning component in backticks and no '(unowned: <reason>)' waiver")
				continue
			}
			if len(owners) != 1 {
				var names []string
				for _, m := range owners {
					names = append(names, "`"+m[1]+"`")
				}
				g.Errs = append(g.Errs, "action-ownership row for "+ir.Repr(acts[0])+" names "+fmt.Sprintf("%d", len(owners))+" owning components ("+strings.Join(names, ", ")+"); ownership is singular, so name exactly one owner or one '(unowned: <reason>)' waiver")
				continue
			}
			ok := true
			for _, m := range owners {
				tok := m[1]
				if _, isEl := els[tok]; !isEl && !declared[tok] {
					g.Errs = append(g.Errs, "action-ownership row for "+ir.Repr(acts[0])+" names owner `"+tok+"`, which is neither a workspace.dsl element nor a declared boundary or external")
					ok = false
				}
			}
			if ok {
				g.Count("actions owned", len(acts))
			}
		}
	}
	for _, a := range model.acts {
		if seen[a.key()] == 0 {
			g.Errs = append(g.Errs, "action "+ir.Repr(a.key())+" appears in no action-ownership row; every model action names its owning component, or is waived with '(unowned: <reason>)'")
		}
	}
}
