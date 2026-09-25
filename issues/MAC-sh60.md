---
id: MAC-sh60
title: "Require executable oracle coverage evidence"
status: closed
priority: 0
type: bug
labels: [hard-tdd, red-approved, superseded]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:26Z
created_by: ramirosalas
updated_at: 2026-09-25T01:31:05Z
content_hash: "sha256:ae75febe5b55d51eaf8871994b4874dca54046b6b0431b9ddcab2464d7bcc5c0"
follows: [MAC-a89e, MAC-p8ce, MAC-olrx]
assignee: dev-MAC-sh60
led_to: [MAC-wi5z]
closed_at: 2026-09-06T19:21:16Z
close_reason: "Superseded by MAC-wi5z (compliant successor): mechanical hard-TDD audit failed — later commits modified existing test files without required authorization markers; behavioral passes did not cure the audit failure. History/branch preserved; not accepted. Successor owns internal/gates/oraclecov.go + tests after accepted MAC-hgz1."
---

## Description
## USER INTENT
Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.

## Context (Embedded)
Assessment F2 reproduced full Gt coverage from // TODO parse "Thing.oracle.md" using "|"; unused filename/delimiter constants; a Go test disabled with //go:build ignore; and Elixir # comment oracle IDs. fileNameCited uses raw corpus text, hash comment stripping excludes .exs. Coverage discovery is not execution proof.

## Ownership
Own only these paths and directly associated tests: internal/gates/oraclecov.go, internal/gates/oraclecov_negative_test.go. You are not alone in this codebase; preserve other edits and coordinate any shared-file changes with dispatcher.

## Boundary Map
PRODUCES:
- internal/gates/oraclecov.go -> hardened behavior and regression proof
- internal/gates/oraclecov_negative_test.go -> hardened behavior and regression proof
CONSUMES:
- Existing Machinery source interfaces.
  spec: gates.CheckOracleCoverage(design, impl string) *Gate

### Story Acceptance Criteria
1. Comments, docstrings-only citations, unused filename/delimiter declarations, disabled Go build-tag files and commented Elixir IDs cannot establish oracle-row or wholesale table coverage.
2. Positive real literal-ID tests and genuine conformance table parsers remain discoverable across currently supported languages. Unsupported/ambiguous parser evidence is explicitly uncovered rather than silently credited; no filename-keyword heuristic can confer wholesale coverage.
3. Add full CheckOracleCoverage regression cases for every assessment bypass, zero-test directories, mixed legitimate/disabled tests, malformed oracle references and actual positive parser/literal fixtures.
4. Gate output accurately labels discovery versus actual test execution; this story never claims static references prove assertions ran. Later native execution protocol supplies execution evidence.
5. No loss of stable-ID boundary checks, orphan/missing-oracle diagnostics or clause coverage. Integration exercises actual CLI against temporary real file trees; no injected fake coverage result.

## Testing Requirements
- Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
- Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
- Commands: go test ./internal/gates -run 'Oracle|Conformance|Coverage'; targeted command tests exercising machinery check with gt. RED bypass fixtures must currently produce the wrong accepted result and fail test assertions, while positive control passes.
- No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.

## OUT OF SCOPE
- Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
- Global heavy preflight, main merge and local release binary belong to final epic gate.

## DIFF BUDGET
- ~3-5 files, under 700 changed LOC; material overrun requires PM investigation, not weakened requirements.

## MANDATORY SKILLS
- developer for implementation; codebase-memory for discovery; pm_acceptor for independent acceptance.

## Delivery Requirements
Use pvg story deliver, never close. Append RED/GREEN SHAs, exact commands and outputs, test inventory, independent proof per AC, and any residual limits to shared nd. Do not use pushing pvg story merge.

## nd_contract
status: new

### evidence
- Created 2026-09-05; source signatures verified at assessment base 497419ab4512fcff765cd5feb27aed4c67b5608d.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified

## Acceptance Criteria


## Design


## Notes
AUTHORITATIVE USER CONSTRAINTS 2026-09-05: Machinery product must be standalone, never require Paivot/pvg/nd, workflow labels or commit conventions. Local development coordination only may use Paivot. Another agent uses installed Machinery in NIL: do not replace installed binary/plugins/skills/agents; no dev-link or live install/update. Build isolated candidate only. No GitHub push/mutation during work. Full scripts/preflight.sh only final epic gate. RED author may update preexisting tests that encode superseded unsafe behavior with explicit review and genuine assertion-failure proof; after RED approval freeze exact tests/fixtures/config bytes.
ANCHOR ROUND-1 RUNTIME CLASSIFICATION: This story's required current cases are service-free Go/native filesystem/local-process tests and real local CLI paths; no Docker/Java/Node dependency is implicit in ordinary native suites. Preserve actual non-mocked path tests. If implementation introduces any external runtime-backed case, it MUST add a dedicated closed fragment under testdata/integration-lanes via the shared required lane and declare ownership/dependency before delivery; no skip-if-missing, env-gated omission or reliance on later incidental execution. Missing service cannot silently convert required coverage to success.
PRE-RED PM EXACT TEST-EDIT AUTHORIZATION — MAC-sh60 — 2026-09-05

TEST-EDIT AUTHORIZED: internal/gates/oraclecov_test.go — replace ONLY the four positive fixture source expressions currently composed of package text + unused const oraclePath + parseEvidence with these calls to the new covActiveParser helper:
1. TestCheckOracleCoverageConformanceParse, impl/oracle_test.go:
   covActiveParser("thing", "../../design/machines/Thing.oracle.md")
2. TestCheckOracleCoverageConformanceParseIsNotSubstring, impl/purchase_test.go:
   covActiveParser("p", "../../design/machines/PurchaseOrder.oracle.md")
3. TestGtCitationBoundaries / "hyphenated sibling does not cover", impl/purchase_test.go:
   covActiveParser("p", "../../design/machines/purchase-order.oracle.md")
4. TestCheckOracleCoverageFormalOracles / "covered by file-name literal", impl/authz_test.go:
   covActiveParser("authz", "../../design/formal/Policy.oracle.md")

Reason: the four existing positive fixtures require wholesale coverage but contain no active test and only an unused delimiter splitter. That expectation encodes the unsafe evidence accepted by the assessed filename/delimiter heuristic. Replacing only their source expressions with a real active test parser preserves the intended positive and filename-boundary assertions while permitting AC1 rejection of unused declarations/functions.

Frozen edit boundary: preserve every existing assertion, count expectation, test/subtest name, oracle fixture, sibling spelling/case, remaining fixture, helper, import and unrelated byte. In particular keep parseEvidence and its existing negative/control uses unchanged. This is not permission to rewrite oraclecov_test.go generally or alter any existing command test. The commit containing these amendments must include BOTH tdd-red and [test-edit-authorized] in its subject. Once RED is committed/approved, its tests and fixtures are immutable through GREEN absent a separately named repair review.

Authorized helper semantics: covActiveParser(pkg, path string) string lives in NEW internal/gates/oraclecov_negative_test.go. The inspected helper emits syntactically valid Go with os/strings/testing imports and an active TestOracle; uses os.ReadFile with the supplied literal oracle path; iterates parsed table rows, trims cells, rejects absent stable-ID cells and zero exercised rows; compares the fixture transition A/on:go -> B and B/on:stop -> A to the oracle target, the formal fixture admin any -> allow and rep other -> deny to its expectation, and the minimal three-column boundary fixture source to A. It must not embed the fixture stable IDs in generated test text or use a comment/unused delimiter as the positive parser authority. This is a fixture implementation illustrating an active parser; it does not dictate the production recognition architecture or establish a general parser for arbitrary malformed table shapes.

The helper inspected at line 18 of the uncommitted new file is authorized for these four expressions. Full file inspection SHA256 was 8202fb949a883ac7bcbf5ebc7aa50505ee05e1f729a1dd169c47346e78ac64fc, independently matching the author's reported review snapshot. Other new-file tests may continue being authored before RED freeze; a material change to these authorized helper semantics needs further review. Whole-file final RED hash and diff must be reported independently.

Proof requirements retained: execute TestOracleCoverageParserRuntimeControl's real Go fixture using its actual existing oracle path. Require the unchanged native run to pass and log both checked rows. Change only the expected first transition target B -> A and require the reached assertion "oracle transition mismatch: got B want A", never compile/import/read-path/timeout failure. The four older source-only discovery fixtures retain their exact old relative path spellings; this authorization does not claim those independent temp-tree fixtures were themselves executed. The dedicated runtime fixture binds the actual path. Other-language source fixtures establish discovery only, not native execution evidence.

The full CheckOracleCoverage negative bypass tests must fail by the old production's observed unsafe acceptance, with the actual positive parser/literal controls passing. Keep stable-ID boundaries, orphan/missing-oracle and clause diagnostics; exact counts/assertion failures remain subject to independent RED replay. No filename-keyword-only wholesale inference is authorized. These tests must not imply static discovery proves assertions executed.

Confirmed directly associated AC3/5 scope: NEW cmd/machinery/oraclecov_negative_test.go may exercise a real isolated built CLI against actual temporary filesystem trees; existing command tests stay unchanged. Production ownership remains internal/gates/oraclecov.go; RED makes no production edits. This is not review/approval of the complete new test files, no final five-AC proof, and no approve-red, delivery, acceptance or reject transition.

Evidence (read-only review):
- Shared pvg nd show MAC-sh60 --json read full canonical story with all five AC. Before write: in_progress / hard-tdd / assignee dev-MAC-sh60 / parent MAC-ui8a.
- pm_acceptor rules previously read fully; developer skill read fully for phase and frozen-test rules.
- Graph index_status/search_graph (21 relevant results, has_more false), CheckOracleCoverage outgoing trace, check_index_coverage. Main graph generation 2026-09-05T23:58:53Z, best-effort metadata_match/no_recorded_issue for oraclecov.go and oraclecov_test.go; exact source reads establish this review's branch evidence.
- git show 6cb2d974:internal/gates/oraclecov_test.go and oraclecov.go verified all four source expressions, the real fixture tables and current fileNameCited/hasParseEvidence heuristic.
- Read new dev-MAC-sh60/internal/gates/oraclecov_negative_test.go fully through its current end; shasum -a 256 matched the review snapshot above.
- git -C .claude/worktrees/dev-MAC-sh60 rev-parse HEAD returned 6cb2d974ea8aea211a5974f453cef2b5802bb11e. git diff -- internal/gates/oraclecov_test.go internal/gates/oraclecov.go was empty; status showed only the two new untracked negative test files. PM did not edit either new file or any production/existing test.
- No PM tests, build, preflight, installation, main change, network mutation, claim release or workflow transition occurred. Author native proof is pending and not presumed from source inspection.
2026-09-24: branch story/MAC-sh60 (a6ac10eb, 9 unmerged commits) deleted after reassessment: every test case it added, including the final commits' constant_target_field and call_budget cases, exists on main in internal/gates/oraclecov_scope_test.go and oraclecov_negative_test.go via MAC-wi5z; only internal helper names differed. No residual value.

## Independent PM supplemental TDD audit adjudication — MAC-sh60
Reviewed candidate a6ac10ebadc0e6b6a344d50aa10d0b810ce6b940 on 2026-09-05. Disposition: NO FROZEN RED TAMPERING FOUND in the three flagged deltas; the mechanical verify-tdd result remains FAIL and has not been waived or converted to PASS. This is a bounded audit adjudication, not acceptance, rejection, delivery, claim release or a workflow transition.

### Exact timeline and delta findings
1. The approved original RED tip is fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3. The supplemental path internal/gates/oraclecov_scope_test.go does not exist there. git log a6ac10e --diff-filter=A identifies first addition as GREEN commit 4cda3c388c3efef2cdf952adb31d6c7507c307da at 2026-09-05T17:58:06-07:00, after production GREEN commits 6f2c165/e13a2fd. It is not an original RED file.
2. Commit 3d47b80ce59af1f521539e0b48e67be9ee4e9a71, 17:58:43-07:00, changes only construction of that new file's Python source_after_test fixture (+5/-1). Previously the appended Python body retained four spaces and therefore belonged to the preceding test function. The correction removes one indentation level when emitting source_after_test, making it the intended non-test source. It preserves all assertion conditions, helper_after_test/active fixtures and case names. This is correction of a newly created GREEN fixture, not repair, weakening or retroactive replacement of frozen RED.
3. My canonical scope review was recorded at 2026-09-06T01:23:55Z, explicitly authorizing additional regression tests in this existing GREEN supplemental file and stating they are not historical pre-GREEN RED. It reviewed the corrected 3d47b80 checkpoint and required the current 18 scope cases remain intact. My re-freeze/continuation decision at 01:32:02Z retained that same allowance. These authorizations precede both subsequent helper commits.
4. Commit 088ef8a6453df139aec6dcc9386f1021ab7ca6fc, 18:38:51-07:00, appends 66 lines containing TestOracleHelperDiscoveryRequiresConnectedRows. The already-reviewed first 54 lines and all 18 earlier cases are unchanged. The new positive direct/assigned helper-return cases and uncalled/unused/constant/wrong-oracle/uncalled-closure/recursive/ambiguous-path/unrelated-receiver/depth-bound negatives implement the previously authorized connected-provenance and bounded-analysis scope. No prior assertion is removed or weakened.
5. Commit a6ac10ebadc0e6b6a344d50aa10d0b810ce6b940, 18:41:02-07:00, adds only two supplemental matrix entries: constant_target_field and call_budget. They prevent inferred row provenance from a constant target and require exhausted helper work to remain uncovered. Both are within the same earlier authorized provenance/bounds scope. Assertions and preexisting fixtures remain unchanged.
6. git diff --numstat 3d47b80..a6ac10e -- internal/gates/oraclecov_scope_test.go is exactly 68 insertions/0 deletions. Cumulative candidate diff from 6cb2d974 is 932 changed LOC across the six authorized paths, inside the reviewed 1000-line/six-path ceiling. No history/commit marker edits are part of this review.

### Frozen evidence independently verified
The exact candidate git-object bytes hash to all four approved values:
- internal/gates/oraclecov_negative_test.go: 42f1237819cfecc4a265fd5dfa4a90050d8f0f77f9aa0baf27b42995d8834e45
- internal/gates/oraclecov_test.go: dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994
- cmd/machinery/oraclecov_negative_test.go: 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08
- testdata/golden/check-go-crm/stdout.txt: d14ccc0e945d7bd21bbb681894682aa64240a8eb9c995a815f3bf4068b77b184

Golden integration 7b200b6 carries only the separately approved/re-frozen disclosure amendment. No authorizations here permit further frozen-test changes. The supplemental additions are honestly GREEN-time work; they must not be relabeled as pre-GREEN RED.

### Mechanical audit and remaining policy boundary
Raw /tmp/MAC-sh60-green-final.8U4Ho2/tdd-audit.txt says the nine-commit range checked zero skipped merges and FAILS exactly the three supplemental-file commits above for lacking a tdd-red or [test-edit-authorized] commit-subject marker. The raw result is real and remains authoritative as the tool's result. The file-origin/delta evidence shows the flagged actions do not violate the substantive immutable-RED boundary, and the two later additive edits had prior explicit PM scope authorization. That is not the same as producing a clean audit.

The current pm_acceptor skill, /Users/ramirosalas/.codex/skills/pm_acceptor/SKILL.md:84, explicitly states: "A `verify-tdd` failure is a rejection." That sentence governs a future GREEN acceptance review. Neither the earlier bounded implementation/test-scope allowance nor this request to adjudicate the audit explicitly waives that categorical acceptance rule. Therefore this PM does not silently manufacture an audit pass or grant acceptance despite it. Since this is not a delivered acceptance review, no rejection transition is applied now.

Available bounded resolutions for the dispatcher to seek, without changing the product or rewriting evidence:
- Explicit user direction authorizing a one-time exception to the skill's mechanical-audit acceptance rule for exactly commits 3d47b80/088ef8a/a6ac10e and exactly internal/gates/oraclecov_scope_test.go, based on this independent no-frozen-RED-tampering finding. Keep the raw audit FAIL visible and all four frozen hashes protected. This would permit subsequent ordinary GREEN delivery/review; it would not itself accept the implementation.
- Separately authorize a future correction to the private audit's treatment of GREEN-created supplemental files, with dedicated proof, then rerun the audit. Such tool work is outside the current authorization and must not become a Machinery product dependency or a silent bypass. No such change is performed or assumed here.

History rewrite/rewording, retroactive tdd-red/test-edit marker insertion, product changes to satisfy private Paivot tooling, or claiming this command passed are not valid resolutions under the present constraints. Prior PM scope approval is recognized as evidence of authorized test work; it is not retrospectively presented as a waiver of the audit-return-code rule.

### Evidence scope and limitations
- Fully read healthy-hold-audit.md, tdd-audit.txt and all three referenced tdd-supplement-*.diff artifacts. Inspected the exact candidate commit sequence and first-added supplemental file, independently verified +68/-0 since the reviewed checkpoint and all four git-object hashes, and checked shared authorization timestamps/current status.
- The report records final exact native gates 130 PASS/0 FAIL/0 SKIP, CLI 10 PASS/0 FAIL/0 SKIP; full native gates 1187 PASS/1 SKIP and cmd 409 PASS/3 SKIP, with command package completing in 273.048s. Actual example proof records 74 passing native subtest leaves plus four unreachable Policy rows deliberately excluded. These are delivery artifacts for later full acceptance review, not a new acceptance decision in this bounded audit. No native rerun was needed to determine the three history/delta findings.
- Earlier covered-run failures remain separate historical evidence; MAC-yig6 owns the independently proved representative instrumentation/protocol defect. No claim attributes all 23 earlier failures to it. Filesystem/runtime opt-in skips remain unexecuted assurance. Earlier timeout correction remains intact: TestInstallCommand passed; three other installer tests were still active.
- Retained author worktree readback is clean at a6ac10ebadc0e6b6a344d50aa10d0b810ce6b940. No code, test, fixture, history, private tooling, installed assets, worktree or process state was modified. Only append-only review notes/comment are written. No full preflight, external runtime lane, remote/main/epic action or workflow transition occurred.

## nd_contract
status: in_progress
phase: green-audit-policy-disposition-pending

### evidence
- Independently verified the three flags are GREEN supplemental-only edits: initial fixture correction then previously authorized additive helper controls; all four approved frozen hashes unchanged, exact candidate 932 LOC/six paths.
- Mechanical verify-tdd remains FAIL for three missing-marker commits. No audit waiver, fabricated pass or acceptance decision made; exact policy sentence and bounded resolution choices recorded above.

### proof
- [x] Frozen RED preservation verified; no weakening in the three flagged supplemental deltas.
- [x] Prior authorization timeline for helper additions verified; corrected Python fixture was new GREEN work reviewed before helper expansion.
- [ ] Explicit disposition of mechanical-audit acceptance-rule conflict pending before a successful GREEN acceptance can be claimed.
- [ ] Formal GREEN delivery and full independent five-AC acceptance review remain pending.

## Independent PM golden RED re-freeze — MAC-sh60
APPROVED [2026-09-05]: exact golden amendment 8f843256d30068816234d008a85239d662e8bb15 is reviewed and re-frozen. The golden-amendment hold on healthy GREEN is released. The dispatcher may integrate ONLY this approved tests-only commit into retained healthy GREEN 3d47b80ce59af1f521539e0b48e67be9ee4e9a71, then resume the already authorized bounded helper compatibility work. This is not story acceptance, production approval or a workflow transition; no approve-red/deliver/accept/reject/claim/release command was invoked.

### Independent review evidence
- Read complete /tmp/MAC-sh60-golden-red.E4Jh4K/proof.md, exact commit metadata/diff, native golden/control JSON terminal events, full actual CLI stdout/stderr hashes and stdout.diff. Compared with my exact scope/golden authorization at /tmp/MAC-sh60-pm-scope.mXj9lU/scope-golden-authorization.md and current shared canonical status/terminal proof.
- git rev-parse 8f843256^ is fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3. Entire delta is one insertion/one deletion at testdata/golden/check-go-crm/stdout.txt:55. No other file changed; git diff --check passes. Commit subject contains tdd-red and [test-edit-authorized]. Production and the three previously approved frozen tests remain identical to pure RED.
- Exact appended suffix: `, static discovery; tests not executed; unsupported parser structures remain uncovered`. All preexisting counts, notably `2 formal oracles covered`, other stdout bytes, empty stderr and zero exit expectation remain unchanged.
- Native author command: go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestGoldenCheck$/^go-crm$'. Raw golden.jsonl confirms exactly one leaf TestGoldenCheck/go-crm FAIL at golden_test.go:246 stdout mismatch; parent TestGoldenCheck/package records are not additional leaves. No skip, stderr/exitcode assertion, setup/import/compile/path/timeout failure. Package terminal 2.083s; intended leaf 1.30s.
- Native author positive command: go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestCheckGreenSummaryLines$'. Raw control.jsonl confirms one PASS, zero fail/skip, test 0.65s.
- The separately built actual native CLI command in proof.md exits 0. Full actual-stdout.txt SHA256 independently equals the original RED golden: 2abeaf18bcc08300ffcf6083f0715c4790e12d26e017f59df0c71d8c89f0e12f. Actual-stderr.txt SHA256 is the empty-file hash e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855. Full stdout.diff independently contains only the precise suffix-only line difference, resolving the golden harness's clipped diagnostic. All prior counts and platform-green output therefore remain intact.
- New frozen stdout.txt SHA256 independently matches d14ccc0e945d7bd21bbb681894682aa64240a8eb9c995a815f3bf4068b77b184. Unchanged exitcode.txt SHA256 remains 9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa; unchanged stderr.txt remains empty.
- Proof is complete, consistent and attributable to this exact candidate. Per pm_acceptor evidence rules, no redundant native rerun was needed. The author's pvg verify result scanned zero text-golden files and is not treated as behavioral evidence; exact diff/hashes plus real native assertions provide the proof.
- Read-only status checks show the author's detached proof checkout remains clean and healthy GREEN remains clean at 3d47b80ce59af1f521539e0b48e67be9ee4e9a71. Neither was edited or removed by PM. All source/test/fixture bytes remain untouched by this review.

