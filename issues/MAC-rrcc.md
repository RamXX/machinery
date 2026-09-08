---
id: MAC-rrcc
title: "custody root close fails with STALE_CAPABILITY broker channel closed on loaded macOS"
status: open
priority: 1
type: bug
labels: [custody, macos, ci]
created_at: 2026-09-08T20:08:53Z
created_by: ramirosalas
updated_at: 2026-09-08T20:33:27Z
content_hash: "sha256:6bcfbf5d71f4dbf772010bf93741a432200ab87e957bcdd36fe8f72634f0dca5"
---

## Description

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments

### 2026-09-08T20:33:10Z ramirosalas
Root cause: internal/processscope/scope.go:162-174, the select in (*scope).request. The broker answers the root close and then exits (broker.go:485 sends msgClosed and returns isRoot; serveChannel returns, main teardown runs retireAll/writeLedger and calls conn.Close() at broker.go:352-357). The caller's single demux goroutine (scope.go:79-134) buffers that reply into replyCh (cap 8) and then reads the end of the channel and closes eofCh. A caller that had not yet reached its select when both were ready got a uniformly random case: half the time <-s.eofCh, which returned STALE_CAPABILITY: root: broker channel closed, and Close() (scope.go:551) then synthesised the cleanup-failed report with Jobs:[] and CUSTODY_ERROR 'close did not complete'. Exactly the hosted shape in run 34269771587: root only (no other close ends the broker), no job diagnostics, and fast (1.85s and 4.02s test wall, well inside the 5s authWindow and the 45s close ctx), so it was never a cap, a timeout, a guardian exit, an owner-liveness misfire, or the descriptor collector. Fix e704f24: request drains an already-delivered reply in the eofCh branch before reporting a lost channel. Settled read, not a second race: the demux is one goroutine and delivers every reply before it closes eofCh. No shipped cap changed, docs/native-custody-contract.md untouched. Evidence: new TestRequestPrefersADeliveredReplyOverTheEndOfTheChannel (internal/processscope/scope_test.go) puts a request in exactly that state deterministically (reply written, peer CloseWrite, request issued only after eofCh closes). Pre-fix it fails on iteration 0 with the verbatim CI error string; post-fix 64/64 pass. go test -race -count=3 ./internal/processscope ./internal/processcontrol green (190s / 89s). gofmt, go vet, golangci-lint clean.

### 2026-09-08T20:33:27Z ramirosalas
Gates still in flight at close: (1) A/B stress, before.test (c306a19) vs after.test (e704f24), -test.count=25 over TestTypeScriptAdapterRejectsEarlyProcessExit, TestPythonAdapterRejectsStalePreparedOutput and TestGoAdapter*, both under six concurrent 'go build -a ./...' loops (host load average 130 on 16 cores), minimal env: 0 failures on either side so far, so the natural window has not opened on this host, which is expected because the window is one goroutine being descheduled between two statements. (2) the required lane. The deterministic in-package test is the load-bearing evidence, not the stress.
