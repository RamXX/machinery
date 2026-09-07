---
id: MAC-hwdb
title: "Bound OpenCode governance subprocesses"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd, delivered]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:27Z
created_by: ramirosalas
updated_at: 2026-09-07T08:24:55Z
content_hash: "sha256:92cf47c7c9cfd6f6de3cc59a8d06b529e7c819fe63e4fe88e1b165c760affab0"
blocks: [MAC-gcrr, MAC-ou97]
was_blocked_by: [MAC-hpqp]
follows: [MAC-hpqp, MAC-yhg5]
assignee: dev-MAC-hwdb
---

## Description
## USER INTENT
Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.

## Context (Embedded)
Assessment additional finding: defaultRunner accumulates arbitrary stdout/stderr with no deadline and runMachinery accepts parsed JSON of arbitrary shape. Existing seven adapter tests inject runner functions and do not establish real subprocess behavior.

## Ownership
Own only these paths and directly associated tests: adapters/opencode/plugins/machinery.js, adapters/opencode/plugins/machinery.test.mjs. You are not alone in this codebase; preserve other edits and coordinate any shared-file changes with dispatcher.

## Boundary Map
PRODUCES:
- adapters/opencode/plugins/machinery.js -> hardened behavior and regression proof
- adapters/opencode/plugins/machinery.test.mjs -> hardened behavior and regression proof
CONSUMES:
- Existing Machinery source interfaces.
  spec: function defaultRunner(root, payload); async function runMachinery(run, root, payload)

### Story Acceptance Criteria
1. Default runner applies finite deadline and independent bounded stdout/stderr, terminates owned subprocesses on expiry/overflow, closes resources and settles exactly once across error/close/EPIPE races.
2. Validate allowed hook response shapes and decisions; malformed JSON, unknown/wrong-shape responses, invalid fields, nonzero exit and transport failure block. Preserve documented empty-success and valid host allow/deny protocols.
3. Positive actual machinery hook invocation passes valid governance output. Negative actual child processes hang, flood stdout/stderr, return invalid shapes, exit early or ignore termination; bounded failure is observed without retained children.
4. No shell interpolation; preserve safe argv/stdin construction and correct path handling. Test Node runtime explicitly and compatibility with supported Bun host without claiming unrun environments.
5. Add native-subprocess tests in addition to existing injected unit tests; no skip-if-missing and no mocked runner accepted as integration proof.

## Testing Requirements
- Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
- Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
- Commands: node --test adapters/opencode/plugins/machinery.test.mjs; real process tests use isolated temporary PATH fixtures for adversarial child behavior plus one actual built machinery binary for positive protocol.
- No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.

## OUT OF SCOPE
- Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
- Global heavy preflight, main merge and local release binary belong to final epic gate.

## DIFF BUDGET
- ~2-3 files, under 650 changed LOC; material overrun requires PM investigation, not weakened requirements.

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
ANCHOR ROUND-1 AUTHORITATIVE EXECUTION-LANE REPAIR
General rule: required runtime tests need deterministic provisioning, explicit closed inventory, actual native execution accounting and teardown. Missing infrastructure must fail the REQUIRED lane, not be silently skipped. Ordinary native suites may explicitly exclude registered service-backed tests using a dedicated build tag.
Classification: Actual Node subprocess/real Machinery-hook transport cases need provisioned Node and built isolated binary; no fake runner replaces them.
Node native test run must enumerate actual cases and reject skips/zero execution. Existing runner-injected unit cases remain but are not execution evidence.
PRODUCES:
- testdata/integration-lanes/opencode.json -> this story's closed suite fragment, with exact source/test IDs, runtime/pin requirements and bounded execution configuration
- adapters/opencode/plugins/machinery.test.mjs -> actual named runtime cases registered in the fragment
CONSUMES:
- MAC-hpqp: testdata/integration-lanes/schema.json
  schema: Closed versioned native-runner suite fragment with exact source/test identities, runtime requirements and bounded command selection.
- MAC-hpqp: scripts/integration-lane/main.go
  endpoint: go run ./scripts/integration-lane --lane required (Makefile test-integration invokes same entrypoint).
