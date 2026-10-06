# Threat-driven design and shift-right verification

This document is the contract for two process rules machinery enforces from
0.11.0 on. Part A makes design and TDD start from an adversary, not from a
happy path. Part B moves the expensive evidence out of every landing and into a
periodic checkpoint, without loosening what a landing must prove.

Both rules come from the same observation. A verifier can pass a large mutation
and tamper suite and still accept a validly signed record whose status was
revoked, a countersignature made with the primary signer's own key, a signed
field relabelled through delimiter injection, a structure whose unsigned digests
were all recomputed after an edit, a missing optional input that silently
produced a pass, and a "not verified" result with exit code 0. Every one of
those tests assumed a single attacker: an outsider without a key who edits
bytes and does not recompute digests. Criteria were written only as positive
outcomes, correctness was judged by agreement with a reference implementation
that shared the flaws, and known gaps were closed as "documented". Separately, a
full evidence ceremony on every landing cost the most and found the least.

## Part A: threat-driven design and TDD

### The threat ledger: `design/threats.yaml`

Modelith's schema is closed, so adversary classification lives in a sidecar
next to the domain model. `Gz-threat` holds it. The file is optional; without it
Gz runs in audit mode (see Migration).

```yaml
mode: enforce                 # audit | enforce
enforced_since: 2026-10-05    # required with enforce
subjects:
  - subject: Session          # an entity, or Entity.action, declared in the model
    kinds: [authenticates]
    adversaries:
      - adversary: outsider_without_key
        invariant: session-requires-valid-credential
        negative_test: SESS-forged-credential-refused
      - adversary: trusted_key_holder
        invariant: session-key-role-bound
        negative_test: SESS-wrong-role-key-refused
      - adversary: unsigned_document_controller
        not_applicable: "a session carries no unsigned digest"
      - adversary: valid_signature_wrong_meaning
        invariant: session-status-active
        negative_test: SESS-disabled-user-refused
      - adversary: optional_input_omission
        invariant: session-omitted-input-fails-closed
        negative_test: SESS-missing-expiry-refused
      - adversary: duplicate_replay_reorder
        accepted_risk:
          owner: Product owner
          date: 2026-10-05
          reason: "local single-user store; replay needs disk access"
          visible_in: "`session status` prints replay-unchecked"
      - adversary: signed_field_injection
        not_applicable: "no signed text fields"
      - adversary: cross_bundle_substitution
        not_applicable: "no bundles"
not_security_relevant:
  - subject: Activity
    reason: "a free-text log entry; it gates nothing"
waivers:
  - subject: LegacyImport
    owner: Product owner
    date: 2026-10-05
    reason: "excavated as-is; classification scheduled with the importer rewrite"
verifiers:
  - component: authz          # the workspace.dsl identifier
    decides: "record visibility and CRUD admission"
```

**Kinds.** A subject is security-relevant when it verifies, authenticates,
signs, hashes, approves, admits input, or gates a decision. `kinds` names which:
`verifies`, `authenticates`, `signs`, `hashes`, `approves`, `admits_input`,
`gates_decision`. At least one is required.

**Adversaries.** The closed vocabulary is the standard set. A classified subject
lists every one of them exactly once:

| id | adversary |
|---|---|
| `outsider_without_key` | an outsider without any trusted key |
| `trusted_key_holder` | the holder of any trusted key, including a trusted key used in a role it should not have (a countersignature by the primary key, one key counted twice) |
| `unsigned_document_controller` | the controller of every unsigned document, who recomputes all unsigned digests after an edit |
| `valid_signature_wrong_meaning` | a valid signature over the wrong meaning: status, domain, subject, or version |
| `optional_input_omission` | omission of any optional input |
| `duplicate_replay_reorder` | duplicate, replay, or reorder |
| `signed_field_injection` | delimiter, encoding, normalization, or case-folding injection in signed fields |
| `cross_bundle_substitution` | substitution of a document taken from another valid bundle |

**Rows.** Each adversary row takes exactly one disposition:

- `invariant` plus `negative_test`: the threat invariant this adversary
  produces, and the locked negative test that refuses it. The invariant is a
  Modelith invariant id, so the ordinary carrier gate (Gc) requires a carrier
  for it like any other invariant. The test is a stable oracle id or a test
  file path. With `--impl`, Gz requires the token to appear in the
  implementation's test files.
