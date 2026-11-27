# Changelog

## Unreleased: 2.0.0 (experimental, branch `experimental/v2`)

v2 is a new tool. v1 and its history are on `main`. Nothing of v1 that is not listed under
"Carried forward" remains in this tree. `docs/formats.md` is the contract for every format,
command, output line and exit code.

### Design

- Design sources as data: a Modelith 0.6.0 model, machines in the XState v5 JSON subset, OpenAPI
  or proto contracts, failure modes and a risk file. `machinery lint` checks each and
  cross-checks them (`xref-model-machine`, `xref-contract`, `xref-failure`).
- Obligations with stable ids from the model (`inv`, `sc`), the machines (`tc`: transition,
  ignore, refusal and guard-clause cases), failure modes (`fm`), formal models (`fv`, `cx`),
  external checkers (`ck`) and vital properties (`pf`). `machinery gen` writes them with the
  runtime tables, and stage `gen-drift` keeps them current.
- Stage `xstate`: the generated tables are compared with what the real XState 5.33.4 executes.
- Machines bind to the model: a lifecycle machine per entity (one per `_aspect` when its state
  lives in several attributes), or an operational machine that lists `_serves`. Also
  `_entries`, `_terminal`, `_requires`, `_delays`, `_guards` with clause cases, and
  fail-closed tags.
- Failures are first class: states tagged `failure` and `_failure_events` ask for a failure
  mode, as every invoke, dependency and tiered action does. A failure-looking name that is not
  marked is hinted, and `_not_failures` dismisses a hint with a reason.
- `_aliases` (an event that is a model action under another name) and `_refusal_events` (an
  event by which a machine writes that it refused an action, with a reason). Stage
  `xref-model-machine` prints and counts both. An alias hides nothing, an action is never
  external, and an event that moves the machine into a failure state it was not in is no
  refusal.
- Failure-mode subjects `machine:<id>` (a machine as a whole) and `system:<name>` (declared
  under `systems`), and `no_subject: "<reason>"` for a failure mode about nothing modeled.
- Statuses in `design/status.yaml`: a proposed id is a candidate and yields no obligation; a
  retired id is no longer promised.
- Risk tiers in `design/risk.yaml`: routine, consequential, critical and vital. A vital area
  states exact properties over its ids (proof obligations, awaiting proof), a core held to a
  shape a later proof can use (stage `arch`), and assumptions with their evidence.
  `machinery vital candidates` lists the ids that look vital.
- Formal models by question: TLA+ skeletons generated from machines, Alloy skeletons from the
  model; reach predicates, numbered assumptions, mutations, a baseline, a sync marker with
  drift classification, and counterexample traces replayed as `cx` obligations.

### Evidence

- `machinery run` records receipts bound to a commit, `verify` joins them to the obligations,
  `owed.yaml` is the single ratchet, and `proof` and `status` report. The receipts header
  records the machinery build's version and commit.
- A receipts file counts only when `machinery run` entered it in the run ledger of the git
  directory and nothing changed it since, and a `pass` proves only when its `test` names a
  test file that claims the id (`receipt.Test` in `pkg/receipt` builds it). A refused receipts
  file (typed, changed or deleted after its run) or an incomplete run fails `verify` and the
  story gate, replaces the counts of `proof` and stops `owed sync`. This guards against
  mistakes and casual edits, not against a party who can run commands as the tool.
- Run adapter `go-test-json`: `machinery run --adapter go-test-json -- go test ...` reads the
  event stream of an existing Go suite and writes the receipts itself, for the ids the test
  files already claim, by function or by file. Direct mode accepts a narrow `go test` in the
  project's own module and records the go environment in the header; `--streams` reads the
  streams a wrapper leaves in `MACHINERY_GO_TEST_JSON`, mapped by `evidence.adapter.sources`.
  An entry of the sources names its files (`files`, which may be staged test sources under
  `trace.tests` that are not `*_test.go` files; a wildcard entry gives each package the files of
  its own directory when it names some) or gives each package of an import path pattern the test
  files of its own directory (`dir: true`). A wrapper names each staged set with
  `go tool test2json -t -p <name>` or by rewriting the `Package` field of its `go test -json`
  stream. The summary limits non-test function marker diagnostics to packages seen in the run
  and counts those markers on the whole-project line.
