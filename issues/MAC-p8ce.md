---
id: MAC-p8ce
title: "Bind consumer READS to each event edge"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd, red-approved, delivered]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:26Z
created_by: ramirosalas
updated_at: 2026-09-05T22:55:07Z
content_hash: "sha256:c86e2201195103a6db513119faa6a2f4f081dc4af100ba780f542112fe5f0960"
blocks: [MAC-gcrr, MAC-ou97]
assignee: dev-MAC-p8ce
follows: [MAC-olrx]
---

## Description
## USER INTENT
Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.

## Context (Embedded)
Assessment F6/NEXT12: read completeness searches matrices by event alone. Declaration in payments consumer counts as declaration for audit sibling too. User needs exact read set per event-consumer edge rather than prose intersection.

## Ownership
Own only these paths and directly associated tests: internal/gates/readscomplete.go, internal/gates/reads_consumer_test.go. You are not alone in this codebase; preserve other edits and coordinate any shared-file changes with dispatcher.

## Boundary Map
PRODUCES:
- internal/gates/readscomplete.go -> hardened behavior and regression proof
- internal/gates/reads_consumer_test.go -> hardened behavior and regression proof
CONSUMES:
- Existing Machinery source interfaces.
  spec: checkReadsComplete(g *Gate, design, archText string)

### Story Acceptance Criteria
1. Support exact per-consumer event-contract reads using an explicit row-local reads declaration/column or unambiguous matrix owner binding; G2/Gx enforce that edge's required fields against its actual payload.
2. With event fan-out to two consumers, one declaration never satisfies the other's missing declaration or reduces the other's field requirements. One consumer may legitimately read a strict superset without imposing it on siblings.
3. Retain compatible single-consumer declarations only when ownership resolves uniquely; ambiguous legacy fan-out declarations fail with actionable migration guidance, not silent aggregate fallback.
4. Handle repeated event/consumer rows, duplicate/conflicting overrides, unknown fields, renamed consumers, no-reads reason waivers and empty/malformed overrides deterministically. Waiver never transfers to a sibling.
5. Positive two-consumer distinct read sets pass; removing either owner declaration, narrowing only its payload, reassigning its owner or duplicating a conflicting row blocks. Real CLI checks exercise both G2 and Gx.

## Testing Requirements
- Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
- Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
- Commands: go test ./internal/gates -run 'Reads|EventContract|Consumer'; full machinery check isolated design fan-out fixtures. RED includes one consumer masking another from assessment.
- No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.

## OUT OF SCOPE
- Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
- Global heavy preflight, main merge and local release binary belong to final epic gate.

## DIFF BUDGET
- ~3-6 files, under 800 changed LOC; material overrun requires PM investigation, not weakened requirements.

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


## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## Implementation Evidence

PROOF:

### CI/Test Results

Commands run:
- All shell/test commands used /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-p8ce on story/MAC-p8ce.
- go test -count=1 -timeout=5m ./internal/gates -run 'Reads|EventContract|Consumer' -json -coverprofile=/tmp/MAC-p8ce-green-final.cover
- go test -count=1 -timeout=5m ./internal/gates -run 'G2|C4|Traceability|EventWiring|Reads|EventContract|Consumer|Payload' -json -coverprofile=/tmp/MAC-p8ce-green-blast.cover
- go test -count=1 -timeout=5m ./internal/gates -run '^TestEventCells' -json
- go test -timeout=5m ./internal/gates -list 'G2|C4|Traceability|EventWiring|Reads|EventContract|Consumer|Payload'
- go tool cover -func=/tmp/MAC-p8ce-green-final.cover
- go tool cover -func=/tmp/MAC-p8ce-green-blast.cover
- pvg verify internal/gates/readscomplete.go internal/gates/reads_consumer_test.go internal/gates/reads_consumer_supplemental_test.go README.md skills/machinery/references/c4-standalone.md skills/machinery/references/build-md-template.md --include-tests --format text
- pvg story verify-tdd --base 497419ab4512fcff765cd5feb27aed4c67b5608d
- git diff --exit-code 0d52f43b -- internal/gates/reads_consumer_test.go
- git diff --check 497419a..HEAD
- git diff --numstat 497419a..HEAD
- git status --porcelain

Summary: final targeted run PASS 78 leaves / 0 FAIL / 0 SKIP (41 frozen story cases including 9 actual CLI cases; 12 new supplemental cases; 25 existing cases), exit 0, 4.463s. Broader downstream run PASS 134 leaves / 0 FAIL / 0 SKIP, exit 0, 3.066s. Separate G2 EventCells run PASS 15 leaves / 0 FAIL / 0 SKIP. The broader run contains the targeted cases; combined breadth is 149 distinct passing leaves, not an additive claim over repeated executions. JSON reducers emitted every terminal leaf result and every OutputType=error; final errors arrays were empty. No mocks, environmental skips, missing prerequisites, or service dependency.

Coverage: downstream package statement coverage 24.9%; readscomplete.go covered 138/141 statements (97.87%) from the targeted profile. Function coverage: readsCompleteArmed 100%, armedReadsEvents 100%, collectConsumerReads 92.3%, parseConsumerReadSet 100%, bindConsumerReads 100%, checkReadsComplete 100%. Targeted runtime printed 19.4% package coverage while go tool cover rounded its aggregate to 19.3%; source-file numerator/denominator above is measured directly from the profile. Uncovered collector statements are diagnostic/fallback paths, not an assertion of complete path coverage.

