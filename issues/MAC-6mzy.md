---
id: MAC-6mzy
title: "Current attestation: git-tracked inventory, committed exclusion list, compact root-digest manifest (keep full-root guarantees)"
status: open
priority: 1
type: feature
labels: [gv, attest, h2, from-next]
created_at: 2026-09-24T21:32:49Z
created_by: ramirosalas
updated_at: 2026-09-25T19:38:38Z
content_hash: "sha256:c21924b71c98806c68843010937f62de401d723382a8e5a4876855fb51b7706d"
related: [MAC-38er, MAC-zlbk]
---

## Description
Current attestation (Gv kind: current, policy full-root-v1) is sound but unworkable at scale; make it stable and cheap WITHOUT narrowing its scope. Owner decision 2026-09-25: keep every MAC-p7jd guarantee; attestor-chosen scope globs are rejected (they reintroduce the under-scoping hole p7jd closed). Derived per-claim scope is a separate follow-up story.

Problem (H2, 2026-09-20, v0.8.0, M2 seal): a current row carries a full-root-v1 manifest of every regular file under the implementation root (3117 entries, about 15,000 lines of attestations.yaml for a 3000-file tree).
1. Checkout and CI container never hold the same tree: the checkout has ignored files (.env.local, caches, provider state); the container drops paths the project's Dagger module ignores (.claude, .vault, which git tracks). A row generated in either is STALE in the other. H2 invented a "wall-shaped clone" and moved every check --impl and attest --impl into it.
2. The manifest is inline, so every regeneration is a 15,000-line diff and history grows by the manifest each time.

Evidence: internal/gates/attest.go:718 accepts only policy full-root-v1; attest.go:874 hard-codes it; the inventory is a filesystem walk.

Fix:
- Inventory = git-tracked files of the implementation root (git ls-files semantics, index or HEAD, decided and documented), not a filesystem walk. Untracked and ignored files never enter.
- A committed exclusion list (for paths a CI export drops, e.g. .vault, .claude), itself hashed into the row, so narrowing it stales every row that depends on it. No per-row or per-attestor selectors.
- Compact manifest: the row stores a root digest (e.g. sorted-entry Merkle root); the per-file entries live in a content-addressed sidecar referenced by hash, outside attestations.yaml.

Acceptance criteria:
1. A checkout with untracked/ignored files and a container export with the committed exclusion list produce the same root digest for the same commit.
2. Any tracked add, remove, rename, mode or content change outside the exclusion list stales the row (all MAC-p7jd negative cases still fail as before).
3. Changing the exclusion list stales every current row.
4. attestations.yaml grows by a bounded number of lines per current row regardless of tree size (test with a large synthetic tree).
5. Migration from inline full-root-v1 rows is explicit and never grandfathers freshness; CHANGELOG compatibility note.
6. H2's wall-shaped clone is no longer needed (documented).

## Acceptance Criteria


## Design


## Notes


## History


## Links
- Related: [[MAC-38er]], [[MAC-zlbk]]

## Comments
