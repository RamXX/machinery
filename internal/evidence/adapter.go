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
	dirIP    map[string]string   // directory -> its import path, "" for none; read once
	cands    map[string][]string // package -> its candidates with streams, read once
	all      []string            // every file of the project, read once
	units    []claimUnit
	effects  map[string]bool // claimed tc ids
	replay   map[string]bool // claimed cx ids
	// unjudged holds claims above functions the adapter does not judge.
	unjudged map[claimUnit]bool
	// claimants are those of every id under trace.tests, for the rule of
	// every claimant (section 10.3)
	claimants *claimants
	// AllowNoStreams: with streams, a run that reads no stream is no failure
	AllowNoStreams bool

	// streams are the event streams read: the command's standard output, or
	// with fromDir the files the command left in MACHINERY_GO_TEST_JSON
	streams  []*goStream
	fromDir  bool
	summary  []string
	receipts [][]byte
	// kept: the streams copied for --keep-streams, and where (as the header
	// names it)
	kept   int
	keptAt string
	// sets are the helpers judged by their set (section 10.3), sorted
	sets []string
	// wrote counts the receipts written; whole is the index of the
	// whole-project line in summary, idleAll the ids it counts as not run
	wrote    int
	whole    int
	idleAll  []string
	emptyAll int

	// env is the go environment of a direct-mode run, recorded in the
	// header; gomod is the go.mod go env names
	env   *GoEnv
	gomod string
}

// goStream is one stream of go test -json.
type goStream struct {
	name string // the file, in streams mode
	// pkgs: the runs of each package, in the order of the stream. A
	// package-level end event ends a run (section 10.3)
	pkgs     map[string][]*goPackage
	nameless int // verdicts of test functions that name no package
	// unnamed: the events that name no package, read as one whole: never
	// split into runs (section 10.3, "A fail goes by name")
	unnamed *goPackage
	// restarted: the first package with a start event before the end of
	// its current run; the run is left incomplete (section 10.3)
	restarted string
}

// goPackage is what a stream said about one run of a package.
type goPackage struct {
	events   int                 // the events read into the run
	end      string              // the package-level pass, fail or skip
	cached   bool                // answered from the test cache
	verdicts map[string][]string // test function -> its pass, fail and skip events
	started  map[string]int      // test function -> its run events, one per iteration
	subs     map[string]string   // test function -> the worst verdict of its subtests, fail or skip
	// subNames: test function -> the subtests with a verdict, at any depth
	subNames map[string]map[string]bool
}

