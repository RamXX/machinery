# The machinery Claude Code plugin

machinery's methodology lives in a skill, and a skill depends on the model's willingness to follow
it. The plugin closes that gap for Claude Code: the deterministic half of the gates moves from
"instructed" to "enforced" by the harness, through hooks that do not care whether the model
remembered the methodology. The attested half (whether a guard's semantics enforce the invariant it
names, the zero-context claim) stays judgment, exactly as the gate design intends, and CI remains
the outer wall.

The repository root is the plugin. Installing it gives you the same `skills/machinery` skill and
the two role agents that `machinery install` lays into agent homes, plus slash commands and hooks.
Nothing is duplicated in the repo; there is one skill and one set of role docs. Codex reuses the
same skill and hook implementation, while OpenCode uses a thin event translator. See the
[agent portability guide](agent-portability.md) for the cross-host contract and feature matrix.

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

The no-target `machinery install` keeps placing the skill and role docs into `~/.agents` (and any
`--home` you name) exactly as before. When the plugin is detected under `~/.claude/plugins`, the
default install skips `~/.claude` with a note, so the skill is never present twice; an explicit
`--home ~/.claude` overrides the skip. Codex and OpenCode users can opt into native wrappers with
`machinery install --target codex|opencode|all`.

For later releases, `machinery update` refreshes the binary and every recorded direct agent home
or adapter. If this Claude plugin is installed, it also runs the supported host-owned flow
(`claude plugin marketplace update machinery`, then `claude plugin update machinery@machinery`)
instead of writing `~/.claude/plugins/cache` itself. Run `/reload-plugins` afterward to activate the
new skill, agents, commands, and hooks in the current session. Managed plugin scopes may refuse the
refresh; update reports that as a warning with the command an administrator can run.

## When the hooks act, and when they are a no-op

Every hook routes through `hooks/machinery-hook.sh`. When the binary is available, the shim always
dispatches and the Go hook determines whether the project is managed. This lets a Post or Stop event
recover the immutable route and project-wide dirty obligation recorded before a shell command, even
if that command deleted both mutable project markers. A genuinely unmanaged project still produces
no output.

When the binary is unavailable, the shim searches the physical working directory and every real
ancestor for `.machinery.json` or `design/domain.modelith.yaml`. It exits silently only when that
entire path is unmanaged. A marker found at any ancestor blocks with exit 2; missing `HOME` does not
change either result. A failing binary also blocks, because version skew or broken durable state
must never silently remove governance.

## What the hooks enforce

| Event | Behavior |
|---|---|
| SessionStart | Injects the governance contract into context: design dir, staged gates, the read-only artifact list, and `design/STATE.md` (the session ledger) when present. Every session in the repo knows the rules, whether or not the skill ever triggers. |
| PreToolUse | Denies file or shell operations that name generated artifacts: `<design>/**/*.oracle.md`, `<design>/formal/*.tla`, `*.cfg` and `*.als`, `<design>/packs/**` (generated packs), `<design>/pack/**` (the frozen pack a child was built against), `<design>/ratchet.json` (the baseline snapshot), and `.machinery.json` itself (an agent edit there could switch governance off or reroute the gates; a human maintains it). Also denies creating or editing the wave sentinel `.machinery-wave` anywhere in the repository. Before an allowed operation starts, it durably records an exact `tool_use_id` token and the immutable session route. Shell commands conservatively arm both the design and configured implementation obligations because their dynamically computed targets cannot be reconstructed from command text. The refusal names the regeneration command. |
| PostToolUse / PostToolUseFailure | Durably closes only the matching `tool_use_id` while retaining the project-wide design or implementation obligation. Success and failure are both terminal tool events; neither runs gates mid-edit. A missing terminal event leaves the token in flight and Stop cannot discharge it. |
| Stop / SubagentStop | Refuses to discharge while an exact tool token or host-reported background task remains. Otherwise, if the project touched anything watched, runs `machinery check` (in-process; same suite semantics as the CLI). DRIFT findings block the stop with the gate output as the reason; the model fixes and the check re-runs. G4 import-boundary findings block only when they are ARMED: `<design>/ratchet.json` exists, written by `machinery baseline`. Before that snapshot exists, import findings warn with the arming instruction instead of blocking, because blocking a session on pre-existing boundary debt it did not create invites the model to "fix" the debt by adding allow rules, which is silent amnesty. Plain ERRORs only warn, because a half-built design is a normal interrogation state. The exact canonical wave sentinel content `open` defers red gates as a message while retaining the obligation; deleting it closes the wave, and malformed sentinel state gates normally with a diagnostic. No clock, mtime, or host identity participates in that decision. |

