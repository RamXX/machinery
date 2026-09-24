# machinery

[![CI](https://github.com/RamXX/machinery/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/RamXX/machinery/actions/workflows/ci.yml)
[![Formal Verification](https://github.com/RamXX/machinery/actions/workflows/formal.yml/badge.svg?branch=main)](https://github.com/RamXX/machinery/actions/workflows/formal.yml)
[![Security](https://github.com/RamXX/machinery/actions/workflows/security.yml/badge.svg?branch=main)](https://github.com/RamXX/machinery/actions/workflows/security.yml)
[![Nightly](https://github.com/RamXX/machinery/actions/workflows/nightly.yml/badge.svg?branch=main)](https://github.com/RamXX/machinery/actions/workflows/nightly.yml)
[![CI: Dagger](https://img.shields.io/badge/CI-Dagger%20v0.21.9-131226)](.dagger/main.go)
[![Go Reference](https://pkg.go.dev/badge/github.com/RamXX/machinery.svg)](https://pkg.go.dev/github.com/RamXX/machinery)
[![Go Report Card](https://goreportcard.com/badge/github.com/RamXX/machinery)](https://goreportcard.com/report/github.com/RamXX/machinery)

machinery is a design methodology and a toolchain for building software with AI coding agents.
You design the system once, as a domain model, an architecture, and a set of state machines. The
tests, the architecture contracts, the build instructions, and machine-checked proofs are then
generated from that design or checked against it. The result is a blueprint that a coding agent
with no prior context can implement under hard TDD, and a gate suite that keeps the code and the
design from drifting apart afterwards.

## The problem

AI coding agents make writing software cheap. They do not make writing large software safe.

On a small program, an agent's mistakes are visible: the program runs or it does not. Past a few
thousand lines, mistakes stop being loud. A rule stated in one file is broken three files away. The
design says one thing and the code does another, and nothing notices, because nothing compares
them. A failure mode that nobody listed shows up in production, and the postmortem finds that every
individual change looked reasonable.

Consider one sentence from the guard-contract table of a design document (an illustration):

> `canPay`: an order may be paid only while it is unpaid and the captured amount matches the total.

One agent implements the amount check. Another writes the test, which exercises the amount check.
Both read the same sentence and each took one half of it. The review passes, because the reviewer
reads the sentence, sees an amount check and a test for it, and the sentence "sounds covered." The
order can now be paid twice.

The usual answer is "review it carefully." That answer does not scale, for three reasons:

- **Review reads prose, and prose has no closed meaning.** Every reader, human or model, fills the
  gaps slightly differently. Two agents working from the same paragraph can build two different
  systems and both be right about what the paragraph said.
- **The defects that matter are relational.** They live between artifacts: the enum in the domain
  model against the states in the machine, the fields an event carries in one table against another
  table, the rows in an authorization list against the actions the system performs on its own. A
  reviewer looking at one file at a time cannot see them, and there are too many pairs to check by
  eye.
- **Agents change things faster than people can re-review.** A review is a judgment about a
  snapshot. Ten edits later, nobody knows whether it still holds.

"Review it carefully" really means "trust the model, and trust the reviewer." Trust does not scale.

## The stance: construct and check

machinery treats correctness as something you construct and check, never something you hope for.

The design is the single source of truth. Everything downstream is one of two things:

- **Generated from the design**, so it cannot drift. The transition oracles (the tables of expected
  behavior that tests key on), the TLA+ specifications, the Alloy models, the authorization decision
  tables, the checker projections, and the executor packets are all produced by the `machinery`
  binary. Committed copies are byte-compared against a fresh generation on every check, and a stale
  one is reported as DRIFT.
- **Checked against the design** by a deterministic tool or an exhaustive proof, so a mistake is
  caught instead of shipped. `machinery check` runs with no model in the loop. TLC and the Alloy
  analyzer search every state or configuration within a bound and return a concrete
  counterexample when something is wrong.

The model (the LLM) does the creative work: interrogating the product owner, naming things,
proposing invariants and failure postures. The machine holds the line.

Why keep the model out of the gates? A gate is only useful if the same bytes always get the same
verdict. A model asked "does this design satisfy its invariants?" can answer differently on two
runs, can be talked out of a finding, and cannot show its work in a form anyone can re-check. A
deterministic check gives the same answer every time and says exactly what it counted. That is
also why generation beats co-authoring: an artifact derived from the design by a program cannot
disagree with the design, while a hand-maintained copy starts drifting on the first edit.

Two rules hold this stance in place.

**Every gate has two halves, and they stay separate.** The deterministic half is what the tool
verifies: the contract parses, the oracle is fresh, every id resolves, every hash binds. The
attested half is what a reviewer judges: whether a guard enforces the invariant it names, whether an
interface contract has the right shape, whether the blueprint is buildable with zero context. The
tool never pretends to check the second half. It records who judged it, against which bytes, and
reports the judgment stale the moment those bytes change (see Gv-attest below).

**A gate cannot pass on absence.** Every gate prints a `checked:` line with the exact counts of what
it verified. A gate that finds nothing to check fails instead of passing, so an empty directory, a
misnamed file, or a parser that silently matched nothing does not read as green.

## The thesis: most software is a state machine

Most software, where it matters, is a state machine. An order is placed, paid, shipped, delivered,
or refunded. A saga reserves, charges, ships, and compensates. A background job is queued, running,
retrying, or dead. Make those machines explicit and most of the rest follows.

machinery splits a design into three layers:

- **The what: a domain model** in [Modelith](https://modelith.sh/). Entities, relationships,
  lifecycle enums, actions with their actors, and the invariants that must always hold. Linted.
- **The how: an architecture** in [C4](https://c4model.com/), written as Structurizr DSL plus an
  Architecture Contract. Components, allowed and denied dependencies, interface contracts, the event
  contract, and what every dependency does when it fails. Contract-checked.
- **The behavior: state machines** in the [XState](https://github.com/statelyai/xstate) v5 JSON
  format. Every state, transition, guard, timeout, and failure path, conditioned on the architecture
  the previous layer fixed. Model-checked.

The layers compose; they do not merely sit side by side. A lifecycle enum in the domain model is
the state set of a machine. An action in the model is an event and a transition. An invariant is a
guard, or an edge that structurally cannot exist. A side effect is an `invoke` with a
success path, a failure path, and a timeout, and the dependency it calls has a failure posture in the
architecture. The gates check those links by construction, so a state with no enum value, an event
with no action, or an invariant with no enforcement row is a finding.

The machines come last because they need the other two as inputs. About half of each machine is
derived from the domain model rather than invented, which is why a mid-tier model can safely write
machines once a strong one has fixed the domain (see [Which model to use where](#which-model-to-use-where)).

A narrow design ends in one self-contained `BUILD.md`. A large design ends in a root milestone and
demo manifest whose linkage is either pairwise (one bounded, self-contained execution packet per
milestone) or matrix (reciprocal many-to-many links between milestones and reusable domain shards).
Either way the machines remain both the test oracle and the formal specification.

### About the XState reference

machinery uses the XState config format as a linted notation and does not run the XState library.
The lint is machinery's own (a deliberately narrowed subset, plus annotations such as `_role` and
`_exhaustive` that are not XState), and so are the oracle generator and the model checking. The
de-annotated config loads into [Stately](https://stately.ai/) and `@xstate/graph` for optional
visualization and covering-path generation. A TypeScript build may adopt XState directly; Go, Rust,
Python, and Elixir targets hand-roll the state field. The guarantees come from machinery's tooling
and from [TLC](https://github.com/tlaplus/tlaplus), the model checker for
[TLA+](https://lamport.azurewebsites.net/tla/tla.html) (Leslie Lamport's specification language for
state-transition systems). They never come from XState.

### Why these notations

Each notation was chosen because it is text, can be generated, and can be checked by a machine.
Familiarity was not the criterion.

- **Modelith, over UML class diagrams or prose.** It carries entity lifecycles and invariants as
  lintable data, which is what lets half of each machine be derived. A diagram or a prose spec
  carries none of that as checkable structure.
- **C4, over UML component diagrams or arc42.** Its Structurizr DSL is text a gate can parse and
  bind, and it centers components and their dependencies, which is where each failure posture
  lives. UML is diagram-first and heavier; arc42 is a documentation template, not a checkable model.
- **XState v5 JSON, over SCXML or a bespoke DSL.** JSON is easy to generate and to diff byte for
  byte, the v5 shape already has hierarchy, guards, `invoke`, and delays, and Stately and
  `@xstate/graph` provide visualization and covering paths. SCXML is XML with weaker tooling, and a
  bespoke DSL means owning a parser and a visualizer forever.
- **TLA+ and TLC for behavior, over property-based testing.** TLC checks a finite instance
  exhaustively for termination, deadlock-freedom, safety, and liveness, and returns a concrete
  counterexample. Property-based tests sample the state space; they do not exhaust it.
- **Alloy for static relations, where TLC never looks.** Some invariants are relations over
  configurations, not steps over time. No machine enforces them, so TLC cannot check them, and a
  linter sees only well-formed prose. Bounded relational search is what Alloy does. Nobody
  hand-writes the Alloy, just as nobody hand-writes the TLA+.
- **Datalog for consistency between artifacts.** Whether two tables agree, whether every autonomous
  write has an authorization row, whether every fact a contract names resolves: these are joins
  over declared facts. They are written as rules and run under two engines (see
  [the rules gate](#where-the-design-must-agree-with-itself-the-rules-gate)).

## What each layer catches

A layer earns its place by catching a defect nothing else would. Except for the `canPay`
illustration carried over from above, the scenes below come from the bundled examples and from this
project's history, and each ends with the mechanism that now catches it.

### The domain model: an enum that drifted from its machine

In the order-fulfillment example, the domain model's saga status said `Running` while the saga
machine had split that into `Reserving`, `Paying`, and `Shipping`. Worse, the machine had a
`FailedDirty` residual state (compensation retries exhausted, a person must intervene) that the
enum did not list at all. Code generated from the enum would have had no way to represent a saga
stuck half-compensated.

Caught by **Gx-trace**, the cross-layer traceability gate: every machine state must map to an enum
value and every event to an action. The same gate demanded the five lifecycle machines the design
had not yet written.

### The machines: a refund that could leave a customer charged

The same example's saga, as first drawn, attempted a refund once during compensation. If that
single attempt failed, the saga moved on: the payment stayed captured and nothing was returned.
Every state looked reasonable in isolation.

Caught by **TLC**, run by `machinery verify-formal` over a data-refined model generated from a
six-line semantics annotation. It produced the exact six-step counterexample. The fix, one
idempotent retried compensation step with an explicit `FailedDirty` residual when retries run out,
is now proven: money and stock are never silently lost, with compensation modeled per obligation so
a partial compensation is a real, checked state. **G3-machine** holds the structural side on every
check: every `invoke` has an error path and a timeout, every retry loop has its own bounded
counter, and every fully guarded handler states what happens when the guard is false.

### The guard written in prose: an order paid twice

Back to the `canPay` sentence from [The problem](#the-problem). The defect was that one sentence
held two conditions and nothing forced anyone to test both. machinery makes the clauses data. The
guard's row in the named-unit matrix declares them:

```
| `canPay` | guard | ... | CLAUSES{unpaid, amountMatches} | ...
```

Now every oracle row the guard governs owes one suffixed falsifying-clause test id per clause, and
**Gt-tests** reports each one no test names. A suite that only ever falsifies `amountMatches`
leaves the `unpaid` id uncovered, by name. **Gd-idcite** warns on a guard row whose contract is a
conjunction and declares no clause vocabulary, so the sentence cannot stay a sentence by accident.

### Access control: a manager who provably cannot write their own records

The go-crm example shipped with role- and ownership-based access rules that had survived prose
review, every gate, eight TLC proofs, and a full TDD build. They still held two defects. A Manager
without a team could not write even the records it owned, because write scope was granted by team
membership and nothing required a Manager to have a team. And a Manager could reassign a record to
someone outside its own authority in one legal step.

Caught by the **policy layer**. One short annotation, `design/formal/policy.relational.yaml`,
states who acts (the role enum), what they act on (the owned entities), and per-verb scopes from a
four-word algebra (`all | own | team | none`). `machinery alloy` compiles it with the domain model
into `Policy.als`, whose generated meta-checks include `CapableWritesOwn` and
`ReassignRetainsAuthority`. The Alloy analyzer answered with a counterexample in domain vocabulary:

```
FAIL  Policy/CapableWritesOwn
      counterexample: User$5{role=Manager$0, team=(none)} Record$0{owner=User$5}
```

The same annotation compiles to `Policy.oracle.md`, a 70-row authorization decision table that the
implementation's test suite asserts case by case. **Gp-policy** fails the build when the annotation
stops matching the model, when a cross-cutting invariant is neither compiled nor waived, or when a
committed artifact is stale. The full guide is [docs/policy-layer.md](docs/policy-layer.md).

### Structure: two orders sharing one payment

Fulfillment's model said each order has one payment. Forward field multiplicity alone does not say
that a payment belongs to at most one order, so two orders could share one.

Caught by the **integrity layer** (`integrity.relational.yaml`, compiled to `Integrity.als`, held by
**Gi-integrity**). It carries the inverse side of a `1:1` relationship as a fact of the model, and it
checks uniqueness, singletons ("exactly one pipeline is the default"), and mandatory relationships
for joint satisfiability. Two constraints that are each well-formed but cannot both hold turn the
model red, which the linter cannot see. See [docs/integrity-layer.md](docs/integrity-layer.md).

### Tenancy: a task that leaks a deal

A sales rep may read their own task. The task links to a deal. If nothing forbids it, the deal
belongs to another team, and reading the task leaks it. Access rules never see this, because the
leak is in the reference graph and no single record's ownership is wrong.

Caught by the **isolation layer** (`isolation.relational.yaml`, compiled to `Isolation.als` and a
tenant-scoping decision table `Isolation.oracle.md`, held by **Gn-isolation**). It proves, among
other things, that two records whose reference fields share even one target are owned in the same
tenant. See [docs/isolation-layer.md](docs/isolation-layer.md).

Policy, integrity, and isolation are the three opt-in Alloy algebras. Each is generated from the
domain model plus a short annotation, each is solver-checked by `verify-formal`, and each costs
nothing where it does not apply: a design without the annotation never runs any of it. Together
with the consistency rules below they are machinery's four relational layers: three answer
questions about what configurations are legal, and one answers whether the design's own statements
agree.

### The architecture: an edge nobody allowed

An agent adds an import from the web handler straight into the payments database adapter because
it was the shortest path. Each file still compiles.

Caught by **G2-c4** at design time and **G4-import** at code time. G2 parses the Architecture
Contract, binds it to `workspace.dsl`, requires an acyclic allow graph, a failure posture (mitigation
row) for every dependency, an interface contract for every allowed crossing, and an NFR record
covering security, capacity, and observability, and it judges every relationship the diagram draws
by the same allow, deny, and baseline rules. G4 then reads the implementation's import statements
(Go, Python, TypeScript/JavaScript, Elixir, Rust) and judges every edge by those rules.

## The correctness ladder

The scenes above use four kinds of check, and they differ in strength. machinery pushes each layer
as far up this ladder as it can go.

1. **Generate, do not co-author.** Anything derivable is generated from the design sources: the
   transition oracles (with content-derived stable ids that survive design revisions), the TLA+
   specs, and, where an annotation exists, the Alloy models and decision oracles. G3 byte-diffs
   every committed oracle against a fresh generation on every check. The pinned Modelith engine
   regenerates every committed domain render in required CI, followed by the mechanical em-dash
   normalization and a byte-diff. `verify-formal` regenerates the formal specs from source, and the
   required formal workflow and the nightly clean-tree job both assert that the committed copies
   match. Staleness is caught as drift, never assumed away.
2. **Deterministic symbolic gates that cannot pass on absence.** With no model in the loop,
   `machinery check` verifies that machines are well-formed (reachability, unambiguous targets, no
   dead ends, an error path and a timeout on every side effect, every event handled or explicitly
   ignored in every resting state), that the architecture contract binds to the C4 model and every
   dependency has a failure posture, that the layers trace to each other, that the design's
   declared facts agree (the rules gate, below), and that the code respects the boundaries.
3. **Model checking.** Each machine is finite, so TLC checks it exhaustively: retry loops bounded
   (each loop with its own counter), every operation terminates, nothing gets stuck half-done, no
   deadlock. Every generated spec states its assumptions in its header (guards erased soundly for
   safety, liveness conditional on guard exhaustiveness that the linter discharges, single instance,
   no data at this rung), so a green check reads as exactly what it is. The same rung extends
   sideways into static relations through the three Alloy layers.
4. **Refinement and assume-guarantee.** Data and composition annotations are reconciled against the
   machines before anything is emitted, so a drifted annotation fails generation instead of proving
   a stale twin. Each subsystem is proven to refine the small contract its neighbors rely on; the
   composition instances that same contract module, and TLC checks that the composition satisfies
   it. Parts are verified against contracts, never against the flattened system, which is how this
   scales past one machine.

Rungs 1 through 3 are generated from the design automatically; rungs 2 and 4 also use short
declarative annotations that the generators verify against the machines. The toolchain is held the
same way: a Go test suite encodes every vacuity and drift attack from adversarial design reviews as
a permanent regression, and CI runs the tests, the gates, the proofs, and the example builds on
every push.

## Where the design must agree with itself: the rules gate

The layers above each check one kind of artifact against its own rules. The hardest remaining
defects are statements in two places that must agree and do not. Four illustrations, in the
fulfillment example's vocabulary:

- A payment matrix row says the `order.paid` event carries four fields, `{orderId, amount,
  currency, paidAt}`. The Architecture Contract's event table says it carries three. The consumer
  built from one and the producer built from the other disagree at runtime.
- The domain model declares `Order.markPaid` with actor `System`: the system performs it on its
  own, with no person behind it. Nobody wrote a row in the authorization inventory saying which
  component is allowed to do that. The system performs a write no one ever authorized.
- A matrix row says a unit writes `Order.paidTwice`, an attribute the model does not have.
- An actor unit calls an external service and nothing says what carries its effect: a column, an
  outbox event, a sink, a signal.

### The first attempt, and why it was removed

Release 0.9.0 caught several of these by reading prose. The checks decided whether a matrix cell
was a write from phrases such as "writes nothing" or "never adjusts the row", whether a cell named
a closed vocabulary from words such as "reason class", and whether a backticked token was a fact
from the words around it. They carried 35 compiled regular expressions. A design that phrased the
same contract differently got a different verdict, and a `System` action whose description said
"writes nothing" was treated as read-only and owed no authorization row. Heuristics over prose are
exactly the "trust the reader" problem this project exists to remove, so they were deleted
(2,639 lines, with their tests) and replaced with the consistency layer, which rests on four ideas.
The consistency layer ships in release 0.10.0; [CHANGELOG.md](CHANGELOG.md) records what it
adds and the migration a 0.9.0 design needs.

### 1. Declarations are grammar

A design states each fact the gates act on in a declaration with a closed grammar: an upper-case
group name followed by a braced, comma-separated list, placed in one table cell.

- `WRITES{Order.status, OutboxMessage.status}`: the stored facts a unit writes. `WRITES{}` alone
  states a read-only unit.
- `USES{Order.totalCents, LineItem.quantity}`: the facts a unit reads or names.
- `PRODUCES{Order.markPaid}`: on the row that names a cascade or consumer arm, the model actions
  that arm performs. Each produced action owes an authorization row whatever its actor.
- `CARRIES{column:Order.status, outbox:OutboxMessage}`: what carries an effect. Kinds are
  `column`, `outbox`, `sink`, `signal`, and `action`.
- `VALUES{a, b, c}` or `VALUES OrderState{...}`: a closed vocabulary, bound to a Modelith enum by
  exact name.
- `payload {orderId, amount}`: an event payload twin on a matrix event row.
- `derived: fact_name (<reason>)`: a row-local waiver for a computed fact that is never stored.
- `CLAUSES{...}` (with `RETIRED{...}`): a guard's falsifying clauses. `READS{field, ...}`: the
  payload fields one consumer reads. `ORACLESET{...}`: a named set of oracle ids a milestone cites.
- `SUPERSEDES{type:LegacyDeal}` and `RESERVED{type:ReadbackReceipt}`: on Architecture Contract
  rows only, a type this row replaces, or reserves as not yet defined.
- The authorization inventory: a table in `AUTHORIZATION.md` carrying the marker
  `<!-- machinery:authorization-inventory -->`, one row per autonomous action, admitting it with a
  backticked capability declared in `workspace.dsl` or waiving it with
  `(no authorization: <reason>)`.

Two rules close the grammar. An upper-case `NAME{` in a matrix cell that is not one of `CLAUSES`,
`RETIRED`, `READS`, `VALUES`, `ORACLESET`, `WRITES`, `USES`, `PRODUCES`, `CARRIES`, `SUPERSEDES`,
or `RESERVED` is an error, so a misspelled group never passes as prose. A design whose own tooling
reads group marks of its own keeps them by listing each name once in the Architecture Contract,
`private_groups: [OWNED-BY]`; machinery then skips those groups (no error, no fact, no obligation
met), while any unlisted name still fails, and listing a public name is a G2 error. And a
backticked snake_case or `Entity.attr` token in a contract, clause, or payload cell that no group
on its row declares is resolved first: an action, named unit, enum value, context key, event,
invariant id, or file name is silent; a model attribute is a Gl-ledger warning (declare it in
`USES{}` or `WRITES{}`); a token that names nothing gets a softer warning (drop the backticks or
declare it). A matrix with more than 20 such warnings prints one summary line; `--verbose` lists
each. **Prose never declares.** A sentence may quote a fact; it cannot create one.

### 2. Facts are projected once

`machinery project` is the single reader of the design. It turns the model, the machines, the
matrices, the event contract, the C4 model, the authorization inventory, the oracles, the milestones,
and the supersession rows into relations (projection schema 2.0), each row with a stable id and its
source `path:line`. Only identifiers and enumerated values are projected, never prose.
`machinery project <design> --facts <dir>` writes those relations as tab-separated `.facts` files,
the input format Soufflé and the in-process evaluator read directly, plus a `relations.txt` index.
The same facts feed the rules gate and any external checker, so every gate sees one reading of the
design.

### 3. Rules are data, and two engines must agree

The methodology rules are Datalog files under [`rules/consistency/`](rules/README.md), embedded in
the binary at build time. There are eight: `authz.dl`, `bindings.dl`, `carriers.dl`, `facts.dl`,
`payload.dl`, `records.dl`, `supersession.dl`, and `values.dl`, 326 lines in all. Each output
relation named `finding_<code>` is an ERROR with that code. A rule reads like the obligation it
enforces:

```
finding_authz_missing(A) :- system_write(A), !admitted(A), !no_authorization(A).
```

Gy-rules evaluates them with `internal/datalog`, an in-process semi-naive evaluator with stratified
negation, written against the Go standard library only. `machinery check` therefore stays hermetic:
no Docker, no solver, and it runs in the stop hook. The rule language is a subset that real
[Soufflé](https://souffle-lang.github.io/) accepts unchanged, and parity is tested, not assumed:

- `TestRulesParity` runs every rule file over every bundled example design plus one synthetic design
  on which every output relation has tuples: 8 rule files by 9 designs, 72 runs, and each output
  relation must be identical under both engines (rows are sorted before comparing, because Soufflé
  fixes no row order).
- `TestParity` in `internal/datalog` runs the evaluator's own corpus of 19 programs (transitive
  closure, stratified negation, aggregates, and the programs both engines must reject, such as a
  cycle through negation) under both engines.
- The `datalog-parity` CI job runs both in an image built from pinned inputs
  (`scripts/souffle.dockerfile`), and fails if a test skipped its Soufflé half.

A new consistency check is a rule file plus a fixture that fails without it and a near neighbor
that must not fire. It needs no new Go.

### 4. Every finding says why

`machinery check <design> --gate gy --explain` prints, under each finding, the derivation that
produced it: the rule file and rule number, then every fact it matched with its source. Remove the
`Order.markPaid` row from the fulfillment example's `AUTHORIZATION.md`, add a member the model does
not declare to a `WRITES{}` group, and the gate prints:

```
== Gy-rules  consistency rules over projected facts ==
  ERROR  fulfillment.modelith.yaml:127: row 'Order.markPaid': authz_missing
         finding_authz_missing("Order.markPaid")  [authz.dl rule 7]
           system_write("Order.markPaid")  [authz.dl rule 4]
             action("Order.markPaid", "Order", "markPaid", "System")  [fact fulfillment.modelith.yaml:127]
  ERROR  machines/Order.matrix.md:12: row 'Order.persistOrder': fact_unresolved (fact 'Order.paidTwice')
         finding_fact_unresolved("Order.persistOrder", "Order.paidTwice")  [facts.dl rule 9]
           named_fact("Order.persistOrder", "Order.paidTwice")  [facts.dl rule 8]
             unit_writes("Order.persistOrder", "Order.paidTwice")  [fact machines/Order.matrix.md:12]
  checked: 8 rule files evaluated, 1269 facts

2 blocking (ERROR/DRIFT) finding(s)
```

You can read that trace without knowing Datalog. The action is declared at line 127 with actor
`System`. Rule 4 makes it a system write, and rule 7 fires because nothing admits it.

### What the rules catch, file by file

- `authz.dl`: a `System` or produced action with no admission (`authz_missing`), a stale row
  (`authz_orphan`), a capability no `workspace.dsl` element declares
  (`authz_unknown_capability`), and a `PRODUCES{}` member the model does not declare
  (`produces_unknown_action`).
- `facts.dl`: a `USES{}` or `WRITES{}` member that resolves to no attribute, enum member, machine
  context key, payload field, or same-row `derived:` or `VALUES` declaration (`fact_unresolved`).
- `values.dl`: a `VALUES` group that disagrees with its enum (`values_disagree`), and two units
  spelling one shared vocabulary differently (`values_conflict`).
- `payload.dl`: a `payload {}` twin that is not equal to the payload of an event-contract edge its
  unit's component produces or consumes (`payload_twin`, naming the edge), a unit whose component
  is on no edge of the event (`payload_no_edge`), and a payload for an event with no contract row
  (`payload_unknown_event`). This is the four-fields-against-three scene. The component comes from
  the ARCHITECTURE.md action-ownership table; a unit with no owner is held to every edge.
- `carriers.dl`: an actor, or an action with a non-empty `WRITES{}`, that names no `CARRIES{}`
  (`effect_uncarried`), and a `CARRIES{}` on a unit that is neither (`carrier_misplaced`).
- `supersession.dl`: two owners of one type, supersession cycles, a replacement whose old type
  nobody owns, a `RESERVED` type some artifact already owns, and an executor packet that cites the
  row of a superseded type.
- `records.dl`: a matrix with neither a machine nor a `(no machine: <reason>)` placement waiver
  (`orphan_matrix`), and a waiver on a component that has a machine. This is how an append-only,
  contract-only record is declared.
- `bindings.dl`: under `--impl` only, a BUILD.md oracle binding table that says an oracle id is
  unbound while a test binds it (`milestone_binding_stale`), or names a test file that binds
  nothing for it (`milestone_binding_phantom`).

What a green Gy-rules proves is stated in its release notes: over declared facts only, every
`System` and produced action has one admission naming a declared capability, every declared fact
resolves, every enum-bound `VALUES` group equals its enum, every `payload {}` twin equals each
contract edge its unit's component takes part in, every effect names a carrier, and type
supersession is acyclic with owned replacements. It says nothing about facts a design only
mentions in prose. A design Gy-rules cannot project whole is red: each row it cannot read is an
ERROR, the rules run on the rest, and every finding printed beside such an ERROR is marked
`[projection partial: ...]`, because it may follow from the omitted row.

A design written for 0.9.0 migrates by declaring what the prose used to imply; the skill's
"Migrating from the 0.9.0 prose inference" section walks through each case. For one release,
Gx-trace warns on a design that declares no `WRITES{}`, `USES{}`, or `PRODUCES{}` anywhere and quotes
a fact-shaped token in prose.

### Bring your own checker

Some invariants need domain knowledge machinery does not have: a sensitive-data-flow property, a
statute encoded as rules, a units-of-measure check. The external checker contract lets any
deterministic analyzer gate a design with the same discipline as the built-in gates. You commit a
tool-neutral manifest (`design/checkers/<id>.checker.yaml`) naming the projection slice you need and
the invariants you claim, and the evidence your engine produced. **Gk-\<id\>** then checks, with no
engine and no network, that the committed projection is fresh, that the evidence binds to it by
`input_hash` and to the manifest's `runtime_closure`, that the verdict is pass, and that every
claimed invariant is covered or declared a residual with a reason. `machinery verify-checkers`
is the engine half: it re-runs your adapter from an immutable, digest-pinned OCI image and confirms
the committed evidence reproduces. The guide is [docs/external-checkers.md](docs/external-checkers.md).

The reference is [`examples/pii-flow`](examples/pii-flow/README.md). Its central invariant, no
sensitive attribute reaches the export sink unredacted, is a Datalog program, `rules.dl`, that
Soufflé 2.5 evaluates inside a digest-pinned `linux/amd64` image. The Python adapter beside it only
translates between machinery's JSON contracts and Soufflé's files; it computes no part of the
verdict, and it fails closed (writing no evidence) on any engine error, any unexpected output, or a
projection other than the one its manifest asked for.

## The envelope for agentic systems

"Most software is a state machine" is a claim about control flow, not about cognition, and agentic
programs are where the difference shows. An agent is a non-deterministic policy (an LLM choosing
which tool to call) running inside a deterministic envelope: the perceive-act-observe loop, tool
execution, budgets, guardrails, approval gates, side-effect compensation, and sub-agent
orchestration. The envelope is the state machine. The policy is not, and machinery does not pretend
otherwise.

What machinery models is the containment:

- Every tool is an `invoke` with an enumerated failure set, a timeout, and an idempotency key, so the
  action space is bounded even when the choice within it is not.
- The loop is a machine whose termination and budget are model-checked, so it provably stops.
- Every irreversible action is a saga with compensation and an explicit stalled-dirty residual.
- Every guardrail is a guard tied to an invariant.
- Sub-agents coordinate through the event-contract table.

The LLM decision itself is a contracted, non-deterministic oracle that machinery fences but never
tries to prove. You get liveness (it stops, within budget, and cleans up) without usefulness (that
it chose well). So keep the action space enumerated and the high-stakes guardrails deterministic:
that is the region machinery can prove, and whatever you leave to the model's discretion is a region
it cannot.

machinery is built this way itself: a gated pipeline around a non-deterministic conductor.

## Holding the code to the design

A blueprint is only useful if the code stays bound to it. These gates exist because each of these
things happened, or nearly did.

### A test suite that looks complete and runs nothing

A suite names every oracle id, but some of those tests are commented out or disabled, some are
declared and never called, and some ids appear only in a comment outside any test body. A coverage count based on the names
alone reports full coverage.

**Gt-tests** (with `--impl`) requires every stable id in the committed oracles to appear as a whole
token in an active test body, or a test that parses the committed oracle table in one connected
read-parse-assert flow. Commented-out and disabled tests, unused declarations, and uncalled helpers
earn nothing. Gt labels itself "static discovery; tests not executed", because it executes nothing:
whether the tests pass is your test runner's job, and machinery says so instead of implying more.

### A design that stays green while the compiler that parses it breaks

An implementation parses a design file at build time: a compile-time macro reads a matrix and
rejects tokens it does not know. Someone adds a token to the matrix. Every design gate is green,
G4 and Gt are green, and the next build fails.

**Gr-reads** makes that coupling explicit. A root `reads:` row in the Architecture Contract names
the design artifact, the implementation reader, and the full reviewed Git commit. Without `--impl`
each row warns that the reader must land with its follow-up. With `--impl`, the gate walks every
commit since the reviewed one, and any commit (or uncommitted change set) that changes the artifact
without changing its reader is an ERROR. See [docs/declared-reads.md](docs/declared-reads.md).

### A milestone closed by assertion

"M3 is done" said in a chat message, checked by nobody. **Ga-accept** requires committed evidence,
`design/acceptance/M<n>.yaml`, for every milestone marked `Status: closed`, bound to a reviewed
commit in the repository history and to the literal or `ORACLESET{...}`-expanded oracle ids its
definition of done cites. See [docs/acceptance-gate.md](docs/acceptance-gate.md).

### A review that went stale

Someone judged that the interface contracts were right. Two weeks later an agent edited
`ARCHITECTURE.md`. The judgment is now about a file that no longer exists.

**Gv-attest** records the attested halves in `design/attestations.yaml`: a claim from a closed
vocabulary, an attestor, a date, and the covered artifacts with their SHA-256 content hashes. An
edited artifact makes the row stale and blocks. Each row carries a kind: `plan` (a design-time
judgment), `current` (a reviewed implementation, bound to a complete implementation manifest and
invalidated when its bytes move), or `historical` (an acceptance-file judgment, which establishes
history and never current approval). A design with no implementation yet is plan-only: a claim
that owes a current review warns that the review is missing and the check still exits 0. The
pii-flow example ships no implementation, so it carries exactly that one warning by design:

```
== Gv-attest  attestation evidence ==
  warn   gt.conformance-test-shape: plan only; current implementation review missing
```

Whether a judgment is true is never checked; that is the point of the split. `machinery attest`
prints the hashes the gate expects. See [docs/attestation-evidence.md](docs/attestation-evidence.md).

### An agent that ends its turn with the design red

Inside a Claude Code or Codex session with the repository plugin installed, hooks make the
deterministic half of the gates part of the session itself. A turn that touched the design, or the
watched implementation, cannot end until `machinery check` has run. DRIFT blocks the stop with
the gate output as the reason, and the model fixes it and tries again. Import-boundary findings
block once `design/ratchet.json` arms them. Plain ERRORs only warn, because a half-built design is
a normal state mid-interrogation. The hook refuses to discharge while any tool call it saw start has
not reported an end, so a shell command that is still writing cannot slip past an unchanged tree.
OpenCode's adapter runs the same check when the session goes idle, but its event API cannot hold a
turn open, so a red result there is a warning. CI's `machinery check` remains the merge authority
on every host.

### An executor that cannot hold the whole design

A large milestone plus its shard plus the architecture can exceed the executor's context window.
Splitting the sources by hand loses obligations silently. Instead you author `design/slices.yaml`,
and `machinery packet <design> --milestone M1 --out <dir>` projects one packet per slice containing
only what the slice cites, each excerpt verbatim under its stable id and source `path:line`.
**Gw-packet** holds the map: every citation resolves, every packet fits its declared byte budget,
every shared fixture names all of its consuming slices, and every obligation the milestone owes is
claimed by exactly one slice or carries a recorded waiver. See
[docs/packet-projection.md](docs/packet-projection.md).

## The pipeline

The conductor (the machinery skill, running in your agent) takes a design through these phases. It
is an interrogation, not a form: it pushes on naming and on "what must always be true," and it does
not advance a phase until that phase's gate passes. Each line under a phase lists the deterministic
half (`tool:`) and the attested half (`attested:`).

```
Phase 0  Frame        what, who, purpose, target language
Rebuild  Transition   legacy model + target model + migration.yaml + surface ledger (opt-in)
         tool: Gm-transition (complete disposition, mappings, phases, cutover, risks)
               Gs-surface (every legacy route/command/table/job disposed against the target)
         attested: what is worth saving; transformation and rollback semantics; sweep completeness
Phase 1  Modelith     domain model
         tool: modelith lint clean; pinned render engine reproduces every committed *.modelith.md
               after the mechanical em-dash normalization
               Gc-carrier (every invariant has a named carrier or a reasoned waiver)
         attested: lifecycle enums, action pre/post, invariant owners, scenario coverage
Phase 1.5 Relational  static relational models (opt-in, per invariant shape):
           policy    access control      -> Gp-policy    (Policy.als + Policy.oracle.md)
           integrity structure/keys      -> Gi-integrity (Integrity.als)
           isolation multi-tenant refs   -> Gn-isolation (Isolation.als + Isolation.oracle.md)
         tool: each gate binds + covers; committed models fresh; solver via verify-formal
         attested: the annotations state what the prose invariants mean
Opt-in   External     bring-your-own deterministic checker (SAST, AST, Datalog, graph reasoner):
           checkers   -> Gk-<id> (input_hash binds, coverage holds; engine via verify-checkers)
         tool: projection is fresh; committed evidence binds to it; verdict is pass; coverage complete
         attested: the manifest claims the elements the checker is responsible for
Phase 2  C4           architecture + contract (+ surfaces.yaml, the target surface ledger)
         tool: G2-c4 (contract parses and binds; allow graph acyclic; mitigation coverage;
               drawn relationships judged by the allow/deny/baseline rules; an interface
               contract per allowed edge; NFR record with all three topics; event tables
               answered and resolved; opt-in action-ownership and adoption-closure tables)
               Gu-surfaces (every act a person performs has a named surface or a deferral;
               sources: names where the act list was enumerated from)
               engine: verify-c4 (workspace.dsl compiles under structurizr-cli)
         attested: each contract's shape is the right one; the NFR record's content; the
                   dependency declaration and the persona walk are complete
Phase 3  XState       state machines + named-unit matrices + generated oracles
         tool: G3-machine (lint; every invoke has onError + timeout; every fully guarded handler
               declares its guard-false disposition; delays consumed both ways; oracle fresh;
               matrix reconciled)
               Gd-idcite (stable-id citations resolve; CLAUSES and READS declarations hold)
               Gy-rules (the consistency rules over the declared, projected facts)
               engine: verify-formal (TLC over the generated TLA+, Alloy over the relational models)
         attested: each guard enforces the invariant it names; residual failure transitions kept
                   (bindable: the mitigation table's opt-in "handled by" column resolves per row)
Phase 4  BUILD.md     the blueprint
         tool: Gx-trace + Gr-reads + Gb-plan + Ge-embed (+ G4-import and Gt-tests once code
               exists; Gw-packet once a slice map projects bounded per-slice executor packets)
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
         tool: Gv-attest (claim from a closed vocabulary, attestor, covered artifacts and
               their content hashes; an edited artifact makes the judgment STALE)
         attested: whether each judgment is true, which is the point of the split
Always   Ledgers      STATE.md / DECISIONS.md formats + house style
         tool: Gl-ledger (self-review grammar; dated entries; em-dash/emoji scan, warn
               tier, except em dashes in a *.modelith.md render, which error)
         attested: the ledgers' content; which phases owe a self-review line
```

The skill ([skills/machinery/SKILL.md](skills/machinery/SKILL.md)) spells out the split per gate
and is the operator's contract.

## The gates

`machinery check <design>` runs the default suite: every gate whose artifacts exist. `--gate`
takes a comma list from exactly this vocabulary:

```
gm,gs,gu,gp,gi,gn,gc,g2,g3,gd,gl,gx,gy,gr,gk,gb,gw,ge,ga,gj,gv,g4,gt,g5
```

Each gate prints a header, a `checked:` line with its counts, and either `ok` or findings at one of
three severities: **ERROR** blocks; **DRIFT** means a generated artifact is stale, and also blocks;
**WARN** is advisory (`--warnings-as-errors` promotes it). The last line counts blocking findings.
`--complete` (which requires `--impl`) is the final-handoff mode: every phase artifact present,
milestones closed, zero warnings.

Gate 1 is `modelith lint` on the domain model. The binary has no `g1`, by design: Phase 1's gate is
Modelith's own linter.

### Transition and surface ledgers

- **Gm-transition** (rebuild and hybrid designs, `migration.yaml`): every legacy entity disposed,
  replaced attributes and lifecycle values fully mapped, ordered source-of-truth phases with
  rollback, an evidence-based cutover, owned transitional risks. Mapping rows may declare `tests:`,
  and with `--impl` every named regression identifier must appear in the test corpus.
- **Gs-surface** (a legacy surface ledger, `legacy/surface.yaml`): all six surface classes
  inventoried or waived, every route, command, table, job, event, and integration covered, dropped,
  or deferred, and covered bindings resolve against the target. An `as_of:` anchor that is neither
  an ISO date, a VCS revision, nor a tag-like token warns.
- **Gu-surfaces** (a target surface ledger, `surfaces.yaml`): every action whose actor is a person
  is mapped to a named surface (screen, admin command, API route, config release) or deferred with a
  reason; each mapped act resolves against the domain model and matches its actor; every act is
  stated once; `sources:` names where the act list came from; once BUILD.md declares milestones,
  every `milestone:` resolves.

### Model and relational layers

- **Gc-carrier**: every declared invariant has a named carrier (an action's `preserves`, a
  relational layer, a machine matrix unit, an external checker's coverage claim) or an explicit
  waiver with a reason. It needs only the domain model, so it runs from Phase 1: an obligation the
  design does not carry fails the moment it is written.
- **Gp-policy** (a policy annotation): the annotation binds to the domain model, covers every
  top-level invariant, and the committed `Policy.als` and `Policy.oracle.md` byte-match a fresh
  generation.
- **Gi-integrity** and **Gn-isolation** (each with its annotation): the annotation binds to the
  domain model, and the committed `Integrity.als`, or `Isolation.als` and `Isolation.oracle.md`,
  byte-match a fresh generation.
- **Gk-\<id\>** (one per checker manifest): the committed projection is fresh, evidence binds to it
  via `input_hash`, the verdict is pass, and the coverage claim is complete.

### Architecture

- **G2-c4**: the Architecture Contract parses and binds to `workspace.dsl`; the allow graph is
  acyclic (cycles that close only through `baseline:` edges warn as ratchet debt); `assert: no_path`
  claims hold over the transitive closure; every dependency has a mitigation row; the NFR record
  mentions security, capacity, and observability; every event-contract table names its enumeration
  source and every row answers producer, consumer, payload, delivery, ordering, and dedupe, with
  producer and consumer resolving to declared elements; every allowed crossing has an interface
  contract row (edge, shape, errors, idempotency) or a `(no contract: <reason>)` waiver, and no row
  exists for an edge nobody allows. Every relationship the DSL draws is judged by the same rules
  G4 judges a code edge by. Opt-in by table presence: action ownership (every model action owned
  once by a resolvable component) and adoption closure. `machinery verify-c4` is the engine phase.
- **Gr-reads** (contracts with root `reads:` rows): described above.

### Behavior and traceability

- **G3-machine**: structural lint, the TLA generator's own admissibility pass (a machine G3 passes
  is one `verify-formal` can generate), retry-counter accounting, derived-deadline span checks, the
  dotted-reference name-rot audit, `_branch_order` for overlapping guarded branches; committed
  oracles byte-match a fresh generation, matrices reconcile, named units are covered. A matrix with
  no machine is accepted only with a `(no machine: <reason>)` placement waiver.
- **Gd-idcite** (designs with machines): every stable-id citation in hand-written files resolves to
  a committed oracle row; positional `T-<TAG>-NN` citations warn; `CLAUSES{...}` and `READS{...}`
  declarations catch clause drift and payload-sufficiency drift; a conjunctive guard contract with
  no clause vocabulary warns, waivable per row with `(single-clause: <reason>)`.
- **Gx-trace**: states to enum values, events to actions, invariants to enforcement rows, entities to
  persistence-placement rows; mitigation `handled by` names resolve to a machine or invoke actor;
  event-contract rows reconcile against the machines (a consumed event is handled or ignored, a
  produced event appears in some action or matrix cell, or the cell carries a
  `(no machine: <reason>)` waiver). Armed by a `<!-- machinery:reads-complete -->` marker, every
  `(event, consumer)` edge owes its own `READS{...}` declaration or a `(no reads: <reason>)` waiver.
  Gx also reports the shape errors of every declaration group and of the authorization inventory;
  what they mean is Gy-rules'.
- **Gy-rules** (designs with `machines/` or an `AUTHORIZATION.md`): the shipped rules over the
  projected facts, as described [above](#where-the-design-must-agree-with-itself-the-rules-gate).
  `--explain` prints each finding's derivation.

### Build plan and delivery

- **Gb-plan** (a BUILD.md): milestones are unique `**M<n> - <title>**` markers, the walking skeleton
  comes first (or carries a waiver), every milestone has a `DoD:` line, the skeleton's DoD cites a
  committed oracle id, and the skeleton carries an `NFR:` line. In `Mode: manifest`, each milestone
  has exactly one `Demo:` line. `Linkage: pairwise` (the default) keeps one bounded
  `BUILD/M<n>-*.md` packet per milestone; `Linkage: matrix` requires reciprocal milestone-to-shard
  links that match exactly in both directions.
- **Gw-packet** (a `slices.yaml`): described above.
- **Ge-embed** (a `machinery:embed` marker): every marked table is held to its declared source
  (`subset`: each row byte-identical to a source row; `complete`: every selected source row
  present), with `(shard-local: <reason>)` exempting what it names. `machinery embed refresh`
  writes what this gate checks.
- **Ga-accept**, **Gj-adjudication**, **Gv-attest**: acceptance evidence, characterization verdicts
  (every verdict binds to one committed oracle id; a model-is-truth verdict names its filed defect),
  and attestation evidence, each active when its artifact exists.
- **Gl-ledger** (always): `STATE.md` self-review lines parse, `DECISIONS.md` entries carry real
  dates, and the house-style scan warns on em dashes and emojis across the hand-written design tree
  (an em dash in a generated `*.modelith.md` render is an ERROR, because the post-render strip is
  mechanical). A `.machineryignore` at the design root (gitignore-shaped, patterns relative to the
  root, no negations) keeps non-design paths, such as a spike's vendored dependencies, out of every
  design-tree walker.

### Code and composition

- **G4-import** (with `--impl`): code imports respect the contract boundaries in Go, Python,
  TypeScript/JavaScript, Elixir, and Rust.
- **Gt-tests** (with `--impl`): described above, including one falsifying-clause id per active
  clause of every `CLAUSES{...}`-declared guard. A decomposed parent with no machines still runs Gt
  when it owns relational obligations.
- **G5-pack** (decomposed designs): packs fresh, children pinned to the current packs, refinement
  proofs fresh, and every boundary-event row held to its direction (`consumes` or `produces`).

## Proof it works: the examples

### go-crm

[`examples/go-crm`](examples/go-crm) is a Go CRM with a native CLI over an embedded
[LadybugDB](https://github.com/LadybugDB/go-ladybug) graph and role- and ownership-based access
control, taken end to end:

- **Designed** through all four phases as the production target of a checked rebuild. The legacy
  prototype and `migration.yaml` account for 4 legacy entities, 16 data mappings, 9 lifecycle
  mappings, 4 coexistence and cutover phases, and 3 transition risks. The target domain model lints
  clean with 9 entities and 27 invariants; the C4 model carries the dependency posture an embedded
  store forces (corruption is fatal until restore, not a transient); five state machines with 197
  transitions; a 1,194-line `BUILD.md`.
- **Built by a zero-context coding agent** under hard TDD: a test-writer wrote the suite from the
  blueprint, the tests were locked, and an implementer made them pass without touching a test. The
  one impossible test was escalated as a design defect and fixed in the design, not the code.
- **Gated** by `machinery check --impl`: all gates green, with 197 matrix rows reconciled, 186 named
  units covered, 13 import edges verified, 6 closed milestones bound to reviewed commits, and every
  one of the 275 committed oracle rows referenced from the test suite.
- **Proven** by `machinery verify-formal`: eight TLC proofs plus 24 Alloy commands across policy,
  integrity, and isolation (32 checks), regenerated from source every run. The policy suite found the
  two access-control defects described [above](#access-control-a-manager-who-provably-cannot-write-their-own-records).

Every number above is real output in this repository.

### The other examples

- [`examples/fulfillment`](examples/fulfillment) is the distributed stress test: microservices, a
  saga orchestrator, compensation, and a transactional outbox, with six machines (the saga plus the
  Order, Payment, Reservation, Shipment, and OutboxMessage lifecycles). Its data-refined model proves
  money and stock are never silently lost, and it is where the refund defect and the enum drift above
  were caught. `FINDINGS.md` records what strained and what was fixed.
- [`examples/surreal-crm`](examples/surreal-crm/README.md) rebuilds go-crm over SurrealDB in Docker.
  It exercises Gs-surface and an all-reuse `migration.yaml`. Its machines and oracles are
  byte-identical to go-crm's, which is the lesson: mitigations reclassify failures, and the domain
  does not move.
- [`examples/portfolio-engine`](examples/portfolio-engine) is a Python drawdown portfolio
  recommender in a different domain. It exercises the terminal-lifecycle pattern and a persistence
  overlay renamed from the defaults, and its root BUILD manifest with six milestone packets is the
  worked large-project, small-model handoff.
- [`examples/checkout-split`](examples/checkout-split) is recursive decomposition: a parent that
  stops at contracts plus two child designs, each a full machinery run against its frozen pack,
  with TLC-checked proofs that each child refines the contract its neighbor relies on.
- [`examples/pii-flow`](examples/pii-flow/README.md) is the external-checker reference described
  above. The full default suite passes on it with zero blocking findings and the one Gv warning it
  carries by design.

## Brownfield, rebuild, and hybrid systems

### Brownfield

The pipeline reads as greenfield, but the toolchain does not care. On an existing system the phases
run as archaeology. Before architecture, classify every coherent legacy corpus area as behavior to
preserve or port, capability to rearchitect or adapt, learning-only evidence, historical-only
material, or unresolved intent. A missing classification never means "must port," and an
unresolved product decision blocks the architecture handoff. Then model the target the evidence
supports and let the gates arbitrate what they can see.

Be precise about what that is. The only code-facing gates are G4-import, which reads import
statements, and Gt-tests and Gr-reads, which read test references and Git history. No gate executes
code or reads its behavior. Behavioral drift surfaces through characterization tests derived from
the generated oracles: stable ids give every transition a durable test key, and G3's DRIFT keeps
the oracles regenerated as the model is corrected. Oracle-derived tests start descriptive and are
locked (hard TDD) once a person adjudicates each failing row as code-is-truth (fix the model) or
model-is-truth (file the code defect), recorded under `design/adjudications/` and held by
Gj-adjudication. From there, revision mode is the operating loop: design changes as diffs,
stable-id oracle diffs as the affected-test list, and state-migration notes for persisted machines.

Day one is a command: `machinery baseline <design> --impl .` scans the code exactly as G4 does,
proposes `baseline:` rules that tolerate today's violating edges (distinct from intended `allow:`
rules, and compatible with a `deny:` that keeps the intent written), and records
`design/ratchet.json`. From then on a new offending file on an amnestied edge is a blocking finding,
so the baseline can only shrink. The staged team protocol is in the
[brownfield team guide](docs/brownfield-team-guide.md).

### Rebuild and hybrid

When the current platform works but should not be the production foundation, do not model a
halfway system. Keep `design/legacy/domain.modelith.yaml` as the current truth,
`design/domain.modelith.yaml` as the target truth, and `design/migration.yaml` as the checked
bridge. Gm-transition requires a disposition for every legacy entity, a mapped-or-new decision for
every target entity, a salvage decision for implementation assets, complete field and lifecycle
coverage for replacements, ordered source-of-truth phases, an explicit cutover and rollback window,
and owned failure postures for temporary migration dependencies. Shadow reads must define parity;
dual writes must define idempotency, conflict resolution, and reconciliation. A legacy type whose
disposition names a target is owned by that disposition, so a contract row's
`SUPERSEDES{type:...}` resolves against it.

Gm's coverage universe is the legacy model the run declared, so it cannot see a subsystem the
excavation missed. The legacy surface ledger, `design/legacy/surface.yaml`, closes that hole by
inventorying what can be enumerated mechanically (routes, CLI commands, tables, jobs, events,
integrations) and disposing every item. The forward twin is the target surface ledger,
`design/surfaces.yaml`: a model can declare that an administrator suspends a tenant, the architecture
can place it, a machine can encode it, and the design can pass green while nothing names the screen
or command the administrator uses. Gu-surfaces holds that set closed. Guides:
[rebuild](docs/rebuild-guide.md), [surface ledger](docs/surface-ledger.md),
[target surfaces](docs/target-surfaces.md).

## Which model to use where

The gates check structure, not substance. A shallow domain model with the wrong invariants gates
completely clean. Extracting the right invariants, pushing on fuzzy definitions, and deciding
failure postures (Phases 0 through 2) is judgment with no machine backstop, so use the strongest
reasoning model you have there. Phases 3 and 4 are different: half of each machine is derived, and
lint, oracle diff, reconciliation, the rules, and TLC catch most of what a weaker model would get
wrong, so a mid-tier model is much safer for that synthesis. The deterministic layer narrows the
failure mode without removing it: with a weak interrogator you get a structurally consistent,
formally verified model of the wrong system.

Execution is a different workload from design. For a large project the root `BUILD.md` is only the
ordered milestone and demo authority; each milestone links to a packet under `BUILD/` carrying its
domain, architecture, behavior, oracle, TDD, implementation, risk, recovery, and acceptance context.
A pairwise packet stays under 64 KiB and may not depend on another packet for missing context. That
keeps implementation within reach of smaller or local models without weakening the tests, proofs,
or acceptance boundary.

## When not to use machinery

CRUD screens, UI-heavy surfaces, and pure transforms get a contract spec and ordinary tests, not the
four-phase pipeline. Forcing a machine onto them is ceremony, and it reads as ceremony. machinery
pays off where state bites: lifecycle, saga, protocol, retry, and workflow logic, and the
deterministic envelopes around agentic systems. Model the stateful core, not the whole repository.

## What machinery does not verify

The gates and proofs are exactly as strong as stated above. No deterministic check or proof covers:

- whether the interrogation extracted the right invariants (a shallow domain model gates clean);
- guard and action semantics in code (the named-unit contracts carry them into unit, integration,
  and property tests, and a wrong implementation of a correctly named guard is caught by tests, not
  proofs);
- races between concurrent machine instances, and message loss, duplication, or reordering between
  machines (the models are single-instance; the event-contract table and idempotency contracts
  govern those seams, and tests exercise them);
- whether migration transformations preserve real production data (Gm proves decision coverage;
  mapping, reconciliation, and rollback tests hold the rest);
- coupling through shared database tables or bus topics, which import analysis cannot see;
- security, capacity, and observability beyond what the Phase 2 NFR record captures;
- facts a design states only in prose (the rules read declarations).

The methodology names each residual in the design artifacts rather than letting a green check
imply it is covered.

Inside what is checked, four claims stay separate: artifact consistency (a committed artifact is
byte-fresh against the current design), proof execution (a solver ran and reproduced what the
committed evidence says, in `verify-formal` or `verify-checkers`), test execution (your suite ran
and passed; the `Gt` gate credits static discovery of test references only, and tests were not
executed by any machinery gate), and current review (`Gv` attestations carry an explicit kind,
`plan`, `current`, or `historical`, so a design-time review is never read as a current
implementation review). The known runtime exclusions (replay, races between concurrent machine
instances, message duplication and reordering, migration, restore, and load) stay named, owned test
obligations until the executable test-assurance contract's enforcement lanes land. Naming an
obligation in guidance enforces nothing by itself.

When machinery writes design artifacts and a publication is interrupted, readers block fail-closed.
`machinery recover <design-dir>` reports the state read-only, and `--apply` completes only a fully
revalidated publication.

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
  Environment variables override it, for example `MACHINERY_VERSION=v0.10.0`,
  `INSTALL_DIR=/usr/local/bin`, `MACHINERY_HOMES="$HOME/Agent Home"`, or
  `MACHINERY_TARGETS="codex opencode"`. `MACHINERY_HOMES` takes one full path per line, preserving
  spaces; separate several homes with a literal newline.

  **Binary by hand** (macOS arm64 and x86, Linux amd64 and arm64): download
  `machinery-<os>-<arch>` from the [releases page](https://github.com/RamXX/machinery/releases),
  put it on your `PATH`, and let it install its own skill:
  ```bash
  machinery install                        # fetches the matching skill + role docs into your agent homes
  ```

  **Windows:** v0.10.0 publishes a cross-compiled `machinery-windows-amd64` binary and
  `machinery_<version>_windows_amd64.tar.gz` as release artifacts. The one-line installer and
  `machinery update` do not support Windows, so download the asset by hand. Native Windows runtime
  guarantees are not claimed: process custody, formal verification, and the assurance lanes are
  unix-only and refuse on Windows. Use Linux or macOS for the full toolchain.

  **Build from source** (with [Go](https://go.dev/dl/) 1.27+; `go.mod` pins 1.27.1). This is
  also how you run a change before it is released:
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
machinery check <design> --gate gl --verbose
                                      # every undeclared-fact line, no per-file summary
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
machinery update --version v0.10.0        # force an exact release
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
`scripts/ci-linux.dockerfile`, the single owner of the Go (1.27.1), Node (26.9.0) with TypeScript
(7.0.2), CPython (3.14.7), and Elixir (1.20.4) on OTP (29.1.1) identities, shared with
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
