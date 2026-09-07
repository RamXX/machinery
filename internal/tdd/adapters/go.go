// Package adapters owns the closed first-release native adapters of
// docs/test-assurance-contract.md section 6. This file is the go-testing/v1
// adapter (MAC-wi2u): it embeds the byte-pinned machinery-check/v1 helper
// transport from the owned assets directory, prepares captured Go suites
// (verified bundle materialization, typed assertion call-site validation,
// native build selection proof, exact frozen argv) and executes them under
// processscope custody through the pinned Go 1.27.1 runtime closure,
// normalizing native events to machinery.tdd.event/v1.
package adapters

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/processscope"
	"github.com/RamXX/machinery/internal/runtimeclosure"
	"github.com/RamXX/machinery/internal/tdd"
	"github.com/RamXX/machinery/internal/tdd/protocol"
)

//go:embed assets/go/machinerycheck/machinerycheck.go
var goHelperSource []byte

// Closed constants of the go-testing/v1 adapter.
const (
	// AdapterGoTesting is the closed adapter identity.
	AdapterGoTesting = protocol.AdapterGoTesting

	// GoEventSchema is the normalized event identity of every emitted event.
	GoEventSchema = "machinery.tdd.event/v1"

	// GoHelperPinnedSHA256 pins the exact embedded helper bytes; the pin is
	// verified at materialization and re-verified after every native run.
	GoHelperPinnedSHA256 = "5b0b1c48f6a5dd95d257b499d6b76fc22d97048f4746c5cc35862287e35bef53"

	// GoConformanceTestSHA256 and GoConformanceGoModSHA256 pin the frozen
	// native conformance fixture bytes executed by the required contributor
	// lane fragment testdata/integration-lanes/assurance-go.json.
	GoConformanceTestSHA256  = "1900b2ae10201d956c20f20242dace862baff4a537b200d87b8a4fc3fbbfb1d1"
	GoConformanceGoModSHA256 = "60ac10e392e35c417b5f78480803124a01750e634c9ef1494c85798322d7a734"

	// GoHelperDir is the directory the helper is materialized into inside
	// every prepared module; frozen suite sources import it as
	// <module-path>/machinerycheck.
	GoHelperDir = "machinerycheck"

	goRequiredVersion = "1.27.1"
	goListTimeoutMS   = int64(120000)
	goCompileTimeout  = int64(300000)
)

var (
	goAssertionIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)
	goEnvNamePattern     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	goImportPathPattern  = regexp.MustCompile(`^[A-Za-z0-9._\-~/][A-Za-z0-9._\-~/]*$`)
)

// GoAdapter is the closed go-testing/v1 Adapter.
type GoAdapter struct{}

// Compile-time contract check of the closed Adapter boundary.
var _ tdd.Adapter = (*GoAdapter)(nil)

// Go returns the go-testing/v1 adapter.
func Go() *GoAdapter { return &GoAdapter{} }

// Lookup resolves one closed adapter identity. MAC-wi2u registered the go
// adapter, MAC-avfp the TypeScript adapter and MAC-imtz the Python adapter
// through this same registry seam; the union resolves the closed
// first-release identities delivered so far.
func Lookup(id string) (tdd.Adapter, error) {
	switch id {
	case AdapterGoTesting:
		return Go(), nil
	case AdapterNodeTestTS:
		return TypeScript(), nil
	case AdapterPythonUnittest:
		return Python(), nil
	}
	return nil, fmt.Errorf("UNSUPPORTED_ADAPTER: %q is not a closed first-release native adapter", id)
}

// ID implements tdd.Adapter.
func (a *GoAdapter) ID() string { return AdapterGoTesting }

// HelperSource returns a copy of the embedded byte-pinned helper transport.
func HelperSource() []byte { return append([]byte(nil), goHelperSource...) }

// goPrepared is the adapter-owned opaque prepared state: exact argv, closed
// environment, native inventory and the digests late mutation is checked
// against. Only this adapter's Run accepts it.
type goPrepared struct {
	suiteID     string
	goBin       string
	goroot      string
	closure     string
	moduleDir   string
	declared    []tdd.Test
	packages    []string
	argv        []string
	env         []string
	timeoutMS   int64
	limits      processscope.Limits
	scope       processscope.Scope
	fileDigests map[string]string
}

