# Fixture projects

Scripts copy a fixture into their work directory with `fixture <name>` and then edit their own
copy. Do not change a fixture to suit one script: every script depends on these exact files.

## `empty`

An empty directory.

## `shop`

A small, valid machinery project: one model, one lifecycle machine, one failure mode, one OpenAPI
contract, two Go files and one evidence emitter. Generated files are not part of the fixture;
scripts run `machinery gen` themselves.

### Served ids

`act:Order.place`, `act:Order.pay`, `act:Order.ship`, `act:Order.cancel`.
`act:Order.place` has no machine event; `machinery.yaml` waives rule `action-event` for it.

### Obligations (16)

| Id | Title | Effects | In the fixture |
|---|---|---|---|
| `inv:customer-email-unique` | No two Customer records share an email. | | proven |
| `inv:paid-before-shipped` | An Order reaches the shipped status only from the paid status. | | proven |
| `inv:terminal-is-final` | A shipped or cancelled Order never changes status again. | | proven |
| `sc:customer-pays-and-the-order-ships` | Customer pays and the order ships | | proven |
| `sc:cancelled-order-refuses-payment` | Cancelled order refuses payment | | proven |
| `tc:ORDE-2e883d` | Pending on:pay [hasPaymentMethod] -> paying | stampAttempt | proven |
| `tc:ORDE-70d94f` | Pending on:pay -> (internal) | recordPayRefused | proven |
| `tc:ORDE-1ded14` | Pending on:cancel -> Cancelled | recordCancellation | proven |
| `tc:ORDE-ad945c` | paying after:payTimeout -> Pending | recordPaymentTimeout | proven |
| `tc:ORDE-8bd6e7` | paying onDone:chargePayment -> Paid | recordCharge | partial, owed |
| `tc:ORDE-267d12` | paying onError:chargePayment -> Pending | recordPaymentFailure | proven |
| `tc:ORDE-e6d28f` | Paid on:ship -> Shipped | recordShipment, notifyCustomer | proven |
| `tc:ORDE-507e4f` | Paid on:cancel -> Cancelled | refundPayment, recordCancellation | proven |
| `tc:ORDE-2fa2af` | Pending ignore:ship -> (unchanged) | | proven |
| `tc:ORDE-8799a1` | Paid ignore:pay -> (unchanged) | | proven |
| `fm:payment-provider-down` | The payment provider does not answer or answers with an error. | | no receipt, owed |

`notifyCustomer` is the entry action of `Shipped`: it shows that effects include the target
state's entry actions.

"In the fixture" is what `machinery verify` reports after
`machinery run -- sh e2e/orders.sh`: 14 proven, 1 partial, 2 owed, 15 receipts.

### Other counts

- Model: 2 entities, 4 actions, 3 invariants, 2 scenarios.
- Machine `order`, tag `ORDE`: 5 states, 8 transitions, 2 ignore cases, 1 invoke (`chargePayment`).
- Contract: 5 operations, 4 bound to an action.
- Code: 2 Go files in 2 layers, 1 import between them, 2 functions.

## `nested`

One operational machine `job` (tag `JOB`) with a compound state `Running` (children `warming`,
`working`), entry and exit actions on every state, and `tools.xstate: true`. Its 7 `on` rows cover a
transition into the compound state, a deep target, one out of it, a sibling inside it, a compound
self-target and an atomic self-target. No model; generated files are not part of the fixture.

## `receiptprog`

A tiny Go program, its own module, that emits receipts with `pkg/receipt` from its arguments
(`pass`, `fail`, `skip`, `replay`, `outcome`, `many`) and prints what `Enabled`, `Only` and
`Wanted` return. Its `go.mod` requires `github.com/RamXX/machinery`; a script adds the `replace`
directive pointing at `$REPO` and builds it.

## `checkerprog`

A tiny Go program (its own module, standard library only) that stands in for an external
checker: `checkerprog [flags] <projection path> <evidence path>` reads the projection and writes
valid evidence. Scripts build it with `go build` into `.machinery/bin` and register it in
`.machinery/checkers.local.yaml`, or call it directly. Flags: `-id`, `-covers` (ids or
`prefix*` patterns, default every id), `-extra-cover`, `-finding error|warn`, `-subject`,
`-obligation a,b`, `-wrong-hash`, `-invalid-json`, `-bad-verdict`, `-extra-key`, `-exit N`,
`-sleep SECONDS` and `-no-write`.

