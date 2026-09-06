---
id: MAC-6h0s
title: "Reject incomplete executable assurance declarations"
status: open
priority: 1
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T08:58:18Z
created_by: ramirosalas
updated_at: 2026-09-06T08:58:18Z
content_hash: "sha256:19c09abc2c3d1e915328f3039ba926566cf36d3bac4bf7b49df9c86ac429e8c6"
blocked_by: [MAC-qlw2]
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
- internal/tdd/manifest.go -> internal/tdd -> LoadPlan(design string) (Plan, error); LoadManifest(path string) (Manifest, error); Validate(plan Plan, manifests []Manifest, inventory Inventory) error. Exact request/response/Adapter/CheckExecutor record types of section 7. internal/gates/assurance_inventory.go -> AssuranceInventory(design string) (tdd.Inventory, error). Public aliases may use internal/tdd/protocol to avoid an import cycle without changing the approved API.
- internal/tdd/types.go -> owned implementation/test artifact for the same contract
- internal/tdd/manifest_test.go -> owned implementation/test artifact for the same contract
- internal/tdd/protocol/types.go -> owned implementation/test artifact for the same contract
- internal/gates/assurance_inventory.go -> owned implementation/test artifact for the same contract
- internal/gates/assurance_inventory_test.go -> owned implementation/test artifact for the same contract
- testdata/assurance/schema-cases.json -> owned implementation/test artifact for the same contract

### CONSUMES
MAC-qlw2: internal/processscope/scope.go
  spec: Scope interface in typed requests; no process is launched by declaration validation.
MAC-l7m0: docs/test-assurance-contract.md
  schema: Sections 2-3 exact closed plan/milestone JSON, qualified obligation/test identity, limits, review-digest projection; section 7 Go signatures.

### ACCEPTANCE CRITERIA
1. Implement closed versioned plan and milestone decoders and complete exact typed records; reject unknown/duplicate keys, wrong type/null position, invalid revisions/digests, duplicate identities and unsupported adapter/framework/runtime labels. Only go-testing/v1, node-test-typescript/v1, python-unittest/v1, elixir-exunit/v1 are first-release adapters.
2. Derive AssuranceInventory from a held immutable design snapshot, using qualified {design,kind,owner,id} keys for oracle-row, guard-clause, invariant and runtime obligations. Every oracle/guard/invariant is assigned to at least one current milestone; same text ID in another child/owner is not evidence. FullTestRef includes design,milestone,suite,test; leaf/milestone deletion and owner pooling cannot reduce requirements.
3. Require all seven approved runtime/NFR categories to have explicit test, bound-reviewed not-applicable, or unverified dispositions; unverified blocks strict completion. Validate manifest obligation mappings, reviewed source anchors/digests, acyclic parent/child references and no generic runtime row hiding missing categories.
4. Require red_controls for EVERY expected baseline assertion failure and safe/unsafe sensitivity pairs for every negative test: same qualified test/assertion and frozen closure, distinct subject digest, exact expected target outcomes and unchanged non-target outcomes, public-boundary coverage per milestone. All registered assertions must be reached; early abort cannot satisfy inventory.
5. Implement exact review digest projection, ignoring only specified review fields at specified positions; no recursive ignore-by-key. Full reviewed control bytes remain independently captured. Path roles honor RootPath '.', exact control namespace, no absolute/traversal/alias/symlink/device acceptance.
6. Validate exact defaults and caps: per-milestone wall default 600000ms max3600000ms; cleanup10000ms max30000ms; closed event/output/bundle/count/depth/jobs caps from contract, with one cumulative budget not per-command resets. Decoder arithmetic overflow/duplicate graph, unsafe case aliases, missing suite controls and unsupported modes fail with deterministic diagnostics.
7. Real integration loads valid and deliberately corrupted design+manifest trees through AssuranceInventory and Validate, rather than testing only hand-built Inventory structs. This declaration producer is consumed by capture/register/replay and normal gates; it does not issue execution evidence.

### TEST COMMANDS
Run targeted native package tests for the owned implementation and the specifically inventoried required contributor lane fragments; record the exact selected leaf inventory and command in RED. No heavy preflight during this story.

### OUT OF SCOPE
Immutable bundle/store persistence and actual runner execution are separately owned downstream; no receipt file can substitute for those consumers.

### DIFF BUDGET
~8 files, under 2100 changed LOC including tests.

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
- [ ] AC #7: executable evidence pending.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-06T09:09:58Z dep_added: blocked_by MAC-qlw2

## Links
- Parent: [[MAC-ui8a]]
- Blocked by: [[MAC-qlw2]]

## Comments
