declare module "node:test" {
  interface TestContext {
    skip(message?: string): void;
  }
  function test(name: string, fn: (t: TestContext) => void | Promise<void>): void;
}
declare module "node:assert/strict" {
  function equal(actual: unknown, expected: unknown, message?: string): void;
}
