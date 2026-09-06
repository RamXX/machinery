---
id: MAC-sh60
title: "Require executable oracle coverage evidence"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:26Z
created_by: ramirosalas
updated_at: 2026-09-06T00:08:54Z
content_hash: "sha256:8727b4496bffc19a10ed575d078253fbd1eed66623f3f379b6cd6acd97bfa44c"
blocks: [MAC-vx24, MAC-ou97]
assignee: dev-MAC-sh60
follows: [MAC-a89e]
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

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-ou97]]
- Follows: [[MAC-a89e]]

## Comments
