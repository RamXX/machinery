# Adopting machinery in an existing project

You have code, tests, a way of working and a `bd` backlog, and work is in flight. This guide
brings one area of the project under machinery without a rewrite, without a freeze and without
touching the backlog you have. Every command in a `sh` block is executed, in order, by the test
suite (`brownfield_walkthrough.txtar`) on a small existing Go project, the `brownfield` fixture:
a depot with a shipping area and a billing area, unit tests, a smoke script and its own backlog.

The contract is `docs/formats.md` sections 18 (`assess`), 17.3 (`adopt --rehearse`) and 17.2
(`adopt`). The order is assess, rehearse, then adopt. Three properties hold at every step:

- `assess` and `adopt --rehearse` write nothing in the project, and `adopt` never changes existing
  code: it writes `machinery.yaml` and files under `design/`.
- Only the paths you point it at are covered. Everything else is left alone.
- Coverage only grows: the scope does not narrow, baselines do not grow without a wider scope,
  and nothing proven becomes owed (`machinery check --base <ref>`).

## 1. Assess before you adopt

Before anything is written into the project, see where it stands against the design it already
has. Write a first `machinery.yaml` by hand and keep it outside the project: the scope you care
about, where the end-to-end tests live, and the design you already keep in your own formats
(`import` and `trace.markers`, `docs/formats.md` section 2). Then assess:

```sh
cat > ../depot.machinery.yaml <<'EOF'
version: 2
scope: ["internal/**"]
tests:
  e2e: ["e2e/**"]
EOF
machinery assess --config ../depot.machinery.yaml --out ../depot-assessment
```

`assess` takes a snapshot of `HEAD` with `git archive` into the out directory and runs
everything there: the inventory `adopt` would report, every lint stage, `gen`, the trace, tests
and arch stages, a sanity run of the formal tools on every machine and model, how the model
changed since it was first committed, a census of decision records (`assess.decisions`) and agent
instruction files, and the tracker export, read as a file. Nothing in the project changes, not
even `.git`. Files you have not committed are not assessed; the report counts them.

`report.md` in the out directory is for the team, `report.json` for tools, and `stages/` holds
the full output of every stage. Here the tests section fails with two `unit-test` findings, one
per area: what adoption will baseline rather than delete.

## 2. Rehearse the adoption

`assess` says where the project stands; `adopt --rehearse` says what adopting it would take. It
takes another snapshot, runs `adopt --write`, `machinery check` and `machinery plan` there, and
never runs the backlog tool:

```sh
machinery adopt --rehearse --config ../depot.machinery.yaml --out ../depot-rehearsal
```

A design-side file you are still writing (the links of a failure catalog, a status or risk file,
the config itself) does not have to be committed into the project to be tried: put it in a
directory of its own, at the path it will have in the project, and pass `--overlay <dir>` to
`assess` or to `adopt --rehearse`. The files are laid over the snapshot, and the reports list each
with its sha256 and say that the result is not the result of `HEAD`. An overlay holds only the
config and the design sources a config names (`docs/formats.md` section 18.2), judged by the
`machinery.yaml` of `HEAD` or else by `--config`: it changes what the design says, never the
evidence.

`rehearsal.md` (and `rehearsal.json`) lists the files adoption would write with their sizes, the
waivers it would add by rule, the baselines, what it would owe by kind, the result of the check,
the stories `plan` would create by kind and tier, and what blocks adoption: each stage that fails
or is not demonstrated, with its first findings. A finding of a cross-reference rule comes with
the waiver that would cover it, offered and never written. A finding about the sources
themselves, such as a machine that does not lint or a failure catalog with a row outside its
tables or an id listed twice, is listed under "Fix in the project first": no waiver covers it,
and `adopt --write` owes nothing until it is fixed. Fix those in the project, commit, and
rehearse again until nothing you did not choose blocks.

## 3. Look before you write

Point `adopt` at the area that hurts most. Without `--write` it changes nothing:

```sh
machinery adopt --paths 'internal/shipping/**' --e2e 'e2e/**'
```

The report says what it found: languages, code files in scope, test files (how many are unit
tests, how many use test doubles), design sources, and the state of the backlog.

## 4. Adopt the area

```sh
machinery adopt --paths 'internal/shipping/**' --e2e 'e2e/**' --write
echo '.machinery/' >> .gitignore
git add -A
git commit -q -m "adopt machinery for the shipping area"
machinery check
```

`adopt` does not edit `.gitignore`; it tells you to add `.machinery/`, where the receipts of test
runs are kept. Receipts are evidence of one run at one commit and are not committed.

`check` passes on day one. The unit tests the area already has are listed in
`design/tests-baseline.yaml` and print as warnings; nobody has to delete them today. A new unit
test or a new test double in the area fails from now on: new evidence comes from the real
system. The billing area is outside the scope and is not looked at.

## 5. Model the riskiest stateful resource

Not the whole system: the one resource whose legal next step depends on its past. Here it is the
shipment. Write its Modelith model and its machine from the code as it is (the fixture
`brownfield-design` holds them), and say why creation has no event:

```sh
cp -R "$MACHINERY_REPO/e2e/testdata/fixtures/brownfield-design/." .
printf 'waivers:\n  - {rule: action-event, id: "act:Shipment.create", reason: "creation enters the initial state"}\n' >> machinery.yaml
machinery adopt --write
machinery check
git add -A
git commit -q -m "model the shipment lifecycle"
```

