package cli

import "github.com/RamXX/machinery/internal/stages"

func init() {
	Register(&Command{Name: "trace", Summary: "Check that code serves actions and tests claim obligations.", Run: func(e *Env, args []string) int {
		fs := e.Flags("trace")
		if !e.Parse(fs, args) {
			return 2
		}
		if fs.NArg() > 0 {
			return e.Usage("machinery trace takes no arguments")
		}
		p, ok := e.Project()
		if !ok {
			return 2
		}
		return e.RunStages("trace", &stages.Context{P: p}, []string{"trace"})
	}})
}
