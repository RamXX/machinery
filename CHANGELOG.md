# Changelog

## Unreleased: 2.0.0 (experimental, branch `experimental/v2`)

v2 is a new tool. v1 and its history are on `main`. Nothing of v1 that is not listed under
"Carried forward" remains in this tree.

### New

- One contract for every format, output line and exit code: `docs/formats.md`.
- Stage outcomes: passed, failed, not applicable, not demonstrated (which blocks like failed).
  Exit codes 0, 1, 2.
- Obligations with stable ids from the model (`inv`, `sc`), the machines (`tc`: transition,
  ignore, refusal and guard-clause cases), failure modes (`fm`), formal models (`fv`, `cx`) and
  external checkers (`ck`).
- Evidence loop: `run` records receipts bound to a commit, `verify` joins them to the
  obligations, `owed.yaml` is the single ratchet, `proof` and `status` report.
- `probe`: mutation probing of the generated tables as the measure of test quality.
- Stage `tests`: end-to-end only, no test doubles, with a baseline that only shrinks.
- Stage `xstate`: generated tables are compared with what the real XState 5.33.4 executes.
- Risk tiers (`design/risk.yaml`): routine, consequential, critical.
- Formal models by question: TLA+ skeletons generated from machines, Alloy skeletons derived
  from the Modelith model; reach predicates, numbered assumptions, mutations, a baseline,
  a sync marker with drift classification, counterexample traces replayed as obligations.
- Backlog bridge on `bd`: `plan` (stories and edges from what is owed), `waves`, contract freeze,
  `brief` (by file pointer, with model routing), `predict`, `accept`, `story` (tier gates).
- Shift-right testing: `affected`, targeted `run` and `verify`, and `landing` as the one full
  verification with a count of fix-forwards.
- `adopt`: bring an existing project under machinery one area at a time, without touching its
  code; `scope` bounds the static stages.
- `init`, `checks`, external checkers through a neutral projection and evidence file.
- A reference project, `examples/crm`, on a real Postgres.
- Roles for a many-agent run: one prompt each for the planner, the modeler, the builder, the
  verifier, the checker and an experimental delegate, and `docs/roles.md` (who may write what,
  what the tool verifies about each, and the judgment rules with the failure each prevents).
  Default routing by role: design roles on the strongest class of model, builders on a standard
  class, a verifier at least at its builder's.
- The checker's review is recorded (`machinery review backlog|design <id>|intake <id>`) and gates
  dispatch: `waves --freeze` and `intake plan --apply` wait for an approved review from another
  context than the planner (`review` in `machinery.yaml`; the ratchet reports one switched off).
- Stage `scope` (`machinery check --base <ref> --story <id>`, and the commit hook on a branch
  named after a story): a story changes only files of its scope, and design sources only as the
  design story that `intake plan` now creates first.
- An answer on the owner's behalf comes from a `*-delegate` agent that did not ask the question.
- `decisions import`: the records of an existing decision file become backlog decisions, once
  each.
- `adopt --rehearse`: adoption on a snapshot outside the project, with what it writes and owes,
  the check, the plan, and what blocks it; `adopt` names exactly the findings that stop
  generation.
- `import.machines.annotate`: for a dialect that has no key for them, the config says which
  events arrive from outside, which report a failure, which states record one (names and `*`
  patterns) and which event is a model action under another name. A pattern that matches nothing
  is an error, and a pattern of `*` only is refused. The native annotation `_aliases` carries
  the last one, and rules `event-action` and `action-event` read an aliased event as its action.
  An alias hides nothing: an event that is an action takes none, an action with an event of its
  own name is the action of none, and stage `xref-model-machine` prints and counts every alias.
- Rule `event-action` reports an event listed in `_external_events` or `_failure_events` that is
  an action of the model: an action is a caller's command. For an operational machine every
  action of the entities behind what it serves counts, for a listed and for an aliased event.
- The name hint about a failure-looking name is dismissed by name, with a reason, in the root
  `_not_failures` (`not_failures` in `import.machines.annotate`). Listing an event as external
  or giving it an alias does not stop it, and a dismissal that dismisses no hint is an error.
- Claims beside owed: `adopt` owes an id a test already claims as `claimed at adoption` and an
  id the project marked planned (`import.planned`) as `planned at adoption` with its story;
  `status`, `assess` and the rehearsal count ids by claim; `plan` gives the claimed ones one
  `Wire evidence` story per test file (per directory past `plan.wire_max_files`).
- `trace.markers.scenarios_exact`: a marker that claims only when its text is a scenario name.
  Serves tags that name ids which are not actions are one counted warn of stage `trace` for the
  whole run; `machinery trace --non-action-serves` lists them.
- Failure modes: a subject `machine:<id>` for a machine as a whole; `columns.required` in
  `import.failure_modes` for a catalog that has the column; and stage `xref-failure` counts on
  its own the failure modes that name a subject and no required value (still unlinked).
