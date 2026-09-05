---
id: MAC-a89e
title: "Keep regeneration advice from accepting new debt"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd, delivered]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:27Z
created_by: ramirosalas
updated_at: 2026-09-05T21:23:23Z
content_hash: "sha256:ecb3d6466683d9610372e0102307f5b6624443ee7ffb76263ba19cb6e9196843"
blocks: [MAC-gcrr, MAC-ou97]
assignee: dev-MAC-a89e
follows: [MAC-olrx]
---

## Description
## USER INTENT
Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.

## Context (Embedded)
NEXT9: version-skew advice includes machinery baseline when ratchet exists. Baseline snapshots tolerated import offenders and can widen accepted architecture debt; ratchet has no version stamp. Routine regeneration must not silently authorize debt.
Same-story AC3 defect confirmed during RED preparation: explicit baseline on an already-baselined edge with a newly added offender proposes zero dependency rules, prints "the contract already covers every observed edge; nothing new to baseline", then rewrites ratchet.json to accept the new offender. Current help's "after review" refers only to pasted dependency rules. Help and successful output must explain ratchet/offender debt acceptance and review even when no rule is proposed. Explicit deliberate baseline remains supported.

## Ownership
Own exactly four files: internal/gates/gates.go (routine regeneration advice), cmd/machinery/baseline.go (narrow help and successful-output debt-review messaging), internal/gates/regeneration_safety_test.go (new focused and real CLI regression coverage), and internal/gates/gates_test.go (only the independently authorized TestVersionSkewNoteNamesEveryApplicableCommand expectation repair). Preserve that existing test's ratchet-present fixture and oracle/Alloy/verify-formal/pack expectations; replace only baseline expectation with explicit baseline absence. No other existing test/assertion changes are authorized. You are not alone in this codebase; preserve other edits and coordinate shared-file changes with dispatcher.

## Boundary Map
PRODUCES:
- internal/gates/gates.go -> hardened behavior and regression proof
- internal/gates/regeneration_safety_test.go -> hardened behavior and real CLI regeneration/baseline regression proof
- cmd/machinery/baseline.go -> help and successful output explaining deliberate ratchet/offender debt acceptance and required review
- internal/gates/gates_test.go -> narrowly authorized baseline-absence expectation in TestVersionSkewNoteNamesEveryApplicableCommand
CONSUMES:
- Existing Machinery source interfaces.
  spec: regenCommands(design string) []string; VersionSkewNote(design string, gs []*Gate) string
- Existing internal/gates/baseline.go (read-only behavior consumer).
  spec: BuildBaseline(design, impl, date string) (*BaselineReport, error)
  source: already-baselined edges are re-snapshotted using current offender files; zero Proposed rules does not mean ratchet debt is unchanged.
- Existing cmd/machinery/baseline.go.
  spec: newBaselineCmd() *cobra.Command
  source: baseline <design-dir> --impl <dir>; --date controls deterministic snapshot date; successful command publishes ratchet.json.

### Story Acceptance Criteria
1. Version-skew regeneration instructions never include machinery baseline or any debt-accepting mutation. Existing oracle/Alloy/formal/pack regeneration remains accurate and deterministic.
2. A design with ratchet and newly introduced offender continues failing architecture checks after following all routine regeneration instructions; regenerated stamps cannot accept the offender.
3. Explicit machinery baseline remains a supported deliberate user-invoked debt-acceptance operation. Before invocation, baseline --help must explain that rerunning rewrites ratchet.json and may accept newly added offender files, and tell users to review ratchet/offender changes before adopting them. After successful publication, output must convey the same debt-change review guidance even when zero dependency rules were proposed. The zero-rule message must describe only the absence of new rule proposals and must not claim that no debt was accepted or nothing changed. Real CLI regression: start with an already-baselined edge, add a new offender file, prove the architecture check fails, deliberately run baseline successfully, verify the new offender is now recorded/accepted with zero new rule proposals, and assert help plus successful output require ratchet/offender-change review. This deliberate acceptance is allowed; routine regeneration must never perform it. No automatic migration, new confirmation protocol, version stamp or prohibition of explicit baseline.
4. Positive no-debt/version-skew fixture yields correct required generator commands; negative existing-ratchet/new-offender regression demonstrates failure before and after advised regeneration.

