---
id: MAC-d3ov
title: "doctor reports a fresh 0.7.0 install as invalid: receipt-bound artifact digest differs from the recomputed digest"
status: open
priority: 0
type: bug
labels: [install, doctor, field-defect]
created_at: 2026-09-09T00:50:33Z
created_by: ramirosalas
updated_at: 2026-09-09T00:50:33Z
content_hash: "sha256:334cf31567fd3046a0ff9a4048043426a0eac28f520060ed6daa4b99506ab017"
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


## Links


## Comments
