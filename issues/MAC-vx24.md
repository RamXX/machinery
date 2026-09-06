---
id: MAC-vx24
title: "Enforce replayable negative test assurance"
status: open
priority: 1
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:35:06Z
created_by: ramirosalas
updated_at: 2026-09-06T12:05:48Z
content_hash: "sha256:3e2194941d6da3e1dd44c6f4e977892b5871a7946aef90a3b0f1d92c87b1ec9f"
blocked_by: [MAC-2n83, MAC-lnu6, MAC-hpqp, MAC-qlw2, MAC-cn7q, MAC-6h0s, MAC-p9z1, MAC-62s6, MAC-bz1y, MAC-wi2u, MAC-avfp, MAC-imtz, MAC-8yai, MAC-pe9v, MAC-sd7g, MAC-sqpt, MAC-wbxq, MAC-rau8, MAC-u4oo, MAC-5ft8, MAC-al5u, MAC-1u2v, MAC-wi5z]
blocks: [MAC-gcrr, MAC-ou97]
was_blocked_by: [MAC-olrx, MAC-p7jd, MAC-l7m0, MAC-p9wm, MAC-sh60]
---

## Description
## USER INTENT
Provide strongest honest standalone Machinery guarantees for LLM-generated software, including test sensitivity to unsafe behavior.

## Context (Embedded)
BLOCKED ARCHITECTURE CONTRACT: do not implement until architecture story is accepted and exact interfaces/fields are copied here. User explicitly requests standalone Machinery-owned deterministic hard-TDD negative testing across its consumer process. Paivot is local development coordination only, never a runtime dependency. Existing schema-free BUILD prose and test filename keywords cannot satisfy this feature.

## Ownership
internal/tdd/manifest.go, internal/tdd/runner.go, internal/tdd/replay.go, internal/tdd/evidence.go, internal/tdd/assurance_test.go, cmd/machinery/tdd.go, cmd/machinery/tdd_test.go, cmd/machinery/main.go, internal/gates/tdd.go, internal/gates/suite.go, cmd/machinery/check.go, internal/hook/hook.go, agents/machinery-build-writer.md, skills/machinery/SKILL.md, skills/machinery/references/build-md-template.md. Not alone in codebase; preserve other edits. Newly listed files are explicitly owned outputs, not pre-existing interfaces.

## Boundary Map
PRODUCES:
- internal/tdd/manifest.go -> standalone enforced test-assurance contract
- internal/tdd/runner.go -> standalone enforced test-assurance contract
- internal/tdd/replay.go -> standalone enforced test-assurance contract
- internal/tdd/evidence.go -> standalone enforced test-assurance contract
- internal/tdd/assurance_test.go -> standalone enforced test-assurance contract
- cmd/machinery/tdd.go -> standalone enforced test-assurance contract
- cmd/machinery/tdd_test.go -> standalone enforced test-assurance contract
- cmd/machinery/main.go -> standalone enforced test-assurance contract
- internal/gates/tdd.go -> standalone enforced test-assurance contract
- internal/gates/suite.go -> standalone enforced test-assurance contract
- cmd/machinery/check.go -> standalone enforced test-assurance contract
- internal/hook/hook.go -> standalone enforced test-assurance contract
- agents/machinery-build-writer.md -> standalone enforced test-assurance contract
- skills/machinery/SKILL.md -> standalone enforced test-assurance contract
- skills/machinery/references/build-md-template.md -> standalone enforced test-assurance contract
CONSUMES:
- MAC-l7m0: docs/test-assurance-contract.md
  schema: Approved closed native-runner replay contract; exact names must be copied after independent architecture acceptance before implementation.
- MAC-sh60: internal/gates/oraclecov.go
  spec: gates.CheckOracleCoverage(design, impl string) *Gate
- MAC-olrx: internal/gates/suite.go
  spec: gates.Select(design, gateList, impl string) (Selection, error)
- MAC-2n83: cmd/machinery/main.go
  source: newRootCmd() *cobra.Command command registration
- MAC-p7jd: docs/attestation-evidence.md
  source: Current implementation subject binding and historical acceptance distinction

