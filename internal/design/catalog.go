package design

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/dialect"
	"github.com/RamXX/machinery/internal/project"
)

// Link gives one imported failure-mode row (id) or many (ids) a subject, a
// required value and a kind, or says why they name no subject (section 5.4.1).
type Link struct {
	ID        string   `yaml:"id,omitempty"`
	IDs       []string `yaml:"ids,omitempty"`
	Kind      string   `yaml:"kind,omitempty"`
	Subject   string   `yaml:"subject,omitempty"`
	NoSubject *string  `yaml:"no_subject,omitempty"`
	Required  string   `yaml:"required,omitempty"`
}

type linksFile struct {
	Version int    `yaml:"version"`
	Links   []Link `yaml:"links"`
}

// LoadImportedFailureModes reads the catalog of import.failure_modes and its
// links file. native are the failure modes of design.failure_modes: an id in
// both is a problem and the row is not read. configured is false when the
// project imports no catalog.
func LoadImportedFailureModes(p *project.Project, native []FailureMode) (fms []FailureMode, problems []string, configured bool) {
	c := p.Cfg.ImportedFailureModes()
	if c == nil {
		return nil, nil, false
	}
	data, err := p.Read(c.File)
	if err != nil {
		return nil, []string{fmt.Sprintf("%s: cannot read: %s", c.File, shortErr(err))}, true
	}
	re, err := c.IDPattern()
	if err != nil {
		return nil, []string{fmt.Sprintf("import.failure_modes.id: %v", err)}, true
	}
	rows, problems := dialect.Catalog(c.File, data, re, dialect.Columns{Failure: c.Columns.Failure, Behavior: c.Columns.Behavior})
	nativeIDs := map[string]bool{}
	for _, fm := range native {
		nativeIDs[fm.ID] = true
	}
	byID := map[string]int{}
	for _, r := range rows {
		if nativeIDs[r.ID] {
			problems = append(problems, fmt.Sprintf("%s:%d: %s: fm:%s is also declared in %s", c.File, r.Line, r.Cell, r.ID, p.Cfg.Design.FailureModes))
			continue
		}
		byID[r.ID] = len(fms)
		fms = append(fms, FailureMode{ID: r.ID, Kind: "fault", Failure: r.Failure, Behavior: r.Behavior, Source: c.File, Imported: true})
	}
	if c.Links == "" {
		return fms, problems, true
	}
	ldata, err := p.Read(c.Links)
	if err != nil {
		return fms, append(problems, fmt.Sprintf("%s: cannot read: %s", c.Links, shortErr(err))), true
	}
	var lf linksFile
	if err := strictYAML(ldata, &lf); err != nil {
		return fms, append(problems, fmt.Sprintf("%s: %v", c.Links, err)), true
	}
	if lf.Version != 1 {
		problems = append(problems, fmt.Sprintf("%s: version must be 1", c.Links))
	}
	problems = append(problems, applyLinks(c.Links, c.File, lf.Links, fms, byID)...)
	return fms, problems, true
}

// maxRange is how many ids one range of a links entry may name.
const maxRange = 10000

// listed is how many ids a finding about a links entry prints in full.
const listed = 5

