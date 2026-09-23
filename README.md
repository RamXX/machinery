# machinery

[![CI](https://github.com/RamXX/machinery/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/RamXX/machinery/actions/workflows/ci.yml)
[![Formal Verification](https://github.com/RamXX/machinery/actions/workflows/formal.yml/badge.svg?branch=main)](https://github.com/RamXX/machinery/actions/workflows/formal.yml)
[![Security](https://github.com/RamXX/machinery/actions/workflows/security.yml/badge.svg?branch=main)](https://github.com/RamXX/machinery/actions/workflows/security.yml)
[![Nightly](https://github.com/RamXX/machinery/actions/workflows/nightly.yml/badge.svg?branch=main)](https://github.com/RamXX/machinery/actions/workflows/nightly.yml)
[![CI: Dagger](https://img.shields.io/badge/CI-Dagger%20v0.21.9-131226)](.dagger/main.go)
[![Go Reference](https://pkg.go.dev/badge/github.com/RamXX/machinery.svg)](https://pkg.go.dev/github.com/RamXX/machinery)
[![Go Report Card](https://goreportcard.com/badge/github.com/RamXX/machinery)](https://goreportcard.com/report/github.com/RamXX/machinery)

**Design software once, as a state machine, and let everything else be derived and proven from it:
the tests, the architecture contracts, the build instructions, and machine-checked proofs of
correctness.** machinery is a design methodology and toolchain that turns a fuzzy idea into a
build-ready, formally verified blueprint that a coding agent with zero prior context can implement
under hard TDD.

## Why it exists

AI coding agents make it cheap to write software fast. They do not make it safe to write large
software. On anything past a toy, correctness degrades quietly: the design and the code drift apart,
a cross-cutting invariant gets violated three files away from where it was written, a failure mode
nobody enumerated takes down production at 3am. The usual answer is "review it carefully," which is
another way of saying "trust the model." Trust does not scale.

machinery takes the opposite stance. It treats correctness as something you **construct and check**,
not something you hope for. The design is a single source of truth. Everything downstream is either
generated from it (so it cannot drift) or checked against it by a deterministic tool or an exhaustive
proof (so a mistake is caught, not shipped). The model does the creative work; the machine holds the
line.

## The thesis

Most software is a state machine. Make that explicit and the rest follows. machinery separates a
design into three layers that compose rather than merely coexist:

- **The what**, a domain model ([Modelith](https://modelith.sh/)): the entities, the relationships,
  and the invariants that must always hold. Linted.
- **The how**, an architecture ([C4](https://c4model.com/)): the components, the deployment, and what
  every dependency does when it fails. Contract-checked.
- **The behavior**, state machines in the [XState](https://github.com/statelyai/xstate) v5 JSON
  format (the notation, not the library): every state, transition, guard, timeout, and failure mode,
  conditioned on the architecture the previous layer fixed. Model-checked.

The state machines come last because they need the other two as inputs, and half of each machine is
*derived* from the domain model rather than invented. A narrow design ends in one self-contained
`BUILD.md`. A large design ends in a root milestone/demo manifest. Its declared linkage is either
pairwise (one bounded, self-contained execution packet per milestone) or matrix (reciprocal
many-to-many links between milestones and reusable domain shards). The machines remain both the
test oracle and the formal spec.

For a rebuild or hybrid migration, machinery keeps a second, legacy domain model instead of blending
current and intended behavior into one document. A checked `migration.yaml` disposes every legacy
entity and salvageable asset, maps replaced data and lifecycle states, and defines the coexistence,
rollback, and cutover contract. The target still goes through the complete pipeline below.

To be clear about that XState reference: machinery uses the XState config **format** as a linted
notation and does not run the XState library. The lint is our own (a deliberately narrowed subset,
plus annotations like `_role` and `_exhaustive` that are not XState), and so are the oracle generator
and the model checking. The de-annotated config loads into [Stately](https://stately.ai/) and
`@xstate/graph` for optional visualization and covering-path generation, and a TypeScript build may
adopt XState directly, while Go, Rust, Python, and Elixir targets hand-roll the state field. The
guarantees come from machinery's tooling and [TLC](https://github.com/tlaplus/tlaplus), the model
checker for [TLA+](https://lamport.azurewebsites.net/tla/tla.html) (Leslie Lamport's formal
specification language for state-transition systems), never from XState.

### Why these notations, and not the obvious alternatives

Each layer's notation was chosen to be text-based, generable, and machine-checkable, not for
familiarity:

- **Modelith, not UML class diagrams or prose.** It is purpose-built to carry entity lifecycle and
  invariants as first-class, lintable data, which is what lets half of each state machine be
  *derived* instead of hand-written; a diagram or a prose spec carries none of that as checkable
  structure.
- **C4, not UML component diagrams or arc42.** It has a text form (Structurizr DSL) a gate can parse
  and bind, and it centers components and their dependencies, exactly where the per-dependency
  failure posture lives; UML is diagram-first and heavier, and arc42 is a documentation template
  rather than a checkable model.
- **XState v5 JSON, not SCXML or a hand-rolled DSL.** JSON is trivially generable and byte-diffable,
  the v5 shape already has the constructs we need (hierarchy, guards, `invoke`, delays), and the
  ecosystem (Stately, `@xstate/graph`) gives visualization and covering-path generation for free;
  SCXML is XML with weaker tooling, and a bespoke DSL means owning a parser and a visualizer forever.
- **TLA+ and TLC for behavior, not property-based testing.** TLC checks a finite instance
  *exhaustively* for the temporal properties that matter here (termination, deadlock-freedom,
  safety, liveness) and returns a concrete counterexample; property-based tests sample the state
  space rather than exhaust it.
- **Alloy for the static relational layers, where TLC never looks.** Some invariants are relations
  over configurations, not steps over time: no state machine enforces them, so TLC cannot check them,
  and a structural linter sees only well-formed prose. Bounded relational search is exactly Alloy's
  strength. machinery has three opt-in relational algebras, each generated from the domain model plus
  a short annotation (`machinery alloy`), each solver-checked by `verify-formal`: **policy** (access
  control: roles, ownership, team scoping), **integrity** (structure: uniqueness, singletons,
  cardinality), and **isolation** (multi-tenant reference isolation). Nobody hand-writes Alloy, the
  same rule as the TLA+.

## Agentic systems: the machine is the envelope, not the agent

"Most software is a state machine" is a claim about control flow, not about cognition, and agentic
programs are where the difference bites. An agent is a non-deterministic policy (an LLM choosing which
tool to call) running inside a deterministic envelope: the perceive-act-observe loop, tool execution,
budgets, guardrails, approval gates, side-effect compensation, sub-agent orchestration. The envelope
is the state machine; the policy is not, and machinery does not pretend otherwise.

What machinery models is the containment. Every tool is an `invoke` with an enumerated failure set, a
timeout, and an idempotency key, so the action space is bounded even when the choice within it is not;
the loop is a machine whose termination and budget are model-checked, so it provably stops; every
irreversible action is a saga with compensation and an explicit stalled-dirty residual; every
guardrail is a guard tied to an invariant; sub-agents coordinate through the event-contract table. The
LLM decision itself is a contracted, non-deterministic oracle machinery fences but never tries to
prove. You get liveness (it stops, within budget, and cleans up) without usefulness (that it chose
well), so keep the action space enumerated and the high-stakes guardrails deterministic: that is the
region machinery can prove, and whatever you leave to the model's discretion is region it cannot.
Fittingly, machinery is itself built this way, a gated pipeline around a non-deterministic conductor.

## The pipeline

```
Phase 0  Frame        what, who, purpose, target language
Rebuild  Transition   legacy model + target model + migration.yaml + surface ledger (opt-in)
         tool: Gm-transition (complete disposition, mappings, phases, cutover, risks)
               Gs-surface (every legacy route/command/table/job disposed against the target)
         attested: what is worth saving; transformation and rollback semantics; sweep completeness
Phase 1  Modelith     domain model
         tool: modelith lint clean; pinned render engine reproduces every committed *.modelith.md
               after the mechanical em-dash normalization
         attested: lifecycle enums, action pre/post, invariant owners, scenario coverage
Phase 1.5 Relational  static relational models (opt-in, per invariant shape):
           policy    access control      -> Gp-policy    (Policy.als + Policy.oracle.md)
           integrity structure/keys      -> Gi-integrity (Integrity.als)
           isolation multi-tenant refs   -> Gn-isolation (Isolation.als + Isolation.oracle.md)
         tool: each gate binds + covers; committed models fresh; solver via verify-formal
         attested: the annotations state what the prose invariants mean
Opt-in   External     bring-your-own deterministic checker (SAST, AST, Datalog, graph reasoner):
           checkers   -> Gk-checkers (input_hash binds, coverage holds; engine via verify-checkers)
         tool: projection is fresh; committed evidence binds to it; verdict is pass; coverage complete
         attested: the manifest claims the elements the checker is responsible for
Phase 2  C4           architecture + contract (+ surfaces.yaml, the target surface ledger)
         tool: G2-c4 (contract parses and binds; allow graph acyclic; mitigation coverage;
               every relationship the DSL DRAWS judged by the same allow/deny/baseline rules
               G4 judges an import by; an interface contract per allowed edge, and no row for
               an edge nobody allows; NFR record present with all three topics; every event
               table names its source, answers every column, and names participants the model
               declares; opt-in tables held closed: action ownership, adoption closure)
               Gu-surfaces (every act a person performs has a named surface or a deferral;
               sources: names where the act list was enumerated from)
               engine: verify-c4 (workspace.dsl compiles under structurizr-cli)
         attested: each contract's shape is the right one; the NFR record's content; the
                   dependency declaration and the persona walk are complete
Phase 3  XState       state machines
         tool: G3-machine (lint; every invoke has onError + timeout; every fully guarded handler
               declares its guard-false disposition; delays consumed both ways; oracle fresh;
               matrix reconciled)
         attested: each guard enforces the invariant it names; residual failure transitions kept
                   (bindable: the mitigation table's opt-in "handled by" column resolves per row)
Phase 4  BUILD.md     the blueprint
         tool: Gx-trace + Gr-reads + Gb-plan + Ge-embed (+ G4-import and Gt-tests once code exists;
               Gw-packet once a slice map projects bounded per-slice executor packets)
               Gx also reconciles the event-contract rows against the machines: a consumed
               event some machine handles or ignores, a produced event some action or matrix
               row fires, or a `(no machine: <reason>)` waiver in the cell that owes it
         attested: a zero-context coding agent could build it
Build    Acceptance   discharging a milestone (once the build starts)
         tool: Ga-accept (committed evidence per closed milestone, bound to the
               reviewed commit and to the oracle ids its DoD cites)
         attested: whether the reviewer judged well
Brownfld Adjudication characterization verdicts (design/adjudications/, activates on presence)
         tool: Gj-adjudication (every verdict binds to a committed oracle row; a
               model-is-truth verdict names its filed defect)
         attested: whether each verdict is right
Any      Attestation  the attested halves themselves (design/attestations.yaml, on presence)
         tool: Gv-attest (each row names a claim from a closed vocabulary, an attestor,
               and the covered artifacts with their content hashes; an edited artifact
               makes the judgment STALE)
         attested: whether each judgment is true, which is the point of the split
Always   Ledgers      STATE.md / DECISIONS.md formats + house style
         tool: Gl-ledger (self-review grammar; dated entries; em-dash/emoji scan, warn
               tier, except em dashes in a *.modelith.md render, which error)
         attested: the ledgers' content; which phases owe a self-review line
```

An interrogation, not a form. The conductor pushes on naming and on "what must always be true," and
does not advance a phase until its gate passes. Every gate splits into a deterministic half the
tool verifies and an attested half the conductor checks by judgment; the skill spells out that
split per gate, and the table above keeps the two apart.

## The policy layer: access control as a checked model

Most business systems carry a handful of cross-cutting access rules: who may read, write, or
reassign what, scoped by role, ownership, and team. These rules are the classic blind spot: no
state machine enforces them (so model checking never sees them), a linter only checks that the
prose is well-formed, and every agent that touches the codebase interprets the English slightly
differently. The go-crm example shipped with an RBAC rule set that had survived prose review,
every gate, eight formal proofs, and a full TDD build, and it still contained two real defects
(a Manager without a team provably could not write its own records, and a Manager could hand a
record to someone outside its own authority in one legal step).

The policy layer closes that gap without asking anyone to learn a formal language. You (or the
conductor, during the Phase 1 interrogation) write one short annotation,
`design/formal/policy.relational.yaml`: who acts (the role enum), what they act on (the owned
entities), and per-verb scopes from a four-word algebra (`all | own | team | none`). Everything
else is generated and machine-held:

- **`machinery alloy <design>`** compiles the annotation plus the domain model into two artifacts.
  `Policy.als` is a relational model with a standard meta-check suite; the
  [Alloy analyzer](https://alloytools.org/) searches every configuration within a bound and
  returns a concrete counterexample when the rules hide a hole ("here is the teamless Manager and
  the record it cannot touch"). `Policy.oracle.md` is the same policy enumerated as a decision
  table: every role, verb, and ownership case with its expected verdict, under stable ids.
- **Your implementation consumes the table.** One conformance test parses the oracle and asserts
  the authorization function on every reachable row. From then on, policy-versus-code drift in
  either direction is a failing test that names the exact case.
- **The gates hold all of it.** Gp-policy fails the build when the annotation stops matching the
  domain model, when a cross-cutting invariant is neither compiled nor explicitly waived, or when
  a committed artifact is stale; the plugin hooks refuse hand-edits to the generated files.

The layer is opt-in and costs nothing where it does not apply: a design without access-control
invariants (fulfillment, portfolio-engine) never sees it. Where it does apply, the annotation is
also an interrogation instrument: it forces the questions prose lets you skip, like "must a
Manager have a team?" and "where may a reassigned record go?", which is exactly where the go-crm
defects lived. The full guide, including the annotation reference, the oracle test pattern, and
the brownfield workflow, is [docs/policy-layer.md](docs/policy-layer.md).

## What makes it production-grade: the correctness ladder

A domain-model linter is table stakes. The differentiator is that machinery pushes deterministic and
formal correctness into every layer, strongest first:

1. **Generate, do not co-author.** Anything derivable is generated from the design sources: the
   transition oracle (with content-derived stable ids that survive design revisions), the TLA+
   specs, and, on designs with a policy annotation, the Alloy model plus the authorization oracle
   (the policy enumerated as a decision table the implementation tests consume).
   G3 then byte-diffs every committed oracle against a fresh generation on every check. The pinned
   Modelith engine regenerates every committed domain render in required CI (including any legacy
   render), followed by the mechanical em-dash normalization and a byte-diff. The
   formal specs are regenerated from source by `verify-formal` (with the required formal workflow
   and the nightly regen-clean-tree job both asserting the committed copies match), so staleness is
   caught as drift, never assumed away. The sole non-generated exception is a strict manual TLA
   pair: the module's first line is exactly `\* machinery:manual`, a same-basename `.cfg` is
   mandatory, and any unmarked orphan pair or half is an error. Manual pairs are TLC-checked and
   counted as declared and checked, but are never regenerated.
2. **Deterministic symbolic gates that cannot pass on absence.** `machinery check` verifies, with no
   LLM in the loop: machines are well-formed (reachability, unambiguous targets, no dead ends, every
   side effect has an error path and a timeout, every resting state handles or explicitly ignores
   every event), the architecture contract binds to the C4 model and every dependency has a failure
   posture, the layers trace to each other by construction (states to enum values, events to actions,
   invariants to enforcement rows), and the code respects the boundaries (Go, Python,
   TypeScript/JavaScript, Elixir, Rust). Every gate prints what it actually checked; a gate that
   finds nothing to check fails instead of passing.
3. **Model checking.** Each machine is finite, so TLC checks it exhaustively: retry loops bounded
   (each loop with its own counter), every operation terminates, nothing gets stuck half-done, no
   deadlock. Every generated spec states its assumptions in its header (guards erased soundly for
   safety, liveness conditional on guard exhaustiveness that the linter discharges, single instance,
   no data at this rung), so a green check reads as exactly what it is. The same rung extends
   sideways into statics: when a design carries access-control invariants, `machinery alloy`
   compiles them (from a short policy annotation) into a bounded relational model that Alloy
   searches exhaustively within scope, with generated meta-checks for the classes of defect a
   policy hides best: a write-capable role whose own records are out of scope, a reassignment that
   escapes the actor's own authority, a granted verb that is exercisable nowhere.
4. **Refinement and assume-guarantee.** The data-and-composition annotations are reconciled against
   the machines before anything is emitted, so a drifted annotation fails generation instead of
   proving a stale twin. Each subsystem is proven to refine the small contract its neighbors rely
   on; the composition instances that same contract module and TLC additionally checks the
   composition satisfies it. Parts are verified against contracts, never against the flattened
   system, which is the only way this scales to real size.

Rungs 1 through 3 are generated from the design automatically. Rungs 2 and 4 are generated from
short declarative annotations that the generators verify against the machines. And the toolchain
that holds all of this together is itself held: a Go test suite encodes every vacuity and drift
attack from an adversarial design review as a permanent regression, and CI runs the tests, the
gates, the proofs, and the example build on every push.

Generating every artifact (the oracle, the TLA+ specs, the reconciled models) is the `machinery`
Go binary itself and needs no [Java](https://adoptium.net/). Only the *checking* of the proofs (rungs 3 and 4, and rung 2's refinement) runs under
TLC, which is a Java program, so **Java is required only for `machinery verify-formal`**. That step is
optional but recommended: the deterministic gates (rung 2's generation and every symbolic check)
already catch malformed machines, drift, and boundary erosion, but they cannot tell you that a saga
can strand money, that a retry can loop forever, or that a subsystem violates the contract its
neighbors assume. The model checking is what proves those, exhaustively, with a concrete
counterexample when it fails (it caught two real defects in the examples below, and the relational
policy layer caught two more in go-crm's RBAC). A Java-free setup is
a complete, gated design; adding Java upgrades "structurally consistent" to "machine-checked."

## Proof it works: the go-crm example

`examples/go-crm` is a Go CRM with a native CLI over an embedded
[LadybugDB](https://github.com/LadybugDB/go-ladybug) graph and role- and
ownership-based access control, taken end to end:

- **Designed** through all four phases as the production target of a checked rebuild. The legacy
  prototype and `migration.yaml` account for 4 legacy entities, 16 field mappings, 9 lifecycle
  mappings, 4 coexistence/cutover phases, and 3 transition risks. The target domain model lints
  clean (9 entities, 27 invariants); C4 model
  with the dependency posture that an embedded store forces (corruption is fatal-until-restore, not a
  transient); five state machines; a 1138-line `BUILD.md`.
- **Built by a zero-context coding agent** under hard TDD: a test-writer wrote the suite from the
  blueprint, the tests were locked, and an implementer made them pass without touching a test. Result:
  362 tests green, 83% coverage, architecture boundaries upheld in the source. The one impossible test
  was escalated as a design defect and fixed in the design, not the code.
- **Gated** by `machinery check`: it certified the design consistent and caught a real contract defect
  the prose review had missed. The hardened gates verify it non-vacuously: 194 transitions reconciled
  row by row against the matrices, every guard, action, and actor covered by a named-unit contract,
  every import edge checked against the architecture contract.
- **Proven** by `make verify-formal`: eight TLC proofs plus 24 Alloy commands across policy,
  integrity, and isolation (32 checks total), all green, regenerated from source every run. The
  policy suite earned its place the hard way: run
  against the RBAC invariants as first written, it found a Manager without a team could not write
  even the records it owned (the write-scope rule granted by team membership only, and nothing
  required a Manager to have a team), and a Manager could reassign a record to a user outside its
  team, beyond its own authority. Both were invariant defects invisible to the linter and to TLC;
  both are fixed in the domain model, and the checks that caught them now run on every design with
  a policy annotation. The same annotation also compiles to a 70-row authorization oracle
  (`Policy.oracle.md`) that the implementation's test suite asserts case by case, so the code is
  held to the policy the same way the machines are held to their oracles.

Every number above is real output in this repository, not an illustration.

And it holds on a second, deliberately different design. `examples/fulfillment` is a distributed
order-fulfillment platform: microservices, a saga orchestrator, compensation, and a transactional
outbox, with six state machines (the saga plus the Order, Payment, Reservation, Shipment, and
OutboxMessage lifecycles). The same generators produced its formal models, and TLC checked them: the
saga always terminates, and its data-refined model shows that money and stock are never silently
lost, with compensation modeled per obligation so partial compensation is a real, checked state.
Its integrity layer also carries the inverse side of the `Order.payment` 1:1 relationship as an
axiom of the model (a `Cardinality_*` fact), so two
orders cannot share one payment - a constraint that forward field multiplicity alone does not imply.
Building that proof caught a real bug in the saga as first drawn, where a single failed refund could
leave a customer charged with nothing returned. TLC produced the exact counterexample and the fix is
checked. The hardened cross-layer gate then caught a second real defect: the domain model's saga
enum had drifted from the machine and was missing the FailedDirty residual entirely. Across the
example designs, `make verify-formal` checks every proof, all green, regenerated from source on each
run (and only this step needs Java).

## Brownfield systems

The pipeline reads as greenfield, but the toolchain does not care. On an existing system you run
the phases as archaeology instead of invention. Before architecture, classify every coherent legacy
corpus area as behavior to preserve/port, capability to rearchitect/adapt, learning-only evidence,
historical-only material, or unresolved intent. Missing classification never means "must port," and
an unresolved product decision blocks the architecture handoff. Then write the Modelith model, the
contract, and the machines for the target the evidence supports, and let the gates arbitrate what
they can see. Be
precise about what that is: the only code-facing gate is G4-import, and it reads import statements
only, so what you get from day one is the import-boundary drift map, not a behavioral comparison.
No gate executes or reads code behavior. Behavioral code-vs-model drift surfaces through
characterization tests derived from the generated oracles; the oracle stable ids give every
transition a durable test key, and G3's staleness DRIFT keeps the oracles regenerated as the model
is corrected, which is the loop that maps the drift. The adjudication rule: oracle-derived tests
start descriptive and become locked (hard TDD) once a human adjudicates each failing row as
code-is-truth (fix the model) or model-is-truth (file the code defect). From there, revision mode
is the operating loop: design changes as diffs, stable-id oracle diffs as the affected-test list,
and state-migration notes for persisted machines, which a brownfield system has on day one. The
first modeling pass is a real investment, roughly proportional to how undocumented the system is.
Day one of adoption is a command, not a transcription exercise: `machinery baseline <design>
--impl .` scans the code exactly as G4 does, proposes the `baseline:` rules that tolerate today's
violating edges (structurally distinct from intended `allow:` rules, and compatible with a `deny:`
that keeps the intent written), and records `design/ratchet.json`, the snapshot that makes any NEW
offender file on an amnestied edge a blocking finding, so the baseline can only shrink silently,
never grow. For the staged team adoption protocol (the baseline and ratchet flow, incremental
`--gate` lists, merge and CI recipes), see the
[brownfield team guide](docs/brownfield-team-guide.md).

## Rebuild and hybrid systems

When the current platform works but should not be the production foundation, do not model a
fictional halfway system. Keep `design/legacy/domain.modelith.yaml` as the current truth,
`design/domain.modelith.yaml` as the target truth, and `design/migration.yaml` as the checked bridge.
The optional file activates **Gm-transition**, which requires a disposition for every legacy entity,
a mapped-or-new decision for every target entity, a salvage decision for implementation assets,
complete field and lifecycle coverage for replacements, ordered source-of-truth phases, an explicit
target-only cutover and rollback window, and owned failure postures for temporary migration
dependencies. Shadow reads must define parity; dual writes must define idempotency, conflict
resolution, and reconciliation.

This keeps "save what can be saved" precise: characterized behavior, export readers, data, tests,
or modules can be reused or wrapped deliberately, while obsolete schemas and topology are replaced
without silently constraining the target. Gm checks contract completeness and narrative integration;
the migration implementation and transformations remain the responsibility of the locked tests in
BUILD.md. The [rebuild and hybrid guide](docs/rebuild-guide.md) contains the full contract reference,
workflow, test checklist, and the worked Go CRM rebuild.

Gm's coverage universe is the legacy model the run declared, so it cannot see a subsystem the
excavation missed. `design/legacy/surface.yaml`, the capability disposition ledger, closes that
hole: it inventories the legacy system's mechanically enumerable surface (routes, CLI commands,
tables, jobs, events, integrations) and maps every item to a target design element or an explicit
dropped/deferred disposition. The file activates **Gs-surface**, and it deliberately does not
depend on `migration.yaml`: a clean-break rebuild that drops the migration machinery keeps its
completeness anchor. The [surface ledger guide](docs/surface-ledger.md) has the schema, the
opening/closing sweep protocol, and the worked SurrealDB CRM rebuild.

The forward twin of that question is asked by `design/surfaces.yaml`, the target surface ledger.
Every gate before it looks at the design's internals; none of them asks how a person actually
reaches an act. A model can declare that an administrator suspends a tenant, the architecture can
place it, a machine can encode it, and the whole design can pass green while nothing names the
screen, admin command, API route, or config release the administrator uses. The ledger maps every
action whose actor is a person to a named surface, or defers it with a reason, and **Gu-surfaces**
holds that set closed against the domain model. The [target surface guide](docs/target-surfaces.md)
has the schema, the persona-walk sweep, and the gate's checks.

## Which model to use where

The gates check structure, not substance: a shallow domain model with the wrong invariants gates
completely clean. Extracting the right invariants, pushing on fuzzy definitions, and deciding
failure postures (Phases 0 through 2) is pure judgment with no machine backstop, so use the
strongest reasoning model you have there. Phases 3 and 4 are a different regime: half of each
machine is derived mechanically, and lint, oracle diff, reconciliation, and TLC catch most of what
a weaker model would fumble, so a mid-tier model is much safer for the synthesis. The deterministic
layer narrows the failure mode rather than removing it: with a weak interrogator you get a
structurally consistent, formally verified model of the wrong system.

Execution is deliberately a different workload from design. For a large project, the root
`BUILD.md` is only the ordered milestone and demo authority; every milestone links to one packet
under `BUILD/` containing its domain, architecture, behavior, oracle, TDD, implementation, risk,
recovery, and acceptance context. Each packet is capped at 64 KiB and may not depend on another
packet for missing context. This keeps implementation tractable for smaller or local models without
weakening the tests, proofs, or acceptance boundary.

### When not to use machinery

CRUD screens, UI-heavy surfaces, and pure transforms get a contract spec and ordinary tests, not
the four-phase pipeline; forcing a machine onto them is ceremony, and it reads as ceremony.
machinery pays off where state actually bites: lifecycle, saga, protocol, retry, and workflow
logic, and the deterministic envelopes around agentic systems. Model the stateful core, not the
whole repo.

## What machinery does not verify

The gates and proofs are exactly as strong as stated above, and no stronger. Not covered by any
deterministic check or proof, by construction: whether the interrogation extracted the RIGHT
invariants (a shallow domain model gates clean); guard and action semantics in code (the named-unit
contracts carry them into unit, integration, and property tests, but a wrong implementation of a
correctly-named guard is caught by tests, not proofs); races between concurrent machine instances
and message loss, duplication, or reordering between machines (the models are single-instance; the
event-contract table and the idempotency contracts govern those seams, and the tests exercise them);
whether migration transformations preserve real production data (Gm proves decision coverage, not
the implementation or a database run; mapping, reconciliation, and rollback tests hold that);
coupling through shared database tables or bus topics (invisible to import analysis); and security,
capacity, and observability beyond what the Phase 2 NFR record captures. The methodology's stance is
to name every one of these residuals in the design artifacts rather than let a green check imply
they are covered.

Inside what is checked, four claims stay separate and must not be conflated: artifact consistency
(a committed artifact is byte-fresh against the current design), proof execution (a solver actually
ran and reproduced what the committed evidence says, in `verify-formal`/`verify-checkers`), test
execution (your suite actually ran and passed: the `Gt` oracle-coverage gate credits static
discovery of test references only, and tests were not executed by any machinery gate; its own
label reads "static discovery; tests not executed"), and
current review (`Gv` attestations carry an explicit kind, `plan`, `current`, or `historical`, so
a design-time review is never silently read as a current implementation review). The known runtime
exclusions above stay named, owned test obligations (replay, races between concurrent machine
instances, message duplication and reordering, migration, restore, and load) until the standalone
test-assurance contract's enforcement lanes land; naming an obligation in guidance enforces nothing
by itself. When machinery itself writes design artifacts, an interrupted publication blocks readers
fail-closed and `machinery recover <design-dir>` reports it read-only, with `--apply` completing
only a fully revalidated publication.

## Install

### Prerequisites

`machinery doctor` reports prerequisite and installation status, including the governance hook
state store described under
[the plugin's durable state](docs/claude-plugin.md#the-durable-state-store-and-its-retention).
`machinery doctor --repair` compacts that store. `machinery preflight` enforces the required
versions and release checks. Neither command installs anything.

**Required**

- **[modelith](https://modelith.sh/)**, for Phase 1 domain-model lint and render. With
  [Homebrew](https://brew.sh/) (macOS and Linux): `brew install stacklok/tap/modelith`. With the
  [Go](https://go.dev/dl/) toolchain on any OS:
  `go install github.com/stacklok/modelith/cmd/modelith@v0.4.0`, then put `$(go env GOPATH)/bin`
  on your `PATH`. Or download a prebuilt binary (macOS, Linux, Windows) from the
  [releases](https://github.com/stacklok/modelith/releases). machinery pins modelith at `v0.4.0`,
  and `machinery preflight` fails when the installed version does not match. Full options:
  [modelith.sh/cli](https://modelith.sh/cli/).
- **machinery**: the gate tools and formal generators, plus the agent skill and role docs. It is a
  single static binary with no Python and no Go runtime. There are three ways to install it.

  **One line, no clone.** This downloads the checksum-verified binary and installs the skill. It
  needs no Git, Go, or Make:
  ```bash
  curl -fsSL https://raw.githubusercontent.com/RamXX/machinery/main/install.sh | sh
  ```
  The script's exit status reports the install. It ends with `machinery preflight`; a gap that
  check finds (typically modelith, which only Phase 1 authoring needs, never `machinery check`) is
  printed and the script still exits 0. Set `MACHINERY_REQUIRE_PREFLIGHT=1` to make that check
  fatal in images that must carry the complete toolchain. The script puts `machinery` on
  `~/.local/bin` and runs `machinery install` to place the skill and role docs into your agent homes
  (real files under `~/.agents`, symlinked into `~/.claude`; see [Agent homes](#agent-homes)).
  Environment variables override it, for example `MACHINERY_VERSION=v0.9.0`,
  `INSTALL_DIR=/usr/local/bin`, `MACHINERY_HOMES="$HOME/Agent Home"`, or
  `MACHINERY_TARGETS="codex opencode"`. `MACHINERY_HOMES` takes one full path per line, preserving
  spaces; separate several homes with a literal newline.

  **Binary by hand** (macOS arm64 and x86, Linux amd64 and arm64): download
  `machinery-<os>-<arch>` from the [releases page](https://github.com/RamXX/machinery/releases),
  put it on your `PATH`, and let it install its own skill:
  ```bash
  machinery install                        # fetches the matching skill + role docs into your agent homes
  ```

  **Windows:** v0.9.0 publishes a cross-compiled `machinery-windows-amd64` binary and
  `machinery_<version>_windows_amd64.tar.gz` as release artifacts. The one-line installer and
  `machinery update` do not support Windows, so download the asset by hand. Native Windows runtime
  guarantees are not claimed: process custody, formal verification, and the assurance lanes are
  unix-only and refuse on Windows. Use Linux or macOS for the full toolchain.

  **Build from source** (with [Go](https://go.dev/dl/) 1.27+; `go.mod` pins 1.27.1). This is also
  how you get the unreleased consistency layer before the next release:
  ```bash
  go build -o machinery ./cmd/machinery    # then: machinery install --from .
  ```

**Optional**

- **[Java](https://adoptium.net/) 21.0.12.1+1**, only for `machinery verify-formal` and
  `machinery verify-c4`. machinery accepts distributor-independent OpenJDK/HotSpot builds of this
  exact checksum-pinned Temurin build, fingerprints the complete runtime closure, and runs it in a
  minimal fixed environment. `verify-formal` runs [TLC](https://github.com/tlaplus/tlaplus) to
  model-check the proofs and [Alloy](https://alloytools.org/) to check the relational models; the
  binary fetches the pinned, checksum-verified
  [tla2tools.jar](https://github.com/tlaplus/tlaplus/releases) and
  [org.alloytools.alloy.dist.jar](https://github.com/AlloyTools/org.alloytools.alloy/releases) into
  your cache on first use. The default engine path ignores ambient `java` and provisions the archive
  pinned in `.java-runtime-pin` into a private cache. To use a JDK you installed yourself (macOS
  `brew install --cask temurin`; Linux `sudo apt install default-jdk` or
  [download Temurin](https://adoptium.net/temurin/releases/); Windows
  `winget install EclipseAdoptium.Temurin.21.JDK`), set `MACHINERY_JAVA`; the override is accepted
  only with the exact source-controlled closure digest in `MACHINERY_JAVA_CLOSURE_SHA256`, and the
  version string alone is never trusted. Without Java you still get the full design and every
  deterministic gate. With it you add the machine-checked proofs.
- **[Structurizr CLI](https://github.com/structurizr/cli)**, only to export C4 diagrams from
  `workspace.dsl` (the [Structurizr DSL](https://github.com/structurizr/dsl) text and every gate
  need no export). It needs the same Java 21.0.12.1+1 runtime. Download a
  [release zip](https://github.com/structurizr/cli/releases), unzip it, and add it to `PATH`
  (`structurizr.sh` on macOS and Linux, `structurizr.bat` on Windows), or run the
  [container](https://hub.docker.com/r/structurizr/cli): `docker pull structurizr/cli`. The pure G2
  gate stays dependency-free. machinery provisions the ZIP named by its embedded `.structurizr-pin`
  trust root; an explicit `MACHINERY_STRUCTURIZR_CLI` override also requires its source-controlled
  full-tree digest in `MACHINERY_STRUCTURIZR_CLI_CLOSURE_SHA256`.

Why Java is optional: generating every artifact (the oracles, the TLA+ specs, the Alloy models) is
the Go binary's job and needs no JVM. Only checking the proofs runs under TLC and Alloy, which are
Java programs. The deterministic gates already catch malformed machines, drift, and boundary
erosion, but they cannot tell you that a saga can strand money, that a retry can loop forever, or
that a subsystem breaks the contract its neighbors assume. Model checking proves those
exhaustively. A setup without Java is a complete, gated design; adding Java upgrades "structurally
consistent" to "machine-checked."

### Commands

Everything after install is a `machinery` subcommand run on your own design path, with no clone and
no Make:

```sh
machinery install                     # place the skill + role docs into your agent homes
machinery update                      # force-refresh the binary and every recorded harness
machinery uninstall                   # remove them
machinery install --target all        # add native Claude, Codex, and OpenCode integration
machinery doctor --target all         # inspect every host-specific artifact
machinery uninstall --target all      # remove every host adapter and the shared skill
machinery preflight                   # enforce the pinned release prerequisites
machinery check <your-design>         # run the deterministic gate suite
machinery check <design> --gate gy --explain
                                      # the rules gate, with each finding's derivation
machinery check <design> --impl <dir> \
  --complete --warnings-as-errors     # require a final, closed, zero-warning handoff
machinery baseline <design> --impl .  # brownfield Stage 1: propose baseline rules, write the ratchet
machinery verify-formal <your-design> # regenerate + TLC/Alloy-check the proofs (needs Java)
machinery verify-c4 <your-design>     # compile workspace.dsl under structurizr-cli (needs Java)
machinery project <design>            # write the external-checker projections
machinery project <design> --facts <dir>
                                      # write the design's fact relations as .facts files
machinery verify-checkers <design>    # re-run external checkers and confirm their evidence
machinery oracle <dir|files...>       # regenerate transition oracles (a directory, or named machine files;
                                      #   valid machines regenerate past a broken sibling)
machinery oracle <dir> --diff \        # classify the churn instead of writing; --against reads the
  --against <git-ref>                 #   baseline at a git ref, so the affected-test list survives
                                      #   a regeneration that is already written
machinery alloy <design>              # generate the opted-in relational models and decision oracles
machinery packet <design> --milestone M1 --out <dir>
                                      # project bounded per-slice executor packets
machinery attest <path>               # print the content hash an attestation row must carry
machinery sweep <name> <design>       # list every hand-written mention of a unit/guard/event/knob
machinery embed refresh <design>      # re-copy every machinery:embed table from its source
                                      #   (--dry-run to report without writing)
machinery scale <design>              # measure a design and recommend sharding or recursion
machinery recover <design-dir>        # inspect an interrupted publication; --apply completes it
```

The `Makefile` is for contributors (building and testing machinery itself); `make help` lists its
targets. End users never need it. If you are hacking on machinery, see
[CONTRIBUTING.md](CONTRIBUTING.md) and run `make hooks` once to arm the pre-push gate.

### Agent homes

machinery's core is host-neutral: one Agent Skill, one pair of canonical role bodies, one artifact
contract, and one deterministic CLI. `machinery install` places the skill under
`<home>/skills/machinery` and the two role docs under `<home>/agents`. The default homes are
`~/.agents` (the cross-agent convention) and `~/.claude` (Claude Code). The first holds real files
and the rest are symlinked to it, so there is one canonical copy to update. Each release also
publishes `machinery-source.tar.gz`, a reproducible, commit-timestamped snapshot with normalized
ownership. Its digest is in `checksums-sha256.txt`, and install and update fetch that exact asset
for matching skill and role sources instead of an unversioned branch archive.

- **`machinery install` (recommended).** Fetches the skill from the release that matches the binary
  and lays it down as above. `--home` (repeatable) overrides the set, `--copy` copies into every
  home instead of symlinking, and `--from <dir>` installs from a local checkout.
  `machinery uninstall` removes it.
- **`make dev-link` (developer).** Symlinks every home into a working-tree checkout so edits to the
  skill are live. Use it when hacking on machinery itself.

For native host surfaces, use repeatable `--target claude|codex|opencode|all`. Codex gets TOML
subagents under `~/.codex/agents`; OpenCode gets native subagents, commands, and a governance
plugin under `~/.config/opencode`; both reuse the skill at `~/.agents/skills`. The renderers strip
host-specific frontmatter and wrap the same canonical role bodies, so there are no per-host prompt
forks to drift apart. `--target` and `--home` cannot be combined.

```bash
machinery install --target codex
machinery install --target opencode
machinery install --target all
machinery doctor --target all
```

Every runtime follows the capability contract in the skill: use fresh-context roles when supported,
otherwise run the same role inline; use lifecycle hooks when supported, otherwise run explicit
checks. CI `machinery check` is always the merge authority. The topology, feature matrix, the
OpenCode stop-hook limitation, and the adapter extension rules are in the
[agent portability guide](docs/agent-portability.md).

### Updating

Every successful CLI install records its exact topology (custom home groups, symlink or copy mode,
and native targets) in the user config directory. `machinery update` uses that receipt, plus
standard-path discovery for older installations, to force-download the requested release, verify its
published checksum and reported version, replace the running binary atomically, and refresh every
recorded harness from the same release source. It does not skip work when the requested version
matches the installed version.

```bash
machinery update                         # latest release, all detected installations
machinery update --version v0.9.0        # force an exact release
machinery update --target all            # restrict the harness refresh explicitly
machinery update --skip-plugins          # leave host-managed plugin caches alone
```

An installation spans several filesystem roots (the binary, a direct home group, native targets).
`machinery update` swaps the binary first and then refreshes every recorded placement from the same
release, inside one transaction: if any step fails, every root is restored to the previous release
and the command exits non-zero naming the failed step. Between the binary swap and the end of the
refresh, a running agent host can briefly see the new binary next to the previous release's skill
and role docs; that window lasts as long as the refresh, and nothing in it is left behind.

Re-running the one-line installer over an existing install converges with `machinery update`: when
a valid receipt exists, the bootstrap uses the complete recorded home, native-target, and
host-plugin plan, exactly what an ordinary `machinery update` without selectors uses, preserving
each recorded group's copy or symlink mode. A first bootstrap with no receipt installs the
plugin-aware default homes. Explicit `MACHINERY_HOMES` or `MACHINERY_TARGETS` do not extend that
bootstrap plan; they replace it with ordinary update selectors.

Claude Code and Codex own their plugin caches. When those plugins are detected, update asks the host
CLI to refresh them. It never edits cache directories directly, and it cannot roll a host plugin
cache back. Host plugin refresh runs after the binary and every direct placement have committed, so
a missing host CLI, a failed or misunderstood plugin inventory, a managed-scope refusal, or a failed
refresh is a returned failure, not a warning over a successful update: the direct update stays
committed, the error names the exact retry (`claude plugin update machinery@machinery` or
`codex plugin add machinery@machinery`), and the unmet obligation is recorded so the next update
retries it. That post-commit plugin failure is distinct from plugin discovery, which runs earlier
while planning: an unsafe or uncertain plugin-ownership discovery error fails the whole update
before anything changes. Pass `--skip-plugins` to opt out of host plugin management entirely;
receipt and ownership inspection still run. Open a new Codex task or run Claude Code's
`/reload-plugins` after a plugin refresh. Failure and recovery behavior is in the
[agent portability guide](docs/agent-portability.md#updating-a-release).

Keep the binary and the Claude Code plugin on the same version. The plugin cache is host-owned and
updates separately (`/plugin`), so one can move without the other. machinery detects the skew and
refuses rather than running a hook against a binary it was not built for:

```
cached machinery plugin version 0.7.2 does not match running machinery v0.7.1;
run 'claude plugin update machinery@machinery'
```

### Claude Code plugin (optional, recommended for Claude Code)

For Claude Code, this repository is also a plugin: the same skill and role agents, the
`/machinery:design`, `/machinery:check`, `/machinery:init`, and `/machinery:status` commands, and
hooks that make the deterministic half of the gates part of every session. Install the binary first
(the hooks call it), then the plugin:

```bash
curl -fsSL https://raw.githubusercontent.com/RamXX/machinery/main/install.sh | sh
```

```
/plugin marketplace add RamXX/machinery
/plugin install machinery@machinery
```

In a machinery-managed project (a `.machinery.json` at the root, or the conventional
`design/domain.modelith.yaml`), the hooks announce the governance contract at session start, deny
hand-edits to generated artifacts (`*.oracle.md`, `formal/*.tla`, `*.cfg`, and `*.als`, `packs/`,
`pack/`, `ratchet.json`), and run `machinery check` before any turn that touched the design (or
watched sources, with `"impl"` configured) is allowed to end. DRIFT and armed import-boundary
violations block; mid-phase ERRORs only warn. During a deliberate multi-agent wave, the operator
creates `<design>/.machinery-wave` and red gates surface as messages instead of blocking while it
is open. The sentinel is operator-created, never agent-created: the PreToolUse hook denies agent
writes to it, so a session cannot defer its own gates (deleting it, which closes the wave, stays
allowed). In every other repository the hooks are a strict no-op and never disturb other projects
or plugins. Details, including the `.machinery.json` reference and the exact sentinel contents:
the [Claude Code plugin guide](docs/claude-plugin.md).

With the plugin installed, the default `machinery install` detects it and skips `~/.claude` (the
plugin already serves the skill and agents there). Codex and OpenCode users can opt into their
native adapters with `--target`.

## Quickstart

Install without cloning, then run the binary on any design:

```bash
curl -fsSL https://raw.githubusercontent.com/RamXX/machinery/main/install.sh | sh
machinery check <your-design>            # deterministic gates
machinery verify-formal <your-design>    # proofs, if Java is present
```

To try it on the bundled examples, clone and build (the only reason to clone):

```bash
git clone https://github.com/RamXX/machinery && cd machinery
make build
.bin/machinery check examples/go-crm/design --impl examples/go-crm/impl
```

The check prints one block per gate, each with its `checked:` counts and a verdict. The last lines
summarize:

```
0 blocking (ERROR/DRIFT) finding(s)
platform-green: design gates, G4-import, and Gt-tests all green
```

Then, if Java is present:

```bash
make verify-formal   # regenerates and checks all 35 TLC proofs + the relational (Alloy) suites
```

## Use

In an agent session (Claude Code, Codex, OpenCode, or any runtime that loads Agent Skills), from the
project you want to design:

```
Design a new <system> with machinery.
```

With the Claude Code plugin, `/machinery:design <what you want>` does the same thing explicitly, and
`/machinery:status` reports where a design stands. The conductor starts at Phase 0. It is fully
standalone: no tracker, no project settings, no other process dependencies. Target languages it
realizes: Elixir, Go, Rust, TypeScript, Python.

The skill also defines a revision mode (design changes after code exists: stable test ids, oracle
diffs as the affected-test list, and a mandatory state-migration note for persisted machines) and a
sharding rule for designs beyond roughly ten stateful components. `machinery scale` measures a
design and recommends sharding or recursive decomposition; whether a subsystem team needs its own
contract is still a human decision.

## How it is put together

- `skills/machinery/SKILL.md`: the conductor, plus phase-selected `references/` (XState format, C4
  technique, verification evidence, archaeology classification, the BUILD.md template, and bounded
  execution packets) and `tools/` (the TLC shell wrappers `tlc.sh` and `verify_formal.sh`, and the
  tools README).
- `cmd/machinery/`: the single Go binary (cobra CLI): `lint`, `oracle`, `tla`, `alloy`, `refine`,
  `compose`, `check`, `attest`, `project`, `packet`, `verify-checkers`, `baseline`,
  `verify-formal`, `verify-c4`, `pack`, `scale`, `sweep`, `embed`, `tokens-equal`, `doctor`,
  `preflight`, `install`, `update`, `uninstall`, `recover`, `completion`, and `version`.
- `internal/`: the Go toolchain. `ir/` (order-preserving machine model), `lint/`, `oracle/`,
  `tla/`, `alloy/` (the relational generators), `refine/`, `compose/`, `gates/` (the gate suite),
  `checker/` (the projection and its relation catalog), `datalog/` (the in-process rule evaluator),
  `pack/` (recursive decomposition via contract packs), `formal/` (TLC and Alloy orchestration),
  `install/` (skill placement), `hook/` (the governance hooks), and `experiments/` (the adversarial
  mutation suite). Every package has unit tests.
- `rules/consistency/`: the shipped Datalog rules, catalogued in [rules/README.md](rules/README.md).
- `schemas/`: the projection (1.0 and 2.0) and evidence JSON schemas.
- `agents/`: the two synthesis roles (the machine author and the build-doc writer).
- `commands/`, `hooks/`, `.claude-plugin/`, and `.codex-plugin/`: the Claude Code and Codex plugin
  surfaces (slash commands where supported, the shared gate-enforcing hooks and shim, and both
  manifests). The repository root is the plugin.
- `adapters/opencode/`: the OpenCode commands and a thin JavaScript translation layer. The
  methodology and gate behavior stay in the shared skill and binary.
- `examples/`: go-crm, surreal-crm, fulfillment, portfolio-engine, checkout-split, and pii-flow, as
  described above. `examples/inventory.tsv` lists every bundled design and how CI runs it.
- `testdata/golden/`: the byte-for-byte golden corpus (see below).
- `docs/`: the guides.
  - Layers: [policy](docs/policy-layer.md), [integrity](docs/integrity-layer.md),
    [isolation](docs/isolation-layer.md), [external checkers](docs/external-checkers.md) (the
    projection and evidence schemas, the manifest, the git-ignored resolution registry, the pure
    `gk` gate and the `verify-checkers` engine phase, and how to wrap an engine you cannot modify),
    and the [consistency layer](docs/consistency-layer-proposal.md) (declarations, projection 2.0,
    the rules, and each stage as implemented).
  - Delivery: [milestone acceptance](docs/acceptance-gate.md),
    [packet projection](docs/packet-projection.md), [compile-time design reads](docs/declared-reads.md),
    [attestation evidence](docs/attestation-evidence.md).
  - Existing systems: [rebuild and hybrid](docs/rebuild-guide.md),
    [surface ledger](docs/surface-ledger.md), [target surfaces](docs/target-surfaces.md),
    [brownfield team guide](docs/brownfield-team-guide.md).
  - Hosts: [Claude Code plugin](docs/claude-plugin.md) and the
    [agent portability guide](docs/agent-portability.md).
  - Contracts not yet shipped as commands: the
    [decision-lifecycle refinement pattern](docs/decision-lifecycle-pattern.md) (a draft rung-4 design
    note), the [native custody contract](docs/native-custody-contract.md), and the
    [executable test assurance contract](docs/test-assurance-contract.md). The assurance contract
    opens with an "Implementation status in 0.7.0" section: the release ships the version-1
    declaration grammar, the obligation inventory, the capture and registration store, and four
    native test adapters exercised by the required integration lane, but no `machinery tdd`
    command, no `check --store` or `--assurance strict` flag, and no gate that reads
    `design/assurance/`.

See `skills/machinery/tools/README.md` for the checkers and generators, and
`examples/go-crm/design/formal/README.md` for the proofs.

## Testing and CI

Every Go package has unit tests. The adversarial mutation suite in `internal/experiments` encodes
every vacuity and drift attack found in the design reviews (lint mutations plus the full gate suite
run against synthesized design and implementation fixtures) as permanent regressions. Current
own-package coverage:

| Package | Coverage | Role |
|---------|----------|------|
| `internal/datalog` | 96.1% | the in-process Datalog evaluator |
| `internal/version` | 94.4% | version stamps and skew detection |
| `internal/oracle` | 92.9% | transition oracle (content-hashed ids) |
| `internal/lint` | 89.4% | structural lint and matrix reconciliation |
| `internal/tla` | 89.2% | TLA+ control-flow generator |
| `internal/alloy` | 89.2% | policy, integrity, and isolation generators and oracles |
| `internal/refine` | 86.3% | data refinement (3 patterns) |
| `internal/compose` | 85.7% | cross-aggregate composition |
| `internal/gates` | 85.6% | the gate suite (G5 also exercised through `internal/experiments`) |
| `internal/hook` | 77.8% | progressive governance and generated-artifact protection |
| `internal/checker` | 74.8% | projection, relation catalog, checker registry and evidence |
| `internal/install` | 74.2% | skill placement behind `machinery install` and `update` |
| `internal/formal` | 74.2% | TLC and Alloy orchestration (solver-run paths need Java) |
| `internal/pack` | 73.1% | contract packs (the mutation suite lives in `internal/experiments`) |
| `internal/ir` | 67.1% | shared IR (covered further through lint and gates) |

These are own-package figures from `go test -cover` on this branch; the cross-package adversarial
suites in `internal/experiments` exercise gates and packs further, and `cmd/` is thin CLI plumbing.

Run `go test -coverprofile=cover.out ./internal/... && go tool cover -func=cover.out` locally. CI
runs `go test -race ./...`. Beyond unit tests, these nets are always green in CI:

- **Golden corpus.** `testdata/golden` byte-compares stdout, stderr, exit code, and every generated
  artifact for the deterministic subcommands: lint, oracle, and tla on the four standalone examples
  (go-crm, fulfillment, portfolio-engine, pii-flow), refine and compose where semantics annotations
  exist, check on every example design root (pinning the Gy-rules, G5-pack, and Gk output among the
  rest), and pack generate and scale on checkout-split (`make golden`, re-captured with
  `make golden-update` after intended output changes). The same corpus runs natively on Linux and
  macOS; Windows is cross-compiled only.
- **Formal verification.** `machinery verify-formal` regenerates and TLC-model-checks all 35 TLA+
  proofs across the seven example designs that carry formal suites (8 in go-crm, 8 in surreal-crm,
  8 in fulfillment, 6 in portfolio-engine, 4 in checkout-split, two per child including the
  contract-refinement proofs, and 1 in pii-flow), plus the Alloy suites, and the required workflow
  rejects any generated diff after the solver run. The sole hand-written exception is a strict
  manual TLA pair: the module's first line is exactly `\* machinery:manual` and a same-basename
  `.cfg` is mandatory. Manual pairs are reported as declared/not-regenerated while TLC still checks
  them; an unmarked orphan pair or half fails.
- **Rules parity.** The `datalog-parity` job runs every shipped rule file and the evaluator's
  program corpus under both engines, as described above.
- **Engine reproduction.** Required CI installs Modelith v0.4.0 and reproduces every committed
  domain render after the mechanical house-style normalization; compiles every example
  `workspace.dsl` with checksum-pinned Structurizr CLI v2025.11.09; provisions the pinned checker
  runtime by digest for `linux/amd64`, verifying its local `RepoDigests` and OS/architecture;
  rebuilds the pii-flow image reproducibly and refuses any digest but its pin; and re-runs every
  registered checker offline with `--pull=never`. These jobs are the engine halves;
  `machinery check` stays hermetic and dependency-free.
- **Required integration lane.** `go run ./scripts/integration-lane --lane required` runs every
  infrastructure-dependent suite (Docker-backed checker lifecycle, publication recovery, native
  custody, adapter governance, and the documented checker registry and bind-path example) with
  exact test inventories, pinned runtimes, real process and teardown accounting, and no skips.
  Missing infrastructure fails the lane with a diagnostic. The same lane fragments run in local
  preflight and hosted CI.

### Where the gates run

The same gates run in tiers. A lower tier never substitutes for a higher one, and each tier states
what it cannot cover instead of reporting a thinner run as green.

| Tier | Command | Covers | Cost |
|------|---------|--------|------|
| Pre-push | `make preflight-fast` | the cheap tier the hook enforces on every push | under 5 minutes |
| Local, native | `make preflight` | the fast tier plus the race sweep, the integration lane, formal verification, C4 compilation, checker reproduction, and rules parity | tens of minutes |
| Local, containerized | `make dagger-ci` | every containerizable hosted job, identical to what CI runs | tens of minutes |
| Release gate | hosted CI | the same module functions, plus the macOS-only jobs | every push |

The [Dagger](https://dagger.io/) module in `.dagger/` is how CI runs, not a local copy of it: every
Linux job in `ci.yml`, `formal.yml`, and `security.yml` is a `dagger call` of the function named
after it, so each gate has one definition and `make dagger-ci` runs what the runner runs. Run one
job with `make dagger-job JOB=lint`; `dagger functions` lists them. The hosted runner installs the
Dagger CLI at the version `dagger.json` owns (v0.21.9), verified against a committed checksum.

The module does not restate the runtime pins. Its base container is built from
`scripts/ci-linux.dockerfile`, the single owner of the Go (1.27.1), Node (26.8.1) with TypeScript
(7.0.2), CPython (3.14.7), and Elixir (1.20.4) on OTP (29.0.6) identities, shared with
`make ci-linux`. The race sweep runs as an unprivileged user inside the container, because root
bypasses permission bits and would turn the custody tests that assert an unwritable path is refused
into silent passes. The wiring guards in `cmd/machinery/repository_contract_test.go` and
`scripts/integration-lane/main_test.go` require the module to carry each pinned command verbatim and
the workflows to delegate rather than restate it, so neither half can drift from the other.

Three things the containerized tier does not claim:

- **The macOS jobs.** `native-tests`, `golden-native`, and the native darwin `build-native` exercise
  the supported non-Linux filesystem and the byte corpus on darwin. A Linux container cannot
  reproduce them, so hosted CI is their only gate.
- **The two platform-pinned jobs on an arm64 host.** The assurance catalog pins `linux/amd64` and
  `darwin/arm64` as the only native assurance platforms, so `test` and `integration-required` fail
  closed in a `linux/arm64` container. On Apple Silicon, run them natively through
  `make preflight`, or containerized on a `linux/amd64` machine. Emulation is not a workaround: the
  identity probes build a closed environment that cannot carry the BEAM flags an emulated OTP needs.
- **GitHub-native checks.** `dependency-review` is a hosted action with no local equivalent.

## Built on

machinery is a thin methodology over these projects. It invokes them or emits their notations and
bundles none of them.

- [Modelith](https://modelith.sh/): the domain-model language and linter (Phase 1).
- [C4 model](https://c4model.com/): the architecture technique (Phase 2).
- [Structurizr DSL](https://github.com/structurizr/dsl) and
  [Structurizr CLI](https://github.com/structurizr/cli): architecture as code, and optional C4
  diagram export.
- [XState](https://github.com/statelyai/xstate) and [Stately](https://stately.ai/): the
  state-machine JSON format (notation only; machinery does not run the library) and its visualizer.
- [TLA+ and TLC](https://github.com/tlaplus/tlaplus): the specification language and model checker
  for behavior.
- [Alloy](https://alloytools.org/): the relational model finder for the static relational layers.
- [Soufflé](https://souffle-lang.github.io/): the Datalog engine the shipped rules are held to in
  the parity lane, and the engine inside the pii-flow reference image.
- [Eclipse Temurin / Adoptium](https://adoptium.net/): the JVM that runs TLC and Alloy.
- [Go](https://go.dev/): to build machinery from source and to install Modelith.
- [Dagger](https://dagger.io/): the CI module every Linux job runs through.
- [LadybugDB](https://github.com/LadybugDB/go-ladybug): the embedded store used only by the go-crm
  example, not a machinery dependency.

## License

Copyright 2026 Ramiro Salas. Licensed under the Apache License 2.0; see `LICENSE`. machinery invokes
`modelith` and emits XState and C4 notation; it bundles none of them, so no dependency's license
binds it. The tools it works with are permissively licensed and compatible with Apache-2.0:
Modelith and Structurizr are Apache-2.0, XState and LadybugDB are MIT, and C4 is an open notation.