The durable store records its own first initialization in a private home-rooted location outside the
configuration and state parent. Its first creation is safe and silent. If the state parent later
disappears, every managed event fails closed instead of recreating an empty store; a crash or cleanup
therefore cannot erase outstanding work.

Gate selection at stop time is progressive when no staged list is configured: Gm once
`migration.yaml` exists (rebuild/hybrid transition contract; see the
[rebuild guide](rebuild-guide.md)); Gs once `legacy/surface.yaml` exists (the surface ledger; see
the [surface ledger guide](surface-ledger.md)); Gp / Gi / Gn once the
matching `formal/{policy,integrity,isolation}.relational.yaml` exists (the relational layers; see the
[policy](policy-layer.md), [integrity](integrity-layer.md), and [isolation](isolation-layer.md)
guides), G2 once `workspace.dsl` or `ARCHITECTURE.md` exists, G3
once `machines/*.machine.json` exist, Gx once the domain model and machines both exist, Gb once
`BUILD.md` exists, Ga once `acceptance/` exists or a milestone is marked `Status: closed` (see the
[milestone acceptance guide](acceptance-gate.md)), Gv once `attestations.yaml` exists (the
committed attested-claim rows; see the [attestation evidence guide](attestation-evidence.md), and
note that the stop hook is exactly where staleness must surface, since the turn that edited a
covered artifact is the turn that invalidated the judgment over it), G5 on decomposed designs, G4 and Gt only
when `impl` is configured. A phase you have
not reached is not demanded of you; a phase you have reached is held.

A machine-less decomposed parent can still own Policy or Isolation test obligations. With
`--impl`, `machinery check`'s default selection retains Gt when the parent has either a matching
source annotation or a committed decision oracle. Deleting a generated oracle does not disarm
its source annotation: Gt remains selected, and Gp or Gn reports the missing required output.
The hook also runs Gt whenever `impl` is configured, including for parent-owned relational
obligations.

For a genuinely obligation-free machine-less parent, the CLI's default selection skips Gt with
the note "gt skipped: no machines", while the hook with `impl` configured runs Gt and reports
"0 machines" on its checked line. These describe an empty coverage obligation only when the
parent also has no relational obligations; zero machines alone does not establish that.

Ga is the same shape: the hook runs it, but never binds a commit, because mid-turn the commit the
review ran on does not exist yet. The gate says so on its own note line, and CI (which passes
`--commit`) is where that binding is actually held.