## Testing Requirements
- Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
- Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
- Commands: go test -count=1 ./internal/gates -run 'VersionSkew|Regen|Ratchet'; real built CLI integration on an isolated ratchet design with a genuine new boundary violation. New named tests should match the scoped selection or record their exact supplemental command.
- AC3 CLI proof is required alongside the routine-regeneration negative: capture baseline --help; then exercise the already-baselined/zero-proposed-rule/new-offender success path and assert actual ratchet expansion plus explicit debt-review guidance. A vague "after review" referring only to dependency rules is insufficient. Pin the semantic requirement, not arbitrary exact prose. Preserve actual explicit-baseline functionality and passing controls; no mocks/stubs/skips or external service dependency is needed.
- Existing PM comment at 2026-09-05T21:14:38Z authorizes only the named gates_test.go expectation repair. That repair commit subject must include tdd-red and [test-edit-authorized]. This development coordination convention is never a product/runtime requirement. Freeze revised tests/fixtures after independent RED review; GREEN may not weaken them.
- No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.

## OUT OF SCOPE
- Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
- Global heavy preflight, main merge and local release binary belong to final epic gate.
- Baseline algorithm, ratchet schema/date semantics and new approval/confirmation mechanisms: not needed for the identified advice/messaging defect; preserve existing deliberate baseline behavior.
- Machinery product must remain standalone; no Paivot/pvg/nd dependency, workflow labels or commit conventions in shipped behavior.

## DIFF BUDGET
- Exactly four owned files, under 300 total changed LOC. The file-count increase from 2-3 is justified by one bounded CLI help/output change and the independently authorized existing-test expectation repair alongside gates.go/new regression coverage. Keep the original tight LOC ceiling; report concrete real-CLI fixture needs before any material overrun instead of weakening proof.

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

## Acceptance Criteria


## Design


## Notes
AUTHORITATIVE USER CONSTRAINTS 2026-09-05: Machinery product must be standalone, never require Paivot/pvg/nd, workflow labels or commit conventions. Local development coordination only may use Paivot. Another agent uses installed Machinery in NIL: do not replace installed binary/plugins/skills/agents; no dev-link or live install/update. Build isolated candidate only. No GitHub push/mutation during work. Full scripts/preflight.sh only final epic gate. RED author may update preexisting tests that encode superseded unsafe behavior with explicit review and genuine assertion-failure proof; after RED approval freeze exact tests/fixtures/config bytes.
ANCHOR ROUND-1 RUNTIME CLASSIFICATION: This story's required current cases are service-free Go/native filesystem/local-process tests and real local CLI paths; no Docker/Java/Node dependency is implicit in ordinary native suites. Preserve actual non-mocked path tests. If implementation introduces any external runtime-backed case, it MUST add a dedicated closed fragment under testdata/integration-lanes via the shared required lane and declare ownership/dependency before delivery; no skip-if-missing, env-gated omission or reliance on later incidental execution. Missing service cannot silently convert required coverage to success.
RED-DISPUTE: paused before delivery at dispatcher direction. RED commit 9b85ef6 adds internal/gates/regeneration_safety_test.go only. Authorized superseded assertion repair commit f482c7683f5005e40722123e3a382e01b15219d2 changes only TestVersionSkewNoteNamesEveryApplicableCommand as permitted by independent PM comment (tdd-red and [test-edit-authorized] subject). Production unchanged; worktree clean.

PROOF (provisional, NOT RED approval): go test -count=1 -timeout=5m ./internal/gates -run "VersionSkewRegenerationOnlySafeGenerators|RegenRatchetRealCLI" at 9b85ef6 completed in 5.046s: 10 leaves, 7 pass, 3 fail, 0 skip. Unit ratchet-only and all-with-ratchet fail because advice includes baseline; five single-family controls pass. CLI no-debt-version-skew-control passes. CLI new-offender-survives-all-advice fails on exactly unsafe advice, changed ratchet bytes, and unexpected green G4 after executing every printed command: alpha/a.go snapshot expands to alpha/a.go plus alpha/b.go and G4 returns zero findings. CLI explicit-baseline-reviews-debt-change superficially passes, but its review substring matches the temporary subtest pathname; that AC3 assertion is invalid evidence and requires bounded repair authorization. Proposed repair checks actual prose guidance and adds help-before-invocation coverage when canonical scope is repaired. No compile/setup/runtime prerequisite failures. go test -count=1 -timeout=5m ./internal/gates -run "^TestVersionSkewNoteNamesEveryApplicableCommand$" at f482c7683f5005e40722123e3a382e01b15219d2: 0 pass, 1 fail, 0 skip, intended baseline-absence assertion. Full requested target with JSON inventory and coverage remains pending after repair.

