# Agent portability

machinery has one portable methodology and thin host adapters. The portable core is the
`skills/machinery` skill, the canonical bodies of the two author roles in `agents/`, the design
artifact formats, and the `machinery` binary. Claude Code, Codex, OpenCode, and other Agent Skills
runtimes must produce and accept the same design. Host integration changes ergonomics and how early
a problem is reported; it does not change the gates or the definition of a valid blueprint.

## Compatibility contract

The following are invariants across every host:

- The skill is the conductor and the CLI is the deterministic authority. No adapter reimplements a
  gate.
- Phase 3 and Phase 4 use one canonical role body each. Installation renders host-native wrappers
  from those bodies; it does not maintain separate prompts.
- Missing subagent support falls back to executing the same role inline. Missing commands fall back
  to a plain-language request. Missing lifecycle hooks fall back to an explicit `machinery check`
  before each handoff and the consuming repository's CI gate.
- Generated artifacts stay read-only. An edit hook may reject a write immediately, but a clean
  `machinery check` is the host-independent acceptance test.
- Host adapters do not pin a model. The user or host chooses the model; machinery fixes the inputs,
  outputs, and checks.

This is deliberately capability-based. A future agent that discovers `~/.agents/skills` can use the
methodology immediately even if machinery has no named adapter for it. It will run the role work
inline and use the CLI gates until a thin adapter adds native subagents or lifecycle hooks.

## Install paths

The existing command remains backward-compatible:

```bash
machinery install
```

It places a real canonical copy under `~/.agents` and symlinks the same skill and role docs into
`~/.claude`. Existing scripts using repeatable `--home`, `--copy`, or `--from` keep that behavior.

Host-aware installation is opt-in:

```bash
machinery install --target claude
machinery install --target codex
machinery install --target opencode
machinery install --target all

machinery doctor --target all
machinery uninstall --target all
```

`--target` and `--home` are mutually exclusive so there is no ambiguous topology. The bootstrap
installer exposes the same mode:

```bash
MACHINERY_TARGETS="codex opencode" \
  curl -fsSL https://raw.githubusercontent.com/RamXX/machinery/main/install.sh | sh
```

A single-host Codex or OpenCode uninstall removes that host's native adapter and preserves the
shared `~/.agents` skill because another Agent Skills runtime may still use it. A complete
`--target all` removal also removes the shared copy.

## Updating a release

`machinery update` is a forced, topology-aware refresh:

```bash
machinery update
machinery update --version v0.3.4
```

It performs the following contract in order:

1. Resolve `latest` or the exact requested release tag.
2. Download the platform binary and `checksums-sha256.txt`, reject any checksum mismatch, make the
   candidate executable, and require `machinery version` to report the requested tag.
3. Load the installation receipt. If no receipt exists, discover the legacy `~/.agents` plus
   `~/.claude` topology and native Codex/OpenCode adapters at their standard paths.
4. Download and extract the source archive for that same tag before changing the installed binary.
5. Atomically replace the binary, then invoke the new binary's `install --from <staged-source>` for
   every recorded home group and native target. This lets a new release own its installation format
   instead of asking the old updater to understand future wrappers.
6. Ask detected Claude Code and Codex CLIs to refresh their machinery plugins. Plugin cache files
   are never edited directly.

The command intentionally performs the work even when the requested tag equals the installed
version. Explicit selectors restrict only the harness refresh while the binary still updates:

```bash
machinery update --home ~/.agents --target codex
machinery update --target all --copy
machinery update --skip-plugins
machinery update --install-dir ~/.local/bin
```

`--home` and `--target` may be combined here because an existing installation can legitimately
contain both custom portable homes and native host adapters. Native target placement runs last and
is authoritative when an explicit custom home overlaps a standard host path.

All downloads and the direct-install plan are staged before binary replacement. If checksum,
candidate-version, receipt, or source download validation fails, the existing binary is untouched.
If a direct harness refresh fails after replacement, the new binary remains installed and update
returns a non-zero, retryable error naming the failed install command. Host plugin refresh failures
are warnings because managed scopes and host policy may forbid them; rerun the printed host command
or use `--skip-plugins` deliberately.

Successful `machinery install` calls maintain the receipt, and successful `machinery uninstall`
calls remove the corresponding entries. Single-target removal keeps the shared Agent Skills copy;
complete removal deletes the empty receipt.

## What each target installs

