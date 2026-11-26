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
  failure_modes:                # optional: a failure-mode catalog kept as Markdown tables (section 5.4.1)
    file: docs/risks.md         # required
    id: "^RISK-[0-9]+$"         # required: the first cell of a failure-mode row matches it in full
    columns:                    # required: header names, case-insensitive, first match wins
      failure: ["Failure", "What goes wrong"]
      behavior: ["Required behavior", "Expected"]
    links: design/risk-links.yaml   # optional: the subject and required value of each row
contracts:                      # optional
  - kind: openapi               # openapi | proto
    path: contracts/openapi.yaml
checkers:                       # optional: external checkers (section 22)
  - id: rules
    covers: ["act:*", "inv:*"]
dependencies:                   # optional: external dependencies of the system
  - name: postgres
    posture: real               # real | sandbox
trace:                          # optional
  code: ["internal/**/*.go"]    # files that may carry machinery:serves tags; default: scope
  tests: ["e2e/**"]             # files that may carry machinery:verifies tags
  markers:                      # optional: the project's own markers, literal prefixes (section 8.1)
    serves: ["Implements:"]
    verifies: ["Checks:", "Proves:"]
    scenarios: ["Scenario:"]
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
commands:
  e2e: ["go", "test", "./e2e/...", "-count=1"]   # optional: the project's end-to-end command
backlog:                        # optional: the backlog bridge (section 15)
  tool: bd                      # only bd
  base: main                    # optional: the branch work is cut from; default main
routing:                        # optional: model routing guidance printed in briefs (section 15.5)
  routine: "small model"
  consequential: "standard model; fresh verifier"
  critical: "strongest model for the formal model; standard model for code; fresh verifier"
  vital: "strongest model for the formal model and the core; standard model elsewhere; fresh verifier"
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
  `import.failure_modes` needs `file` (`import.failure_modes.file is required`), an `id` that is
  a regular expression (`import.failure_modes.id is required`,
  `import.failure_modes.id is not a regular expression: <error>`) and a non-empty list for each of
  `columns.failure` and `columns.behavior` (`import.failure_modes.columns.failure is required`).
  It may be set beside `design.failure_modes`.
- An empty marker in `trace.markers` is a config error
  (`trace.markers.verifies: a marker must not be empty`).
- `constraints.cores` needs `max_file_lines` and `max_func_lines`, each a positive number
  (`constraints.cores.max_file_lines is required and must be positive`), and an empty entry of
  `deny_imports` is a config error (`constraints.cores.deny_imports: an entry must not be empty`).
- An empty word in `vital.names`, `vital.failure` or `vital.effects` is a config error
  (`vital.names: a word must not be empty`). A list that is set replaces its default; words are
  compared lower-cased.
- `routing` keys are tier names: `routine`, `consequential`, `critical` and `vital`.
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
- The receipts directory is never committed. Everything under `design/` is committed.
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
| `machinery trace` | Stage `trace` (section 8). |
| `machinery check` | Every stage of `lint`, then `gen-drift`, `formal`, `checkers`, `arch`, `tests`, `trace`, `contract`, `intake`. With `--base <git-ref>` also stage `ratchet` (section 11.3). |
| `machinery checks` | Prints the declaration of applicable checks (section 3.4). A report, never a gate. |
| `machinery run -- <command...>` | Runs a test command and records a receipts file (section 10). |
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
| `machinery observations` | Lists the observations of stories that are not filed; `--file` files them (section 15.8). |
| `machinery learnings` | Prints the learnings of closed stories, by intake (section 15.8). `--since <ref>`. |
| `machinery checkers run [<id>...]` | Runs the external checkers registered on this machine and records their evidence (section 22). |
| `machinery checkers list` | Lists the configured checkers, whether each is registered locally, and the state of its evidence. |
| `machinery init` | Scaffolds a new project (section 17.1). |
| `machinery adopt` | Brings an existing project under machinery without touching its code (section 17.2). |
| `machinery assess` | Reports, read-only, where an existing project stands against its own design (section 18). |
| `machinery vital candidates` | Lists the design ids that look as if they belong in a vital area and are in none, with the reason (section 23.3). A report, never a gate. |
| `machinery config` | Prints the effective configuration as YAML, defaults applied. |
| `machinery version` | Prints `machinery <version>`. |

