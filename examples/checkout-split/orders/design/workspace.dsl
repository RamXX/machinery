workspace "Orders Subsystem" "The orders half of checkout-split." {
  model {
    customer = person "Customer" "Places orders."
    sys = softwareSystem "Orders" "The orders service." {
      orders   = container "Orders Service" "Order lifecycle." "Go"
      bus      = container "Bus" "The message broker (shared)." "NATS" "Queue"
      ordersdb = container "Orders DB" "Order state." "Postgres" "Database"
    }
    customer -> orders "Places orders"
    orders -> bus "Publishes request; consumes markPaid, markDeclined"
    orders -> ordersdb "Persists orders"
  }
}
