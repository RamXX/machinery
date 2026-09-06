---
id: MAC-ou97
title: "E2e: Consumers receive trustworthy standalone assurance"
status: open
priority: 1
type: task
labels: [capstone]
parent: MAC-ui8a
created_at: 2026-09-05T19:36:14Z
created_by: ramirosalas
updated_at: 2026-09-05T19:45:33Z
content_hash: "sha256:ab8991e1f5807b793644d9673a6803501b0d7e97c7cf47f5664eff6850923cc3"
blocked_by: [MAC-hlae, MAC-sh60, MAC-yhg5, MAC-hwdb, MAC-2n83, MAC-hy71, MAC-gcrr, MAC-l7m0, MAC-vx24, MAC-lnu6, MAC-hpqp, MAC-yig6, MAC-lhu5, MAC-hgz1, MAC-qlw2, MAC-cn7q, MAC-6h0s, MAC-p9z1, MAC-62s6, MAC-bz1y, MAC-wi2u, MAC-avfp, MAC-imtz, MAC-8yai, MAC-pe9v, MAC-sd7g, MAC-sqpt, MAC-wbxq, MAC-rau8, MAC-u4oo]
was_blocked_by: [MAC-olrx, MAC-p8ce, MAC-a89e, MAC-2u36, MAC-p7jd, MAC-uzxr]
---

## Description
## USER INTENT
A consumer can rely on standalone Machinery to refuse known unsafe or unevidenced software changes, and the maintainer can hand off a thoroughly checked isolated local binary without disturbing live installations.

## Context (Embedded)
Final capstone for deterministic assurance hardening. Every sibling must be accepted first. Real production CLI and infrastructure only, not helper-only assertions. The separate epic completion stage runs the heavy preflight after this capstone; no prior story runs it.

## Ownership
Own cmd/machinery/hardening_e2e_test.go plus directly required fixture data under cmd/machinery/testdata/hardening-e2e. Do not modify production implementations; discovered bugs return to Sr PM as blocking P0 stories. Not alone: preserve edits.

## Boundary Map
PRODUCES:
- cmd/machinery/hardening_e2e_test.go -> full-path positive and adversarial consumer scenarios
- cmd/machinery/testdata/hardening-e2e -> isolated fixtures for user workflows
CONSUMES:
- MAC-hlae: internal/refine/refine.go
  source: Accepted saga user-facing behavior and regression contracts; invoke standalone Machinery CLI.
- MAC-sh60: internal/gates/oraclecov.go
  source: Accepted coverage user-facing behavior and regression contracts; invoke standalone Machinery CLI.
- MAC-olrx: internal/gates/suite.go
  source: Accepted owners user-facing behavior and regression contracts; invoke standalone Machinery CLI.
- MAC-p8ce: internal/gates/readscomplete.go
  source: Accepted reads user-facing behavior and regression contracts; invoke standalone Machinery CLI.
- MAC-yhg5: cmd/machinery/verify_checkers.go
  source: Accepted oci user-facing behavior and regression contracts; invoke standalone Machinery CLI.
- MAC-hwdb: adapters/opencode/plugins/machinery.js
  source: Accepted adapter user-facing behavior and regression contracts; invoke standalone Machinery CLI.
- MAC-2u36: internal/install/update.go
  source: Accepted install user-facing behavior and regression contracts; invoke standalone Machinery CLI.
- MAC-a89e: internal/gates/gates.go
  source: Accepted baseline user-facing behavior and regression contracts; invoke standalone Machinery CLI.
- MAC-p7jd: internal/gates/attest.go
  source: Accepted attest user-facing behavior and regression contracts; invoke standalone Machinery CLI.
- MAC-2n83: internal/designlock/designlock.go
  source: Accepted recovery user-facing behavior and regression contracts; invoke standalone Machinery CLI.
- MAC-hy71: .github/workflows/release.yml
  source: Accepted workflow user-facing behavior and regression contracts; invoke standalone Machinery CLI.
- MAC-gcrr: README.md
  source: Accepted docs user-facing behavior and regression contracts; invoke standalone Machinery CLI.
- MAC-l7m0: docs/test-assurance-contract.md
  source: Accepted architecture user-facing behavior and regression contracts; invoke standalone Machinery CLI.
- MAC-vx24: cmd/machinery/tdd.go
  source: Accepted process user-facing behavior and regression contracts; invoke standalone Machinery CLI.
- MAC-lnu6: cmd/machinery/tokensequal.go
  source: Accepted tokens user-facing behavior and regression contracts; invoke standalone Machinery CLI.

