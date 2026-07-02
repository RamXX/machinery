# DECISIONS

Every design decision made without asking the product owner, one line each:
the question, the answer, why.

## Process

- Q: Spawn machinery-fsm-author / machinery-build-writer subagents, or perform their roles inline?
  A: Perform inline as the conductor, following the two agent role documents verbatim.
  Why: the formal reconcilers and byte-exact oracle demand tight, in-loop control against the
  tools; the clean-room rule also forbids subagents from reading examples/, so an inline pass
  is both safer and cleaner. Every role-document step is still executed.
- Q: Ask the product owner for uncovered details, or decide?
  A: Decide pragmatically and log here.
  Why: the seed answers are comprehensive and this run is meant to execute end to end.

## Phase 0 - Frame

- Q: Target language? A: Go (single self-contained binary), per the seed.
- Q: Datastore? A: Embedded LadybugDB via github.com/LadybugDB/go-ladybug; no server, no network.

## Phase 1 - Domain model

- Q: Deal pipeline stage names and order? A: Prospecting -> Qualification -> Proposal ->
  Negotiation, then terminal Won / Lost. Why: "first contact through negotiation, ends won or
  lost"; four ordered open stages with negotiation last matches the seed and the reopen target.
- Q: Where does reopen return a deal? A: Negotiation. Why: the seed says "returns to negotiation".
- Q: Which stage records the close date? A: Won. Why: "a won deal must record when it closed."
  (Lost also stamps closedAt for symmetry, but only the Won rule is an invariant.)
- Q: Model user active/inactive as an enum lifecycle or a boolean? A: Boolean attribute `active`.
  Why: there is no multi-state user lifecycle in the seed (no reactivation workflow described as a
  lifecycle), and a status/stage/state enum would force a state machine that adds no behavior.
  Deactivation and reactivation are simple flag flips guarded by role.
- Q: Task lifecycle states? A: Open -> InProgress -> Done | Abandoned; terminals are closed.
  Why: "created, worked on, then finished or abandoned; terminal tasks stay closed."
- Q: Interaction correction model? A: append-only; correction is delete + re-log, no edit action.
  Why: the seed states exactly this.
- Q: Ownable records? A: Company, Contact, Deal carry an owner (a User). Task carries an assignee.
  Interaction is owned by the record it attaches to. Why: the seed says reps own records.
- Q: Is `reassign` (move ownership) a distinct action? A: Yes, on Company, Contact, Deal, gated to
  Manager/Admin. Why: "Only managers or admins may move ownership of a record."

## Phase 2 - Architecture

- Q: Package boundaries for the single binary? A: crm.cli -> crm.app -> {crm.domain, crm.repo} ->
  crm.model; crm.repo is the sole importer of go-ladybug. Why: keeps the domain pure (testable
  without a DB), isolates all graph access behind one boundary, matches Go hexagonal layering.
- Q: Where does the load-act-save loop live? A: crm.app orchestrates load (repo) -> pure domain
  transition (domain) -> save (repo). Why: keeps domain free of I/O; the FSM persist actor maps to
  the app->repo relationship.
- Q: Is an event-contract table required? A: No (marked N/A with reason). Why: single process, no
  message bus, no cross-component async events or choreography; all component coupling is
  synchronous in-process calls, and the one external is the embedded DB behind crm.repo.
- Q: How is LadybugDB modeled in C4? A: a container `store` tagged Database, bound by the contract
  external `external.ladybug` (imports github.com/LadybugDB/go-ladybug). Why: it is the state of
  record and the only external dependency; tagging drives G2 mitigation coverage.
- Q: Concurrency serialization? A: optimistic lock (a `version` property per node); conflicting
  writes retry with backoff up to 3 then refuse. Why: "never corrupt data; politely refuse one of
  two with a short automatic retry is acceptable."

## Phase 3 - State machines

- Q: Which components get a machine? A: Deal (lifecycle), Task (lifecycle), CommandExecution
  (operational envelope). Company/Contact/Interaction/User/Team are pure CRUD or flag flips: no
  lifecycle, waived in the placement table. Why: "not everything is a state machine"; only Deal and
  Task have real lifecycles, and the CLI needs one operational machine for open/auth/authz/execute.
- Q: Do Deal and Task route transitions through an explicit persist overlay? A: Yes
  (persisting/persistRetry/rolledBack). Why: every aggregate write can hit an optimistic-lock
  conflict; the overlay models the bounded retry then rollback-and-refuse, and it is the shape the
  linear-lifecycle formal pattern refines for Deal.
- Q: Where does the "short automatic retry" live? A: in the aggregate persist overlay
  (persistRetry, bounded by MaxRetries), not duplicated in CommandExecution. Why: avoids a
  redundant second retry loop; the command layer just reports the aggregate's refusal.
- Q: Is there a saga / cross-aggregate composition? A: No. Why: no distributed side effects to
  compensate; deletes cascade by ownership within one local transaction. So no composition.yaml.
- Q: Which formal semantics annotation is authored? A: Deal, pattern linear-lifecycle. Why: it is
  the one supported pattern that applies; Task's lifecycle lacks the advance/win/lose/reopen +
  reopen-to shape the pattern requires, so Task gets control-flow TLA only.
- Q: Where are amount-non-negative and reassign-role enforced? A: as domain/app-layer guards
  (`amountNonNegative`, `canReassign`) used by the non-lifecycle actions (create, updateAmount,
  reassign), not as Deal lifecycle-machine transitions. Why: they do not change lifecycle state;
  routing them through the persist overlay would break the linear-lifecycle onDone-target set.
- Q: MaxRetries for the persist overlay and command timeouts? A: MaxRetries = 3; PERSIST_TIMEOUT 5s,
  RETRY_BACKOFF ~200 ms, open/auth/authz timeouts 3s/2s/2s, EXEC_TIMEOUT 5s. Why: pragmatic defaults
  for a local single-file store; correctness over speed. Named delays, values live in the impl map.
- Q: Does CommandExecution run its own retry loop? A: No. Why: the bounded retry lives in the
  aggregate persist overlay; the envelope only reports the aggregate's refusal (state `refused`).

## Phase 4 - BUILD.md and toolchain

- Q: Go version and libraries? A: Go 1.23.x; go-ladybug pinned in go.mod/go.sum; argon2id or bcrypt
  from golang.org/x/crypto for hashing; property tests via the standard library testing/quick (no
  extra dependency). Why: reproducible environment, minimal dependencies, no state-machine library
  (the switch is clearer and a library would not earn its place).
- Q: closedAt on Lost as well as Won? A: yes, recordClose stamps closedAt on both terminals, but only
  deal-won-has-close-date is an invariant. Why: knowing when a deal was lost is useful; the invariant
  stays scoped to Won as the seed states.
- Q: Keep the generated formal .tla/.cfg files in design/formal alongside the hand-authored
  semantics? A: yes. Why: they are deterministically regenerated by verify_formal.sh; keeping them
  makes the proofs tangible and reviewable, consistent with committing the generated oracle.
