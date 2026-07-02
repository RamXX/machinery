workspace "Go CRM" "Single-binary CRM with an embedded LadybugDB graph and role- and ownership-based access control." {

  model {
    admin    = person "Admin" "Operates the CRM with unrestricted access."
    manager  = person "Manager" "Reads and writes the records of the manager's team."
    rep      = person "Rep" "Owns and works records; reads across the team."
    readonly = person "ReadOnly" "Reads records within scope."

    crm = softwareSystem "Go CRM" "Command-line CRM over an embedded property graph." {

      cli = container "crm binary" "Native command-line binary: parse, resolve session, authorize, execute in a transaction, render." "Go / cobra" {
        commands = component "Command Layer" "Parses args and flags, owns the database open/close and transaction boundary, renders tables or JSON." "Go / cobra"
        session  = component "Session and Auth" "login and logout, password hashing, session-token file, current-user resolution." "Go / argon2id"
        authz    = component "Authorization (RBAC)" "Decides allow or deny for (user, verb, entityType, ownerId, teamId). Enforces the rbac-* invariants." "Go"
        domain   = component "Domain Services" "Aggregates and lifecycle logic for Deal, Task, and User; invariant guards; state transitions." "Go"
        repo     = component "Repository" "Maps domain operations to Cypher, executes them via go-ladybug, maps rows and errors back to the domain." "Go / go-ladybug"
        model    = component "Domain Model" "Shared vocabulary: entity types, enums, and typed errors. Imports nothing." "Go"
      }

      store       = container "Graph Store" "Embedded property graph on local disk: nodes and relationships for every record." "LadybugDB (embedded)" "Database"
      sessionfile = container "Session File" "Local expiring session token at ~/.crm/session." "OS filesystem" "Database"
    }

    admin    -> cli "Runs commands" "CLI"
    manager  -> cli "Runs commands" "CLI"
    rep      -> cli "Runs commands" "CLI"
    readonly -> cli "Runs commands" "CLI"

    commands -> session "Resolves the current user"
    commands -> domain  "Executes the requested action"
    commands -> repo    "Opens the database and owns the transaction boundary"
    session  -> repo    "Loads the user and verifies the password hash"
    session  -> sessionfile "Reads and writes the session token"
    domain   -> authz   "Authorizes the action against record owner and team"
    domain   -> repo    "Reads and writes records"
    domain   -> model   "Uses the shared domain types"
    repo     -> store   "Executes Cypher in a transaction" "go-ladybug / Cypher"
    repo     -> model   "Maps rows to the shared domain types"
  }

  views {
    systemContext crm "Context" "Who uses the CRM." {
      include *
      autoLayout lr
    }

    container crm "Containers" "The binary and its local stores." {
      include *
      autoLayout lr
    }

    component cli "Components" "Inside the crm binary." {
      include *
      autoLayout
    }

    deployment crm "production" "Deployment" {
      include *
      autoLayout
    }

    styles {
      element "Person"          { shape Person;  background #08427B; color #ffffff }
      element "Software System" { background #1168BD; color #ffffff }
      element "Container"       { background #438DD5; color #ffffff }
      element "Component"       { background #85BBF0; color #000000 }
      element "Database"        { shape Cylinder }
    }
  }
}
