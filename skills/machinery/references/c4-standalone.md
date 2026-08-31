# C4 architecture (standalone)

Phase 2 fixes how the system is built and deployed, and what each dependency does when it fails.
This is the standalone C4 technique: Structurizr DSL plus a machine-checkable Architecture Contract,
with no dependency on any project settings or external tracker. Two decisions here feed Phase 3
directly: the **dependency mitigation posture** and the **persistence and placement** of each stateful
component. `machinery check <design> --gate g2` verifies the contract deterministically; run it
before calling Gate 2.

## workspace.dsl (Structurizr) - authoring guide

The DSL is text - the gates parse it for element identifiers and tags, never rendering it. But it
must also be valid Structurizr DSL so `structurizr-cli export` can produce diagrams. **Follow these
rules exactly; the Structurizr CLI parser is strict and rejects shorthand that older versions
accepted.**

### Syntax rules (the parser is strict)

1. **One property per line inside `{ }` blocks.** Never use semicolons to separate properties.
   ```dsl
   # WRONG - newer Structurizr CLI rejects this:
   element "Person" { shape Person; background #08427B; color #ffffff }
   systemContext sys "Context" { include *; autoLayout lr }

   # CORRECT:
   element "Person" {
     shape Person
     background #438DD5
     color #ffffff
   }
   systemContext sys "Context" {
     include *
     autoLayout lr
   }
   ```

2. **Never declare a `deployment` view without deployment nodes.** A deployment view references a
   named environment (e.g. `"production"`). If the model defines no `deploymentNode` for that
   environment, the parser throws "The environment does not exist." Omit the deployment view entirely
   unless you have actual deployment topology to show. Most single-binary or design-only examples
   need only `systemContext`, `container`, and `component` views.

3. **Element declarations**: `person`, `softwareSystem`, `container`, `component`. Each takes
   `identifier "Name" "Description"` and optionally a technology string and tags:
   ```dsl
   store = container "Graph Store" "Embedded property graph." "LadybugDB" "Database"
   ```
   The last quoted string `"Database"` is a **tag**. G2 derives required mitigation coverage from
   the tags `Database`, `Queue`, and `External`.

4. **Relationships**: `source -> dest "Description" "Technology"`.

5. **Identifiers**: lowercase, no spaces. Use the singular canonical names from the domain model.

### Complete valid template

```dsl
workspace "Project" "One-line description." {

  model {
    user = person "User" "Who uses it."

    sys = softwareSystem "System" "What it does." {
      api = container "API" "Business logic." "Elixir/Phoenix"
      db  = container "Database" "State of record." "PostgreSQL" "Database"
      q   = container "Queue" "Async work." "RabbitMQ" "Queue"
    }
    pay = softwareSystem "Payments" "Third-party charges." "External"

    user -> api "Uses" "HTTPS"
    api  -> db  "Reads/writes" "SQL"
    api  -> q   "Publishes" "AMQP"
    api  -> pay "Charges" "REST"
  }

  views {
    systemContext sys "Context" {
      include *
      autoLayout lr
    }

    container sys "Containers" {
      include *
      autoLayout lr
    }

    styles {
      element "Person" {
        shape Person
        background #438DD5
        color #ffffff
      }
      element "Software System" {
        background #2E6295
        color #ffffff
      }
      element "Container" {
        background #438DD5
        color #ffffff
      }
      element "Component" {
        background #6FA8DC
        color #ffffff
      }
      element "Database" {
        shape Cylinder
      }
      element "Queue" {
        shape Pipe
      }
      element "External" {
        background #8E8E93
        color #ffffff
      }
    }
  }
}
```

### Dark-mode-friendly color palette

The standard C4 colors (`#08427B`, `#1168BD`, `#85BBF0` with black text) are designed for white
backgrounds and wash out or become unreadable on dark backgrounds (GitHub dark mode, VS Code dark
themes). The palette above uses brighter, medium-tone blues with **white text throughout**, which
reads cleanly on both light and dark backgrounds:

| Element | Background | Why |
|---------|-----------|-----|
| Person | `#438DD5` | Medium-bright blue, visible on dark; shape Person distinguishes it |
| Software System | `#2E6295` | Slightly darker, recedes behind containers |
| Container | `#438DD5` | Same as Person - the C4 convention; shape distinguishes |
| Component | `#6FA8DC` | Lighter blue, white text (not black - black fails on dark) |
| External | `#8E8E93` | Neutral gray; clearly "not ours" |
| Database | inherits + `shape Cylinder` | |
| Queue | inherits + `shape Pipe` | |

