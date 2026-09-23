# Proposal: the consistency layer (declarations, facts, rules)

**Status: Stages 1 to 4 implemented** (each has an "as implemented" subsection below; Stage 5,
new rules without new Go, is open). Written 2026-09-22 against v0.9.0. It replaces
the prose-heuristic implementation of the four Gx extensions that shipped in 0.9.0 with
a declaration grammar, a fact projection, and rules evaluated as data, and it gives the
remaining consistency gaps (effect carriers, twin tables, source supersession,
contract-only records) a home that needs no new Go per gap.

## 1. What is wrong with the current state

v0.9.0 added four checks to Gx-trace: an authorization inventory for System writes,
named-unit fact resolution, closed `VALUES{}` vocabularies, and event-payload twins.
The declarations they introduced (`VALUES{...}`, `payload {...}`, `derived: x (reason)`,
the AUTHORIZATION.md table) are sound and stay. The way the checks decide what to check
is not:

- `internal/gates/authz.go`, `facts.go`, `values.go` and `payloadtwins.go` carry 35
  compiled regular expressions and 20 identifiers prefixed `h2`. They decide whether a
  matrix cell is a write from phrases such as "writes nothing", "but ... write",
  "never adjusts the row"; whether a cell names a closed vocabulary from "reason class",
  "closed ... enum"; and whether a backticked token is a fact from surrounding words like
  "only for", "marks", "there is no". A design that phrases the same contract differently
  gets a different verdict.
- One consumer's private grammar (`MACHINE-WRITTEN{}`, `MACHINE-WRITTEN-BY{}`, the
  residual verb table, "all presets withheld") is parsed by the open-source tool.
  Machinery is supposed to hold the line for any design; it now holds it for one.
- NEXT.md entry 10, written by the owner, says: "Avoid keyword heuristics advertised as
  semantic verification." The 0.9.0 checks are exactly that.
- The bundled "Datalog" reference, `examples/pii-flow`, does not run a Datalog engine.
  `rules.dl` is documentation; `adapter.py` reimplements the fixed point by hand in
  Python. There is no rule evaluator anywhere in machinery today.

