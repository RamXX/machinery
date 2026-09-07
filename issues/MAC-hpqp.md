---
id: MAC-hpqp
title: "Execute every required infrastructure test deterministically"
status: closed
priority: 0
type: bug
labels: [hard-tdd, red-approved, accepted]
parent: MAC-ui8a
created_at: 2026-09-05T19:44:27Z
created_by: ramirosalas
updated_at: 2026-09-07T04:45:25Z
content_hash: "sha256:5cff0afd45b577719d301e07190277a954de104aefb458dcc5709daa82adf43e"
assignee: dev-MAC-hpqp
follows: [MAC-olrx]
was_blocked_by: [MAC-cn7q]
closed_at: 2026-09-07T04:45:25Z
close_reason: "Accepted: every required infrastructure test executes deterministically under native custody; AC8 hold resolved; merged to local epic"
led_to: [MAC-2n83, MAC-yhg5, MAC-hlae, MAC-hwdb, MAC-hy71, MAC-gcrr]
---

## Description
### USER INTENT
The maintainer can run one mandatory integration lane locally and in CI and know every required real-runtime safety test actually executed with correct prerequisites and no leaked resources.

### Context (Embedded)
Anchor round 1 found a cross-story execution gap: new Docker/Java/Node cases were assigned ordinary package suites, but macOS native CI lacks Docker and preflight provisions checker image after its race suite. Required tests need explicit lane selection, not skip-if-missing. This is repository contributor infrastructure, standalone Go/Node/Java/Docker only; it never requires Paivot.


### Architecture-decision hold: transitive cancellation ownership
The claimed GREEN checkpoint 221525d7baf7a565ece9972d09bf916132e68050 is intentionally paused and undelivered pending the existing user containment choice and a concrete reviewed contributor-lane ownership contract. This is an unresolved AC8 guarantee established by source, not an experimentally observed nested-JVM leak. Current processcontrol.Run owns one Unix process group; nested provisioning helpers, formal tests and full-path meta-tests launch additional groups. Killing the outer group does not establish termination of all nested JVMs. A proposed ProvisionTLC wrapper would remove only one provisioning path, not actual suite nesting or the helper's separate Java identity-probe group, and is not approved as an AC8 solution.
Retain the architecture hold within this P0 story. Do not create an implementation prerequisite or broaden source/API ownership before the pending containment decision. Current contributor baseline remains required Linux CI plus local macOS with Docker Desktop and service-free ordinary portability. Pending product-assurance host/OCI and adapter decisions belong to MAC-l7m0; do not silently reinterpret them as permission to change this contributor contract. The subsequent architecture review must explicitly reconcile the chosen contributor execution mode, exact process-custody interfaces, platform guarantees, runtime/source identity preservation, frozen-test compatibility and ownership. No broad process-name matching, bare PID/PGID registration or unverified group kill establishes authority to terminate a foreign process.
Only affected new containment implementation/test work and AC8 completion are on hold. Existing checkpoint evidence, approved frozen bytes, author claim/status and unrelated stories remain intact. Full final-SHA lane/Node replay and independent acceptance are still pending. A tiny wrapper or existing leaf counts cannot close this hold. No new production/test edits or RED authorization arise from this triage.

### Ownership
Own scripts/integration-lane/main.go, scripts/integration-lane/main_test.go, testdata/integration-lanes/schema.json, testdata/integration-lanes/pilot.json, testdata/integration-lanes/runtime-pins.json, cmd/machinery/integration_lane_test.go, scripts/preflight.sh, .github/workflows/ci.yml, .github/workflows/formal.yml, .github/workflows/nightly.yml, Makefile, scripts/shellcheck-files.txt. Preserve concurrent edits. Runtime stories own separate lane fragments and their tests, never the common inventory engine. The release workflow story modifies shared workflows afterward. internal/processcontrol, internal/formal and internal/runtimeclosure remain read-only inputs during this hold; the proposed new formal provisioning wrapper and shared process ownership interfaces are not authorized outputs.

### Boundary Map
PRODUCES:
- scripts/integration-lane/main.go -> new contributor CLI: go run ./scripts/integration-lane --lane required; closed inventory validation/provisioning/native execution/accounting/cleanup
- scripts/integration-lane/main_test.go -> native unit and actual process runner validation
- testdata/integration-lanes/schema.json -> closed versioned per-suite fragment contract: exact source/test identities, native adapter, runtime requirements, command selection, timeout/output/resource bounds
- testdata/integration-lanes/pilot.json -> nonempty initial suite covering actual provisioned runtime path
- testdata/integration-lanes/runtime-pins.json -> explicit immutable OCI platform/digest and pinned engine/runtime identities
- cmd/machinery/integration_lane_test.go -> build-tagged real runtime pilot tests, excluded only from named ordinary native lane
- scripts/preflight.sh -> mandatory integration lane after prerequisite provisioning, before any phase consuming service-backed evidence
- .github/workflows/ci.yml -> required Linux runtime-lane job plus explicit service-free native portability suites
- .github/workflows/formal.yml -> formal lane/engine provisioning consistent with shared inventory
- .github/workflows/nightly.yml -> declared service-free versus provisioned required-lane execution
- Makefile -> test-integration entrypoint invokes exact same required lane
- scripts/shellcheck-files.txt -> registers any introduced executable shell surface, or verified unchanged closed inventory if implementation remains Go-only
CONSUMES:
- Existing scripts/run-safe/main.go.
  spec: run(args []string, stdout, stderr io.Writer) int; CLI -timeout, -stdout-limit, -stderr-limit, -expect-stdout-file, -expect-stderr-file followed by -- and command argv.
- Existing scripts/example-inventory.sh.
  source: rows/formal/checkers modes validate closed example manifest and emit selected subjects.
- Existing scripts/preflight.sh.
  source: checker image python@sha256:c6ead215bfd31f1e433d968853b7a769989117115b728874824e6c0a27cb96fc on linux/amd64; bounded docker pull, RepoDigests/platform inspect and offline execution establish closure.

- Existing internal/processcontrol/run.go and internal/processcontrol/run_unix.go.
  spec: Run(ctx context.Context, cmd *exec.Cmd) error; prepare(cmd *exec.Cmd) (*treeControl, error).
  source: Unix prepare sets Setpgid:true for each launch; treeControl stores one pgid, and terminate/close signal that group. Run coordinates bounded wait/reap and joins cleanup errors. It has no existing ancestor-owned cross-process scope parameter.
- Existing internal/formal/formal.go and internal/formal/process.go.
  spec: VerifyFormalTo(design string, genOnly bool, stdoutW, stderrW io.Writer) (exitCode int); ensureJar() (string, error); runTLC(tlaPath, cfgPath string) (output string, retErr error); runBoundedProcess(ctx context.Context, cmd *exec.Cmd, timeout time.Duration) (string, error).
  source: ensureJar retains rooted/checksummed cache lock, staging/recovery and durable publication. runTLC snapshots the verified JAR into private metadata scratch, opens/revalidates Java closure, and invokes a separately grouped JVM with its own deadline.
- Existing internal/runtimeclosure/java.go.
  spec: OpenJava() (*Java, error); Environment(home, temp, javaPath string) []string.
  source: Java custody binds actual launcher and runtime closure, checks identity before/after execution, and Environment creates a closed sanitized environment. A future custody transport must deliberately preserve those guarantees; ambient scope variables do not automatically survive.

### Story Acceptance Criteria
1. A required named lane discovers and validates every committed suite fragment using a closed schema and matches exact selected native test identities to its registered source files. Unknown/duplicate/orphan fragments, unregistered integration test files, invalid paths and empty lane fail. Every fragment names actual runnable cases, never future placeholders.
2. Ordinary go test ./... native lanes deliberately exclude build-tagged runtime tests and do not probe/skip absent services. Required lane explicitly selects them and fails if Docker/Node/Java prerequisites are unavailable. Runtime tests cannot become dormant via env flags, t.Skip, skipped cases, xfail, filtering or empty package selection.
3. Provision and verify immutable OCI digest/platform, pinned formal/Java closure and supported Node runtime BEFORE their tests. A cold-cache positive provisions then executes; absent/offline/unavailable runtime and wrong digest/platform reject with clear diagnostics. Existing pinned example image may be reused; never trust a mutable tag.
4. Capture real Go JSON and Node native structured execution output under bounded processes and verify every registered test started, terminated and passed exactly as required. Missing/incomplete/duplicate/skipped tests, cached or malformed/truncated events, extra unexpected failures and fabricated aggregate-only summaries block. Success records actual nonzero execution counts per suite and runtime.
5. Required Linux hosted CI and final local preflight invoke the same lane/inventory. macOS native portability remains service-free while local macOS preflight with Docker Desktop executes the same required safety coverage using pinned platform. Job status must be available for release required-check policy; no remote mutation occurs during implementation.
6. Eliminate preflight ordering hazard: service-backed tests are never executed in the early ordinary race suite before image/runtime provisioning. Preserve existing formal/C4/checker gates, full-history needs and cheapest-first safe ordering; no weakening via SKIP_PREFLIGHT or optional missing-runtime fallback.
7. Initial story is independently executable: include a nonempty actual bounded OCI success/termination pilot plus actual formal/Node invocation where classified as required. Later stories add their own fragment only alongside executable tests. Shared runner discovers their union deterministically without each touching the root manifest.
8. Teardown validates all owned containers/processes/temp roots are removed after success, failed assertion, timeout, cancellation and provisioning error, or fails with actionable cleanup error. Never remove unrelated containers or user files; preserve existing dagger-engine-v0.21.9. This includes real nested JVMs and helpers during provisioning (Java identity probe and TLC) and actual formal-suite/full-path meta-test execution, even when descendants establish separate process groups. Observe the intended active nested process before cancellation, and verify its termination before lane return; bounded outer return or closed output pipes alone are not cleanup evidence. Preserve exact caller-root identity/sentinels and all unrelated live processes/containers. Missing/forged/stale custody, failed ownership establishment, intermediate-parent exit and PID/PGID reuse cannot authorize foreign termination or a false cleanup-success report. Existing formal Java/JAR pin, private snapshot, source binding, sanitized-environment and post-execution identity checks must remain intact. The exact custody mechanism, failure model and contributor-platform contract await the architecture-decision hold above; no wrapper-only fix or unreviewed host/OCI choice is authorized. Normal completion, assertion failure, operation deadline, output-limit cancellation, SIGINT/SIGTERM cancellation and provisioning error remain required; architecture must separately state residual behavior for uncatchable owner death/host failure/hostile descendants without silently weakening those required outcomes.
9. RED tests demonstrate existing missing-lane/incorrect-order or wrong-accounting behavior through genuine runtime assertions with passing controls; freeze tests/config/fixtures after approval. No compile/infra-only RED. Targeted lane/provisioning/accounting tests and actionlint pass; do NOT run full scripts/preflight.sh before final epic completion.
10. Real full-path no-mock tests cover provisioned success, fresh cache, missing runtime, zero execution, skipped required test, wrong pin, partial output and leaked owned resource. Run native suites without services to prove separation and required lane with services to prove coverage. Machine-readable report identifies exactly what ran; it cannot stand in for native events.

### Testing Requirements
Hard TDD explicitly authorized. Unit plus Integration tests: MANDATORY (no mocks), actual bounded processes and real Docker daemon; no skip-if-missing. Unit parser fixtures may test malformed events but cannot replace real execution proof. Targeted go test ./scripts/integration-lane and go run ./scripts/integration-lane --lane required; actionlint and native ordinary-lane selection checks. Full heavy preflight ONLY at end.
No GitHub push/sync/mutation, no installed binary/plugin/skill/agent replacement. Isolated builds/homes only. No Paivot product dependency.

Required supplemental AC8 verification after the architecture decision and independent concrete test authorization: actual pinned Java/TLC through the real contributor lane, with independently observed active nested JVMs in BOTH provisioning and actual formal-suite/meta-test paths. Cover outer SIGINT/SIGTERM cancellation and bounded timeout, plus normal finite safe/unsafe TLC completion; add failed-assertion/output-overflow and early intermediate-parent-exit cases for the chosen custody lifecycle. Require intended errors, bounded return, exact owned JVM/helper absence before return and honest cleanup evidence, with live unrelated process/container and caller-directory/sentinel controls. Challenge absent/forged/stale ownership and failed custody establishment without using broad process matching or bare numeric identity as cleanup authority. Tests need independently verified process identity and bounded task-owned emergency cleanup established before execution, no fake Java/mock runner or never-started-child false positive. Exact new files/fragments/interfaces and any frozen edits require independent review; no new test-edit or RED authorization is granted here.
Platform evidence must match the contributor execution decision. The current baseline requires Linux amd64 native CI and Darwin arm64 local Docker Desktop execution; service-free ordinary native suites remain separate. Existing Linux arm64 and Windows amd64 cross-builds prove compilation only; Windows is not a supported runtime and neither cross-build supplies nested-process cleanup evidence. If approved OCI execution replaces any native path, explicitly review immutable platform/source/runtime/cache/report contracts and actual host/supervisor cleanup proof rather than claiming Linux-only kernel behavior proves native macOS custody. No hosted run is authorized during this work; its eventual required job remains part of acceptance/release evidence.
Preserve the frozen 96-case union and every selected native event/accounting assertion. Current checkpoint proof is 64 native Go leaves passed; 31 tagged runtime leaves were 30 pass/1 cold-cache path-alias failure, with the corrected cold-cache leaf passing separately; standalone Node and final full current-SHA lane replay remain pending. These counts do not exercise the newly identified outer-cancellation/nested-JVM boundary and cannot close AC8.

### OUT OF SCOPE
- Individual saga/OCI/recovery/OpenCode/process/capstone behavior: sibling stories own their real tests and inventory fragments, consuming this lane.
- Final full-preflight execution belongs to epic completion after all siblings.

