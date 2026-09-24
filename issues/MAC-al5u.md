---
id: MAC-al5u
title: "Keep every complete example on the strict verification path"
status: open
priority: 2
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T09:07:15Z
created_by: ramirosalas
updated_at: 2026-09-24T21:33:45Z
content_hash: "sha256:963556de8896b25645cc9f27d67e381d5bde4dc6db0f4b07271c1bcf61587ee5"
blocked_by: [MAC-5ft8]
blocks: [MAC-vx24, MAC-ou97]
was_blocked_by: [MAC-bz1y]
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
- scripts/assurance-examples.sh -> scripts/assurance-examples.sh -> closed current example inventory-to-explicit-store strict workflow; existing Makefile/CI/preflight complete-example entrypoints invoke current native replay. scripts/example-inventory.sh classification remains truthful; no implemented example downgrade.
- scripts/assurance-examples-test.sh -> owned artifact for the same bounded contract
- scripts/example-inventory.sh -> owned artifact for the same bounded contract
- Makefile -> owned artifact for the same bounded contract
- .github/workflows/ci.yml -> owned artifact for the same bounded contract
- scripts/preflight.sh -> owned artifact for the same bounded contract
- testdata/integration-lanes/assurance-examples.json -> owned artifact for the same bounded contract

### CONSUMES
MAC-5ft8: examples/go-crm/design/assurance/plan.json
  MAC-6h0s: prospective current strict plan with all legacy regression mappings and exact reviewed helper calibration.
MAC-u4oo: cmd/machinery/tdd.go
  endpoint: machinery tdd store init/export/import/capture/register/red/green/verify; machinery check <design> --impl <path> --store <path> --complete.
MAC-bz1y: scripts/integration-lane/assurance_catalog.go
  MAC-6h0s: complete required fragment union and two native platform accounting.
Existing scripts/example-inventory.sh
  source: rows output consumed by Makefile check and CI/preflight; row[0] design, row[1] impl, row[5] complete.

### ACCEPTANCE CRITERIA
1. Enumerate every shipped example currently claiming implemented complete assurance using the authoritative inventory, not a hand-picked new list. All such rows use the approved strict prospective workflow; design-only rows remain explicitly design-only. Cannot reclassify/deselect an implemented example because migration is difficult.
2. Wire ordinary make check, CI complete-example lane and final preflight to explicit isolated external stores and actual current complete verification. No prior native PASS/old acceptance receipt or downgraded ordinary check substitutes. Use exact reviewed plan/head/object import integrity when replay inputs are transported; transport does not certify replay.
3. Run legacy required regressions and new strict tests as separate accountable inventories; no skipped/empty omitted branch, negative selector filtering, runtime absence fallback or test-name keyword evidence. Preserve all prior example/golden regressions from accepted work.
4. Real entrypoint tests cover complete and design-only rows, fresh environment with no PASS receipt, stale/deleted head/object/control, current-code or helper mutation, missing runtime, invalid plan and removal of a required complete row. Expected diagnostics are checked with valid controls; use actual candidate binary/runtime, no mock CLI.
5. Add new required fragment and persistent native CI/final-preflight wiring without editing hpqp frozen pilot. The full heavy preflight remains final-only; targeted script and example lane commands during this story.
6. Publish exact classification/migration/evidence inventory and ensure remaining unsupported required example guarantees block the release rather than claim a green lane. No private workspace/store paths, Paivot tools or installed binary replacement required.

### TEST COMMANDS
Run targeted native package tests for owned files and exact required contributor fragments; record selected test identities and exact commands in RED. Do not run heavy preflight.

### OUT OF SCOPE
New implementation or strict tests for another newly discovered complete example require separately reviewed bounded ownership before work; no broad source rewrite. Existing Go CRM strict migration is produced upstream, not redone here.

### DIFF BUDGET
~8 files, under 1200 changed LOC including tests.

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
Triage 2026-09-24: NEEDS-REWRITE. Retarget ci.yml wiring to .dagger/main.go (a9dd4b53) and the tiered preflight (0ee03664).

## History
- 2026-09-06T09:10:12Z dep_added: blocked_by MAC-5ft8
- 2026-09-06T09:10:13Z dep_added: blocked_by MAC-bz1y
- 2026-09-06T09:10:13Z dep_added: blocks MAC-1u2v
- 2026-09-06T09:10:30Z dep_added: blocks MAC-vx24
- 2026-09-06T09:10:31Z dep_added: blocks MAC-ou97
- 2026-09-07T10:08:34Z dep_removed: was_blocked_by MAC-bz1y
- 2026-09-24T21:33:54Z dep_removed: no_longer_blocks MAC-1u2v

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-ou97]]
- Blocked by: [[MAC-5ft8]]
- Was blocked by: [[MAC-bz1y]]