The shape of every one of these gaps is the same, and it is the shape entries 13, 15,
16, 20 and 37 also have: a closed set of design facts, an obligation ("every X owes a
Y"), an agreement ("two statements of Z are equal"), or a reachability ("Z resolves to
a declared source"). These are relations over facts. They should be written as rules
over facts, not as string scanners over prose.

## 2. Principles

1. **Prose never declares.** A fact the gate acts on comes from a declaration with a
   grammar, or from a structured artifact (Modelith YAML, the machine JSON, the contract
   YAML, a table with a marker). A sentence may quote a fact; it cannot create one.
2. **Facts have stable ids and are projected once.** `machinery project` is the single
   reader of the design. Every gate, internal or external, sees the same facts.
3. **Rules are data.** A methodology rule is a versioned text file shipped with
   machinery, reviewable in a diff, and runnable under a reference engine. A consumer
   rule lives in the consumer's design under the existing Gk contract.
4. **Machinery stays engine-neutral and dependency-free.** `machinery check` remains
   hermetic (no Docker, no solver). The reference engine runs in the engine phase and in
   CI, the same split TLC and Alloy already have.
5. **A consumer's grammar lives in the consumer.** If a design carries its own
   authorization notation, the consumer maps it to the public declaration; machinery
   never learns it.

## 3. Architecture

Three layers, each usable without the next.

### 3.1 Declarations (the grammar family)

Machinery already has one declaration idiom: an upper-case word followed by a braced,
comma-separated closed list, placed in a matrix cell. `CLAUSES{}`, `READS{}`,
`ORACLESET{}`, `VALUES{}` (0.9.0) and `payload {}` (0.9.0) are its members. The layer
adds the members the four checks and the open gaps need, and retires the prose scanners.

| Declaration | Where | Meaning | Replaces |
|---|---|---|---|
| `WRITES{Entity.attr, ...}` or `WRITES{}` | matrix row, contract column | The closed set of stored facts this unit writes. `WRITES{}` states a read-only unit. | `h2NoWrite`, `h2ConditionalWrite`, `h2PositiveWrite` and the verb regexes |
| `USES{fact, ...}` | matrix row, contract or clause column | The closed set of facts this unit reads or names. Each must resolve (3.3). | the backtick scan plus `valueRole`, `negativeFact`, `producerRole`, `prosePayloadField` |
| `derived: fact (reason)` | matrix row | Unchanged from 0.9.0: a fact the unit computes and never stores. | nothing |
| `VALUES{a, b}` and `VALUES name{a, b}` | matrix row | Unchanged from 0.9.0. Presence of the group is the whole trigger. | `closedVocabWord`, `reasonClassWord`, `reasonClassMembers` and the entity-definition sniffing |
| `payload {f, ...}` | matrix event row | Unchanged from 0.9.0. | nothing |
| `CARRIES{column:Order.status, outbox:order.confirmed, sink:auditLog, signal:..., action:Machine.unit}` | matrix row, for units of kind action or actor | What carries the unit's effect (NEXT.md entry 20). | nothing (new) |
| AUTHORIZATION.md table with the `machinery:authorization-inventory` marker | design root | Unchanged from 0.9.0: subject `Entity.action`, admission a backticked capability or `(no authorization: reason)`. | the H2 resource-cell, `MACHINE-WRITTEN`, residual-verb readers |
| `SUPERSEDES{type:OldName}` on a contract row, or `supersedes:` in a schema catalog | Architecture Contract | Source ownership and replacement of a stable type id (entry 15). | nothing (new) |

Two rules make the grammar closed:

- A backticked snake_case or `Entity.attr` token in a contract, clause or payload cell
  that is not inside a group is a **warning** under Gl-ledger ("undeclared fact
  reference; declare it in USES{} or WRITES{} or drop the backticks"). It never fails a
  gate and never resolves. This is what stops prose from becoming policy by accident.
- A group name machinery does not know is an **error** in the matrix lint, exactly as an
  unknown `_` annotation is in the machine lint.

H2's `MACHINE-WRITTEN{}` and residual-verb notation are not adopted. H2 already parses
those marks at compile time (that is why Gr-reads exists), so it owns a reader for them;
a twenty-line script in H2 emits the public AUTHORIZATION.md rows from that reader, and
Gr-reads binds the two files. Machinery ships the public grammar and one migration note.

### 3.2 Facts (projection v2)

`machinery project` grows from three layers to the set the rules need. Every element
keeps the v1 stable-id discipline, so evidence binds to identity rather than to a line.

| Layer | Facts (relation names as emitted) | Stable id |
|---|---|---|
| `model` (v1) | `entity`, `attr`, `enum_member`, `relationship` | `entity:X`, `attr:X.a`, `enum:X.a.v`, `rel:...` |
| `invariants` (v1) | `invariant`, `invariant_owner` | `inv:id` |
| `actions` | `action(id, entity, name, actor)`, `action_writes(id, attr)` | `action:Entity.name` |
| `machines` | `machine`, `state`, `transition(id, from, event, to)`, `guard_on`, `action_on`, `invoke`, `context_key(machine, key)` | `machine:M`, `state:M.s`, `tr:M.n` |
| `matrices` | `unit(id, machine, name, kind)`, `unit_writes`, `unit_uses`, `unit_derived(id, fact, reason)`, `unit_values(id, group, member)`, `unit_clauses`, `unit_reads`, `unit_carries(id, kind, target)`, `unit_payload(id, event, field)` | `unit:M.name` |
| `events` | `event(id, source)`, `event_participant`, `event_payload_field(id, field)`, `event_consumer(id, c4)` | `event:name` |
| `c4` | `c4_element(id, kind, parent)`, `c4_relationship`, `boundary`, `allowed_edge`, `reads_row` | `c4:id` |
| `authorization` | `admission(subject, capability)`, `no_authorization(subject, reason)` | `action:Entity.name` |
| `oracles` | `oracle_row(id, machine, transition)` | `orc:id` |
| `milestones` | `milestone`, `dod_id(m, orc)`, `slice_claim(slice, orc)`, `bound_at(orc, path)` (with `--impl`) | `ms:id`, `slice:id` |
| `supersession` | `type_owner(type, source)`, `supersedes(new, old)` | `type:Name` |

Two output forms from one projector:

- `projection.json` under `projection_schema: "2.0"`, a superset of v1. Manifests that
  ask for v1 layers keep byte-identical output, so `examples/pii-flow` and every
  committed evidence file stay valid.
- `machinery project --facts <dir>` writes one tab-separated `<relation>.facts` file per
  relation, the input format Soufflé and every Datalog engine reads directly. No
  adapter is needed to run rules against a design.

The reserved-layer error in v1 ("requesting `machines` fails loudly") is kept until each
layer lands; a layer is added to the manifest vocabulary only with its schema and a
golden fixture.

#### Projection v2, as implemented

Stage 2 landed every layer of the table; `scenarios` stays reserved. The contract is
`schemas/projection-v2.schema.json`, generated from the relation catalog in
`internal/checker/relations.go`; the reader is `internal/gates/projection_readers.go`;
`docs/external-checkers.md` is the user reference. Decisions settled while implementing:

- **Selection.** A manifest naming only `model`, `invariants` and `relationships` gets the
  1.0 projection byte for byte; any other layer selects 2.0. The 1.0 schema file is left
  untouched and 2.0 is a sibling file, so no v1 consumer sees a changed contract.
- **Shape.** 2.0 keeps the 1.0 `model` block (present exactly when a v1 layer is included)
  and adds `layers.<layer>.<relation>`, one row per tuple with `stable_id`, `source` and
  one string per column. `relationship` sits in the `relationships` layer, matching the
  v1 include vocabulary.
- **No prose columns.** `unit_derived(unit, fact)` and `no_authorization(subject)` drop
  the reason column the rules sketch in section 4 gives them, because the reason is
  prose. `event(id, producer)` names the producer; `event_participant` is producers and
  consumers together; `unit_clauses` carries `active` or `retired`; `boundary` carries a
  `role` (`boundary` or `external`); `supersedes(new, old)` takes the new type from the
  declaring row's subject and `type_owner(type, owner)` names the declaring artifact.
- **Ids.** Transition ids are the generated oracle's stable ids (`tr:Order.ORDE-eb2d3b`).
  Fact id columns carry the stable id without its kind prefix, so `WRITES{Order.status}`
  joins `attr` directly. A declaration on a matrix row that names no unit (a consumed-event
  row) attaches to `matrix:M`, and a matrix with no machine JSON projects with `machine`
  empty.
- **Presence.** A layer is present only when its sources exist; a present layer claims all
  of its relations, empty ones included. Two definitions of one stable id from different
  places fail, naming both.
- **Omitted.** `action_writes` (Modelith has no structured post-condition) and `bound_at`
  (no `--impl` reader is wired into the projection yet).
- **Facts form.** `machinery project <design> --facts <dir>` writes every present layer's
  relations plus `relations.txt`; a round-trip test over every bundled example loads the
  directory with `internal/datalog` and matches each relation's count to the JSON.

### 3.3 Rules

**Syntax.** A Soufflé-compatible subset: `.decl`, `.input`, `.output`, Horn clauses,
stratified negation with `!`, equality and inequality, `count` and `min` aggregates as
the only arithmetic. Symbols only, no records. Every rule file runs unchanged under real
Soufflé; that is the parity contract, not a hope.

**Two engines, one grammar.**

- `internal/datalog`, a semi-naive evaluator with stratified negation, standard library
  only, in the order of six hundred lines. It runs the shipped methodology rules
  in-process so `machinery check` stays hermetic and can answer "what is broken now"
  without Docker. It records, for each derived finding, the rule and the body facts
  that produced it, which is the trace the gate prints under `--explain`.
- Soufflé, digest-pinned in an OCI image, in `machinery verify-checkers` and in the CI
  parity lane. Every shipped rule file plus every golden-corpus fact set is evaluated by
  both engines and the output relations must be byte-identical. `souffle -t explain`
  gives the reference derivation tree. This is the same discipline the RustReason /
  PyReason parity harness uses, applied to our own evaluator.

The alternative, running rules only through Docker, was rejected: it would make the
consistency checks unavailable in the hook and in a design-only lane, which is where the
H2 gaps were found late.

**Two tiers of rules.**

- *Methodology rules* ship under `rules/consistency/<gate>.dl`, versioned with the
  binary, embedded at build time. They implement the gate `Gy-rules` (letter open; `gy`
  is free in the `--gate` vocabulary). Each output relation named `finding_<code>` becomes
  a finding with that code; a `warn_<code>` relation becomes a warning.
- *Consumer rules* live in `design/checkers/<id>/rules.dl` under the existing Gk
  contract, now over the v2 projection. Regulatory encodings held in NIL, and anything
  else confidential, use this path through the private registry exactly as today.

**Findings.** A finding row is `(code, subject_id, detail)`. The gate resolves
`subject_id` back to the file and row through the projection's source map, so the
message still says `machines/Order.matrix.md row persistOrder`, as it does now.

#### Subset, as implemented

`internal/datalog` (Stage 3, first slice: package and tests only, no gate wired)
accepts exactly the following, and every program it accepts runs unchanged under
Soufflé 2.5 with identical output relations. The parity test runs each program under
`internal/datalog/testdata/programs` through both engines when `souffle` is on PATH
and compares the outputs after sorting the lines of both (Soufflé does not fix the
row order of its output files).

- **Items.** `.decl name(attr:type, ...)` with one or more attributes;
  `.input name` and `.output name`, one relation each, no IO parameters; rules
  `head(...) :- literal, ... .` with a single head. Comments `//` and `/* */`.
- **Types.** `symbol`. `number` only as an attribute type for relations that carry
  aggregate results; an `.input` relation must be all `symbol`.
- **Terms.** Variables (any identifier; `_x` is an ordinary variable), the wildcard
  `_` (not in a head, not in a comparison), and double-quoted string constants
  without backslashes, tabs, newlines or carriage returns.
- **Literals.** Positive atoms, negated atoms `!r(...)`, comparisons `=` and `!=`,
  `V = count : { ... }` and `V = min X : { ... }` with the braced body only. `min`
  is accepted only over a `number` variable, because Soufflé rejects `min` over
  symbols; in practice `min` ranges over `count` results.
- **Aggregate grouping.** A variable inside an aggregate that also occurs outside
  it groups the aggregate and must be bound by a positive atom outside it (as in
  Soufflé). `count` of an empty group is 0; `min` of an empty group yields no tuple.
  Nested aggregates are rejected.
- **Range restriction.** Every head variable, and every variable in a negated atom
  or comparison, must be bound by a positive body atom or be an aggregate result.
- **Stratification.** Negation or aggregation inside a strongly connected component
  is a compile error that prints the cycle, for example
  `cycle through negation p -> r -> q -> p`.
- **Rejected with a position.** Records, ADTs, arithmetic, number literals,
  functors (built-in and `@user`), comparisons other than `=`/`!=`, `.type`,
  `.plan`, `.printsize`, `.functor`, `.comp`, `.init`, `.pragma`, relation
  qualifiers (`brie`, `eqrel`, `inline`, `magic`, `choice-domain`, ...),
  subsumption, disjunction, multiple heads, ground facts in the program text,
  nullary relations, `max`/`sum`/`mean`, `count :` without braces, the
  preprocessor, and Soufflé keywords used as names.

Decisions settled while implementing:

- **Tabs.** A tab is always the column separator in `.facts` and `.csv` files, so a
  symbol can never contain a tab, newline or carriage return. Such values are
  rejected where they enter: Go inputs (error), program constants (parse error), fact
  files (the line has the wrong column count). The two characters `\t` in a fact file
  are two ordinary characters in both engines. A trailing carriage return on a fact
  line is dropped, as Soufflé drops it; an empty line is a one-column tuple holding
  the empty symbol, as in Soufflé.
- **Inputs.** A missing input for an `.input` relation is an error (Soufflé errors on
  a missing fact file). Inputs for relations the program does not mark `.input` are
  ignored, as Soufflé ignores stray fact files, so one `--facts` directory can feed
  many rule files. Duplicate input rows collapse; inputs are sorted before loading.
- **`.input` with rules** is allowed, as in Soufflé: facts are seeded and the rules
  add to them. **`.output` on an `.input` relation** is allowed and writes the
  deduplicated facts.
- **Output order.** `Result.Output` and the written `.csv` files are sorted column by
  column, symbols bytewise and numbers numerically.
- **Derivations.** With `Options.Explain`, each derived tuple records its first
  derivation: the 1-based rule index, the rule's position, and the tuples matched by
  the rule's positive body atoms outside aggregates (negations and aggregates have no
  witness tuples). `Result.Explain` returns a tree whose leaves are input facts; an
  input fact is a one-node tree.
- **Limits.** `MaxTuples` (default 10,000,000, input facts included) and
  `MaxIterations` (default 1,000,000 evaluation rounds summed over all strata). Hitting
  either returns an error wrapping `ErrLimit` and no result.

#### Stage 3, as implemented

`Gy-rules` (letter `gy`, in the `--gate` vocabulary and the default suite after `gx`) runs the
rule files under `rules/consistency/`, embedded by the `rules` package: `authz.dl`, `facts.dl`,
`values.dl`, `payload.dl`, `carriers.dl`, `supersession.dl`. Each is plain data in the subset above
and runs unchanged under Soufflé.

- **Activation.** A design with a `machines/` directory or an `AUTHORIZATION.md`. An explicit
  `--gate gy` on any other design prints the gate with `note   not activated: the design has no
  machines/ and no AUTHORIZATION.md, ...`; the default suite skips it silently.
- **Facts.** Built in process by the Stage 2 reader (`LoadDesignFacts`); nothing is written. Every
  catalog relation is supplied, so a relation of a layer the design lacks is an empty input, never
  a missing-input error. A design that cannot be projected is an ERROR of the gate.
- **Load-time contract**, checked once per process and by `TestShippedRulesLoad` at test time, so a
  broken rule file fails this repository's tests rather than a user's check: every file parses;
  every `.input` is a catalog relation of the same arity; every `.output` is named
  `finding_<code>` or `warn_<code>`, and every relation so named is an `.output`; no two files emit
  one relation; every output attribute is a symbol, and the first names the subject kind (`unit`,
  `action`, `subject`, `type`), which is how the gate locates the subject.
- **Severity.** Stage 3 ran `finding_*` tuples as a non-blocking preview severity of their own;
  Stage 4 made them ERRORs and removed that severity (see Stage 4 below). `warn_*` tuples are
  ordinary warnings.
- **Messages.** `path:line: row 'X': <code> (attr 'value', ...)`, where `X` is the subject id and
  the location is the first projected row whose first column is `X` in the relations its kind
  names (`unit`, then the declaration rows `unit_declares` and `unit_derived`, so a finding on a
  matrix row that names no unit is still located; `action`; `admission`, `no_authorization`,
  `action` for a `subject`; `type_owner`, `supersedes` for a `type`). A subject no row carries
  prints as `X (no source): <code>`.
- **Limits.** `MaxTuples` 2,000,000 and `MaxIterations` 100,000 per rule file, set explicitly. A
  hit is `ERROR rules/consistency/<file>: evaluation stopped at a resource limit (...)`; the other
  files still run.
- **`--explain`.** Under each finding, indented, the derivation tree: the finding tuple with
  `[<file> rule <n>]` (1-based index in source order), then each tuple the rule's positive body
  atoms matched, derived ones with their rule and input facts with `[fact <path>:<line>]`. Negated
  and aggregated literals hold by absence and have no witness, so they are not printed. A design
  with no findings prints nothing extra.

What the rules read and decide, where the text above left it open:

- **authz.dl.** A System write is a Modelith action with actor `System`, unless the matrix unit
  of the same id declares `WRITES{}` with no member (the unit and action ids coincide when the
  matrix is named after the entity). Matrix cascade and consumer producers are not obligations:
  projection v2 has no relation for them (the producer cell is prose-shaped), so the 0.9.0
  heuristic still covers them. Capabilities resolve against `c4_element` ids only; no declared
  capability list exists yet (NEXT.md's authorization entry).
- **facts.dl.** A `USES{}` or `WRITES{}` member resolves to an `attr` id, an `enum_member` id, a
  `context_key`, or an `event_payload_field`, or to a `derived:` waiver or `VALUES` member on the
  same unit (both are row-local declarations). `WRITES{Order.status}` joins `attr` directly,
  because fact columns carry stable ids without their kind prefix; the fixture
  `rules-fact-unresolved` pins it.
- **values.dl.** Group and enum names are compared exactly, and an enum no attribute uses is not
  projected, so it binds no group.
- **carriers.dl.** Entry 20 ships in two tiers. `warn_effect_uncarried`: an action or actor unit
  with a non-empty `WRITES{}` and no `CARRIES{}` (a real warning; no bundled design has one).
  `finding_actor_uncarried`: an actor unit with no `CARRIES{}` that is not declared read-only by
  `WRITES{}`. It ran as a preview, not a warning, because the bundled designs' 41 actor units
  predated `CARRIES{}` and the example gate policy treats any warning as a failure; Stage 4 added
  the carriers and merged both relations into `finding_effect_uncarried`.
- **supersession.dl.** At Stage 3 `duplicate_owner` could not fire (ARCHITECTURE.md was the only
  artifact owning a type) and `dangling_replacement` fired on go-crm's `SUPERSEDES{type:LegacyDeal}`,
  because `migration.yaml` projected no `type_owner`. The rule was right on its input; Stage 4
  closed the projection's gap.
- **`unit_declares(unit, group)`** was added to projection 2.0 for these rules (one row per group
  present on a row: `WRITES`, `USES`, `CARRIES`, `VALUES`, `CLAUSES`, `READS`, `payload`).

Evidence. `TestRulesParity` runs every rule file over every bundled example and one synthetic
design on which every output has tuples (at Stage 3, every output but `finding_duplicate_owner`),
under Soufflé and `internal/datalog`, comparing sorted outputs: 54 runs (6 files by 9 designs), all
equal. The CI
job `datalog-parity` runs it, with the evaluator's own parity corpus, in an image built from pinned
inputs (`scripts/souffle.dockerfile`: Ubuntu 24.04 and golang 1.27.1 by digest, the Soufflé 2.5
`.deb` by sha256, dependencies from a fixed Ubuntu snapshot), and fails if a test skipped its
Soufflé half. At Stage 3 an agreement test compared the class A to D subjects of the 0.9.0 Gx
checks with the `finding_*` subjects: on every activated example both were empty. Three fixtures
recorded the structural discrepancies, each with a verdict (Stage 4 keeps them as rules-only
fixtures):

| Case | Only in | Right side and why |
|---|---|---|
| a System action whose description says it writes nothing | rules (`authz_missing`) | rules: the claim is prose; `WRITES{}` on the unit declares it |
| a fact backticked in a contract cell, in no group | heuristic (unresolved fact) | rules: a bare token is prose, which Gl-ledger already warns on |
| `VALUES{...}` on unit `orderState` against enum `OrderState` | heuristic (`VALUES` mismatch) | heuristic, on intent; the fix is a normalized group key in the projection, or the explicit `VALUES OrderState{...}` |

#### Stage 4, as implemented

The prose heuristics are gone and Gy-rules decides alone, at ERROR tier.

- **Coverage first.** Nothing the heuristics covered was dropped before they were deleted:
  - *Producers.* `PRODUCES{Entity.action, ...}` on the matrix row naming a cascade or consumer arm,
    parsed with the Stage 1 groups, projected as `unit_produces(unit, action)`. In `authz.dl` a
    produced action is a System write (so it owes an admission whatever its Modelith actor) unless
    its own unit declares `WRITES{}`. A member the model does not declare is its own finding,
    `produces_unknown_action`, and owes no admission (there is no action to admit). No bundled
    example needed a `PRODUCES` group: the 0.9.0 producer-column inference found no matrix
    producer on any of them.
  - *Migration owners.* Every `migration.yaml` disposition naming a target projects
    `type_owner(<legacy>, migration.yaml)`; a retired legacy entity names no target and is claimed
    by nothing. go-crm's `LegacyDeal` replacement resolves, and `duplicate_owner` can now fire.
  - *VALUES binding.* Exact names, no case folding and no fuzzy join: `VALUES Enum{...}` binds
    `Enum`, and an unnamed group binds only when its unit name equals the enum name exactly.
    Gl-ledger warns on an unnamed group whose unit name equals an enum name only when lower-cased,
    naming the fix.
  - *Carriers.* All 41 actor units of the bundled examples declare `CARRIES{}` from their own
    signature, pre/post and maps-to columns. `warn_effect_uncarried` and `finding_actor_uncarried`
    merged into `finding_effect_uncarried`; `finding_carrier_misplaced` rejects `CARRIES{}` on a
    unit that is neither an action nor an actor.
  - *Named vocabularies.* The one real defect the VALUES heuristic caught that no rule covered
    (two units spelling one enum-less named group differently) is `finding_values_conflict`.
- **Severity.** Every `finding_*` relation is an ERROR, printed with its `--explain` derivation
  under the ERROR line. The Stage 3 preview severity is removed whole (the gate field, the
  printing, the `checked:` count, the golden lines).
- **Deletion.** `authz.go`, `facts.go`, `values.go` and `payloadtwins.go` are deleted with their
  tests (2,639 lines), including every regular expression that decided a fact from prose and every
  identifier naming one consumer's notation. The parsers they held (the `AUTHORIZATION.md` table,
  `payload {}`, `derived:`, `VALUES{}`) live in `declarations.go`. Gx-trace narrows to
  traceability plus the shape errors of those parsers and of the Stage 1 groups.
- **One release of overlap.** Gx warns on a design that declares no `WRITES{}`, `USES{}` or
  `PRODUCES{}` anywhere and quotes a snake_case or `Entity.member` token in prose on a row with no
  group, naming the migration note in the machinery skill. A migrated design never sees it. It
  fires on no bundled example (checkout-split and portfolio-engine gained the `USES{}` their prose
  had named) and is removed in the release after next.
- **Where it runs.** The stop hook selects `gy` with the CLI's activation rule (`machines/` or
  `AUTHORIZATION.md`); `make preflight` runs the parity lane through
  `make dagger-job JOB=datalog-parity`.
