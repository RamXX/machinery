---
id: MAC-qlw2
title: "Retain native child ownership until cleanup completes"
status: open
priority: 0
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T08:58:17Z
created_by: ramirosalas
updated_at: 2026-09-06T08:58:17Z
content_hash: "sha256:220ce5e6f563a2c7029ff27356ace9aa547cba09ab0e0fec8289c8b036738c0e"
blocked_by: [MAC-l7m0]
blocks: [MAC-cn7q]
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
- internal/processscope/scope.go -> internal/processscope -> Open(context.Context, Options) (Scope, error); Join(context.Context, Capability) (Scope, error); ServeInternal(args []string, io InternalIO) (handled bool, exitCode int). Scope methods Run(context.Context, Command, Streams) (Result, error), Child(context.Context) (Scope, error), Attach(Command) (Command, error), Close(context.Context) (CleanupReport, error). Exact Command/Streams/Result/Options/ResourceState fields and capability semantics are the approved section 8 contract.
- internal/processscope/broker.go -> owned implementation/test artifact for the same contract
- internal/processscope/guardian_unix.go -> owned implementation/test artifact for the same contract
- internal/processscope/internal.go -> owned implementation/test artifact for the same contract
- internal/processscope/scope_test.go -> owned implementation/test artifact for the same contract
- internal/processscope/custody_integration_test.go -> owned implementation/test artifact for the same contract

### CONSUMES
MAC-l7m0: docs/test-assurance-contract.md
  schema: Section 8 complete native scope/broker/guardian contract; section 3 cumulative cleanup budgets.

### ACCEPTANCE CRITERIA
1. Implement the exact Scope/Command broker protocol with a private 0700 authority directory, scope/job capabilities and verified inherited-channel authentication. Register exact resource ownership BEFORE launch admission. Internal ServeInternal cannot be activated by an environment claim alone; unsupported platforms return UNSUPPORTED_PLATFORM.
2. Prove OPEN -> CLOSING refusal of new launches and sibling/child-scope cancellation, registration-before-start, terminal kill of owned process group BEFORE guardian Wait/reap, cleanup reporting and owner-liveness FD behavior. Direct-child group guardian remains a live unreaped identity anchor until cleanup; no PID/name scanning or historical ledger cleanup authority.
3. Real native Linux amd64 and Darwin arm64 tests start grandchildren, nested joined sibling launches and an early-exiting intermediate; demonstrate all owned resources terminated/reaped before success under normal finish, expected assertion failure, timeout, output overflow, interruption and lost owner channel. Do not rely on outer process-group inheritance alone.
4. Negative tests reject forged/stale/mismatched capabilities, stale PID records, registration failure, closed/reused scope, malformed handles and missing helper identity. Live unrelated sibling processes survive every cleanup test; same-name processes are not touched. ResourceState reports exact registration/termination/reaping; daemon-owned container cleanup uses registered exact container IDs or reports cleanup-failed, never assumes process death removes them.
5. One inherited absolute remaining deadline plus one shared bounded cleanup allowance governs nested scopes. Exhaustion is terminal; child Close cannot renew cleanup. Real bounded wall-time tests supplement deterministic cumulative clock arithmetic; unsupported hostile detached/uncooperative escape is reported honestly, not claimed contained.
6. Publish embedded same-binary ServeInternal test helper use and a real scope API consumer conformance test. This producer does not claim the normal CLI or contributor lane is integrated; explicit downstream custody wiring and MAC-hpqp consume it.

### TEST COMMANDS
Run targeted native package tests for the owned implementation and the specifically inventoried required contributor lane fragments; record the exact selected leaf inventory and command in RED. No heavy preflight during this story.

### OUT OF SCOPE
Formal/CLI compatibility wiring is the custody attachment consumer; four adapter semantics and final flow belong to their explicit downstream stories. Hostile-host containment and new platform support are not v1.

### DIFF BUDGET
~10 files, under 2600 changed LOC; investigate overruns before expanding.

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
- 2026-09-06T09:09:57Z dep_added: blocked_by MAC-l7m0
- 2026-09-06T09:09:57Z dep_added: blocks MAC-cn7q

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-cn7q]]
- Blocked by: [[MAC-l7m0]]

## Comments
