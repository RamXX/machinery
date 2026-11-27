# Adopting machinery in an existing project

You have code, tests, a way of working and a `bd` backlog, and work is in flight. This guide
brings one area under machinery without a rewrite, a freeze or a migration of the backlog.
Every command in a `sh` block runs, in order, in the test suite (`brownfield_walkthrough.txtar`)
on the `brownfield` fixture: a small Go depot with a shipping and a billing area, unit tests, a
smoke script and its own backlog. The contract is `docs/formats.md`.

| Step | Command | Writes in the project | Contract |
|---|---|---|---|
| 1. Assess | `machinery assess --config <file> --out <dir>` | Nothing | 18 |
| 2. Config | a `machinery.yaml` kept outside the project until you adopt | Nothing | 2 |
| 3. Rehearse | `machinery adopt --rehearse --config <file> --out <dir>` | Nothing | 17.3 |
| 4. Links file | the subject and required value of each catalog row | One design file | 5.4.1 |
| 5. Adopt | `machinery adopt --paths <glob> --write` | A `machinery.yaml` when there is none, `design/` | 17.2 |
| 6. Evidence | `machinery run`, with `--adapter go-test-json` for a Go suite | Receipts, never committed | 10 |
| 7. Owed sync, verify | `machinery owed sync --prune-only`, `machinery verify` | `design/owed.yaml` | 11, 12 |
| 8. Plan | `machinery plan --apply` | Stories in your backlog | 15.2 |
| 9. What stays owed | `machinery status` | Nothing | 14 |

`assess` and the rehearsal write nothing in the project, not even in `.git`. `adopt` never
changes existing code. Coverage only grows: the scope does not narrow, baselines do not grow
without a wider scope, and nothing proven becomes owed (`machinery check --base <ref>`).

## 1. Assess

Write a first `machinery.yaml` by hand, outside the project: the scope you care about and
where the end-to-end tests live. Then assess:

```sh
cat > ../depot.machinery.yaml <<'EOF'
version: 2
scope:
  - "internal/**"
tests:
  e2e:
    - "e2e/**"
EOF
machinery assess --config ../depot.machinery.yaml --out ../depot-assessment
```

`assess` runs on a snapshot of `HEAD` (`git archive`): the inventory, every lint stage, `gen`,
the trace, tests and arch stages, the formal tools, the history of the model, a census of
decision records (`assess.decisions`) and agent instruction files, and the tracker export, read
as a file. `report.md` is for the team, `report.json` for tools, `stages/` holds every output.
Here the tests section fails with two `unit-test` findings: what adoption will baseline.

## 2. Say where your design already lives

A design you keep in formats of your own is read as it is, never converted and never written:

| Key | Reads |
|---|---|
| `import.machines` | Machines in a JSON dialect of your own: `files` (globs anywhere), `states` (`capitalize`, `pascal`, `keep`), `rename` per level, `terminal.from_root`, `register` (the transitions your code makes, checked both ways) and `bind` (`lifecycle_of`, with an `aspect`, or `serves`). |
| `import.machines.annotate` | What the dialect has no key for, per glob of files or machine id: `external_events`, `failure_events` and `failure_states` (names and `*` patterns), `aliases` (an event that is a model action under another name), `refusal_events` (an event that writes the refusal of an action, with a reason) and `not_failures` (a failure-looking name dismissed, with a reason). |
| `import.failure_modes` | A failure catalog kept as Markdown tables: `file`, the `id` pattern, `columns` (`failure`, `behavior`, and `required` when the catalog has one) and `links` (step 4). |
| `import.planned` | Your own planned marks: `in` (`model-statement`, `scenario-description` or `failure-row`) and a `pattern` whose group `story` names the story. |
| `trace.markers` | Your own markers in comments: `serves`, `verifies`, `scenarios`, and `scenarios_exact` for a marker that also carries other text. |
| `systems` | Parts of the system a failure mode may be about that the design does not model. |

```yaml
import:
  machines:
    files: ["machines/*.json"]
    states: pascal
    rename:
      state: {skip: _ignores}
    bind:
      "machines/loan-desk.json": {lifecycle_of: Loan}
      hold_queue: {serves: ["act:Loan.lend"]}
    annotate:
      "machines/*.json":
        failure_events: ["*_failed"]
      loan_desk:
        external_events: ["due_*"]
        failure_states: ["*_failed"]
        aliases: {handed_back: giveBack}
      hold_queue:
        external_events: [notice_sent]
        aliases: {lend_requested: lend}
  failure_modes:
    file: docs/risks.md
    id: "^RISK-[0-9]+$"
    columns:
      failure: ["Failure", "What goes wrong"]
      behavior: ["Required behavior", "Expected"]
      required: ["Required"]
    links: design/risk-links.yaml
  planned:
    - {in: scenario-description, pattern: '^Planned \(story (?P<story>[a-z0-9-]+)\)\.'}
systems:
  - {name: build, about: "how the desk is built and released"}
trace:
  markers:
    serves: ["Implements:"]
    verifies: ["Checks:", "Proves:"]
    scenarios_exact: ["Objective:"]
```

