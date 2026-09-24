---
id: MAC-iku6
title: "verify-formal: the Alloy jar download has no retry; one connection reset failed the formal workflow on the 0.8.0 release commit"
status: open
priority: 2
type: bug
labels: [ci, formal, reliability]
created_at: 2026-09-10T17:32:53Z
created_by: ramirosalas
updated_at: 2026-09-24T21:31:56Z
content_hash: "sha256:becaca3490f051ae1ba1b65236525b2ad2e2d67db3c2f73eb1fc3fddf74a4020"
---

## Description
Observed 2026-09-10 on release commit 39e2c9ff, formal run 34503459934 attempt 1: verify-formal over examples/fulfillment/design reported '8 passed, 1 failed' after all eight TLA specs passed; the failed item was the Alloy Integrity check, whose jar fetch from release-assets.githubusercontent.com died with 'read: connection reset by peer'. A rerun of the failed job passed and the release proceeded. The Dagger CLI download in .github/actions/dagger already retries (92592a86); the Alloy jar fetch in the formal engine does not. Acceptance: the Alloy jar fetch retries a bounded number of times on transient network errors (reset, EOF, 5xx) with backoff, verifies the pinned checksum after each attempt, and reports the final failure with the attempt count; a unit test drives the fetch through a flaky test server. Consider the same treatment for the TLC jar fetch if it shares the path.

## Acceptance Criteria


## Design


## Notes
Triage 2026-09-24: valid. Single attempt in internal/formal/formal.go fetchJarContext (~427, Do at ~494). Dagger CLI fetch retries since 92592a86; jar fetch does not. Independent of zafm/y8lj.

## History


## Links


## Comments