### Story Acceptance Criteria
1. Implement the independently accepted standalone test-assurance contract using Machinery-owned native runner execution/replay. No self-reported receipt import or generic command exit code can establish execution or assertion causality.
2. RED proves missing behavior with bound expected assertion failures and a passing harness control; required negative cases additionally distinguish safe control from approved unsafe implementation challenge using the same exact frozen test closure. Negative cases may legitimately already pass ordinary RED safe stubs.
3. GREEN reruns the exact declared tests and frozen dependency/config/fixture closure, with all required tests selected/executed/pass. Any missing, duplicate, empty, skipped, xfail, cached, unexpected failure, compilation/panic/timeout or malformed/truncated evidence blocks rather than looking green.
4. Strong final complete assurance independently replays retained RED/challenges and current GREEN through trusted runner. Changed current implementation/design/test inventory, removed contract/evidence, narrowed test selection, helper/config changes or forged receipt cannot evade enforcement.
5. Wire standalone CLI, default/staged host governance and strong complete handoff consistently. Cheap freshness checks explicitly do not claim replay occurred. Unsupported adapters fail clearly under approved compatibility policy and never receive the strongest label.
6. Remove BUILD template authorization for frozen test formatting via tokens-equal; exact byte/inventory identity is required. Tests prove spaces inside string literals and indentation changes invalidate frozen evidence even when strings.Fields normalization matches.
7. Migrate shipped consumer examples under the accepted compatibility contract and demonstrate actual generated-software flow. Named owned runtime residual obligations connect race/replay/migration/restore/load requirements to evidence rather than prose alone; unsupported guarantees remain explicit.
8. Positive full standalone CLI flow requires no pvg/nd installed or metadata. Negative full-path cases exercise every bypass through actual runner/hooks/complete command. No mocks, stubs or skip-if-missing. Preserve installed Machinery used in NIL.

## Testing Requirements
New focused internal/tdd and cmd/machinery TDD tests; actual native runner subprocesses on temp git snapshots; actual hook and check --complete integration. Separate RED author freezes tests; independent PM replays expected failures and GREEN/challenge results. Full preflight only final gate.
Integration tests: MANDATORY (no mocks) for executable implementation, no skip-if-missing. Architecture review is not execution proof.
No push/sync/GitHub mutation. No installed binary/skills/plugins/agents replacement or dev-link. Isolated candidate only. Full scripts/preflight.sh only final epic gate.

## OUT OF SCOPE
- Mathematical proof of arbitrary test semantic adequacy or malicious host safety beyond explicitly approved trust boundary: expose these as residual limits, never silently claim them.
- Any Paivot runtime dependency: prohibited by user.

## DIFF BUDGET
- ~15-24 files, initial ceiling 3000 changed LOC; MUST split execution/replay and consumer wiring after architecture if this cannot remain reviewable.

## MANDATORY SKILLS
- architect for contract authoring/review; developer and pm_acceptor for implementation; codebase-memory for source discovery.

## nd_contract
status: new

### evidence
- Created 2026-09-05 from user requirements and independent read-only design analysis.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified
- [ ] AC #6: independently verified
- [ ] AC #7: independently verified
- [ ] AC #8: independently verified

Observable outcome: the user can run standalone Machinery and receive an explicit pass or blocking diagnostic based on actual replay.

## Acceptance Criteria


## Design


## Notes
CONSUMES:
- MAC-lnu6: skills/machinery/references/build-md-template.md
  source: Exact-byte frozen-test guidance replacing unsafe tokens-equal authorization; process integration modifies this template sequentially.
ANCHOR ROUND-1 AUTHORITATIVE EXECUTION-LANE REPAIR
General rule: required runtime tests need deterministic provisioning, explicit closed inventory, actual native execution accounting and teardown. Missing infrastructure must fail the REQUIRED lane, not be silently skipped. Ordinary native suites may explicitly exclude registered service-backed tests using a dedicated build tag.
Classification: Native-runner replay/hook/complete E2E requires approved adapter runtimes; exact supported runtime contract remains architecture-blocked.
Record standalone replay suite in required lane even if initial adapter only Go. If approved contract needs additional runtimes, provision in lane before execution. No fallback to receipt-only validation or env-gated omission.
PRODUCES:
- testdata/integration-lanes/tdd.json -> this story's closed suite fragment, with exact source/test IDs, runtime/pin requirements and bounded execution configuration
- cmd/machinery/tdd_test.go -> actual named runtime cases registered in the fragment
CONSUMES:
- MAC-hpqp: testdata/integration-lanes/schema.json
  schema: Closed versioned native-runner suite fragment with exact source/test identities, runtime requirements and bounded command selection.
- MAC-hpqp: scripts/integration-lane/main.go
  endpoint: go run ./scripts/integration-lane --lane required (Makefile test-integration invokes same entrypoint).
