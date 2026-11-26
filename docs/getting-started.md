# Getting started

A new project, from an empty directory to a proven design, in ten minutes. Every command in a
`sh` block on this page is executed, in order, by the test suite (`docs_getting_started.txtar`).

You need `machinery` and `modelith` 0.6.0 on your PATH, and git. The example project is a ticket
desk whose whole system is one shell script; the files are in
`e2e/testdata/fixtures/starter/` of the machinery repository (`MACHINERY_REPO` below is a checkout
of it).

## 1. Scaffold

```sh
machinery init
```

This writes a `machinery.yaml` in which nothing applies yet, and the design directories.

## 2. Write the design first

The design is small, structured data: a Modelith model (the vocabulary: entities, actions,
invariants, scenarios) and one state machine per entity whose next legal step depends on its
past. Copy the starter's design, its system and its test over the scaffold:

```sh
cp -R "$MACHINERY_REPO/e2e/testdata/fixtures/starter/." .
```

- `design/model/tickets.modelith.yaml`: one entity `Ticket` with actions `open`, `close`,
  `reopen`, one invariant and one scenario.
- `design/machines/Ticket.machine.json`: states `Open` and `Closed`. Every state answers every
  event: a transition, or an `_ignores` entry with the reason and the refusal the caller sees.
  The format is `docs/machine-format.md`.
- `machinery.yaml`: lists the model and the machines directory, says where code and tests live,
  and waives one cross-reference rule with a reason.

## 3. Generate what follows from the design

```sh
machinery gen
machinery lint
```

`gen` writes `design/gen/obligations.json` and one runtime table per machine. Each transition,
each ignore, each invariant and each scenario is now an **obligation** with a stable id
(`tc:TICK-837662` is "Open on:close -> Closed"). An obligation must be proven on the running
system, or listed in `design/owed.yaml`.

## 4. Build the system and test it end to end

The system is `bin/tickets`. The test is `e2e/tickets_test.sh`: it runs the real script, asserts
what it does, and only then appends one receipt per obligation to the file machinery names in
`MACHINERY_RECEIPTS`. A receipt proves only from a run made by `machinery run`, and only when it
names a test file that claims the id, as `tickets_test.sh` does in the comment below. There are
no unit tests and no mocks, here or in any project that uses machinery. Code says which actions
it serves and tests say which obligations they verify, in a comment:

```
# machinery:serves act:Ticket.open act:Ticket.close act:Ticket.reopen
# machinery:verifies tc:TICK-* sc:agent-closes-and-reopens-a-ticket inv:closed-has-resolution
```

## 5. Check, run, verify

Evidence is bound to a commit, so commit first:

```sh
git init -q -b main
git add -A
git commit -q -m "ticket desk"
machinery check
machinery run -- sh e2e/tickets_test.sh
machinery verify
```

`check` runs every static stage. `run` records the receipts of one test run. `verify` joins the
obligations to that evidence: each is proven, partial (a transition seen without all its
effects), failed or unproven. A stage ends in one of four outcomes: `passed`, `failed`,
`not applicable`, or `not demonstrated`, and the last one blocks like a failure.

## 6. See where you are

```sh
machinery status
machinery proof --prefix tc:TICK-
machinery owed list
machinery checks
```

## Where next

- Something you cannot prove yet goes into `design/owed.yaml` with a reason
  (`machinery owed sync` writes it from the evidence). The owed list only shrinks.
- Risk tiers, backlog stories and parallel waves: `docs/pm.md`. A new requirement starts with
  `machinery intake new`, changes the design, and only then becomes stories, all at once, with
  `machinery intake plan --apply`.
- A formal model for a critical area: `docs/formal.md`.
- An existing project: `docs/brownfield.md`.
- Every format, output line and exit code: `docs/formats.md`.
