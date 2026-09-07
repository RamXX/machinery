---
id: MAC-p9z1
title: "Retain exact replay inputs outside governed sources"
status: in_progress
priority: 1
type: feature
labels: [hard-tdd, accepted]
parent: MAC-ui8a
created_at: 2026-09-06T09:00:50Z
created_by: ramirosalas
updated_at: 2026-09-07T10:08:34Z
content_hash: "sha256:7518e4b6b1ba08d9d1115de24fd65fbf13d421db7f0c298ac7f1a8c3beb24a8a"
blocks: [MAC-62s6, MAC-wbxq, MAC-vx24, MAC-ou97]
was_blocked_by: [MAC-6h0s]
follows: [MAC-6h0s]
assignee: dev-MAC-p9z1
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
- internal/tdd/bundle.go -> internal/tdd -> Capture(ctx context.Context, req CaptureRequest) (BundleRef, error); Status(ctx context.Context, req StatusRequest) (StatusReport, error); exact approved external store initialization/export/import and immutable object transport used by CLI. Store/head/bundle schemas are section 3, no subprocesses or successful execution claim.
- internal/tdd/store.go -> owned artifact for the same bounded contract
- internal/tdd/capture.go -> owned artifact for the same bounded contract
- internal/tdd/status.go -> owned artifact for the same bounded contract
- internal/tdd/bundle_test.go -> owned artifact for the same bounded contract
- internal/tdd/store_test.go -> owned artifact for the same bounded contract
- internal/tdd/store_transport_test.go -> owned artifact for the same bounded contract

### CONSUMES
MAC-6h0s: internal/tdd/manifest.go
  spec: LoadPlan(design string) (Plan, error); LoadManifest(path string) (Manifest, error); Validate(plan Plan, manifests []Manifest, inventory Inventory) error; InputView with nonnil Revalidate/Release callbacks and exact StatusReport/BundleRef fields.

### ACCEPTANCE CRITERIA
1. Capture verified immutable InputView sources plus separately bound control/judgment identities; all non-subject files are frozen by default, including ignored/untracked helpers, fixtures, dependency locks, build/test configuration, directories and empty directories. Preserve exact bytes and permission bits; whitespace or executable-mode changes are never equivalent frozen identity.
2. Implement exact machinery.tdd.tree/v1 canonical sorted UTF8 path encoding including kind, U64 lengths, U32 modes, content digest and role; reproduce digest across platforms. Reject escaping links, aliases/case collisions, special files, overlapping source/control/store roots, frozen descendants under mutable subject directories and topology/mode races. Only explicitly approved control exclusions apply; existing full-root Gv policy remains unchanged.
3. Initialize only a new external 0700 store path with machinery.tdd.store/v1 identity and durable generation-zero machinery.tdd.head/v1 (null previous, empty arrays). Store must lie outside all governed/source/frozen/dependency/Gv trees. Reject adoption of existing nonempty/partial/mismatched project paths and environment-based store discovery.
4. Make content objects immutable, read-verified and replay-retained with bounded inventories. Export to a new archive/import to a new destination verifies canonical head chain, all referenced objects, exact modes/topology and failed histories; malformed, missing, oversized or known local rollback state fails. Import does not certify replay or universal newest state.
5. Status is cheap, read-only and explicitly replay-not-performed: independent StoreProjectID/StoreHeadDigest/SourceDigest/ControlDigest/JudgmentDigest dimensions and actionable diagnostics distinguish missing/unregistered/draft/stale/historical evidence. No hash-only receipt becomes proof tests ran.
6. Real filesystem integration exercises capture -> materialize -> byte/mode/empty-directory round trip, corrupt objects, stale control/judgment/head, failed-history transport, concurrent reader/writer pressure, permissions and close/fsync failures. Every operation is bounded by approved 600000ms non-execution owner and single 10000ms cleanup; failure leaves no success assertion and never retries automatically.

### TEST COMMANDS
Run targeted native package tests for owned files and exact required contributor fragments; record selected test identities and exact commands in RED. Do not run heavy preflight.

