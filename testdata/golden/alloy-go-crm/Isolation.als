// Code generated from domain.modelith.yaml + isolation.relational.yaml by machinery alloy. DO NOT EDIT.
//
// Static relational model of multi-tenant ISOLATION: the tenant of a record
// is its owner's tenant, and every reference the annotation names must stay
// inside one tenant. The policy layer checks who may touch a record; this
// layer checks that the records a record REFERENCES cannot belong to another
// tenant, so following a link never crosses a tenant boundary.
//
// ASSUMPTIONS (what this abstraction erases; results are conditional on them):
//   1. Bounded search: exhaustive only within scope 6 (at most 6 atoms per
//      signature). PASS on a check means no counterexample within the bound.
//   2. Statics only: this rung checks admissible configurations, never how
//      the system moves between them; lifecycle properties are the TLC rung.
//   3. tenant(record) = owner's tenant; a teamless owner has no tenant, and a
//      reference from or to a teamless owner is never same-tenant. Only the
//      named references are held; every other relationship is unmodeled.

module Isolation

// the tenant
sig Team {}

// subjects hold a tenant (membership lone)
sig User {
  team: lone Team
}

sig Task {
  owner: one User,  // tenant = owner's team
  deal: lone Deal   // reference -> Deal (n:1)
}

sig Deal {
  owner: one User   // tenant = owner's team
}

sig Activity {
  owner: one User,  // tenant = owner's team
  contact: lone Contact   // reference -> Contact (n:1)
}

sig Contact {
  owner: one User   // tenant = owner's team
}

// two subjects share a tenant: both hold one and it is the same. A teamless
// subject shares a tenant with nobody, so a link touching one is never
// same-tenant.
pred sameTenant[a, b: User] { some a.team and a.team = b.team }

// task-deal-same-tenant: a Task and the Deal it references are owned in the same tenant
fact Isolation_Task_Deal {
  all x: Task | some x.deal implies sameTenant[x.owner, x.deal.owner]
}

// activity-contact-same-tenant: an Activity and the Contact it references are owned in the same tenant
fact Isolation_Activity_Contact {
  all x: Activity | some x.contact implies sameTenant[x.owner, x.contact.owner]
}

// --- Generated meta-checks: the standard suite, identical for every design ---

// PASS = instance found: a genuinely multi-tenant world (two inhabited
// tenants) with at least one cross-entity link still exists. No instance =
// the isolation facts collapse tenancy (over-isolation: links force one
// tenant), which is as wrong as leaking across tenants.
run SomeWorld {
  (some disj ta, tb: Team | (some u: User | u.team = ta) and (some u: User | u.team = tb))
  and ((some x: Task | some x.deal) or (some x: Activity | some x.contact))
} for 6

// PASS = no counterexample: two Task records that reference the same Deal are
// owned in the same tenant. A counterexample would be a Deal referenced from
// two tenants at once -- a shared referent bridging the boundary.
check SharedReferent_Task_Deal {
  all x, y: Task | (some x.deal and x.deal = y.deal) implies sameTenant[x.owner, y.owner]
} for 6

// PASS = no counterexample: two Activity records that reference the same Contact are
// owned in the same tenant. A counterexample would be a Contact referenced from
// two tenants at once -- a shared referent bridging the boundary.
check SharedReferent_Activity_Contact {
  all x, y: Activity | (some x.contact and x.contact = y.contact) implies sameTenant[x.owner, y.owner]
} for 6

// PASS = instance found: a Task referencing a Deal exists, so the isolation
// fact is not vacuously satisfied by forbidding the link entirely.
run Possible_Task_Deal { some x: Task | some x.deal } for 6

// PASS = instance found: an Activity referencing a Contact exists, so the isolation
// fact is not vacuously satisfied by forbidding the link entirely.
run Possible_Activity_Contact { some x: Activity | some x.contact } for 6
