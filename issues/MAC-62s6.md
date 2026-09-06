---
id: MAC-62s6
title: "Register reviewed assurance revisions explicitly"
status: open
priority: 1
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T09:00:51Z
created_by: ramirosalas
updated_at: 2026-09-06T09:00:51Z
content_hash: "sha256:751824fe8f13f4061adb0147051f3ca2632eaf6e5f67d367061b2eedec0b0c3b"
blocked_by: [MAC-p9z1]
blocks: [MAC-sqpt, MAC-u4oo, MAC-vx24]
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
- internal/assuranceflow/register.go -> internal/assuranceflow -> Register(ctx context.Context, req RegisterRequest, output io.Writer) (Registration, error), RegisterRequest{Design,Implementation,Store,ExpectedHead string;Milestones []string}; private-constructed Registration with ProjectID, StoreID, PreviousHead, HeadDigest, Generation, sorted MilestoneKeys and State accessors. Exact section 7 API and section 3 registration transaction; no public constructor or execution success.
- internal/assuranceflow/register_test.go -> owned artifact for the same bounded contract
- internal/tdd/registration_store.go -> owned artifact for the same bounded contract
- internal/tdd/registration_store_test.go -> owned artifact for the same bounded contract

### CONSUMES
MAC-p9z1: internal/tdd/store.go
  MAC-6h0s: machinery.tdd.store/v1, machinery.tdd.head/v1, immutable object/head archive and external 0700 store; Capture(ctx context.Context, req CaptureRequest) (BundleRef, error), Status(ctx context.Context, req StatusRequest) (StatusReport, error).
MAC-6h0s: internal/gates/assurance_inventory.go
  spec: AssuranceInventory(design string) (tdd.Inventory, error); tdd.Validate(plan Plan, manifests []Manifest, inventory Inventory) error.

### ACCEPTANCE CRITERIA
1. Expose the exact Register API with explicit Store and ExpectedHead, sorted unique selected Milestones (repeat flags in the later CLI; omission means all, empty selection is invalid). Validate approved review/source/control/inventory graphs; children register first. First revision is 1 with null predecessor; successor is prior revision+1 and exact same-key predecessor.
2. On first use accept the exact empty generation-zero head without any prior PASS/execution receipt. Registration proves only registered-not-executed or already-registered-not-executed; no process, Gv acceptance, current correctness or successful replay is implied.
3. Build a deterministic desired successor against ExpectedHead; archive canonical exact controls/objects/head before durable CAS. Plan change must advance every registered milestone of that design; registered deletion and untargeted registered graph mismatch block. Untargeted draft controls may remain pending selected RED, but block complete.
4. Release initial read reservation, acquire writer and compare the exact expected head; validate source/control identities, release/reacquire/revalidate/release original view in the approved order before publication. CAS loser returns HEAD_CONFLICT, never rebases/retries automatically. Idempotence only when current is the exact desired successor and controls match; another writer's later head requires caller rebase.
5. Before commit, errors consume no revision; postcommit close/output failure returns no successful Registration although head may have advanced. Exact retry confirms already-registered state without erasing failed history. No Execute, Green, Verify or check operation may advance this head.
6. Real API/store integration covers fresh register, multiple milestones/children, conflicting writers, same-head retry, intervening head, source ABA, review mismatch, draft vs registered mismatches, deletion, failure before/after durable commit and failing output. Bound the whole operation to 600000ms plus one 10000ms cleanup, including locks/publication/output.

### TEST COMMANDS
Run targeted native package tests for owned files and exact required contributor fragments; record selected test identities and exact commands in RED. Do not run heavy preflight.

### OUT OF SCOPE
CLI command construction belongs to the normal CLI story; test execution/replay/final verification belong to their consumers. No registration may be inferred from a writable receipt or prior milestone acceptance.

### DIFF BUDGET
~6 files, under 1500 changed LOC including tests.

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
- 2026-09-06T09:09:59Z dep_added: blocked_by MAC-p9z1
- 2026-09-06T09:10:07Z dep_added: blocks MAC-sqpt
- 2026-09-06T09:10:11Z dep_added: blocks MAC-u4oo
- 2026-09-06T09:10:18Z dep_added: blocks MAC-vx24

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-sqpt]], [[MAC-u4oo]], [[MAC-vx24]]
- Blocked by: [[MAC-p9z1]]

## Comments
