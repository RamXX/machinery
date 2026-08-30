---
description: Run the deterministic machinery gates and explain every finding
argument-hint: "[design-dir] [--impl dir] [--commit sha] [--gate gm,gs,gu,gp,gi,gn,gc,g2,g3,gd,gl,gx,gk,gb,ge,ga,gj,gv,g4,gt,g5]"
allowed-tools: Bash(machinery:*), Read, Grep, Glob
---

Run `machinery check` for this repository and interpret the result.

1. Resolve the design directory: the first argument if given, else the
   `design` field of `.machinery.json`, else `design/`.
2. Honor `.machinery.json` (`gates` staged list, `impl` dir) unless the
   arguments override them:
   `machinery check <design> [--gate <list>] [--impl <dir>]`.
   Pass `--commit <sha>` (or export `MACHINERY_COMMIT`) when a milestone is
   being closed: Ga-accept then binds the committed acceptance evidence to
   that commit by identity. Without either, it defaults to the HEAD of the git
   repository the design sits in and binds by ancestry (the evidence commit
   must be a commit that repository holds, reachable from HEAD). Only outside
   a repository does it say so instead of checking it.
3. Report the result plainly: every ERROR and DRIFT finding, the artifact it
   points at, and the concrete fix. Read the `checked:` counts and state
   what was actually verified.
4. DRIFT always means a generated artifact is stale. The fix is to
   regenerate (`machinery oracle` | `machinery verify-formal --gen-only` |
   `machinery pack generate`), never to hand-edit the generated file.
5. Report only. Do not edit anything unless the user then asks for fixes.

Arguments: $ARGUMENTS
