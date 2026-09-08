---
id: MAC-k3rb
title: "elixir adapter: ExUnit teardown noproc under load is reported as BUILD_ERROR after a completed run"
status: open
priority: 2
type: bug
labels: [flaky-under-load, adapters, elixir]
created_at: 2026-09-08T08:21:09Z
created_by: ramirosalas
updated_at: 2026-09-08T08:21:09Z
content_hash: "sha256:3fd7c4a3db10d2823c72cce3ce548d1d97b492e42e5aa2aeb2cb10a84550e8ab"
---

## Description
## Symptom
During the pre-push full race sweep on the 0.7.0 candidate (d1f2264) under host load (two other Go test suites running, load average about 5):

```
--- FAIL: TestElixirAdapterProvesNativeAssertionFailureRED (13.07s)
    elixir_integration_test.go:337: unsafe challenge must be reconciled, not errored: BUILD_ERROR: suite calibration-unsafe aborted at the mix level after execution (exit=1 stdout: Generated machinery_assurance_harness app
        07:59:14.856 [error] ** Task #PID<0.164.0> terminating
        ** Reason for termination ==
        ** {{:noproc, {GenServer, :stop, [#PID<0.172.0>, :normal, :infinity]}},
           [{GenServer, :stop, 3, [file: ~c"lib/gen_server.ex", line: 1144]},
            {ExUnit.EventManager, :"-stop/1-fun-0-", 2, [file: ~c"lib/ex_unit/event_manager.ex", line: 26]}, ...
```

In isolation the test passes: `env -i PATH HOME TMPDIR USER SHELL LANG go test -count=2 -run TestElixirAdapterProvesNativeAssertionFailureRED ./internal/tdd/adapters` is ok in 12.7 s (Elixir 1.20.4 / OTP 29).

## Why it matters
The adapter classified an ExUnit event-manager shutdown race (`noproc` on GenServer.stop during ExUnit teardown) as BUILD_ERROR "aborted at the mix level after execution". The native run had already executed the suite; the failure is in ExUnit's own teardown, which is load sensitive. The adapter should either reconcile a completed run whose teardown crashed (the assertion outcomes were already witnessed) or classify it distinctly from a build error, and the harness may need to stop ExUnit deterministically (for example `ExUnit.run/0` in a controlled process rather than relying on the default autorun teardown).

## Ask
Reproduce under load, decide the classification, and make the harness shutdown deterministic.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
