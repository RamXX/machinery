---
id: MAC-lnu6
title: "Remove unsafe frozen-test formatting exemptions"
status: open
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:36:13Z
created_by: ramirosalas
updated_at: 2026-09-05T19:45:35Z
content_hash: "sha256:b67914deddf4f0bcd3fdc8a1ae0183c38c61e566780b203483c4eed9f8d405c0"
blocks: [MAC-vx24, MAC-ou97]
---

## Description
## USER INTENT
Frozen tests must not be weakened under a misleading formatting exemption.

## Context (Embedded)
cmd/machinery/tokensequal.go uses strings.Fields. Different indentation or spaces within string literals can change program behavior while this comparison reports equal. BUILD guidance currently permits frozen-test edits using tokens-equal. The utility can remain as an honest whitespace-token comparison, never semantic or hard-TDD authority.

## Ownership
Own cmd/machinery/tokensequal.go, cmd/machinery/tokensequal_semantics_test.go, skills/machinery/references/build-md-template.md. Preserve other agents' edits; process story consumes template afterward.

## Boundary Map
PRODUCES:
- cmd/machinery/tokensequal.go -> honest whitespace-token comparison CLI
- cmd/machinery/tokensequal_semantics_test.go -> semantic-risk regression cases
- skills/machinery/references/build-md-template.md -> exact-byte frozen-test guidance
CONSUMES:
- Existing CLI implementation.
  source: strings.Fields comparison in cmd/machinery/tokensequal.go; no semantic parser.

### Story Acceptance Criteria
1. Remove all authorization to edit frozen RED tests based on tokens-equal; exact bytes and inventory define identity, and any amendment needs explicit new evidence revision plus replay.
2. CLI help/output/docs accurately describe whitespace-token comparison, not preserved program meaning or proof of formatting-only change.
3. Negative tests use spacing inside quoted literals and indentation-sensitive source that compare token-equal but have changed semantics; no guidance/gate treats that as approved frozen-test edit.
4. Positive genuine whitespace-only comparison utility retains documented exit behavior without claiming hard-TDD approval.
5. Real CLI calls and shipped template contract test establish user-facing behavior; no Paivot product dependency.

## Testing Requirements
Hard TDD RED author updates obsolete tests/guidance assertions before GREEN; expected behavioral failures and passing control, independent replay. Integration tests: MANDATORY (no mocks); real temporary files and actual binary. Run go test ./cmd/machinery -run 'TokensEqual|Frozen'. No skip-if-missing. No full preflight, pushes, remote mutations or installed binary replacement.

## OUT OF SCOPE
- Full language parser or semantic equivalence proof: not promised by this utility.
- Replay protocol implementation belongs to the process story.

## DIFF BUDGET
- ~3 files, under 300 changed LOC.

## MANDATORY SKILLS
- developer; codebase-memory; pm_acceptor.

## nd_contract
status: new

### evidence
- Created 2026-09-05 from architecture source finding.

### proof
- [ ] AC #1: frozen guidance rejects exemption.
- [ ] AC #2: honest help/output.
- [ ] AC #3: semantic counterexamples.
- [ ] AC #4: utility compatibility.
- [ ] AC #5: standalone real CLI.

## Acceptance Criteria


## Design


## Notes
ANCHOR ROUND-1 RUNTIME CLASSIFICATION: This story's required current cases are service-free Go/native filesystem/local-process tests and real local CLI paths; no Docker/Java/Node dependency is implicit in ordinary native suites. Preserve actual non-mocked path tests. If implementation introduces any external runtime-backed case, it MUST add a dedicated closed fragment under testdata/integration-lanes via the shared required lane and declare ownership/dependency before delivery; no skip-if-missing, env-gated omission or reliance on later incidental execution. Missing service cannot silently convert required coverage to success.

## History
- 2026-09-05T19:36:14Z dep_added: blocks MAC-vx24
- 2026-09-05T19:36:17Z dep_added: blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-ou97]]

## Comments
