---
id: MAC-0cuu
title: "custody provisioning subtest races the pinned Java identity probe it observes"
status: open
priority: 1
type: bug
labels: [integration-lane, ci, custody, flake]
created_at: 2026-09-08T17:44:34Z
created_by: ramirosalas
updated_at: 2026-09-08T17:44:34Z
content_hash: "sha256:8b3d44486a01f4505a4475cd0ecfc02e591916e69098f5cb900ad7afba356a40"
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
