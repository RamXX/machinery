// OpenCode adapter for machinery's shared hook protocol. The deterministic
// behavior remains in the machinery binary; this file only translates native
// OpenCode events into that protocol.

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

async function runMachinery($, root, payload) {
  const failed = (detail) => ({
    decision: "block",
    reason: `Machinery governance failed closed: ${detail}. Repair or reinstall the machinery binary before continuing.`,
  })
  try {
    const result = await $`printf %s ${JSON.stringify(payload)} | machinery hook --root ${root}`
      .quiet()
      .nothrow()
    if (result.exitCode !== 0) {
      const stderr = result.stderr?.toString().trim()
      return failed(stderr || `machinery hook exited ${result.exitCode}`)
    }
    const stdout = result.stdout.toString().trim()
    if (!stdout) return null
    try {
      return JSON.parse(stdout)
    } catch {
      return failed("machinery hook returned malformed JSON")
    }
  } catch (error) {
    return failed(error?.message || "machinery binary could not be executed")
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

export const MachineryPlugin = async ({ client, $, directory, worktree }) => {
  const root = worktree || directory

  return {
    "tool.execute.before": async (input, output) => {
      const tool = toolNames[input.tool]
      if (!tool) return
      const response = await runMachinery($, root, {
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
      const response = await runMachinery($, root, {
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

      const response = await runMachinery($, root, {
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
