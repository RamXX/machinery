---
id: MAC-l7m0
title: "Define standalone executable test-assurance contract"
status: blocked
priority: 1
type: task
parent: MAC-ui8a
created_at: 2026-09-05T19:35:06Z
created_by: ramirosalas
updated_at: 2026-09-05T19:36:50Z
content_hash: "sha256:6f8f4cf1505ed859f54b01516d0a7f756edb3334228fc572a741526b7ae5b714"
blocks: [MAC-vx24, MAC-ou97]
---

## Description
## USER INTENT
Provide strongest honest standalone Machinery guarantees for LLM-generated software, including test sensitivity to unsafe behavior.

## Context (Embedded)
New mechanically enforced negative hard-TDD is requested, but no shipped runtime protocol currently exists. Existing buildplan.go only checks narrative section names. Architecture author is independently assessing the closed runner adapter and trust boundary; do not invent interfaces before approval. Product must never depend on Paivot.

## Ownership
docs/test-assurance-contract.md. Not alone in codebase; preserve other edits. Newly listed files are explicitly owned outputs, not pre-existing interfaces.

## Boundary Map
PRODUCES:
- docs/test-assurance-contract.md -> approved closed protocol
CONSUMES:
- Existing Machinery source.
  source: internal/gates/buildplan.go CheckBuildPlan(design string) *Gate; gate suite and hook contracts.

### Acceptance Criteria
1. Resolve and record user choices on initial native runner languages and trusted-host versus hostile-code boundary. Record approved concrete CLI, versioned closed manifest/evidence schemas, limits, error statuses, activation/compatibility and ownership before implementation dispatch.
2. Specify Machinery-captured or independently replayed real native test outcomes on immutable snapshots, expected assertion witnesses (native nonzero alone is insufficient), named passing control, no empty/skip/xfail/build/import/panic/timeout/malformed/truncated/fabricated-summary success.
3. Bind exact bytes and complete inventories of tests/helpers/fixtures/runner config/dependency locks; prohibit tokens-equal/whitespace equivalence for frozen evidence. Unsafe challenge variants may alter implementation only, and every required negative behavior needs sensitivity proof against a reviewed unsafe variant, not merely a passing health control.
4. Require replay of retained RED/challenge snapshots plus current GREEN before strong final assurance. Static receipt hashes prove binding, not execution/authenticity. Cheap check/hook freshness must explicitly state replay not performed; historical milestone acceptance remains distinct.
5. Make protocol standalone Machinery only; ordinary Git snapshots are allowed, Paivot metadata/labels/commands are not. Define current implementation staleness, runtime residual obligations, schema migration and unsupported-runner behavior failclosed.
6. Independent architecture review approves contract and dispatcher copies exact new field/API names into blocked implementation story before it is executable.

## Testing Requirements
Read source-verified APIs; adversarial scenario table and standalone contract review only. No production edits or heavy preflight. Architecture task has no RED because it resolves contracts rather than executable behavior.
Integration tests: MANDATORY (no mocks) for executable implementation, no skip-if-missing. Architecture review is not execution proof.
No push/sync/GitHub mutation. No installed binary/skills/plugins/agents replacement or dev-link. Isolated candidate only. Full scripts/preflight.sh only final epic gate.

## OUT OF SCOPE
- Mathematical proof of arbitrary test semantic adequacy or malicious host safety beyond explicitly approved trust boundary: expose these as residual limits, never silently claim them.
- Any Paivot runtime dependency: prohibited by user.

## DIFF BUDGET
- 1 document, under 700 lines.

## MANDATORY SKILLS
- architect for contract authoring/review; developer and pm_acceptor for implementation; codebase-memory for source discovery.

## nd_contract
status: new

### evidence
- Created 2026-09-05 from user requirements and independent read-only design analysis.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified
- [ ] AC #6: independently verified

Observable outcome: the maintainer can review a closed standalone contract that displays its trust boundary and blocks unapproved implementation.

