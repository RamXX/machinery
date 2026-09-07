---
id: MAC-o82q
title: "Bug: ctime-witness ABA tests miss sub-tick mutations on coarse-clock Linux kernels"
status: in_progress
priority: 1
type: bug
assignee: dev-MAC-o82q
parent: MAC-ui8a
created_at: 2026-09-07T04:45:24Z
created_by: ramirosalas
updated_at: 2026-09-07T21:57:40Z
content_hash: "sha256:bd25d22f94d3cf7e8e79af0e76b54be9f10812e9a6b75712529759ee55675610"
follows: [MAC-8yai]
---

## Description
Genuine platform finding from the first native Linux run of the formal/runtimeclosure suites (MAC-cn7q Linux verification, REPORT e7a8d2ce4427f39d73f05fd1c4547c4085f59807de20e19cbc420f53919cbede, /tmp/cn7q-linux-rerun.3028129242/). TestFormalDirectoryInventoryRejectsSameDirectoryABA and TestOpenJavaLauncherRejectsSameInodeMetadataABA fail on Linux 6.8.0-138-generic: the sub-millisecond in-test mutation lands inside the kernel's ~4-8ms coarse inode-timestamp tick, so before/after ctimes compare equal; empirically proven via ns-precision probe, deterministic in isolation. ABA rejection remains proven on Darwin (finer clock). The ctime witness mechanism needs a granularity-independent signal (e.g. statx btime/nsec where available, broker-maintained generation counter hybrid, or content-hash witness) so same-inode ABA rejection does not depend on timestamp tick luck. Likely pre-existing on main (suites were never run on native Linux before this session). Evidence retained; not a custody-logic regression.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-07T21:01:13Z status: open -> in_progress
- 2026-09-07T21:01:14Z auto-follows: linked to predecessor MAC-8yai

## Links
- Parent: [[MAC-ui8a]]
- Follows: [[MAC-8yai]]

## Comments

### 2026-09-07T21:57:40Z ramirosalas
ACCEPTED 2026-09-07 — RED efb8142 (deterministic coarse-tick blindness proof via test-anchored 1-hour-bucket coarsener seam) -> GREEN ce0fe0b. Hybrid witness: ALL existing stat conjuncts kept + kernel mutation-event sentinel (inotify Linux / kqueue darwin-bsd; options 1-3 evaluated inapplicable at these sites — documented). Linux confirmation on the original failing host: TestFormalDirectoryInventoryRejectsSameDirectoryABA PASS 3/3 isolated + in-suite; TestOpenJavaLauncherRejectsSameInodeMetadataABA PASS 3/3 + in-suite; 4 coarse-clock tests pass; build/vet clean; JDK provisioned pin-verified. Frozen ABA tests NOT amended (zero existing test lines touched). +596/-2, 11 files. FOLLOW-UP FILED: journal recovery path still ctime-only (new bug, release-relevant flake). REMOTE-HOST-ONLY GAP (not CI): runtimeclosure adapters suites need Elixir/Node/Python on PATH — hosted CI provisions them per bz1y wiring. Coordinator merged. Record: .git/machinery-evidence-20260906.TEFZ7D/o82q-linux-report.md