`lint` and `check` accept `--stage <name>` (repeatable): only the named stages run, in their usual
order, and the exit code depends on them alone. An unknown stage name is a usage error (exit 2).

`machinery` with no arguments prints help and exits 0. `version`, `init`, `adopt`,
`formal tools` and help work without a `machinery.yaml`, and `assess` with `--config`; every
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
7. **Tag.** `_oracle_tag` is added when the result has none: the first four letters or digits of
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

The keys a binding adds, then an added `_oracle_tag`, come last in the root; every other key
keeps its place. A key that is still unknown after renaming is a finding of stage `machines`, as
in a native file: nothing is dropped silently. The runtime table of an imported machine names the source file in `source`, and its
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

`id`, `subject`, `failure`, `required` and `behavior` are required. Unknown keys are an ERROR.

Subject forms: `invoke:<machine id>.<src>`, `dependency:<name>`, `act:<Entity>.<action>`,
`entity:<Name>`, `state:<machine id>.<State>` (a failure state, a nested one by its dotted path)
and `event:<machine id>.<event>` (a failure event). `required: refuse` is only valid with
`kind: abuse`, and `kind: abuse` needs an `act` or `entity` subject.

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
- **Ids.** The machinery id is the cell lower-cased (`RISK-12` is `risk-12`, obligation
  `fm:risk-12`). It must match the pattern of section 5.4:
  `ERROR  <file>:<line>: <cell>: the id lower-cased, <id>, must match ^[a-z][a-z0-9-]+$`. A
  duplicate is `ERROR  <file>:<line>: <cell>: duplicate id, first on line <n>`, and the later row
  is not read. An id that `design.failure_modes` also declares is
  `ERROR  <file>:<line>: <cell>: fm:<id> is also declared in <native file>`, and the row is not
  read.
- **Text.** The failure cell is the failure mode's `failure`, and so the obligation title; the
  behavior cell is its `behavior`. Both are kept as written, trimmed.

An imported row has no subject and no required value until it is linked. The links file gives
them:

```yaml
version: 1
links:
  - {id: RISK-3, subject: "invoke:loan_desk.reserveCopy", required: retry-bounded}
  - {id: RISK-7, kind: abuse, subject: "act:Loan.renew", required: refuse}
```

`id` is the catalog id, compared case-insensitively. `subject` and `required` are required and
`kind` is optional (default `fault`), with the value rules of section 5.4 and its messages,
prefixed `<links file>: <id>`. An id that is not an imported row is
`ERROR  <links file>: <id>: no failure-mode row of <file> has this id`, one listed twice is
`ERROR  <links file>: <id>: listed twice`, and an unknown key is an ERROR.

- A **linked** row is a failure mode like a native one, and takes part in every rule of stage
  `xref-failure`.
- An **unlinked** row is an obligation all the same: `fm:<id>` is owed, claimed and traced like
  any failure mode. It names nothing, so it takes part in no subject rule of `xref-failure`: it
  neither resolves nor covers an invoke, a dependency, an action, a failure state or a failure
  event.

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
`_entries`, `_terminal`, `_external_events`, `_failure_events`, `_serves` (a non-empty list of
distinct `entity:<Name>` and `act:<Entity>.<action>` ids; whether they exist is section 6.4) and
`_aspect` (a short name matching `^[A-Za-z][A-Za-z0-9_-]*$`; which machines may declare it is
section 6.4). Two machines whose id tags
collide (section 7.2), or that share a machine `id`, are an ERROR.

