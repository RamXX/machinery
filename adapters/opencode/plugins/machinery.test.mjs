import assert from "node:assert/strict"
import { readFile } from "node:fs/promises"
import test from "node:test"

const source = await readFile(new URL("./machinery.js", import.meta.url), "utf8")
const moduleURL = `data:text/javascript;base64,${Buffer.from(source).toString("base64")}`
const { MachineryPlugin } = await import(moduleURL)

// A fake runner stands in for the spawn transport: the plugin is constructed
// with NO `$` anywhere, exactly like OpenCode's Node plugin host provides it.
function fakeRunner({ ok = true, exitCode = 0, stdout = "", stderr = "", error } = {}, calls = []) {
  return async (root, payload) => {
    calls.push({ root, payload })
    if (!ok) return { ok: false, error }
    return { ok: true, exitCode, stdout, stderr }
  }
}

async function afterHandler(result, calls = []) {
  const plugin = await MachineryPlugin(
    { client: {}, directory: "/project", worktree: "" },
    { runner: fakeRunner(result, calls) },
  )
  return plugin["tool.execute.after"]
}

const postInput = { tool: "write", sessionID: "session-1", args: { path: "design/BUILD.md" } }

test("PostToolUse throws when machinery exits nonzero", async () => {
  const after = await afterHandler({ exitCode: 17, stderr: "ledger write failed" })
  await assert.rejects(() => after(postInput, {}), /ledger write failed/)
})

test("PostToolUse throws when machinery returns malformed JSON", async () => {
  const after = await afterHandler({ stdout: "{not-json" })
  await assert.rejects(() => after(postInput, {}), /malformed JSON/)
})

test("PostToolUse propagates an explicit block decision", async () => {
  const after = await afterHandler({ stdout: JSON.stringify({ decision: "block", reason: "state ledger refused" }) })
  await assert.rejects(() => after(postInput, {}), /state ledger refused/)
})

test("shell tools participate in PostToolUse governance", async () => {
  const after = await afterHandler({ exitCode: 17, stderr: "shell ledger failed" })
  await assert.rejects(() => after({ tool: "bash", sessionID: "session-shell", args: { command: "touch design/BUILD.md" } }, {}), /shell ledger failed/)
})

test("plugin constructs without a Bun $ and routes governed calls through the runner", async () => {
  // The production regression: OpenCode's Node plugin host passes `$` as
  // undefined, and the old tagged-template transport threw "$ is not a
  // function" before the machinery binary ever ran.
  const calls = []
  const plugin = await MachineryPlugin(
    { client: {}, directory: "/project", worktree: "" },
    { runner: fakeRunner({ stdout: "" }, calls) },
  )
  await plugin["tool.execute.before"]({ tool: "write", sessionID: "session-node", callID: "call-1" }, { args: { filePath: "design/BUILD.md" } })
  assert.equal(calls.length, 1)
  assert.equal(calls[0].root, "/project")
  assert.equal(calls[0].payload.hook_event_name, "PreToolUse")
  assert.equal(calls[0].payload.tool_name, "Write")
  assert.equal(calls[0].payload.tool_use_id, "call-1")
  assert.equal(calls[0].payload.tool_input.file_path, "design/BUILD.md")
})

test("a spawn failure fails closed with the transport error", async () => {
  const after = await afterHandler({ ok: false, error: new Error("spawn machinery ENOENT") })
  await assert.rejects(() => after(postInput, {}), (err) => {
    assert.match(err.message, /ENOENT/)
    assert.match(err.message, /Machinery governance failed closed/)
    return true
  })
})
