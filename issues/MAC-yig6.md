---
id: MAC-yig6
title: "Keep checker test fixtures protocol-correct under Go coverage"
status: open
priority: 0
type: bug
labels: [hard-tdd, red-approved]
parent: MAC-ui8a
created_at: 2026-09-06T01:17:06Z
created_by: ramirosalas
updated_at: 2026-09-06T04:08:21Z
content_hash: "sha256:5df24ebc04edfe59ec659cd7f329226d8b376166b2402b0bcfc31df3909c5b49"
blocks: [MAC-ou97]
assignee: dev-MAC-yig6
follows: [MAC-2u36, MAC-a89e]
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
# MAC-yig6 independent RED review

Decision: APPROVED RED ONLY. `pvg story approve-red MAC-yig6` completed using the shared live vault. This is not final acceptance. Existing-helper amendment authorization is HELD pending independent review of an exact, unapplied textual patch. Native Linux GREEN execution remains mandatory.

Reviewed commit c59c89de31c7f1268a220caf0fa0d67a8e166d0f, base 70652b948bf090008b1965c85daf36ea374daea4, commit subject includes tdd-red. Detached review checkout: /tmp/machinery-yig6-pm.P55qWO/checkout. Diff is exactly one new test file, 246 additions/0 deletions. Frozen cmd/machinery/checker_fixture_coverage_test.go SHA256 80b590315a79f4faa3c9741eb1347e6070d3b01046338c76ce6e7000d6e92677. No implementation or source/test amendment made in review.

Author report /tmp/machinery-yig6-red.09ui51/REPORT.md SHA256 648c4d87d5facac1d46110ab796436a8a7a19ebeceadb15b8e8a53edbc4e6eb8 and HELPER-AMENDMENT-PROPOSAL.md SHA256 555a3ac33a1a1b5023d3f378e56719f4dee7707c3cb7b0657dab5471ddc53497 read completely and verified. Canonical pvg nd story, full append-only evidence/history, committed original helpers, strict parser/runner/environment, TestMain ownership, golden harness and frozen tests reviewed. Author raw JSONL/profile hashes match recorded evidence. Independent replay follows.

## Reproduction and exact commands

The executable script beside this report, replay.sh SHA256 4acc08fffef9edc190fd8d625c9ec03e5cfad364fdbe236c58bd15a700a0e357, records every complete command, selector, artifact path and timeout. It pins the detached checkout. All commands synchronous; /usr/bin/time -p plus Perl alarm 200s for meta runs, 90s for direct tests, 30s for profile parsing. Test deadlines are 180s outer and 60s children/builds. Actual host Go 1.27.1 darwin/arm64, GOFLAGS empty.

| Run | Exit | Exact terminal outcome | Package / wall seconds |
| --- | --- | --- | --- |
| meta-native | 1 | Native compiled child 10 leaves PASS; covered child 5 PASS/5 FAIL; outer native PASS, covered FAIL | 6.549 / 8.56 |
| original-native | 0 | Original unchanged TestVerifyCheckersReproducible 1 PASS | 0.943 / 1.31 |
| original-covered | 1 | Original 1 FAIL, actual strict trailing-data assertion | 0.717 / 1.14 |
| selection-native | 0 | 10 leaves PASS, plus protocol parent PASS | 2.035 / 2.40 |
| selection-covered | 1 | Direct 5 PASS/5 FAIL; outer native PASS, covered FAIL; JSON test terminal totals 6 PASS/8 FAIL including parent suites | 6.929 / 7.32 |
| cover-original / cover-selection | 0 / 0 | Actual Go profile parser succeeds | 0.17 / 0.17 wall |

Meta elapsed: native-parent outer 6.205789583s (native 2.81s, covered 3.39s); covered-parent outer 5.173448917s (native 2.63s, covered 2.55s). Selected top-level inventory is TestVerifyCheckersReproducible, TestCheckerFixtureProtocolControls, TestRunCheckerBoundsOutput, TestRunCheckerReportsStreamsInDeterministicOrder, TestRunCheckerBoundsDescendantPipeWait, TestRunCheckerTimeoutDiagnostic, TestVerifyLocalOCIImageBoundsUnresponsiveEngine. Protocol children are correct, wrong-digest, wrong-platform, extra-data. No recursion; compiled child selectors exclude the outer meta. No skipped, setup/compile/import/deadline-failure leaves. Compiler stdout/stderr clean. Outer command stderr files contain timing only.

Raw actual child stdout is 131 bytes for ordinary controls and 153 bytes with deliberate fourth JSON. Native stderr is empty. Covered fixture stderr is exactly 54 bytes: `warning: GOCOVERDIR not set, no coverage data emitted\n`. Both compiled modes preserve intended input identity/platform. Native correct is accepted; native wrong-digest reaches `do not contain exact reference`; wrong-platform reaches `does not match required platform`; deliberate fourth JSON reaches trailing-data rejection. Covered wrong-digest/platform are UNREACHED because instrumentation fails the preceding EOF check; these are not credited as successful sensitivity negatives. Covered extra-data also fails incidental-stderr assertion despite reaching trailing-data diagnosis. Five unchanged stream/bounds/timeout leaves pass in both modes.

## Coverage and constraints

Both profiles have 2541 blocks/4015 statements. Original executes 397 statements, 361 in verify_checkers.go, 9.9%. Selection executes 402 statements, 366 in verify_checkers.go, 10.0%. Independent awk summation and actual go tool cover parsing agree with frozen meta assertions. Both profiles are byte-identical to author profiles. Covered production execution is real; existing ordinary golden CLI behavior is not claimed as instrumented subprocess coverage.

AC1–3: tests require original success assertions, clean actual helper streams, exact raw protocol values, strict distinct negative diagnoses, production stream ordering and existing bounds. With unchanged production and preserved assertions, these are sufficient RED behavioral constraints. AC4: required exact inventories, deadlines and owned cleanup are explicit; successful native Linux GREEN execution still required in addition to this Darwin replay. AC5: frozen tdd-red commit and independently reproduced matched native PASS/covered FAIL establish RED. GREEN must pass same frozen tests plus existing checker-focused regressions on one delivered SHA and preserve all existing assertions/modes.

