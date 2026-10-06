---
id: MAC-awrq
title: "install: fixed 90 s watchdog panic in bootstrap finalization tests under host load"
status: closed
priority: 3
type: bug
labels: [install, flaky-under-load, preflight]
created_at: 2026-09-08T22:25:52Z
created_by: ramirosalas
updated_at: 2026-10-06T05:53:40Z
content_hash: "sha256:dd3055f14571c7cf851435317c88207a3627ed6cb4c0825f2bbdb3598c2bc86a"
related: [MAC-qo6n]
closed_at: 2026-10-06T05:53:40Z
close_reason: "Fixed in 0.11.1 (e13d2e2a RED, c44e90de GREEN): parent updates finish within installer budgets; no fixed 90 s watchdog."
---

## Description
## Symptom
Pre-push race sweep on main e704f24 (2026-09-08, host shared with a concurrent H2 design-check run):

```
panic: real parent Update exceeded 90-second operation bound
goroutine 328 [running]: github.com/RamXX/machinery/internal/install.bootstrapFinalizationCase.func6() internal/install/bootstrap_receipt_test.go:972
FAIL github.com/RamXX/machinery/internal/install 651.422s
```

## Cause
The test wraps a real installer Update in a fixed 90 second watchdog that panics the whole package. The Update itself carries its own budgets; the watchdog is a test assumption sized for an idle host. The package is fsync-bound and already the slowest in the sweep (about 1190 s under race), so any concurrent IO pushes one Update past 90 s and the panic takes down every other test in the binary, with no diagnostic about what the Update was doing.

## Ask
Replace the panic watchdog with the Update's own deadline (or a t.Fatal after the operation returns, with the operation's elapsed time and the installer's own progress state in the message), derive the bound from a measured baseline with a stated margin, and move this suite out of the parallel race sweep into the required lane per MAC-qo6n so its timing is measured on a quiet lane.

## Acceptance Criteria


## Design


## Notes
Triage 2026-09-24: valid. Watchdog unchanged at internal/install/bootstrap_receipt_test.go:972 (time.AfterFunc 90s panic). Do the t.Fatal/deadline fix first (independent, cheap); the quiet-lane move depends on MAC-qo6n.
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Replace the panic watchdog with t.Fatal after the operation returns (elapsed time and installer progress in the message) or the Update's own deadline; moving to a quiet lane is separate (MAC-qo6n). Evidence: internal/install/bootstrap_receipt_test.go:972 still has time.AfterFunc(90*time.Second, panic(...)); no commit changing it after 2026-09-08. Notes: The first half (t.Fatal/deadline) is independent; only the lane move depends on MAC-qo6n. No blockedBy set, relation is Related only, which is correct.

## History
- 2026-10-06T05:53:40Z status: open -> closed

## Links
- Related: [[MAC-qo6n]]

## Comments
