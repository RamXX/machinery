---
id: MAC-olrx
title: "Preserve oracle ownership through decomposition"
status: closed
priority: 0
type: bug
labels: [hard-tdd, red-approved, accepted]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:26Z
created_by: ramirosalas
updated_at: 2026-09-05T20:28:15Z
content_hash: "sha256:b7646d6e761484710efe85729ddba0e749b50aada0bb526dff9713578b652249"
assignee: dev-MAC-olrx
closed_at: 2026-09-05T20:27:03Z
close_reason: "All five ACs independently verified; immutable RED72/72 pass, owner-local and parent obligations preserved, DOCS_STALE repaired at6ba45625."
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
LOCAL MERGE COMPLETE: accepted closed state independently verified, then merged with --no-ff to epic/MAC-ui8a at f24b2df3cb1e1521f97b406a7516f72bb7bc7890. Accepted GREEN6ba45625fd86307dd7fa996c70ff54d74ea4e0bd is an ancestor. Clean retained developer checkout and merged story branch removed; all commits retained in epic history; handles cleared. No remote fetch/sync/push, no main merge or installed artifacts changed. Targeted merged-epic replay started; full preflight remains deferred.

## nd_contract
status: accepted

### evidence
- PM closeout applied via pvg story accept on 2026-09-05.

### proof
- [x] Story closed after accepted label was applied.


## PM GREEN Re-review
ACCEPTED [2026-09-05]: DOCS_STALE repaired; all five ACs meet the evidence bar at final GREEN 6ba45625fd86307dd7fa996c70ff54d74ea4e0bd.

Re-read current story and delivery evidence; inspected exact diff from prior reviewed GREEN8168b3d1d1599fd7860dd5afda5d7e55f49113aa. Only docs/claude-plugin.md changed (11 additions, 5 deletions). The replacement paragraph accurately distinguishes parent-owned Policy/Isolation obligations, activation retained by source annotations after generated-oracle deletion, missing-output findings in Gp/Gn, and the CLI-skip/hook-empty-count distinction only for genuinely obligation-free parents. Compared directly with suite.go:335-359 and the current actual gate tests. No stubs, stale changed-behavior documentation or out-of-scope changes remain in this bounded review.

Fresh independent synchronous verification at 6ba45625fd86307dd7fa996c70ff54d74ea4e0bd:
- pvg story verify-tdd --base epic/MAC-ui8a --json: 5 commits checked, zero violations; frozen RED27373f421640f644a98be89f38ce112c30f55a85 test bytes unchanged by direct diff.
- go test -count=1 -timeout=5m -cover ./internal/gates -run 'Clause|Parent|Selection|Obligation' -v: 46 top-level / 72 leaf cases PASS, 0 FAIL, 0 skipped, 9.000s, 37.5% targeted package statement coverage. All original RED assertions pass unchanged, including each required negative mutation and real CLI positives/negatives.
- git diff --check epic/MAC-ui8a...HEAD clean; retained worktree clean; exact HEAD verified.

Prior independent proof remains applicable to unchanged production/test/fixture bytes, and is explicitly not relabeled as a fresh run at the docs SHA: 9/9 CLI golden leaves passed at8168b3d in3.286s; affected suite103PASS/0FAIL/1native filesystem skip at52.5% in5.442s. The native skip was disclosed, not counted as PM execution; reviewed developer case-sensitive APFS replay showed4/4 pass with bounded task-volume teardown. No new required test skips exist.

AC verification:
1. Default parent relational obligations retained; required ID deletion and generated-oracle deletion remain blocked; covered and genuinely empty controls pass. Public guide now accurately matches.
2. Shared validator keys owner plus guard; fresh Alpha/Beta independent vocabulary/count/undeclared-sibling and drift cases pass without contamination.
3. Every local falsifying suffix, duplicate/conflicting/malformed/orphan declaration and missing owner oracle is exercised; owner+guard diagnostics and stable generated bytes preserved.
4. Actual filesystem/generator/gate/CLI positive and negative integration paths execute without mocks or skips.
5. Actual selected gate results and precise ownership diagnostics verified; mandatory public documentation gap resolved.

