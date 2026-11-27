package cli

import (
	"fmt"

	"github.com/RamXX/machinery/v2/internal/backlog"
	"github.com/RamXX/machinery/v2/internal/pm"
)

func init() {
	Register(&Command{Name: "observations", Summary: "List the observations of stories not yet filed; --file files them.", Run: runObservations})
}

func runObservations(e *Env, args []string) int {
	fs := e.Flags("observations")
	file := fs.Bool("file", false, "create one backlog item per observation and mark it filed")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery observations takes no arguments")
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
	obs := pm.Unfiled(issues)
	if !*file {
		for _, o := range obs {
			fmt.Fprintf(e.Stdout, "%s  %s: %s\n", o.Item.ID, o.Kind, o.Text)
		}
		fmt.Fprintf(e.Stdout, "observations: %d not filed\n", len(obs))
		return 0
	}
	for i := range obs {
		if err := pm.FileObservation(c, &obs[i]); err != nil {
			fmt.Fprintf(e.Stdout, "observations: %d filed\n", i)
			return e.Fail(err)
		}
		fmt.Fprintf(e.Stdout, "filed: %s  %s: %s (from %s)\n", obs[i].Filed, obs[i].Kind, obs[i].Text, obs[i].Item.ID)
	}
	fmt.Fprintf(e.Stdout, "observations: %d filed\n", len(obs))
	return 0
}
