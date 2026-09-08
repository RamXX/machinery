---
id: MAC-ipa1
title: "processscope: a terminal result that overtakes its caller is lost, and the finished job reports TIMEOUT"
status: open
priority: 1
type: bug
labels: [custody, ci]
created_at: 2026-09-08T13:07:30Z
created_by: ramirosalas
updated_at: 2026-09-08T13:07:30Z
content_hash: "sha256:605ae9d17afc626bd40871663918a0370ac5a59e869e65859c057ddf8bd2085e"
---

## Description
## Symptom
Hosted CI run 34225004428 (ubuntu-latest, 2 vCPU, `go test -race`):

- `internal/runtimeclosure` `TestTypeScriptClosureValidateProbesUnderCustody`: `forged node identity must fail the probe: CUSTODY_ERROR: the node identity probe failed under custody: processscope: TIMEOUT: job-...: job terminated by scope cancellation`, after 37 s.
- `internal/processscope` `TestSequentialRetirementsEachGetTheirOwnBudget` on macos-latest: `round 5: run: processscope: TIMEOUT: job-...: job terminated by scope cancellation`, after 30 s.

Both jobs are trivially short (`/bin/echo ok`, a `sh -c 'echo v25.9.9'` shim). Both waited out their whole run deadline.

## Cause
`internal/processscope/scope.go`: the control-channel demux registers a job's result channel when it reads the `started` frame, and deletes that registration when it delivers the `result` frame. `Run` claims the channel with `jobChanFor(jobID)` only after the `started` reply reaches it.

One goroutine reads both frames sequentially. For a job short enough to finish before its caller is rescheduled, the demux delivers the result and deletes the registration first; `jobChanFor` then registers a second, empty channel and `Run` waits on that one until its own deadline, reports TIMEOUT, and the broker retires a job that had already exited cleanly. The result sits in the abandoned channel.

The window is a scheduling window, so it opens on a small or loaded runner and stays shut on a fast idle host.

## Ask
Make the registration belong to the caller for the whole call rather than to the delivery, so a result that arrives first is waiting in the channel the caller claims. A duplicate result (the retirement of an already-reported job) must be dropped rather than block the reader every other frame on the connection depends on.

## Acceptance Criteria
- A result delivered before its caller claims the job channel is returned to that caller.
- A duplicate result never blocks the control-channel reader.
- The existing custody suite stays green under race, including inside a 2-CPU container.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
