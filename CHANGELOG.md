# Changelog

Notable user-facing changes to machinery. Release notes for generated-output and proof-scope
changes follow the discipline in [docs/release-notes.md](docs/release-notes.md); entries move
under their version heading when a release is cut.

## [Unreleased]

## [0.10.1] - 2026-09-24

**Why this release.** 0.10.0 was withdrawn the day it shipped because it blocked valid designs.
An event on several rows of the event-contract table (a fan-out) projected as a duplicate stable
id, so Gy-rules stopped at the projection and evaluated nothing; a design's own group notation in
its matrices (`OWNED-BY{...}`) was an unknown-group error with no way to keep it; and Gl-ledger
warned on every backticked action, enum value and file name as if it were an undeclared fact.
0.10.1 fixes those, removes one over-strict rule (`waived_machine_present`), and adds the Gy/Gl
baseline, so an existing design adopts the consistency layer by recording its current findings
and burning them down instead of fixing everything in one stop-the-world migration.

**Consumer-corpus diff (release safety, CONTRIBUTING).** `consumer-diff` was run with the v0.9.0
release binary against this candidate over the largest consumer design available to the
maintainer (a read-only clone of its pin branch with `private_groups:` declared and a Gy/Gl
baseline recorded): 0 unexplained new findings; new findings 0 blocking, 0 warnings, 505 notes (236
baselined Gy-rules, 268 baselined Gl-ledger lines, 1 migration note); 1 resolved blocking finding
(the 0.9.0 contract-schema rejection of `private_groups:`); the allow file held one entry for
version-stamp drift, citing the regeneration commands below, and it went unused because both runs
keyed the stamp lines identically. The same consumer's conductor ran the candidate independently on
a fresh clone and reported matching counts. The remaining blocking findings on that tree are its own
stale attestations after the `private_groups:` edit, which the pin commit re-attests.

### Added

- **Consistency-layer adoption baseline.** `machinery baseline <design> --gate gy,gl` records a
  design's current Gy-rules findings and Gl-ledger undeclared-fact warnings in
  `design/ratchet.json`, the file that already holds G4's baselined edges, so an existing design
  can adopt the layer and burn its debt down instead of fixing everything in one change. Two
  optional sections join `edges`: `rule_findings` (`{relation, tuple}`: the rule's output
  relation and the finding's subject ids) and `undeclared_facts` (`{file, unit, token, count}`);
  neither carries a line number. A recorded finding prints as `note   baselined: <finding>` and is
  counted on the gate's `checked:` line (`N baselined`); it never blocks or warns, so strict and
  `--warnings-as-errors` runs pass, and the stop hook classifies the same way. A finding the
  ratchet does not record reports exactly as before. A recorded entry no longer observed is a
  note (`... resolved; run machinery baseline to shrink the ratchet`), never an error. The first
  `gy` (or `gl`) run records every current finding; later runs keep only the recorded entries
  still observed, so that part of the ratchet only shrinks, unless `--grow` accepts new debt.
  `--impl` is required only with `g4` (the default, unchanged); for `gy` it adds the
  implementation's oracle bindings as `check --impl` does. The baseline refuses to record while
  Gy-rules has projection errors, which are a broken design, not debt. `machinery check
  --complete` refuses while any baselined Gy/Gl finding remains and prints the count per gate.
- **Compatibility.** A ratchet with only an `edges` section loads, renders and drives G4 byte for
  byte as before, and a plain `machinery baseline <design> --impl <dir>` prints and writes what
  it did (it now keeps any recorded `rule_findings` and `undeclared_facts`). A ratchet recording
  only Gy/Gl debt omits `edges`: G4 treats it as no snapshot, and it does not arm import blocking
  at turn end.

### Fixed

- **The pre-cutover deprecation is one design-level line, and a note under a recorded baseline.**
  0.10.0 reported "the design declares no WRITES{}, USES{} or PRODUCES{}" on one arbitrary matrix
  row, which read as a row defect; Gx-trace now prints one line per design counting the rows and
  files that quote fact-shaped tokens in prose and naming the first. A design whose `ratchet.json`
  records a Gy-rules or Gl-ledger baseline is adopting the layer on purpose, so the line is a note
  there and `--warnings-as-errors` passes on the pin commit; the declarations follow in the
  burn-down. Both points came from the first consumer run of the 0.10.1 candidate.

- **A fan-out event no longer fails the projection, and Gy-rules no longer evaluates nothing.**
  The event-contract table is one row per producer-consumer edge, but 0.10.0 projected
  `event(id, producer)` as a defining row per table row, so an event on several rows was a
  duplicate stable id and Gy-rules stopped at the projection. The events layer of projection 2.0
  is now `event(id)` (defined once), `event_edge(edge, event, producer, consumer)` with the
  content-derived id `<event>|<producer>-><consumer>` (one per producer-consumer pair of a row),
  `event_edge_payload_field(edge, field)`, and the non-defining sets `event_producer`,
  `event_consumer`, `event_participant` and `event_payload_field` (the union over the edges). One
  edge stated with two payloads is a projection problem naming both rows. External rule files
  that declared `event` with two columns must drop the second and read `event_producer`.
- **`payload {}` binds to the unit's own edges.** `payload.dl` compares a declaration with each
  edge whose producer or consumer is the unit's component (from the action-ownership table,
  projected as the new `action_owner(action, component)`), and names the edge in `payload_twin`.
  New findings: `payload_no_edge` (the component is on no edge of the event) and
  `payload_unknown_event` (the event has no contract row). A unit with no owning component is
  held to every edge, which for a one-row event is the 0.10.0 behavior.
- **Gy-rules degrades per row.** A row the projection cannot read (a malformed or unknown
  declaration group, an edge with two payloads) is one ERROR naming the row, the row's facts are
  omitted, and the rules run on everything else; the gate stays red, and every finding printed
  beside such an ERROR carries `[projection partial: ...]`. `machinery project` stays strict.
- **Parallel unnamed relationships are a model finding.** Two relationships between one entity
  pair with one cardinality and no role or name no longer fail the projection: they project as
  one tuple and Gy-rules reports them, naming both and asking for a `role:`.

- **A design's own group notation is declared instead of failing Gx-trace.** Under 0.10.0 every
  upper-case `NAME{...}` in a matrix table cell outside the public vocabulary was an
  unknown-group error, so a design whose own tooling reads group marks in its matrices (a mark
  such as `OWNED-BY{...}`) could not keep them, although the 0.10.0 notes said such a design kept
  its notation. The Architecture Contract v2 fence now accepts `private_groups: [OWNED-BY, ...]`:
  a non-empty list of upper-case group names in the scanner's own grammar (letters, digits,
  hyphen, underscore), each listed once and none a public group; declaring `WRITES` or `RETIRED`
  private is a G2 error, and an invalid entry is never honored. A declared private group is
  skipped by the declaration parser and the projection: it is not an error, it is not projected,
  its members declare nothing and satisfy no obligation, and Gx-trace counts the skipped spans on
  its `checked:` line. An undeclared unknown name is still an error, so `WRITE{...}` beside a
  declared `OWNED-BY{...}` still fails, and the message now lists every known group (`RETIRED`
  was missing) and names the `private_groups:` escape. The list lives in the contract, not in
  `.machinery.json`: `machinery check` never reads `.machinery.json` (only the hooks do), and the
  declaration changes what the design means, so it is reviewed with the design.
- **Gl-ledger resolves a backticked token before calling it an undeclared fact.** The 0.10.0
  warning fired on every backticked snake_case or `Entity.attr` token outside a group, including
  action names, machine-qualified units, enum values and file names (`ARCHITECTURE.md` has the
  `Entity.attr` shape). A token is now resolved against the model's attributes, actions and
  invariant ids, the enum values (and their snake_case form), the named units of every matrix
  (bare and qualified by machine), the machines' context keys, the event contract's event names,
  and file names (a known extension, or a file under the design). A model attribute keeps the
  warning ("declare it in USES{} or WRITES{}"); a token naming nothing gets a softer one ("is not
  a declared fact, action, unit or value; drop the backticks or declare it"); every other class
  is silent and counted on the `checked:` line. A matrix with more than 20 such warnings prints
  one summary line with the split and the first three tokens; `machinery check --verbose` prints
  every line, and the counts on the `checked:` line are exact either way.

### Removed

- **`waived_machine_present` (`records.dl`).** 0.10.0 reported a `(no machine: <reason>)`
  placement waiver on a component that also has a machine. That placement next to a small machine
  of the same name is an accepted convention (an envelope machine for a record-only entity) which
  0.9.0 accepted; the rule assumed the two are exclusive and would fail valid designs. The waiver
  still declares nothing contract-only when the machine exists, `orphan_matrix` (a matrix with
  neither a machine nor a waiver) stays, and so does contract-only record support. The `component`
  subject kind, which only that rule used, is gone from the Gy-rules subject vocabulary.

### Compatibility and migration

**From 0.10.0.** 0.10.0 is withdrawn; do not stay on it. Everything 0.10.0 accepted is accepted
here, except that an external rule file declaring `event` with two columns must drop the second
column and read `event_producer` (see Fixed). The declarations 0.10.0 introduced keep their
meaning.

**From 0.9.0: adopting the consistency layer.** A design green under 0.9.0 can arrive red on
Gy-rules and loud on Gl-ledger; that is debt the prose inference used to hide, and the 0.10.0
migration notes below say what to declare. Adopt in this order:

1. Pin the release: `machinery update --version v0.10.1` (or `MACHINERY_VERSION=v0.10.1` for the
   one-line installer), so every run of the adoption reads the same rules.
2. If the design's matrices carry its own upper-case group notation (`OWNED-BY{...}` and the
   like), list each name under `private_groups:` in the Architecture Contract v2 fence. Without
   it those marks are Gx-trace errors, and Gy-rules projection errors, which the baseline refuses
   to record.
3. Record the current debt: `machinery baseline <design> --gate gy,gl` (add `--impl <dir>` when
   you check with one, so Gy-rules also reads the oracle bindings), and commit
   `design/ratchet.json`. Recorded findings print as `baselined:` notes and never block.
4. Burn it down in ordinary changes: fix findings, watch the resolved notes, and rerun the same
   `machinery baseline` command to shrink the ratchet. It never grows without `--grow`, and
   `machinery check --complete` refuses while any Gy/Gl finding is still baselined.

**Proof scope.** A green Gy-rules on a design with baselined findings establishes the 0.10.0
scope for every finding the ratchet does not record, and nothing about the recorded ones;
`--complete` is the run that establishes the full scope. Removing `waived_machine_present` means
Gy-rules no longer asserts that a `(no machine: <reason>)` waiver and a machine are exclusive.
Gl-ledger's undeclared-fact warning now fires only for a model attribute or a token that names
nothing, so a quiet Gl-ledger no longer implies that no action, unit, enum value or file name is
backticked outside a group.

**Generated output.** The oracle, TLA+, Alloy, checker-projection, and packet generators now emit
the `v0.10.1` machinery stamp. This restamps the committed example oracles and formal artifacts,
the pii-flow checker projection, and the golden corpus, including its packet fixture. Run
`machinery oracle <design>/machines`, `machinery verify-formal <design> --gen-only` (which also
regenerates opted-in Alloy layers), and `machinery project <design>` for those example families;
run `make golden-update` to recapture the corpus. The release-preparation regeneration changes
only version stamps. Consumers should regenerate and commit their generated design artifacts on
upgrade.

## [0.10.0] - 2026-09-23

Withdrawn 2026-09-23; see 0.10.1.

### Added

