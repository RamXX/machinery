# The machinery Claude Code plugin

machinery's methodology lives in a skill, and a skill depends on the model's willingness to follow
it. The plugin closes that gap for Claude Code: the deterministic half of the gates moves from
"instructed" to "enforced" by the harness, through hooks that do not care whether the model
remembered the methodology. The attested half (whether a guard's semantics enforce the invariant it
names, the zero-context claim) stays judgment, exactly as the gate design intends, and CI remains
the outer wall.

The repository root is the plugin. Installing it gives you the same `skills/machinery` skill and
the two role agents that `machinery install` lays into agent homes, plus slash commands and hooks.
Nothing is duplicated in the repo; there is one skill and one set of role docs.

## Install

Two steps, in either order:

```bash
# 1. the binary (the hooks and gates call it)
curl -fsSL https://raw.githubusercontent.com/RamXX/machinery/main/install.sh | sh
```

```
# 2. the plugin, inside Claude Code
/plugin marketplace add RamXX/machinery
/plugin install machinery@machinery
```

Non-Claude agents are untouched by all of this: `machinery install` keeps placing the skill and
role docs into `~/.agents` (and any `--home` you name) exactly as before. When the plugin is
detected under `~/.claude/plugins`, the default `machinery install` skips `~/.claude` with a note,
so the skill is never present twice; an explicit `--home ~/.claude` overrides the skip.

## When the hooks act, and when they are a no-op

Every hook routes through one shim, `hooks/machinery-hook.sh`, whose first act is detection:

- the project root has a `.machinery.json`, or
- the conventional `design/domain.modelith.yaml` exists.

If neither holds, the shim exits 0 before reading stdin or looking for the binary. A non-machinery
repository never sees output, never pays more than two file stats, and never conflicts with other
plugins. If the project is managed but the binary is missing, the shim warns on stderr and still
exits 0: governance degrades loudly to absent, it never breaks a session.

## What the hooks enforce

| Event | Behavior |
|---|---|
| SessionStart | Injects the governance contract into context: design dir, staged gates, the read-only artifact list, and `design/STATE.md` (the session ledger) when present. Every session in the repo knows the rules, whether or not the skill ever triggers. |
| PreToolUse | Denies Edit/Write/MultiEdit/NotebookEdit on generated artifacts: `<design>/**/*.oracle.md`, `<design>/formal/*.tla`, `*.cfg` and `*.als`, `<design>/packs/**` (generated packs), `<design>/pack/**` (the frozen pack a child was built against), `<design>/ratchet.json` (the baseline snapshot). The refusal names the regeneration command. |
| PostToolUse | Silently records that the session touched the design (or watched sources, when `impl` is configured). No gates run mid-edit; authoring stays fluid. |
| Stop / SubagentStop | If the session touched anything watched, runs `machinery check` (in-process; same suite semantics as the CLI). DRIFT findings block the stop with the gate output as the reason; the model fixes and the check re-runs. G4 import-boundary findings block only when they are ARMED: `<design>/ratchet.json` exists, written by `machinery baseline`. Before that snapshot exists, import findings warn with the arming instruction instead of blocking, because blocking a session on pre-existing boundary debt it did not create invites the model to "fix" the debt by adding allow rules, which is silent amnesty. Plain ERRORs only warn, because a half-built design is a normal interrogation state. After one blocked-and-continued attempt, the hook warns instead of blocking again, so it can never loop. |

Gate selection at stop time is progressive when no staged list is configured: Gm once
`migration.yaml` exists (rebuild/hybrid transition contract; see the
[rebuild guide](rebuild-guide.md)); Gp / Gi / Gn once the
matching `formal/{policy,integrity,isolation}.relational.yaml` exists (the relational layers; see the
[policy](policy-layer.md), [integrity](integrity-layer.md), and [isolation](isolation-layer.md)
guides), G2 once `workspace.dsl` or `ARCHITECTURE.md` exists, G3
once `machines/*.machine.json` exist, Gx once `BUILD.md` exists, G5 on decomposed designs, G4 only
when `impl` is configured. A phase you have
not reached is not demanded of you; a phase you have reached is held.

What the hooks deliberately do not do: they cannot make the interrogation good, they do not check
guard semantics, and a `sed` through the Bash tool can still touch an oracle; that lands as DRIFT
at the next stop. Users can disable hooks; the consuming repo's CI `machinery check` remains the
non-negotiable backstop (see the [brownfield team guide](brownfield-team-guide.md)).

## `.machinery.json`

Optional at the project root; `/machinery:init` writes it. Presence alone marks the repo as
machinery-managed. All fields optional:

```json
{
  "design": "design",
  "gates": "g2,g4",
  "impl": ".",
  "strict": false,
  "hooks": true
}
```

- `design`: design directory relative to the root. Default `design`.
- `gates`: a staged `--gate` list, the brownfield adoption ratchet from the
  [team guide](brownfield-team-guide.md). Empty selects gates progressively by which artifacts
  exist.
- `impl`: implementation directory for G4-import. Setting it turns on import-boundary enforcement
  for ordinary coding sessions, the "no drift" case: an undeclared cross-boundary import, or a new
  offender file on a baselined edge, blocks the turn that wrote it instead of waiting for CI.
  Requires the contract's boundaries to declare `code:` globs, and blocking arms only once
  `machinery baseline <design> --impl <dir>` has written `<design>/ratchet.json` (run it with zero
  findings on a greenfield repo; the empty snapshot is the arming marker). Until then import
  findings warn. Unset, G4 never runs from hooks.
- `strict`: block the end of any turn on ANY blocking finding, not only DRIFT and G4. Right for a
  repo whose design is complete; wrong mid-interrogation.
- `hooks`: set `false` to keep the repo marked as machinery-managed while opting out of hook
  governance entirely.

A config that fails to parse counts as managed with defaults plus a warning: a typo degrades
loudly, it does not silently disable governance.

## Slash commands

- `/machinery:design [greenfield|brownfield|rebuild|hybrid] <what>`: start or resume the four-phase conductor
  (reads `design/STATE.md` to resume).
- `/machinery:check [design-dir] [--impl d] [--gate list]`: run the gates and explain every
  finding, honoring `.machinery.json`.
- `/machinery:init [design-dir]`: mark the repo as managed and write `.machinery.json`
  (staged gates, impl, strict) after one batched question.
- `/machinery:status`: phase ledger, artifact inventory, gate health, next action.

## Layout (for contributors)

- `.claude-plugin/plugin.json`: the manifest; `.claude-plugin/marketplace.json` makes the repo
  installable via `/plugin marketplace add RamXX/machinery` with the repo root as the plugin
  source, which is how the plugin reuses `skills/` and `agents/` without copies.
- `hooks/hooks.json` + `hooks/machinery-hook.sh`: every event, one shim, detection first.
- `commands/*.md`: the four commands.
- The hook logic itself is `machinery hook` (hidden subcommand, `internal/hook`), so it is
  versioned, tested (`internal/hook/hook_test.go`, including a regression net over `hooks.json`
  and the manifests), and shares the exact gate-suite semantics with `machinery check` through
  `internal/gates.Select` / `RunSelected`. The shim maps any binary failure to a warning plus
  exit 0, so a plugin newer than the binary degrades to no governance instead of breaking tools.

Hooks load at session start: after installing or upgrading the plugin, restart the Claude Code
session in the project.
