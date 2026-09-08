---
id: MAC-k3rb
title: "elixir adapter: ExUnit teardown noproc under load is reported as BUILD_ERROR after a completed run"
status: open
priority: 2
type: bug
labels: [flaky-under-load, adapters, elixir]
created_at: 2026-09-08T08:21:09Z
created_by: ramirosalas
updated_at: 2026-09-08T11:44:38Z
content_hash: "sha256:42385fe65d039e26626bee1a3a54c3f36c4eb6087128df6545e3d2226f440df8"
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

### 2026-09-08T11:44:38Z ramirosalas
Fixed on branch fix/MAC-k3rb at 4d789cf (worktree /Users/ramirosalas/workspace/machinery-worktrees/fix-k3rb). Left open for review/merge.

ROOT CAUSE (deterministic, not a guess)
The embedded formatter self-terminated on the last lifecycle event:
- internal/tdd/adapters/assets/elixir/machinery/reporter.exs:135 (pre-fix) returned {:stop, :normal, close(state)} from handle_cast({:suite_finished, _times_us}, state).
ExUnit calls the formatter teardown immediately after that cast, in the same frame:
- ExUnit.Runner.run_with_trap/2 (lib/ex_unit/runner.ex:69) calls EventManager.suite_finished (an async cast) then, two statements later, EventManager.stop/1.
- ExUnit.EventManager.stop/1 (lib/ex_unit/event_manager.ex:25-26) iterates DynamicSupervisor.which_children(sup) and calls GenServer.stop(pid, :normal, :infinity) on each listed child.
- Elixir GenServer.stop/3 (lib/gen_server.ex:1144) wraps a :proc_lib.stop exit as {reason, {GenServer, :stop, [pid, :normal, :infinity]}}.
So a formatter that dies inside the suite_finished cast leaves a window in which it is already dead while the supervisor has not yet processed its exit signal; which_children still lists the pid, GenServer.stop hits a dead process, and the exit is {:noproc, {GenServer, :stop, [...]}}. That kills the ExUnit.async_run task, and mix exits 1 AFTER the suite has fully executed. Exit-signal delivery from a dying process versus a call from a third process is not ordered, so host load widens the window; that is why it only showed up in the loaded release-gate sweeps and passed in isolation.

EVIDENCE
Standalone probe replicating the exact sequence (DynamicSupervisor + two temporary GenServer children + suite_finished cast + a sync stats call + EventManager.stop):
- self-stopping formatter: 173 noproc / 20000 runs (0.87%), at ordinary load.
- non-self-stopping formatter: 0 noproc / 20000 runs.
Both sweep logs show the identical stack (event_manager.ex:26 -> runner.ex:69), once on calibration-unsafe and once on calibration-safe.

FIX
1. Harness: reporter.exs now returns {:noreply, close(state)} on suite_finished. The stream is still closed there (sentinel written, device closed); the process stays alive so ExUnit's EventManager.stop/1 stops it exactly once. There is no window left. Pin ElixirReporterPinnedSHA256 updated to 6f547fde32de5dcd10947b9aedc8a215d0d8006db71f7d65a71166065fa753d3.
2. Classification (internal/tdd/adapters/elixir.go:465-475): exit=1 AFTER a complete reconciled stream is no longer BUILD_ERROR. Reaching that branch requires reconciliation to have succeeded, which proves the suite compiled and ran, so a build error is impossible there. It is now UNEXPECTED_FAILURE naming the reconciled outcome, and the witnessed events are preserved on the execution record. Exit concordance is untouched, so such a run still never certifies (docs/test-assurance-contract.md requires suite completion PLUS exit concordance for a successful record). Contract section 6 updated with both rules.

REGRESSION TESTS
- TestElixirReporterSurvivesItsOwnSuiteFinished (elixir_test.go): locks the formatter's suite_finished clause against any {:stop, ...} return.
- TestElixirAdapterSeparatesPostExecutionFaultFromBuildError (elixir_integration_test.go): deterministic post-execution fault via ExUnit.after_suite(fn _ -> exit(...) end), which ExUnit invokes from the same run_with_trap/2 frame that stops the event manager. Verified RED against the old classification and GREEN with the fix.

VERIFICATION
- go test -race -count=3 ./internal/tdd/adapters -timeout 25m (minimal env): PASS, 1356.6s (22m41s).
- Same with parallel go build -a load loops (load avg ~39): 34m58s, one failure, TestContributorLaneExecutesGoConformanceFragment, a go runtime identity probe custody TIMEOUT; zero elixir failures, zero noproc, zero mix-level aborts across all 3 counts. An earlier attempt under 4 load loops (load avg ~73) hit the go test wall clock and failed 4 go/python probe validations the same way. Both are the known load-sensitive processscope probe family, unrelated to this fix.
- go test -count=1 -run 'Assurance|Lane|Elixir' ./cmd/machinery ./scripts/integration-lane -timeout 20m: PASS (0.5s + 142.5s).
- go run ./scripts/integration-lane --lane required: PASS, 8 suites, 8 assurance suites across 4 native adapters, 4m43s.
- gofmt clean, go vet ./... clean, golangci-lint run ./... 0 issues.