### Re-freeze and continuation boundary
The original RED author's exact one-line authorization is fulfilled and exhausted; it grants no continuing test-edit permission. The golden file is now frozen at the hash above alongside the original three frozen tests:
- internal/gates/oraclecov_negative_test.go: 42f1237819cfecc4a265fd5dfa4a90050d8f0f77f9aa0baf27b42995d8834e45
- internal/gates/oraclecov_test.go: dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994
- cmd/machinery/oraclecov_negative_test.go: 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08

Root owns integration; GREEN may not edit any of these frozen bytes. Any future fixture repair requires a separately named authorization and genuine tests-only RED proof. This later amendment is explicitly subsequent to the healthy GREEN checkpoint; it is not claimed as original pre-GREEN evidence.

The previously approved <=1000 cumulative changed LOC / six named paths and same-file helper/Scanner/typed-row/t.Run boundaries remain authoritative. Implement connected target-oracle-to-row-to-active-failing-comparison provenance and required negative controls, preserve existing example semantics and formal counts, then run native example/targeted/full-cmd verification exactly as scoped. New helper tests in the GREEN supplemental file remain supplemental, not historical RED.

The finite full native cmd timeout allowance remains 10m without coverage instrumentation; timeout or skipped tests cannot count as completed assurance. Prior raw-log correction stands: TestInstallCommand passed; TestInstallAndDoctorTargetAll, TestInstallScript and TestInstallScriptHostTargets were active at the broad 180s timeout. Separate MAC-yig6 owns the proved coverage-instrumented fixture protocol defect; no claim attributes all 23 broad failures to it. No full preflight, external runtime lane, installed asset/binary changes, remote action, main/epic mutation or unrelated installer/OCI fix is authorized by this re-freeze.

## nd_contract
status: in_progress
phase: green-ready-after-golden-refreeze

### evidence
- Golden RED amendment 8f843256d30068816234d008a85239d662e8bb15 independently reviewed and frozen; exact native one-leaf disclosure FAIL and one positive PASS, zero skips, full CLI suffix-only difference proven.
- Healthy GREEN hold released for dispatcher-owned integration of only this approved tests commit and already-scoped helper implementation; current labels/status/claim unchanged.

### proof
- [x] AC4 golden amendment reviewed/re-frozen; complete prior counts, zero exit and empty stderr retained.
- [x] Original three frozen tests and pure RED production unchanged by amendment.
- [ ] Dispatcher integration and AC2 bounded helper compatibility/negative tests pending.
- [ ] Native examples, required targeted checks and complete full-cmd correctness verification pending.
- [ ] Five-AC GREEN delivery and independent acceptance pending; no production completion claimed.

## MAC-sh60 golden RED amendment proof

This amendment was authorized AFTER the healthy GREEN checkpoint 3d47b80ce59af1f521539e0b48e67be9ee4e9a71. It is an honest later tests-only amendment, not evidence claimed to predate GREEN implementation. The independent scope/golden authorization in /tmp/MAC-sh60-pm-scope.mXj9lU/scope-golden-authorization.md was read completely. No GREEN checkout/source/branch was modified.

### Commit and exact boundary
- New detached commit: 8f843256d30068816234d008a85239d662e8bb15
- Subject: test(MAC-sh60): tdd-red [test-edit-authorized] disclose static discovery in go-crm golden
- Parent: pure approved RED fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3; production still 6cb2d974ea8aea211a5974f453cef2b5802bb11e.
- Retained own clean detached proof checkout: /tmp/MAC-sh60-golden-red.E4Jh4K/review. No story branch moved.
- Exact entire diff: testdata/golden/check-go-crm/stdout.txt line 55 only, one insertion/one deletion. Appended exactly ", static discovery; tests not executed; unsupported parser structures remain uncovered" before the existing newline.
- Every old count preserved, including 2 formal oracles covered. All other stdout bytes, stderr.txt empty and exitcode.txt 0 plus newline unchanged. No update/regeneration flag, test/helper/example/source edit, golden bulk rewrite, or additional fixture.
- git diff --check passed; git status --short empty.

### Commands and native outcomes
All commands executed from /tmp/MAC-sh60-golden-red.E4Jh4K/review at the new committed SHA.

1. go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestGoldenCheck$/^go-crm$' > /tmp/MAC-sh60-golden-red.E4Jh4K/golden.jsonl
   Exit 1 EXPECTED RED. Exactly one native leaf: TestGoldenCheck/go-crm FAIL. TestGoldenCheck and package FAIL records are parents, not extra leaves. No skips.
   Sole assertion is golden_test.go:246 stdout golden mismatch. No stderr/exitcode mismatch; no compile/import/path/timeout error.
2. go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestCheckGreenSummaryLines$' > /tmp/MAC-sh60-golden-red.E4Jh4K/control.jsonl
   Exit 0. Exactly one native leaf: TestCheckGreenSummaryLines PASS; no fail/skip. Full positive behavior preserved on unchanged production.
3. go build -o /tmp/MAC-sh60-golden-red.E4Jh4K/machinery ./cmd/machinery
   Exit 0, isolated candidate binary only.
4. env HOME=/tmp/MAC-sh60-golden-red.E4Jh4K/private-home MACHINERY_CONFIG_DIR=/tmp/MAC-sh60-golden-red.E4Jh4K/private-config /tmp/MAC-sh60-golden-red.E4Jh4K/machinery check /tmp/MAC-sh60-golden-red.E4Jh4K/review/examples/go-crm/design --impl /tmp/MAC-sh60-golden-red.E4Jh4K/review/examples/go-crm/impl > /tmp/MAC-sh60-golden-red.E4Jh4K/actual-stdout.txt 2> /tmp/MAC-sh60-golden-red.E4Jh4K/actual-stderr.txt
   Exit 0. Actual stderr zero bytes. Actual stdout matches the original approved RED golden byte for byte: cmp with git show fa842ed3:testdata/golden/check-go-crm/stdout.txt exits 0. All original counts and platform-green preserved.
5. diff -u /tmp/MAC-sh60-golden-red.E4Jh4K/actual-stdout.txt testdata/golden/check-go-crm/stdout.txt > /tmp/MAC-sh60-golden-red.E4Jh4K/stdout.diff
   Exit 1 EXPECTED single exact line difference; full raw stdout captured because golden harness clips after 4000 bytes.
6. pvg verify testdata/golden/check-go-crm/stdout.txt --include-tests --format text
   VERIFY: PASSED (0 files scanned, 0 issues). This source scanner does not inspect the text golden; exact git diff, cmp, hashes and native assertion above provide the relevant proof, not the zero-file scan.

### Exact changed line (actual -> expected)
Actual:
  checked: 14 test files scanned, 5 machines, 275 oracle rows, 197 ids covered by literal, 2 formal oracles, 2 formal oracles covered, 6 clause-declared guards checked, 36 falsifying-clause ids covered
Expected:
  checked: 14 test files scanned, 5 machines, 275 oracle rows, 197 ids covered by literal, 2 formal oracles, 2 formal oracles covered, 6 clause-declared guards checked, 36 falsifying-clause ids covered, static discovery; tests not executed; unsupported parser structures remain uncovered

### SHA256
- stdout.txt BEFORE and actual CLI stdout: 2abeaf18bcc08300ffcf6083f0715c4790e12d26e017f59df0c71d8c89f0e12f
- stdout.txt AFTER: d14ccc0e945d7bd21bbb681894682aa64240a8eb9c995a815f3bf4068b77b184
- exitcode.txt unchanged: 9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa
- stderr.txt unchanged and actual CLI stderr: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
- frozen gate negatives unchanged: 42f1237819cfecc4a265fd5dfa4a90050d8f0f77f9aa0baf27b42995d8834e45
- frozen original gate tests unchanged: dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994
- frozen CLI negatives unchanged: 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08
- unchanged RED production oraclecov.go: 9f4bfabcfe43362b62068e59a59f313cd6d27a3490392f66c5fcf71be70684ad

### Limits and preserved state
- Canonical readback remains in_progress, assignee dev-MAC-sh60, hard-tdd/red-approved, parent MAC-ui8a. No deliver/approve-red/accept/reject/claim/release/state transition was invoked.
- Parent owns independent re-freeze review and integration into healthy GREEN; this amendment has not itself been approved or integrated.
- No full preflight, external runtime lane, coverage instrumentation, installed binary/asset/skill/plugin, remote/main/epic change or worktree removal. All bounded processes completed.
- Previous approved tests/authorizations and original RED evidence remain intact; only exact AC4 golden disclosure changes here. GREEN helper compatibility and broader verification remain the other agent's pending work.

LEARNINGS:
- Golden disclosure must preserve complete legacy counts; a missing formal-oracle recognition result is production regression, not golden expected behavior.
- Full CLI capture resolves golden output clipping and establishes the missing suffix as the sole behavioral RED cause.

## nd_contract
status: in_progress
phase: golden-red-amendment-awaiting-independent-review

### evidence
- Original RED author committed 8f843256d30068816234d008a85239d662e8bb15 on isolated approved-RED checkout after healthy GREEN checkpoint; exact authorized single-line golden change only.
- One intended golden native leaf FAIL, one positive summary leaf PASS, zero skips; actual CLI exits 0 with empty stderr and complete original counts. Full proof/raw logs/diff in /tmp/MAC-sh60-golden-red.E4Jh4K.

### proof
- [x] AC #4 amendment: exact disclosure suffix required by real golden assertion and RED cause independently reviewable.
- [x] Preservation: original counts, zero exit, empty stderr and three previously frozen test hashes unchanged.
- [ ] Independent amended-RED review/re-freeze and parent integration pending.
- [ ] GREEN AC2 helper compatibility and complete five-AC production delivery remain pending; no acceptance claim.


## PM bounded scope and golden authorization — MAC-sh60
Reviewed healthy GREEN checkpoint 3d47b80ce59af1f521539e0b48e67be9ee4e9a71 on 2026-09-05. This is a scope/budget and exact fixture-edit decision only. It is NOT delivery, rejection, re-RED approval, acceptance, claim release or authorization to merge. The current GREEN hold remains until the dispatcher completes the separate RED-amendment review and explicitly resumes GREEN.

### Scope decision: authorized within AC2
AC2 requires genuine conformance parsers to remain discoverable. The unchanged go-crm examples are concrete necessary compatibility, not speculative expansion:
- examples/go-crm/impl/internal/authz/oracle_test.go:25/31/131: literal const oraclePath -> loadOracle(t) -> os.Open(filepath.FromSlash(oraclePath)) -> bufio.NewScanner(f), sc.Scan/sc.Text -> strings.Split and trimmed cells -> append oracleRow fields -> returned rows -> active TestOracleConformance range -> want derived from row.expectation -> t.Run/nested ranges -> got.Allowed != want -> t.Errorf.
- examples/go-crm/impl/internal/authz/tenant_oracle_test.go:26/32/95: same connected loader/typed-row pipeline for Isolation.oracle.md; active TestTenantOracleConformance assigns returned rows, ranges them, derives want from row.expectation and checks AuthorizeLink(...).Allowed inside t.Run.
- The loader's own read-error, malformed-row and empty-result failures do not establish row conformance. The decisive evidence is the returned-row-dependent comparison in the active test. The Policy test intentionally excludes rows marked unreachable; discovery still must not assert that every row executed.
- Current fileNameCited rebuilds only active test declarations, so same-file loaders/constants are absent; goOracleParser handles direct ReadFile/range/Split and cannot follow these Scanner/typed-return/t.Run shapes. TestCheckGreenSummaryLines and TestGoldenCheck/go-crm demonstrate the regression. Preserving the examples rather than rewriting them into a new parser idiom is necessary for AC2.

Authorized implementation boundary: internal/gates/oraclecov.go only, plus directly associated additional regression tests in the existing GREEN supplemental internal/gates/oraclecov_scope_test.go. Permit bounded same-file reachable helper summaries, literal immutable path resolution including filepath.FromSlash, os.Open -> Scanner Scan/Text provenance, typed append/composite/selector propagation, returned rows consumed by active test ranges, and recognized testing.T.Run callbacks. This is an implementation allowance, not an assertion that any particular proposed algorithm is correct.

Required constraints:
1. Every wholesale credit must retain a connected target-oracle read -> actual parsed cells/row values -> returned collection -> active caller iteration -> row-dependent failing comparison. A parser-shaped helper, file name, delimiter, helper name, unused return value or unrelated assertion cannot confer credit.
2. Follow only resolvable same-file calls reached from valid active Go tests. Do not treat every function declaration or arbitrary function literal as invoked. Recognize t.Run through the active testing parameter/callback context; an uncalled closure or unrelated receiver's Errorf must not become a test assertion.
3. Resolve only proven literal/immutable path bindings and reject shadowed/reassigned or otherwise ambiguous bindings. Preserve exact target filename boundaries; a loader for a different oracle cannot cover the requested oracle. Do not pool unrelated helpers/files into a synthetic proof chain.
4. Use explicit finite cycle/depth bounds. Cycles, unresolved calls, ambiguous returns and exhausted analysis limits remain uncovered; limits cannot turn into success or hang the scanner. No general inter-package call analysis, external compiler/runtime, network execution or new dependency is authorized.
5. Preserve existing active test/build selection, comment/docstring rejection, stable-ID boundaries, orphan/missing-oracle and clause obligations. Static discovery remains distinct from execution.

Required supplementary proof before GREEN delivery: actual CheckOracleCoverage fixtures for both direct-range helper return and assign-then-range forms, representing literal const/FromSlash/Open/Scanner/typed rows/selectors/t.Run. Negative controls must cover uncalled loader; loaded-but-unused returned rows; real loader plus unrelated constant assertion; wrong-oracle loader; an uncalled assertion closure; and bounded recursion/ambiguous path reassignment remaining uncovered. These may be table-driven/additive in oraclecov_scope_test.go; use connected positive fixtures and exact no-credit diagnostics/counts. They are GREEN supplemental regression tests, not historical pre-GREEN RED. Current 18 supplementary cases and all frozen original tests remain intact.

Budget investigation: current cumulative diff against 6cb2d974 is exactly 709 changed LOC across five files (649 additions/60 deletions), with 54 lines of supplemental test coverage and no padding. The forecast 140–200 additional lines is justified by preserving the existing real parser idiom and adding safety controls. Authorize up to 1000 cumulative changed LOC across SIX exact paths: internal/gates/oraclecov.go, internal/gates/oraclecov_scope_test.go, the three already-frozen test files, and testdata/golden/check-go-crm/stdout.txt. Forecast remains about 850–910 plus explicit safety-test room; 1000 is a review ceiling, not a target. Any additional file or material growth beyond that bound needs renewed concrete review. No architect decision is presently required for this bounded same-file compatibility repair.

### Exact golden amendment authorization
TEST-EDIT AUTHORIZED: testdata/golden/check-go-crm/stdout.txt — change ONLY the existing Gt checked line (line 55 at this checkpoint) by appending exactly `, static discovery; tests not executed; unsupported parser structures remain uncovered` before its existing newline. This is required AC4 disclosure. Preserve every existing count, all other lines/bytes, stderr.txt (empty), exitcode.txt (`0\n`) and all command test assertions. No golden regeneration/update switch or broad fixture rewrite is authorized.

Exact OLD line:
```text
  checked: 14 test files scanned, 5 machines, 275 oracle rows, 197 ids covered by literal, 2 formal oracles, 2 formal oracles covered, 6 clause-declared guards checked, 36 falsifying-clause ids covered
```
Exact NEW line:
```text
  checked: 14 test files scanned, 5 machines, 275 oracle rows, 197 ids covered by literal, 2 formal oracles, 2 formal oracles covered, 6 clause-declared guards checked, 36 falsifying-clause ids covered, static discovery; tests not executed; unsupported parser structures remain uncovered
```

The two missing formal-oracle errors and loss of `2 formal oracles covered` in healthy GREEN are a production regression to fix. They must NOT be captured as expected golden output. Gate.Emit appends checkedExtra after the existing ordered counts with comma-space separation, supporting this exact suffix-only expectation once helper compatibility is repaired.

Before amendment SHA256 independently read from 3d47b80:
- stdout.txt: 2abeaf18bcc08300ffcf6083f0715c4790e12d26e017f59df0c71d8c89f0e12f
- exitcode.txt: 9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa
- stderr.txt: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855

Separate original RED author owns this amendment, not the GREEN implementer. Produce a tests-only commit on the pure approved RED candidate fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3 (production still 6cb2d974), with both `tdd-red` and `[test-edit-authorized]` in its subject. Only the single golden line may change. Independently replay `go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestGoldenCheck$/^go-crm$'`: expected RED must be specifically the missing disclosure suffix while actual exit stays zero, old counts remain complete and stderr stays empty. Preserve full raw CLI stdout or an exact line diff if the golden harness clips its mismatch. Positive control `go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestCheckGreenSummaryLines$'` must pass on that unchanged production. Record SHA, diff, hashes, counts and assertion causes for separate independent PM re-RED review. This authorization does not itself approve the amended RED. The dispatcher owns integration with the paused healthy GREEN; no source/test edits occur in this review.

The three previously frozen tests remain byte-for-byte immutable at their approved hashes: gate negative 42f1237819cfecc4a265fd5dfa4a90050d8f0f77f9aa0baf27b42995d8834e45; original gate tests dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994; CLI negatives 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08. git diff fa842ed3..3d47b80 over those paths is empty. The golden authorization grants no permission to edit them or either real go-crm example.

### Verification allowance and honest limits
- After resumed implementation and approved golden amendment: rerun frozen gate/CLI tests and all supplementary helper/scope cases; run TestCheckGreenSummaryLines and TestGoldenCheck/go-crm unchanged. Also run actual local `go test -json -count=1 -timeout=120s ./internal/authz -run '^(TestOracleConformance|TestTenantOracleConformance)$'` from examples/go-crm/impl as real native example proof; keep any intentional unreachable-row exclusion explicit. No source-discovery claim substitutes for native execution.
- For the needed full cmd package correctness replay, authorize `go test -json -count=1 -timeout=10m ./cmd/machinery` without coverage instrumentation. This is a finite package budget, not a waiver for hung tests. Local `go help testflag` confirms Go's default timeout is ten minutes; repository CI/preflight native/race commands omit a shorter timeout. The broad instrumented package had already spent about 139 seconds before its final parallel installer group resumed, with TestGoldenGen 18.23s and TestPreflightReturnsOutputFailure 19.73s among observed costs. Three minutes was not evidence of an appropriately budgeted complete run. Ten minutes restores the tool's normal cap while preserving test/process deadlines and requiring terminal results. If it times out, diagnose/report the actual running tests; do not call it a pass or increase indefinitely.
- Correction to healthy-hold.md: the raw log shows TestInstallCommand PASSED in 8.96s. At the 180-second package timeout, the running tests were TestInstallAndDoctorTargetAll, TestInstallScript and TestInstallScriptHostTargets, each shown at 41s. This review does not assign their cause. Existing installer work remains separately owned. The fuller timeout is permission for accurate verification, not assurance those tests are healthy.
- Broader instrumented run remains gates 1171 pass/0 fail/1 skip; cmd 381 pass/23 fail/3 skip terminal leaves plus timeout. It did not complete. Report all failures and unexecuted conditions; do not claim all 23 share one cause. Read-only baseline diagnostic proves only the representative TestVerifyCheckersReproducible instrumentation protocol issue (native main pass 1.203s, covered fail 0.747s; GOCOVERDIR warning appended to strict OCI JSON). Root's separate P0 MAC-yig6 owns that defect; no OCI/parser/environment changes are authorized here and strict identity validation must remain intact.
- Case-sensitive filesystem tests and the official Structurizr/OCI engine opt-in cases remain unexecuted assurance in the supplied run. Their explicit lanes/final gate own required prerequisite coverage; list skips exactly and do not count them as passes. No full scripts/preflight.sh or external runtime lane is authorized during this review/story step.

### Review evidence and source boundary
- Read full /tmp/MAC-sh60-green.GtkNbt/healthy-hold.md and /tmp/machinery-coverage-diagnostic.hKOuC4/REPORT.md; reread canonical five AC and exact prior test-edit restrictions/current GREEN checkpoint via shared pvg nd.
- Read exact candidate oraclecov.go/supplemental test source, complete two actual go-crm oracle tests, golden stdout/exit/stderr, TestGoldenCheck, TestCheckGreenSummaryLines, Gate.Emit and relevant installer test sources. No candidate implementation was modified or executed in this scope-only review; existing raw test evidence is identified as author/root evidence above.
- Graph search found seven exact scoped symbols with has_more false; TestOracleConformance outbound trace included loadOracle and its actual authorizer callees. Direct source established the testing.T.Run and value flows; graph's unrelated heuristic CLI Run edge was not relied upon. Coverage generation 2026-09-05T23:58:53Z reports metadata_match/no recorded issues for unchanged examples and harness, but main-index metadata does not establish GREEN source freshness. New supplementary/golden paths were missing/not_tracked in graph metadata; exact git-show source is authoritative fallback. No graph completeness claim.
- Initial/final state must remain in_progress, hard-tdd/red-approved, assignee dev-MAC-sh60. No story transition, delivery, test/source/golden edit, worktree cleanup, installation, live binary/plugin/skill mutation, remote mutation or main/epic change performed. Only append-only shared review notes/comment are written.

## nd_contract
status: in_progress
phase: green-healthy-hold

### evidence
- Bounded same-file helper compatibility and <=1000 changed LOC / six-path scope authorized for AC2; exact one-line AC4 golden amendment authorized to separate original RED author, pending tests-only replay and independent review.
- Healthy checkpoint 3d47b80ce59af1f521539e0b48e67be9ee4e9a71 retained; frozen three-file delta empty. No workflow state transition in this review.

