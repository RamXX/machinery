---
id: MAC-n35t
title: "Directory ABA residuals in containers: WalkBounded outer window and plugin-cache walk still ctime-only; EMFILE/ENOSPC unretried"
status: open
priority: 1
type: bug
labels: [linux, containers, aba, dirscan]
created_at: 2026-09-24T21:32:05Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:05Z
content_hash: "sha256:459b2c9e1e4cfe8a9072841ba427a3ca08145fc362d30326ea780f31f2206879"
blocks: [MAC-4cbc]
---

## Description
Residual of MAC-33sp (the core fix shipped in 0.8.0: a56bbe32 dirscan/gates mutation channel, 1f62e9ab designlock). On pre-6.13 kernels in containers (now the CI substrate via Dagger), the ctime ABA witness is blind to sub-tick create-delete; four places still rely on it or fail opaquely:

1. dirscan.WalkBounded's outer window is still stamp-only (internal/dirscan/dirscan.go:285 vs :324).
2. walkPluginCacheTopology still uses the ctime witness (internal/install/install.go:610-637).
3. EMFILE/ENOSPC from the mutation channel are not retried, and the diagnostic names no sysctl (fs.inotify.max_user_instances / max_user_watches).
4. The scripts/tree-inventory ABA tests fail in ci-linux containers (09-09 comment on MAC-33sp; CHANGELOG 0.7.2 residual).

Acceptance criteria:
1. WalkBounded's outer window and walkPluginCacheTopology use the mutation-channel witness where available, and fail closed (not accept) where it is not.
2. EMFILE/ENOSPC are retried with bounded backoff; the terminal error names the exhausted limit and the sysctl to raise.
3. The tree-inventory ABA tests pass in the Dagger ci-linux container on a 6.8-era kernel.
4. Required before declaring the Dagger lane hosted-equivalent (MAC-zafm).

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-24T21:33:56Z dep_added: blocks MAC-4cbc

## Links
- Blocks: [[MAC-4cbc]]

## Comments
