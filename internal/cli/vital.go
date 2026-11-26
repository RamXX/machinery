package cli

import (
	"fmt"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/vital"
)

func init() {
	Register(&Command{Name: "vital candidates", Summary: "List the design ids that look vital and are in no vital area.", Run: runVitalCandidates})
}

func runVitalCandidates(e *Env, args []string) int {
	fs := e.Flags("vital candidates")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery vital candidates takes no arguments")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	cands, dismissed := vital.Candidates(design.Load(p))
	for _, c := range cands {
		fmt.Fprintln(e.Stdout, c.Line())
	}
	fmt.Fprintln(e.Stdout, vital.Summary(len(cands), dismissed))
	return 0
}
