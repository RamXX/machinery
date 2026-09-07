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
updated_at: 2026-09-06T12:45:47Z
content_hash: "sha256:bfb6d97d635e245a9d14157e9876fb7b678cc26053d457416704dc6687edb452"
blocks: [MAC-pe9v, MAC-hpqp, MAC-vx24, MAC-ou97]
was_blocked_by: [MAC-qlw2]
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
- 2026-09-06T09:09:57Z dep_added: blocked_by MAC-qlw2
- 2026-09-06T09:10:04Z dep_added: blocks MAC-pe9v
- 2026-09-06T09:10:14Z dep_added: blocks MAC-hpqp
- 2026-09-06T09:10:15Z dep_added: blocks MAC-vx24
- 2026-09-06T09:10:16Z dep_added: blocks MAC-ou97
- 2026-09-07T00:29:10Z dep_removed: was_blocked_by MAC-qlw2

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-pe9v]], [[MAC-hpqp]], [[MAC-vx24]], [[MAC-ou97]]
- Was blocked by: [[MAC-qlw2]]

## Comments

### 2026-09-06T09:16:29Z ramirosalas
CANONICAL MACHINE-READABLE BOUNDARY MAP 2026-09-06
This repeats the existing ownership/signatures in the parser-supported form; it does not create additional scope or weaken any AC. All prior exact acceptance/testing requirements and holds remain current.

PRODUCES:
- internal/processcontrol/scope.go -> internal/processcontrol -> WithScope(ctx, scope), AttachScope(cmd, scope), ExitStatus(err), keeping Run(ctx context.Context, cmd *exec.Cmd) error compatible. Exact new API contracts in section 8 apply; do not invent alternate error semantics. Explicit formal/runtimeclosure call sites attach the verified scope after sanitized environments. cmd/machinery/main.go -> same-binary ServeInternal interception before ordinary Cobra parsing.
- internal/processcontrol/scope_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/formal/custody_integration_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/runtimeclosure/custody_integration_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/processcontrol/run.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/formal/formal.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/runtimeclosure/java.go -> owned bounded artifact; behavior and tests specified in the current story AC
- cmd/machinery/main.go -> owned bounded artifact; behavior and tests specified in the current story AC

CONSUMES:
- MAC-qlw2: internal/processscope/scope.go
  spec: Scope.Run/Child/Attach/Close and ServeInternal(args []string, io InternalIO) (handled bool, exitCode int); first authenticate inherited authority, then normal command parsing if unhandled.
Existing source: internal/processcontrol/run.go -> Run(ctx context.Context, cmd *exec.Cmd) error; internal/formal/formal.go -> VerifyFormalTo(design string, genOnly bool, stdoutW, stderrW io.Writer) (exitCode int); internal/runtimeclosure/java.go -> OpenJava() (*Java, error), Environment(home, temp, javaPath string) []string.
- MAC-l7m0: docs/test-assurance-contract.md
  schema: Exact approved public contract SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8; accepted delivery required, never an uncommitted external proposal.

Observable outcome: A user can cancel real formal verification without leaving its owned nested JVM running; malformed custody returns an explicit error.

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
- [ ] AC #6: current story acceptance requirement remains pending.

### 2026-09-06T09:47:59Z ramirosalas
ROUND-1 RULE 1 REPAIR: COMPLETE FORMAL CUSTODY OWNERSHIP
This is the complete CURRENT ownership/acceptance map for this bounded attachment story. It supersedes the earlier incomplete exclusive path list, preserving all prior guarantees, frozen evidence and the approved architecture. No source edit or implementation approval is made by this note.

PRODUCES:
- internal/processcontrol/scope.go -> approved WithScope/AttachScope/ExitStatus integration.
- internal/processcontrol/scope_test.go -> supplemental compatibility and fail-closed tests.
- internal/processcontrol/run.go -> preserve Run(ctx context.Context,cmd *exec.Cmd) error, attach explicit verified custody.
- internal/formal/formal.go -> actual VerifyFormalTo/runTLC caller chain carries the approved execution context/scope.
- internal/formal/process.go -> actual openFormalJava identity probe and runBoundedProcess use inherited execution context and verified scope after environment sanitation.
- internal/formal/alloy.go -> actual runAlloy probe and separate JVM launch use that SAME execution context/scope after sanitation.
- internal/formal/custody_integration_test.go -> separately reviewed real TLC/Alloy/probe/command-chain native supplemental tests.
- internal/runtimeclosure/java.go -> explicit custody attachment while preserving opened Java closure identity.
- internal/runtimeclosure/custody_integration_test.go -> supplemental runtime probe/cancellation evidence.
- cmd/machinery/main.go -> authenticated same-binary ServeInternal before command parsing.
- cmd/machinery/stubs.go -> ONLY newVerifyFormalCmd context/scope handoff to the existing formal command chain; preserve other commands.

