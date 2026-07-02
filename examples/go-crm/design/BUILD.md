# BUILD: Go CRM

> Single deliverable. Self-contained by design: a coding agent with zero prior context builds the
> system from this file alone, under hard TDD (section 10). Source-of-truth files are referenced per
> section for full detail, but you do not need to open them to build. When this document and a source
> file disagree, the source file wins and this document is a defect: stop and fix it.
>
> Sources (under `design/`): `domain.modelith.yaml` / `domain.modelith.md` (domain), `workspace.dsl` /
> `ARCHITECTURE.md` (architecture), `machines/*.machine.json` (XState v5 machines), `machines/*.matrix.md`
> (transition oracles), `machines/README.md` (non-machine catalog).

---

## 1. Purpose and scope

Go CRM is a single statically linked Go binary, `crm`, that runs one customer-relationship-management
command per invocation over an embedded LadybugDB property graph on local disk. Its users are four kinds
of operator distinguished only by role (Admin, Manager, Rep, ReadOnly); there is no server and no network.
It exists to give a single team a fast, offline, auditable CRM whose visibility and CRUD are governed by
role and record ownership, with every write applied atomically inside one database transaction.

**In scope**

- CRUD and lifecycle for nine record types (User, Team, Account, Contact, Deal, Pipeline, Activity, Task,
  Tag) via `crm <noun> <verb>` commands.
- Local login/logout with an on-disk session token; argon2id password hashing.
- Role- and ownership-based authorization on every command.
- Deal, Task, and User status lifecycles as explicit state machines.
- Atomic single-transaction writes with bounded retry on a busy store, and `crm backup` / `crm restore`.

**Out of scope**

- Any network server, multi-node deployment, replication, or high availability (the store is embedded and
  single-file).
- Multi-tenant isolation beyond the single local database directory.
- Concurrent multi-user write throughput: concurrent writers serialize on the single-writer lock; the
  second is refused after bounded retry.

## 2. Glossary

The only source for these words.

**Roles and access**

- **Admin** - a `User` role with unrestricted CRUD and visibility across all records.
- **Manager** - a `User` role that reads and writes records owned by any member of the manager's `Team`,
  and may reassign ownership within that team.
- **Rep** - a `User` role that can create records, write records it owns, and read records within its `Team`.
- **ReadOnly** - a `User` role that may read records within its `VisibilityScope` but may not create,
  update, or delete.
- **Owner** - the `User` to whom a record belongs; ownership is the unit of row-level visibility. Set at
  create, changed only by reassign.
- **VisibilityScope** - the set of records a `User` may read: own records for a `Rep`, team records for a
  `Manager`, all records for an `Admin`. `ReadOnly` reads within the same own/team scope but cannot write.
- **Session** - a local expiring credential written to `~/.crm/session` after a successful login that
  identifies the acting `User` for later commands until logout or expiry.

**Records** (defined in section 3): **User**, **Team**, **Account**, **Contact**, **Deal**, **Pipeline**,
**Activity**, **Task**, **Tag**.

**Technical terms**

- **LadybugDB** - the embedded property-graph store; single-file, single-writer, on local disk (default
  `~/.crm/db`). Accessed only through the Repository via Cypher.
- **Cypher** - LadybugDB's query language; all reads and writes are parameterized Cypher statements.
- **argon2id** - the memory-hard password hashing algorithm (`golang.org/x/crypto/argon2`) used for
  `User.passwordHash`.
- **Aggregate** - a record whose lifecycle is a state machine (Deal, Task, User). Loaded, acted on in
  memory, and saved inside one write transaction (no long-lived in-memory actor).
- **Guard** - a boolean precondition on a transition. If false, the transition does not fire and a
  `record*Denied` action surfaces the reason.
- **Actor (machine sense)** - an invoked side-effecting unit (`saveDeal`, `verifyCredentials`); distinct
  from a person-role actor. In this document the acting person is the **caller** / **acting user**.
- **Transition** - a state change fired by an event, an invoke result (`onDone`/`onError`), an `after`
  delay, or an `always` condition.
- **Write transaction / write Tx** - the one Cypher `BEGIN ... COMMIT` an invocation owns; load-act-save
  happens inside it; any error rolls it back with no partial write.
- **Single-writer lock** - the store permits one writer at a time across processes; a second concurrent
  `crm` write gets `ErrLocked` and is retried up to three times (~1.5s) before it is refused.
- **Walking skeleton** - the thinnest end-to-end slice that exercises one real transition through one real
  boundary, built first to prove the topology before breadth is added.
- **Hard TDD** - a test-writer agent writes tests from sections 6 and 7; tests are locked; the implementer
  makes them pass without editing them (section 10).

## 3. Domain model (the what)

Single canonical schema. Source of truth: `design/domain.modelith.yaml` (lints clean). Later sections
reference these names and types and never restate them.

### 3.1 Entity-relationship diagram

```mermaid
erDiagram
    Account {}
    Activity {}
    Contact {}
    Deal {}
    Pipeline {}
    Tag {}
    Task {}
    Team {}
    User {}
    Account ||--o{ Contact : "referenced"
    Account ||--o{ Deal : "referenced"
    Account }o--|| User : "referenced"
    Activity }o--|| User : "referenced"
    Activity }o--|| Contact : "referenced"
    Contact }o--|| User : "referenced"
    Deal }o--|| User : "referenced"
    Deal }o--o{ Contact : "referenced"
    Pipeline ||--o{ Deal : "referenced"
    Tag }o--o{ Deal : "referenced"
    Task }o--|| User : "referenced"
    Task }o--|| Deal : "referenced"
    Team ||--o{ User : "referenced"
```

### 3.2 Enums

| enum | values (in order) |
|---|---|
| `UserRole` | `Admin`, `Manager`, `Rep`, `ReadOnly` |
| `UserStatus` | `Active`, `Disabled` |
| `DealStage` | `Lead`, `Qualified`, `Proposal`, `Negotiation`, `Won`, `Lost` |
| `TaskStatus` | `Open`, `InProgress`, `Done`, `Cancelled` |
| `ActivityType` | `Call`, `Meeting`, `Email`, `Note` |

`DealStage` forward order is `Lead < Qualified < Proposal < Negotiation`; `Won`/`Lost` are terminal
outcomes reachable from any non-terminal stage. `next(stage)`: Lead->Qualified, Qualified->Proposal,
Proposal->Negotiation; Negotiation has no forward stage (win/lose only). `TaskStatus` non-terminal:
`Open`, `InProgress`; terminal: `Done`, `Cancelled`.

### 3.3 Data dictionary

Every entity is a graph node with a synthetic string `id` (primary key) not listed below. Owned entities
also carry an `ownerId` string referencing the owning `User` (the storage form of the `*-owned` invariants
and the ER `}o--|| User` edges); set at create, changed only by reassign.

| entity | attribute | type | notes |
|---|---|---|---|
| **User** | `username` | string | unique (`username-unique`) |
| | `passwordHash` | string | argon2id encoded hash only (`password-hashed`) |
| | `role` | `UserRole` | |
| | `status` | `UserStatus` | machine state (User aggregate) |
| | `createdAt` | timestamp | |
| | (rel) team | `Team` n:1 | at most one (`single-team`) |
| **Team** | `name` | string | unique (`team-name-unique`) |
| | (rel) members | `User` 1:n | |
| **Account** | `name` | string | |
| | `domain` | string | |
| | `industry` | string | |
| | `ownerId` | string | owning `User` (`account-owned`) |
| | (rel) contacts, deals | `Contact` 1:n, `Deal` 1:n | |
| **Contact** | `fullName` | string | |
| | `email` | string | |
| | `phone` | string | |
| | `title` | string | |
| | `ownerId` | string | owning `User` (`contact-owned`) |
| **Deal** | `title` | string | |
| | `amountCents` | integer | >= 0 (`deal-amount-nonneg`) |
| | `stage` | `DealStage` | machine state (Deal aggregate) |
| | `closeDate` | timestamp | required when `stage == Won` (`deal-won-has-closedate`) |
| | `ownerId` | string | owning `User` (`deal-owned`) |
| | (rel) contacts | `Contact` n:n | |
| **Pipeline** | `name` | string | |
| | `isDefault` | boolean | exactly one true across all pipelines (`one-default-pipeline`) |
| | (rel) deals | `Deal` 1:n | |
| **Activity** | `type` | `ActivityType` | |
| | `subject` | string | |
| | `body` | string | immutable after create (`activity-immutable`) |
| | `occurredAt` | timestamp | immutable after create (`activity-immutable`) |
| | `ownerId` | string | logging `User` (`activity-owned`) |
| | (rel) contact | `Contact` n:1 | |
| **Task** | `title` | string | |
| | `dueDate` | timestamp | |
| | `status` | `TaskStatus` | machine state (Task aggregate) |
| | `ownerId` | string | owning `User` (`task-owned`) |
| | (rel) deal | `Deal` n:1 | optional link |
| **Tag** | `name` | string | unique (`tag-name-unique`) |
| | `color` | string | |
| | (rel) deals | `Deal` n:n | |

### 3.4 Invariants (24, non-negotiable)

Enforcement point, component, and test ids are in the section 6 matrix; not duplicated here.

| id | statement | owner |
|---|---|---|
| `username-unique` | No two `User` records share a username. | User |
| `password-hashed` | A `User` password is persisted only as a hash, never plaintext. | User |
| `disabled-cannot-auth` | A `Disabled` `User` cannot establish a `Session`. | User/Session |
| `single-team` | A `User` belongs to at most one `Team`. | User |
| `team-name-unique` | No two `Team` records share a name. | Team |
| `account-owned` | Every `Account` has exactly one owning `User`. | Account |
| `contact-owned` | Every `Contact` has exactly one owning `User`. | Contact |
| `deal-owned` | Every `Deal` has exactly one owning `User`. | Deal |
| `deal-amount-nonneg` | A `Deal` amount is zero or positive. | Deal |
| `deal-stage-forward` | A `Deal` moves only to a later stage or to Won/Lost; never backward except by explicit reopen. | Deal |
| `deal-terminal` | A `Deal` in Won or Lost is terminal and changes only by reopen. | Deal |
| `deal-won-has-closedate` | A `Deal` in Won has a closeDate. | Deal |
| `one-default-pipeline` | Exactly one `Pipeline` is marked default. | Pipeline |
| `activity-immutable` | An `Activity` body and occurredAt never change after creation. | Activity |
| `activity-owned` | Every `Activity` records the `User` who logged it. | Activity |
| `task-owned` | Every `Task` has exactly one owning `User`. | Task |
| `task-terminal` | A `Task` in Done or Cancelled is terminal. | Task |
| `task-assignee-visible` | A `Task` may be reassigned only to a `User` within the assigner's `VisibilityScope`. | Task |
| `tag-name-unique` | No two `Tag` records share a name. | Tag |
| `rbac-crud-verbs` | A verb is allowed only if the role grants it: Admin/Manager/Rep may create/read/update/delete; ReadOnly only read. | RBAC |
| `rbac-read-visibility` | Read only within `VisibilityScope`: Admin all; others own or team-owned records. | RBAC |
| `rbac-write-scope` | Update/delete only in scope: Admin any; Manager any team member's record; Rep only its own; ReadOnly none. | RBAC |
| `rbac-reassign-authority` | Only an Admin, or a Manager acting within the manager's `Team`, may change a record's `Owner`. | RBAC |
| `session-active-user` | A `Session` is valid only while its `User` status is Active. | Session |

## 4. Architecture (the how)

Source of truth: `design/workspace.dsl` and `design/ARCHITECTURE.md`. Data shapes are section 3.

### 4.1 Context and containers

```mermaid
flowchart LR
  admin[Admin] --> cli
  manager[Manager] --> cli
  rep[Rep] --> cli
  readonly[ReadOnly] --> cli
  subgraph crm[Go CRM system]
    cli[crm binary: Go / cobra] --> store[(Graph Store: LadybugDB embedded)]
    cli --> sf[(Session File: ~/.crm/session)]
  end
```

### 4.2 Components inside the `crm` binary

```mermaid
flowchart LR
  commands[Command Layer crm.commands] --> session[Session and Auth crm.session]
  commands --> domain[Domain Services crm.domain]
  commands --> repo[Repository crm.repo]
  session --> repo
  session --> sf[(Session File)]
  domain --> authz[Authorization RBAC crm.authz]
  domain --> repo
  repo --> store[(Graph Store LadybugDB)]
```

- **Command Layer (`crm.commands`)** owns process lifecycle: parses argv, opens the database, begins and
  commits or rolls back the single write transaction, renders output. Machine: `CommandExecution`.
- **Session and Auth (`crm.session`)** performs login (verify argon2id hash), reads/writes the session
  token, resolves the current `User`. Enforces `disabled-cannot-auth` and `session-active-user`. Machine:
  `Session`.
- **Authorization (`crm.authz`)** is a pure decision function over `(actor, verb, entityType, ownerId,
  teamId)`. Single home of the four `rbac-*` invariants. No I/O, no machine: a contract spec.
- **Domain Services (`crm.domain`)** hold the aggregates whose lifecycles are machines (`Deal`, `Task`,
  `User`); call Authorization before every mutation and Repository to read and persist.
