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
