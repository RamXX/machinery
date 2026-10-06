---
id: MAC-wbxq
title: "Expose current assurance freshness without claiming replay"
status: open
priority: 2
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T09:05:11Z
created_by: ramirosalas
updated_at: 2026-10-06T04:04:16Z
content_hash: "sha256:3d6cf4f58f4f5f8f9d24a41ee890a5c6aa0b3cfe15a6a7d952a7af9b0d79dc97"
blocked_by: [MAC-pe9v]
blocks: [MAC-rau8, MAC-vx24]
was_blocked_by: [MAC-lnu6, MAC-p9z1]
related: [MAC-zti7]
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
- internal/gates/tdd.go -> internal/gates/tdd.go -> CheckTDDAssurance(design, impl string, inventory tdd.Inventory, status tdd.StatusReport) *Gate; RunOptions.TDDStatus *tdd.StatusReport, TDDRequired bool; Gtd suite registration/selection and required-status validation. Normal gate output always says replay not performed for this cheap path.
- internal/gates/tdd_test.go -> owned artifact for the same bounded contract
- internal/gates/suite.go -> owned artifact for the same bounded contract
- internal/gates/tdd_suite_test.go -> owned artifact for the same bounded contract

### CONSUMES
MAC-p9z1: internal/tdd/status.go
  spec: Status(ctx context.Context,req StatusRequest)(StatusReport,error); independent source/control/judgment/store-head state dimensions.
MAC-6h0s: internal/gates/assurance_inventory.go
  spec: AssuranceInventory(design string)(tdd.Inventory,error).
MAC-pe9v: internal/gates/suite.go
  spec: RunOptions.Execution *GateExecution and ExecutionRequired bool; preserve these fields while adding TDDStatus/TDDRequired.
MAC-lnu6: cmd/machinery/tokensequal.go
  source: existing frozen-byte policy repair; no tokens-equal permission to mutate tests.

### ACCEPTANCE CRITERIA
1. Add exact CheckTDDAssurance and Gtd suite wiring. Required assurance with nil/mismatched/stale StatusReport blocks; callers cannot omit a required manifest by selecting other gates. Distinguish draft/unregistered, current, stale and historical milestone evidence with actionable reasons.
2. Cheap Gtd checks identity/freshness only and always explicitly reports replay not performed. It must never convert a matching hash, old milestone acceptance or user-writable receipt into proof of actual current execution.
3. Validate full qualified inventory/current milestone coverage, exact store/source/control/judgment identity dimensions and registered-head relationship. Missing manifests, dropped owner/child, same-ID different owner, current-code change, whitespace/mode/topology test mutation and unverified runtime disposition block strict eligibility.
4. Keep existing accepted full-root Gv capture/exclusion and ancestor Ga semantics unchanged; Gtd adds current assurance rather than relabeling history. No Paivot metadata, label or branch convention is consulted.
5. Real snapshot integration calls Snapshot.Select/RunSelected with actual Status() over external stores and corrupted cases, proves normal selected Gtd runs, and preserves the execution-scoped Ga fields. No replay occurs in this cheap function; normal strict/complete replay is an explicit downstream consumer.

### TEST COMMANDS
Run targeted native package tests for owned files and exact required contributor fragments; record selected test identities and exact commands in RED. Do not run heavy preflight.

### OUT OF SCOPE
Owner replay/finalization and CLI/Stop activation are downstream; this gate cannot issue sealed Verification or import execution receipts.

### DIFF BUDGET
~5 files, under 1000 changed LOC including tests.

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

## Acceptance Criteria


## Design


