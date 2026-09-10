---
id: MAC-8j82
title: "oracle: per-file publications share one directory-wide sentinel; concurrent invocations delete every oracle"
status: open
priority: 1
type: bug
labels: [oracle, publication, concurrency, h2]
created_at: 2026-09-10T07:50:17Z
created_by: ramirosalas
updated_at: 2026-09-10T07:50:17Z
content_hash: "sha256:e8a4efc1c947614fb774d1b77cdb5f60200fd09115c422306dd8604ee30eb7b1"
---

## Description
Observed 2026-09-10 on H2 (design with 90 machines, machinery v0.7.2). Four agents ran `machinery oracle design/machines/<Name>.machine.json` on DIFFERENT files concurrently. The publication is directory-wide: the sentinel design/.machinery-design-publish.json lists every machines/*.oracle.md as an output even for a single-file invocation, and the interrupted run left ALL 90 oracle files absent from the working tree (git status: 90 ' D' entries), with recover reporting 'input inventory does not match the recorded publication inputs ... action: rerun-writer'. Every later check, verify-c4 and lint invocation refused with 'interrupted Machinery publication "oracle" prevents a consistent design snapshot'. Expected: a single-file oracle publication scopes its outputs to that file (or takes a per-file lock), so concurrent regeneration of different machines is safe, or the command refuses to start while another publication holds the directory instead of both proceeding and destroying the outputs. Recovery on H2 was a full `machinery oracle design/machines` rerun after all writers stopped. Also worth documenting in the skill: never run oracle regeneration concurrently within one design.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
