package evidence

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/tags"
)

// GoTestJSON is the name of the run adapter for go test (section 10.3).
const GoTestJSON = "go-test-json"

// SourceAdapter is the source of a receipt a run adapter wrote.
const SourceAdapter = "adapter"

// UsageError is a refusal of the adapter before anything runs (exit 2).
type UsageError struct{ Msg string }

func (e *UsageError) Error() string { return e.Msg }

func refuse(format string, a ...any) error {
	return &UsageError{Msg: "--adapter " + GoTestJSON + ": " + fmt.Sprintf(format, a...)}
}

// claimUnit is one attribution of a claimed id: to a test function, or to
// the whole file when Func is "".
type claimUnit struct{ ID, File, Func string }

// GoTest is the run adapter go-test-json: it reads the event stream of one
// go test command and writes the receipts of the ids its test files claim.
type GoTest struct {
	// Argv is the command as it runs, -json added.
	Argv []string

	p *project.Project
	// tags are the build tags of the run; nil when they are not known
	tags []string
	// listFlags are the flags of the command that decide which files a
	// package builds; go list is asked with them
	listFlags []string
	files     map[string]*goTestFile // the test files under trace.tests
	other     map[string]*goTestFile // other files parsed as candidates, by project-relative or absolute path
	sources   []project.AdapterSource
	units     []claimUnit
	effects   map[string]bool // claimed tc ids
	replay    map[string]bool // claimed cx ids

	// streams are the event streams read: the command's standard output, or
	// with fromDir the files the command left in MACHINERY_GO_TEST_JSON
	streams  []*goStream
	fromDir  bool
	summary  []string
	receipts [][]byte
}

// goStream is one stream of go test -json.
type goStream struct {
	name     string // the file, in streams mode
	pkgs     map[string]*goPackage
	nameless int // verdicts of test functions that name no package
}

// goPackage is what a stream said about one package.
type goPackage struct {
	end      string              // the package-level pass, fail or skip
	cached   bool                // answered from the test cache
	verdicts map[string][]string // test function -> its pass, fail and skip events
}

