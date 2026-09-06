---
id: MAC-uzxr
title: "Bind Go CRM FSM conformance to committed oracle expectations"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T03:01:38Z
created_by: ramirosalas
updated_at: 2026-09-06T03:34:05Z
content_hash: "sha256:55a3a5f31fc8d6866b5a04e7a4252aaaf4d1b3709d92968cb1610e6b242c8545"
blocks: [MAC-ou97, MAC-hgz1]
assignee: dev-MAC-uzxr
follows: [MAC-2u36]
---

## Description
## USER INTENT
A current example conformance claim must describe real tests bound to committed FSM expectations, not handwritten expectations that merely reuse stable row names.

## Context (Embedded)
Confirmed absent wholesale FSM parser in Go CRM's five transition suites. PM's actual replay passed 218 leaves (domain130/session60/cli28), zero failures/skips, with all197 committed IDs present in executed names. This proves bounded existing transition execution, not linkage to parsed current oracle expectations. Raw /tmp/MAC-lnu6-pm-go-crm-transitions-713184d.jsonl SHA256 8a88436b9d6212a45eb422c4f4d609902679a15343ba796b1c72174aff03a662; wall1.503486s.
internal/gates/attest.go:151 requires wholesale committed oracle parsing plus per-row next state AND expected actions. Five handwritten test tables call real Fire/State but do not parse the committed FSM oracle tables. Existing firedInOrder loops accept expected actions as a subsequence and any actual list for an empty expectation. Source establishes that adequacy gap; no extra-effect mutant was executed by the discovery review. Registry wording does not expressly require exact-list equality. Entry/exit effects are legitimate and MUST be reconciled from committed semantics rather than blindly replacing containment with transition-column equality.

## Confirmed Task entry-action defect and bounded GREEN repair
Runtime-confirmed on unchanged70652b948 production while planning AC2 reconciliation: TASK-754183 (rolledBack, always, fallback) reaches Cancelled but returns only recordRoutingError; committed Task.oracle.md and Task.machine.json require Cancelled entry recordTaskClosed after the transition action. Matched TASK-67b0ff normal persist-to-Cancelled returns commitStatus,recordTaskClosed and passes. Actual isolated probe command from /tmp/MAC-uzxr-task-entry-probe.FGj02Q/impl: go test -count=1 -timeout=120s ./internal/domain -run '^TestTaskOracleEntryProbe$' -json. Author's corrected pipefail execution exited1:1PASS/1FAIL/0SKIP,package0.286s. Initial tee pipeline status0 was only the pipeline wrapper status, not test success.
Verified raw probe-pipefail.jsonl SHA25690e3d11c3df38725d77637bc187d7e27dcf0a2b65f0934e6b71c785b177e783c; isolated task_entry_probe_test.go SHA256f6809da70ce0130f79020f8d032194c878e3b17acdeebfaa12f22ca54bc97e84; copied and committed task.go SHA25639af26a093f855acf269da153915fbecf6e78c0f3dbba85b087bf4b83586e35b. Task oracle SHA25670d3f0669117c85aef8c3a6e061df147f43c6ced036805c24eb200cfe36e0d56; machine SHA256ba243728cf359b02cc3d7cb0691eea2bd2b0a90a49c11a46846b820ac250d9b5.
Additional isolated copied-test reconnaissance tightened length/order and hand-reconciled Task/CommandExecution entry expectations across five transition suites plus TestTaskTerminalRejectsEverything and TestCommandExecutionTerminalExits:225 leaves224PASS/1FAIL TASK-754183/0SKIP,author-reported outer0.917s,actual exit1. Raw exact-effects.jsonl SHA256c6ce626ee381f97d020a2bea090d1a626e9b9adfa421c75ffd877cdfb8ee42a4. Its four copied-test diffs and raw outcomes were read; these handwritten diagnostic probes are NOT committed parser-backed tests, approved RED or completeness proof.
Same-P0 scope disposition: this tiny application mismatch directly blocks the story's required complete-effect conformance; repair it here rather than creating a duplicate owner/prerequisite. Only Task.fireRolledBack's fallback may change in production: retain recordRoutingError and Cancelled routing, then execute existing recordTaskClosed and report both actions in that exact order. Existing recordTaskClosed clears Rejection; returning its name without performing that real context mutation is not a fix. Do not change guard priority, valid prior routing, event filtering, final-state behavior, action definitions, public interfaces or other production functions.
Bounded exact-source inspection: Deal/User/Session declare no entry/exit actions; Task declares entry only for Done/Cancelled and normal persist terminal routes already call it; CommandExecution declares eight entry actions and its target transitions use existing enter handling. This establishes no additional missing-entry variant in that bounded review, not absence of all possible defects. Graph generation02:42:16Z reported metadata_match/no recorded gap best effort; receiver-name collisions required exact70652b948 source fallback.
No production edits are authorized during RED. Independent exact old-test/helper/reconciliation review remains required BEFORE those edits; only after genuine RED delivery and approval may the separate GREEN implementer make this narrow task.go repair. Any further concrete mismatch returns to Sr PM before expanding production scope. MAC-hgz1 and MAC-ou97 already depend on this story, so no dependency change; consumer uses the accepted parser tests AND this repaired Task source. MAC-lnu6 evidence hold remains unchanged.

