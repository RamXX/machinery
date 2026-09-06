---
id: MAC-lhu5
title: "Make portfolio milestone packets satisfy their standalone handoff contract"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T03:03:05Z
created_by: ramirosalas
updated_at: 2026-09-06T11:50:40Z
content_hash: "sha256:070bed22fc00039ad0496facee1800a9d754bf629a2d533d8ba8f0162a220aad"
blocks: [MAC-ou97, MAC-hgz1]
assignee: dev-MAC-lhu5
follows: [MAC-2u36]
---

## Description
## USER INTENT
An implementation agent given one portfolio milestone packet must receive the concrete obligations the shipped packet-alone handoff promises, without guessing the optimizer objective, units or required assertions.

## Context (Embedded)
Confirmed packet-alone zero-context defect: root examples/portfolio-engine/design/BUILD.md:12 explicitly applies the claim independently to each packet, and :298 calls each packet complete execution context. M3 omits minimization of historical maximum drawdown and exact10000-bps allocation, referring instead to chosen numeric representation/domain tolerance. Root/model already define integer basis points for Holding.weight and Portfolio.maxDrawdown, exactly16 unique candidates/holdings, nonnegative weights and exact sum10000. This is an obligation omission, not evidence that a nonexistent optimizer ran incorrectly.
All six packets were read. M0 points to domain schemas but omits concrete local forms; M1 describes feed/optimizer/repo ports and explicit timeouts without concrete local signatures/bounds; M2 references threshold/cooldown/stable error envelope without concrete local definitions. M4 requires parsed rows/ordered actions but omits next-state assertion; M5 omits next-state/full expected-action assertion. These are bounded scope for the same packet-alone handoff; M3 is a pure transform and must not acquire an invented FSM.

## Ownership
Seven paths:
- examples/portfolio-engine/design/BUILD/M0-walking-skeleton.md
- examples/portfolio-engine/design/BUILD/M1-run-pipeline.md
- examples/portfolio-engine/design/BUILD/M2-feed-breaker.md
- examples/portfolio-engine/design/BUILD/M3-optimizer.md
- examples/portfolio-engine/design/BUILD/M4-portfolio-review.md
- examples/portfolio-engine/design/BUILD/M5-reference-operations.md
- cmd/machinery/portfolio_packet_contract_test.go (new bounded handoff/semantic contract proof).
Each packet edit must be justified by a source-established missing local obligation, not a wholesale rewrite. Read-only: root BUILD.md, ARCHITECTURE.md, domain.modelith.yaml/md, committed machine/oracle/matrix files, all attestations and acceptance. Root and evidence alignment belongs to the serialized consumer migration; no simultaneous MAC-lnu6 shared-file writes. Packet hash changes make existing attestation covers stale and must be handed off explicitly, never silently rehashed here.
You are not alone; preserve all unrelated content and prior exact frozen-formatting-policy changes. Existing tests/goldens are unchanged; any discovered necessary amendment requires independent PM authorization.

## Boundary Map
PRODUCES:
- Six exact milestone packet files -> self-contained concrete local obligations consistent with existing root/domain/architecture and permitted referenced machine/oracle sources.
- cmd/machinery/portfolio_packet_contract_test.go -> actual packet-input handoff/semantic consistency and removal/contradiction sensitivity, not optimizer implementation or keyword-only proof.
CONSUMES:
- Existing root handoff and domain authority.
  source: BUILD.md:12/298 packet-alone promise; Holding.weight and Portfolio.maxDrawdown integer bps; six optimizer invariants including16 unique holdings and exact10000 sum; domain.modelith.yaml defines min historical maximum drawdown.
- Existing architecture contracts.
  source: ARCHITECTURE.md pf.app -> pf.optimizer optimize(candidates, prices, k=16, lookbackDays) -> Portfolio{holdings,maxDrawdown} | InfeasibleError; other current ports/error maps and committed packet-referenced machine/oracle inputs remain authoritative.
- Existing prospective conformance vocabulary.
  source: committed machine .oracle.md row source/trigger/guard/target/actions plus entry/exit tables; current claim requires parsed next state and expected actions, while an unimplemented packet is only prospective design.

### Story Acceptance Criteria
1. Preserve the packet-alone promise; each affected packet states the concrete local domain value forms, interfaces/errors/bounds and behavior required for its own named milestone, consistently with existing authoritative sources. A referenced committed machine/oracle is allowed as the root explicitly permits, but missing local obligations cannot be replaced by an undefined domain tolerance/chosen representation or undocumented ambient/root context. Do not silently weaken to root-plus-packet to pass Gv.
2. M3 explicitly requires deterministic16-of-N historical maximum-drawdown minimization, exactly16 unique in-candidate holdings, nonnegative integer-bps weights totaling exactly10000, integer-bps stored drawdown and the declared deterministic tie break, plus bounded/invalid/infeasible-input behavior. Preserve pure-transform/no-I/O semantics. Specify a concrete source-supported numeric comparison/representation/rounding policy sufficient to make its property expectations deterministic; if existing sources do not settle a necessary numerical detail, obtain a bounded independent technical decision before authoring that detail rather than inventing a tolerance or algorithm.
3. M4 and M5 explicitly require wholesale parsing and assertions of each committed row's next state and expected actions, reconciled with legitimate entry/exit/internal-transition semantics; all current rows and guard branches remain owed. Preserve M0-M2 conformance/real-boundary obligations. M3 uses semantic properties and boundary cases, never a fabricated lifecycle machine.
4. New real packet-input contract tests plus independent fresh-context handoff review demonstrate concrete valid and invalid scenarios using the packet and its explicitly allowed inputs, with root available only to the independent consistency oracle, not as hidden execution context. Include two differing drawdown candidates to distinguish minimization from arbitrary selection, valid10000 and invalid9999/10001 sums, integer/fractional representation boundaries, tie/permutation and infeasible cases, and next-state/action obligations for M4/M5. Deleting or contradicting objective/units/sum/local port/error information must fail the intended handoff/semantic assertion. Mere keyword presence or an LLM's unsupported judgment is not sufficient executable proof.
5. Actual isolated Machinery design checks and unchanged applicable golden compatibility account for the delivered packet changes; expected stale attestation diagnostics are explicitly attributed for the downstream migration, not waived as fresh. Report exact old/new packet hashes and full changed obligation inventory to that owner. No current implementation, actual optimizer execution, formal proof, service run or new attestation is claimed from design-only checks.

## Testing Requirements
Hard TDD: new tests first with a matched well-specified packet control and actual old deficient packet failure; independently review the exact semantic test method and any unresolved technical decision before RED. All current source/golden tests remain frozen.
Unit plus Integration tests: MANDATORY (no mocks for file/process/gate outcomes). Actual packet files, temporary deficient/contradictory packet variants, real built standalone CLI and deterministic Go test calculations/contract inspection. Tests must explain how packet-derived facts determine observable scenario expectations; parsing a numeric literal alone does not prove the optimizer objective. This is plan/handoff proof, not a fabricated optimizer implementation.
Initial native selector: go test -count=1 -timeout=120s ./cmd/machinery -run 'PortfolioPacket' -json. Name all new leaves/controls and report raw outcomes, exact input hashes and host. Final real check <portfolio-design> --gate gv and other actually affected design gates use isolated configuration and accurately classify stale old evidence until migration. No Docker/Python runtime requirement is invented for this documentation-contract repair; actual future portfolio implementation/runtime belongs to future consumer work, not this story.

## OUT OF SCOPE
- Portfolio optimizer/application implementation, new financial objective or weaker root-plus-packet contract.
- Core attestation schema/freshness (MAC-p7jd), root BUILD/evidence migration and MAC-lnu6 policy amendments.
- Generated oracle changes, acceptance renewal, external services or full preflight.
- Unspecified technical numeric/port decisions are a named review checkpoint, not permission for arbitrary feature design.

## DIFF BUDGET
- Seven files, forecast350-650 changed LOC; six narrow packet repairs plus semantic/handoff regression may dominate. Report actual justified packet hunks and test cost, no trimming proof to fit.
- Source baseline packets total approximately250 lines; substantial growth needs explicit rationale. Native tests initial120s bound; actual cost measured and recorded.

## Discovered During
MAC-lnu6 independent substantive amendment review at 713184db16a12b8c3763b4aa8f5bf025721abf22. Full report /tmp/MAC-lnu6-PM-AMENDMENT-713184d.md SHA256 6fc5223d3e5d48170f89ff9e29b634b256bfe2454b710378f08cb60d875aadf1 was read completely (227 newline-terminated lines; final content included). No runtime mutation proof beyond the explicitly reported native replay is claimed. Exact epic 70652b948 source inspected; graph generation 2026-09-06T02:42:16Z is best effort, not completeness proof.

## MANDATORY SKILLS
- developer; codebase-memory; pm_acceptor.

## Delivery Requirements
Hard TDD with independent RED and exact existing-test amendment review before edits. Preserve all unrelated tests, goldens, generated evidence and existing claims. Shared tracker is development coordination only; product remains standalone. No remote, full preflight, installed assets, user services or healthy-worktree cleanup. No source/docs/tests changed during triage. Use supported delivery; independent PM accepts.

## nd_contract
status: new

### evidence
- Confirmed source-established packet-alone handoff omissions; all six packets/root/model/architecture inspected. No optimizer runtime or mutation was executed in triage.
- Existing integer-bps representation supplies concrete authority; any still-unstated numerical precision/comparison or port detail requires bounded technical review.
- Packet-only ownership avoids concurrent writes to MAC-lnu6's root BUILD/evidence; consumer sequencing remains explicit downstream.

### proof
- [ ] AC #1: complete packet-alone local contract with source consistency.
- [ ] AC #2: concrete optimizer obligation/units/sum and reviewed numeric semantics.
- [ ] AC #3: complete prospective FSM obligations without invented M3 machine.
- [ ] AC #4: real positive/negative semantic handoff proof.
- [ ] AC #5: honest design checks and exact packet hash handoff.


## Acceptance Criteria


## Design


## Notes
## Pre-RED method checkpoint (not delivery)

External proposal: /tmp/MAC-lhu5-method.A1so4U/PROPOSAL.md; 299 lines; SHA256 1dd258d9b228e68e5b6bf8763653a591469c07264d56852a5d9b187f89668886. Prepared on clean story/MAC-lhu5 at 70652b948bf090008b1965c85daf36ea374daea4, Darwin 25.6.0 arm64. Read all six packets, root BUILD, domain YAML/rendered MD, architecture and all four committed machine/oracle/matrix triplets. Graph generation 2026-09-06T02:42:16Z used for bounded orientation; exact source fallback and best-effort limitations recorded.

Source-established gaps: missing local schemas/ports/errors/bounds, M3 min-drawdown/integer-bps/exact10000 obligation, M4 next-state and M5 next-state/full-action obligations. M0 repeat-command idempotency contradicts architecture177/191 and Run.machine11 new-run rule. Review decisions required for price/value representation and exact comparison, stored rounding, complete ticker/weight tie rule, admission/limits/error policy, cross-source port errors (BusyError/TerminalError) and necessary M5 value/operational bounds. No numerical policy or adapter mapping silently selected.

Method proposes actual packet-only allowed outcome/witness evaluation, independently sourced expected outcomes, matched complete/original/mutant file inputs, full parsed committed oracle plus entry/exit obligations and real source-built isolated CLI gates. It explicitly does not implement or claim a future optimizer/runtime. Exact decision and method approval remain required before any RED writes. Seven-path forecast640-945 changed lines versus original350-650 is justified in proposal; no cost measured and no proof omitted to fit. Initial future selector PortfolioPacket remains120s. All old packet hashes and stale gt.conformance-test-shape/g4.zero-context cover ownership handed off for later MAC-hgz1, after accepted p7+uzxr+lhu5 and before lnu6.

## nd_contract
status: in_progress

### evidence
- git status --short empty; branch story/MAC-lhu5; HEAD70652b948bf090008b1965c85daf36ea374daea4. External proposal SHA256 verified as above.
- No shared tests/packets/source/goldens/evidence edited; no build/tests/service/formal/design gate run, RED commit, delivery, release, approval or acceptance. Installed assets and remote state untouched.
- STOP at story-required independent method/technical review checkpoint; root routes the reviewer. Five AC remain pending.

