# Attestation evidence and the Gv-attest gate

Every machinery gate splits its domain in two. The deterministic half is what the tool
checks: the contract parses, the oracle is fresh, the ids resolve, the hash binds. The
attested half is what a reviewer judges: whether a guard's semantics actually enforce the
invariant it names, whether an interface contract is the right one, whether BUILD.md is
really buildable with zero context.

`Ga-accept` gave one attested half a committed record. Every other one lived in a
conversation. SKILL.md says "include the verdicts in your summary", and the summary
scrolls away, so months later the standing answer to "who judged this, and is the
judgment still current?" was somebody's memory.

`Gv-attest` closes that with the same pattern Ga uses: **attested judgment, committed
evidence, deterministic binding**. The judgment stays judgment. What becomes deterministic
is the bookkeeping around it, which is the part that was never anybody's job.

## The evidence

One file per design, `design/attestations.yaml`. Rows are keyed by a claim id from a
closed vocabulary the gate owns.

```yaml
attestation_version: 1
attestations:
  - claim: g2.interface-contract-rightness
    attestor: Ramiro Salas
    date: 2026-08-30
    note: the payment edge's error list was extended after the PSP retry review
    covers:
      - path: ARCHITECTURE.md
        hash: sha256:3f786850e387550fdab836ed7e6dc881de23001b42caa1de5c1bd8e56c0a1a3b
  - claim: g3.guard-semantics
    attestor: Ramiro Salas
    date: 2026-08-30
    covers:
      - path: machines/Deal.machine.json
        hash: sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae
      - path: machines/Payment.machine.json
        hash: sha256:fcde2b2edba56bf408601fb721fe9b5c338d10ee429ea04fae5511b68fbf8fb9
```

| Field | Meaning |
|---|---|
| `attestation_version` | the integer `1`. |
| `claim` | one id from the vocabulary below. Unknown ids are an ERROR: an invented claim records diligence instead of being held to it. |
| `attestor` | who or what made the judgment. An attestation without an attestor attributes nothing, so an empty value is an ERROR. This is where the named-owner rule lands, exactly as Ga's `reviewer` field does. |
| `date` | `YYYY-MM-DD`, a real calendar date. |
| `covers` | a non-empty list of `{path, hash}`. `path` is design-relative (never absolute, never climbing out of the design). `hash` is `sha256:<64 lower-case hex>` over the file's bytes at attestation time. |
| `note` | optional free prose: what the judgment turned on. |

A claim appears at most once. Git history is the record of prior attestations, so a
re-attestation edits the row rather than appending a second one, the same rule Ga uses for
milestone rounds and Gj for verdicts.

## Getting the hash right

```bash
machinery attest design/ARCHITECTURE.md design/machines/Deal.machine.json
sha256:3f786850e387550fdab836ed7e6dc881de23001b42caa1de5c1bd8e56c0a1a3b  design/ARCHITECTURE.md
sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae  design/machines/Deal.machine.json
```

Paste the printed value into the row (the `path` stays design-relative). The shell
equivalent is `shasum -a 256 <path>` (or `sha256sum <path>`) with the `sha256:` prefix
added by hand; the command exists so nobody has to.

`machinery attest --claims` prints the vocabulary from the binary, so an attestor never
works from a transcription that has drifted.

## What the gate checks

`Gv-attest` activates on the artifact, like `Ga` and `Gj`: no `attestations.yaml`, no
gate. A design that has not adopted the record is not failed for not having adopted it.
Once the file exists, the gate verifies:

- the file parses, carries `attestation_version: 1`, and holds a non-empty
  `attestations` list (an empty record is a failure, not a pass);
- every `claim` resolves to the closed vocabulary, and no claim appears twice;
- every row names an `attestor` and a real date;
- every covered `path` is design-relative and names a file the design carries;
- every `hash` matches that file's current bytes.

The last one is the point of the whole design. A covered artifact edited after the
attestation makes the row **STALE**, and STALE is a blocking ERROR:

```
== Gv-attest  attestation evidence ==
  ERROR  attestations.yaml: g2.interface-contract-rightness is STALE: ARCHITECTURE.md
         changed since it was attested (recorded sha256:3f7868..., current sha256:9b71d2...);
         re-read the artifact, judge it again, and update the row with
         'machinery attest design/ARCHITECTURE.md'
  checked: 4 attested claims, 5 covered artifacts current, 6 claims owed
```

It is an ERROR rather than a DRIFT deliberately. DRIFT means a generated artifact fell
behind its source and is fixed by regenerating; nothing regenerates a judgment. The remedy
is a person reading the changed artifact and deciding again, and calling it DRIFT would
misdescribe that.

