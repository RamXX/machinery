---
id: MAC-gcrr
title: "Publish precise consumer assurance guidance"
status: open
priority: 0
type: task
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:33:47Z
created_by: ramirosalas
updated_at: 2026-09-05T19:33:47Z
content_hash: "sha256:59e585471ddac2d26404b718e1430b3569de68cdb2dcbeab5f619700c69a680c"
blocked_by: [MAC-vx24, MAC-hy71, MAC-p7jd]
---

## Description
## USER INTENT
Strengthen Machinery mission-critical assurance with observable fail-closed behavior and precise limits.

## Context (Embedded)
NEXT5 registry-relative input workaround, NEXT7 daemon-visible bind-path explanation, NEXT8 generator semantics release notes, NEXT2 failclosed plugin ownership, NEXT13 explicitly deferred hierarchy. Assessment runtime residuals: concurrency, replay/duplication/loss, migration/restore/load remain outside bounded proofs. User requires clarity and deterministic delivery obligations, not claims of unbounded correctness.

## Ownership
Own only README.md, docs/external-checkers.md, docs/release-notes.md, CHANGELOG.md, NEXT.md, cmd/machinery/assurance_docs_test.go and directly associated tests. You are not alone: preserve other edits and coordinate shared paths.

## Boundary Map
PRODUCES:
- README.md -> hardened contract and regression evidence
- docs/external-checkers.md -> hardened contract and regression evidence
- docs/release-notes.md -> hardened contract and regression evidence
- CHANGELOG.md -> hardened contract and regression evidence
- NEXT.md -> hardened contract and regression evidence
- cmd/machinery/assurance_docs_test.go -> hardened contract and regression evidence
CONSUMES:
- Existing Machinery implementation.
  source: Existing machinery check, verify-checkers, verify-formal, baseline, attest CLI contracts; docs/external-checkers.md runtime registry and daemon bind mounts.

## Acceptance Criteria
1. Document daemon-visible bind source paths accurately: host socket can work with shared path layout; dind/same namespace is one option. Include containerized CI validation and concrete error diagnosis without pretending engine ambient availability is guaranteed.
2. Document repo-root --registry workaround for registry-relative inputs and same-platform emulation alternative; NEXT5/6 claims must not say impossible where a safe supported workaround exists.
3. Introduce consumer-visible release note discipline covering generated-output/proof-scope changes, compatibility and migration, including historic v0.6.3 omission and current hardening changes. Do not silently call changed proofs equivalent.
4. Update NEXT with delivered vs remaining dispositions based on actual accepted stories; retain feature extensions unresolved until authorized and hierarchy deferred; #2 remains failclosed ownership policy, not unsafe fallback.
5. Explain artifact consistency, proof execution, test execution and current review as separate claims. Known runtime exclusions lead to named owned replay/race/migration/restore/load test obligations in new standalone enforcement guidance after its contract is delivered; do not suggest text alone enforces correctness.
6. Add targeted executable documentation/CLI contract checks for commands and safety-critical guidance, plus actual supported registry/bind-path example flow. Product docs require Machinery only, never Paivot.

## Testing Requirements
- Explicit hard TDD: separate RED test author, expected behavioral assertion failure plus passing control on unmodified production, independent PM replay, frozen RED test/fixture bytes retained through GREEN. Compilation, import, timeout or unavailable infrastructure are not valid RED. Existing tests encoding the unsafe contract may be corrected during reviewed RED; GREEN cannot silently weaken them.
- Integration tests: MANDATORY (no mocks). Exercise real process/filesystem/runtime boundaries; no stubs, no skip-if-missing. Fixture policy inputs may test parser logic but are not live remote proof.
- go test ./cmd/machinery -run 'AssuranceDocs|RepositoryContract'; real temp registry/checker supported-path invocation. Targeted tests only; final heavy preflight elsewhere.
- Full scripts/preflight.sh ONLY at end of epic. No pushes, sync, remote mutation, installed binary/plugin/agent/skill replacement, or dev-link. Another agent uses installed Machinery in NIL; builds/tests use isolated output and homes.
- Shipped product enforcement is standalone Machinery: never requires pvg, nd, Paivot labels/metadata or commit conventions. Paivot is local development coordination only.

## OUT OF SCOPE
- Other assessment areas have sibling stories; final preflight/local main merge/isolated candidate binary are final gate responsibilities. Do not omit small directly related safety fixes.

## DIFF BUDGET
- ~6 files, under 900 changed LOC; overrun triggers PM investigation rather than weaker proof.

## MANDATORY SKILLS
- developer; codebase-memory; pm_acceptor.

## Delivery Requirements
Append exact RED/GREEN SHAs, commands/output, ownership inventory and per-AC proof. Use pvg story deliver; PM accepts separately. Never invoke pushing pvg story merge.

## nd_contract
status: new

### evidence
- Created 2026-09-05 from assessment and source-verified interfaces.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified
- [ ] AC #6: independently verified

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-05T19:35:07Z dep_added: blocked_by MAC-vx24
- 2026-09-05T19:35:08Z dep_added: blocked_by MAC-hy71
- 2026-09-05T19:35:08Z dep_added: blocked_by MAC-p7jd

## Links
- Parent: [[MAC-ui8a]]
- Blocked by: [[MAC-vx24]], [[MAC-hy71]], [[MAC-p7jd]]

## Comments
