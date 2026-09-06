---
id: MAC-1u2v
title: "Exercise fresh standalone assurance in all four languages"
status: open
priority: 1
type: feature
labels: [integration]
parent: MAC-ui8a
created_at: 2026-09-06T09:07:16Z
created_by: ramirosalas
updated_at: 2026-09-06T09:07:16Z
content_hash: "sha256:72ea4543db0ba297603e2e90f22f044251f6318040cdf361377f0a84e182f90f"
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
- cmd/machinery/assurance_standalone_e2e_test.go -> cmd/machinery/assurance_standalone_e2e_test.go -> real public CLI and hook end-to-end scenarios with Go, TypeScript node:test, Python unittest and Elixir ExUnit consumer fixtures; required native two-platform fragment. Tests-only acceptance, no production fix authority.
- cmd/machinery/testdata/assurance-standalone -> owned artifact for the same bounded contract
- testdata/integration-lanes/assurance-standalone.json -> owned artifact for the same bounded contract

### CONSUMES
MAC-u4oo: cmd/machinery/tdd.go
  endpoint: exact store init/scaffold/capture/register/red/green/status/verify/strict/complete command sequence and real hook payloads.
MAC-al5u: scripts/assurance-examples.sh
  source: complete-example strict invocation and required legacy/new evidence inventories.
MAC-rau8: internal/assuranceflow/run.go
  spec: Run(ctx context.Context,req Request,output io.Writer)(Verification,error), indirectly via actual candidate binary only.

### ACCEPTANCE CRITERIA
1. For EACH of Go, TypeScript, Python and Elixir start from an empty consumer directory and a genuinely new external store with no previous head beyond generation0, no PASS receipt, no tracker metadata and no pvg/nd/toolchain-management executables available. Complete real init -> scaffold -> authored capture variants -> register -> RED -> GREEN -> verify -> complete plus actual hooks.
2. Tests use actual production candidate binary/native runtimes/public application boundary, not mocked commands. Independently observe named native assertion/lifecycle events, expected baseline failure with red_controls, safe/unsafe negative sensitivity and all GREEN tests/checks passing.
3. Unsafe implementation mutations fail the intended bound assertions; mutation of frozen test/helper/config/lock bytes, mode or empty-directory topology fails identity checks. Forge a receipt/summary, omit milestone/owner/test, change registered head, narrow selector, skip/xfail, stale current code, unsupported framework/runtime or missing prerequisite must be refused for the correct reason.
4. Run real failure/cancellation/output-overflow/early intermediate cases and prove owned children/containers clean before result, no unrelated resource touched, no late process after barrier and no sealed output on cleanup/publication/output error. Actual native platform evidence is required for Linux amd64 and Darwin arm64.
5. Prove fresh explicit register CAS and already-registered retry, failed history retained after retry/import, wrong expected head conflict, selected verify scope cannot pass complete and cheap CLI/Stop freshness explicitly says replay not performed.
6. Tests-only: no manufactured missing-feature RED is required after upstream functionality exists. Author/freeze exact E2E fixtures and prove mutation sensitivity with green controls; missing required behavior becomes a blocking P0 story, never weakened test. This is a focused prerequisite, NOT the sole epic capstone and not authorization to run heavy preflight.
7. Register all exact E2E leaves in the required native fragment and prove omission/skip/empty/fabricated result fails persistent lane accounting. Final MAC-ou97 remains blocked by this delivery and still exercises every earlier hardening behavior.

### TEST COMMANDS
Run targeted native package tests for owned files and exact required contributor fragments; record selected test identities and exact commands in RED. Do not run heavy preflight.

### OUT OF SCOPE
Production fixes and the final epic-wide preflight/build/local merge/publication are not owned here. No installed binary/plugin replacement.

### DIFF BUDGET
~18 E2E fixture/test files, under 2000 changed LOC.

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


## Links
- Parent: [[MAC-ui8a]]

## Comments
