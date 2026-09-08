---
id: MAC-dl7y
title: "processscope: a close that skips an in-flight retirement reports the job's half-written state as cleanup-failed"
status: open
priority: 1
type: bug
labels: [custody, ci]
created_at: 2026-09-08T13:07:42Z
created_by: ramirosalas
updated_at: 2026-09-08T13:27:14Z
content_hash: "sha256:cf80fdc9501cafdea5dbc5f915620787864d9ef10dd28ba2af5734da37cd75c8"
---

## Description
## Symptom
Hosted CI run 34225004428 (macos-latest, non-race), `internal/processscope` `TestRunTimeoutCleanup`:

```
custody_integration_test.go:446: cleanup status cleanup-failed: {Status:cleanup-failed Jobs:[{ID:job-... Registered:true Terminated:true Reaped:false}] Containers:[] Diagnostics:[]}
```

`Terminated:true Reaped:false` with no diagnostic is a state no completed retirement can produce: `retireJob` clears `terminated` whenever the reap does not succeed. It is only observable while a retirement is running.

## Cause
`internal/processscope/broker.go`. A job cancelled by its caller is retired asynchronously (`go b.retireJob(j)` in the cancel handler). The guardian may report the target's death as an ordinary exit rather than a terminal drain, in which case the broker forwards the result immediately and `Run` returns while that retirement is still between marking the job terminated and reaping its guardian.

The close that follows collects only jobs with `!j.retired`, so it skips the job the in-flight retirement already claimed, and `buildReport` publishes whatever half-written state it finds. On a fast host the retirement wins the race; on a slow one the close does.

## Ask
A retirement path that finds the job already claimed must join the claiming retirement rather than skip it, bounded by that retirement's own budget so a job that truly refuses to die is still reported unreaped. No shipped cap changes.

## Acceptance Criteria
- A close over a retirement still in flight reports the settled job state.
- A job that cannot be terminated and reaped inside its own budget is still reported cleanup-failed with its TIMEOUT diagnostic.
- The existing custody suite stays green under race, including inside a 2-CPU container.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments

### 2026-09-08T13:27:14Z ramirosalas
Fixed on fix/custody-budgets in 4a4fa3f.

internal/processscope/broker.go: each job carries a retireDone channel closed when its single retirement settles. retireJob now joins a retirement that already claimed the job instead of returning past it, bounded by retireWaitCap + hardReapWindow + a 2s scheduling slack, so a job that truly refuses to die is still reported unreaped with its TIMEOUT diagnostic (TestHungJobStillReportsBudgetExceeded unchanged). No shipped cap changed and nothing runs longer; only the report waits for a settled answer, so docs/native-custody-contract.md is unchanged.

Guard: TestCloseJoinsARetirementAlreadyInFlight holds the kill inside the broker's own groupSignal seam to open the window deterministically. Without the join it reports Terminated:false Reaped:false and cleanup-failed.

Not reproduced verbatim: the exact macOS signature (Terminated:true Reaped:false, mid-reap-loop) needs a host slow enough to lose that race; the container reaps too fast. The state is unreachable from any completed retirement, which is what identifies it as this race.

Verified: processscope+processcontrol green under -race x2 in a 2-CPU, 7GB container (75.8s / 54.4s) and under -race on darwin/arm64; required integration lane green.
