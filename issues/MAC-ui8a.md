---
id: MAC-ui8a
title: "Deterministic assurance hardening"
status: open
priority: 2
type: epic
created_at: 2026-09-05T19:28:10Z
created_by: ramirosalas
updated_at: 2026-09-24T21:33:45Z
content_hash: "sha256:7fbd37018b1ebb4353d284c34364c743113ad7ecb238c80fc1f458303d81ecaa"
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
Triage 2026-09-24: 42 of 53 children closed. Remaining 10 are the replay/CLI half; none of their files exist yet. P0 -> P2: nothing broken for consumers (0.8.0 through 0.10.1 shipped without it); raise if consumer executable assurance is wanted before MAC-cup9. Child bodies embed 09-06 session state (no-push, pushurl guard, preflight-final-only, ci.yml ownership); CI now lives in .dagger/main.go. Refresh each body before dispatch.

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


### 2026-09-06T09:54:20Z ramirosalas
## Assurance backlog ROUND1 general-rule repair — sealed handoff for independent ROUND2

This is an append-only Sr PM repair report, not approval, delivery, test execution evidence, or dispatch authorization. Architecture and user scope are unchanged. Root's explicit independent-review hold remains authoritative for l7 and every new assurance slice.

### Review basis and exact source verification

- Entire independent ROUND1 review read: /tmp/machinery-assurance-anchor-r1.Xehbyn/REVIEW.md, SHA256 ce2fdf7a20b2bf32ad6189553e0559ef0c6af75a4bab38f37de2f566f23a5a63. Its two critical findings were generalized across all 19 new stories and affected existing consumers; its non-blocking section-navigation advisory was also repaired.
- Pre-repair review handoff: epic Body SHA256 3d3cb362bf1c5281045f4a8ed83ca9df7d96bb3c8d474372052230fbf5dac9aa at nd/backlog aa958077fa5e796869df32fed6c31973b050cc06. No historical review text was replaced.
- Read-only source verified against accepted epic 7e36f3e7ddcf25565d5d4fe60b328df254eee91d. Verify-tier graph generation 2026-09-06T08:09:26Z was ready; direct callers/callees traced for openFormalJava, runAlloy and VerifyFormalTo. Relevant coverage was metadata_match; scripts/shellcheck-files.txt had not_tracked coverage and was read directly. Coverage metadata is not a completeness theorem.
- Actual normal path includes cmd/machinery/stubs.go:newVerifyFormalCmd -> formal.VerifyFormalTo -> TLC and Alloy, shared openFormalJava -> runBoundedProcess. Existing openFormalJava uses its own Background timeout; Alloy separately uses Background after Environment. Thus process.go, alloy.go AND the narrow stubs.go command context handoff are explicitly owned by cn7q. Compatibility wrappers may remain; the scoped normal path must never silently fall through an unscoped wrapper.
- Actual existing repository contract suite is cmd/machinery/repository_contract_test.go; no root-level file exists. Current vx24 scope withdraws the historical wrong path without deleting its historical bytes.
- Read actual full ShellCheck inventory and inventory validator; both new al5u scripts require exact persistent corpus entries. Read actual Gv inventory-first/current-content checks in internal/gates/attest.go and current CLI generator in cmd/machinery/attest.go; full-root subject additions yield GV_SCOPE_INVENTORY and later modifications yield GV_STALE_CONTENT. No new exclusion or weakened judgment is permitted.

### General rule 1: every real executable path has an owner and consumer

cn7q now has a complete 11-file scoped custody/formal/CLI map with both TLC and Alloy, their shared probe, explicit context/scope after every environment sanitization, and real native two-platform integration. hpqp consumes the exact reviewed supplemental process/JVM/probe/command case inventory in custody.json while its seven frozen 96-case pilot files and all prior proof remain unchanged. This is a prospective supplemental inventory requirement, not invention or retroactive acceptance of future test IDs.

Each of the four adapter stories explicitly owns its language-specific executable asset directory under internal/tdd/adapters/assets, bounded to eight executable/config files plus README; existing adapter source owns embedding, materialization, byte validation and transport. Every exact filename/role/entrypoint/import and planned test is independently inventoried before RED; README alone is not a helper. Production adapter -> replay/complete must execute the actual embedded helper. Missing, altered, unembedded, extra or unused assets fail; frozen byte closure includes their real support/config inputs. Maximum 15 actual files and 2100 lines per adapter is a review boundary, not permission to expand silently.

al5u owns scripts/shellcheck-files.txt in its complete current output map and must prove both scripts are actually linted with omission/duplicate/phantom/ordering/missing-script negative controls. It consumes the existing corpus rather than inserting nonexistent future paths earlier. vx24 owns the actual cmd package regression suite and must retain/run existing regressions under its reviewed RED amendment rules. bz1y records downstream asset-inventory obligations without a reverse dependency; final lane union must account for actual production helper execution, not README presence.

### General rule 2: changes to a current assured subject require new substantive judgment

MAC-5ft8 now explicitly owns examples/go-crm/design/attestations.yaml and cmd/machinery/go_crm_assurance_migration_test.go alongside its existing strict migration outputs. Its complete ten-AC revision preserves all legacy regressions, full implementation/test inventory, prior histories, and the preceding lhu5 -> hgz1 -> lnu6 ordering.

All new authored tests, controls, helpers, config and immutable variant references must be final before the independent judgment and execution they support. The real current Gv path must reject retained pre-migration records after root additions. A substantive independent review then examines the entire changed current subject, assertion adequacy and actual runtime/legacy evidence; only after that may the existing standalone CLI generate exact v2 current records for affected claims with real attribution and dates. Hash generation is not judgment or authenticated identity. Historical rows and unaffected evidence remain intact. Current-after normal Gv/strict complete must succeed, while old-record reinsertion, missing judgment or subsequent subject mutation must fail.

The authored migration driver supplies stale-before/current-after negative controls; it does not mutate the canonical repository during acceptance. No automatic reviewer/date, hash-only refresh, newly excluded test/helper path, downgraded claim, historical/current relabeling or implementation-only narrower root is authorized. Later actual golden/fixture changes require separately reviewed exact ownership and renewed substantive judgment. al5u, vx24 and final ou97 explicitly consume this current full-root assurance; they receive no authority to manufacture it or edit later fixtures. These obligations do not authorize rewriting hgz1/lnu6 protected scope, evidence or frozen tests.