No test repair authorized or performed. No source edits by PM, preflight, push/sync/GitHub mutation, live binary/plugin/skill replacement or product Paivot dependency. Full preflight and main merge remain dispatcher final-epic responsibilities.

## nd_contract
status: accepted

### evidence
- Final GREEN6ba45625fd86307dd7fa996c70ff54d74ea4e0bd, frozen RED27373f421640f644a98be89f38ce112c30f55a85; fresh72/72pass0skip, five-commit TDD check clean; docs-only correction verified.

### proof
- [x] AC #1: parent activation and zero-obligation behavior verified.
- [x] AC #2: independent owner-local coverage and drift verified.
- [x] AC #3: declaration validation and stable IDs verified.
- [x] AC #4: actual positive/negative integration executed.
- [x] AC #5: selected gates, diagnostics and accurate public documentation verified.

## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## Implementation Evidence (GREEN REWORK: DOCS_STALE)

PROOF:

### CI/Test Results
Commands run:
- go test -count=1 -timeout=5m ./internal/gates -run '^TestObligationParent' -v
- pvg verify docs/claude-plugin.md --format text
- pvg story verify-tdd --base epic/MAC-ui8a --json
- git diff --check epic/MAC-ui8a...HEAD
- git diff --stat 8168b3d1d1599fd7860dd5afda5d7e55f49113aa..HEAD
- git diff --exit-code 27373f421640f644a98be89f38ce112c30f55a85 -- internal/gates/obligation_ownership_test.go
- git status --short

Summary: Documentation-only repair at 6ba45625fd86307dd7fa996c70ff54d74ea4e0bd. Fresh parent regression run: 3 top-level / 7 leaf cases PASS, 0 FAIL, 0 skipped, 7.800s, including actual default and explicit CLI checks, source annotation activation after oracle deletion, covered parents and the obligation-free control. Coverage was not measured in this docs-only rework run. Historical full targeted 72/72 and golden 9/9 passes were produced and independently reviewed at 8168b3d1d1599fd7860dd5afda5d7e55f49113aa; they are not represented as new runs at this documentation SHA.

Documentation repair: docs/claude-plugin.md:88 now explains that machine-less parents retain Gt with --impl when they own Policy/Isolation source annotations or committed decision oracles. Deleting generated output preserves source activation and Gp/Gn report the missing required output. The hook runs Gt when impl is configured. The existing CLI skip versus hook zero-machine count distinction is limited to truly obligation-free parents. Zero machines no longer implies zero relational obligations.

Fresh diff from the previous delivery changes only docs/claude-plugin.md: 11 additions / 5 deletions. No production, test, fixture, golden, or configuration changes. The frozen RED file is byte-identical to 27373f421640f644a98be89f38ce112c30f55a85; TDD verification checked 5 commits with zero violations. Worktree clean, git diff --check clean. pvg verify returned PASS with 0 files scanned / 0 issues because its source scanner does not inspect Markdown; this is not claimed as automated prose validation. The exact changed paragraph was manually compared with suite.go source-annotation/oracle activation and Gp/Gn missing-output diagnostics.

### Commit
- Branch: story/MAC-olrx.
- Frozen RED: 27373f421640f644a98be89f38ce112c30f55a85.
- Behavioral GREEN: 8168b3d1d1599fd7860dd5afda5d7e55f49113aa.
- Documentation rework / final GREEN: 6ba45625fd86307dd7fa996c70ff54d74ea4e0bd.
- Subject: docs(MAC-olrx): explain parent-owned relational test obligations.

### AC Verification
| AC # | Proof | Status |
|------|-------|--------|
| 1 | Fresh 7-case parent replay confirms retained relational obligations, deletion activation, covered parents and zero-obligation control; public guide now matches these results | PASS |
| 2 | Prior independent PM ownership review and targeted tests remain applicable; no behavior changed in rework | PASS |
| 3 | Prior independent PM declaration/stable-ID review remains applicable; frozen tests unchanged and verify-tdd has zero violations | PASS |
| 4 | Fresh actual CLI parent positive/negative cases pass; prior independent machine positive/negative proof unchanged | PASS |
| 5 | DOCS_STALE corrected precisely in docs/claude-plugin.md:88; public explanation now agrees with implementation and fresh parent regression evidence | PASS |