The second `adopt --write` finds the new design sources, adds them to `machinery.yaml`, runs
`machinery gen` and puts every obligation into `design/owed.yaml` with the reason `unproven at
adoption`. That list is your baseline of what was never proven. Nothing fails; nothing is
claimed.

**Mark the failures.** The code you model already fails somewhere: a status value such as
`failed`, a worker that reports a crash, a lease that runs out. Tag the state that records it with
`"tags": "failure"` and list the events that report it in the root `_failure_events`
(`docs/machine-format.md`, "Failure idioms"); `machinery lint` warns about a state or event whose
name looks like a failure and is not marked. Each failure state and failure event then needs a
failure mode in `design.failure_modes`. The next `adopt --write` waives the ones that have none,
with the reason `unmodeled at adoption`, so the check still passes; when you write a failure mode
for one, its waiver becomes a `stale waiver` ERROR and you delete it. The list only shrinks.

**Bind every machine.** Each machine says what in the model it is for: a lifecycle machine names
its entity, and an operational machine lists in `_serves` the entities and actions it exists for
(`docs/formats.md` section 5.2). Machines read through `import` get their binding from
`import.machines.bind` when the source does not say. `adopt --write` waives the machines that
bind to nothing yet and the entities that have actions and no lifecycle machine, with the reason
`unbound at adoption`; a waiver goes stale when its machine is bound or its lifecycle written.
An entity whose state lives in two attributes (an order's payment and its delivery) gets one
lifecycle machine per attribute, each an aspect with its own `_aspect` name
(`{lifecycle_of, aspect}` in `import.machines.bind`). An operational machine's events are held to what it serves: an action
it serves, or an event it lists in `_external_events`.

**Link the failure catalog.** A catalog read through `import.failure_modes` owes every row, and
stage `xref-failure` is not demonstrated while one row names no subject: it prints how many are
linked and how many are not, and one linked row among hundreds is no pass. The links file
(`docs/formats.md` section 5.4.1) takes many rows per entry, `ids: [RISK-10..RISK-24, RISK-31]`,
so a catalog is linked group by group. A row about the system as a whole or a part of it that
is no entity, dependency or machine (a build, a deployment) takes the subject `system:<name>`,
with the name declared under `systems` in `machinery.yaml`; a row that names nothing in the
system at all is declared `no_subject`, with the reason.

**Read your machines as they are.** `import.machines` (`docs/formats.md` section 5.2.1) takes
globs anywhere in the project, so a lifecycle file kept beside its service is read with the
rest. `states: pascal` turns `on_hold` into the domain state `OnHold`;
`terminal: {from_root: <key>}` turns a root note that a machine never ends into `_terminal` on
the states it is never left in; and a register that lists the transitions your code makes
(`import.machines.register`) is checked both ways against the machines.

## 6. Turn what is owed into stories, beside the ones you have

```sh
machinery owed list
machinery plan --apply
machinery waves
```

`plan --apply` adds one story per group of owed ids to the same backlog, with dependency edges
and a tier label. Your existing stories are untouched. To attach ids to a story you already have,
use `machinery plan --link <story> <id>...`. `waves` shows what can run in parallel.

Decisions you already keep in a file move into the backlog record by record:
`machinery decisions import --file docs/DECISIONS.md --record '^## D[0-9]+'` prints one decision
per record, with the design ids its text names, and `--apply` creates them; the file is never
touched, and the last line says how many lines your instructions would shed once agents stop
loading it. With a backlog, `machinery waves --freeze` and `machinery intake plan --apply` wait
for the checker's review (`machinery review`); an adopted project may set `review.backlog` and
`review.design` to `off` until it plans that way, and the ratchet reports turning one off later.

## 7. Prove one story end to end

A story is done when its ids are observed on the real system. The fixture `brownfield-proof` adds
what the story "Prove shipment pack" needs: a `machinery:serves` comment in the shipping package
and an end-to-end script that builds the real binary, drives it and emits receipts.

```sh
cp -R "$MACHINERY_REPO/e2e/testdata/fixtures/brownfield-proof/." .
git add -A
git commit -q -m "prove packing end to end"
machinery run -- sh e2e/shipment_e2e.sh
machinery owed sync --prune-only
git add -A
git commit -q -m "prune the owed list"
machinery verify
STORY=$(machinery waves | awk '/Prove shipment pack/ {print $1; exit}')
machinery proof --story "$STORY" --post
machinery story "$STORY"
machinery status
```

`owed sync --prune-only` removes exactly the ids the run proved. `verify` passes: everything is
proven or still owed. `story` is the gate for that one story; a consequential or critical story
would also need a fresh verifier (`docs/pm.md`).

## 8. Hold the line, then widen

```sh
machinery check --base HEAD~2
machinery adopt --paths 'internal/billing/**' --write
machinery check
```

The ratchet compares with an earlier commit: nothing that was proven there is owed now. When the
team is ready, a second `adopt` widens the scope to the next area; its unit tests join the
baseline, and from then on the area is held to the same rules.

## What you did not have to do

Rewrite code. Stop the stories in flight. Migrate the backlog. Model everything. Delete tests.
Each of those can happen later, one story at a time, and the owed list and the baselines show
exactly how far along you are.

Nor did you have to convert a design you already keep. Machines in a JSON dialect of your own,
failure modes in Markdown tables and your own trace markers in comments are read as they are,
through `import` and `trace.markers` in `machinery.yaml` (`docs/formats.md` sections 5.2.1, 5.4.1
and 8.1); the fixture `e2e/testdata/fixtures/library` is a small example.
