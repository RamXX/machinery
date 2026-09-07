---
id: MAC-qlw2
title: "Retain native child ownership until cleanup completes"
status: closed
priority: 0
type: feature
labels: [hard-tdd, accepted]
parent: MAC-ui8a
created_at: 2026-09-06T08:58:17Z
created_by: ramirosalas
updated_at: 2026-09-07T00:29:10Z
content_hash: "sha256:9d48b4e063f443ea91fec4885bf091d9112b9db21bf1fe10e7a05ed81e5d5b53"
was_blocked_by: [MAC-l7m0, MAC-p9wm]
assignee: dev-MAC-qlw2
follows: [MAC-l7m0]
closed_at: 2026-09-07T00:29:10Z
close_reason: "Accepted: native child ownership until cleanup completes, proven Darwin+Linux; critical-path unblocker delivered; merged to local epic"
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
- 2026-09-06T12:31:39Z dep_removed: was_blocked_by MAC-p9wm
- 2026-09-06T17:01:33Z status: in_progress -> deferred
- 2026-09-06T23:26:32Z status: deferred -> open
- 2026-09-07T00:29:10Z status: open -> closed
- 2026-09-07T00:29:10Z dep_removed: no_longer_blocks MAC-cn7q
- 2026-09-07T00:29:10Z dep_removed: no_longer_blocks MAC-6h0s
- 2026-09-07T00:29:10Z dep_removed: no_longer_blocks MAC-vx24
- 2026-09-07T00:29:10Z dep_removed: no_longer_blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Was blocked by: [[MAC-l7m0]], [[MAC-p9wm]]
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


### 2026-09-06T12:45:35Z ramirosalas
ACCEPTED PUBLIC CUSTODY CONTRACT / AUTHORITATIVE CURRENT PROSPECTIVE SCOPE — 2026-09-06

This true-EOF amendment supersedes only the earlier six-path/2600-LOC scope forecast and the now-satisfied publication blocker. Every earlier stronger acceptance/testing constraint, failed record, frozen evidence item, claim, dependency history and native-proof obligation remains intact. This is prospective story scope and workflow authority only: it is not test implementation approval, RED approval, source authorization, native execution evidence, delivery or acceptance. Terminal V4 architecture remains approved with BLOCKING none; no fourth architecture review is requested.

ACCEPTED PRODUCER GROUNDING
- MAC-p9wm is closed/accepted. Accepted docs candidate `2a63e4bb0d7b57c42dfd6f45fecfdc4f8a18cfbf` is present in epic `2a73454d5f133a7b5fb4db0346232fd389810d28`.
- `docs/native-custody-contract.md`: Git blob `1c1581d1aec324d979593613b44976ed0007c45b`, 328 lines / 27,783 bytes, SHA256 `bee289e553809b550b1f5cedf73415f600379ab54830a11733ba0dc7caaaba2a`.
- `docs/test-assurance-contract.md`: Git blob `acce64fee8db5a7565e8fa33422334de125e913f`, 576 lines / 109,579 bytes, SHA256 `171ebaa3e992b60bd90f1fc280818a125689390f8440b3a9a7b2c97e7f8b747d`; its accepted 575-line body remains byte-identical outside the one companion link.
- Independent acceptance report: `/tmp/MAC-p9wm-pm-review.Hf8lA2/REPORT.md`, SHA256 `19c0c4be6b4eb4b7035db3e9e576352ad1cfd3b24f414e287be7d8dfded38f8e`.
- Approved architecture proposal/review: `/tmp/MAC-qlw2-supplement-v4.HaJAeo/PROPOSAL.md` SHA256 `d56d0104e965cace70e89b3be383ff3f51377c33ea8227c86833ef731be331be`; `/tmp/MAC-qlw2-anchor-v4.fH9ocw/REVIEW.md` SHA256 `dfa09b70d599a83d8cde10ab31d19fe5debd1eadf4c19f86c3aea79c0a9ac1e3`.

