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
updated_at: 2026-09-06T12:05:48Z
content_hash: "sha256:4b92ae6bd47b49df2cf6e00c4183efda1b8a84243fc7ddf93df7dc200578fbc7"
blocked_by: [MAC-hwdb, MAC-hy71, MAC-gcrr, MAC-vx24, MAC-p9z1, MAC-62s6, MAC-bz1y, MAC-wi2u, MAC-avfp, MAC-imtz, MAC-8yai, MAC-pe9v, MAC-sd7g, MAC-sqpt, MAC-wbxq, MAC-rau8, MAC-u4oo, MAC-5ft8, MAC-al5u, MAC-1u2v]
was_blocked_by: [MAC-olrx, MAC-p8ce, MAC-a89e, MAC-2u36, MAC-p7jd, MAC-uzxr, MAC-l7m0, MAC-p9wm, MAC-yig6, MAC-lhu5, MAC-sh60, MAC-hgz1, MAC-lnu6, MAC-wi5z, MAC-qlw2, MAC-cn7q, MAC-6h0s, MAC-hpqp, MAC-2n83, MAC-yhg5, MAC-hlae]
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
- 2026-09-06T09:10:30Z dep_added: blocked_by MAC-5ft8
- 2026-09-06T09:10:31Z dep_added: blocked_by MAC-al5u
- 2026-09-06T09:10:32Z dep_added: blocked_by MAC-1u2v
- 2026-09-06T10:30:30Z dep_removed: was_blocked_by MAC-l7m0
- 2026-09-06T12:04:53Z dep_added: blocked_by MAC-p9wm
- 2026-09-06T12:31:39Z dep_removed: was_blocked_by MAC-p9wm
- 2026-09-06T16:34:07Z dep_added: blocked_by MAC-wi5z
- 2026-09-06T18:35:57Z dep_removed: was_blocked_by MAC-yig6
- 2026-09-06T19:21:07Z dep_removed: was_blocked_by MAC-lhu5
- 2026-09-06T19:21:17Z dep_removed: was_blocked_by MAC-sh60
- 2026-09-06T20:22:41Z dep_removed: was_blocked_by MAC-hgz1
- 2026-09-06T23:26:04Z dep_removed: was_blocked_by MAC-lnu6
- 2026-09-06T23:26:04Z dep_removed: was_blocked_by MAC-wi5z
- 2026-09-07T00:29:10Z dep_removed: was_blocked_by MAC-qlw2
- 2026-09-07T01:42:39Z dep_removed: was_blocked_by MAC-cn7q
- 2026-09-07T01:42:40Z dep_removed: was_blocked_by MAC-6h0s
- 2026-09-07T04:45:25Z dep_removed: was_blocked_by MAC-hpqp
- 2026-09-07T07:22:44Z dep_removed: was_blocked_by MAC-2n83
- 2026-09-07T07:22:44Z dep_removed: was_blocked_by MAC-yhg5
- 2026-09-07T08:24:55Z dep_removed: was_blocked_by MAC-hlae

## Links
- Parent: [[MAC-ui8a]]
- Blocked by: [[MAC-hwdb]], [[MAC-hy71]], [[MAC-gcrr]], [[MAC-vx24]], [[MAC-p9z1]], [[MAC-62s6]], [[MAC-bz1y]], [[MAC-wi2u]], [[MAC-avfp]], [[MAC-imtz]], [[MAC-8yai]], [[MAC-pe9v]], [[MAC-sd7g]], [[MAC-sqpt]], [[MAC-wbxq]], [[MAC-rau8]], [[MAC-u4oo]], [[MAC-5ft8]], [[MAC-al5u]], [[MAC-1u2v]]
- Was blocked by: [[MAC-olrx]], [[MAC-p8ce]], [[MAC-a89e]], [[MAC-2u36]], [[MAC-p7jd]], [[MAC-uzxr]], [[MAC-l7m0]], [[MAC-p9wm]], [[MAC-yig6]], [[MAC-lhu5]], [[MAC-sh60]], [[MAC-hgz1]], [[MAC-lnu6]], [[MAC-wi5z]], [[MAC-qlw2]], [[MAC-cn7q]], [[MAC-6h0s]], [[MAC-hpqp]], [[MAC-2n83]], [[MAC-yhg5]], [[MAC-hlae]]

