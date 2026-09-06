---
id: MAC-ui8a
title: "Deterministic assurance hardening"
status: open
priority: 0
type: epic
created_at: 2026-09-05T19:28:10Z
created_by: ramirosalas
updated_at: 2026-09-06T09:36:56Z
content_hash: "sha256:3d3cb362bf1c5281045f4a8ed83ca9df7d96bb3c8d474372052230fbf5dac9aa"
---

## Description
## USER INTENT
Machinery is mission-critical open-source software that creates software. Strengthen its own guarantees and consumer correctness through deterministic enforcement, genuine hard-TDD RED failures including negative cases, full-path verification, and trustworthy provenance.

## Context
Assessment dated 2026-09-05 at 497419ab4512fcff765cd5feb27aed4c67b5608d reproduced: unsafe saga routes accepted by proofs; comments/constants/disabled tests credited as oracle coverage; stale implementation attestations; Docker checker containers surviving timeout; release publication outrunning CI; parent-policy and consumer READS obligation gaps; shallow nightly failures. Additional OpenCode runner bounds, installer receipt consistency, guard-clause ownership, safe recovery, baseline advice, documentation are in scope. NEXT modeling features 6/10/11/14 remain separately inventoried pending user scope clarification; do not infer security semantics for speculative hierarchy #13.

## Execution constraints
- All code stories hard-tdd: separate RED test author and GREEN implementer; immutable RED tests, independent reviewer replays assertion failures, controls pass.
- Integration tests MUST exercise real commands/services; no mocks, no stubs, no skip-if-missing. Missing prerequisites are explicit blockers, not success.
- Targeted tests during stories. Run scripts/preflight.sh ONLY after all implementation work, during final epic completion. Do not substitute target tests for final preflight.
- No GitHub push, sync, remote settings changes, or release publication during work. pvg loop setup and pvg story merge currently push: dispatcher must use explicitly local-only alternatives under user override.
- Story branches from epic; accepted stories merge locally into epic; final successful gate then local merge main and build versioned binary. No branch checkout by backlog author.
- Keep existing files and other agents' edits. All nd changes via pvg shared tracker; append authoritative nd_contract blocks.
- Final user-facing report must distinguish established proofs, assumptions/residuals, executed tests, current implementation review, and checks not run.

## Boundary Map
PRODUCES:
- Local hardening delivery and its verified release candidate.
CONSUMES:
- Existing Machinery CLI and consumer design contracts.
  source: Current source at the assessment SHA; exact signatures embedded in child stories.

### Story Acceptance Criteria
1. Every assessment finding has a bounded child story with positive/negative proof and no artificial exclusions.
2. No unsafe route, missing owner, stale review, missing execution, or malformed runner evidence silently yields current assurance.
3. Capstone depends on every implementation story, proves full user flows using real infrastructure, and final preflight passes before main merge or local binary handoff.
4. NEXT dispositions are accurate and no speculative feature is implemented without a settled contract.

## MANDATORY SKILLS
- sr_pm for backlog; developer for implementation; pm_acceptor for acceptance; orchestrator for local dispatch.

## nd_contract
status: new

### evidence
- Created 2026-09-05 from read-only assessment, current source signatures, NEXT.md, and prior real-integration/hard-TDD vault lessons.

### proof
- [ ] Complete hardening with independently reviewed evidence.

## Acceptance Criteria


## Design