CONSUMES:
- MAC-qlw2: internal/processscope/scope.go
  spec: Open(context.Context,Options)(Scope,error); Scope.Run/Child/Attach/Close; ServeInternal(args []string,io InternalIO)(handled bool,exitCode int), exact approved section8.
- MAC-l7m0: docs/test-assurance-contract.md
  schema: exact approved public contract; section4 cumulative budgets, section8 custody, section7 normal API/owner boundary.
- Existing exact epic source.
  source: openFormalJava(workdir string)(*runtimeclosure.Java,error); runBoundedProcess(ctx context.Context,cmd *exec.Cmd,timeout time.Duration)(string,error); runAlloy(alsPath string,commands []alloy.Command)(result []AlloyVerdict,notes []string,retErr error); VerifyFormalTo(design string,genOnly bool,stdoutW,stderrW io.Writer)(exitCode int); newVerifyFormalCmd() *cobra.Command.

CURRENT ACCEPTANCE CRITERIA
1. Preserve ordinary Run and existing callable formal/test APIs. Scoped execution uses explicit verified context/scope-bearing internal call paths (compatibility wrappers may retain old signatures), with the exact private signature/caller inventory reviewed before supplemental RED. Required scoped calls cannot select an unscoped wrapper or context.Background fallback. No new public product command, ambient-global scope substitute or copied parallel probe.
2. Wire actual command -> VerifyFormalTo -> both runTLC and runAlloy -> openFormalJava -> runBoundedProcess/processcontrol paths. The source-verified openFormalJava probe and separate Alloy JVM currently create background contexts; replace those scoped-path contexts with bounded children of the SAME owner. Apply WithScope and AttachScope AFTER each runtimeclosure.Environment sanitation. Do not omit Alloy or duplicate the probe to avoid its real source file.
3. Retain opened Java/JAR identity, checksum/revalidation, strict output/receipt checks, existing runtime limits and cleanup/publication guarantees. Audit affected *exec.ExitError inspection and preserve classification with the approved accessor; malformed ExtraFiles/SysProcAttr or custody mismatch fails closed, no fallback.
4. Authenticate ServeInternal before ordinary parsing; untrusted/malformed internal activation cannot become a normal launch. Preserve ordinary Cobra activation/recovery and all unrelated stubs.go commands.
5. On Linux amd64 AND Darwin arm64, real native tests observe active pinned nested JVMs before cancellation in actual provisioning/probe and suite/meta verification paths, exercising BOTH TLC and Alloy branches. Assert owned descendants terminate/reap and unrelated processes/containers survive. No fake Java, source-only proof or outer provisioning wrapper substitute.
6. Supplemental positive/negative tests cover normal output, intended assertion failure, nested early-parent exit, cancellation, timeout, stream overflow, malformed/forged/closed scope, probe/engine cleanup failure, and lost attachment after environment rebuilding. All preparation/probes/engines consume inherited cumulative deadlines and one shared cleanup grace; no per-step reset.
7. Produce the exact source/test/fixture/config inventory BEFORE RED review; preserve existing formal/runtimeclosure frozen tests byte-for-byte. The new formal/runtimeclosure supplemental cases are explicitly consumed by MAC-hpqp's testdata/integration-lanes/custody.json alongside its original96+new lane cases, never silently left out of required CI/preflight execution.
8. Demonstrate actual normal command and library producer/consumer call paths, including the stubs.go handoff. Publish the actual process-producing call-graph inventory; every required launch is attached or rejected before dispatch. No acceptance on an unused new helper.
9. Existing MAC-hpqp claim/pilot bytes and its pending same-source proof remain intact. This delivery supplies attachment implementation only; hpqp supplies lane execution integration. Developer delivers, independent PM verifies native proof. No preflight until final epic, installed replacement, remote action or unrelated teardown.

DIFF BUDGET: complete current scope11 files, under2300 changed LOC including supplemental tests (earlier ~9/1700 forecast repaired to include the three missing real call surfaces); investigate overrun before broadening.
MANDATORY SKILLS: developer, codebase-memory for exact call-site inventory, pm_acceptor.
OUT OF SCOPE: hpqp frozen96pilot edits; new formal engines or architecture; final Git/Ga implementation owned by MAC-pe9v. All old stronger constraints remain required.