Name hints are warnings and never fail the stage. A state whose name contains, case-insensitively,
`fail`, `error`, `abort`, `crash` or `timeout` and that is not tagged `failure` prints
`warn   <file>: state <S> looks like a failure state and is not tagged failure` (`<S>` is the
dotted path). An event the machine handles or ignores whose name contains one of those words, and
that is neither in `_failure_events` nor an action of the machine's lifecycle entity, prints
`warn   <file>: event <e> looks like a failure event and is not in _failure_events`.

`checked:` counts machines, states and transitions. Not applicable when neither `design.machines`
nor `import.machines` is configured. With `import.machines` the translation findings of section
5.2.1 are findings of this stage, and so are the rules of its register (`register-undeclared`,
`register-unobserved`), whose entries and observed transitions `checked:` then counts.

### 6.3 `failure-modes`

Parses the file against section 5.4, and the catalog and its links file against section 5.4.1.
Unique ids, valid `kind` and `required` values, well-formed `subject`. `checked:` counts failure
modes, native and imported, and, of those, abuse cases. When `import.failure_modes` is configured
it also counts the imported rows and, of those, the unlinked ones:
`checked: 9 failure modes, 1 abuse cases, 7 imported, 4 unlinked`. Not applicable when neither
`design.failure_modes` nor `import.failure_modes` is configured.

### 6.4 `xref-model-machine`

For every machine, and every entity of the models:

| Rule | Check |
|---|---|
| `machine-binding` | The machine binds to the model (section 5.2): it is the lifecycle machine of an existing entity, or an operational machine whose `_serves` names only entities and actions of the model. An entity has one lifecycle machine, or several that each declare a different `_aspect`. |
| `state-enum` | Lifecycle machine of entity E: let D be its domain states. Among the enums used as the `type` of an attribute of E, exactly one must have a value-name set equal to D. Otherwise ERROR, naming the missing and extra names against the candidate enum with the largest overlap (ties broken by enum name). When no candidate has a value in D, the finding says that E has no state enum for the machine and names no enum. An entity with no enum-typed attribute at all is one finding for the entity, whatever the number of its lifecycle machines. With aspects, each aspect machine is matched against an enum of its own: an enum that exactly matches one machine is that machine's, and no enum is the state enum of two machines (a machine that matches only an enum another one holds is a finding, and the enums other aspects hold are left out of its candidates). |
| `event-action` | Lifecycle machine of E: every event name under any `on` must be the name of an action of E, or be listed in the root `_external_events` or `_failure_events` (a failure event arrives from outside). When E declares no action, the events that are not listed are one finding for the entity, not one per event: the one missing declaration causes them all. Operational machine that binds to the model (no `machine-binding` finding): every event name under any `on` must be the name of an action of an entity it serves, the action of an `act:` id it serves, or be listed in `_external_events` or `_failure_events`. When it serves only entities that declare no action, the events that are not listed are one finding for the machine: nothing was compared with the model. |
| `action-event` | Every action of an entity that has a lifecycle machine must appear as an `on` event somewhere in that machine, or, with aspects, in at least one of them. |
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
| `subject-resolves` | Every failure-mode subject names an existing invoke, a configured dependency, a model action, a model entity, a state of the named machine that is tagged `failure`, or an event in the named machine's `_failure_events`. |
| `invoke-failure-mode` | Every `invoke` `src` of every machine is the subject of at least one failure mode. |
| `dependency-failure-mode` | Every configured dependency is the subject of at least one failure mode. |
| `tiered-action-abuse` | Every action that belongs to a consequential, critical or vital area (section 23) is the subject of at least one failure mode of kind `abuse`. |
| `failure-state-mode` | Every state tagged `failure` of every machine is the subject (`state:<machine id>.<State>`) of at least one failure mode. |
| `failure-event-mode` | Every `_failure_events` entry of every machine is the subject (`event:<machine id>.<event>`) of at least one failure mode. |