The config cannot hide a command: an event that is an action of the model stays a finding
whether you list it as external or alias it. A pattern that matches nothing in a machine it
applies to is an error, and a pattern of `*` alone is refused. An event that moves the machine
into a failure state it was not in is no refusal event. Stage `xref-model-machine` prints every
alias and refusal event in effect, with its reason. The fixture `library` is read this way.

## 3. Rehearse

The rehearsal takes another snapshot, runs `adopt --write`, `machinery check` and
`machinery plan` there, and never runs the backlog tool:

```sh
machinery adopt --rehearse --config ../depot.machinery.yaml --out ../depot-rehearsal
```

`rehearsal.md` (and `rehearsal.json`) holds:

| Part | What |
|---|---|
| Files, waivers, baselines | What adopt would write, with sizes, the waivers by rule and the baseline entries. |
| Owed | By kind, then by claim: claimed by a test with no receipt yet, named by no test, planned. |
| Check | The result of `machinery check` and each stage's outcome. |
| Plan | The stories `plan` would create, by kind and tier, wire-evidence stories last. |
| Blocks adoption | Each stage that fails or is not demonstrated; a cross-reference finding comes with the waiver that would cover it, offered and never written. |
| Fix in the project first | Findings about the sources themselves (a machine that does not lint, a catalog row outside its tables): no waiver covers them, and `adopt --write` owes nothing until they are fixed. |

Every reason and offered waiver names the commit of your project, never that of the snapshot.
A rehearsal reads only what is committed: to try a config or a links file first, commit it on a
scratch branch of a clone and rehearse there.

## 4. Link the failure catalog

A catalog read through `import.failure_modes` owes every row. Stage `xref-failure` prints how
many rows are linked and is not demonstrated while one is unlinked: one linked row among
hundreds is no pass. The links file gives each row its subject and required value:

```yaml
version: 1
links:
  - {id: RISK-1, kind: abuse, subject: "act:Loan.lend", required: refuse}
  - {id: RISK-2, subject: "state:loan_desk.ScanFailed", required: surface}
  - {id: RISK-3, subject: "event:loan_desk.scan_failed"}
  - {id: RISK-4, subject: "event:hold_queue.notice_failed", required: retry-bounded}
  - {ids: [RISK-10..RISK-24, RISK-31], subject: "system:build", required: fail-closed}
  - {id: RISK-32, subject: "machine:hold_queue", required: surface}
  - {id: RISK-40, no_subject: "the shelf list is kept on paper, outside the system"}
```

| Entry | Means |
|---|---|
| `id` or `ids` | One row, or many: ids and inclusive ranges (`RISK-10..RISK-24`). Two entries that name one row are a finding that names both. |
| `subject` | `act:`, `entity:`, `invoke:`, `dependency:`, `state:`, `event:`, `machine:<id>` (the machine as a whole, covering none of its states, events or invokes) or `system:<name>` (declared under `systems`; an unknown name has no waiver). |
| `required` | `fail-closed`, `retry-bounded`, `surface` or `refuse` (abuse only). Left out, the row's `columns.required` cell gives it (RISK-3 above). |
| `no_subject` | The row names nothing in the system, with the reason. |

A row is linked when its subject resolves and it has a required value. A row with a subject and
no required value is no error: `gen` and `adopt` go on, the row is owed, and stages
`failure-modes` and `xref-failure` count it apart (`name a subject and no required value`). It
is unlinked all the same. A `required` that is given and is none of the four is an error.

## 5. Adopt

Point `adopt` at the area that hurts most. Without `--write` it changes nothing:

```sh
machinery adopt --paths 'internal/shipping/**' --e2e 'e2e/**'
machinery adopt --paths 'internal/shipping/**' --e2e 'e2e/**' --write
echo '.machinery/' >> .gitignore
git add -A
git commit -q -m "adopt machinery for the shipping area"
machinery check
```

