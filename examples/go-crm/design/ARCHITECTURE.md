# Architecture: Go CRM

The narrative twin of `workspace.dsl`. It explains *why*; the DSL and the Architecture Contract
below define *what*, machine-checkably. Data shapes are not restated here: the single source of truth
for the domain is `domain.modelith.yaml`.

## 1. Shape and deployment

One statically linked Go binary, `crm`, invoked per command. All state lives in a local LadybugDB
graph directory (default `~/.crm/db`) plus a small session token file (`~/.crm/session`). There is no
server and no network dependency. A single invocation is the unit of work: it opens the database,
resolves the caller's session, authorizes the action, runs it inside one write transaction, renders
output, and exits.

Concurrency is bounded by the store. LadybugDB is embedded and single-writer, and the Go binding's
`Connection` is not documented as goroutine-safe, so within a process we use one connection and one
write transaction at a time, and across processes the on-disk database is held by whichever invocation
opened it for writing. Two concurrent `crm` writes therefore serialize or the second is refused; the
design treats that refusal as a first-class, recoverable failure rather than a crash.

## 2. Technology

| concern | choice | why |
|---|---|---|
| language | Go 1.22+ | target language; single static binary, good CLI ergonomics |
| CLI | cobra | subcommands, flags, help; matches `crm <noun> <verb>` |
| password hash | argon2id (`golang.org/x/crypto/argon2`) | enforces `password-hashed`; memory-hard |
| store | LadybugDB via `github.com/LadybugDB/go-ladybug` | embedded property graph; CRM is relationship-shaped |
| query | Cypher (through `Connection.Query` / `Prepare` + `Execute`) | LadybugDB's query language; parameterized statements |

## 3. Components (inside the `crm` binary)

- **Command Layer** owns process lifecycle: it opens the database, begins and commits or rolls back the
  single write transaction, and renders results. It is the operational envelope.
- **Session and Auth** performs login (verify argon2id hash), writes and reads the session token, and
  resolves the current `User`. Enforces `disabled-cannot-auth` and `session-active-user`.
- **Authorization (RBAC)** is a pure decision function over `(user, verb, entityType, ownerId, teamId)`.
  It is the single home of `rbac-crud-verbs`, `rbac-read-visibility`, `rbac-write-scope`, and
  `rbac-reassign-authority`. Pure logic, no I/O: it gets a contract spec, not a state machine.
- **Domain Services** hold the aggregates whose lifecycles are state machines (`Deal`, `Task`, `User`)
  and call Authorization before every mutation and Repository to read and persist.
- **Repository** is the only component that imports `go-ladybug`. It translates domain reads and writes
  to Cypher, executes them in the caller's transaction, and maps LadybugDB errors to typed domain errors.

## 4. Architecture Contract

```yaml
contract_version: 1
boundaries:
  - id: crm.commands
    kind: component
    code: [ "internal/cli/**" ]
    exposes: [ "internal/cli/root.go" ]
  - id: crm.session
    kind: component
    code: [ "internal/session/**" ]
    exposes: [ "internal/session/session.go" ]
  - id: crm.authz
    kind: component
    code: [ "internal/authz/**" ]
    exposes: [ "internal/authz/authz.go" ]
  - id: crm.domain
    kind: component
    code: [ "internal/domain/**" ]
    exposes: [ "internal/domain/service.go" ]
  - id: crm.repo
    kind: component
    code: [ "internal/repo/**" ]
    exposes: [ "internal/repo/repo.go" ]
dependency_rules:
  allow:
    - crm.commands -> crm.session
    - crm.commands -> crm.domain
    - crm.commands -> crm.repo      # open db, own the transaction boundary
    - crm.session  -> crm.repo
    - crm.domain   -> crm.authz
    - crm.domain   -> crm.repo
  deny:
    - crm.commands -> crm.authz     # authorization is decided inside domain services
    - "crm.* -> external.ladybug"   # only crm.repo may import go-ladybug
  notes:
    - "All graph access goes through crm.repo. Only crm.repo imports go-ladybug."
    - "Authorization is enforced in crm.domain, never in the command layer, so no command path can skip it."
```

## 5. Interface contracts (boundary shapes for the hard-TDD contract tests)