LEARNINGS:
- Initial GREEN updated diagnostics and source comments but missed a contradictory public plugin paragraph. Documentation freshness must include the user-facing CLI/hook comparison for changed gate selection.
- The prior ownership, source-activation and filesystem replay learnings remain authoritative; this rework changes only the public explanation.
- pvg verify scans no Markdown files, so its zero-issue result supplements neither prose review nor behavioral verification.

Limits: No broad suite or full preflight rerun for a documentation-only change. No pushes, syncs, GitHub mutations, installation/dev-link or live Machinery binary/plugin/skill changes. All seven freshly selected tests executed synchronously without skips.

## nd_contract
status: delivered

### evidence
- Documentation repair committed at 6ba45625fd86307dd7fa996c70ff54d74ea4e0bd, one Markdown file only.
- Fresh parent regression: 7 PASS / 0 FAIL / 0 skipped at the new SHA; coverage not measured.
- verify-tdd: 5 commits / zero violations; frozen RED bytes unchanged; clean diff and worktree.
- pvg verify: 0 Markdown files scanned / 0 issues, explicitly not proof of prose validation.

### proof
- [x] AC #1: fresh parent CLI/source-activation/zero-control replay passes; documentation corrected.
- [x] AC #2: prior independently verified owner isolation unchanged.
- [x] AC #3: prior independently verified declaration and stable-ID guarantees unchanged; frozen bytes preserved.
- [x] AC #4: fresh real parent CLI positive/negative tests pass; independent machine evidence unchanged.
- [x] AC #5: public parent-owned relational coverage description is accurate; DOCS_STALE resolved.


## nd_contract
status: rejected

### evidence
- PM rejection applied via pvg story reject on 2026-09-05.

### proof
- [ ] Story requires another developer delivery before it can be accepted.


## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## Implementation Evidence (GREEN DELIVERED)

PROOF:

### CI/Test Results
Commands run:
- go test -count=1 -timeout=5m -cover ./internal/gates -run 'Clause|Parent|Selection|Obligation' -v
- go test -count=1 -timeout=5m -cover ./internal/gates -run 'Select|IDCite|OracleCoverage|Gt|Snapshot|CheckPolicy|CheckIsolation|FailClosed|Rooted|Inventory' -v
- go test -count=1 -timeout=5m ./cmd/machinery -run '^TestGoldenCheck' -v
- TMPDIR=/tmp/machinery-olrx-green.WjsBu3/case-mount go test -count=1 -timeout=1m ./internal/gates -run '^TestSelectRejectsNonportableAndAliasedDesignPaths$' -v
- pvg verify internal/gates/suite.go internal/gates/clauses.go internal/gates/clausecov.go internal/gates/obligation_ownership_green_test.go --format text --include-tests
- pvg story verify-tdd --base epic/MAC-ui8a --json
- git diff --check epic/MAC-ui8a...HEAD
- git diff 27373f421640f644a98be89f38ce112c30f55a85 -- internal/gates/obligation_ownership_test.go
- git diff --name-status 27373f421640f644a98be89f38ce112c30f55a85..HEAD

Summary: GREEN PASS. At final SHA 8168b3d1d1599fd7860dd5afda5d7e55f49113aa, required targeted run: 46 top-level tests / 72 leaf cases PASS, 0 FAIL, 0 skipped, 9.701s, 37.5% internal/gates statement coverage. Includes all 23 frozen RED leaves, 46 preexisting leaves, and 3 additional GREEN leaves. Expanded affected run: 89 top-level tests / 104 leaves, 103 PASS, 0 FAIL, 1 preexisting host-filesystem skip, 5.073s, 52.5% package statement coverage. That exact skipped case was then independently executed on a task-owned case-sensitive APFS volume at the same SHA: all 4 leaves PASS, 0 FAIL, 0 skipped, 0.296s. Thus every expanded selected leaf was actually executed successfully; the native run's skip remains disclosed. CLI golden run: 4 top-level / 9 leaves PASS, 0 FAIL, 0 skipped, 4.129s, including go-crm, fulfillment, portfolio-engine, pii-flow, checkout-split parent/orders/payments, surreal-crm and external checker. Test groups overlap; counts must not be summed as unique coverage.

