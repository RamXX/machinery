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
  is an error. The native annotation `_aliases` carries the last one, and rules `event-action`
  and `action-event` read an aliased event as its action.
- Rule `event-action` reports an event listed in `_external_events` or `_failure_events` that is
  an action of the model: an action is a caller's command. The name hint no longer warns about
  an event that is listed as external or has an alias.

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
