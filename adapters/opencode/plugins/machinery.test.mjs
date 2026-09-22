import assert from "node:assert/strict"
import { execFile } from "node:child_process"
import { chmod, mkdir, mkdtemp, readFile, readdir, rm, writeFile } from "node:fs/promises"
import os from "node:os"
import path from "node:path"
import test from "node:test"

const source = await readFile(new URL("./machinery.js", import.meta.url), "utf8")
const moduleURL = `data:text/javascript;base64,${Buffer.from(source).toString("base64")}`
const { MachineryPlugin, defaultRunner } = await import(moduleURL)

// The native-subprocess lane drives the real spawn transport: adversarial
// `machinery` executables are provisioned into isolated PATH fixtures, so the
// plugin's default runner resolves them exactly as production resolves the
// real binary. Nothing here may skip when a runtime is missing: the explicit
// Node runtime and the real built binary are prerequisites, not options.
const repoRoot = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..", "..", "..")
const scratch = await mkdtemp(path.join(os.tmpdir(), "machinery-opencode-tests-"))
const fixtures = {}
const fixtureScripts = {
  // Never answers. Obeys SIGTERM. Self-caps so an unbounded runner still
  // terminates within a bounded test window.
  hang: `import process from "node:process"
process.stdin.resume()
setTimeout(() => process.exit(0), 8000)
`,
  // Never answers and swallows termination signals; only escalation kills it.
  // Self-cap keeps an unbounded runner from hanging the suite forever.
  ignore: `import process from "node:process"
process.on("SIGTERM", () => {})
process.on("SIGINT", () => {})
process.stdin.resume()
setTimeout(() => process.exit(0), 8000)
`,
  // Floods stdout past any bounded capture, ignoring SIGTERM, then idles with
  // a self-cap exit so the suite always reaps it.
  "flood-stdout": `import process from "node:process"
process.on("SIGTERM", () => {})
process.on("SIGINT", () => {})
const chunk = "x".repeat(65536)
let left = 2 * 1024 * 1024
const pump = () => {
  if (left <= 0) { setTimeout(() => process.exit(0), 6000); return }
  left -= chunk.length
  process.stdout.write(chunk, () => {})
  setTimeout(pump, 5)
}
pump()
`,
  // Same flood on stderr: the two capture ceilings are independent.
  "flood-stderr": `import process from "node:process"
process.on("SIGTERM", () => {})
process.on("SIGINT", () => {})
const chunk = "y".repeat(65536)
let left = 2 * 1024 * 1024
const pump = () => {
  if (left <= 0) { setTimeout(() => process.exit(0), 6000); return }
  left -= chunk.length
  process.stderr.write(chunk, () => {})
  setTimeout(pump, 5)
}
pump()
`,
  // Exits nonzero immediately without reading stdin (forces the stdin EPIPE
  // race against the close verdict).
  "exit-7": `import process from "node:process"
process.stderr.write("boom: ledger exploded\\n", () => process.exit(7))
`,
  // Valid JSON transport, invalid payload: not JSON at all.
  garbage: `import process from "node:process"
process.stdout.write("{not-json", () => process.exit(0))
`,
  // Valid JSON, wrong shape: an unknown field the hook protocol never emits.
  "wrong-shape": `import process from "node:process"
const body = JSON.stringify({ unexpected: true })
process.stdout.write(body, () => process.exit(0))
`,
  // The documented empty-success protocol: exit 0, no output at all.
  silent: `import process from "node:process"
process.exit(0)
`,
  // A documented allow decision round trip through a real child.
  allow: `import process from "node:process"
const body = JSON.stringify({ hookSpecificOutput: { hookEventName: "PreToolUse", permissionDecision: "allow", permissionDecisionReason: "" } })
process.stdout.write(body, () => process.exit(0))
`,
}

test.before(async () => {
  for (const [name, script] of Object.entries(fixtureScripts)) {
    const dir = path.join(scratch, `bin-${name}`)
    await mkdir(dir, { recursive: true })
    await writeFile(path.join(dir, "machinery"), `#!/usr/bin/env node\n${script}`, "utf8")
    await chmod(path.join(dir, "machinery"), 0o755)
    fixtures[name] = dir
  }
  fixtures.absent = await mkdir(path.join(scratch, "bin-absent"), { recursive: true })
})

test.after(async () => {
  await rm(scratch, { recursive: true, force: true })
})

