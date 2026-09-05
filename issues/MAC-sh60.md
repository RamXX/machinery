---
id: MAC-sh60
title: "Require executable oracle coverage evidence"
status: open
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:26Z
created_by: ramirosalas
updated_at: 2026-09-05T19:45:34Z
content_hash: "sha256:86af3ed225e2ce8e146fbdfdece526bdae7a0b328b457c30631cee438471839e"
blocks: [MAC-vx24, MAC-ou97]
---

## Description
## USER INTENT
Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.

## Context (Embedded)
Assessment F2 reproduced full Gt coverage from // TODO parse "Thing.oracle.md" using "|"; unused filename/delimiter constants; a Go test disabled with //go:build ignore; and Elixir # comment oracle IDs. fileNameCited uses raw corpus text, hash comment stripping excludes .exs. Coverage discovery is not execution proof.

## Ownership
Own only these paths and directly associated tests: internal/gates/oraclecov.go, internal/gates/oraclecov_negative_test.go. You are not alone in this codebase; preserve other edits and coordinate any shared-file changes with dispatcher.

## Boundary Map
PRODUCES:
- internal/gates/oraclecov.go -> hardened behavior and regression proof
- internal/gates/oraclecov_negative_test.go -> hardened behavior and regression proof
CONSUMES:
- Existing Machinery source interfaces.
  spec: gates.CheckOracleCoverage(design, impl string) *Gate

### Story Acceptance Criteria
1. Comments, docstrings-only citations, unused filename/delimiter declarations, disabled Go build-tag files and commented Elixir IDs cannot establish oracle-row or wholesale table coverage.
2. Positive real literal-ID tests and genuine conformance table parsers remain discoverable across currently supported languages. Unsupported/ambiguous parser evidence is explicitly uncovered rather than silently credited; no filename-keyword heuristic can confer wholesale coverage.
3. Add full CheckOracleCoverage regression cases for every assessment bypass, zero-test directories, mixed legitimate/disabled tests, malformed oracle references and actual positive parser/literal fixtures.
4. Gate output accurately labels discovery versus actual test execution; this story never claims static references prove assertions ran. Later native execution protocol supplies execution evidence.
5. No loss of stable-ID boundary checks, orphan/missing-oracle diagnostics or clause coverage. Integration exercises actual CLI against temporary real file trees; no injected fake coverage result.

## Testing Requirements
- Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
- Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
- Commands: go test ./internal/gates -run 'Oracle|Conformance|Coverage'; targeted command tests exercising machinery check with gt. RED bypass fixtures must currently produce the wrong accepted result and fail test assertions, while positive control passes.
- No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.

## OUT OF SCOPE
- Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
- Global heavy preflight, main merge and local release binary belong to final epic gate.

## DIFF BUDGET
- ~3-5 files, under 700 changed LOC; material overrun requires PM investigation, not weakened requirements.

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

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-ou97]]

## Comments
