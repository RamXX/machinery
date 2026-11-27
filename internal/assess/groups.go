package assess

import (
	"sort"
	"strings"

	"github.com/RamXX/machinery/v2/internal/report"
)

// The findings of every section grouped by the machines and the entities
// they name (section 18.9).

// group is the findings that name one machine or one entity.
type group struct {
	name, source         string
	errors, drift, warns int
	counts               map[ruleKey]int
}

type ruleKey struct {
	sev report.Severity
	key string
}

func (g *group) add(f report.Finding, rule string) {
	switch f.Sev {
	case report.Error:
		g.errors++
	case report.Drift:
		g.drift++
	default:
		g.warns++
	}
	g.counts[ruleKey{f.Sev, rule}]++
}

// rules lists the counts per rule, most frequent first, as section rules are.
func (g *group) rules() []rule {
	var res []rule
	for k, n := range g.counts {
		res = append(res, rule{k.sev, k.key, n})
	}
	sortRules(res)
	return res
}

func sortRules(res []rule) {
	sort.Slice(res, func(i, j int) bool {
		if res[i].count != res[j].count {
			return res[i].count > res[j].count
		}
		if res[i].sev != res[j].sev {
			return res[i].sev < res[j].sev
		}
		return res[i].key < res[j].key
	})
}

// isWordByte reports a byte that continues a name: a letter, a digit or _.
func isWordByte(b byte) bool {
	return b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

// containsWord reports whether w occurs in msg with no name byte right
// before or right after it; in also lists the extra bytes that continue a
// name (for a path: / . -).
func containsWord(msg, w, in string) bool {
	if w == "" {
		return false
	}
	cont := func(b byte) bool { return isWordByte(b) || strings.IndexByte(in, b) >= 0 }
	for i := 0; ; {
		j := strings.Index(msg[i:], w)
		if j < 0 {
			return false
		}
		start, end := i+j, i+j+len(w)
		before := start == 0 || !cont(msg[start-1])
		after := end == len(msg) || !cont(msg[end]) || (msg[end] == '.' && (end+1 == len(msg) || !cont(msg[end+1])))
		if before && after {
			return true
		}
		i = start + 1
	}
}

// groups returns the findings of every section per machine and per entity
// that at least one finding names, each sorted by name. A finding names a
// machine by its id, its file or a case id with its tag, and an entity by
// its name (which entity:<E> and act:<E>.<action> contain).
func (a *assessment) groups() (machines, entities []*group) {
	d := a.ctx.Design()
	names := designNames(d)
	byMachine := map[string]*group{}
	byEntity := map[string]*group{}
	var ents []string
	for _, md := range d.Models {
		for _, e := range md.Entities {
			ents = append(ents, e.Name)
		}
	}
	for _, s := range a.sections {
		lists := []struct {
			stage string
			fs    []report.Finding
		}{{s.name, s.own.Findings}}
		for _, st := range s.stages {
			lists = append(lists, struct {
				stage string
				fs    []report.Finding
			}{st.Name, st.Findings})
		}
		for _, l := range lists {
			for _, f := range l.fs {
				r := ruleOf(l.stage, f.Msg, names)
				for _, m := range d.Machines {
					if !containsWord(f.Msg, m.ID, "") && !containsWord(f.Msg, m.Path, "/.-") && !strings.Contains(f.Msg, "tc:"+m.Tag+"-") {
						continue
					}
					g := byMachine[m.ID]
					if g == nil {
						g = &group{name: m.ID, source: m.Path, counts: map[ruleKey]int{}}
						byMachine[m.ID] = g
					}
					g.add(f, r)
				}
				for _, e := range ents {
					if !containsWord(f.Msg, e, "") {
						continue
					}
					g := byEntity[e]
					if g == nil {
						g = &group{name: e, counts: map[ruleKey]int{}}
						byEntity[e] = g
					}
					g.add(f, r)
				}
			}
		}
	}
	for _, g := range byMachine {
		machines = append(machines, g)
	}
	for _, g := range byEntity {
		entities = append(entities, g)
	}
	sort.Slice(machines, func(i, j int) bool { return machines[i].name < machines[j].name })
	sort.Slice(entities, func(i, j int) bool { return entities[i].name < entities[j].name })
	return machines, entities
}
