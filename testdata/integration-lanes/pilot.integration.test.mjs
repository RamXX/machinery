import assert from "node:assert/strict"
import { execFileSync } from "node:child_process"
import test from "node:test"

test("required Node pilot executes a real child process", () => {
  assert.ok(Number(process.versions.node.split(".")[0]) >= 22)
  const result = JSON.parse(execFileSync(process.execPath, ["-e", "process.stdout.write(JSON.stringify({pid:process.pid,value:6*7}))"], {
    encoding: "utf8", timeout: 5000, maxBuffer: 4096,
  }))
  assert.equal(result.value, 42)
  assert.ok(Number.isSafeInteger(result.pid) && result.pid > 0)
  assert.notEqual(result.pid, process.pid)
})
