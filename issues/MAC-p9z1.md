---
id: MAC-p9z1
title: "Retain exact replay inputs outside governed sources"
status: open
priority: 1
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T09:00:50Z
created_by: ramirosalas
updated_at: 2026-09-06T09:00:50Z
content_hash: "sha256:efc83eee51091f8995b8a84e8d26df9c4a3155b16b8f241e2e76e5421d765ca4"
blocked_by: [MAC-6h0s]
blocks: [MAC-62s6]
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

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-62s6]]
- Blocked by: [[MAC-6h0s]]

## Comments
