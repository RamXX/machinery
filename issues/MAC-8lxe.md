---
id: MAC-8lxe
title: "SKILL.md not updated for v0.7.0 features, and doctor reports a content-stale installed skill as ok"
status: open
priority: 3
type: bug
labels: [docs, skill, doctor]
parent: MAC-ui8a
created_at: 2026-09-08T07:55:53Z
created_by: ramirosalas
updated_at: 2026-09-08T07:55:53Z
content_hash: "sha256:1f5b2b0ff9837817ab7fcee3a4f9c3fafa0dbc6e105a3c0d364d90c246419eba"
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


## Links
- Parent: [[MAC-ui8a]]

## Comments
