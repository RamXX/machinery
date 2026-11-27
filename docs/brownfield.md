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

A rehearsal reads only what is committed. To try a links file before adopting it, commit it on
a scratch branch of a clone of the project and rehearse there. Every reason and offered waiver
it writes names the commit of your project that it rehearsed, never the commit of its snapshot.

`rehearsal.md` (and `rehearsal.json`) lists the files adoption would write with their sizes, the
waivers it would add by rule, the baselines, what it would owe by kind, the result of the check,
the stories `plan` would create by kind and tier, and what blocks adoption: each stage that fails
or is not demonstrated, with its first findings. A finding of a cross-reference rule comes with
the waiver that would cover it, offered and never written. A finding about the sources
themselves, such as a machine that does not lint or a failure catalog with a row outside its
tables or an id listed twice, is listed under "Fix in the project first": no waiver covers it,
and `adopt --write` owes nothing until it is fixed. Fix those in the project, commit, and
rehearse again until nothing you did not choose blocks.

The rehearsal also tells apart what your tests already claim. Under "Owed" it counts the ids a
test names (`claimed by a test, no receipt yet`), the ids no test names, and the ids your own
sources mark as planned; the plan has the usual stories for the ids no test names, one `Wire evidence`
story per test file for the claimed ones and none for an id you had planned, so a project with a thousand claimed ids is not asked
for a thousand stories. A claim is not evidence: every one of them is still owed until its test
emits a receipt, or, for a Go suite, until a run through the adapter sees the test pass
(section 7).

## 3. Look before you write

Point `adopt` at the area that hurts most. Without `--write` it changes nothing:

```sh
machinery adopt --paths 'internal/shipping/**' --e2e 'e2e/**'
```

The report says what it found: languages, code files in scope, test files (how many are unit
tests, how many use test doubles), design sources, and the state of the backlog. Reading
the backlog runs `bd`; `adopt`, `status` and `check` skip it with `--no-backlog` or with
`MACHINERY_NO_BACKLOG=1` in the environment and say `backlog: not read`, and a `bd` that is
missing or fails is never an error for them (`verify`, `trace` and `owed` never run it).

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
claimed. Adopt writes only `machinery.yaml` and files under `design/` and the gen directory:
when you keep a model elsewhere with a Markdown rendering of your own beside it, adopt leaves
that file alone and says when it is stale (`machinery gen` rewrites it when you run it).

Here no test names an id yet. In a project whose tests already point at the design, an id a
test claims is owed with the reason `claimed at adoption`, and an id your own sources mark as
planned (`import.planned`: a sentence that opens an invariant's statement or a scenario's
description, a cell of a row of your failure catalog, each naming its story) is owed with `planned
at adoption` and that story.
`machinery status` then counts the owed ids by what stands behind each: claimed by a test with
no receipt yet, named by no test, planned, and, once a run has proven some, proven and still
listed. A story id your mark names is kept as it is: `plan --apply` creates nothing for an id
whose story is not in this backlog and says so, and a mark that captures more than a story id
stops `adopt` before it writes.

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
with the name declared under `systems` in `machinery.yaml`; a row about a machine as a whole (a
queue that stalls) takes `machine:<id>`, which covers none of the machine's failure states,
failure events and invokes; a row that names nothing in the system at all is declared
`no_subject`, with the reason. A catalog that has a column for the required behavior names it
in `columns.required`, and an entry may then leave `required` out. A row whose entry names a
subject and gives no required value is counted on its own (`name a subject and no required
value`), so the rows that only wait for that value are told apart from the rows nobody linked;
it is unlinked all the same. It is no error: `gen` and `adopt` go on, the row is owed like any
other, and stages `failure-modes` and `xref-failure` print the same counts of linked, unlinked,
subject-only and subject-less rows. A `required` value that is given and is none of the four is
an error, fixed in the links file or the cell.