## Comments

### 2026-09-06T09:16:34Z ramirosalas
CANONICAL MACHINE-READABLE BOUNDARY MAP 2026-09-06
This repeats the existing ownership/signatures in the parser-supported form; it does not create additional scope or weaken any AC. All prior exact acceptance/testing requirements and holds remain current.

PRODUCES:
- scripts/assurance-examples.sh -> scripts/assurance-examples.sh -> closed current example inventory-to-explicit-store strict workflow; existing Makefile/CI/preflight complete-example entrypoints invoke current native replay. scripts/example-inventory.sh classification remains truthful; no implemented example downgrade.
- scripts/assurance-examples-test.sh -> owned bounded artifact; behavior and tests specified in the current story AC
- scripts/example-inventory.sh -> owned bounded artifact; behavior and tests specified in the current story AC
- Makefile -> owned bounded artifact; behavior and tests specified in the current story AC
- .github/workflows/ci.yml -> owned bounded artifact; behavior and tests specified in the current story AC
- scripts/preflight.sh -> owned bounded artifact; behavior and tests specified in the current story AC
- testdata/integration-lanes/assurance-examples.json -> owned bounded artifact; behavior and tests specified in the current story AC

CONSUMES:
- MAC-5ft8: examples/go-crm/design/assurance/plan.json
  MAC-6h0s: prospective current strict plan with all legacy regression mappings and exact reviewed helper calibration.
  schema: prospective current strict plan with all legacy regression mappings and exact reviewed helper calibration.
- MAC-u4oo: cmd/machinery/tdd.go
  endpoint: machinery tdd store init/export/import/capture/register/red/green/verify; machinery check <design> --impl <path> --store <path> --complete.
- MAC-bz1y: scripts/integration-lane/assurance_catalog.go
  MAC-6h0s: complete required fragment union and two native platform accounting.
  schema: complete required fragment union and two native platform accounting.
Existing scripts/example-inventory.sh
  source: rows output consumed by Makefile check and CI/preflight; row[0] design, row[1] impl, row[5] complete.
- MAC-l7m0: docs/test-assurance-contract.md
  schema: Exact approved public contract SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8; accepted delivery required, never an uncommitted external proposal.

Observable outcome: A maintainer receives a complete-example lane that runs actual strict verification for every implemented example; a missing migration returns failure.

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

### 2026-09-06T09:22:41Z ramirosalas
SCHEMA MARKER INSERTION AUDIT 2026-09-06
Root-authorized supported nd edit inserted ONLY 2 valid indented schema signature line(s) into the newly authored canonical CONSUMES block. Every original byte/contract/status/evidence/history was preserved; no deletion/replacement. This repairs the mechanical label substitution, not the contract values.
Before raw Body SHA256: 0e817bec99f385df73eec9da2f5ebc852a937913afb12070bfea4ecd9f2c92fe
After insertion-only raw Body SHA256 (before this audit comment): 6e27f4e93b882fe4003532181463ccb1ad392a22cd1b62660fa57488d31595aa
Exact inserted lines (zero-based original Body line positions shown):
- after Body line 109: "  schema: prospective current strict plan with all legacy regression mappings and exact reviewed helper calibration."
- after Body line 113: "  schema: complete required fragment union and two native platform accounting."
Read-back pvg nd show Body exactly equals prior Body plus these insertions. Editor required expected hash, count, exact target and 13-entry total; installed pvg source revision c0957106a81346033d7b1d82fde5f434a9db6bab confirms scanner checks every historical entry.

## nd_contract
status: new

### evidence
- Signature syntax corrected via supported guarded editor; exact before/after evidence above.
- No implementation/native proof; independent Anchor and canonical document holds remain.

### proof
- [ ] All current story ACs remain pending without weakening.

### 2026-09-06T09:48:00Z ramirosalas
ROUND-1 RULES 1/2 REPAIR: COMPLETE EXAMPLE INVOCATION OWNERSHIP
Complete CURRENT ownership and acceptance map replaces the incomplete earlier script/inventory list. Existing behavior and all historical evidence remain required.

