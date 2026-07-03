package experiments

// Runner registry: the enforcement side of the "do not remove or weaken an
// entry" contract. Every runner (a test that applies an experiment's mutation
// and asserts the finding) registers the experiment names it covers; one test
// (TestEveryDeclaredExperimentHasARunner) fails when a declared experiment has
// no registered runner. Registering an undeclared name panics, so a stale
// registration breaks just as loudly as a missing one.
//
// The registry enforces NAME coverage only: it cannot verify that the
// registering test actually applies the mutation and asserts the finding.
// That half of the contract lives in review; a registration whose test
// asserts nothing is a lie the registry cannot catch.

import "sort"

var runnerRegistry = map[string]string{}

// RegisterRunner records that the named runner covers the given experiments.
// Call it from an init() adjacent to the covering tests.
func RegisterRunner(runner string, names ...string) {
	declared := map[string]bool{}
	for _, e := range All() {
		declared[e.Name] = true
	}
	for _, n := range names {
		if !declared[n] {
			panic("experiments: runner " + runner + " registers unknown experiment " + n + "; the declared tables in experiments.go are the source of truth")
		}
		runnerRegistry[n] = runner
	}
}

// UncoveredExperiments returns every declared experiment with no registered
// runner, sorted.
func UncoveredExperiments() []string {
	var out []string
	for _, e := range All() {
		if _, ok := runnerRegistry[e.Name]; !ok {
			out = append(out, e.Name)
		}
	}
	sort.Strings(out)
	return out
}
