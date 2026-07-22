# Go CRM - Phase 3 state machines

Five machines, one per stateful component from ARCHITECTURE.md section 7. Each has a
`<Component>.machine.json` (XState v5, JSON-serializable; guards/actions/actors are string
names implemented by the coding agent; delays are named) and a `<Component>.matrix.md`
(named-unit contracts, failure catalog, and a transition table reconciled row by row against
the machine by G3). The CANONICAL transition oracle is the generated `<Component>.oracle.md`
(`machinery oracle`; stable test ids; G3 diffs it against a fresh generation).

| machine | kind | traces to |
|---|---|---|
| `Deal` | domain lifecycle | enum `DealStage`, Deal actions advanceStage/win/lose/reopen |
| `Task` | domain lifecycle | enum `TaskStatus`, Task actions start/complete/cancel/reassign |
| `User` | domain lifecycle (status only) | enum `UserStatus`, User actions disable/enable |
| `Session` | operational / auth | glossary `Session`; login/logout/resume, argon2id verify, token file |
| `CommandExecution` | operational envelope | the per-invocation crm binary lifecycle; owns the single write Tx |

## Entities that are PURE RECORDS with NO machine

These have no status enum and no lifecycle: they are CRUD over the graph, specified by a
contract (repository interface + invariant checks), not a state machine.

- **Contact** - CRUD + contract spec, not a lifecycle. Owner fixed at create (`contact-owned`, structural); no status enum, no transitions.
- **Account** - CRUD + contract spec, not a lifecycle. Owner fixed at create (`account-owned`, structural); groups Contacts/Deals but does not itself transition.
- **Pipeline** - CRUD + contract spec, not a lifecycle. A namespace for Deals; its only rule, `one-default-pipeline`, is a cross-record invariant enforced transactionally by the `setDefault` operation, not by a per-record lifecycle.
- **Activity** - append-only log entry (`activity-immutable`), never transitions. Only `log` and `delete` (correction); no status enum, so there is nothing to sequence.
- **Tag** - freeform label with "no lifecycle of its own" (per the domain model). `tag-name-unique` is a DB uniqueness constraint; create/apply/remove are plain CRUD.
- **Team** - grouping record for visibility scope. `team-name-unique` is a DB uniqueness constraint; create/rename are plain CRUD; no status enum.

## Authorization is a pure decision function (NOT a machine)

`crm.authz` is a pure `(actor, verb, entityType, ownerId, teamId) -> Decision` function with no
I/O (ARCHITECTURE.md section 3). It gets a contract spec and contract tests, not a machine. Its
four invariants (`rbac-crud-verbs`, `rbac-read-visibility`, `rbac-write-scope`,
`rbac-reassign-authority`) are enforced at a single call site: the `guardAuthorized` guard on
`CommandExecution.Authorizing`, with domain-level re-checks in the Deal/Task/User guards
(`guardCanReopen`, `guardCanReassign`, `guardAdminAuthority`) per the "authorization is enforced in
crm.domain, never in the command layer" rule.

## Invariant enforcement map (Gate 3 coverage)

Every invariant in `domain.modelith.yaml` has an enforcement point. Classes: **guard** (a machine
guard), **structural** (impossible given the state graph / data model), **DB constraint** (a repo
uniqueness/constraint surfaced as `ErrConstraint -> CommandExecution.ValidationFailed`), or
**operation** (a transactional read-modify-write on a pure record).

| invariant | class | enforcement point |
|---|---|---|
| `disabled-cannot-auth` | guard | Session `guardUserDisabled` (Authenticating.onDone / isErrDisabled) |
| `session-active-user` | guard | Session `guardSessionUserActive` (CheckingUser.onDone) |
| `deal-stage-forward` | guard + structural | Deal `guardCanAdvance`; Negotiation exposes no forward advance; reopen is the sanctioned exception |
| `deal-won-has-closedate` | guard | Deal `guardCanWin` |
| `deal-amount-nonneg` | guard | Deal `guardCanAdvance` / `guardCanWin` / `guardCanLose` (precondition of every persist) |
| `deal-terminal` | structural | Deal Won/Lost expose only `reopen`; other events are explicitly rejected |
| `deal-owned` | structural | owner set at create, immutable under advanceStage/win/lose/reopen |
| `task-assignee-visible` | guard | Task `guardCanReassign` |
| `task-terminal` | structural | Task Done/Cancelled are `final` (no reopen exists) |
| `task-owned` | structural + guard | owner set at create; `guardCanReassign` admits exactly one in-scope new owner |
| `rbac-crud-verbs` | guard | CommandExecution `guardAuthorized`; User `guardAdminAuthority` |
| `rbac-read-visibility` | guard | CommandExecution `guardAuthorized` (reads authorized too) |
| `rbac-write-scope` | guard | CommandExecution `guardAuthorized`; domain `guardCan*` re-checks |
| `rbac-reassign-authority` | guard | CommandExecution `guardAuthorized`; Deal `guardCanReopen`; Task `guardCanReassign` |
| `username-unique` | DB constraint | repo unique index on `username` (register/changePassword) |
| `team-name-unique` | DB constraint | repo unique index on `Team.name` |
| `tag-name-unique` | DB constraint | repo unique index on `Tag.name` |
| `password-hashed` | structural | only argon2id hashes are ever written (crm.session); no action stores plaintext |
| `single-team` | structural | data model: User has at most one Team relationship |
| `account-owned` | structural | owner set at create; required n:1 relationship |
| `contact-owned` | structural | owner set at create; required n:1 relationship |
| `activity-immutable` | structural | no update action exists; Activity is append-only |
| `activity-owned` | structural | `log` records the acting User |
| **`one-default-pipeline`** | **operation (NOT a machine guard, NOT structural)** | enforced by the `setDefault` transaction (unset prior default + set new, atomically) inside `CommandExecution.Executing`; Pipeline is a pure record with no lifecycle machine |

### The one invariant with no enforcing guard and not structurally impossible

`one-default-pipeline` ("exactly one Pipeline is marked default") is a cross-record cardinality
rule. It is NOT a machine guard (Pipeline has no lifecycle machine) and NOT structurally impossible
(nothing in the state graph prevents zero or two defaults). It is enforced by the `setDefault`
operation as an atomic read-modify-write inside the one write Tx. Recommended coverage: a
**contract/property test** on `setDefault` asserting the post-condition `count(isDefault==true) == 1`,
plus a repo-level partial-unique or invariant check. Flagged here so it is not lost: it lives in the
`setDefault` operation, not in any of the five machines.
