workspace "DrawdownRecommender" "Local tool that recommends a 16-stock, lowest-drawdown portfolio from deduped index constituents" {

  model {
    analyst = person "Analyst" "Configures indices and starts recommendation runs"
    manager = person "Manager" "Reviews a recommended portfolio and accepts or rejects it"

    pf = softwareSystem "DrawdownRecommender" "Single local command-line tool; no server" {
      cli       = container "CLI" "Command parsing, output, exit codes" "Python"
      app       = container "Application" "Run orchestration, review commands, load-act-save" "Python"
      domain    = container "Domain" "RecommendationRun and Portfolio state machines, guards, invariant predicates" "Python"
      optimizer = container "Optimizer" "Pure min-drawdown selection of 16 of N candidates" "Python"
      feed      = container "Feed" "Market-data adapter with a circuit breaker" "Python"
      repo      = container "Repository" "Local persistence, optimistic locking, integrity, backup/restore" "Python"
      model     = container "Model" "Entity types and enums shared by every layer" "Python"
      store     = container "Store" "Embedded local columnar database file" "DuckDB (duckdb)" "Database"
    }
    mkt = softwareSystem "MarketData" "Third-party price and constituent provider over HTTP" "External"

    analyst -> cli "Runs recommend and build commands" "TTY"
    manager -> cli "Runs review commands" "TTY"

    cli    -> app       "Dispatches a parsed command"
    cli    -> model     "Formats model types for output"
    app    -> domain    "Runs pure state-machine transitions"
    app    -> optimizer "Requests the 16-stock min-drawdown selection"
    app    -> feed      "Requests price history through the breaker"
    app    -> repo      "Loads and saves runs, portfolios, candidate sets"
    app    -> model     "Reads and builds model types"
    domain -> model     "Reads model types and enums"
    optimizer -> model  "Reads candidates and prices, returns holdings"
    feed   -> model     "Returns price series as model types"
    feed   -> mkt       "Fetches prices and constituents" "HTTPS"
    repo   -> model     "Maps rows to model types"
    repo   -> store     "Reads and writes with optimistic version checks" "SQL"
  }

  views {
    systemContext pf "context" {
      include *
      autoLayout
    }
    container pf "containers" {
      include *
      autoLayout
    }

    styles {
      element "Database" {
        shape Cylinder
      }
      element "External" {
        background #999999
      }
      element "Person" {
        shape Person
        background #08427b
        color #ffffff
      }
    }
  }
}