Additional acceptance criteria: fragment matches actual test sources both directions; all registered cases actually start/terminate with expected positive/negative outcomes; no cached/skipped/empty/partial/fabricated-summary success; real provisioned positive and missing-runtime/fresh-cache failure diagnostics; no owned container/process leaks. Required local preflight and hosted CI execute the same union. Do not edit shared root inventory; own only this fragment. RED source, fixture, fragment and runner configuration are frozen together after review. Any exact test names introduced in RED must remain registered through GREEN.
No heavy preflight until final gate; no GitHub mutation; no active installation replacement. This note supersedes any earlier command implying service-backed tests execute in unprovisioned ordinary package suites.

## History
- 2026-09-05T19:35:06Z dep_added: blocked_by MAC-l7m0
- 2026-09-05T19:35:07Z dep_added: blocked_by MAC-sh60
- 2026-09-05T19:35:07Z dep_added: blocked_by MAC-olrx
- 2026-09-05T19:35:07Z dep_added: blocked_by MAC-2n83
- 2026-09-05T19:35:07Z dep_added: blocked_by MAC-p7jd
- 2026-09-05T19:35:07Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:14Z dep_added: blocked_by MAC-lnu6
- 2026-09-05T19:36:17Z dep_added: blocks MAC-ou97
- 2026-09-05T19:45:32Z dep_added: blocked_by MAC-hpqp
- 2026-09-05T20:27:03Z dep_removed: was_blocked_by MAC-olrx
- 2026-09-06T07:47:13Z dep_removed: was_blocked_by MAC-p7jd
- 2026-09-06T09:10:14Z dep_added: blocked_by MAC-qlw2
- 2026-09-06T09:10:15Z dep_added: blocked_by MAC-cn7q
- 2026-09-06T09:10:16Z dep_added: blocked_by MAC-6h0s
- 2026-09-06T09:10:17Z dep_added: blocked_by MAC-p9z1
- 2026-09-06T09:10:18Z dep_added: blocked_by MAC-62s6
- 2026-09-06T09:10:19Z dep_added: blocked_by MAC-bz1y
- 2026-09-06T09:10:20Z dep_added: blocked_by MAC-wi2u
- 2026-09-06T09:10:21Z dep_added: blocked_by MAC-avfp
- 2026-09-06T09:10:22Z dep_added: blocked_by MAC-imtz
- 2026-09-06T09:10:23Z dep_added: blocked_by MAC-8yai
- 2026-09-06T09:10:24Z dep_added: blocked_by MAC-pe9v
- 2026-09-06T09:10:25Z dep_added: blocked_by MAC-sd7g
- 2026-09-06T09:10:26Z dep_added: blocked_by MAC-sqpt
- 2026-09-06T09:10:27Z dep_added: blocked_by MAC-wbxq
- 2026-09-06T09:10:27Z dep_added: blocked_by MAC-rau8
- 2026-09-06T09:10:28Z dep_added: blocked_by MAC-u4oo
- 2026-09-06T09:10:29Z dep_added: blocked_by MAC-5ft8
- 2026-09-06T09:10:30Z dep_added: blocked_by MAC-al5u
- 2026-09-06T09:10:31Z dep_added: blocked_by MAC-1u2v
- 2026-09-06T10:30:30Z dep_removed: was_blocked_by MAC-l7m0
- 2026-09-06T12:04:53Z dep_added: blocked_by MAC-p9wm
- 2026-09-06T12:31:39Z dep_removed: was_blocked_by MAC-p9wm
- 2026-09-06T16:34:03Z dep_added: blocked_by MAC-wi5z
- 2026-09-06T19:21:17Z dep_removed: was_blocked_by MAC-sh60

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-gcrr]], [[MAC-ou97]]
- Blocked by: [[MAC-2n83]], [[MAC-lnu6]], [[MAC-hpqp]], [[MAC-qlw2]], [[MAC-cn7q]], [[MAC-6h0s]], [[MAC-p9z1]], [[MAC-62s6]], [[MAC-bz1y]], [[MAC-wi2u]], [[MAC-avfp]], [[MAC-imtz]], [[MAC-8yai]], [[MAC-pe9v]], [[MAC-sd7g]], [[MAC-sqpt]], [[MAC-wbxq]], [[MAC-rau8]], [[MAC-u4oo]], [[MAC-5ft8]], [[MAC-al5u]], [[MAC-1u2v]], [[MAC-wi5z]]
- Was blocked by: [[MAC-olrx]], [[MAC-p7jd]], [[MAC-l7m0]], [[MAC-p9wm]], [[MAC-sh60]]

