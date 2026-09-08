---
id: MAC-wktt
title: "integration lane hides the failing native test output and its evidence report"
status: open
priority: 1
type: bug
labels: [integration-lane, ci, diagnostics]
created_at: 2026-09-08T17:30:21Z
created_by: ramirosalas
updated_at: 2026-09-08T17:41:57Z
content_hash: "sha256:533638318a22b650e5305c3815c56705ec5aad514163046b9dd4123f9e515e26"
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

### 2026-09-08T17:41:57Z ramirosalas
Fixed on branch fix/lane-diag, commit d251cdf.

Lane (scripts/integration-lane/main.go): a failed suite execution now returns a nativeStreamFailure carrying the runner's bounded stdout and stderr with the error; the suite loop replays their tail after the accounting line. laneStreamTail keeps the last 200 lines and at most 64 KiB of them and states what it dropped; laneStreamReplay delimits each block with the suite and stream name. An empty stream is stated, not omitted. A new --report-dir, defaulted from MACHINERY_INTEGRATION_REPORT_DIR (env-driven because laneValidateWorkflow and laneValidatePreflight pin the invocation to an exact argument string), names the directory holding report.json and the retained event streams; unset keeps the os.MkdirTemp default.

Workflows: ci.yml and formal.yml set that variable to runner.temp/integration-required-evidence on the lane step (step env, so the pinned run string is untouched) and upload the directory with actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a (v7.0.1, the pin release.yml already uses), if: failure(), retention-days: 7.

Tests added: TestLaneReplaysBoundedNativeStreamsOfAFailedSuite (end-to-end, asserts the delimited block and the failing test's own message), TestLaneReplaysNoNativeStreamsWhenTheSuitePasses, TestLaneStreamTailIsBoundedAndStatesWhatItDropped (line bound, byte bound, single oversized line, empty), TestLaneStreamReplayNamesTheSuiteAndItsBounds, TestLaneReportDirectoryIsNamedByEnvironmentAndDefaultsToATemporaryOne.

Verification: go test ./scripts/integration-lane 158s ok (laneValidatePreflight and the wiring guards included); go test ./scripts/release-policy ok; the nine ci.yml-pinning tests in cmd/machinery ok; gofmt clean, golangci-lint 0 issues, actionlint clean.
