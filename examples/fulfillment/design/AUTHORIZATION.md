# System authorization inventory

<!-- machinery:authorization-inventory -->

Each autonomous write is admitted by the internal capability of its owning component. This table
is closed against every Modelith action whose actor is `System`.

| authorization subject | admission |
|---|---|
| `Customer.register` | `orderSvc.customer.write` |
| `Inventory.reserve` | `inventorySvc.inventory.write` |
| `Inventory.release` | `inventorySvc.inventory.write` |
| `Inventory.commit` | `inventorySvc.inventory.write` |
| `Order.confirm` | `orderSvc.order.write` |
| `Order.markReserved` | `orderSvc.order.write` |
| `Order.markPaid` | `orderSvc.order.write` |
| `Order.markShipped` | `orderSvc.order.write` |
| `Order.markDelivered` | `orderSvc.order.write` |
| `Order.fail` | `orderSvc.order.write` |
| `Reservation.hold` | `inventorySvc.reservation.write` |
| `Reservation.commit` | `inventorySvc.reservation.write` |
| `Reservation.release` | `inventorySvc.reservation.write` |
| `FulfillmentSaga.start` | `orderSvc.saga.write` |
| `FulfillmentSaga.advance` | `orderSvc.saga.write` |
| `FulfillmentSaga.compensate` | `orderSvc.saga.write` |
| `FulfillmentSaga.complete` | `orderSvc.saga.write` |
| `FulfillmentSaga.abort` | `orderSvc.saga.write` |
| `OutboxMessage.enqueue` | `orderSvc.outbox.write` |
| `OutboxMessage.publish` | `orderSvc.outbox.write` |
| `OutboxMessage.markConsumed` | `orderSvc.outbox.write` |
