---
id: MAC-uzxr
title: "Bind Go CRM FSM conformance to committed oracle expectations"
status: open
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T03:01:38Z
created_by: ramirosalas
updated_at: 2026-09-06T03:01:38Z
content_hash: "sha256:e87667873e9c14a761bf7f69895d033b445557e8cdfeac3c6b1c738be5241ae1"
blocks: [MAC-ou97]
---

## Description
## USER INTENT
A current example conformance claim must describe real tests bound to committed FSM expectations, not handwritten expectations that merely reuse stable row names.

## Context (Embedded)
Confirmed absent wholesale FSM parser in Go CRM's five transition suites. PM's actual replay passed 218 leaves (domain130/session60/cli28), zero failures/skips, with all197 committed IDs present in executed names. This proves bounded existing transition execution, not linkage to parsed current oracle expectations. Raw /tmp/MAC-lnu6-pm-go-crm-transitions-713184d.jsonl SHA256 8a88436b9d6212a45eb422c4f4d609902679a15343ba796b1c72174aff03a662; wall1.503486s.
internal/gates/attest.go:151 requires wholesale committed oracle parsing plus per-row next state AND expected actions. Five handwritten test tables call real Fire/State but do not parse the committed FSM oracle tables. Existing firedInOrder loops accept expected actions as a subsequence and any actual list for an empty expectation. Source establishes that adequacy gap; no extra-effect mutant was executed by the discovery review. Registry wording does not expressly require exact-list equality. Entry/exit effects are legitimate and MUST be reconciled from committed semantics rather than blindly replacing containment with transition-column equality.

## Ownership
Seven forecast paths:
- examples/go-crm/impl/internal/domain/deal_test.go
- examples/go-crm/impl/internal/domain/task_test.go
- examples/go-crm/impl/internal/domain/user_test.go
- examples/go-crm/impl/internal/session/machine_test.go
- examples/go-crm/impl/internal/cli/command_test.go
- examples/go-crm/impl/internal/testoracle/fsm.go (new test-support parser/reconciliation only)
- examples/go-crm/impl/internal/testoracle/fsm_test.go (new parser and real subprocess sensitivity proof)
Only exact independently PM-authorized existing test/helper amendments may change the five old files. Preserve their real inputs/guard branch cases, terminal supplements and safety intentions. New test-support paths are reviewed outputs, not assumed existing APIs; concrete helper shape/use must be reviewed before authoring if it affects frozen assertions.
Read-only: five committed design/machines/{Deal,Task,User,Session,CommandExecution}.oracle.md and .machine.json files; production Fire implementations; root Machinery parser/gate/attestation code; all BUILD/attestations/acceptance/go.mod/go.sum. No generated-oracle edits or fixture weakening to make tests pass. Unexpected production mismatch returns to Sr PM; no blanket implementation-fix ownership. You are not alone; preserve other work.

## Boundary Map
PRODUCES:
- Five exact transition test files -> actual Fire/State conformance driven by parsed committed row expectations with complete row/guard input reconciliation.
- internal/testoracle/fsm.go under the Go CRM impl -> bounded test-support parser and expectation reconciliation, not a new Machinery production API.
- internal/testoracle/fsm_test.go under the Go CRM impl -> matched real native controls and adversarial parser/assertion sensitivity.
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
Focused existing control from examples/go-crm/impl: go test -count=1 -timeout=120s ./internal/domain ./internal/session ./internal/cli -run '^(TestDealTransitions|TestTaskTransitions|TestUserTransitions|TestSessionTransitions|TestCommandExecutionTransitions)$' -json
Run new internal/testoracle tests explicitly plus relevant existing terminal supplements. Bound outer meta-tests and avoid recursive selection of themselves. Propose exact new names/commands and mutation hunks for independent RED review; freeze unchanged-control and unsafe variants separately with SHA/input hashes. No hidden skip, generated expected-output replacement, or acceptance based on one error. Record host/toolchain, exact source/reference/mutant SHAs, commands, leaf/row inventories, logs and durations.

## OUT OF SCOPE
- Core plan/current/historical schema and freshness custody belong to MAC-p7jd; future authenticated runner/replay enforcement belongs to MAC-l7m0/MAC-vx24.
- BUILD/attestation consumer migration is a separately sequenced repair; no simultaneous ownership of MAC-lnu6 files.
- New application behavior, root parser/generator edits, external services and global preflight.

## DIFF BUDGET
- Seven paths, forecast700-1100 changed LOC, largely explicit row/guard reconciliation and paired subprocess tests. Report actual per-file cost; independent review required for material growth, never trim safety witnesses.
- Existing focused baseline1.503486s; new real copied-tree builds/mutations cost more. Initial outer proof budget5m with bounded individual120s child suites; report measured runtime rather than assuming this fit.

## Discovered During
MAC-lnu6 independent substantive amendment review at 713184db16a12b8c3763b4aa8f5bf025721abf22. Full report /tmp/MAC-lnu6-PM-AMENDMENT-713184d.md SHA256 6fc5223d3e5d48170f89ff9e29b634b256bfe2454b710378f08cb60d875aadf1 was read completely (227 newline-terminated lines; final content included). No runtime mutation proof beyond the explicitly reported native replay is claimed. Exact epic 70652b948 source inspected; graph generation 2026-09-06T02:42:16Z is best effort, not completeness proof.

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


## History
- 2026-09-06T03:03:05Z dep_added: blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-ou97]]

## Comments
