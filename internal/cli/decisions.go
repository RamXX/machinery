package cli

import (
	"fmt"
	"strings"

	"github.com/RamXX/machinery/v2/internal/backlog"
	"github.com/RamXX/machinery/v2/internal/pm"
)

func init() {
	Register(&Command{Name: "decisions", Summary: "List the current decisions, by the ids they govern.", Run: runDecisions})
}

func runDecisions(e *Env, args []string) int {
	fs := e.Flags("decisions")
	var ids StringList
	fs.Var(&ids, "ids", "keep the decisions that govern one of these ids or patterns (repeatable, comma-separated)")
	vetoes := fs.Bool("open-vetoes", false, "keep the vetoable decisions nobody confirmed")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery decisions takes no arguments")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	c, err := backlog.New(p, "")
	if err != nil {
		return e.Fail(err)
	}
	issues, err := c.List()
	if err != nil {
		return e.Fail(err)
	}
	filter := splitIDs(ids)
	n := 0
	for _, d := range pm.CurrentDecisions(issues) {
		if (len(filter) > 0 && !d.Governs(filter)) || (*vetoes && !d.OpenVeto()) {
			continue
		}
		n++
		governs := "none"
		if len(d.IDs) > 0 {
			governs = strings.Join(d.IDs, " ")
		}
		by := d.Actor
		if d.Vetoable {
			by += ", vetoable"
		}
		for _, a := range d.Confirmed {
			by += ", confirmed by " + a
		}
		fmt.Fprintf(e.Stdout, "%s  %s\n  governs: %s\n  by: %s\n", d.ID, d.Title, governs, by)
	}
	if *vetoes {
		fmt.Fprintf(e.Stdout, "decisions: %d open to veto\n", n)
	} else {
		fmt.Fprintf(e.Stdout, "decisions: %d current\n", n)
	}
	return 0
}
