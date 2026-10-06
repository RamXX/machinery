---
id: MAC-u4oo
title: "Enforce standalone assurance through normal Machinery commands"
status: open
priority: 2
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T09:05:11Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:55Z
content_hash: "sha256:619c3fa64300f7272c85577e13e99ee8965a116a8ca4798bca0ed91b932f079e"
blocked_by: [MAC-rau8]
blocks: [MAC-5ft8, MAC-1u2v, MAC-vx24]
was_blocked_by: [MAC-2n83, MAC-62s6]
related: [MAC-4ah8]
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
- cmd/machinery/tdd.go -> cmd/machinery/tdd.go -> exact section 7 tdd store init/export/import, scaffold, capture, register, red, green, status, verify commands; normal check --assurance strict and --complete route through assuranceflow.Run; hook assurance_store and assurance_required fields with cheap freshness enforcement and exact-byte frozen pre-write protection.
- cmd/machinery/tdd_test.go -> owned artifact for the same bounded contract
- cmd/machinery/tdd_integration_test.go -> owned artifact for the same bounded contract
- cmd/machinery/main.go -> owned artifact for the same bounded contract
- cmd/machinery/check.go -> owned artifact for the same bounded contract
- internal/hook/hook.go -> owned artifact for the same bounded contract
- internal/hook/assurance_test.go -> owned artifact for the same bounded contract
- testdata/integration-lanes/assurance-cli.json -> owned artifact for the same bounded contract

### CONSUMES
MAC-rau8: internal/assuranceflow/run.go
  spec: Run(ctx context.Context,req Request,output io.Writer)(Verification,error); modes red|green|verify|strict-check|complete-check.
MAC-62s6: internal/assuranceflow/register.go
  spec: Register(ctx context.Context,req RegisterRequest,output io.Writer)(Registration,error); explicit ExpectedHead and selected Milestones.
MAC-p9z1: internal/tdd/store.go
  spec: Capture(ctx context.Context,req CaptureRequest)(BundleRef,error); Status(ctx context.Context,req StatusRequest)(StatusReport,error); exact store init/export/import schemas.
MAC-2n83: cmd/machinery/main.go
  source: normal activation/recovery command registration retained; newRootCmd() *cobra.Command.

### ACCEPTANCE CRITERIA
1. Register exactly the approved CLI forms and required flags. Store is explicit --store for capture/red/green/status/verify/strict; no environment fallback. Register requires --expected-head and repeat --milestone (omission means all). There is no import-receipt, accept-exit-code, skip-negative, force-green or arbitrary command adapter.
2. Implement fresh store initialization, new-path-only scaffold drafts/helpers, immutable capture and explicit register without any prior PASS receipt. Scaffold must not invent assertion semantics/review approval or overwrite existing paths; capture only prints retained bundle reference, register only registered-not-executed/already-registered-not-executed. Only verify and strict/complete checks perform independent replay.
3. Normal machinery check <design> --impl <path> --store <path> --assurance strict and --complete invoke the exact assuranceflow owner; complete requires all current milestones, controls, negative sensitivity, checks and real final gates. Targeted verify reports selected scope and cannot supply complete assurance for omitted milestones.
4. Use exact exit semantics: 0 requested contract satisfied (RED only exact expected native failures), 1 evaluated mismatch/stale/missing assurance, 2 invocation/schema/unsupported/runtime/cleanup/protocol error; interruption uses2 and records causal signal. Well-formed draft status may return0 with explicit non-GREEN state, never strict success.
5. Wire .machinery.json assurance_store and assurance_required true with no hidden fallback. Ordinary CLI/Stop cheap path is explicit replay-not-performed and blocks required stale/missing assurance; pre-write hook protects exact frozen bytes/config/topology and rejects old tokens-equal exemption. Unsupported frameworks/platforms produce actionable fail-closed diagnostics rather than silently grandfathering strict attestations.
6. Actual candidate-binary and real hook-payload tests exercise first-use init/scaffold/capture/register/RED/GREEN/status/verify/strict/complete plus missing store, stale receipt/current-code mutation, unsupported adapter/runtime, dropped milestone, scope downgrade, signal and output failure. No mocked command dispatcher, pvg/nd metadata or installed binary dependence.
7. Verify new CLI and existing recovery/activation/ServeInternal paths coexist, all subprocesses stay in actual scope/deadline graph and required native CLI fragment runs on both platforms. Guidance content is owned by existing MAC-gcrr after this consumer; do not edit installed skills/plugins or publish user-specific tracker details.

### TEST COMMANDS
Run targeted native package tests for owned files and exact required contributor fragments; record selected test identities and exact commands in RED. Do not run heavy preflight.