The stage applies when `design.failure_modes` or `import.failure_modes` is configured, or when
any machine declares a failure state or a failure event (section 5.2). An imported row that is
not linked (section 5.4.1) is counted among the failure modes and takes part in no rule. A
finding that asks for a failure mode says where to declare it: in the file of
`design.failure_modes` when there is one, else in the links file of `import.failure_modes`
(`a links file (import.failure_modes.links)` when none is configured). `checked:` counts
failure modes, invokes,
dependencies, tiered actions, failure states and failure events. With no failure-mode file
configured only `failure-state-mode` and `failure-event-mode` run: each failure state and failure
event is a finding that says to configure `design.failure_modes`, and `checked:` counts failure
states and failure events only. Not applicable when there is neither a failure-mode file nor a
failure state or failure event.

When there are failure modes and none of them names a subject (every one is an imported row that
is not linked, section 5.4.1), the stage compared no failure mode with the design, and its result
is `not demonstrated: no failure mode names a subject` (`nothing to check` when there is nothing
at all to compare), never passed, waivers or not; with findings it is failed (section 3.2). A
waiver does not link a row.

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
`machines` or `failure-modes` stage has an ERROR.

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

A proposed or retired id counts as an id of the design for rule `unknown-id`. A `*` token claims
only the ids among its matches that yield an obligation, so a pattern that also matches a
proposed or retired id is not a finding. A `pf:` token is matched against the proof obligations
alone, and a pattern among them (`pf:*`) is a claim of each one it matches; a bare marker token
is never read as a `pf` id. `unserved` and `unverified` ask only for ids that yield
an obligation, and the owed list is checked as in section 4.1.

`checked:` counts code files, test files, `act` ids and obligations. Only files inside `scope`
are read. Zero files matched by a configured glob list is an ERROR. Not applicable when `trace:`
is absent.

Trace is static and cheap. It proves that a test claims an id, not that the test passed. Only
`verify` proves that.

### 8.1 A project's own markers

A project that already points its code and tests at its design in its own words names those
markers in `trace.markers`, as literal prefixes. `machinery:serves` and `machinery:verifies` keep
working beside them, unchanged. A marker claims what the tag of the same kind claims, wherever
tags are read: stage `trace`, `machinery affected` (section 12.1) and the served ids of
`machinery adopt` (section 17.2).

- **Where.** A `serves` marker is read in the files of `trace.code` inside `scope`, a `verifies`
  or `scenarios` marker in the files of `trace.tests`. A marker counts only where it follows
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

The rules and the `checked:` line of stage `trace` do not change.

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

`machinery run`:

1. reads the current commit (`git rev-parse HEAD`) and whether the tree is dirty
   (`git status --porcelain` not empty, ignoring the owed file, the receipts files (the
   `.jsonl` files directly in the receipts directory, and nothing else in it) and the property
   records `<gen>/vital/*.json`, whose statuses render the owed file (section 23.4));
2. deletes receipt files in the receipts directory that are not valid for the current commit
   (section 10.1);
3. creates `<receipts dir>/<UTC timestamp>-<short sha>.jsonl` (timestamp
   `20060102T150405.000Z`) and writes the header line;
4. runs the command with extra environment variables: `MACHINERY_RECEIPTS` (absolute path of
   the file), `MACHINERY_COMMIT` (the full commit sha) and, for a targeted run (section 12.1),
   `MACHINERY_ONLY` (the ids to exercise, comma separated);
5. appends the end line when the command exits;
6. exits with the command's exit code, after printing `machinery run: <file> (exit <n>)` on
   stderr. A command that cannot be started exits 127 and leaves no file; a command killed by a
   signal exits 128 plus the signal number.

The command runs in the project root. The variables machinery sets for a run
(`MACHINERY_RECEIPTS`, `MACHINERY_COMMIT`, `MACHINERY_ONLY`, `MACHINERY_TABLES_DIR`,
`MACHINERY_PROBE_CASE`) are never inherited from the caller.

