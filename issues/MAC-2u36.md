---
id: MAC-2u36
title: "Converge installer reruns on recorded targets"
status: open
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:27Z
created_by: ramirosalas
updated_at: 2026-09-05T19:30:27Z
content_hash: "sha256:6495e6487843a4282de560a9be03f1d6ec85e477eafb17237f8855ed963d2f2a"
---

## Description
## USER INTENT
Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.

## Context (Embedded)
NEXT1: installer always passes --bootstrap-defaults; updatePlan branches to default homes before receipt planning, so recorded native targets stay stale. Receipt-based normal update already exists. NEXT2 plugin discovery is ownership-critical; do not degrade uncertain ownership to blind fallback.

## Ownership
Own only these paths and directly associated tests: internal/install/update.go, internal/install/bootstrap_receipt_test.go. You are not alone in this codebase; preserve other edits and coordinate any shared-file changes with dispatcher.

## Boundary Map
PRODUCES:
- internal/install/update.go -> hardened behavior and regression proof
- internal/install/bootstrap_receipt_test.go -> hardened behavior and regression proof
CONSUMES:
- Existing Machinery source interfaces.
  spec: updatePlan(opts UpdateOptions) (refreshPlan, error)

## Acceptance Criteria
1. When a valid receipt exists, --bootstrap-defaults uses its complete recorded home/native/plugin target plan so installer rerun converges identically to machinery update; no-receipt first bootstrap retains defaults.
2. Malformed/stale/unsafe receipt fails closed with actionable diagnostic, never silently selects defaults. Explicit homes/targets remain incompatible with bootstrap as before.
3. Positive real isolated install followed by installer-equivalent rerun updates binary and all recorded targets, verifies receipts/digests, and is idempotent; test multiple homes plus native targets.
4. Negative tests cover corrupt receipt, plugin ownership discovery failure even under --skip-plugins, disappeared target, mixed copy modes and interrupted update rollback without altering unrelated host files.
5. Integration uses temporary home/target directories and actual built CLI update/install flow, not only updatePlan. No live user installation mutations in story tests.

## Testing Requirements
- Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
- Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
- Commands: go test ./internal/install -run 'Bootstrap|Receipt|UpdatePlan'; scoped cmd install/update integration using temporary configured roots.
- No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.

## OUT OF SCOPE
- Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
- Global heavy preflight, main merge and local release binary belong to final epic gate.

## DIFF BUDGET
- ~2-4 files, under 450 changed LOC; material overrun requires PM investigation, not weakened requirements.

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


## History


## Links
- Parent: [[MAC-ui8a]]

## Comments
