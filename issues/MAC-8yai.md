---
id: MAC-8yai
title: "Prove Elixir assertions through native ExUnit"
status: closed
priority: 1
type: feature
labels: [hard-tdd, accepted]
parent: MAC-ui8a
created_at: 2026-09-06T09:01:52Z
created_by: ramirosalas
updated_at: 2026-09-07T21:01:13Z
content_hash: "sha256:017564e5f9a426e0da7558f1eed963cdf0f29db9de8a89bf155e39d5ffc96690"
was_blocked_by: [MAC-6h0s, MAC-bz1y]
follows: [MAC-6h0s, MAC-bz1y]
assignee: dev-MAC-8yai
closed_at: 2026-09-07T21:01:13Z
close_reason: "Accepted: Elixir assertions proven through native ExUnit; merged to local epic"
led_to: [MAC-o82q]
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
- internal/tdd/adapters/elixir.go -> internal/tdd/adapters/elixir.go -> the closed elixir-exunit/v1 implementation of Adapter { ID() string; Prepare(context.Context, SuiteRequest) (PreparedSuite, error); Run(context.Context, PreparedSuite, EventSink) (Execution, error) }; embedded elixir assertion/harness assets under the owned assets directory; internal/runtimeclosure/elixir.go -> exact approved RuntimeHandle implementation for Elixir/ExUnit/Mix 1.20.4, OTP 29.0.6, ERTS 17.0.6, no new user-selectable adapter API; named required contributor fragment.
- internal/tdd/adapters/elixir_test.go -> owned artifact for the same bounded contract
- internal/tdd/adapters/elixir_integration_test.go -> owned artifact for the same bounded contract
- internal/tdd/adapters/assets/elixir/README.md -> owned artifact for the same bounded contract
- internal/runtimeclosure/elixir.go -> owned artifact for the same bounded contract
- internal/runtimeclosure/elixir_test.go -> owned artifact for the same bounded contract
- testdata/integration-lanes/assurance-elixir.json -> owned artifact for the same bounded contract

### CONSUMES
MAC-6h0s: internal/tdd/types.go
  spec: Adapter, SuiteRequest{Inputs InputView;Suite Suite;Source BundleRef;Scratch string;Runtime RuntimeHandle;Scope processscope.Scope;Limits Limits}, EventSink func(Event) error and machinery.tdd.event/v1 normalized identities; PreparedSuite is opaque and cannot be deserialized.
MAC-bz1y: scripts/integration-lane/assurance_catalog.go
  MAC-6h0s: closed required fragment union with exact Elixir/ExUnit/Mix 1.20.4, OTP 29.0.6, ERTS 17.0.6 native closure and Linux amd64/Darwin arm64 execution accounting.
MAC-qlw2: internal/processscope/scope.go
  spec: Scope.Run(context.Context, Command, Streams) (Result, error); use remaining absolute budget and owned child scope, never direct unowned subprocess.

### ACCEPTANCE CRITERIA
1. Implement ONLY elixir-exunit/v1 with the exact approved version catalog and native invocation, source-binding, assertion, lifecycle and unsupported-feature rules. Validate/open the real complete runtime closure, reject absent/mismatched/mutated runtime before work; public support statements match the trusted-host native-cooperative boundary.
2. Use pinned Elixir/Mix/OTP closure, fresh private MIX_BUILD_PATH, offline exact dependencies/configuration and native Mix compilation with warnings-as-errors. Embedded bootstrap and formatter use the closed approved argv, never a project alias, stale BEAM or user formatter. Validate effective ExUnit options after test_helper executes; seed and max_cases are explicit.
3. Bind exact module/name/file/line lifecycle (including async ordering) and registered byte-pinned helper ExUnit.AssertionError in the test body. setup/setup_all/on_exit error, throw/raise/exit, timeout and non-helper assertion are never expected RED. All registered assertion sites must be reached; formatter summary alone is not evidence.
4. Positive calibration on native Linux amd64 and Darwin arm64 proves real synchronous/async passing controls and expected assertion failure. Negative calibration rejects excludes/includes/filter/max-failures/partitions, helper reconfiguration, custom frameworks/doctest, missing/duplicate/incomplete tests, compile warnings/errors, stale build cache, forged events, runtime/formatter replacement and leaked BEAM child ownership.
5. Normalize exact machinery.tdd.event/v1 sequence and qualified design/milestone/suite/native/assertion identities. Retain bounded raw streams/events; reject unknown/missing/duplicate/out-of-order/truncated lifecycle, required skip/xfail, empty selection and unexpected failures. No native event can reassign owner-provided identity. Fixtures/protocol unit tests supplement actual runtime executions.
6. Use processscope custody for preparation, compilation, runtime probes and test processes; retain source/control/runtime identity through execution and private cleanup. Apply the same cumulative milestone deadline across compile/probes/tests and one inherited cleanup grace, fail on overflow/leak/late source mutation. RuntimeHandle.Close is pure post-no-launch revalidation, not a late subprocess.
7. Add its required native conformance fragment to the closed contributor union without changing existing frozen pilot bytes or another adapter's files. Demonstrate real selected fragment success and omission/skip/runtime absence failure in persistent CI on both native platforms; no env gate or fake executable as positive proof. Downstream replay and normal CLI must consume this Adapter before the integration umbrella is deliverable.