Native inventory:
- Frozen tests: LegacySingleControl (1), RepeatedLegacyStillHasOneOwner (1), ExactEdgesPass (6), RejectsOwnershipMutations (13), AmbiguousLegacyNeedsMigration (2), MalformedDeclarationsFailClosed (8), ExplicitBlankOwnerCannotUseLegacyFallback (1), CLI (9): all 41 PASS unchanged.
- CLI cases: legacy_control, distinct_sets, missing_audit, missing_payments, audit_payload_narrowed, payments_payload_narrowed, renamed_known_consumer, conflicting_duplicate, unknown_participant_G2: all PASS. Tests compile ./cmd/machinery to a real temporary binary and execute check <temporary-design> --gate g2,gx; unknown-participant case uses g2. Both valid fixtures and intended blocking exits are asserted.
- Supplemental: reordered/annotated owner, escaped pipe, per-table column positions, unique legacy prose, order-independent exact sets, duplicate owner columns, short explicit row, malformed closing brace, unclosed waiver, duplicate waivers, unknown extra owner, nonconsumer header: 12 PASS.
- Downstream run includes CheckC4 and CheckTraceability against all three repository examples, event wiring/waivers, interface contracts, allow graph, unarmed READS behavior and warn-tier handoff. Separate EventCells inventory covers participant resolution, externals, nested annotations, every table, column answers and dedupe.
- Per user scope, full package/repository suite and scripts/preflight.sh were not run. No installed binary, installed skills/plugins, Docker container, remote git or GitHub state changed.

Intermediate results and errors:
- First GREEN at 6131abf: original 66 leaves all PASS, including frozen 41 and actual CLI 9.
- New supplemental tests initially exposed three in-scope unsafe accepts at that intermediate implementation: extra closing brace, unclosed waiver beside a valid declaration, duplicate waivers hiding an empty reason. That run was 9 PASS / 3 FAIL / 0 SKIP. All three were corrected in owned source and pass at final SHA. Frozen RED was never edited.
- pvg issues comment --help returned exit 1 with 'flag: help requested'; documentation output was read, not counted as a product failure. pvg story deliver --help printed a misleading OK; subsequent story read confirmed status in_progress with hard-tdd/red-approved and no delivered label or mutation. Actual story delivery is performed once below.
- No unresolved product test failures or warnings encountered.

### Commit

Branch: story/MAC-p8ce
RED SHA: 0d52f43b393d961160aeea0db43b13a3fa5c284a
GREEN SHA: 337b9cec17a3f551a7564bed9394045b36043976
Implementation commit: 6131abf; supplemental/strict-malformed/docs commit: 337b9ce.
Frozen RED file byte-identical to approved SHA; local-base verify-tdd PASS: checked 3 commits, skipped 0 merges, no unauthorized test edits.
Whole story diff from 497419a: 6 files, 619 insertions / 33 deletions = 652 changed LOC, including frozen 336-line RED addition. Worktree clean; diff check PASS. No rebase or merge performed.

### Wiring

Existing production paths remain checkEventCells via CheckC4 at gates.go:971 and checkReadsComplete via CheckTraceability at gates.go:1992. The new binder and parser are called directly from the armed check; they use existing readDesignFile, strictSortedGlob, ir.ParseMdTables, ir.FindCol and ir.CleanCell. Product remains standalone Go with no Paivot runtime dependency. Ownership is exact cleaned event-contract consumer, never machine basename. Legacy resolution counts distinct consumers, including waived siblings; explicit sibling coverage never licenses ambiguous legacy fallback.

### pvg verify

VERIFY: PASSED (3 files scanned, 0 issues). All six changed paths were supplied explicitly; this tool scanned the three Go files and did not claim Markdown validation.
Doc freshness manually reviewed: dispatcher explicitly authorized only READS passages in README.md, skills/machinery/references/c4-standalone.md and skills/machinery/references/build-md-template.md. All three now document exact participant binding, unique legacy migration, sibling isolation and malformed/conflicting declarations while preserving unarmed Gd semantics. No installed skill/plugin path or SKILL.md edited.
Codebase-memory Verify used root project Users-ramirosalas-workspace-machinery, generation 2026-09-05T20:28:41Z; relevant graph search and both-direction trace found sole checkReadsComplete caller CheckTraceability. Coverage metadata matched original source; worktree-only tests were missing from the root graph. Every consumed/changed path was read directly in the worktree as the changed/unindexed fallback. Graph freshness is not claimed for this branch.

### AC Verification

| AC # | Requirement | Code Location | Test Location | Status |
|------|-------------|---------------|---------------|--------|
| 1 | Exact per-consumer declaration and own payload; joint G2/Gx enforcement | readscomplete.go collectConsumerReads, bindConsumerReads, checkReadsComplete; existing gates.go G2/Gx call sites | frozen ExactEdgesPass/distinct_sets, annotation_does_not_change_participant; CLI/distinct_sets and unknown_participant_G2 | PASS |
| 2 | Missing siblings cannot be masked; strict supersets remain local | readsEdge key, per-edge canonical read set, row payload loop | frozen RejectsOwnershipMutations missing owners/reassignment/independently narrowed payloads; ExactEdgesPass/strict_superset_is_local | PASS |
| 3 | Unique legacy retained; fan-out gives migration guidance | bindConsumerReads distinct-owner resolution and explicit-column errors | LegacySingleControl, RepeatedLegacyStillHasOneOwner, both AmbiguousLegacyNeedsMigration cases, ExplicitBlankOwnerCannotUseLegacyFallback | PASS |
| 4 | Repeats, conflicts, unknown fields, renamed owners, malformed values and local waivers deterministic | canonical sorted sets, strict parseConsumerReadSet, exact owner lookup, row-local waiver validation | frozen repeats/multimachine agreements/conflicts/near-match/rename/malformed/waiver cases; 12 supplemental authoring-boundary cases | PASS |
| 5 | Positive distinct sets and all prescribed negative mutations through real CLI | same G2/Gx production path | all nine frozen TestReadsConsumerCLI leaves PASS with correct success/blocking exit and intended findings | PASS |

