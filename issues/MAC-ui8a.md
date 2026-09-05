---
id: MAC-ui8a
title: "Deterministic assurance hardening"
status: open
priority: 0
type: epic
created_at: 2026-09-05T19:28:10Z
created_by: ramirosalas
updated_at: 2026-09-05T20:45:02Z
content_hash: "sha256:7196ec769b13a0ece3221c4245e0f6ee635b8943b8da28545ca1876768b87e54"
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