## Comments

### 2026-09-06T09:15:16Z ramirosalas
FINAL CAPSTONE ASSURANCE EXPANSION 2026-09-06

All original final capstone AC1-AC8 remain required, including saga, coverage/ownership, reads, OCI, OpenCode, installer/recovery, baselines, attestation and release policy behavior. This append-only expansion adds the complete approved assurance contract and blocks this story on EVERY new bounded deliverable. No original dependency is transferred or removed; MAC-sh60 decision/hold is unchanged.

CONSUMES:
- MAC-vx24: docs/test-assurance-integration.md
  source: accepted full original-AC integration map and standalone process guidance.
- MAC-1u2v: cmd/machinery/assurance_standalone_e2e_test.go
  source: accepted fresh first-use native flows for all four languages with exact sensitivity evidence.
- MAC-bz1y: scripts/integration-lane/assurance_catalog.go
  schema: complete closed required-fragment union including original frozen 96-case pilot, supplemental custody, four adapters, checks/replay/Git/finalization/CLI/examples/E2E.
- MAC-rau8: internal/assuranceflow/run.go
  spec: Run(ctx context.Context,req Request,output io.Writer)(Verification,error), actual normal CLI only for capstone tests.

ADDITIONAL REQUIRED E2E/FULL-CLOSURE PROOF
A9. From empty stores/consumer checkouts, EACH Go, TypeScript, Python and Elixir user completes actual init/scaffold/capture/register/RED/GREEN/verify/complete and hooks with no previous PASS receipt and no pvg/nd/toolchain manager. Explicit reviewed current state, bound native assertions and same-frozen safe/unsafe sensitivity must be observed, not inferred from an adapter unit fixture.
A10. Exercise the complete bypass matrix through actual public commands: forged/imported summary/receipt, stale/missing/deleted head/object/plan, wrong expected-head CAS, hidden milestone/owner/test/negative/control, frozen literal/indentation/mode/empty-directory mutation, compilation/setup/import/panic/timeout/skip/xfail/empty/extra failure, unsupported language framework/runtime and current-code mutation. Validate exact causal diagnostic/assertion plus passing controls.
A11. Obtain genuine Linux amd64 and Darwin arm64 native custody/assertion/Git/finalizer proof for the SAME final candidate source and exact runtime/fixture inventories. Local native runs or cross-compilation cannot substitute for the other platform. No unavailable infrastructure waiver. Run actual process-producer call-graph audit plus real cumulative deadline/owner-budget tests, no late launch after barrier and no seal on release/cleanup/publication/output failure.
A12. Tests-only final capstone remains WITHOUT hard-tdd label: do not manufacture missing-feature RED after accepted implementation. Freeze new E2E cases and demonstrate safe/unsafe mutation sensitivity; every missing required feature becomes a blocking reviewed P0 repair, never a weakened test. All exact leaves appear in the required lane; no skip-if-missing, env dormancy, mocks or empty/fabricated/partial completion.
A13. Only after ALL siblings and this capstone are independently accepted does root run the full scripts/preflight.sh on the fully integrated candidate, with provision-before-test ordering, complete original+supplemental fragment union, all complete examples/legacy regressions and positive/negative counts. Any fix after that invalidates affected final proof and requires targeted plus full final rerun. Do not run heavy preflight during earlier stories.
A14. Root's final completion owns local accepted epic-to-main merge and an isolated candidate binary only after final gate; record exact version/source SHA/binary SHA256 and real smoke verification. No GH push or mutation until explicitly coordinated at the user-authorized end; no installed binary/plugins/skills/agents replacement until NIL is done and root obtains installation clearance. This story cannot replace an active installation or erase unrelated user processes/containers.