What the gate never checks is whether a judgment is TRUE. It cannot, and pretending
otherwise is what the split exists to prevent.

## Coverage: warn, not error

A claim whose artifact exists is **owed**: the Gate 2 claims once `ARCHITECTURE.md`
exists, the Gate 3 claims once `machines/*.machine.json` do, and so on down the table
below. An owed claim with no row is a WARN naming the claim and the artifact that made it
owed.

That is a deliberate departure from machinery's usual "absence is an ERROR" posture, and
the reason is adoption. The evidence file is opt-in and is adopted in the middle of a
design's life. If the first commit of it failed the gate for every claim not yet
re-judged, adopting the record would cost more than not adopting it, which guarantees the
attested halves stay in conversation forever. So a partial record is allowed to be
partial and says so out loud, while a record that is WRONG blocks: an unknown claim, an
unattributed row, a dangling referent, a stale hash. A misleading record is worse than a
missing one. The absence rule still bites where it must, at the file level: an evidence
file with no rows is an ERROR.

## The vocabulary

Closed, and enumerated in `internal/gates/attest.go` from the LLM-attested blocks of
`skills/machinery/SKILL.md` plus the attestation list in `agents/machinery-fsm-author.md`.
Adding an attested half to SKILL.md means adding its id here; that coupling is the point.

| Claim id | The judgment | Owed once |
|---|---|---|
| `g2.action-ownership` | every Modelith action maps to an owning component (checked instead when the design authors the action-ownership table) | `ARCHITECTURE.md` |
| `g2.interface-contract-rightness` | each interface contract is the RIGHT one: the shape matches what the code will exchange, the error list is exhaustive, the idempotency claim survives a retry | `ARCHITECTURE.md` |
| `g2.placement-rightness` | each persistence-and-placement decision is the RIGHT one | `ARCHITECTURE.md` |
| `g2.adoption-closure-discovery` | the adoption closure is fully DISCOVERED: a member nobody declared is invisible to the gate | `ARCHITECTURE.md` |
| `g2.event-contract-completeness` | the event-contract table covers every cross-component event and the dependency declaration itself is complete | `ARCHITECTURE.md` |
| `g2.nfr-content` | the NFR record's CONTENT is true (presence and topic coverage are checked; the posture is judgment) | `ARCHITECTURE.md` |
| `g3.guard-semantics` | each guard's semantics actually enforce the invariant it names | `machines/*.machine.json` |
| `g3.invariant-enforcement` | every Modelith invariant is guarded or structurally impossible; any that is neither is listed | `machines/*.machine.json` |
| `g3.residual-transitions` | every C4 dependency failure has its residual transition, reclassified by its mitigation rather than deleted | `machines/*.machine.json` |
| `g3.event-redelivery` | every consumed external event has its event-contract row and a redelivery story | `machines/*.machine.json` |
| `gt.conformance-test-shape` | a wholesale-conformance test parses the committed oracle table and asserts, per row, the next state AND the expected actions | `BUILD.md` |
| `g4.zero-context` | a coding agent with no prior context could build the system from `BUILD.md` alone (per shard, when sharded) | `BUILD.md` |
| `g4.standin-coverage` | isolated child only: the neighbor stand-in section exists, every neighboring boundary has a stand-in held to its oracle, and the environment recipe is self-contained | a `Neighbor stand-ins` section in the build document |
| `g4.pack-event-discipline` | pack child only: the implementation carries no emitter or handler for an event absent from its pack | `pack/` |
| `ga.review-quality` | the milestone reviewer judged WELL: the DoD was really met, the acceptance file's attestations are true, and its findings list is complete | `acceptance/` |

## Relationship to Ga-accept

`design/acceptance/M<n>.yaml` keeps its free-prose `attestations:` list, which is what the
milestone review checked by judgment on that commit, and nothing about existing acceptance
files changes. The two records answer different questions: Ga's list is scoped to one
milestone review at one commit, while a `Gv` row is the standing judgment on one artifact,
invalidated the moment that artifact moves.

Where an acceptance attestation string restates a Class C claim, prefer the id: write
`ga.review-quality` (or the specific claim) as the `attestations:` entry and carry the
detail in `attestations.yaml`. The acceptance file then reads as the milestone's own
findings rather than as a second copy of a standing judgment.

## Relationship to the PR-checklist idea

`docs/brownfield-team-guide.md` section 6 previously proposed carrying the attested lists
as a checklist in every design PR description, checked by a named reviewer, invalidated by
any change to the artifact it covered. That idea is right and this gate is its committed
form: the checklist becomes rows, the named reviewer becomes `attestor`, and "invalidated
by any change to the artifact" becomes the content hash, which a PR description could
never enforce. Use the file; the checklist is superseded.