PRODUCES:
- internal/processscope/scope.go -> exact public custody records and APIs: InheritedInternalIO(context.Context) (InternalIO, bool, error); (InternalIO).Close() error; InheritedCapability(context.Context) (Capability, error); (Capability).Close() error; Open(context.Context, Options) (Scope, error); Join(context.Context, Capability) (Scope, error); ServeInternal([]string, InternalIO) (bool, int); Scope.Run/Child/Attach/Close; exact Limits, Diagnostic, Command, Streams, Result, Options, CleanupReport and ResourceState records.
- internal/processscope/broker.go -> live authenticated broker, root/child guardian registration and base finite dispatch required by the accepted contract; the later checker-only dispatch arm is MAC-yhg5 and may modify this file only sequentially after qlw2.
- internal/processscope/guardian_unix.go -> Unix guardian implementation with explicit build constraints for supported Darwin/Linux behavior, direct-child live identity and terminal group-kill-before-Wait/reap ordering.
- internal/processscope/internal.go -> existing untagged portable internal surface, including the unsupported-platform definition used when the Unix guardian implementation is unavailable; no false native fallback.
- internal/processscope/contributor_docker.go -> DockerRuntime request/capture/inherit/descriptor/Validate/Close API; closed ContributorDocker service, five-program registry, bounded observation protocol and read-only exact-ID InspectDockerContainer API.
- internal/processscope/scope_test.go -> exact separately frozen constructor, handle, broker, limits, guardian and portable-unsupported tests.
- internal/processscope/custody_integration_test.go -> real same-binary exported-API custody, nested scope, live-first process/container and independent foreign-survival tests.
- internal/processscope/contributor_docker_test.go -> closed Docker runtime/descriptor/service/observation/create-start-cleanup tests and calibration controls.

CONSUMES:
- MAC-p9wm: docs/native-custody-contract.md
  schema: exact accepted public constructors/Close semantics; acquisition-versus-work lifetime and error precedence; scope records/Limits/Diagnostic; authenticated broker/guardian; cumulative wall/owner/cleanup rules; DockerRuntime descriptor/private client; closed contributor profile; conservative unresolved-create lifecycle; live-first calibration and native host matrices, at SHA256 bee289e553809b550b1f5cedf73415f600379ab54830a11733ba0dc7caaaba2a.
- MAC-p9wm: docs/test-assurance-contract.md
  source: accepted companion contract preserving every non-custody obligation and linking the refinement at SHA256 171ebaa3e992b60bd90f1fc280818a125689390f8440b3a9a7b2c97e7f8b747d.

CURRENT PROSPECTIVE ACCEPTANCE / TEST WORKFLOW
1. Exactly EIGHT owned paths apply: five implementation paths (`scope.go`, `broker.go`, `guardian_unix.go`, `internal.go`, `contributor_docker.go`) and three test paths (`scope_test.go`, `custody_integration_test.go`, `contributor_docker_test.go`). Checker-specific profile implementation belongs to MAC-yhg5, not this story. All public records, defaults, absolute caps, opaque live authorities and Close/error semantics match the accepted producer exactly.
2. Root bootstrap, descendant Join, command attachment and Docker runtime/service work retain one authenticated owner. Constructors validate nil/deadline/cancellation before absence or intent, use the bounded acquisition-only lifetime, and transfer one usable handle atomically; post-handoff probe cancellation does not revoke work, while authenticated owner/operation expiry, Join cancellation and liveness loss do. No marker, path, descriptor text, PID, label or historical record creates authority.
3. The contributor service remains synchronous and closed: exact five programs only; fixed profile; observations are bounded read-only data; inspection is exact-ID/same-daemon/read-only. Actual guardian and container ownership is registered before start admission. Unresolved create/start is never retried or reported clean; cleanup failure stays distinct from target result. No checker-specific arbitrary argv/resource path is added here.
4. One cumulative milestone wall deadline and fixed owner ceiling cover queueing, nested work and retries; terminal failure enters one shared cleanup grace. Cleanup-only helpers are registered under that original grace and may perform only fixed terminal-response/identity/exact-owned inspect/remove operations. They never provision, create/start targets, replay, run tests, use Background/unscoped fallback or renew grace. CLOSING rejects new public work, and unresolved create/start can never become false-clean.
5. Required live-first proof uses the real same-binary exported API and independent same-daemon observer before the trigger. Per-invariant unsafe/safe pairs prove normal completion, intended assertion failure, host output overflow, timeout, SIGINT/SIGTERM, owner loss, registration/admission races, early parent exit, nested Join, malformed/stale/forged/cross-scope/closed/reused handles and conservative daemon ambiguity. Exact owned absence and helper/guardian reaping precede success; independently owned foreign process/container controls survive. Historic PID/name/labels never authorize cleanup.
6. Native Linux amd64 and Darwin arm64 must run the same production candidate and frozen inventory before acceptance. Linux amd64 remains pending and is not waived by an amd64 image, arm64 daemon, cross-build or architecture approval; its absence is not a barrier to permitted Darwin planning and authoring under the gates below.
7. DIFF BUDGET: exactly 8 files, forecast 3,800–6,200 changed LOC. This is an investigated forecast, not a trim target. Any overrun triggers explicit PM investigation; required tests/contracts are not weakened or deleted to fit it.
8. All original stronger constraints remain: standalone Machinery with private pvg/nd used only for development coordination; no product dependency; no remote/install/preflight/unrelated cleanup; no audit exception, disposition, successor waiver or edge change; developer delivers and an independent PM accepts.