## Notes
AUTHORITATIVE USER CONSTRAINTS 2026-09-05: Machinery product must be standalone, never require Paivot/pvg/nd, workflow labels or commit conventions. Local development coordination only may use Paivot. Another agent uses installed Machinery in NIL: do not replace installed binary/plugins/skills/agents; no dev-link or live install/update. Build isolated candidate only. No GitHub push/mutation during work. Full scripts/preflight.sh only final epic gate. RED author may update preexisting tests that encode superseded unsafe behavior with explicit review and genuine assertion-failure proof; after RED approval freeze exact tests/fixtures/config bytes.
BACKLOG READY FOR ANCHOR — 2026-09-05
One epic, 16 bounded children. Scoped pvg lint --backlog --epic MAC-ui8a --json => [] (zero errors and review findings). pvg rtm check --epic MAC-ui8a => passed, 0 tagged requirements, 17 records checked. pvg nd dep cycles => none; stale => none. No close-eligible epic while work remains (expected). Source worktree clean; no branches checked out or production edits by Sr PM. All code fixes hard-tdd; pure integrated capstone uses safe/unsafe mutation sensitivity rather than manufactured RED. Architecture MAC-l7m0 explicitly blocked pending user/architect; implementation MAC-vx24 cannot dispatch before it.
NEXT mapping: #1 MAC-2u36; #2 retain failclosed discovery policy documented MAC-gcrr; #3 MAC-olrx; #4 MAC-2n83; #5 current root-registry workaround MAC-gcrr (feature extension unapproved); #6 platform extension pending user, emulation workaround documented; #7/#8 MAC-gcrr plus release MAC-hy71; #9 MAC-a89e; #10/#11/#14 concrete model features inventoried pending scope approval; #12 MAC-p8ce; #13 hierarchy deferred until actual access-boundary need.
Workflow observation (local tracker only, no external tool fix): pvg issues create with canonical nested headings produces duplicate managed sections; nd update --description inserts ahead of authored Markdown headings instead of replacing entire authored body. Repaired with parent-authorized pvg nd edit and exact apply_patch-backed editor, preserving notes/history/contracts. nd EDITOR requires a single executable path, not command plus arguments.
ANCHOR ROUND-1 RULE 1 OWNERSHIP CLARIFICATION: The repair must explicitly own scripts/preflight.sh and affected .github/workflows/ci.yml/formal.yml/nightly.yml surfaces; any shared test-lane runner/closed inventory, pinned test-image or engine fixture/manifests, Makefile entrypoint and shell inventory entries if introduced; and the capstone's invocation/selected-test assertions. Add producer dependencies from affected runtime stories to the final wiring validation/capstone while keeping early reusable lane scaffolding executable without cycles. A selected named integration lane is different from t.Skip when Docker is missing: ordinary native suites may deliberately exclude an inventoried service lane, but the required local/CI lane must fail on missing prerequisites or zero selected/executed cases and must be included in final completion.
ANCHOR ROUND-1 GENERALIZED REPAIR — READY FOR ROUND 2
Specific gap: runtime-backed tests lacked required persistent provisioning/execution lanes.
General rule: every mandatory external-runtime suite must be explicitly inventoried, provisioned before invocation, proven actually selected/executed, and torn down. Native portability suites may exclude only the explicitly registered integration lane; the required lane fails on missing infrastructure or any empty/skipped/incomplete execution.
New MAC-hpqp owns contributor lane CLI, closed per-suite schema/discovery, immutable runtime pin manifest, initial real pilot, scripts/preflight.sh, CI/formal/nightly wiring, Makefile and shell inventory coverage. It is independently executable before bug stories; no placeholder suites. Release MAC-hy71 follows it for shared workflow ownership and adds lane job to exact-SHA required status policy.
Full sweep of all original 16 children:
- MAC-hlae: TLC/Java -> own saga fragment + lane dependency; pure generator tests stay native.
- MAC-yhg5: Docker lifecycle -> own oci fragment + lane dependency.
- MAC-2n83: Docker cross-identity recovery -> recovery fragment/new tagged integration file + lane dependency; native recovery stays native.
- MAC-hwdb: Node real processes -> opencode fragment + lane dependency.
- MAC-vx24: approved native adapter runtimes -> tdd fragment + lane dependency; architecture blocker retained.
- MAC-gcrr: real checker documentation examples -> consumer-docs fragment/new tagged integration file + lane dependency.
- MAC-ou97: combined full-system runtimes -> capstone fragment + lane dependency and final union/actual-case assertions; pure capstone sensitivity policy retained.
- MAC-hy71: pinned actionlint plus native git policy; follows lane-owned shared workflows, required release status includes runtime lane.
- MAC-sh60/MAC-olrx/MAC-p8ce/MAC-p7jd/MAC-2u36/MAC-a89e/MAC-lnu6: service-free native Go/filesystem/local-process cases; each explicitly forbids incidental runtime prerequisites and requires inventory ownership if runtime scope expands.
- MAC-l7m0: read-only architecture, not execution evidence; exact adapter runtimes flow to MAC-vx24.
Each runtime story owns a separate fragment, so no concurrent shared root-manifest edits. Frozen RED tests/fixtures/runner configuration include those fragments. No env-gated dormant tests or skip-if-missing added.
Self-review: all 17 children clean within their approved or explicitly blocked scope; lane is deliberately infrastructure-focused but delivers observable real provision/run/account/cleanup flow, not static wiring alone.
Validation after repair: pvg lint --backlog --epic MAC-ui8a --json => []; pvg rtm check --epic MAC-ui8a => passed, 0 tagged requirements, 18 records; pvg nd dep cycles => none. Full preflight not run.
LOCAL PROGRESS CHECKPOINT 2026-09-05: MAC-olrx accepted and merged locally to epic/MAC-ui8a=f24b2df3cb1e1521f97b406a7516f72bb7bc7890; merged-epic targeted ownership replay PASS10.274s. Main remains497419ab4512fcff765cd5feb27aed4c67b5608d. MAC-hpqp REDv1=612f65f3b4502a3267828507faaf0e395c8dd558 was independently rejected with four test-contract gaps; authorized RED rework active in retained worktree via /root/red_hpqp. MAC-2u36 RED=99956740ae5559262790a9473b5597c1775928f2 final targeted replay active via /root/red_2u36. MAC-p8ce RED approved0d52f43b393d961160aeea0db43b13a3fa5c284a remains queued, no GREEN yet. No full preflight completed/run intentionally; no GH mutations or installation updates. Installed binary SHA2565205883aaa4276d7eb6edb25b6ad43ac39a04bcb9a8b5ee55498127b04950849 confirmed unchanged again. Local remote.origin.pushurl=/dev/null remains intentional guard (original unset). Do not use pushing sync/merge/setup commands; manual accepted-gated local merges only. Do not run destructive recovery while current agents are healthy. Architecture/native-adapter choices still await previously relayed user answers.
VERIFICATION HANDOFF: Full scripts/preflight.sh remains deferred to final epic gate. GREEN deliveries should nevertheless run proportionate targeted compiler/tests, affected-package golangci-lint/actionlint where applicable, and exact frozen-input verification now; pvg proof-shape/quality scans do not replace Go lint or behavioral execution. No claim of complete preflight from these checks. Runtime REDv2 e55238223961fec922896c361d8af6cafc454a8e independently approved (96 new leaves39pass57intendedfail0skip,20 existing controls); installer ordinary missing-file diagnostic uncovered real premature full-receipt inventory and is being grouped into MAC-2u36 before further RED edits.

## ANCHOR REVIEW (backlog_review, round 2 of 3)
REVIEW_RESULT: APPROVED

### Evidence
- Independently reran pvg lint --backlog --epic MAC-ui8a --json FIRST: [].
- Read new MAC-hpqp in full and compared authoritative repairs across every original child; reviewed epic repair summary and dependency mapping.
- Independently ran pvg nd dep cycles: No dependency cycles found.
- Scope is now 17 children. RTM's 0 tagged requirements is not substantive requirements proof; the round-1 manual assessment/NEXT mapping remains the basis.

### Previous rule resolution
The required real-infrastructure execution-lane gap is fixed at backlog-contract level and generalized:
- MAC-hpqp owns a nonempty real provision/execute/account/cleanup pilot, closed fragment discovery and pin contracts, shared runner, CI/formal/nightly, preflight ordering, Makefile and shell inventory.
- Runtime stories own distinct suite fragments and executable test identities, depend on that lane, and cannot leave service-backed cases unconditionally in ordinary native suites.
- Service-free stories and architecture-only work are explicitly classified; additional runtime scope requires inventory/dependency repair.
- Required local/hosted lanes reject absent infrastructure, zero/missing/skipped/cached/truncated execution and cleanup residue. Explicit native-lane exclusion is not a silent skip.
- Release MAC-hy71 follows shared workflow ownership and requires the exact integration job status.
- Capstone depends on the new lane and checks the delivered union plus actual selected/executed case evidence.

### Judgment
No remaining critical or major backlog findings. The standalone product constraint, no live installation replacement, no remote writes during work, exact frozen tests, independent behavioral RED review, negative mutation sensitivity, safe recovery identity preservation, and final-only heavy preflight remain intact.

