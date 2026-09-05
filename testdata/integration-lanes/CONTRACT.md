# Required integration lane RED contract

This is an explicit test contract for MAC-hpqp, not operational code. The
dispatcher approved the fail-closed `run` bootstrap and Node fixture before the
RED freeze. The bootstrap does not validate anything; generic rejection is not
evidence that malformed input was correctly diagnosed.

## Public entry point

`go run ./scripts/integration-lane --lane required` discovers every fragment in
`testdata/integration-lanes`. `schema.json`, `runtime-pins.json`, this contract and
native test sources are not fragments. Other unknown inventory members fail.
Additional contributor fragments join the union automatically; duplicate native
identities and unregistered integration files/cases fail, even across fragments.

Test/diagnostic options are `--root DIR`, `--report FILE`, `--work-dir DIR`,
`--cache-dir DIR` and `--pins FILE`. Defaults resolve from the repository and a
fresh private work root. Existing caller-owned roots are never recursively
removed. All paths, source ownership, source test selection and bounded values
must be validated before any suite runs. No shell command/argv override exists.
Go selection uses the `machinery_integration` build tag, exact test identities,
fresh uncached execution and native JSON. Node selection uses real native TAP
events with exact test identities. The report does not substitute for events.

## Report

The report contains `version: 1`, `lane`, `status`, `suites`, `runtimes` and
`cleanup.status`. Each suite contains `id`, `selected`, `started`, `passed`,
`failed`, `skipped`, exact `tests` (`name`, `source`, `status`), and retained
`events_file` plus `events_sha256`. Runtime records contain `id`, `identity` and
`status`. Both successful and unsuccessful runs report what actually executed;
unavailable prerequisites cannot produce passing runtime or suite records.

## Resource ownership

The runner creates a private `MACHINERY_INTEGRATION_WORK` for each suite and a
private `MACHINERY_INTEGRATION_CACHE` for its pinned runtime closure. Tests can
register Docker ownership by writing Docker's native `--cidfile` under the
suite work root's `containers/` directory. Container creation must additionally
use the exact per-run label `dev.machinery.integration-run` provided in
`MACHINERY_INTEGRATION_RUN_ID`; IDs/labels are verified before cleanup. Tests
may create only bounded, offline, immutable-image containers. Owned containers
must be absent before the lane reports success, including when their test
process exits early. A detected leak fails the lane even if safely reclaimed.
Cancellation must terminate children and reclaim only these verified resources.

The real full-path meta-test creates a separate tiny module/inventory (never a
replacement runtime executable). It invokes the runner against that module,
so nested execution cannot recursively select the repository pilot suite.
Provisioning failures are hard failures, never skips. Pinned Java/TLC use the
existing checksum-verified provisioning path, not the ambient launcher.

## Evidence limits

Source/workflow checks establish wiring, not runtime correctness. Native parser
fixtures establish parsing, not runtime correctness. Only actual fresh native
processes and actual daemon-backed execution establish integration evidence.
Full preflight remains deferred until final epic completion.
