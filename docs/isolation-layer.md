# The isolation layer: a practical guide

This guide is for someone who wants multi-tenant isolation checked at design time and enforced in
code, without first learning Alloy or the machinery internals. It is the third relational algebra,
alongside the [policy layer](policy-layer.md) and the [integrity layer](integrity-layer.md).

## What it is, in one minute

The policy layer checks who may read or write a record, by role and direct ownership. It never looks
at what a record **references**. But a record an actor may legitimately read can carry a link to a
record in another tenant, and following that link returns data the actor could never read directly.
A Rep may read its own `Task`; that `Task` links to a `Deal`; if nothing forbids it, the `Deal` can
be owned in another team, and reading the task leaks the deal. Access rules never see this, because
the leak is in the reference graph, not in any single record's ownership.

The isolation layer carries the meaning of the **cross-entity tenant-consistency invariants**: a
record and the records it references belong to one tenant. `tenant(record) = owner's tenant`; a
reference is tenant-consistent when the two owners share a tenant. From one annotation,
`machinery alloy` generates:

1. **`design/formal/Isolation.als`**, a bounded relational model whose checks prove, most sharply,
   that two records in different tenants can never share a referent (`SharedReferent`) -- a
   non-trivial consequence the single-hop facts do not give for free.
2. **`design/formal/Isolation.oracle.md`**, the tenant-scoping decision table: for each reference and
   each tenant relationship between the two owners, the expected allow/deny verdict, with stable ids.
   The implementation's link-authorization function is held to it, exactly as the policy oracle holds
   the access code.

## Do I need it?

Use the layer when all of these hold:

- the domain model has a **tenant** entity (a team, workspace, organization, account),
- a **subject** entity that belongs to a tenant (usually the record owner's team),
- **cross-entity references** between tenant-scoped records that must not cross a tenant boundary.

Skip it when there is no tenant boundary (fulfillment, portfolio-engine have none), or when records
never reference each other across the tenant-scoped set.

The distinct value over the policy layer is precisely the reference graph. If the only tenant fact is
"a user reads its own team's records" (direct ownership), the policy layer already carries it; the
isolation layer earns its place only when following a link could leak across tenants.

## The five-minute walkthrough

Using the go-crm example as the template:

```yaml
# design/formal/isolation.relational.yaml
tenant:
  entity: Team
subject:
  entity: User
  tenant_attr: team        # the User field that carries tenant membership
  membership: lone         # a User holds at most one Team (state one | lone explicitly)
references:
  - {from: Task, to: Deal, field: deal, invariant: task-deal-same-tenant}
  - {from: Activity, to: Contact, field: contact, invariant: activity-contact-same-tenant}
```

Then:

```bash
machinery alloy design/            # generate Isolation.als + Isolation.oracle.md
machinery check design/            # Gn-isolation: binding, freshness (no Java needed)
machinery verify-formal design/    # run the isolation checks next to the TLC proofs (needs Java)
```

Finally, wire the link-authorization conformance test in the implementation (one test, once; see
"The tenant oracle" below).

## The annotation reference

The file is `design/formal/isolation.relational.yaml`. Unknown keys anywhere are hard errors. Every
referenced entity, attribute, and invariant id must bind to the domain model or generation fails.

### `tenant` (required)

`entity`: the tenant entity (a Modelith entity). It must have a relationship to the subject entity.

### `subject` (required)

| key | meaning |
|---|---|
| `entity` | the acting/owning entity that holds a tenant |
| `tenant_attr` | the subject field that carries tenant membership (rendered as a relation into the tenant) |
| `membership` | `lone` (a subject may be tenantless) or `one` (always tenanted). Modelith cardinality cannot express which; state it |

### `references` (required, one or more)

Each entry names a cross-entity reference the domain model already declares, and the tenant-
consistency invariant it holds. Keys: `from`, `to`, optional `field` (default lowercased `to`),
`invariant` (required: every reference carries the id it holds). Both `from` and `to` must be
tenant-scoped records, each with an n:1 ownership relationship to the subject entity. An `n:1`
reference (many `from` can point at one `to`) additionally gets a `SharedReferent` check.

### `residuals`

Optional. Tenant invariants the algebra cannot carry, each with a `reason`.

### `scope`

Optional integer 2 to 12 (default 6): the Alloy search bound.

## The generated checks, in plain language

| check | question it answers | a FAIL means |
|---|---|---|
| `SomeWorld` (run) | can a genuinely multi-tenant world with a link still exist? | the isolation facts collapse tenancy (over-isolation: links force one tenant) |
| `SharedReferent_<From>_<To>` (check) | can two records of different tenants reference the same target? | a shared referent bridges the tenant boundary -- the sharp leak |
| `Possible_<From>_<To>` (run) | is a same-tenant link actually constructible? | the isolation fact is vacuous (it forbids the link entirely) |

The layer was mutation-verified during development: strip the enforcing facts and `SharedReferent`
finds a `Deal` referenced from two tenants at once. With the facts (the invariants enforced), it
holds. That is the isolation analog of the teamless-Manager catch: a leak the reference graph always
admitted but nothing enforced.

## The tenant oracle and its test

`Isolation.oracle.md` enumerates the tenant-scoping decision as rows:

```
| test id     | stable id     | reference    | tenant case | expectation | invariants |
| O-TENANT-02 | TENANT-573586 | Task -> Deal | cross-tenant| deny        | task-deal-same-tenant |
```

Because the algebra decides every case from one boolean (do the two owners share a tenant), the four
tenant cases (`same-tenant`, `cross-tenant`, `source-teamless`, `target-teamless`) are the complete
semantics. Stable ids hash the case, never the verdict: a design change flips expectations under
unchanged ids.

**The tenant oracle test** is one test, written once, that parses the table and asserts the pure
link-authorization function on every row, expanding each tenant case into its concrete owner-team
pairs. The reference implementation is
`examples/go-crm/impl/internal/authz/tenant_oracle_test.go`, asserting `authz.AuthorizeLink`. Keep
the tenant decision in one pure function so the rows map onto direct calls, and key test names on the
stable id.

The loop this closes: change a reference invariant, regenerate, and the conformance test fails on the
named rows until the code follows. Mutation-verified: loosening `AuthorizeLink` to allow cross-tenant
links failed the suite on exactly the `cross-tenant`, `source-teamless`, and `target-teamless` rows.

## Gates, and where each thing runs

| concern | held by | needs Java |
|---|---|---|
| annotation parses, binds to the domain model | Gn-isolation (`machinery check`) | no |
| committed `Isolation.als` and `Isolation.oracle.md` byte-match a fresh generation | Gn-isolation (DRIFT, blocking) | no |
| invariants compiled by the annotation trace as enforced | Gx-trace ("isolation-checked" class) | no |
| the isolation checks actually hold | `machinery verify-formal` (pinned Alloy 6.2.0 jar) | yes |
| the implementation agrees with the tenant oracle | the conformance test in your suite | no |
| nobody hand-edits generated artifacts | plugin hooks deny writes to `formal/*.als` and `formal/*.oracle.md` | no |

Gate selection is automatic: `gn` runs when `formal/isolation.relational.yaml` exists, otherwise
never. `--gate gn` forces it.

## Limits, stated plainly

- **Bounded, not unbounded.** Exhaustive only up to the scope (default 6 atoms per signature).
- **Design-side plus test-side, not runtime.** The solver holds the invariant set; the oracle test
  holds the implementation. A code path that establishes a link without calling the authorization
  function bypasses all of it. Keep link authorization at one call site.
- **One tenant dimension, single-owner tenancy.** `tenant(record) = owner's tenant`. Hierarchical
  tenants, records with multiple tenants, or tenancy not derived from ownership are residuals.
- **The references are the ones you name.** A reference not listed in the annotation is not held to a
  tenant. The annotation is the enumeration of what must stay in-tenant; keep it complete.