### proof
- [x] Scope review: genuine existing helper/Scanner/typed-row/t.Run evidence and safety constraints identified.
- [x] Exact golden edit authority: disclosure suffix only; counts, exit zero and empty stderr preserved.
- [ ] Separate tests-only golden RED amendment and independent review pending.
- [ ] AC2 GREEN helper compatibility, supplementary negative controls and actual example replay pending.
- [ ] Complete native correctness verification and honest failure/skip accounting pending; no acceptance or completion claimed.

## PM RED Decision
RED APPROVED [2026-09-05]: MAC-sh60 candidate fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3. This approves the immutable RED acceptance bar for GREEN; it does not accept or close the story and does not claim production satisfies the ACs.

The previous rejection is resolved. Diff from b5d3b8c67f487195215860f3d361432f5b2b6b27 is exactly six inserted entries, zero removals, in TestOracleCoverageRejectsNonExecutableEvidence. Fixture bytes/names match the six expressly authorized Elixir @moduledoc/@doc and Ruby block-comment cases. The existing twenty cases, shared assertions, helper semantics, every other test and all production bytes are unchanged. Commit subject contains tdd-red and [test-edit-authorized]. The four older fixture repairs remain exactly their prior reviewed forms.

## Independent replay evidence
- Reviewed full five-AC canonical story, original and supplemental exact authorizations, prior rejection, new complete proof/LEARNINGS/inventory via shared pvg nd. pm_acceptor and codebase-memory skills were fully read during the first review and applied on this resumed review. Prior source/graph assessment remains valid for unchanged bytes; exact new six-line diff is the source authority, with no repeated graph completeness claim.
- Reviewed in own clean detached /tmp/MAC-sh60-pm-rered.NMFVGG/review at fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3. Production base remains 6cb2d974ea8aea211a5974f453cef2b5802bb11e. Rework +6/-0; cumulative 3 files, 291 insertions/8 deletions (299 changed LOC), within budget. git diff --check passed.
- Fresh static pvg verify internal/gates/oraclecov_negative_test.go internal/gates/oraclecov_test.go cmd/machinery/oraclecov_negative_test.go --include-tests --format text: sole existing oraclecov_test.go:158 TODO marker is unchanged quoted negative fixture input. Source-confirmed test data, not a stub; no suppression/repair. No new public API/config or changed product behavior/doc claim in this RED-only delta.
- Command 1 from detached candidate: go test -json -count=1 -timeout=120s ./internal/gates -run 'Oracle|Conformance|Coverage|GtCitationBoundaries' > /tmp/MAC-sh60-pm-rered.NMFVGG/gates.jsonl. Session 66126 completed exit 1: 96 native leaves, 68 pass/28 intended fail/0 skip.
- Command 2: go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestOracleCoverageCLI' > /tmp/MAC-sh60-pm-rered.NMFVGG/cli.jsonl. Session 32924 completed exit 1: 10 native leaves, 4 pass/6 intended fail/0 skip.
- Command 3: go test -json -count=1 -timeout=90s ./internal/gates -run '^TestGt' > /tmp/MAC-sh60-pm-rered.NMFVGG/gt-controls.jsonl. Completed exit 0: 7 pass/0 fail/0 skip. Two boundary leaves repeat the first run; counts are per command, not summed unique.
- Exact independent leaf inventories: /tmp/MAC-sh60-pm-rered.NMFVGG/{gates,cli,gt-controls}-inventory.json, derived from terminal pass/fail/skip records with parent suites excluded. Raw logs retained at the paths above.
- Every new leaf reaches oraclecov_negative_test.go:88 "non-executable evidence established coverage". elixir_moduledoc_ids, elixir_doc_ids and ruby_block_comment_ids return errs=[] and 2 literal IDs; elixir_moduledoc_parser, elixir_doc_parser and ruby_block_comment_parser return errs=[] and 1 conformance parse. All six are actual false-credit failures, not setup/parse/native-compile errors.
- Existing 20 source negatives, mixed-disabled count/diagnosis and two discovery-label tests retain their intended causes. Actual CLI controls pass and five bypasses fail because real Gt exits 0 instead of 1. Built native CLI, temporary real trees and private HOME/config path remain unchanged and executed; no mocks/injected results.
- Native parser control ran again: unchanged table inner TestOracle exits 0 after exactly two checked-row logs; changing only the first expected target B -> A reaches "oracle transition mismatch: got B want A", inner exit 1 and outer corrupt_true PASS. Other-language fixtures establish discovery only, not native execution. No compile/import/read-path/timeout failure is RED evidence.
- Independently matched SHA256: internal/gates/oraclecov_negative_test.go 42f1237819cfecc4a265fd5dfa4a90050d8f0f77f9aa0baf27b42995d8834e45; internal/gates/oraclecov_test.go dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994; cmd/machinery/oraclecov_negative_test.go 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08; unchanged internal/gates/oraclecov.go 9f4bfabcfe43362b62068e59a59f313cd6d27a3490392f66c5fcf71be70684ad.
- Coverage percentage not measured for intentionally failing RED with no production edits; behavioral counts and exact inventories above establish the reviewed bar. No global preflight, external-runtime suite, installation/dev-link/live binary/plugin/skill change, remote action, main/epic merge, or author worktree edit.
- Canonical pvg story approve-red MAC-sh60 succeeded from the detached candidate. Immediate shared readback: Status open, Labels hard-tdd/red-approved (delivered/rejected absent), Assignee dev-MAC-sh60 retained by canonical transition. Root dispatcher owns the next GREEN claim. No --next or accept/close used.

LEARNINGS:
- Six precise language-comment regressions close the reproduced RED gap without weakening parser positives or native assertion-sensitivity controls.
- GREEN must preserve all approved RED tests/fixtures byte-for-byte. Prior authorizations have been fulfilled and provide no continuing license for edits; any new repair requires a separately named reviewer authorization and re-RED.

## nd_contract
status: new
phase: red-approved

### evidence
- RED fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3 independently approved via canonical approve-red; shared status open with hard-tdd/red-approved. Ready for GREEN, not accepted.
- Fresh targeted replay: gates96 = 68 pass/28 intended fail; CLI10 = 4 pass/6 intended fail; extraGt7 pass; zero skips. Exact artifacts, causes and hashes above.

### proof
- [x] AC #1 RED bar: 26 non-executable source cases plus mixed-disabled coverage constrain reproduced bypasses, including all six prior rejection cases.
- [x] AC #2 RED bar: 11 literal language/extension and 7 parser language controls plus 2 native parser semantic/mutation controls pass unchanged; ambiguous evidence negatives remain.
- [x] AC #3 RED bar: full CheckOracleCoverage bypass/zero/mixed/malformed/positive cases and actual CLI integration executed.
- [x] AC #4 RED bar: unit and CLI discovery-versus-execution assertions reach intended failures; static discovery never claims assertions ran.
- [x] AC #5 RED bar: stable-ID/file boundaries, orphan/missing-oracle, parent/clause and Rust/MJS preservation controls pass; actual CLI proof remains valid.
- [ ] GREEN: implement all five ACs with approved RED bytes unchanged, then deliver independent production proof for PM acceptance.

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


## Implementation Evidence

PROOF:
RED rework delivery responding to the complete PM rejection and exact add-only six-case TEST-EDIT AUTHORIZED note. Prior original RED evidence remains historical; these results supersede its candidate/counts. No production changes.

### CI/Test Results

Commands run:
- go test -json -count=1 -timeout=120s ./internal/gates -run 'Oracle|Conformance|Coverage|GtCitationBoundaries' > /tmp/MAC-sh60-red-rework.Ftwvzy/gates.jsonl
- go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestOracleCoverageCLI' > /tmp/MAC-sh60-red-rework.Ftwvzy/cli.jsonl
- go test -json -count=1 -timeout=90s ./internal/gates -run '^TestGt' > /tmp/MAC-sh60-red-rework.Ftwvzy/gt-controls.jsonl

Summary: gates exit 1 EXPECTED RED: 96 native leaves, 68 pass/28 intended fail/0 skip; CLI exit 1 EXPECTED RED: 10 leaves, 4 pass/6 intended fail/0 skip; extra Gt exit 0: 7 pass/0 fail/0 skip. Two boundary leaves repeat between gates and extra Gt, so counts are per run, not summed unique. New gate failures +6 only, existing passing counts unchanged. All terminal handles completed; no background processes retained.

Coverage: code coverage percentage not measured in intentionally failing RED with no production implementation. Behavioral matrix/exact leaf inventory below. Matching *-inventory.json artifacts in /tmp/MAC-sh60-red-rework.Ftwvzy preserve every native leaf and result, derived from terminal pass/fail/skip records after excluding parents.

### Exact six-case authorized delta and reached failures
In TestOracleCoverageRejectsNonExecutableEvidence, added ONLY:
1. elixir_moduledoc_ids: rows_test.exs containing @moduledoc triple-quoted THIN-aaa111 THIN-bbb222.
2. elixir_doc_ids: rows_test.exs containing @doc triple-quoted IDs and def helper, do: :ok.
3. elixir_moduledoc_parser: rows_test.exs containing @moduledoc triple-quoted parse "Thing.oracle.md" using "|".
4. elixir_doc_parser: rows_test.exs containing @doc triple-quoted parser phrase and def helper, do: :ok.
5. ruby_block_comment_ids: rows_spec.rb with =begin / IDs / =end on separate lines.
6. ruby_block_comment_parser: rows_spec.rb with =begin / parse "Thing.oracle.md" using "|" / =end on separate lines.
Exact fixture bytes read from /tmp/MAC-sh60-pm-red.2yPht9/probes/<case>/ and inserted with ordinary Go string escaping, retaining actual source line breaks and spaces.

Each new leaf reaches oraclecov_negative_test.go:88 assertion "non-executable evidence established coverage". Three *_ids results: errs=[] counts=map[ids covered by literal:2 machines:1 oracle rows:2 test files scanned:1]. Three *_parser results: errs=[] counts=map[machines:1 machines covered by conformance parse:1 oracle rows:2 test files scanned:1]. Thus every added test is genuine assertion-failure RED on unchanged production, not compilation/import/read-path/fixture/runtime infrastructure failure.

Native semantic controls reran as part of the full gate selection. corrupt_false: inner native TestOracle exit nil, two "checked oracle row" logs, PASS. corrupt_true: changing only expected first target B -> A produces inner native exit status 1 and reached "oracle transition mismatch: got B want A"; the outer control PASS confirms assertion sensitivity. Other-language source fixtures are discovery only, never a claim their native tests executed.

### Commit
Branch: story/MAC-sh60
SHA: fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3
Subject: test(MAC-sh60): tdd-red [test-edit-authorized] cover Elixir docs and Ruby block comments
Original RED SHA retained: b5d3b8c67f487195215860f3d361432f5b2b6b27
Production base remains 6cb2d974ea8aea211a5974f453cef2b5802bb11e; GREEN pending separate agent.
Exact rework diff from original RED: one file, six inserted lines / zero removed. Existing 20 entries, shared assertion body, all helper semantics and every other test byte retained. Existing oraclecov_test.go, CLI negative file and oraclecov.go diff from original RED empty. git diff --check passed and worktree clean.
Cumulative story cost: three files, 291 insertions/8 deletions = 299 changed LOC. Gate negative file 205 lines; CLI negative file 82 lines; original gate tests retain exactly four previously authorized fixture replacements (4 inserted/8 deleted).

### Frozen SHA256
- internal/gates/oraclecov_negative_test.go: 42f1237819cfecc4a265fd5dfa4a90050d8f0f77f9aa0baf27b42995d8834e45 (updated by authorized six-line addition)
- internal/gates/oraclecov_test.go: dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994 (unchanged)
- cmd/machinery/oraclecov_negative_test.go: 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08 (unchanged)
- internal/gates/oraclecov.go: 9f4bfabcfe43362b62068e59a59f313cd6d27a3490392f66c5fcf71be70684ad (unchanged production)

### pvg verify
Command: pvg verify internal/gates/oraclecov_negative_test.go internal/gates/oraclecov_test.go cmd/machinery/oraclecov_negative_test.go --include-tests --format text
Output: VERIFY: FAILED (3 files scanned, 1 issues), sole existing oraclecov_test.go:158 quoted TODO inside negative comment fixture. No stub/thin files. PM independently reviewed and explicitly preserves this test-data marker; no suppression/edit applied. This is not unfinished implementation.

### Wiring and limits
Real CheckOracleCoverage over temporary filesystem trees; real worktree-built CLI via preserved goldenBin/runBinWithEnv; unchanged private HOME/config isolation. Local Go/native filesystem/process only, no mocked/injected result, no external runtime dependency or skip gate. No full preflight (final epic only), main/epic/remote/installed binary/plugin/skill mutation. Parent coordinates PM, GREEN and merges.

### AC Verification
- [x] AC #1: full 26-case non-executable matrix now includes exactly the six PM-reproduced Elixir docs/Ruby block comments, all failing on unsafe coverage; mixed-disabled behavior also constrained. GREEN pending.
- [x] AC #2: all 11 literal extension, 7 active parser language and 2 native runtime semantic controls still pass unchanged. Unsupported/ambiguous source negatives remain.
- [x] AC #3: full actual CheckOracleCoverage matrix expanded +6; existing zero/mixed/malformed/positive and actual CLI selections rerun without weakening.
- [x] AC #4: existing unit and actual CLI discovery-versus-execution assertions still reach intended RED.
- [x] AC #5: existing stable-ID/file boundaries, orphan/missing oracle, parent/clause, Rust/MJS and real CLI proof retained and rerun.

LEARNINGS:
- The original RED matrix covered Python docstrings and Elixir hash comments but missed Elixir documentation attributes and Ruby embedded documents. PM reproduced the concrete gap and these six cases now constrain it.
- Preserve native semantic parser success and changed expected-target controls during language lexical coverage repairs.
- Six additive cases close this reviewed RED gap without changing production or previously frozen assertions.

### Exact native leaf inventory (gates, CLI, extra Gt command order)
pass	TestCheckAcceptanceDoDIDCoverage
pass	TestCheckAcceptanceOracleSetExpandsToExactStableIDInventory
pass	TestCheckAcceptanceRejectsMalformedOracleSet
pass	TestAdjudicationMissingOracleFails
pass	TestAttestationCoverageWarnsRatherThanBlocks
pass	TestCheckBuildPlanNoCommittedOracles
pass	TestGkRejectsFailedCoverageRowUnderPassVerdict
pass	TestGkCoverageGapIsError
pass	TestGkResidualWaivesCoverage
pass	TestClauseCoverageEmptyDeclarationErrors
pass	TestClauseCoverageComplete
pass	TestClauseCoverageMissingSuffixErrors
pass	TestClauseCoverageWholesaleParseDoesNotDischarge
pass	TestClauseCoverageUndeclaredGuardCarriesNoObligation
pass	TestClauseCoverageUnguardedRowsExempt
pass	TestOracleIDsInCommentsDoNotCover
pass	TestOracleVersionOnlySkewIsNotDrift
pass	TestOracleContentDriftStillDrift
pass	TestOracleMissingStampIsFreshAndSilent
pass	TestOracleCurrentStampIsSilent
pass	TestIDCiteRemovedOracleTagStillErrors
pass	TestIDCiteNoOraclesWarns
pass	TestCheckIsolationStaleOracleIsDrift
pass	TestObligationParentDeletedRelationalOracleRemainsRequired/gp
pass	TestObligationParentDeletedRelationalOracleRemainsRequired/gn
pass	TestObligationParentSelectionRetainsRelationalCoverage/Policy.oracle.md
pass	TestObligationParentSelectionRetainsRelationalCoverage/Isolation.oracle.md
fail	TestOracleCoverageRejectsNonExecutableEvidence/quoted_go_comment
fail	TestOracleCoverageRejectsNonExecutableEvidence/go_block_comment
fail	TestOracleCoverageRejectsNonExecutableEvidence/unused_go_constants
fail	TestOracleCoverageRejectsNonExecutableEvidence/uncalled_go_parser
fail	TestOracleCoverageRejectsNonExecutableEvidence/go_lowercase_test_helper
fail	TestOracleCoverageRejectsNonExecutableEvidence/go_invalid_test_signature
fail	TestOracleCoverageRejectsNonExecutableEvidence/disabled_go_literal
fail	TestOracleCoverageRejectsNonExecutableEvidence/disabled_go_parser
fail	TestOracleCoverageRejectsNonExecutableEvidence/legacy_disabled_go_literal
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_hash_ids
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_hash_parser
fail	TestOracleCoverageRejectsNonExecutableEvidence/python_module_docstring
fail	TestOracleCoverageRejectsNonExecutableEvidence/python_test_docstring
fail	TestOracleCoverageRejectsNonExecutableEvidence/python_parser_docstring
fail	TestOracleCoverageRejectsNonExecutableEvidence/js_unused_declarations
fail	TestOracleCoverageRejectsNonExecutableEvidence/python_unused_declarations
fail	TestOracleCoverageRejectsNonExecutableEvidence/ruby_unused_declarations
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_unused_declarations
fail	TestOracleCoverageRejectsNonExecutableEvidence/go_unrelated_split
fail	TestOracleCoverageRejectsNonExecutableEvidence/go_read_without_row_checks
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_moduledoc_ids
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_doc_ids
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_moduledoc_parser
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_doc_parser
fail	TestOracleCoverageRejectsNonExecutableEvidence/ruby_block_comment_ids
fail	TestOracleCoverageRejectsNonExecutableEvidence/ruby_block_comment_parser
fail	TestOracleCoverageMixedActiveAndDisabled
pass	TestOracleCoverageLiteralLanguageControls/rows.test.jsx
pass	TestOracleCoverageLiteralLanguageControls/rows.test.ts
pass	TestOracleCoverageLiteralLanguageControls/rows.test.mjs
pass	TestOracleCoverageLiteralLanguageControls/rows.test.cjs
pass	TestOracleCoverageLiteralLanguageControls/test_rows.py
pass	TestOracleCoverageLiteralLanguageControls/rows_spec.rb
pass	TestOracleCoverageLiteralLanguageControls/rows_test.exs
pass	TestOracleCoverageLiteralLanguageControls/tests/rows.rs
pass	TestOracleCoverageLiteralLanguageControls/rows_test.go
pass	TestOracleCoverageLiteralLanguageControls/rows.test.tsx
pass	TestOracleCoverageLiteralLanguageControls/rows.test.js
pass	TestOracleCoverageActiveParserLanguageControls/tests/rows.rs
pass	TestOracleCoverageActiveParserLanguageControls/rows_test.go
pass	TestOracleCoverageActiveParserLanguageControls/rows.test.js
pass	TestOracleCoverageActiveParserLanguageControls/rows.test.ts
pass	TestOracleCoverageActiveParserLanguageControls/test_rows.py
pass	TestOracleCoverageActiveParserLanguageControls/rows_spec.rb
pass	TestOracleCoverageActiveParserLanguageControls/rows_test.exs
pass	TestOracleCoverageParserRuntimeControl/corrupt_false
pass	TestOracleCoverageParserRuntimeControl/corrupt_true
pass	TestOracleCoverageMalformedReferencesStayUncovered/Thing.oracle.md.bak
pass	TestOracleCoverageMalformedReferencesStayUncovered/NotThing.oracle.md
pass	TestOracleCoverageMalformedReferencesStayUncovered/purchase-Thing.oracle.md
fail	TestOracleCoverageOutputLabelsDiscoveryOnly
pass	TestCheckOracleCoverageClean
pass	TestCheckOracleCoverageRejectsAndIgnoresOrphanOracle
pass	TestCheckOracleCoverageMissingIDs
pass	TestCheckOracleCoverageIgnoresProductionSources
pass	TestCheckOracleCoverageNoTestFilesFailsLoudly
pass	TestGtCitationBoundaries/hyphenated_sibling_does_not_cover
pass	TestGtCitationBoundaries/a_suffixed_artifact_is_not_the_oracle
pass	TestCheckOracleCoverageRustTestShapes
pass	TestCheckOracleCoverageConformanceParse
pass	TestCheckOracleCoverageConformanceParseIsNotSubstring
pass	TestCheckOracleCoverageMachinesWithoutOracles
pass	TestCheckOracleCoverageMachineMissingItsOracle
pass	TestCheckOracleCoverageNoMachines
pass	TestCheckOracleCoverageFormalOracles/covered_by_file-name_literal
pass	TestCheckOracleCoverageFormalOracles/uncovered
pass	TestCheckOracleCoverageCapsOffenderList
pass	TestCheckOracleCoverageHonorsContractIgnore
pass	TestCheckOracleCoverageScansMjsTestFiles
pass	TestOracleCoverageCLIRealTrees/literal_control
pass	TestOracleCoverageCLIRealTrees/parser_control
fail	TestOracleCoverageCLIRealTrees/quoted_comment
fail	TestOracleCoverageCLIRealTrees/unused_constants
fail	TestOracleCoverageCLIRealTrees/disabled_go
fail	TestOracleCoverageCLIRealTrees/elixir_comment
pass	TestOracleCoverageCLIRealTrees/zero_tests
fail	TestOracleCoverageCLIRealTrees/mixed_disabled
pass	TestOracleCoverageCLIRealTrees/malformed_reference
fail	TestOracleCoverageCLILabelsDiscoveryOnly
pass	TestGtWholesaleNeedsStringLiteralAndParseEvidence/comment_mention_covers_nothing
pass	TestGtWholesaleNeedsStringLiteralAndParseEvidence/quoted_mention_without_parse_evidence_covers_nothing
pass	TestGtWholesaleNeedsStringLiteralAndParseEvidence/parse_evidence_must_live_in_the_citing_file
pass	TestGtCitationBoundaries/hyphenated_sibling_does_not_cover
pass	TestGtCitationBoundaries/a_suffixed_artifact_is_not_the_oracle
pass	TestGtRustProductionTextIsNotTestCorpus
pass	TestGtCorpusSurvivesUnreadableDir


## nd_contract
status: delivered

### evidence
- RED rework fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3; raw JSON /tmp/MAC-sh60-red-rework.Ftwvzy; gates96=68 pass/28 intended fail, CLI10=4 pass/6 intended fail, Gt7pass, zero skips. Exact six additive entries and hashes recorded above.

