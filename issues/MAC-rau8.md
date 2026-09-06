---
id: MAC-rau8
title: "Seal verification only after the complete native lifecycle"
status: open
priority: 1
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T09:05:11Z
created_by: ramirosalas
updated_at: 2026-09-06T09:05:11Z
content_hash: "sha256:90dfea7e0891c448a4c88a4119bd2ec1e138ff43b3e4c9dce0f2a51c2f8dcbae"
blocked_by: [MAC-sqpt, MAC-wbxq]
blocks: [MAC-u4oo, MAC-vx24]
---

## Description
### USER INTENT
Machinery and software produced with it must gain the strongest honest deterministic correctness guardrails, including assertion-based hard-TDD RED, frozen negative tests, native replay and fail-closed integration. Machinery is standalone: no Paivot product/runtime/build/test dependency, tracker metadata or external orchestration required.

### APPROVED CONTRACT
MAC-l7m0 produces docs/test-assurance-contract.md, public projection SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8. This story is blocked by its accepted delivery. Exact schemas, signatures, event/catalog constants, argv, errors and lifecycle in that permanent contract are normative and are NOT redesign authority. This body specifies a bounded implementation subset; the producer's delivered canonical document supplies its complete exact technical contract. Approval of that document is not implementation or native execution proof. Stop for architecture review if the native guarantee cannot be implemented; do not substitute a weaker mechanism.

### NON-NEGOTIABLE TESTING / DELIVERY
Hard-TDD is user-authorized: a separate RED author freezes exact test/helper/fixture/config/dependency-lock bytes, inventories all new test identities and meaningful expected assertion failures with passing controls before GREEN. Compile/import/setup/infrastructure failures are not accepted RED. For negative tests already passing safe-default RED, retain an explicit immutable unsafe implementation-only challenge causing the expected assertion failure and a safe control; do not require every negative fail on the ordinary stub. Any necessary update of superseded existing tests must be justified and approved in RED, never silently rewritten in GREEN. Existing accepted regressions and frozen prior-story inventories are preserved.
Integration tests: MANDATORY (no mocks). Real native process/filesystem/runtime paths, positive controls and enumerated adversarial cases. Pure parser/unit fixtures supplement but do not replace real runner/custody proof. No skip-if-missing, env-gated dormant tests, empty execution acceptance, fabricated output or fake executable as successful native proof. Required infrastructure cases are explicitly inventoried in the mandatory contributor lane and execute on hosted Linux amd64 and Darwin arm64 plus final local preflight; missing runtime/tool, missing leaf, skip or leak fails. Ordinary suites may exclude ONLY that explicitly closed required lane.
Use targeted tests while implementing; scripts/preflight.sh runs ONLY once the integrated epic is ready for its final gate. Record exact test commands/counts, source+RED SHAs, frozen-byte verification, platform/runtime identities, positive/negative results and actual producer-consumer call proof. Developer delivers, independent PM accepts; no self-acceptance.
Private coordination via pvg nd is not shipped. No remote mutation/push, installed binary/plugin/skill replacement, unrelated process/container teardown, runtime installation or broad cleanup without the root's explicit authorization. Final candidate is isolated; final merge/publish belongs to the root completion gate.

### MANDATORY SKILLS
developer for implementation and hard-TDD delivery; pm_acceptor for independent acceptance. None additional identified.

### BOUNDED OWNERSHIP
Only the PRODUCES files below and named supplemental fixtures; preserve others' edits and every prior frozen inventory. You are not alone in the codebase.

### PRODUCES
- internal/assuranceflow/run.go -> internal/assuranceflow -> Run(ctx context.Context, req Request, output io.Writer) (Verification, error); Request{Mode string;Design,Implementation,Store,Milestone string}, Mode red|green|verify|strict-check|complete-check. Private unexported-nonce immutable Verification constructed only after approved final lifecycle and successful output. assuranceflow imports gates/tdd/runtimeclosure/processscope; neither gates nor tdd imports assuranceflow.
- internal/assuranceflow/verification.go -> owned artifact for the same bounded contract
- internal/assuranceflow/run_test.go -> owned artifact for the same bounded contract
- internal/assuranceflow/finalization_integration_test.go -> owned artifact for the same bounded contract
- testdata/integration-lanes/assurance-finalization.json -> owned artifact for the same bounded contract

### CONSUMES
MAC-sqpt: internal/tdd/replay.go
  spec: Execute(ctx context.Context,req ExecuteRequest)(Candidate,error); Verify(ctx context.Context,req VerifyRequest)(Candidate,error), provisional only.
