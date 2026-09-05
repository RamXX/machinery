---
id: MAC-2n83
title: "Recover interrupted publication without losing ownership"
status: open
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:33:46Z
created_by: ramirosalas
updated_at: 2026-09-05T19:38:45Z
content_hash: "sha256:8276659c31867f26d7d80650a7a233b493eb15b362b31d832bcf388c4d3492f8"
blocks: [MAC-vx24, MAC-ou97]
---

## Description
## USER INTENT
Strengthen Machinery mission-critical assurance with observable fail-closed behavior and precise limits.

## Context (Embedded)
NEXT4 legitimate recovery gap but diagnosis imprecise: outer publication input fingerprint already content/mode while lower artifact journals rely on native identities. A safe inspect/recover user path is preferable to manual sentinel deletion. Preserve capability-rooted path/swap protections. New recover command is explicitly proposed surface to implement, not an existing API.

## Ownership
Own only internal/designlock/designlock.go, internal/designlock/recovery_inspection_test.go, cmd/machinery/recover.go, cmd/machinery/recover_test.go, cmd/machinery/main.go and directly associated tests. You are not alone: preserve other edits and coordinate shared paths.

## Boundary Map
PRODUCES:
- internal/designlock/designlock.go -> hardened contract and regression evidence
- internal/designlock/recovery_inspection_test.go -> hardened contract and regression evidence
- cmd/machinery/recover.go -> hardened contract and regression evidence
- cmd/machinery/recover_test.go -> hardened contract and regression evidence
- cmd/machinery/main.go -> hardened contract and regression evidence
CONSUMES:
- Existing Machinery implementation.
  spec: designlock.Acquire(designRoot string) (*Lock, error); (l *Lock) ResumeExpected(writer, recovery string) error; (l *Lock) PublishExpected(writer, recovery string, expected []OutputExpectation, fn func() error) error

### Story Acceptance Criteria
1. Provide read-only inspection of interrupted publication identifying writer, expected outputs, stage/journal and actionable safe recovery path without deleting evidence or changing artifacts.
2. Recover a fully published content-and-mode matching transaction on a fresh process/host identity only after validating exact expected input/output inventories, no live writer, journal ownership and rooted paths; if automatic safe recovery is impossible, preserve everything and explain exact conflict.
3. Do not globally replace native file identity with hash equality. Reject swapped files/directories, symlink escapes, mismatched content/mode, unexpected output, tampered journals, concurrent writer and partial publication lacking complete proof.
4. Reproduce interruption at meaningful publication boundaries using real subprocess crash and real filesystem; positive resume completes consistent artifacts, idempotent second recovery is harmless, negative cases leave user files/journal recoverable.
5. Exercise actual Docker bind-mounted publication plus host inspection/recovery where available (required local Docker, no skip); distinguish already-complete cross-identity publication from rollback of partial native transactions. Never delete broad directories/sentinels as generic remediation.

## Testing Requirements
- Explicit hard TDD: separate RED test author, expected behavioral assertion failure plus passing control on unmodified production, independent PM replay, frozen RED test/fixture bytes retained through GREEN. Compilation, import, timeout or unavailable infrastructure are not valid RED. Existing tests encoding the unsafe contract may be corrected during reviewed RED; GREEN cannot silently weaken them.
- Integration tests: MANDATORY (no mocks). Exercise real process/filesystem/runtime boundaries; no stubs, no skip-if-missing. Fixture policy inputs may test parser logic but are not live remote proof.
- go test ./internal/designlock -run 'Recover|Resume|Publish'; go test ./cmd/machinery -run Recover; real isolated subprocess and Docker-bound temporary-root recovery integration.
- Full scripts/preflight.sh ONLY at end of epic. No pushes, sync, remote mutation, installed binary/plugin/agent/skill replacement, or dev-link. Another agent uses installed Machinery in NIL; builds/tests use isolated output and homes.
- Shipped product enforcement is standalone Machinery: never requires pvg, nd, Paivot labels/metadata or commit conventions. Paivot is local development coordination only.

## OUT OF SCOPE
- Other assessment areas have sibling stories; final preflight/local main merge/isolated candidate binary are final gate responsibilities. Do not omit small directly related safety fixes.

## DIFF BUDGET
- ~5-9 files, under 1300 changed LOC; overrun triggers PM investigation rather than weaker proof.

## MANDATORY SKILLS
- developer; codebase-memory; pm_acceptor.

## Delivery Requirements
Append exact RED/GREEN SHAs, commands/output, ownership inventory and per-AC proof. Use pvg story deliver; PM accepts separately. Never invoke pushing pvg story merge.

## nd_contract
status: new

### evidence
- Created 2026-09-05 from assessment and source-verified interfaces.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-05T19:35:07Z dep_added: blocks MAC-vx24
- 2026-09-05T19:36:16Z dep_added: blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-ou97]]

## Comments
