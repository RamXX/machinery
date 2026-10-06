---
id: MAC-qo6n
title: "Tier the gate: fast pre-push, heavy evidence in CI and make ci-linux, custody suites in the lane"
status: open
priority: 3
type: feature
labels: [release-process, ci, preflight]
created_at: 2026-09-08T16:36:40Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:52Z
content_hash: "sha256:9ea1df496a412385025f7b344ceccc56fcf95ee5d75ed7228f4daa8fd831746e"
related: [MAC-2k07, MAC-awrq, MAC-y5ho]
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
Triage 2026-09-24: PARTIAL. Items 1-3 and 5 shipped 0ee03664 (0.7.1); non-root sweep 220c2b29 (0.8.0); socket-sharing defect superseded by Dagger nested dockerd. Remaining scope: item 4 only, install/custody suites out of the Dagger Test go test -race sweep into their own lane (CHANGELOG:743).
Revalidated 2026-10-06 against v0.11.0: partial. Remaining: Item 4 only: move the fsync-bound internal/install suite and the load-sensitive custody tests (processscope, processcontrol, adapters) out of the parallel race sweep into the required lane with fixed budgets and receipts, with before/after wall-time measurement; keep the lane validator and the three policy mirrors in sync. Evidence: Items 1-3 and 5 shipped in 0ee03664 (0.7.1: scripts/preflight-fast.sh, make ci-linux at Makefile:104-105, scripts/ci-linux.sh); non-root sweep 220c2b29 (0.8.0). Item 4 still open: .dagger/main.go:352 runs `go test -race -count=1 ./... -timeout=` over every package including internal/install and the custody suites; scripts/ci-linux.sh:119 likewise; CHANGELOG ~946 says the move stays open. Notes: Retitle to the remaining item-4 scope. Related MAC-2k07, MAC-awrq, MAC-y5ho (host-load flakes) are what item 4 would mitigate, not blockers.

## History


## Links
- Related: [[MAC-2k07]], [[MAC-awrq]], [[MAC-y5ho]]

## Comments

### 2026-09-09T16:38:38Z ramirosalas
Landed on main at 0ee0366450c91fd2ac5d351142b45780adbaf7a1 (2026-09-09): items 1 (fast pre-push tier, scripts/preflight-fast.sh, 1m25s measured), 2 (heavy tier unchanged in hosted CI and make preflight), 3 (make ci-linux: pinned 2-CPU linux/amd64 container, scripts/ci-linux.sh + scripts/ci-linux.dockerfile) and 5 (contract tests tightened, lane validator untouched). Still open: item 4 (move internal/install's fsync-bound suite and the load-sensitive custody/adapter suites out of the parallel race sweep into the required lane with fixed budgets and receipts; needs before/after measurement), and running the ci-linux sweep as a non-root user inside the container (root hides uid defects, see MAC-aldv). End-to-end ci-linux evidence run on the Linux VM pending in this session.

### 2026-09-09T16:55:44Z ramirosalas
ci-linux end-to-end on the Linux VM (2026-09-09, candidate 98b47e3, as deploy): the image builds and carries the pinned runtimes; the sweep fails in scripts/tree-inventory on the MAC-33sp container coarse-timestamp witness limitation (hosted ubuntu-latest passes), and because scripts/ci-linux.sh runs the sweep and the lane in one 'set -e' shell the lane does not run after a sweep failure. Follow-ups: run the lane even when the sweep fails (report both), mount a fine-grained-timestamp tmpfs for TMPDIR or land MAC-33sp, set a UTF-8 locale in the image (Elixir warns), and run as a non-root user.

### 2026-09-09T17:10:39Z ramirosalas
VM evidence run, second half (2026-09-09): after LANG=C.UTF-8, a shared TMPDIR mounted at the same path, and safe.directory in the system git config, the required lane inside the container reaches custody-go-pilot and fails there: integration_lane_custody_test.go:273/337 'cancelled provisioning/suite must fail with verified owned-job cleanup' because checker containers started through the host socket run outside the runner container's pid namespace, so process custody cannot witness their cleanup. Conclusion: the socket-sharing design cannot reproduce the integration-required job; ci-linux needs either a nested daemon (dind, privileged) or the lane run natively on the Linux VM with the pinned runtimes, and the sweep needs a fine-grained-timestamp TMPDIR (tree-inventory ABA witness, see the earlier comment). The sweep otherwise passed 30 of 32 packages on 2 CPUs. Local fixes for the three container defects are committed on main after the v0.7.1 tag.