## nd_contract
status: new

### evidence
- Independent round1 findings repaired through supported append-only scope/consumer notes; no architecture/source/test/ref/runtime mutation.
- Exact source surfaces verified against accepted epic7e36f3e7ddcf25565d5d4fe60b328df254eee91d; prior Body/status/labels/history preserved.
- Independent Anchor ROUND2 and canonical-document acceptance remain prerequisites, not implementation evidence.

### proof
- [ ] All current story ACs, strengthened ownership/current-judgment requirements and protected frozen proof remain required.
- [ ] Independent review/native execution/final acceptance pending.

### 2026-09-06T12:45:47Z ramirosalas
ACCEPTED PUBLIC CUSTODY CONTRACT / AUTHORITATIVE CURRENT ATTACHMENT SCOPE — 2026-09-06

This true-EOF amendment consumes the accepted public custody refinement and supersedes only obsolete publication-hold references. It preserves the existing complete ELEVEN-path scope, all NINE current acceptance criteria, prior evidence/history, hard-TDD requirements and the dependency chain. It grants no source/test edit, RED, implementation, native proof, delivery or acceptance.

ACCEPTED PRODUCER GROUNDING
- MAC-p9wm is closed/accepted. Epic `2a73454d5f133a7b5fb4db0346232fd389810d28` contains `docs/native-custody-contract.md` blob `1c1581d1aec324d979593613b44976ed0007c45b` / SHA256 `bee289e553809b550b1f5cedf73415f600379ab54830a11733ba0dc7caaaba2a` and companion `docs/test-assurance-contract.md` blob `acce64fee8db5a7565e8fa33422334de125e913f` / SHA256 `171ebaa3e992b60bd90f1fc280818a125689390f8440b3a9a7b2c97e7f8b747d`.
- Independent acceptance report SHA256: `19c0c4be6b4eb4b7035db3e9e576352ad1cfd3b24f414e287be7d8dfded38f8e`. Approved V4 architecture/review SHA256: `d56d0104e965cace70e89b3be383ff3f51377c33ea8227c86833ef731be331be` / `dfa09b70d599a83d8cde10ab31d19fe5debd1eadf4c19f86c3aea79c0a9ac1e3`.

PRODUCES:
- internal/processcontrol/scope.go -> WithScope(ctx, scope), AttachScope(cmd, scope), ExitStatus(err), preserving Run(context.Context, *exec.Cmd) error compatibility and fail-closed scoped error semantics.
- internal/processcontrol/scope_test.go -> scoped compatibility, attachment, exit classification and malformed-input tests.
- internal/processcontrol/run.go -> existing compatible execution entry with explicit verified custody selection and no required-path unscoped fallback.
- internal/formal/formal.go -> actual VerifyFormalTo/runTLC chain carries inherited owner context/scope and preserves existing engine/JAR behavior.
- internal/formal/process.go -> openFormalJava identity probe and runBoundedProcess use the same bounded owner and attach only after runtime environment sanitation.
- internal/formal/alloy.go -> runAlloy probe and the separate real Alloy JVM use that same bounded owner/scope after sanitation.
- internal/formal/custody_integration_test.go -> separately reviewed real TLC, Alloy, probe and normal-command custody cases.
- internal/runtimeclosure/java.go -> explicit scoped attachment while preserving opened Java closure identity and validation.
- internal/runtimeclosure/custody_integration_test.go -> real runtime probe/cancellation/identity evidence.
- cmd/machinery/main.go -> error-first InheritedInternalIO acquisition and authenticated ServeInternal interception before ordinary Cobra parsing.
- cmd/machinery/stubs.go -> ONLY newVerifyFormalCmd real context/scope handoff; every unrelated command remains unchanged.

CONSUMES:
- MAC-qlw2: internal/processscope/scope.go
  spec: InheritedInternalIO(context.Context) (InternalIO, bool, error); (InternalIO).Close() error; Open(context.Context, Options) (Scope, error); Scope.Run/Child/Attach/Close; ServeInternal([]string, InternalIO) (bool, int), using the accepted acquisition-versus-work lifetime and error-before-bool rules.
- MAC-p9wm: docs/native-custody-contract.md
  schema: exact accepted constructors, validation precedence, opaque-handle ownership, attachment-after-sanitation, real owner propagation, cumulative deadline/one-cleanup-grace and fail-closed internal-service rules at SHA256 bee289e553809b550b1f5cedf73415f600379ab54830a11733ba0dc7caaaba2a.
