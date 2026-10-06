---
id: MAC-iku6
title: "verify-formal: the Alloy jar download has no retry; one connection reset failed the formal workflow on the 0.8.0 release commit"
status: open
priority: 2
type: bug
labels: [ci, formal, reliability]
created_at: 2026-09-10T17:32:53Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:51Z
content_hash: "sha256:536c435819713fd58991923ff40b2791bcf7867a5b05814edc1eda706030d92f"
---

## Description
Observed 2026-09-10 on release commit 39e2c9ff, formal run 34503459934 attempt 1: verify-formal over examples/fulfillment/design reported '8 passed, 1 failed' after all eight TLA specs passed; the failed item was the Alloy Integrity check, whose jar fetch from release-assets.githubusercontent.com died with 'read: connection reset by peer'. A rerun of the failed job passed and the release proceeded. The Dagger CLI download in .github/actions/dagger already retries (92592a86); the Alloy jar fetch in the formal engine does not. Acceptance: the Alloy jar fetch retries a bounded number of times on transient network errors (reset, EOF, 5xx) with backoff, verifies the pinned checksum after each attempt, and reports the final failure with the attempt count; a unit test drives the fetch through a flaky test server. Consider the same treatment for the TLC jar fetch if it shares the path.

## Acceptance Criteria


## Design


## Notes
Triage 2026-09-24: valid. Single attempt in internal/formal/formal.go fetchJarContext (~427, Do at ~494). Dagger CLI fetch retries since 92592a86; jar fetch does not. Independent of zafm/y8lj.
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Add bounded retry with backoff on transient errors (reset, EOF, 5xx), re-verify checksum per attempt, report attempt count, unit test with a flaky server; same path covers TLC jar. Evidence: internal/formal/formal.go:427 fetchJarContext does a single http.DefaultClient.Do (line ~494) with no retry; only retry text is the checksum-mismatch message at :458. Triage note in the issue is confirmed. CHANGELOG has no jar-fetch retry. Notes: Release-path CI flake (hit on the 0.8.0 release commit); small, self-contained.

## History


## Links


## Comments