## `shop-formal`

Laid over `shop`: a risk file with two critical areas and one consequential area, abuse failure
modes, a TLA+ spec of the order payment (`order-payment`) and an Alloy spec of order ownership
(`order-ownership`), and `e2e/replay.sh`, which emits the replay receipt of the race
counterexample. The helper `shopformal` (`e2e/testdata/bin`) generates, commits, stamps
`sources.read_at` and writes the skeletons.

## `shop-vital`

Laid over `shop` and `shop-formal`: the order payment as a vital area. `design/risk.yaml` makes
`order-payment` vital, with two properties (`charged-once` over `act:Order.pay` and
`tc:ORDE-8bd6e7`, `shipped-only-paid` over `inv:paid-before-shipped` and `tc:ORDE-e6d28f`), the core
`internal/domain/**` entered at `internal/domain/order.go#Apply`, and two assumptions: one whose
evidence, `fm:payment-provider-down`, is owed, and one whose evidence, `inv:terminal-is-final`, is
not. `machinery.yaml` adds `constraints.cores` to the layers and sizes of `shop`. The failure
mode `pay-replayed` (owed) names a duplication and is in no vital area, so it is a candidate, as
is `tc:ORDE-70d94f` (the refused pay). `e2e/outage.sh` emits the receipt of
`fm:payment-provider-down`.

Counts the scripts rely on: 22 obligations (12 routine, 1 critical, 9 vital), 2 proof
obligations, 9 vital ids, 2 candidates.

## `shop-gotests`

Laid over `shop`: an ordinary Go suite, package `example.test/shop/e2e/suite`, whose tests emit
no receipt and name the ids they verify in comments. It is run for real by the scripts of the
run adapter (section 10.3).

| File | Test functions | Claims |
|---|---|---|
| `order_test.go` | `TestShipNeedsPayment`, `TestCancelIsFinal` (two subtests), `TestRouting` | Above `TestShipNeedsPayment`: `inv:paid-before-shipped`. Above `TestCancelIsFinal`: `inv:terminal-is-final`, `tc:ORDE-1ded14`. |
| `email_test.go` | `TestEmailUnique` | Above it: `inv:customer-email-unique`. |
| `scenario_test.go` | `TestPayThenShip`, `TestCancelledRefusesPay`; and `TestMain`, which is none | Above no function: both `sc` ids. |
| `outage_test.go` (build tag `faults`) | `TestProviderDown` | Above no function: `fm:payment-provider-down`. |

Environment switches, read by the tests: `SHOP_EMAIL=fail|skip|hang|exit|panic|kill`
(`TestEmailUnique`: fails, skips, blocks past `-timeout`, calls `os.Exit`, panics in a goroutine,
or kills the `go` command that runs it), `SHOP_SCENARIO=fail|skip` (`TestCancelledRefusesPay`),
`SHOP_FAIL_SUBTEST=shipped|cancelled` and `SHOP_SKIP_SUBTEST=shipped|cancelled` (a subtest of
`TestCancelIsFinal` fails or skips), `SHOP_ROUTING=fail` (`TestRouting`, which claims nothing)
and `SHOP_OUTAGE=fail` (`TestProviderDown`).

Without a tag the suite has 6 test functions with a verdict; with `-tags faults`, 7.

`run-suite.sh` is the project's wrapper: it copies the test files to `stage/suite` (package
`example.test/shop/stage/suite`), starts `go test` twice (plain, then `-tags faults -run
TestProviderDown`), writes each event stream to `$MACHINERY_GO_TEST_JSON` when that is set, and
removes `stage`. `SUITE_FLAGS` replaces its default flags, `-count=1`.

## `library`

A project that keeps its design in its own formats, read through `import` and `trace.markers`
without converting anything. Generated files are not part of the fixture.

- `machines/loan.json`: the lifecycle of entity `Loan` in a JSON dialect (`entity`, `notes`,
  `startable`, `skip`, `allowed`, `denied`) with lower-case state names. Translated with
  `states: capitalize`, it is machine `loan`: 4 states (`Lost` is tagged `failure`), 3
  transitions, 3 ignores, 1 refusal, entry state `Lent`.
