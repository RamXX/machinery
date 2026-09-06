---
id: MAC-bz1y
title: "Require four-language native assurance conformance lanes"
status: open
priority: 0
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T09:00:51Z
created_by: ramirosalas
updated_at: 2026-09-06T09:22:36Z
content_hash: "sha256:3d0854c95b00e6ae52cc294b287bdf2e48c69e307e8a6dc85a98a6a543992d24"
blocked_by: [MAC-hpqp, MAC-6h0s, MAC-hy71]
blocks: [MAC-wi2u, MAC-avfp, MAC-imtz, MAC-8yai, MAC-pe9v, MAC-al5u, MAC-vx24, MAC-ou97]
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
- scripts/integration-lane/assurance_catalog.go -> scripts/integration-lane/assurance_catalog.go -> compatible required-fragment union and exact native runtime catalog; testdata/integration-lanes/assurance-runtime-pins.json -> four adapter runtime versions plus Git 2.55.0 and both native platforms; CI/preflight -> persistent required execution accounting for the complete fragment union.
- scripts/integration-lane/assurance_catalog_test.go -> owned artifact for the same bounded contract
- testdata/integration-lanes/assurance-runtime-pins.json -> owned artifact for the same bounded contract
- testdata/integration-lanes/assurance.schema.json -> owned artifact for the same bounded contract
- testdata/integration-lanes/assurance.CONTRACT.md -> owned artifact for the same bounded contract
- .github/workflows/ci.yml -> owned artifact for the same bounded contract
- scripts/preflight.sh -> owned artifact for the same bounded contract

### CONSUMES
MAC-hpqp: scripts/integration-lane/main.go
  MAC-6h0s: accepted contributor registry runner, closed exact leaf inventory, required runtime lanes, failure/skip/leak accounting; preserve e55238223961fec922896c361d8af6cafc454a8e seven-file 96-case frozen pilot byte-for-byte.
MAC-6h0s: internal/tdd/types.go
  MAC-6h0s: closed adapter IDs and RuntimeHandle contract; no consumer TDD record is a contributor-lane record.

### ACCEPTANCE CRITERIA
1. Add a separately reviewed compatible registry extension and NEW supplemental fragments, preserving frozen pilot files, v1 history and the complete original 96-case union. Do not rewrite the frozen runtime-pins or original main_test file. If compatibility requires registry v2, explicitly retain v1 interpretation/history and enforce the complete migrated union; no shrinking selection or implicit test loss.
2. Require exact first-release native catalog: go-testing/v1 Go 1.27.1; node-test-typescript/v1 Node 26.8.1 and TypeScript 7.0.2; python-unittest/v1 CPython 3.14.7; elixir-exunit/v1 Elixir/ExUnit/Mix 1.20.4, OTP 29.0.6, ERTS 17.0.6. Git 2.55.0 is a gate runtime, not a fifth adapter. Validate real immutable runtime closures, no arbitrary PATH/version-string shim or runtime fetch during replay.
3. Hosted Linux amd64 and Darwin arm64 each execute required native custody/assertion conformance for all four adapters and scoped Git once their producer fragments land. Final local preflight executes the native local union and verifies exact other-platform hosted/candidate evidence as specified by final closure; no claim local cross-compilation is native proof.
4. Provision required contributor prerequisites before tests that need them; fail on missing runtime/tool/image, mismatched closure, no collected or terminal leaves, skipped/xfail/unexpected extra failure, stale or absent required fragment, or leaked owned resource. Ordinary Go tests may exclude explicitly inventoried infra-only fragments, never silently skip them.
5. Every downstream adapter/replay/Git/CLI/e2e story owns its NEW named fragment; this catalog accepts only closed reviewed fragment schemas and validates union completeness so missing producer integration cannot become a dormant env option. Exact command selection must include frozen sources from RED and report actual runtime/test identities.
6. Targeted real lane fixture tests demonstrate valid merged union and deliberate omission/duplicate/skip/runtime-mismatch/empty-results failure on both platforms. CI and final preflight call this lane persistently; no preflight is run during this story, no installed tool replacement or unrelated container cleanup.

### TEST COMMANDS
Run targeted native package tests for owned files and exact required contributor fragments; record selected test identities and exact commands in RED. Do not run heavy preflight.

### OUT OF SCOPE
Each adapter owns its language runner/assertion transport and specific runtime closure. This catalog wires their required conformance; it does not implement language semantics or edit existing hpqp frozen tests.

