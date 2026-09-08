---
id: MAC-hlsd
title: "a killed process group reads as a survivor while its members await reaping"
status: open
priority: 1
type: bug
labels: [processscope, ci, custody, linux]
created_at: 2026-09-08T18:00:37Z
created_by: ramirosalas
updated_at: 2026-09-08T18:08:34Z
content_hash: "sha256:3640b5cb1dd709c930d797039f6505fddbb97ae5c6b86989daeb0800b605d91f"
---

## Description
A cancelled required-lane run publishes custody status failed with cleanup-failed and no diagnostic naming a survivor:

    provision: formal provision failed: guarded job canceled: context canceled
    custody cleanup did not verify: guarded root cleanup: report={Status:cleanup-failed
      Jobs:[{ID:job-... Registered:true Terminated:false Reaped:true}] Containers:[] Diagnostics:[]}

retireJob (internal/processscope/broker.go) signals the job group, reaps the guardian, and then probes the group once with kill(-pgid, 0). The members of a job are children of that guardian, so at the instant it is reaped they are orphans the platform's init has yet to reap, and on Linux a killed-but-unreaped member still answers the group probe (macOS skips zombies in killpg, which is why the condition is invisible there). The single probe reads that as a survivor and marks a job that terminated exactly as asked unterminated.

Observed on a 2-CPU Linux host in TestIntegrationLaneCustodyTransitiveTermination: 3 of 40 runs, alongside the separate observation race MAC-0cuu.

Fix: poll the group until it drains, bounded by the reap deadline that already bounds this stage, so nothing waits longer than before and a group that truly refuses to die is still reported unterminated.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments

### 2026-09-08T18:08:34Z ramirosalas
Fixed on branch fix/lane-diag, commit 6ccf78e (internal/processscope/broker.go).

retireJob's single post-reap probe 'else if groupExists(j.pgid)' is now 'else if !groupDrained(j.pgid, reapDeadline)'. groupDrained polls the group until it is empty, bounded by the same reapDeadline the reap loop already declared (hardReapWindow), so no budget grew: a group that truly refuses to die is still reported unterminated after exactly the wait it was given before, and an already-empty group costs one syscall as before.

Evidence for the diagnosis: on Linux, TestRetirementWaitsForAKilledGroupToDrain/draining-group-terminated takes 0.31s (the killed member is held unreaped by the test for 300ms and the group probe keeps answering for the whole window), while the same subtest takes 0.02s on macOS, where killpg skips members awaiting reaping. That is the platform difference behind a failure the hosted Linux runner shows and macOS never does.

Test added in internal/processscope/scope_test.go: TestRetirementWaitsForAKilledGroupToDrain builds the broker and job around a real group whose second member only the test can reap (the deterministic form of an orphan awaiting init), and asserts the retirement reports terminated with a cleaned scope report; a second subtest asserts a group holding a live process is never reported drained and that the drain spends the window it was given.

Verification: go test ./internal/processscope ok; go test -race -count=2 ./internal/processscope ok (80s); on the Linux host, 40 of 40 runs of TestIntegrationLaneCustodyTransitiveTermination passed with this and MAC-0cuu applied, against 8 of 40 failing before (3 of those 8 were this bug: cleanup-failed with Terminated:false Reaped:true and no diagnostic). gofmt clean, golangci-lint 0 issues.
