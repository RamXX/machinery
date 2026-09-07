declare module "node:test" {
  interface TestContext {
    diagnostic(message: string): void;
    skip(message?: string): void;
    test(name: string, fn: (t: TestContext) => void | Promise<void>): Promise<void> | void;
  }
  function test(name: string, fn: (t: TestContext) => void | Promise<void>): void;
}
declare module "node:assert/strict" {
  function ok(value: boolean, message?: string): void;
}