### DIFF BUDGET
Measured current checkpoint: 13 files, 3004 insertions/11 deletions (3015 changed LOC), including 1753 approved RED lines and a 1213-line runner. The original under1800 combined estimate is disproven: it left only47 implementation lines after RED. Preserve approved tests and investigate actual ownership/volume. No final expanded estimate is authorized before the containment architecture and concrete supplemental proof are selected; the next scope review must budget those additions explicitly against this measured baseline, not call an eight-line provisioning wrapper the complete solution.

### MANDATORY SKILLS
developer; codebase-memory; pm_acceptor.

### nd_contract
status: new

#### evidence
Created 2026-09-05 to repair Anchor round-1 general execution-lane gap; source interfaces and provisioning sequence read directly.

#### proof
- [ ] AC #1-10: independently verified with real execution.


## Acceptance Criteria


## Design


## Notes
GREEN scope investigation: approved RED v2 already contributes 1753 lines. Closed inventory and exact native identity validation, Go JSON/Node TAP accounting, pinned OCI/Java/TLC/Node provisioning, owned scratch/container/process cleanup, and required workflow/preflight/Make wiring require an estimated 800-1200 production lines plus approximately 60 wiring lines. Dispatcher acknowledged investigation; independent PM must assess final exact delta. Frozen seven test/config/fixture files remain immutable. Implementation stays in declared production ownership and reuses processcontrol/runtimeclosure/formal primitives.
## AC8 triage readback verification — 2026-09-05
Canonical readback verifies AC1-7/9-10 verbatim; explicit architecture hold, expanded AC8 verification, measured budget and both GREEN pause comments retained. pvg lint --backlog --epic MAC-ui8a --json returned []; pvg rtm check --epic MAC-ui8a passed with0 tagged requirements/18stories1closed (limited traceability assurance); pvg nd dep cycles found no cycles. Story remains P0 in_progress hard-tdd,red-approved assigned dev-MAC-hpqp under MAC-ui8a with identical downstream dependencies. Root clean main...origin/main. Self-review verdict: clean; existing P0 outcome owner retained, no guessed custody mechanism/API or new prerequisite, exact platform/runtime residuals explicit. This is backlog/architecture-hold evidence, not empirical nested-JVM leakage or repaired cleanup evidence.

## nd_contract
status: in_progress

### evidence
- Supported canonical nd edit plus append-only history/triage completed; source and readback evidence above.
- Scoped lint [], RTM passed0tagged requirements, no dependency cycles; claim/status/labels and worktrees unchanged.

### proof
- [x] AC8 unresolved transitive custody risk and minimum real verification recorded under existing owner.
- [x] No provisioning-wrapper-only completion claim or unapproved host/OCI choice.
- [ ] Pending containment decision, exact reviewed architecture/ownership and independent concrete supplemental-test authorization.
- [ ] Complete final-SHA lane/Node replay and independent acceptance pending.

## AUTHORITATIVE AC8 NESTED-PROCESS SCOPE TRIAGE — MAC-hpqp — 2026-09-05

Disposition: retain an explicit architecture-decision hold within existing P0 MAC-hpqp, per dispatcher direction. No prerequisite was created, no implementation ownership broadened and no claim/status/label/dependency changed. Parent MAC-ui8a and existing downstream chain remain intact. MAC-2u36 installer triage is complete and untouched by this task.

Source-established risk, not observed leak: retained221525d runner command uses processcontrol.Run for provision-formal helper and go test. Helper provisionFormal launches a Java identity probe through command and formal.VerifyFormalTo; formal.runTLC invokes runBoundedProcess/processcontrol. Frozen TestIntegrationLanePilotFormal invokes the same verifier; TestIntegrationLaneFullPath launches another lane and nested native commands. Unix prepare unconditionally creates a new group; close/terminate signal only that recorded group. Independent outer cancellation therefore lacks a transitive group custody guarantee. No actual outer-cancellation/nested-JVM leak experiment was run by this triage or reported as completed by GREEN.

Existing interfaces: Run(context.Context,*exec.Cmd) error and captured variants offer no ancestor-scope parameter. Graph trace shows29 callers within two hops across formal/Alloy, C4, Git/checkers/hooks and run-safe; changing shared behavior requires compatibility review. Runtimeclosure.Environment creates a closed sanitized environment, so a future scope transport needs deliberate design. Existing Unix detached-session test checks bounded error return, not disappearance of that detached child; it does not supply missing AC8 liveness proof.

Preserve formal custody: ensureJar uses override checksum, jar path and rooted fetch/cache lock/stage recovery/durable publication. runTLC then takes a private verified JAR snapshot, opens Java identity/closure, launches the bounded actual JVM and validates Java after execution. A ProvisionTLC()->ensureJar wrapper is only provisioning output; it neither owns actual suite descendants nor proves execution identity. Direct runner probes or a new supervisor must retain relevant snapshot/closure/source/report guarantees. No such wrapper/API/design is approved by this note.

Required architecture decision: reconcile the pending containment choice with the contributor lane's current Linux CI + local macOS Docker Desktop native execution contract. Select exact custody and failure semantics, interfaces/transport, launch ordering, cancellation and cleanup acknowledgment, foreign-process/PID-reuse protection, platform proof and frozen-test compatibility. Product-wide strict assurance/adapters remain MAC-l7m0 decisions; do not silently repurpose their outcome into a contributor contract change. No Linux-only kernel or OCI policy is selected here. If new shared source ownership is required, repair it only after an approved interface/ownership contract; do not guess a broad supervisor implementation now.

Verification now explicitly covers actual nested Java identity/TLC provisioning and actual formal-suite/full-path meta-test execution, with witnessed active JVM before external cancel/timeout, normal safe/unsafe completion, bounded cleanup/liveness and unrelated process/container/caller-root preservation. Include assertion/output failure, intermediate-parent exit, absent/forged/stale custody and failed ownership establishment for the selected lifecycle. Plain process matching or numeric PID files are insufficient cleanup authority; tests need independent identity and bounded emergency cleanup for only their own processes. Current native baseline is Linux amd64 and Darwin arm64; Linux arm64/Windows cross-builds are not runtime cancellation proof. Exact supplemental tests and any frozen changes require independent PM authorization; no RED authorization issued by this triage.

Budget/evidence: canonical budget now records measured13files3004insertions/11deletions rather than pretending the original1800 covers the existing1753 RED +1213-line runner. A final new ceiling awaits the containment decision and concrete proof cost; never trim frozen safety tests. Current64 native Go leaves pass; prior tagged31 leaves30pass/1 cold-cache alias fail plus corrected cold-cache separate pass do not replace final full current-SHA replay. Frozen96 union includes the still-pending standalone Node pilot. Existing lint/actionlint/shellcheck/TDD checkpoint claims remain history, not full preflight or nested cancellation proof. No owned container was reported remaining at pause; Dagger remains protected.

## nd_contract
status: in_progress

### evidence
- Read current ten-AC canonical issue and final GREEN pause/correction comments through pvg issues show; read exact retained221525d runner, formal pilot/full-path test and frozen CONTRACT.md.
- Graph Verify project Users-ramirosalas-workspace-machinery, generation2026-09-05T20:28:41Z: processcontrol run/unix/windows/capture and formal.go metadata_match/no_recorded_issue, best-effort only. Candidate runner/test missing from graph on root; read their committed source directly. Read actual Run/Unix group control, Windows job code, captured interfaces, formal process/JAR/Java flow and closed runtime environment.
- Canonical supported nd edit clarifies AC8/hold/verification/measured budget while guarding AC1-7/9-10 verbatim. Old canonical preserved below as quoted history; all prior comments, frozen checkpoint and claim retained.
- No runtime tests, empirical leak probe, source/test/installed edits, branch/worktree changes, new issue/dependency, push/sync/GitHub change or scripts/preflight.sh execution.

### proof
- [x] Source-established nested-group custody gap and wrapper-only insufficiency recorded accurately under existing P0 owner.
- [x] Required real nested-JVM verification, platform evidence, foreign-state safety and formal identity preservation are explicit.
- [ ] Containment decision and exact independently reviewed architecture/interface/ownership contract pending.
- [ ] Concrete supplemental hard-TDD test authorization, complete final-SHA lane/Node evidence and independent AC8/AC10 acceptance pending.


## Historical canonical description before nested-process scope hold
Preserved as quoted history; current Description is authoritative.

> ## Description
> ### USER INTENT
> The maintainer can run one mandatory integration lane locally and in CI and know every required real-runtime safety test actually executed with correct prerequisites and no leaked resources.
> 
> ### Context (Embedded)
> Anchor round 1 found a cross-story execution gap: new Docker/Java/Node cases were assigned ordinary package suites, but macOS native CI lacks Docker and preflight provisions checker image after its race suite. Required tests need explicit lane selection, not skip-if-missing. This is repository contributor infrastructure, standalone Go/Node/Java/Docker only; it never requires Paivot.
> 
> ### Ownership
> Own scripts/integration-lane/main.go, scripts/integration-lane/main_test.go, testdata/integration-lanes/schema.json, testdata/integration-lanes/pilot.json, testdata/integration-lanes/runtime-pins.json, cmd/machinery/integration_lane_test.go, scripts/preflight.sh, .github/workflows/ci.yml, .github/workflows/formal.yml, .github/workflows/nightly.yml, Makefile, scripts/shellcheck-files.txt. Preserve concurrent edits. Runtime stories own separate lane fragments and their tests, never the common inventory engine. The release workflow story modifies shared workflows afterward.
> 
> ### Boundary Map
> PRODUCES:
> - scripts/integration-lane/main.go -> new contributor CLI: go run ./scripts/integration-lane --lane required; closed inventory validation/provisioning/native execution/accounting/cleanup
> - scripts/integration-lane/main_test.go -> native unit and actual process runner validation
> - testdata/integration-lanes/schema.json -> closed versioned per-suite fragment contract: exact source/test identities, native adapter, runtime requirements, command selection, timeout/output/resource bounds
> - testdata/integration-lanes/pilot.json -> nonempty initial suite covering actual provisioned runtime path
> - testdata/integration-lanes/runtime-pins.json -> explicit immutable OCI platform/digest and pinned engine/runtime identities
> - cmd/machinery/integration_lane_test.go -> build-tagged real runtime pilot tests, excluded only from named ordinary native lane
> - scripts/preflight.sh -> mandatory integration lane after prerequisite provisioning, before any phase consuming service-backed evidence
> - .github/workflows/ci.yml -> required Linux runtime-lane job plus explicit service-free native portability suites
> - .github/workflows/formal.yml -> formal lane/engine provisioning consistent with shared inventory
> - .github/workflows/nightly.yml -> declared service-free versus provisioned required-lane execution
> - Makefile -> test-integration entrypoint invokes exact same required lane
> - scripts/shellcheck-files.txt -> registers any introduced executable shell surface, or verified unchanged closed inventory if implementation remains Go-only
> CONSUMES:
> - Existing scripts/run-safe/main.go.
>   spec: run(args []string, stdout, stderr io.Writer) int; CLI -timeout, -stdout-limit, -stderr-limit, -expect-stdout-file, -expect-stderr-file followed by -- and command argv.
> - Existing scripts/example-inventory.sh.
>   source: rows/formal/checkers modes validate closed example manifest and emit selected subjects.
> - Existing scripts/preflight.sh.
>   source: checker image python@sha256:c6ead215bfd31f1e433d968853b7a769989117115b728874824e6c0a27cb96fc on linux/amd64; bounded docker pull, RepoDigests/platform inspect and offline execution establish closure.
> 
> ### Story Acceptance Criteria
> 1. A required named lane discovers and validates every committed suite fragment using a closed schema and matches exact selected native test identities to its registered source files. Unknown/duplicate/orphan fragments, unregistered integration test files, invalid paths and empty lane fail. Every fragment names actual runnable cases, never future placeholders.
> 2. Ordinary go test ./... native lanes deliberately exclude build-tagged runtime tests and do not probe/skip absent services. Required lane explicitly selects them and fails if Docker/Node/Java prerequisites are unavailable. Runtime tests cannot become dormant via env flags, t.Skip, skipped cases, xfail, filtering or empty package selection.
> 3. Provision and verify immutable OCI digest/platform, pinned formal/Java closure and supported Node runtime BEFORE their tests. A cold-cache positive provisions then executes; absent/offline/unavailable runtime and wrong digest/platform reject with clear diagnostics. Existing pinned example image may be reused; never trust a mutable tag.
> 4. Capture real Go JSON and Node native structured execution output under bounded processes and verify every registered test started, terminated and passed exactly as required. Missing/incomplete/duplicate/skipped tests, cached or malformed/truncated events, extra unexpected failures and fabricated aggregate-only summaries block. Success records actual nonzero execution counts per suite and runtime.
> 5. Required Linux hosted CI and final local preflight invoke the same lane/inventory. macOS native portability remains service-free while local macOS preflight with Docker Desktop executes the same required safety coverage using pinned platform. Job status must be available for release required-check policy; no remote mutation occurs during implementation.
> 6. Eliminate preflight ordering hazard: service-backed tests are never executed in the early ordinary race suite before image/runtime provisioning. Preserve existing formal/C4/checker gates, full-history needs and cheapest-first safe ordering; no weakening via SKIP_PREFLIGHT or optional missing-runtime fallback.
> 7. Initial story is independently executable: include a nonempty actual bounded OCI success/termination pilot plus actual formal/Node invocation where classified as required. Later stories add their own fragment only alongside executable tests. Shared runner discovers their union deterministically without each touching the root manifest.
> 8. Teardown validates all owned containers/processes/temp roots are removed after success, failed assertion, timeout, cancellation and provisioning error, or fails with actionable cleanup error. Never remove unrelated containers or user files; preserve existing dagger-engine-v0.21.9.
> 9. RED tests demonstrate existing missing-lane/incorrect-order or wrong-accounting behavior through genuine runtime assertions with passing controls; freeze tests/config/fixtures after approval. No compile/infra-only RED. Targeted lane/provisioning/accounting tests and actionlint pass; do NOT run full scripts/preflight.sh before final epic completion.
> 10. Real full-path no-mock tests cover provisioned success, fresh cache, missing runtime, zero execution, skipped required test, wrong pin, partial output and leaked owned resource. Run native suites without services to prove separation and required lane with services to prove coverage. Machine-readable report identifies exactly what ran; it cannot stand in for native events.
> 
> ### Testing Requirements
> Hard TDD explicitly authorized. Unit plus Integration tests: MANDATORY (no mocks), actual bounded processes and real Docker daemon; no skip-if-missing. Unit parser fixtures may test malformed events but cannot replace real execution proof. Targeted go test ./scripts/integration-lane and go run ./scripts/integration-lane --lane required; actionlint and native ordinary-lane selection checks. Full heavy preflight ONLY at end.
> No GitHub push/sync/mutation, no installed binary/plugin/skill/agent replacement. Isolated builds/homes only. No Paivot product dependency.
> 
> ### OUT OF SCOPE
> - Individual saga/OCI/recovery/OpenCode/process/capstone behavior: sibling stories own their real tests and inventory fragments, consuming this lane.
> - Final full-preflight execution belongs to epic completion after all siblings.
> 
> ### DIFF BUDGET
> ~10-13 files, under 1800 changed LOC; investigate overrun rather than weaken execution guarantees.
> 
> ### MANDATORY SKILLS
> developer; codebase-memory; pm_acceptor.
> 
> ### nd_contract
> status: new
> 
> #### evidence
> Created 2026-09-05 to repair Anchor round-1 general execution-lane gap; source interfaces and provisioning sequence read directly.
> 
> #### proof
> - [ ] AC #1-10: independently verified with real execution.
> 


