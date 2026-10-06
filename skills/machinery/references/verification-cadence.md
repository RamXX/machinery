# Verification cadence: per change, landing, checkpoint

Read this before planning how a build is executed and verified. Verification is
split by cadence. Each layer has a fixed rule set; nothing in a layer is
skipped, and nothing from a heavier layer is pulled into a lighter one. The
expensive evidence runs once per batch, where it finds things, instead of on
every landing, where it mostly repeats itself.

## Per change: the builder

- the finding and its reproducer;
- locked RED tests that fail at the RED commit, then GREEN;
- failing-first coverage companions;
- tests of the changed units only;
- format and lint on the changed units;
- only the cheap gates the diff touches.

No full workspace, reverse-dependency closure, corpus or end-to-end suite,
smoke, or fuzz per change. Two exceptions: a change whose documented effect
alters outputs runs the affected end-to-end suites, and a change that adds an
untrusted input surface runs a short fuzz of it.

## Per change: the reviewer

Adversarial reasoning and targeted real-process probes, threat table first
([threat-modeling.md](threat-modeling.md), section 5). The reviewer does not
rerun suites the builder already ran.

## Landing

Integrate the main line, type-check or compile everything, run the changed
units' tests and the locked RED tests, land. No attestation, record assembly,
SBOM, design gates, or version bump per landing. Where a landing wants a
machinery signal, `machinery check <design> --landing` runs every gate except
the checkpoint-only ones (attestation, acceptance) and reports stale
external-checker evidence as a note.

## Checkpoint

About every eight landed changes, and at every milestone, epic, release, end of
an unattended session, and before any artifact goes to a customer:

1. the full test suite, every end-to-end and corpus suite, smoke, and fuzz;
2. record assembly, and the version bump, once per batch;
3. re-attestation exactly once, as the last content step;
4. SBOM, then `machinery check <design> --impl <dir>`: the design gates run
   after re-attestation and after the SBOM.

A checkpoint failure is bisected across the batch and fixed under the normal
process: a finding, a reproducer, RED, GREEN.

Machinery is the checkpoint gate, not a per-landing gate. Between checkpoints
the main line is expected to be stale for attestation, records, design gates,
and release pins, and that is not a defect. The machinery stop hook runs the
landing selection, so checkpoint-only staleness never blocks ordinary work. It
still blocks the real errors: import-boundary violations, generated-artifact
DRIFT, and, in strict mode, lint errors in the design sources.

## Locked tests: purpose over ritual

A locked test exists to protect a decided behavior. Its identity is its exact
bytes and file inventory. When that decision changes, the change that alters it
amends the locked test in place, in the same change. Such an amendment must be
explicit, is a new evidence revision, and replays RED and every applicable gate
before the amended test locks again. The amended test's header:

1. quotes the original line it replaces;
2. gives the reason (the decision that changed, with its date and owner);
3. says what the test still protects;

and a byte or behavior proof backs it (the old and new assertions run against
the old and new behavior). Weakening is never allowed: an amendment may not
accept anything the old test refused unless the decision itself changed what
must be refused. Tightening is always allowed. Neither formatting nor token
equality authorizes an editing exemption. Stop and ask only when the contract
is undecided.

## Parallelism

Every unblocked change may run at once, within resource limits. Priority orders
scarce capacity; it never pauses ready work. Concurrent work uses separate
worktrees. Record fragments are disjoint per change (one file per change or per
milestone, such as `acceptance/M<n>.yaml`), never a shared generated file that
two changes both rewrite, so merges never conflict on generated records.

## Hook robustness

A crashed or interrupted hook-state write, for example on a full disk, never
blocks every later command. The hook removes a leftover temp file by itself
when it is empty or provably stale. Otherwise it preserves the file under a
crash-evidence name, marks the project dirty so the stop-time checks still run,
and refuses that one command with a one-line recovery note. The next command
proceeds. Nobody is needed just to delete an empty temp file.