The capstone retains its existing test/fixture/fragment ownership, no production source ownership. Expanded scenarios reuse accepted focused fixtures when safe but must exercise actual final candidate paths. DIFF BUDGET remains bounded tests-only ~2-8 files/<1500 changed LOC; separately reviewed expansion required if integrating all scenarios cannot fit. No false acceptance of architecture as native implementation evidence.

## nd_contract
status: new

### evidence
- Final capstone now depends on all nineteen new required assurance deliveries, preserving every original prerequisite.
- No implementation, native execution, preflight, merge, build or publication performed by this backlog repair.

### proof
- [ ] Original AC1-AC8: all hardening outcomes and final local closure remain required.
- [ ] A9-A12: complete four-language/two-platform real E2E and adversarial/custody/deadline proof.
- [ ] A13-A14: final full preflight and isolated local candidate closure after independent acceptance.


### 2026-09-06T09:48:05Z ramirosalas
ROUND-1 RULES 1/2 FINAL CONSUMER CLOSURE
Every original and expanded final capstone AC remains required; tests-only ownership/status/label and dependency DAG unchanged.
CONSUMES:
- MAC-5ft8: examples/go-crm/design/attestations.yaml
  schema: substantive independent post-strict-test current judgment and exact generated full-root-v1 evidence, with authored controls finalized first and real stale-before/current-after controls.
- MAC-al5u: scripts/shellcheck-files.txt
  source: both new scripts included in the preserved byte-exact sorted unique corpus.
- MAC-vx24: cmd/machinery/repository_contract_test.go
  source: actual existing contract regression path; historical root-path typo is not an output.
Final user-perspective proof must exercise TLC AND Alloy actual scoped paths, embedded executable assets for each native adapter, complete script inventory, and current CRM judgment after all added tests/controls. An older accepted subject, hash-only refresh, new exclusion, missing legacy regression, unreviewed fixture/golden amendment or design-only downgrade cannot satisfy complete. No implementation/source/judgment refresh authority is added to this tests-only capstone.

## nd_contract
status: new

### evidence
- Independent round1 findings repaired through supported append-only scope/consumer notes; no architecture/source/test/ref/runtime mutation.
- Exact source surfaces verified against accepted epic7e36f3e7ddcf25565d5d4fe60b328df254eee91d; prior Body/status/labels/history preserved.
- Independent Anchor ROUND2 and canonical-document acceptance remain prerequisites, not implementation evidence.

### proof
- [ ] All current story ACs, strengthened ownership/current-judgment requirements and protected frozen proof remain required.
- [ ] Independent review/native execution/final acceptance pending.

### 2026-09-06T12:05:48Z ramirosalas
FINAL CAPSTONE PUBLICATION DEPENDENCY

MAC-ou97 now directly depends on MAC-p9wm, preserving the capstone rule that every epic child is accepted first. Its user-facing final scenarios must find the native-custody supplement through the accepted test-assurance companion link and must distinguish published architecture from implementation/native execution proof.

CONSUMES:
- MAC-p9wm: docs/native-custody-contract.md
  source: accepted standalone public native-custody contract, including exact trust/residual and remaining-proof boundaries.
- MAC-p9wm: docs/test-assurance-contract.md
  source: accepted discoverability/refinement notice with the original contract otherwise preserved.

No production or test ownership is transferred to MAC-p9wm or this capstone. MAC-ou97 remains open/new with all previous blockers and outcomes intact.

## nd_contract
status: new

### evidence
- MAC-p9wm added as a direct final-capstone blocker/consumer.
- No existing dependency removed and no status/label/claim altered.

### proof
- [ ] Final capstone validates public discoverability and truthful architecture-versus-proof claims after all implementations are accepted.
- [ ] All pre-existing MAC-ou97 ACs remain pending.

