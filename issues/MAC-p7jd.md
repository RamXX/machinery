---
id: MAC-p7jd
title: "Invalidate reviews when implementation subjects change"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:33:46Z
created_by: ramirosalas
updated_at: 2026-09-05T23:09:26Z
content_hash: "sha256:ae51febaf41b5bbd6dc2caaa39ebb782bb52462fd60360ba1a63d4b0a9b55c82"
blocks: [MAC-vx24, MAC-gcrr, MAC-ou97]
assignee: dev-MAC-p7jd
follows: [MAC-p8ce]
---

## Description
## USER INTENT
Strengthen Machinery mission-critical assurance with observable fail-closed behavior and precise limits.

## Context (Embedded)
Assessment F3: gt.conformance-test-shape covers BUILD artifacts, while g4.pack-event-discipline covers pack. Neither binds tested/reviewed implementation. Historic Ga ancestor acceptance is valid history, not current-tree assurance. Preserve judgment-vs-mechanical distinction.

## Ownership
Own only internal/gates/attest.go, internal/gates/attest_implementation_test.go, cmd/machinery/attest.go, docs/attestation-evidence.md and directly associated tests. You are not alone: preserve other edits and coordinate shared paths.

## Boundary Map
PRODUCES:
- internal/gates/attest.go -> hardened contract and regression evidence
- internal/gates/attest_implementation_test.go -> hardened contract and regression evidence
- cmd/machinery/attest.go -> hardened contract and regression evidence
- docs/attestation-evidence.md -> hardened contract and regression evidence
CONSUMES:
- Existing Machinery implementation.
  spec: attestationRequiredPaths(g *Gate, design, claim string) []string; stableAttestationHashes(paths []string) ([]string, error)

### Story Acceptance Criteria
1. Implementation/test behavior claims bind a complete explicit implementation/test scope under rooted inventory and content hashes, not only BUILD or pack. Any code/test/config addition, removal, rename, content change or scope narrowing affecting the claim invalidates freshness.
2. Distinguish plan-only claims, current implementation review and historical milestone acceptance in closed schema, gate diagnostics and docs. Historical ancestor records remain historical; they cannot alone imply current implementation approval.
3. Provide explicit compatibility migration for existing attestations. Legacy design-only covers never quietly grandfather implementation assertions as fresh; users receive actionable missing-subject diagnostics.
4. Negative tests remove assertions after review, alter event handlers, add excluded files, change scope, alias paths/symlinks and replay stale attestations. Positive unchanged reviewed scope and harmless evidence-only commit remain usable.
5. Real CLI attest/check path exercises changed implementation and historical/current distinction. Hashing proves binding, not reviewer honesty or that tests executed; output never claims otherwise.

## Testing Requirements
- Explicit hard TDD: separate RED test author, expected behavioral assertion failure plus passing control on unmodified production, independent PM replay, frozen RED test/fixture bytes retained through GREEN. Compilation, import, timeout or unavailable infrastructure are not valid RED. Existing tests encoding the unsafe contract may be corrected during reviewed RED; GREEN cannot silently weaken them.
- Integration tests: MANDATORY (no mocks). Exercise real process/filesystem/runtime boundaries; no stubs, no skip-if-missing. Fixture policy inputs may test parser logic but are not live remote proof.
- go test ./internal/gates -run Attest; go test ./cmd/machinery -run Attest; CLI integration with real temp git repository and design+implementation roots.
- Full scripts/preflight.sh ONLY at end of epic. No pushes, sync, remote mutation, installed binary/plugin/agent/skill replacement, or dev-link. Another agent uses installed Machinery in NIL; builds/tests use isolated output and homes.
- Shipped product enforcement is standalone Machinery: never requires pvg, nd, Paivot labels/metadata or commit conventions. Paivot is local development coordination only.

## OUT OF SCOPE
- Other assessment areas have sibling stories; final preflight/local main merge/isolated candidate binary are final gate responsibilities. Do not omit small directly related safety fixes.

## DIFF BUDGET
- ~4-7 files, under 1000 changed LOC; overrun triggers PM investigation rather than weaker proof.

## MANDATORY SKILLS
- developer; codebase-memory; pm_acceptor.

## Delivery Requirements
Append exact RED/GREEN SHAs, commands/output, ownership inventory and per-AC proof. Use pvg story deliver; PM accepts separately. Never invoke pushing pvg story merge.

## nd_contract
status: new

### evidence
- Created 2026-09-05 from assessment and source-verified interfaces.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified

## Acceptance Criteria


## Design


## Notes
ANCHOR ROUND-1 RUNTIME CLASSIFICATION: This story's required current cases are service-free Go/native filesystem/local-process tests and real local CLI paths; no Docker/Java/Node dependency is implicit in ordinary native suites. Preserve actual non-mocked path tests. If implementation introduces any external runtime-backed case, it MUST add a dedicated closed fragment under testdata/integration-lanes via the shared required lane and declare ownership/dependency before delivery; no skip-if-missing, env-gated omission or reliance on later incidental execution. Missing service cannot silently convert required coverage to success.

## History
- 2026-09-05T19:35:07Z dep_added: blocks MAC-vx24
- 2026-09-05T19:35:08Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:16Z dep_added: blocks MAC-ou97
- 2026-09-05T23:09:26Z status: open -> in_progress
- 2026-09-05T23:09:26Z auto-follows: linked to predecessor MAC-p8ce
- 2026-09-05T23:09:26Z claimed by dev-MAC-p7jd

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-gcrr]], [[MAC-ou97]]
- Follows: [[MAC-p8ce]]

## Comments
