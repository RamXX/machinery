---
id: MAC-nf0m
title: "Current attestation: per-claim scope derived by machinery (attestor cannot narrow), so unrelated commits do not stale the row"
status: open
priority: 2
type: feature
labels: [gv, attest, h2]
created_at: 2026-09-25T19:38:38Z
created_by: ramirosalas
updated_at: 2026-09-25T19:38:38Z
content_hash: "sha256:7fe9906482ced315f13c89e68bbaf1f9c80e6cded4abba6b36d9de45ea5a5e6b"
blocked_by: [MAC-6mzy]
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


## History
- 2026-09-25T19:38:38Z dep_added: blocked_by MAC-6mzy

## Links
- Blocked by: [[MAC-6mzy]]

## Comments