// NewGoTest reads the command and the claims of the test files, and refuses
// what the adapter cannot join (section 10.3).
//
// With streams the command is any command: it is not read, and the streams
// are the files it leaves in MACHINERY_GO_TEST_JSON.
func NewGoTest(p *project.Project, d *design.Design, argv []string, streams bool) (*GoTest, error) {
	a := &GoTest{p: p, other: map[string]*goTestFile{}, effects: map[string]bool{}, replay: map[string]bool{}, unjudged: map[claimUnit]bool{}, fromDir: streams,
		pkgFiles: map[string][]string{}, pkgDir: map[string]string{}, dirGo: map[string][]string{}, dirIP: map[string]string{}, cands: map[string][]string{}}
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
	entries := p.Cfg.AdapterSources()
	// with streams a Go file an entry names under trace.tests is a test
	// source whatever its name: one the project stages before go test
	isStaged := stagedSource(p)
	staged := func(rel string) bool { return streams && isStaged(rel) }
	if a.files, err = readGoTests(p, files, staged); err != nil {
		return nil, err
	}
	var goFiles []string
	for rel := range a.files {
		goFiles = append(goFiles, rel)
	}
	sort.Strings(goFiles)
	for _, rel := range goFiles {
		if !strings.HasSuffix(rel, "_test.go") {
			continue // a staged source is no test file of its own directory
		}
		dir := path.Dir(rel)
		if ip, ok := a.importPathOf(dir); ok {
			a.pkgFiles[ip] = append(a.pkgFiles[ip], rel)
			if _, seen := a.pkgDir[ip]; !seen {
				a.pkgDir[ip] = dir
			}
		}
	}
	for i, s := range entries {
		switch {
		case streams && s.Dir:
			found := false
			for ip := range a.pkgDir {
				found = found || s.Matches(ip)
			}
			if !found {
				return nil, refuse("evidence.adapter.sources: entry %d (package %s): the pattern matches the import path of no directory that holds a _test.go file under trace.tests: correct the pattern, or list the files in trace.tests", i+1, s.Name())
			}
		case streams:
			if len(matching(goFiles, s.Files)) == 0 {
				return nil, refuse("evidence.adapter.sources: entry %d (package %s): files %s match no Go file under trace.tests (a file whose name ends in .go): correct the globs, or list the files in trace.tests", i+1, s.Name(), strings.Join(s.Files, ", "))
			}
		default:
			for _, ip := range sortedKeys(a.pkgDir) { // not read without streams: an entry may not seem to narrow a package
				if s.Matches(ip) {
					return nil, refuse("evidence.adapter.sources: entry %d (package %s) matches %s, the package of %s: entries are read only with --streams; remove the entry, or run the suite through --streams", i+1, s.Name(), ip, a.pkgDir[ip])
				}
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
	a.units, a.unjudged, a.effects, a.replay = readClaims(d, a.files, found)
	if a.claimants, err = readClaimants(p, d); err != nil {
		return nil, err
	}
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
		return refuse("%s", dups[0])
	}
	if work := a.env.GOWORK; work != "" && work != "off" {
		unused, err := unusedModules(a.p, claiming, work)
		if err != nil {
			return err
		}
		if len(unused) > 0 {
			return refuse("%s", unused[0])
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

// subtestFilter reports a -run or -skip value that selects subtests: a parent
// whose subtests it leaves out passes with nothing checked (section 10.3).
func subtestFilter(name, val string) bool {
	switch name {
	case "run", "skip", "test.run", "test.skip":
		return strings.Contains(val, "/")
	}
	return false
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
			if subtestFilter(name, val) {
				return notAccepted("-" + name + "=" + val)
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
		case !valueFlags[name] && !switchFlags[name], subtestFilter(name, val): // -json=false is harmless: the command has -json
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
	st := &goStream{pkgs: map[string][]*goPackage{}}
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
			if what := st.unended(); what != "" {
				return &Incomplete{What: "go test -json: " + what, Todo: "find what stopped go test"}
			}
			return nil
		}
		if err != nil {
			return &Incomplete{What: fmt.Sprintf("reading the event stream: %v", err), Todo: fixStream}
		}
	}
}

// unended says of the first package, by name, whose last run in the stream
// has no package-level end event what is missing, or returns "".
func (s *goStream) unended() string {
	for _, name := range sortedKeys(s.pkgs) {
		switch runs := s.pkgs[name]; {
		case runs[len(runs)-1].end != "":
		case len(runs) == 1:
			return "package " + name + " has no end event"
		default:
			return "package " + name + " has events after its last end event"
		}
	}
	return ""
}

// Incomplete is a stream that leaves the run incomplete (section 10.3): no
// receipt and no end line are written. Run sets File, the receipts file the
// message tells to delete.
type Incomplete struct {
	What, Todo string
	File       string // project-relative
}

func (e *Incomplete) Error() string {
	return fmt.Sprintf("%s; the run is left incomplete: %s, delete %s and run again", e.What, e.Todo, e.File)
}

const (
	fixStream     = "fix what writes the stream"
	restartedText = " starts again before its run ended"
)

// StreamsEnv names the directory a wrapper leaves its streams in.
const StreamsEnv = "MACHINERY_GO_TEST_JSON"

// ReadStreams reads every .json and .jsonl file directly in dir as one
// stream, in the order of their names. A file that is not a well-formed
// stream is an error: nothing of the run counts. With keep, every such file
// is first copied into that directory (--keep-streams), named by label.
func (a *GoTest) ReadStreams(dir, keep, label string) error {
	entries, err := os.ReadDir(dir) // sorted by name
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if name := e.Name(); e.Type().IsRegular() && (strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".jsonl")) {
			names = append(names, name)
		}
	}
	if keep != "" {
		for _, name := range names {
			if err := copyFile(filepath.Join(dir, name), filepath.Join(keep, name)); err != nil {
				return fmt.Errorf("--keep-streams: %v", err)
			}
		}
		a.kept, a.keptAt = len(names), label
	}
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		st := &goStream{name: name, pkgs: map[string][]*goPackage{}}
		events := 0
		for i, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var ev goEvent
			if json.Unmarshal([]byte(line), &ev) != nil || ev.Action == "" {
				return &Incomplete{What: fmt.Sprintf("stream %s: line %d: not a go test -json event", name, i+1), Todo: fixStream}
			}
			events++
			if st.read(ev); st.restarted != "" {
				return &Incomplete{What: fmt.Sprintf("stream %s: line %d: package %s%s", name, i+1, st.restarted, restartedText),
					Todo: "find what stopped go test or give each go test run a file of its own"}
			}
		}
		if events == 0 {
			return &Incomplete{What: "stream " + name + ": no event", Todo: fixStream}
		}
		if what := st.unended(); what != "" {
			return &Incomplete{What: "stream " + name + ": " + what, Todo: fixStream}
		}
		a.streams = append(a.streams, st)
	}
	return nil
}

// copyFile copies a file whole.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func isVerdict(action string) bool { return action == Pass || action == Fail || action == Skip }

func newRun() *goPackage {
	return &goPackage{verdicts: map[string][]string{}, started: map[string]int{}, subs: map[string]string{}, subNames: map[string]map[string]bool{}}
}

