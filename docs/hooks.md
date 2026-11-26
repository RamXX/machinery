# Hooks

The Claude Code plugin wires three hooks, in `hooks/hooks.json`, to one command:
`machinery hook <event>`. The command reads the hook's JSON on stdin and answers with hook JSON on
stdout, so the logic lives in the tool and the plugin file only names the events. The contract
is `docs/formats.md` section 21.1.

| Hook event | When | What happens |
|---|---|---|
| `SessionStart` | A session starts, resumes, clears or compacts. | The model gets the status summary: the red line, the ids, the proofs awaiting, the open intakes and questions. The session's start (HEAD) is recorded and `MACHINERY_SESSION` exported. It never blocks. |
| `PreToolUse` on `Bash` | Before `git commit` or `git push`. | `machinery check --base <backlog.base, else main>` runs; on findings the command is refused and the model reads the first 40 of them. A push is also refused while the integration branch is red, with the marker and the repair path. Any other command: no opinion. |
| `Stop` | The model ends its turn. | When files changed this session, `machinery check` runs. Its findings are shown to you (warn, the default) or, in block mode, the turn goes on once with the findings in front of the model. |

## Setup

1. Put the `machinery` executable on `PATH`: from a checkout, `go install ./cmd/machinery`.
2. Install the plugin. Its hooks load from `hooks/hooks.json` at the plugin root.
3. Optionally set the modes in `machinery.yaml`; these are the defaults:

```yaml
hooks:
  commit: block   # block | warn | off
  push: block
  stop: warn
```

`warn` hands the findings over without stopping anything; `off` leaves that point alone.

## What the hooks are not

- Not the gates. `make check`, `machinery landing` and CI still fail closed where they run; a
  hook only brings the same check to the moment an agent acts.
- Never a failure of the harness. When machinery cannot run (a broken `machinery.yaml`, stdin
  that is not hook JSON), a message says why and the action goes ahead. When the executable is
  missing, Claude Code shows a hook error notice and the action goes ahead.
- Best effort on what counts as a git command: the `if` filter and the tool read the command
  line, not the shell. `sh -c "git commit"` is not seen.

## Sessions and the verifier

The session-start hook appends `export MACHINERY_SESSION=<session id>` to `CLAUDE_ENV_FILE`, so
the Bash commands of the session carry it. `machinery proof --post`, `predict` and `accept`
record it. A subagent belongs to the session that started it: a verifier run as a subagent of the
builder's session shares its session id. So the commands also record `MACHINERY_AGENT` when it is
set, a name for the worker context within the session. The brief of a builder or a verifier says
which value to export (`export MACHINERY_AGENT=<story id>-<role>`), and the planner gives each
worker a distinct one. The story gate refuses a verifier whose prediction has the builder's session
and the builder's agent. The same session with different agents passes. The same session with an
agent missing on either line passes with a warning, and so does a harness that sets no session:
independence then rests on the actor names.

## Other harnesses

Any harness can call `machinery hook <event>` with JSON of the same shape (`session_id`, `cwd`,
`tool_name` and `tool_input.command`, `stop_hook_active`) and read the same fields back. The
Codex plugin wires no hooks.
