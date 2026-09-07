import { test } from "node:test";
import { check } from "./machinery-check.js";

test("conformance witness executes native assertion", (t) => {
  check(t, "conformance/witness", 6 * 7 === 42);
});

test("conformance parent identity", async (t) => {
  await t.test("conformance nested identity", (ct) => {
    check(ct, "conformance/nested", 2 * 7 === 14);
  });
});

test("conformance multiple assertions", (t) => {
  check(t, "conformance/multi-a", 3 * 5 === 15);
  check(t, "conformance/multi-b", 4 * 5 === 20);
});