The report counts languages, code files, test files (unit tests, those with test doubles),
design sources and the backlog. Reading the backlog runs `bd`; `adopt`, `status` and `check`
skip it with `--no-backlog` or `MACHINERY_NO_BACKLOG=1` and say `backlog: not read`, and a `bd`
that is missing or fails is never an error for them (`verify`, `trace` and `owed` never run it).
`adopt` never edits `.gitignore`: it tells you to add
`.machinery/`, where receipts are kept. `check` passes on day one: the unit tests of the area
are listed in `design/tests-baseline.yaml` and print as warnings, and a new unit test or test
double in the area fails from now on.

Now model the riskiest stateful resource, the shipment, from the code as it is (fixture
`brownfield-design`), and name the ids in the end-to-end tests that check them (fixture
`brownfield-tests`: a shell script that emits receipts and a Go test that emits none):

```sh
cp -R "$MACHINERY_REPO/e2e/testdata/fixtures/brownfield-design/." .
cp -R "$MACHINERY_REPO/e2e/testdata/fixtures/brownfield-tests/." .
printf 'waivers:\n  - {rule: action-event, id: "act:Shipment.create", reason: "creation enters the initial state"}\n' >> machinery.yaml
machinery adopt --write
machinery owed list
machinery check
git add -A
git commit -q -m "model the shipment lifecycle"
```

The first `adopt --write` wrote `machinery.yaml`, and from then on the file is yours: the line
above appends a waiver to it by hand. The second `adopt --write` does not touch it. It puts the
new sources in `design/adoption.yaml`, runs `gen`, and owes every obligation in
`design/owed.yaml` with the first reason that applies:

| Reason | For |
|---|---|
| `planned at adoption` | An id your own mark says is planned (`import.planned`), with its story. A mark that captures more than a story id stops `adopt` before it writes. |
| `claimed at adoption` | An id a test already claims. A claim is not evidence: it stays owed until a receipt proves it. |
| `unproven at adoption` | Every other obligation, and every action no code serves. |

It also waives what the design already had, so the check passes: a failure state or event with
no failure mode yet (`unmodeled at adoption`), and a machine that binds to nothing or an entity
with actions and no lifecycle machine (`unbound at adoption`). A waiver whose cause is fixed is
a `stale waiver` ERROR, so the list only shrinks.

Adopt never writes a `machinery.yaml` that exists, whatever it holds and however it is laid
out. What a later adoption adds to the configuration (a wider scope, the design sources it
finds, baselines, waivers) goes to `design/adoption.yaml`, a file the tool owns and writes
whole. Every command reads the two files as one configuration: lists are joined, and a key
such as `tests.baseline` set to two different values is an error that names both files.
`machinery config` prints the result. You may edit `design/adoption.yaml`, say to drop a waiver
once its cause is fixed, but a comment you leave there is lost the next time adopt adds
something. Commit it with the rest of `design/`.

Besides those two files adopt writes only under `design/` and the gen directory: when you
keep a model elsewhere with a Markdown rendering of your own beside it, adopt leaves that file
alone and says when it is stale (`machinery gen` rewrites it when you run it).

Mark failures where the code records them: tag the state `failure` and list the events in
`_failure_events` (`docs/machine-format.md`, "Failure idioms"), or say it in
`import.machines.annotate`. Bind every machine: a lifecycle machine names its entity, an
operational one lists in `_serves` what it exists for, and an entity whose state lives in two
attributes gets one lifecycle machine per `_aspect`.

## 6. Evidence

A receipt proves only when `machinery run` wrote its file, nothing changed it since, and its
`test` names a test file that claims the id (`docs/formats.md` section 10.2). A file typed by
hand or edited after the run is refused, and a refused file fails `verify`, the story gate and
`owed sync`. That guards against mistakes, not against someone who can run commands as the tool.

**A Go suite, direct.** Tests that name their ids in comments get evidence without a change:

```sh
machinery run --adapter go-test-json -- go test ./e2e/...
```

machinery adds `-json` and `-count=1`, reads the event stream and writes the receipts itself.
The command is held narrow: `go test` with relative package patterns and the usual test flags,
in the module of the project root, for the machine it runs on. `-C`, `-modfile`, `-exec`, a
`-run` that selects subtests and the like go through `--streams`. The header records the go
version, binary, `GOFLAGS`, `GOWORK`, `GOOS`/`GOARCH` and the digests of `go.mod` and `go.work`.