### proof
- [ ] AC #1: Complete source-consistent packet-alone contracts pending.
- [ ] AC #2: Optimizer numerical decisions, obligations and properties pending independent review.
- [ ] AC #3: Full next-state/ordered entry-transition-exit prospective conformance pending.
- [ ] AC #4: Approved real packet-input positive/negative semantic proof and fresh-context review pending.
- [ ] AC #5: Actual isolated checks, unchanged golden accounting and old/new hash handoff pending.


## nd_contract
status: new

### evidence
- Created P0 under MAC-ui8a; open/unclaimed/hard-tdd. MAC-ou97 explicitly depends on this repair. No existing state/claim/test/source changes.
- Actual canonical readback confirms five AC, explicit nonoverlapping ownership and intact quoted command. Scoped lint PASS36issues/0errors/0review; dependency cycles none; RTM0extracted requirements/21stories/4closed is structural only, not AC proof.
- Independent test/handoff review and exact existing-test amendment authorization remain pending; no RED approval or implementation permission granted by triage.
- Confirmed source-established packet-alone handoff omissions; all six packets/root/model/architecture inspected. No optimizer runtime or mutation was executed in triage.
- Existing integer-bps representation supplies concrete authority; any still-unstated numerical precision/comparison or port detail requires bounded technical review.
- Packet-only ownership avoids concurrent writes to MAC-lnu6's root BUILD/evidence; consumer sequencing remains explicit downstream.

### proof
- [ ] AC #1: complete packet-alone local contract with source consistency.
- [ ] AC #2: concrete optimizer obligation/units/sum and reviewed numeric semantics.
- [ ] AC #3: complete prospective FSM obligations without invented M3 machine.
- [ ] AC #4: real positive/negative semantic handoff proof.
- [ ] AC #5: honest design checks and exact packet hash handoff.


## History
- 2026-09-06T03:03:20Z dep_added: blocks MAC-ou97
- 2026-09-06T03:17:08Z dep_added: blocks MAC-hgz1
- 2026-09-06T05:48:13Z status: open -> in_progress
- 2026-09-06T05:48:14Z auto-follows: linked to predecessor MAC-2u36
- 2026-09-06T05:48:14Z claimed by dev-MAC-lhu5

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-ou97]], [[MAC-hgz1]]
- Follows: [[MAC-2u36]]

## Comments

### 2026-09-06T03:05:28Z ramirosalas
## nd_contract
status: new

### evidence
- Created P0 under MAC-ui8a; open/unclaimed/hard-tdd. MAC-ou97 explicitly depends on this repair. No existing state/claim/test/source changes.
- Actual canonical readback confirms five AC, explicit nonoverlapping ownership and intact quoted command. Scoped lint PASS36issues/0errors/0review; dependency cycles none; RTM0extracted requirements/21stories/4closed is structural only, not AC proof.
- Independent test/handoff review and exact existing-test amendment authorization remain pending; no RED approval or implementation permission granted by triage.
- Confirmed source-established packet-alone handoff omissions; all six packets/root/model/architecture inspected. No optimizer runtime or mutation was executed in triage.
- Existing integer-bps representation supplies concrete authority; any still-unstated numerical precision/comparison or port detail requires bounded technical review.
- Packet-only ownership avoids concurrent writes to MAC-lnu6's root BUILD/evidence; consumer sequencing remains explicit downstream.

### proof
- [ ] AC #1: complete packet-alone local contract with source consistency.
- [ ] AC #2: concrete optimizer obligation/units/sum and reviewed numeric semantics.
- [ ] AC #3: complete prospective FSM obligations without invented M3 machine.
- [ ] AC #4: real positive/negative semantic handoff proof.
- [ ] AC #5: honest design checks and exact packet hash handoff.


### 2026-09-06T05:57:12Z ramirosalas
## nd_contract
status: in_progress

### evidence
- Pre-RED checkpoint only: /tmp/MAC-lhu5-method.A1so4U/PROPOSAL.md,299lines,SHA2561dd258d9b228e68e5b6bf8763653a591469c07264d56852a5d9b187f89668886. Exact source-backed gap inventory and proposed semantic witnesses await independent method/technical review.
- No tests,packets,source,goldens,evidence,installed assets or remote changed. Clean story/MAC-lhu5 HEAD70652b948bf090008b1965c85daf36ea374daea4. No tests/builds or gates run; no RED/delivery/acceptance claim.
- Append-notes readback placed new checkpoint before an older status:new Notes block. This final append-only comment makes authoritative latest contract in_progress without canonical body repair. Root routes independent review; developer stops and retains claim.

### proof
- [ ] AC #1: Complete source-consistent local packet contracts pending.
- [ ] AC #2: Numerical decisions and optimizer obligation proof pending.
- [ ] AC #3: Full prospective FSM next-state/action obligations pending.
- [ ] AC #4: Approved packet-input controls and independent handoff review pending.
- [ ] AC #5: Real isolated checks,golden accounting and old/new hash handoff pending.


### 2026-09-06T10:24:20Z ramirosalas
## AUTHORITATIVE BOUNDED PORTFOLIO SCOPE / AC RECONCILIATION — 2026-09-06

This true-EOF amendment is authoritative for MAC-lhu5's prospective ownership, five acceptance criteria, testing checkpoint and forecast wherever the preserved historical Body says seven paths, read-only root/architecture/model/matrices, packet-only source authority, or an earlier estimate. All historical text and proof remain intact as history. This is an explicitly root-authorized tracker scope repair after the user's “Define reviewed illustrative policies” choice and independent ROUND 3 architecture/policy approval; it is not a status transition, source edit permission, RED approval, evidence renewal or acceptance.

Policy authority: immutable /tmp/MAC-lhu5-policy-v3.8zFT4P/PROPOSAL.md SHA256 dbaf11814dff1d219114fee64a051d759ba29fa4a898b0a31f1e511091356073, 78,695 bytes, 304 newline-terminated lines, final LF; independent /tmp/MAC-lhu5-policy-v3.8zFT4P/CHALLENGE-3.md SHA256 29aa5701bf6868fc664644d0d98e512ec60522f1c7aec92f9cdddbd72054acdc, REVIEW_RESULT: APPROVED, zero blocking. The old 305-line metadata is incorrect. Architecture/policy approval does not grant exact source-hunk/method/fixture/RED/generated/evidence authority. Historical method /tmp/MAC-lhu5-method.A1so4U/PROPOSAL.md SHA256 1dd258d9b228e68e5b6bf8763653a591469c07264d56852a5d9b187f89668886 remains preserved, not automatically approved by this repair. No new user domain choice is needed; only a concrete newly found contradiction warrants escalation.

## USER INTENT
An agent given any one portfolio milestone packet receives the complete local illustrative domain, interface, failure and assertion obligations promised by the packet-alone handoff. The source-authority consistency work below is necessary because reviewed missing policies cannot be truthfully supplied only in packets while canonical contracts say something else. Native portfolio implementation is future consumer work.

## Exact prospective ownership
Exactly 21 paths below, no blanket directory grant. The first 20 exist in Git and the final Go test is new. Every source write remains HELD pending independent exact scope/source-hunk and PRE-RED method review. domain.modelith.md is tool-generated only after review; the three machine JSON files are metadata-only as scoped. No oracle/formal/golden/evidence/acceptance bytes may change here. If actual generated freshness requires another path, stop for exact scope/review before editing it.

PRODUCES:
- examples/portfolio-engine/design/ARCHITECTURE.md -> bounded illustrative handoff artifact.
  schema: Canonical numeric/value/limits contracts; explicit RepoSession/operation API and per-command actor binding; admission-vs-return bounds; resolved model envelope, unknown-publication residual, complete error/outcome filters; reconcile sections7–11 and no-callback/reverse-import rule.
- examples/portfolio-engine/design/domain.modelith.yaml -> bounded illustrative handoff artifact.
  schema: Existing approved value/identity/lookback/rounding/timestamp clarifications; add model-applicability prose distinguishing modeled lifecycle transitions from unresolved operational publication. Preserve entity/invariant IDs, enums,16 records and integer types; no new unknown domain state.
- examples/portfolio-engine/design/domain.modelith.md -> bounded illustrative handoff artifact.
  schema: Authorized TOOL-GENERATED mirror only; no hand edits.
- examples/portfolio-engine/design/BUILD.md -> bounded illustrative handoff artifact.
  schema: Match dictionary/interfaces/CLI/persistence barrier; replace unconditional error->onError and runtime-termination implications in sections4,7,10,12 with outcome-filter/abstract-proof applicability; preserve all lnu6-reserved blocks and add necessary caveats outside them.
- examples/portfolio-engine/design/surfaces.yaml -> bounded illustrative handoff artifact.
  schema: Same four existing command surface strings gain the previously proposed relevant limit arguments; no invented Backup/Restore domain act.
- examples/portfolio-engine/design/machines/RecommendationRun.matrix.md -> bounded illustrative handoff artifact.
  schema: Pre-run LoadCandidateSet admission/error path; immutable snapshot/limits actor binding; accepted-only feed/pure results; separate durable terminal-publication barrier and resolved-envelope assumption. Guard/action contracts and8 oracle rows retained.
- examples/portfolio-engine/design/machines/ReferenceDataCommand.matrix.md -> bounded illustrative handoff artifact.
  schema: Full ranked-row/normalization/limits contract and bound handle passage through reference adapters; recordReferenceError requires confirmed nonpublication; cancellation/timeout actions require denied publication and completed drain. Unresolved has no failed-row delivery. Remove universal bounded-success/failure/20s-return wording;20s is admission deadline.
- examples/portfolio-engine/design/machines/Portfolio.matrix.md -> bounded illustrative handoff artifact.
  schema: REQUIRED NEW OWNERSHIP. persistDecision bound Save/handle/outcome API; isRetriable receives only confirmed-Unpublished errors. Failure catalog conflict/exhaustion/nonretriable/timeout rows state causal no-publication, not global unchanged storage; unknown publication is separate no-transition/no-auto-retry residual.5s is admission deadline. Existing20 rows/guards remain, not blindly every IOError->reverted.
- examples/portfolio-engine/design/machines/MarketDataFeed.matrix.md -> bounded illustrative handoff artifact.
  schema: REQUIRED NEW OWNERSHIP. Qualify “never a hang” and “run retries then fails” as bounded counter/scheduling behavior for resolved accepted provider outcomes; no total native return guarantee. State stale-callback rejection. Keep threshold/probe/reset predicates and6 rows.
- examples/portfolio-engine/design/machines/RecommendationRun.machine.json -> bounded illustrative handoff artifact.
  schema: METADATA ONLY: _comment/_counters descriptions qualify “drives to terminal” by resolved operational/persistence envelope; _delays fetch/optimize descriptions name result-admission deadlines. Keep context,invoke.input,states,guards,actions,transitions and numerical timing values exact.
- examples/portfolio-engine/design/machines/Portfolio.machine.json -> bounded illustrative handoff artifact.
  schema: METADATA ONLY: _comment qualifies conflict/retry/rollback as resolved-Unpublished outcomes and points to driver envelope; _delays.COMMIT_TIMEOUT says5000ms publication-admission deadline, admitted commits/drain may finish later. No context/graph/action change.
- examples/portfolio-engine/design/machines/ReferenceDataCommand.machine.json -> bounded illustrative handoff artifact.
  schema: METADATA ONLY: _comment states resolved outcome envelope excludes unresolved publication; _delays.COMMAND_TIMEOUT says20000ms publication-admission deadline, not total return bound. No context/graph/action change.
- examples/portfolio-engine/design/STATE.md -> bounded illustrative handoff artifact.
  schema: Preserve dated gate/proof rows as historical; label “All gates green” historical to those reviewed bytes, and add current policy applicability/verification-pending note. No fabricated rerun, date refresh or acceptance claim.
