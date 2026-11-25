package cli

import "github.com/RamXX/machinery/internal/adopt"

func init() {
	Register(&Command{Name: "adopt", Summary: "Bring an existing project under machinery without touching its code.", Run: runAdopt})
}

func runAdopt(e *Env, args []string) int {
	fs := e.Flags("adopt")
	var paths, e2e StringList
	fs.Var(&paths, "paths", "code paths to cover (repeatable glob); default every path")
	fs.Var(&e2e, "e2e", "where end-to-end tests live (repeatable glob); default e2e/**")
	write := fs.Bool("write", false, "write the files; without it adopt only reports")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery adopt takes no arguments")
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
	if len(res.Blocked) > 0 {
		return 1
	}
	return 0
}
