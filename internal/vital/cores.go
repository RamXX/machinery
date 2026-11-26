// Package vital derives what machinery says about the vital areas of a
// design: the cores that stage arch holds, the candidates and the property
// records (docs/formats.md sections 9 and 23).
package vital

import (
	"strings"

	"github.com/RamXX/machinery/internal/arch"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/tags"
)

// Cores returns the core of every vital area for stage arch (section 9), with
// the serves claims of the act ids the area's properties are over.
func Cores(d *design.Design) ([]arch.Core, error) {
	vas := d.VitalAreas()
	if len(vas) == 0 {
		return nil, nil
	}
	p := d.P
	traced := p.Cfg.Trace != nil
	var found []tags.Tag
	if traced {
		code, err := tags.CodeFiles(p)
		if err != nil {
			return nil, err
		}
		if found, err = tags.Scan(p, code); err != nil {
			return nil, err
		}
	}
	ids, scenarios := d.DeclaredIDs(), d.ScenarioIDs()
	var cores []arch.Core
	for _, a := range vas {
		c := arch.Core{Area: a.Name, Traced: traced}
		if a.Core != nil {
			c.Paths, c.Entry = a.Core.Paths, a.Core.Entry
		}
		acts := map[string]bool{}
		for _, id := range design.OverIDs([]design.Area{a}) {
			if strings.HasPrefix(id, "act:") {
				acts[id] = true
			}
		}
		c.Acts = len(acts) > 0
		seen := map[string]bool{}
		for _, t := range found {
			if t.Kind != tags.Serves {
				continue
			}
			for _, tok := range t.IDs {
				for _, id := range tags.Resolve(t, tok, ids, scenarios) {
					if key := t.File + " " + id; acts[id] && !seen[key] {
						seen[key] = true
						c.Serves = append(c.Serves, arch.Serve{File: t.File, ID: id})
					}
				}
			}
		}
		cores = append(cores, c)
	}
	return cores, nil
}