Approval concerns backlog readiness, not delivery or permission to bypass existing blockers. MAC-l7m0 still requires user choices and independent architecture acceptance; MAC-vx24 must receive the exact accepted contract before implementation. Pending NEXT feature scope remains visible and cannot be called completed merely by finishing bug stories. No source changes, installed-runtime changes, GitHub mutations or heavy preflight were performed in this review.

## nd_contract
status: new

### evidence
- Anchor round-2 review approved after independent clean mechanical lint and complete generalized-repair inspection.
- 17 child contracts reviewed; independent cycle check clean.

### proof
- [x] All original assessment/NEXT bug findings have bounded owning work.
- [x] Required integration lanes, provisioning, inventory and final preflight wiring assigned.
- [x] Standalone/no-live-install/no-remote-write constraints preserved.
- [x] Backlog passes independent Anchor review.
- [ ] Architecture user choices and exact implementation contract settled.
- [ ] Implementation, independent acceptance, capstone and final heavy preflight completed.


## ANCHOR REVIEW (backlog_review, round 1 of 3)
REVIEW_RESULT: REJECTED

### Evidence
- Independently ran pvg lint --backlog --epic MAC-ui8a --json first: [].
- Read epic and all 16 child bodies including authoritative append-only corrections, full assessment report and NEXT.md.
- Manually mapped assessment F1-F7, OpenCode risk, NEXT bugs and newly identified tokens-equal risk. Scoped RTM has zero tagged requirements and is not substantive requirements assurance.
- Read relevant hard-TDD adversarial-review and real-integration vault patterns.
- Verified current CI/preflight source against graph generation 2026-09-04T15:44:34Z; coverage metadata_match. No preflight or test suite run by this review.

### Gaps
1. CRITICAL: required real-infrastructure tests lack an executable persistent CI/preflight lane.
ISSUE: MAC-yhg5 requires real Docker tests under cmd/machinery with no skip; MAC-2n83 requires real Docker publication recovery; MAC-hlae requires pinned TLC CLI verification; MAC-hwdb adds real Node tests; MAC-ou97 combines those boundaries. Current .github/workflows/ci.yml native-tests runs go test -count=1 ./... on macos-latest with only Go provisioning, not a Docker daemon. The Linux test job also invokes the full package suite without provisioning the story's immutable synthetic OCI image. scripts/preflight.sh runs go test -race -count=1 ./... at stage 9 but provisions the external-checker image only at stage 15. New mandatory unconditional package tests therefore either break these environments or invite the skip-based false green prohibited by the user. No story owns scripts/preflight.sh or explicitly integrates new suites into a required, correctly provisioned test lane; MAC-hy71's workflow criteria cover release status and history, not this lifecycle.
RULE: Every new infrastructure-backed regression suite must have a named required execution lane, deterministic prerequisite provisioning before invocation, explicit selection/inventory proving it actually ran, and teardown verification. Ordinary native portability jobs must not implicitly depend on unavailable services. Local final preflight and required hosted CI must exercise the same safety coverage; selection is permissible, silently missing/skipping a required suite is not.
SCOPE: Sweep all 16 children for external-runtime dependencies, especially MAC-hlae, MAC-yhg5, MAC-2n83, MAC-hwdb, MAC-vx24 and MAC-ou97. Assign CI/preflight/inventory ownership to a bounded wiring story or explicit amended MAC-hy71 contract with appropriate producer dependencies. Include fresh-cache/absent-runtime negative checks and a provisioned positive, selected-test nonzero inventory, supported-host lane behavior, and no-orphan verification. Do not run the heavy preflight during the repair; test the wiring with targeted contract/execution checks and leave actual full preflight to the final gate.

### Positive judgment retained
- The independent bug stories materially cover the reproduced unsafe paths with intended-diagnostic assertions and positive controls rather than accepting arbitrary failure.
- Whole-machine saga reconciliation and unsafe mutations are required; recovery explicitly preserves rooted identity protections rather than globally replacing inodes with hashes.
- Standalone Machinery, no installed replacement while NIL uses it, no push/remote mutation during work, and final-only preflight constraints are explicit.
- MAC-l7m0 is genuinely blocked for user/architecture decisions; MAC-vx24 cannot dispatch before its contract is approved and copied. This is not a demand for greenfield D&F or grounds to redesign the independent bug fixes.
- The capstone's append-only correction correctly supersedes manufactured-RED wording and requires mutation sensitivity after all siblings.
- NEXT modeling extensions remain explicitly pending user scope; this review does not authorize silently dropping them or claiming the entire user goal complete while those decisions remain unresolved.

## nd_contract
status: new

### evidence
- Anchor round-1 manual review completed; mechanical backlog lint [].
- One critical general rule reported with complete identified instances; source/backlog story bodies not edited by reviewer, only this epic note appended.

### proof
- [x] All 16 children reviewed against assessment and NEXT.
- [x] Standalone and no-live-install/no-remote-write constraints checked.
- [ ] Required integration test lanes, provisioning, inventories and final preflight wiring assigned.
- [ ] Backlog passes independent Anchor review.


## nd_contract
status: new

### evidence
- Backlog authored from current source signatures, assessment probes, NEXT, prior hard-TDD and real-integration lessons.
- Scoped lint [] and RTM passed; zero dependency cycles; git status --short empty.

### proof
- [x] All confirmed findings tracked with positive/negative and real-path verification criteria.
- [x] Standalone Machinery/no installed replacement/no remote mutation constraints captured.
- [ ] Architecture user choices settled and implementation story exact interfaces repaired.
- [ ] Implementation, independent PM reviews, E2E and final heavy preflight completed.

## History


## Links
## Comments

### 2026-09-06T09:26:01Z ramirosalas
FINAL SR PM ASSURANCE BACKLOG HANDOFF 2026-09-06

STATE: Ready for INDEPENDENT ANCHOR BACKLOG REVIEW, NOT developer dispatch. MAC-l7m0 remains blocked; every new implementation slice has a transitive prerequisite on its accepted canonical document. None of this report is implementation, native proof, final-preflight or release evidence.