LEARNINGS:
- Consumer identity is architectural; machine filenames cannot safely establish it. Table-local columns preserve identity across column order changes and multiple tables.
- Comparing canonical complete sets prevents both sibling field leakage and accidental unions of conflicting declarations across machines.
- Malformed overrides must remain visible even when a valid declaration would otherwise satisfy the edge; waiver syntax needs the same strictness.
- The existing markdown parser preserves escaped pipes, allowing exact ownership without a second cell grammar.
- Full-path positive controls and native leaf inventory distinguish semantic gate rejection from unrelated fixture or process failures.

## nd_contract
status: delivered

### evidence
- GREEN 337b9cec17a3f551a7564bed9394045b36043976; frozen RED 0d52f43b393d961160aeea0db43b13a3fa5c284a unchanged.
- Targeted 78 PASS; broader G2/Gx 134 PASS; separate EventCells 15 PASS; all 0 FAIL/0 SKIP. Actual CLI 9 PASS.
- Source statement coverage 138/141 (97.87%), downstream package 24.9%; pvg verify 3 code files/0 issues; verify-tdd local base PASS.

### proof
- [x] AC #1: exact event-consumer binding and own payload validated through production G2/Gx.
- [x] AC #2: sibling absence and local strict-superset semantics independently verified.
- [x] AC #3: unique legacy accepted and fan-out rejected with explicit migration guidance.
- [x] AC #4: deterministic repeat/conflict/malformed/rename/waiver behavior verified.
- [x] AC #5: all nine real CLI control/mutation cases pass unchanged.


## nd_contract
status: red-approved

### evidence
- RED tests approved via pvg story approve-red on 2026-09-05.

### proof
- [ ] GREEN developer must implement against the approved RED tests without modifying them.


## Independent PM RED review — 2026-09-05
RED APPROVED: frozen SHA 0d52f43b393d961160aeea0db43b13a3fa5c284a on story/MAC-p8ce meets the hard-TDD specification bar. This is RED approval only, not GREEN acceptance or closure.

Independent evidence:
- Read the full story and all 336 lines of internal/gates/reads_consumer_test.go plus the unchanged READS implementation and legacy fixture definitions. The diff from 497419ab4512fcff765cd5feb27aed4c67b5608d adds only this test file; subject includes tdd-red. Worktree remained clean and HEAD unchanged.
- Replayed go test -count=1 -timeout=240s ./internal/gates -run 'Reads|EventContract|Consumer' -json. Initial output exceeded tool display capacity, so repeated the same command with a jq reducer preserving terminal leaf results and every OutputType=error assertion. Exit 1; 66 leaves: 39 PASS, 27 intended assertion FAIL, 0 SKIP. New TestReadsConsumer cases: 41 leaves, 14 PASS/27 FAIL. Existing matched cases: 25 PASS. CLI: 9 leaves, 4 PASS/5 intended FAIL.
- Independently replayed go test -count=1 -timeout=240s ./internal/gates -run '^TestReadsConsumerLegacySingleControl$|^TestReadsConsumerRepeatedLegacyStillHasOneOwner$|^TestReadsConsumerCLI$/legacy_control$' -v: exit 0, 3 leaves PASS, 0 FAIL/SKIP.
- All 27 failures are unsafe contract acceptance or rejection of valid distinct consumer payloads. No compilation, import, subprocess-launch or prerequisite failure counted as RED. The real temporary CLI build and bounded native invocations executed; no mocks or conditional skips.
- git diff --check 497419ab4512fcff765cd5feb27aed4c67b5608d..HEAD passed. Stub/skip scan found no markers. No product code changed, so RED does not yet change documented public behavior.
- Codebase-memory coverage was checked; the new test file is absent in the root graph. Exact retained-worktree source was therefore read directly; no claim of graph completeness.
- Statement coverage percentage was not measured; the above are actual behavioral test counts. No full preflight, installed binary replacement, remote mutation, or product dependency on Paivot.

AC review:
1. Exact reads are bound by explicit row-local consumer column, never machine basename. Distinct per-edge payloads and annotated participant spellings are positive cases. Real G2/Gx invocation enforces the joint guarantee: G2 participant resolution and Gx READS.
2. Missing either owner, reassigning an owner, each independently narrowed payload, and a legitimate strict-superset sibling are sensitive cases. Existing event-wide matching fails these as intended.
3. Unique single-consumer legacy declarations and repeated rows with one distinct owner pass controls. Ambiguous all-legacy and mixed explicit/legacy fan-out require event/consumer ownership migration guidance. Present-but-blank ownership cannot use legacy fallback.
4. Tests cover repeated identical edges, repeated short payloads, multiple machines with agreeing exact sets, conflicting sets across machines/rows/cells, duplicate members, unknown fields, malformed/empty declarations, near-match and renamed owners, and empty waiver reasons. A reasoned waiver passes locally but cannot satisfy the sibling. Blocked ownership mutations compare repeated diagnostics for determinism.
5. Native CLI positive control/distinct sets and missing-owner, narrowed-payload, known-consumer rename, conflicting duplicate, and G2 unknown-participant cases are real filesystem/CLI checks with exit-code and relevant finding assertions.

Remaining work: GREEN implementation must make the frozen suite pass without editing its tests/fixture bytes. No RED acceptance-criteria gaps found.

## nd_contract
status: delivered

### evidence
- Independent RED review at 0d52f43b393d961160aeea0db43b13a3fa5c284a: 39 PASS/27 intended FAIL/0 SKIP, separate controls 3 PASS.
- RED approval recommended; canonical approve-red transition follows this record.

