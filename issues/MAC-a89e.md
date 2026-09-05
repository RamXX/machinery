---
id: MAC-a89e
title: "Keep regeneration advice from accepting new debt"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:27Z
created_by: ramirosalas
updated_at: 2026-09-05T21:16:47Z
content_hash: "sha256:1fabc3f28678f2b701a3f2ec06772cad963c4b4d6bb4fc21a73bca12f5a2f833"
blocks: [MAC-gcrr, MAC-ou97]
assignee: dev-MAC-a89e
follows: [MAC-olrx]
---

## Description
## USER INTENT
Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.

## Context (Embedded)
NEXT9: version-skew advice includes machinery baseline when ratchet exists. Baseline snapshots tolerated import offenders and can widen accepted architecture debt; ratchet has no version stamp. Routine regeneration must not silently authorize debt.

## Ownership
Own only these paths and directly associated tests: internal/gates/gates.go, internal/gates/regeneration_safety_test.go. You are not alone in this codebase; preserve other edits and coordinate any shared-file changes with dispatcher.

## Boundary Map
PRODUCES:
- internal/gates/gates.go -> hardened behavior and regression proof
- internal/gates/regeneration_safety_test.go -> hardened behavior and regression proof
CONSUMES:
- Existing Machinery source interfaces.
  spec: regenCommands(design string) []string

### Story Acceptance Criteria
1. Version-skew regeneration instructions never include machinery baseline or any debt-accepting mutation. Existing oracle/Alloy/formal/pack regeneration remains accurate and deterministic.
2. A design with ratchet and newly introduced offender continues failing architecture checks after following all routine regeneration instructions; regenerated stamps cannot accept the offender.
3. Explicit baseline remains a deliberate user-invoked operation with clear debt-change review guidance, not automatic migration. Do not add a version stamp as a substitute for preventing debt expansion.
4. Positive no-debt/version-skew fixture yields correct required generator commands; negative existing-ratchet/new-offender regression demonstrates failure before and after advised regeneration.

## Testing Requirements
- Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
- Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
- Commands: go test ./internal/gates -run 'VersionSkew|Regen|Ratchet'; CLI integration on isolated ratchet design with a genuine new boundary violation.
- No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.

## OUT OF SCOPE
- Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
- Global heavy preflight, main merge and local release binary belong to final epic gate.

## DIFF BUDGET
- ~2-3 files, under 300 changed LOC; material overrun requires PM investigation, not weakened requirements.

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

## History
- 2026-09-05T19:35:08Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:15Z dep_added: blocks MAC-ou97
- 2026-09-05T21:11:54Z status: open -> in_progress
- 2026-09-05T21:11:54Z auto-follows: linked to predecessor MAC-olrx
- 2026-09-05T21:11:54Z claimed by dev-MAC-a89e

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
