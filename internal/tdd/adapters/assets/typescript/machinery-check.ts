import assert from "node:assert/strict";

export interface CheckTest {
  diagnostic(message: string): void;
}

const witnessSchema = "machinery.tdd.witness/v1";

function externalSite(): string {
  const frames = String(new Error().stack ?? "").split("\n");
  for (const frame of frames) {
    if (frame.includes("machinery-check")) {
      continue;
    }
    const match = frame.match(/((?:file:\/\/)?\/[^\s()]+):(\d+):(\d+)/);
    if (match) {
      return match[1].replace(/^file:\/\//, "") + ":" + match[2] + ":" + match[3];
    }
  }
  return "";
}

export function check(t: CheckTest, assertionId: string, condition: boolean): void {
  if (typeof condition !== "boolean") {
    throw new TypeError("machinery-check/v1 condition must be strictly boolean");
  }
  const site = externalSite();
  if (!condition) {
    let thrown = "unknown";
    try {
      assert.ok(condition, "machinery-check/v1 assertion " + assertionId + " failed");
    } catch (error) {
      thrown = error instanceof Error ? error.name : "unknown";
      t.diagnostic(JSON.stringify({ schema: witnessSchema, assertion: assertionId, phase: "evaluated", site: site, condition: false, thrown: thrown }));
      throw error;
    }
  }
  t.diagnostic(JSON.stringify({ schema: witnessSchema, assertion: assertionId, phase: "evaluated", site: site, condition: true, thrown: null }));
}
