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
   parallel waves (`machinery plan`, `machinery waves`), and gates each story by its risk tier
   (`machinery story`).

Three rules shape everything below:

- **Formality follows risk.** Each area of the design has a tier chosen at planning (section 22):
  routine (machines and generated end-to-end cases), consequential (plus a fresh verifier who
  predicts before reading the proof), critical (plus a formal model).
- **Evidence comes from the real system.** There are no unit tests and no test doubles, in
  machinery or in a project that uses it (section 9.4). Test quality is measured by probing
  (section 19), not by counting tests.
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
  risk: design/risk.yaml        # optional: risk tiers (section 22)
  formal: design/formal         # optional, this is the default; used when the directory exists
  facts: design/facts.yaml      # optional: facts the design declares about itself (section 21.2)
  owed: design/owed.yaml        # optional, this is the default
  gen: design/gen               # optional, this is the default
contracts:                      # optional
  - kind: openapi               # openapi | proto
    path: contracts/openapi.yaml
checkers:                       # optional: external checkers (section 21)
  - id: rules
    covers: ["act:*", "inv:*"]
dependencies:                   # optional: external dependencies of the system
  - name: postgres
    posture: real               # real | sandbox
trace:                          # optional
  code: ["internal/**/*.go"]    # files that may carry machinery:serves tags; default: scope
  tests: ["e2e/**"]             # files that may carry machinery:verifies tags
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
tools:
  modelith: modelith            # optional, executable name or path; this is the default
  java: java                    # optional
  node: node                    # optional
  bd: bd                        # optional
  dir: ""                       # optional: tools directory (section 16.8)
  xstate: false                 # optional: true enables stage xstate (section 6.8)
waivers:                        # optional, see section 6.7
  - {rule: action-event, id: "act:Order.place", reason: "creation enters the initial state"}
```

Rules:

- Unknown keys anywhere are a config error. A config error prints `config: <message>` on stderr
  and exits 2.
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
| `machinery check` | Every stage of `lint`, then `gen-drift`, `formal`, `checkers`, `arch`, `tests`, `trace`, `contract`. With `--base <git-ref>` also stage `ratchet` (section 11.3). |
| `machinery checks` | Prints the declaration of applicable checks (section 3.4). A report, never a gate. |
| `machinery run -- <command...>` | Runs a test command and records a receipts file (section 10). |
| `machinery verify` | Stage `verify` (section 12). `--allow-dirty`, `--affected --base <ref>`, `--story <id>`. |
| `machinery affected --base <ref>` | Prints the obligation ids a change touches (section 12.1). |
| `machinery landing --base <ref>` | The end gate: `check`, the full suite through `run`, `verify` (section 12.2). |
| `machinery owed list` | Prints the owed list. |
| `machinery owed sync` | Rewrites `owed.yaml` from the current evidence (section 11.2). `--prune-only` never adds. |
| `machinery proof` | Prints a proof block (section 13). `--story <id> --post` writes it to the story. |
| `machinery status` | Prints a summary (section 14). A report, never a gate. |
| `machinery probe -- <command...>` | Stage `probe`: mutates the generated tables and expects the tests to notice (section 19). Never part of `check`. |
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
| `machinery checkers run [<id>...]` | Runs the external checkers registered on this machine and records their evidence (section 21). |
| `machinery checkers list` | Lists the configured checkers, whether each is registered locally, and the state of its evidence. |
| `machinery init` | Scaffolds a new project (section 17.1). |
| `machinery adopt` | Brings an existing project under machinery without touching its code (section 17.2). |
| `machinery config` | Prints the effective configuration as YAML, defaults applied. |
| `machinery version` | Prints `machinery <version>`. |

`lint` and `check` accept `--stage <name>` (repeatable): only the named stages run, in their usual
order, and the exit code depends on them alone. An unknown stage name is a usage error (exit 2).

`machinery` with no arguments prints help and exits 0. `version`, `init`, `adopt`,
`formal tools` and help work without a `machinery.yaml`; every other command needs one.

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
| `risk` | Each risk area names real design ids, so tier gates attach to real behavior. |
| `xref-model-machine` | The lifecycle a machine implements is the lifecycle the model promises. |
| `xref-contract` | Every action is reachable through a published operation. |
| `xref-failure` | Every side effect, dependency and tiered action has a declared failure behavior. |
| `xstate` | A generated table means what XState would execute for the same machine. |
| `gen-drift` | Tables and obligations in the tree are the ones the design produces. |
| `formal` | The formal model of a critical area is current, not vacuous and not weakened. |
| `checkers` | Verdicts of external checkers are for this exact design. |
| `arch` | The layering the design declares holds in the code. |
| `tests` | Evidence comes from the real system: no unit tests, no test doubles. |
| `trace` | Every action is served by code and every obligation is claimed by a test or owed. |
| `contract` | Stories running in parallel build against the same design. |
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
| `fv` | `fv:<area>` | Each critical area of the risk file (section 22) | Obligation, proven by the formal stage. |
| `cx` | `cx:<spec>:<run>` | Each counterexample trace of a formal model (section 16.10) | Obligation, proven by replay. |
| `ck` | `ck:<checker id>:<name>` | Each obligation an external checker's evidence declares (section 21.5) | Obligation. |

- The scenario slug is the scenario name lowercased, every run of characters outside `[a-z0-9]`
  replaced by one `-`, with leading and trailing `-` removed. Example: `Customer pays and the
  order ships` becomes `sc:customer-pays-and-the-order-ships`.
- Ids are unique across the whole project. A duplicate is an ERROR in the stage that reads it.
- An **obligation** is an id that must be either proven at the current commit or listed in
  `owed.yaml`. The obligation kinds are `inv`, `sc`, `tc`, `fm`, `fv`, `cx` and `ck`.
- An id pattern is a full id or a prefix ending in `*` (`tc:ORDE-*`, `act:Order.*`).

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

Every machine is one of:

- a **lifecycle machine** of a model entity: the root key `_lifecycle_of: "<Entity>"`, or a file
  base name equal to an entity name (`Order.machine.json` for entity `Order`);
- an **operational machine**: the root key `_role: "operational"`.

In a lifecycle machine, top-level states whose name starts with an uppercase letter are **domain
states**. States whose name starts with a lowercase letter are transient overlay states.

### 5.3 Contracts

- `kind: openapi`: an OpenAPI 3.x document in YAML or JSON. An operation binds to a model action
  with the extension `x-machinery-action: "<Entity>.<action>"` (a string or a list of strings).
- `kind: proto`: a `.proto` file. An rpc binds to a model action with a comment line
  `// machinery:action <Entity>.<action>` directly above the `rpc` line.

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
`entity:<Name>`. `required: refuse` is only valid with `kind: abuse`, and `kind: abuse` needs an
`act` or `entity` subject.

