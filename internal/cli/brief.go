package cli

import (
	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/pm"
)

func init() {
	Register(&Command{Name: "brief", Summary: "Print a brief for a backlog story by file pointer.", Run: runBrief})
}

func runBrief(e *Env, args []string) int {
	fs := e.Flags("brief")
	role := fs.String("role", pm.Builder, "builder, verifier or modeler")
	pos, ok := parseMixed(e, fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		return e.Usage("machinery brief takes one story id")
	}
	switch *role {
	case pm.Builder, pm.Verifier, pm.Modeler:
	default:
		return e.Usage("--role must be builder, verifier or modeler, got %q", *role)
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	c, err := backlog.New(p, "")
	if err != nil {
		return e.Fail(err)
	}
	is, code := showStory(e, c, pos[0])
	if is == nil {
		return code
	}
	if err := pm.Brief(e.Stdout, design.Load(p), is, *role); err != nil {
		return e.Fail(err)
	}
	return 0
}