- **Consumer notation.** A design with a private authorization notation generates the
  `AUTHORIZATION.md` rows from its own reader and binds the generated file to that reader with a
  `reads:` row (Gr-reads); machinery reads only the public table.

## 4. The rules, written out

These replace the four 0.9.0 checks and cover the open gaps. Each is short enough to
review in a pull request, and each is testable by one fixture that fails without it.

```
// A. Every System write owes an admission row (entry 34).
.decl system_write(a:symbol)
system_write(A) :- action(A, _, _, "System").
system_write(A) :- unit_carries(_, "action", A).
.decl finding_authz_missing(a:symbol)
finding_authz_missing(A) :- system_write(A), !admission(A, _), !no_authorization(A, _).
.decl finding_authz_orphan(a:symbol)
finding_authz_orphan(A) :- admission(A, _), !system_write(A).
.decl finding_authz_unknown_capability(a:symbol, c:symbol)
finding_authz_unknown_capability(A, C) :- admission(A, C), !capability(C).
.decl capability(c:symbol)
capability(C) :- c4_element(C, _, _).
capability(C) :- declared_capability(C).      // the capability list NEXT.md now asks for

// B. Every fact a unit names resolves (entry 35).
.decl declared_fact(f:symbol)
declared_fact(F) :- attr(F, _, _).
declared_fact(F) :- enum_member(F, _, _).
declared_fact(F) :- context_key(_, F).
declared_fact(F) :- event_payload_field(_, F).
declared_fact(F) :- unit_derived(_, F, _).
declared_fact(F) :- unit_values(_, _, F).
.decl finding_fact_unresolved(u:symbol, f:symbol)
finding_fact_unresolved(U, F) :- unit_uses(U, F), !declared_fact(F).
finding_fact_unresolved(U, F) :- unit_writes(U, F), !attr(F, _, _).

// C. A VALUES group agrees with a same-named model enum (entry 36).
.decl finding_values_disagree(u:symbol, g:symbol, v:symbol)
finding_values_disagree(U, G, V) :- unit_values(U, G, V), enum_exists(G), !enum_member(_, G, V).
finding_values_disagree(U, G, V) :- enum_member(_, G, V), enum_exists(G), unit_values(U, G, _), !unit_values(U, G, V).

// D. Payload twins (entry 30) and the milestone table twin (entry 37 residual).
.decl finding_payload_twin(u:symbol, e:symbol, f:symbol)
finding_payload_twin(U, E, F) :- unit_payload(U, E, F), !event_payload_field(E, F).
finding_payload_twin(U, E, F) :- unit_payload(U, E, _), event_payload_field(E, F), !unit_payload(U, E, F).
.decl finding_milestone_binding_stale(o:symbol)
finding_milestone_binding_stale(O) :- milestone_says_unbound(O), bound_at(O, _).

// E. Every declared effect names a carrier (entry 20).
.decl finding_effect_uncarried(u:symbol)
finding_effect_uncarried(U) :- unit(U, _, _, "action"), unit_writes(U, _), !unit_carries(U, _, _).
finding_effect_uncarried(U) :- unit(U, _, _, "actor"), !unit_carries(U, _, _).

// F. Source supersession (entry 15).
.decl finding_duplicate_owner(t:symbol)
finding_duplicate_owner(T) :- type_owner(T, S1), type_owner(T, S2), S1 != S2, !supersedes(S2, S1), !supersedes(S1, S2).
.decl chain(a:symbol, b:symbol)
chain(A, B) :- supersedes(A, B).
chain(A, C) :- chain(A, B), supersedes(B, C).
.decl finding_supersession_cycle(t:symbol)
finding_supersession_cycle(T) :- chain(T, T).
.decl finding_dangling_replacement(n:symbol, o:symbol)
finding_dangling_replacement(N, O) :- supersedes(N, O), !type_owner(O, _).

// G. Contract-only records (entry 13): a matrix with no machine is legal iff waived.
.decl finding_orphan_matrix(m:symbol)
finding_orphan_matrix(M) :- matrix(M), !machine(M), !no_machine_waiver(M, _).
```

