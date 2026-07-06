# The policy layer: a practical guide

This guide is for someone who wants the benefits of the policy layer without first learning Alloy,
relational logic, or the machinery internals. It covers what the layer is, when to use it, the
complete annotation reference, how to read a failure, how to wire the implementation test, and how
the workflow differs on greenfield and brownfield projects.

## What it is, in one minute

Access-control rules (roles, ownership, team scoping) are **static relations**: they constrain
which configurations of users, teams, and records are legal, not how the system moves between
states. That puts them outside both of machinery's other nets: TLC model-checks behavior over
time, and the domain-model linter checks structure, so a policy rule that is well-formed prose
with a hole in its meaning passes everything. Meanwhile the rules are exactly what every coding
agent must interpret on every authorization-adjacent change, and prose interpretation is where
multi-agent drift starts.

The policy layer replaces interpretation with generation. One annotation file states what the
policy invariants mean in a small closed vocabulary. From it, `machinery alloy` generates:

1. **`design/formal/Policy.als`**, a relational model plus a standard suite of meta-checks. The
   Alloy analyzer exhaustively searches every configuration within a bound. A failed check comes
   back as a concrete counterexample: an actual arrangement of users, teams, and records that the
   rules as written permit but should not.
2. **`design/formal/Policy.oracle.md`**, the authorization oracle: the same policy enumerated as a
   decision table (role x verb x ownership case, expected verdict per row, stable ids). Your
   implementation's test suite asserts its authorization function against every reachable row, the
   same pattern as the machine transition oracles.

Both artifacts are generated, committed, byte-diffed for freshness by the **Gp-policy** gate, and
protected from hand edits by the plugin hooks. Nobody writes Alloy, and nobody hand-maintains the
decision table.

## Do I need it?

Use the layer when the domain model carries access-control invariants: statements shaped like "a
`<Role>` may `<verb>` records `<scope>`". The signals are a role enum on an acting entity,
ownership relationships from records to that entity, and cross-cutting `rbac-*`-style invariants.

Skip it when there is no such policy. Two of the bundled examples skip it legitimately:
fulfillment's cross-cutting invariants are saga and money-conservation properties (the TLC rung
owns those), and portfolio-engine's role language lives only in its glossary, with no modeled
subject entity to bind to. A design without the annotation file never runs any of this; outputs
are byte-identical to a machinery without the layer.

One honest boundary up front: the v1 algebra covers role plus ownership plus team scoping. It does
not express attribute-based conditions ("only during business hours"), role hierarchies, or
cross-entity conditions. Rules like that become named residuals (see below), enforced by tests
like any other code-level property. If your policy is mostly residuals, this layer is the wrong
tool for it, and that is worth knowing before you start.

## The five-minute walkthrough

Using the go-crm example as the template:

```yaml
# design/formal/policy.relational.yaml
subjects:
  entity: User                # who acts
  role_attr: role             # its role enum attribute (UserRole)
  team:
    entity: Team
    membership: lone          # a User holds at most one team; zero is legal...
    required_for: [Manager]   # ...except a Manager, which must hold one
    invariant: [single-team, manager-has-team]
resources: [Account, Contact, Deal, Task, Activity]   # what is acted on
owned_invariants: [account-owned, contact-owned, deal-owned, task-owned, activity-owned]
rules:
  - invariant: rbac-crud-verbs          # which verbs each role holds at all
    grants:
      Admin: [create, read, update, delete]
      Manager: [create, read, update, delete]
      Rep: [create, read, update, delete]
      ReadOnly: [read]
  - invariant: rbac-read-visibility     # who may read what
    verbs: [read]
    scope: {Admin: all, "*": own | team}
  - invariant: rbac-write-scope         # who may change what
    verbs: [update, delete]
    scope: {Admin: all, Manager: team, Rep: own, ReadOnly: none}
  - invariant: [rbac-reassign-authority, task-assignee-visible]
    reassign:                           # who may change an owner, and where it may go
      scope: {Admin: all, Manager: team}
      target: {Admin: any, Manager: team}
residuals:                              # what the algebra cannot carry, waived with a reason
  - invariant: session-active-user
    reason: behavioral; enforced by the Session machine and checked at the TLC rung
```

