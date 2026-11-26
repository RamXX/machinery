package pm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/report"
)

// WaveStory is one story of a wave.
type WaveStory struct {
	ID    string   `json:"id"`
	Tier  string   `json:"tier"`
	Title string   `json:"title"`
	Route string   `json:"route"`
	Scope []string `json:"scope"`
}

// Waves is the layering of the open backlog graph (section 15.3).
type Waves struct {
	Contract *Contract
	Layers   [][]WaveStory
	Stage    *report.Stage
}

// BuildWaves layers the stories that are not closed, not deferred, not
// epics, not intake items, not decisions and not questions by their blocking
// edges.
func BuildWaves(d *design.Design, issues []backlog.Issue) *Waves {
	w := &Waves{Contract: ContractOf(d), Stage: report.New("waves")}
	s := w.Stage
	intakes := map[string]bool{}
	for i := range issues {
		if backlog.ParseTags(&issues[i]).Intake {
			intakes[issues[i].ID] = true
		}
	}
	open := map[string]*backlog.Issue{}
	var ids []string
	for i := range issues {
		is := &issues[i]
		if is.Open() && is.Type != "epic" && !intakes[is.ID] && !IsDecision(is) && !IsQuestion(is) {
			open[is.ID] = is
			ids = append(ids, is.ID)
		}
	}
	sort.Strings(ids)
	designIDs(s, ids, open, intakes)
	blockers := map[string][]string{}
	edges := 0
	for _, id := range ids {
		for _, b := range open[id].Blockers {
			if open[b] != nil && b != id {
				blockers[id] = append(blockers[id], b)
				edges++
			}
		}
	}
	placed := map[string]int{}
	for {
		var layer []string
		for _, id := range ids {
			if _, done := placed[id]; done {
				continue
			}
			ready := true
			for _, b := range blockers[id] {
				if _, done := placed[b]; !done {
					ready = false
				}
			}
			if ready {
				layer = append(layer, id)
			}
		}
		if len(layer) == 0 {
			break
		}
		n := len(w.Layers) + 1
		var row []WaveStory
		for _, id := range layer {
			placed[id] = n
			row = append(row, waveStory(d, open[id], s))
		}
		w.Layers = append(w.Layers, row)
		competingWriters(s, n, row)
	}
	for _, c := range cycles(ids, blockers, placed) {
		s.Errorf("cycle: %s", strings.Join(c, " -> "))
	}
	s.Count(len(ids), "stories").Count(edges, "edges").Count(len(w.Layers), "waves")
	if len(ids) == 0 {
		s.NotDemonstrated("nothing to check")
	}
	return w
}

// designIDs holds every story in the design's name to its ids: one labelled
// machinery or belonging to an intake names the ids it proves, unless it is a
// capstone. The other stories without ids are only counted (section 15.3).
func designIDs(s *report.Stage, ids []string, open map[string]*backlog.Issue, intakes map[string]bool) {
	outside := 0
	for _, id := range ids {
		is := open[id]
		t := backlog.ParseTags(is)
		if len(t.IDs) > 0 || t.Capstone {
			continue
		}
		if is.HasLabel("machinery") || t.IntakeOf != "" || intakes[is.Parent] {
			s.Errorf("no design ids: %s: a story proves ids; a requirement starts with machinery intake new", id)
			continue
		}
		outside++
	}
	if outside > 0 {
		s.Warnf("%d open stories outside machinery (no design ids)", outside)
	}
}

func waveStory(d *design.Design, is *backlog.Issue, s *report.Stage) WaveStory {
	t := backlog.ParseTags(is)
	tier, _ := StoryTier(d, is, t.IDs)
	if len(t.Scope) == 0 {
		s.Warnf("no scope declared: %s", is.ID)
	}
	scope := append([]string{}, t.Scope...)
	sort.Strings(scope)
	return WaveStory{ID: is.ID, Tier: tier, Title: is.Title, Route: Route(d, tier), Scope: scope}
}

// cycles finds the blocking cycles among the stories no wave could hold.
// Each cycle starts and ends at its smallest id.
func cycles(ids []string, blockers map[string][]string, placed map[string]int) [][]string {
	var res [][]string
	seen := map[string]bool{}
	for _, start := range ids {
		if _, ok := placed[start]; ok {
			continue
		}
		path := findCycle(start, start, blockers, placed, map[string]bool{})
		if path == nil {
			continue
		}
		min := 0
		for i, id := range path {
			if id < path[min] {
				min = i
			}
		}
		rot := append(append([]string{}, path[min:]...), path[:min]...)
		key := strings.Join(rot, " ")
		if seen[key] {
			continue
		}
		seen[key] = true
		res = append(res, append(rot, rot[0]))
	}
	sort.Slice(res, func(i, j int) bool { return strings.Join(res[i], " ") < strings.Join(res[j], " ") })
	return res
}

// findCycle returns a path from cur back to start, without repeating start.
func findCycle(start, cur string, blockers map[string][]string, placed map[string]int, visiting map[string]bool) []string {
	visiting[cur] = true
	for _, b := range blockers[cur] {
		if _, ok := placed[b]; ok {
			continue
		}
		if b == start {
			return []string{cur}
		}
		if visiting[b] {
			continue
		}
		if rest := findCycle(start, b, blockers, placed, visiting); rest != nil {
			return append([]string{cur}, rest...)
		}
	}
	return nil
}