- **Repository (`crm.repo`)** is the only component that imports `go-ladybug`. Translates domain reads and
  writes to Cypher, executes them in the caller's transaction, maps LadybugDB errors to typed domain errors.

### 4.3 Technology stack

| concern | choice | why |
|---|---|---|
| language | Go 1.22+ | single static binary; good CLI ergonomics |
| CLI | cobra | subcommands, flags, help; matches `crm <noun> <verb>` |
| password hash | argon2id (`golang.org/x/crypto/argon2`) | enforces `password-hashed`; memory-hard |
| store | LadybugDB via `github.com/LadybugDB/go-ladybug` | embedded property graph; CRM is relationship-shaped |
| query | Cypher via `Connection.Query` / `Prepare` + `Execute` | parameterized statements |

### 4.4 Deployment topology

One binary invoked per command. All state is local: the LadybugDB directory (`~/.crm/db`) plus the session
token file (`~/.crm/session`). No server, no replicas, no operator, no HA. **HA / replication: N/A by
design** - the store is a single embedded file with a single-writer lock; there is nothing to replicate and
no failover, which is exactly why corruption is fatal-until-restore and is recovered only from a
`crm backup` (section 6). Concurrency is bounded by the store: one connection and one write transaction per
process; two concurrent writes serialize or the second is refused (`ErrLocked`), treated as a first-class
recoverable failure, not a crash.

### 4.5 Architecture Contract (boundaries + dependency rules)

The coding agent must not introduce any cross-boundary dependency outside `allow`. Enforced by contract
test **C-ARCH-01** (section 7).

```yaml
contract_version: 1
boundaries:
  - id: crm.commands   # internal/cli/**        exposes internal/cli/root.go
  - id: crm.session    # internal/session/**    exposes internal/session/session.go
  - id: crm.authz      # internal/authz/**      exposes internal/authz/authz.go
  - id: crm.domain     # internal/domain/**     exposes internal/domain/service.go
  - id: crm.repo       # internal/repo/**       exposes internal/repo/repo.go
dependency_rules:
  allow:
    - crm.commands -> crm.session
    - crm.commands -> crm.domain
    - crm.commands -> crm.repo      # open db, own the transaction boundary
    - crm.session  -> crm.repo
    - crm.domain   -> crm.authz
    - crm.domain   -> crm.repo
  deny:
    - crm.commands -> crm.authz     # authorization decided inside domain services
    - "crm.* -> external.ladybug"   # only crm.repo may import go-ladybug
  notes:
    - "All graph access goes through crm.repo. Only crm.repo imports go-ladybug."
    - "Authorization is enforced in crm.domain, never in the command layer, so no command path can skip it."
```

### 4.6 Interface contracts at each boundary

Go-flavored pseudocode; types reference section 3. Each interface is the seam for the section 7 contract
tests (C-REPO-*, C-AUTHZ-*, C-SESS-*).

```go
// crm.repo  (only importer of go-ladybug; all data methods run inside an open write Tx)
type Repo interface {
  Open(path string) (Tx, error)        // ErrLocked, ErrCorrupt, ErrUnavailable
  BeginWrite(Tx) error                 // Cypher BEGIN TRANSACTION
  Commit(Tx) error
  Rollback(Tx) error                   // idempotent; store guarantees no partial write
  GetUserByName(Tx, name string) (User, error)  // ErrNotFound
  GetDeal(Tx, id string) (Deal, error)          // ErrNotFound
  SaveDeal(Tx, Deal) error                       // ErrConstraint, ErrConflict, ErrDiskFull, ErrTimeout, ErrLocked
  // ... GetTask/SaveTask, GetUser/SaveUser, GetAccount/SaveAccount, GetContact/SaveContact,
  //     SaveActivity (append-only; no update), SavePipeline, SetDefaultPipeline, SaveTag, SaveTeam
}
// Typed errors (from go-ladybug):
//   ErrLocked, ErrCorrupt, ErrUnavailable, ErrNotFound, ErrConstraint, ErrConflict, ErrDiskFull, ErrTimeout

// crm.authz  (pure; no I/O)
type Authorizer interface {
  Authorize(actor User, verb Verb, entity EntityType, ownerID, teamID string) Decision
}
type Decision struct { Allowed bool; Reason string }   // Reason set iff !Allowed
// Verb in {create, read, update, delete, reassign}; EntityType is one of the nine record types.

// crm.session
type Sessions interface {
  Login(name, password string) (Session, error)  // ErrBadCredentials, ErrDisabled, ErrLocked
  Current() (User, error)                         // ErrNoSession, ErrExpired
  Logout() error
}
```

**Idempotency and retry (contract-level).** Reads are safe to retry. Writes run in one transaction and are
retried only on `ErrLocked` (the transaction never partially committed, so a retry re-applies the whole
unit exactly once). `Login` is not retried on `ErrBadCredentials`. Retry bound everywhere: <= 3 attempts,
~1.5s total (backoff ~500ms).

### 4.7 Persistence and placement

CLI invocations are short-lived and single-process, so **there are no in-memory actors**. Every stateful
aggregate is loaded, acted on, and saved inside the one write transaction the Command Layer owns. This is a
hard constraint the section 9 Go realization must honor.

| component | placement | persistence | concurrency serialization |
|---|---|---|---|
| Deal aggregate | ephemeral in-process; load-act-save in the Tx | graph node `stage` attribute | read-modify-write in one write Tx; cross-process by single-writer lock |
| Task aggregate | ephemeral in-process; load-act-save in the Tx | graph node `status` attribute | as above |
| User aggregate | ephemeral in-process; load-act-save in the Tx | graph node `status` attribute | as above |
| Session | in-process during a command; token on disk | `~/.crm/session` (user id + expiry, HMAC-signed) | last write wins; single local user |
| Command execution | ephemeral per invocation (the envelope) | none | one invocation owns the write Tx |

## 5. Behavior: the state machines (the logic)

Five machines, one per stateful component (source: `design/machines/*.machine.json`, XState v5,
JSON-serializable; guards/actions/actors are string names the coding agent implements; delays are named).
For each: a plain-language lifecycle, the state list and key transitions, the named-unit contract table
(the units to implement), and the failure catalog. The full transition oracle (every transition and guard
branch) is section 7.

**Shared persist overlay (Deal, Task, User).** The domain transition is wrapped by
`persisting -> {persistRetry | rolledBack}`. `persisting` invokes the repo `saveX` actor; `onDone` routes
by `pendingIsX` to the committed state; `onError` classifies the typed repo error; `isErrLocked` goes to
`persistRetry` (backoff 500ms, back to `persisting`, until `retriesExhausted` at 3 ~1.5s -> `rolledBack`);
every other error and `after persistTimeout` (10s) go to `rolledBack`, which routes by `priorIsX` back to
the pre-transition state. The persist is atomic, so any failure leaves the store and the aggregate at
`priorStage`/`priorStatus`.

### 5.1 Deal aggregate (`crm.domain`) - `machines/Deal.machine.json`

States trace to `DealStage`; events to `Deal` actions.

**Lifecycle.** A Deal is created at `Lead` owned by its creator. Its owner advances it one stage forward at
a time (Lead -> Qualified -> Proposal -> Negotiation), never backward. From any non-terminal stage the owner
may win it (requires a closeDate) or lose it, reaching terminal `Won`/`Lost`. Terminal deals accept nothing
but `reopen`, which only a Manager or Admin in scope may fire and which returns the deal to `Negotiation`
(the one sanctioned backward move). Every accepted move is persisted atomically inside the command's write
Tx before the in-memory stage changes; a persist failure leaves the deal in its prior stage.

**States.** Resting: `Lead`, `Qualified`, `Proposal`, `Negotiation`, `Won`, `Lost`. Overlay (transient):
`persisting`, `persistRetry`, `rolledBack`.

**Key transitions.** `Lead/Qualified/Proposal --advanceStage[guardCanAdvance]--> persisting -> next`;
`nonterminal --win[guardCanWin]--> persisting -> Won (+commitCloseDate)`;
`nonterminal --lose[guardCanLose]--> persisting -> Lost`;
`Negotiation --advanceStage--> recordAdvanceDenied` (no forward stage; structural `deal-stage-forward`);
`Won/Lost --reopen[guardCanReopen]--> persisting -> Negotiation`;
`Won/Lost --advanceStage|win|lose--> recordTerminalRejected` (structural `deal-terminal`).

**Named-unit contract table.**

| name | kind | signature | pre / post | maps to |
|---|---|---|---|---|
| `saveDeal` | actor | `(input{dealId,stage,amountCents,closeDate,ownerId,actor}) -> DealRow \| err{ErrConstraint,ErrConflict,ErrDiskFull,ErrTimeout,ErrLocked}` | pre: guard passed, tx open. post: node `stage`=pendingStage atomically, else store unchanged | `crm.domain -> crm.repo -> store` (SaveDeal) |
| `guardCanAdvance` | guard | `(ctx,evt)->bool` | true iff pendingStage is next forward AND caller may write AND amountCents>=0 | `deal-stage-forward`,`rbac-write-scope`,`deal-amount-nonneg` |
| `guardCanWin` | guard | `(ctx,evt)->bool` | true iff evt supplies closeDate AND caller may write AND amountCents>=0 | `deal-won-has-closedate`,`rbac-write-scope`,`deal-amount-nonneg` |
| `guardCanLose` | guard | `(ctx,evt)->bool` | true iff caller may write AND amountCents>=0 | `rbac-write-scope`,`deal-amount-nonneg` |
| `guardCanReopen` | guard | `(ctx,evt)->bool` | true iff caller is Manager/Admin in scope | `rbac-reassign-authority`,`rbac-write-scope`; exception to `deal-stage-forward` |
| `pendingIsQualified/Proposal/Negotiation/Won/Lost` | guard | `(ctx)->bool` | true iff `pendingStage` equals that stage | persist success routing |
| `priorIsLead/Qualified/Proposal/Negotiation/Won/Lost` | guard | `(ctx)->bool` | true iff `priorStage` equals that stage | rollback routing |
| `isErrLocked/isErrConstraint/isErrDiskFull/isErrTimeout` | guard | `(ctx,evt)->bool` | true iff `evt.error` is that typed repo error | section 4.6 classes |
| `retriesExhausted` | guard | `(ctx)->bool` | true iff `retries>=3` | retry bound |
| `setPendingAdvance` | action | `(ctx,evt)->ctx` | `priorStage:=stage; pendingStage:=next(stage)` | - |
| `setPendingWin` | action | `(ctx,evt)->ctx` | `priorStage:=stage; pendingStage:=Won; pendingCloseDate:=evt.closeDate` | - |
| `setPendingLose` | action | `(ctx,evt)->ctx` | `priorStage:=stage; pendingStage:=Lost` | - |
| `setPendingReopen` | action | `(ctx,evt)->ctx` | `priorStage:=stage; pendingStage:=Negotiation` | - |
| `commitStage` | action | `(ctx)->ctx` | `stage:=pendingStage` | - |
| `commitCloseDate` | action | `(ctx)->ctx` | `closeDate:=pendingCloseDate` | `deal-won-has-closedate` |
| `incrementRetries` | action | `(ctx)->ctx` | `retries:=retries+1` | - |
| `recordError/recordConstraint/recordDiskFull/recordTimeout/recordUnknownError/recordRetriesExhausted/recordRoutingError` | action | `(ctx,evt)->ctx` | `lastError:=classified error` | maps repo errors to a domain error |
| `recordAdvanceDenied/recordWinDenied/recordLoseDenied/recordReopenDenied/recordReopenNotTerminal/recordTerminalRejected` | action | `(ctx,evt)->ctx` | set rejection reason; no state change | surfaces the violated invariant |

**Failure catalog.**

| failure | detection | transition | recovery | bound / residual |
|---|---|---|---|---|
| Constraint violation | `saveDeal` onError `isErrConstraint` | persisting->rolledBack->priorStage | surface validation error; store unchanged | one write Tx. Residual: none |
| Disk full | `saveDeal` onError `isErrDiskFull` | persisting->rolledBack->priorStage | fail loudly; DB consistent | atomic. Residual: free disk |
| Timeout | `saveDeal` onError `isErrTimeout` OR after persistTimeout 10s | persisting->rolledBack->priorStage | abort, surface, roll back | SetTimeout 10s. Residual: none |
| Store locked | `saveDeal` onError `isErrLocked` | persisting->persistRetry->persisting, then rolledBack when retriesExhausted | bounded retry then surface | retry <=3 ~1.5s. Residual: refused after 3 |
| Conflict / unknown | `saveDeal` onError catch-all | persisting->rolledBack->priorStage | surface; re-run | Residual: envelope also retries `ErrConflict` |
| Illegal move (backward/terminal/unauthorized/neg amount/missing closeDate) | guard false or terminal reject | internal, `record*Denied`/`recordTerminalRejected` | reject with invariant id; no write | structural. Residual: none |

### 5.2 Task aggregate (`crm.domain`) - `machines/Task.machine.json`

States trace to `TaskStatus`; events to `Task` actions.

