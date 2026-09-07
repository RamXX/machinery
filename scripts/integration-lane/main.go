// Command integration-lane executes the closed required native test inventory.
//
// Every process this lane launches executes under verified native
// processscope custody (processcontrol.WithScope): registration before
// launch, guardian-owned process groups, terminal group kill before reap, a
// cumulative wall budget and owner liveness. The formal provisioning helper
// joins the inherited capability so its nested Java identity probes and TLC
// engine executions are guarded jobs of the same root. Custody that cannot
// be established, and cleanup that cannot be verified, fail closed.
//
// Residual behavior, stated honestly rather than papered over: an uncatchable
// owner death or a descendant that deliberately escapes both registration and
// its own process group (for example a direct setsid, or an engine launched
// with a fresh background context inside foreign test code) is beyond
// transitive custody. processscope reports such evidence as cleanup-failed,
// never as false success, and this lane never uses process-name or PID
// pattern matching as cleanup authority.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/RamXX/machinery/internal/formal"
	"github.com/RamXX/machinery/internal/processcontrol"
	"github.com/RamXX/machinery/internal/processscope"
	"github.com/RamXX/machinery/internal/runtimeclosure"
)

const maxOutput = 64 << 20
const tlcSHA = "936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88"
const ownerLabel = "dev.machinery.integration-run"
const schemaSHA = "867229ec5b7f3b22d4fe2fe3b88c6ed4cea5bf9f1ec2239a3e0635b431c1e7f2"

type suite struct {
	ID           string   `json:"id"`
	Lane         string   `json:"lane"`
	Adapter      string   `json:"adapter"`
	Package      string   `json:"package"`
	Sources      []string `json:"source_files"`
	Tests        []string `json:"tests"`
	Runtimes     []string `json:"runtimes"`
	Timeout      string   `json:"timeout"`
	StdoutLimit  int      `json:"stdout_limit"`
	StderrLimit  int      `json:"stderr_limit"`
	sourceByTest map[string]string
	sourceHashes map[string]string
}
type fragment struct {
	Version int     `json:"version"`
	Suites  []suite `json:"suites"`
}
type pins struct {
	Version int `json:"version"`
	Docker  struct {
		Image    string `json:"image"`
		Platform string `json:"platform"`
		Memory   string `json:"memory"`
		CPUs     string `json:"cpus"`
		PIDs     int    `json:"pids_limit"`
	} `json:"docker"`
	Java struct {
		File    string `json:"pin_file"`
		Version string `json:"version"`
	} `json:"java"`
	TLC struct {
		Version string `json:"version"`
		SHA     string `json:"sha256"`
	} `json:"tlc"`
	Node struct {
		Minimum int `json:"minimum_major"`
	} `json:"node"`
}
type testReceipt struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Status string `json:"status"`
}
type suiteReceipt struct {
	ID        string        `json:"id"`
	Adapter   string        `json:"adapter"`
	Selected  int           `json:"selected"`
	Started   int           `json:"started"`
	Passed    int           `json:"passed"`
	Failed    int           `json:"failed"`
	Skipped   int           `json:"skipped"`
	Tests     []testReceipt `json:"tests"`
	Events    string        `json:"events_file"`
	EventsSHA string        `json:"events_sha256"`
}
type runtimeReceipt struct {
	ID         string `json:"id"`
	Identity   string `json:"identity"`
	Status     string `json:"status"`
	Path       string `json:"path"`
	SHA        string `json:"sha256"`
	ClosureSHA string `json:"closure_sha256,omitempty"`
	ArchiveSHA string `json:"archive_sha256,omitempty"`
}
type report struct {
	Version   int              `json:"version"`
	Lane      string           `json:"lane"`
	Status    string           `json:"status"`
	Suites    []suiteReceipt   `json:"suites"`
	Runtimes  []runtimeReceipt `json:"runtimes"`
	Assurance *assuranceReport `json:"assurance,omitempty"`
	Cleanup   struct {
		Status string `json:"status"`
	} `json:"cleanup"`
	Custody custodyReceipt `json:"custody"`
}

// custodyReceipt binds verifiable native custody evidence into the report:
// the guard mode, the number of guarded jobs, and a status that is passed
// only when the custody close verified every job registered, terminated and
// reaped.
type custodyReceipt struct {
	Status string `json:"status"`
	Mode   string `json:"mode"`
	Jobs   int    `json:"jobs"`
}

