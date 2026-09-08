---
id: MAC-a8s2
title: "verify-formal fails on large designs: shared absolute cleanup deadline marks cleanly reaped jobs as budget-exceeded"
status: closed
priority: 0
type: bug
labels: [regression, processscope, verify-formal]
parent: MAC-ui8a
created_at: 2026-09-08T07:54:06Z
created_by: ramirosalas
updated_at: 2026-09-08T09:27:48Z
content_hash: "sha256:7eac6f1ea097c6982f6d40344213134bebcf88b52f6578e5e931dbb3320d6906"
closed_at: 2026-09-08T09:27:48Z
close_reason: "Landed on main (release 0.7.0 line) at 5c35463: 10c4158 custody cleanup grace per retirement; H2 verify-formal 167/167 on the combined 0.7.0 binary"
---

## Description
## Symptom

`machinery verify-formal` fails on any design large enough that retiring its process-custody jobs
outlives a single shared cleanup instant. On a real consumer design (H2 Hextropian Platform,
86 formal specs) v0.7.0 fails 65 of 86 specs and exits 1, where v0.6.11 passes all 167 checks
and exits 0 on the identical tree.

## Reproduction

Host: darwin 25.6.0 (arm64), Java 21 (Temurin), no other load beyond the run itself.
Design: /Users/ramirosalas/workspace/H2 at commit c59bc81 (86 machines, 83 .machine.json plus
3 relational layers), copied to a detached worktree.

```
git worktree add --detach /private/tmp/h2-assess HEAD
cd /private/tmp/h2-assess
machinery verify-formal design   # v0.7.0
```

## Expected

`167 passed, 0 failed`, exit 0 (this is exactly what v0.6.11 produces on the same tree,
in 2m25s, leaving no drift).

## Actual

Exit 1, in 1m16s, reproduced identically on two consecutive runs:

```
  PASS  AgentMandate
  ... 21 consecutive PASS ...
  FAIL  CorrelationMapping
        error: processcontrol: scoped child cleanup status cleanup-failed: {Status:cleanup-failed Jobs:[{ID:job-aec48c3bfb0dda47 Registered:true Terminated:true Reaped:true}] Containers:[] Diagnostics:[{Code:TIMEOUT Subject:job-aec48c3bfb0dda47 Message:shared cleanup budget exhausted before retirement completed}]}
  FAIL  CoverageGap
        error: verify supported Java runtime: processcontrol: scoped child cleanup status cleanup-failed: {... same TIMEOUT ...}
  ... 63 more FAIL ...

21 passed, 65 failed
verify-formal: native custody cleanup did not verify: err=processscope: STALE_CAPABILITY: root: broker channel closed report={Status:cleanup-failed Jobs:[] Containers:[] Diagnostics:[{Code:CUSTODY_ERROR Subject:root Message:close did not complete}]}
```

Note the failing jobs report `Registered:true Terminated:true Reaped:true`: the child was cleanly
terminated and reaped. Only the budget accounting says otherwise. After the first exceedance the
broker channel closes and every subsequent Java-runtime verification fails with
`STALE_CAPABILITY`, so one accounting error poisons the rest of the run.

## Root cause (code reading)

`internal/processscope/broker.go`

- `retireJob` (line 740): when `b.cleanupArmed` is set, `waitUntil = b.cleanupPending`, a single
  ABSOLUTE instant, additionally hard-capped to `now + 3s` at lines 766-767.
- `retireAll` (line 814) then calls `retireJob` for every job SEQUENTIALLY, each against that same
  absolute instant.
- Line 803: `if time.Now().After(waitUntil) { j.budgetExceed = true }`.

So the budget is a wall-clock deadline shared by an unbounded number of sequentially retired jobs.
Once cumulative retirement time passes the instant, `time.Until(waitUntil)` is negative for every
remaining job, the select falls through immediately, and each remaining job is marked
`budgetExceed` even though it terminated and reaped correctly.

`buildReport` (line 840) then turns any `budgetExceed` into `StatusCleanupFailed`, which
`verify-formal` surfaces as a per-spec FAIL.

The threshold is therefore a function of design size. Every bundled example is far below it:
`examples/fulfillment` 6 machines, `go-crm` 5, `surreal-crm` 5, `portfolio-engine` 4, `pii-flow` 1.
No example or CI fixture in this repo reaches ~22 custody jobs in one verify-formal run, which is
why the regression shipped.

## Expected fix shape

The cleanup budget must be per-retirement, or the shared budget must be replenished/derived per
job the way the 0.7.0 wall-budget fix derived the open context from the declared wall. A job that
was registered, terminated and reaped is a clean cleanup and must not be reported as
cleanup-failed. A budget exceedance on one job must also not close the broker channel for the
whole run.

