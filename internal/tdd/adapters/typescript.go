// Package adapters owns the closed first-release native adapters of
// docs/test-assurance-contract.md section 6. This file is the
// node-test-typescript/v1 adapter (MAC-avfp): it embeds the byte-pinned
// machinery-check/v1 assertion transport, the ambient node type closure, the
// frozen compiler configuration, the frozen module identity and the embedded
// Machinery reporter from the owned assets directory, prepares captured
// TypeScript suites (verified bundle materialization, typed assertion
// call-site validation, separate fresh pinned-compiler build step with
// noEmitOnError, non-incremental clean output, source maps with embedded
// sources whose sourcesContent must equal the frozen bundle bytes) and
// executes only that invocation's compiled outputs through Node's actual
// node:test runner under processscope custody, normalizing native events to
// machinery.tdd.event/v1.
package adapters

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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

//go:embed assets/typescript/machinery-check.ts
var tsHelperSource []byte

//go:embed assets/typescript/node-ambient.d.ts
var tsAmbientSource []byte

//go:embed assets/typescript/tsconfig.json
var tsConfigSource []byte

//go:embed assets/typescript/package.json
var tsPackageSource []byte

//go:embed assets/typescript/reporter.mjs
var tsReporterSource []byte

// Closed constants of the node-test-typescript/v1 adapter.
const (
	// AdapterNodeTestTS is the closed adapter identity.
	AdapterNodeTestTS = protocol.AdapterNodeTestTS

	// TypeScriptEventSchema is the normalized event identity of every
	// emitted event.
	TypeScriptEventSchema = "machinery.tdd.event/v1"

	// TypeScriptWitnessSchema is the closed helper witness identity.
	TypeScriptWitnessSchema = "machinery.tdd.witness/v1"

	// TypeScriptReporterSentinel is the terminal embedded-reporter marker;
	// its absence proves a truncated stream.
	TypeScriptReporterSentinel = "machinery:reporter:end"

	// The pinned exact sha256 of every embedded asset; each pin is verified
	// at materialization and re-verified after every native run.
	TypeScriptHelperPinnedSHA256   = "f44691b8d2c80af24ee1e990ad86dfb8ef4fadc2b520281e334a2106d2d8319f"
	TypeScriptAmbientPinnedSHA256  = "d38bd4c9d73cc06ed42ee716aafa34f4dc2fceed4a3a65b37db3c1a7b5d745c1"
	TypeScriptConfigPinnedSHA256   = "bd796144d36a5da159b072f4bd731d1d15f3fb94b25aed46d71e432782598257"
	TypeScriptPackagePinnedSHA256  = "256acf4930104c357b78e8d9ea02eb35e5a72948fb0a015bb3534470838912ba"
	TypeScriptReporterPinnedSHA256 = "04fef043d0307ff729886146db95c1188c766eac79a95816dea0e106ccd03d5f"

	// TypeScriptConformanceFixtureSHA256 pins the frozen native conformance
	// fixture bytes executed by the required contributor lane fragment
	// testdata/integration-lanes/assurance-typescript.json.
	TypeScriptConformanceFixtureSHA256 = "5fcd808b1f62d4adcb9ab0d0e246a8c59a64df2caa3dfe13f9f408fddf8e60bc"

	// TypeScriptCompileTimeoutMS bounds the separate pinned build step.
	TypeScriptCompileTimeoutMS = int64(300000)

	// tsSrcDir, tsAssetsDir and tsOutDir are the prepared layout members.
	tsSrcDir     = "src"
	tsAssetsDir  = "assets"
	tsOutDir     = "out"
	tsMaxLines   = 100000
	tsMaxLineLen = 1 << 20
)

var (
	tsAssertionIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)
	tsEnvNamePattern     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	tsSitePattern        = regexp.MustCompile(`^(.+):(\d+):(\d+)$`)
	tsDroppedEnvKeys     = map[string]bool{
		"NODE_OPTIONS": true, "NODE_PATH": true, "NODE_TEST_CONTEXT": true, "NODE_ENV": true,
		"MACHINERY_PROCESSSCOPE_CAP": true, "MACHINERY_PROCESSSCOPE_CHILD": true,
		"MACHINERY_PROCESSSCOPE_ATTACHMENT": true,
	}
	// tsKnownEventTypes is the closed embedded-reporter vocabulary; watch,
	// rerun and shard identities never appear because the frozen argv never
	// selects them, and an unknown identity fails closed.
	tsKnownEventTypes = map[string]bool{
		"test:enqueue": true, "test:dequeue": true, "test:start": true, "test:pass": true,
		"test:fail": true, "test:plan": true, "test:diagnostic": true, "test:stderr": true,
		"test:stdout": true, "test:complete": true, "test:summary": true,
	}
)

// tsDigestHex is the shared exact-byte digest helper of the asset pins.
func tsDigestHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// TypeScriptAdapter is the closed node-test-typescript/v1 Adapter.
type TypeScriptAdapter struct{}

// Compile-time contract check of the closed Adapter boundary.
var _ tdd.Adapter = (*TypeScriptAdapter)(nil)

// TypeScript returns the node-test-typescript/v1 adapter.
func TypeScript() *TypeScriptAdapter { return &TypeScriptAdapter{} }

// ID implements tdd.Adapter.
func (a *TypeScriptAdapter) ID() string { return AdapterNodeTestTS }

// TypeScriptHelperSource returns a copy of the embedded byte-pinned helper
// transport.
func TypeScriptHelperSource() []byte { return append([]byte(nil), tsHelperSource...) }

// TypeScriptAssets returns the frozen asset inventory: embedded asset name
// to its pinned exact sha256.
func TypeScriptAssets() map[string]string {
	return map[string]string{
		"machinery-check.ts": TypeScriptHelperPinnedSHA256,
		"node-ambient.d.ts":  TypeScriptAmbientPinnedSHA256,
		"tsconfig.json":      TypeScriptConfigPinnedSHA256,
		"package.json":       TypeScriptPackagePinnedSHA256,
		"reporter.mjs":       TypeScriptReporterPinnedSHA256,
	}
}

// tsEntry is one prepared suite entry file with its compiled outputs.
type tsEntry struct {
	sourceRel string
	srcDigest string
	outRel    string
	jsDigest  string
	mapDigest string
	entryFile string
}

// tsPrepared is the adapter-owned opaque prepared state: the frozen suite
// inventory, the prepared scratch layout identities, the verified source
// maps, the exact closed argv/environment and the pinned runtime members.
// Only this adapter's Run accepts it.
type tsPrepared struct {
	suite     *tdd.Suite
	inputs    tdd.InputView
	scope     processscope.Scope
	scratch   string
	entries   []tsEntry
	maps      map[string]tsSourceMap
	limits    tdd.Limits
	nodePath  string
	nodeDir   string
	compiler  string
	nodeSHA   string
	argv      []string
	env       []string
	assets    map[string]string
	assetDir  string
	ran       bool
	declared  map[string]*tdd.Test
	assertion map[string]*tdd.Assertion
	ownerTest map[string]string
}

// tsWritePinned writes exact bytes and verifies the written copy against the
// pinned digest before returning.
func tsWritePinned(path string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return err
	}
	got, err := os.ReadFile(path)
	if err != nil || tsDigestHex(got) != tsDigestHex(body) {
		return fmt.Errorf("INVALID_SCHEMA: materialized asset %s does not reproduce its frozen bytes", path)
	}
	return nil
}

