# The target surface ledger and the Gu-surfaces gate

Every gate before this one asks a question about the design's internals. G2 asks whether the
boundaries hold, Gx asks whether every entity has a home and every action a machine, Gp asks who
is allowed to do what. None of them asks the question a user asks first: **how does a person
actually reach this act?**

That question has no artifact, so it has no gate and no closed review list, and the failure mode
is quiet. A domain model can declare that a tenant administrator suspends a tenant, an
architecture can place the action in the right boundary, a machine can encode its transition, and
the whole design can pass every gate green while nothing anywhere names the screen, the admin
command, the API route, or the configuration release through which the administrator does it. The
act is designed; the interface to it is not. Nobody notices, because no list ever had to be
walked to completion.

The finding that produced this gate: eight admin-gated acts in a real design survived ten deep
reviews with no named interface. The cause was not carelessness. The human-act-to-surface mapping
had no artifact, so there was nothing for a reviewer to walk and nothing for a tool to hold.

`design/surfaces.yaml` is that artifact, and `Gu-surfaces` is that gate. It is the forward twin of
`Gs-surface`: Gs holds the design to the LEGACY system's observable surface, Gu holds it to its
OWN. Where Gs asks "does the new design account for this old capability", Gu asks "does the new
design name the interface a person uses for this new act".

## The closed set

Gu works because its obligated set is mechanically derivable, exactly like the entity list Gx
holds the placement table to. The domain model declares actions, and Modelith actions carry an
`actor`. So:

- an action whose `actor` is a person **owes a named surface**;
- an action whose `actor` is `System` owes nothing (the system reaches it on its own);
- an action with **no** `actor` field owes nothing, because the model has not yet said who
  performs it.

That third case is the one that quietly erodes the gate, so the count of actorless actions is
printed on every run. A model where nobody has filled in `actor` yet produces a clean Gu with
zero obligations, and the `checked:` line says so out loud rather than reading as full coverage.

## The persona-walk sweep

The ledger is authored by one named sweep, during Phase 2, before Gate 2.

For every human persona in the glossary, walk that persona's complete action list and name the
surface for each act. Persona by persona, not entity by entity: the entity walk is the one the
design already does, and it is precisely the walk that misses this, because it groups acts by the
thing being acted on rather than by the person acting. A persona walk asks "what can a tenant
administrator do, and where does each of those live", and the missing screen is obvious the moment
the question is asked in that order.

A surface is free text, and it should be the concrete thing: `Admin console > Tenants > Suspend
tenant`, `admin CLI: tenant suspend <id>`, `POST /v1/tenants/{id}/suspend`, `tenant settings
release, Billing tab`. What matters is that a reader can tell whether it exists.

Configuration knobs get rows too, under the `knob:<key>` form. A knob a person is expected to
change is a surface obligation like any other, and a design that assumes an operator can turn
something on has assumed an interface.

Phase 2 is the right moment because the surface is an architecture question (it decides screens,
commands, routes, and releases) and because the answer feeds the boundary and placement decisions
the same phase makes. Author it before Gate 2, and carry the coverage question into the
phase-exit self-review.

## The artifact

`design/surfaces.yaml`, at the design root beside the domain model it resolves against.

```yaml
surface_version: 1
sources:
  - domain.modelith.yaml action list, walked persona by persona
  - glossary personas: Seller, TenantAdmin, Auditor
acts:
  - act: Deal.create
    actor: Seller
    surface: Deals > New deal screen
  - act: Deal.approve
    actor: TenantAdmin
    surface: Admin console > Approvals queue
    milestone: M2
  - act: Tenant.suspend
    actor: TenantAdmin
    surface: "admin CLI: tenant suspend <id>"
  - act: knob:billing.grace_period_days
    actor: TenantAdmin
    surface: Tenant settings release, Billing tab
deferrals:
  - act: Deal.reopen
    reason: Reopening is a support-ticket path until M4; no self-serve surface is designed yet.
  - act: actor:Auditor
    reason: The auditor portal is a separate program of work, scoped after cutover.
```

### Root keys (strict; unknown keys fail the gate)

| key | required | meaning |
|---|---|---|
| `surface_version` | yes | the integer `1` |
| `sources` | yes | non-empty list of strings naming where the act list was enumerated from (an enumeration with no named source is a completeness claim with no evidence) |
| `acts` | yes | the list of acts and the surfaces that carry them |
| `deferrals` | no | acts and personas deliberately left without a surface, with reasons |
| `_comment` | no | free text |

### `acts` rows

