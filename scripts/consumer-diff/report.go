package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

const tagStampSkew = "stamp-skew"

// diffItem is one normalized finding whose count differs between the runs.
type diffItem struct {
	Gate      string    `json:"gate"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Count     int       `json:"count"`
	Tags      []string  `json:"tags,omitempty"`
	AllowedBy *allowRef `json:"allowed_by,omitempty"`
	order     int
}

type allowRef struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Reason string `json:"reason"`
}

type tally struct {
	Blocking int `json:"blocking"`
	Warnings int `json:"warnings"`
	Notes    int `json:"notes"`
}

func (t *tally) add(sev string, n int) {
	switch sev {
	case sevError, sevDrift:
		t.Blocking += n
	case sevWarn:
		t.Warnings += n
	default:
		t.Notes += n
	}
}

type gateTally struct {
	Gate string `json:"gate"`
	tally
	Unexplained int `json:"unexplained"`
}

type runSummary struct {
	Binary  string     `json:"binary"`
	Version string     `json:"version"`
	Exit    int        `json:"exit"`
	Status  string     `json:"status,omitempty"`
	Skew    *stampSkew `json:"stamp_skew,omitempty"`
	Writes  []string   `json:"writes_into_copy,omitempty"`
	tally
}

type verdictChange struct {
	Gate string `json:"gate"`
	Old  string `json:"old"`
	New  string `json:"new"`
}

type report struct {
	Design         string          `json:"design"`
	Impl           string          `json:"impl,omitempty"`
	Args           []string        `json:"check_args"`
	Old            runSummary      `json:"old"`
	New            runSummary      `json:"new"`
	NewFindings    []diffItem      `json:"new_findings"`
	NewTally       tally           `json:"new_tally"`
	NewByGate      []gateTally     `json:"new_by_gate"`
	Resolved       []diffItem      `json:"resolved_findings"`
	ResolvedTally  tally           `json:"resolved_tally"`
	ResolvedByGate []gateTally     `json:"resolved_by_gate"`
	Verdicts       []verdictChange `json:"changed_verdicts"`
	Unused         []*allowEntry   `json:"unused_allow_entries"`
	Unexplained    tally           `json:"unexplained"`
	Exit           int             `json:"exit"`
}

func summarize(s *side) runSummary {
	r := s.result
	return runSummary{
		Binary: s.binary, Version: s.version, Exit: r.Exit, Status: r.Status, Skew: r.Skew, Writes: s.writes,
		tally: tally{Blocking: r.count(sevError) + r.count(sevDrift), Warnings: r.count(sevWarn), Notes: r.count(sevNote)},
	}
}

// buildReport diffs the two runs as multisets of normalized findings and
// matches every new gating finding against the allow entries.
func buildReport(oldSide, newSide *side, allows []*allowEntry) *report {
	oldRun, newRun := oldSide.result, newSide.result
	rep := &report{Old: summarize(oldSide), New: summarize(newSide), Unused: []*allowEntry{}}
	gateOrder := map[string]int{}
	for _, r := range []*checkRun{newRun, oldRun} {
		for _, g := range r.Gates {
			if _, ok := gateOrder[g.ID]; !ok {
				gateOrder[g.ID] = len(gateOrder)
			}
		}
	}
	gateOrder[runGate] = len(gateOrder)

	rep.NewFindings = surplus(newRun, oldRun, gateOrder)
	rep.Resolved = surplus(oldRun, newRun, gateOrder)
	for i := range rep.NewFindings {
		it := &rep.NewFindings[i]
		if !(finding{Severity: it.Severity}).gating() {
			continue
		}
		for _, e := range allows {
			if e.matches(finding{Gate: it.Gate, Severity: it.Severity, Message: it.Message}) {
				e.Used += it.Count
				it.AllowedBy = &allowRef{File: e.File, Line: e.Line, Reason: e.Reason}
				break
			}
		}
		if it.AllowedBy == nil {
			rep.Unexplained.add(it.Severity, it.Count)
		}
	}
	for _, e := range allows {
		if e.Used == 0 {
			rep.Unused = append(rep.Unused, e)
		}
	}
	rep.NewTally, rep.NewByGate = tallies(rep.NewFindings)
	rep.ResolvedTally, rep.ResolvedByGate = tallies(rep.Resolved)

	ov, nv := oldRun.verdicts(), newRun.verdicts()
	gates := make([]string, 0, len(gateOrder))
	for g := range gateOrder {
		gates = append(gates, g)
	}
	sort.Slice(gates, func(i, j int) bool { return gateOrder[gates[i]] < gateOrder[gates[j]] })
	for _, g := range gates {
		if g == runGate {
			continue
		}
		o, n := ov[g], nv[g]
		if o == "" {
			o = verdictAbsent
		}
		if n == "" {
			n = verdictAbsent
		}
		if o != n {
			rep.Verdicts = append(rep.Verdicts, verdictChange{Gate: g, Old: o, New: n})
		}
	}
	if rep.Unexplained.Blocking+rep.Unexplained.Warnings > 0 {
		rep.Exit = exitUnexplained
	}
	return rep
}

// surplus lists the findings a carries more often than b, tagging the ones
// a version-stamp skew in run a accounts for.
func surplus(a, b *checkRun, gateOrder map[string]int) []diffItem {
	have := map[string]int{}
	for _, f := range b.findings() {
		have[f.key()]++
	}
	byKey := map[string]*diffItem{}
	var items []*diffItem
	for i, f := range a.findings() {
		k := f.key()
		if have[k] > 0 {
			have[k]--
			continue
		}
		if it, ok := byKey[k]; ok {
			it.Count++
			continue
		}
		it := &diffItem{Gate: f.Gate, Severity: f.Severity, Message: f.Message, Count: 1, order: i}
		if a.Skew != nil && (f.Severity == sevDrift || (f.Gate == runGate && skewRE.MatchString(f.Message))) {
			it.Tags = append(it.Tags, tagStampSkew)
		}
		byKey[k] = it
		items = append(items, it)
	}
	sort.SliceStable(items, func(i, j int) bool {
		x, y := items[i], items[j]
		if gateOrder[x.Gate] != gateOrder[y.Gate] {
			return gateOrder[x.Gate] < gateOrder[y.Gate]
		}
		if sevRank[x.Severity] != sevRank[y.Severity] {
			return sevRank[x.Severity] < sevRank[y.Severity]
		}
		return x.order < y.order
	})
	out := make([]diffItem, len(items))
	for i, it := range items {
		out[i] = *it
	}
	return out
}

func tallies(items []diffItem) (tally, []gateTally) {
	var total tally
	var by []gateTally
	idx := map[string]int{}
	for _, it := range items {
		total.add(it.Severity, it.Count)
		i, ok := idx[it.Gate]
		if !ok {
			i = len(by)
			idx[it.Gate] = i
			by = append(by, gateTally{Gate: it.Gate})
		}
		by[i].add(it.Severity, it.Count)
		if (finding{Severity: it.Severity}).gating() && it.AllowedBy == nil {
			by[i].Unexplained += it.Count
		}
	}
	if by == nil {
		by = []gateTally{}
	}
	return total, by
}

func (r *report) writeJSON(w io.Writer) error {
	if r.NewFindings == nil {
		r.NewFindings = []diffItem{}
	}
	if r.Resolved == nil {
		r.Resolved = []diffItem{}
	}
	if r.Verdicts == nil {
		r.Verdicts = []verdictChange{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func (r *report) writeText(w io.Writer) {
	fmt.Fprintf(w, "consumer-diff: %s -> %s over %s\n", r.Old.Version, r.New.Version, r.Design)
	fmt.Fprintf(w, "  run      machinery %s\n", strings.Join(r.Args, " "))
	for _, s := range []struct {
		label string
		sum   runSummary
	}{{"old", r.Old}, {"new", r.New}} {
		fmt.Fprintf(w, "  %s      %s (%s): %d blocking, %d warnings, %d notes; exit %d\n",
			s.label, s.sum.Version, s.sum.Binary, s.sum.Blocking, s.sum.Warnings, s.sum.Notes, s.sum.Exit)
	}
	for _, s := range []struct {
		label string
		sum   runSummary
	}{{"old", r.Old}, {"new", r.New}} {
		if s.sum.Skew == nil {
			fmt.Fprintf(w, "  stamps   %s run: no version-stamp skew\n", s.label)
			continue
		}
		fmt.Fprintf(w, "  stamps   %s run: artifacts stamped by machinery %s, running %s; its DRIFT and the skew note carry [%s]\n",
			s.label, strings.Join(s.sum.Skew.Stamped, ", "), s.sum.Skew.Running, tagStampSkew)
	}

	fmt.Fprintf(w, "\n== new findings: %s ==\n", tallyText(r.NewTally))
	writeByGate(w, r.NewByGate, true)
	writeItems(w, r.NewFindings, true)

	fmt.Fprintf(w, "\n== resolved findings: %s ==\n", tallyText(r.ResolvedTally))
	writeByGate(w, r.ResolvedByGate, false)
	writeItems(w, r.Resolved, false)

	fmt.Fprintln(w, "\n== changed gate verdicts ==")
	if len(r.Verdicts) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, v := range r.Verdicts {
		fmt.Fprintf(w, "  %-14s %s -> %s\n", v.Gate, v.Old, v.New)
	}

	fmt.Fprintln(w, "\n== unused allow entries ==")
	if len(r.Unused) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, e := range r.Unused {
		fmt.Fprintf(w, "  %s  %s %s /%s/  %s\n", e, e.Gate, strings.ToLower(e.Severity), e.Pattern, e.Reason)
	}

	for _, s := range []struct {
		label string
		sum   runSummary
	}{{"old", r.Old}, {"new", r.New}} {
		if len(s.sum.Writes) > 0 {
			fmt.Fprintf(w, "\nnote: the %s binary wrote into its private copy of the tree (the original is unchanged): %s\n", s.label, preview(s.sum.Writes, 10))
		}
	}

	if r.Exit == exitOK {
		fmt.Fprintf(w, "\nPASS: every new blocking finding and warning is explained (%d blocking, %d warnings new)\n", r.NewTally.Blocking, r.NewTally.Warnings)
		return
	}
	fmt.Fprintf(w, "\nFAIL: %d unexplained new finding(s): %d blocking, %d warnings\n",
		r.Unexplained.Blocking+r.Unexplained.Warnings, r.Unexplained.Blocking, r.Unexplained.Warnings)
}

func tallyText(t tally) string {
	return fmt.Sprintf("%d blocking, %d warnings, %d notes", t.Blocking, t.Warnings, t.Notes)
}

func writeByGate(w io.Writer, by []gateTally, withUnexplained bool) {
	if len(by) == 0 {
		fmt.Fprintln(w, "  (none)")
		return
	}
	if withUnexplained {
		fmt.Fprintf(w, "  %-14s %8s %8s %6s %11s\n", "gate", "blocking", "warnings", "notes", "unexplained")
	} else {
		fmt.Fprintf(w, "  %-14s %8s %8s %6s\n", "gate", "blocking", "warnings", "notes")
	}
	for _, g := range by {
		if withUnexplained {
			fmt.Fprintf(w, "  %-14s %8d %8d %6d %11d\n", g.Gate, g.Blocking, g.Warnings, g.Notes, g.Unexplained)
		} else {
			fmt.Fprintf(w, "  %-14s %8d %8d %6d\n", g.Gate, g.Blocking, g.Warnings, g.Notes)
		}
	}
	fmt.Fprintln(w)
}

func writeItems(w io.Writer, items []diffItem, withAllow bool) {
	for _, it := range items {
		var marks []string
		if it.Count > 1 {
			marks = append(marks, fmt.Sprintf("x%d", it.Count))
		}
		for _, t := range it.Tags {
			marks = append(marks, "["+t+"]")
		}
		if withAllow && it.AllowedBy != nil {
			marks = append(marks, fmt.Sprintf("[allowed %s:%d]", it.AllowedBy.File, it.AllowedBy.Line))
		}
		prefix := fmt.Sprintf("  %-14s %-5s ", it.Gate, it.Severity)
		if len(marks) > 0 {
			prefix += strings.Join(marks, " ") + " "
		}
		fmt.Fprintf(w, "%s%s\n", prefix, strings.ReplaceAll(it.Message, "\n", "\n"+strings.Repeat(" ", 24)))
	}
}
