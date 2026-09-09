---
id: MAC-d3ov
title: "doctor reports a fresh 0.7.0 install as invalid: receipt-bound artifact digest differs from the recomputed digest"
status: closed
priority: 0
type: bug
labels: [install, doctor, field-defect]
created_at: 2026-09-09T00:50:33Z
created_by: ramirosalas
updated_at: 2026-09-09T16:18:44Z
content_hash: "sha256:9e1702d1155d240070684267cfa399070fafe08c3905cd1156fc500c8ab65396"
closed_at: 2026-09-09T16:18:44Z
close_reason: "Fixed on main at 69e54b0: delegated placement child records its own placement when the update parent announces no receipt ownership (MACHINERY_INTERNAL_INSTALL_RECEIPT_OWNER); TestDelegatedPlacementChildReceiptOwnership. Reproduced from the published v0.6.11 and v0.7.0 assets in a fresh HOME; field verification (0.6.11 to 0.7.1 in a fresh HOME, doctor clean after one update) follows the release."
---

## Description
## Symptom
Immediately after a successful `machinery update --version v0.7.0` (direct update committed; only the Claude plugin refresh failed, MAC-9zpf), `machinery doctor` reports:

```
invalid machinery skill under /Users/ramirosalas/.agents: artifact digest is sha256:e2262eda..., want receipt-bound sha256:cc1d8dd4...
invalid build-writer role under /Users/ramirosalas/.agents: artifact digest is sha256:6c7c6cce..., want receipt-bound sha256:7db10557...
```

The installed SKILL.md is byte-identical to the release commit's skills/machinery/SKILL.md (sha256 e19140be...), the install is a plain copy (not a dev-link symlink), and the files carry the update's timestamp. So the receipt's artifact digest and doctor's recomputed artifact digest disagree on a correct installation.

## Suspected cause
The receipt digest is computed by update from the release source archive (normalized ownership and modes, per the archive contract) while doctor recomputes from the on-disk tree with different inputs (file modes, directory entries, or the receipt-bound normalization missing). Either the two must use one artifact-digest function over the same normalized view, or doctor must state which view it hashed.

## Impact
Every fresh 0.7.0 install reports itself invalid in doctor; doctor exits non-zero, so any script gating on it fails. Field defect; 0.7.1.

## Ask
Reproduce with a fresh HOME (install from the v0.7.0 release, run doctor), unify the digest, add a contract test that install/update followed by doctor is clean for copy and symlink homes.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-09T16:06:01Z status: open -> in_progress
- 2026-09-09T16:18:44Z status: in_progress -> closed

## Links


## Comments

### 2026-09-09T00:50:52Z ramirosalas
Correction after a second run: a second `machinery update --version v0.7.0` converged both artifacts and doctor is now clean (no invalid lines). So the digest function is consistent; the defect is in the FIRST update over an existing v0.6.11 install: it left ~/.agents/skills/machinery and the build-writer role out of step with the receipt it wrote (fsm-author was fine), and only the rerun converged them. This contradicts the 0.7.0 'installer reruns converge' contract's first-run guarantee. Reproduce: fresh HOME with a v0.6.11 install (copy-mode home group plus symlinked ~/.claude), then update to v0.7.0 once, then doctor. Expected clean after one update.

### 2026-09-09T16:06:01Z ramirosalas
Reproduced in a fresh HOME from the published assets: install v0.6.11 (install.sh at v0.6.11), machinery update --version v0.7.0 --skip-plugins, doctor: same two digest pairs as the field report (e2262eda vs cc1d8dd4, 6c7c6cce vs 7db10557). The receipt after the first update is byte-identical to the 0.6.11 receipt; the tree on disk is 0.7.0. Root cause: commit 0ad71eb (MAC-2u36) moved receipt finalization from the placement child (internal/install/install.go:246 'opts.Record && !tx.delegated') to the update parent (internal/install/update.go:212 recordRefreshPlanLocked). On a cross-version update the parent is the OLD binary: 0.6.11's updateLocked has no recordRefreshPlanLocked (it relied on the child), and the 0.7.0 child, being delegated, leaves the receipt untouched. Nobody writes it. The second update runs with a 0.7.0 parent, which finalizes, so it converges. Fix: the 0.7.1 parent announces receipt ownership to its children through MACHINERY_INTERNAL_INSTALL_RECEIPT_OWNER=parent (internal/install/lock.go); a delegated child without the announcement records its own placement (recordDelegatedHomeInstallLocked / recordDelegatedTargetInstallLocked, retaining the recorded digest for an artifact a sibling child still has to place). Env rather than the capability payload because pre-0.7.1 children compare the payload byte for byte and a downgrade runs such a child. Test: TestDelegatedPlacementChildReceiptOwnership (real child under a prepared parent transaction, both modes); fails on pre-fix install.go with 'child left the receipt describing the previous release'.
