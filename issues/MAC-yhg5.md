---
id: MAC-yhg5
title: "Own checker container lifetime deterministically"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:27Z
created_by: ramirosalas
updated_at: 2026-09-07T04:45:47Z
content_hash: "sha256:8f7ee01f9796b49d403e78af4e8833c3baaf502eb42baf7c933d73e577921ee7"
blocks: [MAC-gcrr, MAC-ou97]
was_blocked_by: [MAC-hpqp]
follows: [MAC-hpqp]
assignee: dev-MAC-yhg5
---

## Description
## USER INTENT
Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.

## Context (Embedded)
Assessment F4 actual processcontrol timeout killed docker CLI while daemon container remained running. runCheckerOCI uses docker run --rm without container ID capture/removal or cpu/memory/pids budgets. Existing network/read-only/capability restrictions must remain.

## Ownership
Own only these paths and directly associated tests: cmd/machinery/verify_checkers.go, cmd/machinery/checker_oci_lifecycle_test.go. You are not alone in this codebase; preserve other edits and coordinate any shared-file changes with dispatcher.

## Boundary Map
PRODUCES:
- cmd/machinery/verify_checkers.go -> hardened behavior and regression proof
- cmd/machinery/checker_oci_lifecycle_test.go -> hardened behavior and regression proof
CONSUMES:
- Existing Machinery source interfaces.
  spec: runCheckerOCI(engineArgs []string, image, platform string, checkerArgs []string, runtimeDigest string, timeout time.Duration, workDir string) (string, error)

### Story Acceptance Criteria
1. Every OCI checker run owns an unambiguous container identity; timeout, cancellation, output-limit breach and startup/runtime failures perform bounded daemon-side cleanup before reporting completion, or explicitly report cleanup failure. Never remove unrelated containers.
2. Enforce deterministic finite memory, CPU and PID budgets with bounded defaults and closed validated overrides if exposed. Preserve network=none, read-only root, capability/no-new-privileges restrictions and exact runtime closure binding.
3. Handle cancellation before/after container creation, missing ID, failed engine cleanup, already-exited container, SIGTERM-ignoring workload, huge output and repeated runs without orphan accumulation.
4. A real pinned local OCI image executes a success case and a timeout case; after each return actual docker inspect/list proves no owned running container remains. Use unique disposable names and clean only test-owned resources.
5. Target tests include resource exhaustion/invalid budgets and CLI propagation; no fake executable result substitutes for real daemon lifecycle proof. Docker missing/unavailable fails required integration, never skip.

## Testing Requirements
- Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
- Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
- Commands: go test ./cmd/machinery -run 'CheckerOCI|CheckerLifecycle'; required real-daemon full verify-checkers integration with an immutable synthetic checker image and bounded workloads. Preserve preexisting dagger container untouched.
- No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.

## OUT OF SCOPE
- Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
- Global heavy preflight, main merge and local release binary belong to final epic gate.

## DIFF BUDGET
- ~3-5 files, under 900 changed LOC; material overrun requires PM investigation, not weakened requirements.

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
ANCHOR ROUND-1 AUTHORITATIVE EXECUTION-LANE REPAIR
General rule: required runtime tests need deterministic provisioning, explicit closed inventory, actual native execution accounting and teardown. Missing infrastructure must fail the REQUIRED lane, not be silently skipped. Ordinary native suites may explicitly exclude registered service-backed tests using a dedicated build tag.
Classification: Real daemon lifecycle/resource/cleanup scenarios require pinned OCI image and verified platform before invocation.
All real daemon cases are explicitly build-tagged/inventoried. Ordinary command tests may test pure argv/schema logic without Docker, but cannot substitute for lane.
PRODUCES:
- testdata/integration-lanes/oci.json -> this story's closed suite fragment, with exact source/test IDs, runtime/pin requirements and bounded execution configuration
- cmd/machinery/checker_oci_lifecycle_test.go -> actual named runtime cases registered in the fragment
CONSUMES:
- MAC-hpqp: testdata/integration-lanes/schema.json
  schema: Closed versioned native-runner suite fragment with exact source/test identities, runtime requirements and bounded command selection.
- MAC-hpqp: scripts/integration-lane/main.go
  endpoint: go run ./scripts/integration-lane --lane required (Makefile test-integration invokes same entrypoint).
Additional acceptance criteria: fragment matches actual test sources both directions; all registered cases actually start/terminate with expected positive/negative outcomes; no cached/skipped/empty/partial/fabricated-summary success; real provisioned positive and missing-runtime/fresh-cache failure diagnostics; no owned container/process leaks. Required local preflight and hosted CI execute the same union. Do not edit shared root inventory; own only this fragment. RED source, fixture, fragment and runner configuration are frozen together after review. Any exact test names introduced in RED must remain registered through GREEN.
No heavy preflight until final gate; no GitHub mutation; no active installation replacement. This note supersedes any earlier command implying service-backed tests execute in unprovisioned ordinary package suites.