### 5.5 Risk file

`risk.yaml` assigns risk tiers to areas of the design. See section 22.

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

Lints every `*.machine.json` in the machines directory with the rules in
`docs/machine-format.md`: subset conformance, name patterns, strict target resolution, declared
delays, reachability, dead ends, every invoke has `onError` and a timeout, event completeness on
resting states (`_ignores`), guard-false dispositions (`_refusal`, `_exhaustive`), fail-closed
tags. Two machines whose id tags collide (section 7.2) are an ERROR.

`checked:` counts machines, states and transitions. Not applicable when `design.machines` is
absent.

### 6.3 `failure-modes`

Parses the file against section 5.4. Unique ids, valid `kind` and `required` values, well-formed
`subject`. `checked:` counts failure modes and, of those, abuse cases. Not applicable when
`design.failure_modes` is absent.

### 6.4 `xref-model-machine`

For every machine:

| Rule | Check |
|---|---|
| `machine-binding` | The machine is a lifecycle machine of an existing entity, or operational (section 5.2). |
| `state-enum` | Lifecycle machine of entity E: let D be its domain states. Among the enums used as the `type` of an attribute of E, exactly one must have a value-name set equal to D. Otherwise ERROR, naming the missing and extra names against the candidate enum with the largest overlap (ties broken by enum name). |
| `event-action` | Lifecycle machine of E: every event name under any `on` must be the name of an action of E, or be listed in the root `_external_events`. |
| `action-event` | Every action of an entity that has a lifecycle machine must appear as an `on` event somewhere in that machine. |

`checked:` counts machines, domain states, events and actions.

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
| `subject-resolves` | Every failure-mode subject names an existing invoke, a configured dependency, a model action or a model entity. |
| `invoke-failure-mode` | Every `invoke` `src` of every machine is the subject of at least one failure mode. |
| `dependency-failure-mode` | Every configured dependency is the subject of at least one failure mode. |
| `tiered-action-abuse` | Every action that belongs to a consequential or critical area (section 22) is the subject of at least one failure mode of kind `abuse`. |

`checked:` counts failure modes, invokes, dependencies and tiered actions. Not applicable when
`design.failure_modes` is absent.

### 6.7 Waivers

A cross-reference finding (sections 6.4 to 6.6) can be waived in `machinery.yaml`:

```yaml
waivers:
  - {rule: action-event, id: "act:Order.place", reason: "creation enters the initial state"}
```