### OUT OF SCOPE
Explicit plan registration/CAS advancement is the registration consumer; Execute/Verify execution records and final sealing are downstream. This story cannot automatically register or certify tests.

### DIFF BUDGET
~9 files, under 2300 changed LOC including tests.

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

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-06T09:09:58Z dep_added: blocked_by MAC-6h0s
- 2026-09-06T09:09:59Z dep_added: blocks MAC-62s6
- 2026-09-06T09:10:08Z dep_added: blocks MAC-wbxq
- 2026-09-06T09:10:17Z dep_added: blocks MAC-vx24
- 2026-09-06T09:10:18Z dep_added: blocks MAC-ou97
- 2026-09-07T01:42:40Z dep_removed: was_blocked_by MAC-6h0s
- 2026-09-07T09:15:42Z status: open -> in_progress
- 2026-09-07T09:15:42Z auto-follows: linked to predecessor MAC-6h0s

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-62s6]], [[MAC-wbxq]], [[MAC-vx24]], [[MAC-ou97]]
- Was blocked by: [[MAC-6h0s]]
- Follows: [[MAC-6h0s]]

## Comments

### 2026-09-06T09:16:30Z ramirosalas
CANONICAL MACHINE-READABLE BOUNDARY MAP 2026-09-06
This repeats the existing ownership/signatures in the parser-supported form; it does not create additional scope or weaken any AC. All prior exact acceptance/testing requirements and holds remain current.

PRODUCES:
- internal/tdd/bundle.go -> internal/tdd -> Capture(ctx context.Context, req CaptureRequest) (BundleRef, error); Status(ctx context.Context, req StatusRequest) (StatusReport, error); exact approved external store initialization/export/import and immutable object transport used by CLI. Store/head/bundle schemas are section 3, no subprocesses or successful execution claim.
- internal/tdd/store.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/tdd/capture.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/tdd/status.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/tdd/bundle_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/tdd/store_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- internal/tdd/store_transport_test.go -> owned bounded artifact; behavior and tests specified in the current story AC

CONSUMES:
- MAC-6h0s: internal/tdd/manifest.go
  spec: LoadPlan(design string) (Plan, error); LoadManifest(path string) (Manifest, error); Validate(plan Plan, manifests []Manifest, inventory Inventory) error; InputView with nonnil Revalidate/Release callbacks and exact StatusReport/BundleRef fields.
- MAC-l7m0: docs/test-assurance-contract.md
  schema: Exact approved public contract SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8; accepted delivery required, never an uncommitted external proposal.

Observable outcome: A user can retain exact immutable replay inputs externally and receive truthful freshness status; tampered bytes or topology return a blocking error.

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

### 2026-09-06T09:48:06Z ramirosalas
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

### 2026-09-07T10:08:34Z ramirosalas
ACCEPTED 2026-09-07 — RED c4044f2 (29 subjects semantic vs refusing stubs; frozen SHAs recorded) + b28707d fixture repairs (3 unsatisfiable corrected, justified) -> GREEN baa0429. machinery.tdd.tree/v1 exact encoding vs independent oracle; durable gen-0 0700 content-addressed store outside all governed roots (canonical non-lexical checks); immutable read-verified objects, closed MTDDARCV export, validated atomic import preserving failed history (recorded-only never replayed); head-chain tamper/rollback fail-closed; read-only replay-not-performed status with 5 independent digests; concurrency -race clean; cancellation no-retry; budgets honored. 34/34 pass; vet/build clean; GOOS=linux build clean. Overrun 11 files/4957 LOC vs ~2300 — closed-schema decode + mandated breadth (established pattern); 8 own-RED fixture amendments justified. RESIDUALS: fsync-EIO injection infeasible (permission-bit failures used); capture defers project binding to Status; NegativeSensitivity=missing until replay exists (downstream); watch: windows toolchain breakage in processscope (not this path). Coordinator merged; tdd suite + lane green on epic. Record: .git/machinery-evidence-20260906.TEFZ7D/p9z1-record.md
