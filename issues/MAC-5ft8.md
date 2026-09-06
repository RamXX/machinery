---
id: MAC-5ft8
title: "Migrate Go CRM to prospective strict assurance"
status: open
priority: 1
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T09:07:15Z
created_by: ramirosalas
updated_at: 2026-09-06T09:16:34Z
content_hash: "sha256:8a69f7c31d614e4de24e4e892794c6c3ad57097d7d7cc2deb27789879fff4dcb"
blocked_by: [MAC-u4oo]
blocks: [MAC-al5u, MAC-vx24, MAC-ou97]
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
- examples/go-crm/design/assurance/plan.json -> examples/go-crm/design/assurance -> NEW prospective registered-plan/milestone manifests and local strict assertion helpers/tests with complete required-test/obligation mapping; required contributor fragment runs legacy regressions independently plus new strict replay. No historical RED/acceptance rewrite.
- examples/go-crm/design/assurance/milestones -> owned artifact for the same bounded contract
- examples/go-crm/impl/internal/assurancetest -> owned artifact for the same bounded contract
- examples/go-crm/impl/internal/authz/strict_assurance_test.go -> owned artifact for the same bounded contract
- examples/go-crm/impl/internal/domain/strict_assurance_test.go -> owned artifact for the same bounded contract
- examples/go-crm/impl/internal/session/strict_assurance_test.go -> owned artifact for the same bounded contract
- examples/go-crm/impl/internal/cli/strict_assurance_test.go -> owned artifact for the same bounded contract
- examples/go-crm/impl/internal/repo/strict_assurance_test.go -> owned artifact for the same bounded contract
- testdata/integration-lanes/assurance-go-crm.json -> owned artifact for the same bounded contract

### CONSUMES
MAC-u4oo: cmd/machinery/tdd.go
  endpoint: machinery tdd store init/scaffold/capture/register/red/green/verify and machinery check --impl <path> --store <path> --complete, exact approved flags.
MAC-wi2u: internal/tdd/adapters/go.go
  spec: go-testing/v1 closed Adapter; native bound helper assertions, no direct t.Fatal compatibility claim.
MAC-hgz1: examples/go-crm/design/BUILD.md
  source: accepted prospective v2 BUILD/evidence/golden migration; its 41-path scope does not authorize strict test rewrites.
MAC-sh60: internal/gates/oraclecov.go
  spec: CheckOracleCoverage(design, impl string) *Gate; preserve its independent story/hold and all accepted oracle coverage regressions.

### ACCEPTANCE CRITERIA
1. Inventory every Go CRM current milestone, oracle/guard/invariant/runtime obligation and every existing required regression. Create an explicit prospective assurance revision with a total mapping to new strict native tests; retain original approved/failing RED, accepted history and existing required tests byte-for-byte.
2. Existing direct t.Fatal/native tests remain independently required in the contributor lane until properly migrated under separate approval. They are not silently selected away, relabeled as strict or rewritten under this story; old native PASS is not new strict evidence. New tests/helper/config get their own reviewed RED freeze and assertion-specific safe/unsafe calibration.
3. Use actual public Go CRM behavior and real supported fixture paths for each new assertion. No mocked repository as proof of actual persistence behavior; unsupported required storage/CGO/service/credential/runtime needs are explicit blockers requiring architecture review, never replaced with fake success or a weakened complete label.
4. Execute fresh external store initialization, capture retained baseline/control/unsafe/current variants, reviewed explicit registration, actual RED/GREEN/replay and complete on the current implementation. Every new expected baseline failure has matching red_control and every negative pair fails the correct assertion; tests/helpers/config/modes/topology remain identical.
5. Bind reviewed runtime residual dispositions for race/replay/migration/restore/load and every other required category to named tests or bound not-applicable review; unsupported/unverified guarantees block complete. Historical ancestor acceptance stays separate from current implementation assurance.
6. Required native two-platform fragment runs BOTH all legacy native regressions and new strict assurance, with exact case inventory and no skips/empty/fake/missing-runtime success. New manifest/control data contains no absolute store paths, private tracker metadata, secrets or fabricated imported PASS.
7. Demonstrate unchanged required regression inventory before/after migration, real failed unsafe variants, green safe controls and frozen-byte hashes. If the mapping cannot be completed within this scope, stop for reviewed decomposition, never drop a test or reclassify Go CRM as design-only.

### TEST COMMANDS
Run targeted native package tests for owned files and exact required contributor fragments; record selected test identities and exact commands in RED. Do not run heavy preflight.