// NewGoTest reads the command and the claims of the test files, and refuses
// what the adapter cannot join (section 10.3).
//
// With streams the command is any command: it is not read, and the streams
// are the files it leaves in MACHINERY_GO_TEST_JSON.
func NewGoTest(p *project.Project, d *design.Design, argv []string, streams bool) (*GoTest, error) {
	a := &GoTest{p: p, other: map[string]*goTestFile{}, effects: map[string]bool{}, replay: map[string]bool{}, fromDir: streams}
	for _, w := range strings.Fields(os.Getenv("GOFLAGS")) { // a count there reaches every go test of the run
		if name, val, _ := flagOf(w); (name == "count" || name == "test.count") && val != "1" {
			return nil, refuse("GOFLAGS has -%s=%s, which is refused: take it out of GOFLAGS", name, val)
		}
	}
	if streams {
		a.Argv = argv // the command is not read, and its build tags are not known
	} else if err := a.readCommand(argv); err != nil {
		return nil, err
	}
	if p.Cfg.Trace == nil || len(p.Cfg.Trace.Tests) == 0 {
		return nil, refuse("trace.tests is not set: the adapter joins the run to the claims of the test files; list them in machinery.yaml")
	}
	files, err := tags.TestFiles(p)
	if err != nil {
		return nil, err
	}
	if a.files, err = readGoTests(p, files); err != nil {
		return nil, err
	}
	var goFiles []string
	for rel := range a.files {
		goFiles = append(goFiles, rel)
	}
	sort.Strings(goFiles)
	a.sources = p.Cfg.AdapterSources()
	for i, s := range a.sources {
		if len(matching(goFiles, s.Files)) == 0 {
			return nil, refuse("evidence.adapter.sources: entry %d: files match no file under trace.tests", i+1)
		}
	}
	found, err := tags.Scan(p, goFiles)
	if err != nil {
		return nil, err
	}
	ids, scenarios := d.DeclaredIDs(), d.ScenarioIDs()
	seen := map[claimUnit]bool{}
	for _, t := range found {
		if t.Kind != tags.Verifies {
			continue
		}
		for _, tok := range t.IDs {
			for _, id := range tags.Resolve(t, tok, ids, scenarios) {
				o := d.Obligation(id)
				switch {
				case o == nil:
				case o.Kind == "tc":
					a.effects[id] = true
				case o.Kind == "cx":
					a.replay[id] = true
				case contains(project.AdapterKinds, o.Kind):
					if u := (claimUnit{ID: id, File: t.File, Func: a.files[t.File].owner(t.Line)}); !seen[u] {
						seen[u] = true
						a.units = append(a.units, u)
					}
				}
			}
		}
	}
	sort.Slice(a.units, func(i, j int) bool {
		x, y := a.units[i], a.units[j]
		if x.ID != y.ID {
			return x.ID < y.ID
		}
		if x.File != y.File {
			return x.File < y.File
		}
		return x.Func < y.Func
	})
	return a, nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func matching(files, globs []string) []string {
	var res []string
	for _, f := range files {
		if project.Match(globs, f) {
			res = append(res, f)
		}
	}
	return res
}

// readCommand holds the command to go test, adds -json and reads -count and
// -tags among the flags before -args.
func (a *GoTest) readCommand(argv []string) error {
	if len(argv) < 2 || filepath.Base(argv[0]) != "go" || argv[1] != "test" {
		n := len(argv)
		if n > 2 {
			n = 2
		}
		return refuse("the command must be go test, got %s", strings.Join(argv[:n], " "))
	}
	rest := argv[2:]
	insert, hasJSON, hasCount, afterArgs := 0, false, false, false
	var tagList string
	for i := 0; i < len(rest); i++ {
		name, val, hasVal := flagOf(rest[i])
		next := func() string { // the value of a flag written as two words
			if !hasVal && i+1 < len(rest) {
				i++
				return rest[i]
			}
			return val
		}
		if afterArgs { // the words after -args reach the test binary
			if name == "test.count" {
				if v := next(); v != "1" {
					return refuse("-%s=%s is refused: write -count=1", name, v)
				}
			}
			continue
		}
		switch name {
		case "args":
			afterArgs = true
		case "C":
			first := i == 0
			dir := next()
			if first {
				insert = i + 1
				a.listFlags = append([]string{"-C", dir}, a.listFlags...)
			}
		case "json":
			if hasVal && (val == "false" || val == "0" || val == "f" || val == "F" || val == "FALSE" || val == "False") {
				return refuse("-json=false is refused: the adapter reads the event stream")
			}
			hasJSON = true
		case "count", "test.count":
			if v := next(); v != "1" {
				return refuse("-%s=%s is refused: write -count=1", name, v)
			}
			hasCount = true
		case "tags":
			tagList = next()
			a.listFlags = append(a.listFlags, "-tags="+tagList)
		case "mod", "modfile", "overlay":
			a.listFlags = append(a.listFlags, "-"+name+"="+next())
		}
	}
	a.tags = tagsOf(tagList)
	a.Argv = append([]string{}, argv[:2+insert]...)
	if !hasJSON {
		a.Argv = append(a.Argv, "-json")
	}
	if !hasCount {
		a.Argv = append(a.Argv, "-count=1") // go test never answers from its cache
	}
	a.Argv = append(a.Argv, rest[insert:]...)
	return nil
}

// tagsOf splits the value of -tags; without one it reads GOFLAGS. Never nil.
func tagsOf(tagList string) []string {
	if tagList == "" {
		for _, w := range strings.Fields(os.Getenv("GOFLAGS")) {
			if name, val, _ := flagOf(w); name == "tags" {
				tagList = val
			}
		}
	}
	tags := strings.FieldsFunc(tagList, func(r rune) bool { return r == ',' || r == ' ' })
	if tags == nil {
		tags = []string{}
	}
	return tags
}

// flagOf splits a command-line word into a flag name and its value; name is
// "" for a word that is no flag.
func flagOf(word string) (name, val string, hasVal bool) {
	rest, ok := strings.CutPrefix(word, "--")
	if !ok {
		if rest, ok = strings.CutPrefix(word, "-"); !ok {
			return "", "", false
		}
	}
	name, val, hasVal = strings.Cut(rest, "=")
	return name, val, hasVal
}

// Name is the name the run header records.
func (a *GoTest) Name() string { return GoTestJSON }

// goEvent is one line of go test -json.
type goEvent struct {
	Action  string
	Package string
	Test    string
	Output  string
}

// Consume reads the event stream to its end. It prints the output text of
// every event, and a line that is no event as it is.
func (a *GoTest) Consume(stream io.Reader, out io.Writer) error {
	st := &goStream{pkgs: map[string]*goPackage{}}
	a.streams = append(a.streams, st)
	r := bufio.NewReader(stream)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			var ev goEvent
			if json.Unmarshal(line, &ev) != nil || ev.Action == "" {
				out.Write(line)
			} else {
				io.WriteString(out, ev.Output)
				st.read(ev)
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// StreamsEnv names the directory a wrapper leaves its streams in.
const StreamsEnv = "MACHINERY_GO_TEST_JSON"

// ReadStreams reads every .json and .jsonl file directly in dir as one
// stream, in the order of their names. A file that is not a well-formed
// stream is an error: nothing of the run counts.
func (a *GoTest) ReadStreams(dir string) error {
	entries, err := os.ReadDir(dir) // sorted by name
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if !e.Type().IsRegular() || !(strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".jsonl")) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		st := &goStream{name: name, pkgs: map[string]*goPackage{}}
		events := 0
		for i, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var ev goEvent
			if json.Unmarshal([]byte(line), &ev) != nil || ev.Action == "" {
				return fmt.Errorf("stream %s: line %d: not a go test -json event; %s", name, i+1, leftIncomplete)
			}
			events++
			st.read(ev)
		}
		if events == 0 {
			return fmt.Errorf("stream %s: no event; %s", name, leftIncomplete)
		}
		a.streams = append(a.streams, st)
	}
	return nil
}

