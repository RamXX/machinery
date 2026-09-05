---
id: MAC-ui8a
title: "Deterministic assurance hardening"
status: open
priority: 0
type: epic
created_at: 2026-09-05T19:28:10Z
created_by: ramirosalas
updated_at: 2026-09-05T19:39:15Z
content_hash: "sha256:0ffa4402c4b4f6e983ed13952e5f772fed9d623eb66430ca50ab5476ecbade15"
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
