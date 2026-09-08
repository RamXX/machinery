---
id: MAC-y5ho
title: "assuranceflow: writer.lock mkdir returns EINVAL under host load in TestRegisterConflictingWriters"
status: open
priority: 2
type: bug
labels: [flaky-under-load, assuranceflow]
created_at: 2026-09-08T08:21:09Z
created_by: ramirosalas
updated_at: 2026-09-08T08:21:09Z
content_hash: "sha256:4011eddea2a5038416c3323e3cf197bdaf723c85a873e58acbf6f842bf8255c4"
---

## Description
## Symptom
During the pre-push full race sweep on the 0.7.0 candidate (d1f2264), with two other agents running targeted Go test suites on the same host (load average about 5):

```
--- FAIL: TestRegisterConflictingWriters (2.60s)
    register_test.go:558: unexpected error: CUSTODY_ERROR: acquiring store writer: mkdir /var/folders/gh/.../T/TestRegisterConflictingWriters3114919719/002/store/staging/writer.lock: invalid argument
FAIL	github.com/RamXX/machinery/internal/assuranceflow	31.664s
```

In isolation the test passes: `go test -race -count=3 -run TestRegisterConflictingWriters ./internal/assuranceflow` is ok in 2.8 s.

## Why it matters
`mkdir` returning EINVAL is not a contention outcome (EEXIST is). Either the writer-lock path is built from a value that can be malformed under concurrent registration (a path component with an unexpected byte, an over-long component, or a parent that was renamed away mid-call), or the error is being mis-mapped. The conflicting-writers test exists precisely to prove the store stays correct under contention, so an unexplained EINVAL under load is a correctness question, not test noise.

## Ask
Reproduce under load (run the package with -count=20 while another race sweep runs), capture the exact path and errno at the mkdir call, and either fix the path construction or prove the failure is a macOS tmpfs artifact and harden the test setup.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