### proof
- [x] AC #1: 26 negatives include six authorized supported-language documentation/comment failures.
- [x] AC #2: unchanged literal/parser and native semantic controls pass.
- [x] AC #3: expanded full-gate regression and real CLI matrix executed.
- [x] AC #4: unchanged discovery-label RED assertions reached.
- [x] AC #5: preservation controls and real CLI path rerun; GREEN implementation pending.


## nd_contract
status: rejected

### evidence
- PM rejection applied via pvg story reject on 2026-09-05.

### proof
- [ ] Story requires another developer delivery before it can be accepted.


## Implementation Evidence

PROOF:

### CI/Test Results

Commands run:
- go test -json -count=1 -timeout=120s ./internal/gates -run 'Oracle|Conformance|Coverage|GtCitationBoundaries'
- go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestOracleCoverageCLI'
- go test -json -count=1 -timeout=90s ./internal/gates -run '^TestGt'

Summary: expected RED gates 68 pass / 22 fail / 0 skip (90 leaves); CLI 4 pass / 6 fail / 0 skip (10 leaves); extra Gt 7 pass / 0 fail / 0 skip (2 repeated boundary leaves). Every failure is intended behavior: 20 unsafe-evidence acceptance unit cases, mixed-disabled unit, 5 actual CLI false-green exits and 2 discovery-label failures. Native parser controls pass after verifying 2 rows and reaching expected-target mutation error 'oracle transition mismatch: got B want A'. No compile/import/path/timeout failure used as RED evidence.

Coverage: behavioral matrix enumerated in previous RED Delivery note and exact JSON inventories; code coverage percentage not measured in RED.
Raw terminal artifacts: /tmp/MAC-sh60-red-proof.D3QqXS/gates.jsonl, cli.jsonl, gt-controls.jsonl and matching *-inventory.json. Exact leaf inventory and producing commands also preserved in Notes.

### Commit
Branch: story/MAC-sh60
SHA: b5d3b8c67f487195215860f3d361432f5b2b6b27
RED-only tests; unchanged production base 6cb2d974ea8aea211a5974f453cef2b5802bb11e. GREEN pending fresh agent.
Diff budget: 3 files, 285 inserted/8 removed. New gate tests 199 lines; new real CLI tests 82 lines; existing test four exact independent PM-authorized fixture expressions (4 added/8 removed), assertions and unrelated bytes preserved.

### Frozen test SHA256
- internal/gates/oraclecov_negative_test.go: 77940442bc579b2ec64715c79555d89c44dd27e498e49d296899e2d8f90ff042
- internal/gates/oraclecov_test.go: dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994
- cmd/machinery/oraclecov_negative_test.go: 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08
- unchanged internal/gates/oraclecov.go: 9f4bfabcfe43362b62068e59a59f313cd6d27a3490392f66c5fcf71be70684ad

### pvg verify
Command: pvg verify internal/gates/oraclecov_negative_test.go internal/gates/oraclecov_test.go cmd/machinery/oraclecov_negative_test.go --include-tests --format text
Output: VERIFY: FAILED (3 files scanned, 1 issues). Sole issue is deliberate existing quoted TODO input at oraclecov_test.go:158. No stub/thin findings. PM explicitly preserves this existing comment-rejection fixture; marker is test data, not unfinished implementation. Initial unsupported --format=text was corrected after reading help. git diff --check passed; worktree clean.

### Wiring and limits
Actual CheckOracleCoverage calls; actual built worktree CLI via unchanged goldenBin/runBinWithEnv -> machinery check --gate gt using real temporary design/impl files and isolated HOME/config. Native Go parser fixture executes without service/mocks or skips. Other-language parser/literal fixtures establish discovery only, not runtime execution. No Docker/Java/Node runtime introduced. Full preflight deferred explicitly to final epic, no installed binary/skills/main/remote mutation.

### AC Verification
- [x] AC #1: 20 non-executable source negatives + mixed disabled test reach intended unsafe-coverage assertions; GREEN pending.
- [x] AC #2: 11 literal extension controls, 7 active parser language controls and 2 native semantic/mutation controls PASS; ambiguous evidence rejected by RED expectations.
- [x] AC #3: every assessment bypass, real full gate fixture, zero/mixed/malformed/positive and actual CLI matrix executed.
- [x] AC #4: unit and real CLI discovery-versus-execution label assertions fail as intended.
- [x] AC #5: stable-ID/file boundaries, orphan/missing-oracle, parent/clause, Rust/MJS preservation tests pass; real CLI path exercises false-green bypasses with expected RED.

LEARNINGS:
- Filename/delimiter matching credits unused/commented source; Go Test-prefix matching additionally credits native non-test helpers.
- Semantic parser control needs real result-vs-expected comparison and an altered expected-result assertion control.
- Static source discovery remains distinct from test execution; no production AC completion claimed by this RED delivery.

## nd_contract
status: delivered

### evidence
- RED b5d3b8c67f487195215860f3d361432f5b2b6b27; gate90=68 pass/22 intended fail, CLI10=4 pass/6 intended fail, extraGt7 pass; raw JSON/inventory/hashes above. Implementation pending GREEN.

### proof
- [x] AC #1: executable-evidence RED bar frozen and false-positive assertions reached.
- [x] AC #2: actual positive discovery fixtures and native semantic control pass.
- [x] AC #3: full CheckOracleCoverage and real CLI regression matrix ran.
- [x] AC #4: discovery-versus-execution RED assertions reached.
- [x] AC #5: selected existing preservation controls pass; actual CLI exercised.


## Implementation Evidence (DELIVERED)

PROOF:

### CI/Test Results
- Commands run:
  - go test -json -count=1 -timeout=120s ./internal/gates -run 'Oracle|Conformance|Coverage|GtCitationBoundaries'
  - go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestOracleCoverageCLI'
  - go test -json -count=1 -timeout=90s ./internal/gates -run '^TestGt'
- Summary: RED expected failures: gates 68 PASS / 22 FAIL / 0 SKIP from 90 leaves; CLI 4 PASS / 6 FAIL / 0 SKIP from 10 leaves; extra Gt controls 7 PASS / 0 FAIL / 0 SKIP (2 boundary leaves repeated). All failures reached intended coverage assertions. Positive native parser checked both rows; changed expected target reached exact transition mismatch assertion. No compile/import/fixture/timeout failure counted.
- Coverage: not measured (RED phase, no production edits); full behavioral matrix/raw JSON/exact leaf inventory in preceding RED Delivery note.
- Artifacts: /tmp/MAC-sh60-red-proof.D3QqXS/gates.jsonl, cli.jsonl, gt-controls.jsonl and matching *-inventory.json.
- Scope: targeted native Go/real local CLI/filesystem, no services; no preflight or unrelated heavy suites.

### Commit
- Branch: story/MAC-sh60
- SHA: b5d3b8c67f487195215860f3d361432f5b2b6b27
- RED-only tests on unchanged production 6cb2d974ea8aea211a5974f453cef2b5802bb11e; GREEN pending.
- 3 files, 285 inserted/8 removed. Frozen hashes and four exact PM-authorized expression edits recorded above.

### Wiring
- New unit tests call actual CheckOracleCoverage.
- New CLI tests use unchanged goldenBin -> go build -> runBinWithEnv -> built machinery check --gate gt on real temp trees.
- Native runtime control executes actual Go fixture and asserts expected row mismatch after oracle-target mutation.

### pvg verify
- pvg verify internal/gates/oraclecov_negative_test.go internal/gates/oraclecov_test.go cmd/machinery/oraclecov_negative_test.go --include-tests --format text
- VERIFY: FAILED (3 files scanned, 1 issues), sole intentional existing TODO comment fixture at oraclecov_test.go:158; no stub/thin files. Exact PM preservation boundary forbids editing this unrelated negative fixture. Documented input marker is not unfinished work.

### AC Verification
| AC # | Requirement | Test Location | Status |
|---|---|---|---|
| 1 | reject comments/docstrings/declarations/disabled source | oraclecov_negative_test.go RejectsNonExecutableEvidence/MixedActiveAndDisabled | RED assertions reached, GREEN pending |
| 2 | preserve active literal/parser discovery, reject ambiguous evidence | LiteralLanguageControls/ActiveParserLanguageControls/ParserRuntimeControl | 20 positive native leaves PASS; semantic mutation reached |
| 3 | full gate matrix, zero/mixed/malformed/positive | new gate/CLI negative files and existing zero-test control | Executed; expected RED only |
| 4 | discovery versus execution label | OutputLabelsDiscoveryOnly/CLILabelsDiscoveryOnly | 2 intended RED failures |
| 5 | boundaries/orphan/missing/clause/real CLI | preserved existing controls and CLIRealTrees | preservation controls PASS; CLI bypass RED |

LEARNINGS:
- Filename/delimiter scanning falsely credits unused or commented source; literal scanning also accepts fake Go test identities.
- Semantic parser control must compare real fixture behavior against expected oracle results and prove the mismatch path.
- Full proof and immutable hashes are recorded; this is RED delivery, never a claim the production ACs already pass.

## nd_contract
status: delivered

### evidence
- RED SHA b5d3b8c67f487195215860f3d361432f5b2b6b27; exact runs, counts, raw output, inventories, hashes, native semantic controls recorded.

### proof
- [x] AC #1: expected unsafe-coverage failures reached.
- [x] AC #2: positive parser/literal controls pass.
- [x] AC #3: full gate and actual CLI matrix executed.
- [x] AC #4: output distinction RED reached.
- [x] AC #5: selected preservation controls pass.


## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## RED Delivery — MAC-sh60
PROOF:
- Phase: RED tests only. Commit b5d3b8c67f487195215860f3d361432f5b2b6b27 on story/MAC-sh60; exact unchanged production base 6cb2d974ea8aea211a5974f453cef2b5802bb11e. GREEN SHA: pending separate agent.
- Commit subject: test(MAC-sh60): tdd-red [test-edit-authorized] require active oracle coverage evidence.
- Diff: 3 files, 285 insertions / 8 deletions (293 changed LOC): new internal/gates/oraclecov_negative_test.go 199 lines; new directly associated cmd/machinery/oraclecov_negative_test.go 82 lines; existing internal/gates/oraclecov_test.go exactly four PM-authorized fixture expression replacements, 4 added/8 deleted. All assertions, names, sibling paths, parseEvidence, other production/tests preserved. Production oraclecov.go diff from base empty; worktree clean.
- Exact commands from committed SHA:
  1. go test -json -count=1 -timeout=120s ./internal/gates -run 'Oracle|Conformance|Coverage|GtCitationBoundaries' > /tmp/MAC-sh60-red-proof.D3QqXS/gates.jsonl
     Exit 1 EXPECTED RED; 90 native leaves: 68 pass, 22 fail, 0 skip.
  2. go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestOracleCoverageCLI' > /tmp/MAC-sh60-red-proof.D3QqXS/cli.jsonl
     Exit 1 EXPECTED RED; 10 native leaves: 4 pass, 6 fail, 0 skip.
  3. go test -json -count=1 -timeout=90s ./internal/gates -run '^TestGt' > /tmp/MAC-sh60-red-proof.D3QqXS/gt-controls.jsonl
     Exit 0; 7 native leaves pass, 0 fail, 0 skip. Includes unreadable-directory, comment, filename and Rust controls. Two filename-boundary leaves repeat command 1; counts are per run, not summed unique.
- Raw JSON and exact leaf inventory files: /tmp/MAC-sh60-red-proof.D3QqXS/{gates,cli,gt-controls}.jsonl and corresponding {gates,cli,gt-controls}-inventory.json. Inventories computed from terminal pass/fail/skip records after excluding parents with child names.
- 20/20 new non-executable unit negatives reached intended behavioral assertion: Gt returned errs=[] and credited either 2 literal IDs or 1 machine by conformance parse. Includes quoted/block Go comments, unused constants, uncalled parser, lowercase Testhelper and wrong-signature TestRows helper, modern/legacy disabled literals, disabled parser, Elixir hash IDs/parser, Python module/test/parser docstrings, JS/Python/Ruby/Elixir unused declarations, unrelated Go split and read without row checks. The invalid-signature SOURCE fixture is never compiled as execution proof; the valid outer native test fails on false Gt coverage.
- Mixed active/disabled unit result: errs=[] counts ids covered by literal:2, machines:1, oracle rows:2, test files scanned:2; intended one uncovered ID assertion fails.
- Five actual CLI bypass leaves reach real Gt with empty stderr then fail exit assertion: exit=0 want=1 for quoted_comment, unused_constants, disabled_go, elixir_comment, mixed_disabled. This is a built worktree CLI process via unchanged goldenBin/runBinWithEnv, real temp design/impl trees and private HOME/config; no fake result or mock.
- Unit and CLI reporting leaves fail because real emitted output lacks explicit discovery/not-execution labeling. CLI includes a fatal literal-ID test source; scanner discovers its literal without running it.
- Passing controls: 11 literal extension fixtures; 7 active parser language fixtures; 3 malformed reference controls; 2 native parser runtime leaves; 4 actual CLI controls (literal, parser, zero tests, malformed reference); all selected legacy tests. Genuine native Go parser reads an actual path, parses 2 rows and compares computed fixture transition to oracle expected target. Raw output contains two 'checked oracle row' logs with successful native exit. Mutating ONLY first expected target B -> A causes native exit status 1 with reached 'oracle transition mismatch: got B want A'; outer corrupt_true control PASSES. No compile/import/read-path/timeout failure counted as RED.
- Other-language fixtures are static discovery examples only: Python, Ruby, Elixir, Rust and JS/TS compare fixture transition results to parsed expected targets; no claim their runtimes/assertions executed. No Docker/Java/Node runtime introduced, required external service lane not applicable. Go local filesystem/process prerequisites ran unconditionally.
- pvg verify internal/gates/oraclecov_negative_test.go internal/gates/oraclecov_test.go cmd/machinery/oraclecov_negative_test.go --include-tests --format text => 'VERIFY: FAILED (3 files scanned, 1 issues)', sole finding intentional existing quoted TODO at oraclecov_test.go:158 inside comment-rejection fixture. No stub/thin-file finding. PM frozen boundary explicitly preserves this fixture; it is test input, not unfinished implementation. The skill's initial --format=text spelling was rejected; help established supported --format text and scan completed.
- git diff --check passed before commit. Test coverage percentage not measured: RED authors no production and intentionally failing suite; behavioral matrix and exact leaf inventory supplied instead. Full preflight and unrelated package suites not run by explicit story constraint; targeted clause/parent/orphan/missing/boundary regressions above passed.
- Frozen SHA256:
  internal/gates/oraclecov_negative_test.go 77940442bc579b2ec64715c79555d89c44dd27e498e49d296899e2d8f90ff042
  internal/gates/oraclecov_test.go dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994
  cmd/machinery/oraclecov_negative_test.go 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08
  unchanged production oraclecov.go 9f4bfabcfe43362b62068e59a59f313cd6d27a3490392f66c5fcf71be70684ad
- No install/dev-link/toolchain/remote/main mutation. Installed Machinery/NIL binary untouched. Parent owns dispatch, independent RED approval, fresh GREEN and merges.

### AC-to-test proof (RED, not production completion)
1. RejectsNonExecutableEvidence 20 intended failures plus MixedActiveAndDisabled: uncovered safety bar frozen.
2. LiteralLanguageControls 11 pass; ActiveParserLanguageControls 7 pass; ParserRuntimeControl 2 pass; malformed/ambiguous negatives; exact four repaired existing positive fixtures pass with old assertions preserved.
3. Full CheckOracleCoverage bypass/zero/mixed/malformed/positive fixtures exercised; actual CLI matrix 9 leaves reached.
4. OutputLabelsDiscoveryOnly and CLILabelsDiscoveryOnly both intended failures; no static execution claim.
5. Existing stable boundaries, orphan/missing diagnostics, clause obligations, parent selection, Rust/MJS controls pass; actual CLI temp-tree execution proven.

LEARNINGS:
- Raw file-level filename/delimiter matching credits entirely uncalled/commented evidence even when row-ID extraction strips Go comments.
- Go name-prefix-only scanning also credits native non-test helper names and invalid signatures; these are discovery failures, not native compile proof.
- Fixture parser success needs row-behavior assertions plus altered expected-result control to avoid replacing an unsafe positive fixture with mere shape checking.
- Existing literal TODO in a negative input triggers pvg quality heuristics and must remain explicitly documented under the exact PM edit boundary.

### Exact native leaf inventory (three runs in command order; repeated boundary controls shown)
pass	TestCheckAcceptanceDoDIDCoverage
pass	TestCheckAcceptanceOracleSetExpandsToExactStableIDInventory
pass	TestCheckAcceptanceRejectsMalformedOracleSet
pass	TestAdjudicationMissingOracleFails
pass	TestAttestationCoverageWarnsRatherThanBlocks
pass	TestCheckBuildPlanNoCommittedOracles
pass	TestGkRejectsFailedCoverageRowUnderPassVerdict
pass	TestGkCoverageGapIsError
pass	TestGkResidualWaivesCoverage
pass	TestClauseCoverageEmptyDeclarationErrors
pass	TestClauseCoverageComplete
pass	TestClauseCoverageMissingSuffixErrors
pass	TestClauseCoverageWholesaleParseDoesNotDischarge
pass	TestClauseCoverageUndeclaredGuardCarriesNoObligation
pass	TestClauseCoverageUnguardedRowsExempt
pass	TestOracleIDsInCommentsDoNotCover
pass	TestOracleVersionOnlySkewIsNotDrift
pass	TestOracleContentDriftStillDrift
pass	TestOracleMissingStampIsFreshAndSilent
pass	TestOracleCurrentStampIsSilent
pass	TestIDCiteRemovedOracleTagStillErrors
pass	TestIDCiteNoOraclesWarns
pass	TestCheckIsolationStaleOracleIsDrift
pass	TestObligationParentDeletedRelationalOracleRemainsRequired/gp
pass	TestObligationParentDeletedRelationalOracleRemainsRequired/gn
pass	TestObligationParentSelectionRetainsRelationalCoverage/Policy.oracle.md
pass	TestObligationParentSelectionRetainsRelationalCoverage/Isolation.oracle.md
fail	TestOracleCoverageRejectsNonExecutableEvidence/quoted_go_comment
fail	TestOracleCoverageRejectsNonExecutableEvidence/go_block_comment
fail	TestOracleCoverageRejectsNonExecutableEvidence/unused_go_constants
fail	TestOracleCoverageRejectsNonExecutableEvidence/uncalled_go_parser
fail	TestOracleCoverageRejectsNonExecutableEvidence/go_lowercase_test_helper
fail	TestOracleCoverageRejectsNonExecutableEvidence/go_invalid_test_signature
fail	TestOracleCoverageRejectsNonExecutableEvidence/disabled_go_literal
fail	TestOracleCoverageRejectsNonExecutableEvidence/disabled_go_parser
fail	TestOracleCoverageRejectsNonExecutableEvidence/legacy_disabled_go_literal
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_hash_ids
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_hash_parser
fail	TestOracleCoverageRejectsNonExecutableEvidence/python_module_docstring
fail	TestOracleCoverageRejectsNonExecutableEvidence/python_test_docstring
fail	TestOracleCoverageRejectsNonExecutableEvidence/python_parser_docstring
fail	TestOracleCoverageRejectsNonExecutableEvidence/js_unused_declarations
fail	TestOracleCoverageRejectsNonExecutableEvidence/python_unused_declarations
fail	TestOracleCoverageRejectsNonExecutableEvidence/ruby_unused_declarations
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_unused_declarations
fail	TestOracleCoverageRejectsNonExecutableEvidence/go_unrelated_split
fail	TestOracleCoverageRejectsNonExecutableEvidence/go_read_without_row_checks
fail	TestOracleCoverageMixedActiveAndDisabled
pass	TestOracleCoverageLiteralLanguageControls/test_rows.py
pass	TestOracleCoverageLiteralLanguageControls/rows_spec.rb
pass	TestOracleCoverageLiteralLanguageControls/rows_test.exs
pass	TestOracleCoverageLiteralLanguageControls/tests/rows.rs
pass	TestOracleCoverageLiteralLanguageControls/rows.test.tsx
pass	TestOracleCoverageLiteralLanguageControls/rows.test.js
pass	TestOracleCoverageLiteralLanguageControls/rows.test.jsx
pass	TestOracleCoverageLiteralLanguageControls/rows.test.mjs
pass	TestOracleCoverageLiteralLanguageControls/rows_test.go
pass	TestOracleCoverageLiteralLanguageControls/rows.test.ts
pass	TestOracleCoverageLiteralLanguageControls/rows.test.cjs
pass	TestOracleCoverageActiveParserLanguageControls/test_rows.py
pass	TestOracleCoverageActiveParserLanguageControls/rows_spec.rb
pass	TestOracleCoverageActiveParserLanguageControls/rows_test.exs
pass	TestOracleCoverageActiveParserLanguageControls/tests/rows.rs
pass	TestOracleCoverageActiveParserLanguageControls/rows_test.go
pass	TestOracleCoverageActiveParserLanguageControls/rows.test.js
pass	TestOracleCoverageActiveParserLanguageControls/rows.test.ts
pass	TestOracleCoverageParserRuntimeControl/corrupt_false
pass	TestOracleCoverageParserRuntimeControl/corrupt_true
pass	TestOracleCoverageMalformedReferencesStayUncovered/Thing.oracle.md.bak
pass	TestOracleCoverageMalformedReferencesStayUncovered/NotThing.oracle.md
pass	TestOracleCoverageMalformedReferencesStayUncovered/purchase-Thing.oracle.md
fail	TestOracleCoverageOutputLabelsDiscoveryOnly
pass	TestCheckOracleCoverageClean
pass	TestCheckOracleCoverageRejectsAndIgnoresOrphanOracle
pass	TestCheckOracleCoverageMissingIDs
pass	TestCheckOracleCoverageIgnoresProductionSources
pass	TestCheckOracleCoverageNoTestFilesFailsLoudly
pass	TestGtCitationBoundaries/hyphenated_sibling_does_not_cover
pass	TestGtCitationBoundaries/a_suffixed_artifact_is_not_the_oracle
pass	TestCheckOracleCoverageRustTestShapes
pass	TestCheckOracleCoverageConformanceParse
pass	TestCheckOracleCoverageConformanceParseIsNotSubstring
pass	TestCheckOracleCoverageMachinesWithoutOracles
pass	TestCheckOracleCoverageMachineMissingItsOracle
pass	TestCheckOracleCoverageNoMachines
pass	TestCheckOracleCoverageFormalOracles/covered_by_file-name_literal
pass	TestCheckOracleCoverageFormalOracles/uncovered
pass	TestCheckOracleCoverageCapsOffenderList
pass	TestCheckOracleCoverageHonorsContractIgnore
pass	TestCheckOracleCoverageScansMjsTestFiles
pass	TestOracleCoverageCLIRealTrees/literal_control
pass	TestOracleCoverageCLIRealTrees/parser_control
fail	TestOracleCoverageCLIRealTrees/quoted_comment
fail	TestOracleCoverageCLIRealTrees/unused_constants
fail	TestOracleCoverageCLIRealTrees/disabled_go
fail	TestOracleCoverageCLIRealTrees/elixir_comment
pass	TestOracleCoverageCLIRealTrees/zero_tests
fail	TestOracleCoverageCLIRealTrees/mixed_disabled
pass	TestOracleCoverageCLIRealTrees/malformed_reference
fail	TestOracleCoverageCLILabelsDiscoveryOnly
pass	TestGtWholesaleNeedsStringLiteralAndParseEvidence/comment_mention_covers_nothing
pass	TestGtWholesaleNeedsStringLiteralAndParseEvidence/quoted_mention_without_parse_evidence_covers_nothing
pass	TestGtWholesaleNeedsStringLiteralAndParseEvidence/parse_evidence_must_live_in_the_citing_file
pass	TestGtCitationBoundaries/hyphenated_sibling_does_not_cover
pass	TestGtCitationBoundaries/a_suffixed_artifact_is_not_the_oracle
pass	TestGtRustProductionTextIsNotTestCorpus
pass	TestGtCorpusSurvivesUnreadableDir


