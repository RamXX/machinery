---
id: MAC-gcrr
title: "Publish precise consumer assurance guidance"
status: closed
priority: 0
type: task
labels: [hard-tdd, accepted]
parent: MAC-ui8a
created_at: 2026-09-05T19:33:47Z
created_by: ramirosalas
updated_at: 2026-09-07T09:15:11Z
content_hash: "sha256:1e409c7e1de480a8ae2a9f710a2783f98be6b500534b4d1e55aa3e0e2db951c8"
blocked_by: [MAC-vx24]
was_blocked_by: [MAC-p8ce, MAC-a89e, MAC-2u36, MAC-p7jd, MAC-hpqp, MAC-yhg5, MAC-hwdb, MAC-hy71]
follows: [MAC-p8ce, MAC-a89e, MAC-2u36, MAC-p7jd, MAC-hpqp, MAC-yhg5, MAC-hwdb]
assignee: dev-MAC-gcrr
closed_at: 2026-09-07T09:15:11Z
close_reason: "Accepted: precise consumer assurance guidance published; merged to local epic"
---

## Description
## USER INTENT
Strengthen Machinery mission-critical assurance with observable fail-closed behavior and precise limits.

## Context (Embedded)
NEXT5 registry-relative input workaround, NEXT7 daemon-visible bind-path explanation, NEXT8 generator semantics release notes, NEXT2 failclosed plugin ownership, NEXT13 explicitly deferred hierarchy. Assessment runtime residuals: concurrency, replay/duplication/loss, migration/restore/load remain outside bounded proofs. User requires clarity and deterministic delivery obligations, not claims of unbounded correctness.

## Ownership
Own the seven core paths README.md, docs/agent-portability.md, docs/external-checkers.md, docs/release-notes.md, CHANGELOG.md, NEXT.md and cmd/machinery/assurance_docs_test.go. Preserve the two already authorized runtime-lane paths from the ANCHOR ROUND-1 execution-lane repair: cmd/machinery/assurance_docs_integration_test.go and testdata/integration-lanes/consumer-docs.json (nine total). NEXT.md remains user-owned/gitignored/local-only; never force-add it. You are not alone: preserve other edits and coordinate shared paths.

## Boundary Map
PRODUCES:
- README.md -> hardened contract and regression evidence, including installer/bootstrap parity and plugin failure limits
- docs/agent-portability.md -> receipt-aware bootstrap, safe repair and direct-versus-host-plugin transaction boundaries
- cmd/machinery/assurance_docs_integration_test.go -> previously authorized real checker registry/bind-path example
- testdata/integration-lanes/consumer-docs.json -> previously authorized closed required runtime fragment
- docs/external-checkers.md -> hardened contract and regression evidence
- docs/release-notes.md -> hardened contract and regression evidence
- CHANGELOG.md -> hardened contract and regression evidence
- NEXT.md -> hardened contract and regression evidence
- cmd/machinery/assurance_docs_test.go -> hardened contract and regression evidence
CONSUMES:
- Existing Machinery implementation.
  source: Existing machinery check, verify-checkers, verify-formal, baseline, attest CLI contracts; docs/external-checkers.md runtime registry and daemon bind mounts.