| Target | Shared skill | Native roles | Commands | Governance |
|---|---|---|---|---|
| Claude Code | `~/.claude/skills/machinery` | `~/.claude/agents/*.md` | Provided by the repository plugin | Plugin hooks can deny generated-file edits and block stop on applicable red gates |
| Codex | `~/.agents/skills/machinery` | `~/.codex/agents/*.toml` | Use the skill directly | The repository is a Codex plugin through `.codex-plugin/plugin.json`; its shared hook configuration handles Codex `apply_patch`, including multi-file patches |
| OpenCode | `~/.agents/skills/machinery` | `~/.config/opencode/agents/*.md` | `~/.config/opencode/commands/{design,check,init,status}.md` | `~/.config/opencode/plugins/machinery.js` denies generated-file edits, records touched files, and reports idle checks |
| Other Agent Skills host | `~/.agents/skills/machinery` | Inline fallback unless the host understands a role format | Plain-language request | Explicit checks plus CI |

The role renderers remove Claude-specific frontmatter before writing Codex TOML or OpenCode YAML.
The canonical instruction body is byte-for-byte the body from `agents/`; model and tool metadata do
not leak between hosts.

## Claude Code

For the full Claude experience, install the binary and the repository plugin as described in the
[Claude Code plugin guide](claude-plugin.md). The plugin supplies commands and lifecycle hooks in
addition to the skill and author roles. `machinery install --target claude` is useful when plugin
installation is not available; it installs the skill and roles, while explicit CLI checks and CI
provide enforcement.

## Codex

The repository carries `.codex-plugin/plugin.json` and reuses the same `skills/` and `hooks/`
directories as Claude Code. The hook shim discovers the Git root when `CLAUDE_PROJECT_DIR` is absent,
and `machinery hook` understands both direct file paths and Codex's `apply_patch` payload. A patch is
denied if any file in the patch is generated, and every touched design or implementation path is
recorded for the stop gate.

Run `machinery install --target codex` to install the shared skill and render the two canonical roles
as native Codex TOML agents. Install the repository as a Codex plugin when you also want lifecycle
hook governance. Restart or open a new task after installing or upgrading a plugin so its skill and
hooks are reloaded.

## OpenCode

OpenCode natively discovers `~/.agents/skills`, so the shared skill needs no translation. The target
installer renders the role wrappers, installs four native commands, and installs a dependency-free
JavaScript plugin. The plugin translates `write`, `edit`, and `apply_patch` calls into machinery's
shared hook protocol. In particular, it forwards OpenCode's `patchText`, whose paths are embedded in
`*** Add/Update/Move/Delete File` markers.

The adapter can reject a generated-artifact edit synchronously in `tool.execute.before`. It also
runs the stop check on `session.idle`, but OpenCode's event API does not provide the same reliable
"block this stop and force another agent turn" contract as Claude Code and Codex. A red idle check is
therefore surfaced in a warning toast and the application log, then its touched-file ledger is
cleared so later idle events do not repeat stale results. This is an ergonomics difference, not a
correctness exception: `machinery check` in CI remains authoritative for every host.

OpenCode also lacks an equivalent guaranteed SessionStart context-injection hook. The native Agent
Skills discovery and the installed `/design` command load the portability contract instead; after
compaction, the agent must reload the skill when it resumes machinery work. The adapter does not
pretend an application log message is model context.

Restart OpenCode after installing or upgrading the adapter so the global plugin and commands are
reloaded.

## Enforcement layers

Use three layers rather than treating a plugin hook as the proof:

1. The skill tells every agent which files are generated and when to run a gate.
2. A host adapter rejects invalid edits early and runs inner-loop checks where its API permits.
3. The repository CI runs `machinery check` and, when configured, `machinery verify-formal` against
   committed artifacts. This is the non-bypassable merge boundary.

That layering is what makes mixed-agent teams safe. A Claude session can create a design, a Codex
task can revise it, and an OpenCode session can implement it without changing the contract or
depending on every host having identical lifecycle APIs.

## Adding another host

A new adapter should remain thin:

1. Reuse `skills/machinery/SKILL.md`; do not fork it.
2. Render native roles from the body of `agents/machinery-fsm-author.md` and
   `agents/machinery-build-writer.md`; do not copy their prompt bodies into source-controlled host
   files.
3. Translate native edit events into `machinery hook` when possible. Include all paths in a
   multi-file edit.
4. Treat stop enforcement as optional capability and document whether it blocks, warns, or is
   unavailable.
5. Add the host to `machinery install --target`, `doctor --target`, and `uninstall --target`, then
   test the rendered topology and protocol fixtures.
6. Keep CI as the final authority and verify the existing no-target installer is unchanged.

The implementation lives in `internal/install/targets.go`; the shared protocol lives in
`internal/hook`. The receipt and forced updater live in `internal/install/receipt.go` and
`internal/install/update.go`. Host source adapters belong under `adapters/<host>/`.
