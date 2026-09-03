import assert from "node:assert/strict"
import { readFile } from "node:fs/promises"
import test from "node:test"

const source = await readFile(new URL("./machinery.js", import.meta.url), "utf8")
const moduleURL = `data:text/javascript;base64,${Buffer.from(source).toString("base64")}`
const { MachineryPlugin } = await import(moduleURL)

function shellResult({ exitCode = 0, stdout = "", stderr = "" }) {
  const result = {
    exitCode,
    stdout: Buffer.from(stdout),
    stderr: Buffer.from(stderr),
  }
  return () => ({
    quiet() { return this },
    nothrow() { return Promise.resolve(result) },
  })
}

async function afterHandler(result) {
  const plugin = await MachineryPlugin({
    client: {},
    $: shellResult(result),
    directory: "/project",
    worktree: "",
  })
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