## Ownership
Eight forecast paths (the prior seven test-support paths plus one narrowly limited production file):
- examples/go-crm/impl/internal/domain/deal_test.go
- examples/go-crm/impl/internal/domain/task_test.go
- examples/go-crm/impl/internal/domain/user_test.go
- examples/go-crm/impl/internal/session/machine_test.go
- examples/go-crm/impl/internal/cli/command_test.go
- examples/go-crm/impl/internal/testoracle/fsm.go (new test-support parser/reconciliation only)
- examples/go-crm/impl/internal/testoracle/fsm_test.go (new parser and real subprocess sensitivity proof)
- examples/go-crm/impl/internal/domain/task.go (GREEN only, fireRolledBack fallback TASK-754183 entry-action call and ordered Effect.Actions; no other production edits)
Only exact independently PM-authorized existing test/helper amendments may change the five old files. Preserve their real inputs/guard branch cases, terminal supplements and safety intentions. New test-support paths are reviewed outputs, not assumed existing APIs; concrete helper shape/use must be reviewed before authoring if it affects frozen assertions.
Read-only: five committed design/machines/{Deal,Task,User,Session,CommandExecution}.oracle.md and .machine.json files; production Fire implementations except the exact task.go fallback extension above; root Machinery parser/gate/attestation code; all BUILD/attestations/acceptance/go.mod/go.sum. No generated-oracle edits or fixture weakening to make tests pass. Unexpected production mismatch returns to Sr PM; no blanket implementation-fix ownership. You are not alone; preserve other work.

## Boundary Map
PRODUCES:
- Five exact transition test files -> actual Fire/State conformance driven by parsed committed row expectations with complete row/guard input reconciliation.
- internal/testoracle/fsm.go under the Go CRM impl -> bounded test-support parser and expectation reconciliation, not a new Machinery production API.
- internal/testoracle/fsm_test.go under the Go CRM impl -> matched real native controls and adversarial parser/assertion sensitivity.
- examples/go-crm/impl/internal/domain/task.go -> existing func (t *Task) Fire(evt TaskEvent) Effect delegates to func (t *Task) fireRolledBack(evt TaskEvent) Effect; corrected TASK-754183 preserves Cancelled routing and executes existing func (t *Task) recordTaskClosed() after recordRoutingError, with actual context change and ordered Effect.Actions.
CONSUMES:
- Existing Go CRM transition interfaces.
  source: TestDealTransitions/TestTaskTransitions/TestUserTransitions/TestSessionTransitions/TestCommandExecutionTransitions instantiate real domain/session/CLI machines, call Fire on concrete events and inspect State and Effect.Actions.
- Existing committed oracle grammar and semantics.
  source: each .oracle.md contains State entry / exit actions plus Transitions columns test id, stable id, source, trigger, guard, target, actions; corresponding .machine.json defines ordered guarded alternatives, internal/external transitions and entry/exit actions. Parse these actual committed sources, not matrix citations only.
- Existing claim vocabulary.
  source: internal/gates/attest.go gt.conformance-test-shape requires wholesale parser, next state and expected actions; no core claim-model edit here.

