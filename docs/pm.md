# Planning and scheduling

The planner owns the backlog: intake, tiers, waves and landing. It never builds, and it writes no
design source: it chooses the tiers and the modeler writes them (`docs/roles.md` has every role,
what the tool verifies about each, and the reason for each rule). machinery bridges the design to
the backlog so that planning is data: what is owed becomes stories, the graph becomes waves, and a
story is accepted with the formality of its tier. The backlog tool is `bd`. The contract is
`docs/formats.md` sections 15 and 23.

## Tiers: formality follows risk

`design/risk.yaml` lists the areas that are not routine, with the reason:

| Tier | For | Adds |
|---|---|---|
| routine | Cheap to see and undo. | Generated transition cases, end-to-end evidence. |
| consequential | Authorization, money, audit, published contracts, migrations. | An abuse case per action; a fresh verifier who predicts before reading the proof. |
| critical | Races, lost or repeated work, structural rules. | A formal model (`docs/formal.md`). |
| vital | A path that must never go wrong: a double charge, a lost audit record, a signature on the wrong person. | The formal model, plus exact properties over its ids, a core shaped for a later machine-checked proof, and its assumptions with evidence (`docs/formal.md`, "Beyond model checking"). |

Choose the tier at planning. When unsure, the higher one. Keep vital for the few paths whose
break cannot be undone: each vital area is a promise that its core will one day be proven, and
its proof obligations are counted until then. At planning, run `machinery vital candidates`: it
lists the ids that look vital from the design alone. Make each one vital, or dismiss it under
`not_vital` in the risk file with the reason; the list only shrinks. Lowering a tier later fails the
ratchet unless the risk file records it under `lowered:` with a reason, and so does a vital area
that states less: a property, an assumption, a core glob or entry removed or reduced, or a vital id
dismissed. Record each as `{area, what, reason}`, `what` being the ratchet finding's text.

## Statuses: proposed and retired

An id of the design is owed or proven, unless `design/status.yaml` says otherwise
(`docs/formats.md` section 4.1):

```yaml
version: 1
proposed:
  - {id: "inv:refund-window", note: "candidate; no delivery commitment"}
retired:
  - {id: "act:Order.gift", reason: "removed in the 2.0 contract", since: "v2.0"}
```

- A proposed id is a candidate: no obligation, no story, and no code or test may claim it. An
  intake lists it apart and plans none of it. Commit it by removing its entry.
- A retired id is no longer promised; the entry stays as the record. Un-retire it by removing
  the entry.
- The ratchet refuses moving an owed or proven id to proposed, and reports retiring a proven one.

## A new requirement: intake

A requirement that arrives while work runs changes the design before any story exists, and its
stories are then written all at once from what the design owes. machinery holds that order
(`docs/formats.md` section 15.7):

```
machinery intake new "<title>" --intent "<who needs what, and what is observed>" --refuses "<what must be refused>"
# change the design: model, machines, failure modes, tiers, a formal spec for a critical area
machinery lint                      # then machinery gen, and commit the design change
machinery intake status <id>        # each step done or missing, with the reason
machinery review design <id> --verdict approved   # the checker, in a fresh context
machinery intake plan <id>          # the stories it would create
machinery intake plan <id> --apply  # all of them in one backlog call, owed, with the capstone
```

- `intake plan` refuses until the design changed, lints, is generated and committed, has its
  tiers, and every critical area it touches has a formal spec. `--apply` also waits for the
  checker's approved design review of the change (`review.design` in `machinery.yaml`). It
  refuses while a wave is frozen: a contract change lands alone.
- The design story, `Land the design: <title>`, comes first when the design changed: it is the
  one story that may change design sources (stage `scope`), its gate is the design sound,
  generated and approved, and every other story of the intake waits for it.
- The capstone, `Demonstrate: <title>`, closes last. It shows the intent and each refusal on
  the running system through its real interface, records the commands as `machinery:demo`
  lines, and is accepted by a fresh verifier.
- A requirement the design already covers: append `machinery:no-design-change <reason>` to the
  item's notes; only the capstone is planned.
- `machinery check` warns about an intake whose design changed with nothing planned, and about a
  delivered intake left open. `machinery waves` fails on a story labelled `machinery` that
  names no ids.

## From owed to stories