**Lifecycle.** A Task is created at `Open` owned by its creator, optionally linked to a Deal. The owner
starts it (Open -> InProgress), completes it (-> Done), or cancels it (-> Cancelled). Done and Cancelled are
`final`: they accept no events, which enforces `task-terminal` structurally (there is no reopen for a Task,
unlike a Deal). A Manager/Admin in scope may reassign the task to another user inside the assigner's
VisibilityScope; reassign changes the owner but keeps the status, so the persist lands on the same state.

**States.** Resting: `Open`, `InProgress`; terminal (`final`): `Done`, `Cancelled`. Overlay: `persisting`,
`persistRetry`, `rolledBack`.

**Key transitions.** `Open --start[guardCanStart]--> persisting -> InProgress`;
`Open/InProgress --complete[guardCanComplete]--> persisting -> Done`;
`Open/InProgress --cancel[guardCanCancel]--> persisting -> Cancelled`;
`Open/InProgress --reassign[guardCanReassign]--> persisting -> same status`;
`InProgress --start--> recordAlreadyStarted` (idempotent); `Done/Cancelled --any--> final, no transition`.

**Named-unit contract table.**

| name | kind | signature | pre / post | maps to |
|---|---|---|---|---|
| `saveTask` | actor | `(input{taskId,status,ownerId,newAssigneeId,actor}) -> TaskRow \| err{...}` | pre: guard passed, tx open. post: node `status`(+`owner` on reassign) atomically, else unchanged | `crm.domain -> crm.repo -> store` (SaveTask) |
| `guardCanStart` | guard | `(ctx,evt)->bool` | true iff caller may write (owner/manager/admin in scope) | `rbac-write-scope` |
| `guardCanComplete` | guard | `(ctx,evt)->bool` | true iff source non-terminal AND caller may write | `task-terminal`,`rbac-write-scope` |
| `guardCanCancel` | guard | `(ctx,evt)->bool` | true iff source non-terminal AND caller may write | `task-terminal`,`rbac-write-scope` |
| `guardCanReassign` | guard | `(ctx,evt)->bool` | true iff new assignee in assigner VisibilityScope AND caller Manager/Admin in scope | `task-assignee-visible`,`rbac-reassign-authority`,`rbac-write-scope` |
| `pendingIsOpen/InProgress/Done/Cancelled` | guard | `(ctx)->bool` | true iff `pendingStatus` equals that status | persist success routing |
| `priorIsOpen/InProgress` | guard | `(ctx)->bool` | true iff `priorStatus` equals that status | rollback routing |
| `isErrLocked/isErrConstraint/isErrDiskFull/isErrTimeout`, `retriesExhausted` | guard | as Deal | typed error / retry bound | section 4.6 |
| `setPendingStart/Complete/Cancel` | action | `(ctx)->ctx` | `priorStatus:=status; pendingStatus:=InProgress/Done/Cancelled` | - |
| `setPendingReassign` | action | `(ctx,evt)->ctx` | `priorStatus:=status; pendingStatus:=status; newAssigneeId:=evt.assigneeId` | `task-owned` |
| `commitStatus` | action | `(ctx)->ctx` | `status:=pendingStatus` (owner if reassign) | - |
| `incrementRetries`, error `record*`, `recordStartDenied/CompleteDenied/CancelDenied/ReassignDenied/AlreadyStarted` | action | `(ctx,evt)->ctx` | classified error, or rejection reason with no state change | surfaces invariant |
| `recordTaskClosed` | action | `(ctx)->ctx` | entry marker on a terminal status | - |

**Failure catalog.**

| failure | detection | transition | recovery | bound / residual |
|---|---|---|---|---|
| Constraint/disk-full/timeout/locked on write | `saveTask` onError classes, after persistTimeout | persisting->rolledBack->priorStatus (locked via persistRetry) | as Deal 5.1 | as Deal 5.1 |
| Reassign to out-of-scope user | `guardCanReassign` false | Open/InProgress internal, `recordReassignDenied` | reject; no write | `task-assignee-visible` before invoke. Residual: none |
| Mutate a terminal task | event at Done/Cancelled (`final`) | none (structurally rejected) | closed; no-op | `task-terminal` structural. Residual: none |

### 5.3 User aggregate (`crm.domain`) - `machines/User.machine.json`

Status lifecycle only. States trace to `UserStatus`; events to `disable`/`enable` (both actor Admin).

**Lifecycle.** This machine covers only the Active <-> Disabled status transitions driven by the Admin-only
`disable`/`enable` actions. An Active user may be disabled by an Admin; a Disabled user may be re-enabled by
an Admin; the redundant direction is an idempotent no-op. `register`/`changePassword`/`assignRole` are
create/update paths owned by `crm.session` and the repo, and `login`/`logout` belong to the Session machine,
not here. Disabling a user has a downstream effect on live sessions, enforced by `session-active-user` in
the Session machine.

**States.** Resting: `Active`, `Disabled`. Overlay: `persisting`, `persistRetry`, `rolledBack`.

**Key transitions.** `Active --disable[guardAdminAuthority]--> persisting -> Disabled`;
`Disabled --enable[guardAdminAuthority]--> persisting -> Active`;
`Active --enable--> recordAlreadyActive`; `Disabled --disable--> recordAlreadyDisabled` (idempotent);
non-admin `disable`/`enable` -> `recordAuthorityDenied`.

**Named-unit contract table.**

| name | kind | signature | pre / post | maps to |
|---|---|---|---|---|
| `saveUser` | actor | `(input{userId,status,actor}) -> UserRow \| err{...}` | pre: guard passed, tx open. post: node `status` atomically, else unchanged | `crm.domain -> crm.repo -> store` (SaveUser) |
| `guardAdminAuthority` | guard | `(ctx,evt)->bool` | true iff `actor.role==Admin` | `rbac-crud-verbs` (disable/enable are Admin verbs) |
| `pendingIsActive/Disabled`, `priorIsActive/Disabled` | guard | `(ctx)->bool` | true iff pending/prior status equals that value | persist / rollback routing |
| `isErrLocked/isErrConstraint/isErrDiskFull/isErrTimeout`, `retriesExhausted` | guard | as Deal | typed error / retry bound | section 4.6 |
| `setPendingDisable/Enable` | action | `(ctx)->ctx` | `priorStatus:=status; pendingStatus:=Disabled/Active` | - |
| `commitStatus`, `incrementRetries`, error `record*` | action | `(ctx,evt)->ctx` | as Deal | - |
| `recordAuthorityDenied/recordAlreadyActive/recordAlreadyDisabled` | action | `(ctx,evt)->ctx` | rejection or idempotent no-op reason; no state change | surfaces `rbac-crud-verbs` denial |

**Failure catalog.**

| failure | detection | transition | recovery | bound / residual |
|---|---|---|---|---|
| Constraint/disk-full/timeout/locked on write | `saveUser` onError classes, after persistTimeout | persisting->rolledBack->priorStatus (locked via persistRetry) | as Deal 5.1 | as Deal 5.1 |
| Non-admin attempts disable/enable | `guardAdminAuthority` false | Active/Disabled internal, `recordAuthorityDenied` | reject; no write | `rbac-crud-verbs`. Residual: none |

### 5.4 Session (`crm.session`) - `machines/Session.machine.json`

Operational/auth machine (Session is not a Modelith entity; it is the credential from the glossary).
Enforces `disabled-cannot-auth` (login) and `session-active-user` (resume) as guards.

**Lifecycle.** From `Anonymous`, a `login` verifies credentials (`verifyCredentials` against the repo,
argon2id). If the verified user is Disabled, the machine denies (`AuthDenied`); otherwise it writes the
signed token (`WritingSession`) and becomes `Active`. A `resume` reads the token file (`Resolving`); an
expired token goes to `Expired`, a valid token loads the user (`CheckingUser`) and requires the user still
be Active (`session-active-user`) to reach `Active`, else `Invalidated`. `Active` handles `useSession`
(command proceeds), `logout` (clears token -> `LoggedOut`), and expires after `sessionTTL` (8h). Bad
password -> `AuthFailed` (never auto-retried); store-locked verify/load -> `VerifyRetry` (bounded);
file/verify/load errors and timeouts -> `SessionUnavailable`. Every terminal-ish state handles every event
explicitly (a no-session state rejects `useSession`/`logout` rather than silently ignoring).

**States.** `Anonymous`, `Authenticating`, `VerifyRetry`, `WritingSession`, `Resolving`, `CheckingUser`,
`Active`, `LoggingOut`, `Expired`, `LoggedOut`, `AuthFailed`, `AuthDenied`, `Invalidated`,
`SessionUnavailable`.

**Named-unit contract table.**

| name | kind | signature | pre / post | maps to |
|---|---|---|---|---|
| `verifyCredentials` | actor | `(input{username,password}) -> User \| err{ErrBadCredentials,ErrDisabled,ErrLocked,ErrUnavailable}` | pre: username present. post: returns User iff argon2id hash matches; never on bad credentials | `crm.session -> crm.repo` |
| `writeSessionFile` | actor | `(input{userId,expiresAt}) -> ok \| err` | post: HMAC-signed token written to `~/.crm/session` | `crm.session -> crm.sessionfile` |
| `readSessionFile` | actor | `() -> {userId,expiresAt} \| err{ErrNoSession,ErrExpired,ErrUnreadable}` | post: parsed token or typed error | `crm.session -> crm.sessionfile` |
| `loadUser` | actor | `(input{userId}) -> User \| err{ErrNotFound,ErrLocked,ErrUnavailable}` | post: returns User with current status | `crm.session -> crm.repo` |
| `clearSessionFile` | actor | `() -> ok \| err` | post: token removed/truncated (best-effort) | `crm.session -> crm.sessionfile` |
| `guardUserDisabled` | guard | `(ctx,evt)->bool` | true iff verified user status==Disabled (deny path) | `disabled-cannot-auth` |
| `guardSessionUserActive` | guard | `(ctx,evt)->bool` | true iff loaded user status==Active | `session-active-user` |
| `guardSessionExpired` | guard | `(ctx,evt)->bool` | true iff token `expiresAt<=now` | expiry window for `session-active-user` |
| `isErrBadCredentials/isErrDisabled/isErrLocked/isErrNoSession/isErrExpired/isErrNotFound` | guard | `(ctx,evt)->bool` | true iff `evt.error` is that typed error | section 4.6 error types |
| `retriesExhausted` | guard | `(ctx)->bool` | true iff `retries>=3` | retry bound |
| `setCredentials` | action | `(ctx,evt)->ctx` | `username:=evt.username` (password held transiently for the invoke, never stored) | `password-hashed` |
| `captureUser` | action | `(ctx,evt)->ctx` | `userId,role,teamId,userStatus := verified/loaded user` | - |
| `captureToken` | action | `(ctx,evt)->ctx` | `userId,expiresAt := token` | - |
| `incrementRetries` | action | `(ctx)->ctx` | `retries:=retries+1` | - |
| `recordExpired` | action | `(ctx)->ctx` | mark expired; drop in-memory identity | `session-active-user` |
| `recordDisabled/recordBadCredentials/recordUserNotActive/recordUserMissing` | action | `(ctx,evt)->ctx` | set auth-denial reason | `disabled-cannot-auth`/`session-active-user` |
| `recordError/recordVerifyError/recordFileError/recordLoadError/recordTimeout/recordRetriesExhausted` | action | `(ctx,evt)->ctx` | `lastError:=classified error` | maps repo/file errors |
| `recordLogoutWarning` | action | `(ctx,evt)->ctx` | note best-effort logout (token may remain) | residual-risk marker |
| `recordSessionUsed/recordAlreadyActive/recordAlreadyResolved/recordNoSession/recordNoSessionToLogout/recordNoActiveSession/recordExpiredNeedsLogin/recordSessionExpired` | action | `(ctx,evt)->ctx` | no-op/reject reason; no state change | explicit event handling |

**Failure catalog.**

| failure | detection | transition | recovery | bound / residual |
|---|---|---|---|---|
| Bad password | `verifyCredentials` onError `isErrBadCredentials` | Authenticating->AuthFailed | user re-runs `crm login` (NOT auto-retried) | brute-force slowed by argon2id. Residual: none |
| Disabled user logs in | onDone `guardUserDisabled` or onError `isErrDisabled` | Authenticating->AuthDenied | admin must `enable` | `disabled-cannot-auth`. Residual: none |
| Store locked during verify | `verifyCredentials` onError `isErrLocked` | Authenticating->VerifyRetry->Authenticating, then SessionUnavailable when retriesExhausted | bounded retry then surface | retry <=3. Residual: refused after 3 |
| Store unavailable/corrupt during verify | onError catch-all / after verifyTimeout 5s | Authenticating->SessionUnavailable | surface; envelope reports DBError/Corrupt | Corrupt fatal at envelope. Residual: restore from backup |
| Token write fails | `writeSessionFile` onError / after fileIoTimeout 2s | WritingSession->SessionUnavailable | verify passed but no token; retry login | fail closed. Residual: no session established |
| No session on resume | `readSessionFile` onError `isErrNoSession` | Resolving->Anonymous | require `crm login` | Residual: none |
| Token expired | onDone `guardSessionExpired` / onError `isErrExpired` / after sessionTTL | -> Expired | require `crm login` | signed expiry authoritative. Residual: none |
| Token unreadable | onError catch-all / after fileIoTimeout | Resolving->SessionUnavailable | surface; delete + re-login | Residual: corrupt token |
| User no longer Active on resume | `loadUser` onDone `!guardSessionUserActive` / onError `isErrNotFound` | CheckingUser->Invalidated | re-auth (denied if still disabled) | `session-active-user`. Residual: none |
| Store locked/timeout on resume load | `loadUser` onError `isErrLocked`->VerifyRetry / after loadUserTimeout 10s | as noted | bounded retry / surface | retry <=3, timeout 10s |
| Logout cannot clear token | `clearSessionFile` onError / after fileIoTimeout | LoggingOut->LoggedOut (best-effort) | in-memory identity dropped regardless | Residual: stale token; mitigated by HMAC + expiry + resume re-validation |

