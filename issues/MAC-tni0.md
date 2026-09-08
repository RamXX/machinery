---
id: MAC-tni0
title: "custody test: the owner-loss handoff is bounded by a fixed helper poll and reports an empty file when it expires"
status: closed
priority: 1
type: bug
labels: [custody, flaky-under-load]
created_at: 2026-09-08T15:45:58Z
created_by: ramirosalas
updated_at: 2026-09-08T16:30:55Z
content_hash: "sha256:6c418f1c330dc3a5572ea219c9aa3c1a91c8802adbd5ec1182295365c17dbd45"
closed_at: 2026-09-08T16:30:55Z
---

## Description
Under the full go test -race ./... sweep on a loaded host, TestOwnerLossBrokerSelfCleanup fails after 20s:

    custody_integration_test.go:584: file .../handoff.pid never appeared

This is a test-only assumption, not a product budget. The owner helper (TestHelperTarget, role owner) polls for its grandchild's pid file for a fixed 10s and then writes whatever it read to the handoff file. When the chain has not come up inside that window it reads nothing and writes an EMPTY file, which the reader's waitFile ignores (it requires len > 0), so the reader waits out its own separate 20s and reports a file that never appeared. Neither number is derived from the wall the helper declares (60s), and neither the helper's own Run error nor its log reaches the failure message.

The chain the helper must bring up before it can report is three nested launches of the race-instrumented test binary plus a SHA-256 of that binary for each of them (owner digest, Open digest, broker self-digest, guardian runtime digest). On a loaded host that is seconds, and the fixed 10s is the first thing to run out.

Fix (test only, no product budget touched): the helper polls under the wall it already declares, keeping back a fixed report slack, and always writes a record naming what happened (the pid, run:CODE, or childpid:absent). The reader derives its wait from that same declared wall and prints the helper's own log when the record is missing or is not a pid.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-08T16:30:55Z status: open -> closed

## Links


## Comments

### 2026-09-08T16:30:55Z ramirosalas
Fixed on fix/custody-load in f33e9aa (internal/processscope/custody_integration_test.go: ownerHelperWall/ownerHelperReportSlack/ownerHandoffMissing constants, the owner helper role, waitOwnerHandoff, waitPidFile).

Test-only. The helper now polls for its grandchild under the 60s wall it already declares, keeps back a 10s report slack, and always writes a record: the pid, run:CODE when the run that owns the grandchild failed, or childpid:absent. The reader derives its wait from the same declared wall and prints the helper's own log when the record is missing or is not a pid; waitPidFile now names the content it could not parse. No product budget or shipped cap changed; docs/native-custody-contract.md untouched.

Not reproduced in isolation: 40 runs of TestOwnerLossBrokerSelfCleanup under a parallel load generator at load average 78 to 105 all passed, so the fixed 10s helper poll was not observed to expire on this host outside the full sweep. What is certain from the code is the failure mode reported: the helper writes an empty file when its poll expires, waitFile ignores an empty file, and the reader then reports 'never appeared' after its own separate 20s, naming neither the helper's state nor its log. That silence is what is fixed.

Gates: go test -race -count=5 ./internal/processscope ./internal/processcontrol green idle and under load (5 runs of this test in each); required integration lane green; gofmt, go vet, golangci-lint clean.
