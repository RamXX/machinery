---
name: machinery
metadata:
  version: "0.3.2"
description: >
  Design software as a build-ready blueprint, greenfield, brownfield, hybrid, or rebuild. Use when the user
  wants to design a new system, service, or app from scratch, produce a BUILD.md for a
  coding agent, spec something out before implementation, express program logic as state
  machines, model an existing system as it is, bring a legacy repo under control, or
  adopt machinery on an existing codebase, preserve valuable behavior while replacing a prototype,
  or plan legacy/target coexistence and cutover. Runs a four-phase interrogation: domain model
  (Modelith) then architecture (C4) then state machines (XState) then BUILD.md. Triggers:
  "design a new ...", "greenfield", "brownfield", "hybrid", "rebuild", "replace the platform",
  "save what can be saved", "build a blueprint", "spec this out",
  "state machine design", "produce a BUILD.md", "design agent", "model an existing
  system", "adopt machinery".
---

# machinery

Turn a fuzzy product idea into a `BUILD.md` that a coding agent with zero prior context
can implement under hard TDD. You are the conductor. You interrogate the user, enforce a
gate between every phase, reuse the `modelith` tooling, embed the C4 technique, and delegate
the heavy synthesis to the two author roles (as subagents where your runtime spawns them,
otherwise inline). You do not write production code here. The artifact is the design.

## Runtime capability contract

Run the methodology from capabilities, never from a host name. The skill, design artifacts, and
`machinery` CLI are the portable core; commands, subagents, and lifecycle hooks are optional host
accelerators. At the start of a run, use the strongest capability the runtime actually exposes:

- If fresh-context subagents are available, delegate Phase 3 to `machinery-fsm-author` and Phase 4
  to `machinery-build-writer`. If either role or delegation is unavailable, execute that role's
  canonical instructions inline and keep the same inputs, outputs, and gates.
- If host commands are available, they may start or resume this skill. If not, a plain-language
  request such as "design this rebuild with machinery" is equivalent.
- If edit and stop hooks are available, let them protect generated artifacts and run the inner-loop
  gates. If they are absent or advisory, treat generated artifacts as read-only by instruction,
  run `machinery check` explicitly before every phase handoff, and rely on the consuming repository's
  CI check as the authoritative enforcement boundary.
- Never require a host-specific tool name, model name, frontmatter field, or command syntax in a
  design artifact. Detect available read, write, search, shell, question, and delegation tools and
  adapt locally. Host adapters may wrap this contract; they may not weaken or redefine it.

The result must be equivalent across Claude Code, Codex, OpenCode, and any Agent Skills runtime:
the same canonical role bodies, artifact schemas, deterministic gates, and BUILD.md handoff. What
may differ is ergonomics and how early a violation is caught, never what counts as a valid design.

## The thesis

Most software is a state machine. machinery makes that explicit across three layers that
compose, they do not just sit next to each other:

- **Domain model (the what)** names what exists and what must always be true.
- **C4 (the how)** fixes how it is built, deployed, and what each dependency does when it fails.
- **State machine (the behavior)** is every state, transition, guard, timeout, and failure
  mode, conditioned on the deployment C4 already fixed.

The FSM is authored **last** because it needs both prior layers as inputs. But half of it
is *derived*, not invented: the domain lifecycle is already latent in the domain model. The
FSM phase is a synthesis step that weaves the domain lifecycle together with a
failure-and-recovery overlay that only C4 can inform.

## The traceability spine (hold this the whole way through)

| Domain model (Modelith) | State machine (XState) | Architecture (C4) |
|---|---|---|
| entity status enum values | states (atomic / compound) | - |
| `action` name | `event` + transition | component that owns the action |
| `invariant` id | `guard` on the transition, or a structurally impossible edge | component enforcing it |
| `action.preserves: [inv-id]` | the guard names that invariant id | - |
| entity attributes + types | typed `context` shape | datastore schema |
| side effect implied by an action | `invoke` actor with `onDone` / `onError` / `after` timeout | relationship to a datastore or external system, and its mitigation posture |
| `scenario` + `invariants_touched` | a path through the machine, which is a test case | - |

Rules that keep the layers aligned:

- **Modelith is the single source of truth for data.** C4 datastores and XState `context`
  reference the Modelith attributes. They never redefine them. Three schemas drift; one does not.
- **Every state traces to an enum value, every event to an action, every guard to an invariant id,
  every `invoke` to a C4 relationship.** If any invariant has no enforcing guard and is not
  structurally impossible, that is a hole. Flag it, do not paper over it.

## Output layout

Write everything into a single `design/` directory in the target project so nothing litters:

```
design/
  domain.modelith.yaml      # Phase 1 (source of truth for the domain)
  domain.modelith.md        # generated by `modelith render` (writes <name>.modelith.md beside the source)
  legacy/domain.modelith.yaml # rebuild/hybrid only: current domain truth
  legacy/surface.yaml       # any run with a legacy system: capability disposition ledger (Gs)
  migration.yaml            # rebuild/hybrid only: checked legacy-to-target transition contract
  workspace.dsl             # Phase 2 (Structurizr C4 model)
  ARCHITECTURE.md           # Phase 2 narrative + Architecture Contract (+ event-contract table)
  machines/<Component>.machine.json   # Phase 3 (one per stateful component)
  machines/<Component>.matrix.md      # Phase 3 named-unit contracts + failure catalog (+ optional transition table)
  machines/<Component>.oracle.md      # GENERATED by machinery oracle; canonical test oracle, do not edit
  formal/<Machine>.semantics.yaml     # Phase 3 formal annotation (source): lifecycle pattern per machine
  formal/<name>.composition.yaml      # Phase 3 formal annotation (source): cross-aggregate composition
  formal/policy.relational.yaml       # Phase 1 relational annotation (source): access policy (opt-in)
  formal/integrity.relational.yaml    # Phase 1 relational annotation (source): structural integrity (opt-in)
  formal/isolation.relational.yaml    # Phase 1 relational annotation (source): multi-tenant isolation (opt-in)
  formal/*.tla + formal/*.cfg         # GENERATED by machinery verify-formal; committed alongside the sources
  formal/Policy.als                   # GENERATED by machinery alloy from the domain model + policy annotation
  formal/Policy.oracle.md             # GENERATED by machinery alloy: the authorization decision table the impl tests consume
  formal/Integrity.als                # GENERATED by machinery alloy: the structural-admissibility model
  formal/Isolation.als                # GENERATED by machinery alloy: the multi-tenant isolation model
  formal/Isolation.oracle.md          # GENERATED by machinery alloy: the tenant-scoping decision table the impl tests consume
  BUILD.md                  # Phase 4 (the blueprint; manifest over BUILD/ when sharded; Build plan held by Gb)
  BUILD/<context>.md        # only for sharded designs (see "Sharding large designs")
  STATE.md                  # session ledger for multi-session runs (see "Session ledger")
```