`rule` is a rule name from the tables above. `id` is the subject the finding names (an `act:` id,
or `<machine id>.<event>` for `event-action`, or `invoke:<machine id>.<src>`). `reason` is
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
- `projection.json` (section 21.3), only when `checkers:` or `design.facts` is configured;
- `tables/<machine id>.table.json` for each machine (section 7.3);
- `tables/<machine id>.table.md`, a readable rendering of the same table;

and beside each model its `<name>.modelith.md`.

All generated files are committed and never edited by hand. JSON is written with two-space
indentation, keys in the order given in this document, and a trailing newline. Writes are atomic
(temporary file, then rename). `machinery gen` refuses to write (ERROR) when the `model`,
`machines` or `failure-modes` stage has an ERROR.

`machinery gen --check` regenerates in memory and reports `DRIFT  <path>: stale`, `DRIFT  <path>:
missing`, and `ERROR  <path>: unexpected file in gen directory`.

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
- `source` is the file the id comes from: the model, the machine, the failure-mode file, the
  risk file for `fv`, the trace file for `cx`, the evidence file for `ck`.
- `effects` is present only for `tc` and may be empty.
- `served` holds every `act` id, sorted by id.

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

- `states` lists every state in document order, nested states by dotted path. `kind` is `atomic`,
  `compound` or `final`. `resting` is true for a top-level, non-final state with no `invoke` and
  no `always`.
- `transitions` keeps document order, because branch order is evaluation order: the first branch
  whose guard holds wins. `event` is the event name, the delay name, or the invoke `src`; it is
  `""` for `always` and `stateDone`. `guard` and `target` are `""` when absent. `target` is the
  resolved dotted path.
- `clauses` lists the clause cases in the order of their transitions, then of the declaration. It
  is an empty list when the machine declares no `_guards`. (The example line comes from another
  machine; the fixture machine has none.)
- `units` lists every guard, action and actor name the machine uses, each sorted.
- `delays` maps each `_delays` name to its bound in milliseconds (the leading integer of the
  declaration).

A service loads its tables from the directory named by `MACHINERY_TABLES_DIR` when that variable
is set, and from its own configured location otherwise. Section 19 depends on this.

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
- A `tc:` token may end with `*` and then matches every `tc` id with that prefix.
- `machinery:serves` is read from files matched by `trace.code`. `machinery:verifies` is read
  from files matched by `trace.tests`.

Stage `trace` rules:

| Rule | Check |
|---|---|
| `unknown-id` | Every tagged id exists in the design; every `*` prefix matches at least one id. |
| `unserved` | Every `act` id has at least one `machinery:serves`, or is listed in `owed.yaml`. |
| `unverified` | Every obligation of kind `inv`, `sc`, `tc`, `fm`, `cx` and `ck` has at least one `machinery:verifies`, or is listed in `owed.yaml`. `fv` ids are proven by the formal stage and are not claimed by tests. |

`checked:` counts code files, test files, `act` ids and obligations. Only files inside `scope`
are read. Zero files matched by a configured glob list is an ERROR. Not applicable when `trace:`
is absent.

Trace is static and cheap. It proves that a test claims an id, not that the test passed. Only
`verify` proves that.

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
shrinks. `machinery arch --write-baseline` writes the current violations and is a deliberate act.

`checked:` counts files, imports and functions. Only files inside `scope` are read. Not applicable
when `constraints:` is absent.

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
neither `tests:` nor `trace.tests` is configured. A project with e2e roots and no test file at all
is `not demonstrated: nothing to check`.

## 10. Runs and receipts

Evidence is produced by the project's own end-to-end tests, in any language, run through
`machinery run`:

```
machinery run [--stack "<description>"] -- <command> [args...]
```

`machinery run`:

1. reads the current commit (`git rev-parse HEAD`) and whether the tree is dirty
   (`git status --porcelain` not empty, ignoring the receipts directory and the owed file);
2. deletes receipt files in the receipts directory that are not valid for the current commit
   (section 10.1);
3. creates `<receipts dir>/<UTC timestamp>-<short sha>.jsonl` and writes the header line;
4. runs the command with extra environment variables: `MACHINERY_RECEIPTS` (absolute path of
   the file), `MACHINERY_COMMIT` (the full commit sha) and, for a targeted run (section 12.1),
   `MACHINERY_ONLY` (the ids to exercise, comma separated);
5. appends the end line when the command exits;
6. exits with the command's exit code.

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
current commit only in the owed file (`git diff --name-only <receipt commit> HEAD` lists nothing
but the owed file). The owed file is bookkeeping about evidence, not part of the system under
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
keys, duplicate ids and unknown ids are an ERROR wherever the file is read. A missing file is an
empty list.

