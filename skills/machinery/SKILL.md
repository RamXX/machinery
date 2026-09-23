---
name: machinery
metadata:
  version: "0.9.0"
description: >
  Design software as a build-ready blueprint for greenfield, brownfield,
  rebuild, or hybrid work. Use for domain modeling, C4 architecture, state
  machines, formal verification, legacy adoption, and zero-context BUILD
  handoffs under hard TDD. Do not use for ordinary implementation-only tasks.
---

# machinery

Turn product intent and existing evidence into a checked design that an
implementation agent can execute under hard TDD. The deliverable is the design,
not production code.

This entrypoint is deliberately short. Read only the references selected by the
current phase; do not load the entire methodology at once.

## Runtime contract

Use capabilities, not host names. The skill, artifacts, and `machinery` CLI are
portable. Subagents and hooks are optional accelerators:

- Use fresh-context `machinery-fsm-author` and `machinery-build-writer` roles
  when available; otherwise execute their installed instructions inline.
- Hooks may protect generated files and run checks. Without hooks, treat
  generated artifacts as read-only and run checks explicitly.
- In archaeology, use codebase-memory graph tools before broad filesystem
  discovery whenever they are available.
- Keep artifacts free of host-specific commands, models, sockets, or metadata.

The design contract is identical across Claude Code, Codex, OpenCode, and other
Agent Skills runtimes.

## Start and route

1. Read `design/STATE.md` and `design/DECISIONS.md` if they exist.
2. Classify the run: greenfield, brownfield, rebuild, or hybrid.
3. State the system purpose, users, target languages, and current open decision.
4. Read the reference for the work now in front of you:

| Work | Required reference |
|---|---|
| Domain model | the installed `domain-model-author` instructions; use `modelith lint` |
| Architecture and boundaries | [references/c4-standalone.md](references/c4-standalone.md) |
| Human-facing actions | [references/target-surfaces.md](references/target-surfaces.md) |
| Machine authoring | [references/xstate-format.md](references/xstate-format.md) |
| Formal, checker, attestation, acceptance, or import findings | [references/verification-evidence.md](references/verification-evidence.md) |
| Small build handoff | [references/build-md-template.md](references/build-md-template.md) |
| Large or small-model execution handoff | [references/execution-packets.md](references/execution-packets.md) |
| Existing-code intent | [references/archaeology-classification.md](references/archaeology-classification.md) |
| Rebuild or hybrid migration | [references/rebuild-guide.md](references/rebuild-guide.md) |
| Legacy capability inventory | [references/surface-ledger.md](references/surface-ledger.md) |
| CLI commands and gate names | [tools/README.md](tools/README.md) |

Do not inspect Machinery source to discover an artifact schema. The selected
reference must be sufficient. If it is not, report the missing instruction as a
Machinery defect instead of reverse-engineering a private contract into the
client design.

## Traceability spine

Hold these exact relationships through every phase:

| Domain | Behavior | Architecture |
|---|---|---|
| status enum value | state | owning component |
| action | event and transition | component performing it |
| invariant id | guard or structurally impossible edge | enforcing component |
| attribute | typed context reference | datastore schema |
| side effect | invoke with success, failure, and timeout | dependency plus mitigation |
| scenario | path and test case | participating interfaces |

Modelith is the single source of truth for data. C4 and machines reference it;
they do not redefine it.

## The four handoffs

### Domain

Interrogate breadth-first: vocabulary and entities, then invariants/actions,
then happy and negative scenarios. Sweep ubiquitous, event-driven,
state-driven, optional, and unwanted behavior. Every lifecycle has a status
enum; every invariant has an owner and a carrier or reasoned waiver.

Run `modelith lint` and `machinery check <design> --gate gc`. Do not advance on
errors or warnings.

### Architecture

Author `workspace.dsl`, `ARCHITECTURE.md`, and `surfaces.yaml`. Fix component
ownership, allowed and denied crossings, interface shapes, persistence,
concurrency, dependency failures, mitigations, NFRs, and human surfaces. Treat
technology adoption as a closure, not a product name.

For existing systems, finish the intent classifications before this handoff.
An unresolved corpus area blocks architecture; missing classification never
means must-port.