Final logs:
- /tmp/machinery-olrx-green.WjsBu3/targeted-final.log
- /tmp/machinery-olrx-green.WjsBu3/affected-final.log
- /tmp/machinery-olrx-green.WjsBu3/golden-final.log
- /tmp/machinery-olrx-green.WjsBu3/case-sensitive.log

The case-sensitive replay uses a bounded 256 MiB sparse disk image under the task's mktemp directory. hdiutil created it with the documented filesystem name 'Case-sensitive APFS', attached only the task-owned image at case-mount (disk10/disk11), and the EXIT cleanup detached it successfully: '"disk10" ejected.' Subsequent mount inventory confirms no task mount remains. No existing volume or container was altered. Initial APFSX alias attempt failed before creating any image; using hdiutil's documented filesystem name succeeded.

### Commit
- Branch: story/MAC-olrx.
- Frozen RED: 27373f421640f644a98be89f38ce112c30f55a85.
- GREEN implementation commits: 1b47643, 43b7357, 8168b3d.
- Final GREEN SHA: 8168b3d1d1599fd7860dd5afda5d7e55f49113aa.
- Final worktree clean. Entire story versus epic: 5 files, 702 additions / 32 deletions (includes frozen RED file). GREEN changes only three owned production files plus NEW obligation_ownership_green_test.go.
- Frozen RED file diff is empty; no preexisting test, fixture, golden or configuration bytes changed.
- verify-tdd checked 4 commits from local epic base: violations null, merge commits skipped 0.
- pvg verify: VERIFY: PASSED (4 files scanned, 0 issues). git diff --check clean.

### Wiring and ownership
- SelectRunAndNote -> selectInSnapshot in suite.go:335 retains Gt on machine-less parents with --impl when either a Policy/Isolation source annotation or its committed oracle exists. Source activation persists after generated-output deletion; actual Gp/Gn execute and block on the missing required oracle. The parent's note names its relational obligations. Obligation-free behavior and existing selection/golden text remain unchanged, and explicit Gt reports 0 machines / 0 test files scanned honestly.
- CheckOracleCoverage -> checkClauseCoverage in clausecov.go:62 -> collectClauseDecls in clauses.go:55. Gd's checkClauseDrift uses the same declaration/owner validation. Matrix filename stems bind declarations to that machine source and committed oracle; only matching guard rows in that owner's oracle produce suffixed obligations. No stable-ID generation changes.
- Duplicate/conflicting/empty/malformed declarations, repeated/active-retired conflicting vocabulary, missing owner source/oracle and guard absent from its owning oracle yield blocking diagnostics containing machine and guard.
- Machine artifact drift is scoped by filename owner. Shared narrative resolves a uniquely owned guard, or an explicitly named owner when several machines share it. An unresolved clause enumeration produces one honest ambiguity warning naming candidate owners. A guard mention with no active/retired clause tokens stays unarmed under the established all-or-none rule. Narrative never adds, removes or discharges Gt obligations. These ownership rules are documented in production comments and diagnostics.

### AC Verification
| AC # | Requirement | Code Location | Test Location | Status |
|------|-------------|---------------|---------------|--------|
| 1 | Retain parent Policy/Isolation coverage; zero obligations valid | suite.go:335 | frozen ParentSelectionRetainsRelationalCoverage and ParentRealCLI; new ParentDeletedRelationalOracleRemainsRequired | PASS: actual default/explicit CLI rejects each missing decision; covered and zero controls pass; deleting last output retains selected Gt and default Gp/Gn specifically block missing oracle |
| 2 | Independent machine plus guard ownership | clauses.go:55, clausecov.go:75, checkClauseDrift | frozen IndependentMachines, UndeclaredSiblingIsNotArmed, LocalDriftNamesOwner; new SharedNarrativeOwnership | PASS: Alpha two / Beta three clauses count exactly five; no sibling contamination or accidental arming |
| 3 | Every local falsifying clause, declaration validation, stable IDs | collectClauseDecls, checkClauseCoverage | frozen EveryLocalClauseRequired, InvalidDeclarationRejectedByBothGates, DuplicateDeclarationRejected, OwnerCannotResolveThroughSibling, SingleMachinePositiveControl | PASS: all suffix deletions reject, malformed/orphan cases reject in both Gt/Gd, generator output remains byte-identical |
| 4 | Real positive/negative full-path coverage | suite.go, clauses.go, clausecov.go | frozen parent CLI, omitted machine, wrong owner, removed oracle, independent machine controls | PASS: actual filesystem, production oracle generator, snapshots, selected gates and subprocess CLI; no mocks |
| 5 | Actual selection/results with precise diagnostics | selectInSnapshot, shared declaration validator, owner-local drift/coverage | frozen selectedObligationGates and requireOwnerError; new source-deletion and narrative ownership tests | PASS: owner plus guard appears within the same blocking error; local drift warnings retain owner and clause details |

