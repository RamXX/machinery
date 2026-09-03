package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/compose"
)

func newComposeCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "compose <composition.yaml> <coordinator.machine.json> [out-dir]",
		Short: "Generate the cross-aggregate composition spec validated against the coordinator",
		Args:  cobra.RangeArgs(2, 3),
	}
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		outdir := ""
		if len(args) > 2 {
			outdir = args[2]
		}
		if err := compose.RunTo(args[0], args[1], outdir, output.stdout); err != nil {
			fmt.Fprintln(output.stderr, err)
			return commandExitBecause(1, err)
		}
		return nil
	}
	return c
}