Static verification: no incomplete markers in new test file; pvg verify explicit file --include-tests --format text PASS (1 file, 0 issues); pvg story verify-tdd --base 70652b948bf090008b1965c85daf36ea374daea4 --json PASS (1 commit, violations null); git diff --check PASS; detached checkout clean. Delivery shape is not behavioral proof. No documentation behavior changes in RED.

Codebase-memory Verify tier: project Users-ramirosalas-workspace-machinery, generation 2026-09-06T02:42:16Z. Exact helper discovery and bidirectional depth-1 fixture-engine trace read; complete relevant pages. Existing reviewed paths metadata_match/no recorded gap; frozen new file missing in main graph, read fully from exact commit. Graph is best-effort only; conclusions rely on committed source and actual replay.

## Existing-helper amendment HELD

The proposed three replacements of initial os.Executable/error blocks in writeRegistryFile, checkerFixtureEngineArgs and checkerProcessFixtureCommand are a reasonable narrow boundary. The sync.Once real uninstrumented helper build under unchanged cmdTestControlRoot is also sound in principle and preserves the requesting binary's production coverage. However, the proposal specifies roughly 50–70 new support lines in prose. It does not supply an exact textual patch for independent inspection of cache error/cancellation handling, runtime.Caller resolution, compiler environment, temporary ownership and bounded execution. No existing-file write or [test-edit-authorized] permission is granted by this RED approval.

Required next step: fresh GREEN implementer prepares an unapplied patch outside the checkout. PM reviews its exact bytes/hash before shared existing-helper writes. Scope remains solely the three selectors and minimal support in verify_checkers_test.go; frozen new test file, TestMain, golden helpers, TestCheckerProcessFixture, runCheckerOCIEngineFixture, all existing arguments/markers/modes/assertions/exits, production/parser/environment/workflows/coverage settings remain unchanged. No blanket authorization.

## Artifacts and resource state

All artifacts below live in /tmp/machinery-yig6-pm.P55qWO and remain as review evidence. Test temporary trees were removed by their existing owners; both meta control roots named in raw command logs were independently checked absent. Process inventory contained only the inspecting shell/rg, no owned live checker/build process. Test/build session ended. Review checkout retained clean for traceability. No fetch/pull/push/sync/GH, Docker/service, setup/recover, full preflight, installed asset or Paivot product dependency operations. Installed /Users/ramirosalas/.local/bin/machinery SHA256 remains 5205883aaa4276d7eb6edb25b6ad43ac39a04bcb9a8b5ee55498127b04950849. No Linux, Windows runtime or Docker lifecycle assurance claimed.

SHA256:
- meta-native.stdout: e5eea4caf40c7b35c98571f8c78089f9dbf75e2b41967e363c7e231458ea541f
- original-native.stdout: 5940dd399a0de2e88b3b8cb78bc57329a92e95cb522c216e44409b172ed4b78a
- original-covered.stdout: c48748774c6b08890b7708051f4bd5d26deebdde436c6fd6f582a4346b8dbc4f
- selection-native.stdout: e689c932698e83328b2652d8e21e7b800cb8e15c408fa870a78d67a1cd885b3a
- selection-covered.stdout: 745f05cc9973dd64970d9b650e741602ec8d15deea39dddf7113e3ad69a26240
- original-coverage.out: b6c192eddd83c4aec6a1d01a8d558bb4f4521c79e057265b71d7e454af67fa4a
- selection-coverage.out: d2b22bcd3ca5745eea84b2ac88d720e1a97c530f7b09c31c0d896fc9b33997e5
- meta-native.stderr: 4b5632b917e1b7296bc01054125123884ce3ac8322eb9280925591a1bba76b21
- original-native.stderr: e68a5e318f60bf24947f1145d44df675434baa0e11c26bf07cc96f6e1859d8ad
- original-covered.stderr: 328baf88b9810cf9e1798c162d2d5dd256d90aaee4c0d9cc9810af74d06b8466
- selection-native.stderr: 5f05b015af94f3cba917c4659f830a0f732eed6883ddba33b5565f917a75c22a
- selection-covered.stderr: de12e7e811d0f63d0160a2fa275912bf3fd0a3be6865247fd5f42cf2bdd6317e

LEARNINGS: A failing wrong-identity test proves nothing about identity sensitivity when a prior protocol EOF check masks it. Independent same-SHA profile equality corroborates the narrow diagnosis. Exact helper authorization requires concrete implementation bytes even when a prose design boundary is sound.


## nd_contract
status: red-approved

### evidence
- RED tests approved via pvg story approve-red on 2026-09-05.

### proof
- [ ] GREEN developer must implement against the approved RED tests without modifying them.


## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## RED delivery evidence

PROOF:
- RED commit c59c89de31c7f1268a220caf0fa0d67a8e166d0f on story/MAC-yig6; new frozen file SHA25680b590315a79f4faa3c9741eb1347e6070d3b01046338c76ce6e7000d6e92677, 246 additions/0 deletions, no existing helper/source edits.
- Full exact commands, selected inventory, raw paths, classifications, timings and AC mapping are in the appended REPORT.md and /tmp/machinery-yig6-red.09ui51/REPORT.md SHA256648c4d87d5facac1d46110ab796436a8a7a19ebeceadb15b8e8a53edbc4e6eb8.
- Actual Go1.27.1 darwin/arm64: unchanged original native1 PASS (package0.998s) / covered1 FAIL (package0.800s) due strict OCI trailing-data failure. Paired compiled native10 terminal leaves PASS / covered5 PASS5 FAIL. Covered identity/platform negative diagnoses explicitly UNREACHED; no credit for masked negatives. Zero skips/setup failures/deadline failures. Five unchanged stream/bounds/timeout leaves pass in both modes.
- Frozen outer meta native child PASS2.99s and covered child FAIL3.47s, total6.46s; under covered parent native child PASS2.69s/covered FAIL2.55s, total5.24s. Real profile2541 blocks/4015 statements/402 executed (366 verify_checkers.go),10.0%; direct original profile397 executed (361 production),9.9%. Go cover parser accepted both.
- Raw SHA256: meta-native.jsonl836213a078c83cbd1142df1859b7e7f38e28f3e2b98bf2d2fd4246f8a511526b; original-native.jsonl a8b4bc7717e17cf293178b76422d64cc2995cd3a27b5239b6aac9ad280f7578a; original-covered.jsonl fb085049e5ed5bd7e22191e96917c0d371d1bfbd990e59e1814e6b85c89cd04d; selection-native.jsonl7ddc99ad818c53ac80e68a354459df4b8837b6ee4356d99cbc7c665f02c01bdf; selection-covered.jsonl5c3ed8845d3055db8a21bdab0b81303f246684a968c8d9699a6b89be7a57cae8. All reside beside REPORT.md.
- Original coverage profile SHA256b6c192eddd83c4aec6a1d01a8d558bb4f4521c79e057265b71d7e454af67fa4a; selection profile d2b22bcd3ca5745eea84b2ac88d720e1a97c530f7b09c31c0d896fc9b33997e5.
- pvg verify explicit new path --include-tests --format text PASS1file0issues; verify-tdd base70652b9 PASS1commit0violations. Clean worktree, all test/compiler sessions terminated, no owned live checker process. Installed binary hash unchanged. No remote/install/Docker/full-preflight operations.
- Pending helper review document /tmp/machinery-yig6-red.09ui51/HELPER-AMENDMENT-PROPOSAL.md SHA256555a3ac33a1a1b5023d3f378e56719f4dee7707c3cb7b0657dab5471ddc53497 specifies only three existing executable selectors plus directly associated private compiler/cache helper. No amendment applied. Independent PM must approve exact amendment before GREEN writes.

