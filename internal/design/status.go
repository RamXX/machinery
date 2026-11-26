package design

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/project"
)

// The statuses design/status.yaml gives (section 4.1). An id without one is
// owed or proven, which the owed list decides.
const (
	Proposed = "proposed"
	Retired  = "retired"
)

// StatusKinds are the id kinds that take a status (section 4.1).
var StatusKinds = map[string]bool{"act": true, "inv": true, "sc": true, "tc": true, "fm": true}

// StatusEntry is one entry of the status file.
type StatusEntry struct {
	ID     string `yaml:"id"`
	Note   string `yaml:"note,omitempty"`
	Reason string `yaml:"reason,omitempty"`
	Since  string `yaml:"since,omitempty"`
}

type statusFile struct {
	Version  int           `yaml:"version"`
	Proposed []StatusEntry `yaml:"proposed,omitempty"`
	Retired  []StatusEntry `yaml:"retired,omitempty"`
}

// StatusList is the parsed status file.
type StatusList struct {
	Proposed []StatusEntry
	Retired  []StatusEntry
}

// Entries counts the entries of both lists.
func (l StatusList) Entries() int { return len(l.Proposed) + len(l.Retired) }

// ParseStatus parses status file bytes and checks what needs no design.
func ParseStatus(rel string, data []byte) (StatusList, []string) {
	var f statusFile
	if err := strictYAML(data, &f); err != nil {
		return StatusList{}, []string{fmt.Sprintf("%s: %v", rel, err)}
	}
	var problems []string
	if f.Version != 1 {
		problems = append(problems, fmt.Sprintf("%s: version must be 1", rel))
	}
	return StatusList{Proposed: f.Proposed, Retired: f.Retired}, problems
}

// LoadStatus reads the status file; present is false when it does not exist.
func LoadStatus(p *project.Project) (l StatusList, problems []string, present bool) {
	rel := p.Cfg.Design.Status
	if rel == "" || !p.Exists(rel) {
		return StatusList{}, nil, false
	}
	data, err := p.Read(rel)
	if err != nil {
		return StatusList{}, []string{fmt.Sprintf("%s: cannot read: %s", rel, shortErr(err))}, true
	}
	l, problems = ParseStatus(rel, data)
	return l, problems, true
}

// StatusMark is the status of one id and the entry that gives it.
type StatusMark struct {
	Status string
	Entry  StatusEntry
}

// resolveStatus gives each declared id the status of the entries that match
// it. An entry with a problem of its own gives none, and an id that both
// lists match keeps no status: it stays an obligation.
func resolveStatus(rel string, l StatusList, declared []string) (map[string]StatusMark, []string) {
	var problems []string
	marks := map[string]StatusMark{}
	both := map[string]bool{}
	for _, list := range []struct {
		name    string
		entries []StatusEntry
	}{{Proposed, l.Proposed}, {Retired, l.Retired}} {
		listed := map[string]bool{}
		for i, e := range list.entries {
			id := strings.TrimSpace(e.ID)
			if id == "" {
				problems = append(problems, fmt.Sprintf("%s: %s entry %d: id is required", rel, list.name, i+1))
				continue
			}
			if listed[id] {
				problems = append(problems, fmt.Sprintf("%s: %s: %s is listed twice", rel, list.name, id))
				continue
			}
			listed[id] = true
			kind, _, _ := strings.Cut(id, ":")
			if !StatusKinds[kind] {
				problems = append(problems, fmt.Sprintf("%s: %s: %s: only act, inv, sc, tc and fm ids take a status", rel, list.name, id))
				continue
			}
			if list.name == Retired && strings.TrimSpace(e.Reason) == "" {
				problems = append(problems, fmt.Sprintf("%s: %s: %s: reason is required", rel, list.name, id))
				continue
			}
			matched := false
			for _, d := range declared {
				if !MatchID(id, d) {
					continue
				}
				matched = true
				if m, ok := marks[d]; ok && m.Status != list.name {
					both[d] = true
					continue
				}
				if _, ok := marks[d]; !ok {
					marks[d] = StatusMark{Status: list.name, Entry: e}
				}
			}
			if !matched {
				problems = append(problems, fmt.Sprintf("%s: %s: %s matches no design id", rel, list.name, id))
			}
		}
	}
	var ids []string
	for id := range both {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		delete(marks, id)
		problems = append(problems, fmt.Sprintf("%s: %s is both proposed and retired", rel, id))
	}
	return marks, problems
}

// Status returns the status the status file gives an id: Proposed, Retired
// or "" for an id that is owed or proven.
func (d *Design) Status(id string) string { return d.statuses[id].Status }

// StatusEntryOf returns the entry that gives an id its status.
func (d *Design) StatusEntryOf(id string) StatusEntry { return d.statuses[id].Entry }

// WithStatus lists the ids that have the status, sorted.
func (d *Design) WithStatus(status string) []string {
	var res []string
	for id, m := range d.statuses {
		if m.Status == status {
			res = append(res, id)
		}
	}
	sort.Strings(res)
	return res
}

// DeclaredIDs lists every act and obligation id the design declares, sorted,
// the proposed and retired ones included.
func (d *Design) DeclaredIDs() []string { return append([]string(nil), d.declared...) }

// Declared reports whether the design declares an id, whatever its status.
func (d *Design) Declared(id string) bool {
	i := sort.SearchStrings(d.declared, id)
	return i < len(d.declared) && d.declared[i] == id
}

// StatusCounts counts every declared id by status (section 4.1): owed is an
// id the owed list holds, proven every other id that yields an obligation.
func (d *Design) StatusCounts() (proposed, owed, proven, retired int) {
	owedSet := d.OwedSet()
	for _, id := range d.AllIDs() {
		if _, ok := owedSet[id]; ok {
			owed++
		} else {
			proven++
		}
	}
	return len(d.WithStatus(Proposed)), owed, proven, len(d.WithStatus(Retired))
}

// OwedIssues lists the problems of the owed entries that the design gives
// (sections 4.1, 11.1): an unknown id, a proposed or a retired one. Sorted by id.
func (d *Design) OwedIssues() []string {
	es := append([]OwedEntry(nil), d.Owed...)
	sort.SliceStable(es, func(i, j int) bool { return es[i].ID < es[j].ID })
	rel := d.P.Cfg.Design.Owed
	var res []string
	for _, e := range es {
		switch {
		case e.ID == "":
		case d.Status(e.ID) == Proposed:
			res = append(res, fmt.Sprintf("proposed: %s is in %s; a proposed id is not owed: remove it, or commit it first", e.ID, rel))
		case d.Status(e.ID) == Retired:
			res = append(res, fmt.Sprintf("retired: %s is in %s; a retired id is not owed: remove it, or un-retire it", e.ID, rel))
		case d.IsProof(e.ID):
			res = append(res, fmt.Sprintf("proof: %s is in %s; a proof obligation is not owed: it awaits a machine-checked proof, which this version cannot give; remove the entry", e.ID, rel))
		case !d.Known(e.ID):
			res = append(res, "unknown owed id: "+e.ID)
		}
	}
	return res
}
