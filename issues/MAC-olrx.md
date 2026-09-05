---
id: MAC-olrx
title: "Preserve oracle ownership through decomposition"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:26Z
created_by: ramirosalas
updated_at: 2026-09-05T19:47:31Z
content_hash: "sha256:2a383ae267ae63963bcaa3f5b8198e93ba7a419bf3f92befef493931d276bf63"
blocks: [MAC-vx24, MAC-ou97]
assignee: dev-MAC-olrx
---

## Description
## USER INTENT
Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.

## Context (Embedded)
Assessment F6 parent: selectInSnapshot removes gt for machine-less decomposed parents even with Policy/Isolation oracle obligations; explicit Gt correctly reports missing IDs. NEXT3 clauseSet drops machine owner and Gt/Gd pools guard names across all machines.

## Ownership
Own only these paths and directly associated tests: internal/gates/suite.go, internal/gates/clauses.go, internal/gates/clausecov.go, internal/gates/obligation_ownership_test.go. You are not alone in this codebase; preserve other edits and coordinate any shared-file changes with dispatcher.

## Boundary Map
PRODUCES:
- internal/gates/suite.go -> hardened behavior and regression proof
- internal/gates/clauses.go -> hardened behavior and regression proof
- internal/gates/clausecov.go -> hardened behavior and regression proof
- internal/gates/obligation_ownership_test.go -> hardened behavior and regression proof
CONSUMES:
- Existing Machinery source interfaces.
  spec: gates.Select(design, gateList, impl string) (Selection, error); gates.CheckOracleCoverage(design, impl string) *Gate; collectClauseDecls(g *Gate, design string) []clauseSet; checkClauseDrift(g *Gate, design string); checkClauseCoverage(g *Gate, design string, corpus testCorpusData)

### Story Acceptance Criteria
1. Default selection with --impl retains Gt for machine-less decomposed parents that own relational oracle obligations; absence of machines cannot erase Policy/Isolation coverage. Truly obligation-free parents remain valid with honest zero-obligation reporting.
2. CLAUSES declarations and corresponding drift/coverage rows key on owning machine plus guard; identical guard names in different machines are independent. Declaring or satisfying one never arms or satisfies the other.
3. Positive same-machine clauses still require every falsifying clause, reject duplicate/conflicting/missing declarations and preserve stable IDs; malformed/orphan owner identity fails clearly.
4. Negative full-path tests cover parent missing tests, same-name guards with distinct clauses, one machine omitted, mismatched owner and removed oracle; positive controls show correctly covered parents and two valid independent machines.
5. Run actual gate selection plus Gt/Gd results, not only helper structures. Document precise ownership in relevant diagnostics without relying on prose to enforce it.

## Testing Requirements
- Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
- Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
- Commands: go test ./internal/gates -run 'Clause|Parent|Selection|Obligation'; temporary complete decomposed parent designs through machinery check default selection and explicit gates.
- No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.

## OUT OF SCOPE
- Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
- Global heavy preflight, main merge and local release binary belong to final epic gate.

## DIFF BUDGET
- ~4-6 files, under 800 changed LOC; material overrun requires PM investigation, not weakened requirements.

## MANDATORY SKILLS
- developer for implementation; codebase-memory for discovery; pm_acceptor for independent acceptance.

## Delivery Requirements
Use pvg story deliver, never close. Append RED/GREEN SHAs, exact commands and outputs, test inventory, independent proof per AC, and any residual limits to shared nd. Do not use pushing pvg story merge.

## nd_contract
status: new

### evidence
- Created 2026-09-05; source signatures verified at assessment base 497419ab4512fcff765cd5feb27aed4c67b5608d.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified

## Acceptance Criteria


## Design


## Notes
AUTHORITATIVE USER CONSTRAINTS 2026-09-05: Machinery product must be standalone, never require Paivot/pvg/nd, workflow labels or commit conventions. Local development coordination only may use Paivot. Another agent uses installed Machinery in NIL: do not replace installed binary/plugins/skills/agents; no dev-link or live install/update. Build isolated candidate only. No GitHub push/mutation during work. Full scripts/preflight.sh only final epic gate. RED author may update preexisting tests that encode superseded unsafe behavior with explicit review and genuine assertion-failure proof; after RED approval freeze exact tests/fixtures/config bytes.
ANCHOR ROUND-1 RUNTIME CLASSIFICATION: This story's required current cases are service-free Go/native filesystem/local-process tests and real local CLI paths; no Docker/Java/Node dependency is implicit in ordinary native suites. Preserve actual non-mocked path tests. If implementation introduces any external runtime-backed case, it MUST add a dedicated closed fragment under testdata/integration-lanes via the shared required lane and declare ownership/dependency before delivery; no skip-if-missing, env-gated omission or reliance on later incidental execution. Missing service cannot silently convert required coverage to success.

## History
- 2026-09-05T19:35:07Z dep_added: blocks MAC-vx24
- 2026-09-05T19:36:14Z dep_added: blocks MAC-ou97
- 2026-09-05T19:47:31Z status: open -> in_progress
- 2026-09-05T19:47:31Z claimed by dev-MAC-olrx

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-ou97]]

## Comments