### 5.5 CommandExecution (`crm.commands`) - `machines/CommandExecution.machine.json`

Operational-envelope machine: the per-invocation lifecycle of the `crm` binary. Owns the single write Tx.
Home of the LadybugDB open/write/timeout failures from ARCHITECTURE.md section 6. `Parsing`, `Authorizing`,
and `Rendering` are pure (no I/O) so they use `always`; `Authorizing` is the single call site of the pure
`crm.authz` decision (the four `rbac-*` invariants).

**Lifecycle.** `Parsing` validates argv; `Opening` opens the DB (lock -> `DBLocked` bounded retry; corrupt
-> `Corrupt` fatal; unavailable/timeout -> `DBError`). `ResolvingSession` resolves the caller (delegates to
the Session machine; no-session/expired -> `Denied`; locked -> `DBLocked`). `Authorizing` calls authz; deny
-> `Denied`. `Executing` runs the domain mutation inside the Tx (BEGIN -> aggregate machine + SaveX ->
COMMIT); constraint -> `ValidationFailed`; locked/conflict -> `DBLocked` (retry whole Tx); disk-full/timeout
-> `DBError`; all with `ensureRolledBack`. `Rendering` prints and reaches `Done`. Five terminal states set
the process exit code.

**States.** `Parsing`, `Opening`, `DBLocked`, `ResolvingSession`, `Authorizing`, `Executing`, `Rendering`;
terminal (`final`): `Done`, `Denied`, `ValidationFailed`, `DBError`, `Corrupt`.

**Named-unit contract table.**

| name | kind | signature | pre / post | maps to |
|---|---|---|---|---|
| `openDatabase` | actor | `(input{dbPath}) -> Tx \| err{ErrLocked,ErrCorrupt,ErrUnavailable}` | post: DB open for writing, or typed error | `crm.commands -> crm.repo -> store` (Repo.Open) |
| `resolveSession` | actor | `(input{argv}) -> Actor \| err{ErrNoSession,ErrExpired,ErrLocked,ErrUnavailable}` | post: current User resolved (delegates to Session machine) | `crm.commands -> crm.session` (Sessions.Current) |
| `executeInTx` | actor | `(input{verb,entityType,actor}) -> Result \| err{ErrConstraint,ErrConflict,ErrDiskFull,ErrTimeout,ErrLocked}` | pre: authorized, tx begun. post: BEGIN -> domain mutation -> COMMIT atomically; on err rolled back, no partial write | `crm.commands -> crm.repo` (Tx) with `crm.domain -> crm.repo -> store` |
| `guardParseOk` | guard | `(ctx,evt)->bool` | true iff argv parses to a valid (verb, entity, flags) | input validation |
| `guardAuthorized` | guard | `(ctx,evt)->bool` | true iff pure authz `Decision.Allowed` for (actor,verb,entityType,ownerId,teamId) | `rbac-crud-verbs`,`rbac-read-visibility`,`rbac-write-scope`,`rbac-reassign-authority` |
| `phaseIsOpen/phaseIsExecute` | guard | `(ctx)->bool` | true iff `ctx.phase` is that phase (routes retry to the right step) | - |
| `isErrLocked/isErrCorrupt/isErrUnavailable/isErrNoSession/isErrExpired/isErrConstraint/isErrConflict/isErrDiskFull/isErrTimeout` | guard | `(ctx,evt)->bool` | true iff `evt.error` is that typed error | section 4.6 error types |
| `retriesExhausted` | guard | `(ctx)->bool` | true iff `retries>=3` | retry bound |
| `captureArgs` | action | `(ctx,evt)->ctx` | `verb,entityType,targetOwnerId,targetTeamId := parsed argv` | - |
| `setPhaseOpen/setPhaseExecute` | action | `(ctx)->ctx` | `phase := open \| execute` (entry action) | - |
| `captureTx/captureActor/captureResult` | action | `(ctx,evt)->ctx` | record tx handle / resolved actor / result | - |
| `incrementRetries` | action | `(ctx)->ctx` | `retries:=retries+1` | - |
| `ensureRolledBack` | action | `(ctx)->ctx` | roll back the write Tx (idempotent; store guarantees no partial write) | atomicity |
| `renderOutput` | action | `(ctx)->ctx` | format tables/JSON to stdout (entry of Rendering) | - |
| `recordAllowed/recordDenyReason` | action | `(ctx,evt)->ctx` | record authz outcome | `rbac-*` surfacing |
| `recordError/recordCorrupt/recordUnavailable/recordOpenError/recordNeedLogin/recordSessionError/recordConstraint/recordConflict/recordDiskFull/recordTimeout/recordExecuteError/recordLockExhausted` | action | `(ctx,evt)->ctx` | `lastError:=classified error` | maps repo/session errors |
| `recordSuccessExit/recordDeniedExit/recordValidationExit/recordDBErrorExit/recordCorruptExit` | action | `(ctx)->ctx` | set process `exitCode` (entry of each terminal state) | CLI exit contract |

**Failure catalog** (every ARCHITECTURE.md section 6 row lands here or in Session 5.4).

| failure (section 6 row) | detection | transition | recovery | bound / residual |
|---|---|---|---|---|
| DB open: locked by another `crm` | `openDatabase` onError `isErrLocked` | Opening->DBLocked->Opening, then DBError when retriesExhausted | bounded retry then clear message | retry <=3 ~1.5s. Residual: exit "database busy" |
| DB open: corrupt / version-incompatible | onError `isErrCorrupt` | Opening->Corrupt (final) | fail loudly; tell user to `crm restore` | fatal, no auto-recovery. Residual: restore from backup |
| DB open: unavailable / open timeout | onError `isErrUnavailable` / after openTimeout 5s | Opening->DBError (final) | fail loudly | Residual: environment/permissions |
| Session missing / expired | `resolveSession` onError `isErrNoSession`/`isErrExpired` | ResolvingSession->Denied (final) | require `crm login` | Residual: none |
| Store locked during session resolve | onError `isErrLocked` | ResolvingSession->DBLocked (phase=open) | bounded retry | Residual: as open-lock |
| Session resolve unavailable / timeout | onError catch-all / after sessionResolveTimeout 5s | ResolvingSession->DBError (final) | fail loudly | Residual: none |
| Authorization denied | `guardAuthorized` false | Authorizing->Denied (final) | none; caller lacks verb/scope | `rbac-*`. Residual: none |
| DB write: constraint / Cypher violation | `executeInTx` onError `isErrConstraint` | Executing->ValidationFailed (final), ensureRolledBack | surface validation error | one write Tx. Residual: none |
| DB write: disk full | onError `isErrDiskFull` | Executing->DBError (final), ensureRolledBack | fail loudly; DB consistent | atomic. Residual: free disk |
| DB query/write: runaway / timeout | onError `isErrTimeout` / after queryTimeout 10s | Executing->DBError (final), ensureRolledBack | abort, surface, roll back | SetTimeout+Interrupt 10s. Residual: none |
| DB write: locked / conflict mid-Tx | onError `isErrLocked`/`isErrConflict` | Executing->DBLocked (phase=execute)->retry Executing | bounded retry of whole Tx | retry <=3. Residual: refused after 3 |
| Bad CLI args | `guardParseOk` false | Parsing->ValidationFailed (final) | show usage/help | Residual: none |

### 5.6 Non-machines (records and the pure authz function)

Source: `design/machines/README.md`. These have no lifecycle machine and are built to a contract, not a
transition oracle.

**Six pure-record entities** (CRUD over the graph + invariant checks; no status enum, no transitions):

- **Contact** - owner fixed at create (`contact-owned`, structural). CRUD only.
- **Account** - owner fixed at create (`account-owned`, structural); groups Contacts/Deals but does not
  transition.
- **Pipeline** - a namespace for Deals; its only rule, `one-default-pipeline`, is a cross-record invariant
  enforced transactionally by the `setDefault` operation (not a per-record lifecycle).
- **Activity** - append-only log (`activity-immutable`); only `log` and `delete` (correction). No status.
- **Tag** - freeform label with no lifecycle; `tag-name-unique` is a DB uniqueness constraint; create/apply/
  remove are plain CRUD.
- **Team** - grouping record for visibility scope; `team-name-unique` is a DB uniqueness constraint;
  create/rename are plain CRUD.

**Authorization is a pure decision function, not a machine.** `crm.authz` is a pure
`(actor, verb, entityType, ownerId, teamId) -> Decision` with no I/O. Its four invariants (`rbac-crud-verbs`,
`rbac-read-visibility`, `rbac-write-scope`, `rbac-reassign-authority`) are enforced at a single call site:
the `guardAuthorized` guard on `CommandExecution.Authorizing`, with domain-level re-checks in the
Deal/Task/User guards (`guardCanReopen`, `guardCanReassign`, `guardAdminAuthority`) per the "authorization
is enforced in crm.domain, never in the command layer" rule. It gets a contract spec and contract tests
(C-AUTHZ-*), not a machine.

## 6. Traceability matrix

Every one of the 24 invariants, with its enforcement point (machine guard / structural / DB-constraint /
operation-level), owning component, the interface contract that carries it (section 4.6), and its test ids
(section 7). No invariant is dropped. The one invariant enforced by neither a guard nor a structural
guarantee (`one-default-pipeline`) is called out as a named residual (also section 11).

| invariant | enforced by (class) | in component | interface contract | test id(s) |
|---|---|---|---|---|
| `username-unique` | DB-constraint (unique index on `username`) | crm.repo | `Repo.SaveUser` -> ErrConstraint | P-username-unique, C-REPO-17 |
| `password-hashed` | structural (only argon2id hashes ever written; no action stores plaintext) | crm.session | `Sessions.Login` / `verifyCredentials`, `setCredentials` | P-password-hashed, C-SESS-10 |
| `disabled-cannot-auth` | machine guard (`guardUserDisabled`; `isErrDisabled`) | crm.session | `Sessions.Login` -> ErrDisabled | T-SESS-05, T-SESS-08, P-disabled-cannot-auth, C-SESS-03 |
| `single-team` | structural (data model: User has at most one Team relationship) | crm.repo / crm.domain | `Repo.SaveUser` (write discipline) | P-single-team |
| `team-name-unique` | DB-constraint (unique index on `Team.name`) | crm.repo | `Repo.SaveTeam` -> ErrConstraint | P-team-name-unique, C-REPO-18 |
| `account-owned` | structural (owner set at create; required n:1) | crm.domain / crm.repo | `Repo.SaveAccount` | P-account-owned |
| `contact-owned` | structural (owner set at create; required n:1) | crm.domain / crm.repo | `Repo.SaveContact` | P-contact-owned |
| `deal-owned` | structural (owner set at create; immutable under advance/win/lose/reopen) | crm.domain | `saveDeal` / `Repo.SaveDeal` | P-deal-owned |
| `deal-amount-nonneg` | machine guard (`guardCanAdvance`/`guardCanWin`/`guardCanLose`) | crm.domain | `saveDeal` | T-DEAL-01..06,10..13,17..20,23..26, P-deal-amount-nonneg |
| `deal-stage-forward` | machine guard + structural (`guardCanAdvance`; Negotiation no forward; reopen exception) | crm.domain | `saveDeal` | T-DEAL-01,02,08,09,15,16,22,28,33, P-deal-stage-forward |
| `deal-terminal` | structural (Won/Lost expose only reopen; others rejected) | crm.domain | `saveDeal` | T-DEAL-30,31,32,35,36,37, P-deal-terminal |
| `deal-won-has-closedate` | machine guard (`guardCanWin`; `commitCloseDate`) | crm.domain | `saveDeal` | T-DEAL-03,04,10,11,17,18,23,24,41, P-deal-won-has-closedate |
| `one-default-pipeline` | **operation-level** (setDefault atomic read-modify-write; NOT a guard, NOT structural) | crm.domain setDefault op / crm.repo | `Repo.SetDefaultPipeline` | P-one-default-pipeline, C-REPO-20 |
| `activity-immutable` | structural (no update action exists; append-only) | crm.domain / crm.repo | `Repo.SaveActivity` (no update path) | P-activity-immutable, C-REPO-23 |
| `activity-owned` | structural (`log` records the acting User) | crm.domain | `Repo.SaveActivity` | P-activity-owned |
| `task-owned` | structural + machine guard (owner set at create; `guardCanReassign` admits one in-scope owner) | crm.domain | `saveTask` | T-TASK-07,08,14,15, P-task-owned |
| `task-terminal` | structural (Done/Cancelled are `final`; no reopen) | crm.domain | `saveTask` | T-TASK-16,17, P-task-terminal |
| `task-assignee-visible` | machine guard (`guardCanReassign`) | crm.domain | `saveTask` | T-TASK-07,08,14,15, P-task-assignee-visible |
| `tag-name-unique` | DB-constraint (unique index on `Tag.name`) | crm.repo | `Repo.SaveTag` -> ErrConstraint | P-tag-name-unique, C-REPO-19 |
| `rbac-crud-verbs` | machine guard (`guardAuthorized`; User `guardAdminAuthority`) | crm.authz + crm.domain | `Authorizer.Authorize` | T-CMD-18,19, T-USER-01,02,04,05, C-AUTHZ-01..03, P-rbac-crud-verbs |
| `rbac-read-visibility` | machine guard (`guardAuthorized`, reads authorized too) | crm.authz | `Authorizer.Authorize` | T-CMD-18,19, C-AUTHZ-04..07, P-rbac-read-visibility |
| `rbac-write-scope` | machine guard (`guardAuthorized`; domain `guardCan*` re-checks) | crm.authz + crm.domain | `Authorizer.Authorize` | T-CMD-18,19, C-AUTHZ-08,09, P-rbac-write-scope |
| `rbac-reassign-authority` | machine guard (`guardAuthorized`; Deal `guardCanReopen`; Task `guardCanReassign`) | crm.authz + crm.domain | `Authorizer.Authorize` | T-CMD-18,19, T-DEAL-28,29,33,34, T-TASK-07,08,14,15, C-AUTHZ-10..12, P-rbac-reassign-authority |
| `session-active-user` | machine guard (`guardSessionUserActive`) | crm.session | `Sessions.Current` | T-SESS-23,24, P-session-active-user, C-SESS-08 |