### proof
- [x] AC #1: exact per-consumer binding and combined G2/Gx path specified.
- [x] AC #2: sibling isolation and strict-superset independence specified.
- [x] AC #3: compatible unique legacy ownership and actionable ambiguity migration specified.
- [x] AC #4: repeats, conflicts, malformed values, renames and non-transferring waivers specified.
- [x] AC #5: native CLI mutation/control coverage independently replayed.


## Implementation Evidence

PROOF:

### CI/Test Results

Commands run:
- go test -count=1 -timeout=240s ./internal/gates -run 'Reads|EventContract|Consumer' -v
- go test -count=1 -timeout=240s ./internal/gates -run '^TestReadsConsumerLegacySingleControl$|^TestReadsConsumerRepeatedLegacyStillHasOneOwner$|^TestReadsConsumerCLI$/legacy_control$' -v
- pvg verify internal/gates/reads_consumer_test.go --include-tests --format text
- git diff --check

Summary: RED run has 66 leaf cases: 39 PASS, 27 intended behavioral FAIL, 0 skipped. Story-specific 41 leaf cases: 14 PASS / 27 FAIL. Existing selected cases: 25 PASS / 0 FAIL. Real CLI 9 cases: 4 PASS / 5 intended FAIL. Separate control run: 3 PASS / 0 FAIL. Quality scan PASS: 1 file, 0 issues. Diff check PASS.

Coverage: statement coverage percentage was not measured. Behavioral mutation inventory is recorded above. Tests are not rerun during this evidence-format repair; outputs are reused from their honest runs at the unchanged RED SHA.

Key output:
- Unsafe missing_audit and missing_payments contracts returned real CLI success with 0 blocking findings.
- Distinct valid per-consumer payloads were rejected because sibling fields leaked across edges.
- The committed-SHA run contains no compile/import/infrastructure failures. All 27 failures are intended RED behavior assertions.

### Commit

Branch: story/MAC-p8ce
SHA: 0d52f43b393d961160aeea0db43b13a3fa5c284a
Only frozen file: internal/gates/reads_consumer_test.go. No source changes or test changes during this formatting repair.

### Wiring

The test builds a native candidate in a temporary directory from this worktree and invokes actual check --gate g2,gx against isolated complete G2/Gx fixtures. G2 validates participants, Gx validates READS; unknown-participant control invokes G2 alone. No installed binary replacement, mocks, skips, services, full preflight or remote writes.

### pvg verify

VERIFY: PASSED (1 files scanned, 0 issues).

### AC Verification

| AC # | Requirement | Code Location | Test Location | Status |
|------|-------------|---------------|---------------|--------|
| 1 | Exact event-consumer read set | Unchanged readscomplete.go | TestReadsConsumerExactEdgesPass; CLI/distinct_sets | RED executed |
| 2 | Missing sibling and strict-superset independence | Unchanged readscomplete.go | RejectsOwnershipMutations; ExactEdgesPass/strict_superset_is_local | Intended RED plus passing narrowed-payload controls |
| 3 | Unique legacy compatibility and ambiguity migration | Unchanged readscomplete.go | LegacySingleControl; RepeatedLegacyStillHasOneOwner; AmbiguousLegacyNeedsMigration; ExplicitBlankOwnerCannotUseLegacyFallback | Controls PASS and intended ambiguity RED |
| 4 | Repeats, conflicts, malformed data, renames, waivers | Unchanged readscomplete.go | ExactEdgesPass; RejectsOwnershipMutations; MalformedDeclarationsFailClosed | Positive and negative cases executed |
| 5 | Real G2/Gx CLI mutations | Existing CLI path | TestReadsConsumerCLI | 9 cases: 4 PASS / 5 intended FAIL |

LEARNINGS:
- Matrix stems identify machines rather than consumer components. Exact consumer-column binding avoids false inference and supports several machines per component.
- Valid full-path controls prevent unrelated gate failures from masquerading as RED.
- Canonical evidence shape requires exact headings and unbulleted Commands run and Summary labels.
- A repeated deliver currently fails because the delivered label already exists; no manual label edits performed.
- Frozen RED tests remain unchanged. Independent review and GREEN implementation remain pending.

## nd_contract
status: delivered

### evidence
- RED SHA: 0d52f43b393d961160aeea0db43b13a3fa5c284a.
- Exact producing test commands and 39 PASS / 27 intended FAIL / 0 skipped results recorded in canonical Implementation Evidence above.
- Controls 3 PASS; quality scan 1 file / 0 issues. Formatting repair reused recorded results.

### proof
- [x] AC 1: Exact consumer edge cases are specified and executed.
- [x] AC 2: Missing sibling declarations reproduced through real CLI and unit paths.
- [x] AC 3: Unique legacy controls pass; ambiguous fan-out and blank explicit owner demonstrate genuine RED.
- [x] AC 4: Repeat, conflict, malformed, rename and waiver cases are executed.
- [x] AC 5: Nine native CLI cases executed with four PASS and five intended FAIL. GREEN implementation remains pending.


## Implementation Evidence (DELIVERED — RED phase; formatting repair)
PROOF:

### CI/Test Results
- Commands run:
  - go test -count=1 -timeout=240s ./internal/gates -run 'Reads|EventContract|Consumer' -v
  - go test -count=1 -timeout=240s ./internal/gates -run '^TestReadsConsumerLegacySingleControl$|^TestReadsConsumerRepeatedLegacyStillHasOneOwner$|^TestReadsConsumerCLI$/legacy_control$' -v
  - pvg verify internal/gates/reads_consumer_test.go --include-tests --format text
  - git diff --check