### Story Acceptance Criteria
1. All five FSM conformance suites parse the committed oracle tables during real execution and derive expected next state/actions from them, with an explicit closed mapping from each current row/guard branch to actual setup/event inputs. Check source/trigger/guard/target identity, not only stable-ID string membership. Missing, duplicate, malformed, newly added unmapped or unused rows fail with named diagnostics; preserve complete existing guard-clause coverage and additional legitimate supplement cases.
2. Reconcile full observable action expectations using the actual committed transition AND entry/exit semantics, including internal transitions and fallback/guard priority. Independently review the concrete reconciliation before freezing revised assertions. Correct legitimate entry/exit actions pass; missing, reordered, duplicated or unrelated extra effects fail, including an empty-transition-action row. Do not claim this exactness was an explicit old registry quotation; it is required adequacy proof for this repaired current claim.
3. Real native subprocess sensitivity on isolated copied source/oracle fixtures demonstrates that changing committed expected next state or actions without changing implementation causes the intended conformance failure, not parse/setup failure. Separately inject a reviewed extra-effect implementation variant and require the same frozen assertions to reject it, with unchanged implementation/oracle controls passing. No returned fake process result, warning-only negative, modified shared oracle or production fault API.
4. Report bidirectional committed-row versus actual executed-row inventory for all five machines; current baseline197 IDs/218 leaves is historical, not a hardcoded ceiling. All selected rows/guard witnesses terminate with expected real outcomes and zero skips; inspect real next-state/action diagnostics and preserve terminal supplements. A passing parser unit test or named-ID count alone is insufficient.
5. The actual focused native Go CRM suite and parser/sensitivity proof pass on the delivered candidate, old unrelated tests remain unchanged, and the consumer migration can use the exact reviewed source/test scope and logs to assess a current claim. No automatic attestation, implementation acceptance renewal, full-suite or authenticated-execution assertion is made by this story.

## Testing Requirements
Unit tests plus Integration tests: MANDATORY (no mocks of Fire, compiler, filesystem, process output or assertion result). Service-free native Go and real temporary copies/subprocesses; no Docker or Paivot dependency. Review exact old-test amendments before RED writes; test-support scaffolding may support a qualifying RED only when the unchanged old suites demonstrate the specific missing parser/sensitivity behavior. Compile/API/fixture/timeout failures are not RED.
AC2 mandatory regression details for the confirmed fallback: revised task_test.go coverage must derive expected actions from the committed transition plus entry semantics, assert Cancelled and exactly [recordRoutingError,recordTaskClosed], and assert actual Rejection clearing. Matched TASK-67b0ff control asserts [commitStatus,recordTaskClosed] plus real entry behavior; retain TASK-c56bd7 Done, TASK-3f585f/ TASK-98c3ba valid prior branches, non-always no-transition and terminal-event controls. No new standalone test file or production fault seam is needed. The exact assertion/setup changes in the already owned task_test.go must appear in the independent PM amendment proposal before authoring.
Keep evidence classes separate: the discovered fallback assertion must genuinely fail unchanged70652b948 for missing entry behavior; it is NOT proof that the absent parser is detected. Original AC1/AC3 still require independent intended parser/oracle-mutation/extra-effect RED with matched valid controls; choose unaffected scoped controls or classify the known fallback failure explicitly, never relabel an already-broken full control as passing. GREEN must pass the complete frozen suite on the candidate and reject a real isolated regression that removes the entry call or only fabricates its returned name; reuse bounded sensitivity infrastructure, with precise reviewed mutation and no shared-source mutation. This extends the existing missing-effect adequacy proof, not permission to alter frozen assertions after production implementation.
Focused existing control from examples/go-crm/impl: go test -count=1 -timeout=120s ./internal/domain ./internal/session ./internal/cli -run '^(TestDealTransitions|TestTaskTransitions|TestUserTransitions|TestSessionTransitions|TestCommandExecutionTransitions)$' -json
Run new internal/testoracle tests explicitly plus relevant existing terminal supplements. Bound outer meta-tests and avoid recursive selection of themselves. Propose exact new names/commands and mutation hunks for independent RED review; freeze unchanged-control and unsafe variants separately with SHA/input hashes. No hidden skip, generated expected-output replacement, or acceptance based on one error. Record host/toolchain, exact source/reference/mutant SHAs, commands, leaf/row inventories, logs and durations.

## OUT OF SCOPE
- Core plan/current/historical schema and freshness custody belong to MAC-p7jd; future authenticated runner/replay enforcement belongs to MAC-l7m0/MAC-vx24.
- BUILD/attestation consumer migration is a separately sequenced repair; no simultaneous ownership of MAC-lnu6 files.
- New application behavior beyond the exact committed TASK-754183 entry-action correction, root parser/generator edits, external services and global preflight.

## DIFF BUDGET
- Eight paths, revised forecast750-1150 changed LOC, superseding seven/700-1100: approximately30-50 additional lines for explicit context-effect regression/sensitivity plus a minimal existing-function production correction, with rounded planning allowance. This is not a new production layer/API or automatic cap/permission. Report actual per-file additions/deletions and runtime; independent review required for material growth, never trim original parser/guard/action safety witnesses.
- Existing focused baseline1.503486s; new real copied-tree builds/mutations cost more. Initial outer proof budget5m with bounded individual120s child suites; report measured runtime rather than assuming this fit.

