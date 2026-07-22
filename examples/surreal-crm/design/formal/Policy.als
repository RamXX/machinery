// Code generated from domain.modelith.yaml + policy.relational.yaml by machinery alloy. DO NOT EDIT.
// machinery-version: v0.3.4-dev
//
// Static relational model of the policy invariants: which configurations of
// subjects, teams, and record ownership the invariant set admits. Alloy
// searches every configuration within the bound, so a green check is
// exhaustive there and silent beyond it.
//
// ASSUMPTIONS (what this abstraction erases; results are conditional on them):
//   1. Bounded search: exhaustive only within scope 6 (at most 6 atoms per
//      signature). PASS on a check means no counterexample within the bound.
//   2. Statics only: this rung checks admissible configurations, never how
//      the system moves between them; lifecycle and temporal properties are
//      the TLC rung (machinery tla / machinery refine).
//   3. Only what the annotation names is modeled: roles, team membership,
//      ownership. Sessions, verb semantics in code, and every other
//      attribute are carried by the machines and the implementation tests.
//   4. Every resource entity is policy-equivalent: one Record signature
//      with exactly one owner stands for all of them (the policy treats
//      them alike). A single sig, not subsignatures, so the solver's
//      receipt renders the owner relation in every counterexample.
//
// RESIDUALS (invariants the algebra does not carry; enforced elsewhere):
//   - session-active-user: behavioral; enforced by the Session machine and checked at the TLC rung

module Policy

// role enum UserRole
abstract sig Role {}
one sig Admin, Manager, Rep, ReadOnly extends Role {}

// team scoping: membership lone (single-team, manager-has-team)
sig Team {}

// subjects: User
sig User {
  role: one Role,
  team: lone Team
}

// roles that must hold a team (required_for)
fact RequiredTeams {
  all u: User | u.role in Manager implies some u.team
}

// resources: Account, Contact, Deal, Task, Activity
// every record has exactly one owner (account-owned, contact-owned, deal-owned, task-owned, activity-owned)
sig Record { owner: one User }

// a member of the subject's team: both hold a team and it is the same one;
// a teamless subject is nobody's teammate
pred sameTeam[a, b: User] { some a.team and a.team = b.team }

// rbac-crud-verbs: verb capability by role
pred grantsCreate[u: User] { u.role in (Admin + Manager + Rep) }
pred grantsRead[u: User] { u.role in (Admin + Manager + Rep + ReadOnly) }
pred grantsUpdate[u: User] { u.role in (Admin + Manager + Rep) }
pred grantsDelete[u: User] { u.role in (Admin + Manager + Rep) }

// rbac-read-visibility: read scope by role
pred canRead[u: User, r: Record] {
  grantsRead[u] and (
    u.role = Admin
    or u.role in (Manager + Rep + ReadOnly) and (r.owner = u or sameTeam[u, r.owner])
  )
}

// rbac-write-scope: update/delete scope by role
// ReadOnly: none (no update scope)
pred canUpdate[u: User, r: Record] {
  grantsUpdate[u] and (
    u.role = Admin
    or u.role = Manager and sameTeam[u, r.owner]
    or u.role = Rep and r.owner = u
  )
}
// ReadOnly: none (no delete scope)
pred canDelete[u: User, r: Record] {
  grantsDelete[u] and (
    u.role = Admin
    or u.role = Manager and sameTeam[u, r.owner]
    or u.role = Rep and r.owner = u
  )
}

// rbac-reassign-authority, task-assignee-visible: who may change an owner, and where the record may go
pred canReassign[u: User, r: Record, t: User] {
  u.role = Admin and t in User // target: any
  or u.role = Manager and sameTeam[u, r.owner] and sameTeam[u, t] // target: team
}

// --- Generated meta-checks: the standard suite, identical for every design ---

// PASS = instance found: the invariants admit at least one world with every
// role inhabited and a record present. No instance = the policy contradicts
// itself (vacuity guard: everything below would pass emptily).
run SomeWorld {
  (all ro: Role | some u: User | u.role = ro) and some Record
} for 6

// PASS = no counterexample: anyone who may write a record may also read it
// (write scope stays inside read scope).
check WriteImpliesRead {
  all u: User, r: Record | (canUpdate[u, r] or canDelete[u, r]) implies canRead[u, r]
} for 6

// PASS = no counterexample: every role granted a write verb can write the
// records it owns. A counterexample here is an unstated assumption: some
// write-capable subject exists whose own records are outside its scope.
check CapableWritesOwn {
  all u: User, r: Record | (u.role in (Admin + Manager + Rep) and r.owner = u) implies canUpdate[u, r]
  all u: User, r: Record | (u.role in (Admin + Manager + Rep) and r.owner = u) implies canDelete[u, r]
} for 6

// PASS = no counterexample: a legal reassign leaves the record inside the
// actor's reassign authority (no one-step escape: the actor cannot hand a
// record somewhere it could not touch again).
check ReassignRetainsAuthority {
  all u, t: User, r: Record | canReassign[u, r, t] implies
    (u.role = Admin or u.role = Manager and sameTeam[u, t])
} for 6

// PASS = instance found, one per (role, granted verb): the grant is
// exercisable somewhere. No instance = the grant is vacuous (granted a
// verb whose scope is empty in every admissible world).
run Possible_Admin_read { some u: User, r: Record | u.role = Admin and canRead[u, r] } for 6
run Possible_Admin_update { some u: User, r: Record | u.role = Admin and canUpdate[u, r] } for 6
run Possible_Admin_delete { some u: User, r: Record | u.role = Admin and canDelete[u, r] } for 6
run Possible_Manager_read { some u: User, r: Record | u.role = Manager and canRead[u, r] } for 6
run Possible_Manager_update { some u: User, r: Record | u.role = Manager and canUpdate[u, r] } for 6
run Possible_Manager_delete { some u: User, r: Record | u.role = Manager and canDelete[u, r] } for 6
run Possible_Rep_read { some u: User, r: Record | u.role = Rep and canRead[u, r] } for 6
run Possible_Rep_update { some u: User, r: Record | u.role = Rep and canUpdate[u, r] } for 6
run Possible_Rep_delete { some u: User, r: Record | u.role = Rep and canDelete[u, r] } for 6
run Possible_ReadOnly_read { some u: User, r: Record | u.role = ReadOnly and canRead[u, r] } for 6