// Runs fn with the PATH fixture resolved first, restoring the saved PATH even
// on assertion failure. Tests in this file execute serially, which the
// sequential default of node:test guarantees inside one file.
async function withPath(dir, fn) {
  const saved = process.env.PATH
  process.env.PATH = dir + path.delimiter + saved
  try {
    return await fn()
  } finally {
    process.env.PATH = saved
  }
}

function hookPayload() {
  return {
    session_id: "session-native",
    tool_use_id: "call-native",
    cwd: "/project",
    hook_event_name: "PreToolUse",
    tool_name: "Edit",
    tool_input: { file_path: "design/BUILD.md", command: "" },
  }
}

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

// Polls until the child is observable as dead (kill(pid, 0) throws ESRCH),
// proving the runner did not retain an owned subprocess.
async function reaped(pid, withinMs = 4000) {
  const deadline = Date.now() + withinMs
  for (;;) {
    try {
      process.kill(pid, 0)
    } catch {
      return true
    }
    if (Date.now() >= deadline) return false
    await sleep(50)
  }
}

// A fake runner stands in for the spawn transport: the plugin is constructed
// with NO `$` anywhere, exactly like OpenCode's Node plugin host provides it.
// Unknown result fields (for example the truncation flags) pass through so
// injected cases exercise the same result surface the real runner reports.
function fakeRunner(config = {}, calls = []) {
  const { ok = true, exitCode = 0, stdout = "", stderr = "", error, ...rest } = config
  return async (root, payload) => {
    calls.push({ root, payload })
    if (!ok) return { ok: false, error, ...rest }
    return { ok: true, exitCode, stdout, stderr, ...rest }
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
  await assert.rejects(() => after(postInput, {}), (err) => {
    assert.match(err.message, /ledger write failed/)
    assert.match(err.message, /Run machinery doctor/)
    assert.doesNotMatch(err.message, /reinstall.*before continuing/i)
    return true
  })
})

