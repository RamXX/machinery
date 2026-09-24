---
id: MAC-idn3
title: "contributor lane: go/python runtime identity probes time out under host load with a bare TIMEOUT"
status: closed
priority: 2
type: bug
labels: [flaky-under-load, integration-lane, custody]
created_at: 2026-09-08T12:15:30Z
created_by: ramirosalas
updated_at: 2026-09-24T21:31:55Z
content_hash: "sha256:77c30e7f8640dfc96bae549ab0370ebd389659b5ec536cb74acf5101268d8528"
closed_at: 2026-09-24T21:31:55Z
close_reason: "Root cause was MAC-ipa1, fixed by 2db00b90 (demux dropped fast job result -> bare TIMEOUT). Diagnostic-wording residual not worth tracking. Triage 2026-09-24."
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
- 2026-09-24T21:31:55Z status: open -> closed

## Links


## Comments

### 2026-09-08T13:27:22Z ramirosalas
Likely the same root cause as MAC-ipa1, fixed on fix/custody-budgets in 1170df1.

The go/python identity probes are exactly the shape that loses a result there: a job short enough (go version, python3 --version) to finish before its caller is rescheduled. The demux delivered the result and retired the registration before Run claimed the job channel, so Run registered a second empty channel, waited out the whole probe deadline and reported a bare 'processscope: TIMEOUT'. That matches the symptom, the load dependence, and the bare TIMEOUT diagnostic with no budget information.

If so, the probe budget needs no change and no retry: the probe was never slow, its result was dropped. Leaving this open for the owner to re-run the lane under load and confirm before deciding whether the diagnostic ask still stands.
