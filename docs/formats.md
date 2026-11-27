# machinery v2 formats and contracts

This document is the contract between every part of machinery v2. Code, tests and other
documents cite it by section number. A change here is a design change: update this file first,
then the code, then the tests.

Status: experimental. v1 lives on the `main` branch and is unaffected.

Pinned upstream semantics: Modelith 0.6.0 (model format `v1`) and XState 5.33.4 (the v5
JSON-serializable subset in `docs/machine-format.md`). Formal tools: tla2tools v1.7.4 and
Alloy v6.2.0, both fetched by checksum (section 16.8).

## 1. What the tool does

A project keeps hand-written design sources: a domain model (Modelith YAML), state machines (an
XState v5 JSON subset), contracts (API definitions), failure modes and a risk file. machinery
reads them and:

1. lints them and cross-checks them against each other (`machinery lint`);
2. generates what can be derived: runtime transition tables, the list of obligations, and the
   skeletons of formal models (`machinery gen`, `machinery formal gen`);
3. checks the code statically against the design (`machinery check`);
4. joins the design to evidence from real end-to-end test runs (`machinery run`,
   `machinery verify`), with one ratchet file for everything not yet proven (`owed.yaml`);
5. turns what is owed into backlog stories with a dependency graph and schedules them in
   parallel waves (`machinery plan`, `machinery waves`), writes the stories of a new requirement
   only from the design it changed (`machinery intake`), and gates each story by its risk tier
   (`machinery story`).

Three rules shape everything below:

- **Formality follows risk.** Each area of the design has a tier chosen at planning (section 23):
  routine (machines and generated end-to-end cases), consequential (plus a fresh verifier who
  predicts before reading the proof), critical (plus a formal model), vital (plus the exact
  properties its code must keep, a core shaped so that a later version can prove them, and the
  assumptions that core makes, each with its evidence).
- **Evidence comes from the real system.** There are no unit tests and no test doubles, in
  machinery or in a project that uses it (section 9.4). Test quality is measured by probing
  (section 20), not by counting tests.
- **A blocking check protects a named product behavior.** Every finding names a design id or a
  file that carries one. No check exists only to confirm that a sequence of steps was followed
  (section 3.4).

## 2. Project layout and `machinery.yaml`

A project is the directory that contains `machinery.yaml`. Every command finds it by walking up
from the current directory. The global flag `--project <dir>` (short `-C`) names it explicitly.
All paths in the file are relative to the project root, use forward slashes, and must stay inside
the root (`..` segments and absolute paths are a config error).

```yaml
version: 2                      # required, must be 2
scope: ["internal/**"]          # optional: the code paths this project covers; default ["**"]
design:
  models:                       # optional list of Modelith files (a new adoption may have none yet)
    - design/model/shop.modelith.yaml
  machines: design/machines     # optional: directory holding *.machine.json
  failure_modes: design/failure-modes.yaml   # optional
  risk: design/risk.yaml        # optional: risk tiers (section 23)
  formal: design/formal         # optional, this is the default; used when the directory exists
  facts: design/facts.yaml      # optional: facts the design declares about itself (section 22.2)
  owed: design/owed.yaml        # optional, this is the default
  status: design/status.yaml    # optional, this is the default; used when present (section 4.1)
  gen: design/gen               # optional, this is the default
import:                         # optional: design sources kept in the project's own formats
  machines:                     # optional, instead of design.machines (section 5.2.1)
    files: ["machines/*.json"]  # required: globs of the project's machine files
    states: pascal              # optional: capitalize | pascal | keep; default keep (section 5.2.1)
    rename:                     # optional: the project's key -> the machinery key, per level
      root: {notes: _comment, startable: _entries}
      state: {skip: _ignores}
      transition: {allowed: _requires, denied: _refusal}
    terminal:                   # optional: a root key whose reason marks the states never left (section 5.2.1)
      from_root: forever
    register:                   # optional: a register of the transitions the code makes (section 5.2.1)
      file: design/transitions.yaml
      machine: flow             # the key of an entry that names its machine
      observed: seen            # the key of an entry that lists {from, event, to}
    bind:                       # optional: what a machine binds to in the model (section 5.2.1)
      "machines/loan*.json": {lifecycle_of: Loan}       # a glob of source files, or
      "machines/loan-fines.json": {lifecycle_of: Loan, aspect: fines}   # one aspect of a lifecycle
      hold_queue: {serves: ["act:Loan.lend"]}            # a machine id
    annotate:                   # optional: what the dialect has no key for (section 5.2.1)
      "machines/*.json":        # a glob of source files, or a machine id; every matching entry applies
        external_events: ["due_*"]         # events that arrive from outside: exact names, or patterns with *
        failure_events: ["*_failed"]       # events that report a failure
        failure_states: ["*_failed"]       # states that record one
        aliases: {handed_back: giveBack}   # an event that is a model action under another name
        refusal_events:                    # an event by which the machine writes that it refused an action, with the reason
          lend_refused: {action: lend, reason: "a blocked member is refused a loan; nothing fails"}
        not_failures: {count_aborted: "a stocktake that is called off changes no loan"}   # a name hint dismissed, with the reason
  failure_modes:                # optional: a failure-mode catalog kept as Markdown tables (section 5.4.1)
    file: docs/risks.md         # required
    id: "^RISK-[0-9]+$"         # required: the first cell of a failure-mode row matches it in full
    columns:                    # required: header names, case-insensitive, first match wins
      failure: ["Failure", "What goes wrong"]
      behavior: ["Required behavior", "Expected"]
      required: ["Required", "Class"]   # optional: the column that holds the required value, for a catalog that has one
    links: design/risk-links.yaml   # optional: the subject and required value of each row
  planned:                      # optional: the project's own "planned" marks, read by adopt (section 17.2)
    - {in: model-statement, pattern: '^Planned \(story (?P<story>[a-z0-9-]+)\)\.'}   # an invariant statement
    - {in: scenario-description, pattern: '^Planned \(story (?P<story>[a-z0-9-]+)\)\.'}   # a scenario description
    - {in: failure-row, pattern: '^planned: (?P<story>[a-z0-9-]+)$'}                  # a cell of a catalog row
contracts:                      # optional
  - kind: openapi               # openapi | proto
    path: contracts/openapi.yaml
checkers:                       # optional: external checkers (section 22)
  - id: rules
    covers: ["act:*", "inv:*"]
dependencies:                   # optional: external dependencies of the system
  - name: postgres
    posture: real               # real | sandbox
systems:                        # optional: the system as a whole and its named parts, for failure-mode subjects (section 5.4)
  - name: build                 # required: ^[a-z][a-z0-9-]*$, unique
    about: "how the shop is built and released"   # optional
trace:                          # optional
  code: ["internal/**/*.go"]    # files that may carry machinery:serves tags; default: scope
  tests: ["e2e/**"]             # files that may carry machinery:verifies tags
  markers:                      # optional: the project's own markers, literal prefixes (section 8.1)
    serves: ["Implements:"]
    verifies: ["Checks:", "Proves:"]
    scenarios: ["Scenario:"]
    scenarios_exact: ["Objective:"]   # claim only when the text is a scenario name, silent otherwise
tests:                          # optional: the end-to-end-only policy (section 9.4)
  e2e: ["e2e/**"]               # where tests live; default: trace.tests
  files: []                     # optional: replaces the default test-file patterns
  doubles: []                   # optional: replaces the default test-double markers
  baseline: design/tests-baseline.yaml   # optional
constraints:                    # optional
  layers:                       # ordered from lowest to highest
    - {name: domain, paths: ["internal/domain/**"]}
    - {name: api, paths: ["internal/api/**"]}
  size:                         # optional; a size limit is a design constraint only when set
    paths: ["internal/**/*.go"] # optional, defaults to trace.code
    max_file_lines: 500
    max_func_lines: 80
  cores:                        # optional: the shape of the core of every vital area (section 9)
    deny_imports: ["net/http", "database/sql", "os", "time", "math/rand"]   # optional
    max_file_lines: 300         # required
    max_func_lines: 60          # required
  baseline: design/arch-baseline.yaml   # optional
receipts:
  dir: .machinery/receipts      # optional, this is the default
evidence:                       # optional
  adapter:                      # optional: what a pass written by a run adapter proves (section 10.3)
    allow: [inv, sc, fm, ck]    # optional, this is the default: the kinds an adapter pass may prove
    tiers: [routine, consequential]   # optional, this is the default: the tiers it may prove an id of
    min_granularity: file       # optional, this is the default: function | file, or a mapping (below)
    sources:                    # optional, read only with --streams: the files each test package of the run is built from
      - package: "example.test/shop/staged/**"   # required: a glob of package import paths
        files: ["e2e/suite/*_test.go"]           # required: globs of the test files the package is built from
commands:
  e2e: ["go", "test", "./e2e/...", "-count=1"]   # optional: the project's end-to-end command
backlog:                        # optional: the backlog bridge (section 15)
  tool: bd                      # only bd
  base: main                    # optional: the branch work is cut from; default main
plan:                           # optional: machinery plan (section 15.2)
  wire_max_files: 20            # optional, this is the default: past this many test files, wire-evidence stories are one per directory
hooks:                          # optional: what machinery hook does (section 21.1); block | warn | off
  commit: block                 # optional, this is the default
  push: block                   # optional, this is the default
  stop: warn                    # optional, this is the default
review:                         # optional: the checker's reviews that gate dispatch (section 15.10); required | off
  backlog: required             # default required when backlog.tool is set: waves --freeze waits for an approved review
  design: required              # default required when backlog.tool is set: intake plan --apply waits for one
delegate:                       # optional, experimental: answers on the owner's behalf (section 15.9)
  policy: ../owner/delegate.yaml   # required: the owner's policy file; may lie outside the project
routing:                        # optional: model routing by tier, printed in briefs and waves (section 21); these are the defaults
  routine: "standard model"
  consequential: "standard model; fresh verifier of at least the builder's class"
  critical: "strongest model for the formal model; standard model for code; fresh verifier of at least the builder's class"
  vital: "strongest model for the formal model and the core; standard model elsewhere; fresh verifier of at least the builder's class"
vital:                          # optional: the word lists of machinery vital candidates (section 23.3)
  names: [pay, charge]          # optional: replaces the default list for action, entity and event names
  failure: [lost, twice]        # optional: replaces the default list for failure-mode text
  effects: [charge, emit]       # optional: replaces the default list for transition effects
tools:
  modelith: modelith            # optional, executable name or path; this is the default
  java: java                    # optional
  node: node                    # optional
  bd: bd                        # optional
  dir: ""                       # optional: tools directory (section 16.8)
  xstate: false                 # optional: true enables stage xstate (section 6.8)
waivers:                        # optional, see section 6.7
  - {rule: action-event, id: "act:Order.place", reason: "creation enters the initial state"}
assess:                         # optional: read only by machinery assess (section 18)
  decisions:                    # optional: the project's own decision records (section 18.7)
    files: ["docs/adr/*.md"]   # required: globs of the files that hold them
    record: "^# ADR-[0-9]+: "  # required: a regular expression matching the heading line of one record
```

Rules:

- Unknown keys anywhere are a config error. A config error prints `config: <message>` on stderr
  and exits 2.
- `import` reads design sources that a project keeps in its own formats, so an existing project
  can be assessed and adopted before anything is converted. machinery never writes those sources.
  `design.machines` and `import.machines` together are a config error
  (`import.machines: design.machines is also set; read the machines from one of them`), and so
  are an empty `import.machines.files` (`import.machines.files is required`), a `states` value
  other than `capitalize`, `pascal` or `keep`
  (`import.machines.states must be capitalize, pascal or keep, got "<value>"`), an empty key
  or name in `rename`, and an `import.machines.terminal` without `from_root`
  (`import.machines.terminal.from_root is required`) or whose key `rename.root` also maps
  (`import.machines.terminal.from_root: <key> is also renamed by import.machines.rename.root; keep one`),
  and an `import.machines.register` without `file`, `machine` or `observed`
  (`import.machines.register.file is required`).
  `import.machines.bind` maps a glob of source files or a machine id to exactly one of
  `lifecycle_of` (an entity name, with an optional `aspect`, section 5.2) and `serves` (a
  non-empty list of model ids): an `aspect` without `lifecycle_of`
  (`import.machines.bind.<key>: aspect needs lifecycle_of; an aspect is part of an entity's lifecycle`), an empty key or
  one that is not a valid glob (`import.machines.bind: "<key>" is not a valid glob`), an entry
  with both or neither (`import.machines.bind.<key>: set lifecycle_of or serves, not both`,
  `import.machines.bind.<key>: set lifecycle_of or serves`) and an empty id in `serves`
  (`import.machines.bind.<key>.serves: an id must not be empty`) are config errors.
  `import.machines.annotate` maps a glob of source files or a machine id to at least one of
  `external_events`, `failure_events`, `failure_states` (lists of names and patterns),
  `aliases` (event name to action name), `refusal_events` (event name to `{action, reason}`: the action it refuses and
  the reason the event is a refusal and no failure) and `not_failures` (event or state name to a
  reason).
  An empty key or one that is not a valid glob
  (`import.machines.annotate: "<key>" is not a valid glob`), a key that is no path inside the
  project (`import.machines.annotate: "<key>" must stay inside the project root`), an entry with
  none of the six
  (`import.machines.annotate.<key>: set external_events, failure_events, failure_states, aliases, refusal_events or not_failures`),
  an empty pattern (`import.machines.annotate.<key>.<list>: a pattern must not be empty`), a
  pattern made of `*` only, which would name everything
  (`import.machines.annotate.<key>.<list>: pattern "<pattern>" has no literal part; a pattern names the events or states it is for`),
  an empty event or action in `aliases`
  (`import.machines.annotate.<key>.aliases: an event and its action must not be empty`), the
  same in `refusal_events`
  (`import.machines.annotate.<key>.refusal_events: an event and its action must not be empty`), a
  refusal without a reason
  (`import.machines.annotate.<key>.refusal_events.<event>: reason is required: say why the event is a refusal and no failure`),
  a refusal entry that is no `{action, reason}`, such as a bare action name
  (`machinery.yaml: line <n>: a refusal_events entry must be {action: <action>, reason: "<why the event is a refusal and no failure>"}, got <value>`),
  a name, an action or a reason of `aliases`, `refusal_events` or `not_failures` that is not one
  line, since the tool prints them and a line break would put text of the author's choice at
  the start of an output line
  (`import.machines.annotate.<key>.<list>: "<text>" has a line break or a control character; a name, an action and a reason are one line`,
  the text quoted with its escapes), and an
  empty name or reason in `not_failures`
  (`import.machines.annotate.<key>.not_failures: a name and its reason must not be empty`) are
  config errors.
  `import.failure_modes` needs `file` (`import.failure_modes.file is required`), an `id` that is
  a regular expression (`import.failure_modes.id is required`,
  `import.failure_modes.id is not a regular expression: <error>`) and a non-empty list for each of
  `columns.failure` and `columns.behavior` (`import.failure_modes.columns.failure is required`).
  `columns.required` is optional: the header names of the column that holds the required value,
  for a catalog that has one (section 5.4.1). An empty header name in any of the three lists is
  `import.failure_modes.columns.<name>: a header name must not be empty`.
  It may be set beside `design.failure_modes`.
  `import.planned` lists where a project already marks an id as planned, with the story that will
  deliver it. Each entry has `in`, `model-statement` (the statement of every invariant of the
  models), `scenario-description` (the `description` of every scenario of the models that has
  one) or `failure-row` (every cell of every row of the imported catalog), and `pattern`, a
  regular expression searched in that text, trimmed, whose named group `story` is the story id;
  anchor it with `^` to ask for a statement that begins with the mark. The first entry that
  matches an id decides, and a match whose `story` is empty is no mark. What `story` captured
  must be a story id, `^[A-Za-z0-9._-]+$`: anything else is refused by `machinery adopt`, by
  source line, and never written (section 17.2, step 3). Only `machinery adopt`
  reads the marks. Config errors, each naming its entry by number, from 1:
  `import.planned: entry <n>: in must be model-statement, scenario-description or failure-row, got "<value>"`,
  `import.planned: entry <n>: pattern is required`,
  `import.planned: entry <n>: pattern is not a regular expression: <error>`,
  `import.planned: entry <n>: pattern "<pattern>" needs a named group story, as in (?P<story>[a-z0-9-]+)` and,
  without `import.failure_modes`,
  `import.planned: entry <n>: in: failure-row needs import.failure_modes; it reads the rows of that catalog`.
- `evidence.adapter` is the policy for adapter evidence (section 10.3). `allow` lists kinds
  among `inv`, `sc`, `fm` and `ck`
  (`evidence.adapter.allow: <value>: the adapter proves only inv, sc, fm and ck ids`) and `tiers`
  lists tiers (`evidence.adapter.tiers: "<value>" is not a tier: routine, consequential, critical or vital`).
  A list that is set replaces its default, and an empty one admits nothing. `min_granularity` is
  `function` or `file`, or a mapping whose keys are `default`, a kind or a tier and whose values
  are `function` or `file`: `{default: file, fm: function, consequential: function}` asks
  function granularity of every `fm` id and of every id of a consequential area. Config errors:
  `evidence.adapter.min_granularity must be function or file, got "<value>"`,
  `evidence.adapter.min_granularity.<key> must be function or file, got "<value>"` and
  `evidence.adapter.min_granularity: "<key>" is not default, a kind (inv, sc, fm, ck) or a tier`.
  Each entry of `sources` needs `package`, a valid glob
  (`evidence.adapter.sources: entry <n>: package is required`,
  `evidence.adapter.sources: entry <n>: package "<value>" is not a valid glob`), and a non-empty
  list `files` of valid globs inside the project
  (`evidence.adapter.sources: entry <n>: files is required`,
  `evidence.adapter.sources: entry <n>: files "<value>" is not a valid glob`, and for a glob
  that leaves the project `evidence.adapter.sources.files: "<value>" must use forward slashes`,
  `... must be relative to the project root` or `... must stay inside the project root`),
  entries numbered from 1. The
  entries are read only by `machinery run --adapter go-test-json --streams`; without `--streams`
  an entry for a package tested in its own directory is refused (section 10.3).
- `plan.wire_max_files` is a count; `0` or unset is the default, 20, and a negative one is a
  config error (`plan.wire_max_files must not be negative`).
- `review.backlog` and `review.design` are `required` or `off`
  (`review.backlog must be required or off, got "<value>"`). Each defaults to `required` when
  `backlog.tool` is set and to `off` otherwise. A project being adopted may set them `off`; the
  ratchet reports a review switched off against a base that required it (section 11.3).
- `delegate.policy` is the one path that may leave the project root: it is absolute, or relative to
  the root, and names a file the owner keeps (section 15.9). An empty one is a config error
  (`delegate.policy is required`).
- `systems` names what a failure mode may be about that is not an entity, a dependency or a
  machine: the system as a whole, or a part of it such as its build or its deployment. A failure
  mode names one as `system:<name>` (section 5.4). An entry without a name
  (`systems: name is required`), a name that does not match the pattern
  (`systems: name "<name>" must match ^[a-z][a-z0-9-]*$`) and a name declared twice
  (`systems: duplicate name "<name>"`) are config errors.
- An empty marker in `trace.markers` is a config error
  (`trace.markers.verifies: a marker must not be empty`), and so is a marker listed in both
  `scenarios` and `scenarios_exact`
  (`trace.markers.scenarios_exact: "<marker>" is also in trace.markers.scenarios; a marker has one mode`).
- `constraints.cores` needs `max_file_lines` and `max_func_lines`, each a positive number
  (`constraints.cores.max_file_lines is required and must be positive`), and an empty entry of
  `deny_imports` is a config error (`constraints.cores.deny_imports: an entry must not be empty`).
- An empty word in `vital.names`, `vital.failure` or `vital.effects` is a config error
  (`vital.names: a word must not be empty`). A list that is set replaces its default; words are
  compared lower-cased.
- `routing` keys are tier names: `routine`, `consequential`, `critical` and `vital`. A tier that is
  not set keeps its default text. The texts name classes of model, never a vendor's model.
- `assess.decisions` needs a non-empty `files` list of valid globs
  (`assess.decisions.files is required`) and a `record` that is a regular expression
  (`assess.decisions.record is required`,
  `assess.decisions.record is not a regular expression: <error>`).
- Globs use doublestar syntax: `**` matches any number of directories, `*` matches inside one path
  segment, `{a,b}` alternates.
- `scope` bounds the static stages that read code (`arch`, `tests`, `trace`): a file outside every
  scope glob is not looked at. This is what lets an existing project adopt one area at a time
  (section 17.2). Scope only widens: narrowing it against the base is a ratchet error
  (section 11.3).
- `posture: real` means tests run against the real dependency. `posture: sandbox` means tests run
  against the vendor's real sandbox endpoint. There is no posture for a stand-in: a dependency
  proven only against a double is not proven.
- The receipts directory is never committed. Everything under `design/` is committed. The run
  ledger (section 10.2) is in the git directory, outside the working tree.
- The tools directory is `tools.dir` when set, else the environment variable `MACHINERY_TOOLS`,
  else `.machinery/tools` in the project.

## 3. Commands, output and exit codes

### 3.1 Commands

| Command | Does |
|---|---|
| `machinery lint` | Stages `model`, `machines`, `failure-modes`, `risk`, `xref-model-machine`, `xref-contract`, `xref-failure`, `xstate`. |
| `machinery gen` | Writes generated files (section 7). |
| `machinery gen --check` | Stage `gen-drift`: fails when a generated file is stale, missing or unexpected. Writes nothing. |
| `machinery arch` | Stage `arch` (section 9). `--write-baseline` rewrites the baseline file. |
| `machinery tests` | Stage `tests` (section 9.4). `--write-baseline` rewrites the baseline file. |
| `machinery trace` | Stage `trace` (section 8). `--non-action-serves` lists the ids that serves tags name and that are not actions, and runs no check. |
| `machinery check` | Every stage of `lint`, then `gen-drift`, `formal`, `checkers`, `arch`, `tests`, `trace`, `contract`, `intake`. With `--base <git-ref>` also stage `ratchet` (section 11.3), and with `--story <id>` as well stage `scope` (section 15.11). |
| `machinery checks` | Prints the declaration of applicable checks (section 3.4). A report, never a gate. |
| `machinery run -- <command...>` | Runs a test command and records a receipts file (section 10). `--adapter go-test-json -- go test ...` also writes the receipts of an existing Go suite from its event stream, and with `--streams -- <command>` from the streams a wrapper leaves (section 10.3). |
| `machinery verify` | Stage `verify` (section 12). `--allow-dirty`, `--affected --base <ref>`, `--story <id>`. |
| `machinery affected --base <ref>` | Prints the obligation ids a change touches (section 12.1). |
| `machinery landing --base <ref>` | The end gate: `check`, the full suite through `run`, `verify` (section 12.2). A failed landing marks the integration branch red. `--repair`; `--clear-red --reason <text>`. |
| `machinery owed list` | Prints the owed list. |
| `machinery owed sync` | Rewrites `owed.yaml` from the current evidence (section 11.2). `--prune-only` never adds. |
| `machinery proof` | Prints a proof block (section 13). `--limit`, `--learning`, `--observation`; `--story <id> --post` writes it to the story. |
| `machinery status` | Prints a summary (section 14). A report, never a gate. |
| `machinery probe -- <command...>` | Stage `probe`: mutates the generated tables and expects the tests to notice (section 20). Never part of `check`. |
| `machinery formal new <spec>` | Scaffolds a spec manifest and a property module (section 16.2). |
| `machinery formal gen` | Writes the generated skeletons of the formal models (section 16.2). `--check` reports drift. |
| `machinery formal check [<spec>...]` | Runs the model checkers and records results (section 16.9). |
| `machinery formal baseline` | Records the baseline of properties, fairness and bounds (section 16.6). |
| `machinery formal sync [<spec>]` | Shows or classifies source drift (section 16.7). |
| `machinery formal tools` | Fetches the pinned formal tools by checksum (section 16.8). |
| `machinery plan` | Prints backlog stories for owed ids; `--apply` creates them in the backlog (section 15.2). |
| `machinery waves` | Prints the parallel waves of the backlog graph (section 15.3). `--freeze`, `--thaw` (section 15.4). |
| `machinery brief <story>` | Prints a brief by file pointer (section 15.5). |
| `machinery predict <story>` | Records a verifier's prediction (section 15.6). |
| `machinery accept <story>` | Records a verifier's verdict and runs the story gate (section 15.6). |
| `machinery story <story>` | Stage `story`: the tier gate of one story (section 15.6). |
| `machinery intake new "<title>" --intent "<text>"` | Records a requirement as an intake item (section 15.7). `--refuses`, `--parent`. |
| `machinery intake status <id>` | Prints where a requirement stands, one line per step (section 15.7). `--json`. |
| `machinery intake plan <id>` | Prints the stories the design change owes; `--apply` creates them in one backlog call (section 15.7). |
| `machinery decide "<title>" --why "<text>"` | Records a decision in the backlog (section 15.8). `--ids`, `--supersedes`, `--vetoable`. |
| `machinery decisions` | Lists the current decisions (section 15.8). `--ids`, `--open-vetoes`. |
| `machinery decisions import --file <path> --record "<regex>"` | Prints one backlog decision per record of an existing decision file; `--apply` creates them (section 15.8). |
| `machinery review <backlog\|design <id>\|intake <id>>` | Records the checker's review, approved or rejected, that `waves --freeze` and `intake plan --apply` wait for (section 15.10). |
| `machinery ask "<question>"` | Records a question for the owner, with its options, a recommendation and the default (section 15.9). Experimental. |
| `machinery answer <id>` | Records the owner's answer; with `--delegate`, an answer on the owner's behalf under the delegate policy (section 15.9). Experimental. |
| `machinery questions` | Lists the open questions and what each waits for (section 15.9). `--held`. |
| `machinery observations` | Lists the observations of stories that are not filed; `--file` files them (section 15.8). |
| `machinery learnings` | Prints the learnings of closed stories, by intake (section 15.8). `--since <ref>`. |
| `machinery checkers run [<id>...]` | Runs the external checkers registered on this machine and records their evidence (section 22). |
| `machinery checkers list` | Lists the configured checkers, whether each is registered locally, and the state of its evidence. |
| `machinery init` | Scaffolds a new project (section 17.1). |
| `machinery adopt` | Brings an existing project under machinery without touching its code (section 17.2). `--rehearse` adopts a snapshot outside the project and reports what blocks (section 17.3). |
| `machinery assess` | Reports, read-only, where an existing project stands against its own design (section 18). |
| `machinery vital candidates` | Lists the design ids that look as if they belong in a vital area and are in none, with the reason (section 23.3). A report, never a gate. |
| `machinery hook <event>` | Reads a harness's hook JSON on stdin and answers with hook JSON on stdout: `session-start`, `pre-tool`, `stop` (section 21.1). Always exits 0. |
| `machinery config` | Prints the effective configuration as YAML, defaults applied. |
| `machinery version` | Prints `machinery <version>`. |

`lint` and `check` accept `--stage <name>` (repeatable): only the named stages run, in their usual
order, and the exit code depends on them alone. An unknown stage name is a usage error (exit 2).

`machinery` with no arguments prints help and exits 0. `version`, `init`, `adopt`,
`formal tools`, `hook` and help work without a `machinery.yaml`, and `assess` with `--config`; every
other command needs one.

### 3.2 Stage output

Every stage prints one block on stdout, in this exact shape:

```
== <stage name> ==
checked: <n> <noun>, <n> <noun>
  ERROR  <message>
  DRIFT  <message>
  warn   <message>
<outcome>
```

The last line is the stage's outcome, exactly one of four:

| Outcome line | Meaning | Blocks |
|---|---|---|
| `passed` | The stage looked at something and found no ERROR and no DRIFT. | No |
| `failed: <n> finding(s)` | `<n>` counts ERROR plus DRIFT. | Yes |
| `not applicable: <absent feature>` | The project does not have what the stage checks, and the line names what is absent (`no contracts configured`). No `checked:` line is printed. | No |
| `not demonstrated: <reason>` | The stage applies but could not establish its claim: a required tool is missing, nothing was executed, there is no evidence. | Yes |

- The `checked:` line lists what the stage looked at, as counts.
- Findings follow, one per line, ERROR first, then DRIFT, then warn, each group sorted by message.
- When a stage has findings and a reason for not demonstrated, the outcome is `failed`.
- The version rules (Modelith 0.6.0 or newer, XState 5.33.4 or newer and below 6) are floors,
  so that a patch release upstream does not break a project. This repository pins the exact
  versions in its Makefile.
- A stage that applies and found nothing to check is `not demonstrated: nothing to check`. A pass
  needs positive evidence; absence is never a pass.

After the last stage a command prints one summary line:

```
machinery <command>: <p> passed, <f> failed, <d> not demonstrated, <n> not applicable
```

Messages name files by project-relative path. Output is deterministic: no timestamps, no absolute
paths, stable ordering.

### 3.3 Exit codes

| Code | Meaning |
|---|---|
| 0 | Every stage passed or was not applicable. |
| 1 | At least one stage failed or was not demonstrated. |
| 2 | Usage or config error. Nothing was checked. |

`machinery run` is the exception: it exits with the exit code of the command it ran (section 10).

### 3.4 The declaration of applicable checks

`machinery checks` prints one line per stage: its name, whether it applies to this project
(with the absent feature when it does not) and the product behavior it protects. The
configuration file is the versioned declaration; this command renders it.

| Stage | Protects |
|---|---|
| `model` | The vocabulary every other artifact uses: entities, actions, invariants, scenarios. |
| `machines` | Every state answers every event: a transition, a refusal or a reasoned ignore. |
| `failure-modes` | Each declared failure and abuse case is well formed and has a required behavior. |
| `risk` | Each risk area and status entry names real design ids, so tier gates and statuses attach to real behavior. |
| `xref-model-machine` | The lifecycle a machine implements is the lifecycle the model promises. |
| `xref-contract` | Every action is reachable through a published operation. |
| `xref-failure` | Every side effect, dependency, failure state, failure event and tiered action has a declared failure behavior. |
| `xstate` | A generated table means what XState would execute for the same machine. |
| `gen-drift` | Tables and obligations in the tree are the ones the design produces. |
| `formal` | The formal model of a critical area is current, not vacuous and not weakened. |
| `checkers` | Verdicts of external checkers are for this exact design. |
| `arch` | The layering the design declares holds in the code. |
| `tests` | Evidence comes from the real system: no unit tests, no test doubles. |
| `trace` | Every action is served by code and every obligation is claimed by a test or owed. |
| `contract` | Stories running in parallel build against the same design. |
| `intake` | A requirement reaches the backlog only through the design it changed, and is closed once delivered. |
| `ratchet` | What was proven, covered or modeled stays so. |
| `scope` | A story writes only inside its declared scope, and leaves the design to an intake. |
| `verify` | Each obligation is observed on the running system at this commit. |
| `probe` | The tests would notice a wrong transition or a missing effect. |
| `story` | A story is accepted with the formality its risk tier requires. |

## 4. Ids

Everything the design declares has an id. Ids are how code, tests, receipts, stories and the owed
list refer to the design. An id is `<kind>:<name>`.

| Kind | Form | Comes from | Role |
|---|---|---|---|
| `act` | `act:<Entity>.<action>` | Each action of each model entity | Served: code must declare it serves it. |
| `inv` | `inv:<invariant-id>` | Top-level and entity-level model invariants | Obligation. |
| `sc` | `sc:<slug>` | Each model scenario, slug of its `name` | Obligation. |
| `tc` | `tc:<TAG>-<hex>` | Each transition case of each machine (section 7.2) | Obligation. |
| `fm` | `fm:<failure-mode-id>` | Each failure mode (section 5.4) | Obligation. |
| `fv` | `fv:<area>` | Each critical or vital area of the risk file (section 23) | Obligation, proven by the formal stage. |
| `cx` | `cx:<spec>:<run>` | Each counterexample trace of a formal model (section 16.10) | Obligation, proven by replay. |
| `ck` | `ck:<checker id>:<name>` | Each obligation an external checker's evidence declares (section 22.5) | Obligation. |
| `pf` | `pf:<area>:<name>` | Each property of a vital area of the risk file (section 23.1) | Proof obligation: awaiting proof, never owed and never proven in this version (section 23.2). |

- The scenario slug is the scenario name lowercased, every run of characters outside `[a-z0-9]`
  replaced by one `-`, with leading and trailing `-` removed. Example: `Customer pays and the
  order ships` becomes `sc:customer-pays-and-the-order-ships`.
- Ids are unique across the whole project. A duplicate is an ERROR in the stage that reads it.
- An **obligation** is an id that must be either proven at the current commit or listed in
  `owed.yaml`. The obligation kinds are `inv`, `sc`, `tc`, `fm`, `fv`, `cx` and `ck`.
- A **proof obligation** (`pf`) is what a machine-checked proof of the code will one day
  discharge. Nothing in this version can: a `pf` id is awaiting proof, is not owed (it would
  block every gate forever), is never proven, is not in `obligations.json` and no test may claim
  it (section 23.2).
- An id pattern is a full id or a prefix ending in `*` (`tc:ORDE-*`, `act:Order.*`).
- An `fv` id and a `pf` id have the tier of their area, and a `cx` id the highest tier among the
  areas its formal model lists (section 23).

### 4.1 Statuses: `design/status.yaml`

Every `act` and obligation id the design declares has one of four statuses:

| Status | Meaning | Yields an obligation |
|---|---|---|
| `proposed` | A candidate. The design declares it; nobody is committed to deliver it. | No |
| `owed` | Committed and not proven yet: listed in `owed.yaml` (section 11). | Yes |
| `proven` | Committed and not owed: `verify` holds it proven (section 12). | Yes |
| `retired` | No longer promised. The design still declares it; its entry is the record. | No |

Owed and proven are what every id was before statuses existed. Proposed and retired are data in an
optional file, `design.status` (default `design/status.yaml`), read when it exists:

```yaml
version: 1
proposed:
  - {id: "inv:refund-window", note: "candidate; no delivery commitment"}     # note optional
retired:
  - {id: "act:Order.gift", reason: "removed in the 2.0 contract", since: "v2.0"}   # since optional
```

- `id` is an id or an id pattern (section 4) of kind `act`, `inv`, `sc`, `tc` or `fm`. A pattern
  gives its status to every id it matches.
- A proposed or retired id yields no obligation. It is not in `obligations.json` (section 7.1);
  stage `trace` asks no claim for it; `verify`, `owed sync`, `proof` and `plan` leave it out; it
  is not a new id of an intake (section 15.7). Nothing may claim it (section 8) and it may not be
  owed: an entry of `owed.yaml` that names it is
  `ERROR  proposed: <id> is in <owed file>; a proposed id is not owed: remove it, or commit it first`
  (`retired: ... a retired id is not owed: remove it, or un-retire it`) wherever the owed file is
  read, as an unknown owed id is.
- The lint stages judge the design as declared, with one exception: `xref-contract` and
  `xref-failure` ask an operation, a failure mode or an abuse case of every action that yields an
  obligation, so a proposed or retired action needs none.
- Committing a proposed id, and un-retiring a retired one, is removing its entry: the id becomes an
  obligation again, owed or proven like any other.
- The file is a design source: it is part of the contract hash (section 15.4), and a change to it
  changes `obligations.json`.
- The ratchet (section 11.3) holds the move between statuses: what was owed or proven at the base
  is never demoted to proposed, and retiring what was proven is reported.

Stage `risk` (section 23) checks the file. Each of these is an ERROR, `<file>: <message>`: a file
that does not parse or has an unknown key; `version must be 1`; `proposed entry <n>: id is
required` (`retired entry <n>`); `retired: <id>: reason is required`; `<list>: <id>: only act,
inv, sc, tc and fm ids take a status`; `<list>: <id> matches no design id`; `<list>: <id> is listed
twice`; `<id> is both proposed and retired`, for an id that entries of both lists match.

## 5. Design sources

### 5.1 Domain model

Modelith YAML, format `v1`, Modelith 0.6.0 or newer. machinery does not reimplement Modelith: it
runs the `modelith` executable for validation and rendering, and reads the YAML itself only to
extract ids, entity attributes, enums and actions.

The committed rendering `<name>.modelith.md` sits beside each model. It is produced by
`modelith render --stdout <file>` with one mechanical change: every em dash (U+2014) is replaced
by a hyphen-minus.

### 5.2 Machines

One file per machine, `<Name>.machine.json`, in the machines directory. The format is the XState
v5 JSON-serializable subset documented in `docs/machine-format.md`: guards, actions and actors are
named by string and implemented in code.

Every machine binds to the model, as one of:

- a **lifecycle machine** of a model entity: the root key `_lifecycle_of: "<Entity>"`, or a file
  base name equal to an entity name (`Order.machine.json` for entity `Order`);
- an **operational machine**: the root key `_role: "operational"` and the root `_serves`, a
  non-empty list of the model ids it exists for, each `entity:<Name>` or `act:<Entity>.<action>`
  (a command-execution envelope serves the actions it runs, a notifier the entity it reports on).

A machine that binds to nothing is a finding of stage `xref-model-machine` (section 6.4): no
cross-check is skipped because nothing says what a machine is for.

An entity has one lifecycle machine, or several that are **aspects** of one lifecycle: each
declares the root `_aspect: "<name>"`, a short name unique among the lifecycle machines of that
entity, and holds the states of a state attribute of its own (an order's payment and its
delivery, a parcel and its customs check). A second lifecycle machine without `_aspect` is a
finding, never accepted silently.

In a lifecycle machine, top-level states whose name starts with an uppercase letter are **domain
states**. States whose name starts with a lowercase letter are transient overlay states.

A machine also declares, with annotations XState ignores and a tag XState carries:

- **entry states**: the root `_entries` lists top-level states, besides `initial`, in which an
  instance may be created directly (an imported or migrated instance);
- **terminal states**: a non-final leaf state with `_terminal: "<reason>"` is never left by design
  but still answers events (a durable resource keeps refusing after its end);
- **failure states**: a state with the XState tag `failure` records that something went wrong;
- **failure events**: the root `_failure_events` lists the events that report a failure arriving
  from outside (a dependency failed, a crash was detected, a lease was lost);
- **external events**: the root `_external_events` lists the events that arrive from outside the
  caller (a timer, a callback, another machine) and are no model action;
- **aliases**: the root `_aliases` maps an event to the model action it is under another name
  (`{"handed_back": "giveBack"}`), so the cross-reference rules of section 6.4 read the event
  as that action;
- **refusal events**: the root `_refusal_events` maps an event by which the machine itself writes
  that it refused a caller's command to the model action it refuses and the reason it is a
  refusal and no failure
  (`{"cancel_refused": {"action": "cancel", "reason": "a paid order is not cancelled"}}`, for a
  machine that handles `cancel` and, where it does not accept it, records `cancel_refused`).
  The event is no action, no event from outside and no failure: it is declared, with its
  reason, and printed (section 6.4);
- **dismissed hints**: the root `_not_failures` maps an event name or the dotted path of a state
  to the reason its name only looks like a failure (section 6.2);
- **callers**: a transition branch with `_requires` names who may cause it, an actor or role, or
  `system/<name>` for an internal caller.

`docs/machine-format.md` has their rules; stage `machines` (section 6.2) holds them.

#### 5.2.1 Machines in a project's own format

A project that already keeps state machines in a JSON dialect of its own names them in
`import.machines` instead of `design.machines` (section 2). machinery translates each file in
memory, the same way every time, and never writes it. Every stage (`machines`, the
cross-reference stages, `xstate`, `gen`, `formal gen`, `trace`, `verify`) reads the translated
machine exactly as if it were a native file, and every finding names the source file, so the
team sees where to look.

Each file the `files` globs match is translated, in path order, once whatever glob matches it.
The globs may name files anywhere in the project: machines kept beside the code they drive (a
`states.json` in each service directory) are read with the rest. A file under the gen
directory is generated, never a source, and is not read, and neither is the register file
below, even when a glob matches it:

1. **Parsing.** A file that is not valid JSON is `ERROR  <file>: invalid JSON in <file>: <error>`,
   and one whose top level is not an object is `ERROR  <file>: machine must be a JSON object`.
2. **Keys.** The keys of the root object, of every state object (under `states`, at any depth)
   and of every transition branch (under `on`, `after`, `always`, a state's `onDone`, and an
   invoke's `onDone` and `onError`) are renamed by the `rename` map of their level (`root`,
   `state`, `transition`); every other key is kept as written. Then:
   - A value that lands on `_comment` from a renamed key is prefixed `<source key>: ` (a value
     that is not a string is written as compact JSON). When several values land on `_comment` at
     one node, they are joined with ` | ` in key order; a native `_comment` joins unprefixed.
   - A transition key renamed to `_refusal` is removed from the branch and added to its state's
     `_refusal`, keyed by the handler name of `docs/machine-format.md` ("Guard-false
     dispositions"): the event for an `on` branch, `after:<delay>`, `onDone` for a state's own
     completion, `<src>.onDone` and `<src>.onError` for an invoke, and `always` for an
     `always` branch, which the lint then reports as a stale disposition. Dispositions that land
     on one handler are joined with ` | ` in document order.
   - Two keys of one node that land on the same other key are
     `ERROR  <file>: <node>: keys <a> and <b> both become <key>`, and the first is kept. The node is `root`,
     `state <path>`, or `state <path> <trigger>` (the trigger of section 7.2) with ` branch <n>`
     for a branch of a list; paths are in source names and `<n>` counts from 1.
3. **States.** `states` says how every state name is written, so that the project's names
   become domain states (section 5.2):
   - `capitalize` upper-cases the first letter of the name and keeps the rest as written:
     `on_hold` becomes `On_hold`, `inReview` becomes `InReview`;
   - `pascal` reads the name as words separated by `_` or `-`, upper-cases the first letter of
     each word, keeps its other letters as written and drops the separators: `on_hold` and
     `on-hold` become `OnHold`, `in_review` becomes `InReview`, `v2_draft` becomes `V2Draft`
     (a name made of separators only is kept as written);
   - `keep` (the default) changes nothing.

   The change applies to the keys under every `states`, every `initial`, every target (a bare
   name, and each path segment after the head of a `#<id>.<path>` target), and the values of
   `_entries`, native or renamed. Two sibling states that become one name are
   `ERROR  <file>: states <a> and <b> both become <name> with states: <value>`, and the first is
   kept.
4. **Terminal states.** A project may say once, at the root, that a machine never ends: with
   `terminal: {from_root: <key>}`, the value of that root key is the reason (a value that is not
   a string is written as compact JSON). The key is taken out of the root, and the reason becomes
   `_terminal` on every absorbing state of the translated machine: a non-final leaf state with
   no `_terminal` of its own that has transitions and none that leaves it, on itself or on an
   ancestor (`docs/machine-format.md`, "Reachability"). A machine that has the key and no such
   state keeps the reason as `_comment`, exactly as `rename.root: {<key>: _comment}` would. A
   machine without the key is not changed. A state key that `rename.state` maps to `_terminal`
   gives its state the reason directly, as written; stage `machines` holds either to the rules of
   `_terminal` (a non-empty reason on a non-final leaf state that is never left).
5. **Id.** The original id is the root `id`, or the file name without its extension when there
   is none. The machine id is the original id with every character outside `[A-Za-z0-9_]`
   replaced by `_`. It is the translated root `id` (written first when the source has no `id`),
   and a target head `#<original id>` becomes `#<machine id>`. The translated file name is
   `<machine id>.machine.json`, so a lifecycle binding by file name (above) uses the machine id.
   Two files that translate to the same id are
   `ERROR  <second file>: translates to machine id <id>, as <first file> does; give one of them another id`,
   and the second is not read.
6. **Binding.** The translation never invents a binding. A result that has `_lifecycle_of`,
   `_role` or `_serves` binds itself and is kept as it is. Any other is bound by the entry of
   `import.machines.bind` whose key is its machine id or a glob that matches its source file:
   `lifecycle_of: <Entity>` adds `_lifecycle_of` (and `aspect: <name>` adds `_aspect` after it),
   and `serves: [<ids>]` adds `_role: "operational"` and `_serves`. A machine with no binding,
   from its source or from `bind`, is reported by `machine-binding` like any other (section 6.4).
   Then:
   - A machine that binds itself and is matched by an entry is
     `ERROR  <file>: bound in the source and by import.machines.bind <key>; keep one`.
   - An entry with `aspect` for a machine whose source has `_aspect` is
     `ERROR  <file>: _aspect is set in the source and by import.machines.bind <key>; keep one`.
   - A machine matched by two entries is
     `ERROR  <file>: import.machines.bind <a> and <b> both match; keep one` (the first two
     keys, in sorted order), and none is applied.
   - An entry that matches no machine is
     `ERROR  import.machines.bind <key>: matches no machine file and no machine id`.
7. **Annotations.** A dialect may have no key that says which events are not caller actions.
   Every entry of `import.machines.annotate` whose key is the machine id or a glob that matches
   the source file applies to the machine, in key order, and what the entries say adds up:
   - `external_events` and `failure_events` hold event names and patterns, in which `*` stands
     for any run of characters and every other character for itself (`*_refused`, `due_*`). The
     events of the machine are the ones a state handles under `on` or lists in `_ignores`. Every
     event a pattern matches is added to the root `_external_events` or `_failure_events`, after
     the names the source lists there, in sorted order, each once.
   - `failure_states` holds state names and patterns. A pattern matches a state by its name or
     by its dotted path, in the names the source writes or in the translated ones (`*_failed`
     and `ScanFailed` both name the state `scan_failed` under `states: pascal`). Every state it
     matches, at any depth, gets the tag `failure`, added to the tags it has.
   - `aliases` maps an event of the machine, by its exact name, to an action name. Each is added
     to the root `_aliases` (section 5.2), after the ones the source has, in event order.
   - `refusal_events` maps an event of the machine, by its exact name, to `{action, reason}`:
     the name of the action it refuses and the reason. Each is added to the root
     `_refusal_events` (section 5.2), after the ones the source has, in event order. When
     several entries name one event with the same action, the reason is that of the first.
   - `not_failures` maps an exact name to a reason: an event of the machine, or a state by its
     dotted path or by its name, in source or translated names. A dotted path names its state; a
     bare name names the one state that carries it, and is refused when several do. Each is
     added to the root `_not_failures` (section 5.2), an event under its name and a state under
     its translated dotted path, after the ones the source has. A name that is both an event
     and a state of the machine dismisses both. A name with no failure word in it (section 6.2)
     has no hint, so dismissing it is refused.

   The translated machine then carries the annotations of a native file, and every rule about
   them is the native one: an event in both lists, an aliased event in `_external_events`, a
   refusal event that is also listed or aliased (section 6.2), an alias or a refusal that names
   no action, and an event listed as external or as a failure that is an action of the model
   (section 6.4), whose finding names the entry and the pattern that listed it. An event may be
   given an alias and be matched by `failure_events` at once: the action it stands for is then
   itself the failing outcome (section 6.4). No entry hides a caller's command: an event that is
   an action of the model is neither
   listed nor aliased nor declared a refusal, every alias and every refusal event in effect is
   printed and counted by `xref-model-machine` (section 6.4), and neither listing an event as
   external nor giving it an alias stops its name hint (section 6.2); a declared refusal has
   none. Nothing is a silent no-op:
   - A pattern that matches no event of a machine it is applied to is
     `ERROR  <file>: import.machines.annotate <key>: <list> pattern <pattern> matches no event of the machine; correct the pattern or remove it`
     (`<list>` is `external_events` or `failure_events`), and one that matches no state is
     `ERROR  <file>: import.machines.annotate <key>: failure_states pattern <pattern> matches no state of the machine; correct the pattern or remove it`.
     A glob entry holds its patterns to every machine it matches.
   - An alias for an event the machine does not have is
     `ERROR  <file>: import.machines.annotate <key>: aliases names event <event>, which no state of the machine handles or ignores; correct the name or remove the alias`.
   - Two entries that give one event different actions are
     `ERROR  <file>: import.machines.annotate <a> and <b> give event <event> different actions (<x>, <y>); keep one`
     (the first is kept), and an entry that gives an event another action than the source's own
     `_aliases` is
     `ERROR  <file>: import.machines.annotate <key> gives event <event> the action <x>, and the source gives it <y>; keep one`
     (the source's is kept).
   - The same three hold for `refusal_events`:
     `ERROR  <file>: import.machines.annotate <key>: refusal_events names event <event>, which no state of the machine handles or ignores; correct the name or remove the refusal`,
     `ERROR  <file>: import.machines.annotate <a> and <b> give event <event> different refused actions (<x>, <y>); keep one`
     and
     `ERROR  <file>: import.machines.annotate <key> gives event <event> the refused action <x>, and the source gives it <y>; keep one`.
   - A refusal the config declares for an event that is also listed or aliased names the entry
     to change, and what lists the event, since the source file has none of these keys:
     `ERROR  <file>: import.machines.annotate <key>: refusal_events names <event>, which is also in _external_events (<by>); the machine writes a refusal itself, so it does not arrive from outside: keep one`,
     `ERROR  <file>: import.machines.annotate <key>: refusal_events names <event>, which is also in _failure_events (<by>); a refusal is declared, not a failure: keep one`
     and
     `ERROR  <file>: import.machines.annotate <key>: refusal_events names <event>, which also has an alias (<by>); an event refuses an action or is one under another name: keep one`.
     `<by>` is `import.machines.annotate <key> lists it for <file> (<list> pattern <pattern>)`
     or `the source lists it`, and for an alias `import.machines.annotate <key> gives it` or
     `the source gives it`. Such a refusal is not written to the translated machine, so it is
     the only finding. A source entry that is no `{action, reason}` is the finding of the
     lint alone (section 6.2), whatever the config says about the event. A refusal the source itself declares is held by the native rule
     (section 6.2), which names the source file.
   - A dismissal that names nothing is
     `ERROR  <file>: import.machines.annotate <key>: not_failures names <name>, which is no event and no state of the machine; correct the name or remove it`;
     one name dismissed by two entries is
     `ERROR  <file>: import.machines.annotate <a> and <b> both dismiss <name>; keep one`, and one
     the source dismisses already is
     `ERROR  <file>: import.machines.annotate <key> dismisses <name>, which the source dismisses already; keep one`.
     A bare name that several states carry is
     `ERROR  <file>: import.machines.annotate <key>: not_failures names <name>, which is the name of several states (<paths>); name one by its dotted path`
     (translated paths, in document order), and a name that has no hint to dismiss is
     `ERROR  <file>: import.machines.annotate <key>: not_failures names <name>, which does not look like a failure (no fail, error, abort, crash or timeout in the name), so there is no hint to dismiss; remove it`.
     Neither is written to the translated machine.
     One entry that names one state in two ways is
     `ERROR  <file>: import.machines.annotate <key>: not_failures names <a> and <b>, which are one state (<path>); keep one`
     (the first is kept).
   - A declared failure is not dismissed, whether the source or an entry declares it:
     `ERROR  <file>: import.machines.annotate <key>: not_failures names <name>, which is a failure event of the machine; a declared failure is not dismissed (keep one)`,
     and for a state
     `ERROR  <file>: import.machines.annotate <key>: not_failures names <name>, which is the state <path>, tagged failure; a declared failure is not dismissed (keep one)`.
     Such a dismissal is not written to the translated machine. Neither is one of a refusal
     event, which is never hinted:
     `ERROR  <file>: import.machines.annotate <key>: not_failures names <name>, which is a refusal event of the machine; a refusal is never hinted, so there is no hint to dismiss (remove it)`.
   - An entry that matches no machine is
     `ERROR  import.machines.annotate <key>: matches no machine file and no machine id; correct the key or remove the entry`.
8. **Tag.** `_oracle_tag` is added when the result has none: the first four letters or digits of
   the machine id, upper-cased, followed by the first four hex digits, upper-cased, of the SHA-256
   of the original id. Dozens of machines whose ids share a prefix get different tags, and a tag
   does not change while the id does not.

**The register.** A project may keep a register that ties each machine to the tables and code
that hold its state and lists the transitions the code is seen to make. `import.machines.register`
reads it as a checked source. `file` is a JSON or YAML document whose top level is a list of
entries; `machine` names the key of an entry that holds its machine (the original id or the
machine id), and `observed` the key that holds a list of objects with the strings `from`,
`event` and `to`: state paths in source names, written with the `states` setting (each dotted
segment), and an event name. Other keys of an entry are not read. Stage `machines` then holds:

- `register-undeclared`: every observed transition is a declared transition of its machine, an
  `on` row of the runtime table (section 7.3) whose event is `event`, whose source is `from`
  or an ancestor of it, and whose target, or the leaf that target enters, is `to` (`from` for
  an internal row):
  `ERROR  register-undeclared: <machine id>: <register> observes <from> -> <to> on <event>, which the machine does not declare; declare it in <source file>, or correct the register`,
  with `<from>`, `<to>` as the register writes them.
- `register-unobserved`: every `on` row of every imported machine is observed by its entry:
  `ERROR  register-unobserved: <machine id>: <source> -> <target> on <event> is declared and not observed in <register>; add it to the register, or remove the transition from <source file>`,
  with ` [<guard>]` after the event for a guarded row and `<source> (internal)` as the target of
  an internal one. A machine with `on` rows that no entry names is one finding, not one per row:
  `ERROR  register-unobserved: <machine id>: no entry of <register> names the machine, so none of its <n> transition(s) is observed; add an entry for it (<machine key>: <original id>)`.
- The register itself: a file that cannot be read or parsed, or whose top level is not a list,
  is `ERROR  <register>: <what is wrong>`. An entry that is not an object, has no string under
  the machine key, or whose observed key is not a list of such objects is
  `ERROR  <register>: entry <n>: <what is wrong>` and is not read. An entry that names a machine
  that was not imported is
  `ERROR  <register>: entry <n>: <machine key> <name> names no imported machine; add its file to import.machines.files, or correct the entry`,
  and a second entry for one machine is
  `ERROR  <register>: entry <n>: machine <name> is also named by entry <m>; keep one`.

`<n>` counts entries from 1. With a register, `checked:` of stage `machines` adds the entries
and the observed transitions read:
`checked: 3 machines, 9 states, 8 transitions, 3 register entries, 7 observed transitions`. The
register is a design source: it is part of the contract hash (section 15.4).

The keys a binding adds, then an added `_external_events`, `_failure_events`, `_aliases` and
`_not_failures`, then an added `_oracle_tag`, come last in the root; every other key keeps its
place, and so does a list the source has and an entry adds to. An added `tags` comes last in its
state. A key that is still unknown after renaming is a finding of stage `machines`, as in a
native file: nothing is dropped silently. The runtime table of an imported machine names the
source file in `source`, and its
`source_sha256` is the SHA-256 of the source bytes (section 7.3). The translated bytes are what
`machinery gen` writes under `<gen>/imported/machines/` (section 7) and what stage `xstate`
loads.

### 5.3 Contracts

- `kind: openapi`: an OpenAPI 3.x document in YAML or JSON. An operation binds to a model action
  with the extension `x-machinery-action: "<Entity>.<action>"` (a string or a list of strings).
- `kind: proto`: a `.proto` file. An rpc binds to a model action with a comment line
  `// machinery:action <Entity>.<action>` directly above the `rpc` line. Such a comment anywhere
  else is an ERROR: a binding is never dropped silently.

### 5.4 Failure modes

`failure-modes.yaml` holds everything that can go wrong and what the system must then do. That
includes abuse: an adversary's attempt is a failure mode of kind `abuse`. There is no separate
threat document.

```yaml
version: 1
failure_modes:
  - id: payment-provider-down            # ^[a-z][a-z0-9-]+$
    kind: fault                          # fault | abuse; optional, default fault
    subject: invoke:order.chargePayment  # see the subject forms below
    failure: "The payment provider does not answer."
    required: retry-bounded              # fail-closed | retry-bounded | surface | refuse
    behavior: "The order returns to Pending and the caller sees PAYMENT_UNAVAILABLE."
  - id: ship-by-non-clerk
    kind: abuse
    subject: act:Order.ship
    failure: "A caller who is not a Clerk asks to ship an order."
    required: refuse
    behavior: "The request is refused with FORBIDDEN and the order does not change."
```

`id`, `subject`, `failure`, `required` and `behavior` are required. Unknown keys are an ERROR. In
this file a missing `required` is an ERROR of stage `failure-modes`; only an entry of the links
file of an imported catalog may leave it out (section 5.4.1).

Subject forms: `invoke:<machine id>.<src>`, `dependency:<name>`, `act:<Entity>.<action>`,
`entity:<Name>`, `state:<machine id>.<State>` (a failure state, a nested one by its dotted path),
`event:<machine id>.<event>` (a failure event, or a refusal event, section 5.2),
`machine:<id>` (a machine as a whole) and `system:<name>` (the system as a whole or a
named part of it, declared under `systems`, section 2). `required: refuse` is only valid with
`kind: abuse`, and `kind: abuse` needs an `act` or `entity` subject. A subject of another form
is `ERROR  <file>: <id>: subject "<subject>" is not invoke:<machine id>.<src>, dependency:<name>, act:<Entity>.<action>, entity:<Name>, state:<machine id>.<State>, event:<machine id>.<event>, machine:<id> or system:<name>`.

A `machine:` subject is for a failure mode about a machine as a whole, not about one of its
states, events or invokes: a queue that stops taking work, a flow that stalls. The id is the id
of any machine of the design, a lifecycle machine, an operational one, or one that binds to
nothing. Rule `subject-resolves` holds it to that (section 6.6), and that is the only rule it
takes part in: it covers no failure state, no failure event and no invoke of the machine, each
of which still asks for a failure mode of its own, and it is no subject for `kind: abuse`.

A `system:` subject is for a failure mode about nothing the model, the machines or the
dependencies hold: a release built without a signature, clocks that disagree. Its name must be
declared: an unknown one is the ERROR `unknown-system` of section 6.6, which no waiver covers.
That is the only check it takes part in: it is compared with no entity, action, state or event,
and no rule asks for a failure mode per declared system. `required`, `failure` and `behavior` are as mandatory as for
any failure mode.

A failure mode that names no subject at all says so, with the reason, in place of `subject`:

```yaml
  - id: audit-log-full
    no_subject: "the audit log is written by the host, which the design does not model"
    failure: "The audit log cannot take another line."
    required: fail-closed
    behavior: "The shop stops taking orders until there is room."
```

It is an obligation like any other and takes part in no subject rule of `xref-failure`. Both
keys together are `ERROR  <file>: <id>: subject and no_subject are both set; name the subject or say why there is none`,
a blank reason is `ERROR  <file>: <id>: no_subject needs a reason`, a value that is not a YAML
string (a boolean, a number, a list) is
`ERROR  <file>: line <n>: no_subject must be a string that gives the reason` and the file is then
not read, in the links file too, and `kind: abuse` is refused
as for any failure mode without an `act` or `entity` subject.

#### 5.4.1 Failure modes kept as Markdown tables

A project that already keeps its failure modes as tables in a Markdown document names it in
`import.failure_modes` (section 2). machinery reads the tables and never writes the document.

- **Tables.** A table is a header row, a delimiter row (cells of `-`, with an optional `:` at
  either end) and the rows after it, up to the first line that is not a table row. Leading and
  trailing pipes are optional, and `\|` is a pipe inside a cell. A table inside a fenced code
  block is not read. A row line outside every table whose first cell matches `id` (a blank line
  ended its table, or it never had a header) is
  `ERROR  <file>:<line>: <cell>: the row is in no table (a table needs a header row and a delimiter row above its rows, and a blank line ends it)`,
  and it is not read. Consecutive row lines outside every table have one missing header between
  them, so they are one finding, at the first matching row, when more than one of them matches:
  `ERROR  <file>:<line>: <cell>: the row and the <n> after it, to <last cell> on line <m>, are in no table (a table needs a header row and a delimiter row above its rows, and a blank line ends it)`.
- **Rows.** A row whose first cell, trimmed, matches `id` in full is a failure mode. A row whose
  first cell does not match is ignored, and so is a table with no such row.
- **Columns.** The header row of a row's own table decides its columns, by name and never by
  position: for each of `failure` and `behavior`, the first name of the list that is a header
  cell (compared trimmed and case-insensitively) is the column. A failure-mode row in a table
  whose header has no failure column is
  `ERROR  <file>:<line>: <cell>: the table header has no failure column (<names>)`, and the same
  for behavior. An empty failure or behavior cell is
  `ERROR  <file>:<line>: <cell>: the failure cell is empty`: section 5.4 requires both.
- **The required column.** With `columns.required`, the column is found the same way, per
  table, and a cell that holds one of the four values of section 5.4, compared trimmed and
  case-insensitively, is the row's `required` value. An empty cell, and a table whose header has
  no such column, give the row no value and are no finding. Any other cell is
  `ERROR  <file>:<line>: <cell>: the required cell "<value>" must be fail-closed, retry-bounded, surface or refuse`,
  and gives the row no value.
- **Ids.** The machinery id is the cell lower-cased (`RISK-12` is `risk-12`, obligation
  `fm:risk-12`). It must match the pattern of section 5.4:
  `ERROR  <file>:<line>: <cell>: the id lower-cased, <id>, must match ^[a-z][a-z0-9-]+$`. A
  duplicate is `ERROR  <file>:<line>: <cell>: duplicate id, first on line <n>`, and the later row
  is not read. An id that `design.failure_modes` also declares is
  `ERROR  <file>:<line>: <cell>: fm:<id> is also declared in <native file>`, and the row is not
  read.
- **Text.** The failure cell is the failure mode's `failure`, and so the obligation title; the
  behavior cell is its `behavior`. Both are kept as written, trimmed.

An imported row has no subject until it is linked, and no required value either unless a cell of
its own gives one (`columns.required`). The links file gives them:

```yaml
version: 1
links:
  - {id: RISK-3, subject: "invoke:loan_desk.reserveCopy", required: retry-bounded}
  - {id: RISK-7, kind: abuse, subject: "act:Loan.renew", required: refuse}
  - {ids: [RISK-10..RISK-24, RISK-31], subject: "system:build", required: fail-closed}
  - {id: RISK-40, no_subject: "the shelf list is kept on paper, outside the system"}
```

An entry names one row with `id` or many with `ids`, never both
(`ERROR  <links file>: link <n>: id and ids are both set; keep one`,
`ERROR  <links file>: link <n>: id or ids is required`; `<n>` counts the entries from 1). An id is
the catalog id, compared case-insensitively.

- **`ids`** is a list of ids and of inclusive ranges `<first>..<last>`. The two ends of a range
  share everything before their trailing digits, compared case-insensitively, and the range
  names that prefix followed by every number from the first to the last, each written with at
  least as many digits as the first end has: `FM-01..FM-23` is `FM-01`, `FM-02` to `FM-23`, and
  `FM-08..FM-101` is `FM-08` to `FM-99`, then `FM-100` and `FM-101`. A range holds at most 10000
  ids.
- **Values.** `subject` is required; `required` and `kind` are optional (default `fault`),
  with the value rules of section 5.4 and its messages. An entry that names a subject and gives
  no `required`, for a row whose own cell gives none either, is no finding: the row **names a
  subject and no required value** (section 6.6). It is unlinked for the verdict of stage
  `xref-failure` and counted there and by stage `failure-modes`, and it stops neither
  `machinery gen` nor `machinery adopt` (sections 7 and 17.2): it is owed and generated like
  any row. A `required` that is given and is none of the four stays an ERROR, with the message
  of section 5.4. `no_subject: "<reason>"` in place of `subject` declares the rows subject-less
  (section 5.4). Every row an entry names takes its values. The value rules are checked once
  per entry.
- **`required` beside a required column.** With `columns.required`, an entry may leave
  `required` out: each row keeps the value of its own cell. A row whose cell is empty, or whose
  table has no such column, then has no value and is no finding: it names a subject and no
  required value, as above. One finding per entry, each naming the rows as the entry writes them
  (the first five and `and <n> more`): a row whose cell holds something that is no value has the
  finding of its cell and no second one;
  an entry whose `required` is another value than a row's cell is
  `ERROR  <links file>: <entry>: required <value> differs from the required cell of <ids>; keep one`,
  and the row keeps the value of its cell; and a cell that says `refuse` for a row whose entry
  is not `kind: abuse` is
  `ERROR  <links file>: <entry>: required: refuse (the required cell of <ids>) is only valid with kind: abuse`.
- **Messages** are prefixed `<links file>: <entry>`, where `<entry>` is the `id`, or the items of
  `ids` as written, joined by `, ` (the first three and `, ...` when there are more). A
  malformed item is `<item> is not an id or a range <first>..<last>`,
  `the ends of the range <item> must end in a number`,
  `the ends of the range <item> do not share a prefix`, `the range <item> runs backwards` or
  `the range <item> names <n> ids; a range holds at most 10000`, and the entry is then not read.
  An id an entry names twice is `<id> is listed twice`.
- **Every id must be a row.** For `id`:
  `ERROR  <links file>: <id>: no failure-mode row of <file> has this id`. For `ids`, one finding
  per entry: `no failure-mode row of <file> has the id <id>`, or
  `has the ids <id>, <id>` with the first five and `and <n> more`. The rows that do exist take the
  entry's values.
- **Overlap.** A row two entries name is one finding per pair of entries, naming both:
  `ERROR  <links file>: links <n> (<entry>) and <m> (<entry>) overlap: <ids>` (the first five ids
  as the later entry writes them, and `and <n> more`). The row keeps the values of the first.
- An unknown key is an ERROR.

Every imported row is then one of three:

- A **linked** row: an entry gives it a subject of one of the forms of section 5.4 and a
  `required` value, and the subject resolves in the design (section 6.6). It is a failure mode
  like a native one, and takes part in every rule of stage `xref-failure`.
- A **subject-less** row (an entry declares it `no_subject`, with the reason) is an obligation
  like any failure mode and takes part in no subject rule.
- An **unlinked** row is every other one: no entry names it, or the entry that names it gives
  no subject, a malformed one, one that resolves to nothing, or no valid `required` value
  (neither the entry nor a cell of the row gives one). A row that names a subject which
  resolves and only lacks the value is counted on its own by stages `failure-modes` and
  `xref-failure` (sections 6.3 and 6.6), and is unlinked all the same. An
  entry is never a link by being there. It is an obligation all the same: `fm:<id>` is owed,
  claimed and traced like any failure mode. It names nothing, so it takes part in no subject
  rule of `xref-failure`: it neither resolves nor covers an invoke, a dependency, an action, a
  failure state or a failure event. While one row is unlinked, stage `xref-failure` is not
  demonstrated (section 6.6).

### 5.5 Risk file

`risk.yaml` assigns risk tiers to areas of the design. See section 23.

## 6. Lint stages

### 6.1 `model`

For each configured model file:

- Run `<modelith> --version`. The output is `modelith version X.Y.Z` (an optional leading `v` on
  the number is accepted). Older than 0.6.0, or not runnable, makes the stage
  `not demonstrated: modelith 0.6.0 or newer not found (go install
  github.com/stacklok/modelith/cmd/modelith@v0.6.0)`.
- Run `<modelith> lint --format json <file>`. Each Modelith error becomes an ERROR, each warning a
  warn, both prefixed with the model path.
- Compare the committed `<name>.modelith.md` with a fresh rendering (section 5.1). Missing or
  different is a DRIFT.
- Extract ids (section 4). A duplicate id is an ERROR.

`checked:` counts models, entities, actions, invariants and scenarios.

### 6.2 `machines`

Lints every `*.machine.json` in the machines directory, or every imported machine
(section 5.2.1), with the rules in `docs/machine-format.md`: subset conformance, name patterns, strict target resolution, declared
delays, reachability from the initial state and from every entry state (`_entries`), dead ends,
absorbing states unless declared `_terminal`, every invoke has `onError` and a timeout, event
completeness on resting states (`_ignores`), guard-false dispositions (`_refusal`, `_exhaustive`),
fail-closed tags, no state tagged both `failure` and `accepting`, and the shape of `_requires`,
`_entries`, `_terminal`, `_external_events`, `_failure_events`, `_aliases` (an object that maps
an event some state handles or ignores to a non-empty action name, no event of it also in
`_external_events`; one may be in `_failure_events`, section 6.4), `_refusal_events` (an object
that maps an event some state handles or ignores to an object with a non-empty `action` and a
non-empty `reason`, no event of it
also in `_external_events`, in `_failure_events` or in `_aliases`), `_not_failures` (an object
that maps an event some
state handles or ignores, or the dotted path of a state, to a non-empty reason; none of them in
`_failure_events` or a state tagged `failure`, and none without a hint to dismiss: a name with
no failure word in it, an action of the machine's lifecycle entity, or an event in
`_refusal_events`), `_serves` (a non-empty list of
distinct `entity:<Name>` and `act:<Entity>.<action>` ids; whether they exist is section 6.4) and
`_aspect` (a short name matching `^[A-Za-z][A-Za-z0-9_-]*$`; which machines may declare it is
section 6.4). Two machines whose id tags
collide (section 7.2), or that share a machine `id`, are an ERROR.

What the tool prints from an annotation is one line. An action of `_aliases`, an action or a
reason of `_refusal_events` and a reason of `_not_failures` that holds a line break or another
control character is
`ERROR  <file>: _aliases['<event>'] must be one line without control characters` (and
`_refusal_events['<event>'].action`, `.reason`, `_not_failures['<name>']`), and such an entry is
in effect nowhere: no stage prints it.

The findings of `_refusal_events`:
`ERROR  <file>: _refusal_events must be an object that maps an event name to {action, reason}: the model action it refuses and why it is a refusal and no failure`,
`ERROR  <file>: _refusal_events['<event>'] must be an object with action and reason, and nothing else`,
`ERROR  <file>: _refusal_events['<event>'].action must be a non-empty action name`,
`ERROR  <file>: _refusal_events['<event>'].reason must be a non-empty reason: say why the event is a refusal and no failure`,
`ERROR  <file>: _refusal_events['<event>'].action must be one line without control characters`
(and the same for `.reason`),
`ERROR  <file>: _refusal_events names '<event>' but no state handles or ignores it; the marker is stale (remove it, or wire the event)`,
and, for an event another annotation also holds,
`ERROR  <file>: event '<event>' is in both _refusal_events and <other> (keep one)`, where `<other>` is
`_external_events; the machine writes a refusal itself, so it does not arrive from outside`,
`_failure_events; a refusal is declared, not a failure` or
`_aliases; an event refuses an action or is one under another name`.

Name hints are warnings and never fail the stage. A state whose name contains, case-insensitively,
`fail`, `error`, `abort`, `crash` or `timeout` and that is not tagged `failure` prints
`warn   <file>: state <S> looks like a failure state and is not tagged failure` (`<S>` is the
dotted path). An event the machine handles or ignores whose name contains one of those words, and
that is neither in `_failure_events`, nor in `_refusal_events`, nor an action of the machine's
lifecycle entity, prints
`warn   <file>: event <e> looks like a failure event and is not in _failure_events`. An event in
`_external_events` or with an entry in `_aliases` is still hinted: what arrives from outside, or
under another name, may report a failure. A refusal event is not: a refusal is declared, not a
failure. A hint is dismissed by name, with the reason, in the
root `_not_failures`: the event name, or the dotted path of the state. An entry there that
dismisses no hint is an ERROR, as a stale marker is:
`ERROR  <file>: _not_failures names '<name>', which does not look like a failure (no fail, error, abort, crash or timeout in the name), so there is no hint to dismiss (remove it)`
(for a state, the last segment of its path is its name), and
`ERROR  <file>: _not_failures names '<name>', which is an action of <E> and is never hinted, so there is no hint to dismiss (remove it)`,
and
`ERROR  <file>: _not_failures names '<name>', which is in _refusal_events and is never hinted, so there is no hint to dismiss (remove it)`.

`checked:` counts machines, states and transitions. Not applicable when neither `design.machines`
nor `import.machines` is configured. With `import.machines` the translation findings of section
5.2.1 are findings of this stage, and so are the rules of its register (`register-undeclared`,
`register-unobserved`), whose entries and observed transitions `checked:` then counts.

### 6.3 `failure-modes`

Parses the file against section 5.4, and the catalog and its links file against section 5.4.1.
Unique ids, valid `kind` and `required` values, well-formed `subject`. `checked:` counts failure
modes, native and imported, and, of those, abuse cases, and, when `import.failure_modes` is
configured, the imported rows. Then it prints the account of the failure modes of section 6.6,
the one stage `xref-failure` prints: the same definition of linked, unlinked, naming a subject
and no required value, and subject-less, computed once per command, so that both stages print
the same numbers for the same failure modes. As there, `linked` and `unlinked` are printed when
`import.failure_modes` is configured or a failure mode is unaccounted for, and the two others
when they count one:
`checked: 9 failure modes, 1 abuse cases, 7 imported, 2 linked, 4 unlinked, 1 name a subject and no required value, 3 subject-less`.
The account decides no outcome of this stage: an unlinked failure mode is the verdict of stage
`xref-failure`, and a links entry that names a subject and no required value is no finding here
(section 5.4.1). Not applicable when neither `design.failure_modes` nor `import.failure_modes` is
configured.

### 6.4 `xref-model-machine`

For every machine, and every entity of the models:

| Rule | Check |
|---|---|
| `machine-binding` | The machine binds to the model (section 5.2): it is the lifecycle machine of an existing entity, or an operational machine whose `_serves` names only entities and actions of the model. An entity has one lifecycle machine, or several that each declare a different `_aspect`. |
| `state-enum` | Lifecycle machine of entity E: let D be its domain states. Among the enums used as the `type` of an attribute of E, exactly one must have a value-name set equal to D. Otherwise ERROR, naming the missing and extra names against the candidate enum with the largest overlap (ties broken by enum name). When no candidate has a value in D, the finding says that E has no state enum for the machine and names no enum. An entity with no enum-typed attribute at all is one finding for the entity, whatever the number of its lifecycle machines. With aspects, each aspect machine is matched against an enum of its own: an enum that exactly matches one machine is that machine's, and no enum is the state enum of two machines (a machine that matches only an enum another one holds is a finding, and the enums other aspects hold are left out of its candidates). |
| `event-action` | Lifecycle machine of E: every event name under any `on` must be the name of an action of E, be an alias of one (the root `_aliases` maps it to an action of E), be a declared refusal of one (the root `_refusal_events` maps it to an action of E that has an `on` event of its own name in the machine), or be listed in the root `_external_events` or `_failure_events` (a failure event arrives from outside). An aliased event may also be listed in `_failure_events`, when the action it stands for is itself the failing outcome (model action `fail`, machine events `send_failed` and `verify_failed`): it is then that action for this rule and for `action-event`, and a failure event for section 6.6; every other rule about an alias holds for it. An action is a caller's command: an event in either list that is the name of an action of E is a finding, and so is an alias for an event that is itself an action of E. Several events may be aliases of one action, but an action that has an `on` event of its own name in a lifecycle machine of E is the action of no alias: it would absorb any other event. When E declares no action, the events that are not listed are one finding for the entity, not one per event: the one missing declaration causes them all. Operational machine that binds to the model (no `machine-binding` finding): every event name under any `on` must be the name of an action of an entity it serves, the action of an `act:` id it serves, an alias of one of those actions, a declared refusal, or be listed in `_external_events` or `_failure_events`. An alias names an action the machine serves, one that has no `on` event of its own name in the machine. A refusal names an action of an entity behind `_serves`, the entity of an `act:` id too, that has an `on` event of its own name in the machine. A listed, aliased or refusal event is held to more: it is a finding when it is the name of any action of any entity behind `_serves`, the entity of an `act:` id too, served or not, so no list, no alias and no refusal hides a command of an entity the machine works for. When it serves only entities that declare no action, the events that are not listed are one finding for the machine: nothing was compared with the model. |
| `action-event` | Every action of an entity that has a lifecycle machine must appear as an `on` event somewhere in that machine, under its own name or under an alias (`_aliases`) that holds, or, with aspects, in at least one of them. |
| `entity-machine` | Every entity that declares at least one action has a lifecycle machine. An entity that has no state of its own is waived, with the reason. |

The findings of `machine-binding`:

- `machine-binding: <id>: not a lifecycle machine of a model entity and not operational; set _lifecycle_of or name the file after an entity, or set _role: operational and name what it serves in _serves`
- `machine-binding: <id>: _lifecycle_of names entity <E>, which is not in the model`
- `machine-binding: <id>: _lifecycle_of and _role: operational are both set; a machine is one or the other`
- `machine-binding: <id>: operational and serves nothing in the model; bind it to an entity as its lifecycle (_lifecycle_of), or name what it serves in _serves (entity:<Name> or act:<Entity>.<action>)`:
  `_serves` is absent, empty or not a list.
- `machine-binding: <id>: _serves names <x>, which is not an entity or an action of the model`, once
  per such entry.
- `machine-binding: <id>: _serves is set on the lifecycle machine of <E>; a lifecycle machine serves its entity: remove _serves, or set _role: operational`
- `machine-binding: <id>: a second lifecycle machine of <E>, beside <first>; several lifecycle machines of one entity are aspects of its lifecycle: set _aspect on each, a short name unique for <E>`:
  an entity has more than one lifecycle machine and not every one declares `_aspect`. Every
  machine after the first, in machine order, is reported, naming the first.
- `machine-binding: <id>: _aspect <name> is also the aspect of <other>; the aspects of <E> need different names`
- `machine-binding: <id>: _aspect is set on an operational machine; an aspect is part of an entity's lifecycle: remove _aspect, or bind the machine with _lifecycle_of`

and of `entity-machine`:
`entity-machine: entity:<E>: declares <n> action(s) and no machine is its lifecycle; write its lifecycle machine (_lifecycle_of: <E>), or waive the rule with the reason it has no state`.

The findings of `state-enum` and `event-action` that stand for a whole entity or machine:

- `state-enum: entity:<E>: has no state enum (no attribute of <E> has an enum type); the domain states of <id> (<states>) need one`;
- `state-enum: <id>: entity <E> has no state enum for this machine: no enum of its attributes (<enums>) has a value among the domain states (<states>); add an enum attribute with these values`;
- `state-enum: <id>: enum <X> is already the state enum of <other>; each aspect of <E> needs an enum attribute of its own`, and
  `state-enum: <id>: entity <E> has no state enum for this machine: every enum of its attributes is the state enum of another aspect; add an enum attribute with the domain states (<states>)`;
- `event-action: entity:<E>: declares no actions, so none of the <n> event(s) of <id> is one of its actions (<events>); add the actions to the model, or list the events in _external_events (or in _failure_events when one reports a failure)`;
- `event-action: <id>.<event>: event is not an action of what the machine serves (<serves>); serve the action (act:<Entity>.<action>), or list the event in _external_events (or in _failure_events when it reports a failure)`, for an operational machine, `<serves>` its `_serves` as written;
- `event-action: <id>: serves only entities that declare no actions (<entities>), so none of its <n> event(s) was compared with the model (<events>); add the actions to the model, serve the actions the machine runs (act:<Entity>.<action>), or list the events in _external_events (or in _failure_events when one reports a failure)`.

The findings of `event-action` about an action that is listed or aliased, where `<of>` is `<E>`
for a lifecycle machine and `what the machine serves (<serves>)` for an operational one:

- `event-action: <id>.<event>: listed in <list> and an action of <E>; an action is a caller's command, not an event from outside: remove it from <list> in <file>`,
  `<list>` being `_external_events` or `_failure_events` and `<E>` the entity that declares the
  action. When an entry of `import.machines.annotate` listed the event (section 5.2.1), the
  finding ends instead with
  `import.machines.annotate <key> lists it for <file> (<list> pattern <pattern>); narrow the pattern or remove it`,
  `<list>` there being `external_events` or `failure_events` and the pattern the first that
  matched;
- `event-action: <id>.<event>: _aliases gives the event the action <action>, and <event> is itself an action of <E>; an action is not hidden behind another: <where>`;
- `event-action: <id>.<event>: _aliases names action <action>, which is not an action of <E>; name an action of <E>, or <where>`, and for an operational machine
  `event-action: <id>.<event>: _aliases names action <action>, which is not an action of what the machine serves (<serves>); serve the action (act:<Entity>.<action>), or <where>`;
- `event-action: <id>.<event>: _aliases gives the event the action <action>, and no state handles <event> under on; an event that is only ignored stands for no action: <where>`:
  rule `action-event` asks for an `on` event, so an alias for an event that states only ignore
  would satisfy nothing, and it is neither printed nor counted;
- `event-action: <id>.<event>: _aliases gives the event the action <action>, and <action> has an event of its own name in <machine id>; an action has its own event or stands behind aliases, not both: <where>`.

`<where>` is `remove it from _aliases in <file>`, or, when an entry of
`import.machines.annotate` gave the alias,
`remove the alias from import.machines.annotate <key> (for <file>)`.

An alias that does not hold makes its event neither an action nor a second finding: the action it
should have been is then reported by `action-event`. These findings are about every entry of the
lists and of `_aliases`, whether a state handles the event under `on` or only ignores it.

A **refusal event** is the event by which a machine itself writes that it refused a caller's
command: the machine handles `cancel` and, where it does not accept it, records
`cancel_refused`. The caller sent the command, so the event is not external, and the action
has its own event, so the event is no alias. `_refusal_events` declares it, and a declaration
holds when the event is no action, the action it names is one of the machine's entity (for an
operational machine, of an entity behind what it serves), that action has an `on` event of its
own name in the same machine, and no transition on the event moves the machine into a state
tagged `failure` that it was not in already.
The findings of `event-action` about one that does not hold:

- `event-action: <id>.<event>: _refusal_events gives the event the action <action>, and <event> is itself an action of <E>; an action is a caller's command, not the refusal of one: <where>`;
- `event-action: <id>.<event>: _refusal_events names action <action>, which is not an action of <E>; name an action of <E>, or <where>`, and for an operational machine
  `event-action: <id>.<event>: _refusal_events names action <action>, which is not an action of an entity behind what the machine serves (<serves>); name one, or <where>`;
- `event-action: <id>.<event>: _refusal_events gives the event the action <action>, and <action> has no event of its own name in <machine id>; a refusal is of a command the machine handles: <where>`;
- `event-action: <id>.<event>: _refusal_events declares the event, and on it the machine can come to <state>, a state tagged failure; an event that records a failure is a failure event and asks for a failure mode: list it in _failure_events instead, or <where>`:
  `<state>` is the first state tagged `failure`, in document order, that an `on` transition of
  the event moves the machine into. The states it enters are followed until it comes to rest: the target and the
  states the target is inside; for a compound target, its initial child, and that child's, down
  to a leaf (a failure state beside the initial child is not entered and is no ground); and
  from the leaf reached, the target of every `always` branch of it or of a state it is inside,
  and of the `onDone` of its parent when the leaf is final, each followed the same way, guarded
  or not. A delay (`after`), an invoke's outcome and a later event are not followed: they are
  another cause. A state tagged `failure` among those is no ground when the state whose `on`
  holds the transition is that state or a state inside it: the machine was in it already, and
  the event leaves it there (`failed: {tags: [failure], on: {cancel_refused: failed}}`, or a
  transition between two states inside it that are not tagged `failure` themselves). Every
  other one is: a move from a state that is not tagged `failure`, and a move from one state
  tagged `failure` into a different one, directly or by a branch followed after the event,
  because the event then records a new failure. The rule is held per transition: a transition
  that stays in its failure state does not excuse another one on the same event that moves into
  it from elsewhere.

`<where>` is `remove it from _refusal_events in <file>`, or
`remove the refusal from import.machines.annotate <key> (for <file>)`. As for an alias, a
declaration that does not hold makes its event no second finding. A refusal event satisfies
nothing else: rule `action-event` still asks for the action's own event.

**What a declaration takes away, and why that is sound.** Declaring an event a refusal removes
what a failure event would owe: its name hint (section 6.2) and the failure mode that rule
`failure-event-mode` asks for (section 6.6); a failure mode may still name it. That is sound
for a refusal and for nothing else: the caller's command was refused and nothing failed, so
there is no failure whose required behavior could be missing, and the refused command is
already an obligation of its own (the action, and the transition and ignore cases of its
event). The tool cannot read that meaning from a name, so the declaration is held to what can
be checked and is otherwise put in front of a reviewer:

- it carries a reason, non-blank, for every event (section 6.2);
- the event is in neither `_failure_events` nor `_external_events` and has no alias, so a
  declared failure is never declared away (section 6.2);
- the event moves the machine into no state tagged `failure` that it was not in already,
  directly or through eventless transitions (the finding above), so an event the machine itself
  records as a failure stays one, whatever it is called;
- the reason is one line, so the printed line is one declaration and nothing else;
- every declaration that holds is printed with its reason and counted, on every run.

These lines and the reasons in them are the guard against a failure renamed a refusal: no rule
can tell a true refusal such as `payment_refused` from a `provider_timeout` that changes no state
and is
declared a refusal. A review of a design reads them.

No alias and no refusal event is silent. Every alias that holds is one line of the stage, a
warning that never fails it, `warn   alias: <machine id>: <event> stands for <action>`, and so
is every refusal event that holds, `warn   refusal: <machine id>: <event> refuses <action> (<reason>)`;
both are counted in `checked:` (below).

`<states>` and `<events>` are sorted and comma-separated.

A machine is **bound** when it has no `machine-binding` finding, waived or not; a waiver accepts
an unbound machine, it does not bind it. When no machine is the lifecycle machine of an existing
entity, the stage compared no state, event or action, and its result is
`not demonstrated: no machine is bound to a model entity` (`nothing to check` when there is no
machine file at all), never passed; with findings it is failed (section 3.2).

`checked:` counts machines, lifecycle machines (of an existing entity), operational machines,
domain states, events (of the lifecycle machines and of the operational machines that bind),
actions (of the entities that have a lifecycle machine, each once) and entities with actions:
`checked: 2 machines, 1 lifecycle machines, 1 operational machines, 3 domain states, 2 events, 2 actions, 1 entities with actions`.
With at least one alias in effect it adds their number, and then that of the refusal events in
effect: `..., 1 entities with actions, 2 aliases, 1 refusal events`.

### 6.5 `xref-contract`

For every configured contract:

| Rule | Check |
|---|---|
| `operation-id` | Every OpenAPI operation has an `operationId`; every bound rpc has a name. |
| `operation-action` | Every action named by `x-machinery-action` (or `machinery:action`) exists in the model. |
| `action-operation` | Every model action is named by at least one operation. |

Operations without a binding are counted and are not an error (reads and queries).
`checked:` counts contracts, operations, bound operations and actions. Not applicable when
`contracts:` is absent.

### 6.6 `xref-failure`

| Rule | Check |
|---|---|
| `subject-resolves` | Every failure-mode subject names an existing invoke, a configured dependency, a model action, a model entity, a state of the named machine that is tagged `failure`, an event in the named machine's `_failure_events` or `_refusal_events`, or, for `machine:<id>`, a machine of the design, bound to the model or not. |
| `invoke-failure-mode` | Every `invoke` `src` of every machine is the subject of at least one failure mode. |
| `dependency-failure-mode` | Every configured dependency is the subject of at least one failure mode. |
| `tiered-action-abuse` | Every action that belongs to a consequential, critical or vital area (section 23) is the subject of at least one failure mode of kind `abuse`. |
| `failure-state-mode` | Every state tagged `failure` of every machine is the subject (`state:<machine id>.<State>`) of at least one failure mode. |
| `failure-event-mode` | Every `_failure_events` entry of every machine is the subject (`event:<machine id>.<event>`) of at least one failure mode. |

The stage applies when `design.failure_modes` or `import.failure_modes` is configured, or when
any machine declares a failure state or a failure event (section 5.2). An imported row that is
not linked (section 5.4.1) and a failure mode declared subject-less (`no_subject`, section 5.4)
are counted among the failure modes and take part in no rule. A `machine:` subject takes part in
`subject-resolves` alone: naming a machine covers none of its invokes, failure states and
failure events, so `invoke-failure-mode`, `failure-state-mode` and `failure-event-mode` still
ask for a failure mode per invoke, state and event. An `event:` subject that names a refusal
event resolves, and no rule asks for one: a refusal is declared, not a failure. An event that
is neither is
`subject-resolves: event:<machine id>.<event>: failure mode <id>: <event> is not in the _failure_events of machine <machine id> (nor in its _refusal_events)`.
A `system:` subject takes part in none of the rules of the table. Its name must be one of
`systems` (section 2):
`ERROR  unknown-system: system:<name>: failure mode <id>: <name> is not a configured system; declare it under systems in machinery.yaml (no waiver covers an unknown system)`.
`unknown-system` is not a rule a waiver may name (section 6.7), and a waiver of
`subject-resolves` for a `system:` subject matches no finding and is stale. A
finding that asks for a failure mode says where to declare it: in the file of
`design.failure_modes` when there is one, else in the links file of `import.failure_modes`
(`a links file (import.failure_modes.links)` when none is configured). `checked:` counts
failure modes, invokes,
dependencies, tiered actions, failure states and failure events. When `import.failure_modes` is
configured, or a failure mode is unaccounted for (below), it counts, right after the failure
modes, those that are linked (`linked`, native ones included) and those unaccounted for
(`unlinked`); when at least one failure mode is declared subject-less it counts those next:
`checked: 48 failure modes, 30 linked, 4 unlinked, 14 subject-less, 9 invokes, 2 dependencies, 0 tiered actions, 4 failure states, 1 failure events`.
When at least one of the unlinked ones names a subject that resolves and only lacks a valid
`required` value (below), their number follows `unlinked`:
`..., 4 unlinked, 3 name a subject and no required value, ...`. Stage `failure-modes` prints the
same counts from the same account (section 6.3). `machinery assess` carries the line as it is
(section 18).
With no failure-mode file
configured only `failure-state-mode` and `failure-event-mode` run: each failure state and failure
event is a finding that says to configure `design.failure_modes`, and `checked:` counts failure
states and failure events only. Not applicable when there is neither a failure-mode file nor a
failure state or failure event.

A failure mode, native or imported, is **linked** when all of this holds: its subject has one of
the forms of section 5.4, its `required` value is one of the four, it has no `no_subject` key,
and the subject resolves: rule `subject-resolves` has no finding for it, waived or not, and a
`system:` subject names a configured system. It is **subject-less** when `no_subject` gives a
reason and there is no `subject`. Every other failure mode is **unaccounted for** and counts as
`unlinked`: an imported row no links entry names (section 5.4.1); a subject that is missing,
empty or malformed; a subject that resolves to nothing; a missing or invalid `required`; a
`no_subject` with a blank reason or beside a `subject`. Only a linked failure mode covers an
invoke, a dependency, an action, a failure state or a failure event. Stage `failure-modes` has
the ERROR of a malformed entry; this stage counts it as `unlinked` all the same, so that it
never passes over one, run alone, in `check`, in `assess` or in a rehearsal. A failure mode
that **names a subject and no required value** is an unaccounted one that somebody did look at:
its subject has one of the forms of section 5.4 and resolves, it has no `no_subject` key, and its
`required` is missing or not one of the four. It is counted on its own, as a part of
`unlinked`, so that the rows that only wait for the value are told apart from the rows nobody
linked. It stays unlinked for every rule and for the result. In the links file a missing value
is no finding of stage `failure-modes` and stops neither generation nor adoption (section
5.4.1); a value that is given and is none of the four is an ERROR there, and in the failure-mode
file a missing one is too (section 5.4). A subject of one of the forms of section 5.4 that names
nothing is a `subject-resolves` finding (or `unknown-system`) whatever its `required` value:
such a failure mode is unlinked, and is not counted as naming a subject. While one failure mode is unaccounted for,
the stage did
not compare every failure mode with the design: one linked row among many is no pass. Its result
is
`not demonstrated: <n> of <m> failure modes name no subject; link each in <links file> or declare it there with no_subject and a reason`,
where `<n>` counts those unaccounted for, `<m>` every failure mode, and `<links file>` is
`a links file (import.failure_modes.links)` when none is configured; when none of them is an
imported row the middle part is `give each a subject in <failure-mode file>`. When some of them
name a subject and no required value the result ends
`; <k> of them name a subject and no required value: add required to each`. When none is
unaccounted for and
still no failure mode names a subject (every one is declared subject-less), it compared none:
`not demonstrated: no failure mode names a subject` (`nothing to check` when there is nothing at
all to compare). Neither is ever passed, waivers or not; with findings the stage is failed
(section 3.2). A waiver does not link a row: a waived `subject-resolves` finding prints as a
warning and its failure mode stays unlinked.

### 6.7 Waivers

A cross-reference finding (sections 6.4 to 6.6) can be waived in `machinery.yaml`:

```yaml
waivers:
  - {rule: action-event, id: "act:Order.place", reason: "creation enters the initial state"}
```

`rule` is a rule name from the tables above. `id` is the subject the finding names: an `act:` id;
`<machine id>.<event>` for `event-action` (`entity:<Name>` for an entity that declares no
action, the machine id for an operational machine that serves only such entities); the machine id for `machine-binding` and `state-enum` (`entity:<Name>` for an entity
with no enum-typed attribute); `entity:<Name>` for `entity-machine`;
`<METHOD> <path>` or `RPC <Service>/<Rpc>` for `operation-id`; the failure-mode subject for
`subject-resolves`; `invoke:<machine id>.<src>`, `dependency:<name>`, `state:<machine id>.<State>`
or `event:<machine id>.<event>` for the failure rules. A
finding prints as `<rule>: <subject>: <what is wrong>`. `reason` is
required. A waived finding prints as `warn   waived: <message> (<reason>)`. A waiver that matches
no finding is an ERROR `stale waiver`, so the list only shrinks. Only cross-reference rules can be
waived.

### 6.8 `xstate`

Applies when `tools.xstate` is true. The stage runs an embedded script with `tools.node` against
the real `xstate` package found at `<tools dir>/xstate/node_modules/xstate` (version 5.33.4 or
newer, below 6). For each machine it calls `createMachine` with the machine file as written, and
for every transition case of kind `on` in the generated table (section 7.3) it asks XState for
the next snapshot and the actions it would execute from the case's source state (the initial leaf
of a compound source) with the case's guard true and every other guard false, then compares:

| Rule | Check |
|---|---|
| `target` | The leaf XState enters is the case's `target`, or its initial leaf when the target is compound, or the source's initial leaf for an internal transition. XState then settles in the same step: it takes an eventless (`always`) row of the active leaf or of an ancestor, innermost first, and the state `onDone` row of a compound state whose final child was entered. The comparison follows the same rows in the table, taking the first row of a state whose guard is empty or is the case's guard, and uses the leaf where they stop. |
| `actions` | The named actions XState would execute, in order and deduplicated, equal the case's `effects` followed by the `effects` of the rows the `target` rule followed, deduplicated. Built-in `xstate.*` actions (spawning an invoke, scheduling and cancelling an `after`) are left out. |
| `initial` | XState's initial leaf is the table's `initial`, followed down its initial children and settled as in the `target` rule with every guard false. |

A disagreement is `ERROR  xstate: <machine id> <case id>: <what differs>`; for the `initial` rule
the case id is the word `initial`, and a machine XState refuses to load is
`ERROR  xstate: <machine id>: XState rejects the machine: <error>`. Node or the package
missing, or a version outside the range, is `not demonstrated` with the reason. `checked:` counts
machines and cases compared. Not applicable when `tools.xstate` is false or absent.

## 7. Generation

`machinery gen` writes, under the gen directory:

- `obligations.json` (section 7.1);
- `projection.json` (section 22.3), only when `checkers:` or `design.facts` is configured;
- `tables/<machine id>.table.json` for each machine (section 7.3);
- `tables/<machine id>.table.md`, a readable rendering of the same table;
- `imported/machines/<machine id>.machine.json` for each imported machine (section 5.2.1): the
  translated machine, so that people and XState tooling can read it. It is generated like every
  file here; the source file stays the one to change;
- `vital/<area>.json` for each vital area of the risk file: the property record (section 23.4);

and beside each model its `<name>.modelith.md`.

All generated files are committed and never edited by hand. JSON is written with two-space
indentation, keys in the order given in this document, and a trailing newline. Writes are atomic
(temporary file, then rename). `machinery gen` refuses to write (ERROR) when the `model`,
`machines` or `failure-modes` stage has an ERROR. A links entry that names a subject and no
required value is none (section 5.4.1): its rows are generated like any failure mode.

`machinery gen --check` regenerates in memory and reports `DRIFT  <path>: stale`, `DRIFT  <path>:
missing`, and `ERROR  <path>: unexpected file in gen directory`, the translated machines
included. A project with no design source configured (under `design` or `import`) and an empty
gen directory is `not applicable: no design sources configured`.

### 7.1 `obligations.json`

```json
{
  "version": 1,
  "obligations": [
    {"id": "inv:paid-before-shipped", "kind": "inv", "source": "design/model/shop.modelith.yaml", "title": "An Order reaches Shipped only from Paid."},
    {"id": "tc:ORDE-2e883d", "kind": "tc", "source": "design/machines/Order.machine.json", "title": "Pending on:pay [hasPaymentMethod] -> paying", "effects": ["stampAttempt"]}
  ],
  "served": [
    {"id": "act:Order.pay", "source": "design/model/shop.modelith.yaml"}
  ]
}
```

- `obligations` holds every `inv`, `sc`, `tc`, `fm`, `fv`, `cx` and `ck` id, sorted by id in byte order.
- `title` is the invariant statement, the scenario name, the case title (section 7.2), the
  failure-mode `failure` text, `Formal model of <area>: <why>` for `fv`, and
  `Replay <violated property> (<spec> run <run>)` for `cx`.
- `source` is the file the id comes from: the model, the machine (for an imported machine its
  source file), the failure-mode file or the imported catalog, the risk file for `fv`, the trace
  file for `cx`, the evidence file for `ck`.
- `effects` is present only for `tc` and may be empty.
- `served` holds every `act` id, sorted by id.
- A proposed or retired id (section 4.1) is in neither list.

### 7.2 Transition cases

Each machine yields cases. One case is one thing a test must observe on the running system.

| Case kind | One per | Expected observation |
|---|---|---|
| `transition` | Each transition branch (`on`, `after`, `always`, invoke `onDone` and `onError`, state `onDone`) | The machine moves to `target` (or stays, for an internal transition) and every effect happens. |
| `ignore` | Each `_ignores` entry of each state | The event is refused or absorbed as documented; the state does not change; no effect happens. |
| `refusal` | Each `_refusal` entry of each state | With every guard of the handler false, the state does not change; no effect happens. |
| `clause` | Each declared clause of the guard of each guarded transition | With only that clause false and every other clause true, the guarded branch does not fire. |

**Id.** `tc:<TAG>-<hex>` where:

- `TAG` is the root `_oracle_tag` when present, otherwise the machine `id` uppercased and cut to
  four characters;
- `hex` is the first 6 hex characters of `sha256("<TAG>|<source>|<trigger>|<guard>")`;
- `source` is the dotted state path;
- `trigger` is `on:<event>`, `after:<delay>`, `always`, `onDone:<src>`, `onError:<src>` or
  `stateDone` for transitions, `ignore:<event>` for ignore cases, `refusal:<handler>` for refusal
  cases;
- `guard` is the guard name, or empty when there is none. For a clause case it is
  `<guard>!<clause>` and `source` and `trigger` are those of the guarded transition.

When two different cases of one machine share the 6-character prefix, extend both by two
characters at a time until they differ. Two machines producing the same `TAG` is an ERROR in the
`machines` stage: set `_oracle_tag` on one of them.

This is the v1 stable-id scheme, unchanged, so existing ids survive.

**Title.** `<source> <trigger> [<guard>] -> <target>`, with `[<guard>]` omitted when there is no
guard and `-> (internal)` when there is no target. Ignore and refusal cases use
`<source> <trigger> -> (unchanged)`. Clause cases use
`<source> <trigger> [<guard>!<clause>] -> (branch does not fire)`.

**Guard clauses.** A guard that is a conjunction hides bugs behind its first false clause: a test
that makes the whole guard false proves one clause at most. The machine root may declare the
clauses of a guard:

```json
"_guards": {"canWinDeal": ["callerMayWriteDeal", "amountIsPositive"]}
```

The key is a guard the machine uses; the value lists at least two clause names, each matching
`^[a-z][A-Za-z0-9]*$`, without duplicates. Every guarded transition that uses the guard then
yields one clause case per clause. A test proves a clause case by arranging exactly that clause
to be false and every other clause true, firing the trigger, and observing that the guarded
branch's case did not fire. Declare `_guards` for every guard that checks more than one thing.

**Effects.** Clause cases have no effects. For ignore and refusal cases the list is empty. For a
transition case the effects are the named actions XState 5 executes when the transition fires,
taking the initial leaf of the source as the active state (for a state `onDone`, the source's first
final child):

1. the exit actions of the states left, innermost first;
2. the transition's own actions;
3. the entry actions of the states entered, outermost first, down through the initial children of
   the target;
4. when the target leaf is a top-level final state, its exit actions, because the machine stops.

The states left and entered are the active states and the target's chain below the transition's
domain. The domain is the source itself when the target is the source or one of its descendants
(XState does not exit and re-enter the source; `reenter` is outside the subset); otherwise it is
the nearest ancestor of the target that contains the source, or the machine root. So a transition
with a target exits the active leaf and every ancestor up to the domain and enters the target's
ancestors below the domain, the target and its initial children; a self-target on an atomic state
exits and enters nothing; a self-target on a compound state exits and re-enters only its children;
an internal transition (no target) runs only its own actions. Names are deduplicated, first
occurrence wins. Effects stop at the row: the `always` and `onDone` rows XState may take next are
their own cases.

### 7.3 Runtime tables

`tables/<machine id>.table.json` is what a service embeds to drive the machine. The service looks
transitions up in this table; it does not hard-code them.

```json
{
  "version": 1,
  "machine": "order",
  "tag": "ORDE",
  "source": "design/machines/Order.machine.json",
  "source_sha256": "<sha256 of the machine file bytes>",
  "initial": "Pending",
  "states": [
    {"path": "Pending", "kind": "atomic", "resting": true, "entry": [], "exit": [], "invoke": []}
  ],
  "transitions": [
    {"id": "tc:ORDE-2e883d", "source": "Pending", "kind": "on", "event": "pay", "guard": "hasPaymentMethod", "target": "paying", "actions": ["stampAttempt"], "effects": ["stampAttempt"]}
  ],
  "ignores": [
    {"id": "tc:ORDE-2fa2af", "state": "Pending", "event": "ship", "reason": "an unpaid order cannot ship; the API refuses with ORDER_NOT_PAID"}
  ],
  "refusals": [],
  "clauses": [
    {"id": "tc:DEAL-346352", "transition": "tc:DEAL-d09c4d", "guard": "canWinDeal", "clause": "callerMayWriteDeal"}
  ],
  "units": {"guards": ["hasPaymentMethod"], "actions": ["recordCancellation"], "actors": ["chargePayment"]},
  "delays": {"payTimeout": 5000}
}
```

- For an imported machine (section 5.2.1) `source` is the source file and `source_sha256` the
  SHA-256 of its bytes.
- `entries`, the key after `initial`, lists the `_entries` states, sorted. It is present only
  when the machine declares `_entries`.
- `states` lists every state in document order, nested states by dotted path. `kind` is `atomic`,
  `compound` or `final`. `resting` is true for a top-level, non-final state with no `invoke` and
  no `always`.
- `transitions` keeps document order, because branch order is evaluation order: the first branch
  whose guard holds wins. `event` is the event name, the delay name, or the invoke `src`; it is
  `""` for `always` and `stateDone`. `guard` and `target` are `""` when absent. `target` is the
  resolved dotted path. `requires`, the last key of a row, lists the branch's `_requires`,
  sorted; it is present only when the branch declares `_requires`.
- `clauses` lists the clause cases in the order of their transitions, then of the declaration. It
  is an empty list when the machine declares no `_guards`. (The example line comes from another
  machine; the fixture machine has none.)
- `units` lists every guard, action and actor name the machine uses, each sorted.
- `delays` maps each `_delays` name to its bound in milliseconds (the leading integer of the
  declaration).

A service loads its tables from the directory named by `MACHINERY_TABLES_DIR` when that variable
is set, and from its own configured location otherwise. Section 20 depends on this.

## 8. Trace

Code and tests point at the design with comment tags. The tag syntax is the same in every
language because it lives in a comment:

```
machinery:serves act:Order.pay act:Order.cancel
machinery:verifies tc:ORDE-2e883d sc:customer-pays-and-the-order-ships
machinery:verifies tc:ORDE-*
```

- A tag is the literal `machinery:serves` or `machinery:verifies`, then one or more ids separated
  by spaces, to the end of the line. An id token matches `[A-Za-z0-9_.:*-]+`; parsing stops at
  the first token that does not.
- A token may end with `*` and then matches every id with that prefix (`tc:ORDE-*`).
- `machinery:serves` is read from files matched by `trace.code`. `machinery:verifies` is read
  from files matched by `trace.tests`. A file both match is read once, so a finding in it is
  reported once.

Stage `trace` rules:

| Rule | Check |
|---|---|
| `unknown-id` | Every tagged id exists in the design; every `*` prefix matches at least one id. |
| `unserved` | Every `act` id has at least one `machinery:serves`, or is listed in `owed.yaml`. |
| `unverified` | Every obligation of kind `inv`, `sc`, `tc`, `fm`, `cx` and `ck` has at least one `machinery:verifies`, or is listed in `owed.yaml`. `fv` ids are proven by the formal stage and are not claimed by tests. |
| `proposed` | No tag or marker names a proposed id (section 4.1): `ERROR  proposed: <id> is claimed by <file>; commit it first`, once per id and file. |
| `retired` | No tag or marker names a retired id: `ERROR  retired: <id> is claimed by <file>; remove the claim, or un-retire it`. |
| `proof` | No tag or marker names a proof obligation (section 23.2): `ERROR  proof: <id> is claimed by <file>; tests do not discharge a proof obligation: it awaits a machine-checked proof`, once per id and file. A `pf:` token that names no proof obligation is `unknown-id`. |
| `adapter` | Only in a project that configures `evidence.adapter`, and when a `_test.go` file under `trace.tests` claims an id of kind `inv`, `sc`, `fm` or `ck`: the layout of the Go modules is one the adapter can attribute by import path (section 10.3, "The layout of the modules", with the files it reads): `ERROR  adapter: module <path> is declared by <a> and <b>: give each module its own path, or move the copy under a testdata directory` for every module path two tracked `go.mod` files declare, and `ERROR  adapter: <workspace> does not use the module of <file> (<go.mod>): add the module to the workspace, or set GOWORK=off` for every module of a claiming file the workspace leaves out, naming its first claiming file. |

A serves tag credits only actions. A `machinery:serves` tag or a serves marker that names an id
which is not an `act` id is no ERROR and no credit. The stage says so once per run, in one
counted line, whatever the number of tags (a project whose convention is to mark the code that
holds an invariant has hundreds):
`warn   serves: <n> tag(s) name <m> id(s) that are not actions; machinery trace --non-action-serves lists them`,
where `<n>` counts the tags (a file and a line) that name at least one such id and `<m>` the
distinct ids. Each id still needs what its kind needs, a test that claims it.

`machinery trace --non-action-serves` prints the list and nothing else, one line per tag and id,
`<file>:<line>: <id>`, sorted by file, line and id. It runs no check and is a report, never a
gate: exit 0, with no output when there is none. When the code or test files cannot be listed
or read, the list is not known and none is printed: one line per cause on stderr,
`machinery: trace --non-action-serves: <what stage trace reports>; the list is not known`, and
exit 1. Without `trace:` it is
`config: no trace configured: machinery trace --non-action-serves lists serves tags, and trace says where they are`
and exit 2.

A proposed or retired id counts as an id of the design for rule `unknown-id`. A `*` token claims
only the ids among its matches that yield an obligation, so a pattern that also matches a
proposed or retired id is not a finding. A `pf:` token is matched against the proof obligations
alone, and a pattern among them (`pf:*`) is a claim of each one it matches; a bare marker token
is never read as a `pf` id. `unserved` and `unverified` ask only for ids that yield
an obligation, and the owed list is checked as in section 4.1.

`checked:` counts code files, test files, `act` ids and obligations and, when a serves tag names
one, the distinct ids that are served and are not actions (`2 non-action serves`). Only files inside `scope`
are read. Zero files matched by a configured glob list is an ERROR. Not applicable when `trace:`
is absent.

Trace is static and cheap. It proves that a test claims an id, not that the test passed. Only
`verify` proves that. The claim is also what lets a receipt prove: `verify` takes a `pass` only
from a test whose file claims the id (section 10.2).

### 8.1 A project's own markers

A project that already points its code and tests at its design in its own words names those
markers in `trace.markers`, as literal prefixes. `machinery:serves` and `machinery:verifies` keep
working beside them, unchanged. A marker claims what the tag of the same kind claims, wherever
tags are read: stage `trace`, `machinery affected` (section 12.1) and the served ids of
`machinery adopt` (section 17.2).

- **Where.** A `serves` marker is read in the files of `trace.code` inside `scope`, a `verifies`,
  `scenarios` or `scenarios_exact` marker in the files of `trace.tests`. A marker counts only where it follows
  nothing but whitespace and comment characters (`/`, `#`, `*`, `-`, `;`, `%`, `!`, `<`) at the
  start of its line: `// Checks: RISK-3` and ` * Checks: RISK-3` count, `log("Checks: RISK-3")`
  does not. When several markers match one line, the longest wins.
- **Ids.** After a `serves` or `verifies` marker come ids, separated by commas, spaces or both.
  Reading stops at the first token that is not an id token (section 8), and a period that ends
  the list is ignored (`/* Implements: Order.cancel. */` serves `act:Order.cancel`). When
  every token of a line is an id and the line ends with a comma, the list goes on with the next
  line, read after its leading whitespace and comment characters, and so on while each line ends
  with a comma; a line that carries a marker of its own starts its own list instead.
- **Resolution.** A token with a kind prefix (`inv:`, `act:`, `sc:`, `tc:`, `fm:`, `fv:`, `cx:`,
  `ck:`) is taken as written, a `*` prefix included. A bare token is tried, in order, as
  `inv:<token>`, `act:<token>`, `fm:<token lower-cased>` and `sc:<token>`, and the first that
  names an id of the design is taken (a bare token ending in `*` is tried the same way, as a
  prefix). A token that resolves to nothing is `ERROR  unknown-id: <file>:<line>: <token>`, with
  the line the token is on.
- **Scenarios.** After a `scenarios` marker the rest of the line, trimmed, is the name of a model
  scenario, and the marker claims (verifies) the `sc` id of the scenario with exactly that name.
  No scenario of that name is `ERROR  unknown-id: <file>:<line>: scenario "<name>"`.
- **Scenarios, exact.** A `scenarios_exact` marker is for a marker the project also uses for
  other text, such as the objective of a test. It claims the same way when the rest of the line,
  trimmed, is exactly the name of a model scenario, upper and lower case included. Any other text
  claims nothing and is no finding. The longest marker of both lists decides a line, with its own
  mode.

The rules of stage `trace` do not change.

## 9. Architecture constraints

Stage `arch` enforces the `constraints:` block.

**Layers.** Layers are ordered from lowest to highest. A source file belongs to the first layer
whose globs match it. A file in a lower layer must not import a package that belongs to a higher
layer: `ERROR  layer: <file> (<layer>) imports <package> (<higher layer>)`. Files in no layer are
counted and not checked. Go is supported first: imports come from the Go parser, and an import
path is mapped to a project path through the module path in the project's `go.mod`.

**Size.** For files matched by `constraints.size.paths`: a file longer than `max_file_lines` is
`ERROR  file-size: <file> has <n> lines (max <m>)`; a Go function longer than `max_func_lines` is
`ERROR  func-size: <file>:<func> has <n> lines (max <m>)`.

**Cores.** The core of a vital area (section 23.1) is where the code of a path that must not fail
lives: the files its `core.paths` globs match. A later version will hand these files to a
machine-checked proof, so they are held to a shape a proof can work with. `constraints.cores`
gives the limits, and the core of every vital area is held to them:

| Rule | Finding |
|---|---|
| `core-missing` | A `core.paths` glob matches no file: `ERROR  core-missing: <area>: <glob> matches no file`. |
| `core-scope` | A core file lies outside scope: `ERROR  core-scope: <area>: <file> is outside scope; widen the scope or move the core`. It is not read. |
| `core-file-size` | A core file is longer than `max_file_lines`: `ERROR  core-file-size: <file> has <n> lines (max <m> for a core)`. |
| `core-func-size` | A Go function of a core file is longer than `max_func_lines`: `ERROR  core-func-size: <file>:<func> has <n> lines (max <m> for a core)`. |
| `core-entry` | Each `core.entry` names a core file and a symbol in it, found by the Go parser: a function, a method (`Type.Method`) or a type. Otherwise `ERROR  core-entry: <area>: <file>#<symbol>: <what is wrong>`, one of `no such file`, `<file> is not matched by core.paths` and `no function, method or type <symbol> in <file>`. |
| `core-impure` | A core file imports a package that `deny_imports` names, or one below it (`os` denies `os/exec`): `ERROR  core-impure: <file> imports <package>, which constraints.cores.deny_imports denies`. |
| `core-bypass` | A file that serves (section 8) an `act` id a property of the area is over, is not a core file of the area and does not import one: `ERROR  core-bypass: <file> serves <act id> outside the core of <area> and does not import it: the path that must not fail has a second implementation`. A Go file imports a core file when one of its imports is the directory of a core file; a file in another language when one of its imports resolves to a core file. |

- A core file obeys the layer rule like any other file: a core never imports a higher layer.
- What cannot be read is not demonstrated, never passed. Denied imports, function sizes and entry
  symbols are read in Go files only. For a core file in another language the stage prints
  `warn   core <area>: <file>: <what> is checked in Go files only`, `<what>` being
  `deny_imports` (when the list is not empty) and `max_func_lines`, and
  `warn   core <area>: entry <file>#<symbol>: symbols are found in Go files only` for an entry
  there. `core-bypass` reads the serves tags of `trace.code`: without `trace:` it is
  `warn   core <area>: core-bypass reads serves tags and trace is not configured` when a
  property of the area is over an `act` id, and a serving file in a language whose imports are
  not read (section 9, below) is `warn   core <area>: <file>: imports are not read for this
  language`. With any of these warnings the stage is `not demonstrated: <the first of them,
  sorted>`.
- A vital area while `constraints.cores` is absent is
  `ERROR  cores: <area> is vital and constraints.cores is not set; set max_file_lines and max_func_lines for its core`.
- Baseline subjects: `<area>: <glob>` for `core-missing`, `<file>` for `core-scope` and
  `core-file-size`, `<file>:<func>` for `core-func-size`, `<file>#<symbol>` for `core-entry`,
  `<file> -> <package>` for `core-impure` and `<file> -> <act id>` for `core-bypass`.
- `checked:` adds, per vital area in name order, its core files, their lines and its entry
  points: `checked: 3 files, 1 imports, 2 functions, 1 core files of order-payment, 9 core lines
  of order-payment, 1 entry points of order-payment`.

**Baseline.** Existing violations may be listed in the baseline file:

```yaml
version: 1
baseline:
  - {rule: layer, subject: "internal/domain/order.go -> internal/api"}
  - {rule: file-size, subject: "internal/api/server.go"}
  - {rule: func-size, subject: "internal/api/server.go:Serve"}
```

A violation listed in the baseline prints as `warn   baselined: <message>`. A baseline entry with
no matching violation is `ERROR  stale baseline entry: <rule> <subject>`. The baseline only
shrinks. `machinery arch --write-baseline` writes the current violations (sorted by rule, then
subject), prints `arch: wrote <n> baseline entries to <file>` and then runs the stage; it is a
deliberate act, and a config error when no baseline path is configured.

Subjects: a layer violation is `<file> -> <package directory or file>`; a function is
`<file>:<Func>` or `<file>:<Type>.<Method>`. A Go file that does not parse is
`ERROR  parse: <file>: <error>`, and a layered Go file with no `go.mod` above it is
`ERROR  go-mod: <file>: no go.mod found`.

`checked:` counts files, imports and functions. Only files inside `scope` are read. Not applicable
when `constraints:` is absent and the risk file has no vital area.

Other languages: imports are read by pattern from Python (`import`, `from`), TypeScript and
JavaScript (`import`, `require`), Elixir (`alias`, `import`, `use`) and Rust (`use crate::`,
`mod`). A relative or project-rooted import is mapped to a project path; anything else is
external and ignored. Function size is checked for Go only.

### 9.4 Stage `tests`: end-to-end only

Evidence is only as good as the system it was taken from. The stage enforces two rules on test
files, where a test file is a file inside `scope` or under the e2e roots that matches a test-file
pattern:

| Rule | Check |
|---|---|
| `unit-test` | A test file must be under one of the e2e roots (`tests.e2e`). Anywhere else it is a unit test: `ERROR  unit-test: <file>`. |
| `test-double` | A test file under the e2e roots must not contain a test-double marker: `ERROR  test-double: <file>: <marker>`. |

Default test-file patterns: `**/*_test.go`, `**/*.{test,spec}.{js,jsx,ts,tsx,mjs,cjs}`,
`**/test_*.py`, `**/*_test.py`, `**/*_test.exs`, `**/*_test.rs`. Default markers (plain
substrings): `gomock`, `testify/mock`, `sqlmock`, `httpmock`, `unittest.mock`, `MagicMock`,
`jest.mock(`, `vi.mock(`, `sinon`, `nock(`, `Mox.`, `mockall`. `tests.files` and `tests.doubles`
replace the defaults.

An existing project lists what it already has in the baseline, with the same rules as the
architecture baseline: a listed violation prints as `warn   baselined: <message>`, a stale entry
is an ERROR, the file only shrinks, and `machinery tests --write-baseline` writes it.

```yaml
version: 1
baseline:
  - {rule: unit-test, subject: "internal/billing/calc_test.go"}
  - {rule: test-double, subject: "e2e/api_test.go: gomock"}
```

`checked:` counts test files, files under e2e roots and markers looked for. Not applicable when
neither `tests:` nor `trace.tests` is configured. A project with nothing under its e2e roots and
no test file at all is `not demonstrated: nothing to check`. End-to-end tests in a form the
patterns do not know (a shell script, say) count as files under the e2e roots; name them with
`tests.files` to have them scanned for test doubles.

## 10. Runs and receipts

Evidence is produced by the project's own end-to-end tests, in any language, run through
`machinery run`:

```
machinery run [--stack "<description>"] -- <command> [args...]
```

With `--adapter go-test-json` before the `--`, machinery also reads the event stream of an
existing Go suite and writes receipts itself (section 10.3). Everything in this section holds
for such a run too.

`machinery run`:

1. reads the current commit (`git rev-parse HEAD`) and whether the tree is dirty
   (`git status --porcelain` not empty, ignoring the owed file, the receipts files (the
   `.jsonl` files directly in the receipts directory, and nothing else in it) and the property
   records `<gen>/vital/*.json`, whose statuses render the owed file (section 23.4));
2. deletes receipt files in the receipts directory that are not valid for the current commit
   (section 10.1), and their entries in the run ledger, together with the entries of this
   receipts directory whose file is no longer there (section 10.2);
3. creates `<receipts dir>/<UTC timestamp>-<short sha>.jsonl` (timestamp
   `20060102T150405.000Z`), writes the header line and enters the file in the run ledger;
4. runs the command with extra environment variables: `MACHINERY_RECEIPTS` (absolute path of
   the file), `MACHINERY_COMMIT` (the full commit sha) and, for a targeted run (section 12.1),
   `MACHINERY_ONLY` (the ids to exercise, comma separated);
5. appends the end line when the command exits, and records the digest of the finished file in
   the run ledger;
6. exits with the command's exit code, after printing `machinery run: <file> (exit <n>)` on
   stderr. A command that cannot be started exits 127 and leaves no file and no ledger entry; a
   command killed by a signal exits 128 plus the signal number.

The command runs in the project root. The variables machinery sets for a run
(`MACHINERY_RECEIPTS`, `MACHINERY_COMMIT`, `MACHINERY_ONLY`, `MACHINERY_TABLES_DIR`,
`MACHINERY_PROBE_CASE`, `MACHINERY_GO_TEST_JSON`) are never inherited from the caller.

The file is JSON Lines. Three line types:

```json
{"v":1,"type":"run","commit":"<40 hex>","dirty":false,"command":["go","test","./e2e/..."],"stack":"postgres 17 in docker","started":"2026-01-02T03:04:05Z"}
{"v":1,"type":"receipt","id":"tc:ORDE-2e883d","outcome":"pass","test":"order_test.go::TestOrder/ORDE-2e883d","effects":["stampAttempt"],"observed":"Pending -> paying; attempt row written"}
{"v":1,"type":"end","exit":0,"finished":"2026-01-02T03:04:09Z"}
```

A test emits one `receipt` line per obligation it checked, by appending to the file named by
`MACHINERY_RECEIPTS`:

- `id`: an obligation id.
- `outcome`: `pass`, `fail` or `skip`.
- `test`: the test that made the observation, required, written as `<file>` or
  `<file>::<name>`: the test file by its path or its base name, then optionally the name of the
  test in it (section 10.2).
- `effects`: for `tc` ids, the effects the test actually observed. Omitted or empty means none.
- `observed`: one short sentence, at most 300 characters, saying what was seen.
- `replay`: for `cx` ids only, what the real system did when the counterexample was replayed:
  `reproduced` or `refuted` (section 16.10).
- `source`, `granularity`, `tags`: only on a receipt a run adapter wrote (section 10.3). A
  receipt a test writes with any of them is refused (section 12).

A test writes each line with a single append so that parallel tests do not interleave. When
`MACHINERY_RECEIPTS` is not set, the test emits nothing and still runs. When `MACHINERY_ONLY` is
set, a suite may run only the tests for those ids; a suite that ignores it is still correct.
The Go helper `pkg/receipt` implements all of this for Go suites (`receipt.Test` builds the
`test` of the calling file); other languages append the JSON line themselves.

A receipt is a claim made by a test. A test asserts first and emits afterwards: it emits `pass`
only after its assertions held. Section 10.2 says which receipts are read at all.

### 10.1 When evidence is valid

Evidence is bound to a commit. A receipts file is **valid for the current commit** when its header
commit is the current commit, or when the tree of its header commit differs from the tree of the
current commit only in the owed file, in receipts files and in the property records
`<gen>/vital/*.json` (`git diff --name-only <receipt commit> HEAD` lists nothing else). A property
record changes on its own only when the owed file does: any other change to it comes from a risk
file, a design source or a core file, which are not exempt. Receipts are not meant to be committed; when they are, by a broad
`git add`, the evidence stays valid. The owed file is bookkeeping about evidence, not part of the system under
test, so pruning it and committing does not invalidate the run that justified the pruning.
Evidence from any other commit says nothing about the current one.

### 10.2 Which receipts count

Two conditions hold before a receipt is read as proof. Neither is about what the line says; both
are about where it came from.

**The file was written by `machinery run`.** `machinery run` keeps a run ledger outside the
working tree, one entry per receipts file, at
`<git common dir>/machinery/runs/<name of the receipts file>`. An entry is one JSON line:

```json
{"v":1,"commit":"<40 hex>","dir":"<receipts directory, absolute>","file":"<sha256 of the finished file>"}
```

The entry is written without `file` when the run starts: the commit of the run and the receipts
directory the file is in. `file`, the digest of the whole file with its header and its end
line, is added when the command has exited. Every command that reads evidence asks the ledger
before it believes a header, for every `.jsonl` file of the receipts directory and for every
entry that names this receipts directory:

| Finding | When |
|---|---|
| `ERROR  receipts: <file>: not produced by machinery run; delete the file and run the tests through machinery run` | The file has a run header whose commit is valid for the current commit (section 10.1) and the ledger has no entry of its name. |
| `ERROR  receipts: <file>: changed after the run; delete the file and run the tests again through machinery run` | The entry has a digest and the file does not have it, whatever its header says now, or whether it still has one. Or the entry has no digest (the run never ended), its commit is valid for the current commit, and the file has an end line, no header, or a header of another commit. |
| `ERROR  receipts: <file>: recorded by machinery run and no longer present; run the tests again through machinery run` | An entry names this receipts directory, its commit is valid for the current commit, and no file of that name is in the directory. |

Nothing in a refused file counts: it is not a run, none of its receipts is read, a failure in it
included, and nothing else about it is reported. An untouched run of another commit matches its
entry and is passed over without a finding, as before. A file with no end line whose entry has
no digest, and whose header still carries the commit of the entry, is the incomplete run of
section 12. An entry that is not such a line is no entry.

A failing run does not vanish when its file is edited, cut or deleted: the three findings above
and the incomplete run are carried everywhere a result is given. `verify`, `verify --story` and
the tier gates print the ERROR (section 15.6), `machinery status` prints a `refused` line
(section 14), `machinery proof` gives the number of refused files in place of counts (section
13), and `machinery owed sync` prunes nothing (section 11.2). What clears them is a new
`machinery run`, after deleting the refused file.

- The entry goes by the name of the file, so a copy of a real run under another name has no
  entry. `machinery run` deletes the entry with the file, and the entries of its receipts
  directory whose file is gone (step 2 of section 10).
- The ledger is shared by the worktrees of a repository and is in none of them: it is never
  dirty, never committed and never cloned. An entry of another worktree or project names another
  receipts directory and is never reported as gone here. Evidence is produced in the repository
  where it is judged.
- Everything that appends receipts finishes before the command exits. A process that outlives
  the command and appends later changes the file after the run.

**The test names a test file that claims the id.** The `test` of a receipt is `<file>` or
`<file>::<name>`, where `<name>` is free text (the test function, a case). `<file>` is:

1. the project-relative path of a file matched by `trace.tests` (`e2e/orders.sh`,
   `e2e/order_test.go::TestOrder/ORDE-2e883d`); or
2. a base name that exactly one file matched by `trace.tests` has (`orders.sh`,
   `order_test.go::TestOrder`).

That file must statically claim the receipt's id: by a `machinery:verifies` tag, a `verifies`
marker or a `scenarios` marker, read and resolved exactly as stage `trace` reads them (sections
8 and 8.1), `*` patterns included. Nothing else is tried: the text of the test files is not
searched for the test's name. Otherwise the finding is
`ERROR  receipt: <id>: <test> does not claim it; <what to do>`, once per id and test, where
`<what to do>` is one of:

| `<what to do>` | When |
|---|---|
| `trace.tests is not set: list the test files there in machinery.yaml` | The project has no `trace.tests`. |
| `no file under trace.tests is named <file>: write test as <path of the test file> or <path>::<name>` | `<file>` is neither such a path nor such a base name. |
| `<n> files under trace.tests are named <file>: write the path in test` | Several test files have that base name. |
| `add machinery:verifies <id> to <path>, or emit the receipt from the test that claims it` | The file is found and does not claim the id. |

Such a receipt cannot prove. A `pass` among them is not used, so it proves nothing and prunes
nothing; a replay `pass` that would prove its `cx` id is not used either. What it reports
against the system is kept: a `fail` still fails its obligation, and a replay that fails its
trace is still reported, in `verify`, in the tier gates, in `proof` and in `status`, exactly as
if the test claimed the id, and the ERROR above is printed as well. A receipt whose id is not an
obligation gets the finding section 12 has for it instead.

**What this defends against, and what it does not.** It defends against mistakes and casual
edits:

- a receipts file written or pasted by hand, or copied from another run or another project;
- a receipt added, removed or reworded after the run, a header edited to another commit or to
  `"dirty":false`, an end line added to a run that never finished;
- a `pass` emitted for an id that the file of its test does not say it verifies: a mistyped id,
  a helper that emits for every id it is given, an emitter left behind when the claim moved, a
  test name that points at no test file.

It does not defend against a party who can run commands as the tool:

- Whoever can run `machinery run` can run a command that appends any line to
  `MACHINERY_RECEIPTS`, and the ledger binds that file like any other. The ledger says that
  machinery ran a command and that the file is what the command left. It does not say that the
  command ran tests, that the test a receipt names ran, or that its assertions held.
- Deleting the file of a failing run and then running anything through `machinery run` forgets
  that run: the finding for a deleted file lasts until the next run in that receipts directory,
  not until the failing test runs again.
- Whoever can write the git directory can write a ledger entry. The digest is not secret and
  nothing is signed.
- The claim rule compares two things the emitter chooses: the file name in `test` and the
  claims in that file. A line that names any test file claiming the id passes it, whether or not
  that file emitted the line, and the party who writes the receipts usually writes the test
  files too.

A receipt therefore stays what section 10 says: a claim made by a test. What checks the tests
themselves is the probe (section 20), the verifier's fresh context (section 15.6) and a reader
of the test file; the claim rule tells that reader which file to open.

### 10.3 Evidence from an existing Go suite: the adapter `go-test-json`

A project that already has a Go suite whose tests say in comments which ids they verify can
have its passing tests credited without changing one of them:

```
machinery run --adapter go-test-json [--stack "<description>"] -- go test [flags] [packages]
```

The run is a run of section 10 in every respect: the same file, the same header and end line,
the same ledger entry (section 10.2), the same environment for the command, the same exit code.
Two things differ. machinery reads the event stream of `go test -json` from the command's
standard output, and after the command has exited it writes receipts itself, before the end
line. A test that appends receipts of its own to `MACHINERY_RECEIPTS` during such a run keeps
doing so, and both kinds are in the file.

**What the adapter defends against, and what it does not.** Adapter evidence is exactly as
trustworthy as the project's test code and the command that ran it, the same as a receipt a test
emits itself: a test can emit a `pass` without checking anything (section 10.2), and a test the
adapter credits can pass without checking anything. The adapter defends against mistakes and
accidents: a pass replayed from a stale cache, a function of the same name in another file, a
claiming file the run did not build, a run in the wrong directory or of the wrong module, a fail
that would go unnoticed (a test that fails, hangs, exits or crashes, a subtest that skipped or
was filtered out, a stream cut short). It does not defend against a party who adds decoy code or decoy modules
to the project, or who controls the environment of the run (the `go` it finds, the variables of
`go`, the toolchain, the machine). What such a party plants is the business of review, of the
ratchet (section 11.3), and of the rule that by default an id of a critical or vital area never
accepts adapter evidence (below). The rules of this section are aimed at mistakes, and a party
who plants content can get around them; what a run used is recorded in its header so that a
reader sees it.

**What adapter evidence is, and what it is not.** An adapter receipt says that a test which
names the id passed in this run. It does not say that the test checked the id. A receipt a test
emits follows that test's own assertions about that id (section 10); an adapter receipt follows
only the verdict of `go test` on a function whose file, or whose own comment, carries the id.
A marker above a test that asserts something else, a marker at the top of a file whose tests
have drifted away from it, and a test that asserts nothing all yield a `pass`. That is why
adapter evidence is counted as a class of its own wherever a result is given (sections 12, 13
and 14), why its granularity is recorded, and why by default it never satisfies an id of a
critical or vital area.

**The command: direct mode is deliberately narrow.** The adapter attributes a verdict by the
directories of the project's module as they are on disk ("Whose verdict", below), so direct mode
accepts only a short list of flags and relative patterns, in a layout of modules it can read
("The layout of the modules", below), for a test binary built for the machine that runs it. The
list keeps a habit or a slip (`-C`, `-modfile`, `-exec`, an import path) from crediting other
files; it is no fence against a party who controls the environment (above). Every suite the list
refuses runs through `--streams` (below), which trusts the wrapper and its entries of
`evidence.adapter.sources` instead.

The command must be `go test`: `go`, found on `PATH`, or a path whose base name is `go`, then
`test`. A `go` that is a file inside the project, by the path the command names or the one
found on `PATH`, links not followed, is refused. Then
only these words, flags with one dash or two, a flag that takes a value as one word
(`-run=X`) or two (`-run X`):

- the flags `-run`, `-skip`, `-tags`, `-race`, `-count`, `-timeout`, `-v`, `-json`, `-p`,
  `-parallel`, `-short`, `-failfast`, `-shuffle`, `-cpu`, `-cover`, `-coverprofile`,
  `-covermode` and `-coverpkg`, and the spellings `-test.run`, `-test.skip`, `-test.count`,
  `-test.timeout`, `-test.v`, `-test.parallel`, `-test.short`, `-test.failfast`,
  `-test.shuffle`, `-test.cpu` and `-test.coverprofile`;
- package patterns relative to the project root, where the command runs: `.`, or a word that
  begins with `./` and has no `..` element (`./...`, `./e2e/suite`);
- `-args`, as the last word: nothing may follow it.

A `-run` or `-skip` value (any spelling) that contains `/` selects subtests, and a parent
whose subtests a filter left out passes with nothing checked: it is refused too, printed as
`-<flag>=<value>` on the command line and as written in `GOFLAGS`, with the message of every
other refused word. Every other word is refused: another flag (`-C`, `-exec`, `-mod`, `-modfile`, `-overlay`,
`-vet` and the rest), an absolute pattern, a module or import path (`example.test/shop/...`,
`all`), a file name that does not begin with `./`, a word after `-args`. The environment is
held to the same rules, as `go env` reads it in the project root (the variables and the go
environment file): every word of `GOFLAGS` must be on the list above (`-json=false` there is
harmless: the `-json` of the command overrides it), `GOMOD` must be the `go.mod` of the
project root, and `GOOS` and `GOARCH` must be `GOHOSTOS` and `GOHOSTARCH`: `go` runs a test
binary built for another system through a program `go_<GOOS>_<GOARCH>_exec` it finds on `PATH`,
and what that program prints is not `go test`. A workspace in effect (`GOWORK`: a file the
variable names, or the `go.work` of the project root or of a directory above it) must use the
modules of the claiming files ("The layout of the modules", below). Then:

- `-json` is added when it is absent, right after `test`.
- `-count=1` is added when no count is given, after `-json`, so that `go test` never answers
  from its cache. Every spelling that sets another count is refused: `-count` and `-test.count`,
  each as one word or two, and the same words in `GOFLAGS`. With `--streams`, `GOFLAGS` is read
  from the environment of machinery; it is all machinery can see of the count, and a count
  other than 1 or an `-overlay` there is refused too.
- `-tags` gives the tags recorded in the receipts, in the order written (separated by commas or
  spaces); without the flag they are read from a `-tags=` word of `GOFLAGS`. They are a record
  of the command and decide nothing: `-race`, `GOOS` and the like set tags of their own, which
  are not recorded, and no build constraint is ever evaluated ("Whose verdict", below). With
  `--streams` the tags are not known (below).

The header records the command as it ran, `-json` and `-count=1` included,
`"adapter":"go-test-json"` and, in direct mode, what `go env` gave in the project root right
before the run:

```json
"environment":{"go_version":"go1.27.4","go_binary":"/usr/local/go/bin/go","GOFLAGS":"","GOWORK":"","GOOS":"linux","GOARCH":"amd64","go_mod_sha256":"<64 hex>","go_work_sha256":""}
```

`go_version` is `GOVERSION`; `go_binary` is the `go` of the command, as found on `PATH` or as
written (from the project root), links not followed; `GOFLAGS`, `GOWORK`, `GOOS` and `GOARCH`
are what `go env` gave; `go_mod_sha256` is the SHA-256 of the file `GOMOD` names and
`go_work_sha256` that of the file `GOWORK` names, `""` when there is none. It is a record and
decides nothing: two runs at the same commit in different environments are both evidence, and
the difference is no finding. `verify` and `proof` print it, one line per adapter run they use
(sections 12 and 13):
`adapter environment: <file>: <go_version> <go_binary>, GOOS/GOARCH <GOOS>/<GOARCH>, GOFLAGS "<GOFLAGS>", GOWORK "<GOWORK>", go.mod <hex>, go.work <hex>`,
where `<hex>` is the first 12 characters of the digest and `go.work none` stands for a run
without a workspace. A run through `--streams` records no environment, since machinery did not
run `go`; its line is `adapter environment: <file>: not recorded: the command ran through --streams`.

**The layout of the modules.** Direct mode attributes a verdict by import path, so before
anything runs it reads the `go.mod` files of the project and the workspace in effect, as files,
and refuses two layouts under which `go` builds a package of a project's import path from
files the adapter does not attribute it to:

- a module path declared by two `go.mod` files of the project:
  `module <path> is declared by <a> and <b>: give each module its own path, or move the copy under a testdata directory`,
  the files sorted (`<a>, <b> and <c>` for three). Only the `go.mod` files git tracks in the
  project count, and none whose path has an element `testdata` or `vendor`: a fixture module,
  a vendored copy or a module cache kept in the project (untracked) is no module of the project;
- a workspace that does not use the module of every claiming file:
  `<workspace> does not use the module of <file> (<go.mod>): add the module to the workspace, or set GOWORK=off`,
  naming the module by its `go.mod` and by the first claiming file it holds. A claiming file is
  a `_test.go` file under `trace.tests` that claims an id of kind `inv`, `sc`, `fm` or `ck`;
  its module is the directory of the nearest `go.mod` at or above it; a workspace uses it when
  one of its `use` directives, read from the directory of the workspace file, names that
  directory. The workspace is the one in effect (`go env GOWORK`; none with `GOWORK=off`), and
  it is read only when it is a file git tracks in the project, outside `testdata` and
  `vendor`; `<workspace>` is its project-relative path.

A run in direct mode is refused for the first of them, module paths and files in sorted order
(the table below), whatever the project configures. Stage `trace` reports every one, from the
files alone, only for a project that configures `evidence.adapter` (section 8); a project
without that block sees nothing of these rules there. In stage `trace` the workspace is the
file the variable `GOWORK` names, or else the `go.work` of the project root, under the same
conditions, and none when `GOWORK` is `off`. With `--streams` the run does not read them: the
entries of `evidence.adapter.sources` decide which files a package has. A decoy that git does
not track makes the tree dirty (section 10), and its run counts only with `--allow-dirty`;
one that git ignores is planted content (the threat model, above).

machinery prints the `Output` text of every event on its standard output, and a line of the
stream that is no event as it is.

Refusals, each `usage: <message>` with exit 2, before anything runs and with no file written,
checked in the order of the table and the words of the command in the order written. With
`--streams` the command and `go env` are not read: there only the rows on `GOFLAGS` that say so,
`--streams` alone, `trace.tests` and the entries of `evidence.adapter.sources` apply. Each says
what to do:

| Message | When |
|---|---|
| `--adapter: unknown adapter "<name>": write --adapter go-test-json, the only adapter` | Another name. |
| `--adapter go-test-json: the command must be go test, got <first two words>: run go test itself, or add --streams and have the command leave its go test -json output in MACHINERY_GO_TEST_JSON` | The command is not `go test`: a wrapper cannot be given `-json` and its output is not known to be the event stream. |
| `--adapter go-test-json: <path>: a go inside the project is not accepted by the adapter in direct mode; run the command through --streams` | The `go` of the command is a file inside the project. |
| `--adapter go-test-json: -count=<value> is refused: write -count=1` | A count other than 1 on the command line; the flag is printed as it was written (`-test.count=<value>`). |
| `--adapter go-test-json: -json=false is refused: the adapter reads the event stream; leave it out` | `-json` switched off. |
| `--adapter go-test-json: <word>: not accepted by the adapter in direct mode; run the command through --streams` | A word of the command outside the list above, as written. |
| `--adapter go-test-json: go env failed (<first line it printed>): the adapter reads GOFLAGS, GOMOD and GOWORK from it; fix the environment of go, or run the command through --streams` | `go env` cannot answer. |
| `--adapter go-test-json: GOFLAGS has -count=<value>, which is refused: take it out of GOFLAGS` | A count other than 1 in `GOFLAGS`, with or without `--streams`. |
| `--adapter go-test-json: GOFLAGS has -overlay, which is refused: take it out of GOFLAGS` | With `--streams`, `-overlay` in `GOFLAGS`. |
| `--adapter go-test-json: <word> in GOFLAGS: not accepted by the adapter in direct mode; run the command through --streams` | A word of `GOFLAGS` outside the list, a count other than 1 excepted (above). |
| `--adapter go-test-json: go env GOMOD is "<value>", not the go.mod of the project root: run the command through --streams` | Another module, or none. |
| `--adapter go-test-json: go env GOOS/GOARCH is <GOOS>/<GOARCH>, not the host's <GOHOSTOS>/<GOHOSTARCH>: go would run the test binary through go_<GOOS>_<GOARCH>_exec; unset GOOS and GOARCH, or run the command through --streams` | A test binary built for another system. |
| `--streams needs --adapter go-test-json: add it before --streams` | `--streams` alone. |
| `--adapter go-test-json: trace.tests is not set: the adapter joins the run to the claims of the test files; list them in machinery.yaml` | No `trace.tests`. |
| `--adapter go-test-json: evidence.adapter.sources: entry <n> (package <glob>) matches <import path>, the package of <directory>: entries are read only with --streams; remove the entry, or run the suite through --streams` | Without `--streams`, an entry whose `package` matches the import path of a directory that holds files under `trace.tests` ("Whose verdict", below). The first such entry, and for it the first such import path in sorted order. |
| `--adapter go-test-json: module <path> is declared by <a> and <b>: give each module its own path, or move the copy under a testdata directory` | Without `--streams`, a module path declared twice (above). |
| `--adapter go-test-json: <workspace> does not use the module of <file> (<go.mod>): add the module to the workspace, or set GOWORK=off` | Without `--streams`, a workspace that leaves out the module of a claiming file (above). |
| `--adapter go-test-json: evidence.adapter.sources: entry <n> (package <glob>): files <globs> match no file under trace.tests: correct the globs, or list the files in trace.tests` | With `--streams`, an entry that maps onto nothing; `<globs>` as written, separated by `, `. |

**Cached results never count.** Without `-count=1`, `go test` may
answer for a package from its cache and replay the events of an earlier run. machinery adds the
flag to a command it runs itself; it cannot add it to the `go test` a wrapper starts
(`--streams`, below), and there this rule is the whole protection. `go test` marks a cached
answer in the stream: a package-level `output` event (one with no `Test`) whose text begins
with `ok` and has the field `(cached)`, `ok  \t<package>\t(cached)`. In a run of a package
that has that mark (a stream may hold several runs of one package, "A package that failed",
below), no pass of the package is read: a replayed pass checked nothing in this run, it counts
for nothing, and the summary names the package. Its fails and skips are read like those of any
run. Each run is judged on its own: a package answered from the cache in one run and run in
another, of the same stream or of another, has the verdicts of the run that ran it, and no
verdict of any run is dropped because the package was cached somewhere else.

**A package that failed.** A stream holds one run of a package for each package-level end event
of the package (an event with no `Test` whose action is `pass`, `fail` or `skip`), as when
a wrapper appends the output of two `go test -json` commands to one file: the events of the
package up to and including its first end event are its first run, the events after that up to
and including the next end event its second, and so on. `go test -json` and `go tool test2json`
begin each run of a package with a package-level `start` event; a `start` event of a package
whose current run in the stream has events and no end event yet (a run that was killed and then
started again, two runs of the package writing into one file at the same time) leaves the run
incomplete (below), since the events of the two runs can no longer be told apart. A stream
without `start` events (lines typed by hand) is split at its end events alone. Each run is judged by the rules of this
section exactly as the runs of several streams are ("One package in several runs", below), so
a fail, a crash or a missing verdict in any run of the package is a fail of the function,
whatever another run gave it. A package fails in a run when the run's end event is `fail`: a
test failed, the test binary did not finish (a `-timeout`, `os.Exit` or `log.Fatal`, a panic
in any goroutine, a crash), or the package did not build. What it fails:

- in that run, a test function of the package that has a `run` event and no verdict failed:
  whatever stopped the binary stopped it, also when another run gave it a verdict (its
  verdicts differ, "Which verdict", below);
- every test function of a candidate file under `trace.tests` of the package ("Whose
  verdict", below) that has no verdict in any run of the package, in any stream, failed too. That includes a
  function that never started, because the binary stopped before it or `-run` did not select
  it, and one of a file the build left out (machinery never decides which files were built):
  a false fail rather than a lost one. Its receipt says `no verdict; the package failed`;
- no pass of the package counts, in any run of it: the failure may be that of a function that
  passed (a goroutine it left running, a check of `TestMain` after `m.Run`), so in a package
  that did not end with pass a `pass` event counts for nothing, while a `fail` or a `skip`
  is read as it is. A function that passed there gets no receipt from this run, and a pass of
  an earlier run at the same commit still counts for it; every function the run read as failed
  has a `fail` receipt, and a fail of any run fails its id, whatever an earlier run passed
  (section 12).

A stream in which a package has events and no package-level end event (`go test` was killed,
the stream was cut short), or events after its last one with no end event of their own (a
second run of the package cut short), is no complete run: machinery writes no receipt and no
end line and exits with the command's exit code, or 1 when that was 0. It prints
`machinery: run: go test -json: package <import path> has no end event; the run is left incomplete: find what stopped go test, delete <receipts> and run again`,
and with `--streams` `machinery: run: stream <file>: package <import path> has no end event; the run is left incomplete: fix what writes the stream, delete <receipts> and run again`,
where `<receipts>` is the receipts file of the run by its project-relative path; for events
after the last end event the message says `has events after its last end event` instead of
`has no end event`. A `start` event before the end of the run it would interrupt is read where
it stands, before the lines after it, and only with `--streams` (one `go test -json` run gives
each package one `start` event):
`machinery: run: stream <file>: line <n>: package <import path> starts again before its run ended; the run is left incomplete: find what stopped go test or give each go test run a file of its own, delete <receipts> and run again`.
`verify` reports the file as an incomplete run (section 12). So within
the threat model no fail is lost: a failed function, a function a failure stopped and the
claims of a package that failed all write fails, in whichever run of the package they happened,
and a stream that does not say how its packages ended, or that mixes two runs of one, counts
for nothing. A fail of any run,
of any stream and of any package writes a fail receipt for every id its function's claims name
("Which receipt", below).

**Test functions.** The adapter counts as a test function every top-level function without a
receiver whose name begins with `Test` and that has exactly one parameter, whatever its type is
written as: `*testing.T`, `*T` through an alias or a dot import of `testing`, or anything else.
That is more than `go test` runs, on purpose: it also counts `Testfoo`, which `go` does not run,
and signatures `go` refuses. A function with no parameter or with two (`TestdataDir()`,
`TestdataJoin(dir, name string)`) is none. It is what makes a name ambiguous ("Whose
verdict", below) and what a claim of a whole file waits for ("Whose claim"). The one exception
is a `TestMain` whose parameter is written as a pointer to a type named `M` (`*testing.M`,
`*M`): it is the test main. A `TestMain` with another parameter, `TestMain(t *testing.T)`
for one, is a test function, and `go test` runs it as a test. Benchmarks, fuzz tests and
examples are not test functions. Files are parsed as Go source and never compiled, and their build constraints are not
read: a file left out of the build is read like any other. machinery never decides which files
a run built (below).

**Whose claim.** The claims of a file under `trace.tests` are read exactly as stage `trace`
reads them (sections 8 and 8.1). Each is attributed:

- a tag or marker on a line of the comment directly above a test function (its doc comment: the
  comment group that ends on the line before `func`) belongs to that function. The receipt has
  `granularity: function`;
- a tag or marker on a line of the comment directly above any other function declared at the
  top level of the file (a fuzz test, a benchmark, an example, `TestMain`, a helper, a method)
  belongs to that function, which the adapter does not judge. The claim gives no receipt and is
  counted neither as `claimed, not run` nor as `claimed, verdict not counted`; the summary
  names it once,
  `adapter: <file>: <id>: the adapter does not judge <function>; put the marker above a Test function or at file level`.
  It never falls back to the file: a test function of the file that passed says nothing of a
  fuzz test whose seeds were skipped or not selected, or of a benchmark, which direct mode never
  runs;
- a tag or marker anywhere else in the file (above no function) belongs to every test function of the file. The
  receipt has `granularity: file`, and it needs a verdict of every one of them ("Which receipt",
  below): a function of the file that has none in the run, because it was not built or not
  selected, leaves the id without a receipt, unless its package failed (above).

**Which verdict.** The verdict of a function is its `pass`, `fail` or `skip` event, read with
those of its subtests (a `Test` that begins with the function's name and a `/`, at any depth)
in the same run of its package: the function failed when it or a subtest failed, it was skipped when it or
a subtest was skipped, and it passed only when it passed and no subtest of it failed or was
skipped. `go test` passes a parent whose subtests all skipped; for the adapter that parent is a
skip, as one whose own body calls `t.Skip` is. A filter that leaves subtests out is refused in
direct mode (above); with `--streams` the command is not read, and a parent whose subtests the
wrapper's filter left out reads as a pass. A function with no verdict did not run, whatever
other events name it, except in a package that failed (above). A pass of a package answered
from the cache is not read (above). A
function with several verdicts that are not all the same failed, whichever they are: in one
run, across the runs of one stream or of several ("A package that failed", above, and
`--streams`, below), or for one file whose function ran in several packages. The summary names it:
`adapter: <import path>.<function>: <verdict>, <verdict>: verdicts differ, read as fail`, each
verdict followed by ` in <file>` when it comes from a stream file, and
`adapter: <file>::<function>: <verdict> in package <import path>, <verdict> in package <import path>: verdicts differ, read as fail`
for one file in several packages. With equal verdicts in several packages, the function has the
worst of what each gives it, in the order of the table below.

**Whose verdict.** The stream names a package and a function, never a file, and it does not say
which files were built. machinery does not try to find out: build constraints, tags, `-race`
and `go list` play no part. A verdict goes to a file only when no other file could have given
it.

1. **The candidates of a package.** Without `--streams` they are the `_test.go` files under
   `trace.tests` in the package's own directory. The import path of a directory is the module
   path of the nearest `go.mod` at or above it (its `module` line), followed by the path of the
   directory below that `go.mod`. That is how `go` reads it for a command of direct mode in a
   layout of modules the rules above accept; nothing more is checked (the threat model, above).
   A package whose import path is not that of a
   directory holding files under `trace.tests` has no candidate. `evidence.adapter.sources` is
   not read in this mode, and an entry whose `package` matches the import path of such a
   directory is refused (above), so that no entry seems to narrow or widen what it cannot. With
   `--streams` the candidates are the files of the entries of `evidence.adapter.sources` whose
   `package` glob matches the package (section 2): every `_test.go` file their `files` globs
   match in the project, under `trace.tests` or not. A package that no entry matches has no
   candidate, also one tested in its own directory.
2. **The files that could have given the verdict** are the candidates and every other Go file of
   a directory that holds a candidate, test file or not, whatever its build constraints. All of
   them are parsed; a file that does not parse may define any name.
3. Of those, when exactly one defines the function as a test function, the verdict is that
   file's, and it counts when that file is a candidate under `trace.tests`. When two or more
   define it, or a file that does not parse is among them, the name is **ambiguous**: a pass or a
   skip counts for none of them, a fail fails each of them that is under `trace.tests`, and the
   summary names the function, the files and what to do. When no file defines it, or only a
   file that is not a candidate under `trace.tests`, the verdict is in no file under
   `trace.tests`: it is counted in the summary, with its package, and used for nothing.

So a test function of the same name in another file of the directory, under any build
constraint and whatever its parameter is written as, never lends its pass to the file that
claims: a claiming file the run did not build has no verdict of
its own, and its claims either meet a function that did not run or an ambiguous name. Renaming
one of the two functions is what makes the claim count. An entry of `evidence.adapter.sources`
is the project's statement that the package is built from its files and from no Go file outside
their directories; machinery cannot check that, and a same-named function copied into the
package from another directory is not seen.

**Which receipt.** For every obligation id of kind `inv`, `sc`, `fm` or `ck` that a test file
claims, and for each attribution of it (a function, or a file), the adapter looks at the
functions the claim belongs to, in this order:

| The functions | The adapter writes |
|---|---|
| One of them failed, an ambiguous name and a function a package failure read as failed included. | A `fail` receipt. |
| One of them has an ambiguous name. | Nothing. |
| One of them was skipped. | A `skip` receipt. |
| One of them did not run. | Nothing: the id stays without evidence. |
| One of them passed only in runs that answered its package from the cache, or its package failed in a run. | Nothing. |
| Every one passed, and each package ended every run of it with a package-level `pass` event. | A `pass` receipt. |

A claim of a whole file that has no test function, and a claim in a file that does not parse,
give nothing, and the file is named in the summary. The policy of the project (below) is not applied here: the receipt records what
happened, and `verify` decides what it proves.

An adapter receipt is a receipt line with three more keys:

```json
{"v":1,"type":"receipt","id":"inv:paid-before-shipped","outcome":"pass","test":"e2e/suite/order_test.go::TestShipNeedsPayment","observed":"go test: TestShipNeedsPayment passed","source":"adapter","granularity":"function","tags":["faults"]}
{"v":1,"type":"receipt","id":"sc:customer-pays-and-the-order-ships","outcome":"pass","test":"e2e/suite/scenario_test.go","observed":"go test: all 2 test function(s) of the file passed","source":"adapter","granularity":"file","tags":[]}
```

- `test` is `<file>::<function>` at function granularity and `<file>` at file granularity,
  the file by its project-relative path, so the claim rule of section 10.2 holds by
  construction.
- `observed` is `go test: <function> passed` (`failed`, `skipped`) at function granularity. At
  file granularity it is `go test: all <n> test function(s) of the file passed`, or
  `go test: <functions> failed` (`skipped`) with the names of those functions, sorted, the
  first five and then `and <n> more`. A fail that is no `fail` event, only a package that
  failed ("A package that failed", above), reads `go test: <functions>: no verdict; the package failed`,
  at file granularity when no function of the file has a `fail` event.
- `source` is `adapter`. A receipt without `source` is one a test emitted.
- `granularity` is `function` or `file`, required with `source: adapter` and refused without.
- `tags` are the tags of `-tags` (above), always present: a list, `[]` when the command gave
  none, or the word `"unknown"` when machinery cannot know them, which is every `--streams` run.
  An event stream of `go test` does not say under which tags it was built, and the flags a
  wrapper passes are not seen: `[]` there would say more than is known.

Receipts are written sorted by id, then file, then function, the file-level one first. They and
the end line are written in one append, after the command has exited, and the end line of an
adapter run has one more key, `adapter_receipts`, the number of lines the adapter wrote right
before it:

```json
{"v":1,"type":"end","exit":0,"finished":"2026-01-02T03:04:09Z","adapter_receipts":6}
```

Those lines are the only ones that may carry `source`, `granularity` or `tags`. A receipt line
that has any of them and is not one of the lines before the end line of a run whose header
names the adapter, appended by a test during the run for instance, is refused (section 12).

**Suites run through a wrapper: `--streams`.** A project may run its tests through a make
target or a script that stages sources and starts `go test` itself, perhaps several times.
machinery cannot add `-json` to that, so the wrapper hands over the streams:

```
machinery run --adapter go-test-json --streams [--stack "<description>"] -- <command> [args...]
```

The command is any command. machinery creates a new empty directory where the system keeps
temporary files (under `TMPDIR` when it is set, which may be inside the project), exports
its absolute path as `MACHINERY_GO_TEST_JSON` (a variable never inherited from the caller, like
those of section 10), runs the command with its standard output untouched, and when it has
exited reads every regular file directly in that directory whose name ends in `.json` or
`.jsonl`, in the order of their names, each as one stream of `go test -json`. Nothing else in
the directory is read. The directory is removed after reading, whatever was found.
`--streams` without `--adapter` is `usage: --streams needs --adapter go-test-json: add it before --streams`.

- **Every rule above holds for every stream**: test functions, attribution, verdicts, the cached
  package, the package that must end with pass. `evidence.adapter.sources` is read in this mode
  only, and it is the only way a verdict finds its file: staged tests run as packages of other
  directories, and machinery did not run what built them ("Whose verdict", above). The command
  is not read: the count and `-overlay` are checked in `GOFLAGS` alone, and the tags are
  recorded as `"unknown"`.
- **A stream must be well formed.** Every line that is not blank is a JSON object with an
  `Action`, the file has at least one, and every package it names has a package-level end
  event, with no event of the package after its last one and no `start` event of the package
  before the end of its current run ("A package that failed", above). Otherwise the run is left incomplete: machinery
  writes no receipt and no end line, prints
  `machinery: run: stream <file>: line <n>: not a go test -json event; the run is left incomplete: fix what writes the stream, delete <receipts> and run again`
  (`stream <file>: no event; ...` for a file without one), `<receipts>` as above, and exits with the command's exit
  code, or 1 when that was 0. `verify` reports the file as an incomplete run (section 12).
- **No stream at all** is a run that ran nothing the adapter can read. The run is complete, it
  has no adapter receipt, and the summary begins with
  `adapter go-test-json: no stream in MACHINERY_GO_TEST_JSON: the command ran nothing the adapter can read, and nothing is proven`.
- **One package in several runs**, in several streams or several times in one ("A package that
  failed", above). A run that answered the package from the cache gives none of its passes; the
  other runs give theirs, and a fail of any run is a fail ("Cached results never count",
  above). A function whose verdicts differ between runs failed ("Which verdict", above), a
  function with a `run` event and no verdict in a run where the package failed included; each
  verdict of the summary line is followed by the file of its run, the same file for two runs of
  one stream. A pass counts only when the package ended with pass in every run of it.
- **What a stream is trusted for.** machinery did not start what wrote the files. Lines typed
  into a file of the directory are read like a stream of `go test`: this mode trusts the
  wrapper exactly as section 10.2 trusts a test that appends its own receipts, and no more can
  be checked.
- **A test binary run directly.** A wrapper may build a test binary and run it under
  `go tool test2json`. Such a stream has no `ok` line of `go test`, but with
  `-p <import path>` it names its package on every event and ends with a package-level `pass`
  or `fail` event, and it is never answered from a cache. It is read under the same rules and
  nothing more is asked of it. What it says is weaker in one respect, and no rule can check
  it: the package is the one the wrapper named, not one `go test` resolved and built. Without
  `-p` the events name no package: their verdicts belong to no file, are used for nothing, and
  are counted in the summary.

The first line of the summary then counts the streams as well:
`adapter go-test-json: <n> stream(s), <n> package(s), <n> test function(s) with a verdict: ...`,
a package or a function seen in several streams counted once, a function whose verdicts differ
as failed.

**What the adapter never proves.** A `tc` id needs a receipt whose `effects` the test observed
(section 12), and a `cx` id a receipt with what the replay did (section 16.10). A verdict of
`go test` carries neither. A claim of such an id in a Go test file writes nothing and is
counted in the summary as `needs a receipt with effects` (`with a replay`). An `fv` id is
proven by the formal stage and a `pf` id by nothing yet.

**The summary.** After the command has exited, on standard error, before the last line of
section 10, each line after `machinery run: `:

```
adapter go-test-json: <n> package(s), <n> test function(s) with a verdict: <n> passed, <n> failed, <n> skipped
adapter: receipts written: <n> pass, <n> fail, <n> skip
adapter: claimed, not run: <n> id(s)
adapter: claimed, verdict not counted: <n> id(s)
adapter: needs a receipt with effects: <n> tc id(s)
adapter: needs a receipt with a replay: <n> cx id(s)
adapter: <n> test function(s) with a verdict are in no file under trace.tests: package(s) <import paths>; <what to do>
adapter: <n> verdict(s) name no package and are used for nothing: run the test binary under go tool test2json -p <import path>
adapter: package <import path>: answered from the cache; a pass replayed from the cache counts nothing: run go test with -count=1
adapter: package <import path>: did not end with pass; no pass is written from it
adapter: package <import path>: <n> test function(s) without a verdict, read as failed: <functions>
adapter: <import path>.<function>: defined in <file>, <file>: ambiguous: a pass or a skip counts for none of them, a fail fails each; rename one of the functions
adapter: <import path>.<function>: <verdict> in <file>, <verdict> in <file>: verdicts differ, read as fail
adapter: <file>::<function>: <verdict> in package <import path>, <verdict> in package <import path>: verdicts differ, read as fail
adapter: <file>: <id>: the adapter does not judge <function>; put the marker above a Test function or at file level
adapter: <file>: no test function; its claims count nothing
adapter: <file>: not Go source that parses; its claims count nothing
```

The first three lines always print; the others only when they count or name something, in this
order, the named ones sorted. `<what to do>` is
`a package's verdicts go to the files under trace.tests in its own directory, and a package built from copies is run through --streams with an entry in evidence.adapter.sources`,
and with `--streams`
`with --streams a package's verdicts go only to the files of its entry in evidence.adapter.sources: add one`.
In the ambiguous line a file that does not parse is followed by ` (does not parse)`. A package that did not end with pass is named when it failed in a
run or did not end with pass in one that gives it verdicts, and with it the functions it made
fail for want of a verdict, sorted, the first five
and then `and <n> more`; a function with a `run` event and no verdict is also counted as
failed in the first line. The packages of the `in no file` line are sorted, the first five and then
`and <n> more`. Packages and functions are counted over the whole stream.
`claimed, not run` counts the ids for which no receipt was written and a function of one of
their claims did not run; `claimed, verdict not counted` the other ids without a receipt whose
claims met a pass replayed from the cache, a package that did not end with pass, or an
ambiguous name. A claim above a function the adapter does not judge counts in neither.

**What the evidence proves: `evidence.adapter`.** `verify`, and every command that judges
evidence, reads an adapter receipt under sections 10.1 and 10.2 like any other. A `fail` fails
its obligation and a `skip` counts for nothing, whatever the policy says. A `pass` counts only
when all of these hold, with the defaults of section 2:

| Condition | Default | Otherwise the reason is |
|---|---|---|
| The kind of the id is `inv`, `sc`, `fm` or `ck`. | fixed | `a <kind> id is never proven by the adapter` |
| `evidence.adapter.allow` lists the kind. | `inv`, `sc`, `fm`, `ck` | `evidence.adapter.allow does not list <kind>` |
| `evidence.adapter.tiers` lists the tier of the id (section 23). | `routine`, `consequential` | `it is in a <tier> area (<areas>): it needs a receipt the test emits after its own assertions`, and for an id in no area `evidence.adapter.tiers does not list routine` |
| The granularity of the receipt is at least the one asked of the id: the strictest of `evidence.adapter.min_granularity` (its `default`), of the entry for the kind and of the entry for the tier, `function` being stricter than `file`. | `file` | `evidence.adapter.min_granularity asks for function granularity` |

An adapter pass that does not count proves nothing and prunes nothing. When the id is not
proven by other receipts, stage `verify` prints, once per id,
`warn   adapter: <id> not counted: a test that names it passed (<granularity> granularity); <reason>`,
for the function-granularity receipt when the id has both; the id is then `unproven` like any
id without evidence. An id that a counted adapter pass proves, and that no receipt emitted by a
test proves on its own, is **proven by adapter**, at function granularity when one of its
counted adapter passes has it and at file granularity otherwise.

**What this does not defend against** is what section 10.2 lists and what the threat model at
the top of this section leaves to review: code or modules planted in the project, a `go` or a
toolchain that does not do what `go test` does, the variables of the run, the build cache, a C
compiler, and a `TestMain` that runs a function of its own under the name of a test. The header
records the go environment of a direct-mode run so that a reader sees what ran; nothing in it is
checked against another run. With `--streams` the adapter trusts the wrapper and the entries of
`evidence.adapter.sources` ("Whose verdict", above): that mode is the way for every suite the
direct mode refuses, and its trust is stated, not checked.

## 11. The owed list

### 11.1 `owed.yaml`

```yaml
version: 1
owed:
  - id: tc:ORDE-8bd6e7
    reason: "effect recordCharge is a noop in the skeleton"
    story: bd-123            # optional: tracker id of the story that will discharge it
```

machinery writes entries sorted by id; a hand-edited file may be in any order. `id` and `reason`
are required. An entry names an obligation, or an `act` id that no code serves yet. Unknown
keys, duplicate ids and unknown ids are an ERROR wherever the file is read. A proof obligation is
not owed (section 23.2): an entry that names a `pf` id is
`ERROR  proof: <id> is in <owed file>; a proof obligation is not owed: it awaits a machine-checked proof, which this version cannot give; remove the entry`
wherever the file is read. A missing file is an empty list.

**A file with a problem.** An owed file that cannot be read, is not YAML or has an unknown
key has no entries to trust, and it is never read as an empty list. Every command that works in
the project then prints, on stderr and with nothing on stdout,
`ERROR  <owed file>: <error>; an owed list that does not parse is not read as empty: fix the file`
and exits 1: `status`, `verify`, `check`, `plan`, `landing` and the rest alike. For an empty
file `<error>` is
`the file is empty; write "version: 1" and "owed: []" in it, or delete it (a missing file is an empty list)`.
A file that parses and has a problem of its own (a version other than 1, an entry without an
id or a reason, an id listed twice) is refused the same way, before any stage runs: one
`ERROR  <owed file>: <problem>` line per problem, in file order (`duplicate id <id>`,
`<id>: reason is required`), then
`ERROR  <owed file>: fix the file; an owed list with a problem is not read`, and exit 1. An
entry that names an id the design does not have is not such a problem: it is a finding of the
stages that compare the list with the design, as before. `machinery adopt`
refuses with the same lines before it writes anything (the first after `machinery: `, exit 1),
`machinery assess` reports the problems of the snapshot's owed file as ERRORs of its `design`
section, and the hooks hold the lines, joined by `; `, as their one finding (section 21.1). machinery itself writes
the file through a YAML encoder, every value included, so what it writes always parses back.

`machinery adopt` writes three reasons (section 17.2, step 3), each followed by
` (<short sha>)`: `unproven at adoption`, `claimed at adoption` for an id a test already claimed,
and `planned at adoption` for an id the project had marked planned, with its `story`. The last
two are read back for as long as the entry keeps them: `machinery plan` wires the evidence of an
entry whose reason begins `claimed at adoption` instead of giving it a story of its own
(section 15.2), and `machinery status` counts an entry whose reason begins
`planned at adoption` as planned (section 14). Rewording the reason of an entry ends that for
the entry. `owed sync` treats these entries like any other: one that is proven is removed.

The owed list is the single ratchet of the method. In a new project nearly everything is owed
right after the skeleton; each story proves some ids and removes them. In an existing project the
owed list is the baseline of what was never proven.

### 11.2 `machinery owed sync`

Reads the evidence exactly as `verify` does (section 12). While a receipts file is refused
(section 10.2) or a run is incomplete, it writes nothing, whatever the other runs prove: it
prints the finding `verify` has for each such file as `ERROR  <finding>` and exits 1. A receipt
whose test claims nothing proves nothing here either. Otherwise it rewrites `owed.yaml`:

- removes every entry whose id is proven;
- unless `--prune-only`, adds every obligation that is neither proven nor listed, with the reason
  `unproven at <short sha>`;
- keeps existing reasons and story fields.

It prints `owed: <n> kept, <n> added, <n> removed` and exits 0. It refuses with
`ERROR  no complete run valid for <short sha>: run the tests through machinery run first` (exit 1)
when there is no complete run valid for the current commit, and when the owed file has problems.
An owed file that does not parse stops it first, as it stops every command (section 11.1).
`machinery owed list` prints one line per entry, `<id>  <reason>` followed by two spaces and the
story when there is one, then `owed: <n> entries` (`1 entry`). Because of section 10.1, the sequence
`machinery run`, `machinery owed sync`, commit, `machinery verify` works without running the
tests twice.

### 11.3 Stage `ratchet` (`machinery check --base <git-ref>`)

Compares the current state with the same files at `<git-ref>` (read with `git show`). Nothing
that was established at the base may quietly stop being established:

| Rule | Finding |
|---|---|
| `proven-to-owed` | An id that was an obligation at the base and was not owed there is owed now: `ERROR  ratchet: <id> was proven at <ref> and is owed now`. New obligations may be owed. |
| `scope-narrowed` | A scope glob of the base is missing now: `ERROR  ratchet: scope narrowed: <glob>`. |
| `stage-disabled` | A stage that applied at the base no longer applies because its config key is gone: `ERROR  ratchet: stage disabled: <stage> applied at <ref> and <key> is gone now`. Deleting a block of `machinery.yaml` does not switch a gate off. Stage `machines` applies with `design.machines` or `import.machines`, and stage `failure-modes` with `design.failure_modes` or `import.failure_modes`: moving from one to the other keeps it on. |
| `obligation-removed` | An obligation of the base the design no longer declares: `warn   ratchet: obligation removed since <ref>: <id>`. A design may retire a promise; it may not do so unseen. |
| `demoted` | An `act` or obligation id of the base (owed or proven there) is proposed now (section 4.1): `ERROR  ratchet: demoted to proposed: <id>; proposed is for candidates, not a way out of an obligation`. |
| `retired` | An `act` or obligation id that was proven at the base (not owed there) is retired now: `warn   ratchet: retired: <id> (<reason>)`. Retiring an id that was owed at the base prints nothing; the entry is the record. |
| `tier-lowered` | An id has a lower tier than at the base: `ERROR  ratchet: tier lowered: <id> <old> -> <new>`, unless the risk file records the lowering under `lowered:` (section 23), which prints as `warn` with the recorded reason. |
| `baseline-grew` | An architecture or tests baseline has an entry the base did not have: `ERROR  ratchet: baseline grew: <rule> <subject>`, unless the entry's file is newly in scope (it matches a scope glob the base did not have and was outside the base's scope). |
| `spec-weakened` | A formal baseline acceptance (section 16.6) that the base did not have prints as `warn   ratchet: weakened since <ref>: <spec>: <change> (<reason>)`. |
| `review-off` | A review gate (section 15.10) that was required at the base is off now: `warn   ratchet: review switched off: review.<kind> was required at <ref> and is off now`. A project being adopted switches a gate off on purpose; the ratchet makes the change seen. |
| `adapter-widened` | The policy `evidence.adapter` (section 10.3) lets adapter evidence prove an id it could not prove at the base, or at a weaker granularity. Policies are compared by what they admit, not by how they are written: for each tier and each kind among `inv`, `sc`, `fm` and `ck`, the least granularity an adapter pass needs to prove such an id, `function`, `file`, or `never` (the kind is not in `allow` or the tier not in `tiers`). One finding per tier and change, `<kinds>` in the order `inv`, `sc`, `fm`, `ck`, separated by `, `: `ERROR  ratchet: adapter widened: <tier> <kinds> ids: never -> <granularity>; an id of a critical or vital area needs a receipt its test emits after its own assertions` for a critical or vital tier, and `warn   ratchet: adapter widened: <tier> <kinds> ids: <old> -> <new>` for every other change that admits more (`never -> function`, `never -> file`, `function -> file`, in this order within a tier, the tiers from routine to vital). For `evidence.adapter.sources`, compared as pairs of a `package` glob and a `files` glob: `warn   ratchet: adapter widened: evidence.adapter.sources: <package> <- <files> added`, one per package glob in the order written, with the file globs the base did not map from it. A base without the block had the defaults. Narrowing raises nothing, and neither does a policy written another way that admits the same. |
| `vital-weakened` | A vital area of the base (section 23.1) states less now: `ERROR  ratchet: vital weakened: <area>: <what>; record it under lowered: in the risk file with a reason`, unless the risk file records that area and that `<what>` under `lowered:` (section 23), which prints as `warn   ratchet: vital weakened: <area>: <what> (<reason>)`. Adding or strengthening raises nothing. |

The `<what>` of `vital-weakened`, one finding per change, in the order the base's area lists its
parts (properties, then assumptions, then `core.paths`, then `core.entry`):

| Change against the base | `<what>` |
|---|---|
| No area of that name is in the risk file | `area removed` |
| A property of that name is gone | `property <name> removed` |
| A property's `statement` differs (white space at either end ignored) | `property <name>: statement changed` |
| A property's `kind` differs | `property <name>: kind <old> -> <new>` |
| An id of a property's `over` is gone | `property <name>: over <id> removed` |
| An assumption of that name is gone | `assumption <name> removed` |
| An id of an assumption's `evidence` is gone | `assumption <name>: evidence <id> removed` |
| A glob of `core.paths` is gone | `core path <glob> removed` |
| An entry of `core.entry` is gone | `core entry <entry> removed` |
| A `not_vital` entry the base did not have matches an id that a vital area held at the base | `not_vital <id> added` (the area is the first vital area of the base that held it) |

Parts are matched by name: a renamed property is one removed and one added. An area that is still
in the risk file at a lower tier is a lowering, which `tier-lowered` holds: `vital-weakened` says
nothing more about it. A removed area is one finding; its parts are not listed again. An id
added to `over` or `evidence`, a new property, assumption, glob or entry, and a new vital area
raise nothing. The ids a `not_vital` entry is matched against are the ids of the design now and
the obligations and actions of the base.

`checked:` counts obligations, scope globs, tiered ids and baseline entries compared, and, when the
base has a vital area, its vital areas. An unknown
ref is a usage error. A base at which the project has no `machinery.yaml` established nothing:
the stage is `not applicable: no machinery.yaml at <ref>: nothing was established there`.

The paths of the obligations, the owed list, the risk file and the baselines at the base come
from the base's own `machinery.yaml`.

## 12. Verify

Stage `verify` joins obligations to evidence.

Evidence is every receipts file in the receipts directory that is valid for the current commit
(section 10.1) and that `machinery run` produced and nothing changed since (section 10.2). When
the working tree has uncommitted changes outside the owed file, the receipts directory and the
property records, the evidence does not describe what is on disk: without `--allow-dirty` that
is `ERROR  working tree has uncommitted changes`. A file without an `end` line is
`ERROR  incomplete run: <file>` and its receipts are not used. A file whose header says
`dirty: true` is used only with `--allow-dirty`, and then the stage prints
`warn   dirty evidence: not valid for acceptance`. A run whose `end` line has a non-zero exit
prints `warn   run exited <n>: <file>`; its receipts are still used.

Per obligation:

| State | When |
|---|---|
| `failed` | At least one `fail` receipt. For a `cx` id: a receipt whose `replay` differs from the trace's `expect`. |
| `proven` | Not failed, at least one `pass` receipt, and for a `tc` id at least one `pass` receipt whose `effects` equal the case's effects as a set. For a `cx` id: a receipt whose `replay` equals the trace's `expect`. For an `fv` id: the formal stage finds nothing against the area (section 16.9); no receipt is involved. |
| `partial` | A `tc` id that is not failed, has a `pass` receipt, but no `pass` receipt observed every effect. |
| `unproven` | Anything else. `skip` receipts never count. |

A `pass` a run adapter wrote (`source: adapter`, section 10.3) is a `pass` receipt of this table
only when the policy `evidence.adapter` lets it count for its id; its `fail` always is a `fail`
receipt. An id proven by such passes alone is proven by adapter, at function or at file
granularity, and is counted apart.

Findings:

- `ERROR  receipts: <file>: not produced by machinery run; ...`,
  `ERROR  receipts: <file>: changed after the run; ...` and
  `ERROR  receipts: <file>: recorded by machinery run and no longer present; ...` (section 10.2,
  with the full text).
  Nothing in the file is used.
- `ERROR  receipt: <id>: <test> does not claim it; <what to do>` (section 10.2). A `pass` is not
  used; a `fail` still fails its obligation.
- `ERROR  malformed receipts file: <file>: line <n>: <what>`; a receipt without `id` or without
  `test` is one (`<id>: receipt without test: name the test file in test`), and so is a receipt
  whose `source` is not `adapter` (`<id>: source must be adapter or absent, got "<value>"`),
  whose `source` is `adapter` and whose `granularity` is neither `function` nor `file`
  (`<id>: granularity must be function or file, got "<value>"`), that has a `granularity` and
  no `source` (`<id>: granularity without source: adapter`), or that has `source`, `granularity`
  or `tags` and is not one of the lines the adapter wrote before the end line of an adapter run
  (section 10.3:
  `<id>: source, granularity and tags are written only by the run adapter: a test that emits receipts leaves them out`).
  Such a line is not read. A file whose first
  line is not a run header is not a receipts file and is ignored.
- `ERROR  unknown id in receipt: <id>` (an `act` id is not an obligation), and for a proposed or
  retired id (section 4.1) `ERROR  proposed id in receipt: <id>; a proposed id yields no obligation:
  drop the receipt, or take the id out of <status file>` (`retired`). For a proof obligation
  (section 23.2) it is `ERROR  proof id in receipt: <id>; tests do not discharge a proof obligation: drop the receipt`.
- `ERROR  unexpected effect: <id> observed <name>` when a receipt lists an effect the case does
  not have.
- `ERROR  failed: <id> (<test>)`.
- `ERROR  counterexample reproduced on the real system: <id>: a real bug; fix the code, or record
  the run as a finding with a story` and `ERROR  counterexample did not reproduce: <id>: the model
  is presumed wrong; fix the model, not the property` (section 16.10).
- `ERROR  unproven: <id>` for an obligation that is `unproven` or `partial` and not owed.
- `warn   adapter: <id> not counted: a test that names it passed (<granularity> granularity); <reason>`
  for an id that is not proven and has an adapter pass the policy does not let count
  (section 10.3, with the reasons). Owed or not, the id is unproven.
- `warn   adapter environment: <file>: <what the header recorded>`, once for every adapter run
  that is used (section 10.3, with the text). It is a record: environments that differ between
  runs at the same commit are no finding. With `--story` it prints for every such run.
- `ERROR  stale owed entry: <id> is proven` for an owed id that is proven. Run
  `machinery owed sync --prune-only`.
- `ERROR  unknown owed id: <id>`.

When no complete run exists for the current commit the stage is
`not demonstrated: no evidence at <short sha>`.

`checked:` counts obligations, proven, partial, owed and receipts. `receipts` is the number of
receipt lines read from the runs that are used; a `pass` that is not used is among them and has
its ERROR. When an id is proven by adapter (section 10.3), the count of the proven is followed
by how many of them that is, each part only when it counts something:
`14 proven, 3 by adapter at function granularity, 2 by adapter at file granularity, 0 partial`.

When the risk file has a vital area, the stage also prints, so that its evidence never reads as
more than it is (section 23.2): `warn   proofs awaiting: <n>`, the number of proof obligations;
`warn   vital: <id> proven (by test and model; proof awaited)` for every id a property is over
that the evidence proves; and `warn   assumed, unproven: <area> <name>: <statement>` for every
assumption whose evidence is owed. With `--story` they are limited to the vital areas of the
story's ids and to those ids.

This is how a noop shows up. A skeleton moves state for real but its effects do nothing, so its
tests observe the target state and no effects: the case is `partial`, therefore owed. When a story
makes the effect real and the test observes it, the case becomes `proven` and leaves the list.

`--story <id>` judges only the ids of that backlog story (section 15.1); every other obligation is
ignored except for `failed` ones, which always print. A refused receipts file, an incomplete
run and `receipt: <id>: <test> does not claim it` (section 10.2) always print too, whatever id
they name. The `adapter: <id> not counted` warning prints for the ids of the story only.

### 12.1 Targeted runs: `machinery affected`

Testing shifts right: per change, run what the change can break; run everything once, at the end
(section 12.2).

`machinery affected --base <ref>` prints, one per line and sorted, the obligation ids touched by
the difference between `<ref>` and the working tree:

1. ids whose entry in `obligations.json` is new or differs from the base (title, effects);
2. ids named by a `machinery:verifies` tag, or claimed by a verifies or scenarios marker
   (section 8.1), in a changed test file;
3. for each `act` id named by a `machinery:serves` tag or a serves marker in a changed code file:
   every `tc` id of
   the entity's lifecycle machine whose trigger is that action's event, and every `inv` id the
   action `preserves`;
4. every `tc` id of a machine whose file changed.

Rule 1 compares the design as it is now with `obligations.json` at the base, so it is right
before `machinery gen` ran. In rule 3 the cases of an action are its `on:<action>` transitions,
their guard-clause cases and the `ignore:<action>` cases.

With nothing touched it prints nothing and exits 0.

`machinery run --affected --base <ref> -- <command>` computes the same list and exports it as
`MACHINERY_ONLY`; when the list is empty it runs nothing and exits 0. `--affected` and `--base`
go together. `machinery verify --affected --base <ref>` reports `unproven` only for those
ids. A targeted verify is a check for one change; only a full `verify` supports a landing.

### 12.2 The end gate: `machinery landing`

```
machinery landing --base <ref> [--stack "<description>"] [--repair] [-- <command> [args...]]
machinery landing --clear-red --reason "<text>"
```

Runs, in order and stopping at the first that does not pass: `machinery check --base <ref>`;
`machinery formal check --verify` when the project has formal models (section 16.9); the
project's whole end-to-end suite through `machinery run` (the command after `--`, else
`commands.e2e`); `machinery verify`; and with `--probe` (optionally `--probe-max <n>`) stage
`probe` over the proven transition cases (section 20). It prints the stage blocks of each and one final line:

```
landing: <passed|failed> at <short sha> (attempt <n>, fix-forward <k> of 2)
```

The whole suite runs once, at the head that is ready, not per story. machinery counts attempts
per branch in `<git common dir>/machinery/landing.json`: an attempt at a new head after a failed
attempt is a fix-forward. After two fix-forwards a further failure also prints
`stop: two fix-forwards used; return the failing stories to the backlog`. The count is a report:
it never turns a passing landing into a failing one. A passed landing resets it. A repeated
attempt at the same head raises the attempt number only. With neither a command after `--` nor
`commands.e2e` it is a config error and nothing is counted.

**A red integration branch.** A landing that fails writes `<git common dir>/machinery/red.json`,
which every worktree of the repository shares:

```json
{"version": 1, "commit": "<full sha>", "base": "<the --base ref>", "stages": ["<stage>"]}
```

`commit` is the head the landing failed at. `stages` lists, in the order they ran, the stages
whose outcome was failed or not demonstrated; the end-to-end suite is `run` when its command did
not start or exited non-zero. A later failure rewrites the file; a landing that passes removes it.
While the file exists the integration branch is red, and two commands refuse to move work onto
it:

- `machinery landing` at a head other than the marker's commit (another attempt at the red head
  is allowed);
- `machinery accept --close` (section 15.6).

Both print `ERROR  the integration branch is red since <short sha> (<stages>): repair or revert
first` and exit 1 before anything runs or is written, unless given `--repair`. With `--repair` a
landing runs as usual (a fix-forward is still counted), and `accept --close` first appends
`machinery:repair <short sha of the red commit>` to the story's notes. `--repair` on `accept`
needs `--close`; without a red marker it changes nothing.

`machinery landing --clear-red --reason "<text>"` removes the marker after a revert, prints
`red: cleared <short sha> (<stages>): <reason>`, or `red: none` when there is no marker, and
exits 0. It runs nothing else, takes no `--base`, and an empty reason is a usage error.
`machinery status` prints the red line first (section 14).

## 13. Proof

```
machinery proof [--ids <id>,<id>...] [--prefix <id prefix>] [--story <id> [--post]]
                [--limit "<text>"]... [--learning "<text>"]... [--observation "<kind>: <text>"]...
```

Prints a block for the selected obligations (all obligations when no selector is given), from the
evidence at the current commit:

```
PROOF
commit:   <full sha>
tier:     <highest tier among the selected ids> (<areas>)
command:  <command of the run, one line per run>
adapter environment: <file>: <what the header of an adapter run recorded>
result:   <p> passed / <f> failed / <s> skipped, against <stack>
adapter:  <n> proven by adapter at function granularity, <n> proven by adapter at file granularity
criteria:
  <id> -> <test> -> pass: <observed>
  <id> -> <test> -> FAIL: <observed>
not proven:
  <id> (<state>)
  assumption <spec> <A-id>: <text>
limits:
  <a claim the change makes that is still unproven>
learnings:
  <a rule the next story or the planner should know>
observations:
  <kind>: <something seen outside this story's scope>
```

`--ids` and `--prefix` select the union; each must match something. A proof obligation
(section 23.2) has no evidence to select:
`usage: --ids: <id> is a proof obligation: it awaits a machine-checked proof and has no evidence to show`. Without evidence the block
has no `command:` line and `result:   no evidence at <short sha>`. The `, against <stack>` part is
left out when no run names a stack. An owed id prints under `not proven` as
`<id> (<state>, owed)`. When a receipts file is refused (section 10.2) or a run is incomplete
(section 12), no count stands: the line is
`result:   <n> receipts file(s) refused: delete them and run the tests through machinery run`
and the exit code is 1, whatever the other runs prove. `--post` posts that block as it is.

An `adapter environment:` line follows the `command:` line of every adapter run, with the text
of section 10.3, whatever ids are selected.

The `adapter:` line prints only when a selected obligation is proven by adapter (section 10.3),
with the parts that count something: the block never shows such an id as if its test had
emitted the receipt. When an id has receipts of both kinds, its line shows one a test emitted. Its `pass` line under `criteria` ends with
` (by adapter at <granularity> granularity)`, before the words of a vital property. A selected
obligation that is not proven and has an adapter pass the policy does not let count prints under
`not proven` as `<id> (<state>, adapter pass not counted)`, after `owed` when it is owed.

`criteria` lists one line per selected obligation that has a receipt, sorted by id; an `fv` id
prints as `<id> -> formal -> pass: <n> run(s), <n> mutation(s) detected`. The `pass` line of an
id that a property of a vital area is over ends with ` (by test and model; proof awaited)`
(section 23.2). `not proven` lists the selected obligations that are partial, unproven or owed,
every unproven assumption of a formal model that covers a selected id, and, for every vital area
a selected id belongs to (the ids of the story, with `--story`), each of its proof obligations
as `<pf id> (awaiting proof)` and each assumption whose evidence is owed as
`assumed, unproven: <area> <name>: <statement>`. Exit 0 when every selected obligation is proven,
1 otherwise, 2 when a selector matches nothing; the lines of a vital area do not change it.

`limits`, `learnings` and `observations` are written by the author, one line per `--limit`,
`--learning` and `--observation` (each repeatable), in the order given; a part without one prints
`  none`. A newline inside a value becomes a space. An observation is `<kind>: <text>`, the kind one
of `bug`, `security`, `debt`, `perf` and `doc`; a value of another shape is a usage error
(`usage: --observation "<value>": write <kind>: <text>, the kind one of bug, security, debt, perf, doc`),
and so is an empty value.

`--story <id>` selects the ids of a backlog story. `--post` appends the block to the story's notes
after a line `machinery:proof <full sha> <actor>` (section 15.1), and after the block, in the same
note, one `machinery:learning <text>` line per learning and one `machinery:observation <kind>
<text>` line per observation: the lines `machinery learnings` and `machinery observations` read
(section 15.8).

## 14. Status

```
red: since <short sha> (<stages>): repair or revert first
project: <root directory name>
obligations: <n> (tc <n>, sc <n>, inv <n>, fm <n>)
tiers: routine <n>, consequential <n>, critical <n>, vital <n>
ids: <n> proposed, <n> owed, <n> proven, <n> retired
proven: <n> (<n> by adapter at function granularity, <n> by adapter at file granularity)  partial: <n>  failed: <n>  owed: <n>  unproven and not owed: <n>
adapter, not counted: <n> (<kind> <n>, ...)
proofs awaiting: <n>
vital: <id> proven (by test and model; proof awaited)
assumed, unproven: <area> <name>: <statement>
owed without a story: <n>
owed, claimed by a test, no receipt yet: <n> (<kind> <n>, ...)
owed, no test names it: <n> (<kind> <n>, ...)
owed, planned: <n> (<kind> <n>, ...)
owed, proven by a receipt, still listed: <n> (<kind> <n>, ...)
owed, actions: <n> served by code, <n> not served
refused: <receipts file>: <why: the text of section 10.2 before the semicolon, or incomplete run>
evidence: <short sha> <clean|dirty>, <n> run(s), <n> receipt(s)
```

The `red` line prints only while the integration branch is red (section 12.2). The first count
line lists only kinds that occur, in the order `tc, sc, inv, fm, fv, cx, ck`. The `ids` line
counts every `act` and obligation id by its status (section 4.1): `owed` the ids the owed list
holds, `proven` the other ids that yield an obligation, which is what `verify` holds them to; the
next line says what the evidence shows. After its count of the proven it says, in parentheses,
how many of them are proven by adapter (section 10.3), by granularity, each part only when it
counts something and no parentheses when neither does. The line `adapter, not counted` prints
only when it counts something: the ids that are not proven and have an adapter pass the policy
`evidence.adapter` does not let count, by kind.
The `tiers` line counts obligations by the tier of their area and is omitted when there is no
risk file; its `vital` part, and the three lines after the `proven` line, print only when the
risk file has a vital area (section 23.2): the number of proof obligations, one `vital` line per
id a property is over that the evidence proves, and one `assumed, unproven` line per assumption
whose evidence is owed, each group sorted.

The `owed,` lines print only in an adopted project: while the owed list holds an entry
with a reason `machinery adopt` wrote (section 11.1). They count the owed ids the design knows,
each obligation on the first line that applies:

| Line | Counts |
|---|---|
| `planned` | An entry planned at adoption (section 11.1), whatever names or proves it. |
| `proven by a receipt, still listed` | An id the evidence proves at the current commit (section 12): its entry is stale, and `owed sync` removes it. |
| `claimed by a test, no receipt yet` | An id a test file names, read statically as stage `trace` reads tags and markers (section 8), that no valid receipt proves. |
| `no test names it` | Every other one. |

The `claimed` and `no test names it` lines always print, in that order, then `planned`, `proven`
and `actions`, each only when it counts something. The kinds are listed in the order
`tc, sc, inv, fm, cx, ck`, only those that occur, and a count of zero prints as `0`. An owed
`act` id is in none of the four: an action is served, not claimed, and the `actions` line
counts the owed ones by whether code serves them. An `fv` id is proven by the formal stage
and is in no line. A claim is not evidence: the `claimed` line says where a receipt is all that
is missing.

One `refused` line prints per receipts file that section 10.2 refuses and per incomplete run
(section 12), after the `owed,` lines. When there is no evidence the line is
`evidence: none at <short sha>`. When
`backlog.tool` is set and the backlog can be read, one line follows per open intake item, sorted by
id: the summary line of `machinery intake status` (section 15.7),
`intake <id>: <n> of 7 steps done; next: <step>`; then the latest review of each kind (section 15.10):
`review backlog: <verdict> at <12 hex> by <actor>[, <n> issue(s)]` for the latest backlog review, and
`review <design|intake> <intake id>: <verdict> at <12 hex> by <actor>[, <n> issue(s)]` for the latest
review of that kind on each open intake item, by intake id, the 12 hex being the contract hash
the review was for; and then, when a question is open (section 15.9),
`questions: <n> open, <k> held`. Exit 0 unless the config is invalid (2) or the owed file does
not parse (1, section 11.1).

## 15. Backlog and scheduling

The backlog is the plan and the record of delivery. machinery bridges the design to it: what is
owed becomes stories, the dependency graph becomes parallel waves, and a story is accepted with
the formality of its tier. The backlog tool is `bd`; these commands apply when `backlog.tool` is
set, and are a config error otherwise.

### 15.1 The adapter and story tags

machinery talks to the backlog only by running the `bd` executable (`tools.bd`) in the project
directory and reading its JSON output. The backlog is never committed and machinery never writes
its files directly. When `bd` cannot be run, a command that needs it exits 2 with
`config: bd not runnable: <reason>`.

A story carries machine-readable lines, anywhere in its description or notes. They use the tag
syntax of section 8:

```
machinery:ids tc:ORDE-2e883d tc:ORDE-70d94f
machinery:scope internal/api/** e2e/orders_test.go
machinery:proof <full sha> <actor> [session=<session>] [agent=<agent>]
machinery:prediction commit=<full sha> actor=<actor> [session=<session>] [agent=<agent>] observation="<text>" expect="<text>"
machinery:verdict <accept|reject> commit=<full sha> actor=<actor> [session=<session>] [agent=<agent>] observed="<text>"
machinery:intake
machinery:intake <intake id>
machinery:intake-base <full sha> <contract hash> [actor=<actor>] [session=<session>] [agent=<agent>]
machinery:intake-planned <full sha> <contract hash> [actor=<actor>] [session=<session>] [agent=<agent>]
machinery:intake-design
machinery:no-design-change <reason>
machinery:capstone
machinery:demo <command>
machinery:learning <text>
machinery:observation <kind> <text>
machinery:observation-filed <item id> <kind> <text>
machinery:repair <short sha>
machinery:decision <actor>
machinery:decision-source <path>#<12 hex>
machinery:supersedes <decision id>
machinery:vetoable
machinery:confirmed <actor>
machinery:overruled <actor>
machinery:question <actor> [session=<session>] [agent=<agent>]
machinery:option <n> <text>
machinery:recommend <n>
machinery:default <text>
machinery:about <topic> <topic>
machinery:amount <number> <unit>
machinery:story <id>
machinery:answer <n|refuse|hold> actor=<actor> [clause=<clause id>] [session=<session>] [agent=<agent>] reason="<text>"
machinery:review <backlog|design|intake> <approved|rejected> commit=<full sha> contract=<contract hash> actor=<actor> [session=<session>] [agent=<agent>] [stories=<id>,<id>] [issue1="<text>"]...
```

- `machinery:ids` lists the design ids the story delivers. Several lines add up.
- `machinery:scope` lists the path globs the story writes.
- The tier of a story is the highest tier among its ids (section 23). The label `tier:<tier>` on
  the story mirrors it for people and for routing; when label and ids disagree the ids win.
- The actor is `--actor`, else the environment variable `BEADS_ACTOR`, else `git config user.name`.
- The session is the environment variable `MACHINERY_SESSION`, white space at either end removed.
  When it is set and not empty, `proof --post`, `predict` and `accept` write `session=<value>` on
  their line, and no session part otherwise; so do `intake new` and `intake plan --apply` on the
  intake lines, `review`, `ask` and `answer --delegate`. A value that is not one word (it holds white space
  or a `"`) is a usage error, before anything is written:
  `usage: MACHINERY_SESSION must be one word, got "<value>"`. In Claude Code the `session-start`
  hook exports the harness's session id as `MACHINERY_SESSION` (section 21.1); another harness sets
  it or leaves it unset. The last words of a proof line are its session and its agent when they
  start with `session=` and `agent=`; the words between the sha and them are the actor. The
  question line is read the same way: the words after `machinery:question` are the actor.
- The agent is the environment variable `MACHINERY_AGENT`, read like the session: white space at
  either end removed, one word or else `usage: MACHINERY_AGENT must be one word, got "<value>"`
  before anything is written, and `agent=<value>` on the same three lines (after the session)
  only when it is set and not empty, and on the other lines that write the session. It names the
  worker context within a session. A harness
  gives the subagents of a session the session's id, so the session alone cannot tell a builder
  from a verifier run as a subagent: whoever spawns a worker sets `MACHINERY_AGENT` for it to a
  value no other worker of the session has. `machinery brief` prints that value for a builder
  and a verifier (section 15.5): `<story id>-<role>`.
- `machinery:intake` with nothing after it marks an intake item; `machinery:intake <id>` marks a
  story of that intake. The other intake lines are described in section 15.7, the review line in
  section 15.10.
- `machinery:repair` records that a story was closed while the integration branch was red, as
  its repair (section 12.2). The learning, observation and decision lines are described in
  section 15.8, the question and answer lines in section 15.9.

### 15.2 `machinery plan`

Prints one story per group of owed ids, as Markdown (one `##` heading per story with `### Type`,
`### Priority`, `### Labels`, `### Description` and `### Acceptance Criteria` sections), or with
`--json` as a `bd create --graph` plan.

Groups:

- owed `tc` ids are grouped by machine and trigger event: one story per `(machine, event)` holds
  every case about that event (the `on:<event>` transition cases, the `ignore:<event>` cases,
  the guard-clause cases of those transitions and the refusal cases whose handler names the
  event); one story per machine named `internal transitions` holds the cases with no caller
  event (`after`, `always`, `onDone`, `onError`, `stateDone`);
- one story per owed `sc`, `fm`, `inv`, `cx` and `ck` id;
- one story per owed `fv` id, titled `Model <area> (<tool>)`;
- one story `Serve <Entity> actions` per entity with owed `act` ids.

**Wire evidence.** An id a test already claimed when the project was adopted needs no story of
its own: its test exists, and what is missing is the receipt. An owed entry whose reason begins
`claimed at adoption` (section 11.1), and whose id a test file still claims (read as stage
`trace` reads it, section 8), leaves the groups above and joins one story per test file,
`Wire evidence <file>`, that lists the ids the file must emit receipts for. An id several files
claim goes to the first of them in path order, and its acceptance line names them all. When
more test files would get a story than `plan.wire_max_files` (section 2, default 20), the
stories are one per directory instead, `Wire evidence <directory>/`, each holding the ids of
the files directly in it. Every other owed id is grouped as above: an id no test names, an id
owed for any other reason though a test claims it now, an id claimed at adoption whose claim is
gone, and an id planned at adoption, which already carries the project's story. A project in
which adopt found nothing claimed is planned exactly as before.

Every story has type `task` and a priority from its tier: 2 routine, 1 consequential, 0
critical and vital.

A story title is `Prove <machine> <event>` or `Prove <id>` unless given above. A wire-evidence
story has, under `Excludes`, a second line:
`- No new test: the tests that claim these ids exist; this story makes them emit receipts.`, and
each of its acceptance lines starts `- <id>: claimed by <file>[, <file>...]; ` before the text of
the table below. Its labels
are `tier:<tier>` and `machinery`. Its acceptance criteria (the backlog's own field) say that
each listed id is proven by `machinery verify` at the story's commit and removed from
`owed.yaml`, and for a consequential, critical or vital tier that `machinery story <id>` passes.

**Story text.** The description of every story a plan creates, here and in section 15.7, has
five sections in this order, each a header line followed by its lines, one blank line between
sections:

```
Intent:
<what the ids prove, one line>
Requirement <intake id>: <the intent of the intake>

Acceptance:
- <id>: <what is observed>

Ids:
machinery:ids <id> <id>...
machinery:intake <intake id>

Excludes:
- No design change: a design change is its own story.

Needs:
- <title of the blocking story>: <the output needed>
```

The `Requirement` and `machinery:intake` lines are written only for a story of an intake.
`Acceptance` has one line per id, in id order. `Needs` has one line per edge, sorted by title,
and `- none` without one. No line outside the `Ids` section starts a tag (section 15.1).

| Story | Intent line |
|---|---|
| `Prove <machine> <event>` | `Machine <machine> answers <event> on the running system as its table says.` |
| `Prove <machine> internal transitions` | `Machine <machine> makes its internal transitions on the running system as its table says.` |
| `Prove sc:<slug>` | `The scenario runs end to end on the running system.` |
| `Prove inv:<id>` | `The invariant holds on the running system.` |
| `Prove fm:<id>` | `The failure gets its required behavior on the running system.` |
| `Prove cx:<spec>:<run>` | `The counterexample of the formal model cannot happen on the running system.` |
| `Prove ck:<checker>:<name>` | `The external checker holds the obligation for this design.` |
| `Model <area> (<tool>)` | `The formal model of <area> answers its questions and is proven.` |
| `Serve <Entity> actions` | `Code serves the actions of <Entity> through their operations.` |
| `Wire evidence <file>` | `The tests of <file> emit a receipt for each id they already claim.` |
| `Wire evidence <directory>/` | `The tests under <directory>/ emit a receipt for each id they already claim.` |

| Id | Acceptance line |
|---|---|
| `tc`, transition case | `- <id>: <case title>, effects: <effects, or none>; observed on the running system` |
| `tc`, ignore case | `- <id>: refused: <event> in <state> changes nothing and has no effect (<reason>)` |
| `tc`, refusal case | `- <id>: refused: in <state>, with every guard of <handler> false, nothing changes and no effect happens` |
| `tc`, clause case | `- <id>: refused: with only <clause> of <guard> false, <source> <trigger> does not fire` |
| `inv` | `- <id>: holds on the running system: <statement>` |
| `sc` | `- <id>: runs end to end on the running system: <name>` |
| `fm` | `- <id>: <failure> Required behavior (<required>): <behavior>`; with `required: refuse` it starts `- <id>: refused: ` |
| `fv` | `- <id>: machinery formal check proves the model: <why>` |
| `cx` | `- <id>: refused on the running system: the replay of <spec> run <run> keeps <violated property>` |
| `ck` | `- <id>: the checker's evidence for this design holds it: <title>` |
| `act` | `- <id>: served by code and reachable through its operation: action <action> of <Entity>` |

Edges. An edge means that one story needs a specific output of another:

| From (blocked) | To (blocker) | The output needed |
|---|---|---|
| Every `Prove` or `Wire evidence` story with an id in a critical or vital area | The `Model <area>` story | The formal model the tier requires. |
| `Prove fm:<id>` whose subject is `invoke:<machine>.<src>` | The `internal transitions` story of that machine | The invoke path the failure is induced on. |
| `Prove cx:<spec>:<run>` | The `Prove` stories of the cases its trace steps name | The transitions the replay drives. |

Edges are only created between stories of the same plan, except that the capstone of an intake
is also blocked by the stories its intake already has (section 15.7).

`machinery plan --apply [--parent <id>]` creates the stories and edges in the backlog, skips a
group whose ids all carry a `story` that exists in the backlog, writes the new story ids into the
`story` field of the owed entries, and prints `plan: <n> created, <n> existing, <n> edge(s)`.
When the plan has a wire-evidence story a second line follows, counting those of the plan,
created or not:
`plan: <n> wire-evidence stories for <n> claimed id(s), one per test <file|directory>`
(`1 wire-evidence story`).

**A story the project supplied.** An entry planned at adoption carries the story of the
project's own mark, and plan never replaces it. Such an id is in no group: the plan has no
story for it, printed, with `--json` or applied, so no count of stories includes one that
`--apply` would not create for this reason, and the other ids of what would have been its group
keep their story. When its story is in the backlog nothing is said. When it is not, as when
the project's ids come from another tracker, `--apply` prints, after the lines above, one line
per such id: `plan: <id>: external story <story>, not in the backlog; nothing created`. The
entry keeps its story on every run. `plan --link <story> <id>` to another story is a usage error:
`<id> carries the story <story> the project planned it under; plan never replaces it: edit <owed file> to change it`.
Printing the plan does not read the backlog, so a printed plan still shows a group whose ids
all carry a story that `--apply` may find in the backlog and skip.
`machinery plan --link <story> <id>...` attaches ids to an existing story instead: it appends a
`machinery:ids` note to the story, sets the `story` field of those owed entries and prints
`plan: <n> id(s) linked to <story>`.

Output is sorted and deterministic. With nothing owed it prints nothing and exits 0.

### 15.3 `machinery waves`

Reads every story that is not closed, not deferred, not an epic, not an intake item (section
15.7), not a decision (section 15.8) and not a question (section 15.9), with its blocking edges, and prints the layers of the graph: wave 1 holds the stories with no open blocker, wave 2 those
blocked only by wave 1, and so on. Stories of one wave can run in parallel.

```
contract: <first 12 hex of the contract hash> (<n> design file(s))
wave 1
  <story id>  <tier>  <title>
    route: <routing text for the tier>
    scope: <globs, or none>
wave 2
  ...
== waves ==
checked: <n> stories, <n> edges, <n> waves
passed
```

Findings:

- `ERROR  cycle: <a> -> <b> -> <a>`;
- `ERROR  competing writers in wave <n>: <a> and <b> both write <glob>` when two stories of one
  wave declare scopes that overlap (equal globs, or one glob's fixed directory prefix contains
  the other's). Add an edge or split the story;
- `warn   no scope declared: <story>`;
- `ERROR  no design ids: <story>: a story proves ids; a requirement starts with machinery intake new`
  for an open story labelled `machinery`, or belonging to an intake (section 15.7), that has no
  `machinery:ids` line and is neither a capstone nor a design story (`machinery:intake-design`). Work in the design's name names the ids it proves;
  a new requirement goes through `machinery intake` instead;
- `warn   <n> open stories outside machinery (no design ids)`: the other open stories without a
  `machinery:ids` line, counted on one line when there is at least one, never listed one by one.

Two scopes overlap when they are equal or when one glob's fixed directory prefix (the path
before its first wildcard segment) contains the other's. With no open story the stage is
`not demonstrated: nothing to check`. bd refuses to store a blocking cycle, so the cycle finding
guards against a backlog edited by other means.

`--json` prints the same data as one JSON document: `{contract, design_files, waves: [{wave,
stories: [{id, tier, title, route, scope}]}], stage}`. The exit code follows section 3.3.

### 15.4 Contract freeze

Parallel work is safe only against one contract. The **contract hash** is the sha256 over the
sorted list of `<path> <sha256 of the file>` lines of the design sources: the model files, every
`*.machine.json` or imported machine source, the register of the imported machines, the
failure-mode file, the imported failure-mode catalog and its links file, the risk file, the status
file (section 4.1), the facts file and the contract files.
Generated files, the owed list and formal models are not part of it.

`machinery waves --freeze` requires a clean `gen-drift` stage and writes
`<git common dir>/machinery/wave.json`, which every worktree of the repository shares:

```json
{"version": 1, "contract": "<sha256>", "commit": "<full sha>", "files": {"<path>": "<sha256>"}, "stories": ["<wave 1 story ids>"]}
```

Stage `contract` (part of `machinery check`) is `not applicable: no wave is frozen` without that
file. With it, a design source whose hash differs from the frozen one is
`ERROR  contract changed during fan-out: <path>`. A contract change is its own story: land it
alone, then freeze again. For the same reason `machinery intake plan` refuses while a wave is
frozen (section 15.7): stories against a changed contract must not join a running wave. Freezing and thawing are the planner's acts and leave no trace but the wave file.
`machinery waves --thaw` removes the file when the wave has landed (`wave: thawed`, or
`wave: none frozen`). `--freeze` prints `wave: frozen <12 hex> at <short sha>, <n> stories` and
refuses when stage `gen-drift` or the waves stage itself does not pass. A design source that was
added or removed since the freeze counts as changed.

With `review.backlog` required (section 2), the waves stage of `--freeze` also holds the
checker's review (section 15.10). The latest backlog review for the current contract hash and the
current set of wave-1 story ids must be approved, else
`ERROR  no approved review of this backlog at <12 hex>: the checker runs machinery review backlog`.
It must come from another context than the one that freezes, compared as in section 15.10, else
`ERROR  the review of this backlog at <12 hex> is the planner's: <reason>; the checker reviews in a fresh context`;
the warnings of that comparison print as `warn` lines. `checked:` then also counts the reviews
read. Without `--freeze` the review is not looked at.
`checked:` counts design files.

### 15.5 `machinery brief`

```
machinery brief <story> [--role builder|verifier|modeler]
```

Prints what a worker needs as pointers, never as inlined content. The default role is `builder`.

```
BRIEF <story id> (<role>)
title: <story title>
tier: <tier> (<area>: <why>)
route: <routing text for the tier>
agent: <story id>-<role>                  # builder and verifier only
contract: <first 12 hex> at <short sha>
ids:
  <id>  <title>
read:
  <path>[#<anchor>]
decisions:
  <backlog id>  <title>
scope:
  <glob>
run:
  <command>
done when: <one line>
```

- `read` lists, for the ids of the story: the machine file and its generated table for `tc` ids,
  the model file for `act`, `inv` and `sc` ids, the failure-mode file for `fm` ids, the spec
  manifest and trace file for `fv` and `cx` ids, and `docs/formats.md` when the project has it.
- `decisions` lists, sorted by id, the current decisions (section 15.8) that govern an id of the
  story: an id or pattern of the decision's `machinery:ids` and an id of the story overlap. It is
  a pointer: the decision is read in the backlog.
- `run` for a builder: `machinery check --base <base> --story <id>`, `machinery run --affected --base
  <base> -- <commands.e2e>`, `machinery verify --story <id>`, `machinery proof --story <id> --post`.
- For a verifier, `read` never contains the proof, and `run` starts with
  `machinery predict <id> --observation "..." --expect "..."`, then
  `machinery proof --story <id>`, then `machinery accept <id> --verdict accept|reject --observed
  "..."`. The brief also lists every formal weakening accepted since the base (section 16.6).
- For a builder and a verifier, the line `agent: <story id>-<role>` follows `route`, and the first
  line of `run` is `export MACHINERY_AGENT=<story id>-<role>` (section 15.1): the worker exports it
  before it runs the other lines. A modeler brief has neither.
- For a modeler, `read` is the machine or model the formal model is derived from and the manifest;
  `run` is `machinery formal gen`, `machinery formal check <spec>`, `machinery formal baseline`.

An empty section prints `none`. Without `commands.e2e` the run lines say `<the end-to-end
command>`. A brief for a routine story fits in 40 lines.

### 15.6 Tier gates: `predict`, `accept`, `story`

`machinery story <story>` is stage `story`: it reads the story from the backlog and judges it at
the current commit.

| Tier | The gate requires |
|---|---|
| routine | Every id of the story is `proven` by evidence valid for the current commit (section 12) and is not owed. A `machinery:proof` line exists for a commit the evidence is valid for. |
| consequential | The routine gate. A `machinery:prediction` for that commit by an actor other than the proof's actor. A `machinery:verdict accept` for that commit by the predicting actor, recorded after the prediction. |
| critical | The consequential gate. Every `fv` id of the areas the story's ids belong to is `proven`, and every `cx` id of the formal models covering them is `proven`. |
| vital | The critical gate, the areas being the critical and vital ones. Every evidence id of every assumption of the vital areas the story's ids belong to is not owed: `ERROR  assumed, unproven: <area> <name>: <id> is owed; a story of a vital area needs the evidence of every assumption proven`. |

A vital gate never waits on a proof obligation and never counts one as proven: it prints
`warn   proofs awaiting: <n>` for the proof obligations of the story's vital areas and
`warn   vital: <id> proven (by test and model; proof awaited)` for each id of the story that a
property of those areas is over and the evidence proves. A story whose `machinery:ids` names a
`pf` id is `ERROR  proof: <id> awaits a machine-checked proof; a story does not discharge it: take it out of the story`.

An `act` id of the story only has to be not owed. A reject verdict is recorded and fails the gate
(`machinery accept --verdict reject` itself exits 0; `--close` with `reject` is a usage error).
The verifier must differ from every actor who posted a proof the evidence is valid for.

When the tier asks for a verifier, the sessions and agents of section 15.1 are compared as well.
Take a prediction for the commit, by an actor who posted no such proof, and a proof the evidence
is valid for. They are compared only when their sessions are equal; different sessions are no
finding. In the same session:

- both record an agent and the agents are equal: `ERROR  verifier shares a session and an agent
  with the builder: <session> <agent>`, once per session and agent;
- both record an agent and the agents differ: no finding (the harness gave subagents their
  parent's session, and the worker contexts are distinct);
- either records no agent: `warn   verifier and builder share session <session> and no agent is
  recorded for <proof|prediction>: independence rests on the actor name`, once per session and
  line kind, naming each line kind that lacks an agent.

When such a proof records no session the gate prints
`warn   no session recorded for proof: independence rests on the actor name`, and when such a
prediction records none, `warn   no session recorded for prediction: independence rests on the actor name`.
A missing session is never an error: harnesses that give none exist.

Findings name what is missing, for example `ERROR  no prediction for <short sha>: a verifier runs
machinery predict before reading the proof`, `ERROR  verifier is the builder: <actor>`,
`ERROR  unproven: <id>`. Adapter evidence (section 10.3) satisfies the gate only where the policy
`evidence.adapter` lets it count, which by default is never for an id of a critical or vital
area: such an id needs a receipt its test emits after its own assertions. An id of the story
that is not proven and has an adapter pass which does not count is
`ERROR  unproven: <id>; adapter evidence does not count: <reason>`, with the reason of section
10.3. A refused receipts file or an incomplete run fails the gate of every
story with the finding `verify` has for it (`ERROR  receipts: <file>: ...`,
`ERROR  incomplete run: <file>`; sections 10.2 and 12): the failure of the story may be in it.
`checked:` counts ids, proofs, predictions and verdicts. A story without a `machinery:ids` line
is `not demonstrated: story names no design ids`.

**The design story.** A story that carries `machinery:intake-design` lands the design change of its
intake (section 15.7) and names no ids. `machinery story` judges it at the current commit by the
design, not by evidence: every lint stage and stage `gen-drift` pass or are not applicable
(`ERROR  design: <stage>: <outcome line>`), the working tree is clean, and with `review.design`
required the latest design review of its intake for the current contract hash is approved
(`ERROR  no approved design review of <intake> at <12 hex>: the checker runs machinery review design <intake>`).
`checked:` counts the stages and the design reviews read. It asks for no verifier: the checker's
review is the independent reading of a design change.

An intake item (section 15.7) is not a unit of build work. The gate refuses it:
`ERROR  <id> is an intake item: an intake item is planned, not built; run machinery intake status <id>`.

**The capstone.** The capstone of an intake (section 15.7) has no `machinery:ids` line of its own:
its ids are the ids of the other stories of its intake, and `machinery story`,
`machinery proof --story` and `machinery brief` read those. Its tier is the highest of their tiers
and of its `tier:` label, and at least consequential. On top of the gate of its tier it requires:

- at least one `machinery:demo <command>` line:
  `ERROR  no demo: record each command that shows the intent on its own machinery:demo line`;
- every other story of the intake closed:
  `ERROR  open story of the intake: <id> <title>: the capstone closes last`;
- when it has no id at all (an intake that added none), evidence at the current commit:
  `ERROR  no evidence at <short sha>: run the tests through machinery run`.

`checked:` then also counts the demo lines and the other stories of the intake. For a capstone
without ids, `machinery proof --story` prints a block with no criteria.

`machinery predict <story> --observation "<what would disprove the claim>" --expect "<predicted
outcome>"` appends the prediction line. It refuses when a proof for the current commit was posted
by the same actor.

`machinery accept <story> --verdict accept|reject --observed "<what was seen>" [--close [--repair]]`
appends the verdict line. With `accept` it then runs the gate and exits with its code; `--close`
closes the story in the backlog when the gate passed. While the integration branch is red,
`--close` refuses without `--repair` (section 12.2).

A story that names a proposed or retired id (section 4.1) is not accepted on it:
`ERROR  proposed: <id> yields no obligation; commit it first` (`retired: <id> yields no
obligation; take it out of the story`).

The verifier is a fresh context that did not build the change. It receives the verifier brief,
predicts, and only then reads the proof. machinery cannot see what an agent read, and an actor
name is only as trustworthy as the backlog's own record of who wrote a note. What the gate
checks is what is recorded: that a prediction exists for this commit, under another name than
the proof's and, where both record a session and an agent, not from the same session and agent,
before the verdict.

### 15.7 Intake: `machinery intake`

Backlogs are not static: a requirement arrives while work runs. Left alone, agents write the
stories (or the code) first and patch the design later, or never. Intake makes the order a
property of the tool: the requirement is recorded; the design changes (model, machines, failure
modes and tiers, formal models where the tier asks for them, generation); only then are its
stories written, all at once, from what the design now owes.

```
machinery intake new "<title>" --intent "<who needs what, and the observable result>" [--refuses "<what must be refused>"]... [--parent <id>]
machinery intake status <id> [--json]
machinery intake plan <id> [--apply]
```

**The intake item.** `intake new` creates one backlog item, the record of one requirement, of
type `feature` with the label `machinery-intake` (a child of `--parent` when given), and prints
its id. Its description:

```
Intent:
<intent>

Refuses:
- <refusal>

machinery:intake
machinery:intake-base <full sha of HEAD> <contract hash> actor=<actor>[ session=<session>][ agent=<agent>]
```

There is one `- ` line per `--refuses`, in the order given, and `- none` without one; a newline
inside a value becomes a space. The base is the commit the requirement arrived at and the
contract hash (section 15.4) there, and the planner who recorded it: the actor of section 15.1,
with the session and the agent when they are set. An empty title or intent is a usage error. `intake new`
refuses (exit 1) without a commit, and while a design source has uncommitted changes:
`machinery: uncommitted design change: <path>, ...; a requirement starts from a committed design`.

An item is an intake item when its description or notes carry a `machinery:intake` line with
nothing after it. It is never a unit of build work: `waves` leaves it out (section 15.3) and
`machinery story` refuses it (section 15.6). The **stories of an intake** are the items that
carry `machinery:intake <id>` or are its children; its **capstone** is the story of the intake
that carries `machinery:capstone`. `intake status` and `intake plan` refuse an item that is not an
intake item: `usage: <id> is not an intake item: it has no machinery:intake line` (exit 2).

**Steps.** `machinery intake status <id>` prints where the requirement stands at the current
commit, one line per step, in this order:

```
intake <id>: <title>
base: <short sha>, contract <first 12 hex>
  design changed  done     <reason>
  design sound    done     <reason>
  generated       done     <reason>
  new ids         done     <reason>
    <id>  <tier>[ (<area>, <area>)]
    proposed, not planned: <id> <id>
  tiers           done     <reason>
    routine by default: <act id> <act id>
  formal          done     <reason>
  planned         missing  <reason>
intake <id>: 6 of 7 steps done; next: planned
```

| Step | Done when (reason) | Missing (reason) |
|---|---|---|
| `design changed` | The contract hash differs from the base hash and no design source has uncommitted changes (`contract <12 hex>, base <12 hex>`). Or the hash is the base hash, the item carries `machinery:no-design-change <reason>` and no id is new: the requirement is one the design already covers (`no design change: <reason>; the design already covers it`). | `uncommitted design change: <path>, ...; commit it`; `the contract is the base contract: change the design first, or record machinery:no-design-change <reason>`; `machinery:no-design-change needs a plan without new ids: <n> id(s) are new` |
| `design sound` | Every lint stage (section 3.1) passes or is not applicable (`every lint stage passes or is not applicable`). | `machinery lint: <stage>: <outcome line>; ...`, one part per stage that failed or was not demonstrated |
| `generated` | Stage `gen-drift` passes or is not applicable and no file of the gen directory has uncommitted changes (`machinery gen --check: no drift`). | `machinery gen --check: <outcome line>; run machinery gen and commit`; `uncommitted generated file: <path>, ...; commit it` |
| `new ids` | The base commit exists (`<n> new id(s) since <short sha>`, or `none since <short sha>: only the capstone is planned`). | `base commit <short sha> is not in this repository` |
| `tiers` | Stage `risk` passes or is not applicable (`stage risk passed`, or `no design.risk configured: every new id is routine`). | `stage risk: <outcome line>; run machinery lint --stage risk` |
| `formal` | Every critical or vital area that holds a new id has its `fv` obligation proven, or for each of its questions a spec of the matching tool that lists the area (section 16.1) (`no critical area holds a new id`, or `modeled: <area>, ...`). | `fv:<area>: no <tool> spec answers <question>; ...; scaffold it with machinery formal new` |
| `planned` | Every new id is in the `machinery:ids` of a story of the intake, the intake has its capstone and, when its contract hash is not the base hash, its design story (`every new id has a story; capstone <id>`, then `; design story <id>` when there is one). | `<n> new id(s) without a story; run machinery intake plan <id> --apply`, `no capstone; run machinery intake plan <id> --apply`, or `no design story; run machinery intake plan <id> --apply` |

`machinery:no-design-change <reason>` is written by the planner on the item's notes
(`bd update <id> --append-notes "machinery:no-design-change <reason>"`). The **new ids** are the
`act` and obligation ids now (section 4) that are not in the generated obligation list at the base
commit, read with `git show <base>:<gen>/obligations.json` (the gen directory of the base's own
`machinery.yaml`); a base without that file has no ids, so every id is new. A proposed id (section
4.1) yields no obligation and is never new: the proposed ids now that are neither in that list nor
proposed in the base's status file print on one line under the step,
`proposed, not planned: <id> <id>`, left out when there is none, and `intake plan` plans none of
them. The new ids print sorted, each with its tier and the areas that give it. Under `tiers`, the new `act` ids that are in no area
print as `routine by default`, so the planner sees what was left routine; the line is left out
when there is none. An item without a `machinery:intake-base` line (one not made by `intake new`)
has neither `design changed` nor `new ids` done, and while the new ids are not known `planned`
is missing too.

The last line is the summary: `intake <id>: <n> of 7 steps done`, followed by
`; next: <first missing step>` when a step is missing. Exit 0 when every step is done, 1
otherwise, 2 for a usage error. `--json` prints the same data as one document; `next` is empty when
every step is done:

```json
{"intake": "<id>", "title": "<title>", "base": {"commit": "<full sha>", "contract": "<sha256>"}, "contract": "<sha256>",
 "steps": [{"step": "design changed", "done": true, "reason": "<reason>"}],
 "new_ids": [{"id": "<id>", "tier": "<tier>", "areas": ["<area>"]}], "proposed": ["<id>"], "routine_by_default": ["<act id>"],
 "done": 6, "next": "planned"}
```

**Plan.** `machinery intake plan <id> [--apply]` refuses (exit 1) while a wave is frozen
(section 15.4): `machinery: a wave is frozen: land or thaw it; a contract change lands alone`. It
refuses unless the steps `design changed`, `design sound`, `generated`, `new ids`, `tiers` and
`formal` are done, with one line per missing step:
`machinery: cannot plan intake <id>: <step> is missing: <reason>`.

With `--apply`, when `review.design` is required (section 2) and the contract hash is not the base
hash, it also refuses (exit 1) unless the latest design review of the intake for the current
contract hash is approved (section 15.10):
`machinery: cannot plan intake <id>: no approved design review of <id> at <12 hex>: the checker runs machinery review design <id>`,
and unless that review comes from another context than the one applying the plan, compared as in
section 15.10:
`machinery: cannot plan intake <id>: the design review of <id> at <12 hex> is the planner's: <reason>; the checker reviews in a fresh context`.
The warnings of the comparison print on stdout as `warn: <text>`. Without `--apply` the review is
not looked at.

Then it does what `machinery plan` does (section 15.2) for the new ids that no story of the
intake lists yet, instead of for the owed list: the same groups, titles, tiers, priorities, edges
and story text. Every story is a child of the intake and carries `machinery:intake <id>` and the
intake's intent. One more story, the **capstone**, is added when the intake has none:

- title `Demonstrate: <intake title>`, type `task`, labels `machinery` and `tier:<tier>`, where
  the tier is the highest tier of the new ids and at least consequential, priority from that tier;
- blocked by every other story of the intake: the ones created now and the ones it already has;
- the story text with the intent line `Show the requirement on the running system through its
  real interface.`; the acceptance lines `- The intent is observed: <intent>` and
  `- Refused: <refusal>` for each refusal of the intake; `machinery:capstone` and
  `machinery:intake <id>` in `Ids`, without a `machinery:ids` line; the exclusion
  `- No new behavior: the capstone shows what the other stories built.` before the design one;
  and its blockers under `Needs`, each with the output `the behavior shown`;
- acceptance criteria: the intent is shown on the running system through its real interface,
  including each refusal the intake lists; the commands that reproduce it are recorded on the
  story, one `machinery:demo <command>` line each; every other story of the intake is closed;
  `machinery story` passes for it (section 15.6).

When the capstone exists already, the stories created now are added to its blockers.

One more story, the **design story**, is added when the contract hash is not the base hash and no
story of the intake carries `machinery:intake-design`. It is the one story that lands the design
change, alone and before the stories that build on it, so stage `scope` (section 15.11) lets it
change design sources:

- title `Land the design: <intake title>`, type `task`, labels `machinery` and `tier:<tier>`, where
  the tier is the highest tier of the new ids (`routine` without one), priority from that tier;
- it blocks every other story created now: each lists it under `Needs` with the output
  `the design it builds on`, and the capstone, blocked by every other story of its intake, with
  `the behavior shown`;
- the story text with the intent line `Land the design change of the requirement alone, before
  the stories that build on it.`; the acceptance lines `- The design sources are committed, sound
  and generated: machinery lint and machinery gen --check pass.` and `- The checker approved the
  design change: machinery review design <id>.`; in `Ids`, `machinery:intake-design`, then
  `machinery:scope` with the design sources of the contract hash (section 15.4) and `<gen>/**`,
  sorted, then `machinery:intake <id>`, without a `machinery:ids` line; the exclusion
  `- No code and no test: the stories of the intake build on this design.`; and `- none` under
  `Needs`;
- acceptance criteria: the same two lines, and `machinery story` passes for it (section 15.6).

The design was committed before the plan (step `design changed`): the story records the change
and is landed by its branch, `machinery check --base <ref> --story <id>` holding it to its scope.
A planner who wants another story to land the design appends `machinery:intake-design` to it
before planning; then none is created.

Without `--apply` it prints the stories as the Markdown of section 15.2, the design story first and
the capstone last. With
`--apply`:

1. it creates every story and edge in one `bd create --graph` call; an edge to a story that
   exists already uses its id. bd creates the graph whole or not at all;
2. only after that call succeeded it writes `owed.yaml`: each id of a story created now is added
   when it is not owed yet, with the reason `owed by intake <id>`, and the `story` field of each of
   those ids names its story. When the call fails nothing is written, so the owed list and the
   backlog are as they were;
3. it appends `machinery:intake-planned <full sha> <contract hash> actor=<actor>[ session=<session>][ agent=<agent>]`
   to the intake's notes, unless a line with the current contract hash is there already.

It prints `plan: <n> created, <n> existing, <n> edge(s)` (existing: the stories the intake had
before), `owed: <n> added`, and `intake <id>: planned at <short sha>, contract <12 hex>` when step 3
wrote a line. Both forms first print `intake <id>: no new id: the design change adds none; only
the capstone is planned` for an intake without new ids (a rewording, a removal, or a
requirement the design already covers), with `only the design story and the capstone are
planned` in place of the last part when the contract hash is not the base hash, and `intake <id>: every new id has a story; nothing to
create` when there is nothing left to create. Running it again therefore creates nothing twice.

When it does not refuse, both forms print first a reminder when closed stories carry learnings
the planner has not read (section 15.8): `learnings: <n> unread since <short sha>; machinery learnings --since
<short sha>`, where the commit is the last planned one, the newest commit of a
`machinery:intake-planned` line of any intake item, and `<n>` counts the learning lines of the
stories closed since it. With no planned line yet the reminder is
`learnings: <n> unread; machinery learnings`. It is a reminder, never a refusal, and it is not
printed when `<n>` is 0.

**Stage `intake`.** Part of `machinery check`. It reads the backlog with one `bd list`, only when
`backlog.tool` is set and `bd` is runnable. It is `not applicable: no backlog configured`,
`not applicable: bd not runnable`, `not applicable: the backlog cannot be read` or
`not applicable: no open intake item` otherwise. For each open intake item:

- `warn   intake <id>: the design changed and no stories were planned from it; run machinery intake plan <id>`
  when the contract hash differs from its base and no `machinery:intake-planned` line carries the
  current contract hash;
- `warn   intake <id> is delivered and still open; close it` when its capstone and every other
  story of the intake are closed.

Warnings only: the stage passes. `checked:` counts the open intake items.

### 15.8 Decisions, observations and learnings

The backlog keeps three more records of the work: decisions taken, observations made outside a
story's scope, and learnings for the next story. Each is a tag line (section 15.1), so the backlog
stays the one record.

**Decisions.**

```
machinery decide "<title>" --why "<reason and alternatives>" [--ids <id>...] [--supersedes <decision id>] [--vetoable] [--actor <name>]
machinery decisions [--ids <id>...] [--open-vetoes]
```

`decide` creates one backlog item of type `decision` and prints `decision: <id>`. Its description:

```
Why:
<why>

machinery:decision <actor>
machinery:ids <id> <id>
machinery:supersedes <decision id>
machinery:vetoable
```

The `machinery:ids` line lists the ids and id patterns the decision governs, sorted, and is left
out without `--ids`; `--ids` is repeatable and takes comma-separated values, and each must name
an id of the design or a pattern that matches one (section 4.1 statuses included). The
`machinery:supersedes` and `machinery:vetoable` lines are written only with their flags. A
vetoable decision is one an agent took within its remit that the owner may overturn. A newline in
the title or the reason becomes a space. Refusals (exit 2): an empty title or reason
(`usage: machinery decide needs --why`), an id that names nothing
(`usage: --ids <id> names no design id`), no actor, and a `--supersedes` item that is not a
current decision (`usage: <id> is not a decision: it has no machinery:decision line`,
`usage: <id> is superseded already by <id>`).

With `--supersedes <id>` the new item is created first, then `bd supersede <id> --with <new id>`
closes the older one with a link to it, and `superseded: <id> by <new id>` is printed.

An item is a decision when its description or notes carry a `machinery:decision` line. It is
**current** unless another decision carries `machinery:supersedes <its id>`, whatever its status,
or it carries `machinery:overruled <actor>` (an answer on the owner's behalf that the owner
answered otherwise, section 15.9).
The owner confirms a vetoable decision by appending `machinery:confirmed <actor>` to its notes
(`bd update <id> --append-notes "machinery:confirmed <actor>"`) and overturns it with a decision
that supersedes it. A decision is never a unit of build work: `waves` leaves it out.

`machinery decisions` lists the current decisions, sorted by id, three lines each, then a count:

```
<id>  <title>
  governs: <id> <id>
  by: <actor>[, vetoable][, confirmed by <actor>]
decisions: <n> current
```

`governs: none` when the decision has no `machinery:ids` line. `--ids` (repeatable,
comma-separated, patterns allowed) keeps the decisions that govern one of them: a decision governs
an id or pattern when one of its own ids or patterns overlaps it (equal ids, or an id with the
prefix of a pattern, or two patterns of which one prefix contains the other). A decision without
ids governs none. `--open-vetoes` keeps the vetoable decisions that are not confirmed, and the last
line is then `decisions: <n> open to veto`. `machinery brief` lists the same decisions for the ids
of a story (section 15.5). Exit 0.

**Importing decisions.** A project that already keeps its decisions in a file moves them into the
backlog record by record:

```
machinery decisions import --file <path> --record "<regex of a record's heading line>" [--apply] [--actor <name>]
```

`--file` is a path relative to the project root, inside it. A **record** starts at a line the
`--record` expression matches and runs to the line before the next such line, or to the end of the
file; lines before the first record belong to none. Per record, in file order:

- the **title** is the heading line with its leading `#` characters and white space removed (the
  heading as written when that leaves nothing);
- the **body** is the record's other lines, blank lines at either end dropped (the title when no
  line is left); a `machinery:` in it is written `machinery :` so that no line of the body starts
  a tag;
- the **ids** are every design id the text (heading and body) names that exists in the design
  (section 4.1 statuses included): each token of `[A-Za-z0-9_.:-]` characters, trailing `.` and
  `:` dropped, is resolved as a token of a project's own trace marker is (section 8.1): with a
  kind prefix as written, a bare one as `inv:`, `act:`, `fm:` (lower-cased) and `sc:`, the first
  that names an id; sorted, once each;
- the **source** is `<path>#<12 hex>`, the 12 hex the start of the sha256 of the heading line
  (trailing white space dropped).

Without `--apply` it prints, per record:

```
<line>: <title>
  source: <path>#<12 hex>
  governs: <id> <id>
  backlog: new
```

`governs: none` without ids, and `backlog: <item id>` when an item of the backlog, open or closed,
carries `machinery:decision-source <the same source>`: the record was imported already. The
backlog is read with one `bd list` when `backlog.tool` is set and `bd` can run; otherwise every
record is new and the summary says so (`backlog: not read: <reason>; every record counts as
new`). Then:

```
decisions import: <n> record(s) in <path>, <k> naming design ids, <c> new, <e> imported already
shed: <n> line(s): an instructions file that loads <path> is that much shorter once the file is removed from what agents load
```

where `shed` counts the lines of the file as section 18.7 does. With `--apply` it creates every
new record, in one `bd create --graph` call, as one backlog item of type `decision` whose title
is the record's title and whose description is

```
Why:
<body>

machinery:decision <actor>
machinery:ids <id> <id>
machinery:decision-source <path>#<12 hex>
```

(the `machinery:ids` line only with ids), prints `backlog: <new id>` for each and `<c> created` in
the summary in place of `<c> new`. Running it again creates nothing twice. It reads the file and
never writes, moves or deletes it. Refusals, exit 2, before anything is written:
`usage: machinery decisions import needs --file <path> and --record "<regex>"`,
`usage: --record is not a regular expression: <error>`, `usage: --file <path>: cannot read`,
`usage: --record "<regex>" matches no line of <path>`, a path outside the project, no actor (with
`--apply`) and, with `--apply`, a project without a backlog (`config:`, section 15.1).

**Observations.** A builder who sees something outside the story's scope records it with
`machinery proof --observation "<kind>: <text>"` (section 13); posted, it is a line
`machinery:observation <kind> <text>` on the story.

```
machinery observations [--file]
```

lists, for every item of the backlog, open or closed, sorted by item id and then in the order of
the lines, each observation line that is not filed: `<item id>  <kind>: <text>`, then
`observations: <n> not filed`. An observation is filed when the same item carries
`machinery:observation-filed <id> <kind> <text>` with the same kind and text; the same kind and
text twice on one item are one observation.

With `--file` each of them becomes one backlog item: type `bug` for `bug` and `security`, `task`
for `debt`, `perf` and `doc`; the label `observation:<kind>`; the text as the title; the
description `Observed in <item id> (<item title>): <kind>: <text>`; and a `discovered-from`
link to the item it was seen in. Then the line
`machinery:observation-filed <new id> <kind> <text>` is appended to that item, so a second run
files nothing. It prints `filed: <new id>  <kind>: <text> (from <item id>)` per observation and
`observations: <n> filed`. Exit 0; a backlog call that fails stops the run, and what was filed
before it stays filed and marked.

**Learnings.** A builder records a rule the next story or the planner should know with
`machinery proof --learning "<text>"`; posted, it is a line `machinery:learning <text>`.

```
machinery learnings [--since <git ref>]
```

prints the learning lines of the closed stories, grouped by intake (the intake of a story, section
15.7), the intakes sorted by id and the stories outside an intake last:

```
intake <intake id>: <intake title>
  <story id>  <learning>
outside an intake
  <story id>  <learning>
learnings: <n> from <n> closed stories since <short sha>
```

Within a group the lines are sorted by story id and keep their order on the story. With
`--since`, a closed story counts when one of its `machinery:proof` commits is neither the ref nor
an ancestor of it (a commit this repository does not have counts); a closed story with no proof
line counts only without `--since`, and the last line then has no `since` part. An unknown ref is
a usage error (`usage: --since <ref>: not a commit of the project`). Exit 0. The planner reads
them before the next plan; `machinery intake plan` reminds it (section 15.7).

### 15.9 Questions for the owner

**Experimental.** Questions and answers on the owner's behalf are an experiment: the commands,
lines and policy format of this section may change in any release.

Agents stall when a decision is the owner's and the owner is away. A question puts the decision
in the backlog with what the owner needs to answer it at a glance, and with what happens if
nobody does. A policy the owner writes lets an agent answer some questions on the owner's
behalf, within limits, as a decision the owner may overturn.

```
machinery ask "<question>" --option "<a>" --option "<b>" [--option "<c>"]... --recommend <n> --default "<what happens if nobody answers>"
              [--about <topic>]... [--amount <number> --unit <unit>] [--story <id>] [--actor <name>]
machinery answer <id> --option <n> --reason "<text>" [--actor <name>]
machinery answer <id> --delegate --clause <clause id> (--option <n> | --hold | --refuse) --reason "<text>" [--actor <name>]
machinery questions [--held]
```

**Asking.** `ask` creates one backlog item of type `task` whose title is the question, and prints
`question: <id>`. Its description:

```
machinery:question <actor>
machinery:option 1 <a>
machinery:option 2 <b>
machinery:recommend <n>
machinery:default <what happens if nobody answers>
machinery:about <topic> <topic>
machinery:amount <number> <unit>
machinery:story <id>
```

The options are numbered from 1 in the order given. `machinery:about` lists the topics sorted, once
each; it, `machinery:amount` and `machinery:story` are written only with their flags. A newline in a
value becomes a space. A topic is a free word of `^[a-z][a-z0-9-]*$` that says what the question is
about (`refund`, `naming`, `dependency`). Five topics are reserved, because a wrong answer to them
is hard to undo or changes the rules themselves: `publish`, `payment`, `destructive`, `method` and
`delegate-policy`. An amount is what the answer may spend or move: a number of
`^[0-9]+(\.[0-9]+)?$` in a unit that is one word (`EUR`, `hours`). `--story` names the story the
question holds up.

`ask` refuses an incomplete question, exit 2, before anything is written:
`usage: machinery ask takes one question`, `usage: a question needs at least two --option`,
`usage: --option: an option must not be empty`,
`usage: a question needs --recommend <n>: the number of the option you recommend`,
`usage: --recommend <n> names no option: give 1 to <k>`,
`usage: a question needs --default: what happens if nobody answers`,
`usage: --about "<topic>": a topic is one word of ^[a-z][a-z0-9-]*$`,
`usage: --amount and --unit go together`, `usage: --amount "<value>" is not a number`,
`usage: --unit "<value>" is not one word`, no actor, and a `--story` the backlog does not hold.

An item is a question when its description carries a `machinery:question` line. A question is
never a unit of build work: `waves` leaves it out.

**The owner's answer.** `machinery answer <id> --option <n> --reason "<text>"` appends
`machinery:answer <n> actor=<actor> reason="<text>"` to the item's notes, prints
`answered: <id> option <n>: <option>`, closes the item when it is open and prints `closed: <id>`.
When the question carries an answer given on the owner's behalf that decided (an option or a
refusal), the owner's answer settles it, in the same note: the same option appends
`machinery:confirmed <actor>` and prints `confirmed: <id>`; another answer appends
`machinery:overruled <actor>` and prints `overruled: <id>`. Usage errors, exit 2:
`usage: machinery answer takes one question id`, `usage: machinery answer needs --option <n>, or --delegate`,
`usage: <id> is not a question: it has no machinery:question line`,
`usage: --option <n> names no option of <id>: give 1 to <k>`, `usage: machinery answer needs --reason`,
`usage: --hold, --refuse and --clause go with --delegate`, `usage: <id> is answered already by <actor>`
(an answer of the owner exists) and no actor. When a delegate policy is configured, the actor must
be the owner it names, else exit 1:
`machinery: refused: <actor> is not the owner the delegate policy names (<owner>); an agent answers with --delegate`.

**An answer on the owner's behalf.** `delegate.policy` in `machinery.yaml` (section 2) names the
owner's policy file:

```yaml
version: 1
owner: <name>                    # the actor whose answers are the owner's
clauses:
  - id: small-refunds            # ^[a-z][a-z0-9-]+$, unique
    about: [refund]              # the topics it covers; "*" covers every topic that is not reserved
    effect: allow                # allow | refuse | hold
    limit: {amount: 200, unit: EUR}   # optional: the most the option answers under it move together
    confirm: true                # optional, default false: the item stays open for the owner
    note: "refunds below the limit while I am away"   # required: why the owner grants it
```

The path is absolute or relative to the project root and may lie outside the project. It is the
owner's file: machinery reads it and never writes it, and keeping it outside the project keeps it
out of the agents' reach. A problem with it is a config error, exit 2,
`config: delegate.policy <path>: <message>`: the file cannot be read or is not YAML of this shape
(unknown keys included), `version must be 1`, `owner is required`,
`clause <n>: id must match ^[a-z][a-z0-9-]+$`, `clause <id>: duplicate id`,
`clause <id>: about needs at least one topic`,
`clause <id>: effect must be allow, refuse or hold, got "<effect>"`,
`clause <id>: limit needs an amount (a number) and a unit`, `clause <id>: note is required`.

An answer on the owner's behalf comes from the delegate, never from the context that asked.
`ask` writes the session and the agent on the question line (section 15.1). With `--delegate`,
before the conditions below, `MACHINERY_AGENT` must be set and end in `-delegate`, and the
question must not have been asked by this session and agent; each refuses, exit 1, nothing
written: `machinery: refused: MACHINERY_AGENT is not set: an answer on the owner's behalf comes from the delegate, whose agent ends in -delegate`,
`machinery: refused: MACHINERY_AGENT <agent> does not end in -delegate: an answer on the owner's behalf comes from the delegate`,
`machinery: refused: <id> was asked in session <session> by agent <agent>: the asker does not answer its own question`
(compared only when the question records both). The answer line records the session and the agent.

With `--delegate`, `--clause` and exactly one of `--option`, `--hold` and `--refuse` are required
(`usage: --delegate needs --clause`, `usage: --delegate needs one of --option, --hold and --refuse`),
and a question that carries an answer of the owner, or an answer on the owner's behalf that
decided, is `usage: <id> is answered already by <actor>`. The answer is accepted only when every
condition below holds, checked in this order; the first that fails refuses it, exit 1, with
`machinery: refused: <reason>` on stderr, and nothing is written. A hold decides nothing: it is
accepted once the clause exists, so every later refusal ends with `; answer --hold`.

| Condition | Reason when it fails |
|---|---|
| `delegate.policy` is set | `no delegate policy: delegate.policy is not set in machinery.yaml; only the owner answers <id>` |
| The clause is in the policy | `clause <clause> is not in the delegate policy` |
| The question names a topic | `<id> names no topic: a clause covers only the topics it names; answer --hold` |
| Each reserved topic of the question is named in the clause's `about` | `<id> is about <topic>, a reserved topic, and clause <clause> does not name it; answer --hold` |
| The clause covers every topic of the question: `about` names it, or holds `*` and the topic is not reserved | `clause <clause> does not cover <topic>; answer --hold` |
| The clause's effect permits the answer: `allow` an option, `refuse` `--refuse`, `hold` only `--hold` | `clause <clause> has effect <effect>: it does not permit <--option or --refuse>; answer --hold` |
| An option to a question with an amount: the clause has a limit | `clause <clause> sets no limit: an answer that moves <amount> <unit> needs one; answer --hold` |
| ... in the question's unit | `clause <clause> limits <unit>, the question is in <unit>; answer --hold` |
| ... that the amount and the earlier option answers under the clause do not exceed | `<amount> <unit> with <spent> <unit> answered under clause <clause> exceeds its limit of <limit> <unit>; answer --hold` |

The earlier option answers under a clause are the `machinery:answer <n> ... clause=<clause>` lines
of every item of the backlog, open or closed, read when the answer is given, each counted with
the amount of its question when that amount is in the limit's unit. An answer overruled or
superseded later still counts: the ledger records what agents committed to. Amounts are added
exactly, as decimals, and printed without trailing zeros.

An accepted option or refusal appends, in one note, `machinery:answer <n|refuse> actor=<actor>
clause=<clause> reason="<text>"`, `machinery:decision <actor>` and `machinery:vetoable`: the
question is now a vetoable decision (section 15.8), listed by `machinery decisions --open-vetoes`
until the owner confirms or overrules it with `machinery answer`, or supersedes it with
`machinery decide --supersedes`. It prints `answered: <id> option <n>: <option> (on the owner's
behalf, clause <clause>)`, or `answered: <id> refuse (on the owner's behalf, clause <clause>)`, then
`closed: <id>`; under a clause with `confirm: true` the item stays open and the line is
`open: <id> waits for the owner's confirmation (clause <clause>)`. An accepted hold appends
`machinery:answer hold actor=<actor> clause=<clause> reason="<text>"`, prints
`held: <id> (clause <clause>): <reason>` and leaves the item open.

**Listing.** `machinery questions` lists the open questions, sorted by id, three lines each, then a
count:

```
<id>  <question>[ (story <story id>)]
  recommended: <n> <option>; default: <default>
  waits for: <what>
questions: <n> open, <k> held
```

`<what>` follows the last answer: `the owner` when there is none, `the owner; held under clause
<clause>: <reason>` after a hold, `the owner's confirmation of option <n> under clause <clause>` or
`the owner's confirmation of the refusal under clause <clause>` after an answer on the owner's
behalf, `nothing: answered by <actor>` after an answer of the owner. A question is **held** when
its last answer is a hold. `--held` keeps the held questions, and the last line is then
`questions: <k> held`. Exit 0. `machinery status` prints the count line when a question is open
(section 14).

### 15.10 Reviews: `machinery review`

The checker (`agents/machinery-checker.md`) is called at set points and answers pass or fail. A
review that lives only in a reply cannot hold anything, so the checker records it, and the
gates of dispatch wait for it:

```
machinery review backlog --verdict approved|rejected [--issue "<rule>: <scope>: <instances>"]... [--actor <name>]
machinery review design <intake id> --verdict approved|rejected [--issue "..."]... [--actor <name>]
machinery review intake <intake id> --verdict approved|rejected [--issue "..."]... [--actor <name>]
```

| Kind | When | Recorded on | Waited for by |
|---|---|---|---|
| `backlog` | Before a wave is dispatched. | The review item. | `machinery waves --freeze` (section 15.4), with `review.backlog` required. |
| `design` | After the design change of an intake, before its stories. | The intake item. | `machinery intake plan <id> --apply` (section 15.7), with `review.design` required. |
| `intake` | After `intake plan <id> --apply`: the stories of one intake. | The intake item. | Nothing: `machinery status` shows it. |

A design review names its intake, since it is the change of that one requirement that it reads
and the plan of that one intake that waits for it. The command appends one line to the notes:

```
machinery:review <kind> <approved|rejected> commit=<full sha of HEAD> contract=<contract hash> actor=<actor>[ session=<session>][ agent=<agent>][ stories=<id>,<id>][ issue1="<text>" ... issue5="<text>"]
```

- `stories` is written for a backlog review only: the ids of wave 1 of `machinery waves` at the
  review, sorted, comma-separated, left out when wave 1 is empty.
- The **review item** holds the backlog reviews: the first item, by id, with the label
  `machinery-review`. When there is none, `review backlog` creates it (type `task`, the label
  `machinery-review`, title `Reviews of the backlog`, description `Reviews of the backlog by the
  checker, one machinery:review line each.`) and closes it at once: it is a record, never work.
- The **latest review** of a kind is the last such line, in the order of the lines of an item and
  the items in id order. A gate reads the latest one for the current contract hash (and for a
  backlog review the current wave-1 ids): a rejection after an approval withdraws it.
- It prints `review: <kind>[ <intake id>] <verdict> at <12 hex> on <item id>`, and the warnings
  of the comparison below as `warn: <text>`.

Refusals, exit 2, before anything is written: `usage: machinery review takes backlog, design <intake id> or intake <intake id>`,
`usage: machinery review needs --verdict approved|rejected`,
`usage: an approved review takes no --issue`,
`usage: a rejected review needs one to five --issue "<rule>: <scope>: <instances>"`,
`usage: --issue "<text>": an issue names its rule and its scope: "<rule>: <scope>: <instances>"`
(an issue is split at `: `; its rule and its scope must not be empty, its instances may be; a
newline becomes a space), `usage: <id> is not an intake item: it has no machinery:intake line`,
no actor, and a session or agent that is not one word (section 15.1).

**The reviewer is not the planner.** The planning lines of a review are the
`machinery:intake-base` and `machinery:intake-planned` lines that record an actor: those of the
intake item for a design or intake review, and those of every intake item with a story in wave 1
for a backlog review. The reviewer (its actor, session and agent) is compared with each, as the
verifier is with the builder (section 15.6):

| Reviewer and planner | Outcome |
|---|---|
| Both record a session, and the sessions differ | Two contexts: no finding. |
| The same session, both record an agent, the agents equal | One context: `same session <session> and agent <agent>`. |
| The same session, both record an agent, the agents differ | Two contexts: no finding. |
| The same session, an agent missing, the same actor | One context: `same actor <actor> in session <session>, and no agent tells them apart`. |
| The same session, an agent missing, other actors | `warn: reviewer and planner share session <session> and no agent is recorded for <which>: independence rests on the actor name` |
| A session missing, the same actor | One context: `same actor <actor>, and no session tells them apart`. |
| A session missing, other actors | `warn: no session recorded for <which>: independence rests on the actor name` |

`<which>` is `the review`, `the plan` or `the review and the plan`, whichever lacks it. One context
refuses, exit 1, nothing written:
`machinery: refused: the reviewer is the planner: <reason>; the checker reviews in a fresh context`.
`waves --freeze` and `intake plan --apply` compare the review they wait for in the same way with
the context that runs them, the planner's.

### 15.11 Stage `scope`

A story's scope (`machinery:scope`, section 15.1) says which paths it writes. `waves` compares
scopes to find competing writers; stage `scope` holds a story to its own:

```
machinery check --base <ref> --story <id>
```

The changed files are those of `git diff --name-only --no-renames --cached <merge base>`, the
merge base being `git merge-base <ref> HEAD`: what is committed since the branch left the base,
and what is staged to be committed. The owed list and the tests and architecture baselines are
not judged: a builder prunes them (`machinery owed sync --prune-only`), and the ratchet holds
them. For every other changed file, in path order:

- a design source (a file of the contract hash, section 15.4), unless the story carries
  `machinery:intake-design` (section 15.7):
  `ERROR  scope: <file> is a design source; a design change is an intake`;
- a file no glob of the story's `machinery:scope` lines matches (doublestar, section 2):
  `ERROR  scope: <file> is outside the scope of <story>`.

`checked:` counts the files and the scope globs. A story without a `machinery:scope` line is
`not demonstrated: <story> declares no scope; the planner adds a machinery:scope line`, and no
changed file is `not demonstrated: nothing changed since <ref>`. Without `--story` the stage does
not run; asked for with `--stage scope` it is `not applicable: no story named; run machinery check --base <ref> --story <id>`.
`--story` without `--base` is `usage: --story needs --base <ref>`, a story the backlog does not
hold is `usage: no story <id> in the backlog: <reason>`, and a project without a backlog is a
config error (section 15.1). The pre-commit hook runs the stage when the branch is named after a
story (section 21.1).

## 16. Formal models

A critical or vital area (section 23) gets a formal model. A model is worth nothing when it is vacuous,
out of date, quietly weakened, or built on unstated assumptions, so every model carries the
mechanical checks below. `machinery formal check` runs the tools and records results; stage
`formal` is pure and reads only committed files, so `machinery check` needs no Java.

### 16.1 Which tool

The area's `questions` pick the tool:

| Question | Meaning | Tool |
|---|---|---|
| `ordering` | Which sequences of events are legal, and what each leaves behind. | TLA+ |
| `concurrency` | What happens when callers act at the same time. | TLA+ |
| `recovery` | What happens when work is lost or repeated: crash, timeout, retry. | TLA+ |
| `structure` | Which shapes of data may exist: ownership, authorization reachability, referential rules. | Alloy |

A critical or vital area needs, for each of its questions, at least one spec of the matching tool that
lists the area. Otherwise: `ERROR  fv:<area>: no <tool> spec answers <question>`.

### 16.2 Files

Everything lives in the formal directory (`design.formal`):

| File | Written by | Holds |
|---|---|---|
| `<spec>.spec.yaml` | A person or agent (`machinery formal new` scaffolds it) | The manifest. |
| `<Machine>.tla` | `machinery formal gen` | TLA+ skeleton generated from a machine. Never edited. |
| `<model>.als` | `machinery formal gen` | Alloy skeleton derived from a Modelith model. Never edited. |
| `<properties>.tla` / `<properties>.als` | A person or agent | The properties. Extends or opens the skeleton. |
| `<spec>.result.json` | `machinery formal check` | Results bound to the spec bytes (section 16.9). |
| `baseline.json` | `machinery formal baseline` | Section 16.6. |
| `traces/<spec>.<run>.trace.json` | `machinery formal check` | Counterexamples (section 16.10). |

The manifest:

```yaml
version: 1
spec: order-payment              # ^[a-z][a-z0-9-]+$, equals the file name stem
tool: tla                        # tla | alloy
areas: [order-payment]           # the risk areas this spec is a model of
questions: [concurrency]         # which of the areas' questions it answers
machine: Order                   # tla: base name of the machine file the skeleton comes from
model: design/model/shop.modelith.yaml   # alloy: the model the skeleton is derived from
properties: OrderProps           # module or file stem of the hand-written properties
sources:                         # section 16.7
  read_at: <full sha>
  paths: ["design/machines/Order.machine.json", "internal/**"]
  log: []
assumptions:                     # section 16.3
  - {id: A1, text: "A transition is applied as one compare-and-set on the stored state.", status: proven, evidence: "cx:order-payment:race"}
  - {id: A2, text: "The payment provider answers, fails or times out.", status: unproven}
runs:                            # section 16.4
  - name: strict
    constants: {Instances: 2, Callers: 2, Atomic: true}
    invariants: [TypeOK, ChargedOnce]
    properties: [Settles]
    reach: [SomePaid, SomeCancelled]
    expect: pass
  - name: race
    constants: {Instances: 1, Callers: 2, Atomic: false}
    invariants: [ChargedOnce]
    expect: finding
    assumption: A1
mutations:                       # section 16.5
  - id: M1
    bug: "A paid order can be charged again."
    file: Order.tla
    from: "st[i] = \"Pending\""
    to: "st[i] \\in {\"Pending\", \"Paid\"}"
    detects: ChargedOnce
    run: strict
    origin: ""
```

Unknown keys are an ERROR in stage `formal`. `machine` is required for `tla`, `model` for `alloy`.

**The TLA+ skeleton.** Generated from the machine's runtime table (section 7.3), so it changes
exactly when the table does. The module is named after the machine file. It declares:

- `CONSTANTS Instances, Callers, Atomic`. `Instances` and `Callers` are sets of strings; `Atomic`
  is a boolean.
- `VARIABLES st, fx, pend, last` and `vars`. `st[i]` is the leaf state path of instance `i`.
  `fx[i][e]` counts how often effect `e` ran for `i`, saturating at 2 (0, 1, more than once).
  `pend` is the set of steps in flight, records `[c, i, id]`. `last` is the step just taken, a
  record `[id, i, c, step]` of strings; traces are read from it.
- `States`, `Effects`, `Final` (leaf paths of top-level final states), `Resting`, `InitialState`
  (the initial leaf), `TypeOK`, `Init`. When the machine declares `_entries`, `EntryStates` holds
  the initial leaf of each entry state and `Init` starts each instance in `InitialState` or in a
  state of `EntryStates`; otherwise every instance starts in `InitialState`.
- For each transition row, in table order, with the suffix `<TAG>_<hex>` of its id:
  `Apply_<s>(i)` sets the target (the leaf reached through `initial` for a compound target; the
  state is unchanged for an internal transition) and bumps the row's effects. A row of kind `on`
  is caller-driven and yields `Do_<s>(i, c)` (enabled when `Atomic`: checks the source and
  applies in one step), `Begin_<s>(i, c)` (enabled when not `Atomic`: checks the source and puts
  the step in `pend`; a caller has at most one step in flight) and `Commit_<s>(i, c)` (applies a
  pending step without looking at the state again). Every other row is engine-driven and yields
  `Do_<s>(i)` only, with `c` recorded as `"engine"`.
- `CallerStepOf(i)`, `CommitStepOf(i)`, `EngineStepOf(i)` and `InternalOf(i)` (the engine-driven
  and commit steps of one instance); `Terminated` (every instance final and nothing pending:
  stutter, so termination is not a deadlock); `Next`;
  `Fairness == \A i \in Instances : WF_vars(InternalOf(i))`, so no instance is starved by
  another; and `Spec == Init /\ [][Next]_vars /\ Fairness`.

Guards are not modeled: any branch may fire, the environment decides. Ignore, refusal and clause
cases change nothing and are not actions. The generated header states these as assumptions `G1`
(engine-driven steps are atomic), `G2` (any guard may hold) and `G3` (a source on a compound state
applies in each of its leaves).

The property module `EXTENDS` the skeleton and defines invariants, temporal properties and reach
predicates. It may define its own specification (more actions, other fairness) and a run then
names it with `spec: <name>`.

**The Alloy skeleton.** Derived from the Modelith model. The module is named after the model file.

| Model | Alloy |
|---|---|
| Enum `E` with values | `abstract sig E {}` and `one sig V extends E {}` per value. A value name used twice in the model is prefixed with its enum name. |
| Entity `A` | `sig A { ... }`; with `subtypeOf: B`, `sig A extends B`. |
| Attribute whose type is an enum or an entity | A field with multiplicity `one`. Attributes of primitive types are not structural and are left out. |
| Relationship of `A` to `B` with cardinality `l:r` | A field of `A` named by the lower-camel `role` when it is a single word, else the lower-camel name of `B` (a number is appended on a clash). Its multiplicity comes from `r`: `1` is `one`, `0..1` is `lone`, `1..n` is `some`, `n` is `set`; a count or other range is `set` plus a fact bounding `#`. A fact states `l` for the other direction: for `1`, `all b: B | one f.b`; `0..1` gives `lone`, `1..n` gives `some`, `n` nothing. |
| `ownership: owned` | An extra fact `all b: B | some f.b`: an owned thing has an owner. |
| Two relationships that mirror each other (`A` to `B` and `B` to `A`, one pair only) | A fact that the two fields are inverses. |

Scoped references to imported models and derived entities are skipped and listed in the header.
A name of the model that is an Alloy keyword (`set`, `once`, `releases` and the other reserved
words of Alloy 6) appears with a trailing underscore (`releases_`), in the skeleton and wherever
machinery writes Alloy. A fact names a field by itself, or as `(<Entity> <: <field>)` when another
entity has a field of the same name and target, where the bare name would be ambiguous.
The property file starts with `open <model>` and holds facts, predicates and assertions, and no
commands: machinery writes the commands from the runs.

`machinery formal gen` writes the skeleton of every manifest and never touches a property file.
`--check` prints a block named `formal-gen` with `DRIFT  <path>: stale` or `missing`.
`machinery formal new <spec> --tool tla --machine <Name> --area <area>` (or `--tool alloy --model
<path>`) writes a manifest with `sources.read_at` set to the current commit and one `strict` run,
and a property file: for TLA+ a reach predicate `SomeFinal` beside the skeleton's `TypeOK`; for
Alloy a reach predicate `NonEmpty` and a placeholder assertion `ReplaceMe` that the mutation rule
(section 16.5) then asks to be replaced. It needs a git repository with a commit and never
overwrites.

### 16.3 Assumptions

A model is a statement about a system under assumptions about its host: what is atomic, what the
platform guarantees, what a dependency does. They are numbered data, not comments:

- `id` matches `^A[0-9]+$` and is unique in the spec; `text` is required.
- `status: proven` requires `evidence`, the id of an obligation (often the `cx` id of the run that
  negates the assumption, section 16.4). Evidence that is not an obligation is an ERROR. While
  the evidence is in the owed list the assumption counts as unproven.
- Stage `formal` reads no receipts: whether the evidence is in fact proven is `verify`'s
  judgment, which fails on an obligation that is neither proven nor owed.
- Every unproven assumption prints in stage `formal` as
  `warn   <spec>: assumption <id> unproven: <text>`, in every proof block that touches the area
  (section 13) and in the verifier brief. It cannot be hidden, and it does not block.

### 16.4 Runs, reach predicates and findings

A run is one configuration of the model.

| Key | Meaning |
|---|---|
| `name` | `^[a-z][a-z0-9-]+$`, unique in the spec. |
| `constants` | TLA+: values for the constants. `Instances` and `Callers` are counts and become sets of strings (`"i1"`, `"c1"`, ...); defaults 1, 2 and `Atomic: true`. Alloy: `scope` is the only constant (default 4). |
| `spec` | TLA+: the specification to check; default `Spec`. |
| `invariants`, `properties` | TLA+: state invariants and temporal properties. Alloy: `invariants` are assertions, checked with `check <name> for <scope>`. |
| `reach` | Predicates the model must be able to satisfy. TLA+: a state predicate, reached when its negation is violated as an invariant. Alloy: a predicate, reached when `run <name> for <scope>` finds an instance. |
| `expect` | `pass` or `finding`. |
| `assumption` / `story` | Required with `expect: finding`: the assumption the run negates, or the tracker id of the known bug it records. Exactly one. |
| `timeout_seconds` | Default 300. A tool run that reaches it did not complete (section 16.9). |

**Reach.** A property that holds because the interesting state is never reached proves nothing.
Every run with `expect: pass` lists at least one reach predicate, and each must be reached:
otherwise `ERROR  <spec> run <run>: reach <name> not reached: the pass is vacuous`.

A violated run's trace is taken with one worker, whatever `--workers` says, so that the trace
file does not change between runs.

**Findings.** A run with `expect: finding` is expected to fail. It records a known bug (`story`)
or shows what breaks when a host assumption is false (`assumption`). It keeps failing in the
open; nobody weakens the property to make it pass. When such a run passes:
`ERROR  <spec> run <run>: expected a finding and the run passed; the bug is gone or the model
changed: set expect: pass`.

A run with `expect: pass` that is violated is
`ERROR  <spec> run <run>: <property> violated; replay <cx id> against the real system`.

### 16.5 Mutations

A property that no bug can violate checks nothing. Beside the properties, each spec lists
plausible bugs. A mutation is a text swap in one of the spec's files and names the property that
must catch it.

- `id` matches `^M[0-9]+$`; `bug` says what would be wrong, in the system's words.
- `file` is the skeleton or the property file (default: the skeleton). `from` must occur exactly
  once in it; it is replaced by `to` in a temporary copy. The tree is never modified.
- `detects` names an invariant or property of `run`, which must be a run with `expect: pass`.
- `origin` is optional: the commit of a bug fix, which must exist. A past bug becomes a mutation
  that puts the old behavior back, so the model is shown to catch what already went wrong once.

`machinery formal check` first applies each mutation. When `from` does not occur exactly once in
its file, the mutation is `inapplicable`: that is decided before any tool runs, and it is the only
way a mutation is inapplicable. Otherwise the mutated spec runs with only the named property, and
the mutation is `detected` (the run completed and the property is violated) or `missed` (the run
completed and the property holds). When the run did not complete (section 16.9), a mutated spec
the tool cannot parse included, the mutation is `incomplete` with its `reason`, never detected,
missed or inapplicable.

Stage `formal`: `ERROR  <spec> mutation <id> missed: <detects> does not catch "<bug>"`,
`ERROR  <spec> mutation <id> does not apply`, and for every invariant and property of an
`expect: pass` run that no mutation names:
`ERROR  <spec>: <property> has no mutation it must catch`. `TypeOK` is exempt. An `incomplete`
mutation makes the stage not demonstrated (section 16.9).

### 16.6 Baseline

An agent asked to make a failing check pass can weaken the property instead of fixing the system.
The baseline makes that visible and deliberate. `baseline.json`:

```json
{
  "version": 1,
  "specs": {
    "order-payment": {
      "properties": {"ChargedOnce": "<sha256>", "Settles": "<sha256>", "SomePaid": "<sha256>"},
      "fairness": "<sha256>",
      "runs": {"strict": {"expect": "pass", "constants": {"Atomic": true, "Callers": 2, "Instances": 2}}},
      "accepted": [{"at": "<short sha>", "change": "property Settles changed", "reason": "<text>"}]
    }
  }
}
```

- A property hash is the sha256 of the definition's text with runs of whitespace collapsed: for
  TLA+ from `<Name> ==` to the next definition that starts a line or the end of the module; for
  Alloy the `assert` or `pred` block. Every name used in a run's `invariants`, `properties` and
  `reach` is recorded.
- `fairness` is the hash of the definition named `Fairness` in effect (empty for Alloy).

Stage `formal` compares the tree with the baseline:

| Difference | Finding |
|---|---|
| A spec, property or run the baseline does not have | `DRIFT  baseline: <spec>: <what> not recorded; run machinery formal baseline` |
| A recorded property is gone or its hash differs | `ERROR  weakened: <spec>: property <name> removed` or `changed` |
| Fairness differs | `ERROR  weakened: <spec>: fairness changed` |
| A recorded run is gone, its `expect` went from `pass` to `finding`, a numeric constant is lower, or another constant differs | `ERROR  weakened: <spec>: run <run>: <what>` |
| A recorded spec has no manifest any more | `ERROR  weakened: <spec>: spec removed` |

`machinery formal baseline` records additions and larger bounds. It refuses the differences in the
ERROR rows unless given `--accept --reason "<text>"`, which records the new state and appends to
`accepted`. Stage `ratchet` shows acceptances the base did not have (section 11.3), and the
verifier brief lists them (section 15.5). A changed property is always called weakened: whether a
change strengthens is a judgment, and the reason records it.

### 16.7 Sync

A model describes the sources as someone read them at one commit. `sources.read_at` is that
commit and `sources.paths` the files and globs that were read. The spec's own files are never
sources.

Stage `formal`: when a commit after `read_at` touches a source path,
`DRIFT  sync: <spec>: <n> commit(s) touched its sources since <short sha>; classify with
machinery formal sync <spec>`. A `read_at` that is not an ancestor of the current commit is an
ERROR.

`machinery formal sync <spec>` lists those commits and the files they changed.
`machinery formal sync <spec> --class <class> [--evidence "<text>"] [--story <id>]` classifies:

| Class | Meaning | Requires | Effect |
|---|---|---|---|
| `no-effect` | The change does not touch what the model describes. | `--evidence` saying why. | `read_at` moves to the current commit. |
| `model-drift` | The model was behind and has been updated from the code. | A file of the spec differs from its content at `read_at`. | `read_at` moves. |
| `promise-change` | The promise itself changed on purpose. | `--evidence` naming the decision. | `read_at` moves. |
| `possible-bug` | The code may now break the promise. | `--story`. | `read_at` stays. Stage `formal` reports `ERROR  sync: <spec>: possible bug under investigation (<story>)` until a later classification. |

Each classification appends `{at, from, class, evidence, story}` to `sources.log`. The model is
updated from the code, never from the result of a check.

### 16.8 Tools

The pinned tools:

| Tool | Version | File in the tools directory | sha256 |
|---|---|---|---|
| tla2tools | v1.7.4 | `jars/tla2tools.jar` | `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88` |
| Alloy | v6.2.0 | `jars/alloy.jar` | `6b8c1cb5bc93bedfc7c61435c4e1ab6e688a242dc702a394628d9a9801edb78d` |

`machinery formal tools` downloads what is missing from the projects' release pages into the
tools directory (section 2) and verifies the checksum; `--check` only verifies. A file with
another checksum is never used and never replaced: `ERROR  <file>: checksum mismatch`. The
environment variable `MACHINERY_TOOLS_MIRROR` names a base URL (`https://` or `file://`) that holds
the two files under the names `tla2tools.jar` and `alloy.jar`, for offline use; the checksums
still apply. `machinery formal check` needs
the jar of each tool it runs and a Java runtime (`tools.java`); when one is missing the command is
`not demonstrated` with the reason and exits 1. It never passes on absence.

### 16.9 Results and stage `formal`

`machinery formal check [<spec>...] [--workers <n>]` (all specs when none is named; workers
default 2), for each spec: verifies the skeleton is current (else `ERROR  run machinery formal gen
first`), copies the spec's files to a private temporary directory, runs every run, then every reach
predicate, then every mutation, writes trace files (section 16.10), writes
`<spec>.result.json`, and regenerates `obligations.json` so new `cx` ids appear. It prints stage
`formal` for the specs it ran.

```json
{
  "version": 1,
  "spec": "order-payment",
  "tool": "tla",
  "tool_version": "tla2tools v1.7.4",
  "input_sha256": "<sha256>",
  "runs": [
    {"name": "strict", "outcome": "passed", "violated": "", "reach": {"SomeCancelled": true, "SomePaid": true}},
    {"name": "race", "outcome": "violated", "violated": "ChargedOnce", "reach": {}, "trace": "traces/order-payment.race.trace.json"}
  ],
  "mutations": [{"id": "M1", "outcome": "detected"}]
}
```

`machinery formal check --verify` runs the tools the same way, writes nothing, and prints a block
named `formal-verify` with `ERROR  <path>: a fresh run of the tools does not reproduce the
committed file` for every result or trace file that differs. Committed results are data anyone
can edit; `machinery landing` runs this step, so a result that the tools did not produce does not
land. When a fresh run does not complete, nothing is compared and the block is
`not demonstrated: <spec> run <run> did not complete: <reason>` (or `mutation <id>`): a run with
no verdict neither confirms nor refutes a committed file.

`outcome` is `passed`, `violated` or `incomplete`. `violated` names the violated property;
`reason`, present only when the outcome is `incomplete`, says why the run did not complete. A
mutation's `outcome` is one of section 16.5, with `reason` when it is `incomplete`. `input_sha256`
is the sha256 over the manifest without its `sources` block (canonical JSON), the skeleton bytes
and the property file bytes. The file has no timestamps and no durations.

**Runs that did not complete.** A tool run completes when the tool ends on its own with a
verdict: TLC exits 0 after `Model checking completed. No error has been found.`, or exits with a
violation status (10 to 14) after naming the violation; Alloy exits 0 and writes a receipt that
holds every command machinery gave it. Anything else did not complete: the run reached
`timeout_seconds`, the tool could not start, crashed or was killed, reported an error (a spec it
cannot parse included), or left output that cannot be read. Such a run gives no verdict. When a
run's own check, one of its reach predicates, the check that isolates the violated property or
the one-worker trace (16.4) did not complete, the run is `incomplete`; when a mutation's run did
not complete, the mutation is `incomplete`. `reason` says why, naming files relative to the
spec's directory: `timed out after 300s`, `TLC was killed: signal: killed`, or the tool's first
error line. An incomplete run is never judged passed, violated, reached or not reached, and its
trace file is left as it was; an incomplete mutation is never detected, missed or inapplicable.

Every tool run has a private working directory and a private Java temporary directory
(`java.io.tmpdir`), so runs at the same time, of one command or of several, share no file. TLC
unpacks its standard modules (`Naturals.tla` and the others) into the Java temporary directory
and deletes them when it exits; with a shared one, a run can delete or truncate the modules
another run is reading.

Stage `formal` applies when the formal directory holds a manifest or the risk file has a critical
or vital area. With no manifest at all and every such area owed it is
`not applicable: no formal model yet: <n> critical area(s) owed` (vital areas counted among them). It reads the manifests, the results, the baseline, the risk file and git history, and runs
no tool:

- manifest errors; skeleton drift (as `formal gen --check`);
- `not demonstrated: no result for <spec>; run machinery formal check` when a result file is
  missing; `DRIFT  <spec>.result.json: stale` when `input_sha256` does not match;
- the run, reach and finding rules of 16.4, the mutation rules of 16.5, the baseline rules of
  16.6, the sync rules of 16.7, the assumption warnings of 16.3, the coverage rule of 16.1;
- `ERROR  <spec>: unknown area <name>`;
- `warn   <spec> run <run> did not complete: <reason>` and
  `warn   <spec> mutation <id> did not complete: <reason>` for every incomplete run and
  mutation, and the stage is `not demonstrated: <the first of them>; fix the cause and run
  machinery formal check` (runs before mutations, in manifest order). There is no verdict to
  judge, so the stage neither passes nor fails on that account; when it also has an ERROR or
  DRIFT it is failed (section 3.2), and the warn lines still say what did not complete.

An `fv:<area>` obligation is proven when the area is covered (16.1), no ERROR or DRIFT names a
spec that lists the area, and every such spec has a result in which every run and mutation
completed. When `fv:<area>` is in the owed list, findings about specs whose areas are all owed
print as `warn   owed: <message>`, and such a spec's missing result or incomplete runs do not make
the stage not demonstrated: a model under construction does not block unrelated work.

`checked:` counts specs, runs, reach predicates, mutations and assumptions.

### 16.10 Counterexamples and replay

Every violation the tools find is written as a trace file and becomes an obligation
`cx:<spec>:<run>`. A counterexample is a claim about the real system, and only the real system can
settle it.

```json
{
  "version": 1,
  "id": "cx:order-payment:race",
  "spec": "order-payment",
  "run": "race",
  "tool": "tla",
  "violates": "ChargedOnce",
  "expect": "refuted",
  "machine": "order",
  "steps": [
    {"n": 1, "case": "tc:ORDE-2e883d", "instance": "i1", "caller": "c2", "step": "begin", "state": {"i1": "Pending"}},
    {"n": 2, "case": "tc:ORDE-2e883d", "instance": "i1", "caller": "c1", "step": "begin", "state": {"i1": "Pending"}},
    {"n": 3, "case": "tc:ORDE-2e883d", "instance": "i1", "caller": "c1", "step": "commit", "state": {"i1": "paying"}},
    {"n": 4, "case": "tc:ORDE-2e883d", "instance": "i1", "caller": "c2", "step": "commit", "state": {"i1": "paying"}}
  ],
  "loop": 0,
  "instance": {}
}
```

- TLA+: `steps` come from the `last` and `st` variables of each state after the initial one;
  `state` is `st` after the step. `loop` is the step a liveness counterexample returns to, else 0.
- Alloy: `steps` is empty and `instance` maps each atom to its fields, as the tool reports them,
  with atom names stripped of the module prefix and of `$` (`Order0`).
- `expect` says what the real system must do when a test replays the trace:

| The run | `expect` | Meaning |
|---|---|---|
| `expect: finding` with `assumption` | `refuted` | The real host keeps the assumption, so the trace cannot be followed to its end. A replay that reproduces it shows the assumption is false: a real bug. |
| `expect: finding` with `story` | `reproduced` | A known bug. The replay reproduces it until the story fixes it; then the test fails, and the run is changed to `expect: pass`. |
| `expect: pass`, violated | `undetermined` | Stage `formal` already fails. The replay tells which side is wrong: reproduced means the code, refuted means the model. |

A project's end-to-end suite replays a trace by driving the real system through the steps: steps
of one instance whose `begin` steps precede their `commit` steps are fired concurrently, and the
test observes whether the violated property's bad outcome occurs. It emits one receipt for the
`cx` id with `replay: reproduced` or `replay: refuted` (section 10). Stage `verify` compares
(section 12). With `undetermined` no replay proves the id: a reproduced replay is reported as a
real bug and a refuted one as a wrong model, and both are failures until the code or the model is
fixed.

A trace file whose run no longer produces a violation is removed by `machinery formal check`, and
its `cx` id disappears with it. Trace files are generated and never edited.

## 17. Starting: `init` and `adopt`

### 17.1 `machinery init`

For a new project. Creates, in an empty or existing directory with no `machinery.yaml`: a
`machinery.yaml` in which nothing applies yet, `design/model/`, `design/machines/`, an empty
`design/owed.yaml`, and the line `.machinery/` in `.gitignore` when it is not there; and prints
the Makefile targets the project should add (`check`, `verify`). It never overwrites a file.

### 17.2 `machinery adopt`

For a project that already has code, tests, a way of working and a backlog. Adoption must not
need a rewrite, a freeze or a migration of the backlog. It starts from the code and the backlog as
they are, covers only what it is pointed at, and is widened over time.

```
machinery adopt [--paths <glob>]... [--e2e <glob>]... [--write]
```

Without `--write` it changes nothing and prints the report and the files it would write. The
report:

```
adopt: <root directory name> at <short sha>
scope: <globs>
languages: <go, typescript, ... found in scope>
code files: <n> in scope
test files: <n> under e2e roots, <n> elsewhere (unit tests), <n> with test doubles
design: <n> model(s), <n> machine(s), <n> contract(s)
backlog: bd, <n> open, <n> ready, <n> closed, cycles: <none|n>
would write:
  <path>  <what>
next:
  <one line per suggested step>
```

`backlog:` is `none found` when the project has no `.beads` directory. Files under the gen
directory are generated, never design sources: adopt does not find models, machines or contracts
there. An `import` section is configured by hand (section 2); adopt detects nothing for it, but
it works with an existing `machinery.yaml` that has one: `design:` counts the files
`import.machines.files` matches as machines, and adopt never adds `design.machines` beside
`import.machines`. With `--write`:

1. **Config.** Without a `machinery.yaml`, writes one: `scope` from `--paths`, `tests.e2e` from
   `--e2e` (default `e2e/**`), and, once design sources exist, `trace.code` equal to the scope
   and `trace.tests` equal to the e2e roots, with the model, machine and contract files found (`*.modelith.yaml`, `*.machine.json`, `openapi.{yaml,yml,json}`),
   `tests.baseline`, and `backlog.tool: bd` when a backlog was found. With an existing file, adds
   the new `--paths` to `scope`, adds design sources it finds that are not configured yet (so a
   model written after the first adoption is picked up), and changes nothing else in it;
   comments are kept.
2. **Baselines.** Writes the tests baseline with every unit test and test double now in scope
   (section 9.4), and the architecture baseline when `constraints` are configured. Existing
   entries are kept; entries are added only for paths newly in scope.
3. **Owed.** Runs `machinery gen` (with no design yet it writes an empty obligation list). When
   design sources exist and stages `model`, `machines` and `failure-modes` pass, it adds to
   `owed.yaml` every obligation and every unserved `act` id that is new: neither in the owed
   list nor in the obligation list of the last commit (an id that list already has was owed
   before, or has been proven and pruned since; owing it again would undo the ratchet). Each
   is added with one of three reasons, the first that applies:

   | Reason | For |
   |---|---|
   | `planned at adoption (<short sha>)` | An obligation whose source carries one of the project's own planned marks (`import.planned`, section 2): an invariant by its statement, a scenario by its description, a catalog row by a cell. The entry gets `story: <id>`, the story the mark names. It is owed like any other, also when a test claims it: the project said it is not delivered. |
   | `claimed at adoption (<short sha>)` | An obligation a test file already claims, by a tag or a marker (section 8). No receipt exists yet, so it is owed; `machinery plan` wires its evidence instead of giving it a story (section 15.2). |
   | `unproven at adoption (<short sha>)` | Every other obligation, and every unserved `act` id. |

   It never removes an entry and never changes a reason, so an entry owed before adoption keeps
   the reason it has. A planned mark whose `story` is not a story id (section 2) stops adoption
   before anything is written, with or without `--write`: one line per mark,
   `<file>:<line>: <id>: import.planned entry <n> captured the story "<text>"; a story id has only letters, digits, ".", "-" and "_": tighten the pattern`,
   the first after `machinery: `, and exit 1. The line is that of the invariant's statement, of the
   scenario's description or of the catalog row. A catalog row whose links entry names a subject
   and no required value is no finding of these stages (section 5.4.1): it is owed like any row,
   and stage `xref-failure` counts it. When a stage fails the config and the baselines
   are still
   written, no owed entry is written and the exit code is 1. The report then says exactly what
   blocks, and nothing else (warnings do not block and are not printed):

   ```
   blocked: machinery gen refuses while <n> finding(s) stand; no owed entry is written until they are fixed
     <stage>: <message>
     <stage>: not demonstrated: <reason>
   fix in the project first: no waiver covers a finding of stage model, machines or failure-modes; only cross-reference rules are waived (section 6.7)
   ```

   one line per ERROR of the stages `model`, `machines` and `failure-modes`, in that order and
   sorted within a stage, and one per stage that is not demonstrated; the last line prints when
   an ERROR was listed. No adoption waiver is offered for these: a waiver covers only a
   cross-reference rule (section 6.7), and steps 4 and 5 add one only for the rules they name. A
   finding about the form of an imported failure catalog (a row in no table, a duplicate id, a
   table without its column, an empty cell, an id that does not lower-case to a valid one,
   section 5.4.1) is fixed in the catalog itself.
4. **Failure waivers.** Findings of `failure-state-mode` and `failure-event-mode` (section 6.6)
   must not block day one: a failure state or failure event the design already has is
   unmodeled, not new. For each such finding no waiver covers, it adds a waiver to
   `machinery.yaml` (written with step 1) with the reason `unmodeled at adoption (<short sha>)`.
   It never removes or rewrites a waiver, so the list only shrinks as failure modes are written:
   a waiver whose finding is gone is an ERROR `stale waiver` (section 6.7). While the
   failure-mode file has an ERROR it adds none, since what the file names is then unknown.
   Without `--write`, the `machinery.yaml` line under `would write:` says `new waivers: <n>`.
5. **Unbound waivers.** Findings of `machine-binding` and `entity-machine` (section 6.4) must not
   block day one either: a machine the design already has that binds to nothing, or an entity
   with actions and no lifecycle machine yet, is unbound, not new. For each such finding no
   waiver covers, it adds a waiver with the reason `unbound at adoption (<short sha>)`, one per
   rule and subject. It never removes or rewrites one, so the list only shrinks as machines are
   bound and lifecycles written. While a model or a machine cannot be loaded (a file that cannot
   be read or parsed, or a translation finding of section 5.2.1) it adds none, since what binds
   to what is then unknown. Without `--write`, the `machinery.yaml` line says
   `unbound waivers: <n>`. A waived machine is still unbound: while no machine is the lifecycle
   machine of a model entity, stage `xref-model-machine` is not demonstrated (section 6.4).

`adopt` never creates, changes or deletes a file outside `machinery.yaml` and the design
directory. That includes `.gitignore`: when the receipts directory is not ignored it prints
`note: add .machinery/ to .gitignore` and leaves the edit to the team. It prints `adopt: wrote <n> file(s)` and the paths. Running it twice with the same
arguments writes nothing the second time.

After `adopt --write`, `machinery check` passes: what exists is baselined, owed or waived, and
anything new in scope is held to the rules. The one exception is a design in which no machine is
the lifecycle machine of a model entity: nothing about the machines has been compared with the
model yet, so `xref-model-machine` is not demonstrated until the first one is bound. From there:

- the area's design is written or extended a piece at a time (the riskiest stateful resource
  first), and each `machinery gen` plus `machinery adopt --write` owes the new obligations;
- `machinery plan --apply` adds stories for the owed ids to the existing backlog, beside the
  stories already there, and `machinery plan --link` attaches ids to stories that already exist;
- stories prove ids and prune the owed list; baselines shrink as files are touched;
- `machinery adopt --paths <more> --write` widens the scope when the team is ready.

The ratchet (section 11.3) holds each step: scope does not narrow, baselines do not grow without
a wider scope, and nothing proven becomes owed.

### 17.3 Rehearsing: `machinery adopt --rehearse`

Adoption writes into the project. Before it does, `--rehearse` adopts a copy and reports what
adoption would take, as `assess` reports where the project stands:

```
machinery adopt --rehearse [--config <file>] [--out <dir>] [--paths <glob>]... [--e2e <glob>]...
```

- The project, `--config`, `--out` and the snapshot are those of `assess` (sections 18.1 and
  18.2), the same code with `adopt --rehearse` in the messages and `machinery-rehearse-*` as the
  default out directory. Without `--config` the project's `machinery.yaml` is used when it
  exists; without either the snapshot has none and adoption starts from scratch, as `adopt` does.
  `--write` is a usage error (`usage: --rehearse writes only in its out directory; it takes no --write`).
- In the snapshot it runs `machinery adopt --write` with the `--paths` and `--e2e` given,
  commits the result (hooks and signing off), runs the stages of `machinery check`, and builds
  what `machinery plan` would print, which it writes to `<out>/plan.md` and never applies.
- It never runs the backlog tool: adopt reports the backlog as `backlog: bd, not read in a
  rehearsal` when a `.beads` directory exists, and stage `intake` is
  `not applicable: a rehearsal never reads the backlog`.
- Nothing is written in the project. The output of adopt and of check is saved as
  `<out>/stages/adopt.txt` and `<out>/stages/check.txt`.
- Every commit a file of the rehearsal names is the project's HEAD that was snapshotted, the
  `<short sha>` of `project: <directory name> at <short sha>`: in the reasons adopt writes to
  `owed.yaml` (`planned at adoption (<short sha>)`, `claimed at ...`, `unproven at ...`), in the
  reasons of the waivers it adds to the snapshot's `machinery.yaml` (`unmodeled at adoption
  (<short sha>)`, `unbound at ...`) and in every `waivable:` line (`found at adoption
  (<short sha>)`). The commits of the snapshot itself (`snapshot of <sha>`, then `rehearsed
  adoption`) exist so that the stages have a commit to read; no file of the out directory names
  one, outside the snapshot's own `.git`.

`<out>/rehearsal.md` (for people) and `<out>/rehearsal.json` hold the same content, without a
timestamp or an absolute path:

| Part | What |
|---|---|
| `backlog` | The backlog line of the adopt report (section 17.2): `backlog: none found`, or `backlog: bd, not read in a rehearsal`. |
| `files` | The files adopt wrote in the snapshot, each with its size in bytes. |
| `waivers` | The waivers adopt added, by rule, with counts. |
| `baselines` | The tests and architecture baselines, each with its number of entries. |
| `owed` | The entries of the owed list, by kind (`tc`, `sc`, `inv`, `fm`, `fv`, `cx`, `ck`, `act`), and their total; then the same ids as the `owed,` lines of `machinery status` count them (section 14), each after `- ` instead of `owed, `: `- claimed by a test, no receipt yet: <n> (<kind> <n>, ...)`, `- no test names it: <n> (...)` and, when they count something, `- planned: <n> (...)`, `- proven by a receipt, still listed: <n> (...)` and `- actions: <n> served by code, <n> not served`. |
| `blocked` | When adopt was blocked (section 17.2): the lines of its `blocked` report. |
| `check` | The summary line of `machinery check` and the outcome line of each stage. |
| `plan` | The stories `machinery plan` would create, by kind (the kind of their ids, and `wire`, listed last, for a wire-evidence story) and tier, with counts, their total and the number of edges. The total is exact unless every id of a story carries a story already: `--apply` skips such a group when that story is in the backlog, which a rehearsal never reads, so the line then starts `Up to <n> stories` and ends `; <m> may exist already: every id of the group carries a story, and a rehearsal never reads the backlog.` Both groups show: when the plan has a wire-evidence story, `- wire evidence: <n> stories for <n> claimed id(s), one per test <file|directory>` (`1 story`), and when ids were planned at adoption, `- already planned in the project: <n> id(s) planned at adoption keep the story the project gave them; no story is planned for them`. |
| `blocks` | What blocks adoption: each stage of the check that failed or was not demonstrated, with its outcome line and its first 5 findings, and the number of the others. |
| `fix_first` | Fix in the project first: every ERROR of stage `model`, `machines`, `failure-modes` or `risk`. No waiver covers one (section 6.7); a finding about the form of an imported failure catalog is one of them. |

A finding under `blocks` carries its disposition: `fix in the project first` for an ERROR of the
stages of `fix_first`; for an ERROR of a cross-reference rule (sections 6.4 to 6.6, the message
`<rule>: <subject>: <what>`), the waiver that would cover it, offered and never written:
`waivable: - {rule: <rule>, id: "<subject>", reason: "found at adoption (<short sha>)"}`; none
otherwise. `rehearsal.json` is `{version, project, head, config, backlog, dirty, outside_links, files:
[{path, bytes}], waivers: [{rule, count}], baselines: [{path, entries}], owed: {by_kind, total,
claimed: {by_kind, total}, unnamed: {by_kind, total}, planned: {by_kind, total}, proven: {by_kind, total},
actions: {served, unserved}},
blocked: [<line>], check: {result, stages: [{name, outcome}]}, plan: {stories: [{kind, tier,
count}], total, edges, planned_ids, may_exist, wire: {stories, ids, by}}, blocks: [{stage, outcome, findings: [{severity, message, disposition,
waiver}], more}], fix_first: [{stage, message}]}`, keys sorted.

stdout:

```
out: <absolute path of the out directory>
project: <directory name> at <short sha>
dirty: <n> changed, <n> untracked file(s) are not rehearsed
outside links: ...
adopt: wrote <n> file(s), <n> waiver(s), <n> owed
adopt: blocked: machinery gen refuses while <n> finding(s) stand; nothing is owed
check: machinery check: <p> passed, <f> failed, <d> not demonstrated, <n> not applicable
plan: <n> stories, <n> edge(s)
plan: up to <n> stories, <n> edge(s); <m> may exist already: a rehearsal never reads the backlog
wire evidence: <n> stories for <n> claimed id(s), one per test <file|directory>
blocks adoption: <n> stage(s): <stage>, <stage>
fix in the project first: <n> finding(s)
rehearse: rehearsal.md, rehearsal.json
```

One of the two `adopt:` lines and one of the two `plan:` lines print (the second when a story's
ids all carry a story already); the `wire evidence` line only when the plan has such a
story (`1 story`); `blocks adoption: nothing` when every stage passed or was
not applicable, and the `fix in the project first` line only when there is one. Exit 0 when the
rehearsal ran, whatever it found; 1 when the snapshot could not be taken; 2 for a usage or config
error, nothing written.

## 18. Assessing: `machinery assess`

Before a team decides to adopt, `machinery assess` gives a deterministic account of where an
existing project stands against its own design. It is read-only: nothing is written inside the
project, ever, including its `.git` directory and its tracker.

```
machinery assess [--config <file>] [--out <dir>] [--since <git-ref>]...
```

### 18.1 Inputs and refusals

- The project is the directory `-C` names. Without it, the nearest directory upward from the
  current one that holds `machinery.yaml`, else the top of the git work tree that holds the
  current directory. It must be a git repository with a commit: assess reads `HEAD`.
- `--config` names the `machinery.yaml` to use. It may live outside the project; its paths are
  relative to the project root as always (section 2). Without it the project's own
  `machinery.yaml` is used. Either is read from the file, never from `HEAD`. A relative
  `tools.dir` is resolved against the project, which assess reads and never writes.
- `--out` is the directory for everything assess produces. It must be absent or an empty
  directory, outside the project and outside the git work tree that holds it. Default: a new
  directory `machinery-assess-*` under the system temp directory, which must then be outside the
  project too.
- `--since` (repeatable) names commits for the history section (section 18.6).

Each refusal exits 2 before anything is written:

| Case | stderr |
|---|---|
| Neither `--config` nor a `machinery.yaml` | `config: no machinery.yaml in <dir> and no --config: write one by hand (docs/brownfield.md, step 1) and pass it with --config` |
| An invalid config | `config: <message>` (section 2) |
| No commit | `usage: <dir> is not a git repository with a commit; assess reads its HEAD` |
| `--out` inside the project or its work tree | `usage: --out <dir> is inside <root>; assess never writes in the project: name a directory outside it` |
| `--out` not empty, or not a directory | `usage: --out <dir> is not empty; name an absent or empty directory` |
| An unknown `--since` | `usage: --since <ref>: not a commit of the project` |

### 18.2 The snapshot

assess pipes `git archive HEAD`, run in the project root (a project inside a larger repository
is archived alone), into an extraction at `<out>/snapshot`: no worktree and no checkout. It
writes the config there as `machinery.yaml`, runs `git init` and one commit inside the snapshot
(hooks and signing off) so that stages that need a commit work, and runs every section there.

The extraction writes through a root bound to `<out>/snapshot`, so nothing lands outside it. A
symbolic link of `HEAD` whose target is absolute, or resolves outside the snapshot (by `..`, or
through another link), is not created: what it points at is not assessed. stdout and the report
say `outside links: <n> symbolic link(s) leave the project and are not in the snapshot: <paths>`,
and the inventory has `warn   outside-link: <path>: leaves the project; not in the snapshot, not
assessed` for each. Links that stay inside are kept. An archive entry whose name is absolute or
holds a `..` segment stops the snapshot (exit 1).

What assess reads of the project itself: `git` commands that write nothing (`archive`, `show`,
`log`, `rev-parse`, `for-each-ref`, and `status` with optional locks off, so the index is
not refreshed), the config file and the tracker export (section 18.8). When the work tree
differs from `HEAD` the report says `dirty: <n> changed, <n> untracked file(s) are not
assessed`: changed counts the tracked files that differ, untracked the files git neither tracks
nor ignores. They are not in the snapshot.

### 18.3 Sections

In this order. Each has a result (`passed`, `failed`, `not demonstrated`, `not applicable`,
decided as for a stage, section 3.2), its `checked:` counts and the count of its findings per
rule, most frequent first. A section made of stages fails when one of them fails, else is not
demonstrated when one is, else passes when one passes, else is not applicable. A section that
crashes is failed with `ERROR  crashed: <error>`, and the others still run. The full output of
each section is saved under `<out>/stages/`.

| Section | What runs | Saved as |
|---|---|---|
| `inventory` | The facts of the adopt report (section 17.2) for the snapshot: scope, languages, code files, test files (under e2e roots, elsewhere, with test doubles), design sources. `checked:` counts the files. | `inventory.txt` |
| `design` | The stages of `machinery lint` (section 3.1), and one ERROR of the section's own per problem of the owed file (section 11.1). When at least one machine was read, `checked:` adds the machines bound to the model and those unbound (section 6.4): `checked: 8 stages, 1 bound machines, 1 unbound machines`. The line `ids: <n> proposed, <n> owed, <n> proven, <n> retired` counts the ids by status (section 14). When the risk file has a vital area, `proofs awaiting: <n>` follows, then `vital: <id> proven (by test and model; proof awaited)` for every id a property is over that is not owed, and `assumed, unproven: <area> <name>: <statement>` for every assumption whose evidence is owed (section 23.2). Then the candidates of section 23.3, always: `vital candidates: <n>, dismissed: <n>`, and one line `  <id>  <reason>` per candidate. | `<stage>.txt` each |
| `generation` | `machinery gen` (section 7). When gen refuses, not demonstrated: `gen refused: <stages> have errors`. | `generation.txt` |
| `trace` | The stage of that name. With a `trace` block, the `owed,` lines of `machinery status` (section 14) follow, counting the owed ids of the snapshot the same way, whether or not the project is adopted: `owed, claimed by a test, no receipt yet: <n> (<kind> <n>, ...)`, `owed, no test names it: <n> (...)` and the others when they count something. | `trace.txt` |
| `tests`, `arch` | The stage of that name. | `<stage>.txt` |
| `formal` | The sanity run, section 18.5. | `formal.txt` |
| `history` | Section 18.6. | `history.txt` |
| `decisions` | Section 18.7. | `decisions.txt` |
| `backlog` | Section 18.8. | `backlog.txt` |

### 18.4 The rule of a finding

A finding's rule is the name of its stage or section and the leading rule word of its message,
when it has one that is not a design name (`trace: unserved`,
`xref-failure: invoke-failure-mode`). Otherwise the message is reduced to a pattern: leading file
names (`<path>:` or `<path>:<line>:`, and JSON pointers) are dropped, the text is cut at the
first `; ` or ` (`, and every word that is not plain lower-case text, or is a name the design
declares (a machine, state, event, entity, enum value, action, invariant or failure-mode id),
becomes `*` (`machines: dead-end non-final leaf state *`). A rule counts its findings of one
severity.

### 18.5 Formal sanity

Not the formal models of section 16: those carry hand-written properties, and their manifests and
results record no state counts, so the sanity run is kept separate and writes nothing under
`design/formal`.

**Machines.** For every machine, assess writes `<out>/formal/machines/<machine id>/Sanity.tla`
from its runtime table (section 7.3): one variable `st` over the leaf states; the initial leaf
and the leaves of the entry states (`_entries`) as the initial states; one action per
transition row, from each leaf of its source to the leaf its target reaches (an internal row
leaves `st` unchanged; guards are not modeled, any branch may fire); and a stutter step in final
leaves and in the leaves of states declared `_terminal`. It holds no hand-written property. TLC
runs it with one worker and deadlock checking on, dumping the states it found; after a deadlock it
runs again with deadlock checking off for the full set of reachable states. Findings:

- `ERROR  <machine>: unreachable state <state>`: no behavior from the initial or an entry state
  reaches the leaf.
- `ERROR  <machine>: deadlock in state <state>: neither final nor _terminal, and no transition
  applies`. TLC reports the first deadlock it reaches.

**Models.** For every model, assess writes `<out>/formal/models/<module>/<module>.als` (the
skeleton of section 16.2) and `Sanity.als`, which opens it and runs `Inhabited` (an atom of every
entity signature at once; `some univ` when there is none) `for 4`. Alloy must find an instance:
otherwise `ERROR  <model>: no instance in which every entity exists (for 4)`.

**Runs that did not complete.** A sanity run completes as a run of section 16.9 does: TLC exits
0 after `Model checking completed. No error has been found.`, or with the deadlock status after
naming the deadlock; Alloy exits 0 and writes a receipt that holds `Inhabited`. Anything else
gave no verdict: the run reached 300 seconds, the tool could not start, crashed or was killed,
reported an error (a generated spec it cannot parse included), reported a violation the sanity
spec has no property for, or left output that cannot be read (no state dump). A machine with no
state has no run. Such a machine or model is never judged passed or failed on its account:
`warn   <machine>: the sanity run did not complete: <reason>` (or `<model>`), with the reason of
section 16.9 naming files relative to the run's directory, and the section is
`not demonstrated: <the first of them>; fix the cause and run machinery assess again` (machines
in design order, then models). When the section also has an ERROR it is failed (section 3.2),
and the warn lines still say what did not complete. A deadlock found by the first TLC run stays
an ERROR when the run that explores past it does not complete.

The tools are the pinned jars and the Java runtime, found as `machinery formal check` finds them
(section 16.8). When one the project needs is missing the section is not demonstrated with the
reason, never passed. Lines: `<machine>: <n> states, <n> distinct states found, passed` (or
`failed`) per machine, or `<machine>: <n> states, did not complete`; `<model>: instance found`
(or `no instance`, or `did not complete`) per model. `checked:` counts machines, states and
models.

### 18.6 History

For each `--since` ref, or by default for each configured model file the first commit that adds
it (one comparison per distinct commit), assess compares the ids of the models at that commit,
read with `git show <ref>:<path>` on the project, with those at `HEAD`:

| Kind | Id | Its text |
|---|---|---|
| `inv` | `inv:<id>` | the statement |
| `sc` | `sc:<slug>` | the steps |
| `entity` | the entity name | the definition |
| `act` | `act:<Entity>.<action>` | the description |

An id is added, removed, or reworded when its text differs once runs of whitespace are collapsed.
A model file absent at a ref has no ids there. Per comparison and kind, the lines give the
counts, then the removed and the reworded ids in full; added ids only as a count. Not applicable
without a configured model; not demonstrated when no model file has a commit. History has no
findings.

### 18.7 Decisions

`assess.decisions` (section 2) names a project's own decision records. For each file its globs
match in the snapshot: `<file>: <n> records, <n> lines, <n> words`, where a record is a line the
`record` expression matches. `ERROR  assess.decisions.files matches no file`;
`warn   <file>: no line matches assess.decisions.record`.

The agent instruction files at the root of the snapshot, `CLAUDE.md` and `AGENTS.md`, are
counted too: `<file>: <n> lines, <n> words`. When one is a symbolic link to the other, or holds a
single non-empty line that names the other, it is counted once, under the file it points at, with
`(<file> links to it)`.

When the tracker's export (section 18.8) can be read, the decision items it holds are counted
too, so that a project that keeps its decisions in the backlog (section 15.8) shows it:
`backlog: <n> decision item(s) in .beads/issues.jsonl, <n> current`, and `checked:` adds
`<n> backlog decisions`. A decision item is one of type `decision` or one that carries a
`machinery:decision` line; it is current unless an item of the export carries
`machinery:supersedes <its id>`. Not applicable when `assess.decisions` is not configured,
neither instruction file exists and the export holds no decision item.

### 18.8 Backlog

assess reads the tracker's export, `.beads/issues.jsonl` in the project work tree, and never runs
the tracker. Each line is a JSON object; `id`, `title`, `status` and `issue_type` are read, and
`title`, `description`, `design`, `notes` and `acceptance_criteria` are searched for ids. A
line whose `_type` is present and is not `issue` is skipped. Lines: issues by status and by type;
the issues in progress (status `in_progress`), id and title; how many open issues (every status
but `closed`, `deferred` and `tombstone`) name an obligation id of the design (the full id, or
for an imported failure mode the project's own id, in any case); and the local branches other
than the current one (`git for-each-ref refs/heads`). A line that is not a JSON object is
`ERROR  .beads/issues.jsonl:<n>: not a JSON object`. Without the file the section is
`not applicable: none found (no .beads/issues.jsonl)`, and the branches line still prints.

### 18.9 Output and exit codes

`<out>/report.md` (for people) and `<out>/report.json` hold the same content: the project
directory name, the short `HEAD` sha, where the config came from, the dirty counts, the outside
links, and per section its result, reason, `checked:` counts, number of blocking findings, rules,
lines and, for a section made of several stages, each stage's result. `report.json` also gives
the numbers of the lines as facts. Neither has a timestamp or an absolute path.

```json
{
  "config": "machinery.yaml",
  "dirty": {"changed": 0, "untracked": 0},
  "entities": [{"drift": 0, "entity": "Loan", "errors": 3, "rules": [{"count": 3, "rule": "trace: unserved", "severity": "error"}], "warnings": 0}],
  "head": "<short sha>",
  "machines": [{"drift": 0, "errors": 2, "machine": "loan", "rules": [{"count": 2, "rule": "trace: unverified", "severity": "error"}], "source": "machines/loan.json", "warnings": 0}],
  "outside_links": [],
  "project": "<directory name>",
  "sections": [
    {"checked": [{"n": 9, "noun": "files"}], "facts": {"code_files": 1}, "findings": 0,
     "lines": ["code files: 1 in scope"], "name": "inventory", "output": "stages/inventory.txt",
     "reason": "", "result": "passed", "stages": [],
     "rules": [{"count": 2, "rule": "machines: dead-end non-final leaf state *", "severity": "error"}]}
  ],
  "summary": {"failed": 0, "not_applicable": 0, "not_demonstrated": 0, "passed": 10, "sections": 10},
  "version": 1
}
```

Both reports also group the findings of every section by what they name, so a large design is
read one machine or one entity at a time. A finding names a machine when its message holds the
machine id or the machine's file as a whole word (no letter, digit or `_` right before or after
it, and for the file no `/`, `.` or `-` either), or a case id with the machine's tag
(`tc:<TAG>-`). It names an entity when it holds the entity name as a whole word, as
`entity:<E>` and `act:<E>.<action>` do. A finding may name several machines and entities, and
counts for each. `report.md` ends with `## Findings per machine` and `## Findings per entity`:
one line per machine or entity that at least one finding names, sorted by name,
`- <machine id> (<file>): <n> error(s), <n> drift, <n> warn(s)` (`- <Entity>: ...`), then its
rules (section 18.4) with their counts, most frequent first, as `  - <count> <severity>: <rule>`;
`No finding names a machine.` (`No finding names an entity.`) when none does. `report.json`
gives the same as `machines` and `entities`, sorted by name.

Keys are sorted. `config` is `machinery.yaml` or `--config <file name>`. `result` is `passed`,
`failed`, `not demonstrated` or `not applicable`, and `reason` holds the reason of the last two.
A stage entry has `checked`, `findings`, `name`, `output`, `reason` and `result`. The facts:

| Section | Keys of `facts` |
|---|---|
| `inventory` | `code_files`, `contracts`, `double_test_files`, `e2e_test_files`, `languages`, `machines`, `models`, `scope`, `unit_test_files` |
| `design` | `statuses`: `{owed, proposed, proven, retired}`; `bound_machines`, `unbound_machines` (when at least one machine was read); `proofs_awaiting`; `vital_candidates`: `{id, reason}`; `vital_dismissed` |
| `generation` | `files` (generated files), `written` |
| `trace` | `claimed`, `unnamed`, `planned`, `proven`: each `{by_kind, total}`; `actions`: `{served, unserved}` (with a `trace` block) |
| `formal` | `machines`: `{deadlock, distinct, machine, reason, result, states, unreachable}`; `models`: `{model, reason, result}`; `result` is `passed`, `failed` or `incomplete` (`instance found`, `no instance` or `incomplete` for a model), and `reason` says why a run did not complete, else is empty |
| `history` | `comparisons`: `{commit, kinds, since}`, `kinds` mapping `inv`, `sc`, `entity`, `act` to `{added, removed, reworded}` |
| `decisions` | `files`: `{lines, path, records, words}`; `instructions`: `{linked, lines, path, words}`; `backlog`: `{current, decisions}` when the export can be read |
| `backlog` | `branches`, `by_status`, `by_type`, `export`, `in_progress` (`{id, title}`), `issues`, `open`, `open_naming_obligations` |

stdout:

```
out: <absolute path of the out directory>
project: <directory name> at <short sha>
dirty: <n> changed, <n> untracked file(s) are not assessed
outside links: <n> symbolic link(s) leave the project and are not in the snapshot: <paths>
<section>: <outcome>; checked: <counts>
assess: <n> sections, <p> passed, <f> failed, <d> not demonstrated, <a> not applicable
```

The dirty line prints only when the tree is dirty, and the outside links line only when there
is one. `<outcome>` is the last line of a stage block (section 3.2); a not applicable section has
no `checked:` part. The out path is the only absolute path assess prints.

| Code | Meaning |
|---|---|
| 0 | The assessment ran, whatever it found. |
| 1 | The snapshot could not be taken; nothing was assessed. |
| 2 | A usage or config error; nothing was written. |

## 19. The order of work this supports

**Intake.** Every new requirement starts as an intake item (section 15.7), before any story and
before any code. The tool holds the order:

1. `machinery intake new "<title>" --intent "..." --refuses "..."` records it at the current
   commit.
2. The design changes first and lands alone: the model and the machines, the failure modes and
   the tiers in `risk.yaml`, a formal model where a critical or vital area asks for one
   (`machinery formal new`), then `machinery lint` and `machinery gen`, committed.
3. `machinery intake status <id>` names the step that is missing and why.
4. The checker reads the change against the requirement and records it:
   `machinery review design <id> --verdict approved|rejected` (section 15.10).
5. `machinery intake plan <id> --apply` writes every story at once from what the design now owes,
   owes the new ids, and adds the design story that lands the change and the capstone that will
   show the requirement on the running system.
6. `machinery waves` schedules the stories; the checker reviews the backlog
   (`machinery review backlog`), and `machinery waves --freeze` pins the contract for a
   wave. A design change found later is its own change that lands alone, never an edit inside a
   running wave, and `machinery intake plan` refuses while a wave is frozen.

**Plan.** The planner (the PM role) owns the backlog and the risk file. Obligations owed without a
requirement behind them (the skeleton of a new project, an adoption) become stories with
`machinery plan --apply`. Either way, a story in the design's name names the ids it proves:
`machinery waves` fails on one that does not, and counts the stories outside machinery.

**Build.** One story, one builder, one worktree, briefed with `machinery brief <story>`:

1. Write the end-to-end check for the story's ids, run it against the real stack, see it fail for
   the right reason.
2. Change the code until it passes. Per change: `machinery check --base <base> --story <id>`
   (stage `scope` holds the story to its paths) and a targeted run
   (`machinery run --affected`, `machinery verify --story <id>`).
3. `machinery owed sync --prune-only`, commit, `machinery proof --story <id> --post`, with a
   `--limit` for each claim still unproven, a `--learning` for each rule the next story should
   know and an `--observation` for each thing seen outside the story's scope.

**Accept.** By tier (section 15.6): routine on evidence; consequential, critical and vital
through a fresh verifier who predicts, reads, then accepts; critical and vital with the formal
model current; vital with the evidence of its assumptions proven. The
capstone of an intake closes last: when every other story of the intake is closed, its builder
records the commands that show the intent (`machinery:demo` lines) and a fresh verifier accepts
it. Then the intake item is closed; stage `intake` warns while a delivered one stays open.

**Land.** Stories of a wave are integrated; then the whole suite runs once at the head that is
ready, with `machinery landing --base <base>`, and at most two fix-forwards (section 12.2). A failed
landing marks the integration branch red: no story closes and no other head lands until a repair
(`--repair`) or a revert (`--clear-red`). Between plans the planner files the observations
(`machinery observations --file`) and reads the learnings (`machinery learnings`); decisions are
recorded with `machinery decide` and reach the briefs of the stories they govern.
`machinery probe` runs per batch and before a release.

A new project starts with a full-breadth skeleton: every action has an operation and a mounted
handler, every machine moves state through its generated table, and effects do nothing yet. At
that point `verify` passes with a long owed list. The work that remains is the owed list.

An existing project starts with `machinery assess` (section 18), then
`machinery adopt --rehearse` (section 17.3), then `machinery adopt` (section 17.2): the first owed list is whatever the first modeled area declares, and everything
outside the scope is left alone.

## 20. Probe

A receipt is a claim made by a test. `machinery probe` checks that the tests would notice if the
system were wrong, by giving the system a wrong table on purpose:

```
machinery probe [--ids <id>,...] [--prefix <id prefix>] [--max <n>] -- <command> [args...]
```

It relies on one convention: **when the environment variable `MACHINERY_TABLES_DIR` is set, the
system under test loads its transition tables from that directory instead of its usual place.**
The tests keep reading the committed tables. So during a probe run the system follows a mutated
table while the tests still expect the real one. A system that does not take its transitions from
the tables, or that ignores the variable, survives every mutation, and that is exactly what the
probe reports.

Precondition: evidence valid for the current commit (section 10.1) and a clean working tree, as
for `verify` without `--allow-dirty`. Otherwise the stage is
`not demonstrated: no evidence at <short sha>`, or `not demonstrated: working tree has uncommitted
changes: the probe needs the evidence of a clean tree`, and the command exits 1.

For every selected `tc` id of kind `transition` that is `proven` by the current evidence
(section 12), in id order, and for each mutation that applies to it:

| Mutation | Change made in the copy of the table | Applies when |
|---|---|---|
| `retarget` | The transition's `target` becomes another state of the same machine: the first state in document order that is neither its target nor its source. | The transition has a target and the machine has such a state. |
| `drop-effect` | The first name is removed from the transition's `actions` and from its `effects`. | The transition has at least one action. |

With no selector every proven transition case is probed; `--max <n>` keeps the first `n`
mutations. For each mutation, `machinery probe`:

1. copies the committed tables directory to a temporary directory and applies the one change
   there; nothing in the project is modified;
2. runs the command as `machinery run` does, with three variables: `MACHINERY_TABLES_DIR` (the
   temporary directory), `MACHINERY_PROBE_CASE` (the case id, so a project may run only the tests
   that matter) and `MACHINERY_RECEIPTS` (a temporary file outside the receipts directory);
3. judges the mutation from that run's receipts alone: it is **killed** when the case id is not
   `proven` in that run (a `fail` receipt, a `pass` that did not observe every effect, or no
   receipt for that id), and it **survived** when the case id is still `proven`. A run that wrote
   no receipt at all proves nothing either way: `ERROR  probe run produced no receipts: <id>
   <mutation>`, counted neither as killed nor as survived;
4. removes the temporary directory and file.

`<what was changed>` is `target <old> -> <new>` or `dropped <name>`. The command's output goes to
stderr.

Stage `probe` reports `ERROR  survived: <id> <mutation> (<what was changed>)` for every survivor
and `checked: <n> mutations, <n> killed, <n> survived`. Selecting nothing that can be mutated is
`not demonstrated: nothing to check`. The exit code follows section 3.3. Probe runs never count
as evidence: a probe run has no entry in the run ledger, and its receipts are judged as they are,
without the claim rule of section 10.2, which the evidence that selected the case has already
passed.

The probe is expensive: one test run per mutation. It is run per batch and at the end of an epic,
not per story, and it is never part of `machinery check` or `machinery verify`.

The probe is the measure of test quality in this method. There are no unit tests to count; a
suite is good when it kills the mutations of the behavior it claims.

## 21. Agent surfaces

Agents use machinery through its commands. The checks are tools, not prose: an agent is told
which command to run and reads its findings, and is never asked to remember a rule.

| Surface | Where | Holds |
|---|---|---|
| Skill | `skills/machinery/SKILL.md` | The entry point: which role a context is in and the command whose output says what to do next. At most 120 lines; details are in `docs/` and are opened only when needed. |
| Agent prompts | `agents/machinery-pm.md` (the planner), `agents/machinery-modeler.md`, `agents/machinery-builder.md`, `agents/machinery-verifier.md`, `agents/machinery-checker.md`, `agents/machinery-delegate.md` (experimental, section 15.9) | One role each, at most 60 lines: the role's goal, what it may and may not write, the gates the tool holds on it, and the rules that are judgment, each with the failure it prevents. |
| Roles | `docs/roles.md` | Who may write what, what the tool verifies about each role, when the checker and the delegate are called, and the judgment rules with the failure each prevents. At most 120 lines. |
| Plugin manifests | `.claude-plugin/plugin.json`, `.codex-plugin/plugin.json` | Name, version, and the paths above. |
| Hooks | `hooks/hooks.json` (the Claude Code plugin) | Three hook events, each running `machinery hook <event>` (section 21.1). |

The hooks put the checks in front of an agent at the points where it acts: the status when a
session starts, the static check before `git commit` and `git push`, and after a turn that changed
files. They are a convenience in front of the gates, not the gates: the gates fail closed where
they run, `make check`, `machinery landing` and CI. A hook keeps one small file per session and
never takes custody of the project's files.

### 21.1 Hooks: `machinery hook <event>`

```
machinery hook session-start | pre-tool | stop
```

One command a harness calls from its hooks. It reads the harness's hook input, one JSON object,
on stdin and writes the harness's hook output, one JSON object or nothing, on stdout. The logic
is in the tool; a plugin only wires it. The fields are those of the Claude Code hooks reference:

| Field | Read or written | Use |
|---|---|---|
| `session_id` | read, every event | the session (section 15.1) |
| `cwd` | read, every event | where the project is looked for, walking up; the process directory when absent; `-C` wins |
| `tool_name`, `tool_input.command` | read, `pre-tool` | the Bash command about to run |
| `stop_hook_active` | read, `stop` | the loop guard: true while the turn continues because of a stop hook |
| `CLAUDE_ENV_FILE` (environment) | appended to, `session-start` | `export MACHINERY_SESSION=<session_id>` for the session's later commands |
| `hookSpecificOutput.hookEventName`, `.additionalContext` | written | context for the model (`SessionStart`; `PreToolUse` in warn mode) |
| `hookSpecificOutput.permissionDecision` (`deny`), `.permissionDecisionReason` | written, `pre-tool` | refuses the command; the reason is shown to the model |
| `decision` (`block`), `reason` | written, `stop` | the turn goes on; the reason is shown to the model |
| `systemMessage` | written | a message shown to the user that changes nothing (`stop` in warn mode, an internal error) |

`machinery hook` always exits 0: a hook must never fail the harness, and for a harness exit 2
would block. Any internal error (stdin that is not hook JSON, an invalid `machinery.yaml`, a git
or backlog command that fails, an unknown event) prints
`{"systemMessage":"machinery hook <event> could not run: <why>"}` and changes nothing. A directory
that is not in a machinery project prints nothing. An owed file that does not parse
(section 11.1) or has a problem of its own is not an internal error: `pre-tool` and `stop` run no stage and hold
`owed: ERROR  <owed file>: <error>; ...` as their one finding, so a commit or push is refused in
block mode, and `session-start` prints the `could not run` message with it. Other harnesses call the same command with
JSON of the same shape.

Config in `machinery.yaml` (section 2), each `block`, `warn` or `off`:

```yaml
hooks:
  commit: block     # default block: git commit
  push: block       # default block: git push
  stop: warn        # default warn: the end of a turn
```

`block` refuses (`pre-tool`) or continues the turn (`stop`); `warn` hands the same text over
without stopping anything (`additionalContext` for `pre-tool`, `systemMessage` for `stop`, since
context on a stop continues the turn); `off` prints nothing. A value outside the three is a config
error (`hooks.commit must be block, warn or off, got "<value>"`).

**session-start.** Never blocks. In a project it prints
`{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"<text>"}}`, the text
being the line `machinery: <root directory name> is a machinery project; machinery status prints the full summary.`
and then, in their order there, the lines of `machinery status` (section 14) that start with `red:`,
`ids:`, `proofs awaiting:`, `intake ` and `questions:`. At most 30 lines: beyond that the first 29
and `<k> more line(s): run machinery status`. When `session_id` is one word of
`^[A-Za-z0-9][A-Za-z0-9._-]*$`, it also records the session's start in
`<git common dir>/machinery/sessions/<session_id>.json`,
`{"version": 1, "session": "<session_id>", "head": "<full sha of HEAD, empty before the first commit>"}`,
unless that file exists already (a resumed or compacted session keeps its first start), and, when
`CLAUDE_ENV_FILE` is set, appends `export MACHINERY_SESSION=<session_id>` to that file. The
session files stay; they are a few bytes each.

**pre-tool.** Without loading the project, it reads `tool_input.command` when `tool_name` is
`Bash` and finds the git commands in it: the command is split into simple commands at `;`, `&`,
`|`, newlines, parentheses and backquotes outside quotes; leading `NAME=value` words and one
`env`, `command`, `exec`, `nohup`, `time` or `sudo` are skipped; a simple command is a git
command when its first word is `git` (or ends in `/git`), and its subcommand is the first word
after git's own options (`-C`, `-c`, `--git-dir`, `--work-tree`, `--namespace` and
`--super-prefix` take the next word). Anything other than `commit` or `push` prints nothing; a
command line that holds both is a push. Then, by `hooks.commit` or `hooks.push`, it runs the
stages of `machinery check` and stage `ratchet` against the base: `backlog.base`, or `main`
without a backlog. When `backlog.tool` is set and the name of the current branch is the id of an
item of the backlog (`bd show` finds it), stage `scope` runs too, for that story (section 15.11):
a builder's worktree named after its story is held to its scope at every commit. Before the repository's first commit the ratchet is left out; when HEAD
exists and the base names no commit the ratchet is `not demonstrated: base <ref> is not a commit;
set backlog.base to the integration branch`. For a push, a red integration branch (section 12.2)
is a finding too:
`red: the integration branch is red since <short sha> (<stages>); marker <path>; repair: fix forward and run machinery landing --base <ref> --repair, or revert and run machinery landing --clear-red --reason "<what was reverted>"`,
`<path>` relative to the project root. The findings are, in the order the stages ran (the red line
first), each ERROR and DRIFT line of a stage that failed, `<stage>: <ERROR|DRIFT>  <message>`, and
each stage that was not demonstrated, `<stage>: not demonstrated: <reason>`: a stage that is not
demonstrated blocks like one that failed. Without findings it prints nothing. With findings the
text is

```
machinery: git <commit|push> refused: machinery check --base <ref> found <n> finding(s)
<the first 40 findings>
<k> more finding(s): run machinery check --base <ref>
```

(the last line only past 40), returned as `permissionDecision` `deny` with the text as
`permissionDecisionReason`; in warn mode the first line reads
`machinery: git <commit|push> goes ahead (hooks.<commit|push>: warn): machinery check --base <ref> found <n> finding(s)`
and the text is `additionalContext`.

**stop.** By `hooks.stop`, when files changed since the session started (`git status` of the
project is not empty, or HEAD is not the head of the session's file; without a session file only
`git status` counts), it runs the stages of `machinery check` and collects the findings as
`pre-tool` does. Without a change or without findings it prints nothing. With findings the text is

```
machinery: machinery check found <n> finding(s) in the files changed this session
<the first 40 findings>
<k> more finding(s): run machinery check
```

In block mode it prints `{"decision":"block","reason":"<text>"}`, so the turn goes on with the
findings in front of the model, except while `stop_hook_active` is true: then, as in warn mode, it
prints `{"systemMessage":"<text>"}`, so a stop chain is blocked at most once.

A surface names a command only as the tool has it: every `machinery <command>` in the skill and the
agent prompts exists, with the flags shown. A surface states a rule the tool enforces as a pointer
to the command that enforces it, and keeps its own sentences for what the tool cannot see.

Routing. Model classes follow the role first and the tier second. The design roles (planner,
modeler, checker, delegate) run on the strongest class whatever the tier: a wrong design or plan
is repeated by every story built on it. A builder runs on a standard class, and a verifier on a
class at least as strong as its builder's: a weaker one cannot disprove what the builder claims.
A brief and `machinery waves` print the text for the tier (`routing` in `machinery.yaml`, section
2), whose defaults say this per tier; a vital core is built on the strongest class. The texts name
classes, never a vendor's model, and the agent prompts carry no model id. A brief gives pointers
and commands; context is read from the files it names, in the worker's own session.

## 22. External checkers

A project can plug its own deterministic checker into `machinery check`: a static analyzer, a
logic solver, or a rule engine that encodes a standard, a control catalog or a regulation.
machinery never learns what the checker is. It defines one input file, one output file and where
the command that connects them is kept.

### 22.1 The parts

| Part | Who writes it | Where | Committed |
|---|---|---|---|
| The manifest | The project | `checkers:` in `machinery.yaml` | Yes |
| The projection | `machinery gen` | `<gen>/projection.json` | Yes |
| The evidence | The checker | `design/checkers/<id>.evidence.json` | Yes |
| The registry | Each person, per machine | `.machinery/checkers.local.yaml` | Never |

The split matters. The manifest, the projection and the evidence are neutral: they name a checker
only by an id the project chose, so a repository can be public while the tool behind a checker is
private. The registry, which holds the actual command, stays on the machine. A person without the
tool can still run `machinery check`: the stage is pure and reads only committed files.

A manifest entry:

```yaml
checkers:
  - id: rules                     # ^[a-z][a-z0-9-]+$, unique
    covers: ["act:*", "inv:*"]    # required: the design ids this checker claims to judge
    evidence: design/checkers/rules.evidence.json   # optional; default design/checkers/<id>.evidence.json
```

A `covers` entry is a full id or a prefix ending in `*`. It selects among the ids of the
projection: `act`, `inv`, `sc`, `tc` (every transition, ignore, refusal and guard-clause case of every
machine), `fm` and `entity:<Name>`.

### 22.2 Declared facts

Some things a checker needs are true of the system but are in no other design source: that an
action handles payment data, that the system makes automated decisions about people. The optional
facts file states them:

```yaml
version: 1
facts:
  - subject: "act:Order.pay"      # an id of the projection, or the word system
    predicate: handles-payment-data   # ^[a-z][a-z0-9-]+$
    value: true                   # a string, a number or a boolean
```

Unknown keys, an unknown subject, and two facts with the same subject and predicate are ERRORs in
stage `checkers`. machinery gives facts no meaning; it carries them to the checkers.

### 22.3 The projection

`projection.json` is the whole design as one deterministic JSON document. It is the only thing a
checker reads; a checker never reads the project's sources. Every list is sorted by id or name.

```json
{
  "version": 1,
  "entities": [
    {"id": "entity:Order", "name": "Order",
     "attributes": [{"name": "status", "type": "OrderStatus"}],
     "relationships": [{"entity": "Customer", "cardinality": "n:1", "ownership": "referenced", "role": ""}],
     "actions": [{"id": "act:Order.pay", "name": "pay", "actor": "Customer", "preserves": ["inv:paid-before-shipped"]}]}
  ],
  "enums": [{"name": "OrderStatus", "values": ["Pending", "Paid", "Shipped", "Cancelled"]}],
  "invariants": [{"id": "inv:paid-before-shipped", "entity": "Order", "statement": "..."}],
  "scenarios": [{"id": "sc:customer-pays-and-the-order-ships", "name": "...", "actors": ["Customer", "Clerk"], "invariants": ["inv:paid-before-shipped"]}],
  "machines": [
    {"id": "order", "tag": "ORDE", "role": "lifecycle", "lifecycle_of": "Order",
     "states": [{"path": "Pending", "kind": "atomic", "resting": true}],
     "transitions": [{"id": "tc:ORDE-2e883d", "source": "Pending", "kind": "on", "event": "pay", "guard": "hasPaymentMethod", "target": "paying", "effects": ["stampAttempt"]}],
     "ignores": [{"id": "tc:ORDE-2fa2af", "state": "Pending", "event": "ship"}],
     "refusals": [{"id": "tc:ORDE-f3dfea", "state": "Pending", "handler": "cancel"}],
     "clauses": [{"id": "tc:ORDE-1457d2", "transition": "tc:ORDE-2e883d", "guard": "hasPaymentMethod", "clause": "cardOnFile"}],
     "invokes": [{"state": "paying", "src": "chargePayment"}]}
  ],
  "operations": [{"contract": "contracts/openapi.yaml", "id": "payOrder", "method": "POST", "path": "/orders/{id}/pay", "actions": ["act:Order.pay"]}],
  "failure_modes": [{"id": "fm:payment-provider-down", "subject": "invoke:order.chargePayment", "required": "surface"}],
  "dependencies": [{"name": "postgres", "posture": "real"}],
  "facts": [{"subject": "act:Order.pay", "predicate": "handles-payment-data", "value": true}]
}
```

An invariant's `entity` is empty for a top-level invariant. A failure mode carries the values it
has, linked or not (section 5.4.1): an imported row no links entry names has an empty `subject`
and `required`, and a row that has no required value (neither its entry nor its cell gives one)
an empty `required`. A machine's `role` is `lifecycle` or
`operational`. A machine lists, after its `ignores`, its `refusals` and its guard `clauses` (section 7.2),
each with its `tc` id. Lists that have nothing are empty lists, never absent. The projection does not
contain `ck` obligations: it is the input of the checkers, not their output.

### 22.4 The evidence

A checker writes one JSON document:

```json
{
  "version": 1,
  "checker": "rules",
  "input_sha256": "<sha256 of the projection.json bytes it read>",
  "verdict": "pass",
  "covers": ["act:Order.pay", "inv:paid-before-shipped"],
  "findings": [
    {"severity": "error", "subject": "act:Order.pay", "rule": "R-12", "message": "what is wrong and what would fix it"}
  ],
  "obligations": [
    {"name": "operator-can-halt", "title": "An operator can halt order processing", "subject": "system"}
  ],
  "details": {}
}
```

- `checker` equals the manifest id.
- `input_sha256` binds the verdict to one exact design. Evidence for another projection is stale.
- `verdict` is `pass` or `fail`, and is `fail` exactly when there is a finding with severity
  `error`. `severity` is `error` or `warn`.
- `covers` lists the ids the checker actually judged.
- A finding's `subject` and an obligation's `subject` are ids of the projection, or `system`.
  `rule` is the checker's own name for the rule, free text.
- `obligations` is optional. Each `name` matches `^[a-z][a-z0-9-]+$` and is unique in the file.
- `details` is optional and free: a checker may keep its own proof or trace there. machinery
  ignores it.

Anything else in the document, or a document that breaks one of these rules, is invalid evidence.

### 22.5 Obligations from a checker

A checker can do more than judge the design. It can say what the running system must be shown to
do: "an operator can halt processing", "every automated decision is logged with its inputs". Each
entry of `obligations` becomes an obligation `ck:<checker id>:<name>` with the entry's title. From
there it is an obligation like any other: it appears in `obligations.json`, a test claims it with
`machinery:verifies`, a receipt proves it, and until then it is in `owed.yaml`. This is how a
requirement that comes from outside the design reaches the same proof loop as the rest.

### 22.6 The registry and `machinery checkers run`

`.machinery/checkers.local.yaml`, or the file named by `--registry` or by the environment variable
`MACHINERY_CHECKERS_REGISTRY`:

```yaml
version: 1
checkers:
  rules:
    command: ["sh", "tools/rules-adapter.sh", "{projection}", "{evidence}"]
    timeout_seconds: 120          # optional, default 300
```

`{projection}`, `{evidence}` and `{project}` are replaced by absolute paths. The command is run
directly, without a shell.

`machinery checkers run [<id>...]` (all configured checkers when no id is given), for each one:

1. refuses when `projection.json` is stale or missing: `ERROR  run machinery gen first`;
2. finds the command; no registry, or no entry for the id, is a config error (exit 2):
   `config: no command registered for checker <id> in <registry path>`;
3. runs it in the project directory with `{evidence}` pointing at a temporary file;
4. a non-zero exit, a timeout or no file written is `ERROR  checker <id> did not run: <reason>`;
   the committed evidence is left as it was;
5. validates the document (section 22.4), including that `input_sha256` is the hash of the
   projection it was given; invalid evidence is an ERROR and is not installed;
6. moves the document to the evidence path, and prints its findings and
   `checker <id>: <verdict>, <n> finding(s), <n> obligation(s), <n> covered`.

After the last checker it regenerates `obligations.json`, so new `ck` obligations appear at once.
Exit 1 when any checker has an ERROR or the verdict `fail`, else 0.

### 22.7 Stage `checkers`

Part of `machinery check`, after `formal`. It runs no tool and reads no registry. Not
applicable when neither `checkers:` nor `design.facts` is present.

| Finding | When |
|---|---|
| `ERROR  <facts path>: <problem>` | The facts file is invalid (section 22.2). |
| `ERROR  <id>: no evidence; run machinery checkers run <id>` | The evidence file does not exist. |
| `ERROR  <evidence path>: <problem>` | The evidence is invalid. |
| `DRIFT  <evidence path>: stale` | `input_sha256` is not the hash of the current `projection.json`. |
| `ERROR  <id>: <subject>: <message> [<rule>]` | A finding with severity `error`. |
| `warn   <id>: <subject>: <message> [<rule>]` | A finding with severity `warn`. |
| `ERROR  <id>: covers matches nothing: <pattern>` | A `covers` pattern selects no id. |
| `ERROR  <id>: not covered: <design id>` | An id selected by `covers` is missing from the evidence's `covers`. |

`checked: <n> checkers, <n> covered ids, <n> findings, <n> obligations, <n> facts`.

Stale evidence still has its findings reported, so a person sees both that it is stale and what
it said.

## 23. Risk tiers

Formality follows risk. The planner decides, per area of the design and before work starts, how
much formality the area gets, and writes it down as data:

```yaml
version: 1
areas:
  - name: order-payment          # ^[a-z][a-z0-9-]+$, unique
    tier: vital                  # consequential | critical | vital
    why: "money moves; a customer charged twice cannot be made whole by a retry"
    ids: ["act:Order.pay", "tc:ORDE-*", "inv:paid-before-shipped"]
    questions: [concurrency]     # required for critical and vital: ordering | concurrency | recovery | structure
    property:                    # vital only, required (section 23.1)
      - name: charged-once
        statement: "An order is charged at most once, whatever its callers do at the same time."
        kind: exactly-once       # invariant | pre-post | exactly-once | never
        over: ["act:Order.pay", "tc:ORDE-8bd6e7"]
    core:                        # vital only, required
      paths: ["internal/domain/**"]
      entry: ["internal/domain/order.go#Apply"]
    assumes:                     # vital only, required
      - name: provider-charges-once
        statement: "The payment provider charges a payment at most once, however often it is asked."
        evidence: ["fm:payment-provider-down"]
  - name: order-ownership
    tier: critical
    why: "an order belongs to exactly one customer, whatever the data looks like"
    ids: ["act:Order.place"]
    questions: [structure]
  - name: shipping-permission
    tier: consequential
    why: "authorization: only a Clerk ships"
    ids: ["act:Order.ship"]
not_vital:                       # optional: dismissed candidates (section 23.3)
  - {id: "tc:ORDE-70d94f", reason: "a refused pay records the refusal and moves no money"}
```

An id in no area is **routine**. An id in several areas has the highest of their tiers, in the
order routine, consequential, critical, vital.

| Tier | Use it for | What it adds |
|---|---|---|
| routine | Work where a wrong result is cheap to see and to undo. | The machine, its generated transition cases, end-to-end evidence. |
| consequential | Published contract semantics, authorization, tenant isolation, money or audit integrity, data migrations. | An abuse case for every action in the area (section 6.6), and acceptance by a fresh verifier who predicts a disproving observation before reading the proof (section 15.6). |
| critical | Promises whose break harms users and that tests rarely reach: races, lost or repeated work, structural rules that must hold for every shape of data. | A formal model with the checks of section 16. The area yields the obligation `fv:<name>`. |
| vital | Paths that must never go wrong: a customer charged twice, an audit record lost, a signature attributed to the wrong person. | Everything critical adds; the exact properties the code must keep, the core that holds them, shaped for a proof (section 9), and what that core assumes, each assumption with its evidence (section 23.1). Each property yields the proof obligation `pf:<area>:<name>` (section 23.2) and the area its property record (section 23.4). |

**What vital is for, and what it is not.** A formal model (section 16) checks the design; it does
not check the code that runs. On most paths that gap is acceptable: the tests and the model
together catch what matters. A vital area names the few paths where it is not, and makes them
explicit, countable and shaped for proof: each property is stated exactly, over the design ids it
holds for; the code that must keep it is confined to a small core; what the core takes on trust
is written down with its evidence. This version proves nothing more about a vital area than about
a critical one. A later version will require a machine-checked proof of each property; until
then every report that calls an id of a vital property proven says what kind of proof it has:
`(by test and model; proof awaited)`.

Stage `risk`:

| Rule | Check |
|---|---|
| `area-shape` | Unique names, a valid tier, a non-empty `why`, at least one id pattern. |
| `id-resolves` | Every pattern matches at least one `act`, `inv`, `sc`, `tc` or `fm` id. |
| `critical-question` | A critical or vital area lists at least one question; another area lists none. |
| `vital-shape` | A vital area has `property`, `core` and `assumes`, each well formed; another area has none of them (section 23.1). |
| `vital-refs` | Every id a property is over, and every evidence id of an assumption, is an id of the design of the right kind (section 23.1). |
| `not-vital` | Every dismissed candidate names ids of the design that are in no vital area, with a reason (section 23.3). |
| `status-file` | The status file (section 4.1) is well formed and every entry names ids of the design. |

Stage `risk` holds both files the planner writes about ids: tiers and statuses. `checked:` counts
areas, consequential ids and critical ids when `design.risk` is set, and, when an area is vital,
vital ids, properties and assumptions; then the dismissals when `not_vital` has an
entry; then, when the status file exists, the proposed ids and the retired ids it gives. Not
applicable when `design.risk` is absent and there is no status file
(`not applicable: no design.risk configured`). Not demonstrated when it has no area and no status
entry.

Evidence follows the tier too: a pass written by a run adapter proves an id of a routine or
consequential area and, unless `evidence.adapter.tiers` says otherwise, never one of a critical
or vital area (section 10.3).

The tier is chosen at planning, not discovered at review. When it is unclear which tier applies,
the higher one does. Lowering a tier is a deliberate, recorded act: the ratchet (section 11.3)
fails on a lowered tier unless the risk file says so:

```yaml
lowered:
  - {ids: ["act:Order.pay", "tc:ORDE-*"], from: critical, reason: "card payments moved to the hosted page"}
```

`ids`, `from` (the tier the ids had: `consequential`, `critical` or `vital`) and `reason` are
required. The entry stays in the file as the record of the decision. Lowering a vital area to
critical is a lowering like any other.

What a vital area states can be weakened while its tier stays (section 11.3, rule
`vital-weakened`). Such a change is recorded with the second form of entry, the area and the
exact `<what>` of the ratchet finding:

```yaml
lowered:
  - {area: order-payment, what: "property charged-once: over tc:ORDE-8bd6e7 removed", reason: "the retry path moved to the provider"}
```

An entry has `ids` and `from`, or `area` and `what`, never both; `reason` is required in both
forms. Each problem is an ERROR of stage `risk`: `<risk file>: lowered entry <n>: an entry has ids
and from, or area and what, not both`, `<risk file>: lowered entry <n>: area must match
^[a-z][a-z0-9-]+$`, `<risk file>: lowered entry <n>: what is required`, and the messages of the
first form (`ids needs at least one id pattern`, `from must be consequential, critical or vital`,
`reason is required`).

### 23.1 Vital areas

A vital area has every key of a critical area and three more, all required:

- `property`: one or more entries, each an exact thing that must hold, stated over inputs and
  stored state:
  - `name` matches `^[a-z][a-z0-9-]+$` and is unique in the area;
  - `statement` is the property in the system's words;
  - `kind` is `invariant` (it holds in every reachable state), `pre-post` (an operation that
    starts where its precondition holds ends where its postcondition holds), `exactly-once` (an
    effect happens once per request, however often the request is repeated) or `never` (no
    behavior reaches the state or the effect);
  - `over` lists one or more full ids (no `*`) of kind `inv`, `act` or `tc` that yield an
    obligation and that the area's `ids` match: the design ids the property is a statement about.
- `core`: where the code of the path lives. `paths` is one or more globs; `entry` is one or
  more entry points, each `<file>#<symbol>`. Stage `arch` holds the files to the shape of
  section 9.
- `assumes`: one or more entries, each something the core relies on from its surroundings (the
  database enforces a unique key; the caller is authenticated): `name` (`^[a-z][a-z0-9-]+$`,
  unique in the area), `statement`, and `evidence`, one or more obligation ids. An assumption is
  only as good as its evidence: while one of its evidence ids is in the owed list it is
  **assumed, unproven**.

Each problem is an ERROR of stage `risk`, `<risk file>: <area>: <message>`:

- `a vital area states at least one property`; `a vital area names its core (core.paths and core.entry)`;
  `a vital area states what its core assumes (at least one entry under assumes)`;
- `only a vital area has property, core and assumes`;
- `property <n>: name must match ^[a-z][a-z0-9-]+$`, `property <name>: duplicate name`,
  `property <name>: statement is required`,
  `property <name>: kind must be invariant, pre-post, exactly-once or never, got "<kind>"`,
  `property <name>: over needs at least one id`;
- `property <name>: over <id>: a property is over inv, act or tc ids`,
  `property <name>: over <id>: name an exact id, not a pattern`,
  `property <name>: over <id> is not an id of the design`,
  `property <name>: over <id> is proposed: it yields no obligation` (`retired`),
  `property <name>: over <id> is not in the area; add it to ids, or name an id the area holds`;
- `core.paths needs at least one glob`, `core.paths: "<glob>" is not a valid glob`,
  `core.entry needs at least one <file>#<symbol>`, `core.entry: "<entry>" is not <file>#<symbol>`;
- `assumes <n>: name must match ^[a-z][a-z0-9-]+$`, `assumes <name>: duplicate name`,
  `assumes <name>: statement is required`, `assumes <name>: evidence needs at least one id`,
  `assumes <name>: evidence <id> is not an obligation of the design`.

### 23.2 Proof obligations

Each property yields the proof obligation `pf:<area>:<name>`, with the tier of its area. In this
version nothing can prove one, and machinery says so wherever the evidence of a vital area is
reported:

- A `pf` id is **awaiting proof**, a status of its own: never owed and never proven. It is not in
  `obligations.json` (it is in the property record, section 23.4), `owed sync` neither adds nor
  removes it, and an owed entry that names one is an ERROR (section 11.1).
- No test discharges it: a tag or marker that names one is a trace ERROR (section 8), a receipt
  that names one a verify ERROR (section 12), and a story whose ids name one a story ERROR
  (section 15.6).
- `status`, `verify` and the design section of `assess` count the proof obligations on their own
  line, `proofs awaiting: <n>` (sections 14, 12, 18.3), and a proof block lists those of the
  selected areas as `<pf id> (awaiting proof)` under `not proven` (section 13).
- An id a property is over is proven as any id is (section 12): by its tests, and for the area by
  its formal model. Wherever such an id is reported as proven (status, verify, the proof block,
  the story gate, assess), the line ends with `(by test and model; proof awaited)`.
- An assumption with an owed evidence id prints as `assumed, unproven: <area> <name>: <statement>`
  in status, verify, the proof block and assess, fails the story gate (section 15.6), and is
  `assumed, unproven` in the property record.

The `fv` obligation of a vital area, and the `inv`, `tc` and `act` ids its properties are over,
behave exactly as they do in a critical area.

### 23.3 Candidates: `machinery vital candidates`

So that nobody has to remember to look, `machinery vital candidates` lists the design ids that
look as if they belong in a vital area and are in none, from fixed rules over the design alone:
it reads no code. An id is in a vital area when the `ids` of a vital area match it. Only ids that
yield an obligation (section 4.1) are candidates. The rules, in this order:

| Rule | Candidate | Reason |
|---|---|---|
| action | An `act` id whose action name matches a word of `vital.names`. | `action <action> matches "<word>"` |
| entity | An `act` id whose entity name matches a word of `vital.names`. | `entity <Entity> matches "<word>"` |
| event | A `tc` id of a transition of kind `on` whose event matches a word of `vital.names`. | `event <event> matches "<word>"` |
| event entity | A `tc` id of a transition of kind `on` of a lifecycle machine whose entity name matches a word of `vital.names`. | `entity <Entity> matches "<word>"` |
| failure | An `fm` id whose `failure` or `behavior` text matches a word of `vital.failure`. | `failure mode text matches "<word>"` |
| effect | A `tc` id of a transition, of any kind, one of whose effects matches a word of `vital.effects`. | `effect <effect> matches "<word>"` |

The default lists:

- `vital.names`: pay, charge, refund, transfer, debit, credit, sign, approve, destroy, delete,
  erase, grant, revoke, seal, audit, ledger;
- `vital.failure`: lost, twice, duplicate, double, replay;
- `vital.effects`: charge, debit, credit, emit, send, publish.

A name or a text matches a word when one of its words, lower-cased, starts with it. Words are
split at every character that is not a letter or a digit, between a lower-case letter or a digit
and an upper-case letter, and before the last upper-case letter of a run that a lower-case letter
follows: `recordCharge` is `record`, `charge`; `HTTPServer` is `http`, `server`; `pay_out` is
`pay`, `out`. So `pay` matches `Payment` and not `display`. The reason names the first word of
the list that matches; for an effect, the first effect of the row that matches one.

Each candidate prints once, with the reason of the first rule that matched it, sorted by id in
byte order, then the summary line:

```
fm:charge-replayed  failure mode text matches "twice"
tc:ORDE-507e4f  effect refundPayment matches "refund"
vital candidates: 2, dismissed: 1
```

A candidate is dismissed by listing it in the risk file under `not_vital`, with the reason, so
that the list only shrinks and every dismissal stays on record. `id` is a full id or a pattern
(section 4); `dismissed` counts the candidates its entries match. Stage `risk` holds the entries,
`<risk file>: <message>`: `not_vital entry <n>: id is required`,
`not_vital: <id>: reason is required`, `not_vital: <id> is listed twice`,
`not_vital: <id> matches no act, inv, sc, tc or fm id` and
`not_vital: <id> is in vital area <area>; a dismissed id is not vital`.

The command needs a `machinery.yaml`, takes no arguments, writes nothing and exits 0 (2 on a usage
or config error). The design section of `machinery assess` prints the same list (section 18.3).

### 23.4 The property record: `<gen>/vital/<area>.json`

`machinery gen` writes one file per vital area, the hand-off a prover integration will read, and
`gen --check` holds it like every generated file (section 7):

```json
{
  "version": 1,
  "area": "order-payment",
  "why": "money moves; a customer charged twice cannot be made whole by a retry",
  "source": "design/risk.yaml",
  "properties": [
    {"id": "pf:order-payment:charged-once", "name": "charged-once", "kind": "exactly-once", "statement": "An order is charged at most once, whatever its callers do at the same time.", "over": ["act:Order.pay", "tc:ORDE-8bd6e7"], "status": "awaiting proof"}
  ],
  "core": {
    "paths": ["internal/domain/**"],
    "entry": ["internal/domain/order.go#Apply"],
    "files": [{"path": "internal/domain/order.go", "sha256": "<sha256 of the file bytes>"}]
  },
  "assumes": [
    {"name": "provider-charges-once", "statement": "The payment provider charges a payment at most once, however often it is asked.", "evidence": ["fm:payment-provider-down"], "status": "assumed, unproven"}
  ],
  "over": [
    {"id": "act:Order.pay", "status": "proven (by test and model; proof awaited)"},
    {"id": "tc:ORDE-8bd6e7", "status": "owed"}
  ]
}
```

- `properties` and `assumes` keep the order of the risk file, and so do the ids of a property's
  `over` and of an assumption's `evidence`. `core.paths` and `core.entry` are as written;
  `core.files` are the files `core.paths` matches, sorted, each with the SHA-256 of its bytes. The
  top-level `over` lists every id the properties are over, once, sorted.
- Statuses come from committed files only, so that the record is the same for everyone at one
  commit: an id is `owed` while the owed list holds it and `proven (by test and model; proof
  awaited)` otherwise (its status of section 4.1, with the wording of section 23.2); an
  assumption is `assumed, unproven` while one of its evidence ids is owed and `assumed,
  evidenced` otherwise; a property is always `awaiting proof`. Whether the evidence at a commit
  in fact proves an id is the judgment of `verify`.
- The records render the owed file: after `machinery owed sync`, run `machinery gen`. They are
  exempt with the owed file from the dirty rule and the validity rule of evidence (sections 10
  and 10.1), so committing them does not invalidate the run that justified the change.
- A file under `<gen>/vital/` whose area is not vital is
  `ERROR  <path>: unexpected file in gen directory`, as in section 7.
