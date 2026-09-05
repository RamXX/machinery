# Required integration lane RED contract

Candidate v2 revises rejected v1 (612f65f3b4502a3267828507faaf0e395c8dd558)
under the PM's explicit test-edit authorization. V1 was never RED-approved.

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
`events_file` plus `events_sha256`, and the exact native `adapter`. Runtime
records contain `id`, `identity`, `status`, `path` and `sha256`. Both successful
and unsuccessful runs report what actually executed;
unavailable prerequisites cannot produce passing runtime or suite records.

The runtime set equals the union of selected suite requirements, with no missing,
duplicate or invented identities. Go identity is the exact `go version` output;
Node identity is exact `node --version`. Both bind the actual executable bytes.
Docker identity is the immutable image followed by one space and the platform;
its sha256 is the pinned OCI digest. Java identity is the pinned probe release
21.0.12.1+1-LTS, with launcher sha256 plus independently checkable
`closure_sha256` and the platform pin's `archive_sha256`. TLC identity is v1.7.4,
with the pinned jar sha256. Provisioned formal paths remain inspectable inside
the selected private cache and are not the ambient Java/JAR.

After verified provisioning and before test selection, the runner supplies
`MACHINERY_INTEGRATION_JAVA` and `MACHINERY_INTEGRATION_TLC_JAR` to native tests.
The CLI-path cold-cache fixture invokes those exact paths for safe and unsafe
TLC models, captures their hashes, and compares them with the runner receipt.

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
An existing caller-owned work directory is a parent for private scratch, never
the deletion target. Its native identity and user files survive every path.
Host descendants must also be terminated after success, failure, timeout and
cancellation; the tests independently inspect exact PID plus private executable
script identity and retain bounded emergency cleanup for their own processes.

## Mandatory wiring

The workflow lane step is exactly `go run ./scripts/integration-lane --lane required`
in an unconditional Linux job and step, without ignored failures or conditional
expressions. The Make target resolves to that same exact command. Preflight
uses a transparent top-level bare invocation under strict error propagation,
after the service-free race suite and before formal/C4/checker consumers.
Executable bypass variables, ignored errors, success exits and opaque transfers
cannot bypass it. Test-only structural guards and real bounded Bash sensitivity
controls reject disabled variants without executing the full preflight.

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