- examples/portfolio-engine/design/DECISIONS.md -> bounded illustrative handoff artifact.
  schema: Preserve historical interrogation/maintenance answers; append explicit accepted-policy provenance WHEN actually approved and abstract-termination-vs-runtime applicability, including unresolved operations and expanded interfaces. Do not retroactively rewrite original answers as if they included this policy.
- examples/portfolio-engine/design/BUILD/M0-walking-skeleton.md -> bounded illustrative handoff artifact.
  schema: Local numeric/ports/limits/operation handles, new-run rule and atomic CommitRecommendation publication barrier; separate pure row observations from actual durable result.
- examples/portfolio-engine/design/BUILD/M1-run-pipeline.md -> bounded illustrative handoff artifact.
  schema: Admission errors before run, immutable actor wiring, separate persistence outcome, resolved trace applicability and honest bound categories.
- examples/portfolio-engine/design/BUILD/M2-feed-breaker.md -> bounded illustrative handoff artifact.
  schema: Existing6 rows plus accepted provider outcomes/stale callbacks, call-count obligations, cooldown/counter/admission-vs-return distinction.
- examples/portfolio-engine/design/BUILD/M3-optimizer.md -> bounded illustrative handoff artifact.
  schema: Preserve exact scoring/rounding/ties/16zero-allowed weights/admission semantics; pure-only result gate, no repository/operation handle inside optimizer.
- examples/portfolio-engine/design/BUILD/M4-portfolio-review.md -> bounded illustrative handoff artifact.
  schema: Bound Save operation/outcome filter; resolved nonpublication rollback/retry versus unresolved no-transition report; complete20 row/entry/exit obligations and5s admission semantics.
- examples/portfolio-engine/design/BUILD/M5-reference-operations.md -> bounded illustrative handoff artifact.
  schema: Closed ranks and reference adapter handles; explicit Backup/Restore operation passage;20s reference versus supplied backup admission deadline; denied/drained failures versus published/unknown outcomes.
- cmd/machinery/portfolio_packet_contract_test.go -> bounded illustrative handoff artifact.
  schema: NEW bounded prose/semantic/observation test file only, after exact method/RED review; no actual optimizer or fake native execution.

CONSUMES:
- Existing examples/portfolio-engine/design/ARCHITECTURE.md and the approved V3 policy copied below.
  source: Existing pf.app -> pf.domain/optimizer/feed/repo and pf.repo -> model/store topology; V3 D1-D6 supplies prospective API/value contracts verbatim. These proposed signatures are not claimed as existing native implementations.
- Existing examples/portfolio-engine/design/domain.modelith.yaml.
  schema: Retain entity/invariant IDs, enum states, integer weight/stored-drawdown values, 16 distinct Holding records and exact 10000 allocation; operational unknowns add no persisted lifecycle state.
- Existing examples/portfolio-engine/design/machines/RecommendationRun.machine.json and RecommendationRun.oracle.md.
  source: Unchanged context/invoke.input/graph and 8 committed oracle rows; full entry/exit and transition columns remain test input.
- Existing examples/portfolio-engine/design/machines/MarketDataFeed.machine.json and MarketDataFeed.oracle.md.
  source: Unchanged machine graph, threshold/probe/reset predicates and all 6 committed rows; no MarketDataFeed.machine.json write ownership.
- Existing examples/portfolio-engine/design/machines/Portfolio.machine.json and Portfolio.oracle.md.
  source: Unchanged graph, all 20 rows, ordered guards, retry/rollback route and full action lists; metadata scope only for the machine file.
- Existing examples/portfolio-engine/design/machines/ReferenceDataCommand.machine.json and ReferenceDataCommand.oracle.md.
  source: Unchanged graph and all 12 rows with full target/action/entry/exit obligations; metadata scope only for the machine file.
- Existing cmd/machinery/golden_test.go.
  spec: goldenBin(t *testing.T) string; runBin(t *testing.T, args ...string) (string, string, int); repoRootDir(t *testing.T) string. Read-only actual source-built CLI and private configuration harness; no existing helper/test edits.
- Existing reserved frozen-guidance blocks from /tmp/MAC-lnu6-proposal.NZsERt/guidance-deltas.json.
  source: SHA256 d21d80e66e18356110be1a4c1753adebf6a3440846c8286b3664da945deac011 identifies nine exact old/new blocks; preserve original old blocks byte-exact and uniquely located, with portfolio root changes outside them. MAC-lnu6 alone applies those replacements after accepted migration and fresh review.

## Superseding five Story Acceptance Criteria
1. Preserve the packet-alone promise and supply complete local milestone contracts from approved V3 D1-D6, copied below. Reconcile all 20 owned existing source paths without changing unrelated content, topology/security posture, entity/invariant IDs, numerical types or machine graphs. Update only the four existing recommend/refresh/upsert/build surface strings with their required limits; invent no Backup/Restore domain act. Root consistency edits remain outside every lnu6-reserved block. Append actual dated approved-policy/applicability provenance to DECISIONS; preserve earlier answers. STATE's prior all-green and proof rows remain historical, with current verification pending. An admission-only timeout is a MATERIAL contract change requiring full consistency review, not comment cleanup.
2. M3 remains pure/no-I/O with exact rational decimal-string buy-and-hold maximum drawdown, comparison before storage rounding, exactly 16 distinct zero-allowed holdings, integer nonnegative weights summing 10000, complete ticker-then-weight-vector tie order, canonical identities and whole-input admission with closed reason precedence. Required OptimizerLimits and ReferenceLimits are immutable explicit inputs with no default/global lookup; rank scalar bounds precede parsing/filtering. Prices, pinned common calendar, lookback observation count, decimal grammar, canonical ticker grammar, timestamps and backup/schema/receipt rules match all literal V3 contracts and witnesses. Future optimizer owes the stated global objective; finite Go proposal comparisons prove no global optimization feasibility/runtime guarantee.
3. Explicit RepoSession-owned OperationId/PublicationRequest/opaque OperationHandle and independently captured expectedId bind exact payload/session/command/attempt/target/version before mutation. Published, confirmed Unpublished and drained Unresolved remain distinct; admission is not publication, and only confirmed nonpublication permits failed-row delivery, rollback/retry. Preserve the reference adapter's original deadline through preparation. LoadCandidateSet is bounded pre-run repository admission: on its typed errors create no run, call no feed, spend zero feed retries. Freeze successful snapshot/limits into actor closures with unchanged machine input keys and accepted-only CollectedInput. CommitRecommendation atomically publishes run+portfolio+16 holdings for Ready, or run+null portfolio+empty holdings for Failed; neither durable Ready NOR durable Failed may be claimed before confirmed publication. App read/pure-result admission is separate from repo write admission. No implicit cancellation, mutable map handle, ambient current operation, reverse repo->app edge, fake timeout rollback, unbounded automatic retry or native/crash-exactly-once/refinement claim.
4. New real packet-input tests and independent fresh-context review demonstrate the full finite prose/semantic method copied below. Full committed inventories: Run 8, Feed 6, Portfolio 20, Reference 12; M0's Run subset 3; M3 no FSM. Parse IDs/full row columns/entry-exit lists and assert immediate targets plus exact ordered exit-transition-entry effects, empty/omitted/extra/reordered actions, ordered guarded selection, separate always microsteps, external self and explicit internal/targetless controls, observable Run terminal entry, refusal/final/guard-false/corrupt-prior negatives. Require the complete prose-only positive control, actual original M3 with two concrete admissible completions and independent literal expected ledger, plus whole-clause outcome-changing mutants reaching valid target scenarios. Dedicated negatives reject durable Ready AND durable Failed before confirmed publication, admitted-versus-published confusion, unresolved-versus-rollback, and result-admission-versus-total-return confusion. The Go test evaluates bounded observations/proposals; it does not simulate native execution and call it conformance.
5. After independent exact review and authorized work, actual isolated source-built Machinery checks and unchanged applicable golden accounting report each real outcome, including stale Gv subjects and any generated freshness implications. Preserve all historical proof and old bytes; no auto-rehash/renewal or fabricated current all-green/formal/runtime claim. Hand MAC-hgz1 every actual old/new path/hash, complete changed obligation/applicability inventory, exact isolated diagnostics, independent review evidence and measured cost. hgz1 performs separate substantive changed-subject/complete-cover review before any conditional evidence write, then hands its accepted baseline to MAC-lnu6; this story completes neither consumer's AC.

## Mandatory independent PRE-RED checkpoint
Before ANY RED file or source write, present the COMPLETE exact extraction table (supported whole clauses, byte spans, typed facts, recognized contradictions and explicitly unrecognized text), independently authored literal expectation ledger with source spans and policy provenance, fully expanded fixture bytes, exact proposed test/source hunks, every native test leaf and matched positive/negative control, expected causal failures, and estimated time/cost. Independent reviewer must approve that exact package; policy approval and this ownership note do not substitute. Existing frozen tests/goldens remain exact; any necessary existing-test amendment needs its own exact BEFORE-EDIT authority.

The complete prose-only M3 control must reach and select B in the supplied objective witness without JSON. Actual deficient M3 must permit both min and max completions satisfying ALL its extracted original constraints; generic UNKNOWN, keyword absence, parser failure or an unsupported clause alone is not semantic evidence. Redundant equivalent clauses and independent-root-mutation controls remain required. Only an actual intended semantic assertion failure counts as RED: compile/setup/infrastructure/timeouts/skips/panics/missing fixtures or future flags never count.

## Testing Requirements
Hard TDD retained, no RED approval granted. Unit plus Integration tests: MANDATORY (no mocks for filesystem/process/gate outcomes). Use real packets, exact temporary deficient/contradictory variants, current committed allowed machine/oracle/matrix files, deterministic Go calculations and the real source-built standalone CLI with isolated configuration. Root/domain/architecture supply the independent expected ledger but are not hidden evaluator execution context. Independent expected literals must not be computed through the evaluator's own scoring/comparison function.
Initial future command: go test -count=1 -timeout=120s ./cmd/machinery -run 'PortfolioPacket' -json.
Future actual built CLI check <portfolio-design> --gate gv and other actually affected gates must classify intended stale evidence honestly; additional commands require change-based justification. Record actual host, source/binary/input hashes, all named leaves and raw results/durations. No Docker/Python/native portfolio runtime is introduced for this bounded documentation-contract proof.

## OUT OF SCOPE
- Native optimizer/application/repository/provider implementation and runtime cancellation/atomicity/liveness proof: future consumer implementation, never credited here.
- Oracle/formal/golden/attestation/acceptance changes: no such source custody; MAC-hgz1 owns its separately reviewed conditional evidence/golden migration. Additional generated freshness changes require exact reviewed scope before edits.
- MAC-lnu6's nine frozen-guidance replacements and its CLI/frozen tests: retained downstream after accepted hgz1, no concurrent shared-file writer.
- Core schema/future authentication/replay redesign or any other story scope/status/claim/dependency: existing owners retain them. Machinery remains standalone; pvg/nd are private coordination only.
- Installed binary/skills/plugins/agents, NIL assets, Dagger/services, remotes and full preflight: protected; final preflight remains the epic completion gate.

## DIFF BUDGET
Exactly 21 prospective paths; forecast 1290–2075 changed LOC supersedes seven-path and earlier revision estimates. This is unmeasured investigation guidance, not a cap or permission to trim proof. Report actual per-path additions/deletions, source/test purpose, isolated check runtime and substantive review cost; investigate unsupported growth. Additional generated paths stay held for exact review. Initial selector limit remains 120s, never a native portfolio command completion promise.

## Embedded approved policy / method requirements
The following sections are copied verbatim from immutable V3 for self-contained technical context. Their historical words “propose”/“needs approval” describe the proposal stage; the round-3 approval resolves architecture/policy only. The independent exact source/method/fixture/RED/generated/evidence checkpoints above remain authoritative. No policy names, fields, signatures, formulas or literals have been renamed.
## D1: observations, values and ranking

Proposed fixed policy:

- `lookbackDays` means the number of observed daily closing-price dates, INCLUDING the initial valuation date; it is an integer >=2. The legacy field name remains, but calendar elapsed days are not its meaning. This clarification must appear in the canonical attribute description and all consuming packets. It avoids inventing a trading-calendar service. Gaps between observation dates are permitted; this example makes no claim that the input calendar contains every exchange session.
- `PriceMatrix` is a canonical value with `dates: [Date]`, `priceBasis: "unadjusted-close"`, and `rows: ticker -> [DecimalPrice]`. A Date is an actual Gregorian date serialized `YYYY-MM-DD`, years 0001 through 9999. Dates must be strictly increasing and distinct. There are exactly `lookbackDays` dates; each candidate has exactly one value for every declared date. The last date is the as-of date for these immutable inputs; no “today” is read inside the optimizer.
- A successful matrix has exactly the candidate ticker keys and complete rows. Alignment is positional against the single explicit dates vector, never an inner join, forward fill, interpolation or truncation. The pinned provider/cache RESPONSE ITSELF must declare one common ordered vector of exactly the requested lookbackDays dates and its priceBasis. The feed verifies that declaration and copies it unchanged; it does not derive a calendar from individual candidates. Every returned history must align with that vector exactly. A provider surface lacking that explicit common-calendar contract is unsupported for this example and yields FeedError(INVALID_RESPONSE); independent candidate calendars must not be intersected, unioned, shifted or filled. “Most recent” describes the provider's pinned response selection, not a feed algorithm or ambient clock. Tests pin the entire response including dates, basis and rows. This is input-calendar completeness, not independently verified provider-calendar completeness.
- For deficient-input tests and failure diagnosis, a cell can decode as the explicit MissingPrice sentinel (JSON null) in an otherwise correctly sized row. It is never a valid price. A missing candidate row or a null cell is INCOMPLETE_HISTORY; a present row with length other than lookbackDays is INVALID_INPUT. An empty string, malformed decimal or nonpositive price is INVALID_INPUT, not “missing.” Extra/duplicate keys and malformed containers are shape errors. These distinctions are part of the admitted input decoder; no sentinel is silently inserted into a short row.
- Prices are strictly positive, finite decimal values. The exact interchange spelling is a string matching `(0|[1-9][0-9]*)(\.[0-9]+)?`; zero, signs, whitespace, exponent notation, NaN/Infinity and binary floating-point objects are not admissible. The decimal's rational value is authoritative: `100` and `100.00` are equal prices. Total decimal digits, including zeros but excluding the point, are bounded by the explicit limit below before arithmetic. This string form prevents a JSON decoder's binary-float choice from defining financial meaning.
- Only unadjusted closing prices are accepted. Corporate-action adjustments, dividends, reinvestment, fees, cash flows and currency conversion are not applied. The feed rejects a differently tagged series; it must not relabel or silently adjust it. This is deliberately a simple illustrative policy, not a recommendation for real portfolio analysis.
- Holdings are initial allocations. For observation t, `V[t] = Σ_i (weight_i / 10000) × (P[i,t] / P[i,0])`; quantities are held fixed across the observation window. Thus V[0]=1. This is a normalized buy-and-hold calculation, with no periodic rebalancing.
- `peak[t] = max(V[0],...,V[t])`; exact drawdown `D = max_t ((peak[t] - V[t]) / peak[t])`, including t=0. Compare D as an exact rational, before any rounding. Equivalent exact decimal or integer-ratio implementations are acceptable if they produce this ordering without tolerance; the contract does not select a solver or library. A numeric backend unable to preserve the ordering must fail explicitly, never guess a tie.
- The future optimizer owes a global minimum over the feasible set described in D3, then its deterministic tie order. The Go handoff test compares only supplied proposals and NEVER searches this space or claims global optimality. The 60s application timeout can fail a difficult search; these rules do not establish that all admitted problems can be solved in that time.

Alternatives rejected for this example: calendar-day lookbacks require an extra calendar/boundary rule; adjusted prices require an adjustment definition/provider agreement; periodic rebalancing changes the objective; rounded-score ranking turns distinct objectives into artificial ties. These are valid alternative products, not equivalent implementations of this policy.

## D2: integer storage

For the selected exact D, store `maxDrawdown = floor(10000 × D + 1/2)` as an integer number of basis points (nearest, ties upward). Do not round intermediate V, peaks, D, or scores used to select the winner. With strictly positive prices and fully allocated nonnegative weights, exact D is in [0,1); serialized maxDrawdown can be 10000 after rounding. Weight and stored-drawdown values are mathematical integer domain values, excluding booleans and float-typed objects; decimal-price spelling is a different type.

Rationale: nearest rounding limits display/storage distortion symmetrically; ties-up has one explicit boundary rule. Ceiling would systematically move every nonintegral value upward; flooring would move it downward. The choice is illustrative, not sourced historical behavior. No persisted float field or sub-bps schema expansion is introduced.

## D3: identities, feasible portfolios and total ties

- At reference-data ingress, remove only leading/trailing ASCII space, tab, carriage return and line feed; uppercase ASCII a–z. The resulting ticker must match `[A-Z0-9][A-Z0-9.-]*`. Reject an empty result, internal whitespace, other punctuation or non-ASCII character. Preserve dots, hyphens and exchange suffix text; do not infer aliases. Canonical identity is exact byte equality of the resulting ASCII string.
- The shared TRANSFORMATION is owned by pf.domain: `normalize_ticker(raw: string, maxScalarBytes: uint64) -> NormalizationResult`, a pure result value containing either a CanonicalTicker or the reason INVALID_SYNTAX/LIMIT_EXCEEDED. It checks the raw byte bound before trimming. pf.app calls this allowed app->domain API at reference ingress, maps an invalid result to ValidationError, and passes canonical identities to repo upsert/candidate construction. pf.feed returns bounded raw reference rows; it does not normalize them. pf.repo receives canonical keys and enforces its unique constraint; it does not call pf.domain or rename keys. pf.model defines only CanonicalTicker/NormalizationResult and other types, with no operational normalization helper.
- The optimizer receives already canonical tickers; it performs a local canonical-grammar admission check and rejects a noncanonical ticker or duplicate identity instead of transforming its universe. That check is not a second normalization implementation. Names and sectors do not participate in identity or ordering. No new architecture dependency edge is needed: only pf.app calls the shared pf.domain transformation.
- A feasible portfolio has exactly 16 Holding RECORDS referencing 16 distinct candidate identities. Every weight is an integer >=0; the exact sum is 10000. A zero-weight record counts, remains in output, and is neither deleted nor replaced. There is no positive minimum, diversification, return or sector rule.
- Sort a portfolio's 16 records by ascending ASCII ticker bytes. Its tie key is `(tickerVector, weightVector)` with lexicographic ascending comparison; compare the entire ticker vector first, then the integer weight vector in that same ticker order. Choose the least key among equal EXACT D portfolios. IDs, insertion order, names, sectors, map iteration and current time never break a tie. This is a total order on the decision-bearing allocation, not on incidental database IDs.
- Output holdings use that canonical order. The result contains the chosen security/weight pairs and stored drawdown; pf.app/pf.repo provide durable IDs/timestamps later. The existing optimizer Portfolio value denotes this value projection, not permission for pure code to obtain clock/random IDs.

Rationale: a ticker-only key cannot resolve distinct weights on the same set. Zero weights preserve the existing invariant. This permits economically sparse allocations represented by 16 records; the design must say this plainly instead of silently imposing 16 positive positions.

## D4: admission and finite budgets

Fixed admission policy: reject the WHOLE input if any candidate has a missing, malformed or misaligned history, even when 16 others are complete. No partial-universe filtering. Fewer than 16 candidates cannot produce a result. Extra matrix rows, noncanonical/duplicate candidate identities, invalid k (anything except the integer 16), or lookback mismatch are invalid. This extends the source's explicit missing-history refusal coherently; the prior “fewer than 16 full histories” wording must become a sufficient failure example, not an implied iff.

No universal candidate/history capacity number is inferred from “thousands/few years.” Instead propose one required immutable, caller-owned value:

`OptimizerLimits{maxCandidates, maxLookbackDays, maxScalarBytes}`

