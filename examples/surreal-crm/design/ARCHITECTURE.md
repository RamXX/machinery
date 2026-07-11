# Architecture: Surreal CRM

The narrative twin of `workspace.dsl`. It explains *why*; the DSL and the Architecture Contract
below define *what*, machine-checkably. Data shapes are not restated here: the single source of truth
for the domain is `domain.modelith.yaml`.

This is a rebuild of the running go-crm system that keeps the domain intact and replaces the
persistence foundation: the embedded LadybugDB graph directory becomes a SurrealDB instance in a
local Docker container. The domain model, the invariants, and all five state machines carry over
unchanged; what changes is the class and bound of every store failure, which is exactly what this
document must reclassify.

## 1. Shape and deployment

One statically linked Go binary, `crm`, invoked per command, plus one long-running SurrealDB
container managed by the local Docker Engine (`docker compose up -d surrealdb`, bound to
`127.0.0.1:8000`, data on a named volume). A single invocation is the unit of work: it connects to
SurrealDB, resolves the caller's session, authorizes the action, runs it inside one transaction,
renders output, and exits.

The store is now client-server. The single-writer file lock is gone: concurrent `crm` invocations
hold separate connections and serialize inside SurrealDB's transaction engine, so the "database
locked" failure class of the embedded store becomes "transaction conflict or store unavailable",
transient and retryable. The new failure the embedded store never had is availability: if the
container is stopped, starting, or the Docker daemon is down, every command fails at connect. The
CommandExecution machine already models this envelope (DBLocked with bounded retry, DBError,
Corrupt); the mitigation table below reclassifies what each residual transition means here.

## 2. Technology

| concern | choice | why |
|---|---|---|
| language | Go 1.22+ | target language; single static binary, good CLI ergonomics |
| CLI | cobra | subcommands, flags, help; matches `crm <noun> <verb>` |
| password hash | argon2id (`golang.org/x/crypto/argon2`) | enforces `password-hashed`; memory-hard |
| store | SurrealDB 2.x via `github.com/surrealdb/surrealdb.go` | multi-model store; one table per aggregate, record links for relationships |
| store deployment | Docker container, `restart: unless-stopped`, named volume, `127.0.0.1` bind | reproducible local instance; survives daemon restarts; no remote exposure |
| query | SurrealQL (parameterized statements over the driver) | SurrealDB's query language; transactions via BEGIN/COMMIT |

## Transition architecture

This is a store swap, not a domain rewrite, and the transition contract is scoped accordingly:
`legacy/domain.modelith.yaml` and `domain.modelith.yaml` carry the same entities, every disposition
in `migration.yaml` is `reuse`, and there are no field or lifecycle mappings because nothing about
the records' meaning changes. What must move safely is the data itself and the query layer above it.

The legacy surface ledger (`legacy/surface.yaml`) anchors the swap: every `crm` command and every
node label of the running system is mapped to this design, and the two store-operational commands
are the only surface that does not carry over (`crm backup` is dropped for the SurrealDB export
procedure; `crm restore` is deferred to the operations iteration, tracked in BUILD.md section 12).

Three transition components exist only during the migration: a read-only LadybugDB exporter (walks
the graph, emits a signed manifest of records and links in stable-id order), a SurrealQL importer
(idempotent by record id, safe to replay), and a shadow-read comparator (runs every read against
both stores, compares normalized rows and authorization decisions, always serves the legacy
result). During baseline and shadow the embedded store remains authoritative; cutover freezes
legacy writes, drains the final export, reconciles to zero drift, and repoints the binary's
connection string. Rollback within the 72h window is a configuration change back to the embedded
store plus a replay of target-side writes from the signed manifests.

## 3. Components (inside the `crm` binary)

- **Command Layer** owns process lifecycle: it connects to the store, begins and commits or rolls
  back the single transaction, and renders results. It is the operational envelope.
- **Session and Auth** performs login (verify argon2id hash), writes and reads the session token,
  and resolves the current `User`. Enforces `disabled-cannot-auth` and `session-active-user`.
- **Authorization (RBAC)** is a pure decision function over `(user, verb, entityType, ownerId, teamId)`.
  It is the single home of `rbac-crud-verbs`, `rbac-read-visibility`, `rbac-write-scope`, and
  `rbac-reassign-authority`. Pure logic, no I/O: it gets a contract spec, not a state machine.
- **Domain Services** hold the aggregates whose lifecycles are state machines (`Deal`, `Task`, `User`)
  and call Authorization before every mutation and Repository to read and persist.
- **Repository** is the only component that imports the SurrealDB driver. It translates domain
  reads and writes to SurrealQL, executes them in the caller's transaction, and maps driver and
  store errors to typed domain errors.

## 4. Architecture Contract