// competingWriters reports overlapping scopes of two stories of one wave.
func competingWriters(s *report.Stage, n int, row []WaveStory) {
	for i := 0; i < len(row); i++ {
		for j := i + 1; j < len(row); j++ {
			seen := map[string]bool{}
			for _, ga := range row[i].Scope {
				for _, gb := range row[j].Scope {
					g, ok := overlap(ga, gb)
					if ok && !seen[g] {
						seen[g] = true
						s.Errorf("competing writers in wave %d: %s and %s both write %s", n, row[i].ID, row[j].ID, g)
					}
				}
			}
		}
	}
}

// overlap reports whether two scope globs overlap: equal, or one glob's fixed
// prefix contains the other's. The narrower glob is returned.
func overlap(a, b string) (string, bool) {
	if a == b {
		return a, true
	}
	pa, pb := fixedPrefix(a), fixedPrefix(b)
	switch {
	case contains(pa, pb):
		return b, true
	case contains(pb, pa):
		return a, true
	}
	return "", false
}

// fixedPrefix is the part of a glob before its first segment with a
// wildcard, ending in "/"; a glob without a wildcard is its own prefix.
func fixedPrefix(g string) string {
	segs := strings.Split(g, "/")
	for i, seg := range segs {
		if strings.ContainsAny(seg, "*?[{") {
			return strings.Join(segs[:i], "/") + "/"
		}
	}
	return g
}

// contains reports whether directory prefix a contains path or prefix b.
func contains(a, b string) bool {
	if a == "/" {
		return true
	}
	return strings.HasSuffix(a, "/") && strings.HasPrefix(b, a)
}

// Print writes the waves and the stage block, and returns the exit code.
func (w *Waves) Print(out io.Writer) int {
	fmt.Fprintf(out, "contract: %s (%d design file(s))\n", w.Contract.Short(), len(w.Contract.Paths))
	for i, row := range w.Layers {
		fmt.Fprintf(out, "wave %d\n", i+1)
		for _, st := range row {
			scope := "none"
			if len(st.Scope) > 0 {
				scope = strings.Join(st.Scope, " ")
			}
			fmt.Fprintf(out, "  %s  %s  %s\n    route: %s\n    scope: %s\n", st.ID, st.Tier, st.Title, st.Route, scope)
		}
	}
	return report.PrintAll(out, "waves", []*report.Stage{w.Stage})
}

// JSON writes the same data as one JSON document, and returns the exit code.
func (w *Waves) JSON(out io.Writer) int {
	type wave struct {
		Wave    int         `json:"wave"`
		Stories []WaveStory `json:"stories"`
	}
	doc := struct {
		Contract    string    `json:"contract"`
		DesignFiles int       `json:"design_files"`
		Waves       []wave    `json:"waves"`
		Stage       StageJSON `json:"stage"`
	}{Contract: w.Contract.Hash, DesignFiles: len(w.Contract.Paths), Waves: []wave{}, Stage: StageDoc(w.Stage)}
	for i, row := range w.Layers {
		doc.Waves = append(doc.Waves, wave{i + 1, row})
	}
	out.Write(JSON(doc))
	return report.ExitCode([]*report.Stage{w.Stage})
}

// JSON renders a document indented, without HTML escaping, with a final newline.
func JSON(v any) []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
	return b.Bytes()
}

// Wave1 lists the story ids of the first wave.
func (w *Waves) Wave1() []string {
	res := []string{}
	if len(w.Layers) > 0 {
		for _, st := range w.Layers[0] {
			res = append(res, st.ID)
		}
	}
	return res
}

// StageJSON is a stage block as JSON.
type StageJSON struct {
	Name     string        `json:"name"`
	Checked  []CountJSON   `json:"checked"`
	Findings []FindingJSON `json:"findings"`
	Outcome  string        `json:"outcome"`
}

// CountJSON is one entry of the checked line.
type CountJSON struct {
	N    int    `json:"n"`
	Noun string `json:"noun"`
}

// FindingJSON is one finding.
type FindingJSON struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// StageDoc renders a stage as JSON data, findings sorted as in the text block.
func StageDoc(s *report.Stage) StageJSON {
	doc := StageJSON{Name: s.Name, Checked: []CountJSON{}, Findings: []FindingJSON{}}
	for _, c := range s.Checked {
		doc.Checked = append(doc.Checked, CountJSON{c.N, c.Noun})
	}
	fs := append([]report.Finding(nil), s.Findings...)
	sort.SliceStable(fs, func(i, j int) bool {
		if fs[i].Sev != fs[j].Sev {
			return fs[i].Sev < fs[j].Sev
		}
		return fs[i].Msg < fs[j].Msg
	})
	for _, f := range fs {
		sev := map[report.Severity]string{report.Error: "ERROR", report.Drift: "DRIFT", report.Warn: "warn"}[f.Sev]
		doc.Findings = append(doc.Findings, FindingJSON{sev, f.Msg})
	}
	var b strings.Builder
	report.Print(&b, s)
	lines := strings.Split(strings.TrimSpace(b.String()), "\n")
	doc.Outcome = lines[len(lines)-1]
	return doc
}
