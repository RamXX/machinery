---
id: MAC-o82q
title: "Bug: ctime-witness ABA tests miss sub-tick mutations on coarse-clock Linux kernels"
status: open
priority: 1
type: bug
assignee: dev-aba
parent: MAC-ui8a
created_at: 2026-09-07T04:45:24Z
created_by: ramirosalas
updated_at: 2026-09-07T04:45:24Z
content_hash: "sha256:7fbeba7e64c11b8ce0d7ee8c0dbe172dfde09b72ed33c30ca64dc3f140722536"
---

## Description
Genuine platform finding from the first native Linux run of the formal/runtimeclosure suites (MAC-cn7q Linux verification, REPORT e7a8d2ce4427f39d73f05fd1c4547c4085f59807de20e19cbc420f53919cbede, /tmp/cn7q-linux-rerun.3028129242/). TestFormalDirectoryInventoryRejectsSameDirectoryABA and TestOpenJavaLauncherRejectsSameInodeMetadataABA fail on Linux 6.8.0-138-generic: the sub-millisecond in-test mutation lands inside the kernel's ~4-8ms coarse inode-timestamp tick, so before/after ctimes compare equal; empirically proven via ns-precision probe, deterministic in isolation. ABA rejection remains proven on Darwin (finer clock). The ctime witness mechanism needs a granularity-independent signal (e.g. statx btime/nsec where available, broker-maintained generation counter hybrid, or content-hash witness) so same-inode ABA rejection does not depend on timestamp tick luck. Likely pre-existing on main (suites were never run on native Linux before this session). Evidence retained; not a custody-logic regression.

## Acceptance Criteria


## Design


## Notes


## History


## Links
- Parent: [[MAC-ui8a]]

## Comments