## Comments

### 2026-09-06T09:15:16Z ramirosalas
CURRENT BOUNDED INTEGRATION UMBRELLA 2026-09-06

This is an honest unfinished integration umbrella, not cancelled/superseded/accepted. The original eight behavioral ACs and every positive/negative test obligation remain required and are mapped below to bounded producers. Earlier broad production ownership (manifest/runner/replay/CLI/gates/hook) and ~3000-LOC one-story estimate are replaced by those producers; this note does not authorize editing their frozen tests. The historical proposed internal/tdd/runner.go and aggregate tdd.json are no longer separate implementation obligations; exact executable coverage is supplied by all named mandatory producer fragments, never discarded. MAC-sh60 original dependency remains unchanged; no successor, transfer or cancellation is authorized.

CURRENT OWNERSHIP / PRODUCES:
- agents/machinery-build-writer.md -> standalone hard-TDD authoring process using the actual supported Machinery commands, no Paivot prerequisite.
- skills/machinery/SKILL.md -> truthful standalone capture/register/RED/GREEN/replay guidance.
- skills/machinery/references/build-md-template.md -> executable obligation/test inventory, negative sensitivity/red_controls, exact frozen closure and current-vs-historical assurance instructions.
- repository_contract_test.go -> focused positive/negative guidance-contract regressions (existing file modified after upstream template repair).
- docs/test-assurance-integration.md -> public implementation/interface/guarantee coverage matrix, without private tracker IDs, workspace paths, chronology or unearned native claims.
All runtime implementation ownership is upstream. Consumer code is not rewritten under this umbrella.

CONSUMES:
- MAC-qlw2: internal/processscope/scope.go
  source: accepted bounded custody producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-cn7q: internal/processcontrol/scope.go
  source: accepted bounded attach producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-6h0s: internal/tdd/manifest.go
  source: accepted bounded schema producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-p9z1: internal/tdd/bundle.go
  source: accepted bounded store producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-62s6: internal/assuranceflow/register.go
  source: accepted bounded register producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-bz1y: scripts/integration-lane/assurance_catalog.go
  source: accepted bounded lane producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-wi2u: internal/tdd/adapters/go.go
  source: accepted bounded go producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-avfp: internal/tdd/adapters/typescript.go
  source: accepted bounded ts producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-imtz: internal/tdd/adapters/python.go
  source: accepted bounded python producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-8yai: internal/tdd/adapters/elixir.go
  source: accepted bounded elixir producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-pe9v: internal/runtimeclosure/git.go
  source: accepted bounded git producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-sd7g: internal/assuranceflow/checks.go
  source: accepted bounded checks producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-sqpt: internal/tdd/execute.go
  source: accepted bounded replay producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-wbxq: internal/gates/tdd.go
  source: accepted bounded gtd producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-rau8: internal/assuranceflow/run.go
  source: accepted bounded flow producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-u4oo: cmd/machinery/tdd.go
  source: accepted bounded cli producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-5ft8: examples/go-crm/design/assurance/plan.json
  source: accepted bounded crm producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-al5u: scripts/assurance-examples.sh
  source: accepted bounded examples producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-1u2v: cmd/machinery/assurance_standalone_e2e_test.go
  source: accepted bounded standalone producer contract and real integration proof; exact signatures copied in its canonical story and approved docs/test-assurance-contract.md.
- MAC-lnu6: skills/machinery/references/build-md-template.md
  source: accepted exact-byte frozen test guidance, preserved sequentially.
- MAC-sh60: internal/gates/oraclecov.go
  spec: CheckOracleCoverage(design,impl string)*Gate; original unfinished coverage integration obligation retained.
- MAC-l7m0: docs/test-assurance-contract.md
  schema: approved 575-line standalone canonical contract and exact API/schema/argv/lifecycle requirements.

