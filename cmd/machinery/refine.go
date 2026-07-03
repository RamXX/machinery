package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/refine"
)

func newRefineCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "refine <machine.json> <semantics.yaml> [out-dir]",
		Short: "Generate the data-refined model + refinement from a semantics annotation",
		Args:  cobra.RangeArgs(2, 3),
	}
	c.RunE = func(cmd *cobra.Command, args []string) error {
		outdir := ""
		if len(args) > 2 {
			outdir = args[2]
		}
		if err := refine.Run(args[0], args[1], outdir); err != nil {
			fmt.Fprintln(stderrW, err)
			exitFunc(1)
		}
		return nil
	}
	return c
}
