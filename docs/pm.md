# Planning and scheduling

The planner (the PM role) owns the backlog and the risk file. machinery bridges the design to the
backlog so that planning is data: what is owed becomes stories, the graph becomes waves, and a
story is accepted with the formality of its tier. The backlog tool is `bd`. The contract is
`docs/formats.md` sections 15 and 23.

## Tiers: formality follows risk

`design/risk.yaml` lists the areas that are not routine, with the reason:

| Tier | For | Adds |
|---|---|---|
| routine | Cheap to see and undo. | Generated transition cases, end-to-end evidence. |
| consequential | Authorization, money, audit, published contracts, migrations. | An abuse case per action; a fresh verifier who predicts before reading the proof. |
| critical | Races, lost or repeated work, structural rules. | A formal model (`docs/formal.md`). |

Choose the tier at planning. When unsure, the higher one. Lowering a tier later fails the
ratchet unless the risk file records it under `lowered:` with a reason.

## A new requirement: intake

A requirement that arrives while work runs changes the design before any story exists, and its
stories are then written all at once from what the design owes. machinery holds that order
(`docs/formats.md` section 15.7):

```
machinery intake new "<title>" --intent "<who needs what, and what is observed>" --refuses "<what must be refused>"
# change the design: model, machines, failure modes, tiers, a formal spec for a critical area
machinery lint                      # then machinery gen, and commit the design change
machinery intake status <id>        # each step done or missing, with the reason
machinery intake plan <id>          # the stories it would create
machinery intake plan <id> --apply  # all of them in one backlog call, owed, with the capstone
```

- `intake plan` refuses until the design changed, lints, is generated and committed, has its
  tiers, and every critical area it touches has a formal spec. It refuses while a wave is
  frozen: a contract change lands alone.
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
machinery waves --freeze       # pin the contract for the wave
machinery brief <story>        # what one worker needs, as pointers
machinery waves --thaw         # when the wave has landed
```

- Fix cycles and competing writers before dispatch. Two stories of one wave that write the same
  paths need an edge or a split. Stories declare what they write with a `machinery:scope` line.
- While a wave is frozen, a change to a design source fails stage `contract`. A contract change
  is its own story and lands alone.
- A brief is pointers and commands. Route by tier: the brief names the model class.

## Accepting a story

```
machinery proof --story <id> --post      # builder: post the proof to the story
machinery predict <id> --observation "..." --expect "..."   # verifier, before reading the proof
machinery accept <id> --verdict accept --observed "..." --close
machinery story <id>                     # the gate, by tier
```

Routine stories pass on evidence. Consequential and critical ones need a verifier who is not the
builder, a prediction for the same commit, and then the verdict. Critical ones also need their
formal model proven and its counterexamples replayed.

## Landing

Testing shifts right. Per change a builder runs the affected checks only
(`machinery run --affected`, `machinery verify --story`). The whole suite runs once, at the head
that is ready:

```
machinery landing --base main
```

It runs `check` with the ratchet, re-runs the formal tools against the committed results, runs
the full end-to-end suite and `verify`, and counts attempts. After two fix-forwards a failing
story goes back to the backlog. `machinery probe` is the measure of test quality: run it per
batch and before a release, or as the last step of the landing with `--probe`.
