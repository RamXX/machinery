// Code generated from domain.modelith.yaml + integrity.relational.yaml by machinery alloy. DO NOT EDIT.
// machinery-version: v0.3.4-dev
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

module Integrity

sig User {
  username: one Val_User_Username   // unique key
}

sig Team {
  name: one Val_Team_Name   // unique key
}

sig Tag {
  name: one Val_Tag_Name   // unique key
}

sig Pipeline {
  isDefault: one Bool   // singleton flag
}

// boolean domain for singleton flags
abstract sig Bool {}
one sig True, False extends Bool {}

// value domains: one atom per distinct attribute value (bounded types
// get exactly as many atoms as the type admits)
sig Val_User_Username {}
sig Val_Team_Name {}
sig Val_Tag_Name {}

// username-unique: no two User records share username
fact Unique_User_Username {
  all disj x, y: User | x.username != y.username
}

// team-name-unique: no two Team records share name
fact Unique_Team_Name {
  all disj x, y: Team | x.name != y.name
}

// tag-name-unique: no two Tag records share name
fact Unique_Tag_Name {
  all disj x, y: Tag | x.name != y.name
}

// one-default-pipeline: exactly one Pipeline record has isDefault set
fact Singleton_Pipeline_IsDefault {
  one x: Pipeline | x.isDefault = True
}

// --- Generated meta-checks: the admissibility suite, identical for every design ---

// PASS = instance found: the whole constraint set is jointly satisfiable with
// every modeled entity inhabited. No instance = the constraints contradict each
// other (a structural over-specification the linter cannot see).
run SomeWorld {
  some User and some Team and some Tag and some Pipeline
} for 6

// PASS = instance found: every modeled entity can hold 3 records
// simultaneously under all constraints. No instance = a cardinality or
// uniqueness constraint starves the model (it admits a token world but cannot
// scale), which is a structural defect masquerading as a valid design.
run Populatable {
  #User >= 3 and
  #Team >= 3 and
  #Tag >= 3 and
  #Pipeline >= 3
} for 6

// PASS = instance found: two User records with different username coexist, so the
// unique key is a real constraint over a non-trivial value domain, not a
// fact that vacuously forces at most one record.
run Distinct_User_Username { some disj x, y: User | x.username != y.username } for 6

// PASS = instance found: two Team records with different name coexist, so the
// unique key is a real constraint over a non-trivial value domain, not a
// fact that vacuously forces at most one record.
run Distinct_Team_Name { some disj x, y: Team | x.name != y.name } for 6

// PASS = instance found: two Tag records with different name coexist, so the
// unique key is a real constraint over a non-trivial value domain, not a
// fact that vacuously forces at most one record.
run Distinct_Tag_Name { some disj x, y: Tag | x.name != y.name } for 6