## Discovered During
MAC-lnu6 independent substantive amendment review at 713184db16a12b8c3763b4aa8f5bf025721abf22. Full report /tmp/MAC-lnu6-PM-AMENDMENT-713184d.md SHA256 6fc5223d3e5d48170f89ff9e29b634b256bfe2454b710378f08cb60d875aadf1 was read completely (227 newline-terminated lines; final content included). The original review's no-runtime-mutation limit remains historical; the later isolated Task and exact-effects probes above are now explicitly reported but do not establish approved parser RED. Exact epic 70652b948 source inspected; graph generation 2026-09-06T02:42:16Z is best effort, not completeness proof.

## MANDATORY SKILLS
- developer; codebase-memory; pm_acceptor.

## Delivery Requirements
Hard TDD with independent RED and exact existing-test amendment review before edits. Preserve all unrelated tests, goldens, generated evidence and existing claims. Shared tracker is development coordination only; product remains standalone. No remote, full preflight, installed assets, user services or healthy-worktree cleanup. No source/docs/tests changed during triage. Use supported delivery; independent PM accepts.

## nd_contract
status: new

### evidence
- P0 source-established conformance parser defect plus separately identified action-adequacy gap; parent supplied independently verified native218-pass control. No new mutation run or repair performed.
- Existing claims/tests/ownership remain held until independent exact test amendment review; no core schema duplication.

### proof
- [ ] AC #1: actual committed-row parser and closed input/guard reconciliation.
- [ ] AC #2: correct entry/exit-aware complete effects and adversarial sensitivity.
- [ ] AC #3: real expected-state/action and extra-effect unsafe variants fail for intended causes.
- [ ] AC #4: complete executed-row inventory and preserved controls.
- [ ] AC #5: native candidate proof without automatic claim renewal.

## Acceptance Criteria


## Design


## Notes
## nd_contract
status: new

### evidence
- Created P0 under MAC-ui8a; open/unclaimed/hard-tdd. MAC-ou97 explicitly depends on this repair. No existing state/claim/test/source changes.
- Actual canonical readback confirms five AC, explicit nonoverlapping ownership and intact quoted command. Scoped lint PASS36issues/0errors/0review; dependency cycles none; RTM0extracted requirements/21stories/4closed is structural only, not AC proof.
- Independent test/handoff review and exact existing-test amendment authorization remain pending; no RED approval or implementation permission granted by triage.
- P0 source-established conformance parser defect plus separately identified action-adequacy gap; parent supplied independently verified native218-pass control. No new mutation run or repair performed.
- Existing claims/tests/ownership remain held until independent exact test amendment review; no core schema duplication.

### proof
- [ ] AC #1: actual committed-row parser and closed input/guard reconciliation.
- [ ] AC #2: correct entry/exit-aware complete effects and adversarial sensitivity.
- [ ] AC #3: real expected-state/action and extra-effect unsafe variants fail for intended causes.
- [ ] AC #4: complete executed-row inventory and preserved controls.
- [ ] AC #5: native candidate proof without automatic claim renewal.


## History
- 2026-09-06T03:03:05Z dep_added: blocks MAC-ou97
- 2026-09-06T03:17:08Z dep_added: blocks MAC-hgz1
- 2026-09-06T03:25:44Z status: open -> in_progress
- 2026-09-06T03:25:44Z auto-follows: linked to predecessor MAC-2u36
- 2026-09-06T03:25:44Z claimed by dev-MAC-uzxr

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-ou97]], [[MAC-hgz1]]
- Follows: [[MAC-2u36]]

## Comments

### 2026-09-06T03:05:27Z ramirosalas
## nd_contract
status: new

### evidence
- Created P0 under MAC-ui8a; open/unclaimed/hard-tdd. MAC-ou97 explicitly depends on this repair. No existing state/claim/test/source changes.
- Actual canonical readback confirms five AC, explicit nonoverlapping ownership and intact quoted command. Scoped lint PASS36issues/0errors/0review; dependency cycles none; RTM0extracted requirements/21stories/4closed is structural only, not AC proof.
- Independent test/handoff review and exact existing-test amendment authorization remain pending; no RED approval or implementation permission granted by triage.
- P0 source-established conformance parser defect plus separately identified action-adequacy gap; parent supplied independently verified native218-pass control. No new mutation run or repair performed.
- Existing claims/tests/ownership remain held until independent exact test amendment review; no core schema duplication.

### proof
- [ ] AC #1: actual committed-row parser and closed input/guard reconciliation.
- [ ] AC #2: correct entry/exit-aware complete effects and adversarial sensitivity.
- [ ] AC #3: real expected-state/action and extra-effect unsafe variants fail for intended causes.
- [ ] AC #4: complete executed-row inventory and preserved controls.
- [ ] AC #5: native candidate proof without automatic claim renewal.

