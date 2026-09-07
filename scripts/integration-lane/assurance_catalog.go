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

	"github.com/RamXX/machinery/internal/processscope"
	"github.com/RamXX/machinery/internal/runtimeclosure"
	"github.com/RamXX/machinery/internal/tdd"
	"github.com/RamXX/machinery/internal/tdd/adapters"
)

// assuranceSchemaSHA binds the closed assurance fragment schema bytes.
const assuranceSchemaSHA = "58d1f97a79570f24b1a382711a57745f130bf8dc3ffd93753d11e83328419bb1"

const assuranceSchemaIdentity = "machinery.assurance.lane/v1"
const assuranceProbeKind = "runtime-probe"
const assuranceConformanceKind = "native-conformance"
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

// assuranceConformanceInventory is the closed per-adapter native-conformance
// case inventory of the downstream adapter stories; a native-conformance
// suite declares exactly its adapter's inventory. MAC-wi2u (Go) and MAC-avfp
// (TypeScript) are the first two entries; the Python/Elixir adapter stories
// plug theirs in when they deliver, and until then those adapters own no
// conformance inventory and no conformance suite is required of them.
var assuranceConformanceInventory = map[string][]string{
	"go-testing/v1": {
		"TestConformanceWitnessPass",
		"TestConformanceSubtestIdentity",
		"TestConformanceSubtestIdentity/first",
		"TestConformanceSubtestIdentity/second",
		"TestConformanceMultipleAssertions",
	},
	"node-test-typescript/v1": {
		"conformance witness executes native assertion",
		"conformance parent identity",
		"conformance parent identity > conformance nested identity",
		"conformance multiple assertions",
	},
	"elixir-exunit/v1": {
		"conformance witness executes native assertion",
		"conformance parent identity conformance nested identity",
		"conformance multiple assertions",
	},
}

// assuranceConformanceAssertions declares the registered machinery-check/v1
// assertion call sites of each closed conformance leaf (empty for inventory
// parents); the exact frozen lines are located in the fixture bytes at
// execution time.
var assuranceConformanceAssertions = map[string][]string{
	"TestConformanceWitnessPass":                                {"conformance/witness-pass"},
	"TestConformanceSubtestIdentity/first":                      {"conformance/subtest-first"},
	"TestConformanceSubtestIdentity/second":                     {"conformance/subtest-second"},
	"TestConformanceMultipleAssertions":                         {"conformance/multi-a", "conformance/multi-b"},
	"conformance witness executes native assertion":             {"conformance/witness"},
	"conformance parent identity > conformance nested identity": {"conformance/nested"},
	"conformance multiple assertions":                           {"conformance/multi-a", "conformance/multi-b"},
	"conformance parent identity conformance nested identity":   {"conformance/nested"},
}

// assuranceConformanceAssetPrefix is the exclusively owned adapter asset
// directory each native-conformance fixture must live under.
var assuranceConformanceAssetPrefix = map[string]string{
	"go-testing/v1":           "internal/tdd/adapters/assets/go/",
	"node-test-typescript/v1": "internal/tdd/adapters/assets/typescript/",
	"elixir-exunit/v1":        "internal/tdd/adapters/assets/elixir/",
}

