package cli

import (
	"fmt"

	"github.com/RamXX/machinery/v2/internal/stages"
	"github.com/RamXX/machinery/v2/internal/testpolicy"
)

func init() {
	Register(&Command{Name: "tests", Summary: "Check that tests are end to end: no unit tests, no test doubles.", Run: runTests})
}

func runTests(e *Env, args []string) int {
	fs := e.Flags("tests")
	write := fs.Bool("write-baseline", false, "record the current violations in the baseline file")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery tests takes no arguments")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	if *write && testpolicy.Applies(p) {
		vs, _, err := testpolicy.Violations(p)
		if err == nil {
			err = testpolicy.WriteBaseline(p, vs)
		}
		if err != nil {
			return e.Fail(err)
		}
		fmt.Fprintf(e.Stdout, "tests: wrote %d baseline entries to %s\n", len(vs), testpolicy.BaselinePath(p))
	}
	return e.RunStages("tests", &stages.Context{P: p}, []string{"tests"})
}
