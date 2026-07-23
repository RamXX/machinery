// Code generated from fulfillment.modelith.yaml + integrity.relational.yaml by machinery alloy. DO NOT EDIT.
// machinery-version: v0.3.5-dev
//
// Static relational model of the STRUCTURAL invariants: which configurations
// of entities, relationships, unique keys, and singleton flags the constraint
// set admits. This rung checks ADMISSIBILITY, not safety: the commands are
// runs (a satisfying instance must exist), and no instance is the finding.
//
// ASSUMPTIONS (what this abstraction erases; results are conditional on them):
//   1. Bounded search: exhaustive only within scope 6 (at most 6 atoms per
//      signature). A run that finds no instance within the bound is a FAIL:
//      the constraint set contradicts itself, or cannot populate to scale.
//   2. Statics only: this rung checks admissible configurations, never how
//      the system moves between them; lifecycle properties are the TLC rung.
//   3. Only the named constraints are modeled: cardinality of the listed
//      relationships, uniqueness of the listed attributes, and the listed
//      singleton flags. Attribute values are abstract atoms (an injective
//      relation into an unconstrained domain), never their concrete types.
//   4. Inverse exclusivity of the declared 1:1 / 1:n edges is enforced as a
//      fact (Cardinality_*), an axiom of the model. No check command restates
//      it: a check identical to a fact is a tautology that can never fail.

module Integrity

sig Customer {
  email: one Val_Customer_Email   // unique key
}

sig Order {
  customer: one Customer,  // customer -> Customer (n:1, mandatory)
  payment: one Payment   // payment -> Payment (1:1, mandatory)
}

sig Payment {}

// 1:1: the Order side of Order -> Payment is exclusive (enforced as a fact)
fact Cardinality_Order_Payment {
  all target: Payment | lone target.~payment
}

// value domains: one atom per distinct attribute value (bounded types
// get exactly as many atoms as the type admits)
sig Val_Customer_Email {}

// customer-email-unique: no two Customer records share email
fact Unique_Customer_Email {
  all disj x, y: Customer | x.email != y.email
}

// --- Generated meta-checks: the admissibility suite, identical for every design ---

// PASS = instance found: the whole constraint set is jointly satisfiable with
// every modeled entity inhabited. No instance = the constraints contradict each
// other (a structural over-specification the linter cannot see).
run SomeWorld {
  some Customer and some Order and some Payment
} for 6

// PASS = instance found: every modeled entity can hold 3 records
// simultaneously under all constraints. No instance = a cardinality or
// uniqueness constraint starves the model (it admits a token world but cannot
// scale), which is a structural defect masquerading as a valid design.
run Populatable {
  #Customer >= 3 and
  #Order >= 3 and
  #Payment >= 3
} for 6

// PASS = instance found: two Customer records with different email coexist, so the
// unique key is a real constraint over a non-trivial value domain, not a
// fact that vacuously forces at most one record.
run Distinct_Customer_Email { some disj x, y: Customer | x.email != y.email } for 6
