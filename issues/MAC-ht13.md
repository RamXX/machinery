---
id: MAC-ht13
title: "Hook: admit read-only git verbs on protected generated paths"
status: open
priority: 3
type: feature
labels: [hook, h2, from-next]
created_at: 2026-09-24T21:32:50Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:50Z
content_hash: "sha256:177c72687237a66b266d99f6455645ba204f286766ae9347f9fa9b29024d571e"
---

## Description
Hook: admit read-only git verbs on protected generated paths

Problem (H2): a lane needed `git show 78f6814c -- design/machines/Connector.oracle.md` to read a withdrawn commit's oracle and was denied: "shell commands may not reference protected output regardless of verb". Reading history is not regeneration; the lane fell back to `git cherry-pick -n`, heavier and less transparent.

Evidence: internal/hook/hook.go:960 still emits "shell commands may not reference protected output regardless of verb".

Proposed fix: the hook's shell classifier admits read-only git verbs (`show`, `diff`, `log`, `cat-file`, `ls-tree`, `blame`) on protected paths when the command is a single git invocation with no output redirection, no `--output`, and no write-capable options, and keeps refusing verbs and forms that write (checkout, restore, apply, cherry-pick onto the path, shell redirection, `sed -i`, etc.).

Acceptance criteria:
1. `git show <commit> -- <protected oracle path>` is admitted.
2. `git diff`, `git log -p`, `git cat-file -p <blob>` and `git ls-tree` on protected paths are admitted.
3. `sed -i` on the same path, `git show ... > <protected path>`, `git checkout <commit> -- <protected path>` and `git restore` on it are still refused.
4. Compound commands (pipes to a writer, `&&` with a write) keep the current refusal.
5. Existing hook deny tests stay green.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
