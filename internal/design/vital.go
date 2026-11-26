package design

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/project"
)

// The wording of the evidence of a vital area (section 23.2).
const (
	// ProofAwaited ends every line that reports an id a vital property is
	// over as proven.
	ProofAwaited = "(by test and model; proof awaited)"
	// AwaitingProof is the status of a proof obligation.
	AwaitingProof = "awaiting proof"
	// AssumedUnproven is an assumption one of whose evidence ids is owed.
	AssumedUnproven = "assumed, unproven"
	// AssumedEvidenced is an assumption none of whose evidence ids is owed.
	AssumedEvidenced = "assumed, evidenced"
)

// Property is one entry of a vital area's property list (section 23.1).
type Property struct {
	Name      string   `yaml:"name"`
	Statement string   `yaml:"statement"`
	Kind      string   `yaml:"kind"`
	Over      []string `yaml:"over"`
}

// Core is where the code of a vital path lives (section 23.1).
type Core struct {
	Paths []string `yaml:"paths"`
	Entry []string `yaml:"entry"`
}

// Assumption is what a vital core relies on from its surroundings (section 23.1).
type Assumption struct {
	Name      string   `yaml:"name"`
	Statement string   `yaml:"statement"`
	Evidence  []string `yaml:"evidence"`
}

// Dismissal takes a candidate off the list of machinery vital candidates and
// keeps the reason on record (section 23.3).
type Dismissal struct {
	ID     string `yaml:"id"`
	Reason string `yaml:"reason"`
}

// PropertyKinds are the kinds a property may have.
var PropertyKinds = map[string]bool{"invariant": true, "pre-post": true, "exactly-once": true, "never": true}

var overKinds = map[string]bool{"inv": true, "act": true, "tc": true}

var entryRe = regexp.MustCompile(`^[^#\s]+#[^#\s]+$`)

// ProofID is the id of the proof obligation of a property.
func ProofID(area, property string) string { return "pf:" + area + ":" + property }

// vitalShape checks the keys of section 23.1 that need no design. where
// prefixes every problem.
func vitalShape(where string, a Area) (problems []string) {
	if a.Tier != Vital {
		if len(a.Property) > 0 || a.Core != nil || len(a.Assumes) > 0 {
			problems = append(problems, where+": only a vital area has property, core and assumes")
		}
		return problems
	}
	if len(a.Property) == 0 {
		problems = append(problems, where+": a vital area states at least one property")
	}
	if a.Core == nil {
		problems = append(problems, where+": a vital area names its core (core.paths and core.entry)")
	}
	if len(a.Assumes) == 0 {
		problems = append(problems, where+": a vital area states what its core assumes (at least one entry under assumes)")
	}
	seen := map[string]bool{}
	for i, pr := range a.Property {
		at := fmt.Sprintf("%s: property %d", where, i+1)
		switch {
		case !kebabRe.MatchString(pr.Name):
			problems = append(problems, at+": name must match ^[a-z][a-z0-9-]+$")
		case seen[pr.Name]:
			problems = append(problems, fmt.Sprintf("%s: property %s: duplicate name", where, pr.Name))
			continue
		default:
			at = fmt.Sprintf("%s: property %s", where, pr.Name)
		}
		seen[pr.Name] = true
		if strings.TrimSpace(pr.Statement) == "" {
			problems = append(problems, at+": statement is required")
		}
		if !PropertyKinds[pr.Kind] {
			problems = append(problems, fmt.Sprintf("%s: kind must be invariant, pre-post, exactly-once or never, got %q", at, pr.Kind))
		}
		if len(pr.Over) == 0 {
			problems = append(problems, at+": over needs at least one id")
		}
	}
	if c := a.Core; c != nil {
		if len(c.Paths) == 0 {
			problems = append(problems, where+": core.paths needs at least one glob")
		}
		for _, g := range c.Paths {
			if !project.ValidGlob(g) {
				problems = append(problems, fmt.Sprintf("%s: core.paths: %q is not a valid glob", where, g))
			}
		}
		if len(c.Entry) == 0 {
			problems = append(problems, where+": core.entry needs at least one <file>#<symbol>")
		}
		for _, e := range c.Entry {
			if !entryRe.MatchString(e) {
				problems = append(problems, fmt.Sprintf("%s: core.entry: %q is not <file>#<symbol>", where, e))
			}
		}
	}
	seen = map[string]bool{}
	for i, as := range a.Assumes {
		at := fmt.Sprintf("%s: assumes %d", where, i+1)
		switch {
		case !kebabRe.MatchString(as.Name):
			problems = append(problems, at+": name must match ^[a-z][a-z0-9-]+$")
		case seen[as.Name]:
			problems = append(problems, fmt.Sprintf("%s: assumes %s: duplicate name", where, as.Name))
			continue
		default:
			at = fmt.Sprintf("%s: assumes %s", where, as.Name)
		}
		seen[as.Name] = true
		if strings.TrimSpace(as.Statement) == "" {
			problems = append(problems, at+": statement is required")
		}
		if len(as.Evidence) == 0 {
			problems = append(problems, at+": evidence needs at least one id")
		}
	}
	return problems
}

