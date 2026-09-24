---
id: MAC-aldv
title: "recovery-docker lane tests leave root-owned residue on Linux hosts (fixture ran as root)"
status: closed
priority: 1
type: bug
labels: [ci, integration-lane, linux]
created_at: 2026-09-08T19:03:40Z
created_by: ramirosalas
updated_at: 2026-09-24T21:31:54Z
content_hash: "sha256:34728ce689b7dd21aa41f72b70836a6719ec3d5d7b93a05623590e9a13b487ff"
closed_at: 2026-09-24T21:31:54Z
close_reason: "Fixed by c306a192 (fixture runs --user uid:gid); integration-required lane green at 4bb85b78. Triage 2026-09-24."
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
- 2026-09-24T21:31:54Z status: open -> closed

## Links


## Comments