pvg verify internal/gates/regeneration_safety_test.go --format text: PASS, 1 file, 0 issues. pvg verify internal/gates/gates_test.go internal/gates/regeneration_safety_test.go --format text: FAILED, 2 files, 1 stub at gates_test.go:125 return empty string; this is outside the narrowly authorized amendment and must be reviewed rather than silently edited.

AC mapping: AC1 precise generator unit fixtures and authorized existing expectation fail behaviorally; AC2 real CLI byte and G4 checks fail behaviorally with initial G4 green and genuine new-offender rejection before regeneration; AC3 explicit baseline functionality works but guidance assertion is disputed; AC4 passing real no-debt control plus negative CLI reproduced, complete inventory pending. Coverage percentage not yet measured; no delivery claim. Codebase-memory Verify inherited parent exact symbols/coverage; index_status confirmed ready, worktree excluded so exact source fallback used.

LEARNINGS:
- Output assertions can accidentally match temporary test directory names; test prose rather than unrestricted substring presence.
- Real baseline reruns expand an already-baselined edge even when output says nothing new to baseline.
- Executing all printed advice with placeholder substitution demonstrates debt mutation directly without mocks or external services.
## RED Delivery Evidence

PROOF:

### Commits and immutable candidate
- RED: 9b85ef69ed426a337e4e2584867a698904676643.
- Narrow existing-test amendment: f482c7683f5005e40722123e3a382e01b15219d2, authorized 2026-09-05T21:14:38Z; retained ratchet fixture/all four safe generator expectations and asserted baseline absence only.
- Authorized guidance repair / tested candidate: e95be63019d89b93ae527f27ae1ce43326def038, authorized 2026-09-05T21:18:55Z; both literal tdd-red and [test-edit-authorized] markers included. Prior commits preserved without amend/rebase.
- Branch story/MAC-a89e; worktree /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-a89e. Production unchanged. Diff against epic/MAC-ui8a: regeneration_safety_test.go +243; gates_test.go +3/-1; 247 total changed LOC across 2 test files, leaving bounded room for 2 production files within canonical 4-file/under-300 budget.
- Test SHA256: internal/gates/gates_test.go e4f630ec2ff7684eea436881b8a55706496f348c0ae7cd8748a255a6b6d439ff; internal/gates/regeneration_safety_test.go 2e1801d96618aaf146621e822a342b4bfdf69cdace50b94d0585d7f1ad4f20c3.

### Exact commands / retained evidence
All commands ran with explicit cd to the story worktree.
- set -o pipefail && go test -count=1 -timeout=5m ./internal/gates -run 'VersionSkew|Regen|Ratchet' -json -coverprofile=/tmp/MAC-a89e-red.wlHWaG/coverage.out | tee /tmp/MAC-a89e-red.wlHWaG/tests.json
- Exit 1: expected RED behavioral assertion failures. 37 leaf tests, 32 PASS, 5 FAIL, 0 SKIP; package elapsed 4.832s. No compile/setup/deadline/missing-prerequisite failures. All new tests are selected by the specified regexp; no supplemental selector needed.
- Raw JSON /tmp/MAC-a89e-red.wlHWaG/tests.json SHA256 405f556e1036a36547bf9231d38d3cdedca5d52acc6879b997f95839789584bb.
- Exact leaf inventory /tmp/MAC-a89e-red.wlHWaG/inventory.txt SHA256 7e20dff8a081de4734b40f68cf07da4d334362e37ed1654cea2b9c121c90e8af.
- Coverage profile /tmp/MAC-a89e-red.wlHWaG/coverage.out SHA256 1ff93989e5a6fb5a2c6c70dd42dc3214e241ff62a74f1429471dbc0f44e1eef0.
- go tool cover -func=/tmp/MAC-a89e-red.wlHWaG/coverage.out: internal/gates total 9.9%; gates.go regenCommands 100.0%; gates.go VersionSkewNote 93.8%. This is scoped package coverage; separately built real CLI subprocess is not included in Go package instrumentation.
- git diff --check epic/MAC-ui8a: PASS. Worktree clean.
- pvg verify internal/gates/regeneration_safety_test.go --format text: PASS, 1 file, 0 issues.
- pvg verify internal/gates/gates_test.go internal/gates/regeneration_safety_test.go --format text: reports one [stub] finding, gates_test.go:125 return "". Raw result /tmp/MAC-a89e-red.wlHWaG/verify.txt. Specific independent PM false-positive disposition at 2026-09-05T21:18:55Z: coveringInterfaceTable returns empty only for no concrete allow rows; otherwise renders the full interface table, consumed by c4GraphFixture and graph tests. Preserved helper unchanged. This is a recorded single finding, not blanket scanner waiver; earlier note's helper name fixtureInterfaceRows was incorrect and is superseded by coveringInterfaceTable.