### OUT OF SCOPE
No mutation or supersession of MAC-sh60/hgz1/lhu5/lnu6 claims, tests or scope. Complete-example CI invocation wiring is the next consumer; contributor tests are not forced to masquerade as strict consumer manifests.

### DIFF BUDGET
~16 new test/manifest/helper files, under 2300 changed LOC; investigate before broadening.

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
- 2026-09-06T09:10:12Z dep_added: blocked_by MAC-u4oo
- 2026-09-06T09:10:12Z dep_added: blocks MAC-al5u
- 2026-09-06T09:10:29Z dep_added: blocks MAC-vx24
- 2026-09-06T09:10:30Z dep_added: blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-al5u]], [[MAC-vx24]], [[MAC-ou97]]
- Blocked by: [[MAC-u4oo]]

## Comments

### 2026-09-06T09:09:30Z ramirosalas
DEPENDENCY CLARIFICATION 2026-09-06
This prospective migration consumes the delivered normal CLI and existing current oracle-coverage behavior; it does NOT require a new direct dependency on MAC-sh60 and does not authorize a successor, cancellation, supersession or dependency transfer for that held story. The earlier MAC-sh60 API reference is existing integration context only, not a new producer requirement. Its original MAC-vx24/MAC-ou97 final-integration edges remain intact and those final gates rerun the complete migrated example against its eventual accepted coverage behavior. Ordering after MAC-hgz1 is already enforced transitively through MAC-u4oo -> MAC-rau8 -> MAC-wbxq -> MAC-lnu6 -> MAC-hgz1; no redundant new direct edge to hgz1.

CURRENT CONSUMES:
- MAC-u4oo: cmd/machinery/tdd.go
  endpoint: exact standalone store init/scaffold/capture/register/red/green/verify/check --complete commands.
- MAC-wi2u: internal/tdd/adapters/go.go
  spec: closed go-testing/v1 Adapter and bound native assertion helper.
- MAC-hgz1: examples/go-crm/design/BUILD.md
  source: accepted v2 example contract, ordered transitively; no alteration of its healthy claim or frozen files.

## nd_contract
status: new

### evidence
- Dependency-only clarification preserves every migration AC and all original coverage-story authority.
- No implementation or native execution proof claimed; independent Anchor/l7 holds remain.

### proof
- [ ] AC #1-7: original prospective migration obligations remain pending.

### 2026-09-06T09:16:34Z ramirosalas
CANONICAL MACHINE-READABLE BOUNDARY MAP 2026-09-06
This repeats the existing ownership/signatures in the parser-supported form; it does not create additional scope or weaken any AC. All prior exact acceptance/testing requirements and holds remain current.

PRODUCES:
- examples/go-crm/design/assurance/plan.json -> examples/go-crm/design/assurance -> NEW prospective registered-plan/milestone manifests and local strict assertion helpers/tests with complete required-test/obligation mapping; required contributor fragment runs legacy regressions independently plus new strict replay. No historical RED/acceptance rewrite.
- examples/go-crm/design/assurance/milestones -> owned bounded artifact; behavior and tests specified in the current story AC
- examples/go-crm/impl/internal/assurancetest -> owned bounded artifact; behavior and tests specified in the current story AC
- examples/go-crm/impl/internal/authz/strict_assurance_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- examples/go-crm/impl/internal/domain/strict_assurance_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- examples/go-crm/impl/internal/session/strict_assurance_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- examples/go-crm/impl/internal/cli/strict_assurance_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- examples/go-crm/impl/internal/repo/strict_assurance_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- testdata/integration-lanes/assurance-go-crm.json -> owned bounded artifact; behavior and tests specified in the current story AC

CONSUMES:
- MAC-u4oo: cmd/machinery/tdd.go
  endpoint: machinery tdd store init/scaffold/capture/register/red/green/verify and machinery check --impl <path> --store <path> --complete, exact approved flags.
- MAC-wi2u: internal/tdd/adapters/go.go
  spec: go-testing/v1 closed Adapter; native bound helper assertions, no direct t.Fatal compatibility claim.
- MAC-hgz1: examples/go-crm/design/BUILD.md
  source: accepted prospective v2 BUILD/evidence/golden migration; its 41-path scope does not authorize strict test rewrites.
- MAC-l7m0: docs/test-assurance-contract.md
  schema: Exact approved public contract SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8; accepted delivery required, never an uncommitted external proposal.

Observable outcome: The Go CRM consumer receives current strict assurance only after new calibrated tests and all legacy required regressions actually pass.

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
