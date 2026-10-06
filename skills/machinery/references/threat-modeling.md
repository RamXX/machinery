# Threat modeling, paired criteria, and threat-first review

Read this when the design has anything that verifies, authenticates, signs,
hashes, approves, admits input, or gates a decision, and before any build
handoff or review of such work. It is the complete contract for
`design/threats.yaml`, the Gz-threat gate, the security rules Gb-plan and
Ga-accept add, and the fail-closed lint.

## Why

A verifier can pass a large tamper suite and still accept a validly signed
record whose status was revoked, a countersignature made with the primary
key, a signed field relabelled through delimiter injection, a structure whose
unsigned digests were all recomputed, a missing optional input, and a "not
verified" result that exits 0. Those tests all imagined one attacker: an
outsider without a key who edits bytes and recomputes nothing. Criteria said
only what should pass, correctness meant agreeing with a reference that shared
the flaws, and gaps were closed as "documented". Every rule below closes one of
those doors.

## 1. Model the adversaries (Phase 1 and excavation)

Every entity or action that verifies, authenticates, signs, hashes, approves,
admits input, or gates a decision is security-relevant. For each one, consider
the whole standard adversary set:

| id | adversary |
|---|---|
| `outsider_without_key` | an outsider without any trusted key |
| `trusted_key_holder` | the holder of any trusted key, including one used in a role it should not have (a countersignature by the primary key, one key counted twice) |
| `unsigned_document_controller` | the controller of every unsigned document, who recomputes all unsigned digests after an edit |
| `valid_signature_wrong_meaning` | a valid signature over the wrong meaning: status, domain, subject, or version |
| `optional_input_omission` | omission of any optional input |
| `duplicate_replay_reorder` | duplicate, replay, or reorder |
| `signed_field_injection` | delimiter, encoding, normalization, or case-folding injection in signed fields |
| `cross_bundle_substitution` | a document substituted from another valid bundle |

Record the classification in `design/threats.yaml` (Modelith's schema is
closed, so it lives beside the model):

```yaml
mode: enforce                  # audit | enforce
enforced_since: 2026-10-05     # required with enforce
subjects:
  - subject: Credential        # a model entity or Entity.action
    kinds: [authenticates]     # verifies, authenticates, signs, hashes, approves, admits_input, gates_decision
    adversaries:
      - adversary: outsider_without_key
        invariant: credential-signature-required
        negative_test: CRED-forged-signature-refused
      - adversary: valid_signature_wrong_meaning
        invariant: credential-pass-implies-active
        negative_test: CRED-revoked-status-refused
      - adversary: duplicate_replay_reorder
        accepted_risk:
          owner: Product owner
          date: 2026-10-05
          reason: "single local store; replay needs disk access"
          visible_in: "the status command prints replay-unchecked"
      - adversary: cross_bundle_substitution
        not_applicable: "credentials are never bundled"
      # ... every adversary of the set, exactly once
not_security_relevant:
  - subject: Note
    reason: "free text; it gates nothing"
waivers:                       # brownfield excavation only
  - subject: LegacyImport
    owner: Product owner
    date: 2026-10-05
    reason: "excavated as-is; classified with the importer rewrite"
verifiers:
  - component: Verifier        # a workspace.dsl element
    decides: "credential admission"
```

Each adversary row takes exactly one disposition:

- **`invariant` + `negative_test`**: the threat invariant the adversary
  produces and the locked negative test that refuses it. The invariant is an
  ordinary Modelith invariant, so Gc requires a carrier for it like any other.
  Write it as a refusal property: "PASS implies status approved",
  "countersigning keys are distinct from the primary key and from each
  other", "displayed fields are derived from the signed source, never copied
  from the checked artifact", "an omitted input fails closed".
- **`accepted_risk`**: owner-signed (`owner`, `date`), with a `reason` and
  `visible_in`: where the component's own output shows the risk to its caller.
  A risk no caller can see is hidden, not accepted.
- **`not_applicable`**: the adversary has no attack surface on this subject at
  all. It is a classification, never the closure of a known gap.

Brownfield excavation records behavior and, for every security-relevant
subject it touches, the adversary classification or an owner-signed waiver.
"The code does X" is not a classification.