## nd_contract
status: red-approved

### evidence
- RED tests approved via pvg story approve-red on 2026-09-05.

### proof
- [ ] GREEN developer must implement against the approved RED tests without modifying them.


## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## Implementation Evidence

PROOF:
RED candidate v2, authored under the PM test-edit authorization. Candidate v1 at 612f65f3b4502a3267828507faaf0e395c8dd558 remains preserved and was NEVER approved. New SHA: e55238223961fec922896c361d8af6cafc454a8e on story/MAC-hpqp, subject includes tdd-red and [test-edit-authorized]. No operational GREEN code was written.

### CI/Test Results

Commands run:
Working directory for every command: /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-hpqp
- go test -count=1 -timeout=2m -coverprofile=/tmp/machinery-hpqp-red-v2.DQvyDf/native.cover -json ./scripts/integration-lane
- go test -tags machinery_integration -count=1 -timeout=15m -json ./cmd/machinery -run '^TestIntegrationLane(Pilot(OCI|Formal)|FullPath|HostDescendantControl|NodeEventControls)$'
- node --test --test-reporter=tap testdata/integration-lanes/pilot.integration.test.mjs
- go test -count=1 -timeout=90s -json ./scripts/run-safe ./internal/processcontrol
- go test -count=1 -timeout=2m ./scripts/integration-lane -run '^TestLane(WiringGuardsRejectDisabledExecution|PreflightFailurePropagationControl)$'
- go test -tags machinery_integration -count=1 -timeout=2m ./cmd/machinery -run '^TestIntegrationLane(HostDescendantControl|NodeEventControls)$' -v
- go test -tags machinery_integration -count=1 -timeout=10m ./cmd/machinery -run '^TestIntegrationLanePilotFormal$' -v
- pvg verify scripts/integration-lane/main.go scripts/integration-lane/main_test.go cmd/machinery/integration_lane_test.go testdata/integration-lanes/pilot.integration.test.mjs --include-tests --format text
- actionlint .github/workflows/ci.yml .github/workflows/formal.yml .github/workflows/nightly.yml
- go tool cover -func=/tmp/machinery-hpqp-red-v2.DQvyDf/native.cover
- git diff --check
- git diff 612f65f3b4502a3267828507faaf0e395c8dd558 HEAD -- scripts/integration-lane/main.go
- docker ps -a --filter label=dev.machinery.integration-run --format '{{.ID}} {{.Names}} {{.Status}}'
- docker inspect --format '{{.Id}} {{.State.Running}}' dagger-engine-v0.21.9
- ps -eo pid,ppid,stat,command | awk '/\/bin\/sh .*\/(owned-descendant|real-owned-child)\.sh/ && !/awk/ {print}'

Summary:
- Native: 12 roots, 70 starts/70 terminals, 34 PASS/36 intended FAIL/0 SKIP including aggregate parents. Leaf-only: 64 = 32 PASS/32 intended FAIL.
- Runtime Go: all five registered roots, 34 starts/34 terminals, 8 PASS/26 intended FAIL/0 SKIP including aggregate parents. Leaf-only: 31 = 6 PASS/25 intended FAIL.
- Node pilot: one exact registered native leaf PASS; zero failures/skips/cancelled/todo.
- New RED total: 96 leaves = 39 passing controls/57 intended failing requirements/0 skipped. Existing bounded-runner controls add 20 passing leaves (18 roots,22 events,all PASS,zero skips).
- All 25 full-path scenario identities and all five Go pilot roots ran. All native started IDs have one terminal; duplicate started/terminal inventories are empty. No selected case is missing. Raw JSON/TAP retained under /tmp/machinery-hpqp-red-v2.DQvyDf/.
- Candidate lane assertions remain intentionally red at the replaceable fail-closed bootstrap; no compile/import/absent-infrastructure failure is counted as RED. Pipelines preserved exit status with set -o pipefail.
- Native behavior coverage: 66.7% bootstrap statements, run100%/main0%; NOT a claim of implemented lane coverage. Outcome coverage is mapped below.
- pvg verify PASS (3 source files scanned,0 issues); actionlint PASS; git diff --check PASS; clean worktree. main.go is byte-unchanged from v1.
- Candidate total:8 files,1753 lines,under1800. V2 revision:4 files,+761/-74. Added volume is exact runtime/ownership assertion code plus bypass sensitivity controls; no budget weakening.

### Four rejection gaps closed in the RED contract

1. Candidate CLI formal positive and genuinely fresh private closure:
TestIntegrationLaneFullPath/cold-cache now selects Go+Java+TLC, not Docker. It checks that its private cache is absent/empty before invocation, then requires candidate-provisioned launcher and jar paths. The exact native Go fixture runs safe and invariant-breaking TLC models through those paths, captures their hashes, and compares them with retained runtime records inside the private cache. Offline-formal/wrong-Java-pin/wrong-TLC-pin use the SAME native fixture. A runner rejecting every formal closure fails the cold-cache positive.
Sensitivity/control proof: TestIntegrationLanePilotFormal retains the original real pinned safe/unsafe test and additionally executes the exact generated Go fixture with the actual provisioned Java and TLC paths. This passed. It prevents fixture/parser errors masquerading as RED but does NOT replace the separate failing candidate-CLI assertion.

2. Exact native events and runtime union:
Every full-path success now checks suite ID/adapter/source/test, selected/started/passed cardinality, zero failures/skips, retained event bytes and sha256. Go checks exact package/test run/pass identities. Node requires exact Subtest/ok identity and terminal cardinality plus native plan/counts. Runtime records must equal the selected union, with exact Go/Node version and executable hashes, immutable OCI digest/platform, pinned Java probe/archive/closure/launcher identities, and exact TLC jar version/hash. Arbitrary nonempty records are no longer acceptable.
Sensitivity/control proof: real Node duplicate test names produced two native starts/terminals; an actually hanging Node test produced an incomplete native stream and was bounded/terminated. Both controls passed. Candidate node-duplicate and node-missing-terminal cases now require rejection; the timeout case additionally requires the body-entered marker. Existing Go duplicate/aggregate/partial-output controls remain.

3. Descendant termination and caller-owned directory preservation:
All25 scenarios now start with an existing --work-dir and user sentinel. Unconditional cleanup assertions check native directory identity, exact sentinel bytes, and removal of only runner-owned additions, including prerequisite failures. New process-success/failure/timeout/cancellation cases execute real Go tests that spawn private shell descendants; the harness records exact PID+private script path, independently checks post-return liveness, and has exact-ownership emergency cleanup.
Sensitivity/control proof: the real descendant control first verifies PID/liveness and termination, then executes the exact generated Go fixture. The Go native parent passed/exited while its real child remained alive, proving why killing only the lane/test parent is inadequate; the control then cleaned its exact child. Both that and existing Docker lifecycle controls pass.

4. Mandatory workflow/preflight enforcement:
Parsed workflow assertions now require exactly one exact command, unconditional job AND step, Linux runner, no ignored errors/expressions, and required full history. Shell token assertions distinguish executable commands from comments/quoted diagnostics, require top-level exact bare lane execution under strict error propagation, reject bypass variables/success exits/opaque transfers/shadowed executables/disabled errexit, and preserve native-before-lane-before-formal/C4/checker ordering. Make dry-run output must be the exact command.
Sensitivity/control proof:25 disabled wiring fixtures are rejected, including job.if, job/step continue-on-error, step.if, echoed/commented/inexact commands, renamed bypass diagnostic, ignored failures, conditional/function/subshell-only calls, wrong order and removed/early consumers. Four actual bounded Bash controls demonstrate strict failure versus ignored/conditional/renamed-bypass execution. These are small real shell programs, NOT full preflight execution or replacement runtime binaries.

### AC Verification

| AC | V2 frozen evidence |
|---|---|
|1|Existing closed schema/15 inventory mutations/source identity/union controls retained.|
|2|Service-free actual native selection retained; mandatory runtime failures plus job/step/bypass sensitivity now explicit.|
|3|Candidate cold formal closure positive and same-fixture offline/wrong-pin negatives; exact archive/closure/jar identities.|
|4|Exact retained/hash-bound Go AND Node streams, exact runtime union/identity, real Node duplicate/incomplete controls.|
|5|Exact mandatory Linux job+step and Make entrypoint; native macOS remains service-free.|
|6|Parsed top-level strict ordering, executable bypass rejection, formal/C4/checker consumer preservation, real Bash sensitivity.|
|7|Nonempty fragment registers all5 executable Go roots and real Node pilot; candidate formal positive is independently required.|
|8|Real Docker and host descendants across lifecycle paths; all25 existing caller roots/sentinels protected.|
|9|Authorized v2 tdd-red SHA, meaningful behavioral failures, passing real controls, no full preflight.|
|10|All25 full-path cases executed; successful candidate reports must bind exact native events and runtime bytes.|

### Ownership and limits
No labeled test containers or private host descendant processes remained. Real unrelated Docker sentinel survived every full-path callback before its own exact cleanup. User dagger-engine-v0.21.9 remains running with ID18576903a871d895c8b414ee0a41897313ce31d7b5b6284c29b489553e4fae99. No shared Docker image was removed.
No full scripts/preflight.sh, full native suite, hosted workflow, push/sync/GitHub mutation, installed binary/plugin/skill replacement, or Paivot product dependency. Candidate CLI still has no operational implementation: positive full-path tests intentionally fail until GREEN. Independent PM review remains required.

LEARNINGS:
- V1 standalone formal controls did not force candidate formal provisioning. V2 pairs positive and negative cases across the same CLI and executes the exact fixture as an independent health control.
- Nonempty report fields are not execution evidence. Every adapter now requires exact retained native identities and runtime byte/pin bindings.
- A real Go test can pass while leaving a host descendant alive. Container-only checks and fresh-directory-only checks missed separate process and caller-ownership obligations.
- Mandatory source wiring must inspect job and step error/condition policies and executable shell structure. Diagnostic wording is not a bypass oracle; real Bash controls demonstrate sensitivity.
- Pre-freeze authoring corrections (raw fixture placement and macOS lack of /bin/false) were corrected and replayed; the final v2 commit has no compilation or infrastructure failures.


## nd_contract
status: rejected

### evidence
- PM rejection applied via pvg story reject on 2026-09-05.

### proof
- [ ] Story requires another developer delivery before it can be accepted.


## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## Implementation Evidence

PROOF:
RED PHASE ONLY. Frozen test/config/fixture commit: 612f65f3b4502a3267828507faaf0e395c8dd558 on story/MAC-hpqp. Eight files, 1066 added lines. No operational lane implementation exists; the dispatcher-approved compile-safe run bootstrap returns a specific fail-closed status. The frozen tests do not treat its blanket nonzero return as successful negative validation.

### CI/Test Results

Commands run:
- Workdir for every command: /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-hpqp
- go test -count=1 -timeout=2m -coverprofile=/tmp/machinery-hpqp-red.f8Yslv/native.cover -json ./scripts/integration-lane
- go test -tags machinery_integration -count=1 -timeout=15m -json ./cmd/machinery -run '^TestIntegrationLane(Pilot(OCI|Formal)|FullPath)$'
- node --test --test-reporter=tap testdata/integration-lanes/pilot.integration.test.mjs
- go test -count=1 -timeout=90s -json ./scripts/run-safe ./internal/processcontrol
- go run ./scripts/integration-lane --lane required
- go test -tags machinery_integration -run '^$' -timeout=120s ./cmd/machinery (compilation-only check, zero executions, NOT RED evidence)
- pvg verify scripts/integration-lane/main.go scripts/integration-lane/main_test.go cmd/machinery/integration_lane_test.go testdata/integration-lanes/pilot.integration.test.mjs --include-tests --format text
- actionlint .github/workflows/ci.yml .github/workflows/formal.yml .github/workflows/nightly.yml
- go tool cover -func=/tmp/machinery-hpqp-red.f8Yslv/native.cover
- git diff --check
- docker ps -a --filter label=dev.machinery.integration-run --format '{{.ID}} {{.Names}} {{.Status}}'
- docker inspect --format '{{.Id}} {{.State.Running}}' dagger-engine-v0.21.9

