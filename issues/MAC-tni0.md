---
id: MAC-tni0
title: "custody test: the owner-loss handoff is bounded by a fixed helper poll and reports an empty file when it expires"
status: open
priority: 1
type: bug
labels: [custody, flaky-under-load]
created_at: 2026-09-08T15:45:58Z
created_by: ramirosalas
updated_at: 2026-09-08T15:45:58Z
content_hash: "sha256:2b560d4aeaa532ed1d9d9cf86a7cc8014e0a95e83fc3f45eebde68c104d420ba"
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


## Links


## Comments
