---
id: MAC-qlw2
title: "Retain native child ownership until cleanup completes"
status: in_progress
priority: 0
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T08:58:17Z
created_by: ramirosalas
updated_at: 2026-09-06T12:05:48Z
content_hash: "sha256:8de87b773eb1cd60d63b289cc6f5d6c81cd524df91035f9853547ae5cc41dfaa"
blocks: [MAC-cn7q, MAC-6h0s, MAC-vx24, MAC-ou97]
was_blocked_by: [MAC-l7m0]
assignee: dev-MAC-qlw2
follows: [MAC-l7m0]
blocked_by: [MAC-p9wm]
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
## BEFORE-RED inventory checkpoint

External immutable proposal: `/tmp/MAC-qlw2-before-red-proposal.md` SHA256 `6b1fdf8f88751ffaeced19b95ca70832aca37f50d3881286779ca8721fa82ad2`.

No repository source/test/fixture/config bytes, RED commit, native process/container execution, or delivery transition occurred. Independent approval is required before authoring the exact six-file inventory. Concrete review gates: exact Go representation of contract-required `Limits` and `Diagnostic`; private authenticated `InternalIO` descriptor semantics; approval of same-test-binary producer helper; authorized native Linux amd64 executor.

## nd_contract
status: in_progress

### evidence
- Read current `pvg nd show MAC-qlw2` through EOF and contract `docs/test-assurance-contract.md` (575 lines; SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8).
- External proposal hash: 6b1fdf8f88751ffaeced19b95ca70832aca37f50d3881286779ca8721fa82ad2.
- Read-only feasibility: Darwin arm64 Go 1.27.1 present; available Docker server is linux/arm64, not required Linux amd64.

### proof
- [ ] AC #1: pending approved RED inventory and exact missing interface definitions.
- [ ] AC #2: pending real native guardian proof.
- [ ] AC #3: pending Darwin arm64 and Linux amd64 native execution.
- [ ] AC #4: pending approved negative custody tests.
- [ ] AC #5: pending approved cumulative-budget tests.
- [ ] AC #6: pending downstream CLI/contributor consumer wiring.

## History
- 2026-09-06T09:09:57Z dep_added: blocked_by MAC-l7m0
- 2026-09-06T09:09:57Z dep_added: blocks MAC-cn7q
- 2026-09-06T09:09:58Z dep_added: blocks MAC-6h0s
- 2026-09-06T09:10:14Z dep_added: blocks MAC-vx24
- 2026-09-06T09:10:15Z dep_added: blocks MAC-ou97
- 2026-09-06T10:30:30Z dep_removed: was_blocked_by MAC-l7m0
- 2026-09-06T10:33:11Z status: open -> in_progress
- 2026-09-06T10:33:11Z auto-follows: linked to predecessor MAC-l7m0
- 2026-09-06T10:33:11Z claimed by dev-MAC-qlw2
- 2026-09-06T12:04:53Z dep_added: blocked_by MAC-p9wm

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-cn7q]], [[MAC-6h0s]], [[MAC-vx24]], [[MAC-ou97]]
- Blocked by: [[MAC-p9wm]]
- Was blocked by: [[MAC-l7m0]]
- Follows: [[MAC-l7m0]]

## Comments

### 2026-09-06T09:16:29Z ramirosalas
CANONICAL MACHINE-READABLE BOUNDARY MAP 2026-09-06
This repeats the existing ownership/signatures in the parser-supported form; it does not create additional scope or weaken any AC. All prior exact acceptance/testing requirements and holds remain current.

PRODUCES:
- internal/processscope/scope.go -> internal/processscope -> Open(context.Context, Options) (Scope, error); Join(context.Context, Capability) (Scope, error); ServeInternal(args []string, io InternalIO) (handled bool, exitCode int). Scope methods Run(context.Context, Command, Streams) (Result, error), Child(context.Context) (Scope, error), Attach(Command) (Command, error), Close(context.Context) (CleanupReport, error). Exact Command/Streams/Result/Options/ResourceState fields and capability semantics are the approved section 8 contract.
- internal/processscope/broker.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/processscope/guardian_unix.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/processscope/internal.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/processscope/scope_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/processscope/custody_integration_test.go -> owned bounded artifact; behavior and tests specified in the current story AC

CONSUMES:
- MAC-l7m0: docs/test-assurance-contract.md
  MAC-6h0s: Section 8 complete native scope/broker/guardian contract; section 3 cumulative cleanup budgets.
  schema: Section 8 complete native scope/broker/guardian contract; section 3 cumulative cleanup budgets.
- MAC-l7m0: docs/test-assurance-contract.md
  schema: Exact approved public contract SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8; accepted delivery required, never an uncommitted external proposal.

Observable outcome: A caller receives cleaned ownership evidence only after its actual native children are terminated and reaped; unsafe or unowned cleanup attempts return a blocking error.

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