LEARNINGS:
- Coverage stderr crosses strict OCI EOF verification even when stdout remains valid; merely asserting rejection would falsely credit masked identity/platform negatives.
- Validate executed production statement blocks as well as profile syntax; actual instrumentation must remain on the caller.
- Any future once-built helper must live in the TestMain-owned root, not an individual test's temporary directory; its intentionally uninstrumented coverage boundary must be reported.

## nd_contract
status: delivered

### evidence
- RED phase only, frozen SHA c59c89de31c7f1268a220caf0fa0d67a8e166d0f; report and raw artifacts above; no implementation or acceptance claimed.
- Native real controls pass; covered real controls fail for observed54-byte Go coverage stderr contamination, with real nonempty coverage profile and zero skipped/deadline/setup failures.
- pvg verify PASS and verify-tdd PASS; no existing test/production change; exact helper proposal awaits independent review.

### proof
- [x] AC #1 RED: unchanged original success assertions pass natively and fail behaviorally under actual coverage on same revision; GREEN success pending.
- [x] AC #2 RED: separate actual stdout/stderr under closed production environment proves instrumentation contamination; GREEN cleanliness pending.
- [x] AC #3 RED: native four-control sensitivity observed; covered wrong-digest/platform marked UNREACHED rather than credited; frozen outcome tests demand intended diagnoses.
- [ ] AC #4 FINAL: Darwin bounded execution complete, native Linux execution remains required; no Docker/Windows assurance claimed.
- [ ] AC #5 FINAL: RED frozen and paired proof complete; independent RED/helper amendment approval and GREEN evidence pending.

# MAC-yig6 exact GREEN helper amendment proposal — not applied

Frozen RED: c59c89de31c7f1268a220caf0fa0d67a8e166d0f. Independent PM authorization is required before any existing helper edit. The RED file remains byte-for-byte frozen.

Proposed existing-file scope: cmd/machinery/verify_checkers_test.go only.

1. In writeRegistryFile, checkerFixtureEngineArgs, and checkerProcessFixtureCommand, replace only the initial os.Executable/error block with `executable := checkerFixtureExecutable(t)`. Retain every argument, marker, mode, JSON encoding, registry augmentation, assertion, exit and error behavior after those initial blocks.
2. Add the `sync` standard-library import and directly associated private support: a sync.Once-protected cached helper executable path/build error and `checkerFixtureExecutable(t *testing.T) string` beside the existing fixture helpers.
3. The private helper returns os.Executable normally when testing.CoverMode() is empty. Under coverage it uses the real Go compiler to build this same package's test executable with `go test -c -cover=false -coverpkg= -o <owned-private-directory>/checker-fixture.test .`. Resolve package source directory from runtime.Caller. Use a directory beneath the existing cmdTestControlRoot so unchanged TestMain owns cleanup after all tests; do not cache a path under the first test's t.TempDir. Compile synchronously with a 60-second context derived from the calling test, capture real stdout/stderr, and fail the requesting test on compiler failure or deadline. Never return an installed binary, fall back to contaminated self-execution, or ignore build failure. The cache avoids repeated compilation in a covered package run.
4. Do not change TestCheckerProcessFixture or runCheckerOCIEngineFixture: the uninstrumented helper executes their existing source and all existing modes. Do not change TestMain, golden helpers, production, deterministic environment, parser, workflows, coverage settings, or RED assertions.

Coverage boundary: only the protocol fixture executable is intentionally uninstrumented. The requesting test binary and production functions it directly calls remain actually instrumented, and requested package coverage is still emitted and validated. Ordinary goldenBin already builds an ordinary CLI; this proposal neither changes that existing infrastructure nor claims helper-executable coverage. The compiled helper remains a protocol fixture, with no Docker claim.

Forecast: approximately 50–70 additions and 12 deletions in the existing test support file, total story approximately 310–330 changed LOC. GREEN must measure actual cost and pass the frozen RED plus unchanged checker-focused controls in native Darwin and native Linux execution. Cross-compilation alone does not meet Linux proof.

This is a reviewable proposed implementation boundary, not authorization or an implementation commit. If PM requires exact textual patch approval rather than the specified statements/behavior, the GREEN agent must prepare that patch for independent review before applying it.

# MAC-yig6 frozen RED evidence

Revision c59c89de31c7f1268a220caf0fa0d67a8e166d0f, story/MAC-yig6, base 70652b948bf090008b1965c85daf36ea374daea4. One new file, 246 additions/0 deletions. Frozen file cmd/machinery/checker_fixture_coverage_test.go SHA256 80b590315a79f4faa3c9741eb1347e6070d3b01046338c76ce6e7000d6e92677. No existing test/helper/production changes.

