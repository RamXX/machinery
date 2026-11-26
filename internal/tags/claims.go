package tags

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/project"
)

// Claims is what the tags and markers of a project claim, read statically:
// the reading of stage trace (sections 8 and 8.1). It says that a test names
// an id, never that the test passed.
type Claims struct {
	// Served are the ids a serves tag or marker names, an action or not.
	Served map[string]bool
	// Tests maps an id to the test files that claim it, sorted.
	Tests map[string][]string
}

// Read scans the code and test files of the project and returns what they
// claim. Without a trace block nothing is claimed.
func Read(p *project.Project, d *design.Design) (*Claims, error) {
	code, err := CodeFiles(p)
	if err != nil {
		return nil, err
	}
	tests, err := TestFiles(p)
	if err != nil {
		return nil, err
	}
	found, err := Scan(p, append(append([]string{}, code...), tests...))
	if err != nil {
		return nil, err
	}
	return ClaimsOf(d, found), nil
}

// ClaimsOf resolves the tags against the design. A proof obligation is never
// claimed, and a proposed or retired id yields no obligation to claim.
func ClaimsOf(d *design.Design, found []Tag) *Claims {
	c := &Claims{Served: map[string]bool{}, Tests: map[string][]string{}}
	ids := d.DeclaredIDs()
	scenarios := d.ScenarioIDs()
	for _, t := range found {
		for _, tok := range t.IDs {
			if !t.Scenario && strings.HasPrefix(tok, "pf:") {
				continue
			}
			for _, id := range Resolve(t, tok, ids, scenarios) {
				switch {
				case d.Status(id) != "":
				case t.Kind == Serves:
					c.Served[id] = true
				case !contains(c.Tests[id], t.File):
					c.Tests[id] = append(c.Tests[id], t.File)
				}
			}
		}
	}
	for id := range c.Tests {
		sort.Strings(c.Tests[id])
	}
	return c
}

// ClaimKinds are the kinds a claim line counts, in the order it lists them.
// An fv id is proven by the formal stage and is in no claim line; an action
// is served, not claimed, and has a line of its own.
var ClaimKinds = []string{"tc", "sc", "inv", "fm", "cx", "ck"}

// Tally counts ids by kind.
type Tally struct {
	ByKind map[string]int
	Total  int
}

func (t *Tally) add(kind string) {
	if t.ByKind == nil {
		t.ByKind = map[string]int{}
	}
	t.ByKind[kind]++
	t.Total++
}

// TallyOf counts ids by their kind.
func TallyOf(ids []string) Tally {
	var t Tally
	for _, id := range ids {
		kind, _, _ := strings.Cut(id, ":")
		t.add(kind)
	}
	return t
}

// String renders "<n> (<kind> <n>, ...)", or "0".
func (t Tally) String() string {
	if t.Total == 0 {
		return "0"
	}
	var parts []string
	for _, k := range ClaimKinds {
		if n := t.ByKind[k]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", k, n))
		}
	}
	return fmt.Sprintf("%d (%s)", t.Total, strings.Join(parts, ", "))
}

// Doc renders the tally for a JSON report.
func (t Tally) Doc() map[string]any {
	byKind := map[string]any{}
	for k, n := range t.ByKind {
		byKind[k] = n
	}
	return map[string]any{"by_kind": byKind, "total": t.Total}
}

// Split is the owed ids by what stands behind each (section 14): planned
// by the project, proven by a receipt and still listed, named by a test with
// no receipt yet, named by no test; and the owed actions by whether code
// serves them.
type Split struct {
	Claimed, Unnamed, Planned, Proven Tally
	ActServed, ActUnserved            int
}

// SplitOwed counts the owed entries the design knows. proven reports an id
// that a valid pass receipt proves at the current commit.
func (c *Claims) SplitOwed(d *design.Design, proven func(id string) bool) Split {
	var s Split
	for _, e := range d.Owed {
		if !d.Known(e.ID) {
			continue
		}
		kind, _, _ := strings.Cut(e.ID, ":")
		switch {
		case kind == "act" && c.Served[e.ID]:
			s.ActServed++
		case kind == "act":
			s.ActUnserved++
		case !contains(ClaimKinds, kind):
		case e.PlannedAtAdoption():
			s.Planned.add(kind)
		case proven(e.ID):
			s.Proven.add(kind)
		case len(c.Tests[e.ID]) > 0:
			s.Claimed.add(kind)
		default:
			s.Unnamed.add(kind)
		}
	}
	return s
}

// Lines renders the claim lines, each after the prefix: the first two
// always, the others only when they count something.
func (s Split) Lines(prefix string) []string {
	lines := []string{
		fmt.Sprintf("%sclaimed by a test, no receipt yet: %s", prefix, s.Claimed),
		fmt.Sprintf("%sno test names it: %s", prefix, s.Unnamed),
	}
	if s.Planned.Total > 0 {
		lines = append(lines, fmt.Sprintf("%splanned: %s", prefix, s.Planned))
	}
	if s.Proven.Total > 0 {
		lines = append(lines, fmt.Sprintf("%sproven by a receipt, still listed: %s", prefix, s.Proven))
	}
	if s.ActServed+s.ActUnserved > 0 {
		lines = append(lines, fmt.Sprintf("%sactions: %d served by code, %d not served", prefix, s.ActServed, s.ActUnserved))
	}
	return lines
}

// Doc renders the split for a JSON report.
func (s Split) Doc() map[string]any {
	return map[string]any{"claimed": s.Claimed.Doc(), "unnamed": s.Unnamed.Doc(), "planned": s.Planned.Doc(),
		"proven": s.Proven.Doc(), "actions": map[string]any{"served": s.ActServed, "unserved": s.ActUnserved}}
}
