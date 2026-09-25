---
id: MAC-snrm
title: "Design file size gate at the packet budget (64 KiB) with ratchet baseline for existing oversized files"
status: open
priority: 1
type: feature
labels: [context-budget, gates]
parent: MAC-syos
created_at: 2026-09-25T19:39:57Z
created_by: ramirosalas
updated_at: 2026-09-25T19:39:57Z
content_hash: "sha256:d4ddc589ef398d5fd8c905b86c52131d9cc4d9b6f92d20665c4762a5bb276473"
blocked_by: [MAC-w44m]
---

## Description
Add a file-size gate: every design file machinery reads is at most the packet budget (64 KiB; one shared constant with Gw-packet). Existing designs are not broken on upgrade: a ratchet baseline records each oversized file's current size, which may shrink but not grow; any new file must comply. The message names the file, its size, the budget, and the command that shards it.

Acceptance criteria:
1. The budget constant is shared with Gw-packet (one number, documented as about 16k tokens).
2. A new file over budget fails; a baselined file that grows fails; a baselined file that shrinks under budget drops out of the baseline.
3. The gate covers design/ files machinery reads, including docset shards and INDEX files; implementation trees are out of scope.
4. CHANGELOG compatibility note; the baseline is created by machinery baseline, not by hand.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-25T19:39:57Z dep_added: blocked_by MAC-w44m

## Links
- Parent: [[MAC-syos]]
- Blocked by: [[MAC-w44m]]

## Comments