Actual host/toolchain: Go 1.27.1 darwin/arm64. GOFLAGS empty. Installed machinery SHA256 remained 5205883aaa4276d7eb6edb25b6ad43ac39a04bcb9a8b5ee55498127b04950849. No Linux execution claimed. No Docker daemon, installed-asset operation, full preflight or remote operation occurred.

All shell commands ran after `cd /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-yig6`. Each test command used `/usr/bin/time -p perl -e 'alarm N; exec @ARGV'` before `go`: N=200 for outer-meta commands, N=90 for ordinary targeted commands. Standard output and stderr/timing were recorded separately under this report directory. Tests themselves used 180s outer deadlines and 60s child/test/compiler bounds; production controls retained 5s or existing 100ms operation limits.

## Exact commands and outcomes

- `go test -json -count=1 -timeout=180s ./cmd/machinery -run '^TestCheckerFixtureCoverageRegression$'`: exit 1, behavioral RED. Native child 10 terminal leaves pass; covered child 5 pass/5 fail. Outer native subtest PASS2.99s; covered FAIL3.47s; meta6.46s; package6.964s; wall8.87s. Files meta-native.jsonl/meta-native.stderr.
- `go test -json -count=1 -timeout=60s ./cmd/machinery -run '^TestVerifyCheckersReproducible$'`: exit0, 1 pass/0 fail/0 skip, original unchanged assertions all completed. Package0.998s, wall1.40s. Files original-native.jsonl/original-native.stderr.
- `go test -json -count=1 -timeout=60s -coverprofile=/tmp/machinery-yig6-red.09ui51/original-coverage.out ./cmd/machinery -run '^TestVerifyCheckersReproducible$'`: exit1, 0 pass/1 fail/0 skip, assertion at verify_checkers_test.go:486 fails because strict OCI parser rejects trailing data before subsequent reproduced/config/summary assertions. Package0.800s, wall1.23s. Files original-covered.jsonl/original-covered.stderr.
- `go test -json -count=1 -timeout=60s ./cmd/machinery -run '^(TestCheckerFixtureProtocolControls|TestVerifyCheckersReproducible|TestRunCheckerBoundsOutput|TestRunCheckerReportsStreamsInDeterministicOrder|TestRunCheckerBoundsDescendantPipeWait|TestRunCheckerTimeoutDiagnostic|TestVerifyLocalOCIImageBoundsUnresponsiveEngine)$'`: exit0, 10 terminal leaves pass/0 fail/0 skip; 11 terminal test events including protocol parent. Wall2.45s. Files selection-native.jsonl/selection-native.stderr.
- `go test -json -count=1 -timeout=180s -coverprofile=/tmp/machinery-yig6-red.09ui51/selection-coverage.out ./cmd/machinery -run '^(TestCheckerFixtureCoverageRegression|TestCheckerFixtureProtocolControls|TestVerifyCheckersReproducible|TestRunCheckerBoundsOutput|TestRunCheckerReportsStreamsInDeterministicOrder|TestRunCheckerBoundsDescendantPipeWait|TestRunCheckerTimeoutDiagnostic|TestVerifyLocalOCIImageBoundsUnresponsiveEngine)$'`: exit1, expected RED. Direct leaves5 pass/5 fail; outer meta native PASS2.69s/covered FAIL2.55s; meta5.24s; wall7.37s. JSON terminal events6 pass/8 fail including parent suites; zero skips. Files selection-covered.jsonl/selection-covered.stderr.
- `go tool cover -func=/tmp/machinery-yig6-red.09ui51/original-coverage.out` and equivalent selection-coverage.out: both exit0. Original profile2541 blocks/4015 statements/397 executed, including361 executed verify_checkers.go statements, total9.9%. Selection profile2541 blocks/4015 statements/402 executed, including366 production statements, total10.0%. Frozen meta independently parses block grammar, asserts executed production statements, and asks Go to parse its own covered profile.
- `pvg verify cmd/machinery/checker_fixture_coverage_test.go --include-tests --format text`: PASS1file0issues.
- `pvg story verify-tdd --base 70652b948bf090008b1965c85daf36ea374daea4 --json`: PASS1commit0violations.
- `git diff --check`: PASS; worktree clean after commit.

## Selected leaves and diagnosis

Exact inner top-level inventory is TestVerifyCheckersReproducible, TestCheckerFixtureProtocolControls, TestRunCheckerBoundsOutput, TestRunCheckerReportsStreamsInDeterministicOrder, TestRunCheckerBoundsDescendantPipeWait, TestRunCheckerTimeoutDiagnostic, TestVerifyLocalOCIImageBoundsUnresponsiveEngine. The protocol parent contains correct, wrong-digest, wrong-platform, extra-data. Inner selectors exclude the outer meta-test, preventing recursion. The meta explicitly asserts entry of all seven top-level selections and all four protocol children and rejects any skip.

Native correct accepts expected digest/platform; wrong-digest reaches `do not contain exact reference`; wrong-platform reaches `does not match required platform`; deliberate fourth JSON reaches `OCI RepoDigests response has trailing data`. Their raw fixture stdout is131B, except deliberate extra153B; stderr0B.

Covered raw stdout contains the same intended input values, but every fixture adds54B stderr `warning: GOCOVERDIR not set, no coverage data emitted\n`. Correct is rejected early, and wrong-digest/platform are explicitly reported UNREACHED at their intended diagnosis because instrumentation triggers trailing data. Extra-data has the deliberate fourth JSON physically observed and strict trailing-data diagnosis reached, but its overall covered control still FAILS for incidental stderr; it is not credited as a clean covered sensitivity proof. All five existing stream/bounds/timeout leaves pass in native and covered runs. No compiler/import/setup/deadline error, timeout hang or skip underlies RED.

Separate raw streams are observed under the unchanged deterministic environment. A subsequent actual runChecker invocation must equal stdout followed by stderr exactly. Actual unchanged verifyLocalOCIImage then checks each control. No mock compiler/process/result, arbitrary warning filter, parser change or package coverage disabling exists in RED.

## AC mapping and remaining work