- Summary: RED verification executed 66 leaf cases: 39 PASS, 27 intended behavioral FAIL, 0 skipped. New story inventory: 41 leaf cases, 14 PASS / 27 FAIL. Existing matched cases: 25 PASS / 0 FAIL. Real CLI: 9 cases, 4 PASS / 5 intended FAIL. Separate controls: 3 PASS / 0 FAIL. Quality scan PASS (1 file, 0 issues), diff check PASS.
- Coverage: statement percentage not measured; behavioral inventory and negative mutations are the measured evidence. No full suite or heavy preflight was run.
- Key output: unsafe missing_audit and missing_payments contracts returned CLI success with 0 blocking findings; valid distinct per-consumer read sets incorrectly rejected. These expected RED assertions expose unchanged production behavior. No compile/import/prerequisite failures in committed-SHA evidence.
- These are honestly reused outputs from the original committed-SHA runs, not new test executions during this formatting-only repair.

### Commit
- Branch: story/MAC-p8ce
- SHA: 0d52f43b393d961160aeea0db43b13a3fa5c284a
- Frozen file: internal/gates/reads_consumer_test.go (336 added lines). No production implementation. This repair makes no git changes.

### Wiring
- Native temporary CLI candidate built from this story worktree, not installed binary.
- CLI check with --gate g2,gx exercises real architecture participant validation plus consumer READS completeness; G2 alone tests unknown participant. No mocks, skips or services.

### pvg verify
- VERIFY: PASSED (1 files scanned, 0 issues).
- Supported installed syntax uses --format text; --format=text is rejected by the local coordination CLI.

### AC Verification
| AC # | Requirement | Test Location | Status |
|------|-------------|---------------|--------|
| 1 | Exact edge-local read sets | TestReadsConsumerExactEdgesPass and TestReadsConsumerCLI/distinct_sets in internal/gates/reads_consumer_test.go | RED specified and executed |
| 2 | Missing sibling and superset independence | TestReadsConsumerRejectsOwnershipMutations and ExactEdgesPass/strict_superset_is_local | RED behavior reproduced; narrowed-payload controls PASS |
| 3 | Unique legacy compatibility and ambiguity migration | LegacySingleControl, RepeatedLegacyStillHasOneOwner, AmbiguousLegacyNeedsMigration, ExplicitBlankOwnerCannotUseLegacyFallback | Controls PASS; ambiguity genuine RED |
| 4 | Repeats, conflicts, malformed declarations, renames, local waivers | ExactEdgesPass, RejectsOwnershipMutations, MalformedDeclarationsFailClosed | Positive/negative cases executed |
| 5 | Real CLI positive and negative mutations | TestReadsConsumerCLI | 4 PASS / 5 intended FAIL across 9 cases |

LEARNINGS:
- Matrix filename identifies a machine, not its architectural consumer; explicit consumer-column identity is the proposed contract, documented fully in prior append-only RED notes.
- Multiple machines may agree on the same consumer edge; conflicting exact sets must not silently union or override one another.
- A passing full-path legacy fixture distinguishes safety-assertion RED from unrelated infrastructure failures.
- Delivery metadata requires canonical headings in addition to substantive proof. This entry repairs formatting only; prior notes/history are preserved.
- GREEN implementation remains pending independent RED approval and must not alter frozen tests.

## nd_contract
status: delivered

### evidence
- RED SHA 0d52f43b393d961160aeea0db43b13a3fa5c284a.
- Commands run: go test -count=1 -timeout=240s ./internal/gates -run 'Reads|EventContract|Consumer' -v.
- Summary: 66 leaf cases, 39 PASS / 27 intended FAIL / 0 skipped; controls separately 3 PASS. No tests rerun for this metadata-only repair.
- pvg verify internal/gates/reads_consumer_test.go --include-tests --format text: PASS, 1 file / 0 issues.

### proof
- [x] AC 1: Exact event-consumer READS cases authored and executed through real Gx and combined G2/Gx CLI; genuine RED recorded.
- [x] AC 2: Missing either sibling declaration returns unsafe success today; unit and CLI mutations expose it. Strict-superset independence specified.
- [x] AC 3: Single-consumer and repeated-unique legacy controls pass; ambiguous fan-out and blank explicit owner produce genuine RED assertions.
- [x] AC 4: Repeated rows, same/different multiple-machine ownership, duplicates/conflicts, malformed fields, renames and local waivers covered.
- [x] AC 5: Nine real CLI cases executed, 4 PASS / 5 intended FAIL, no mocks/skips; RED complete, GREEN behavior still pending.


## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## RED delivery — MAC-p8ce
PROOF:
- Branch: story/MAC-p8ce. RED SHA: 0d52f43b393d961160aeea0db43b13a3fa5c284a. Subject contains tdd-red. Only internal/gates/reads_consumer_test.go added (336 lines); no production edits, no changes to prior tests. Freeze these bytes for GREEN after independent RED approval.
- Exact main command at that SHA: go test -count=1 -timeout=240s ./internal/gates -run 'Reads|EventContract|Consumer' -v
- Result: exit 1, 29 top-level groups / 66 leaf cases; 39 PASS, 27 intended assertion FAIL, 0 skipped. Existing matched cases: 25 PASS / 0 FAIL. New story cases: 41 leaves, 14 PASS / 27 FAIL. New CLI cases: 9 leaves, 4 PASS / 5 FAIL. No compile/import/runtime-prerequisite failure in this committed-SHA run.
- Exact passing-control command: go test -count=1 -timeout=240s ./internal/gates -run '^TestReadsConsumerLegacySingleControl$|^TestReadsConsumerRepeatedLegacyStillHasOneOwner$|^TestReadsConsumerCLI$/legacy_control$' -v
- Control result: exit 0, 3 leaf cases PASS / 0 FAIL / 0 skipped. Fixture passes both CheckC4 and CheckTraceability with zero errors/drift. CLI builds a native temporary candidate from this worktree via go build -o <t.TempDir>/machinery ./cmd/machinery; then real check <isolated-design> --gate g2,gx. No installed binary is used/replaced. Build and CLI processes have explicit context deadlines.
- Runtime assertion evidence: missing_payments and missing_audit fail with 'unsafe consumer READS contract accepted'; unchanged production counts both rows declared when only one declaration exists. CLI variants fail with 'unsafe missing_audit contract returned CLI success' and '0 blocking (ERROR/DRIFT) finding(s)'. Renaming to declared archive consumer and conflicting duplicate also incorrectly exit 0. Conversely distinct valid per-consumer payloads produce unrelated sibling-field drift and exit 1. These are behavior failures, not infrastructure RED.
- Self-check: pvg verify internal/gates/reads_consumer_test.go --include-tests --format text -> VERIFY: PASSED (1 files scanned, 0 issues). git diff --check exit 0; worktree clean after commit.
- Coverage percentage: not measured (no coverage instrumentation; no percentage claim). Test inventory and behavioral mutation outcomes above are the measured coverage. Heavy preflight/full suite deferred to final epic gate as explicitly requested.