## Installer documentation deliverable (delegated by MAC-2u36)
This is the explicit doc output for existing AC3/4/6, not an installer outcome or test waiver. MAC-2u36 keeps all seven implementation AC and frozen native/real CLI proof. MAC-gcrr owns both README.md and docs/agent-portability.md before capstone/final release; its existing dependency on MAC-2u36 and MAC-ou97's downstream dependency remain unchanged.
- Correct README.md:621-622 and docs/agent-portability.md:122-124 (source inspected at 15252d255fa12a131fbc2cdb13f593a212fd4ccb): bootstrap on a valid supported receipt uses the complete recorded/discovered home/native/plugin plan, just like ordinary update without selectors, retaining per-group copy/symlink modes. No-receipt bootstrap retains plugin-aware defaults; explicit homes/targets are incompatible with bootstrap. Remove false defaults-only/not-from-receipt/native-only-via-update claims.
- Explain safely edited/missing owned regular content repair with unchanged safe parents/topology; do not imply arbitrary missing-root recreation. Corrupt/unsupported/unsafe receipt and unsafe artifact/parent substitutions or uncertain plugin ownership fail closed with diagnostics, including inspection despite --skip-plugins. No silent fallback to defaults.
- Scope transactional restoration precisely to binary/direct home/native/receipt changes before direct commit, including final parent receipt failure and absent pre-state, subject to refusal to overwrite concurrent foreign changes and retained recoverable journal/diagnosis. Do not promise unconditional restoration of every root or atomic host-plugin rollback.
- Correct README.md:605-608 warning-only language and corresponding portability/release guidance where applicable. Exact source update.go:63-66, 225-237 and refreshHostPlugins:532-628 establishes host CLI absence, understood managed-scope refusal/failure and bad inventory/response as returned failures, not successful-update warnings. Direct changes may already be committed when later host plugin refresh fails. Host-owned plugin caches are updated via host CLIs, not edited/rolled back by Machinery; preserve/report retry obligations and actual side-effect limits. Distinguish this post-direct-commit failure from pre-direct-plan plugin discovery/ownership failure and explicit --skip-plugins behavior.
- Consume accepted MAC-2u36 frozen real native CLI convergence, safety and rollback contract/evidence and exact delivered source; do not copy obsolete prose as authority. Safety-critical native docs tests in assurance_docs_test.go must read BOTH shipped files, reject the original false defaults-only and warning-only claims, and require accurate parity/repair/fail-closed/transaction-limit guidance. Keep these paired with actual accepted CLI observations; string checks/mock integration are not substitutes for MAC-2u36 runtime proof. Preserve the separately required real registry/bind-path flow and its lane inventory.
- Include these semantics and compatibility/repair implications in the already owned release-note discipline. No docs write or test amendment is granted by this scope repair; normal independent RED/test review and GREEN ownership apply.

### Story Acceptance Criteria
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
- Revised forecast: nine explicit paths (seven core including agent-portability, plus the two previously authorized integration/lane paths), approximately 900-1050 changed LOC. Original six-file/under-900 estimate omitted the previously authorized lane paths and this bounded documentation repair. Report actual per-file changes and native/runtime costs; overrun triggers PM investigation rather than weaker proof. No permission to discard prior runtime proof or force-add local NEXT.md.

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

Observable outcome: the user can follow accurate standalone commands and see explicit remaining assurance limits.

## Acceptance Criteria


## Design


## Notes
ANCHOR ROUND-1 AUTHORITATIVE EXECUTION-LANE REPAIR
General rule: required runtime tests need deterministic provisioning, explicit closed inventory, actual native execution accounting and teardown. Missing infrastructure must fail the REQUIRED lane, not be silently skipped. Ordinary native suites may explicitly exclude registered service-backed tests using a dedicated build tag.
Classification: Actual checker registry/bind-path documentation example requires Docker/pinned checker closure.
Split real OCI docs example into cmd/machinery/assurance_docs_integration_test.go; ordinary doc-contract tests remain native. Follow required lane for real example proof.
PRODUCES:
- testdata/integration-lanes/consumer-docs.json -> this story's closed suite fragment, with exact source/test IDs, runtime/pin requirements and bounded execution configuration
- cmd/machinery/assurance_docs_integration_test.go -> actual named runtime cases registered in the fragment
CONSUMES:
- MAC-hpqp: testdata/integration-lanes/schema.json
  schema: Closed versioned native-runner suite fragment with exact source/test identities, runtime requirements and bounded command selection.
- MAC-hpqp: scripts/integration-lane/main.go
  endpoint: go run ./scripts/integration-lane --lane required (Makefile test-integration invokes same entrypoint).
