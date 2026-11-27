package assess

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/v2/internal/design"
	"github.com/RamXX/machinery/v2/internal/gitx"
)

// historyKinds are the kinds history compares, in order (section 18.6).
var historyKinds = []string{"inv", "sc", "entity", "act"}

type historyItem struct{ kind, text string }

func normalize(s string) string { return strings.Join(strings.Fields(s), " ") }

// modelItems lists the ids of the given model files and their texts.
func modelItems(files map[string][]byte) (map[string]historyItem, []string) {
	items := map[string]historyItem{}
	var problems []string
	var rels []string
	for rel := range files {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		m, err := design.ParseModel(rel, files[rel])
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		for _, inv := range m.Invariants {
			items["inv:"+inv.ID] = historyItem{"inv", normalize(inv.Statement)}
		}
		for _, e := range m.Entities {
			items[e.Name] = historyItem{"entity", normalize(e.Definition)}
			for _, inv := range e.Invariants {
				items["inv:"+inv.ID] = historyItem{"inv", normalize(inv.Statement)}
			}
			for _, ac := range e.Actions {
				items[design.ActID(e.Name, ac.Name)] = historyItem{"act", normalize(ac.Description)}
			}
		}
		for _, sc := range m.Scenarios {
			items["sc:"+design.Slug(sc.Name)] = historyItem{"sc", normalize(strings.Join(sc.Steps, "\n"))}
		}
	}
	return items, problems
}

// history compares the model ids at each --since ref, or at the first
// commit of each model file, with those at HEAD.
func (a *assessment) history(s *section) {
	models := a.p.Cfg.Design.Models
	if len(models) == 0 {
		s.own.NotApplicable("no model configured")
		return
	}
	refs := a.since
	if len(refs) == 0 {
		seen := map[string]bool{}
		for _, rel := range models {
			sha, err := gitx.FirstAdd(a.root, rel)
			if err != nil || sha == "" {
				s.lines = append(s.lines, rel+": no commit adds it")
				continue
			}
			if !seen[sha] {
				seen[sha] = true
				refs = append(refs, sinceRef{label: "first commit of " + rel, sha: sha})
			}
		}
	}
	if len(refs) == 0 {
		s.own.NotDemonstrated("no configured model file has a commit")
		return
	}
	cur := map[string][]byte{}
	for _, rel := range models {
		if data, err := a.p.Read(rel); err == nil {
			cur[rel] = data
		}
	}
	now, problems := modelItems(cur)
	for _, pr := range problems {
		s.own.Errorf("%s", pr)
	}
	comparisons := []any{}
	for _, ref := range refs {
		old := map[string][]byte{}
		for _, rel := range models {
			if data, ok := gitx.Show(a.root, ref.sha, rel); ok {
				old[rel] = data
			}
		}
		then, problems := modelItems(old)
		for _, pr := range problems {
			s.own.Errorf("at %s: %s", gitx.Short(ref.sha), pr)
		}
		s.lines = append(s.lines, fmt.Sprintf("since %s (%s): %d ids then, %d at HEAD", ref.label, gitx.Short(ref.sha), len(then), len(now)))
		kinds := map[string]any{}
		for _, kind := range historyKinds {
			added := 0
			removed, reworded := []string{}, []string{}
			for id, it := range now {
				if it.kind != kind {
					continue
				}
				switch was, ok := then[id]; {
				case !ok:
					added++
				case was.text != it.text:
					reworded = append(reworded, id)
				}
			}
			for id, it := range then {
				if _, ok := now[id]; !ok && it.kind == kind {
					removed = append(removed, id)
				}
			}
			sort.Strings(removed)
			sort.Strings(reworded)
			s.lines = append(s.lines, fmt.Sprintf("  %s: %d added, %d reworded, %d removed", kind, added, len(reworded), len(removed)))
			if len(removed) > 0 {
				s.lines = append(s.lines, "    removed: "+strings.Join(removed, ", "))
			}
			if len(reworded) > 0 {
				s.lines = append(s.lines, "    reworded: "+strings.Join(reworded, ", "))
			}
			kinds[kind] = map[string]any{"added": added, "removed": removed, "reworded": reworded}
		}
		comparisons = append(comparisons, map[string]any{"since": ref.label, "commit": gitx.Short(ref.sha), "kinds": kinds})
	}
	s.own.Count(len(refs), "comparisons").Count(len(now), "ids at HEAD")
	s.facts = map[string]any{"comparisons": comparisons}
}
