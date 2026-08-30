# Target surface ledger reference

Every other gate looks at the design's internals. None of them asks how a PERSON reaches an act.
That question had no artifact, so it had no gate and no closed review list: a model can declare
that an administrator suspends a tenant, the architecture can place it, a machine can encode it,
and the whole design passes green while nothing names the screen, admin command, API route, or
config release the administrator uses. Eight admin-gated acts once survived ten deep reviews that
way. This ledger is the forward twin of `legacy/surface.yaml`: Gs holds the design to the LEGACY
system's surface, Gu holds it to its own.

```text
design/surfaces.yaml                # the target surface ledger; activates Gu-surfaces
```

## The closed set

The obligated set is derived from the domain model, so it can be held mechanically:

- action whose `actor` is a person: **owes a named surface**;
- action whose `actor` is `System`: owes nothing;
- action with **no** `actor` field: owes nothing, because the model has not said who performs it.

The third case is the one that erodes the gate, so the actorless count prints on every run. A
model where nobody filled in `actor` yet gates clean with zero obligations and says so in a note.
That is a signal to finish Phase 1, not a pass.

## The persona-walk sweep (Phase 2, before Gate 2)

For every human persona in the glossary, walk that persona's COMPLETE action list and name the
surface for each act. Persona by persona, not entity by entity: the entity walk is the one the
design already does, and it is exactly the walk that misses this, because it groups acts by the
thing acted on rather than by the person acting. Name the concrete thing, so a reader can tell
whether it exists: `Admin console > Tenants > Suspend tenant`, `admin CLI: tenant suspend <id>`,
`POST /v1/tenants/{id}/suspend`, `tenant settings release, Billing tab`.

Configuration knobs get rows too (`knob:<key>`): a knob a person is expected to change is an
interface, and a design that assumes an operator can turn something on has assumed one.

## The artifact

```yaml
surface_version: 1
sources:
  - domain.modelith.yaml action list, walked persona by persona
acts:
  - act: Deal.approve
    actor: TenantAdmin
    surface: Admin console > Approvals queue
    milestone: M2
  - act: knob:billing.grace_period_days
    actor: TenantAdmin
    surface: Tenant settings release, Billing tab
deferrals:
  - act: Deal.reopen
    reason: Reopening is a support-ticket path until M4; no self-serve surface is designed yet.
  - act: actor:Auditor
    reason: The auditor portal is a separate program of work, scoped after cutover.
```

Root keys (strict; unknown keys fail): `surface_version` (the integer `1`, required), `sources`
(optional list of non-empty strings naming where the act list was enumerated from), `acts`
(required), `deferrals` (optional), `_comment` (optional).

`acts` rows: `act` (required, `Entity.action` or `knob:<key>`), `actor` (required), `surface`
(required, the screen, admin command, API route, or config release), `milestone` (optional,
non-empty when present), `_comment` (optional).

`deferrals` rows: `act` (required, `Entity.action`, `knob:<key>`, or `actor:<Name>` to defer one
persona wholesale), `reason` (required), `_comment` (optional). `actor:<Name>` is the honest form
for a persona whose whole interface is out of scope for now: one row a reviewer can see, instead
of the same reason scattered across a dozen act-level deferrals.

## The gate

`Gu-surfaces` (`machinery check <design> --gate gu`) activates automatically once the file exists.
It verifies, deterministically: the schema is strict at every level; every obligated action is
covered by an `acts` row naming it exactly or by a deferral (its own or its persona's), and
anything else is an ERROR naming the act; every `Entity.action` row resolves against the target
model AND matches the actor the model declares (a ledger that misattributes an act closes the
review question with the wrong answer); every act value is stated exactly once across `acts` and
`deferrals` together; a `knob:` row resolves against nothing, since configuration is an open set,
so only its shape and uniqueness are held; and an `actor:<Name>` deferral must name an actor some
action in the model declares.

The `checked:` line prints six numbers, zeros included:

```text
checked: 12 obligated actions, 10 covered, 1 deferred acts, 1 deferred personas, 3 knob rows, 4 actorless actions
```

`covered` counts obligated actions satisfied by an `acts` row, so it reads against `obligated
actions` as a coverage fraction. The deferral numbers are row counts and one persona deferral can
discharge several acts, so the six deliberately do not sum.

LLM-attested (you check these; the tool cannot): each named surface is the RIGHT one and will
actually exist; the deferral reasons are decisions rather than placeholders; and, above all, the
model's `actor` fields are filled in. The gate holds the ledger to the model; only the persona
walk holds the model to reality.

## Relationship to the other gates

- **Gs-surface** covers the legacy system's observable surface, **Gu-surfaces** the target
  design's. A rebuild wants both: Gs proves nothing was dropped by accident, Gu proves the
  replacement is reachable.
- **Gx-trace** holds the persistence-placement table to the entity list; Gu holds this ledger to
  the action list. Same mechanism, same reason: the set is closed, so coverage is checked rather
  than trusted.
- **Gp-policy** answers who is ALLOWED to perform an act; Gu answers how they perform it. A design
  can pass Gp with a perfect authorization rule protecting an act no interface exposes.
- Gu carries no generated artifacts, so it has no DRIFT class; the ledger is a source file.
