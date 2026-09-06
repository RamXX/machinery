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
updated_at: 2026-09-06T17:01:43Z
content_hash: "sha256:2f955e63102257e72cc3d8869130261972d57796e1798cb735950683103a7b5f"
blocks: [MAC-ou97]
follows: [MAC-2u36, MAC-a89e, MAC-p8ce, MAC-uzxr]
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
## PM RED Decision
APPROVED RED ONLY on 2026-09-05 Pacific. pvg story approve-red MAC-yig6 succeeded and canonical pvg nd show verified Status open, labels hard-tdd/red-approved, no delivered/accepted label and no closure. RED approval is not GREEN acceptance and grants no existing-helper amendment permission.

Independent report /tmp/machinery-yig6-pm.P55qWO/REPORT.md SHA256 28da07e6ef55f01430d00297c9aa5e7fc1fd1992f15d69cab49f8e786d08f745 was appended in full; exact replay.sh SHA256 4acc08fffef9edc190fd8d625c9ec03e5cfad364fdbe236c58bd15a700a0e357 and raw streams/profiles retained beside it. Review used detached checkout at frozen c59c89de31c7f1268a220caf0fa0d67a8e166d0f, frozen file SHA256 80b590315a79f4faa3c9741eb1347e6070d3b01046338c76ce6e7000d6e92677, exactly 246 additions/0 deletions. No source/test edits.

HELPER AMENDMENT HELD: proposal SHA256 555a3ac33a1a1b5023d3f378e56719f4dee7707c3cb7b0657dab5471ddc53497 is sound in principle, but roughly 50–70 new support lines are prose, not a concrete textual patch. Fresh GREEN implementer must prepare an unapplied exact patch outside checkout; independent PM must inspect/cache failure and cancellation/source resolution/build environment/cleanup details and authorize its bytes/hash BEFORE existing-helper writes. No TEST-EDIT AUTHORIZED tag or blanket permission is granted. Scope remains three executable selectors and minimal associated support in verify_checkers_test.go. Frozen RED, original assertions/modes/arguments/markers/exits, TestMain, golden helpers, fixture dispatcher/OCI helper, production/parser/environment/workflows/coverage settings remain unchanged.

## nd_contract
status: new

### evidence
- Phase RED approved and returned open/red-approved via supported shared-vault transition; verified landed. Final story remains unfinished.
- Independent Go 1.27.1 darwin/arm64 same-SHA replay: original native 1 PASS (0.943s package)/covered 1 behavioral FAIL (0.717s); native compiled inventory 10 leaves PASS/covered 5 PASS 5 FAIL; direct native 10 PASS; covered selection 6 PASS/8 FAIL terminal test events including parent suites. Zero skips/setup/deadline failures. Five unchanged stream/bounds/timeout leaves pass both modes.
- Actual raw stdout 131B/153B deliberate extra, native stderr 0B, covered stderr exact 54B warning. Wrong-digest/platform covered diagnoses UNREACHED and not credited. Both profile parsers succeed; selection 2541 blocks/4015 statements/402 executed/366 verify_checkers.go, 10.0%; original 397/361, 9.9%; profiles byte-identical to author evidence.
- pvg verify 1 file/0 issues; verify-tdd 1 commit/no violations; clean detached checkout, no live owned process, observed owned meta roots removed. Only report/log/profile artifacts and clean review checkout retained. Installed binary SHA256 5205883aaa4276d7eb6edb25b6ad43ac39a04bcb9a8b5ee55498127b04950849 unchanged.
- AC4 native Linux GREEN runtime proof remains required. Darwin RED does not waive it. No full preflight, remote operation, installed changes, Docker service or lifecycle/Windows claim.

### proof
- [x] AC #1 RED: frozen original native complete success versus actual covered behavioral defect independently reproduced.
- [x] AC #2 RED: actual clean/native versus contaminated/covered protocol boundary and production closed environment verified.
- [x] AC #3 RED: native intended distinct diagnoses reached; frozen covered assertions require those diagnoses, not early-error negatives.
- [x] AC #5 RED: tdd-red marker, frozen bytes, actual compiler/subprocess controls independently reviewed; sufficient behavioral RED bar.
- [ ] AC #4 FINAL: same frozen GREEN native Darwin and native Linux execution, exact inventory and cleanup required.
- [ ] AC #5 FINAL: exact helper amendment authorization HELD; unchanged frozen GREEN tests and existing checker-focused controls must pass before final acceptance.

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
- 2026-09-06T07:16:02Z status: open -> in_progress
- 2026-09-06T07:16:02Z auto-follows: linked to predecessor MAC-p8ce
- 2026-09-06T07:16:02Z claimed by dev-MAC-yig6
- 2026-09-06T09:47:27Z status: in_progress -> open
- 2026-09-06T10:06:07Z status: open -> in_progress
- 2026-09-06T10:06:07Z auto-follows: linked to predecessor MAC-uzxr
- 2026-09-06T10:06:07Z claimed by dev-MAC-yig6
- 2026-09-06T17:01:43Z status: in_progress -> open
- 2026-09-06T17:01:43Z released by ramirosalas

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-ou97]]
- Follows: [[MAC-2u36]], [[MAC-a89e]], [[MAC-p8ce]], [[MAC-uzxr]]

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

### 2026-09-06T04:08:59Z ramirosalas
## PM RED Decision
APPROVED RED ONLY on 2026-09-05 Pacific. pvg story approve-red MAC-yig6 succeeded and canonical pvg nd show verified Status open, labels hard-tdd/red-approved, no delivered/accepted label and no closure. RED approval is not GREEN acceptance and grants no existing-helper amendment permission.

Independent report /tmp/machinery-yig6-pm.P55qWO/REPORT.md SHA256 28da07e6ef55f01430d00297c9aa5e7fc1fd1992f15d69cab49f8e786d08f745 was appended in full; exact replay.sh SHA256 4acc08fffef9edc190fd8d625c9ec03e5cfad364fdbe236c58bd15a700a0e357 and raw streams/profiles retained beside it. Review used detached checkout at frozen c59c89de31c7f1268a220caf0fa0d67a8e166d0f, frozen file SHA256 80b590315a79f4faa3c9741eb1347e6070d3b01046338c76ce6e7000d6e92677, exactly 246 additions/0 deletions. No source/test edits.