Summary:
- Native RED: 10 top-level tests; 39 started/terminated test events including aggregate parents, 3 passed / 36 intended failed / 0 skipped. Leaf-only: 35 cases, 3 passed / 32 intended failed. The parent wiring test additionally asserts preflight order and the Make target.
- Runtime RED: all 3 registered Go pilot roots selected; 24 started/terminated test events including aggregate parents, 4 passed / 20 intended failed / 0 skipped. Leaf-only: 22 cases, 3 passed / 19 intended failed.
- Node pilot: 1 selected / 1 executed / 1 passed / 0 failed / 0 skipped / 0 cancelled / 0 todo.
- Existing bounded-runner controls: 18 roots, 22 started/terminated events, 22 passed / 0 failed / 0 skipped. Leaf-only: 20 passed / 0 failed.
- Every started ID has exactly a terminal event; missing-terminal inventories are empty. Counts match all 10 native root tests, 3 Go pilot roots (OCI 2 children, formal 1 root, full-path 19 scenarios), and the single registered Node case. No expected root/scenario was omitted.
- New leaf outcomes combined: 58 cases = 7 passing controls + 51 intended RED failures; no skips. Existing controls add 20 passing leaf cases.
- All RED failures concern the absent operational lane, missing category-specific rejection/report/execution, or current missing/unsafe workflow ordering. No frozen test failed compilation or because a required real runtime was unavailable.
- go run entrypoint exits 1 with 'required integration lane is not implemented', confirming the explicit bootstrap rather than a missing-package/build failure.
- pvg verify: PASSED (3 source files scanned, 0 issues). actionlint PASS. git diff --check PASS. Worktree clean.
- Coverage: 66.7% statements for the minimal bootstrap (run 100%, main 0%). This is NOT meaningful implementation coverage; no production lane behavior is implemented. Outcome/AC coverage is the explicit RED mapping below.
- Raw native/runtime JSON, Node TAP, controls JSON and coverage retained in /tmp/machinery-hpqp-red.f8Yslv/. Pipelines used set -o pipefail; expected RED command exit status remained 1.

### Real runtime and cleanup evidence

- Pinned image pulled and verified against RepoDigests and linux/amd64: python@sha256:c6ead215bfd31f1e433d968853b7a769989117115b728874824e6c0a27cb96fc.
- Actual OCI arithmetic returned 42; separate sleeping container was terminated. Every owned container had explicit network/read-only/memory/CPU/PID bounds, exact native cidfile, and unique ownership label. Containers were individually removed and absence checked.
- Actual pinned Java/TLC ran from a fresh private cache. Safe two-state finite model passed; same model with an invariant-breaking transition yielded the named Broken counterexample. No ambient Java launcher substituted for pinned provisioning.
- Actual Node pilot spawned a real child Node process, asserted different PID and computed result 42.
- Full-path harness itself provisioned the image and created a real unrelated sentinel container. All 19 scenario cleanups verified that sentinel remained running. It was removed afterward by its exact owned ID/label. No labeled test containers remained at final inspection.
- Existing user container dagger-engine-v0.21.9 remained running, ID 18576903a871d895c8b414ee0a41897313ce31d7b5b6284c29b489553e4fae99.
- Important RED limit: bootstrap rejection means the full-path candidate has NOT yet executed its nested tests or proved its own lifecycle handling. Those are intentional failing requirements for GREEN, not current success claims.
- Ordinary native selection was checked with unavailable Docker/Java environment and actual go list on ./cmd/machinery: integration_lane_test.go is explicitly excluded; no required runtime case is skipped.
- No full native suite, hosted Linux workflow or full scripts/preflight.sh execution performed. Final cross-platform/full-lane replay remains the epic completion gate.

### AC Verification

| AC | Frozen RED location and outcome asserted | RED state |
|---|---|---|
| 1 | TestLaneClosedInventoryRejectsUnprovedSelections (15 mutations), TestLaneDiscoversUnionAndRejectsIdentityRegisteredTwice, schema.json: closed inventory, source ownership, paths, orphan/duplicate/empty/future selection | Defined; intended RED |
| 2 | TestLaneNativeSelectionDoesNotProbeServices, TestLaneRuntimeAbsenceFailsBeforeTests, native skip/zero cases, full-path missing Docker/Node cases: explicit native exclusion and mandatory runtime failures | Passing separation control; required behavior RED |
| 3 | TestIntegrationLanePilotOCI and PilotFormal; full-path cold-cache, wrong mutable pin/digest/platform/Java/TLC pin and offline-formal cases; runtime-pins.json | Real provisioned controls PASS; lane validation RED |
| 4 | TestLaneSuccessRequiresActualNonzeroExecution verifies retained native events/hash and repeated real execution; native failure/forgery/duplicate cases; actual repeated m.Run control; Node zero/skip/partial/aggregate cases | Controls PASS; accounting RED |
| 5 | TestLaneWiringIsMandatoryAndNativeJobsAreServiceFree parses all three workflows and checks Make entrypoint; actual Darwin/arm64 runtime pilot uses pinned linux/amd64 | Wiring RED; local runtime control PASS |
| 6 | Wiring test asserts ordinary race before self-provisioning required lane before formal, mandatory invocation, no SKIP_PREFLIGHT bypass; full-history checkout checks | Existing missing lane/order/bypass RED |
| 7 | Nonempty pilot.json registers all 3 actual Go roots and 1 actual Node case; union discovery test; real OCI, TLC and Node controls | Runtime controls PASS; shared union runner RED |
| 8 | Real OCI success/termination, full-path leak/assertion/timeout/cancellation/provision failure challenges, exact-ID/label emergency cleanup, unrelated sentinel preservation, owned-work assertions | Standalone controls PASS; candidate lifecycle RED |
| 9 | tdd-red commit 612f65f3b4502a3267828507faaf0e395c8dd558; meaningful assertion failures and separately passing real controls; actionlint/pvg verify | RED delivered for independent approval only |
| 10 | TestIntegrationLaneFullPath has all 19 real-process/daemon scenarios, no replacement executables; raw events required in successful reports; no service skips | Complete RED scenario set; operational success awaits GREEN |

LEARNINGS:
- A new CLI requires an explicitly reviewed fail-closed bootstrap; missing packages and compiler errors do not constitute useful RED. Category-specific negative assertions prevent a blanket failure stub from satisfying validation tests.
- Native Go can genuinely execute the same test twice when TestMain invokes m.Run twice. The real duplicate-execution control confirms why started/terminal cardinality must be checked, not just pass totals.
- Go wraps test stderr into JSON stdout; resource ceilings must account for the actual adapter stream topology. Reports must retain/hash real native events rather than replace them.
- Docker daemon resources require native cidfile plus exact label ownership and cleanup registered before launch. Docker29 lowercase no-such-object text required a pre-freeze portability adjustment.
- Before freeze, the TLA fixture needed EXTENDS Integers and one Go declaration needed correction. Neither compiler/parser failure was counted as RED; final pinned-runtime controls pass.

### Limits and deferred review

Workflow checks establish wiring, not actual hosted execution. Preserve existing formal/C4/checker gates during GREEN review; full preflight is intentionally deferred. Fresh Java/TLC cache provisioning is proven; the existing shared Docker image was not removed merely to simulate a cold daemon, because that could affect the NIL agent. The CLI must verify/provision the pinned image on every path and its full-path tests cannot use fabricated runtime binaries.

No push/sync/GitHub mutation, installed binary/plugin/skill replacement, Paivot product dependency, full preflight, or changes to the root checkout occurred. Story must be RED-approved, not accepted/closed.


## RED contract decisions
Dispatcher approved the minimal fail-closed run(args, stdout, stderr) int bootstrap before authoring it. It has no validation, provisioning, execution, or cleanup behavior. Generic bootstrap rejection is not proof of correct negative validation; negative RED assertions require diagnostic categories. Dispatcher also approved the additive real Node pilot fixture. CONTRACT.md documents source selection, native events, report semantics, bounded resources, and exact container ownership. No Machinery product dependency on Paivot is introduced.

Targeted preparation: existing scripts/run-safe and internal/processcontrol suites passed; exact OCI pull and RepoDigests/platform inspection passed; actionlint passed. Full preflight remains deferred. Initial Docker pilot exposed a test diagnostic portability assumption: Docker29 emits lowercase no such object, so pre-freeze cleanup assertions were corrected case-insensitively; created test containers were already removed successfully. This is test-authoring repair, not an implementation defect.

## nd_contract
status: new

### evidence
- Backlog-only integration-lane contract authored from actual CI/preflight ordering and bounded runner interfaces.
- No production edits or heavy preflight; parent-authorized Anchor round-1 repair.

### proof
- [ ] AC #1: independent execution-lane verification pending
- [ ] AC #2: independent execution-lane verification pending
- [ ] AC #3: independent execution-lane verification pending
- [ ] AC #4: independent execution-lane verification pending
- [ ] AC #5: independent execution-lane verification pending
- [ ] AC #6: independent execution-lane verification pending
- [ ] AC #7: independent execution-lane verification pending
- [ ] AC #8: independent execution-lane verification pending
- [ ] AC #9: independent execution-lane verification pending
- [ ] AC #10: independent execution-lane verification pending

## History
- 2026-09-05T19:45:31Z dep_added: blocks MAC-hlae
- 2026-09-05T19:45:31Z dep_added: blocks MAC-yhg5
- 2026-09-05T19:45:32Z dep_added: blocks MAC-2n83
- 2026-09-05T19:45:32Z dep_added: blocks MAC-hwdb
- 2026-09-05T19:45:32Z dep_added: blocks MAC-vx24
- 2026-09-05T19:45:33Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:45:33Z dep_added: blocks MAC-ou97
- 2026-09-05T19:45:33Z dep_added: blocks MAC-hy71
- 2026-09-05T20:00:19Z status: open -> in_progress
- 2026-09-05T20:00:19Z claimed by dev-MAC-hpqp
- 2026-09-05T20:25:54Z status: in_progress -> in_progress
- 2026-09-05T20:32:43Z status: in_progress -> open
- 2026-09-05T20:32:43Z released by ramirosalas
- 2026-09-05T20:33:44Z status: open -> in_progress
- 2026-09-05T20:33:44Z auto-follows: linked to predecessor MAC-olrx
- 2026-09-05T20:33:44Z claimed by dev-MAC-hpqp
- 2026-09-05T20:54:28Z status: in_progress -> in_progress
- 2026-09-05T21:01:13Z status: in_progress -> open
- 2026-09-05T21:43:52Z status: open -> in_progress
- 2026-09-05T21:43:52Z claimed by dev-MAC-hpqp
- 2026-09-06T09:09:59Z dep_added: blocks MAC-bz1y
- 2026-09-06T09:10:14Z dep_added: blocked_by MAC-cn7q
- 2026-09-07T01:42:39Z dep_removed: was_blocked_by MAC-cn7q
- 2026-09-07T04:45:25Z status: in_progress -> closed
- 2026-09-07T04:45:25Z dep_removed: no_longer_blocks MAC-hlae
- 2026-09-07T04:45:25Z dep_removed: no_longer_blocks MAC-yhg5
- 2026-09-07T04:45:25Z dep_removed: no_longer_blocks MAC-2n83
- 2026-09-07T04:45:25Z dep_removed: no_longer_blocks MAC-hwdb
- 2026-09-07T04:45:25Z dep_removed: no_longer_blocks MAC-vx24
- 2026-09-07T04:45:25Z dep_removed: no_longer_blocks MAC-gcrr
- 2026-09-07T04:45:25Z dep_removed: no_longer_blocks MAC-ou97
- 2026-09-07T04:45:25Z dep_removed: no_longer_blocks MAC-hy71
- 2026-09-07T04:45:25Z dep_removed: no_longer_blocks MAC-bz1y

## Links
- Parent: [[MAC-ui8a]]
- Was blocked by: [[MAC-cn7q]]
- Follows: [[MAC-olrx]]
- Led to: [[MAC-2n83]], [[MAC-yhg5]], [[MAC-hlae]], [[MAC-hwdb]], [[MAC-hy71]], [[MAC-gcrr]]

## Comments

### 2026-09-05T20:27:03Z ramirosalas
### Commit
- Branch: story/MAC-hpqp
- SHA: 612f65f3b4502a3267828507faaf0e395c8dd558

### Frozen RED inventory clarification
Frozen test/fixture inputs: scripts/integration-lane/main_test.go; cmd/machinery/integration_lane_test.go; testdata/integration-lanes/schema.json, pilot.json, runtime-pins.json, pilot.integration.test.mjs and CONTRACT.md. The explicitly reviewed scripts/integration-lane/main.go bootstrap is not frozen implementation: GREEN must replace its fail-closed body with operational behavior, preserving the approved CLI/test contract. Its presence is recorded to distinguish behavioral RED from a missing package or compile error.

### Delivery metadata repair
The canonical deliver transition was run exactly once. The initial verify-delivery saw an older Notes contract last in serialized issue content and did not recognize the prose SHA. This supported append-only terminal comment adds canonical SHA metadata and authoritative delivered RED contract without removing history or changing labels.

## nd_contract
status: delivered

### evidence
- RED SHA: 612f65f3b4502a3267828507faaf0e395c8dd558; 8 files, 1066 lines, within story budget.
- New RED leaf cases: 58 total, 7 passing controls / 51 expected failing requirements / 0 skips. Existing controls: 20 passing leaf cases, 0 failures/skips.
- Exact commands, raw native events, AC mapping, PROOF and LEARNINGS are in Implementation Evidence; logs /tmp/machinery-hpqp-red.f8Yslv.
- pvg verify PASS; actionlint PASS; no owned runtime resources remain; user dagger-engine remains running.
- No full preflight, push, install, or installed binary replacement. Delivery is RED only, never acceptance/closure.

### proof
- [x] AC #1: closed inventory/source ownership RED tests and schema defined.
- [x] AC #2: mandatory-versus-native lane RED tests and passing separation controls defined.
- [x] AC #3: pin/provisioning RED tests and actual OCI/Java/TLC controls defined.
- [x] AC #4: native events/accounting/duplicate/truncation RED tests defined.
- [x] AC #5: shared CI/Make/native-lane wiring RED tests defined.
- [x] AC #6: ordering/full-history/mandatory-gate RED assertions defined.
- [x] AC #7: nonempty executable Go/Node pilot fragment and union RED test defined.
- [x] AC #8: owned cleanup/lifecycle challenges and real sentinel controls defined.
- [x] AC #9: compile-safe behavioral RED frozen and independently reviewable.
- [x] AC #10: all 19 full-path scenarios plus raw-report assertions defined; operational lane remains RED awaiting GREEN.


