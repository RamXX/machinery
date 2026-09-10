---
id: MAC-d0bw
title: "packet: milestone:<shard> citation unresolvable when the shard's Build plan section opens with an N/A preamble"
status: open
priority: 2
type: bug
labels: [packet, gw, h2]
created_at: 2026-09-10T19:13:53Z
created_by: ramirosalas
updated_at: 2026-09-10T19:13:53Z
content_hash: "sha256:840f49c6f4ea5c1410f8fd946a6b69554bf19e6f7e5f7535e611ce539902d0dc"
---

## Description
H2 shards (design/BUILD/core.md, trust.md, assess.md) open section 9 with 'N/A - the build plan is the root BUILD.md section 9 sealed-layer plan' followed by real per-milestone list items ('- **M1** (...)') that Gb-plan links to the root. docs/packet-projection.md and the H2 handoff describe milestone:<shard> as resolving to that shard's own M1 item (about 320 lines), but planBlocksOf short-circuits on the N/A first line and the projector answers 'BUILD/core.md declares no Build plan section'. Every H2 slice therefore cites section:<shard>#9 whole, dragging M2 to M10 (about 30KB) into each packet. Expected: an N/A preamble followed by milestone list items is a plan with items, or the doc says milestone:<shard> requires a non-N/A opening.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