HELPER AMENDMENT HELD: proposal SHA256 555a3ac33a1a1b5023d3f378e56719f4dee7707c3cb7b0657dab5471ddc53497 is sound in principle, but roughly 50–70 new support lines are prose, not a concrete textual patch. Fresh GREEN implementer must prepare an unapplied exact patch outside checkout; independent PM must inspect/cache failure and cancellation/source resolution/build environment/cleanup details and authorize its bytes/hash BEFORE existing-helper writes. No TEST-EDIT AUTHORIZED tag or blanket permission is granted. Scope remains three executable selectors and minimal associated support in verify_checkers_test.go. Frozen RED, original assertions/modes/arguments/markers/exits, TestMain, golden helpers, fixture dispatcher/OCI helper, production/parser/environment/workflows/coverage settings remain unchanged.

## nd_contract
status: new

### evidence
- Phase RED approved and returned open/red-approved via supported shared-vault transition; verified landed. Final story remains unfinished.
- Independent Go 1.27.1 darwin/arm64 same-SHA replay: original native 1 PASS (0.943s package)/covered 1 behavioral FAIL (0.717s); native compiled inventory 10 leaves PASS/covered 5 PASS 5 FAIL; direct native 10 PASS; covered selection 6 PASS/8 FAIL terminal test events including parent suites. Zero skips/setup/deadline failures. Five unchanged stream/bounds/timeout leaves pass both modes.
- Actual raw stdout 131B/153B deliberate extra, native stderr 0B, covered stderr exact 54B warning. Wrong-digest/platform covered diagnoses UNREACHED and not credited. Both profile parsers succeed; selection 2541 blocks/4015 statements/402 executed/366 verify_checkers.go, 10.0%; original 397/361, 9.9%; profiles byte-identical to author evidence.
- pvg verify 1 file/0 issues; verify-tdd 1 commit/no violations; clean detached checkout, no live owned process, observed owned meta roots removed. Only report/log/profile artifacts and clean review checkout retained. Installed binary SHA256 5205883aaa4276d7eb6edb25b6ad43ac39a04bcb9a8b5ee55498127b04950849 unchanged.
- AC4 native Linux GREEN runtime proof remains required. Darwin RED does not waive it. No full preflight, remote operation, installed changes, Docker service or lifecycle/Windows claim.

### proof
- [x] AC #1 RED: frozen original native complete success versus actual covered behavioral defect independently reproduced.
- [x] AC #2 RED: actual clean/native versus contaminated/covered protocol boundary and production closed environment verified.
- [x] AC #3 RED: native intended distinct diagnoses reached; frozen covered assertions require those diagnoses, not early-error negatives.
- [x] AC #5 RED: tdd-red marker, frozen bytes, actual compiler/subprocess controls independently reviewed; sufficient behavioral RED bar.
- [ ] AC #4 FINAL: same frozen GREEN native Darwin and native Linux execution, exact inventory and cleanup required.
- [ ] AC #5 FINAL: exact helper amendment authorization HELD; unchanged frozen GREEN tests and existing checker-focused controls must pass before final acceptance.

### 2026-09-06T08:29:57Z ramirosalas
Dispatcher checkpoint: healthy claimed GREEN attempt remains PAUSED-AMENDMENT, not delivered or rejected. No helper edit authorized. Exact unapplied proposal /tmp/machinery-yig6-green-candidate.LPCYyO/MAC-yig6-helper-amendment.patch SHA256 e1a35cf42389c7e8de2e856686499a8a099b3e17aa7bc9b448c55fe723631f8c and REVIEW-INDEX.md SHA256 8d8c663656a090f14c2b2d99275555d2a41871dbe07a3e03a772e4fb4f0d87ab were read by root. Its coverpkg-empty argument re-enables helper instrumentation on the observed Go 1.27.1 host; proposal diagnostics are not GREEN proof. Independent PM must first review a revised compiler/environment boundary, then exact revised unapplied bytes before any existing-helper edit. Frozen c59c89de31c7f1268a220caf0fa0d67a8e166d0f and new-test SHA256 80b590315a79f4faa3c9741eb1347e6070d3b01046338c76ce6e7000d6e92677 remain authoritative. Production parser/env, TestMain, fixture dispatcher, assertions and coverage settings remain outside amendment permission. Native final verification is now user-authorized; actual Linux execution remains owed and is not replaced by Darwin or cross-compilation. Installed assets, preflight and remotes untouched.

## nd_contract
status: in_progress

### evidence
- External candidate proposal exposed compiler-flag incompatibility before existing-helper writes; independent revised-amendment review pending.
- Root inspected proposal/report and canonical story only; no new behavioral test execution claimed.

### proof
- [x] AC #5 RED: prior independent RED approval and exact frozen history preserved.
- [ ] AC #1-3 GREEN: revised exact helper authorization and native/covered same-SHA proof pending.
- [ ] AC #4 FINAL: actual bounded Darwin and Linux execution and cleanup pending.
- [ ] AC #5 FINAL: independent GREEN acceptance pending.

### 2026-09-06T08:47:53Z ramirosalas
Independent compiler-boundary method review completed, not a story rejection/acceptance transition: /tmp/machinery-yig6-pm-boundary.f5p6dZ/REPORT.md SHA256 70dbbe39244148680e842dfecdef981f6cafb87a8a860013c19c11aaf397171b, read fully and hash-verified by dispatcher. REVIEW_RESULT: REJECTED for BEFORE-WRITE readiness only. Corrected compiler direction is sound in principle: child-only nonempty GOFLAGS=-cover=false, no coverpkg reset, unchanged processcontrol.Run plus bounded WaitDelay, validated absolute source, cached errors/no fallback, preserved parent instrumentation. Real frozen-archive compiler diagnostics are not GREEN proof. Feasible persistent negatives are missing for deliberate ambient/persisted coverage flags, helper compiler cancellation/setup failure, cached failure/no retry and unavailable source. Healthy GREEN author resumed for external exact revised helper patch plus concrete supplemental-test proposal only; provisional third test file is NOT yet owned or authorized. Original two-path scope and all frozen bytes remain unchanged until independent exact review and supported Sr PM scope repair. No installed/preflight/remote/service changes.

## nd_contract
status: in_progress

### evidence
- Independent method review and root verification above; no behavioral delivery claimed and no helper writes authorized.
- Healthy existing claim preserved; external proposal preparation active.

