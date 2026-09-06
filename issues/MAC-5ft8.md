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
updated_at: 2026-09-06T09:48:03Z
content_hash: "sha256:86d60c6279716347f7c022a7353108aceedcd8ef90cec0c0e16749f13b66fa16"
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

### 2026-09-06T09:48:03Z ramirosalas
ROUND-1 RULE 2 REPAIR: COMPLETE PROSPECTIVE CRM SUBJECT / CURRENT JUDGMENT OWNERSHIP
This is the complete CURRENT bounded ownership/acceptance map. It preserves all prior migration obligations and expressly assigns the post-change substantive judgment/evidence which new strict tests invalidate. No earlier hgz1/lnu6 approval or hash-only refresh is authority for these later subject changes.

PRODUCES:
- examples/go-crm/design/assurance/plan.json -> prospective complete strict plan.
- examples/go-crm/design/assurance/milestones -> bounded current milestone manifests and exact reviewed controls.
- examples/go-crm/impl/internal/assurancetest -> new strict helper assets with exact pre-RED inventory.
- examples/go-crm/impl/internal/authz/strict_assurance_test.go -> new calibrated strict public-boundary assertions.
- examples/go-crm/impl/internal/domain/strict_assurance_test.go -> new calibrated strict domain assertions.
- examples/go-crm/impl/internal/session/strict_assurance_test.go -> new calibrated strict session assertions.
- examples/go-crm/impl/internal/cli/strict_assurance_test.go -> new calibrated strict CLI assertions.
- examples/go-crm/impl/internal/repo/strict_assurance_test.go -> new calibrated strict real supported persistence assertions, no mocked persistence claim.
- examples/go-crm/design/attestations.yaml -> exact generated current evidence ONLY after substantive independent judgment of the FINAL new complete implementation/test subject and authored controls.
- cmd/machinery/go_crm_assurance_migration_test.go -> new real candidate-CLI stale-before-refresh/current-after-review and inventory/history preservation controls; exact test inventory independently reviewed before RED.
- testdata/integration-lanes/assurance-go-crm.json -> required legacy PLUS new strict/migration control leaf inventory.

CONSUMES:
- MAC-u4oo: cmd/machinery/tdd.go
  endpoint: exact standalone store/scaffold/capture/register/red/green/verify/check --complete workflow.
- MAC-wi2u: internal/tdd/adapters/go.go
  spec: closed go-testing/v1 Adapter and bound assertion helper; legacy direct t.Fatal tests are not relabeled as strict.
- MAC-hgz1: examples/go-crm/design/attestations.yaml
  schema: upstream accepted v2 evidence/history; preserve attribution/history, then independently review the new subject rather than copying prior PASS.
- MAC-lnu6: examples/go-crm/design/attestations.yaml
  source: earlier narrow metadata/BUILD-hash policy repair, already ordered transitively; it does not refresh later implementation entries.
- Existing accepted current evidence generator.
  endpoint: machinery attest --design <design> --claim <claim> --kind current --impl <complete-implementation-root> --attestor <actual-reviewer-attribution> --date <actual-review-date> [--note <verified-note>]; generation PRINTS a complete v2 row and does not write attestations.yaml.
- Existing internal/gates/attest.go.
  source: full-root-v1 compares entire current/recorded entry sets (GV_SCOPE_INVENTORY), then exact types/modes/bytes/hash (GV_STALE_CONTENT); no new exclusions.

