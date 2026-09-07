import { test } from "node:test";
import assert from "node:assert/strict";

test("assurance runtime probe executes a native TypeScript closure", (t) => {
  const answer: number = 6 * 7;
  assert.equal(answer, 42);
});