### proof
- [x] AC #5 RED: original independent RED and frozen 246-line file preserved.
- [ ] AC #1-3 GREEN: exact revised helper/supplemental-test readiness and actual protocol/coverage proofs pending.
- [ ] AC #4 FINAL: actual same-SHA Darwin/Linux execution and cleanup pending.
- [ ] AC #5 FINAL: independent GREEN acceptance pending.

### 2026-09-06T09:29:02Z ramirosalas
CURRENT COMPLETE FOUR-PATH SCOPE / STAGED AMENDMENT 2026-09-06

This is a SEPARATE bounded scope repair after the sealed assurance-decomposition handoff (immutable nd/backlog f9d3612bfc3feac8df59ad65cb0d623268700a99). It does not alter that report/snapshot or invent a new architecture. Existing healthy in_progress/red-approved claim, original RED and all evidence/history remain intact. No delivery/rejection/acceptance, label, claim, ref or source transition is made.

Independent exact BEFORE-WRITE review: /tmp/machinery-yig6-pm-final.rirEOm/REPORT.md, 80 lines, SHA256 fa958a74440d41058b53b9d112cd1eebe40fbf7658a13f10e758b4fb988c8f3b, fully read and hash-verified by Sr PM. Its APPROVED decision is exact proposal/method readiness CONDITIONAL on the staged gates below, NOT supplemental RED approval, GREEN acceptance, Linux proof or blanket existing-test amendment authority. The previous proposed three-path/two-path budgets and unapproved 94/33/189-line drafts are historical only.

CURRENT EXCLUSIVE OWNERSHIP
PRODUCES:
- cmd/machinery/checker_fixture_coverage_test.go -> original 246-line frozen RED, UNCHANGED; SHA256 80b590315a79f4faa3c9741eb1347e6070d3b01046338c76ce6e7000d6e92677 at c59c89de31c7f1268a220caf0fa0d67a8e166d0f.
- cmd/machinery/verify_checkers_test.go -> ONLY exact independently reviewed helper patch, +91/-12, conditional on supplemental committed RED freeze and subsequent exact amendment authorization.
- cmd/machinery/checker_fixture_helper_red_test.go -> NEW baseline-callable 147-line supplemental RED; SHA256 ea67a87453e92d0f3e056ffa15367a0ad85cb6592cb6ce826bfd83ff84e93fcd.
- cmd/machinery/checker_fixture_helper_build_test.go -> NEW prospective-only 44-line tests; SHA256 6274d5685e6804912780761cafbbfb8f370d8f2863430f65f399720a960575ef; may accompany ONLY the later authorized exact helper amendment, never the baseline RED commit.

CONSUMES:
- Existing frozen checker fixture and production protocol.
  source: exact c59c89de31c7f1268a220caf0fa0d67a8e166d0f baseline and unchanged original 246-line test; actual writeRegistryFile, checkerFixtureEngineArgs and checkerProcessFixtureCommand selectors, existing TestMain/fixture dispatcher/protocol behavior.
- Exact reviewed external helper patch.
  source: /tmp/machinery-yig6-revised-proposal.4O8e2O/MAC-yig6-revised-helper.patch SHA256 f6c4ea4e5d0ea57300cecf4857c582e732ca72a2eb8ecc393b43856d8e1e8d13; immutable index /tmp/machinery-yig6-final-index.9Fh2wK.md SHA256 30ee81af8373caed3b6bee6908da026ceeb62d7adccd6b9931a023a7f42dc95b.
- Existing process ownership.
  spec: processcontrol.Run(ctx context.Context, cmd *exec.Cmd) error is used unchanged; joined context/wait/cleanup/reap errors must not be mistaken for expected native exit1.

COMPLETE CURRENT ACCEPTANCE / PHASE MAP
1. Preserve the original story AC1-AC5 in full: actual unchanged TestVerifyCheckersReproducible native/covered success, clean real protocol streams, distinct correct/wrong-digest/wrong-platform/extra-data sensitivity, valid production-executed coverage, bounded native Darwin/Linux cleanup and independent frozen RED/GREEN evidence. Current total ownership is the FOUR paths above; no other source/test edit is authorized.
2. NEXT SOURCE PHASE is ONLY the exact 147-line baseline-callable supplemental RED on unchanged c59c89de helper/source plus original246. Actual developer commit is required; independent PM must verify committed SHA/bytes and intended behavioral baseline failures, then freeze through the supported workflow BEFORE helper or44-line writes. External archive calibration is not that commit/approval. No undefined prospective seam, compile/setup/infrastructure failure or unchanged covered wrong-diagnosis masking is credited as behavioral RED.
3. Only AFTER that committed supplemental RED freeze may the exact reviewed helper patch and exact prospective44-line file receive the required explicit amendment authorization record and [test-edit-authorized] commit marker. This scope note does not itself grant that later write permission. Any byte change requires fresh independent exact review; no blanket test rewrite. Original246 and newly frozen147 stay byte-for-byte unchanged throughout GREEN.
4. The helper amendment changes only the three original os.Executable selector blocks plus exact reviewed support. Preserve original assertions/modes/argv/markers/exits, TestMain, fixture dispatcher/OCI helper, golden helpers, production parser/env, processcontrol, workflows and coverage settings. Normal noncovered os.Executable behavior stays. Covered helper uses real go test -c -cover=false -o <owned> ., child-only nonempty GOFLAGS=-cover=false, NO coverpkg reset; parent remains instrumented. Absolute regular source, t.Context-derived60s bound, sync.Once cached success/error, no retry/fallback/published path on failure, bounded process ownership/output and unchanged TestMain-owned root remain exact reviewed semantics.
5. Supplemental147 proves actual ambient and persisted coverage protocol controls, covered caller execution and persistent compiler failure/no retry after PATH repair. Prospective44 separately proves valid-control, already-canceled, relative-source, missing-source and nonregular-source preconditions. Real compiler/process/output only, no fake outcomes or arbitrary warning filtering. Error classification must reject nil, canceled/deadline, wrong native exit/signal, unknown/wait/cleanup/reap leaves and every mixed joined-error tree; accepted expected-exit evidence must not hide cleanup failure. Source verification of error-tree policy is not fabricated runtime proof.
6. Final GREEN requires actual SAME-revision native Darwin AND Linux execution of both frozen files plus prospective44, paired original native/covered test and existing checker stream/bounds/timeout controls, separate raw streams, real parsed profiles with executed production statements, exact selected inventory/zero skips and owned cleanup. Independent reviewed Darwin archive calibration (147 baseline2FAIL versus candidate3PASS; originalcovered1PASS and valid14.0% profile) is scoped before-write evidence only; no full-suite/Linux/Windows/Docker lifecycle claim. The intentionally uninstrumented helper supplies no claimed helper coverage.
7. Developer provides exact phase SHAs, before/after hashes, actual commands/counts/profile validity and AC table. Independent PM accepts only after all phase/native gates; no self-acceptance. No installed binary/plugin changes, NIL/Dagger/service mutations, remotes/push, or full scripts/preflight.sh during this repair. Any other path/byte/scope change returns for review before writes.

