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
content_hash: "sha256:c7238ac7bc1a6b17247e3c2d5d0b371b6a617e4dec67801bd2d93d6c76934f87"
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


## Links
- Parent: [[MAC-syos]]

## Comments
