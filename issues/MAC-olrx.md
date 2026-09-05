---
id: MAC-olrx
title: "Preserve oracle ownership through decomposition"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd, delivered]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:26Z
created_by: ramirosalas
updated_at: 2026-09-05T19:58:17Z
content_hash: "sha256:f7c19f1bcbd30ec165dbac53c2f2ebeeed39fc13ab62a59175b1e16c22f1a710"
blocks: [MAC-vx24, MAC-ou97]
assignee: dev-MAC-olrx
---

## Description
## USER INTENT
Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.

## Context (Embedded)
Assessment F6 parent: selectInSnapshot removes gt for machine-less decomposed parents even with Policy/Isolation oracle obligations; explicit Gt correctly reports missing IDs. NEXT3 clauseSet drops machine owner and Gt/Gd pools guard names across all machines.

## Ownership
Own only these paths and directly associated tests: internal/gates/suite.go, internal/gates/clauses.go, internal/gates/clausecov.go, internal/gates/obligation_ownership_test.go. You are not alone in this codebase; preserve other edits and coordinate any shared-file changes with dispatcher.

## Boundary Map
PRODUCES:
- internal/gates/suite.go -> hardened behavior and regression proof
- internal/gates/clauses.go -> hardened behavior and regression proof
- internal/gates/clausecov.go -> hardened behavior and regression proof
- internal/gates/obligation_ownership_test.go -> hardened behavior and regression proof
CONSUMES:
- Existing Machinery source interfaces.
  spec: gates.Select(design, gateList, impl string) (Selection, error); gates.CheckOracleCoverage(design, impl string) *Gate; collectClauseDecls(g *Gate, design string) []clauseSet; checkClauseDrift(g *Gate, design string); checkClauseCoverage(g *Gate, design string, corpus testCorpusData)

### Story Acceptance Criteria
1. Default selection with --impl retains Gt for machine-less decomposed parents that own relational oracle obligations; absence of machines cannot erase Policy/Isolation coverage. Truly obligation-free parents remain valid with honest zero-obligation reporting.
2. CLAUSES declarations and corresponding drift/coverage rows key on owning machine plus guard; identical guard names in different machines are independent. Declaring or satisfying one never arms or satisfies the other.
3. Positive same-machine clauses still require every falsifying clause, reject duplicate/conflicting/missing declarations and preserve stable IDs; malformed/orphan owner identity fails clearly.
4. Negative full-path tests cover parent missing tests, same-name guards with distinct clauses, one machine omitted, mismatched owner and removed oracle; positive controls show correctly covered parents and two valid independent machines.
5. Run actual gate selection plus Gt/Gd results, not only helper structures. Document precise ownership in relevant diagnostics without relying on prose to enforce it.

## Testing Requirements
- Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
- Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
- Commands: go test ./internal/gates -run 'Clause|Parent|Selection|Obligation'; temporary complete decomposed parent designs through machinery check default selection and explicit gates.
- No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.

## OUT OF SCOPE
- Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
- Global heavy preflight, main merge and local release binary belong to final epic gate.

## DIFF BUDGET
- ~4-6 files, under 800 changed LOC; material overrun requires PM investigation, not weakened requirements.

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
## Implementation Evidence (RED DELIVERED)

PROOF:

### CI/Test Results
Commands run:
- go test -count=1 -timeout=5m -cover ./internal/gates -run 'Clause|Parent|Selection|Obligation' -v
- go test -count=1 -timeout=5m ./internal/gates -run 'TestObligationClausesSingleMachinePositiveControl$|TestObligationParentRealCLI$/obligation-free-parent-control$' -v
- pvg verify internal/gates/obligation_ownership_test.go --format text --include-tests
- git diff --check

Summary: RED EXPECTED — targeted 44 top-level tests: 33 PASS, 11 intended behavioral FAIL; 69 leaf cases: 48 PASS, 21 intended behavioral FAIL, 0 skipped. New RED cases: 2 passing controls and 21 failing leaves. All 46 existing targeted leaves pass. Separate positive-control replay: 2 PASS, 0 FAIL, 0 skipped. Scanner PASS (1 file, 0 issues). No compile/import/infrastructure failures in final runs.
Coverage: 27.7% internal/gates statements in targeted final run; excludes separately built CLI subprocess. Not full-suite coverage.
Commit SHA: 27373f421640f644a98be89f38ce112c30f55a85 on story/MAC-olrx. Test-only immutable tdd-red commit; 463 lines added.
Expected failure examples: actual default CLI returns platform-green after required parent Policy/Isolation ID deletion; owner-local clauses count 9 instead of 5; malformed/orphan declarations silently pass; owner+guard diagnostics missing. Detailed commands, outputs, complete test inventory and actual-integration design are in the preceding append-only RED evidence.