// assuranceLaneProjectUUID is the fixed project identity of the lane's
// private capture store (a fresh store under the suite scratch root).
const assuranceLaneProjectUUID = "7c1a4f6e-3b25-4d98-9a77-5a2c6f0e31aa"

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
	conformancePresent := map[string]bool{}
	anyConformance := false
	for _, s := range catalog.suites {
		present[s.Adapter] = true
		if s.Kind == assuranceConformanceKind {
			conformancePresent[s.Adapter] = true
			anyConformance = true
		}
		for _, id := range s.Runtimes {
			catalog.neededGit = catalog.neededGit || id == "git"
		}
	}
	for _, id := range assuranceClosedAdapters {
		if !present[id] {
			return nil, fmt.Errorf("assurance union incomplete: missing adapter %s", id)
		}
	}
	// MAC-wi2u/MAC-avfp union reconciliation: the machinery repository itself
	// carries the complete closed catalog, so every adapter owning conformance
	// inventory must register its native-conformance suite there. Foreign
	// fixture roots (each adapter story's RED harness seeds exactly its own
	// fragment) must carry the closed conformance lane — at least one
	// native-conformance suite — with every present suite still validated
	// exactly against its adapter's closed inventory.
	if machineryLaneRoot(root) {
		for _, id := range assuranceClosedAdapters {
			if _, owned := assuranceConformanceInventory[id]; owned && !conformancePresent[id] {
				return nil, fmt.Errorf("assurance union incomplete: adapter %s is missing its required native-conformance suite", id)
			}
		}
	} else if len(assuranceConformanceInventory) > 0 && !anyConformance {
		return nil, fmt.Errorf("assurance union incomplete: closed catalog carries no native-conformance suite")
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
	if s.Kind != assuranceProbeKind && s.Kind != assuranceConformanceKind {
		return fmt.Errorf("unknown assurance suite kind %q", s.Kind)
	}
	if s.Kind == assuranceConformanceKind && len(assuranceConformanceInventory[s.Adapter]) == 0 {
		return fmt.Errorf("adapter %s owns no closed native-conformance inventory", s.Adapter)
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
	if s.Kind == assuranceConformanceKind {
		inventory = assuranceConformanceInventory[s.Adapter]
	}
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
		if s.Kind == assuranceConformanceKind {
			if err := validateConformanceSource(s.Adapter, source); err != nil {
				return err
			}
		} else if err := validateAssuranceSource(s.Adapter, source); err != nil {
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

// validateConformanceSource pins each native-conformance fixture inside its
// adapter's exclusively owned asset directory with the adapter's closed
// fixture grammar (go: the frozen module and test files; node: the frozen
// module identity, the ambient type closure, the helper transport and the
// TypeScript fixture).
func validateConformanceSource(adapter, source string) error {
	base := filepath.Base(source)
	prefix, owned := assuranceConformanceAssetPrefix[adapter]
	if !owned {
		return fmt.Errorf("adapter %s owns no conformance asset directory", adapter)
	}
	if !strings.HasPrefix(source, prefix) {
		return fmt.Errorf("conformance source %s is outside the adapter's owned asset directory %s", source, prefix)
	}
	switch adapter {
	case "go-testing/v1":
		if base != "go.mod" && !strings.HasSuffix(base, "_test.go") {
			return fmt.Errorf("go conformance source %s is not a module or test input", source)
		}
	case "node-test-typescript/v1":
		if base != "package.json" && base != "node-ambient.d.ts" && base != "machinery-check.ts" && !strings.HasSuffix(base, ".ts") {
			return fmt.Errorf("node conformance source %s is not a TypeScript or module input", source)
		}
	case "elixir-exunit/v1":
		if !strings.HasPrefix(source, prefix+"conformance/") || !strings.HasSuffix(base, "_test.exs") {
			return fmt.Errorf("elixir conformance source %s is not a frozen test input of the owned conformance fixture", source)
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

// executeGoConformanceSuite executes the frozen go-testing/v1 conformance
// fixture through the REAL production assurance chain: the frozen bytes are
// captured into a private content-addressed store bundle, the pinned Go
// 1.27.1 runtime closure is opened and validated under the lane's custody
// scope, the closed adapter prepares the suite (verified materialization,
// embedded byte-pinned helper transport, typed assertion call-site
// validation, native build selection proof) and runs the exact native
// invocation, and the normalized machinery.tdd.event/v1 stream is accounted
// exactly. This is adapter conformance evidence, not a mock or a receipt.
func executeGoConformanceSuite(ctx context.Context, custody *laneCustody, root, scratch, evidence string, s assuranceSuite) (suiteReceipt, error) {
	r := suiteReceipt{ID: s.ID, Adapter: s.Adapter, Selected: len(s.Tests), Tests: []testReceipt{}}
	for _, name := range s.Tests {
		r.Tests = append(r.Tests, testReceipt{Name: name, Source: s.Sources[0], Status: "not_started"})
	}
	src := filepath.Join(scratch, "src")
	if err := os.MkdirAll(src, 0o700); err != nil {
		return r, err
	}
	for _, source := range s.Sources {
		path, err := sourcePath(root, source)
		if err != nil {
			return r, err
		}
		b, err := regularBytes(path, 4<<20)
		if err != nil {
			return r, err
		}
		if err := os.WriteFile(filepath.Join(src, filepath.Base(source)), b, 0o600); err != nil {
			return r, err
		}
	}
	testFile := "conformance_test.go"
	testBytes, err := regularBytes(filepath.Join(src, testFile), 4<<20)
	if err != nil {
		return r, err
	}
	modBytes, err := regularBytes(filepath.Join(src, "go.mod"), 1<<20)
	if err != nil {
		return r, err
	}
	control := filepath.Join(scratch, "control")
	if err := os.MkdirAll(control, 0o700); err != nil {
		return r, err
	}
	if err := os.WriteFile(filepath.Join(control, "plan.json"), []byte(`{"schema":"machinery.tdd.plan/v1"}`), 0o600); err != nil {
		return r, err
	}
	scope, err := custody.begin(scratch)
	if err != nil {
		return r, err
	}
	suiteErr := executeGoConformance(ctx, scope, scratch, evidence, s, testBytes, modBytes, testFile, &r)
	closeErr := custody.end(scope)
	if err := errors.Join(suiteErr, closeErr); err != nil {
		return r, err
	}
	for source, want := range s.sourceHashes {
		path, err := sourcePath(root, source)
		if err != nil {
			return r, err
		}
		b, err := regularBytes(path, 4<<20)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(b)) != want {
			return r, fmt.Errorf("conformance source changed during execution: %s: %v", source, err)
		}
	}
	return r, nil
}

// executeGoConformance runs one prepared conformance execution under the
// live lane scope: closure validation, capture, adapter Prepare/Run and the
// pure post-run closure revalidation.
func executeGoConformance(ctx context.Context, scope processscope.Scope, scratch, evidence string, s assuranceSuite, testBytes, modBytes []byte, testFile string, r *suiteReceipt) error {
	handle, err := runtimeclosure.OpenGo(ctx, runtimeclosure.GoRequest{})
	if err != nil {
		return fmt.Errorf("pinned go closure: %w", err)
	}
	if err := handle.Validate(ctx, scope); err != nil {
		_ = handle.Close()
		return fmt.Errorf("pinned go closure validation: %w", err)
	}
	module := goModuleOf(modBytes)
	suite := tdd.Suite{
		ID:      s.ID,
		Adapter: s.Adapter,
		Runtime: handle.Identity(),
		Root:    ".",
		Files:   []string{"go.mod", testFile},
	}
	for _, name := range s.Tests {
		test := tdd.Test{ID: name, Source: testFile, Native: tdd.NativeID{Package: module, Test: name}}
		for _, id := range assuranceConformanceAssertions[name] {
			line, err := conformanceCallLine(testBytes, id)
			if err != nil {
				_ = handle.Close()
				return err
			}
			test.Assertions = append(test.Assertions, tdd.Assertion{ID: id, Source: testFile, Line: line, Helper: tdd.AssertionHelperV1})
		}
		suite.Tests = append(suite.Tests, test)
	}
	frozen := map[string][]byte{"go.mod": modBytes, testFile: testBytes}
	released := false
	inputs := tdd.InputView{
		SourceRoot:  srcRootOf(scratch),
		DesignPath:  ".",
		ControlRoot: filepath.Join(scratch, "control"),
		Revalidate: func() error {
			for name, want := range frozen {
				got, err := os.ReadFile(filepath.Join(srcRootOf(scratch), name))
				if err != nil || fmt.Sprintf("%x", sha256.Sum256(got)) != fmt.Sprintf("%x", sha256.Sum256(want)) {
					return fmt.Errorf("conformance fixture input %s changed", name)
				}
			}
			return nil
		},
		Release: func() error {
			if released {
				return fmt.Errorf("double release")
			}
			released = true
			return nil
		},
	}
	store := filepath.Join(scratch, "store")
	if _, err := tdd.InitStore(ctx, store, assuranceLaneProjectUUID); err != nil {
		_ = handle.Close()
		return fmt.Errorf("conformance capture store: %w", err)
	}
	manifest := tdd.Manifest{
		Schema: tdd.SchemaMilestone, ID: "M1", Revision: 1, Repository: ".",
		ImplementationRoots: []string{"."}, Suites: []tdd.Suite{suite},
	}
	bundle, err := tdd.Capture(ctx, tdd.CaptureRequest{
		Inputs: inputs, Manifest: manifest, Name: "conformance", Store: store,
		Limits: tdd.Limits{WallMS: 300000},
	})
	if err != nil {
		_ = handle.Close()
		return fmt.Errorf("conformance capture: %w", err)
	}
	adapter, err := adapters.Lookup(s.Adapter)
	if err != nil {
		_ = handle.Close()
		return err
	}
	duration, err := time.ParseDuration(s.Timeout)
	if err != nil {
		_ = handle.Close()
		return err
	}
	prepared, err := adapter.Prepare(ctx, tdd.SuiteRequest{
		Inputs: inputs, Suite: suite, Source: bundle,
		Scratch: filepath.Join(scratch, "run"), Runtime: handle, Scope: scope,
		Limits: tdd.Limits{WallMS: duration.Milliseconds(), CleanupMS: 10000},
	})
	if err != nil {
		_ = handle.Close()
		return fmt.Errorf("conformance prepare: %w", err)
	}
	var events []tdd.Event
	execution, runErr := adapter.Run(ctx, prepared, func(e tdd.Event) error {
		events = append(events, e)
		return nil
	})
	closeErr := handle.Close()
	if err := errors.Join(runErr, closeErr); err != nil {
		return fmt.Errorf("native conformance execution failed: %w", err)
	}
	if execution.Outcome != "pass" || execution.ExitCode == nil || *execution.ExitCode != 0 {
		return fmt.Errorf("native conformance outcome %q exit %+v is not a verified pass", execution.Outcome, execution.ExitCode)
	}
	if execution.Custody.Status != "cleaned" {
		return fmt.Errorf("native conformance custody did not verify: %+v", execution.Custody)
	}
	started, passed, assertions := map[string]int{}, map[string]int{}, map[string]int{}
	for _, e := range events {
		switch e.Kind {
		case "test-start":
			if e.Native != nil {
				started[e.Native.Test]++
			}
		case "test-end":
			if e.Native != nil && e.Outcome == "pass" {
				passed[e.Native.Test]++
			}
		case "assertion":
			if e.Outcome == "pass" {
				assertions[e.Assertion]++
			}
		}
	}
	wantAssertions := 0
	for _, name := range s.Tests {
		if started[name] != 1 || passed[name] != 1 {
			return fmt.Errorf("conformance case %s not accounted exactly (started=%d passed=%d)", name, started[name], passed[name])
		}
		for _, id := range assuranceConformanceAssertions[name] {
			wantAssertions++
			if assertions[id] != 1 {
				return fmt.Errorf("conformance assertion %s not witnessed exactly once (%d)", id, assertions[id])
			}
		}
		for i := range r.Tests {
			if r.Tests[i].Name == name {
				r.Tests[i].Status = "passed"
			}
		}
	}
	if wantAssertions == 0 {
		return fmt.Errorf("closed conformance inventory carries no registered assertions")
	}
	r.Started = len(started)
	r.Passed = len(passed)
	var payload []byte
	for _, e := range events {
		body, err := json.Marshal(e)
		if err != nil {
			return err
		}
		payload = append(append(payload, body...), '\n')
	}
	eventsFile, err := os.CreateTemp(evidence, s.ID+"-events-*")
	if err != nil {
		return err
	}
	r.Events = eventsFile.Name()
	r.EventsSHA = fmt.Sprintf("%x", sha256.Sum256(payload))
	_, writeErr := eventsFile.Write(payload)
	closeErr = eventsFile.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	return nil
}

func srcRootOf(scratch string) string { return filepath.Join(scratch, "src") }

func goModuleOf(modBytes []byte) string {
	for _, line := range strings.Split(string(modBytes), "\n") {
		if fields := strings.Fields(strings.TrimSpace(line)); len(fields) == 2 && fields[0] == "module" {
			return fields[1]
		}
	}
	return ""
}

// conformanceCallLine locates the exact line of a registered assertion's
// typed helper call in the frozen fixture bytes. The closed anchor forms are
// the go adapter's machinerycheck.Check(t, ...) and the node adapter's
// check(t, ...) / check(ct, ...) transports; each adapter's frozen fixture
// only ever carries its own form.
func conformanceCallLine(testBytes []byte, id string) (int64, error) {
	for _, prefix := range []string{`machinerycheck.Check(t, "`, `check(t, "`, `check(ct, "`, `Machinery.Check.check(ctx, "`} {
		anchor := prefix + id + `"`
		for i, line := range strings.Split(string(testBytes), "\n") {
			if strings.Contains(line, anchor) {
				return int64(i + 1), nil
			}
		}
	}
	return 0, fmt.Errorf("conformance assertion %s has no typed call site in the frozen fixture", id)
}

// goModulePathOf reads the module path of a materialized fixture module.
func goModulePathOf(dir string) (string, error) {
	raw, err := regularBytes(filepath.Join(dir, "go.mod"), 1<<20)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if fields := strings.Fields(strings.TrimSpace(line)); len(fields) == 2 && fields[0] == "module" {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("conformance fixture go.mod declares no module path")
}

// executeTypeScriptConformanceSuite executes the frozen node-test-typescript
// /v1 conformance fixture through the REAL production assurance chain: the
// frozen bytes are captured into a private content-addressed store bundle,
// the pinned Node 26.8.1 / TypeScript 7.0.2 runtime closure is opened and
// validated under the lane's custody scope, the closed adapter prepares the
// suite (verified materialization, embedded byte-pinned helper transport and
// reporter, typed assertion call-site validation, separate pinned-compiler
// build step) and runs the exact native invocation, and the normalized
// machinery.tdd.event/v1 stream is accounted exactly. This is adapter
// conformance evidence, not a mock or a receipt.
func executeTypeScriptConformanceSuite(ctx context.Context, custody *laneCustody, root, scratch, evidence string, s assuranceSuite) (suiteReceipt, error) {
	r := suiteReceipt{ID: s.ID, Adapter: s.Adapter, Selected: len(s.Tests), Tests: []testReceipt{}}
	for _, name := range s.Tests {
		r.Tests = append(r.Tests, testReceipt{Name: name, Source: s.Sources[0], Status: "not_started"})
	}
	src := filepath.Join(scratch, "src")
	if err := os.MkdirAll(src, 0o700); err != nil {
		return r, err
	}
	for _, source := range s.Sources {
		path, err := sourcePath(root, source)
		if err != nil {
			return r, err
		}
		b, err := regularBytes(path, 4<<20)
		if err != nil {
			return r, err
		}
		if err := os.WriteFile(filepath.Join(src, filepath.Base(source)), b, 0o600); err != nil {
			return r, err
		}
	}
	testFile := "conformance.ts"
	testBytes, err := regularBytes(filepath.Join(src, testFile), 4<<20)
	if err != nil {
		return r, err
	}
	control := filepath.Join(scratch, "control")
	if err := os.MkdirAll(control, 0o700); err != nil {
		return r, err
	}
	if err := os.WriteFile(filepath.Join(control, "plan.json"), []byte(`{"schema":"machinery.tdd.plan/v1"}`), 0o600); err != nil {
		return r, err
	}
	scope, err := custody.begin(scratch)
	if err != nil {
		return r, err
	}
	suiteErr := executeTypeScriptConformance(ctx, scope, scratch, evidence, s, testBytes, testFile, &r)
	closeErr := custody.end(scope)
	if err := errors.Join(suiteErr, closeErr); err != nil {
		return r, err
	}
	for source, want := range s.sourceHashes {
		path, err := sourcePath(root, source)
		if err != nil {
			return r, err
		}
		b, err := regularBytes(path, 4<<20)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(b)) != want {
			return r, fmt.Errorf("conformance source changed during execution: %s: %v", source, err)
		}
	}
	return r, nil
}

// executeTypeScriptConformance runs one prepared conformance execution under
// the live lane scope: closure validation, capture, adapter Prepare/Run and
// the pure post-run closure revalidation.
func executeTypeScriptConformance(ctx context.Context, scope processscope.Scope, scratch, evidence string, s assuranceSuite, testBytes []byte, testFile string, r *suiteReceipt) error {
	handle, err := runtimeclosure.OpenTypeScript(ctx, runtimeclosure.TypeScriptRequest{})
	if err != nil {
		return fmt.Errorf("pinned typescript closure: %w", err)
	}
	if err := handle.Validate(ctx, scope); err != nil {
		_ = handle.Close()
		return fmt.Errorf("pinned typescript closure validation: %w", err)
	}
	suite := tdd.Suite{
		ID:      s.ID,
		Adapter: s.Adapter,
		Runtime: handle.Identity(),
		Root:    ".",
		Files:   []string{testFile},
	}
	for _, name := range s.Tests {
		path := strings.Split(name, " > ")
		test := tdd.Test{ID: name, Source: testFile, Native: tdd.NativeID{Source: testFile, Path: path, Line: conformanceDeclLine(testBytes, path[0]), Column: 1}}
		for _, id := range assuranceConformanceAssertions[name] {
			line, err := conformanceCallLine(testBytes, id)
			if err != nil {
				_ = handle.Close()
				return err
			}
			test.Assertions = append(test.Assertions, tdd.Assertion{ID: id, Source: testFile, Line: line, Helper: tdd.AssertionHelperV1})
		}
		suite.Tests = append(suite.Tests, test)
	}
	frozen := map[string][]byte{testFile: testBytes}
	released := false
	inputs := tdd.InputView{
		SourceRoot:  srcRootOf(scratch),
		DesignPath:  ".",
		ControlRoot: filepath.Join(scratch, "control"),
		Revalidate: func() error {
			for name, want := range frozen {
				got, err := os.ReadFile(filepath.Join(srcRootOf(scratch), name))
				if err != nil || fmt.Sprintf("%x", sha256.Sum256(got)) != fmt.Sprintf("%x", sha256.Sum256(want)) {
					return fmt.Errorf("conformance fixture input %s changed", name)
				}
			}
			return nil
		},
		Release: func() error {
			if released {
				return fmt.Errorf("double release")
			}
			released = true
			return nil
		},
	}
	store := filepath.Join(scratch, "store")
	if _, err := tdd.InitStore(ctx, store, assuranceLaneProjectUUID); err != nil {
		_ = handle.Close()
		return fmt.Errorf("conformance capture store: %w", err)
	}
	manifest := tdd.Manifest{
		Schema: tdd.SchemaMilestone, ID: "M1", Revision: 1, Repository: ".",
		ImplementationRoots: []string{"."}, Suites: []tdd.Suite{suite},
	}
	bundle, err := tdd.Capture(ctx, tdd.CaptureRequest{
		Inputs: inputs, Manifest: manifest, Name: "conformance", Store: store,
		Limits: tdd.Limits{WallMS: 300000},
	})
	if err != nil {
		_ = handle.Close()
		return fmt.Errorf("conformance capture: %w", err)
	}
	adapter, err := adapters.Lookup(s.Adapter)
	if err != nil {
		_ = handle.Close()
		return err
	}
	duration, err := time.ParseDuration(s.Timeout)
	if err != nil {
		_ = handle.Close()
		return err
	}
	prepared, err := adapter.Prepare(ctx, tdd.SuiteRequest{
		Inputs: inputs, Suite: suite, Source: bundle,
		Scratch: filepath.Join(scratch, "run"), Runtime: handle, Scope: scope,
		Limits: tdd.Limits{WallMS: duration.Milliseconds(), CleanupMS: 10000},
	})
	if err != nil {
		_ = handle.Close()
		return fmt.Errorf("conformance prepare: %w", err)
	}
	var events []tdd.Event
	execution, runErr := adapter.Run(ctx, prepared, func(e tdd.Event) error {
		events = append(events, e)
		return nil
	})
	closeErr := handle.Close()
	if err := errors.Join(runErr, closeErr); err != nil {
		return fmt.Errorf("native conformance execution failed: %w", err)
	}
	if execution.Outcome != "pass" || execution.ExitCode == nil || *execution.ExitCode != 0 {
		return fmt.Errorf("native conformance outcome %q exit %+v is not a verified pass", execution.Outcome, execution.ExitCode)
	}
	if execution.Custody.Status != "cleaned" {
		return fmt.Errorf("native conformance custody did not verify: %+v", execution.Custody)
	}
	started, passed, assertions := map[string]int{}, map[string]int{}, map[string]int{}
	for _, e := range events {
		switch e.Kind {
		case "test-start":
			if e.Native != nil {
				started[strings.Join(e.Native.Path, " > ")]++
			}
		case "test-end":
			if e.Native != nil && e.Outcome == "pass" {
				passed[strings.Join(e.Native.Path, " > ")]++
			}
		case "assertion":
			if e.Outcome == "pass" {
				assertions[e.Assertion]++
			}
		}
	}
	wantAssertions := 0
	for _, name := range s.Tests {
		if started[name] != 1 || passed[name] != 1 {
			return fmt.Errorf("conformance case %s not accounted exactly (started=%d passed=%d)", name, started[name], passed[name])
		}
		for _, id := range assuranceConformanceAssertions[name] {
			wantAssertions++
			if assertions[id] != 1 {
				return fmt.Errorf("conformance assertion %s not witnessed exactly once (%d)", id, assertions[id])
			}
		}
		for i := range r.Tests {
			if r.Tests[i].Name == name {
				r.Tests[i].Status = "passed"
			}
		}
	}
	if wantAssertions == 0 {
		return fmt.Errorf("closed conformance inventory carries no registered assertions")
	}
	r.Started = len(started)
	r.Passed = len(passed)
	var payload []byte
	for _, e := range events {
		body, err := json.Marshal(e)
		if err != nil {
			return err
		}
		payload = append(append(payload, body...), '\n')
	}
	eventsFile, err := os.CreateTemp(evidence, s.ID+"-events-*")
	if err != nil {
		return err
	}
	r.Events = eventsFile.Name()
	r.EventsSHA = fmt.Sprintf("%x", sha256.Sum256(payload))
	_, writeErr := eventsFile.Write(payload)
	closeErr = eventsFile.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	return nil
}

// conformanceDeclLine locates a declared case's test( declaration line.
func conformanceDeclLine(testBytes []byte, name string) int64 {
	anchor := `test("` + name + `"`
	for i, line := range strings.Split(string(testBytes), "\n") {
		if strings.Contains(line, anchor) {
			return int64(i + 1)
		}
	}
	return 1
}

// executeElixirConformanceSuite executes the frozen elixir-exunit/v1
// conformance fixture through the REAL production assurance chain: the
// frozen bytes are captured into a private content-addressed store bundle,
// the pinned Elixir 1.20.4 / OTP 29.0.6 (ERTS 17.0.6) runtime closure is
// opened and validated under the lane's custody scope, the closed adapter
// prepares the suite (verified materialization into the embedded harness,
// byte-pinned transport, typed assertion call-site validation) and runs
// the exact native Mix/ExUnit invocation, and the normalized
// machinery.tdd.event/v1 stream is accounted exactly. This is adapter
// conformance evidence, not a mock or a receipt.
func executeElixirConformanceSuite(ctx context.Context, custody *laneCustody, root, scratch, evidence string, s assuranceSuite) (suiteReceipt, error) {
	r := suiteReceipt{ID: s.ID, Adapter: s.Adapter, Selected: len(s.Tests), Tests: []testReceipt{}}
	for _, name := range s.Tests {
		r.Tests = append(r.Tests, testReceipt{Name: name, Source: s.Sources[0], Status: "not_started"})
	}
	src := filepath.Join(scratch, "src")
	if err := os.MkdirAll(filepath.Join(src, "test"), 0o700); err != nil {
		return r, err
	}
	for _, source := range s.Sources {
		path, err := sourcePath(root, source)
		if err != nil {
			return r, err
		}
		b, err := regularBytes(path, 4<<20)
		if err != nil {
			return r, err
		}
		if err := os.WriteFile(filepath.Join(src, "test", filepath.Base(source)), b, 0o600); err != nil {
			return r, err
		}
	}
	testFile := "test/" + filepath.Base(s.Sources[0])
	testBytes, err := regularBytes(filepath.Join(src, filepath.FromSlash(testFile)), 4<<20)
	if err != nil {
		return r, err
	}
	control := filepath.Join(scratch, "control")
	if err := os.MkdirAll(control, 0o700); err != nil {
		return r, err
	}
	if err := os.WriteFile(filepath.Join(control, "plan.json"), []byte(`{"schema":"machinery.tdd.plan/v1"}`), 0o600); err != nil {
		return r, err
	}
	scope, err := custody.begin(scratch)
	if err != nil {
		return r, err
	}
	suiteErr := executeElixirConformance(ctx, scope, scratch, evidence, s, testBytes, testFile, &r)
	closeErr := custody.end(scope)
	if err := errors.Join(suiteErr, closeErr); err != nil {
		return r, err
	}
	for source, want := range s.sourceHashes {
		path, err := sourcePath(root, source)
		if err != nil {
			return r, err
		}
		b, err := regularBytes(path, 4<<20)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(b)) != want {
			return r, fmt.Errorf("conformance source changed during execution: %s: %v", source, err)
		}
	}
	return r, nil
}

// executeElixirConformance runs one prepared conformance execution under the
// live lane scope: closure validation, capture, adapter Prepare/Run and the
// pure post-run closure revalidation.
func executeElixirConformance(ctx context.Context, scope processscope.Scope, scratch, evidence string, s assuranceSuite, testBytes []byte, testFile string, r *suiteReceipt) error {
	handle, err := runtimeclosure.OpenElixir(ctx, runtimeclosure.ElixirRequest{})
	if err != nil {
		return fmt.Errorf("pinned elixir closure: %w", err)
	}
	if err := handle.Validate(ctx, scope); err != nil {
		_ = handle.Close()
		return fmt.Errorf("pinned elixir closure validation: %w", err)
	}
	suite := tdd.Suite{
		ID:      s.ID,
		Adapter: s.Adapter,
		Runtime: handle.Identity(),
		Root:    ".",
		Files:   []string{testFile},
	}
	for _, name := range s.Tests {
		test := tdd.Test{ID: name, Source: testFile, Native: tdd.NativeID{Module: "Machinery.ConformanceTest", Name: name, File: testFile, Line: elixirConformanceDeclLine(testBytes, name)}}
		for _, id := range assuranceConformanceAssertions[name] {
			line, err := conformanceCallLine(testBytes, id)
			if err != nil {
				_ = handle.Close()
				return err
			}
			test.Assertions = append(test.Assertions, tdd.Assertion{ID: id, Source: testFile, Line: line, Helper: tdd.AssertionHelperV1})
		}
		suite.Tests = append(suite.Tests, test)
	}
	frozen := map[string][]byte{testFile: testBytes}
	released := false
	inputs := tdd.InputView{
		SourceRoot:  srcRootOf(scratch),
		DesignPath:  ".",
		ControlRoot: filepath.Join(scratch, "control"),
		Revalidate: func() error {
			for name, want := range frozen {
				got, err := os.ReadFile(filepath.Join(srcRootOf(scratch), filepath.FromSlash(name)))
				if err != nil || fmt.Sprintf("%x", sha256.Sum256(got)) != fmt.Sprintf("%x", sha256.Sum256(want)) {
					return fmt.Errorf("conformance fixture input %s changed", name)
				}
			}
			return nil
		},
		Release: func() error {
			if released {
				return fmt.Errorf("double release")
			}
			released = true
			return nil
		},
	}
	store := filepath.Join(scratch, "store")
	if _, err := tdd.InitStore(ctx, store, assuranceLaneProjectUUID); err != nil {
		_ = handle.Close()
		return fmt.Errorf("conformance capture store: %w", err)
	}
	manifest := tdd.Manifest{
		Schema: tdd.SchemaMilestone, ID: "M1", Revision: 1, Repository: ".",
		ImplementationRoots: []string{"."}, Suites: []tdd.Suite{suite},
	}
	bundle, err := tdd.Capture(ctx, tdd.CaptureRequest{
		Inputs: inputs, Manifest: manifest, Name: "conformance", Store: store,
		Limits: tdd.Limits{WallMS: 300000},
	})
	if err != nil {
		_ = handle.Close()
		return fmt.Errorf("conformance capture: %w", err)
	}
	adapter, err := adapters.Lookup(s.Adapter)
	if err != nil {
		_ = handle.Close()
		return err
	}
	duration, err := time.ParseDuration(s.Timeout)
	if err != nil {
		_ = handle.Close()
		return err
	}
	prepared, err := adapter.Prepare(ctx, tdd.SuiteRequest{
		Inputs: inputs, Suite: suite, Source: bundle,
		Scratch: filepath.Join(scratch, "run"), Runtime: handle, Scope: scope,
		Limits: tdd.Limits{WallMS: duration.Milliseconds(), CleanupMS: 10000},
	})
	if err != nil {
		_ = handle.Close()
		return fmt.Errorf("conformance prepare: %w", err)
	}
	var events []tdd.Event
	execution, runErr := adapter.Run(ctx, prepared, func(e tdd.Event) error {
		events = append(events, e)
		return nil
	})
	closeErr := handle.Close()
	if err := errors.Join(runErr, closeErr); err != nil {
		return fmt.Errorf("native conformance execution failed: %w", err)
	}
	if execution.Outcome != "pass" || execution.ExitCode == nil || *execution.ExitCode != 0 {
		return fmt.Errorf("native conformance outcome %q exit %+v is not a verified pass", execution.Outcome, execution.ExitCode)
	}
	if execution.Custody.Status != "cleaned" {
		return fmt.Errorf("native conformance custody did not verify: %+v", execution.Custody)
	}
	started, passed, assertions := map[string]int{}, map[string]int{}, map[string]int{}
	for _, e := range events {
		switch e.Kind {
		case "test-start":
			if e.Native != nil {
				started[e.Native.Name]++
			}
		case "test-end":
			if e.Native != nil && e.Outcome == "pass" {
				passed[e.Native.Name]++
			}
		case "assertion":
			if e.Outcome == "pass" {
				assertions[e.Assertion]++
			}
		}
	}
	wantAssertions := 0
	for _, name := range s.Tests {
		if started[name] != 1 || passed[name] != 1 {
			return fmt.Errorf("conformance case %s not accounted exactly (started=%d passed=%d)", name, started[name], passed[name])
		}
		for _, id := range assuranceConformanceAssertions[name] {
			wantAssertions++
			if assertions[id] != 1 {
				return fmt.Errorf("conformance assertion %s not witnessed exactly once (%d)", id, assertions[id])
			}
		}
		for i := range r.Tests {
			if r.Tests[i].Name == name {
				r.Tests[i].Status = "passed"
			}
		}
	}
	if wantAssertions == 0 {
		return fmt.Errorf("closed conformance inventory carries no registered assertions")
	}
	r.Started = len(started)
	r.Passed = len(passed)
	var payload []byte
	for _, e := range events {
		body, err := json.Marshal(e)
		if err != nil {
			return err
		}
		payload = append(append(payload, body...), '\n')
	}
	eventsFile, err := os.CreateTemp(evidence, s.ID+"-events-*")
	if err != nil {
		return err
	}
	r.Events = eventsFile.Name()
	r.EventsSHA = fmt.Sprintf("%x", sha256.Sum256(payload))
	_, writeErr := eventsFile.Write(payload)
	closeErr = eventsFile.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	return nil
}

var elixirTestDeclPattern = regexp.MustCompile(`^\s*test\s+"([^"]+)"`)
var elixirDescribePattern = regexp.MustCompile(`^\s*describe\s+"([^"]+)"`)

// elixirConformanceDeclLine locates a declared case's native test
// declaration line: the declared name is either the plain test text or the
// describe-prefixed exact native name atom ExUnit reports.
func elixirConformanceDeclLine(testBytes []byte, name string) int64 {
	var describes []string
	for _, line := range strings.Split(string(testBytes), "\n") {
		if match := elixirDescribePattern.FindStringSubmatch(line); match != nil {
			describes = append(describes, match[1])
		}
	}
	for i, line := range strings.Split(string(testBytes), "\n") {
		match := elixirTestDeclPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		if match[1] == name {
			return int64(i + 1)
		}
		for _, describe := range describes {
			if describe+" "+match[1] == name {
				return int64(i + 1)
			}
		}
	}
	return 1
}