// applyLinks gives the rows the values of the links entries that name them.
// byID maps a machinery id to its row in fms. A row two entries name keeps
// the values of the first.
func applyLinks(rel, catalog string, links []Link, fms []FailureMode, byID map[string]int) (problems []string) {
	owner := map[string]int{} // machinery id -> index of the entry that holds it
	labels := make([]string, len(links))
	for i, l := range links {
		where := fmt.Sprintf("%s: link %d", rel, i+1)
		items := l.IDs
		switch {
		case l.ID != "" && l.IDs != nil:
			problems = append(problems, where+": id and ids are both set; keep one")
			continue
		case l.ID != "":
			items = []string{l.ID}
		case len(l.IDs) == 0:
			problems = append(problems, where+": id or ids is required")
			continue
		}
		labels[i] = entryLabel(items)
		where = fmt.Sprintf("%s: %s", rel, labels[i])
		ids, probs := expandIDs(items)
		if len(probs) > 0 {
			for _, p := range probs {
				problems = append(problems, where+": "+p)
			}
			continue
		}
		values := FailureMode{Kind: l.Kind, Subject: l.Subject, NoSubject: l.NoSubject, Required: l.Required}
		problems = append(problems, checkValues(where, &values, true)...)
		seen := map[string]bool{}
		var missing []string
		overlaps := map[int][]string{}
		for _, written := range ids {
			id := strings.ToLower(written)
			if seen[id] {
				problems = append(problems, fmt.Sprintf("%s: %s is listed twice", where, written))
				continue
			}
			seen[id] = true
			at, ok := byID[id]
			if !ok {
				missing = append(missing, written)
				continue
			}
			if first, taken := owner[id]; taken {
				overlaps[first] = append(overlaps[first], written)
				continue
			}
			owner[id] = i
			fm := &fms[at]
			fm.Kind, fm.Subject, fm.NoSubject, fm.Required, fm.Linked = values.Kind, values.Subject, values.NoSubject, values.Required, true
		}
		switch {
		case len(missing) > 0 && l.ID != "":
			problems = append(problems, fmt.Sprintf("%s: no failure-mode row of %s has this id", where, catalog))
		case len(missing) == 1:
			problems = append(problems, fmt.Sprintf("%s: no failure-mode row of %s has the id %s", where, catalog, missing[0]))
		case len(missing) > 1:
			problems = append(problems, fmt.Sprintf("%s: no failure-mode row of %s has the ids %s", where, catalog, firstOf(missing)))
		}
		for first := 0; first < i; first++ {
			if both := overlaps[first]; len(both) > 0 {
				problems = append(problems, fmt.Sprintf("%s: links %d (%s) and %d (%s) overlap: %s", rel, first+1, labels[first], i+1, labels[i], firstOf(both)))
			}
		}
	}
	return problems
}

// entryLabel names a links entry by what it lists: its first three items.
func entryLabel(items []string) string {
	if len(items) > 3 {
		return strings.Join(items[:3], ", ") + ", ..."
	}
	return strings.Join(items, ", ")
}

// firstOf joins the first ids of a list and counts the others.
func firstOf(ids []string) string {
	if len(ids) <= listed {
		return strings.Join(ids, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(ids[:listed], ", "), len(ids)-listed)
}

// expandIDs returns the ids a list of ids and ranges names, in order. A
// range <prefix><a>..<prefix><b> names <prefix><n> for every n from a to b,
// each written with at least as many digits as <a> has.
func expandIDs(items []string) (ids, problems []string) {
	for _, item := range items {
		parts := strings.Split(item, "..")
		switch {
		case len(parts) == 1 && strings.TrimSpace(item) != "":
			ids = append(ids, item)
			continue
		case len(parts) != 2 || parts[0] == "" || parts[1] == "":
			problems = append(problems, fmt.Sprintf("%s is not an id or a range <first>..<last>", item))
			continue
		}
		prefix, from, okFrom := numericSuffix(parts[0])
		prefixTo, to, okTo := numericSuffix(parts[1])
		a, errA := strconv.Atoi(from)
		b, errB := strconv.Atoi(to)
		switch {
		case !okFrom || !okTo || errA != nil || errB != nil:
			problems = append(problems, fmt.Sprintf("the ends of the range %s must end in a number", item))
		case !strings.EqualFold(prefix, prefixTo):
			problems = append(problems, fmt.Sprintf("the ends of the range %s do not share a prefix", item))
		case a > b:
			problems = append(problems, fmt.Sprintf("the range %s runs backwards", item))
		case b-a+1 > maxRange:
			problems = append(problems, fmt.Sprintf("the range %s names %d ids; a range holds at most %d", item, b-a+1, maxRange))
		default:
			for n := a; n <= b; n++ {
				ids = append(ids, fmt.Sprintf("%s%0*d", prefix, len(from), n))
			}
		}
	}
	return ids, problems
}

// numericSuffix splits an id into what comes before its trailing digits and
// those digits. ok is false when it does not end in a digit.
func numericSuffix(id string) (prefix, digits string, ok bool) {
	i := len(id)
	for i > 0 && id[i-1] >= '0' && id[i-1] <= '9' {
		i--
	}
	return id[:i], id[i:], i < len(id)
}
