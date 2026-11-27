package cli

import (
	"fmt"

	"github.com/RamXX/machinery/v2/internal/arch"
	"github.com/RamXX/machinery/v2/internal/stages"
)

func init() {
	Register(&Command{Name: "arch", Summary: "Check the layer and size constraints; --write-baseline records violations.", Run: runArch})
}

func runArch(e *Env, args []string) int {
	fs := e.Flags("arch")
	write := fs.Bool("write-baseline", false, "record the current violations in the baseline file")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery arch takes no arguments")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	if *write && arch.Applies(p) {
		vs, _, _, err := stages.ArchViolations(&stages.Context{P: p})
		if err == nil {
			err = arch.WriteBaseline(p, vs)
		}
		if err != nil {
			return e.Fail(err)
		}
		fmt.Fprintf(e.Stdout, "arch: wrote %d baseline entries to %s\n", len(vs), arch.BaselinePath(p))
	}
	return e.RunStages("arch", &stages.Context{P: p}, []string{"arch"})
}
