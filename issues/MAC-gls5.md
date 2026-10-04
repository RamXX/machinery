---
id: MAC-gls5
title: "tree-inventory snapshot hashes gitignored .codebase-memory/; pre-push modelith-render fails while the indexer writes"
status: open
priority: 3
type: bug
labels: [preflight, flake]
created_at: 2026-10-04T23:21:17Z
created_by: ramirosalas
updated_at: 2026-10-04T23:21:17Z
content_hash: "sha256:5760194cf8205022080cfa0806b96afdc3d5680bf62f74dd650fbc81aeaf5457"
---

## Description
Seen 2026-10-04 pushing the 0.10.3 release commit: pre-push 'make modelith-render' failed with 'tree-inventory: snapshot inventory changed while hashing: inventory changed between bounded passes at .codebase-memory' then 'committed Modelith renders are stale or the pinned engine is unavailable' (misleading: renders were fine). .codebase-memory/ is gitignored (.gitignore:6) but the bounded repository snapshot still walks it, and the codebase-memory-mcp watcher writes there concurrently. Retry passed. Fix: exclude gitignored paths (or at least .codebase-memory/) from the render snapshot, and make the failure message distinguish snapshot instability from stale renders. Related: MAC-2k07 (nested worktrees counted).

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
