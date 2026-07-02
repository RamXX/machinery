workspace "Order Fulfillment" "Distributed fulfillment platform: a saga coordinates reserve, pay, and ship across services with compensation and reliable messaging." {

  model {
    customer = person "Customer" "Places and tracks orders."
    operator = person "Operator" "Manages catalog and stock."

    stripe = softwareSystem "Payment Gateway" "Authorizes, captures, and refunds." "External"
    carrier = softwareSystem "Carrier" "Dispatches and tracks parcels." "External"

    plat = softwareSystem "Fulfillment Platform" "Takes an order to delivery across services." {
      orderSvc = container "Order Service" "Owns orders and the fulfillment saga." "Elixir/Phoenix" {
        api = component "API" "Accepts and reports orders." "Phoenix"
        saga = component "Saga Orchestrator" "Drives reserve, pay, ship; compensates on failure." "gen_statem"
        orderRepo = component "Order Repository" "Reads and writes orders." "Ecto"
        outbox = component "Outbox" "Writes events in the order transaction, publishes at least once." "Ecto + Oban"
      }
      inventorySvc = container "Inventory Service" "Holds and releases stock." "Elixir"
      paymentSvc = container "Payment Service" "Captures and refunds via the gateway." "Elixir"
      shippingSvc = container "Shipping Service" "Dispatches via the carrier." "Elixir"
      bus = container "Message Bus" "Commands and events between services." "RabbitMQ" "Queue"
      orderDb = container "Order DB" "Orders, saga state, outbox." "PostgreSQL" "Database"
      inventoryDb = container "Inventory DB" "Stock and reservations." "PostgreSQL" "Database"
      paymentDb = container "Payment DB" "Payments and refunds." "PostgreSQL" "Database"
      shippingDb = container "Shipping DB" "Shipments." "PostgreSQL" "Database"
    }

    customer -> api "Places and tracks orders" "HTTPS"
    operator -> inventorySvc "Manages stock" "HTTPS"

    api -> orderRepo "Reads/writes orders"
    api -> saga "Starts fulfillment"
    saga -> orderRepo "Reads/writes saga state"
    saga -> outbox "Emits commands"
    orderRepo -> orderDb "SQL"
    outbox -> orderDb "SQL"
    outbox -> bus "Publishes at least once" "AMQP"

    bus -> inventorySvc "reserve / release" "AMQP"
    bus -> paymentSvc "capture / refund" "AMQP"
    bus -> shippingSvc "dispatch" "AMQP"
    inventorySvc -> bus "reserved / released" "AMQP"
    paymentSvc -> bus "captured / refunded / failed" "AMQP"
    shippingSvc -> bus "dispatched / delivered / lost" "AMQP"

    inventorySvc -> inventoryDb "SQL"
    paymentSvc -> paymentDb "SQL"
    paymentSvc -> stripe "Charges" "REST"
    shippingSvc -> shippingDb "SQL"
    shippingSvc -> carrier "Ships" "REST"
  }

  views {
    systemContext plat "Context" { include *; autoLayout lr }
    container plat "Containers" { include *; autoLayout }
    component orderSvc "OrderService" { include *; autoLayout }
    deployment plat "production" "Deployment" { include *; autoLayout }
    styles {
      element "Person" { shape Person; background #08427B; color #ffffff }
      element "Software System" { background #1168BD; color #ffffff }
      element "Container" { background #438DD5; color #ffffff }
      element "Component" { background #85BBF0; color #000000 }
      element "Database" { shape Cylinder }
      element "Queue" { shape Pipe }
      element "External" { background #999999 }
    }
  }
}