## nd_contract
status: delivered

### evidence
- RED b5d3b8c67f487195215860f3d361432f5b2b6b27, 28 intended failing leaves across main gate/CLI runs, passing controls and exact raw JSON recorded above. GREEN pending.

### proof
- [x] AC #1: RED safety failures reproduced, implementation pending.
- [x] AC #2: native/source positive controls and ambiguous-evidence negative bar established.
- [x] AC #3: real CheckOracleCoverage and real CLI regression fixtures executed.
- [x] AC #4: RED output distinction assertions fail as intended.
- [x] AC #5: selected preservation controls pass and CLI/process path is real.


## nd_contract
status: in_progress

### evidence
- Independent PM exact four-fixture test-edit authorization and reviewed helper semantics recorded; no RED approval or AC completion inferred.
- Preserve claim, hard-tdd, parent and status; require tdd-red plus [test-edit-authorized] commit subject and independent replay.

### proof
- [ ] AC #1: bypass rejection requires executed RED/GREEN proof
- [ ] AC #2: positive active parser/literal discovery and native control require replay
- [ ] AC #3: full regression matrix and actual fixtures require independent review
- [ ] AC #4: output must distinguish static discovery from execution
- [ ] AC #5: boundary/clause/diagnostic preservation and actual CLI proof remain pending


## History
- 2026-09-05T19:35:07Z dep_added: blocks MAC-vx24
- 2026-09-05T19:36:14Z dep_added: blocks MAC-ou97
- 2026-09-06T00:01:08Z status: open -> in_progress
- 2026-09-06T00:01:08Z auto-follows: linked to predecessor MAC-a89e
- 2026-09-06T00:01:08Z claimed by dev-MAC-sh60
- 2026-09-06T00:12:56Z status: in_progress -> in_progress
- 2026-09-06T00:12:56Z auto-follows: linked to predecessor MAC-p8ce
- 2026-09-06T00:28:10Z status: in_progress -> open
- 2026-09-06T00:28:10Z released by ramirosalas
- 2026-09-06T00:29:17Z status: open -> in_progress
- 2026-09-06T00:29:17Z auto-follows: linked to predecessor MAC-olrx
- 2026-09-06T00:29:17Z claimed by dev-MAC-sh60
- 2026-09-06T00:32:37Z status: in_progress -> in_progress
- 2026-09-06T00:42:45Z status: in_progress -> open
- 2026-09-06T00:50:04Z status: open -> in_progress
- 2026-09-06T00:50:04Z claimed by dev-MAC-sh60
- 2026-09-06T19:21:16Z status: in_progress -> closed
- 2026-09-06T19:21:17Z dep_removed: no_longer_blocks MAC-vx24
- 2026-09-06T19:21:17Z dep_removed: no_longer_blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Follows: [[MAC-a89e]], [[MAC-p8ce]], [[MAC-olrx]]
- Led to: [[MAC-wi5z]]

## Comments

### 2026-09-06T00:15:10Z ramirosalas
## Implementation Evidence

PROOF:

### CI/Test Results

Commands run:
- go test -json -count=1 -timeout=120s ./internal/gates -run 'Oracle|Conformance|Coverage|GtCitationBoundaries'
- go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestOracleCoverageCLI'
- go test -json -count=1 -timeout=90s ./internal/gates -run '^TestGt'

Summary: expected RED gates 68 pass / 22 fail / 0 skip (90 leaves); CLI 4 pass / 6 fail / 0 skip (10 leaves); extra Gt 7 pass / 0 fail / 0 skip (2 repeated boundary leaves). Every failure is intended behavior: 20 unsafe-evidence acceptance unit cases, mixed-disabled unit, 5 actual CLI false-green exits and 2 discovery-label failures. Native parser controls pass after verifying 2 rows and reaching expected-target mutation error 'oracle transition mismatch: got B want A'. No compile/import/path/timeout failure used as RED evidence.

Coverage: behavioral matrix enumerated in previous RED Delivery note and exact JSON inventories; code coverage percentage not measured in RED.
Raw terminal artifacts: /tmp/MAC-sh60-red-proof.D3QqXS/gates.jsonl, cli.jsonl, gt-controls.jsonl and matching *-inventory.json. Exact leaf inventory and producing commands also preserved in Notes.

### Commit
Branch: story/MAC-sh60
SHA: b5d3b8c67f487195215860f3d361432f5b2b6b27
RED-only tests; unchanged production base 6cb2d974ea8aea211a5974f453cef2b5802bb11e. GREEN pending fresh agent.
Diff budget: 3 files, 285 inserted/8 removed. New gate tests 199 lines; new real CLI tests 82 lines; existing test four exact independent PM-authorized fixture expressions (4 added/8 removed), assertions and unrelated bytes preserved.

### Frozen test SHA256
- internal/gates/oraclecov_negative_test.go: 77940442bc579b2ec64715c79555d89c44dd27e498e49d296899e2d8f90ff042
- internal/gates/oraclecov_test.go: dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994
- cmd/machinery/oraclecov_negative_test.go: 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08
- unchanged internal/gates/oraclecov.go: 9f4bfabcfe43362b62068e59a59f313cd6d27a3490392f66c5fcf71be70684ad

### pvg verify
Command: pvg verify internal/gates/oraclecov_negative_test.go internal/gates/oraclecov_test.go cmd/machinery/oraclecov_negative_test.go --include-tests --format text
Output: VERIFY: FAILED (3 files scanned, 1 issues). Sole issue is deliberate existing quoted TODO input at oraclecov_test.go:158. No stub/thin findings. PM explicitly preserves this existing comment-rejection fixture; marker is test data, not unfinished implementation. Initial unsupported --format=text was corrected after reading help. git diff --check passed; worktree clean.

### Wiring and limits
Actual CheckOracleCoverage calls; actual built worktree CLI via unchanged goldenBin/runBinWithEnv -> machinery check --gate gt using real temporary design/impl files and isolated HOME/config. Native Go parser fixture executes without service/mocks or skips. Other-language parser/literal fixtures establish discovery only, not runtime execution. No Docker/Java/Node runtime introduced. Full preflight deferred explicitly to final epic, no installed binary/skills/main/remote mutation.

### AC Verification
- [x] AC #1: 20 non-executable source negatives + mixed disabled test reach intended unsafe-coverage assertions; GREEN pending.
- [x] AC #2: 11 literal extension controls, 7 active parser language controls and 2 native semantic/mutation controls PASS; ambiguous evidence rejected by RED expectations.
- [x] AC #3: every assessment bypass, real full gate fixture, zero/mixed/malformed/positive and actual CLI matrix executed.
- [x] AC #4: unit and real CLI discovery-versus-execution label assertions fail as intended.
- [x] AC #5: stable-ID/file boundaries, orphan/missing-oracle, parent/clause, Rust/MJS preservation tests pass; real CLI path exercises false-green bypasses with expected RED.

LEARNINGS:
- Filename/delimiter matching credits unused/commented source; Go Test-prefix matching additionally credits native non-test helpers.
- Semantic parser control needs real result-vs-expected comparison and an altered expected-result assertion control.
- Static source discovery remains distinct from test execution; no production AC completion claimed by this RED delivery.

## nd_contract
status: delivered

### evidence
- RED b5d3b8c67f487195215860f3d361432f5b2b6b27; gate90=68 pass/22 intended fail, CLI10=4 pass/6 intended fail, extraGt7 pass; raw JSON/inventory/hashes above. Implementation pending GREEN.

### proof
- [x] AC #1: executable-evidence RED bar frozen and false-positive assertions reached.
- [x] AC #2: actual positive discovery fixtures and native semantic control pass.
- [x] AC #3: full CheckOracleCoverage and real CLI regression matrix ran.
- [x] AC #4: discovery-versus-execution RED assertions reached.
- [x] AC #5: selected existing preservation controls pass; actual CLI exercised.


### 2026-09-06T00:28:10Z ramirosalas
## PM Decision
REJECTED [2026-09-05]: RED coverage gap, not a rejection of expected RED failures. Reviewed candidate b5d3b8c67f487195215860f3d361432f5b2b6b27 on unchanged production 6cb2d974ea8aea211a5974f453cef2b5802bb11e.

EXPECTED: AC1 says "Comments, docstrings-only citations ... cannot establish oracle-row or wholesale table coverage." AC2 preserves currently supported languages. RED must constrain that outcome before GREEN freezes the suite.
DELIVERED: The 20-case negative matrix tests Python docstrings and Elixir hash comments, but contains no Elixir @doc/@moduledoc or Ruby =begin/=end block-comment case. Both languages have positive literal/parser fixtures and remain supported. Direct source inspection of executableTestText/stripTestComments/fileNameCited plus six independent real CLI probes confirms these exact omissions each currently yield Gt ok, zero blocking findings and exit 0 with empty stderr: elixir_moduledoc_ids, elixir_doc_ids, ruby_block_comment_ids credit 2 literal IDs; elixir_moduledoc_parser, elixir_doc_parser, ruby_block_comment_parser credit 1 whole conformance parse.
GAP: A GREEN implementation could satisfy every frozen RED assertion by handling Python triple quotes and Elixir # comments while still manufacturing coverage from Elixir documentation and Ruby block comments, violating AC1. This is a reproduced supported-language hole, not a request for a general parser architecture or external language runtimes.
FIX: Add the six exact cases below to TestOracleCoverageRejectsNonExecutableEvidence. Reuse its real CheckOracleCoverage fixture and existing assertions (nonempty coverage diagnosis, zero literal IDs, zero wholesale machines). Replay on unchanged production: each must fail because unsafe coverage was credited; preserve all existing positive/runtime/preservation controls and actual CLI proof. Deliver an amended tdd-red commit with updated SHA/hashes/counts for fresh independent review. No production implementation or RED approval yet.

TEST-EDIT AUTHORIZED: internal/gates/oraclecov_negative_test.go -- add ONLY six entries to the existing TestOracleCoverageRejectsNonExecutableEvidence cases slice, using the exact fixture forms below. Preserve the current 20 entries, shared assertion body, helper semantics, other tests and all unrelated bytes. This supplements the prior four-expression authorization; it does not reopen internal/gates/oraclecov_test.go or cmd/machinery/oraclecov_negative_test.go. Repair commit subject must carry both tdd-red and [test-edit-authorized].

1. elixir_moduledoc_ids, rows_test.exs: `defmodule RowsTest do\n  @moduledoc """\n  THIN-aaa111 THIN-bbb222\n  """\nend\n`
2. elixir_doc_ids, rows_test.exs: `defmodule RowsTest do\n  @doc """\n  THIN-aaa111 THIN-bbb222\n  """\n  def helper, do: :ok\nend\n`
3. elixir_moduledoc_parser, rows_test.exs: `defmodule RowsTest do\n  @moduledoc """\n  parse "Thing.oracle.md" using "|"\n  """\nend\n`
4. elixir_doc_parser, rows_test.exs: `defmodule RowsTest do\n  @doc """\n  parse "Thing.oracle.md" using "|"\n  """\n  def helper, do: :ok\nend\n`
5. ruby_block_comment_ids, rows_spec.rb: `=begin\nTHIN-aaa111 THIN-bbb222\n=end\n`
6. ruby_block_comment_parser, rows_spec.rb: `=begin\nparse "Thing.oracle.md" using "|"\n=end\n`

The displayed \n sequences designate actual source line breaks; ordinary Go string escaping is allowed when inserting these fixture bytes. Exact standalone fixture files are preserved under /tmp/MAC-sh60-pm-red.2yPht9/probes/<case>/.

## Independent PM evidence
- Full shared canonical story, five AC, four-fixture authorization, complete delivery proof/LEARNINGS read via pvg nd show MAC-sh60 --json. Candidate inspected in own detached /tmp/MAC-sh60-pm-red.2yPht9/review; author worktree retained untouched. Actual branch is epic/MAC-ui8a; initial concatenated branch spelling in dispatcher prompt was resolved read-only.
- Graph list_projects/index_status and check_index_coverage: generation 2026-09-05T23:58:53Z, production/original-test/CLI-harness metadata_match with no recorded issues. New negative files absent from main index; full exact candidate source read is authoritative fallback. No completeness claim from the graph.
- Static source scan and pvg verify internal/gates/oraclecov_negative_test.go internal/gates/oraclecov_test.go cmd/machinery/oraclecov_negative_test.go --include-tests --format text: sole scanner finding is existing oraclecov_test.go:158 quoted TODO inside the negative comment fixture; verified source, not unfinished work. No suppression or edit. No stubs, skip gates or missing type signatures. RED changes only tests; implementation/doc freshness awaits GREEN.
- Diff from 6cb2d974 to candidate: 3 files, 285 insertions/8 deletions. Existing oraclecov_test.go diff is exactly the four authorized source expressions; original assertions, names, counts, paths, parseEvidence preserved. Production unchanged. git diff --check passed.
- Frozen SHA256 independently matched: gate negatives 77940442bc579b2ec64715c79555d89c44dd27e498e49d296899e2d8f90ff042; original gate tests dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994; CLI negatives 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08; oraclecov.go 9f4bfabcfe43362b62068e59a59f313cd6d27a3490392f66c5fcf71be70684ad.
- Independent command 1, from detached candidate: go test -json -count=1 -timeout=120s ./internal/gates -run 'Oracle|Conformance|Coverage|GtCitationBoundaries' > /tmp/MAC-sh60-pm-red.2yPht9/gates.jsonl. Session 3851 completed exit 1, 90 leaves = 68 pass/22 intended fail/0 skip. Twenty unsafe acceptance failures plus mixed-disabled and discovery-label failures, with actual returned counts/errors inspected.
- Independent command 2: go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestOracleCoverageCLI' > /tmp/MAC-sh60-pm-red.2yPht9/cli.jsonl. Session 28279 completed exit 1, 10 leaves = 4 pass/6 intended fail/0 skip. Five real CLI false-green exits plus discovery-label failure. All reach Gt with empty stderr.
- Independent command 3: go test -json -count=1 -timeout=90s ./internal/gates -run '^TestGt' > /tmp/MAC-sh60-pm-red.2yPht9/gt-controls.jsonl. Exit 0, 7 pass/0 fail/0 skip, including 2 repeated boundary leaves. Exact leaf inventories stored alongside as gates-inventory.json, cli-inventory.json, gt-controls-inventory.json; no sums claimed as unique tests.
- Native Go fixture control passes after two checked-row logs. Altering only first oracle target B -> A reaches exact assertion "oracle transition mismatch: got B want A"; corrupt_true outer test passes. No compile/import/read/timeout error counts as RED. Other-language fixtures establish static discovery only. Go lowercase/wrong-signature source negatives test discovery, not native execution.
- Read unchanged goldenBin/runBinWithEnv source: actual local go build and subprocess against real temporary filesystem trees, private HOME/config, no mocks or injected coverage. CLI disabled_go has an empty detail expectation, but checks actual Gt identity, clean stderr and blocking exit; paired disabled unit cases constrain zero credit/coverage diagnosis. It is not an independent rejection ground and no existing CLI edit is authorized.
- Probe binary command: go build -o /tmp/MAC-sh60-pm-red.2yPht9/machinery ./cmd/machinery. For each exact six case names above, ran from detached candidate: env HOME=/tmp/MAC-sh60-pm-red.2yPht9/private-home MACHINERY_CONFIG_DIR=/tmp/MAC-sh60-pm-red.2yPht9/private-config /tmp/MAC-sh60-pm-red.2yPht9/machinery check /tmp/MAC-sh60-pm-red.2yPht9/probes/design --impl /tmp/MAC-sh60-pm-red.2yPht9/probes/<case> --gate gt > /tmp/MAC-sh60-pm-red.2yPht9/<case>.out 2> /tmp/MAC-sh60-pm-red.2yPht9/<case>.err. All six exit 0; all stderr files zero bytes. These diagnostic fixture additions are outside all checkouts and are not edits to delivered code/tests.
- Coverage percentage not measured in intentionally failing RED; exact behavioral inventory supplied. No full preflight, external runtime suite, install/dev-link/toolchain/live binary/plugin/skill change, remote mutation, main change or merge.

LEARNINGS:
- Supported-language comment/docstring boundaries need explicit regression examples: recognizing Python triple quotes and Elixir hash comments alone does not constrain Elixir documentation attributes or Ruby embedded documents.
- The parser control and immutable four-fixture repair are adequate; preserve them during this narrow RED rework.

## nd_contract
status: rejected

### evidence
- Independent candidate replay and six real CLI probes recorded above. Canonical rejection is for incomplete RED coverage only; production completion remains pending.

### proof
- [ ] AC #1: add six reproduced Elixir documentation/Ruby block-comment negatives before RED approval.
- [x] AC #2: existing literal/parser positives and native assertion-sensitivity control are valid; retain unchanged.
- [ ] AC #3: extend full CheckOracleCoverage negative matrix with the six exact newly confirmed bypasses; existing assessment/zero/mixed/malformed matrix ran as claimed.
- [x] AC #4: current unit and actual CLI label assertions are valid intended RED; no static execution claim.
- [x] AC #5: boundaries/orphan/missing/clause controls pass; real CLI integration is proven.

### 2026-09-06T00:28:17Z ramirosalas
## PM Decision
REJECTED [2026-09-05]: RED coverage gap, not a rejection of expected RED failures. Reviewed candidate b5d3b8c67f487195215860f3d361432f5b2b6b27 on unchanged production 6cb2d974ea8aea211a5974f453cef2b5802bb11e.

EXPECTED: AC1 says "Comments, docstrings-only citations ... cannot establish oracle-row or wholesale table coverage." AC2 preserves currently supported languages. RED must constrain that outcome before GREEN freezes the suite.
DELIVERED: The 20-case negative matrix tests Python docstrings and Elixir hash comments, but contains no Elixir @doc/@moduledoc or Ruby =begin/=end block-comment case. Both languages have positive literal/parser fixtures and remain supported. Direct source inspection of executableTestText/stripTestComments/fileNameCited plus six independent real CLI probes confirms these exact omissions each currently yield Gt ok, zero blocking findings and exit 0 with empty stderr: elixir_moduledoc_ids, elixir_doc_ids, ruby_block_comment_ids credit 2 literal IDs; elixir_moduledoc_parser, elixir_doc_parser, ruby_block_comment_parser credit 1 whole conformance parse.
GAP: A GREEN implementation could satisfy every frozen RED assertion by handling Python triple quotes and Elixir # comments while still manufacturing coverage from Elixir documentation and Ruby block comments, violating AC1. This is a reproduced supported-language hole, not a request for a general parser architecture or external language runtimes.
FIX: Add the six exact cases below to TestOracleCoverageRejectsNonExecutableEvidence. Reuse its real CheckOracleCoverage fixture and existing assertions (nonempty coverage diagnosis, zero literal IDs, zero wholesale machines). Replay on unchanged production: each must fail because unsafe coverage was credited; preserve all existing positive/runtime/preservation controls and actual CLI proof. Deliver an amended tdd-red commit with updated SHA/hashes/counts for fresh independent review. No production implementation or RED approval yet.

TEST-EDIT AUTHORIZED: internal/gates/oraclecov_negative_test.go -- add ONLY six entries to the existing TestOracleCoverageRejectsNonExecutableEvidence cases slice, using the exact fixture forms below. Preserve the current 20 entries, shared assertion body, helper semantics, other tests and all unrelated bytes. This supplements the prior four-expression authorization; it does not reopen internal/gates/oraclecov_test.go or cmd/machinery/oraclecov_negative_test.go. Repair commit subject must carry both tdd-red and [test-edit-authorized].