Proposed input contract for independent PM review:
- A READS-declaring named-unit matrix table may carry an explicit consumer column. Each declaring row names exactly one architectural event-contract participant, matched by exact cleaned participant ID (backticks/parenthesized annotations do not rename the ID). Matrix filename is NOT consumer ownership: existing Payment.matrix.md belongs to paymentSvc, and multiple machines can belong to one component.
- Missing consumer column is legacy syntax, allowed only when the named event has one distinct consumer across all event-contract rows. Repeated identical rows do not create a second owner. Fan-out with unscoped legacy declarations, even mixed with explicit declarations, must fail with guidance to declare consumer ownership. Present but blank owner is invalid, not legacy fallback.
- READS is an exact set for its event-consumer edge. Identical declarations in two machines sharing that consumer are compatible; conflicting exact sets are ambiguous and rejected, never first-wins or silently unioned. Duplicate members, multiple READS overrides in one cell, empty/malformed declarations and multiple-owner cells fail closed.
- Each contract row's payload independently satisfies its owner's set. A strict-superset consumer does not widen its sibling's requirements. Repeated same-edge rows are each checked. Unknown payload fields fail whole-token reconciliation. This does not add a new general payload-to-domain-schema validator.
- A reasoned no-reads consumer waiver is row-local; it never transfers a matrix declaration or waiver to another consumer. Empty reasons remain errors.
- G2 continues to own participant/cell resolution; Gx owns READS completeness. Real combined --gate g2,gx cases prove the joint guarantee; unknown participant is separately exercised through --gate g2. No requirement to duplicate READS findings in G2 (dispatcher explicitly confirmed).

AC verification (RED specification, not completed GREEN behavior):
1. Exact per-consumer fields: TestReadsConsumerExactEdgesPass/distinct_sets, strict_superset_is_local, annotation_does_not_change_participant; CLI distinct_sets. Genuine RED from cross-consumer leakage.
2. Sibling missing declaration/noninterference: TestReadsConsumerRejectsOwnershipMutations/missing_payments, missing_audit, payload_narrowed_only_for_audit/payments; strict_superset_is_local. Missing declarations fail test assertions on unchanged production; existing missing-field protection passes.
3. Legacy compatibility/migration: LegacySingleControl and RepeatedLegacyStillHasOneOwner PASS; AmbiguousLegacyNeedsMigration both mixed cases FAIL as intended; ExplicitBlankOwnerCannotUseLegacyFallback FAIL as intended.
4. Repeats/conflicts/malformed/rename/waivers: ExactEdgesPass repeated identical edge and two machines agreeing PASS; RejectsOwnershipMutations conflicting rows/overrides, two-machine disagreement, near-match/renamed owners, waiver sibling; MalformedDeclarationsFailClosed eight cases. Diagnostics re-run twice and compared for deterministic order after blocked outcomes.
5. Real CLI: TestReadsConsumerCLI 9 cases covers positive control/distinct sets, each removed declaration, each narrowed payload, renamed known owner, conflicting duplicate and G2 unknown participant. 4 PASS / 5 genuine RED.

LEARNINGS:
- Matrix stems identify machines, not consumer components; a basename shortcut would encode false ownership. An explicit consumer column avoids that inference and supports multiple machines per consumer.
- Full-path fixtures need lifecycle semantics, placement, invariant enforcement and a coherent architecture contract; otherwise unrelated gate failures can masquerade as RED. The native CLI legacy control proves this fixture is viable.
- The existing implementation both under-enforces (missing sibling declarations pass) and over-enforces (legitimate distinct read sets fail).
- The installed pvg verify parser rejects --format=text despite skill examples; --format text works. This is local coordination tooling only; no Machinery dependency was introduced.
- An initial uncommitted compile typo and overly case-specific CLI message assertion were corrected before the tdd-red commit; neither is counted as RED evidence.

## nd_contract
status: delivered

### evidence
- RED 0d52f43b393d961160aeea0db43b13a3fa5c284a, only frozen test file.
- Targeted run 39 PASS / 27 intended FAIL / 0 skipped over 66 leaf cases; controls separately 3 PASS.
- pvg verify 1 file, 0 issues; native CLI integration, no mocks/services/installed binary changes.

### proof
- [x] AC #1: RED exact-edge and distinct payload cases authored and executed.
- [x] AC #2: RED missing-sibling mutations reproduced through unit and CLI paths.
- [x] AC #3: passing unique-legacy controls and failing ambiguous migration cases recorded.
- [x] AC #4: repeat/conflict/malformed/rename/waiver negative matrix executed.
- [x] AC #5: real G2/Gx CLI controls and mutations executed; GREEN implementation remains pending.


