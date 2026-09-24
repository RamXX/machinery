---
id: MAC-6mzy
title: "Scoped current attestation: bind a Gv current row to its subject set, not the whole tree (reverses MAC-p7jd ruling)"
status: open
priority: 2
type: feature
labels: [gv, attest, h2, needs-owner-decision, from-next]
created_at: 2026-09-24T21:32:49Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:49Z
content_hash: "sha256:a82cad47c46994677452744598a365a2a2d4c60720cb701122876603463aa174"
related: [MAC-38er]
---

## Description
Scoped current attestation: bind a Gv current row to its subject set, not the whole tree

Problem (H2, 2026-09-20, v0.8.0, M2 seal): the design protocol re-attests `gt.conformance-test-shape` as `kind: current` after implementation review. A current row carries the `full-root-v1` manifest: every regular file under the implementation root with mode, size and hash; 3117 entries and about fifteen thousand lines of `attestations.yaml` for a 3000-file tree. The row goes STALE when any file anywhere in the root is added, removed or changed.

Consequences:
1. Checkout and CI container never hold the same tree: the checkout has ignored files (`.env.local`, editor and tool caches, provider state); the container drops paths the project's Dagger module ignores (here `.claude` and `.vault`, which git tracks). A row generated in either is STALE in the other. H2 invented a "wall-shaped clone" (fresh clone minus the container ignore list) and moved every `machinery check --impl` and `machinery attest --impl` into it.
2. Every commit, even a status-ledger or plan-file line, stales the row, so each commit ends with a regeneration and amend: a fifteen thousand-line diff per commit for an unchanged judgment, and history grows by the manifest each time.
3. The row does not say which files are the SUBJECT of the claim. For `gt.conformance-test-shape` the subject is the conformance suites and the oracle registry; the manifest binds the README, the infrastructure module and the plans directory with equal weight.

Evidence: internal/gates/attest.go:718 accepts only `policy == "full-root-v1"`; attest.go:874 hard-codes it. MAC-p7jd (closed) deliberately specified "policy must equal full-root-v1. There are no user include, exclude, extension, gitignore or subtree selectors", so this is a policy extension that must preserve MAC-p7jd's guarantees for rows that keep `full-root-v1`. `machinery attest` has no `--ignore` or scope flag (cmd/machinery/attest.go:92-99).

Proposed fix:
- A scoped policy beside `full-root-v1`: the attestor names subject globs (for example `test/conformance/**`, `test/support/**`, the oracle registry) plus the design covers; the gate binds those files only. Policy name and globs live in the row so the scope is reviewable.
- An `--ignore` list for generation and checking that takes the same patterns a CI export ignores, so checkout and container agree without a clone.
- A compact manifest form (one line per entry, or a Merkle root per directory with per-file lines in a sidecar referenced by hash) so the record stays human-readable.

Acceptance criteria:
1. A current row scoped to a subject set survives a commit that touches only files outside the set.
2. A change inside the subject set stales the row.
3. A checkout and an exported tree with the same ignore list produce the same scope hash.
4. Record growth per regeneration is bounded by the subject set, not the tree (test with a large synthetic tree).
5. Rows with `full-root-v1` keep byte-identical behavior and every MAC-p7jd guarantee (existing attest tests green).
6. CHANGELOG states the proof-scope difference between a scoped row and a full-root row.

## Acceptance Criteria


## Design


## Notes


## History


## Links
- Related: [[MAC-38er]]

## Comments
