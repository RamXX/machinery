package dialect

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/RamXX/machinery/v2/internal/project"
)

// Observed is one transition a register entry says the code makes, with
// state names as the register writes them.
type Observed struct {
	From, Event, To string
}

// RegisterEntry is one entry of the register (section 5.2.1).
type RegisterEntry struct {
	N        int    // 1-based position in the file
	Machine  string // the machine it names, as written
	Observed []Observed
}

// Register reads the register of import.machines.register: a list of
// entries, each naming a machine and listing its observed transitions under
// the configured keys. configured is false when there is none. Problems
// name the register file; an entry with a problem is not returned.
func Register(p *project.Project) (entries []RegisterEntry, problems []string, configured bool) {
	im := p.Cfg.ImportedMachines()
	if im == nil || im.Register == nil {
		return nil, nil, false
	}
	rg := im.Register
	data, err := p.Read(rg.File)
	if err != nil {
		return nil, []string{fmt.Sprintf("%s: cannot read the register: %v", rg.File, shortErr(err))}, true
	}
	var doc any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, []string{fmt.Sprintf("%s: the register does not parse: %v", rg.File, err)}, true
	}
	list, ok := doc.([]any)
	if !ok {
		return nil, []string{rg.File + ": the register must be a list of entries"}, true
	}
	for i, item := range list {
		n := i + 1
		bad := func(format string, a ...any) {
			problems = append(problems, fmt.Sprintf("%s: entry %d: %s", rg.File, n, fmt.Sprintf(format, a...)))
		}
		obj, ok := item.(map[string]any)
		if !ok {
			bad("an entry must be an object")
			continue
		}
		name, ok := obj[rg.Machine].(string)
		if !ok || name == "" {
			bad("%s must name the machine (import.machines.register.machine)", rg.Machine)
			continue
		}
		raw, ok := obj[rg.Observed].([]any)
		if !ok {
			bad("%s must be a list of observed transitions (import.machines.register.observed)", rg.Observed)
			continue
		}
		e := RegisterEntry{N: n, Machine: name}
		good := true
		for j, o := range raw {
			t, ok := o.(map[string]any)
			from, okF := t["from"].(string)
			event, okE := t["event"].(string)
			to, okT := t["to"].(string)
			if !ok || !okF || !okE || !okT || from == "" || event == "" || to == "" {
				bad("%s %d must be an object with the strings from, event and to", rg.Observed, j+1)
				good = false
				continue
			}
			e.Observed = append(e.Observed, Observed{from, event, to})
		}
		if good {
			entries = append(entries, e)
		}
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].N < entries[j].N })
	return entries, problems, true
}

// shortErr drops the path an os error repeats.
func shortErr(err error) string {
	if pe, ok := err.(interface{ Unwrap() error }); ok && pe.Unwrap() != nil {
		return pe.Unwrap().Error()
	}
	return err.Error()
}
