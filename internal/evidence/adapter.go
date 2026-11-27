package evidence

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
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
	// tags are the tags of -tags; nil when they are not known
	tags    []string
	files   map[string]*goTestFile  // the test files under trace.tests
	other   map[string]*goTestFile  // other Go files parsed, by project-relative path
	sources []project.AdapterSource // read with streams only
	// pkgFiles: the import path of a directory that holds test files under
	// trace.tests -> those files (direct mode, section 10.3)
	pkgFiles map[string][]string
	pkgDir   map[string]string   // that import path -> its directory
	dirGo    map[string][]string // directory -> its Go files, read once
	units    []claimUnit
	effects  map[string]bool // claimed tc ids
	replay   map[string]bool // claimed cx ids

	// streams are the event streams read: the command's standard output, or
	// with fromDir the files the command left in MACHINERY_GO_TEST_JSON
	streams  []*goStream
	fromDir  bool
	summary  []string
	receipts [][]byte

	// env is the go environment of a direct-mode run, recorded in the
	// header; gomod is the go.mod go env names
	env   *GoEnv
	gomod string
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
	a := &GoTest{p: p, other: map[string]*goTestFile{}, effects: map[string]bool{}, replay: map[string]bool{}, fromDir: streams,
		pkgFiles: map[string][]string{}, pkgDir: map[string]string{}, dirGo: map[string][]string{}}
	if streams {
		// the command is not read and its build tags are not known; GOFLAGS of
		// the environment is all machinery sees of what the wrapper's go test gets
		for _, w := range strings.Fields(os.Getenv("GOFLAGS")) {
			switch name, val, _ := flagOf(w); {
			case (name == "count" || name == "test.count") && val != "1":
				return nil, refuse("GOFLAGS has -%s=%s, which is refused: take it out of GOFLAGS", name, val)
			case name == "overlay":
				return nil, refuse("GOFLAGS has -overlay, which is refused: take it out of GOFLAGS")
			}
		}
		a.Argv = argv
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
	for _, rel := range goFiles {
		dir := path.Dir(rel)
		if ip, ok := importPath(p.Abs(dir)); ok {
			a.pkgFiles[ip] = append(a.pkgFiles[ip], rel)
			if _, seen := a.pkgDir[ip]; !seen {
				a.pkgDir[ip] = dir
			}
		}
	}
	entries := p.Cfg.AdapterSources()
	for i, s := range entries {
		if streams && len(matching(goFiles, s.Files)) == 0 {
			return nil, refuse("evidence.adapter.sources: entry %d (package %s): files %s match no file under trace.tests: correct the globs, or list the files in trace.tests", i+1, s.Package, strings.Join(s.Files, ", "))
		}
		if streams {
			continue
		}
		for _, ip := range sortedKeys(a.pkgDir) { // not read without streams: an entry may not seem to narrow a package
			if project.Match([]string{s.Package}, ip) {
				return nil, refuse("evidence.adapter.sources: entry %d (package %s) matches %s, the package of %s: entries are read only with --streams; remove the entry, or run the suite through --streams", i+1, s.Package, ip, a.pkgDir[ip])
			}
		}
	}
	if streams {
		a.sources = entries
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
	if !streams {
		if err := a.checkLayout(ClaimingGoTests(d, found)); err != nil {
			return nil, err
		}
	}
	return a, nil
}

// checkLayout holds the go.mod files of the project and the workspace in
// effect to a layout direct mode attributes by import path (section 10.3),
// and records the digests of go.mod and go.work.
func (a *GoTest) checkLayout(claiming []string) error {
	dups, err := duplicateModules(a.p)
	if err != nil {
		return err
	}
	if len(dups) > 0 {
		return refuse("%s: give each module its own path, or run the command through --streams", dups[0])
	}
	if work := a.env.GOWORK; work != "" && work != "off" {
		if unused := unusedModules(a.p, claiming, work); len(unused) > 0 {
			return refuse("%s: add the module to the workspace, or run the command with GOWORK=off", unused[0])
		}
		a.env.WorkSum = fileSum(work)
	}
	a.env.GoModSum = fileSum(a.gomod)
	return nil
}

// Environment is what the run header records of the go environment
// (section 10.3); nil with --streams, where machinery does not run go.
func (a *GoTest) Environment() any {
	if a.env == nil {
		return nil
	}
	return a.env
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

// The words direct mode accepts after go test (section 10.3): the flags that
// take a value, and the flags that are switches.
var (
	valueFlags = wordSet("run", "skip", "tags", "count", "timeout", "p", "parallel", "shuffle", "cpu",
		"coverprofile", "covermode", "coverpkg", "test.run", "test.skip", "test.count", "test.timeout",
		"test.parallel", "test.shuffle", "test.cpu", "test.coverprofile")
	switchFlags = wordSet("race", "v", "json", "short", "failfast", "cover", "test.v", "test.short", "test.failfast")
)

func wordSet(words ...string) map[string]bool {
	m := map[string]bool{}
	for _, w := range words {
		m[w] = true
	}
	return m
}

func notAccepted(word string) error {
	return refuse("%s: not accepted by the adapter in direct mode; run the command through --streams", word)
}

func isFalse(v string) bool {
	return v == "false" || v == "0" || v == "f" || v == "F" || v == "FALSE" || v == "False"
}

// relativePattern reports a package pattern relative to the project root
// that does not leave it: ".", or "./..." with no ".." element.
func relativePattern(word string) bool {
	if word == "." {
		return true
	}
	rest, ok := strings.CutPrefix(word, "./")
	if !ok {
		return false
	}
	for _, seg := range strings.Split(rest, "/") {
		if seg == ".." {
			return false
		}
	}
	return true
}

// readCommand holds the command to the narrow go test of direct mode (section
// 10.3), adds -json and -count=1, and reads the go environment of the run.
func (a *GoTest) readCommand(argv []string) error {
	if len(argv) < 2 || filepath.Base(argv[0]) != "go" || argv[1] != "test" {
		n := len(argv)
		if n > 2 {
			n = 2
		}
		return refuse("the command must be go test, got %s: run go test itself, or add --streams and have the command leave its go test -json output in %s", strings.Join(argv[:n], " "), StreamsEnv)
	}
	goBin := a.goBinary(argv[0])
	if rel, in := inside(a.p.Root, goBin); in {
		return refuse("%s: a go inside the project is not accepted by the adapter in direct mode; run the command through --streams", rel)
	}
	rest := argv[2:]
	hasJSON, hasCount := false, false
	var tagList string
	for i := 0; i < len(rest); i++ {
		word := rest[i]
		name, val, hasVal := flagOf(word)
		switch {
		case name == "args" && i == len(rest)-1:
		case name == "args":
			return notAccepted(rest[i+1]) // nothing may follow -args
		case name == "":
			if !relativePattern(word) {
				return notAccepted(word)
			}
		case valueFlags[name]:
			if !hasVal && i+1 < len(rest) { // the value written as a second word
				i++
				val = rest[i]
			}
			switch name {
			case "count", "test.count":
				if val != "1" {
					return refuse("-%s=%s is refused: write -count=1", name, val)
				}
				hasCount = true
			case "tags":
				tagList = val
			}
		case switchFlags[name]:
			if name == "json" && hasVal && isFalse(val) {
				return refuse("-json=false is refused: the adapter reads the event stream; leave it out")
			}
			hasJSON = hasJSON || name == "json"
		default:
			return notAccepted(word)
		}
	}
	goflags, err := a.checkGoEnv(goBin)
	if err != nil {
		return err
	}
	a.tags = tagsOf(tagList, goflags)
	a.Argv = append([]string{}, argv[:2]...)
	if !hasJSON {
		a.Argv = append(a.Argv, "-json")
	}
	if !hasCount {
		a.Argv = append(a.Argv, "-count=1") // go test never answers from its cache
	}
	a.Argv = append(a.Argv, rest...)
	return nil
}

// goBinary returns the go of the command as a path: found on PATH for a bare
// name, from the project root for a relative path. Links are not followed.
func (a *GoTest) goBinary(argv0 string) string {
	bin := argv0
	if !strings.Contains(argv0, "/") {
		found, err := exec.LookPath(argv0)
		switch {
		case err == nil:
			bin = found
		case errors.Is(err, exec.ErrDot):
			bin, _ = filepath.Abs(found)
		default:
			return argv0 // not found: go env says so
		}
	} else if !filepath.IsAbs(bin) {
		bin = filepath.Join(a.p.Root, bin)
	}
	return bin
}

// checkGoEnv asks go env, in the project root and with the environment of
// the run, for the go environment, holds it to direct mode and keeps it for
// the header (section 10.3). It returns GOFLAGS.
func (a *GoTest) checkGoEnv(goBin string) (string, error) {
	cmd := exec.Command(goBin, "env", "-json", "GOFLAGS", "GOMOD", "GOWORK", "GOOS", "GOARCH", "GOHOSTOS", "GOHOSTARCH", "GOVERSION")
	cmd.Dir = a.p.Root
	var out, errOut strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errOut
	var env struct{ GOFLAGS, GOMOD, GOWORK, GOOS, GOARCH, GOHOSTOS, GOHOSTARCH, GOVERSION string }
	runErr := cmd.Run()
	if runErr == nil {
		runErr = json.Unmarshal([]byte(out.String()), &env)
	}
	if runErr != nil {
		line, _, _ := strings.Cut(strings.TrimSpace(errOut.String()), "\n")
		if line == "" {
			line = runErr.Error()
		}
		return "", refuse("go env failed (%s): the adapter reads GOFLAGS, GOMOD and GOWORK from it; fix the environment of go, or run the command through --streams", line)
	}
	for _, w := range strings.Fields(env.GOFLAGS) {
		name, val, _ := flagOf(w)
		switch {
		case (name == "count" || name == "test.count") && val != "1":
			return "", refuse("GOFLAGS has -%s=%s, which is refused: take it out of GOFLAGS", name, val)
		case !valueFlags[name] && !switchFlags[name]: // -json=false is harmless: the command has -json
			return "", refuse("%s in GOFLAGS: not accepted by the adapter in direct mode; run the command through --streams", w)
		}
	}
	if rel, in := inside(a.p.Root, env.GOMOD); !fileExists(env.GOMOD) || !in || rel != "go.mod" {
		return "", refuse("go env GOMOD is %q, not the go.mod of the project root: run the command through --streams", env.GOMOD)
	}
	if env.GOOS != env.GOHOSTOS || env.GOARCH != env.GOHOSTARCH {
		return "", refuse("go env GOOS/GOARCH is %s/%s, not the host's %s/%s: go would run the test binary through go_%s_%s_exec; unset GOOS and GOARCH, or run the command through --streams",
			env.GOOS, env.GOARCH, env.GOHOSTOS, env.GOHOSTARCH, env.GOOS, env.GOARCH)
	}
	a.env = &GoEnv{Version: env.GOVERSION, Binary: goBin, GOFLAGS: env.GOFLAGS, GOWORK: env.GOWORK, GOOS: env.GOOS, GOARCH: env.GOARCH}
	a.gomod = env.GOMOD
	return env.GOFLAGS, nil
}

// tagsOf splits the value of -tags; without one it reads GOFLAGS. Never nil.
func tagsOf(tagList, goflags string) []string {
	if tagList == "" {
		for _, w := range strings.Fields(goflags) {
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
	stBlocked // a pass replayed from the cache, or one of a package that did not end with pass
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
	// the state of each function, by the file its verdict belongs to
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
	raw := map[string]map[string][2]string{} // file -> function -> its verdict and the package it came from
	for _, name := range sortedKeys(names) {
		// what each stream says of the package. A pass replayed from the
		// cache is not read; every other verdict of every stream is
		cached, notOK := false, false
		type seen struct {
			verdict, stream string
		}
		read := map[string][]seen{}
		replayed := map[string]bool{}
		for _, s := range a.streams {
			pkg := s.pkgs[name]
			if pkg == nil {
				continue
			}
			cached = cached || pkg.cached
			if pkg.end != Pass && len(pkg.verdicts) > 0 {
				notOK = true
			}
			for fn, vs := range pkg.verdicts {
				for _, v := range vs {
					if pkg.cached && v == Pass {
						replayed[fn] = true
						continue
					}
					read[fn] = append(read[fn], seen{v, s.name})
				}
			}
		}
		if cached {
			refused = append(refused, fmt.Sprintf("adapter: package %s: answered from the cache; a pass replayed from the cache counts nothing: run go test with -count=1", name))
		}
		if notOK {
			refused = append(refused, fmt.Sprintf("adapter: package %s: did not end with pass; no pass is written from it", name))
		}
		fns := sortedKeys(read)
		for fn := range replayed {
			if read[fn] == nil {
				fns = append(fns, fn)
			}
		}
		sort.Strings(fns)
		for _, fn := range fns {
			verdict := "" // only a pass replayed from the cache
			if got := read[fn]; len(got) > 0 {
				verdict = got[0].verdict
				same := true
				var parts []string
				for _, g := range got {
					same = same && g.verdict == verdict
					if g.stream != "" {
						parts = append(parts, g.verdict+" in "+g.stream)
					} else {
						parts = append(parts, g.verdict)
					}
				}
				if !same { // verdicts that differ: a fail, whichever they are
					verdict = Fail
					differ = append(differ, fmt.Sprintf("adapter: %s.%s: %s: verdicts differ, read as fail", name, fn, strings.Join(parts, ", ")))
				}
			}
			st := stBlocked
			switch verdict {
			case Fail:
				st = stFail
				failed++
			case Skip:
				st = stSkip
				skipped++
			default:
				if verdict == Pass && !notOK {
					st = stPass
				}
				passed++
			}
			defs, cands := a.definers(name, fn)
			switch {
			case len(defs) > 1 || (len(defs) == 1 && a.parsed(defs[0]).Broken):
				// another file could have given the verdict: a pass or a
				// skip counts for none of them, a fail fails each
				var named []string
				for _, f := range defs {
					if a.parsed(f).Broken {
						f += " (does not parse)"
					}
					named = append(named, f)
				}
				ambiguous = append(ambiguous, fmt.Sprintf("adapter: %s.%s: defined in %s: ambiguous: a pass or a skip counts for none of them, a fail fails each; rename one of the functions", name, fn, strings.Join(named, ", ")))
				for _, f := range defs {
					if a.files[f] == nil {
						continue
					}
					if st == stFail {
						set(f, fn, stFail)
					} else {
						set(f, fn, stAmbiguous)
					}
				}
			case len(defs) == 1 && cands[defs[0]] && a.files[defs[0]] != nil:
				file := defs[0]
				if raw[file] == nil {
					raw[file] = map[string][2]string{}
				}
				if prev, seen := raw[file][fn]; seen && verdict != "" && prev[0] != verdict {
					// one file, several packages, verdicts that differ: a fail
					differ = append(differ, fmt.Sprintf("adapter: %s::%s: %s in package %s, %s in package %s: verdicts differ, read as fail", file, fn, prev[0], prev[1], verdict, name))
					st = stFail
				} else if verdict != "" {
					raw[file][fn] = [2]string{verdict, name}
				}
				set(file, fn, st)
			default:
				homeless++
				lost[name] = true
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
		todo := "a package's verdicts go to the files under trace.tests in its own directory, and a package built from copies is run through --streams with an entry in evidence.adapter.sources"
		if a.fromDir {
			todo = "with --streams a package's verdicts go only to the files of its entry in evidence.adapter.sources: add one"
		}
		a.summary = append(a.summary, fmt.Sprintf("adapter: %d test function(s) with a verdict are in no file under trace.tests: package(s) %s; %s", homeless, nameList(sortedKeys(lost)), todo))
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

// parsed returns a Go file of the project, read once: a file under
// trace.tests, or any other by its project-relative path.
func (a *GoTest) parsed(file string) *goTestFile {
	if f := a.files[file]; f != nil {
		return f
	}
	if f := a.other[file]; f != nil {
		return f
	}
	f := &goTestFile{Path: file, Broken: true}
	if data, err := a.p.Read(file); err == nil {
		f = parseGoTest(file, data)
	}
	a.other[file] = f
	return f
}

// candidates returns the files a verdict of the package may belong to
// (section 10.3, "Whose verdict"): without streams the test files under
// trace.tests in the package's own directory, with streams the test files
// the source map entries of the package name.
func (a *GoTest) candidates(pkg string) []string {
	if !a.fromDir {
		return a.pkgFiles[pkg]
	}
	var res []string
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
				res = append(res, file)
			}
		}
	}
	return res
}

// definers returns the files that could have given the verdict of a
// function of the package, sorted, and the candidates of the package. Those
// files are the candidates and every Go file of a directory that holds one,
// whatever its build constraints; one that does not parse may define any
// name.
func (a *GoTest) definers(pkg, fn string) (defs []string, cands map[string]bool) {
	cands = map[string]bool{}
	dirs := map[string]bool{}
	for _, file := range a.candidates(pkg) {
		cands[file] = true
		dirs[path.Dir(file)] = true
	}
	seen := map[string]bool{}
	consider := func(file string) {
		if seen[file] {
			return
		}
		seen[file] = true
		if f := a.parsed(file); f.Broken || f.defines(fn) {
			defs = append(defs, file)
		}
	}
	for file := range cands {
		consider(file)
	}
	for dir := range dirs {
		for _, file := range a.goFiles(dir) {
			consider(file)
		}
	}
	sort.Strings(defs)
	return defs, cands
}

// goFiles returns the Go files directly in a project directory, read once.
func (a *GoTest) goFiles(dir string) []string {
	if files, ok := a.dirGo[dir]; ok {
		return files
	}
	files := []string{}
	entries, _ := os.ReadDir(a.p.Abs(dir))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
			files = append(files, path.Join(dir, e.Name()))
		}
	}
	a.dirGo[dir] = files
	return files
}

// importPath returns the import path of a directory: the module path of the
// nearest go.mod at or above it, then the path of the directory below it.
func importPath(abs string) (string, bool) {
	file, ok := moduleFile(abs)
	if !ok {
		return "", false
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return "", false
	}
	mod := modulePath(data)
	rel, err := filepath.Rel(filepath.Dir(file), abs)
	switch {
	case mod == "" || err != nil:
		return "", false
	case rel == ".":
		return mod, true
	}
	return mod + "/" + filepath.ToSlash(rel), true
}

// modulePath reads the module line of a go.mod file.
func modulePath(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module")
		if !ok || rest == "" || !strings.ContainsAny(rest[:1], " \t\"") {
			continue
		}
		if i := strings.Index(rest, "//"); i >= 0 {
			rest = rest[:i]
		}
		return strings.Trim(strings.TrimSpace(rest), "\"")
	}
	return ""
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
