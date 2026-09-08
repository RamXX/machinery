---
id: MAC-idn3
title: "contributor lane: go/python runtime identity probes time out under host load with a bare TIMEOUT"
status: open
priority: 2
type: bug
labels: [flaky-under-load, integration-lane, custody]
created_at: 2026-09-08T12:15:30Z
created_by: ramirosalas
updated_at: 2026-09-08T12:15:30Z
content_hash: "sha256:54ce4743c2e079fb30a112602ce9c0929570c75fecfceb6cc3592d75aefbccdb"
---

## Description
## Symptom
Under heavy host load (load average about 39 from two parallel `go build -a ./...` loops, darwin/arm64) the contributor-lane runtime identity probes for go and python time out inside process custody:

```
TestContributorLaneExecutesGoConformanceFragment: CUSTODY_ERROR: processscope: TIMEOUT
```

At load average about 73 (four loops) four go/python probe validations failed the same way. Zero failures at normal load (three consecutive `-count=3` runs of internal/tdd/adapters, 22m41s). Observed 2026-09-08 while verifying MAC-k3rb.

## Why it matters
The identity probe is a prerequisite check (run `go version` / `python3 --version` under custody and compare with the pin). Its budget is sized for an idle host, the same defect class as the lane bootstrap clamp fixed in 24aa407. A probe that times out fails the whole lane closed with a diagnostic that says nothing about load. The hosted CI runners are small but not idle.

## Ask
Decide the probe budget from measured cost with a stated margin (and log the measured duration in the receipt), or retry the identity probe once within the lane's wall before failing, and make the diagnostic say the probe exceeded its budget rather than a bare TIMEOUT.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