AC1: frozen outer runs original assertions in both modes; native complete, covered behavioral failure proven. GREEN pending.
AC2: frozen raw-stream and actual production checks prove contamination only in covered fixture. GREEN pending.
AC3: four frozen real controls; native diagnoses reached, covered wrong identity/platform masked and not credited. GREEN pending.
AC4: bounded Darwin execution and owned temporary cleanup complete; native Linux execution remains mandatory for final acceptance. No Windows/Docker assurance claimed. All tracked tool sessions terminated, and process inventory showed only the diagnostic shell/rg itself, no live owned checker/build process. Test binaries/profile temp trees are t.TempDir-owned; unchanged TestMain owns golden and child scratch roots. Only evidence logs/profiles remain intentionally in this private report directory.
AC5: RED frozen at above hash and behavior reproduced on same revision; independent PM RED approval and exact existing-helper amendment review pending. See HELPER-AMENDMENT-PROPOSAL.md. Full suite/preflight deferred per explicit user scope.

## Learnings and tool-use notes

- A coverage warning is fatal protocol contamination when strict stdout-plus-stderr decoding correctly requires EOF. Negative tests must prove their intended diagnosis, not merely nonzero exit.
- A real statement profile and positive production execution protect against accidentally proving a helper-only uninstrumented path.
- Existing TestMain owns a per-process temporary root; a future cached helper must use its lifetime, not the first test's temporary directory.
- Tool discovery mistakes were not behavioral evidence: a guessed wrong epic name returned unknown revision before the correct epic/MAC-ui8a history was read; `pvg issues show --help` returned usage/error before canonical JSON was read; pvg rejected documented `--format=text`, then supported separate `--format text` passed. `pvg story deliver --help` returned `OK: deliver --help ...` rather than help; reported to root for read-only investigation. No source adjustment resulted.

Codebase-memory Verify tier used supplied exact-symbol traces and refreshed status/coverage at generation2026-09-06T02:42:16Z. Existing relevant files metadata_match/no recorded gap; new file missing from main graph and read directly. Graph coverage is best-effort, not exhaustive.

## nd_contract
status: new

### evidence
- Sr PM created MAC-yig6, P0 bug, parent MAC-ui8a, open/hard-tdd/unclaimed; MAC-ou97 now explicitly depends on it. No other story state/claim/scope changed.
- Source and exact diagnostic report/raw SHA256 values verified against unchanged main 497419a. Observed native PASS 1.203s versus covered FAIL 0.747s for one identical leaf; real child stdout 131 bytes plus instrumentation stderr 54 bytes explains strict trailing-data failure. Current noncoverage CI/preflight failure is NOT claimed.
- Two test-support paths, 250-400 LOC forecast; no production parser/environment/output-policy changes. Existing-helper amendment requires independent PM authorization; this creation is not RED approval.
- Scoped backlog lint PASS: 34 issues, 0 errors/review findings. No dependency cycles. RTM PASS: 0 extracted requirements, 19 stories/3 closed; structural only, not implementation AC proof.
- No tests/source/docs edits or runtime reruns by this triage. Root clean main retained.

### proof
- [ ] AC #1: actual native/covered existing success control and valid coverage profile.
- [ ] AC #2: clean fixture protocol at the real instrumented boundary without production bypass.
- [ ] AC #3: intended strict negative controls in both modes.
- [ ] AC #4: bounded native Darwin/Linux execution and owned-resource cleanup.
- [ ] AC #5: independently frozen RED/GREEN and truthful scoped evidence.


## History
- 2026-09-06T01:17:22Z dep_added: blocks MAC-ou97
- 2026-09-06T03:36:07Z status: open -> in_progress
- 2026-09-06T03:36:07Z auto-follows: linked to predecessor MAC-2u36
- 2026-09-06T03:36:07Z claimed by dev-MAC-yig6
- 2026-09-06T03:46:40Z status: in_progress -> in_progress
- 2026-09-06T03:46:40Z auto-follows: linked to predecessor MAC-a89e
- 2026-09-06T04:06:34Z status: in_progress -> open

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-ou97]]
- Follows: [[MAC-2u36]], [[MAC-a89e]]

## Comments

### 2026-09-06T01:18:00Z ramirosalas
## nd_contract
status: new

### evidence
- Sr PM created MAC-yig6, P0 bug, parent MAC-ui8a, open/hard-tdd/unclaimed; MAC-ou97 now explicitly depends on it. No other story state/claim/scope changed.
- Source and exact diagnostic report/raw SHA256 values verified against unchanged main 497419a. Observed native PASS 1.203s versus covered FAIL 0.747s for one identical leaf; real child stdout 131 bytes plus instrumentation stderr 54 bytes explains strict trailing-data failure. Current noncoverage CI/preflight failure is NOT claimed.
- Two test-support paths, 250-400 LOC forecast; no production parser/environment/output-policy changes. Existing-helper amendment requires independent PM authorization; this creation is not RED approval.
- Scoped backlog lint PASS: 34 issues, 0 errors/review findings. No dependency cycles. RTM PASS: 0 extracted requirements, 19 stories/3 closed; structural only, not implementation AC proof.
- No tests/source/docs edits or runtime reruns by this triage. Root clean main retained.

### proof
- [ ] AC #1: actual native/covered existing success control and valid coverage profile.
- [ ] AC #2: clean fixture protocol at the real instrumented boundary without production bypass.
- [ ] AC #3: intended strict negative controls in both modes.
- [ ] AC #4: bounded native Darwin/Linux execution and owned-resource cleanup.
- [ ] AC #5: independently frozen RED/GREEN and truthful scoped evidence.


### 2026-09-06T03:47:58Z ramirosalas
## Implementation Evidence (RED DELIVERED)

PROOF:

### CI/Test Results

Commands run:
- From /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-yig6: `go test -json -count=1 -timeout=180s ./cmd/machinery -run '^TestCheckerFixtureCoverageRegression$'`.
- `go test -json -count=1 -timeout=60s ./cmd/machinery -run '^TestVerifyCheckersReproducible$'`.
- `go test -json -count=1 -timeout=60s -coverprofile=/tmp/machinery-yig6-red.09ui51/original-coverage.out ./cmd/machinery -run '^TestVerifyCheckersReproducible$'`.
- Native targeted selection of TestCheckerFixtureProtocolControls, original, and five existing stream/bounds/timeout leaves; covered equivalent additionally includes outer meta. Full exact selectors and command timings are in /tmp/machinery-yig6-red.09ui51/REPORT.md SHA256648c4d87d5facac1d46110ab796436a8a7a19ebeceadb15b8e8a53edbc4e6eb8, already appended as evidence.
- `go tool cover -func=/tmp/machinery-yig6-red.09ui51/original-coverage.out` and `go tool cover -func=/tmp/machinery-yig6-red.09ui51/selection-coverage.out`.
- `pvg verify cmd/machinery/checker_fixture_coverage_test.go --include-tests --format text`; `pvg story verify-tdd --base 70652b948bf090008b1965c85daf36ea374daea4 --json`.

