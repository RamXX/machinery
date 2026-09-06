---
id: MAC-wi2u
title: "Prove Go assertions through the native test runner"
status: open
priority: 1
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T09:01:51Z
created_by: ramirosalas
updated_at: 2026-09-06T09:01:51Z
content_hash: "sha256:7237367afdc1def33602a8596bdcc2e6798b33bc01bc05d0e75962471ee34642"
blocked_by: [MAC-bz1y, MAC-6h0s]
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
- internal/tdd/adapters/go.go -> internal/tdd/adapters/go.go -> the closed go-testing/v1 implementation of Adapter { ID() string; Prepare(context.Context, SuiteRequest) (PreparedSuite, error); Run(context.Context, PreparedSuite, EventSink) (Execution, error) }; embedded go assertion/harness assets under the owned assets directory; internal/runtimeclosure/go.go -> exact approved RuntimeHandle implementation for Go 1.27.1, no new user-selectable adapter API; named required contributor fragment.
- internal/tdd/adapters/go_test.go -> owned artifact for the same bounded contract
- internal/tdd/adapters/go_integration_test.go -> owned artifact for the same bounded contract
- internal/tdd/adapters/assets/go/README.md -> owned artifact for the same bounded contract
- internal/runtimeclosure/go.go -> owned artifact for the same bounded contract
- internal/runtimeclosure/go_test.go -> owned artifact for the same bounded contract
- testdata/integration-lanes/assurance-go.json -> owned artifact for the same bounded contract

### CONSUMES
MAC-6h0s: internal/tdd/types.go
  spec: Adapter, SuiteRequest{Inputs InputView;Suite Suite;Source BundleRef;Scratch string;Runtime RuntimeHandle;Scope processscope.Scope;Limits Limits}, EventSink func(Event) error and machinery.tdd.event/v1 normalized identities; PreparedSuite is opaque and cannot be deserialized.
MAC-bz1y: scripts/integration-lane/assurance_catalog.go
  MAC-6h0s: closed required fragment union with exact Go 1.27.1 native closure and Linux amd64/Darwin arm64 execution accounting.
MAC-qlw2: internal/processscope/scope.go
  spec: Scope.Run(context.Context, Command, Streams) (Result, error); use remaining absolute budget and owned child scope, never direct unowned subprocess.

### ACCEPTANCE CRITERIA
1. Implement ONLY go-testing/v1 with the exact approved version catalog and native invocation, source-binding, assertion, lifecycle and unsupported-feature rules. Validate/open the real complete runtime closure, reject absent/mismatched/mutated runtime before work; public support statements match the trusted-host native-cooperative boundary.
2. Use the exact closed go test -json -count=1 -shuffle=off invocation, anchored selected package/test identities, declared build tags and bounded native timeout from the approved contract. Compile/build-selected sources and typed helper calls define discoverable identities; authoritative native run events cover fixed dynamic subtests. No GOFLAGS override, cache result, module fetch, custom TestMain, fuzz or benchmark framework.
3. Ship byte-pinned Machinery assertion helper/transport and bind each registered assertion witness to its typed source location, test identity and actual testing lifecycle. A false registered helper condition performs native failure; direct Error/Fatal/Fail, setup/import/build/panic/timeout and fabricated JSON summary are not eligible assertion-based RED. A helper name or line substring alone is not proof.
4. Positive calibration proves a real passing and failing registered assertion, controls and subtest identity on native Linux amd64 and Darwin arm64. Negative calibration includes direct t.Fatal in setup/body, TestMain interception, omitted/duplicate/paused-unfinished/skipped subtests, cached/empty runs, malformed/truncated/forged output, mismatched file/line/witness, early process exit, extra failure, changed helper/dependency and missing runtime. Every registered assertion reaches execution in each required state.
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
- 2026-09-06T09:10:00Z dep_added: blocked_by MAC-bz1y
- 2026-09-06T09:10:01Z dep_added: blocked_by MAC-6h0s

## Links
- Parent: [[MAC-ui8a]]
- Blocked by: [[MAC-bz1y]], [[MAC-6h0s]]

## Comments
