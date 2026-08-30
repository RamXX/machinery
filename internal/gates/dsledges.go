// Drawn-edge reconciliation (G2-c4). dslElements reads only the element
// DECLARATIONS of workspace.dsl; the DSL's own `a -> b` relationship lines
// were never read by any gate. The C4 diagram could therefore draw an edge the
// Architecture Contract denies, or an edge no rule covers at all, and G2 still
// passed: the picture and the contract are two statements about the same
// architecture and nothing held them to each other.
//
// This closes that gap with the SAME judgment G4-import passes on an observed
// code edge (allow / deny / baseline, wildcards matched by filepath.Match), so
// a drawn edge and an imported edge are never judged by two rules. The
// converse is deliberately NOT required: a diagram is legitimately partial, so
// an allow rule with no drawn edge is no finding.

package gates

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// dslRelRe matches a Structurizr relationship line: an identifier (possibly
// hierarchical), an arrow, and a second identifier. Anchored at the start of
// the line so a view expression ("include a -> b") or prose can never be read
// as a drawn relationship, and the tail must open a new token (a quote, a
// brace, or end of line) so a longer arrow-bearing construct is left alone
// rather than half-read.
var dslRelRe = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_.]*)\s*->\s*([A-Za-z_][A-Za-z0-9_.]*)\s*("|\{|$)`)

// dslRel is one drawn relationship with the 1-based line it was drawn on.
type dslRel struct {
	Src, Dst string
	Line     int
}

// dslRelationships parses the relationship lines of a workspace.dsl text.
func dslRelationships(text string) []dslRel {
	var out []dslRel
	for i, line := range strings.Split(text, "\n") {
		m := dslRelRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		out = append(out, dslRel{Src: m[1], Dst: m[2], Line: i + 1})
	}
	return out
}

// matchEdgeRule reports whether any rule matches (src, dst), with wildcards
// matched by filepath.Match. The one implementation G4-import and the
// drawn-edge check share, so a code edge and a diagram edge can never be
// judged by two different readings of the same rule list.
func matchEdgeRule(edges [][2]string, src, dst string) bool {
	for _, e := range edges {
		ok1, _ := filepath.Match(e[0], src)
		ok2, _ := filepath.Match(e[1], dst)
		if ok1 && ok2 {
			return true
		}
	}
	return false
}

// elementBindings maps each workspace.dsl element name to the contract id
// (boundary or external) bound to it. Boundaries bind through `element:` or,
// absent that, the last segment of their id (the same fallback the binding
// check uses); externals bind only through an explicit `element:`. A boundary
// wins over an external on the same element: the boundary is the design's own
// code, and G4 resolves it the same way.
func elementBindings(boundaries, externals []*ir.Value) map[string]string {
	out := map[string]string{}
	for _, x := range externals {
		xo := x.AsObject()
		if xo == nil || xo.GetString("id") == "" {
			continue
		}
		if el := xo.GetString("element"); el != "" {
			out[el] = xo.GetString("id")
		}
	}
	for _, b := range boundaries {
		bo := b.AsObject()
		if bo == nil || bo.GetString("id") == "" {
			continue
		}
		el := bo.GetString("element")
		if el == "" {
			el = lastSegment(bo.GetString("id"))
		}
		out[el] = bo.GetString("id")
	}
	return out
}

// checkDrawnEdges holds every relationship drawn in workspace.dsl to the
// contract's dependency_rules.
//
// Scope, in three layers, each with its own reason:
//
//   - An endpoint that is not a declared element is an ERROR: the diagram
//     names something the model does not declare, exactly as a rule naming an
//     undeclared boundary is an ERROR.
//   - An endpoint that IS declared but binds to no contract id carries no
//     obligation: a person, a system-context box, or a container the contract
//     never claimed is outside the dependency vocabulary, so no rule could
//     speak about it. The count keeps that remainder visible.
//   - An edge between two elements of the SAME contract id is not a crossing,
//     and G4 skips those too.
//
// The verdict on a bound edge is G4's: an edge both baselined and allowed is
// allowed (G2 already reports that contradiction once), a baselined edge is
// tolerated debt, a denied edge no allow rule covers is an ERROR, and an edge
// no rule covers at all is an undeclared ERROR. The ratchet has nothing to say
// here: it snapshots offender FILES, and a drawn edge has none.
func checkDrawnEdges(g *Gate, dslText string, els map[string]dslEl, boundaries, externals []*ir.Value, allow, deny, baseline [][2]string) {
	rels := dslRelationships(dslText)
	if len(rels) == 0 {
		return
	}
	bind := elementBindings(boundaries, externals)
	// a wildcard baseline rule would amnesty the whole edge space, so it never
	// matches here either; G2 owns the hard ERROR on the rule itself (GATE-7)
	baseline = dropWildcardEdges(baseline)
	resolve := func(name string) (string, bool) {
		if _, ok := els[name]; ok {
			return name, true
		}
		// a hierarchical identifier ("crm.repo") names the same element as its
		// last segment, the fallback the boundary binding already uses
		if seg := lastSegment(name); seg != name {
			if _, ok := els[seg]; ok {
				return seg, true
			}
		}
		return "", false
	}
	for _, r := range rels {
		// `this` is the DSL's self-reference keyword inside an element block,
		// not a name the model declares; it binds to no contract id and so
		// carries no obligation
		if r.Src == "this" || r.Dst == "this" {
			g.Count("drawn edges outside the contract vocabulary")
			continue
		}
		src, srcOK := resolve(r.Src)
		dst, dstOK := resolve(r.Dst)
		if !srcOK || !dstOK {
			for _, side := range []struct {
				name string
				ok   bool
			}{{r.Src, srcOK}, {r.Dst, dstOK}} {
				if !side.ok {
					g.Errs = append(g.Errs, fmt.Sprintf("workspace.dsl:%d: relationship %s -> %s names %s, which no element declares", r.Line, r.Src, r.Dst, ir.Repr(side.name)))
				}
			}
			continue
		}
		g.Count("drawn relationships")
		srcID, dstID := bind[src], bind[dst]
		if srcID == "" || dstID == "" {
			g.Count("drawn edges outside the contract vocabulary")
			continue
		}
		if srcID == dstID {
			continue // not a crossing, as in G4
		}
		denied := matchEdgeRule(deny, srcID, dstID)
		allowed := matchEdgeRule(allow, srcID, dstID)
		baselined := matchEdgeRule(baseline, srcID, dstID)
		switch {
		case baselined && allowed:
			// G2 reports the allow+baseline contradiction once already; judge
			// as allowed here so the finding is not duplicated per edge
			g.Count("drawn edges verified")
		case baselined:
			// tolerated debt, exactly as in G4: the deny records the intent
			// while the baseline records the amnesty being burned down
			g.Count("drawn edges baselined")
		case denied && !allowed:
			g.Errs = append(g.Errs, fmt.Sprintf("workspace.dsl:%d: the diagram draws %s -> %s (%s -> %s), which the contract denies; either the diagram is wrong or the contract needs an explicit allow", r.Line, r.Src, r.Dst, srcID, dstID))
		case !allowed && !denied:
			g.Errs = append(g.Errs, fmt.Sprintf("workspace.dsl:%d: the diagram draws %s -> %s (%s -> %s), which no dependency rule covers; add an explicit allow or deny to the contract", r.Line, r.Src, r.Dst, srcID, dstID))
		default:
			g.Count("drawn edges verified")
		}
	}
}