A security gap is closed by a fix or by an owner-signed accepted risk visible in
the output. A note in a document, a `formal/waivers.yaml` waiver of a threat
invariant, or a disposition that reads "documented" is not a closure, and Gz
rejects all three. A negative test that only claims agreement with a reference
implementation ("matches the reference") is rejected too: agreement is not
correctness, and a security property needs an independent negative test.

## 2. Govern every verifier (Phase 2)

Independent verifiers, checkers, CLIs, and gate scripts that make security
decisions are components like any other: they are `workspace.dsl` elements,
the Architecture Contract's import rules govern them, and `threats.yaml` lists
them under `verifiers`. Being "a tool" or "integrity only" is not an exemption.
Gz fails a declared verifier the model or the contract does not hold, and a
verifier-looking element (its name or description speaks of verifying,
checking, attesting, signing, authorizing, or gating) that is neither declared
nor listed as not security-relevant.

## 3. Fail closed by contract (Phase 3)

A component that reports a verdict or an exit status reports success only when
every applicable check passed. "Not checked" is visible and never yields
caller-facing success: an unverified result never exits 0. In its machine, tag
the success state `accepting` and every state where a check has not run or an
input was absent `unchecked` or `omitted`. The lint fails any transition from an
`unchecked` or `omitted` state into an `accepting` state, and an `accepting`
initial state. See [xstate-format.md](xstate-format.md).

## 4. Encode the deliverable (Phase 4)

A milestone is security-relevant when its block, packet, or shard names a
classified subject or one of its threat invariants. Its handoff carries:

1. **Paired criteria.** Every property is an `Accept:` line followed by its
   `Refuse:` line. A packet with only positive criteria fails Gb.

   ```markdown
   Accept: a credential signed by the issuer key with status active passes.
   Refuse: the same credential with status revoked fails with "credential revoked".
   ```

2. **A threat table** in the RED section: one row per adversary row of each
   subject it names (not-applicable rows excepted), each mapped to its locked
   negative test or its owner-signed accepted risk.

   ```markdown
   | adversary | negative test | accepted risk |
   |---|---|---|
   | outsider_without_key | `CRED-forged-signature-refused` | |
   | valid_signature_wrong_meaning | `CRED-revoked-status-refused` | |
   | duplicate_replay_reorder | | Product owner, 2026-10-05 |
   ```

   Tamper tests target the untrusted artifact being checked, not only the
   trusted inputs, and every tamper test also asserts that each check it did
   not touch still passes.
3. **The third RED quality check**, on one `Pass-wrongly:` line: what would
   make this pass wrongly? Answer it concretely (an omitted input, a reused
   key, a recomputed digest) and make sure a row of the table refuses it.

## 5. Review starts from threats

The independent reviewer of a security-relevant milestone:

1. writes their own threat table first, before reading the builder's;
2. probes each row with a real process (the real binary or entry point and a
   real crafted input), not a unit call;
3. asserts the failure reason of every negative probe; a non-zero exit is not a
   reason;
4. reopens earlier documented limits when the change widens what a pass means.

The acceptance file records it, and Ga holds it for every security-relevant
milestone accepted on or after `enforced_since`:

```yaml
threat_review:
  - adversary: valid_signature_wrong_meaning
    subject: Credential
    probe: "revoked credential passed to the verify command"
    expected_reason: "credential revoked"
    observed_reason: "credential revoked"
reopened_limits: []
```

## Adoption

Without `threats.yaml`, Gz runs in audit mode: it lists the subjects missing a
classification and the milestones missing paired criteria or a threat table as
notes, and blocks nothing. `mode: audit` validates what the file says and keeps
the missing classifications as notes. `mode: enforce` makes them errors. On an
existing design, run `machinery baseline <design> --gate gz --date
<YYYY-MM-DD>` first: each unclassified subject is recorded with a hash of its
model definition, prints as `baselined:`, and only a new subject or a changed
definition blocks. Closed milestones and acceptances dated before
`enforced_since` are history and are not rechecked.

Detection is a broad word match over names, definitions, attributes, and action
names. It finds candidates; a person decides. A false positive costs one
`not_security_relevant` row with a reason.
