package cli

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

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
	allowNone := fs.Bool("allow-no-streams", false, "with --streams: a command that leaves no stream is no failure")
	keep := fs.String("keep-streams", "", "with --streams: copy every stream read into <dir>/<run id>/")
	if !e.Parse(fs, args) {
		return 2
	}
	argv := fs.Args()
	if len(argv) == 0 {
		return e.Usage("machinery run [--stack <description>] -- <command> [args...]\n       machinery run --adapter go-test-json [--stack <description>] -- go test [flags] [packages]\n       machinery run --adapter go-test-json --streams [--stack <description>] -- <command> [args...]")
	}
	if *adapter != "" && *adapter != evidence.GoTestJSON {
		return e.Usage("--adapter: unknown adapter %q: write --adapter %s, the only adapter", *adapter, evidence.GoTestJSON)
	}
	if *targeted != (*base != "") {
		return e.Usage("--affected and --base go together: machinery run --affected --base <ref> -- <command> [args...]")
	}
	if *streams && *adapter == "" {
		return e.Usage("--streams needs --adapter %s: add it before --streams", evidence.GoTestJSON)
	}
	if *allowNone && !*streams {
		return e.Usage("--allow-no-streams needs --streams: it is about the streams a wrapper leaves; add --streams or leave it out")
	}
	keepSet := false
	fs.Visit(func(f *flag.Flag) { keepSet = keepSet || f.Name == "keep-streams" })
	if keepSet && !*streams {
		return e.Usage("--keep-streams needs --streams: it keeps the streams a wrapper leaves; add --streams or leave it out")
	}
	if keepSet && strings.TrimSpace(*keep) == "" {
		return e.Usage("--keep-streams needs a directory: --keep-streams <dir>")
	}
	p, ok := e.Project()
	if !ok {
		return 2
	}
	opts := evidence.RunOptions{Stack: *stack, Stdout: e.Stdout, Stderr: e.Stderr}
	if keepSet {
		abs, err := filepath.Abs(*keep)
		if err != nil {
			return e.Fail(err)
		}
		opts.KeepStreams = abs
	}
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
		goTest.AllowNoStreams = *allowNone
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
		if goTest.Wrote() == 0 { // a run that produced no evidence (section 10.3)
			if n, err := evidence.ReceiptCount(file); err == nil && n == 0 {
				if ev, err := evidence.Evaluate(p, design.Load(p), true); err == nil {
					goTest.Rescope(ev.HasReceipt)
				}
			}
		}
		for _, l := range goTest.Summary() {
			fmt.Fprintf(e.Stderr, "machinery run: %s\n", l)
		}
		if goTest.NoStream() && !*allowNone && exit == 0 {
			exit = 1 // a run that reads no stream is not a success
		}
	}
	fmt.Fprintf(e.Stderr, "machinery run: %s (exit %d)\n", p.Rel(file), exit)
	return exit
}
