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

// Every governance spawn is bounded. The deadline must outlast the in-process
// gate runs a Stop hook performs, but it is finite so a wedged binary can
// never pin an OpenCode session forever. Each output stream gets its own
// capture ceiling: a binary that floods either pipe is terminated, not
// buffered without end. Both bounds are injectable so the native subprocess
// tests can prove termination with small budgets; production always uses the
// defaults below. Termination escalates: SIGTERM first, then SIGKILL once the
// grace expires, because a misbehaving child may swallow SIGTERM.
const hookDeadlineMs = 600000
const hookMaxOutputBytes = 1 << 20
const terminationGraceMs = 750

// defaultRunner executes `machinery hook --root <root>` with the payload on
// stdin and settles exactly once: an `error` event (ENOENT, EACCES) may or may
// not be followed by `close`, the deadline and either capture ceiling may fire
// in any interleaving with them, and whichever settlement lands first wins.
// Every owned subprocess is terminated on deadline or overflow, and all
// timers and streams are released on settlement.
function defaultRunner(root, payload, limits = {}) {
  const timeoutMs = Math.max(1, Number(limits.timeoutMs) || hookDeadlineMs)
  const maxOutputBytes = Math.max(1, Number(limits.maxOutputBytes) || hookMaxOutputBytes)
  return new Promise((resolve) => {
    let settled = false
    let deadlineTimer = null
    let killTimer = null
    let child = null
    const settle = (value) => {
      if (settled) return
      settled = true
      if (deadlineTimer !== null) clearTimeout(deadlineTimer)
      resolve(value)
    }
    // terminate releases the owned subprocess: SIGTERM, then SIGKILL after the
    // grace. The escalation timer is cleared when `close` proves the child is
    // gone, so no timer outlives the run.
    const terminate = () => {
      if (child === null) return
      try {
        child.kill("SIGTERM")
      } catch {
        // The child is already gone; `close` settles the verdict.
      }
      killTimer = setTimeout(() => {
        killTimer = null
        try {
          child.kill("SIGKILL")
        } catch {
          // Already reaped.
        }
      }, terminationGraceMs)
      try {
        child.stdin.destroy()
      } catch {
        // The write end may already be closed.
      }
    }
    // capture bounds one stream independently: once the ceiling is crossed the
    // stream is destroyed, the child is terminated, and the run settles as a
    // bounded transport failure rather than buffering the flood.
    const capture = (stream, kind) => {
      let text = ""
      let truncated = false
      stream.setEncoding("utf8")
      stream.on("data", (chunk) => {
        if (truncated) return
        if (text.length + chunk.length > maxOutputBytes) {
          truncated = true
          text = text.slice(0, maxOutputBytes)
          try {
            stream.destroy()
          } catch {
            // Destruction is best-effort; truncation is already recorded.
          }
          terminate()
          settle({
            ok: false,
            error: new Error(`machinery hook ${kind} exceeded its bounded capture limit (${maxOutputBytes} bytes) and the child was terminated`),
            pid: child === null ? undefined : child.pid,
          })
          return
        }
        text += chunk
      })
      return () => ({ text, truncated })
    }
    deadlineTimer = setTimeout(() => {
      deadlineTimer = null
      terminate()
      settle({
        ok: false,
        error: new Error(`machinery hook exceeded its ${timeoutMs}ms deadline and was terminated`),
        pid: child === null ? undefined : child.pid,
      })
    }, timeoutMs)
    try {
      child = spawn("machinery", ["hook", "--root", root], { stdio: ["pipe", "pipe", "pipe"] })
    } catch (error) {
      settle({ ok: false, error })
      return
    }
    const stdoutState = capture(child.stdout, "stdout")
    const stderrState = capture(child.stderr, "stderr")
    child.on("error", (error) => settle({ ok: false, error, pid: child.pid }))
    child.on("close", (exitCode) => {
      if (killTimer !== null) {
        clearTimeout(killTimer)
        killTimer = null
      }
      const stdout = stdoutState()
      const stderr = stderrState()
      settle({
        ok: true,
        exitCode,
        stdout: stdout.text,
        stderr: stderr.text,
        stdoutTruncated: stdout.truncated,
        stderrTruncated: stderr.truncated,
        pid: child.pid,
      })
    })
    child.stdin.on("error", () => {
      // A child that exits before reading stdin closes the pipe; the close
      // event carries the verdict, so an EPIPE here is not a transport failure.
    })
    child.stdin.end(JSON.stringify(payload))
  })
}

// The hook protocol the machinery binary actually emits: an empty stdout is
// documented success; otherwise exactly one of a permission decision envelope
// (allow or deny) or a stop decision (block) or a bare system message. The
// allow/deny values are the host protocol's; `ask` is deliberately unsupported
// because the OpenCode adapter has no interactive prompt to delegate to, so
// it fails closed. Anything else - unknown fields, combined protocols, arrays,
// other decision strings - is unrecognizable output and must block rather
// than pass, because a response the adapter cannot interpret cannot grant
// permission.
const recognizedResponseFields = new Set(["decision", "reason", "systemMessage", "hookSpecificOutput"])
const recognizedPermissionFields = new Set(["hookEventName", "permissionDecision", "permissionDecisionReason"])