const leftIncomplete = "the run is left incomplete: fix what writes the stream, delete the receipts file and run again"

func isVerdict(action string) bool { return action == Pass || action == Fail || action == Skip }

func (s *goStream) read(ev goEvent) {
	if ev.Package == "" {
		if ev.Test != "" && isVerdict(ev.Action) && !strings.Contains(ev.Test, "/") {
			s.nameless++
		}
		return
	}
	pkg := s.pkgs[ev.Package]
	if pkg == nil {
		pkg = &goPackage{verdicts: map[string][]string{}}
		s.pkgs[ev.Package] = pkg
	}
	verdict := isVerdict(ev.Action)
	switch {
	case ev.Test == "" && verdict:
		pkg.end = ev.Action
	case ev.Test == "" && ev.Action == "output":
		// go test marks a package it answered from its cache: "ok  \t<package>\t(cached)"
		if strings.HasPrefix(ev.Output, "ok") && contains(strings.Split(strings.TrimSpace(ev.Output), "\t"), "(cached)") {
			pkg.cached = true
		}
	case verdict && !strings.Contains(ev.Test, "/"): // a subtest has its parent's verdict
		pkg.verdicts[ev.Test] = append(pkg.verdicts[ev.Test], ev.Action)
	}
}

// States of a function and of a claim, the one that decides first.
const (
	stFail = iota
	stAmbiguous
	stSkip
	stNotRun
	stBlocked // passed in a cached package, or in one that did not end with pass
	stPass
)

// adapterReceipt is the receipt line the adapter writes.
type adapterReceipt struct {
	V           int    `json:"v"`
	Type        string `json:"type"`
	ID          string `json:"id"`
	Outcome     string `json:"outcome"`
	Test        string `json:"test"`
	Observed    string `json:"observed"`
	Source      string `json:"source"`
	Granularity string `json:"granularity"`
	Tags        any    `json:"tags"` // the build tags, or "unknown"
}

