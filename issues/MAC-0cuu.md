---
id: MAC-0cuu
title: "custody provisioning subtest races the pinned Java identity probe it observes"
status: open
priority: 1
type: bug
labels: [integration-lane, ci, custody, flake]
created_at: 2026-09-08T17:44:34Z
created_by: ramirosalas
updated_at: 2026-09-08T18:08:19Z
content_hash: "sha256:851920ea889677a4c8cb33b081d4b690b7105eb04acd54af2185cbe6fa2b6d96"
---

## Description
TestIntegrationLaneCustodyTransitiveTermination/provisioning (cmd/machinery/integration_lane_custody_test.go) failed the integration-required job of CI run 34254790799 on ubuntu-latest, and reproduces on a 2-CPU Linux host: 8 of 40 runs failed, 6 of them with

    integration_lane_custody_test.go:225: provisioning JVM process 0 does not belong to the lane process tree

The observation reads three separate process-table snapshots: one to decide a nested pinned JVM is live, a second to name its pid, a third to walk its ancestry. The lane provisions Java by running a pinned 'java -XshowSettings:properties -version' identity probe under its private cache (scripts/integration-lane/main.go provisionFormal), which satisfies the first snapshot and exits within a fraction of a second, so the second snapshot finds nothing (pid 0) or the third finds an ancestry that reaches nothing (the observed pid seen in run 1 of the same loop: 'provisioning JVM process 305 does not belong to the lane process tree').

Fix: answer all three questions from one snapshot. Not reproducible on macOS in ordinary runs; it is a Linux timing exposure, not a platform difference in custody.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments

### 2026-09-08T18:08:19Z ramirosalas
Fixed on branch fix/lane-diag, commit 6ccf78e (cmd/machinery/integration_lane_custody_test.go).

The observation now takes one process-table snapshot and answers all three questions from it: custodyPinnedJVM(table, cache) selects the JVM (TLC when the snapshot holds it, otherwise the pinned identity probe) over pids in sorted order, and custodyAssertGuardedAncestry walks the ancestry in that same snapshot instead of reading a fresh one. Both report-mismatch Fatalf calls now print the lane's own stdout and stderr, which is what identified the second, unrelated Linux failure (MAC-hlsd).

Reproduction and verification on a two-core Linux host (Docker golang:1.27.1, --cpus=2 --memory=3g), 40 runs each:
- before: 8 of 40 failed, 6 of them 'provisioning JVM process 0 does not belong to the lane process tree' (one variant named a live pid: 'provisioning JVM process 305 ...', the ancestry snapshot losing the probe);
- after (with MAC-hlsd): 40 of 40 passed, both subtests.
macOS: the full test passes (75s, real Java provisioning).

Root cause: scripts/integration-lane/main.go provisionFormal runs a pinned 'java -XshowSettings:properties -version' identity probe under the private cache. It satisfies the liveness predicate and exits within a fraction of a second, before the second and third snapshots were taken.
