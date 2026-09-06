---
id: MAC-yig6
title: "Keep checker test fixtures protocol-correct under Go coverage"
status: open
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T01:17:06Z
created_by: ramirosalas
updated_at: 2026-09-06T01:17:06Z
content_hash: "sha256:f39236f9388d43abd3f9d99c46238d6c422fe87b84ee32df55a0f45bfa5aa0a3"
---

## Description
## USER INTENT
Contributors must be able to collect real Go coverage without instrumentation corrupting the checker test protocol or making a valid control fail for the wrong reason.

## Context (Embedded)
Confirmed test-harness defect on unchanged main 497419ab4512fcff765cd5feb27aed4c67b5608d, Go 1.27.1 darwin/arm64. The same actual TestVerifyCheckersReproducible passes without coverage (1.203s package) and fails with -coverprofile (0.747s package), reporting OCI RepoDigests response has trailing data. No skips/timeouts. The existing test uses a test-executable OCI protocol fixture, not a real Docker daemon; this is not real OCI lifecycle evidence.
Root report /tmp/machinery-coverage-diagnostic.hKOuC4/REPORT.md SHA256 ac0be40f54a71ae69a73f0fbb87040c5c0a6cf31f86a7fa849a6afbaeba17958 includes exact commands. Native raw baseline-native.jsonl SHA256 a45efe19416bde01952e5e99b1d25f387bf9e4337887ee5903b7bbbfc12c2f16; covered baseline-coverage.jsonl SHA256 62fba1cd8071b74743cbd8d6a58be79212195c7e6ebf70343a5b9e8c2ec80b11, all in that directory.
An actual go test -c -cover child inspection emitted correct three-value JSON stdout (131 bytes), plus 54-byte stderr: warning: GOCOVERDIR not set, no coverage data emitted. This establishes the one paired failure's cause. Other failures in a broad MAC-sh60 coverage run are not individually attributed here. Current CI/nightly/preflight use ordinary/race tests without -coverprofile; no current CI-gate failure from this cause is established.

## Root Cause
cmd/machinery/verify_checkers_test.go writeRegistryFile and checkerFixtureEngineArgs select os.Executable(), the current test binary. TestCheckerProcessFixture dispatches runCheckerOCIEngineFixture, which emits image/platform JSON and exits. Coverage instrumentation in that child adds the warning. Production runChecker uses its intentionally closed environment, captures bounded stdout/stderr then appends stderr in deterministic order. verifyLocalOCIImage decodes exactly RepoDigests, OS and architecture then requires EOF. That strict production rejection is correct; the fixture's instrumentation/protocol boundary is not.

## Ownership
- cmd/machinery/verify_checkers_test.go: narrow test-support changes only in writeRegistryFile, checkerFixtureEngineArgs, checkerProcessFixtureCommand, TestCheckerProcessFixture and runCheckerOCIEngineFixture, plus minimal directly associated fixture support if needed. Preserve every existing test assertion and intentional fixture exit/error mode. Exact existing-helper amendment must receive independent PM review before writes; ownership is not frozen-test amendment authorization.
- cmd/machinery/checker_fixture_coverage_test.go (new): bounded real compiled/instrumented/noninstrumented subprocess regression and protocol-sensitivity tests; any new private helper implementation belongs here when possible.
You are not alone; preserve other edits. No edits to production verify_checkers.go, processcontrol, deterministic environment, shared golden helpers, workflows or coverage settings. No duplicate ownership with MAC-sh60 or the real-Docker lifecycle story MAC-yhg5. If an actual proposed test-support fix needs another path or altered assertions, return the precise proposal for independent review first.

## Boundary Map
PRODUCES:
- cmd/machinery/verify_checkers_test.go -> coverage-safe existing checker fixture invocation, retaining actual protocol semantics, bounded real execution and existing test assertions.
- cmd/machinery/checker_fixture_coverage_test.go -> paired covered/noncovered regression and strict-protocol controls.
CONSUMES:
- Existing checker fixture helpers and dispatcher.
  source: writeRegistryFile(t *testing.T, body string) string; checkerFixtureEngineArgs(t *testing.T, mode string) string; checkerProcessFixtureCommand(t *testing.T, mode string) []string; TestCheckerProcessFixture(t *testing.T); runCheckerOCIEngineFixture(wrongDigest, wrongPlatform bool), verify_checkers_test.go.
- Existing unchanged production protocol.
  spec: verifyLocalOCIImage(engineArgs []string, image, digest, platform string, timeout time.Duration, workDir string) error; runChecker(args []string, timeout time.Duration, workDir string) (string, error), verify_checkers.go.
- Existing actual CLI test control.
  source: TestVerifyCheckersReproducible at verify_checkers_test.go:478-507; current goldenBin/runBin infrastructure remains unchanged.