// Receipts joins the stream to the claims and returns the receipt lines, in
// the order they are written. Call it once, after Consume.
func (a *GoTest) Receipts() [][]byte {
	// the verdicts of each function, by the file that defines it
	state := map[string]map[string]int{} // file -> function -> state
	set := func(file, fn string, st int) {
		if state[file] == nil {
			state[file] = map[string]int{}
		}
		if cur, ok := state[file][fn]; !ok || st < cur {
			state[file][fn] = st
		}
	}
	var passed, failed, skipped, homeless, nameless int
	var ambiguous, refused, differ []string
	lost := map[string]bool{} // packages with a verdict that found no file
	names := map[string]bool{}
	for _, s := range a.streams {
		nameless += s.nameless
		for name := range s.pkgs {
			names[name] = true
		}
	}
	built, listErr := map[string][]string(nil), ""
	if !a.fromDir && len(names) > 0 {
		built, listErr = a.listBuilt(sortedKeys(names))
	}
	raw := map[string]map[string][2]string{} // file -> function -> its verdict and the package it came from
	for _, name := range sortedKeys(names) {
		// what each stream says of the package: one cached answer anywhere
		// refuses it in the whole run
		cached, notOK := false, false
		type seen struct {
			verdict, stream string
		}
		byFn := map[string][]seen{}
		for _, s := range a.streams {
			pkg := s.pkgs[name]
			if pkg == nil {
				continue
			}
			if pkg.cached {
				cached = true
			} else if pkg.end != Pass && len(pkg.verdicts) > 0 {
				notOK = true
			}
			for fn, vs := range pkg.verdicts {
				for _, v := range vs {
					byFn[fn] = append(byFn[fn], seen{v, s.name})
				}
			}
		}
		if cached {
			refused = append(refused, fmt.Sprintf("adapter: package %s: answered from the cache; nothing in it counts: run with -count=1", name))
		}
		if notOK {
			refused = append(refused, fmt.Sprintf("adapter: package %s: did not end with pass; no pass is written from it", name))
		}
		for _, fn := range sortedKeys(byFn) {
			got := byFn[fn]
			verdict, same := got[0].verdict, true
			var parts []string
			for _, g := range got {
				same = same && g.verdict == verdict
				if g.stream != "" {
					parts = append(parts, g.verdict+" in "+g.stream)
				} else {
					parts = append(parts, g.verdict)
				}
			}
			if !same && !cached { // verdicts that differ: a fail, whichever they are
				verdict = Fail
				differ = append(differ, fmt.Sprintf("adapter: %s.%s: %s: verdicts differ, read as fail", name, fn, strings.Join(parts, ", ")))
			}
			st := stPass
			switch {
			case cached:
				st = stBlocked
			case verdict == Fail:
				st = stFail
			case verdict == Skip:
				st = stSkip
			case notOK:
				st = stBlocked
			}
			switch verdict {
			case Pass:
				passed++
			case Fail:
				failed++
			default:
				skipped++
			}
			defs := a.definers(name, fn, built)
			switch {
			case len(defs) == 0 || (len(defs) == 1 && a.files[defs[0]] == nil):
				homeless++
				lost[name] = true
			case len(defs) == 1:
				file := defs[0]
				if raw[file] == nil {
					raw[file] = map[string][2]string{}
				}
				if prev, seen := raw[file][fn]; seen && !cached && prev[0] != verdict {
					// one file, several packages, verdicts that differ: a fail
					differ = append(differ, fmt.Sprintf("adapter: %s::%s: %s in package %s, %s in package %s: verdicts differ, read as fail", file, fn, prev[0], prev[1], verdict, name))
					st = stFail
				} else if !cached {
					raw[file][fn] = [2]string{verdict, name}
				}
				set(file, fn, st)
			default:
				ambiguous = append(ambiguous, fmt.Sprintf("adapter: %s.%s: defined in %s: ambiguous, its verdict counts for none of them", name, fn, strings.Join(defs, ", ")))
				for _, f := range defs {
					if a.files[f] != nil {
						set(f, fn, stAmbiguous)
					}
				}
			}
		}
	}

	// one receipt per attribution of a claimed id
	wrote := map[string]int{}
	// per id: a receipt was written, a claim met a function that did not
	// run, a claim met a verdict that does not count
	has, idle, held := map[string]bool{}, map[string]bool{}, map[string]bool{}
	var lines [][]byte
	for _, u := range a.units {
		f := a.files[u.File]
		funcs := f.Funcs
		if u.Func != "" {
			funcs = []string{u.Func}
		}
		if len(funcs) == 0 {
			continue // no test function, or a file that does not parse: named in the summary
		}
		st := stPass
		var failedFns, skippedFns []string
		for _, fn := range funcs {
			fst, ran := state[u.File][fn]
			if !ran {
				fst = stNotRun
			}
			if fst < st {
				st = fst
			}
			switch fst {
			case stFail:
				failedFns = append(failedFns, fn)
			case stSkip:
				skippedFns = append(skippedFns, fn)
			}
		}
		r := adapterReceipt{V: 1, Type: "receipt", ID: u.ID, Test: u.File, Source: SourceAdapter,
			Granularity: project.GranularityFile, Tags: "unknown"}
		if a.tags != nil {
			r.Tags = a.tags
		}
		if u.Func != "" {
			r.Test, r.Granularity = u.File+"::"+u.Func, project.GranularityFunction
		}
		switch st {
		case stPass:
			r.Outcome, r.Observed = Pass, fmt.Sprintf("go test: all %d test function(s) of the file passed", len(funcs))
			if u.Func != "" {
				r.Observed = "go test: " + u.Func + " passed"
			}
		case stFail:
			r.Outcome, r.Observed = Fail, "go test: "+nameList(failedFns)+" failed"
		case stSkip:
			r.Outcome, r.Observed = Skip, "go test: "+nameList(skippedFns)+" skipped"
		case stNotRun:
			idle[u.ID] = true
			continue
		default:
			held[u.ID] = true
			continue
		}
		wrote[r.Outcome]++
		has[u.ID] = true
		lines = append(lines, jsonLine(r))
	}
	var notRun, notCounted int
	for id := range idle {
		if !has[id] {
			notRun++
		}
	}
	for id := range held {
		if !has[id] && !idle[id] {
			notCounted++
		}
	}

	nFuncs := passed + failed + skipped
	a.summary = nil
	counted := ""
	if a.fromDir {
		counted = fmt.Sprintf("%d stream(s), ", len(a.streams))
		if len(a.streams) == 0 {
			a.summary = append(a.summary, fmt.Sprintf("adapter %s: no stream in %s: the command ran nothing the adapter can read, and nothing is proven", GoTestJSON, StreamsEnv))
		}
	}
	if !a.fromDir || len(a.streams) > 0 {
		a.summary = append(a.summary, fmt.Sprintf("adapter %s: %s%d package(s), %d test function(s) with a verdict: %d passed, %d failed, %d skipped", GoTestJSON, counted, len(names), nFuncs, passed, failed, skipped))
	}
	a.summary = append(a.summary,
		fmt.Sprintf("adapter: receipts written: %d pass, %d fail, %d skip", wrote[Pass], wrote[Fail], wrote[Skip]),
		fmt.Sprintf("adapter: claimed, not run: %d id(s)", notRun),
	)
	add := func(n int, format string) {
		if n > 0 {
			a.summary = append(a.summary, fmt.Sprintf(format, n))
		}
	}
	add(notCounted, "adapter: claimed, verdict not counted: %d id(s)")
	add(len(a.effects), "adapter: needs a receipt with effects: %d tc id(s)")
	add(len(a.replay), "adapter: needs a receipt with a replay: %d cx id(s)")
	if homeless > 0 {
		a.summary = append(a.summary, fmt.Sprintf("adapter: %d test function(s) with a verdict are in no file under trace.tests: package(s) %s; a package built from copies, and every package of a --streams run, is mapped to its files in evidence.adapter.sources", homeless, nameList(sortedKeys(lost))))
	}
	if listErr != "" {
		a.summary = append(a.summary, fmt.Sprintf("adapter: go list failed (%s): which files the packages built is not known, and only evidence.adapter.sources attributes a verdict", listErr))
	}
	add(nameless, "adapter: %d verdict(s) name no package and are used for nothing: run the test binary under go tool test2json -p <import path>")
	sort.Strings(differ)
	a.summary = append(append(append(a.summary, refused...), ambiguous...), differ...)
	claiming := map[string]bool{}
	for _, u := range a.units {
		claiming[u.File] = true
	}
	var empty, broken []string
	for _, rel := range sortedKeys(claiming) {
		switch f := a.files[rel]; {
		case f.Broken:
			broken = append(broken, fmt.Sprintf("adapter: %s: not Go source that parses; its claims count nothing", rel))
		case len(f.Funcs) == 0:
			empty = append(empty, fmt.Sprintf("adapter: %s: no test function; its claims count nothing", rel))
		}
	}
	a.summary = append(append(a.summary, empty...), broken...)
	return lines
}