### OUT OF SCOPE
Legacy example wholesale migration is the prospective example consumer; public guidance remains MAC-gcrr. Installed candidate replacement requires root clearance after NIL, never this story.

### DIFF BUDGET
~10 files, under 2200 changed LOC including full CLI/hook tests.

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
Triage 2026-09-24: absorbs the only unique remnant of closed MAC-vx24: update consumer guidance (agents/machinery-build-writer.md, SKILL.md, build-md-template) for replayable negative test assurance.
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Add `machinery tdd` subcommands, `check --store --assurance strict` and `--complete` routed through assuranceflow.Run, and the hook assurance fields and frozen-byte pre-write protection. Evidence: CHANGELOG 0.7.0 says the assurance substrate ships with no CLI/gate consumer. Greps of *.go find no CheckTDDAssurance, GateExecution, TDDStatus, assurance_store or strict-check. internal/assuranceflow holds only register.go. internal/tdd has no execute.go, replay.go or evidence.go, and has CheckExecutor only as a type in types.go. cmd/machinery has no tdd command. Only assurance_docs tests mention assurance. internal/hook has no assurance_store or assurance_required fields. Notes: This is the story that gives consumers any CLI surface for assurance (MAC-tk0i, closed, recorded the gap). It collides with 0.11.0 hook work (--landing, crash recovery), so the hook ACs must be re-based on current internal/hook. Blocks 5ft8 and 1u2v. MAC-2n83 and MAC-62s6 blockers are already removed.

## History
- 2026-09-06T09:10:10Z dep_added: blocked_by MAC-rau8
- 2026-09-06T09:10:11Z dep_added: blocked_by MAC-62s6
- 2026-09-06T09:10:11Z dep_added: blocked_by MAC-2n83
- 2026-09-06T09:10:12Z dep_added: blocks MAC-5ft8
- 2026-09-06T09:10:14Z dep_added: blocks MAC-1u2v
- 2026-09-06T09:10:28Z dep_added: blocks MAC-vx24
- 2026-09-06T09:10:29Z dep_added: blocks MAC-ou97
- 2026-09-07T07:22:44Z dep_removed: was_blocked_by MAC-2n83
- 2026-09-07T21:43:47Z dep_removed: was_blocked_by MAC-62s6
- 2026-09-24T21:33:55Z dep_removed: no_longer_blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-5ft8]], [[MAC-1u2v]], [[MAC-vx24]]
- Blocked by: [[MAC-rau8]]
- Was blocked by: [[MAC-2n83]], [[MAC-62s6]]
- Related: [[MAC-4ah8]]

## Comments

### 2026-09-06T09:16:33Z ramirosalas
CANONICAL MACHINE-READABLE BOUNDARY MAP 2026-09-06
This repeats the existing ownership/signatures in the parser-supported form; it does not create additional scope or weaken any AC. All prior exact acceptance/testing requirements and holds remain current.

PRODUCES:
- cmd/machinery/tdd.go -> cmd/machinery/tdd.go -> exact section 7 tdd store init/export/import, scaffold, capture, register, red, green, status, verify commands; normal check --assurance strict and --complete route through assuranceflow.Run; hook assurance_store and assurance_required fields with cheap freshness enforcement and exact-byte frozen pre-write protection.
- cmd/machinery/tdd_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- cmd/machinery/tdd_integration_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- cmd/machinery/main.go -> owned bounded artifact; behavior and tests specified in the current story AC
- cmd/machinery/check.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/hook/hook.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/hook/assurance_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- testdata/integration-lanes/assurance-cli.json -> owned bounded artifact; behavior and tests specified in the current story AC

CONSUMES:
- MAC-rau8: internal/assuranceflow/run.go
  spec: Run(ctx context.Context,req Request,output io.Writer)(Verification,error); modes red|green|verify|strict-check|complete-check.
- MAC-62s6: internal/assuranceflow/register.go
  spec: Register(ctx context.Context,req RegisterRequest,output io.Writer)(Registration,error); explicit ExpectedHead and selected Milestones.
- MAC-p9z1: internal/tdd/store.go
  spec: Capture(ctx context.Context,req CaptureRequest)(BundleRef,error); Status(ctx context.Context,req StatusRequest)(StatusReport,error); exact store init/export/import schemas.
- MAC-2n83: cmd/machinery/main.go
  source: normal activation/recovery command registration retained; newRootCmd() *cobra.Command.
- MAC-l7m0: docs/test-assurance-contract.md
  schema: Exact approved public contract SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8; accepted delivery required, never an uncommitted external proposal.

Observable outcome: A user can complete a standalone first-use assurance workflow through normal Machinery commands and receives fail-closed diagnostics for stale or unsupported inputs.

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
