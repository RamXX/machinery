package evidence

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/tags"
)

// Claimant is a test function with a marker for an id directly above it,
// or a file with a file-level marker for the id and at least one test
// function, Func "" (section 10.3, "Every claimant").
type Claimant struct{ File, Func string }

// String is the test of the claimant's adapter receipt: <file>::<function>,
// or <file> for a file.
func (c Claimant) String() string {
	if c.Func == "" {
		return c.File
	}
	return c.File + "::" + c.Func
}

// claimantOf is the claimant an adapter receipt is for.
func claimantOf(r Receipt) Claimant {
	file, fn, _ := strings.Cut(r.Test, "::")
	if r.Granularity == project.GranularityFile {
		fn = ""
	}
	return Claimant{File: file, Func: fn}
}

// readClaims reads the claims of the Go test files (section 10.3, "Whose
// claim"): one unit per attribution of an id of kind inv, sc, fm or ck, sorted
// by id, file and function; the claims above functions the adapter does not
// judge; and the tc and cx ids claimed, which a verdict never proves.
func readClaims(d *design.Design, files map[string]*goTestFile, found []tags.Tag) (units []claimUnit, unjudged map[claimUnit]bool, effects, replay map[string]bool) {
	unjudged, effects, replay = map[claimUnit]bool{}, map[string]bool{}, map[string]bool{}
	ids, scenarios := d.DeclaredIDs(), d.ScenarioIDs()
	seen := map[claimUnit]bool{}
	for _, t := range found {
		f := files[t.File]
		if t.Kind != tags.Verifies || f == nil {
			continue
		}
		for _, tok := range t.IDs {
			for _, id := range tags.Resolve(t, tok, ids, scenarios) {
				o := d.Obligation(id)
				switch {
				case o == nil:
				case o.Kind == "tc":
					effects[id] = true
				case o.Kind == "cx":
					replay[id] = true
				case contains(project.AdapterKinds, o.Kind):
					fn, test := f.owner(t.Line)
					u := claimUnit{ID: id, File: t.File, Func: fn}
					if fn != "" && !test {
						unjudged[u] = true
						continue
					}
					if !seen[u] {
						seen[u] = true
						units = append(units, u)
					}
				}
			}
		}
	}
	sort.Slice(units, func(i, j int) bool {
		x, y := units[i], units[j]
		if x.ID != y.ID {
			return x.ID < y.ID
		}
		if x.File != y.File {
			return x.File < y.File
		}
		return x.Func < y.Func
	})
	return units, unjudged, effects, replay
}

// stagedSource reports a Go file that an entry of evidence.adapter.sources
// with files names: a test source whatever its name (section 10.3).
func stagedSource(p *project.Project) func(string) bool {
	entries := p.Cfg.AdapterSources()
	return func(rel string) bool {
		if !strings.HasSuffix(rel, ".go") {
			return false
		}
		for _, s := range entries {
			if !s.Dir && project.Match(s.Files, rel) {
				return true
			}
		}
		return false
	}
}

// claimants are the claimants of every id under trace.tests, and what the
// entries of evidence.adapter.excused cover of them.
type claimants struct {
	of      map[string][]Claimant // id -> its claimants, sorted
	excused map[Claimant]bool     // covered by an entry
	files   map[string]*goTestFile
	// uses: one line per entry in effect; problems: its patterns that cover
	// nothing
	uses, problems []string
}

// readClaimants reads the claimants of the project's Go test files: the
// _test.go files under trace.tests and the staged sources the entries of
// evidence.adapter.sources name there, as the adapter reads them.
func readClaimants(p *project.Project, d *design.Design) (*claimants, error) {
	return readClaimantsWith(p, d, p.Cfg.AdapterClaimsAll())
}

// readClaimantsWith reads the claimants, and with excuse applies the entries
// of evidence.adapter.excused to them.
func readClaimantsWith(p *project.Project, d *design.Design, excuse bool) (*claimants, error) {
	c := &claimants{of: map[string][]Claimant{}, excused: map[Claimant]bool{}, files: map[string]*goTestFile{}}
	if p.Cfg.Trace == nil || len(p.Cfg.Trace.Tests) == 0 {
		return c, nil
	}
	list, err := tags.TestFiles(p)
	if err != nil {
		return nil, err
	}
	files, err := readGoTests(p, list, stagedSource(p))
	if err != nil {
		return nil, err
	}
	c.files = files
	found, err := tags.Scan(p, sortedKeys(files))
	if err != nil {
		return nil, err
	}
	units, _, _, _ := readClaims(d, files, found)
	for _, u := range units {
		if f := files[u.File]; u.Func == "" && (f.Broken || len(f.Funcs) == 0) {
			continue // no test function: no claimant
		}
		c.of[u.ID] = append(c.of[u.ID], Claimant{File: u.File, Func: u.Func})
	}
	if excuse {
		c.excuse(p.Cfg.AdapterExcused())
	}
	return c, nil
}