### AC Verification
| AC # | Requirement | Test Location | Status |
|------|-------------|---------------|--------|
| 1 | Parent selection retains relational obligations and supports honest zero obligations | obligation_ownership_test.go: TestObligationParentSelectionRetainsRelationalCoverage; TestObligationParentRealCLI | RED reproduced; zero-obligation control PASS |
| 2 | Independent machine+guard coverage and drift | TestObligationClausesIndependentMachines; TestObligationClausesUndeclaredSiblingIsNotArmed; TestObligationClausesLocalDriftNamesOwner | RED reproduced |
| 3 | Every clause, declaration validation, stable IDs and owner validity | SingleMachinePositiveControl; EveryLocalClauseRequired; InvalidDeclarationRejectedByBothGates; DuplicateDeclarationRejected; OwnerCannotResolveThroughSibling | RED reproduced; single-machine control PASS; oracle bytes unchanged |
| 4 | Full-path omitted machine, wrong owner, removed oracle, parent negative and independent positives | MissingLocalCaseNamesOwner; OmittedMachineTests; OwnerCannotResolveThroughSibling; ParentRealCLI | RED reproduced across actual files, generators, Gt/Gd and CLI |
| 5 | Real selection, actual Gt/Gd results and precise owner diagnostics | selectedObligationGates calls SelectRunAndNote; requireOwnerError checks same diagnostic; real CLI built and executed | RED reproduced; no mocks |

LEARNINGS:
- G4 positive import fixture must be real and clean to prevent unrelated errors masking missing Gt.
- Machine matrix stems provide unambiguous local ownership without inventing narrative syntax.
- Generated stable IDs and immutable oracle comparisons avoid assumptions about owner tags.

## nd_contract
status: delivered

### evidence
- RED commit 27373f421640f644a98be89f38ce112c30f55a85.
- Final targeted command produced 48 PASS / 21 intended RED FAIL leaf cases; positive replay 2 PASS / 0 FAIL.
- No production changes, full preflight, GitHub mutation or installed binary replacement.

### proof
- [x] AC #1: RED tests authored; parent selection/CLI false green reproduced; zero control passes.
- [x] AC #2: RED owner-local Gt/Gd regressions authored and reproduced.
- [x] AC #3: RED every-clause, malformed/duplicate/owner validation and stable-ID tests authored.
- [x] AC #4: RED real filesystem/public gate/CLI negative and positive inventory authored.
- [x] AC #5: RED actual selection and machine+guard diagnostic outcomes asserted.
- [ ] GREEN implementation: not performed; fresh independent implementer and PM acceptance required.

## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## RED Delivery — MAC-olrx

PROOF:
- Phase: RED ONLY. Production remains unchanged; acceptance means the test specification is ready for independent RED review, NOT that the behavior is fixed.
- Branch: story/MAC-olrx. Frozen RED SHA: 27373f421640f644a98be89f38ce112c30f55a85.
- Changed files: internal/gates/obligation_ownership_test.go (463 added lines; tests and fixture-building helpers only).
- Exact final command from the story worktree: go test -count=1 -timeout=5m -cover ./internal/gates -run 'Clause|Parent|Selection|Obligation' -v
- Result at frozen SHA: exit 1; 44 top-level tests (33 PASS, 11 intended RED FAIL); 69 leaf test cases (48 PASS, 21 intended RED FAIL), 0 skipped. New tests comprise 12 top-level / 23 leaf cases: 2 passing leaves and 21 intended failing leaves. All 32 existing top-level / 46 existing leaf tests in this targeted selection passed.
- Coverage: 27.7% of statements in internal/gates for this targeted selection. This is package-wide targeted-run coverage, not a claim of full package coverage; the subprocess-built CLI is not included in that percentage.
- Independent positive-control command at the same SHA: go test -count=1 -timeout=5m ./internal/gates -run 'TestObligationClausesSingleMachinePositiveControl$|TestObligationParentRealCLI$/obligation-free-parent-control$' -v
- Positive-control result: exit 0; 2 top-level tests, 2 leaf cases PASS, 0 FAIL, 0 skipped, 1.707s.
- Integration uses actual SelectRunAndNote, actual Gt/Gd, production oracle.Generate, real filesystem snapshots, pack.LoadDecomposition + CheckPack on copied complete checkout-split parent/children, and a real binary built by go build -o <temp>/machinery ./cmd/machinery. It runs machinery check <temp-parent> --impl <temp-impl> both default and --gate gt. Positive parent implementation fixtures compile and execute via go test -count=1 ./... with only a real local Go dependency.
- Frozen targeted final run elapsed 8.584s. No full suite or preflight run, as requested. No live binary/plugin/skill changes, installs, GitHub mutations, pushes or syncs.
- Quality command: pvg verify internal/gates/obligation_ownership_test.go --format text --include-tests => VERIFY: PASSED (1 files scanned, 0 issues). git diff --check clean; worktree clean after commit.

