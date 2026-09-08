---
id: MAC-ipa1
title: "processscope: a terminal result that overtakes its caller is lost, and the finished job reports TIMEOUT"
status: closed
priority: 1
type: bug
labels: [custody, ci]
created_at: 2026-09-08T13:07:30Z
created_by: ramirosalas
updated_at: 2026-09-08T13:27:03Z
content_hash: "sha256:ec31e7cab313a607ba6321567b710e3fa3f72c223342e56bf8e97d26bc31aabf"
closed_at: 2026-09-08T13:27:03Z
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
- 2026-09-08T13:27:03Z status: open -> closed

## Links


## Comments

### 2026-09-08T13:27:03Z ramirosalas
Fixed on fix/custody-budgets in 1170df1.

internal/processscope/scope.go: the job's result-channel registration now belongs to Run for the whole call (the demux no longer deletes it on delivery; Run drops it on return), so a result that arrives before the caller reaches jobChanFor is waiting in the channel it claims. A duplicate result finds the buffered channel full and is dropped rather than blocking the control-channel reader.

Reproduced in a 2-CPU, 7GB golang:1.27.1 container with --init (a subreaper is required; without one the orphan zombies mask this with a different signature): TestSequentialRetirementsEachGetTheirOwnBudget failed 3 of 8 runs at rounds 20, 70, 126 and 169, each waiting the full 30s. Broker instrumentation shows gexit terminal=false and forwardResult landing 30s before the client's cancel. After the fix: 8/8 clean, 25.1s.

Guard: TestTerminalResultSurvivesOvertakingItsCaller orders the frames deterministically with a reply-frame barrier and fails without the fix.

No budget changed. The runtimeclosure node probe failure could not be reproduced locally (the TypeScript closure skips on linux/arm64 and the container is arm64), but it is the same signature on a 30,000ms probe deadline with a one-line shell shim.
