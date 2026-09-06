---
id: MAC-sd7g
title: "Run required checks against the exact replay state"
status: open
priority: 1
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T09:03:33Z
created_by: ramirosalas
updated_at: 2026-09-06T09:03:33Z
content_hash: "sha256:0ef34878b521b27e3047abc95479ff409aa2b11eb55322a475cd2f246341eb52"
blocked_by: [MAC-wi2u, MAC-avfp, MAC-imtz]
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
- internal/assuranceflow/checks.go -> internal/assuranceflow/checks.go -> closed private production CheckExecutor implementation with CheckRequest{InvocationID,StateID string;Inputs InputView;Checks []Check;SourceDigest,ControlDigest string;Scope processscope.Scope;Runtimes []RuntimeHandle}; Run(context.Context, CheckRequest) (PendingChecks, error). PendingChecks.Provisional() ([]CheckResult, error), Final() ([]CheckResult, error), private owner-bound invocation nonce and exact section 7 profiles.
- internal/assuranceflow/checks_test.go -> owned artifact for the same bounded contract
- internal/assuranceflow/checks_integration_test.go -> owned artifact for the same bounded contract
- testdata/integration-lanes/assurance-checks.json -> owned artifact for the same bounded contract

### CONSUMES
MAC-6h0s: internal/tdd/types.go
  spec: CheckExecutor/CheckRequest/PendingChecks/CheckResult with InvocationID,StateID,Profile,SourceDigest,ControlDigest,Status and Diagnostics; no deserialized or callback-supplied production executor.
MAC-wi2u: internal/tdd/adapters/go.go
  spec: Adapter Prepare(context.Context, SuiteRequest) (PreparedSuite,error), Run(context.Context,PreparedSuite,EventSink)(Execution,error); exact Go RuntimeHandle.
MAC-avfp: internal/tdd/adapters/typescript.go
  spec: same closed Adapter interface and exact Node/TypeScript RuntimeHandle.
MAC-imtz: internal/tdd/adapters/python.go
  spec: same closed Adapter interface and exact CPython RuntimeHandle.
MAC-8yai: internal/tdd/adapters/elixir.go
  spec: same closed Adapter interface and exact Elixir/OTP RuntimeHandle.
MAC-pe9v: internal/gates/suite.go
  spec: RunOptions.Execution *GateExecution, ExecutionRequired bool; Snapshot.RunSelected(impl string,sel Selection,opt RunOptions) []*Gate.

### ACCEPTANCE CRITERIA
1. Implement exactly machinery-design/v1, machinery-architecture/v1, go-format/v1, go-vet/v1, typescript-typecheck/v1, python-compile/v1, elixir-format/v1 and elixir-compile/v1. Reject unknown/arbitrary shell profiles or user callbacks/reporters; python-compile is compilation, not a claimed linter.
2. machinery-design/v1 runs the normal canonical selection minus ONLY G4/Gt/Ga/Gv/Gtd; machinery-architecture/v1 runs exactly G4/Gt. Both use the exact captured state/inventory and current scoped execution where needed, not a stale hand-selected fixture or different checkout. Required unsupported checks block.
3. Native language profiles use fixed approved argv, real pinned runtimes and exact captured source/frozen configs in private scratch. All producers are scoped, bounded by remaining milestone deadline and shared cleanup; preparation, runtime probes and check execution each consume the same aggregate budget.
4. CheckResult binds private invocation and state, source/control digest and profile. PendingChecks cannot become Final before every relevant InputView release; release/check/cleanup errors invalidate provisional success. Hand-authored/imported results, wrong invocation/state, stale digest and non-owner finalization fail.
5. Real integration exercises every profile with valid and failing source, type/format/vet/compile failure, wrong frozen config, missing runtime/profile, source mutation and cancellation. No mocks or fabricated check output as integration proof. Prove required normal gates actually ran and that removing a required check/profile fails.
6. Provide normal production registry consumption to replay/final assuranceflow, plus a mandatory native contributor fragment covering actual process call graph and cumulative budget across multiple individually-fast checks. No acceptance claim until downstream normal command integration is delivered.

### TEST COMMANDS
Run targeted native package tests for owned files and exact required contributor fragments; record selected test identities and exact commands in RED. Do not run heavy preflight.

### OUT OF SCOPE
No new formatter/linter frameworks or arbitrary command adapters. Replay phase semantics and final verification sealing remain separate consumers.

### DIFF BUDGET
~7 files, under 1500 changed LOC including real check fixtures.

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

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-06T09:10:05Z dep_added: blocked_by MAC-wi2u
- 2026-09-06T09:10:05Z dep_added: blocked_by MAC-avfp
- 2026-09-06T09:10:06Z dep_added: blocked_by MAC-imtz

## Links
- Parent: [[MAC-ui8a]]
- Blocked by: [[MAC-wi2u]], [[MAC-avfp]], [[MAC-imtz]]

## Comments