### Complete 19-story sweep

| Story | Generalized rule disposition |
|---|---|
| MAC-qlw2 | Custody scope, limits and deadline producer retained; schema/store/budget navigation corrected to section 4; no persistent GoCRM subject edits. |
| MAC-cn7q | Complete owned real call graph now includes formal/process.go openFormalJava, formal/alloy.go runAlloy and cmd/machinery/stubs.go newVerifyFormalCmd; context/scope after sanitation and both JVM branches explicitly tested. |
| MAC-6h0s | Closed manifest/inventory/topology producer retained; section 4 navigation corrected; no adapter/helper or GoCRM ownership assumed. |
| MAC-p9z1 | External store/capture/CAS producer retained; section 4 navigation corrected; captures do not confer judgment or execution proof. |
| MAC-62s6 | Explicit registration/head-CAS producer retained; section 4 navigation corrected; no implicit initial PASS or hash-only attestation. |
| MAC-bz1y | Existing closed integration lane/catalog producer retained; exact executable adapter-asset inventories are downstream producer obligations, verified by final union without a reverse dependency. |
| MAC-wi2u | Go adapter owns assets/go directory, executable support, embedding/materialization/byte binding and transport in go.go; exact bounded asset/file inventory reviewed before RED. |
| MAC-avfp | TypeScript adapter owns assets/typescript directory, executable support, embedding/materialization/byte binding and transport in typescript.go; exact bounded inventory reviewed before RED. |
| MAC-imtz | Python adapter owns assets/python directory, executable support, embedding/materialization/byte binding and transport in python.go; exact bounded inventory reviewed before RED. |
| MAC-8yai | Elixir adapter owns assets/elixir directory, executable support, embedding/materialization/byte binding and transport in elixir.go; exact bounded inventory reviewed before RED. |
| MAC-pe9v | Scoped ordinary-Git runtime closure and replayed materialization ownership retained; consumes corrected custody/formal producer rather than creating hidden helper authority. |
| MAC-sd7g | Closed native check executor ownership retained; all four real executable adapters remain required producers, no generic output-regex substitute. |
| MAC-sqpt | Retained-source RED/control/challenge/GREEN replay and cumulative budget ownership retained; adapters supply executable assets, receipts do not replace execution. |
| MAC-wbxq | Gt/Gd freshness and scoped Ga gate ownership retained; current implementation versus historical milestone distinction unchanged; no Gv exclusion introduced. |
| MAC-rau8 | Assurance finalization/release/publication ownership retained; real native call graph and cumulative owner/milestone deadlines remain mandatory; no early publish/launch shortcut. |
| MAC-u4oo | Standalone normal CLI/check/complete/hook integration ownership retained; root context reaches actual producers; no Paivot runtime/build/test dependence. |
| MAC-5ft8 | Owns prospective strict GoCRM subjects, exact current attestations.yaml and new bounded real-CLI migration driver; all authored controls precede substantive independent full-root judgment and execution; stale-before/current-after/invalidated-after controls required. |
| MAC-al5u | Owns exact sorted ShellCheck corpus for both new scripts, plus example lane/CI/preflight wiring; consumes 5ft8's current substantive judgment, keeps one complete GoCRM and seven design-only examples. |
| MAC-1u2v | Standalone true-first-use and two-platform capstone retained; consumes completed example migration through existing ordering, no prior PASS receipt or private tracker prerequisites. |

Affected existing consumers repaired: hpqp supplemental execution inventory; vx24 complete real-file integration and current-assurance consumption; ou97 final real-path/current-judgment closure. l7, hy71, hgz1 and lnu6 Bodies/statuses/labels/dependency arrays remain exactly unchanged by this round. sh60 has no successor, cancellation, exception or dependency transfer. Protected Y received no Sr PM mutation this round; a concurrent actor appended supplemental RED evidence, recorded separately below rather than being overwritten.

Section-reference-only notes in qlw2, 6h0s, p9z1 and 62s6 correct schema/store/registration/budget citations to section 4. Section 3 still correctly governs obligations/activation. No normative schema/API/argv/status/limit or approved public projection changed.

### Exact mutation audit

Exactly 15 child operations were supported pvg nd comments add; each actual full Body was read back, original Body preserved as exact prefix, and submitted text verified at true EOF with only nd's observed extra terminal newline. All 15 prior statuses, labels, BlockedBy and Blocks arrays were verified unchanged. No dependency operation, guarded editor, status/label/claim mutation or original-byte modification occurred. Historical path text did not need editing because scoped lint has no errors.