// listBuilt asks go list which test files each package of the run compiled,
// under the flags and the environment of the run (section 10.3). The files
// are project-relative where they are inside the project. errLine is the
// first line go list printed when it could not answer.
func (a *GoTest) listBuilt(pkgs []string) (built map[string][]string, errLine string) {
	args := []string{"list"}
	if len(a.listFlags) > 0 && a.listFlags[0] == "-C" { // -C comes first on the command line
		args = append(append(args, a.listFlags[:2]...), "-e", "-json=ImportPath,Dir,TestGoFiles,XTestGoFiles")
		args = append(args, a.listFlags[2:]...)
	} else {
		args = append(append(args, "-e", "-json=ImportPath,Dir,TestGoFiles,XTestGoFiles"), a.listFlags...)
	}
	cmd := exec.Command(a.Argv[0], append(args, pkgs...)...)
	cmd.Dir = a.p.Root
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		line, _, _ := strings.Cut(strings.TrimSpace(errOut.String()), "\n")
		if line == "" {
			line = err.Error()
		}
		return nil, line
	}
	root := a.p.Root
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}
	built = map[string][]string{}
	dec := json.NewDecoder(&out)
	for {
		var pkg struct {
			ImportPath, Dir           string
			TestGoFiles, XTestGoFiles []string
		}
		if err := dec.Decode(&pkg); err != nil {
			break
		}
		if pkg.Dir == "" {
			continue // go list does not know the package
		}
		dir := pkg.Dir
		if r, err := filepath.EvalSymlinks(dir); err == nil {
			dir = r
		}
		files := []string{}
		for _, name := range append(append([]string{}, pkg.TestGoFiles...), pkg.XTestGoFiles...) {
			file := filepath.Join(dir, name)
			if rel, err := filepath.Rel(root, file); err == nil && !strings.HasPrefix(rel, "..") {
				file = filepath.ToSlash(rel)
			}
			files = append(files, file)
		}
		built[pkg.ImportPath] = files
	}
	return built, ""
}