DIFF BUDGET (CURRENT, EXPLICITLY REPAIRED)
Four paths aggregate +528/-12 against the story base: original246 frozen + helper91/-12 + supplemental147 + prospective44. The current increment against c59c89de is three paths +282/-12. This replaces the original ~250-400-line/two-path forecast; expansion is the reviewed negative coverage and baseline/prospective phase separation, not permission for broader refactoring.

MANDATORY SKILLS: developer and pm_acceptor; codebase-memory for implementation source verification. Sr PM performed tracker scope repair only.
OUT OF SCOPE: production/runtime architecture, assertion weakening, original246 rewrite, helper edit before supplemental freeze, prospective44 in the RED commit, broad error-classifier/processcontrol refactor, installed assets and release operations.

## nd_contract
status: in_progress

### evidence
- Independent exact before-write report fully read and SHA256 verified; complete current four-path scope and staged gates recorded here.
- Original approved246 RED and healthy claim preserved; no source/ref/label/status/claim or runtime changes by Sr PM.
- This scope repair is not supplemental RED approval, test-edit authorization, GREEN acceptance or native Linux proof.

### proof
- [x] Original RED history and exact frozen246 identity preserved as existing approved evidence.
- [ ] AC1: original full behavior/protocol/coverage/cleanup requirements remain.
- [ ] AC2: exact147 committed on unchanged helper baseline and independently frozen.
- [ ] AC3-AC5: subsequently authorized exact helper+44 amendment, frozen147/246 and all reviewed negative controls.
- [ ] AC6-AC7: actual same-revision native Darwin/Linux GREEN, truthful evidence and independent final acceptance.


### 2026-09-06T09:43:36Z ramirosalas
## SUPPLEMENTAL-RED PROOF (2026-09-06)

Phase: supplemental RED only. This comment records the newly committed baseline-callable 147-line test. It is not GREEN, not supplemental PM freeze/approval, not delivery, and not acceptance/rejection. No helper implementation or prospective 44-line test was written.

### Commit and byte identity
- Worktree: /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-yig6 on story/MAC-yig6.
- Frozen base: c59c89de31c7f1268a220caf0fa0d67a8e166d0f.
- Supplemental RED commit: ac439c517de169addb68637c50040a9434bc8cca (`test(MAC-yig6): tdd-red supplemental helper coverage controls`).
- Only changed path from base: cmd/machinery/checker_fixture_helper_red_test.go, +147/-0.
- New file SHA256: ea67a87453e92d0f3e056ffa15367a0ad85cb6592cb6ce826bfd83ff84e93fcd; exact byte comparison with /tmp/machinery-yig6-revised-proposal.4O8e2O/checker_fixture_helper_red_test.go passed before commit.
- Frozen original RED SHA256: cmd/machinery/checker_fixture_coverage_test.go = 80b590315a79f4faa3c9741eb1347e6070d3b01046338c76ce6e7000d6e92677 (unchanged).
- Frozen helper source/test SHA256: cmd/machinery/verify_checkers_test.go = ce795eb21f6598df29d9cfa327e8b4019bcb0130e37332f2e646a374faf6c4fe (unchanged).
- Before-write review remains method-only: /tmp/machinery-yig6-pm-final.rirEOm/REPORT.md SHA256 fa958a74440d41058b53b9d112cd1eebe40fbf7658a13f10e758b4fb988c8f3b.

### Actual bounded execution
- Expected RED command: `env GOWORK=off GOPROXY=off GOTOOLCHAIN=local perl -e 'alarm 180; exec @ARGV' go test -count=1 -timeout=120s ./cmd/machinery -run '^TestCheckerFixtureHelper(AmbientCoverageProtocol|CachedCompilerFailure)$' -v`.
  - Actual terminal exit: 1 (expected behavioral RED). Raw stdout: /tmp/machinery-yig6-supplemental-red.N5Cinq/supplemental-red-rerun.stdout SHA256 24a1649c34ffb39c22cf25b7e2a718828d7460536038f3b19ce52ff962295978; stderr: /tmp/machinery-yig6-supplemental-red.N5Cinq/supplemental-red-rerun.stderr SHA256 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855; exit file: /tmp/machinery-yig6-supplemental-red.N5Cinq/supplemental-red-rerun.exit SHA256 4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865.
  - Exactly two top-level intended FAILs: TestCheckerFixtureHelperAmbientCoverageProtocol (both ambient/persisted subtests expose the 54-byte `warning: GOCOVERDIR not set, no coverage data emitted`) and TestCheckerFixtureHelperCachedCompilerFailure (frozen selector child exited 0; test rejects it). No timeout, panic, skip, or compiler/setup failure.
  - Covered wrong-digest/platform intended diagnoses remain UNREACHED behind the existing strict trailing-data error; this is baseline RED evidence and is not credited as negative proof.
- Unchanged native control: `env GOWORK=off GOPROXY=off GOTOOLCHAIN=local perl -e 'alarm 90; exec @ARGV' go test -json -count=1 -timeout=60s ./cmd/machinery -run '^TestVerifyCheckersReproducible$'`.
  - Actual terminal exit: 0; exact one selected test PASS (0.75s; package 1.055s). Raw JSONL: /tmp/machinery-yig6-supplemental-red.N5Cinq/original-native.jsonl SHA256 10cdad4c9eef2714fe24c37a899b5e43c226116aadaba1b6a712ed56ee849efa; stderr empty SHA256 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855; exit file SHA256 9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa.

