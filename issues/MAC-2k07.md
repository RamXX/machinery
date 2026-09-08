---
id: MAC-2k07
title: "tree-inventory snapshot counts nested worktrees; preflight render check fails in any checkout with worktrees"
status: open
priority: 1
type: bug
labels: [preflight, tooling]
created_at: 2026-09-08T07:58:27Z
created_by: ramirosalas
updated_at: 2026-09-08T07:58:27Z
content_hash: "sha256:57d78bf0c59651bb8fc32d333d22282daff132f357865ff75c5fbdbe6079d94f"
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


## History


## Links


## Comments