**Known / named risk.** `one-default-pipeline` is the only invariant with no enforcing machine guard and no
structural guarantee: Pipeline has no lifecycle machine, and nothing in any state graph prevents zero or two
defaults. It is enforced solely by the `setDefault` operation as an atomic read-modify-write inside the one
write Tx (unset the prior default, set the new one). Coverage is the operation-level property test
**P-one-default-pipeline** asserting the post-condition `count(isDefault==true) == 1`, plus the repo-level
check **C-REPO-20**. Carried explicitly in section 11.

## 7. Test specification (the hard-TDD oracle)

This section is the input to the test-writer agent (section 10). It writes tests from here; it does not
invent them. Sources: the five `design/machines/*.matrix.md` transition matrices (flattened 1:1 below),
the section 4.6 interface contracts, and the section 3.4 invariants.

**Test id scheme.** `T-<MACHINE>-NN` = one transition/guard-branch row (MACHINE in DEAL, TASK, USER, SESS,
CMD; NN is the matrix row number, so `T-DEAL-07` is Deal.matrix row 7). `C-<BOUNDARY>-NN` = a contract test
at a section-4.6 boundary (REPO, AUTHZ, SESS) plus `C-ARCH-01` for the dependency contract. `P-<invariant>`
= a property test, one per invariant.

**Guard-branch completeness note.** Each guard-false row whose guard is a conjunction (for example
`guardCanAdvance` = next-stage AND may-write AND amount>=0) must be instantiated once per falsifying clause
(a/b/c). The property tests (7.3) pin the invariant-level clauses; the transition rows pin the routing. A
row is not "covered" until every falsifying clause of its guard has a case.

**Covering-path completeness.** Because the machines are XState v5 JSON, the test-writer may load each
`machines/<M>.machine.json` into `@xstate/graph` and call `getShortestPaths` / `getSimplePaths` /
`getAdjacencyMap` to enumerate every edge (event + guard branch) and confirm the flattened rows below cover
the full adjacency map with no transition or guard branch dropped. The flattened tables are the canonical
oracle; the covering paths are the completeness check. Test-writer procedure: (1) generate the adjacency
map per machine, (2) assert one T-row exists per edge, (3) fail the suite build if any edge lacks a row.

### 7.1 Transition tests (flattened matrices)

Columns: test id | component | given state + context | event / trigger | expected next state | expected actions | derived from.

**Deal (`crm.domain`, `machines/Deal.machine.json`) - 57 rows.**

| test id | given state + context | event / trigger | next state | actions | derived from |
|---|---|---|---|---|---|
| T-DEAL-01 | Lead; caller may write; amount>=0 | advanceStage | persisting | setPendingAdvance | advanceStage / deal-stage-forward,rbac-write-scope,deal-amount-nonneg |
| T-DEAL-02 | Lead; guard false (not-writable OR amount<0) | advanceStage | Lead (internal) | recordAdvanceDenied | guard false |
| T-DEAL-03 | Lead; closeDate present; may write; amount>=0 | win | persisting | setPendingWin | win / deal-won-has-closedate,rbac-write-scope,deal-amount-nonneg |
| T-DEAL-04 | Lead; guard false (no closeDate OR not-writable OR amount<0) | win | Lead (internal) | recordWinDenied | guard false |
| T-DEAL-05 | Lead; may write; amount>=0 | lose | persisting | setPendingLose | lose / rbac-write-scope,deal-amount-nonneg |
| T-DEAL-06 | Lead; guard false | lose | Lead (internal) | recordLoseDenied | guard false |
| T-DEAL-07 | Lead (non-terminal) | reopen | Lead (internal) | recordReopenNotTerminal | deal-terminal (reopen only on terminal) |
| T-DEAL-08 | Qualified; may write; amount>=0 | advanceStage | persisting | setPendingAdvance (->Proposal) | deal-stage-forward,rbac-write-scope,deal-amount-nonneg |
| T-DEAL-09 | Qualified; guard false | advanceStage | Qualified (internal) | recordAdvanceDenied | guard false |
| T-DEAL-10 | Qualified; closeDate; may write; amount>=0 | win | persisting | setPendingWin | deal-won-has-closedate,rbac-write-scope,deal-amount-nonneg |
| T-DEAL-11 | Qualified; guard false | win | Qualified (internal) | recordWinDenied | guard false |
| T-DEAL-12 | Qualified; may write; amount>=0 | lose | persisting | setPendingLose | rbac-write-scope,deal-amount-nonneg |
| T-DEAL-13 | Qualified; guard false | lose | Qualified (internal) | recordLoseDenied | guard false |
| T-DEAL-14 | Qualified (non-terminal) | reopen | Qualified (internal) | recordReopenNotTerminal | deal-terminal |
| T-DEAL-15 | Proposal; may write; amount>=0 | advanceStage | persisting | setPendingAdvance (->Negotiation) | deal-stage-forward,rbac-write-scope,deal-amount-nonneg |
| T-DEAL-16 | Proposal; guard false | advanceStage | Proposal (internal) | recordAdvanceDenied | guard false |
| T-DEAL-17 | Proposal; closeDate; may write; amount>=0 | win | persisting | setPendingWin | deal-won-has-closedate,rbac-write-scope,deal-amount-nonneg |
| T-DEAL-18 | Proposal; guard false | win | Proposal (internal) | recordWinDenied | guard false |
| T-DEAL-19 | Proposal; may write; amount>=0 | lose | persisting | setPendingLose | rbac-write-scope,deal-amount-nonneg |
| T-DEAL-20 | Proposal; guard false | lose | Proposal (internal) | recordLoseDenied | guard false |
| T-DEAL-21 | Proposal (non-terminal) | reopen | Proposal (internal) | recordReopenNotTerminal | deal-terminal |
| T-DEAL-22 | Negotiation (no forward stage) | advanceStage | Negotiation (internal) | recordAdvanceDenied | deal-stage-forward (win/lose only) |
| T-DEAL-23 | Negotiation; closeDate; may write; amount>=0 | win | persisting | setPendingWin | deal-won-has-closedate,rbac-write-scope,deal-amount-nonneg |
| T-DEAL-24 | Negotiation; guard false | win | Negotiation (internal) | recordWinDenied | guard false |
| T-DEAL-25 | Negotiation; may write; amount>=0 | lose | persisting | setPendingLose | rbac-write-scope,deal-amount-nonneg |
| T-DEAL-26 | Negotiation; guard false | lose | Negotiation (internal) | recordLoseDenied | guard false |
| T-DEAL-27 | Negotiation (non-terminal) | reopen | Negotiation (internal) | recordReopenNotTerminal | deal-terminal |
| T-DEAL-28 | Won; caller Manager/Admin in scope | reopen | persisting | setPendingReopen (->Negotiation) | reopen / rbac-reassign-authority,rbac-write-scope |
| T-DEAL-29 | Won; guard false (not Manager/Admin in scope) | reopen | Won (internal) | recordReopenDenied | guard false |
| T-DEAL-30 | Won (terminal) | advanceStage | Won (internal) | recordTerminalRejected | deal-terminal (structural) |
| T-DEAL-31 | Won (terminal) | win | Won (internal) | recordTerminalRejected | deal-terminal (structural) |
| T-DEAL-32 | Won (terminal) | lose | Won (internal) | recordTerminalRejected | deal-terminal (structural) |
| T-DEAL-33 | Lost; caller Manager/Admin in scope | reopen | persisting | setPendingReopen (->Negotiation) | reopen / rbac-reassign-authority,rbac-write-scope |
| T-DEAL-34 | Lost; guard false | reopen | Lost (internal) | recordReopenDenied | guard false |
| T-DEAL-35 | Lost (terminal) | advanceStage | Lost (internal) | recordTerminalRejected | deal-terminal (structural) |
| T-DEAL-36 | Lost (terminal) | win | Lost (internal) | recordTerminalRejected | deal-terminal (structural) |
| T-DEAL-37 | Lost (terminal) | lose | Lost (internal) | recordTerminalRejected | deal-terminal (structural) |
| T-DEAL-38 | persisting; pendingStage=Qualified | saveDeal onDone | Qualified | commitStage | persist success routing |
| T-DEAL-39 | persisting; pendingStage=Proposal | saveDeal onDone | Proposal | commitStage | persist success routing |
| T-DEAL-40 | persisting; pendingStage=Negotiation | saveDeal onDone | Negotiation | commitStage | persist success routing |
| T-DEAL-41 | persisting; pendingStage=Won | saveDeal onDone | Won | commitStage, commitCloseDate | persist success; deal-won-has-closedate |
| T-DEAL-42 | persisting; pendingStage=Lost | saveDeal onDone | Lost | commitStage | persist success routing |
| T-DEAL-43 | persisting; pending routes to none | saveDeal onDone (else) | rolledBack | recordRoutingError | defensive |
| T-DEAL-44 | persisting; error=ErrLocked | saveDeal onError | persistRetry | recordError | store-locked |
| T-DEAL-45 | persisting; error=ErrConstraint | saveDeal onError | rolledBack | recordConstraint | constraint |
| T-DEAL-46 | persisting; error=ErrDiskFull | saveDeal onError | rolledBack | recordDiskFull | disk-full |
| T-DEAL-47 | persisting; error=ErrTimeout | saveDeal onError | rolledBack | recordTimeout | timeout |
| T-DEAL-48 | persisting; error=other | saveDeal onError (else) | rolledBack | recordUnknownError | catch-all |
| T-DEAL-49 | persisting; 10s elapsed | after persistTimeout | rolledBack | recordTimeout | timeout 10s |
| T-DEAL-50 | persistRetry; retries>=3 | always | rolledBack | recordRetriesExhausted | retry bound |
| T-DEAL-51 | persistRetry; retries<3 | after persistRetryBackoff | persisting | incrementRetries | backoff ~0.5s |
| T-DEAL-52 | rolledBack; priorStage=Lead | always | Lead | - | atomic rollback |
| T-DEAL-53 | rolledBack; priorStage=Qualified | always | Qualified | - | atomic rollback |
| T-DEAL-54 | rolledBack; priorStage=Proposal | always | Proposal | - | atomic rollback |
| T-DEAL-55 | rolledBack; priorStage=Negotiation | always | Negotiation | - | atomic rollback |
| T-DEAL-56 | rolledBack; priorStage=Won | always | Won | - | atomic rollback |
| T-DEAL-57 | rolledBack; priorStage=Lost | always | Lost | - | atomic rollback |

**Task (`crm.domain`, `machines/Task.machine.json`) - 32 rows.**

