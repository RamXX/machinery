# System authorization inventory

<!-- machinery:authorization-inventory -->

Each autonomous write names a declared C4 component as its admission subject. This table
is closed against every Modelith action whose actor is `System`.

| authorization subject | admission |
|---|---|
| `Customer.register` | `orderSvc` |
| `Inventory.reserve` | `inventorySvc` |
| `Inventory.release` | `inventorySvc` |
| `Inventory.commit` | `inventorySvc` |
| `Order.confirm` | `orderSvc` |
| `Order.markReserved` | `orderSvc` |
| `Order.markPaid` | `orderSvc` |
| `Order.markShipped` | `orderSvc` |
| `Order.markDelivered` | `orderSvc` |
| `Order.fail` | `orderSvc` |
| `Reservation.hold` | `inventorySvc` |
| `Reservation.commit` | `inventorySvc` |
| `Reservation.release` | `inventorySvc` |
| `FulfillmentSaga.start` | `orderSvc` |
| `FulfillmentSaga.advance` | `orderSvc` |
| `FulfillmentSaga.compensate` | `orderSvc` |
| `FulfillmentSaga.complete` | `orderSvc` |
| `FulfillmentSaga.abort` | `orderSvc` |
| `OutboxMessage.enqueue` | `orderSvc` |
| `OutboxMessage.publish` | `orderSvc` |
| `OutboxMessage.markConsumed` | `orderSvc` |