## Phases and gates

Never advance until the current gate passes. State the gate result to the user before moving on.

The deterministic gates live in `machinery check <design> [--impl <dir>] [--gate gm,gs,gp,gi,gn,g2,g3,gx,gb,g4,gt,g5]`
(g5 runs automatically on decomposed designs; gm runs once `migration.yaml` exists; gs once
`legacy/surface.yaml` exists; gp, gi, and gn each run automatically once the matching
`formal/{policy,integrity,isolation}.relational.yaml` exists; gb once `design/BUILD.md` exists;
gt, like g4, only with `--impl`; see Recursive decomposition, the
legacy surface ledger, and the Phase 1 relational layers).
It is a single Go binary; install it with the one-line installer
(`curl -fsSL https://raw.githubusercontent.com/RamXX/machinery/main/install.sh | sh`) or
`go build ./cmd/machinery`.
Gates fail on absence: a missing artifact or an empty check is an ERROR, never a silent pass, and
every gate prints a `checked: <counts>` line of what it actually verified. Use `--gate` to run a
subset while a phase is in flight; the default runs everything applicable.

### Phase 0 - Frame (short)

One paragraph. What are we building, who uses it, the one-sentence purpose, and the target
language(s) from {Elixir, Go, Rust, TypeScript, Python}. Language matters: it changes how the
FSM is realized (Elixir `gen_statem` or a GenServer per aggregate is a near 1:1 mapping; Go, Rust,
Python need an explicit state field plus a lock or event sourcing) and it is a C4 input.

Also classify the run as greenfield, brownfield, rebuild, or hybrid. Existing code does not by
itself decide the mode: brownfield improves the current implementation in place; rebuild creates a
new target and retires the old one; hybrid keeps old and new truths in coexistence for a material
period. If the user wants to preserve selected behavior or assets while replacing the production
foundation, choose rebuild and use the transition contract below.

The mode is a standing decision, not a one-time Phase 0 fact. Any later answer about transition
posture (clean break vs coexistence, in-place vs replace, what data survives) is a
mode-reclassification trigger: restate the classification aloud, name which artifacts are picked
up or dropped (`legacy/domain.modelith.yaml`, `migration.yaml` and Gm, the surface ledger and Gs,
the G4 baseline/ratchet), and record the decision in DECISIONS.md. A clean break rarely means
nothing survives; ask "does any data or behavior survive?" explicitly before dropping the
migration contract. A run reclassified to greenfield-with-corpus (greenfield gates, legacy code
kept as evidence) keeps the surface ledger: that posture is exactly when the completeness anchor
matters most.

### Phase 1 - Domain model (the what)

Invoke the `domain-model-author` skill to run this conversation; use `domain-model-lint` for the
gate. When those skills are not installed in the agent home, run the interrogation inline and use
`modelith lint` directly as the gate. Interrogate breadth-first (skeleton, then invariants and
scenarios, then refinement). Push on naming and on "what must always be true."

When eliciting invariants and scenario edge paths, sweep the five EARS behavior categories by
name: ubiquitous (the system shall), event-driven (when X, shall), state-driven (while X, shall),
optional (where feature X, shall), unwanted (shall not X, even when Y). Interrogators and clients
under-specify the unwanted and state-driven categories most, and those two map directly onto
machinery constructs: unwanted behaviors become negative invariants now, failure-catalog rows and
falsifying-clause tests later; state-driven conditions become lifecycle states and guards in
Phase 3. This is an elicitation sweep, a writing discipline for the interrogation, not a gate and
not a required syntax for the model file.

After `modelith render`, strip em dashes from the generated `*.modelith.md` (the renderer emits them,
house style forbids them): `perl -CSD -i -pe 's/\x{2014}/-/g' design/*.modelith.md`.

**GATE 1:** `modelith lint design/domain.modelith.yaml` is clean (no errors). Every entity that has
a lifecycle has a status enum. Every action has its pre/post captured (`description` and `preserves`).
Every invariant has an owner (entity-level or top-level). Scenarios cover happy plus edge paths.

#### The static relational layers (Phase 1.5, opt-in but not optional when they apply)

Some domain invariants are static relations no state machine enforces and no structural lint can
falsify: TLC never sees them. There are three sibling algebras for them, each its own opt-in
annotation compiled by `machinery alloy design/` into a bounded relational model that
`machinery verify-formal` solver-checks alongside the TLC proofs. Each is independently opt-in; one
`machinery alloy` command emits every present layer. Author a layer only when the domain model
carries invariants of its shape:

- **Policy** (`gp`): access control (roles, ownership, team scoping). Proves nothing bad is
  permitted (UNSAT checks). Detailed below; full guide in `docs/policy-layer.md`.
- **Integrity** (`gi`): structural admissibility (uniqueness, singletons, cardinality). Proves the
  intended structures are admissible and scale (SAT runs). Guide in `docs/integrity-layer.md`.
- **Isolation** (`gn`): multi-tenant reference isolation. Proves no cross-entity reference crosses a
  tenant boundary. Guide in `docs/isolation-layer.md`.

The policy layer, in detail. If the domain model carries access-control invariants (any
`rbac-*`-shaped rule), author `design/formal/policy.relational.yaml`, a short declarative annotation
in a closed algebra, and run `machinery alloy design/` to compile it (plus the domain model) into
`formal/Policy.als`, a bounded relational model with a standard meta-check suite (satisfiability,
write-implies-read, capable-writes-own, reassign-retains-authority, per-role grant exercisability).

The annotation shape (see the go-crm example for a complete one):

- `subjects`: the acting entity, its role enum attribute, and optionally team scoping. State the
  membership multiplicity explicitly (`lone` or `one`) and which roles MUST hold a team
  (`required_for`): Modelith's `1:n` cardinality cannot express the difference, and that exact
  ambiguity is where the go-crm design hid a real defect (a teamless Manager whose write scope was
  provably empty).
- `resources`: the owned entities the policy governs (each needs an n:1 relationship to the
  subject entity in the domain model).