LEARNINGS:
- Ownership must travel with the declaration into committed oracle row selection; guard vocabulary alone is not identity.
- Parent source annotations preserve activation after deleting generated output; Gp/Gn independently establish missing-output failure.
- Shared prose with zero clause tokens is deliberately unarmed. An initial overly broad ambiguity warning broke fulfillment's unchanged golden; implementation was corrected to warn only for actual unresolved clause enumeration.
- The new source-activation test initially copied two domain model files. Its newly authored temporary fixture was corrected to use one model before the clean final run; no frozen test was repaired.
- A native filesystem-specific skip can be independently replayed without changing the test: the task-owned APFS case-sensitive TMPDIR exercised it and was detached. All newly required cases executed without skips on native Go.

Limits: Proportional suites only, as authorized; full repository suite and scripts/preflight.sh remain the final epic gate's responsibility. Coverage percentages are targeted package statement coverage, not full-suite or subprocess CLI coverage. No runtime-backed test was introduced, no live Machinery binary/plugins/skills/agents were installed or replaced, and no GitHub push/sync/mutation, tool upgrade or product Paivot dependency was added. The new parent source-deletion regression proves its selected relational gate behavior; its copied parent architecture is not represented as a complete CRM architecture.

## nd_contract
status: delivered

### evidence
- RED 27373f421640f644a98be89f38ce112c30f55a85 -> GREEN 8168b3d1d1599fd7860dd5afda5d7e55f49113aa.
- Required 72/72 leaves pass with no skips, 37.5% targeted coverage; 9/9 CLI golden leaves pass.
- Expanded 103 pass / 1 native host skip; exact skipped case independently passed on case-sensitive APFS, 4/4 replay leaves pass.
- verify-tdd 4 commits / zero violations; pvg verify 4 files / zero issues; frozen bytes preserved; clean worktree.

### proof
- [x] AC #1: parent-owned relational coverage retained, source deletion remains selected and blocked, positive/zero controls pass.
- [x] AC #2: machine plus guard isolation for coverage and drift.
- [x] AC #3: each suffix and invalid/duplicate/orphan declarations enforced with unchanged stable IDs.
- [x] AC #4: actual filesystem/CLI positive and negative integration cases pass.
- [x] AC #5: actual selected gates and owner-specific blocking diagnostics pass.


## nd_contract
status: red-approved

### evidence
- RED tests approved via pvg story approve-red on 2026-09-05.

### proof
- [ ] GREEN developer must implement against the approved RED tests without modifying them.


## PM RED Review — 2026-09-05

RED REVIEW: APPROVED. This approves the frozen test specification only; no GREEN implementation or story acceptance is claimed.

Reviewed LOCAL epic/MAC-ui8a...story/MAC-olrx at 27373f421640f644a98be89f38ce112c30f55a85. Exactly one 463-line test file is added, with tdd-red in the immutable commit subject and no production edits. Read the complete source and actual production clause/selection paths. No stubs, TODOs, skip-if-missing, environment gates, mocks, or product Paivot dependencies are present. Fixture test identifiers are inputs to the coverage gate under test; they are not claimed as proof of a generated application implementation.