- A links entry that names a subject and no required value is no error: `gen` and `adopt` go on
  and the row is owed; a required value that is given and invalid stays an error. Stages
  `failure-modes` and `xref-failure` print one account of linked, unlinked, subject-only and
  subject-less failure modes, and a subject that names nothing is a `subject-resolves` finding
  whatever its required value.
- `import.planned` reads scenario descriptions (`in: scenario-description`).
- `_refusal_events` (`refusal_events` in `import.machines.annotate`) declares the event by which
  a machine writes that it refused a caller's command, with the action and a reason; rule
  `event-action` accepts it unless the event moves the machine into a failure state it was not in
  already (a transition that stays in its failure state is accepted; one from a state that is
  not tagged failure, or from one failure state into another, is not), the stage prints it with
  its reason and counts it, and it is neither hinted nor asked for a failure mode.
- An aliased event may be listed in `_failure_events` when the action it stands for is itself
  the failing outcome.
- A rehearsal names the project's HEAD in every reason and waivable line it writes, never the
  commit of its snapshot.
- An owed file that does not parse is never read as empty: every command refuses with the
  parse error, and machinery writes the file through a YAML encoder.
- Stage `xref-failure` prints how many failure modes are linked and how many are not, and is
  not demonstrated while one names no subject: one linked row among many no longer passes. A
  failure mode says it has none with `no_subject: "<reason>"`, in the failure-mode file or the
  links file.
- Failure-mode subject `system:<name>` for the system as a whole or a named part of it, with the
  names declared under `systems` in `machinery.yaml`; no waiver covers an unknown name.
- Links entries that cover many rows: `ids` takes ids and inclusive ranges
  (`FM-01..FM-23`); two entries that name the same row are a finding that names both.
- Evidence cannot be typed by hand: a receipts file counts only when `machinery run` entered it
  in the run ledger of the git directory and nothing changed it since, and a `pass` receipt
  proves only when its `test` names a test file that claims the id (`receipt.Test` in
  `pkg/receipt` builds it); a `fail` in a run that is used always fails. A refused receipts
  file (typed, changed, or deleted after its run) or an incomplete run fails the story gate,
  replaces the counts of `proof` and stops `owed sync`. A receipt needs a `test`. This guards
  against mistakes and casual edits, not against a party who can run commands as the tool
  (`docs/formats.md` section 10.2).
- Run adapter `go-test-json`: `machinery run --adapter go-test-json -- go test ...` reads the
  event stream of an existing Go suite and writes the receipts itself, joining each verdict to
  the ids the test files already claim (the function under a marker, or every test function of
  the file). Cached packages are refused, `tc` and `cx` ids are never proven this way, and
  `status`, `verify` and `proof` count adapter evidence apart, by granularity. The policy
  `evidence.adapter` (kinds, tiers, minimum granularity, a map for test packages built from
  copies) decides what it proves; by default never an id of a critical or vital area.
  A verdict goes to the file under `trace.tests` in its package's directory that defines the
  function, and to none when another Go file of that directory defines a function of the same
  name that begins with `Test` and takes one parameter, whatever the build constraints or the
  way the parameter is written: a twin behind another tag never lends its pass. The direct form
  accepts only a narrow `go test` (relative patterns, a list of test flags, the project's own
  module, `GOOS` and `GOARCH` of the host), and refuses a module path that two `go.mod` files
  declare and a workspace that leaves out the module of a claiming file, which stage `trace`
  reports as well; everything else runs through `--streams`. The header of a direct run
  records its go environment (version, binary, `GOFLAGS`, `GOWORK`, `GOOS`/`GOARCH`, the
  digests of `go.mod` and `go.work`), and `verify` and `proof` print it. Adapter evidence is
  as trustworthy as the project's test code and the command: the rules catch mistakes, not
  planted code or a controlled environment. With
  `--streams` (a wrapper that starts `go test` itself and leaves the streams in
  `MACHINERY_GO_TEST_JSON`) the files come from `evidence.adapter.sources`, which is read in that
  mode only. A pass replayed from the cache counts nothing. A package that failed (a timeout,
  `os.Exit`, a panic, a build failure) writes a fail for every claimed function without a
  verdict and no pass; a stream without a package end leaves the run incomplete, and the error
  names the receipts file. A stream file that holds several runs of one package is judged run by
  run, so a fail, a crash or a missing verdict in any of them is a fail. A marker directly above a
  fuzz test, a benchmark or any other function that is no test counts nothing and is named in the
  summary. A parent
  with a skipped subtest is a skip, and direct mode refuses a `-run` or `-skip` that selects
  subtests. Stage `trace` reports the module layout rules only for a project that configures
  `evidence.adapter`, and they read only tracked `go.mod` and `go.work` files outside
  `testdata` and `vendor`, with `GOWORK=off` honoured: a project without that block sees
  no change. A
  receipt a test appends with the adapter's fields is refused. The ratchet compares the policy
  by what it admits (`adapter-widened`), an error for a critical or vital tier.

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
