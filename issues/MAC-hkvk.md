---
id: MAC-hkvk
title: "custody: a handed-over channel descriptor is collected in flight and its socket flushed under its new owner"
status: open
priority: 1
type: bug
labels: [custody, flaky-under-load]
created_at: 2026-09-08T15:45:46Z
created_by: ramirosalas
updated_at: 2026-09-08T16:30:43Z
content_hash: "sha256:ae6595ff8d34889e868b479268d790a6d3acfed01d0779b1f67c3141b4ed9111"
---

## Description
Under a loaded host (full go test -race ./... sweep, load average 6 to 10, reproduced here at 80 to 105 with a parallel go build -a load generator), TestSequentialRetirementsEachGetTheirOwnBudget fails within a few seconds:

    custody_integration_test.go:891: processscope: STALE_CAPABILITY: scope-981fcb9dd18867cd: broker channel closed

Root cause (internal/processscope/broker.go, childreq dispatch and handleRun capability channel): the broker creates a socket pair for a new child scope, sends one end to the caller as SCM_RIGHTS, and closes its own copy of that end immediately after the send. While the descriptor is in flight, its only reference is the copy inside the message. That is exactly what the platform's in-flight descriptor collector (XNU unp_gc) treats as unreachable: it flushes the receive side of the socket the descriptor names. The caller then receives a live, connected socket it can still send on, whose every read returns end of file.

Evidence collected with temporary instrumentation:
- The broker process is alive and healthy at the moment of failure. It logged the attach it received on the failing child channel; its serveChannel goroutine for that channel is still blocked in read; no panic, no signal, no exit.
- lsof of both processes at the failure shows the pair intact and cross-referencing: broker fd 150 device 0xc4837c4ca05d1cfd -> 0x58214d2ed201686b, client fd 124 device 0x58214d2ed201686b -> 0xc4837c4ca05d1cfd.
- The client's read returns rn=0 oobn=0 err=EOF, and a retry read with a 3s deadline returns EOF again immediately. A connected socket both of whose ends are open cannot return end of file unless its receive side was flushed.

Fix: the sender keeps its own open reference to a handed-over descriptor while it is in flight, and releases it when the owner proves it holds it, by the first record it sends on the channel. fg_count then exceeds fg_msgcount for the whole in-flight window, so the descriptor is always reachable from a descriptor table and is never a collection candidate.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments

### 2026-09-08T16:30:43Z ramirosalas
Fixed on fix/custody-load in d2a7096 (internal/processscope/broker.go:77 channel.peer and releasePeer, :374 serveChannel, :504 childreq, :609 capability channel; internal/processscope/guardian_unix.go:124 runtime.KeepAlive across the send).

Evidence before the fix, 200-retirement custody run under a parallel go build -a load generator (load average 80 to 105):
- unchanged tree: 4 failures in 20 runs (2.20s, 1.04s, 7.18s, 3.31s in), all STALE_CAPABILITY: scope-...: broker channel closed at the attach on a freshly created child scope.
- the broker process was alive and serving in every case: it logged the attach it had just read on the failing child channel; its reader for that channel was still blocked in read; no panic, no signal, no exit, broker.log clean.
- lsof of both processes 300ms after the failure showed the pair open and cross-referencing: broker fd 150 device 0xc4837c4ca05d1cfd -> 0x58214d2ed201686b, caller fd 124 device 0x58214d2ed201686b -> 0xc4837c4ca05d1cfd.
- the caller's read returned rn=0 oobn=0 err=EOF, and a retry read under a 3s deadline returned EOF again at once. A connected pair with both ends open cannot report end of file unless its receive side was flushed, which is what the platform's in-flight descriptor collector does to a socket whose descriptor is reachable only through the copy in flight.

After the fix: 0 failures in 55 runs under the same load generator.

Deterministic cover: TestChannelHoldsHandedOverDescriptorUntilOwnerSpeaks (internal/processscope/scope_test.go). Mutation-checked: removing the release in serveChannel fails it, removing the hold fails to compile.

Gates: go test -race -count=5 ./internal/processscope ./internal/processcontrol green idle and under load; go test -count=1 -run 'Formal|Custody|Scope|Verify' ./cmd/machinery green; required integration lane 8/8 suites, 8 assurance suites across 4 native adapters; gofmt, go vet, golangci-lint clean.