Summary: RED behavior proven. Original native1 PASS/0 FAIL; original covered0 PASS/1 FAIL at real trailing-data assertion. Matched compiled native10 terminal leaves PASS/0 FAIL; covered5 PASS/5 FAIL. Zero skips/setup/deadline failures. Existing stream/bounds/timeout five leaves PASS in both modes. pvg verify PASS1file0issues; verify-tdd PASS1commit0violations. Native direct selection10 PASS/0 FAIL. Covered full selection terminal test events6 PASS/8 FAIL including two failed parent suites; direct covered leaves5 PASS/5 FAIL plus outer meta native PASS/covered FAIL.

Coverage: actual Go profile parsed successfully,2541 blocks/4015 statements/402 executed including366 verify_checkers.go statements,10.0%; original397 executed including361 production,9.9%. Real Go1.27.1 darwin/arm64 only. Outer meta6.46s; covered outer5.24s; original native package0.998s/covered0.800s. All compiler/process calls bounded and terminated. Linux native proof remains required at GREEN/final; no Docker/Windows claim.

### Commit
- Branch: story/MAC-yig6
- SHA: c59c89de31c7f1268a220caf0fa0d67a8e166d0f
- Frozen new test SHA25680b590315a79f4faa3c9741eb1347e6070d3b01046338c76ce6e7000d6e92677,246 additions/0 deletions. Existing helpers and production unchanged.

### pvg verify
- VERIFY: PASSED (1 files scanned, 0 issues).
- verify-tdd:1commit, violations:null.

### AC Verification
| AC # | Requirement | Test Location | RED status |
|------|-------------|---------------|------------|
| 1 | Original actual native/covered assertions | TestCheckerFixtureCoverageRegression and unchanged TestVerifyCheckersReproducible | Native pass; covered behavioral fail; GREEN pending |
| 2 | Actual separate streams and closed production env | TestCheckerFixtureProtocolControls | Native stderr0B; covered warning54B, valid stdout131B; contamination frozen |
| 3 | Correct/wrong digest/wrong platform/extra data distinct diagnoses | Four TestCheckerFixtureProtocolControls children | Native intended diagnoses reached; covered digest/platform UNREACHED, never credited |
| 4 | Bounded host execution and cleanup | Meta180s, children60s, existing bounded leaves | Darwin complete, Linux native pending; zero skips/live owned processes |
| 5 | Independent RED and unchanged GREEN proof | tdd-red commit and evidence report | Frozen RED ready for independent review; GREEN pending |

LEARNINGS:
- Coverage stderr corrupts strict protocol although stdout remains valid; a negative must prove its distinct diagnosis.
- Real parsed coverage must include executed production statements; helper-only instrumentation is insufficient.
- Future cached helper lifetime belongs to unchanged TestMain root, not first test temporary directory.

Helper amendment proposal (not applied): /tmp/machinery-yig6-red.09ui51/HELPER-AMENDMENT-PROPOSAL.md SHA256555a3ac33a1a1b5023d3f378e56719f4dee7707c3cb7b0657dab5471ddc53497. Independent PM review required before the three exact executable-selector changes and associated real uninstrumented helper-build/cache support. No frozen test edits authorized by this delivery.

Raw files under /tmp/machinery-yig6-red.09ui51: meta-native.jsonl SHA256836213a078c83cbd1142df1859b7e7f38e28f3e2b98bf2d2fd4246f8a511526b; original-native.jsonl a8b4bc7717e17cf293178b76422d64cc2995cd3a27b5239b6aac9ad280f7578a; original-covered.jsonl fb085049e5ed5bd7e22191e96917c0d371d1bfbd990e59e1814e6b85c89cd04d; selection-native.jsonl7ddc99ad818c53ac80e68a354459df4b8837b6ee4356d99cbc7c665f02c01bdf; selection-covered.jsonl5c3ed8845d3055db8a21bdab0b81303f246684a968c8d9699a6b89be7a57cae8. Profile hashes and all exact timings are in full report. Documentation syntax discovery errors and first verify-delivery shape failure were coordination-only, not compiler or behavioral evidence; this terminal comment repairs evidence layout only.

## nd_contract
status: delivered

### evidence
- RED only, SHA c59c89de31c7f1268a220caf0fa0d67a8e166d0f. Full commands/counts/coverage/raw hashes above. Native matching control passes; covered real protocol fails for observed54-byte Go warning. No test amendments, skips, deadline/setup failures, installed modifications, remotes, Docker or full preflight.
- Clean worktree and no live owned process. Independent RED and helper-amendment review pending. GREEN implementation and Linux native execution remain required.

### proof
- [x] AC #1 RED: unchanged original native PASS/covered behavioral FAIL on frozen SHA; GREEN pending.
- [x] AC #2 RED: real raw streams expose coverage contamination with unchanged production environment/parser.
- [x] AC #3 RED: native distinct diagnoses observed; covered wrong-digest/platform UNREACHED and not credited.
- [ ] AC #4 FINAL: Darwin bounded execution complete; native Linux execution pending.
- [ ] AC #5 FINAL: frozen RED proved; independent approval and GREEN evidence pending.

### 2026-09-06T03:48:09Z ramirosalas
## Implementation Evidence (DELIVERED)

PROOF:

### CI/Test Results

