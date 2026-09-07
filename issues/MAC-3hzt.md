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
updated_at: 2026-09-07T22:29:14Z
content_hash: "sha256:ead0d6307e03ec01cd1fa660d6fe959537eb35c627e6ee465314372227d6f5ec"
follows: [MAC-62s6]
labels: [accepted]
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

### 2026-09-07T22:29:14Z ramirosalas
ACCEPTED 2026-09-07 — RED b8e4c91 (coarse-tick journal blindness 5/5 deterministic; benign control green) -> GREEN 4eee40c. Sticky-latched inode-bound mutation sentinel on the journal fd (inotify /proc/self/fd/N Linux; kqueue NOTE_WRITE|EXTEND|REVOKE BSD; no-channel fallback elsewhere) drained as final conjunct in requireHeld/requireHeldAfterUnlink/refreshAfterRename; formalJournalChangeID routed through the o82q coarsener seam; all stat conjuncts kept; no frozen tests amended. Linux confirmation on original flake host: full formal suite rc=0 114/0/0; retained-handle ABA in-suite PASS + 5/5 isolated; coarse tests deterministic; JDK provisioned pin-verified. Race pass environment-blocked (no gcc on ad-hoc host; Linux -race rides hosted CI). 333+/4- in 5 files. Coordinator merged. Record: .git/machinery-evidence-20260906.TEFZ7D/3hzt-record.md; Linux REPORT 965b82e4.
