package cli

import (
	"fmt"

	"github.com/RamXX/machinery/v2/internal/affected"
	"github.com/RamXX/machinery/v2/internal/design"
	"github.com/RamXX/machinery/v2/internal/gitx"
	"github.com/RamXX/machinery/v2/internal/project"
)

func init() {
	Register(&Command{Name: "affected", Summary: "Print the obligation ids a change touches since a base ref.", Run: runAffected})
}

func runAffected(e *Env, args []string) int {
	fs := e.Flags("affected")
	base := fs.String("base", "", "git ref to compare the working tree with")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 || *base == "" {
		return e.Usage("machinery affected --base <ref>")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	if !e.CheckRef(p, *base) {
		return 2
	}
	ids, err := affected.Compute(p, design.Load(p), *base)
	if err != nil {
		return e.Fail(err)
	}
	for _, id := range ids {
		fmt.Fprintln(e.Stdout, id)
	}
	return 0
}

// CheckRef reports whether ref names a commit; otherwise it prints the usage
// error and the caller exits 2.
func (e *Env) CheckRef(p *project.Project, ref string) bool {
	if _, err := gitx.Resolve(p.Root, ref); err != nil {
		e.Usage("unknown git ref %s", ref)
		return false
	}
	return true
}