Additional acceptance criteria: fragment matches actual test sources both directions; all registered cases actually start/terminate with expected positive/negative outcomes; no cached/skipped/empty/partial/fabricated-summary success; real provisioned positive and missing-runtime/fresh-cache failure diagnostics; no owned container/process leaks. Required local preflight and hosted CI execute the same union. Do not edit shared root inventory; own only this fragment. RED source, fixture, fragment and runner configuration are frozen together after review. Any exact test names introduced in RED must remain registered through GREEN.
No heavy preflight until final gate; no GitHub mutation; no active installation replacement. This note supersedes any earlier command implying service-backed tests execute in unprovisioned ordinary package suites.
COMPLETED: RED 7865d3f (15 intended assertion failures/15 controls green, wrapper+fragment opencode.json frozen, lane inventory acceptance proven via pins-stop) -> GREEN eb5cd68 (30/30 node --test adapters/opencode/plugins/machinery.test.mjs, 0 skips; wrapper green; go build ./... ok; TestOpenCodeAdapterContracts ok; internal/install targeted artifact tests ok; scripts/integration-lane ok). defaultRunner: 600s deadline + independent 1MiB stdout/stderr ceilings, SIGTERM->750ms->SIGKILL escalation, stream/stdin destruction, single settle flag across error/close/deadline/overflow/EPIPE; runMachinery validates response shapes (allow/deny/block/systemMessage only, ask+unknown+combined+non-object block), truncation blocks. Native adversarial children (hang, SIGTERM-ignore, floods, early-exit, garbage, wrong-shape) all bounded with ps-proven termination and zero retained children; positive real built binary deny round trip on managed root. Bun 1.0.26 run: module+validation matrix green (bun-run3.txt); Bun bare-name child_process PATH resolution is defective (stale resolution + sync throw) - documented residual, not claimed. Post-RED repair: preexisting fakeRunner now forwards result fields (...rest), re-RED proven against unchanged production (genuine assertion failure), disclosed for PM review. Diff 777 lines/4 files: 633 excl anchor-mandated lane wrapper+fragment (144) vs 650 budget - flagged for PM adjudication. Full record: /tmp/MAC-hwdb-delivery/RECORD.md. IN PROGRESS: PM acceptance. NEXT: independent PM replay.

## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-07.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## History
- 2026-09-05T19:35:09Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:15Z dep_added: blocks MAC-ou97
- 2026-09-05T19:45:32Z dep_added: blocked_by MAC-hpqp
- 2026-09-07T04:45:25Z dep_removed: was_blocked_by MAC-hpqp
- 2026-09-07T07:22:57Z status: open -> in_progress
- 2026-09-07T07:22:57Z auto-follows: linked to predecessor MAC-hpqp
- 2026-09-07T08:19:14Z status: in_progress -> in_progress
- 2026-09-07T08:19:14Z auto-follows: linked to predecessor MAC-yhg5

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-gcrr]], [[MAC-ou97]]
- Was blocked by: [[MAC-hpqp]]
- Follows: [[MAC-hpqp]], [[MAC-yhg5]]

## Comments

### 2026-09-07T08:24:55Z ramirosalas
ACCEPTED 2026-09-06 — RED 7865d3f (30 cases: 15 intended failures incl. adversarial native children; 15 controls) -> GREEN eb5cd68. defaultRunner: 600s deadline, independent 1MiB stdout/stderr ceilings, SIGTERM->SIGKILL escalation, single settle flag across error/close/EPIPE races; runMachinery validates hook response shapes (empty-success/allow/deny/block/systemMessage only; malformed/unknown/nonzero/transport block). ps-proof: adversarial child alive mid-run, gone after. Real built binary deny round-trip. Node 30/30 0 skips; Bun 1.0.26 green (bare-name spawn resolution defect documented as host residual, not claimed). One post-RED fakeRunner passthrough repair with re-RED proof (disclosed, reviewed here: infrastructure repair, no expectation weakened). 777 lines vs 650 nominal (633 excl. anchor-mandated lane artifacts) — within disclosed-overrun pattern. opencode lane fragment validated; lane 7 suites green on epic. Record: .git/machinery-evidence-20260906.TEFZ7D/hwdb-record.md
