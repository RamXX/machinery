---
id: MAC-n87x
title: "Gw-packet: headroom warning band, fixed-overhead line, design-only reading of GV_IMPL_REQUIRED"
status: open
priority: 2
type: feature
labels: [packet, gw, gv, h2, from-next]
created_at: 2026-09-24T21:32:50Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:50Z
content_hash: "sha256:7466299ae91963026ea18273315748fb16b3d33b472c7d478797399a88d7f18f"
---

## Description
Gw-packet: headroom warning band, a fixed-overhead line, and a design-only reading of GV_IMPL_REQUIRED

Problem (H2, 2026-09-21, M4 design edit): the edit tripped `Gw-packet` on M1-S4 (598097 of 600000 bytes before the edit wrote a line) because M1-S4 cited `section:BUILD/assess.md#9` and the M4-and-M9 partition lived in a subsection of 9. Citing subsections worked once the subsection was promoted a heading level, but M1-S8 now sits at 599841 of 600000 (159 bytes of headroom), and every packet grew by a uniform 1676 bytes in that edit, including packets citing nothing the edit touched.

Separately, a design-only run on a corpus whose `gt.conformance-test-shape` is `kind: current` always reads one blocking finding (`GV_IMPL_REQUIRED`), so a design lane can never be "0 blocking".

Evidence: internal/gates/packet.go has no headroom or overhead reporting; `check` has no design-only mode (cmd/machinery/check.go:25-31). MAC-jr21 (open, awaiting H2 AC1 confirmation) delivered the budget gate itself in 0.8.0.

Proposed fix:
(a) `Gw-packet` reports per-slice headroom and warns inside a band (for example under 2 percent) before it fails.
(b) The packet's fixed overhead (bytes every packet carries regardless of cites) is reported as its own line per run.
(c) A `--design-only` reading that classifies `GV_IMPL_REQUIRED` for current rows as "owed to the impl review" (informational, non-blocking) so a design lane's green means green; the full check still blocks.

Acceptance criteria:
1. A slice within 2 percent of its budget produces a warning naming the slice, size, budget and headroom; a slice over budget still fails.
2. The packet report prints a fixed-overhead line whose value equals the size of an empty-cites packet.
3. `check --design-only` on a corpus with a current `gt.conformance-test-shape` row reports 0 blocking and lists the deferred GV_IMPL_REQUIRED item; `check` without it is unchanged.
4. Packet bytes and existing golden output are unchanged apart from the new report lines.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