- `rules`: one per policy invariant. Three shapes: `grants` (role to verb capability),
  `verbs` + `scope` (per-role scope expression: `all | own | team | none`, unions of own|team),
  and `reassign` (actor scope AND a mandatory `target` per role: `any | team`; where a record may
  GO must be stated, not implied).
- `residuals`: every top-level invariant the algebra cannot carry, each with a reason. Coverage is
  a hard rule: a top-level invariant that is neither a rule nor a residual fails generation.

Authoring the annotation is itself an interrogation instrument: it forces the multiplicity
decision, the reassign-target decision, and any fuzzy definition (a scope defined over records but
used as a set of users cannot be written down in the algebra) back into Phase 1 conversation
BEFORE anything downstream consumes them.

The same generation also emits `formal/Policy.oracle.md`, the **authorization oracle**: the policy
enumerated as a role x verb x ownership-case decision table with content-derived stable ids (ids
hash the case, never the verdict, so a design revision flips expectations under unchanged keys).
The scope algebra decides every case from two booleans, so the table is the complete semantics of
the policy. The implementation consumes it the way machine tests consume the transition oracles:
one conformance test parses the table and asserts the pure authorization function on every
reachable row, expanding each abstract case into all policy-equivalent concrete variants (see
`examples/go-crm/impl/internal/authz/oracle_test.go` and the P-authz-oracle row in its BUILD.md).
Rows marked `unreachable` are configurations the domain invariants forbid; tests skip them and the
write discipline refuses to construct them. This closes the loop to code: TLC and Alloy hold the
design, the oracle holds the implementation to the design.

The integrity layer, in brief. If the domain model carries structural invariants (attribute
uniqueness, singleton flags, mandatory-relationship cardinality), author
`design/formal/integrity.relational.yaml`: `entities` to model, `relationships` (multiplicity read
from the domain model, the mandatory decision stated here), `unique` attributes, `singleton` flags,
each binding a domain invariant id. Where policy proves "nothing bad is permitted" (UNSAT checks),
integrity proves "the intended structures are admissible and scale" (SAT runs): `SomeWorld` (the
constraint set is jointly satisfiable), `Populatable` (every entity reaches a population target
under all constraints), per-unique `Distinct` witnesses, and inverse exclusivity checks for `1:1`
and `1:n` relationships (forward field multiplicity alone cannot prevent two sources from sharing
one target). Value domains are bounded to the exact type
cardinality, so uniqueness declared on a domain too small to populate (a boolean, a small enum)
fails `Populatable` at the solver. No oracle: integrity is a design-side property, held by the
`Gi-integrity` gate (freshness) and `verify-formal` (the solver). Full guide in
`docs/integrity-layer.md`.

The isolation layer, in brief. If the domain model carries cross-entity tenant-consistency
invariants (a record and the records it references must share a tenant), author
`design/formal/isolation.relational.yaml`: the `tenant` entity, the `subject` that holds a tenant
(with membership multiplicity), and the `references` that must stay in-tenant, each binding a domain
invariant. `tenant(record) = owner's tenant`; a reference is tenant-consistent when the two owners
share a tenant. The generated `Isolation.als` proves, most sharply, that two records in different
tenants can never share a referent (`SharedReferent`, a non-trivial consequence the single-hop facts
do not give for free). It also emits `Isolation.oracle.md`, the tenant-scoping decision table
(reference x tenant-case) the implementation's link-authorization function is held to, exactly as
the policy oracle holds the access code. Held by the `Gn-isolation` gate and `verify-formal`. Full
guide in `docs/isolation-layer.md`.

**GATE 1.5 (each runs only when its annotation exists):** `machinery check <design> --gate gp,gi,gn`.
**Gp-policy**, **Gi-integrity**, and **Gn-isolation** each verify deterministically: the annotation
reconciles against the domain model (entities, enum values, relationships, invariant ids all bind),
coverage holds (policy: every top-level invariant compiled or waived), and the committed artifacts
byte-match a fresh generation (staleness is DRIFT). The solver run itself happens in
`machinery verify-formal` (needs Java, same as TLC): every generated `check` must report no
counterexample and every generated `run` must find an instance. A FAIL prints the counterexample or
the vacuity note; treat it exactly like a TLC counterexample: a design defect, fixed in the domain
model (usually one invariant), never patched in the generated file.

### Phase 2 - Architecture (the how)

Author `design/workspace.dsl` (Structurizr) and `design/ARCHITECTURE.md` with a machine-checkable
Architecture Contract (v2 format: `element` bindings, `externals`, `ignore`; see
`references/c4-standalone.md`). Decide deployment topology and tech stack. For **every** external
dependency (datastore, queue, external API) declare a failure-and-mitigation posture. For **every**
stateful component decide persistence and placement (in-memory actor vs persisted aggregate). This
is the real bridge into Phase 3. For multi-component designs, also author the **event-contract
table** (producer, consumer, payload by Modelith attribute reference, delivery guarantee, ordering
assumption, dedupe key): coupling through shared DB tables or bus topics is invisible to G4-import,
so this table is the governing artifact for it. On a design that decomposes, the table must follow
the machine-checkable format in `references/c4-standalone.md` (an event column, exactly one
component per producer/consumer cell, annotations only in parentheses, fan-outs expanded to one
row per pair): pack generation extracts from it by exact component name and fails loudly on any
cell it cannot resolve.

