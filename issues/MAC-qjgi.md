---
id: MAC-qjgi
title: "custody: a close steps over a retirement already claimed and reports its half-written state"
status: open
priority: 1
type: bug
labels: [custody, flaky-under-load]
created_at: 2026-09-08T16:12:18Z
created_by: ramirosalas
updated_at: 2026-09-08T16:12:18Z
content_hash: "sha256:af97c351cc8bcd86c38c30ba7aa6b1498c5e139da087095efc5f6343988e26fb"
---

## Description
go test -race -count=5 ./internal/processscope fails on an idle host:

    --- FAIL: TestRunTimeoutCleanup
        custody_integration_test.go:499: cleanup status cleanup-failed: {Status:cleanup-failed Jobs:[{ID:job-... Registered:true Terminated:true Reaped:false}] Containers:[] Diagnostics:[]}

Terminated:true with Reaped:false is not a state any retirement ends in: retireJob clears terminated when the reap fails. It is the state a retirement occupies while it is running, between marking the job terminated and reaping its guardian.

Root cause (internal/processscope/broker.go, retireAll and retireSubtree): 9acc0b8 gave retireJob a join, so that a retirement arriving at a job another path already claimed waits for it to settle instead of reporting past it. But both callers filter the job table on !j.retired before calling retireJob, so a job whose retirement was claimed before the snapshot is never passed to it and the join is never reached. A Run cancelled by its caller is retired on its own goroutine (the cancel dispatch), and the close that follows the cancelled Run arrives while that retirement is mid-flight: the close collects nothing, and buildReport publishes the half-written state as cleanup-failed with no diagnostic, on a job that goes on to terminate and reap cleanly.

Fix: both collectors pass every job in scope through retireJob, which joins a claimed retirement under that retirement's own budget and returns at once for one that has already settled. The two paths that settle a job which never reached a guardian take their claim under the broker lock, and the cancel dispatch reads the claim through the same lock, so the claim is no longer read and written unsynchronized.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