Obligations owed without a requirement behind them (a new project's skeleton, an adoption):

```
machinery owed list            # what is unproven
machinery plan                 # the stories it would create
machinery plan --apply         # create them in the backlog, with edges
machinery plan --link <story> <id>...   # attach ids to a story that already exists
```

An edge means one story needs a named output of another: a proof story of a critical area needs
the area's formal model; a failure-mode story needs the invoke path it is induced on.

## Waves

```
machinery waves                # layers of the graph: one wave runs in parallel
machinery review backlog --verdict approved   # the checker, in a fresh context
machinery waves --freeze       # pin the contract for the wave
machinery brief <story>        # what one worker needs, as pointers
machinery waves --thaw         # when the wave has landed
```

- Fix cycles and competing writers before dispatch. Two stories of one wave that write the same
  paths need an edge or a split. Stories declare what they write with a `machinery:scope` line,
  and stage `scope` (`machinery check --base <base> --story <id>`, run by the builder's brief and
  by the commit hook on a branch named after the story) holds each story to it.
- While a wave is frozen, a change to a design source fails stage `contract`. A contract change
  is its own story and lands alone.
- A brief is pointers and commands, and it is all a worker gets: context pasted beside it carries
  the planner's assumptions to the builder and to the verifier alike.
- Routing: the `route:` line of a wave and of a brief names the class of model for the story's
  tier (`routing` in `machinery.yaml`). By default a builder runs on a standard class and a
  verifier on one at least as strong; planner, modeler, checker and delegate run on the strongest.

## Before a wave is dispatched

The tool checks the graph. It cannot check the plan, so the planner reviews its own backlog and
then has the checker review it (`agents/machinery-checker.md`), before `machinery waves --freeze`:

- The first wave is a walking skeleton: one thin path through every layer, so integration fails
  on the first day and not at the landing.
- Each story is a vertical slice with something to observe through the real interface. A story
  for one layer has nothing to prove until another story lands.
- Each intake has its capstone, and each refusal of the intake a story that proves it.
- A premise several stories share is settled, as a decision or in a formal run, before they start.

The checker answers pass or fail with at most five issues, each with its general rule and scope:
fix the class, not the instance, and ask again. It is also called after every design change,
before `machinery intake plan <id> --apply`, to read the change against its requirement for
omission, invention and contradiction; and periodically for drift, with `machinery assess` on the
project itself. The checker records each review with `machinery review`: `waves --freeze` waits
for an approved backlog review for the contract and the first wave, and `intake plan --apply` for
an approved design review, each from another session and agent than the planner's
(`review.backlog` and `review.design` in `machinery.yaml`; an adopted project may switch them
off, and the ratchet reports it).

## Accepting a story

```
machinery proof --story <id> --post      # builder: post the proof to the story
machinery predict <id> --observation "..." --expect "..."   # verifier, before reading the proof
machinery accept <id> --verdict accept --observed "..." --close
machinery story <id>                     # the gate, by tier
```

Routine stories pass on evidence. Consequential, critical and vital ones need a verifier who is
not the builder, a prediction for the same commit, and then the verdict. Critical and vital ones
also need their formal model proven and its counterexamples replayed; vital ones also need the
evidence of every assumption proven. A proof obligation never blocks a story and never counts as
proven. When the harness sets `MACHINERY_SESSION` (the Claude Code plugin's hooks do, see
`docs/hooks.md`), the proof, the prediction and the verdict record it, and the gate refuses a
verifier who predicted from the builder's session and agent. Subagents share their parent's
session, so a worker exports the `MACHINERY_AGENT` its brief names; the same session with
different agents passes.

The builder posts the proof with what it does not prove and what it learned:
`--limit "<claim still unproven>"`, `--learning "<rule for the next story>"`,
`--observation "<bug|security|debt|perf|doc>: <seen outside the story>"`, each repeatable.

## Decisions, observations, learnings

```
machinery decide "<title>" --why "<reason; alternatives>" --ids <id>,<pattern>   # a decision item
machinery decide "<title>" --why "..." --supersedes <decision id>               # replaces one
machinery decisions --ids <id>               # the current decisions that govern it
machinery decisions --open-vetoes            # agent decisions the owner has not confirmed
machinery observations --file                # one backlog item per observation, filed once
machinery learnings --since <last plan>      # read before the next plan
```

- A decision an agent took within its remit is `--vetoable`; the owner confirms it with
  `bd update <id> --append-notes "machinery:confirmed <name>"` or supersedes it.
- `machinery brief` lists the decisions that govern a story's ids, as pointers.

## Questions for the owner (experimental)

```
machinery ask "<question>" --option "<a>" --option "<b>" --recommend <n> --default "<if nobody answers>" --about <topic>
machinery questions                          # what waits for the owner, and why
machinery answer <id> --option <n> --reason "..."                                   # the owner
machinery answer <id> --delegate --clause <clause> --option <n>|--hold|--refuse --reason "..."   # an agent, under the owner's policy
```

- Ask instead of stalling: the question carries your recommendation and what happens if nobody
  answers, and you go on with other work. Topics `publish`, `payment`, `destructive`, `method`
  and `delegate-policy` are reserved: only a clause that names them may answer.
- `delegate.policy` names the owner's policy file, outside the project. An answer on the owner's
  behalf is a vetoable decision; when a clause does not permit it, the only answer left is
  `--hold`. Amounts under a clause add up against its limit.
- `machinery intake plan` reminds you of the learnings posted since the last plan.
- The delegate (`agents/machinery-delegate.md`) is called sparingly, for a question that holds up
  a wave while the owner is away. The policy file is the owner's; no agent writes it.

## Landing

Testing shifts right. Per change a builder runs the affected checks only
(`machinery run --affected`, `machinery verify --story`). The whole suite runs once, at the head
that is ready:

```
machinery landing --base main
```

It runs `check` with the ratchet, re-runs the formal tools against the committed results, runs
the full end-to-end suite and `verify`, and counts attempts. After two fix-forwards a failing
story goes back to the backlog.

A failed landing marks the integration branch red (`machinery status` says so first). While it
is red, `machinery accept --close` and a landing at another head refuse: repair it (`machinery landing --repair`,
`machinery accept <id> ... --close --repair` for the story that repairs it), or revert and run
`machinery landing --clear-red --reason "<what was reverted>"`. `machinery probe` is the measure of test quality: run it per
batch and before a release, or as the last step of the landing with `--probe`.

## When every story is closed

Closed stories are not the goal. An intake is delivered when its capstone is accepted and the
landing is green; `machinery check` then warns until it is closed. When every story is closed and
the owner's goal is still not observed on the running system, the plan was wrong, not the stories:
read `machinery learnings`, file the observations, and start a new intake for what is missing.
Reopening or patching closed stories would change the system without changing the design.