### 2026-09-05T20:32:44Z ramirosalas
## PM Decision
REJECTED [2026-09-05]: RED contract needs stronger acceptance assertions before freeze. The authorized fail-closed bootstrap itself is not a rejection reason.

EXPECTED: AC #3/#7/#10 require provisioned full-path success and a cold-cache positive through the required lane.
DELIVERED: cmd/machinery/integration_lane_test.go:143 calls formal.VerifyFormalTo directly; CLI formal scenarios at lines 220-222 are only negative. success and cold-cache both select Go+Docker after the harness already pulls the image (lines 174, 198, 218), with no fresh closure/provisioning assertion.
GAP: A lane that rejects every formal request, or never provisions a cold runtime closure, can satisfy these assertions while the separate formal control passes.
FIX: Add a real CLI-path valid Java/TLC suite starting from a verified empty private cache, executing the provisioned closure and verifying its expected identities and native events. Keep offline and invalid-pin negatives paired with that positive; do not remove shared Docker images.

EXPECTED: AC #4/#10 require exact native test identities, complete events and per-runtime evidence; reports cannot substitute for native events.
DELIVERED: Full-path receipt (lines 342-369) decodes only Started/Passed for suites and accepts any nonempty runtime ID/Identity. Node success never reads its TAP/events file, hash, exact selected source/name, or terminal cardinality. The stronger raw-event assertion exists only for a one-test Go fixture.
GAP: Node aggregate counts with a marker and an arbitrary runtime record pass the success assertions; exact Node event evidence and required runtime union/pins are unproved.
FIX: Assert retained/hash-bound native events and exact source/test/terminal identities for Node and full-path Go, exact required runtime set and expected verified identities, and real duplicate/missing-terminal Node rejection controls.

EXPECTED: AC #8 requires owned processes and temporary roots reclaimed on success/failure/timeout/cancellation/provision error while preserving user files.
DELIVERED: Lifecycle fixtures create only Docker containers and fresh roots (lines 197-200, 371-387, 392-417); no independently observed descendant process or preexisting caller-owned work-root sentinel is challenged.
GAP: Killing only the lane parent and deleting caller-owned directories can escape these tests.
FIX: Add bounded real descendant PID/liveness challenges with exact-ownership emergency cleanup across the lifecycle paths, and an existing --work-dir containing an unrelated user sentinel whose content must survive.

EXPECTED: AC #2/#5/#6 require mandatory CI/preflight execution without hidden optional gating.
DELIVERED: Workflow parser at scripts/integration-lane/main_test.go:384-417 ignores job-level if and step-level continue-on-error; invocation/order checks are substring matches. Preflight bypass check at 432 only rejects one diagnostic text.
GAP: A job-level false condition or continue-on-error lane step passes this mandatory-lane oracle; renaming the bypass message evades the preflight check.
FIX: Assert execution-affecting job and step conditions/error policies, and validate executable mandatory preflight ordering/bypass behavior rather than comment/message presence. Add negative fixture controls showing each disabled or ignored lane is rejected.

## nd_contract
status: rejected

### evidence
- Reviewed full story, all eight files at 612f65f3b4502a3267828507faaf0e395c8dd558 and the documented bootstrap exception; no test/fixture edits.
- Independent targeted native/runtime replay evidence and exact final counts are recorded in the terminal PM comment.

### proof
- [x] AC #1: closed inventory/source mutations and union contract reviewed.
- [ ] AC #2: mandatory gating assertions incomplete.
- [ ] AC #3: valid fresh formal closure through candidate lane unproved.
- [ ] AC #4: exact Node/native runtime evidence incomplete.
- [ ] AC #5: workflow job/step bypasses unchallenged.
- [ ] AC #6: preflight bypass oracle depends on message text.
- [ ] AC #7: formal pilot runs directly, not a valid candidate-lane closure positive.
- [x] AC #9: behavioral RED marker and authorized bootstrap verified; full preflight deferred.
- [ ] AC #8: host-process and existing user-root preservation unproved.
- [ ] AC #10: full-path positive/report contract needs the above repairs.

### Revised RED authorization
TEST-EDIT AUTHORIZED: scripts/integration-lane/main_test.go; cmd/machinery/integration_lane_test.go; testdata/integration-lanes/schema.json, pilot.json, runtime-pins.json, pilot.integration.test.mjs and CONTRACT.md -- the current candidate RED has not been approved. Revise these tests/fixtures/contracts only as needed to close the four explicit gaps above, then commit a new tdd-red candidate with [test-edit-authorized] in the repair commit subject. Preserve the original 612f65f3b4502a3267828507faaf0e395c8dd558 reproduction evidence, label candidate versions clearly, and independently re-review the new SHA before freezing. Controls must reject an implementation omitting required formal or Node execution and one deleting caller-owned --work-dir content. These guarantees belong in MAC-hpqp, not a deferred capstone. The dispatcher-authorized main.go bootstrap remains replaceable implementation; this review authorizes no operational GREEN behavior in RED.

### 2026-09-05T20:33:18Z ramirosalas
### Commit
- Branch: story/MAC-hpqp
- SHA: 612f65f3b4502a3267828507faaf0e395c8dd558

### PM independent RED review
Canonical rejection recorded with four EXPECTED/DELIVERED/GAP/FIX findings and explicit TEST-EDIT AUTHORIZED revisions in PM Decision. This is rejected RED candidate v1, never accepted/closed or frozen as approved. Preserve its reproduction evidence; submit repaired RED v2 at a new tdd-red SHA for independent review. No deferred capstone substitution or root override.

PROOF:
Working directory /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-hpqp:
- go test -count=1 -timeout=2m -json ./scripts/integration-lane: expected exit 1; exact 10 roots/39 starts/39 terminal events; 35 leaves = 3 PASS/32 intended FAIL/0 SKIP.
- go test -tags machinery_integration -count=1 -timeout=15m -json ./cmd/machinery -run '^TestIntegrationLane(Pilot(OCI|Formal)|FullPath)$': expected exit 1; exact three registered roots/24 starts/24 terminals; 22 leaves = 3 PASS/19 intended FAIL/0 SKIP. All 19 declared scenario identities verified individually. Actual OCI success/termination pass; pinned fresh-cache safe/unsafe Java/TLC control passes in 62.61s.
- node --test --test-reporter=tap testdata/integration-lanes/pilot.integration.test.mjs: 1 exact registered leaf PASS, 0 fail/skip/cancel/todo.
- go test -count=1 -timeout=90s -json ./scripts/run-safe ./internal/processcontrol: 18 roots/22 starts/22 terminals; 20 leaves PASS, no failures/skips.
- Parsed every JSON event: no duplicate started/terminal IDs or missing terminals in native/runtime/controls. New total 58 leaves = 7 PASS/51 intended FAIL/0 SKIP; existing controls add 20 PASS.
- actionlint .github/workflows/ci.yml .github/workflows/formal.yml .github/workflows/nightly.yml PASS; git diff --check PASS; worktree clean.
- docker ps -a --filter label=dev.machinery.integration-run returned empty after replay. docker inspect verified dagger-engine-v0.21.9 ID 18576903a871d895c8b414ee0a41897313ce31d7b5b6284c29b489553e4fae99 remains running. All 19 scenario cleanup callbacks preserved real unrelated sentinel.
- Raw replay logs retained /tmp/machinery-hpqp-pm.9vqSxW/{native,runtime,controls}.json. No coverage claim added: developer bootstrap coverage 66.7% is not operational coverage.
- Read entire eight-file/1066-line diff and closed contract. Code graph root snapshot excluded/missed candidate paths; all relevant source read directly. No production, test, fixture, other-worktree or installed artifact edits; no full preflight, GitHub/push/sync or global cleanup.

LEARNINGS:
Passing standalone runtime controls establish infrastructure health but do not force a candidate lane to provision and execute that runtime. Candidate positive and negative pairs must cross the same CLI boundary. Native events and runtime identities need exact assertions for every adapter.

## nd_contract
status: rejected

### evidence
- Independent review and exact event replay at SHA 612f65f3b4502a3267828507faaf0e395c8dd558 reproduced developer counts and safe cleanup.
- Canonical pvg story reject MAC-hpqp completed; test-revision authorization and four actionable gaps recorded above in PM Decision.

### proof
- [x] AC #1: closed inventory and actual source-selection RED reviewed.
- [ ] AC #2: mandatory gating assertions need job/step/bypass controls.
- [ ] AC #3: positive cold formal closure through candidate CLI required.
- [ ] AC #4: exact Node/full-path native events and runtime identity set required.
- [ ] AC #5: workflow mandatory job and step policies unproved.
- [ ] AC #6: executable preflight bypass/order assertions required.
- [ ] AC #7: candidate-lane formal positive cannot be replaced by direct verifier control.
- [ ] AC #8: real owned descendants and preexisting user work-root preservation required.
- [x] AC #9: valid behavioral RED and bootstrap exception verified; rework authorization explicit.
- [ ] AC #10: strengthen full-path success/evidence/lifecycle controls before approval.

### 2026-09-05T20:54:55Z ramirosalas
### Commit
- Branch: story/MAC-hpqp
- SHA: e55238223961fec922896c361d8af6cafc454a8e

### RED candidate v2 authoritative delivery
V2 revises rejected v1 under TEST-EDIT AUTHORIZED. V1 SHA612f65f3b4502a3267828507faaf0e395c8dd558 and all reproduction/PM evidence remain intact; neither candidate is called approved.

Frozen v2 test/config/fixture inventory, exactly:
- scripts/integration-lane/main_test.go
- cmd/machinery/integration_lane_test.go
- testdata/integration-lanes/schema.json
- testdata/integration-lanes/pilot.json
- testdata/integration-lanes/runtime-pins.json
- testdata/integration-lanes/pilot.integration.test.mjs
- testdata/integration-lanes/CONTRACT.md

scripts/integration-lane/main.go remains the separately authorized, byte-unchanged fail-closed bootstrap. It is replaceable GREEN implementation, NOT frozen implementation bytes; GREEN must preserve the approved CLI contract and all frozen inputs above.

PROOF:
Exact v2 commands, counts, four-gap sensitivity proofs, AC1-10 mapping, coverage limits and LEARNINGS are in the new Implementation Evidence block. Canonical deliver was run once for this v2 delivery; this supported terminal comment ensures serialized Comments ordering cannot leave an older rejected contract authoritative.

## nd_contract
status: delivered

### evidence
- V2 SHA: e55238223961fec922896c361d8af6cafc454a8e;8files1753lines,within budget.
- New leaves96=39PASS/57intendedRED/0SKIP; existing controls20leavesPASS. Exact roots,starts,terminals,missing/duplicate inventories checked.
- Native70events34PASS36RED; runtime34events8PASS26RED; real Node pilot1PASS.
- Raw JSON/TAP: /tmp/machinery-hpqp-red-v2.DQvyDf/.
- pvg verify and actionlint PASS; no owned containers/host descendants remain; Dagger/user sentinel protection verified.
- No operational implementation, full preflight, push/sync/install or installed artifact replacement.

### proof
- [x] AC #1: closed inventory/source contract preserved.
- [x] AC #2: native separation and mandatory job/step/bypass assertions.
- [x] AC #3: candidate cold formal closure positive with paired negatives and exact pins.
- [x] AC #4: exact Go/Node events/runtime union plus real duplicate/incomplete controls.
- [x] AC #5: mandatory Linux and exact shared Make invocation.
- [x] AC #6: transparent strict ordering and executable bypass sensitivity.
- [x] AC #7: all5 Go roots plus Node pilot registered, formal CLI positive mandatory.
- [x] AC #8: host/container lifecycle and preexisting caller-root preservation challenges.
- [x] AC #9: authorized v2 behavioral RED committed for independent review.
- [x] AC #10: all25 full-path scenarios plus exact execution/report assertions.


### 2026-09-05T21:01:37Z ramirosalas
### Commit
- Branch: story/MAC-hpqp
- SHA: e55238223961fec922896c361d8af6cafc454a8e

### PM RED v2 decision
RED APPROVED after independent re-review from static gates through actual native/runtime replay. All four original rejection gaps are closed in the acceptance contract. Original rejected v1 evidence remains historical; v2 is the reviewed frozen candidate. This does not accept/close the story or claim operational GREEN behavior.

PROOF:
Working directory /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-hpqp:
- go test -count=1 -timeout=2m -json ./scripts/integration-lane: expected exit 1; 12 roots,70 starts/70 terminals;64 leaves=32PASS/32intendedFAIL/0SKIP.
- go test -tags machinery_integration -count=1 -timeout=15m -json ./cmd/machinery -run '^TestIntegrationLane(Pilot(OCI|Formal)|FullPath|HostDescendantControl|NodeEventControls)$': expected exit1; all5 registered roots,34 starts/34 terminals;31 leaves=6PASS/25intendedFAIL/0SKIP. All25 full-path scenario IDs individually present.
- node --test --test-reporter=tap testdata/integration-lanes/pilot.integration.test.mjs: exact registered case1PASS/0FAIL/0SKIP/0cancel/todo.
- go test -count=1 -timeout=90s -json ./scripts/run-safe ./internal/processcontrol:18 roots/22 starts/22 terminals,20 leafPASS/0FAIL/0SKIP.
- Exact JSON inventories contain no missing terminal events, duplicate starts, duplicate terminals, or omitted selected identities. Combined new96 leaves=39passing controls/57intendedRED/0SKIP.
- The39PASS controls include25 structural workflow/preflight mutation checks and4 actual small Bash sensitivity programs. Those29 are not candidate runtime successes. Actual OCI arithmetic/termination, fresh pinned Java/TLC safe/unsafe plus exact generated formal fixture, real orphan-descendant/liveness control, real duplicate/hanging Node event controls and real Node child pilot passed.
- pvg verify scripts/integration-lane/main.go scripts/integration-lane/main_test.go cmd/machinery/integration_lane_test.go testdata/integration-lanes/pilot.integration.test.mjs --include-tests --format text PASS(3files/0issues); actionlint on ci/formal/nightly PASS; git diff --check PASS.
- Re-read full current story and v1-to-v2 diff, plus relevant unchanged contract/tests. Candidate8files1753addedlines; v2 alters only4 authorized test/contract/fixture files. tdd-red and [test-edit-authorized] markers verified. main.go byte-identical to approved replaceable bootstrap.
- Raw replay JSON retained /tmp/machinery-hpqp-pm-v2.BGiMqQ/{native,runtime,controls}.json. Bootstrap66.7% developer statement coverage is not operational implementation coverage.
- Code graph remains stale/excluded for candidate paths; direct source used for all review claims. No product Paivot dependency, installed artifact edits, full preflight, full native suite, hosted run, push/sync/GitHub mutation or unrelated-worktree changes.