// Prepare implements tdd.Adapter: it validates the closed request, opens and
// validates the pinned Go 1.27.1 runtime closure under the live scope,
// materializes and re-verifies the captured source bundle plus the embedded
// helper transport, validates every registered assertion as a typed helper
// call at its exact frozen line, proves the build selection of every suite
// source, compiles the selected test binaries and freezes the exact argv.
func (a *GoAdapter) Prepare(ctx context.Context, req tdd.SuiteRequest) (tdd.PreparedSuite, error) {
	if req.Suite.Adapter != AdapterGoTesting {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_ADAPTER: suite %s declares adapter %q, not %s", req.Suite.ID, req.Suite.Adapter, AdapterGoTesting)
	}
	if req.Scope == nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s requires a live processscope scope", req.Suite.ID)
	}
	if req.Runtime == nil {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_VERSION: suite %s requires the pinned Go %s runtime handle", req.Suite.ID, goRequiredVersion)
	}
	handle, ok := req.Runtime.(*runtimeclosure.Go)
	if !ok {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_VERSION: suite %s requires the exact approved Go runtime closure handle, got %T", req.Suite.ID, req.Runtime)
	}
	// Reject absent/mismatched/mutated runtime before any suite work.
	if err := handle.Validate(ctx, req.Scope); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_VERSION: suite %s runtime validation failed: %w", req.Suite.ID, err)
	}
	identity := handle.Identity()
	if req.Suite.Runtime != identity {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_VERSION: suite %s declares runtime %+v, but the pinned closure is %+v", req.Suite.ID, req.Suite.Runtime, identity)
	}
	if req.Suite.Root != protocol.RepositoryRoot {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_FEATURE: go-testing/v1 suites are module-root suites; suite %s declares root %q", req.Suite.ID, req.Suite.Root)
	}
	if len(req.Suite.DependencyRoots) != 0 {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_FEATURE: go-testing/v1 resolves the complete local module closure from the captured bundle; suite %s declares dependency roots %v", req.Suite.ID, req.Suite.DependencyRoots)
	}
	if len(req.Suite.Files) == 0 || len(req.Suite.Tests) == 0 {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s declares no files or no tests", req.Suite.ID)
	}
	if req.Scratch == "" {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s requires a private scratch root", req.Suite.ID)
	}
	if req.Inputs.Revalidate == nil || req.Inputs.Release == nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s requires held input view callbacks", req.Suite.ID)
	}
	if err := req.Inputs.Revalidate(); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("STALE_INPUT: the held input view of suite %s failed revalidation: %v", req.Suite.ID, err)
	}
	for _, test := range req.Suite.Tests {
		if !goImportPathPattern.MatchString(test.Native.Package) || strings.HasPrefix(test.Native.Package, "-") {
			return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: test %s declares invalid native package %q", test.ID, test.Native.Package)
		}
	}
	limits, err := processscope.NormalizeLimits(processscope.Limits(req.Limits))
	if err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s limits: %v", req.Suite.ID, err)
	}
	for _, dir := range []string{"home", "tmp", "gocache", "gomodcache", "gopath", "build"} {
		if err := os.MkdirAll(filepath.Join(req.Scratch, dir), 0o700); err != nil {
			return tdd.PreparedSuite{}, fmt.Errorf("CUSTODY_ERROR: preparing private %s root: %w", dir, err)
		}
	}
	// Verified bundle materialization: only a capture-produced BundleRef
	// carries a materialized store object root.
	objectRoot := req.Source.Materialized()
	if objectRoot == "" {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s source bundle was not produced by capture", req.Suite.ID)
	}
	storeRoot := filepath.Dir(filepath.Dir(objectRoot))
	projectID, err := goStoreProjectID(storeRoot)
	if err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("STALE_INPUT: suite %s store identity: %v", req.Suite.ID, err)
	}
	moduleDir := filepath.Join(req.Scratch, "module")
	if err := tdd.MaterializeBundle(ctx, storeRoot, projectID, req.Source.Ref(), moduleDir); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("STALE_INPUT: materializing the captured bundle of suite %s: %w", req.Suite.ID, err)
	}
	// Byte-pinned helper transport materialization with round-trip check.
	helperPath := filepath.Join(moduleDir, GoHelperDir, "machinerycheck.go")
	if err := os.MkdirAll(filepath.Dir(helperPath), 0o755); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("CUSTODY_ERROR: materializing the helper transport: %w", err)
	}
	if err := os.WriteFile(helperPath, goHelperSource, 0o644); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("CUSTODY_ERROR: writing the helper transport: %w", err)
	}
	if readback, err := os.ReadFile(helperPath); err != nil || !bytes.Equal(readback, goHelperSource) {
		return tdd.PreparedSuite{}, fmt.Errorf("CUSTODY_ERROR: the materialized helper transport does not round-trip its pinned bytes: %v", err)
	}
	if sum := fmt.Sprintf("%x", sha256.Sum256(goHelperSource)); sum != GoHelperPinnedSHA256 {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: the embedded helper transport does not match its frozen pin sha256:%s", sum)
	}
	modulePath, err := goModulePath(moduleDir)
	if err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s module: %v", req.Suite.ID, err)
	}
	if err := validateGoSuiteSources(moduleDir, modulePath, &req.Suite); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("suite %s frozen sources: %w", req.Suite.ID, err)
	}
	env, err := buildGoEnv(req.Scratch, goRootOf(handle.Binary()), req.Suite.Environment)
	if err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s environment: %v", req.Suite.ID, err)
	}
	// Build-selection proof: every declared package must exist and every
	// declared test source must be part of the native test compilation.
	packages := goSuitePackages(&req.Suite)
	for _, pkg := range packages {
		if err := goProveBuildSelection(ctx, req.Scope, handle, moduleDir, env, pkg, &req.Suite); err != nil {
			return tdd.PreparedSuite{}, err
		}
	}
	pattern, err := buildGoRunPattern(req.Suite.Tests)
	if err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("suite %s selection: %w", req.Suite.ID, err)
	}
	timeoutMS := limits.WallMS - 3000
	if timeoutMS < 1000 {
		timeoutMS = 1000
	}
	argv := append([]string{
		"test", "-json", "-count=1", "-shuffle=off",
		"-timeout", fmt.Sprintf("%dms", timeoutMS),
		"-run", pattern,
	}, packages...)
	seenAssertion := map[string]bool{}
	for i := range req.Suite.Tests {
		for j := range req.Suite.Tests[i].Assertions {
			id := req.Suite.Tests[i].Assertions[j].ID
			if seenAssertion[id] {
				return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: assertion id %s is registered more than once in suite %s", id, req.Suite.ID)
			}
			seenAssertion[id] = true
		}
	}
	digests := map[string]string{}
	for _, name := range append(append([]string{}, req.Suite.Files...), path.Join(GoHelperDir, "machinerycheck.go")) {
		sum, err := goHashFile(filepath.Join(moduleDir, filepath.FromSlash(name)))
		if err != nil {
			return tdd.PreparedSuite{}, fmt.Errorf("STALE_INPUT: hashing prepared source %s: %w", name, err)
		}
		digests[name] = sum
	}
	return tdd.NewPreparedSuite(&goPrepared{
		suiteID: req.Suite.ID, goBin: handle.Binary(), goroot: goRootOf(handle.Binary()),
		closure: identity.Closure, moduleDir: moduleDir, declared: append([]tdd.Test{}, req.Suite.Tests...),
		packages: packages, argv: argv, env: env, timeoutMS: timeoutMS,
		limits: limits, scope: req.Scope, fileDigests: digests,
	}), nil
}