```yaml
contract_version: 2
boundaries:
  - id: crm.commands
    kind: component
    element: commands
    code: [ "internal/cli/**" ]
    exposes: [ "internal/cli/root.go" ]
  - id: crm.session
    kind: component
    element: session
    code: [ "internal/session/**" ]
    exposes: [ "internal/session/session.go" ]
  - id: crm.authz
    kind: component
    element: authz
    code: [ "internal/authz/**" ]
    exposes: [ "internal/authz/authz.go" ]
  - id: crm.domain
    kind: component
    element: domain
    code: [ "internal/domain/**" ]
    exposes: [ "internal/domain/service.go" ]
  - id: crm.repo
    kind: component
    element: repo
    code: [ "internal/repo/**" ]
    exposes: [ "internal/repo/repo.go" ]
  - id: crm.model
    kind: component
    element: model
    code: [ "internal/model/**" ]   # shared domain types; no exposes list, all of it is API
externals:
  - id: external.surrealdb
    element: store
    imports: [ "github.com/surrealdb/surrealdb.go" ]
ignore:
  - "internal/testsupport/**"   # hard-TDD fakes shared by the test suites
  - "internal/arch/**"          # the architecture-boundary test package itself
  - "internal/migrate/**"       # transition-only exporter/importer/comparator; removed after cutover
dependency_rules:
  allow:
    - crm.commands -> crm.session
    - crm.commands -> crm.domain
    - crm.commands -> crm.repo      # open the connection, own the transaction boundary
    - crm.commands -> crm.model
    - crm.session  -> crm.repo
    - crm.session  -> crm.model
    - crm.authz    -> crm.model
    - crm.domain   -> crm.authz
    - crm.domain   -> crm.repo
    - crm.domain   -> crm.model
    - crm.repo     -> crm.model
    - crm.repo     -> external.surrealdb   # the repository is the sole importer of the store driver
  deny:
    - crm.commands -> crm.authz     # authorization is decided inside domain services
    - "crm.* -> external.surrealdb"   # only crm.repo may import the SurrealDB driver
  notes:
    - "All store access goes through crm.repo. Only crm.repo imports surrealdb.go."
    - "Authorization is enforced in crm.domain, never in the command layer, so no command path can skip it."
    - "crm.model is the shared vocabulary (types, enums, typed errors); every component may import it, it imports nothing."
```

## 5. Interface contracts (boundary shapes for the hard-TDD contract tests)

Signatures are Go-flavored pseudocode; the types reference `domain.modelith.yaml`. The Repo
interface is unchanged from the legacy system except at the edges the store swap touches: `Open`
becomes `Connect`, and the error vocabulary trades the file-lock class for the availability and
conflict classes.

The typed error taxonomy is retained verbatim from the legacy design, because the five machines
(and their `isErr*` guards) carry over unchanged; what changes is the mapping from driver and
store conditions to those classes. `ErrLocked` remains the transient-retryable class: it no longer
means a file lock, it means the container is starting, restarting, or the store briefly refused
the connection mid-command.

```
// crm.repo  (the only importer of surrealdb.go; all methods run inside an open Tx)
type Repo interface {
  Connect(url, ns, db string) (Tx, error)  // ErrLocked (container starting/restarting),
                                           // ErrCorrupt (bad volume), ErrUnavailable (daemon down,
                                           // wrong address or credentials)
  BeginWrite(Tx) error                      // SurrealQL BEGIN TRANSACTION
  Commit(Tx) error; Rollback(Tx) error
  GetUserByName(Tx, name string) (User, error)     // ErrNotFound
  GetDeal(Tx, id string) (Deal, error)             // ErrNotFound
  SaveDeal(Tx, Deal) error                         // ErrConstraint, ErrConflict, ErrDiskFull, ErrTimeout, ErrLocked
  // ... GetTask/SaveTask, GetAccount/SaveAccount, etc.
}
// Typed errors (mapped from the SurrealDB driver and store; same eight classes as the legacy
// design): ErrLocked, ErrCorrupt, ErrUnavailable, ErrNotFound, ErrConstraint, ErrConflict,
// ErrDiskFull, ErrTimeout.

// crm.authz  (pure; no I/O)
type Authorizer interface {
  Authorize(actor User, verb Verb, entity EntityType, ownerID, teamID string) Decision
}
type Decision struct { Allowed bool; Reason string }   // Reason set when denied

// crm.session
type Sessions interface {
  Login(name, password string) (Session, error)   // ErrBadCredentials, ErrDisabled, ErrLocked
  Current() (User, error)                          // ErrNoSession, ErrExpired
  Logout() error
}
```

Idempotency: reads are safe to retry; writes run in one transaction and are retried only on
`ErrLocked` (bounded, by the machines' retry overlay) and `ErrConflict` (the invocation envelope
retries the whole transaction); the transaction never partially committed. `Login` is not retried
on `ErrBadCredentials`.

## 6. Dependency mitigation posture (drives the Phase 3 failure transitions)

The machines are unchanged from the legacy design; this table is where the store swap actually
lands. A mitigation reclassifies a failure, it does not delete it: the embedded store's
"file locked by another process" (serialized, local) becomes "store unavailable or transaction
conflict" (transient, bounded by retry and the container restart policy), and the same DBLocked
retry overlay in CommandExecution absorbs it with new detection and new bounds.

