---
id: MAC-sqpt
title: "Reject assurance that fails retained-state replay"
status: open
priority: 2
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T09:03:33Z
created_by: ramirosalas
updated_at: 2026-09-24T21:33:44Z
content_hash: "sha256:eb5c7b4106f18bd24e260685b35d20475260a0a3dd9b1c7658bd33f05674af38"
blocked_by: [MAC-sd7g]
blocks: [MAC-rau8, MAC-vx24, MAC-ou97]
was_blocked_by: [MAC-62s6]
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
- internal/tdd/execute.go -> internal/tdd -> Execute(ctx context.Context, req ExecuteRequest) (Candidate, error); Verify(ctx context.Context, req VerifyRequest) (Candidate, error). Exact section 7 requests, private provisional Candidate, retained run/failed records and phase transition state machine. tdd MUST NOT import gates or assuranceflow.
- internal/tdd/replay.go -> owned artifact for the same bounded contract
- internal/tdd/evidence.go -> owned artifact for the same bounded contract
- internal/tdd/execute_test.go -> owned artifact for the same bounded contract
- internal/assuranceflow/replay_integration_test.go -> owned artifact for the same bounded contract
- testdata/integration-lanes/assurance-replay.json -> owned artifact for the same bounded contract

### CONSUMES
MAC-62s6: internal/assuranceflow/register.go
  spec: Register(ctx context.Context,req RegisterRequest,output io.Writer)(Registration,error); exact registered external head/revision graph, no implicit registration.
MAC-p9z1: internal/tdd/capture.go
  spec: Capture(ctx context.Context,req CaptureRequest)(BundleRef,error); Status(ctx context.Context,req StatusRequest)(StatusReport,error), immutable bundles and control/head identities.
MAC-sd7g: internal/assuranceflow/checks.go
  spec: closed production CheckExecutor.Run(context.Context,CheckRequest)(PendingChecks,error), Final() unavailable until owner releases views. Integration tests in assuranceflow use this actual production executor with the four upstream native adapters.

### ACCEPTANCE CRITERIA
1. Execute RED only on an explicitly registered retained immutable baseline; require exact expected registered assertion failures plus passing controls and complete test/assertion lifecycle. RED error classification excludes compilation/import/setup/panic/timeout/empty/skip/xfail/unexpected failure. Record actual native raw outcomes, not a receipt import.
2. Execute the same exact frozen suite against every declared safe/unsafe subject-only variant; every baseline expected assertion failure has its matching safe red_control, and every negative has passing safe and expected-failing unsafe sensitivity proof with exact target/non-target outcomes. Reject equal subject digests, changed frozen file sets/bytes/modes/directories or bypassed public boundary.
3. GREEN runs the current immutable implementation against the same complete frozen tests/helpers/config/locks and inventory; every required test/assertion/check passes. Deletion, rename, whitespace/literal/indentation edit, skip/selector shrink, changed build inputs, missing fixture or stale design/control/head invalidates evidence.
4. Verify independently replays retained RED, controls, unsafe variants and current GREEN with all four closed native adapters and required actual checks. Hash-only or forged/imported repo-writable receipts, prior ancestor acceptance and historical successful replay cannot stand for current execution. Historical failed records remain retained and separately reported.
5. Candidate is provisional only and cannot be publicly constructed/deserialized or claim sealed verification. Bind immutable inputs, project/store exact head and current qualified milestone graph to every record; no execution call may advance the registration head. Required pending/unverified obligation blocks completion.
6. Enforce aggregate per-milestone preparation/build/test/variant/check/parse/cleanup-wait deadline, counts/bytes/output/event caps and one inherited cleanup allowance; no per-suite reset. Real native tests combine individually under-budget steps into an over-budget operation and assert terminal failure/no new dispatch. Virtual clocks supplement, never replace actual native cancellation/cleanup.
7. Real replay matrix exercises positive and deliberately failing states for all four adapters through actual production CheckExecutor, retaining exact artifacts and proving stale/missing/forged evidence fails. Required lane fragment runs persistently on both native platforms; no caller-controlled fake CheckExecutor as success proof.

### TEST COMMANDS
Run targeted native package tests for owned files and exact required contributor fragments; record selected test identities and exact commands in RED. Do not run heavy preflight.

### OUT OF SCOPE
Public Verification sealing/output, normal CLI/gates/hooks and source example migration are downstream. Replay is not proof of actual historical authoring chronology or malicious-host containment.

### DIFF BUDGET
~8 files, under 2100 changed LOC including real replay matrices.

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

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-06T09:10:07Z dep_added: blocked_by MAC-62s6
- 2026-09-06T09:10:08Z dep_added: blocked_by MAC-sd7g
- 2026-09-06T09:10:09Z dep_added: blocks MAC-rau8
- 2026-09-06T09:10:26Z dep_added: blocks MAC-vx24
- 2026-09-06T09:10:26Z dep_added: blocks MAC-ou97
- 2026-09-07T21:43:47Z dep_removed: was_blocked_by MAC-62s6

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-rau8]], [[MAC-vx24]], [[MAC-ou97]]
- Blocked by: [[MAC-sd7g]]
- Was blocked by: [[MAC-62s6]]

## Comments

### 2026-09-06T09:16:32Z ramirosalas
CANONICAL MACHINE-READABLE BOUNDARY MAP 2026-09-06
This repeats the existing ownership/signatures in the parser-supported form; it does not create additional scope or weaken any AC. All prior exact acceptance/testing requirements and holds remain current.

PRODUCES:
- internal/tdd/execute.go -> internal/tdd -> Execute(ctx context.Context, req ExecuteRequest) (Candidate, error); Verify(ctx context.Context, req VerifyRequest) (Candidate, error). Exact section 7 requests, private provisional Candidate, retained run/failed records and phase transition state machine. tdd MUST NOT import gates or assuranceflow.
- internal/tdd/replay.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/tdd/evidence.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/tdd/execute_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/assuranceflow/replay_integration_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- testdata/integration-lanes/assurance-replay.json -> owned bounded artifact; behavior and tests specified in the current story AC

CONSUMES:
- MAC-62s6: internal/assuranceflow/register.go
  spec: Register(ctx context.Context,req RegisterRequest,output io.Writer)(Registration,error); exact registered external head/revision graph, no implicit registration.
- MAC-p9z1: internal/tdd/capture.go
  spec: Capture(ctx context.Context,req CaptureRequest)(BundleRef,error); Status(ctx context.Context,req StatusRequest)(StatusReport,error), immutable bundles and control/head identities.
- MAC-sd7g: internal/assuranceflow/checks.go
  spec: closed production CheckExecutor.Run(context.Context,CheckRequest)(PendingChecks,error), Final() unavailable until owner releases views. Integration tests in assuranceflow use this actual production executor with the four upstream native adapters.
- MAC-l7m0: docs/test-assurance-contract.md
  schema: Exact approved public contract SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8; accepted delivery required, never an uncommitted external proposal.

Observable outcome: A consumer receives provisional successful replay only when retained RED, controls, negative challenges and current GREEN genuinely satisfy the contract.

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
- [ ] AC #7: current story acceptance requirement remains pending.
