package main

import (
	"github.com/RamXX/machinery/internal/hook"
	"github.com/spf13/cobra"
)

func newHookStateCmd() *cobra.Command {
	command := &cobra.Command{Use: "hook-state", Short: "Inspect and transition the durable governance hook store", Args: cobra.NoArgs}
	var root, from string
	adopt := &cobra.Command{Use: "adopt", Short: "Reaffirm the recorded store binding while preserving obligations", Args: cobra.NoArgs}
	adopt.Flags().StringVar(&root, "root", "", "project root whose retained obligations are reported")
	adopt.Flags().StringVar(&from, "from", "", "restore the recorded store from a quarantine directory")
	_ = adopt.MarkFlagRequired("root")
	adopt.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		return hook.AdoptState(output.stdout, root, from)
	}
	command.AddCommand(adopt)
	return command
}
