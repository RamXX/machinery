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
   A local run needs no flag: inside a git repository the gate defaults to that repository's HEAD.

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
- **The review looked at the right obligations, in both directions.** `dod_ids` must list every
  committed oracle id (test id or stable id, from `machines/*.oracle.md` plus
  `formal/Policy.oracle.md` and `formal/Isolation.oracle.md` when they exist) that the milestone's
  DoD cites whole-token, at or after the `DoD:` token. And every id `dod_ids` lists must itself
  resolve to a committed oracle id: an entry that resolves to nothing (a typo, or an id a
  regeneration renamed out from under the file) is an ERROR naming it. Only the forward rule was
  checked at first, so a list could be padded with ids that existed nowhere and still read as
  coverage. This is the same discipline as Gk binding evidence to a projection by `input_hash`: it
  does not prove the review was good, it proves the review was about this work.
- **The evidence binds to the commit**, under one of two rules. Which one applies depends on where
  the commit under review came from, and the `checked:` line always names it, so the rule in force
  is never inferred from the absence of a note. See "The two binding modes" below.
- **Milestone numbers are unambiguous.** Evidence is keyed by number alone. Root `BUILD.md` is the
  sole plan and acceptance authority; bounded `BUILD/*.md` execution packets cannot declare
  milestones, so a packet cannot silently introduce a second acceptance target.

## The two binding modes

The evidence names the commit its review ran on. What the gate can fairly demand of that name
depends on whether a caller told the gate which commit is under review, or the gate had to go
find one, so there are two rules and the `checked:` line always says which is in force.

**Explicit (identity).** `--commit <sha>`, or `MACHINERY_COMMIT` when the flag is absent. The
caller has named the one commit this closure is being judged against, so the evidence must name
it too: an exact match, or either value an unambiguous prefix of the other of at least 7
characters (git's own abbreviation floor). This is CI's contract. On a pull request the commit
under review is the head commit, which is what an acceptance review runs against; a merge commit
that did not exist when the review ran will not bind, and that is the intended behavior, not a
bug to work around. The `checked:` line reads `commit under review supplied by --commit or
MACHINERY_COMMIT; evidence commit bound by identity`.

**Derived (ancestry).** No flag and no environment variable, and the design directory sits inside
a git repository. The gate defaults the commit to `git rev-parse HEAD` of THAT repository,
resolved from the design path rather than the shell's working directory, so running `machinery
check some/other/design` from an unrelated checkout can never bind a design to a history that is
not its own. The evidence commit must then

1. resolve to a commit that repository holds (`git rev-parse --verify`), and
2. be reachable from HEAD, equality included (`git merge-base --is-ancestor`).

Either failure is an ERROR. The `checked:` line reads `commit under review derived from git HEAD
of the repository holding the design; evidence commit bound by ancestry`.

Ancestry, not identity, is the honest question here, and the difference is not a softening. When
a caller names a commit, identity asks "was the review run on exactly this commit"; when nobody
names one, the same question has a guaranteed wrong answer, because the commit that ADDS the
evidence file already has a different sha than the commit the evidence names. An identity rule on
a derived commit would go red on the very next commit and stay red for the life of the milestone,
which teaches people to turn the gate off. What ancestry still catches is exactly what the old
note tier let through: a sha with a typo in it, a fabricated one, and one from a branch this
history never took. Those are caught now on every local run and at stop time, without a flag
anyone has to remember.

**Neither.** Outside a git repository, or with no usable git, the binding degrades to a
non-blocking note naming what was not checked, never to a silent pass. CI is expected to pass the
reviewed commit; the note says so.

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

or, equivalently, `MACHINERY_COMMIT=$GITHUB_SHA machinery check design --impl impl`. Passing the
commit is what puts CI in the explicit mode, and the identity rule there is deliberate: see "The
two binding modes" above. A CI job that passes no commit falls into the derived mode against
whatever the runner checked out, which is a weaker claim than CI should be making, so pass it.

## Where this sits in the suite

Ga runs after Gb in the default suite ordering (`gm,gs,gu,gp,gi,gn,gc,g2,g3,gd,gl,gx,gk,gb,ge,ga,gj,gv,g4,gt,g5`):
the plan's shape is settled before its discharge is judged. It is a build-time gate, not a
design-phase one; a design that has closed no milestone never sees it.