### Original gap closure
1. cold-cache now invokes the candidate CLI on a verified-empty private Go+Java+TLC cache. The exact generated native fixture runs safe/unsafe TLC via supplied closure paths, binds actual launcher/jar hashes to report identities and private cache paths, and pairs with wrong-pin/offline negatives. A blanket formal rejection cannot pass.
2. Full-path positive reports require exact adapter/source/test and started/passed cardinality; retained Go JSON/Node TAP bytes and hashes are verified. Required runtime union, native versions/executable hashes and OCI/Java/TLC pin identities are checked. Node duplicate/incomplete native controls and candidate negative cases are present. Omitting Node execution cannot pass its marker/events positive.
3. All25 paths preserve existing caller-root native identity, exact user-sentinel content and no owned residue, with cleanup assertions registered before invocation. Four actual host-descendant scenarios independently inspect exact PID/private script and require termination; standalone orphan control proves the challenge is live. Existing Docker ownership/sentinel checks remain.
4. Mandatory workflow guards now inspect job and step conditions/error policies and exact invocation; preflight checks executable tokens, strict top-level ordering and bypasses. 25 disabled wiring fixtures plus4 actual bounded Bash controls establish sensitivity to the reported faults.

### Frozen RED boundary and GREEN handoff
Freeze scripts/integration-lane/main_test.go; cmd/machinery/integration_lane_test.go; testdata/integration-lanes/schema.json, pilot.json, runtime-pins.json, pilot.integration.test.mjs and CONTRACT.md at v2 SHA. scripts/integration-lane/main.go remains replaceable implementation, preserving the frozen public CLI contract. All57 candidate requirements remain intentionally red at that bootstrap and must pass unchanged in GREEN. Re-review complete CI/preflight wiring and final implementation; full epic preflight/hosted coverage remain deferred as authorized. GREEN may exceed the combined1800LOC estimate: investigate and justify final scope/volume; never trim approved tests to fit.

LEARNINGS:
Positive and negative integration contracts now cross the same CLI boundary. Exact native streams, runtime-byte identities, independently observed descendants and caller-owned sentinels make the previously missing outcomes observable.

### Final ownership verification
docker ps -a --filter label=dev.machinery.integration-run returned empty; bounded private shell-descendant inventory returned empty. All25 caller-root/sentinel cleanup callbacks passed. Dagger ID18576903a871d895c8b414ee0a41897313ce31d7b5b6284c29b489553e4fae99 remains running. Worktree clean. One PM ad-hoc JSON summarizer typo was corrected before final inventory checks; it was not a test or product failure.

## nd_contract
status: new

### evidence
- Canonical pvg story approve-red MAC-hpqp completed; verified open with hard-tdd,red-approved and without delivered/rejected/accepted.
- Independent RED v2 SHA e55238223961fec922896c361d8af6cafc454a8e;96new leaves39PASS/57intendedFAIL/0SKIP;20existing controlsPASS.
- RED-only approval returns story to ready GREEN queue. No acceptance or closure; serialized terminal contract appended after historical Comments.

### proof
- [x] AC #1: closed fragment/schema/source/identity/union RED contract reviewed.
- [x] AC #2: service-free native exclusion and mandatory runtime/gating RED assertions verified.
- [x] AC #3: same-CLI cold formal positive/negative closure contract and real fixture control verified.
- [x] AC #4: exact native Go/Node events and runtime union identities required by RED.
- [x] AC #5: mandatory Linux job/step and shared Make lane contract verified.
- [x] AC #6: executable bypass/order guard sensitivity verified; existing consumers retained in contract.
- [x] AC #7: five runnable Go roots plus real Node case; independent candidate formal positive mandatory.
- [x] AC #8: caller ownership, owned host/container lifecycle and unrelated sentinel protection asserted.
- [x] AC #9: authorized behavioral RED v2 frozen; replaceable bootstrap exception retained.
- [x] AC #10: all25 real-process full-path challenges executed in RED, exact positive native evidence required for GREEN.


### 2026-09-05T22:09:12Z ramirosalas
### GREEN scope-review pause (undelivered; claim retained)

Current committed SHA: 221525d7baf7a565ece9972d09bf916132e68050 on story/MAC-hpqp. Healthy worktree retained. Dispatcher explicitly requested this scope-review pause; no additional file/API scope is authorized yet and pvg story deliver has NOT been run.

PROOF (checkpoint only, not complete delivery):
Commands run:
- go test -count=1 -timeout=2m -json ./scripts/integration-lane: native-3.json PASS, frozen 64 leaves, 70 starts/70 terminal events, 0 failures/skips, 11.446s. Includes 25 structural mutation controls and 4 real bounded Bash controls; those are not runtime success evidence.
- go test -count=1 -timeout=2m -coverprofile=/tmp/machinery-hpqp-green.g6rAM2/coverage.out -json ./scripts/integration-lane: native-2.json PASS at 4157200, statement coverage 58.9%; this excludes subprocess runtime coverage.
- go test -tags machinery_integration -count=1 -timeout=15m -json ./cmd/machinery -run '^TestIntegrationLane(Pilot(OCI|Formal)|FullPath|HostDescendantControl|NodeEventControls)$': runtime-1.json at 4157200 produced 31 leaves, 30 PASS/1 FAIL/0 SKIP. Only failed leaf was cold-cache: correct actual closure hashes but Java path /private/var spelling did not remain lexically rooted in caller /var cache. All 24 other full-path scenarios and real OCI/formal/host/Node controls passed.
- go test -tags machinery_integration -count=1 -timeout=5m -json ./cmd/machinery -run '^TestIntegrationLaneFullPath$/^cold-cache$': cold-2.json at 72bd28d PASS; exact selected cold-cache leaf passed after canonical containment plus caller-path spelling fix, 63.17s. This targeted result is not an unfiltered final runtime replay.
- Frozen union reconciliation is exactly 64 native Go leaves +31 tagged Go runtime leaves +1 standalone Node pilot =96. Standalone Node pilot and actual go run ./scripts/integration-lane --lane required remain outstanding at final GREEN SHA.
- golangci-lint run --config .golangci.yml --timeout 3m ./scripts/integration-lane: 0 issues on current code. actionlint .github/workflows/ci.yml .github/workflows/formal.yml .github/workflows/nightly.yml PASS. shellcheck -x scripts/preflight.sh PASS. Initial bare shellcheck lacked -x dependency following and was corrected; no source suppression.
- scripts/shellcheck-inventory.sh PASS; scripts/shellcheck-files.txt unchanged because no shell surface added.
- pvg verify scripts/integration-lane/main.go scripts/preflight.sh .github/workflows/ci.yml .github/workflows/formal.yml .github/workflows/nightly.yml Makefile --format text: PASS (1 supported source file scanned, 0 issues).
- pvg story verify-tdd --range 497419ab4512fcff765cd5feb27aed4c67b5608d..HEAD --json: 6 commits checked, no violations. Includes original RED v1 and independently authorized RED v2 marker history.
- Exact git diff against frozen e55238223961fec922896c361d8af6cafc454a8e for both frozen Go tests and entire testdata/integration-lanes: empty. No new supplemental tests yet.
- Current combined original-base delta:13 files,3004 insertions/11 deletions; main.go1213 lines. The initial estimate was investigated explicitly; runtime/schema/native-accounting/ownership duties cannot fit47 lines remaining after1753 RED lines.
- docker ps -a --filter label=dev.machinery.integration-run returned empty. Dagger18576903a871d895c8b414ee0a41897313ce31d7b5b6284c29b489553e4fae99 remains running. Native/runtime test cleanup checks preserved user sentinels and host descendants. No installed binary/skills/plugins/agents, root branch, unrelated worktrees, remote or GitHub changes; no full preflight.

### Scope extension requested for AC8 (not authorized)
Code-path evidence: scripts/integration-lane/main.go provision line759 launches the bounded self helper; provisionFormal line845 calls existing formal.VerifyFormalTo. internal/formal/formal.go runTLC lines546-591 creates a JVM using processcontrol; internal/processcontrol/run_unix.go prepare sets a fresh process group. If the outer runner cancels while the helper is in the formal verifier, killing the helper process group does not necessarily kill the nested JVM group. This is a material unclosed AC8 concern, not a claimed observed leak or a test dispute. Approved tests currently cover native-suite cancellation, not this specific provisioning-stage nested-group cancellation.

Concrete minimal proposal:
- Authorize NEW production internal/formal/provision.go (approximately8 lines) exposing internal-package ProvisionTLC() (string,error), delegating unchanged ensureJar() for existing checksum, cache-lock, staging/recovery semantics.
- Replace helper formal.VerifyFormalTo with runtimeclosure.OpenJava + formal.ProvisionTLC. Keep helper provisioning free of descendant JVM groups. Runner directly owns bounded Java probe/TLC validation commands before suite selection; required frozen formal pilot still proves actual safe/unsafe verification.
- Provision into a task-owned child of caller cache, preserve successful inspectable runtime artifacts, and remove only that exact child after failed/cancelled provisioning so interrupted cache stages cannot remain hidden.
- Add NEW supplemental negative/control coverage for provisioning-stage cancellation and retained caller roots/resources; do not modify frozen tests/config/fixtures. Exact new test/fragment paths should be approved alongside API scope.
- Independent PM must review extension and final combined overrun. No processcontrol shared-module changes are proposed.

LEARNINGS:
- A bounded capture buffer must cancel the producer on overflow; initial implementation waited for timeout. Immediate cancellation now reuses processcontrol.Run and bounded io.Writer.
- macOS /var and /private/var alias equivalence must be checked canonically while preserving caller cache spelling in the report and test argv.
- A bounded outer helper does not prove ownership of grandchildren that create separate process groups. Reuse provisioning independently of engine execution to make cancellation authority direct.

## nd_contract
status: in_progress

### evidence
- Committed221525d7baf7a565ece9972d09bf916132e68050; checkpoint logs /tmp/machinery-hpqp-green.g6rAM2/.
- Dispatcher requested pause for canonical scope review; no delivery, acceptance or claim release.

### proof
- [x] AC #9: approved RED bytes unchanged; no full preflight.
- [ ] AC #8: provisioning-stage nested JVM cleanup remains under explicit scope review.
- [ ] AC #10: final unfiltered required-lane/native proof remains outstanding after scope resolution.


### 2026-09-05T22:10:13Z ramirosalas
### AC8 scope-review correction: actual suite nesting also matters

The provisioning-only wrapper proposal is insufficient to claim AC8 solved. The same concrete call chain exists during required test execution:
- scripts/integration-lane/main.go execute -> command -> processcontrol.Run launches go test in its own group.
- Frozen TestIntegrationLanePilotFormal (cmd/machinery/integration_lane_test.go) calls formal.VerifyFormalTo -> runTLC.
- internal/formal/formal.go runTLC launches Java through runBoundedProcess/processcontrol, which creates another process group in internal/processcontrol/run_unix.go prepare.
- The frozen full-path meta-test also launches nested integration-lane processes that establish their own groups.

These nested paths executed in the real frozen runtime replay. Cancellation at the outer lane boundary while a nested verifier/JVM is active has not yet been experimentally exercised; therefore this is a source-established ownership gap with live call paths, not claimed observed leakage.

Revised scope options for Sr PM:
1. The small ProvisionTLC wrapper removes only provisioning-stage nesting. It does not eliminate nested process groups created by actual required native tests. Keep its usefulness and its limit explicit.
2. Review a shared processcontrol ownership design that propagates a per-lane, verifiable ancestor-owned process scope through nested launches and permits cleanup of every registered owned group while preventing PID/group reuse from authorizing foreign cleanup. This materially expands file/platform scope and needs its own hard-TDD proof for Linux/macOS, normal completion, nested cancellation and failed registration. Plain numeric PID files, broad process matching or unverified group kills are not adequate.
3. Alternatively review a suite execution supervisor that owns all nested processes (and exact closure/source snapshots) with native platform containment. The local macOS Docker lane requirement means Linux-only cgroups cannot alone satisfy this story. Running the entire lane inside a private OCI supervisor would require explicit source/cache/runtime/report contract review, not a silent implementation choice.

Formal execution guarantees to preserve: runTLC snapshots the verified JAR into its private meta root and revalidates the Java identity after execution. A new exported provisioning pathname is only a checked provisioning result; it must not replace actual engine execution evidence or weaken those snapshot/revalidation guarantees. Any direct parent Java/TLC validation must retain the relevant immutable executable/JAR closure guarantees and bounded private scratch, with a new cancellation control/negative proving that the actual nested JVM disappears before lane return.

No additional source/API/test/fragment changes were made for these options. Current221525d checkpoint remains undelivered with claim retained for scope review.


### 2026-09-06T09:12:48Z ramirosalas
APPROVED CUSTODY INTEGRATION SCOPE REVISION 2026-09-06

This append-only revision is narrowly limited to the architecture-dependent AC8 custody integration. Preserve all earlier ACs, failed records, healthy GREEN claim/status and the 96-case frozen pilot. Architecture approval is not native evidence, and this story remains undelivered.

The reviewed contract is delivered by MAC-l7m0 as docs/test-assurance-contract.md (575-line public projection SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8). Required new producer chain: MAC-qlw2 broker/guardians -> MAC-cn7q processcontrol/formal/runtimeclosure attachment -> this lane integration. This story now depends on MAC-cn7q; no cycle runs back from either custody producer to this lane.