### Static/TDD checks
- `gofmt -d cmd/machinery/checker_fixture_helper_red_test.go`: no output.
- `pvg verify cmd/machinery/checker_fixture_helper_red_test.go --include-tests --format text`: `VERIFY: PASSED (1 files scanned, 0 issues)`.
- `pvg story verify-tdd --range c59c89de31c7f1268a220caf0fa0d67a8e166d0f..ac439c517de169addb68637c50040a9434bc8cca --json`: commits_checked=1, red_marker=tdd-red, violations=null.
- Initial `pvg verify ... --format=text` was rejected by this installed CLI; `pvg verify --help` confirmed the supported spelling is `--format text`, which passed above. No source/test change resulted.

### AC map (this phase only)
- [x] Supplemental RED gate: exact 147 bytes committed on frozen helper baseline; expected two behavioral baseline failures reproduced with raw streams/exit.
- [x] Original native unchanged control: selected TestVerifyCheckersReproducible passes.
- [ ] AC1-AC5 GREEN/final: helper patch, prospective44, covered-success/profile proof, Darwin/Linux final execution, and independent acceptance remain pending and were not attempted.

LEARNINGS:
- The baseline RED isolates both ambient and persisted Go coverage configuration contamination without an implementation seam.
- `processcontrol.Run`-derived child failure classification is deliberately strict: a frozen child exit 0 is rejected, rather than treated as a substitute for the expected compiler failure.
- This phase preserves the immutable 246-line original RED and the existing helper byte-for-byte; no Docker, installed binary, remote, service, preflight, or coverage-policy action occurred.

## nd_contract
status: in_progress

### evidence
- Supplemental RED commit ac439c517de169addb68637c50040a9434bc8cca contains only the exact approved 147-line test; protected frozen hashes verified.
- Required native bounded RED exit=1 and unchanged control exit=0 are recorded above with raw paths/hashes.
- TDD verifier and supported static verification pass; no GREEN or PM transition performed.

### proof
- [x] Supplemental RED: baseline behavioral defect and missing-helper behavior are expressed in committed exact tests.
- [x] Frozen original test/helper identity preserved.
- [ ] GREEN/final requirements: explicitly pending independent supplemental freeze and later authorized phase.

### 2026-09-06T09:54:54Z ramirosalas
## PM SUPPLEMENTAL COMMITTED RED APPROVAL — 2026-09-06

APPROVED SUPPLEMENTAL RED ONLY: ac439c517de169addb68637c50040a9434bc8cca, parent c59c89de31c7f1268a220caf0fa0d67a8e166d0f. This is the independent reviewer's genuine exact committed freeze recorded through supported pvg nd comments add. It is not GREEN delivery/acceptance and does not claim the failed pvg story approve-red command succeeded.

Frozen supplemental cmd/machinery/checker_fixture_helper_red_test.go is exactly 147 additions/0 deletions, SHA256 ea67a87453e92d0f3e056ffa15367a0ad85cb6592cb6ce826bfd83ff84e93fcd. Original cmd/machinery/checker_fixture_coverage_test.go remains immutable 246 lines, SHA256 80b590315a79f4faa3c9741eb1347e6070d3b01046338c76ce6e7000d6e92677. Existing helper cmd/machinery/verify_checkers_test.go remains SHA256 ce795eb21f6598df29d9cfa327e8b4019bcb0130e37332f2e646a374faf6c4fe. Prospective44 is absent from this RED commit.

Independent exact-commit archive review/replay: /tmp/machinery-yig6-pm-supplemental.gOsP36/REPORT.md SHA256 855fc6566c7ced40ee3c5133bb3b3fd373a92e6f4ec555b475464a3d26013b4c; replay.sh SHA256 3fc4f6611715cd86db16ed10d8fb6b37b7f919f95a9540e9a5c4727429aaecc2.
- Native Go1.27.1 darwin/arm64 bounded supplemental selection: exact2 intended top-level FAIL, package4.329s, actual ambient/persisted54-byte protocol warning and frozen cached-selector child exit0 correctly rejected; no compiler/setup/deadline/panic/skip outcome credited.
- Exact same-commit unchanged original native TestVerifyCheckersReproducible: 1 PASS, test0.81s/package1.104s. Both stderr artifacts empty.
- Full story audit 70652b948bf090008b1965c85daf36ea374daea4..ac439c517de169addb68637c50040a9434bc8cca:2 tdd-red commits, zero violations, only2 newly added RED files(+393/-0). Helper/production/processcontrol/TestMain/goldens unchanged. Static verify1file/0issues; gofmt empty.
- Raw supplemental stdout SHA256 a0c973328da9a6495826e7c9e0a639ca0ecddd7498483c973e9225449fa7ebfe; original-native.jsonl SHA256 fa46ac3ac581026330b150abbad3427e3f5cd63c1e8eb950f52f32d08effde9d, both beside the report. No new behavioral rerun is claimed by this completion note.

Partial-transition provenance, preserved: pvg story approve-red MAC-yig6 exited1 because label red-approved already existed. It first changed metadata Status to open; Assignee dev-MAC-yig6 and hard-tdd/red-approved labels were retained. It did not append its approval contract or run its remaining nd doctor --fix. Installed pvg1.64.0 build revision c0957106a81346033d7b1d82fde5f434a9db6bab was verified against clean matching source. No label removal, retry, acceptance, claim, waiver, private-code fix or tool replacement occurred.

The dispatcher explicitly authorized completion of the remaining NOTE plus DOCTOR operations separately, with this failure visible. This comment records the NOTE step using valid global contract status:new for existing open metadata. The DOCTOR step remains pending fresh zero-finding precheck, dispatcher clearance and before/after comparison of all56issue metadata and Body hashes. No installed nd source parity is claimed: actual documented help and actual doctor outputs govern verification.

Helper amendment authorization remains HELD until that supported completion is verified. Original storyAC1-5 and repaired four-path scope remain intact: actual same-revision Darwin AND Linux GREEN, clean raw protocol, distinct negative diagnoses, real production-executed profiles, exactinventory/zero skips/owned cleanup and final independent acceptance remain required. No installed Machinery/NIL/Dagger/service, remote, preflight, source or ref changes by this PM.

## nd_contract
status: new

### evidence
- Exact committed supplemental RED independently approved/frozen above, preserving original frozen246 and all history.
- Full2commit TDD audit and actual semanticRED/nativepositivecontrol complete.
- Failed approve-red partial transition is documented, not represented as success; supported remaining doctor completion pending.

