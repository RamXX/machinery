---
id: MAC-y5ho
title: "assuranceflow: writer.lock mkdir returns EINVAL under host load in TestRegisterConflictingWriters"
status: closed
priority: 3
type: bug
labels: [flaky-under-load, assuranceflow]
created_at: 2026-09-08T08:21:09Z
created_by: ramirosalas
updated_at: 2026-10-06T05:53:40Z
content_hash: "sha256:ca131f0da09c428b1a559815f05d652f2217fc9f98d44790d7f9d8d6c4b09044"
related: [MAC-qo6n]
closed_at: 2026-10-06T05:53:40Z
close_reason: "Fixed in 0.11.1 (a9b8ce58 RED, 69250f77 GREEN): writer acquisition retries after a staging identity change."
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
Triage 2026-09-24: still valid, unexplained. EINVAL maps to CUSTODY_ERROR at internal/tdd/registration_store.go:568-570; 65183fc1 predates the sighting so did not fix it. One darwin-local sighting since 09-08. Re-measure on the quiet lane MAC-qo6n defines.
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Reproduce under load with -count=20, capture path and errno at the mkdir, then fix path construction or prove a macOS tmpfs artifact and harden the test. Evidence: Verified at HEAD 8c620d8f (v0.11.0). EINVAL still maps to CUSTODY_ERROR at internal/tdd/registration_store.go:567-569 (acquire path, 'acquiring store writer'); only related commit is 65183fc1 (teardown ordering), no commit mentions EINVAL; no reproduction or hardening since the 09-24 triage. Notes: Sibling race MAC-1jhn is closed (e2db8e49). Re-measure on the quiet lane MAC-qo6n defines but that is not a technical prerequisite.

## History
- 2026-10-06T05:53:40Z status: open -> closed

## Links
- Related: [[MAC-qo6n]]

## Comments

### 2026-09-10T07:35:14Z ramirosalas
Related: MAC-1jhn is a second, distinct race in this same test. A reader enumerating ledger/heads/ observes a publisher's in-flight .publish-<nonce> entry (internal/tdd/store.go:173 stages the temp inside the directory it publishes into; store.go:592 rejects unknown entries) and fails INVALID_SCHEMA, not the EINVAL described here. Fixing either does not fix the other, so this title understates what TestRegisterConflictingWriters is catching. Observed 2026-09-10 on linux/amd64 kernel 7.0.0-30 in a container under the full non-race suite.
