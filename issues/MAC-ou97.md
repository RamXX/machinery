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
updated_at: 2026-09-05T19:38:47Z
content_hash: "sha256:681677c880b2868882d879233d54788af6471f8b4a3955cada5ee35d9b2f424e"
blocked_by: [MAC-hlae, MAC-sh60, MAC-olrx, MAC-p8ce, MAC-yhg5, MAC-hwdb, MAC-2u36, MAC-a89e, MAC-p7jd, MAC-2n83, MAC-hy71, MAC-gcrr, MAC-l7m0, MAC-vx24, MAC-lnu6]
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

## Links
- Parent: [[MAC-ui8a]]
- Blocked by: [[MAC-hlae]], [[MAC-sh60]], [[MAC-olrx]], [[MAC-p8ce]], [[MAC-yhg5]], [[MAC-hwdb]], [[MAC-2u36]], [[MAC-a89e]], [[MAC-p7jd]], [[MAC-2n83]], [[MAC-hy71]], [[MAC-gcrr]], [[MAC-l7m0]], [[MAC-vx24]], [[MAC-lnu6]]

## Comments
