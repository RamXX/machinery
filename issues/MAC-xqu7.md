---
id: MAC-xqu7
title: "Hook state store bricks after a block-volume reattach on Linux (device number changes, adopt cannot recover)"
status: open
priority: 1
type: bug
labels: [hook, governance, linux, kubernetes, h2-origin]
created_at: 2026-10-06T15:30:56Z
created_by: ramirosalas
updated_at: 2026-10-06T15:30:56Z
content_hash: "sha256:ce4e631d01aa4d58caf6b913073172294da388e8c4805ed6ff67cd175930222e"
---

## Description
## Summary
On Linux the governance hook binds its durable state store to the directory's `device:inode`. A block volume that is detached and reattached (a Kubernetes PersistentVolume rescheduled to another node, an EBS/OCI block volume moved between VMs, some LVM/remount cases) keeps the inode but gets a new device number. From then on EVERY managed hook event fails closed and there is no supported way to recover. macOS already tolerates this (`sameHookNativeIdentity` compares the inode only on darwin); Linux does not.

## Field reproduction (H2 dev box, 2026-10-06, machinery v0.11.1, plugin 0.10.3 and 0.11.1)
- Home directory on an OCI block-volume PVC (`/home/dev`). The pod was evicted for node disk pressure and rescheduled to another node.
- `.store-identity` recorded `directory unix:850:80e50`; after the reattach the directory witness is `unix:840:80e50` (same inode 0x80e50, device 0x850 -> 0x840).
- Every SessionStart, PreToolUse and Stop event fails: `durable hook state directory ~/.config/machinery-hook-state-<id> changed native identity; refusing to accept a replacement store`. The Stop hook blocks every turn, so Claude Code loops until its stop-hook block cap.
- `machinery doctor` prescribes `machinery hook-state adopt --root <root>`. It refuses with the same check (`internal/hook/adopt.go:83`). `adopt --from <quarantined copy>` also refuses (the copy carries the same old binding; the identity comparison runs before the generation comparison).
- Result: the operator's only options are to disable the plugin or hand-edit the integrity record. Neither is acceptable for a governance tool.

## Minimal reproducer (no cluster needed)
Unit level: `sameHookNativeIdentity("unix:850:80e50", "unix:840:80e50")` returns false on Linux. Integration level: `TestDarwinLegacyStoreBindingSurvivesVolatileFields` rewrites the bound device and passes on darwin; with its darwin gate removed it fails on Linux.

## Fix (local commit 6553172b on branch fix/hook-identity-device-reattach, NOT pushed or released)
- `sameHookNativeIdentity` applies the darwin rule on every Unix: equal inode (field 2 of the `unix:` witness) is the same directory; a different inode or a non-Unix witness is not.
- Rationale: the device number is not a stable identity for a directory that never moved. The random 32-byte store generation bound into the independent initialization marker (home-rooted, outside the config parent) still detects a replaced store, which is the threat the check exists for. Trade-off for the release review: a different store that happens to reuse the same inode number on another filesystem AND carries the matching generation would now be accepted; forging the generation already requires reading the marker.
- Tests: `TestStoreBindingSurvivesDeviceRenumbering` (former darwin-only test, now all Unix) and new `TestNativeIdentityToleratesOnlyDeviceChange`. `go test ./internal/hook` passes on darwin (175 s). Still owed: run the hook suite on Linux (amd64) before release.
- Verified in the field: a linux/amd64 build of 6553172b (sha256 d8ce3ef2a8848438...) installed to the box's `~/.local/bin` makes `machinery doctor` report the store healthy and SessionStart/PreToolUse succeed.

## Also consider
- `adopt` should be able to rebind an operator-verified store (same generation as the marker) to a new native identity, with the old and new identity written to its output, so any future identity drift has a supported recovery path instead of a dead end.
- `doctor` should not prescribe `adopt` when adopt is guaranteed to refuse.

## Is machinery usable by others with this bug?
Yes on macOS and on Linux machines whose home directory never changes block device. No for anyone whose home (or XDG config dir) lives on a reattachable block volume: Kubernetes dev pods with PVCs, cloud VMs whose data volume is moved, devcontainers with named volumes migrated between hosts. For them the first reschedule permanently blocks every agent session in every governed repo until someone edits the integrity record by hand. That is a release blocker for the dev-pod use case.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
