# The legacy surface ledger and the Gs-surface gate

When a design run has a legacy system behind it (rebuild, hybrid, a greenfield run that keeps
the old code as evidence, or brownfield archaeology), the design's completeness has a failure
mode no other gate can see. `Gm-transition` proves every entity in
`design/legacy/domain.modelith.yaml` has a disposition, but its coverage universe is
self-declared: nothing anchors the legacy *model* to the legacy *system*. If the opening
excavation missed a subsystem, the legacy model omits it, every disposition it would have
needed is silently absent, and Gm passes green. The capabilities most likely to be missed are
exactly the ones that live outside the documents: route inventories, table names, scheduled
jobs, and integration endpoints.

The surface ledger closes that hole. It is a checked inventory of the legacy system's
mechanically enumerable surface, with every item mapped to a target design element or given an
explicit dropped or deferred disposition. Unmapped surface fails loudly.

## The two bookend sweeps

The ledger is authored by two named sweeps that bracket the design run:

1. **Opening sweep (Phase 0/1).** Enumerate the legacy surface mechanically: route tables,
   CLI command registrations, schema catalogs, cron and worker lists, outbound API calls,
   queue topics. Use the codebase graph when the runtime has one. Record every item, most of
   them as `deferred` with an opening rationale, because the target design they would map to
   does not exist yet. The opening ledger is the interrogation's work list, not its answer.
2. **Closing sweep (after Gate 4).** Once the design stands, re-mine the legacy system against
   the finished design and settle every row: `covered` (a design element carries the
   capability), `dropped` (deliberately not carried, with rationale), or `deferred` (punted to
   a later iteration, with a real rationale, never an opening placeholder). The diff between
   the opening and closing ledgers is the sweep's work product; anything the docs-first
   interrogation missed surfaces here as a row that cannot be honestly disposed.

The gate holds the ledger's internal consistency and its bindings at every stage. What it
cannot prove is that the enumeration itself is complete; that stays with the conductor, and
the ledger's structure is designed to force the question: every surface class must be either
inventoried with its evidence source or explicitly waived with a reason.

## The artifact

`design/legacy/surface.yaml`. It is independent of `migration.yaml` by design: a run that
drops the migration machinery (a clean-break rebuild reclassified to greenfield-with-corpus)
keeps its completeness anchor.

```yaml
surface_version: 1
system: go-crm v1, a single-binary CLI CRM over an embedded LadybugDB graph store
classes:
  routes:
    none: The legacy system is a local CLI binary; it exposes no network API.
  commands:
    source: impl/internal/cli/command.go (cobra command registrations)
    items:
      - name: crm login
        disposition: covered
        via: action
        target: User.login
      - name: crm export legacy-graph
        disposition: dropped
        rationale: One-shot migration tooling; the target's exporter replaces it.
  tables:
    source: impl/internal/repo/repo.go (node labels and relationship types)
    items:
      - name: node label Deal
        disposition: covered
        via: entity
        target: Deal
  jobs:
    none: No scheduled or background work; every effect is a synchronous command.
  events:
    none: Single-process binary; no queues or topics.
  integrations:
    none: No outbound calls to external services.
```

### Root keys (strict; unknown keys fail the gate)

| key | required | meaning |
|---|---|---|
| `surface_version` | yes | the integer `1` |
| `system` | yes | one line naming the legacy system and its shape |
| `classes` | yes | the six surface classes, all of them (see below) |
| `_comment` | no | free text |

### The six surface classes

`routes`, `commands`, `tables`, `jobs`, `events`, `integrations`. Every class must appear
under `classes`, each as exactly one of:

- **An inventory:** `source` (where the enumeration came from: the file, catalog, or command
  that produced it) plus `items` (a non-empty list). An empty `items` list is an error; use a
  waiver instead.
- **A waiver:** `none: <reason>`. The reason is required. A waiver combined with `source` or
  `items` is an error.

The fixed vocabulary is the point: a class you forgot to enumerate is a missing key, which is
an error, never a silent pass. The classes map to what is mechanically enumerable in practice:
network API surface (`routes`), CLI surface (`commands`), persistent shape (`tables`, meaning
tables, collections, node labels, or file stores), scheduled and background work (`jobs`),
async topics consumed or produced (`events`), and outbound dependencies (`integrations`).