Then:

```bash
machinery alloy design/            # generate Policy.als + Policy.oracle.md; commit both
machinery check design/            # Gp-policy: binding, coverage, freshness (no Java needed)
machinery verify-formal design/    # run the Alloy meta-checks next to the TLC proofs (needs Java)
```

Finally, wire the conformance test in the implementation (one test, once; see "The oracle test"
below). Done: the policy is now solver-checked at design time and test-enforced at build time.

## The annotation reference

The file is `design/formal/policy.relational.yaml`. Unknown keys anywhere are hard errors (a typo
must not silently weaken the policy). Every referenced entity, attribute, enum value, and
invariant id must exist in the domain model; every disagreement fails generation with a message
naming the mismatch. This is the same rule the data-refinement annotations follow: a drifted
annotation fails, it never proves a stale twin.

### `subjects` (required)

| key | meaning |
|---|---|
| `entity` | the acting entity (must be a Modelith entity) |
| `role_attr` | an attribute of that entity whose type is an enum; its values are the roles |
| `team.entity` | optional: the team/tenant entity; requires a relationship to the subject entity |
| `team.membership` | `lone` (at most one team, zero legal) or `one` (exactly one). Required when team is declared: Modelith's `1:n` cardinality cannot express which, and the difference is load-bearing |
| `team.required_for` | roles that must hold a team even under `lone` (redundant, and rejected, under `one`) |
| `team.invariant` | the invariant id(s) this multiplicity carries |

### `resources` (required)

The entities the policy governs. Each must have an `n:1` relationship to the subject entity (that
relationship is the ownership the scopes quantify over). The generated model collapses them into
one `Record` signature because the v1 algebra treats resources alike; the oracle test still runs
every row against every resource entity type.

### `rules` (required, one per policy invariant)

Each rule carries `invariant:` (one id or a list) and exactly one shape:

- **grants**: `{Role: [verbs]}`. The verb capability map. Vocabulary: `create`, `read`, `update`,
  `delete`. At most one grants rule per design. A role absent from the map holds no verbs.
- **scope**: `verbs: [...]` plus `scope: {RoleOrStar: expr}`. Scope expressions: `all`, `none`, or
  unions of `own` and `team` (for example `own | team`). `"*"` binds the roles not named
  explicitly. Each of read/update/delete may be scoped by at most one rule; `create` takes no
  scope (there is no record yet).
- **reassign**: `scope: {Role: expr}` for authority over the record, plus a **mandatory**
  `target: {Role: any | team}` deciding where the record may go, per role with authority. The
  target must be stated, not implied: leaving it implicit is precisely the under-specification the
  layer exists to catch.

### `residuals`

Every top-level (cross-cutting) invariant the rules do not compile must appear here with a
`reason`. This is the coverage rule, and it is the layer's anti-drift ratchet: an agent adding a
cross-cutting invariant must either compile it or explicitly waive it, or generation fails. A
waiver is visible in the gate output (`residuals (waived with reason)`), so residual creep is
measurable. Entity-level invariants are not forced through this rule (their enforcement is
machine guards and matrices), but the ones the annotation does carry (team multiplicity,
ownership) count as enforcement for Gx-trace.

### `scope`

Optional integer 2 to 12 (default 6): the Alloy search bound, the maximum number of atoms per
signature. Checks are exhaustive within the bound and silent beyond it; the generated header
states this.

## The generated meta-checks, in plain language

Every design gets the same suite; none of it is authored per design.