CURRENT SUPPLEMENTAL OWNERSHIP / PRODUCES:
- scripts/integration-lane/main.go -> integrate approved processscope.Open/Child/Close and verified ServeInternal before lane parsing; use actual authenticated custody throughout provisioning and suite execution.
- scripts/integration-lane/custody_integration_test.go -> NEW separately reviewed supplemental native custody cases, not edits to frozen main_test.go.
- testdata/integration-lanes/custody.json -> NEW supplemental exact-case fragment, preserving original union and source/fixture/config identities.
Original remaining CI/preflight/lane ownership stays intact. Do not edit the seven frozen v2 files at e55238223961fec922896c361d8af6cafc454a8e: scripts/integration-lane/main_test.go, testdata/integration-lanes/schema.json, pilot.json, runtime-pins.json, pilot.integration.test.mjs, CONTRACT.md, cmd/machinery/integration_lane_test.go (directory prefixes as recorded in the original inventory). Preserve prior v1 history and all 96 named leaves, originally 39 PASS / 57 intended FAIL / 0 SKIP. No prior test rewrite is authorized by new supplemental RED.

CONSUMES:
- MAC-cn7q: internal/processcontrol/scope.go
  spec: Run(ctx context.Context,cmd *exec.Cmd) error compatibility; WithScope(ctx,scope), AttachScope(cmd,scope), ExitStatus(err), actual formal/runtimeclosure attachments AFTER environment sanitization.
- MAC-qlw2: internal/processscope/scope.go
  spec: Open(context.Context,Options)(Scope,error); Scope.Run(context.Context,Command,Streams)(Result,error), Child(context.Context)(Scope,error), Close(context.Context)(CleanupReport,error); ServeInternal(args []string,io InternalIO)(handled bool,exitCode int).
- MAC-l7m0: docs/test-assurance-contract.md
  schema: Section8 exact custody protocol and aggregate deadline/cleanup contract.

ADDITIONAL AC8 PROOF OBLIGATIONS:
A8a. Authenticate native broker/guardian custody through the actual lane -> provisioning/helper -> formal -> pinned Java/TLC path and lane -> real suite/meta test -> nested pinned JVM path. Observe an actually active nested JVM BEFORE cancellation in both paths on Linux amd64 and Darwin arm64; source analysis, fake Java or an outer wrapper alone is insufficient.
A8b. New supplemental RED inventories/fixtures/config are reviewed and frozen separately. Show expected assertion failures and passing safe controls for early intermediate exit, interruption, timeout, overflow, closed/forged/stale scope and cleanup failure. Demonstrate terminal group kill before identity guardian reaping; exact ownership/capability lifecycle, no foreign PID/container cleanup and no residual owned process.
A8c. All preparation/probes/suite launch/parse/cleanup calls use remaining aggregate deadlines and one shared cleanup grace; enumerate actual process-producing call sites. Missing/malformed attachment is fail-closed, never background fallback. Runtime validation is preserved after environment sanitation.
A8d. Final same-source native lane proof includes unchanged original 96 leaves PLUS separately named supplemental custody leaves, no skip/empty/lost/duplicate cases and no runtime leaks. Existing cold-cache residual and standalone Node/full-lane pending evidence remain pending until actually rerun; do not treat the approved architecture as satisfying them.
A8e. Four-language runtime catalog/fragments and compatible contributor schema evolution are owned by MAC-bz1y after this delivery; language adapters and their conformance are separate producers. Final-Ga GateExecution/runtimeclosure.Git/accept.go is owned by MAC-pe9v, NOT quietly added to this story's frozen pilot or source scope.

DIFF BUDGET ADDENDUM: this lane consumer adds at most ~3 owned supplemental files / 650 changed LOC beyond its existing measured scope; investigate overruns. Broker/attachment implementation is budgeted in its own upstream stories. No heavy preflight here, no installed replacement, no remote mutation or unrelated user process/container cleanup.
HOLD: independent Anchor backlog review and accepted upstream contract/custody delivery required before the root resumes this architecture-dependent GREEN work.

## nd_contract
status: in_progress

### evidence
- Approved architecture backlog integration only; original GREEN checkpoint 221525d7baf7a565ece9972d09bf916132e68050 remains undelivered.
- Original frozen pilot and prior failed histories preserved; no source/test edit or native execution by Sr PM.

### proof
- [ ] Existing AC1-AC8 and pending same-source 96-case lane evidence remain required.
- [ ] A8a-A8e: actual new supplemental custody proof pending.


### 2026-09-06T09:16:35Z ramirosalas
CANONICAL SUPPLEMENTAL OUTPUT MAP 2026-09-06
PRODUCES:
- scripts/integration-lane/main.go -> exact bounded current ownership in preceding approved scope revision; no additional implementation authority
- scripts/integration-lane/custody_integration_test.go -> exact bounded current ownership in preceding approved scope revision; no additional implementation authority
- testdata/integration-lanes/custody.json -> exact bounded current ownership in preceding approved scope revision; no additional implementation authority

## nd_contract
status: in_progress

### evidence
- Parser-supported output declaration only; earlier current AC/proof map and healthy claim preserved.

### proof
- [ ] Every existing and supplemental current AC remains required; no execution proof claimed.

### 2026-09-06T09:48:04Z ramirosalas
ROUND-1 RULE 1 CONSUMER MAPPING (SUPPLEMENTAL ONLY)
Current ownership/claim/frozen96pilot and prior custody ACs remain unchanged. The existing dependency MAC-cn7q now explicitly produces the complete actual formal chain, including internal/formal/process.go openFormalJava, internal/formal/alloy.go runAlloy and cmd/machinery/stubs.go newVerifyFormalCmd. This is required producer coverage, not a scope bypass or duplicate probe.
CONSUMES:
- MAC-cn7q: internal/formal/process.go
  source: actual identity probe with inherited owner context/scope attached after runtime environment sanitation.
- MAC-cn7q: internal/formal/alloy.go
  source: actual Alloy probe plus separate JVM execution with SAME verified context/scope and preserved Java/JAR identity.
- MAC-cn7q: cmd/machinery/stubs.go
  source: real command-to-formal context/scope handoff, preserving ordinary command compatibility.
The already owned testdata/integration-lanes/custody.json MUST include the exact separately reviewed supplemental processscope, formal TLC/Alloy/probe and actual command-chain cases from these producers alongside its lane-level cases. Freeze that exact source/test/fixture/config union before supplemental RED; no omitted Alloy/probe branch, native-only fake script, missing leaf, skip or orphan process may pass. Both native platforms still owed; original seven files/96leaves remain byte-for-byte protected. No extra source ownership is transferred to hpqp.

## nd_contract
status: in_progress

### evidence
- Independent round1 findings repaired through supported append-only scope/consumer notes; no architecture/source/test/ref/runtime mutation.
- Exact source surfaces verified against accepted epic7e36f3e7ddcf25565d5d4fe60b328df254eee91d; prior Body/status/labels/history preserved.
- Independent Anchor ROUND2 and canonical-document acceptance remain prerequisites, not implementation evidence.

### proof
- [ ] All current story ACs, strengthened ownership/current-judgment requirements and protected frozen proof remain required.
- [ ] Independent review/native execution/final acceptance pending.

### 2026-09-06T12:45:56Z ramirosalas
ACCEPTED PUBLIC CUSTODY CONTRACT / AUTHORITATIVE CURRENT 96-OBLIGATION MIGRATION SCOPE — 2026-09-06

This true-EOF amendment selects the approved FULL prospective current-revision strategy. It supersedes only the earlier supplemental-only three-file/650-LOC addendum and, FOR THE FUTURE CURRENT REVISION ONLY, the prior absolute seven-file unchanged rule—subject to the independent before-edit gates below. It preserves every historical Body byte, failed/approved record, original seven-file Git identity, raw 96-obligation assertion/event record, current claim, held GREEN source and all stronger AC1–AC10/AC8 duties. It is not test-edit authorization, RED approval, source authorization, native proof, delivery or acceptance.

ACCEPTED PRODUCER GROUNDING
- MAC-p9wm is closed/accepted. Epic `2a73454d5f133a7b5fb4db0346232fd389810d28` contains native contract blob `1c1581d1aec324d979593613b44976ed0007c45b` / SHA256 `bee289e553809b550b1f5cedf73415f600379ab54830a11733ba0dc7caaaba2a` and companion blob `acce64fee8db5a7565e8fa33422334de125e913f` / SHA256 `171ebaa3e992b60bd90f1fc280818a125689390f8440b3a9a7b2c97e7f8b747d`.
- Independent acceptance report SHA256 `19c0c4be6b4eb4b7035db3e9e576352ad1cfd3b24f414e287be7d8dfded38f8e`; approved V4 proposal/review SHA256 `d56d0104e965cace70e89b3be383ff3f51377c33ea8227c86833ef731be331be` / `dfa09b70d599a83d8cde10ab31d19fe5debd1eadf4c19f86c3aea79c0a9ac1e3`.

PRODUCES:
- scripts/integration-lane/main.go -> current required-lane implementation with explicit `--docker-endpoint`, once-only owned endpoint preparation input, authenticated pre-parser ServeInternal handling, root/child scope propagation, captured DockerRuntime after pinned provisioning and before native selection/replay, closed contributor service calls and exact execution/accounting/cleanup.
- scripts/integration-lane/main_test.go -> prospectively revised current-revision native runner/inventory/accounting/wiring tests, only after exact before-edit authorization; historical bytes remain immutable evidence.
- testdata/integration-lanes/schema.json -> prospectively versioned compatible closed current suite-fragment schema, preserving v1 history.
- testdata/integration-lanes/pilot.json -> current required pilot registry with every old obligation mapped and executed.
- testdata/integration-lanes/runtime-pins.json -> frozen v1 pin history plus only an independently reviewed compatible current revision where necessary; never fake-add local daemon identity/hash.
- testdata/integration-lanes/pilot.integration.test.mjs -> current real Node pilot obligations, preserving historical identity/evidence.
- testdata/integration-lanes/CONTRACT.md -> exact current contributor execution/migration contract; this is the held-tree thirteenth original owned path.
- cmd/machinery/integration_lane_test.go -> current real OCI/formal/Node/full-path/generated-nested-Go native tests and independent foreign-owner harness controls.
- scripts/preflight.sh -> same mandatory owned `python-version` custody leaf after owned provisioning and before preserved formal/C4/checker gates; remove the direct unowned step-15 `docker run --rm` smoke only in the authorized current revision.
- .github/workflows/ci.yml -> mandatory Linux amd64 required-lane execution and service-free native separation.
- .github/workflows/formal.yml -> preserved formal provisioning/execution ordering through the shared current lane.
- .github/workflows/nightly.yml -> explicit provisioned versus service-free execution and no optional bypass.
- Makefile -> exact shared required-lane entrypoint.
- scripts/integration-lane/custody_integration_test.go -> NEW current native custody cases across lane/provisioning/helper/formal/suite/meta-test and conservative daemon lifecycle.
- testdata/integration-lanes/custody.json -> NEW closed current custody fragment registering processscope, TLC, Alloy, probe, normal command, contributor service and lane-level leaves.
- testdata/integration-lanes/custody-migration.json -> NEW exact 96-row historical-to-current obligation mapping with schema `machinery.integration.custody-migration/v1`, `old_source` `e55238223961fec922896c361d8af6cafc454a8e`, `new_revision` `custody-v3`.

CONSUMES:
- MAC-qlw2: internal/processscope/scope.go
  spec: error-first InheritedInternalIO(context.Context) (InternalIO,bool,error); Open(context.Context,Options) (Scope,error); Scope.Run/Child/Attach/Close; ServeInternal([]string,InternalIO) (bool,int); exact Limits/Diagnostic and authenticated owner/cumulative cleanup semantics.
- MAC-qlw2: internal/processscope/contributor_docker.go
  spec: CaptureDockerRuntime(context.Context,Scope,DockerRuntimeRequest) (*DockerRuntime,error); InheritedDockerRuntime(context.Context,Scope,string) (*DockerRuntime,error); DockerRuntime.Descriptor/Validate/Close; OpenContributorDocker(context.Context,Scope,*DockerRuntime) (*ContributorDocker,error); ContributorDocker.Run/Close; InspectDockerContainer exact-ID read-only observation.
- MAC-cn7q: internal/processcontrol/scope.go
  spec: Run(context.Context,*exec.Cmd) error compatibility; WithScope(ctx,scope), AttachScope(cmd,scope), ExitStatus(err), with actual formal/runtimeclosure attachment after sanitation.
- MAC-cn7q: internal/formal/process.go
  source: actual openFormalJava identity probe and runBoundedProcess use inherited owner context/scope.
- MAC-cn7q: internal/formal/alloy.go
  source: actual Alloy probe and separate JVM launch use the SAME verified owner context/scope after sanitation.
- MAC-cn7q: cmd/machinery/stubs.go
  source: real newVerifyFormalCmd command-to-formal context/scope handoff.
- MAC-p9wm: docs/native-custody-contract.md
  schema: exact accepted endpoint/runtime/contributor/observation/unresolved-create/live-first/current-migration contract at SHA256 bee289e553809b550b1f5cedf73415f600379ab54830a11733ba0dc7caaaba2a.
- MAC-p9wm: docs/test-assurance-contract.md
  source: exact accepted contributor-lane, cumulative-budget, formal/nested-process and historical migration obligations at SHA256 171ebaa3e992b60bd90f1fc280818a125689390f8440b3a9a7b2c97e7f8b747d.
- (Existing source): scripts/shellcheck-files.txt
  source: READ-ONLY closed executable-shell inventory used for mandatory validation; preserve its bytes exactly. If an actual later registry edit is necessary, stop for separate scope/budget investigation rather than silently creating a seventeenth write path.

