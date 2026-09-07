---
id: MAC-3hzt
title: "Bug: journal recovery ABA witness still ctime-only (coarse-clock flake)"
status: in_progress
priority: 1
type: bug
assignee: dev-jrn
parent: MAC-ui8a
created_at: 2026-09-07T21:57:39Z
created_by: ramirosalas
updated_at: 2026-09-07T21:57:40Z
content_hash: "sha256:e5c7bbda668db67311b6b26418d11742896937e2603392e85afd2821c3de5de5"
follows: [MAC-62s6]
---

## Description
Found during MAC-o82q Linux confirmation (REPORT f3c2d8f4907283b63f7286fc905e5f48d90d76769e1c7e3c5b0f5d30db0b7e90): TestFormalRecoveryRejectsJournalContentABAThroughRetainedHandle failed in full-suite run on Linux 6.8.0 (passed isolated; load-dependent tick luck; probe shows 6/8 same-tick mutations leave ctime unchanged). formalJournalChangeID (internal/formal/transaction.go:439) still compares raw ctime. Extend the accepted o82q hybrid mutation-event witness (inotify/kqueue conjunct, internal/formal/directory_mutation_bsd.go pattern) to the journal recovery path. Release-relevant: flaky formal-suite test can randomly fail the formal workflow the release gate requires.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-07T21:57:40Z status: open -> in_progress
- 2026-09-07T21:57:40Z auto-follows: linked to predecessor MAC-62s6

## Links
- Parent: [[MAC-ui8a]]
- Follows: [[MAC-62s6]]

## Comments