1. elixir_moduledoc_ids, rows_test.exs: `defmodule RowsTest do\n  @moduledoc """\n  THIN-aaa111 THIN-bbb222\n  """\nend\n`
2. elixir_doc_ids, rows_test.exs: `defmodule RowsTest do\n  @doc """\n  THIN-aaa111 THIN-bbb222\n  """\n  def helper, do: :ok\nend\n`
3. elixir_moduledoc_parser, rows_test.exs: `defmodule RowsTest do\n  @moduledoc """\n  parse "Thing.oracle.md" using "|"\n  """\nend\n`
4. elixir_doc_parser, rows_test.exs: `defmodule RowsTest do\n  @doc """\n  parse "Thing.oracle.md" using "|"\n  """\n  def helper, do: :ok\nend\n`
5. ruby_block_comment_ids, rows_spec.rb: `=begin\nTHIN-aaa111 THIN-bbb222\n=end\n`
6. ruby_block_comment_parser, rows_spec.rb: `=begin\nparse "Thing.oracle.md" using "|"\n=end\n`

The displayed \n sequences designate actual source line breaks; ordinary Go string escaping is allowed when inserting these fixture bytes. Exact standalone fixture files are preserved under /tmp/MAC-sh60-pm-red.2yPht9/probes/<case>/.

## Independent PM evidence
- Full shared canonical story, five AC, four-fixture authorization, complete delivery proof/LEARNINGS read via pvg nd show MAC-sh60 --json. Candidate inspected in own detached /tmp/MAC-sh60-pm-red.2yPht9/review; author worktree retained untouched. Actual branch is epic/MAC-ui8a; initial concatenated branch spelling in dispatcher prompt was resolved read-only.
- Graph list_projects/index_status and check_index_coverage: generation 2026-09-05T23:58:53Z, production/original-test/CLI-harness metadata_match with no recorded issues. New negative files absent from main index; full exact candidate source read is authoritative fallback. No completeness claim from the graph.
- Static source scan and pvg verify internal/gates/oraclecov_negative_test.go internal/gates/oraclecov_test.go cmd/machinery/oraclecov_negative_test.go --include-tests --format text: sole scanner finding is existing oraclecov_test.go:158 quoted TODO inside the negative comment fixture; verified source, not unfinished work. No suppression or edit. No stubs, skip gates or missing type signatures. RED changes only tests; implementation/doc freshness awaits GREEN.
- Diff from 6cb2d974 to candidate: 3 files, 285 insertions/8 deletions. Existing oraclecov_test.go diff is exactly the four authorized source expressions; original assertions, names, counts, paths, parseEvidence preserved. Production unchanged. git diff --check passed.
- Frozen SHA256 independently matched: gate negatives 77940442bc579b2ec64715c79555d89c44dd27e498e49d296899e2d8f90ff042; original gate tests dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994; CLI negatives 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08; oraclecov.go 9f4bfabcfe43362b62068e59a59f313cd6d27a3490392f66c5fcf71be70684ad.
- Independent command 1, from detached candidate: go test -json -count=1 -timeout=120s ./internal/gates -run 'Oracle|Conformance|Coverage|GtCitationBoundaries' > /tmp/MAC-sh60-pm-red.2yPht9/gates.jsonl. Session 3851 completed exit 1, 90 leaves = 68 pass/22 intended fail/0 skip. Twenty unsafe acceptance failures plus mixed-disabled and discovery-label failures, with actual returned counts/errors inspected.
- Independent command 2: go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestOracleCoverageCLI' > /tmp/MAC-sh60-pm-red.2yPht9/cli.jsonl. Session 28279 completed exit 1, 10 leaves = 4 pass/6 intended fail/0 skip. Five real CLI false-green exits plus discovery-label failure. All reach Gt with empty stderr.
- Independent command 3: go test -json -count=1 -timeout=90s ./internal/gates -run '^TestGt' > /tmp/MAC-sh60-pm-red.2yPht9/gt-controls.jsonl. Exit 0, 7 pass/0 fail/0 skip, including 2 repeated boundary leaves. Exact leaf inventories stored alongside as gates-inventory.json, cli-inventory.json, gt-controls-inventory.json; no sums claimed as unique tests.
- Native Go fixture control passes after two checked-row logs. Altering only first oracle target B -> A reaches exact assertion "oracle transition mismatch: got B want A"; corrupt_true outer test passes. No compile/import/read/timeout error counts as RED. Other-language fixtures establish static discovery only. Go lowercase/wrong-signature source negatives test discovery, not native execution.
- Read unchanged goldenBin/runBinWithEnv source: actual local go build and subprocess against real temporary filesystem trees, private HOME/config, no mocks or injected coverage. CLI disabled_go has an empty detail expectation, but checks actual Gt identity, clean stderr and blocking exit; paired disabled unit cases constrain zero credit/coverage diagnosis. It is not an independent rejection ground and no existing CLI edit is authorized.
- Probe binary command: go build -o /tmp/MAC-sh60-pm-red.2yPht9/machinery ./cmd/machinery. For each exact six case names above, ran from detached candidate: env HOME=/tmp/MAC-sh60-pm-red.2yPht9/private-home MACHINERY_CONFIG_DIR=/tmp/MAC-sh60-pm-red.2yPht9/private-config /tmp/MAC-sh60-pm-red.2yPht9/machinery check /tmp/MAC-sh60-pm-red.2yPht9/probes/design --impl /tmp/MAC-sh60-pm-red.2yPht9/probes/<case> --gate gt > /tmp/MAC-sh60-pm-red.2yPht9/<case>.out 2> /tmp/MAC-sh60-pm-red.2yPht9/<case>.err. All six exit 0; all stderr files zero bytes. These diagnostic fixture additions are outside all checkouts and are not edits to delivered code/tests.
- Coverage percentage not measured in intentionally failing RED; exact behavioral inventory supplied. No full preflight, external runtime suite, install/dev-link/toolchain/live binary/plugin/skill change, remote mutation, main change or merge.

LEARNINGS:
- Supported-language comment/docstring boundaries need explicit regression examples: recognizing Python triple quotes and Elixir hash comments alone does not constrain Elixir documentation attributes or Ruby embedded documents.
- The parser control and immutable four-fixture repair are adequate; preserve them during this narrow RED rework.

## nd_contract
status: rejected

### evidence
- Independent candidate replay and six real CLI probes recorded above. Canonical rejection is for incomplete RED coverage only; production completion remains pending.

### proof
- [ ] AC #1: add six reproduced Elixir documentation/Ruby block-comment negatives before RED approval.
- [x] AC #2: existing literal/parser positives and native assertion-sensitivity control are valid; retain unchanged.
- [ ] AC #3: extend full CheckOracleCoverage negative matrix with the six exact newly confirmed bypasses; existing assessment/zero/mixed/malformed matrix ran as claimed.
- [x] AC #4: current unit and actual CLI label assertions are valid intended RED; no static execution claim.
- [x] AC #5: boundaries/orphan/missing/clause controls pass; real CLI integration is proven.

### 2026-09-06T00:32:37Z ramirosalas
## Implementation Evidence

PROOF:
RED rework delivery responding to the complete PM rejection and exact add-only six-case TEST-EDIT AUTHORIZED note. Prior original RED evidence remains historical; these results supersede its candidate/counts. No production changes.

### CI/Test Results

Commands run:
- go test -json -count=1 -timeout=120s ./internal/gates -run 'Oracle|Conformance|Coverage|GtCitationBoundaries' > /tmp/MAC-sh60-red-rework.Ftwvzy/gates.jsonl
- go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestOracleCoverageCLI' > /tmp/MAC-sh60-red-rework.Ftwvzy/cli.jsonl
- go test -json -count=1 -timeout=90s ./internal/gates -run '^TestGt' > /tmp/MAC-sh60-red-rework.Ftwvzy/gt-controls.jsonl

Summary: gates exit 1 EXPECTED RED: 96 native leaves, 68 pass/28 intended fail/0 skip; CLI exit 1 EXPECTED RED: 10 leaves, 4 pass/6 intended fail/0 skip; extra Gt exit 0: 7 pass/0 fail/0 skip. Two boundary leaves repeat between gates and extra Gt, so counts are per run, not summed unique. New gate failures +6 only, existing passing counts unchanged. All terminal handles completed; no background processes retained.

Coverage: code coverage percentage not measured in intentionally failing RED with no production implementation. Behavioral matrix/exact leaf inventory below. Matching *-inventory.json artifacts in /tmp/MAC-sh60-red-rework.Ftwvzy preserve every native leaf and result, derived from terminal pass/fail/skip records after excluding parents.

### Exact six-case authorized delta and reached failures
In TestOracleCoverageRejectsNonExecutableEvidence, added ONLY:
1. elixir_moduledoc_ids: rows_test.exs containing @moduledoc triple-quoted THIN-aaa111 THIN-bbb222.
2. elixir_doc_ids: rows_test.exs containing @doc triple-quoted IDs and def helper, do: :ok.
3. elixir_moduledoc_parser: rows_test.exs containing @moduledoc triple-quoted parse "Thing.oracle.md" using "|".
4. elixir_doc_parser: rows_test.exs containing @doc triple-quoted parser phrase and def helper, do: :ok.
5. ruby_block_comment_ids: rows_spec.rb with =begin / IDs / =end on separate lines.
6. ruby_block_comment_parser: rows_spec.rb with =begin / parse "Thing.oracle.md" using "|" / =end on separate lines.
Exact fixture bytes read from /tmp/MAC-sh60-pm-red.2yPht9/probes/<case>/ and inserted with ordinary Go string escaping, retaining actual source line breaks and spaces.

Each new leaf reaches oraclecov_negative_test.go:88 assertion "non-executable evidence established coverage". Three *_ids results: errs=[] counts=map[ids covered by literal:2 machines:1 oracle rows:2 test files scanned:1]. Three *_parser results: errs=[] counts=map[machines:1 machines covered by conformance parse:1 oracle rows:2 test files scanned:1]. Thus every added test is genuine assertion-failure RED on unchanged production, not compilation/import/read-path/fixture/runtime infrastructure failure.

Native semantic controls reran as part of the full gate selection. corrupt_false: inner native TestOracle exit nil, two "checked oracle row" logs, PASS. corrupt_true: changing only expected first target B -> A produces inner native exit status 1 and reached "oracle transition mismatch: got B want A"; the outer control PASS confirms assertion sensitivity. Other-language source fixtures are discovery only, never a claim their native tests executed.

### Commit
Branch: story/MAC-sh60
SHA: fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3
Subject: test(MAC-sh60): tdd-red [test-edit-authorized] cover Elixir docs and Ruby block comments
Original RED SHA retained: b5d3b8c67f487195215860f3d361432f5b2b6b27
Production base remains 6cb2d974ea8aea211a5974f453cef2b5802bb11e; GREEN pending separate agent.
Exact rework diff from original RED: one file, six inserted lines / zero removed. Existing 20 entries, shared assertion body, all helper semantics and every other test byte retained. Existing oraclecov_test.go, CLI negative file and oraclecov.go diff from original RED empty. git diff --check passed and worktree clean.
Cumulative story cost: three files, 291 insertions/8 deletions = 299 changed LOC. Gate negative file 205 lines; CLI negative file 82 lines; original gate tests retain exactly four previously authorized fixture replacements (4 inserted/8 deleted).

### Frozen SHA256
- internal/gates/oraclecov_negative_test.go: 42f1237819cfecc4a265fd5dfa4a90050d8f0f77f9aa0baf27b42995d8834e45 (updated by authorized six-line addition)
- internal/gates/oraclecov_test.go: dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994 (unchanged)
- cmd/machinery/oraclecov_negative_test.go: 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08 (unchanged)
- internal/gates/oraclecov.go: 9f4bfabcfe43362b62068e59a59f313cd6d27a3490392f66c5fcf71be70684ad (unchanged production)

### pvg verify
Command: pvg verify internal/gates/oraclecov_negative_test.go internal/gates/oraclecov_test.go cmd/machinery/oraclecov_negative_test.go --include-tests --format text
Output: VERIFY: FAILED (3 files scanned, 1 issues), sole existing oraclecov_test.go:158 quoted TODO inside negative comment fixture. No stub/thin files. PM independently reviewed and explicitly preserves this test-data marker; no suppression/edit applied. This is not unfinished implementation.

### Wiring and limits
Real CheckOracleCoverage over temporary filesystem trees; real worktree-built CLI via preserved goldenBin/runBinWithEnv; unchanged private HOME/config isolation. Local Go/native filesystem/process only, no mocked/injected result, no external runtime dependency or skip gate. No full preflight (final epic only), main/epic/remote/installed binary/plugin/skill mutation. Parent coordinates PM, GREEN and merges.

### AC Verification
- [x] AC #1: full 26-case non-executable matrix now includes exactly the six PM-reproduced Elixir docs/Ruby block comments, all failing on unsafe coverage; mixed-disabled behavior also constrained. GREEN pending.
- [x] AC #2: all 11 literal extension, 7 active parser language and 2 native runtime semantic controls still pass unchanged. Unsupported/ambiguous source negatives remain.
- [x] AC #3: full actual CheckOracleCoverage matrix expanded +6; existing zero/mixed/malformed/positive and actual CLI selections rerun without weakening.
- [x] AC #4: existing unit and actual CLI discovery-versus-execution assertions still reach intended RED.
- [x] AC #5: existing stable-ID/file boundaries, orphan/missing oracle, parent/clause, Rust/MJS and real CLI proof retained and rerun.

LEARNINGS:
- The original RED matrix covered Python docstrings and Elixir hash comments but missed Elixir documentation attributes and Ruby embedded documents. PM reproduced the concrete gap and these six cases now constrain it.
- Preserve native semantic parser success and changed expected-target controls during language lexical coverage repairs.
- Six additive cases close this reviewed RED gap without changing production or previously frozen assertions.

### Exact native leaf inventory (gates, CLI, extra Gt command order)
pass	TestCheckAcceptanceDoDIDCoverage
pass	TestCheckAcceptanceOracleSetExpandsToExactStableIDInventory
pass	TestCheckAcceptanceRejectsMalformedOracleSet
pass	TestAdjudicationMissingOracleFails
pass	TestAttestationCoverageWarnsRatherThanBlocks
pass	TestCheckBuildPlanNoCommittedOracles
pass	TestGkRejectsFailedCoverageRowUnderPassVerdict
pass	TestGkCoverageGapIsError
pass	TestGkResidualWaivesCoverage
pass	TestClauseCoverageEmptyDeclarationErrors
pass	TestClauseCoverageComplete
pass	TestClauseCoverageMissingSuffixErrors
pass	TestClauseCoverageWholesaleParseDoesNotDischarge
pass	TestClauseCoverageUndeclaredGuardCarriesNoObligation
pass	TestClauseCoverageUnguardedRowsExempt
pass	TestOracleIDsInCommentsDoNotCover
pass	TestOracleVersionOnlySkewIsNotDrift
pass	TestOracleContentDriftStillDrift
pass	TestOracleMissingStampIsFreshAndSilent
pass	TestOracleCurrentStampIsSilent
pass	TestIDCiteRemovedOracleTagStillErrors
pass	TestIDCiteNoOraclesWarns
pass	TestCheckIsolationStaleOracleIsDrift
pass	TestObligationParentDeletedRelationalOracleRemainsRequired/gp
pass	TestObligationParentDeletedRelationalOracleRemainsRequired/gn
pass	TestObligationParentSelectionRetainsRelationalCoverage/Policy.oracle.md
pass	TestObligationParentSelectionRetainsRelationalCoverage/Isolation.oracle.md
fail	TestOracleCoverageRejectsNonExecutableEvidence/quoted_go_comment
fail	TestOracleCoverageRejectsNonExecutableEvidence/go_block_comment
fail	TestOracleCoverageRejectsNonExecutableEvidence/unused_go_constants
fail	TestOracleCoverageRejectsNonExecutableEvidence/uncalled_go_parser
fail	TestOracleCoverageRejectsNonExecutableEvidence/go_lowercase_test_helper
fail	TestOracleCoverageRejectsNonExecutableEvidence/go_invalid_test_signature
fail	TestOracleCoverageRejectsNonExecutableEvidence/disabled_go_literal
fail	TestOracleCoverageRejectsNonExecutableEvidence/disabled_go_parser
fail	TestOracleCoverageRejectsNonExecutableEvidence/legacy_disabled_go_literal
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_hash_ids
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_hash_parser
fail	TestOracleCoverageRejectsNonExecutableEvidence/python_module_docstring
fail	TestOracleCoverageRejectsNonExecutableEvidence/python_test_docstring
fail	TestOracleCoverageRejectsNonExecutableEvidence/python_parser_docstring
fail	TestOracleCoverageRejectsNonExecutableEvidence/js_unused_declarations
fail	TestOracleCoverageRejectsNonExecutableEvidence/python_unused_declarations
fail	TestOracleCoverageRejectsNonExecutableEvidence/ruby_unused_declarations
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_unused_declarations
fail	TestOracleCoverageRejectsNonExecutableEvidence/go_unrelated_split
fail	TestOracleCoverageRejectsNonExecutableEvidence/go_read_without_row_checks
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_moduledoc_ids
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_doc_ids
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_moduledoc_parser
fail	TestOracleCoverageRejectsNonExecutableEvidence/elixir_doc_parser
fail	TestOracleCoverageRejectsNonExecutableEvidence/ruby_block_comment_ids
fail	TestOracleCoverageRejectsNonExecutableEvidence/ruby_block_comment_parser
fail	TestOracleCoverageMixedActiveAndDisabled
pass	TestOracleCoverageLiteralLanguageControls/rows.test.jsx
pass	TestOracleCoverageLiteralLanguageControls/rows.test.ts
pass	TestOracleCoverageLiteralLanguageControls/rows.test.mjs
pass	TestOracleCoverageLiteralLanguageControls/rows.test.cjs
pass	TestOracleCoverageLiteralLanguageControls/test_rows.py
pass	TestOracleCoverageLiteralLanguageControls/rows_spec.rb
pass	TestOracleCoverageLiteralLanguageControls/rows_test.exs
pass	TestOracleCoverageLiteralLanguageControls/tests/rows.rs
pass	TestOracleCoverageLiteralLanguageControls/rows_test.go
pass	TestOracleCoverageLiteralLanguageControls/rows.test.tsx
pass	TestOracleCoverageLiteralLanguageControls/rows.test.js
pass	TestOracleCoverageActiveParserLanguageControls/tests/rows.rs
pass	TestOracleCoverageActiveParserLanguageControls/rows_test.go
pass	TestOracleCoverageActiveParserLanguageControls/rows.test.js
pass	TestOracleCoverageActiveParserLanguageControls/rows.test.ts
pass	TestOracleCoverageActiveParserLanguageControls/test_rows.py
pass	TestOracleCoverageActiveParserLanguageControls/rows_spec.rb
pass	TestOracleCoverageActiveParserLanguageControls/rows_test.exs
pass	TestOracleCoverageParserRuntimeControl/corrupt_false
pass	TestOracleCoverageParserRuntimeControl/corrupt_true
pass	TestOracleCoverageMalformedReferencesStayUncovered/Thing.oracle.md.bak
pass	TestOracleCoverageMalformedReferencesStayUncovered/NotThing.oracle.md
pass	TestOracleCoverageMalformedReferencesStayUncovered/purchase-Thing.oracle.md
fail	TestOracleCoverageOutputLabelsDiscoveryOnly
pass	TestCheckOracleCoverageClean
pass	TestCheckOracleCoverageRejectsAndIgnoresOrphanOracle
pass	TestCheckOracleCoverageMissingIDs
pass	TestCheckOracleCoverageIgnoresProductionSources
pass	TestCheckOracleCoverageNoTestFilesFailsLoudly
pass	TestGtCitationBoundaries/hyphenated_sibling_does_not_cover
pass	TestGtCitationBoundaries/a_suffixed_artifact_is_not_the_oracle
pass	TestCheckOracleCoverageRustTestShapes
pass	TestCheckOracleCoverageConformanceParse
pass	TestCheckOracleCoverageConformanceParseIsNotSubstring
pass	TestCheckOracleCoverageMachinesWithoutOracles
pass	TestCheckOracleCoverageMachineMissingItsOracle
pass	TestCheckOracleCoverageNoMachines
pass	TestCheckOracleCoverageFormalOracles/covered_by_file-name_literal
pass	TestCheckOracleCoverageFormalOracles/uncovered
pass	TestCheckOracleCoverageCapsOffenderList
pass	TestCheckOracleCoverageHonorsContractIgnore
pass	TestCheckOracleCoverageScansMjsTestFiles
pass	TestOracleCoverageCLIRealTrees/literal_control
pass	TestOracleCoverageCLIRealTrees/parser_control
fail	TestOracleCoverageCLIRealTrees/quoted_comment
fail	TestOracleCoverageCLIRealTrees/unused_constants
fail	TestOracleCoverageCLIRealTrees/disabled_go
fail	TestOracleCoverageCLIRealTrees/elixir_comment
pass	TestOracleCoverageCLIRealTrees/zero_tests
fail	TestOracleCoverageCLIRealTrees/mixed_disabled
pass	TestOracleCoverageCLIRealTrees/malformed_reference
fail	TestOracleCoverageCLILabelsDiscoveryOnly
pass	TestGtWholesaleNeedsStringLiteralAndParseEvidence/comment_mention_covers_nothing
pass	TestGtWholesaleNeedsStringLiteralAndParseEvidence/quoted_mention_without_parse_evidence_covers_nothing
pass	TestGtWholesaleNeedsStringLiteralAndParseEvidence/parse_evidence_must_live_in_the_citing_file
pass	TestGtCitationBoundaries/hyphenated_sibling_does_not_cover
pass	TestGtCitationBoundaries/a_suffixed_artifact_is_not_the_oracle
pass	TestGtRustProductionTextIsNotTestCorpus
pass	TestGtCorpusSurvivesUnreadableDir


## nd_contract
status: delivered

### evidence
- RED rework fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3; raw JSON /tmp/MAC-sh60-red-rework.Ftwvzy; gates96=68 pass/28 intended fail, CLI10=4 pass/6 intended fail, Gt7pass, zero skips. Exact six additive entries and hashes recorded above.