Immutable child/backlog content checkpoint before this report: nd/backlog e72dd2d363a20e8e7fd4a0d65f099a0fc8caee1f.
Root source unchanged and clean: main 497419ab4512fcff765cd5feb27aed4c67b5608d; epic 7e36f3e7ddcf25565d5d4fe60b328df254eee91d.
Approved private source SHA e467a3b6c6f657e7a4cb28e85c688be0df24c56a7e3c76fdcfa224f7ff5d2624; independent CHALLENGE-3 SHA0205b50a8eb2e135230173bbf4a7b7eccdde5934fabab5aa4b2dde95d474ac66 approved0blocking. Public one-doc projection575lines SHA22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8, exact reviewed extraction in MAC-l7m0. Private tracker/provenance/user coordination stays in nd, not the public contract.

NEW BOUNDED PRODUCERS/CONSUMERS (19, all unfinished)
| ID | Observable delivery | Actual direct prerequisites | Primary ownership |
| --- | --- | --- | --- |
| MAC-qlw2 | Retain native child ownership until cleanup completes | MAC-l7m0 | internal/processscope/scope.go |
| MAC-cn7q | Keep formal subprocesses inside native custody | MAC-qlw2 | internal/processcontrol/scope.go |
| MAC-6h0s | Reject incomplete executable assurance declarations | MAC-qlw2 | internal/tdd/manifest.go |
| MAC-p9z1 | Retain exact replay inputs outside governed sources | MAC-6h0s | internal/tdd/bundle.go |
| MAC-62s6 | Register reviewed assurance revisions explicitly | MAC-p9z1 | internal/assuranceflow/register.go |
| MAC-bz1y | Require four-language native assurance conformance lanes | MAC-hpqp, MAC-6h0s, MAC-hy71 | scripts/integration-lane/assurance_catalog.go |
| MAC-wi2u | Prove Go assertions through the native test runner | MAC-bz1y, MAC-6h0s | internal/tdd/adapters/go.go |
| MAC-avfp | Prove TypeScript assertions through native node tests | MAC-bz1y, MAC-6h0s | internal/tdd/adapters/typescript.go |
| MAC-imtz | Prove Python assertions through native unittest | MAC-bz1y, MAC-6h0s | internal/tdd/adapters/python.go |
| MAC-8yai | Prove Elixir assertions through native ExUnit | MAC-bz1y, MAC-6h0s | internal/tdd/adapters/elixir.go |
| MAC-pe9v | Keep final acceptance Git queries in the verification scope | MAC-cn7q, MAC-bz1y | internal/runtimeclosure/git.go |
| MAC-sd7g | Run required checks against the exact replay state | MAC-wi2u, MAC-avfp, MAC-imtz, MAC-8yai, MAC-pe9v | internal/assuranceflow/checks.go |
| MAC-sqpt | Reject assurance that fails retained-state replay | MAC-62s6, MAC-sd7g | internal/tdd/execute.go |
| MAC-wbxq | Expose current assurance freshness without claiming replay | MAC-p9z1, MAC-pe9v, MAC-lnu6 | internal/gates/tdd.go |
| MAC-rau8 | Seal verification only after the complete native lifecycle | MAC-sqpt, MAC-wbxq | internal/assuranceflow/run.go |
| MAC-u4oo | Enforce standalone assurance through normal Machinery commands | MAC-rau8, MAC-62s6, MAC-2n83 | cmd/machinery/tdd.go |
| MAC-5ft8 | Migrate Go CRM to prospective strict assurance | MAC-u4oo | examples/go-crm/design/assurance/plan.json |
| MAC-al5u | Keep every complete example on the strict verification path | MAC-5ft8, MAC-bz1y | scripts/assurance-examples.sh |
| MAC-1u2v | Exercise fresh standalone assurance in all four languages | MAC-al5u, MAC-u4oo | cmd/machinery/assurance_standalone_e2e_test.go |

EXISTING STORY REPAIRS
- MAC-l7m0: append-only complete current one-document A1-A6, exact public projection and developer-delivers/independent-PM-accepts role split; no public review/status index outputs.
- MAC-hpqp: healthy in_progress/red-approved claim retained, original frozen96union and failed history unchanged; supplemental lane custody only after MAC-cn7q. New broker belongs MAC-qlw2, explicit formal attachment MAC-cn7q, four-language catalog MAC-bz1y, scoped final Git/Ga MAC-pe9v. No frozen pilot rewrite or native-proof assertion.
- MAC-vx24: unfinished bounded integration/guidance umbrella; original eight behavioral ACs mapped across all19 producers, original coverage/template/example obligations retained. Five owned guidance/contract-test/matrix outputs instead of one oversized runtime implementation.
- MAC-ou97: original all-hardening final capstone retained, extended with first-use four-language/two-platform actual proof, full bypass/custody/deadline matrix and exact final closure. Still tests-only/no hard-tdd label, blocked on every new sibling.
- MAC-lhu5, MAC-hgz1, MAC-lnu6 and MAC-yig6 claims/scope/frozen proof untouched. Only nd-native new Blocks/history metadata on lnu6 arises from necessary MAC-wbxq ordering. hgz1 is ordered transitively, no redundant edge. MAC-sh60 has NO mutation/successor/cancellation/transfer/new edge; original vx24/ou97 obligations remain.

