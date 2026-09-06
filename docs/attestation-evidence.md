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
attestation_version: 2
attestations:
  - claim: g2.interface-contract-rightness
    kind: plan
    attestor: Ramiro Salas
    date: 2026-08-30
    note: the payment edge's error list was extended after the PSP retry review
    covers:
      - path: ARCHITECTURE.md
        hash: sha256:3f786850e387550fdab836ed7e6dc881de23001b42caa1de5c1bd8e56c0a1a3b
  - claim: g3.guard-semantics
    kind: plan
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
| `attestation_version` | the integer `2`; legacy `1` has the migration rules below. |
| `claim` | one id from the vocabulary below. Unknown ids are an ERROR: an invented claim records diligence instead of being held to it. |
| `kind` | `plan`, `current`, or `historical`, according to the classification below. |
| `attestor` | who or what made the judgment. An attestation without an attestor attributes nothing, so an empty value is an ERROR. This is where the named-owner rule lands, exactly as Ga's `reviewer` field does. |
| `date` | `YYYY-MM-DD`, a real calendar date. |
| `covers` | a non-empty list of `{path, hash}`. `path` is design-relative (never absolute, never climbing out of the design). `hash` is `sha256:<64 lower-case hex>` over the file's bytes at attestation time. |
| `note` | optional free prose: what the judgment turned on. |
| `implementation` | required only for `current`; forbidden for other kinds. The full-root manifest described below. |

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

## Generate a reviewed row

After reviewing the actual implementation and tests, generate a complete v2 document:

```bash
machinery attest --design design --impl src \
  --claim gt.conformance-test-shape --kind current \
  --attestor 'Reviewer name' --date 2026-09-03 --note 'What was reviewed'
machinery check design --impl src --gate gv
```

Generation requires an explicit claim, kind, attestor and real calendar date.
For `plan` or `historical`, omit `--impl`; it is forbidden for those kinds.
Generation flags cannot be combined with explicit-file hashing or `--claims`.
The command does not run tests, infer the review, sign a receipt, or write the
evidence file. Inspect its document and merge the row into the record, replacing
an older row for that claim. Do not redirect over the live record being read;
use a separate output file outside the reviewed root and merge later.

Discovery, validation, final unchanged checks and owned cleanup all finish before
generation emits stdout. Failure returns exit 1 with no document bytes. A sink
write failure also returns nonzero, but an arbitrary pipe can accept partial
bytes: output writes are not an atomic filesystem transaction.

Hashes bind the observed files and scope; they do not prove tests ran, reviewer identity, or judgment correctness. Top-level Git administration and the exact Machinery attestation record are excluded; applications that use them as runtime inputs are outside this review boundary.

## Closed v2 kinds and grammar

The six `g2.*` claims, four `g3.*` claims, and `g4.zero-context` allow only `plan`.
`gt.conformance-test-shape`, `g4.pack-event-discipline`, and `g4.standin-coverage`
allow `current`, or an explicitly limited `plan` that leaves current evidence
missing (a warning). `ga.review-quality` allows only `historical` and never
approves today's implementation. Existing Ga milestone semantics are unchanged.

Root keys are `attestation_version`, `attestations`, and optional string
`_comment`. Row keys are the fields above, plus optional string `_comment`.
Duplicate or unknown keys, aliases, merges, nulls and wrong scalar types are
errors. Covers are closed `{path, hash, optional string _comment}` mappings: unique portable ASCII
design-relative paths, no casefold aliases, no self-reference to
`attestations.yaml`, and lowercase SHA256 hashes. Every required design subject
must be covered, not just any file. Version 2 requires an explicit kind.

## The current implementation manifest

`implementation` has exactly `root`, `policy`, `entries`, and `hash`.
`root` is the normalized slash-separated lexical locator from the logical design
root to the supplied `--impl` root, for example `../src`, `.`, or `src`. Leading
`..` components are allowed only in this locator. It must match the invocation;
it grants no authority to open a different root from the receipt. Equal roots,
disjoint roots, implementation inside design, and design inside implementation
are supported while retaining the original design generation.

`policy` is exactly `full-root-v1`. The inventory includes the root `.` directory,
every other directory (including empty ones), and every regular file: hidden,
ignored, generated, vendor, configuration, documentation and test inputs alike.
There are no include/exclude, extension, gitignore, or subtree selectors.
Only top-level `.git` (a real directory or regular gitfile) and the exact
Machinery design `attestations.yaml` are excluded. An unrelated file of that
name remains an input. Nested `.git` is unsupported metadata and fails closed.
Evidence is still bound by the retained design snapshot during each invocation;
its exclusion only prevents persistent receipt self-reference.

Entries are sorted by raw ASCII path bytes and use portable relative paths.
Directory entries contain exactly `{path, type: directory, mode}`. File entries
contain exactly `{path, type: file, mode, size, hash}`. Modes are four-character
octal permission strings `0000` through `0777`; size is a nonnegative integer;
file hashes use `sha256:<64 lowercase hexadecimal digits>`. Parents must precede
children. Duplicate/casefold paths, file/descendant conflicts, hardlink identity
aliases, symlinks and special files are rejected. Root and file identities are
held and checked; copying cannot erase an original alias or replacement.

The scope hash is SHA256 over this exact ASCII stream, ending every line with
LF. Fields below are separated by one TAB, without YAML quoting:

```text
machinery-attestation-scope-v1
root<TAB><root locator>
policy<TAB>full-root-v1
exclude<TAB>vcs-root:.git
exclude<TAB>evidence:<implementation-relative exact evidence path, or none>
directory<TAB><path><TAB><mode>
file<TAB><path><TAB><mode><TAB><decimal size><TAB>sha256:<hex>
```

The final two line forms repeat in inventory order. The evidence descriptor is
fixed by root topology even if the evidence file does not yet exist. The gate
checks manifest shape and its own digest, the independently acquired inventory
in both directions, then modes, sizes and hashes. Added and removed paths are
stale evidence, even when all listed files still match. No current-review count
is published until all retained-root checks and cleanup complete successfully.

## Fixed evidence limits

The complete logical inventory, including any retained design overlay, has one
budget: 100,000 entries, directory depth 64 (root depth 0), 1 GiB per regular
file, and 8 GiB aggregate regular-file bytes. Bounded traversal/copy rejects an
excess rather than truncating or resetting the budget for each directory or
overlay. Symlinks and special files are rejected before opening their contents.

The YAML document, each required design cover, and each file in legacy explicit
hash mode are limited to 16 MiB. Generated serialized YAML must fit 16 MiB too;
independently generated rows must still fit the same bound when merged. A scope
can fit its inventory budget but be too large to serialize: `GV_EVIDENCE_LIMIT`
reports the size and limit; no partial manifest is emitted. There are no flags
to relax these ceilings. Scope hashes do not bind environment variables,
external services, files outside the root, or all transitive execution inputs.

## Migrate version 1 deliberately

Version 1 accepts only its original keys. Design-plan rows retain their original
cover checks and receive an informational migration note, not a new warning.
Legacy `ga.review-quality` is historical acceptance evidence only.
Legacy rows for the three implementation-behavior claims always fail with
`GV_MISSING_IMPLEMENTATION_SUBJECT`, even when BUILD/pack hashes match and even
without `--impl`. Review the full implementation/test scope and generate v2
`kind: current`, or explicitly recast the statement as v2 `kind: plan` and leave
current evidence missing. No legacy row is silently upgraded or grandfathered.
Checking a v2 current row without `--impl` fails with `GV_IMPL_REQUIRED`.

## What the gate checks

`Gv-attest` activates when evidence exists or phase artifacts make claims owed.
An owed but missing evidence file is an error. Once the file exists, the gate verifies:

- the file parses, carries a supported version, and holds a non-empty
  `attestations` list (an empty record is a failure, not a pass);
- every `claim` resolves to the closed vocabulary, and no claim appears twice;
- every row names an `attestor` and a real date;
- every covered `path` is design-relative and names a file the design carries;
- every `hash` matches that file's current bytes;
- kinds match the vocabulary and current rows bind the complete supplied root;
- retained original/copy custody and cleanup succeed before publishing results.

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

A partial record may warn about owed judgments not yet recorded. An explicit
plan for an implementation-behavior claim also warns that current evidence is
missing. `--complete` promotes these warnings to handoff failure. Wrong or empty
records block immediately: unknown claims, unattributed rows, dangling subjects,
stale hashes, incomplete scope and custody failures are not warnings or DRIFT.
Stop/SubagentStop finalize before decisions, output, or ledger clearing; custody
failures block even under relaxed policy, an open wave, or empty gate selection.

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
| `g2.event-contract-completeness` | the event-contract table covers every cross-component event and the dependency declaration itself is complete (each row's columns and participants are checked by G2, and its wiring to the machines by Gx; what stays attested is that no cross-component event is MISSING) | `ARCHITECTURE.md` |
| `g2.nfr-content` | the NFR record's CONTENT is true (presence and topic coverage are checked; the posture is judgment) | `ARCHITECTURE.md` |
| `g3.guard-semantics` | each guard's semantics actually enforce the invariant it names | `machines/*.machine.json` |
| `g3.invariant-enforcement` | every Modelith invariant is guarded or structurally impossible; any that is neither is listed | `machines/*.machine.json` |
| `g3.residual-transitions` | every C4 dependency failure has its residual transition, reclassified by its mitigation rather than deleted | `machines/*.machine.json` |
| `g3.event-redelivery` | every consumed external event has its event-contract row and a redelivery story (the row-to-machine direction is checked by Gx; the machine-to-row direction has no deterministic marker and stays here) | `machines/*.machine.json` |
| `gt.conformance-test-shape` | a wholesale-conformance test parses the committed oracle table and asserts, per row, the next state AND the expected actions | `BUILD.md` |
| `g4.zero-context` | a coding agent with no prior context could execute each milestone from its packet alone (or the single `BUILD.md` in full mode) | `BUILD.md` and every non-index `BUILD/*.md` packet |
| `g4.standin-coverage` | isolated child only: the neighbor stand-in section exists, every neighboring boundary has a stand-in held to its oracle, and the environment recipe is self-contained | a `Neighbor stand-ins` section in the build document |
| `g4.pack-event-discipline` | pack child only: the implementation carries no emitter or handler for an event absent from its pack | `pack/` |
| `ga.review-quality` | the milestone reviewer judged WELL: the DoD was really met, the acceptance file's attestations are true, and its findings list is complete | `acceptance/` |

## Relationship to Ga-accept

`design/acceptance/M<n>.yaml` keeps its free-prose `attestations:` list, which is what the
milestone review checked by judgment on that commit, and nothing about existing acceptance
files changes. The two records answer different questions: Ga's list is scoped to one
milestone review at one commit, while a `Gv` current row is the standing judgment
on its design and implementation scope, invalidated when that scope changes.

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