### Exact test inventory
PASS TestG4RatchetSnapshotNote
PASS TestG4BaselineWithoutRatchetFails
PASS TestG4RatchetGreenAtSnapshot
PASS TestG4RatchetGrowthFails
PASS TestG4RatchetShrinkAndStaleEdgesNote
PASS TestRatchetRoundTrip
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/unknown_root
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/missing_date
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/missing_edges
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_date
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_edges
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_edge_name
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/date_number
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/date_null
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edges_array
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edge_value_string
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edge_entry_null
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/trailing_value
PASS TestWriteRatchetRejectsSymlinkTarget
FAIL TestVersionSkewNoteNamesEveryApplicableCommand
PASS TestVersionSkewNoteFormalCommandFollowsGeneratedTLA
PASS TestRatchetSnapshotNoteBothFormatsIsClockIndependent
FAIL TestVersionSkewRegenerationOnlySafeGenerators/ratchet-only
PASS TestVersionSkewRegenerationOnlySafeGenerators/oracle
PASS TestVersionSkewRegenerationOnlySafeGenerators/alloy
PASS TestVersionSkewRegenerationOnlySafeGenerators/semantics
PASS TestVersionSkewRegenerationOnlySafeGenerators/composition
PASS TestVersionSkewRegenerationOnlySafeGenerators/pack
FAIL TestVersionSkewRegenerationOnlySafeGenerators/all-with-ratchet
PASS TestRegenBaselineGuidanceSensitivity/genuine
PASS TestRegenBaselineGuidanceSensitivity/equivalent-wrapped
PASS TestRegenBaselineGuidanceSensitivity/path-bait
PASS TestRegenBaselineGuidanceSensitivity/old-rule-review-help
PASS TestRegenBaselineGuidanceSensitivity/unrelated-review
PASS TestRegenRatchetRealCLI/no-debt-version-skew-control
FAIL TestRegenRatchetRealCLI/new-offender-survives-all-advice
FAIL TestRegenRatchetRealCLI/explicit-baseline-reviews-debt-change

### Independent AC-to-test evidence
| AC | Test and observable result | RED status |
|---|---|---|
| 1 | VersionSkewRegenerationOnlySafeGenerators pins exact ordered command list with no additional mutations for every family, individual controls and all-with-ratchet. Ratchet-only/all-with-ratchet plus authorized existing expectation fail because machinery baseline is printed. Unchanged repeated inputs produce identical advice; single-family positive controls pass. | Intended failure |
| 2 | RealCLI/new-offender-survives-all-advice starts with CLI-created ratchet containing alpha/a.go and green g3,g4, then adds alpha/b.go and gets exact G4 growth finding. Executes EVERY printed routine command (including unsafe baseline after substituting its documented impl placeholder), proves current oracle stamp generated, then compares exact ratchet bytes and requires G4 still reject alpha/b.go. Unchanged production expands ratchet to [alpha/a.go,alpha/b.go], followed by G4 zero blocking findings. | Intended failure on advice, bytes and G4 result |
| 3 | RealCLI/explicit-baseline-reviews-debt-change logs actual --help before baseline invocation; requires ratchet replacement, possible new-offender acceptance and review-before-adoption prose. Existing ratchet + new offender fails G4 first; deliberate real baseline succeeds with explicit "0 need a baseline rule", records both offenders, remains unstamped and subsequent G4 passes. Current help fails guidance; successful output fails guidance and "nothing new to baseline" falsely implies no debt change. | Intended failure for missing guidance / misleading zero-rule output; explicit operation remains successful |
| 4 | RealCLI/no-debt-version-skew-control runs oracle on a valid source, introduces stamp-only skew, obtains exactly the oracle advice, executes it, verifies current stamp, no skew after regeneration, g3,g4 green, and no new ratchet. All five sensitivity controls pass (two equivalent valid prose variants, unsafe output with review/reviews path bait, old dependency-rule-only review help, unrelated review words). | Controls PASS; negative AC2 demonstrably fails |