// Prepare implements tdd.Adapter: it validates the closed request, opens and
// validates the pinned Node 26.8.1 / TypeScript 7.0.2 runtime closure under
// the live scope, materializes and re-verifies the captured source bundle
// plus every embedded pinned asset, validates each registered assertion as a
// typed helper call at its exact frozen line, compiles the suite fresh with
// the pinned native compiler (noEmitOnError, non-incremental clean output,
// source maps with embedded sources verified against the frozen bytes) and
// freezes the exact closed argv and environment.
func (a *TypeScriptAdapter) Prepare(ctx context.Context, req tdd.SuiteRequest) (tdd.PreparedSuite, error) {
	if req.Suite.Adapter != AdapterNodeTestTS {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_ADAPTER: suite %s declares adapter %q, not %s", req.Suite.ID, req.Suite.Adapter, AdapterNodeTestTS)
	}
	if req.Scope == nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s requires a live processscope scope", req.Suite.ID)
	}
	if req.Runtime == nil {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_VERSION: suite %s requires the pinned Node %s / TypeScript %s runtime handle", req.Suite.ID, runtimeclosure.RequiredNodeVersion, runtimeclosure.RequiredTypeScriptVersion)
	}
	handle, ok := req.Runtime.(*runtimeclosure.TypeScript)
	if !ok {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_VERSION: suite %s requires the exact approved TypeScript runtime closure handle, got %T", req.Suite.ID, req.Runtime)
	}
	if err := handle.Validate(ctx, req.Scope); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_VERSION: suite %s runtime validation failed: %w", req.Suite.ID, err)
	}
	if identity := handle.Identity(); req.Suite.Runtime != identity {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_VERSION: suite %s declares runtime %+v, but the pinned closure is %+v", req.Suite.ID, req.Suite.Runtime, identity)
	}
	if req.Inputs.Revalidate == nil || req.Inputs.Release == nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s requires a held InputView with revalidate/release callbacks", req.Suite.ID)
	}
	if err := req.Inputs.Revalidate(); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("STALE_INPUT: suite %s held inputs failed revalidation: %w", req.Suite.ID, err)
	}
	if req.Suite.Root != protocol.RepositoryRoot {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_FEATURE: node-test-typescript/v1 suites are flat root suites; suite %s declares root %q", req.Suite.ID, req.Suite.Root)
	}
	if len(req.Suite.DependencyRoots) != 0 {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_FEATURE: node-test-typescript/v1 resolves the frozen local dependency closure from the captured bundle; suite %s declares dependency roots %v", req.Suite.ID, req.Suite.DependencyRoots)
	}
	if !filepath.IsAbs(req.Scratch) {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s requires an absolute scratch root", req.Suite.ID)
	}
	if bundle := req.Source.Materialized(); bundle == "" {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s source bundle was not produced by capture", req.Suite.ID)
	}
	storeRoot := filepath.Dir(filepath.Dir(req.Source.Materialized()))
	projectID, err := tsStoreProjectID(storeRoot)
	if err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("STALE_INPUT: suite %s store identity: %w", req.Suite.ID, err)
	}
	bundleDir := filepath.Join(req.Scratch, "bundle")
	if err := os.MkdirAll(req.Scratch, 0o755); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("CUSTODY_ERROR: preparing the scratch root of suite %s: %w", req.Suite.ID, err)
	}
	if err := tdd.MaterializeBundle(ctx, storeRoot, projectID, req.Source.Ref(), bundleDir); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("STALE_INPUT: materializing the captured bundle of suite %s: %w", req.Suite.ID, err)
	}
	sources := map[string][]byte{}
	for _, file := range req.Suite.Files {
		if filepath.ToSlash(filepath.Dir(file)) != req.Suite.Root || !strings.HasSuffix(file, ".ts") || strings.HasSuffix(file, ".d.ts") {
			return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_FEATURE: suite %s declares non-flat or non-TypeScript file %q", req.Suite.ID, file)
		}
		body, err := os.ReadFile(filepath.Join(bundleDir, filepath.FromSlash(file)))
		if err != nil {
			return tdd.PreparedSuite{}, fmt.Errorf("MISSING_CONTRACT: suite file %s is absent from the captured bundle: %w", file, err)
		}
		sources[file] = body
	}
	if err := validateTypeScriptSuite(&req.Suite, sources); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("suite %s: %w", req.Suite.ID, err)
	}
	prepared, err := tsMaterializeAndCompile(ctx, req, handle, sources)
	if err != nil {
		return tdd.PreparedSuite{}, err
	}
	return tdd.NewPreparedSuite(prepared), nil
}