- **Declaration groups `WRITES{}`, `USES{}`, `CARRIES{}` and `SUPERSEDES{}` (consistency layer,
  Stage 1).** Matrix rows may declare the stored facts a unit writes (`WRITES{}` alone states a
  read-only unit), the facts it uses, and what carries its effect (`column`, `outbox`, `sink`,
  `signal`, `action`); Architecture Contract rows may declare `SUPERSEDES{type:OldName}`. Gx-trace
  parses them into typed declarations and fails malformed, empty, duplicate, unterminated, nested,
  and repeated groups; Gy-rules decides on them.
- **Closed group-name vocabulary.** An upper-case `NAME{` in a matrix table cell that is not
  `CLAUSES`, `READS`, `VALUES`, `ORACLESET`, `WRITES`, `USES`, `PRODUCES`, `CARRIES` or
  `SUPERSEDES` is a
  Gx-trace error naming the row. In 0.10.0 that included a design's own group names, with no way
  to keep them; 0.10.1 adds the `private_groups:` contract declaration (see 0.10.1, Fixed).
- **Undeclared fact references warn in Gl-ledger.** A backticked snake_case or `Entity.attr`
  token in a matrix contract, clause, or payload cell, outside every group and not declared by its
  own row, is a warning: declare it in `USES{}` or `WRITES{}`, or drop the backticks.
- **Projection 2.0 (consistency layer, Stage 2).** A checker manifest may now include `actions`,
  `machines`, `matrices`, `events`, `c4`, `authorization`, `oracles`, `milestones` and
  `supersession`. Such a manifest gets `projection_schema: "2.0"`
  ([`schemas/projection-v2.schema.json`](schemas/projection-v2.schema.json)): the 1.0 `model` block
  when a v1 layer is included, plus `layers`, one array of rows per relation, each row with its
  stable id and a design-relative `source` `{path, line}`. Only identifiers and enumerated values
  are projected, never prose. `Gk` and `verify-checkers` regenerate 2.0 projections from the design
  and bind evidence to them exactly as for 1.0. A manifest naming only `model`, `invariants` and
  `relationships` keeps the 1.0 projection byte for byte, so every committed v1 projection and
  evidence file (`examples/pii-flow` among them) still binds. `scenarios` stays reserved, and a
  layer the design does not have fails loudly.
- **`machinery project <design> --facts <dir>`.** Writes every fact relation the design has as
  tab-separated `<relation>.facts` files plus a `relations.txt` index (name, arity, layer,
  columns): the input Souffle and the in-process evaluator read without an adapter. The directory
  is staged and renamed into place, refuses to replace anything but facts output, and is
  byte-identical across reruns of an unchanged design. No checker manifest is needed.
- **Gy-rules (consistency layer, Stages 3 and 4).** A new gate, `gy` in the `--gate` vocabulary,
  in the default suite and in the stop hook's selection, active on a design with `machines/` or an
  `AUTHORIZATION.md`. It evaluates the Datalog rules shipped under `rules/consistency/`
  (`authz`, `facts`, `values`, `payload`, `carriers`, `supersession`) with the in-process
  evaluator over the design's facts, built in memory (nothing is written). Every `finding_*`
  result is an ERROR reading `path:line: row 'X': <code>`: `authz_missing`, `authz_orphan`,
  `authz_unknown_capability`, `produces_unknown_action`, `fact_unresolved`, `values_disagree`,
  `values_conflict`, `payload_twin`, `effect_uncarried`, `carrier_misplaced`,
  `duplicate_owner`, `supersession_cycle` and `dangling_replacement`. A `warn_*` result would be
  a warning; none ships.
- **`PRODUCES{Entity.action}` (consistency layer, Stage 4).** The matrix row that names a cascade
  or consumer arm declares the Modelith actions that arm performs. Each member is one
  `Entity.action`; a produced action owes an authorization admission whatever its Modelith actor,
  and a member the model does not declare is `produces_unknown_action`. Projected as
  `unit_produces(unit, action)`.
- **`migration.yaml` owns the legacy types it disposes.** Every disposition naming a target
  projects `type_owner(<legacy>, migration.yaml)`, so `SUPERSEDES{type:<legacy>}` resolves; a
  retired legacy entity is claimed by nothing.
- **Gl-ledger warns when an unnamed `VALUES{...}` misses its enum only by case.** Groups bind to
  Modelith enums by exact name, and an unnamed group takes its unit's name; `VALUES{...}` on unit
  `orderState` binds no enum `OrderState`. The warning names the fix, `VALUES OrderState{...}`.
- **`machinery check --explain`.** Prints under each Gy-rules finding the derivation that
  produced it: the rule file and 1-based rule index, then the matched facts with their
  `path:line` sources.
- **`unit_declares(unit, group)` in projection 2.0.** One row per declaration group present on a
  matrix row (`WRITES`, `USES`, `PRODUCES`, `CARRIES`, `VALUES`, `CLAUSES`, `READS`, `payload`), so
  `WRITES{}` (declared read-only) is distinguishable from no `WRITES` group. The published
  `schemas/projection-v2.schema.json` gains the relation; it is now regenerated from the relation
  catalog by `go test ./internal/checker -run TestProjectionV2SchemaIsGenerated -update`.
- **CI `datalog-parity` job.** Runs every shipped rule file over every bundled example, and the
  evaluator's own program corpus, under both engines against Souffle 2.5 in an image built from
  pinned inputs (`scripts/souffle.dockerfile`), mirrored as `dagger call datalog-parity` and run by
  `make preflight` through `make dagger-job JOB=datalog-parity`.
- **Contract-only records (consistency layer, Stage 5).** An immutable or creation-time record has a
  named-unit matrix and no machine. Its ARCHITECTURE.md placement row declares that with
  `(no machine: <reason>)`, and that waiver is the only declaration: G3 accepts the matrix without
  a machine (and counts it), Gd accepts its `CLAUSES{}` without an owning machine or oracle, and
  Gy-rules reads the same waiver. `records.dl` adds `orphan_matrix` (a matrix with neither a
  machine nor a waiver) and `waived_machine_present` (a waiver on a component that has a machine).
  A contract-only clause set owes no suffixed transition id; its obligation is the assurance
  inventory's `guard-clause` key per active clause.
- **`RESERVED{type:Name}` (consistency layer, Stage 5).** An Architecture Contract row may reserve a
  type id as not yet defined; the reserving row owns nothing, and the group is an error in a
  matrix. `supersession.dl` adds `stale_reservation` (a reserved type some artifact owns) and
  `superseded_in_packet` (a `slices.yaml` `row:` citation whose key is a superseded type).
- **BUILD.md oracle binding table (consistency layer, Stage 5).** A BUILD.md table with an `oracle`
  column and a `bound at` column states where the suite binds each oracle id (test file paths
  relative to the implementation root, or `unbound`). Under `machinery check --impl`, `bindings.dl`
  compares it with the oracle rows each test file binds under Gt's credit rules:
  `milestone_binding_stale` (the table says unbound, a test binds it) and
  `milestone_binding_phantom` (the table names a path that binds nothing for the id). Without
  `--impl` neither fires.
- **Projection 2.0 relations for Stage 5.** `matrix(id)` (matrices), `no_machine_waiver(component)`
  (c4), `reserved(type, row)` (supersession), and `packet_cites(slice, row)`,
  `milestone_says_bound(oracle, path)`, `milestone_says_unbound(oracle)`, `bound_at(oracle, path)`
  and `test_file(path)` (milestones). `bound_at` and `test_file` are filled only by
  `machinery check --impl`; `machinery project` emits them empty. The schema is regenerated.
- **`rules/README.md`** lists every shipped rule file, its findings, and the declarations it reads.

### Changed

- **G3's orphan-matrix error names the contract-only declaration.** A matrix with no machine and
  no placement waiver still fails G3; the message now says that a contract-only record's matrix is
  declared by `(no machine: <reason>)` on its placement row. `Gy-rules` evaluates eight rule files.
- **The consistency checks cut over from prose heuristics to the shipped rules (consistency
  layer, Stage 4).** The 0.9.0 Gx-trace authorization inventory, named-unit fact resolution,
  closed-vocabulary and payload-twin checks are deleted with every regular expression that decided
  a fact from prose and every reader of one consumer's private notation. Gy-rules owns those
  decisions over declared facts only, at ERROR tier. Gx-trace narrows back to traceability plus
  the shape errors of the declaration parsers it keeps (`WRITES`, `USES`, `PRODUCES`, `CARRIES`,
  `SUPERSEDES`, `VALUES`, `payload {}`, `derived:`, the marked `AUTHORIZATION.md` table); its
  `checked:` line loses the heuristic counts and gains `authorization rows read`, `VALUES
  declarations parsed`, `payload declarations parsed` and `derived waivers parsed`. Gx warns, for
  one release, on a design that declares no `WRITES{}`, `USES{}` or `PRODUCES{}` anywhere and
  quotes a snake_case or `Entity.member` token in prose on a row with no group: such a design
  relied on the old inference. That warning is removed in the release after next. The bundled
  examples declare a `CARRIES{}` on all 41 actor units and a `USES{}` where their prose named a
  fact, and every Gy-rules result on them is empty.
- **`examples/pii-flow` runs its rules under a real Datalog engine (consistency layer, Stage 2).**
  `rules.dl` is now evaluated by Souffle 2.5 inside a digest-pinned `linux/amd64` image, and
  `adapter.py` only translates: projection and config to `.facts`, `souffle --no-preprocessor`, then
  `leak.csv` and `tainted.csv` to evidence with a complete coverage row, one blocking finding per
  leaking sink, an engine `attestation`, and a committed trace at `generated/souffle-outputs.json`.
  The hand-written fixed point is gone. The adapter fails closed, writing no evidence, when souffle
  is absent, exits non-zero, prints anything, or leaves a declared output unwritten, when a fact
  value holds a tab or line break, and when the projection is not exactly the 1.0 three-layer
  contract. The registry's `verify` command is now a real replay that re-derives the evidence and
  trace byte for byte. The image is built from `examples/pii-flow/souffle-image/Dockerfile` (every
  input pinned by content) by `scripts/pii-flow-image.sh`, which rebuilds it reproducibly with a
  pinned BuildKit and refuses any digest but
  `localhost:5959/machinery/pii-flow-souffle@sha256:51981e17aef416020a1faa778042473a45dda347eab30cbbd9648a264f6f0df7`.
  The image's final stage runs no command, only a `COPY` of files staged by the fetch stage, so a
  native amd64 build and an emulated one on arm64 (which leaves `/root/.cache/rosetta` behind in
  any layer where it executes) produce the same digest.
  The manifest stays on projection 1.0, so `input_hash` is unchanged; `runtime_closure` moves to
  `sha256:bdde2cabffd09544dde5591cf1b26b137bbca8764af4166aa6fd180f2ad04b19` and the checker version
  to `pii-flow-souffle-1`. The pii-flow full-check golden now counts 14 scanned design files (the
  trace) instead of 13; the `Gk-pii-flow` line is unchanged. The design-engines CI job and
  `make preflight` provision the image and run the new `TestPiiFlowSouffleVerdicts`, which proves a
  `fail` verdict naming the leaking sink and every fail-closed path in the pinned image.

### Compatibility and migration

**Existing designs.** A design green under 0.9.0 can now fail Gy-rules, and loses nothing it
did not declare. Nothing is inferred from prose any more; declare instead:

- `WRITES{Entity.attr, ...}` on each unit that writes stored facts, and `WRITES{}` on a unit the
  0.9.0 scan treated as read-only because its prose said it writes nothing (a `System` action
  whose unit declares `WRITES{}` owes no admission).
- `USES{fact, ...}` on each unit whose contract names a fact; every member must resolve to a
  Modelith attribute or enum member, a machine context key, an event payload field, or the row's
  own `derived:` waiver or `VALUES` member. A backticked token outside every group is prose.
- `PRODUCES{Entity.action, ...}` on each matrix row naming a cascade or consumer arm that
  performs an action; each produced action needs an `AUTHORIZATION.md` row.
