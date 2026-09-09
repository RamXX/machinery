---
id: MAC-qo6n
title: "Tier the gate: fast pre-push, heavy evidence in CI and make ci-linux, custody suites in the lane"
status: open
priority: 1
type: feature
labels: [release-process, ci, preflight]
created_at: 2026-09-08T16:36:40Z
created_by: ramirosalas
updated_at: 2026-09-09T16:38:38Z
content_hash: "sha256:1af76db6bfa42a2253add000851277b9a447702639d71af682bdc012fddd4ed2"
---

## Description
## Intent
Releasing a one-line patch must not cost a 50-minute local gate per push. Keep the exact-commit release gate (ci, formal, security green on the tagged SHA) unchanged; move the heavy evidence to where it is enforced and make Linux evidence obtainable before a push.

## Evidence (0.7.0 release night, 2026-09-08)
Six push attempts, each paying the full pre-push gate (about 50 minutes: 20 to 30 minute race sweep dominated by internal/install's serial fsync suite, required lane 5 to 8 minutes, TLC, C4, Docker checkers). Five hosted-CI failures were invisible to the macOS gate (linux-only lint, cold module cache, 2-vCPU custody timing, setup-beam OTP layout, ci.yml mirror drift). Two custody tests failed only under the parallel sweep's load, never in isolation.

## Scope
1. Pre-push hook runs the cheap tier only: whitespace, gofmt, vet, golangci-lint (darwin and GOOS=linux), actionlint, shellcheck, tidy, docs gate, render check, build, scripts/example-gates.sh, golden corpus. Target under 5 minutes. No bypass variable.
2. Heavy tier stays authoritative in hosted CI: race sweep, required lane, verify-formal, verify-c4, verify-checkers, native macOS suite. `make preflight` keeps running everything for a deliberate full local run.
3. `make ci-linux`: the race sweep and the required lane inside a 2-CPU linux/amd64 container (pinned golang image plus the lane's pinned runtimes), runnable on a laptop or on the Linux VM, producing the same evidence hosted CI produces. Document it in CONTRIBUTING.md as the step before any push that touches custody, adapters, or the lane.
4. Move internal/install's fsync-bound suite and the load-sensitive custody tests (processscope, processcontrol, adapters contributor-lane tests) out of the parallel race sweep into the required lane with fixed budgets, so the ordinary sweep drops to a few minutes and custody timing is measured on a quiet lane, not beside sibling packages.
5. scripts/integration-lane's laneValidatePreflight and the cmd/machinery tests pinning preflight.sh, ci.yml, and the Makefile are updated in the same change; the three mirrors keep one owner per policy (scripts/example-gates.sh pattern).

## Acceptance
- `git push` of a docs-only commit completes its hook in under 5 minutes on the documented host class.
- `make ci-linux` on the Linux VM reproduces the hosted ci test job and integration-required job results for the same commit (green on a commit hosted CI marks green; red on 06aed43 for the same reasons).
- Hosted CI required job set and docs/release-policy.json unchanged; release gate unchanged.
- Race sweep wall time under 10 minutes on the documented host class; the moved suites run in the required lane with their own receipts.
- CHANGELOG entry and CONTRIBUTING.md updated.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments

### 2026-09-09T16:38:38Z ramirosalas
Landed on main at 0ee0366450c91fd2ac5d351142b45780adbaf7a1 (2026-09-09): items 1 (fast pre-push tier, scripts/preflight-fast.sh, 1m25s measured), 2 (heavy tier unchanged in hosted CI and make preflight), 3 (make ci-linux: pinned 2-CPU linux/amd64 container, scripts/ci-linux.sh + scripts/ci-linux.dockerfile) and 5 (contract tests tightened, lane validator untouched). Still open: item 4 (move internal/install's fsync-bound suite and the load-sensitive custody/adapter suites out of the parallel race sweep into the required lane with fixed budgets and receipts; needs before/after measurement), and running the ci-linux sweep as a non-root user inside the container (root hides uid defects, see MAC-aldv). End-to-end ci-linux evidence run on the Linux VM pending in this session.
