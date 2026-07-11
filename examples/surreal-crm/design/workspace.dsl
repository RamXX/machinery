workspace "Surreal CRM" "The go-crm CLI rebuilt over SurrealDB on a local Docker instance; same domain, new persistence foundation." {

  model {
    admin    = person "Admin" "Operates the CRM with unrestricted access."
    manager  = person "Manager" "Reads and writes the records of the manager's team."
    rep      = person "Rep" "Owns and works records; reads across the team."
    readonly = person "ReadOnly" "Reads records within scope."

    crm = softwareSystem "Surreal CRM" "Command-line CRM over a SurrealDB instance running in a local Docker container." {

      cli = container "crm binary" "Native command-line binary: parse, resolve session, authorize, execute in a transaction, render." "Go / cobra" {
        commands = component "Command Layer" "Parses args and flags, owns the database connection and transaction boundary, renders tables or JSON." "Go / cobra"
        session  = component "Session and Auth" "login and logout, password hashing, session-token file, current-user resolution." "Go / argon2id"
        authz    = component "Authorization (RBAC)" "Decides allow or deny for (user, verb, entityType, ownerId, teamId). Enforces the rbac-* invariants." "Go"
        domain   = component "Domain Services" "Aggregates and lifecycle logic for Deal, Task, and User; invariant guards; state transitions." "Go"
        repo     = component "Repository" "Maps domain operations to SurrealQL, executes them over the SurrealDB connection, maps rows and errors back to the domain." "Go / surrealdb.go"
        model    = component "Domain Model" "Shared vocabulary: entity types, enums, and typed errors. Imports nothing." "Go"
      }

      store       = container "SurrealDB" "Multi-model store holding every record: one table per aggregate, record links for relationships." "SurrealDB 2.x in Docker" "Database"
      sessionfile = container "Session File" "Local expiring session token at ~/.crm/session." "OS filesystem" "Database"
      dockerd     = container "Docker Engine" "Runs and restarts the SurrealDB container; owns the data volume." "Docker" "External"
    }

    admin    -> cli "Runs commands" "CLI"
    manager  -> cli "Runs commands" "CLI"
    rep      -> cli "Runs commands" "CLI"
    readonly -> cli "Runs commands" "CLI"

    commands -> session "Resolves the current user"
    commands -> domain  "Executes the requested action"
    commands -> repo    "Opens the connection and owns the transaction boundary"
    session  -> repo    "Loads the user and verifies the password hash"
    session  -> sessionfile "Reads and writes the session token"
    domain   -> authz   "Authorizes the action against record owner and team"
    domain   -> repo    "Reads and writes records"
    domain   -> model   "Uses the shared domain types"
    repo     -> store   "Executes SurrealQL in a transaction" "surrealdb.go / WebSocket"
    repo     -> model   "Maps rows to the shared domain types"
    store    -> dockerd "Runs inside" "Container"
  }

  views {
    systemContext crm "Context" "Who uses the CRM." {
      include *
      autoLayout lr
    }

    container crm "Containers" "The binary, the SurrealDB container, and the local session file." {
      include *
      autoLayout lr
    }

    component cli "Components" "Inside the crm binary." {
      include *
      autoLayout
    }

    styles {
      element "Person" {
        shape Person
        background #438DD5
        color #ffffff
      }
      element "Software System" {
        background #2E6295
        color #ffffff
      }
      element "Container" {
        background #438DD5
        color #ffffff
      }
      element "Component" {
        background #6FA8DC
        color #ffffff
      }
      element "Database" {
        shape Cylinder
      }
      element "External" {
        background #999999
        color #ffffff
      }
    }
  }
}