| What the run saw | Receipt |
|---|---|
| Every function the claim belongs to passed, and its package ended with pass | `pass` |
| A function failed | `fail` |
| Its package failed and the function started with no verdict (a timeout, `os.Exit`, a panic) | `fail`; no pass of that package counts |
| Its package ran none of its tests (a build failure, a `TestMain` that exits first) | `fail` for every claimed function |
| A `-run` filter left the function out, or a crash came before it | none |
| A function, or one of its subtests, skipped | `skip` |
| A function did not run, or its pass came from the test cache | none |
| Two Go files of the directory define the same `Test` function, whatever their build tags | none for a pass or a skip; a fail fails both |
| The stream ended before its package did | none: the run is incomplete; delete its receipts file |

A marker directly above a test function claims for that function (`granularity: function`); a
marker above no function, or above a helper, claims for every test function of the file
(`granularity: file`); one above a fuzz test, a benchmark or an example gives no receipt and is
named in the summary (its file is then a claimant the adapter cannot judge, below). A file of
helpers with no test function of its own is judged by its set: it passes when every test
function of its package passed in the same run, and a `-run` filter or a build tag that leaves
one out gives it nothing. A pass
through a parent test records how many subtests ran (`subtests: 2`).

**Every claimant.** An id that several tests claim is proven by the adapter only when every one
of them passed at the commit, in this run or another one. One that did not leaves the id
`partly run: 1 of 2 claimants passed`: not proven, not failed, kept in the owed list. `verify`
names up to three claimants without a pass for each of the first ten such ids, and how many
more there are; `verify --partly` names every one. The claimants are
all the files under `trace.tests` that carry a marker for the id, not only the Go test files: a
shell script, a test in another language, a Go file that is not named `_test.go`, a file that
does not parse. The adapter cannot judge such a file, so it needs a receipt its own test emits
at the commit (or, for a staged Go source, an entry in `evidence.adapter.sources`), and
`verify` says so by name. An id with no counted adapter pass is not under this rule: a receipt
a test emits proves it as before. A test that
cannot run where you make the evidence goes into `evidence.adapter.excused` with a reason:

```yaml
evidence:
  adapter:
    excused:
      - tests: ["e2e/cluster/*_test.go"]
        reason: "needs three nodes; runs in the nightly job"
```

`verify` prints every entry with what it covers, and an id whose claimants are all excused is
not proven (`no claimant can run here`). `claims: any` brings back the rule of one claimant;
the ratchet reports it, and an entry added or widened, as a widening. The same entries may be
kept in `design/excused.yaml` (`version: 1` and a list `excused:`), which the tool reads with
`machinery.yaml`: a change to that file leaves the runs of earlier commits valid, while a change
to `machinery.yaml` voids them.

**A Go suite behind a wrapper: `--streams`.** When a make target or a script starts `go test`
itself, machinery cannot add `-json`. The wrapper writes each stream to a file in the directory
machinery names in `MACHINERY_GO_TEST_JSON`, and `evidence.adapter.sources` names the files each
package is built from. The smallest wrapper:

```sh
cat > e2e/go-suite.sh <<'EOF'
#!/bin/sh
set -eu
go test -count=1 -json ./e2e/... > "${MACHINERY_GO_TEST_JSON:?}/e2e.json"
EOF
cat >> machinery.yaml <<'EOF'
evidence:
  adapter:
    sources:
      - {package: example.test/depot/e2e, files: ["e2e/*_test.go"]}
EOF
git add -A
git commit -q -m "run the Go suite through a wrapper"
machinery run --adapter go-test-json --streams -- sh e2e/go-suite.sh
```

The wrapper passes `-count=1` itself: machinery reads only `GOFLAGS`. A run that reads no stream
exits 1: the wrapper ran no `go test`, or started it under `env -i`, which drops
`MACHINERY_GO_TEST_JSON`. `--allow-no-streams` accepts a wrapper that may run nothing, and
`--keep-streams <dir>` keeps a copy of every stream in `<dir>/<run id>/` so that what the
wrapper ran can be read later. A file may hold several runs
appended one after the other, each judged on its own, and a fail in any of them is a fail; a run
that starts again before the one before it ended (a killed run and its retry, two runs writing at
once) leaves the run incomplete. The entries of `evidence.adapter.sources` are read only in this
mode, and a direct run refuses an entry for a package tested in its own directory: pick one way per
package. One entry `{package: "<module path>/e2e/...", dir: true}` gives every package it
matches the test files of its own directory. Tests staged as copies elsewhere map back with
`files`, which may name sources kept under `trace.tests` without the `_test.go` name. When the
wrapper tests several staged sets, name each one by running its compiled test binary under
`go tool test2json -t -p <name>` or by setting the `Package` field of its `go test -json`
events before writing the stream. Use the import path of the source directory as the name so
one wildcard entry can map each set to its own sources. A pass needs its package and
its entry; a fail goes by function name to every file of the entries that defines it, also from a
test binary run without `-p`, and that name gets no pass in the run. The summary identifies
when one run of a package suppresses passes from another. It reports individual non-test
function marker claims for the packages seen and counts those claims across the whole project.