The owed list is the single ratchet of the method. In a new project nearly everything is owed
right after the skeleton; each story proves some ids and removes them. In an existing project the
owed list is the baseline of what was never proven.

### 11.2 `machinery owed sync`

Reads the evidence exactly as `verify` does (section 12), then rewrites `owed.yaml`:

- removes every entry whose id is proven;
- unless `--prune-only`, adds every obligation that is neither proven nor listed, with the reason
  `unproven at <short sha>`;
- keeps existing reasons and story fields.

It prints `owed: <n> kept, <n> added, <n> removed` and exits 0. It refuses with an ERROR when
there is no complete run valid for the current commit. Because of section 10.1, the sequence
`machinery run`, `machinery owed sync`, commit, `machinery verify` works without running the
tests twice.

### 11.3 Stage `ratchet` (`machinery check --base <git-ref>`)

Compares the current state with the same files at `<git-ref>` (read with `git show`). Nothing
that was established at the base may quietly stop being established:

| Rule | Finding |
|---|---|
| `proven-to-owed` | An id that was an obligation at the base and was not owed there is owed now: `ERROR  ratchet: <id> was proven at <ref> and is owed now`. New obligations may be owed. |
| `scope-narrowed` | A scope glob of the base is missing now: `ERROR  ratchet: scope narrowed: <glob>`. |
| `tier-lowered` | An id has a lower tier than at the base: `ERROR  ratchet: tier lowered: <id> <old> -> <new>`, unless the area's `why` changed too, which prints as `warn` with the new reason. |
| `baseline-grew` | An architecture or tests baseline has an entry the base did not have: `ERROR  ratchet: baseline grew: <rule> <subject>`, unless the scope widened in the same change. |
| `spec-weakened` | A formal baseline acceptance (section 16.6) that the base did not have prints as `warn   ratchet: weakened since <ref>: <spec>: <change> (<reason>)`. |

`checked:` counts obligations, scope globs, tiered ids and baseline entries compared.

## 12. Verify

Stage `verify` joins obligations to evidence.

Evidence is every receipts file in the receipts directory that is valid for the current commit
(section 10.1). When the working tree has uncommitted changes outside the owed file and the
receipts directory, the evidence does not describe what is on disk: without `--allow-dirty` that
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

- `ERROR  unknown id in receipt: <id>`.
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
2. ids named by a `machinery:verifies` tag in a changed test file;
3. for each `act` id named by a `machinery:serves` tag in a changed code file: every `tc` id of
   the entity's lifecycle machine whose trigger is that action's event, and every `inv` id the
   action `preserves`;
4. every `tc` id of a machine whose file changed.

With nothing touched it prints nothing and exits 0.

`machinery run --affected --base <ref> -- <command>` computes the same list and exports it as
`MACHINERY_ONLY`. `machinery verify --affected --base <ref>` reports `unproven` only for those
ids. A targeted verify is a check for one change; only a full `verify` supports a landing.

### 12.2 The end gate: `machinery landing`

```
machinery landing --base <ref> [--stack "<description>"] [-- <command> [args...]]
```

Runs, in order and stopping at the first that does not pass: `machinery check --base <ref>`; the
project's whole end-to-end suite through `machinery run` (the command after `--`, else
`commands.e2e`); `machinery verify`. It prints the stage blocks of each and one final line:

```
landing: <passed|failed> at <short sha> (attempt <n>, fix-forward <k> of 2)
```

The whole suite runs once, at the head that is ready, not per story. machinery counts attempts
per branch in `<git common dir>/machinery/landing.json`: an attempt at a new head after a failed
attempt is a fix-forward. After two fix-forwards a further failure also prints
`stop: two fix-forwards used; return the failing stories to the backlog`. The count is a report:
it never turns a passing landing into a failing one. A passed landing resets it.

## 13. Proof

```
machinery proof [--ids <id>,<id>...] [--prefix <id prefix>] [--story <id> [--post]]
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
learnings:
```

`criteria` lists one line per selected obligation that has a receipt, sorted by id; an `fv` id
prints as `<id> -> formal -> pass: <n> run(s), <n> mutation(s) detected`. `not proven` lists the
selected obligations that are partial, unproven or owed, and every unproven assumption of a
formal model that covers a selected id. `learnings:` is left empty for the author. Exit 0 when
every selected obligation is proven, 1 otherwise, 2 when a selector matches nothing.

`--story <id>` selects the ids of a backlog story. `--post` appends the block to the story's notes
after a line `machinery:proof <full sha> <actor>` (section 15.1).

## 14. Status