// covers reports whether an entry of evidence.adapter.excused covers a
// claimant: its file matches the tests of the entry, and its function, or
// for a file every test function of it, a functions pattern when the entry
// has any.
func (c *claimants) covers(e project.AdapterExcused, cl Claimant) bool {
	if !project.Match(e.Tests, cl.File) {
		return false
	}
	for _, fn := range functionsOf(cl, c.files) {
		if !e.MatchFunction(fn) {
			return false
		}
	}
	return true
}

// excuse applies the entries of evidence.adapter.excused (section 10.3).
// Each entry is described once, and each of its patterns that covers no
// claimant is a problem.
func (c *claimants) excuse(entries []project.AdapterExcused) {
	files := c.files
	for i, e := range entries {
		covered, byID := map[Claimant]bool{}, map[string]bool{}
		globHit, fnHit := map[string]bool{}, map[string]bool{}
		for id, cls := range c.of {
			for _, cl := range cls {
				for _, g := range e.Tests {
					if project.Match([]string{g}, cl.File) {
						globHit[g] = true
						for _, f := range e.Functions {
							for _, fn := range functionsOf(cl, files) {
								if ok, _ := path.Match(f, fn); ok {
									fnHit[f] = true
								}
							}
						}
					}
				}
				if c.covers(e, cl) {
					covered[cl], byID[id] = true, true
					c.excused[cl] = true
				}
			}
		}
		c.uses = append(c.uses, fmt.Sprintf("adapter excused: entry %d (%s): %d claimant(s) of %d id(s): %s", i+1, e.Name(), len(covered), len(byID), e.Reason))
		for _, g := range e.Tests {
			if !globHit[g] {
				c.problems = append(c.problems, fmt.Sprintf("adapter excused: entry %d: tests %q covers no claimant: correct the glob or remove it", i+1, g))
			}
		}
		for _, f := range e.Functions {
			if !fnHit[f] {
				c.problems = append(c.problems, fmt.Sprintf("adapter excused: entry %d: functions %q covers no claimant of its tests: correct the pattern or remove it", i+1, f))
			}
		}
	}
}

// functionsOf returns the test functions a claimant stands for: its own,
// or every one of its file.
func functionsOf(cl Claimant, files map[string]*goTestFile) []string {
	if cl.Func != "" {
		return []string{cl.Func}
	}
	return files[cl.File].Funcs
}

// CoveredIDs reads the claimants of the project and returns a function that
// gives the ids, sorted, a claimant of which an entry of
// evidence.adapter.excused would cover. The ratchet asks it of the pairs an
// entry adds (section 11.3).
func CoveredIDs(p *project.Project, d *design.Design) (func(project.AdapterExcused) []string, error) {
	cs, err := readClaimantsWith(p, d, false)
	if err != nil {
		return nil, err
	}
	return func(e project.AdapterExcused) []string {
		var res []string
		for _, id := range sortedKeys(cs.of) {
			for _, cl := range cs.of[id] {
				if cs.covers(e, cl) {
					res = append(res, id)
					break
				}
			}
		}
		return res
	}, nil
}

// open returns the claimants of an id that are not excused, and how many are.
func (c *claimants) open(id string) (open []Claimant, excused int) {
	for _, cl := range c.of[id] {
		if c.excused[cl] {
			excused++
		} else {
			open = append(open, cl)
		}
	}
	return open, excused
}

// Partly is an id that is partly run: its adapter evidence would prove it
// if every claimant that is not excused had passed (section 10.3).
type Partly struct {
	Passed, Of, Excused int
	// Missing are the claimants without a pass, sorted; Skipped those of
	// them whose receipt is a skip
	Missing []Claimant
	Skipped map[Claimant]bool
}

