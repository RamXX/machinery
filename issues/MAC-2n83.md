---
id: MAC-2n83
title: "Recover interrupted publication without losing ownership"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:33:46Z
created_by: ramirosalas
updated_at: 2026-09-07T07:22:43Z
content_hash: "sha256:5f933a56b8c6422d0f73b4532a5ca7eedfb30721f9d62ce14e16d3d259a5668c"
blocks: [MAC-vx24, MAC-ou97, MAC-u4oo]
was_blocked_by: [MAC-hpqp]
follows: [MAC-hpqp]
assignee: dev-MAC-2n83
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
ANCHOR ROUND-1 AUTHORITATIVE EXECUTION-LANE REPAIR
General rule: required runtime tests need deterministic provisioning, explicit closed inventory, actual native execution accounting and teardown. Missing infrastructure must fail the REQUIRED lane, not be silently skipped. Ordinary native suites may explicitly exclude registered service-backed tests using a dedicated build tag.
Classification: Docker bind-mount/host-identity publication scenarios require real daemon; native process/filesystem recovery tests stay ordinary native.
Split service-backed cases into cmd/machinery/recover_integration_test.go with dedicated lane selection; do not put Docker-required cases unconditionally in existing native recovery tests.
PRODUCES:
- testdata/integration-lanes/recovery.json -> this story's closed suite fragment, with exact source/test IDs, runtime/pin requirements and bounded execution configuration
- cmd/machinery/recover_integration_test.go -> actual named runtime cases registered in the fragment
CONSUMES:
- MAC-hpqp: testdata/integration-lanes/schema.json
  schema: Closed versioned native-runner suite fragment with exact source/test identities, runtime requirements and bounded command selection.
- MAC-hpqp: scripts/integration-lane/main.go
  endpoint: go run ./scripts/integration-lane --lane required (Makefile test-integration invokes same entrypoint).
Additional acceptance criteria: fragment matches actual test sources both directions; all registered cases actually start/terminate with expected positive/negative outcomes; no cached/skipped/empty/partial/fabricated-summary success; real provisioned positive and missing-runtime/fresh-cache failure diagnostics; no owned container/process leaks. Required local preflight and hosted CI execute the same union. Do not edit shared root inventory; own only this fragment. RED source, fixture, fragment and runner configuration are frozen together after review. Any exact test names introduced in RED must remain registered through GREEN.
No heavy preflight until final gate; no GitHub mutation; no active installation replacement. This note supersedes any earlier command implying service-backed tests execute in unprovisioned ordinary package suites.

## History
- 2026-09-05T19:35:07Z dep_added: blocks MAC-vx24
- 2026-09-05T19:36:16Z dep_added: blocks MAC-ou97
- 2026-09-05T19:45:32Z dep_added: blocked_by MAC-hpqp
- 2026-09-06T09:10:11Z dep_added: blocks MAC-u4oo
- 2026-09-07T04:45:25Z dep_removed: was_blocked_by MAC-hpqp
- 2026-09-07T04:45:47Z status: open -> in_progress
- 2026-09-07T04:45:47Z auto-follows: linked to predecessor MAC-hpqp

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-ou97]], [[MAC-u4oo]]
- Was blocked by: [[MAC-hpqp]]
- Follows: [[MAC-hpqp]]

## Comments

### 2026-09-07T07:22:43Z ramirosalas
ACCEPTED 2026-09-06 — RED 004e384 (4 reviewed corrections documented) -> GREEN 5c89fc6..749c1ae. InspectRecovery (strictly read-only: writer/outputs/journal inventory, native-identity statuses, live-writer probe, finalize/rerun/conflict taxonomy) + RecoverInterrupted (fully-validated transactions only, via the publication machinery's own resume/clear path) + new recover CLI in main.go. Native file identity preserved (no hash-equality shortcut); refusals: swapped/symlink-escape/mismatch/tampered-journal/concurrent-writer/partial. Real subprocess crash + Docker bind-mount exit-9 interruption + host-side recovery proven; idempotent second recovery harmless; fixture container cleaned by exact ID w/ label verification (t.Cleanup born in RED). Overrun +2182/-14 vs ~1300 (78% tests) — disclosed, proof not trimmed (established accepted pattern). Coordinator verified designlock+cmd suites, merged; lane green on epic (5 suites). Record: .git/machinery-evidence-20260906.TEFZ7D/2n83-record.md