### Story Acceptance Criteria
1. User's valid design/implementation completes the standalone contract through real RED/unsafe challenges/GREEN replay and strong final handoff; no pvg/nd executable or metadata is required. Inspect named native execution outcomes and proof/review distinction.
2. Unsafe compensation mutation, comment-only/disabled test evidence, omitted parent policy/consumer/clause owner, changed frozen test/config or stale implementation review cannot obtain the same green assurance. Expected diagnostic/assertion is checked, not merely any failure.
3. Real OCI checker success/timeout/resource failure leaves no owned container; real OpenCode host translation handles valid response and blocks malformed/hanging/flooding children without residue.
4. Isolated installer receipt rerun converges all recorded targets; interrupted publication inspection/recovery preserves ownership and refuses conflicting content; routine regeneration never baselines new debt.
5. Local release policy rejects pending/failed/stale/wrong-SHA checks, and shallow ancestry fixture becomes valid only with required history. Do not claim GitHub policy is live until parent explicitly applies and verifies it at the authorized end.
6. E2e tests ONLY. No unit tests, no integration tests as substitutes. Tests exercise full system as a user would; no mocks of any kind, no stubs, no skip-if-missing. Missing real prerequisites fail clearly.
7. After capstone acceptance, epic completion runs scripts/preflight.sh fully once all changes are finalized; any failure must be fixed and the affected/full final gate rerun, never waived. Capture complete logs, final SHA and positive/negative inventory.
8. Only after final gate succeeds does dispatcher locally merge epic to main and cut an isolated candidate binary, recording version, SHA256 and smoke-test output. Never replace installed binary/plugins/skills/agents (NIL uses them), never dev-link. No GitHub push or remote mutation during work; final publication requires explicit parent coordination and accurate status.

## Testing Requirements
E2e tests ONLY. No unit tests, no integration tests. No mocks of any kind. Hard TDD author commits scenario assertions first; missing user behavior fails for intended runtime assertion with positive controls. Freeze RED tests and independently rerun. Use actual Go/Node/Java/Docker already available, isolated homes/workdirs, pinned engine assets. Leave user's dagger-engine-v0.21.9 container untouched.
Commands: go test ./cmd/machinery -run HardeningE2E -count=1; final epic stage scripts/preflight.sh. The final gate, not this test-only implementation, owns merge/build orchestration.

## OUT OF SCOPE
- Production fixes: discovered failures block this story and become P0 repair stories.
- Replacing active installation or publishing before final user-authorized end: prohibited.

## DIFF BUDGET
- ~2-8 test/fixture files, under 1500 changed LOC.

## MANDATORY SKILLS
- developer for E2E authoring; pm_acceptor for independent review; orchestrator for final local completion.

## nd_contract
status: new

### evidence
- Created 2026-09-05, blocked by all hardening sibling stories.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified
- [ ] AC #6: independently verified
- [ ] AC #7: independently verified
- [ ] AC #8: independently verified

Observable outcome: the user can complete valid software delivery and see unsafe or stale cases refused by the actual binary.

## Acceptance Criteria


## Design


## Notes
AUTHORITATIVE CAPSTONE TESTING CORRECTION: This is an integrated tests-only capstone after all implementations already pass. No manufactured missing-feature RED is required and hard-tdd label is removed. Instead every unsafe mutation must trigger the intended assertion/block while safe positive controls pass, proving sensitivity. Preserve reviewed test bytes and provide exact actual E2E execution evidence. Earlier capstone RED wording is superseded by this correction.
ANCHOR ROUND-1 AUTHORITATIVE EXECUTION-LANE REPAIR
General rule: required runtime tests need deterministic provisioning, explicit closed inventory, actual native execution accounting and teardown. Missing infrastructure must fail the REQUIRED lane, not be silently skipped. Ordinary native suites may explicitly exclude registered service-backed tests using a dedicated build tag.
Classification: Final user-perspective tests combine Docker/Java/Node/native replay; all runtime prerequisites mandatory.
Replace prior bare go test -run HardeningE2E command with shared required lane selecting capstone after provisioning. Validate union contains all delivered sibling fragments and every expected case ran. Pure capstone remains no hard-tdd label; safe/unsafe sensitivity required.
PRODUCES:
- testdata/integration-lanes/capstone.json -> this story's closed suite fragment, with exact source/test IDs, runtime/pin requirements and bounded execution configuration
- cmd/machinery/hardening_e2e_test.go -> actual named runtime cases registered in the fragment
CONSUMES:
- MAC-hpqp: testdata/integration-lanes/schema.json
  schema: Closed versioned native-runner suite fragment with exact source/test identities, runtime requirements and bounded command selection.
- MAC-hpqp: scripts/integration-lane/main.go
  endpoint: go run ./scripts/integration-lane --lane required (Makefile test-integration invokes same entrypoint).
Additional acceptance criteria: fragment matches actual test sources both directions; all registered cases actually start/terminate with expected positive/negative outcomes; no cached/skipped/empty/partial/fabricated-summary success; real provisioned positive and missing-runtime/fresh-cache failure diagnostics; no owned container/process leaks. Required local preflight and hosted CI execute the same union. Do not edit shared root inventory; own only this fragment. RED source, fixture, fragment and runner configuration are frozen together after review. Any exact test names introduced in RED must remain registered through GREEN.
No heavy preflight until final gate; no GitHub mutation; no active installation replacement. This note supersedes any earlier command implying service-backed tests execute in unprovisioned ordinary package suites.