| Story | Body SHA256 before | Body SHA256 after |
|---|---|---|
| MAC-cn7q | 228426f8b40e4a6c86f5b6a8a50f9c629e6b3b204d24f079e7b7b995f1ee401c | cbb959ca1a5a75f0d13b30272568c66f102d752fa748d80bf8741760cc0bd0ca |
| MAC-al5u | e37d285c0bb763ac999343055b2e8f37637698390aee309c2d555022502cfc64 | f5581cc389f1660f1a31770d13c18aac3feb973f3a67ee5dbc4d070ec252679c |
| MAC-vx24 | ffdfc505e82a4538cf7d7f3db9f4636b2793dfcf67b603b0f8e96eb371072d45 | c2a5711dfc651e9a95dfaa8c6e5a9b78d2bc8b97acd5fe9e54ea261fe0fd6179 |
| MAC-wi2u | aa6e3d4c52c06ee2c829de8a7a9bd21325d85ab40070074f4353204188b1c7e4 | c7be0c62a9f24ad96290d46cd4f6a9e472ed96b8f984d4e6ee5ceebc00a14cb9 |
| MAC-avfp | 687a2844e30e53cbec2c3d39f83c539b6a256c58e7eb154a448aadf5c4c848fa | 48b9cd47a3512c4eee8da38b9b655c193a51001af8d5d1a4bc778fcbcbe5a8b6 |
| MAC-imtz | 4d3a1221d7d221b36b6ef617aff40bac813568b91f6e9f6ba8c01f8aee56d952 | 2612ac65099577d73467d9a59c9440e56c235f687449d0bed288fcd661830b75 |
| MAC-8yai | 410f643e0491f4732eb6a847de7697e6023d332c01cd2bbfd73ddac4bcfe6fdc | 706977331aa519bed965eea14fa11d3d72e8abd49dc7ac4e9c789e858ff1dea7 |
| MAC-5ft8 | 8a69f7c31d614e4de24e4e892794c6c3ad57097d7d7cc2deb27789879fff4dcb | 86d60c6279716347f7c022a7353108aceedcd8ef90cec0c0e16749f13b66fa16 |
| MAC-hpqp | b7e1dd533905e6c7353e224ef7bdd9a819a8b4922e7ca2e37051d52d4b7102e7 | a50938351e48f3c3c513765999e2df87c7e3c9adb7286af61991815f7b79574c |
| MAC-bz1y | 1501ce46f3a25d4b9b5d0c5d0e713161ea214d787a55ec29a30d30999b0895a7 | 9b5d9e2ad2a9dc9078ffd64c65d7d1fb94f6e81cb4c18dfedf80994d66aaabab |
| MAC-ou97 | 72c7746412e413506afa5755c785825a98b0da5961fee99366f31bf228147994 | a946be6cc705da9843c8060965ce9a6708114d29c6930ea15b639d5bdd713274 |
| MAC-qlw2 | 0a1658cc39ef26aeda75639188b5c3bf90339d7956bbe113d419819278201746 | a2f58a0d549beb3cd50bd1b7069026f1d996477d12acbdd4e41d456e291e9850 |
| MAC-6h0s | 78d3a80fb1b0c3378883ec7daf3e02d1a8e186116374e42f603100b89a993d27 | 362676f2ba11fe3ad04f4a5a8c43c2234c4cab240a5fc66c69c5eea48729eaa9 |
| MAC-p9z1 | a5f9cfd947d2b6df42e08ec1a8938a1987cd2a59b3a008014cde30ccb4e894bc | fe89f7b48df3d56b8b1dd93f04632ae1c69017d8933ad6752274bf83becc1998 |
| MAC-62s6 | 116169252a0219056e6141ba9e5843799000825b666a1172d2b50997defd1140 | 22e96e7e11727b13ab2f74af098ee2a7e954be43f3b21f5fc62f807a4a1a3b65 |

Other observed current Body hashes:
- MAC-pe9v: 866dbb90980867238a342a6cc98c7eb7a69a8a71dfe89d50c09bf4d99de696ff; unchanged.
- MAC-sd7g: 95e34e0698cd91f8a946df7c3fc7cc558a23f284039d302c60ee0f8766daf3d0; unchanged.
- MAC-sqpt: 5712d13f4c469adb0121ef572d33e5da9669b67fec38b1bc7518cc166130c40c; unchanged.
- MAC-wbxq: c00ffb101f70bd3d90026a301f9e1f637bf11bec81c4a68cc9af8f47f45d53f7; unchanged.
- MAC-rau8: dd97b19e34759b01aad55f6069cfacb2d587aa0655533767ed497f8920f316b9; unchanged.
- MAC-u4oo: 3bfe4c020a91e20ea912d80a677403b1610fd9a34cba3e0fc4e8bb972c93df1f; unchanged.
- MAC-1u2v: 381a2a7d57c21e318d06ac0ad121482bde02e1fc03ef27cb69710ca701cb5077; unchanged.
- MAC-l7m0: bcc47cfd554e9a4d1ccb25d5765584f69d15406aed8bf41303e1941efb2b8e5a; unchanged.
- MAC-hy71: c21ba10377ffbd0f538cb3a21625ce62483ece19bc31fbff2062e998eca7e428; unchanged.
- MAC-hgz1: 2a323bafd5f50877286b79ea45243945c109ad9e7dcfcd59933a22dab9c5e82e; unchanged.
- MAC-lnu6: 9a49a8007a2e86bdf9c55fd01392e7844df56e109c8f70a176ddedb90c35f0ee; unchanged.
- MAC-yig6: 2db0a7f18ca4e3f3da1fcee8e5d7addb10771984bcdd9224ce6f040d920a46df; concurrently changed by another actor; no mutation by Sr PM.

Y's concurrent current Body ends in supplemental RED commit ac439c517de169addb68637c50040a9434bc8cca evidence; this report makes no independent acceptance claim for that work. Its concurrent change is not part of these 15 mutations.

### Existing dependency DAG retained — zero new edges

