---
id: MAC-lhu5
title: "Make portfolio milestone packets satisfy their standalone handoff contract"
status: open
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T03:03:05Z
created_by: ramirosalas
updated_at: 2026-09-06T03:03:05Z
content_hash: "sha256:532722e778948d9d5a9c45d8aee1cca9fbff72fabcfc982f041a76c1a39ec05b"
blocks: [MAC-ou97]
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


## History
- 2026-09-06T03:03:20Z dep_added: blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-ou97]]

## Comments
