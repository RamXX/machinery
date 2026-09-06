---
id: MAC-pe9v
title: "Keep final acceptance Git queries in the verification scope"
status: open
priority: 1
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T09:03:33Z
created_by: ramirosalas
updated_at: 2026-09-06T09:03:33Z
content_hash: "sha256:367f9623b4d2c0ec3ef2b6af9f85f9c92bc38cd35c9fe2a7c59d81f136c4ee4d"
blocked_by: [MAC-cn7q]
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
- internal/runtimeclosure/git.go -> internal/runtimeclosure/git.go -> OpenGit(context.Context, GitRequest) (*Git, error), GitRequest{RuntimeRoot string;ExpectedClosure string;Scope processscope.Scope}; Git.Executable() string, Digest() string, Validate(context.Context, processscope.Scope) error, Close() error. internal/gates/suite.go -> GateExecution{Context context.Context;Scope processscope.Scope;Git *runtimeclosure.Git}; RunOptions.Execution *GateExecution, ExecutionRequired bool. Scoped Ga helpers per approved section 7.
- internal/runtimeclosure/git_test.go -> owned artifact for the same bounded contract
- internal/gates/accept.go -> owned artifact for the same bounded contract
- internal/gates/suite.go -> owned artifact for the same bounded contract
- internal/gates/accept_execution_test.go -> owned artifact for the same bounded contract
- testdata/integration-lanes/assurance-git.json -> owned artifact for the same bounded contract

### CONSUMES
MAC-cn7q: internal/processcontrol/scope.go
  spec: WithScope(ctx, scope), AttachScope(cmd, scope), ExitStatus(err); attach AFTER gitcontrol.Environment sanitization.
MAC-bz1y: testdata/integration-lanes/assurance-runtime-pins.json
  MAC-6h0s: real native Git 2.55.0 exact closure plus Linux amd64/Darwin arm64 required accounting.
Existing internal/gates/suite.go
  spec: (s *Snapshot) RunSelected(impl string, sel Selection, opt RunOptions) []*Gate; (s *Snapshot) Release() error. Existing accepted snapshot/finalization guarantees are preserved.

### ACCEPTANCE CRITERIA
1. Open and pin the actual Git 2.55.0 runtime closure through scoped native probes, validate complete executable/library/topology identity as required and expose the exact approved Git methods. Close is pure final byte/topology revalidation plus handle release; it cannot launch a process after no-launch.
2. Add GateExecution and RunOptions.Execution/ExecutionRequired, with strict flow always requiring a live matching scope/context/runtime. Missing, closed, mismatched or cancelled execution fails before spawn, never falls back to PATH or background context.
3. Implement checkAcceptanceWithExecution and runGitExactWithExecution(execution *GateExecution, dir string, args ...string), threading the SAME execution through resolveReviewCommit/Exact, gitHeadAt/Exact, gitCommitOf, gitIsAncestor, checkCommitBinding and checkCommitAncestry. Use context.WithTimeout(execution.Context, gitCommandTimeout), validated absolute Git and existing gitcontrol.Environment, followed by BOTH WithScope and AttachScope.
4. Preserve existing ordinary wrappers and historical milestone acceptance semantics while required scoped selection uses the new path. Actual Snapshot.RunSelected selecting Ga must exercise real Git through the scope; a new unused helper or fake Git executable cannot satisfy integration.
5. Real native two-platform tests cover valid commit binding/ancestry, wrong commit/tree, ancestor historical record versus current source, PATH spoofing, runtime replacement, invalid/missing scope, cancellation during Git query, timeout/overflow and cleanup failure. Ensure clean owned teardown and no surviving helper after cancellation.
6. Audit the final Ga reachable process call graph and record every launch site against its scoped owner. Any reachable producer not supported under this execution context fails UNSUPPORTED_EXECUTION before work. Preserve original tests; new supplemental required fragment is persistently invoked by lane CI/final preflight.

### TEST COMMANDS
Run targeted native package tests for owned files and exact required contributor fragments; record selected test identities and exact commands in RED. Do not run heavy preflight.

### OUT OF SCOPE
TDDStatus/Gtd and normal CLI/hooks are downstream; this story owns only RunOptions execution fields, scoped Ga and Git closure. Final owner sealing belongs to assuranceflow.

### DIFF BUDGET
~8 files, under 1500 changed LOC including real Git tests.

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
- 2026-09-06T09:10:04Z dep_added: blocked_by MAC-cn7q

## Links
- Parent: [[MAC-ui8a]]
- Blocked by: [[MAC-cn7q]]

## Comments