function responseProblem(response) {
  if (typeof response !== "object" || response === null || Array.isArray(response)) {
    return "the response is not a hook JSON object"
  }
  const fields = Object.keys(response)
  if (fields.length === 0) return "the response carries no decision"
  for (const field of fields) {
    if (!recognizedResponseFields.has(field)) return `the response carries an unrecognized field "${field}"`
  }
  if ("decision" in response && "hookSpecificOutput" in response) {
    return "the response combines the decision and permission protocols"
  }
  if ("decision" in response) {
    if (response.decision !== "block") return `the decision "${response.decision}" is not a supported decision`
    if ("reason" in response && typeof response.reason !== "string") return "the decision reason is not a string"
  }
  if ("systemMessage" in response && typeof response.systemMessage !== "string") {
    return "systemMessage is not a string"
  }
  if ("hookSpecificOutput" in response) {
    const specific = response.hookSpecificOutput
    if (typeof specific !== "object" || specific === null || Array.isArray(specific)) {
      return "hookSpecificOutput is not an object"
    }
    const permissionFields = Object.keys(specific)
    if (permissionFields.length === 0) return "hookSpecificOutput carries no permission decision"
    for (const field of permissionFields) {
      if (!recognizedPermissionFields.has(field)) return `hookSpecificOutput carries an unrecognized field "${field}"`
    }
    if ("hookEventName" in specific && typeof specific.hookEventName !== "string") {
      return "hookEventName is not a string"
    }
    if (!("permissionDecision" in specific)) return "hookSpecificOutput carries no permission decision"
    if (specific.permissionDecision !== "allow" && specific.permissionDecision !== "deny") {
      return `the permission decision "${specific.permissionDecision}" is not supported`
    }
    if ("permissionDecisionReason" in specific && typeof specific.permissionDecisionReason !== "string") {
      return "permissionDecisionReason is not a string"
    }
  }
  return ""
}

async function runMachinery(run, root, payload) {
  const failed = (detail) => {
    const transient = detail.includes("another operation holds the lock") ||
      detail.includes("wait for active machinery install")
    const action = transient
      ? "Retry after the active machinery install, update, or uninstall finishes."
      : "Run machinery doctor; reinstall only if it reports an installation or version mismatch."
    return {
      decision: "block",
      reason: `Machinery governance failed closed: ${detail}. ${action}`,
    }
  }
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
  if (result.stdoutTruncated || result.stderrTruncated) {
    return failed("machinery hook overflowed its bounded output capture and was terminated")
  }
  const stdout = (result.stdout ?? "").toString().trim()
  if (!stdout) return null
  let response
  try {
    response = JSON.parse(stdout)
  } catch {
    return failed("machinery hook returned malformed JSON")
  }
  const problem = responseProblem(response)
  if (problem) {
    return failed(`machinery hook returned an unrecognized response (${problem})`)
  }
  return response
}

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
// through, which is how the tests drive the transport. The real spawn
// transport is exported so the native subprocess tests can hold its deadline,
// capture ceilings and termination to observed behavior.
export const MachineryPlugin = async ({ client, directory, worktree }, options = {}) => {
  const run = options.runner ?? defaultRunner
  const root = worktree || directory
  const pending = new Map()
  const permissions = new Map()
  const sessionCalls = (id) => {
    if (!pending.has(id)) pending.set(id, new Map())
    return pending.get(id)
  }
  const failedCompletion = async (id, callID, reason) => {
    const calls = pending.get(id)
    const call = calls?.get(callID)
    if (!call) return
    const response = await runMachinery(run, root, {
      session_id: id,
      tool_use_id: callID,
      cwd: root,
      hook_event_name: "PostToolUseFailure",
      tool_name: call.tool,
      tool_input: call.input,
      error: reason,
    })
    const failure = denial(response)
    if (failure) throw new Error(failure)
    calls.delete(callID)
  }

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
      sessionCalls(sessionID(input)).set(toolUseID(input), { tool, input: toolInput(output.args) })
    },

    "tool.execute.after": async (input, output) => {
      const tool = toolNames[input.tool]
      if (!tool) return
      if (input.error || output.error) {
        await failedCompletion(sessionID(input), toolUseID(input), "OpenCode tool.execute.after reported a tool error")
        return
      }
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
      pending.get(sessionID(input))?.delete(toolUseID(input))
    },

    event: async ({ event }) => {
      const id = sessionID(event)
      if (event.type === "permission.asked" || event.type === "permission.updated") {
        const permission = event.properties || {}
        if (permission.id && permission.callID) permissions.set(permission.id, { id, callID: permission.callID })
        return
      }
      if (event.type === "permission.replied") {
        const reply = event.properties || {}
        const permission = permissions.get(reply.permissionID)
        permissions.delete(reply.permissionID)
        if (permission && (reply.response === "reject" || reply.response === "deny")) {
          await failedCompletion(permission.id, permission.callID, "OpenCode permission.replied refused the tool call")
        }
        return
      }
      if (event.type !== "session.idle") return

      // OpenCode's idle event closes its synchronous file-tool turn. A shell
      // can leave an external writer running after idle; its missing after
      // event is not a reliable denial witness. Keep that token armed until
      // OpenCode reports an exact permission reply or tool completion.
      for (const [callID, call] of [...(pending.get(id)?.entries() || [])]) {
        if (call.tool !== "Bash") {
          await failedCompletion(id, callID, "OpenCode session.idle after file-tool execute.before without execute.after")
        }
      }

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

export { defaultRunner }