// tsMaterializeAndCompile materializes the pinned assets and captured
// sources into the scratch root and runs the separate pinned-compiler build
// step, verifying every compiled artifact and its source map.
func tsMaterializeAndCompile(ctx context.Context, req tdd.SuiteRequest, handle *runtimeclosure.TypeScript, sources map[string][]byte) (*tsPrepared, error) {
	scratch := req.Scratch
	if err := os.MkdirAll(filepath.Join(scratch, tsSrcDir), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(scratch, tsAssetsDir), 0o755); err != nil {
		return nil, err
	}
	assetTargets := []struct {
		rel  string
		body []byte
		pin  string
	}{
		{rel: "package.json", body: tsPackageSource, pin: TypeScriptPackagePinnedSHA256},
		{rel: filepath.Join(tsAssetsDir, "tsconfig.json"), body: tsConfigSource, pin: TypeScriptConfigPinnedSHA256},
		{rel: filepath.Join(tsAssetsDir, "reporter.mjs"), body: tsReporterSource, pin: TypeScriptReporterPinnedSHA256},
	}
	pins := TypeScriptAssets()
	for _, target := range assetTargets {
		if tsDigestHex(target.body) != pins[filepath.Base(target.rel)] {
			return nil, fmt.Errorf("INVALID_SCHEMA: embedded asset %s does not match its frozen pin", target.rel)
		}
		if err := tsWritePinned(filepath.Join(scratch, filepath.FromSlash(target.rel)), target.body); err != nil {
			return nil, err
		}
	}
	compileInputs := []string{}
	for _, file := range req.Suite.Files {
		rel := filepath.ToSlash(filepath.Join(tsSrcDir, file))
		if err := tsWritePinned(filepath.Join(scratch, filepath.FromSlash(rel)), sources[file]); err != nil {
			return nil, err
		}
		compileInputs = append(compileInputs, rel)
	}
	// The helper transport and its ambient type closure are materialized
	// byte-pinned next to every suite source directory.
	dir := filepath.ToSlash(filepath.Join(tsSrcDir, req.Suite.Root))
	helperRel := filepath.ToSlash(filepath.Join(dir, "machinery-check.ts"))
	ambientRel := filepath.ToSlash(filepath.Join(dir, "node-ambient.d.ts"))
	if tsDigestHex(tsHelperSource) != TypeScriptHelperPinnedSHA256 || tsDigestHex(tsAmbientSource) != TypeScriptAmbientPinnedSHA256 {
		return nil, fmt.Errorf("INVALID_SCHEMA: embedded helper transport does not match its frozen pin")
	}
	if err := tsWritePinned(filepath.Join(scratch, filepath.FromSlash(helperRel)), tsHelperSource); err != nil {
		return nil, err
	}
	if err := tsWritePinned(filepath.Join(scratch, filepath.FromSlash(ambientRel)), tsAmbientSource); err != nil {
		return nil, err
	}
	compileInputs = append(compileInputs, ambientRel, helperRel)
	sort.Strings(compileInputs)
	// The build step runs fresh: any pre-existing output is refused so a
	// stale dist can never be executed.
	outDir := filepath.Join(scratch, tsOutDir)
	if _, err := os.Lstat(outDir); err == nil {
		return nil, fmt.Errorf("STALE_INPUT: prepared scratch %s already carries compiled output", scratch)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	argv := append([]string{
		handle.CompilerPath(),
		"--strict", "--module", "nodenext", "--moduleResolution", "nodenext", "--target", "es2023",
		"--noEmitOnError", "--sourceMap", "--inlineSources", "--incremental", "false",
		"--outDir", tsOutDir,
	}, compileInputs...)
	compileEnv := []string{"PATH=/usr/bin:/bin", "HOME=" + scratch, "TMPDIR=" + scratch, "LC_ALL=C.UTF-8", "TZ=UTC"}
	if err := tsRunScoped(ctx, req.Scope, argv, scratch, compileEnv, "sha256:"+handle.CompilerDigest(), req.Limits, TypeScriptCompileTimeoutMS, tsCompileOutput); err != nil {
		return nil, fmt.Errorf("BUILD_ERROR: suite %s pinned TypeScript compilation failed: %w", req.Suite.ID, err)
	}
	prepared := &tsPrepared{
		suite: &req.Suite, inputs: req.Inputs, scope: req.Scope, scratch: scratch,
		maps: map[string]tsSourceMap{}, limits: req.Limits,
		nodePath: handle.NodePath(), nodeDir: filepath.Dir(handle.NodePath()),
		compiler: handle.CompilerPath(), nodeSHA: handle.NodeDigest(),
		assets: TypeScriptAssets(), assetDir: filepath.Join(scratch, tsAssetsDir),
		declared: map[string]*tdd.Test{}, assertion: map[string]*tdd.Assertion{}, ownerTest: map[string]string{},
	}
	for _, file := range req.Suite.Files {
		base := strings.TrimSuffix(filepath.Base(file), ".ts")
		outRel := filepath.ToSlash(filepath.Join(tsOutDir, base+".js"))
		mapRel := filepath.ToSlash(filepath.Join(tsOutDir, base+".js.map"))
		jsBytes, err := os.ReadFile(filepath.Join(scratch, filepath.FromSlash(outRel)))
		if err != nil {
			return nil, fmt.Errorf("BUILD_ERROR: compilation produced no output for %s: %w", file, err)
		}
		mapBytes, err := os.ReadFile(filepath.Join(scratch, filepath.FromSlash(mapRel)))
		if err != nil {
			return nil, fmt.Errorf("BUILD_ERROR: compilation produced no source map for %s: %w", file, err)
		}
		sourceMap, err := loadTypeScriptSourceMap(mapBytes, file, sources[file])
		if err != nil {
			return nil, fmt.Errorf("STALE_INPUT: compiled source map for %s does not bind the frozen bytes: %w", file, err)
		}
		prepared.maps[file] = sourceMap
		prepared.entries = append(prepared.entries, tsEntry{
			sourceRel: file, srcDigest: tsDigestHex(sources[file]),
			outRel: outRel, jsDigest: tsDigestHex(jsBytes), mapDigest: tsDigestHex(mapBytes),
			entryFile: filepath.Join(scratch, filepath.FromSlash(outRel)),
		})
	}
	entryArgs := []string{}
	for _, entry := range prepared.entries {
		entryArgs = append(entryArgs, entry.outRel)
	}
	reporterAbs := filepath.Join(prepared.assetDir, "reporter.mjs")
	prepared.argv = append([]string{
		prepared.nodePath, "--test", "--test-concurrency=1",
		"--test-reporter=" + reporterAbs, "--test-reporter-destination=stdout",
	}, entryArgs...)
	var err error
	prepared.env, err = buildTypeScriptRunEnv(scratch, prepared.nodeDir, req.Suite.Environment)
	if err != nil {
		return nil, err
	}
	for i := range req.Suite.Tests {
		test := &req.Suite.Tests[i]
		prepared.declared[tsIdentityKey(test.Native.Source, test.Native.Path)] = test
		for ai := range test.Assertions {
			assertion := &test.Assertions[ai]
			prepared.assertion[assertion.ID] = assertion
			prepared.ownerTest[assertion.ID] = tsIdentityKey(test.Native.Source, test.Native.Path)
		}
	}
	return prepared, nil
}

const tsCompileOutput = 1 << 22

// tsStoreProjectID reads the closed store identity of a capture store.
func tsStoreProjectID(storeRoot string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(storeRoot, "store.json"))
	if err != nil || len(raw) > 1<<20 {
		return "", fmt.Errorf("cannot read the store identity: %w", err)
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

// tsRunScoped executes one bounded guarded command through a fresh child
// scope of the live scope: the child is closed with verified terminal
// cleanup on every path, so the parent's job budget is not consumed by
// retired-but-unreaped guardians of completed commands.
func tsRunScoped(ctx context.Context, scope processscope.Scope, argv []string, dir string, env []string, runtimeDigest string, limits tdd.Limits, timeoutMS int64, outputLimit int64) error {
	deadline := timeoutMS
	if limits.WallMS > 0 && limits.WallMS < deadline {
		deadline = limits.WallMS
	}
	child, err := scope.Child(ctx)
	if err != nil {
		return fmt.Errorf("CUSTODY_ERROR: open the guarded child scope: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		if report, closeErr := child.Close(closeCtx); closeErr != nil || report.Status != processscope.StatusCleaned {
			for _, job := range report.Jobs {
				if !job.Registered || !job.Terminated || !job.Reaped {
					return
				}
			}
		}
	}()
	attached, err := child.Attach(processscope.Command{
		Executable: argv[0], Args: argv[1:], Dir: dir, Env: env,
		RuntimeDigest: runtimeDigest, DeadlineMS: deadline,
	})
	if err != nil {
		return fmt.Errorf("CUSTODY_ERROR: attach guarded command: %w", err)
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(deadline)*time.Millisecond)
	defer cancel()
	var out, errOut bytes.Buffer
	result, runErr := child.Run(runCtx, attached, processscope.Streams{
		Stdout: &out, Stderr: &errOut, StdoutLimit: outputLimit, StderrLimit: outputLimit,
	})
	if runErr != nil {
		var scopeErr *processscope.Error
		if errors.As(runErr, &scopeErr) {
			return fmt.Errorf("%s: guarded command failed: %w", scopeErr.Code, runErr)
		}
		return fmt.Errorf("CUSTODY_ERROR: guarded command failed: %w", runErr)
	}
	if !result.Completed || result.ExitCode != 0 {
		combined := out.String() + errOut.String()
		if len(combined) > 4096 {
			combined = combined[:4096]
		}
		return fmt.Errorf("guarded command exited incomplete/status %d: %s", result.ExitCode, combined)
	}
	return nil
}

// Run implements tdd.Adapter: it executes the prepared compiled outputs
// through Node's actual node:test runner with the embedded Machinery
// reporter under processscope custody, normalizes the native stream to the
// closed machinery.tdd.event/v1 sequence, reconciles the complete
// dequeue/complete lifecycle against the frozen inventory with
// witness-bound assertion outcomes, enforces exit concordance and re-verifies
// prepared sources against late mutation.
func (a *TypeScriptAdapter) Run(ctx context.Context, prepared tdd.PreparedSuite, sink tdd.EventSink) (tdd.Execution, error) {
	state, ok := prepared.PreparedState().(*tsPrepared)
	if !ok || state == nil {
		return tdd.Execution{}, fmt.Errorf("INVALID_SCHEMA: Run requires this adapter's own prepared state; a foreign carrier carries no authority")
	}
	if state.ran {
		return tdd.Execution{}, fmt.Errorf("INVALID_SCHEMA: the prepared suite state is single-use")
	}
	state.ran = true
	if err := tsReverifyPrepared(state); err != nil {
		return tdd.Execution{}, err
	}
	if err := state.inputs.Revalidate(); err != nil {
		return tdd.Execution{}, fmt.Errorf("STALE_INPUT: held inputs changed before execution: %w", err)
	}
	entryIndex := tsEntryIndex{}
	for _, entry := range state.entries {
		entryIndex[entry.entryFile] = entry.sourceRel
		if resolved, err := filepath.EvalSymlinks(entry.entryFile); err == nil && resolved != entry.entryFile {
			entryIndex[resolved] = entry.sourceRel
		}
	}
	deadline := state.limits.WallMS
	if deadline <= 0 {
		deadline = protocol.LimitWallDefaultMS
	}
	runChild, err := state.scope.Child(ctx)
	if err != nil {
		return tdd.Execution{}, fmt.Errorf("CUSTODY_ERROR: open the native execution child scope: %w", err)
	}
	execution := tdd.Execution{Suite: state.suite.ID, Custody: tdd.CustodyReport{Status: processscope.StatusCleaned}}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		if report, closeErr := runChild.Close(closeCtx); closeErr != nil || report.Status != processscope.StatusCleaned {
			for _, job := range report.Jobs {
				if !job.Registered || !job.Terminated || !job.Reaped {
					execution.Custody.Status = processscope.StatusCleanupFailed
					return
				}
			}
		}
	}()
	attached, err := runChild.Attach(processscope.Command{
		Executable: state.argv[0], Args: state.argv[1:], Dir: state.scratch, Env: state.env,
		RuntimeDigest: "sha256:" + state.nodeSHA, DeadlineMS: deadline,
	})
	if err != nil {
		return execution, fmt.Errorf("CUSTODY_ERROR: attach the native test process: %w", err)
	}
	stdoutLimit := state.limits.StdoutBytes
	if stdoutLimit <= 0 {
		stdoutLimit = protocol.LimitStdoutDefault
	}
	stderrLimit := state.limits.StderrBytes
	if stderrLimit <= 0 {
		stderrLimit = protocol.LimitStderrDefault
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(deadline)*time.Millisecond)
	defer cancel()
	var stdout, stderr bytes.Buffer
	result, runErr := runChild.Run(runCtx, attached, processscope.Streams{
		Stdout: &stdout, Stderr: &stderr, StdoutLimit: stdoutLimit, StderrLimit: stderrLimit,
	})
	execution.StdoutDigest = tsDigestHex(stdout.Bytes())
	execution.StderrDigest = tsDigestHex(stderr.Bytes())
	if result.Completed {
		code := int64(result.ExitCode)
		execution.ExitCode = &code
	}
	if runErr != nil {
		var scopeErr *processscope.Error
		if errors.As(runErr, &scopeErr) {
			switch scopeErr.Code {
			case processscope.CodeTimeout:
				return execution, fmt.Errorf("TIMEOUT: the native test process exceeded its wall deadline: %w", runErr)
			case processscope.CodeOutputLimit:
				return execution, fmt.Errorf("OUTPUT_LIMIT: the native test process exceeded its stream budget: %w", runErr)
			case processscope.CodeCanceled:
				return execution, fmt.Errorf("CUSTODY_ERROR: the native test process was cancelled: %w", runErr)
			}
		}
		execution.Custody.Status = processscope.StatusCleanupFailed
		return execution, fmt.Errorf("CUSTODY_ERROR: the native test process failed under custody: %w", runErr)
	}
	events, err := parseTypeScriptReporterStream(stdout.Bytes())
	if err != nil {
		return execution, fmt.Errorf("INCOMPLETE_EVENTS: the embedded reporter stream is not closed: %w", err)
	}
	rec, err := reconcileTypeScriptStream(events, state.suite, entryIndex, state.maps)
	if err != nil {
		return execution, err
	}
	// Exit concordance: the native runner exits zero only for an all-pass
	// inventory; anything else is an accounted failure or a violation.
	if rec.Outcome == "pass" && (execution.ExitCode == nil || *execution.ExitCode != 0) {
		return execution, fmt.Errorf("INCOMPLETE_EVENTS: reconciled pass disagrees with the native exit status %+v", execution.ExitCode)
	}
	if rec.Outcome == "fail" && execution.ExitCode != nil && *execution.ExitCode == 0 {
		return execution, fmt.Errorf("INCOMPLETE_EVENTS: reconciled assertion failure disagrees with the native exit status 0")
	}
	eventCount := state.limits.EventCount
	if eventCount <= 0 {
		eventCount = protocol.LimitEventCountDefault
	}
	if int64(len(rec.Events)) > eventCount {
		return execution, fmt.Errorf("OUTPUT_LIMIT: normalized event count %d exceeds the budget %d", len(rec.Events), eventCount)
	}
	if sink != nil {
		for _, event := range rec.Events {
			if err := sink(event); err != nil {
				return execution, fmt.Errorf("INVALID_SCHEMA: the event sink rejected the normalized stream: %w", err)
			}
		}
	}
	execution.Events = rec.Events
	execution.Outcome = rec.Outcome
	return execution, nil
}

// tsReverifyPrepared re-hashes every prepared artifact against the digests
// frozen at Prepare; late mutation fails closed.
func tsReverifyPrepared(state *tsPrepared) error {
	pins := TypeScriptAssets()
	helperDir := filepath.Join(state.scratch, tsSrcDir)
	if root := state.suite.Root; root != "." {
		helperDir = filepath.Join(helperDir, filepath.FromSlash(root))
	}
	locations := map[string]string{
		"package.json":       filepath.Join(state.scratch, "package.json"),
		"tsconfig.json":      filepath.Join(state.assetDir, "tsconfig.json"),
		"reporter.mjs":       filepath.Join(state.assetDir, "reporter.mjs"),
		"machinery-check.ts": filepath.Join(helperDir, "machinery-check.ts"),
		"node-ambient.d.ts":  filepath.Join(helperDir, "node-ambient.d.ts"),
	}
	for name, pin := range pins {
		path, ok := locations[name]
		if !ok {
			return fmt.Errorf("STALE_INPUT: the frozen asset %s has no prepared location", name)
		}
		body, err := os.ReadFile(path)
		if err != nil || tsDigestHex(body) != pin {
			return fmt.Errorf("STALE_INPUT: the materialized asset %s no longer matches its frozen pin", name)
		}
	}
	for _, entry := range state.entries {
		src, err := os.ReadFile(filepath.Join(state.scratch, filepath.FromSlash(filepath.Join(tsSrcDir, entry.sourceRel))))
		if err != nil || tsDigestHex(src) != entry.srcDigest {
			return fmt.Errorf("STALE_INPUT: prepared source %s changed after preparation", entry.sourceRel)
		}
		js, err := os.ReadFile(filepath.Join(state.scratch, filepath.FromSlash(entry.outRel)))
		if err != nil || tsDigestHex(js) != entry.jsDigest {
			return fmt.Errorf("STALE_INPUT: compiled output %s changed after preparation", entry.outRel)
		}
		mapBytes, err := os.ReadFile(filepath.Join(state.scratch, filepath.FromSlash(strings.TrimSuffix(entry.outRel, ".js")+".js.map")))
		if err != nil || tsDigestHex(mapBytes) != entry.mapDigest {
			return fmt.Errorf("STALE_INPUT: source map %s changed after preparation", entry.outRel)
		}
	}
	return nil
}

// tsWitness is one parsed machinery-check/v1 witness diagnostic.
type tsWitness struct {
	assertion string
	site      string
	condition bool
	thrown    string
}

// parseTypeScriptWitness parses one helper witness diagnostic message; a
// null thrown class encodes the no-failure empty spelling.
func parseTypeScriptWitness(message string) (tsWitness, bool) {
	var doc struct {
		Schema    string          `json:"schema"`
		Assertion string          `json:"assertion"`
		Phase     string          `json:"phase"`
		Site      string          `json:"site"`
		Condition *bool           `json:"condition"`
		Thrown    json.RawMessage `json:"thrown"`
	}
	if err := json.Unmarshal([]byte(message), &doc); err != nil {
		return tsWitness{}, false
	}
	thrown := ""
	switch {
	case string(doc.Thrown) == "null":
	case len(doc.Thrown) > 0 && doc.Thrown[0] == '"':
		if err := json.Unmarshal(doc.Thrown, &thrown); err != nil {
			return tsWitness{}, false
		}
	default:
		return tsWitness{}, false
	}
	if doc.Schema != TypeScriptWitnessSchema || doc.Phase != "evaluated" || doc.Assertion == "" || doc.Site == "" || doc.Condition == nil || len(doc.Thrown) == 0 {
		return tsWitness{}, false
	}
	if !tsAssertionIDPattern.MatchString(doc.Assertion) {
		return tsWitness{}, false
	}
	return tsWitness{assertion: doc.Assertion, site: doc.Site, condition: *doc.Condition, thrown: thrown}, true
}

// tsNativeError is the shaped native failure cause of one test event.
type tsNativeError struct {
	Name    string
	Message string
	Code    string
}

// tsSummaryCounts is the shaped final native summary accounting.
type tsSummaryCounts struct {
	Tests     int64
	Failed    int64
	Passed    int64
	Cancelled int64
	Skipped   int64
	Todo      int64
	Success   bool
}

// tsNativeEvent is one decoded embedded-reporter event of the closed
// node:test reporter vocabulary.
type tsNativeEvent struct {
	Type      string
	Nesting   int64
	Name      string
	File      string
	EntryFile string
	Line      int64
	Column    int64
	TestID    int64
	ParentID  int64
	Skip      bool
	Todo      bool
	Message   string
	Error     *tsNativeError
	Counts    *tsSummaryCounts
}

// tsRawEvent mirrors the reporter line shape {"t":..., "d":...}.
type tsRawEvent struct {
	Type string          `json:"t"`
	D    json.RawMessage `json:"d"`
}

// parseTypeScriptReporterStream decodes the closed embedded-reporter JSONL
// vocabulary; unknown event identities, malformed lines and missing,
// duplicated or non-terminal sentinels fail closed.
func parseTypeScriptReporterStream(raw []byte) ([]tsNativeEvent, error) {
	text := strings.TrimSuffix(string(raw), "\n")
	if text == "" {
		return nil, fmt.Errorf("INCOMPLETE_EVENTS: the reporter stream is empty")
	}
	lines := strings.Split(text, "\n")
	if int64(len(lines)) > tsMaxLines {
		return nil, fmt.Errorf("OUTPUT_LIMIT: the reporter stream exceeds %d lines", tsMaxLines)
	}
	events := make([]tsNativeEvent, 0, len(lines))
	sentinel := -1
	for i, line := range lines {
		if len(line) > tsMaxLineLen {
			return nil, fmt.Errorf("OUTPUT_LIMIT: a reporter line exceeds the bound")
		}
		var raw tsRawEvent
		if err := json.Unmarshal([]byte(line), &raw); err != nil || raw.Type == "" {
			return nil, fmt.Errorf("INCOMPLETE_EVENTS: malformed reporter line %d", i+1)
		}
		if raw.Type == TypeScriptReporterSentinel {
			if sentinel != -1 {
				return nil, fmt.Errorf("INCOMPLETE_EVENTS: duplicated terminal sentinel")
			}
			if i != len(lines)-1 {
				return nil, fmt.Errorf("INCOMPLETE_EVENTS: the terminal sentinel is not terminal")
			}
			sentinel = i
			events = append(events, tsNativeEvent{Type: raw.Type})
			continue
		}
		if !tsKnownEventTypes[raw.Type] {
			return nil, fmt.Errorf("INCOMPLETE_EVENTS: unknown native event identity %q", raw.Type)
		}
		event, err := tsDecodeNativeEvent(raw.Type, raw.D)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if sentinel == -1 {
		return nil, fmt.Errorf("INCOMPLETE_EVENTS: the terminal sentinel is missing; the stream is truncated")
	}
	return events, nil
}

func tsDecodeNativeEvent(kind string, d json.RawMessage) (tsNativeEvent, error) {
	event := tsNativeEvent{Type: kind}
	if len(d) == 0 {
		return event, nil
	}
	var doc struct {
		Nesting   int64           `json:"nesting"`
		Name      string          `json:"name"`
		File      string          `json:"file"`
		EntryFile string          `json:"entryFile"`
		Line      int64           `json:"line"`
		Column    int64           `json:"column"`
		TestID    int64           `json:"testId"`
		ParentID  int64           `json:"parentId"`
		Skip      json.RawMessage `json:"skip"`
		Todo      json.RawMessage `json:"todo"`
		Message   string          `json:"message"`
		Details   *struct {
			Passed *bool `json:"passed"`
			Error  *struct {
				Name    string `json:"name"`
				Message string `json:"message"`
				Code    string `json:"code"`
			} `json:"error"`
		} `json:"details"`
		Counts *struct {
			Tests     int64 `json:"tests"`
			Failed    int64 `json:"failed"`
			Passed    int64 `json:"passed"`
			Cancelled int64 `json:"cancelled"`
			Skipped   int64 `json:"skipped"`
			Todo      int64 `json:"todo"`
		} `json:"counts"`
		Success *bool `json:"success"`
	}
	if err := json.Unmarshal(d, &doc); err != nil {
		return event, fmt.Errorf("INCOMPLETE_EVENTS: malformed %s payload: %w", kind, err)
	}
	event.Nesting, event.Name, event.File, event.EntryFile = doc.Nesting, doc.Name, doc.File, doc.EntryFile
	event.Line, event.Column, event.TestID, event.ParentID = doc.Line, doc.Column, doc.TestID, doc.ParentID
	event.Skip, event.Todo, event.Message = tsFlagSet(doc.Skip), tsFlagSet(doc.Todo), doc.Message
	if doc.Details != nil && doc.Details.Error != nil {
		event.Error = &tsNativeError{Name: doc.Details.Error.Name, Message: doc.Details.Error.Message, Code: doc.Details.Error.Code}
	}
	if kind == "test:summary" && doc.Counts != nil {
		success := doc.Success != nil && *doc.Success
		event.Counts = &tsSummaryCounts{
			Tests: doc.Counts.Tests, Failed: doc.Counts.Failed, Passed: doc.Counts.Passed,
			Cancelled: doc.Counts.Cancelled, Skipped: doc.Counts.Skipped, Todo: doc.Counts.Todo, Success: success,
		}
		// Only the summary without file scope is the final accounting; the
		// per-file summaries are retained cross-checks.
		if doc.File != "" || doc.EntryFile != "" {
			event.File, event.EntryFile = doc.File, doc.EntryFile
		} else {
			event.File, event.EntryFile = "", ""
		}
	}
	return event, nil
}

// tsFlagSet decodes a native skip/todo flag that node spells as a boolean
// or as the skip-message string.
func tsFlagSet(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "false" {
		return false
	}
	if raw[0] == '"' {
		return len(raw) > 2
	}
	return string(raw) == "true"
}

// tsSourceMap is one verified compiler-produced source map whose embedded
// sourcesContent equals the frozen bundle bytes.
type tsSourceMap struct {
	Generation int64
	Source     string
	segments   []tsMapSegment
}

// tsMapSegment is one decoded mapping segment: generated column to original
// line/column in the single frozen source.
type tsMapSegment struct {
	GenLine int64
	GenCol  int64
	SrcLine int64
	SrcCol  int64
}

// loadTypeScriptSourceMap decodes and verifies one compiler-produced .js.map:
// the named source's embedded content must equal the frozen source bytes.
func loadTypeScriptSourceMap(mapBytes []byte, sourceName string, frozen []byte) (tsSourceMap, error) {
	var doc struct {
		Version        int      `json:"version"`
		Sources        []string `json:"sources"`
		SourcesContent []string `json:"sourcesContent"`
		Mappings       string   `json:"mappings"`
	}
	if err := json.Unmarshal(mapBytes, &doc); err != nil {
		return tsSourceMap{}, fmt.Errorf("malformed source map: %w", err)
	}
	if doc.Version != 3 || len(doc.Sources) == 0 || len(doc.SourcesContent) != len(doc.Sources) {
		return tsSourceMap{}, fmt.Errorf("source map is not a closed version-3 map with embedded sources")
	}
	index := -1
	for i, source := range doc.Sources {
		if filepath.ToSlash(filepath.Base(source)) == filepath.ToSlash(filepath.Base(sourceName)) {
			index = i
			break
		}
	}
	if index < 0 {
		return tsSourceMap{}, fmt.Errorf("source map does not name the frozen source %s", sourceName)
	}
	if doc.SourcesContent[index] != string(frozen) {
		return tsSourceMap{}, fmt.Errorf("source map embedded content differs from the frozen bytes of %s", sourceName)
	}
	segments, err := tsDecodeMappings(doc.Mappings)
	if err != nil {
		return tsSourceMap{}, err
	}
	return tsSourceMap{Generation: 3, Source: sourceName, segments: segments}, nil
}

// SourcePosition maps one generated position back to the frozen original
// source position.
func (m tsSourceMap) SourcePosition(genLine, genCol int64) (srcLine, srcCol int64, ok bool) {
	found := -1
	for i, segment := range m.segments {
		if segment.GenLine != genLine {
			continue
		}
		if segment.GenCol <= genCol {
			found = i
		} else {
			break
		}
	}
	if found < 0 {
		return 0, 0, false
	}
	return m.segments[found].SrcLine, m.segments[found].SrcCol, true
}

// tsDecodeMappings decodes the standard VLQ mappings string into 1-based
// original positions with globally accumulated source state.
func tsDecodeMappings(mappings string) ([]tsMapSegment, error) {
	var segments []tsMapSegment
	var srcIdx, srcLine, srcCol int64
	for genLine := int64(1); genLine < 1<<30; genLine++ {
		end := strings.IndexByte(mappings, ';')
		var line string
		if end < 0 {
			line = mappings
		} else {
			line = mappings[:end]
		}
		genCol := int64(0)
		if line != "" {
			for _, seg := range strings.Split(line, ",") {
				if seg == "" {
					continue
				}
				fields, err := tsDecodeVLQ(seg)
				if err != nil {
					return nil, err
				}
				if len(fields) == 0 {
					continue
				}
				genCol += fields[0]
				if len(fields) >= 4 {
					srcIdx += fields[1]
					srcLine += fields[2]
					srcCol += fields[3]
					if srcIdx != 0 {
						return nil, fmt.Errorf("source map references more than one source index")
					}
					segments = append(segments, tsMapSegment{GenLine: genLine, GenCol: genCol, SrcLine: srcLine + 1, SrcCol: srcCol + 1})
				}
			}
		}
		if end < 0 {
			break
		}
		mappings = mappings[end+1:]
	}
	return segments, nil
}

func tsDecodeVLQ(segment string) ([]int64, error) {
	var out []int64
	var value, shift int64
	for i := 0; i < len(segment); i++ {
		digit := strings.IndexByte(tsBase64Alphabet, segment[i])
		if digit < 0 {
			return nil, fmt.Errorf("malformed VLQ segment %q", segment)
		}
		value |= int64(digit&31) << shift
		if digit&32 == 0 {
			negative := value&1 == 1
			value >>= 1
			if negative {
				value = -value
			}
			out = append(out, value)
			value, shift = 0, 0
			continue
		}
		shift += 5
		if shift > 40 {
			return nil, fmt.Errorf("malformed VLQ segment %q", segment)
		}
	}
	if shift != 0 || len(out) == 0 {
		return nil, fmt.Errorf("malformed VLQ segment %q", segment)
	}
	return out, nil
}

const tsBase64Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

// tsEntryIndex maps native entry-file identities to suite-relative sources.
type tsEntryIndex map[string]string

func tsIdentityKey(source string, path []string) string {
	return source + "\x00" + strings.Join(path, " > ")
}

type tsTestState struct {
	test      *tdd.Test
	enqueued  bool
	dequeued  int
	completed int
	started   int
	pass      int
	fail      int
	passed    bool
	failed    bool
	skipped   bool
	order     int64
}

// tsReconciliation is the reconciled normalized result of one native stream.
type tsReconciliation struct {
	Events     []tdd.Event
	Outcome    string
	Discovered []tdd.NativeID
	Started    []tdd.NativeID
	Completed  []tdd.NativeID
	Assertions []tdd.AssertionOutcome
	Failed     []tdd.NativeID
}

// reconcileTypeScriptStream reconciles native lifecycle events against the
// complete frozen inventory: enqueue/dequeue/complete per declared identity,
// skip/todo/cancelled rejection, witness-bound assertion outcomes at the
// exact registered call sites through the verified source maps, and final
// summary concordance.
func reconcileTypeScriptStream(events []tsNativeEvent, suite *tdd.Suite, entries tsEntryIndex, maps map[string]tsSourceMap) (tsReconciliation, error) {
	states := map[string]*tsTestState{}
	for i := range suite.Tests {
		test := &suite.Tests[i]
		states[tsIdentityKey(test.Native.Source, test.Native.Path)] = &tsTestState{test: test}
	}
	// Per-entry native id registries reconstruct name hierarchies from the
	// enqueue events (numeric native ids alone are never global).
	natives := map[string]map[int64]tsNativeIdent{}
	dequeueOrder := []string{}
	witnesses := map[string][]tsWitness{}
	var finalSummary *tsSummaryCounts
	if len(events) == 0 || events[len(events)-1].Type != TypeScriptReporterSentinel {
		return tsReconciliation{}, fmt.Errorf("INCOMPLETE_EVENTS: the terminal reporter sentinel is absent")
	}
	for _, event := range events {
		switch event.Type {
		case TypeScriptReporterSentinel, "test:plan", "test:stdout", "test:stderr":
			continue
		case "test:diagnostic":
			if witness, ok := parseTypeScriptWitness(event.Message); ok {
				witnesses[witness.assertion] = append(witnesses[witness.assertion], witness)
			}
			continue
		case "test:summary":
			if event.Counts != nil && event.File == "" && event.EntryFile == "" {
				if finalSummary != nil {
					return tsReconciliation{}, fmt.Errorf("INCOMPLETE_EVENTS: duplicated final native summary")
				}
				finalSummary = event.Counts
			}
			continue
		}
		if event.EntryFile == "" {
			continue // the synthetic file-level root test is not leaf evidence
		}
		source, ok := entries[event.EntryFile]
		if !ok {
			return tsReconciliation{}, fmt.Errorf("INCOMPLETE_EVENTS: native event for unknown entry file %q", event.EntryFile)
		}
		if natives[event.EntryFile] == nil {
			natives[event.EntryFile] = map[int64]tsNativeIdent{}
		}
		ident := natives[event.EntryFile]
		if event.Type == "test:enqueue" {
			ident[event.TestID] = tsNativeIdent{name: event.Name, parentID: event.ParentID}
		}
		path, ok := tsNativePath(ident, event.TestID)
		if !ok {
			path, ok = tsNativePathByName(ident, event.Name)
		}
		key := tsIdentityKey(source, path)
		state := states[key]
		if !ok || state == nil {
			// The native id registry cannot resolve this event (for example
			// a forged stream whose enqueue was dropped). Bind by declared
			// name when exactly one declared identity of this entry source
			// carries it; an unknown executed name stays a violation.
			var matches []*tsTestState
			for _, candidate := range states {
				if candidate.test.Native.Source == source && len(candidate.test.Native.Path) > 0 && candidate.test.Native.Path[len(candidate.test.Native.Path)-1] == event.Name {
					matches = append(matches, candidate)
				}
			}
			if len(matches) != 1 {
				if len(matches) == 0 {
					return tsReconciliation{}, fmt.Errorf("MISSING_TEST: unregistered native identity %q of %s executed", event.Name, source)
				}
				return tsReconciliation{}, fmt.Errorf("INCOMPLETE_EVENTS: ambiguous native identity %q of %s", event.Name, source)
			}
			state = matches[0]
			key = tsIdentityKey(state.test.Native.Source, state.test.Native.Path)
		}
		switch event.Type {
		case "test:enqueue":
			state.enqueued = true
		case "test:dequeue":
			state.dequeued++
		case "test:start":
			state.started++
		case "test:complete":
			state.completed++
			state.passed = event.Error == nil
			state.failed = event.Error != nil
			if event.Skip || event.Todo {
				state.skipped = true
			}
		case "test:pass":
			state.pass++
			if event.Skip || event.Todo {
				state.skipped = true
			}
		case "test:fail":
			state.fail++
		}
		if state.order == 0 && state.enqueued {
			state.order = int64(len(dequeueOrder)) + 1
			dequeueOrder = append(dequeueOrder, key)
		}
	}
	if finalSummary == nil {
		return tsReconciliation{}, fmt.Errorf("INCOMPLETE_EVENTS: the final native summary is missing")
	}
	var violations []error
	failures := 0
	for key, state := range states {
		if !state.enqueued || state.dequeued == 0 {
			violations = append(violations, fmt.Errorf("MISSING_TEST: declared identity %q was never enqueued and dequeued by the native runner", key))
			continue
		}
		if state.dequeued != 1 || state.completed != 1 {
			violations = append(violations, fmt.Errorf("INCOMPLETE_EVENTS: declared identity %q has an incomplete dequeue/complete lifecycle (dequeued=%d completed=%d)", key, state.dequeued, state.completed))
			continue
		}
		if state.skipped {
			violations = append(violations, fmt.Errorf("UNSUPPORTED_FEATURE: declared identity %q was skipped or marked todo; skipped or todo tests cannot certify anything", key))
			continue
		}
		if state.started != 1 || (state.failed && state.fail != 1) || (!state.failed && state.pass != 1) {
			violations = append(violations, fmt.Errorf("INCOMPLETE_EVENTS: declared identity %q has an inconsistent buffered lifecycle (start=%d pass=%d fail=%d failed=%v)", key, state.started, state.pass, state.fail, state.failed))
			continue
		}
		if state.failed {
			failures++
		}
	}
	// Witness binding: every registered assertion executes exactly once per
	// run, at its exact registered frozen call site.
	assertionOutcomes := map[string]string{}
	for id, assertion := range suiteAssertions(suite) {
		seen := witnesses[id]
		if len(seen) != 1 {
			violations = append(violations, fmt.Errorf("ASSERTION_MISMATCH: registered assertion %s has %d witness evaluations, want exactly 1", id, len(seen)))
			continue
		}
		witness := seen[0]
		source, line, ok := tsResolveWitnessSite(witness.site, entries, maps)
		if !ok {
			violations = append(violations, fmt.Errorf("ASSERTION_MISMATCH: witness of %s does not resolve to a prepared entry", id))
			continue
		}
		if source != assertion.Source || line != assertion.Line {
			violations = append(violations, fmt.Errorf("ASSERTION_MISMATCH: witness of %s resolves to %s:%d, registered at %s:%d", id, source, line, assertion.Source, assertion.Line))
			continue
		}
		if witness.condition {
			assertionOutcomes[id] = protocol.OutcomePass
			continue
		}
		if witness.thrown != "AssertionError" {
			violations = append(violations, fmt.Errorf("UNEXPECTED_FAILURE: assertion %s failed without a native AssertionError cause (thrown %q)", id, witness.thrown))
			continue
		}
		assertionOutcomes[id] = protocol.OutcomeAssertionFail
	}
	for id := range witnesses {
		if _, ok := suiteAssertions(suite)[id]; !ok {
			violations = append(violations, fmt.Errorf("ASSERTION_MISMATCH: witness for unregistered assertion %s", id))
		}
	}
	// A failing test must fail at one of its own registered assertions.
	for key, state := range states {
		if !state.failed {
			continue
		}
		bound := false
		for _, assertion := range state.test.Assertions {
			if assertionOutcomes[assertion.ID] == protocol.OutcomeAssertionFail {
				bound = true
			}
		}
		if !bound {
			violations = append(violations, fmt.Errorf("UNEXPECTED_FAILURE: failing identity %q is not bound to a helper-backed assertion failure", key))
		}
	}
	if finalSummary.Skipped != 0 || finalSummary.Todo != 0 {
		violations = append(violations, fmt.Errorf("UNSUPPORTED_FEATURE: the native summary reports skipped=%d todo=%d", finalSummary.Skipped, finalSummary.Todo))
	}
	if finalSummary.Cancelled != 0 {
		violations = append(violations, fmt.Errorf("INCOMPLETE_EVENTS: the native summary reports cancelled=%d", finalSummary.Cancelled))
	}
	if finalSummary.Failed != int64(failures) {
		violations = append(violations, fmt.Errorf("INCOMPLETE_EVENTS: the native summary reports %d failures, reconciled %d", finalSummary.Failed, failures))
	}
	if finalSummary.Success && failures != 0 {
		violations = append(violations, fmt.Errorf("INCOMPLETE_EVENTS: the native summary claims success over %d reconciled failures", failures))
	}
	if len(violations) > 0 {
		return tsReconciliation{}, errors.Join(violations...)
	}
	rec := tsReconciliation{Outcome: "pass", Events: []tdd.Event{}, Discovered: []tdd.NativeID{}, Started: []tdd.NativeID{}, Completed: []tdd.NativeID{}, Assertions: []tdd.AssertionOutcome{}, Failed: []tdd.NativeID{}}
	ordered := append([]string(nil), dequeueOrder...)
	sort.Slice(ordered, func(i, j int) bool { return states[ordered[i]].order < states[ordered[j]].order })
	sequence := int64(0)
	emit := func(kind string, test *tdd.Test, assertion string, outcome string) {
		sequence++
		event := tdd.Event{
			Schema: TypeScriptEventSchema, Sequence: sequence, Suite: suite.ID,
			Kind: kind, Assertion: assertion, Outcome: outcome,
		}
		if test != nil {
			native := test.Native
			event.Native = &native
			event.Source = native.Source
			event.Line = native.Line
		}
		rec.Events = append(rec.Events, event)
	}
	emit("suite-start", nil, "", "")
	for _, key := range ordered {
		state := states[key]
		native := state.test.Native
		rec.Discovered = append(rec.Discovered, native)
		emit("discovered", state.test, "", "")
		emit("test-start", state.test, "", "")
		rec.Started = append(rec.Started, native)
		for _, assertion := range state.test.Assertions {
			outcome := assertionOutcomes[assertion.ID]
			emit("assertion", state.test, assertion.ID, outcome)
			rec.Assertions = append(rec.Assertions, tdd.AssertionOutcome{Test: tdd.TestRef{Suite: suite.ID, Test: state.test.ID}, ID: assertion.ID, Outcome: outcome})
		}
		outcome := protocol.OutcomePass
		if state.failed {
			outcome = protocol.OutcomeAssertionFail
			rec.Failed = append(rec.Failed, native)
			rec.Outcome = "fail"
		}
		emit("test-end", state.test, "", outcome)
		rec.Completed = append(rec.Completed, native)
	}
	emit("suite-end", nil, "", "")
	return rec, nil
}

func suiteAssertions(suite *tdd.Suite) map[string]*tdd.Assertion {
	out := map[string]*tdd.Assertion{}
	for ti := range suite.Tests {
		for ai := range suite.Tests[ti].Assertions {
			assertion := &suite.Tests[ti].Assertions[ai]
			out[assertion.ID] = assertion
		}
	}
	return out
}

// tsResolveWitnessSite maps a witness site (compiled entry path:line:col)
// back to the frozen source position through the verified source maps.
func tsResolveWitnessSite(site string, entries tsEntryIndex, maps map[string]tsSourceMap) (string, int64, bool) {
	match := tsSitePattern.FindStringSubmatch(site)
	if match == nil {
		return "", 0, false
	}
	line, err := strconv.ParseInt(match[2], 10, 64)
	if err != nil {
		return "", 0, false
	}
	column, err := strconv.ParseInt(match[3], 10, 64)
	if err != nil {
		return "", 0, false
	}
	file := match[1]
	source, ok := entries[file]
	if !ok {
		return "", 0, false
	}
	sourceMap, ok := maps[source]
	if !ok {
		return "", 0, false
	}
	srcLine, _, ok := sourceMap.SourcePosition(line, column)
	if !ok {
		return "", 0, false
	}
	return source, srcLine, true
}

// tsNativeIdent is one registered native test identity of an entry file.
type tsNativeIdent struct {
	name     string
	parentID int64
}

// tsNativePath reconstructs the full parent/child name path of one native
// test identity from its entry file's enqueue registry. The walk is bounded
// by the registry size; cycles fail closed.
func tsNativePath(ident map[int64]tsNativeIdent, id int64) ([]string, bool) {
	var reversed []string
	seen := map[int64]bool{}
	for current := id; ; {
		entry, ok := ident[current]
		if !ok {
			return nil, false
		}
		if seen[current] {
			return nil, false
		}
		seen[current] = true
		reversed = append(reversed, entry.name)
		if entry.parentID == 0 || entry.parentID == current {
			break
		}
		current = entry.parentID
	}
	path := make([]string, len(reversed))
	for i, name := range reversed {
		path[len(reversed)-1-i] = name
	}
	return path, true
}

// tsNativePathByName resolves an identity by its unique registered name when
// an event carries no usable native id; duplicated names fail closed.
func tsNativePathByName(ident map[int64]tsNativeIdent, name string) ([]string, bool) {
	var found []string
	for _, entry := range ident {
		if entry.name != name {
			continue
		}
		if found != nil {
			return nil, false
		}
		path, ok := tsNativePath(ident, tsKeyOfIdent(ident, entry))
		if !ok {
			return nil, false
		}
		found = path
	}
	return found, found != nil
}

func tsKeyOfIdent(ident map[int64]tsNativeIdent, want tsNativeIdent) int64 {
	for id, entry := range ident {
		if entry == want {
			return id
		}
	}
	return 0
}

// buildTypeScriptRunEnv builds the closed native environment: private
// HOME/TMPDIR, UTC, no color, minimal PATH, declared suite variables only;
// loader injection, native test-context and custody transport variables are
// never inherited.
func buildTypeScriptRunEnv(scratch, nodeDir string, extra []tdd.EnvironmentVar) ([]string, error) {
	env := []string{
		"HOME=" + filepath.Join(scratch, "home"),
		"TMPDIR=" + filepath.Join(scratch, "tmp"),
		"TZ=UTC",
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
		"NO_COLOR=1",
		"PATH=" + nodeDir + string(os.PathListSeparator) + "/usr/bin" + string(os.PathListSeparator) + "/bin",
	}
	for _, item := range extra {
		if !tsEnvNamePattern.MatchString(item.Name) {
			return nil, fmt.Errorf("INVALID_SCHEMA: declared environment name %q is not a closed variable name", item.Name)
		}
		if tsDroppedEnvKeys[item.Name] || strings.HasPrefix(item.Name, "MACHINERY_") {
			continue
		}
		env = append(env, item.Name+"="+item.Value)
	}
	sort.Strings(env)
	return env, nil
}

// validateTypeScriptSuite validates the closed suite shape: node-native
// identities bound to declared files, typed helper call sites at the exact
// frozen lines, and the frozen assertion helper identity.
func validateTypeScriptSuite(suite *tdd.Suite, sources map[string][]byte) error {
	if len(suite.Files) == 0 || len(suite.Tests) == 0 {
		return fmt.Errorf("INVALID_SCHEMA: suite %s declares no files or no tests", suite.ID)
	}
	seen := map[string]bool{}
	for _, test := range suite.Tests {
		if test.Native.Package != "" || test.Native.Module != "" || test.Native.Method != "" || test.Native.Name != "" || test.Native.File != "" || test.Native.Class != "" {
			return fmt.Errorf("UNSUPPORTED_FEATURE: test %s carries a non-node native identity", test.ID)
		}
		if test.Native.Source == "" || len(test.Native.Path) == 0 {
			return fmt.Errorf("INVALID_SCHEMA: test %s lacks its node native source/path identity", test.ID)
		}
		for _, element := range test.Native.Path {
			if element == "" {
				return fmt.Errorf("INVALID_SCHEMA: test %s declares an empty native path element", test.ID)
			}
		}
		key := tsIdentityKey(test.Native.Source, test.Native.Path)
		if seen[key] {
			return fmt.Errorf("DUPLICATE_TEST: suite %s declares the native identity %q twice", suite.ID, strings.Join(test.Native.Path, " > "))
		}
		seen[key] = true
		if _, ok := sources[test.Native.Source]; !ok {
			return fmt.Errorf("INVALID_SCHEMA: test %s binds to undeclared source %s", test.ID, test.Native.Source)
		}
		if test.Native.Line <= 0 {
			return fmt.Errorf("INVALID_SCHEMA: test %s native line must be positive", test.ID)
		}
	}
	assertionIDs := map[string]bool{}
	for _, test := range suite.Tests {
		if _, ok := sources[test.Source]; !ok {
			return fmt.Errorf("INVALID_SCHEMA: test %s binds to undeclared source %s", test.ID, test.Source)
		}
		lines := strings.Split(string(sources[test.Source]), "\n")
		for _, assertion := range test.Assertions {
			if assertion.Helper != tdd.AssertionHelperV1 {
				return fmt.Errorf("UNSUPPORTED_FEATURE: assertion %s uses helper %q; only %s is closed", assertion.ID, assertion.Helper, tdd.AssertionHelperV1)
			}
			if assertion.Source != test.Source {
				return fmt.Errorf("INVALID_SCHEMA: assertion %s binds to source %s outside its test's source %s", assertion.ID, assertion.Source, test.Source)
			}
			if !tsAssertionIDPattern.MatchString(assertion.ID) {
				return fmt.Errorf("INVALID_SCHEMA: assertion identity %q is not a closed ID", assertion.ID)
			}
			if assertionIDs[assertion.ID] {
				return fmt.Errorf("DUPLICATE_TEST: assertion %s is registered twice", assertion.ID)
			}
			assertionIDs[assertion.ID] = true
			if assertion.Line <= 0 || assertion.Line > int64(len(lines)) {
				return fmt.Errorf("ASSERTION_MISMATCH: assertion %s line %d is outside its frozen source", assertion.ID, assertion.Line)
			}
			line := lines[assertion.Line-1]
			if !strings.Contains(line, "check(") || strings.Contains(line, "// check(") {
				return fmt.Errorf("ASSERTION_MISMATCH: assertion %s line %d is not a typed helper call site", assertion.ID, assertion.Line)
			}
		}
	}
	return nil
}