### proof
- [x] AC #1: 26 negatives include six authorized supported-language documentation/comment failures.
- [x] AC #2: unchanged literal/parser and native semantic controls pass.
- [x] AC #3: expanded full-gate regression and real CLI matrix executed.
- [x] AC #4: unchanged discovery-label RED assertions reached.
- [x] AC #5: preservation controls and real CLI path rerun; GREEN implementation pending.


### 2026-09-06T00:44:09Z ramirosalas
## PM RED Decision
RED APPROVED [2026-09-05]: MAC-sh60 candidate fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3. This approves the immutable RED acceptance bar for GREEN; it does not accept or close the story and does not claim production satisfies the ACs.

The previous rejection is resolved. Diff from b5d3b8c67f487195215860f3d361432f5b2b6b27 is exactly six inserted entries, zero removals, in TestOracleCoverageRejectsNonExecutableEvidence. Fixture bytes/names match the six expressly authorized Elixir @moduledoc/@doc and Ruby block-comment cases. The existing twenty cases, shared assertions, helper semantics, every other test and all production bytes are unchanged. Commit subject contains tdd-red and [test-edit-authorized]. The four older fixture repairs remain exactly their prior reviewed forms.

## Independent replay evidence
- Reviewed full five-AC canonical story, original and supplemental exact authorizations, prior rejection, new complete proof/LEARNINGS/inventory via shared pvg nd. pm_acceptor and codebase-memory skills were fully read during the first review and applied on this resumed review. Prior source/graph assessment remains valid for unchanged bytes; exact new six-line diff is the source authority, with no repeated graph completeness claim.
- Reviewed in own clean detached /tmp/MAC-sh60-pm-rered.NMFVGG/review at fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3. Production base remains 6cb2d974ea8aea211a5974f453cef2b5802bb11e. Rework +6/-0; cumulative 3 files, 291 insertions/8 deletions (299 changed LOC), within budget. git diff --check passed.
- Fresh static pvg verify internal/gates/oraclecov_negative_test.go internal/gates/oraclecov_test.go cmd/machinery/oraclecov_negative_test.go --include-tests --format text: sole existing oraclecov_test.go:158 TODO marker is unchanged quoted negative fixture input. Source-confirmed test data, not a stub; no suppression/repair. No new public API/config or changed product behavior/doc claim in this RED-only delta.
- Command 1 from detached candidate: go test -json -count=1 -timeout=120s ./internal/gates -run 'Oracle|Conformance|Coverage|GtCitationBoundaries' > /tmp/MAC-sh60-pm-rered.NMFVGG/gates.jsonl. Session 66126 completed exit 1: 96 native leaves, 68 pass/28 intended fail/0 skip.
- Command 2: go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestOracleCoverageCLI' > /tmp/MAC-sh60-pm-rered.NMFVGG/cli.jsonl. Session 32924 completed exit 1: 10 native leaves, 4 pass/6 intended fail/0 skip.
- Command 3: go test -json -count=1 -timeout=90s ./internal/gates -run '^TestGt' > /tmp/MAC-sh60-pm-rered.NMFVGG/gt-controls.jsonl. Completed exit 0: 7 pass/0 fail/0 skip. Two boundary leaves repeat the first run; counts are per command, not summed unique.
- Exact independent leaf inventories: /tmp/MAC-sh60-pm-rered.NMFVGG/{gates,cli,gt-controls}-inventory.json, derived from terminal pass/fail/skip records with parent suites excluded. Raw logs retained at the paths above.
- Every new leaf reaches oraclecov_negative_test.go:88 "non-executable evidence established coverage". elixir_moduledoc_ids, elixir_doc_ids and ruby_block_comment_ids return errs=[] and 2 literal IDs; elixir_moduledoc_parser, elixir_doc_parser and ruby_block_comment_parser return errs=[] and 1 conformance parse. All six are actual false-credit failures, not setup/parse/native-compile errors.
- Existing 20 source negatives, mixed-disabled count/diagnosis and two discovery-label tests retain their intended causes. Actual CLI controls pass and five bypasses fail because real Gt exits 0 instead of 1. Built native CLI, temporary real trees and private HOME/config path remain unchanged and executed; no mocks/injected results.
- Native parser control ran again: unchanged table inner TestOracle exits 0 after exactly two checked-row logs; changing only the first expected target B -> A reaches "oracle transition mismatch: got B want A", inner exit 1 and outer corrupt_true PASS. Other-language fixtures establish discovery only, not native execution. No compile/import/read-path/timeout failure is RED evidence.
- Independently matched SHA256: internal/gates/oraclecov_negative_test.go 42f1237819cfecc4a265fd5dfa4a90050d8f0f77f9aa0baf27b42995d8834e45; internal/gates/oraclecov_test.go dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994; cmd/machinery/oraclecov_negative_test.go 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08; unchanged internal/gates/oraclecov.go 9f4bfabcfe43362b62068e59a59f313cd6d27a3490392f66c5fcf71be70684ad.
- Coverage percentage not measured for intentionally failing RED with no production edits; behavioral counts and exact inventories above establish the reviewed bar. No global preflight, external-runtime suite, installation/dev-link/live binary/plugin/skill change, remote action, main/epic merge, or author worktree edit.
- Canonical pvg story approve-red MAC-sh60 succeeded from the detached candidate. Immediate shared readback: Status open, Labels hard-tdd/red-approved (delivered/rejected absent), Assignee dev-MAC-sh60 retained by canonical transition. Root dispatcher owns the next GREEN claim. No --next or accept/close used.

LEARNINGS:
- Six precise language-comment regressions close the reproduced RED gap without weakening parser positives or native assertion-sensitivity controls.
- GREEN must preserve all approved RED tests/fixtures byte-for-byte. Prior authorizations have been fulfilled and provide no continuing license for edits; any new repair requires a separately named reviewer authorization and re-RED.

## nd_contract
status: new
phase: red-approved

### evidence
- RED fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3 independently approved via canonical approve-red; shared status open with hard-tdd/red-approved. Ready for GREEN, not accepted.
- Fresh targeted replay: gates96 = 68 pass/28 intended fail; CLI10 = 4 pass/6 intended fail; extraGt7 pass; zero skips. Exact artifacts, causes and hashes above.

### proof
- [x] AC #1 RED bar: 26 non-executable source cases plus mixed-disabled coverage constrain reproduced bypasses, including all six prior rejection cases.
- [x] AC #2 RED bar: 11 literal language/extension and 7 parser language controls plus 2 native parser semantic/mutation controls pass unchanged; ambiguous evidence negatives remain.
- [x] AC #3 RED bar: full CheckOracleCoverage bypass/zero/mixed/malformed/positive cases and actual CLI integration executed.
- [x] AC #4 RED bar: unit and CLI discovery-versus-execution assertions reach intended failures; static discovery never claims assertions ran.
- [x] AC #5 RED bar: stable-ID/file boundaries, orphan/missing-oracle, parent/clause and Rust/MJS preservation controls pass; actual CLI proof remains valid.
- [ ] GREEN: implement all five ACs with approved RED bytes unchanged, then deliver independent production proof for PM acceptance.

### 2026-09-06T01:04:19Z ramirosalas
GREEN HEALTHY HOLD: root requested independent PM scope review before helper-call expansion/golden changes. Clean committed 3d47b80ce59af1f521539e0b48e67be9ee4e9a71, frozen hashes unchanged, cumulative709changedLOC/5files. Full checkpoint/evidence/proposed bounded helper repair: /tmp/MAC-sh60-green.GtkNbt/healthy-hold.md. Frozen gate96 and actual CLI10 pass; new scope18pass on committed correction. Broader package run terminal81763 completed: gates1171pass/0fail/1skip; cmd381pass/23fail/3skip terminal leaves plus180s package timeout, not full passing assurance. Genuine go-crm helper recognition remains required; separate OCI fixture failures retained for diagnosis. No story delivery or release; claim retained at dispatcher request.

## nd_contract
status: in_progress

### evidence
- HEAD3d47b80ce59af1f521539e0b48e67be9ee4e9a71; raw logs and exact inventories in /tmp/MAC-sh60-green.GtkNbt; no running sessions.

### proof
- [x] Frozen RED bytes preserved; required direct-parser/literal/discovery controls pass.
- [ ] AC #2: retain existing go-crm helper parser discovery after PM scope review.
- [ ] Full delivery: resolve scoped regression and review output golden before delivery.

### 2026-09-06T01:23:55Z ramirosalas
## PM bounded scope and golden authorization — MAC-sh60
Reviewed healthy GREEN checkpoint 3d47b80ce59af1f521539e0b48e67be9ee4e9a71 on 2026-09-05. This is a scope/budget and exact fixture-edit decision only. It is NOT delivery, rejection, re-RED approval, acceptance, claim release or authorization to merge. The current GREEN hold remains until the dispatcher completes the separate RED-amendment review and explicitly resumes GREEN.

### Scope decision: authorized within AC2
AC2 requires genuine conformance parsers to remain discoverable. The unchanged go-crm examples are concrete necessary compatibility, not speculative expansion:
- examples/go-crm/impl/internal/authz/oracle_test.go:25/31/131: literal const oraclePath -> loadOracle(t) -> os.Open(filepath.FromSlash(oraclePath)) -> bufio.NewScanner(f), sc.Scan/sc.Text -> strings.Split and trimmed cells -> append oracleRow fields -> returned rows -> active TestOracleConformance range -> want derived from row.expectation -> t.Run/nested ranges -> got.Allowed != want -> t.Errorf.
- examples/go-crm/impl/internal/authz/tenant_oracle_test.go:26/32/95: same connected loader/typed-row pipeline for Isolation.oracle.md; active TestTenantOracleConformance assigns returned rows, ranges them, derives want from row.expectation and checks AuthorizeLink(...).Allowed inside t.Run.
- The loader's own read-error, malformed-row and empty-result failures do not establish row conformance. The decisive evidence is the returned-row-dependent comparison in the active test. The Policy test intentionally excludes rows marked unreachable; discovery still must not assert that every row executed.
- Current fileNameCited rebuilds only active test declarations, so same-file loaders/constants are absent; goOracleParser handles direct ReadFile/range/Split and cannot follow these Scanner/typed-return/t.Run shapes. TestCheckGreenSummaryLines and TestGoldenCheck/go-crm demonstrate the regression. Preserving the examples rather than rewriting them into a new parser idiom is necessary for AC2.

Authorized implementation boundary: internal/gates/oraclecov.go only, plus directly associated additional regression tests in the existing GREEN supplemental internal/gates/oraclecov_scope_test.go. Permit bounded same-file reachable helper summaries, literal immutable path resolution including filepath.FromSlash, os.Open -> Scanner Scan/Text provenance, typed append/composite/selector propagation, returned rows consumed by active test ranges, and recognized testing.T.Run callbacks. This is an implementation allowance, not an assertion that any particular proposed algorithm is correct.

Required constraints:
1. Every wholesale credit must retain a connected target-oracle read -> actual parsed cells/row values -> returned collection -> active caller iteration -> row-dependent failing comparison. A parser-shaped helper, file name, delimiter, helper name, unused return value or unrelated assertion cannot confer credit.
2. Follow only resolvable same-file calls reached from valid active Go tests. Do not treat every function declaration or arbitrary function literal as invoked. Recognize t.Run through the active testing parameter/callback context; an uncalled closure or unrelated receiver's Errorf must not become a test assertion.
3. Resolve only proven literal/immutable path bindings and reject shadowed/reassigned or otherwise ambiguous bindings. Preserve exact target filename boundaries; a loader for a different oracle cannot cover the requested oracle. Do not pool unrelated helpers/files into a synthetic proof chain.
4. Use explicit finite cycle/depth bounds. Cycles, unresolved calls, ambiguous returns and exhausted analysis limits remain uncovered; limits cannot turn into success or hang the scanner. No general inter-package call analysis, external compiler/runtime, network execution or new dependency is authorized.
5. Preserve existing active test/build selection, comment/docstring rejection, stable-ID boundaries, orphan/missing-oracle and clause obligations. Static discovery remains distinct from execution.

Required supplementary proof before GREEN delivery: actual CheckOracleCoverage fixtures for both direct-range helper return and assign-then-range forms, representing literal const/FromSlash/Open/Scanner/typed rows/selectors/t.Run. Negative controls must cover uncalled loader; loaded-but-unused returned rows; real loader plus unrelated constant assertion; wrong-oracle loader; an uncalled assertion closure; and bounded recursion/ambiguous path reassignment remaining uncovered. These may be table-driven/additive in oraclecov_scope_test.go; use connected positive fixtures and exact no-credit diagnostics/counts. They are GREEN supplemental regression tests, not historical pre-GREEN RED. Current 18 supplementary cases and all frozen original tests remain intact.

Budget investigation: current cumulative diff against 6cb2d974 is exactly 709 changed LOC across five files (649 additions/60 deletions), with 54 lines of supplemental test coverage and no padding. The forecast 140–200 additional lines is justified by preserving the existing real parser idiom and adding safety controls. Authorize up to 1000 cumulative changed LOC across SIX exact paths: internal/gates/oraclecov.go, internal/gates/oraclecov_scope_test.go, the three already-frozen test files, and testdata/golden/check-go-crm/stdout.txt. Forecast remains about 850–910 plus explicit safety-test room; 1000 is a review ceiling, not a target. Any additional file or material growth beyond that bound needs renewed concrete review. No architect decision is presently required for this bounded same-file compatibility repair.

### Exact golden amendment authorization
TEST-EDIT AUTHORIZED: testdata/golden/check-go-crm/stdout.txt — change ONLY the existing Gt checked line (line 55 at this checkpoint) by appending exactly `, static discovery; tests not executed; unsupported parser structures remain uncovered` before its existing newline. This is required AC4 disclosure. Preserve every existing count, all other lines/bytes, stderr.txt (empty), exitcode.txt (`0\n`) and all command test assertions. No golden regeneration/update switch or broad fixture rewrite is authorized.

Exact OLD line:
```text
  checked: 14 test files scanned, 5 machines, 275 oracle rows, 197 ids covered by literal, 2 formal oracles, 2 formal oracles covered, 6 clause-declared guards checked, 36 falsifying-clause ids covered
```
Exact NEW line:
```text
  checked: 14 test files scanned, 5 machines, 275 oracle rows, 197 ids covered by literal, 2 formal oracles, 2 formal oracles covered, 6 clause-declared guards checked, 36 falsifying-clause ids covered, static discovery; tests not executed; unsupported parser structures remain uncovered
```

The two missing formal-oracle errors and loss of `2 formal oracles covered` in healthy GREEN are a production regression to fix. They must NOT be captured as expected golden output. Gate.Emit appends checkedExtra after the existing ordered counts with comma-space separation, supporting this exact suffix-only expectation once helper compatibility is repaired.

Before amendment SHA256 independently read from 3d47b80:
- stdout.txt: 2abeaf18bcc08300ffcf6083f0715c4790e12d26e017f59df0c71d8c89f0e12f
- exitcode.txt: 9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa
- stderr.txt: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855

Separate original RED author owns this amendment, not the GREEN implementer. Produce a tests-only commit on the pure approved RED candidate fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3 (production still 6cb2d974), with both `tdd-red` and `[test-edit-authorized]` in its subject. Only the single golden line may change. Independently replay `go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestGoldenCheck$/^go-crm$'`: expected RED must be specifically the missing disclosure suffix while actual exit stays zero, old counts remain complete and stderr stays empty. Preserve full raw CLI stdout or an exact line diff if the golden harness clips its mismatch. Positive control `go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestCheckGreenSummaryLines$'` must pass on that unchanged production. Record SHA, diff, hashes, counts and assertion causes for separate independent PM re-RED review. This authorization does not itself approve the amended RED. The dispatcher owns integration with the paused healthy GREEN; no source/test edits occur in this review.

The three previously frozen tests remain byte-for-byte immutable at their approved hashes: gate negative 42f1237819cfecc4a265fd5dfa4a90050d8f0f77f9aa0baf27b42995d8834e45; original gate tests dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994; CLI negatives 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08. git diff fa842ed3..3d47b80 over those paths is empty. The golden authorization grants no permission to edit them or either real go-crm example.

### Verification allowance and honest limits
- After resumed implementation and approved golden amendment: rerun frozen gate/CLI tests and all supplementary helper/scope cases; run TestCheckGreenSummaryLines and TestGoldenCheck/go-crm unchanged. Also run actual local `go test -json -count=1 -timeout=120s ./internal/authz -run '^(TestOracleConformance|TestTenantOracleConformance)$'` from examples/go-crm/impl as real native example proof; keep any intentional unreachable-row exclusion explicit. No source-discovery claim substitutes for native execution.
- For the needed full cmd package correctness replay, authorize `go test -json -count=1 -timeout=10m ./cmd/machinery` without coverage instrumentation. This is a finite package budget, not a waiver for hung tests. Local `go help testflag` confirms Go's default timeout is ten minutes; repository CI/preflight native/race commands omit a shorter timeout. The broad instrumented package had already spent about 139 seconds before its final parallel installer group resumed, with TestGoldenGen 18.23s and TestPreflightReturnsOutputFailure 19.73s among observed costs. Three minutes was not evidence of an appropriately budgeted complete run. Ten minutes restores the tool's normal cap while preserving test/process deadlines and requiring terminal results. If it times out, diagnose/report the actual running tests; do not call it a pass or increase indefinitely.
- Correction to healthy-hold.md: the raw log shows TestInstallCommand PASSED in 8.96s. At the 180-second package timeout, the running tests were TestInstallAndDoctorTargetAll, TestInstallScript and TestInstallScriptHostTargets, each shown at 41s. This review does not assign their cause. Existing installer work remains separately owned. The fuller timeout is permission for accurate verification, not assurance those tests are healthy.
- Broader instrumented run remains gates 1171 pass/0 fail/1 skip; cmd 381 pass/23 fail/3 skip terminal leaves plus timeout. It did not complete. Report all failures and unexecuted conditions; do not claim all 23 share one cause. Read-only baseline diagnostic proves only the representative TestVerifyCheckersReproducible instrumentation protocol issue (native main pass 1.203s, covered fail 0.747s; GOCOVERDIR warning appended to strict OCI JSON). Root's separate P0 MAC-yig6 owns that defect; no OCI/parser/environment changes are authorized here and strict identity validation must remain intact.
- Case-sensitive filesystem tests and the official Structurizr/OCI engine opt-in cases remain unexecuted assurance in the supplied run. Their explicit lanes/final gate own required prerequisite coverage; list skips exactly and do not count them as passes. No full scripts/preflight.sh or external runtime lane is authorized during this review/story step.

### Review evidence and source boundary
- Read full /tmp/MAC-sh60-green.GtkNbt/healthy-hold.md and /tmp/machinery-coverage-diagnostic.hKOuC4/REPORT.md; reread canonical five AC and exact prior test-edit restrictions/current GREEN checkpoint via shared pvg nd.
- Read exact candidate oraclecov.go/supplemental test source, complete two actual go-crm oracle tests, golden stdout/exit/stderr, TestGoldenCheck, TestCheckGreenSummaryLines, Gate.Emit and relevant installer test sources. No candidate implementation was modified or executed in this scope-only review; existing raw test evidence is identified as author/root evidence above.
- Graph search found seven exact scoped symbols with has_more false; TestOracleConformance outbound trace included loadOracle and its actual authorizer callees. Direct source established the testing.T.Run and value flows; graph's unrelated heuristic CLI Run edge was not relied upon. Coverage generation 2026-09-05T23:58:53Z reports metadata_match/no recorded issues for unchanged examples and harness, but main-index metadata does not establish GREEN source freshness. New supplementary/golden paths were missing/not_tracked in graph metadata; exact git-show source is authoritative fallback. No graph completeness claim.
- Initial/final state must remain in_progress, hard-tdd/red-approved, assignee dev-MAC-sh60. No story transition, delivery, test/source/golden edit, worktree cleanup, installation, live binary/plugin/skill mutation, remote mutation or main/epic change performed. Only append-only shared review notes/comment are written.

## nd_contract
status: in_progress
phase: green-healthy-hold

### evidence
- Bounded same-file helper compatibility and <=1000 changed LOC / six-path scope authorized for AC2; exact one-line AC4 golden amendment authorized to separate original RED author, pending tests-only replay and independent review.
- Healthy checkpoint 3d47b80ce59af1f521539e0b48e67be9ee4e9a71 retained; frozen three-file delta empty. No workflow state transition in this review.

### proof
- [x] Scope review: genuine existing helper/Scanner/typed-row/t.Run evidence and safety constraints identified.
- [x] Exact golden edit authority: disclosure suffix only; counts, exit zero and empty stderr preserved.
- [ ] Separate tests-only golden RED amendment and independent review pending.
- [ ] AC2 GREEN helper compatibility, supplementary negative controls and actual example replay pending.
- [ ] Complete native correctness verification and honest failure/skip accounting pending; no acceptance or completion claimed.

### 2026-09-06T01:27:41Z ramirosalas
## MAC-sh60 golden RED amendment proof

This amendment was authorized AFTER the healthy GREEN checkpoint 3d47b80ce59af1f521539e0b48e67be9ee4e9a71. It is an honest later tests-only amendment, not evidence claimed to predate GREEN implementation. The independent scope/golden authorization in /tmp/MAC-sh60-pm-scope.mXj9lU/scope-golden-authorization.md was read completely. No GREEN checkout/source/branch was modified.

### Commit and exact boundary
- New detached commit: 8f843256d30068816234d008a85239d662e8bb15
- Subject: test(MAC-sh60): tdd-red [test-edit-authorized] disclose static discovery in go-crm golden
- Parent: pure approved RED fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3; production still 6cb2d974ea8aea211a5974f453cef2b5802bb11e.
- Retained own clean detached proof checkout: /tmp/MAC-sh60-golden-red.E4Jh4K/review. No story branch moved.
- Exact entire diff: testdata/golden/check-go-crm/stdout.txt line 55 only, one insertion/one deletion. Appended exactly ", static discovery; tests not executed; unsupported parser structures remain uncovered" before the existing newline.
- Every old count preserved, including 2 formal oracles covered. All other stdout bytes, stderr.txt empty and exitcode.txt 0 plus newline unchanged. No update/regeneration flag, test/helper/example/source edit, golden bulk rewrite, or additional fixture.
- git diff --check passed; git status --short empty.