MAC-sd7g: internal/assuranceflow/checks.go
  spec: production CheckExecutor.Run(context.Context,CheckRequest)(PendingChecks,error); Final() unavailable until relevant views are released.
MAC-wbxq: internal/gates/tdd.go
  spec: CheckTDDAssurance(design,impl string,inventory tdd.Inventory,status tdd.StatusReport)*Gate; RunOptions.TDDStatus/TDDRequired.
MAC-pe9v: internal/gates/suite.go
  spec: RunOptions.Execution *GateExecution, ExecutionRequired bool; full Ga execution uses same owner scope and runtimeclosure.Git.
MAC-p9z1: internal/tdd/store.go
  MAC-6h0s: exact registered external head and immutable run-object publication, no execution head advance.

### ACCEPTANCE CRITERIA
1. Implement exact Request modes and private sealed Verification, never returned from a deserialized/hash-only/imported receipt or provisional Candidate. Open held immutable inputs, exact registered head, root scope and ALL needed runtime handles (including Git); launch tests/checks only in owned execution children.
2. Preserve the approved nine-stage finalization order: drain execution children with root/runtimes open; release original views to finalize pending Gv/checks; release read store reservation and writer-compare exact head; reacquire final original snapshot and compare all identities without rebinding; run actual full current strict/complete gates including Ga/Gv/Gtd in a closing child against the same Candidate without recursive Verify.
3. After closing-child probes/drain, enforce ROOT no-launch barrier, then pure RuntimeHandle.Close/Git.Close revalidation and materialization cleanup; final original view remains held and unchanged through all cleanup, then Release and resolve pending results. No late subprocess in runtime close, view release, store cleanup or output. A failure before final reacquire cannot acquire a new view merely to manufacture success.
4. Only after all final releases succeed publish durable raw/run records as recorded-only WITHOUT advancing the registration head; fsync/close/publication errors prevent seal. Finally check cancellation, render complete output successfully, then construct Verification. Output failure cannot return success even if recorded history exists; retain failed history.
5. Enforce one 14400000ms (4h) owner cap even with context.Background, no flag increase, including all milestones, waits/locks/final gates/publication/output. Every child receives remaining absolute deadline; per-milestone wall caps remain cumulative. First cancellation/error is terminal, stops new work and shares one max-selected cleanup allowance <=30000ms (default10000ms), not a renewed grace per child.
6. Native two-platform integration tests execute actual adapters/production checks/full gates/real Git while mutating each identity or failing each lifecycle boundary: original/final release, ABA/head CAS, pending Final early, runtime postcheck, cleanup, stream overflow, timeout, publication/fsync/output and cancellation. Verify no seal, no late launch, no stolen resource cleanup and no accidental head advance.
7. Audit the actual process-producing call graph from every Run mode through adapters/checks/Ga/runtime probes, with source-site to scope/deadline mapping; any uncovered reachable producer fails UNSUPPORTED_EXECUTION. Real cumulative deadline tests cover many individually-fast stages exceeding owner/milestone budget; virtual clocks only supplement native custody proof.
8. Expose normal callable Run to CLI/hooks and register the required finalization fragment in both native CI lanes. This producer's API proof does not claim public command adoption until the explicit CLI consumer is accepted.

### TEST COMMANDS
Run targeted native package tests for owned files and exact required contributor fragments; record selected test identities and exact commands in RED. Do not run heavy preflight.

### OUT OF SCOPE
Register transaction remains separately owned; no remote/installed binary change or final heavy preflight here. Hostile host isolation, proof of historical chronology and universal semantic test adequacy are explicitly not claimed.

### DIFF BUDGET
~8 files, under 2300 changed LOC including native finalization matrices.

## nd_contract
status: new

### evidence
- Approved contract decomposition, not implementation/native proof.
- HOLD: independent Anchor backlog approval and MAC-l7m0 acceptance required before normal developer dispatch.

### proof
- [ ] AC #1: executable evidence pending.
- [ ] AC #2: executable evidence pending.
- [ ] AC #3: executable evidence pending.
- [ ] AC #4: executable evidence pending.
- [ ] AC #5: executable evidence pending.
- [ ] AC #6: executable evidence pending.
- [ ] AC #7: executable evidence pending.
- [ ] AC #8: executable evidence pending.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-06T09:10:09Z dep_added: blocked_by MAC-sqpt
- 2026-09-06T09:10:10Z dep_added: blocked_by MAC-wbxq
- 2026-09-06T09:10:10Z dep_added: blocks MAC-u4oo
- 2026-09-06T09:10:27Z dep_added: blocks MAC-vx24

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-u4oo]], [[MAC-vx24]]
- Blocked by: [[MAC-sqpt]], [[MAC-wbxq]]

## Comments