func (s *goStream) read(ev goEvent) {
	start := ev.Action == "start" && ev.Test == ""
	if ev.Package == "" {
		// a test binary under test2json without -p: a pass of these events
		// counts nothing, a fail goes by name (section 10.3)
		if ev.Test != "" && isVerdict(ev.Action) && !strings.Contains(ev.Test, "/") {
			s.nameless++
		}
		if s.unnamed == nil {
			s.unnamed = newRun()
		}
		s.unnamed.read(ev)
		return
	}
	runs := s.pkgs[ev.Package]
	if start && len(runs) > 0 && runs[len(runs)-1].end == "" && s.restarted == "" {
		// a run killed and started again, or two runs writing at once
		s.restarted = ev.Package
	}
	if len(runs) == 0 || runs[len(runs)-1].end != "" {
		// the first event of the package, or one after its end event: a run
		// of its own (section 10.3)
		runs = append(runs, newRun())
		s.pkgs[ev.Package] = runs
	}
	runs[len(runs)-1].read(ev)
}

// read adds one event to the run.
func (pkg *goPackage) read(ev goEvent) {
	pkg.events++
	verdict := isVerdict(ev.Action)
	parent, _, sub := strings.Cut(ev.Test, "/")
	switch {
	case ev.Test == "" && verdict:
		pkg.end = ev.Action
	case ev.Test == "" && ev.Action == "output":
		// go test marks a package it answered from its cache: "ok  \t<package>\t(cached)"
		if strings.HasPrefix(ev.Output, "ok") && contains(strings.Split(strings.TrimSpace(ev.Output), "\t"), "(cached)") {
			pkg.cached = true
		}
	case ev.Test == "":
	case ev.Action == "run" && !sub:
		pkg.started[ev.Test]++
	case verdict && sub: // read with its parent: a fail fails it, a skip skips it
		if ev.Action == Fail || (ev.Action == Skip && pkg.subs[parent] != Fail) {
			pkg.subs[parent] = ev.Action
		}
		if pkg.subNames[parent] == nil {
			pkg.subNames[parent] = map[string]bool{}
		}
		pkg.subNames[parent][ev.Test] = true
	case verdict:
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
	Subtests    int    `json:"subtests,omitempty"`
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
	// a fail that is no fail event: a function that started and gave no
	// verdict in a run where its package failed (section 10.3), by file, and by
	// package for the summary; explicit: a function with a fail event
	silent, explicit := map[string]map[string]bool{}, map[string]map[string]bool{}
	mark := func(m map[string]map[string]bool, file, fn string) {
		if m[file] == nil {
			m[file] = map[string]bool{}
		}
		m[file][fn] = true
	}
	silentBy := map[string]map[string]bool{}
	// a package that did not run its tests in a run: it failed and no test
	// function of it has a run event there (section 10.3)
	unran := map[string]bool{}
	unrun := map[string]map[string]bool{}
	var passed, failed, skipped, nameless int
	var ambiguous, refused, differ []string
	// a verdict that found no file: by package when no file under
	// trace.tests belongs to the package, by name when one does (section 10.3)
	lostIn := map[string]int{}
	var undefined, outside []string
	lose := func(pkg, fn string, defs []string, cands map[string]bool) {
		switch {
		case len(defs) == 1 && cands[defs[0]]: // a candidate outside trace.tests
			outside = append(outside, fmt.Sprintf("adapter: %s: %s: defined only in %s, which is not under trace.tests: its verdict counts nothing; list the file in trace.tests", pkg, fn, defs[0]))
		case a.mapped(pkg):
			undefined = append(undefined, pkg+": "+fn)
		default:
			lostIn[pkg]++
		}
	}
	// with streams a fail goes by name (section 10.3): the packages each
	// function failed in, whether it had a fail event, and the fails the
	// mapping found no file for
	failedIn, failEvent := map[string]map[string]bool{}, map[string]bool{}
	type pkgFn struct {
		pkg, fn string
		defs    []string
		cands   map[string]bool
	}
	var unplaced []pkgFn
	names := map[string]bool{}
	for _, s := range a.streams {
		nameless += s.nameless
		for name := range s.pkgs {
			names[name] = true
		}
	}
	pkgOK := map[string]bool{}               // package -> every run of it ended with pass
	pkgState := map[string]map[string]int{}  // package -> function -> its state there
	raw := map[string]map[string][2]string{} // file -> function -> its verdict and the package it came from
	subCount := map[string]map[string]int{}  // file -> function -> how many subtests ran (section 10.3)
	for _, name := range sortedKeys(names) {
		// what each run of each stream says of the package. A pass replayed
		// from the cache is not read; every other verdict of every run is
		cached, notOK := false, false
		packageFailed := false
		type seen struct {
			verdict, stream string
			silent          bool // a run event and no verdict, in a stream where the package failed
			packagePassed   bool
		}
		read := map[string][]seen{}
		replayed := map[string]bool{}
		subsOf := map[string]map[string]bool{} // function -> its subtests with a verdict, in runs not answered from the cache
		for _, s := range a.streams {
			for _, pkg := range s.pkgs[name] {
				cached = cached || pkg.cached
				for fn, subs := range pkg.subNames {
					if pkg.cached {
						continue
					}
					if subsOf[fn] == nil {
						subsOf[fn] = map[string]bool{}
					}
					for sub := range subs {
						subsOf[fn][sub] = true
					}
				}
				notOK = notOK || pkg.end == Fail || (pkg.end != Pass && len(pkg.verdicts) > 0)
				for fn, vs := range pkg.verdicts {
					for _, v := range vs {
						switch sub := pkg.subs[fn]; {
						case sub == Fail:
							v = Fail
						case sub == Skip && v == Pass:
							v = Skip
						}
						if pkg.cached && v == Pass {
							replayed[fn] = true
							continue
						}
						read[fn] = append(read[fn], seen{verdict: v, stream: s.name, packagePassed: pkg.end == Pass})
					}
				}
				if pkg.end == Fail { // whatever stopped the binary stopped these
					packageFailed = true
					if len(pkg.started) == 0 {
						unran[name] = true
					}
					for fn, n := range pkg.started {
						if len(pkg.verdicts[fn]) < n { // an iteration that started and gave no verdict
							read[fn] = append(read[fn], seen{verdict: Fail, stream: s.name, silent: true})
						}
					}
				}
			}
		}
		if cached {
			refused = append(refused, fmt.Sprintf("adapter: package %s: answered from the cache; a pass replayed from the cache counts nothing: run go test with -count=1", name))
		}
		if notOK {
			refused = append(refused, fmt.Sprintf("adapter: package %s: did not end with pass; no pass is written from it", name))
		}
		pkgOK[name] = !notOK
		fns := sortedKeys(read)
		for fn := range replayed {
			if read[fn] == nil {
				fns = append(fns, fn)
			}
		}
		sort.Strings(fns)
		suppressed := 0
		for _, fn := range fns {
			verdict := "" // only a pass replayed from the cache
			isSilent, isExplicit := false, false
			passedInSuccessfulRun := false
			if got := read[fn]; len(got) > 0 {
				verdict = got[0].verdict
				same := true
				var parts []string
				for _, g := range got {
					same = same && g.verdict == verdict
					passedInSuccessfulRun = passedInSuccessfulRun || (g.verdict == Pass && g.packagePassed)
					label := g.verdict
					if g.silent {
						label = "no verdict"
						isSilent = true
						if silentBy[name] == nil {
							silentBy[name] = map[string]bool{}
						}
						silentBy[name][fn] = true
					} else if g.verdict == Fail {
						isExplicit = true
					}
					if g.stream != "" {
						label += " in " + g.stream
					}
					parts = append(parts, label)
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
				if a.fromDir {
					if failedIn[fn] == nil {
						failedIn[fn] = map[string]bool{}
					}
					failedIn[fn][name] = true
					failEvent[fn] = failEvent[fn] || isExplicit
				}
			case Skip:
				st = stSkip
				skipped++
			default:
				if verdict == Pass && !notOK {
					st = stPass
				}
				if verdict == Pass && packageFailed && passedInSuccessfulRun {
					suppressed++
				}
				passed++
			}
			if pkgState[name] == nil { // the verdict of the function in this package, for a set (section 10.3)
				pkgState[name] = map[string]int{}
			}
			pkgState[name][fn] = st
			markFail := func(file string) {
				switch {
				case isExplicit:
					mark(explicit, file, fn)
				case isSilent:
					mark(silent, file, fn)
				}
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
						markFail(f)
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
				if n := len(subsOf[fn]); n > 0 {
					if subCount[file] == nil {
						subCount[file] = map[string]int{}
					}
					subCount[file][fn] = max(subCount[file][fn], n)
				}
				set(file, fn, st)
				if st == stFail {
					markFail(file)
				}
			case st == stFail && a.fromDir:
				unplaced = append(unplaced, pkgFn{name, fn, defs, cands}) // it may still go by name, below
			default:
				lose(name, fn, defs, cands)
			}
		}
		if a.fromDir && suppressed > 0 {
			refused = append(refused, fmt.Sprintf("adapter: %s: %d pass(es) not counted because another run of this package failed; give each run its own package name to keep them apart", name, suppressed))
		}
	}

	// with streams a fail needs no package: the fails of the events that name
	// none go by name too (section 10.3), under the package ""
	if a.fromDir {
		for _, s := range a.streams {
			for fn, isExplicit := range s.unnamed.failed() {
				if failedIn[fn] == nil {
					failedIn[fn] = map[string]bool{}
				}
				failedIn[fn][""] = true
				failEvent[fn] = failEvent[fn] || isExplicit
			}
		}
	}

	// a package that did not run its tests: every test function of its
	// candidate files under trace.tests failed, whatever another run gave it.
	// A function without a run event in a run where another has one was not
	// selected or not reached, and gets nothing (section 10.3)
	unranBy := map[string]map[string]bool{}
	for _, name := range sortedKeys(unran) {
		for _, file := range a.candidates(name) {
			f := a.files[file]
			if f == nil {
				continue
			}
			for _, fn := range f.Funcs {
				set(file, fn, stFail)
				mark(unrun, file, fn)
				if unranBy[name] == nil {
					unranBy[name] = map[string]bool{}
				}
				unranBy[name][fn] = true
			}
		}
	}
	// with streams a fail goes by name to every file of the entries that
	// defines the function, whatever package it failed in (section 10.3)
	var byName []string
	placed := map[string]bool{}
	if a.fromDir {
		named := a.sourceFiles()
		for _, fn := range sortedKeys(failedIn) {
			var reached []string
			for _, file := range named {
				if f := a.files[file]; f.Broken || !f.defines(fn) {
					continue
				}
				placed[fn] = true
				if cur, ok := state[file][fn]; ok && cur == stFail {
					continue
				}
				set(file, fn, stFail)
				if failEvent[fn] {
					mark(explicit, file, fn)
				} else {
					mark(silent, file, fn)
				}
				reached = append(reached, file)
			}
			if len(reached) > 0 {
				byName = append(byName, fmt.Sprintf("adapter: %s: failed in %s, so read as failed in %s: a fail goes by name to every file of evidence.adapter.sources that defines it; rename one of the functions if they are unrelated", fn, failedWhere(failedIn[fn]), nameList(reached)))
			}
		}
	}
	for _, u := range unplaced {
		if !placed[u.fn] {
			lose(u.pkg, u.fn, u.defs, u.cands)
		}
	}
	for _, name := range sortedKeys(silentBy) {
		refused = append(refused, fmt.Sprintf("adapter: package %s: %d test function(s) started and gave no verdict, read as failed: %s", name, len(silentBy[name]), nameList(sortedKeys(silentBy[name]))))
	}
	for _, name := range sortedKeys(unranBy) {
		refused = append(refused, fmt.Sprintf("adapter: package %s: did not run its tests: %d test function(s) read as failed: %s", name, len(unranBy[name]), nameList(sortedKeys(unranBy[name]))))
	}

	// the claims the summary counts are those of the files of the packages
	// of the run: their candidates, and every file a verdict reached
	scope := map[string]bool{}
	for name := range names {
		for _, file := range a.candidates(name) {
			scope[file] = true
		}
	}
	var unjudged []string
	for u := range a.unjudged {
		if scope[u.File] {
			unjudged = append(unjudged, fmt.Sprintf("adapter: %s: %s: the adapter does not judge %s; put the marker above a Test function or at file level", u.File, u.ID, u.Func))
		}
	}
	sort.Strings(unjudged)
	for file := range state {
		scope[file] = true
	}

	// one receipt per attribution of a claimed id
	wrote := map[string]int{}
	// per id: a receipt was written, a claim of the run's packages met a
	// function that did not run (any claim, for idleAll), a claim met a
	// verdict that does not count
	has, idle, idleAll, held := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	passedBy, failedID := map[string]map[Claimant]bool{}, map[string]bool{}
	var lines [][]byte
	// a helper judged by its set (section 10.3): its file -> the packages of
	// its set
	sets := map[string][]string{}
	for _, u := range a.units {
		f := a.files[u.File]
		var funcs []fileFn
		var pkgs []string // the packages of the set of a helper
		broken := false
		switch {
		case u.Func != "":
			funcs = []fileFn{{u.File, u.Func, ""}}
		case len(f.Funcs) > 0:
			for _, fn := range f.Funcs {
				funcs = append(funcs, fileFn{u.File, fn, ""})
			}
		case !f.Broken && !f.Constrained: // a file a run may not have built is judged by no set
			funcs, pkgs, broken = a.setOf(u.File, names)
		}
		if len(funcs) == 0 {
			continue // no test function and no set, or a file that does not parse: named in the summary
		}
		bySet := len(pkgs) > 0
		if bySet {
			sets[u.File] = pkgs
		}
		st := stPass
		if broken { // a candidate of the set does not parse: its test functions are not known
			st = stAmbiguous
		}
		for _, pkg := range pkgs { // the set passes only when its package ended with pass
			if !pkgOK[pkg] && st > stBlocked {
				st = stBlocked
			}
		}
		var failedFns, startedFns, unranFns, skippedFns []string
		for _, ff := range funcs {
			file, fn := ff.file, ff.fn
			fst, ran := state[file][fn]
			if !ran {
				fst = stNotRun
			}
			if ff.pkg != "" { // in a set the function is judged in the package of the set as well
				if pst, ok := pkgState[ff.pkg][fn]; !ok {
					fst = min(fst, stNotRun)
				} else {
					fst = min(fst, pst)
				}
			}
			if fst < st {
				st = fst
			}
			switch {
			case fst == stFail && explicit[file][fn]:
				failedFns = append(failedFns, fn)
			case fst == stFail && silent[file][fn]:
				startedFns = append(startedFns, fn)
			case fst == stFail && unrun[file][fn]:
				unranFns = append(unranFns, fn)
			case fst == stFail:
				failedFns = append(failedFns, fn)
			case fst == stSkip:
				skippedFns = append(skippedFns, fn)
			}
		}
		ofSet := "" // how the observed text says the functions are those of a set
		if bySet {
			ofSet = " of its set"
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
			switch {
			case u.Func != "":
				r.Observed = "go test: " + u.Func + " passed"
			case bySet:
				r.Observed = fmt.Sprintf("go test: all %d test function(s) of its set passed (%s)", len(funcs), packageWords(pkgs))
			}
			for _, ff := range funcs { // a claim satisfied through a parent test says how many subtests ran
				r.Subtests += subCount[ff.file][ff.fn]
			}
			if r.Subtests > 0 {
				r.Observed += fmt.Sprintf(" (subtests: %d)", r.Subtests)
			}
		case stFail:
			r.Outcome = Fail
			switch {
			case len(failedFns) > 0:
				r.Observed = "go test: " + nameList(failedFns) + ofSet + " failed"
			case len(startedFns) > 0:
				r.Observed = "go test: " + nameList(startedFns) + ofSet + ": started, no verdict; the package failed"
			default:
				r.Observed = "go test: " + nameList(unranFns) + ofSet + ": the package did not run its tests"
			}
		case stSkip:
			r.Outcome, r.Observed = Skip, "go test: "+nameList(skippedFns)+ofSet+" skipped"
		case stNotRun:
			idleAll[u.ID] = true
			if scope[u.File] {
				idle[u.ID] = true
			}
			continue
		default:
			held[u.ID] = true
			continue
		}
		wrote[r.Outcome]++
		has[u.ID] = true
		lines = append(lines, jsonLine(r))
		switch r.Outcome {
		case Pass:
			if passedBy[u.ID] == nil {
				passedBy[u.ID] = map[Claimant]bool{}
			}
			passedBy[u.ID][Claimant{File: u.File, Func: u.Func}] = true
		case Fail:
			failedID[u.ID] = true
		}
	}
	a.wrote = len(lines)
	a.sets = sortedKeys(sets)
	// partly run in this run: an id with a pass and no fail here, and a
	// claimant that is not excused without a pass here (section 10.3)
	var partly, partlyPassed, partlyOf int
	if a.p.Cfg.AdapterClaimsAll() {
		for _, id := range sortedKeys(passedBy) {
			open, _ := a.claimants.open(id)
			if failedID[id] || len(open) == 0 {
				continue
			}
			n := 0
			for _, cl := range open {
				if passedBy[id][cl] {
					n++
				}
			}
			if n < len(open) {
				partly, partlyPassed, partlyOf = partly+1, partlyPassed+n, partlyOf+len(open)
			}
		}
	}
	var notRun, notCounted int
	for id := range idle {
		if !has[id] {
			notRun++
		}
	}
	a.idleAll = nil
	for _, id := range sortedKeys(idleAll) {
		if !has[id] {
			a.idleAll = append(a.idleAll, id)
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
		switch {
		case len(a.streams) > 0:
		case a.AllowNoStreams:
			a.summary = append(a.summary, fmt.Sprintf("adapter %s: no stream in %s: the command ran nothing the adapter can read, and nothing is proven", GoTestJSON, StreamsEnv))
		default: // a run that reads no stream is not a success (section 10.3)
			a.summary = append(a.summary, fmt.Sprintf("adapter %s: no stream was written to %s: the command ran no go test, or it cleared the environment (env -i) before starting one; pass --allow-no-streams if it may run nothing", GoTestJSON, StreamsEnv))
		}
	}
	if !a.fromDir || len(a.streams) > 0 {
		a.summary = append(a.summary, fmt.Sprintf("adapter %s: %s%d package(s), %d test function(s) with a verdict: %d passed, %d failed, %d skipped", GoTestJSON, counted, len(names), nFuncs, passed, failed, skipped))
	}
	// the claiming files: those of the run's packages without a test function
	// are named, every one is counted
	claiming := map[string]bool{}
	for _, u := range a.units {
		claiming[u.File] = true
	}
	var empty, broken []string
	a.emptyAll = 0
	for _, rel := range sortedKeys(claiming) {
		switch f := a.files[rel]; {
		case f.Broken:
			broken = append(broken, "adapter: "+unreadableClaimant(rel, f.Err))
		case len(f.Funcs) == 0:
			a.emptyAll++
			switch {
			case !scope[rel]:
			case f.Constrained:
				empty = append(empty, fmt.Sprintf("adapter: %s: no test function; it carries a build constraint, so no set judges it: its claims count nothing", rel))
			case len(sets[rel]) > 0:
				empty = append(empty, fmt.Sprintf("adapter: %s: no test function; its claims are judged by its set: every test function of %s", rel, packageWords(sets[rel])))
			default:
				empty = append(empty, fmt.Sprintf("adapter: %s: no test function; its claims count nothing", rel))
			}
		}
	}
	a.summary = append(a.summary,
		fmt.Sprintf("adapter: receipts written: %d pass, %d fail, %d skip", wrote[Pass], wrote[Fail], wrote[Skip]),
		fmt.Sprintf("adapter: claimed, not run: %d id(s)", notRun),
	)
	a.whole = len(a.summary)
	a.summary = append(a.summary, a.wholeLine(nil))
	add := func(n int, format string) {
		if n > 0 {
			a.summary = append(a.summary, fmt.Sprintf(format, n))
		}
	}
	if partly > 0 {
		a.summary = append(a.summary, fmt.Sprintf("adapter: partly run: %d id(s): %d of %d claimants passed in this run", partly, partlyPassed, partlyOf))
	}
	add(notCounted, "adapter: claimed, verdict not counted: %d id(s)")
	add(len(a.effects), "adapter: needs a receipt with effects: %d tc id(s)")
	add(len(a.replay), "adapter: needs a receipt with a replay: %d cx id(s)")
	if len(lostIn) > 0 { // every package named, with its count
		todo := "a package's verdicts go to the files under trace.tests in its own directory, and a package built from copies is run through --streams with an entry in evidence.adapter.sources"
		if a.fromDir {
			todo = "with --streams a package's verdicts go only to the files of its entry in evidence.adapter.sources: add one"
		}
		n := 0
		var parts []string
		for _, pkg := range sortedKeys(lostIn) {
			n += lostIn[pkg]
			parts = append(parts, fmt.Sprintf("%s (%d)", pkg, lostIn[pkg]))
		}
		a.summary = append(a.summary, fmt.Sprintf("adapter: %d test function(s) with a verdict are in package(s) no file under trace.tests belongs to: %s; %s", n, strings.Join(parts, ", "), todo))
	}
	advice := "list its file in trace.tests, or run a package built from copies through --streams"
	if a.fromDir {
		advice = "if it is staged from another directory, give that set its own name"
	}
	sort.Strings(undefined)
	for _, u := range undefined {
		a.summary = append(a.summary, fmt.Sprintf("adapter: %s: defined in no candidate file; %s", u, advice))
	}
	sort.Strings(outside)
	a.summary = append(a.summary, outside...)
	if a.fromDir {
		add(nameless, "adapter: %d verdict(s) name no package: a pass or a skip of them counts nothing, a fail goes by name; run the test binary under go tool test2json -p <import path>")
	} else {
		add(nameless, "adapter: %d verdict(s) name no package and are used for nothing: run the test binary under go tool test2json -p <import path>")
	}
	sort.Strings(differ)
	a.summary = append(append(append(append(append(a.summary, refused...), ambiguous...), differ...), byName...), unjudged...)
	a.summary = append(append(a.summary, empty...), broken...)
	if a.keptAt != "" {
		a.summary = append(a.summary, fmt.Sprintf("adapter: streams kept in %s: %d file(s)", a.keptAt, a.kept))
	}
	return lines
}

// wholeLine renders the whole-project line of the summary, leaving out the
// ids evidenced says another run has evidence for (nil: none left out).
func (a *GoTest) wholeLine(evidenced func(string) bool) string {
	n := 0
	for _, id := range a.idleAll {
		if evidenced == nil || !evidenced(id) {
			n++
		}
	}
	return fmt.Sprintf("adapter: in the whole project: %d claimed id(s) not run, %d claiming file(s) with no test function, %d claim(s) above functions the adapter does not judge", n, a.emptyAll, len(a.unjudged))
}

// Wrote is the number of receipts the adapter wrote, after Receipts.
func (a *GoTest) Wrote() int { return a.wrote }

// Sets are the helpers the run judged by their set, sorted (section 10.3):
// the end line of the run records them, after Receipts.
func (a *GoTest) Sets() []string { return a.sets }

// fileFn is a test function of a file and, in a set, the package of the
// set (section 10.3).
type fileFn struct{ file, fn, pkg string }

// setOf returns the functions a helper is judged by (section 10.3, "Whose
// claim"): for every package of the run whose candidate it is, every test
// function of that package's candidate files, and those packages, sorted.
// broken reports a candidate that does not parse, whose test functions are
// not known.
func (a *GoTest) setOf(file string, names map[string]bool) (funcs []fileFn, pkgs []string, broken bool) {
	seen := map[fileFn]bool{}
	for _, name := range sortedKeys(names) {
		cands := a.candidates(name)
		if !contains(cands, file) {
			continue
		}
		pkgs = append(pkgs, name)
		for _, c := range cands {
			f := a.parsed(c)
			if f.Broken {
				broken = true
				continue
			}
			for _, fn := range f.Funcs {
				if ff := (fileFn{c, fn, name}); !seen[ff] {
					seen[ff] = true
					funcs = append(funcs, ff)
				}
			}
		}
	}
	return funcs, pkgs, broken
}

// packageWords names packages: "package <p>", or "packages <p>, <q>".
func packageWords(pkgs []string) string {
	if len(pkgs) == 1 {
		return "package " + pkgs[0]
	}
	return "packages " + nameList(append([]string(nil), pkgs...))
}

// NoStream reports a run through --streams that read no stream at all.
func (a *GoTest) NoStream() bool { return a.fromDir && len(a.streams) == 0 }

// Rescope counts on the whole-project line of a run that produced no
// evidence only the ids without evidence at the commit (section 10.3).
func (a *GoTest) Rescope(evidenced func(string) bool) {
	if a.whole < len(a.summary) {
		a.summary[a.whole] = a.wholeLine(evidenced)
	}
}

// failed returns the test functions the events of a stream that name no
// package read as failed (section 10.3, "A fail goes by name"): true for a
// fail event, its own or a subtest's, and false for a function with more
// run events than verdicts. The events are one whole, so a run cut short is
// not hidden by a later one that passes, and how a run ended is not asked.
func (pkg *goPackage) failed() map[string]bool {
	res := map[string]bool{}
	if pkg == nil {
		return res
	}
	for fn, vs := range pkg.verdicts {
		if contains(vs, Fail) {
			res[fn] = true
		}
	}
	for fn, sub := range pkg.subs {
		if sub == Fail {
			res[fn] = true
		}
	}
	for fn, n := range pkg.started {
		if _, seen := res[fn]; !seen && len(pkg.verdicts[fn]) < n {
			res[fn] = false
		}
	}
	return res
}

// failedWhere says where a function failed, for the summary: its packages,
// sorted, and "" for the events that name no package.
func failedWhere(in map[string]bool) string {
	var pkgs, parts []string
	for name := range in {
		if name != "" {
			pkgs = append(pkgs, name)
		}
	}
	if len(pkgs) > 0 {
		parts = append(parts, "package(s) "+nameList(pkgs))
	}
	if in[""] {
		parts = append(parts, "events that name no package")
	}
	return strings.Join(parts, " and in ")
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
	} else {
		f.Err = readError(err)
	}
	a.other[file] = f
	return f
}

// candidates returns the files a verdict of the package may belong to
// (section 10.3, "Whose verdict"): without streams the test files under
// trace.tests in the package's own directory, with streams the files the
// entries of evidence.adapter.sources map to the package.
func (a *GoTest) candidates(pkg string) []string {
	if !a.fromDir {
		return a.pkgFiles[pkg]
	}
	if res, ok := a.cands[pkg]; ok {
		return res
	}
	set := map[string]bool{}
	for _, s := range a.sources {
		if !s.Matches(pkg) {
			continue
		}
		if s.Dir { // the test files of the package's own directory
			for _, file := range a.pkgFiles[pkg] {
				set[file] = true
			}
			continue
		}
		var named []string
		for _, file := range a.projectFiles() {
			if strings.HasSuffix(file, ".go") && project.Match(s.Files, file) {
				named = append(named, file)
			}
		}
		// a glob of packages maps to each the files of its own directory when it
		// names some, and every file to a package built from copies elsewhere
		var own []string
		if s.Wildcard() {
			for _, file := range named {
				if ip, ok := a.importPathOf(path.Dir(file)); ok && ip == pkg {
					own = append(own, file)
				}
			}
		}
		if len(own) == 0 {
			own = named
		}
		for _, file := range own {
			set[file] = true
		}
	}
	res := sortedKeys(set)
	a.cands[pkg] = res
	return res
}

// mapped reports whether a file under trace.tests is a candidate of the
// package: a verdict of it that finds no file is then one its candidates do
// not define (section 10.3).
func (a *GoTest) mapped(pkg string) bool {
	for _, file := range a.candidates(pkg) {
		if a.files[file] != nil {
			return true
		}
	}
	return false
}

// sourceFiles returns the files under trace.tests that the entries of
// evidence.adapter.sources name, sorted: the Go files an entry with files
// matches, and the test files of every directory whose import path the
// pattern of an entry with dir: true matches.
func (a *GoTest) sourceFiles() []string {
	set := map[string]bool{}
	for _, s := range a.sources {
		if s.Dir {
			for ip, files := range a.pkgFiles {
				if s.Matches(ip) {
					for _, file := range files {
						set[file] = true
					}
				}
			}
			continue
		}
		for file := range a.files {
			if strings.HasSuffix(file, ".go") && project.Match(s.Files, file) {
				set[file] = true
			}
		}
	}
	return sortedKeys(set)
}

// projectFiles returns every file of the project, read once.
func (a *GoTest) projectFiles() []string {
	if a.all == nil {
		a.all, _ = a.p.Files()
		if a.all == nil {
			a.all = []string{}
		}
	}
	return a.all
}

// importPathOf returns the import path of a project directory, read once.
func (a *GoTest) importPathOf(dir string) (string, bool) {
	if ip, ok := a.dirIP[dir]; ok {
		return ip, ip != ""
	}
	ip, _ := importPath(a.p.Abs(dir))
	a.dirIP[dir] = ip
	return ip, ip != ""
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