Observed intended RED evidence:
1. Independent Alpha(2 clauses)/Beta(3 clauses) with identical guardReady: Gt wrongly demands ALPH-1d4d8ec and counts 9 clause IDs instead of 5; Gd wrongly calls each owner's full vocabulary a partial enumeration of the sibling's.
2. A declaration only on Alpha incorrectly arms Beta's otherwise undeclared same-name guard and its matrix narrative.
3. Missing individual owner-local clause and omitted-machine coverage errors lack a precise machine+guard identity; tests also prove the actual generated stable ID remains in the error and oracle bytes remain unchanged.
4. Duplicate/conflicting/empty/malformed CLAUSES declarations, orphan/empty matrix owner, Alpha declaring Beta-only guardOther, and removed Alpha oracle are not rejected by both Gt and Gd with machine+guard diagnostics. These tests deliberately keep the sibling valid so the missing owner's obligation cannot resolve through it.
5. For both Policy.oracle.md and Isolation.oracle.md, default selection erases gt. Real CLI with one required decision ID removed returns exit 0 and platform-green; explicit --gate gt rejects with the oracle and missing stable ID. Covered parent and obligation-free parent controls pass. G4, packs and all other default gates are genuinely clean in this fixture, so an unrelated setup error cannot supply RED.
6. Retired and partial local drift remains required, while sibling vocabulary must not contaminate the diagnostic.

AC-to-test proof:
| AC | Frozen test evidence | RED status |
| 1 | TestObligationParentSelectionRetainsRelationalCoverage; TestObligationParentRealCLI (Policy/Isolation and obligation-free-parent-control) | expected selection/default-CLI assertion failures; positive and zero-obligation controls pass |
| 2 | TestObligationClausesIndependentMachines; TestObligationClausesUndeclaredSiblingIsNotArmed; TestObligationClausesLocalDriftNamesOwner | expected owner-crossing Gt/Gd assertion failures |
| 3 | TestObligationClausesSingleMachinePositiveControl; TestObligationClausesEveryLocalClauseRequired; TestObligationClausesInvalidDeclarationRejectedByBothGates; TestObligationClausesDuplicateDeclarationRejected; TestObligationClausesOwnerCannotResolveThroughSibling | positive control passes; precise validation/owner diagnostic assertions fail as intended; stable oracle bytes checked unchanged |
| 4 | All parent CLI negative cases; MissingLocalCaseNamesOwner; OmittedMachineTests; OwnerCannotResolveThroughSibling | actual file/gate paths, not mocked helper structures; two complete independent machines covered |
| 5 | selectedObligationGates routes through SelectRunAndNote, actual gate results; requireOwnerError asserts owner+guard within the same blocking diagnostic; actual CLI parent run | intended false-green CLI and owner diagnostics failures |

LEARNINGS:
- Parent regression fixtures must include a genuinely checked allowed G4 import edge; otherwise unrelated empty-import errors can disguise the missing-Gt bug. The final fixture uses a local JSON-encoding dependency and compiles/runs its Go tests.
- Matrix filename ownership is sufficient to make these drift cases unambiguous; narrative is appended to each real owning matrix, without inventing a new prose-reference syntax.
- Production-generated IDs avoid assuming tag-to-owner identity. Machines can share a guard name while owning independent row IDs, suffix counts and vocabularies.
- The installed pvg parser accepts --format text, not the skill's --format=text spelling; the corrected quality command passed. Coordination tooling appears only in development notes, never in Machinery product/test dependencies.

## nd_contract
status: delivered

### evidence
- RED SHA 27373f421640f644a98be89f38ce112c30f55a85; test-only commit with tdd-red subject.
- Targeted final run: 48 passing / 21 intended failing leaf cases, 0 skipped; 27.7% targeted package statement coverage.
- Separate positive replay: 2 passing / 0 failing leaf cases.
- pvg verify 1 test file / 0 issues; no source or installed-tool mutations.

### proof
- [x] AC #1: RED parent selection/default CLI and explicit Gt tests, plus passing zero-obligation control authored.
- [x] AC #2: RED independent machine+guard coverage and drift tests authored.
- [x] AC #3: RED declaration validation, every local suffix, stable-ID preservation and positive control authored.
- [x] AC #4: RED full-path negative/positive test inventory authored with valid recursive parent and generated machine fixtures.
- [x] AC #5: RED actual gate selection/results and owner-specific diagnostics authored.
- [ ] GREEN: implementation and independent acceptance remain required; frozen RED tests must not be weakened.

## History
- 2026-09-05T19:35:07Z dep_added: blocks MAC-vx24
- 2026-09-05T19:36:14Z dep_added: blocks MAC-ou97
- 2026-09-05T19:47:31Z status: open -> in_progress
- 2026-09-05T19:47:31Z claimed by dev-MAC-olrx
- 2026-09-05T19:57:39Z status: in_progress -> in_progress
- 2026-09-05T19:58:17Z status: in_progress -> in_progress

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-ou97]]

## Comments
