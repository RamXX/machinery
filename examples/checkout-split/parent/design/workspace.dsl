workspace "Checkout Split" "Two services coupled only through the bus." {
  model {
    customer = person "Customer" "Places orders."
    sys = softwareSystem "Checkout" "The split checkout." {
      orders   = container "Orders Service" "Order lifecycle." "Go"
      payments = container "Payments Service" "Payment settlement." "Go"
      bus      = container "Bus" "The message broker." "NATS" "Queue"
      ordersdb = container "Orders DB" "Order state." "Postgres" "Database"
      paydb    = container "Payments DB" "Payment state." "Postgres" "Database"
    }
    customer -> orders "Places orders"
    orders -> bus "Publishes request; consumes markPaid, markDeclined"
    payments -> bus "Consumes request; publishes markPaid, markDeclined"
    orders -> ordersdb "Persists orders"
    payments -> paydb "Persists payments"
  }
}
