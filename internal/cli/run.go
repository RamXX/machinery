package cli

import (
	"errors"
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
	adapter := fs.String("adapter", "", "write the receipts of an existing suite from its event stream: go-test-json")
	streams := fs.Bool("streams", false, "with --adapter: read the streams the command leaves in MACHINERY_GO_TEST_JSON")
	if !e.Parse(fs, args) {
		return 2
	}
	argv := fs.Args()
	if len(argv) == 0 {
		return e.Usage("machinery run [--stack <description>] -- <command> [args...]\n       machinery run --adapter go-test-json [--stack <description>] -- go test [flags] [packages]\n       machinery run --adapter go-test-json --streams [--stack <description>] -- <command> [args...]")
	}
	if *adapter != "" && *adapter != evidence.GoTestJSON {
		return e.Usage("--adapter: unknown adapter %q; the only adapter is %s", *adapter, evidence.GoTestJSON)
	}
	if *targeted != (*base != "") {
		return e.Usage("--affected and --base go together: machinery run --affected --base <ref> -- <command> [args...]")
	}
	if *streams && *adapter == "" {
		return e.Usage("--streams needs --adapter %s", evidence.GoTestJSON)
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
	var goTest *evidence.GoTest
	if *adapter != "" { // section 10.3
		var err error
		if goTest, err = evidence.NewGoTest(p, design.Load(p), argv, *streams); err != nil {
			var ue *evidence.UsageError
			if errors.As(err, &ue) {
				return e.Usage("%s", ue.Msg)
			}
			return e.Fail(err)
		}
		argv, opts.Adapter, opts.Streams = goTest.Argv, goTest, *streams
	}
	exit, file, err := evidence.Run(p, opts, argv)
	if err != nil {
		fmt.Fprintf(e.Stderr, "machinery: run: %v\n", err)
		if exit == 0 {
			exit = 1
		}
		return exit
	}
	if goTest != nil {
		for _, l := range goTest.Summary() {
			fmt.Fprintf(e.Stderr, "machinery run: %s\n", l)
		}
	}
	fmt.Fprintf(e.Stderr, "machinery run: %s (exit %d)\n", p.Rel(file), exit)
	return exit
}