| test id | given state + context | event / trigger | next state | actions | derived from |
|---|---|---|---|---|---|
| T-TASK-01 | Open; caller may write | start | persisting | setPendingStart | start / rbac-write-scope |
| T-TASK-02 | Open; guard false | start | Open (internal) | recordStartDenied | guard false |
| T-TASK-03 | Open; non-terminal; may write | complete | persisting | setPendingComplete | complete / task-terminal,rbac-write-scope |
| T-TASK-04 | Open; guard false | complete | Open (internal) | recordCompleteDenied | guard false |
| T-TASK-05 | Open; non-terminal; may write | cancel | persisting | setPendingCancel | cancel / task-terminal,rbac-write-scope |
| T-TASK-06 | Open; guard false | cancel | Open (internal) | recordCancelDenied | guard false |
| T-TASK-07 | Open; new assignee in scope; caller Manager/Admin in scope | reassign | persisting | setPendingReassign | reassign / task-assignee-visible,rbac-reassign-authority,rbac-write-scope |
| T-TASK-08 | Open; guard false (assignee out of scope OR caller not Mgr/Admin) | reassign | Open (internal) | recordReassignDenied | guard false |
| T-TASK-09 | InProgress (already started) | start | InProgress (internal) | recordAlreadyStarted | idempotent no-op |
| T-TASK-10 | InProgress; non-terminal; may write | complete | persisting | setPendingComplete | complete / task-terminal,rbac-write-scope |
| T-TASK-11 | InProgress; guard false | complete | InProgress (internal) | recordCompleteDenied | guard false |
| T-TASK-12 | InProgress; non-terminal; may write | cancel | persisting | setPendingCancel | cancel / task-terminal,rbac-write-scope |
| T-TASK-13 | InProgress; guard false | cancel | InProgress (internal) | recordCancelDenied | guard false |
| T-TASK-14 | InProgress; assignee in scope; caller Mgr/Admin in scope | reassign | persisting | setPendingReassign | reassign / task-assignee-visible,rbac-reassign-authority,rbac-write-scope |
| T-TASK-15 | InProgress; guard false | reassign | InProgress (internal) | recordReassignDenied | guard false |
| T-TASK-16 | Done (final) | any of start/complete/cancel/reassign | none (final) | - | task-terminal (structural) |
| T-TASK-17 | Cancelled (final) | any of start/complete/cancel/reassign | none (final) | - | task-terminal (structural) |
| T-TASK-18 | persisting; pendingStatus=Open | saveTask onDone | Open | commitStatus | reassign-in-Open routing |
| T-TASK-19 | persisting; pendingStatus=InProgress | saveTask onDone | InProgress | commitStatus | start / reassign-in-InProgress routing |
| T-TASK-20 | persisting; pendingStatus=Done | saveTask onDone | Done | commitStatus | complete routing |
| T-TASK-21 | persisting; pendingStatus=Cancelled | saveTask onDone | Cancelled | commitStatus | cancel routing |
| T-TASK-22 | persisting; pending routes to none | saveTask onDone (else) | rolledBack | recordRoutingError | defensive |
| T-TASK-23 | persisting; error=ErrLocked | saveTask onError | persistRetry | recordError | store-locked |
| T-TASK-24 | persisting; error=ErrConstraint | saveTask onError | rolledBack | recordConstraint | constraint |
| T-TASK-25 | persisting; error=ErrDiskFull | saveTask onError | rolledBack | recordDiskFull | disk-full |
| T-TASK-26 | persisting; error=ErrTimeout | saveTask onError | rolledBack | recordTimeout | timeout |
| T-TASK-27 | persisting; error=other | saveTask onError (else) | rolledBack | recordUnknownError | catch-all |
| T-TASK-28 | persisting; 10s elapsed | after persistTimeout | rolledBack | recordTimeout | timeout 10s |
| T-TASK-29 | persistRetry; retries>=3 | always | rolledBack | recordRetriesExhausted | retry bound |
| T-TASK-30 | persistRetry; retries<3 | after persistRetryBackoff | persisting | incrementRetries | backoff ~0.5s |
| T-TASK-31 | rolledBack; priorStatus=Open | always | Open | - | atomic rollback |
| T-TASK-32 | rolledBack; priorStatus=InProgress | always | InProgress | - | atomic rollback |

**User (`crm.domain`, `machines/User.machine.json`) - 19 rows.**

| test id | given state + context | event / trigger | next state | actions | derived from |
|---|---|---|---|---|---|
| T-USER-01 | Active; actor.role==Admin | disable | persisting | setPendingDisable | disable / rbac-crud-verbs |
| T-USER-02 | Active; actor not Admin | disable | Active (internal) | recordAuthorityDenied | guard false |
| T-USER-03 | Active | enable | Active (internal) | recordAlreadyActive | idempotent no-op |
| T-USER-04 | Disabled; actor.role==Admin | enable | persisting | setPendingEnable | enable / rbac-crud-verbs |
| T-USER-05 | Disabled; actor not Admin | enable | Disabled (internal) | recordAuthorityDenied | guard false |
| T-USER-06 | Disabled | disable | Disabled (internal) | recordAlreadyDisabled | idempotent no-op |
| T-USER-07 | persisting; pendingStatus=Active | saveUser onDone | Active | commitStatus | enable routing |
| T-USER-08 | persisting; pendingStatus=Disabled | saveUser onDone | Disabled | commitStatus | disable routing |
| T-USER-09 | persisting; pending routes to none | saveUser onDone (else) | rolledBack | recordRoutingError | defensive |
| T-USER-10 | persisting; error=ErrLocked | saveUser onError | persistRetry | recordError | store-locked |
| T-USER-11 | persisting; error=ErrConstraint | saveUser onError | rolledBack | recordConstraint | constraint |
| T-USER-12 | persisting; error=ErrDiskFull | saveUser onError | rolledBack | recordDiskFull | disk-full |
| T-USER-13 | persisting; error=ErrTimeout | saveUser onError | rolledBack | recordTimeout | timeout |
| T-USER-14 | persisting; error=other | saveUser onError (else) | rolledBack | recordUnknownError | catch-all |
| T-USER-15 | persisting; 10s elapsed | after persistTimeout | rolledBack | recordTimeout | timeout 10s |
| T-USER-16 | persistRetry; retries>=3 | always | rolledBack | recordRetriesExhausted | retry bound |
| T-USER-17 | persistRetry; retries<3 | after persistRetryBackoff | persisting | incrementRetries | backoff ~0.5s |
| T-USER-18 | rolledBack; priorStatus=Active | always | Active | - | atomic rollback |
| T-USER-19 | rolledBack; priorStatus=Disabled | always | Disabled | - | atomic rollback |

**Session (`crm.session`, `machines/Session.machine.json`) - 60 rows.**

| test id | given state + context | event / trigger | next state | actions | derived from |
|---|---|---|---|---|---|
| T-SESS-01 | Anonymous | login | Authenticating | setCredentials | login |
| T-SESS-02 | Anonymous | resume | Resolving | - | Current() resolution |
| T-SESS-03 | Anonymous | logout | Anonymous (internal) | recordNoSessionToLogout | explicit ignore |
| T-SESS-04 | Anonymous | useSession | Anonymous (internal) | recordNoActiveSession | explicit reject |
| T-SESS-05 | Authenticating; verified user Disabled | verifyCredentials onDone | AuthDenied | recordDisabled | disabled-cannot-auth |
| T-SESS-06 | Authenticating; verified user Active | verifyCredentials onDone (else) | WritingSession | captureUser | verify ok |
| T-SESS-07 | Authenticating; error=ErrBadCredentials | verifyCredentials onError | AuthFailed | recordBadCredentials | ErrBadCredentials |
| T-SESS-08 | Authenticating; error=ErrDisabled | verifyCredentials onError | AuthDenied | recordDisabled | disabled-cannot-auth |
| T-SESS-09 | Authenticating; error=ErrLocked | verifyCredentials onError | VerifyRetry | recordError | store-locked |
| T-SESS-10 | Authenticating; error=other | verifyCredentials onError (else) | SessionUnavailable | recordVerifyError | store unavailable/corrupt |
| T-SESS-11 | Authenticating; 5s elapsed | after verifyTimeout | SessionUnavailable | recordTimeout | verify timeout 5s |
| T-SESS-12 | VerifyRetry; retries>=3 | always | SessionUnavailable | recordRetriesExhausted | retry bound |
| T-SESS-13 | VerifyRetry; retries<3 | after verifyRetryBackoff | Authenticating | incrementRetries | backoff ~0.5s |
| T-SESS-14 | WritingSession | writeSessionFile onDone | Active | - | token written |
| T-SESS-15 | WritingSession | writeSessionFile onError | SessionUnavailable | recordFileError | token write failed |
| T-SESS-16 | WritingSession; 2s elapsed | after fileIoTimeout | SessionUnavailable | recordTimeout | file io timeout 2s |
| T-SESS-17 | Resolving; token expiresAt<=now | readSessionFile onDone | Expired | recordExpired | token expiry |
| T-SESS-18 | Resolving; token valid | readSessionFile onDone (else) | CheckingUser | captureToken | token valid |
| T-SESS-19 | Resolving; error=ErrNoSession | readSessionFile onError | Anonymous | recordNoSession | ErrNoSession |
| T-SESS-20 | Resolving; error=ErrExpired | readSessionFile onError | Expired | recordExpired | ErrExpired |
| T-SESS-21 | Resolving; error=other | readSessionFile onError (else) | SessionUnavailable | recordFileError | token unreadable |
| T-SESS-22 | Resolving; 2s elapsed | after fileIoTimeout | SessionUnavailable | recordTimeout | file io timeout 2s |
| T-SESS-23 | CheckingUser; loaded user Active | loadUser onDone | Active | captureUser | session-active-user |
| T-SESS-24 | CheckingUser; loaded user not Active | loadUser onDone (else) | Invalidated | recordUserNotActive | session-active-user (deny) |
| T-SESS-25 | CheckingUser; error=ErrLocked | loadUser onError | VerifyRetry | recordError | store-locked |
| T-SESS-26 | CheckingUser; error=ErrNotFound | loadUser onError | Invalidated | recordUserMissing | user deleted |
| T-SESS-27 | CheckingUser; error=other | loadUser onError (else) | SessionUnavailable | recordLoadError | store unavailable |
| T-SESS-28 | CheckingUser; 10s elapsed | after loadUserTimeout | SessionUnavailable | recordTimeout | query timeout 10s |
| T-SESS-29 | Active | logout | LoggingOut | - | logout |
| T-SESS-30 | Active | useSession | Active (internal) | recordSessionUsed | command uses session |
| T-SESS-31 | Active | login | Active (internal) | recordAlreadyActive | explicit ignore |
| T-SESS-32 | Active | resume | Active (internal) | recordAlreadyResolved | explicit ignore |
| T-SESS-33 | Active; 8h elapsed | after sessionTTL | Expired | recordExpired | token expiry |
| T-SESS-34 | LoggingOut | clearSessionFile onDone | LoggedOut | - | token cleared |
| T-SESS-35 | LoggingOut | clearSessionFile onError | LoggedOut | recordLogoutWarning | best-effort logout |
| T-SESS-36 | LoggingOut; 2s elapsed | after fileIoTimeout | LoggedOut | recordLogoutWarning | best-effort logout |
| T-SESS-37 | Expired | login | Authenticating | setCredentials | re-auth |
| T-SESS-38 | Expired | resume | Expired (internal) | recordExpiredNeedsLogin | explicit reject |
| T-SESS-39 | Expired | logout | Expired (internal) | recordNoSessionToLogout | explicit ignore |
| T-SESS-40 | Expired | useSession | Expired (internal) | recordSessionExpired | explicit reject |
| T-SESS-41 | LoggedOut | login | Authenticating | setCredentials | re-auth |
| T-SESS-42 | LoggedOut | resume | LoggedOut (internal) | recordNoSession | explicit reject |
| T-SESS-43 | LoggedOut | logout | LoggedOut (internal) | recordNoSessionToLogout | explicit ignore |
| T-SESS-44 | LoggedOut | useSession | LoggedOut (internal) | recordNoActiveSession | explicit reject |
| T-SESS-45 | AuthFailed | login | Authenticating | setCredentials | retry auth |
| T-SESS-46 | AuthFailed | resume | AuthFailed (internal) | recordNoSession | explicit reject |
| T-SESS-47 | AuthFailed | logout | AuthFailed (internal) | recordNoSessionToLogout | explicit ignore |
| T-SESS-48 | AuthFailed | useSession | AuthFailed (internal) | recordNoActiveSession | explicit reject |
| T-SESS-49 | AuthDenied | login | Authenticating | setCredentials | retry (denied if still disabled) |
| T-SESS-50 | AuthDenied | resume | AuthDenied (internal) | recordNoSession | explicit reject |
| T-SESS-51 | AuthDenied | logout | AuthDenied (internal) | recordNoSessionToLogout | explicit ignore |
| T-SESS-52 | AuthDenied | useSession | AuthDenied (internal) | recordNoActiveSession | explicit reject |
| T-SESS-53 | Invalidated | login | Authenticating | setCredentials | re-auth |
| T-SESS-54 | Invalidated | resume | Invalidated (internal) | recordNoSession | explicit reject |
| T-SESS-55 | Invalidated | logout | Invalidated (internal) | recordNoSessionToLogout | explicit ignore |
| T-SESS-56 | Invalidated | useSession | Invalidated (internal) | recordNoActiveSession | explicit reject |
| T-SESS-57 | SessionUnavailable | login | Authenticating | setCredentials | retry auth |
| T-SESS-58 | SessionUnavailable | resume | Resolving | - | retry resolution |
| T-SESS-59 | SessionUnavailable | logout | SessionUnavailable (internal) | recordNoSessionToLogout | explicit ignore |
| T-SESS-60 | SessionUnavailable | useSession | SessionUnavailable (internal) | recordNoActiveSession | explicit reject |