The file is JSON Lines. Three line types:

```json
{"v":1,"type":"run","commit":"<40 hex>","dirty":false,"command":["go","test","./e2e/..."],"stack":"postgres 17 in docker","started":"2026-01-02T03:04:05Z"}
{"v":1,"type":"receipt","id":"tc:ORDE-2e883d","outcome":"pass","test":"TestOrder/ORDE-2e883d","effects":["stampAttempt"],"observed":"Pending -> paying; attempt row written"}
{"v":1,"type":"end","exit":0,"finished":"2026-01-02T03:04:09Z"}
```

A test emits one `receipt` line per obligation it checked, by appending to the file named by
`MACHINERY_RECEIPTS`:

- `id`: an obligation id.
- `outcome`: `pass`, `fail` or `skip`.
- `test`: the name of the test that made the observation.
- `effects`: for `tc` ids, the effects the test actually observed. Omitted or empty means none.
- `observed`: one short sentence, at most 300 characters, saying what was seen.
- `replay`: for `cx` ids only, what the real system did when the counterexample was replayed:
  `reproduced` or `refuted` (section 16.10).

A test writes each line with a single append so that parallel tests do not interleave. When
`MACHINERY_RECEIPTS` is not set, the test emits nothing and still runs. When `MACHINERY_ONLY` is
set, a suite may run only the tests for those ids; a suite that ignores it is still correct.
The Go helper `pkg/receipt` implements all of this for Go suites; other languages append the
JSON line themselves.

A receipt is a claim made by a test. A test asserts first and emits afterwards: it emits `pass`
only after its assertions held.

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

The owed list is the single ratchet of the method. In a new project nearly everything is owed
right after the skeleton; each story proves some ids and removes them. In an existing project the
owed list is the baseline of what was never proven.

### 11.2 `machinery owed sync`

Reads the evidence exactly as `verify` does (section 12), then rewrites `owed.yaml`:

- removes every entry whose id is proven;
- unless `--prune-only`, adds every obligation that is neither proven nor listed, with the reason
  `unproven at <short sha>`;
- keeps existing reasons and story fields.

It prints `owed: <n> kept, <n> added, <n> removed` and exits 0. It refuses with
`ERROR  no complete run valid for <short sha>: run the tests through machinery run first` (exit 1)
when there is no complete run valid for the current commit, and when the owed file has problems.
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
(section 10.1). When the working tree has uncommitted changes outside the owed file, the
receipts directory and the property records, the evidence does not describe what is on disk: without `--allow-dirty` that
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

Findings:

- `ERROR  malformed receipts file: <file>: line <n>: <what>`. A file whose first line is not a
  run header is not a receipts file and is ignored.
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
- `ERROR  stale owed entry: <id> is proven` for an owed id that is proven. Run
  `machinery owed sync --prune-only`.
- `ERROR  unknown owed id: <id>`.

When no complete run exists for the current commit the stage is
`not demonstrated: no evidence at <short sha>`.

