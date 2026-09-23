---
id: MAC-j39j
title: "Gy: event projection treats fan-out rows of one event as duplicate stable ids, so Gy evaluates nothing on H2"
status: open
priority: 0
type: bug
labels: [consistency-layer, projection, gy, h2, blocks-h2]
created_at: 2026-09-23T23:56:54Z
created_by: ramirosalas
updated_at: 2026-09-23T23:56:54Z
content_hash: "sha256:9461021f07418a4e4a9e1e27c7b6a2a56444ba840c40dfe8f975eec4c5ffc380"
---

## Description
v0.10.0 on H2 design (branch design/machinery-0.10.0 at 0f2736ed): Gy fails projection with 'duplicate stable id "event:record.applied": defined at ARCHITECTURE.md:1475 and at ARCHITECTURE.md:1476', 22 event ids. Root cause: internal/checker/relations.go marks relation event(id, producer) as Defines:true, and internal/gates/projection_readers.go factBuilder.events adds one event row per table row, but the event contract table is one row per (event, producer, consumer) edge by design (per-consumer READS). H2 has 22 fan-out events over 77 rows; 7 of them vary the producer per row (audit.append has 7 producers) and 12 vary the payload cell per row. Fix: event(id) defined once per event name (first row is the source; later rows are edges, not duplicates); project edges as event_edge(edge, event, producer, consumer) with a content-derived edge id (not a line number) and per-edge payload fields; keep event_producer/event_consumer/event_participant as non-defining sets; rebind payload.dl so a unit's payload {} compares with the edges its component participates in, not the union. Also: two unnamed Modelith relationships between the same entity pair and cardinality collide on rel:From->To:card (H2 NormLink->Norm); report that as a clear model finding instead of failing the whole projection. Test with a synthetic fan-out fixture (multi-producer, multi-consumer, per-edge payloads) and against the H2 tree. Why missed: no bundled example has a multi-row event.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
