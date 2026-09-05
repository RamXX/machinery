---
id: MAC-hpqp
title: "Execute every required infrastructure test deterministically"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd, delivered]
parent: MAC-ui8a
created_at: 2026-09-05T19:44:27Z
created_by: ramirosalas
updated_at: 2026-09-05T20:27:03Z
content_hash: "sha256:e287c511354ffe4c28b0f139b06362103cbd15989c5f921ab5a5ee9e7ebd6019"
blocks: [MAC-hlae, MAC-yhg5, MAC-2n83, MAC-hwdb, MAC-vx24, MAC-gcrr, MAC-ou97, MAC-hy71]
assignee: dev-MAC-hpqp
---

## Description
### USER INTENT
The maintainer can run one mandatory integration lane locally and in CI and know every required real-runtime safety test actually executed with correct prerequisites and no leaked resources.

### Context (Embedded)
Anchor round 1 found a cross-story execution gap: new Docker/Java/Node cases were assigned ordinary package suites, but macOS native CI lacks Docker and preflight provisions checker image after its race suite. Required tests need explicit lane selection, not skip-if-missing. This is repository contributor infrastructure, standalone Go/Node/Java/Docker only; it never requires Paivot.

### Ownership
Own scripts/integration-lane/main.go, scripts/integration-lane/main_test.go, testdata/integration-lanes/schema.json, testdata/integration-lanes/pilot.json, testdata/integration-lanes/runtime-pins.json, cmd/machinery/integration_lane_test.go, scripts/preflight.sh, .github/workflows/ci.yml, .github/workflows/formal.yml, .github/workflows/nightly.yml, Makefile, scripts/shellcheck-files.txt. Preserve concurrent edits. Runtime stories own separate lane fragments and their tests, never the common inventory engine. The release workflow story modifies shared workflows afterward.

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

### Story Acceptance Criteria
1. A required named lane discovers and validates every committed suite fragment using a closed schema and matches exact selected native test identities to its registered source files. Unknown/duplicate/orphan fragments, unregistered integration test files, invalid paths and empty lane fail. Every fragment names actual runnable cases, never future placeholders.
2. Ordinary go test ./... native lanes deliberately exclude build-tagged runtime tests and do not probe/skip absent services. Required lane explicitly selects them and fails if Docker/Node/Java prerequisites are unavailable. Runtime tests cannot become dormant via env flags, t.Skip, skipped cases, xfail, filtering or empty package selection.
3. Provision and verify immutable OCI digest/platform, pinned formal/Java closure and supported Node runtime BEFORE their tests. A cold-cache positive provisions then executes; absent/offline/unavailable runtime and wrong digest/platform reject with clear diagnostics. Existing pinned example image may be reused; never trust a mutable tag.
4. Capture real Go JSON and Node native structured execution output under bounded processes and verify every registered test started, terminated and passed exactly as required. Missing/incomplete/duplicate/skipped tests, cached or malformed/truncated events, extra unexpected failures and fabricated aggregate-only summaries block. Success records actual nonzero execution counts per suite and runtime.
5. Required Linux hosted CI and final local preflight invoke the same lane/inventory. macOS native portability remains service-free while local macOS preflight with Docker Desktop executes the same required safety coverage using pinned platform. Job status must be available for release required-check policy; no remote mutation occurs during implementation.
6. Eliminate preflight ordering hazard: service-backed tests are never executed in the early ordinary race suite before image/runtime provisioning. Preserve existing formal/C4/checker gates, full-history needs and cheapest-first safe ordering; no weakening via SKIP_PREFLIGHT or optional missing-runtime fallback.
7. Initial story is independently executable: include a nonempty actual bounded OCI success/termination pilot plus actual formal/Node invocation where classified as required. Later stories add their own fragment only alongside executable tests. Shared runner discovers their union deterministically without each touching the root manifest.
8. Teardown validates all owned containers/processes/temp roots are removed after success, failed assertion, timeout, cancellation and provisioning error, or fails with actionable cleanup error. Never remove unrelated containers or user files; preserve existing dagger-engine-v0.21.9.
9. RED tests demonstrate existing missing-lane/incorrect-order or wrong-accounting behavior through genuine runtime assertions with passing controls; freeze tests/config/fixtures after approval. No compile/infra-only RED. Targeted lane/provisioning/accounting tests and actionlint pass; do NOT run full scripts/preflight.sh before final epic completion.
10. Real full-path no-mock tests cover provisioned success, fresh cache, missing runtime, zero execution, skipped required test, wrong pin, partial output and leaked owned resource. Run native suites without services to prove separation and required lane with services to prove coverage. Machine-readable report identifies exactly what ran; it cannot stand in for native events.

### Testing Requirements
Hard TDD explicitly authorized. Unit plus Integration tests: MANDATORY (no mocks), actual bounded processes and real Docker daemon; no skip-if-missing. Unit parser fixtures may test malformed events but cannot replace real execution proof. Targeted go test ./scripts/integration-lane and go run ./scripts/integration-lane --lane required; actionlint and native ordinary-lane selection checks. Full heavy preflight ONLY at end.
No GitHub push/sync/mutation, no installed binary/plugin/skill/agent replacement. Isolated builds/homes only. No Paivot product dependency.

### OUT OF SCOPE
- Individual saga/OCI/recovery/OpenCode/process/capstone behavior: sibling stories own their real tests and inventory fragments, consuming this lane.
- Final full-preflight execution belongs to epic completion after all siblings.

### DIFF BUDGET
~10-13 files, under 1800 changed LOC; investigate overrun rather than weaken execution guarantees.

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

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-hlae]], [[MAC-yhg5]], [[MAC-2n83]], [[MAC-hwdb]], [[MAC-vx24]], [[MAC-gcrr]], [[MAC-ou97]], [[MAC-hy71]]

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