## Notes
Triage 2026-09-24: suite.go changed 9 times since 09-07; refresh interface text.
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Add CheckTDDAssurance, Gtd suite wiring and RunOptions.TDDStatus/TDDRequired with replay-not-performed messaging. Evidence: CHANGELOG 0.7.0 says the assurance substrate ships with no CLI/gate consumer. Greps of *.go find no CheckTDDAssurance, GateExecution, TDDStatus, assurance_store or strict-check. internal/assuranceflow holds only register.go. internal/tdd has no execute.go, replay.go or evidence.go, and has CheckExecutor only as a type in types.go. internal/gates has assurance_inventory.go (MAC-6h0s) but no tdd.go and no Gtd registration. Notes: The pe9v dependency is real because suite.go gains TDDStatus beside Execution and the AC says to preserve those fields. If this is done first the two touch the same struct and merge. Note 0.11.0 added Gz, so the Gtd letter and selection grouping must be checked against current gates.go. tdd.Status already exists at internal/tdd/status.go:36.
0.11.0 took gate letter gz (Gz-threat). Pick a free letter (f, h, o, q) when this lands.

## History
- 2026-09-06T09:10:08Z dep_added: blocked_by MAC-p9z1
- 2026-09-06T09:10:08Z dep_added: blocked_by MAC-pe9v
- 2026-09-06T09:10:09Z dep_added: blocked_by MAC-lnu6
- 2026-09-06T09:10:10Z dep_added: blocks MAC-rau8
- 2026-09-06T09:10:27Z dep_added: blocks MAC-vx24
- 2026-09-06T09:10:27Z dep_added: blocks MAC-ou97
- 2026-09-06T23:26:04Z dep_removed: was_blocked_by MAC-lnu6
- 2026-09-07T10:08:35Z dep_removed: was_blocked_by MAC-p9z1
- 2026-09-24T21:33:55Z dep_removed: no_longer_blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-rau8]], [[MAC-vx24]]
- Blocked by: [[MAC-pe9v]]
- Was blocked by: [[MAC-lnu6]], [[MAC-p9z1]]
- Related: [[MAC-zti7]]

## Comments

### 2026-09-06T09:16:33Z ramirosalas
CANONICAL MACHINE-READABLE BOUNDARY MAP 2026-09-06
This repeats the existing ownership/signatures in the parser-supported form; it does not create additional scope or weaken any AC. All prior exact acceptance/testing requirements and holds remain current.

PRODUCES:
- internal/gates/tdd.go -> internal/gates/tdd.go -> CheckTDDAssurance(design, impl string, inventory tdd.Inventory, status tdd.StatusReport) *Gate; RunOptions.TDDStatus *tdd.StatusReport, TDDRequired bool; Gtd suite registration/selection and required-status validation. Normal gate output always says replay not performed for this cheap path.
- internal/gates/tdd_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/gates/suite.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/gates/tdd_suite_test.go -> owned bounded artifact; behavior and tests specified in the current story AC

CONSUMES:
- MAC-p9z1: internal/tdd/status.go
  spec: Status(ctx context.Context,req StatusRequest)(StatusReport,error); independent source/control/judgment/store-head state dimensions.
- MAC-6h0s: internal/gates/assurance_inventory.go
  spec: AssuranceInventory(design string)(tdd.Inventory,error).
- MAC-pe9v: internal/gates/suite.go
  spec: RunOptions.Execution *GateExecution and ExecutionRequired bool; preserve these fields while adding TDDStatus/TDDRequired.
- MAC-lnu6: cmd/machinery/tokensequal.go
  source: existing frozen-byte policy repair; no tokens-equal permission to mutate tests.
- MAC-l7m0: docs/test-assurance-contract.md
  schema: Exact approved public contract SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8; accepted delivery required, never an uncommitted external proposal.

Observable outcome: A user receives explicit current freshness diagnostics saying replay not performed; missing required status returns a blocking gate.

## nd_contract
status: new

### evidence
- Canonical boundary syntax reconciled without code or test changes.
- HOLD: independent Anchor backlog approval and accepted canonical contract required.

### proof
- [ ] AC #1: current story acceptance requirement remains pending.
- [ ] AC #2: current story acceptance requirement remains pending.
- [ ] AC #3: current story acceptance requirement remains pending.
- [ ] AC #4: current story acceptance requirement remains pending.
- [ ] AC #5: current story acceptance requirement remains pending.
