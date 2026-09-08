---
id: MAC-hlsd
title: "a killed process group reads as a survivor while its members await reaping"
status: open
priority: 1
type: bug
labels: [processscope, ci, custody, linux]
created_at: 2026-09-08T18:00:37Z
created_by: ramirosalas
updated_at: 2026-09-08T18:00:37Z
content_hash: "sha256:0837d7a1ddd41f795bbd4ccbfe64705ca8adb31076519aa5e77a54fbb48b19f1"
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