- `machines/hold-queue.json`: machine `hold_queue`, 2 states, 4 transitions, one invoke
  (`sendNotice`). It says nothing of its binding: `import.machines.bind` makes it operational,
  serving `entity:Loan`. Its event `notify` comes from the scheduler, listed in `from_outside`
  (renamed to `_external_events`).
- `design/model/library.modelith.yaml`: entity `Loan` (statuses Reserved, Lent, Returned, Lost),
  3 actions, 2 invariants, 2 scenarios.
- `docs/risks.md`: 4 failure modes `RISK-1` to `RISK-4` in two table layouts, beside a table that
  holds none; `design/risk-links.yaml` links 3 of them (one abuse case) and declares the fourth
  subject-less (`no_subject`). `fm:risk-4` is owed.
- `internal/desk/desk.go` and `e2e/loans.sh` claim every id through the markers `Implements:`,
  `Checks:`, `Proves:` and `Scenario:`.

Counts the scripts rely on: 19 obligations (11 cases, 2 invariants, 2 scenarios, 4 failure
modes), 3 act ids; `machinery check` passes.

## `brownfield`

A small existing Go project, module `example.test/depot`, standard library only, with no
`machinery.yaml` and no design directory. It builds and runs: `depot ship
create|pack|dispatch|deliver|cancel|show <id>` and `depot invoice <id> <cents>` over a JSON file
store (`--db <file>`), printing the shipment status after each command and refusing an illegal
move with a code on stderr (for example `SHIPMENT_NOT_PACKED`) and exit 1.

- `internal/shipping/`: the shipment lifecycle (created, packed, dispatched, delivered,
  cancelled) hard-coded in Go, and the store. `shipment_test.go` is an ordinary unit test; its
  comment mentions `gomock` (the team once used it).
- `internal/billing/`: invoice arithmetic and `invoice_test.go`.
- `e2e/smoke.sh`: builds the binary and drives it for real. It is the only file under `e2e/`.
- `backlog.sh`: creates the project's own bd backlog in the current directory: one epic, four
  stories, two blocking edges, one story closed (4 open, 3 ready, 1 closed). Scripts run it after
  `gitinit`; it refuses to run where `.beads` exists.

Counts the scripts rely on: 4 code files and 2 unit tests in all, of which 2 code files and 1 unit
test (with the `gomock` text) under `internal/shipping/`.

## `brownfield-design`

What a team adds when it models its riskiest stateful resource. Copy it over `brownfield` with
`fixture brownfield-design`: `design/model/depot.modelith.yaml` (entity Shipment, five statuses, five
actions, two invariants, two scenarios; passes `modelith lint` 0.6.0) and
`design/machines/Shipment.machine.json` (the lifecycle: 5 states, 5 transitions, 7 ignores). It has
no `machinery.yaml`: `machinery adopt` adds the files to the config. The action
`act:Shipment.create` has no machine event (creation enters the initial state), so a script that
runs the cross-reference stage appends a waiver for rule `action-event`.

Obligations once generated: 12 cases (`tc:SHIP-*`), 2 invariants, 2 scenarios (16), and 5 act ids.

## `brownfield-tests`

The end-to-end tests a team points at the design once it models the shipment. Copy it over
`brownfield` and `brownfield-design` with `fixture brownfield-tests`. Both tests build the real
`depot` binary and drive it over a store of their own.

| File | Emits | Claims |
|---|---|---|
| `e2e/shipment_e2e.sh` | One receipt per id, with effects, after its assertions. | `tc:SHIP-422414` (Created on:pack), `tc:SHIP-ab88ab` and `tc:SHIP-e82c34` (pack refused when Packed or Dispatched). |
| `e2e/shipment_test.go` (package `example.test/depot/e2e`) | Nothing: an ordinary Go test, for the run adapter. | Above `TestDelivery`: `sc:shipment-is-packed-dispatched-and-delivered`, `inv:dispatched-only-from-packed`. Above `TestFinal`: `sc:dispatched-shipment-refuses-cancellation`, `inv:terminal-is-final`, `tc:SHIP-58b48e`. |

Counts the scripts rely on: 8 claimed ids (4 `tc`, 2 `sc`, 2 `inv`), 2 test functions; the
adapter writes 4 passes and cannot prove `tc:SHIP-58b48e`, which needs effects.