When the event contract arms consumer read completeness, every `(event,
consumer)` edge owes its own declaration: one `READS{field, ...}` group on a
payload-matrix row whose consumer column names that exact participant, or a
`(no reads: <reason>)` waiver in that consumer cell. One consumer's declaration
never covers a sibling and never transfers its waiver. A row carries exactly one
complete group, so join a group wrapped across lines, and every row and machine
stating one edge must agree on the exact field set. Members are ubiquitous
language, not code symbols. `READS` in ordinary prose stays prose.

When implementation code parses a design artifact during compilation or build, declare that
dependency with a root `reads:` row in the Architecture Contract: design-relative `artifact`,
implementation-relative `reader`, and the full reviewed Git commit. This lower-case YAML grammar is
not the event-payload `READS{...}` grammar above. `Gr-reads` warns in a design-only run and, with
`--impl`, fails when an artifact-changing commit did not change its declared reader in the same
commit. The exact closed grammar is in `docs/declared-reads.md`.

When a matrix event row restates a complete payload, use exactly one
`payload {field, ...}` or `payload is exactly {field, ...}` group. It binds to
that row's one event, and Gy-rules requires exact set equality with the
Architecture Contract payload cell. Payload prose without a group defines
nothing.

Every action whose Modelith actor is `System`, and every action a matrix row
declares in `PRODUCES{}`, needs an authorization admission: one row of a marked
hand-written inventory (`AUTHORIZATION.md` carrying the
`<!-- machinery:authorization-inventory -->` marker), covered by g2
attestations. The row's subject is the `Entity.action` id and its admission is
one backticked capability declared in `workspace.dsl`, or
`(no authorization: <reason>)`. A `System` action whose matrix unit declares
`WRITES{}` is read-only and owes no row. Gy-rules reports a missing, orphan, or
unresolvable admission; Gx-trace reports a malformed row. A design with its own
authorization notation generates these rows from its own reader (see the
migration note below).

Run `machinery check <design> --gate g2,gu` plus each artifact-activated gate,
and `machinery verify-c4 <design>`. Record the required attestation rows.

### Behavior

Author one machine per lifecycle or operational envelope, its named-unit
matrix, its oracle, and its formal semantics. Every dependency failure remains
a transition even when architecture mitigates it. Every fully guarded handler
states refusal behavior; every resting state states ignored events.

A guard's falsifying-clause vocabulary is one `CLAUSES{...}` group on that
guard's own row of the named-unit contract table, with at most one `RETIRED{...}`
group inside the same declaration. Matrix prose that quotes a vocabulary is
narrative, not a declaration. A declaration binds to its own machine:
`Alpha.matrix.md` binds only `Alpha.machine.json` and `Alpha.oracle.md`, so two
machines may reuse one guard name, and a declaration whose oracle rows only a
sibling machine could supply is an error naming that sibling. A declared guard
no oracle governs, such as one on a creation edge, owes nothing.

A fact a unit reads or names is declared in `USES{}` (and a stored fact it
writes in `WRITES{}`); a backticked token in prose is quotation, never a fact.
Each member resolves to a Modelith attribute (`Entity.attr`) or enum member, a
machine context key, an event payload field, or, on the same row, a `VALUES`
member or a `derived: fact_name (<reason>)` waiver for a unit-local computed
fact. The reason is mandatory and the waiver does not declare the fact for any
other row.

A closed vocabulary is one `VALUES{a, b, c}` group on a named-unit row.
Different units sharing a vocabulary name it with `VALUES reason_class{a, b, c}`.
A group binds to a Modelith enum by exact name and must then spell exactly the
enum's members: `VALUES OrderState{...}` binds enum `OrderState`, and an
unnamed `VALUES{...}` binds only when the unit name equals the enum name
exactly. There is no case folding and no fuzzy matching: `VALUES{...}` on unit
`orderState` binds nothing, and Gl-ledger warns on exactly that case, telling
you to write `VALUES OrderState{...}`. Prose that calls a vocabulary closed
declares nothing.

Five more groups state what a unit touches. Gx-trace parses them and fails a
malformed one; Gy-rules (below) decides on them. Members use the dotted
identifier grammar of payload fields; a group opens and closes in one table
cell; a row carries at most one group of each name.

- `WRITES{Order.status, OutboxMessage.status}`: the stored facts the unit
  writes. `WRITES{}` states a read-only unit.
- `USES{Order.totalCents, LineItem.quantity}`: the facts the unit reads or
  names. An empty `USES{}` is an error; omit the group instead.
