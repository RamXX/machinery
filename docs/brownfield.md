# Adopting machinery in an existing project

You have code, tests, a way of working and a `bd` backlog, and work is in flight. This guide
brings one area of the project under machinery without a rewrite, without a freeze and without
touching the backlog you have. Every command in a `sh` block is executed, in order, by the test
suite (`brownfield_walkthrough.txtar`) on a small existing Go project, the `brownfield` fixture:
a depot with a shipping area and a billing area, unit tests, a smoke script and its own backlog.

The contract is `docs/formats.md` section 17.2. Three properties hold at every step:

- `adopt` never changes existing code. It writes `machinery.yaml` and files under `design/`.
- Only the paths you point it at are covered. Everything else is left alone.
- Coverage only grows: the scope does not narrow, baselines do not grow without a wider scope,
  and nothing proven becomes owed (`machinery check --base <ref>`).

## 1. Look before you write

Point `adopt` at the area that hurts most. Without `--write` it changes nothing:

```sh
machinery adopt --paths 'internal/shipping/**' --e2e 'e2e/**'
```

The report says what it found: languages, code files in scope, test files (how many are unit
tests, how many use test doubles), design sources, and the state of the backlog.

## 2. Adopt the area

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

## 3. Model the riskiest stateful resource

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
claimed.

**Mark the failures.** The code you model already fails somewhere: a status value such as
`failed`, a worker that reports a crash, a lease that runs out. Tag the state that records it with
`"tags": "failure"` and list the events that report it in the root `_failure_events`
(`docs/machine-format.md`, "Failure idioms"); `machinery lint` warns about a state or event whose
name looks like a failure and is not marked. Each failure state and failure event then needs a
failure mode in `design.failure_modes`. The next `adopt --write` waives the ones that have none,
with the reason `unmodeled at adoption`, so the check still passes; when you write a failure mode
for one, its waiver becomes a `stale waiver` ERROR and you delete it. The list only shrinks.

## 4. Turn what is owed into stories, beside the ones you have

```sh
machinery owed list
machinery plan --apply
machinery waves
```

`plan --apply` adds one story per group of owed ids to the same backlog, with dependency edges
and a tier label. Your existing stories are untouched. To attach ids to a story you already have,
use `machinery plan --link <story> <id>...`. `waves` shows what can run in parallel.

## 5. Prove one story end to end

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

`owed sync --prune-only` removes exactly the ids the run proved. `verify` passes: everything is
proven or still owed. `story` is the gate for that one story; a consequential or critical story
would also need a fresh verifier (`docs/pm.md`).

## 6. Hold the line, then widen

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
failure modes in Markdown tables and your own trace markers in comments are read as they are,
through `import` and `trace.markers` in `machinery.yaml` (`docs/formats.md` sections 5.2.1, 5.4.1
and 8.1); the fixture `e2e/testdata/fixtures/library` is a small example.