// Run implements tdd.Adapter: it executes the frozen argv of the prepared
// suite as one guarded custody job, normalizes the native stream to
// machinery.tdd.event/v1, reconciles the complete lifecycle against the
// frozen inventory with the registered assertion witnesses, enforces exit
// concordance and re-verifies the prepared sources (late mutation).
func (a *GoAdapter) Run(ctx context.Context, prepared tdd.PreparedSuite, sink tdd.EventSink) (tdd.Execution, error) {
	state, ok := prepared.PreparedState().(*goPrepared)
	if !ok || state == nil {
		return tdd.Execution{}, fmt.Errorf("INVALID_SCHEMA: run requires a prepared suite produced by this adapter's Prepare")
	}
	execution := tdd.Execution{Suite: state.suiteID}
	if err := ctx.Err(); err != nil {
		return execution, fmt.Errorf("CUSTODY_ERROR: run canceled before launch: %w", err)
	}
	var stdout, stderr bytes.Buffer
	result, err := goGuardedChild(ctx, state.scope, processscope.Command{
		Executable:    state.goBin,
		Args:          state.argv,
		Dir:           state.moduleDir,
		Env:           state.env,
		RuntimeDigest: state.closure,
		DeadlineMS:    state.timeoutMS,
	}, &stdout, &stderr, state.limits.StdoutBytes, state.limits.StderrBytes)
	if err != nil {
		return execution, fmt.Errorf("CUSTODY_ERROR: native go test invocation failed under custody: %w (stderr: %s)", err, boundedString(stderr.String(), 2048))
	}
	if int64(stdout.Len()) > state.limits.StdoutBytes || int64(stderr.Len()) > state.limits.StderrBytes {
		return execution, fmt.Errorf("OUTPUT_LIMIT: native stream exceeded its bound (stdout %d, stderr %d)", stdout.Len(), stderr.Len())
	}
	execution.StdoutDigest = "sha256:" + fmt.Sprintf("%x", sha256.Sum256(stdout.Bytes()))
	execution.StderrDigest = "sha256:" + fmt.Sprintf("%x", sha256.Sum256(stderr.Bytes()))
	exit := int64(result.ExitCode)
	execution.ExitCode = &exit
	// Late source mutation is checked before reconciling the evidence
	// stream so a mutated module cannot hide behind stream diagnostics.
	if err := state.verifyUnchanged(); err != nil {
		execution.Outcome = "error"
		return execution, err
	}
	streamEvents, parseErr := parseGoTestStream(stdout.Bytes())
	var rec goReconciliation
	var reconcileErr error
	if parseErr == nil {
		rec, reconcileErr = reconcileGoStream(streamEvents, &tdd.Suite{ID: state.suiteID, Tests: state.declared})
	} else {
		reconcileErr = parseErr
	}
	if reconcileErr != nil {
		execution.Events = rec.Events
		execution.Outcome = "error"
		a.emit(sink, rec.Events)
		return execution, fmt.Errorf("suite %s native stream: %w", state.suiteID, reconcileErr)
	}
	if !result.Completed || result.Signal != "" {
		execution.Outcome = "error"
		a.emit(sink, rec.Events)
		return execution, fmt.Errorf("CUSTODY_ERROR: native go test invocation did not complete cleanly (completed=%v signal=%q exit=%d)", result.Completed, result.Signal, result.ExitCode)
	}
	switch {
	case rec.Outcome == "pass" && result.ExitCode == 0:
	case rec.Outcome == "assertion-fail" && result.ExitCode == 1:
	default:
		execution.Outcome = "error"
		a.emit(sink, rec.Events)
		return execution, fmt.Errorf("UNEXPECTED_FAILURE: suite %s exit status %d does not concord with reconciled outcome %q", state.suiteID, result.ExitCode, rec.Outcome)
	}
	execution.Events = rec.Events
	execution.Outcome = rec.Outcome
	execution.Custody = tdd.CustodyReport{Status: processscope.StatusCleaned}
	a.emit(sink, rec.Events)
	return execution, nil
}

func (a *GoAdapter) emit(sink tdd.EventSink, events []tdd.Event) {
	if sink == nil {
		return
	}
	for _, e := range events {
		if err := sink(e); err != nil {
			return
		}
	}
}

func boundedGoString(s string, n int) string { return boundedString(s, n) }

func boundedString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(bounded)"
}

func goRootOf(goBinary string) string { return filepath.Dir(filepath.Dir(goBinary)) }

// verifyUnchanged re-hashes the prepared sources after the native run.
func (p *goPrepared) verifyUnchanged() error {
	for name, want := range p.fileDigests {
		got, err := goHashFile(filepath.Join(p.moduleDir, filepath.FromSlash(name)))
		if err != nil || got != want {
			return fmt.Errorf("STALE_INPUT: prepared source %s changed after preparation (want sha256:%s, got %s: %v)", name, want, got, err)
		}
	}
	return nil
}

func goHashFile(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(body)), nil
}

// goStoreProjectID reads the closed store identity of a capture store.
func goStoreProjectID(storeRoot string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(storeRoot, "store.json"))
	if err != nil || len(raw) > 1<<20 {
		return "", fmt.Errorf("cannot read the store identity: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", err
	}
	if len(doc) != 3 || doc["schema"] != "machinery.tdd.store/v1" {
		return "", fmt.Errorf("store.json is not the closed store identity")
	}
	id, _ := doc["project_id"].(string)
	if id == "" {
		return "", fmt.Errorf("store.json lacks the project identity")
	}
	return id, nil
}

