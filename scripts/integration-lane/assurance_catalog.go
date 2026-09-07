// The four-language native assurance conformance catalog of the required
// integration lane (MAC-bz1y). This is the compatible registry extension the
// downstream adapter stories (MAC-wi2u Go, MAC-avfp TypeScript, MAC-imtz
// Python, MAC-8yai Elixir) plug their named fragments into: it validates the
// closed assurance fragment union against the exact first-release runtime
// catalog, provisions and verifies all four native language runtimes before
// any assurance suite runs, and executes the frozen per-language probe
// fixtures as real native invocations under the lane's process custody. It
// does not implement adapter semantics (assertion transports, strict helper
// witnesses, closure digests); those remain the adapter stories' fragments.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// assuranceSchemaSHA binds the closed assurance fragment schema bytes.
const assuranceSchemaSHA = "58d1f97a79570f24b1a382711a57745f130bf8dc3ffd93753d11e83328419bb1"

const assuranceSchemaIdentity = "machinery.assurance.lane/v1"
const assuranceProbeKind = "runtime-probe"
const machineryModulePath = "github.com/RamXX/machinery"

// The exact first-release native catalog of docs/test-assurance-contract.md
// section 6. Git is a gate runtime, not a fifth adapter.
const (
	assuranceGoVersion         = "1.27.1"
	assuranceNodeVersion       = "26.8.1"
	assuranceTypeScriptVersion = "7.0.2"
	assurancePythonVersion     = "3.14.7"
	assuranceElixirVersion     = "1.20.4"
	assuranceOTPVersion        = "29"
	assuranceErtsVersion       = "17.0.6"
	assuranceMixVersion        = "1.20.4"
	assuranceGitVersion        = "2.55.0"
)

// assuranceClosedAdapters is the closed four-language union in canonical
// order; the catalog fails unless every one owns at least one suite.
var assuranceClosedAdapters = []string{
	"go-testing/v1", "node-test-typescript/v1", "python-unittest/v1", "elixir-exunit/v1",
}

// assuranceAdapterRuntimes is each adapter's required runtime closure.
var assuranceAdapterRuntimes = map[string][]string{
	"go-testing/v1":           {"go"},
	"node-test-typescript/v1": {"node", "tsc"},
	"python-unittest/v1":      {"python"},
	"elixir-exunit/v1":        {"elixir"},
}

// assuranceProbeInventory is the closed native case inventory the frozen
// probe fixtures actually execute; a runtime-probe suite declares exactly it.
var assuranceProbeInventory = map[string][]string{
	"go-testing/v1":           {"TestAssuranceRuntimeProbe"},
	"node-test-typescript/v1": {"assurance runtime probe executes a native TypeScript closure"},
	"python-unittest/v1":      {"probe_test.AssuranceRuntimeProbe.test_assurance_runtime_probe"},
	"elixir-exunit/v1":        {"assurance runtime probe executes a native ExUnit closure"},
}

var assuranceOwnerPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]*-[a-z0-9]{4}$`)
var assuranceOTPIdentity = regexp.MustCompile(`Erlang/OTP ` + assuranceOTPVersion + `([^\d.]|$)`)

type assuranceSuite struct {
	ID           string   `json:"id"`
	Lane         string   `json:"lane"`
	Adapter      string   `json:"adapter"`
	Kind         string   `json:"kind"`
	Package      string   `json:"package"`
	Sources      []string `json:"source_files"`
	Tests        []string `json:"tests"`
	Runtimes     []string `json:"runtimes"`
	Timeout      string   `json:"timeout"`
	StdoutLimit  int      `json:"stdout_limit"`
	StderrLimit  int      `json:"stderr_limit"`
	sourceHashes map[string]string
}

type assuranceFragmentFile struct {
	Schema   string           `json:"schema"`
	Fragment string           `json:"fragment"`
	Owner    string           `json:"owner"`
	Suites   []assuranceSuite `json:"suites"`
}

type assuranceAdapterPin struct {
	Runtime    string `json:"runtime"`
	Version    string `json:"version"`
	TypeScript string `json:"typescript"`
	OTP        string `json:"otp"`
	Erts       string `json:"erts"`
	Mix        string `json:"mix"`
}

type assurancePins struct {
	Version   int                            `json:"version"`
	Platforms []string                       `json:"platforms"`
	Adapters  map[string]assuranceAdapterPin `json:"adapters"`
	Git       struct {
		Version string `json:"version"`
	} `json:"git"`
}

type assuranceCatalog struct {
	suites    []assuranceSuite
	neededGit bool
}

type adapterReceipt struct {
	ID     string `json:"id"`
	Suites int    `json:"suites"`
	Status string `json:"status"`
}

type assuranceReport struct {
	Schema   string           `json:"schema"`
	Status   string           `json:"status"`
	Platform string           `json:"platform"`
	Adapters []adapterReceipt `json:"adapters"`
	Runtimes []runtimeReceipt `json:"runtimes"`
	Suites   []suiteReceipt   `json:"suites"`
}

// machineryLaneRoot reports whether a lane root is the machinery repository
// itself, whose required lane always carries the assurance catalog.
func machineryLaneRoot(root string) bool {
	b, err := regularBytes(filepath.Join(root, "go.mod"), 4096)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(b), "\n") {
		if fields := strings.Fields(line); len(fields) == 2 && fields[0] == "module" {
			return fields[1] == machineryModulePath
		}
	}
	return false
}

// loadAssuranceCatalog validates the closed assurance catalog of a lane root.
// It returns nil (v1-only semantics preserved exactly) for a foreign root
// without any assurance inventory, and fails closed when the machinery
// repository's required catalog is absent or malformed.
func loadAssuranceCatalog(root string, v1 []suite) (*assuranceCatalog, error) {
	dir := filepath.Join(root, "testdata", "integration-lanes")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var fragments []string
	hasSchema, hasPins, hasProbes := false, false, false
	for _, entry := range entries {
		name := entry.Name()
		switch {
		case name == "assurance.schema.json":
			hasSchema = true
		case name == "assurance-runtime-pins.json":
			hasPins = true
		case name == "assurance.CONTRACT.md":
		case name == "assurance-probes":
			if !entry.IsDir() {
				return nil, fmt.Errorf("assurance-probes must be the probe fixture directory")
			}
			hasProbes = true
		case strings.HasPrefix(name, "assurance-") && strings.HasSuffix(name, ".json"):
			fragments = append(fragments, name)
		case strings.HasPrefix(name, "assurance"):
			return nil, fmt.Errorf("unknown assurance inventory member %s", name)
		}
	}
	if !hasSchema && !hasPins && len(fragments) == 0 && !hasProbes {
		if machineryLaneRoot(root) {
			return nil, fmt.Errorf("assurance catalog absent from required machinery lane")
		}
		return nil, nil
	}
	if !hasSchema || !hasPins {
		return nil, fmt.Errorf("assurance catalog incomplete: schema=%v pins=%v", hasSchema, hasPins)
	}
	schema, err := regularBytes(filepath.Join(dir, "assurance.schema.json"), 1<<20)
	if err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(schema)) != assuranceSchemaSHA {
		return nil, fmt.Errorf("assurance schema does not match supported closed version 1")
	}
	var pins assurancePins
	if err := decodeFile(filepath.Join(dir, "assurance-runtime-pins.json"), &pins); err != nil {
		return nil, fmt.Errorf("assurance pins: %w", err)
	}
	if err := validateAssurancePins(pins); err != nil {
		return nil, err
	}
	platform := runtime.GOOS + "/" + runtime.GOARCH
	pinnedPlatform := false
	for _, candidate := range pins.Platforms {
		pinnedPlatform = pinnedPlatform || candidate == platform
	}
	if !pinnedPlatform {
		return nil, fmt.Errorf("UNSUPPORTED_PLATFORM: %s is not a pinned native assurance platform", platform)
	}
	sort.Strings(fragments)
	catalog := &assuranceCatalog{suites: []assuranceSuite{}}
	ids := map[string]bool{}
	for _, s := range v1 {
		ids[s.ID] = true
	}
	native := map[string]bool{}
	owned := map[string]bool{}
	for _, name := range fragments {
		var fragment assuranceFragmentFile
		if err := decodeFile(filepath.Join(dir, name), &fragment); err != nil {
			return nil, fmt.Errorf("assurance fragment %s: %w", name, err)
		}
		if fragment.Schema != assuranceSchemaIdentity {
			return nil, fmt.Errorf("assurance fragment %s: unknown schema %q", name, fragment.Schema)
		}
		if stem := strings.TrimSuffix(name, ".json"); fragment.Fragment != stem {
			return nil, fmt.Errorf("assurance fragment %s: fragment identity %q does not name its file", name, fragment.Fragment)
		}
		if !assuranceOwnerPattern.MatchString(fragment.Owner) {
			return nil, fmt.Errorf("assurance fragment %s: invalid owner %q", name, fragment.Owner)
		}
		if len(fragment.Suites) == 0 {
			return nil, fmt.Errorf("assurance fragment %s: empty fragment", name)
		}
		for i := range fragment.Suites {
			if err := validateAssuranceSuite(root, &fragment.Suites[i], ids, native, owned); err != nil {
				return nil, fmt.Errorf("assurance fragment %s: %w", name, err)
			}
			catalog.suites = append(catalog.suites, fragment.Suites[i])
		}
	}
	present := map[string]bool{}
	for _, s := range catalog.suites {
		present[s.Adapter] = true
		for _, id := range s.Runtimes {
			catalog.neededGit = catalog.neededGit || id == "git"
		}
	}
	for _, id := range assuranceClosedAdapters {
		if !present[id] {
			return nil, fmt.Errorf("assurance union incomplete: missing adapter %s", id)
		}
	}
	if hasProbes {
		probeRoot := filepath.Join(dir, "assurance-probes")
		err := filepath.WalkDir(probeRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			key := filepath.ToSlash(rel)
			if !owned[key] {
				return fmt.Errorf("unregistered assurance source %s", key)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(catalog.suites, func(i, j int) bool { return catalog.suites[i].ID < catalog.suites[j].ID })
	return catalog, nil
}

func validateAssurancePins(p assurancePins) error {
	if p.Version != 1 {
		return fmt.Errorf("assurance pin mismatch: version %d", p.Version)
	}
	if len(p.Platforms) != 2 || p.Platforms[0] != "darwin/arm64" || p.Platforms[1] != "linux/amd64" {
		return fmt.Errorf("assurance pin mismatch: platforms %v", p.Platforms)
	}
	want := map[string]assuranceAdapterPin{
		"go-testing/v1":           {Runtime: "go", Version: assuranceGoVersion},
		"node-test-typescript/v1": {Runtime: "node", Version: assuranceNodeVersion, TypeScript: assuranceTypeScriptVersion},
		"python-unittest/v1":      {Runtime: "python", Version: assurancePythonVersion},
		"elixir-exunit/v1":        {Runtime: "elixir", Version: assuranceElixirVersion, OTP: assuranceOTPVersion, Erts: assuranceErtsVersion, Mix: assuranceMixVersion},
	}
	if len(p.Adapters) != len(want) {
		return fmt.Errorf("assurance pin mismatch: adapter set has %d entries", len(p.Adapters))
	}
	for id, pin := range want {
		got, ok := p.Adapters[id]
		if !ok || got != pin {
			return fmt.Errorf("assurance pin mismatch: adapter %s is %+v", id, got)
		}
	}
	if p.Git.Version != assuranceGitVersion {
		return fmt.Errorf("assurance pin mismatch: git gate runtime %q", p.Git.Version)
	}
	return nil
}

func validateAssuranceSuite(root string, s *assuranceSuite, ids, native, owned map[string]bool) error {
	if !suitePattern.MatchString(s.ID) || s.Lane != "assurance" {
		return fmt.Errorf("invalid assurance suite ID/lane %s", s.ID)
	}
	if ids[s.ID] {
		return fmt.Errorf("duplicate assurance suite id %s", s.ID)
	}
	ids[s.ID] = true
	closedAdapter := false
	for _, id := range assuranceClosedAdapters {
		closedAdapter = closedAdapter || s.Adapter == id
	}
	if !closedAdapter {
		return fmt.Errorf("unknown assurance adapter %s", s.Adapter)
	}
	if s.Kind != assuranceProbeKind {
		return fmt.Errorf("unknown assurance suite kind %q", s.Kind)
	}
	pkg := strings.TrimPrefix(s.Package, "./")
	if s.Package != "." && (!strings.HasPrefix(s.Package, "./") || !pathPattern.MatchString(pkg) || filepath.Clean(pkg) != pkg || strings.Contains(pkg, "..")) {
		return fmt.Errorf("invalid assurance package %q", s.Package)
	}
	if len(s.Tests) == 0 {
		return fmt.Errorf("empty assurance case inventory in suite %s", s.ID)
	}
	if err := errors.Join(unique(s.Sources, "assurance source"), unique(s.Tests, "assurance test"), unique(s.Runtimes, "assurance runtime")); err != nil {
		return err
	}
	duration, err := time.ParseDuration(s.Timeout)
	if err != nil || !timeoutPattern.MatchString(s.Timeout) || duration <= 0 || duration > 30*time.Minute {
		return fmt.Errorf("invalid assurance timeout limit %s", s.Timeout)
	}
	if s.StdoutLimit < 1 || s.StderrLimit < 1 || s.StdoutLimit > maxOutput || s.StderrLimit > maxOutput {
		return fmt.Errorf("invalid assurance output limit")
	}
	required := append([]string(nil), assuranceAdapterRuntimes[s.Adapter]...)
	sort.Strings(required)
	declared := append([]string(nil), s.Runtimes...)
	sort.Strings(declared)
	if strings.Join(declared, ",") != strings.Join(required, ",") {
		return fmt.Errorf("assurance suite %s runtimes %v do not close over adapter %s", s.ID, s.Runtimes, s.Adapter)
	}
	inventory := assuranceProbeInventory[s.Adapter]
	known := map[string]bool{}
	for _, name := range inventory {
		known[name] = true
	}
	for _, name := range s.Tests {
		if !known[name] {
			return fmt.Errorf("registered probe case %s is absent from the closed probe inventory of %s", name, s.Adapter)
		}
	}
	if len(s.Tests) != len(inventory) {
		return fmt.Errorf("closed probe inventory of %s is not fully registered in suite %s", s.Adapter, s.ID)
	}
	s.sourceHashes = map[string]string{}
	for _, source := range s.Sources {
		if owned[source] {
			return fmt.Errorf("duplicate assurance source identity %s", source)
		}
		owned[source] = true
		path, err := sourcePath(root, source)
		if err != nil {
			return err
		}
		b, err := regularBytes(path, 4<<20)
		if err != nil {
			return err
		}
		if err := validateAssuranceSource(s.Adapter, source); err != nil {
			return err
		}
		s.sourceHashes[source] = fmt.Sprintf("%x", sha256.Sum256(b))
	}
	for _, name := range s.Tests {
		key := s.Adapter + ":" + name
		if native[key] {
			return fmt.Errorf("duplicate assurance native identity %s", key)
		}
		native[key] = true
	}
	return nil
}

func validateAssuranceSource(adapter, source string) error {
	base := filepath.Base(source)
	switch adapter {
	case "go-testing/v1":
		if !strings.HasSuffix(base, "_test.go") {
			return fmt.Errorf("go probe source %s is not a test file", source)
		}
	case "node-test-typescript/v1":
		if base != "package.json" && !strings.HasSuffix(base, ".ts") {
			return fmt.Errorf("node probe source %s is not a TypeScript or package input", source)
		}
	case "python-unittest/v1":
		if !strings.HasSuffix(base, ".py") {
			return fmt.Errorf("python probe source %s is not a Python module", source)
		}
	case "elixir-exunit/v1":
		prefix := "testdata/integration-lanes/assurance-probes/elixir/"
		if !strings.HasPrefix(source, prefix) {
			return fmt.Errorf("elixir probe source %s is outside the mix project fixture", source)
		}
	}
	return nil
}

// provisionAssurance verifies every pinned adapter runtime natively before
// any assurance suite runs: identity comes from the runtime's own version
// invocation, executable bytes are bound by sha256, and nothing is fetched
// or installed during replay. Git 2.55.0 is verified only when a suite
// declares the gate runtime.
func provisionAssurance(ctx context.Context, custody *laneCustody, work string, catalog *assuranceCatalog) ([]runtimeReceipt, error) {
	type runtimeProbe struct {
		receipt  string
		exe      string
		arg      string
		pin      string
		identity func(string) bool
	}
	probes := []runtimeProbe{
		{"go", "go", "version", "go" + assuranceGoVersion, func(v string) bool {
			return strings.Contains(" "+v+" ", " go"+assuranceGoVersion+" ")
		}},
		{"node", "node", "--version", "v" + assuranceNodeVersion, func(v string) bool { return v == "v"+assuranceNodeVersion }},
		{"tsc", "tsc", "--version", "Version " + assuranceTypeScriptVersion, func(v string) bool {
			return v == "Version "+assuranceTypeScriptVersion
		}},
		{"python", "python3", "--version", "Python " + assurancePythonVersion, func(v string) bool {
			return v == "Python "+assurancePythonVersion
		}},
		{"elixir", "elixir", "--version", "Elixir " + assuranceElixirVersion + " / OTP " + assuranceOTPVersion + " / erts-" + assuranceErtsVersion, func(v string) bool {
			return strings.Contains(v, "Elixir "+assuranceElixirVersion) &&
				strings.Contains(v, "erts-"+assuranceErtsVersion) &&
				assuranceOTPIdentity.MatchString(v)
		}},
		{"mix", "mix", "--version", "Mix " + assuranceMixVersion, func(v string) bool {
			return strings.Contains(v, "Mix "+assuranceMixVersion)
		}},
		{"git", "git", "--version", "git version " + assuranceGitVersion, func(v string) bool {
			return v == "git version "+assuranceGitVersion
		}},
	}
	var receipts []runtimeReceipt
	for _, probe := range probes {
		if probe.receipt == "git" && !catalog.neededGit {
			continue
		}
		receipt, err := toolReceipt(ctx, custody, work, probe.exe, probe.arg)
		receipt.ID = probe.receipt
		if err != nil {
			return receipts, fmt.Errorf("assurance runtime %s: %w", probe.receipt, err)
		}
		if !probe.identity(receipt.Identity) {
			return receipts, fmt.Errorf("unsupported %s runtime/pin %q (pinned %s)", probe.receipt, receipt.Identity, probe.pin)
		}
		receipts = append(receipts, receipt)
	}
	return receipts, nil
}

// executeAssuranceSuite materializes the frozen probe fixture bytes into a
// private scratch module and executes the adapter's real native invocation
// as a guarded custody job of the lane, accounting exact per-case outcomes
// from the native event stream itself.
func executeAssuranceSuite(ctx context.Context, custody *laneCustody, root, scratch, evidence string, s assuranceSuite, paths map[string]string) (suiteReceipt, error) {
	r := suiteReceipt{ID: s.ID, Adapter: s.Adapter, Selected: len(s.Tests), Tests: []testReceipt{}}
	for _, name := range s.Tests {
		r.Tests = append(r.Tests, testReceipt{Name: name, Source: s.Sources[0], Status: "not_started"})
	}
	if err := materializeProbe(root, scratch, s); err != nil {
		return r, err
	}
	duration, err := time.ParseDuration(s.Timeout)
	if err != nil {
		return r, err
	}
	var out, evidenceStream string
	var runErr, eventErr error
	switch s.Adapter {
	case "go-testing/v1":
		listOut, listErr, err := custody.run(ctx, scratch, scratch, environment(nil), time.Minute, 1<<20, 1<<20, paths["go"], "list", "-json", s.Package)
		if err != nil || listErr != "" {
			return r, fmt.Errorf("native probe selection failed: %w %s", err, listErr)
		}
		var pkg struct {
			ImportPath                string
			TestGoFiles, XTestGoFiles []string
		}
		if err := json.Unmarshal([]byte(listOut), &pkg); err != nil {
			return r, err
		}
		if pkg.ImportPath == "" {
			return r, fmt.Errorf("empty native probe selection")
		}
		base := filepath.Base(s.Sources[0])
		active := false
		for _, file := range append(pkg.TestGoFiles, pkg.XTestGoFiles...) {
			active = active || file == base
		}
		if !active {
			return r, fmt.Errorf("probe source absent from native selection: %s", base)
		}
		out, _, runErr = custody.run(ctx, scratch, scratch, environment(nil), duration, min(s.StdoutLimit, s.StderrLimit), s.StderrLimit, paths["go"], "test", "-json", "-count=1", s.Package)
		evidenceStream = out
		eventErr = accountGo(out, pkg.ImportPath, &r)
	case "node-test-typescript/v1":
		tsArgs := []string{"--strict", "--module", "nodenext", "--moduleResolution", "nodenext", "--target", "es2023", "--outDir", "out"}
		entry := ""
		for _, source := range s.Sources {
			base := filepath.Base(source)
			if strings.HasSuffix(base, ".ts") {
				tsArgs = append(tsArgs, base)
				if !strings.HasSuffix(base, ".d.ts") {
					entry = base
				}
			}
		}
		if entry == "" {
			return r, fmt.Errorf("node probe lacks a TypeScript entry")
		}
		compileOut, compileErr, err := custody.run(ctx, scratch, scratch, environment(nil), 5*time.Minute, 1<<20, 1<<20, append([]string{paths["tsc"]}, tsArgs...)...)
		if err != nil {
			return r, fmt.Errorf("pinned TypeScript compilation failed: %w %s %s", err, compileOut, compileErr)
		}
		compiled := strings.TrimSuffix(entry, ".ts") + ".js"
		out, _, runErr = custody.run(ctx, scratch, scratch, environment(nil), duration, s.StdoutLimit, s.StderrLimit, paths["node"], "--test", "--test-reporter=tap", filepath.Join("out", compiled))
		evidenceStream = compileOut + "\n" + out
		eventErr = accountNode(out, &r)
	case "python-unittest/v1":
		// unittest writes its verbose per-case results and summary to the
		// native stderr channel; that stream is the accounted evidence.
		var pythonErr string
		_, pythonErr, runErr = custody.run(ctx, scratch, scratch, environment(nil), duration, s.StdoutLimit, s.StderrLimit, paths["python"], "-I", "-m", "unittest", "discover", "-v", "-s", ".", "-p", filepath.Base(s.Sources[0]))
		evidenceStream = pythonErr
		eventErr = accountPython(pythonErr, &r)
	case "elixir-exunit/v1":
		home := filepath.Join(scratch, "home")
		env := environment(map[string]string{
			"HOME": home, "MIX_HOME": filepath.Join(home, ".mix"),
			"MIX_BUILD_PATH": filepath.Join(home, "_build"), "MIX_ENV": "test",
		})
		out, _, runErr = custody.run(ctx, scratch, scratch, env, duration, s.StdoutLimit, s.StderrLimit, paths["mix"], "test", "--trace", "--seed", "0", "--max-cases", "1")
		evidenceStream = out
		eventErr = accountElixir(out, &r)
	default:
		return r, fmt.Errorf("unknown assurance adapter %s", s.Adapter)
	}
	events, err := os.CreateTemp(evidence, s.ID+"-events-*")
	if err != nil {
		return r, errors.Join(runErr, err)
	}
	r.Events = events.Name()
	r.EventsSHA = fmt.Sprintf("%x", sha256.Sum256([]byte(evidenceStream)))
	_, writeErr := events.WriteString(evidenceStream)
	closeErr := events.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return r, errors.Join(runErr, err)
	}
	if runErr != nil {
		runErr = fmt.Errorf("native assurance execution failed: %w", runErr)
	}
	for source, want := range s.sourceHashes {
		path, err := sourcePath(root, source)
		if err != nil {
			eventErr = errors.Join(eventErr, err)
			continue
		}
		b, err := regularBytes(path, 4<<20)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(b)) != want {
			eventErr = errors.Join(eventErr, fmt.Errorf("probe source changed during execution: %s: %w", source, err))
		}
	}
	return r, errors.Join(runErr, eventErr)
}

// materializeProbe copies the exact frozen probe fixture bytes into the
// private scratch layout each native runtime executes, adding only generated
// scaffolding (the go.mod/native.go of the tiny probe module); probe bytes
// themselves are never generated.
func materializeProbe(root, scratch string, s assuranceSuite) error {
	writeFile := func(rel, target string) error {
		path, err := sourcePath(root, rel)
		if err != nil {
			return err
		}
		b, err := regularBytes(path, 4<<20)
		if err != nil {
			return err
		}
		destination := filepath.Join(scratch, filepath.FromSlash(target))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return err
		}
		return os.WriteFile(destination, b, 0o600)
	}
	switch s.Adapter {
	case "go-testing/v1":
		if err := os.MkdirAll(filepath.Join(scratch, "probe"), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(scratch, "go.mod"), []byte("module assurance.probe/lane\n\ngo 1.27.0\n"), 0o600); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(scratch, "probe", "native.go"), []byte("package probe\n"), 0o600); err != nil {
			return err
		}
		return writeFile(s.Sources[0], "probe/"+filepath.Base(s.Sources[0]))
	case "node-test-typescript/v1":
		for _, source := range s.Sources {
			if err := writeFile(source, filepath.Base(source)); err != nil {
				return err
			}
		}
		return nil
	case "python-unittest/v1":
		return writeFile(s.Sources[0], filepath.Base(s.Sources[0]))
	case "elixir-exunit/v1":
		prefix := "testdata/integration-lanes/assurance-probes/elixir/"
		for _, source := range s.Sources {
			if err := writeFile(source, strings.TrimPrefix(source, prefix)); err != nil {
				return err
			}
		}
		return os.MkdirAll(filepath.Join(scratch, "home"), 0o700)
	}
	return fmt.Errorf("unknown assurance adapter %s", s.Adapter)
}

var pythonCaseLine = regexp.MustCompile(`^(\S+) \(([^()]+)\) \.\.\. (.+)$`)
var pythonRanLine = regexp.MustCompile(`^Ran (\d+) tests? in `)

// accountPython reconciles real `python -m unittest -v` output against the
// declared full TestCase.id() inventory: skipped, error, expected-failure
// and unexpected-success outcomes all fail closed.
func accountPython(output string, r *suiteReceipt) error {
	indices := map[string]int{}
	for i, t := range r.Tests {
		indices[t.Name] = i
	}
	terminals := map[string]int{}
	var failures []error
	ran, sawOK := -1, false
	for _, line := range strings.Split(output, "\n") {
		if match := pythonCaseLine.FindStringSubmatch(line); match != nil {
			index, ok := indices[match[2]]
			if !ok {
				return fmt.Errorf("unexpected python native identity %s", match[2])
			}
			terminals[match[2]]++
			if terminals[match[2]] != 1 {
				return fmt.Errorf("duplicate python native execution %s", match[2])
			}
			r.Started++
			r.Tests[index].Status = "passed"
			switch {
			case match[3] == "ok":
				r.Passed++
			case strings.Contains(match[3], "skip"):
				r.Skipped++
				r.Tests[index].Status = "skipped"
				failures = append(failures, fmt.Errorf("python native skip %s", match[2]))
			default:
				r.Failed++
				r.Tests[index].Status = "failed"
				failures = append(failures, fmt.Errorf("python native failure %s: %s", match[2], match[3]))
			}
			continue
		}
		if match := pythonRanLine.FindStringSubmatch(line); match != nil {
			ran, _ = strconv.Atoi(match[1])
		}
		if line == "OK" || strings.HasPrefix(line, "OK ") {
			sawOK = true
		}
		if strings.HasPrefix(line, "FAILED") {
			failures = append(failures, fmt.Errorf("python native summary reported failure: %s", line))
		}
	}
	if ran != r.Selected {
		failures = append(failures, fmt.Errorf("python execution incomplete: selected %d ran %d", r.Selected, ran))
	}
	if !sawOK {
		failures = append(failures, fmt.Errorf("python execution missing OK summary"))
	}
	if r.Started != r.Selected || r.Passed != r.Selected {
		failures = append(failures, fmt.Errorf("python execution incomplete: selected %d started %d passed %d", r.Selected, r.Started, r.Passed))
	}
	return errors.Join(failures...)
}

var exunitDoneLine = regexp.MustCompile(`^\s*\* test (.+) \((?:[0-9.]+ms|skipped)\) \[L#\d+\]$`)
var exunitStartLine = regexp.MustCompile(`^\s*\* test (.+) \[L#\d+\]$`)
var exunitFailureLine = regexp.MustCompile(`^\s+\d+\) test (.+) \([^)]+\)$`)
var exunitResultLine = regexp.MustCompile(`^Result: (\d+) passed$`)

// accountElixir reconciles real `mix test --trace` output against the
// declared native test names: skips, failures, excluded or truncated
// summaries all fail closed.
func accountElixir(output string, r *suiteReceipt) error {
	indices := map[string]int{}
	for i, t := range r.Tests {
		indices[t.Name] = i
	}
	starts, terminals := map[string]int{}, map[string]int{}
	failed := map[string]bool{}
	var failures []error
	passedSummary := -1
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r", "\n"), "\n") {
		if match := exunitDoneLine.FindStringSubmatch(line); match != nil {
			name := match[1]
			index, ok := indices[name]
			if !ok {
				return fmt.Errorf("unexpected exunit native identity %q", name)
			}
			terminals[name]++
			if terminals[name] != 1 {
				return fmt.Errorf("duplicate exunit native execution %q", name)
			}
			r.Started++
			r.Tests[index].Status = "passed"
			if strings.Contains(match[0], "(skipped)") {
				r.Skipped++
				r.Tests[index].Status = "skipped"
				failures = append(failures, fmt.Errorf("exunit native skip %q", name))
			} else {
				r.Passed++
			}
			continue
		}
		if match := exunitStartLine.FindStringSubmatch(line); match != nil {
			if _, ok := indices[match[1]]; !ok {
				return fmt.Errorf("unexpected exunit native identity %q", match[1])
			}
			starts[match[1]]++
			if starts[match[1]] != 1 {
				return fmt.Errorf("duplicate exunit native execution %q", match[1])
			}
			continue
		}
		if match := exunitFailureLine.FindStringSubmatch(line); match != nil {
			name := match[1]
			if _, ok := indices[name]; !ok {
				return fmt.Errorf("unexpected exunit failure identity %q", name)
			}
			failed[name] = true
			continue
		}
		if match := exunitResultLine.FindStringSubmatch(line); match != nil {
			passedSummary, _ = strconv.Atoi(match[1])
			continue
		}
		if strings.HasPrefix(line, "Failed: ") {
			failures = append(failures, fmt.Errorf("exunit reported failed tests: %s", line))
		}
	}
	for i, t := range r.Tests {
		if terminals[t.Name] != 1 || starts[t.Name] != 1 {
			failures = append(failures, fmt.Errorf("incomplete exunit native execution %q", t.Name))
		}
		if failed[t.Name] {
			r.Passed--
			r.Failed++
			r.Tests[i].Status = "failed"
			failures = append(failures, fmt.Errorf("exunit native failure %q", t.Name))
		}
	}
	if passedSummary != r.Selected {
		failures = append(failures, fmt.Errorf("exunit summary mismatch: selected %d summary %d", r.Selected, passedSummary))
	}
	if r.Started != r.Selected || r.Passed != r.Selected {
		failures = append(failures, fmt.Errorf("exunit execution incomplete: selected %d started %d passed %d", r.Selected, r.Started, r.Passed))
	}
	return errors.Join(failures...)
}
