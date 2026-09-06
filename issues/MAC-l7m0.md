---
id: MAC-l7m0
title: "Define standalone executable test-assurance contract"
status: blocked
priority: 1
type: task
parent: MAC-ui8a
created_at: 2026-09-05T19:35:06Z
created_by: ramirosalas
updated_at: 2026-09-06T08:47:43Z
content_hash: "sha256:d4244516b254d7a05ce468e6add52130548f9be48732c6aca52da14161096c9c"
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

### Story Acceptance Criteria
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

## Acceptance Criteria


## Design


## Notes
BLOCKED — USER CHOICE / ARCHITECTURE REVIEW. Do not dispatch a generic developer. Pending user decisions: supported initial native runner languages and trusted-host versus adversarial-code execution boundary. Independent architect owns exact contract. Only after answers plus reviewed contract may Sr PM repair implementation interfaces and release this blocker.
ANCHOR ROUND-1 RUNTIME CLASSIFICATION: Architecture is read-only contract work, not an executable runtime suite. Its accepted contract must explicitly identify any native adapter runtimes consumed by implementation; MAC-vx24 must register/provision those using the required closed integration lane. No execution assurance may be claimed from schema or architectural review alone.

## History
- 2026-09-05T19:35:06Z dep_added: blocks MAC-vx24
- 2026-09-05T19:36:16Z dep_added: blocks MAC-ou97
- 2026-09-05T19:36:47Z status: open -> blocked

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-ou97]]

## Comments

### 2026-09-06T08:47:43Z ramirosalas
AUTHORITATIVE SCOPE REVISION 2026-09-06 — approved architecture landing only
This append-only revision supersedes earlier unresolved-user-choice wording for this story without erasing it. Root confirms user choices and independent CHALLENGE-3 approval (zero blocking findings). Architecture approval is NOT implementation, native feasibility proof or permission to dispatch unreviewed successor stories. Root's backlog-review hold remains until independent Anchor approval.

USER INTENT: land one permanent exact standalone Machinery contract so bounded implementation stories consume reviewed schemas/APIs rather than a temporary proposed report.

PRODUCES:
- docs/test-assurance-contract.md -> exact 589-line v3-final architecture bytes, SHA256 e467a3b6c6f657e7a4cb28e85c688be0df24c56a7e3c76fdcfa224f7ff5d2624
- docs/test-assurance-review.md -> exact 32-line independent CHALLENGE-3 bytes, SHA256 0205b50a8eb2e135230173bbf4a7b7eccdde5934fabab5aa4b2dde95d474ac66
- docs/test-assurance-status.md -> short canonical approval/status index: approved architecture pair above, no implementation/native proof yet; its index explicitly resolves historical PROPOSED wording in the immutable source without rewriting that source
CONSUMES:
- Approved immutable architecture and independent review provided by root.
  source: /tmp/machinery-assurance-architecture.fXW3vg/test-assurance-contract-v3-final.md and CHALLENGE-3.md; verified hashes above.

AC A1: copy the complete architecture/review to owned canonical paths unchanged; exact hashes and line counts match. Status index links both and says review-approved contract, not software acceptance or execution evidence.
AC A2: preserve four first-release closed native adapters and exact catalogs: Go1.27.1; Node26.8.1+TypeScript7.0.2; CPython3.14.7; Elixir/Mix/ExUnit1.20.4+OTP29.0.6/ERTS17.0.6, plus scoped gate Git2.55.0; supported native Linuxamd64/Darwinarm64. No arbitrary shell adapters, Paivot dependency, audit exception or installed replacement.
AC A3: validate retained exact CLI/API/schema/root topology/external-store/explicit-register/expected-head CAS/empty-head/history/failed-RED contracts and entire lifecycle/budget text; no omission or semantic paraphrase.
AC A4: status index carries all three review advisories as unfulfilled implementation obligations: actual two-platform custody/assertion proof; real first-use init/scaffold/capture/register/RED/GREEN/verify/complete without prior PASS receipt; actual process-producing call-graph audit plus cumulative deadlines.
AC A5: no source/test/runtime/CI/installed edits under this documentation-only story. Preserve accepted p7 full-root/Gv custody and legacy frozen tests/96-case hpqp union; new implementation requires separately reviewed stories. No heavy preflight; hash/link/doc checks only.
AC A6: independent PM verifies hashes and canonical references, then delivers/accepts documentation through normal local workflow. No contract imported as a pass result.

DIFF BUDGET: 3 documentation files, ~650 lines. MANDATORY SKILLS: developer for literal landing, pm_acceptor for independent hash/link review. No hard-TDD label is required for immutable documentation landing.

## nd_contract
status: new

### evidence
- Sr PM fully read all589architecture lines and all32review lines; SHA256 and line counts independently matched on2026-09-06.
- Canonical tracker body read completely before this true-EOF scope revision; no old contract overwritten.

### proof
- [x] Approved immutable sources and exact landing scope identified.
- [ ] AC A1-A6: canonical docs landed and independently accepted.