BEFORE-FIRST-EDIT HARD-TDD HOLD / ACTIONABLE NEXT
- Current owner remains the existing `dev-MAC-qlw2` claim. The accepted publication removes only the former MAC-p9wm blocker; it does not release this before-edit hold. Root-reported branch base remains old epic `a94e768adf461178e0562ad135c7c06d8163a3a4` with no source/test work yet.
- NEXT: the current developer must return a separate prospective RED-author package BEFORE touching source/test bytes: exact leaf/helper/fixture/config/dependency-lock inventory; expected-failure plus passing-control matrix; same-binary real helper and independent-live-first method; exact source base; and the actual selected RED author handle. Do not invent or freeze test names in this PM amendment.
- Before the first RED/test authoring dispatch, record the actual RED-author handle and a DIFFERENT actual calibration-implementation-author handle. Select the independent PM handle before that independent review. Select the separate actual production GREEN-author handle only after independent RED approval and before GREEN dispatch; a future GREEN handle is not an artificial prerequisite for initial RED authoring. The four role semantics remain distinct and no speculative handle is approved here. Tests/helpers/fixtures/config freeze first; only then may the calibration author modify copies of the five implementation paths. Pre-inventory planning may occur before the later reviewer/GREEN roles are selected, but no source/test bytes may be written before the applicable before-edit gate.
- State roles are fixed prospectively: S0 is the absent-API/setup state and is retained honestly but is NOT custody RED; S1 is a new operative unsafe implementation-only reference; S2 is its safe counterpart; additional S1-x/S2-x pairs cover distinct invariants; G is later production authored only after independent actual RED approval. Variant implementation bytes freeze before replay. Tests/config/dependency bytes are identical across calibration states. A safe reference never substitutes for G and production G cannot be transplanted from it.
- If the canonical workflow truly cannot admit prospective S1/S2 calibration, stop and escalate or create a legitimate independently reviewed construction foundation/current revision. Never launder S0 setup failure, historical evidence or a later safe implementation into RED.

## nd_contract
status: in_progress

### evidence
- Accepted public producer and terminal V4 architecture hashes are fixed above; all named artifacts were read completely and hash-checked.
- Canonical repair is tracker-only and preserves the existing claim/status/dependency history. No source/test/ref/worktree/runtime/native/preflight/remote/install action occurred.

### proof
- [x] Publication prerequisite is satisfied by accepted MAC-p9wm without claiming implementation.
- [x] Prospective eight-path ownership, budget, producer/consumer map and calibration roles are bounded.
- [ ] Exact prospective RED inventory plus actual distinct RED/calibration handles must be recorded before first RED/test authoring; independent PM and separate production GREEN handles are selected only at their later applicable gates.
- [ ] Implementation, actual RED/GREEN and same-candidate Darwin arm64 plus Linux amd64 native proof remain pending.