- `CARRIES{kind:target, ...}` on every actor (unless it declares `WRITES{}`) and on every action
  with a non-empty `WRITES{}`: `column:Entity.attr` for a repository write, `outbox:event` for an
  outbox emission, `sink:element` for an external call or resource, `signal:name`, and
  `action:Machine.unit` for an effect another machine's unit answers.
- A named `VALUES Enum{...}` wherever an unnamed group's unit name differs from its enum's name,
  case included.

A design that carries its own authorization notation (resource action lists, producer marks,
residual verb tables) can keep it for its own tooling; machinery reads none of it, and what
happens depends on its shape. Notation in any shape but a group (prose, its own tables,
lower-case marks) is ignored and never fails. Notation in the group shape, an upper-case
`NAME{...}` in a matrix table cell, fails Gx-trace (and Gy-rules' projection) under 0.10.0 as an
unknown declaration group, so the design either rewrites it out of that shape or, from 0.10.1,
lists each name in the Architecture Contract's `private_groups:`, after which machinery skips
it. Either way, generate
the marked `AUTHORIZATION.md` rows (`Entity.action` subject, backticked capability or
`(no authorization: <reason>)`) from the consumer's own reader, and declare the generated file
and that reader in a `reads:` row so Gr-reads binds the pair. The migration note in the machinery
skill ("Migrating from the 0.9.0 prose inference") walks through each case.

**Proof scope.** A green Gy-rules establishes, over declared facts only, that every `System`
and produced action has one admission naming a declared capability, every declared fact
resolves, every enum-bound `VALUES` group equals its enum and every enum-less shared group agrees
with itself, every `payload {}` twin equals its contract row, every effect names a carrier, and
type supersession is acyclic with owned replacements. It says nothing about facts a design only
mentions in prose.

### Fixed

- **The forged-activation lane test no longer depends on the parent's descriptor table.** On the
  hosted macOS runner the test process inherits a socket on fd 9, which turned the forged
  `MACHINERY_INTERNAL_CTL=9` claim into a present-but-unmarked channel and a different (equally
  correct) refusal than the case asserts. The child now gets fds 3 through 9 pinned to `/dev/null`.

- **Concurrent cold starts of the pinned Java runtime share one provisioner.** The
  provisioning lock was a scope lock, and test binaries keep their scope locks beside their own
  executable, so several packages under one `go test ./...` that cold-started the same user cache
  each believed they held it. They staged into `<cache>/java/.java-stage-<n>/` at once, and each
  one's stage recovery removed or witnessed the others' live stages, which failed as a vanished
  `runtime.archive`, a missing `extracted` directory, or a stage file that "grew beyond its exact
  snapshot" (the Dagger `test` job on a fresh cache). The lock now lives beside the runtime as
  `<cache>/java/.java-provision.lock` (new `filelock.AcquireFileWaitContext`). One caller
  provisions under the unchanged witness discipline; every other caller waits, bounded by its
  context and the filelock acquisition limit, then opens the published runtime through the
  normal receipt, closure-digest, and custody validation. A holder that dies releases the lock
  with its process, and only the next holder removes the stage it left. `OpenJavaContext` carries
  the caller's context into that wait. The runtime bytes and receipt are identical whether one or
  five provisioners raced. The same holds for the two sibling caches that had the same scope-lock
  hole: the TLA+ and Alloy jars (lock in a private `.machinery-formal-jar-lock` directory under the
  user cache, outside the replaceable `machinery` parent, so replacing that parent still cannot
  split it) and the Structurizr CLI (`<cache>/machinery/structurizr/.structurizr-provision.lock`).
  Waiters rehash the installed jar or revalidate the Structurizr receipt and closure digest, and
  their wait honors the caller's context.

## [0.9.0] - 2026-09-22

### Added

- **`Gr-reads` binds compile-time design-file consumers to their implementation follow-ups.** A
  contract v2 Architecture Contract may declare root `reads:` rows with a design-relative
  `artifact`, implementation-relative `reader`, and full reviewed Git commit. Design-only checks
  warn that the coupling exists. With `--impl`, the gate fails closed unless the review commit is
  an ancestor, the artifact and reader lineage were tracked there, and every later commit or current change set that
  changes the artifact also changes the reader. Merge-parent tree diffs and reader renames are
  included. The new gate is artifact-activated, joins the
  default and explicit gate vocabulary, and is proven by a warning golden plus Git-backed failing
  and passing fixtures. Existing designs without `reads:` keep byte-identical gate output.
- **Slice packets carry their fixture-module obligations.** A slice may now declare an optional,
  non-empty `fixtures:` list of clean implementation-relative paths in `design/slices.yaml`.
  `Gw-packet` fails closed on null, malformed, nonportable, or duplicate paths, reverse-indexes shared fixtures, and
  states the complete sorted consumer-slice set in every affected packet so executors know which
  suites a fixture change obligates. The committed packet fixture now binds one shared module to
  two slices, so its golden packet bytes, size counts, and section numbering intentionally change;
  the fixture corrects the corpus to exercise the new gate behavior. Maps without `fixtures:` keep
  their prior packet shape and gate output.
- **Gx-trace now reconciles declared event-payload twins.** A matrix event row may declare one
  closed payload with `payload {field, ...}` or `payload is exactly {field, ...}`. The declaration
  binds to that row's one event and must equal the Architecture Contract event row's payload field
  set. Malformed, duplicate, conflicting, missing, multi-event, and unequal declarations are
  blocking findings, and a mismatch prints both spellings. Existing examples remain green because
  ordinary payload prose is not a declaration; this is an opt-in compatibility extension.
- **Gx-trace now closes autonomous writes against an authorization inventory.** Every Modelith
  action whose actor is `System`, plus each producer named by a matrix cascade or consumer arm,
  must have one exact row in a marked, g2-attested hand-written inventory. The row names one backticked dotted admitting
  capability or uses `(no authorization: <reason>)`; missing, duplicate, orphan, empty, and
  reasonless rows fail. This intentionally made the fulfillment, portfolio-engine, and
  checkout-split child examples red; their previously implicit internal capabilities are now
  recorded, and the golden corpus is regenerated with the new authorization counts.
- **Gx-trace now resolves named-unit facts.** Backticked facts in contract, clause, and
  payload columns follow snake-case and Modelith attribute spelling styles and resolve to model attributes, enum
  members, actions, machine context keys, event names and payload fields, failure-catalog ids,
  same-row VALUES members,
  or a reasoned row-local `derived: fact_name (<reason>)` waiver. This intentionally exposed
  `feed_circuit_open` in the portfolio-engine corpus; its row now records that the operator log
  signal is derived rather than stored.
- **Gx-trace now closes matrix vocabularies.** A row that calls a vocabulary closed, an enum, or a
  reason class must declare exactly one `VALUES{a, b, c}` group. A same-named Modelith enum must
  agree as a set; otherwise the row is the one closed source. `VALUES name{...}` binds a shared
  vocabulary across differently named units. Empty, duplicate, malformed, repeated,
  and conflicting declarations fail. The existing example corpus required no correction.

### Changed

- **Gx-trace accepts H2's machine-written authorization grammar.** Resource
  action lists, producer-narrowed residual marks, and the all-presets-withheld
  residual verb fallback are first-class inventory
  declarations without a marker document. A marked admission must name an exact
  declared C4 element, matrix producer, or residual-table preset; a fabricated
  capability suffix is a finding. The gate proves row existence and declared-subject
  resolution, with capability-list resolution tracked in NEXT. Fact resolution now reads declared content and relation
  sources, groups prose-only nested map keys into one typing gap, and excludes
  H2's quoted rejections. Closed-vocabulary detection distinguishes finite
  declarations from pass identities, human reasons, referenced owner sets, and
  typed Modelith enums. A declared verification that does not change its
  resource owes no resource-write admission.
- **The packet reference states the lane-scope protocol assumed by per-slice execution.** A
  commit named `integrator-request(<slice>)` may carry integrator-owned paths for reconciliation,
  and a design lane's complete diff is treated as that same kind of request. Projects can
  implement scope gates against this convention.

### Fixed

- **Host-denied tools carry an exact terminal signal into governance.** OpenCode translates a
  refused permission, a tool-part error, a tool error in the after hook, or an idle turn with a file-tool `tool.execute.before` but no matching
  `tool.execute.after` into `PostToolUseFailure` with the call id and explicit reason. Shell calls
  without a terminal event remain armed because they may leave a delayed writer. Stop keeps
  every remaining pending token armed, even when the governed tree is unchanged or Stop is
  re-fired. New pending tokens no longer fingerprint the tree; existing hashed ledger lines remain
  readable. The gate obligation runs only after every exact tool completion is recorded.

### Compatibility and migration

**Generated output.** The oracle, TLA+, Alloy, checker-projection, and packet generators now
emit the `v0.9.0` machinery stamp. This restamps the committed example oracles and formal
artifacts, pii-flow checker projection, and golden corpus, including its packet fixture. Run
`machinery oracle <design>/machines`, `machinery verify-formal <design> --gen-only` (which also
regenerates opted-in Alloy layers), and `machinery project <design>` for those example families;
run `make golden-update` to recapture the corpus, including `machinery packet` output. The
release-preparation regeneration changes only version stamps. The packet fixture's shared
fixture obligations changed its content earlier in this release cycle. Consumers should
regenerate and commit their generated design artifacts on upgrade.

**Proof scope.** A green Gx-trace now establishes four additional design-side conditions: an
explicit closed event payload equals its Architecture Contract fields; every autonomous write
has one declared authorization admission; named-unit facts resolve to a declared source or a
reasoned row-local derivation; and a closed vocabulary has one exact `VALUES` set. H2's
machine-written inventory forms are accepted with declared-subject resolution. This is stronger
evidence than a green 0.8.0 Gx-trace. `Gr-reads` adds review-history evidence only for contracts
that declare `reads:`; design-only checks warn and `--impl` checks artifact-reader follow-ups in
Git history. `Gw-packet` now records shared fixture-module obligations when `fixtures:` is present.
These gates do not execute an implementation or prove the correctness of a reader edit.

**Existing designs.** A design green under 0.8.0 can now report Gx findings for an event row's
explicit `payload {field, ...}` or `payload is exactly {field, ...}` that disagrees with the
Architecture Contract; an unlisted `System` action or matrix cascade/consumer producer;
an unresolved backticked fact; or a row calling a vocabulary closed, an enum, or a reason class
without exactly one matching `VALUES{a, b, c}` group. Answer these with the matching contract
payload fields, a g2-attested `AUTHORIZATION.md` row with an `Entity.action` subject and
either a backticked declared capability or `(no authorization: reason)` admission, a declared
model/matrix source or
`derived: fact_name (reason)` in that fact's row, and an exact `VALUES{...}` group
(`VALUES name{...}` for a shared vocabulary). H2 resource action lists,
producer-narrowed residual marks, and all-presets-withheld residual verbs are also valid
authorization declarations. For a compile-time design reader, add a root contract-v2
`reads:` row with design-relative `artifact`, implementation-relative `reader`, and full
40-character reviewed Git commit, then check with `machinery check <design> --impl <impl>`.
The optional `fixtures:` list in `design/slices.yaml` uses unique, clean implementation-relative
paths; regenerate packets to carry every consumer slice. Designs without those optional
declarations retain their previous shapes. Upgrade an install with
`machinery update --version v0.9.0`.

## [0.8.0] - 2026-09-10

### Added

- **`machinery packet` and the `Gw-packet` gate: bounded per-slice executor packets projected
  from an authored slice map.** A manifest-mode design can exceed the context window of the
  executor that has to build one of its slices (H2's M1: 300K to 470K tokens per slice against a
  200K budget). The owner ruled that the sources are not split by hand and the executor is not
  swapped for a larger one. The design gains one authored, machine-readable artifact,
  `design/slices.yaml`, binding each slice of a milestone to exactly one BUILD shard, a byte
  budget, and the element ids it cites: oracle rows by stable id, whole oracle sets, matrices,
  shard sections by heading id, milestone blocks, files, Architecture Contract boundaries,
  externals, dependency rules and table rows, and invariants. `machinery packet <design>
  --milestone M1 --out <dir>` projects one packet per slice carrying only what the slice cites,
  every excerpt a verbatim copy under its citation and source `path:line`, plus the milestone's
  computed obligation ledger, the acceptance entry shape, and the lines drawn per source. The
  projection reads ids and never prose: re-wording the slice narrative in BUILD.md changes no
  packet. It is byte-reproducible for the same design bytes, and the golden corpus pins it over a
  committed fixture design. The budget is a byte proxy with one fixed documented divisor (3 bytes
  per token; declare a 200K-token executor as `budget: 600000`); no tokenizer is vendored or
  called. `Gw-packet` auto-activates on `slices.yaml`, runs after `Gb-plan`, and fails closed:
  every citation resolves to exactly one excerpt, every packet fits its budget, and every
  obligation the milestone owes (the DoD-cited oracle ids `Ga-accept` binds evidence to,
  `ORACLESET{...}` expanded) is claimed by exactly one slice or carries a recorded waiver;
  unclaimed, double-claimed and claimed-and-waived obligations, stale waivers, dangling
  citations and missing shards are ERRORs. The command runs the gate first and writes nothing
  when it fails. The stop hook selects `gw` whenever the map exists. Reference:
  [docs/packet-projection.md](docs/packet-projection.md).

### Fixed

- **A publish in flight is no longer read as store corruption.** `publishImmutableFile` stages its
  bytes in a private temp beside the final path, because a hard link into place requires the same
  filesystem, so the temp is briefly visible to any reader enumerating that directory. Four strict
  enumerations (`blobs/`, `controls/`, `runs/` and `ledger/heads/`, plus `objects/`) rejected it as
  an unknown entry, so a healthy store reported `INVALID_SCHEMA: ledger/heads/ holds unknown entry
  ".publish-<nonce>"` purely because another writer was mid-publish. Observed as an intermittent
  `TestRegisterConflictingWriters` failure under a loaded containerized full-suite run. The temp's
  name is now part of the schema rather than an accident: one documented predicate matches exactly
  `.publish-` plus sixteen hex digits, every strict enumeration skips that and keeps rejecting
  everything else, including any other dot-entry. The regression test interleaves an enumeration
  with a publish deterministically through a hook between staging and linking, rather than relying
  on load.

- **A directory ABA is caught on hosts whose inode clock is too coarse to see it.** The
  fail-closed directory inventories in `internal/gates`, `internal/dirscan` and the `machinery
  oracle` inventory proved that an enumeration was a snapshot by comparing native change stamps
  across two independent passes. On Unix that stamp is the inode ctime, and on Linux kernels
  without multigrain timestamps (mainline 6.13) every inode timestamp comes from the kernel's
  coarse clock, one timer tick wide. A create-then-delete, or a rename-and-back, that completes
  inside one tick leaves ctime, mtime, size and link count all identical, and
  `STATX_CHANGE_COOKIE` is stripped from userspace requests on those kernels, so there was no
  finer stamp to fall back to: the ABA defense accepted the enumeration, which is the fail-open
  direction. Measured on Ubuntu 24.04, kernel `6.8.0-138-generic` with `CONFIG_HZ=1000`, inside a
  container on both overlayfs and an ext4 bind mount, as root and as an unprivileged user; the
  hosted CI runner did not show it because it runs `6.17.0-1022-azure`. Every such inventory now
  also arms a kernel mutation-event watch (inotify on Linux, kqueue on the BSDs) for the whole
  witness window, through the already-open directory descriptor rather than a path, so it stays
  inside an `os.Root` confinement. Events do not depend on any clock and cannot be un-queued by a
  later restore, so the ABA is reported whatever the timestamp resolution. Windows is unchanged:
  its witness is the NTFS ChangeTime read from the retained handle, which this blindness does not
  apply to.

- **A file content ABA is caught on those same hosts.** The directory inventories were not the only
  surface the coarse clock blinded. `internal/designlock` proves a tracked external input and every
  entry of a rooted design inventory did not change under it by comparing mode, size, mtime and the
  same inode change stamp across the read window. A write, a restore of the original bytes, and an
  `os.Chtimes` back to the original mtime, all completing inside one tick, left every one of those
  identical, so a content ABA was accepted. Both fingerprint paths now arm the same kernel
  mutation-event watch through the already-open descriptor for the read window, and refuse the
  fingerprint outright when the channel cannot be armed rather than falling back to a stamp that may
  be blind. One channel serves a whole rooted traversal. The regression tests stub the change stamp
  to a constant, which is what a coarse-clock host effectively returns, so they reproduce the
  blindness and prove the fix on every host regardless of the real clock resolution.

### Compatibility and migration

**Regeneration stamp.** The `machinery-version:` families and the pii-flow checker projection move
from `v0.7.2` to `v0.8.0` and nothing else in them changes. Regenerate with the commands the gate
suite prints and commit the stamp-only diff on its own.

**A new gate and a new command, both opt-in by artifact.** `Gw-packet` joins the suite after
`Gb-plan` and activates only when `design/slices.yaml` exists, so no existing design changes
behavior on upgrade; `gw` joins the `--gate` vocabulary and the stop hook's selection. `machinery
packet` writes packets to the directory you name and never into the design; packets are
regenerated per executor run and are not committed. The byte budget's divisor (3 bytes per
token) is a documented constant of this release, not a per-design setting.

**The directory and content witnesses can now refuse where they previously accepted.** This is a
proof-scope change in the stronger direction, and it has an operational edge. On Linux and the BSDs
the witness arms a kernel mutation-event channel (an inotify instance or a kqueue) for the duration
of each enumeration. A host that cannot supply one, most plausibly a Linux box whose
`fs.inotify.max_user_instances` is exhausted, now fails the enumeration closed rather than falling
back to a change stamp whose resolution may be blind. The remedy is to raise that limit; the gate
will not silently degrade. Windows is unchanged: its witness is the NTFS ChangeTime read from the
retained handle, which this blindness does not apply to, so no event channel is used or required
there. A platform machinery does not build for refuses rather than degrading.

**What a green gate now establishes.** On a host whose inode clock is one timer tick wide, which is
every Linux kernel before mainline 6.13, a create-then-delete or a rename-and-back inside one tick
was previously accepted by the directory inventories, and a write-restore-and-reset-mtime inside one
tick was previously accepted by the file content fingerprints. Both are now caught. A design that
passed `machinery check` on such a host under 0.7.2 was proved less than the output claimed; re-run
the gates on 0.8.0 to get the evidence the run reports.

**No schema, attestation or installation-receipt changes.** A 0.7.2 install migrates with
`machinery update --version v0.8.0` and nothing else. The one new CLI surface is the `packet`
command above; nothing existing changed shape.

## [0.7.2] - 2026-09-09

### Fixed

- **`machinery update` proves the Claude Code marketplace refresh by its success line.** On
  0.7.1, with the inventory reader fixed, the refresh got one step further and failed again:
  Claude Code 2.1.266 prints `Refreshing marketplace cache (timeout: 120s)…` between the update
  banner and `✔ Successfully updated marketplace: machinery`, and the canonical-output check pinned
  the exact lines, so a successful refresh was returned as `non-canonical Claude marketplace
  success output` and the plugin never refreshed. This is the third closed-format defect in the
  host-plugin reader class (the 0.6.9 plugin cache, the 0.7.0 inventory, now the marketplace
  output). The check now requires the update banner first and the exact success line last, and
  rejects any intermediate line carrying a failure marker (`✘`, `error`, `failed`, `warning`);
  the progress lines between them belong to Claude Code and are no longer pinned. The 2.1.266
  output is pinned as a fixture beside the older shape, with five rejected shapes. The plugin
  update output (`Checking for updates ... ✔ Plugin "machinery" updated from 0.7.0 to 0.7.1 for
  scope user.`) still matches its contract unchanged.

- **`make ci-linux` carries a UTF-8 locale and reports both hosted jobs.** The first end-to-end
  run on a linux/amd64 VM (2026-09-09) showed two container defects: the image had no locale, so
  `elixir --version` printed the BEAM's latin1 warning on stderr and the lane's runtime probe
  rejected the Elixir identity (`TestAssuranceCatalogNativeSkipCannotBecomeSuccess` and the
  required lane both fail on that diagnostic); and the sweep and the lane ran in one `set -e`
  shell, so a red sweep hid the lane's verdict. The image now sets `LANG` and `LC_ALL` to
  `C.UTF-8`, and the script runs the sweep and the lane as two container invocations and fails on
  either, the way hosted CI reports two jobs. Known residual: inside the
  container the directory ABA witness in `scripts/tree-inventory` is blind on a coarse-timestamp
  filesystem (two of its tests fail there and pass on the hosted runner), so the containerized
  sweep is not yet a faithful mirror of the hosted test job for that one package.

### Compatibility and migration

**Regeneration stamp.** As in 0.7.1: the `machinery-version:` families and the pii-flow checker
projection move from `v0.7.1` to `v0.7.2` and nothing else in them changes. Regenerate with the
commands the gate suite prints and commit the stamp-only diff on its own.

**Claude Code marketplace refresh.** Unchanged: a refresh whose output ends in anything other than
the exact success line, or carries a failure marker on any line, is still a returned error with a
recorded obligation. Accepted now: any progress lines Claude Code prints between the banner and the
success line. No migration; a 0.7.1 install whose update reported the non-canonical output migrates
by running `machinery update --version v0.7.2` twice (the first run's parent is the 0.7.1 binary,
which still rejects the output; the second run's parent is 0.7.2).

## [0.7.1] - 2026-09-09

### Fixed

- **`machinery update` accepts the plugin inventory the current Claude Code writes.** The Claude
  Code plugin refresh read each `claude plugin list --json` entry as a closed record and rejected
  the fields Claude Code now adds (`enabled`, `installPath`, `installedAt`, `lastUpdated`,
  `mcpServers`, `version`), so on 0.7.0 every Claude Code user saw `Claude Code plugin inventory
  was not understood: Claude plugin entry 0 has unknown fields [...]`, the update returned an
  error, and the plugin never refreshed. machinery consumes `id` and `scope` only;
  the entry is now an open record for the fields it does not read, and it still fails closed
  when a consumed field is absent or malformed. The exact inventory Claude Code wrote on
  2026-09-08 is pinned as a fixture. The Codex inventory reader is unchanged: it is still a
  closed record, because the fields it consumes are the whole record.
- **The first `machinery update` over a 0.6.11 install no longer leaves the receipt describing
  the previous release.** After `machinery update --version v0.7.0` on a 0.6.11 install, doctor
  reported the installed skill and the build-writer role as invalid (`artifact digest is
  sha256:e2262eda..., want receipt-bound sha256:cc1d8dd4...`) although every file on disk was
  byte-identical to the release, and a second update converged. Cause: 0.7.0 moved
  receipt finalization from the placement child to the update parent, but on a
  cross-version update the parent is the old binary and the child is the new one. A 0.6.11
  parent never finalizes, and the 0.7.0 child left the receipt to a parent that would not write
  it, so the committed receipt still carried the 0.6.11 digests. The parent now announces to its
  placement children that it publishes the receipt; a child that hears no announcement (a
  parent older than 0.7.1) records its own placement, retaining the recorded digest of any
  artifact a sibling child still has to place. The announcement rides the environment rather
  than the lock capability payload because every child before 0.7.1 compares that payload byte
  for byte, and a downgrade through `machinery update` runs exactly such a child. A test runs a
  real placement child under a prepared parent transaction both ways: without the announcement
  the receipt is rewritten and doctor's receipt-bound check passes for the placed tree; with it
  the receipt is untouched. Reproduced and verified against the published v0.6.11 and v0.7.0
  assets in a fresh `HOME`.

- **Ga-accept binds the acceptance commit in both milestone states.** The gate resolved the
  commit under review only when the build plan carried a milestone marked `Status: closed`, and
  bound an evidence commit only inside that closed-milestone loop. Acceptance evidence for a
  milestone still open was parsed, schema-checked and held to its DoD ids, and then its `commit:`
  field was never asked to name anything: `design/acceptance/M1.yaml` carrying a fabricated 40-hex
  sha, on an open M1, reported `Ga-accept ok` with no finding, and the `checked:` line named no
  binding mode because no commit had been resolved to name one. The resolution and
  ancestry path existed and worked; on an open milestone nothing reached it. The evidence commit
  is now bound for every acceptance file whose milestone the plan declares, whatever that
  milestone's status, with the diagnostics the closed path already used: a sha the repository
  holding the design does not hold, or one that is not an ancestor of the history anchor, is an
  ERROR naming the sha and the anchor. Closed milestones bind where they always did, so finding
  order and the `checked:` counts are unchanged. The degraded mode is kept and is now stated as a
  mode rather than as silence: outside a repository, or with no usable git, the `checked:` line
  reads `no commit under review could be derived; evidence commits UNBOUND to git history` beside
  the existing non-blocking note, so an unbound run is never inferred from a missing note.

- **The hook repairs its own state instead of bricking every repository.** A store filled past the
  fail-closed limit used to deny every governed tool call until a human pruned it by hand. The first
  inventory that hits the limit now compacts once under a repair ceiling and retries. This covers
  ledgers written by 0.7.1 and later: a ledger written by 0.7.0 or older carries no project root, so
  nothing can tell its obligation from a live one and compaction retains it. A store already over
  the limit at upgrade time therefore needs a one-time manual removal of its `<digest>.state` files,
  never of the store directory, whose initialization marker distinguishes a first run from a lost
  store.

### Added

- **`machinery doctor --repair` compacts the governance hook state store.** The per-user store the
  hooks keep their gate obligations in now has a retention policy applied on every arming write: at
  most 8 route snapshots per project root, and a store-wide ceiling of 512 entries, an eighth of the
  4096-entry limit past which every bounded inventory fails and every governed shell and write tool
  on the machine is blocked. Compaction reclaims only whole generations whose recorded project root
  no longer exists on disk, so it can shrink the store but never discharge a live obligation.
  `machinery doctor` reports the store's path, entry count and both bounds; `machinery doctor
  --repair` compacts it. Both work when the store is already at or above the limit, which is the
  only moment either matters. See the plugin guide's durable-state section for the policy and its
  residuals.

### Changed

- **`machinery doctor` exits nonzero on an unusable hook state store.** The new store report fails
  the command when the store is at or above its 4096-entry limit, or when the store path cannot be
  resolved or inspected (a container with no resolvable HOME, for instance). A store above the
  retention ceiling but below the limit reports and passes, so the common state does not newly fail.
  Any CI job or Makefile target that treats `machinery doctor` as a gate has a new failure source.

- **The gate is tiered: cheap on every push, heavy where it is enforced.** Releasing a one-line
  patch cost a 50-minute local gate per push, while five of the failures that broke the 0.7.0
  release night were hosted-CI-only and invisible to it. The push gate is now
  `scripts/preflight-fast.sh`, the single owner of the cheap tier and the only thing
  `.githooks/pre-push` runs: the aggregate whitespace diff, gofmt, `go vet`, golangci-lint for the
  host and for `GOOS=linux`, actionlint, ShellCheck, `go mod tidy`, the docs gate, the Modelith
  render check, the build, the example gate suites, and the golden corpus plus the gate-experiment
  suite. Measured at 1m25s on the documented host class, against a 5-minute budget. There is still
  no bypass variable, in either tier. `scripts/preflight.sh` keeps its role as the full local
  mirror and now runs the cheap tier first, then the heavy tier unchanged (race sweep, required
  native lane, implementation modules, formal verification, C4, external checkers). Hosted CI is
  unchanged and remains authoritative for the heavy tier; the required job set and
  `docs/release-policy.json` are untouched, so release publication is gated exactly as before.
- **`make ci-linux` produces the Linux evidence before the push, not after it.** The hosted
  `test` and `integration-required` jobs now run locally inside a pinned 2-CPU `linux/amd64`
  container that carries the exact runtime identities `.github/actions/assurance-runtimes`
  installs (Go 1.27.1, Node 26.8.1 with TypeScript 7.0.2, CPython 3.14.7, Elixir 1.20.4 on OTP
  29.0.6), with the cpuset pinned so a runtime that sizes its scheduler from the visible core
  count sees two cores the way the hosted runner does. `CONTRIBUTING.md` names it as the step
  before any push that touches custody, the TDD adapters, or the lane, and states what the
  container does and does not reproduce. The container runs the sweep as root; running it as an
  unprivileged user, and moving the fsync-bound install suite and the load-sensitive custody
  suites out of the parallel sweep into the lane with fixed budgets, stay open.

### Compatibility and migration

Every entry below states what still works unchanged, what is rejected now, and the exact command
that migrates. Nothing here changes a schema or a generator; the regeneration stamp is the only
artifact change.

**Regeneration stamp.** The families that carry `machinery-version:` (`machines/*.tla`,
`machines/*.cfg`, `machines/*.oracle.md`, `formal/*.als`) and the pii-flow checker projection move
from `v0.7.0` to `v0.7.1` and nothing else in them changes. Regenerate with the commands the gate
suite prints for the design (`machinery oracle`, `machinery tla`, `machinery alloy`, and
`machinery refine`/`machinery compose`/`machinery pack generate` where the design carries those
families) and commit the stamp-only diff on its own.

**Ga-accept on open milestones.** Unchanged: evidence for a closed milestone binds exactly as
before, and a design with no acceptance file is untouched. Rejected now: an acceptance file on an
open milestone whose `commit:` names a sha the repository holding the design does not hold, or one
that is not an ancestor of the history anchor. Finding:

```
design/acceptance/M1.yaml: commit deadbeef... names no commit in the repository holding the design (history anchor is ...)
```

Migrate: record the sha the review actually read. A second consequence is visible on the command
line: the `--commit` anchor has always been specified as a hex object id (`docs/acceptance-gate.md`),
but on 0.7.0 a design with only open milestones never reached the resolver, so `--commit HEAD`
passed silently there. On 0.7.1 it is rejected with `supplied history anchor must be a lowercase
hexadecimal VCS object id or unambiguous prefix (7 to 64 characters)`. Pass `--commit
"$(git rev-parse HEAD)"`, `$GITHUB_SHA`, or `MACHINERY_COMMIT=<sha>`; a repository-backed run with no
anchor still derives HEAD itself.

**Installation receipt after a cross-version update.** Unchanged: the receipt schema (2), every
`machinery install` and `machinery uninstall` path, and updates whose parent is 0.7.1 or newer. Fixed
in the field: a 0.6.11 install updated once to 0.7.1 converges in that one run. A 0.7.0 install whose
`machinery doctor` already reports `artifact digest is ..., want receipt-bound ...` migrates by
running `machinery update --version v0.7.1` once: the 0.7.0 parent finalizes the receipt itself. The
announcement between parent and child (`MACHINERY_INTERNAL_INSTALL_RECEIPT_OWNER`) is internal and
not a user surface.

**Claude Code plugin inventory.** Unchanged: the Codex inventory reader, and a Claude inventory that
is missing `id` or `scope` still fails closed with `Claude plugin entry N is missing required field`.
Accepted now: any additional field Claude Code writes on an entry. No migration.

**Contributor gate.** The pre-push hook no longer runs the heavy tier, so a push no longer proves the
race sweep, the required lane, formal, C4, or the external checkers locally; hosted CI is
authoritative for those, and `make preflight` or `make ci-linux` runs them locally on purpose. Run
`make hooks` once after pulling so the hook points at the fast tier.

- **The hook state ledger records its project root, which an older binary rejects.** This is what
  lets retention prove an obligation is dead rather than guessing. A binary older than 0.7.1 reads a
  ledger written by 0.7.1 as noncanonical and fails closed on it, so a downgrade, or a second older
  binary sharing the same user home, blocks governance for the affected projects until their
  `<digest>.state` files are removed. Remove the files, never the store directory. Reading in the
  other direction is unaffected: 0.7.1 reads a pre-upgrade ledger normally and only declines to
  reclaim it.

## [0.7.0] - 2026-09-08

### Fixed

- **The partial-witness journal recovery proof no longer cuts a record it did not write.** The
  proof appends the first N bytes of a witness record to a formal transaction journal, for every
  N, and requires recovery to derive the rest. It took the number of cuts from a record seeded in
  one directory but appended the bytes of a record seeded fresh in another. A native witness is
  `unix:<device>:<inode>:<creation sec>:<creation nsec>` in hex, so the encoded record length moves
  with the hex width of the inode and of the creation-time nanoseconds: over 200 seeds on one host
  the record came out 83, 84, or 85 bytes. When the record under test was the shorter one, the last
  cuts sliced past its end into the spare capacity of the encoding buffer, appending a complete
  record plus stray bytes, and the reader rejected those bytes as malformed trailing data, which is
  what they were. It surfaced as `new/byte-082` on a hosted macOS runner and reproduced here at
  `new/byte-084` in 1 of 3 runs. Each cut is now bounded by the record it appends, and two pinned
  cases hold the boundary steady on any host: a sweep with the narrowest and the widest
  creation-time fields a witness can carry, and a check that a byte which cannot begin the next
  expected record stays rejected. The journal reader is unchanged: every true prefix of a valid
  record was already recovered, and the ABA and tamper rejections are untouched.
- **A delivered close report is no longer reported as a lost control channel.** The broker answers
  the root close and then exits: it writes the report, leaves the root channel loop, retires,
  writes its ledger and closes the control connection, all within a few milliseconds. On the
  caller's side one demux goroutine reads that reply into a buffered channel and then, a moment
  later, reads the end of the channel and closes the end-of-channel signal. A caller that had not
  yet reached its own select when both became ready got whichever of the two the runtime picked,
  so a `cleaned` report already sitting in the buffer was announced as `STALE_CAPABILITY: root:
  broker channel closed` with a synthesised `cleanup-failed` report carrying no job diagnostics.
  It only ever hit the root close, since no other close ends the broker, and only on a host slow
  enough to deschedule the caller between sending the close and waiting for its answer: three
  adapter suites failed that way in the hosted non-race macOS job on a loaded three-core runner.
  The caller now drains an already-delivered reply before it reports the end of the channel, which
  is a settled read rather than a second race: the single demux goroutine cannot signal the end of
  the channel before it has delivered every reply it read. No budget or cap changed.
- **A killed process group is no longer read as a survivor while its members await reaping.** A
  job's retirement signalled the group, reaped the guardian, and then probed the group once. The
  members of a job are children of that guardian, so at the instant it is reaped they are orphans
  the platform's init has yet to reap, and on Linux a killed-but-unreaped member still answers the
  group probe (macOS skips those members, which is why the condition never showed there). The
  single probe read them as a survivor and marked a job that terminated exactly as asked
  unterminated: a cancelled required-lane run published `cleanup-failed` with `Terminated:false
  Reaped:true` and no diagnostic naming any survivor, in 3 of 40 runs on a two-core Linux host.
  The retirement now polls the group until it drains, inside the reap window that already bounds
  that stage, so nothing waits longer than before and a group that truly refuses to die is still
  reported unterminated. No budget or cap changed.
- **The custody provisioning test no longer races the pinned Java identity probe it observes.**
  The observation read three process-table snapshots: one to decide a nested pinned JVM was live,
  a second to name its pid, a third to walk its ancestry. Provisioning runs a pinned
  `java -XshowSettings:properties -version` identity probe under the private cache, which
  satisfies the first snapshot and exits within a fraction of a second, so the second found
  nothing and the walk reached nothing: `provisioning JVM process 0 does not belong to the lane
  process tree`, in 6 of 40 runs on a two-core Linux host and in the hosted required lane. All
  three questions are now answered from one snapshot, and the ancestry is walked in the same
  snapshot the process was observed live in. Test-only: the observation is unchanged in what it
  demands, and no budget or cap changed.
- **A failed integration lane says what its native runner reported, and CI keeps the evidence.**
  The lane accounted a failed suite from the runner's event stream and printed only that
  accounting (`native test ... fail failed/skip`, `required execution incomplete: selected 1
  started 1 passed 0`), while the event stream itself and the report went to a private temporary
  directory no workflow collects. A failure only a hosted runner reproduces was undiagnosable
  from the log it left behind. A failed suite execution now replays the bounded tail of that
  runner's own stdout and stderr next to the accounting: the last 200 lines, at most 64 KiB of
  them, each block delimited and naming the suite. `--report-dir`, defaulted from
  `MACHINERY_INTEGRATION_REPORT_DIR` because the workflow and preflight invocations of the lane
  are pinned to an exact argument string, names the directory the report and its retained event
  streams are written to; an unset variable keeps the private temporary directory. `ci.yml` and
  `formal.yml` name that directory and upload it as an artifact when the job fails, with a
  seven-day retention. No budget or cap changed.
- **A child scope's control channel is no longer collected while its descriptor is in flight.**
  The broker created the socket pair for a new child scope, sent one end to the caller, and closed
  its own copy the moment the send returned. For as long as the descriptor was in flight its only
  reference was the copy inside that message, which is what the platform's in-flight descriptor
  collector treats as unreachable: it flushes the receive side of the socket the descriptor names.
  The caller received a live, connected channel it could still send on and every read of which
  returned end of file, so its next request failed `STALE_CAPABILITY: scope-...: broker channel
  closed` while the broker was alive, healthy, and still serving that very channel. On a loaded
  host this took out roughly one run in six of a 200-retirement custody run, and in a
  `verify-formal` pass it ends the run. The sender now keeps its own reference to a handed-over
  descriptor for as long as it is in flight and releases it once the owner proves it holds it,
  with the first record it sends; the capability channel a joined child is launched with follows
  the same rule, and a control record's descriptor rights are held alive across the send that
  transfers them. No budget or cap changed.
- **A custody suite's owner-loss handoff says why it never arrived.** The owner helper polled for
  its grandchild's pid for a fixed 10 s and then wrote whatever it had read, so a chain that had
  not come up inside that window wrote an empty file, which the reader ignores; the reader then
  waited out its own separate 20 s and reported a file that never appeared. Neither number was
  derived from the 60 s wall the helper declares, and neither the helper's own launch failure nor
  its log reached the message. The helper now polls under the wall it declares, keeping back a
  fixed slack to report with, and always writes a record naming what happened; the reader derives
  its wait from that same wall and prints the helper's log when the record is missing or is not a
  pid. Test-only: no product budget or shipped cap changed.
- **A custody challenge test no longer races the broker it built from a hook it replaces.** The
  challenge cases run the broker in process and replace a package-level hook to build a second
  broker from the replacement. The broker reads those hooks once, when it is built, and everything
  the test observes of the first broker afterwards travels through a kernel socket, which gives
  the race detector no ordering: `go test -race -count=5 ./internal/processscope` reported
  `race detected during execution of test` on an idle host within a second, while a single-count
  sweep never did. The harness now joins the broker goroutine it started as part of its cleanup,
  and each case retires and joins the broker built from the previous hook before replacing it.
  Test-only: no product behaviour changed.
- **A scope close joins a retirement already claimed instead of collecting nothing.** The join
  0.7.0 gave `retireJob` was unreachable for the case it was written for: both collectors filtered
  the job table on jobs no retirement had claimed, so a job claimed before the snapshot was never
  passed to the join at all. A `Run` cancelled by its caller is retired on its own goroutine, and
  the close that follows it arrived while that retirement sat between marking the job terminated
  and reaping its guardian, so the close published that half-written state:
  `cleanup-failed` with `Terminated:true Reaped:false` and no diagnostic, on a job that went on to
  reap cleanly. `go test -race -count=5 ./internal/processscope` reported it on an idle host. Both
  collectors now pass every job in scope through the join, which returns at once for a retirement
  that has already settled, and the claim itself is taken and read under the broker's own lock. No
  budget or cap changed.
- **A scope close reports the settled state of a job already being retired.** A job cancelled by
  its caller is retired on its own goroutine, and the guardian may report the target's death as an
  ordinary exit rather than a terminal drain, so `Run` returns while that retirement is still
  between marking the job terminated and reaping its guardian. The close that followed collected
  only jobs no retirement had claimed, skipped that one, and published whatever half-written state
  it found: `cleanup-failed` with `Terminated:true Reaped:false` and no diagnostic, on a job that
  went on to terminate and reap cleanly. A close now joins a retirement already in flight instead
  of skipping it, bounded by that retirement's own budget, so a job that truly refuses to die is
  still reported unreaped with its `TIMEOUT` diagnostic. No shipped cap changed: the per-job
  cleanup budget, the 30,000 ms cleanup cap and the 2 s hard reap window are unchanged, and
  nothing runs longer than before, only the report waits for a settled answer.
- **A guarded job that finishes before its caller can wait for it is no longer reported as a
  timeout.** The control channel registered a job's result channel when the broker's started frame
  arrived and retired that registration on delivery of the result, while `Run` claimed the channel
  only after the started reply reached it. One reader handles both frames in order, so a job short
  enough to finish first, an identity probe or any small guarded command, had its result delivered
  and its registration retired before its own caller claimed it: `Run` then registered a second,
  empty channel, waited out its whole deadline on it, and reported `TIMEOUT: job terminated by
  scope cancellation` on a job that had already exited 0. The registration now belongs to the
  caller for the whole call, so a result that arrives first is waiting in the channel the caller
  claims, and a duplicate result for an already-reported job is dropped instead of stalling the
  reader every other frame on that connection depends on. No budget changed.
- **A custody root grants the wall it declares, not the window its owner was opened with.**
  `processscope.Open` bounds a scope's cumulative wall by the earliest of the declared `wall_ms`
  and the open context's own deadline, so an open context sized as a bootstrap window quietly
  becomes the real root wall. Every custody root the suites open now derives its open context from
  the wall it declares, the correction 0.7.0 already applied to the contributor lane and to
  `verify-formal`'s candidate scope. A 7-spec TLC portfolio measured at 85 s under the race
  detector on 2 vCPU used to run out of a 60 s bootstrap window and fail every remaining spec with
  `BUDGET_EXHAUSTED: root: inherited wall deadline has passed` against a declared 2,400,000 ms
  wall. No cap changed: the declared budgets are what they always were.
- **A finished Elixir suite whose ExUnit teardown crashes is no longer reported as a build
  error.** The embedded `elixir-exunit/v1` formatter used to stop itself while handling ExUnit's
  final `suite_finished` event, racing the `ExUnit.EventManager.stop/1` call the runner makes
  immediately afterwards: when the formatter supervisor had not yet processed the exit signal, the
  teardown exited `{:noproc, {GenServer, :stop, [pid, :normal, :infinity]}}` and killed the runner
  after the suite had already executed, intermittently on a loaded host. The formatter now closes
  its event stream in place and stays alive, so ExUnit stops it exactly once. Independently, a mix
  abort that happens after a complete, reconciled native run is classified as
  `UNEXPECTED_FAILURE` naming the reconciled outcome, with the witnessed assertion outcomes
  preserved on the execution record, instead of a `BUILD_ERROR` claiming a suite that compiled and
  ran failed to build.
- **A matrix's prose is narrative again, not a malformed `CLAUSES` declaration.** A declaration is
  a `CLAUSES{...}` group on a row of the named-unit contract table, which is the only shape 0.6.11
  read. An invariant-coverage bullet quoting a guard's vocabulary, an enumeration wrapped across
  lines, a contract cell that mentions "the CLAUSES declaration here" or describes a RETIRED site,
  and a row for a non-guard unit are all accepted as they were, instead of being reported as
  malformed declarations whose "guard name" is a sentence fragment. Ambiguity is still rejected: a
  row carrying two `CLAUSES{...}` groups, or a `RETIRED{...}` group the declaration does not carry,
  names no single vocabulary and fails.
- **A declared guard no oracle governs owes nothing again.** 0.6.11 resolved a declaration's
  falsifying-clause obligations across every committed oracle, so a machine could declare the
  clauses of a guard that sits on its creation edge rather than on a guarded transition, and owe
  no tests. That case is silent once more. The ownership rule this release added stands: a
  declaration whose rows only a SIBLING machine's oracle could supply is still an error, and the
  diagnostic now names the sibling.
- **Consumer `READS` members keep their ubiquitous-language grammar.** A member is any trimmed,
  non-empty text, as it was in 0.6.11: `READS{Order.id, occurrence time, PAIR KEY}` declares three
  fields. Empty members and duplicates are still rejected, and an empty member is now reported as
  an empty member rather than as an "empty or malformed READS field" naming text that is neither.
- **`READS` is an English verb again outside a declaration.** A consumer declaration is a
  `READS{...}` group, as it was in 0.6.11. A residual or contract cell narrating that a machine
  "only READS those rows" off an event another layer produces is prose: it is no longer collected
  as a declaration and then reported as an incomplete one. A row that does carry a group is held
  to exactly one complete declaration, unchanged: a `READS{...}` wrapped across two lines is an
  incomplete declaration and still says so, because the group is what a row declares. Join the
  lines.
- **Installer reruns converge on the recorded installation.** The one-line installer over an
  existing install now refreshes the complete recorded home/native-target/plugin plan (each
  group's copy/symlink mode preserved) instead of the default homes only, identical to
  `machinery update` without selectors. Safely edited or missing owned artifacts are repaired to
  exact current-release content; corrupt, unsupported, or unsafe receipts, unsafe artifact or
  parent substitutions, and uncertain plugin ownership fail closed with diagnostics (including
  under `--skip-plugins`), never silently falling back to the defaults. A late failure rolls the
  whole direct transaction back to the exact pre-run state (a previously absent artifact is
  restored to absence), and receipt publication is validated and parent-owned; concurrent foreign
  changes are refused, not overwritten.
- **Host plugin refresh failures are reported as failures.** A missing host CLI, a failed or
  misunderstood plugin inventory, or a failed refresh is a returned error naming the exact retry;
  the direct update stays committed and the obligation is recorded for the next update, instead
  of a warning over a claimed-successful update. Host-owned plugin caches remain outside
  machinery's rollback.
- **Oracle coverage (`Gt`) credits discovery only.** Static test-reference analysis is labeled
  "static discovery; tests not executed", and bypass shapes (commented-out or unused tests,
  uncalled parsers and helpers, evidence living outside the test) are rejected instead of
  passing.
- **OpenCode governance subprocesses are bounded and validated.** A deadline plus a capped output
  capture terminates runaway children, and unrecognized, truncated, or combined-protocol hook
  responses block instead of best-effort parsing.
- **Unmodeled saga compensation routes are rejected** in refine and compose instead of proving
  green against routes no model defines.
- **Go-crm boundary mapping** covers `internal/testoracle`, and FSM conformance binds to committed
  oracle expectations.
- **Guarded jobs keep their declared wall budget.** The required integration lane and
  `verify-formal` opened their custody scopes through a 60-second bootstrap context that silently
  clamped the declared 3,600,000 ms wall (and with it the formal-provision, image-pull, and suite
  budgets), so a cold JVM/TLC provision on a loaded host died as a scope timeout. The open context
  now derives from the declared wall itself; caps are unchanged and an overrunning job still dies.
- **Large designs verify again.** `machinery verify-formal` failed on designs big enough to
  outgrow three single-shot assumptions in native custody, reporting cleanly terminated and
  reaped engine runs as `cleanup-failed` and then failing every remaining check with
  `STALE_CAPABILITY`. The cleanup grace is now accounted per retirement instead of as one
  absolute instant shared by every job a run ever launches; the broker decides whether a close
  ends it while holding its own lock, instead of a concurrent read that could kill it outright;
  and a control record larger than the socket send buffer, such as a cleanup report naming every
  retired job, is written whole instead of abandoned as a short write. Every shipped budget keeps
  its current value, cleanup time stays bounded, and a job that really does overrun its budget is
  still reported.
- **Custody results are awaited for the job's real deadline.** The broker's terminal-result wait
  is bounded by the effective deadline plus the cleanup grace instead of a fixed window, so a
  legitimately long job on a loaded host is no longer abandoned as a custody error, and a result
  frame arriving while the job channel was still being registered is no longer lost.
- **Assurance store writers release staging in ownership order.** A committed writer no longer
  fails with a custody error (or deletes a fresh live token) when a concurrent writer reacquires
  the staging lock the instant it drops.
- **Same-inode content ABA is rejected on Linux.** Formal transaction journals, directory
  inventories, and the Java runtime closure add a kernel mutation-event witness to their identity
  checks, so a rewrite that leaves coarse inode timestamps equal is still detected.

### Added

- **`machinery recover`**: pass a design directory for a read-only report of an interrupted
  publication: expected outputs with content/mode status, journal and residue locations,
  live-writer status, and the safe recovery decision. `--apply` finalizes only a fully
  revalidated interrupted publication; refusals preserve all evidence.
- **Required integration lane**: `go run ./scripts/integration-lane --lane required` executes
  every infrastructure-dependent suite (Docker-backed checker lifecycle, publication recovery,
  native custody, adapter governance, and the documented checker registry/bind-path example) with
  exact test inventories, pinned runtimes, and real provisioning/teardown accounting. Missing
  infrastructure fails the lane with a diagnostic; nothing is skipped.
- **Native custody for machinery's own subprocesses.** Formal verification and tooling
  subprocesses run inside an owned process-custody scope with registration before launch, bounded
  budgets, and terminal-kill-before-reap cleanup. Limits: cleanup failures are reported,
  never concealed, and custody is process-ownership containment, not a defense against a
  hostile host.
- **Checker container budgets and lifetime ownership.** Every checker container gets closed
  finite budgets (128 MiB memory, half a CPU, a 32-process pid limit), a registered identity,
  and force-removal on every exit path; an output breach aborts and cleans up instead of
  buffering.
- **Standalone contract documents** for native custody and executable test assurance
  (`docs/native-custody-contract.md`, `docs/test-assurance-contract.md`).
- **Executable test assurance: the version-1 substrate and four native test adapters, with no CLI
  surface yet.** This release lands the mechanism, not a consumer command. There is no
  `machinery tdd` command, no `machinery check --store` or `--assurance strict`, and no gate that
  reads `design/assurance/`: a design that commits `plan.json` today sees no diagnostic and no
  change in its finding count from `machinery check`. What ships is the closed version-1 document
  grammar and its validation, the authoritative obligation inventory over a held design snapshot
  (BUILD milestones, committed oracle rows, guard-clause ids, invariant ids, declared runtime
  obligations), and closed adapters for Go (`go test`, Go 1.27.1), TypeScript (`tsc` plus
  `node --test`, Node 26.8.1 / TypeScript 7.0.2), Python (`python -m unittest`, CPython 3.14.7),
  and Elixir (`mix test`, Elixir 1.20.4 / OTP 29). The one place those adapters execute is the
  required integration lane, which verifies every adapter runtime against the exact pins and runs
  frozen per-language probe and native-conformance fixtures under process custody with exact
  per-case accounting; a missing or mismatched runtime fails the lane. Inside that lane each
  registered assertion must be a typed helper call at its frozen line, its outcome is
  witness-bound to that call site, and skips, expected failures, unwitnessed failures, and late
  source mutation are rejected. `docs/test-assurance-contract.md` carries an
  "Implementation status in 0.7.0" section naming shipped versus target surfaces; its section 7
  CLI grammar is the target contract. Until it lands, bind tests to oracle rows with the
  `Gt-tests` stable-id discipline.
- **Replay inputs are retained outside governed sources** (library surface; no command exposes it
  yet). Capture publishes the exact subject, dependency, frozen, and design payload inputs as
  content-addressed blobs in a private assurance store (symlink escapes, hardlink aliases, and
  special files are rejected; the control namespace, `.git`, and `attestations.yaml` are
  excluded), with a typed bundle encoding, a validated export/import archive, and a read-only
  status report that never performs a replay.
- **Reviewed assurance revisions are registered explicitly** (library surface; no command exposes
  it yet). Registration validates lineage (revision 1, or the predecessor plus one with the same
  key), archives the exact control bytes, stages the successor, and advances the store head by
  compare-and-advance under a bounded deadline; drafts never discharge registered requirements,
  and a registered deletion or an untargeted registered mismatch blocks. It commits authored data
  only and returns registered-not-executed; it never claims execution or replay.
- **Release publication is gated on exact-commit verification.** `docs/release-policy.json` is
  the single policy source: a release requires successful `ci`, `formal`, and `security` runs on
  the exact release commit (every required job, including the required integration lane), a tag
  matching the numeric version pattern, and main ancestry; scheduled runs never count as
  evidence. The release workflow runs `scripts/release-policy` against that commit before
  building and again before publishing, the policy is tested by `go test ./scripts/release-policy`,
  and the CI, formal, and security workflows run with read-only permissions. The branch-protection
  payload is documented but has not been applied remotely.
- **windows/amd64 release artifacts.** Each release now publishes a cross-compiled
  `machinery-windows-amd64` binary and `machinery_<version>_windows_amd64.tar.gz` next to the
  Linux and macOS assets, and the publish inventory is accounted exactly across build, manifest,
  and publish. The one-line installer and `machinery update` still refuse Windows, and native
  Windows runtime guarantees (process custody, formal verification, assurance lanes) are not
  claimed.

### Changed

- **Guard clause declarations bind to their owning machine.** `Alpha.matrix.md` binds only
  `Alpha.machine.json` and `Alpha.oracle.md`, so two machines may use one guard name without
  either arming or satisfying the other's falsifying-clause obligation. Upgrade impact: a
  declaration whose oracle rows only another machine could supply is now an error, and one row
  declares one vocabulary (a second `CLAUSES{...}` group on the row, or a `RETIRED{...}` group the
  declaration does not carry, is rejected). Move the declaration to the matrix of the machine
  whose oracle governs the guard, and keep one group per row.
- **Attestation evidence v2 kinds (`plan`/`current`/`historical`).** Example attestations
  migrated to explicit kinds: a `current` claim requires an implementation subject, legacy
  design-only covers are rejected with migration guidance, and implementation-subject changes
  invalidate stale reviews so a review binds the exact scope it attests.
- **Example attestations carry explicit v2 kinds.** All eight bundled examples declare a kind on
  every row: the seven design-only examples attest their behavioral claims as plans, and go-crm
  carries a current implementation review over `examples/go-crm/impl`. `machinery check` on a
  design-only example therefore reports its missing-current warnings and exits 0.
- **Preflight asserts the per-example attestation state.** `scripts/preflight.sh` phase 11 keeps
  go-crm strict under `--warnings-as-errors --impl --complete` and requires each design-only
  example to pass plain `check` with exactly the plan-only warning set derived from its committed
  `attestations.yaml`; any other warning, error, or drift fails. The race-sweep per-package alarm
  moves to 30 minutes in preflight and its CI mirror.
- **Nightly runs use full history.** The nightly golden job checks out the full history so the Ga
  acceptance gate can prove reviewed commits are ancestors of HEAD, and the nightly workflow runs
  the required integration lane.
- **Frozen-test identity requires exact bytes.** Formatting-only rewrites are no longer treated
  as the same frozen test; evidence replay requires exact identity with no whitespace exemption.
- **Event-contract consumer `reads` bind per edge.** A consumer that legitimately reads more than
  its siblings declares its exact read set on its own edge instead of sharing the intersection.
- **Regeneration advice no longer accepts new debt**: baseline/ratchet guidance lists only the
  steps the ratchet actually needs.
- **Milestone packets satisfy the standalone portfolio contract**, and incomplete executable
  assurance declarations are rejected by the declaration validator (a library surface; no
  command reads a declaration in this release).

### Compatibility and migration

Every tightening below is deliberate and was accepted with its own evidence before release; none of
it is incidental. A design that was green on v0.6.11 can still become blocking on 0.7.0 without any
design edit, so each entry states what still works unchanged, what is rejected now, the exact
finding a consumer sees, and the exact edit or command that migrates. The reference counts come
from one real private design that exited 0 on v0.6.11 and reports 78 blocking findings on 0.7.0.

**A directory inventory refuses a host that cannot arm a mutation-event watch.** Unchanged: every
supported host (Linux with inotify, macOS and the BSDs with kqueue, Windows on its NTFS ChangeTime
stamp) enumerates exactly as before. Rejected now: a host where the kernel event channel behind the
granularity-independent half of the witness cannot be opened, because there is then no way to tell
an ABA mutation from a quiescent directory. In practice that means an exhausted
`fs/inotify/max_user_instances`, a kernel built without inotify, or a platform machinery does not
ship for. Finding:

```
design inventory directory design has no reliable change witness: open directory mutation
channel: too many open files
```

Migration: raise the limit (`sysctl fs.inotify.max_user_instances=256`) or free the instances the
user already holds. This is the fail-closed direction and it is deliberate: accepting the
enumeration on a change stamp whose resolution cannot be vouched for is what the 0.7.0 fix above
removes.

**Regeneration stamp.** Upgrading restamps generated artifacts and nothing else. The families that
carry `machinery-version:` are `machines/*.tla`, `machines/*.cfg`, `machines/*.oracle.md`, and
`formal/*.als`; on the reference design that is 254 files whose only changed line moves from
`machinery-version: v0.6.11` to `machinery-version: v0.7.0`. Regenerate with the commands the gate
suite prints for the design (`machinery oracle <design>`, `machinery tla <design>`,
`machinery alloy <design>`, `machinery refine`/`machinery compose`/`machinery pack generate` where
the design carries those families) and commit the stamp-only diff on its own.

**Per-edge consumer `READS` declarations (`Gx-trace`, armed designs only).** Unchanged: a design
that does not arm `machinery:reads-complete` in `ARCHITECTURE.md` is untouched, and a single-
consumer event keeps its existing single `READS{...}` declaration as long as ownership resolves to
exactly one participant. Rejected now: an event that fans out to two or more consumers, where one
`READS{...}` used to satisfy every participant. Each `(event, consumer)` edge owes its own
declaration. Finding (45 on the reference design):

```
event-contract row 7 (event 'markPaid'): no consumer READS declaration for event 'markPaid', consumer 'audit'; state READS{field, ...} beside the event name and name this participant in the matrix consumer column, or waive this consumer cell with '(no reads: <reason>)'
```

Migrate: in the payload matrix, give the row a `consumer` column naming the exact event-contract
participant and write that participant's own `READS{...}`; repeat per consumer. A participant that
reads nothing gets `(no reads: <reason>)` in its consumer cell instead. A waiver never transfers to
a sibling, and one consumer may read a strict superset without imposing it on the others.

**Legacy `READS` with ambiguous consumer ownership.** Unchanged: a legacy declaration with no
consumer column still resolves when the event has exactly one distinct, named owner. Rejected now:
the same declaration when the event has more than one participant, or an unnamed one. Finding (26
on the reference design):

```
Payments.matrix.md:41: legacy READS for event 'markPaid' has ambiguous consumer ownership; add an explicit consumer column matching each event-contract participant
```

Migrate: add the `consumer` column to that matrix table and split the row per participant, as
above. A consumer name that is not an exact event-contract participant is its own finding
(`names consumer '...', not an exact event-contract consumer`), so copy the participant spelling
from the `ARCHITECTURE.md` event-contract table.

**Conflicting `READS` field sets across rows and matrices.** Unchanged: repeating the same edge
with the identical field set in several rows or several matrices. Rejected now: two rows for one
`(event, consumer)` edge whose exact field sets differ. Finding (4 on the reference design):

```
Audit.matrix.md:12: conflicting READS for event 'markPaid', consumer 'audit'; exact field set differs from Payments.matrix.md:41 (all rows and machines for this edge must agree)
```

Migrate: pick the true read set for that edge and make every row and machine state it identically,
or split the edge by naming distinct consumers.

**One complete `READS` group per row.** Unchanged: `READS` as an English verb in residual prose or
in a contract cell ("the projection only READS those rows") is narrative again and is no longer
collected as a declaration; a member is any trimmed, non-empty text in the design's ubiquitous
language, so `READS{Order.id, occurrence time, PAIR KEY}` declares three fields. Empty and
duplicate members are rejected as they always were, and an empty member is now reported as an empty
member rather than as a field name that is neither. Rejected now: a row that carries a `READS{...}`
group wrapped across two markdown lines, or two groups on one row. Findings (1 on the reference
design):

```
Payments.matrix.md:62: event 'markPaid', consumer 'ledger': expected exactly one complete READS{field, ...} declaration per row
Payments.matrix.md:62: event 'markPaid', consumer 'ledger': empty READS member ''; name each field explicitly, and drop the stray separator
```

Migrate: join the wrapped group onto one line so the row carries exactly one closed
`READS{field, ...}`, and drop the stray separator that produced the empty member.

**One `CLAUSES` vocabulary per row (`Gd-idcite`).** Unchanged: a matrix's prose is narrative, not a
declaration. An invariant-coverage bullet that quotes a guard's vocabulary, a contract cell that
mentions "the CLAUSES declaration here", a description of a `RETIRED` site, an enumeration wrapped
across lines, and a row for a non-guard unit are all accepted exactly as they were on v0.6.11.
Rejected now: a named-unit contract row carrying two `CLAUSES{...}` groups, or a `RETIRED{...}`
group that the declaration itself does not carry, because such a row names no single vocabulary.
Finding (1 on the reference design):

```
Payments.matrix.md:18: machine "Payments" guard "guardPaid": malformed CLAUSES declaration; require one named row with a single CLAUSES{...} and optional RETIRED{...}
```

Migrate: keep one `CLAUSES{...}` group, with at most one `RETIRED{...}` inside the same
declaration, on the unit's own row of the named-unit contract table.

**Guard clause ownership resolves against sibling oracles.** Unchanged: a declared guard that no
committed oracle governs at all owes nothing and reports nothing, which is v0.6.11 behavior
restored. This covers the common creation-edge guard: a guard that gates an insert rather than a
guarded transition is silent, and 0.7.0 does not require it to gate a transition. Rejected now: a
declaration in `Alpha.matrix.md` whose falsifying-clause rows only `Beta.oracle.md` could supply.
`Alpha.matrix.md` binds only `Alpha.machine.json` and `Alpha.oracle.md`, so two machines may reuse
one guard name without either arming or satisfying the other's obligation. Finding:

```
Alpha.matrix.md:23: machine "Alpha" guard "guardApproved": owning oracle has no transition governed by this guard, but Beta's does; another machine cannot supply it
```

Migrate: move the declaration to the matrix of the machine whose oracle governs the guard.

**Oracle coverage (`Gt-tests`) credits active discovery only.** Unchanged: a suite that names each
stable id as a whole token in an active test body, or that cites the oracle file by name inside a
single test with a connected read-parse-assert flow, still passes; `Gt` still executes no tests and
labels itself `static discovery; tests not executed; unsupported parser structures remain
uncovered`. Rejected now: evidence that is not executable at all, which used to pass: commented-out
or disabled tests, unused declarations, uncalled parsers and helpers, and mentions outside a test
body. Finding:

```
Payments.oracle.md: 12 of 40 stable ids appear in no test file (PAY-001, PAY-002, ...); key the tests on the stable ids, or parse the committed table at runtime by naming Payments.oracle.md in a test
```

Migrate: move the id references into active test bodies, or keep one real test that reads and
parses the committed oracle table.

**Attestation evidence v2 kinds (`Gv-attest`).** Unchanged: `attestation_version: 1` files still
parse, and every claim the vocabulary classifies as design-only keeps passing as an implicit plan
judgment, with a note asking for the explicit kind. `machinery check <design>` on a design-only
design reports its missing-current warnings and still exits 0. Rejected now: a v1 row for one of the
three claims the vocabulary classifies as `current` (`gt.conformance-test-shape`,
`g4.standin-coverage`, `g4.pack-event-discipline`), which used to pass as a design-only cover; a v2
`kind: current` row with no complete implementation subject; and a `kind: current` row evaluated
without `--impl`. Findings:

```
GV_MISSING_IMPLEMENTATION_SUBJECT: gt.conformance-test-shape has legacy design-only covers; review the complete implementation/test scope and run machinery attest --design <design> --claim gt.conformance-test-shape --kind current --impl <root> --attestor <reviewer> --date YYYY-MM-DD, or explicitly recast as v2 kind=plan
GV_MISSING_IMPLEMENTATION_SUBJECT: gt.conformance-test-shape requires a complete implementation subject
GV_IMPL_REQUIRED: gt.conformance-test-shape requires --impl <reviewed-root>; the receipt locator grants no read authority
```

Migrate, whichever is true of the design. Design-only, no implementation to review yet: set
`attestation_version: 2` and add `kind: plan` to the row; `machinery check` then warns
`gt.conformance-test-shape: plan only; current implementation review missing` and exits 0. A real
reviewed implementation: run

```
machinery attest --design <design> --claim <claim> --kind current --impl <root> --attestor <reviewer> --date YYYY-MM-DD
```

and pass `--impl <root>` to every later `machinery check` that evaluates the claim. A subject whose
bytes move after the review invalidates it (`GV_STALE_CONTENT`, `GV_SCOPE_INVENTORY`); review the
changed scope and attest again. `ga.review-quality` is the one historical-class claim: it takes
`kind: historical` and establishes history, not current implementation approval. Every other claim
in the vocabulary is plan-class and takes `kind: plan`.

**`--impl` arming.** Unchanged: `G4-import` and `Gt-tests` still run only when `--impl` is
supplied; `--complete` still requires `--impl`; an explicit `--gate g4` or `--gate gt` without
`--impl` is still an invocation error. Changed: a decomposed parent with no `machines/` and a
supplied `--impl` now also runs `Gt` when the parent owns relational obligations (a policy or
isolation annotation, or a committed relational oracle under `formal/`). v0.6.11 dropped `Gt` for
that parent and reported nothing, so its policy and isolation test obligations were invisible.
The selection note says so: `gt checks parent-owned relational obligations`. Migrate: give the
parent's own tests the parent oracle's stable ids, or run without `--impl` to keep the design-only
selection.

**`G4-import` ignore globs.** Unchanged, and stated because it is easy to reach for the wrong file:
`G4` prunes the implementation tree with the `ignore` glob list of the Architecture Contract in
`ARCHITECTURE.md`, not with `.machineryignore`. `.machineryignore` at the design root still governs
design-tree reads only, with the same gitignore-shaped subset (no negations, no re-inclusions).
Neither file changed in 0.7.0.

**Regeneration advice no longer proposes `machinery baseline`.** Unchanged: `machinery baseline
<design> --impl <dir>` remains a supported explicit command. Changed: a version-skew or
regeneration message on a design carrying a ratchet no longer lists it among the regeneration
steps, because a baseline snapshot accepts tolerated import offenders and can widen accepted
architecture debt. Its own output and help now say that a baseline accepts debt and needs review
even when it proposes no new dependency rule. Migrate: nothing, unless a script pasted the printed
regeneration list verbatim, in which case drop the baseline line from it and run baselines
deliberately.

**Installer reruns and update receipts.** Unchanged: `machinery update` without selectors, and
explicit `--home`/`--target` selectors, behave as before; selectors still cannot be combined with
`--bootstrap-defaults`. Changed: rerunning the one-line installer over an existing install now
refreshes the complete recorded home, native-target, and host-plugin plan, preserving each recorded
group's copy/symlink mode, instead of refreshing the default homes only. Rejected now: a corrupt,
unsupported, or unsafe receipt, an unsafe artifact or parent substitution, and uncertain plugin
ownership all fail closed with a diagnostic naming the problem, including under `--skip-plugins`,
where older releases silently refreshed the default homes instead. Safely edited or missing owned
regular content with unchanged safe parents is repaired to exact current-release content. Migrate:
a failing rerun names the problem; repair or remove the reported artifact and rerun, or run
`machinery install` with explicit `--home`/`--target` selectors to record a fresh plan.

**Host plugin refresh failures are returned failures.** Changed: a missing host CLI, a failed or
misunderstood plugin inventory, or a failed refresh is now a returned error naming the exact retry,
where 0.6.11 reported a warning over a claimed-successful update. The direct update stays
committed, and the retry obligation is recorded for the next update. Host-owned plugin caches are
never rolled back by machinery. Migrate: run the exact command the error names (for Claude Code,
`claude plugin update machinery@machinery`), or pass `--skip-plugins` to opt out of host plugin
refresh entirely.

**windows/amd64 release artifacts without installer support.** Added, with a stated limit: releases
now publish `machinery-windows-amd64` and `machinery_<version>_windows_amd64.tar.gz`, but the
one-line installer and `machinery update` still refuse Windows, so the asset is downloaded and
placed by hand. Native Windows runtime guarantees are not claimed: process custody, formal
verification, and the assurance lanes are unix-only and refuse on Windows. Migrate: use Linux or
macOS for the full toolchain.

**Contributors: the `SKIP_PREFLIGHT` bypass is gone.** Rejected now: `SKIP_PREFLIGHT=1 git push`.
`scripts/preflight.sh` no longer honors the variable, and the pre-push hook, `CONTRIBUTING.md`, and
the `make hooks` message say so. Migrate: run the full preflight, or remove the hook locally
(`git config --unset core.hooksPath`) and accept that the exact-commit release gate is the only
remaining check.

**Known limitation, unchanged since 0.6.x: directory change witnesses on Linux before 6.13.** The
fail-closed directory inventories (`internal/dirscan`, the gate confinement walker, the oracle
inventory, the governance hook's design inventory) detect concurrent mutation through the
directory's inode change stamp. On Linux kernels without multigrain timestamps (6.12 and older,
including Ubuntu 24.04 LTS at 6.8) that stamp is one timer tick coarse and `STATX_CHANGE_COOKIE`
is not exposed, so a create-then-delete completed inside one tick is not detected; hosted CI
passes because its runner kernel is 6.17. Not a regression. A kernel-event
witness (inotify, kqueue) is under review for 0.7.1. Until then, inventory results on such kernels
are stamp-witnessed only.
