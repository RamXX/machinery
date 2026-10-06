---
id: MAC-y8lj
title: "release.yml is the last workflow not delegating to the Dagger module"
status: open
priority: 3
type: task
labels: [ci, dagger, release]
created_at: 2026-09-10T14:34:57Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:53Z
content_hash: "sha256:f7cc4b6182e9a27a095f8dd190ca05032612e62850289d611fbc14410021316f"
related: [MAC-4cbc]
---

## Description
## Context

Every other workflow now delegates to the Dagger module: `ci.yml`, `nightly.yml`, `formal.yml` and
`security.yml` are three steps per job (checkout, install the pinned CLI, one `dagger call`), and
the wiring guards require the module to carry each pinned command and the workflow to delegate
rather than restate it. `release.yml` is the one workflow still carrying its own definitions.

## Why it was not converted with the rest

Three reasons, in order of weight:

1. **It cannot be validated without publishing a release.** `release.yml` triggers on a `v*` tag
   push or on `workflow_dispatch`, and both run the whole pipeline through to `publish`. There is
   no path today that exercises the workflow without creating a GitHub Release. Every other
   conversion was validated on a branch before landing; this one could not be, and landing an
   unvalidated change to the release path is the risk the migration was meant to remove.
2. **Most of it is not containerizable work.** `gate` queries the GitHub API for run conclusions
   with `github.token` and needs `actions: read`. `publish` needs `contents: write`,
   `id-token: write` and `attestations: write` for SLSA build provenance, and creates the release.
   Version resolution reads `github.event_name`, `inputs.version` and `github.ref_name`. None of
   that becomes clearer inside a container, and moving the token into one widens the blast radius
   of the most security-sensitive job in the repository.
3. **The duplication it leaves is small.** The genuine overlap with the module is the
   cross-compile in `build`, which is close to `Machinery.Build` but not the same job: the release
   build stamps a resolved version, binds it to the checked-out commit, and exports artifacts for
   `manifest` and `publish`, where `Build` only proves every target compiles.

## What conversion would actually look like

- `build` -> a `ReleaseBuild(version string) *dagger.Directory` function that cross-compiles all
  five targets with the resolved version stamp and returns the artifact directory, with the
  workflow doing `dagger call release-build --version=... export --path=dist`. This is the piece
  worth doing: it is real duplication and its failure mode is a failed release build, which fails
  closed and blocks publication rather than corrupting it.
- `gate` -> possible, passing `github.token` as a Dagger secret, but see reason 2. Recommend
  leaving it as a workflow step.
- `manifest`, `publish` -> not worth converting. Keep as workflow steps.

## Recommended sequencing

Do this in the same pass as the next real release, not before. That release is the natural
validation opportunity: convert `build`, verify the exported artifacts byte-match what the current
pipeline produces for the same commit, and cut the release with a human watching. Converting it
cold, with no release pending, means landing an unexercised change to the one pipeline whose
failure is most expensive.

## Acceptance criteria

- `release.yml`'s `build` job delegates to a module function that carries the cross-compile.
- The exported artifacts for a given commit and version are byte-identical to what the pre-change
  pipeline produced for that same commit and version, demonstrated before the release is published.
- `gate`, `manifest` and `publish` keep their current permissions and remain workflow-owned, with a
  comment in `release.yml` recording that this is deliberate.
- The wiring guards learn about whichever half moves, the way they did for ci, formal and nightly.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Add ReleaseBuild in the Dagger module, delegate release.yml build, prove byte-identical artifacts against a prior release commit before tagging, add a deliberate-ownership comment and update wiring guards. Evidence: Verified at HEAD 8c620d8f (v0.11.0). .github/workflows/release.yml still carries its own build/gate/manifest/publish definitions (checkout/setup-go/inline run steps, lines 43-388); no ReleaseBuild function exists anywhere in .dagger or the repo. CHANGELOG 0.11.0 does not list it. Notes: Issue itself says do it in the same pass as a real release, validated by comparing to the previous release's assets for that commit. Low urgency, fail-closed failure mode. Related MAC-4cbc is unrelated work, not a prerequisite.

## History


## Links
- Related: [[MAC-4cbc]]

## Comments

### 2026-09-10T16:40:13Z ramirosalas
2026-09-10: deliberately left unconverted for the 0.8.0 release. The acceptance criterion requires the converted build's exported artifacts to byte-match what the pre-change pipeline produced for the same commit and version before publication, and the pre-change pipeline only runs on the tag push that publishes, so no validation path existed inside this release. Next release: land the ReleaseBuild function and the delegated build job on main first, then compare its artifacts against the previous release's assets for that release's commit before tagging.