**Read your machines as they are.** `import.machines` (`docs/formats.md` section 5.2.1) takes
globs anywhere in the project, so a lifecycle file kept beside its service is read with the
rest. `states: pascal` turns `on_hold` into the domain state `OnHold`;
`terminal: {from_root: <key>}` turns a root note that a machine never ends into `_terminal` on
the states it is never left in; and a register that lists the transitions your code makes
(`import.machines.register`) is checked both ways against the machines.

**Say which events are not caller actions.** A dialect of your own rarely has a key for it, and
you do not have to add one to every file. `import.machines.annotate` says it per glob of files
or per machine id: `external_events` for what arrives from outside the caller (a timer, a
callback, another machine), `failure_events` and `failure_states` for what reports and records
a failure, each a list of names and `*` patterns (`"*_refused"`, `"*_failed"`), and `aliases`
for an event that is a model action under another name (`handed_back: giveBack`). The
"not an action" findings stop for the names you declare, and the "looks like a failure" warnings
for the failures you declare. An external or aliased event whose name looks like a failure is
still hinted: list it in `failure_events`, or dismiss the hint by name in `not_failures`, with
the reason. An alias and `failure_events` go together when the action is itself the failing
outcome (the model action `fail`, the events `send_failed` and `verify_failed`). A machine that
writes the refusal of a command as an event of its own (`cancel` and `cancel_refused`) declares
it in `refusal_events`, with the action and the reason
(`cancel_refused: {action: cancel, reason: "a paid order is not cancelled"}`): it is no event
from outside and no alias, and a declared refusal is neither hinted nor asked for a failure
mode. That holds only for a refusal, where nothing failed: an event on which the machine
moves into a failure state it was not in is refused as one (from a state that is not a failure
state, or from one failure state into another), while one that leaves a failure state where it
is (`failed` on `cancel_refused` stays `failed`) is a refusal, and every declaration is printed
with its reason. A
pattern that matches nothing in a machine it is applied to is an error, a pattern
of `*` only is refused, and an event that is an action of the model stays a finding whether you
list it as external or give it an alias: the config cannot hide a command. Every alias and every
refusal event in effect
is a line of `xref-model-machine`, so the list of names you bridged stays in view.

## 6. Turn what is owed into stories, beside the ones you have

```sh
machinery owed list
machinery plan --apply
machinery waves
```

`plan --apply` adds one story per group of owed ids to the same backlog, with dependency edges
and a tier label. Your existing stories are untouched. To attach ids to a story you already have,
use `machinery plan --link <story> <id>...`. `waves` shows what can run in parallel. Ids a test
claimed at adoption get no story of their own: `plan` adds one `Wire evidence <test file>` story
per file (per directory past `plan.wire_max_files` files) that lists the ids the file must emit
receipts for.

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

A receipt proves only when `machinery run` wrote its file and its `test` names a test file that
claims the id (`docs/formats.md` section 10.2): the script's receipts name
`shipment_e2e.sh`, the file that carries the `machinery:verifies` comment. A receipts file typed
by hand, or edited after the run, is refused and proves nothing. That guards against mistakes,
not against someone who can run commands as the tool.

**A Go suite you already have.** When your tests already name the ids they verify in comments,
you do not have to rewrite them to get evidence. Run the suite through the adapter:

```
machinery run --adapter go-test-json -- go test -count=1 ./e2e/...
machinery owed sync --prune-only
machinery status
```

