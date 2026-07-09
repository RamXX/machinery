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
	c.RunE = func(cmd *cobra.Command, args []string) error {
		outdir := ""
		if len(args) > 1 {
			outdir = args[1]
		}
		if err := alloy.Run(args[0], outdir); err != nil {
			fmt.Fprintln(stderrW, err)
			exitFunc(1)
		}
		return nil
	}
	return c
}