```
project: <root directory name>
obligations: <n> (tc <n>, sc <n>, inv <n>, fm <n>)
tiers: routine <n>, consequential <n>, critical <n>
proven: <n>  partial: <n>  failed: <n>  owed: <n>  unproven and not owed: <n>
owed without a story: <n>
evidence: <short sha> <clean|dirty>, <n> run(s), <n> receipt(s)
```

The first count line lists only kinds that occur, in the order `tc, sc, inv, fm, fv, cx, ck`.
The `tiers` line counts obligations by the tier of their area and is omitted when there is no
risk file. When there is no evidence the last line is `evidence: none at <short sha>`. Exit 0
unless the config is invalid.

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
machinery:proof <full sha> <actor>
machinery:prediction commit=<full sha> actor=<actor> observation="<text>" expect="<text>"
machinery:verdict <accept|reject> commit=<full sha> actor=<actor> observed="<text>"
```

- `machinery:ids` lists the design ids the story delivers. Several lines add up.
- `machinery:scope` lists the path globs the story writes.
- The tier of a story is the highest tier among its ids (section 22). The label `tier:<tier>` on
  the story mirrors it for people and for routing; when label and ids disagree the ids win.
- The actor is `--actor`, else the environment variable `BEADS_ACTOR`, else `git config user.name`.

### 15.2 `machinery plan`

Prints one story per group of owed ids, as Markdown (one `##` heading per story with `### Type`,
`### Priority`, `### Labels`, `### Description` and `### Acceptance Criteria` sections), or with
`--json` as a `bd create --graph` plan.

Groups:

- owed `tc` ids are grouped by machine and trigger event: one story per `(machine, event)` for
  `on` triggers, and one story per machine named `internal transitions` for every other trigger;
- one story per owed `sc`, `fm`, `inv`, `cx` and `ck` id;
- one story per owed `fv` id, titled `Model <area> (<tool>)`;
- one story `Serve <Entity> actions` per entity with owed `act` ids.

A story title is `Prove <machine> <event>` or `Prove <id>` unless given above. Its description
lists the owed ids with their titles and effects and carries the `machinery:ids` line. Its labels
are `tier:<tier>` and `machinery`. Its acceptance criteria say that each listed id is proven by
`machinery verify` at the story's commit and removed from `owed.yaml`, and for a consequential or
critical tier that `machinery story <id>` passes.

Edges. An edge means that one story needs a specific output of another:

| From (blocked) | To (blocker) | The output needed |
|---|---|---|
| Every `Prove` story with an id in a critical area | The `Model <area>` story | The formal model the tier requires. |
| `Prove fm:<id>` whose subject is `invoke:<machine>.<src>` | The `internal transitions` story of that machine | The invoke path the failure is induced on. |
| `Prove cx:<spec>:<run>` | The `Prove` stories of the cases its trace steps name | The transitions the replay drives. |

Edges are only created between stories of the same plan.

`machinery plan --apply [--parent <id>]` creates the stories and edges in the backlog, skips a
group whose ids all carry a `story` that exists in the backlog, writes the new story ids into the
`story` field of the owed entries, and prints `plan: <n> created, <n> existing, <n> edge(s)`.
`machinery plan --link <story> <id>...` attaches ids to an existing story instead: it appends a
`machinery:ids` note to the story and sets the `story` field of those owed entries.

Output is sorted and deterministic. With nothing owed it prints nothing and exits 0.

### 15.3 `machinery waves`

Reads every story that is not closed, not deferred and not an epic, with its blocking edges, and
prints the layers of the graph: wave 1 holds the stories with no open blocker, wave 2 those
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
- `warn   no scope declared: <story>`.

`--json` prints the same data as one JSON document. The exit code follows section 3.3.

### 15.4 Contract freeze

Parallel work is safe only against one contract. The **contract hash** is the sha256 over the
sorted list of `<path> <sha256 of the file>` lines of the design sources: the model files, every
`*.machine.json`, the failure-mode file, the risk file, the facts file and the contract files.
Generated files, the owed list and formal models are not part of it.

`machinery waves --freeze` requires a clean `gen-drift` stage and writes
`<git common dir>/machinery/wave.json`, which every worktree of the repository shares:

```json
{"version": 1, "contract": "<sha256>", "commit": "<full sha>", "files": {"<path>": "<sha256>"}, "stories": ["<wave 1 story ids>"]}
```

Stage `contract` (part of `machinery check`) is `not applicable: no wave is frozen` without that
file. With it, a design source whose hash differs from the frozen one is
`ERROR  contract changed during fan-out: <path>`. A contract change is its own story: land it
alone, then freeze again. `machinery waves --thaw` removes the file when the wave has landed.
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
scope:
  <glob>
