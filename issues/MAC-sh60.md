---
id: MAC-sh60
title: "Require executable oracle coverage evidence"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd, red-approved]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:26Z
created_by: ramirosalas
updated_at: 2026-09-06T01:04:19Z
content_hash: "sha256:7161d4e0f468b0c9bf7d6cec83b746fbe47755b9f91dddbe4d52728d631bf279"
blocks: [MAC-vx24, MAC-ou97]
follows: [MAC-a89e, MAC-p8ce, MAC-olrx]
assignee: dev-MAC-sh60
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

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-ou97]]
- Follows: [[MAC-a89e]], [[MAC-p8ce]], [[MAC-olrx]]

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