`checked:` counts obligations, proven, partial, owed and receipts.

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
ignored except for `failed` ones, which always print.

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
result:   <p> passed / <f> failed / <s> skipped, against <stack>
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
`<id> (<state>, owed)`.

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
proven: <n>  partial: <n>  failed: <n>  owed: <n>  unproven and not owed: <n>
proofs awaiting: <n>
vital: <id> proven (by test and model; proof awaited)
assumed, unproven: <area> <name>: <statement>
owed without a story: <n>
evidence: <short sha> <clean|dirty>, <n> run(s), <n> receipt(s)
```

The `red` line prints only while the integration branch is red (section 12.2). The first count
line lists only kinds that occur, in the order `tc, sc, inv, fm, fv, cx, ck`. The `ids` line
counts every `act` and obligation id by its status (section 4.1): `owed` the ids the owed list
holds, `proven` the other ids that yield an obligation, which is what `verify` holds them to; the
next line says what the evidence shows.
The `tiers` line counts obligations by the tier of their area and is omitted when there is no
risk file; its `vital` part, and the three lines after the `proven` line, print only when the
risk file has a vital area (section 23.2): the number of proof obligations, one `vital` line per
id a property is over that the evidence proves, and one `assumed, unproven` line per assumption
whose evidence is owed, each group sorted. When there is no evidence the line is `evidence: none at <short sha>`. When
`backlog.tool` is set and the backlog can be read, one line follows per open intake item, sorted by
id: the summary line of `machinery intake status` (section 15.7),
`intake <id>: <n> of 7 steps done; next: <step>`. Exit 0 unless the config is invalid.

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
machinery:proof <full sha> <actor> [session=<session>]
machinery:prediction commit=<full sha> actor=<actor> [session=<session>] observation="<text>" expect="<text>"
machinery:verdict <accept|reject> commit=<full sha> actor=<actor> [session=<session>] observed="<text>"
machinery:intake
machinery:intake <intake id>
machinery:intake-base <full sha> <contract hash>
machinery:intake-planned <full sha> <contract hash>
machinery:no-design-change <reason>
machinery:capstone
machinery:demo <command>
machinery:learning <text>
machinery:observation <kind> <text>
machinery:observation-filed <item id> <kind> <text>
machinery:repair <short sha>
machinery:decision <actor>
machinery:supersedes <decision id>
machinery:vetoable
machinery:confirmed <actor>
```

- `machinery:ids` lists the design ids the story delivers. Several lines add up.
- `machinery:scope` lists the path globs the story writes.
- The tier of a story is the highest tier among its ids (section 23). The label `tier:<tier>` on
  the story mirrors it for people and for routing; when label and ids disagree the ids win.
- The actor is `--actor`, else the environment variable `BEADS_ACTOR`, else `git config user.name`.
- The session is the environment variable `MACHINERY_SESSION`, white space at either end removed.
  When it is set and not empty, `proof --post`, `predict` and `accept` write `session=<value>` on
  their line, and no session part otherwise. A value that is not one word (it holds white space
  or a `"`) is a usage error, before anything is written:
  `usage: MACHINERY_SESSION must be one word, got "<value>"`. In Claude Code the `session-start`
  hook exports the harness's session id as `MACHINERY_SESSION` (section 21.1); another harness sets
  it or leaves it unset. The last word of a proof line is its session when it starts with
  `session=`; the words between the sha and it are the actor.
- `machinery:intake` with nothing after it marks an intake item; `machinery:intake <id>` marks a
  story of that intake. The other intake lines are described in section 15.7.
- `machinery:repair` records that a story was closed while the integration branch was red, as
  its repair (section 12.2). The learning, observation and decision lines are described in
  section 15.8.

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

Every story has type `task` and a priority from its tier: 2 routine, 1 consequential, 0
critical and vital.

A story title is `Prove <machine> <event>` or `Prove <id>` unless given above. Its labels
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
| Every `Prove` story with an id in a critical or vital area | The `Model <area>` story | The formal model the tier requires. |
| `Prove fm:<id>` whose subject is `invoke:<machine>.<src>` | The `internal transitions` story of that machine | The invoke path the failure is induced on. |
| `Prove cx:<spec>:<run>` | The `Prove` stories of the cases its trace steps name | The transitions the replay drives. |

Edges are only created between stories of the same plan, except that the capstone of an intake
is also blocked by the stories its intake already has (section 15.7).

`machinery plan --apply [--parent <id>]` creates the stories and edges in the backlog, skips a
group whose ids all carry a `story` that exists in the backlog, writes the new story ids into the
`story` field of the owed entries, and prints `plan: <n> created, <n> existing, <n> edge(s)`.
`machinery plan --link <story> <id>...` attaches ids to an existing story instead: it appends a
`machinery:ids` note to the story, sets the `story` field of those owed entries and prints
`plan: <n> id(s) linked to <story>`.

