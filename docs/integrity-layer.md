# The integrity layer: a practical guide

This guide is for someone who wants the structural-integrity checks without first learning Alloy or
the machinery internals. It is the sibling of the [policy layer](policy-layer.md), one concern over:
where policy governs access, integrity governs structure.

## What it is, in one minute

Some domain invariants are neither access rules nor lifecycle rules. "No two users share a
username." "Exactly one pipeline is the default." "Every order belongs to exactly one customer."
These are **structural relations**: they constrain which configurations of records, keys, and links
are legal, not who may touch them and not how they change over time. That puts them outside every
other machinery net: the policy layer governs access, TLC model-checks behavior, and the domain-model
linter checks that the prose is well-formed. A uniqueness rule that is well-formed prose but jointly
unsatisfiable with the rest of the model passes all of them.

The integrity layer compiles those invariants into a bounded relational model and proves they hold
together. One annotation states the structural constraints in a small closed vocabulary. From it,
`machinery alloy` generates **`design/formal/Integrity.als`**, a relational model plus a standard
suite of admissibility checks.

The rung is **admissibility, not safety**. The policy layer asks "is anything bad permitted?" and
answers with checks that must find no counterexample. The integrity layer asks "are the intended
structures admissible, and do they scale?" and answers with runs that must find a satisfying
instance (plus inverse cardinality checks where multiplicity requires exclusivity). A green
integrity model proves the whole constraint set is jointly satisfiable and each constraint is
non-vacuous; it goes red the moment an edit makes two constraints incompatible, which
the linter cannot see. There is no oracle: integrity is a design-side property with no per-request
decision function to conformance-test.

## Do I need it?

Use the layer when the domain model carries structural invariants of these shapes:

- **uniqueness**: no two records of an entity share an attribute value.
- **singleton**: exactly one record of an entity has a boolean flag set.
- **mandatory / cardinality**: a relationship every record must have, or whose multiplicity is
  load-bearing (a 1:1 owned chain, an n:1 ownership).