- MAC-qlw2 depends on: MAC-l7m0.
- MAC-cn7q depends on: MAC-qlw2.
- MAC-6h0s depends on: MAC-qlw2.
- MAC-p9z1 depends on: MAC-6h0s.
- MAC-62s6 depends on: MAC-p9z1.
- MAC-bz1y depends on: MAC-hpqp, MAC-6h0s, MAC-hy71.
- MAC-wi2u depends on: MAC-bz1y, MAC-6h0s.
- MAC-avfp depends on: MAC-bz1y, MAC-6h0s.
- MAC-imtz depends on: MAC-bz1y, MAC-6h0s.
- MAC-8yai depends on: MAC-bz1y, MAC-6h0s.
- MAC-pe9v depends on: MAC-cn7q, MAC-bz1y.
- MAC-sd7g depends on: MAC-wi2u, MAC-avfp, MAC-imtz, MAC-8yai, MAC-pe9v.
- MAC-sqpt depends on: MAC-62s6, MAC-sd7g.
- MAC-wbxq depends on: MAC-p9z1, MAC-pe9v, MAC-lnu6.
- MAC-rau8 depends on: MAC-sqpt, MAC-wbxq.
- MAC-u4oo depends on: MAC-rau8, MAC-62s6, MAC-2n83.
- MAC-5ft8 depends on: MAC-u4oo.
- MAC-al5u depends on: MAC-5ft8, MAC-bz1y.
- MAC-1u2v depends on: MAC-al5u, MAC-u4oo.
- MAC-l7m0 depends on: (none; explicit review hold still applies).
- MAC-hpqp depends on: MAC-cn7q.
- MAC-hy71 depends on: MAC-hpqp.
- MAC-hgz1 depends on: MAC-lhu5.
- MAC-lnu6 depends on: MAC-hgz1.
- MAC-vx24 depends on: MAC-l7m0, MAC-sh60, MAC-2n83, MAC-lnu6, MAC-hpqp, MAC-qlw2, MAC-cn7q, MAC-6h0s, MAC-p9z1, MAC-62s6, MAC-bz1y, MAC-wi2u, MAC-avfp, MAC-imtz, MAC-8yai, MAC-pe9v, MAC-sd7g, MAC-sqpt, MAC-wbxq, MAC-rau8, MAC-u4oo, MAC-5ft8, MAC-al5u, MAC-1u2v.
- MAC-ou97 depends on: MAC-hlae, MAC-sh60, MAC-yhg5, MAC-hwdb, MAC-2n83, MAC-hy71, MAC-gcrr, MAC-l7m0, MAC-vx24, MAC-lnu6, MAC-hpqp, MAC-yig6, MAC-lhu5, MAC-hgz1, MAC-qlw2, MAC-cn7q, MAC-6h0s, MAC-p9z1, MAC-62s6, MAC-bz1y, MAC-wi2u, MAC-avfp, MAC-imtz, MAC-8yai, MAC-pe9v, MAC-sd7g, MAC-sqpt, MAC-wbxq, MAC-rau8, MAC-u4oo, MAC-5ft8, MAC-al5u, MAC-1u2v.

The full graph has no cycles. Already-sufficient producer/consumer ordering is retained, including hgz1 -> lnu6 -> vx24 and hpqp -> hy71 -> bz1y serialization. All 19 required deliveries remain blockers of vx24 and ou97; final capstone is not bypassed.

### Scoped mechanical checks and limits

- pvg lint --backlog --epic MAC-ui8a --json: exit 0. Output is a JSON ARRAY, not an object. Exactly ten review-only vertical-slice observable-verb heuristics for 5ft8, 8yai, avfp, bz1y, imtz, qlw2, rau8, sd7g, sqpt and wi2u; zero errors. These are the same non-blocking heuristics identified by independent ROUND1, not claimed semantic approval.
- pvg rtm check: exit 0; zero extracted/covered/uncovered tagged requirements, 56 global stories checked, 21 closed. Empty tagged extraction is not evidence of full semantic coverage; the explicit rule sweep above is the review artifact.
- pvg nd dep cycles: exit 0, No dependency cycles found.
- git status --porcelain: empty. HEAD remains 497419ab4512fcff765cd5feb27aed4c67b5608d; accepted epic remains 7e36f3e7ddcf25565d5d4fe60b328df254eee91d. No source/test edits, worktree/ref/remote changes, services, installations, heavy preflight, source builds or runtime execution were performed.
- Read-only diagnostics were honestly corrected: graph index_status first needed the documented project argument; a prior-turn missing in-memory map was reconstructed from canonical live notes; one large raw nd show exceeded output size and was rerun through a compact exact Body-hash/prefix verifier. None caused partial tracker mutations or a claim of successful truncated inspection.

## nd_contract
status: in_progress

### evidence
- Both independent ROUND1 general rules and the section-navigation advisory repaired by 15 append-only child notes; all current child scope/AC contracts validated at actual EOF.
- Full 19-story rule sweep, unchanged dependency DAG, exact before/after hashes and protected-state distinctions recorded above.
- Scoped lint exits 0 with zero errors; RTM and dependency-cycle checks exit 0, with their limits stated.
- No normal developer dispatch, implementation acceptance, native proof, heavy preflight, installation replacement or remote publication claimed.

### proof
- [x] Every missing actual integration surface has bounded producer ownership and required consumer accounting.
- [x] GoCRM post-change substantive current judgment, stale/current controls and evidence refresh have an explicit owner and ordering.
- [x] Prior histories/frozen contracts preserved; no dependency, status, label or claim changes by this repair.
- [ ] Independent assurance backlog ROUND2 approval remains required.
- [ ] Actual implementation, native two-platform proof, final integrated preflight and isolated local candidate remain pending.


### 2026-09-06T10:06:32Z ramirosalas
# Independent assurance backlog review — ROUND 2/3

REVIEW_RESULT: APPROVED

Zero blocking findings. This is approval of the repaired backlog contracts, not implementation acceptance, execution evidence, or permission to bypass the existing documentation, RED-review, native-platform, or final integration gates.

## Reviewed authority and raw gates

- Epic MAC-ui8a full canonical Body through its actual EOF, including “Assurance backlog ROUND1 general-rule repair — sealed handoff for independent ROUND2”: 71893 UTF-8 bytes, SHA256 f5cfe52b4de7196b4f30cd3c97b9a8d9ed69d6c0f1ed662f53624a519fff1171. Sealed handoff snapshot: 7e8a87512cb7fd960da966e51c7d8559b68d43c3.
- Full current contracts for all 19 assurance children, with the previous complete-contract review retained and every appended repair read completely. A fresh pvg nd show raw-Body verifier independently checked the exact hashes below and all 15 original prefixes. The seven untouched new contracts have identical hashes; hgz1, hy71, lnu6 and l7m0 also retain their reviewed hashes.
- Previous immutable review: /tmp/machinery-assurance-anchor-r1.Xehbyn/REVIEW.md, SHA256 ce2fdf7a20b2bf32ad6189553e0559ef0c6af75a4bab38f37de2f566f23a5a63.
- Normative architecture: /tmp/machinery-assurance-architecture.fXW3vg/test-assurance-contract-v3-final.md, full 589 lines previously reviewed; SHA256 reconfirmed e467a3b6c6f657e7a4cb28e85c688be0df24c56a7e3c76fdcfa224f7ff5d2624. Independent CHALLENGE-3.md SHA256 reconfirmed 0205b50a8eb2e135230173bbf4a7b7eccdde5934fabab5aa4b2dde95d474ac66.
- Exact public projection remains SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8. MAC-l7m0 remains blocked and documentation-only; private identifiers, paths and operational chronology must not enter that public document.
- First round-two task check was pvg lint --backlog --epic MAC-ui8a --json: exit 0, JSON array, zero errors and the same ten review-only observable-verb heuristics. Manual review below resolves their semantic question; lint is not semantic proof.
- pvg nd dep cycles: exit 0, No dependency cycles found. Exact BlockedBy arrays checked against the handoff; no new edges were needed for these repairs.
- Source tree remains clean. HEAD 497419ab4512fcff765cd5feb27aed4c67b5608d and epic/MAC-ui8a 7e36f3e7ddcf25565d5d4fe60b328df254eee91d were freshly verified. nd/backlog was concurrently advanced to 4b9d2aee1e9c34ef1bddbbfafddecdbab9899e40 while the reviewed epic Body remained exact. This is not a claim that all tracker metadata or Y stayed unchanged.
- Read-only graph discovery and exact epic-source inspection confirmed cmd/machinery/stubs.go:newVerifyFormalCmd actually calls formal.VerifyFormalTo without an explicit command-context argument today. Thus the repaired narrow command handoff is a real required surface, not a speculative addition. Graph coverage was metadata_match/no_recorded_issue at generation 2026-09-06T08:09:26Z; that best-effort signal is not proof of completeness. Prior exact-source findings on formal/process.go, formal/alloy.go, ShellCheck corpus, the actual cmd package repository test, and full-root Gv remain applicable because source refs did not move.