Output is sorted and deterministic. With nothing owed it prints nothing and exits 0.

### 15.3 `machinery waves`

Reads every story that is not closed, not deferred, not an epic, not an intake item (section
15.7) and not a decision (section 15.8), with its blocking edges, and prints the layers of the graph: wave 1 holds the stories with no open blocker, wave 2 those
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
  `machinery:ids` line and is not a capstone. Work in the design's name names the ids it proves;
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
- `run` for a builder: `machinery check --base <base>`, `machinery run --affected --base <base> --
  <commands.e2e>`, `machinery verify --story <id>`, `machinery proof --story <id> --post`.
- For a verifier, `read` never contains the proof, and `run` starts with
  `machinery predict <id> --observation "..." --expect "..."`, then
  `machinery proof --story <id>`, then `machinery accept <id> --verdict accept|reject --observed
  "..."`. The brief also lists every formal weakening accepted since the base (section 16.6).
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

When the tier asks for a verifier, the sessions of section 15.1 are compared as well. A prediction
for the commit, by an actor who posted no such proof, whose session is the session of a proof the
evidence is valid for is `ERROR  verifier shares a session with the builder: <session>`, once per
session. When such a proof records no session the gate prints
`warn   no session recorded for proof: independence rests on the actor name`, and when such a
prediction records none, `warn   no session recorded for prediction: independence rests on the actor name`.
A missing session is never an error: harnesses that give none exist.

Findings name what is missing, for example `ERROR  no prediction for <short sha>: a verifier runs
machinery predict before reading the proof`, `ERROR  verifier is the builder: <actor>`,
`ERROR  unproven: <id>`. `checked:` counts ids, proofs, predictions and verdicts. A story without
a `machinery:ids` line is `not demonstrated: story names no design ids`.

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
the proof's and, where both record one, from another session, before the verdict.

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
machinery:intake-base <full sha of HEAD> <contract hash>
```

There is one `- ` line per `--refuses`, in the order given, and `- none` without one; a newline
inside a value becomes a space. The base is the commit the requirement arrived at and the
contract hash (section 15.4) there. An empty title or intent is a usage error. `intake new`
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
| `planned` | Every new id is in the `machinery:ids` of a story of the intake, and the intake has its capstone (`every new id has a story; capstone <id>`). | `<n> new id(s) without a story; run machinery intake plan <id> --apply`, or `no capstone; run machinery intake plan <id> --apply` |

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

Without `--apply` it prints the stories as the Markdown of section 15.2, the capstone last. With
`--apply`:

1. it creates every story and edge in one `bd create --graph` call; an edge to a story that
   exists already uses its id. bd creates the graph whole or not at all;
2. only after that call succeeded it writes `owed.yaml`: each id of a story created now is added
   when it is not owed yet, with the reason `owed by intake <id>`, and the `story` field of each of
   those ids names its story. When the call fails nothing is written, so the owed list and the
   backlog are as they were;
3. it appends `machinery:intake-planned <full sha> <contract hash>` to the intake's notes, unless a
   line with the current contract hash is there already.

It prints `plan: <n> created, <n> existing, <n> edge(s)` (existing: the stories the intake had
before), `owed: <n> added`, and `intake <id>: planned at <short sha>, contract <12 hex>` when step 3
wrote a line. Both forms first print `intake <id>: no new id: the design change adds none; only
the capstone is planned` for an intake without new ids (a rewording, a removal, or a
requirement the design already covers), and `intake <id>: every new id has a story; nothing to
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
**current** unless another decision carries `machinery:supersedes <its id>`, whatever its status.
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
   is added, with the reason `unproven at adoption (<short sha>)`. It never removes an entry
   and never changes a reason. When a stage fails the config and the baselines are still
   written, the findings print, no owed entry is written and the exit code is 1.
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
| `design` | The stages of `machinery lint` (section 3.1). When at least one machine was read, `checked:` adds the machines bound to the model and those unbound (section 6.4): `checked: 8 stages, 1 bound machines, 1 unbound machines`. The line `ids: <n> proposed, <n> owed, <n> proven, <n> retired` counts the ids by status (section 14). When the risk file has a vital area, `proofs awaiting: <n>` follows, then `vital: <id> proven (by test and model; proof awaited)` for every id a property is over that is not owed, and `assumed, unproven: <area> <name>: <statement>` for every assumption whose evidence is owed (section 23.2). Then the candidates of section 23.3, always: `vital candidates: <n>, dismissed: <n>`, and one line `  <id>  <reason>` per candidate. | `<stage>.txt` each |
| `generation` | `machinery gen` (section 7). When gen refuses, not demonstrated: `gen refused: <stages> have errors`. | `generation.txt` |
| `trace`, `tests`, `arch` | The stage of that name. | `<stage>.txt` |
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
4. `machinery intake plan <id> --apply` writes every story at once from what the design now owes,
   owes the new ids, and adds the capstone that will show the requirement on the running system.