EXACT NEW DEPENDENCY SET (77 added edges; none removed)
- MAC-qlw2 depends on: MAC-l7m0
- MAC-cn7q depends on: MAC-qlw2
- MAC-6h0s depends on: MAC-qlw2
- MAC-p9z1 depends on: MAC-6h0s
- MAC-62s6 depends on: MAC-p9z1
- MAC-bz1y depends on: MAC-hpqp, MAC-6h0s, MAC-hy71
- MAC-wi2u depends on: MAC-bz1y, MAC-6h0s
- MAC-avfp depends on: MAC-bz1y, MAC-6h0s
- MAC-imtz depends on: MAC-bz1y, MAC-6h0s
- MAC-8yai depends on: MAC-bz1y, MAC-6h0s
- MAC-pe9v depends on: MAC-cn7q, MAC-bz1y
- MAC-sd7g depends on: MAC-wi2u, MAC-avfp, MAC-imtz, MAC-8yai, MAC-pe9v
- MAC-sqpt depends on: MAC-62s6, MAC-sd7g
- MAC-wbxq depends on: MAC-p9z1, MAC-pe9v, MAC-lnu6
- MAC-rau8 depends on: MAC-sqpt, MAC-wbxq
- MAC-u4oo depends on: MAC-rau8, MAC-62s6, MAC-2n83
- MAC-5ft8 depends on: MAC-u4oo
- MAC-al5u depends on: MAC-5ft8, MAC-bz1y
- MAC-1u2v depends on: MAC-al5u, MAC-u4oo
- MAC-hpqp depends on: MAC-cn7q
- MAC-vx24 depends on: MAC-qlw2, MAC-cn7q, MAC-6h0s, MAC-p9z1, MAC-62s6, MAC-bz1y, MAC-wi2u, MAC-avfp, MAC-imtz, MAC-8yai, MAC-pe9v, MAC-sd7g, MAC-sqpt, MAC-wbxq, MAC-rau8, MAC-u4oo, MAC-5ft8, MAC-al5u, MAC-1u2v
- MAC-ou97 depends on: MAC-qlw2, MAC-cn7q, MAC-6h0s, MAC-p9z1, MAC-62s6, MAC-bz1y, MAC-wi2u, MAC-avfp, MAC-imtz, MAC-8yai, MAC-pe9v, MAC-sd7g, MAC-sqpt, MAC-wbxq, MAC-rau8, MAC-u4oo, MAC-5ft8, MAC-al5u, MAC-1u2v
The direction is consumer depends on producer. Existing old dependencies remain. The normal integration path is schema/store/register + custody/lane/adapters + scoped Git/checks -> replay/Gtd -> final owner -> CLI -> prospective CRM -> complete-example lane -> four-language standalone E2E -> vx24 -> existing guidance -> final ou97. The lhu5 -> hgz1 -> lnu6 -> vx24 chain remains; required execution branches do not point back from custody into hpqp.

INTEGRATION / REQUIREMENT COVERAGE
| Requirement | Responsible delivery | Actual acceptance boundary |
| --- | --- | --- |
| Qualified inventory, closed schemas, reviewed negative/red_control topology | MAC-6h0s | real snapshot-derived declaration validation |
| Exact bytes, modes, empty-directory topology, external store/history | MAC-p9z1 | capture/materialize/transport corruption matrix |
| Empty-head first registration, explicit CAS, idempotent retry, failed history | MAC-62s6 | real store transaction, no execution/PASS prerequisite |
| Native custody, formal/JVM path, original pilot preservation | MAC-qlw2, MAC-cn7q, MAC-hpqp | active nested pinned JVM observed before cancel on both platforms |
| Closed first-release runtime catalog, mandatory lanes | MAC-bz1y | CI+final-preflight union, provision before execution, missing/skip/empty/leak fails |
| Four actual native assertion adapters | MAC-wi2u, MAC-avfp, MAC-imtz, MAC-8yai | pinned Go/node:test+tsc/unittest/ExUnit positive and negative conformance |
| Actual required checks and scoped final Git/Ga | MAC-sd7g, MAC-pe9v | exact state, closed profiles, real Git2.55.0 and same execution owner |
| Retained RED/controls/unsafe/GREEN replay | MAC-sqpt | native structured lifecycle, assertion cause, exact frozen closure |
| Cheap freshness vs actual final replay | MAC-wbxq, MAC-rau8 | no false replay label; nine-stage final no-launch/release/publication/output lifecycle |
| Owner4h/per-milestone cumulative budgets/shared cleanup | MAC-qlw2, MAC-sqpt, MAC-sd7g, MAC-rau8 | actual process-producing call-graph audit + real cumulative timeout tests |
| Ordinary CLI/hook/complete standalone adoption | MAC-u4oo | real binary/payloads, explicit store, unsupported failclosed |
| Preserve complete examples + all legacy regressions | MAC-5ft8, MAC-al5u | prospective strict revisions, no historical PASS or design-only downgrade |
| True first-use each language without prior PASS/tracker | MAC-1u2v, MAC-ou97 | actual init/scaffold/capture/register/RED/GREEN/verify/complete/hooks |
| Shipped exact-byte hard-TDD process | MAC-lnu6, MAC-vx24, MAC-gcrr | template/agent/skill integration, no tokens-equal exemption |
| Final all-assessment behavior, native proof/preflight/candidate | MAC-ou97 + root final gate | every sibling accepted; full preflight only at end, isolated local binary, no installation replacement |

STRUCTURAL CHECKS
- pvg lint --backlog --epic MAC-ui8a --json: EXIT0, actual result is a JSON ARRAY, zero error findings, ten review-only vertical-slice observable-verb heuristics.
- pvg rtm check: EXIT0, zero tagged requirements extracted/covered/uncovered (56 global stories checked,21closed). This mechanical empty-tag result is NOT semantic coverage proof; the mapping above is the manual coverage audit.
- pvg nd dep cycles: EXIT0, no dependency cycles.
- git status --porcelain: EXIT0, empty.
- Root independently confirmed same scoped lint result.

SELF-REVIEW / REVIEW-ONLY FINDING JUSTIFICATION
- MAC-qlw2: accepted with rationale for wording heuristic; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. The outcome says the consumer receives a result or proof; the scanner's narrow verb list misses that wording. ACs require actual observable native outcomes, not an empty horizontal helper.
- MAC-cn7q: clean after syntax/edge correction; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. 
- MAC-6h0s: clean after syntax/edge correction; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. 
- MAC-p9z1: clean after syntax/edge correction; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. 
- MAC-62s6: clean after syntax/edge correction; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. 
- MAC-bz1y: accepted with rationale for wording heuristic; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. The outcome says the consumer receives a result or proof; the scanner's narrow verb list misses that wording. ACs require actual observable native outcomes, not an empty horizontal helper.
- MAC-wi2u: accepted with rationale for wording heuristic; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. The outcome says the consumer receives a result or proof; the scanner's narrow verb list misses that wording. ACs require actual observable native outcomes, not an empty horizontal helper.
- MAC-avfp: accepted with rationale for wording heuristic; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. The outcome says the consumer receives a result or proof; the scanner's narrow verb list misses that wording. ACs require actual observable native outcomes, not an empty horizontal helper.
- MAC-imtz: accepted with rationale for wording heuristic; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. The outcome says the consumer receives a result or proof; the scanner's narrow verb list misses that wording. ACs require actual observable native outcomes, not an empty horizontal helper.
- MAC-8yai: accepted with rationale for wording heuristic; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. The outcome says the consumer receives a result or proof; the scanner's narrow verb list misses that wording. ACs require actual observable native outcomes, not an empty horizontal helper.
- MAC-pe9v: clean after syntax/edge correction; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. 
- MAC-sd7g: accepted with rationale for wording heuristic; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. The outcome says the consumer receives a result or proof; the scanner's narrow verb list misses that wording. ACs require actual observable native outcomes, not an empty horizontal helper.
- MAC-sqpt: accepted with rationale for wording heuristic; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. The outcome says the consumer receives a result or proof; the scanner's narrow verb list misses that wording. ACs require actual observable native outcomes, not an empty horizontal helper.
- MAC-wbxq: clean after syntax/edge correction; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. 
- MAC-rau8: accepted with rationale for wording heuristic; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. The outcome says the consumer receives a result or proof; the scanner's narrow verb list misses that wording. ACs require actual observable native outcomes, not an empty horizontal helper.
- MAC-u4oo: clean after syntax/edge correction; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. 
- MAC-5ft8: accepted with rationale for wording heuristic; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. The outcome says the consumer receives a result or proof; the scanner's narrow verb list misses that wording. ACs require actual observable native outcomes, not an empty horizontal helper.
- MAC-al5u: clean after syntax/edge correction; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. 
- MAC-1u2v: clean after syntax/edge correction; bounded owned files, real positive+negative native/API outcome, exact approved contract, explicit downstream normal integration and frozen regressions. 
- MAC-l7m0: fixed complete authoritative one-doc AC/proof map; no normative change/public private-coordinate leak authorized.
- MAC-hpqp: fixed new custody scope/dependency only, retained all96frozen cases and pending empirical proof.
- MAC-vx24: fixed honest bounded integration ownership, original semantic ACs retained.
- MAC-ou97: clean expanded final tests-only closure, all40epic children ordered with accepted old siblings remaining historical.

