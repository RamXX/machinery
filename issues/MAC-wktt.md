---
id: MAC-wktt
title: "integration lane hides the failing native test output and its evidence report"
status: open
priority: 1
type: bug
labels: [integration-lane, ci, diagnostics]
created_at: 2026-09-08T17:30:21Z
created_by: ramirosalas
updated_at: 2026-09-08T17:30:21Z
content_hash: "sha256:9db7649697c4da7a894db14b85af8da155cd9acc7271e47b56253b7a37802b8b"
---

## Description
CI run 34254790799 failed only in integration-required. The runner log carries nothing but the fail/skip accounting:

    custody-go-pilot: native execution failed: guarded job exited with status 1
    native test TestIntegrationLaneCustodyTransitiveTermination/provisioning fail failed/skip
    ...
    required execution incomplete: selected 1 started 1 passed 0

The lane never prints what the native runner said, and its evidence report (report.json plus the retained go test -json event stream) is written under os.MkdirTemp on the runner, which no workflow uploads. A remote-only failure is therefore undiagnosable.

Fix: on a failed suite execution print a bounded, clearly delimited tail of that runner stdout and stderr (last 200 lines, at most 64 KiB) naming the suite, alongside the existing accounting; give the report directory a predictable location (env-var backed, since laneValidateWorkflow pins the workflow invocation string exactly); and upload that directory from ci.yml and formal.yml on job failure with a SHA-pinned actions/upload-artifact and a short retention.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
