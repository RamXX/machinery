// Package affected finds the obligation ids a change touches
// (docs/formats.md section 12.1).
package affected

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/tags"
)

type baseEntry struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Effects []string `json:"effects"`
}

// Compute returns the sorted obligation ids touched by the difference between
// ref and the working tree. The four rules of section 12.1 apply in order.
func Compute(p *project.Project, d *design.Design, ref string) ([]string, error) {
	changed, err := gitx.ChangedSince(p.Root, ref)
	if err != nil {
		return nil, err
	}
	changedSet := map[string]bool{}
	var existing []string
	for _, f := range changed {
		changedSet[f] = true
		if p.Exists(f) {
			existing = append(existing, f)
		}
	}
	hit := map[string]bool{}
	add := func(id string) {
		if d.Obligation(id) != nil {
			hit[id] = true
		}
	}

	// 1. new or different entries of obligations.json
	base := map[string]baseEntry{}
	if data, ok := gitx.Show(p.Root, ref, strings.TrimSuffix(p.Cfg.Design.Gen, "/")+"/obligations.json"); ok {
		var doc struct {
			Obligations []baseEntry `json:"obligations"`
		}
		if err := json.Unmarshal(data, &doc); err == nil {
			for _, e := range doc.Obligations {
				base[e.ID] = e
			}
		}
	}
	for _, o := range d.Obligations {
		b, ok := base[o.ID]
		if !ok || b.Title != o.Title || strings.Join(b.Effects, ",") != strings.Join(o.Effects, ",") {
			add(o.ID)
		}
	}

	// 2 and 3. tags in changed files
	tg, err := tags.Scan(p, existing)
	if err != nil {
		return nil, err
	}
	var obligationIDs, actIDs []string
	for _, o := range d.Obligations {
		obligationIDs = append(obligationIDs, o.ID)
	}
	for _, a := range d.Served {
		actIDs = append(actIDs, a.ID)
	}
	for _, t := range tg {
		for _, tok := range t.IDs {
			switch t.Kind {
			case tags.Verifies:
				for _, id := range tags.Expand(tok, obligationIDs) {
					add(id)
				}
			case tags.Serves:
				for _, act := range tags.Expand(tok, actIDs) {
					servedBy(d, act, add)
				}
			}
		}
	}

	// 4. every case of a machine whose file changed
	for _, m := range d.Machines {
		if changedSet[m.Path] {
			for _, c := range m.Table.Cases() {
				add(c.ID)
			}
		}
	}

	res := make([]string, 0, len(hit))
	for id := range hit {
		res = append(res, id)
	}
	sort.Strings(res)
	return res, nil
}

// servedBy adds what a changed serves tag for an act id can break: the cases
// of the lifecycle machine that fire on the action's event, and the
// invariants the action preserves.
func servedBy(d *design.Design, act string, add func(string)) {
	name := strings.TrimPrefix(act, "act:")
	entity, action, ok := strings.Cut(name, ".")
	if !ok {
		return
	}
	if e, _ := design.FindEntity(d.Models, entity); e != nil {
		for _, a := range e.Actions {
			if a.Name == action {
				for _, inv := range a.Preserves {
					add("inv:" + inv)
				}
			}
		}
	}
	for _, m := range d.Machines {
		if d.LifecycleEntity(m) != entity {
			continue
		}
		fired := map[string]bool{}
		for _, tr := range m.Table.Transitions {
			if tr.Kind == "on" && tr.Event == action {
				fired[tr.ID] = true
				add(tr.ID)
			}
		}
		for _, cl := range m.Table.Clauses {
			if fired[cl.Transition] {
				add(cl.ID)
			}
		}
		for _, ig := range m.Table.Ignores {
			if ig.Event == action {
				add(ig.ID)
			}
		}
	}
}
