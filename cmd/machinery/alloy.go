package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/alloy"
)

func newAlloyCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "alloy <design-dir> [out-dir]",
		Short: "Generate opted-in relational proofs and decision oracles",
		Args:  cobra.RangeArgs(1, 2),
	}
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		outdir := ""
		if len(args) > 1 {
			outdir = args[1]
		}
		if err := alloy.RunTo(args[0], outdir, output.stdout); err != nil {
			fmt.Fprintln(output.stderr, err)
			return commandExitBecause(1, err)
		}
		return nil
	}
	return c
}
