---
id: MAC-8lxe
title: "SKILL.md not updated for v0.7.0 features, and doctor reports a content-stale installed skill as ok"
status: closed
priority: 3
type: bug
labels: [docs, skill, doctor]
parent: MAC-ui8a
created_at: 2026-09-08T07:55:53Z
created_by: ramirosalas
updated_at: 2026-09-08T09:27:50Z
content_hash: "sha256:40163992144e3f7ef2a0e0aacb0b4964c7dc341e5403ebfd205c69483a1c2c96"
closed_at: 2026-09-08T09:27:50Z
close_reason: "Landed on main (release 0.7.0 line) at 5c35463: b372e19 SKILL.md updated for 0.7.0 and doctor reports a stale skill as stale (release-level only)"
---

## Description
## Symptom

Two agent-facing surfaces did not move with the v0.7.0 release.

### 1. skills/machinery/SKILL.md has no mention of the release's headline features

```
$ grep -n "tdd\|assurance\|elixir-exunit\|recover" skills/machinery/SKILL.md
(no output)
```

`machinery recover` is a new command (CHANGELOG 0.7.0 "Added"); executable test assurance,
the four native adapters and replay-input retention are the release's largest additions. An agent
driving machinery from the skill will not know any of them exist, and the skill is the primary
consumer interface for exactly the agents this tool is built for.

### 2. `machinery doctor` reports a stale installed skill as ok

```
$ machinery doctor            # v0.7.0 binary
install status:
  ok       skill at /Users/ramirosalas/.claude/skills/machinery
  ...
$ diff -q <worktree>/skills/machinery/SKILL.md ~/.claude/skills/machinery/SKILL.md
Files ... differ
```

Doctor reports `ok` for an installed skill whose bytes do not match the running binary's release.
Since the installer's whole 0.7.0 story is "reruns converge on the recorded installation" and
"safely edited or missing owned artifacts are repaired to exact current-release content", doctor
saying `ok` here is the one place a user would expect to be told to run `machinery update`.

## Expected

1. SKILL.md covers `recover` and states the assurance position (once MAC-tk0i is settled, whether
   the assurance flow is available or design-only).
2. Doctor compares installed artifact content against the running binary's release and reports
   `stale` with the exact refresh command, rather than `ok`.

## Actual

As above.

## H2 impact

MINOR but real. H2 is machinery-governed and its conductor sessions load the skill. On upgrade the
project would silently keep driving v0.6.11-era procedure from a v0.7.0 binary.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-08T09:27:50Z status: open -> closed

## Links
- Parent: [[MAC-ui8a]]

## Comments

### 2026-09-08T09:21:53Z ramirosalas
Fixed on fix/MAC-3iga-wcrd at db82d90, refined at 07ac215.

1. skills/machinery/SKILL.md now covers the 0.7.0 surface, inside the sections that already exist and with the version field held at 0.7.0:
- Architecture: per-edge consumer READS declarations, the consumer column, the (no reads: <reason>) waiver, one complete group per row, exact-field-set agreement across rows and machines.
- Behavior: one CLAUSES vocabulary per row, at most one RETIRED group inside the declaration, matrix prose is narrative, the declaration binds to its own machine, a guard no oracle governs owes nothing.
- Build handoff: the executable-assurance status exactly as MAC-tk0i established it (no machinery tdd, no --store or --assurance strict, no gate reads design/assurance/), plus Gt as static discovery of active references.
- Evidence and closure: the attestation v2 kinds (plan, current with an implementation subject and --impl, historical), and machinery recover with its --apply revalidation rule.
- Scale and decomposition: parent-owned relational Gt obligations under --impl.
- Upgrade discipline: installer/update convergence on the recorded plan, host plugin refresh failures as returned failures, and the windows/amd64 artifact with no installer support and no native Windows runtime guarantee.
No hook-state note was warranted: the only internal/hook change since v0.6.11 is snapshot custody finalization ordering inside the stop hook, with no consumer-facing procedure change.

2. machinery doctor now reports a stale skill. install.InstalledSkillVersion reads the release the installed SKILL.md front matter declares; doctor compares it against the running binary in both the default and --target report paths and prints one line per skill: 'stale skill at <path> is release 0.6.11; this binary is 0.7.0 -- run machinery update', recorded as a failure. Verified live on this machine: doctor now exits 1 and names both installed homes. ValidateArtifact is unchanged, so install and update paths are unaffected.

Limit, stated in the code and worth recording here: this compares declared releases, not bytes. Two skills that declare the same release but differ in content are still indistinguishable, because the binary carries no copy of the release content and the receipt records the bytes that were installed rather than the release they came from. Closing that would mean embedding the skill in the binary, which is not a small change. The case the issue's impact section names (a v0.7.0 binary over a v0.6.11 skill) is covered.

Tests: new TestInstalledSkillVersionReadsDeclaredReleaseAndFailsClosed (internal/install) and TestDoctorReportsSkillReleaseAgainstRunningBinary (cmd/machinery). Tests that pin SKILL.md content or the installed layout, all green: internal/hook TestPluginManifest (SKILL.md metadata version must equal .claude-plugin/plugin.json), internal/gates attest vocabulary, cmd/machinery install/uninstall/output-failure topology assertions, and internal/install install/update/receipt/discovery/plugin-skip layout tests. gofmt, go vet ./... and golangci-lint run ./... are clean.

Pre-existing flake, unrelated and not touched: internal/install TestBootstrapReceiptCLI/{,missing_prestate_}later_target_failure_rolls_back_bootstrap_false failed once under concurrent load because their 30s context deadline killed the CLI before it reached the OpenCode source failure; both pass in 36s each when run alone. Same class as 497419a.
