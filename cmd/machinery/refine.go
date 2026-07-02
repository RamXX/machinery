package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ramirosalas/machinery/internal/refine"
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
		refine.Run(args[0], args[1], outdir)
		return nil
	}
	_ = fmt.Sprintf
	return c
}
