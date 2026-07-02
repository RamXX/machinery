workspace "TermCRM" "Terminal CRM for a small sales team over an embedded local graph database" {

  model {
    admin   = person "Admin" "Operates the system; may act on any record"
    manager = person "Manager" "Oversees a team; reads and writes the team's records"
    rep     = person "Sales Rep" "Owns and works their own records; reads across their team"
    reader  = person "Read-only User" "Reads records for reporting"

    crm = softwareSystem "TermCRM" "Single self-contained command-line binary; no server, no network" {
      cli    = container "CLI" "Command parsing, session I/O, human-readable output and exit codes" "Go"
      app    = container "Application" "Command execution envelope, authentication, authorization, load-act-save orchestration" "Go"
      domain = container "Domain" "Deal and Task state machines, guards, transition logic, invariant predicates" "Go"
      repo   = container "Repository" "Graph persistence, optimistic locking, integrity check, backup and restore" "Go"
      model  = container "Model" "Entity types and enums shared by every layer" "Go"
      store  = container "Store" "Embedded local graph database" "LadybugDB (github.com/LadybugDB/go-ladybug)" "Database"
    }

    admin   -> cli "Runs commands" "TTY"
    manager -> cli "Runs commands" "TTY"
    rep     -> cli "Runs commands" "TTY"
    reader  -> cli "Runs read commands" "TTY"

    cli    -> app    "Dispatches a parsed command with the caller's session"
    cli    -> model  "Formats model types for output"
    app    -> domain "Runs pure state-machine transitions"
    app    -> repo   "Loads and saves aggregates; opens, backs up, restores the store"
    app    -> model  "Reads and builds model types"
    domain -> model  "Reads model types and enums"
    repo   -> model  "Maps graph nodes to model types"
    repo   -> store  "Reads and writes nodes with optimistic version checks" "go-ladybug"
  }

  views {
    systemContext crm "context" { include *; autoLayout }
    container crm "containers" { include *; autoLayout }
    deployment crm "laptop" { include *; autoLayout }

    styles {
      element "Database" { shape Cylinder }
      element "Software System" { background #1168bd; color #ffffff }
      element "Person" { shape Person; background #08427b; color #ffffff }
    }
  }
}