// notVitalShape checks the dismissals of section 23.3 that need no design.
func notVitalShape(rel string, ds []Dismissal) (problems []string) {
	seen := map[string]bool{}
	for i, dm := range ds {
		id := strings.TrimSpace(dm.ID)
		switch {
		case id == "":
			problems = append(problems, fmt.Sprintf("%s: not_vital entry %d: id is required", rel, i+1))
			continue
		case seen[id]:
			problems = append(problems, fmt.Sprintf("%s: not_vital: %s is listed twice", rel, id))
			continue
		}
		seen[id] = true
		if strings.TrimSpace(dm.Reason) == "" {
			problems = append(problems, fmt.Sprintf("%s: not_vital: %s: reason is required", rel, id))
		}
	}
	return problems
}

// SplitEntry splits a core entry point into its file and symbol.
func SplitEntry(e string) (file, symbol string) {
	i := strings.LastIndex(e, "#")
	if i < 0 {
		return e, ""
	}
	return e[:i], e[i+1:]
}

// VitalAreas lists the vital areas of the risk file, in file order.
func (d *Design) VitalAreas() []Area {
	var res []Area
	for _, a := range d.Areas {
		if a.Tier == Vital {
			res = append(res, a)
		}
	}
	return res
}

// VitalAreasOf lists the vital areas the ids belong to, a pf or fv id
// naming its own area, in file order.
func (d *Design) VitalAreasOf(ids []string) []Area {
	want := map[string]bool{}
	for _, id := range ids {
		_, in := d.Tier(id)
		for _, a := range in {
			want[a.Name] = true
		}
	}
	var res []Area
	for _, a := range d.VitalAreas() {
		if want[a.Name] {
			res = append(res, a)
		}
	}
	return res
}

// inventoryProofs derives the proof obligations of the vital areas (section
// 23.2), sorted by id.
func (d *Design) inventoryProofs() {
	d.Proofs = nil
	seen := map[string]bool{}
	for _, a := range d.VitalAreas() {
		for _, pr := range a.Property {
			id := ProofID(a.Name, pr.Name)
			if pr.Name == "" || seen[id] {
				continue
			}
			seen[id] = true
			d.Proofs = append(d.Proofs, Obligation{ID: id, Kind: "pf", Source: d.P.Cfg.Design.Risk, Title: pr.Statement})
		}
	}
	sort.Slice(d.Proofs, func(i, j int) bool { return d.Proofs[i].ID < d.Proofs[j].ID })
}

// ProofIDs lists the proof obligations, sorted.
func (d *Design) ProofIDs() []string {
	var ids []string
	for _, o := range d.Proofs {
		ids = append(ids, o.ID)
	}
	return ids
}

// IsProof reports whether id is a proof obligation.
func (d *Design) IsProof(id string) bool {
	for _, o := range d.Proofs {
		if o.ID == id {
			return true
		}
	}
	return false
}

// ProofsOf lists the proof obligations of the given areas, sorted.
func ProofsOf(areas []Area) []string {
	var ids []string
	for _, a := range areas {
		for _, pr := range a.Property {
			if pr.Name != "" {
				ids = append(ids, ProofID(a.Name, pr.Name))
			}
		}
	}
	sort.Strings(ids)
	return dedupe(ids)
}

// OverIDs lists, sorted and once each, the ids the properties of the areas
// are over.
func OverIDs(areas []Area) []string {
	var ids []string
	for _, a := range areas {
		for _, pr := range a.Property {
			ids = append(ids, pr.Over...)
		}
	}
	sort.Strings(ids)
	return dedupe(ids)
}

// IsOver reports whether a property of a vital area is over the id.
func (d *Design) IsOver(id string) bool {
	for _, o := range OverIDs(d.VitalAreas()) {
		if o == id {
			return true
		}
	}
	return false
}

// UnevidencedAssumption is an assumption of a vital area with owed evidence.
type UnevidencedAssumption struct {
	Area       string
	Assumption Assumption
	Owed       []string // its evidence ids in the owed list, in file order
}

// Line renders the assumption as reports print it (section 23.2).
func (u UnevidencedAssumption) Line() string {
	return fmt.Sprintf("%s: %s %s: %s", AssumedUnproven, u.Area, u.Assumption.Name, u.Assumption.Statement)
}