Additional acceptance criteria: fragment matches actual test sources both directions; all registered cases actually start/terminate with expected positive/negative outcomes; no cached/skipped/empty/partial/fabricated-summary success; real provisioned positive and missing-runtime/fresh-cache failure diagnostics; no owned container/process leaks. Required local preflight and hosted CI execute the same union. Do not edit shared root inventory; own only this fragment. RED source, fixture, fragment and runner configuration are frozen together after review. Any exact test names introduced in RED must remain registered through GREEN.
No heavy preflight until final gate; no GitHub mutation; no active installation replacement. This note supersedes any earlier command implying service-backed tests execute in unprovisioned ordinary package suites.
WORKTREE HANDOFF: NEXT.md is an existing user-owned gitignored file at /Users/ramirosalas/workspace/machinery/NEXT.md, excluded intentionally via local git exclude. Do not assume story worktrees contain it and do not force-add it to git. Read the current root copy when revising dispositions; preserve unrelated user content and report how its local-only update is handed back. Repository docs/release notes must carry durable public safety guidance independently. Existing release pvg-compatible tarball wording is consumer compatibility, not a runtime dependency; preserve compatible artifact naming without requiring Paivot.
CANONICAL DOCUMENTATION DIVISION — root-authorized independent PM split. No implementation or test waiver.

## nd_contract
status: new

### evidence
- Sr PM canonical documentation division only: MAC-gcrr owns README.md + docs/agent-portability.md and safety regression/release guidance; installer seven AC/frozen proof unchanged, independent PM acceptance pending. Source 15252d2 confirms receipt-aware parity, safe repair, strict inspection, direct rollback/foreign-change protection and post-direct-commit host-plugin error boundary. Existing dependencies retained; no docs/source/test writes or status/claim change.
- Created 2026-09-05 from assessment and source-verified interfaces.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified
- [ ] AC #6: independently verified