// Text is "<n> of <m> claimants passed", with the excused ones when there are.
func (pr Partly) Text() string {
	s := fmt.Sprintf("%d of %d claimants passed", pr.Passed, pr.Of)
	if pr.Excused > 0 {
		s += fmt.Sprintf(" (%d claimant(s) excused)", pr.Excused)
	}
	return s
}

// Partly returns what is missing for a partly run id (section 10.3).
func (e *Evaluation) Partly(id string) (Partly, bool) {
	pr, ok := e.partly[id]
	return pr, ok
}

// PartlyIDs returns the ids that are partly run, sorted.
func (e *Evaluation) PartlyIDs() []string { return sortedKeys(e.partly) }

// ExcusedOf returns how many claimants of a proven id are excused.
func (e *Evaluation) ExcusedOf(id string) int { return e.excusedOK[id] }

// ExcusedCounts returns how many ids are proven with claimants excused, how
// many claimants that is, and how many ids no claimant of which can run here.
func (e *Evaluation) ExcusedCounts() (proven, claimants, none int) {
	for _, n := range e.excusedOK {
		proven, claimants = proven+1, claimants+n
	}
	for id := range e.noClaimant {
		if e.states[id] == Unproven {
			none++
		}
	}
	return proven, claimants, none
}

// NoClaimant returns, for an id that is not proven because every claimant of
// it is excused, how many claimants are.
func (e *Evaluation) NoClaimant(id string) (int, bool) {
	n, ok := e.noClaimant[id]
	return n, ok && e.states[id] == Unproven
}

// HasReceipt reports whether a receipt of a usable run names the id.
func (e *Evaluation) HasReceipt(id string) bool { return e.seen[id] }

// FindingID returns the id a per-id finding of stage verify names after one
// of the prefixes ("unproven: <id>", "partly run: <id>: ..."), and false for
// any other finding.
func FindingID(msg string, prefixes []string) (string, bool) {
	for _, pre := range prefixes {
		if rest, ok := strings.CutPrefix(msg, pre); ok {
			id, _, _ := strings.Cut(rest, " ")
			return strings.TrimSuffix(id, ":"), true
		}
	}
	return "", false
}

// partlyShown is how many missing claimants verify names without --partly.
const partlyShown = 10

// PartlyFindings names the claimants without a pass of the partly run ids
// (of only, when it is set): the first ten in the order they print, and every
// one with all (section 12).
func (e *Evaluation) PartlyFindings(only map[string]bool, all bool) []report.Finding {
	var msgs []string
	for _, id := range e.PartlyIDs() {
		if only != nil && !only[id] {
			continue
		}
		pr := e.partly[id]
		for _, cl := range pr.Missing {
			what := "no verdict"
			if pr.Skipped[cl] {
				what = "skipped"
			}
			msgs = append(msgs, fmt.Sprintf("partly run: %s: %s: %s", id, what, cl))
		}
	}
	sort.Strings(msgs)
	var res []report.Finding
	for i, m := range msgs {
		if !all && i == partlyShown {
			res = append(res, report.Finding{Sev: report.Warn, Msg: fmt.Sprintf("partly run: %d more claimant(s) without a pass: machinery verify --partly lists every one", len(msgs)-partlyShown)})
			break
		}
		res = append(res, report.Finding{Sev: report.Warn, Msg: m})
	}
	return res
}

// SubtestHints names the tests whose adapter passes record a different
// number of subtests in different runs at the commit (of the ids of only,
// when it is set): a hint, not a finding (section 12).
func (e *Evaluation) SubtestHints(only map[string]bool) []string {
	counts := map[string]map[string]int{} // test -> receipts file -> subtests
	for _, r := range e.passes {
		if only != nil && !only[r.ID] {
			continue
		}
		if counts[r.Test] == nil {
			counts[r.Test] = map[string]int{}
		}
		counts[r.Test][r.File] = max(counts[r.Test][r.File], r.Subtests)
	}
	var res []string
	for _, test := range sortedKeys(counts) {
		files := sortedKeys(counts[test])
		differ := false
		var parts []string
		for _, f := range files {
			differ = differ || counts[test][f] != counts[test][files[0]]
			parts = append(parts, fmt.Sprintf("%d in %s", counts[test][f], f))
		}
		if differ {
			res = append(res, fmt.Sprintf("%s: subtests differ between runs at %s: %s", test, gitx.Short(e.Head), strings.Join(parts, ", ")))
		}
	}
	return res
}