### proof
- [x] Current scope phase2: exact147 committed and independently reviewed/frozen on unchanged helper/original246 baseline.
- [x] Original AC5 RED: both frozen identities and genuine baseline behavioral evidence preserved.
- [ ] Remaining workflow completion: actual zero-finding doctor precheck/fix and allissue invariance.
- [ ] Current phase3: narrow exacthelper+44 authorization remains held until completion.
- [ ] Original AC1-5 GREEN/current phase6-7: same-revision nativeDarwin/Linux, real profile/protocol/control/cleanup proof and independent final acceptance pending.


### 2026-09-06T09:57:10Z ramirosalas
## PM SUPPLEMENTAL FREEZE COMPLETION AND EXACT HELPER AMENDMENT AUTHORIZATION — 2026-09-06

The genuine independent supplemental RED approval at ac439c517de169addb68637c50040a9434bc8cca is now recorded at true EOF and the remaining supported doctor operation has completed. The failed pvg story approve-red invocation remains a FAILED partial transition; this note does not relabel it as successful. Its existing-label failure, prior status change to open, retained assignee/labels and unperformed note/doctor operations remain visible in the immediately preceding PM approval and independent report.

Supported completion evidence:
- After dispatcher clearance and the other writer's completion, captured all 56 issues using pvg nd list --all --limit 0 --json. Each snapshot retains every metadata field and replaces Body with a SHA256 hash; IDs are unique and the count is enforced.
- Fresh pvg nd doctor --json: exit 0, All 56 issues passed validation, empty stderr.
- Actual pvg nd doctor --fix: exit 0, All 56 issues passed validation, empty stderr.
- Before/after snapshots are byte-identical: SHA256 753671d79aa8e7f92290515a3a659bfbbd46b9b94d403cff332b5c081014b0db. No body, status, label, claim/assignee, dependency, timestamp or other captured issue metadata changed by doctor.
- Snapshot/log/script artifacts are under /tmp/machinery-yig6-pm-supplemental.gOsP36: issues-before-doctor.json, issues-after-doctor.json, issue-snapshot.cjs, doctor-precheck.stdout/stderr and doctor-fix.stdout/stderr.
- This explicitly completes only the supported NOTE plus DOCTOR operations remaining from the attempted workflow. No duplicate-label removal, retry, acceptance, claim, private-code fix, tool replacement or audit waiver occurred. Actual documented doctor behavior and observed outputs are the evidence; no installed nd source parity is asserted.

TEST-EDIT AUTHORIZED: cmd/machinery/verify_checkers_test.go -- ONLY the exact independently reviewed helper patch /tmp/machinery-yig6-revised-proposal.4O8e2O/MAC-yig6-revised-helper.patch SHA256 f6c4ea4e5d0ea57300cecf4857c582e732ca72a2eb8ecc393b43856d8e1e8d13 (+91/-12), against the unchanged helper SHA256 ce795eb21f6598df29d9cfa327e8b4019bcb0130e37332f2e646a374faf6c4fe at committed supplemental RED ac439c517de169addb68637c50040a9434bc8cca. Required commit subject marker: [test-edit-authorized].

The associated NEW prospective-only cmd/machinery/checker_fixture_helper_build_test.go may now be added ONLY with exact SHA256 6274d5685e6804912780761cafbbfb8f370d8f2863430f65f399720a960575ef (44 lines) alongside that authorized helper amendment. It is not RED and was not present in the frozen RED commit. This is precise amendment authority after actual committed supplemental freeze and supported remaining-work completion, not permission to rewrite existing assertions or bypass a gate.

Immutable RED identities throughout GREEN:
- Original 246-line cmd/machinery/checker_fixture_coverage_test.go: SHA256 80b590315a79f4faa3c9741eb1347e6070d3b01046338c76ce6e7000d6e92677.
- Supplemental 147-line cmd/machinery/checker_fixture_helper_red_test.go: SHA256 ea67a87453e92d0f3e056ffa15367a0ad85cb6592cb6ce826bfd83ff84e93fcd at ac439c517de169addb68637c50040a9434bc8cca.
- Full story TDD range 70652b948bf090008b1965c85daf36ea374daea4..ac439c517de169addb68637c50040a9434bc8cca: both tdd-red commits, zero violations, only the two added RED files. No production/helper delta before this authorization.

Exact boundary remains three original os.Executable selectors plus reviewed source-local, 60-second t.Context, sync.Once cache, child-only nonempty GOFLAGS=-cover=false, real compiler/processcontrol.Run, bounded WaitDelay and TestMain-owned support. Preserve every original assertion, mode, argument, marker and exit; fixture dispatcher, OCI helper, TestMain, shared golden helpers, production parser/environment, processcontrol, workflows and parent coverage settings stay unchanged. Any byte or scope change requires fresh independent review. Repaired aggregate scope remains four paths, +528/-12.

Independent committed RED report: /tmp/machinery-yig6-pm-supplemental.gOsP36/REPORT.md SHA256 855fc6566c7ced40ee3c5133bb3b3fd373a92e6f4ec555b475464a3d26013b4c, supplemented by these explicit remaining-work records. Exact before-write review: /tmp/machinery-yig6-pm-final.rirEOm/REPORT.md SHA256 fa958a74440d41058b53b9d112cd1eebe40fbf7658a13f10e758b4fb988c8f3b. Actual RED: two intended top-level failures, native original one pass; full commands/raw hashes/AC mapping remain in those records. No fresh behavioral rerun is claimed here.

Actual SAME-revision native Darwin AND Linux GREEN remains mandatory: both frozen files plus prospective 44-line tests, paired original native/covered success, existing checker stream/bounds/timeout controls, separate raw streams, real parsed production-executed profiles, exact inventory with zero skips, owned cleanup and independent final PM acceptance. No Linux or final GREEN assurance is supplied by this authorization. This PM performed no helper/source/ref write, automatic GREEN claim, installed Machinery/NIL/Dagger/service action, remote operation or preflight.

## nd_contract
status: new

### evidence
- Exact committed supplemental RED independently approved/frozen; full TDD audit and semantic RED/native control recorded.
- Failed approve-red provenance preserved; supported note plus actual doctor completion verified with all 56 issue records invariant.
- Narrow exact helper patch and new prospective44 authorization recorded above with required [test-edit-authorized] commit marker.