The 0.9.0 Go implementations of A through D total about 1,300 lines plus 950 of tests.
The rules above are 60 lines. The difference is not cleverness; it is that the Go code
spends most of itself deciding what a sentence meant.

## 5. Migration from v0.9.0, staged

Each stage ships on its own and leaves the suite green.

**Stage 0. Freeze.** A CONTRIBUTING rule: a gate may not decide a fact from prose. New
regular expressions over matrix cells are rejected in review unless they parse a
declared group. The `h2*` identifiers are marked deprecated in code.

**Stage 1. Declarations.** Add `WRITES{}`, `USES{}`, `CARRIES{}` and `SUPERSEDES{}` to
the matrix and contract parsers (the `VALUES{}` parser is the template). Add the Gl warn
tier for bare backticked facts. Update the skill, the matrix template and the four
example designs. No gate semantics change yet; the four 0.9.0 checks keep running.

**Stage 2. Projection v2 and `--facts`.** Add the layers of 3.2 one at a time, each with
its schema fragment, stable ids, a golden projection and a source map. Keep v1 output
byte-identical. Make `examples/pii-flow` run its `rules.dl` under a digest-pinned Soufflé
image and delete the hand-rolled fixed point from `adapter.py`, so the reference example
is a real engine.

**Stage 3. Evaluator, parity lane, preview gate.** Land `internal/datalog` with the
parser, stratification, semi-naive evaluation and derivation recording. Add the CI
parity lane (Docker, Soufflé pinned by digest) that evaluates every shipped rule file over
every golden fact set on both engines and diffs the outputs. Land `Gy-rules` with rules A
through D running as a preview: findings are emitted without blocking and the golden corpus
records both the Gx findings and the Gy findings side by side. Any subject the two
disagree on is a fixture to write.

