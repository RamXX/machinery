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

// Claimant is what claims an id in a file under trace.tests (section 10.3,
// "Every claimant"): the test function a marker for the id stands directly
// above, in a file the adapter reads, and otherwise the file, Func "".
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
	// unjudged: a file claimant of an id that the adapter cannot give a
	// verdict (Func "") -> why; only a receipt its own test emits satisfies it
	unjudged map[claimUnit]string
	// uses: one line per entry in effect; problems: its patterns that cover
	// nothing
	uses, problems []string
}

// What to do about a partly run id, and about an id no claimant of which can
// run here (sections 12 and 15.6).
const (
	PartlyRemedy     = "run the claimants without a pass at this commit, or excuse those that cannot run here in evidence.adapter.excused"
	NoClaimantRemedy = "run a claimant of it here and narrow evidence.adapter.excused, or prove it with a receipt a test emits"
)

// unreadableClaimant is what the run summary says of a claiming file the
// adapter reads that cannot be read as Go (section 10.3).
func unreadableClaimant(file, why string) string {
	return fmt.Sprintf("%s: cannot be read as Go (%s): its claims count as not passed; fix the file or remove its markers", file, why)
}

// Why the adapter cannot judge a file claimant, where what to do depends on
// it (section 10.3, "A file the adapter cannot judge").
const (
	whyNotGo  = "not a Go file"
	whyBroken = "cannot be read as Go: "
)

// cannotJudge is what verify says of a file claimant the adapter cannot
// judge, with what to do (sections 10.3 and 12): a file that cannot be read
// as Go is to be fixed; a file that is not Go may be no test at all.
func cannotJudge(file, why string) string {
	todo := "it needs a receipt its own test emits, a sources entry if it is a staged Go source, or an excused entry with a reason"
	switch {
	case strings.HasPrefix(why, whyBroken):
		todo = "fix the file so that it can be read as Go, or give it an excused entry with a reason"
	case why == whyNotGo:
		todo += "; if the file is not a test (a README, a golden file), remove its marker for the id"
	}
	return fmt.Sprintf("%s: cannot be judged by the adapter (%s); %s", file, why, todo)
}

// readClaimants reads the claimants of the project: every file under
// trace.tests that carries a marker for an id, as stage trace reads markers.
func readClaimants(p *project.Project, d *design.Design) (*claimants, error) {
	return readClaimantsWith(p, d, p.Cfg.AdapterClaimsAll())
}

// readClaimantsWith reads the claimants, and with excuse applies the entries
// of evidence.adapter.excused to them.
func readClaimantsWith(p *project.Project, d *design.Design, excuse bool) (*claimants, error) {
	c := &claimants{of: map[string][]Claimant{}, excused: map[Claimant]bool{}, files: map[string]*goTestFile{}, unjudged: map[claimUnit]string{}}
	if p.Cfg.Trace == nil || len(p.Cfg.Trace.Tests) == 0 {
		return c, nil
	}
	list, err := tags.TestFiles(p)
	if err != nil {
		return nil, err
	}
	// the files the adapter reads among them; every other file claims as a whole
	if c.files, err = readGoTests(p, list, stagedSource(p)); err != nil {
		return nil, err
	}
	found, err := tags.Scan(p, list)
	if err != nil {
		return nil, err
	}
	if err := c.derive(d, found); err != nil {
		return nil, err
	}
	if excuse {
		c.excuse(p.Cfg.AdapterExcused())
	}
	return c, nil
}

// derive computes the claimants from the markers, and beside them the files
// that claim each id for stage trace, from the same reading: the two must
// name the same files for every id, so that no claim is dropped for what its
// file is called or holds (section 10.3).
func (c *claimants) derive(d *design.Design, found []tags.Tag) error {
	ids, scenarios := d.DeclaredIDs(), d.ScenarioIDs()
	seen, judged := map[claimUnit]bool{}, map[claimUnit]bool{}
	above := map[claimUnit]string{} // the first function, not a test function, a marker of the id stands above
	for _, t := range found {
		if t.Kind != tags.Verifies {
			continue
		}
		for _, tok := range t.IDs {
			for _, id := range tags.Resolve(t, tok, ids, scenarios) {
				if o := d.Obligation(id); o == nil || !contains(project.AdapterKinds, o.Kind) {
					continue
				}
				u := claimUnit{ID: id, File: t.File}
				if f := c.files[t.File]; f != nil && !f.Broken {
					switch fn, test := f.owner(t.Line); {
					case test:
						u.Func = fn
					case fn != "":
						if above[u] == "" {
							above[u] = fn
						}
					case len(f.Funcs) > 0:
						judged[u] = true // the adapter writes a receipt for the file
					}
				}
				if !seen[u] {
					seen[u] = true
					c.of[id] = append(c.of[id], Claimant{File: u.File, Func: u.Func})
				}
			}
		}
	}
	for u := range seen {
		if u.Func == "" && !judged[u] {
			c.unjudged[u] = c.why(u.File, above[u])
		}
	}
	trace := tags.ClaimsOf(d, found).Tests
	for i := range d.Obligations {
		o := &d.Obligations[i]
		if !contains(project.AdapterKinds, o.Kind) {
			continue
		}
		cls := c.of[o.ID]
		sort.Slice(cls, func(i, j int) bool {
			if cls[i].File != cls[j].File {
				return cls[i].File < cls[j].File
			}
			return cls[i].Func < cls[j].Func
		})
		var files []string
		for _, cl := range cls {
			if len(files) == 0 || files[len(files)-1] != cl.File {
				files = append(files, cl.File)
			}
		}
		if strings.Join(files, "\x00") != strings.Join(trace[o.ID], "\x00") {
			return fmt.Errorf("the claimants of %s are in %s and the files that claim it are %s: a defect of machinery; nothing is judged",
				o.ID, fileList(files), fileList(trace[o.ID]))
		}
	}
	return nil
}