func main() {
	// Authenticated internal custody activation comes before any ordinary
	// lane execution so this binary can serve as the broker/guardian helper
	// of its own custody root. Malformed or forged claims are refused, never
	// silently reduced to an unscoped run.
	if code, handled := serveInternalActivation(os.Args[1:]); handled {
		os.Exit(code)
	}
	// Provisioning runs in a child so cache/HOME and engine overrides never
	// mutate the caller process. Its output and entire lifetime are bounded.
	if len(os.Args) == 3 && os.Args[1] == "provision-formal" {
		if err := provisionFormal(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// serveInternalActivation acquires the inherited internal channel error-first
// and serves verified broker/guardian requests with this same binary. It
// reports handled=false only for the verified absence of any internal claim.
func serveInternalActivation(args []string) (exitCode int, handled bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	io, present, err := processscope.InheritedInternalIO(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "integration-lane: refusing untrusted internal activation:", err)
		return 1, true
	}
	if !present {
		if len(args) > 0 && args[0] == processscope.InternalMarker {
			fmt.Fprintln(os.Stderr, "machinery: internal marker without an authenticated internal channel; refusing normal lane execution")
			return 1, true
		}
		return 0, false
	}
	defer io.Close()
	wasHandled, code := processscope.ServeInternal(args, io)
	if !wasHandled {
		fmt.Fprintln(os.Stderr, "integration-lane: internal channel claim without the internal marker; refusing normal execution")
		return 1, true
	}
	return code, true
}

func run(args []string, stdout, stderr io.Writer) (status int) {
	flags := flag.NewFlagSet("integration-lane", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "repository root")
	lane := flags.String("lane", "", "required named lane")
	reportPath := flags.String("report", "", "retained machine-readable report")
	workParent := flags.String("work-dir", "", "parent for private scratch")
	cache := flags.String("cache-dir", "", "private runtime cache")
	pinPath := flags.String("pins", "", "runtime pin file")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *lane != "required" {
		fmt.Fprintln(stderr, "require --lane required and no positional commands")
		return 2
	}
	r := report{Version: 1, Lane: *lane, Status: "failed", Suites: []suiteReceipt{}, Runtimes: []runtimeReceipt{}}
	r.Cleanup.Status = "passed"
	r.Custody.Status = "failed"
	status = 1
	var work string
	defer func() {
		if work != "" {
			if err := os.RemoveAll(work); err != nil {
				fmt.Fprintln(stderr, "cleanup:", err)
				r.Cleanup.Status = "failed"
				status = 1
			}
			if _, err := os.Lstat(work); !os.IsNotExist(err) {
				fmt.Fprintln(stderr, "cleanup: owned scratch remains", work)
				r.Cleanup.Status = "failed"
				status = 1
			}
		}
		if status == 0 {
			r.Status = "passed"
		}
		if *reportPath != "" {
			if err := writeJSON(*reportPath, r); err != nil {
				fmt.Fprintln(stderr, "report:", err)
				status = 1
			}
		}
	}()
	var err error
	*root, err = filepath.Abs(*root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	suites, err := inventory(*root)
	if err != nil {
		fmt.Fprintln(stderr, "inventory:", err)
		return 1
	}
	// The four-language assurance catalog validates before any provisioning
	// or suite execution so a stale, incomplete or tampered catalog fails
	// closed fast; foreign roots without it keep exact v1 semantics.
	catalog, err := loadAssuranceCatalog(*root, suites)
	if err != nil {
		fmt.Fprintln(stderr, "assurance:", err)
		return 1
	}
	if *pinPath == "" {
		*pinPath = filepath.Join(*root, "testdata", "integration-lanes", "runtime-pins.json")
	}
	var policy pins
	if err := decodeFile(*pinPath, &policy); err != nil {
		fmt.Fprintln(stderr, "pins:", err)
		return 1
	}
	if *reportPath == "" {
		dir, e := os.MkdirTemp("", "machinery-integration-evidence-")
		if e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
		*reportPath = filepath.Join(dir, "report.json")
	}
	*reportPath, err = filepath.Abs(*reportPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err = os.MkdirAll(filepath.Dir(*reportPath), 0700); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *workParent != "" {
		if err = os.MkdirAll(*workParent, 0700); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	work, err = os.MkdirTemp(*workParent, "machinery-integration-work-")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	work, err = filepath.Abs(work)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *cache == "" {
		*cache = filepath.Join(filepath.Dir(*reportPath), "runtime-cache")
	}
	*cache, err = filepath.Abs(*cache)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err = os.MkdirAll(*cache, 0700); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	// Native custody guards every process the lane launches. A present-but-
	// malformed inherited custody capability fails closed instead of degrading
	// to unscoped execution; a valid one is honored for the whole run.
	custody, err := newLaneCustody()
	if err != nil {
		fmt.Fprintln(stderr, "custody:", err)
		return 1
	}
	defer func() {
		if err := custody.finalize(&r.Custody); err != nil {
			status = 1
			fmt.Fprintln(stderr, "custody cleanup did not verify:", err)
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	runtimes, err := provision(ctx, custody, *root, work, *cache, policy, suites)
	r.Runtimes = runtimes
	if err != nil {
		fmt.Fprintln(stderr, "provision:", err)
		return 1
	}
	var assuranceRuntimes []runtimeReceipt
	if catalog != nil {
		r.Assurance = &assuranceReport{
			Schema: assuranceSchemaIdentity, Status: "failed",
			Platform: runtime.GOOS + "/" + runtime.GOARCH,
			Adapters: []adapterReceipt{}, Runtimes: []runtimeReceipt{}, Suites: []suiteReceipt{},
		}
		// All four native adapter runtimes are verified before any suite of
		// either lane runs; a missing or mismatched runtime is a hard
		// prerequisite failure, never a skipped fragment.
		assuranceRuntimes, err = provisionAssurance(ctx, custody, work, catalog)
		r.Assurance.Runtimes = assuranceRuntimes
		if err != nil {
			fmt.Fprintln(stderr, "assurance provision:", err)
			return 1
		}
	}
	for _, s := range suites {
		scratch, e := os.MkdirTemp(work, s.ID+"-")
		if e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
		receipt, e := execute(ctx, custody, *root, scratch, *cache, filepath.Dir(*reportPath), s, runtimes)
		r.Suites = append(r.Suites, receipt)
		dockerRequired := false
		for _, id := range s.Runtimes {
			dockerRequired = dockerRequired || id == "docker"
		}
		cleanupErr := cleanupContainers(custody, scratch, filepath.Base(scratch), policy, dockerRequired)
		if cleanupErr != nil {
			r.Cleanup.Status = "failed"
		}
		if err = errors.Join(e, cleanupErr); err != nil {
			fmt.Fprintln(stderr, s.ID+":", err)
			return 1
		}
	}
	if catalog != nil {
		paths := map[string]string{}
		for _, runtime := range assuranceRuntimes {
			paths[runtime.ID] = runtime.Path
		}
		for i := range catalog.suites {
			probe := catalog.suites[i]
			scratch, e := os.MkdirTemp(work, probe.ID+"-")
			if e != nil {
				fmt.Fprintln(stderr, e)
				return 1
			}
			var receipt suiteReceipt
			if probe.Kind == assuranceConformanceKind {
				// Adapter-native conformance suites execute through the
				// closed production adapter chain, not the probe harness.
				switch probe.Adapter {
				case "go-testing/v1":
					receipt, e = executeGoConformanceSuite(ctx, custody, *root, scratch, filepath.Dir(*reportPath), probe)
				case "node-test-typescript/v1":
					receipt, e = executeTypeScriptConformanceSuite(ctx, custody, *root, scratch, filepath.Dir(*reportPath), probe)
				default:
					e = fmt.Errorf("adapter %s owns no conformance executor", probe.Adapter)
				}
			} else {
				receipt, e = executeAssuranceSuite(ctx, custody, *root, scratch, filepath.Dir(*reportPath), probe, paths)
			}
			r.Assurance.Suites = append(r.Assurance.Suites, receipt)
			if e != nil {
				fmt.Fprintln(stderr, probe.ID+":", e)
				return 1
			}
		}
		counts := map[string]int{}
		for _, probe := range catalog.suites {
			// The per-adapter receipt summarizes the frozen runtime-probe
			// catalog; native-conformance suites are accounted in the suite
			// receipts and their retained normalized event streams.
			if probe.Kind == assuranceProbeKind {
				counts[probe.Adapter]++
			}
		}
		for _, id := range assuranceClosedAdapters {
			r.Assurance.Adapters = append(r.Assurance.Adapters, adapterReceipt{ID: id, Suites: counts[id], Status: "passed"})
		}
		r.Assurance.Status = "passed"
	}
	if catalog != nil {
		_, err = fmt.Fprintf(stdout, "required integration lane: %d suites passed; %d assurance suites passed across %d native adapters; report %s\n", len(r.Suites), len(r.Assurance.Suites), len(r.Assurance.Adapters), *reportPath)
	} else {
		_, err = fmt.Fprintf(stdout, "required integration lane: %d suites passed; report %s\n", len(r.Suites), *reportPath)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func writeJSON(path string, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0600)
}

func decodeFile(path string, value any) error {
	b, err := regularBytes(path, 4<<20)
	if err != nil {
		return err
	}
	if err := uniqueJSON(json.NewDecoder(bytes.NewReader(b)), 0); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.Join(fmt.Errorf("invalid trailing JSON"), err)
	}
	return nil
}

// JSON permits repeated keys, but a closed policy cannot have two competing
// values for one field. Reject them before decoding into typed policy.
func uniqueJSON(d *json.Decoder, depth int) error {
	if depth > 32 {
		return fmt.Errorf("JSON nesting limit exceeded")
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	delimiter, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	for d.More() {
		if delimiter == '{' {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok {
				return fmt.Errorf("invalid JSON key")
			}
			if seen[name] {
				return fmt.Errorf("duplicate JSON field %s", name)
			}
			seen[name] = true
		}
		if err := uniqueJSON(d, depth+1); err != nil {
			return err
		}
	}
	_, err = d.Token()
	return err
}

func regularBytes(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("path %s must be bounded regular file, not symlink", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	b, readErr := io.ReadAll(io.LimitReader(f, limit+1))
	opened, statErr := f.Stat()
	closeErr := f.Close()
	if err := errors.Join(readErr, statErr, closeErr); err != nil {
		return nil, err
	}
	if !os.SameFile(info, opened) || len(b) > int(limit) {
		return nil, fmt.Errorf("path identity/limit changed: %s", path)
	}
	return b, nil
}

var pathPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_./-]*$`)
var suitePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var timeoutPattern = regexp.MustCompile(`^[1-9][0-9]*(ms|s|m)$`)
var nodeCasePattern = regexp.MustCompile(`\btest(?:\.(?:skip|todo))?\(\s*(['"])([^'"\r\n]+)['"]`)

func sourcePath(root, path string) (string, error) {
	if !pathPattern.MatchString(path) || filepath.IsAbs(path) || filepath.Clean(path) != path || path == ".." || strings.HasPrefix(path, "../") {
		return "", fmt.Errorf("invalid source path %q", path)
	}
	joined := root
	for _, part := range strings.Split(path, "/") {
		joined = filepath.Join(joined, part)
		info, err := os.Lstat(joined)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlink source path %s", path)
		}
	}
	return joined, nil
}

func unique(values []string, label string) error {
	if len(values) == 0 {
		return fmt.Errorf("empty %s", label)
	}
	seen := map[string]bool{}
	for _, v := range values {
		if v == "" || seen[v] {
			return fmt.Errorf("empty/duplicate %s %q", label, v)
		}
		seen[v] = true
	}
	return nil
}

func inventory(root string) ([]suite, error) {
	dir := filepath.Join(root, "testdata", "integration-lanes")
	schema, err := regularBytes(filepath.Join(dir, "schema.json"), 1<<20)
	if err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(schema)) != schemaSHA {
		return nil, fmt.Errorf("schema does not match supported closed version 1")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var suites []suite
	for _, entry := range entries {
		name := entry.Name()
		if name == "schema.json" || name == "runtime-pins.json" || name == "CONTRACT.md" {
			continue
		}
		// Assurance-catalog members (MAC-bz1y) are validated by the closed
		// assurance loader, not interpreted as v1 fragments.
		if name == "assurance.schema.json" || name == "assurance-runtime-pins.json" || name == "assurance.CONTRACT.md" || name == "assurance-probes" {
			continue
		}
		if strings.HasPrefix(name, "assurance-") && strings.HasSuffix(name, ".json") {
			continue
		}
		if strings.HasSuffix(name, ".integration.test.mjs") {
			continue
		}
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			return nil, fmt.Errorf("unknown fragment %s", name)
		}
		var f fragment
		if err = decodeFile(filepath.Join(dir, name), &f); err != nil {
			return nil, fmt.Errorf("fragment %s: %w", name, err)
		}
		if f.Version != 1 {
			return nil, fmt.Errorf("fragment version %d", f.Version)
		}
		if len(f.Suites) == 0 {
			return nil, fmt.Errorf("empty fragment %s", name)
		}
		suites = append(suites, f.Suites...)
	}
	if len(suites) == 0 {
		return nil, fmt.Errorf("empty required lane")
	}
	ids, owned, identities := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for i := range suites {
		s := &suites[i]
		if ids[s.ID] {
			return nil, fmt.Errorf("duplicate suite %s", s.ID)
		}
		ids[s.ID] = true
		if !suitePattern.MatchString(s.ID) || s.Lane != "required" {
			return nil, fmt.Errorf("invalid suite ID/lane %s", s.ID)
		}
		if s.Adapter != "go-json" && s.Adapter != "node-tap" {
			return nil, fmt.Errorf("unknown adapter %s", s.Adapter)
		}
		pkg := strings.TrimPrefix(s.Package, "./")
		if s.Package != "." && (!strings.HasPrefix(s.Package, "./") || !pathPattern.MatchString(pkg) || filepath.Clean(pkg) != pkg || strings.Contains(pkg, "..")) {
			return nil, fmt.Errorf("invalid package %q", s.Package)
		}
		if err := errors.Join(unique(s.Sources, "source"), unique(s.Tests, "test"), unique(s.Runtimes, "runtime")); err != nil {
			return nil, err
		}
		duration, e := time.ParseDuration(s.Timeout)
		if e != nil || !timeoutPattern.MatchString(s.Timeout) || duration <= 0 || duration > 30*time.Minute {
			return nil, fmt.Errorf("invalid timeout limit %s", s.Timeout)
		}
		if s.StdoutLimit < 1 || s.StderrLimit < 1 || s.StdoutLimit > maxOutput || s.StderrLimit > maxOutput {
			return nil, fmt.Errorf("invalid output limit")
		}
		runtimeSet := map[string]bool{}
		for _, r := range s.Runtimes {
			if r != "go" && r != "node" && r != "docker" && r != "java" && r != "tlc" {
				return nil, fmt.Errorf("unknown runtime %s", r)
			}
			runtimeSet[r] = true
		}
		if (s.Adapter == "go-json" && !runtimeSet["go"]) || (s.Adapter == "node-tap" && !runtimeSet["node"]) || runtimeSet["java"] != runtimeSet["tlc"] {
			return nil, fmt.Errorf("incomplete adapter/runtime closure")
		}
		s.sourceByTest = map[string]string{}
		s.sourceHashes = map[string]string{}
		for _, source := range s.Sources {
			if owned[source] {
				return nil, fmt.Errorf("duplicate source identity %s", source)
			}
			owned[source] = true
			path, e := sourcePath(root, source)
			if e != nil {
				return nil, e
			}
			b, e := regularBytes(path, 4<<20)
			if e != nil {
				return nil, e
			}
			s.sourceHashes[source] = fmt.Sprintf("%x", sha256.Sum256(b))
			var names []string
			if s.Adapter == "go-json" {
				if !strings.HasSuffix(source, "_test.go") || filepath.Dir(source) != pkg {
					return nil, fmt.Errorf("source/package mismatch %s", source)
				}
				if !integrationTagged(b) {
					return nil, fmt.Errorf("source %s needs explicit machinery_integration build tag", source)
				}
				f, e := parser.ParseFile(token.NewFileSet(), path, b, 0)
				if e != nil {
					return nil, e
				}
				for _, decl := range f.Decls {
					if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test") && fn.Name.Name != "TestMain" {
						names = append(names, fn.Name.Name)
					}
				}
			} else {
				if !strings.HasSuffix(source, ".integration.test.mjs") {
					return nil, fmt.Errorf("invalid Node source path %s", source)
				}
				for _, match := range nodeCasePattern.FindAllSubmatch(b, -1) {
					names = append(names, string(match[2]))
				}
			}
			if len(names) == 0 {
				return nil, fmt.Errorf("empty source execution inventory %s", source)
			}
			for _, name := range names {
				if s.sourceByTest[name] != "" {
					return nil, fmt.Errorf("duplicate native test %s", name)
				}
				s.sourceByTest[name] = source
			}
		}
		for _, name := range s.Tests {
			if s.sourceByTest[name] == "" {
				return nil, fmt.Errorf("registered test %s absent from source", name)
			}
			key := s.Adapter + ":" + s.Package + ":" + name
			if identities[key] {
				return nil, fmt.Errorf("duplicate test identity %s", key)
			}
			identities[key] = true
		}
		if len(s.Tests) != len(s.sourceByTest) {
			return nil, fmt.Errorf("unregistered case in suite %s", s.ID)
		}
	}
	// Hidden work roots and independent modules are not this module's sources.
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		rel, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") || entry.Name() == "node_modules" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			if _, e = os.Stat(filepath.Join(path, "go.mod")); e == nil {
				return filepath.SkipDir
			}
			return nil
		}
		required := strings.HasSuffix(rel, ".integration.test.mjs")
		if strings.HasSuffix(rel, "_test.go") {
			b, e := regularBytes(path, 4<<20)
			if e != nil {
				return e
			}
			for _, line := range strings.Split(string(b), "\n") {
				if strings.HasPrefix(line, "//go:build ") && strings.Contains(line, "machinery_integration") {
					required = true
					break
				}
				if strings.HasPrefix(line, "package ") {
					break
				}
			}
		}
		if required && !owned[filepath.ToSlash(rel)] {
			return fmt.Errorf("unregistered integration source %s", rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(suites, func(i, j int) bool { return suites[i].ID < suites[j].ID })
	return suites, nil
}

func integrationTagged(body []byte) bool {
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "//go:build ") {
			expr, err := constraint.Parse(line)
			return err == nil && !expr.Eval(func(tag string) bool { return tag != "machinery_integration" }) && expr.Eval(func(tag string) bool {
				return tag == "machinery_integration" || tag == runtime.GOOS || tag == runtime.GOARCH
			})
		}
		if strings.HasPrefix(line, "package ") {
			break
		}
	}
	return false
}

// laneCustody guards every process the lane launches with verified native
// custody, shaped by the accepted custody chain's shipped budgets:
//
//   - A job result is awaited only within the shared cleanup grace
//     (cleanup_ms + 10s, hard-capped), so jobs that legitimately outlive it —
//     suite executions, engine provisioning, image pulls — follow the
//     long-job protocol the custody suite itself uses: the guarded job writes
//     its own bounded stdout/stderr/exit evidence, the Run call proceeds in
//     the background, and completion is observed from that evidence.
//   - Closing any scope arms one broker-wide final-cleanup grace and live
//     jobs are capped per broker, so the lane opens one custody root per
//     guarded run and verifies its terminal close before continuing. Every
//     job is broker-registered before launch, guardian-owned, and terminally
//     killed before reap, or cleanup is honestly reported failed.
//
// Nothing here invents a new process ownership interface.
type laneCustody struct {
	inherited processscope.Scope
	mode      string
	mu        sync.Mutex
	jobs      int
	closed    []error
}

// newLaneCustody validates any inherited custody capability error-first: a
// present-but-malformed or stale capability fails closed instead of being
// silently discarded in favor of unscoped execution.
func newLaneCustody() (*laneCustody, error) {
	if os.Getenv(processscope.EnvCapability) != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		capability, err := processscope.InheritedCapability(ctx)
		if err != nil {
			return nil, fmt.Errorf("inherited custody capability is malformed or stale: %w", err)
		}
		scope, err := processscope.Join(ctx, capability)
		if err != nil {
			return nil, fmt.Errorf("join inherited custody: %w", err)
		}
		return &laneCustody{inherited: scope, mode: "joined"}, nil
	}
	return &laneCustody{mode: "root"}, nil
}

// begin returns the scope for one guarded run: the honored inherited scope
// when the lane itself runs inside outer custody, or a fresh verified root.
func (l *laneCustody) begin(scratch string) (processscope.Scope, error) {
	if l.inherited != nil {
		return l.inherited, nil
	}
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	body, err := regularBytes(self, 256<<20)
	if err != nil {
		return nil, fmt.Errorf("custody helper identity: %w", err)
	}
	sum := sha256.Sum256(body)
	openCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	scope, err := processscope.Open(openCtx, processscope.Options{
		HelperExecutable: self,
		HelperDigest:     hex.EncodeToString(sum[:]),
		ScratchRoot:      scratch,
		Limits:           processscope.Limits{Jobs: 4, WallMS: 3600000, CleanupMS: 30000},
	})
	if err != nil {
		return nil, fmt.Errorf("open native custody: %w", err)
	}
	return scope, nil
}

// end closes one guarded run's root with a bounded, deliberately fresh
// deadline so terminal retirement and reaping are verified even after the
// operating context was cancelled, and records the verified evidence.
func (l *laneCustody) end(scope processscope.Scope) error {
	if scope == l.inherited {
		return nil
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	rep, err := scope.Close(closeCtx)
	verified := err == nil && rep.Status == processscope.StatusCleaned
	for _, j := range rep.Jobs {
		if !j.Registered || !j.Terminated || !j.Reaped {
			verified = false
		}
	}
	l.mu.Lock()
	l.jobs += len(rep.Jobs)
	if !verified {
		l.closed = append(l.closed, fmt.Errorf("guarded root cleanup: %w report=%+v", err, rep))
	}
	l.mu.Unlock()
	if !verified {
		return fmt.Errorf("guarded root cleanup did not verify: %w report=%+v", err, rep)
	}
	return nil
}

// finalize folds the verified custody evidence into the report: it fails the
// lane whenever any guarded root cleanup did not verify, and closes an
// honored inherited scope exactly once at lane end.
func (l *laneCustody) finalize(receipt *custodyReceipt) error {
	l.mu.Lock()
	jobs := l.jobs
	failures := append([]error(nil), l.closed...)
	l.mu.Unlock()
	if l.inherited != nil {
		closeCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		rep, err := l.inherited.Close(closeCtx)
		verified := err == nil && rep.Status == processscope.StatusCleaned
		for _, j := range rep.Jobs {
			if !j.Registered || !j.Terminated || !j.Reaped {
				verified = false
			}
		}
		jobs += len(rep.Jobs)
		if !verified {
			failures = append(failures, fmt.Errorf("inherited custody close: %w report=%+v", err, rep))
		}
	}
	receipt.Mode = l.mode
	receipt.Jobs = jobs
	if len(failures) == 0 {
		receipt.Status = "passed"
		return nil
	}
	receipt.Status = "failed"
	return errors.Join(failures...)
}

// benignRunError reports whether err is the documented result-wait ceiling
// that a healthy long job outlives; it is not a launch failure.
func benignRunError(err error) bool {
	if err == nil {
		return false
	}
	var serr *processscope.Error
	if errors.As(err, &serr) {
		return serr.Code == processscope.CodeCustodyError && strings.Contains(serr.Message, "broker did not deliver a terminal result")
	}
	return false
}

// run executes argv as one guarded custody job. stdout/stderr are captured by
// the job itself into bounded evidence files (the scoped stream pumps do not
// outlive the custody result-wait ceiling), the exit status lands in its own
// evidence file, and the run's custody root is closed with verified terminal
// retirement on completion, deadline, output overflow or cancellation.
func (l *laneCustody) run(ctx context.Context, scratch, dir string, env []string, timeout time.Duration, outLimit, errLimit int, argv ...string) (string, string, error) {
	if env == nil {
		env = ambientEnv()
	}
	if len(argv) == 0 {
		return "", "", fmt.Errorf("guarded job requires an executable")
	}
	exe := argv[0]
	if !filepath.IsAbs(exe) {
		resolved, err := exec.LookPath(exe)
		if err != nil {
			return "", "", fmt.Errorf("guarded job executable %s: %w", exe, err)
		}
		exe = resolved
	}
	digest, err := guardedDigest(exe)
	if err != nil {
		return "", "", err
	}
	scope, err := l.begin(scratch)
	if err != nil {
		return "", "", err
	}
	evidence, err := os.MkdirTemp(scratch, "guarded-")
	if err != nil {
		_ = l.end(scope)
		return "", "", err
	}
	defer func() { _ = l.end(scope) }()
	outPath := filepath.Join(evidence, "stdout")
	errPath := filepath.Join(evidence, "stderr")
	exitPath := filepath.Join(evidence, "exit")
	envPath := filepath.Join(evidence, "env")
	// The declared environment travels in an exactly shell-escaped file the
	// wrapper sources, not in the custody control frame: the frame must stay
	// far below the smallest shipped platform socket buffer so the control
	// write can never split mid-message.
	if err := writeShellEnv(envPath, env); err != nil {
		return "", "", err
	}
	sh := "/bin/sh"
	script := fmt.Sprintf(`. %s; "$@" >%s 2>%s; printf '%%s\n' "$?" >%s`, shellQuote(envPath), shellQuote(outPath), shellQuote(errPath), shellQuote(exitPath))
	// With sh -c, the first parameter after the script is $0; the executable
	// must therefore start at $1 so "$@" expands to the real command.
	wrapperArgs := append([]string{"-c", script, "guarded-job"}, argv...)
	attached, err := scope.Attach(processscope.Command{
		Executable:    sh,
		Args:          wrapperArgs,
		Dir:           dir,
		Env:           []string{"PATH=/usr/bin:/bin"},
		RuntimeDigest: digest,
	})
	if err != nil {
		return "", "", fmt.Errorf("attach guarded custody: %w", err)
	}
	runErr := make(chan error, 1)
	go func() {
		bounded, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		_, rerr := scope.Run(bounded, attached, processscope.Streams{})
		runErr <- rerr
	}()
	deadline := time.Now().Add(timeout)
	tick := time.NewTimer(50 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case rerr := <-runErr:
			// A nil or ceiling error means the guarded job is proceeding
			// normally (short jobs deliver their natural result, long jobs
			// outlive the result-wait ceiling); completion is observed from
			// the evidence files instead. Any other error is a genuine launch
			// or custody failure and aborts fail-closed.
			if rerr != nil && !benignRunError(rerr) {
				return "", "", fmt.Errorf("guarded job failed: %w", rerr)
			}
			runErr = nil
		case <-ctx.Done():
			return "", "", fmt.Errorf("guarded job canceled: %w", ctx.Err())
		case <-tick.C:
			if info, e := os.Lstat(exitPath); e == nil && info.Mode().IsRegular() {
				out, outErr := regularBytes(outPath, int64(outLimit)+1)
				errout, errOutErr := regularBytes(errPath, int64(errLimit)+1)
				statusBytes, exitErr := regularBytes(exitPath, 64)
				status := 0
				if exitErr == nil {
					if n, e := strconv.Atoi(strings.TrimSpace(string(statusBytes))); e == nil {
						status = n
					}
				} else {
					exitErr = nil
				}
				var runErrs []error
				if len(out) > outLimit {
					runErrs = append(runErrs, fmt.Errorf("native output limit %d exceeded", outLimit))
				}
				if len(errout) > errLimit {
					runErrs = append(runErrs, fmt.Errorf("native stderr limit %d exceeded", errLimit))
				}
				if status != 0 {
					runErrs = append(runErrs, fmt.Errorf("guarded job exited with status %d", status))
				}
				return string(out), string(errout), errors.Join(append(runErrs, outErr, errOutErr, exitErr)...)
			}
			for path, limit := range map[string]int{outPath: outLimit, errPath: errLimit} {
				if info, e := os.Lstat(path); e == nil && info.Mode().IsRegular() && info.Size() > int64(limit) {
					return "", "", fmt.Errorf("native output limit %d exceeded", limit)
				}
			}
			if time.Now().After(deadline) {
				return "", "", fmt.Errorf("timeout: guarded job exceeded %s", timeout)
			}
			tick.Reset(50 * time.Millisecond)
		}
	}
}

// guardedDigest binds the sha256 identity of the executable the job will
// run, following launcher symlinks to the real regular file, matching the
// custody runtime-identity convention.
func guardedDigest(exe string) (string, error) {
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("guarded job executable identity %s: %w", exe, err)
	}
	b, err := regularBytes(resolved, 256<<20)
	if err != nil {
		return "", fmt.Errorf("guarded job executable identity %s: %w", exe, err)
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// shellQuote single-quotes a literal path for the guarded wrapper script.
// Evidence paths come from os.MkdirTemp and contain no quote characters.
func shellQuote(path string) string {
	return "'" + path + "'"
}

func toolReceipt(ctx context.Context, custody *laneCustody, scratch, id, arg string) (runtimeReceipt, error) {
	r := runtimeReceipt{ID: id, Status: "failed"}
	path, err := exec.LookPath(id)
	if err != nil {
		return r, fmt.Errorf("%s unavailable: %w", id, err)
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return r, err
	}
	b, err := regularBytes(path, 256<<20)
	if err != nil {
		return r, err
	}
	out, errout, err := custody.run(ctx, scratch, "", nil, 15*time.Second, 1<<20, 1<<20, path, arg)
	if err != nil || errout != "" {
		return r, fmt.Errorf("%s version failed: %w %s", id, err, errout)
	}
	r.Path = path
	r.SHA = fmt.Sprintf("%x", sha256.Sum256(b))
	r.Identity = strings.TrimSpace(out)
	r.Status = "passed"
	return r, nil
}

func provision(ctx context.Context, custody *laneCustody, root, work, cache string, p pins, suites []suite) ([]runtimeReceipt, error) {
	needed := map[string]bool{}
	for _, s := range suites {
		for _, id := range s.Runtimes {
			needed[id] = true
		}
	}
	if p.Version != 1 {
		return nil, fmt.Errorf("unknown pin version")
	}
	var result []runtimeReceipt
	for _, id := range []string{"go", "node", "docker"} {
		if !needed[id] {
			continue
		}
		if id != "docker" {
			arg := "version"
			if id == "node" {
				arg = "--version"
			}
			r, err := toolReceipt(ctx, custody, work, id, arg)
			if err != nil {
				return result, err
			}
			if id == "node" {
				major, err := strconv.Atoi(strings.Split(strings.TrimPrefix(r.Identity, "v"), ".")[0])
				if err != nil || p.Node.Minimum < 22 || major < p.Node.Minimum {
					return result, fmt.Errorf("unsupported Node runtime/pin %s", r.Identity)
				}
			}
			result = append(result, r)
			continue
		}
		if !regexp.MustCompile(`^[a-zA-Z0-9./:_-]+@sha256:[a-f0-9]{64}$`).MatchString(p.Docker.Image) {
			return result, fmt.Errorf("docker image pin must be immutable digest")
		}
		if p.Docker.Platform != "linux/amd64" && p.Docker.Platform != "linux/arm64" {
			return result, fmt.Errorf("unsupported pinned platform %s", p.Docker.Platform)
		}
		if p.Docker.Memory != "128m" || p.Docker.CPUs != "0.5" || p.Docker.PIDs != 32 {
			return result, fmt.Errorf("invalid docker resource pin")
		}
		out, errout, err := custody.run(ctx, work, root, nil, 10*time.Minute, 1<<20, 1<<20, "docker", "pull", "--quiet", "--platform", p.Docker.Platform, p.Docker.Image)
		if err != nil {
			return result, fmt.Errorf("docker pin provisioning: %w %s %s", err, out, errout)
		}
		out, errout, err = custody.run(ctx, work, root, nil, 30*time.Second, 1<<20, 1<<20, "docker", "image", "inspect", "--format", "{{json .RepoDigests}} {{.Os}}/{{.Architecture}}", p.Docker.Image)
		if err != nil || !strings.Contains(out, `"`+p.Docker.Image+`"`) || !strings.HasSuffix(strings.TrimSpace(out), p.Docker.Platform) {
			return result, fmt.Errorf("docker pin/platform verification failed: %w %s %s", err, out, errout)
		}
		result = append(result, runtimeReceipt{ID: id, Status: "passed", Identity: p.Docker.Image + " " + p.Docker.Platform, SHA: strings.Split(p.Docker.Image, "sha256:")[1]})
	}
	if needed["java"] {
		if p.Java.Version != runtimeclosure.PinnedJavaRuntimeVersion || p.Java.File != ".java-runtime-pin" || p.TLC.Version != "v1.7.4" || p.TLC.SHA != tlcSHA {
			return result, fmt.Errorf("formal runtime pin mismatch")
		}
		b, err := regularBytes(filepath.Join(root, p.Java.File), 8192)
		if err != nil {
			return result, fmt.Errorf("java pin: %w", err)
		}
		key := "JAVA_RUNTIME_" + strings.ToUpper(runtime.GOOS) + "_" + strings.ToUpper(runtime.GOARCH) + "_SHA256="
		archive := ""
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, key) {
				archive = strings.TrimPrefix(line, key)
			}
		}
		if len(archive) != 64 || !strings.Contains(string(b), "JAVA_RUNTIME_VERSION="+p.Java.Version+"\n") {
			return result, fmt.Errorf("java archive/version pin missing")
		}
		self, err := os.Executable()
		if err != nil {
			return result, err
		}
		env := environment(map[string]string{"HOME": cache, "XDG_CACHE_HOME": filepath.Join(cache, "cache"), "TMPDIR": work, "TMP": work, "TEMP": work})
		// The helper runs as one guarded job of the lane's custody and opens
		// its own verified custody root for the nested Java identity probes
		// and TLC engine executions: cancelling the lane kills the helper job,
		// and the helper's broker then self-cleans its engine jobs through
		// owner-liveness. Both custody roots are verified before any success.
		out, errout, err := custody.run(ctx, work, root, env, 12*time.Minute, 1<<20, 1<<20, self, "provision-formal", work)
		if err != nil {
			return result, fmt.Errorf("formal provision failed: %w %s %s", err, out, errout)
		}
		var formalReceipts []runtimeReceipt
		if err = json.Unmarshal([]byte(out), &formalReceipts); err != nil || len(formalReceipts) != 2 {
			return result, fmt.Errorf("formal provision receipt: %w", err)
		}
		if formalReceipts[0].ArchiveSHA != archive {
			return result, fmt.Errorf("java archive pin mismatch")
		}
		// Java resolves the macOS /var -> /private/var alias. Keep receipts and
		// test argv rooted in the caller's selected cache spelling, after checking
		// that each resolved runtime actually belongs to that directory.
		realCache, err := filepath.EvalSymlinks(cache)
		if err != nil {
			return result, err
		}
		for i := range formalReceipts {
			realPath, err := filepath.EvalSymlinks(formalReceipts[i].Path)
			if err != nil {
				return result, err
			}
			rel, err := filepath.Rel(realCache, realPath)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return result, fmt.Errorf("provisioned formal path escaped cache")
			}
			formalReceipts[i].Path = filepath.Join(cache, rel)
		}
		result = append(result, formalReceipts...)
	}
	return result, nil
}

// custodyReservedEnv lists the custody transport variables that are never
// inherited implicitly: they are granted only by the authenticated broker at
// launch, or requested explicitly for the provisioning helper.
func custodyReservedEnv() map[string]bool {
	return map[string]bool{
		processscope.EnvCapability:       true,
		processscope.EnvChildRequest:     true,
		processscope.EnvAttachment:       true,
		processscope.EnvInternalCtl:      true,
		processscope.EnvInternalOwner:    true,
		processscope.EnvInternalParent:   true,
		processscope.EnvInternalDeadline: true,
	}
}

// ambientEnv is the explicit environment for tool invocations: the caller's
// ambient environment minus the reserved custody transport variables, which
// scoped execution must never inherit implicitly.
func ambientEnv() []string {
	reserved := custodyReservedEnv()
	var out []string
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if !reserved[key] {
			out = append(out, item)
		}
	}
	return out
}

// environment preserves runtime discovery and explicit network configuration,
// but clears ambient test filters, loader injection, engine overrides, and
// the reserved custody transport variables (those are re-granted only
// deliberately, never inherited).
func environment(overrides map[string]string) []string {
	drop := map[string]bool{"GOFLAGS": true, "GOWORK": true, "NODE_OPTIONS": true, "NODE_TEST_CONTEXT": true, "MACHINERY_JAVA": true, "MACHINERY_JAVA_CLOSURE_SHA256": true, "TLA_TOOLS_JAR": true, "TLA_TOOLS_JAR_SHA256": true, "JAVA_TOOL_OPTIONS": true, "JDK_JAVA_OPTIONS": true, "CLASSPATH": true}
	for key := range custodyReservedEnv() {
		drop[key] = true
	}
	var env []string
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if _, ok := overrides[key]; !ok && !drop[key] && !strings.HasPrefix(key, "MACHINERY_INTEGRATION_") {
			env = append(env, item)
		}
	}
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		env = append(env, key+"="+overrides[key])
	}
	return append(env, "GOFLAGS=", "GOWORK=off")
}

// provisionFormal provisions the pinned Java/TLC closure under the helper's
// own verified native custody. It runs only as a guarded job of the
// integration lane: its Java identity probes and TLC engine executions attach
// to the helper's custody root after environment sanitation, that root must
// close verified before the helper reports success, and the lane's outer
// custody guarantees the helper job's terminal cleanup.
func provisionFormal(work string) (retErr error) {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("provision-formal: %w", err)
	}
	body, err := regularBytes(self, 256<<20)
	if err != nil {
		return fmt.Errorf("provision-formal: custody helper identity: %w", err)
	}
	sum := sha256.Sum256(body)
	scratch := filepath.Join(work, "helper-custody")
	baseCtx, cancel := context.WithTimeout(context.Background(), 11*time.Minute)
	defer cancel()
	scope, err := processscope.Open(baseCtx, processscope.Options{
		HelperExecutable: self,
		HelperDigest:     hex.EncodeToString(sum[:]),
		ScratchRoot:      scratch,
		Limits:           processscope.Limits{Jobs: 4, WallMS: 660000, CleanupMS: 30000},
	})
	if err != nil {
		return fmt.Errorf("provision-formal: open native custody: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		rep, closeErr := scope.Close(closeCtx)
		if closeErr != nil || rep.Status != processscope.StatusCleaned {
			retErr = errors.Join(retErr, fmt.Errorf("provision-formal: custody cleanup did not verify: %w %+v", closeErr, rep))
		}
	}()
	ctx := processcontrol.WithScope(baseCtx, scope)
	java, err := runtimeclosure.OpenJava()
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, java.Close()) }()
	// The identity probe uses its own guarded custody root, opened and
	// verified before the engine verification starts, so no scope channel is
	// closed while the pinned runtime closure is still being provisioned.
	probeCustody, err := newLaneCustody()
	if err != nil {
		return fmt.Errorf("provision-formal: %w", err)
	}
	defer func() { _ = probeCustody.finalize(&custodyReceipt{}) }()
	probeCtx, cancelProbe := context.WithTimeout(baseCtx, time.Minute)
	defer cancelProbe()
	out, errout, err := probeCustody.run(probeCtx, work, work, runtimeclosure.Environment(work, work, java.Path()), 30*time.Second, 1<<20, 1<<20, java.Path(), "-XshowSettings:properties", "-version")
	if err != nil {
		return err
	}
	if err := java.BindIdentity(out + errout); err != nil {
		return err
	}
	design, err := os.MkdirTemp(work, "formal-probe-")
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, os.RemoveAll(design)) }()
	if err := os.Mkdir(filepath.Join(design, "formal"), 0700); err != nil {
		return err
	}
	model := "\\* machinery:manual\n---- MODULE Probe ----\nEXTENDS Integers\nVARIABLE x\nInit == x = 0\nNext == x' = 1 - x\nSpec == Init /\\ [][Next]_x\nSafe == x \\in {0,1}\n====\n"
	if err := os.WriteFile(filepath.Join(design, "formal", "Probe.tla"), []byte(model), 0600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(design, "formal", "Probe.cfg"), []byte("SPECIFICATION Spec\nINVARIANT Safe\n"), 0600); err != nil {
		return err
	}
	if formal.VerifyFormalInScope(ctx, design, false, os.Stderr, os.Stderr) != 0 {
		return fmt.Errorf("pinned TLC provisioning execution failed")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	jar := filepath.Join(cache, "machinery", "tla2tools-v1.7.4.jar")
	jarBytes, err := regularBytes(jar, 128<<20)
	if err != nil {
		return err
	}
	if fmt.Sprintf("%x", sha256.Sum256(jarBytes)) != tlcSHA {
		return fmt.Errorf("TLC pin mismatch")
	}
	javaBytes, err := regularBytes(java.Path(), 16<<20)
	if err != nil {
		return err
	}
	closure, err := runtimeclosure.JavaClosureDigest(java.Path())
	if err != nil {
		return err
	}
	receipt, err := regularBytes(filepath.Join(filepath.Dir(filepath.Dir(java.Path())), ".machinery-java-receipt"), 1024)
	if err != nil {
		return err
	}
	archive := strings.TrimPrefix(strings.Split(string(receipt), "\n")[0], "archive_sha256=")
	if err := java.Validate(); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode([]runtimeReceipt{
		{ID: "java", Identity: runtimeclosure.RequiredJavaRelease, Status: "passed", Path: java.Path(), SHA: fmt.Sprintf("%x", sha256.Sum256(javaBytes)), ClosureSHA: closure, ArchiveSHA: archive},
		{ID: "tlc", Identity: "v1.7.4", Status: "passed", Path: jar, SHA: tlcSHA},
	})
}

func execute(ctx context.Context, custody *laneCustody, root, work, cache, evidence string, s suite, runtimes []runtimeReceipt) (suiteReceipt, error) {
	r := suiteReceipt{ID: s.ID, Adapter: s.Adapter, Selected: len(s.Tests), Tests: []testReceipt{}}
	for _, name := range s.Tests {
		r.Tests = append(r.Tests, testReceipt{Name: name, Source: s.sourceByTest[name], Status: "not_started"})
	}
	overrides := map[string]string{"MACHINERY_INTEGRATION_WORK": work, "MACHINERY_INTEGRATION_RUN_ID": filepath.Base(work), "MACHINERY_INTEGRATION_CACHE": cache}
	paths := map[string]string{}
	for _, runtime := range runtimes {
		paths[runtime.ID] = runtime.Path
		if runtime.ID == "java" {
			overrides["MACHINERY_INTEGRATION_JAVA"] = runtime.Path
		}
		if runtime.ID == "tlc" {
			overrides["MACHINERY_INTEGRATION_TLC_JAR"] = runtime.Path
		}
		if runtime.Path != "" {
			b, err := regularBytes(runtime.Path, 256<<20)
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(b)) != runtime.SHA {
				return r, fmt.Errorf("runtime identity changed: %s: %w", runtime.ID, err)
			}
		}
	}
	env := environment(overrides)
	var argv []string
	packageID := ""
	if s.Adapter == "go-json" {
		out, errout, err := custody.run(ctx, work, root, env, time.Minute, 1<<20, 1<<20, paths["go"], "list", "-json", "-tags", "machinery_integration", s.Package)
		if err != nil || errout != "" {
			return r, fmt.Errorf("native package selection failed: %w %s", err, errout)
		}
		var pkg struct {
			ImportPath                string
			TestGoFiles, XTestGoFiles []string
		}
		if err := json.Unmarshal([]byte(out), &pkg); err != nil {
			return r, err
		}
		if pkg.ImportPath == "" {
			return r, fmt.Errorf("empty native package selection")
		}
		packageID = pkg.ImportPath
		active := map[string]bool{}
		for _, file := range append(pkg.TestGoFiles, pkg.XTestGoFiles...) {
			active[filepath.Join(strings.TrimPrefix(s.Package, "./"), file)] = true
		}
		for _, source := range s.Sources {
			if !active[source] {
				return r, fmt.Errorf("registered source absent from native selection: %s", source)
			}
		}
		names := make([]string, len(s.Tests))
		for i, name := range s.Tests {
			names[i] = regexp.QuoteMeta(name)
		}
		argv = []string{paths["go"], "test", "-json", "-count=1", "-tags", "machinery_integration", "-run", "^(" + strings.Join(names, "|") + ")$", s.Package}
	} else {
		argv = []string{paths["node"], "--test", "--test-reporter=tap"}
		argv = append(argv, s.Sources...)
	}
	duration, err := time.ParseDuration(s.Timeout)
	if err != nil {
		return r, err
	}
	// Go combines the test process streams into Output events. Apply the
	// stricter budget to that combined native stream. The suite execution is
	// a guarded custody job following the long-job protocol: the job writes
	// its own native stream and exit evidence, and the child scope closes
	// with verified terminal cleanup on every outcome.
	limit := s.StdoutLimit
	if s.Adapter == "go-json" {
		limit = min(limit, s.StderrLimit)
	}
	out, errout, runErr := custody.run(ctx, work, root, env, duration, limit, s.StderrLimit, argv...)
	events, err := os.CreateTemp(evidence, s.ID+"-events-*")
	if err != nil {
		return r, errors.Join(runErr, err)
	}
	r.Events = events.Name()
	r.EventsSHA = fmt.Sprintf("%x", sha256.Sum256([]byte(out)))
	_, writeErr := events.WriteString(out)
	closeErr := events.Close()
	if err = errors.Join(writeErr, closeErr); err != nil {
		return r, errors.Join(runErr, err)
	}
	var eventErr error
	if s.Adapter == "go-json" {
		eventErr = accountGo(out, packageID, &r)
	} else {
		eventErr = accountNode(out, &r)
	}
	if errout != "" {
		runErr = errors.Join(runErr, fmt.Errorf("native stderr: %s", errout))
	}
	if runErr != nil {
		runErr = fmt.Errorf("native execution failed: %w", runErr)
	}
	for source, want := range s.sourceHashes {
		path, err := sourcePath(root, source)
		if err != nil {
			eventErr = errors.Join(eventErr, err)
			continue
		}
		b, err := regularBytes(path, 4<<20)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(b)) != want {
			eventErr = errors.Join(eventErr, fmt.Errorf("native source changed during execution: %s: %w", source, err))
		}
	}
	return r, errors.Join(runErr, eventErr)
}

func accountGo(output, packageID string, r *suiteReceipt) error {
	starts, terminals := map[string]int{}, map[string]int{}
	selected := map[string]int{}
	for i, t := range r.Tests {
		selected[t.Name] = i
	}
	packageStarted, packagePassed := false, false
	var failures []error
	for _, line := range strings.Split(strings.TrimSuffix(output, "\n"), "\n") {
		var event struct {
			Action, Package, Test, Output string
			Elapsed                       *float64
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return fmt.Errorf("malformed native execution JSON: %w", err)
		}
		if event.Package != packageID {
			return fmt.Errorf("native execution package %q differs from selected package %q", event.Package, packageID)
		}
		if strings.Contains(event.Output, "(cached)") {
			return fmt.Errorf("cached execution forbidden")
		}
		if event.Test == "" {
			switch event.Action {
			case "start":
				if packageStarted {
					return fmt.Errorf("duplicate package execution")
				}
				packageStarted = true
			case "pass":
				if packagePassed || !packageStarted || event.Elapsed == nil {
					return fmt.Errorf("invalid package execution terminal")
				}
				packagePassed = true
			case "fail", "skip":
				failures = append(failures, fmt.Errorf("package execution %s failed/skip", event.Action))
			case "output":
			default:
				return fmt.Errorf("unknown package execution action %q", event.Action)
			}
			continue
		}
		root, _, _ := strings.Cut(event.Test, "/")
		index, ok := selected[root]
		if !ok {
			return fmt.Errorf("unexpected native execution identity %s", event.Test)
		}
		isRoot := root == event.Test
		switch event.Action {
		case "run":
			starts[event.Test]++
			if starts[event.Test] != 1 {
				return fmt.Errorf("duplicate execution %s", event.Test)
			}
			if isRoot {
				r.Started++
				r.Tests[index].Status = "started"
			}
		case "pass", "fail", "skip":
			terminals[event.Test]++
			if starts[event.Test] != 1 || terminals[event.Test] != 1 {
				return fmt.Errorf("duplicate or missing native execution start/terminal %s", event.Test)
			}
			if event.Action != "pass" {
				failures = append(failures, fmt.Errorf("native test %s %s failed/skip", event.Test, event.Action))
			}
			if isRoot {
				switch event.Action {
				case "pass":
					r.Passed++
					r.Tests[index].Status = "passed"
				case "fail":
					r.Failed++
					r.Tests[index].Status = "failed"
				case "skip":
					r.Skipped++
					r.Tests[index].Status = "skipped"
				}
			}
		case "output", "pause", "cont":
		default:
			return fmt.Errorf("unknown native execution action %q", event.Action)
		}
	}
	for name := range starts {
		if terminals[name] != 1 {
			failures = append(failures, fmt.Errorf("incomplete native execution %s", name))
		}
	}
	if !packageStarted || !packagePassed || r.Started != r.Selected || r.Passed != r.Selected {
		failures = append(failures, fmt.Errorf("required execution incomplete: selected %d started %d passed %d", r.Selected, r.Started, r.Passed))
	}
	return errors.Join(failures...)
}

func accountNode(output string, r *suiteReceipt) error {
	indices := map[string]int{}
	for i, t := range r.Tests {
		indices[t.Name] = i
	}
	starts, terminals := map[string]int{}, map[string]int{}
	var failures []error
	terminalPattern := regexp.MustCompile(`^(ok|not ok) ([0-9]+) - (.*)$`)
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "# Subtest: ") {
			name := strings.TrimPrefix(line, "# Subtest: ")
			index, ok := indices[name]
			if !ok {
				return fmt.Errorf("unexpected Node execution %q", name)
			}
			starts[name]++
			if starts[name] != 1 {
				return fmt.Errorf("duplicate Node execution %s", name)
			}
			r.Started++
			r.Tests[index].Status = "started"
		}
		if match := terminalPattern.FindStringSubmatch(line); match != nil {
			name, _, _ := strings.Cut(match[3], " #")
			index, ok := indices[name]
			if !ok {
				return fmt.Errorf("unexpected Node execution terminal %q", name)
			}
			terminals[name]++
			sequence, err := strconv.Atoi(match[2])
			if err != nil || sequence != len(terminals) {
				return fmt.Errorf("invalid Node execution terminal sequence")
			}
			if terminals[name] != 1 || starts[name] != 1 {
				return fmt.Errorf("duplicate/incomplete Node execution %s", name)
			}
			if strings.Contains(line, "# SKIP") || strings.Contains(line, "# TODO") {
				r.Skipped++
				r.Tests[index].Status = "skipped"
				failures = append(failures, fmt.Errorf("node execution skip %s", name))
			} else if match[1] != "ok" {
				r.Failed++
				r.Tests[index].Status = "failed"
				failures = append(failures, fmt.Errorf("node execution failed %s", name))
			} else {
				r.Passed++
				r.Tests[index].Status = "passed"
			}
		}
	}
	for _, expected := range []string{"TAP version 13\n", fmt.Sprintf("\n1..%d\n", r.Selected), fmt.Sprintf("\n# tests %d\n", r.Selected), fmt.Sprintf("\n# pass %d\n", r.Selected), "\n# fail 0\n", "\n# cancelled 0\n", "\n# skipped 0\n", "\n# todo 0\n"} {
		if strings.Count(output, expected) != 1 {
			failures = append(failures, fmt.Errorf("incomplete Node execution summary %q", expected))
		}
	}
	if r.Started != r.Selected || r.Passed != r.Selected {
		failures = append(failures, fmt.Errorf("node execution incomplete: selected %d started %d passed %d", r.Selected, r.Started, r.Passed))
	}
	return errors.Join(failures...)
}

// cleanupContainers reconciles scoped Docker ownership. Its bounded commands
// run under the lane's custody on fresh deadline contexts so owned containers
// are still reclaimed when the suite context was already cancelled.
func cleanupContainers(custody *laneCustody, work, runID string, p pins, dockerRequired bool) error {
	custodyCtx := func(timeout time.Duration) (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.Background(), timeout)
	}
	dir := filepath.Join(work, "containers")
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var failures []error
	ids := map[string]bool{}
	for _, entry := range entries {
		b, e := regularBytes(filepath.Join(dir, entry.Name()), 128)
		if e != nil {
			failures = append(failures, e)
			continue
		}
		id := strings.TrimSpace(string(b))
		if !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(id) {
			failures = append(failures, fmt.Errorf("cleanup refuses invalid container ID %q", id))
			continue
		}
		ids[id] = true
	}
	// The unique label is a second ownership witness. It finds a container
	// whose test exited between Docker creation and writing its cidfile.
	if dockerRequired {
		ctx, cancel := custodyCtx(15 * time.Second)
		out, errout, e := custody.run(ctx, work, "", nil, 15*time.Second, 1<<20, 1<<20, "docker", "ps", "-aq", "--no-trunc", "--filter", "label="+ownerLabel+"="+runID)
		cancel()
		if e != nil {
			failures = append(failures, fmt.Errorf("cleanup ownership inventory failed: %w %s", e, errout))
		} else {
			for _, id := range strings.Fields(out) {
				if !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(id) {
					failures = append(failures, fmt.Errorf("cleanup invalid labelled container ID"))
					continue
				}
				ids[id] = true
			}
		}
	}
	ordered := make([]string, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	for _, id := range ordered {
		ctx, cancel := custodyCtx(30 * time.Second)
		out, errout, e := custody.run(ctx, work, "", nil, 10*time.Second, 1<<20, 1<<20, "docker", "inspect", "--format", `{{index .Config.Labels "`+ownerLabel+`"}} {{.Config.Image}}`, id)
		if e != nil && strings.Contains(strings.ToLower(out+errout), "no such object") {
			cancel()
			continue
		}
		if e != nil || strings.TrimSpace(out) != runID+" "+p.Docker.Image {
			failures = append(failures, fmt.Errorf("cleanup refuses unverified container %s: %w %s %s", id, e, out, errout))
			cancel()
			continue
		}
		failures = append(failures, fmt.Errorf("owned container leak %s", id))
		out, errout, e = custody.run(ctx, work, "", nil, 10*time.Second, 1<<20, 1<<20, "docker", "rm", "-f", id)
		if e != nil {
			failures = append(failures, fmt.Errorf("cleanup %s: %w %s %s", id, e, out, errout))
		}
		out, errout, e = custody.run(ctx, work, "", nil, 10*time.Second, 1<<20, 1<<20, "docker", "inspect", id)
		if e == nil || !strings.Contains(strings.ToLower(out+errout), "no such object") {
			failures = append(failures, fmt.Errorf("cleanup cannot verify removal %s", id))
		}
		cancel()
	}
	return errors.Join(failures...)
}

// writeShellEnv persists an exact environment as shell-quoted export lines.
// Single-quote escaping preserves every byte of every value (including
// newlines), so sourcing the file restores the declared environment exactly.
func writeShellEnv(path string, env []string) error {
	var b strings.Builder
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || !envKeyPattern.MatchString(key) {
			return fmt.Errorf("guarded job environment entry %q is not exportable", entry)
		}
		b.WriteString("export ")
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteByte('\'')
		b.WriteString(shellEscape(value))
		b.WriteString("'\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0600)
}

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// shellEscape makes value safe inside single quotes in POSIX shell.
func shellEscape(value string) string {
	return strings.ReplaceAll(value, "'", `'\''`)
}
