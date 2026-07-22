# BUILD: Checkout Split (parent manifest)

Mode: manifest (recursive decomposition; the buildable designs are the children).

This root fixes what the children may not change: the domain model
(checkout.modelith.yaml), the container topology and boundary rules
(workspace.dsl + the Architecture Contract), the event contracts, and the two
contract machines under contracts/. Each subsystem's pack under packs/ is the
GENERATED, frozen interface its child design consumes; regenerate with
machinery pack generate, never edit. The delivery-topology decision (both
children isolated, contract stand-ins required) is in DECISIONS.md.

## Glossary

The parent's ubiquitous language; the packs carry the slice each child sees.

- **Order** - a customer order; owned by the orders subsystem. Lifecycle enum `OrderStatus`
  (Placed, Paid, Shipped, Declined, Cancelled), frozen by the orders pack.
- **Payment** - a payment attempt for one order; owned by the payments subsystem. Lifecycle enum
  `PaymentStatus` (Requested, Captured, Declined, Refunded), frozen by the payments pack.
- **Subsystem** - one independently built and tested half of the split (orders, payments), each a
  full machinery design consuming its pack.
- **Pack** - the generated, frozen interface between this parent and one child: the owned domain
  slice, the boundary event rows, the delegated invariants, the contract machine, and a content
  hash. Regenerated only here (`machinery pack generate`).
- **Contract machine** - the abstract protocol a subsystem's neighbors rely on (contracts/;
  plain on-transitions and finals). Each child's exposed machine must refine its own contract
  (`machinery pack refine`); each child's stand-in executes its neighbor's contract.
- **Boundary event** - one of the three bus events in the event-contract table: `request`
  (orders to payments), `markPaid` and `markDeclined` (payments to orders). There are no other
  cross-subsystem channels.
- **Contract stand-in** - an executable of a neighbor's signed, frozen contract, used by an
  isolated child in place of the real neighbor; not a mock (see DECISIONS.md and each child's
  section 4.7).

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

## Cross-context test spec

What each child proves alone: conformance to its own oracles, refinement of its own contract
machine (TLC-checked via `machinery verify-formal`), and conformance of its neighbor stand-in to
the neighbor's contract oracle (each child's section 4.7). Each child's section 11 carries the
gate-anchored hard-TDD protocol its suite is built under.

What only this parent can prove, the cross-context assembly suite run when both children are
deployed together against the real broker:

- One end-to-end happy path: place an order, observe `request`, capture, observe `markPaid`, ship.
  One end-to-end decline path: place, `request`, decline, `markDeclined`, order Declined.
- At-least-once fault injection across the real seam: duplicate and reordered deliveries of all
  three events between the REAL services (the children prove this only against stand-ins).
- The parent residuals, named: end-to-end checkout latency (no contract-machine vocabulary for
  time); liveness across the two mutually dependent contracts (each child's proof covers only its
  own termination; cross-contract reliance is kept to safety plus each contract's own
  termination); and any unmodeled channel (nothing besides the three events may couple the
  services; the assembly environment asserts no other connectivity exists).

A failure in this suite is a PARENT defect (an event-contract or contract-machine gap): fix here,
regenerate packs, and the pack diff is each child's affected-obligation list. It is never fixed by
editing a child against its pack.

## Toolchain and versions

The children pin their own toolchains. Parent-level verification: machinery
pack generate + machinery check (g2, g5) + the children's own gates.