Prior canonical Description (historical, superseded only by bounded documentation binding):
> ## USER INTENT
> Strengthen Machinery mission-critical assurance with observable fail-closed behavior and precise limits.
> 
> ## Context (Embedded)
> NEXT5 registry-relative input workaround, NEXT7 daemon-visible bind-path explanation, NEXT8 generator semantics release notes, NEXT2 failclosed plugin ownership, NEXT13 explicitly deferred hierarchy. Assessment runtime residuals: concurrency, replay/duplication/loss, migration/restore/load remain outside bounded proofs. User requires clarity and deterministic delivery obligations, not claims of unbounded correctness.
> 
> ## Ownership
> Own only README.md, docs/external-checkers.md, docs/release-notes.md, CHANGELOG.md, NEXT.md, cmd/machinery/assurance_docs_test.go and directly associated tests. You are not alone: preserve other edits and coordinate shared paths.
> 
> ## Boundary Map
> PRODUCES:
> - README.md -> hardened contract and regression evidence
> - docs/external-checkers.md -> hardened contract and regression evidence
> - docs/release-notes.md -> hardened contract and regression evidence
> - CHANGELOG.md -> hardened contract and regression evidence
> - NEXT.md -> hardened contract and regression evidence
> - cmd/machinery/assurance_docs_test.go -> hardened contract and regression evidence
> CONSUMES:
> - Existing Machinery implementation.
>   source: Existing machinery check, verify-checkers, verify-formal, baseline, attest CLI contracts; docs/external-checkers.md runtime registry and daemon bind mounts.
> 
> ### Story Acceptance Criteria
> 1. Document daemon-visible bind source paths accurately: host socket can work with shared path layout; dind/same namespace is one option. Include containerized CI validation and concrete error diagnosis without pretending engine ambient availability is guaranteed.
> 2. Document repo-root --registry workaround for registry-relative inputs and same-platform emulation alternative; NEXT5/6 claims must not say impossible where a safe supported workaround exists.
> 3. Introduce consumer-visible release note discipline covering generated-output/proof-scope changes, compatibility and migration, including historic v0.6.3 omission and current hardening changes. Do not silently call changed proofs equivalent.
> 4. Update NEXT with delivered vs remaining dispositions based on actual accepted stories; retain feature extensions unresolved until authorized and hierarchy deferred; #2 remains failclosed ownership policy, not unsafe fallback.
> 5. Explain artifact consistency, proof execution, test execution and current review as separate claims. Known runtime exclusions lead to named owned replay/race/migration/restore/load test obligations in new standalone enforcement guidance after its contract is delivered; do not suggest text alone enforces correctness.
> 6. Add targeted executable documentation/CLI contract checks for commands and safety-critical guidance, plus actual supported registry/bind-path example flow. Product docs require Machinery only, never Paivot.
> 
> ## Testing Requirements
> - Explicit hard TDD: separate RED test author, expected behavioral assertion failure plus passing control on unmodified production, independent PM replay, frozen RED test/fixture bytes retained through GREEN. Compilation, import, timeout or unavailable infrastructure are not valid RED. Existing tests encoding the unsafe contract may be corrected during reviewed RED; GREEN cannot silently weaken them.
> - Integration tests: MANDATORY (no mocks). Exercise real process/filesystem/runtime boundaries; no stubs, no skip-if-missing. Fixture policy inputs may test parser logic but are not live remote proof.
> - go test ./cmd/machinery -run 'AssuranceDocs|RepositoryContract'; real temp registry/checker supported-path invocation. Targeted tests only; final heavy preflight elsewhere.
> - Full scripts/preflight.sh ONLY at end of epic. No pushes, sync, remote mutation, installed binary/plugin/agent/skill replacement, or dev-link. Another agent uses installed Machinery in NIL; builds/tests use isolated output and homes.
> - Shipped product enforcement is standalone Machinery: never requires pvg, nd, Paivot labels/metadata or commit conventions. Paivot is local development coordination only.
> 
> ## OUT OF SCOPE
> - Other assessment areas have sibling stories; final preflight/local main merge/isolated candidate binary are final gate responsibilities. Do not omit small directly related safety fixes.
> 
> ## DIFF BUDGET
> - ~6 files, under 900 changed LOC; overrun triggers PM investigation rather than weaker proof.
> 
> ## MANDATORY SKILLS
> - developer; codebase-memory; pm_acceptor.
> 
> ## Delivery Requirements
> Append exact RED/GREEN SHAs, commands/output, ownership inventory and per-AC proof. Use pvg story deliver; PM accepts separately. Never invoke pushing pvg story merge.
> 
> ## nd_contract
> status: new
> 
> ### evidence
> - Created 2026-09-05 from assessment and source-verified interfaces.
> 
> ### proof
> - [ ] AC #1: independently verified
> - [ ] AC #2: independently verified
> - [ ] AC #3: independently verified
> - [ ] AC #4: independently verified
> - [ ] AC #5: independently verified
> - [ ] AC #6: independently verified
> 
> Observable outcome: the user can follow accurate standalone commands and see explicit remaining assurance limits.
> 

