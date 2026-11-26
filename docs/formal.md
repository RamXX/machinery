# Formal models

This guide shows how to give a critical area a formal model that cannot quietly lie. The contract
is `docs/formats.md` section 16; this page is the working order.

## When

Only for areas the risk file marks `critical` or `vital` (section 23): promises whose break harms
users and that tests rarely reach. Everything else is covered by generated transition cases and end-to-end
evidence. The area's `questions` pick the tool:

| You are asking | Tool |
|---|---|
| Which orders of events are legal? What if two callers act at once? What if work is lost or repeated? | TLA+ (`ordering`, `concurrency`, `recovery`) |
| Which shapes of data may exist? Who can reach what? | Alloy (`structure`) |

## The files

```
design/formal/
  order-payment.spec.yaml     the manifest: areas, runs, assumptions, mutations (you write it)
  Order.tla                   skeleton generated from the machine (never edit)
  OrderProps.tla              the properties (you write it)
  order-payment.result.json   written by machinery formal check (commit it)
  baseline.json               written by machinery formal baseline (commit it)
  traces/                     counterexamples (generated, commit them)
```

## The working order

```
machinery formal tools                      # once: fetch the pinned jars by checksum
machinery formal new order-payment --tool tla --machine Order --area order-payment
machinery formal gen                        # writes Order.tla from the machine
# write properties, reach predicates, assumptions, runs and mutations
machinery formal check order-payment        # runs TLC for real
machinery formal baseline                   # records what the spec promises
machinery check                             # the pure stage: needs no Java
machinery formal check --verify             # re-run, write nothing, compare with what is committed
```

## What the skeleton gives you

For a machine, one TLA+ action per transition case, named after the case id. `st[i]` is the state
of instance `i`; `fx[i][e]` counts how often effect `e` ran (0, 1, more). With the constant
`Atomic` true a caller's step reads and writes the state at once; with `Atomic` false it is split
into a begin and a commit, which is how races appear. Guards are not modeled: any branch may
fire.

For a Modelith model, one Alloy signature per entity and enum, one field per relationship with
the multiplicity of its cardinality, and facts for ownership and mirrored relationships.

## The five checks every model carries

1. **Reach.** Each passing run names states it must be able to reach. An unreachable one fails:
   the pass would be vacuous.
2. **Assumptions.** What you assume about the host is numbered data. An assumption is unproven
   until an obligation proves it, and unproven ones print in every report. The usual proof: a
   run with the assumption switched off (`expect: finding`), whose counterexample the real system
   refuses to follow.
3. **Mutations.** For each property, a plausible bug as a text swap that the property must
   catch. A property no mutation names is a finding. Past bug fixes become mutations (`origin`).
4. **Baseline.** A hash of every property, the fairness and the bounds. Changing one is called a
   weakening until you accept it with a reason, and the verifier sees the reason.
5. **Sync.** The manifest records the commit at which the sources were read. Later commits to
   those sources drift the spec until you classify them: no effect, model drift, promise change,
   or possible bug.

## Counterexamples are obligations

Every violation becomes a trace file and a `cx:<spec>:<run>` obligation. An end-to-end test
replays it against the real system and reports `replay: reproduced` or `replay: refuted` in its
receipt. The trace says which is expected:

| The run | Expected | If the replay says otherwise |
|---|---|---|
| A finding that negates an assumption | refuted | The assumption is false in the real system: a real bug. |
| A finding that records a known bug | reproduced | The bug is gone: set the run to `expect: pass`. |
| A passing run that was violated | undetermined | Reproduced: fix the code. Refuted: fix the model. |

Nobody weakens the property to make the run pass.

## Beyond model checking

A model check answers questions about the design: given these states and these steps, can two
callers both charge? It says nothing about the code that runs. The tests close part of that gap.
For the few paths that must never go wrong (a customer charged twice, an audit record lost, a
signature attributed to the wrong person) the risk file marks the area `vital`
(`docs/formats.md` section 23). A vital area keeps everything above and adds:

- **Properties.** Each states exactly what must hold, over inputs and stored state, and names the
  design ids it is about (`over`). Each is a proof obligation `pf:<area>:<name>`. Nothing in this
  version proves one: it prints as awaiting proof, is never owed and no test may claim it.
- **A core.** The files where the path lives and its entry points. Stage `arch` keeps the core
  small, free of the packages `constraints.cores.deny_imports` names, and the only implementation
  of the actions its properties are over.
- **Assumptions.** What the core takes on trust from its surroundings (the database enforces a
  unique key; the caller is authenticated), each with the obligation that is its evidence. While
  that evidence is owed the assumption prints as `assumed, unproven`, and a story of the area is
  not accepted.

`machinery gen` writes `design/gen/vital/<area>.json`: the properties, the core files with their
hashes, the assumptions and the status of every id, the hand-off a machine-checked proof of the
core will start from. Until a later version requires that proof, every report that calls an id of
a vital property proven says how far the proof goes: `(by test and model; proof awaited)`.

`machinery vital candidates` lists the ids that look vital and are in no vital area, from the
names and texts of the design alone. Make each one vital, or dismiss it under `not_vital` in the
risk file with the reason.