**CommandExecution (`crm.commands`, `machines/CommandExecution.machine.json`) - 33 rows.**

| test id | given state + context | event / trigger | next state | actions | derived from |
|---|---|---|---|---|---|
| T-CMD-01 | Parsing; argv valid | always | Opening | captureArgs | arg validation ok |
| T-CMD-02 | Parsing; argv invalid | always | ValidationFailed | recordParseError | bad args |
| T-CMD-03 | Opening | openDatabase onDone | ResolvingSession | captureTx | db open ok |
| T-CMD-04 | Opening; error=ErrLocked | openDatabase onError | DBLocked | recordError | open-lock |
| T-CMD-05 | Opening; error=ErrCorrupt | openDatabase onError | Corrupt | recordCorrupt | corrupt (fatal) |
| T-CMD-06 | Opening; error=ErrUnavailable | openDatabase onError | DBError | recordUnavailable | unavailable |
| T-CMD-07 | Opening; error=other | openDatabase onError (else) | DBError | recordOpenError | catch-all |
| T-CMD-08 | Opening; 5s elapsed | after openTimeout | DBError | recordTimeout | open timeout 5s |
| T-CMD-09 | DBLocked; retries>=3 | always | DBError | recordLockExhausted | retry bound |
| T-CMD-10 | DBLocked; phase=open; retries<3 | after dbRetryBackoff | Opening | incrementRetries | retry the open |
| T-CMD-11 | DBLocked; phase=execute; retries<3 | after dbRetryBackoff | Executing | incrementRetries | retry the write Tx |
| T-CMD-12 | ResolvingSession | resolveSession onDone | Authorizing | captureActor | session resolved |
| T-CMD-13 | ResolvingSession; error=ErrNoSession | resolveSession onError | Denied | recordNeedLogin | no session |
| T-CMD-14 | ResolvingSession; error=ErrExpired | resolveSession onError | Denied | recordNeedLogin | expired |
| T-CMD-15 | ResolvingSession; error=ErrLocked | resolveSession onError | DBLocked | recordError | open-lock (session hits repo) |
| T-CMD-16 | ResolvingSession; error=other | resolveSession onError (else) | DBError | recordSessionError | unavailable |
| T-CMD-17 | ResolvingSession; 5s elapsed | after sessionResolveTimeout | DBError | recordTimeout | session resolve timeout 5s |
| T-CMD-18 | Authorizing; Decision.Allowed | always | Executing | recordAllowed | rbac-crud-verbs,rbac-read-visibility,rbac-write-scope,rbac-reassign-authority |
| T-CMD-19 | Authorizing; !Decision.Allowed | always | Denied | recordDenyReason | authz deny |
| T-CMD-20 | Executing | executeInTx onDone | Rendering | captureResult | commit ok |
| T-CMD-21 | Executing; error=ErrConstraint | executeInTx onError | ValidationFailed | ensureRolledBack, recordConstraint | constraint |
| T-CMD-22 | Executing; error=ErrLocked | executeInTx onError | DBLocked | ensureRolledBack, recordError | write-lock |
| T-CMD-23 | Executing; error=ErrConflict | executeInTx onError | DBLocked | ensureRolledBack, recordConflict | conflict |
| T-CMD-24 | Executing; error=ErrDiskFull | executeInTx onError | DBError | ensureRolledBack, recordDiskFull | disk-full |
| T-CMD-25 | Executing; error=ErrTimeout | executeInTx onError | DBError | ensureRolledBack, recordTimeout | timeout |
| T-CMD-26 | Executing; error=other | executeInTx onError (else) | DBError | ensureRolledBack, recordExecuteError | catch-all |
| T-CMD-27 | Executing; 10s elapsed | after queryTimeout | DBError | ensureRolledBack, recordTimeout | query timeout 10s |
| T-CMD-28 | Rendering | always | Done | renderOutput (entry) | render + exit 0 |
| T-CMD-29 | Done (final) | - | - | recordSuccessExit | success exit |
| T-CMD-30 | Denied (final) | - | - | recordDeniedExit | authn/authz exit |
| T-CMD-31 | ValidationFailed (final) | - | - | recordValidationExit | validation exit |
| T-CMD-32 | DBError (final) | - | - | recordDBErrorExit | db-error exit |
| T-CMD-33 | Corrupt (final) | - | - | recordCorruptExit | fatal exit (restore) |

### 7.2 Contract tests (per boundary, from section 4.6)

One test per interface method x outcome. Repo tests run against a real temporary LadybugDB directory
(integration; no mocks). Authz tests are pure. Session tests use a real token file and a real (temp) repo.

**Architecture contract.**

- **C-ARCH-01** - static import check: only `internal/repo/**` imports `github.com/LadybugDB/go-ladybug`;
  `internal/cli/**` does not import `internal/authz`; every import edge is in the section 4.5 `allow` list.
  (ast-grep/go list based; fails the build on any violation.)

**Repo boundary (`crm.repo`).**

| test id | method / scenario | expected |
|---|---|---|
| C-REPO-01 | Open on a healthy dir | returns Tx, no error |
| C-REPO-02 | Open while another process holds the write lock | ErrLocked |
| C-REPO-03 | Open on a corrupt / version-incompatible dir | ErrCorrupt |
| C-REPO-04 | Open on an unreadable/absent path | ErrUnavailable |
| C-REPO-05 | BeginWrite then a write, no Commit | change not visible to a fresh Open |
| C-REPO-06 | BeginWrite, write, Commit | change durable and visible to a fresh Open |
| C-REPO-07 | BeginWrite, write, Rollback | store byte-for-byte unchanged (no partial write) |
| C-REPO-08 | GetUserByName existing | returns User row |
| C-REPO-09 | GetUserByName missing | ErrNotFound |
| C-REPO-10 | GetDeal existing | returns Deal with stage |
| C-REPO-11 | GetDeal missing | ErrNotFound |
| C-REPO-12 | SaveDeal then GetDeal | persisted stage equals written stage |
| C-REPO-13 | SaveDeal violating a Cypher/uniqueness constraint | ErrConstraint |
| C-REPO-14 | SaveDeal under a write conflict | ErrConflict |
| C-REPO-15 | SaveDeal with the disk full | ErrDiskFull |
| C-REPO-16 | SaveDeal exceeding query timeout | ErrTimeout |
| C-REPO-17 | SaveUser with a duplicate username | ErrConstraint (username-unique) |
| C-REPO-18 | SaveTeam with a duplicate name | ErrConstraint (team-name-unique) |
| C-REPO-19 | SaveTag with a duplicate name | ErrConstraint (tag-name-unique) |
| C-REPO-20 | SetDefaultPipeline(p) over N pipelines | post-condition count(isDefault==true)==1 (one-default-pipeline) |
| C-REPO-21 | Two SaveActivity of the same logical event | two immutable nodes; no in-place update path exists |
| C-REPO-22 | Idempotency: SaveDeal retried after ErrLocked | applied exactly once (no double node/edge) |
| C-REPO-23 | Attempt to mutate an existing Activity body/occurredAt | no repo method exists to do so (compile/contract) (activity-immutable) |

**Authorizer boundary (`crm.authz`, pure).**

| test id | scenario | expected |
|---|---|---|
| C-AUTHZ-01 | ReadOnly + create | Denied, Reason set (rbac-crud-verbs) |
| C-AUTHZ-02 | Rep/Manager/Admin + create | Allowed |
| C-AUTHZ-03 | ReadOnly + read in scope -> Allowed; ReadOnly + update/delete -> Denied | as stated (rbac-crud-verbs) |
| C-AUTHZ-04 | Admin + read any record | Allowed (all records) |
| C-AUTHZ-05 | Rep + read own record | Allowed (rbac-read-visibility) |
| C-AUTHZ-06 | Rep + read same-team record | Allowed |
| C-AUTHZ-07 | Rep + read other-team record | Denied (rbac-read-visibility) |
| C-AUTHZ-08 | Manager + update/delete team member's record | Allowed (rbac-write-scope) |
| C-AUTHZ-09 | Rep + update/delete a not-owned record | Denied (rbac-write-scope) |
| C-AUTHZ-10 | Manager + reassign within own team | Allowed (rbac-reassign-authority) |
| C-AUTHZ-11 | Rep/ReadOnly + reassign | Denied (rbac-reassign-authority) |
| C-AUTHZ-12 | Admin + reassign any record | Allowed |
| C-AUTHZ-13 | Any denied decision | Decision.Reason non-empty; empty when Allowed |
| C-AUTHZ-14 | Same inputs twice | identical Decision; no I/O performed (purity) |

**Session boundary (`crm.session`).**

| test id | scenario | expected |
|---|---|---|
| C-SESS-01 | Login with valid credentials | Session with future expiry; token written |
| C-SESS-02 | Login with a wrong password | ErrBadCredentials; NOT retried |
| C-SESS-03 | Login as a Disabled user | ErrDisabled (disabled-cannot-auth) |
| C-SESS-04 | Login while the store is locked | ErrLocked then bounded retry <=3 |
| C-SESS-05 | Current with a valid token | returns the User |
| C-SESS-06 | Current with no token file | ErrNoSession |
| C-SESS-07 | Current with an expired token | ErrExpired |
| C-SESS-08 | Current when the user is now Disabled | invalidated / not Active (session-active-user) |
| C-SESS-09 | Logout then Current | token cleared; Current -> ErrNoSession |
| C-SESS-10 | Inspect stored credential after register/login | only an argon2id encoded hash on disk; no plaintext (password-hashed) |

### 7.3 Property tests (one per invariant, 24)

Each is a randomized/generative property over the relevant operation. Format: `P-<invariant>` - property.

| test id | property |
|---|---|
| P-username-unique | For any two register attempts with the same username, the second fails; the store never holds two Users with one username. |
| P-password-hashed | For any password, the persisted `passwordHash` is a valid argon2id encoding and never equals the plaintext; no plaintext appears in the DB or token file. |
| P-disabled-cannot-auth | For any Disabled user and any password, Login never yields a Session (ErrDisabled). |
| P-single-team | For any sequence of team assignments, a User is a member of at most one Team. |
| P-team-name-unique | For any two Team creates with the same name, the second fails; never two Teams with one name. |
| P-account-owned | Every persisted Account has exactly one non-empty ownerId. |
| P-contact-owned | Every persisted Contact has exactly one non-empty ownerId. |
| P-deal-owned | Every persisted Deal has exactly one ownerId, unchanged by advance/win/lose/reopen. |
| P-deal-amount-nonneg | No accepted Deal transition ever persists amountCents < 0; a create/transition with amount<0 is rejected. |
| P-deal-stage-forward | For any accepted non-reopen transition, the new stage index is strictly greater than the old, or is Won/Lost; reopen is the only backward move and only from Won/Lost. |
| P-deal-terminal | From Won/Lost, no event other than reopen changes the deal. |
| P-deal-won-has-closedate | Every Deal in Won has a non-null closeDate. |
| P-one-default-pipeline | After any sequence of pipeline create/setDefault operations, `count(isDefault==true) == 1` (named residual; operation-level). |
| P-activity-immutable | For any logged Activity, its body and occurredAt equal their create-time values across all later reads; no operation mutates them. |
| P-activity-owned | Every persisted Activity has a non-empty ownerId equal to the logging user. |
| P-task-owned | Every persisted Task has exactly one ownerId; reassign changes it to exactly one in-scope user. |
| P-task-terminal | From Done/Cancelled, no event changes the task. |
| P-task-assignee-visible | Any accepted reassign lands on a user inside the assigner's VisibilityScope; out-of-scope targets are rejected. |
| P-tag-name-unique | For any two Tag creates with the same name, the second fails; never two Tags with one name. |
| P-rbac-crud-verbs | For any (role, verb): ReadOnly is Allowed only for read; Admin/Manager/Rep Allowed for create/read/update/delete (subject to scope). |
| P-rbac-read-visibility | For any read: Admin Allowed for all; others Allowed only for own or same-team records. |
| P-rbac-write-scope | For any update/delete: Admin any; Manager only team members' records; Rep only own; ReadOnly none. |
| P-rbac-reassign-authority | For any reassign: Allowed only for Admin, or Manager acting within the manager's team. |
| P-session-active-user | For any resume, the session resolves to Active only while the user's status is Active; a status flip to Disabled invalidates it. |

## 8. Build plan

Walking skeleton first (prove the topology through one real boundary), then one aggregate lifecycle per
vertical slice, each slice fully green before the next. Definition of done (DoD) is stated per milestone;
the global gates are section 10 (all transitions have a T-row test, all invariants a P-test, all boundaries
a C-test, no cross-boundary violation, >= 80% combined coverage).