### Limits and scope
This delivery is RED ONLY, not product completion or permission to begin GREEN without independent RED approval. Integration builds ./cmd/machinery into t.TempDir and uses real CLI/native files unconditionally, no mocks, services, skips or env gates. Real executable fixture carries oracle generation, so all applicable printed commands are executed. Other safe generator families are pinned as exact advice-selection controls; no Alloy solver/TLA model-check execution is claimed. No full preflight, installed binary/plugin/skill modification, push/sync/GitHub write, main branch move, runtime schema change or Paivot product dependency.

LEARNINGS:
- Initial AC3 check falsely passed because the temporary pathname inherited "reviews" from its subtest. After independent authorization, matching now removes path tokens and relates review/change/ratchet/offender semantics within prose, with explicit passing sensitivity controls.
- Existing baseline rerun expands offender snapshots despite zero new dependency-rule proposals; rule proposal counts and accepted debt changes are distinct observations.
- Executing all printed commands exposes the exact harmful mutation: regenerated current stamps alone do not preserve architecture enforcement if the advice also re-baselines debt.
- A scanner can flag a legitimate conditional empty fixture section; preserve implemented behavior and record an exact reviewer disposition.
- Retained raw JSON and hashes make failure provenance independently replayable; initial summarized-only evidence was insufficient for that purpose.

## nd_contract
status: in_progress

### evidence
- Candidate e95be63019d89b93ae527f27ae1ce43326def038; complete scoped RED 32 pass / 5 intended fail / 0 skip, raw evidence above.
- Authorized repaired tests, preserved original commits and specific static-finding disposition.

### proof
- [x] AC #1: behavioral RED with exact safe advice and positive controls authored.
- [x] AC #2: real CLI harmful mutation independently observable in ratchet bytes and G4 output.
- [x] AC #3: missing help/output guidance now genuinely fails; explicit operation and matcher controls exercised.
- [x] AC #4: real no-debt positive plus genuine negative path executed; full inventory recorded.


## nd_contract
status: in_progress

### evidence
- Provisional RED and authorized amendment committed; paused for narrow test repair authorization and AC3 scope update.
- No delivery, claim release, production change, installed asset change, or remote mutation.

### proof
- [ ] AC #1: intended assertion failures established; independent RED approval pending.
- [ ] AC #2: real CLI regression established; independent RED approval pending.
- [ ] AC #3: RED-DISPUTE false-positive assertion must be repaired after authorization.
- [ ] AC #4: complete target inventory and coverage pending repair.

## Historical canonical description before MAC-a89e scope repair
Preserved verbatim as quoted history; current Description is authoritative.