machinery adds `-json`, reads the event stream and writes the receipts itself
(`docs/formats.md` section 10.3). An id gets a pass when every test function its marker belongs
to passed and the package ended ok: the function under the marker when the marker is in the
comment directly above it, every test function of the file when it stands above no function. A
marker directly above a fuzz test, a benchmark or any other function that is no test counts
nothing, and the summary of the run names it. A failed function writes
a fail, a skipped one a skip (a parent with a skipped subtest too), and one that did not run
writes nothing. In a package that failed, a test that hangs past `-timeout`, calls `os.Exit`
or panics writes a fail ("started, no verdict"), a function a `-run` filter left out or the
crash never reached writes nothing, and no pass of the package counts; a package that ran none
of its tests (a build that fails, a `TestMain` that exits first) writes a fail for every
claimed function. A
stream cut short leaves the run incomplete, naming the receipts file to delete; a pass answered from
the test cache counts nothing, and machinery adds `-count=1` to keep that from happening. A
verdict belongs to a file only when no other Go file of its directory defines a function of the
same name that begins with `Test` and takes one parameter, whatever its build tags or the
way its parameter is written: rename such a twin, or neither counts. The summary of the run
says how many ids of the packages it ran were claimed and not run, and the number for the
whole project once. The command is held narrow on purpose, against
mistakes: `go test` with relative package patterns and the usual test flags, in the module of
the project root, for the machine it runs on; `-C`, `-modfile`, `-exec`, a `-run` that
selects subtests and the like run through `--streams` (below). A module path that two tracked
`go.mod` files outside `testdata` and `vendor` declare, or a tracked `go.work` that leaves out
the module of the claiming test files, is refused, and once `machinery.yaml` has an
`evidence.adapter` block `machinery check` reports both (stage `trace`). The run's header records the go version, the `go` binary,
`GOFLAGS`, `GOWORK`, `GOOS`/`GOARCH` and the digests of `go.mod` and `go.work`, and
`verify` and `proof` print them. The adapter trusts your test code and the environment of the
run exactly as much as a receipt a test emits: it catches a stale cache, a twin test, a file
that was not built, a wrong directory or a lost fail, not code or modules planted on purpose.

This evidence is weaker than a receipt, and every report says so. It shows that a test which
names the id passed, not that the test checked the id. `status`, `verify` and `proof` count it
apart (`proven: 40 (31 by adapter at file granularity)`), a transition case (`tc`) is never
proven this way because it needs the effects the test observed, and by default an id of a
critical or vital area is not either: those need a receipt the test emits after its own
assertions. `evidence.adapter` in `machinery.yaml` tightens or widens this: which kinds and
tiers the adapter may prove, and whether a marker at the top of a file is enough or the marker
must stand above the function (`min_granularity: function`). If your suite runs through a make target or a script that starts
`go test` itself, run that through `machinery run --adapter go-test-json --streams -- make e2e`
and have it write each `go test -json` output to a file in the directory machinery names in
`MACHINERY_GO_TEST_JSON` (a file may hold several runs, appended one after the other: each is
judged on its own, and a fail in any of them is a fail; a run that starts before the one before
it ended, a killed run and its retry or two runs writing at once, leaves the run incomplete); there the wrapper must pass `-count=1` itself, and because machinery
cannot see what such a run built, every package it tests needs an entry in
`evidence.adapter.sources` naming the files it is built from. One entry
`{package: "<module path>/e2e/...", dir: true}` gives every package it matches the test files of
its own directory. If you build your tests from copies staged in another directory, an entry with
`files` maps the staged package back to the files that carry the markers, and those may be
sources you keep under `trace.tests` without the `_test.go` name (under a `testdata`
directory, say). When the wrapper tests several such sets, name each one: `go test -c` the set,
then run the binary under `go tool test2json -t -p <name>`; named by the import path of the
directory its sources are kept in, one entry with a wildcard maps each set to its own sources
(`docs/formats.md` section 10.3). The entries are read only with `--streams`.

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
failure modes in Markdown tables and your own trace markers in comments are read as they are
(a marker you also use for other text goes in `trace.markers.scenarios_exact`, which claims only
when the text is a scenario name),
through `import` and `trace.markers` in `machinery.yaml` (`docs/formats.md` sections 5.2.1, 5.4.1
and 8.1); the fixture `e2e/testdata/fixtures/library` is a small example.