- A verdict goes to the claiming file in its package's directory, never through a test of the
  same name in another file. No fail is lost: in a package that failed, a function with a run
  event and no verdict (a timeout, `os.Exit`, a panic) fails, a function a filter left out or a
  crash never reached gets nothing, and no pass counts; a package that ran none of its tests (a
  build failure, a `TestMain` that exits first) fails every claimed function. With `--streams`
  a fail needs no package and no exact entry: it goes by function name to every file of the
  sources that defines it, also from a test binary run under `test2json` without `-p` (there a
  function with more `run` events than verdicts in its stream fails too), and that name gets no
  pass in the run; a pass needs its package and its entry. The summary reports how many
  passing functions lost credit because another run of their package failed. A stream cut
  short or holding two interleaved runs leaves the run incomplete. A cached pass counts
  nothing, and `tc` and `cx` ids are never proven this way. The rules aim at mistakes, not at
  planted code or a controlled environment.
- Policy `evidence.adapter` (kinds, tiers, minimum granularity): by default adapter evidence
  never proves an id of a critical or vital area. `status`, `verify` and `proof` count it
  apart, and the ratchet reports a policy that admits more (`adapter widened`, an ERROR for a
  critical or vital tier).
- A symbolic link under `trace.tests` or `trace.code` (a linked file or a linked directory) is not
  read, while a test runner may build what it points to: every such link is an ERROR of stage
  `trace` (`trace: <path> is a symbolic link; markers behind it are not read`), so a failing test
  behind a linked directory never leaves the project passing `check`. Links are found at any
  depth, a link inside a linked directory included (`e2e/linked/inner`), and a cycle of links
  ends the walk. Nothing under `node_modules`, `.git` and the other directories the scan never
  walks is a finding, nor is a link with one of those names. A waiver of rule `trace` is refused
  with what to do instead, and `trace --non-action-serves` names on stderr each link it did not
  read.
- Every claimant must pass (`evidence.adapter.claims: all`, the default). The claimants of an
  id are every file under `trace.tests` that carries a marker for it, as stage `trace` reads
  markers, whatever its name or language: the test function a marker stands above in a Go test
  file the adapter reads, and otherwise the file. Once a counted adapter pass is used for an
  id, it is proven only when each claimant is satisfied at the commit, by a verdict of the
  adapter or by a pass receipt its own test emitted. Otherwise it is
  `partly run: <n> of <m> claimants passed` in `verify`, `status`, `proof` and the run
  summary: not proven, not failed, not pruned, and the story gate fails it. `verify` names up
  to three claimants without a pass for each of the first ten such ids, then how many more
  there are, `verify --partly` every one, and says what to do; a file the adapter cannot judge
  (not Go, not named `_test.go`, not a mapped staged source, not parsing, without a test
  function and judged by no set) is named with why and with what it needs: fixing the file
  when it does not parse, and removing the marker when it is not Go and no test (a README, a
  golden file). A claiming file with no test function of its own that is a candidate of a
  package of the run (a helper) is judged by its set: it passes only when every test function
  of that package passed in one run, a fail of the set fails it, and `verify` says `by its
  set`. A marker above a helper, a method or `TestMain` is a marker of its file. A file under
  `trace.tests` that cannot be read fails the commands that read markers. An id with no
  counted adapter pass is judged as before: a receipt a test emits proves it. `claims: any` is
  the rule of one claimant, and the ratchet reports it.
- `evidence.adapter.excused` leaves claimants that cannot run here out of that rule, each entry
  with globs of test files, optional function patterns and a one-line reason; a file-level
  claimant is excused by its file, never by a function. `verify` prints every entry with the
  claimants it covers and how many globs it has (the globs under `verify --partly`), and an
  ERROR, with why, for each glob or pattern through which it covers nothing; an id whose
  claimants are all excused is not proven; the ratchet names an entry added or widened, an
  ERROR when it covers a claimant of a critical or vital id. The entries may be kept in
  `design/excused.yaml`, read with `machinery.yaml`, where a change keeps earlier runs valid.