> ## USER INTENT
> Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.
> 
> ## Context (Embedded)
> NEXT9: version-skew advice includes machinery baseline when ratchet exists. Baseline snapshots tolerated import offenders and can widen accepted architecture debt; ratchet has no version stamp. Routine regeneration must not silently authorize debt.
> 
> ## Ownership
> Own only these paths and directly associated tests: internal/gates/gates.go, internal/gates/regeneration_safety_test.go. You are not alone in this codebase; preserve other edits and coordinate any shared-file changes with dispatcher.
> 
> ## Boundary Map
> PRODUCES:
> - internal/gates/gates.go -> hardened behavior and regression proof
> - internal/gates/regeneration_safety_test.go -> hardened behavior and regression proof
> CONSUMES:
> - Existing Machinery source interfaces.
>   spec: regenCommands(design string) []string
> 
> ### Story Acceptance Criteria
> 1. Version-skew regeneration instructions never include machinery baseline or any debt-accepting mutation. Existing oracle/Alloy/formal/pack regeneration remains accurate and deterministic.
> 2. A design with ratchet and newly introduced offender continues failing architecture checks after following all routine regeneration instructions; regenerated stamps cannot accept the offender.
> 3. Explicit baseline remains a deliberate user-invoked operation with clear debt-change review guidance, not automatic migration. Do not add a version stamp as a substitute for preventing debt expansion.
> 4. Positive no-debt/version-skew fixture yields correct required generator commands; negative existing-ratchet/new-offender regression demonstrates failure before and after advised regeneration.
> 
> ## Testing Requirements
> - Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
> - Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
> - Commands: go test ./internal/gates -run 'VersionSkew|Regen|Ratchet'; CLI integration on isolated ratchet design with a genuine new boundary violation.
> - No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.
> 
> ## OUT OF SCOPE
> - Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
> - Global heavy preflight, main merge and local release binary belong to final epic gate.
> 
> ## DIFF BUDGET
> - ~2-3 files, under 300 changed LOC; material overrun requires PM investigation, not weakened requirements.
> 
> ## MANDATORY SKILLS
> - developer for implementation; codebase-memory for discovery; pm_acceptor for independent acceptance.
> 
> ## Delivery Requirements
> Use pvg story deliver, never close. Append RED/GREEN SHAs, exact commands and outputs, test inventory, independent proof per AC, and any residual limits to shared nd. Do not use pushing pvg story merge.
> 
> ## nd_contract
> status: new
> 
> ### evidence
> - Created 2026-09-05; source signatures verified at assessment base 497419ab4512fcff765cd5feb27aed4c67b5608d.
> 
> ### proof
> - [ ] AC #1: independently verified
> - [ ] AC #2: independently verified
> - [ ] AC #3: independently verified
> - [ ] AC #4: independently verified


## History
- 2026-09-05T19:35:08Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:15Z dep_added: blocks MAC-ou97
- 2026-09-05T21:11:54Z status: open -> in_progress
- 2026-09-05T21:11:54Z auto-follows: linked to predecessor MAC-olrx
- 2026-09-05T21:11:54Z claimed by dev-MAC-a89e
- 2026-09-05T21:23:23Z status: in_progress -> in_progress

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-gcrr]], [[MAC-ou97]]
- Follows: [[MAC-olrx]]

## Comments

### 2026-09-05T21:14:38Z ramirosalas
TEST-EDIT AUTHORIZED: internal/gates/gates_test.go -- pre-RED repair limited to TestVersionSkewNoteNamesEveryApplicableCommand. Replace the expected "machinery baseline <design> --impl <dir>" regeneration command with an explicit assertion that the note does not contain "machinery baseline"; preserve the ratchet-present fixture and all existing oracle, Alloy, verify-formal, and pack generator expectations. Its present baseline expectation directly contradicts AC1 and encodes the unsafe behavior being removed. The repair commit subject must contain both tdd-red and [test-edit-authorized]. No other existing tests or assertions are authorized for modification by this decision.

Evidence: read pvg issues show MAC-a89e --json; inspected committed main 497419ab4512fcff765cd5feb27aed4c67b5608d and epic f24b2df3cb1e1521f97b406a7516f72bb7bc7890. The reviewed gates_test.go and cmd/machinery/baseline.go are identical between these refs. gates.go regenCommands currently appends baseline solely when RatchetFile exists. No uncommitted author work inspected, tests replayed, source files edited, or story status/labels changed.

AC3 SCOPE GAP: Current baseline help advises review of pasted dependency rules and rerunning after burning down debt, but does not explain that rerunning can accept newly added offender files by replacing ratchet.json. The zero-proposed-rules path prints "the contract already covers every observed edge; nothing new to baseline", then publishes the new ratchet and declares it armed, without debt-change review guidance. AC3 therefore cannot be proven by pinning the current help text.

Required bounded Sr PM ownership clarification: include cmd/machinery/baseline.go for help/output messaging and directly associated regression coverage. State explicitly that baseline is a deliberate debt-acceptance operation, rerunning may expand accepted offenders, and users must review ratchet.json/offender changes before adopting them. Successful output must convey this even when no dependency rule is proposed; help should explain it before invocation. Preserve explicit baseline functionality and do not add automatic migration, a version stamp, or a new confirmation protocol. A real CLI RED case that deliberately invokes baseline on an existing edge with a new offender, demonstrates the changed ratchet/accepted offender, and requires clear review guidance covers the actual missing behavior. This note does not authorize source edits outside current story ownership; Sr PM must repair that scope first.