PRODUCES:
- scripts/assurance-examples.sh -> real complete-example strict workflow.
- scripts/assurance-examples-test.sh -> real entrypoint positive/negative controls.
- scripts/shellcheck-files.txt -> exact sorted unique current shell corpus including BOTH new scripts, preserving every existing entry.
- scripts/example-inventory.sh -> truthful classification, no implemented-example downgrade.
- Makefile -> normal check invocation uses current strict evidence.
- .github/workflows/ci.yml -> sequential strict complete-example native lane.
- scripts/preflight.sh -> final-only complete-example invocation.
- testdata/integration-lanes/assurance-examples.json -> required closed exact-case fragment.

CONSUMES:
- MAC-5ft8: examples/go-crm/design/attestations.yaml
  schema: post-strict-test substantive independent current judgment and exact generated full-root-v1 evidence; authored controls finalized first; not an upstream historical/hash-only refresh.
- MAC-5ft8: examples/go-crm/design/assurance/plan.json
  schema: prospective complete strict plan plus unchanged independently required legacy regression inventory.
- MAC-u4oo: cmd/machinery/tdd.go
  endpoint: exact standalone store/capture/register/red/green/verify and check --impl <path> --store <path> --complete commands.
- MAC-bz1y: scripts/integration-lane/assurance_catalog.go
  schema: complete native fragment union; CI ownership follows MAC-hy71 -> MAC-bz1y -> this story.
- MAC-hpqp: scripts/shellcheck-files.txt
  source: preserved earlier complete shell corpus; amend later when this story actually adds its two scripts.
- Existing scripts/shellcheck-inventory.sh
  source: discovers all scripts/*.sh and compares byte-exact sorted unique corpus with scripts/shellcheck-files.txt; no source change required.

CURRENT ACCEPTANCE CRITERIA
1. Preserve authoritative inventory of one currently implemented-complete Go CRM row and seven explicitly design-only rows. Do not implement or reclassify those seven, downgrade Go CRM, hand-pick a smaller list or alter any legacy regression obligation.
2. Wire actual make check, CI and final preflight to isolated explicit external stores and actual current complete verification. Consume the completed MAC-5ft8 substantive post-change judgment/evidence, not old hgz1/lnu6 approval or imported PASS. Transport integrity alone is not replay.
3. Add both new shell scripts to scripts/shellcheck-files.txt at this story's implementation step. Preserve the entire prior corpus and exact byte-sort/uniqueness. Real inventory positive control must pass with both files; omit either entry, omit an existing entry, add duplicate/unsorted/phantom entry or missing script and the inventory must fail for its intended diagnostic. No future placeholder registration in hpqp.
4. Run legacy required regressions and new strict tests as separately accountable complete sets. No missing/skip/empty branch, selector hiding, runtime absence fallback or test-name keyword evidence.
5. Real candidate-binary entrypoint tests cover fresh environment/no PASS, current completed row, design-only rows, stale pre-refresh versus current independently reviewed evidence, deleted head/object/control, code/helper mutation, missing runtime, invalid plan and removal of the required complete row. No mocked CLI.
6. Persist the exact required fragment in both native CI lanes and final preflight; retain original frozen pilot. Run targeted script/lane validation only during this story, not full preflight.
7. Report exact classification/migration/current-evidence mapping and blocking unsupported requirements truthfully. This invocation owner cannot silently refresh judgments, rewrite fixture/golden bytes or change Gv exclusions; any later subject/golden change needs its own exact reviewed ownership and renewed substantive judgment.
DIFF BUDGET: current9 files, under1300 changed LOC including exact corpus update and controls; investigate overrun.
MANDATORY SKILLS: developer and pm_acceptor.
OUT OF SCOPE: new CRM implementation/strict test edits, automatic attestations refresh, fixture/golden amendments not explicitly reviewed, private tooling/product dependencies, installation or release operations.

## nd_contract
status: new

### evidence
- Independent round1 findings repaired through supported append-only scope/consumer notes; no architecture/source/test/ref/runtime mutation.
- Exact source surfaces verified against accepted epic7e36f3e7ddcf25565d5d4fe60b328df254eee91d; prior Body/status/labels/history preserved.
- Independent Anchor ROUND2 and canonical-document acceptance remain prerequisites, not implementation evidence.

### proof
- [ ] All current story ACs, strengthened ownership/current-judgment requirements and protected frozen proof remain required.
- [ ] Independent review/native execution/final acceptance pending.
