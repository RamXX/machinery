---
id: MAC-vx24
title: "Enforce replayable negative test assurance"
status: open
priority: 1
type: feature
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:35:06Z
created_by: ramirosalas
updated_at: 2026-09-05T19:45:32Z
content_hash: "sha256:5ef3262dd94a9639668333d69de0f0fcd371bf1afc1f417805e954a1055fd973"
blocked_by: [MAC-l7m0, MAC-sh60, MAC-2n83, MAC-lnu6, MAC-hpqp, MAC-qlw2, MAC-cn7q, MAC-6h0s, MAC-p9z1, MAC-62s6, MAC-bz1y, MAC-wi2u, MAC-avfp, MAC-imtz, MAC-8yai]
blocks: [MAC-gcrr, MAC-ou97]
was_blocked_by: [MAC-olrx, MAC-p7jd]
---

## Description
## USER INTENT
Provide strongest honest standalone Machinery guarantees for LLM-generated software, including test sensitivity to unsafe behavior.

## Context (Embedded)
BLOCKED ARCHITECTURE CONTRACT: do not implement until architecture story is accepted and exact interfaces/fields are copied here. User explicitly requests standalone Machinery-owned deterministic hard-TDD negative testing across its consumer process. Paivot is local development coordination only, never a runtime dependency. Existing schema-free BUILD prose and test filename keywords cannot satisfy this feature.

## Ownership
internal/tdd/manifest.go, internal/tdd/runner.go, internal/tdd/replay.go, internal/tdd/evidence.go, internal/tdd/assurance_test.go, cmd/machinery/tdd.go, cmd/machinery/tdd_test.go, cmd/machinery/main.go, internal/gates/tdd.go, internal/gates/suite.go, cmd/machinery/check.go, internal/hook/hook.go, agents/machinery-build-writer.md, skills/machinery/SKILL.md, skills/machinery/references/build-md-template.md. Not alone in codebase; preserve other edits. Newly listed files are explicitly owned outputs, not pre-existing interfaces.

## Boundary Map
PRODUCES:
- internal/tdd/manifest.go -> standalone enforced test-assurance contract
- internal/tdd/runner.go -> standalone enforced test-assurance contract
- internal/tdd/replay.go -> standalone enforced test-assurance contract
- internal/tdd/evidence.go -> standalone enforced test-assurance contract
- internal/tdd/assurance_test.go -> standalone enforced test-assurance contract
- cmd/machinery/tdd.go -> standalone enforced test-assurance contract
- cmd/machinery/tdd_test.go -> standalone enforced test-assurance contract
- cmd/machinery/main.go -> standalone enforced test-assurance contract
- internal/gates/tdd.go -> standalone enforced test-assurance contract
- internal/gates/suite.go -> standalone enforced test-assurance contract
- cmd/machinery/check.go -> standalone enforced test-assurance contract
- internal/hook/hook.go -> standalone enforced test-assurance contract
- agents/machinery-build-writer.md -> standalone enforced test-assurance contract
- skills/machinery/SKILL.md -> standalone enforced test-assurance contract
- skills/machinery/references/build-md-template.md -> standalone enforced test-assurance contract
CONSUMES:
- MAC-l7m0: docs/test-assurance-contract.md
  schema: Approved closed native-runner replay contract; exact names must be copied after independent architecture acceptance before implementation.
- MAC-sh60: internal/gates/oraclecov.go
  spec: gates.CheckOracleCoverage(design, impl string) *Gate
- MAC-olrx: internal/gates/suite.go
  spec: gates.Select(design, gateList, impl string) (Selection, error)
- MAC-2n83: cmd/machinery/main.go
  source: newRootCmd() *cobra.Command command registration
- MAC-p7jd: docs/attestation-evidence.md
  source: Current implementation subject binding and historical acceptance distinction

