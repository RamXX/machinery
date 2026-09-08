# Release notes discipline

Machinery regenerates committed artifacts and re-derives proofs on every release, so a release
note is a safety document, not marketing. This page states the discipline every machinery release
follows, records the historic omission that motivated it, and prepares the notes for the upcoming
hardening release.

## The rules

1. **Generated-output changes are called out explicitly.** Any change to a generator (TLA+,
   Alloy, oracle, projection, packs, golden corpus) that alters committed artifact bytes ships a
   release note naming the generator, the artifact families affected, and the regeneration
   command consumers run.
2. **Proof-scope changes are never silently called equivalent.** If a release changes what a
   green gate establishes (stronger evidence, weaker evidence, or differently-scoped evidence),
   the note says so in those terms. A gate that now proves more is not "the same gate, faster",
   and a check that now proves less is a breaking change, not a cleanup.
3. **Compatibility and migration get their own section** whenever a release changes schemas,
   installation receipts, attestations, registries, or CLI behavior: what still works unchanged,
   what is rejected now, and the exact commands to migrate.
4. **Known exclusions stay named.** The README's "What machinery does not verify" section is the
   canonical residual list; a release note never shrinks it implicitly by claiming broader
   coverage.
5. **Publication gating is stated as it is, not as planned.** Release publication is being
   hardened to require exact-commit verification of CI, formal, and security status for the exact
   release commit before anything publishes. Until that enforcement lands and is applied, notes
   must not describe it as active; its locally tested release policy is deliberately not claimed
   as remotely enforced.

## The historic omission: v0.6.3

The v0.6.3 TLA generator change (`Live_OverlayResolves`, and the WF conjunct omitted for
machines with a `_refusal` state) altered regenerated `.tla` bytes for every consumer with a
`_refusal` machine and shipped with no release note at all. Consumers regenerated proofs,
byte-compared artifacts, and had no way to know the change was intended. That omission is the
motivating incident for this discipline.

## Preparing the current hardening release

The accepted hardening batch (unreleased at the time of writing; `CHANGELOG.md` carries the
living list under Unreleased) changes several proof scopes and one installer contract. The
release notes for it must state:

- **Installer convergence (receipt-aware bootstrap).** Rerunning the one-line installer over an
  existing install now converges with `machinery update`: the complete recorded plan, per-group
  copy/symlink modes, safe repair of edited/missing owned artifacts, fail-closed unsafe receipts.
  Compatibility: explicit homes/targets still cannot combine with bootstrap defaults; a corrupt
  or unsafe receipt now fails closed where older releases silently refreshed the default homes.
- **Host plugin refresh failures are failures.** Previously documented as warnings over a
  successful update; they are returned failures after the direct commit, with recorded retry
  obligations. Host plugin caches remain outside machinery's rollback.
- **Oracle coverage (`Gt`) is discovery, not execution.** The gate labels its result static
  discovery and states that tests were not executed; bypass shapes that previously passed are
  rejected.
- **Attestation evidence v2 kinds (`plan`/`current`/`historical`).** Example attestations
  migrated; a current claim requires an implementation subject, and implementation-subject
  changes invalidate stale reviews.
- **Frozen-test identity requires exact bytes.** Formatting-only rewrites are no longer the same
  frozen test; evidence replay requires exact identity.
- **Deterministic checker-container lifetime and budgets.** Closed resource budgets, output-breach
  aborts, and force-removal on every exit path.
- **Native custody for machinery's own subprocesses**, with honest limits: cleanup failures are
  reported, never concealed, and custody is process-ownership containment, not a hostile-host
  sandbox.
- **`machinery recover`** for interrupted design publication.
- **The required integration lane** executes infrastructure-dependent suites deterministically;
  missing infrastructure fails the lane rather than skipping.
