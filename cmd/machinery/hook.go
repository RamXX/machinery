package main

import (
	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/hook"
	"github.com/RamXX/machinery/internal/install"
)

// newHookCmd is the shared host-adapter plumbing: a hook shim or plugin pipes
// each normalized event (JSON on stdin) through `machinery hook`, and the answer
// (deny/block/context JSON, or nothing) goes to stdout. Hidden because it is
// machine-to-machine, not a user command; humans run `machinery check`.
func newHookCmd() *cobra.Command {
	c := &cobra.Command{
		Use:    "hook",
		Short:  "Handle one normalized agent-host hook event (JSON on stdin; adapter plumbing)",
		Hidden: true,
		Args:   cobra.NoArgs,
	}
	var root string
	c.Flags().StringVar(&root, "root", "", "project root (default: $CLAUDE_PROJECT_DIR, then the event's cwd)")
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		// Host governance must never inspect a half-installed adapter or design
		// substrate. The same lock used by install/update first recovers a
		// PREPARED transaction and remains held until the decision is emitted.
		return install.WithInstallInspectionLock(func() error {
			return hook.Run(stdinR, output.stdout, root)
		})
	}
	return c
}