**A test that emits receipts.** A transition case (`tc`) needs the effects the test observed,
which a verdict of `go test` does not carry. `e2e/shipment_e2e.sh` drives the real binary,
asserts, and only then appends one receipt per id to `MACHINERY_RECEIPTS` (`pkg/receipt` does
that for a Go test):

```sh
machinery run -- sh e2e/shipment_e2e.sh
```

## 7. Sync the owed list, then verify

```sh
machinery owed sync --prune-only
git add -A
git commit -q -m "prune the owed list"
machinery verify
```

`owed sync --prune-only` removes exactly the ids the runs proved and adds nothing. Prune before
you verify: `verify` fails on an owed id that is proven (`stale owed entry`). Pruning changes
only the owed file, so the runs stay valid at the new commit. `owed sync` writes nothing while
`machinery.yaml`, the adoption or excused file, a design source or a test file has uncommitted
changes, and names them: commit them first, or pass `--allow-dirty`, which also uses a run made
on a dirty tree, as `verify --allow-dirty` does.

A run stays valid at a later commit when every path changed since is the tool's own (the owed
list, the adoption and excused files, receipts) or matches a glob of `evidence.exempt` (say
`docs/**`), and its commit is an ancestor of HEAD; `verify` and `status` print
`valid at HEAD through <n> exempt commit(s)` for such a run. A glob that would exempt a test, the
code, a design source, `machinery.yaml` or a build manifest is a config error, and the ratchet
reports every glob added or widened. `verify` counts adapter evidence
apart (`4 by adapter at function granularity`) and prints the go environment of each adapter run.

## 8. Plan: stories beside the ones you have

```sh
machinery plan --apply
machinery waves
```

`plan --apply` adds one story per group of owed ids to the same backlog, with dependency edges
and a tier label; your stories are untouched. An id a test claimed at adoption gets no story of
its own: one `Wire evidence <test file>` story per file (per directory past
`plan.wire_max_files`) makes the tests that exist emit receipts. An id planned at adoption keeps
the story your mark names. `machinery plan --link <story> <id>...` attaches ids to a story you
have. Decisions kept in a file move into the backlog with `machinery decisions import`. An
adopted project may set `review.backlog` and `review.design` to `off` until the checker reviews
its plans; the ratchet reports turning one off later.

## 9. What stays owed, and why

```sh
machinery status
```

`status` counts the owed ids by what stands behind each: claimed by a test with no receipt yet,
named by no test, planned, and actions by whether code serves them.

| Still owed | Why | What ends it |
|---|---|---|
| A `tc` id a Go test claims | The adapter never proves a `tc` or `cx` id: a verdict carries no effects and no replay. | Its `Wire evidence` story: the test emits receipts. |
| An id of a critical or vital area | By default adapter evidence proves routine and consequential ids only (`warn   adapter: <id> not counted`). | A receipt the test emits after its own assertions. |
| An id no test names | Nothing claims it. | Its `Prove` story. |
| A claimed id with no verdict | Its test did not run, was cached, or has a twin (`claimed, not run`, `claimed, verdict not counted`). | Run it, or rename the twin. |
| A partly run id | One of the files that claim it has no pass at the commit (`partly run: 1 of 2 claimants passed`): a test that did not run, or a file the adapter cannot judge. | Run that test at the same commit, have a file the adapter cannot judge emit its own receipt, or excuse it with a reason. |
| An id planned at adoption | Your project said it is not delivered. | The story your mark names. |
| An action | No `machinery:serves` in the code yet. | The `Serve <Entity> actions` story. |

`evidence.adapter` changes what adapter evidence proves: `allow` (kinds), `tiers` and
`min_granularity` (`function` asks for the marker above the function). The ratchet compares the
policy by what it admits: `adapter widened` warns, and is an ERROR for a critical or vital tier.

## 10. Hold the line, then widen

```sh
machinery check --base HEAD~2
machinery adopt --paths 'internal/billing/**' --write
machinery check
```

The ratchet compares with an earlier commit: nothing proven there is owed now, and the adapter
sources added since are named. A second `adopt` widens the scope to the next area, in
`design/adoption.yaml`; its unit tests join the baseline, and from then on the area is held to
the same rules.

## What you did not have to do

Rewrite code, stop the stories in flight, migrate the backlog, model everything, delete tests or
convert a design you keep. The owed list and the baselines show how far along you are.