### proof
- [x] Current scope phase2: original246 and supplemental147 are exact immutable committed RED.
- [x] Remaining workflow completion: fresh zero-finding doctor precheck, actual doctor --fix and all-issue invariance verified.
- [x] Current phase3 authorization: exact helper patch plus exact prospective44 may proceed within the recorded boundary.
- [ ] Current phase3-5 implementation: authorized bytes must be applied/committed and both frozen files preserved.
- [ ] Original AC1-5 GREEN/current phase6-7: actual same-revision native Darwin/Linux protocol, profiles, controls, cleanup and final independent acceptance remain pending.


### 2026-09-06T10:11:46Z ramirosalas
## GREEN DARWIN EVIDENCE AND NATIVE-LINUX HOLD — 2026-09-06

Status: GREEN implementation committed and native Darwin scoped verification complete; this is NOT delivery, acceptance, closure, waiver, or a claim of Linux/Docker/Windows execution. The healthy claim remains retained.

### Immutable candidate and byte inventory
- Branch: story/MAC-yig6; candidate: ef505052dc73cb9032dd91f1a3f99837f76204a8 (`test(MAC-yig6): isolate covered checker fixture [test-edit-authorized]`).
- Authorized existing-test delta matches /tmp/machinery-yig6-revised-proposal.4O8e2O/MAC-yig6-revised-helper.patch exactly: +91/-12. Prospective cmd/machinery/checker_fixture_helper_build_test.go is exactly 44 lines, SHA256 6274d5685e6804912780761cafbbfb8f370d8f2863430f65f399720a960575ef.
- Immutable RED hashes remain: checker_fixture_coverage_test.go = 80b590315a79f4faa3c9741eb1347e6070d3b01046338c76ce6e7000d6e92677; checker_fixture_helper_red_test.go = ea67a87453e92d0f3e056ffa15367a0ad85cb6592cb6ce826bfd83ff84e93fcd. Aggregate base 70652b948bf090008b1965c85daf36ea374daea4..candidate is exactly +528/-12 across the authorized four paths (246, 147, 44, 91/-12).

### Actual bounded native Darwin execution
Host/toolchain: go1.27.1 darwin/arm64; `Darwin 25.6.0 arm64`. All commands ran synchronously from this story worktree with `GOWORK=off GOPROXY=off GOTOOLCHAIN=local` and Perl outer alarms. Raw immutable artifacts: /tmp/machinery-yig6-green.hZtIWT.

- Selected native: `perl -e 'alarm 240; exec @ARGV' go test -json -count=1 -timeout=180s ./cmd/machinery -run '^(TestVerifyCheckersReproducible|TestCheckerFixtureCoverageRegression|TestCheckerFixtureProtocolControls|TestCheckerFixtureHelperAmbientCoverageProtocol|TestCheckerFixtureHelperCachedCompilerFailure|TestCheckerFixtureHelperBuildProspectivePreconditions|TestRunCheckerBoundsOutput|TestRunCheckerReportsStreamsInDeterministicOrder|TestRunCheckerBoundsDescendantPipeWait|TestRunCheckerTimeoutDiagnostic|TestVerifyLocalOCIImageBoundsUnresponsiveEngine)$'`: exit 0, 25 PASS terminal test events, 0 FAIL, 0 SKIP, stderr 0 bytes. Raw JSONL SHA256 839d01aea57fff4ba1fdf7d2a7c1846b06992b6eacefdb46eb4147bf1515a63c.\n- Same selected covered command with `-coverprofile=/tmp/machinery-yig6-green.hZtIWT/selected-covered.out`: exit 0, 25 PASS, 0 FAIL, 0 SKIP, stderr 0 bytes. JSONL SHA256 16ffd3e0c78c26d753852a9246fda32d011dae89123e8cb6b984651e7e74e788; profile SHA256 f2f9d558057cc8eabbc2a72f06f0df24d355e90952925d758cd61755b2cdf4a0. `go tool cover -func` passes; 2541 blocks/4015 statements/574 executed/555 executed verify_checkers.go statements; total 14.3%.\n- Dedicated original native: `perl -e 'alarm 90; exec @ARGV' go test -json -count=1 -timeout=60s ./cmd/machinery -run '^TestVerifyCheckersReproducible$'`: exit 0, 2 PASS terminal events, 0 FAIL/SKIP, stderr 0 bytes; JSONL SHA256 7d5afad91e319951c770bfd7b42d6a19eac8a6a6be1739f0e73ca1ca59295294.\n- Dedicated original covered equivalent with `-coverprofile=/tmp/machinery-yig6-green.hZtIWT/original-covered.out`: exit 0, 2 PASS terminal events, 0 FAIL/SKIP, stderr 0 bytes; JSONL SHA256 e1f6e83ba0dba069c2acbba87ec85e1fa35d7c22a915a8019c49b52e8e4f832d; profile SHA256 08863e5ddb37b8c5f0e440ee629fa516a5dfe19f2ebe769ea3adb9598e4bd7fc. `go tool cover -func` passes; 2541 blocks/4015 statements/564 executed/545 executed verify_checkers.go statements; total 14.0%.\n- Actual raw covered protocol observations are clean: correct/wrong-digest/wrong-platform stdout 131 bytes and stderr 0; deliberate extra stdout 153 bytes and stderr 0. Correct accepted; distinct diagnoses reached for wrong digest (`do not contain exact reference`), wrong platform (`does not match required platform`), and extra data (`OCI RepoDigests response has trailing data`) in both native and covered compiled paths. No owned checker-fixture/control-root process remained after runs.\n\n### Static/TDD\n- `pvg story verify-tdd --range 70652b948bf090008b1965c85daf36ea374daea4..ef505052dc73cb9032dd91f1a3f99837f76204a8 --json`: 3 commits checked, null violations, authorized marker recognized.\n- Required `pvg verify cmd/machinery/verify_checkers_test.go cmd/machinery/checker_fixture_coverage_test.go cmd/machinery/checker_fixture_helper_red_test.go cmd/machinery/checker_fixture_helper_build_test.go --include-tests --format text` was run and FAILED only on its pre-existing heuristic: `verify_checkers_test.go:1595 [stub] return empty string`. Exact source is the legitimate no-match return of checkerProcessFixtureArgument; no byte is changed because it is outside the exact approved patch. This result is retained, not concealed. `git diff --check` passes and worktree is clean.\n\n### Remaining mandatory healthy hold: native Linux amd64\nNo native Linux amd64 executor is available. Docker is linux/aarch64 and was not used; cross-compilation/emulation is not native proof. Do not deliver/accept/close/waive. On a healthy native Linux amd64 host at this exact candidate SHA, retain raw stdout/stderr/exit files plus both profiles and `go tool cover -func` output, run the same selected native and covered commands above, then the dedicated original native and covered commands above; record `go version`, `uname -srm`, selected inventory, zero skips, raw protocol stream sizes/diagnoses, profile blocks/statements/executed production statements, and owned cleanup/process scan.\n\n## nd_contract\nstatus: in_progress\n\n### evidence\n- Candidate ef505052dc73cb9032dd91f1a3f99837f76204a8 committed with the required [test-edit-authorized] marker and exact approved file identities.\n- Same-candidate native Darwin selected and original paired coverage evidence is retained above with raw hashes and parsed production coverage.\n- Native Linux amd64 runtime remains unavailable and unwaived; no delivery transition was invoked.\n\n### proof\n- [x] AC #1 on native Darwin: original TestVerifyCheckersReproducible passes native and covered with nonempty parsed profiles.\n- [x] AC #2 on native Darwin: real covered parent/child fixture path has clean separately observed streams, without production policy change.\n- [x] AC #3 on native Darwin: correct, wrong digest, wrong platform, and deliberate trailing data controls reach their distinct outcomes in native and covered paths.\n- [ ] AC #4 FINAL: same candidate native Linux amd64 command matrix, raw artifacts, profile proof, zero skips, and cleanup remain required.\n- [ ] AC #5 FINAL: final independent PM review/acceptance remains required; static pvg verify heuristic result above must be evaluated without altering unapproved bytes.

