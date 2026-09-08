---
id: MAC-aldv
title: "recovery-docker lane tests leave root-owned residue on Linux hosts (fixture ran as root)"
status: open
priority: 1
type: bug
labels: [ci, integration-lane, linux]
created_at: 2026-09-08T19:03:40Z
created_by: ramirosalas
updated_at: 2026-09-08T19:03:40Z
content_hash: "sha256:871fa54008e0e2c6b3a630c3f43b64dede02db0815eed232f88e154fff165405"
---

## Description
## Symptom
Hosted runs 34264485378 (ci) and 34264485380 (formal), integration-required job, suite recovery-docker: TestRecoveryDockerBindMountPublication and TestRecoveryDockerPartialRefusesRollback fail with permission denied on .machinery-design-publish.json and generated-tree/nested/report.txt, plus TempDir cleanup failures.

## Cause
cmd/machinery/recover_integration_test.go ran the interrupted-writer fixture container with --user 0:0. On a Linux host the files it writes into the bind mount are root-owned; the test process (uid runner) cannot inspect or remove them. macOS Docker Desktop maps bind-mount ownership to the host user, which hid it on every local run.

## Fix
The fixture runs as the host uid:gid (c306a19). Verified locally; hosted verification pending on the next run.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
