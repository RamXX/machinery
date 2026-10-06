---
id: MAC-2k07
title: "tree-inventory snapshot counts nested worktrees; preflight render check fails in any checkout with worktrees"
status: open
priority: 3
type: bug
labels: [preflight, tooling]
created_at: 2026-09-08T07:58:27Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:49Z
content_hash: "sha256:818b68c13a03824d7b09b7fbd80faa8643d9b2cdcac7756273000c7e4081cf39"
related: [MAC-qo6n, MAC-38er]
---

## Description
## Symptom
`git push` on the 0.7.0 release commit was rejected by the pre-push hook at preflight step 7 (Modelith render freshness):

```
tree-inventory: snapshot files exceed 134217728-byte aggregate limit at .claude/worktrees/dev-MAC-hlae/internal/hook/hook.go
bounded repository snapshot failed
make: *** [modelith-render] Error 1
preflight FAILED: committed Modelith renders are stale or the pinned engine is unavailable
```

The checkout had 30 story worktrees under `.claude/worktrees/` (400 MB) beside a 68 MB main tree. Every worktree was clean and every branch was already on main; the release content was correct.

## Root cause
`scripts/tree-inventory` (used by `scripts/modelith-render.sh` for the bounded repository snapshot) walks every directory under the repository root, including nested git worktrees and other untracked or git-ignored trees. The aggregate byte ceiling is then a function of how many worktrees a contributor keeps, not of the repository content. Two worktrees are enough to cross the limit (68 MB main tree plus one 68 MB copy).

## Expected
The snapshot should cover the tracked tree (or at most tracked plus non-ignored files) and skip nested worktrees (`.git` file present in a subdirectory) and git-ignored paths, so the limit measures repository content. Alternatively the render script should snapshot from `git ls-files` rather than a filesystem walk.

## Also
The failure message is misleading: "committed Modelith renders are stale or the pinned engine is unavailable" hides the snapshot overflow; preflight should surface the tree-inventory error text.

## Reproduction
```
git worktree add /path/inside/repo/.claude/worktrees/x main   # repeat once more
make modelith-render-check
```

## Workaround applied 2026-09-08
Removed the landed worktrees and moved the two live ones to ~/workspace/machinery-worktrees (outside the repository).

## Acceptance Criteria


## Design


## Notes
Triage 2026-09-24: partial. b9242e8a prunes .claude (scripts/modelith-render.sh:117), 330df9c5 same for Dagger source. Residual: walk is filesystem-based, not git ls-files, so other worktrees/ignored trees still count toward 128 MiB; misleading message at scripts/preflight-fast.sh:156. Rewrite scope: snapshot from tracked files and surface the tree-inventory error. Fold into MAC-qo6n pre-push tier work.
Revalidated 2026-10-06 against v0.11.0: partial. Remaining: Snapshot from tracked (or tracked plus non-ignored) files, skip nested .git-file directories, and surface the tree-inventory error text in preflight. Evidence: scripts/modelith-render.sh:117 prunes .git, .codebase-memory and .claude (so nested story worktrees under .claude no longer count; fixed by b9242e8a/330df9c5 per triage). Walk is still filesystem-based via scripts/tree-inventory with no git ls-files; worktrees elsewhere or other ignored trees still count toward the 128 MiB cap. scripts/preflight-fast.sh:156 still reports 'stale or the pinned engine is unavailable' for a snapshot overflow. Notes: Overlaps MAC-gls5 (hashes gitignored .codebase-memory) and MAC-38er in spirit; one git-ls-files-based snapshot would fix 2k07 and gls5 together. Related MAC-qo6n is tiering, not a hard dependency.

## History


## Links
- Related: [[MAC-qo6n]], [[MAC-38er]]

## Comments
