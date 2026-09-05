---
id: MAC-yhg5
title: "Own checker container lifetime deterministically"
status: open
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:27Z
created_by: ramirosalas
updated_at: 2026-09-05T19:38:44Z
content_hash: "sha256:0d753d2ffd6dd679cc656cb76beea607ac74a8854499bf350280a0d2aff0920b"
blocks: [MAC-gcrr, MAC-ou97]
---

## Description
## USER INTENT
Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.

## Context (Embedded)
Assessment F4 actual processcontrol timeout killed docker CLI while daemon container remained running. runCheckerOCI uses docker run --rm without container ID capture/removal or cpu/memory/pids budgets. Existing network/read-only/capability restrictions must remain.

## Ownership
Own only these paths and directly associated tests: cmd/machinery/verify_checkers.go, cmd/machinery/checker_oci_lifecycle_test.go. You are not alone in this codebase; preserve other edits and coordinate any shared-file changes with dispatcher.

## Boundary Map
PRODUCES:
- cmd/machinery/verify_checkers.go -> hardened behavior and regression proof
- cmd/machinery/checker_oci_lifecycle_test.go -> hardened behavior and regression proof
CONSUMES:
- Existing Machinery source interfaces.
  spec: runCheckerOCI(engineArgs []string, image, platform string, checkerArgs []string, runtimeDigest string, timeout time.Duration, workDir string) (string, error)

### Story Acceptance Criteria
1. Every OCI checker run owns an unambiguous container identity; timeout, cancellation, output-limit breach and startup/runtime failures perform bounded daemon-side cleanup before reporting completion, or explicitly report cleanup failure. Never remove unrelated containers.
2. Enforce deterministic finite memory, CPU and PID budgets with bounded defaults and closed validated overrides if exposed. Preserve network=none, read-only root, capability/no-new-privileges restrictions and exact runtime closure binding.
3. Handle cancellation before/after container creation, missing ID, failed engine cleanup, already-exited container, SIGTERM-ignoring workload, huge output and repeated runs without orphan accumulation.
4. A real pinned local OCI image executes a success case and a timeout case; after each return actual docker inspect/list proves no owned running container remains. Use unique disposable names and clean only test-owned resources.
5. Target tests include resource exhaustion/invalid budgets and CLI propagation; no fake executable result substitutes for real daemon lifecycle proof. Docker missing/unavailable fails required integration, never skip.

## Testing Requirements
- Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
- Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
- Commands: go test ./cmd/machinery -run 'CheckerOCI|CheckerLifecycle'; required real-daemon full verify-checkers integration with an immutable synthetic checker image and bounded workloads. Preserve preexisting dagger container untouched.
- No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.

## OUT OF SCOPE
- Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
- Global heavy preflight, main merge and local release binary belong to final epic gate.

## DIFF BUDGET
- ~3-5 files, under 900 changed LOC; material overrun requires PM investigation, not weakened requirements.

## MANDATORY SKILLS
- developer for implementation; codebase-memory for discovery; pm_acceptor for independent acceptance.

## Delivery Requirements
Use pvg story deliver, never close. Append RED/GREEN SHAs, exact commands and outputs, test inventory, independent proof per AC, and any residual limits to shared nd. Do not use pushing pvg story merge.

## nd_contract
status: new

### evidence
- Created 2026-09-05; source signatures verified at assessment base 497419ab4512fcff765cd5feb27aed4c67b5608d.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified

## Acceptance Criteria


## Design


## Notes
AUTHORITATIVE USER CONSTRAINTS 2026-09-05: Machinery product must be standalone, never require Paivot/pvg/nd, workflow labels or commit conventions. Local development coordination only may use Paivot. Another agent uses installed Machinery in NIL: do not replace installed binary/plugins/skills/agents; no dev-link or live install/update. Build isolated candidate only. No GitHub push/mutation during work. Full scripts/preflight.sh only final epic gate. RED author may update preexisting tests that encode superseded unsafe behavior with explicit review and genuine assertion-failure proof; after RED approval freeze exact tests/fixtures/config bytes.

## History
- 2026-09-05T19:35:09Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:15Z dep_added: blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-gcrr]], [[MAC-ou97]]

## Comments