- Evidence across commits: a run of an earlier commit is valid at HEAD when its commit is an
  ancestor of HEAD and every path changed since (renames by both paths) is the tool's own (the
  owed list, the adoption and excused files, receipts, property records) or matches a glob of
  `evidence.exempt`; `verify` and `status` say through how many exempt commits. A glob that
  would exempt a test, the code, a design source, `machinery.yaml` or a build manifest is a
  config error, and the ratchet reports every exempt glob added or widened.
- `owed sync` writes nothing while `machinery.yaml`, the adoption or excused file, a design
  source or a test file has uncommitted changes, and names them; `--allow-dirty` syncs as
  `verify --allow-dirty` judges.
- A `--streams` run that reads no stream exits 1 (the wrapper ran no `go test`, or cleared the
  environment with `env -i`); `--allow-no-streams` accepts a wrapper that may run nothing.
  `--keep-streams <dir>` copies every stream read into `<dir>/<run id>/` and records it in the
  header. The verdicts that find no file are split: packages no file under `trace.tests`
  belongs to, each with its count and none left out, and each function of a mapped package
  that no candidate file defines, by name. The whole-project line of a run that wrote no
  receipt counts only the ids without evidence at the commit. An adapter pass through a parent
  test records how many subtests ran, and `verify --partly` hints at a test whose count
  differs between runs at the commit.
- Stage `tests`: end-to-end only, no test doubles, with a baseline that only shrinks.
- `probe`: mutation probing of the generated tables as the measure of test quality.
- Shift-right testing: `affected`, targeted `run` and `verify`, and `landing` as the one full
  verification, with a count of fix-forwards and a red integration branch until it is repaired.
- An owed file that does not parse, or has a problem of its own, is never read as empty: every
  command refuses with the error, and machinery writes the file through a YAML encoder.

### Brownfield

- `machinery assess`: a read-only report on a snapshot of `HEAD`; nothing is written in the
  project or its `.git`.
- `machinery adopt`: one area at a time, without touching code; `scope` bounds the static
  stages and only widens. Existing unit tests are baselined, and existing failures without a
  failure mode and unbound machines are waived; a waiver goes stale when its cause is fixed.
- `machinery adopt --rehearse`: adoption on a snapshot, with what it writes and owes, the check,
  the plan and what blocks it. Every reason and waivable line names the project's HEAD.
- `adopt --write` writes only a `machinery.yaml` where there is none, files under `design/` and
  the gen directory, the owed list and the baselines: the rendering beside a model kept elsewhere
  is the project's file, and adopt says `<file> is stale; the project regenerates it` (or
  `is missing`) and leaves it.
- `adopt` never writes a `machinery.yaml` that exists, whatever it holds. What adoption adds to
  the configuration of such a project (scope and trace globs, design sources, baselines,
  `backlog.tool`, waivers) goes to `design/adoption.yaml`, which the tool owns and writes whole
  through a YAML encoder. Every command reads the two files merged: lists are joined, and a
  scalar set in both with two values is a config error that names both files. The ratchet reads
  both at the base, `assess` and the rehearsal take both, and `machinery config` prints the
  merged configuration. A project adopted by an earlier build keeps what is in its
  `machinery.yaml`.
- `import`: machines in a dialect of the project's own (`states`, `rename`, `terminal`,
  `register`, `bind`), and `import.machines.annotate` for what the dialect has no key for:
  `external_events`, `failure_events`, `failure_states`, `aliases`, `refusal_events` and
  `not_failures`, by name or `*` pattern. A pattern that matches nothing is an error, and a
  pattern of `*` only is refused.
- Failure catalogs kept as Markdown tables, with a links file: `ids` takes ranges
  (`FM-01..FM-23`), two entries that name one row are a finding, and `columns.required` reads
  the required value from the catalog. A row is linked when its subject resolves and it has a
  required value. A row with a subject and no required value is counted apart and owed, and
  stage `xref-failure` is not demonstrated while a row is unlinked.
