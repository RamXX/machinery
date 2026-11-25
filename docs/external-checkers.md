# Plugging in your own checker

A project can add a deterministic checker of its own to `machinery check`: a static analyzer, a
logic solver, or a rule engine that encodes a standard, a control catalog or a regulation.
machinery never learns what the checker is. The contract is `docs/formats.md` section 21.

## The four parts

| Part | Who writes it | Where | Committed |
|---|---|---|---|
| The manifest | You | `checkers:` in `machinery.yaml` | Yes |
| The projection | `machinery gen` | `design/gen/projection.json` | Yes |
| The evidence | Your checker | `design/checkers/<id>.evidence.json` | Yes |
| The registry | Each person, per machine | `.machinery/checkers.local.yaml` | Never |

Everything committed names the checker only by an id you chose. The command that runs it stays
on the machine that has the tool, so a public repository can be checked by a private tool, and
someone without the tool can still run `machinery check`: the stage reads committed files only.

## Wiring one in

1. Declare it and what it judges:

   ```yaml
   checkers:
     - id: rules
       covers: ["act:*", "inv:*"]
   ```

2. `machinery gen` writes the projection: the whole design as one sorted JSON document
   (entities, enums, invariants, scenarios, machines with every case, operations, failure
   modes, dependencies, declared facts). It is the only thing your checker reads.
3. Write an adapter that takes the projection path and an evidence path and writes the evidence
   document: the hash of the projection it read, a verdict, the ids it judged, findings, and
   optionally obligations.
4. Register the command on your machine, in `.machinery/checkers.local.yaml`:

   ```yaml
   version: 1
   checkers:
     rules:
       command: ["sh", "tools/rules-adapter.sh", "{projection}", "{evidence}"]
   ```

5. `machinery checkers run` runs it, validates the evidence and installs it.
   `machinery checkers list` shows the state of each checker. Commit the evidence.

From then on stage `checkers` fails when the evidence is missing, invalid, stale (the design
changed since the checker ran), reports an error, or does not cover what the manifest claims.

## Facts the design does not otherwise state

Some things a checker needs are true of the system but appear in no design source: that an
action handles payment data, that the system decides about people automatically. State them in
`design/facts.yaml` (section 21.2); machinery gives them no meaning and carries them to the
checkers.

## Requirements that come from the checker

A checker can also say what the running system must be shown to do. Each entry of `obligations`
in its evidence becomes an obligation `ck:<checker id>:<name>`: it appears in
`obligations.json`, a test claims it, a receipt proves it, and until then it is owed, like any
other obligation.

The scripts `checkers_*.txtar` show every case with a small stand-in checker
(`e2e/testdata/fixtures/checkerprog`).