## General-rule disposition

### Rule 1 — every required executable surface must have bounded ownership and a real consumer: FIXED

MAC-cn7q now explicitly owns the full eleven-file process/formal integration map, including internal/formal/process.go, internal/formal/alloy.go and the narrow newVerifyFormalCmd portion of cmd/machinery/stubs.go. Its contract requires the actual command context/scope to reach both TLC and Alloy plus their shared probe, with attachment after environment sanitization and inherited cumulative budgets. Compatibility wrappers cannot become silent unscoped fallback paths. Real native coverage on both supported platforms is required. MAC-hpqp consumes the separately reviewed supplemental inventory while its original seven-file, 96-case pilot and failed history remain frozen.

All four adapter producers now own unambiguous language-specific executable asset directories, bounded to eight executable/config files plus README, and explicitly own their embedding/materialization/byte validation/transport inside the corresponding production adapter source. Exact filenames, roles, entrypoints, imports and test inventory must receive independent review before RED. The missing executable ownership is repaired without inventing unapproved future test names. Missing, altered, unembedded, extra or unused assets cannot be satisfied by a README or synthetic receipt. bz1y and the final lanes account for real production helper execution.

MAC-al5u now owns scripts/shellcheck-files.txt and preserves the existing closed corpus while adding both new scripts. Exact-corpus omission, duplicate, sorting, phantom and missing-script controls are explicit. MAC-vx24 now owns the actual cmd/machinery/repository_contract_test.go; the historical nonexistent root path is expressly superseded, not silently implemented as a replacement suite. Existing test amendments still require their own exact reviewed RED inventory. hpqp, bz1y, al5u, vx24 and ou97 have explicit consumer obligations.

### Rule 2 — a changed currently assured subject requires substantive renewed judgment, not a stale or hash-only record: FIXED

MAC-5ft8 now owns examples/go-crm/design/attestations.yaml and the new real candidate-CLI migration test alongside its strict subject/helper/manifests. All authored subject and control bytes must be finalized before the independent judgment and claimed execution. AC4 proves current full-root Gv rejects the earlier record after additions with GV_SCOPE_INVENTORY. AC5 requires independent substantive review of the complete changed implementation/test subject, assertion adequacy, controls, runtime dispositions and legacy evidence, with real reviewer findings/attribution. AC6 then uses the existing actual Machinery attest CLI to generate exact current v2 rows for every affected claim; generated hashes alone are explicitly insufficient.

AC7 requires current-after-review success on the exact finalized subject and refusal for old-record reinsertion, omitted judgment or subsequent subject/control/mode/topology changes. AC8 requires the actual first-use/register/RED/GREEN/replay/complete workflow on both native platforms and separates earlier calibration/history from final current assurance. Existing regressions, historical rows and frozen source/proof remain protected. Scope does not authorize later fixture/golden edits, automatic attribution, full-root exclusions or hidden legacy-test deletion. Existing hgz1 -> lnu6 ordering remains upstream; 5ft8, not those earlier owners, is responsible for its later change and new judgment. al5u, vx24 and ou97 consume that current evidence.

The section-navigation advisory is also fixed in qlw2, 6h0s, p9z1 and 62s6: schema/store/registration/budget definitions point to normative section 4, while obligation/activation references correctly remain section 3. No normative redesign occurred.

## Complete 19-story semantic sweep