### Validate and export

After authoring, always validate the DSL compiles and exports before committing.
`machinery verify-c4 <design>` runs exactly this as the C4 engine phase (exit 0 iff the export
succeeds; `MACHINERY_STRUCTURIZR_CLI` overrides the binary lookup); the raw commands:

```bash
# Validate + export to Mermaid (renders inline in GitHub README/PRs):
structurizr-cli export -workspace design/workspace.dsl -format mermaid -output design/diagrams/

# Export to SVG (for embedding in docs):
structurizr-cli export -workspace design/workspace.dsl -format svg -output design/diagrams/

# Interactive in-browser (best for exploration):
docker run -it --rm -p 8080:8080 -v $(pwd)/design:/usr/local/structurizr structurizr/lite
```

If `structurizr-cli` is unavailable, install it:
```bash
brew install structurizr-cli   # macOS
# Or: download from https://github.com/structurizr/cli/releases, add to PATH
```

Requires Java 17+.

## ARCHITECTURE.md must carry the Architecture Contract (v2)

Embed a parseable YAML block under a heading containing "Architecture Contract", as a yaml code
fence starting with `contract_version`. It is the machine-checkable twin of the narrative. The
shape, from the go-crm example:

```yaml
contract_version: 2
boundaries:
  - id: crm.domain
    kind: component
    element: domain                       # workspace.dsl identifier this boundary binds to;
                                          # defaults to the last segment of the id
    code: [ "internal/domain/**" ]        # required: file globs mapping code to the boundary
    exposes: [ "internal/domain/service.go" ]  # optional public interface
  - id: crm.repo
    kind: component
    element: repo
    code: [ "internal/repo/**" ]
    exposes: [ "internal/repo/repo.go" ]
  - id: crm.model
    kind: component
    element: model
    code: [ "internal/model/**" ]         # no exposes list: all of it is API
externals:
  - id: external.ladybug
    element: store                        # optional: the dsl element it corresponds to
    imports: [ "github.com/LadybugDB/go-ladybug" ]   # import-path prefixes
    # modules: [ "Ladybug" ]              # module-name prefixes (Elixir)
ignore:
  - "internal/testsupport/**"             # source exempt from boundary mapping (test scaffolding)
dependency_rules:
  allow:
    - crm.domain -> crm.repo
    - crm.domain -> crm.model
    - crm.repo   -> crm.model
    - crm.repo   -> external.ladybug      # the sole importer of the embedded store
  deny:
    - "crm.* -> external.ladybug"         # an explicit allow overrides a matching deny
  notes:
    - "All graph access goes through crm.repo."
```

Field semantics:

- **boundary**: `id` (unique), `kind`, `element` (the `workspace.dsl` identifier it binds to;
  defaults to the last segment of the id, so set it explicitly when they differ), `code` (globs,
  required; G4 cannot map the boundary without them), `exposes` (optional: a file entry exposes
  exactly its package directory, a glob entry matches imports), `modules` (Elixir: module-name
  prefixes belonging to the boundary), `provides` and `consumes` (optional: kebab-case capability
  keys; see composability below).
- **externals**: `{id, element (optional dsl element), imports: [import-path prefixes],
  modules: [module-name prefixes, for Elixir]}`. Any `dependency_rules` reference to `external.*`
  must be declared here.
- **ignore**: globs for source paths exempt from boundary mapping (test scaffolding, generated
  code). Staged brownfield adoption leans on this too: start with broad ignore globs over the
  unmodeled remainder and ratchet them down as slices come under the gates (see
  `docs/brownfield-team-guide.md` in the machinery repo).
- **dependency_rules**: `allow`, `deny`, and `baseline` edges, `src -> dest`. `*` globs are legal
  in `allow` and `deny` only: `baseline:` is an enumerated-edges ratchet, and a wildcard baseline
  rule (which would amnesty the whole edge space) is a G2 ERROR; run `machinery baseline` to
  enumerate today's edges instead.
  The allow graph must be acyclic: a dependency cycle among boundaries is a G2 ERROR (one finding
  per cycle, naming the members and a representative path); a cycle that closes only once
  `baseline:` edges are unioned in is tolerated debt, reported as a warning until the ratchet
  burns it down.