Skip it when the domain model has none of these, or when the only structural facts are already
carried by another layer (ownership single-owner constraints, for instance, may be the policy
layer's `owned_invariants`). portfolio-engine skips it legitimately: its invariants are numeric and
temporal, not relational-structural.

One honest boundary: the v1 algebra proves joint satisfiability, populatability, and uniqueness
non-vacuity. It does not model attribute *semantics* beyond uniqueness (no ordering, no arithmetic),
and it treats open value types (string, integer, timestamp) as unbounded. Rules of that kind belong
to the machines and the implementation tests.

## The five-minute walkthrough

Using the go-crm and fulfillment examples as templates:

```yaml
# design/formal/integrity.relational.yaml (go-crm: keys + singleton, no relationships)
entities: [User, Team, Tag, Pipeline]
unique:
  - {entity: User, attribute: username, invariant: username-unique}
  - {entity: Team, attribute: name, invariant: team-name-unique}
  - {entity: Tag, attribute: name, invariant: tag-name-unique}
singleton:
  - {entity: Pipeline, flag: isDefault, invariant: one-default-pipeline}
```

```yaml
# fulfillment: a mandatory relationship plus uniqueness
entities: [Customer, Order, Payment]
relationships:
  - {from: Order, to: Customer, field: customer, mandatory: true, invariant: order-owned-by-customer}
  - {from: Order, to: Payment, field: payment, mandatory: true}   # structural, no invariant binding
unique:
  - {entity: Customer, attribute: email, invariant: customer-email-unique}
```

Then:

```bash
machinery alloy design/            # generate Integrity.als (emits every present relational layer)
machinery check design/            # Gi-integrity: binding, coverage, freshness (no Java needed)
machinery verify-formal design/    # run the admissibility checks next to the TLC proofs (needs Java)
```

## The annotation reference

The file is `design/formal/integrity.relational.yaml`. Unknown keys anywhere are hard errors. Every
referenced entity, attribute, and invariant id must exist in the domain model; every disagreement
fails generation with a message naming the mismatch (the same drift rule the policy and refinement
annotations follow).

### `entities` (required)

The explicit modeling surface: every entity that appears in a relationship, a unique, or a singleton
must be listed here, so the signature set is bounded and nothing is modeled by accident.

### `relationships`

Each entry names a relationship the domain model already declares and states its `mandatory`
decision (Modelith cardinality cannot express whether a relationship is required). Keys: `from`,
`to`, optional `field` (the Alloy field name, default lowercased `to`), `mandatory` (bool), optional
`invariant`. A relationship may omit `invariant` to model multiplicity structurally without binding
a domain id. Multiplicity is derived: `n:1`/`1:1` become `one` (mandatory) or `lone`; `1:n`/`n:n`
become `some` (mandatory) or `set`. `1:1` and `1:n` also emit an inverse `lone` fact and an
`Exclusive_<From>_<Field>` check: field multiplicity constrains how many targets one source holds,
but only the inverse constraint prevents two sources from sharing a target that belongs to one.

### `unique`

`{entity, attribute, invariant}`: no two records of `entity` share `attribute`. The value domain is
bounded to the exact type cardinality for boolean (2) and enum types, and left unbounded for open
types. Bounding is what lets the solver catch uniqueness declared on a domain too small to populate.

### `singleton`

`{entity, flag, invariant}`: exactly one record of `entity` has the boolean `flag` set. `flag` must
be a boolean attribute of the entity.

### `residuals`

Optional. Structural-shaped invariants the algebra cannot carry, each with a `reason`. Same shape and
drift rule as the policy layer. Integrity does not force top-level coverage (that is the policy
layer's contract); it simply compiles what it names.

### `scope`

Optional integer 3 to 12 (default 6): the Alloy search bound. Three is the minimum because the
standard `Populatable` witness requires three records of every modeled entity.

## The generated checks, in plain language

Every design gets the same suite; none of it is authored per design.

| check | question it answers | a FAIL means |
|---|---|---|
| `SomeWorld` (run) | is the whole constraint set jointly satisfiable? | the constraints contradict each other; a structural over-specification the linter cannot see |
| `Populatable` (run) | can every entity reach a population target of 3 under all constraints? | a cardinality or uniqueness constraint starves the model (it admits a token world but cannot scale) |
| `Distinct_<E>_<attr>` (run) | can two records with different values of a unique key coexist? | the unique key is vacuous over a value domain too small |
| `Exclusive_<From>_<Field>` (check) | for a `1:1` or `1:n` relationship, can two sources share one exclusive target? | the inverse side of the declared multiplicity was weakened or omitted |

`verify-formal` prints one PASS/FAIL line per command. Because the commands are runs, a FAIL is an
empty search: the asserted world does not exist within the bound. The fix is always in the domain
model.

The layer was mutation-verified during development: uniqueness declared on a boolean attribute
bounds the entity to two records, so `Populatable` (which requires three) fails at the solver. That
is the integrity analog of the policy layer's teamless-Manager catch: a constraint that reads fine in
prose but cannot be satisfied.

## Gates, and where each thing runs

| concern | held by | needs Java |
|---|---|---|
| annotation parses, binds to the domain model | Gi-integrity (`machinery check`) | no |
| committed `Integrity.als` byte-matches a fresh generation | Gi-integrity (DRIFT, blocking) | no |
| invariants compiled by the annotation trace as enforced | Gx-trace ("integrity-checked" class) | no |
| the admissibility checks actually hold | `machinery verify-formal` (pinned Alloy 6.2.0 jar) | yes |
| nobody hand-edits the generated model | plugin hooks deny writes to `formal/*.als` | no |

Gate selection is automatic: `gi` runs when `formal/integrity.relational.yaml` exists, otherwise
never. `--gate gi` forces it (and errors if the annotation is missing).

## Limits, stated plainly

- **Bounded, not unbounded.** Exhaustive only up to the scope (default 6 atoms per signature).
- **Admissibility, not semantics.** The layer proves the constraint set is satisfiable and scales; it
  does not model attribute meaning beyond uniqueness (no ordering, arithmetic, or format rules).
- **The algebra is deliberately small.** Uniqueness, singletons, and relationship multiplicity.
  Watch the count in the gate output: if a design carries structural invariants the layer does not
  name, they are residuals or they belong to another rung, not silently uncovered.