// goModulePath reads the module path of a materialized module.
func goModulePath(moduleDir string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(moduleDir, "go.mod"))
	if err != nil || len(raw) > 1<<20 {
		return "", fmt.Errorf("cannot read go.mod: %v", err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("go.mod declares no module path")
}

// goSuitePackages returns the ordered unique declared package identities.
func goSuitePackages(suite *tdd.Suite) []string {
	seen := map[string]bool{}
	var packages []string
	for _, test := range suite.Tests {
		if test.Native.Package == "" || seen[test.Native.Package] {
			continue
		}
		seen[test.Native.Package] = true
		packages = append(packages, test.Native.Package)
	}
	sort.Strings(packages)
	return packages
}

// goProveBuildSelection runs the real `go list -json -test` selection and
// verifies every declared source of the package is part of the native test
// compilation, then compiles the selected test binary (`go test -c`).
func goProveBuildSelection(ctx context.Context, scope processscope.Scope, handle *runtimeclosure.Go, moduleDir string, env []string, pkg string, suite *tdd.Suite) error {
	listOut, err := goRunTool(ctx, scope, handle, moduleDir, env, goListTimeoutMS, "list", "-json", "-test", pkg)
	if err != nil {
		return fmt.Errorf("BUILD_ERROR: suite %s package %s selection failed: %v", suite.ID, pkg, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(listOut))
	found := false
	for decoder.More() {
		var listed struct {
			ImportPath   string
			TestGoFiles  []string
			XTestGoFiles []string
		}
		if err := decoder.Decode(&listed); err != nil {
			return fmt.Errorf("BUILD_ERROR: package %s list output is malformed: %v", pkg, err)
		}
		if listed.ImportPath != pkg {
			continue
		}
		found = true
		sources := map[string]bool{}
		for _, f := range listed.TestGoFiles {
			sources[f] = true
		}
		for _, f := range listed.XTestGoFiles {
			sources[f] = true
		}
		for _, test := range suite.Tests {
			if test.Native.Package != pkg {
				continue
			}
			base := path.Base(test.Source)
			if !sources[base] {
				return fmt.Errorf("MISSING_TEST: suite %s test %s source %s is not part of the native test selection of %s (build-tag excluded or absent)", suite.ID, test.ID, base, pkg)
			}
		}
	}
	if !found {
		return fmt.Errorf("MISSING_TEST: suite %s declares package %s which the module does not provide", suite.ID, pkg)
	}
	binary := filepath.Join(filepath.Dir(moduleDir), "build", path.Base(pkg)+".test")
	if _, err := goRunTool(ctx, scope, handle, moduleDir, env, goCompileTimeout, "test", "-c", "-o", binary, pkg); err != nil {
		return fmt.Errorf("BUILD_ERROR: suite %s package %s did not compile: %v", suite.ID, pkg, err)
	}
	return nil
}

// goGuardedChild runs one guarded custody job in its own child scope of the
// supplied live scope and closes that child with verified terminal
// retirement before returning: completed guardians never leak against the
// owning scope's concurrent job budget, and a cleanup that cannot verify is
// a CUSTODY_ERROR, never a silent pass.
func goGuardedChild(ctx context.Context, scope processscope.Scope, cmd processscope.Command, stdout, stderr *bytes.Buffer, outLimit, errLimit int64) (processscope.Result, error) {
	child, err := scope.Child(ctx)
	if err != nil {
		return processscope.Result{}, fmt.Errorf("CUSTODY_ERROR: open the guarded child scope: %w", err)
	}
	result, runErr := func() (processscope.Result, error) {
		attached, err := child.Attach(cmd)
		if err != nil {
			return processscope.Result{}, fmt.Errorf("CUSTODY_ERROR: attach the guarded job: %w", err)
		}
		return child.Run(ctx, attached, processscope.Streams{
			Stdout: stdout, Stderr: stderr, StdoutLimit: outLimit, StderrLimit: errLimit,
		})
	}()
	closeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	report, closeErr := child.Close(closeCtx)
	verified := closeErr == nil && report.Status == processscope.StatusCleaned
	for _, job := range report.Jobs {
		verified = verified && job.Registered && job.Terminated && job.Reaped
	}
	if !verified {
		return result, errors.Join(runErr, closeErr, fmt.Errorf("CUSTODY_ERROR: guarded child cleanup did not verify: %+v", report))
	}
	return result, runErr
}

// goRunTool executes one pinned toolchain invocation as a guarded child
// custody job of the suite scope.
func goRunTool(ctx context.Context, scope processscope.Scope, handle *runtimeclosure.Go, moduleDir string, env []string, timeoutMS int64, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	result, err := goGuardedChild(ctx, scope, processscope.Command{
		Executable:    handle.Binary(),
		Args:          args,
		Dir:           moduleDir,
		Env:           env,
		RuntimeDigest: handle.Identity().Closure,
		DeadlineMS:    timeoutMS,
	}, &stdout, &stderr, 1<<20, 1<<20)
	if err != nil {
		return stdout.Bytes(), fmt.Errorf("custody: %w (stderr: %s)", err, boundedString(stderr.String(), 2048))
	}
	if !result.Completed || result.ExitCode != 0 {
		return stdout.Bytes(), fmt.Errorf("exit=%d signal=%q stderr: %s", result.ExitCode, result.Signal, boundedString(stderr.String(), 2048))
	}
	return stdout.Bytes(), nil
}

// buildGoRunPattern builds the anchored native selection pattern from the
// declared identities: the deduplicated sorted top-level test functions,
// each regex-quoted and joined into one anchored alternation. The complete
// frozen leaf inventory is enforced by source validation and reconciliation,
// so nothing outside the declared set can execute unnoticed.
func buildGoRunPattern(tests []tdd.Test) (string, error) {
	tops := map[string]bool{}
	for _, test := range tests {
		name := test.Native.Test
		if name == "" {
			return "", fmt.Errorf("INVALID_SCHEMA: declared test %s has no native go identity", test.ID)
		}
		tops[strings.SplitN(name, "/", 2)[0]] = true
	}
	if len(tops) == 0 {
		return "", fmt.Errorf("INVALID_SCHEMA: empty native selection")
	}
	ordered := make([]string, 0, len(tops))
	for name := range tops {
		ordered = append(ordered, regexp.QuoteMeta(name))
	}
	sort.Strings(ordered)
	return "^(" + strings.Join(ordered, "|") + ")$", nil
}

var goClosedEnvKeys = map[string]bool{
	"PATH": true, "HOME": true, "TMPDIR": true, "GOFLAGS": true, "GOWORK": true,
	"GOTOOLCHAIN": true, "GOPROXY": true, "GOSUMDB": true, "CGO_ENABLED": true,
	"GO111MODULE": true, "GOCACHE": true, "GOMODCACHE": true, "GOPATH": true,
	"TZ": true, "LANG": true, "LC_ALL": true,
}

// buildGoEnv builds the closed native environment: the pinned toolchain on
// PATH, private output/home/tmp/cache roots, ambient flags, sumdb and
// toolchain downloads disabled, no network dependency fetching, and the
// explicitly declared suite variables appended. Nothing from the ambient
// environment is inherited.
func buildGoEnv(scratch, goroot string, extra []tdd.EnvironmentVar) ([]string, error) {
	declared := map[string]string{}
	for _, item := range extra {
		if !goEnvNamePattern.MatchString(item.Name) {
			return nil, fmt.Errorf("declared environment name %q is not a closed variable name", item.Name)
		}
		if len(item.Value) > 65536 {
			return nil, fmt.Errorf("declared environment value of %s exceeds the bounded size", item.Name)
		}
		if goClosedEnvKeys[item.Name] {
			return nil, fmt.Errorf("declared environment may not override the closed key %s", item.Name)
		}
		if _, duplicate := declared[item.Name]; duplicate {
			return nil, fmt.Errorf("declared environment name %s is duplicated", item.Name)
		}
		declared[item.Name] = item.Value
	}
	names := make([]string, 0, len(declared))
	for name := range declared {
		names = append(names, name)
	}
	sort.Strings(names)
	env := []string{
		"PATH=" + filepath.Join(goroot, "bin") + ":/usr/bin:/bin",
		"HOME=" + filepath.Join(scratch, "home"),
		"TMPDIR=" + filepath.Join(scratch, "tmp"),
		"GOFLAGS=",
		"GOWORK=off",
		"GOTOOLCHAIN=local",
		"GOPROXY=off",
		"GOSUMDB=off",
		"CGO_ENABLED=0",
		"GO111MODULE=on",
		"GOCACHE=" + filepath.Join(scratch, "gocache"),
		"GOMODCACHE=" + filepath.Join(scratch, "gomodcache"),
		"GOPATH=" + filepath.Join(scratch, "gopath"),
		"TZ=UTC",
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
	}
	for _, name := range names {
		env = append(env, name+"="+declared[name])
	}
	return env, nil
}

// goWitness is one parsed helper witness line.
type goWitness struct {
	id    string
	test  string
	site  string
	value bool
	line  int64
}

// parseWitnessLine parses the closed witness line grammar
//
//	machinery-check/v1 witness id=<ID> value=<true|false> test=<NAME> site=<FILE>:<LINE>
//
// The line is exactly six single-space separated fields; test names and ids
// carry no spaces by construction and the site is a base file name.
func parseWitnessLine(line string) (goWitness, bool) {
	fields := strings.Split(line, " ")
	if len(fields) != 6 || fields[0] != "machinery-check/v1" || fields[1] != "witness" {
		return goWitness{}, false
	}
	id, ok := strings.CutPrefix(fields[2], "id=")
	if !ok || !goAssertionIDPattern.MatchString(id) {
		return goWitness{}, false
	}
	valueText, ok := strings.CutPrefix(fields[3], "value=")
	if !ok || (valueText != "true" && valueText != "false") {
		return goWitness{}, false
	}
	test, ok := strings.CutPrefix(fields[4], "test=")
	if !ok || test == "" || strings.ContainsAny(test, " \t") {
		return goWitness{}, false
	}
	site, ok := strings.CutPrefix(fields[5], "site=")
	if !ok {
		return goWitness{}, false
	}
	file, lineText, found := strings.Cut(site, ":")
	if !found || file == "" || strings.ContainsAny(file, `/\`) {
		return goWitness{}, false
	}
	lineNumber, err := strconv.ParseInt(lineText, 10, 64)
	if err != nil || lineNumber < 1 {
		return goWitness{}, false
	}
	return goWitness{id: id, test: test, site: file, value: valueText == "true", line: lineNumber}, true
}

// goJSONEvent is one decoded native test2json event of the closed
// vocabulary of the pinned toolchain.
type goJSONEvent struct {
	Action     string
	Package    string
	Test       string
	Output     string
	OutputType string
}

var goStreamKeys = map[string]bool{
	"Time": true, "Action": true, "Package": true, "Test": true, "Output": true,
	"OutputType": true, "Elapsed": true, "ImportPath": true, "FailedBuild": true,
}

var goStreamActions = map[string]bool{
	"start": true, "run": true, "pause": true, "cont": true, "pass": true,
	"fail": true, "skip": true, "output": true, "build-output": true, "build-fail": true,
}

// parseGoTestStream decodes the closed test2json stream: one JSON object per
// line carrying only the closed keys and actions of the pinned Go toolchain.
// Any unknown key, unknown action, malformed line or non-string identity is
// a forged stream and fails.
func parseGoTestStream(raw []byte) ([]goJSONEvent, error) {
	var events []goJSONEvent
	for lineNumber, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var doc map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &doc); err != nil {
			return nil, fmt.Errorf("INVALID_SCHEMA: native stream line %d is not a JSON object: %v", lineNumber+1, err)
		}
		for key := range doc {
			if !goStreamKeys[key] {
				return nil, fmt.Errorf("INVALID_SCHEMA: native stream line %d carries unknown key %q", lineNumber+1, key)
			}
		}
		var decoded goJSONEvent
		for _, field := range []struct {
			key    string
			target *string
		}{
			{"Action", &decoded.Action}, {"Package", &decoded.Package},
			{"Test", &decoded.Test}, {"Output", &decoded.Output},
			{"OutputType", &decoded.OutputType},
		} {
			value, present := doc[field.key]
			if !present {
				continue
			}
			if err := json.Unmarshal(value, field.target); err != nil {
				return nil, fmt.Errorf("INVALID_SCHEMA: native stream line %d carries a non-string %s", lineNumber+1, field.key)
			}
		}
		if decoded.Action == "" || !goStreamActions[decoded.Action] {
			return nil, fmt.Errorf("INVALID_SCHEMA: native stream line %d carries unknown action %q", lineNumber+1, decoded.Action)
		}
		events = append(events, decoded)
	}
	return events, nil
}

// goReconciliation is the reconciled normalized result of one native stream.
type goReconciliation struct {
	Events     []tdd.Event
	Outcome    string
	Discovered []tdd.NativeID
	Started    []tdd.NativeID
	Completed  []tdd.NativeID
	Assertions []tdd.AssertionOutcome
}

type goTestState struct {
	test         *tdd.Test
	started      bool
	paused       bool
	terminal     string // "" | pass | fail
	witnessed    map[string]bool
	witnessFalse bool
	failSites    int
}

// reconcileGoStream reconciles the native lifecycle against the complete
// frozen leaf inventory with the registered assertion witnesses and emits
// the exact normalized machinery.tdd.event/v1 sequence.
func reconcileGoStream(events []goJSONEvent, suite *tdd.Suite) (goReconciliation, error) {
	rec := goReconciliation{}
	if len(suite.Tests) == 0 {
		return rec, fmt.Errorf("INCOMPLETE_EVENTS: empty declared selection cannot yield a successful execution")
	}
	states := map[string]*goTestState{}
	declaredPackages := map[string]bool{}
	assertionOwner := map[string]*tdd.Assertion{}
	assertionTest := map[string]*tdd.Test{}
	assertionFalse := map[string]bool{}
	for i := range suite.Tests {
		test := &suite.Tests[i]
		if _, duplicate := states[test.Native.Test]; duplicate {
			return rec, fmt.Errorf("INVALID_SCHEMA: native identity %s is declared more than once", test.Native.Test)
		}
		states[test.Native.Test] = &goTestState{test: test, witnessed: map[string]bool{}}
		declaredPackages[test.Native.Package] = true
		for j := range test.Assertions {
			assertion := &test.Assertions[j]
			if _, duplicate := assertionOwner[assertion.ID]; duplicate {
				return rec, fmt.Errorf("INVALID_SCHEMA: assertion %s is registered more than once", assertion.ID)
			}
			assertionOwner[assertion.ID] = assertion
			assertionTest[assertion.ID] = test
		}
	}
	for testPath := range states {
		if parent, _, found := strings.Cut(testPath, "/"); found {
			if _, ok := states[parent]; !ok {
				return rec, fmt.Errorf("MISSING_TEST: declared subtest %s has no declared parent %s", testPath, parent)
			}
		}
	}
	packageStart := map[string]bool{}
	packageTerminal := map[string]string{}
	suiteStarted := false
	sequence := int64(0)
	emit := func(kind string, test *tdd.Test, assertionID, outcome, source string, line int64) {
		sequence++
		event := tdd.Event{
			Schema: GoEventSchema, Sequence: sequence, Suite: suite.ID,
			Kind: kind, Assertion: assertionID, Outcome: outcome, Source: source, Line: line,
		}
		if test != nil {
			native := test.Native
			event.Native = &native
		}
		rec.Events = append(rec.Events, event)
	}
	for _, event := range events {
		switch event.Action {
		case "start":
			if event.Test != "" {
				return rec, fmt.Errorf("INVALID_SCHEMA: native start event carries a test identity")
			}
			if !declaredPackages[event.Package] {
				return rec, fmt.Errorf("DUPLICATE_TEST: native stream reports undeclared package %s", event.Package)
			}
			if packageStart[event.Package] {
				return rec, fmt.Errorf("DUPLICATE_TEST: package %s started more than once", event.Package)
			}
			packageStart[event.Package] = true
			if !suiteStarted {
				suiteStarted = true
				emit("suite-start", nil, "", "", "", 0)
				for i := range suite.Tests {
					native := suite.Tests[i].Native
					rec.Discovered = append(rec.Discovered, native)
					emit("discovered", &suite.Tests[i], "", "", "", 0)
				}
			}
		case "build-output":
			// bounded build diagnostics; the terminal build-fail action
			// classifies the failure class below
		case "build-fail":
			return rec, fmt.Errorf("BUILD_ERROR: the native build failed before test execution")
		case "run", "pause", "cont", "pass", "fail", "skip":
			if event.Test == "" {
				if event.Action != "pass" && event.Action != "fail" {
					return rec, fmt.Errorf("INVALID_SCHEMA: package-level %s event carries no test identity", event.Action)
				}
				if !declaredPackages[event.Package] {
					return rec, fmt.Errorf("DUPLICATE_TEST: native stream reports undeclared package %s", event.Package)
				}
				if packageTerminal[event.Package] != "" {
					return rec, fmt.Errorf("DUPLICATE_TEST: package %s reported more than one terminal status", event.Package)
				}
				packageTerminal[event.Package] = event.Action
				continue
			}
			state, declared := states[event.Test]
			if !declared {
				return rec, fmt.Errorf("DUPLICATE_TEST: native identity %q is outside the frozen inventory of this suite", event.Test)
			}
			switch event.Action {
			case "run":
				if state.started || state.terminal != "" {
					return rec, fmt.Errorf("DUPLICATE_TEST: native test %s started more than once", event.Test)
				}
				state.started = true
				rec.Started = append(rec.Started, state.test.Native)
				emit("test-start", state.test, "", "", "", 0)
			case "pause":
				if !state.started || state.paused || state.terminal != "" {
					return rec, fmt.Errorf("INVALID_SCHEMA: native pause of %s is not in an active lifecycle", event.Test)
				}
				state.paused = true
			case "cont":
				if !state.paused || state.terminal != "" {
					return rec, fmt.Errorf("INVALID_SCHEMA: native cont of %s has no matching pause", event.Test)
				}
				state.paused = false
			case "pass", "fail", "skip":
				if !state.started || state.terminal != "" || state.paused {
					return rec, fmt.Errorf("DUPLICATE_TEST: native terminal of %s is not in a complete lifecycle (started=%v terminal=%q paused=%v)", event.Test, state.started, state.terminal, state.paused)
				}
				if event.Action == "skip" {
					return rec, fmt.Errorf("UNSUPPORTED_FEATURE: declared test %s was skipped by the native runner; required skips can never become success", event.Test)
				}
				if event.Action == "pass" && state.witnessFalse {
					return rec, fmt.Errorf("ASSERTION_MISMATCH: witness reported a false condition but native test %s passed", event.Test)
				}
				state.terminal = event.Action
				rec.Completed = append(rec.Completed, state.test.Native)
				outcome := "pass"
				if event.Action == "fail" {
					outcome = "assertion-fail"
				}
				emit("test-end", state.test, "", outcome, "", 0)
			}
		case "output":
			if event.Test != "" {
				state, declared := states[event.Test]
				if !declared {
					return rec, fmt.Errorf("DUPLICATE_TEST: native output carries identity %q outside the frozen inventory", event.Test)
				}
				if !state.started || state.terminal != "" {
					return rec, fmt.Errorf("INVALID_SCHEMA: native output of %s is outside its active lifecycle", event.Test)
				}
				for _, line := range strings.Split(event.Output, "\n") {
					line = strings.TrimRight(line, "\r")
					if line == "" {
						continue
					}
					if event.OutputType == "error" && !strings.Contains(line, "machinery-check/v1 assertion ") {
						return rec, fmt.Errorf("UNEXPECTED_FAILURE: test %s reported failure output outside the registered helper transport: %q", event.Test, boundedGoString(line, 160))
					}
					if witness, ok := parseWitnessLine(line); ok {
						assertion, known := assertionOwner[witness.id]
						if !known {
							return rec, fmt.Errorf("ASSERTION_MISMATCH: witness carries assertion %q outside the registered inventory", witness.id)
						}
						owner := assertionTest[witness.id]
						if owner.Native.Test != event.Test || witness.test != event.Test {
							return rec, fmt.Errorf("ASSERTION_MISMATCH: witness of %s claims test %q but arrived inside %q", witness.id, witness.test, event.Test)
						}
						if filepath.ToSlash(filepath.Base(assertion.Source)) != witness.site || assertion.Line != witness.line {
							return rec, fmt.Errorf("ASSERTION_MISMATCH: witness of %s reports site %s:%d, want the registered call site %s:%d", witness.id, witness.site, witness.line, filepath.Base(assertion.Source), assertion.Line)
						}
						if state.witnessed[witness.id] {
							return rec, fmt.Errorf("ASSERTION_MISMATCH: assertion %s was witnessed more than once", witness.id)
						}
						state.witnessed[witness.id] = true
						if !witness.value {
							state.witnessFalse = true
							assertionFalse[witness.id] = true
						}
						outcome := "pass"
						if !witness.value {
							outcome = "assertion-fail"
						}
						emit("assertion", state.test, witness.id, outcome, assertion.Source, assertion.Line)
					}
					if strings.Contains(line, "machinery-check/v1 assertion ") && strings.Contains(line, " evaluated false at ") {
						state.failSites++
					}
				}
			} else {
				if strings.Contains(event.Output, "(cached)") {
					return rec, fmt.Errorf("INVALID_SCHEMA: cached native result detected even though the fixed invocation disables test caching")
				}
				if strings.Contains(event.Output, "[build failed]") {
					return rec, fmt.Errorf("BUILD_ERROR: the native runner reported a build failure")
				}
			}
		}
	}
	if !suiteStarted {
		return rec, fmt.Errorf("INCOMPLETE_EVENTS: the native stream carries no package lifecycle")
	}
	for i := range suite.Tests {
		test := &suite.Tests[i]
		state := states[test.Native.Test]
		if !state.started || state.terminal == "" {
			return rec, fmt.Errorf("INCOMPLETE_EVENTS: declared test %s never executed to a terminal state (started=%v terminal=%q)", test.Native.Test, state.started, state.terminal)
		}
		for _, assertion := range test.Assertions {
			if !state.witnessed[assertion.ID] {
				if state.terminal == "fail" {
					return rec, fmt.Errorf("UNEXPECTED_FAILURE: failing test %s provides no witnessed registered assertion for %s", test.Native.Test, assertion.ID)
				}
				return rec, fmt.Errorf("INCOMPLETE_EVENTS: registered assertion %s never executed in test %s", assertion.ID, test.Native.Test)
			}
		}
		if state.terminal == "fail" {
			witnessedFalse := false
			for _, assertion := range test.Assertions {
				if assertionFalse[assertion.ID] {
					witnessedFalse = true
				}
			}
			if len(test.Assertions) == 0 || !witnessedFalse || state.failSites < 1 {
				return rec, fmt.Errorf("UNEXPECTED_FAILURE: failing test %s has no reconciled registered assertion failure (assertions=%d failSites=%d)", test.Native.Test, len(test.Assertions), state.failSites)
			}
		}
	}
	for i := range suite.Tests {
		test := &suite.Tests[i]
		for _, assertion := range test.Assertions {
			outcome := "pass"
			if assertionFalse[assertion.ID] {
				outcome = "assertion-fail"
			}
			rec.Assertions = append(rec.Assertions, tdd.AssertionOutcome{
				Test: tdd.TestRef{Suite: suite.ID, Test: test.ID}, ID: assertion.ID, Outcome: outcome,
			})
		}
	}
	failed := false
	for i := range suite.Tests {
		if states[suite.Tests[i].Native.Test].terminal == "fail" {
			failed = true
		}
	}
	for pkg := range declaredPackages {
		if !packageStart[pkg] {
			return rec, fmt.Errorf("INCOMPLETE_EVENTS: declared package %s never started", pkg)
		}
		if packageTerminal[pkg] == "" {
			return rec, fmt.Errorf("INCOMPLETE_EVENTS: declared package %s never reported its terminal status", pkg)
		}
		if packageTerminal[pkg] == "pass" && failed {
			return rec, fmt.Errorf("INVALID_SCHEMA: package %s reported success while declared tests failed", pkg)
		}
		if packageTerminal[pkg] == "fail" && !failed {
			return rec, fmt.Errorf("INVALID_SCHEMA: package %s reported failure without a failed declared test", pkg)
		}
	}
	if failed {
		rec.Outcome = "assertion-fail"
	} else {
		rec.Outcome = "pass"
	}
	emit("suite-end", nil, "", rec.Outcome, "", 0)
	return rec, nil
}

type tRunCall struct {
	name string
}

type goSourceFile struct {
	fset     *token.FileSet
	file     *ast.File
	helper   string
	funcs    map[string]*ast.FuncDecl
	runCalls map[string][]tRunCall
}

// validateGoSuiteSources validates frozen suite sources: typed helper import
// and Check call sites at the declared file/line/id, declared fixed subtest
// names, and absence of TestMain/benchmark/fuzz entry points.
func validateGoSuiteSources(moduleDir, modulePath string, suite *tdd.Suite) error {
	helperImport := modulePath + "/" + GoHelperDir
	files := map[string]*goSourceFile{}
	for _, name := range suite.Files {
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(moduleDir, filepath.FromSlash(name)))
		if err != nil {
			return fmt.Errorf("MISSING_TEST: suite source %s is not part of the materialized module: %v", name, err)
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, name, body, 0)
		if err != nil {
			return fmt.Errorf("INVALID_SCHEMA: suite source %s does not parse: %v", name, err)
		}
		info := &goSourceFile{fset: fset, file: parsed, funcs: map[string]*ast.FuncDecl{}, runCalls: map[string][]tRunCall{}}
		for _, spec := range parsed.Imports {
			importPath, unquoteErr := strconv.Unquote(spec.Path.Value)
			if unquoteErr != nil || importPath != helperImport {
				continue
			}
			local := "machinerycheck"
			if spec.Name != nil {
				local = spec.Name.Name
			}
			if local == "_" || local == "." {
				return fmt.Errorf("ASSERTION_MISMATCH: suite source %s may not blank- or dot-import the assertion helper transport", name)
			}
			info.helper = local
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			funcName := fn.Name.Name
			if funcName == "TestMain" {
				return fmt.Errorf("UNSUPPORTED_FEATURE: suite source %s defines a custom TestMain; custom test drivers are outside the strict suite", name)
			}
			if strings.HasPrefix(funcName, "Benchmark") || strings.HasPrefix(funcName, "Fuzz") {
				return fmt.Errorf("UNSUPPORTED_FEATURE: suite source %s defines benchmark/fuzz entry point %s; they are not strict test identities", name, funcName)
			}
			if strings.HasPrefix(funcName, "Test") {
				info.funcs[funcName] = fn
			}
		}
		files[name] = info
	}
	for _, info := range files {
		for fnName, fn := range info.funcs {
			ast.Inspect(fn, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) < 1 {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Run" {
					return true
				}
				if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					subName, _ := strconv.Unquote(lit.Value)
					info.runCalls[fnName] = append(info.runCalls[fnName], tRunCall{name: subName})
				} else {
					info.runCalls[fnName] = append(info.runCalls[fnName], tRunCall{name: ""})
				}
				return true
			})
		}
	}
	declaredPaths := map[string]bool{}
	for _, test := range suite.Tests {
		declaredPaths[test.Native.Test] = true
	}
	for _, test := range suite.Tests {
		if parent, _, found := strings.Cut(test.Native.Test, "/"); found && !declaredPaths[parent] {
			return fmt.Errorf("MISSING_TEST: declared subtest %s has no declared parent %s in the suite inventory", test.Native.Test, parent)
		}
	}
	for i := range suite.Tests {
		test := &suite.Tests[i]
		info, ok := files[test.Source]
		if !ok {
			return fmt.Errorf("MISSING_TEST: test %s declares source %s which is not a suite file", test.ID, test.Source)
		}
		root, child, hasChild := strings.Cut(test.Native.Test, "/")
		fn, ok := info.funcs[root]
		if !ok {
			return fmt.Errorf("MISSING_TEST: source %s does not define the declared native test function %s", test.Source, root)
		}
		if hasChild {
			if child == "" || strings.Contains(child, "/") {
				return fmt.Errorf("MISSING_TEST: subtest identity %s exceeds the supported fixed depth", test.Native.Test)
			}
			found := false
			for _, call := range info.runCalls[root] {
				if call.name == child {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("MISSING_TEST: declared subtest %s has no fixed literal t.Run site in %s", test.Native.Test, test.Source)
			}
		}
		for j := range test.Assertions {
			assertion := &test.Assertions[j]
			if assertion.Source != test.Source {
				return fmt.Errorf("MISSING_TEST: assertion %s registers source %s, but its declaring test %s is frozen in %s", assertion.ID, assertion.Source, test.ID, test.Source)
			}
			if assertion.Helper != protocol.AssertionHelperV1 {
				return fmt.Errorf("ASSERTION_MISMATCH: assertion %s declares helper %q, want %s", assertion.ID, assertion.Helper, protocol.AssertionHelperV1)
			}
			if !goAssertionIDPattern.MatchString(assertion.ID) {
				return fmt.Errorf("ASSERTION_MISMATCH: assertion id %q is not a closed assertion identity", assertion.ID)
			}
			if info.helper == "" {
				return fmt.Errorf("ASSERTION_MISMATCH: source %s does not import the byte-pinned helper transport %s", test.Source, helperImport)
			}
			id, line := findHelperCallAt(info, fn, info.helper, assertion.Line)
			if id == nil {
				return fmt.Errorf("ASSERTION_MISMATCH: %s:%d is not the typed %s.Check call site of assertion %s", test.Source, assertion.Line, info.helper, assertion.ID)
			}
			if *id != assertion.ID {
				return fmt.Errorf("ASSERTION_MISMATCH: the typed call at %s:%d carries assertion id %q, want %q", test.Source, assertion.Line, *id, assertion.ID)
			}
			_ = line
		}
	}
	for _, info := range files {
		for fnName, calls := range info.runCalls {
			declaredRoot := false
			for _, test := range suite.Tests {
				if root, _, _ := strings.Cut(test.Native.Test, "/"); root == fnName {
					declaredRoot = true
				}
			}
			if !declaredRoot {
				continue
			}
			for _, call := range calls {
				if call.name == "" {
					return fmt.Errorf("UNSUPPORTED_FEATURE: test function %s contains a non-literal t.Run name; generated subtest identities are not strict test identities", fnName)
				}
				if !declaredPaths[fnName+"/"+call.name] {
					return fmt.Errorf("MISSING_TEST: fixed subtest %s/%s exists in the frozen source but is not declared in the suite inventory", fnName, call.name)
				}
			}
		}
	}
	return nil
}

// findHelperCallAt returns the assertion id literal of the typed helper call
// starting at exactly the declared source line, or nil when no such typed
// call exists at that line.
func findHelperCallAt(info *goSourceFile, fn *ast.FuncDecl, helperName string, line int64) (*string, int64) {
	var found *string
	foundLine := int64(0)
	ast.Inspect(fn, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 3 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Check" {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != helperName {
			return true
		}
		callLine := int64(info.fset.Position(call.Pos()).Line)
		if callLine != line {
			return true
		}
		lit, ok := call.Args[1].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		id, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		found = &id
		foundLine = callLine
		return false
	})
	return found, foundLine
}