// parsed returns a candidate file, read once: a file under trace.tests, or
// any other by its project-relative or absolute path.
func (a *GoTest) parsed(file string) *goTestFile {
	if f := a.files[file]; f != nil {
		return f
	}
	if f := a.other[file]; f != nil {
		return f
	}
	abs := file
	if !filepath.IsAbs(file) {
		abs = a.p.Abs(file)
	}
	f := &goTestFile{Path: file, Broken: true}
	if data, err := os.ReadFile(abs); err == nil {
		f = parseGoTest(file, data)
	}
	a.other[file] = f
	return f
}

// definers returns the files that define the function of a verdict, sorted
// (section 10.3, "Whose verdict"). One file that is under trace.tests has
// the verdict; several make the name ambiguous; none, or one outside
// trace.tests, leaves it with no file.
//
// The candidates are the files the package built, when go list gave them.
// When those give the verdict to no file under trace.tests, they are the
// files the source map entries of the package name: every _test.go file
// their globs match in the project, claiming or not.
func (a *GoTest) definers(pkg, fn string, built map[string][]string) []string {
	among := func(cands []string) []string {
		seen := map[string]bool{}
		var res []string
		for _, file := range cands {
			if !seen[file] && a.parsed(file).defines(fn) {
				seen[file] = true
				res = append(res, file)
			}
		}
		sort.Strings(res)
		return res
	}
	if files, known := built[pkg]; known {
		if defs := among(files); len(defs) > 1 || (len(defs) == 1 && a.files[defs[0]] != nil) {
			return defs
		}
	}
	var mapped []string
	for _, s := range a.sources {
		if !project.Match([]string{s.Package}, pkg) {
			continue
		}
		all, err := a.p.Glob(s.Files)
		if err != nil {
			continue
		}
		for _, file := range all {
			if strings.HasSuffix(file, "_test.go") {
				mapped = append(mapped, file)
			}
		}
	}
	return among(mapped)
}

// Summary returns the lines of the run summary, after Receipts.
func (a *GoTest) Summary() []string { return a.summary }

// nameList joins sorted names: the first five, then how many more.
func nameList(names []string) string {
	sort.Strings(names)
	if len(names) <= 5 {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(names[:5], ", "), len(names)-5)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