### Story Acceptance Criteria
1. The original real TestVerifyCheckersReproducible succeeds with unchanged intended assertions both without instrumentation and with an actual nonempty -coverprofile on the same delivered revision. It reaches reproduced-checker/config/summary assertions, not an earlier protocol failure; coverage remains enabled for the package under test.
2. The test-support boundary produces the expected inspection protocol without incidental Go coverage diagnostics contaminating it, including when the parent test executable is coverage-instrumented and ordinary production closed-environment execution invokes the fixture. Exercise the actual fixture invocation path with real child processes and separate stdout/stderr capture. A test-only solution must not add production environment exceptions, suppress arbitrary output, ignore child failures or silently discard the requested package coverage.
3. Strict production checks remain unchanged and demonstrably sensitive: matched real fixture controls accept correct digest/platform and reject wrong digest, wrong platform, and deliberate extra protocol data with the intended distinct diagnosis in both covered and noncovered runs. Do not mistake instrumentation trailing-data rejection for the wrong-digest/platform negative. Preserve deterministic stream ordering, output bounds, timeouts and existing fixture modes.
4. All required runs terminate with exact selected inventory, zero skips and bounded cleanup of owned temporary binaries/coverage files/processes. Native Darwin and Linux execution coverage is required for this service-free Go/subprocess harness; record actually tested host/toolchain and do not claim Windows runtime or Docker lifecycle assurance from it. No installed binary changes, Docker dependency, Paivot product dependency or arbitrary external runtime is introduced.
5. Independently reviewed RED proves the existing instrumentation defect with a matched passing noncoverage control; GREEN proves the same frozen tests and existing checker-focused regression controls pass. Report native/covered commands, SHAs, terminal outcomes, timing, raw logs and real coverage-profile validity. Preserve all unrelated source/test/evidence bytes and honestly report any separately discovered failure.

## Testing Requirements
Hard TDD, test-support bug: independently authorize exact existing-helper amendments before GREEN. RED adds only the new test file using existing callable interfaces/actual compiler and subprocesses; compile/API/fixture setup errors are not behavioral proof. The existing parent-provided unchanged-source pair is diagnosis, not substitute for review/freeze of the new regression. Do not build a recursive test invocation loop: nested runs select only the exact existing target/control leaves, excluding the outer coverage regression.
Unit plus Integration tests: MANDATORY (no mocks of compiler/process/output or fabricated outcomes). Existing OCI fixture is expressly a protocol fixture; no claim it validates real Docker. Actual Go-built test executables, actual filesystem and production command path; no skip-if-missing. New protocol-negative inputs may be constructed as deliberate fixture data, not returned fake test outcomes.
Minimum paired commands from the story worktree:
- go test -json -count=1 -timeout=60s ./cmd/machinery -run '^TestVerifyCheckersReproducible$'
- go test -json -count=1 -timeout=60s -coverprofile=<owned-temporary-directory>/coverage.out ./cmd/machinery -run '^TestVerifyCheckersReproducible$'
Also run the new coverage regression and selected existing checker protocol/stream/bounds controls on the same SHA with and without instrumentation. Use real compiled covered and noncovered binaries where required to isolate child protocol; verify requested profile parses and has instrumented statement blocks, not only file existence. A helper-only uninstrumented build is not global package-coverage disabling, but any such choice must explicitly report that helper's instrumentation boundary and preserve the covered caller/production observations; do not silently claim helper coverage.
Use explicit bounded deadlines; initial outer meta-test forecast 180s to allow native/covered builds and bounded child controls, measure actual cost. Preserve existing per-operation bounds. No full scripts/preflight.sh, remote mutations, installed assets or live container changes.

## Discovered During
MAC-sh60 independent review coverage diagnostic; separately reproduced on unchanged main. This story does not change MAC-sh60 scope, proof or claim. No runtime lane dependency: service-free subprocess/Go coverage support. Final capstone MAC-ou97 depends on this repair so the confirmed defect cannot be lost.

## OUT OF SCOPE
- Production OCI parser/identity/env changes: correct strict behavior is preserved.
- Real daemon cleanup/budgets (MAC-yhg5), contributor runtime provisioning (MAC-hpqp), oracle coverage semantics (MAC-sh60).
- General coverage architecture, global coverage disabling, warning filters, broad harness rewrites, Windows runtime expansion or attribution of unpaired failures.
- Full preflight and release operations remain final-epic-only.

## DIFF BUDGET
- Forecast 2 test files, approximately 250-400 changed LOC; report actual additions/deletions and native/covered runtime. Material overrun requires review, not trimmed controls.
- Existing helper changes must remain narrow; test proof may dominate cost because it compiles and executes both instrumented/noninstrumented paths.

## MANDATORY SKILLS
- developer; codebase-memory; pm_acceptor.

## Delivery Requirements
Use supported story delivery only after reviewed RED and independent GREEN evidence. Append exact SHA/commands/raw output/proof; no claim or state changes by this triage.

## nd_contract
status: new

### evidence
- Created by Sr PM bug_triage from parent-confirmed unchanged-main paired runtime defect; source and raw hashes verified, no implementation performed.
- Graph generation 2026-09-05T23:58:53Z reports metadata_match/no recorded gap for verify_checkers.go and verify_checkers_test.go, best effort only; exact committed source inspected.
- Scope deliberately excludes production/parser and MAC-sh60 changes; independent helper/test amendment review remains pending.

### proof
- [ ] AC #1: actual native/covered existing success control and valid coverage profile.
- [ ] AC #2: clean fixture protocol at the real instrumented boundary without production bypass.
- [ ] AC #3: exact strict negative controls in both modes.
- [ ] AC #4: bounded native Darwin/Linux execution and owned-resource cleanup.
- [ ] AC #5: independently frozen RED/GREEN and truthful scoped evidence.


## Acceptance Criteria


## Design


## Notes


## History


## Links
- Parent: [[MAC-ui8a]]

## Comments
