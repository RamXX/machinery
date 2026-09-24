// Package datalog is machinery's in-process Datalog evaluator: a strict
// subset of Soufflé evaluated semi-naively with stratified negation and
// stratified count/min aggregation, standard library only.
//
// # The subset
//
// A program is a sequence of these items and nothing else:
//
//	.decl name(attr:symbol, attr:number, ...)   // one or more attributes
//	.input name                                 // no IO parameters
//	.output name                                // no IO parameters
//	head(T, ...) :- literal, literal, ... .
//
// A term is a variable (any identifier), the wildcard _, or a string
// constant in double quotes. A body literal is one of:
//
//	rel(T, ...)                     positive atom
//	!rel(T, ...)                    negated atom
//	T = T    T != T                 comparison over bound terms
//	V = count : { literal, ... }    count aggregate
//	V = min X : { literal, ... }    min aggregate over a number variable X
//
// Comments are // to end of line and /* ... */ (not nested). Every other
// Soufflé construct (records, ADTs, arithmetic, number literals, functors,
// .type, .plan, .printsize, .functor, .comp, .init, .pragma, relation
// qualifiers such as brie, choice-domain or inline, subsumption, disjunction,
// multiple heads, ground facts in the program text, the preprocessor, max,
// sum and mean) is a parse error carrying the offending line and column.
//
// The only data type is symbol. number is accepted as an attribute type only
// for relations that carry aggregate results; an .input relation must be all
// symbol. min is accepted only over number variables, because Soufflé rejects
// min over symbols. String constants in the program may not contain a
// backslash, tab, newline or carriage return (Soufflé would interpret escape
// sequences; the subset refuses them instead of emulating them).
//
// # Semantics
//
// Rules are range-restricted: every head variable, every variable of a
// negated atom and every variable of a comparison must be bound by a positive
// body atom (or be the result variable of an aggregate). A variable inside an
// aggregate body that also occurs outside it is a grouping variable and must
// be bound by a positive atom outside the aggregate; the aggregate is computed
// per binding of its grouping variables, as Soufflé does. count over an empty
// group yields 0; min over an empty group yields no tuple.
//
// Relations are grouped into strongly connected components of the dependency
// graph; a negated or aggregated dependency inside a component is a compile
// error naming the cycle. Components are evaluated in topological order, each
// to a fixed point by semi-naive iteration. Relations are sets.
//
// An .input relation may also have rules (Soufflé allows it): its facts are
// seeded first and the rules add to them. An .output on an .input relation is
// allowed and writes the deduplicated facts. Supplied inputs for relations the
// program does not declare .input are ignored, as Soufflé ignores stray fact
// files; a missing input for a declared .input relation is an error, as it is
// in Soufflé.
//
// # Parity contract and sort convention
//
// Every program this package accepts runs unchanged under Soufflé and yields
// the same output relations. The parity test in this package runs every
// program under testdata/programs through both engines when the souffle
// binary is on PATH and byte-compares the outputs. Soufflé does not guarantee
// the order of rows in its output files, so the comparison sorts the lines of
// both outputs bytewise before comparing.
//
// Result.Output and Result.WriteCSV return rows sorted column by column:
// symbol columns compare bytewise, number columns compare numerically. The
// files WriteCSV writes use Soufflé's default output format: one row per line,
// columns separated by a tab, a trailing newline, no header, and an empty file
// for an empty relation. Fact files use the same format; a trailing carriage
// return on a line is dropped, as Soufflé drops it. A symbol can therefore
// never contain a tab, newline or carriage return: such values are rejected
// wherever they enter (Go inputs, program constants, fact files).
//
// # Derivations
//
// With Options.Explain, the evaluator records for every derived tuple the rule
// (1-based index in source order, and its position) and the tuples matched by
// the rule's positive body atoms on the tuple's first derivation. Evaluation
// order is fixed (inputs are sorted, rules run in source order), so the
// recorded derivation is deterministic. Result.Explain returns it as a tree
// whose leaves are input facts.
package datalog