COMPLETE CURRENT ACCEPTANCE MAP
1. Original AC1 (standalone execution/replay) is supplied by MAC-6h0s/MAC-p9z1/MAC-62s6, all four adapters, MAC-sqpt/MAC-rau8/MAC-u4oo. Verify every producer is called by the NORMAL complete command; reject uncalled helpers, imported writable evidence, generic nonzero-exit assertion claims or any Paivot dependency.
2. Original AC2 (honest RED and negative sensitivity) is supplied by four native adapters and replay: all baseline failures have same-assertion passing red_controls; every required negative distinguishes frozen safe/unsafe implementation-only variants with exact target/non-target outcomes and passing health controls.
3. Original AC3 (exact GREEN completeness) is supplied by schema/inventory/store/adapters/replay/lane. Verify complete identity/byte/mode/topology inventory, native start/terminal/assertion accounting, no cached/empty/skipped/xfail/malformed/extra/missing failure success and all required checks.
4. Original AC4 (independent current replay) is supplied by store/register/replay/scoped Git/Gtd/finalizer: explicit head CAS, failed history, immutable retained states, source/control/judgment freshness, real final Ga/Gv/Gtd, no-launch/release/publication/output order and aggregate deadlines. Historical acceptance cannot certify current implementation.
5. Original AC5 (normal adoption) is supplied by MAC-u4oo and MAC-wbxq plus THIS shipped guidance. Demonstrate fresh standalone store init/scaffold/capture/register/RED/GREEN/verify/complete and hooks, for all four languages, without a previous PASS receipt. Cheap checks explicitly say replay not performed; strict unsupported cases fail closed with actionable migration diagnostics.
6. Original AC6 (unsafe formatting exemption) remains MAC-lnu6 plus this guidance and store/replay adversarial proof: whitespace inside string literals and indentation, helper/config/lock/mode/topology changes invalidate frozen identity; any amendment starts a newly reviewed revision and replay. Never restore tokens-equal authorization.
7. Original AC7 (consumer migration/runtime residuals) is supplied by MAC-5ft8/MAC-al5u/MAC-1u2v: every complete example remains required, all legacy regressions retained separately, prospective strict tests calibrated and all qualified runtime categories accounted for. No completed example downgrade or old PASS substitution.
8. Original AC8 (full standalone positive/negative paths) is supplied by actual native Linux amd64 AND Darwin arm64 conformance, MAC-1u2v and retained original saga/coverage/OCI/OpenCode/recovery gates at final MAC-ou97. Audit actual process-producing call graph and cumulative owner/milestone deadlines. No mocks, skips, private tracker/runtime dependency or installed binary replacement.
9. This story's own guidance/test changes follow hard-TDD: expected old unsafe/missing wording fails specific assertions before edits; freeze those tests. Integration matrix records actual accepted producer evidence and residual limits accurately (unauthenticated-host/native-cooperative, not malicious-host isolation, semantic completeness or chronology). Developer delivers, independent PM verifies and accepts.

MANDATORY SKILLS: developer and pm_acceptor; no architecture redesign authority.
DIFF BUDGET: ~5 files, under 1200 changed LOC; investigate overrun.
OUT OF SCOPE: production fixes go to their explicit bounded owner or a newly reviewed blocking P0 story; remote/install/final preflight operations remain root final gate only.

## nd_contract
status: new

### evidence
- Nineteen bounded required producer/consumer stories created; none is claimed implemented or accepted.
- All original semantic ACs mapped without dropping original MAC-sh60 or example/template chain.
- Independent Anchor backlog review and MAC-l7m0 acceptance remain dispatch prerequisites.

### proof
- [ ] AC1-AC8: original behavioral obligations and actual normal integration verified after all producers accepted.
- [ ] AC9: standalone guidance, frozen negative doc tests and truthful public matrix delivered/independently accepted.


### 2026-09-06T09:16:35Z ramirosalas
CANONICAL SUPPLEMENTAL OUTPUT MAP 2026-09-06
PRODUCES:
- agents/machinery-build-writer.md -> exact bounded current ownership in preceding approved scope revision; no additional implementation authority
- skills/machinery/SKILL.md -> exact bounded current ownership in preceding approved scope revision; no additional implementation authority
- skills/machinery/references/build-md-template.md -> exact bounded current ownership in preceding approved scope revision; no additional implementation authority
- repository_contract_test.go -> exact bounded current ownership in preceding approved scope revision; no additional implementation authority
- docs/test-assurance-integration.md -> exact bounded current ownership in preceding approved scope revision; no additional implementation authority

## nd_contract
status: new

### evidence
- Parser-supported output declaration only; earlier current AC/proof map and healthy claim preserved.

### proof
- [ ] Every existing and supplemental current AC remains required; no execution proof claimed.

### 2026-09-06T09:48:00Z ramirosalas
ROUND-1 RULE 1 REPAIR: COMPLETE CURRENT INTEGRATION OUTPUT MAP
The current root-level repository_contract_test.go references in earlier scope/provenance are a historical path typo, NOT an owned output. Do not create that root file or treat it as a replacement for existing contract tests. This complete current output map supersedes those historical declarations without deleting evidence.

