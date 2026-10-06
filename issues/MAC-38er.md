---
id: MAC-38er
title: "check --impl honors repository ignore rules by default (checkout 9x slower than clone)"
status: open
priority: 2
type: feature
labels: [check, performance, h2, from-next]
created_at: 2026-09-24T21:32:50Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:49Z
content_hash: "sha256:8ea70cbb625c5f4da70db4a577a221dcd133fb8cdc77cfe9bf9e10c6f657c4b8"
related: [MAC-6mzy, MAC-2k07]
---

## Description
check --impl honors repository ignore rules by default (checkout run nine times slower than a clone)

Problem (H2): `machinery check --impl .` took 3 min 38 s in the developer checkout and 33 s in a clean clone of the same commit, because the checkout holds ignored trees (editor caches, provider state, a Go module cache) the check walks and hashes.

Evidence: the only implementation-side pruning is the Architecture Contract `ignore` glob list (G4/Gt, internal/gates/oraclecov.go:215-218; CHANGELOG notes G4 prunes with the contract ignore list, not `.machineryignore`); `.gitignore` is not read under `--impl`. Related: MAC-2k07 (open) is a separate preflight inventory issue with nested worktrees. Interacts with entry 18 (Gv full-root manifest) which by MAC-p7jd design has no gitignore selector.

Proposed fix: under `--impl`, when the root is a git repository, walk the tracked set (git ls-files plus untracked-not-ignored, or tracked only; decide and document) or honor `.gitignore` by default, with `--no-ignore` to opt out. Gv full-root manifests keep their own policy unless entry 18 changes it.

Acceptance criteria:
1. A fixture repository with a large ignored directory: `check --impl` does not walk it (measured by a walk counter or trace), and results equal a clean clone's.
2. `--no-ignore` restores the current walk.
3. A non-git implementation root keeps the current behavior.
4. Measured on a representative checkout, the run is within ten percent of a clean clone of the same commit (documented measurement).
5. Gate findings on bundled examples are byte-identical.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Under --impl in a git repo walk tracked plus untracked-not-ignored files (or honor .gitignore), add --no-ignore, keep non-git behavior, measure, confirm example findings byte-identical. Evidence: No .gitignore handling under --impl: internal/gates/ignore.go covers only .machineryignore at the design root; the impl-side pruning is the Architecture Contract ignore globs (internal/gates/oraclecov.go testCorpus). grep for no-ignore/ls-files in internal and cmd finds nothing. Not in CHANGELOG through 0.11.0. Notes: Performance and DX, not correctness. Design choice interacts with MAC-6mzy (git-tracked inventory for attestation); sharing one git-tracked inventory helper with 6mzy would avoid two definitions, but this is not a hard blocker.

## History


## Links
- Related: [[MAC-6mzy]], [[MAC-2k07]]

## Comments
