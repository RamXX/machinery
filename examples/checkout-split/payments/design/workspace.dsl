workspace "Payments Subsystem" "The payments half of checkout-split." {
  model {
    sys = softwareSystem "Payments" "The payments service." {
      payments = container "Payments Service" "Payment settlement." "Go"
      bus      = container "Bus" "The message broker (shared)." "NATS" "Queue"
      paydb    = container "Payments DB" "Payment state." "Postgres" "Database"
    }
    orders = softwareSystem "Orders Service" "The peer subsystem this one exchanges settlement events with over the bus." "External"
    payments -> bus "Consumes request; publishes markPaid, markDeclined"
    payments -> paydb "Persists payments"
  }
}
