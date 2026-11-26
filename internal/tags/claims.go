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

// Claimed reports whether something names the id: a test for an obligation,
// code that serves it for an action.
func (c *Claims) Claimed(id string) bool {
	if strings.HasPrefix(id, "act:") {
		return c.Served[id]
	}
	return len(c.Tests[id]) > 0
}

// ClaimKinds are the kinds a claim line counts, in the order it lists them.
// An fv id is proven by the formal stage and is in no claim line.
var ClaimKinds = []string{"tc", "sc", "inv", "fm", "cx", "ck", "act"}

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

// Split is the ids of a list by claim: planned by the project, named by a
// test, named by none.
type Split struct {
	Claimed, Unnamed, Planned Tally
}

// SplitIDs counts ids by whether something names them.
func (c *Claims) SplitIDs(ids []string) Split {
	var s Split
	for _, id := range ids {
		s.count(c, id, false)
	}
	return s
}

// SplitOwed counts the owed entries the design knows: an entry planned at
// adoption is planned, whatever names it (section 14).
func (c *Claims) SplitOwed(d *design.Design) Split {
	var s Split
	for _, e := range d.Owed {
		if d.Known(e.ID) {
			s.count(c, e.ID, e.PlannedAtAdoption())
		}
	}
	return s
}

func (s *Split) count(c *Claims, id string, planned bool) {
	kind, _, _ := strings.Cut(id, ":")
	if !contains(ClaimKinds, kind) {
		return
	}
	switch {
	case planned:
		s.Planned.add(kind)
	case c.Claimed(id):
		s.Claimed.add(kind)
	default:
		s.Unnamed.add(kind)
	}
}

// Lines renders the claim lines, each after the prefix; the planned line
// only when an id is planned.
func (s Split) Lines(prefix, claimed string) []string {
	lines := []string{
		fmt.Sprintf("%s%s: %s", prefix, claimed, s.Claimed),
		fmt.Sprintf("%sno test names it: %s", prefix, s.Unnamed),
	}
	if s.Planned.Total > 0 {
		lines = append(lines, fmt.Sprintf("%splanned: %s", prefix, s.Planned))
	}
	return lines
}

// Doc renders the split for a JSON report.
func (s Split) Doc() map[string]any {
	return map[string]any{"claimed": s.Claimed.Doc(), "unnamed": s.Unnamed.Doc(), "planned": s.Planned.Doc()}
}