### Story Acceptance Criteria
1. Implement the independently accepted standalone test-assurance contract using Machinery-owned native runner execution/replay. No self-reported receipt import or generic command exit code can establish execution or assertion causality.
2. RED proves missing behavior with bound expected assertion failures and a passing harness control; required negative cases additionally distinguish safe control from approved unsafe implementation challenge using the same exact frozen test closure. Negative cases may legitimately already pass ordinary RED safe stubs.
3. GREEN reruns the exact declared tests and frozen dependency/config/fixture closure, with all required tests selected/executed/pass. Any missing, duplicate, empty, skipped, xfail, cached, unexpected failure, compilation/panic/timeout or malformed/truncated evidence blocks rather than looking green.
4. Strong final complete assurance independently replays retained RED/challenges and current GREEN through trusted runner. Changed current implementation/design/test inventory, removed contract/evidence, narrowed test selection, helper/config changes or forged receipt cannot evade enforcement.
5. Wire standalone CLI, default/staged host governance and strong complete handoff consistently. Cheap freshness checks explicitly do not claim replay occurred. Unsupported adapters fail clearly under approved compatibility policy and never receive the strongest label.
6. Remove BUILD template authorization for frozen test formatting via tokens-equal; exact byte/inventory identity is required. Tests prove spaces inside string literals and indentation changes invalidate frozen evidence even when strings.Fields normalization matches.
7. Migrate shipped consumer examples under the accepted compatibility contract and demonstrate actual generated-software flow. Named owned runtime residual obligations connect race/replay/migration/restore/load requirements to evidence rather than prose alone; unsupported guarantees remain explicit.
8. Positive full standalone CLI flow requires no pvg/nd installed or metadata. Negative full-path cases exercise every bypass through actual runner/hooks/complete command. No mocks, stubs or skip-if-missing. Preserve installed Machinery used in NIL.

## Testing Requirements
New focused internal/tdd and cmd/machinery TDD tests; actual native runner subprocesses on temp git snapshots; actual hook and check --complete integration. Separate RED author freezes tests; independent PM replays expected failures and GREEN/challenge results. Full preflight only final gate.
Integration tests: MANDATORY (no mocks) for executable implementation, no skip-if-missing. Architecture review is not execution proof.
No push/sync/GitHub mutation. No installed binary/skills/plugins/agents replacement or dev-link. Isolated candidate only. Full scripts/preflight.sh only final epic gate.

## OUT OF SCOPE
- Mathematical proof of arbitrary test semantic adequacy or malicious host safety beyond explicitly approved trust boundary: expose these as residual limits, never silently claim them.
- Any Paivot runtime dependency: prohibited by user.

## DIFF BUDGET
- ~15-24 files, initial ceiling 3000 changed LOC; MUST split execution/replay and consumer wiring after architecture if this cannot remain reviewable.

## MANDATORY SKILLS
- architect for contract authoring/review; developer and pm_acceptor for implementation; codebase-memory for source discovery.

## nd_contract
status: new

### evidence
- Created 2026-09-05 from user requirements and independent read-only design analysis.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified
- [ ] AC #6: independently verified
- [ ] AC #7: independently verified
- [ ] AC #8: independently verified

Observable outcome: the user can run standalone Machinery and receive an explicit pass or blocking diagnostic based on actual replay.

## Acceptance Criteria


## Design


## Notes
CONSUMES:
- MAC-lnu6: skills/machinery/references/build-md-template.md
  source: Exact-byte frozen-test guidance replacing unsafe tokens-equal authorization; process integration modifies this template sequentially.