Independent synchronous replay from retained dev worktree:
- go test -count=1 -timeout=5m -cover ./internal/gates -run 'Clause|Parent|Selection|Obligation' -v: exit 1, 8.264s; 44 top-level cases (33 PASS, 11 intended FAIL), 69 leaf cases (48 PASS, 21 intended FAIL), 0 skipped; 27.7% targeted package statement coverage. All 46 preexisting leaf cases pass. No compile/import/infrastructure failures.
- go test -count=1 -timeout=5m ./internal/gates -run 'TestObligationClausesSingleMachinePositiveControl$|TestObligationParentRealCLI$/obligation-free-parent-control$' -v: exit 0, 1.889s; 2 passing controls, 0 failed/skipped.
- git diff --check epic/MAC-ui8a...HEAD clean; retained worktree clean and HEAD equals frozen SHA.
- Static marker scan over the delivered file found no stub, skip, environment bypass or Paivot references.

Observed failures match the contract: actual CLI default parent reports platform-green after a required Policy/Isolation ID is removed while explicit Gt rejects it; same-name guard vocabularies cross machine ownership, overcount clause coverage and contaminate drift; malformed/duplicate/conflicting/orphan declarations are accepted or lack blocking owner-specific diagnostics. These are expected behavioral assertions, not unrelated errors. Positive parent checks, recursively valid packs, compiled local implementation tests, generator-produced machine IDs and oracle-byte preservation assertions execute on actual native filesystem/CLI paths.

AC assessment:
1. Parent selection plus default/explicit CLI negatives exercise both relational oracle kinds, covered positives, and honest zero-obligation control.
2. Different clause counts and overlapping vocabulary on Alpha/Beta require independent coverage and drift; undeclared sibling must remain unarmed.
3. Every suffix is separately removed; duplicate/conflicting/empty/malformed declarations, missing owner/guard/oracle and stable generated IDs are explicitly asserted.
4. Missing test IDs, omitted machine, wrong owner and removed oracle use actual gates/files; two independent correctly covered machines and same-machine positive controls prevent blanket rejection from satisfying RED.
5. SelectRunAndNote, Gt, Gd and real CLI are exercised; owner plus guard must appear in the same blocking diagnostic. Retired/partial local drift stays checked without sibling leakage.

GREEN must preserve the exact RED test bytes and pass them unchanged, retain existing behavior tests, and update relevant ownership/selection documentation. No test repair is authorized. No full preflight, push, install, binary replacement or source mutation was performed by PM.

## nd_contract
status: delivered

### evidence
- Independent RED replay and source review at 27373f421640f644a98be89f38ce112c30f55a85 as detailed above.

### proof
- [x] AC #1: meaningful RED selection/CLI assertions and positive control verified.
- [x] AC #2: meaningful RED owner-isolation assertions verified.
- [x] AC #3: meaningful RED validation/stable-ID assertions verified.
- [x] AC #4: required actual-path negative and positive cases verified.
- [x] AC #5: actual gate execution and ownership diagnostics verified.
- [ ] GREEN implementation and final acceptance remain required.

## Implementation Evidence

RED phase only: frozen tests at 27373f421640f644a98be89f38ce112c30f55a85. Detailed PROOF, CI/Test Results, Commands run, Summary, AC Verification and LEARNINGS are in the immediately preceding evidence block. No GREEN implementation is claimed.

## nd_contract
status: delivered

### evidence
- Frozen RED test commit: 27373f421640f644a98be89f38ce112c30f55a85.
- go test -count=1 -timeout=5m -cover ./internal/gates -run 'Clause|Parent|Selection|Obligation' -v: 48 passing / 21 intended failing leaf cases, 0 skipped; 27.7% targeted statement coverage.
- Separate positive-control replay: 2 passing / 0 failing leaf cases.
- pvg verify: 1 file scanned, 0 issues.
- Initial pvg story deliver completed; repeated deliver refused an existing delivered label. No manual label changes made.

### proof
- [x] AC #1: RED parent-selection and real CLI coverage tests authored; zero-obligation control passes.
- [x] AC #2: RED independent machine+guard clause coverage and drift tests authored.
- [x] AC #3: RED declaration validation, every suffix and stable-ID preservation tests authored.
- [x] AC #4: RED full-path positive/negative inventory authored.
- [x] AC #5: RED actual selection, Gt/Gd and owner-specific diagnostic tests authored.
- [ ] GREEN implementation and independent acceptance remain outstanding.

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
- 2026-09-05T20:02:02Z status: in_progress -> open
- 2026-09-05T20:08:04Z status: open -> in_progress
- 2026-09-05T20:08:04Z claimed by dev-MAC-olrx
- 2026-09-05T20:19:58Z status: in_progress -> in_progress
- 2026-09-05T20:22:59Z status: in_progress -> open
- 2026-09-05T20:22:59Z released by ramirosalas
- 2026-09-05T20:23:16Z status: open -> in_progress
- 2026-09-05T20:23:16Z claimed by dev-MAC-olrx
- 2026-09-05T20:25:00Z status: in_progress -> in_progress
- 2026-09-05T20:27:03Z status: in_progress -> closed
- 2026-09-05T20:27:03Z dep_removed: no_longer_blocks MAC-vx24
- 2026-09-05T20:27:03Z dep_removed: no_longer_blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]