| Contract | Reviewed producer/consumer and assurance disposition |
|---|---|
| MAC-qlw2 | Concrete custody/deadline API feeds all scoped consumers; owner and shared terminal cleanup budgets remain cumulative, not renewed. |
| MAC-cn7q | Actual process/JVM/probe/CLI call graph now wholly owned; real two-platform scope/cancellation/containment controls required. |
| MAC-6h0s | Closed plan/inventory/topology/assertion schema producer; duplicate/unknown/missing/extra input rejection and full reviewed inventory retained. |
| MAC-p9z1 | External store/init/capture producer; explicit disjointness, immutable retained source and history remain mandatory. |
| MAC-62s6 | Explicit head-CAS registration with retained controls; first use does not require or infer a prior PASS. |
| MAC-bz1y | Closed lane/runtime catalog and contributor inventories have real downstream users; hpqp -> hy71 -> bz1y serial ownership retained. |
| MAC-wi2u | Closed native Go adapter, actual embedded executable assets and bound same-assertion evidence, not generic textual test output. |
| MAC-avfp | Closed native TypeScript adapter, actual executable/config assets and transport; native full inventory plus adversarial controls retained. |
| MAC-imtz | Closed native Python adapter, actual helper/import surface and transport; all assertions and negative controls remain required. |
| MAC-8yai | Closed native Elixir/OTP adapter, actual assets/transport/runtime identities; real process and assertion evidence required. |
| MAC-pe9v | Actual scoped Git runtime/materialization Ga producer, not a fabricated gate result; ordinary Git closure and launch controls retained. |
| MAC-sd7g | Closed native check execution consumes all four executable adapters and scoped Git; selectors cannot hide required leaves. |
| MAC-sqpt | Retained baseline, every same-assertion control, unsafe challenge and current GREEN replay; recorded receipts cannot replace execution. |
| MAC-wbxq | Cheap Gt/Gd freshness remains distinct from full current verification; full-root Gv and historical/current distinction preserved. |
| MAC-rau8 | Final owner retains execution-close, writer/CAS reacquisition, closing child, no-launch, runtime/materialization close, release, external publication, output closure and only-then Verification construction. |
| MAC-u4oo | Real standalone normal CLI/hook/check/complete integration consumes those producers; no Paivot product/runtime/build/test dependency. |
| MAC-5ft8 | Full prospective CRM migration now owns current substantive judgment/evidence and actual stale/current controls without rewriting old regressions/history. |
| MAC-al5u | Complete-example native lane, real closed ShellCheck corpus and CI/preflight wiring; one implemented-complete GoCRM is not downgraded. |
| MAC-1u2v | Tests-only standalone true-first-use four-language capstone with no prior PASS and required Darwin arm64 plus Linux amd64 native evidence. |

The ten lint heuristics correspond to observable behavior contracts with executable consumers, positive and negative outcomes, and reviewable frozen inventories; they are not horizontal dead ends. Exact future test IDs remain an explicit pre-RED independent-review obligation where not yet approved. This is not a waiver of inventory completeness.

Cross-story closure retains all nineteen deliveries as blockers of vx24 and ou97, existing serialized overlap on hpqp/hy71, hgz1/lnu6 and final integration surfaces, all forty final children and the six already accepted children. The full-root Gv policy is not narrowed by the assurance trees. Native proof must cover both target platforms and the real nested JVM/probe and language asset paths; cross-compilation, mocks or purported authenticated identity on an unauthenticated host cannot substitute. The four-hour owner, cumulative per-milestone wall ceilings and shared terminal cleanup remain required. MAC-sh60's retained failed audit has no exception, cancellation, successor or transferred obligation here.

## Exact reviewed raw UTF-8 Body identities

| Story | Bytes | SHA256 |
|---|---:|---|
| MAC-qlw2 | 13925 | a2f58a0d549beb3cd50bd1b7069026f1d996477d12acbdd4e41d456e291e9850 |
| MAC-cn7q | 18618 | cbb959ca1a5a75f0d13b30272568c66f102d752fa748d80bf8741760cc0bd0ca |
| MAC-6h0s | 15010 | 362676f2ba11fe3ad04f4a5a8c43c2234c4cab240a5fc66c69c5eea48729eaa9 |
| MAC-p9z1 | 12440 | fe89f7b48df3d56b8b1dd93f04632ae1c69017d8933ad6752274bf83becc1998 |
| MAC-62s6 | 14106 | 22e96e7e11727b13ab2f74af098ee2a7e954be43f3b21f5fc62f807a4a1a3b65 |
| MAC-bz1y | 16974 | 9b5d9e2ad2a9dc9078ffd64c65d7d1fb94f6e81cb4c18dfedf80994d66aaabab |
| MAC-wi2u | 19363 | c7be0c62a9f24ad96290d46cd4f6a9e472ed96b8f984d4e6ee5ceebc00a14cb9 |
| MAC-avfp | 19628 | 48b9cd47a3512c4eee8da38b9b655c193a51001af8d5d1a4bc778fcbcbe5a8b6 |
| MAC-imtz | 19322 | 2612ac65099577d73467d9a59c9440e56c235f687449d0bed288fcd661830b75 |
| MAC-8yai | 19511 | 706977331aa519bed965eea14fa11d3d72e8abd49dc7ac4e9c789e858ff1dea7 |
| MAC-pe9v | 12931 | 866dbb90980867238a342a6cc98c7eb7a69a8a71dfe89d50c09bf4d99de696ff |
| MAC-sd7g | 11978 | 95e34e0698cd91f8a946df7c3fc7cc558a23f284039d302c60ee0f8766daf3d0 |
| MAC-sqpt | 12136 | 5712d13f4c469adb0121ef572d33e5da9669b67fec38b1bc7518cc166130c40c |
| MAC-wbxq | 9987 | c00ffb101f70bd3d90026a301f9e1f637bf11bec81c4a68cc9af8f47f45d53f7 |
| MAC-rau8 | 14598 | dd97b19e34759b01aad55f6069cfacb2d587aa0655533767ed497f8920f316b9 |
| MAC-u4oo | 12666 | 3bfe4c020a91e20ea912d80a677403b1610fd9a34cba3e0fc4e8bb972c93df1f |
| MAC-5ft8 | 23055 | 86d60c6279716347f7c022a7353108aceedcd8ef90cec0c0e16749f13b66fa16 |
| MAC-al5u | 17996 | f5581cc389f1660f1a31770d13c18aac3feb973f3a67ee5dbc4d070ec252679c |
| MAC-1u2v | 10825 | 381a2a7d57c21e318d06ac0ad121482bde02e1fc03ef27cb69710ca701cb5077 |
| MAC-l7m0 | 20005 | bcc47cfd554e9a4d1ccb25d5765584f69d15406aed8bf41303e1941efb2b8e5a |
| MAC-hpqp | 97237 | a50938351e48f3c3c513765999e2df87c7e3c9adb7286af61991815f7b79574c |
| MAC-hy71 | 6603 | c21ba10377ffbd0f538cb3a21625ce62483ece19bc31fbff2062e998eca7e428 |
| MAC-hgz1 | 39751 | 2a323bafd5f50877286b79ea45243945c109ad9e7dcfcd59933a22dab9c5e82e |
| MAC-lnu6 | 149082 | 9a49a8007a2e86bdf9c55fd01392e7844df56e109c8f70a176ddedb90c35f0ee |
| MAC-vx24 | 27038 | c2a5711dfc651e9a95dfaa8c6e5a9b78d2bc8b97acd5fe9e54ea261fe0fd6179 |
| MAC-ou97 | 19506 | a946be6cc705da9843c8060965ce9a6708114d29c6930ea15b639d5bdd713274 |
| MAC-sh60 | 180187 | 4f57c52a5dc467e65fa448c8ea07537cdb3f025821aa74a430014d1191559362 |

