---
id: MAC-p8ce
title: "Bind consumer READS to each event edge"
status: open
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:26Z
created_by: ramirosalas
updated_at: 2026-09-05T19:33:48Z
content_hash: "sha256:3816a569e1660cb8b2e6ad67bfea4ae8a5d452b779e48c1738ec04021adcbf34"
blocks: [MAC-gcrr, MAC-ou97]
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

## Acceptance Criteria
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

## History
- 2026-09-05T19:35:08Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:15Z dep_added: blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-gcrr]], [[MAC-ou97]]

## Comments
