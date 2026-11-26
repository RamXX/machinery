package design

import (
	"fmt"
	"strings"

	"github.com/RamXX/machinery/internal/dialect"
	"github.com/RamXX/machinery/internal/project"
)

// Link gives an imported failure-mode row its subject, required value and
// kind (section 5.4.1).
type Link struct {
	ID       string `yaml:"id"`
	Kind     string `yaml:"kind,omitempty"`
	Subject  string `yaml:"subject"`
	Required string `yaml:"required"`
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
		fms = append(fms, FailureMode{ID: r.ID, Kind: "fault", Failure: r.Failure, Behavior: r.Behavior, Source: c.File, Imported: true, Cells: r.Cells, Line: r.Line})
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
	listed := map[string]bool{}
	for i, l := range lf.Links {
		where := fmt.Sprintf("%s: link %d", c.Links, i+1)
		if l.ID != "" {
			where = fmt.Sprintf("%s: %s", c.Links, l.ID)
		}
		id := strings.ToLower(l.ID)
		if listed[id] {
			problems = append(problems, where+": listed twice")
			continue
		}
		listed[id] = true
		at, ok := byID[id]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: no failure-mode row of %s has this id", where, c.File))
			continue
		}
		fm := &fms[at]
		fm.Kind, fm.Subject, fm.Required, fm.Linked = l.Kind, l.Subject, l.Required, true
		problems = append(problems, checkValues(where, fm)...)
	}
	return fms, problems, true
}