Commands run:
- From /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-yig6: `go test -json -count=1 -timeout=180s ./cmd/machinery -run '^TestCheckerFixtureCoverageRegression$'`.
- `go test -json -count=1 -timeout=60s ./cmd/machinery -run '^TestVerifyCheckersReproducible$'`.
- `go test -json -count=1 -timeout=60s -coverprofile=/tmp/machinery-yig6-red.09ui51/original-coverage.out ./cmd/machinery -run '^TestVerifyCheckersReproducible$'`.
- Native targeted selection of TestCheckerFixtureProtocolControls, original, and five existing stream/bounds/timeout leaves; covered equivalent additionally includes outer meta. Full exact selectors and command timings are in /tmp/machinery-yig6-red.09ui51/REPORT.md SHA256648c4d87d5facac1d46110ab796436a8a7a19ebeceadb15b8e8a53edbc4e6eb8, already appended as evidence.
- `go tool cover -func=/tmp/machinery-yig6-red.09ui51/original-coverage.out` and `go tool cover -func=/tmp/machinery-yig6-red.09ui51/selection-coverage.out`.
- `pvg verify cmd/machinery/checker_fixture_coverage_test.go --include-tests --format text`; `pvg story verify-tdd --base 70652b948bf090008b1965c85daf36ea374daea4 --json`.

Summary: RED behavior proven. Original native1 PASS/0 FAIL; original covered0 PASS/1 FAIL at real trailing-data assertion. Matched compiled native10 terminal leaves PASS/0 FAIL; covered5 PASS/5 FAIL. Zero skips/setup/deadline failures. Existing stream/bounds/timeout five leaves PASS in both modes. pvg verify PASS1file0issues; verify-tdd PASS1commit0violations. Native direct selection10 PASS/0 FAIL. Covered full selection terminal test events6 PASS/8 FAIL including two failed parent suites; direct covered leaves5 PASS/5 FAIL plus outer meta native PASS/covered FAIL.

Coverage: actual Go profile parsed successfully,2541 blocks/4015 statements/402 executed including366 verify_checkers.go statements,10.0%; original397 executed including361 production,9.9%. Real Go1.27.1 darwin/arm64 only. Outer meta6.46s; covered outer5.24s; original native package0.998s/covered0.800s. All compiler/process calls bounded and terminated. Linux native proof remains required at GREEN/final; no Docker/Windows claim.

### Commit
- Branch: story/MAC-yig6
- SHA: c59c89de31c7f1268a220caf0fa0d67a8e166d0f
- Frozen new test SHA25680b590315a79f4faa3c9741eb1347e6070d3b01046338c76ce6e7000d6e92677,246 additions/0 deletions. Existing helpers and production unchanged.

### pvg verify
- VERIFY: PASSED (1 files scanned, 0 issues).
- verify-tdd:1commit, violations:null.

### AC Verification
| AC # | Requirement | Test Location | RED status |
|------|-------------|---------------|------------|
| 1 | Original actual native/covered assertions | TestCheckerFixtureCoverageRegression and unchanged TestVerifyCheckersReproducible | Native pass; covered behavioral fail; GREEN pending |
| 2 | Actual separate streams and closed production env | TestCheckerFixtureProtocolControls | Native stderr0B; covered warning54B, valid stdout131B; contamination frozen |
| 3 | Correct/wrong digest/wrong platform/extra data distinct diagnoses | Four TestCheckerFixtureProtocolControls children | Native intended diagnoses reached; covered digest/platform UNREACHED, never credited |
| 4 | Bounded host execution and cleanup | Meta180s, children60s, existing bounded leaves | Darwin complete, Linux native pending; zero skips/live owned processes |
| 5 | Independent RED and unchanged GREEN proof | tdd-red commit and evidence report | Frozen RED ready for independent review; GREEN pending |

LEARNINGS:
- Coverage stderr corrupts strict protocol although stdout remains valid; a negative must prove its distinct diagnosis.
- Real parsed coverage must include executed production statements; helper-only instrumentation is insufficient.
- Future cached helper lifetime belongs to unchanged TestMain root, not first test temporary directory.

Helper amendment proposal (not applied): /tmp/machinery-yig6-red.09ui51/HELPER-AMENDMENT-PROPOSAL.md SHA256555a3ac33a1a1b5023d3f378e56719f4dee7707c3cb7b0657dab5471ddc53497. Independent PM review required before the three exact executable-selector changes and associated real uninstrumented helper-build/cache support. No frozen test edits authorized by this delivery.

Raw files under /tmp/machinery-yig6-red.09ui51: meta-native.jsonl SHA256836213a078c83cbd1142df1859b7e7f38e28f3e2b98bf2d2fd4246f8a511526b; original-native.jsonl a8b4bc7717e17cf293178b76422d64cc2995cd3a27b5239b6aac9ad280f7578a; original-covered.jsonl fb085049e5ed5bd7e22191e96917c0d371d1bfbd990e59e1814e6b85c89cd04d; selection-native.jsonl7ddc99ad818c53ac80e68a354459df4b8837b6ee4356d99cbc7c665f02c01bdf; selection-covered.jsonl5c3ed8845d3055db8a21bdab0b81303f246684a968c8d9699a6b89be7a57cae8. Profile hashes and all exact timings are in full report. Documentation syntax discovery errors and first verify-delivery shape failure were coordination-only, not compiler or behavioral evidence; this terminal comment repairs evidence layout only.

## nd_contract
status: delivered

### evidence
- RED only, SHA c59c89de31c7f1268a220caf0fa0d67a8e166d0f. Full commands/counts/coverage/raw hashes above. Native matching control passes; covered real protocol fails for observed54-byte Go warning. No test amendments, skips, deadline/setup failures, installed modifications, remotes, Docker or full preflight.
- Clean worktree and no live owned process. Independent RED and helper-amendment review pending. GREEN implementation and Linux native execution remain required.

### proof
- [x] AC #1 RED: unchanged original native PASS/covered behavioral FAIL on frozen SHA; GREEN pending.
- [x] AC #2 RED: real raw streams expose coverage contamination with unchanged production environment/parser.
- [x] AC #3 RED: native distinct diagnoses observed; covered wrong-digest/platform UNREACHED and not credited.
- [ ] AC #4 FINAL: Darwin bounded execution complete; native Linux execution pending.
- [ ] AC #5 FINAL: frozen RED proved; independent approval and GREEN evidence pending.

### 2026-09-06T03:48:17Z ramirosalas
## Implementation Evidence

PROOF:

### CI/Test Results