| dependency | failure modes | mitigation (deployment) | residual behavior the FSM must handle | bound |
|---|---|---|---|---|
| `store` (SurrealDB connect) | container starting, restarting, or briefly unreachable | Docker `restart: unless-stopped`; healthcheck gates readiness | `ErrLocked` on connect: retry with backoff (the DBLocked overlay), then exit with a clear message naming the container | retry <= 3, ~1.5s total |
| `store` (SurrealDB connect) | daemon not running, wrong address, bad credentials | documented `docker compose up -d`; config file validated on load | `ErrUnavailable`: fail loudly; the rendered error names the daemon check first | fatal for this invocation |
| `store` (SurrealDB commit) | transaction conflict under concurrent invocations | store-side transaction engine; one transaction per invocation | `ErrConflict`: roll back; the envelope retries the whole transaction (DBLocked, phase=execute) | retry <= 3, atomic |
| `store` (SurrealDB write) | constraint violation | one transaction per invocation; schema asserts uniqueness | `ErrConstraint`: roll back, surface as a domain validation error | atomic, no partial write |
| `store` (SurrealDB write) | data volume full | named volume on the local disk; no quota | `ErrDiskFull`: roll back, fail loudly; store stays consistent | atomic |
| `store` (SurrealDB query) | runaway query | per-query timeout on the driver call | `ErrTimeout`: abort, surface, roll back | timeout 10s |
| `store` (data integrity) | corrupt or image-incompatible volume | pinned image tag; volume snapshot before upgrades; SurrealDB export as backup | `ErrCorrupt`: fail loudly, direct the user to the restore runbook | fatal, no auto-recovery |
| `dockerd` | daemon stopped or container removed | restart policy covers container crashes, not a stopped daemon; the runbook covers the rest | connect fails as `ErrUnavailable` with the daemon check named first | user starts the daemon |
| `sessionfile` | missing / expired / unreadable | none | `ErrNoSession` / `ErrExpired`: require `crm login` | user re-authenticates |

## 7. Persistence and placement (the C4 to FSM bridge)

CLI invocations are short-lived and single-process, so there are no in-memory actors. Every stateful
aggregate is loaded, acted on, and saved inside the one transaction the Command Layer owns.

| component | machine placement | persistence | concurrency serialization |
|---|---|---|---|
| `Deal` aggregate | ephemeral in-process; load-act-save in the Tx | `deal` table `stage` field | read-modify-write in one transaction; cross-process by the store's transaction engine |
| `Task` aggregate | ephemeral in-process; load-act-save in the Tx | `task` table `status` field | as above |
| `User` aggregate | ephemeral in-process; load-act-save in the Tx | `user` table `status` field | as above |
| `Session` | in-process during a command; token on disk | `~/.crm/session` (user id + expiry, HMAC-signed) | last write wins; single local user |
| `CommandExecution` | ephemeral per invocation (the operational envelope) | none | one invocation, one transaction |

## 8. NFR record

- **Security posture**: a local single-user CLI, but no longer without a network surface: the
  store listens on `127.0.0.1:8000` inside Docker and must never bind a routable interface. The
  binary authenticates to SurrealDB with credentials read from a local config file (0600); user
  authentication and authorization are unchanged (argon2id-hashed passwords, HMAC-signed session
  token, the four `rbac-*` invariants decided by the pure Authorization component and re-checked
  in the domain guards). Passwords and secrets are never logged (`password-hashed`). Out of scope
  by design: multi-tenant isolation, remote store deployments, TLS to the local container, and
  encryption at rest beyond volume permissions.
- **Capacity assumptions**: one team's CRM in a single SurrealDB namespace; thousands of records,
  not millions. One process, one connection, one transaction per invocation; concurrent
  invocations serialize in the store's transaction engine with one bounded conflict retry.
  Throughput beyond a handful of interactive users per machine is explicitly out of scope.
- **Observability**: the operator is the user. The signals are the process exit code (one per
  CommandExecution terminal state), rendered output on stdout, and a classified error on stderr
  (Denied, ValidationFailed, DBError, Corrupt), carrying the violated invariant id on a rejected
  transition. `ErrUnavailable` renders the container and daemon check first, because "store down"
  is the failure a fresh machine actually hits. During the transition, the shadow comparator
  logs every parity mismatch with the record id and the differing fields. No metrics backend, no
  structured log file, no tracing: out of scope for a local CLI.

## 9. Gate 2 result

- Every Modelith action maps to an owning component: lifecycle actions to Domain Services, `login`/
  `logout`/`changePassword` to Session, RBAC-gated verbs to Authorization plus Domain Services,
  reads/writes to Repository. PASS.
- Every external dependency has a mitigation posture (section 6, including the Docker Engine the
  embedded design never needed). PASS.
- Boundary and interface contracts defined (sections 4 and 5). PASS.
- Persistence and placement decided for each stateful component (section 7). PASS.

Diagram export is optional: `structurizr-cli export -workspace workspace.dsl -format mermaid -output diagrams/`
(needs Java 17+); otherwise the DSL remains the source of truth.