**M0 - Walking skeleton (thinnest end-to-end thread).** Implement exactly the path
`crm login -> crm deal create -> crm deal advance`, exercising one real LadybugDB write transaction end to
end through every boundary once. This crosses `crm.commands` (CommandExecution Parsing->Opening->
ResolvingSession->Authorizing->Executing->Rendering->Done), `crm.session` (login: Anonymous->Authenticating
->WritingSession->Active; resume: Resolving->CheckingUser->Active), `crm.authz` (one Allowed decision),
`crm.domain` (Deal create at Lead; advanceStage Lead->persisting->Qualified), and `crm.repo` (Open,
BeginWrite, SaveDeal, Commit against a real temp DB dir). DoD: green for T-CMD-01,03,12,18,20,28,29;
T-SESS-01,06,14,02,18,23; T-DEAL-01,38; contract C-REPO-01,05,06,12, C-SESS-01,05, C-AUTHZ-02, C-ARCH-01;
the login token is written and re-resolved on the next command; the advance is durably persisted (re-Open
sees Qualified); one real write Tx is opened and committed.

**M1 - Deal aggregate slice.** Complete the Deal lifecycle and its persist overlay end to end via
`crm deal create/advance/win/lose/reopen/reassign`. DoD: all 57 T-DEAL rows green; P-deal-owned,
P-deal-amount-nonneg, P-deal-stage-forward, P-deal-terminal, P-deal-won-has-closedate green; C-REPO-10..16,22
green; DBLocked bounded retry and rolledBack-to-priorStage verified; no cross-boundary violation.

**M2 - Task aggregate slice.** `crm task create/start/complete/cancel/reassign`. DoD: all 32 T-TASK rows
green; P-task-owned, P-task-terminal, P-task-assignee-visible green; reassign scope enforced via authz +
`guardCanReassign`.

**M3 - User + Session slice (auth lifecycle).** `crm user disable/enable`, `crm login/logout/whoami`, plus
`register/changePassword/assignRole` create/update paths. DoD: all 19 T-USER and 60 T-SESS rows green;
P-disabled-cannot-auth, P-session-active-user, P-password-hashed, P-username-unique, P-single-team green;
C-SESS-01..10 green; argon2id verified (C-SESS-10).

**M4 - CommandExecution failure envelope + backup/restore.** Harden every section 6 failure row and add
`crm backup` / `crm restore`. DoD: all 33 T-CMD rows green including DBLocked open- and execute-phase retry,
Corrupt fatal exit that instructs `crm restore`, disk-full/timeout rollback; `crm backup` then simulated
corruption then `crm restore` recovers the DB; exit codes per terminal state asserted.

**M5 - Authz/RBAC breadth + pure records + one-default-pipeline.** Complete `crm.authz` for all
(role, verb, entity, scope) combinations and the six pure-record CRUD paths (Account, Contact, Pipeline,
Activity, Tag, Team). DoD: C-AUTHZ-01..14 green; all four P-rbac-* green; DB-uniqueness constraints
(C-REPO-17,18,19) green; P-account-owned, P-contact-owned, P-activity-owned, P-activity-immutable,
P-tag-name-unique, P-team-name-unique green; **P-one-default-pipeline and C-REPO-20 green** (the named
residual); C-ARCH-01 still green across the whole tree.

## 9. Language realization notes

Target language: **Go 1.22+**. How the machines and contracts become code.

**Machines as explicit state + transition switch (no XState runtime, no in-memory actors).** Per
ARCHITECTURE.md section 7, invocations are ephemeral and single-process. Each aggregate (Deal, Task, User)
is a Go struct with an explicit state field typed as a Go enum (`DealStage`, `TaskStatus`, `UserStatus`).
The machine JSON is the spec, not the runtime: implement one `func (a *Aggregate) Fire(evt Event, ctx
Ctx) (Effect, error)` that switches on the current state and then on the event, mirroring the matrix rows.
Guards become boolean methods (`guardCanAdvance` etc.); actions become in-memory context mutations
(`setPendingAdvance` etc.); the `record*Denied`/`recordTerminalRejected` actions return a typed rejection
carrying the violated invariant id. The transient `persisting`/`persistRetry`/`rolledBack` overlay is NOT a
resident state object: it is realized as the load-act-save control flow in the Command Layer (below). The
in-memory state field is advanced to `pendingX` (via `commitStage`/`commitStatus`) only after the repo save
commits; on any save error the struct keeps `priorX`.

**Load-act-save inside one write transaction.** The Command Layer (cobra `RunE`) implements the
CommandExecution envelope as straight-line Go with a bounded retry loop, not a resident interpreter:
open DB (`Repo.Open`, retry on ErrLocked up to 3, backoff 500ms, else Corrupt/DBError); resolve session
(`Sessions.Current`); authorize (`Authorizer.Authorize`, deny -> exit Denied); then `Repo.BeginWrite`,
load the aggregate (`Repo.GetX`), `Fire` the event in memory, `Repo.SaveX`, and `Repo.Commit`; classify any
error and `Repo.Rollback` (idempotent) mapping to ValidationFailed/DBError/DBLocked; render; set the exit
code per terminal state. The whole read-modify-write is inside the single write Tx; there is no second
connection and no goroutine sharing the connection.

**Password hashing.** `crm.session` hashes with argon2id (`golang.org/x/crypto/argon2` `IDKey`) using tuned
memory/time/parallelism parameters, stores only the encoded hash string (`$argon2id$...`), and verifies with
a constant-time comparison. `setCredentials` holds the plaintext only transiently for the verify invoke and
never writes it (enforces `password-hashed`; test C-SESS-10, P-password-hashed).

**go-ladybug isolation.** `github.com/LadybugDB/go-ladybug` is imported **only** by `internal/repo/**`. The
Repo maps Cypher/driver errors to the typed domain errors in section 4.6 (`ErrLocked`, `ErrCorrupt`,
`ErrUnavailable`, `ErrNotFound`, `ErrConstraint`, `ErrConflict`, `ErrDiskFull`, `ErrTimeout`) and applies
`Connection.SetTimeout(10s)` + `Interrupt` for the query timeout. The import boundary is enforced by
C-ARCH-01 (ast-grep or `go list -deps`).

**Named delays as Go durations.** persistRetryBackoff/dbRetryBackoff/verifyRetryBackoff = 500ms;
retriesExhausted at 3; persistTimeout/queryTimeout/loadUserTimeout = 10s; openTimeout/verifyTimeout/
sessionResolveTimeout = 5s; fileIoTimeout = 2s; sessionTTL = 8h (the HMAC-signed expiry is authoritative,
the delay is its declarative representation).

**Session token.** `~/.crm/session` holds `userId + expiresAt`, HMAC-signed with a machine-local key,
written with `0600` permissions. `readSessionFile` verifies the signature and expiry; a bad signature or
parse is `ErrUnreadable`.

**cobra command tree.** `crm login`, `crm logout`, `crm whoami`; `crm user register|disable|enable|assign-role|change-password`;
`crm team create|rename`; `crm account create|update|reassign|delete`; `crm contact create|update|reassign|delete`;
`crm deal create|advance|win|lose|reopen|reassign`; `crm pipeline create|set-default`;
`crm activity log|delete`; `crm task create|start|complete|cancel|reassign`; `crm tag create|apply|remove`;
and the operational pair below. Output defaults to a table; `--json` renders JSON (the `renderOutput` action).

**crm backup / crm restore (required).** Section 6 makes `ErrCorrupt` fatal-until-restore with no
auto-recovery, so recovery depends on these commands. `crm backup <dest>` opens the DB read-consistently
and copies the LadybugDB directory to a timestamped archive at `<dest>`; `crm restore <archive>` refuses to
run against a healthy in-use DB, then replaces `~/.crm/db` from the archive. The Corrupt terminal state's
message must direct the user to `crm restore` (T-CMD-33). Both are covered in M4.

## 10. Hard-TDD protocol (read this before writing any code)

1. **Test-writer agent** reads sections 6 and 7 only and writes the full test suite from the spec: one test
   per T-row (7.1), one per C-row (7.2), one property test per P-row (7.3). It does not invent behavior; if
   a needed fact is absent from sections 6-7, that is a spec gap to fix in this document, not a guess.
2. **The tests are then LOCKED.** The implementer agent may not modify, weaken, skip, or delete them to make
   them pass. Locking is structural (the test files are owned by the test-writer; changes require a design
   round-trip).
3. **The implementer agent** writes production code until the locked tests pass, honoring the section 4.5
   Architecture Contract (C-ARCH-01) and the section 9 realization rules.
4. **Completeness gates.** Every transition in section 5/7.1 has a T-test; every invariant in section 3.4 is
   property-tested (7.3); every boundary in section 4.6 has contract tests (7.2). Coverage target and gates
   per project conventions (>= 80% combined; integration tests use a real LadybugDB dir and no mocks; unit
   tests may mock a single collaborator). Use `@xstate/graph` covering paths (7 intro) to prove no edge was
   dropped.
5. **A wrong test is a design defect, not a test to adjust.** If a locked test is wrong, stop: fix the design
   (the machine JSON / matrix / this BUILD.md), then regenerate the affected tests from the corrected spec.
   Never "adjust" a test to pass, and never edit a test to match code that drifted from the spec. The
   round-trip is design -> BUILD.md -> tests -> code, always in that direction.

## 11. Open questions and residual risks

Named risks are cheaper than surprises. Each is either accepted-by-design or covered by a specific test.

1. **`one-default-pipeline` has no guard and no structural guarantee (named residual).** Enforced only by
   the `setDefault` atomic read-modify-write inside the one write Tx. Risks: a direct repo write bypassing
   `setDefault`; a seed step that creates zero defaults. Mitigations: `setDefault` is the only write path to
   `isDefault`; seeding must create exactly one default pipeline; covered by P-one-default-pipeline
   (post-condition `count(isDefault==true)==1`) and C-REPO-20. Pipeline has no delete action, so the sole
   default cannot be removed to reach zero. Accepted with test coverage.
2. **Structural ownership/immutability invariants rely on repo write discipline, not a DB cardinality
   constraint.** `deal-owned`, `account-owned`, `contact-owned`, `task-owned`, `activity-owned`,
   `single-team`, and `activity-immutable` are "structural" because the write paths set the owner at create
   and expose no mutation, not because LadybugDB enforces the cardinality. A buggy repo write could violate
   them. Mitigation: property tests P-*-owned, P-single-team, P-activity-immutable, and C-REPO-21,23 pin the
   discipline. Residual: a new repo method that mutates an owner/immutable field would need a new test;
   flagged for review gate.
3. **Corruption is fatal-until-restore.** If the user never ran `crm backup`, an `ErrCorrupt` open is
   unrecoverable data loss. Mitigation: `crm backup`/`crm restore` (section 9, M4) and the Corrupt-state
   message directing the user to restore. Residual: backup cadence is a user responsibility, not enforced by
   the binary. Consider a future "backup reminder" or auto-snapshot-before-migration.
4. **`ErrConflict` retried as if locked.** CommandExecution routes `isErrConflict` to DBLocked and retries
   the whole Tx (T-CMD-23). In a single-writer store conflicts are rare; if go-ladybug ever surfaces a
   non-retryable conflict, three retries then a DBError is the bound. Accepted.
5. **Stale session token after a failed logout.** `clearSessionFile` is best-effort (T-SESS-35,36); a
   leftover token is mitigated by the HMAC signature, the signed expiry, and resume-time re-validation
   (`session-active-user`, T-SESS-23,24). Residual: token file readable by another local user if perms are
   wrong; realization mandates `0600`.
6. **Session TTL vs clock skew.** `sessionTTL` (8h) is a declarative delay; the signed `expiresAt` is
   authoritative and compared to local time. Large clock changes could expire early or late. Minor,
   accepted.
7. **VisibilityScope drift between authz and repo.** `task-assignee-visible` and the `rbac-read-visibility`
   scope depend on authz computing team membership consistently with what the repo stores. If the two drift,
   a reassign or read could leak. Mitigation: authz is pure and fed owner/team ids resolved by the repo in
   the same Tx; C-AUTHZ-05..11 and P-task-assignee-visible cover it. Residual: integration tests must feed
   authz from real repo reads, not fixtures, for the scope cases.
8. **Concurrency is refuse-after-retry, by design.** Two concurrent `crm` writers: the second retries 3x
   (~1.5s) then is refused (`ErrLocked` -> DBError "database busy"). A burst of concurrent invocations will
   show user-visible failures. This is the accepted embedded-store posture, not a bug (section 4.4).
9. **Spec-to-code drift on the machines.** The machine JSON is the oracle; the Go transition switch is the
   implementation. Drift is prevented by hard TDD (locked T-tests) plus the `@xstate/graph` covering-path
   check that every edge has a T-row. Residual: if the JSON changes, tests regenerate before code (section
   10 rule 5).
10. **Deferred / not modeled.** Deal `reassign` and the Account/Contact/Tag/Activity CRUD verbs are covered
    by authz + repo contract tests, not by a lifecycle machine (they are pure records, section 5.6). Deal
    `update` of non-stage fields (title/amount) is a plain write guarded by `rbac-write-scope`; if amount is
    edited it must re-check `deal-amount-nonneg` (P-deal-amount-nonneg applies). Flagged so it is not assumed
    to be machine-driven.