This is narrow test-edit authorization and AC3 scope analysis only. It is NOT RED approval, delivery acceptance, rejection, or a status transition. Author must produce genuine assertion-failure RED proof and passing controls, followed by a full independent RED review before GREEN.

## nd_contract
status: in_progress

### evidence
- Independent pre-RED dispute review against committed main/epic source; narrowly authorized test expectation repair as recorded above.
- AC3 help/output ownership gap reported to dispatcher for Sr PM clarification.

### proof
- [x] AC #1: existing baseline-positive expectation is contradicted by the criterion; bounded replacement authorized.
- [ ] AC #2: new real CLI before/after regeneration proof awaits RED author and independent review.
- [ ] AC #3: explicit debt-change review guidance requires the bounded scope repair above.
- [ ] AC #4: full RED proof and passing controls await independent review.

## AUTHORITATIVE MAC-a89e SCOPE REPAIR — 2026-09-05
Current canonical Description includes the concrete AC3 baseline debt-review behavior and supersedes only conflicting earlier ownership/AC3/budget wording. Previous description, PM authorization, notes and history remain preserved.

Bug-triage disposition: absorb the newly confirmed explicit-baseline messaging gap into the existing P0 MAC-a89e under MAC-ui8a. It is already required by AC3 and concerns the same distinction between routine regeneration and deliberate debt acceptance. No new issue, dependency, claim or status/label transition.

Scope delta: cmd/machinery/baseline.go is now explicitly owned only for help and successful-output guidance. internal/gates/gates_test.go is owned only for the PM-authorized TestVersionSkewNoteNamesEveryApplicableCommand baseline-positive expectation replacement with baseline absence, preserving its ratchet fixture and all other generator assertions. Existing gates.go and new regeneration_safety_test.go remain owned. Total four files; original under-300-LOC ceiling retained.

AC3 observable contract: help explains baseline reruns rewrite ratchet.json and can accept new offender files; successful output tells users to review ratchet/offender changes before adoption even if no dependency rule was proposed. Zero proposed rules does not mean no debt change. The real isolated CLI test must demonstrate an already-baselined edge failing after a new offender, explicit baseline successfully accepting that offender with zero rule proposals, and the required help/output guidance. Deliberate baseline acceptance remains allowed. No confirmation prompt, stamp, algorithm/schema rewrite, or Paivot runtime coupling.

Evidence: full latest MAC-a89e story and independent PM comment read; graph-first exact source for regenCommands, VersionSkewNote, BuildBaseline, newBaselineCmd and TestVersionSkewNoteNamesEveryApplicableCommand. Source confirms baseline is included when RatchetFile exists, existing test requires it, BuildBaseline re-snapshots current offenders on baselined edges, and CLI zero-rule path claims nothing new before writing the expanded ratchet. Graph coverage 2026-09-05T20:28:41Z reported metadata_match/no_recorded_issue for four cited source/test files; exact snippets were read. No author worktree/source/frozen tests inspected or modified and no runtime test replay performed during this tracker-only scope repair.

The existing 2026-09-05T21:14:38Z TEST-EDIT AUTHORIZED comment remains the authority for that narrow existing-test change and its tdd-red/[test-edit-authorized] development commit markers. This scope repair is not RED approval, delivery or acceptance. Parent will lint and notify the author; full independent RED replay is still required before GREEN.

## nd_contract
status: in_progress

### evidence
- Canonical self-contained description repaired through supported pvg nd edit; append-only scope note written with pvg nd update --append-notes.
- P0 MAC-a89e retains dev-MAC-a89e, hard-tdd, parent MAC-ui8a and existing blocks MAC-gcrr/MAC-ou97.
- Four-file scope with under-300-LOC budget; all previous notes/history/comments/contracts preserved.

### proof
- [x] Scope repair: AC3 explicitly covers help and successful zero-rule output for ratchet/offender debt expansion.
- [x] Scope repair: narrow existing-test authorization embedded without enlarging its allowed assertion changes.
- [ ] AC #1-4: RED execution, independent approval, implementation and acceptance remain pending.


### 2026-09-05T21:18:55Z ramirosalas
TEST-EDIT AUTHORIZED: internal/gates/regeneration_safety_test.go -- bounded pre-approval AC3 assertion repair against committed 9b85ef6 and f482c7683f5005e40722123e3a382e01b15219d2. In TestRegenRatchetRealCLI/explicit-baseline-reviews-debt-change, the existing whole-output strings.Contains(lower, "review") assertion is invalid: t.TempDir incorporates the subtest name containing "reviews", and baseline prints the resulting design path in its wrote-ratchet line. Existing "ratchet", "debt", and "offender" words elsewhere complete the false positive without any actual review instruction. The author-reported AC3 pass is NOT valid evidence.