EXACT MUTATION AUDIT
- 19 new feature stories created through pvg nd, then canonical parser-supported boundary/observable-outcome comments appended to each; no status transitioned to delivered/accepted.
- MAC-l7m0 received the complete one-document AC comment in this resumed pass, in addition to the already recorded public projection repair.
- MAC-5ft8 received dependency clarification preserving MAC-sh60 and using transitive hgz1 ordering.
- MAC-hpqp/MAC-vx24/MAC-ou97 received approved scope/AC revisions at true EOF; hpqp/vx24 additionally received parser-supported current output maps.
- 77 dependency adds listed above, with supported nd-managed producer Blocks/history backlinks only; no removals.
- Exactly13 missing indented schema signature lines INSERTED in11 newly authored stories using root-authorized supported pvg nd edit with an owned deterministic editor: MAC-qlw2(1),MAC-6h0s(1),MAC-62s6(1),MAC-bz1y(2),MAC-wi2u(1),MAC-avfp(1),MAC-imtz(1),MAC-8yai(1),MAC-pe9v(1),MAC-rau8(1),MAC-al5u(2). Every original Body byte and contract was retained; per-story contemporaneous comments list exact insertions and before/after SHA256. All reads compared to prior Body plus ONLY inserted lines. No --description/body-file replacement or direct unsupervised vault edit.
- MAC-bz1y received CI producer/consumer serialization note preserving MAC-hy71 exact-commit gates.
- This epic handoff comment is the final planning mutation. Local nd-native snapshots only; no sync/push/fetch/pull.
- Owned temporary editor /tmp/machinery-assurance-schema-editor.mjs was created via apply_patch under explicit root clearance, exact13entry/hash/path guards dry-checked before use; it was invoked only by supported nd edit. No source/test/worktree/branch/install/runtime/service/remote mutation.
- Tooling observations retained honestly: read-only guessed discovery paths failed before correct locations were verified; an async invocation syntax error executed no nd command; nd comments add appends an observed terminal newline; scanner processes every historic CONSUMES entry and requires literal PRODUCES:/CONSUMES: markers. Source parser revision c0957106a81346033d7b1d82fde5f434a9db6bab matched installed pvg. Each error stopped until root clearance; no failed command claimed success.

RAW BODY HASHES AFTER FINAL CHILD MUTATIONS (before this epic report)
| ID | UTF8 bytes | SHA256 |
| --- | ---: | --- |
| MAC-1u2v | 10825 | 381a2a7d57c21e318d06ac0ad121482bde02e1fc03ef27cb69710ca701cb5077 |
| MAC-2n83 | 7549 | f1d3f34f278a7bf7a685051836f7aebed15ab9d55d46d90cbcf42b885789fa41 |
| MAC-5ft8 | 14102 | 8a69f7c31d614e4de24e4e892794c6c3ad57097d7d7cc2deb27789879fff4dcb |
| MAC-62s6 | 12690 | 116169252a0219056e6141ba9e5843799000825b666a1172d2b50997defd1140 |
| MAC-6h0s | 13594 | 78d3a80fb1b0c3378883ec7daf3e02d1a8e186116374e42f603100b89a993d27 |
| MAC-8yai | 14710 | 410f643e0491f4732eb6a847de7697e6023d332c01cd2bbfd73ddac4bcfe6fdc |
| MAC-al5u | 12894 | e37d285c0bb763ac999343055b2e8f37637698390aee309c2d555022502cfc64 |
| MAC-avfp | 14781 | 687a2844e30e53cbec2c3d39f83c539b6a256c58e7eb154a448aadf5c4c848fa |
| MAC-bz1y | 14754 | 1501ce46f3a25d4b9b5d0c5d0e713161ea214d787a55ec29a30d30999b0895a7 |
| MAC-cn7q | 11756 | 228426f8b40e4a6c86f5b6a8a50f9c629e6b3b204d24f079e7b7b995f1ee401c |
| MAC-hpqp | 95103 | b7e1dd533905e6c7353e224ef7bdd9a819a8b4922e7ca2e37051d52d4b7102e7 |
| MAC-hy71 | 6603 | c21ba10377ffbd0f538cb3a21625ce62483ece19bc31fbff2062e998eca7e428 |
| MAC-imtz | 14517 | 4d3a1221d7d221b36b6ef617aff40bac813568b91f6e9f6ba8c01f8aee56d952 |
| MAC-l7m0 | 20005 | bcc47cfd554e9a4d1ccb25d5765584f69d15406aed8bf41303e1941efb2b8e5a |
| MAC-lnu6 | 149082 | 9a49a8007a2e86bdf9c55fd01392e7844df56e109c8f70a176ddedb90c35f0ee |
| MAC-ou97 | 17644 | 72c7746412e413506afa5755c785825a98b0da5961fee99366f31bf228147994 |
| MAC-p9z1 | 11024 | a5f9cfd947d2b6df42e08ec1a8938a1987cd2a59b3a008014cde30ccb4e894bc |
| MAC-pe9v | 12931 | 866dbb90980867238a342a6cc98c7eb7a69a8a71dfe89d50c09bf4d99de696ff |
| MAC-qlw2 | 12509 | 0a1658cc39ef26aeda75639188b5c3bf90339d7956bbe113d419819278201746 |
| MAC-rau8 | 14598 | dd97b19e34759b01aad55f6069cfacb2d587aa0655533767ed497f8920f316b9 |
| MAC-sd7g | 11978 | 95e34e0698cd91f8a946df7c3fc7cc558a23f284039d302c60ee0f8766daf3d0 |
| MAC-sqpt | 12136 | 5712d13f4c469adb0121ef572d33e5da9669b67fec38b1bc7518cc166130c40c |
| MAC-u4oo | 12666 | 3bfe4c020a91e20ea912d80a677403b1610fd9a34cba3e0fc4e8bb972c93df1f |
| MAC-vx24 | 23038 | ffdfc505e82a4538cf7d7f3db9f4636b2793dfcf67b603b0f8e96eb371072d45 |
| MAC-wbxq | 9987 | c00ffb101f70bd3d90026a301f9e1f637bf11bec81c4a68cc9af8f47f45d53f7 |
| MAC-wi2u | 14600 | aa6e3d4c52c06ee2c829de8a7a9bd21325d85ab40070074f4353204188b1c7e4 |

