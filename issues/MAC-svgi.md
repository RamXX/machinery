---
id: MAC-svgi
title: "custody test roots clamp their declared wall to a bootstrap open context, so long runs fail BUDGET_EXHAUSTED partway through"
status: open
priority: 1
type: bug
labels: [custody, ci]
created_at: 2026-09-08T13:07:54Z
created_by: ramirosalas
updated_at: 2026-09-08T13:26:52Z
content_hash: "sha256:146b69d0a5f24f783fb8752c76751352dadba103d536514e30bac8ef5e9d727f"
---

## Description
## Symptom
Hosted CI run 34225004428 (ubuntu-latest, 2 vCPU, `go test -race`), `internal/formal` `TestVerifyFormalInScopePortfolioDesignUnderCustody`: all 7 portfolio specs fail with

```
verify supported Java runtime: processcontrol: open scoped child custody: processscope: BUDGET_EXHAUSTED: root: inherited wall deadline has passed
```

The test itself took 77.9 s.

## Cause
`processscope.Open` bounds the scope wall by the earliest of the declared `WallMS` and the open context's own deadline. `openFormalCustodyScope` (internal/formal/custody_integration_test.go) declared `WallMS: 2400000` but opened with a 60 s bootstrap context, so the real root wall was 60 s. Once a 7-spec TLC run under race on 2 vCPU passed that window, every child scope open failed BUDGET_EXHAUSTED and every spec failed with it.

This is the clamp class fixed in 24aa407 for the lane and cmd/machinery, still present in the test helpers:

- internal/formal/custody_integration_test.go (60 s context, 2,400,000 ms wall)
- internal/runtimeclosure/custody_integration_test.go (60 s, 1,200,000 ms)
- internal/processcontrol/scope_test.go (30 s, 1,200,000 ms)
- internal/tdd/adapters/{go,typescript,elixir}_integration_test.go (60 s, 1,200,000 ms)
- internal/processscope/custody_integration_test.go (120 s, default 600,000 ms)

## Ask
Derive each helper's open context from the wall that helper declares, so the two cannot drift. No cap changes.

## Acceptance Criteria
- A custody root opened by a test helper grants the wall it declares.
- A regression test fails with a bootstrap-sized open context and passes with the derived one.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments

### 2026-09-08T13:26:52Z ramirosalas
Fixed on fix/custody-budgets in e9b76b6.

Every custody root the suites open now derives its open context from the wall it declares (internal/formal, internal/runtimeclosure, internal/processcontrol, internal/tdd/adapters go/typescript/elixir, internal/processscope). No cap changed.

Measured: the 7-spec portfolio TLC run takes 85.19s under -race in a 2-CPU, 7GB golang:1.27.1 container, against the 60s bootstrap window it used to be clamped to and the 2,400,000ms wall it declares (28x margin). Hosted CI reached 77.9s before failing.

Guard: TestFormalCustodyRootGrantsItsDeclaredWall reads the granted budget back off an attachment; with the old 60s open context it reports 59,971ms and fails.

Verified: internal/formal, internal/processscope, internal/processcontrol, internal/runtimeclosure green under -race on darwin/arm64; processscope+processcontrol green under -race x2 in the 2-CPU container; required integration lane green (5m49s).