5. `machinery waves` schedules the stories; `machinery waves --freeze` pins the contract for a
   wave. A design change found later is its own change that lands alone, never an edit inside a
   running wave, and `machinery intake plan` refuses while a wave is frozen.

**Plan.** The planner (the PM role) owns the backlog and the risk file. Obligations owed without a
requirement behind them (the skeleton of a new project, an adoption) become stories with
`machinery plan --apply`. Either way, a story in the design's name names the ids it proves:
`machinery waves` fails on one that does not, and counts the stories outside machinery.

**Build.** One story, one builder, one worktree, briefed with `machinery brief <story>`:

1. Write the end-to-end check for the story's ids, run it against the real stack, see it fail for
   the right reason.
2. Change the code until it passes. Per change: `machinery check --base <base>` and a targeted run
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

An existing project starts with `machinery assess` (section 18), then `machinery adopt`
(section 17.2): the first owed list is whatever the first modeled area declares, and everything
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
`not demonstrated: nothing to check`. The exit code follows section 3.3. Probe runs never count as evidence.

The probe is expensive: one test run per mutation. It is run per batch and at the end of an epic,
not per story, and it is never part of `machinery check` or `machinery verify`.

The probe is the measure of test quality in this method. There are no unit tests to count; a
suite is good when it kills the mutations of the behavior it claims.

## 21. Agent surfaces

Agents use machinery through its commands. The checks are tools, not prose: an agent is told
which command to run and reads its findings, and is never asked to remember a rule.

| Surface | Where | Holds |
|---|---|---|
| Skill | `skills/machinery/SKILL.md` | When to use machinery and the command for each step. Under 120 lines; details are in `docs/` and are opened only when needed. |
| Agent prompts | `agents/machinery-pm.md`, `agents/machinery-verifier.md`, `agents/machinery-modeler.md` | One role each, under 60 lines, built on `machinery brief`. |
| Plugin manifests | `.claude-plugin/plugin.json`, `.codex-plugin/plugin.json` | Name, version, and the paths above. |

There is no hook. Nothing in machinery blocks an agent session, keeps session state or takes
custody of files. The gates fail closed where they run: `make check`, `machinery landing` and CI.

Routing. A brief names the model class for the tier (`routing` in `machinery.yaml`): the smallest
model that fits routine stories, a stronger one where a wrong result costs more, and the strongest
only for formal models. A brief gives pointers and commands; context is read from the files it
names, in the worker's own session.

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

An invariant's `entity` is empty for a top-level invariant. An imported failure mode that is not
linked (section 5.4.1) has an empty `subject` and `required`. A machine's `role` is `lifecycle` or
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