## USER INTENT
Provide strongest honest standalone Machinery guarantees for LLM-generated software, including test sensitivity to unsafe behavior.

## Context (Embedded)
New mechanically enforced negative hard-TDD is requested, but no shipped runtime protocol currently exists. Existing buildplan.go only checks narrative section names. Architecture author is independently assessing the closed runner adapter and trust boundary; do not invent interfaces before approval. Product must never depend on Paivot.

## Ownership
docs/test-assurance-contract.md. Not alone in codebase; preserve other edits. Newly listed files are explicitly owned outputs, not pre-existing interfaces.

## Boundary Map
PRODUCES:
- docs/test-assurance-contract.md -> approved closed protocol
CONSUMES:
- Existing Machinery source.
  source: internal/gates/buildplan.go CheckBuildPlan(design string) *Gate; gate suite and hook contracts.

## Acceptance Criteria
1. Resolve and record user choices on initial native runner languages and trusted-host versus hostile-code boundary. Record approved concrete CLI, versioned closed manifest/evidence schemas, limits, error statuses, activation/compatibility and ownership before implementation dispatch.
2. Specify Machinery-captured or independently replayed real native test outcomes on immutable snapshots, expected assertion witnesses (native nonzero alone is insufficient), named passing control, no empty/skip/xfail/build/import/panic/timeout/malformed/truncated/fabricated-summary success.
3. Bind exact bytes and complete inventories of tests/helpers/fixtures/runner config/dependency locks; prohibit tokens-equal/whitespace equivalence for frozen evidence. Unsafe challenge variants may alter implementation only, and every required negative behavior needs sensitivity proof against a reviewed unsafe variant, not merely a passing health control.
4. Require replay of retained RED/challenge snapshots plus current GREEN before strong final assurance. Static receipt hashes prove binding, not execution/authenticity. Cheap check/hook freshness must explicitly state replay not performed; historical milestone acceptance remains distinct.
5. Make protocol standalone Machinery only; ordinary Git snapshots are allowed, Paivot metadata/labels/commands are not. Define current implementation staleness, runtime residual obligations, schema migration and unsupported-runner behavior failclosed.
6. Independent architecture review approves contract and dispatcher copies exact new field/API names into blocked implementation story before it is executable.

## Testing Requirements
Read source-verified APIs; adversarial scenario table and standalone contract review only. No production edits or heavy preflight. Architecture task has no RED because it resolves contracts rather than executable behavior.
Integration tests: MANDATORY (no mocks) for executable implementation, no skip-if-missing. Architecture review is not execution proof.
No push/sync/GitHub mutation. No installed binary/skills/plugins/agents replacement or dev-link. Isolated candidate only. Full scripts/preflight.sh only final epic gate.

## OUT OF SCOPE
- Mathematical proof of arbitrary test semantic adequacy or malicious host safety beyond explicitly approved trust boundary: expose these as residual limits, never silently claim them.
- Any Paivot runtime dependency: prohibited by user.

## DIFF BUDGET
- 1 document, under 700 lines.

## MANDATORY SKILLS
- architect for contract authoring/review; developer and pm_acceptor for implementation; codebase-memory for source discovery.

## nd_contract
status: new

### evidence
- Created 2026-09-05 from user requirements and independent read-only design analysis.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified
- [ ] AC #6: independently verified

## Acceptance Criteria


## Design


## Notes
BLOCKED — USER CHOICE / ARCHITECTURE REVIEW. Do not dispatch a generic developer. Pending user decisions: supported initial native runner languages and trusted-host versus adversarial-code execution boundary. Independent architect owns exact contract. Only after answers plus reviewed contract may Sr PM repair implementation interfaces and release this blocker.

## History
- 2026-09-05T19:35:06Z dep_added: blocks MAC-vx24
- 2026-09-05T19:36:16Z dep_added: blocks MAC-ou97
- 2026-09-05T19:36:47Z status: open -> blocked

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-ou97]]

## Comments
