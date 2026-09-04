// OpenCode adapter for machinery's shared hook protocol. The deterministic
// behavior remains in the machinery binary; this file only translates native
// OpenCode events into that protocol.
//
// Transport: node:child_process spawn, never the plugin input's Bun `$`.
// OpenCode >= 1.17 runs its plugin host under Node, where `$` is explicitly
// undefined (the published typings still declare it), so a tagged-template
// call on it threw "$ is not a function" and every governed tool call failed
// closed without the binary ever running. spawn works under Node and Bun
// alike, inherits the environment (so PATH resolves `machinery` exactly as
// the old shell pipeline did), and uses no shell. The protocol is unchanged:
// the JSON payload on stdin, argv `machinery hook --root <root>`.

import { spawn } from "node:child_process"

const toolNames = {
  write: "Write",
  edit: "Edit",
  patch: "apply_patch",
  apply_patch: "apply_patch",
  bash: "Bash",
  shell: "Bash",
}

function sessionID(value) {
  return value?.sessionID ?? value?.sessionId ?? value?.properties?.sessionID ??
    value?.properties?.sessionId ?? value?.properties?.info?.id ?? "opencode"
}

function toolUseID(value) {
  return value?.callID ?? value?.callId ?? value?.toolCallID ?? value?.toolCallId ?? value?.id ?? ""
}

function toolInput(args = {}) {
  return {
    file_path: args.filePath ?? args.file_path ?? args.path ?? "",
    command: args.command ?? args.patchText ?? args.patch ?? args.diff ?? "",
  }
}

// defaultRunner executes `machinery hook --root <root>` with the payload on
// stdin and settles exactly once: an `error` event (ENOENT, EACCES) may or may
// not be followed by `close`, so the first settlement wins.
function defaultRunner(root, payload) {
  return new Promise((resolve) => {
    let settled = false
    const settle = (value) => {
      if (settled) return
      settled = true
      resolve(value)
    }
    let child
    try {
      child = spawn("machinery", ["hook", "--root", root], { stdio: ["pipe", "pipe", "pipe"] })
    } catch (error) {
      settle({ ok: false, error })
      return
    }
    let stdout = ""
    let stderr = ""
    child.stdout.setEncoding("utf8")
    child.stderr.setEncoding("utf8")
    child.stdout.on("data", (chunk) => { stdout += chunk })
    child.stderr.on("data", (chunk) => { stderr += chunk })
    child.on("error", (error) => settle({ ok: false, error }))
    child.on("close", (exitCode) => settle({ ok: true, exitCode, stdout, stderr }))
    child.stdin.on("error", () => {
      // A child that exits before reading stdin closes the pipe; the close
      // event carries the verdict, so an EPIPE here is not a transport failure.
    })
    child.stdin.end(JSON.stringify(payload))
  })
}

async function runMachinery(run, root, payload) {
  const failed = (detail) => ({
    decision: "block",
    reason: `Machinery governance failed closed: ${detail}. Repair or reinstall the machinery binary before continuing.`,
  })
  let result
  try {
    result = await run(root, payload)
  } catch (error) {
    return failed(error?.message || "machinery binary could not be executed")
  }
  if (!result?.ok) {
    return failed(result?.error?.message || "machinery binary could not be executed")
  }
  if (result.exitCode !== 0) {
    const stderr = (result.stderr ?? "").toString().trim()
    return failed(stderr || `machinery hook exited ${result.exitCode}`)
  }
  const stdout = (result.stdout ?? "").toString().trim()
  if (!stdout) return null
  try {
    return JSON.parse(stdout)
  } catch {
    return failed("machinery hook returned malformed JSON")
  }
}

function denial(response) {
  const specific = response?.hookSpecificOutput
  if (specific?.permissionDecision === "deny") {
    return specific.permissionDecisionReason || "Blocked by machinery governance."
  }
  if (response?.decision === "block") {
    return response.reason || "Blocked by machinery governance."
  }
  return ""
}

async function recordWarning(client, message) {
  if (!message) return
  try {
    await client.tui.showToast({
      body: {
        title: "machinery gates",
        message: message.length > 1500 ? message.slice(0, 1500) + "..." : message,
        variant: "warning",
      },
    })
  } catch {
    // Headless OpenCode sessions may not expose a TUI.
  }
  try {
    await client.app.log({
      body: {
        service: "machinery",
        level: "warn",
        message,
      },
    })
  } catch {
    // Logging must never break an OpenCode session.
  }
}

// `$` is deliberately not destructured: the adapter must never depend on it.
// The runner is injectable through the options argument OpenCode passes
// through, which is how the tests drive the transport.
export const MachineryPlugin = async ({ client, directory, worktree }, options = {}) => {
  const run = options.runner ?? defaultRunner
  const root = worktree || directory

  return {
    "tool.execute.before": async (input, output) => {
      const tool = toolNames[input.tool]
      if (!tool) return
      const response = await runMachinery(run, root, {
        session_id: sessionID(input),
        tool_use_id: toolUseID(input),
        cwd: root,
        hook_event_name: "PreToolUse",
        tool_name: tool,
        tool_input: toolInput(output.args),
      })
      const failure = denial(response)
      if (failure) throw new Error(failure)
    },

    "tool.execute.after": async (input, output) => {
      const tool = toolNames[input.tool]
      if (!tool) return
      const args = input.args || output.args || {}
      const response = await runMachinery(run, root, {
        session_id: sessionID(input),
        tool_use_id: toolUseID(input),
        cwd: root,
        hook_event_name: "PostToolUse",
        tool_name: tool,
        tool_input: toolInput(args),
      })
      const failure = denial(response)
      if (failure) throw new Error(failure)
    },

    event: async ({ event }) => {
      const id = sessionID(event)
      if (event.type !== "session.idle") return

      const response = await runMachinery(run, root, {
        session_id: id,
        cwd: root,
        hook_event_name: "Stop",
      })
      await recordWarning(client, denial(response) || response?.systemMessage)
      if (response?.decision === "block") {
        // OpenCode cannot reactivate the agent from session.idle. Retain the
        // shared touched-file ledger so every later idle event remains blocked
        // until the underlying check is green, and fail this idle event rather
        // than acknowledging a stop whose deterministic checks are red.
        throw new Error(denial(response) || "Blocked by machinery governance.")
      }
    },
  }
}