test("lock contention is diagnosed as transient rather than version skew", async () => {
  const after = await afterHandler({ exitCode: 1, stderr: "another operation holds the lock for install.json" })
  await assert.rejects(() => after(postInput, {}), (err) => {
    assert.match(err.message, /Retry after the active machinery install/)
    assert.doesNotMatch(err.message, /older than|version skew|reinstall/i)
    return true
  })
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

test("permission reply and tool error report exact denied completions", async () => {
  const calls = []
  const plugin = await MachineryPlugin({ client: {}, directory: "/project", worktree: "" }, { runner: fakeRunner({}, calls) })
  await plugin["tool.execute.before"]({ tool: "write", sessionID: "s", callID: "denied" }, { args: { filePath: "/project/design/a" } })
  await plugin.event({ event: { type: "permission.updated", properties: { id: "permission-1", sessionID: "s", callID: "denied" } } })
  await plugin.event({ event: { type: "permission.replied", properties: { sessionID: "s", permissionID: "permission-1", response: "reject" } } })
  await plugin["tool.execute.before"]({ tool: "write", sessionID: "s", callID: "errored" }, { args: { filePath: "/project/design/b" } })
  await plugin["tool.execute.after"]({ tool: "write", sessionID: "s", callID: "errored", args: { filePath: "/project/design/b" } }, { error: { message: "tool refused" } })
  const failures = calls.filter(({ payload }) => payload.hook_event_name === "PostToolUseFailure")
  assert.deepEqual(failures.map(({ payload }) => payload.tool_use_id), ["denied", "errored"])
})

test("OpenCode tool-part error closes only the refused shell call", async () => {
  const calls = []
  const plugin = await MachineryPlugin({ client: {}, directory: "/project", worktree: "" }, { runner: fakeRunner({}, calls) })
  await plugin["tool.execute.before"]({ tool: "bash", sessionID: "part-session", callID: "shell-refused" }, { args: { command: "touch design/a" } })
  await plugin["tool.execute.before"]({ tool: "bash", sessionID: "part-session", callID: "shell-still-running" }, { args: { command: "schedule delayed writer" } })
  await plugin.event({ event: { type: "message.part.updated", properties: { part: { id: "prt-456", callID: "shell-refused", sessionID: "part-session", messageID: "msg-1", type: "tool", tool: "bash", state: { status: "error", input: { command: "touch design/a" }, error: "permission denied", time: { start: 1, end: 2 } } } } } })
  assert.deepEqual(calls.filter(({ payload }) => payload.hook_event_name === "PostToolUseFailure").map(({ payload }) => payload.tool_use_id), ["shell-refused"])
})

test("a spawn failure fails closed with the transport error", async () => {
  const after = await afterHandler({ ok: false, error: new Error("spawn machinery ENOENT") })
  await assert.rejects(() => after(postInput, {}), (err) => {
    assert.match(err.message, /ENOENT/)
    assert.match(err.message, /Machinery governance failed closed/)
    return true
  })
})

// --- response-shape validation: unknown or wrong-shape hook output blocks ---

test("an unknown JSON response field blocks instead of passing", async () => {
  const after = await afterHandler({ stdout: JSON.stringify({ unexpected: true }) })
  await assert.rejects(() => after(postInput, {}), (err) => {
    assert.match(err.message, /unrecognized/)
    assert.match(err.message, /Machinery governance failed closed/)
    return true
  })
})

test("a non-object JSON response blocks instead of passing", async () => {
  const after = await afterHandler({ stdout: JSON.stringify([1, 2, 3]) })
  await assert.rejects(() => after(postInput, {}), /unrecognized/)
})

test("an unsupported decision value blocks instead of passing", async () => {
  const after = await afterHandler({ stdout: JSON.stringify({ decision: "approve", reason: "go on" }) })
  await assert.rejects(() => after(postInput, {}), (err) => {
    assert.match(err.message, /unrecognized/)
    assert.match(err.message, /approve/)
    return true
  })
})

test("an unsupported permission decision blocks instead of passing", async () => {
  const after = await afterHandler({ stdout: JSON.stringify({ hookSpecificOutput: { hookEventName: "PreToolUse", permissionDecision: "ask" } }) })
  await assert.rejects(() => after(postInput, {}), (err) => {
    assert.match(err.message, /unrecognized/)
    assert.match(err.message, /ask/)
    return true
  })
})

test("a response combining decision and hookSpecificOutput blocks as unrecognized", async () => {
  const stdout = JSON.stringify({ decision: "block", reason: "x", hookSpecificOutput: { hookEventName: "PreToolUse", permissionDecision: "deny", permissionDecisionReason: "y" } })
  const after = await afterHandler({ stdout })
  await assert.rejects(() => after(postInput, {}), /unrecognized/)
})

test("an empty JSON object response blocks as unrecognized", async () => {
  const after = await afterHandler({ stdout: "{}" })
  await assert.rejects(() => after(postInput, {}), /unrecognized/)
})

test("a documented allow decision passes without blocking", async () => {
  const after = await afterHandler({ stdout: JSON.stringify({ hookSpecificOutput: { hookEventName: "PreToolUse", permissionDecision: "allow" } }) })
  await after(postInput, {})
})

test("a truncated capture blocks instead of parsing partial output", async () => {
  const after = await afterHandler({ stdout: '{"decision":"bl', stdoutTruncated: true })
  await assert.rejects(() => after(postInput, {}), (err) => {
    assert.match(err.message, /overflowed its bounded output capture/)
    assert.match(err.message, /Machinery governance failed closed/)
    return true
  })
})

// --- native subprocess behavior: the real spawn transport, no fake runner ---

test("defaultRunner is exported with injectable bounds for native subprocess tests", () => {
  assert.equal(typeof defaultRunner, "function", "defaultRunner must be exported so its bounds are observable")
})

test("native hanging child is terminated at the deadline", { timeout: 30000 }, async () => {
  assert.equal(typeof defaultRunner, "function")
  await withPath(fixtures.hang, async () => {
    const started = Date.now()
    const result = await defaultRunner("/project", hookPayload(), { timeoutMs: 1500, maxOutputBytes: 65536 })
    const elapsed = Date.now() - started
    assert.ok(!result.ok, `expected a bounded failure, got ${JSON.stringify({ ...result, stdout: result.stdout?.length, stderr: result.stderr?.length })}`)
    assert.match(result.error.message, /deadline/)
    assert.ok(elapsed < 5000, `settlement took ${elapsed}ms; the deadline did not bound it`)
    assert.ok(await reaped(result.pid), `child ${result.pid} survived termination`)
  })
})

test("native child ignoring SIGTERM is still terminated", { timeout: 30000 }, async () => {
  assert.equal(typeof defaultRunner, "function")
  await withPath(fixtures.ignore, async () => {
    const started = Date.now()
    const result = await defaultRunner("/project", hookPayload(), { timeoutMs: 1500, maxOutputBytes: 65536 })
    const elapsed = Date.now() - started
    assert.ok(!result.ok, `expected a bounded failure, got ${JSON.stringify({ ...result, stdout: result.stdout?.length, stderr: result.stderr?.length })}`)
    assert.match(result.error.message, /deadline/)
    assert.ok(elapsed < 5000, `settlement took ${elapsed}ms; escalation did not bound it`)
    assert.ok(await reaped(result.pid), `child ${result.pid} survived SIGTERM escalation`)
  })
})

test("native stdout flood is captured within bounded limits and terminated", { timeout: 30000 }, async () => {
  assert.equal(typeof defaultRunner, "function")
  await withPath(fixtures["flood-stdout"], async () => {
    const result = await defaultRunner("/project", hookPayload(), { timeoutMs: 8000, maxOutputBytes: 65536 })
    assert.ok(!result.ok, "an stdout flood must be a bounded failure")
    assert.match(result.error.message, /stdout/)
    assert.match(result.error.message, /bounded capture/)
    const captured = result.stdout?.length ?? 0
    assert.ok(captured <= 131072, `captured ${captured} stdout bytes; the capture ceiling did not bind`)
    assert.ok(await reaped(result.pid), `child ${result.pid} survived overflow termination`)
  })
})

test("native stderr flood is bounded independently of stdout", { timeout: 30000 }, async () => {
  assert.equal(typeof defaultRunner, "function")
  await withPath(fixtures["flood-stderr"], async () => {
    const result = await defaultRunner("/project", hookPayload(), { timeoutMs: 8000, maxOutputBytes: 65536 })
    assert.ok(!result.ok, "an stderr flood must be a bounded failure")
    assert.match(result.error.message, /stderr/)
    assert.match(result.error.message, /bounded capture/)
    const captured = result.stderr?.length ?? 0
    assert.ok(captured <= 131072, `captured ${captured} stderr bytes; the capture ceiling did not bind`)
    assert.ok(result.stdout === undefined || result.stdout.length <= 131072, "the stderr flood leaked into stdout capture")
    assert.ok(await reaped(result.pid), `child ${result.pid} survived overflow termination`)
  })
})

test("native settle-exactly-once across early exit, flood and EPIPE races", { timeout: 30000 }, async () => {
  assert.equal(typeof defaultRunner, "function")
  await withPath(fixtures["exit-7"], async () => {
    const runs = []
    for (let i = 0; i < 25; i++) {
      runs.push(defaultRunner("/project", { ...hookPayload(), tool_use_id: `call-race-${i}` }, { timeoutMs: 6000, maxOutputBytes: 65536 }))
    }
    const results = await Promise.all(runs)
    assert.equal(results.length, 25)
    for (const result of results) {
      if (result.ok) assert.equal(result.exitCode, 7, "a settled close must carry the real exit code")
      else assert.ok(result.error instanceof Error, "a bounded settlement must carry a transport error")
    }
  })
})

test("native absent machinery binary fails closed without a shell", { timeout: 30000 }, async () => {
  const saved = process.env.PATH
  process.env.PATH = fixtures.absent
  try {
    const plugin = await MachineryPlugin({ client: {}, directory: "/project", worktree: "" }, {})
    await assert.rejects(() => plugin["tool.execute.before"](
      { tool: "write", sessionID: "session-absent", callID: "call-absent" },
      { args: { filePath: "design/BUILD.md" } },
    ), (err) => {
      assert.match(err.message, /Machinery governance failed closed/)
      assert.match(err.message, /could not be executed|ENOENT/)
      return true
    })
  } finally {
    process.env.PATH = saved
  }
})

test("native silent child preserves the documented empty-success protocol", { timeout: 30000 }, async () => {
  await withPath(fixtures.silent, async () => {
    const plugin = await MachineryPlugin({ client: {}, directory: "/project", worktree: "" }, {})
    await plugin["tool.execute.after"](postInput, {})
  })
})

test("native nonzero exit child blocks with its stderr detail", { timeout: 30000 }, async () => {
  await withPath(fixtures["exit-7"], async () => {
    const plugin = await MachineryPlugin({ client: {}, directory: "/project", worktree: "" }, {})
    await assert.rejects(() => plugin["tool.execute.after"](postInput, {}), /boom: ledger exploded/)
  })
})

test("native malformed JSON child blocks", { timeout: 30000 }, async () => {
  await withPath(fixtures.garbage, async () => {
    const plugin = await MachineryPlugin({ client: {}, directory: "/project", worktree: "" }, {})
    await assert.rejects(() => plugin["tool.execute.after"](postInput, {}), /malformed JSON/)
  })
})

test("native wrong-shape JSON child blocks instead of passing", { timeout: 30000 }, async () => {
  await withPath(fixtures["wrong-shape"], async () => {
    const plugin = await MachineryPlugin({ client: {}, directory: "/project", worktree: "" }, {})
    await assert.rejects(() => plugin["tool.execute.after"](postInput, {}), /unrecognized/)
  })
})

test("native documented allow decision child passes without blocking", { timeout: 30000 }, async () => {
  await withPath(fixtures.allow, async () => {
    const plugin = await MachineryPlugin({ client: {}, directory: "/project", worktree: "" }, {})
    await plugin["tool.execute.before"](
      { tool: "write", sessionID: "session-allow", callID: "call-allow" },
      { args: { filePath: "design/BUILD.md" } },
    )
  })
})

test("native real machinery binary governs a managed root end to end", { timeout: 300000 }, async () => {
  assert.equal(typeof defaultRunner, "function")
  const buildDir = path.join(scratch, "build")
  await mkdir(buildDir, { recursive: true })
  await new Promise((resolve, reject) => {
    execFile("go", ["build", "-o", path.join(buildDir, "machinery"), "./cmd/machinery"], { cwd: repoRoot, timeout: 240000, maxBuffer: 1 << 20 }, (err, so, se) => {
      if (err) { err.message = `${err.message}\nstdout: ${so}\nstderr: ${se}`; reject(err); return }
      resolve()
    })
  })
  // Managed root mirrors cmd/machinery TestHookCmdDeniesGeneratedEdit: the
  // conventional marker alone must arm governance.
  const root = await mkdtemp(path.join(scratch, "managed-"))
  await mkdir(path.join(root, "design"), { recursive: true })
  await writeFile(path.join(root, "design", "domain.modelith.yaml"), "model: {}\n")
  const oracle = path.join(root, "design", "machines", "Deal.oracle.md")
  const configDir = await mkdtemp(path.join(scratch, "config-"))
  const savedPath = process.env.PATH
  const savedConfig = process.env.MACHINERY_CONFIG_DIR
  process.env.PATH = buildDir + path.delimiter + savedPath
  process.env.MACHINERY_CONFIG_DIR = configDir
  try {
    const plugin = await MachineryPlugin({ client: {}, directory: root, worktree: "" }, {})
    await assert.rejects(() => plugin["tool.execute.before"](
      { tool: "edit", sessionID: "session-real", callID: "call-real" },
      { args: { path: oracle } },
    ), (err) => {
      assert.match(err.message, /machinery oracle/)
      return true
    })
    // OpenCode refuses the first call after our before hook, emits no after,
    // then another call mutates the tree before every idle (each is a first Stop).
    const observed = []
    const governed = await MachineryPlugin({ client: {}, directory: root, worktree: "" }, {
      runner: async (r, payload) => {
        const result = await defaultRunner(r, payload)
        observed.push({ payload, result })
        return result
      },
    })
    await governed["tool.execute.before"]({ tool: "write", sessionID: "refused-session", callID: "refused-call" }, { args: { filePath: path.join(root, "design", "refused.txt") } })
    await governed["tool.execute.before"]({ tool: "write", sessionID: "refused-session", callID: "other-call" }, { args: { filePath: path.join(root, "design", "other.txt") } })
    await writeFile(path.join(root, "design", "other.txt"), "unrelated mutation\n")
    await governed["tool.execute.after"]({ tool: "write", sessionID: "refused-session", callID: "other-call", args: { filePath: path.join(root, "design", "other.txt") } }, {})
    for (let i = 0; i < 2; i++) {
      await governed.event({ event: { type: "session.idle", properties: { sessionID: "refused-session" } } })
    }
    assert.ok(observed.some(({ payload }) => payload.hook_event_name === "PostToolUseFailure" && payload.tool_use_id === "refused-call"), "refusal must be translated as exact failed completion")
    const stops = observed.filter(({ payload }) => payload.hook_event_name === "Stop")
    assert.equal(stops.length, 2)
    assert.ok(stops.every(({ result }) => !result.stdout.includes('"decision":"block"')), "each idle must discharge its gate result")
    const stateFiles = await readdir(configDir, { recursive: true })
    for (const file of stateFiles.filter((name) => name.endsWith(".state"))) {
      assert.doesNotMatch(await readFile(path.join(configDir, file), "utf8"), /pending /)
    }
    // OpenCode's terminal tool-part carries its own id plus the Bash callID.
    // A sandbox-refused shell has no execute.after, then another call can
    // mutate the design before each idle (each idle is a first Stop).
    const shellObserved = []
    const shell = await MachineryPlugin({ client: {}, directory: root, worktree: "" }, {
      runner: async (r, payload) => {
        const result = await defaultRunner(r, payload)
        shellObserved.push({ payload, result })
        return result
      },
    })
    await shell["tool.execute.before"]({ tool: "bash", sessionID: "shell-denial-session", callID: "call-123" }, { args: { command: "cd /sandbox/refused" } })
    await shell.event({ event: { type: "message.part.updated", properties: { part: { id: "prt-456", callID: "call-123", sessionID: "shell-denial-session", messageID: "msg-123", type: "tool", tool: "bash", state: { status: "error", input: { command: "cd /sandbox/refused" }, error: "permission denied", time: { start: 1, end: 2 } } } } } })
    await shell["tool.execute.before"]({ tool: "write", sessionID: "shell-denial-session", callID: "unrelated-write" }, { args: { filePath: path.join(root, "design", "after-denial.txt") } })
    await writeFile(path.join(root, "design", "after-denial.txt"), "unrelated mutation\n")
    await shell["tool.execute.after"]({ tool: "write", sessionID: "shell-denial-session", callID: "unrelated-write", args: { filePath: path.join(root, "design", "after-denial.txt") } }, {})
    for (let i = 0; i < 2; i++) {
      await shell.event({ event: { type: "session.idle", properties: { sessionID: "shell-denial-session" } } })
    }
    assert.deepEqual(shellObserved.filter(({ payload }) => payload.hook_event_name === "PostToolUseFailure").map(({ payload }) => payload.tool_use_id), ["call-123"])
    assert.equal(shellObserved.filter(({ payload }) => payload.hook_event_name === "Stop").length, 2)
    assert.ok(shellObserved.filter(({ payload }) => payload.hook_event_name === "Stop").every(({ result }) => !result.stdout.includes('"decision":"block"')))
    const late = await MachineryPlugin({ client: {}, directory: root, worktree: "" }, {})
    await late["tool.execute.before"]({ tool: "bash", sessionID: "late-session", callID: "late-writer" }, { args: { command: "schedule delayed writer" } })
    const delayedWrite = sleep(50).then(() => writeFile(path.join(root, "design", "late.txt"), "late write\n"))
    await assert.rejects(() => late.event({ event: { type: "session.idle", properties: { sessionID: "late-session" } } }), /in-flight tool/)
    await delayedWrite
    await assert.rejects(() => late.event({ event: { type: "session.idle", properties: { sessionID: "late-session" } } }), /in-flight tool/)
    // The same transport stays silent on an unmanaged root: the documented
    // empty-success protocol through the real binary.
    const bare = await mkdtemp(path.join(scratch, "unmanaged-"))
    const quiet = await MachineryPlugin({ client: {}, directory: bare, worktree: "" }, {})
    await quiet["tool.execute.before"](
      { tool: "edit", sessionID: "session-real", callID: "call-real-2" },
      { args: { path: path.join(bare, "design", "machines", "X.oracle.md") } },
    )
  } finally {
    process.env.PATH = savedPath
    if (savedConfig === undefined) delete process.env.MACHINERY_CONFIG_DIR
    else process.env.MACHINERY_CONFIG_DIR = savedConfig
  }
})

test("adapter source keeps shell-free argv and stdin construction structural", () => {
  assert.match(source, /spawn\("machinery", \["hook", "--root", root\]/)
  assert.match(source, /child\.stdin\.end\(JSON\.stringify\(payload\)\)/)
  assert.doesNotMatch(source, /shell:\s*true/)
  assert.doesNotMatch(source, /\bexec(Sync)?\(/)
})

test("explicit Node runtime executes the native governance transport", () => {
  const major = Number(process.versions.node.split(".")[0])
  assert.ok(major >= 22, `native transport requires Node >= 22, got ${process.versions.node}`)
  assert.match(process.execPath, /node/)
})
