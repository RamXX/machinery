package evidence

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
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

	tags    []string
	files   map[string]*goTestFile
	direct  map[string][]string // import path -> the test files of that directory
	sources []project.AdapterSource
	units   []claimUnit
	effects map[string]bool // claimed tc ids
	replay  map[string]bool // claimed cx ids

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
	a := &GoTest{direct: map[string][]string{}, effects: map[string]bool{}, replay: map[string]bool{}, fromDir: streams}
	if streams {
		a.Argv, a.tags = argv, tagsOf("")
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
	for rel, f := range a.files {
		goFiles = append(goFiles, rel)
		if f.Pkg != "" {
			a.direct[f.Pkg] = append(a.direct[f.Pkg], rel)
		}
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
	insert, hasJSON, hasCount := 0, false, false
	var tagList string
	for i := 0; i < len(rest); i++ {
		name, val, hasVal := flagOf(rest[i])
		if name == "args" {
			break
		}
		next := func() string { // the value of a flag written as two words
			if !hasVal && i+1 < len(rest) {
				i++
				return rest[i]
			}
			return val
		}
		switch name {
		case "C":
			first := i == 0
			next()
			if first {
				insert = i + 1
			}
		case "json":
			if hasVal && (val == "false" || val == "0" || val == "f" || val == "F" || val == "FALSE" || val == "False") {
				return refuse("-json=false is refused: the adapter reads the event stream")
			}
			hasJSON = true
		case "count":
			if v := next(); v != "1" {
				return refuse("-count=%s is refused: write -count=1", v)
			}
			hasCount = true
		case "tags":
			tagList = next()
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
	V           int      `json:"v"`
	Type        string   `json:"type"`
	ID          string   `json:"id"`
	Outcome     string   `json:"outcome"`
	Test        string   `json:"test"`
	Observed    string   `json:"observed"`
	Source      string   `json:"source"`
	Granularity string   `json:"granularity"`
	Tags        []string `json:"tags"`
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
	names := map[string]bool{}
	for _, s := range a.streams {
		nameless += s.nameless
		for name := range s.pkgs {
			names[name] = true
		}
	}
	for _, name := range sortedKeys(names) {
		// what each stream says of the package; a cached one is not read
		cached, notOK := false, false
		type seen struct {
			verdict, stream string
			ok              bool
		}
		byFn := map[string][]seen{}
		cachedFns := map[string]string{}
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
				v := Pass
				for _, one := range vs {
					if one == Fail || (one == Skip && v != Fail) {
						v = one
					}
				}
				if pkg.cached {
					cachedFns[fn] = v
					continue
				}
				byFn[fn] = append(byFn[fn], seen{v, s.name, pkg.end == Pass})
			}
		}
		if cached {
			refused = append(refused, fmt.Sprintf("adapter: package %s: answered from the cache; nothing in it counts: run with -count=1", name))
		}
		if notOK {
			refused = append(refused, fmt.Sprintf("adapter: package %s: did not end with pass; no pass is written from it", name))
		}
		fns := map[string]bool{}
		for fn := range byFn {
			fns[fn] = true
		}
		for fn := range cachedFns {
			fns[fn] = true
		}
		for _, fn := range sortedKeys(fns) {
			verdict, st := cachedFns[fn], stBlocked // seen in a cached package alone
			if got := byFn[fn]; len(got) > 0 {
				verdict = got[0].verdict
				allOK := true
				var parts []string
				same := true
				for _, g := range got {
					same = same && g.verdict == verdict
					allOK = allOK && g.ok
					parts = append(parts, g.verdict+" in "+g.stream)
				}
				if !same { // one function, several streams, verdicts that differ: a fail
					verdict = Fail
					differ = append(differ, fmt.Sprintf("adapter: %s.%s: %s: verdicts differ, read as fail", name, fn, strings.Join(parts, ", ")))
				}
				switch {
				case verdict == Fail:
					st = stFail
				case verdict == Skip:
					st = stSkip
				case allOK:
					st = stPass
				}
			}
			switch verdict {
			case Pass:
				passed++
			case Fail:
				failed++
			default:
				skipped++
			}
			defs := a.definers(name, fn)
			switch len(defs) {
			case 0:
				homeless++
			case 1:
				set(defs[0], fn, st)
			default:
				ambiguous = append(ambiguous, fmt.Sprintf("adapter: %s.%s: defined in %s: ambiguous, its verdict counts for none of them", name, fn, strings.Join(defs, ", ")))
				for _, f := range defs {
					set(f, fn, stAmbiguous)
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
			Granularity: project.GranularityFile, Tags: a.tags}
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
	add(homeless, "adapter: %d test function(s) with a verdict are in no file under trace.tests")
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

// definers returns the test files a verdict of the package may belong to
// that define the function: the files of the package's own directory, or,
// for a package no test file has, the files its source map entries name.
func (a *GoTest) definers(pkg, fn string) []string {
	cands := a.direct[pkg]
	if len(cands) == 0 {
		var all []string
		for rel := range a.files {
			all = append(all, rel)
		}
		for _, s := range a.sources {
			if project.Match([]string{s.Package}, pkg) {
				cands = append(cands, matching(all, s.Files)...)
			}
		}
	}
	seen := map[string]bool{}
	var res []string
	for _, rel := range cands {
		if !seen[rel] && a.files[rel].defines(fn) {
			seen[rel] = true
			res = append(res, rel)
		}
	}
	sort.Strings(res)
	return res
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
