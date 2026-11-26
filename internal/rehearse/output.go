package rehearse

import (
	"fmt"
	"strings"

	"github.com/RamXX/machinery/internal/pm"
)

// write saves rehearsal.md and rehearsal.json (section 17.3).
func (r *rehearsal) write(out string) error {
	if err := save(out, "rehearsal.md", []byte(r.markdown())); err != nil {
		return err
	}
	return save(out, "rehearsal.json", pm.JSON(r.doc()))
}

func (r *rehearsal) markdown() string {
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	p("# Adoption rehearsal: %s at %s", r.name, r.head)
	p("")
	p("- config: %s", r.cfgLabel)
	if r.changed > 0 || r.untracked > 0 {
		p("- dirty: %d changed, %d untracked file(s) are not rehearsed", r.changed, r.untracked)
	}
	if len(r.outside) > 0 {
		p("- outside links: %s", strings.Join(r.outside, ", "))
	}
	p("- %s", r.adoptLine)
	p("- %s", r.backlogLine)
	p("")
	p("## Files adopt writes")
	p("")
	if len(r.files) == 0 {
		p("None.")
	} else {
		p("| File | Bytes |")
		p("|---|---|")
		for _, f := range r.files {
			p("| `%s` | %d |", f.Path, f.Bytes)
		}
	}
	p("")
	p("## Waivers adopt adds")
	p("")
	list(p, r.waivers, "")
	p("")
	p("## Baselines")
	p("")
	list(p, r.baselines, " entries")
	p("")
	p("## Owed")
	p("")
	if r.owedTotal == 0 {
		p("None.")
	} else {
		for _, c := range r.owed {
			p("- %s: %d", c.Key, c.Count)
		}
		p("- total: %d", r.owedTotal)
		for _, l := range r.claims.Lines("- ") {
			p("%s", l)
		}
	}
	if len(r.blocked) > 0 {
		p("")
		p("## Adopt is blocked")
		p("")
		p("```")
		for _, l := range r.blocked {
			p("%s", l)
		}
		p("```")
	}
	p("")
	p("## Check")
	p("")
	p("`%s`", r.checkLine)
	p("")
	p("| Stage | Outcome |")
	p("|---|---|")
	for _, s := range r.stageOutcomes {
		p("| %s | %s |", s.Key, r.outcomes[s.Key])
	}
	p("")
	p("## Plan")
	p("")
	if r.planError != "" {
		p("%s", r.planError)
	} else {
		if r.mayExist > 0 {
			p("Up to %d stories, %d edge(s), as `machinery plan` would create them (`plan.md`; never applied); %d may exist already: every id of the group carries a story, and a rehearsal never reads the backlog.", r.planTotal, r.planEdges, r.mayExist)
		} else {
			p("%d stories, %d edge(s), as `machinery plan` would create them (`plan.md`; never applied).", r.planTotal, r.planEdges)
		}
		if r.planWire != "" || r.plannedIDs > 0 {
			p("")
		}
		if r.planWire != "" {
			p("- wire evidence: %s", r.planWire)
		}
		if r.plannedIDs > 0 {
			p("- already planned in the project: %d id(s) planned at adoption keep the story the project gave them; no story is planned for them", r.plannedIDs)
		}
		if len(r.planStories) > 0 {
			p("")
			p("| Kind | Tier | Stories |")
			p("|---|---|---|")
			for _, s := range r.planStories {
				p("| %s | %s | %d |", s.Kind, s.Tier, s.Count)
			}
		}
	}
	p("")
	p("## What blocks adoption")
	p("")
	if len(r.blocks) == 0 {
		p("Nothing: every stage passed or was not applicable.")
	}
	for i, bl := range r.blocks {
		if i > 0 {
			p("")
		}
		p("### %s: %s", bl.Stage, bl.Outcome)
		p("")
		if len(bl.Findings) == 0 {
			p("No finding: the outcome line says why.")
		}
		for _, f := range bl.Findings {
			p("- %s  %s", f.Severity, f.Message)
			if f.Disposition != "" {
				p("  - %s", f.Disposition)
			}
		}
		if bl.More > 0 {
			p("- %d more finding(s): `stages/check.txt`", bl.More)
		}
	}
	if len(r.fixFirst) > 0 {
		p("")
		p("## Fix in the project first")
		p("")
		p("No waiver covers these (only cross-reference rules are waived); each is fixed in its source.")
		p("")
		for _, f := range r.fixFirst {
			p("- %s: %s", f[0], f[1])
		}
	}
	return b.String()
}

func list(p func(string, ...any), cs []count, suffix string) {
	if len(cs) == 0 {
		p("None.")
		return
	}
	for _, c := range cs {
		p("- %s: %d%s", c.Key, c.Count, suffix)
	}
}

func (r *rehearsal) doc() map[string]any {
	files := []any{}
	for _, f := range r.files {
		files = append(files, map[string]any{"path": f.Path, "bytes": f.Bytes})
	}
	waivers := []any{}
	for _, c := range r.waivers {
		waivers = append(waivers, map[string]any{"rule": c.Key, "count": c.Count})
	}
	baselines := []any{}
	for _, c := range r.baselines {
		baselines = append(baselines, map[string]any{"path": c.Key, "entries": c.Count})
	}
	byKind := map[string]any{}
	for _, c := range r.owed {
		byKind[c.Key] = c.Count
	}
	owed := r.claims.Doc()
	owed["by_kind"], owed["total"] = byKind, r.owedTotal
	stageList := []any{}
	for _, s := range r.stageOutcomes {
		stageList = append(stageList, map[string]any{"name": s.Key, "outcome": r.outcomes[s.Key]})
	}
	stories := []any{}
	for _, s := range r.planStories {
		stories = append(stories, map[string]any{"kind": s.Kind, "tier": s.Tier, "count": s.Count})
	}
	plan := map[string]any{"stories": stories, "total": r.planTotal, "edges": r.planEdges, "planned_ids": r.plannedIDs, "may_exist": r.mayExist,
		"wire": map[string]any{"stories": r.wireStories, "ids": r.wireIDs, "by": r.wireBy}}
	if r.planError != "" {
		plan["error"] = r.planError
	}
	blocks := []any{}
	for _, bl := range r.blocks {
		fs := []any{}
		for _, f := range bl.Findings {
			fs = append(fs, map[string]any{"severity": strings.TrimSpace(f.Severity), "message": f.Message,
				"disposition": f.Disposition, "waiver": f.Waiver})
		}
		blocks = append(blocks, map[string]any{"stage": bl.Stage, "outcome": bl.Outcome, "findings": fs, "more": bl.More})
	}
	fixFirst := []any{}
	for _, f := range r.fixFirst {
		fixFirst = append(fixFirst, map[string]any{"stage": f[0], "message": f[1]})
	}
	blocked := append([]string{}, r.blocked...)
	outside := append([]string{}, r.outside...)
	return map[string]any{
		"version": 1, "project": r.name, "head": r.head, "config": r.cfgLabel, "backlog": r.backlogLine,
		"dirty":         map[string]any{"changed": r.changed, "untracked": r.untracked},
		"outside_links": outside, "files": files, "waivers": waivers, "baselines": baselines,
		"owed":    owed,
		"blocked": blocked, "check": map[string]any{"result": r.checkLine, "stages": stageList},
		"plan": plan, "blocks": blocks, "fix_first": fixFirst,
	}
}
