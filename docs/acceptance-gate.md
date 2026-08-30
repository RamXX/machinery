# Milestone acceptance and the Ga-accept gate

`Gb-plan` holds the SHAPE of the build plan: milestones are marked, numbered uniquely, the
walking skeleton comes first, every milestone states a definition of done, and the skeleton's
DoD cites a committed oracle id. Nothing in the tool held the other half of a plan's life,
which is a milestone being **discharged**. A milestone was closed by assertion, in a chat
message or a checkbox somewhere, and the assertion was checked by nobody. Whatever the DoD
said, the closure was worth exactly as much as the memory of the person who declared it.

`Ga-accept` closes that hole with machinery's standing pattern: **attested judgment, committed
evidence, deterministic binding**. The review itself is judgment, exactly like a Gk external
checker's verdict or the semantics of a Gx enforcement row. What the gate proves is that the
review happened, on this commit, against these obligations, and that its verdict says what the
plan claims it says.

## The protocol

1. **The review runs on one commit.** Whoever accepts a milestone (a human owner, a reviewing
   agent, both) reads the milestone's DoD and checks it against the tree at one commit.
2. **The evidence is committed**, as `design/acceptance/M<n>.yaml`, one file per milestone.
3. **Only then is the milestone marked closed** in the build plan, with a `Status: closed` line
   in its block.
4. **CI runs the gate with the commit**: `machinery check design --commit $(git rev-parse HEAD)`.

Steps 2 and 3 are deliberately in that order. A closed milestone with no evidence behind it is
an ERROR, so the marker cannot land first and be reconciled later.

## The marker

The milestone block gains one optional status line beside its `DoD:` line:

```markdown
**M3 - Settlement slice.** Payment lifecycle end to end.
DoD: all 14 Payment oracle rows green by stable id (PAY-3f9c21 through PAY-b70e44),
`pay-once` property-tested, contract tests green, G4-import clean.
Status: closed
```

`Status: open` is the default and may be written explicitly. Any other value is a Gb ERROR:
reading an unrecognized value as "not closed" would silently disarm this gate, which is exactly
the failure mode the gate exists to prevent. The line tolerates the same decorations the
milestone markers use (`- Status: closed`, `**Status:** closed`).

## The evidence

```yaml
milestone: 3                     # integer; a milestone the build plan declares
commit: 9f3c1a2b7d4e5f60718293a4b5c6d7e8f9012345   # the commit the review was performed on
verdict: ACCEPTED                # exactly ACCEPTED or REJECTED, upper case
dod_ids:                         # every committed oracle id this milestone's DoD cites
  - PAY-3f9c21
  - PAY-b70e44
attestations:                    # what the review checked by judgment
  - integration tests run against the real ledger; no mocks below the boundary
  - every guard conjunction has its falsifying-clause test
  - the failure catalog's residual states each have an operator signal
findings:                        # may be empty; the key is required
  - retry backoff is fixed, not exponential; tracked as a follow-up, not blocking
reviewer: milestone acceptance review, conductor + owner sign-off
date: 2026-08-27
```

One file per milestone, and no round numbers: git history is the record of prior attempts. A
`README.md` or `index.md` in the directory is human navigation and is exempt; anything else in
`design/acceptance/` is a finding, because a gate that quietly ignores an unrecognized artifact
teaches people to leave unrecognized artifacts.

## What the gate checks

Ga activates automatically once `design/acceptance/` exists, or once any milestone carries the
closed marker. Either one alone is a claim that a milestone is being discharged. An explicit
`--gate ga` forces it, and forcing it on a design with neither is an error naming both missing
things, the way forcing `gp`, `gi`, or `gn` is.

- **Closure has evidence.** Every closed milestone has an acceptance file whose verdict is
  ACCEPTED. A closed milestone with no file, or with a REJECTED file, is an ERROR. A REJECTED
  file on an OPEN milestone is the normal state of a review that found problems, and passes.
- **The evidence is well formed.** It parses, carries every field, names a milestone the build
  plan declares, and matches its own file name. An ACCEPTED verdict with no attestations attests
  nothing and fails: the attestation lines are what make the judgment reviewable later.
- **The review looked at the right obligations.** `dod_ids` must list every committed oracle id
  (test id or stable id, from `machines/*.oracle.md` plus `formal/Policy.oracle.md` and
  `formal/Isolation.oracle.md` when they exist) that the milestone's DoD cites whole-token, at
  or after the `DoD:` token. This is the same discipline as Gk binding evidence to a projection
  by `input_hash`: it does not prove the review was good, it proves the review was about this
  work.
- **The evidence binds to the commit.** With `--commit <sha>` (or `MACHINERY_COMMIT`; the flag
  wins), every accepted closure must name that commit: an exact match, or either value an
  unambiguous prefix of the other of at least 7 characters (git's own abbreviation floor).
  Without a commit the gate prints a non-blocking note that the binding was not checked. CI is
  expected to pass it; the stop-time hook never does, because mid-turn the commit under review
  does not exist yet.
- **Milestone numbers are unambiguous.** Evidence is keyed by number alone, so a number declared
  in two plan-bearing documents (the manifest root and a shard, or two shards) is an ERROR
  naming both. Milestone numbers are global across a sharded design.

Everything else is attested: whether the DoD was really met, whether the attestations are true,
whether the findings list is complete. Ga is the record that someone with a name looked, on a
commit that is written down, at the obligations the plan itself declared.

That last judgment now has a committed home of its own. `Gv-attest` generalizes this file's
pattern to every attested gate half, including this one, as the `ga.review-quality` claim in
`design/attestations.yaml` (`docs/attestation-evidence.md`). The `attestations:` list here stays
exactly as it is: it is scoped to one milestone review at one commit, while a Gv row is the
standing judgment over an artifact, invalidated the moment that artifact's bytes move. Where an
entry in this list restates a Class C claim, write the claim id and carry the detail in
`attestations.yaml`, so the acceptance file reads as this milestone's own findings rather than a
second copy of a standing judgment.

## CI

```yaml
- name: machinery gates
  run: machinery check design --impl impl --commit "${{ github.sha }}"
```

or, equivalently, `MACHINERY_COMMIT=$GITHUB_SHA machinery check design --impl impl`. On a
pull request the commit under review is the head commit, which is what an acceptance review runs
against; a merge commit that did not exist when the review ran will not bind, and that is the
intended behavior, not a bug to work around.

## Where this sits in the suite

Ga runs after Gb in the default suite ordering (`gm,gs,gu,gp,gi,gn,gc,g2,g3,gd,gl,gx,gk,gb,ge,ga,gj,gv,g4,gt,g5`):
the plan's shape is settled before its discharge is judged. It is a build-time gate, not a
design-phase one; a design that has closed no milestone never sees it.