### TEST COMMANDS
Run targeted native package tests for owned files and exact required contributor fragments; record selected test identities and exact commands in RED. Do not run heavy preflight.

### OUT OF SCOPE
Other languages/frameworks, final orchestration, consumer example migrations and normal CLI ownership belong to explicit downstream stories. No unsupported framework may silently inherit this adapter's strong assurance label.

### DIFF BUDGET
~9 files, under 2100 changed LOC including native conformance fixtures; gross overrun requires investigation.

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
- 2026-09-06T09:10:03Z dep_added: blocked_by MAC-bz1y
- 2026-09-06T09:10:03Z dep_added: blocked_by MAC-6h0s
- 2026-09-06T09:10:06Z dep_added: blocks MAC-sd7g
- 2026-09-06T09:10:23Z dep_added: blocks MAC-vx24
- 2026-09-06T09:10:23Z dep_added: blocks MAC-ou97
- 2026-09-07T01:42:40Z dep_removed: was_blocked_by MAC-6h0s
- 2026-09-07T10:08:34Z dep_removed: was_blocked_by MAC-bz1y
- 2026-09-07T18:47:02Z status: open -> in_progress
- 2026-09-07T18:47:02Z auto-follows: linked to predecessor MAC-6h0s
- 2026-09-07T18:47:02Z auto-follows: linked to predecessor MAC-bz1y
- 2026-09-07T21:01:13Z status: in_progress -> closed
- 2026-09-07T21:01:13Z dep_removed: no_longer_blocks MAC-sd7g
- 2026-09-07T21:01:13Z dep_removed: no_longer_blocks MAC-vx24
- 2026-09-07T21:01:13Z dep_removed: no_longer_blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Was blocked by: [[MAC-6h0s]], [[MAC-bz1y]]
- Follows: [[MAC-6h0s]], [[MAC-bz1y]]
- Led to: [[MAC-o82q]]

## Comments

### 2026-09-06T09:16:32Z ramirosalas
CANONICAL MACHINE-READABLE BOUNDARY MAP 2026-09-06
This repeats the existing ownership/signatures in the parser-supported form; it does not create additional scope or weaken any AC. All prior exact acceptance/testing requirements and holds remain current.

PRODUCES:
- internal/tdd/adapters/elixir.go -> internal/tdd/adapters/elixir.go -> the closed elixir-exunit/v1 implementation of Adapter { ID() string; Prepare(context.Context, SuiteRequest) (PreparedSuite, error); Run(context.Context, PreparedSuite, EventSink) (Execution, error) }; embedded elixir assertion/harness assets under the owned assets directory; internal/runtimeclosure/elixir.go -> exact approved RuntimeHandle implementation for Elixir/ExUnit/Mix 1.20.4, OTP 29.0.6, ERTS 17.0.6, no new user-selectable adapter API; named required contributor fragment.
- internal/tdd/adapters/elixir_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/tdd/adapters/elixir_integration_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/tdd/adapters/assets/elixir/README.md -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/runtimeclosure/elixir.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/runtimeclosure/elixir_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- testdata/integration-lanes/assurance-elixir.json -> owned bounded artifact; behavior and tests specified in the current story AC