| key | required | meaning |
|---|---|---|
| `act` | yes | `Entity.action` as the domain model spells it, or `knob:<key>` for a configuration knob |
| `actor` | yes | who performs it; for an `Entity.action` row it must equal the actor the model declares |
| `surface` | yes | the screen, admin command, API route, or config release that carries the act |
| `milestone` | no | when the surface lands; non-empty when present, and once BUILD.md declares milestones it must resolve to one of them by number (`M2`, `m2`, `2`, or the padded form as the plan writes it) or by title |
| `_comment` | no | free text |

### `deferrals` rows

| key | required | meaning |
|---|---|---|
| `act` | yes | `Entity.action`, `knob:<key>`, or `actor:<Name>` to defer one persona wholesale |
| `reason` | yes | why no surface is named; an unexplained gap is not a deferral |
| `_comment` | no | free text |

`actor:<Name>` is the honest form for a persona whose whole interface is out of scope for now. It
discharges every obligation that persona carries, in one row a reviewer can see, instead of
scattering the same reason across a dozen act-level deferrals.

## The gate

`Gu-surfaces` activates automatically when `design/surfaces.yaml` exists, the same way Gs
activates on `legacy/surface.yaml`. Run it alone while authoring:

```bash
machinery check design --gate gu
```

It verifies, deterministically:

1. **Schema.** Strict keys at every level; an unknown key is an error, because a typo silently
   drops the field it meant to set.
2. **Completeness.** Every obligated action is covered by an `acts` row that names it exactly, or
   by a deferral (its own, or its persona's). An act that is neither is an ERROR naming it.
3. **Resolution.** An `Entity.action` row must name an entity and action the target model
   actually declares, and its `actor` must equal the actor the model declares for that action. A
   ledger that misattributes an act is worse than no ledger: it closes the review question with
   the wrong answer.
4. **Uniqueness.** Every act value is stated exactly once, across `acts` and `deferrals` together.
   An act both mapped and deferred is a contradiction, not two statements.
5. **Persona deferrals resolve.** `actor:<Name>` must name an actor some action in the model
   declares, so a deferral for a persona who does not exist cannot silently discharge nothing.
6. **Knob rows** resolve against nothing (configuration is an open set by design), so the gate
   holds their shape and their uniqueness only.
7. **Milestones resolve.** Once `BUILD.md` declares milestones, a row's optional `milestone:` must
   name one of them: its number in any of the spellings the plan admits (`M2`, `m2`, `2`, and the
   padded form as written) or its title. An unresolvable name is an ERROR that lists the declared
   milestones, because "M2" surviving a replan that renumbered the plan reads like a commitment
   and points at nothing. The milestones come from the same root BUILD.md plan parse Gb and Ga
   use; execution packets cannot declare milestones, so the ledger can never bind to a milestone
   Gb does not hold. Before a plan exists the key is only held non-empty: the ledger is authored in Phase 2
   and the plan in Phase 4, so a ledger legitimately runs ahead of it.

The `checked:` line prints six numbers, zeros included, plus a seventh once a plan declares
milestones:

```
checked: 12 obligated actions, 10 covered, 1 deferred acts, 1 deferred personas, 3 knob rows, 4 actorless actions, 6 milestone references resolved
```

`covered` counts obligated actions satisfied by an `acts` row, so it reads against `obligated
actions` as a coverage fraction. The deferral counts are row counts, and one persona deferral can
account for several uncovered acts, so the numbers deliberately do not sum: the point is to show
how much of the model's human surface is named, how much was explicitly punted, and how much of
the model has not yet said who acts at all.

LLM-attested (the conductor checks these; the tool cannot): that each named surface is the RIGHT
one and that it will actually exist, that the deferral reasons are decisions rather than
placeholders, and, above all, that the model's `actor` fields are filled in. The gate holds the
ledger to the model; nothing but the persona walk holds the model to reality.

## Staging

Gu needs the domain model to resolve acts, so the earliest useful run is Phase 1, and the
intended authoring moment is Phase 2. On a model whose actions do not carry `actor` yet, the gate
runs clean with zero obligations and says so in a note; that is the signal to go back to Phase 1
and answer who performs each action, not a pass.

## Relationship to the other gates

- **Gs-surface** covers the legacy system's observable surface; **Gu-surfaces** covers the target
  design's. A rebuild wants both: Gs proves nothing was dropped by accident, Gu proves the
  replacement is reachable.
- **Gx-trace** holds the persistence-placement table to the entity list. Gu holds the surface
  ledger to the action list. Both work the same way and for the same reason: the set is closed, so
  the coverage question can be asked mechanically rather than trusted.
- **Gp-policy** answers who is ALLOWED to perform an act. Gu answers how they perform it. A design
  can pass Gp with a perfectly specified authorization rule protecting an act no interface exposes.
- Gu carries no generated artifacts, so it has no DRIFT class; the ledger is a source file.
