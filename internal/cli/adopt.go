package cli

import (
	"errors"
	"fmt"

	"github.com/RamXX/machinery/internal/adopt"
	"github.com/RamXX/machinery/internal/assess"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/rehearse"
)

func init() {
	Register(&Command{Name: "adopt", Summary: "Bring an existing project under machinery without touching its code.", Run: runAdopt})
}

func runAdopt(e *Env, args []string) int {
	fs := e.Flags("adopt")
	var paths, e2e StringList
	fs.Var(&paths, "paths", "code paths to cover (repeatable glob); default every path")
	fs.Var(&e2e, "e2e", "where end-to-end tests live (repeatable glob); default e2e/**")
	write := fs.Bool("write", false, "write the files; without it adopt only reports")
	rehearseIt := fs.Bool("rehearse", false, "adopt a snapshot outside the project and report what blocks")
	config := fs.String("config", "", "with --rehearse: the machinery.yaml to use; it may live outside the project")
	out := fs.String("out", "", "with --rehearse: the directory for everything it writes, outside the project, absent or empty")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery adopt takes no arguments")
	}
	switch {
	case *rehearseIt && *write:
		return e.Usage("--rehearse writes only in its out directory; it takes no --write")
	case !*rehearseIt && (*config != "" || *out != ""):
		return e.Usage("--config and --out go with --rehearse")
	case *rehearseIt:
		err := rehearse.Run(rehearse.Options{ProjectFlag: e.ProjectFlag, Config: *config, Out: *out, Paths: paths, E2E: e2e,
			Stdout: e.Stdout, Gen: Generate})
		var ue *assess.UsageError
		if errors.As(err, &ue) {
			return e.Usage("%s", ue.Msg)
		}
		if err != nil {
			return e.Fail(err)
		}
		return 0
	}
	root := e.ProjectFlag
	if root == "" {
		root = "."
	}
	res, err := adopt.Run(adopt.Options{Root: root, Paths: paths, E2E: e2e, Write: *write, Gen: Generate})
	if err != nil {
		return e.Fail(err)
	}
	res.Print(e.Stdout)
	// adopt never edits .gitignore; it says when the receipts would be committed
	if *write && gitx.IsRepo(root) && !gitx.Ignored(root, ".machinery/receipts/x.jsonl") {
		fmt.Fprintln(e.Stdout, "note: add .machinery/ to .gitignore; receipts are evidence of one run and are not committed")
	}
	if len(res.Blocked) > 0 {
		return 1
	}
	return 0
}