// Unevidenced lists the assumptions of the areas that have an owed evidence
// id, sorted by area, then name.
func (d *Design) Unevidenced(areas []Area) []UnevidencedAssumption {
	owed := d.OwedSet()
	var res []UnevidencedAssumption
	for _, a := range areas {
		for _, as := range a.Assumes {
			u := UnevidencedAssumption{Area: a.Name, Assumption: as}
			for _, id := range as.Evidence {
				if _, ok := owed[id]; ok {
					u.Owed = append(u.Owed, id)
				}
			}
			if len(u.Owed) > 0 {
				res = append(res, u)
			}
		}
	}
	sort.SliceStable(res, func(i, j int) bool {
		if res[i].Area != res[j].Area {
			return res[i].Area < res[j].Area
		}
		return res[i].Assumption.Name < res[j].Assumption.Name
	})
	return res
}

// CommittedStatus is the status of an id from committed files alone (section
// 4.1): owed while the owed list holds it, else proven.
func (d *Design) CommittedStatus(id string) string {
	if _, ok := d.OwedSet()[id]; ok {
		return "owed"
	}
	return "proven"
}

// VitalProblems checks the references of the vital areas and the dismissals
// against the design (sections 23.1 and 23.3). Shape problems are found while
// parsing.
func (d *Design) VitalProblems() []string {
	rel := d.P.Cfg.Design.Risk
	var problems []string
	declared := d.DeclaredIDs()
	for _, a := range d.VitalAreas() {
		where := fmt.Sprintf("%s: %s", rel, a.Name)
		for _, pr := range a.Property {
			if !kebabRe.MatchString(pr.Name) {
				continue
			}
			for _, id := range pr.Over {
				if msg := d.overProblem(a, id); msg != "" {
					problems = append(problems, fmt.Sprintf("%s: property %s: over %s%s", where, pr.Name, id, msg))
				}
			}
		}
		for _, as := range a.Assumes {
			if !kebabRe.MatchString(as.Name) {
				continue
			}
			for _, id := range as.Evidence {
				if d.Obligation(id) == nil {
					problems = append(problems, fmt.Sprintf("%s: assumes %s: evidence %s is not an obligation of the design", where, as.Name, id))
				}
			}
		}
	}
	seen := map[string]bool{}
	for _, dm := range d.NotVital {
		id := strings.TrimSpace(dm.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		matched := false
		for _, did := range declared {
			if MatchID(id, did) && tierKinds[kindOfID(did)] {
				matched = true
				break
			}
		}
		if !matched {
			problems = append(problems, fmt.Sprintf("%s: not_vital: %s matches no act, inv, sc, tc or fm id", rel, id))
			continue
		}
		for _, a := range d.VitalAreas() {
			if d.dismissalInArea(id, a) {
				problems = append(problems, fmt.Sprintf("%s: not_vital: %s is in vital area %s; a dismissed id is not vital", rel, id, a.Name))
				break
			}
		}
	}
	return problems
}

// overProblem is what is wrong with an id a property is over, or "".
func (d *Design) overProblem(a Area, id string) string {
	switch {
	case !overKinds[kindOfID(id)]:
		return ": a property is over inv, act or tc ids"
	case strings.HasSuffix(id, "*"):
		return ": name an exact id, not a pattern"
	case !d.Declared(id):
		return " is not an id of the design"
	case d.Status(id) != "":
		return fmt.Sprintf(" is %s: it yields no obligation", d.Status(id))
	case !MatchAny(a.IDs, id):
		return " is not in the area; add it to ids, or name an id the area holds"
	}
	return ""
}

// dismissalInArea reports whether a dismissal names an id the area holds.
func (d *Design) dismissalInArea(pattern string, a Area) bool {
	for _, id := range d.DeclaredIDs() {
		if MatchID(pattern, id) && MatchAny(a.IDs, id) {
			return true
		}
	}
	return false
}

// InVitalArea reports whether a vital area holds the id (section 23.3).
func (d *Design) InVitalArea(id string) bool {
	for _, a := range d.VitalAreas() {
		if MatchAny(a.IDs, id) {
			return true
		}
	}
	return false
}

// Dismissed reports whether a not_vital entry matches the id.
func (d *Design) Dismissed(id string) bool {
	for _, dm := range d.NotVital {
		if p := strings.TrimSpace(dm.ID); p != "" && MatchID(p, id) {
			return true
		}
	}
	return false
}

// tierKinds are the id kinds a risk area or a dismissal may name.
var tierKinds = map[string]bool{"act": true, "inv": true, "sc": true, "tc": true, "fm": true}

func kindOfID(id string) string {
	k, _, _ := strings.Cut(id, ":")
	return k
}

func dedupe(sorted []string) []string {
	var res []string
	for i, s := range sorted {
		if i == 0 || s != sorted[i-1] {
			res = append(res, s)
		}
	}
	return res
}
