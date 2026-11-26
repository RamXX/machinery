package cli

import (
	"errors"

	"github.com/RamXX/machinery/internal/assess"
)

func init() {
	Register(&Command{Name: "assess", Summary: "Report, read-only, where an existing project stands against its own design.", Run: runAssess})
}

func runAssess(e *Env, args []string) int {
	fs := e.Flags("assess")
	config := fs.String("config", "", "the machinery.yaml to use; it may live outside the project")
	out := fs.String("out", "", "the directory for everything assess writes: outside the project, absent or empty")
	var since StringList
	fs.Var(&since, "since", "a commit the history section compares HEAD with (repeatable)")
	if !e.Parse(fs, args) {
		return 2
	}
	if fs.NArg() > 0 {
		return e.Usage("machinery assess takes no arguments")
	}
	err := assess.Run(assess.Options{ProjectFlag: e.ProjectFlag, Config: *config, Out: *out, Since: since,
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
