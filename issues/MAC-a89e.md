---
id: MAC-a89e
title: "Keep regeneration advice from accepting new debt"
status: open
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:27Z
created_by: ramirosalas
updated_at: 2026-09-05T19:33:48Z
content_hash: "sha256:f422eae5b84007a711e187886248ffd2084f3ec956f0be942aa163e4ed5e9b0f"
blocks: [MAC-gcrr, MAC-ou97]
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

## Acceptance Criteria
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

## History
- 2026-09-05T19:35:08Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:15Z dep_added: blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-gcrr]], [[MAC-ou97]]

## Comments