- `accepted_risk`: an owner-signed acceptance with `owner`, `date`
  (YYYY-MM-DD), `reason`, and `visible_in`, which says where the component's
  own output shows the risk to its caller. A risk nobody can see from the
  output is not accepted, it is hidden.
- `not_applicable`: a reason the adversary has no attack surface on this
  subject at all. It is a classification, not a closure: a real gap is never
  `not_applicable`.

### Rules Gz-threat enforces

| Rule | Finding |
|---|---|
| A1 classification | a model entity or action that looks security-relevant (its name, definition, attributes, or action names speak of verification, authentication, credentials, signatures, hashes, digests, approval, attestation, admission, authorization, tokens, secrets, or certificates) and is listed under none of `subjects`, `not_security_relevant`, or `waivers` |
| A1 completeness | a classified subject missing an adversary, listing one twice, or naming an unknown one |
| A1 waiver | a waiver without `owner`, `date`, and `reason` |
| A2 threat invariant | a row whose `invariant` is not declared in the model |
| A6 agreement | a `negative_test` that only states agreement with a reference implementation ("matches reference", "agrees with", "same as") |
| A7 closure | an `accepted_risk` without `visible_in`; a threat invariant closed only by a `formal/waivers.yaml` note; any row whose disposition is the word "documented" |
| A8 governance | a declared verifier that is not a `workspace.dsl` element identifier, or that no Architecture Contract boundary binds to code; a `workspace.dsl` element whose identifier, name, or description speaks of verifying, checking, attesting, signing, authorizing, or gating and that is neither a declared verifier nor listed under `not_security_relevant` (which may name a DSL element as well as a model subject) |
| `--impl` | a `negative_test` token that no test file under the implementation contains |

Detection is a deliberately broad word match. It finds candidates; a human
decides. A false positive costs one `not_security_relevant` line with a reason.

### Paired criteria and the threat table (Gb-plan)

A milestone is security-relevant when its block, packet, or shard names a
classified subject or one of its threat invariants as a whole token. For such a
milestone Gb-plan requires, in the milestone block (full mode) or in its packet
or shard (manifest mode):

1. **Paired acceptance criteria.** Each property is an `Accept:` line followed
   by a `Refuse:` line. At least one pair, and no `Accept:` without its
   `Refuse:`. A packet with only positive criteria fails.
2. **A threat table.** A Markdown table with an `adversary` column and a
   `negative test` column (an `accepted risk` column is optional). One row per
   adversary row of every subject the milestone names, except `not_applicable`
   rows. Each row names its locked negative test, or the accepted risk that
   `threats.yaml` records for it. A cell that reads "documented", or that only
   claims agreement with a reference, fails.
3. **The third RED quality check.** Exactly one non-empty `Pass-wrongly:` line that
   answers "what would make this pass wrongly?" for the milestone.

Tamper tests in the table target the untrusted artifact being checked, not only
the trusted inputs, and every tamper test also asserts that each untouched check
still passes. That is a test-writing rule the review enforces by reading; Gb
holds the structure.

### Fail closed by contract (lint)

Any component that reports a verdict or an exit status reports success only
when every applicable check passed. "Not checked" is visible and never yields
caller-facing success. In a machine, tag the states:

- `accepting`: a state that reports caller-facing success;
- `unchecked` or `omitted`: a state in which some applicable check has not run,
  or an input it needs was absent.

`machinery lint` (and so G3) fails a machine with a transition of any kind
(`on`, `always`, `after`, invoke `onDone` or `onError`) from an `unchecked` or
`omitted` state into an `accepting` state, and fails an `accepting` initial
state.

### Review starts from threats (Ga-accept)

The independent review of a security-relevant milestone writes its own threat
table first, then probes each row with a real process, and asserts the failure
reason of every negative probe, not just a non-zero exit. Its acceptance file
records that work:

```yaml
threat_review:
  - adversary: outsider_without_key
    subject: Session
    probe: "forged credential file passed to the real CLI"
    expected_reason: "credential signature invalid"
    observed_reason: "credential signature invalid"
  reopened_limits: []         # earlier documented limits this change widens
```

