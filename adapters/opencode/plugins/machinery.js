// OpenCode adapter for machinery's shared hook protocol. The deterministic
// behavior remains in the machinery binary; this file only translates native
// OpenCode events into that protocol.

const toolNames = {
  write: "Write",
  edit: "Edit",
  patch: "apply_patch",
  apply_patch: "apply_patch",
}

function sessionID(value) {
  return value?.sessionID ?? value?.sessionId ?? value?.properties?.sessionID ??
    value?.properties?.sessionId ?? value?.properties?.info?.id ?? "opencode"
}

function toolInput(args = {}) {
  return {
    file_path: args.filePath ?? args.file_path ?? args.path ?? "",
    command: args.command ?? args.patchText ?? args.patch ?? args.diff ?? "",
  }
}

async function runMachinery($, root, payload) {
  try {
    const result = await $`printf %s ${JSON.stringify(payload)} | machinery hook --root ${root}`
      .quiet()
      .nothrow()
    if (result.exitCode !== 0) return null
    const stdout = result.stdout.toString().trim()
    if (!stdout) return null
    try {
      return JSON.parse(stdout)
    } catch {
      return stdout
    }
  } catch {
    return null
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
        cwd: root,
        hook_event_name: "PreToolUse",
        tool_name: tool,
        tool_input: toolInput(output.args),
      })
      const reason = denial(response)
      if (reason) throw new Error(reason)
    },

    "tool.execute.after": async (input, output) => {
      const tool = toolNames[input.tool]
      if (!tool) return
      const args = input.args || output.args || {}
      await runMachinery($, root, {
        session_id: sessionID(input),
        cwd: root,
        hook_event_name: "PostToolUse",
        tool_name: tool,
        tool_input: toolInput(args),
      })
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
        // OpenCode cannot reactivate the agent from session.idle. Clear the
        // shared touched-file ledger after surfacing this advisory result so
        // every later idle event does not repeat the same stale warning.
        await runMachinery($, root, {
          session_id: id,
          cwd: root,
          hook_event_name: "Stop",
          stop_hook_active: true,
        })
      }
    },
  }
}