### 2026-09-06T10:12:13Z ramirosalas
## nd_contract
status: in_progress

### evidence
- Candidate ef505052dc73cb9032dd91f1a3f99837f76204a8 is committed; native Darwin raw evidence is retained at /tmp/machinery-yig6-green.hZtIWT.
- Exact frozen and prospective file hashes, paired command outputs, profiles, cleanup scan, and static/TDD outcomes are in the immediately preceding GREEN DARWIN EVIDENCE comment.
- Native Linux amd64 is unavailable and unwaived; no delivery, acceptance, or closure was performed.

### proof
- [x] AC #1 Darwin: original native and covered commands pass with parsed nonempty production-executed profiles.
- [x] AC #2 Darwin: covered parent and compiled fixture streams are clean under real command execution.
- [x] AC #3 Darwin: correct, wrong digest, wrong platform, and trailing-data controls pass in both modes.
- [ ] AC #4 FINAL HOLD: run the exact same candidate command matrix on native Linux amd64; retain raw stdout/stderr/exits, both profiles plus go tool cover output, host/toolchain, inventory/zero-skip counts, raw protocol diagnostics, and owned cleanup scan. Docker linux/aarch64, cross-compilation, and emulation do not satisfy this.
- [ ] AC #5 FINAL HOLD: independent PM review/acceptance after Linux proof; the preceding required pvg verify result remains a documented unchanged-source heuristic finding.

### 2026-09-06T10:15:26Z ramirosalas
## GREEN EVIDENCE COUNT CORRECTION

- Correction to the preceding Darwin evidence: selected-native and selected-covered each have 24 test/subtest PASS events plus 1 package PASS event (25 total pass events), with 0 FAIL and 0 SKIP. They do not have 25 test events.
- Dedicated original-native and original-covered each have 1 test PASS plus 1 package PASS event (2 total pass events), with 0 FAIL and 0 SKIP. The covered counts were independently parsed from raw JSONL.
- Immutable external report: /tmp/machinery-yig6-green.hZtIWT/REPORT.md SHA256 a2b4c9be1286a5bcd08631ba8cfc4690e075477a9de1df7d5fe6f5dfa57bba09. It contains exact inventory, commands, raw artifact hashes, parsed coverage, static/TDD/cleanup provenance, AC mapping, and full native Linux amd64 matrix.
- Earlier original raw test artifacts did not include static/TDD/cleanup outputs; current bounded rechecks are retained in the same directory and are explicitly not presented as retroactive raw test evidence.

## nd_contract
status: in_progress

### evidence
- Candidate ef505052dc73cb9032dd91f1a3f99837f76204a8 remains unchanged, committed, and clean.
- Darwin raw event accounting is corrected above; original static heuristic finding and TDD result remain visible.
- Native Linux amd64 has not been executed and remains an unwaived healthy hold.

### proof
- [x] AC #1 Darwin: original native/covered control has one test pass plus package pass in each mode and parsed profiles.
- [x] AC #2 Darwin: selected covered run has clean real fixture streams; 24 test/subtest passes plus package pass.
- [x] AC #3 Darwin: selected native/covered inventory includes strict protocol and stream/bounds/timeout controls with zero skips.
- [ ] AC #4 FINAL HOLD: execute full report matrix on native Linux amd64 with required environment, source identity, profiles, raw logs, and cleanup proof.
- [ ] AC #5 FINAL HOLD: independent PM review/acceptance after Linux proof; no delivery/acceptance/closure was invoked.

### 2026-09-06T10:15:38Z ramirosalas
## REPORT HASH CORRECTION

- The actual SHA256 of /tmp/machinery-yig6-green.hZtIWT/REPORT.md is 8ecba89eb785887bfe8c4d5f145740bac01f8c04145fef6bd5fc9d80ce15e8c6. The preceding a2b4... value was erroneous and is superseded by this line.

## nd_contract
status: in_progress

### evidence
- Candidate ef505052dc73cb9032dd91f1a3f99837f76204a8 is unchanged and clean.
- Corrected event accounting and immutable report hash are recorded; no source, ref, test, or runtime rerun changed.
- Native Linux amd64 remains unavailable and unwaived.

### proof
- [x] AC #1 Darwin: original native/covered evidence is retained with correct event classification.
- [x] AC #2 Darwin: selected covered real fixture execution passes with 24 test/subtest PASS plus one package PASS.
- [x] AC #3 Darwin: strict protocol plus stream/bounds/timeout controls executed with zero skips.
- [ ] AC #4 FINAL HOLD: execute the report matrix on a native Linux amd64 host.
- [ ] AC #5 FINAL HOLD: independent PM review/acceptance after Linux evidence.
