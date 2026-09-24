---
id: MAC-zafm
title: "Move the gate definition to a Dagger module with one owner per policy"
status: closed
priority: 1
type: feature
labels: [release-process, ci, dagger]
created_at: 2026-09-08T16:36:40Z
created_by: ramirosalas
updated_at: 2026-09-24T21:33:44Z
content_hash: "sha256:86cdcdd98f07c79b6f812a60dde1e50b2bcd3ee0b292aeff68384e630d43f9d3"
closed_at: 2026-09-24T21:33:44Z
close_reason: "Mostly delivered in 0.8.0 (da1226e8, a9dd4b53). Residual preflight heavy tier filed as MAC-4cbc; release.yml in MAC-y8lj. Triage 2026-09-24."
---

## Description
## Intent
One gate definition, in Go, that runs identically on a laptop, the Linux VM, and GitHub Actions inside pinned containers. Replaces the three hand-maintained mirrors (scripts/preflight.sh, .github/workflows/ci.yml, Makefile) and the per-host toolchain version checks with a Dagger module owned and reviewed like product code.

## Why
The 0.7.0 release night showed the mirrors drift (gates policy), the host environment leaks in (cold module cache, large shell env, linux-only lint), and toolchain pins are enforced by scripts that compare version strings on whatever the host has installed. A Dagger module makes the environment part of the definition.

## Depends on
The gate-tiering story (fast pre-push, `make ci-linux`, custody suites in the lane) lands first; this story replaces how the tiers are produced, not what they check.

## Scope
1. A Dagger module (Go) under ci/ or dagger/ exposing functions per tier: `lint`, `unit-race`, `golden`, `example-gates`, `required-lane`, `verify-formal`, `verify-c4`, `verify-checkers`, `docs`, `render-check`, `release-artifacts`. Each function pins its container image and toolchain versions from the same single-source files used today (.golangci-version, .actionlint-version, .shellcheck-version, .java-runtime-pin, .structurizr-pin, the lane's runtime pins).
2. Docker-backed checks (external checker OCI lifecycle, checker container budgets) run through Dagger's nested engine or a bound Docker service; document the trust boundary this introduces.
3. .github/workflows/ci.yml and formal.yml become thin callers of the module; the required job names in docs/release-policy.json stay identical so the release gate and branch-protection contexts do not change.
4. scripts/preflight.sh and `make preflight` call the module for the heavy tier; the cheap tier may stay native for speed, but its definition must derive from the module (no second copy of a policy).
5. The native macOS job stays a plain GitHub job (Dagger runs Linux containers); state this in CONTRIBUTING.md.
6. Caching: per-function cache volumes for the Go build cache, module cache, and provisioned engines so a docs-only change does not rebuild or re-download.
7. Governance: the module is linted and tested like product code; laneValidatePreflight and the mirror contract tests move to asserting the module is the single owner.

## Acceptance
- `dagger call unit-race` on a laptop and on the Linux VM produce the same results as the hosted ci test job for the same commit.
- Hosted CI required jobs pass through the module with unchanged job names; the release gate publishes 0.7.x unchanged.
- No toolchain version string comparison against the host remains for the containerized tiers.
- CHANGELOG entry, CONTRIBUTING.md, and docs/release-notes.md updated; the custody and assurance contracts note where execution now happens.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-24T21:33:44Z status: open -> closed

## Links


## Comments