## Comments

### 2026-09-05T20:23:00Z ramirosalas
## PM Decision
REJECTED [2026-09-05]: DOCS_STALE only. Behavioral implementation and frozen RED replay pass; public documentation still contradicts the changed guarantee.

EXPECTED: AC #1 retains parent Policy/Isolation test obligations despite zero machines, and AC #5 documents precise ownership. PM documentation-freshness gate requires user-facing descriptions of changed default selection to remain accurate. The prior RED review explicitly required relevant ownership/selection documentation to be updated.

DELIVERED: GREEN 8168b3d1d1599fd7860dd5afda5d7e55f49113aa correctly retains default Gt with --impl for parent-owned relational source annotations or committed oracles, but unchanged docs/claude-plugin.md:88-92 says on a machine-less decomposed parent the default selection skips Gt; it also says with zero machines the hook has nothing to hold and reports vacuously green, calling both renderings the same fact.

GAP: These statements are false when the parent owns Policy/Isolation obligations. Zero machines is not zero obligations. Users comparing CLI and hook results receive exactly the unsafe inference this story removes. Code comments alone cannot repair a contradictory public plugin guide.

FIX: Update only the relevant docs/claude-plugin.md paragraph to distinguish truly obligation-free parents from parents with Policy/Isolation source annotations or committed decision oracles. With --impl, default CLI retains Gt for those obligations; deletion of generated output must not erase source activation, and Gp/Gn report the missing required output. The hook still runs Gt when impl is configured; empty counts describe only genuinely obligation-free designs, not all machine-less parents. Keep the existing zero-obligation CLI/hook distinction accurate. No unrelated README polish is required. Preserve every frozen RED test byte; no test edit is authorized. Deliver the scoped docs repair with fresh commit/diff evidence.

Independent verification at the delivered SHA:
- pvg story verify-tdd --base epic/MAC-ui8a --json: 4 commits checked, zero violations; git diff from frozen RED27373f421640f644a98be89f38ce112c30f55a85 to HEAD for obligation_ownership_test.go is empty.
- go test -count=1 -timeout=5m -cover ./internal/gates -run 'Clause|Parent|Selection|Obligation' -v: 72/72 leaf cases PASS, zero failed/skipped, 37.5% targeted coverage, 9.669s. This includes all immutable RED cases and actual positive/negative CLI paths.
- go test -count=1 -timeout=5m ./cmd/machinery -run '^TestGoldenCheck' -v: 9/9 leaves PASS, zero failed/skipped, 3.286s; no golden files modified.
- go test -count=1 -timeout=5m -cover ./internal/gates -run 'Select|IDCite|OracleCoverage|Gt|Snapshot|CheckPolicy|CheckIsolation|FailClosed|Rooted|Inventory' -v: 103 PASS, zero FAIL, one preexisting case-folded host-filesystem SKIP, 52.5%, 5.442s. The skipped native case is not counted as executed by PM. Reviewed developer case-sensitive APFS replay log showing all four associated leaves passed at this SHA, and recorded bounded mount teardown; no independent APFS mount was needed.
- Actual source review confirms shared machine+guard declaration validation, owner-local oracle rows, no sibling arming, explicit malformed/orphan diagnostics, source-preserving parent activation and honest shared-narrative ambiguity warnings. Targeted tests remove each required local suffix, omit one owner, rename/delete owner artifacts, remove parent decision IDs and delete last relational output; each mutation is blocked as required. Positive independent owners remain clean.
- Static diff: 5 files total, 702 additions/32 deletions including frozen RED; only three production files and a new test file changed in GREEN. No stubs/production Paivot dependency, preexisting test/fixture/golden edits or source edits by PM. git diff --check clean and retained worktree clean.
- No preflight, live binary/install/plugin/skill replacement, GitHub mutation, push or sync performed. All tests completed synchronously.