Each field is a positive integer representable in an unsigned 64-bit value; additionally maxCandidates>=16 and maxLookbackDays>=2. Scalar bytes mean UTF-8 bytes of each input string (including a price's entire decimal spelling). This bounds ticker/reference strings and numeric lexemes without a second precision knob. The app validates limits before invoking the optimizer, and the pure boundary independently rejects invalid limits. Candidate count <=maxCandidates; exactly lookbackDays observations, with lookbackDays<=maxLookbackDays; each candidate's ticker and each price/date string <=maxScalarBytes. Prices that cannot fit that lexical budget fail before conversion. Candidate names/sectors are not optimizer inputs: pass the Security identity projection.

Proposed explicit signature extension: `optimize(candidates, prices, k=16, lookbackDays, limits: OptimizerLimits) -> PortfolioValue | InfeasibleError`. The added limits argument is necessary new interface scope; it is not an ambient global or a silently optional default. pf.app constructs it from three required recommend command arguments `--max-candidates`, `--max-lookback-days`, `--max-scalar-bytes`; packets must show them in complete example invocations. The existing lookback argument remains separate. Test scenarios may choose small limits, but their values are scenario inputs, not hidden product policy or a deployment-capacity claim.

The feed signature correspondingly becomes `fetchPrices(tickers, lookbackDays, limits: OptimizerLimits) -> PriceMatrix | FeedError | CircuitOpenError`: it applies the same row/date/scalar bounds while assembling the response, not after constructing an unbounded value. A provider response exceeding them is FeedError with reason RESPONSE_LIMIT_EXCEEDED. A direct optimizer input exceeding them is InfeasibleError(LIMIT_EXCEEDED). No second feed-specific size policy is introduced.

### Explicit command-to-machine actor binding and repository admission

pf.app owns `make_run_actors(request: RecommendRequest, repoPort, feedPort, optimizerPort) -> RunActors | AdmissionError`. RecommendRequest contains a newly allocated runId, candidateSetId, lookbackDays, k=16 and validated immutable OptimizerLimits. All three limits flags are parsed once before a RecommendationRun record or driver is created; missing/malformed flags yield ValidationError and no run/actor/provider call. Construction copies/freezes values.

Repository candidate admission occurs ONCE in this factory, before entering the modeled run: `repoPort.LoadCandidateSet(candidateSetId, limits) -> CandidateSnapshot{identities,version} | NotFoundError | CorruptError | IOError | BusyError | ValidationError`. This typed read applies maxCandidates/maxScalarBytes while materializing identities, with ValidationError(LIMIT_EXCEEDED) for violated admission budgets. It has no write/publication handle. pf.app validates canonical/unique identities and sufficient candidate count before freezing the snapshot; invalid candidate semantics is InfeasibleError with D4's reason. Repository read failures are returned to CLI unchanged (the published class/exit mapping applies), consume ZERO feed retries and cause ZERO feed calls. They are not wrapped as FeedError and no fake Collecting.onError is dispatched. This is an admission read with finite input budgets; no wall-clock return bound is claimed for a blocked database read.

RunActors exposes exactly the machine's current actor names:
- `fetchPrices(input:{candidateSetId,lookbackDays}) -> PriceMatrix | FeedError | CircuitOpenError` uses the already-frozen CandidateSnapshot. It performs NO repository Load inside Collecting. These remain EXACTLY the two fields of Collecting.invoke.input; limits/snapshot are explicit construction captures. After checking the input matches the captured request, it calls `feedPort.fetchPrices(snapshot.identities,lookbackDays,request.limits)`.
- `optimize() -> PortfolioValue | InfeasibleError` has no machine input mapping. Only a uniquely accepted collecting result installs an instance-owned CollectedInput(snapshot,PriceMatrix); optimize calls `optimizerPort.optimize(identities,prices,16,request.lookbackDays,request.limits)`. It has no repository/global/clock lookup and does not perform persistence.
- The run driver sets context.candidateSetId/context.lookbackDays from that same request and resolves invoke.src through this one RunActors registry. Captures belong to the driver instance, not a process-global registry, environment, hidden fallback or persisted machine context.
- Each attempt has a driver-owned identity; only its accepted completion can install CollectedInput. Timeout/retired callbacks cannot replace it. An unresolved prior attempt is drained before another attempt starts. Limits and candidate snapshot remain immutable across retries.
- Driver pre-invocation checks require runId/candidateSetId/lookbackDays/limits to match its own validated request. Missing/cross-command registry, absent CollectedInput or binding mismatch is InternalError(ACTOR_BINDING_MISMATCH) BEFORE a port call; never infer defaults. Such construction/runtime binding failure is an operational command failure outside the normal machine input alphabet, not a new transition.

The chain is flags -> RecommendRequest -> bounded repository admission -> frozen CandidateSnapshot + pf.app.make_run_actors -> per-command driver -> unchanged machine input -> bound actor -> explicit limits argument. Only FeedError/CircuitOpenError and accepted feed timeout outcomes enter the collection retry rows. Pure optimizer outcomes enter its existing rows subject to the durable-publication barrier below. RecommendationRun machine context/input/graph and its8 oracle rows remain unchanged; its metadata and matrix must now explicitly state this resolved-outcome applicability.

Durability is a SEPARATE driver boundary, not hidden inside the pure optimizer. Before announcing a Ready result, pf.app constructs the identified/timestamped run+portfolio+holdings write set and calls the explicit CommitRecommendation operation in D5; an atomic Published receipt must exist before publishReady/stdout and current durable Ready success are claimed. The pure optimizer's60s gate bounds accepting its computed result; final repository publication gets the existing5s store-write admission budget, separately and explicitly. A resolved commit failure is an operational command failure; do not claim that a successful optimizer implied a durable Ready run. Unknown or unreturning publication is outside the run FSM's terminal-completion claim. The model's pure transition result can be calculated in advance, but externally publishing its terminal entry effects is held until the persistence outcome is confirmed. Terminal Failed persistence is subject to the same write contract: inability to persist it must be reported, never treated as a proven durable terminal state. M0/M1 and Run.matrix must distinguish logical transition observations from that real persistence boundary.

Keep the EXISTING optimizer error class. Add the closed reason field:

`InfeasibleError.reason ∈ {INVALID_INPUT, INCOMPLETE_HISTORY, INSUFFICIENT_CANDIDATES, LIMIT_EXCEEDED, INVALID_OUTPUT}`.

Validate in this order: limit record and k/lookback type/range; counts/scalar-size bounds; candidate identity and matrix shape/type/date/basis constraints; count>=16; every required price present and positive; then output invariants if validating a proposed result. Missing candidate rows and explicit null-cell sentinels use INCOMPLETE_HISTORY after the earlier shape checks; short/long present rows and all other malformed shape/type/value inputs use INVALID_INPUT; count<16 uses INSUFFICIENT_CANDIDATES; size limits use LIMIT_EXCEEDED; invalid proposed holdings/stored value uses INVALID_OUTPUT. All return the same InfeasibleError class; reasons separate diagnosis without inventing a new optimizer exception family. Negative cases must be otherwise well formed so the named reason wins this precedence. An invalid limit at CLI parsing is ValidationError; a direct optimizer call with it is InfeasibleError(INVALID_INPUT).

The extra limit argument changes the declared interface and needs approval before a packet uses it. It is a finite admission contract, not a runtime complexity or capacity guarantee. A literal global cap would avoid flags but would be an unsupported deployment-capacity decision; this proposal makes only the required resource boundary explicit.

## D5: errors and deterministic time

Preserve existing lower-layer names, explicitly reconcile the missing enumerations:

| Boundary/cause | Exact result or mapping |
|---|---|
| Provider HTTP 5xx, 429, timeout; invalid provider price-basis response | FeedError; original provider exception never escapes feed |
| Breaker open | CircuitOpenError; provider-call count remains zero |
| Recommendation collecting FeedError or CircuitOpenError | collectRetry; after exhaustion Failed; preserve the final typed cause to CLI |
| Optimizer InfeasibleError | Failed with that error and reason; no portfolio result |
| Application optimizer timeout wins before result acceptance | Failed; InternalError with reason OPTIMIZE_TIMEOUT, never a made-up InfeasibleError from the optimizer |
| Repository stale expectedVersion | ConflictError; no mutation; a repeated successful Save now has stale expectedVersion and refuses |
| Repository transient lock/busy | BusyError, distinct from version conflict; add it to architecture repo/store error enumeration, preserving Portfolio.matrix's current retriable set |
| Repository corruption / other I/O | At admission/read: CorruptError / IOError. At a write: the error is inside a confirmed Unpublished result, or publication remains Unresolved; a bare IOError does not prove rollback |
| Confirmed-Unpublished Portfolio commit with ConflictError or BusyError | commitRetry while budget remains; after exhaustion revert the in-memory stage to witnessed prior and return the final typed error; this attempt performed no publication |
| Portfolio timeout wins before publication admission, then nonpublication is confirmed | commitRetry; if exhausted, revert and return IOError with reason COMMIT_TIMEOUT |
| Pure no-matching guarded transition | RejectedError internally; app maps unauthorized role refusal to AuthzError and other invalid review action to ValidationError |
| Later lifecycle event for Ready or Failed Run | TerminalError internally and at CLI; no transition/effect; add to architecture's domain and CLI enumeration |
| Invalid rollback witness | routingFault plus recordRoutingError; CLI InternalError with reason INVALID_PRIOR; do not guess state |
| Reference actor failure with confirmed nonpublication | ValidationError; provider, conflict, busy, corruption and I/O retain their boundary types inside Unpublished results; only these resolved failures enter onError |
| Reference timeout wins before publication admission | close publication permission; cancellation/rollback and confirmed nonpublication precede recordReferenceTimeout; InternalError with reason REFERENCE_TIMEOUT; no partial result |

CLI mapping is explicit and injective over public ERROR CLASSES: success 0; AuthzError 2, NotFoundError 3, ConflictError 4, FeedError 5, InfeasibleError 6, CorruptError 7, ValidationError 8, InternalError 9, BusyError 10, TerminalError 11, IOError 12, CircuitOpenError 13. This numeric assignment is a new illustrative convention, not previously sourced behavior. stderr includes the stable class and, where specified, reason plus a bounded message; stdout is empty on failure. Existing operator wording may accompany the class. Different reasons within a class need not get different process codes. This statement must replace any ambiguous “each residual failure” wording that implies infinitely many provider exceptions require distinct codes.

Fixed timings: FETCH_TIMEOUT=20000ms per attempt; OPTIMIZE_TIMEOUT=60000ms per invocation; reference COMMAND_TIMEOUT=20000ms ONLY for refresh/upsert/build; COMMIT_TIMEOUT=5000ms per write attempt; breaker COOLDOWN=30000ms; threshold=5; MaxRetries=3 with retries initially zero. For retry ordinal j=1,2,3, fetch delays are 1000×2^(j−1)ms (1000,2000,4000), review delays 200×2^(j−1)ms (200,400,800). No jitter, fourth retry or unsourced cap. These exact schedules refine the existing approximate bases. Inject a monotonic clock/scheduler and this fixed retry policy; command inputs do not add configurable timing knobs for these sourced operations.

There are at most four initial/retry attempts. At retries=3, always-exhaustion wins before another backoff timer. Feed threshold checks `failures+1>=5`; a single run with one provider failure per fetch attempt can stop after four failures WITHOUT tripping a fresh breaker. Tests must not claim that run necessarily opens the breaker. A successful provider call resets failures; a cooldown admits one probe.

### Minimal explicit repository publication API

This revision selects pf.repo as the owner of WRITE admission arbitration and outcome records. pf.app owns the command and obtains/passes the handle through its existing app->repo port; repo never calls or imports pf.app. This replaces v2's undeclared app-owned shared gate. pf.app retains an independent command-local read/pure-result gate for feed/optimizer acceptance, which is not passed to a repository.

pf.model contains only TYPE declarations for the following values/interfaces; no operational helper or mutable registry is implemented there:
- `OperationId{commandId:string,attempt:uint64,kind:Save|Reference|Recommendation|Backup|Restore}`, unique per actual write attempt. commandId and attempt are supplied by app; repo rejects reuse in its session. This is operation correlation, not an idempotency-key API or durable execution authentication.
- `PublicationRequest`, a closed union containing the exact future operation's complete typed arguments: Save(value,expectedVersion); Reference(changeSet,expectedVersions); Recommendation(run,portfolio,holdings,expectedVersions); Backup(path,limits); Restore(path,expectedReceipt,limits). The repo session already fixes its store identity. The request is copied/frozen during BeginOperation; target identity, expected versions and all argument values are binding.
- `OperationHandle`, an opaque repo-session capability identifying exactly one OperationId and captured PublicationRequest. It is neither a map the caller may edit nor an environment key. Handles from another repo session, command, kind, target, version, request value or retired attempt cannot authorize this call.
- `OperationSnapshot` variants: Preparing, DeniedAwaitingDrain, Admitted, Published(receipt), Unpublished(error,timeoutWon), Unresolved(reference). Published/Unpublished/Unresolved are terminal returned outcomes only when all workers that could publish for that handle have stopped. Preparing/DeniedAwaitingDrain/Admitted are observable progress, not terminal command outcomes.
- `PublicationReceipt<R>{operationId,storeIdentity,targetIdentity,result:R}`; for Save/reference/recommendation, result includes the affected IDs and resulting versions; Backup includes its BackupReceipt; Restore includes the installed validated BackupReceipt. Receipt ownership is repo; app cannot submit a “Published” assertion to the API.
- `ReconciliationRef{repoSessionId,operationId,kind,storeIdentity,targetIdentity,expectedVersions,expectedResultDescription}`. The result description is immutable intent (IDs/versioned values, or backup digest/length/schema), not a self-authenticating proof. It identifies what is unresolved. It is safe to retain as command diagnostic data; no new journal/database entity is required.
- `RepoReply<R> = CallRejected(InternalError(OPERATION_BINDING_MISMATCH)) | Published(PublicationReceipt<R>) | Unpublished(typedError,timeoutWon) | Unresolved(ReconciliationRef)`. CallRejected means this invocation did not start: it does not cancel or resolve some OTHER operation supplied by mistake.

Public operations on one explicit `RepoSession`:
1. `OpenRepository(path, clock:MonotonicClock) -> RepoSession | RepoAdmissionError`. The clock is a declared injected interface (nowMillis only), with no app import; path, clock and session identity are bound at construction. Open/initial schema semantics are D6's. Other existing Load/Open errors remain direct read/admission errors.
2. `BeginOperation(id, request, deadlineAtMillis) -> OperationHandle | ValidationError`. It validates/copies the request, reserves a unique operation identity in THIS session and binds the absolute deadline. It performs no target write. App computes deadlineAt from the same injected clock and the relevant stated admission budget, with checked duration arithmetic.
3. `Save(value,expectedVersion,operation,expectedId) -> awaitable RepoReply<Ack>`; `ApplyReference(changeSet,expectedVersions,operation,expectedId) -> awaitable RepoReply<Ack>`; `CommitRecommendation(run,portfolio,holdings,expectedVersions,operation,expectedId) -> awaitable RepoReply<Ack>`; `Backup(path,limits,operation,expectedId) -> awaitable RepoReply<BackupReceipt>`; `Restore(path,expectedReceipt,limits,operation,expectedId) -> awaitable RepoReply<Ack>`. Here operation is OperationHandle and expectedId is OperationId independently captured by the command adapter before BeginOperation, not read back from the supplied handle. The repo compares both identities and EXACT captured argument equality before acquiring mutation authority; equal payloads from different commands/attempts cannot bypass this check. “Awaitable” means an in-process pending call observed by the command driver, not a new bus, service or distributed framework. The private repository commit path atomically validates/adopts this handle at its irreversible point; there is no public standalone “grant” boolean that could later authorize a different write. Recommendation's payload is closed: Ready requires a portfolio and exactly16 holdings; Failed requires null portfolio and empty holdings. Both persist their identified run atomically; other payload combinations are ValidationError before publication.
4. `Expire(operation) -> NotDue | TimeoutWon | AlreadyAdmitted | AlreadyResolved | InvalidHandle`. Repo samples its injected clock under the same serialized admission gate; before T it returns NotDue, at/after T it closes a still-Preparing gate. An admitted/publication-resolved operation cannot be reclassified as timeout.
5. `Inspect(operation) -> OperationSnapshot | InvalidHandle` is the explicit progress/result observation. `RequestDrain(operation) -> DrainRequested | AlreadyDrained | InvalidHandle` requests cancellation of reversible work/cleanup; it cannot revoke Admitted publication. Completion of drain is reflected by a terminal outcome, not by this acknowledgment. Native work that cannot stop remains progress, with no terminal return-time promise.
6. `Reconcile(reference) -> Published(receipt) | Unpublished(error,timeoutWon) | Unresolved(reference)` is READ-ONLY and repo-owned. It may use only authoritative outcome retained for that exact session/operation, or a retained native transaction/replace result that definitively identifies it. Matching current row contents/version or matching a current file digest alone does not prove this operation committed, because another writer may have produced them. A previously drained unknown outcome stays Unresolved absent definitive evidence. A reference from a restarted/unknown session also stays Unresolved; this proposal does not invent durable operation journaling or claim crash recovery exactly-once. Reconcile never repeats Save, restores a file, guesses a rollback or creates a new handle.

There is no module-global handle registry: operation records belong to the RepoSession supplied explicitly to the command. Only its implementation may transition outcomes after performing the actual database/file operation. Missing/wrong handles are rejected before mutation. App maps that construction defect to InternalError, without emitting a falsely confirmed failure for a valid different handle. The type-only clock/protocol declarations and passing an opaque value preserve current import edges: app->repo/model and repo->model/store; no repo->app or model operational implementation.

Explicit passage through reference actors: pf.app's refresh/build/upsert command driver first captures request, limits, RepoSession, expectedId and the absolute invoke-start-plus20s deadline. `selectEligibleConstituents` remains a bounded pure preparation; refresh then constructs the Index change set. `buildCandidateSet` prepares the deduplicated value; upsert prepares the canonical Security value. Only once the complete change set and expected versions are known does the adapter call BeginOperation with that frozen payload and the ORIGINAL deadline, then `ApplyReference(changeSet,expectedVersions,operation,expectedId)`. Preparation cannot reset the deadline. A preparation failure has confirmed no publication and follows the existing typed onError row; expiration during preparation cancels/drains that preparation before delivering the timeout row, without inventing a handle or starting a write. No nested Save creates a second independent permit or commits outside the parent handle. Actual operation metadata is captured by the per-command Reference actor adapter, so the existing machine invokes need no new input fields. Portfolio's `persistDecision(portfolioId,pending)` adapter similarly captures expectedVersion, frozen complete Save value, independent expectedId and its operation handle, then calls Save with all four explicit arguments. A retry creates a NEW attempt/id/handle; it never reuses the denied or resolved one.

For Backup/Restore, app captures expectedId, calls BeginOperation with that exact path/receipt/limits, and passes both returned handle and independent expectedId to Backup/Restore. The same record governs staged work and final destination publication/replacement. For M0 final recommendation publication, app uses ONE Recommendation handle plus independent expectedId and CommitRecommendation for run+portfolio+all16 holdings atomically, not separate independently admitted Saves. A Failed run uses the explicitly empty-result variant above. Return values, errors, cancellation and reconciliation traverse the explicit RepoSession interface; no “current operation” context lookup supplies missing values.

### Deadline arbitration, resolved model envelope and bounds

An ADMISSION deadline T is computed from invoke/operation start using the injected monotonic clock. Numerical budgets remain fetch20000ms, optimizer60000ms, reference20000ms, store-write5000ms and caller-supplied BackupLimits.deadlineMs. Reversible preparation must reach the repo's irreversible-admission gate before T. The gate samples now and admits iff now<T and no timeout already won. At exactly T admission is denied. If admitted before T, the actual commit/replacement may finish after T; its authoritative Published/Unpublished/Unresolved result decides the outcome. A successful publication stays success even when acknowledgment arrives after T. An unrelated raw timer never overrides it.

If Expire wins before admission, no later worker may publish with that handle. Only after rollback/drain and confirmed nonpublication does Unpublished(error,timeoutWon=true) permit an after:TIMEOUT event. If work was admitted and then returns a definite nonpublication error, Unpublished(error,timeoutWon=false) permits onError. Only the Unpublished variant—not an error name by itself—is eligible for rollback or a retry. ConflictError/BusyError remain the retriable classes in Portfolio's isRetriable guard after this outcome filter. A successful Save followed by lost/delayed acknowledgment cannot be retried as if it failed.

A drained Unresolved result means that publication may have occurred, but no worker for this attempt can publish later. App may call Reconcile once; if unresolved remains, it emits CLI IOError(PUBLICATION_OUTCOME_UNKNOWN), exit12, with command/operation identity and explicit unknown-publication text. It emits NO confirmed-failure or timeout transition, performs NO in-memory rollback presented as durable truth, and starts NO automatic retry. Other concurrent legitimate writers can still change the store; “unchanged” anywhere in these contracts means THIS failed attempt caused no publication, not globally frozen store bytes or version. An admitted operation that is still running cannot be called drained/Unresolved merely to return sooner; the command remains pending and reports progress as available. There is no total elapsed return bound.

The Portfolio and Reference machines are the modeled envelope for driver-delivered RESOLVED invoke outcomes. Their existing onDone/onError/after rows and all guard branches remain owed under this input contract. Admission arbitration, driver suppression of losing timers, and unresolved native publication are separate required adapter contracts. A command with unresolved publication has no corresponding resolved-terminal machine trace: its last confirmed modeled state and its possibly changed durable state are reported separately; it is NOT a new persisted PortfolioStatus/RunStatus, a synthetic final state, or an accepted “failed with unchanged store” path. Unknown publication is an explicitly reported operational residual outside the model, not a proof that the machine became stuck on a modeled valid event.

RecommendationRun's8-row pipeline similarly covers accepted feed/pure-optimizer outcomes under successful command admission and resolved persistence boundaries. Pre-run repository admission errors never enter its retry loop. An inability to establish terminal persistence, or unresolved final publication, cannot be claimed as a durable Ready/Failed completion. M0/M1 must test the persistence barrier independently of the pure transition row. The gate's existence is itself not proof that future native operations refine the graph.

Formal/control-flow termination statements apply ONLY to abstract traces with their prescribed transition/timer progress and resolution assumptions. Preserving graph rows does not prove the new adapter meets those assumptions, does not prove a blocked native call returns, and does not extend abstract completeness to unknown publication. Do not renew formal/runtime evidence. Existing generated formal/oracle files stay mathematical/historical artifacts; their applicability is stated explicitly in architecture, model/root, matrices, STATE and DECISIONS. If a later source check demands generated changes, stop for exact scoped review rather than rewriting proof inputs.

For feed/optimizer READ/PURE results, pf.app owns the separate command-local acceptance gate: acceptance before T reserves onDone even if driver delivery is late; a private completed computation accepted only at/after T loses and cannot populate CollectedInput. Only after relevant worker drain may the next attempt begin. The feed breaker only processes driver-accepted provider outcomes; stale callbacks cannot reset/trip it. Cooldown=30000ms is a scheduling threshold to permit a probe, not a termination/return-time guarantee; retry backoffs and counts likewise are not return-time bounds.

Bound vocabulary is CLOSED:
- input/admission representation bounds: candidate/source/row counts and maxScalarBytes, including rank before conversion; these limit admitted values, not prior caller allocation or a proved peak-RSS formula;
- publication/result-admission deadlines: the stated T budgets;
- retry bounds: at most3 retries and4 attempts, with fixed schedules;
- cooldown: earliest allowed half-open probe threshold;
- return/drain/reconciliation/OS completion: no fixed upper bound claimed. Any packet acceptance statement saying “bounded command,” “never hangs,” “terminates,” or “5s write timeout” must state the relevant category and resolution assumption, not restore a total-time guarantee.

Alternative rejected: acknowledging timeout solely from delayed observation, revoking an admitted irreversible write, or converting an unresolved IOError into a normal rollback. The selected API makes each impossible outcome an explicit semantic negative. No Go contract witness constitutes native cancellation, database atomicity, live conformance or liveness proof.

Repeated `recommend` creates a NEW run with retries=0. A same-invocation crash/recovery may inspect its durable run; reissuing the command is not idempotency-key lookup. No new idempotency feature is added.

## D6: reference operations and existing file-copy backup

Reference actors share D3's pf.domain normalization through pf.app. The raw ranked-row shape is CLOSED: `RankedConstituent{rank: RankLexeme, ticker: string, name: string, sector: string}`, with exactly these four fields, once each; no extras or missing fields. RankLexeme is a string matching `0|-?[1-9][0-9]*`, so signed integers including bounded negative values are representable, but +31, -0, decimal, exponent, boolean and numeric-object rank forms are invalid. Its UTF-8 byte length is checked against the EXISTING maxScalarBytes BEFORE any integer conversion or eligibility filtering. Rank is the raw input representation; the internal normalized row has a parsed integer rank.

Admission precedence for both provider and direct actor input: validate limits; check row-count bound; check closed field names/multiplicity and string types; check every scalar's byte bound INCLUDING rank, ticker, name and sector; validate rank grammar/convert within that bound; normalize/validate ticker through pf.app->pf.domain; finally filter ranks outside 1..30, sort by (rank, canonicalTicker), deduplicate canonical identities keeping their first sorted occurrence, and return the first30. A row does not evade scalar admission because its rank will later be ineligible. Streaming provider decoding must reject oversized scalar tokens before materializing/converting them; the known field-name set itself bounds key-token recognition, with no new numeric knob. A direct caller's prior allocation is outside the actor's custody; the actor still checks strings before conversion.

Provider over-count/over-scalar errors are FeedError(RESPONSE_LIMIT_EXCEEDED); provider row shape/type/rank-grammar or ticker invalidity is FeedError(INVALID_RESPONSE) at the provider-response adapter owned by pf.app/pf.feed. The feed performs bounded shape/lexical checks; pf.app performs shared ticker normalization and maps an invalid provider row result to FeedError. Direct reference actor versions of those cases are ValidationError(LIMIT_EXCEEDED) or ValidationError(INVALID_INPUT). Invalid limits are always caller ValidationError(INVALID_INPUT), before reading provider data. Error precedence is the ordered validation stages above; within a stage source row order, then fixed rank/ticker/name/sector order, selects the first diagnosis.

This explicitly completes stable tie/top-30 wording. A bounded well-formed rank31 or -1 is ineligible, not an error; an oversized rank lexeme is a limit error even if it would parse above30. Build unions already eligible normalized rows across source indices. A repeated security ticker updates the same record under the version contract; no new row is created.

Propose `ReferenceLimits{maxSourceIndices,maxProviderRowsPerIndex,maxScalarBytes}`, required immutable positive uint64s. Each entire ranked provider response is admitted against maxProviderRowsPerIndex BEFORE top-30 filtering; source index count against maxSourceIndices; each id/rank-lexeme/ticker/name/sector string against maxScalarBytes before conversion or normalization. The upsert result is one row; refresh returns <=30; build returns <=30×sourceCount distinct identities (checked multiplication, no overflow). These derived output limits avoid arbitrary extra result caps. Refresh requires `--max-provider-rows` and `--max-scalar-bytes`, with maxSourceIndices=1 derived from its one-index signature. Upsert requires only `--max-scalar-bytes`, with the two inventory limits=1 derived from its single-record signature. Build requires `--max-source-indices` and `--max-scalar-bytes`; it consumes already eligible persisted source lists, so maxProviderRowsPerIndex=30 is derived from that contract. No private configuration fills omitted values. Signatures become `selectEligibleConstituents(rows, limits)`, `upsertSecurityByTicker(security, limits)`, `buildCandidateSet(indices, limits)` and `fetchConstituents(index, limits)`. A provider response over its bound is FeedError(RESPONSE_LIMIT_EXCEEDED); direct actor count/string limit violations are ValidationError with reason LIMIT_EXCEEDED. Limits are not new persisted entities.

Timestamp policy: pf.model uses instants at microsecond resolution, rendered `YYYY-MM-DDTHH:MM:SS.ffffffZ` in contracts/receipts where timestamps are present. Capture timestamps via injected clock in app/repo; pure optimizer does not supply them. Convert an offset-bearing ingress instant to UTC only if exactly representable at microsecond precision; otherwise ValidationError. Absence is null where the domain allows unset acceptedAt. No backup timestamp field is needed. This fixes value interchange without requiring a database storage encoding.

Backup remains one RAW DuckDB database file copy, not an archive. To make M5's already-required checksum mismatch and wrong-schema tests meaningful without inventing hidden expected metadata, add the smallest explicit return/input value:

`BackupReceipt{byteLength: uint64, sha256: 64 lowercase hexadecimal characters, schemaVersion: 1}`.

`Backup(path, limits: BackupLimits, operation: OperationHandle, expectedId: OperationId) -> awaitable RepoReply<BackupReceipt>`.
`Restore(path, expected: BackupReceipt, limits: BackupLimits, operation: OperationHandle, expectedId: OperationId) -> awaitable RepoReply<Ack>`.
These are the same D5 operations; their complete frozen arguments are captured by BeginOperation. CorruptError/IOError/ValidationError are errors inside confirmed-Unpublished results where nonpublication is known; an uncertain admitted write returns Unresolved, not a bare error.
`BackupLimits{maxBytes: positive uint64, deadlineMs: positive uint64}` is required caller-owned input, with checked conversion to the runtime duration representation. These are the ONLY new backup budgets; they do not reuse the 20s reference timeout. Invalid receipt/limit values are ValidationError before publication; length/digest/schema/content mismatch is confirmed-Unpublished CorruptError; a confirmed-unpublished I/O failure or timeout winner carries IOError (reason BACKUP_TIMEOUT or RESTORE_TIMEOUT for timeout); live store remains unchanged only for a restore whose nonpublication failure is confirmed under D5; an unresolved publication outcome is explicitly unknown, never reported as unchanged.

A source snapshot, restore input, or staged copy larger than maxBytes is ValidationError(LIMIT_EXCEEDED); it is not called corrupt merely because it exceeds an admitted budget. An actual byte-length mismatch against an otherwise valid expected receipt is CorruptError. Structural receipt/limit validation precedes budget admission, which precedes content-integrity comparisons. A Backup destination that already exists is ValidationError(DESTINATION_EXISTS), with no write. stdout receipts use only the three bounded fields above; error messages use fixed class/reason text, not unbounded paths, provider bodies or stack traces.

The receipt is returned as data/stdout from backup and passed explicitly to restore; no sidecar lookup, implicit checksum download or self-computed “expected digest” is allowed. CLI surfaces are `pf backup <file> --max-bytes <B> --deadline-ms <T>` (stdout receipt JSON) and `pf restore <file> --expected-bytes <N> --expected-sha256 <H> --expected-schema-version 1 --max-bytes <B> --deadline-ms <T>`. There is no new receipt-file format, signing service, archive layout, persistence entity or authentication claim. The supplied receipt is a corruption-detection reference, not proof of origin; changing both backup and receipt can evade this check.

Repository metadata has schemaVersion=1 stored inside the DB as one application-owned metadata value. Open on a NONEXISTENT store path may exclusively create a new store and initialize its application schema plus schemaVersion=1 atomically before exposing it. This is M0's fresh-store path. It must not treat an arbitrary already-existing empty/zero-length file as its own new store. Open of an EXISTING database, Backup and Restore require exactly one valid supported metadata value; absent, duplicate, malformed or unsupported version is CorruptError, with no automatic schema1 stamp or migration. Concurrent initialization is serialized by repository ownership; a loser opens and validates the winner's existing result. Exact SQL table realization belongs to future repository implementation, but these value/initialization/compatibility rules may not be omitted. The copied database's version is checked independently of the receipt; a receipt saying1 cannot make schema2 valid. This is repository metadata scope, not a domain entity.

Backup obtains a quiescent consistent snapshot under repository coordination; the saved raw bytes, byte length and digest all describe that SAME closed immutable snapshot. It refuses output overwrite unless explicitly requested by a future separately reviewed operation; this proposal supplies no overwrite switch. Restore stages a copy to an operation-owned isolated path, checking input and staged lengths <=maxBytes, length equality and SHA256 against expected, supported schema and DB integrity, before an atomic replacement under repository coordination. The pre-replacement live store is untouched on confirmed nonpublication failure. D5's shared gate arbitrates restore replacement/backup destination publication against timeout: an admitted successful publication stays successful despite late completion delivery, while a timeout winner prevents publication and drains before a confirmed-failure return. An uncertain commit result is explicitly unknown. No claim of power-loss durability, total return-time bound or arbitrary kernel-call interruption is added.

Byte-exact round trip means: the closed backup snapshot bytes equal the validated staged bytes and installed closed bytes at the replacement boundary, before opening for further writes. A subsequent DuckDB connection's physical rewrite is not compared to that earlier snapshot; persisted entity values, IDs, timestamps and versions must still be equal after reopening. Backup does not promise bytes identical to a live mutating database at an earlier arbitrary instant. Native implementation tests with explicit input/retry/admission bounds and separately observed drain/return behavior are owed in future consumer work; the Go contract test does not copy a fake DB and claim repository conformance.

## Revised RED method: concrete bounded contract

Use one finite prose-to-facts extraction grammar and one bounded admissibility/observation evaluator, plus an independently authored expectation ledger. No general NLP, policy language, optimizer or fake application runtime.

1. The exact original six packet bytes are the baseline. A reviewed extraction table names each SUPPORTED COMPLETE CLAUSE, captures its byte span, and emits typed facts. For example M3's original outcome yields count=16, uniqueness, nonnegative and full-allocation relation; it does NOT yield integer bps, 10000, minimum direction or a value formula. Original “reject missing, non-finite, or insufficient histories” yields those refusal facts. Original M4 “Assert ordered actions” yields action-order assertion but no target assertion. Original M5 timeout/error wording yields those explicit actions but not a full-list/target promise. Unsupported text remains recorded as unrecognized text for fresh-context review, not silently translated.
2. Add controlled prose clauses to matched controls and eventual packets. Prefer this finite grammar over introducing JSON for the normative contract: e.g. “The optimizer minimizes historical maximum drawdown.”, “Holding weights are integer basis points and sum to exactly 10000.” and the precisely defined formula/tie/admission sentences in this proposal. Whole supported clauses carry typed operators and parameters; one-word presence is never the assertion. If JSON is also offered, it must round-trip to equivalent facts and prove it is not required.
3. Mandatory COMPLETE PROSE-ONLY POSITIVE CONTROL: original M3 plus all approved necessary clauses, with no JSON block, loads through the same reader and reaches the same feasible-proposal scenario. It must choose the expected lower-drawdown witness. Also preserve redundant equivalent clauses as a control; removing one of two equivalent statements does not remove the obligation.
4. Original M3 must reach two concrete witness completions, each satisfying every extracted original constraint: one chooses the first canonical 16-history portfolio deterministically and one chooses the second. Both include valid full-allocation weights and the reported drawdown. A missing objective is demonstrated by BOTH completions, not an UNKNOWN string. Each witness must include its explicit unresolved policy completion (e.g. min versus max historical drawdown); these are possible completions of the deficient packet, not silently adopted implementation rules.
5. The root/domain/architecture expectation ledger is read independently, never passed into the packet evaluator. It supplies manually fixed exact expected outcomes and source spans/policy approval ID. The evaluator only receives packet-derived facts, allowed machine/oracle/matrix inputs and scenario values. At least one mutation of root authority must change consistency diagnostics without changing the packet interpretation. Duplicate use of the same comparison/scoring function to construct expected values and actual values is prohibited.
6. Each semantic mutant changes/removes a whole obligation and reaches its target scenario with valid unrelated prerequisites. Record decoded facts, scenario ID, allowed outcome set, one admissible counterexample and the exact expectation violated. Parsing errors, absent JSON, unsupported text, generic UNKNOWN or nonzero process exit alone are not semantic RED. A recognized contradiction must change the outcome relation or expose inconsistent required outcomes, not merely trip syntax validation.

## Exact witness ledger for independent challenge

These are literal mathematical/contract expectations, not results of an executed test. Full witness fixtures expand the indicated 16 named records and all prices; do not generate expected results by calling the evaluator.

| Case | Literal expectation and cheapest wrong alternative it defeats |
|---|---|
| Basic objective | Universe A01..A16 and B01..B16. Each portfolio has 16 weights625. Every A row is [100,80,100], every B row [100,90,100]; three ascending declared dates, lookbackDays=3. Exact D_A=1/5, D_B=1/10; B wins, stored1000 vs2000. Reverse proposal/universe order. Original packet permits both deterministic completions; complete prose control excludes A. Direction=max mutant selects A. |
| Value rule | A portfolio has weights5000 on A01=[100,200,100],5000 on A02=[100,100,100],14 zero records. V=[1,3/2,1], D=1/3, stored3333. A comparator portfolio with16 weights625 and all rows[100,100,70] has D=3/10, stored3000 and wins. Periodic rebalancing gives the first portfolio D=1/4 and incorrectly reverses the ranking. |
| Historical maximum | All16 equal-weight rows[100,80,120,114] give D=1/5 and stored2000. Last-peak decline gives1/20, start-to-end loss gives0: both are wrong. |
| Rounding | All equal-weight rows[10000,9999.50] give exact0.50bps -> stored1; [10000,9999.51] gives0.49bps ->0. No epsilon. |
| Rounded collision | [10000,9999.49] gives0.51bps and [10000,9999.01] gives0.99bps; both store1, but0.51 wins even when the worse portfolio's ticker vector sorts first. Defeats rank-after-rounding. |
| Allocation | 16×625 valid. Last624 ->9999 and last626 ->10001 invalid. Replace two weights by624.5/625.5: sum still10000, type invalid. Weights(-1,1251,14×625): sum10000, sign invalid. Weights(0,1250,14×625): valid16 records, zero retained. |
| Same-ticker tie | Flat histories; same sorted16 tickers. Vectors(0,1250,14×625) and(625,625,14×625) both exactD0; first wins lexicographically, remains16 records. Reordering input records never changes key. |
| Normalization | ` a.b ` and `A.B` normalize to same identity and upsert one row. `A-B` remains distinct. `A B`, empty, non-ASCII are rejected; no suffix removal. Direct optimizer input `a.b` is invalid, not silently normalized. |
| Admission | 15 well-formed complete candidates ->INSUFFICIENT_CANDIDATES.16 complete ->admitted.17 with one missing row ->INCOMPLETE_HISTORY even when16 complete. Extra row or mixed priceBasis ->INVALID_INPUT. Duplicate/nonincreasing dates and L-vs-present-row-length mismatch ->INVALID_INPUT. Correct-length row ["100",null,"110"] ->INCOMPLETE_HISTORY; short row ["100","110"] for L3 ->INVALID_INPUT; empty-string cell ->INVALID_INPUT. |
| Bounds | Set explicit fixture budgets Nmax=16,Lmax=3,scalarMax sufficient for its longest string; count16 andL3 pass. Add17th complete candidate ->LIMIT_EXCEEDED. L4 with complete matching rows ->LIMIT_EXCEEDED. A scalar exactly supplied byte budget passes; one byte longer fails before numeric conversion. The chosen budgets are fixtures, not product limits. |
| Retry | Run retries2 permits delay4000ms then retries3 attempt; retries3 in collectRetry must fail before another delay. Review corresponding delay800ms. Feed failures3 plus failure increments to4; failures4 plus failure trips. Source-success resets counter. |
| Error mapping | Confirmed-Unpublished stale Save ->ConflictError/exit4; confirmed-Unpublished Busy ->retriable BusyError/exit10 if exhausted; confirmed-Unpublished IOError ->nonretriable rollback/exit12. Unresolved with an I/O cause NEVER follows that rollback mapping. Final Run event ->TerminalError/exit11 and no effects. Open breaker ->CircuitOpenError/exit13, zero provider calls. Compare to concrete wrong typed outcomes; do not execute a simulated application and call it conformance. |
| Reference bounds | Input rows equal maxProviderRowsPerIndex pass; one more fails even if only30 would be retained. With scalarMax=4 and other fields each <=4 bytes, rank string "31" is admitted then filtered with no error; "99999" fails before conversion/filtering: provider FeedError(RESPONSE_LIMIT_EXCEEDED), direct ValidationError(LIMIT_EXCEEDED). Within-budget "+31" fails INVALID_RESPONSE/INVALID_INPUT respectively. Missing rank, duplicate rank field or extra field fails closed shape; a numeric-object31 is invalid string type. Oversized rank paired with invalid grammar hits size first, not a parse or out-of-range bypass. Case-equivalent ticker merges; output remains <=30×sourceCount. |
| Explicit limits binding | Driver A validated maxCandidates16; a 17-candidate fixture exceeds its budget. Factory A uses16 for candidate admission; with an admitted snapshot, RunActors A passes the same16 to feed and optimizer. Substituting a command-B registry bound to17 must yield InternalError(ACTOR_BINDING_MISMATCH) BEFORE a port call, not admit17. Missing limits flag ->ValidationError with no run created. Mutating caller limits to17 after make_run_actors leaves A bound to16. Optimizing without accepted CollectedInput is a binding error, never an ambient repo/cache lookup. Collecting.invoke.input stays exactly candidateSetId/lookbackDays. |
| Pinned calendar | Response declares dates[d1,d2,d3]; every candidate aligned to those dates succeeds. One candidate exposing[d1,d3,d4] is invalid; replacing the vector with an intersection[d1,d3] cannot satisfy L3. A response with only separate candidate calendars and no declared common vector is FeedError(INVALID_RESPONSE), even if a convenient intersection exists. |
| Deadline before publication | Let T=5000ms for a write. Preparation finishes at4999; no admission occurred; timeout wins at5000; worker requests publication at5001 and is denied. Drain confirms nonpublication at5003. Only then after:COMMIT_TIMEOUT may drive retry; store/version unchanged. For restore, same sequence under its supplied T returns IOError(RESTORE_TIMEOUT) after drain and does not replace. |
| Delayed acknowledgment after publication | Admission4998, durable commit/replacement4999, deadline callback5000, acknowledgment5001. Exactly one onDone/success; new durable state remains published. A timeout/unchanged-store result is false. Same requirement for reference write, Portfolio Save and Restore. |
| Admitted commit crosses deadline | Admission4999, timeout callback5000, irreversible commit completes5002, acknowledgment5003. Publication owner wins; success is returned at5003. Total completion exceeds T but does not violate admission deadline. Confirms the proposal does not promise impossible cancellation of an admitted commit. |
| Result acceptance vs notification | Pure/read result accepted into handle at T-1, driver notification T+1 ->onDone. Private computation ends T-1 but acceptance attempted T+1 with timeout already won ->timeout, no CollectedInput/publication. Acceptance attempted exactlyT loses. |
| Uncertain publication | Admission<T, worker has stopped but commit outcome cannot be established ->Unresolved(ref), not a bare IOError. Read-only Reconcile(ref) remains unknown absent exact operation evidence. No onError/after event, rollback, automatic retry or current formal-completion claim; CLI exit12 says unknown. Same-current-target-version/data alone cannot turn it into Published. |
| Missing operation handle | Save/ApplyReference/CommitRecommendation/Backup/Restore called with no valid handle ->CallRejected(InternalError(OPERATION_BINDING_MISMATCH)), no mutation and no inferred “current operation.” An independently valid operation elsewhere is not canceled or reported failed. |
| Wrong-operation handle | Handle reserved for commandA/attempt1/Save portfolioP version7 cannot authorize commandB,attempt2,portfolioQ,version8,changed value,or Restore. Each fails captured-request/session equality before mutation. Correct matching handle permits exactly its one operation; reusing the handle is rejected. |
| Admitted is not published | Inspect returns Admitted after pre-T permit. Reporting success before repo records actual commit receipt is invalid; reporting timeout/rollback then is also invalid. Only final repo reply determines resolved outcome. |
| Confirmed failure vs unresolved matrix | For otherwise identical IOError causes, Unpublished(drained=true) selects Portfolio fallback to reverted; Unresolved(drained=true) selects NO modeled transition and unknown CLI report. Reference Unpublished selects failed/recordReferenceError; Unresolved does not. A keyword “IOError” dispatcher fails this pair. |
| Repo read admission | LoadCandidateSet returns NotFoundError,CorruptError,IOError orBusyError in four otherwise valid command cases. Factory returns that class/exit with no run created, no feed calls and retries0; relabeling FeedError or consuming a retry is wrong. |
| Scope of termination | A valid abstract row trace terminating does not admit a witness that labels an Admitted blocked write or Unresolved outcome as a successful runtime termination test. A matched resolved Published/Unpublished case may claim only its observed boundary result; all native proof remains prospective. |
| Fresh versus existing schema | Nonexistent path exclusively created by Open ->atomic schema1 initialization and successful M0 setup. An already-existing DB missing schemaVersion or containing2 ->CorruptError, no mutation/stamp. Receipt1 plus stored2 still fails restore validation. |
| Normalization ownership | pf.app invokes pf.domain.normalize_ticker once on raw ingress. A repo normalizer or pf.model operational helper is an invalid boundary/ownership alternative; canonical repo key and pure optimizer input remain unchanged afterward. |
| Backup | Receipt length/digest correspond to the exact supplied closed snapshot; a same-length one-byte change fails digest, a truncation fails length, schema2 with receipt1 fails schema; size=limit admitted,size=limit+1 refused. A timeout winner BEFORE publication admission yields no replacement after confirmed drain; success published before a delayed acknowledgment remains success under the dedicated witnesses. Byte comparison is before reopen; reopened logical entity/version equality is separate. These are required observations, not an executed DB proof. |

All four FSM requirements from the initial review remain: owning slices cover Run8, Feed6, Portfolio20, Reference12 rows; M0 Run subset3. Parse stable IDs/full columns/entry-exit lists, preserve ordered guard selection and separate always microsteps. Check concrete observation candidates for immediate target and exit-transition-entry action sequence, including empty lists, omitted/extra/reordered actions, external self-transition and explicit internal/targetless synthetic controls. Run Ready entry must remain observable. Refusal/final-state/guard-false/corrupt-prior cases remain separate; M3 has no machine.


## MANDATORY SKILLS
- developer; codebase-memory; pm_acceptor, under the unchanged hard-TDD and explicit exact-review checkpoints.

## nd_contract
status: in_progress

### evidence
- Root-authorized append-only tracker scope repair after immutable V3 architecture/policy approval; five AC and exact 21-path future custody mapped above. No implementation, RED, source/evidence edit, delivery, release or acceptance claim.
- Main 497419ab4512fcff765cd5feb27aed4c67b5608d; epic 7e36f3e7ddcf25565d5d4fe60b328df254eee91d; retained healthy story/MAC-lhu5 70652b948bf090008b1965c85daf36ea374daea4. Existing claim/dev-MAC-lhu5, in_progress and hard-tdd preserved.
- Fully read current lhu5/hgz1/lnu6 Bodies and immutable V3/challenge. Git confirms 20 existing source paths plus one new test. Graph generation 2026-09-06T10:15:08Z reports metadata_match/no recorded issue for 20 existing paths, missing new test, partial formal config ranges outside write scope; best-effort metadata is not completeness/semantic proof.
- Baseline scoped lint has zero errors and the existing ten unrelated vertical-slice review heuristics; cycles none; RTM 0 tagged requirements/56 stories/21 closed is structure only. Final append-prefix/metadata/hash and structural readback must be reported externally.
- No source/test/build/runtime/gate/evidence/golden/installed/service/remote mutation; source and RED remain HELD for independent exact scope/method review. No new delegation or queue/claim selection.

### proof
- [ ] AC #1: Source-consistent packet-alone contracts and complete 21-path applicability sweep pending exact review and implementation.
- [ ] AC #2: Approved numeric/identity/admission policies and all literal witnesses pending exact fixture/method review and proof.
- [ ] AC #3: Explicit bound actors/operation API and confirmed-publication/unknown-outcome obligations pending proof; native consumer implementation not claimed.
- [ ] AC #4: Full FSM inventories, semantic controls and dedicated durable Ready/Failed negatives pending independent PRE-RED review and actual assertion failures.
- [ ] AC #5: Isolated source-built checks, complete old/new subject/hash handoff and separate substantive consumer evidence review pending.


### 2026-09-06T11:16:45Z ramirosalas
BOUNDED D1 METHOD APPROVED — NOT SOURCE OR RED APPROVAL
Independent reviewer approved only the external finite D1 method at /tmp/MAC-lhu5-D1-method-repair-v4.PQFSHg. Main test SHA256147094fbfe1998374d1b9487c303272931378e7345186c101f5371ca5fba057a; survivor test e057ae71ef5b2a26086c6ba3640a7497543cd12be99e3ed2a868c073c5bde85e; canonical M3fixture621255b4ece8863c50c70cc0f9bc7ac3e831a6ad702a23408f5b41f80a3e3c67; authorreport42e649f9209449bcc639561992322abc3815c5703784b5d548c4946298a746c6.
Independent /tmp/MAC-lhu5-D1-PM-v4.QCW6qi/REVIEW.md SHA2568df8549325cb687abd8582a2deb917783ed2b0f31afbcea1729a8d13f1894e4e: fresh52test/subtestPASS+1packagePASS0FAIL/SKIP, plus exactunchangedpriorPMprobes5PASS+1packagePASS. Both previously failing selection leaves now reject removed members without panic. Prior /tmp/MAC-lhu5-D1-PM.Qeo409/REVIEW.md8816901aad8f65552dc633a2a043df41c9ce61ed797c401f143033e04a2c9e83 and failedprobes9d2af4364af2e0ca2f829e3bea35cf52293b4f2b4166a6190724aac59eb8fa75 remain retained, not reclassified.
Root read complete review, code through exact revisions, new67line survivor regression, and verified earlier method hashes/counts. Approval covers finite controlled-clause→typedfact→actual outcome witnesses (normalization, rebalancing, rounding, ties, input bounds/shape and survivor selection), NOT a native finance engine, global optimum or full packet interpretation.
Whole packet/root authority mapping, concrete decoder types, D2–D6, output/stored-value validation, full FSM row/target/ordered-effects witnesses, actual21-path source projection, generated mirror and repository RED remain pending. External prototype additional files are NOT new repository ownership. Full integration inventory/diff budget must be reviewed before any source/test write; all existing reserved lnu6 blocks and downstream hgz1 obligations remain protected.
New external-only bounded D5/D6 operation-fencing/publication-arbitration module assigned to /root/portfolio_d1_method_repair (Sol), no repository/tracker writes authorized for that actor.

## nd_contract
status: in_progress

### evidence
- Independent bounded module review approved; no source commit, RED freeze, delivery or story acceptance occurred.
- Earlier failed drafts/probes preserved with disclosed chronology.

### proof
- [x] Bounded D1 arithmetic and negative-model survivor semantics independently exercised.
- [ ] All story ACs still require their complete integrated source/test/evidence proof.


### 2026-09-06T11:50:40Z ramirosalas
BOUNDED METHOD REVIEW 2026-09-06: D5/D6 external candidate /tmp/MAC-lhu5-D5D6-method.ORvrzx independently reproduced 37 test passes (32 leaves+5 parents), package pass, 0 failures/skips. PM /tmp/MAC-lhu5-D5D6-review.GRFAl4/REVIEW.md SHA256 4ff1dbd8ade6d43331142542737528f013d09e11da7189ed0a1d53a7a20bb27c reports GAPS_FOUND in frozen-intent reconciliation, opaque session collision, exact Ready/Failed projection predicates, and confirmed-Unpublished generic IOError retry classification. Root read full source/report/probes and verified hashes; agrees these four bounded defects. Final PM probes 49run42pass7fail0skip: six blocking probe failures plus one disclosed version-overflow policy/range advisory; no failure discarded. Fresh external-only correction assigned root/portfolio_d5d6_repair_v2, smaller Sol high agent, preserving original artifacts and all D1 bounded approval. This is not repository RED/story rejection/source permission. All whole-source, decoder, FSM, remaining policy-method and before-edit inventory obligations remain; current 21-path repository scope unchanged.