- `PRODUCES{Order.markPaid}`: on the matrix row that names a cascade or
  consumer arm, the Modelith actions that arm performs. Each member is one
  `Entity.action` the model declares, and each owes an authorization
  admission whatever its Modelith actor. An empty `PRODUCES{}` is an error.
- `CARRIES{column:Order.status, outbox:OutboxMessage}`: what carries an
  action's or actor's effect. Kinds are `column`, `outbox`, `sink`, `signal`,
  and `action`.
- `SUPERSEDES{type:LegacyDeal}`: on an Architecture Contract row, never a
  matrix row, the stable type id this row replaces. Only `type:` exists.

An upper-case word directly followed by `{` in a matrix table cell that names
none of `CLAUSES`, `READS`, `VALUES`, `ORACLESET`, `WRITES`, `USES`,
`PRODUCES`, `CARRIES`, or `SUPERSEDES` is an error, so a misspelled or private group never
passes as prose. Gl-ledger warns on a backticked snake_case or `Entity.attr`
token in a contract, clause, or payload cell that sits outside every group and
that the row does not declare: declare it in `USES{}` or `WRITES{}`, or drop the
backticks.

Gy-rules (`--gate gy`, active on a design with `machines/` or an
`AUTHORIZATION.md`) evaluates the shipped Datalog rules under
`rules/consistency/` over the design's projected facts. Every `finding_*`
result is an ERROR: an uncovered System or produced action (`authz_missing`),
a stale or unresolvable admission (`authz_orphan`,
`authz_unknown_capability`), a `PRODUCES{}` member the model does not declare
(`produces_unknown_action`), a `USES{}`/`WRITES{}` member no declaration
resolves (`fact_unresolved`), a `VALUES` group disagreeing with the
same-named enum (`values_disagree`), a `payload {}` twin out of step with its
contract row (`payload_twin`), an actor with no `CARRIES{}` or a writing
action with none (`effect_uncarried`), `CARRIES{}` on a unit that is neither
an action nor an actor (`carrier_misplaced`), and supersession cycles,
dangling replacements and duplicate owners. `machinery check <design> --gate gy --explain`
prints under each finding its derivation: the rule file and rule number, then
the facts it matched with their `path:line` sources.

Run `machinery oracle`, `machinery check <design> --gate g3`, and
`machinery verify-formal <design>`. Read the verification reference for all
four semantics patterns, including `control-flow-only`.

#### Migrating from the 0.9.0 prose inference

machinery 0.9.0 inferred facts, writes, read-only units and producers from the
wording of matrix cells. That inference is gone; a design states each of them.
Gx-trace warns, for one release, on a design that declares no `WRITES{}`,
`USES{}` or `PRODUCES{}` anywhere and quotes a fact-shaped token in prose.

- A fact the prose backticked: declare it in `USES{}` on the row, or drop the
  backticks if it was only a quotation.
- A unit the prose called read-only ("writes nothing"): declare `WRITES{}`.
  A unit that writes: `WRITES{Entity.attr, ...}`.
- A producer the 0.9.0 check read from a cascade or consumer table's producer
  column: `PRODUCES{Entity.action}` on that row, plus its `AUTHORIZATION.md`
  row.
- Every actor, and every action with a non-empty `WRITES{}`: `CARRIES{}` naming
  the column, outbox event, sink, signal, or other machine's unit that carries
  its effect.
- A `VALUES{...}` group whose unit name matches its enum only up to case: name
  it, `VALUES Enum{...}`.
- A design with its own authorization notation (resource action lists, producer
  marks, residual verb tables): machinery no longer reads it. Generate the
  `AUTHORIZATION.md` rows from the reader the design's own tooling already has,
  and declare the file and that reader in a root `reads:` row so Gr-reads binds
  the pair.

### Build handoff

The build plan must be executable without design archaeology. Full mode is for
a genuinely narrow project. Large work uses a root milestone/demo manifest and
declares its linkage. Pairwise linkage gives every milestone exactly one bounded,
self-contained packet for a smaller execution model. Matrix linkage gives
milestones and reusable domain shards an exact reciprocal many-to-many graph;
each execution unit is one milestone-shard pair and reads the root plus that
workstream shard. When root plus shard exceeds the executor's window, author
`design/slices.yaml` (one shard, a byte budget, the element ids each slice
cites, and optional implementation-relative fixture modules that obligate its suite) and hand the executor `machinery packet <design> --milestone <id> --out
<dir>` output instead: only what the slice cites, every excerpt under its stable
id and source path:line, held by `Gw-packet` so no obligation of the milestone is
dropped. The slice map is authored; the packets are generated and never edited.
When slices share a fixture, declare that path on every consuming slice. Each
packet then states the full set of suites that must run after the fixture changes.