## nd_contract
status: rejected

### evidence
- Independent full GREEN review at8168b3d1d1599fd7860dd5afda5d7e55f49113aa; behavior passes, public plugin guide contradicts parent selection.

### proof
- [x] AC #1: behavior verified; corresponding public documentation repair remains required.
- [x] AC #2: owner-local coverage/drift independently verified.
- [x] AC #3: validation and stable-ID invariance independently verified.
- [x] AC #4: actual-path negative and positive tests independently verified.
- [ ] AC #5 / DOCS_STALE: docs/claude-plugin.md:88-92 must accurately explain parent-owned relational obligations.

### 2026-09-05T20:25:34Z ramirosalas
## nd_contract
status: delivered

### evidence
- Documentation-only rework for DOCS_STALE: 6ba45625fd86307dd7fa996c70ff54d74ea4e0bd, docs/claude-plugin.md:88; prior rejection is addressed.
- Full fresh Implementation Evidence (GREEN REWORK: DOCS_STALE), CI/Test Results, Commands run, Summary, AC Verification and LEARNINGS are appended in Notes.
- Fresh go test -count=1 -timeout=5m ./internal/gates -run '^TestObligationParent' -v: 7 PASS / 0 FAIL / 0 skipped, 7.800s, at this SHA. Coverage not measured.
- pvg story verify-tdd --base epic/MAC-ui8a --json: 5 commits / zero violations. Frozen RED27373f421640f644a98be89f38ce112c30f55a85 unchanged.
- pvg verify docs/claude-plugin.md --format text: 0 files scanned / 0 issues (Markdown is not inspected). Manual paragraph/source comparison and fresh regression replay provide the documentation proof.
- pvg story deliver succeeded exactly once for this rework. This terminal append-only contract supersedes the prior PM rejection comment; Notes render before Comments in tracker output.
- Only one documentation file changed since prior GREEN; clean diff/worktree. No push/sync/install/preflight or product dependency changes.

### proof
- [x] AC #1: public parent coverage explanation corrected; fresh parent positive/negative/zero/source-deletion replay passes.
- [x] AC #2: independently reviewed machine plus guard ownership behavior unchanged.
- [x] AC #3: independently reviewed declaration validation/stable IDs unchanged; frozen RED bytes preserved.
- [x] AC #4: actual parent CLI paths freshly pass; prior independent machine cases unchanged.
- [x] AC #5: docs/claude-plugin.md now accurately distinguishes parent relational obligations from genuinely obligation-free parents; DOCS_STALE resolved.


### 2026-09-05T20:27:19Z ramirosalas
## nd_contract
status: accepted

### evidence
- PM re-review accepted final GREEN6ba45625fd86307dd7fa996c70ff54d74ea4e0bd; canonical pvg story accept succeeded and independently verified closed+accepted.
- Full PM GREEN Re-review evidence is preserved in Notes. Fresh independent targeted replay:72/72 leaf cases pass,0fail,0skip,9.000s,37.5% targeted coverage. verify-tdd5commits0violations; frozen RED27373f421640f644a98be89f38ce112c30f55a85 unchanged.
- DOCS_STALE correction in docs/claude-plugin.md independently compared with actual source and CLI tests. Only documentation changed since prior fully reviewed behavioral GREEN; historical golden/expanded evidence remains accurately SHA-scoped.
- This terminal append-only contract reconciles rendered Notes-before-Comments ordering; it supersedes the older delivered comment without changing or erasing history.
- Epic sibling check:16 children remain not closed; epic is not eligible for closure.

### proof
- [x] AC #1: parent-owned relational coverage, source-preserving activation, positive and zero controls verified.
- [x] AC #2: independent machine plus guard coverage/drift verified.
- [x] AC #3: negative declaration/owner/suffix cases and stable IDs verified.
- [x] AC #4: actual positive/negative filesystem/generator/gate/CLI tests executed.
- [x] AC #5: precise ownership diagnostics and corrected public explanation verified.