CURRENT PROSPECTIVE ACCEPTANCE / MIGRATION
1. Scope is exactly SIXTEEN total WRITE paths: the 13 actual changed paths in exact Git diff `497419ab4512fcff765cd5feb27aed4c67b5608d..221525d7baf7a565ece9972d09bf916132e68050` listed above, plus `scripts/integration-lane/custody_integration_test.go`, `testdata/integration-lanes/custody.json` and `testdata/integration-lanes/custody-migration.json`. This corrects the earlier measured-versus-owned count: `scripts/shellcheck-files.txt` was previously declared owned but is unchanged in the held 13-path delta and is now a byte-identical READ-ONLY consumed validation input, not a seventeenth write path. Do not drop the Node pilot, CONTRACT.md or any old outcome. No `internal/processscope`, `internal/processcontrol`, `internal/formal` or `internal/runtimeclosure` source ownership transfers here.
2. Historical v2 RED source `e55238223961fec922896c361d8af6cafc454a8e`, original v1 history, the seven old frozen byte identities, all old failed/passing raw events and exactly 96 unique old identities/assertion obligations remain immutable. Held unaccepted GREEN `221525d7baf7a565ece9972d09bf916132e68050` on original base `497419ab4512fcff765cd5feb27aed4c67b5608d` remains preserved and undelivered.
3. Before ANY edit to the seven formerly frozen current paths, Sr PM plus an independent PM must review the exact proposed source/test/helper/fixture/config/dependency-lock inventory, every necessary byte edit and authorized test-edit commit markers. Before first RED/test authoring, record the actual RED author and a DIFFERENT actual calibration author; select the independent PM before review and the separate production GREEN author only after RED approval and before GREEN dispatch. Pre-inventory planning may precede those later role selections, but no source/test write precedes its applicable before-edit gate. Authorization is prospective and narrow. It does not rewrite history, waive a failed audit, bless held GREEN or allow test repair during GREEN.
4. `custody-migration.json` contains exactly 96 unique old rows and no omission, extra or duplicate. Each row maps exact old source file, native test identity and assertion IDs to current file/test/assertion IDs plus bounded reason. Every prior outcome is preserved or strengthened and every mapped current leaf is registered and actually executes. No status/count-only mapping, historical-only selection, omitted old assertion, invented replacement or inflation by new custody leaves is accepted.
5. Current replay executes ALL 96 mapped obligations plus separately inventoried new custody leaves on the same candidate on native Linux amd64 AND Darwin arm64, with no skip/cache/empty/lost/duplicate/partial/fabricated-summary success. Preserve the known historical checkpoint truth: 64 native leaves passed; tagged 31 were 30 pass/1 cold-cache alias fail with the corrected cold-cache leaf separately passing; standalone Node and full current-lane replay remain owed. These are history, not current acceptance.
6. Replace direct Docker ownership seams only through real authenticated custody: integrationOwnContainer, generated nested-Go TestPilot daemon work, the independently owned `sleep-900` foreign sentinel and preflight step-15 Python-version smoke. The replacement Python-version leaf remains mandatory after owned provisioning and before every preserved checker gate. Generated/native tests retain actual assertion, timeout, output and accounting semantics; source/receipt-only checks do not replace execution.
7. `--docker-endpoint` is an explicit exact transport. Preflight resolves its local Unix socket once during owned bounded preparation unless explicitly supplied, freezes/passes the same value to lane and checker, and no capture/run/replay/cleanup rereads ambient Docker context. Pinned image provisioning occurs under registered custody first; runtime capture follows and precedes selection/replay. Unsupported remote endpoints fail. The generated per-invocation 0400 Docker descriptor is external and live-bound; it is not committed and no fake daemon hash is added to frozen runtime-pins v1.
8. Actual helper, TLC, Alloy, probe, native suite and recursive full-path/meta-test processes inherit the same owner scope AFTER environment sanitation. One cumulative wall deadline and fixed owner ceiling cover all work; one shared cleanup grace covers cleanup-only helpers registered under the original grace. No Background/unscoped fallback, per-phase renewal, post-release runtime probe or unresolved create/start false-clean is allowed.
9. Live-first tests independently observe the exact real container/JVM/helper before the actual failure/cancel/overflow/signal/owner-loss trigger; exact owned absence precedes return while separate foreign owner and caller sentinel survive. Historical PID/PGID/label/CID text is never cleanup authority. Same-assertion prospective unsafe/safe variants freeze before replay; reference safe success never substitutes for production GREEN.
10. DIFF BUDGET: forecast 4,500–6,500 changed LOC TOTAL from original base `497419ab4512fcff765cd5feb27aed4c67b5608d`, INCLUDING the held 3,015 changed LOC. This is not 3,015 plus another ceiling. Investigate overrun explicitly and never trim the 96 mapped obligations or new custody/native proofs to fit.
11. MAC-hpqp remains the lane/current-revision producer for MAC-yhg5. Product checker implementation is not a prerequisite for the 96-obligation pilot beyond the existing one-way `MAC-hpqp -> MAC-yhg5`; do not add a reverse edge or cycle. All original stronger AC1–AC10, no-preflight-until-final-gate, no remote/install/unrelated cleanup and standalone Machinery rules remain.

CURRENT HOLD / OWNER
- Owner remains `dev-MAC-hpqp`; status remains in_progress. Upstream attachment implementation from MAC-qlw2/MAC-cn7q and the exact before-edit independent review are required before resuming the architecture-dependent current revision. Accepted MAC-p9wm is satisfied context, not a blocker.

## nd_contract
status: in_progress

### evidence
- Exact accepted producer, original RED, held GREEN, historical counts, sixteen paths and current migration schema are fixed above.
- Canonical repair is tracker-only; no source/test/ref/worktree/runtime/native/preflight/remote/install action occurred.

### proof
- [x] Full prospective current 96-obligation migration strategy and sixteen-path budget are bounded.
- [x] Historical frozen bytes/evidence and one-way dependency chain are preserved.
- [ ] Exact before-edit inventory, staged actual-role selection, authorized test-edit markers and new RED approval remain pending.
- [ ] All 96 current mappings plus new custody leaves on the same Linux amd64/Darwin arm64 candidate remain pending.

### 2026-09-06T12:55:30Z ramirosalas
PARSER ANNOTATION / EDITOR-NEWLINE RECOVERY AUDIT — 2026-09-06

This true-EOF audit records a formatting-only correction to the newly authored custody-scope amendment. It changes no ownership, acceptance criterion, test authority, status, label, dependency, evidence claim or product semantics.

The first comment-add attempt was made from the external draft directory and failed before vault resolution (`could not find local .vault`); no story mutation occurred. After the corrected four scope comments were added from the pinned repository CWD, scoped lint correctly exposed one parser error: the new hpqp line `- Existing source: scripts/shellcheck-files.txt` was treated as an issue reference. A later supported `pvg nd edit MAC-hpqp` used a guarded apply_patch editor to add only `(` and `)` around `Existing source`. The editor deliberately returned exit 69 because apply_patch also removed the then-terminal LF: it observed one corrected target, zero malformed targets, +1 net byte rather than +2, and a normalized whole-file mismatch. No retry or alternative editor mutation occurred.

Exact recovery evidence before this audit append:
- Frozen failed-lint Body: 110,437 bytes, SHA256 `419e6a79ba74b88032f1ff75b7d02b86cae009f84dea2f9b3e0ae227ec0d5370`.
- Intended stream after inserting only the two parentheses and retaining the LF: 110,439 bytes, SHA256 `432d34f86573148e3a1f9ed911c8ad40acbcb74874d8131866aa4ecd0af6fc68`.
- Actual post-editor stream before this append: 110,438 bytes, SHA256 `5f498bd4f80202a76a16e430f600d24be97d4dd0509d88f78e0b988275654ff1`.
- Exact comparison proved `actual post-editor Body + one LF == intended stream`; there was no other byte difference. The ordinary nd comment framing supplies that missing LF as its first appended byte, after which this audit begins.
- Frozen external evidence: `/tmp/MAC-` + `custody-canonical.iVvkhI/failed-current-body.md` mode 0444 / SHA256 `419e6a79ba74b88032f1ff75b7d02b86cae009f84dea2f9b3e0ae227ec0d5370`; `/tmp/MAC-` + `custody-canonical.iVvkhI/failed-scoped-lint.json` mode 0444 / SHA256 `6108fe2788a7c2a7b68d458a6beb77022d9ddfc57002f71875a3da01c927a429`; guarded editor SHA256 `28374fa86b093af8e40966534e290e5988f73f7c5f3d2d235240230107906701`.

The authoritative parser-safe line is exactly `- (Existing source): scripts/shellcheck-files.txt` with its existing indented `source:` signature. Parentheses are the documented non-issue annotation. A new later CONSUMES entry could not override the malformed historical entry because the linter accumulates and checks every historical entry; therefore the two-character annotation had to repair this newly authored occurrence in place. `scripts/shellcheck-files.txt` remains byte-identical READ-ONLY input, never a seventeenth hpqp write path. The current exact scope remains 13 held changed paths plus 3 approved new paths = 16 WRITE paths.

## nd_contract
status: in_progress

### evidence
- Formatting recovery only; original story history and the complete new scope amendment remain preserved.
- No product source/test/ref/worktree/runtime/native/preflight/remote/install mutation or evidence claim occurred.

### proof
- [x] Parser-safe existing-source annotation records the same read-only shell inventory semantics.
- [x] The failed wrong-CWD command, lint error and guarded editor exit 69 remain explicit history.
- [ ] Exact before-edit inventory/roles/authorization, new RED, implementation and Linux amd64/Darwin arm64 current replay remain pending.

### 2026-09-06T13:00:55Z ramirosalas
EXTERNAL-PATH DISPLAY SPLIT / FINAL PARSER AUDIT — 2026-09-06

This true-EOF audit records only a display-format annotation in the preceding parser-recovery audit. It changes no filesystem path, artifact hash, ownership, acceptance criterion, test authority, status, label, dependency, evidence claim or product semantics.

Scoped lint accumulated every historical reference and interpreted the contiguous external directory display as an issue token even though it was inside inline code. Because a later entry cannot override an earlier malformed reference, the two newly authored path displays on the single frozen-evidence line were annotated in place. Each occurrence now renders as two adjacent literal components—`/tmp/MAC-` + `custody-canonical.iVvkhI/failed-current-body.md` and `/tmp/MAC-` + `custody-canonical.iVvkhI/failed-scoped-lint.json`. Concatenating the two code spans around ` + ` yields the exact original unchanged artifact path; no alias, fake issue, new file path or semantic substitution is claimed.

Exact guarded-edit evidence:
- Before display annotation: 113,956 Body bytes, SHA256 `a162d5e9892dabd6bbfa949a7c3e1c57173e937643e8fd73fece035bf4caf550`.
- Intended result after exactly two five-character insertions with the terminal LF retained: 113,966 bytes, SHA256 `ba7e23380d2a40ec42c966619a2156443d59cb568ea61d7ac4012ffff0ba8b04`.
- Supported `pvg nd edit MAC-hpqp` guarded editor SHA256 `a9d34cf489d3f68996aa6052779f9d1d1de7d273e77b22af20dd9522af29c860` verified the exact issue/body, exactly two original occurrences on one line, zero preexisting split occurrences, and every post-edit byte. It completed in the authorized `expected-minus-terminal-lf` mode: 113,965 bytes, SHA256 `6c03e6e9951c7824b894417020e6b90112a493dfe13b3863a479aed1e7e6d691`.
- Exact comparison proved `post-editor Body + one LF == intended annotated Body`; the ordinary nd comment framing supplies that LF as its first appended byte before this audit begins.
- Frozen pre-display Body artifact SHA256 `a162d5e9892dabd6bbfa949a7c3e1c57173e937643e8fd73fece035bf4caf550`; frozen stale-ref lint artifact SHA256 `23b0c550962f9be70e61336dd81664e0c85a1457b2c5a97964d170e8419d4014`; both are mode 0444 in the same external custody-scope evidence directory.

The prior wrong-CWD no-mutation failure, initial existing-source parser failure, guarded editor exit 69/newline recovery, and stale-reference false positive all remain explicit history. The exact hpqp scope remains 13 held changed paths plus 3 approved new paths = 16 WRITE paths; `scripts/shellcheck-files.txt` remains byte-identical READ-ONLY input. No audit exception, historical rewrite, native evidence, source/test permission or scope change follows.

## nd_contract
status: in_progress

### evidence
- Display-only parser annotation used the supported nd editor with exact byte/count guards; normal comment framing restores the known terminal LF.
- No product source/test/ref/worktree/runtime/native/preflight/remote/install mutation occurred.

### proof
- [x] Both external artifact displays remain faithful by literal component concatenation without an issue-like contiguous token.
- [x] All earlier failure and preservation evidence remains explicit.
- [ ] Exact before-edit inventory/roles/authorization, new RED, implementation and Linux amd64/Darwin arm64 current replay remain pending.


### 2026-09-07T04:45:24Z ramirosalas
ACCEPTED 2026-09-06 — resumed 221525d + epic sync + AC8 custody resolution (46f6c85 RED / 3f4517e GREEN / ef6af49 lint) = tip ef6af49. Architecture hold resolved by coordinator decision recorded in-story: custody = processscope via processcontrol.WithScope/AttachScope (accepted qlw2+cn7q chain), no wrapper hack; contributor baseline unchanged (required Linux lane + macOS Docker Desktop); uncatchable owner death/hostile escape honestly cleanup-failed. Verification: native 17/17; tagged 6 roots/31 subleaves; Node pilot; FULL --lane required green twice (3 suites 1+5+1 tests, custody {passed, root, 11 jobs}, 5 runtimes pinned+verified); untagged separation confirmed; actionlint/vet/golangci/shellcheck clean; no owned residue; dagger-engine untouched. One [test-edit-authorized] native-test revision justified in-record. Shipped-ceiling findings (40s result-wait, 4 jobs/broker, 30s shared grace, macOS 8KB AF_UNIX) measured and designed around via accepted API only. RESIDUAL: hosted Linux CI declared not observed (no remote mutation; bundle /tmp/hpqp-linux-inputs). Coordinator merged; lane re-run green on epic (3 suites passed). Record: .git/machinery-evidence-20260906.TEFZ7D/hpqp-record.md