CONSUMES:
- MAC-6h0s: internal/tdd/types.go
  spec: Adapter, SuiteRequest{Inputs InputView;Suite Suite;Source BundleRef;Scratch string;Runtime RuntimeHandle;Scope processscope.Scope;Limits Limits}, EventSink func(Event) error and machinery.tdd.event/v1 normalized identities; PreparedSuite is opaque and cannot be deserialized.
- MAC-bz1y: scripts/integration-lane/assurance_catalog.go
  MAC-6h0s: closed required fragment union with exact Elixir/ExUnit/Mix 1.20.4, OTP 29.0.6, ERTS 17.0.6 native closure and Linux amd64/Darwin arm64 execution accounting.
  schema: closed required fragment union with exact Elixir/ExUnit/Mix 1.20.4, OTP 29.0.6, ERTS 17.0.6 native closure and Linux amd64/Darwin arm64 execution accounting.
- MAC-qlw2: internal/processscope/scope.go
  spec: Scope.Run(context.Context, Command, Streams) (Result, error); use remaining absolute budget and owned child scope, never direct unowned subprocess.
- MAC-l7m0: docs/test-assurance-contract.md
  schema: Exact approved public contract SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8; accepted delivery required, never an uncommitted external proposal.

Observable outcome: An Elixir consumer receives real ExUnit assertion proof with exact effective settings; filters or setup errors return failure.

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

### 2026-09-06T09:22:39Z ramirosalas
SCHEMA MARKER INSERTION AUDIT 2026-09-06
Root-authorized supported nd edit inserted ONLY 1 valid indented schema signature line(s) into the newly authored canonical CONSUMES block. Every original byte/contract/status/evidence/history was preserved; no deletion/replacement. This repairs the mechanical label substitution, not the contract values.
Before raw Body SHA256: ca13eb725820a0c868cb87c0937e09b588eedbd5bd18b7aba9c25ea5bcfcd1b8
After insertion-only raw Body SHA256 (before this audit comment): 558a3de229bb2d8948cdfe36f8575a855ba598331594b7d9e7013f093c8418a3
Exact inserted lines (zero-based original Body line positions shown):
- after Body line 111: "  schema: closed required fragment union with exact Elixir/ExUnit/Mix 1.20.4, OTP 29.0.6, ERTS 17.0.6 native closure and Linux amd64/Darwin arm64 execution accounting."
Read-back pvg nd show Body exactly equals prior Body plus these insertions. Editor required expected hash, count, exact target and 13-entry total; installed pvg source revision c0957106a81346033d7b1d82fde5f434a9db6bab confirms scanner checks every historical entry.

## nd_contract
status: new

### evidence
- Signature syntax corrected via supported guarded editor; exact before/after evidence above.
- No implementation/native proof; independent Anchor and canonical document holds remain.

### proof
- [ ] All current story ACs remain pending without weakening.

### 2026-09-06T09:48:02Z ramirosalas
ROUND-1 RULE 1 REPAIR: COMPLETE elixir-exunit/v1 EXECUTABLE ASSET OWNERSHIP
This is the unambiguous complete CURRENT ownership for the already approved adapter, replacing README-only/ambiguous directory wording. No new adapter/framework/runtime semantics.

PRODUCES:
- internal/tdd/adapters/elixir.go -> exact approved Adapter implementation AND its executable-asset embedding/materialization/byte-verification/transport wiring (including Go embed declarations); no unowned common embedding file is assumed.
- internal/tdd/adapters/elixir_test.go -> focused parser/helper/transport and immutable-asset negative tests.
- internal/tdd/adapters/elixir_integration_test.go -> real native assertion/lifecycle/custody conformance.
- internal/tdd/adapters/assets/elixir -> EXCLUSIVELY owned bounded executable asset directory: all language helper/assertion transport, bootstrap/reporter/formatter needed by THIS closed adapter, plus its README. At most8 executable/configuration asset files plus README; exact paths reviewed before RED, no undeclared file or vendored runtime.
- internal/runtimeclosure/elixir.go -> exact pinned language runtime closure.
- internal/runtimeclosure/elixir_test.go -> real closure validation and mutation controls.
- testdata/integration-lanes/assurance-elixir.json -> exact required native conformance fragment.

