package e2e

import (
	"flag"
	"os"
	"regexp"
)

// probeFilter narrows a probe run (docs/formats.md section 20) to the one
// case it mutated, named by MACHINERY_PROBE_CASE. machinery probe mutates
// transition cases only, and every transition case is proven by its own
// subtest of TestTransitions, so only that subtest runs: every other test is
// filtered out before it starts and emits nothing. Without the variable
// nothing changes. It runs from TestMain, before the tests.
func probeFilter() error {
	id := os.Getenv("MACHINERY_PROBE_CASE")
	if id == "" {
		return nil
	}
	flag.Parse()
	return flag.Set("test.run", "^TestTransitions$/^"+regexp.QuoteMeta(id)+"$")
}
