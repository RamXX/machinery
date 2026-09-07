---
id: MAC-3hzt
title: "Bug: journal recovery ABA witness still ctime-only (coarse-clock flake)"
status: open
priority: 1
type: bug
assignee: dev-jrn
parent: MAC-ui8a
created_at: 2026-09-07T21:57:39Z
created_by: ramirosalas
updated_at: 2026-09-07T21:57:39Z
content_hash: "sha256:25753ea883e8ac5a9596b3000e11a8eeb7818bf433a4f46aeac6391e06ad5e18"
---

## Description
Found during MAC-o82q Linux confirmation (REPORT f3c2d8f4907283b63f7286fc905e5f48d90d76769e1c7e3c5b0f5d30db0b7e90): TestFormalRecoveryRejectsJournalContentABAThroughRetainedHandle failed in full-suite run on Linux 6.8.0 (passed isolated; load-dependent tick luck; probe shows 6/8 same-tick mutations leave ctime unchanged). formalJournalChangeID (internal/formal/transaction.go:439) still compares raw ctime. Extend the accepted o82q hybrid mutation-event witness (inotify/kqueue conjunct, internal/formal/directory_mutation_bsd.go pattern) to the journal recovery path. Release-relevant: flaky formal-suite test can randomly fail the formal workflow the release gate requires.

## Acceptance Criteria


## Design


## Notes


## History


## Links
- Parent: [[MAC-ui8a]]

## Comments