CURRENT ACCEPTANCE CRITERIA
1. Inventory EVERY current CRM milestone, oracle/guard/invariant/runtime obligation and existing required regression. Create only prospective reviewed strict tests/controls; retain all original approved/failing RED, accepted history and legacy required files byte-for-byte. Legacy native tests continue independently in the required lane, never selector-hidden or silently converted to strict.
2. Use real public CRM behavior and supported native fixture paths, no fake persistence as complete proof. Missing storage/CGO/service/credential/runtime contract is an explicit blocker requiring review, not permission to downgrade the implemented-complete example or infer support from an old PASS.
3. Finish all authored assurance plan/milestone/helper/test/config/lock/control bytes and retained variant references first, with the new tests' own reviewed RED freeze, same-assertion red_controls and safe/unsafe calibration. Review the exact complete final subject inventory BEFORE substantive judgment/current-evidence generation and assurance execution. Any subsequent governed byte/path/mode/topology/control change invalidates that judgment/evidence and requires renewed substantive review; do not insert tests after refreshing the hash.
4. Demonstrate stale-before-refresh through actual current Gv on the newly added strict subject with the preserved earlier record: added paths must cause GV_SCOPE_INVENTORY, not pass from old hgz1/lnu6 evidence. Also preserve required legacy regression counts/hashes and prove deleting/hiding an old test cannot evade the current complete inventory. These are real candidate CLI controls, not hand-built gate verdicts.
5. Obtain substantive INDEPENDENT review of the final new complete implementation/test inventory, assertion adequacy/negative controls, authored assurance controls, required runtime residual dispositions and all legacy regression evidence. Record actual findings, scope and reviewer attribution/date; never automatically copy a reviewer/date or imply authenticated identity. Reviewer may refuse approval; such refusal blocks current assurance rather than triggering mechanical hash renewal.
6. Only after that independent judgment, generate exact current v2 rows with the actual Machinery attest command against the final complete root, retain generated bytes and merge ONLY reviewed row amendments into the owned attestations.yaml. Preserve prior attribution/history and all unchanged rows/legacy acceptance. No manually invented digest, hash-only refresh, new Gv exclusion, scope narrowing, plan/history relabeling or hidden deletion. Each current claim affected by the complete subject change must be reviewed/refreshed, not one convenient claim.
7. Current-after-review control uses that exact independently judged/generated evidence on unchanged finalized source/control bytes and must pass current Gv and the strict normal replay/complete workflow. Reinsert the old record, alter a strict test/helper/mode/empty directory/control or omit a required judgment and it must block with the intended diagnosis. Hash generation alone is not review or proof tests ran.
8. Execute actual store-init/capture/explicit-register/RED/GREEN/retained replay/complete with native Go on both supported platforms, all matching controls, exact frozen closure and every required check. Preserve actual failure history and distinguish historical ancestor acceptance from current implementation assurance. Any calibration already gathered before final judgment remains evidence of its exact state only; final claimed assurance must use finalized authored controls/judgment and current execution.
9. Required native fragment runs ALL legacy regressions and new strict/migration controls with exact identities/zero missing or skipped leaves, actual profiles/runtime identities and clean owned resources. New manifests/evidence contain no secrets, private tracker prerequisites, absolute store paths or imported fabricated PASS.
10. Deliver exact subject/evidence/legacy-inventory before/after hashes, full independent substantive review, exact generated rows and real stale/current native output. No later fixture or golden change is authorized here: if outputs require golden/fixture edits, obtain separately reviewed exact file/byte ownership and renewed affected judgment before that work. No changes to protected hgz1/lnu6/Y/hpqp/p7/U source, claim or frozen proof; MAC-sh60 original final obligations remain unchanged.

DIFF BUDGET: bounded current scope approximately18 files (including exact pre-reviewed manifest/helper inventory), under2500 changed LOC; additions versus earlier forecast are one evidence file and one focused real migration-control test file. Investigate overrun, never trim controls.
MANDATORY SKILLS: developer, independent pm_acceptor for substantive judgment and acceptance; codebase-memory for actual evidence/gate interfaces.
OUT OF SCOPE: rewriting legacy frozen tests, automatic or hash-only judgment refresh, new Gv exclusions, fixture/golden changes without exact separate review, incomplete example downgrade, remote/install/preflight operations. All original user guarantees remain required.

## nd_contract
status: new

### evidence
- Independent round1 findings repaired through supported append-only scope/consumer notes; no architecture/source/test/ref/runtime mutation.
- Exact source surfaces verified against accepted epic7e36f3e7ddcf25565d5d4fe60b328df254eee91d; prior Body/status/labels/history preserved.
- Independent Anchor ROUND2 and canonical-document acceptance remain prerequisites, not implementation evidence.

### proof
- [ ] All current story ACs, strengthened ownership/current-judgment requirements and protected frozen proof remain required.
- [ ] Independent review/native execution/final acceptance pending.