Signatures are Go-flavored pseudocode; the types reference `domain.modelith.yaml`.

```
// crm.repo  (the only importer of go-ladybug; all methods run inside an open write Tx)
type Repo interface {
  Open(path string) (Tx, error)          // errors: ErrLocked, ErrCorrupt, ErrUnavailable
  BeginWrite(Tx) error                    // Cypher BEGIN TRANSACTION
  Commit(Tx) error; Rollback(Tx) error
  GetUserByName(Tx, name string) (User, error)     // ErrNotFound
  GetDeal(Tx, id string) (Deal, error)             // ErrNotFound
  SaveDeal(Tx, Deal) error                         // ErrConstraint, ErrConflict, ErrDiskFull
  // ... GetTask/SaveTask, GetAccount/SaveAccount, etc.
}
// Typed errors (map from go-ladybug): ErrLocked, ErrCorrupt, ErrUnavailable, ErrNotFound,
// ErrConstraint, ErrConflict, ErrDiskFull, ErrTimeout.

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

Idempotency: reads are safe to retry; writes run in one transaction and are retried only on `ErrLocked`
(the transaction never partially committed). `Login` is not retried on `ErrBadCredentials`.

## 6. Dependency mitigation posture (drives the Phase 3 failure transitions)

Embedded, so there is no operator and no HA. The contrast with a networked store matters: some failures
that would be transient-and-bounded behind a Kubernetes operator are here either serialized (single
writer) or fatal-until-restore (corruption). The state machines must handle them accordingly.

| dependency | failure modes | mitigation (deployment) | residual behavior the FSM must handle | bound |
|---|---|---|---|---|
| LadybugDB open | file locked by another `crm` process | none (single-file embedded) | `ErrLocked` on open: retry with backoff, then exit with a clear message | retry <= 3, ~1.5s total |
| LadybugDB open | corrupt or version-incompatible directory | none (no HA); backup via `crm backup` / restore via `crm restore` | `ErrCorrupt`: fail loudly, tell the user to restore from backup | fatal, no auto-recovery |
| LadybugDB write | Cypher/constraint violation | one write transaction per invocation | `ErrConstraint`: roll back, surface as a domain validation error | atomic, no partial write |
| LadybugDB write | disk full | none | `ErrDiskFull`: roll back, fail loudly; DB stays consistent | atomic |
| LadybugDB query | runaway query | `Connection.SetTimeout` + `Interrupt` | `ErrTimeout`: abort, surface, roll back | timeout 10s |
| Session file | missing / expired / unreadable | none | `ErrNoSession` / `ErrExpired`: require `crm login` | user re-authenticates |

## 7. Persistence and placement (the C4 to FSM bridge)

CLI invocations are short-lived and single-process, so there are no in-memory actors. Every stateful
aggregate is loaded, acted on, and saved inside the one write transaction the Command Layer owns.

| component | machine placement | persistence | concurrency serialization |
|---|---|---|---|
| Deal aggregate | ephemeral in-process; load-act-save in the Tx | graph node `stage` attribute | read-modify-write in one write Tx; cross-process by the store's single-writer lock |
| Task aggregate | ephemeral in-process; load-act-save in the Tx | graph node `status` attribute | as above |
| User aggregate | ephemeral in-process; load-act-save in the Tx | graph node `status` attribute | as above |
| Session | in-process during a command; token on disk | `~/.crm/session` (user id + expiry, HMAC-signed) | last write wins; single local user |
| Command execution | ephemeral per invocation (the operational envelope) | none | one invocation owns the write Tx |

## 8. Gate 2 result

- Every Modelith action maps to an owning component: lifecycle actions to Domain Services, `login`/
  `logout`/`changePassword` to Session, RBAC-gated verbs to Authorization plus Domain Services,
  reads/writes to Repository. PASS.
- Every external dependency has a mitigation posture (section 6). PASS.
- Boundary and interface contracts defined (sections 4 and 5). PASS.
- Persistence and placement decided for each stateful component (section 7). PASS.

Diagram export is optional: `structurizr-cli export -workspace workspace.dsl -format mermaid -output diagrams/`
(needs Java 17+); otherwise the DSL remains the source of truth.