### 2026-09-06T14:59:08Z ramirosalas
PRE-EDIT METHOD CHECKPOINT ONLY. Frozen V2 inventory remains unapproved. Independent report /tmp/MAC-qlw2-inventory-v2-review.e5Dw60/REPORT.md SHA256 a392e384a104ed3502a13f8ee095cf1de6fdc05a68a7c792f5984314bea2a385 and immutable ADDENDUM.md SHA256 4a2f5b12ecab3c329f475732eae0b0dca8249ddd90b9d42caa8dd8246f392c51 are GAPS_FOUND, fully read and verified by root. Eight finite method corrections plus disjoint internal-service versus capability-attachment dispatch are assigned to existing /root/custody_red_inventory in a new external V3 package. No source/test authoring or RED authority. H01 external-copy calibration requires prospective actual distinct actor/chronology amendment; forecast 4900–8100 LOC is investigated but not yet canonically substituted. Exact future eight-path ownership and existing claim/status/DAG remain unchanged. No fourth architecture loop, audit exception, native acceptance, preflight, installed replacement or remote action.

### 2026-09-06T15:37:03Z ramirosalas
PRE-EDIT V3 REVIEW CHECKPOINT ONLY. Independent /tmp/MAC-qlw2-inventory-v3-review.mVxvsL/REPORT.md SHA256 e151d0bc05895dd75487627d517994d2b8bd2c63b6009fe6442b975277146b87 GAPS_FOUND: scalar bounds masked by stricter validity, unrealizable H01c constructor claim, P07e nontarget concurrency, aggregate execution budgets, and ESRCH-before-Wait ordering. Root fully read111-line report and13-line ADDENDUM.md SHA256 365d110d5795677da39cc2bcf81f1d13fe4adba2d32f8ef9707447542c48bd5d. Addendum approves simpler prospective sequence only: actual actors/forecast/external-copy H01 authority beforeedit; RED author alone freezes and commits exact tdd-red R0/setup (NOTREDapproval); cal onlythen authors external safe implementation/H01mutants/restores committedtesthashes; productS1/S2 andactualREDreview follow. No pre-freeze calibration-source authority or audit exception. External V4 correction assigned existing RED inventory author; no source/test/native permission. Existing claim/status/DAG/eightpaths unchanged; independent finalmethod approval and actual distinctcalibration appointment/canonical scope amendment remain gates.

### 2026-09-06T15:46:58Z ramirosalas
EXECUTION BLOCKER CHECKPOINT. The delegated external V4 inventory correction by /root/custody_red_inventory was stopped by platform safety controls before a candidate was delivered. Root will not route around that restriction. Frozen V3 GAPS_FOUND report/addendum and all prior evidence remain authoritative; no V4 approval, first-edit authority, source/test/calibration/native execution or acceptance is claimed. Existing claim/status/dependencies stay unchanged; no recovery/release or audit exception. Independent portfolio work may finish separately. Canonical distinct calibration appointment, approved corrected method, prospective H01/forecast authorization, native Linux executor and final gates remain pending.

### 2026-09-06T17:01:55Z ramirosalas
USER-TABLED 2026-09-06: user explicitly asked to table this issue and continue independent areas. Supported pvg nd defer MAC-qlw2 changed only scheduling status to deferred; prior claim, hard-TDD label, dependencies, branch and failed/review history retained. This is not completion, cancellation, test approval, or waiver of the final assurance requirement. No new custody planning/source/test execution assigned. Consumers remain blocked. Resume only when the owner reopens this issue; independent work may continue without claiming full custody/final assurance.

### 2026-09-07T00:29:09Z ramirosalas
ACCEPTED 2026-09-06 — internal/processscope native custody implemented per test-assurance-contract s8+s3. RED f585213 (28/29 intended semantic failures). GREEN Darwin arm64 go1.27.1: 29/29 ok 10.759s zero skips; -race ok 32.158s zero races; vet/build/gofmt clean. Linux amd64 (authorized host, isolated owned root, GOENV/GOTELEMETRY=off): 29/29 ok 10.97s, vet pass, source invariance byte-identical; evidence bundle 33c1faf9..., REPORT 41cb172f.... RESIDUALS (honest): Linux -race not executable (host has no C compiler; race coverage stands on Darwin only); full 'go build .\/...' under GOPROXY=off needs out-of-lane modules (scoped build passed; full build at final gate); container cleanup is registration/reporting (daemon wiring downstream per contract). Diff 4,408 vs ~2,600 budget — within the V4-approved 3,800-6,200 envelope for this scope; 6 disclosed [test-edit-authorized] RED-defect repairs, all strengthening. AC map in .git/machinery-evidence-20260906.TEFZ7D/qlw2-record.md. Coordinator merged + epic suites ok.
