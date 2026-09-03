# machinery

[![CI](https://github.com/RamXX/machinery/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/RamXX/machinery/actions/workflows/ci.yml)
[![Formal Verification](https://github.com/RamXX/machinery/actions/workflows/formal.yml/badge.svg?branch=main)](https://github.com/RamXX/machinery/actions/workflows/formal.yml)
[![Security](https://github.com/RamXX/machinery/actions/workflows/security.yml/badge.svg?branch=main)](https://github.com/RamXX/machinery/actions/workflows/security.yml)
[![Nightly](https://github.com/RamXX/machinery/actions/workflows/nightly.yml/badge.svg?branch=main)](https://github.com/RamXX/machinery/actions/workflows/nightly.yml)
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
*derived* from the domain model rather than invented. The final artifact is a `BUILD.md` a zero-context
coder can build from, plus the machines that are simultaneously the test oracle and the formal spec.

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
         tool: Gx-trace + Gb-plan + Ge-embed (+ G4-import and Gt-tests once code exists)
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
the phases as archaeology instead of invention: write the Modelith model, the contract, and the
machines to describe the system AS IT IS, then let the gates arbitrate what they can see. Be
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

## Install

### Prerequisites

`machinery preflight` (or `machinery doctor`) checks all of these at any time and warns about
anything missing; it installs nothing.

**Required**

- **[modelith](https://modelith.sh/)** -- Phase 1 domain-model lint and render. Primary install is
  [Homebrew](https://brew.sh/) (macOS and Linux): `brew install stacklok/tap/modelith`. Secondary:
  with the [Go](https://go.dev/dl/) toolchain on any OS,
  `go install github.com/stacklok/modelith/cmd/modelith@v0.4.0` (then put `$(go env GOPATH)/bin` on
  your `PATH`); or download a prebuilt
  binary (macOS, Linux, Windows) from the
  [releases](https://github.com/stacklok/modelith/releases). machinery pins modelith at `v0.4.0`,
  and `machinery preflight` fails when the installed version does not match the pin. Full options:
  [modelith.sh/cli](https://modelith.sh/cli/).
- **machinery** -- the deterministic gate tools and formal generators, plus the agent skill and role
  docs. A single static binary (no Python, no Go runtime). Three ways to install:

  **One line, no clone** (downloads the checksum-verified binary, then installs the skill; no Git, no
  Go, no Make):
  ```bash
  curl -fsSL https://raw.githubusercontent.com/RamXX/machinery/main/install.sh | sh
  ```
  This puts the `machinery` binary on `~/.local/bin` and runs `machinery install` to place the skill
  + role docs into your agent homes (real files under `~/.agents`, symlinked into `~/.claude`; see
  [Agent homes](#agent-homes)). Override with environment variables, for example
  `MACHINERY_VERSION=v0.1.7`, `INSTALL_DIR=/usr/local/bin`, `MACHINERY_HOMES="$HOME/Agent Home"`, or
  `MACHINERY_TARGETS="codex opencode"`. `MACHINERY_HOMES` accepts one full path per line, preserving
  spaces; use a literal newline between multiple homes.

  **Binary by hand** (macOS arm64/x86, Linux amd64/arm64, Windows amd64): download
  `machinery-<os>-<arch>` from the [releases page](https://github.com/RamXX/machinery/releases), put
  it on your `PATH`, then let it install its own skill:
  ```bash
  machinery install                        # fetches the matching skill + role docs into your agent homes
  ```

  **Build from source** (if you have [Go](https://go.dev/dl/) 1.27+; `go.mod` pins 1.27.0):
  ```bash
  go build -o machinery ./cmd/machinery    # then: machinery install --from .
  ```

**Optional**

- **[Java](https://adoptium.net/) 21.0.12.1+1** -- only for `machinery verify-formal` and
  `machinery verify-c4`; machinery accepts distributor-independent OpenJDK/HotSpot builds of this
  exact checksum-pinned Temurin build, fingerprints the complete runtime closure, and runs it in a minimal fixed
  environment. `machinery verify-formal` runs
  [TLC](https://github.com/tlaplus/tlaplus) to model-check the proofs and, on designs with a policy
  annotation, [Alloy](https://alloytools.org/) to check the relational policy model (the binary
  fetches the pinned, checksum-verified
  [tla2tools.jar](https://github.com/tlaplus/tlaplus/releases) and
  [org.alloytools.alloy.dist.jar](https://github.com/AlloyTools/org.alloytools.alloy/releases) into
  your cache on first use). macOS:
  `brew install --cask temurin`; Linux: `sudo apt install default-jdk` or
  machinery provisions the archive pinned in `.java-runtime-pin` into its private cache, or [download Temurin](https://adoptium.net/temurin/releases/);
  Windows: `winget install EclipseAdoptium.Temurin.21.JDK` or
  [download](https://adoptium.net/temurin/releases/). Without Java you still get the full design and
  every deterministic gate; with it you add the machine-checked proofs (the top of the correctness
  ladder above). The default engine path ignores ambient `java` and provisions the committed archive
  pin. An explicit `MACHINERY_JAVA` override is accepted only with the exact source-controlled
  closure digest in `MACHINERY_JAVA_CLOSURE_SHA256`; the version string alone is never trusted.
- **[Structurizr CLI](https://github.com/structurizr/cli)** -- only to export C4 diagrams from
  `workspace.dsl` (the [Structurizr DSL](https://github.com/structurizr/dsl) text and every gate need
  no export); needs the same Java 21.0.12.1+1 runtime. Any OS: download a
   [release zip](https://github.com/structurizr/cli/releases), unzip, and add it to `PATH`
   (`structurizr.sh` on macOS/Linux, `structurizr.bat` on Windows); or run the
   [container](https://hub.docker.com/r/structurizr/cli): `docker pull structurizr/cli`. The pure
   G2 gate stays dependency-free; this repository's required CI downloads a checksum-pinned CLI and
   compiles every committed `workspace.dsl` as the engine half. Machinery provisions the ZIP named
   by its embedded `.structurizr-pin` trust root. An explicit `MACHINERY_STRUCTURIZR_CLI` override
   additionally requires its source-controlled full-tree digest in
   `MACHINERY_STRUCTURIZR_CLI_CLOSURE_SHA256`.

Everything after install is a `machinery` subcommand run on your own design path, no clone and no
Make:

```sh
machinery install                     # place the skill + role docs into your agent homes
machinery update                      # force-refresh the binary and every recorded harness
machinery uninstall                   # remove them
machinery install --target all        # add native Claude, Codex, and OpenCode integration
machinery doctor --target all         # inspect every host-specific artifact
machinery uninstall --target all      # remove every host adapter and the shared skill
machinery preflight                   # check prerequisites (warns; installs nothing)
machinery check <your-design>         # run the deterministic gate suite
machinery baseline <design> --impl .  # brownfield Stage 1: propose baseline rules, write the ratchet
machinery verify-formal <your-design> # regenerate + TLC-check the proofs (needs Java)
machinery oracle <dir|files...>       # regenerate transition oracles (a directory, or named machine files;
                                      #   valid machines regenerate past a broken sibling)
machinery oracle <dir> --diff \        # classify the churn instead of writing; --against reads the
  --against <git-ref>                 #   baseline at a git ref, so the affected-test list survives
                                      #   a regeneration that is already written
machinery sweep <name> <design>       # list every hand-written mention of a unit/guard/event/knob
machinery embed refresh <design>      # re-copy every machinery:embed table from its source
                                      #   (--dry-run to report without writing)
```

The `Makefile` is contributor-only (building and testing machinery itself); `make help` lists those
targets. End users never need it. If you are hacking on machinery itself, see
[CONTRIBUTING.md](CONTRIBUTING.md) and run `make hooks` once to arm the pre-push gate.

### Agent homes

machinery's core is host-neutral: one Agent Skill, one pair of canonical role bodies, one artifact
contract, and one deterministic CLI. `machinery install` preserves the original portable layout by
placing the skill under `<home>/skills/machinery` and the two role docs under `<home>/agents`. The
default homes are `~/.agents` (the cross-agent convention) and `~/.claude` (Claude Code); the first
holds real files and the rest are symlinked to it, so there is one canonical copy to update.
Each release also publishes `machinery-source.tar.gz`, a reproducible, commit-timestamped snapshot
with normalized ownership. Its digest is in `checksums-sha256.txt`; install and update fetch that
exact asset for matching skill and role sources instead of an unversioned branch archive.

- **`machinery install` (recommended).** Fetches the skill from the release that matches the binary
  and lays it down as above. `--home` (repeatable) overrides the set, `--copy` copies into every home
  instead of symlinking, and `--from <dir>` installs from a local checkout. `machinery uninstall`
  removes it.
- **`make dev-link` (developer).** Symlinks every home directly into a working-tree checkout, so
  edits to the skill are live. That is what you want when hacking on machinery itself, not for a
  plain install.

For native host surfaces, use repeatable `--target claude|codex|opencode|all`. Codex gets TOML
subagents under `~/.codex/agents`; OpenCode gets native subagents, commands, and a governance plugin
under `~/.config/opencode`; both reuse the skill at `~/.agents/skills`. The renderers strip
host-specific frontmatter and wrap the same canonical role bodies, so there are no Claude, Codex,
and OpenCode prompt forks to drift apart. `--target` and `--home` cannot be combined.

```bash
machinery install --target codex
machinery install --target opencode
machinery install --target all
machinery doctor --target all
```

Every runtime follows the capability contract in the skill: use fresh-context roles when supported,
otherwise execute the same role inline; use lifecycle hooks when supported, otherwise run explicit
checks. CI `machinery check` is always the merge authority. The exact topology, feature matrix,
OpenCode stop-hook limitation, and adapter extension rules are in the
[agent portability guide](docs/agent-portability.md).

Every successful CLI install records its exact topology (custom home groups, symlink/copy mode,
and native targets) in the user config directory. `machinery update` uses that receipt, plus
standard-path discovery for older installations, to force-download the requested release, verify
its published checksum and reported version, replace the running binary atomically, and refresh
every recorded harness from the same release source. It does not skip work when the requested
version matches the installed version.

```bash
machinery update                         # latest release, all detected installations
machinery update --version v0.5.0         # force an exact release
machinery update --target all            # restrict the harness refresh explicitly
machinery update --skip-plugins          # leave host-managed plugin caches alone
```

Claude Code and Codex own their plugin caches. When those plugins are detected, update asks the
host CLI to refresh them; it never edits cache directories directly. A missing host CLI or a
managed-scope refusal is reported as a warning while direct skills and adapters still update. Open
a new Codex task or run Claude Code's `/reload-plugins` after a plugin refresh. Full failure and
recovery behavior is in the [agent portability guide](docs/agent-portability.md#updating-a-release).

The gate tools are a single Go binary (no Python runtime). `verify-formal` downloads a version-pinned,
checksum-verified `tla2tools.jar` on first use. CI runs the test suite, all gate runs, the full formal
suite with a generated-diff assertion, pinned Modelith render reproduction, native macOS and Windows
golden corpora, Structurizr compilation for every example, the pinned OCI external-checker closure,
cross-compile builds, security scanning, and the go-crm build on every push.

### Claude Code plugin (optional, recommended for Claude Code)

For Claude Code specifically, this repository is also a plugin: the same skill and role agents,
plus `/machinery:design`, `/machinery:check`, `/machinery:init`, and `/machinery:status` commands,
plus hooks that make the deterministic half of the gates non-optional inside every session. Install
the binary first (the hooks call it), then the plugin:

```bash
curl -fsSL https://raw.githubusercontent.com/RamXX/machinery/main/install.sh | sh
```

```
/plugin marketplace add RamXX/machinery
/plugin install machinery@machinery
```

In a machinery-managed project (a `.machinery.json` at the root, or the conventional
`design/domain.modelith.yaml`), the hooks: announce the governance contract at session start, deny
hand-edits to generated artifacts (`*.oracle.md`, `formal/*.tla` and `*.cfg`, `packs/`, `pack/`),
and run `machinery check` before any turn that touched the design (or watched sources, with
`"impl"` configured) is allowed to end; DRIFT and import-boundary violations block, mid-phase
ERRORs only warn. During a deliberate multi-agent wave, touch `<design>/.machinery-wave` (first
line: TTL in minutes, default 45) and red gates surface as messages instead of blocking until the
sentinel is deleted or its TTL lapses; a stale sentinel is reported and ignored. The sentinel is
operator-created, never agent-created: the PreToolUse hook denies agent file-tool writes to it, so
a session cannot defer its own gates (deleting it, which closes the wave, stays allowed). In every other repository the hooks are a strict no-op: they exit before reading
anything, so the plugin never disturbs other projects or plugins. Details, including the
`.machinery.json` reference: the [Claude Code plugin guide](docs/claude-plugin.md).

With the plugin installed, the default `machinery install` detects it and skips `~/.claude` (the
plugin already serves the skill and agents there). The legacy no-target install remains unchanged;
Codex and OpenCode users can opt into their native adapters with `--target`.

## Quickstart (five minutes)

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

The check prints one block per gate. Each block carries a `checked:` line, the exact counts of
what was actually verified (a gate that finds nothing to check fails rather than passing), and a
verdict: `ok`, or findings at three severities defined in the table below. The final line
summarizes blocking findings; a zero there is a clean design. Then, if Java is present:

```bash
make verify-formal   # regenerates and checks all 35 TLC proofs + the relational (Alloy) suites
```

| Gate | One line |
|---|---|
| Gate 1 | `modelith lint` on the domain model (Phase 1). The binary has no `g1`, by design: Phase 1's gate is modelith's own linter. |
| Gm-transition | rebuild/hybrid designs only: every legacy entity disposed, replaced attributes and lifecycle values fully mapped, ordered source-of-truth phases with rollback, evidence-based cutover, owned transitional risks; mapping rows may declare `tests:` and with `--impl` every named regression identifier must appear in the test corpus. |
| Gs-surface | designs with a legacy surface ledger only: all six surface classes inventoried or waived, every route/command/table/job/event/integration covered, dropped, or deferred, and covered bindings resolve against the target design. An `as_of:` anchor that is neither an ISO date, a VCS revision, nor a tag-like token warns: the format is not pinned by the schema, but prose where an anchor belongs compares against nothing. |
| Gu-surfaces | designs with a target surface ledger only (`design/surfaces.yaml`): every action whose actor is a person is mapped to a named surface (screen, admin command, API route, config release) or explicitly deferred with a reason, each mapped act resolves against the domain model and matches the actor it declares, every act is stated exactly once, persona-level deferrals name an actor the model actually declares, `sources:` (required) names where the act list was enumerated from, and once BUILD.md declares milestones every `milestone:` resolves to one of them by number or title. |
| Gp-policy | designs with a policy annotation only: it binds to the domain model, covers every top-level invariant, and the committed `Policy.als` and `Policy.oracle.md` byte-match a fresh generation. |
| Gi-integrity | designs with an integrity annotation only: it binds to the domain model and the committed `Integrity.als` byte-matches a fresh generation. |
| Gn-isolation | designs with an isolation annotation only: it binds to the domain model and the committed `Isolation.als` and `Isolation.oracle.md` byte-match a fresh generation. |
| Gk-\<id\> | designs with a checker manifest only, one instance per manifest: the committed projection is fresh, evidence binds to it via `input_hash`, the verdict is pass, and the manifest's coverage claim is complete. Engine via `machinery verify-checkers`; bring-your-own deterministic checker (SAST, AST, Datalog, graph reasoner). |
| Gc-carrier | every declared invariant has a named carrier: an action's `preserves`, a relational layer, a machine matrix unit, an external checker's coverage claim, or an explicit waiver with a reason (`formal/waivers.yaml` or a layer's residuals). Needs only the domain model, so it runs from Phase 1; declaring an obligation the design does not carry fails the moment it is written, not months later. |
| G2-c4 | the Architecture Contract parses, binds to `workspace.dsl`, the allow graph is acyclic (cycles closing only through `baseline:` edges warn as ratchet debt), `assert: no_path` claims hold over the transitive closure, every dependency has a mitigation row, the NFR record exists and mentions security, capacity, and observability, every event-contract table names its enumeration source, and every allowed boundary crossing has an interface-contract row (edge, shape, errors, idempotency) or a `(no contract: <reason>)` waiver, with no row for an edge no allow rule declares. Every relationship `workspace.dsl` DRAWS is judged by the same allow/deny/baseline rules G4-import judges a code edge by (an endpoint no element declares is an error; an endpoint the contract never claimed carries no obligation and stays visible in the counts; the converse is not required, since a diagram is legitimately partial). Every event-contract row answers each of its columns (producer, consumer, payload, delivery, ordering, dedupe) and its producer and consumer resolve to a declared element or external, the same resolution a mitigation row gets. Opt-in by table presence: the action-ownership table (every model action owned exactly once by a resolvable component) and the adoption-closure table (every closure member declared and mitigated, scorecard cells dated). `machinery verify-c4` is the engine phase: workspace.dsl compiles under structurizr-cli export. |
| G3-machine | machines pass structural lint, the TLA generator's own admissibility pass (a machine G3 passes is a machine `verify-formal` can generate), retry-counter accounting (leg-entry resets, or a proof-carrying `_counters` waiver), derived-deadline span checks (the stamped-absolute-deadline idiom), the dotted-reference name-rot audit, and the `_branch_order` requirement for overlapping guarded branches; committed oracles byte-match a fresh generation, matrices reconcile, named units covered. |
| Gd-idcite | designs with machines only: every stable-id citation in hand-written files resolves to a committed oracle row (letter-suffixed forms are falsifying-clause derivatives; `design/removed-ids.txt` is the dated allowance; DECISIONS.md and STATE.md are historical ledgers, counted not judged), positional `T-<TAG>-NN` citations warn, rule-15 form-b row counts are verified against their tables, and opt-in `CLAUSES{...}`/`READS{...}` declarations catch clause drift and payload-sufficiency drift (the `READS{...}` tier stands down for the events an armed event contract names: Gx-trace judges those at ERROR strength instead, so one defect earns one finding); a guard row whose contract statement is a conjunction or disjunction and declares NO clause vocabulary warns, waivable per row with `(single-clause: <reason>)`. |
| Gx-trace | cross-layer traceability: states to enum values, events to actions, invariants to enforcement rows, and entities to persistence-placement rows (every declared entity has a row or a `(not placed: <reason>)` waiver; the entity list is closed, so the table's completeness is checked, not attested). When the mitigation table carries a `handled by` column, every name resolves to a committed machine or one of its invoke actors, or waives with `(no residual: <reason>)`. The event-contract rows are reconciled against the machines with the pack gate's own semantics: a consumed event is handled or `_ignores`-ed by some machine, a produced event appears whole-token in some machine action or matrix cell, or the cell that owes the obligation carries a `(no machine: <reason>)` waiver. A design that carries a pack is left to G5, which reconciles the same rows from the generated `events.md`, so one defect never earns two findings. Opt-in by a `<!-- machinery:reads-complete -->` marker in ARCHITECTURE.md, the consumer-READS completeness tier: every event-contract row then owes a `READS{field, ...}` declaration on a matrix row naming its event, or a `(no reads: <reason>)` waiver in the consumer cell for a consumer that genuinely reads nothing off the payload (a pure signal), and every declared field must appear whole-token in that row's payload cell. A `(no machine: <reason>)` waiver answers a different question and never discharges it. An unarmed design carries no obligation and keeps Gd's opt-in warn tier exactly as it was. |
| Gb-plan | designs with a BUILD.md only: milestones are unique `**M<n> - <title>**` markers, the walking skeleton comes first (or carries an explicit waiver), every milestone has a `DoD:` line, the skeleton DoD cites a committed oracle id, and the skeleton block carries an `NFR:` line naming the NFR-record mechanisms it instantiates. |
| Ga-accept | once the build starts, milestone by milestone: committed acceptance evidence per closed milestone binds to the reviewed commit, either explicitly (`--commit`/`MACHINERY_COMMIT`, by identity) or derived from git HEAD of the repository holding the design (by ancestry, resolving an abbreviated sha to the one commit in this repository it names), and to the oracle ids its DoD cites. |
| Gl-ledger | always: every STATE.md `self-review:` line parses (five keys, `clean`/`fixed`/`fixed(<reason>)`/`accepted(<reason>)`), DECISIONS.md dated entries are real dates, author-proposed unconfirmed items are counted as a note, and the house-style scan warns on em dashes and emojis across the hand-written design tree. An em dash in a generated `*.modelith.md` render is an ERROR instead: the renderer emits them and the post-render strip is mechanical, so a survivor is a skipped step, not a style opinion. A `.machineryignore` at the design root (gitignore-shaped, patterns relative to the root, no negations) keeps paths that are not authored design (a spike's vendored dependencies) out of every design-tree walker; Gl's `checked:` line reports how many paths it ignored. |
| Gj-adjudication | designs with `adjudications/` only: every characterization verdict file parses, binds to a committed oracle stable id (one verdict per id), carries verdict/date/note, and every model-is-truth verdict names its filed defect. |
| Gv-attest | designs with `attestations.yaml` only: every attested-claim row resolves to the closed claim vocabulary (no id twice), names an attestor and a real date, and covers artifacts the design carries whose content hashes still match, so an artifact edited after the judgment makes the row STALE and blocks. A claim whose artifact exists but carries no row warns. Whether a judgment is true is never checked, by design. `machinery attest <path>` prints the hash the gate expects. |
| Ge-embed | designs carrying a `machinery:embed` marker only: every marked table is held to the source it declares (`subset`: each row is byte-identical to a source row; `complete`: every selected source row is present), with `(shard-local: <reason>)` exempting exactly the row or cell it names. A row copied twice into one embed warns (a copy carries the source's rows; neither claim catches a repeat on its own). The sanctioned shard-copy duplication, checked instead of promised. `machinery embed refresh <design>` writes what this gate checks: a keyed, idempotent re-copy that leaves localized rows alone and reports (never deletes) a row with no source row. |
| G4-import | code imports respect the contract boundaries (Go, Python, TypeScript/JavaScript, Elixir, Rust). |
| Gt-tests | with `--impl` only: every stable id in the committed oracles appears whole-token in a test file, or a test parses the committed oracle table (the conformance idiom); for every `CLAUSES{...}`-declared guard, each governed oracle row also owes one suffixed falsifying-clause id per active clause (the conformance parse never discharges those); the hard-TDD RED-exit check made deterministic. |
| G5-pack | decomposed designs only: packs fresh, children pinned to the current packs, refinement proofs fresh, and every boundary-event row held to its direction (`consumes` or `produces` exactly; any other value is an ERROR rather than a silently dropped row). |
| ERROR / DRIFT / WARN | ERROR is blocking; DRIFT means a generated artifact is stale, also blocking; WARN is advisory. |

## Use

In an agent session (Claude Code, Codex, OpenCode, or any runtime that loads Agent Skills), from the
project you want to design:

```
Design a new <system> with machinery.
```

With the Claude Code plugin, `/machinery:design <what you want>` does the same thing explicitly,
and `/machinery:status` reports where a design stands. The conductor takes it from Phase 0. It is fully standalone: no tracker, no project settings, no
other process dependencies. Target languages it realizes: Elixir, Go, Rust, TypeScript, Python.

## How it is put together

- `skills/machinery/SKILL.md` the conductor, plus `references/` (XState format, C4 technique, BUILD.md
  template) and `tools/` (the TLC shell wrappers, `tlc.sh` and `verify_formal.sh`).
- `cmd/machinery/` the single Go binary (cobra CLI): `lint`, `oracle`, `tla`, `alloy`, `refine`,
  `compose`, `check`, `attest`, `project`, `verify-checkers`, `baseline`, `verify-formal`, `verify-c4`,
  `pack`, `scale`, `sweep`, `doctor`, `preflight`, `install`, `update`, `uninstall`.
- `internal/` the Go toolchain: `ir/` (order-preserving machine model), `lint/`, `oracle/`, `tla/`,
  `alloy/` (the relational proof generators), `refine/`, `compose/`, `gates/` (the full gate suite,
  see the table below), `pack/` (recursive decomposition via contract packs), `formal/` (TLC + Alloy
  orchestration), `install/` (skill placement behind `machinery install`), `experiments/` (the
  shared mutation-experiment table). Every package has unit tests.
- `agents/` two synthesis subagents (the machine author and the build-doc writer).
- `commands/`, `hooks/`, `.claude-plugin/`, and `.codex-plugin/` the Claude Code and Codex plugin
  surfaces: slash commands where supported, the shared gate-enforcing hooks and shim, and both
  manifests. The repo root is the plugin; see the [Claude Code plugin guide](docs/claude-plugin.md)
  and [agent portability guide](docs/agent-portability.md).
- `adapters/opencode/` the OpenCode commands and thin JavaScript translation layer. The methodology
  and gate behavior remain in the shared skill and binary.
- `docs/` the deep dives: the [rebuild and hybrid guide](docs/rebuild-guide.md) (dual domain truths,
  migration contract, coexistence and rollback test protocol), the
  [policy layer guide](docs/policy-layer.md) (the annotation
  reference, the meta-checks in plain language, the authorization-oracle test pattern, greenfield
  and brownfield workflows), the [brownfield team guide](docs/brownfield-team-guide.md) (staged
  adoption ladder, baseline allow rules, PR discipline, CI recipes), the
  [Claude Code plugin guide](docs/claude-plugin.md) (hooks, `.machinery.json` reference, commands),
  the [agent portability guide](docs/agent-portability.md) (Claude Code, Codex, OpenCode, and
  capability fallbacks),
  the [external checkers guide](docs/external-checkers.md) (the pluggable-checker contract: the
  projection and evidence schemas, the tool-neutral manifest, the git-ignored resolution registry,
  the pure `gk` gate plus the `verify-checkers` engine phase, and how to adapt a signed-artifact or
  probabilistic engine you cannot modify),
  the [milestone acceptance guide](docs/acceptance-gate.md) (the closed marker, the acceptance
  evidence schema, the `ga` gate's DoD-id and commit bindings, and the CI recipe),
  the [attestation evidence guide](docs/attestation-evidence.md) (the `attestations.yaml` schema,
  the closed claim vocabulary, staleness by content hash, `machinery attest`, and how the `gv` gate
  relates to `ga` and to the superseded PR-checklist idea),
  and the [decision-lifecycle refinement pattern](docs/decision-lifecycle-pattern.md) (a draft
  rung-4 design note, not yet implemented).
- `examples/go-crm/` the worked rebuild example: `design/legacy/` (the working prototype truth),
  `design/migration.yaml` (the checked transition), the target blueprint/formal models, and `impl/`
  (the verified Go build).
- `examples/surreal-crm/` the store-swap rebuild example: the go-crm system moving from the
  embedded store to SurrealDB in Docker. Exercises Gs-surface (the legacy surface ledger,
  `design/legacy/surface.yaml`) and an all-reuse `migration.yaml`; its machines and oracles are
  byte-identical to go-crm's, which is the point: mitigations reclassify failures, the domain
  does not move.
- `examples/fulfillment/` the distributed stress test: `design/` only (six machines, eight proofs,
  and `FINDINGS.md`, the record of what strained and what was fixed).
- `examples/portfolio-engine/` a second design-only example in a different domain and language (a
  Python drawdown portfolio recommender): exercises the terminal-lifecycle pattern and a
  persistence overlay renamed from the defaults, proving the formal layer is not hardcoded to one
  vocabulary.
- `examples/checkout-split/` the recursive-decomposition example: a parent design that stops at
  contracts (`decomposition.yaml`, two abstract contract machines, generated `packs/`) plus two
  child designs, each a full machinery run against its frozen pack, with a TLC-checked proof that
  its machine refines the contract the neighbor relies on (G5-pack holds both sides: pack
  freshness, hash pinning in both directions, frozen public shape, boundary-event coverage, and
  refinement-proof freshness). Designs scale by verifying parts against contracts, never against
  the flattened system; this is that principle applied to the design process itself. When to
  escalate: `machinery scale` measures a design and recommends sharding or recursion.
- `examples/pii-flow/` the external-checker reference: a small but complete design (model with a
  `DataSubject` lifecycle machine, an Architecture Contract whose export-never-reaches-back claim
  is a checked `no_path` assertion) whose central invariant, no sensitive attribute reaches the
  export sink unredacted, is decided by a standard-library fixed-point checker in a digest-pinned
  OCI Python userspace under the Gk contract. The full default gate suite passes on it; the checker's coverage claim is the carrier
  Gc credits for the flow invariant, and its declared residuals carry the two process controls no
  static flow graph can decide.
- `testdata/golden/` the byte-for-byte golden corpus: expected stdout, stderr, exit code, and every
  generated artifact for the deterministic subcommands (lint, oracle, and tla on the four
  standalone examples, refine and compose where semantics annotations exist; check on every
  example design root; pack generate and scale on checkout-split),
  checked by `go test ./cmd/machinery -run TestGolden`; the Go experiment
  table lives in `internal/experiments/`.

See `skills/machinery/tools/README.md` for the checkers and generators, and
`examples/go-crm/design/formal/README.md` for the proofs. The skill also defines a revision mode
(design changes after code exists: stable test ids, oracle diffs as the affected-test list, and a
mandatory state-migration note for persisted machines) and a sharding rule for designs beyond
roughly ten stateful components.

## Testing & CI

The Go toolchain has full unit tests in every package, including `internal/formal`. The adversarial
mutation suite (every vacuity and drift finding from the design reviews, lint mutations plus the
full gate suite run against a synthesized design/impl fixture) runs as Go tests in
`internal/experiments`. Current coverage:

| Package | Coverage | Role |
|---------|----------|------|
| `internal/version` | 96% | version stamps and skew detection |
| `internal/tla` | 93% | TLA+ control-flow generator |
| `internal/hook` | 89% | progressive governance and generated-artifact protection |
| `internal/lint` | 89% | structural lint + matrix reconciliation |
| `internal/alloy` | 88% | policy, integrity, and isolation generators/oracles |
| `internal/oracle` | 87% | transition oracle (content-hashed ids) |
| `internal/refine` | 87% | data-refinement (3 patterns) |
| `internal/compose` | 86% | cross-aggregate composition |
| `internal/gates` | 75% | the full gate suite (G5 also exercised via `internal/experiments`) |
| `internal/install` | 75% | skill placement behind `machinery install` (fetch, extract, canonical+symlink layout) |
| `internal/pack` | 72% | contract packs (the mutation suite lives in `internal/experiments`) |
| `internal/ir` | 63% | shared IR (covered transitively via lint/gates) |
| `internal/formal` | 52% | TLC/Alloy orchestration (solver-run paths need Java) |
| **internal/ overall** | **79%** | own-package tests only; the cross-package adversarial suites in `internal/experiments` exercise gates and pack further (cmd/ is thin CLI plumbing) |

Run `go test -coverprofile=cover.out ./internal/... && go tool cover -func=cover.out` locally.
CI runs `go test -race ./...`. Beyond unit tests, three stronger nets are always green in CI:

- **Golden corpus**: `testdata/golden` byte-compares stdout, stderr, exit code, and every generated
  artifact for the deterministic subcommands: lint, oracle, and tla on the four standalone
  examples (go-crm, fulfillment, portfolio-engine, pii-flow), refine and compose where semantics
  annotations exist; check on every example design root (the checkout-split runs pin the G5-pack
  output, the pii-flow runs pin the full default suite and the hermetic Gk gate); and
  pack generate and scale on checkout-split (`make golden`;
  re-captured with `make golden-update` after intended output changes). Environment-dependent
  commands (verify-formal, doctor, preflight) are exercised by the formal-verification and CI jobs
  instead. The same byte corpus runs natively on Linux, macOS, and Windows.
- **Formal verification**: `machinery verify-formal` regenerates and TLC-model-checks all 35 TLA+
  proofs across the seven example designs that carry formal suites (8 in go-crm, 8 in surreal-crm,
  8 in fulfillment, 6 in portfolio-engine, 4 in checkout-split, two per child including the
  contract-refinement proofs, and 1 in pii-flow). Pii-flow's central sensitive-data invariant is
  still held separately by its external checker; its lifecycle machine supplies the control-flow
  proof. The required workflow also rejects any generated diff after the solver run. Strict manual
  pairs, when present, start with `\* machinery:manual`, require a sibling cfg, and are reported as
  declared/not-regenerated while TLC still checks them; an unmarked orphan pair or half fails.
- **Engine reproduction**: required CI installs Modelith v0.4.0 and reproduces every committed
  domain render after the renderer's mechanical house-style normalization; compiles every example
  `workspace.dsl` with checksum-pinned Structurizr CLI v2025.11.09; provisions the exact Python image
  digest for the registry-bound `linux/amd64` platform; verifies its local `RepoDigests` and
  OS/architecture; and runs the pii-flow adapter offline with the same `--platform` and
  `--pull=never` through the committed checker registry. These jobs are
  the engine halves; `machinery check` remains hermetic and dependency-free.

## Built on

machinery is a thin methodology over these external projects. It invokes or emits their notations and
bundles none of them.

- [Modelith](https://modelith.sh/) -- the domain-model language and linter (Phase 1).
- [C4 model](https://c4model.com/) -- the architecture technique (Phase 2).
- [Structurizr DSL](https://github.com/structurizr/dsl) and
  [Structurizr CLI](https://github.com/structurizr/cli) -- architecture-as-code, and optional C4
  diagram export.
- [XState](https://github.com/statelyai/xstate) and [Stately](https://stately.ai/) -- the
  state-machine JSON format (notation only; machinery does not run the library) and its visualizer.
- [TLA+ and TLC](https://github.com/tlaplus/tlaplus) -- the specification language and model checker
  (the behavioral formal layer).
- [Alloy](https://alloytools.org/) -- the relational model finder (the static relational layers;
  `machinery alloy` emits its notation and `verify-formal` runs the pinned analyzer).
- [Eclipse Temurin / Adoptium](https://adoptium.net/) -- the JVM that runs TLC.
- [Go](https://go.dev/) -- to build machinery from source and to install Modelith.
- [LadybugDB](https://github.com/LadybugDB/go-ladybug) -- the embedded store used only by the go-crm
  example, not a machinery dependency.

## License

Copyright 2026 Ramiro Salas. Licensed under the Apache License 2.0; see `LICENSE`. 
`machinery` invokes `modelith` and emits XState and C4 notation; it bundles none of them, so no dependency's license binds
it. The tools it works with are permissively licensed and compatible with Apache-2.0: Modelith and Structurizr are Apache-2.0, XState and LadybugDB are MIT, and C4 is an open notation.