## H2 impact

BLOCKER. `.github/workflows/design-check.yml` job `design-formal` runs `machinery verify-formal design`
as a required gate. H2 cannot upgrade off the v0.6.11 pin while this stands: the formal wall is
the slow half of its CI and it goes from green to 65 failures. H2 is about to start milestone 1
and the formal suite only grows from here (86 specs today, 101 entities modeled).

## Not verified

Only reproduced on darwin/arm64. Not tested on linux/amd64 (the CI target). The threshold ~21 is
this host's; it is expected to vary with host speed, which makes the failure load-dependent
rather than deterministic across machines.

## Acceptance Criteria


## Design


## Notes
Verified at release commit d1f2264 in the main checkout: internal/processscope/broker.go lines 740 (retireJob), 753-767 (waitUntil derivation and the 3s cap), 803 (budgetExceed), 814 (retireAll), 840 (buildReport) are unchanged. Reproduced again with the relocated 0.7.0 binary: machinery check design still reports 163 blocking findings on the same tree.

## History
- 2026-09-08T09:27:48Z status: open -> closed

## Links
- Parent: [[MAC-ui8a]]

## Comments

### 2026-09-08T08:51:13Z ramirosalas
Fixed on branch fix/MAC-a8s2, commit 10c4158 (not pushed, issue left open).

The reported cause was real but was one of three single-shot assumptions in the broker, all of which only break once a run is large enough. Fixing only the first left verify-formal at exit 1.

1. Shared cleanup instant (as reported). retireJob derived its wait from b.cleanupPending, one absolute instant armed at the first scope close and never re-armed. Every later retirement raced it, so cleanly terminated and reaped jobs were marked budgetExceed.
2. Broker crash. The close dispatch read b.scopes["root"] after releasing the broker lock, racing the sibling child-scope registration that follows it. Go fatal error: concurrent map read and map write. The broker died and every later request failed STALE_CAPABILITY. This is what actually closed the broker channel, not the budget accounting.
3. Truncated control record. writeFrame sent a record with a single WriteMsgUnix and treated a short write as a failure. The root cleanup report names every retired job (172 on H2, about 14 KB), which exceeds the AF_UNIX send buffer, so the close reply was truncated and the final root close failed with STALE_CAPABILITY even with zero budget exceedances. Confirmed by instrumentation: senderr=INTERNAL_ERROR: frame: short control write on a 200-job report.

Chosen budget semantics: the cleanup grace bounds a cleanup pass, not the broker lifetime. beginCleanupPassLocked/endCleanupPass arm one grace per pass; a pass entered while another is in flight shares the grace already armed, so it never renews underneath the jobs it bounds. Each retirement gets the shipped per-job budget (cleanup_ms, capped at the shipped 30,000 ms cap and at 3 s) measured from its own start, clamped by its pass grace. All shipped numbers unchanged. Bounded because a broker admits at most limits.Jobs live jobs (shipped cap 4): a pass waits at most its grace in aggregate and reaps at most limits.Jobs x 2 s hard reap window, independent of how many jobs the run retired earlier. A job that really overruns is still reported budget-exceeded.

Tests (internal/processscope/custody_integration_test.go):
- TestSequentialRetirementsEachGetTheirOwnBudget: 200 sequential child-scope retirements plus a root close. Fails on the unmodified tree with the exact reported shape (Registered:true Terminated:true Reaped:true plus a TIMEOUT diagnostic at round ~4), and catches all three defects.
- TestHungJobStillReportsBudgetExceeded: new ignore-term helper role proves a target that refuses SIGTERM is still cleanup-failed with a TIMEOUT diagnostic. The shipped TestCleanupBudgetExhaustionReported passes unchanged.

Evidence, darwin/arm64:
- go test -race -count=1 ./internal/processscope ./internal/processcontrol: ok, 40s
- go test -count=1 -run 'Formal|Custody|Scope|Verify' ./cmd/machinery: ok, 56s
- make build, gofmt, go vet ./..., golangci-lint run --config .golangci.yml: clean
- go run ./scripts/integration-lane --lane required: 8 suites passed, 8 assurance suites across 4 native adapters, 5m07s
- H2 at c59bc81 in a detached worktree: machinery verify-formal design -> 167 passed, 0 failed, exit 0, no custody error, 2m19s. Only tree change is the machinery-version stamp moving v0.6.11 -> v0.7.0 (expected for the release bump).

Not verified: linux/amd64. Defects 2 and 3 are load- and size-dependent respectively, so the linux threshold will differ.