CONSUMES:
- MAC-6h0s: internal/tdd/types.go
  spec: Adapter { ID() string; Prepare(context.Context,SuiteRequest)(PreparedSuite,error); Run(context.Context,PreparedSuite,EventSink)(Execution,error) }; exact machinery.tdd.event/v1 and closed requests.
- MAC-qlw2: internal/processscope/scope.go
  spec: Scope.Run(context.Context,Command,Streams)(Result,error), remaining deadline and owned cleanup.
- MAC-bz1y: scripts/integration-lane/assurance_catalog.go
  schema: exact native runtime versions and closed required-fragment union on Linux amd64/Darwin arm64.
- MAC-l7m0: docs/test-assurance-contract.md
  schema: unchanged section4 closed identities/budgets, section5 assertion causality, section6 elixir-exunit/v1 semantics, section7 API, section8 custody.

CURRENT ACCEPTANCE / INVENTORY MAP
All seven existing adapter ACs remain required unchanged: exact native profile/runtime, approved fixed invocation, real source-bound assertion cause, full positive+negative two-platform conformance, exact normalized lifecycle, cumulative scoped custody and persistent required-lane integration.
Before ANY RED authoring, provide an exact bounded executable asset inventory (paths, roles, entrypoints, imports/dependencies, embedding owner, byte-source/materialization relation and planned test identities) for independent review. No future test IDs need be invented now; the exact inventory must be approved and frozen before RED execution/implementation authorization. A README alone cannot satisfy helper delivery.
All executable helper/transport/bootstrap/reporter assets for this adapter live in the owned bounded directory and are embedded/wired by the owned elixir.go production adapter. Their exact complete bytes/config/dependency closure become frozen protocol inputs, materialized with approved topology/modes and checked before use. No unreviewed project reporter/helper, alternate filesystem copy, ambient fallback or unowned central asset registrar.
Real native positive/negative conformance must execute the EMBEDDED production assets through this Adapter and normal downstream replay/complete. Missing, substituted, unembedded, extra undeclared, mutated or not-actually-invoked asset must fail; printed fabricated events or tests aimed only at README cannot pass.
Preserve all existing tests/helper semantics and other adapters' directories; no frozen pilot edits. If the exact inventory cannot fit the bound, stop for separately reviewed decomposition, never hide files in a generic directory.
DIFF BUDGET: at most15 actual files including bounded assets, under2100 changed LOC as originally budgeted; investigate overrun before expanding.
MANDATORY SKILLS: developer and pm_acceptor. Product independence, native no-mock/no-skip proof and final-only preflight hold remain.

## nd_contract
status: new

### evidence
- Independent round1 findings repaired through supported append-only scope/consumer notes; no architecture/source/test/ref/runtime mutation.
- Exact source surfaces verified against accepted epic7e36f3e7ddcf25565d5d4fe60b328df254eee91d; prior Body/status/labels/history preserved.
- Independent Anchor ROUND2 and canonical-document acceptance remain prerequisites, not implementation evidence.

### proof
- [ ] All current story ACs, strengthened ownership/current-judgment requirements and protected frozen proof remain required.
- [ ] Independent review/native execution/final acceptance pending.

### 2026-09-07T21:01:13Z ramirosalas
ACCEPTED 2026-09-07 — RED d53467f (31 subjects, 6 frozen assets natively pre-verified) -> GREEN 866a27a. Elixir/OTP closure byte-pinned tree fingerprint (2 scoped probes, pure Close); byte-pinned harness + witness-transport reconciliation; effective-option enforcement; exit concordance (0/2/1); describe-prefixed native name atoms; truncated autorun stream on compile failure handled; no flaky-retry in ExUnit 1.20.4 core (pinned); comment-registered call sites impossible (prior oracle-defect class closed). Owned packages green; -race green; lane green after union 1a25305 (assurance-elixir-conformance 3/3). Overrun ~2.2x — test-dominated, sibling precedent; one disclosed catalog supersession + six enumerated RED-audit repairs. RESIDUALS: Linux rides hosted CI; deps-free harness-only v1 surface. WATCH: one first-run flake of frozen saga-closure TLC case under load (base control + rerun green). Record: .git/machinery-evidence-20260906.TEFZ7D/8yai-record.md
