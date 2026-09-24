---
id: MAC-p8f1
title: "Resolve authorization admissions against a closed capability declaration list (operation and resource scope)"
status: open
priority: 2
type: feature
labels: [consistency-layer, authz, gy-rules, from-next]
created_at: 2026-09-24T21:32:34Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:34Z
content_hash: "sha256:a95f8ea9253ffa59632a02070dc8eacb552b9a1ad42299afe86b02b674ecbe4f"
blocks: [MAC-qa6n]
---

## Description
Resolve authorization admissions against a closed capability declaration list

Problem: since 0.10.0 (commit 1021a18f "close System writes against authorization", then the Stage 4 cutover), Gy-rules `authz.dl` proves every System write and every `PRODUCES{Entity.action}` action has exactly one admission row in the marked `AUTHORIZATION.md` inventory (`machinery:authorization-inventory`) or a `(no authorization: <reason>)` waiver. The capability an admission names resolves only against `c4_element` ids (rules/consistency/authz.dl: `capability(C) :- c4_element(C, _, _).`; docs/consistency-layer-proposal.md:388-389 states "no declared capability list exists yet"). Any C4 element id therefore passes as a capability, and nothing checks which operation the admission grants or over which resource scope. The 0.9.0 note in NEXT.md also named matrix producers and residual-table presets as targets; those readers were removed in the 0.10.0 cutover.

Proposed fix: add a closed, design-level capability declaration list (a declared artifact or Architecture Contract table: capability id, operation, resource scope, owning C4 element). Project it into projection 2.0 as a defining relation (for example `capability(id, operation, resource)`), make admissions name a declared capability, and extend `authz.dl` so an admission resolves to that list including operation and resource scope rather than to any C4 element. Keep implementation permission enforcement a separate review claim (not decided by this rule). Provide a migration path: designs whose admissions name C4 elements get a clear diagnostic and a documented upgrade, and the Gy baseline (`machinery baseline --gate gy`) can record the transitional findings.

Acceptance criteria:
1. A capability declaration format is documented (skill plus docs/external-checkers.md relation table) and projected as a defining relation with stable ids and `{path, line}` sources; `schemas/projection-v2.schema.json` is regenerated from the relation catalog.
2. An admission naming an undeclared capability is `authz_unknown_capability`; a C4 element id that is not a declared capability no longer satisfies it.
3. An admission whose operation or resource scope does not match the declared capability (for example a write action admitted by a read-only capability, or a resource outside the capability's scope) is a named Gy-rules finding.
4. Each new finding has one fixture in internal/experiments that fails without the rule plus a near-neighbour that must not fire; Souffle parity (`datalog-parity`) passes for the rule file.
5. Bundled examples declare their capabilities and stay Gy-clean; the golden corpus change is reviewed.
6. CHANGELOG carries Compatibility and migration plus Proof scope notes stating that implementation permission enforcement is not proven by this gate.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-24T21:33:57Z dep_added: blocks MAC-qa6n

## Links
- Blocks: [[MAC-qa6n]]

## Comments
