// Required-lane wrapper for the OpenCode governance adapter suite. The lane
// runner only accepts *.integration.test.mjs sources, while the adapter's
// native cases live next to the plugin in
// adapters/opencode/plugins/machinery.test.mjs (the suite this story's
// contract names). This wrapper executes that suite as a real bounded child
// process and holds it to a closed inventory: every named case below must
// start exactly once, pass, and never skip. A missing case, an extra case, a
// skip, or a failure fails this lane case; nothing is cached or summarized
// from previous runs.
import assert from "node:assert/strict"
import { execFile } from "node:child_process"
import path from "node:path"
import { promisify } from "node:util"
import test from "node:test"

const execFileAsync = promisify(execFile)

const repoRoot = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..", "..")
const suiteRelative = "adapters/opencode/plugins/machinery.test.mjs"

// The closed inventory of the adapter suite. This list is part of the frozen
// RED contract: names removed from the suite or added to it without a matching
// change here fail the lane in both directions.
const closedInventory = [
  "PostToolUse throws when machinery exits nonzero",
  "lock contention is diagnosed as transient rather than version skew",
  "PostToolUse throws when machinery returns malformed JSON",
  "PostToolUse propagates an explicit block decision",
  "shell tools participate in PostToolUse governance",
  "plugin constructs without a Bun $ and routes governed calls through the runner",
  "a spawn failure fails closed with the transport error",
  "an unknown JSON response field blocks instead of passing",
  "a non-object JSON response blocks instead of passing",
  "an unsupported decision value blocks instead of passing",
  "an unsupported permission decision blocks instead of passing",
  "a response combining decision and hookSpecificOutput blocks as unrecognized",
  "an empty JSON object response blocks as unrecognized",
  "a documented allow decision passes without blocking",
  "a truncated capture blocks instead of parsing partial output",
  "defaultRunner is exported with injectable bounds for native subprocess tests",
  "native hanging child is terminated at the deadline",
  "native child ignoring SIGTERM is still terminated",
  "native stdout flood is captured within bounded limits and terminated",
  "native stderr flood is bounded independently of stdout",
  "native settle-exactly-once across early exit, flood and EPIPE races",
  "native absent machinery binary fails closed without a shell",
  "native silent child preserves the documented empty-success protocol",
  "native nonzero exit child blocks with its stderr detail",
  "native malformed JSON child blocks",
  "native wrong-shape JSON child blocks instead of passing",
  "native documented allow decision child passes without blocking",
  "native real machinery binary governs a managed root end to end",
  "adapter source keeps shell-free argv and stdin construction structural",
  "explicit Node runtime executes the native governance transport",
]

// The lane strips NODE_TEST_CONTEXT/NODE_OPTIONS at its boundary; a direct
// `node --test` invocation of this wrapper does not, and a nested runner
// inheriting NODE_TEST_CONTEXT answers the harness protocol on stdin instead
// of executing the suite. Apply the same drop before spawning.
const suiteEnv = {}
for (const [key, value] of Object.entries(process.env)) {
  if (key === "NODE_TEST_CONTEXT" || key === "NODE_OPTIONS") continue
  suiteEnv[key] = value
}

test("opencode governance adapter executes its closed native subprocess inventory", async () => {
  assert.ok(Number(process.versions.node.split(".")[0]) >= 22)
  let tap
  try {
    ({ stdout: tap } = await execFileAsync(
      process.execPath,
      ["--test", "--test-reporter=tap", suiteRelative],
      { cwd: repoRoot, env: suiteEnv, timeout: 240000, maxBuffer: 8 << 20, encoding: "utf8" },
    ))
  } catch (err) {
    const failing = String(err.stdout || "").split("\n")
      .filter((line) => line.startsWith("not ok") || /^ {2}error:/.test(line) || line.includes("failureType"))
      .slice(0, 30)
      .join("\n")
    assert.fail(`adapter suite failed:\n${failing}\n${err.message}`)
  }

  const started = []
  const terminals = new Map()
  let plan = ""
  const summary = {}
  for (const line of tap.split("\n")) {
    if (line.startsWith("# Subtest: ")) started.push(line.slice("# Subtest: ".length))
    const terminal = /^(ok|not ok) (\d+) - (.+)$/.exec(line)
    if (terminal) {
      const name = terminal[3].split(" #")[0]
      const directive = line.includes("# SKIP") ? "skip" : line.includes("# TODO") ? "todo" : ""
      if (!terminals.has(name)) terminals.set(name, [])
      terminals.get(name).push({ ok: terminal[1] === "ok", directive })
    }
    if (/^1\.\.\d+$/.test(line)) plan = line
    const count = /^# (tests|pass|fail|cancelled|skipped|todo) (\d+)$/.exec(line)
    if (count) summary[count[1]] = Number(count[2])
  }

  const expected = new Set(closedInventory)
  const actual = new Set(started)
  for (const name of closedInventory) {
    assert.ok(actual.has(name), `closed inventory case did not execute: ${name}`)
  }
  for (const name of started) {
    assert.ok(expected.has(name), `suite executed an unregistered case: ${name}`)
  }
  assert.equal(started.length, closedInventory.length, "every case must start exactly once")

  for (const name of closedInventory) {
    const runs = terminals.get(name) ?? []
    assert.equal(runs.length, 1, `case ${name} must terminate exactly once`)
    const run = runs[0]
    assert.equal(run.directive, "", `case ${name} was skipped or marked todo`)
    assert.ok(run.ok, `case ${name} failed`)
  }

  assert.equal(plan, `1..${closedInventory.length}`)
  assert.equal(summary.tests, closedInventory.length)
  assert.equal(summary.pass, closedInventory.length)
  assert.equal(summary.fail, 0)
  assert.equal(summary.cancelled, 0)
  assert.equal(summary.skipped, 0)
  assert.equal(summary.todo, 0)
})
