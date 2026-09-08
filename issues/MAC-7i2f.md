---
id: MAC-7i2f
title: "custody test: replacing a package hook races the in-process broker that read it"
status: open
priority: 1
type: bug
labels: [custody, flaky-under-load]
created_at: 2026-09-08T16:04:00Z
created_by: ramirosalas
updated_at: 2026-09-08T16:31:03Z
content_hash: "sha256:9d97d41517161fb073c876161f2b225cc805049babb192ad3dd372cead2e6729"
---

## Description
go test -race -count=5 ./internal/processscope fails on the unchanged tree (afb5611), idle, in half a second:

    WARNING: DATA RACE
    Write at 0x... by TestChallengeAuthorizeUnsafeAcceptsForgedScope scope_test.go:432
    Previous read at 0x... by runBroker broker.go:303 (goroutine from startTestBroker, scope_test.go:360)
    --- FAIL: TestChallengeAuthorizeUnsafeAcceptsForgedScope: race detected during execution of test

The challenge tests run the broker in process and replace a package-level hook (hookAuthorizeRequest, hookRegisterJob) to build a second broker from the replacement. runBroker reads all three hooks once, when it builds the broker. Everything the test observes of the first broker afterwards travels through a kernel socket, which gives the race detector no ordering at all, so the assignment for the second broker races the construction of the first.

It does not show at -count=1, which is why a single sweep never reported it; any -count>1 run of the package reports it at once.

Fix (test only): startTestBroker joins the broker goroutine it started as part of its cleanup, and each test retires and joins the broker built from the previous hook before replacing it.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments

### 2026-09-08T16:31:03Z ramirosalas
Fixed on fix/custody-load in 36f126d (internal/processscope/scope_test.go: startTestBroker joins the broker goroutine in its now-idempotent cleanup; TestChallengeAuthorizeUnsafeAcceptsForgedScope and TestChallengeRegistrationSkipObserved retire and join the broker built from the previous hook before replacing it).

Before: go test -race -count=5 -run TestChallenge ./internal/processscope failed on an idle host in 0.5s with 'race detected during execution of test' (write at scope_test.go:432 against the read in runBroker at broker.go:303). Reproduced on the unchanged tree at afb5611.

After: go test -race -count=5 -run TestChallenge ./internal/processscope green, and the full go test -race -count=5 ./internal/processscope ./internal/processcontrol green idle and under load.

Test-only, no product behaviour changed.