Authorized repair: strengthen only the AC3 guidance proof and any small directly supporting matcher/sensitivity tests in this new regression file. Read baseline --help before deliberate baseline invocation and assert the canonical AC3 semantics: rerunning rewrites ratchet.json, may accept new offender files, and users must review those ratchet/offender changes before adopting them. Assert successful output carries that debt-change review guidance for the existing-edge/new-offender/zero-new-rule case and that its zero-rule message does not imply no accepted debt or no change. Verify zero proposed rules explicitly. Preserve the real successful baseline call, actual offender expansion, preceding G4 failure, subsequent G4 success, and absence-of-version-stamp checks. Preserve all unrelated generator/advice/no-debt/regeneration assertions and fixtures. No production edits are authorized in this RED amendment.

Robustness bar: inspect actual prose, excluding fixture paths from evidence and relating review guidance to ratchet/offender changes and the possibility of accepting new debt. Unrelated mentions of words across help/output must not suffice. Add focused passing sensitivity controls: genuine guidance must be recognized; the existing unsafe zero-rule output with a temporary pathname containing review/reviews and ratchet/debt/offender bait must be rejected; existing help that reviews pasted dependency rules but never warns about accepting added offenders must be rejected. The matcher should allow natural equivalent wording and whitespace/wrapping, without prescribing one exact production sentence. These controls supplement, never replace, the real CLI assertions.

Commit the repair with both literal tdd-red and [test-edit-authorized] in its subject. Preserve 9b85ef6 and f482c7683f5005e40722123e3a382e01b15219d2 in history; no amend/rebase/squash. Replay and record the scoped target, sensitivity-control results, genuine guidance assertion failures on unchanged production, passing controls, inventory/skips, coverage and exact SHA before delivery. Full independent RED review still required; this decision is not approve-red, acceptance, rejection or a status transition.

Scope evidence: read latest canonical MAC-a89e Description and authoritative Sr PM scope repair; the four-file ownership and expanded observable AC3 are now recorded. Reviewed committed regression source at f482c7683f5005e40722123e3a382e01b15219d2 lines 179-195, baseline help/output source, and the prior narrow gates_test.go diff, which exactly matches the earlier authorization. Shared provisional proof records 10 leaves / 7 pass / 3 intended failures / 0 skip and invalid AC3 pass. No retained raw command-output path is supplied there; I have not independently replayed that run or inspected a raw transcript. The precise false-positive mechanism is independently established from committed source, sufficient for this bounded repair authorization; raw replay evidence remains required for full RED review.

STATIC SCAN FALSE POSITIVE DISPOSITION: pvg verify flag at internal/gates/gates_test.go:125 is legitimate fixture behavior, not a stub. The actual committed helper is coveringInterfaceTable (not fixtureInterfaceRows). Lines 108-122 parse concrete allow edges and build Markdown rows; lines 124-125 return an empty string only when no concrete rows exist, correctly omitting an unnecessary interface-contract section. Lines 127-128 render a full table otherwise. c4GraphFixture line 145 includes that section in ARCHITECTURE.md then executes CheckC4; TestG2AllowGraphAcyclicity and TestG2TransitivePairsCount exercise the generated coherent graph fixtures. The helper predates this story and is unchanged in the two reviewed commits. Preserve it without removal, rewriting, disabled checks or a blanket scan waiver. Record this one explained scanner finding alongside verification evidence; the narrow gates_test.go amendment remains limited to TestVersionSkewNoteNamesEveryApplicableCommand.

## nd_contract
status: in_progress

### evidence
- Independent committed-source review authorized the bounded AC3 test strengthening and scoped sensitivity controls.
- Canonical AC3 ownership repair read; prior gates_test.go amendment matches authorization.
- One precise static scan false positive explained; raw RED output/replay remains for full review.

### proof
- [x] Amendment review: pathname false positive established and robust repair bounded.
- [x] Static disposition: conditional fixture section omission is implemented behavior, not a stub.
- [ ] AC #1-4: complete RED delivery and independent approval remain pending.