## History
- 2026-09-05T19:35:08Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:15Z dep_added: blocks MAC-ou97
- 2026-09-05T19:47:31Z status: open -> in_progress
- 2026-09-05T19:47:32Z claimed by dev-MAC-p8ce
- 2026-09-05T19:56:26Z status: in_progress -> in_progress
- 2026-09-05T19:58:20Z status: in_progress -> in_progress
- 2026-09-05T20:07:41Z status: in_progress -> open
- 2026-09-05T22:43:57Z status: open -> in_progress
- 2026-09-05T22:43:57Z auto-follows: linked to predecessor MAC-olrx
- 2026-09-05T22:43:57Z claimed by dev-MAC-p8ce
- 2026-09-05T22:54:50Z status: in_progress -> in_progress

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-gcrr]], [[MAC-ou97]]
- Follows: [[MAC-olrx]]

## Comments

### 2026-09-05T22:55:07Z ramirosalas
## Implementation Evidence

PROOF:

### CI/Test Results

Commands run:
- All shell/test commands used /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-p8ce on story/MAC-p8ce.
- go test -count=1 -timeout=5m ./internal/gates -run 'Reads|EventContract|Consumer' -json -coverprofile=/tmp/MAC-p8ce-green-final.cover
- go test -count=1 -timeout=5m ./internal/gates -run 'G2|C4|Traceability|EventWiring|Reads|EventContract|Consumer|Payload' -json -coverprofile=/tmp/MAC-p8ce-green-blast.cover
- go test -count=1 -timeout=5m ./internal/gates -run '^TestEventCells' -json
- go test -timeout=5m ./internal/gates -list 'G2|C4|Traceability|EventWiring|Reads|EventContract|Consumer|Payload'
- go tool cover -func=/tmp/MAC-p8ce-green-final.cover
- go tool cover -func=/tmp/MAC-p8ce-green-blast.cover
- pvg verify internal/gates/readscomplete.go internal/gates/reads_consumer_test.go internal/gates/reads_consumer_supplemental_test.go README.md skills/machinery/references/c4-standalone.md skills/machinery/references/build-md-template.md --include-tests --format text
- pvg story verify-tdd --base 497419ab4512fcff765cd5feb27aed4c67b5608d
- git diff --exit-code 0d52f43b -- internal/gates/reads_consumer_test.go
- git diff --check 497419a..HEAD
- git diff --numstat 497419a..HEAD
- git status --porcelain

Summary: final targeted run PASS 78 leaves / 0 FAIL / 0 SKIP (41 frozen story cases including 9 actual CLI cases; 12 new supplemental cases; 25 existing cases), exit 0, 4.463s. Broader downstream run PASS 134 leaves / 0 FAIL / 0 SKIP, exit 0, 3.066s. Separate G2 EventCells run PASS 15 leaves / 0 FAIL / 0 SKIP. The broader run contains the targeted cases; combined breadth is 149 distinct passing leaves, not an additive claim over repeated executions. JSON reducers emitted every terminal leaf result and every OutputType=error; final errors arrays were empty. No mocks, environmental skips, missing prerequisites, or service dependency.

Coverage: downstream package statement coverage 24.9%; readscomplete.go covered 138/141 statements (97.87%) from the targeted profile. Function coverage: readsCompleteArmed 100%, armedReadsEvents 100%, collectConsumerReads 92.3%, parseConsumerReadSet 100%, bindConsumerReads 100%, checkReadsComplete 100%. Targeted runtime printed 19.4% package coverage while go tool cover rounded its aggregate to 19.3%; source-file numerator/denominator above is measured directly from the profile. Uncovered collector statements are diagnostic/fallback paths, not an assertion of complete path coverage.

Native inventory:
- Frozen tests: LegacySingleControl (1), RepeatedLegacyStillHasOneOwner (1), ExactEdgesPass (6), RejectsOwnershipMutations (13), AmbiguousLegacyNeedsMigration (2), MalformedDeclarationsFailClosed (8), ExplicitBlankOwnerCannotUseLegacyFallback (1), CLI (9): all 41 PASS unchanged.
- CLI cases: legacy_control, distinct_sets, missing_audit, missing_payments, audit_payload_narrowed, payments_payload_narrowed, renamed_known_consumer, conflicting_duplicate, unknown_participant_G2: all PASS. Tests compile ./cmd/machinery to a real temporary binary and execute check <temporary-design> --gate g2,gx; unknown-participant case uses g2. Both valid fixtures and intended blocking exits are asserted.
- Supplemental: reordered/annotated owner, escaped pipe, per-table column positions, unique legacy prose, order-independent exact sets, duplicate owner columns, short explicit row, malformed closing brace, unclosed waiver, duplicate waivers, unknown extra owner, nonconsumer header: 12 PASS.
- Downstream run includes CheckC4 and CheckTraceability against all three repository examples, event wiring/waivers, interface contracts, allow graph, unarmed READS behavior and warn-tier handoff. Separate EventCells inventory covers participant resolution, externals, nested annotations, every table, column answers and dedupe.
- Per user scope, full package/repository suite and scripts/preflight.sh were not run. No installed binary, installed skills/plugins, Docker container, remote git or GitHub state changed.

Intermediate results and errors:
- First GREEN at 6131abf: original 66 leaves all PASS, including frozen 41 and actual CLI 9.
- New supplemental tests initially exposed three in-scope unsafe accepts at that intermediate implementation: extra closing brace, unclosed waiver beside a valid declaration, duplicate waivers hiding an empty reason. That run was 9 PASS / 3 FAIL / 0 SKIP. All three were corrected in owned source and pass at final SHA. Frozen RED was never edited.
- pvg issues comment --help returned exit 1 with 'flag: help requested'; documentation output was read, not counted as a product failure. pvg story deliver --help printed a misleading OK; subsequent story read confirmed status in_progress with hard-tdd/red-approved and no delivered label or mutation. Actual story delivery is performed once below.
- No unresolved product test failures or warnings encountered.