## History
- 2026-09-05T19:35:09Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:15Z dep_added: blocks MAC-ou97
- 2026-09-05T19:45:31Z dep_added: blocked_by MAC-hpqp
- 2026-09-07T04:45:25Z dep_removed: was_blocked_by MAC-hpqp
- 2026-09-07T04:45:47Z status: open -> in_progress
- 2026-09-07T04:45:47Z auto-follows: linked to predecessor MAC-hpqp

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-gcrr]], [[MAC-ou97]]
- Was blocked by: [[MAC-hpqp]]
- Follows: [[MAC-hpqp]]

## Comments

### 2026-09-06T12:46:06Z ramirosalas
ACCEPTED PUBLIC CUSTODY CONTRACT / AUTHORITATIVE CURRENT CHECKER SCOPE — 2026-09-06

This true-EOF amendment supersedes the earlier two-path/~900-LOC checker-lifecycle forecast with the approved six-path closed-profile scope. Every prior stronger regression, real-daemon, sandbox, no-skip and independent-acceptance constraint remains. This is prospective backlog scope only—not source/test authorization, RED approval, implementation, runtime/native proof, delivery or acceptance.

ACCEPTED PRODUCER GROUNDING
- MAC-p9wm is closed/accepted. Epic `2a73454d5f133a7b5fb4db0346232fd389810d28` contains native contract blob `1c1581d1aec324d979593613b44976ed0007c45b` / SHA256 `bee289e553809b550b1f5cedf73415f600379ab54830a11733ba0dc7caaaba2a` and companion blob `acce64fee8db5a7565e8fa33422334de125e913f` / SHA256 `171ebaa3e992b60bd90f1fc280818a125689390f8440b3a9a7b2c97e7f8b747d`.
- Independent acceptance report SHA256 `19c0c4be6b4eb4b7035db3e9e576352ad1cfd3b24f414e287be7d8dfded38f8e`; approved V4 proposal/review SHA256 `d56d0104e965cace70e89b3be383ff3f51377c33ea8227c86833ef731be331be` / `dfa09b70d599a83d8cde10ab31d19fe5debd1eadf4c19f86c3aea79c0a9ac1e3`.

PRODUCES:
- cmd/machinery/verify_checkers.go -> real command-context custody and once-only `--docker-endpoint` preparation/transport through verifyCheckersTo, verifyOneChecker, run and committed-evidence replay; closed RunCheckerDocker use replaces direct generic `docker run --rm` on the required path.
- cmd/machinery/checker_oci_lifecycle_test.go -> new exact real checker lifecycle, live-first, output/resource/failure and foreign-survival tests only; existing unrelated/frozen checker tests are not a broad edit grant.
- testdata/integration-lanes/oci.json -> closed exact checker source/test/runtime/pin/leaf inventory consumed by the required lane.
- internal/processscope/checker_docker.go -> closed `RunCheckerDocker(context.Context, Scope, *DockerRuntime, CheckerDockerRequest, Streams, chan<- ContainerObservation) (Result, error)` profile with exact registry, work/input/runtime/config/sandbox and cleanup binding.
- internal/processscope/checker_docker_test.go -> exact profile validation, real daemon lifecycle, run/replay and calibration controls.
- internal/processscope/broker.go -> ONLY the finite checker-profile dispatch/validation arm, modified sequentially AFTER accepted MAC-qlw2 broker work; no base broker redesign or qlw2 test mutation.

CONSUMES:
- MAC-qlw2: internal/processscope/scope.go
  spec: authenticated Scope.Run/Child/Attach/Close, exact cumulative Limits/Result/CleanupReport and fail-closed admission/cleanup behavior.
- MAC-qlw2: internal/processscope/contributor_docker.go
  spec: CaptureDockerRuntime/InheritedDockerRuntime with live Descriptor/Validate/Close plus exact ContainerObservation and read-only InspectDockerContainer semantics reused by the checker profile.
- MAC-hpqp: scripts/integration-lane/main.go
  endpoint: `go run ./scripts/integration-lane --lane required` with explicit `--docker-endpoint`, pinned provisioning, runtime capture before selection/replay and exact native event/accounting/cleanup.
- MAC-hpqp: testdata/integration-lanes/schema.json
  schema: closed compatible current fragment contract that discovers `testdata/integration-lanes/oci.json` and requires every registered leaf to execute.
- MAC-p9wm: docs/native-custody-contract.md
  schema: exact accepted checker API/profile, work-root/input/runtime binding, real context/scope across run and replay, sanitation-before-Attach, once-only endpoint capture and cleanup-before-output/evidence/release at SHA256 bee289e553809b550b1f5cedf73415f600379ab54830a11733ba0dc7caaaba2a.
- MAC-p9wm: docs/test-assurance-contract.md
  source: exact accepted contributor/checker, cumulative-budget, native-matrix and finalization rules at SHA256 171ebaa3e992b60bd90f1fc280818a125689390f8440b3a9a7b2c97e7f8b747d.

