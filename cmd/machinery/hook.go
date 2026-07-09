package main

import (
	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/hook"
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
	c.RunE = func(cmd *cobra.Command, args []string) error {
		return hook.Run(stdinR, stdoutW, root)
	}
	return c
}
