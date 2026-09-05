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
updated_at: 2026-09-05T23:17:15Z
content_hash: "sha256:d73da6bef415437eb7f10396a0098563f9ed9ccb67ecf3543e77af21f0056965"
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
DISPATCHER CONTRACT HOLD: RED author verified clean story/MAC-p7jd at a82277a and stopped before source/test edits, tests, commits or delivery. Existing closed v1 attestation format and design-only CheckAttestations interface cannot express the implementation scope promised by AC1/2/5. Sr PM is reviewing the smallest source-verified contract/ownership repair; schema/CLI choices and suite.go wiring need review before RED. Healthy retained worktree and claim preserved; no recovery cleanup. User-pending strict execution containment policy is separate and must not be silently imposed on freshness binding.
SR PM SAME-STORY PRE-RED ARCHITECTURE TASK (MAC-p7jd; scope proposal, not implementation or TEST-EDIT authorization)

Disposition: retain this P0 repair under MAC-ui8a, with all five canonical AC unchanged. The paused RED author has no authorized schema/API to test yet. Route a bounded read-only technical contract author within this story, followed by independent contract review and explicit PM RED/test-edit authorization. Do not create a prerequisite or couple MAC-l7m0: review freshness is distinct from authenticated execution, containment, or reviewer honesty. Existing user requirements suffice; no new product question is identified. No status/label/assignee/worktree change is requested.

Verified source boundary:
- internal/gates/attest.go: CheckAttestations(design string) *Gate has no implementation-root input. attestationRequiredPaths(g *Gate, design, claim string) []string binds gt.conformance-test-shape to BUILD artifacts and g4.pack-event-discipline to pack. checkAttestationFreshness(g *Gate, design string, row attestRow) hashes only design-relative covers. The closed v1 root/row/cover schema has no implementation scope or plan/current/history field; cover keys are path/hash, with sha256: plus 64 lowercase hexadecimal digits.
- cmd/machinery/attest.go: stableAttestationHashes(paths []string) ([]string, error) hashes explicit files only; newAttestCmd offers positional paths and --claims, not scope generation. It opens all files, rejects identity/path aliases, reads and revalidates all before emitting output. Its helper is cmd/machinery/scale.go: openStableRegular(path string) (*stableRegularFile, error), read and revalidate. It rejects nonregular/symlink leaves and oversize files, compares opened identity, and rechecks identity/metadata/content. These are constraints to preserve, not proof of complete rooted directory custody.
- internal/gates/suite.go already accepts impl in (*Snapshot).RunSelected(impl string, sel Selection, opt RunOptions) []*Gate and materializes it through (*designlock.Lock).MaterializeExternalTree(path string) (*ExternalTreeSnapshot, error), then passes the stable path into runSelectedInSnapshot. Gv discards that input by calling CheckAttestations(design). cmd/machinery/check.go already sends --impl through SelectRunAndNote; its help currently describes G4/Gt. Scope/API review must use the held snapshot rather than reopen an ambient implementation root.
- internal/designlock/external_snapshot.go materialization uses held no-follow rooted copy plus tracked pre/post fingerprints for external trees and retained design source paths for in-design trees. Its existing inclusion/exclusion rules and limits need explicit comparison with the new attested-scope contract; mere reuse does not establish that they cover every AC1 subject.
- Exact epic a82277a suite.go was read. Accepted MAC-p8ce parent-owned relational Gt selection changes are present and must be preserved. A bounded Gv wiring change must not revert them.
- internal/gates/attest_test.go currently requires rejection of version 2. If the reviewed contract chooses a version bump, identify that exact legacy assertion for independent PM amendment; do not treat compilation failure or unreviewed test weakening as RED.

Required contract-author deliverable (proposal in task handoff/shared story note; no source, tests, or docs edits in this read-only phase):
1. Exact closed schema/version decision and representative valid/invalid examples for plan-only judgment, current implementation review, and historical milestone acceptance; claim classification, duplicate/unknown/wrong-type rejection, legacy migration, missing-subject diagnostics, and current-vs-history gate outcomes.
2. Exact public/internal API signatures and root authority flow from real CLI/check and shared suite callers through the held design/implementation snapshot. Specify absent --impl, ordinary versus --complete behavior, logical portable paths versus private snapshot paths, and compatibility for existing direct callers.
3. Complete explicit inventory authority and deterministic digest format: which code/test/config paths and types participate; additions/removals/renames/content changes; ignored/untracked files; narrowly justified evidence-only/generated exclusions; scope-policy binding and anti-narrowing; path ordering/normalization and alias/symlink/race/limit failures. Exclusions cannot create a loophole where changed behavior remains current, and whole-HEAD binding cannot invalidate harmless evidence-only commits. Explain avoidance of self-referential attestation hashes.
4. Exact CLI generation syntax, output schema and failure/partial-output semantics while preserving existing explicit-file hash and --claims behavior where compatible. State accurately that hashes bind observed subjects, not reviewer honesty or test execution.
5. Precise Ga-history/Gv-current interaction and compatibility migration, with historical ancestor acceptance retained as history but insufficient for present approval.
6. A concrete per-AC real filesystem/git/CLI test matrix with matched valid controls and intended rejection categories: unchanged scope and evidence-only commit pass; removed assertion, changed handler, added code/test/config (including excluded-file challenge), deleted/renamed file, scope narrowing, stale replay and path alias/symlink changes fail. Separate parser/schema proof from freshness proof; specify deterministic custody tests using existing permitted seams only after PM review. No mocks, skip-if-missing, Docker/Java/Node requirement, or authenticated-execution claim is introduced.

Proposed implementation ownership for subsequent review ONLY: retain current attest.go, attest_implementation_test.go, CLI attest.go, attestation-evidence.md and exact associated tests; add internal/gates/suite.go solely for reviewed Gv root/API wiring and cmd/machinery/check.go only for required attestation-facing CLI diagnostics/help/wiring, with named associated tests if needed. scale.go and internal/designlock remain consumed/read-only unless the author demonstrates a specific necessary gap and receives a separate scope review. Final API/schema, exact test-edit list and ownership must be canonicalized and independently reviewed before RED resumes. Original ~4-7 files/<1000 changed LOC is an estimate, not permission to trim proof: contract author must give a realistic revised file/LOC estimate if bounded wiring/schema tests exceed it; report actual cost later.

Evidence/custody: initial instruction/context checks, shared pvg issues show, graph-first symbol/call discovery plus exact source reads; graph coverage generation 2026-09-05T23:08:53Z reported metadata_match/no_recorded_issue for nine cited paths, best-effort only. Exact git show a82277a:internal/gates/suite.go verified epic wiring and preserved predecessor behavior. Earlier nonexistent guessed helper path was a reported lookup error, corrected by graph discovery to scale.go; it is not a product defect. No new experimental parser/freshness bypass, tests, implementation or delivery is claimed.

## nd_contract
status: in_progress

### evidence
- Bounded same-story contract triage completed; RED remains intentionally paused before edits at its healthy retained a82277a worktree.
- Scope/API/schema proposal awaits read-only contract author and independent review. No status, labels, claim, dependency or worktree changes made by this triage.

### proof
- [ ] AC #1: implementation scope/inventory contract and RED/GREEN proof pending
- [ ] AC #2: plan/current/history schema and diagnostics contract pending
- [ ] AC #3: explicit migration and missing-subject proof pending
- [ ] AC #4: matched positive/negative real custody tests pending
- [ ] AC #5: real CLI freshness/history proof and accurate limits pending

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
