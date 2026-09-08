---
id: MAC-a8s2
title: "verify-formal fails on large designs: shared absolute cleanup deadline marks cleanly reaped jobs as budget-exceeded"
status: open
priority: 0
type: bug
labels: [regression, processscope, verify-formal]
parent: MAC-ui8a
created_at: 2026-09-08T07:54:06Z
created_by: ramirosalas
updated_at: 2026-09-08T08:05:41Z
content_hash: "sha256:79311d42d196a42ad50160aa771c9e70d66ff5fae80ac88f203728de3c4256f4"
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


## Links
- Parent: [[MAC-ui8a]]

## Comments
