---
id: MAC-nf0m
title: "Current attestation: per-claim scope derived by machinery (attestor cannot narrow), so unrelated commits do not stale the row"
status: open
priority: 2
type: feature
labels: [gv, attest, h2]
created_at: 2026-09-25T19:38:38Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:52Z
content_hash: "sha256:37d7d58e90ff36f06bca8ee6b6dbfd0ac3456c19f4ec7d79ef79094d5afc3168"
blocked_by: [MAC-6mzy]
related: [MAC-zlbk]
---

## Description
Follow-up to MAC-6mzy. Even with a stable, compact full-root row, every commit anywhere in the tree stales every current row, so re-attestation becomes a ritual (see MAC-zlbk) and "current" degrades into "someone ran the command". Owner decision 2026-09-25: solve it with scope DERIVED by machinery per claim kind, never scope chosen by the attestor.

Proposal: each Gv claim kind declares a derivation rule that machinery evaluates from the design and implementation, e.g. gt.conformance-test-shape = the test files Gt binds for the design's oracle ids + the oracle registry + the design covers; the row binds exactly that derived set (plus the committed exclusion list from MAC-6mzy). The attestor cannot add, remove, or narrow entries. A claim kind with no derivation rule stays full-root.

Acceptance criteria:
1. Derivation rules are code, per claim kind, closed; an unknown kind falls back to full-root, never to an empty scope.
2. A commit touching only files outside the derived set leaves the row current; a change inside it stales the row.
3. A change that alters the derived set itself (new bound test file, new oracle id) stales the row.
4. The gate recomputes the derived set; a row whose recorded set differs from the recomputed one fails (no attestor narrowing).
5. Output and docs state the proof scope of a derived row versus a full-root row; CHANGELOG notes it.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Implement closed per-claim-kind derivation rules evaluated by machinery, gate recomputation, and docs on proof scope. Evidence: No derived per-claim scope in internal/gates/attest.go or CHANGELOG through 0.11.0; current rows still bind the full root. Prerequisite MAC-6mzy (git-tracked inventory, exclusion list, compact manifest) is still open. Notes: Real dependency: builds on the committed exclusion list and manifest from MAC-6mzy. Addresses re-attestation ritual cost; related MAC-zlbk. Note the risk: scope derivation is a soundness-sensitive design change, so the unknown-kind-to-full-root rule must hold.

## History
- 2026-09-25T19:38:38Z dep_added: blocked_by MAC-6mzy

## Links
- Blocked by: [[MAC-6mzy]]
- Related: [[MAC-zlbk]]

## Comments