- **provides / consumes (composability over capability keys)**: a boundary may declare the
  capability keys it provides and consumes. Two laws are then checked at design time: provisions
  are DISJOINT (a key with two providers is a G2 ERROR: which one a consumer binds to would be an
  accident of wiring), and consumption is SATISFIED (a consumed key must have a provider, and the
  consumer must hold a direct `allow` edge to that provider; a capability the dependency graph
  cannot reach is a contradiction between the two views of the same architecture). A boundary
  providing and consuming the same key is an error. Runtime lifecycle laws (activation ordering,
  a provider withdrawing a key only after its dependents deactivate) are deliberately NOT modeled
  by this static vocabulary and are not covered by these checks. Contracts without
  `provides`/`consumes` check exactly as before.
- **dependency_rules.assert**: negative reachability claims, proven over the transitive closure of
  the allow graph on every check. `assert: [{no_path: src -> dst}]` fails with the witness path
  the moment any chain of allow edges reaches `dst` from `src`; a path that closes only through
  `baseline:` edges warns as ratchet debt. Concrete boundary ids only (no globs). Whenever the
  contract's prose claims an independence ("domains never import each other"), write the
  assertion; a note is a claim nothing traverses.
  Precedence: an explicit (literal) allow overrides a matching deny GLOB, which is how "deny the
  pattern, allow the one sanctioned edge" is written; but a literal allow and a literal deny of
  the same edge is a G2 error, not an override. Deny rules cannot reference boundaries that do
  not exist yet in `workspace.dsl`; keep planned-but-unbuilt boundaries in comments until they
  have DSL elements.
- **baseline** (brownfield): a baseline edge is a TOLERATED VIOLATION, distinct from an intended
  allow. It is generated by `machinery baseline <design> --impl <dir>` from the observed import
  graph and is held to the committed `design/ratchet.json` snapshot: G4 fails the moment a
  baselined edge gains an offender file the snapshot does not record, so the amnesty never grows
  silently. `deny` plus `baseline` on the same edge is legitimate and recommended (the deny
  records the intent, the baseline the debt; the deny takes over when the baseline entry is
  deleted); `allow` plus `baseline` is a G2 contradiction (an allowed edge is not a violation).
  `ratchet.json` is generated: never hand-edit it, rerun `machinery baseline` (which also
  tightens the snapshot after debt is burned down). G4 prints the ratchet snapshot's date and age
  in days as a non-blocking note on every run, so tolerated debt stays visible instead of
  quietly aging.
- **Ids** (boundary and external) are dot-separated segments; each segment starts with a letter
  or underscore and continues with letters, digits, underscores, and hyphens
  (`crm.api`, `external.rest-of-monolith`). A segment starting with a digit or a hyphen never
  matches its mitigation row. Decomposition subsystem ids are stricter: bare names (letters,
  digits, underscores, hyphens, interior dots), because they become path segments.
- `contract_version: 2` names this format.

G2 verifies: boundaries bind to `workspace.dsl` elements, no duplicate ids, no edge both literally
allowed and literally denied, no edge both allowed and baselined, no wildcard in a baseline rule,
no rule referencing an
undeclared boundary or external, and the mitigation coverage below. G4-import later enforces the
rules against the code, including the ratchet on baselined edges.

## Interface / boundary contracts (feed the hard-TDD contract tests)

Domain contracts (invariants) come from Modelith. Phase 2 adds **interface contracts** at each boundary,
which is what the test-writer needs for contract tests. For every relationship crossing a boundary, pin:

- **shape**: request and response schema (JSON Schema, OpenAPI fragment, or protobuf message).
- **errors**: the enumerated error responses (these become `onError` branches in Phase 3).
- **idempotency**: is the call safe to retry, and keyed by what.