A technology choice is a closure, not a node: adopting X adopts X's operational closure (stateful
backends, sidecars, operators, credentials, egress), and the closure lives in deployment artifacts,
not in anything a scanner reports. Enumerate it, give every member the full dependency treatment
(license check, mitigation row, evidence surface), ask the amortization question ("now that we have
it, what else should it do?") under the boundary guard (passive state never becomes a message bus),
and record an OpenSSF Scorecard score in each OSS candidate's decision box. See "Adoption closure"
in `references/c4-standalone.md`.

**GATE 2:** run `machinery check <design> --gate g2`. **G2-c4** verifies, deterministically:
the contract parses (a yaml code fence starting with `contract_version` under a heading containing
"Architecture Contract"), every boundary binds to a `workspace.dsl` element, no duplicate ids, no
edge both allowed and denied, no rule referencing an undeclared boundary or external, and mitigation
coverage: every contract external plus every DSL element tagged Database, Queue, or External has a
mitigation row naming it backticked in the first column (a backticked name that matches nothing is
an error). Read the `checked:` counts; they tell you what was actually verified.
LLM-attested (you check these; the tool cannot): every Modelith action maps to an owning component;
every boundary crossing has an interface contract (shape, errors, idempotency); persistence and
placement decided per stateful component (the machine-per-placement-row check runs later, in
Gx-trace); every technology choice has its adoption closure enumerated, with closure members
carried into the mitigation table (see the reference); and the **NFR record**: the Architecture Contract conversation must record security
posture (authn/authz approach, secret handling), capacity assumptions (expected volume, latency
budget where relevant), and observability requirements (what must be logged, metered, alerted, in
particular for any FailedDirty-style residual state), even when the answer is "out of scope,
recorded as such".

### Phase 3 - State machines (the behavior)

Hand the full Modelith and C4 context and the target language(s) to the `machinery-fsm-author` role
(run it as a subagent where your runtime supports one, otherwise perform its steps inline). It
composes the domain lifecycle (derived from enums and actions) with the operational
and failure overlay (from C4 mitigations). One machine per stateful component or aggregate, never one
giant machine. See `references/xstate-format.md`.

Machine JSON conventions (enforced by the lint): every machine is either an entity lifecycle
(filename matching the entity counts, or `_lifecycle_of: "<Entity>"` when it does not) or carries
`_role: "operational"`. A state whose `always` list is fully guarded with no unguarded escape needs
an `_exhaustive: "<reason>"` justification. Resting states declare `_ignores: {event: reason}` for
every reacted-to event they do not handle. After authoring, `machinery oracle design/machines`
must be run and the generated `<M>.oracle.md` files committed; they are canonical, never hand-edited.

#### Formal annotations (rungs 3 and 4)

The machines are finite, so they are model-checkable; this phase is where the design earns its
proofs. Rung 3 (control-flow TLA+) is automatic: `machinery tla` derives it from the machine JSON
with no annotation needed, and `machinery verify-formal` runs it for you. Rung 4 takes two short
declarative annotations, authored here and kept in `design/formal/`:

- For each lifecycle machine, author `design/formal/<M>.semantics.yaml` declaring its pattern:
  `linear-lifecycle` (reopenable stage pipelines), `terminal-lifecycle` (one-way phase pipelines
  with retry overlays), or `saga` (compensating coordinators). This lets `machinery refine`
  generate and reconcile the data-refined model, the abstract contract, and the refinement proof.
- For cross-aggregate invariants, author `design/formal/<name>.composition.yaml` (a `coordinator:`
  field plus the step sequence and per-step undo) for `machinery compose`.

Examples of all three patterns live under `examples/*/design/formal/`. Then run
`machinery verify-formal design` and commit `design/formal/` (the annotations are sources; the
generated `.tla`/`.cfg` files, and the relational models `Policy.als` / `Integrity.als` /
`Isolation.als` and their oracles when the matching Phase-1 annotation exists, are committed
alongside). The reconcilers hard-fail on drift: a design
change that invalidates an annotation fails generation rather than proving a stale twin. Java is
optional here; without it, run `machinery verify-formal --gen-only design` so the suite is still
regenerated from source, and record in STATE.md that it is generated but unchecked.

**GATE 3:** run `machinery check <design> --gate g3` (add `,gx` to pull the traceability
check forward, but note: before Phase 4, Gx-trace reports every invariant not enforced by a machine
guard as an ERROR, because their enforcement point is recorded in the Phase-4 BUILD.md traceability
matrix; those errors are expected until BUILD.md exists, so run bare `--gate g3` here and the full
check at Gate 4). **G3-machine** verifies, deterministically: structural lint (only the supported
XState subset; unknown keys, parallel/history states, root-level `on`, and non-string guards are
hard errors; reachability; no dead-end non-final state; every `invoke` has `onError` and an `after`
timeout; no shadowed branch; guarded-always exhaustiveness), the committed oracle byte-identical to
a fresh in-memory generation (a stale oracle is DRIFT), any transition table in the hand matrix
reconciled against the machine structurally, row by row, in both directions, and a named-unit
contract row for every guard, action, and actor the machine fires.
LLM-attested: whether each guard's semantics actually enforce the invariant it names, and whether
every C4 dependency failure has its residual transition, reclassified by its mitigation rather than
deleted (see below). When formal annotations exist under `design/formal/`, a green
`machinery verify-formal design` is part of this gate (deterministic when Java is present; without
Java, run it with `--gen-only` and record in STATE.md that the formal suite is generated but
unchecked).

### Phase 4 - BUILD.md

Hand all three layers to the `machinery-build-writer` role (a subagent where supported, otherwise
inline). It assembles the blueprint from all three layers plus the
traceability matrix and references the generated oracles as the test spec. When the design carries
a policy annotation, BUILD.md must also require the authorization-oracle conformance test: one test
that parses `formal/Policy.oracle.md` and asserts the pure authorization function on every
reachable row (the go-crm example's `impl/internal/authz/oracle_test.go` is the reference shape).
See `references/build-md-template.md`.

**GATE 4:** run `machinery check <design>` (all gates). **Gx-trace** verifies,
deterministically: lifecycle machine TitleCase states are exactly the entity's lifecycle enum values
(the enum-typed attribute named status, stage, or state; lowerCamel states are the operational
overlay), machine `on` events are Modelith actions of that entity, every entity with a lifecycle
enum has a machine, every placement-table component has a machine or a "(no machine: <reason>)"
waiver, and every invariant id appears whole-token in a matrix or BUILD.md table cell (an invariant
compiled by the policy annotation is credited as policy-checked without a matrix row; the
relational model is an enforcement artifact).
**Gb-plan** (active once `design/BUILD.md` exists) verifies the Build plan section (any section
number, heading text "Build plan") deterministically: milestones are bold `**M<n> - <title>**`
markers with unique numbers; the first milestone's title contains "walking skeleton", or the
section carries the explicit waiver line `Walking skeleton: N/A - <reason>` (brownfield gap plans
whose skeleton already exists in production); every milestone block carries a `DoD:` line; and the
skeleton milestone's DoD cites at least one committed oracle id (test id or stable id) as a whole
token. The section itself may be waived only as `N/A - <reason>` per the template's omission rule.
In manifest mode the checks apply per shard in `design/BUILD/*.md` (a `README.md` or `index.md`
there is a shard index, not a plan shard, and is exempt); a manifest root over a
decomposition with no local shards has no plan obligations of its own (the children carry them).
Once code
exists, **G4-import** (`--impl <dir>`) parses imports (Go single and block forms via the go.mod
module name, Python, TypeScript/JavaScript, Elixir alias/import/use against boundary `modules:`,
Rust crate-relative use), enforces `exposes` and `deny` rules, and flags any undeclared
cross-boundary edge or any source file outside every boundary (use the contract `ignore:` list for
test scaffolding); test files are skipped.
**Gt-tests** (also `--impl` only) holds the suite to the oracles: every stable id in the committed
`machines/*.oracle.md` (and `formal/Policy.oracle.md` / `Isolation.oracle.md` when they exist)
appears whole-token in at least one test file under the impl, unless a test file names
`<Component>.oracle.md` as a file name (not as a substring of a longer component's name), the
parse-the-committed-table conformance idiom that covers that
machine's rows by construction (go-crm's authz `oracle_test.go` is the reference shape). It is the
hard-TDD RED-exit check (template section 11.3a) made deterministic: a missing id is a missing test.
LLM-attested: the zero-context claim itself: a coding agent with no prior context could build the
system from `BUILD.md` alone (per shard, when sharded).

## Rebuild and hybrid mode

Use this mode when an existing platform is evidence and a source of valuable assets, but not the
production foundation to improve in place. Keep two domain truths and one transition contract:

- `design/legacy/domain.modelith.yaml` describes current entities, persisted values, and behavior
  that must be preserved, adjudicated, or retired.
- `design/domain.modelith.yaml` is the normative target and runs the ordinary four phases.
- `design/migration.yaml` is the strict, checked bridge. Never merge legacy and target into one
  compromise model merely to make the migration look simpler.

During Phase 1, model both truths and author dispositions (`reuse | wrap | replace | retire`) for
every legacy entity. Declare every unmapped target entity new. Inventory implementation assets
(modules, services, schemas, data, tests) separately: domain replacement does not imply every test
or reader is worthless, and domain reuse does not make the old topology production-worthy.

### The surface ledger and the bookend sweeps

Gm proves every entity in the legacy MODEL is disposed; nothing in it proves the legacy model
captured the legacy SYSTEM. That hole is closed by `design/legacy/surface.yaml`, the capability
disposition ledger (gate: **Gs-surface**, active once the file exists): every legacy route, CLI
command, table, job, event topic, and integration is mapped to a target design element or carries
an explicit dropped/deferred disposition, and every absent class is waived with a reason. Two
named sweeps author it:

- **Opening sweep (Phase 0/1):** enumerate the legacy surface mechanically (route tables, command
  registrations, schema catalogs, cron and worker lists, outbound calls); use the codebase graph
  when the runtime has one. Most rows start `deferred`; the opening ledger is the interrogation's
  work list, not its answer.
- **Closing sweep (after Gate 4):** re-mine the legacy system against the finished design and
  settle every row to `covered`, `dropped`, or a deliberate `deferred`. Whatever the docs-first
  interrogation missed surfaces here as a row that cannot be honestly disposed. At handoff no
  deferred rationale may be an opening placeholder; the gate's `checked:` line prints the
  disposition counts so this attestation is reviewable.

The ledger is independent of `migration.yaml` by design: a clean-break run that drops the
migration machinery keeps its completeness anchor. Full guide: `references/surface-ledger.md`
(expanded repository guide: `docs/surface-ledger.md`); the worked example is
`examples/surreal-crm/design`.

For every `replace`, cover every source and target attribute with a mapping, derivation, or drop,
including transform, validation, and rollback. Cover every legacy `status`, `stage`, or `state`
enum value with a target value or explicit `drain`. During Phase 2, add a `Transition architecture`
section naming the temporary exporter, replication/dual-write, routing, observability, and failure
posture. During Phase 4, add a `Migration implementation plan` that turns every mapping and phase
into locked tests and ordered build work.

The ordered phases must name source of truth, read/write path, backfill, entry/exit criteria,
rollback, and observable signals. A shadow read requires parity semantics. A dual write requires
idempotency, conflict resolution, and reconciliation. The cutover phase must be target-only and
name an earlier rollback phase, evidence-based decision criteria, rollback window, and maximum
data loss. Temporary migration dependencies need detection, mitigation, residual, and owner.

`migration.yaml` activates **Gm-transition** automatically. It is a cross-phase gate, like Gx:
author it early, but expect the narrative-bridge findings until ARCHITECTURE.md and BUILD.md exist.
Gate 4's full `machinery check` must be green before handoff. Gm proves coverage and internal
consistency of the transition contract; it does not execute a migration or prove transformation
code. BUILD.md must require mapping-table, characterization, idempotent replay, reconciliation,
fault-injection, rollback, and cutover tests in addition to the target's ordinary hard-TDD suite.

Full installed reference: `references/rebuild-guide.md`. The expanded repository guide is
`docs/rebuild-guide.md`. The complete
worked example is `examples/go-crm/design/migration.yaml`.

## Brownfield (archaeology) mode

An existing system runs the same four phases with an inverted stance: describe, do not invent.
Two standing rules for every archaeology session:

- **Use the codebase graph when the runtime has one.** When a codebase-graph MCP server is
  available (codebase-memory-mcp or equivalent), always use it: check `index_status` and run
  `index_repository` first if the repo is not indexed, then drive the excavation through its
  tools (`get_architecture`, `search_graph`, `trace_call_path`, `get_code_snippet`) instead of
  raw file search. It maps entities, boundaries, and call chains faster and more completely
  than grep. Fall back to plain search only when no such server is present.
- **Archaeology is still an interrogation.** The code and the graph tell you what IS; only the
  user can tell you what SHOULD BE, and the gap between the two is the design's most valuable
  output. When the structure looks messy, say so concretely and ask: "this looks tangled because
  <specific observation: two modules share a table, one word has two meanings, X reaches into
  Y's internals>; what is your desired end state here?" The answers become the INTENDED
  boundaries of Phase 2, the contested-vocabulary decisions of the domain model, and the deny
  rules that give the baseline something to burn down toward. Never silently infer intent from
  structure: messy structure is precisely where inference fails.

- **Phase 0** additionally records that this is archaeology and what already exists: code, schema,
  production data, deployment.
- **Phase 1** excavates the domain model from the code, the schema, and the production data, AS IT
  IS. Start this conversation on day one, in parallel with the boundary baseline below: the
  Modelith interrogation is the instrument for understanding the mess, not a later phase's
  paperwork, and the intended boundaries are a domain claim you cannot make well before it.
  Where the code is incoherent (one word, two meanings), record the incoherence as an open
  question in the model instead of picking a winner. If the system has role- or ownership-based
  access control, author the policy annotation AS THE CODE BEHAVES today (read the authorization
  code, not the wiki), then let the meta-checks and the generated authz oracle arbitrate: a failed
  check or a failing conformance row is a discovered incoherence, adjudicated like any other
  archaeology finding (code-is-truth: fix the annotation; policy-is-truth: file the code defect).
- **Phase 2** records the existing architecture and declares the INTENDED boundaries. Then run
  `machinery baseline <design> --impl <dir>`: it scans the code exactly as G4 does, prints the
  `baseline:` rules that tolerate today's violating edges (review each, paste into the contract's
  `dependency_rules`; keep intent explicit with a `deny:` for edges that should die), suggests
  `ignore:` globs for unmodeled code, and writes `design/ratchet.json`, the snapshot that pins
  every baselined edge to its current offender files. From then on G4 fails when an amnestied
  edge grows a new offender, and burning debt down is rewarded (`machinery baseline` reruns
  tighten the snapshot). Gates run with a staged `--gate` list: g2,g4 on day one (plus gs once
  the surface ledger is authored; it gives the archaeology a coverage anchor from the start);
  add g3 as machines land; add gx only when every lifecycle enum in the model has a machine,
  because Gx has no per-entity waiver; gb joins once BUILD.md lands, gt once the RED suite exists.
- **Phase 3** machines describe current behavior. Oracle-derived tests run as characterization
  tests, and each failing row is adjudicated: code-is-truth means the model is wrong archaeology,
  fix it; model-is-truth means the code has a defect, file it and quarantine the test with its
  stable id. A test locks (the hard-TDD rule) at adjudication, not at generation.
- **Phase 4**'s zero-context claim applies to the new work carved out of the modeled slice, not to
  the legacy remainder.

Day-one state migration runs in reverse: the first version of any machine whose states are
persisted carries a mapping table from every legacy persisted value to a modeled state, plus a
rule for unmapped strays (fail loudly, never silently coerce). The full team protocol (adoption
ladder, ownership, PR discipline, CI recipes) is `docs/brownfield-team-guide.md` in the machinery
repo.

## Mitigations reclassify failures, they do not delete them

A common trap. If C4 puts Postgres on Kubernetes behind an operator, the operator does not remove the
"database unavailable" failure mode. During failover the app still sees transient unavailability for
seconds. So the FSM still needs `db_unavailable -> retry with backoff -> circuit_open -> degrade`. What
C4 changed is the *class and bound*: from "fatal, data loss" to "transient, bounded, recoverable." The
rule: a C4 mitigation sets the class and bound of each failure transition, it rarely eliminates it.
Record, per failure: detection (which `invoke` error or timeout), transition, recovery, and the
mitigation that bounds it (or the residual risk if none).

## Not everything is a state machine

Discriminate, do not force machines onto pure logic (this is a principle, not a mandate). Business
logic that is a pure transform gets a contract spec, not a machine. Reactive, lifecycle, protocol,
retry, and workflow logic gets an FSM. Even a "stateless" service still has an operational-envelope
machine (healthy / degraded / overloaded / circuit_open). Two levels: domain lifecycle and operational
envelope. Model both where they apply, neither where they do not.

## Hard-TDD handoff

The generated oracle is the test oracle. Every oracle row is a table-driven test case: given state
plus context plus event, expect a next state plus expected actions. Each row carries a sequential
`test id` and a content-derived `stable id`; **tests key on the stable id**, because row numbers
renumber when the design changes while stable ids survive unrelated insertions and change only when
that transition's stimulus changes. `BUILD.md` ends with the protocol: a test-writer agent derives
the tests from the oracle rows and the named-unit contracts; the implementer agent writes the code
and must not modify the tests. `@xstate/graph` covering-path generation remains available for
multi-step path tests on top of the per-transition rows.

The protocol is gate-anchored at both ends, and the BUILD.md you produce must say so explicitly
(the template's section 11 spells it out): RED derives tests only from a design where
`machinery check` is green, because a red design means the oracle-spec itself cannot be trusted;
RED is complete only when every oracle stable id appears whole-token in the suite (Gt-tests holds
this deterministically once `--impl` points at the suite), when
`machinery check <design> --impl <dir>` is green over the scaffolding and stubs the tests compile
against, and when the suite is red on assertions rather than its own errors; GREEN is accepted only
when the locked tests and that same check pass together. This is what makes the discipline hold on
runtimes that cannot spawn a fresh-context test-writer (Codex and other single-context agents): the
same agent runs RED then GREEN sequentially, and the deterministic gate runs separate the phases in
place of context isolation, so the suite provably covers the checked spec and the implementer has
no green path that bypasses the architecture.

## Revision mode (iteration 2 and later)

machinery designs change after code exists. The protocol:

1. Edit the design artifacts, never the generated ones. `*.oracle.md` is generated; the machine
   JSON, matrix, contract, and domain model are the sources.
2. Rerun `modelith lint`, `machinery check`, and `machinery oracle`.
3. Diff the regenerated oracles against the previous commit. Rows whose stable id disappeared are
   deleted tests; new stable ids are new tests; changed rows with the same stable id are modified
   expectations. That diff IS the affected-test list for the implementer.
4. Any machine whose states are persisted (see the placement table) MUST get a state-migration note
   in BUILD.md when a state is renamed, split, or removed: a mapping table from old persisted values
   to new states, or an explicit drain rule for in-flight instances.
5. Generated tests live apart from hand-written tests (a marker comment or a directory), so
   regeneration never clobbers hand-written ones.
6. Renames are the exception to reading the diff naively: stable ids hash the machine name and
   source state, so renaming an entity, machine, or state churns every affected id with no
   behavioral change. Handle a rename as a dedicated mapping change (rename, regenerate, record
   the old-id to new-id pairs), never as delete-all-plus-new, and never hand-edit oracles to
   avoid the churn (G3 flags that as DRIFT).

## Sharding large designs

Beyond roughly ten stateful components or two bounded contexts, do not author everything in one
pass. Run `machinery-fsm-author` once per bounded context, giving each run only:

- its context's Modelith entities and invariants,
- the full enum definitions it references,
- the C4 contract rows for its components,
- the interface contracts plus event-contract rows of its direct neighbors.

BUILD.md shards into `design/BUILD.md` (root: glossary, contract, traceability, cross-context test
spec) plus `design/BUILD/<context>.md` per context; the root document states the sharding
explicitly. Gate 4's self-containment then applies per shard.

## Recursive decomposition (contract packs)

Sharding splits the synthesis; recursion splits the DESIGN. Escalate to recursion only when the
domain model itself no longer fits one conversation: the ubiquitous language forks (the same word
means different things in different areas), the synthesis input blows the token budget, or the
composition proof stops scaling. `machinery scale <design>` measures the signals and prints a
recommendation; shard first, recurse second. Team isolation is an equally legitimate trigger with
no scale component: when independent teams (humans, agents, or both) will build subsystems without
access to the complete platform, decompose so each team's pack is its entire view. `machinery scale`
cannot see that trigger; it is an interrogation answer, not a measurement.
`examples/checkout-split/` is the worked example.

The protocol. The parent run stops at one level: domain model, C4, boundary event contracts, plus
two artifacts per subsystem it will NOT design itself:

1. `design/decomposition.yaml`: the subsystems, each with `owns:` (every Modelith entity has
   exactly one owner), `components:`, `boundaries:`, `delegated_invariants:`, a `contract_machine`,
   and (once the child exists) `child_design:`. A subsystem that genuinely has no boundary events
   declares `boundary_events: {none: "<reason>"}`; without that waiver, extracting zero events for
   a subsystem fails generation (it is almost always an event-table defect).
2. `design/contracts/<Sub>Contract.machine.json`: the abstract protocol the neighbors rely on,
   restricted to plain on-transitions and finals. This is what the child must refine and all the
   parent ever assumes.

`machinery pack generate <parent-design>` then emits `design/packs/<id>.pack/` per subsystem: the
owned domain slice, the boundary event rows, the contract machine plus its TLA+ module, the
delegated invariants, and a content hash. The pack is generated and frozen: the parent's entire
model of the child, and the child's entire view of the parent.

Extraction from the event-contract table is strict: every producer/consumer cell must resolve to
exactly one known participant (a subsystem component or an Architecture Contract boundary
element), annotations only in parentheses, fan-outs expanded to one row per producer-consumer
pair; the machine-checkable format contract is in the c4 reference. A cell that resolves to
nothing or names several components fails generation loudly, naming the row and the offending
cell text. Nothing non-empty is ever silently dropped: a lossy table once shipped near-empty
packs whose events.md still claimed boundary completeness.

Each child is a full, ordinary machinery run (all four phases, all gates) whose Phase 0 is the
pack, copied to `design/pack/`. The child may add anything internal but the pack's public shape is
frozen: owned enums exact, consumed events handled (or explicitly `_ignores`), produced events
emitted, delegated invariants traced. The child writes `design/packmap.yaml` (its exposed machine's
states onto the contract machine's, pinned to the pack hash) and runs
`machinery pack refine <child-design>`: the map is reconciled against both machines, and the
generated `formal/<M>PackRefinement.tla` is the TLC-checked proof (via `verify-formal`) that the
child refines the contract its neighbors hold it to.

### Delivery topology and contract stand-ins

At decomposition time, ask the delivery-topology question and record the answer per subsystem in
DECISIONS.md: will this team have a full multi-service environment to test against, or will it
work isolated from the rest of the platform? Each child conductor re-asks it at its own Phase 0;
the team knows its access situation better than the parent predicted it.

- **Full environment:** impose nothing. The seam is already held by the boundary event
  obligations, each side's conformance to its own oracles, and the refinement proof; integration
  tests run against the real neighbors.
- **Isolated:** the child's BUILD.md must carry a `Neighbor stand-ins and test environment`
  section (conditional in the template, like the migration plan). One contract stand-in per
  neighboring boundary, hand-built in the implementation stack, specified by the pack's boundary
  event rows plus the neighbor's contract machine, a public artifact by construction (plain
  on-transitions and finals) that the parent supplies alongside the pack when this posture is
  declared. The stand-in is held to a conformance suite generated by `machinery oracle` over that
  contract machine, keyed on stable ids; when the parent regenerates packs, the oracle diff is the
  stand-in's affected-obligation list. machinery generates the spec, never the stand-in: the
  stand-in is ordinary implementation code under the same hard-TDD discipline as everything else.

Call it a contract stand-in, never a mock, and hold the distinction: it is not a stub of internals
invented by a test author, it is an executable of the signed, frozen contract, and the refinement
proof is what licenses swapping the real neighbor in at assembly. That is why it does not violate
the no-mocks rule for integration tests, and why the pattern must not dilute into ad-hoc stubs: a
stand-in that drifts from the contract oracle is a defect, not a convenience. Gate 4 for an
isolated child attests, alongside the zero-context claim: the section exists, every neighboring
boundary has a stand-in held to its oracle, and the environment recipe is self-contained (the team
can run the entire suite with nothing beyond the pack, the stand-ins, and the recipe). What
stand-ins cannot prove stays named: the parent's residuals (end-to-end latency, cross-contract
liveness, unmodeled channels) belong to the parent's cross-context assembly suite.

**GATE 5:** `machinery check` runs G5-pack automatically on decomposed designs (a machine-less
parent skips only the machine-dependent gates G3/Gx/G4/Gt and runs g2,g5 plus every
artifact-activated gate whose source exists, gm/gs/gp/gi/gn/gb; machine-less means no
`machines/*.machine.json`, an empty directory included; children run everything). Parent side:
committed packs byte-match a fresh generation,
every pinned child was built against the CURRENT pack, and the `checked:` line prints per-pack
boundary-event counts so an unexpected zero is visible in every run. Because G5 regenerates packs
in memory, a lossy event-contract table fails the gate itself, not only `pack generate`. Child
side: pack hash verified, packmap reconciled, refinement artifacts fresh, owned shape unchanged,
delegated invariants traced, boundary events covered in both directions. A boundary change is
therefore a PARENT edit: regenerate packs, re-copy, and the pack diff is the child's
affected-obligation list, exactly as an oracle diff is the affected-test list.

Residuals to name in the parent's BUILD.md (the proofs do not cover them): properties not
expressible in the contract-machine vocabulary (end-to-end latency, deadlock through resources
outside the modeled channels), and liveness across mutually dependent sibling contracts (keep
cross-contract reliance to safety plus each contract's own termination).

## Session ledger

For multi-session interrogations, keep `design/STATE.md` with one line per phase: status
(pending / in-progress / gate-passed), date, open questions, and, once the gate passes, the
self-review line below. A resuming conductor reads it and knows where the interrogation stopped.

### Phase-exit self-review

Before recording a phase as gate-passed in STATE.md, run one adversarial pass over that phase's
artifact in a challenger's stance. Five questions: (1) reality, every named term, path, technology,
and identifier exists or is decided in DECISIONS.md; (2) depth, the invariants, boundaries, and
machines encode this client's actual rules, not a generic system's (a shallow model gates clean);
(3) scope, nothing bundled in that the client did not ask for, nothing the client said dropped;
(4) coverage, edge and failure paths are enumerated, not just happy paths; (5) consistency, no
contradiction with earlier phases or DECISIONS.md. Record one verdict per question on one line
under the phase entry:

`self-review: reality=clean depth=fixed scope=clean coverage=accepted(<short reason>) consistency=clean`

where `fixed` means the pass found and fixed something and `accepted(<reason>)` means a finding
was consciously waived. A phase entry without a self-review line is not complete.

## Working as a team

- A design change and its regenerated artifacts (oracles, formal, packs) land atomically in one
  change; a change that edits a machine without its regenerated oracle is malformed.
- Never hand-resolve a merge conflict in a generated file: take either side, regenerate, and
  re-run the gates.
- `STATE.md` is single-writer: the active conductor.
- `design/DECISIONS.md` is required once interrogation starts: one dated line per binding
  decision (mode, transition posture, tech choices, invariant adjudications, contested
  vocabulary), recorded when the answer lands, never reconstructed later.
- Consuming repos run `machinery check` (with their staged `--gate` list) in their own CI on every
  change and again after merge. The full recipe is `docs/brownfield-team-guide.md` in the
  machinery repo.

## Operating discipline

- Batch questions. When the choices are discrete, ask them as a single multiple-choice question.
  Converge, do not loop.
- Mine the free text, not just the selected option. When a structured answer carries user-typed
  notes, that text is a requirements source: decompose it, echo each extracted requirement back,
  and record it before the next round. The richest requirements often arrive as annotations on
  multiple-choice answers.
- Echo load-bearing interpretations before building on them. Before asking questions that depend
  on your reading of a term or an earlier answer, restate that reading in one line and let the
  user correct it. One misread term early poisons every question after it.
- Signal progress. In a multi-round interrogation, state the shape each round: which phase this
  is and roughly how many rounds remain. The estimate is a shape, not a commitment; surfaced
  fuzziness legitimately extends it.
- Record decisions at answer cadence. When an answer round lands, record its decisions in
  `design/DECISIONS.md` (verbatim where the user's wording is load-bearing) before asking the
  next round. Anything that binds a later phase qualifies: mode, transition posture, tech
  choices, invariant adjudications, contested vocabulary.
- Contested externals get evidence, not preference. When a discrete choice hinges on facts
  outside the conversation (provider or cloud selection, a queue or framework choice, pricing,
  limits, compliance posture) and the runtime has web access and subagents, spawn research agents
  against primary sources and return a decision box per option: the evidence with sources, a
  recommendation, and the flip conditions under which it reverses. Record the box in
  DECISIONS.md, flip conditions included; they are what makes a later revision defensible. For
  OSS adoption candidates, include a dated OpenSSF Scorecard score in the box (the adoption-closure
  checklist in `references/c4-standalone.md` gives both the zero-install API lane and the CLI
  lane). When
  the runtime lacks those capabilities, record the contest and its flip conditions as an open
  decision instead of silently picking. Reserve this for evidence-arbitrable questions, never
  preferences, and bound the fan-out to the decision at hand.
- Each phase has an exit gate. Stop interrogating the moment it passes.
- Fuzziness is a signal, not an obstacle. When the user cannot give a crisp definition, that gap is
  the point of the exercise. Surface it.
- You are the conductor. Reuse `domain-model-author` for Phase 1; run `machinery-fsm-author` and
  `machinery-build-writer` for Phases 3 and 4 (as subagents where supported, otherwise inline). Do
  the C4 work inline using the reference.
- If your runtime intercepts a file read with a code-discovery gate, retry it once; the design docs
  are plain text, not code discovery. Pass the same note to any subagent you delegate to.
- House style: generated artifacts carry no em dashes and no emojis. `modelith render` output contains
  em dashes, so post-process it (Phase 1). When you delegate to `machinery-fsm-author` and
  `machinery-build-writer`, pass the house-style constraint in their prompts.

## References

- `references/rebuild-guide.md` - rebuild/hybrid mode, the strict `migration.yaml` contract, phase and
  cutover obligations, and the migration regression checklist.
- `references/surface-ledger.md` - the legacy surface ledger: the opening/closing sweep protocol,
  the `legacy/surface.yaml` schema, and the Gs-surface gate.
- `references/xstate-format.md` - the enforced XState v5 JSON-serializable subset, the machine
  annotations (`_role`, `_lifecycle_of`, `_exhaustive`, `_ignores`), and the failure-mode and
  choreography idioms.
- `references/c4-standalone.md` - Structurizr DSL authoring guide (strict syntax rules, dark-mode
  palette, validation via `structurizr-cli export`), the Architecture Contract v2, the
  dependency-mitigation, persistence-placement, and event-contract table formats, and the NFR record.
- `references/build-md-template.md` - the full `BUILD.md` skeleton (full and manifest modes).
- `machinery check` - the deterministic gate suite (Gm-transition, Gs-surface, Gp/Gi/Gn
  relational gates, G2-c4, G3-machine, Gx-trace, Gb-plan, G4-import, Gt-tests,
  G5-pack for decomposed designs).
  Single Go binary. Run it at each gate with `--gate` so correctness does not
  rely on the model getting every cross-reference right. See `tools/README.md`.
- `machinery baseline <design> --impl <dir>` - the brownfield Stage-1 generator: proposes
  `baseline:` rules for today's violating edges, suggests `ignore:` globs, and writes
  `design/ratchet.json` (generated; never hand-edit), the snapshot G4 ratchets baselined
  edges against.
- `machinery oracle` - generates the canonical `<M>.oracle.md` transition oracles from the
  machine JSON. Run after every machine edit and commit the output; G3 diffs it.
- `machinery update [--version <tag>]` - checksum-verifies and force-refreshes the CLI plus every
  recorded direct agent home and native host adapter from one release. Host plugin caches remain
  host-owned and are refreshed through the Claude Code/Codex CLIs when detected.