**Stage 4. Cut over.** Promote Gy findings to error tier. Delete the prose heuristics from
`authz.go`, `facts.go`, `values.go` and `payloadtwins.go`, keeping only the group parsers,
which move next to the other declaration parsers. Remove every `h2*` identifier. Narrow
Gx-trace back to traceability. Record in CHANGELOG the migration for H2: generate
AUTHORIZATION.md from its own reader, add `WRITES{}` and `USES{}` where the 0.9.0 scan
had inferred them. One release of overlap where Gx accepts the old inference with a
deprecation warning is acceptable; two is not.

**Stage 5. New rules, no new Go.** Entries 20, 37 residual, 15 and 13 land as rule files
plus fixtures plus, where needed, one declaration each. From this point the cost of a
consistency gap is a rule and a fixture.

Entry 16 (challenge invented prerequisites and incompatible proof reuse) is judgment,
not a rule, and Stage 5 ships no rule for it. Its questions are whether a newly
blocking dependency is necessary for the consumer claim it blocks, and whether a
reused receipt's authority, lifecycle preconditions and scope hold for a new claim.
Both are questions about meaning. A rule joins only what a design declares, so a rule
for entry 16 would either restate the author's own statement of necessity (and check
nothing) or infer necessity from matching names and field shapes, which is the prose
inference principle 1 forbids and entry 16 itself warns against ("matching field
shapes or names do not import the producer's authority"). The parts of it that are
facts already have rules: a replaced or dangling source (`supersession.dl`), a stale
reservation, an unresolved fact (`facts.dl`). The necessity review stays with the
conductor and the human reviewer, beside readiness and failure traces (section 7).

Order of value: Stage 2 unlocks every consumer checker immediately, including the NIL
hook, even before the evaluator exists. Stage 3 is where the H2-shaped code stops being
needed. Stage 4 is where it leaves.

## 6. Tests and evidence

- **Parity** is the load-bearing test: both engines, every rule file, every golden fact
  set, byte-identical outputs. A rule that only one engine accepts is a syntax error in
  the subset, not a feature.
- **One fixture per rule**, in `internal/experiments`, that fails without the rule and
  passes with it, plus a near-neighbour that must not fire (the false-positive control
  entry 10 asks for).
- **Golden corpus** records the `finding_*` relations per example design; a change in
  the corpus is a reviewed change in policy.
- **Adversarial entries** convert one to one from the review findings of the H2 design
  edits (entry 33's classification), sanitized, as the standing rule requires.

## 7. What this layer does not do

- It does not judge semantics. Readiness (entry 9), failure-trace reviews (10),
  experiments (11) and assembled-value bounds (14) stay attested or experimental.
- It does not add unknown as an outcome. The four-state evidence contract (entry 4) is
  schema work on the Gk side and is independent of this layer; consumer rules that need
  a three-valued answer run in the private registry.
- It does not read implementation code. `--impl` facts such as `bound_at` come from the
  existing Gt and Gr readers; the rules only join them.

## 8. Decisions for the owner before Stage 1

1. The gate letter (`gy` proposed) and whether the four rules stay under Gx's name for
   continuity or move under the new gate at Stage 4.
2. Whether an in-process evaluator is acceptable, given the no-new-dependencies rule.
   The proposal adds no module to `go.mod`; the evaluator is ours, and the parity lane
   is what keeps it correct.
3. How long the 0.9.0 prose inference is tolerated. Proposal: one release, warning
   tier, then deleted.
4. Whether `examples/pii-flow` should be converted to real Soufflé in Stage 2 or left as
   a Python illustration. Proposal: convert; the example is the contract's proof.