## History
- 2026-09-05T19:36:14Z dep_added: blocked_by MAC-hlae
- 2026-09-05T19:36:14Z dep_added: blocked_by MAC-sh60
- 2026-09-05T19:36:14Z dep_added: blocked_by MAC-olrx
- 2026-09-05T19:36:15Z dep_added: blocked_by MAC-p8ce
- 2026-09-05T19:36:15Z dep_added: blocked_by MAC-yhg5
- 2026-09-05T19:36:15Z dep_added: blocked_by MAC-hwdb
- 2026-09-05T19:36:15Z dep_added: blocked_by MAC-2u36
- 2026-09-05T19:36:15Z dep_added: blocked_by MAC-a89e
- 2026-09-05T19:36:16Z dep_added: blocked_by MAC-p7jd
- 2026-09-05T19:36:16Z dep_added: blocked_by MAC-2n83
- 2026-09-05T19:36:16Z dep_added: blocked_by MAC-hy71
- 2026-09-05T19:36:16Z dep_added: blocked_by MAC-gcrr
- 2026-09-05T19:36:16Z dep_added: blocked_by MAC-l7m0
- 2026-09-05T19:36:17Z dep_added: blocked_by MAC-vx24
- 2026-09-05T19:36:17Z dep_added: blocked_by MAC-lnu6
- 2026-09-05T19:45:33Z dep_added: blocked_by MAC-hpqp
- 2026-09-05T20:27:03Z dep_removed: was_blocked_by MAC-olrx
- 2026-09-05T23:07:17Z dep_removed: was_blocked_by MAC-p8ce
- 2026-09-05T23:57:31Z dep_removed: was_blocked_by MAC-a89e
- 2026-09-06T01:17:22Z dep_added: blocked_by MAC-yig6
- 2026-09-06T02:37:14Z dep_removed: was_blocked_by MAC-2u36
- 2026-09-06T03:03:05Z dep_added: blocked_by MAC-uzxr
- 2026-09-06T03:03:20Z dep_added: blocked_by MAC-lhu5
- 2026-09-06T03:17:09Z dep_added: blocked_by MAC-hgz1
- 2026-09-06T07:47:13Z dep_removed: was_blocked_by MAC-p7jd
- 2026-09-06T08:07:54Z dep_removed: was_blocked_by MAC-uzxr
- 2026-09-06T09:10:15Z dep_added: blocked_by MAC-qlw2
- 2026-09-06T09:10:16Z dep_added: blocked_by MAC-cn7q
- 2026-09-06T09:10:17Z dep_added: blocked_by MAC-6h0s
- 2026-09-06T09:10:18Z dep_added: blocked_by MAC-p9z1
- 2026-09-06T09:10:19Z dep_added: blocked_by MAC-62s6
- 2026-09-06T09:10:20Z dep_added: blocked_by MAC-bz1y
- 2026-09-06T09:10:20Z dep_added: blocked_by MAC-wi2u
- 2026-09-06T09:10:21Z dep_added: blocked_by MAC-avfp
- 2026-09-06T09:10:22Z dep_added: blocked_by MAC-imtz
- 2026-09-06T09:10:23Z dep_added: blocked_by MAC-8yai
- 2026-09-06T09:10:24Z dep_added: blocked_by MAC-pe9v
- 2026-09-06T09:10:25Z dep_added: blocked_by MAC-sd7g
- 2026-09-06T09:10:26Z dep_added: blocked_by MAC-sqpt
- 2026-09-06T09:10:27Z dep_added: blocked_by MAC-wbxq
- 2026-09-06T09:10:28Z dep_added: blocked_by MAC-rau8
- 2026-09-06T09:10:29Z dep_added: blocked_by MAC-u4oo

## Links
- Parent: [[MAC-ui8a]]
- Blocked by: [[MAC-hlae]], [[MAC-sh60]], [[MAC-yhg5]], [[MAC-hwdb]], [[MAC-2n83]], [[MAC-hy71]], [[MAC-gcrr]], [[MAC-l7m0]], [[MAC-vx24]], [[MAC-lnu6]], [[MAC-hpqp]], [[MAC-yig6]], [[MAC-lhu5]], [[MAC-hgz1]], [[MAC-qlw2]], [[MAC-cn7q]], [[MAC-6h0s]], [[MAC-p9z1]], [[MAC-62s6]], [[MAC-bz1y]], [[MAC-wi2u]], [[MAC-avfp]], [[MAC-imtz]], [[MAC-8yai]], [[MAC-pe9v]], [[MAC-sd7g]], [[MAC-sqpt]], [[MAC-wbxq]], [[MAC-rau8]], [[MAC-u4oo]]
- Was blocked by: [[MAC-olrx]], [[MAC-p8ce]], [[MAC-a89e]], [[MAC-2u36]], [[MAC-p7jd]], [[MAC-uzxr]]

## Comments