Ga requires, for an ACCEPTED security-relevant milestone whose acceptance date
is on or after `enforced_since`, one `threat_review`
row for every adversary row the milestone's threat table carries, each with a
non-empty `probe`, `expected_reason`, and `observed_reason`, and the two reasons
equal. A non-zero exit is not a reason.

## Part B: shift-right verification

Verification is split by cadence. Each layer has a fixed rule set; nothing in a
lower layer is skipped, and nothing from a higher layer is pulled down.

### Per change, by the builder

- the finding and its reproducer;
- locked RED tests that fail at the RED commit, then GREEN;
- failing-first coverage companions;
- tests of the changed units only;
- format and lint on the changed units;
- only the cheap gates the diff touches.

No full workspace, reverse-dependency closure, corpus, or end-to-end suite,
smoke, or fuzz per change, with two exceptions: a change whose documented effect
alters outputs runs the affected end-to-end suites, and a change adding an
untrusted input surface runs a short fuzz of it.

### Per change, by the reviewer

Adversarial reasoning and targeted real-process probes, threat table first (Part
A). The reviewer does not rerun suites the builder already ran.

### Landing

Integrate the main line, type-check or compile everything, run the changed
units' tests and the locked RED tests, land. No attestation, record assembly,
SBOM, design gates, or version bump per landing.

### Checkpoint

About every eight landed changes, and at every milestone, epic, release, end of
an unattended session, and before any artifact goes to a customer:

1. the full test suite, every end-to-end and corpus suite, smoke, and fuzz;
2. record assembly, and the version bump, once per batch;
3. re-attestation exactly once, as the last content step;
4. SBOM, then the design gates, after re-attestation and after the SBOM.

A checkpoint failure is bisected across the batch and fixed under the normal
process.

### Machinery is the checkpoint gate

Between checkpoints the main line is expected to be stale for attestation,
acceptance records, external-checker evidence, and release pins. That is not a
defect. `machinery check --landing` runs every gate except the checkpoint-only
ones (`gv`, `ga`) and reports checker-evidence freshness as a note; `machinery
check` without it is the checkpoint run. The stop hook uses the landing
selection, so it never blocks ordinary work for checkpoint-only staleness. It
still blocks the real errors: import-boundary violations, generated-artifact
DRIFT, and (in strict mode) lint errors in the design sources.

### Locked tests: purpose over ritual

When decided behavior changes, the change that alters it amends the locked test
in place. The amended test's header quotes the original line, gives the reason,
and says what the test still protects, and a byte or behavior proof backs it.
Weakening is never allowed; tightening is always allowed. Stop only for an
undecided contract.

### Parallelism

Every unblocked change may run at once, within resource limits. Priority orders
scarce capacity; it never pauses ready work. Concurrent work uses separate
worktrees, and record fragments are disjoint per change (one file per change or
per milestone, never a shared generated file both changes rewrite) so merges
never conflict.

### Hook robustness

A crashed or interrupted hook-state write, for example on a full disk, never
blocks every later command. The hook removes a leftover temp file by itself when
it is empty or provably stale (its content is already the live state, or older
than it). Otherwise it moves the temp file aside, marks the project dirty so the
stop-time checks still run, and refuses that one command with a one-line
recovery note naming the preserved file. The next command proceeds.

## Migration

- Existing adopters keep working. Without `design/threats.yaml`, Gz runs in
  audit mode: it lists candidate subjects missing classification and
  security-relevant milestones missing paired criteria or a threat table, as
  notes, and blocks nothing.
- `mode: audit` in the file validates what is written (a malformed ledger is
  always an error) and keeps missing classifications as notes.
- `mode: enforce` turns the findings into errors. To adopt on an existing
  design, run `machinery baseline <design> --gate gz --date <YYYY-MM-DD>`: it
  records each unclassified candidate with a hash of its model definition in
  `design/ratchet.json`. A recorded candidate prints as a `baselined:` note.
  A new candidate, or a recorded one whose definition changed, is an error, so
  the rules apply to new and changed security-relevant entities only.
- The milestone rules follow the same mode. Gb applies them only to milestones
  whose status is open, and Ga only to acceptance files dated on or after
  `enforced_since`: a milestone closed before adoption is history.
