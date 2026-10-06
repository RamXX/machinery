---
id: MAC-n35t
title: "Directory ABA residual: prove mutation-witness handling on a 6.8-era Linux kernel"
status: open
priority: 3
type: bug
labels: [linux, containers, aba, dirscan]
created_at: 2026-09-24T21:32:05Z
created_by: ramirosalas
updated_at: 2026-10-06T05:53:40Z
content_hash: "sha256:4e6cc3b646070e4cf1284fb7e63e37b7eb2d6dc6ac6a0070f64c09feda65a5cb"
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
Revalidated 2026-10-06 against v0.11.0: partial. Remaining: Move WalkBounded outer window and plugin-cache walk to the mutation-channel witness (fail closed where unavailable), retry EMFILE/ENOSPC with named sysctl in the error, and get the tree-inventory ABA tests green in the Dagger ci-linux container. Evidence: Core fix shipped in 0.8.0 (a56bbe32). Remaining confirmed: dirscan.WalkBounded outer window still stamp-only (internal/dirscan/dirscan.go:285 captureDirectoryState vs :324-325, no mutation channel; the channel is armed only in Read at :111); walkPluginCacheTopology still uses installFileChangeID ctime (internal/install/install.go:642,656,668 and :1182-1209); no EMFILE/ENOSPC handling or sysctl hint anywhere (grep of internal/ cmd/ finds none outside comments); tree-inventory ABA tests (scripts/tree-inventory/main_test.go:211,385) not re-verified in ci-linux. Notes: Correctness of a fail-closed gate on pre-6.13 kernels (the CI substrate); it blocks MAC-4cbc and the hosted-equivalence claim for the Dagger lane (MAC-zafm). MAC-4cbc blockedBy is real.
0.11.1 (223ca5ed, 0d0fe63e RED; b65b91a8 GREEN): outer windows and the plugin-cache walk keep mutation witnesses; EMFILE/ENOSPC retried with a sysctl hint; 30-100 race repetitions passed on macOS and the available 7.0 kernel. Remaining only: proof on a 6.8-era kernel.

## History
- 2026-09-24T21:33:56Z dep_added: blocks MAC-4cbc

## Links
- Blocks: [[MAC-4cbc]]

## Comments
