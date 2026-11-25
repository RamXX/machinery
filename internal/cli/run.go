package cli

import (
	"fmt"

	"github.com/RamXX/machinery/internal/affected"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/evidence"
)

func init() {
	Register(&Command{Name: "run", Summary: "Run a test command and record a receipts file.", Run: runRun})
}

func runRun(e *Env, args []string) int {
	fs := e.Flags("run")
	stack := fs.String("stack", "", "what the tests run against")
	targeted := fs.Bool("affected", false, "export the ids a change touches as MACHINERY_ONLY")
	base := fs.String("base", "", "git ref for --affected")
	if !e.Parse(fs, args) {
		return 2
	}
	argv := fs.Args()
	if len(argv) == 0 {
		return e.Usage("machinery run [--stack <description>] [--affected --base <ref>] -- <command> [args...]")
	}
	if *targeted != (*base != "") {
		return e.Usage("--affected and --base go together: machinery run --affected --base <ref> -- <command> [args...]")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	opts := evidence.RunOptions{Stack: *stack, Stdout: e.Stdout, Stderr: e.Stderr}
	if *targeted {
		if !e.CheckRef(p, *base) {
			return 2
		}
		ids, err := affected.Compute(p, design.Load(p), *base)
		if err != nil {
			return e.Fail(err)
		}
		if len(ids) == 0 {
			fmt.Fprintf(e.Stderr, "machinery run: nothing affected since %s; no command run\n", *base)
			return 0
		}
		opts.Only = ids
	}
	exit, file, err := evidence.Run(p, opts, argv)
	if err != nil {
		fmt.Fprintf(e.Stderr, "machinery: run: %v\n", err)
		if exit == 0 {
			exit = 1
		}
		return exit
	}
	fmt.Fprintf(e.Stderr, "machinery run: %s (exit %d)\n", p.Rel(file), exit)
	return exit
}