## History
- 2026-09-05T19:35:07Z dep_added: blocked_by MAC-vx24
- 2026-09-05T19:35:08Z dep_added: blocked_by MAC-hy71
- 2026-09-05T19:35:08Z dep_added: blocked_by MAC-p7jd
- 2026-09-05T19:35:08Z dep_added: blocked_by MAC-p8ce
- 2026-09-05T19:35:08Z dep_added: blocked_by MAC-a89e
- 2026-09-05T19:35:08Z dep_added: blocked_by MAC-2u36
- 2026-09-05T19:35:09Z dep_added: blocked_by MAC-yhg5
- 2026-09-05T19:35:09Z dep_added: blocked_by MAC-hwdb
- 2026-09-05T19:36:16Z dep_added: blocks MAC-ou97
- 2026-09-05T19:45:33Z dep_added: blocked_by MAC-hpqp
- 2026-09-05T23:07:17Z dep_removed: was_blocked_by MAC-p8ce
- 2026-09-05T23:57:31Z dep_removed: was_blocked_by MAC-a89e
- 2026-09-06T02:37:14Z dep_removed: was_blocked_by MAC-2u36
- 2026-09-06T07:47:13Z dep_removed: was_blocked_by MAC-p7jd
- 2026-09-07T04:45:25Z dep_removed: was_blocked_by MAC-hpqp
- 2026-09-07T07:22:44Z dep_removed: was_blocked_by MAC-yhg5
- 2026-09-07T08:24:55Z dep_removed: was_blocked_by MAC-hwdb
- 2026-09-07T08:25:16Z status: open -> in_progress
- 2026-09-07T08:25:16Z auto-follows: linked to predecessor MAC-p8ce
- 2026-09-07T08:25:17Z auto-follows: linked to predecessor MAC-a89e
- 2026-09-07T08:25:17Z auto-follows: linked to predecessor MAC-2u36
- 2026-09-07T08:25:17Z auto-follows: linked to predecessor MAC-p7jd
- 2026-09-07T08:25:17Z auto-follows: linked to predecessor MAC-hpqp
- 2026-09-07T08:25:17Z auto-follows: linked to predecessor MAC-yhg5
- 2026-09-07T08:25:17Z auto-follows: linked to predecessor MAC-hwdb
- 2026-09-07T09:15:08Z dep_removed: was_blocked_by MAC-hy71
- 2026-09-07T09:15:11Z status: in_progress -> closed
- 2026-09-07T09:15:11Z dep_removed: no_longer_blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Blocked by: [[MAC-vx24]]
- Was blocked by: [[MAC-p8ce]], [[MAC-a89e]], [[MAC-2u36]], [[MAC-p7jd]], [[MAC-hpqp]], [[MAC-yhg5]], [[MAC-hwdb]], [[MAC-hy71]]
- Follows: [[MAC-p8ce]], [[MAC-a89e]], [[MAC-2u36]], [[MAC-p7jd]], [[MAC-hpqp]], [[MAC-yhg5]], [[MAC-hwdb]]

## Comments

### 2026-09-06T02:35:49Z ramirosalas
## nd_contract
status: new

### evidence
- Sr PM canonical documentation division only: MAC-gcrr owns README.md + docs/agent-portability.md and safety regression/release guidance; installer seven AC/frozen proof unchanged, independent PM acceptance pending. Source 15252d2 confirms receipt-aware parity, safe repair, strict inspection, direct rollback/foreign-change protection and post-direct-commit host-plugin error boundary. Existing dependencies retained; no docs/source/test writes or status/claim change.
- Created 2026-09-05 from assessment and source-verified interfaces.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified
- [ ] AC #6: independently verified


### 2026-09-07T09:15:10Z ramirosalas
ACCEPTED 2026-09-07 — RED aab8218 (14 doc-contract failures, 1 control) -> GREEN 6283b42, 8 files +1000/-21 within budget. Installer delegation delivered: README/agent-portability receipt-parity bootstrap truth (complete recorded home/native/plugin plan, per-group copy/symlink modes; no-receipt plugin-aware defaults; explicit homes incompatible with bootstrap; false defaults-only/not-from-receipt/native-only-via-update claims removed); fail-closed unsafe receipts incl. skip-plugins inspection; safe repair scope honest. NEXT dispositions: 2 resolved fail-closed documented; 5 registry workaround documented (extension unapproved); 6 emulation is not native-host evidence; 7 daemon-visible paths (dind one option); 8 release discipline + hy71 pointer. CHANGELOG Unreleased section for accepted batch; release-notes discipline incl. v0.6.3 omission. Epic-accepted behaviors stated with honest residuals. NEXT.md local-only updated, never force-added. Lane 8/8 green. Coordinator merged. Record: .git/machinery-evidence-20260906.TEFZ7D/gcrr-record.md