run:
  <command>
done when: <one line>
```

- `read` lists, for the ids of the story: the machine file and its generated table for `tc` ids,
  the model file for `act`, `inv` and `sc` ids, the failure-mode file for `fm` ids, the spec
  manifest and trace file for `fv` and `cx` ids, and `docs/formats.md` when the project has it.
- `run` for a builder: `machinery check --base <base>`, `machinery run --affected --base <base> --
  <commands.e2e>`, `machinery verify --story <id>`, `machinery proof --story <id> --post`.
- For a verifier, `read` never contains the proof, and `run` starts with
  `machinery predict <id> --observation "..." --expect "..."`, then
  `machinery proof --story <id>`, then `machinery accept <id> --verdict accept|reject --observed
  "..."`. The brief also lists every formal weakening accepted since the base (section 16.6).
- For a modeler, `read` is the machine or model the formal model is derived from and the manifest;
  `run` is `machinery formal gen`, `machinery formal check <spec>`, `machinery formal baseline`.

A brief for a routine story fits in 40 lines.

### 15.6 Tier gates: `predict`, `accept`, `story`

`machinery story <story>` is stage `story`: it reads the story from the backlog and judges it at
the current commit.

| Tier | The gate requires |
|---|---|
| routine | Every id of the story is `proven` by evidence valid for the current commit (section 12) and is not owed. A `machinery:proof` line exists for a commit the evidence is valid for. |
| consequential | The routine gate. A `machinery:prediction` for that commit by an actor other than the proof's actor. A `machinery:verdict accept` for that commit by the predicting actor, recorded after the prediction. |
| critical | The consequential gate. Every `fv` id of the areas the story's ids belong to is `proven`, and every `cx` id of the formal models covering them is `proven`. |

Findings name what is missing, for example `ERROR  no prediction for <short sha>: a verifier runs
machinery predict before reading the proof`, `ERROR  verifier is the builder: <actor>`,
`ERROR  unproven: <id>`. `checked:` counts ids, proofs, predictions and verdicts. A story without
a `machinery:ids` line is `not demonstrated: story names no design ids`.

`machinery predict <story> --observation "<what would disprove the claim>" --expect "<predicted
outcome>"` appends the prediction line. It refuses when a proof for the current commit was posted
by the same actor.

`machinery accept <story> --verdict accept|reject --observed "<what was seen>" [--close]` appends
the verdict line. With `accept` it then runs the gate and exits with its code; `--close` closes
the story in the backlog when the gate passed.

The verifier is a fresh context that did not build the change. It receives the verifier brief,
predicts, and only then reads the proof. machinery cannot see what an agent read; it can and does
check who predicted, for which commit, and in which order.

## 16. Formal models

A critical area (section 22) gets a formal model. A model is worth nothing when it is vacuous,
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

A critical area needs, for each of its questions, at least one spec of the matching tool that
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
- `States`, `Effects`, `Final` (leaf paths of top-level final states), `Resting`, `TypeOK`, `Init`.
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
| `timeout_seconds` | Default 300. |

**Reach.** A property that holds because the interesting state is never reached proves nothing.
Every run with `expect: pass` lists at least one reach predicate, and each must be reached:
otherwise `ERROR  <spec> run <run>: reach <name> not reached: the pass is vacuous`.

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

`machinery formal check` runs each mutation with only the named property and records `detected`
(the property is violated), `missed` (it is not) or `inapplicable` (`from` does not occur exactly
once, or the mutated spec does not parse).

