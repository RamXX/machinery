# BUILD: Checkout Split (parent manifest)

Mode: manifest (recursive decomposition; the buildable designs are the children).

This root fixes what the children may not change: the domain model
(checkout.modelith.yaml), the container topology and boundary rules
(workspace.dsl + the Architecture Contract), the event contracts, and the two
contract machines under contracts/. Each subsystem's pack under packs/ is the
GENERATED, frozen interface its child design consumes; regenerate with
machinery pack generate, never edit.

## Children

| subsystem | design | pack |
|---|---|---|
| orders | ../../orders/design | packs/orders.pack |
| payments | ../../payments/design | packs/payments.pack |

## Traceability (parent-level)

| invariant | enforcement |
|---|---|
| no-ship-without-capture | delegated to the orders child (structural: Shipped is reachable only from Paid) |
| order-total-positive | orders child (guard on place) |
| payment-single-capture | payments child (structural: capture only from Requested) |

## Toolchain and versions

The children pin their own toolchains. Parent-level verification: machinery
pack generate + machinery check (g2, g5) + the children's own gates.