CURRENT PROSPECTIVE ACCEPTANCE / TEST WORKFLOW
1. Scope is exactly SIX paths above. `internal/processscope/broker.go` is shared only through the real dependency sequence: MAC-qlw2 owns the base broker first; this story later adds the checker-only arm while preserving all frozen qlw2 tests and existing MAC-yig6/Y regression bytes. No concurrent broker edit is authorized.
2. The real normal checker command uses `cmd.Context()` and the SAME authenticated owner/root/scope and captured DockerRuntime through orchestration, validation, primary run and committed-evidence replay. Deterministic checker environment/sandbox sanitation occurs before Attach. Nil/stale/foreign/cancelled/closed/mismatched/unscoped authority fails before target start; required execution never calls or falls back to a service-free/unscoped wrapper.
3. Own the closed checker request and actual profile only: canonical retained `WorkRoot`/private WorkPath, read-only inputs topology/digest, registry image/platform, runtime closure, phase `run|verify`, original RunArgs/VerifyArgs, UID/GID, timeout and exact config. Preserve current PiiFlow adapter, canonical token/config/manifest projection, `{design}` rejection, file-only evidence, `/work/evidence.json` and `/work/committed/evidence.json` replay, evidence/trace comparisons and snapshot CheckUnchanged/Release. No arbitrary argv, environment, mounts, engine prefixes, resource flags or dynamic profile is accepted.
4. Preserve exact sandbox: pull=never, network=none, read-only root, cap-drop ALL, no-new-privileges, `/work` writable only from the held work root, `/checker` read-only, fixed `/tmp` tmpfs, fixed sanitized environment, validated user, memory 134217728, NanoCPUs 500000000 and PidsLimit 32 with no overrides. Target result and cleanup remain distinct; exact cleanup/absence precedes successful output, evidence publication, WorkRoot release and runtime release.
5. `--docker-endpoint` accepts only an absolute local socket. An explicit endpoint wins; otherwise existing context resolution occurs exactly once during owned bounded preparation, then freezes the concrete endpoint. Capture/run/replay/cleanup use the same runtime/daemon/private-client binding and never reread ambient Docker context; remote endpoints fail before replay; run/replay never pull.
6. Freeze an exact new leaf/helper/fixture/config/dependency-lock and calibration inventory before RED; preserve existing qlw2 and MAC-yig6/Y tests. Real live-first tests independently observe the exact checker container running before actual PiiFlow assertion failure, genuine container stdout/stderr overflow, timeout, SIGINT/SIGTERM, owner loss, SIGTERM-ignoring workload and resource exhaustion. Cover registration/create/start ambiguity, cancellation races, repeated runs and cleanup failure. Exact owned absence and helper reaping precede return while unrelated process/container controls survive; no fake provision script, no never-started resource and no label/PID history proves success.
7. One cumulative wall deadline and fixed owner ceiling govern run plus replay; one shared cleanup grace governs registered cleanup-only helpers. No Background/unscoped fallback, grace renewal, new work after CLOSING, post-runtime-release probe or unresolved create/start false-clean. Linux amd64 AND Darwin arm64 must run the same candidate and frozen inventory before acceptance.
8. MAC-hpqp’s 96-obligation pilot is NOT product checker gating proof and is not a prerequisite beyond the existing one-way dependency `MAC-hpqp -> MAC-yhg5`. Do not add `MAC-yhg5 -> MAC-hpqp` or any cycle. qlw2/cn7q -> hpqp -> yhg5 remains the execution chain. DIFF BUDGET: exactly 6 files, forecast 2,200–4,200 changed LOC; investigate overrun, never trim required tests. All no-preflight-until-final-gate, no remote/install/unrelated cleanup and standalone Machinery constraints remain.

CURRENT HOLD / OWNER
- Story remains open/new and unassigned, blocked only by the existing MAC-hpqp chain. Before first RED/test authoring, record the actual RED-author handle and a DIFFERENT actual calibration-author handle with the exact inventory. Select the independent PM before its review; select the separate actual production GREEN author only after RED approval and before GREEN dispatch. Pre-inventory planning may run before those later roles are selected, but no source/test write precedes its applicable before-edit gate. Accepted MAC-p9wm is satisfied context, not an additional blocker.

## nd_contract
status: new

### evidence
- Accepted producer hashes, exact six-path map, sequential broker ownership and one-way dependency are fixed above.
- Canonical repair is tracker-only; no source/test/ref/worktree/runtime/native/preflight/remote/install action occurred.

### proof
- [x] Closed real checker profile and exact real command/run/replay ownership are prospectively bounded.
- [x] Existing qlw2/MAC-yig6 tests, hpqp chain and public custody invariants are preserved.
- [ ] Exact before-RED inventory, staged actual-role selection, implementation and both native-host matrices remain pending.

