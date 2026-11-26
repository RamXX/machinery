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

## From owed to stories

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
