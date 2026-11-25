package cli

import (
	"fmt"

	"github.com/RamXX/machinery/internal/evidence"
)

func init() {
	Register(&Command{Name: "run", Summary: "Run a test command and record a receipts file.", Run: runRun})
}

func runRun(e *Env, args []string) int {
	fs := e.Flags("run")
	stack := fs.String("stack", "", "what the tests run against")
	if !e.Parse(fs, args) {
		return 2
	}
	argv := fs.Args()
	if len(argv) == 0 {
		return e.Usage("machinery run [--stack <description>] -- <command> [args...]")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	exit, file, err := evidence.Run(p, evidence.RunOptions{Stack: *stack, Stdout: e.Stdout, Stderr: e.Stderr}, argv)
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