PRODUCES:
- agents/machinery-build-writer.md -> truthful standalone process adoption.
- skills/machinery/SKILL.md -> supported capture/register/RED/GREEN/replay guidance.
- skills/machinery/references/build-md-template.md -> exact frozen closure and negative/red_control process.
- cmd/machinery/repository_contract_test.go -> THE existing repository contract regression surface, retaining existing assertions except separately approved RED changes.
- docs/test-assurance-integration.md -> truthful public completed-interface/guarantee coverage matrix.

CONSUMES:
- MAC-lnu6: skills/machinery/references/build-md-template.md
  source: prior exact-byte policy repair, modified only sequentially without restoring exemptions.
- MAC-5ft8: examples/go-crm/design/attestations.yaml
  schema: substantive independently reviewed post-strict-test full current subject, exact generated evidence and preserved history; upstream hgz1/lnu6 current evidence cannot stand for later added tests.
- MAC-al5u: scripts/shellcheck-files.txt
  source: both new scripts actually registered in preserved sorted exact corpus.
- MAC-l7m0: docs/test-assurance-contract.md
  schema: same approved canonical document; schema/store/budget is section4, obligation activation section3, adapter semantics section6, APIs section7, custody section8.

CURRENT ACCEPTANCE MAP
All nine original current integration-umbrella ACs remain required in full. For clarity their required outcomes are: (1) every bounded producer called by normal complete; (2) exact baseline assertion red_controls and safe/unsafe sensitivity; (3) complete native GREEN/inventory/checks; (4) registered retained replay and final no-launch/release/publication/output/aggregate-budget lifecycle; (5) normal four-language first-use commands/hooks with honest cheap freshness; (6) no frozen formatting exemption; (7) all legacy regressions and prospective complete-example migration; (8) actual two-platform full standalone/bypass/custody proof; (9) shipped guidance with frozen positive/negative contract tests and truthful residual limits.
This story owns the five paths above ONLY. New/changed contract tests go into the actual cmd/machinery/repository_contract_test.go, preserve every existing regression, and require exact independent RED amendment review before any existing frozen test edit. Execute that existing package's real contract tests; a new root-package replacement is forbidden.
Complete-example integration specifically requires the MAC-5ft8 post-change substantive judgment refresh and stale-before/current-after controls. No hash-only automatic reviewer/date renewal, Gv exemption, fixture/golden mutation or implementation downgrade.
The current dependency DAG and all nineteen upstream obligations, original MAC-sh60 blockers, hpqp96pilot and protected claims remain unchanged. Budget remains five files/<1200 LOC. No new architecture or implementation proof is asserted.

## nd_contract
status: new

### evidence
- Independent round1 findings repaired through supported append-only scope/consumer notes; no architecture/source/test/ref/runtime mutation.
- Exact source surfaces verified against accepted epic7e36f3e7ddcf25565d5d4fe60b328df254eee91d; prior Body/status/labels/history preserved.
- Independent Anchor ROUND2 and canonical-document acceptance remain prerequisites, not implementation evidence.

### proof
- [ ] All current story ACs, strengthened ownership/current-judgment requirements and protected frozen proof remain required.
- [ ] Independent review/native execution/final acceptance pending.

### 2026-09-06T12:05:48Z ramirosalas
NATIVE-CUSTODY PUBLIC CONTRACT CONSUMER

MAC-vx24 now depends on MAC-p9wm because its owned `docs/test-assurance-integration.md` is the final public completed-interface/guarantee coverage matrix and must consume the accepted standalone native-custody contract rather than a private proposal or an unaccepted implementation claim.

CONSUMES:
- MAC-p9wm: docs/native-custody-contract.md
  source: accepted standalone public native-custody refinements; exact APIs/schemas/profile/lifetime/cleanup/proof limits are authoritative for the integration matrix.
- MAC-p9wm: docs/test-assurance-contract.md
  source: accepted minimum companion link defining refinement precedence while preserving the original contract remainder.

This adds no implementation ownership and does not supersede MAC-qlw2 or any existing producer. MAC-vx24 remains open/new and retains every prior blocker and acceptance obligation.

## nd_contract
status: new

### evidence
- MAC-p9wm added as a direct docs dependency for the already owned final integration document.
- No source/test/status/label/claim change.

### proof
- [ ] Accepted native-custody publication is accurately consumed in final integration guidance.
- [ ] All pre-existing MAC-vx24 ACs remain pending.

