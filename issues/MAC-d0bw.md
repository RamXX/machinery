---
id: MAC-d0bw
title: "packet: milestone:<shard> citation unresolvable when the shard's Build plan section opens with an N/A preamble"
status: closed
priority: 3
type: bug
labels: [packet, gw, h2]
created_at: 2026-09-10T19:13:53Z
created_by: ramirosalas
updated_at: 2026-10-06T05:53:39Z
content_hash: "sha256:0cd7760255fe66bd84c6955cf188ae61106418bf8df24456b591b586fdbbe985"
related: [MAC-n87x]
closed_at: 2026-10-06T05:53:39Z
close_reason: "Fixed in 0.11.1 (ff74d149 RED, ec650172 GREEN): milestone citations resolve after an N/A preamble."
---

## Description
H2 shards (design/BUILD/core.md, trust.md, assess.md) open section 9 with 'N/A - the build plan is the root BUILD.md section 9 sealed-layer plan' followed by real per-milestone list items ('- **M1** (...)') that Gb-plan links to the root. docs/packet-projection.md and the H2 handoff describe milestone:<shard> as resolving to that shard's own M1 item (about 320 lines), but planBlocksOf short-circuits on the N/A first line and the projector answers 'BUILD/core.md declares no Build plan section'. Every H2 slice therefore cites section:<shard>#9 whole, dragging M2 to M10 (about 30KB) into each packet. Expected: an N/A preamble followed by milestone list items is a plan with items, or the doc says milestone:<shard> requires a non-N/A opening.

## Acceptance Criteria


## Design


## Notes
Triage 2026-09-24: valid (planBlocksOf short-circuits on N/A, packet.go ~619) but H2 worked around it with subsections.
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Treat an N/A preamble followed by milestone list items as a plan with items, or document that milestone:<shard> requires a non-N/A opening; add test. Evidence: internal/gates/packet.go:620-622 planBlocksOf returns (nil,false) when the first non-blank line starts with N/A, so milestone:<shard> cannot resolve list items after an N/A preamble; unchanged since triage. Notes: Related MAC-n87x (packet headroom); H2 has a workaround via subsections. Small fix.

## History
- 2026-10-06T05:53:39Z status: open -> closed

## Links
- Related: [[MAC-n87x]]

## Comments