sh60 was reread as protected audit context, not re-accepted. Y's concurrent supplemental RED and metadata repair are outside this approval; no assertion of unchanged global tracker state or Y acceptance is made.

## Scope and remaining execution gates

No source/test/installed/runtime/service/remote mutation, source build, native execution or full scripts/preflight.sh was performed by this review. The preserved Dagger and installed NIL boundaries were not modified. No source dispatch is authorized before the documentation-only accepted producer and root hold are resolved through their existing process. Real hard-TDD assertion RED, independent inventory approval, native execution and all final integration/publication gates remain actual delivery obligations, not conditions attached to this binary backlog approval.

A missing prior-turn in-memory map caused a local JavaScript TypeError before any external call; fresh canonical pvg reads reconstructed it. One oversized combined display was reread in bounded pieces to complete inspection. Neither diagnostic was treated as successful evidence or caused a tracker mutation.

## nd_contract
status: in_progress

### evidence
- REVIEW_RESULT: APPROVED for the exact round-two assurance backlog identities above; zero blocking findings.
- Both round-one generalized rules and their complete nineteen-story/affected-consumer sweep verified, not merely the originally cited instances.
- Scoped lint: zero errors, ten manually reviewed heuristics. Dependency-cycle check: none. Raw prefix/hash and source-ref gates verified.
- This contract records backlog review only; epic remains in progress and no child status, label, claim, dependency or protected history is changed by this review.

### proof
- [x] Real executable surfaces have bounded ownership and required consumers.
- [x] Prospective full-root CRM subject changes require independent substantive current judgment and real stale/current/invalidated controls.
- [x] First-use, retained replay, closed native adapters, complete inventory, final closure and two-platform proof remain explicit delivery requirements.
- [x] Frozen original proof, legacy regressions, existing failed audit and serial ownership remain protected.
- [ ] Canonical document landing, implementation, native proof and final integrated epic preflight remain pending.



### 2026-09-06T10:35:03Z ramirosalas
LOCAL INTEGRATION CHECKPOINT — 2026-09-06

MAC-l7m0 independently accepted documentation-only at candidate 0feaebf725267f9d0045d653ef57a90af0c9d946 and merged locally with --no-ff to epic/MAC-ui8a a94e768adf461178e0562ad135c7c06d8163a3a4. Exact public document SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8 retained after merge; aggregate doc diff whitespace check and candidate ancestry pass. Main remains 497419ab4512fcff765cd5feb27aed4c67b5608d and root checkout stayed on main. Root-owned integration checkout .claude/worktrees/integrate-MAC-ui8a holds epic for subsequent local accepted-gated merges. Clean accepted story checkout removed and merged story branch deleted; its commit remains reachable from epic. No remote operation or preflight/install occurred.

Current accepted/merged count: 7 of 40 children. Independent review /tmp/machinery-assurance-pm.pYsE8w/REVIEW.md SHA256 b7c57fcb51d64b332c1baef0b8d9d2f986968042b58e9dc233de1f9865a1cc8d; completion SHA256 1fbfbaca291222927f6f7d3551b64e0748612140b4b46f863a0a847ea5b460cd. Documentation approval does not establish native custody, adapters, replay or release completeness.

MAC-yig6 candidate ef505052dc73cb9032dd91f1a3f99837f76204a8 contains only exact authorized helper+44 amendment, original246/supplemental147 RED hashes unchanged. Native Darwin paired selected runs each have24 test/subtest PASS plus1package PASS (11top-level tests), original paired runs1test plus1package; no skip/fail. Previous package-as-test miscount corrected append-only. Full report /tmp/machinery-yig6-green.hZtIWT/REPORT.md SHA256 8ecba89eb785887bfe8c4d5f145740bac01f8c04145fef6bd5fc9d80ce15e8c6. Mandatory same-revision native Linux amd64 proof remains absent/unwaived; story retained healthy in_progress, not delivered or accepted.

Portfolio policy V3 approved; bounded21-path lhu5 and existing hgz1 substantive12-row consumer scope repaired append-only, lnu6 unchanged. Independent exact scope/source/method/fixture review remains before any source/RED edits. External proposal writer /root/portfolio_exact_proposal_terra is working without source/tracker mutation. MAC-qlw2 selected by pvg loop next as freshRED, atomically claimed and worktree created at epic a94e768; /root/qlw2_red_inventory_terra is limited to pre-RED exact inventory/method proposal, no source/stub/tests authored before independent review. Minimum compilable contracts/stubs are permitted by developer skill only after that review; compile/setup failures are not RED evidence.

Pending user input: native Linux amd64 host/runner or approved provisioning; compliant MAC-sh60 successor proposal approval. Old sh60 failed audit remains intact, no exception/cancellation/successor/transfer performed. Installed Machinery and Dagger remain protected. Full preflight is still final integrated gate only.

## nd_contract
status: in_progress

### evidence
- Seven accepted local story merges; main/installed/remote state unchanged.
- Current child actors and exact candidates/holds recorded above; no global recovery/setup/sync.

### proof
- [x] Accepted standalone contract has exact local canonical landing and independent positive/negative review.
- [ ] Remaining implementation, actual native Linux proof, independent acceptance and final integrated preflight/local release remain pending.


### 2026-09-06T11:03:02Z ramirosalas
REVIEW CHECKPOINT 2026-09-06 — no additional source acceptance
Accepted/merged remains 7 of 40 children. Main497419ab4512fcff765cd5feb27aed4c67b5608d and epic a94e768adf461178e0562ad135c7c06d8163a3a4 unchanged. Installed Machinery SHA2565205883aaa4276d7eb6edb25b6ad43ac39a04bcb9a8b5ee55498127b04950849 and user Dagger18576903a871d895c8b414ee0a41897313ce31d7b5b6284c29b489553e4fae99 preserved; pushurl remains /dev/null. No preflight, runtime install, remote operation, binary replacement or new repository source/test edit.

