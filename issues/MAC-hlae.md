---
id: MAC-hlae
title: "Reject unmodeled saga compensation paths"
status: open
priority: 0
type: bug
labels: [hard-tdd, walking-skeleton]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:26Z
created_by: ramirosalas
updated_at: 2026-09-05T19:38:44Z
content_hash: "sha256:13cb828bcee1026358744e1713ca051bd79c3658b145fc71d6dddb3d1d99076a"
blocks: [MAC-ou97]
---

## Description
## USER INTENT
Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.

## Context (Embedded)
Assessment F1: setting fulfillment Compensating.after.compensateTimeout.target from compensateRetry to Failed was accepted by EmitSaga with byte-identical output; real verify-formal returned 11 passed. Composition Generate likewise ignores Compensating.invoke.onDone redirect Failed -> Completed. Source reconciliation explicitly admits timeout Failed while emitted CompensateDone requires all obligations clean. Whole-machine closure must match the abstraction, not just forward routes.

## Ownership
Own only these paths and directly associated tests: internal/refine/refine.go, internal/compose/compose.go, internal/refine/saga_closure_test.go, internal/compose/saga_closure_test.go, cmd/machinery/saga_closure_test.go. You are not alone in this codebase; preserve other edits and coordinate any shared-file changes with dispatcher.

## Boundary Map
PRODUCES:
- internal/refine/refine.go -> hardened behavior and regression proof
- internal/compose/compose.go -> hardened behavior and regression proof
- internal/refine/saga_closure_test.go -> hardened behavior and regression proof
- internal/compose/saga_closure_test.go -> hardened behavior and regression proof
- cmd/machinery/saga_closure_test.go -> hardened behavior and regression proof
CONSUMES:
- Existing Machinery source interfaces.
  spec: refine.ReconcileSaga(machine, sem *ir.Value) (err error); refine.EmitSaga(machine, sem *ir.Value, sourceNames [2]string) (mid string, files map[string]string, err error); compose.Generate(comp, machine *ir.Value, machineName string) (name, tla, cfg string, err error)

### Story Acceptance Criteria
1. Running machinery refine/compose/verify-formal on an otherwise valid machine with timeout to clean Failed while cleanup may be outstanding rejects that route with a precise nonzero diagnostic before successful proof or publication, or faithfully models it and produces the expected counterexample; never byte-identical successful proof of the safer model.
2. Reconcile the entire compensation subgraph consistently across both generators: completion/error/timeout, retry exhaustion/backoff, guarded/array/nested extra routes and final states. Unsupported transitions fail closed; do not erase behavior or weaken invariants to regain green.
3. Positive existing fulfillment and checkout examples retain intended success/clean-failure/FailedDirty paths. Establish typed public Go signatures, explicit error handling, existing configuration registration discipline, fail-closed security/provenance checks, deterministic output and no unexpected artifact writes.
4. Add enumerated semantic mutations redirecting timeout/completion/error/retry/final-state escape and obligation removal. Each is rejected for the intended reason or has an explicit model counterexample; an unchanged positive control must succeed.
5. Walking skeleton establishes unit tests, Integration tests: MANDATORY (no mocks), and real CLI E2E tests, with no stubs or skip-if-missing. Hard TDD RED first commits runtime assertion failures reproducing the bug; GREEN preserves frozen RED tests. Targeted tests only here; final full preflight belongs to epic gate.

## Testing Requirements
- Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
- Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
- Commands: go test ./internal/refine ./internal/compose -run Saga; go test ./cmd/machinery -run SagaClosure; build a temporary machinery binary and run real verify-formal on isolated valid and mutated fulfillment designs using pinned TLC. Do not modify shared examples during RED; copy fixtures to t.TempDir. No full preflight.
- No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.

## OUT OF SCOPE
- Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
- Global heavy preflight, main merge and local release binary belong to final epic gate.

## DIFF BUDGET
- ~5-8 files, under 900 changed LOC; material overrun requires PM investigation, not weakened requirements.

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
- 2026-09-05T19:36:14Z dep_added: blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-ou97]]

## Comments