- MAC-p9wm: docs/test-assurance-contract.md
  source: exact accepted formal/processcontrol/runtimeclosure and contributor attachment obligations, preserved by companion SHA256 171ebaa3e992b60bd90f1fc280818a125689390f8440b3a9a7b2c97e7f8b747d.
- Existing exact epic source.
  source: Run(context.Context,*exec.Cmd) error; VerifyFormalTo(string,bool,io.Writer,io.Writer) int; openFormalJava(string) (*runtimeclosure.Java,error); runBoundedProcess(context.Context,*exec.Cmd,time.Duration) (string,error); runAlloy(string,[]alloy.Command) ([]AlloyVerdict,[]string,error); newVerifyFormalCmd() *cobra.Command; OpenJava() (*Java,error); Environment(string,string,string) []string.

CURRENT NINE ACCEPTANCE CRITERIA — PRESERVED AND REFINED
1. Preserve ordinary Run and every existing callable formal/test API. Required scoped paths use explicit context/scope-bearing calls and cannot select a compatibility wrapper, `context.Background`, ambient attachment or root fallback.
2. Constructor use is error-first. Nil/no-deadline acquisition context, prior cancellation/expiry and present-invalid intent follow the accepted precedence; only successful absence proceeds to ordinary CLI parsing. A valid internal request owns its handle through ServeInternal; malformed/untrusted activation never reaches product execution.
3. Wire the actual command -> VerifyFormalTo -> runTLC AND runAlloy -> openFormalJava -> runBoundedProcess/processcontrol chains. Apply custody AFTER each runtimeclosure.Environment sanitation so reconstruction cannot strip or mutate it. The SAME real owner reaches probes and actual JVMs.
4. Retain Java/JAR opened identity, snapshot/checksum/revalidation, strict output/receipt checks, existing runtime limits, cleanup and publication guarantees. Audit dependent *exec.ExitError use and preserve target-versus-custody classification through ExitStatus.
5. Malformed ExtraFiles, conflicting SysProcAttr, stale/forged/closed/cross-scope attachment, rebuilt-environment loss or missing registration fails before Cmd.Start. No required scoped path silently falls back to unscoped execution.
6. On native Linux amd64 AND Darwin arm64, observe active pinned JVMs before cancellation in actual provisioning/probe and suite/meta verification, across BOTH TLC and Alloy. Owned descendants terminate/reap before return while unrelated process/container controls survive; fake Java, source-only inspection and a provisioning-only wrapper do not qualify.
7. Supplemental tests cover normal output, intended assertion failure, early parent exit, cancellation, timeout, stream overflow, attachment loss, malformed/forged/closed scope, probe/engine cleanup failure and one cumulative owner/wall budget plus shared cleanup grace. Preserve all existing formal/runtimeclosure frozen tests byte-for-byte.
8. Before RED, publish the exact source/test/helper/fixture/config/dependency-lock and process-producing call-site inventory. MAC-hpqp consumes every required formal/TLC/Alloy/probe/command-chain leaf in its current custody fragment; an unused helper or omitted branch cannot pass.
9. Deliver attachment implementation only. Do not own or edit `cmd/machinery/verify_checkers.go`; checker dispatch/profile belongs to MAC-yhg5. Do not rewrite MAC-hpqp’s historical frozen inputs or claim its current 96-obligation replay. Developer delivers; independent PM verifies actual native evidence.

DIFF BUDGET: exactly 11 existing owned paths, forecast under 2,300 changed LOC including supplemental tests. Overrun requires explicit investigation, not invented outputs or weakened AC. qlw2 remains this story’s producer and MAC-hpqp remains its consumer; `MAC-qlw2 -> MAC-cn7q -> MAC-hpqp` is unchanged. Accepted MAC-p9wm is satisfied context, not a new blocker. All standalone/no-install/no-remote/no-preflight/no-unrelated-cleanup constraints remain.

## nd_contract
status: new

### evidence
- Accepted producer hashes and complete current eleven-path/nine-AC map are fixed above.
- Status remains open/new and dependency direction remains qlw2 -> cn7q -> hpqp. No source/test/ref/worktree/runtime/native/preflight/remote/install action occurred.

### proof
- [x] Accepted constructor/acquisition/work-lifetime and real-owner/post-sanitation refinements are canonically consumed.
- [x] Eleven paths, nine ACs and the checker ownership exclusion are explicit.
- [ ] Exact before-RED inventory, implementation and both required native-host matrices remain pending.