### Commands and native outcomes
All commands executed from /tmp/MAC-sh60-golden-red.E4Jh4K/review at the new committed SHA.

1. go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestGoldenCheck$/^go-crm$' > /tmp/MAC-sh60-golden-red.E4Jh4K/golden.jsonl
   Exit 1 EXPECTED RED. Exactly one native leaf: TestGoldenCheck/go-crm FAIL. TestGoldenCheck and package FAIL records are parents, not extra leaves. No skips.
   Sole assertion is golden_test.go:246 stdout golden mismatch. No stderr/exitcode mismatch; no compile/import/path/timeout error.
2. go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestCheckGreenSummaryLines$' > /tmp/MAC-sh60-golden-red.E4Jh4K/control.jsonl
   Exit 0. Exactly one native leaf: TestCheckGreenSummaryLines PASS; no fail/skip. Full positive behavior preserved on unchanged production.
3. go build -o /tmp/MAC-sh60-golden-red.E4Jh4K/machinery ./cmd/machinery
   Exit 0, isolated candidate binary only.
4. env HOME=/tmp/MAC-sh60-golden-red.E4Jh4K/private-home MACHINERY_CONFIG_DIR=/tmp/MAC-sh60-golden-red.E4Jh4K/private-config /tmp/MAC-sh60-golden-red.E4Jh4K/machinery check /tmp/MAC-sh60-golden-red.E4Jh4K/review/examples/go-crm/design --impl /tmp/MAC-sh60-golden-red.E4Jh4K/review/examples/go-crm/impl > /tmp/MAC-sh60-golden-red.E4Jh4K/actual-stdout.txt 2> /tmp/MAC-sh60-golden-red.E4Jh4K/actual-stderr.txt
   Exit 0. Actual stderr zero bytes. Actual stdout matches the original approved RED golden byte for byte: cmp with git show fa842ed3:testdata/golden/check-go-crm/stdout.txt exits 0. All original counts and platform-green preserved.
5. diff -u /tmp/MAC-sh60-golden-red.E4Jh4K/actual-stdout.txt testdata/golden/check-go-crm/stdout.txt > /tmp/MAC-sh60-golden-red.E4Jh4K/stdout.diff
   Exit 1 EXPECTED single exact line difference; full raw stdout captured because golden harness clips after 4000 bytes.
6. pvg verify testdata/golden/check-go-crm/stdout.txt --include-tests --format text
   VERIFY: PASSED (0 files scanned, 0 issues). This source scanner does not inspect the text golden; exact git diff, cmp, hashes and native assertion above provide the relevant proof, not the zero-file scan.

### Exact changed line (actual -> expected)
Actual:
  checked: 14 test files scanned, 5 machines, 275 oracle rows, 197 ids covered by literal, 2 formal oracles, 2 formal oracles covered, 6 clause-declared guards checked, 36 falsifying-clause ids covered
Expected:
  checked: 14 test files scanned, 5 machines, 275 oracle rows, 197 ids covered by literal, 2 formal oracles, 2 formal oracles covered, 6 clause-declared guards checked, 36 falsifying-clause ids covered, static discovery; tests not executed; unsupported parser structures remain uncovered

### SHA256
- stdout.txt BEFORE and actual CLI stdout: 2abeaf18bcc08300ffcf6083f0715c4790e12d26e017f59df0c71d8c89f0e12f
- stdout.txt AFTER: d14ccc0e945d7bd21bbb681894682aa64240a8eb9c995a815f3bf4068b77b184
- exitcode.txt unchanged: 9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa
- stderr.txt unchanged and actual CLI stderr: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
- frozen gate negatives unchanged: 42f1237819cfecc4a265fd5dfa4a90050d8f0f77f9aa0baf27b42995d8834e45
- frozen original gate tests unchanged: dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994
- frozen CLI negatives unchanged: 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08
- unchanged RED production oraclecov.go: 9f4bfabcfe43362b62068e59a59f313cd6d27a3490392f66c5fcf71be70684ad

### Limits and preserved state
- Canonical readback remains in_progress, assignee dev-MAC-sh60, hard-tdd/red-approved, parent MAC-ui8a. No deliver/approve-red/accept/reject/claim/release/state transition was invoked.
- Parent owns independent re-freeze review and integration into healthy GREEN; this amendment has not itself been approved or integrated.
- No full preflight, external runtime lane, coverage instrumentation, installed binary/asset/skill/plugin, remote/main/epic change or worktree removal. All bounded processes completed.
- Previous approved tests/authorizations and original RED evidence remain intact; only exact AC4 golden disclosure changes here. GREEN helper compatibility and broader verification remain the other agent's pending work.

LEARNINGS:
- Golden disclosure must preserve complete legacy counts; a missing formal-oracle recognition result is production regression, not golden expected behavior.
- Full CLI capture resolves golden output clipping and establishes the missing suffix as the sole behavioral RED cause.

## nd_contract
status: in_progress
phase: golden-red-amendment-awaiting-independent-review

### evidence
- Original RED author committed 8f843256d30068816234d008a85239d662e8bb15 on isolated approved-RED checkout after healthy GREEN checkpoint; exact authorized single-line golden change only.
- One intended golden native leaf FAIL, one positive summary leaf PASS, zero skips; actual CLI exits 0 with empty stderr and complete original counts. Full proof/raw logs/diff in /tmp/MAC-sh60-golden-red.E4Jh4K.

### proof
- [x] AC #4 amendment: exact disclosure suffix required by real golden assertion and RED cause independently reviewable.
- [x] Preservation: original counts, zero exit, empty stderr and three previously frozen test hashes unchanged.
- [ ] Independent amended-RED review/re-freeze and parent integration pending.
- [ ] GREEN AC2 helper compatibility and complete five-AC production delivery remain pending; no acceptance claim.


### 2026-09-06T01:32:02Z ramirosalas
## Independent PM golden RED re-freeze — MAC-sh60
APPROVED [2026-09-05]: exact golden amendment 8f843256d30068816234d008a85239d662e8bb15 is reviewed and re-frozen. The golden-amendment hold on healthy GREEN is released. The dispatcher may integrate ONLY this approved tests-only commit into retained healthy GREEN 3d47b80ce59af1f521539e0b48e67be9ee4e9a71, then resume the already authorized bounded helper compatibility work. This is not story acceptance, production approval or a workflow transition; no approve-red/deliver/accept/reject/claim/release command was invoked.

### Independent review evidence
- Read complete /tmp/MAC-sh60-golden-red.E4Jh4K/proof.md, exact commit metadata/diff, native golden/control JSON terminal events, full actual CLI stdout/stderr hashes and stdout.diff. Compared with my exact scope/golden authorization at /tmp/MAC-sh60-pm-scope.mXj9lU/scope-golden-authorization.md and current shared canonical status/terminal proof.
- git rev-parse 8f843256^ is fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3. Entire delta is one insertion/one deletion at testdata/golden/check-go-crm/stdout.txt:55. No other file changed; git diff --check passes. Commit subject contains tdd-red and [test-edit-authorized]. Production and the three previously approved frozen tests remain identical to pure RED.
- Exact appended suffix: `, static discovery; tests not executed; unsupported parser structures remain uncovered`. All preexisting counts, notably `2 formal oracles covered`, other stdout bytes, empty stderr and zero exit expectation remain unchanged.
- Native author command: go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestGoldenCheck$/^go-crm$'. Raw golden.jsonl confirms exactly one leaf TestGoldenCheck/go-crm FAIL at golden_test.go:246 stdout mismatch; parent TestGoldenCheck/package records are not additional leaves. No skip, stderr/exitcode assertion, setup/import/compile/path/timeout failure. Package terminal 2.083s; intended leaf 1.30s.
- Native author positive command: go test -json -count=1 -timeout=120s ./cmd/machinery -run '^TestCheckGreenSummaryLines$'. Raw control.jsonl confirms one PASS, zero fail/skip, test 0.65s.
- The separately built actual native CLI command in proof.md exits 0. Full actual-stdout.txt SHA256 independently equals the original RED golden: 2abeaf18bcc08300ffcf6083f0715c4790e12d26e017f59df0c71d8c89f0e12f. Actual-stderr.txt SHA256 is the empty-file hash e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855. Full stdout.diff independently contains only the precise suffix-only line difference, resolving the golden harness's clipped diagnostic. All prior counts and platform-green output therefore remain intact.
- New frozen stdout.txt SHA256 independently matches d14ccc0e945d7bd21bbb681894682aa64240a8eb9c995a815f3bf4068b77b184. Unchanged exitcode.txt SHA256 remains 9a271f2a916b0b6ee6cecb2426f0b3206ef074578be55d9bc94f6f3fe3ab86aa; unchanged stderr.txt remains empty.
- Proof is complete, consistent and attributable to this exact candidate. Per pm_acceptor evidence rules, no redundant native rerun was needed. The author's pvg verify result scanned zero text-golden files and is not treated as behavioral evidence; exact diff/hashes plus real native assertions provide the proof.
- Read-only status checks show the author's detached proof checkout remains clean and healthy GREEN remains clean at 3d47b80ce59af1f521539e0b48e67be9ee4e9a71. Neither was edited or removed by PM. All source/test/fixture bytes remain untouched by this review.

### Re-freeze and continuation boundary
The original RED author's exact one-line authorization is fulfilled and exhausted; it grants no continuing test-edit permission. The golden file is now frozen at the hash above alongside the original three frozen tests:
- internal/gates/oraclecov_negative_test.go: 42f1237819cfecc4a265fd5dfa4a90050d8f0f77f9aa0baf27b42995d8834e45
- internal/gates/oraclecov_test.go: dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994
- cmd/machinery/oraclecov_negative_test.go: 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08

Root owns integration; GREEN may not edit any of these frozen bytes. Any future fixture repair requires a separately named authorization and genuine tests-only RED proof. This later amendment is explicitly subsequent to the healthy GREEN checkpoint; it is not claimed as original pre-GREEN evidence.

The previously approved <=1000 cumulative changed LOC / six named paths and same-file helper/Scanner/typed-row/t.Run boundaries remain authoritative. Implement connected target-oracle-to-row-to-active-failing-comparison provenance and required negative controls, preserve existing example semantics and formal counts, then run native example/targeted/full-cmd verification exactly as scoped. New helper tests in the GREEN supplemental file remain supplemental, not historical RED.

The finite full native cmd timeout allowance remains 10m without coverage instrumentation; timeout or skipped tests cannot count as completed assurance. Prior raw-log correction stands: TestInstallCommand passed; TestInstallAndDoctorTargetAll, TestInstallScript and TestInstallScriptHostTargets were active at the broad 180s timeout. Separate MAC-yig6 owns the proved coverage-instrumented fixture protocol defect; no claim attributes all 23 broad failures to it. No full preflight, external runtime lane, installed asset/binary changes, remote action, main/epic mutation or unrelated installer/OCI fix is authorized by this re-freeze.

## nd_contract
status: in_progress
phase: green-ready-after-golden-refreeze

### evidence
- Golden RED amendment 8f843256d30068816234d008a85239d662e8bb15 independently reviewed and frozen; exact native one-leaf disclosure FAIL and one positive PASS, zero skips, full CLI suffix-only difference proven.
- Healthy GREEN hold released for dispatcher-owned integration of only this approved tests commit and already-scoped helper implementation; current labels/status/claim unchanged.

### proof
- [x] AC4 golden amendment reviewed/re-frozen; complete prior counts, zero exit and empty stderr retained.
- [x] Original three frozen tests and pure RED production unchanged by amendment.
- [ ] Dispatcher integration and AC2 bounded helper compatibility/negative tests pending.
- [ ] Native examples, required targeted checks and complete full-cmd correctness verification pending.
- [ ] Five-AC GREEN delivery and independent acceptance pending; no production completion claimed.

### 2026-09-06T01:49:29Z ramirosalas
GREEN HEALTHY HOLD FOR INDEPENDENT TDD AUDIT REVIEW: final clean a6ac10ebadc0e6b6a344d50aa10d0b810ce6b940; implementation native proof complete, no delivery. Report /tmp/MAC-sh60-green-final.8U4Ho2/healthy-hold-audit.md includes exact authorization timeline, all4frozenhashes,932LOC/6paths, raw inventories and three flagged supplemental-only commit diffs. Final exact gate130pass/0fail/0skip; CLI10pass/0fail/0skip; full native gates1187pass/0fail/1skip; full cmd409pass/0fail/3skip completed273.048s; unchanged native examples74pass,4unreachable Policy rows explicitly excluded. Original18supplement cases preserved;16helper controls pass. pvg verify-tdd flags3d47b80/088ef8a/a6ac10e only for GREEN supplemental file edits without markers; no frozen bytes changed or history rewritten. Dispatcher requests PM adjudication before delivery. All terminal sessions completed; claim retained.

## nd_contract
status: in_progress
phase: green-native-complete-audit-hold

### evidence
- HEAD a6ac10ebadc0e6b6a344d50aa10d0b810ce6b940; report/raw outputs/exact inventories /tmp/MAC-sh60-green-final.8U4Ho2.
- Approved golden8f843256 integrated as7b200b6; all4frozenhashes match.

### proof
- [x] AC #1: frozen non-executable/mixed negatives pass.
- [x] AC #2: preserved direct/literal/helper/native example positives and connected-provenance negatives pass.
- [x] AC #3: full gate matrix and actual10case CLI pass.
- [x] AC #4: explicit discovery-only output and exact approved golden pass.
- [x] AC #5: shared full gate/CLI preservation controls pass.
- [ ] Independent private TDD audit adjudication pending; do not deliver until resolved.

### 2026-09-06T02:13:22Z ramirosalas
## Independent PM supplemental TDD audit adjudication — MAC-sh60
Reviewed candidate a6ac10ebadc0e6b6a344d50aa10d0b810ce6b940 on 2026-09-05. Disposition: NO FROZEN RED TAMPERING FOUND in the three flagged deltas; the mechanical verify-tdd result remains FAIL and has not been waived or converted to PASS. This is a bounded audit adjudication, not acceptance, rejection, delivery, claim release or a workflow transition.

### Exact timeline and delta findings
1. The approved original RED tip is fa842ed374ad5c82d8c8f4f9e0c3aead56c9cfa3. The supplemental path internal/gates/oraclecov_scope_test.go does not exist there. git log a6ac10e --diff-filter=A identifies first addition as GREEN commit 4cda3c388c3efef2cdf952adb31d6c7507c307da at 2026-09-05T17:58:06-07:00, after production GREEN commits 6f2c165/e13a2fd. It is not an original RED file.
2. Commit 3d47b80ce59af1f521539e0b48e67be9ee4e9a71, 17:58:43-07:00, changes only construction of that new file's Python source_after_test fixture (+5/-1). Previously the appended Python body retained four spaces and therefore belonged to the preceding test function. The correction removes one indentation level when emitting source_after_test, making it the intended non-test source. It preserves all assertion conditions, helper_after_test/active fixtures and case names. This is correction of a newly created GREEN fixture, not repair, weakening or retroactive replacement of frozen RED.
3. My canonical scope review was recorded at 2026-09-06T01:23:55Z, explicitly authorizing additional regression tests in this existing GREEN supplemental file and stating they are not historical pre-GREEN RED. It reviewed the corrected 3d47b80 checkpoint and required the current 18 scope cases remain intact. My re-freeze/continuation decision at 01:32:02Z retained that same allowance. These authorizations precede both subsequent helper commits.
4. Commit 088ef8a6453df139aec6dcc9386f1021ab7ca6fc, 18:38:51-07:00, appends 66 lines containing TestOracleHelperDiscoveryRequiresConnectedRows. The already-reviewed first 54 lines and all 18 earlier cases are unchanged. The new positive direct/assigned helper-return cases and uncalled/unused/constant/wrong-oracle/uncalled-closure/recursive/ambiguous-path/unrelated-receiver/depth-bound negatives implement the previously authorized connected-provenance and bounded-analysis scope. No prior assertion is removed or weakened.
5. Commit a6ac10ebadc0e6b6a344d50aa10d0b810ce6b940, 18:41:02-07:00, adds only two supplemental matrix entries: constant_target_field and call_budget. They prevent inferred row provenance from a constant target and require exhausted helper work to remain uncovered. Both are within the same earlier authorized provenance/bounds scope. Assertions and preexisting fixtures remain unchanged.
6. git diff --numstat 3d47b80..a6ac10e -- internal/gates/oraclecov_scope_test.go is exactly 68 insertions/0 deletions. Cumulative candidate diff from 6cb2d974 is 932 changed LOC across the six authorized paths, inside the reviewed 1000-line/six-path ceiling. No history/commit marker edits are part of this review.

### Frozen evidence independently verified
The exact candidate git-object bytes hash to all four approved values:
- internal/gates/oraclecov_negative_test.go: 42f1237819cfecc4a265fd5dfa4a90050d8f0f77f9aa0baf27b42995d8834e45
- internal/gates/oraclecov_test.go: dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994
- cmd/machinery/oraclecov_negative_test.go: 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08
- testdata/golden/check-go-crm/stdout.txt: d14ccc0e945d7bd21bbb681894682aa64240a8eb9c995a815f3bf4068b77b184

Golden integration 7b200b6 carries only the separately approved/re-frozen disclosure amendment. No authorizations here permit further frozen-test changes. The supplemental additions are honestly GREEN-time work; they must not be relabeled as pre-GREEN RED.

### Mechanical audit and remaining policy boundary
Raw /tmp/MAC-sh60-green-final.8U4Ho2/tdd-audit.txt says the nine-commit range checked zero skipped merges and FAILS exactly the three supplemental-file commits above for lacking a tdd-red or [test-edit-authorized] commit-subject marker. The raw result is real and remains authoritative as the tool's result. The file-origin/delta evidence shows the flagged actions do not violate the substantive immutable-RED boundary, and the two later additive edits had prior explicit PM scope authorization. That is not the same as producing a clean audit.

The current pm_acceptor skill, /Users/ramirosalas/.codex/skills/pm_acceptor/SKILL.md:84, explicitly states: "A `verify-tdd` failure is a rejection." That sentence governs a future GREEN acceptance review. Neither the earlier bounded implementation/test-scope allowance nor this request to adjudicate the audit explicitly waives that categorical acceptance rule. Therefore this PM does not silently manufacture an audit pass or grant acceptance despite it. Since this is not a delivered acceptance review, no rejection transition is applied now.

Available bounded resolutions for the dispatcher to seek, without changing the product or rewriting evidence:
- Explicit user direction authorizing a one-time exception to the skill's mechanical-audit acceptance rule for exactly commits 3d47b80/088ef8a/a6ac10e and exactly internal/gates/oraclecov_scope_test.go, based on this independent no-frozen-RED-tampering finding. Keep the raw audit FAIL visible and all four frozen hashes protected. This would permit subsequent ordinary GREEN delivery/review; it would not itself accept the implementation.
- Separately authorize a future correction to the private audit's treatment of GREEN-created supplemental files, with dedicated proof, then rerun the audit. Such tool work is outside the current authorization and must not become a Machinery product dependency or a silent bypass. No such change is performed or assumed here.

History rewrite/rewording, retroactive tdd-red/test-edit marker insertion, product changes to satisfy private Paivot tooling, or claiming this command passed are not valid resolutions under the present constraints. Prior PM scope approval is recognized as evidence of authorized test work; it is not retrospectively presented as a waiver of the audit-return-code rule.

### Evidence scope and limitations
- Fully read healthy-hold-audit.md, tdd-audit.txt and all three referenced tdd-supplement-*.diff artifacts. Inspected the exact candidate commit sequence and first-added supplemental file, independently verified +68/-0 since the reviewed checkpoint and all four git-object hashes, and checked shared authorization timestamps/current status.
- The report records final exact native gates 130 PASS/0 FAIL/0 SKIP, CLI 10 PASS/0 FAIL/0 SKIP; full native gates 1187 PASS/1 SKIP and cmd 409 PASS/3 SKIP, with command package completing in 273.048s. Actual example proof records 74 passing native subtest leaves plus four unreachable Policy rows deliberately excluded. These are delivery artifacts for later full acceptance review, not a new acceptance decision in this bounded audit. No native rerun was needed to determine the three history/delta findings.
- Earlier covered-run failures remain separate historical evidence; MAC-yig6 owns the independently proved representative instrumentation/protocol defect. No claim attributes all 23 earlier failures to it. Filesystem/runtime opt-in skips remain unexecuted assurance. Earlier timeout correction remains intact: TestInstallCommand passed; three other installer tests were still active.
- Retained author worktree readback is clean at a6ac10ebadc0e6b6a344d50aa10d0b810ce6b940. No code, test, fixture, history, private tooling, installed assets, worktree or process state was modified. Only append-only review notes/comment are written. No full preflight, external runtime lane, remote/main/epic action or workflow transition occurred.

## nd_contract
status: in_progress
phase: green-audit-policy-disposition-pending

### evidence
- Independently verified the three flags are GREEN supplemental-only edits: initial fixture correction then previously authorized additive helper controls; all four approved frozen hashes unchanged, exact candidate 932 LOC/six paths.
- Mechanical verify-tdd remains FAIL for three missing-marker commits. No audit waiver, fabricated pass or acceptance decision made; exact policy sentence and bounded resolution choices recorded above.

### proof
- [x] Frozen RED preservation verified; no weakening in the three flagged supplemental deltas.
- [x] Prior authorization timeline for helper additions verified; corrected Python fixture was new GREEN work reviewed before helper expansion.
- [ ] Explicit disposition of mechanical-audit acceptance-rule conflict pending before a successful GREEN acceptance can be claimed.
- [ ] Formal GREEN delivery and full independent five-AC acceptance review remain pending.
