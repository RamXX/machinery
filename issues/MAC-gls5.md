---
id: MAC-gls5
title: "tree-inventory snapshot hashes gitignored .codebase-memory/; pre-push modelith-render fails while the indexer writes"
status: closed
priority: 3
type: bug
labels: [preflight, flake]
created_at: 2026-10-04T23:21:17Z
created_by: ramirosalas
updated_at: 2026-10-06T05:53:39Z
content_hash: "sha256:41b701bb8a7de23ada6be449ee5e393b4de44fa53d05a12ab3092af01dfe71e0"
closed_at: 2026-10-06T05:53:39Z
close_reason: "Fixed in 0.11.1 (357c9e8b RED, 1831d74e GREEN, plus 2d603fd4 fixture): pruned entries are omitted from stability records; diagnostics no longer claim a stale render."
---

## Description
Seen 2026-10-04 pushing the 0.10.3 release commit: pre-push 'make modelith-render' failed with 'tree-inventory: snapshot inventory changed while hashing: inventory changed between bounded passes at .codebase-memory' then 'committed Modelith renders are stale or the pinned engine is unavailable' (misleading: renders were fine). .codebase-memory/ is gitignored (.gitignore:6) but the bounded repository snapshot still walks it, and the codebase-memory-mcp watcher writes there concurrently. Retry passed. Fix: exclude gitignored paths (or at least .codebase-memory/) from the render snapshot, and make the failure message distinguish snapshot instability from stale renders. Related: MAC-2k07 (nested worktrees counted).

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: In tree-inventory, drop pruned entries from the stability comparison (do not stat-compare the pruned directory itself), and make preflight-fast distinguish snapshot instability from stale renders. Evidence: scripts/modelith-render.sh:117 already passes -prune .codebase-memory (commit 637bab04, 2026-09-04), but scripts/tree-inventory/main.go:323-324 still visits the pruned dir as a record (visit(..., emit=false)) and compareInventoryResults (main.go:554) compares its stat before/after, so a watcher writing in .codebase-memory still trips 'snapshot inventory changed while hashing'. scripts/preflight-fast.sh:156 still prints the misleading 'renders are stale or the pinned engine is unavailable'. No commit since 2026-10-04 touches either. Notes: Same area as MAC-2k07 (nested worktrees); could be fixed together but no hard dependency. Retry passes, so low impact.

## History
- 2026-10-06T05:53:39Z status: open -> closed

## Links


## Comments