DISPATCH/RELEASE HOLD
Independent Anchor review is next. Architecture approval is not code or native evidence. No normal source developer is released by this report. Final full preflight remains last; local merge and isolated candidate only after acceptance/gates. Machinery has no Paivot product/build/test dependency. Installed Machinery/plugins/skills remain untouched until the NIL coordination and root installation clearance. No GH publication during work.

## nd_contract
status: in_progress

### evidence
- Backlog decomposition/repair completed with source/parser validation, exact mutation audit, raw Body hashes and structural gates above.
- Nineteen required stories are new unfinished work; prior six accepted epic deliveries are not reclassified.

### proof
- [x] Approved architecture obligations mapped to bounded, producer-consumer, cycle-free stories with normal integration and final capstone.
- [x] No lint errors; RTM/cycle checks passed; original claims/frozen histories preserved.
- [ ] Independent Anchor backlog approval.
- [ ] All implementation/native proof/final preflight/isolated candidate closure remain pending.


### 2026-09-06T09:36:56Z ramirosalas
# Independent assurance backlog review — round 1 of 3

REVIEW_RESULT: REJECTED

Scope: the nineteen new assurance stories under MAC-ui8a, canonical documentation producer MAC-l7m0, and their impacts on MAC-hpqp/MAC-hy71/MAC-hgz1/MAC-lnu6/MAC-vx24/MAC-ou97. This does not reopen the accepted original backlog review or the approved architecture. No conditional pass.

## Blocking findings

### 1. CRITICAL — executable integration ownership must include the actual required edit surfaces

RULE: A bounded story must own every required production, executable asset, and maintained inventory surface at its real path, or consume an explicitly owned and correctly ordered producer. Naming behavior while excluding the files that implement or register it is not executable decomposition.

ISSUE / all identified instances:

- MAC-cn7q promises scoped formal provisioning/probes/verification and an audit of every reachable process producer, but its exclusive PRODUCES list omits `internal/formal/process.go`. At accepted epic 7e36f3e7ddcf25565d5d4fe60b328df254eee91d, `openFormalJava` in that file constructs `context.Background()`, sanitizes `cmd.Env`, and calls `runBoundedProcess` for the actual JVM identity probe. Its existing API has no scope/context input. Editing only formal.go, runtimeclosure/java.go and processcontrol cannot satisfy the required explicit attachment after this sanitation without an unapproved alternate mechanism or bypass. The same sweep finds `internal/formal/alloy.go`: `runAlloy` calls this probe and separately creates a background-context JVM after environment sanitation. These are existing paths in normal VerifyFormalTo, not hypothetical future gates. Assign the exact scoped call-chain changes and supplemental test/fragment mapping while preserving Java/JAR and original frozen tests; do not silently exclude the existing Alloy branch or duplicate the probe to avoid ownership.
- MAC-al5u adds `scripts/assurance-examples.sh` and `scripts/assurance-examples-test.sh` but does not own `scripts/shellcheck-files.txt`. The existing `scripts/shellcheck-inventory.sh` discovers all `.sh` files under scripts and requires byte-exact equality with that closed file. Both additions necessarily require its update. MAC-hpqp owns the inventory earlier in the dependency chain and cannot register nonexistent future scripts as its completed corpus. Assign the later inventory amendment to the script producer and preserve the existing corpus.
- MAC-vx24's current bounded map says it modifies the existing `repository_contract_test.go`, but that repository-root path does not exist. The actual existing file is `cmd/machinery/repository_contract_test.go`, confirmed by graph discovery and the accepted epic tree. Correct ownership to the real test surface and preserve its existing regression bytes except independently approved RED changes; do not create a replacement root file and call that integration with the existing contract tests.
- MAC-wi2u, MAC-avfp, MAC-imtz and MAC-8yai require embedded executable helpers/transports/reporters/bootstraps, while their exclusive file lists name only `internal/tdd/adapters/assets/{go,typescript,python,elixir}/README.md` and refer generically to an "owned assets directory." This leaves conflicting file-versus-directory scope for the actual executable assets. Explicitly own the concrete helper/transport/embedding surfaces, or unambiguously own each bounded asset directory and require its exact file inventory to receive review before RED. A README is not the helper producer. This is an ownership correction, not a demand to invent future test IDs.

SCOPE: Swept all nineteen new current Bodies and the six named existing integration/serialization consumers. Repair this rule across their production paths, embedded assets and mandatory inventories, preserving serial shared-file ownership. The standalone four-adapter architecture and old frozen pilot remain unchanged.

### 2. CRITICAL — prospective strict migration must own the current judgment refresh it necessarily invalidates

RULE: A migration that changes the complete implementation/test subject and requires current full-root Gv success must assign a downstream, substantive independent judgment and exact evidence refresh after those changes. Earlier accepted evidence, hash-only renewal, or making Gv weaker cannot satisfy the migration.