### Items

| key | required | meaning |
|---|---|---|
| `name` | yes | the surface item, unique within its class (`POST /contacts`, `node label Deal`) |
| `disposition` | yes | `covered`, `dropped`, or `deferred` |
| `via` | covered only | `entity`, `action`, `component`, or `machine` |
| `target` | covered only | the design element the capability maps to |
| `rationale` | dropped/deferred | why the capability does not carry into this design |

Rules, all deterministic:

- `covered` requires `via` and `target`, and the binding must resolve:
  - `via: entity` binds `target` to an entity in `design/domain.modelith.yaml`.
  - `via: action` binds `target` (as `Entity.action`) to an action on that entity.
  - `via: component` binds `target` to a `design/workspace.dsl` element name.
  - `via: machine` binds `target` to `design/machines/<target>.machine.json`.
- `dropped` and `deferred` require `rationale` and must not carry `via` or `target`; a
  deferred capability with a design element to point at is a covered capability.
- A duplicate `name` within a class is an error.

Bindings resolve against the **target** design, never the legacy model: the ledger's question
is "does the new design account for this", and the legacy model is itself under suspicion of
being incomplete.

## The gate

`Gs-surface` activates automatically when `design/legacy/surface.yaml` exists, exactly as Gm
activates on `migration.yaml`. Run it alone while authoring:

```bash
machinery check design --gate gs
```

It verifies, deterministically: the schema is strict (unknown keys fail), all six classes are
present and well-formed, every item is disposed exactly once with its required fields, every
covered binding resolves against the target design, and names are unique per class. The
`checked:` line reports per-class item counts plus the covered, dropped, deferred, and waived
totals, so a handoff review sees the disposition profile at a glance. An empty ledger (six
waivers, zero items) is an error: a legacy system with no enumerable surface at all is not a
legacy system.

LLM-attested (the conductor checks these; the tool cannot): the enumeration is complete for
each inventoried class, the waivers are true, and at Gate 4 every `deferred` rationale is a
deliberate decision rather than an opening-sweep placeholder. The deferred count in the
`checked:` line exists to make that attestation reviewable.

### Staging

The gate needs `design/domain.modelith.yaml` to resolve entity and action bindings, so the
earliest useful run is during Phase 1. An opening-sweep ledger binds via `entity` and `action`
or defers; `component` bindings become available when `workspace.dsl` exists (Phase 2) and
`machine` bindings when the machines land (Phase 3). A covered row whose binding artifact does
not exist yet is an ordinary staging error, the same way Gm reports the narrative bridges
until ARCHITECTURE.md and BUILD.md exist: author early, expect those findings, and clear them
as the phases land.

## Relationship to the other gates

- **Gm-transition** covers the declared domain truth: every legacy *entity* disposed, every
  replaced attribute and lifecycle value mapped. **Gs-surface** covers the observable system:
  every *capability* disposed. Together they close both ends; either alone leaves a hole. Gm
  without Gs trusts the legacy model to be complete; Gs without Gm has no data-migration
  contract.
- Gs carries no generated artifacts, so it has no DRIFT class; the ledger is a source file.
- In **brownfield** adoption the ledger works unchanged (the "legacy" system is the current
  one) and gives the staged `--gate` list a coverage anchor from day one: add `gs` to the
  day-one list alongside `g2,g4`.

## Modes

| mode | ledger | why |
|---|---|---|
| rebuild / hybrid | required | the legacy surface is the thing being replaced; unmapped surface is unshipped scope |
| greenfield with a legacy corpus | required | the migration machinery may be dropped, the completeness anchor must not be |
| brownfield | recommended | same anchor question, asked of the system being brought under gates |
| pure greenfield | not applicable | there is no legacy surface to enumerate |

## Worked example

`examples/surreal-crm/design` is the complete reference: a rebuild of the go-crm CRM that
keeps the domain intact and replaces the embedded LadybugDB store with SurrealDB on a Docker
instance. Its ledger enumerates the real CLI surface from `examples/go-crm/impl` (command
registrations, node labels, the session file) and demonstrates all three dispositions plus
class waivers. Because the domain shape is unchanged, its `migration.yaml` disposes every
entity as `reuse`, and the interesting coverage lives exactly where this gate looks: the
commands and the store.