Format rules, checked by G2. The obliged set is `dependency_rules.allow`: the contract already
enumerates every crossing the design permits, so coverage of that closed list is checked rather than
attested (the same reasoning that makes the placement table's completeness checkable).

- The table header must name **edge** (or **crossing**, the older wording, accepted so a design that
  already uses it is told what its rows lack rather than that it has no table), plus **shape**,
  **errors**, and **idempotency**. Every header-matching table in the document is an
  interface-contract table and their rows are read together, so splitting the contracts across
  sections hides nothing.
- The **edge cell** names one or more edges as `from -> to`, written with the contract's own boundary
  and external ids, optionally backticked, separated by commas. Ids are matched whole: a row for
  `a.x -> b.y` credits that pair and nothing else, so a longer id never satisfies a shorter one.
  Annotations, the waiver included, go in parentheses. Anything else in the cell fails loudly rather
  than being half-read: a chain (`a -> b -> c`) is not a pair, and a wildcard is not a contract.
- **One row may cover several edges** when they genuinely share one interface: several consumers of
  the same provider (`app -> store`, `jobs -> store`), or several importers of one shared type
  module. Every pair the cell names is credited with that row's shape, errors, and idempotency, so
  group edges only when all three answers hold for each of them.
- **Every column is answered.** An empty shape, errors, or idempotency cell is an unanswered
  question, not a contract. "none" and "n/a (pure)" are answers; blank is not.
- **Coverage**: every concrete allow edge needs a row, or the waiver `(no contract: <reason>)` in its
  edge cell. A wildcard allow rule names no concrete pair and so carries no obligation, exactly as it
  carries none in the acyclicity and reachability checks.
- **No drift the other way**: every edge a row names must be an allow edge. A contract for a denied,
  undeclared, or merely baselined edge describes an interface the architecture does not have (a
  baselined edge is tolerated debt to burn down, never a designed interface).

| edge | shape | errors | idempotency |
|---|---|---|---|
| `app -> store`, `jobs -> store` | `Store` interface: `Load(id) -> (T, version)`, `Save(T, expectedVersion)` | `ErrNotFound`, `ErrConflict`, `ErrUnavailable` | `Save` is idempotent under `(id, expectedVersion)` |
| `store -> external.db` | pgdriver connection plus SQL; no driver type escapes `store` | driver errors mapped here onto the typed set above | one transaction per save, so a retry repeats the whole unit |
| `app -> model` | type-only: the shared vocabulary; no functions with side effects | none; nothing here can fail | n/a: no calls cross this edge |
| `svc -> external.bus` (no contract: authored in the child design that owns this subsystem) | n/a | n/a | n/a |

Three waiver tokens now live in this document and each answers a different question, so never
substitute one for another: `(no machine: <reason>)` waives a placement row's MACHINE,
`(not placed: <reason>)` waives an entity's PLACEMENT, and `(no contract: <reason>)` waives an
edge's INTERFACE CONTRACT.

What stays attested: whether the stated shape, error set, and idempotency rule are the RIGHT ones.
The gate holds the row and its columns; only a reader can tell whether the shape matches what the
code will actually exchange.

## Dependency mitigation posture (drives Phase 3 failure transitions)

For every external dependency, fill one row. This is what reclassifies failures rather than deleting
them. Format rules, checked by G2:

- The table header must contain **failure** and **mitigation** columns.
- The **first column** of each row names the dependency by its backticked `workspace.dsl` element id
  or contract external id (e.g. `` `db` ``, `` `q` ``, `` `store` ``). A backticked name that matches
  neither is an error (typo catch).
- **Required coverage**: every contract external plus every DSL element tagged Database, Queue, or
  External must have a row (an external may be covered via its bound dsl element).
- Every residual failure state, in particular any FailedDirty-style one, must say how an operator
  learns about it: add a detection/alerting column, or an operator-signal note in the residual column
  (log line, metric, alert).
- **Optional `handled by` column (opt-in, resolved by Gx-trace once machines exist)**: each row
  names, backticked, the machine or invoke actor that carries its residual transitions
  (`` `Deal` ``, `` `saveDeal` ``), or waives with `(no residual: <reason>)` (a fatal-and-loud
  posture needs no machine transition). Every name must resolve against the committed machines and
  their invoke actors; whether the named transitions are semantically adequate stays attested. A
  table without the column carries no obligation.

| dependency | failure modes | deployment mitigation | residual behavior the FSM must handle | bound | operator signal |
|---|---|---|---|---|---|
| `db` (PostgreSQL) | unavailable, slow, conflict | K8s + operator, HA failover, PgBouncer | transient unavailable during failover; serialization conflicts | retry <= 3, ~5s window | `db_retry_exhausted` metric + alert |
| `pay` (Payments API) | 5xx, timeout, duplicate | none (third party) | timeout, partial charge, must be idempotent | timeout 10s, idempotency key | `payment_failed_dirty` alert per stuck order |
| `q` (Queue) | unavailable, redeliver | clustered, at-least-once | duplicate delivery, must dedupe | dedupe by message id | dedupe-drop counter, redelivery log line |

## Adoption closure (a technology choice is a closure, not a node)

Adopting a technology adopts its operational closure: the stateful backends, sidecars, operators,
credentials, and network egress it needs to run the way you will actually run it. The closure is
invisible to code-level tools (no scanner reports that a WAF's cross-replica rate limiting needs a
shared key-value store); it lives in deployment artifacts (Helm charts, operator docs, reference
architectures), so it must be interrogated, not scanned. For every adopted technology, before the
mitigation table is considered complete:

1. **Enumerate the closure.** Ask "what does this bring with it?" against how it will actually be
   deployed: stateful backends, sidecars and operators, required credentials or tokens, network
   egress. Read the deployment artifacts, not just the project README.
2. **Treat every closure member as a first-class dependency.** Each one gets the same treatment as
   the technology that dragged it in: a license check, its own mitigation row, and a note on the
   operational and evidence surface it adds (backup, monitoring, compliance evidence).
3. **Ask the amortization question.** Now that the closure member is paid for, what else should it
   do, and does it let you consolidate something out? Guard: amortization must never corrupt
   boundaries. A dependency adopted as passive state does not thereby become a message bus or a
   shared cross-component store.
4. **Record risk evidence, not vibes.** Score each OSS adoption candidate with OpenSSF Scorecard
   and put the number, dated, in its decision box (DECISIONS.md). Zero-install lane: public GitHub
   repos are scanned weekly; `curl -s https://api.securityscorecards.dev/projects/github.com/<org>/<repo>`
   returns the latest score. CLI lane (unscanned or non-GitHub repos): the `scorecard` binary
   (`machinery preflight` reports it; needs `GITHUB_AUTH_TOKEN` at run time). A low score is a
   discussion item with flip conditions, never an automatic veto.

Carrying a discovered closure is checkable; author the **adoption-closure table** (opt-in,
activated by its own presence) and G2 holds it:

| technology | closure members | scorecard |
|---|---|---|
| PG operator | `store`, `external.vault` | 7.5 (2026-08-01) |
| argon2id via x/crypto | (no closure: a pure in-process library; no backends, credentials, or egress) | n/a - maintained by the Go team |

- The header names **technology** and **closure** columns; a **scorecard** column is optional.
- Every backticked closure member must be a declared `workspace.dsl` element or contract external
  AND have a mitigation row: a closure member is a first-class dependency, so it gets the full
  dependency treatment or the row fails.
- An empty closure cell needs `(no closure: <reason>)`; an empty reason is an error.
- When the scorecard column exists, every cell is `<score> (YYYY-MM-DD)` or `n/a - <reason>`; an
  empty or misshapen cell is an error, so the dated-evidence rule stops being a vibe.
- A **license** column is optional in the same way: when it exists, every cell is an SPDX id or
  expression (`Apache-2.0`, `GPL-3.0-or-later OR MIT`) or `n/a - <reason>`; an empty or prose cell
  is an error, so the license check the adoption step asks for has a checkable home.

What stays LLM-attested: DISCOVERING the closure. G2 verifies mitigation coverage and closure
carry for what is declared; only the conversation catches what was never declared.

## NFR record (part of the Architecture Contract conversation)

G2 checks the shell deterministically: a heading containing "NFR" (or "non-functional") must exist
and its section must mention all three topics below ("out of scope, recorded as such" satisfies the
check by construction). The CONTENT stays attested. Record these during Phase 2, even when the
answer is "out of scope, recorded as such":

- **security posture**: authn/authz approach, secret handling.
- **capacity assumptions**: expected volume, latency budget where relevant.
- **observability**: what must be logged, metered, alerted; in particular, every FailedDirty-style
  residual state needs a stated operator signal (see the mitigation table rule above).

## Action ownership (opt-in, checked by G2 when the table exists)

"Every Modelith action maps to an owning component" was an attested line; both sides are closed
sets, so give the mapping an artifact and G2 holds it in both directions. The header names an
**action** column and an **owning component** (or **owner**) column; rows group freely:

| action | owning component |
|---|---|
| `Deal.create`, `Deal.win` | `domain` |
| `User.login`, `User.logout` | `session` |

- Every action cell entry is an `Entity.action` key resolved against the domain model.
- Every model action appears exactly once across all rows, or is waived with
  `(unowned: <reason>)` in the owner cell; a missing or doubly-owned action is an error.
- Every owner is a backticked `workspace.dsl` element identifier or Architecture Contract
  boundary/external id.
- A design without the table keeps the old posture: the mapping is attested.

## Persistence and placement (the C4 to FSM bridge)

For every **stateful** component, decide and record. This determines how the Phase 3 machine is realized
and how concurrent events are serialized. Format rules, checked by Gx-trace once machines exist:

- The table header must contain **placement** and **persistence**. The design has exactly one such
  table; deleting it is an ERROR, not a way to have no obligations.
- The **first column** names its components in backticks. A row may name several components that
  share one placement decision; the machine rule below binds the first. A name written inside a
  parenthetical annotation is prose about another row's component, never a placement of its own.
- Every named component must have a `machines/<Name>.machine.json`, or the row must contain the
  waiver text `(no machine: <reason>)`.
- **Completeness**: every entity the domain model declares must appear in some row's first column,
  or carry the waiver `(not placed: <reason>)` in a row of its own. Nothing is demanded for enums or
  scenarios, entities only.

| component | machine placement | persistence | concurrency serialization |
|---|---|---|---|
| `Order` aggregate | in-memory actor (Elixir GenServer per id via Registry) | event-sourced to Postgres, rehydrate on start | actor mailbox (one process per order) |
| `Order` (Go/Rust/Python alt) | none; load-act-save | `state` column + `version` | optimistic lock (`WHERE version = ?`) or `SELECT ... FOR UPDATE` |
| `Pricing` (no machine: pure transform, contract spec instead) | n/a | none | n/a |
| `Address` (not placed: a value object stored inline in the `Order` row) | n/a | n/a | n/a |

The two waivers answer different questions, so keep them apart. `(no machine: <reason>)` says the
component IS placed and its row records that placement, but no state machine realizes it.
`(not placed: <reason>)` says the entity has no placement of its own at all (it is value-like, or
its persistence rides another aggregate's row), and therefore no machine either.

Why completeness is checked here and only attested for dependencies: the universe of dependencies is
open, so a dependency nobody declared carries no obligation and only the conversation can catch it.
The entity list is CLOSED, enumerable from the domain model, so an entity that never got a row is a
hole a tool can see. It is a real one: a persisted entity outside this table is invisible to every
gate, including the machine rule above, which only ever holds the rows that exist. Prefer an honest
row over a waiver. A persisted record with no lifecycle is a row (`(no machine: ...)`), not a
waiver; the waiver is for the genuinely unplaced.

Elixir maps almost 1:1 to a supervised process per aggregate. Go, Rust, and Python need the explicit
persisted-state plus lock pattern, or an event-sourced log, because there is no cheap per-entity process.

## Event-contract table (required for multi-component designs)

Coupling through shared DB tables or bus topics is **invisible to G4-import**; this table is the
governing artifact for it. One row per event that crosses a component boundary (every external event
a machine consumes in a choreography must appear here; see the xstate reference for the redelivery
rule). Like the surface ledger's `source:` lines, state where the rows were enumerated FROM,
publisher-first: sweep the code for emit/publish call sites AND the broker or infra configuration
(topic definitions, subscriptions, queue bindings), plus any API spec; name both lanes in a source
note, or waive one with a reason. A table with no named enumeration source is a completeness claim
with no evidence, and the gates can only check the rows that are declared. G2 enforces the note's
presence: every event-contract table (a header naming producer, consumer, and delivery) must have a
`Source:` line (or the phrase "enumerated from", or a `machinery:embed` marker) within the five
lines above its header. Columns:

- **event**: the event name, as the machines spell it. In prose the cell names it backticked
  ("`reserved` / `released` events"); in the machine-checkable format below it is the bare name.
- **producer**: one component, named as the `workspace.dsl` element names it.
- **consumer**: one component, same rule.
- **payload**: by Modelith attribute reference, never redefined shapes.
- **delivery**: at-least-once / at-most-once / exactly-once-effect, and the mechanism.
- **ordering assumption**.
- **dedupe key**.

G2 holds every ROW to those columns: an empty cell is an unanswered question and an ERROR, while
an explicit "none" or "n/a" is an answer and passes, and the producer and consumer must resolve to
a declared `workspace.dsl` element or a declared external, the same resolution a mitigation row
gets. One consistency rule crosses cells: a row whose delivery is at-least-once with a BARE
"none"/"n/a" dedupe cell is an ERROR, because duplicates will arrive; name the key, or state the
reason in the cell (`none (idempotent consumer: upsert by Order.id)`). Once machines exist,
Gx-trace reconciles the rows against them (see the Gate 4 checklist), and a machine that declares
`_external_events` arms the reverse direction: each declared externally-sourced event must have a
row here.

| event | producer | consumer | payload | delivery | ordering | dedupe key |
|---|---|---|---|---|---|---|
| `ORDER_PAID` | `orderSvc` | `shipmentSvc` | Order.id, Order.total | at-least-once (outbox -> `q`) | per-order FIFO (partition by Order.id) | Order.id + event type |

### Consumer-READS completeness (opt-in)

The payload column says what the event CARRIES. Nothing says what the consumer READS, so a row can
declare a reaction the payload cannot support and every gate passes: the cells are answered, the
reaction is declared, the consumer's invariants have enforcement rows, and the insufficiency is
invisible. A design closes that by putting a marker in ARCHITECTURE.md:

```
<!-- machinery:reads-complete -->
```

Anywhere in the document, by convention directly above the event-contract section. It is a claim
about the whole contract, so it arms EVERY event-contract row the document carries (the contract is
legitimately split across several tables). Armed, Gx-trace holds each row to two obligations:

- some matrix line naming the event whole-token declares `READS{field, ...}` (the same declaration
  syntax the opt-in payload-sufficiency check has always read; the natural home is a "consumed
  events" section of the consuming machine's matrix). No declaration is an ERROR naming the row and
  the event.
- every declared field appears whole-token in THAT ROW's payload cell. A field the payload does not
  carry is an ERROR: the payload-sufficiency drift, which is a warning while unarmed.

A consumer that genuinely reads nothing off the payload (a pure signal: it reacts, then refetches by
id) waives with `(no reads: <reason>)` in the consumer cell. The reason is mandatory, as with every
house waiver. A `(no machine: <reason>)` waiver does NOT discharge the reads obligation: it answers
whether anything reacts as a machine event, and a consumer reacting through an invoke actor still
reads the payload. The `checked:` line reports the tier: reads declared, waived, missing, and the
declared fields carried.

Arming is per design document, so on a decomposed child the embedded rows include the ones its peer
consumes; a shard arms the tier only once it can answer for every row it carries (a cell it must
answer differently from the parent is `(shard-local: <reason>)` territory, the Ge escape). An
unarmed design carries no obligation at all.

### Machine-checkable format (mandatory on a decomposed parent)

`machinery pack generate` extracts each subsystem's boundary events from this table by exact
component name, so on a decomposed parent the table is a machine-read artifact with a format
contract, enforced at generation time and again by G5 (which regenerates packs in memory):

- EVERY markdown table whose header names producer, consumer, and delivery is an event-contract
  table: pack generation parses all of them and concatenates their rows (row numbers run
  cumulatively), so splitting the contract across several tables hides nothing.
- an **event** column names every event; no row leaves it empty.
- **producer** and **consumer** cells each hold exactly one component name: a `components:` entry
  from `decomposition.yaml` or an Architecture Contract boundary `element` (a gateway or ui that
  owns no entities and gets no pack is still a valid participant). Contract externals do not
  qualify; an external system that genuinely produces or consumes events must be declared as a
  boundary.
- annotations are allowed only in parentheses: `gateway (SSE)` resolves to `gateway`. Backticks
  are stripped the same way.
- fan-outs are expanded explicitly: one row per producer-consumer pair, never `ALL components`, a
  comma list, a slash, or an arrow.
- rows between two non-pack participants (gateway to ui) are validated like every other row; they
  emit no pack rows.
- G2's row check (above) is the looser sibling of this one: it accepts a declared external as a
  participant and does not require the event column, because an ordinary design's table is not a
  generation input. Where the two differ the pack is stricter, so nothing G2 accepts makes pack
  generation pass silently.
- generation FAILS on any violation, naming the row and the offending cell text; nothing non-empty
  is ever silently dropped. A subsystem extracting zero boundary events is also a generation error
  unless `decomposition.yaml` waives it per subsystem with `boundary_events: {none: "<reason>"}`;
  the reason must be a single line (an embedded newline could forge the generated events.md count
  line), and the generated events.md then carries the reason instead of the boundary-completeness
  claim.

`decomposition.yaml` also carries two root keys the packs reflect: `revision:` (integer >= 1,
default 1), the monotonic amendment counter emitted into every pack manifest as `pack_revision`
(bump it on any pack amendment, so children see rev N -> N+1 on the G5 checked line), and
`retained: {<invariant-id>: "<reason>"}`, the top-level invariants the parent keeps enforcing
itself. Every top-level invariant must be delegated to exactly one subsystem or retained with a
reason; both at once, or neither, fails generation. The full decomposition protocol, including the
pack amendment steps and the unpinned-child hash limit, is in the skill's "Recursive
decomposition" section.

## Gate 2 checklist

Deterministic (run `machinery check <design> --gate g2`):

- The contract parses, boundaries bind to `workspace.dsl` elements, ids are unique, no edge is both
  literally allowed and literally denied, no rule references an undeclared boundary or external.
- Every relationship `workspace.dsl` DRAWS is judged by the same allow/deny/baseline rules
  G4-import judges a code edge by: a drawn edge the contract denies, or that no rule covers, is an
  ERROR, and an endpoint no element declares is an ERROR. An endpoint the contract never claimed (a
  person, a system-context box, a container outside the contract) is outside the dependency
  vocabulary and carries no obligation; the `checked:` line counts it so the unjudged remainder
  stays visible. The converse is NOT required: a diagram is legitimately partial, so an allow rule
  nothing draws is no finding.
- Every contract external and every Database/Queue/External-tagged element has a mitigation row
  naming it backticked in the first column. Coverage is over DECLARED dependencies only: a
  dependency never declared in the DSL or the contract carries no obligation, so completeness of
  the declaration itself is attested, not checked.
- Every concrete allow edge has an interface-contract row with shape, errors, and idempotency all
  answered, or a `(no contract: <reason>)` waiver; and every edge a row names is an allow edge.
- The NFR record exists and mentions security, capacity, and observability.
- Every event-contract table names its enumeration source (a `Source:` note or an embed marker
  directly above the header), and every ROW answers its columns and names participants the model
  declares (see the column list above).
- Every table of each kind counts, never just the first: the event, interface, mitigation, and
  placement locators all take EVERY header match and aggregate their rows, so a design that splits
  a table across sections hides no row from its obligation.
- When the design authors them (opt-in by presence): the action-ownership table is closed in both
  directions, every adoption-closure member is a declared and mitigated dependency (scorecard cells
  well-shaped), and every mitigation `handled by` name resolves against the committed machines
  (that one runs in Gx-trace once machines exist).
- Read the `checked:` counts; an empty check is an ERROR, never a silent pass.

Engine phase (needs Java, like verify-formal): `machinery verify-c4 <design>` compiles
`workspace.dsl` under `structurizr-cli export`; a design is not done with Phase 2 until it passes.

LLM-attested (you verify; the tool cannot):

- Every Modelith action maps to an owning component in `workspace.dsl`, when the design declines
  the action-ownership table (author the table and this is checked instead).
- Whether each interface contract is the RIGHT one: that the shape matches what the code will
  actually exchange, the error list is exhaustive, and the idempotency claim survives a retry. That
  every crossing HAS one is now checked (see above).
- Every stateful component has a persistence-and-placement decision, and each decision is the RIGHT
  one (whether a row's placement, persistence, and serialization actually hold is judgment). Two
  deterministic halves run in Gx-trace once machines exist: a machine per row, and a row per
  declared entity.
- The event-contract table covers every cross-component event (its rows are open-world; the source
  note's presence is checked, its truth is not). Each row's SHAPE is now checked (columns answered,
  participants declared), and once machines exist Gx-trace reconciles each row's event against them:
  a consumed event is handled or `_ignores`-ed by some machine, a produced event appears whole-token
  in some machine action or matrix cell, or the cell that owes the obligation carries a
  `(no machine: <reason>)` waiver. The reverse sweep stays attested: nothing in a machine marks an
  event as externally sourced, so "every external event a machine consumes appears here" is a claim
  the source note evidences rather than a checked set.
- The dependency declaration is complete: everything the deployment actually talks to appears in
  the DSL or the contract (the mitigation-coverage check runs only over what is declared).
- The NFR record's content is true (presence and topic coverage are checked).
