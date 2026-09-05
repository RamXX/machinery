---
id: MAC-p8ce
title: "Bind consumer READS to each event edge"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd, delivered]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:26Z
created_by: ramirosalas
updated_at: 2026-09-05T19:58:19Z
content_hash: "sha256:e713b6e9f7c9b5e00977926f54590f43eb7d2eeb8ea19dff0b5219a3ac2ba4c1"
blocks: [MAC-gcrr, MAC-ou97]
assignee: dev-MAC-p8ce
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

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-gcrr]], [[MAC-ou97]]

## Comments