ANCHOR ROUND-1 AUTHORITATIVE EXECUTION-LANE REPAIR
General rule: required runtime tests need deterministic provisioning, explicit closed inventory, actual native execution accounting and teardown. Missing infrastructure must fail the REQUIRED lane, not be silently skipped. Ordinary native suites may explicitly exclude registered service-backed tests using a dedicated build tag.
Classification: Native-runner replay/hook/complete E2E requires approved adapter runtimes; exact supported runtime contract remains architecture-blocked.
Record standalone replay suite in required lane even if initial adapter only Go. If approved contract needs additional runtimes, provision in lane before execution. No fallback to receipt-only validation or env-gated omission.
PRODUCES:
- testdata/integration-lanes/tdd.json -> this story's closed suite fragment, with exact source/test IDs, runtime/pin requirements and bounded execution configuration
- cmd/machinery/tdd_test.go -> actual named runtime cases registered in the fragment
CONSUMES:
- MAC-hpqp: testdata/integration-lanes/schema.json
  schema: Closed versioned native-runner suite fragment with exact source/test identities, runtime requirements and bounded command selection.
- MAC-hpqp: scripts/integration-lane/main.go
  endpoint: go run ./scripts/integration-lane --lane required (Makefile test-integration invokes same entrypoint).
Additional acceptance criteria: fragment matches actual test sources both directions; all registered cases actually start/terminate with expected positive/negative outcomes; no cached/skipped/empty/partial/fabricated-summary success; real provisioned positive and missing-runtime/fresh-cache failure diagnostics; no owned container/process leaks. Required local preflight and hosted CI execute the same union. Do not edit shared root inventory; own only this fragment. RED source, fixture, fragment and runner configuration are frozen together after review. Any exact test names introduced in RED must remain registered through GREEN.
No heavy preflight until final gate; no GitHub mutation; no active installation replacement. This note supersedes any earlier command implying service-backed tests execute in unprovisioned ordinary package suites.

## History
- 2026-09-05T19:35:06Z dep_added: blocked_by MAC-l7m0
- 2026-09-05T19:35:07Z dep_added: blocked_by MAC-sh60
- 2026-09-05T19:35:07Z dep_added: blocked_by MAC-olrx
- 2026-09-05T19:35:07Z dep_added: blocked_by MAC-2n83
- 2026-09-05T19:35:07Z dep_added: blocked_by MAC-p7jd
- 2026-09-05T19:35:07Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:14Z dep_added: blocked_by MAC-lnu6
- 2026-09-05T19:36:17Z dep_added: blocks MAC-ou97
- 2026-09-05T19:45:32Z dep_added: blocked_by MAC-hpqp
- 2026-09-05T20:27:03Z dep_removed: was_blocked_by MAC-olrx
- 2026-09-06T07:47:13Z dep_removed: was_blocked_by MAC-p7jd
- 2026-09-06T09:10:14Z dep_added: blocked_by MAC-qlw2
- 2026-09-06T09:10:15Z dep_added: blocked_by MAC-cn7q
- 2026-09-06T09:10:16Z dep_added: blocked_by MAC-6h0s
- 2026-09-06T09:10:17Z dep_added: blocked_by MAC-p9z1
- 2026-09-06T09:10:18Z dep_added: blocked_by MAC-62s6
- 2026-09-06T09:10:19Z dep_added: blocked_by MAC-bz1y
- 2026-09-06T09:10:20Z dep_added: blocked_by MAC-wi2u
- 2026-09-06T09:10:21Z dep_added: blocked_by MAC-avfp
- 2026-09-06T09:10:22Z dep_added: blocked_by MAC-imtz
- 2026-09-06T09:10:23Z dep_added: blocked_by MAC-8yai

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-gcrr]], [[MAC-ou97]]
- Blocked by: [[MAC-l7m0]], [[MAC-sh60]], [[MAC-2n83]], [[MAC-lnu6]], [[MAC-hpqp]], [[MAC-qlw2]], [[MAC-cn7q]], [[MAC-6h0s]], [[MAC-p9z1]], [[MAC-62s6]], [[MAC-bz1y]], [[MAC-wi2u]], [[MAC-avfp]], [[MAC-imtz]], [[MAC-8yai]]
- Was blocked by: [[MAC-olrx]], [[MAC-p7jd]]

## Comments