Commands run:
- From /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-yig6: `go test -json -count=1 -timeout=180s ./cmd/machinery -run '^TestCheckerFixtureCoverageRegression$'`.
- `go test -json -count=1 -timeout=60s ./cmd/machinery -run '^TestVerifyCheckersReproducible$'`.
- `go test -json -count=1 -timeout=60s -coverprofile=/tmp/machinery-yig6-red.09ui51/original-coverage.out ./cmd/machinery -run '^TestVerifyCheckersReproducible$'`.
- Native targeted selection of TestCheckerFixtureProtocolControls, original, and five existing stream/bounds/timeout leaves; covered equivalent additionally includes outer meta. Full exact selectors and command timings are in /tmp/machinery-yig6-red.09ui51/REPORT.md SHA256648c4d87d5facac1d46110ab796436a8a7a19ebeceadb15b8e8a53edbc4e6eb8, already appended as evidence.
- `go tool cover -func=/tmp/machinery-yig6-red.09ui51/original-coverage.out` and `go tool cover -func=/tmp/machinery-yig6-red.09ui51/selection-coverage.out`.
- `pvg verify cmd/machinery/checker_fixture_coverage_test.go --include-tests --format text`; `pvg story verify-tdd --base 70652b948bf090008b1965c85daf36ea374daea4 --json`.

Summary: RED behavior proven. Original native1 PASS/0 FAIL; original covered0 PASS/1 FAIL at real trailing-data assertion. Matched compiled native10 terminal leaves PASS/0 FAIL; covered5 PASS/5 FAIL. Zero skips/setup/deadline failures. Existing stream/bounds/timeout five leaves PASS in both modes. pvg verify PASS1file0issues; verify-tdd PASS1commit0violations. Native direct selection10 PASS/0 FAIL. Covered full selection terminal test events6 PASS/8 FAIL including two failed parent suites; direct covered leaves5 PASS/5 FAIL plus outer meta native PASS/covered FAIL.

Coverage: actual Go profile parsed successfully,2541 blocks/4015 statements/402 executed including366 verify_checkers.go statements,10.0%; original397 executed including361 production,9.9%. Real Go1.27.1 darwin/arm64 only. Outer meta6.46s; covered outer5.24s; original native package0.998s/covered0.800s. All compiler/process calls bounded and terminated. Linux native proof remains required at GREEN/final; no Docker/Windows claim.

### Commit
- Branch: story/MAC-yig6
- SHA: c59c89de31c7f1268a220caf0fa0d67a8e166d0f
- Frozen new test SHA25680b590315a79f4faa3c9741eb1347e6070d3b01046338c76ce6e7000d6e92677,246 additions/0 deletions. Existing helpers and production unchanged.

### pvg verify
- VERIFY: PASSED (1 files scanned, 0 issues).
- verify-tdd:1commit, violations:null.

### AC Verification
| AC # | Requirement | Test Location | RED status |
|------|-------------|---------------|------------|
| 1 | Original actual native/covered assertions | TestCheckerFixtureCoverageRegression and unchanged TestVerifyCheckersReproducible | Native pass; covered behavioral fail; GREEN pending |
| 2 | Actual separate streams and closed production env | TestCheckerFixtureProtocolControls | Native stderr0B; covered warning54B, valid stdout131B; contamination frozen |
| 3 | Correct/wrong digest/wrong platform/extra data distinct diagnoses | Four TestCheckerFixtureProtocolControls children | Native intended diagnoses reached; covered digest/platform UNREACHED, never credited |
| 4 | Bounded host execution and cleanup | Meta180s, children60s, existing bounded leaves | Darwin complete, Linux native pending; zero skips/live owned processes |
| 5 | Independent RED and unchanged GREEN proof | tdd-red commit and evidence report | Frozen RED ready for independent review; GREEN pending |

LEARNINGS:
- Coverage stderr corrupts strict protocol although stdout remains valid; a negative must prove its distinct diagnosis.
- Real parsed coverage must include executed production statements; helper-only instrumentation is insufficient.
- Future cached helper lifetime belongs to unchanged TestMain root, not first test temporary directory.

Helper amendment proposal (not applied): /tmp/machinery-yig6-red.09ui51/HELPER-AMENDMENT-PROPOSAL.md SHA256555a3ac33a1a1b5023d3f378e56719f4dee7707c3cb7b0657dab5471ddc53497. Independent PM review required before the three exact executable-selector changes and associated real uninstrumented helper-build/cache support. No frozen test edits authorized by this delivery.

Raw files under /tmp/machinery-yig6-red.09ui51: meta-native.jsonl SHA256836213a078c83cbd1142df1859b7e7f38e28f3e2b98bf2d2fd4246f8a511526b; original-native.jsonl a8b4bc7717e17cf293178b76422d64cc2995cd3a27b5239b6aac9ad280f7578a; original-covered.jsonl fb085049e5ed5bd7e22191e96917c0d371d1bfbd990e59e1814e6b85c89cd04d; selection-native.jsonl7ddc99ad818c53ac80e68a354459df4b8837b6ee4356d99cbc7c665f02c01bdf; selection-covered.jsonl5c3ed8845d3055db8a21bdab0b81303f246684a968c8d9699a6b89be7a57cae8. Profile hashes and all exact timings are in full report. Documentation syntax discovery errors and first verify-delivery shape failure were coordination-only, not compiler or behavioral evidence; this terminal comment repairs evidence layout only.

## nd_contract
status: delivered

### evidence
- RED only, SHA c59c89de31c7f1268a220caf0fa0d67a8e166d0f. Full commands/counts/coverage/raw hashes above. Native matching control passes; covered real protocol fails for observed54-byte Go warning. No test amendments, skips, deadline/setup failures, installed modifications, remotes, Docker or full preflight.
- Clean worktree and no live owned process. Independent RED and helper-amendment review pending. GREEN implementation and Linux native execution remain required.

### proof
- [x] AC #1 RED: unchanged original native PASS/covered behavioral FAIL on frozen SHA; GREEN pending.
- [x] AC #2 RED: real raw streams expose coverage contamination with unchanged production environment/parser.
- [x] AC #3 RED: native distinct diagnoses observed; covered wrong-digest/platform UNREACHED and not credited.
- [ ] AC #4 FINAL: Darwin bounded execution complete; native Linux execution pending.
- [ ] AC #5 FINAL: frozen RED proved; independent approval and GREEN evidence pending.