| check | question it answers | a FAIL means |
|---|---|---|
| `SomeWorld` (run) | can every role be inhabited with records present? | the rules contradict each other; everything else would pass emptily |
| `WriteImpliesRead` | can anyone write a record it cannot read? | write scope escapes read scope |
| `CapableWritesOwn` | can every write-capable role write the records it owns? | an unstated assumption: some legal subject is locked out of its own records (this is the check that caught the teamless Manager) |
| `ReassignRetainsAuthority` | can a legal reassign push a record beyond the actor's own authority? | a one-step escape hatch in the reassign rule |
| `Possible_<Role>_<verb>` (runs) | is each granted verb actually exercisable in some world? | a vacuous grant: the role holds a verb whose scope is empty everywhere |

`verify-formal` prints one PASS/FAIL line per command. A failed check includes the counterexample
in domain vocabulary, for example:

```
FAIL  Policy/CapableWritesOwn
      counterexample: User$5{role=Manager$0, team=(none)} Record$0{owner=User$5}
```

Read it literally: here is a Manager with no team owning a record; work out which rule denies it
the write. The fix is always in the domain model (usually one invariant tightened or reworded),
then `machinery alloy` regenerates and the suite goes green. Never edit the generated files; the
hooks and the DRIFT gate will refuse it anyway.

## The authorization oracle and its test

`Policy.oracle.md` enumerates the policy as decision rows:

```
| test id    | stable id    | verb     | role    | owner case | target          | expectation | invariants |
| O-AUTHZ-57 | AUTHZ-fbc244 | reassign | Manager | own-teamed | target-teammate | allow       | rbac-reassign-authority, task-assignee-visible |
```

Because the algebra decides every case from two booleans (does the actor own the record; do actor
and owner share a team), the table is the complete semantics of the policy, not a sample. The
owner-case vocabulary (`own-teamed`, `own-teamless`, `teammate`, `outsider`, and the concrete
variants each expands to) is defined in the file's own header. Stable ids hash the case and never
the verdict: when the design changes, expectations flip under unchanged ids, and your test diff
names exactly which cases changed behavior.

Rows marked `unreachable` are configurations the domain invariants forbid (for example a teamless
Manager once `manager-has-team` exists). Tests skip them; the write discipline that refuses to
construct them is a separate, named enforcement row in BUILD.md.

**The oracle test** is one test, written once, that parses the table and asserts the pure
authorization function on every reachable row, expanding each abstract case into all of its
concrete variants and running every row against every resource entity type. The reference
implementation is `examples/go-crm/impl/internal/authz/oracle_test.go` (about 180 lines including
the parser). Two rules make it work anywhere:

1. Keep authorization decisions in one pure function (no I/O), so the oracle rows map onto direct
   calls. Reassignment needs the target in the signature; if your authorize function cannot see
   the new owner, the target rule has no home, which is itself a finding.
2. Key test names on the stable id, so failures survive design revisions and renumbering.

The loop this closes: change a policy invariant, regenerate, and the conformance test fails on the
named rows until the code follows (or the change is rolled back). Drift between policy and code is
now a red test in both directions. This was mutation-verified during development: loosening the
reassign target rule in the annotation flipped two oracle rows and failed the go-crm suite on
exactly those rows.

## Gates, hooks, and where each thing runs

| concern | held by | needs Java |
|---|---|---|
| annotation parses, binds to the domain model, covers every top-level invariant | Gp-policy (`machinery check`) | no |
| committed `Policy.als` and `Policy.oracle.md` byte-match a fresh generation | Gp-policy (DRIFT, blocking) | no |
| invariants compiled by the annotation trace as enforced | Gx-trace ("policy-checked" class) | no |
| the meta-checks actually hold | `machinery verify-formal` (pinned, checksum-verified Alloy 6.2.0 jar, fetched on first use) | yes |
| the implementation agrees with the oracle | the conformance test in your suite | no |
| nobody hand-edits generated artifacts | plugin hooks deny writes to `formal/*.als` and `formal/*.oracle.md` | no |