Run the full `machinery check <design>` and, once code exists,
`machinery check <design> --impl <dir>`. A green design is the RED precondition;
the locked tests and the same check are the GREEN acceptance boundary.

Executable test assurance is a contract, not a command, in this release. There
is no `machinery tdd`, `machinery check` accepts no `--store` or
`--assurance strict`, and no gate reads `design/assurance/`, so a committed
`plan.json` changes nothing the check reports. Bind tests to oracle rows through
`Gt-tests` stable ids instead. `Gt` credits static discovery of active test
references and executes nothing, so commented-out tests, unused declarations and
uncalled helpers earn no coverage. `docs/test-assurance-contract.md` states which
surfaces ship and which are the target.

## Evidence and closure

- Generated artifacts are committed with their sources and never hand-edited.
- `attestations.yaml` covers the full subject inventory for every judgment, and
  every row carries an explicit kind. `plan` is a design-time judgment. `current`
  is a reviewed implementation: it requires a complete implementation subject and
  `machinery check <design> --impl <root>`, and a subject whose bytes move
  invalidates the review. `historical` records an acceptance-file judgment and
  establishes history, not current approval. A design with no implementation yet
  is plan-only: it warns that the current review is missing and still exits 0.
- A milestone closes only with `acceptance/M<n>.yaml` bound to a reviewed commit.
- If a design publication is interrupted, `machinery recover <design-dir>`
  reports it read-only: expected outputs with content and mode status, journal
  and residue locations, live-writer status, and the safe recovery decision.
  `--apply` finalizes only a fully revalidated publication, and a refusal
  preserves every piece of evidence.
- Brownfield oracle failures are adjudicated as code-is-truth or
  model-is-truth; neither is silently normalized.
- Run `machinery check` with zero errors, drift, or warnings before handoff.
- If a Paivot nd story owns the work, append the current `nd_contract` with
  command evidence and per-AC proof; use the standard story transition commands.

## Conversation and decisions

Speak to the user about intent, design choices, risks, and deliverables in plain
language. Keep phase numbers, gate ids, schemas, and CLI detail in artifacts and
operator output unless the user asks for them.

Batch questions, echo load-bearing interpretations, and stop interrogating when
the current handoff is decided and clean. Record decisions at answer cadence in
`DECISIONS.md` as `<date> <who>: <decision>`. Author-proposed decisions remain
explicitly unconfirmed. Record phase status, open questions, check evidence, and
the five-part adversarial self-review in `STATE.md`.

## Scale and decomposition

Use milestone packets before recursive decomposition. Recurse only when the
domain language forks, a subsystem team needs an isolated contract, or formal
composition no longer scales. Contract-pack children remain complete Machinery
designs. Boundary changes are parent-owned pack amendments, never unilateral
child edits.

For very large designs, `machinery scale <design>` informs the choice; team
isolation remains a human decision it cannot infer.

A decomposed parent with no `machines/` still owes its own tests: with `--impl`
supplied, the default selection runs `Gt` whenever the parent owns relational
obligations (a policy or isolation annotation, or a committed relational oracle
under `formal/`). Those obligations belong to the parent, never to a child.

## Upgrade discipline

Upgrade the binary, skill, roles, plugin manifests, and generated artifacts as
one versioned change. Run `machinery doctor`; any stale cache, invalid receipt,
or mismatched skill version is a release blocker. Regenerate artifacts in a
dedicated upgrade change and classify stable-id churn before design changes.

`machinery update [--version <tag>]` refreshes the binary and the complete
recorded home, native-target, and plugin plan; rerunning the one-line installer
over an existing install converges on the same plan. A failed host plugin
refresh is a returned failure naming the exact retry, not a warning: host plugin
caches remain host-owned, outside machinery's rollback, and must be refreshed
through the host plugin manager.

A release publishes a cross-compiled `machinery-windows-amd64` artifact, but the
one-line installer and `machinery update` refuse Windows, so that asset is placed
by hand. No native Windows runtime guarantee is claimed: process custody, formal
verification, and the assurance lanes are unix-only. Use Linux or macOS for the
full toolchain.