### Commit

Branch: story/MAC-p8ce
RED SHA: 0d52f43b393d961160aeea0db43b13a3fa5c284a
GREEN SHA: 337b9cec17a3f551a7564bed9394045b36043976
Implementation commit: 6131abf; supplemental/strict-malformed/docs commit: 337b9ce.
Frozen RED file byte-identical to approved SHA; local-base verify-tdd PASS: checked 3 commits, skipped 0 merges, no unauthorized test edits.
Whole story diff from 497419a: 6 files, 619 insertions / 33 deletions = 652 changed LOC, including frozen 336-line RED addition. Worktree clean; diff check PASS. No rebase or merge performed.

### Wiring

Existing production paths remain checkEventCells via CheckC4 at gates.go:971 and checkReadsComplete via CheckTraceability at gates.go:1992. The new binder and parser are called directly from the armed check; they use existing readDesignFile, strictSortedGlob, ir.ParseMdTables, ir.FindCol and ir.CleanCell. Product remains standalone Go with no Paivot runtime dependency. Ownership is exact cleaned event-contract consumer, never machine basename. Legacy resolution counts distinct consumers, including waived siblings; explicit sibling coverage never licenses ambiguous legacy fallback.

### pvg verify

VERIFY: PASSED (3 files scanned, 0 issues). All six changed paths were supplied explicitly; this tool scanned the three Go files and did not claim Markdown validation.
Doc freshness manually reviewed: dispatcher explicitly authorized only READS passages in README.md, skills/machinery/references/c4-standalone.md and skills/machinery/references/build-md-template.md. All three now document exact participant binding, unique legacy migration, sibling isolation and malformed/conflicting declarations while preserving unarmed Gd semantics. No installed skill/plugin path or SKILL.md edited.
Codebase-memory Verify used root project Users-ramirosalas-workspace-machinery, generation 2026-09-05T20:28:41Z; relevant graph search and both-direction trace found sole checkReadsComplete caller CheckTraceability. Coverage metadata matched original source; worktree-only tests were missing from the root graph. Every consumed/changed path was read directly in the worktree as the changed/unindexed fallback. Graph freshness is not claimed for this branch.

### AC Verification

| AC # | Requirement | Code Location | Test Location | Status |
|------|-------------|---------------|---------------|--------|
| 1 | Exact per-consumer declaration and own payload; joint G2/Gx enforcement | readscomplete.go collectConsumerReads, bindConsumerReads, checkReadsComplete; existing gates.go G2/Gx call sites | frozen ExactEdgesPass/distinct_sets, annotation_does_not_change_participant; CLI/distinct_sets and unknown_participant_G2 | PASS |
| 2 | Missing siblings cannot be masked; strict supersets remain local | readsEdge key, per-edge canonical read set, row payload loop | frozen RejectsOwnershipMutations missing owners/reassignment/independently narrowed payloads; ExactEdgesPass/strict_superset_is_local | PASS |
| 3 | Unique legacy retained; fan-out gives migration guidance | bindConsumerReads distinct-owner resolution and explicit-column errors | LegacySingleControl, RepeatedLegacyStillHasOneOwner, both AmbiguousLegacyNeedsMigration cases, ExplicitBlankOwnerCannotUseLegacyFallback | PASS |
| 4 | Repeats, conflicts, unknown fields, renamed owners, malformed values and local waivers deterministic | canonical sorted sets, strict parseConsumerReadSet, exact owner lookup, row-local waiver validation | frozen repeats/multimachine agreements/conflicts/near-match/rename/malformed/waiver cases; 12 supplemental authoring-boundary cases | PASS |
| 5 | Positive distinct sets and all prescribed negative mutations through real CLI | same G2/Gx production path | all nine frozen TestReadsConsumerCLI leaves PASS with correct success/blocking exit and intended findings | PASS |

LEARNINGS:
- Consumer identity is architectural; machine filenames cannot safely establish it. Table-local columns preserve identity across column order changes and multiple tables.
- Comparing canonical complete sets prevents both sibling field leakage and accidental unions of conflicting declarations across machines.
- Malformed overrides must remain visible even when a valid declaration would otherwise satisfy the edge; waiver syntax needs the same strictness.
- The existing markdown parser preserves escaped pipes, allowing exact ownership without a second cell grammar.
- Full-path positive controls and native leaf inventory distinguish semantic gate rejection from unrelated fixture or process failures.

## nd_contract
status: delivered

### evidence
- GREEN 337b9cec17a3f551a7564bed9394045b36043976; frozen RED 0d52f43b393d961160aeea0db43b13a3fa5c284a unchanged.
- Targeted 78 PASS; broader G2/Gx 134 PASS; separate EventCells 15 PASS; all 0 FAIL/0 SKIP. Actual CLI 9 PASS.
- Source statement coverage 138/141 (97.87%), downstream package 24.9%; pvg verify 3 code files/0 issues; verify-tdd local base PASS.

### proof
- [x] AC #1: exact event-consumer binding and own payload validated through production G2/Gx.
- [x] AC #2: sibling absence and local strict-superset semantics independently verified.
- [x] AC #3: unique legacy accepted and fan-out rejected with explicit migration guidance.
- [x] AC #4: deterministic repeat/conflict/malformed/rename/waiver behavior verified.
- [x] AC #5: all nine real CLI control/mutation cases pass unchanged.

Delivery verification: pvg story deliver MAC-p8ce executed once, then pvg story verify-delivery MAC-p8ce passed 9/9. This terminal comment repeats the GREEN evidence because older RED comments render after the Notes section; append-only history is preserved.