### DIFF BUDGET
~8 files, under 1100 changed LOC including tests.

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
- 2026-09-06T09:09:59Z dep_added: blocked_by MAC-hpqp
- 2026-09-06T09:10:00Z dep_added: blocked_by MAC-6h0s
- 2026-09-06T09:10:00Z dep_added: blocks MAC-wi2u
- 2026-09-06T09:10:01Z dep_added: blocks MAC-avfp
- 2026-09-06T09:10:02Z dep_added: blocks MAC-imtz
- 2026-09-06T09:10:03Z dep_added: blocks MAC-8yai
- 2026-09-06T09:10:04Z dep_added: blocks MAC-pe9v
- 2026-09-06T09:10:13Z dep_added: blocks MAC-al5u
- 2026-09-06T09:10:19Z dep_added: blocks MAC-vx24
- 2026-09-06T09:10:20Z dep_added: blocks MAC-ou97
- 2026-09-06T09:22:42Z dep_added: blocked_by MAC-hy71

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-wi2u]], [[MAC-avfp]], [[MAC-imtz]], [[MAC-8yai]], [[MAC-pe9v]], [[MAC-al5u]], [[MAC-vx24]], [[MAC-ou97]]
- Blocked by: [[MAC-hpqp]], [[MAC-6h0s]], [[MAC-hy71]]

## Comments

### 2026-09-06T09:16:30Z ramirosalas
CANONICAL MACHINE-READABLE BOUNDARY MAP 2026-09-06
This repeats the existing ownership/signatures in the parser-supported form; it does not create additional scope or weaken any AC. All prior exact acceptance/testing requirements and holds remain current.

PRODUCES:
- scripts/integration-lane/assurance_catalog.go -> scripts/integration-lane/assurance_catalog.go -> compatible required-fragment union and exact native runtime catalog; testdata/integration-lanes/assurance-runtime-pins.json -> four adapter runtime versions plus Git 2.55.0 and both native platforms; CI/preflight -> persistent required execution accounting for the complete fragment union.
- scripts/integration-lane/assurance_catalog_test.go -> owned bounded artifact; behavior and tests specified in the current story AC
- testdata/integration-lanes/assurance-runtime-pins.json -> owned bounded artifact; behavior and tests specified in the current story AC
- testdata/integration-lanes/assurance.schema.json -> owned bounded artifact; behavior and tests specified in the current story AC
- testdata/integration-lanes/assurance.CONTRACT.md -> owned bounded artifact; behavior and tests specified in the current story AC
- .github/workflows/ci.yml -> owned bounded artifact; behavior and tests specified in the current story AC
- scripts/preflight.sh -> owned bounded artifact; behavior and tests specified in the current story AC

CONSUMES:
- MAC-hpqp: scripts/integration-lane/main.go
  MAC-6h0s: accepted contributor registry runner, closed exact leaf inventory, required runtime lanes, failure/skip/leak accounting; preserve e55238223961fec922896c361d8af6cafc454a8e seven-file 96-case frozen pilot byte-for-byte.
  schema: accepted contributor registry runner, closed exact leaf inventory, required runtime lanes, failure/skip/leak accounting; preserve e55238223961fec922896c361d8af6cafc454a8e seven-file 96-case frozen pilot byte-for-byte.
- MAC-6h0s: internal/tdd/types.go
  MAC-6h0s: closed adapter IDs and RuntimeHandle contract; no consumer TDD record is a contributor-lane record.
  schema: closed adapter IDs and RuntimeHandle contract; no consumer TDD record is a contributor-lane record.
- MAC-l7m0: docs/test-assurance-contract.md
  schema: Exact approved public contract SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8; accepted delivery required, never an uncommitted external proposal.

Observable outcome: A maintainer receives required native test counts on both platforms, and missing or skipped producer cases return a failing lane.

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

### 2026-09-06T09:22:36Z ramirosalas
SCHEMA MARKER INSERTION AUDIT 2026-09-06
Root-authorized supported nd edit inserted ONLY 2 valid indented schema signature line(s) into the newly authored canonical CONSUMES block. Every original byte/contract/status/evidence/history was preserved; no deletion/replacement. This repairs the mechanical label substitution, not the contract values.
Before raw Body SHA256: ed3416e8560e277c04ab47f44c5c8a3f29dccd8fa73b05d275a996f1b879f718
After insertion-only raw Body SHA256 (before this audit comment): 937154158517cf5f5a96b88f009ae32429694ac96247b9ada21901068acc73f3
Exact inserted lines (zero-based original Body line positions shown):
- after Body line 110: "  schema: accepted contributor registry runner, closed exact leaf inventory, required runtime lanes, failure/skip/leak accounting; preserve e55238223961fec922896c361d8af6cafc454a8e seven-file 96-case frozen pilot byte-for-byte."
- after Body line 112: "  schema: closed adapter IDs and RuntimeHandle contract; no consumer TDD record is a contributor-lane record."
Read-back pvg nd show Body exactly equals prior Body plus these insertions. Editor required expected hash, count, exact target and 13-entry total; installed pvg source revision c0957106a81346033d7b1d82fde5f434a9db6bab confirms scanner checks every historical entry.

## nd_contract
status: new

### evidence
- Signature syntax corrected via supported guarded editor; exact before/after evidence above.
- No implementation/native proof; independent Anchor and canonical document holds remain.

### proof
- [ ] All current story ACs remain pending without weakening.
