---
id: MAC-6rp0
title: "processscope: two remaining unsynchronized job-state reads in dispatch and handleRun"
status: open
priority: 2
type: bug
labels: [processscope, race]
created_at: 2026-09-08T08:52:37Z
created_by: ramirosalas
updated_at: 2026-09-08T08:52:37Z
content_hash: "sha256:2b1596865575640399150537c86940f2ed9572700844d8301e9dce337df51c36"
---

## Description
## Context
While fixing MAC-a8s2 (commit 10c4158) the broker's close dispatch was found reading `b.scopes["root"]` after releasing the broker lock, racing the sibling child-scope registration (Go fatal error: concurrent map read and map write on a 172-job H2 run). That read was fixed. Two smaller reads of the same class were left as-is to keep the release diff narrow:

- `dispatch`: the `j.retired` read at the cancel case, outside the job lock
- `handleRun`: two early-failure paths that read job state without synchronization

`go test -race` does not flag them today because the interleaving is rare, and their effect is benign in practice, but they are the same defect class that crashed the broker under load.

## Ask
Synchronize both reads (or restructure so the state is captured under the lock), add a race test that exercises cancel-during-retirement and early run failure concurrently with scope registration, and run the processscope package under -race with -count=20.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