Portfolio: initial external whole-package drafts were substantively inadequate (mostly keyword/count checks, missing policy outcomes and incomplete source projection), so not approved for source/RED. Bounded D1 method now frozen at /tmp/MAC-lhu5-D1-method-repair-v3.xQUt1b: test5577ad3334146482c0e1ffdc318ee518eac1b91f8f8c97b80bd6ebe681f1cc10; report6499222e7da0a95706463eff76b516172f1c2ce118f7b372316a7a0f070aaa99; raw9cb13953789c5ce446b2eacf0f5707a4dfb468615c68cf322ecc98501610c199. Root checked48 test/subtestPASS +1packagePASS,0FAIL/SKIP. This is a finite external method experiment, not repository RED or a portfolio runtime. /root/portfolio_d1_method_pm is independently reviewing it. Whole packet/source integration, D2–D6, full FSM target/effect obligations and actual policy source repair still pending. Prior external draft preservation violations and corrected hashes are disclosed in v3 report, not misrepresented as immutable or repository RED.

Custody: /tmp/MAC-qlw2-boundary-adjudication.onIV3H/REVIEW.md9f85645d893ffcdb6888480a85d8030668fe65c382cbb8866faa708f0339221a found missing callable producers for opaque Capability/InternalIO, unexpressible ExtraFiles/SysProcAttr producer tests, incomplete native observation/calibration and contributor-container handoff. Normal Limits/private framing choices are not architecture blockers. Proposed narrow supplement /tmp/MAC-qlw2-supplement-v2.63aaBU/PROPOSAL.md169f81b34f78eb928cf76994105c7361b4e39642f9da50925e4a42026e86525a and NATIVE-OBSERVATION.md5a0b0e5b80920d4fcf2d4344f219b4925778f87467ff8807b41a901d523775bb are under /root/custody_supplement_anchor review. New constructors/auth cap/calibration staging are NOT yet approved/applied. Root flagged root-owner→first-broker bootstrap contradiction for independent review. q remains claimed, source-free preRED hold with corrected authoritative EOF in_progress checkpoint.

Pending user resource/authority unchanged: required native Linux amd64 executor (current Mac/Docker arm64 not substitute); approval of proposed fresh MAC-sh60 compliant successor/disposition. Existing failed sh60 branch/audit remain intact; no exception, cancellation or blocker transfer. No false completion.

## nd_contract
status: in_progress

### evidence
- Current source/protected-resource identities checked read-only.
- Actual D1 draft model execution counts separated from independent acceptance and native implementation proof.
- Two bounded independent reviewers active; no global recovery/setup/sync.

### proof
- [x] Unsupported/tautological preliminary test claims were rejected before source authoring.
- [ ] Remaining implementation, source/method approval, two-platform native proof and final integrated preflight/local release remain pending.


### 2026-09-06T11:42:35Z ramirosalas
LOCAL EVIDENCE CHECKPOINT 2026-09-06: bounded completed external artifacts archived under git common dir at .git/machinery-evidence-20260906.TEFZ7D/bounded-methods-checkpoint.tar.gz, SHA256 8cfaff4bcd0775813b5bebe4080a6c8de34bad9cdcddcb6585feca8ad5ac5c16; adjacent README states contents and limits. Original artifacts retained. Includes approved policy V3, D1 V3/V4 and both failed/successful independent probe history, unapproved D5/D6 candidate, custody V2/V3 rejection history, pending sh60 successor proposal. Listing and two extracted frozen-source hashes verified. Not an archive of all earlier proof. No source/ref/install/remote/preflight change. main497419a and epica94e768 unchanged; 7/40 children accepted. D5/D6 independent method review and custody V4 authoring remain active, not repository RED approval.

### 2026-09-06T16:31:15Z ramirosalas
USER AUTHORIZATION UPDATE 2026-09-06: user confirmed the supplied remote executor is amd64 and authorized Machinery verification there as needed, with no disturbance to existing containers and cleanup only of our resources. Root read-only SSH readiness check confirmed Linux x86_64, Docker29.7.2 linux/amd64, four CPUs; no remote resources or tests created. Host availability is no longer a permission blocker; actual same-candidate native proof remains owed. User explicitly approved the previously proposed compliant fresh successor to MAC-sh60: retain failed history, new independently reviewed RED and fresh GREEN, no exception. Bounded Sr PM bookkeeping assigned; no successor completion or cancellation asserted yet. User requested a narrow handoff for the platform restriction; root instead prepared an official-feedback diagnostic packet and will not route blocked custody execution through an unrestricted agent or the remote host. Platform restriction remains unresolved. Independent pure portfolio finite-method correction continues separately; no product integration/RED/native acceptance from that work. Main and installed binary remain unchanged; preflight held until final integration and no remote Git publication.

### 2026-09-06T16:51:41Z ramirosalas
2026-09-06 16:52 UTC checkpoint: user-authorized native Linux amd64 access verified read-only; no remote files/containers created or existing resources changed. Compliant audit successor MAC-wi5z created, depends on hgz1 and follows sh60, but non-accepting retirement of old sh60 is unresolved; two scoped producer-collision errors retained. External portfolio values/timestamp/backup V2 independently APPROVED, full132leaves16parents and newreviewer7leaves2parents allpass; reportSHA 0e6b9519bb45663757c2d8264d7a99df692bc6cffe9b94ea2bf59af9b014e835. No source/RED/native/story credit. Eight of42 children accepted. Custody task remains platform-halted with unknown exact classifier reason; diagnostic packet .git/machinery-evidence-20260906.TEFZ7D/PLATFORM-REVIEW-REQUEST.md ready but not submitted. Overall incomplete/blocked; no defensible finish ETA. Main497419ab, epic2a73454d, installedSHA5205883aaa4276d7eb6edb25b6ad43ac39a04bcb9a8b5ee55498127b04950849 unchanged and main/integration clean. No preflight, install, push or blocked-task rerouting. Latest append-only evidence and STATUS.md retained in same private directory.