What the hooks deliberately do not do: they cannot make the interrogation good and they do not check
guard semantics. PreToolUse rejects shell commands that visibly name protected targets, but shell
syntax can assemble paths dynamically. Every allowed shell operation is therefore armed before it
runs; an unobserved mutation cannot become an untouched Stop, and generated-artifact changes still
surface as DRIFT. Users can disable hooks only through a valid operator-owned configuration; the
consuming repo's CI `machinery check` remains the
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
  "hooks": true,
  "dialog": "plain"
}
```

- `design`: design directory relative to the root. Default `design`.
- `gates`: a staged `--gate` list, the brownfield adoption ratchet from the
  [team guide](brownfield-team-guide.md). Empty selects gates progressively by which artifacts
  exist.
- `impl`: implementation directory for the impl-facing gates, G4-import and Gt-tests. Setting it
  turns on import-boundary enforcement
  for ordinary coding sessions, the "no drift" case: an undeclared cross-boundary import, or a new
  offender file on a baselined edge, blocks the turn that wrote it instead of waiting for CI.
  Requires the contract's boundaries to declare `code:` globs, and blocking arms only once
  `machinery baseline <design> --impl <dir>` has written `<design>/ratchet.json` (run it with zero
  findings on a greenfield repo; the empty snapshot is the arming marker). Until then import
  findings warn. Unset, G4 and Gt never run from hooks.
- `strict`: block the end of any turn on ANY blocking finding, not only DRIFT and G4. Right for a
  repo whose design is complete; wrong mid-interrogation.
- `hooks`: set `false` to keep the repo marked as machinery-managed while opting out of hook
  governance entirely.
- `dialog`: set `"plain"` for a repo whose design conversations face users who should not see
  machinery internals. The user-facing hook messages (the end-of-turn notices) switch to plain
  language, and the session-start context reminds the conductor to hold the skill's dialog
  register. Model-facing text (deny reasons, block reasons, the governance contract) keeps full
  machinery vocabulary in both modes; the conductor translates it at relay time. Blocking
  behavior never changes. Default: the operator strings.

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

- `.claude-plugin/plugin.json`: the Claude manifest; `.claude-plugin/marketplace.json` makes the repo
  installable via `/plugin marketplace add RamXX/machinery` with the repo root as the plugin
  source, which is how the plugin reuses `skills/` and `agents/` without copies.
- `.codex-plugin/plugin.json`: the Codex manifest; it points at the same `skills/`, while Codex
  discovers the same `hooks/hooks.json` by convention.
- `hooks/hooks.json` + `hooks/machinery-hook.sh`: every event, one shared Claude/Codex shim. The shim
  uses `CLAUDE_PROJECT_DIR` when present and otherwise passes its physical working directory to the
  Go hook, which ascends real parents to the nearest marker or durable dirty project identity without
  consulting ambient Git. If the binary is missing, the shim performs the ancestor marker search
  itself and fails closed for a managed ancestor.
- `commands/*.md`: the four commands.
- `adapters/opencode/`: native OpenCode command wrappers and the event translator; the gate logic is
  not duplicated there.
- The hook logic itself is `machinery hook` (hidden subcommand, `internal/hook`), so it is
  versioned, tested (`internal/hook/hook_test.go`, including a regression net over `hooks.json`
  and the manifests), and shares the exact gate-suite semantics with `machinery check` through
  `internal/gates.Select` / `RunSelected`. A missing or failing binary in a managed project blocks
  with a nonzero exit so a broken guard cannot silently become no governance.

## The durable state store and its retention

The obligation a governed edit arms has to outlive the process that armed it: a crash, a reboot, or a
replacement session must still owe the stop-time gate run. It therefore lives outside the repository,
in one per-user store under the user config directory (`machinery-hook-state-<home digest>/`), keyed
by canonical project root: one ledger per root, plus one small route snapshot per host session that
records which routing configuration the obligation was armed under. A successful stop-time check
removes both.

Every inventory of that store is bounded, and past 4096 entries the bound fails closed, which is
correct for one store and catastrophic for the machine: with the store full, every governed shell and
write tool in every repository is blocked. A store only grows through obligations nothing ever
discharges, and a heavy local test sweep produces them by the thousand, one per temporary project
root it creates, governs, and deletes.

Retention keeps the store bounded on the write path, with two constants (`internal/hook/retention.go`):

- at most 8 route snapshots per project root, newest first;
- a store-wide retention ceiling of 512 entries, an eighth of the fail-closed limit, above which an
  arming write compacts the store back to 256.

Compaction reclaims whole generations, oldest first, and only those it can prove are dead: the ledger
records its project root, and that root no longer exists on disk. A generation whose root still
exists, whose ledger is absent, unparseable, or predates the recorded root, or which holds durable
crash evidence, is never reclaimed, so compaction can shrink the store but can never discharge a live
obligation. Reclamation runs in bounded batches under a store-wide lock, so bulk removal never
outlasts the bounded enumeration retry of a governed event in another repository. The bound costs an
arming event two whole-store enumerations, which is a few milliseconds at the retention ceiling and
is the price of the ceiling being a property of every write.

Self-repair covers ledgers this version wrote, and only those. A store at or above the fail-closed
limit whose entries carry a recorded project root heals itself: the first inventory that hits the
limit compacts once under a repair-mode ceiling and retries. A store filled before the upgrade does
not, and cannot: a ledger written by machinery 0.7.0 or older carries no project root, so nothing can
tell its obligation from a live one, and compaction retains every one of them. That store stays
failed closed until a human removes the pre-upgrade `<digest>.state` files (never the directory, see
the downgrade note below), which is the one-time remediation for the upgrade and the only case that
still needs it. Whichever kind of store it is, if it remains over the limit the hook keeps failing
closed and its diagnostic names the store, the counts, and `machinery doctor --repair`, which
compacts the same way from the command line and works at or above the limit. `machinery doctor`
alone only reports, and it reports a nonzero exit when the store is at or above the limit or cannot
be inspected.

Two residuals are stated rather than hidden. An absent project root is indistinguishable from one on
detached or unmounted storage, so an obligation for a project on a disconnected volume can be
reclaimed while the store is over its ceiling. That is a fail-open, not a deferral: the next governed
edit in that project re-arms the whole-tree obligation, but if the volume returns and the session
ends with no edit before Stop, that session's gate does not run at all. It stays narrow, since the
root must vanish while a session is live and the store must be over its ceiling at that moment, and
oldest-first reclamation reaches a freshly armed obligation last. And the ledger carries its project
root from this version on, so a binary older than this one reads such a ledger as noncanonical and
fails closed on it: a downgrade that meets a newer ledger blocks until the affected `<digest>.state`
files are removed. Remove those files, never the store directory, whose initialization marker is what
tells a first run from a lost store.

Hooks load at session start: after installing or upgrading the plugin, restart the Claude Code
session in the project.