Gate selection is automatic: Gp runs when `formal/policy.relational.yaml` exists, otherwise never.
`--gate gp` forces it (and errors if the annotation is missing, because an explicitly requested
gate with nothing to check is a failure, not a pass).

## Greenfield workflow

The conductor handles this during Phase 1 (the skill calls it Phase 1.5): once the domain model
carries access-control invariants, it authors the annotation as part of the same interrogation.
Expect the annotation to generate questions, because that is its job: `membership:` forces the
"can a subject have no team?" decision, `required_for:` forces it per role, `target:` forces the
"where may a record go?" decision, and anything unstatable in the algebra forces a definition to
be pinned down before it can become a residual. Answer them in the domain model (real invariants
with ids), not in the annotation alone; the annotation carries ids, never new rules of its own.

Phase 4 then requires the oracle conformance test in BUILD.md, so the zero-context implementer
builds the enforcement loop from day one.

## Brownfield workflow

On an existing system, the annotation is archaeology like everything else: write the policy AS THE
CODE BEHAVES, reading the authorization code rather than the documentation. Then let the machinery
arbitrate:

- A failed meta-check means the system's actual policy has a hole the team probably does not know
  about. That is a finding to adjudicate, not a blocker.
- A failing oracle-conformance row means the annotation and the code disagree; decide
  code-is-truth (fix the annotation) or policy-is-truth (file the code defect), the same
  adjudication rule the transition oracles use for characterization tests.
- The go-crm history is the worked example: the faithful annotation reproduced two latent design
  defects, the domain model was amended, and the code was brought to conformance, all under gates.

The layer slots into the staged adoption ladder wherever Phase 1 lands; it needs no machines, no
contract, and no impl configuration, so it can arrive before G2/G3 on the gate list. See the
[brownfield team guide](brownfield-team-guide.md) for the ladder itself.

## Limits, stated plainly

- **Bounded, not unbounded.** Alloy proves properties for every configuration up to the scope
  (default 6 atoms per signature), not beyond. For policy algebras this small, that bound is far
  past where counterexamples live in practice, but the header of every generated model states the
  caveat and it should be repeated in any audit conversation.
- **Design-side plus test-side, not runtime.** The solver holds the invariant set; the oracle test
  holds the implementation. Nothing here intercepts requests at runtime; a code path that bypasses
  the authorization function bypasses all of it. Keeping authorization at a single call site is an
  architecture decision (go-crm's contract pins it), and G4-import is the fence around it.
- **The algebra is deliberately small.** `all | own | team | none`, one team dimension, one
  ownership dimension. Extensions (attribute conditions, hierarchies, multi-tenancy layers) should
  be driven by a real design that needs them, not speculatively. Watch the residual count in the
  gate output: rules should outnumber waivers by a wide margin, or the layer is decorating the
  policy rather than checking it.

## Troubleshooting

| symptom | meaning |
|---|---|
| `alloy_gen: top-level invariant(s) X are neither compiled by a rule nor waived by a residual` | the coverage rule: classify X as a rule or a residual-with-reason |
| `alloy_gen: subjects.team.membership must be 'lone' or 'one'` | the multiplicity decision cannot be inherited from Modelith cardinality; state it |
| `alloy_gen: reassign.target is required` | where a record may go must be stated per role with authority |
| `alloy_gen: resource X declares no n:1 relationship to Y` | the domain model does not say X is owned by Y; add the relationship or drop the resource |
| `DRIFT formal/Policy.als is stale` | sources changed after the last generation; run `machinery alloy <design>` and commit both artifacts |
| `FAIL Policy/SomeWorld` | the rules contradict each other; read the annotation for a role granted verbs whose every scope is `none`, or team facts that cannot coexist |
| a `run Possible_*` FAIL | that role's granted verb is exercisable nowhere: usually a grants/scope mismatch |
| oracle conformance failures after a design edit | intended: the code no longer matches the policy; follow the stable ids |