- `trace.markers`: the project's own markers, with `scenarios_exact` for a marker that also
  carries other text. Serves tags that name ids which are not actions are one counted warn;
  `machinery trace --non-action-serves` lists them.
- Claims beside owed: `adopt` owes an id a test already claims as `claimed at adoption` and an
  id the project marked planned (`import.planned`) as `planned at adoption`, with its story.
  `status`, `assess` and the rehearsal count ids by claim, and `plan` gives the claimed ones
  one `Wire evidence` story per test file (per directory past `plan.wire_max_files`).
- `machinery decisions import`: the records of an existing decision file become backlog
  decisions, once each.

### Method

- `machinery --version` and `machinery version` report the version, source revision and
  build date.
- Stage outcomes: passed, failed, not applicable, not demonstrated (which blocks like failed).
  Exit codes 0, 1, 2. `machinery checks` names what each stage protects.
- Backlog bridge on `bd`: `plan` (stories and edges from what is owed), `waves`, the contract
  freeze (stage `contract`), `brief` by file pointer with model routing, and tier gates
  (`predict`, `accept`, `story`): from consequential up, a fresh verifier predicts before
  reading the proof.
- Markdown `machinery plan` output gains story count lines. `plan --apply` gains counts by
  kind and prints its wire totals on the shared `wire evidence:` line, including the grouping
  by test file or directory. `plan --summary` prints just the count lines.
- `adopt`, `status`, `verify`, `check`, `trace` and `owed` never need the backlog tool: the
  last three never run it, and `adopt`, `status` and `check` skip it with `--no-backlog` or
  `MACHINERY_NO_BACKLOG=1` and print `backlog: not read`; `--story` is refused under the
  switch.
- Intake (`machinery intake new`, `intake status`, `intake plan`): a requirement changes the
  design first; `intake plan --apply` then writes every story at once, the design story first
  and a capstone last.
- Stage `scope` (`machinery check --base <ref> --story <id>`): a story changes only files of its
  scope, and design sources only as the design story of an intake.
- Roles for a many-agent run: planner, modeler, builder, verifier, checker and an experimental
  delegate, described in `docs/roles.md`. Design roles run on the strongest class of model,
  builders on a standard class, a verifier at least at its builder's.
- The checker's reviews are recorded (`machinery review backlog|design <id>|intake <id>`) and
  gate dispatch; the ratchet reports a review switched off.
- Decisions (`decide`, `decisions`), observations and learnings of stories, and questions for
  the owner (`ask`, `questions`, `answer`), answered on the owner's behalf only by a
  `*-delegate` agent under a clause of the owner's policy (experimental).
- `init`, `config`, and external checkers through a neutral projection and an evidence file.
- A reference project, `examples/crm`, on a real Postgres.

### Harness

- A Claude Code plugin: one skill, six agent prompts and three hooks, each running
  `machinery hook <event>`: the status when a session starts, the check before `git commit` and
  `git push`, and the check after a turn that changed files. `hooks` in `machinery.yaml` sets
  each to block, warn or off.
- The session-start hook exports `MACHINERY_SESSION`. Proofs, predictions and verdicts record
  it with `MACHINERY_AGENT`, and the story gate refuses a verifier from the builder's session
  and agent.
- A Codex plugin manifest with the skill; it wires no hooks.

### Carried forward from v1

- Modelith integration (now 0.6.0), through the real executable.
- The XState v5 JSON subset and its lint, and the stable transition ids.
- The pinned tla2tools v1.7.4 and Alloy v6.2.0, fetched by checksum.
- External checkers as an extension point.

### Removed

- Attestations and re-attestation, acceptance files, checkpoint gates.
- The threat ledger as a separate artifact: an abuse case is now a failure mode.
- The built-in rules engine and its baselines.
- READS, WRITES, USES and similar matrix declarations; matrix and oracle markdown files.
- The custody, integrity and isolation layers, and v1's hooks.
- Locked-test amendment rules, execution packets, slice maps, hard TDD phases.
- Mandatory C4 architecture documents and the four-phase handoff with its agents.
- Unit tests, in this repository and as something machinery asks of a project.
