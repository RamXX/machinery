package cli

import (
	"github.com/RamXX/machinery/v2/internal/project"
)

func init() {
	Register(&Command{Name: "config", Summary: "Print the effective configuration, defaults applied.", Run: func(e *Env, args []string) int {
		fs := e.Flags("config")
		if !e.Parse(fs, args) {
			return 2
		}
		p, ok := e.Project()
		if !ok {
			return 2
		}
		out, err := project.Marshal(&p.Cfg)
		if err != nil {
			return e.Fail(err)
		}
		e.Stdout.Write(out)
		return 0
	}})
}