### 2026-09-06T09:22:34Z ramirosalas
SCHEMA MARKER INSERTION AUDIT 2026-09-06
Root-authorized supported nd edit inserted ONLY 1 valid indented schema signature line(s) into the newly authored canonical CONSUMES block. Every original byte/contract/status/evidence/history was preserved; no deletion/replacement. This repairs the mechanical label substitution, not the contract values.
Before raw Body SHA256: f50313562910475bb9c4e33ec34284f84f1a2f11906a575e2998d2dcbb9eabeb
After insertion-only raw Body SHA256 (before this audit comment): 1eaa76f0738bdf9fffb11e3efb985df42fa194d321c66c7859a3e04cba717f7c
Exact inserted lines (zero-based original Body line positions shown):
- after Body line 101: "  schema: Section 8 complete native scope/broker/guardian contract; section 3 cumulative cleanup budgets."
Read-back pvg nd show Body exactly equals prior Body plus these insertions. Editor required expected hash, count, exact target and 13-entry total; installed pvg source revision c0957106a81346033d7b1d82fde5f434a9db6bab confirms scanner checks every historical entry.

## nd_contract
status: new

### evidence
- Signature syntax corrected via supported guarded editor; exact before/after evidence above.
- No implementation/native proof; independent Anchor and canonical document holds remain.

### proof
- [ ] All current story ACs remain pending without weakening.

### 2026-09-06T09:48:05Z ramirosalas
ROUND-1 ADVISORY: AUTHORITATIVE SECTION REFERENCE CORRECTION
All current ownership, exact APIs/fields/limits and ACs are unchanged. Earlier references calling the closed schema/store/budget/registration material section3 or sections2-3 are superseded: these definitions are in section4 of the exact approved docs/test-assurance-contract.md projection SHA25622b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8. Section2 is component dependency direction; section3 is authoritative obligations/activation; section7 is the public internal API; section8 is custody. Preserve correct obligation/activation references to section3. This corrects cross-reference navigation only, not normative semantics or any historical evidence.

## nd_contract
status: new

### evidence
- Independent round1 findings repaired through supported append-only scope/consumer notes; no architecture/source/test/ref/runtime mutation.
- Exact source surfaces verified against accepted epic7e36f3e7ddcf25565d5d4fe60b328df254eee91d; prior Body/status/labels/history preserved.
- Independent Anchor ROUND2 and canonical-document acceptance remain prerequisites, not implementation evidence.

### proof
- [ ] All current story ACs, strengthened ownership/current-judgment requirements and protected frozen proof remain required.
- [ ] Independent review/native execution/final acceptance pending.

### 2026-09-06T10:45:29Z ramirosalas
BEFORE-RED CHECKPOINT ORDER CORRECTION 2026-09-06
The earlier proposal checkpoint was inserted in Notes before historical Comments. This true-EOF comment is authoritative; prior records are preserved. Proposal /tmp/MAC-qlw2-before-red-proposal.md SHA256 6b1fdf8f88751ffaeced19b95ca70832aca37f50d3881286779ca8721fa82ad2 is not approved RED. No source/test bytes or native custody evidence were produced. Independent boundary/method adjudication is active: distinguish genuine cross-package constructor gaps from permitted private implementation choices. Native Linux amd64 remains required at delivery, not a waiver or a barrier to permitted Darwin work.

## nd_contract
status: in_progress

### evidence
- Canonical status and claim remain in_progress / dev-MAC-qlw2.
- Twelve proposed test headings are planning only, not executed tests or frozen proof.
- Root could not resume the completed metadata worker because the collaboration tool returned agent thread limit reached; root applies only this coordination correction.

### proof
- [ ] AC #1–5: exact method approval, implementation, adversarial calibration and required native proof remain pending.
- [ ] AC #6: producer same-binary helper plus real scope API consumer conformance remains required; downstream normal CLI wiring is not this producer's acceptance claim.


### 2026-09-06T12:05:48Z ramirosalas
APPROVED ARCHITECTURE / PUBLICATION HOLD

Terminal V4 architecture is approved with BLOCKING none: proposal SHA256 `d56d0104e965cace70e89b3be383ff3f51377c33ea8227c86833ef731be331be`; independent review SHA256 `dfa09b70d599a83d8cde10ab31d19fe5debd1eadf4c19f86c3aea79c0a9ac1e3`. No fourth architecture review is requested. MAC-p9wm is the new docs-only P0 owner for the standalone public native-custody contract and minimum accepted-contract companion notice; MAC-qlw2 now depends on its accepted publication before source or RED work.

CONSUMES:
- MAC-p9wm: docs/native-custody-contract.md
  schema: accepted public constructors/types, acquisition-versus-work lifetime, opaque handle/sanitation/cumulative-budget rules, Docker runtime descriptor/private client, finite contributor/checker profiles, conservative unresolved-create cleanup, and live-first proof/trust contract.
- MAC-p9wm: docs/test-assurance-contract.md
  source: accepted companion/refinement link preserving the original 575-line body outside its minimum insertion.

This checkpoint does not name future authors, freeze an exact RED inventory, authorize source/tests/calibration variants, claim native proof, or alter the existing claim. Required native Linux amd64 and Darwin arm64 delivery evidence remains unproved. All qlw2 source/test authority stays held until publication acceptance and the separately required exact before-RED inventory/actor review.

## nd_contract
status: in_progress

### evidence
- MAC-p9wm added as an explicit blocker while the existing `dev-MAC-qlw2` claim remains intact.
- V4 approval and publication prerequisite recorded without source/test/runtime/ref mutation.

### proof
- [x] Approved architecture is identified without requesting another architecture loop.
- [ ] Public contract acceptance, exact author/inventory gate, implementation, and native proof remain pending.