Stage `formal`: `ERROR  <spec> mutation <id> missed: <detects> does not catch "<bug>"`,
`ERROR  <spec> mutation <id> does not apply`, and for every invariant and property of an
`expect: pass` run that no mutation names:
`ERROR  <spec>: <property> has no mutation it must catch`. `TypeOK` is exempt.

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
| `no-effect` | The change does not touch what the model describes. | | `read_at` moves to the current commit. |
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
first`), copies the spec's files to a temporary directory, runs every run, then every reach
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

`outcome` is `passed`, `violated` or `error` (the tool could not check the spec; `violated` then
holds the tool's first error line). `input_sha256` is the sha256 over the manifest without its
`sources` block (canonical JSON), the skeleton bytes and the property file bytes. The file has no
timestamps and no durations.

Stage `formal` applies when the formal directory holds a manifest or the risk file has a critical
area. It reads the manifests, the results, the baseline, the risk file and git history, and runs
no tool:

- manifest errors; skeleton drift (as `formal gen --check`);
- `not demonstrated: no result for <spec>; run machinery formal check` when a result file is
  missing; `DRIFT  <spec>.result.json: stale` when `input_sha256` does not match;
- the run, reach and finding rules of 16.4, the mutation rules of 16.5, the baseline rules of
  16.6, the sync rules of 16.7, the assumption warnings of 16.3, the coverage rule of 16.1;
- `ERROR  <spec>: unknown area <name>`.

An `fv:<area>` obligation is proven when the area is covered (16.1) and no ERROR or DRIFT names a
spec that lists the area. When `fv:<area>` is in the owed list, findings about specs whose areas
are all owed print as `warn   owed: <message>`: a model under construction does not block
unrelated work.

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

For a new project. Creates, in an empty or existing directory with no `machinery.yaml`: a minimal
`machinery.yaml`, `design/model/`, `design/machines/`, an empty `design/owed.yaml`, and prints the
Makefile targets the project should add (`check`, `verify`). It never overwrites a file.

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

`backlog:` is `none found` when the project has no `.beads` directory. With `--write`:

1. **Config.** Without a `machinery.yaml`, writes one: `scope` from `--paths`, `trace.code` equal
   to the scope, `trace.tests` and `tests.e2e` from `--e2e` (default `e2e/**`), the model,
   machine and contract files found (`*.modelith.yaml`, `*.machine.json`, `openapi.{yaml,yml,json}`),
   `tests.baseline`, and `backlog.tool: bd` when a backlog was found. With an existing file, adds
   the new `--paths` to `scope`, adds design sources it finds that are not configured yet (so a
   model written after the first adoption is picked up), and changes nothing else in it;
   comments are kept.
2. **Baselines.** Writes the tests baseline with every unit test and test double now in scope
   (section 9.4), and the architecture baseline when `constraints` are configured. Existing
   entries are kept; entries are added only for paths newly in scope.
3. **Owed.** When design sources exist and stages `model`, `machines` and `failure-modes` pass:
   runs `machinery gen` and adds to `owed.yaml` every obligation and every unserved `act` id that
   is not there, with the reason `unproven at adoption (<short sha>)`. It never removes an entry
   and never changes a reason. When a stage fails it prints the findings and writes no owed
   entries.

`adopt` never creates, changes or deletes a file outside `machinery.yaml` and the design
directory. It prints `adopt: wrote <n> file(s)` and the paths. Running it twice with the same
arguments writes nothing the second time.

After `adopt --write`, `machinery check` passes: what exists is baselined or owed, and anything new
in scope is held to the rules. From there:

- the area's design is written or extended a piece at a time (the riskiest stateful resource
  first), and each `machinery gen` plus `machinery adopt --write` owes the new obligations;
- `machinery plan --apply` adds stories for the owed ids to the existing backlog, beside the
  stories already there, and `machinery plan --link` attaches ids to stories that already exist;
- stories prove ids and prune the owed list; baselines shrink as files are touched;
- `machinery adopt --paths <more> --write` widens the scope when the team is ready.

The ratchet (section 11.3) holds each step: scope does not narrow, baselines do not grow without
a wider scope, and nothing proven becomes owed.

## 18. The order of work this supports

**Plan.** The planner (the PM role) owns the backlog and the risk file. For new work it: changes
the model and machines first (`machinery lint`, `machinery gen`); assigns tiers in `risk.yaml`;
runs `machinery owed sync` or `machinery adopt --write` so the new obligations are owed; turns
them into stories with `machinery plan --apply`; and reads `machinery waves` for the schedule.
Design changes land before the stories that build on them: `machinery waves --freeze` pins the
contract for a wave.

**Build.** One story, one builder, one worktree, briefed with `machinery brief <story>`:

1. Write the end-to-end check for the story's ids, run it against the real stack, see it fail for
   the right reason.
2. Change the code until it passes. Per change: `machinery check --base <base>` and a targeted run
   (`machinery run --affected`, `machinery verify --story <id>`).
3. `machinery owed sync --prune-only`, commit, `machinery proof --story <id> --post`.

**Accept.** By tier (section 15.6): routine on evidence; consequential and critical through a
fresh verifier who predicts, reads, then accepts; critical with its formal model current.

**Land.** Stories of a wave are integrated; then the whole suite runs once at the head that is
ready, with `machinery landing --base <base>`, and at most two fix-forwards (section 12.2).
`machinery probe` runs per batch and before a release.

A new project starts with a full-breadth skeleton: every action has an operation and a mounted
handler, every machine moves state through its generated table, and effects do nothing yet. At
that point `verify` passes with a long owed list. The work that remains is the owed list.

An existing project starts with `machinery adopt` (section 17.2): the first owed list is whatever
the first modeled area declares, and everything outside the scope is left alone.

## 19. Probe

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
`not demonstrated: no evidence at <short sha>` and the command exits 1.

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

Stage `probe` reports `ERROR  survived: <id> <mutation> (<what was changed>)` for every survivor
and `checked: <n> mutations, <n> killed, <n> survived`. Selecting nothing that can be mutated is
`not demonstrated: nothing to check`. The exit code follows section 3.3. Probe runs never count as evidence.

The probe is expensive: one test run per mutation. It is run per batch and at the end of an epic,
not per story, and it is never part of `machinery check` or `machinery verify`.

The probe is the measure of test quality in this method. There are no unit tests to count; a
suite is good when it kills the mutations of the behavior it claims.

## 20. Agent surfaces

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

## 21. External checkers

A project can plug its own deterministic checker into `machinery check`: a static analyzer, a
logic solver, or a rule engine that encodes a standard, a control catalog or a regulation.
machinery never learns what the checker is. It defines one input file, one output file and where
the command that connects them is kept.

### 21.1 The parts

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

### 21.2 Declared facts

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

### 21.3 The projection

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

An invariant's `entity` is empty for a top-level invariant. A machine's `role` is `lifecycle` or
`operational`. A machine lists, after its `ignores`, its `refusals` and its guard `clauses` (section 7.2),
each with its `tc` id. Lists that have nothing are empty lists, never absent. The projection does not
contain `ck` obligations: it is the input of the checkers, not their output.

### 21.4 The evidence

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

### 21.5 Obligations from a checker

A checker can do more than judge the design. It can say what the running system must be shown to
do: "an operator can halt processing", "every automated decision is logged with its inputs". Each
entry of `obligations` becomes an obligation `ck:<checker id>:<name>` with the entry's title. From
there it is an obligation like any other: it appears in `obligations.json`, a test claims it with
`machinery:verifies`, a receipt proves it, and until then it is in `owed.yaml`. This is how a
requirement that comes from outside the design reaches the same proof loop as the rest.

### 21.6 The registry and `machinery checkers run`

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
5. validates the document (section 21.4), including that `input_sha256` is the hash of the
   projection it was given; invalid evidence is an ERROR and is not installed;
6. moves the document to the evidence path, and prints its findings and
   `checker <id>: <verdict>, <n> finding(s), <n> obligation(s), <n> covered`.

After the last checker it regenerates `obligations.json`, so new `ck` obligations appear at once.
Exit 1 when any checker has an ERROR or the verdict `fail`, else 0.

### 21.7 Stage `checkers`

Part of `machinery check`, after `formal`. It runs no tool and reads no registry. Not
applicable when neither `checkers:` nor `design.facts` is present.

| Finding | When |
|---|---|
| `ERROR  <facts path>: <problem>` | The facts file is invalid (section 21.2). |
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

## 22. Risk tiers

Formality follows risk. The planner decides, per area of the design and before work starts, how
much formality the area gets, and writes it down as data:

```yaml
version: 1
areas:
  - name: order-payment          # ^[a-z][a-z0-9-]+$, unique
    tier: critical               # consequential | critical
    why: "money moves; a double charge cannot be undone"
    ids: ["act:Order.pay", "tc:ORDE-*"]
    questions: [concurrency]     # required for critical: ordering | concurrency | recovery | structure
  - name: shipping-permission
    tier: consequential
    why: "authorization: only a Clerk ships"
    ids: ["act:Order.ship"]
```

An id in no area is **routine**. An id in several areas has the highest of their tiers.

| Tier | Use it for | What it adds |
|---|---|---|
| routine | Work where a wrong result is cheap to see and to undo. | The machine, its generated transition cases, end-to-end evidence. |
| consequential | Published contract semantics, authorization, tenant isolation, money or audit integrity, data migrations. | An abuse case for every action in the area (section 6.6), and acceptance by a fresh verifier who predicts a disproving observation before reading the proof (section 15.6). |
| critical | Promises whose break harms users and that tests rarely reach: races, lost or repeated work, structural rules that must hold for every shape of data. | A formal model with the honesty checks of section 16. The area yields the obligation `fv:<name>`. |

Stage `risk`:

| Rule | Check |
|---|---|
| `area-shape` | Unique names, a valid tier, a non-empty `why`, at least one id pattern. |
| `id-resolves` | Every pattern matches at least one `act`, `inv`, `sc`, `tc` or `fm` id. |
| `critical-question` | A critical area lists at least one question; a non-critical area lists none. |

`checked:` counts areas, consequential ids and critical ids. Not applicable when `design.risk` is
absent.

The tier is chosen at planning, not discovered at review. Lowering a tier is visible in the
ratchet (section 11.3). When it is unclear which tier applies, the higher one does.