// fileList renders files for a message.
func fileList(files []string) string {
	if len(files) == 0 {
		return "no file"
	}
	return strings.Join(files, ", ")
}

// why says why the adapter cannot judge a file claimant (section 10.3, "A
// file the adapter cannot judge"); above is the function, not a test
// function, its marker stands above.
func (c *claimants) why(file, above string) string {
	switch f := c.files[file]; {
	case f == nil && !strings.HasSuffix(file, ".go"):
		return whyNotGo
	case f == nil:
		return "a Go file not named _test.go that no entry of evidence.adapter.sources names"
	case f.Broken:
		return whyBroken + f.Err
	case above != "":
		return "its marker for the id is above " + above + ", which is not a test function"
	}
	return "no test function"
}

// covers reports whether an entry of evidence.adapter.excused covers a
// claimant: its file matches the tests of the entry and, when the entry has
// functions, it is a test function a pattern matches. A file claimant is
// excused by its file, never by a function (section 10.3).
func (c *claimants) covers(e project.AdapterExcused, cl Claimant) bool {
	if !project.Match(e.Tests, cl.File) {
		return false
	}
	if len(e.Functions) == 0 {
		return true
	}
	return cl.Func != "" && e.MatchFunction(cl.Func)
}

// excuse applies the entries of evidence.adapter.excused (section 10.3).
// Each entry is described once, and each of its globs and patterns through
// which it covers no claimant is a problem.
func (c *claimants) excuse(entries []project.AdapterExcused) {
	for i, e := range entries {
		covered, byID := map[Claimant]bool{}, map[string]bool{}
		globCovers, fnCovers := map[string]bool{}, map[string]bool{}
		for id, cls := range c.of {
			for _, cl := range cls {
				if !c.covers(e, cl) {
					continue
				}
				covered[cl], byID[id] = true, true
				c.excused[cl] = true
				for _, g := range e.Tests {
					if project.Match([]string{g}, cl.File) {
						globCovers[g] = true
					}
				}
				for _, f := range e.Functions {
					if ok, _ := path.Match(f, cl.Func); ok {
						fnCovers[f] = true
					}
				}
			}
		}
		c.uses = append(c.uses, fmt.Sprintf("adapter excused: entry %d (%s): %d claimant(s) of %d id(s): %s", i+1, e.Name(), len(covered), len(byID), e.Reason))
		for _, g := range e.Tests {
			if !globCovers[g] {
				c.problems = append(c.problems, fmt.Sprintf("adapter excused: entry %d: tests %q covers no claimant: %s", i+1, g, c.whyGlob(g)))
			}
		}
		for _, f := range e.Functions {
			if !fnCovers[f] {
				c.problems = append(c.problems, fmt.Sprintf("adapter excused: entry %d: functions %q covers no claimant of its tests: %s", i+1, f, c.whyPattern(e, f)))
			}
		}
	}
}

// whyGlob says why a tests glob of an entry covers no claimant, and what to
// do: the functions of the entry leave out every claimant of its files, or it
// matches no claiming file.
func (c *claimants) whyGlob(g string) string {
	for _, cls := range c.of {
		for _, cl := range cls {
			if project.Match([]string{g}, cl.File) {
				return "its functions leave out every claimant of the files it matches: correct the functions or remove the glob"
			}
		}
	}
	return "correct the glob or remove it"
}

// whyPattern says why a functions pattern of an entry covers no claimant,
// and what to do: it matches only functions of files whose claim is
// file-level (the first such file), or nothing.
func (c *claimants) whyPattern(e project.AdapterExcused, f string) string {
	file := ""
	for _, cls := range c.of {
		for _, cl := range cls {
			src := c.files[cl.File]
			if cl.Func != "" || src == nil || !project.Match(e.Tests, cl.File) {
				continue
			}
			for _, fn := range src.Funcs {
				if ok, _ := path.Match(f, fn); ok && (file == "" || cl.File < file) {
					file = cl.File
				}
			}
		}
	}
	if file != "" {
		return "it matches a function of " + file + ", whose claim is file-level; a file-level claimant is excused by its file, not by a function: excuse the file in an entry without functions, or move the marker above the function"
	}
	return "correct the pattern or remove it"
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

// Partly is an id that is partly run: an adapter pass is used for it, and a
// claimant of it that is not excused is not satisfied (section 10.3).
type Partly struct {
	Passed, Of, Excused int
	// Missing are the claimants that are not satisfied, sorted; Skipped those
	// of them whose receipt is a skip; Unjudged those the adapter cannot
	// judge, with why
	Missing  []Claimant
	Skipped  map[Claimant]bool
	Unjudged map[Claimant]string
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

// PartlyFindings names the claimants that are not satisfied of the partly run ids
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
			switch why, cannot := pr.Unjudged[cl]; {
			case cannot:
				msgs = append(msgs, fmt.Sprintf("partly run: %s: %s", id, cannotJudge(cl.File, why)))
			case pr.Skipped[cl]:
				msgs = append(msgs, fmt.Sprintf("partly run: %s: skipped: %s", id, cl))
			default:
				msgs = append(msgs, fmt.Sprintf("partly run: %s: no verdict: %s", id, cl))
			}
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