ISSUE: MAC-5ft8 requires new strict tests/helpers under `examples/go-crm/impl/internal/...` and actual current `check --complete`, but owns no `examples/go-crm/design/attestations.yaml` refresh. The accepted Gv implementation compares the entire recorded versus current entry set and returns `GV_SCOPE_INVENTORY` for added paths (`internal/gates/attest.go`, accepted epic lines 812–847); the new strict tests/helper directories therefore invalidate the current implementation subject established upstream. MAC-hgz1 and MAC-lnu6 both precede this migration. MAC-lnu6 explicitly preserves implementation entries/hash and permits only its narrow policy-related metadata/BUILD-hash changes. MAC-al5u is an invocation owner, MAC-vx24 is guidance/integration, and MAC-ou97 is tests-only; none owns the necessary later subject review/evidence amendment. Thus the migration's required full current complete success has no authorized producer.

FIX: Assign a bounded post-strict-test current-judgment/evidence owner, normally as an explicit MAC-5ft8 scope addition or an ordered consumer. Require actual independent review of the new complete test/implementation inventory, exact generated evidence, preserved prior attribution/history and all legacy regression proof. Include the new subject's stale-before-refresh and current-after-review controls. Keep authored assurance controls finished before judgment and execution, preserve full-root Gv without new exclusions, and do not imply automatic reviewer/date or hash refresh authority. Any resulting example fixture/golden changes need their own exact reviewed ownership; this finding does not authorize altering them.

SCOPE: Swept MAC-5ft8/MAC-al5u/MAC-vx24/MAC-ou97 against the complete current ownership constraints of MAC-hgz1/MAC-lnu6 and architecture sections 4, 7 and 10. The current closed example inventory has one implemented-complete row, Go CRM, and seven explicitly design-only rows; no request to implement or downgrade those seven is made.

## Advisory observations and preserved strengths

- The ten lint review findings are observable-verb heuristics, not additional blockers. The stories have concrete native/API outcomes and a continuous downstream normal CLI/complete path; the missing ownership above is substantive.
- Several cross-references call the schema/store/budget section "section 3" or "sections 2–3", while the normative document places the closed schema/store/budget contract in section 4. The exact document hash still identifies the authority. Correct references during the global ownership repair; do not rewrite normative content.
- Deferred exact native test names are acceptable where separately proposed, independently reviewed and frozen before RED authoring. Do not invent names only to satisfy lint. The runtime/conformance matrices must still become complete exact inventories before execution authorization.
- The decomposition correctly preserves explicit empty-head registration/CAS and failed history, same-assertion red_controls, retained-state replay, four closed native adapters, full current Ga/Gv/Gtd, no-launch/release/publication/output ordering, cumulative milestone and four-hour owner budgets, shared terminal cleanup, fresh first-use without prior PASS, and native-cooperative/unauthenticated-host limits. No architectural redesign is requested.
- hpqp's original seven frozen files/96-leaf union and failed history remain protected; the new custody work is supplemental. hy71 -> bz1y -> al5u serializes shared CI ownership. hgz1 -> lnu6 remains before prospective assurance migration. The existing sh60 FAIL/hold and original vx24/ou97 dependencies are untouched; no successor/cancellation/transfer authority follows from this review.

## Evidence and limits

- FIRST command: `pvg lint --backlog --epic MAC-ui8a --json`, exit 0, JSON array with 0 errors and 10 review-only vertical-slice findings. `pvg rtm check`: exit 0, 0 extracted requirements (56 stories, 21 closed); empty tags are not semantic coverage. `pvg nd dep cycles`: no cycles. `pvg settings design.machinery`: off; no enabled-design gate was bypassed.
- Read all nineteen complete current story Bodies, complete canonical epic Body, complete MAC-l7m0, complete vx24/hy71/ou97, and relevant original/current hpqp/hgz1/lnu6 ownership and history through `pvg nd`; no raw vault reads. Independently recomputed the nineteen plus six reviewed child Body hashes against the canonical handoff table: all match.
- Canonical epic Body before this review: 41223 UTF-8 bytes, SHA256 e83cff18d1901f57e104c874d83eb4152f1254d22b7849612a2194bb72b3e0c5; handoff seal f9d3612bfc3feac8df59ad65cb0d623268700a99. Current nd/backlog d176745699de0277a70bdb660cd7245005165a9a also includes the separate Y amendment; reviewed child hashes remain unchanged.
- Read the full 589-line approved private architecture and complete CHALLENGE-3. Hashes match e467a3b6c6f657e7a4cb28e85c688be0df24c56a7e3c76fdcfa224f7ff5d2624 and 0205b50a8eb2e135230173bbf4a7b7eccdde5934fabab5aa4b2dde95d474ac66. MAC-l7m0 retains exact one-document public projection SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8; it remains blocked and unlanded.
- Graph-first Verify-tier source discovery used project Users-ramirosalas-workspace-machinery, generation 2026-09-06T08:09:26Z; relevant paths reported metadata_match. Non-indexed TSV/text inventories were read directly. Material formal/Gv/path claims were independently checked in the exact accepted epic Git object. Graph coverage is best-effort, not a completeness theorem. Tool-guided missing-project argument was corrected; absent optional local AGENTS was confirmed by root, using supplied global instructions.
- Working tree clean; main remains 497419ab4512fcff765cd5feb27aed4c67b5608d and epic remains 7e36f3e7ddcf25565d5d4fe60b328df254eee91d. No source/test/worktree/installed/runtime/service/remote changes, no child agents, no native tests or heavy preflight run. This is backlog judgment, not architecture feasibility or native delivery evidence.

## nd_contract
status: in_progress

### evidence
- Independent round-1 assurance backlog review completed; two blocking general rules with full identified instances and sweep scopes above.
- Report is reviewer-owned and immutable after creation. Only this review may be appended to the epic; no child status/scope/dependency changes.

### proof
- [x] New decomposition reviewed against the exact accepted architecture and relevant existing ownership/frozen-history constraints.
- [x] Mechanical lint/RTM/cycle results and exact source-backed integration gaps recorded honestly.
- [ ] Independent assurance backlog approval: REJECTED pending the two bounded general-rule repairs.
- [ ] Implementation/native two-platform proof/final preflight/isolated candidate remain pending.

