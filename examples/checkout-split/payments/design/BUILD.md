# BUILD: Payments Subsystem

Mode: full (self-contained).

The payments half of checkout-split, built from this document plus the design/
tree; design/pack/ is the parent's frozen interface.

## 1. What this is

One service owning the Payment entity, coupled to orders only through the
three boundary events in design/pack/events.md (request in; markPaid,
markDeclined out).

## 2. Domain model

Source of truth: design/payments.modelith.yaml. The Payment entity and
PaymentStatus enum are the pack's frozen public shape.

## 3. Behavior

design/machines/Payment.machine.json; the generated Payment.oracle.md is the
canonical transition test spec (stable ids). The contract refinement
(design/formal/PaymentPackRefinement.tla) proves the machine refines the
pack's PaymentsContract; verify-formal TLC-checks it.

## 4. Traceability

| invariant | enforcement |
|---|---|
| payment-single-capture | structural: `capture` is handled only in `Requested`; redelivery lands in `_ignores` on `Captured` |

## 5. Test specification

Transition tests from Payment.oracle.md keyed on stable ids; named-unit tests
from the matrix table (a); redelivery property test per the `_ignores` reasons.

## 6. State migration

`Payment.status` is persisted. No persisted instances yet; the protocol
applies from first deployment: any change to a PaymentStatus value ships with
a mapping table or drain rule here, and is a PARENT change (frozen shape).

## 7. Toolchain and versions

Go 1.26, stdlib testing; machinery oracle design/machines; machinery check
design; machinery pack refine design; machinery verify-formal design.

## 8. Hard-TDD protocol

Test-writer derives tests from sections 4-5 keyed on oracle stable ids; tests
lock; the implementer makes them pass without editing them.
