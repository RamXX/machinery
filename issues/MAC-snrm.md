---
id: MAC-snrm
title: "Design file size gate at the packet budget (64 KiB) with ratchet baseline for existing oversized files"
status: open
priority: 2
type: feature
labels: [context-budget, gates]
parent: MAC-syos
created_at: 2026-09-25T19:39:57Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:57Z
content_hash: "sha256:91cf1dd3e63021cc2fc1e964d4dfcdca4fe2515cb773459de130fa817d7204a3"
blocked_by: [MAC-w44m]
blocks: [MAC-7y8d]
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
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Add shared budget constant, size gate with ratchet baseline created by machinery baseline, CHANGELOG note. Evidence: No design file size gate exists. internal/gates/buildplan_test.go:480 shows a 65536-byte maximum for packets only; no shared constant used for design files and no ratchet baseline. Notes: Dependency on w44m is real for AC3 (docset shards and INDEX files) but the gate for single files could ship first; keep blocker, or split AC3.

## History
- 2026-09-25T19:39:57Z dep_added: blocked_by MAC-w44m
- 2026-09-25T19:39:58Z dep_added: blocks MAC-7y8d

## Links
- Parent: [[MAC-syos]]
- Blocks: [[MAC-7y8d]]
- Blocked by: [[MAC-w44m]]

## Comments
