---
id: MAC-cn7q
title: "Keep formal subprocesses inside native custody"
status: open
priority: 0
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T08:58:18Z
created_by: ramirosalas
updated_at: 2026-09-06T08:58:18Z
content_hash: "sha256:4baa62fa1f3e76d000b9bd8f22f135f827d4f10596f47cd10f6eb5fb9ba92d39"
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
Only the PRODUCES files below and their named supplemental test fixtures; preserve others' edits and every prior frozen inventory. You are not alone in the codebase.

### PRODUCES
- internal/processcontrol/scope.go -> internal/processcontrol -> WithScope(ctx, scope), AttachScope(cmd, scope), ExitStatus(err), keeping Run(ctx context.Context, cmd *exec.Cmd) error compatible. Exact new API contracts in section 8 apply; do not invent alternate error semantics. Explicit formal/runtimeclosure call sites attach the verified scope after sanitized environments. cmd/machinery/main.go -> same-binary ServeInternal interception before ordinary Cobra parsing.
- internal/processcontrol/scope_test.go -> owned implementation/test artifact for the same contract
- internal/formal/custody_integration_test.go -> owned implementation/test artifact for the same contract
- internal/runtimeclosure/custody_integration_test.go -> owned implementation/test artifact for the same contract
- internal/processcontrol/run.go -> owned implementation/test artifact for the same contract
- internal/formal/formal.go -> owned implementation/test artifact for the same contract
- internal/runtimeclosure/java.go -> owned implementation/test artifact for the same contract
- cmd/machinery/main.go -> owned implementation/test artifact for the same contract

### CONSUMES
MAC-qlw2: internal/processscope/scope.go
  spec: Scope.Run/Child/Attach/Close and ServeInternal(args []string, io InternalIO) (handled bool, exitCode int); first authenticate inherited authority, then normal command parsing if unhandled.
Existing source: internal/processcontrol/run.go -> Run(ctx context.Context, cmd *exec.Cmd) error; internal/formal/formal.go -> VerifyFormalTo(design string, genOnly bool, stdoutW, stderrW io.Writer) (exitCode int); internal/runtimeclosure/java.go -> OpenJava() (*Java, error), Environment(home, temp, javaPath string) []string.

### ACCEPTANCE CRITERIA
1. Keep ordinary Run compatibility while scope-aware calls use verified custody, bounded output and the approved exit-status accessor. Audit every affected direct *exec.ExitError inspection and preserve error-vs-assertion classification. Malformed ExtraFiles or conflicting SysProcAttr fails closed; no scoped-to-unscoped fallback.
2. Wire actual formal provisioning/probe/verification subprocesses, including nested Java/JAR/TLC paths, through WithScope and AttachScope AFTER Environment sanitation, retaining opened Java/JAR byte identity and existing checksum/lock/publication guarantees. Environment reconstruction must not silently strip custody.
3. Handle authenticated ServeInternal in the candidate Machinery binary before normal command parsing. Malformed/untrusted internal activation is rejected without normal product execution. Normal CLI behavior remains compatible when no internal request exists.
4. Supplemental real native tests on both supported platforms observe an actually running pinned nested JVM before cancellation, both during provisioning and actual formal/meta verification. Assert owned JVM descendants stop and unrelated process/container identities survive. A fake Java script or provisioning-only wrapper cannot satisfy this criterion.
5. Inventory actual process-producing call sites reachable from these operations and prove every one is attached or explicitly unsupported before dispatch. Include nested immediate parent exit, timeout, stream overflow, interruption, malformed custody and cleanup failure cases. Preserve original formal/runtimeclosure tests byte-for-byte; new tests have their own reviewed RED inventory.
6. Produce callable attachment integration consumed by MAC-hpqp; do not rewrite its seven frozen files or claim its pending 96-case/full-lane proof is already delivered.

### TEST COMMANDS
Run targeted native package tests for the owned implementation and the specifically inventoried required contributor lane fragments; record the exact selected leaf inventory and command in RED. No heavy preflight during this story.

### OUT OF SCOPE
The existing MAC-hpqp developer retains its healthy claim and frozen pilot; it owns contributor-lane call sites and the supplemental lane fragment using this delivery. Final-Ga Git closure is a separate consumer.

### DIFF BUDGET
~9 files, under 1700 changed LOC including supplemental tests.

## nd_contract
status: new

### evidence
- Created from approved standalone architecture, not an implementation claim.
- Root dispatch hold remains pending independent Anchor backlog review.

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


## Links
- Parent: [[MAC-ui8a]]

## Comments
